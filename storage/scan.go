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

import "runtime/debug"
import "github.com/launix-de/memcp/scm"

type scanError struct {
	r interface{}
	stack string
}

/* TODO: interface Scannable (scan + scan_order) and (table schema tbl) to get a scannable */

/* TODO:

 - LEFT JOIN handling -> emit a NULL line on certain conditions

*/

// map reduce implementation based on scheme scripts
func (t *table) scan(condition scm.Scmer, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
	/* analyze query */
	boundaries := extractBoundaries(condition)

	values := make(chan scm.Scmer, 4)
	rest := 0
	for _, s := range t.Shards { // TODO: replace for loop with a more efficient algo that takes column boundaries to only pick the few of possibly thousands of shards that are within the min-max bounds
		// parallel scan over shards
		go func(s *storageShard) {
			defer func () {
				if r := recover(); r != nil {
					//fmt.Println("panic during scan:", r, string(debug.Stack()))
					values <- scanError{r, string(debug.Stack())}
				}
			}()
			values <- s.scan(boundaries, condition, callback, aggregate, neutral)
		}(s)
		rest = rest + 1
		// TODO: measure scan balance
	}
	// collect values from parallel scan
	akkumulator := neutral
	if aggregate != nil {
		fn := scm.OptimizeProcToSerialFunction(aggregate)
		for {
			if rest == 0 {
				return akkumulator
			}
			// eat value
			intermediate := <- values
			switch x := intermediate.(type) {
				case scanError:
					panic(x) // cascade panic
				default:
					akkumulator = fn(akkumulator, intermediate)
			}
			rest = rest - 1
		}
	} else {
		for {
			if rest == 0 {
				return akkumulator
			}
			switch x := (<- values).(type) { // eat up values and forget
				case scanError:
					panic(x) // cascade panic
			}
			rest = rest - 1
		}
	}
}

func (t *storageShard) scan(boundaries boundaries, condition scm.Scmer, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
	akkumulator := neutral

	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	callbackFn := scm.OptimizeProcToSerialFunction(callback)
	aggregateFn := func(...scm.Scmer) scm.Scmer {return nil}
	if aggregate != nil {
		aggregateFn = scm.OptimizeProcToSerialFunction(aggregate)
	}
	cargs := condition.(scm.Proc).Params.([]scm.Scmer) // list of arguments condition
	margs := callback.(scm.Proc).Params.([]scm.Scmer) // list of arguments map
	cdataset := make([]scm.Scmer, len(cargs))
	mdataset := make([]scm.Scmer, len(margs))

	// main storage
	ccols := make([]ColumnStorage, len(cargs))
	mcols := make([]ColumnStorage, len(margs))
	for i, k := range cargs { // iterate over columns
		var ok bool
		ccols[i], ok = t.columns[string(k.(scm.Symbol))] // find storage
		if !ok {
			panic("Column does not exist: `" + t.t.schema.Name + "`.`" + t.t.Name + "`.`" + string(k.(scm.Symbol)) + "`")
		}
	}
	for i, k := range margs { // iterate over columns
		if string(k.(scm.Symbol)) == "$update" {
			mcols[i] = nil
		} else {
			var ok bool
			mcols[i], ok = t.columns[string(k.(scm.Symbol))] // find storage
			if !ok {
				panic("Column does not exist: `" + t.t.schema.Name + "`.`" + t.t.Name + "`.`" + string(k.(scm.Symbol)) + "`")
			}
		}
	}
	// remember current insert status (so don't scan things that are inserted during map)
	t.mu.RLock() // lock whole shard for reading since we frequently read deletions
	maxInsertIndex := len(t.inserts)

	// iterate over items (indexed)
	for idx := range t.iterateIndex(boundaries) {
		if _, ok := t.deletions[idx]; ok {
			continue // item is on delete list
		}
		// check condition
		for i, k := range ccols { // iterate over columns
			cdataset[i] = k.GetValue(idx)
		}
		if (!scm.ToBool(conditionFn(cdataset...))) {
			continue // condition did not match
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
		t.mu.RUnlock() // unlock while map callback, so we don't get into deadlocks when a user is updating
		intermediate := callbackFn(mdataset...)
		akkumulator = aggregateFn(akkumulator, intermediate)
		t.mu.RLock()
	}

	// delta storage (unindexed)
	for idx := 0; idx < maxInsertIndex; idx++ { // iterate over table
		item := t.inserts[idx]
		if _, ok := t.deletions[t.main_count + uint(idx)]; ok {
			continue // item is in delete list
		}
		// prepare&call condition function
		for i, k := range cargs { // iterate over columns
			cdataset[i] = item.Get(string(k.(scm.Symbol))) // fill value
		}
		// check condition
		if (!scm.ToBool(conditionFn(cdataset...))) {
			continue // condition did not match
		}

		// prepare&call map function
		for i, k := range margs { // iterate over columns
			if string(k.(scm.Symbol)) == "$update" {
				mdataset[i] = t.UpdateFunction(t.main_count + uint(idx), true)
			} else {
				mdataset[i] = item.Get(string(k.(scm.Symbol))) // fill value
			}
		}
		t.mu.RUnlock() // unlock while map callback, so we don't get into deadlocks when a user is updating
		intermediate := callbackFn(mdataset...)
		akkumulator = aggregateFn(akkumulator, intermediate)
		t.mu.RLock()
	}
	t.mu.RUnlock() // finished reading
	return akkumulator
}
