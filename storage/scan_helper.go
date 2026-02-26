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
	"sync/atomic"
	"time"

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
		if s.IsSymbol() {
			symIndex[strings.ToLower(s.String())] = i
			continue
		}
		if sym, ok := s.Any().(scm.Symbol); ok {
			symIndex[strings.ToLower(string(sym))] = i
		}
	}

	var enc func(scm.Scmer)
	writeSymbolOrColumn := func(s string) {
		sLower := strings.ToLower(s)
		// Prefer mapping lambda param -> column name
		if idx, ok := symIndex[sLower]; ok {
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
		switch {
		case node.IsNil():
			io.WriteString(w, "nil")
		case node.IsBool():
			if node.Bool() {
				io.WriteString(w, "true")
			} else {
				io.WriteString(w, "false")
			}
		case node.IsInt():
			io.WriteString(w, fmt.Sprint(node.Int()))
		case node.IsFloat():
			io.WriteString(w, fmt.Sprint(node.Float()))
		case node.IsString():
			io.WriteString(w, "\"")
			io.WriteString(w, node.String())
			io.WriteString(w, "\"")
		case node.IsSymbol():
			writeSymbolOrColumn(node.String())
		case node.IsSlice():
			slice := node.Slice()
			if len(slice) > 0 {
				if slice[0].IsSymbol() && slice[0].String() == "outer" {
					io.WriteString(w, "?")
					return
				}
			}
			io.WriteString(w, "(")
			for i, item := range slice {
				if i > 0 {
					io.WriteString(w, " ")
				}
				enc(item)
			}
			io.WriteString(w, ")")
		default:
			// Prefer tag-based decoding for special cases.
			if node.IsProc() {
				io.WriteString(w, "?")
				return
			}
			if node.IsNthLocalVar() {
				i := int(node.NthLocalVar())
				if i >= 0 && i < len(columns) {
					io.WriteString(w, columns[i])
				} else {
					io.WriteString(w, "?")
				}
				return
			}
			// Native function: try to resolve declaration if present.
			if def := scm.DeclarationForValue(node); def != nil {
				io.WriteString(w, def.Name)
				return
			}
			// Fallback unknown
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
	row := []scm.Scmer{
		scm.NewString(schema),
		scm.NewString(table),
		scm.NewBool(ordered),
		scm.NewString(filter),
		scm.NewString(order),
		scm.NewInt(inputCount),
		scm.NewInt(outputCount),
		scm.NewInt(analyzeNs),
		scm.NewInt(execNs),
	}
	t.Insert(cols, [][]scm.Scmer{row}, nil, scm.NewNil(), false, nil)
}

// touchTempColumns updates lastAccessed on temp columns used by this scan (lock-free).
func touchTempColumns(t *table, colSets ...[]string) {
	now := time.Now().UnixNano()
	for _, c := range t.Columns {
		if !c.IsTemp {
			continue
		}
		for _, cols := range colSets {
			found := false
			for _, col := range cols {
				if col == c.Name {
					atomic.StoreInt64(&c.lastAccessed, now)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
}
