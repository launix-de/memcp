/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

import "strings"
import "runtime/debug"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"

// newCachedColumnReader returns a per-goroutine ColumnReader for the given storage.
// For StorageEnum this gives O(1) sequential decode; for others it's a no-op.
// When the storage is OverlayBlob wrapping StorageEnum, unwraps to cache the
// enum directly (blob overlay is only used for large-value storage, not compute inputs).
func newCachedColumnReader(col ColumnStorage) ColumnReader {
	// unwrap OverlayBlob: compute inputs are stored in the base, not the blob layer
	if ob, ok := col.(*OverlayBlob); ok {
		return ob.Base.GetCachedReader()
	}
	return col.GetCachedReader()
}

func (t *table) ComputeColumn(name string, inputCols []string, computor scm.Scmer, filterCols []string, filter scm.Scmer) {
	for i, c := range t.Columns {
		if c.Name == name {
			// found the column
			t.Columns[i].Computor = computor // set formula so delta storages and rebuild algo know how to recompute
			t.Columns[i].ComputorInputCols = inputCols
			// register cache invalidation triggers on source tables
			t.registerComputeTriggers(name, computor)
			done := make(chan error, 6)
			shardlist := t.ActiveShards()
			for i, s := range shardlist {
				gls.Go(func(i int, s *storageShard) func() {
					return func() {
						defer func() {
							if r := recover(); r != nil {
								//fmt.Println("panic during compute:", r, string(debug.Stack()))
								done <- scanError{r, string(debug.Stack())}
							}
						}()
						for !s.ComputeColumn(name, inputCols, computor, filterCols, filter, len(shardlist) == 1) {
							// couldn't compute column because delta is still active
							t.mu.Lock()
							s = s.rebuild(false)
							shardlist[i] = s
							t.mu.Unlock()
							// persist new shard UUID after publishing
							t.schema.save()
						}
						done <- nil
					}
				}(i, s))
			}
			for range shardlist {
				err := <-done // collect finish signal before return
				if err != nil {
					panic(err)
				}
			}
			// update CacheManager size for temp columns
			if c.IsTemp {
				var totalRows int64
				for _, s := range shardlist {
					totalRows += int64(s.Count())
				}
				GlobalCache.UpdateSize(c, totalRows*16) // ~16 bytes per value estimate
			}
			return
		}
	}
	panic("column " + t.Name + "." + name + " does not exist")
}

func (s *storageShard) ComputeColumn(name string, inputCols []string, computor scm.Scmer, filterCols []string, filter scm.Scmer, parallel bool) bool {
	if s.deletions.Count() > 0 || len(s.inserts) > 0 {
		return false // can't compute in shards with delta storage
	}
	// We are going to mutate this shard's columns: mark shard as WRITE (not COLD)
	s.srState = WRITE
	// Ensure main_count and input storages are initialized before compute
	s.ensureMainCount(false)

	// Check if proxy already exists (idempotent re-computation)
	s.mu.RLock()
	existing := s.columns[name]
	s.mu.RUnlock()
	if proxy, ok := existing.(*StorageComputeProxy); ok {
		proxy.computor = computor // update lambda
		proxy.InvalidateAll()
		if !filter.IsNil() {
			proxy.CompressFiltered(filterCols, filter)
		} else {
			proxy.Compress()
		}
		return true
	}

	// Create new proxy
	proxy := &StorageComputeProxy{
		delta:     make(map[uint32]scm.Scmer),
		computor:  computor,
		inputCols: inputCols,
		shard:     s,
		colName:   name,
		count:     s.main_count,
	}

	s.mu.Lock()
	s.columns[name] = proxy
	s.mu.Unlock()

	// pre-free memory before allocating the compute result array
	GlobalCache.CheckPressure(int64(s.main_count) * 16)
	if !filter.IsNil() {
		proxy.CompressFiltered(filterCols, filter)
	} else {
		proxy.Compress() // eagerly compute + compress all values (same behavior as before)
	}
	return true
}

type tableRef struct{ schema, table string }

// extractScannedTables walks a Scheme expression tree and returns all
// (schema, table) pairs referenced by scan/scan_order/scalar_scan/scalar_scan_order.
func extractScannedTables(expr scm.Scmer) []tableRef {
	if expr.IsProc() {
		return extractScannedTables(expr.Proc().Body)
	}
	if !expr.IsSlice() {
		return nil
	}
	items := expr.Slice()
	if len(items) >= 3 && items[0].IsSymbol() {
		sym := items[0].String()
		if sym == "scan" || sym == "scan_order" || sym == "scalar_scan" || sym == "scalar_scan_order" {
			result := []tableRef{{scm.String(items[1]), scm.String(items[2])}}
			for _, item := range items[3:] {
				result = append(result, extractScannedTables(item)...)
			}
			return result
		}
	}
	var result []tableRef
	for _, item := range items {
		result = append(result, extractScannedTables(item)...)
	}
	return result
}

// registerComputeTriggers installs AfterInsert/AfterUpdate/AfterDelete triggers
// on source tables so that changes automatically invalidate the computed column.
func (t *table) registerComputeTriggers(name string, computor scm.Scmer) {
	refs := extractScannedTables(computor)
	targetSchema := t.schema.Name
	for _, ref := range refs {
		srcDB := GetDatabase(ref.schema)
		if srcDB == nil {
			continue
		}
		srcTable := srcDB.GetTable(ref.table)
		if srcTable == nil {
			continue
		}
		// skip self-referencing triggers
		if srcTable == t {
			continue
		}
		for _, timing := range []TriggerTiming{AfterInsert, AfterUpdate, AfterDelete} {
			triggerName := ".cache:" + t.Name + ":" + name + "|" + srcTable.Name + "|" + timing.String()
			// idempotency: skip if trigger already exists
			exists := false
			for _, tr := range srcTable.Triggers {
				if tr.Name == triggerName {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			srcTable.AddTrigger(TriggerDescription{
				Name:     triggerName,
				Timing:   timing,
				IsSystem: true,
				Priority: 100,
				Func: buildFKProc(scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("invalidatecolumn"),
					scm.NewString(targetSchema),
					scm.NewString(t.Name),
					scm.NewString(name),
				})),
			})
		}
	}
}

// removeComputeTriggers removes all cache invalidation triggers for a given
// computed column from all tables in the same database.
func (t *table) removeComputeTriggers(name string) {
	prefix := ".cache:" + t.Name + ":" + name + "|"
	for _, srcTable := range t.schema.tables.GetAll() {
		changed := false
		newTriggers := make([]TriggerDescription, 0, len(srcTable.Triggers))
		for _, tr := range srcTable.Triggers {
			if strings.HasPrefix(tr.Name, prefix) {
				changed = true
				continue
			}
			newTriggers = append(newTriggers, tr)
		}
		if changed {
			srcTable.mu.Lock()
			srcTable.Triggers = newTriggers
			srcTable.mu.Unlock()
		}
	}
}
