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
		// skip recompute if proxy is still valid (no invalidation since last compute)
		if proxy.compressed && len(proxy.delta) == 0 {
			if filter.IsNil() {
				return true // fully compressed, nothing to do
			}
			// filter given: ensure filtered rows are valid (CompressFiltered is idempotent)
			proxy.CompressFiltered(filterCols, filter)
			return true
		}
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

// scanJoinInfo describes a source table scanned by a computor and the equality
// join conditions connecting source columns to computor input columns.
type scanJoinInfo struct {
	schema    string
	table     string
	srcCols   []string // source table columns in equality filter
	inputCols []string // corresponding computor input column names
}

// extractScanJoinInfo walks a computor lambda and returns one scanJoinInfo per
// scan call found, together with the equality join pairs extracted from the
// filter lambda. If the filter cannot be analyzed, the info is returned with
// empty srcCols/inputCols so callers can fall back to full invalidation.
func extractScanJoinInfo(computor scm.Scmer) []scanJoinInfo {
	if computor.IsProc() {
		return extractScanJoinInfoBody(computor.Proc().Body)
	}
	return extractScanJoinInfoBody(computor)
}

func extractScanJoinInfoBody(expr scm.Scmer) []scanJoinInfo {
	if !expr.IsSlice() {
		return nil
	}
	items := expr.Slice()
	if len(items) >= 5 && items[0].IsSymbol() {
		sym := items[0].String()
		if sym == "scan" || sym == "scan_order" || sym == "scalar_scan" || sym == "scalar_scan_order" {
			info := scanJoinInfo{
				schema: scm.String(items[1]),
				table:  scm.String(items[2]),
			}
			// items[3] = condCols (list "col1" "col2" ...), items[4] = filter lambda
			condCols := extractStringListFromAST(items[3])
			if len(condCols) > 0 {
				info.srcCols, info.inputCols = extractEqualityJoins(items[4], condCols)
			}
			result := []scanJoinInfo{info}
			for _, item := range items[5:] {
				result = append(result, extractScanJoinInfoBody(item)...)
			}
			return result
		}
	}
	var result []scanJoinInfo
	for _, item := range items {
		result = append(result, extractScanJoinInfoBody(item)...)
	}
	return result
}

// extractStringListFromAST parses (list "a" "b" ...) into a Go string slice.
func extractStringListFromAST(expr scm.Scmer) []string {
	if !expr.IsSlice() {
		return nil
	}
	items := expr.Slice()
	if len(items) < 1 || !items[0].IsSymbol() || items[0].String() != "list" {
		return nil
	}
	result := make([]string, 0, len(items)-1)
	for _, item := range items[1:] {
		if !item.IsString() {
			return nil // non-literal condCol → bail
		}
		result = append(result, scm.String(item))
	}
	return result
}

// extractEqualityJoins inspects a filter lambda for patterns like
// (equal? filterParam (outer inputCol)) and returns matched srcCol/inputCol pairs.
// filterParam is matched by position against condCols.
func extractEqualityJoins(filterExpr scm.Scmer, condCols []string) (srcCols, inputCols []string) {
	// Unwrap lambda to get params and body
	var params []scm.Scmer
	var body scm.Scmer
	if filterExpr.IsProc() {
		proc := filterExpr.Proc()
		if proc.Params.IsSlice() {
			params = proc.Params.Slice()
		}
		body = proc.Body
	} else if filterExpr.IsSlice() {
		items := filterExpr.Slice()
		// (lambda (params...) body)
		if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "lambda" {
			if items[1].IsSlice() {
				params = items[1].Slice()
			}
			body = items[2]
		}
	}
	if len(params) == 0 || body.IsNil() {
		return nil, nil
	}

	// Build param name → index map
	paramIdx := make(map[string]int, len(params))
	for i, p := range params {
		if p.IsSymbol() {
			paramIdx[p.String()] = i
		}
	}

	// Collect equality comparisons from body
	equalities := collectEqualities(body)
	for _, eq := range equalities {
		// Check pattern: (equal? paramSym (outer inputSym)) or reversed
		pIdx, iCol := matchJoinEquality(eq[0], eq[1], paramIdx)
		if pIdx < 0 {
			pIdx, iCol = matchJoinEquality(eq[1], eq[0], paramIdx)
		}
		if pIdx >= 0 && pIdx < len(condCols) {
			srcCols = append(srcCols, condCols[pIdx])
			inputCols = append(inputCols, iCol)
		}
	}
	return srcCols, inputCols
}

