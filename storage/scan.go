/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "fmt"
import "time"
import "runtime/debug"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"

type scanError struct {
	r     interface{}
	stack string
}

func (s scanError) Error() string {
	return fmt.Sprint(s.r) + "\n" + s.stack // room for improvement
}

/* TODO: interface Scannable (scan + scan_order) and (table schema tbl) to get a scannable */

// optimizeScan is the Optimize hook for the scan declaration.
// It explicitly controls callback ownership for the reduce and reduce2 lambdas,
// ensuring the accumulator parameter is marked as owned (enabling _mut swaps
// like set_assoc → set_assoc_mut inside the reduce body).
func optimizeScan(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	// Optimize args 1-6 normally (schema, table, filterCols, filter, mapCols, map)
	for i := 1; i <= 6 && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	// Arg 7 (reduce callback): set callback ownership before optimizing
	if len(v) > 7 && !v[7].IsNil() {
		oc.SetCallbackOwned([]bool{true, false}) // acc is owned
		v[7], _ = oc.OptimizeSub(v[7], true)
	}
	// Arg 8 (neutral)
	if len(v) > 8 {
		v[8], _ = oc.OptimizeSub(v[8], true)
	}
	// Arg 9 (reduce2): also set callback ownership
	if len(v) > 9 && !v[9].IsNil() {
		oc.SetCallbackOwned([]bool{true, false})
		v[9], _ = oc.OptimizeSub(v[9], true)
	}
	// Arg 10 (isOuter)
	if len(v) > 10 {
		v[10], _ = oc.OptimizeSub(v[10], true)
	}
	return scm.NewSlice(v), nil
}

// scanResult bundles per-shard outputs to minimize allocations and type assertions.
type scanResult struct {
	res        scm.Scmer
	outCount   int64
	inputCount int64
	err        scanError // err.r != nil indicates an error
}

// map reduce implementation based on scheme scripts
func (t *table) scan(conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool) scm.Scmer {
	// Measure analysis time (boundary extraction, sharding hints)
	analyzeStart := time.Now()
	/* analyze query */
	boundaries := extractBoundaries(conditionCols, condition)
	lower, upperLast := indexFromBoundaries(boundaries)
	// give sharding hints
	for _, b := range boundaries {
		t.AddPartitioningScore([]string{b.col})
	}

	values := make(chan scanResult, 4)
	analyzeNs := time.Since(analyzeStart).Nanoseconds()
	// Measure execution time (parallel shard scans + collection)
	execStart := time.Now()
	var outCount int64
	var inputCount int64
	gls.Go(func() {
		t.iterateShards(boundaries, func(s *storageShard) {
			// parallel scan over shards
			defer func() {
				if r := recover(); r != nil {
					values <- scanResult{err: scanError{r, string(debug.Stack())}}
				}
			}()
			res, cnt := s.scan(boundaries, lower, upperLast, conditionCols, condition, callbackCols, callback, aggregate, neutral)
			values <- scanResult{res: res, outCount: cnt, inputCount: int64(s.Count())}
		})
		close(values) // last scan is finished
	})
	// collect values from parallel scan
	akkumulator := neutral
	hadValue := false
	if !aggregate2.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate2)
		for msg := range values {
			if msg.err.r != nil {
				panic(msg.err)
			}
			inputCount += msg.inputCount
			outCount += msg.outCount
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
		if !hadValue && isOuter {
			nullRow := make([]scm.Scmer, len(callbackCols))
			for i := range nullRow {
				nullRow[i] = scm.NewNil()
			}
			akkumulator = fn(akkumulator, scm.Apply(callback, nullRow...)) // outer join: push one NULL row
		}
		// log statistics for unordered scan (best-effort, async so it doesn't add latency)
		execNs := time.Since(execStart).Nanoseconds()
		if inputCount > int64(Settings.AnalyzeMinItems) {
			// log statistics for unordered scan (best-effort, async so it doesn't add latency)
			go func(anNs, exNs int64) {
				defer func() { _ = recover() }()
				filterEnc := ""
				if proc, ok := condition.Any().(scm.Proc); ok {
					var params []scm.Scmer
					if proc.Params.IsSlice() {
						params = proc.Params.Slice()
					} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
						params = arr
					}
					filterEnc = encodeScmerToString(proc.Body, conditionCols, params)
				}
				safeLogScan(t.schema.Name, t.Name, false, filterEnc, "", inputCount, outCount, anNs, exNs)
			}(analyzeNs, execNs)
		}
		return akkumulator
	} else if !aggregate.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate)
		for msg := range values {
			if msg.err.r != nil {
				panic(msg.err)
			}
			inputCount += msg.inputCount
			outCount += msg.outCount
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
		if !hadValue && isOuter {
			nullRow := make([]scm.Scmer, len(callbackCols))
			for i := range nullRow {
				nullRow[i] = scm.NewNil()
			}
			akkumulator = fn(akkumulator, scm.Apply(callback, nullRow...)) // outer join: push one NULL row
		}
		execNs := time.Since(execStart).Nanoseconds()
		if inputCount > int64(Settings.AnalyzeMinItems) {
			// log statistics (async)
			go func(anNs, exNs int64) {
				defer func() { _ = recover() }()
				filterEnc := ""
				if proc, ok := condition.Any().(scm.Proc); ok {
					var params []scm.Scmer
					if proc.Params.IsSlice() {
						params = proc.Params.Slice()
					} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
						params = arr
					}
					filterEnc = encodeScmerToString(proc.Body, conditionCols, params)
				}
				safeLogScan(t.schema.Name, t.Name, false, filterEnc, "", inputCount, outCount, anNs, exNs)
			}(analyzeNs, execNs)
		}
		return akkumulator
	} else {
		for msg := range values {
			if msg.err.r != nil {
				panic(msg.err)
			}
			inputCount += msg.inputCount
			outCount += msg.outCount
			hadValue = hadValue || msg.outCount > 0
		}
		if !hadValue && isOuter {
			nullRow := make([]scm.Scmer, len(callbackCols))
			for i := range nullRow {
				nullRow[i] = scm.NewNil()
			}
			scm.Apply(callback, nullRow...) // outer join: push one NULL row
		}
		execNs := time.Since(execStart).Nanoseconds()
		if inputCount > int64(Settings.AnalyzeMinItems) {
			// log statistics (async)
			go func(anNs, exNs int64) {
				defer func() { _ = recover() }()
				filterEnc := ""
				if proc, ok := condition.Any().(scm.Proc); ok {
					var params []scm.Scmer
					if proc.Params.IsSlice() {
						params = proc.Params.Slice()
					} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
						params = arr
					}
					filterEnc = encodeScmerToString(proc.Body, conditionCols, params)
				}
				safeLogScan(t.schema.Name, t.Name, false, filterEnc, "", inputCount, outCount, anNs, exNs)
			}(analyzeNs, execNs)
		}
		return akkumulator
	}
}

