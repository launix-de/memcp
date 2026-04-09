/*
Copyright (C) 2025-2026  MemCP Contributors

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
	"strconv"
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

	var numBuf [64]byte // stack-allocated buffer for number formatting
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
			b := strconv.AppendInt(numBuf[:0], node.Int(), 10)
			w.Write(b)
		case node.IsFloat():
			b := strconv.AppendFloat(numBuf[:0], node.Float(), 'g', -1, 64)
			w.Write(b)
		case node.IsString():
			s, _ := node.AppendString(nil) // zero-alloc for tagString
			io.WriteString(w, "\"")
			io.WriteString(w, s)
			io.WriteString(w, "\"")
		case node.IsSymbol():
			s, _ := node.AppendString(nil) // zero-alloc for tagSymbol
			writeSymbolOrColumn(s)
		case node.IsSlice():
			slice := node.Slice()
			if len(slice) > 0 && slice[0].IsSymbol() {
				head, _ := slice[0].AppendString(nil) // zero-alloc
				if head == "outer" {
					io.WriteString(w, "?")
					return
				}
				// Normalize !list optimizer form back to (list ...) for stable canonical names.
				// (!list NthLocalVar(start) count expr...) encodes (list expr...) but the
				// storage slot (items[1]) varies per call site, so two identical lists would
				// get different canonical names. Strip items[1] and items[2] and use "list".
				if head == "!list" && len(slice) >= 3 {
					count := int(scm.ToInt(slice[2]))
					if count == len(slice)-3 {
						io.WriteString(w, "(list")
						for _, item := range slice[3:] {
							io.WriteString(w, " ")
							enc(item)
						}
						io.WriteString(w, ")")
						return
					}
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
				// Use pointer address to produce a unique stable name per lambda within
				// the session. Prevents YEAR, MONTH, DAY, etc. from colliding on the
				// same canonical index name.
				io.WriteString(w, fmt.Sprintf("%p", node.Proc()))
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
		{"index_cols", "TEXT"},
		{"inputCount", "INT"},
		{"outputCount", "INT"},
		// TODO: measurements are temporary; remove later (store in nanoseconds)
		{"analyze_ns", "INT"},
		{"exec_ns", "INT"},
		{"timestamp", "INT"},
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
func safeLogScan(schema, table string, ordered bool, filter, order, indexCols string, inputCount, outputCount, analyzeNs, execNs int64) {
	defer func() { _ = recover() }()
	db := GetDatabase("system_statistic")
	if db == nil {
		return
	}
	t := db.GetTable("scans")
	if t == nil {
		return
	}

	cols := []string{"schema", "table", "ordered", "filter", "order", "index_cols", "inputCount", "outputCount", "analyze_ns", "exec_ns", "timestamp"}
	row := []scm.Scmer{
		scm.NewString(schema),
		scm.NewString(table),
		scm.NewBool(ordered),
		scm.NewString(filter),
		scm.NewString(order),
		scm.NewString(indexCols),
		scm.NewInt(inputCount),
		scm.NewInt(outputCount),
		scm.NewInt(analyzeNs),
		scm.NewInt(execNs),
		scm.NewInt(time.Now().UnixNano()),
	}
	t.Insert(cols, [][]scm.Scmer{row}, nil, scm.NewNil(), false, nil)
}

// boundaryIndexCols returns a comma-separated list of column names from boundaries,
// representing the columns used for index lookup in this scan.
func boundaryIndexCols(b boundaries) string {
	if len(b) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, bc := range b {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(bc.col)
	}
	return sb.String()
}

// touchTempColumns updates lastAccessed on all temp columns of the table (lock-free).
// Touching every temp column (not just the ones in the scan's column lists) prevents
// a concurrent eviction from modifying t.Columns while the scan is in progress.
func touchTempColumns(t *table, colSets ...[]string) {
	now := time.Now().UnixNano()
	for _, c := range t.Columns {
		if c.IsTemp {
			atomic.StoreInt64(&c.lastAccessed, now)
		}
	}
}

// dropUnsafeComputeBoundaries removes index/range pushdown on StorageComputeProxy
// columns.
//
// These columns are lazy caches, not stable physical index keys:
//   - session-sensitive proxies expose per-session variants while the shard index
//     domain is global
//   - ordered temp columns (ORC/window caches) may be only partially materialized
//     between queries and become incrementally valid via validMask
//
// Using them for iterateIndex can therefore prune rows that would match once the
// proxy is read/recomputed during the scan itself. Keep such predicates as
// post-filter conditions instead of index boundaries.
func dropUnsafeComputeBoundaries(t *table, in boundaries) boundaries {
	if len(in) == 0 {
		return in
	}
	t.shardModeMu.RLock()
	shards := t.Shards
	if len(shards) == 0 {
		shards = t.PShards
	}
	t.shardModeMu.RUnlock()
	if len(shards) == 0 {
		return in
	}

	out := make(boundaries, 0, len(in))
	for _, b := range in {
		drop := false
		for _, s := range shards {
			if s == nil {
				continue
			}
			if s.schemaColumn(b.col) == nil {
				continue
			}
			func() {
				defer func() {
					if recover() != nil {
						// Some computed/prejoin boundary columns exist only on specific
						// runtime paths. Missing local storage means we cannot prove
						// session-sensitive index unsafety here, so keep the boundary.
					}
				}()
				cs := s.getColumnStorageOrPanicEx(b.col, false)
				if _, ok := cs.(*StorageComputeProxy); ok {
					drop = true
				}
			}()
			if drop {
				break
			}
		}
		if !drop {
			out = append(out, b)
		}
	}
	return out
}
