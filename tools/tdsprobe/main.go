/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

// This probe is a client integration test, not a server or SQL implementation.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

type config struct {
	Host             string      `json:"host"`
	Port             int         `json:"port"`
	Database         string      `json:"database"`
	Username         string      `json:"username"`
	Password         string      `json:"password"`
	ExpectLoginError bool        `json:"expect_login_error"`
	Statements       []statement `json:"statements"`
}

type statement struct {
	SQL        string      `json:"sql"`
	Parameters []parameter `json:"parameters"`
	Prepare    bool        `json:"prepare"`
	Expect     expectation `json:"expect"`
}

type parameter struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type expectation struct {
	Error    bool             `json:"error"`
	Contains []string         `json:"contains"`
	Rows     *int             `json:"rows"`
	Data     []map[string]any `json:"data"`
	Columns  []string         `json:"columns"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("TDS client probe passed")
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: tdsprobe <json-config>")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	var cfg config
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}
	u := &url.URL{Scheme: "sqlserver", Host: net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port)), User: url.UserPassword(cfg.Username, cfg.Password)}
	q := url.Values{"database": {cfg.Database}, "encrypt": {"disable"}, "connection timeout": {"10"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if cfg.ExpectLoginError {
		if err == nil {
			return fmt.Errorf("login unexpectedly succeeded")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "login") {
			return fmt.Errorf("expected login rejection, got: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	for i, stmt := range cfg.Statements {
		if err := execute(ctx, conn, stmt); err != nil {
			return fmt.Errorf("statement %d: %w\nSQL: %s", i+1, err, stmt.SQL)
		}
	}
	return nil
}

func execute(ctx context.Context, conn *sql.Conn, stmt statement) error {
	args := make([]any, len(stmt.Parameters))
	for i, p := range stmt.Parameters {
		v := p.Value
		if n, ok := v.(json.Number); ok {
			if strings.ContainsAny(string(n), ".eE") {
				f, err := n.Float64()
				if err != nil {
					return err
				}
				v = f
			} else {
				i, err := n.Int64()
				if err != nil {
					return err
				}
				v = i
			}
		}
		args[i] = sql.Named(p.Name, v)
	}
	query := conn.QueryContext
	if stmt.Prepare {
		prepared, err := conn.PrepareContext(ctx, stmt.SQL)
		if err != nil {
			return checkError(stmt.Expect, err)
		}
		defer prepared.Close()
		query = func(ctx context.Context, _ string, args ...any) (*sql.Rows, error) {
			return prepared.QueryContext(ctx, args...)
		}
	}
	rows, err := query(ctx, stmt.SQL, args...)
	if err != nil {
		return checkError(stmt.Expect, err)
	}
	defer rows.Close()
	var data []map[string]any
	for {
		columns, err := rows.Columns()
		if err != nil {
			return checkError(stmt.Expect, err)
		}
		if stmt.Expect.Columns != nil && !reflect.DeepEqual(columns, stmt.Expect.Columns) {
			return fmt.Errorf("expected columns %v, got %v", stmt.Expect.Columns, columns)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			row := make(map[string]any, len(columns))
			for i, name := range columns {
				row[name] = values[i]
			}
			data = append(data, row)
		}
		if err := rows.Err(); err != nil {
			return checkError(stmt.Expect, err)
		}
		if !rows.NextResultSet() {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return checkError(stmt.Expect, err)
	}
	if stmt.Expect.Error {
		return fmt.Errorf("expected an error, query succeeded")
	}
	if stmt.Expect.Rows != nil && len(data) != *stmt.Expect.Rows {
		return fmt.Errorf("expected %d rows, got %d: %v", *stmt.Expect.Rows, len(data), data)
	}
	if stmt.Expect.Data != nil {
		if len(data) != len(stmt.Expect.Data) {
			return fmt.Errorf("expected %d data rows, got %d", len(stmt.Expect.Data), len(data))
		}
		for i, want := range stmt.Expect.Data {
			for name, expected := range want {
				actual, found := data[i][name]
				if !found || !equalValue(actual, expected) {
					return fmt.Errorf("row %d column %s: got %v (%T), expected %v (%T)", i, name, actual, actual, expected, expected)
				}
			}
		}
	}
	return nil
}

func checkError(want expectation, err error) error {
	if !want.Error {
		return err
	}
	for _, text := range want.Contains {
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(text)) {
			return fmt.Errorf("error %q does not contain %q", err, text)
		}
	}
	return nil
}

func equalValue(actual, expected any) bool {
	if n, ok := expected.(json.Number); ok {
		switch v := actual.(type) {
		case int64:
			i, err := n.Int64()
			return err == nil && v == i
		case float64:
			want, validWant := new(big.Rat).SetString(string(n))
			got, validGot := new(big.Rat).SetString(fmt.Sprint(v))
			return validWant && validGot && want.Cmp(got) == 0
		default:
			return false
		}
	}
	if b, ok := actual.([]byte); ok {
		actual = string(b)
	}
	return reflect.DeepEqual(actual, expected)
}
