/*
Copyright (C) 2025  MemCP Contributors

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
package storage

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/launix-de/memcp/scm"
)

// encodeScmer prints a compact textual encoding of a Scheme AST to w.
// Unknowns print as "?".
// - Unknown symbols (not a global function and not one of the provided column names) => "?".
// - Lambdas (scm.Proc) => "?".
// - Go builtins (func(...scm.Scmer) scm.Scmer) => function name if found in Globalenv, else "?".
// For filters, pass the condition Proc.Body as v and the filter columns as context.
// For sort expressions, pass the string column name or the Proc.Body with its params as context.
// columnSymbols must be the Proc.Params list when encoding a lambda body. If present:
// - Any symbol equal to a param prints as the corresponding column name by index.
// - Any NthLocalVar(i) prints as columns[i] (when i < len(columns)); otherwise "?".
func encodeScmer(v scm.Scmer, w io.Writer, columns []string, columnSymbols []scm.Scmer) {
	cols := make(map[string]bool, len(columns))
	for _, c := range columns {
		cols[strings.ToLower(c)] = true
	}
	// Build symbol->index from Proc.Params to map lambda params to actual columns
	symIndex := make(map[string]int, len(columnSymbols))
	for i, s := range columnSymbols {
		if sym, ok := s.(scm.Symbol); ok {
			symIndex[string(sym)] = i
		}
	}

	var enc func(scm.Scmer)
	writeSymbolOrColumn := func(s string) {
		// Prefer mapping lambda param -> column name
		if idx, ok := symIndex[s]; ok {
			if idx >= 0 && idx < len(columns) {
				io.WriteString(w, columns[idx])
				return
			}
			io.WriteString(w, "?")
			return
		}
		// Otherwise, if it looks like a global function/operator, print symbol
		if scm.Globalenv.FindRead(scm.Symbol(s)) != nil {
			io.WriteString(w, s)
			return
		}
		// Unknown
		io.WriteString(w, "?")
	}

	enc = func(node scm.Scmer) {
		switch x := node.(type) {
		case nil:
			io.WriteString(w, "nil")
		case bool:
			if x {
				io.WriteString(w, "true")
			} else {
				io.WriteString(w, "false")
			}
		case int:
			io.WriteString(w, fmt.Sprint(x))
		case int64:
			io.WriteString(w, fmt.Sprint(x))
		case uint64:
			io.WriteString(w, fmt.Sprint(x))
		case float64:
			io.WriteString(w, fmt.Sprint(x))
		case string:
			// quote strings
			io.WriteString(w, "\"")
			io.WriteString(w, x)
			io.WriteString(w, "\"")
		case scm.LazyString:
			io.WriteString(w, "\"")
			io.WriteString(w, x.GetValue())
			io.WriteString(w, "\"")
		case scm.Symbol:
			writeSymbolOrColumn(string(x))
		case scm.NthLocalVar:
			i := int(x)
			if i >= 0 && i < len(columns) {
				io.WriteString(w, columns[i])
			} else {
				io.WriteString(w, "?")
			}
		case scm.Proc:
			// Lambdas are encoded as unknown per requirements
			io.WriteString(w, "?")
		case []scm.Scmer:
			// If this is an (outer ...) form, replace whole call with "?"
			if len(x) > 0 {
				if head, ok := x[0].(scm.Symbol); ok && string(head) == "outer" {
					io.WriteString(w, "?")
					return
				}
			}
			io.WriteString(w, "(")
			for i := 0; i < len(x); i++ {
				if i > 0 {
					io.WriteString(w, " ")
				}
				enc(x[i])
			}
			io.WriteString(w, ")")
		case func(...scm.Scmer) scm.Scmer:
			// Resolve via scm.DeclarationForValue helper
			if def := scm.DeclarationForValue(x); def != nil {
				io.WriteString(w, def.Name)
			} else {
				io.WriteString(w, "?")
			}
		default:
			// Unknown constructs are intentionally hidden
			io.WriteString(w, "?")
		}
	}

	enc(v)
}

// helper that returns encoded string
func encodeScmerToString(v scm.Scmer, columns []string, columnSymbols []scm.Scmer) string {
	var b bytes.Buffer
	encodeScmer(v, &b, columns, columnSymbols)
	return b.String()
}

// Minimum table size required to collect scan statistics.
// Deprecated: use Settings.AnalyzeMinItems instead
const scanStatsMinInput int64 = 1000

// ensureSystemStatistic ensures the `system_statistic.scans` table exists with expected columns.
func ensureSystemStatistic() {
	const dbName = "system_statistic"
	const tblName = "scans"

	// create database if missing
	if GetDatabase(dbName) == nil {
		CreateDatabase(dbName, true)
	}
	db := GetDatabase(dbName)
	if db == nil {
		return // should not happen; avoid panicking during init
	}

	// create table if missing (use Sloppy persistency to avoid fsync costs)
	t, _ := CreateTable(dbName, tblName, Sloppy, true)
	if t == nil {
		t = db.GetTable(tblName)
		if t == nil {
			return
		}
	}
	// ensure persistency mode is Sloppy even if table pre-existed
	if t.PersistencyMode != Sloppy {
		t.PersistencyMode = Sloppy
		t.schema.save()
	}
	// ensure columns exist
	need := []struct {
		name string
		typ  string
	}{
		{"schema", "TEXT"},
		{"table", "TEXT"},
		{"ordered", "BOOL"},
		{"filter", "TEXT"},
		{"order", "TEXT"},
		{"inputCount", "INT"},
		{"outputCount", "INT"},
		// TODO: measurements are temporary; remove later (store in nanoseconds)
		{"analyze_ns", "INT"},
		{"exec_ns", "INT"},
	}
	have := make(map[string]bool)
	if t != nil {
		for _, c := range t.Columns {
			have[strings.ToLower(c.Name)] = true
		}
		for _, c := range need {
			if !have[strings.ToLower(c.name)] {
				t.CreateColumn(c.name, c.typ, nil, nil)
			}
		}
	}

	// --- New: ensure system_statistic.table_histogram exists ---
	// Schema: table_histogram(schema TEXT, table TEXT, model BLOB, UNIQUE(schema, table))
	ensureTable := func(db *database, name string, pm PersistencyMode) *table {
		tt, _ := CreateTable(dbName, name, pm, true)
		if tt == nil {
			tt = db.GetTable(name)
		}
		if tt == nil {
			return nil
		}
		if tt.PersistencyMode != pm {
			tt.PersistencyMode = pm
			tt.schema.save()
		}
		return tt
	}

	// helper: ensure columns exist by name/type
	ensureCols := func(tt *table, cols []struct{ name, typ string }) {
		if tt == nil {
			return
		}
		have := make(map[string]bool)
		for _, c := range tt.Columns {
			have[strings.ToLower(c.Name)] = true
		}
		for _, c := range cols {
			if !have[strings.ToLower(c.name)] {
				tt.CreateColumn(c.name, c.typ, nil, nil)
			}
		}
	}

	// helper: ensure a unique key with the exact set of columns exists
	ensureUnique := func(tt *table, id string, cols []string) {
		if tt == nil {
			return
		}
		// check if unique with same columns already exists (order-insensitive)
		has := false
		want := make(map[string]bool, len(cols))
		for _, c := range cols {
			want[strings.ToLower(c)] = true
		}
		for _, u := range tt.Unique {
			if len(u.Cols) != len(cols) {
				continue
			}
			ok := true
			for _, c := range u.Cols {
				if !want[strings.ToLower(c)] {
					ok = false
					break
				}
			}
			if ok {
				has = true
				break
			}
		}
		if !has {
			// append unique and persist
			tt.schema.schemalock.Lock()
			tt.Unique = append(tt.Unique, uniqueKey{Id: id, Cols: cols})
			tt.schema.save()
			tt.schema.schemalock.Unlock()
		}
	}

	// table_histogram
	th := ensureTable(db, "table_histogram", Sloppy)
	ensureCols(th, []struct{ name, typ string }{
		{"schema", "TEXT"},
		{"table", "TEXT"},
		{"model", "BLOB"}, // stored as string/blob; overlay handles long data
	})
	ensureUnique(th, "uniq_table_histogram_schema_table", []string{"schema", "table"})

	// base_models: base_models(id PRIMARY KEY, model)
	bm := ensureTable(db, "base_models", Sloppy)
	ensureCols(bm, []struct{ name, typ string }{
		{"id", "TEXT"},
		{"model", "BLOB"},
	})
	// ensure PRIMARY KEY(id)
	// treat any unique on [id] as sufficient; otherwise add PRIMARY
	ensureUnique(bm, "PRIMARY", []string{"id"})
}

// safeLogScan writes a single row into system_statistic.scans. Failures are ignored.
// TODO: measurements are temporary; remove later (nanoseconds)
func safeLogScan(schema, table string, ordered bool, filter, order string, inputCount, outputCount, analyzeNs, execNs int64) {
	defer func() { _ = recover() }()
	db := GetDatabase("system_statistic")
	if db == nil {
		return
	}
	t := db.GetTable("scans")
	if t == nil {
		return
	}

	cols := []string{"schema", "table", "ordered", "filter", "order", "inputCount", "outputCount", "analyze_ns", "exec_ns"}
	row := []scm.Scmer{schema, table, ordered, filter, order, inputCount, outputCount, analyzeNs, execNs}
	t.Insert(cols, [][]scm.Scmer{row}, nil, nil, false, nil)
}
