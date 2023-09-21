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

import "github.com/launix-de/memcp/scm"

type scanError struct {
	r interface{}
}

/* TODO:

 - LEFT JOIN handling -> emit a NULL line on certain conditions
 - ORDER: tell a list of columns to sort (for delta storage: temporaryly sort; merge with main)
 - LIMIT: scan-side limiters
 - ORDER+LIMIT: sync before executing map OR make map side-effect free
 - separate group/collect/sort nodes to add values to (maybe this is the way to do ORDER??)
 - group/collect/sort nodes -> (create sortfn equalfn mergefn offset limit) (add item); (scan mapfn) (merge othersortwindow) -> implements a sorted window; use merge as aggregator, create as neutral element

*/

// map reduce implementation based on scheme scripts
func (t *table) scan(condition scm.Scmer, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
	//start := time.Now() // time measurement

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
					values <- scanError{r}
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
		for {
			if rest == 0 {
				return akkumulator
			}
			// eat value
			intermediate := <- values
			switch x := intermediate.(type) {
				case scanError:
					panic(x.r) // cascade panic
				default:
					akkumulator = scm.Apply(aggregate, []scm.Scmer{akkumulator, intermediate,})
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
					panic(x.r) // cascade panic
			}
			rest = rest - 1
		}
	}
	// fmt.Sprint(time.Since(start))
}

func (t *storageShard) scan(boundaries boundaries, condition scm.Scmer, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
	//start := time.Now() // time measurement
	akkumulator := neutral

	cargs := condition.(scm.Proc).Params.([]scm.Scmer) // list of arguments condition
	margs := callback.(scm.Proc).Params.([]scm.Scmer) // list of arguments map
	cdataset := make([]scm.Scmer, len(cargs))
	mdataset := make([]scm.Scmer, len(margs))

	// TODO: implement a proxy for scripts that routes the scan between nodes first

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
	maxInsertIndex := len(t.inserts)
	// iterate over items (indexed)
	for idx := range t.iterateIndex(boundaries) {
		if _, ok := t.deletions[idx]; ok {
			continue // item is on delete list
		}
		// check condition
		for i, k := range ccols { // iterate over columns
			cdataset[i] = k.getValue(idx)
		}
		if (!scm.ToBool(scm.Apply(condition, cdataset))) {
			continue // condition did not match
		}

		// call map function
		for i, k := range mcols { // iterate over columns
			if k == nil {
				// update/delete function
				mdataset[i] = t.UpdateFunction(idx, true)
			} else {
				mdataset[i] = k.getValue(idx)
			}
		}
		intermediate := scm.Apply(callback, mdataset)
		if aggregate != nil {
			akkumulator = scm.Apply(aggregate, []scm.Scmer{akkumulator, intermediate,})
		}
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
		if (!scm.ToBool(scm.Apply(condition, cdataset))) {
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
		intermediate := scm.Apply(callback, mdataset)
		if aggregate != nil {
			akkumulator = scm.Apply(aggregate, []scm.Scmer{akkumulator, intermediate,})
		}
	}
	//return fmt.Sprint(time.Since(start))
	return akkumulator
}
