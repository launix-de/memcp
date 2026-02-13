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
	callbackFn := scm.OptimizeProcToSerialFunction(callback)
	aggregateFn := func(...scm.Scmer) scm.Scmer { return scm.NewNil() }
	if !aggregate.IsNil() {
		aggregateFn = scm.OptimizeProcToSerialFunction(aggregate)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))
	mdataset := make([]scm.Scmer, len(callbackCols))
	for i := range cdataset {
		cdataset[i] = scm.NewNil()
	}
	for i := range mdataset {
		mdataset[i] = scm.NewNil()
	}

	// main storage
	ccols := make([]ColumnStorage, len(conditionCols))
	mcols := make([]ColumnStorage, len(callbackCols))
	for i, k := range conditionCols { // iterate over columns
		// obtain a safe pointer to column storage (loads on demand)
		ccols[i] = t.getColumnStorageOrPanic(k)
	}
	for i, k := range callbackCols { // iterate over columns
		if k == "$update" {
			mcols[i] = nil
		} else if len(k) >= 4 && k[:4] == "NEW." {
			// ignore NEW.
			mcols[i] = nil
		} else {
			mcols[i] = t.getColumnStorageOrPanic(k)
		}
	}
	// initialize main_count lazily if needed
	t.ensureMainCount(false)
	// remember current insert status (so don't scan things that are inserted during map)
	// Use a guarded lock that will always be released on panic to avoid leaked locks.
	t.mu.RLock() // lock whole shard for reading since we frequently read deletions
	locked := true
	defer func() {
		if locked {
			// Ensure lock is not leaked on early return/panic
			t.mu.RUnlock()
		}
	}()
	maxInsertIndex := len(t.inserts)

	// iterate over items (indexed)
	hadValue := false
	currentTx := CurrentTx()
	t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, func(idx uint) {
		if currentTx != nil && currentTx.Mode == TxACID {
			if !currentTx.IsVisible(t, idx) {
				return
			}
		} else {
			if t.deletions.Get(idx) {
				return // item is on delete list
			}
		}

		// prepare mdataset
		if idx < t.main_count {
			// value from main storage
			// check condition
			for i, k := range ccols { // iterate over columns
				cdataset[i] = k.GetValue(idx)
			}
			if !scm.ToBool(conditionFn(cdataset...)) {
				return // condition did not match
			}

			// call map function
			for i, k := range mcols { // iterate over columns
				if k == nil {
					// update/delete function
					mdataset[i] = scm.NewFunc(t.UpdateFunction(idx, true))
				} else {
					mdataset[i] = k.GetValue(idx)
				}
			}
		} else {
			// value from delta storage
			// prepare&call condition function
			for i, k := range conditionCols { // iterate over columns
				cdataset[i] = t.getDelta(int(idx-t.main_count), k)
			}
			// check condition
			if !scm.ToBool(conditionFn(cdataset...)) {
				return // condition did not match
			}

			// prepare&call map function
			for i, k := range callbackCols { // iterate over columns
				if k == "$update" {
					mdataset[i] = scm.NewFunc(t.UpdateFunction(idx, true))
				} else if len(k) >= 4 && k[:4] == "NEW." {
					// ignore NEW.
					mdataset[i] = scm.NewNil()
				} else {
					mdataset[i] = t.getDelta(int(idx-t.main_count), k) // fill value
				}
			}
		}
		// unlock while map callback, so we don't get into deadlocks when a user is updating
		t.mu.RUnlock()
		locked = false
		intermediate := callbackFn(mdataset...)
		akkumulator = aggregateFn(akkumulator, intermediate)
		hadValue = true
		outCount++
		t.mu.RLock()
		locked = true
	})
	// finished reading
	t.mu.RUnlock()
	locked = false
	if !hadValue {
		return scm.NewNil(), outCount
	}
	return akkumulator, outCount
}
