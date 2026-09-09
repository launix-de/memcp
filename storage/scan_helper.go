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
	"strings"
	"sync/atomic"
	"time"

	"github.com/launix-de/memcp/scm"
)

type scanLogEvent struct {
	schema, table, filter, order, indexCols                    string
	ordered                                                    bool
	inputCount, candidateCount, outputCount, analyzeNs, execNs int64
	condition                                                  scm.Scmer
	conditionCols                                              []string
	encodeCondition                                            bool
}

var scanLogQueue = make(chan scanLogEvent, 1024)

func init() {
	go func() {
		for event := range scanLogQueue {
			if event.encodeCondition {
				if proc, ok := event.condition.Any().(scm.Proc); ok {
					var params []scm.Scmer
					if proc.Params.IsSlice() {
						params = proc.Params.Slice()
					} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
						params = arr
					}
					event.filter = scm.ExpressionName(proc.Body, event.conditionCols, params)
				}
			}
			safeLogScan(event.schema, event.table, event.ordered, event.filter, event.order, event.indexCols,
				event.inputCount, event.candidateCount, event.outputCount, event.analyzeNs, event.execNs)
		}
	}()
}

// enqueueScanLog keeps statistics off the execution path without creating one
// goroutine and callback frame per scan. Telemetry is best-effort; a saturated
// sink must never apply backpressure to query execution.
func enqueueScanLog(event scanLogEvent) {
	select {
	case scanLogQueue <- event:
	default:
	}
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
		{"candidateCount", "INT"},
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
func safeLogScan(schema, table string, ordered bool, filter, order, indexCols string, inputCount, candidateCount, outputCount, analyzeNs, execNs int64) {
	defer func() { _ = recover() }()
	// The telemetry sink and its physical cache relations must not observe
	// themselves. Group-cache reads are still reads of the scan sink; logging
	// them would recursively schedule incremental writes while the cache shard is
	// held for reading and would leave telemetry work in flight at shutdown.
	if schema == "system_statistic" &&
		(table == "scans" || strings.HasPrefix(table, ".")) {
		return
	}
	db := GetDatabase("system_statistic")
	if db == nil {
		return
	}
	t := db.GetTable("scans")
	if t == nil {
		return
	}

	cols := []string{"schema", "table", "ordered", "filter", "order", "index_cols", "inputCount", "candidateCount", "outputCount", "analyze_ns", "exec_ns", "timestamp"}
	row := []scm.Scmer{
		scm.NewString(schema),
		scm.NewString(table),
		scm.NewBool(ordered),
		scm.NewString(filter),
		scm.NewString(order),
		scm.NewString(indexCols),
		scm.NewInt(inputCount),
		scm.NewInt(candidateCount),
		scm.NewInt(outputCount),
		scm.NewInt(analyzeNs),
		scm.NewInt(execNs),
		scm.NewInt(time.Now().UnixNano()),
	}
	t.Insert(cols, [][]scm.Scmer{row}, nil, scm.NewNil(), false, nil)
}

// boundaryIndexCols returns a comma-separated list of column names from analyzed boundaries,
// representing the columns used for index lookup in this scan.
func boundaryIndexCols(b analyzedBoundaries) string {
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

func scanAccessIndexCols(access scanAccess) string {
	if access.len() == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < access.len(); i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(access.boundaryColumn(i))
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

// ensureScanAccessColumns loads every physical source column referenced by an
// access plan before the scan takes the shard read lock. Exact predicates may
// be absent from the residual filter, so its column preflight is intentionally
// not relied upon here.
func (t *storageShard) ensureScanAccessColumns(access scanAccess, alreadyLocked bool, currentTx *TxContext) {
	for i := 0; i < access.len(); i++ {
		mapCols, _ := access.boundaryMap(i)
		if len(mapCols) > 0 {
			for _, col := range mapCols {
				t.getColumnStorageOrPanic(col, alreadyLocked, currentTx)
			}
			continue
		}
		column := access.boundaryColumn(i)
		if column == "" || column[0] == '$' {
			continue
		}
		t.getColumnStorageOrPanic(column, alreadyLocked, currentTx)
	}
}
