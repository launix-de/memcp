/*
Copyright (C) 2024  Carl-Philip Hänsch

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
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"

func (t *table) ComputeColumn(name string, inputCols []string, computor scm.Scmer) {
	for i, c := range t.Columns {
		if c.Name == name {
			// found the column
			t.Columns[i].Computor = computor // set formula so delta storages and rebuild algo know how to recompute
			done := make(chan error, 6)
			shardlist := t.Shards
			if shardlist == nil {
				shardlist = t.PShards
			}
			for i, s := range shardlist {
				gls.Go(func(i int, s *storageShard) func () {
					return func() {
						defer func () {
							if r := recover(); r != nil {
								//fmt.Println("panic during compute:", r, string(debug.Stack()))
								done <- scanError{r, string(debug.Stack())}
							}
						}()
						for !s.ComputeColumn(name, inputCols, computor) {
							// couldn't compute column because delta is still active
							t.mu.Lock()
							s = s.rebuild(false)
							shardlist[i] = s
							t.mu.Unlock()
						}
						done <- nil
					}
				}(i, s))
			}
			for range shardlist {
				err := <- done // collect finish signal before return
				if err != nil {
					panic(err)
				}
			}
			return
		}
	}
	panic("column "+t.Name+"."+name+" does not exist")
}

func (s *storageShard) ComputeColumn(name string, inputCols []string, computor scm.Scmer) bool {
	if s.deletions.Count() > 0 || len(s.inserts) > 0 {
		return false // can't compute in shards with delta storage
	}

	fn := scm.OptimizeProcToSerialFunction(computor)

	cols := make([]ColumnStorage, len(inputCols))
	s.mu.Lock()
	for i, col := range inputCols {
		var ok bool
		cols[i], ok = s.columns[col]
		if !ok {
			panic("column "+s.t.Name+"."+col+" does not exist")
		}
	}
	s.mu.Unlock()
	colvalues := make([]scm.Scmer, len(cols))

	vals := make([]scm.Scmer, s.main_count) // build the stretchy value array
	for i := uint(0); i < s.main_count; i++ {
		for j, col := range cols {
			colvalues[j] = col.GetValue(i) // read values from main storage into lambda params
		}
		vals[i] = fn(colvalues...) // execute computor kernel
	}

	s.mu.Lock() // don't defer because we unlock inbetween
	store := new(StorageSCMER)
	store.values = vals
	s.columns[name] = store
	s.mu.Unlock()
	// TODO: decide whether to rebuild optimized store
	return true
}
