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

// map reduce implementation based on scheme scripts
func (t *table) scan(condition scm.Scmer, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
	//start := time.Now() // time measurement

	/* analyze query */
	// TODO: move from storageShard.scan

	values := make(chan scm.Scmer, 4)
	rest := 0
	for _, s := range t.shards {
		// parallel scan over shards
		go func(s *storageShard) {
			defer func () {
				if r := recover(); r != nil {
					//fmt.Println("panic during scan:", r, string(debug.Stack()))
					values <- scanError{r}
				}
			}()
			values <- s.scan(condition, callback, aggregate, neutral)
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

func (t *storageShard) scan(condition scm.Scmer, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {
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
		ccols[i] = t.columns[string(k.(scm.Symbol))] // find storage
	}
	for i, k := range margs { // iterate over columns
		mcols[i] = t.columns[string(k.(scm.Symbol))] // find storage
	}
	// iterate over items (indexed)
	for idx := range t.iterateIndex(condition) {
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
			mdataset[i] = k.getValue(idx)
		}
		intermediate := scm.Apply(callback, mdataset)
		if aggregate != nil {
			akkumulator = scm.Apply(aggregate, []scm.Scmer{akkumulator, intermediate,})
		}
	}

	// delta storage (unindexed)
	for idx, item := range t.inserts { // iterate over table
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
			mdataset[i] = item.Get(string(k.(scm.Symbol))) // fill value
		}
		intermediate := scm.Apply(callback, mdataset)
		if aggregate != nil {
			akkumulator = scm.Apply(aggregate, []scm.Scmer{akkumulator, intermediate,})
		}
	}
	//return fmt.Sprint(time.Since(start))
	return akkumulator
}
