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

/* TODO:

 - LEFT JOIN handling -> emit a NULL line on certain conditions
 - ORDER: tell a list of columns to sort (for delta storage: temporaryly sort; merge with main)
 - LIMIT: scan-side limiters
 - ORDER+LIMIT: sync before executing map OR make map side-effect free
 - separate group/collect/sort nodes to add values to (maybe this is the way to do ORDER??)
 - group/collect/sort nodes -> (create sortfn equalfn mergefn offset limit) (add item); (scan mapfn) (merge othersortwindow) -> implements a sorted window; use merge as aggregator, create as neutral element

*/

type shardqueue struct {
	shard *storageShard
	items []uint // TODO: refactor to chan, so we can block generating too much entries
	// TODO: maybe instead of []uint, store a func->Scmer?
	err scanError
	mcols []func(uint) scm.Scmer // map column reader
}

// TODO: helper function for priority-q. golangs implementation is kinda quirky, so do our own. container/heap especially lacks the function to test the value at front instead of popping it

// map reduce implementation based on scheme scripts
func (t *table) scan_order(condition scm.Scmer, sortcols []scm.Scmer, offset int, limit int, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {

	/* analyze query */
	boundaries := extractBoundaries(condition)

	total_limit := -1
	if limit >= 0 {
		total_limit = offset + limit
	}

	q := make([]shardqueue, 0)
	q_ := make(chan shardqueue)
	rest := 0
	for _, s := range t.Shards { // TODO: replace for loop with a more efficient algo that takes column boundaries to only pick the few of possibly thousands of shards that are within the min-max bounds
		// parallel scan over shards
		go func(s *storageShard) {
			defer func () {
				if r := recover(); r != nil {
					// fmt.Println("panic during scan:", r, string(debug.Stack()))
					q_ <- shardqueue{s, nil, scanError{r}, nil}
				}
			}()
			q_ <- s.scan_order(boundaries, condition, sortcols, total_limit)
		}(s)
		rest = rest + 1
	}
	// prepare map phase (map has to occur late and ordered)
	margs := callback.(scm.Proc).Params.([]scm.Scmer) // list of arguments map
	// collect all subchans
	for i := 0; i < rest; i++ {
		qe := <- q_
		if qe.err.r != nil {
			panic(qe) // propagate errors that occur inside inner scan
		}

		// prepare map columns
		qe.mcols = make([]func(uint) scm.Scmer, len(margs))
		for i, arg := range margs {
			if string(arg.(scm.Symbol)) == "$update" {
				qe.mcols[i] = func(idx uint) scm.Scmer {
					return qe.shard.UpdateFunction(idx, true)
				}
			} else {
				qe.mcols[i] = qe.shard.ColumnReader(string(arg.(scm.Symbol)))
			}
		}
		q = append(q, qe) // TODO: heap sink
	}

	// collect values from parallel scan
	akkumulator := neutral
	// TODO: do queue polling instead of this naive testing code
	for _, qx := range q {
		for _, idx := range qx.items {
			// prepare args for map function
			mapargs := make([]scm.Scmer, len(margs))
			for i, reader := range qx.mcols {
				mapargs[i] = reader(idx) // read column value into map argument
			}
			// call map function
			value := scm.Apply(callback, mapargs)

			// aggregate
			if aggregate != nil {
				akkumulator = scm.Apply(aggregate, []scm.Scmer{akkumulator, value,})
			}
		}
	}
	return akkumulator
}

func (t *storageShard) scan_order(boundaries boundaries, condition scm.Scmer, sortcols []scm.Scmer, limit int) (result shardqueue) {
	result.shard = t
	// TODO: mergesort sink

	cargs := condition.(scm.Proc).Params.([]scm.Scmer) // list of arguments condition
	cdataset := make([]scm.Scmer, len(cargs))

	// main storage
	ccols := make([]ColumnStorage, len(cargs))
	for i, k := range cargs { // iterate over columns
		var ok bool
		ccols[i], ok = t.columns[string(k.(scm.Symbol))] // find storage
		if !ok {
			panic("Column does not exist: `" + t.t.schema.Name + "`.`" + t.t.Name + "`.`" + string(k.(scm.Symbol)) + "`")
		}
	}
	// remember current insert status (so don't scan things that are inserted during map)
	maxInsertIndex := len(t.inserts)
	// iterate over items (indexed)
	for idx := range t.iterateIndex(boundaries) { // TODO: iterateIndex sort criteria! (and then we can break after limit iterations)
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

		result.items = append(result.items, idx)
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

		result.items = append(result.items, t.main_count + uint(idx))
	}

	// and now sort result!
	return
}