// collectEqualities extracts pairs of expressions from (equal? A B) forms,
// handling (and ...) wrapping.
func collectEqualities(body scm.Scmer) [][2]scm.Scmer {
	if !body.IsSlice() {
		return nil
	}
	items := body.Slice()
	if len(items) < 1 || !items[0].IsSymbol() {
		return nil
	}
	sym := items[0].String()
	if sym == "equal?" && len(items) == 3 {
		return [][2]scm.Scmer{{items[1], items[2]}}
	}
	if sym == "and" {
		var result [][2]scm.Scmer
		for _, item := range items[1:] {
			result = append(result, collectEqualities(item)...)
		}
		return result
	}
	return nil
}

// matchJoinEquality checks if a is a filter param symbol and b is (outer inputCol).
// Returns the param index and input column name, or (-1, "") on mismatch.
func matchJoinEquality(a, b scm.Scmer, paramIdx map[string]int) (int, string) {
	if !a.IsSymbol() {
		return -1, ""
	}
	idx, ok := paramIdx[a.String()]
	if !ok {
		return -1, ""
	}
	// b must be (outer <symbol>)
	if !b.IsSlice() {
		return -1, ""
	}
	bItems := b.Slice()
	if len(bItems) != 2 || !bItems[0].IsSymbol() || bItems[0].String() != "outer" {
		return -1, ""
	}
	if !bItems[1].IsSymbol() {
		return -1, ""
	}
	return idx, bItems[1].String()
}

// buildInvalidateScan builds a scan expression that walks the keytable, matches rows
// by join key values from a trigger dict (OLD or NEW), and invokes $invalidate: closures.
// Pattern: (scan targetSchema targetTable '("inputCol1" ...) (lambda (kt.col ...) (and (equal? kt.col (get_assoc dictSym "srcCol")) ...)) '("$invalidate:colName") (lambda ($inv) ($inv)) + 0 nil false)
func buildInvalidateScan(targetSchema, targetTable, colName string, srcCols, inputCols []string, dictSym string) scm.Scmer {
	// Build filter column list: '("inputCol1" "inputCol2" ...)
	filterColElems := make([]scm.Scmer, 1+len(inputCols))
	filterColElems[0] = scm.NewSymbol("list")
	for i, col := range inputCols {
		filterColElems[1+i] = scm.NewString(col)
	}

	// Build filter lambda params: (kt.inputCol1 kt.inputCol2 ...)
	filterParams := make([]scm.Scmer, len(inputCols))
	for i, col := range inputCols {
		filterParams[i] = scm.NewSymbol(targetTable + "." + col)
	}

	// Build filter body: (equal? kt.col (get_assoc dictSym "srcCol")) per pair
	paramSyms := make([]scm.Scmer, len(inputCols))
	getAssocExprs := make([]scm.Scmer, len(srcCols))
	for i := range inputCols {
		paramSyms[i] = scm.NewSymbol(targetTable + "." + inputCols[i])
		getAssocExprs[i] = fkGetAssocExpr(dictSym, srcCols[i])
	}
	var filterBody scm.Scmer
	if len(inputCols) == 1 {
		filterBody = scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), paramSyms[0], getAssocExprs[0]})
	} else {
		parts := make([]scm.Scmer, 1+len(inputCols))
		parts[0] = scm.NewSymbol("and")
		for i := range inputCols {
			parts[1+i] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), paramSyms[i], getAssocExprs[i]})
		}
		filterBody = scm.NewSlice(parts)
	}

	// Build result col list: '("$invalidate:colName")
	invColName := "$invalidate:" + colName

	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewString(targetSchema), scm.NewString(targetTable),
		scm.NewSlice(filterColElems),
		scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(filterParams)}, filterBody)),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString(invColName)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("$inv")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("$inv")})}),
		scm.NewSymbol("+"), scm.NewInt(0), scm.NewNil(), scm.NewBool(false),
	})
}