func (t *storageShard) scan(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) (scm.Scmer, int64) {
	akkumulator := neutral
	var outCount int64

	conditionFn := scm.OptimizeProcToSerialFunction(condition)

	// condition column readers
	ccols := make([]ColumnStorage, len(conditionCols))
	for i, k := range conditionCols {
		ccols[i] = t.getColumnStorageOrPanic(k)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	// MapReducer for map+reduce phase (builds column readers internally)
	mapper := t.OpenMapReducer(callbackCols, callback, aggregate)
	defer mapper.Close()

	// initialize main_count lazily if needed
	t.ensureMainCount(false)
	// Use a guarded lock that will always be released on panic to avoid leaked locks.
	t.mu.RLock()
	locked := true
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()
	maxInsertIndex := len(t.inserts)

	// filter phase: collect matching IDs into stack buffer, flush to MapReducer
	var buf [1024]uint32
	bufN := 0
	hadValue := false
	currentTx := CurrentTx()

	flush := func() {
		if bufN == 0 {
			return
		}
		// release lock for map+reduce (UpdateFunction needs write lock)
		t.mu.RUnlock()
		locked = false
		outCount += int64(bufN)
		akkumulator = mapper.Stream(akkumulator, buf[:bufN])
		hadValue = true
		bufN = 0
		t.mu.RLock()
		locked = true
	}

	t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, func(idx uint32) {
		if currentTx != nil && currentTx.Mode == TxACID {
			if !currentTx.IsVisible(t, uint32(idx)) {
				return
			}
		} else {
			if t.deletions.Get(idx) {
				return // item is on delete list
			}
		}

		// condition check
		if idx < t.main_count {
			for i, k := range ccols {
				cdataset[i] = k.GetValue(idx)
			}
		} else {
			for i, k := range conditionCols {
				cdataset[i] = t.getDelta(int(idx-t.main_count), k)
			}
		}
		if !scm.ToBool(conditionFn(cdataset...)) {
			return
		}

		// collect matching ID into buffer
		buf[bufN] = idx
		bufN++
		if bufN == 1024 {
			flush()
		}
	})
	flush() // flush remaining

	// finished reading
	t.mu.RUnlock()
	locked = false
	if !hadValue {
		return scm.NewNil(), outCount
	}
	return akkumulator, outCount
}
