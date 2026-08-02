/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type probe struct {
	Host       string      `json:"host"`
	Port       int         `json:"port"`
	Database   string      `json:"database"`
	Username   string      `json:"username"`
	Password   string      `json:"password"`
	Statements []statement `json:"statements"`
}

type statement struct {
	SQL    string      `json:"sql"`
	Expect expectation `json:"expect"`
}

type expectation struct {
	Error       bool             `json:"error"`
	Contains    []string         `json:"contains"`
	NotContains []string         `json:"not_contains"`
	Rows        *int             `json:"rows"`
	Data        []map[string]any `json:"data"`
}

func main() {
	if len(os.Args) != 2 {
		fail("usage: mysql_probe <json-config>")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail("read config: %v", err)
	}
	var cfg probe
	if err := json.Unmarshal(raw, &cfg); err != nil {
		fail("parse config: %v", err)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&multiStatements=false",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fail("open mysql connection: %v", err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		fail("pin mysql connection: %v", err)
	}
	defer conn.Close()
	for i, stmt := range cfg.Statements {
		if err := runStatement(conn, stmt); err != nil {
			fail("statement %d failed: %v\nsql: %s", i+1, err, stmt.SQL)
		}
	}
	fmt.Println("mysql probe passed")
}

func runStatement(conn *sql.Conn, stmt statement) error {
	rows, err := conn.QueryContext(context.Background(), stmt.SQL)
	if stmt.Expect.Error {
		if err == nil {
			rows.Close()
			return fmt.Errorf("expected error, got success")
		}
		msg := err.Error()
		for _, needle := range stmt.Expect.Contains {
			if !strings.Contains(msg, needle) {
				return fmt.Errorf("error missing %q: %s", needle, msg)
			}
		}
		for _, needle := range stmt.Expect.NotContains {
			if strings.Contains(msg, needle) {
				return fmt.Errorf("error unexpectedly contained %q: %s", needle, msg)
			}
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	got, err := collectRows(rows)
	if err != nil {
		return err
	}
	if stmt.Expect.Rows != nil && len(got) != *stmt.Expect.Rows {
		return fmt.Errorf("expected %d rows, got %d", *stmt.Expect.Rows, len(got))
	}
	for rowIdx, expected := range stmt.Expect.Data {
		if rowIdx >= len(got) {
			return fmt.Errorf("missing expected row %d", rowIdx)
		}
		for key, want := range expected {
			if fmt.Sprint(got[rowIdx][key]) != fmt.Sprint(want) {
				return fmt.Errorf("row %d field %s = %v, want %v", rowIdx, key, got[rowIdx][key], want)
			}
		}
	}
	return nil
}

func collectRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				row[col] = string(v)
			default:
				row[col] = v
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