// buildSelectiveInvalidationBody constructs the trigger body for selective cache
// invalidation. For AfterInsert/AfterDelete it scans with NEW/OLD respectively.
// For AfterUpdate it scans both OLD and NEW to invalidate both old and new group keys.
func buildSelectiveInvalidationBody(targetSchema, targetTable, colName string, srcCols, inputCols []string, timing TriggerTiming) scm.Scmer {
	switch timing {
	case AfterInsert:
		return buildInvalidateScan(targetSchema, targetTable, colName, srcCols, inputCols, "NEW")
	case AfterDelete:
		return buildInvalidateScan(targetSchema, targetTable, colName, srcCols, inputCols, "OLD")
	case AfterUpdate:
		return scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			buildInvalidateScan(targetSchema, targetTable, colName, srcCols, inputCols, "OLD"),
			buildInvalidateScan(targetSchema, targetTable, colName, srcCols, inputCols, "NEW"),
		})
	default:
		// Fallback: full invalidation
		return scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("invalidatecolumn"),
			scm.NewString(targetSchema),
			scm.NewString(targetTable),
			scm.NewString(colName),
		})
	}
}

// registerComputeTriggers installs AfterInsert/AfterUpdate/AfterDelete triggers
// on source tables so that changes automatically invalidate the computed column.
// Also installs AfterDropTable so that dropping a source table cascades to the target.
func (t *table) registerComputeTriggers(name string, computor scm.Scmer) {
	refs := extractScanJoinInfo(computor)
	targetSchema := t.schema.Name
	// Collect trigger names placed on source tables for self-cleanup
	type triggerRef struct{ schema, name string }
	var registeredNames []triggerRef
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

		// Determine trigger bodies: selective if join pairs are available
		selective := len(ref.srcCols) > 0 && len(ref.srcCols) == len(ref.inputCols)

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
			if !exists {
				var body scm.Scmer
				if selective {
					body = buildSelectiveInvalidationBody(targetSchema, t.Name, name, ref.srcCols, ref.inputCols, timing)
				} else {
					// Fall back to full invalidation
					body = scm.NewSlice([]scm.Scmer{
						scm.NewSymbol("invalidatecolumn"),
						scm.NewString(targetSchema),
						scm.NewString(t.Name),
						scm.NewString(name),
					})
				}
				srcTable.AddTrigger(TriggerDescription{
					Name:     triggerName,
					Timing:   timing,
					IsSystem: true,
					Priority: 100,
					Func:     buildFKProc(body),
				})
			}
			registeredNames = append(registeredNames, triggerRef{ref.schema, triggerName})
		}
		// AfterDropTable: when source table is dropped, drop the target table too
		dropTriggerName := ".cache:" + t.Name + ":" + name + "|" + srcTable.Name + "|" + AfterDropTable.String()
		dropExists := false
		for _, tr := range srcTable.Triggers {
			if tr.Name == dropTriggerName {
				dropExists = true
				break
			}
		}
		if !dropExists {
			srcTable.AddTrigger(TriggerDescription{
				Name:     dropTriggerName,
				Timing:   AfterDropTable,
				IsSystem: true,
				Priority: 100,
				Func: buildFKProc(scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("droptable"),
					scm.NewString(targetSchema),
					scm.NewString(t.Name),
					scm.NewBool(true),
				})),
			})
		}
		registeredNames = append(registeredNames, triggerRef{ref.schema, dropTriggerName})
	}
	// Register self-cleanup on target table: when this keytable is dropped,
	// remove all triggers we placed on source tables.
	if len(registeredNames) > 0 {
		selfCleanupName := ".self_cleanup:" + t.Name + ":" + name
		exists := false
		for _, tr := range t.Triggers {
			if tr.Name == selfCleanupName {
				exists = true
				break
			}
		}
		if !exists {
			calls := []scm.Scmer{scm.NewSymbol("begin")}
			for _, rn := range registeredNames {
				calls = append(calls, scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("droptrigger"),
					scm.NewString(rn.schema),
					scm.NewString(rn.name),
					scm.NewBool(true),
				}))
			}
			t.AddTrigger(TriggerDescription{
				Name:     selfCleanupName,
				Timing:   AfterDropTable,
				IsSystem: true,
				Priority: 50,
				Func:     buildFKProc(scm.NewSlice(calls)),
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
