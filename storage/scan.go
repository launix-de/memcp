/*
Copyright (C) 2023  Carl-Philip Hänsch

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

type emptyResult struct{}

// map reduce implementation based on scheme scripts
func (t *table) scan(conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool) scm.Scmer {
	/* analyze query */
	boundaries := extractBoundaries(conditionCols, condition)
	lower, upperLast := indexFromBoundaries(boundaries)
	// give sharding hints
	for _, b := range boundaries {
		t.AddPartitioningScore([]string{b.col})
	}

	values := make(chan scm.Scmer, 4)
	gls.Go(func() {
		t.iterateShards(boundaries, func(s *storageShard) {
			// parallel scan over shards
			defer func() {
				if r := recover(); r != nil {
					//fmt.Println("panic during scan:", r, string(debug.Stack()))
					values <- scanError{r, string(debug.Stack())}
				}
			}()
			values <- s.scan(boundaries, lower, upperLast, conditionCols, condition, callbackCols, callback, aggregate, neutral)
		})
		close(values) // last scan is finished
	})
	// collect values from parallel scan
	akkumulator := neutral
	hadValue := false
	if aggregate2 != nil {
		fn := scm.OptimizeProcToSerialFunction(aggregate2)
		for intermediate := range values {
			// eat value
			switch x := intermediate.(type) {
			case scanError:
				panic(x) // cascade panic
			case emptyResult:
				// do nothing
				hadValue = hadValue // do not delete this line, otherwise it will fall through to default
			default:
				akkumulator = fn(akkumulator, intermediate)
				hadValue = true
			}
		}
		if !hadValue && isOuter {
			akkumulator = fn(akkumulator, scm.Apply(callback, make([]scm.Scmer, len(callbackCols))...)) // outer join: push one NULL row
		}
		return akkumulator
	} else if aggregate != nil {
		fn := scm.OptimizeProcToSerialFunction(aggregate)
		for intermediate := range values {
			// eat value
			switch x := intermediate.(type) {
			case scanError:
				panic(x) // cascade panic
			case emptyResult:
				// do nothing
				hadValue = hadValue // do not delete this line, otherwise it will fall through to default
			default:
				akkumulator = fn(akkumulator, intermediate)
				hadValue = true
			}
		}
		if !hadValue && isOuter {
			akkumulator = fn(akkumulator, scm.Apply(callback, make([]scm.Scmer, len(callbackCols))...)) // outer join: push one NULL row
		}
		return akkumulator
	} else {
		for intermediate := range values {
			// eat value
			switch intermediate.(type) { // eat up values and forget
			case scanError:
				panic(intermediate) // cascade panic
			case emptyResult:
				// do nothing
				hadValue = hadValue // do not delete this line, otherwise it will fall through to default
			default:
				hadValue = true
			}
		}
		if !hadValue && isOuter {
			scm.Apply(callback, make([]scm.Scmer, len(callbackCols))...) // outer join: push one NULL row
		}
		return akkumulator
	}
}

func (t *storageShard) scan(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
	akkumulator := neutral

	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	callbackFn := scm.OptimizeProcToSerialFunction(callback)
	aggregateFn := func(...scm.Scmer) scm.Scmer { return nil }
	if aggregate != nil {
		aggregateFn = scm.OptimizeProcToSerialFunction(aggregate)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))
	mdataset := make([]scm.Scmer, len(callbackCols))

	// main storage
	ccols := make([]ColumnStorage, len(conditionCols))
	mcols := make([]ColumnStorage, len(callbackCols))
	for i, k := range conditionCols { // iterate over columns
		// obtain a safe pointer to column storage (loads on demand)
		ccols[i] = t.getColumnStorageOrPanic(k)
	}
	for i, k := range callbackCols { // iterate over columns
		if string(k) == "$update" {
			mcols[i] = nil
		} else if len(k) >= 4 && k[:4] == "NEW." {
			// ignore NEW.
			mcols[i] = nil
		} else {
			mcols[i] = t.getColumnStorageOrPanic(k)
		}
	}
	// initialize main_count lazily if needed
	t.ensureMainCount()
	// remember current insert status (so don't scan things that are inserted during map)
	t.mu.RLock() // lock whole shard for reading since we frequently read deletions
	maxInsertIndex := len(t.inserts)

	// iterate over items (indexed)
	hadValue := false
	t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, func(idx uint) {
		if t.deletions.Get(idx) {
			return // item is on delete list
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
					mdataset[i] = t.UpdateFunction(idx, true)
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
					mdataset[i] = t.UpdateFunction(idx, true)
				} else if len(k) >= 4 && k[:4] == "NEW." {
					// ignore NEW.
				} else {
					mdataset[i] = t.getDelta(int(idx-t.main_count), k) // fill value
				}
			}
		}
		t.mu.RUnlock() // unlock while map callback, so we don't get into deadlocks when a user is updating
		intermediate := callbackFn(mdataset...)
		akkumulator = aggregateFn(akkumulator, intermediate)
		hadValue = true
		t.mu.RLock()
	})
	t.mu.RUnlock() // finished reading
	if !hadValue {
		return emptyResult{}
	} else {
		return akkumulator
	}
}
