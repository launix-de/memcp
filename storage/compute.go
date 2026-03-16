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
import "sync/atomic"
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
	// Ensure shard is loaded from disk before we mark it WRITE (ensureLoaded
	// guards on COLD state; setting WRITE first would skip the load entirely).
	s.ensureLoaded()
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

// ComputeOrderedColumn materializes an ordered-reduce computed (ORC) column.
// The column value for each row is produced by a scan_order pass over the table:
// mapFn receives a $set closure plus any mapCols values; reduceFn threads an
// accumulator, writing results via ($set val).
//
// sortCols: ORDER BY column names (partition cols first)
// sortDirs: false=ASC, true=DESC, one per sortCol
// partCount: leading sortCols that are partition keys (0 = unpartitioned)
// mapCols:   additional input columns passed to mapFn (beyond the implicit $set closure)
// mapFn:     (lambda ($set mapCols...) ...) — passes data through to reduceFn
// reduceFn:  (lambda (acc mapped) ...) — calls ($set newVal), returns new acc
// reduceInit: initial accumulator value
func (t *table) ComputeOrderedColumn(name string, sortCols []string, sortDirs []bool, partCount int, mapCols []string, mapFn scm.Scmer, reduceFn scm.Scmer, reduceInit scm.Scmer) {
	found := false
	paramsChanged := false
	for i, c := range t.Columns {
		if c.Name == name {
			// Detect parameter changes (different OVER clause on same column).
			paramsChanged = !slicesEqual(c.OrcSortCols, sortCols) ||
				!boolSlicesEqual(c.OrcSortDirs, sortDirs) ||
				!slicesEqual(c.OrcMapCols, mapCols)
			t.Columns[i].OrcSortCols = sortCols
			t.Columns[i].OrcSortDirs = sortDirs
			t.Columns[i].OrcMapCols = mapCols
			t.Columns[i].OrcMapFn = mapFn
			t.Columns[i].OrcReduceFn = reduceFn
			t.Columns[i].OrcReduceInit = reduceInit
			found = true
			break
		}
	}
	if !found {
		panic("ComputeOrderedColumn: column " + t.Name + "." + name + " does not exist")
	}

	// Ensure every shard has an ORC proxy (lazy: no eager recompute).
	// If ORC params changed (different OVER clause), invalidate all proxies.
	for _, s := range t.ActiveShards() {
		t.initORCShard(s, name)
		if paramsChanged {
			s.mu.RLock()
			cs := s.columns[name]
			s.mu.RUnlock()
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				proxy.InvalidateAll()
			}
		}
	}

	// Register triggers: mutations → partial invalidation.
	t.registerORCTriggers(name)

	// Persist ORC parameters and trigger registrations.
	t.schema.save()
}

// initORCShard ensures a StorageComputeProxy with isOrdered=true exists on shard s.
// If a proxy already exists and has data (compressed), it is left untouched.
func (t *table) initORCShard(s *storageShard, name string) {
	s.ensureLoaded()
	s.ensureMainCount(false)

	s.mu.RLock()
	existing := s.columns[name]
	s.mu.RUnlock()

	if proxy, ok := existing.(*StorageComputeProxy); ok {
		proxy.isOrdered = true
		// Don't InvalidateAll — keep existing data; triggers handle partial invalidation.
		return
	} else {
		proxy := &StorageComputeProxy{
			delta:     make(map[uint32]scm.Scmer),
			isOrdered: true,
			shard:     s,
			colName:   name,
			count:     s.main_count,
		}
		s.mu.Lock()
		s.columns[name] = proxy
		s.mu.Unlock()
	}
}

// incrementalRecomputeORC recomputes ORC values starting from the first invalid row
// in the scan_order sequence, continuing until all requested rows are valid or
// convergence ($break) is reached. Must be called with t.orcMu held.
//
// The invalidation scan (run by triggers) has already set validMask bits to 0
// for affected rows. This function finds the earliest invalid row's sort key,
// predicts the accumulator from the last valid predecessor, and scans forward.
func (t *table) incrementalRecomputeORC(name string, requestShard *storageShard, requestIdx uint32) {
	// Prevent re-entry: GetValue during scan_order must not trigger another recompute.
	atomic.AddInt32(&t.orcRecomputing, 1)
	defer atomic.AddInt32(&t.orcRecomputing, -1)

	var col *column
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			col = t.Columns[i]
			break
		}
	}
	if col == nil || len(col.OrcSortCols) == 0 {
		panic("incrementalRecomputeORC: column '" + name + "' is not an ORC column on table " + t.Name)
	}

	// Build sort infrastructure.
	sortcolsScmer := make([]scm.Scmer, len(col.OrcSortCols))
	for i, sc := range col.OrcSortCols {
		sortcolsScmer[i] = scm.NewString(sc)
	}
	ltFn := scm.OptimizeProcToSerialFunction(scm.Eval(scm.NewSymbol("<"), &scm.Globalenv))
	gtFn := scm.OptimizeProcToSerialFunction(scm.Eval(scm.NewSymbol(">"), &scm.Globalenv))
	sortdirsFns := make([]func(...scm.Scmer) scm.Scmer, len(col.OrcSortDirs))
	for i, desc := range col.OrcSortDirs {
		if desc {
			sortdirsFns[i] = gtFn
		} else {
			sortdirsFns[i] = ltFn
		}
	}

	// Determine if the reducer has the identity property (stored value == accumulator).
	// Identity enables: skip valid prefix (Phase 1) + convergence break (Phase 3).
	// Non-identity: must compute every row but still benefits from validMask to detect
	// when the recomputed range ends.
	isIdentity := analyzeOrcSuffix(col.OrcReduceFn) == OrcSuffixIdentity

	// Build condition filter. For partitioned ORCs, restrict the scan to the
	// partition containing the requested row. This avoids scanning all partitions
	// when only one was invalidated.
	partCount := analyzeOrcPartition(col)
	var condCols []string
	var condFn scm.Scmer

	if partCount > 0 {
		// Read partition key of the requested row
		partCols := col.OrcSortCols[:partCount]
		partKeys := make([]scm.Scmer, partCount)
		for i, pc := range partCols {
			requestShard.mu.RLock()
			cs := requestShard.columns[pc]
			requestShard.mu.RUnlock()
			if cs != nil && requestIdx < requestShard.main_count {
				partKeys[i] = cs.GetValue(requestIdx)
			} else if requestIdx >= requestShard.main_count {
				partKeys[i] = requestShard.getDelta(int(requestIdx-requestShard.main_count), pc)
			}
		}
		// Filter: only rows in the same partition
		condCols = partCols
		condFn = scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			for i := 0; i < partCount; i++ {
				if scm.Less(a[i], partKeys[i]) || scm.Less(partKeys[i], a[i]) {
					return scm.NewBool(false) // different partition
				}
			}
			return scm.NewBool(true)
		})
	} else {
		condCols = []string{}
		condFn = scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	}

	// Callback: $set + $break + stored ORC value + mapCols
	// During orcRecomputing, GetValue returns nil for invalid rows.
	// The reducer uses nil to detect "needs recompute" vs "valid cached value".
	scanCallbackCols := make([]string, 0, 3+len(col.OrcMapCols))
	scanCallbackCols = append(scanCallbackCols, "$set:"+name)
	scanCallbackCols = append(scanCallbackCols, "$break")
	scanCallbackCols = append(scanCallbackCols, name) // stored ORC value (nil if invalid)
	scanCallbackCols = append(scanCallbackCols, col.OrcMapCols...)

	innerMapFn := scm.OptimizeProcToSerialFunction(col.OrcMapFn)
	scanMapFn := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		brk := args[1]
		storedVal := args[2] // nil = invalid row, non-nil = valid cached value
		innerArgs := make([]scm.Scmer, 1+len(col.OrcMapCols))
		innerArgs[0] = args[0] // $set
		copy(innerArgs[1:], args[3:])
		return scm.NewSlice([]scm.Scmer{brk, storedVal, innerMapFn(innerArgs...)})
	})

	innerReduceFn := scm.OptimizeProcToSerialFunction(col.OrcReduceFn)
	recomputeStarted := false
	scanReduceFn := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		mapped := args[1].Slice()
		brk := mapped[0]
		storedVal := mapped[1] // nil = invalid (orcRecomputing returns nil for invalid rows)
		innerMapped := mapped[2]

		if isIdentity && !recomputeStarted && !storedVal.IsNil() {
			// Phase 1 (identity only): valid row before first invalid.
			// storedVal is non-nil = row was valid. Use as accumulator, skip $set.
			return storedVal
		}
		if !recomputeStarted && storedVal.IsNil() {
			recomputeStarted = true
		}

		// Phase 2: compute new value via inner reducer (calls $set internally)
		newAcc := innerReduceFn(args[0], innerMapped)

		if isIdentity && recomputeStarted && !storedVal.IsNil() && !newAcc.IsNil() {
			// Phase 3 (identity only): valid row after recompute region.
			// Convergence: new matches stored → all subsequent are valid → $break.
			if (newAcc.IsInt() && storedVal.IsInt() && newAcc.Int() == storedVal.Int()) ||
				(newAcc.IsFloat() && storedVal.IsFloat() && newAcc.Float() == storedVal.Float()) ||
				(newAcc.IsString() && storedVal.IsString() && newAcc.String() == storedVal.String()) {
				scm.Apply(brk)
			}
		}

		return newAcc
	})

	t.scan_order(
		condCols, condFn,
		sortcolsScmer, sortdirsFns,
		0, -1,
		scanCallbackCols,
		scanMapFn,
		scanReduceFn,
		col.OrcReduceInit,
		false,
	)
}

// invalidateORCFromSortKey clears validMask bits for rows affected by a mutation.
// sortKeys is a list of values corresponding to col.OrcSortCols.
//
// For partitioned ORCs (detected via analyzeOrcPartition), only rows in the
// SAME partition are invalidated. Other partitions remain valid.
// For unpartitioned ORCs, all rows from the mutation sort key onwards.
func (t *table) invalidateORCFromSortKey(colName string, sortKeys []scm.Scmer) {
	var col *column
	for _, c := range t.Columns {
		if c.Name == colName {
			col = c
			break
		}
	}
	if col == nil || len(col.OrcSortCols) == 0 || len(sortKeys) == 0 {
		return
	}

	nCols := len(col.OrcSortCols)
	if len(sortKeys) < nCols {
		nCols = len(sortKeys)
	}
	partCount := analyzeOrcPartition(col)

	for _, s := range t.ActiveShards() {
		s.mu.RLock()
		cs := s.columns[colName]
		sortCSs := make([]ColumnStorage, nCols)
		for i := 0; i < nCols; i++ {
			sortCSs[i] = s.columns[col.OrcSortCols[i]]
		}
		s.mu.RUnlock()
		proxy, ok := cs.(*StorageComputeProxy)
		if !ok {
			continue
		}
		total := s.main_count + uint32(len(s.inserts))
		for idx := uint32(0); idx < total; idx++ {
			if s.deletions.Get(uint(idx)) {
				continue
			}
			// Read row's sort values
			rowVals := make([]scm.Scmer, nCols)
			for i := 0; i < nCols; i++ {
				if sortCSs[i] != nil && idx < s.main_count {
					rowVals[i] = sortCSs[i].GetValue(idx)
				} else if idx >= s.main_count {
					rowVals[i] = s.getDelta(int(idx-s.main_count), col.OrcSortCols[i])
				}
			}

			// Partition check: if partitioned, skip rows in OTHER partitions.
			if partCount > 0 {
				samePartition := true
				for i := 0; i < partCount && i < nCols; i++ {
					if scm.Less(rowVals[i], sortKeys[i]) || scm.Less(sortKeys[i], rowVals[i]) {
						samePartition = false
						break
					}
				}
				if !samePartition {
					continue // different partition → skip
				}
			}

			// Composite comparison on ORDER columns (after partition prefix):
			// Only invalidate rows at or past the mutation's order key.
			cmp := 0
			for i := partCount; i < nCols && cmp == 0; i++ {
				desc := i < len(col.OrcSortDirs) && col.OrcSortDirs[i]
				if scm.Less(rowVals[i], sortKeys[i]) {
					if desc {
						cmp = 1
					} else {
						cmp = -1
					}
				} else if scm.Less(sortKeys[i], rowVals[i]) {
					if desc {
						cmp = -1
					} else {
						cmp = 1
					}
				}
			}
			if cmp < 0 {
				continue // before mutation's order key within partition → skip
			}
			proxy.validMask.Set(uint(idx), false)
		}
	}
}

// registerORCTriggers installs AfterInsert/AfterUpdate/AfterDelete triggers on the
// table itself so that any mutation invalidates the ORC column.
// The triggers are idempotent (skipped if already registered).
func (t *table) registerORCTriggers(name string) {
	// Find the column to check for partition support
	var col *column
	for _, c := range t.Columns {
		if c.Name == name {
			col = c
			break
		}
	}
	hasSortKey := col != nil && len(col.OrcSortCols) > 0

	// Build composite sort key expression: (list (get_assoc dict "col1") (get_assoc dict "col2") ...)
	buildSortKeyExpr := func(dictSym string) scm.Scmer {
		if len(col.OrcSortCols) == 1 {
			// Single sort col: pass value directly (backward compat)
			return fkGetAssocExpr(dictSym, col.OrcSortCols[0])
		}
		parts := make([]scm.Scmer, 1+len(col.OrcSortCols))
		parts[0] = scm.NewSymbol("list")
		for i, sc := range col.OrcSortCols {
			parts[1+i] = fkGetAssocExpr(dictSym, sc)
		}
		return scm.NewSlice(parts)
	}

	for _, timing := range []TriggerTiming{AfterInsert, AfterUpdate, AfterDelete} {
		triggerName := ".orc:" + t.Name + ":" + name + "|" + timing.String()
		exists := false
		for _, tr := range t.Triggers {
			if tr.Name == triggerName {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		var body scm.Scmer
		if hasSortKey {
			// Invalidate from composite sort key onwards via validMask scan
			switch timing {
			case AfterInsert:
				body = scm.NewSlice([]scm.Scmer{scm.NewSymbol("invalidateorc"), scm.NewString(t.schema.Name), scm.NewString(t.Name), scm.NewString(name), buildSortKeyExpr("NEW")})
			case AfterDelete:
				body = scm.NewSlice([]scm.Scmer{scm.NewSymbol("invalidateorc"), scm.NewString(t.schema.Name), scm.NewString(t.Name), scm.NewString(name), buildSortKeyExpr("OLD")})
			case AfterUpdate:
				// For UPDATE: invalidate from both OLD and NEW positions
				body = scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("begin"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("invalidateorc"), scm.NewString(t.schema.Name), scm.NewString(t.Name), scm.NewString(name), buildSortKeyExpr("OLD")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("invalidateorc"), scm.NewString(t.schema.Name), scm.NewString(t.Name), scm.NewString(name), buildSortKeyExpr("NEW")}),
				})
			}
		} else {
			// No sort key: full column invalidation
			body = scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("invalidatecolumn"),
				scm.NewString(t.schema.Name),
				scm.NewString(t.Name),
				scm.NewString(name),
			})
		}
		t.AddTrigger(TriggerDescription{
			Name:     triggerName,
			Timing:   timing,
			IsSystem: true,
			Priority: 100,
			Func:     buildFKProc(body),
		})
	}
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
// The first element may be the symbol "list" or a resolved native function.
func extractStringListFromAST(expr scm.Scmer) []string {
	if !expr.IsSlice() {
		return nil
	}
	items := expr.Slice()
	if len(items) < 2 {
		return nil
	}
	// Accept first element as either symbol "list" or resolved native function.
	// Reject if it's a literal value (string/int/float/nil/bool).
	first := items[0]
	if first.IsSymbol() {
		if first.String() != "list" {
			return nil
		}
	} else if first.IsString() || first.IsInt() || first.IsFloat() || first.IsNil() || first.IsBool() {
		return nil
	}
	// first is symbol "list" or resolved function — extract string elements
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
		// Check pattern: (equal? paramRef (outer inputExpr)) or reversed
		pIdx, iCol := matchJoinEquality(eq[0], eq[1], paramIdx, len(params), nil)
		if pIdx < 0 {
			pIdx, iCol = matchJoinEquality(eq[1], eq[0], paramIdx, len(params), nil)
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

// matchJoinEquality checks if a is a filter param reference and b is (outer inputCol).
// a may be a symbol or NthLocalVar (compiled proc).
// b's inner expression may be a symbol, (get_column tblvar _ col _), or another
// NthLocalVar (the optimizer may hoist the outer ref into a closure capture).
// Returns the param index and input column name, or (-1, "") on mismatch.
func matchJoinEquality(a, b scm.Scmer, paramIdx map[string]int, paramCount int, computorParams []scm.Scmer) (int, string) {
	var idx int
	if a.IsSymbol() {
		var ok bool
		idx, ok = paramIdx[a.String()]
		if !ok {
			return -1, ""
		}
	} else if a.IsNthLocalVar() {
		idx = int(a.NthLocalVar())
		if idx < 0 || idx >= paramCount {
			return -1, ""
		}
	} else {
		return -1, ""
	}
	// b must be (outer <expr>)
	if b.IsSlice() {
		bItems := b.Slice()
		if len(bItems) == 2 && bItems[0].IsSymbol() && bItems[0].String() == "outer" {
			inner := bItems[1]
			if inner.IsSymbol() {
				return idx, inner.String()
			}
			if inner.IsSlice() {
				gcItems := inner.Slice()
				if len(gcItems) >= 4 && gcItems[0].IsSymbol() && gcItems[0].String() == "get_column" {
					return idx, scm.String(gcItems[3])
				}
			}
		}
	}
	// The optimizer may hoist (outer (get_column tblvar _ col _)) into a
	// NthLocalVar referencing the computor's params. Detect by checking
	// if b is an NthLocalVar whose index maps to a computor param whose name
	// encodes the original column reference.
	if b.IsNthLocalVar() && computorParams != nil {
		bIdx := int(b.NthLocalVar())
		// Filter params come first, computor params follow after paramCount
		cIdx := bIdx - paramCount
		if cIdx >= 0 && cIdx < len(computorParams) {
			p := computorParams[cIdx]
			if p.IsSymbol() {
				// Computor param name is "tblvar.col" or the stringified expression
				return idx, p.String()
			}
		}
	}
	return -1, ""
}

// findScanNode walks a computor expression and returns the scan AST node
// (as a []Scmer slice) for the given source schema+table. Returns nil if not found.
func findScanNode(expr scm.Scmer, schema, table string) []scm.Scmer {
	if expr.IsProc() {
		return findScanNode(expr.Proc().Body, schema, table)
	}
	if !expr.IsSlice() {
		return nil
	}
	items := expr.Slice()
	if len(items) >= 5 && items[0].IsSymbol() {
		sym := items[0].String()
		if (sym == "scan" || sym == "scan_order" || sym == "scalar_scan" || sym == "scalar_scan_order") &&
			scm.String(items[1]) == schema && scm.String(items[2]) == table {
			return items
		}
	}
	for _, item := range items {
		if found := findScanNode(item, schema, table); found != nil {
			return found
		}
	}
	return nil
}

// isAdditiveReduce checks whether a reduce function is the + operator.
// Handles both the unresolved symbol "+" and a resolved native function.
func isAdditiveReduce(reduce scm.Scmer) bool {
	if reduce.IsSymbol() && reduce.String() == "+" {
		return true
	}
	// The query planner may resolve "+" to its native function at compile time.
	// Detect it by testing: reduce(3, 4) == 7.
	ok := false
	func() {
		defer func() { recover() }()
		result := scm.Apply(reduce, scm.NewInt(3), scm.NewInt(4))
		ok = result.IsInt() && result.Int() == 7
	}()
	return ok
}

// isAdditiveAggregate checks whether a scan node represents an additive aggregate
// (reduce=+, neutral=0) whose mapFn contains no inner scans.
func isAdditiveAggregate(scanNode []scm.Scmer) bool {
	if len(scanNode) < 9 {
		return false
	}
	// items[7] = reduce, items[8] = neutral
	reduce := scanNode[7]
	neutral := scanNode[8]
	if !isAdditiveReduce(reduce) {
		return false
	}
	isZero := (neutral.IsInt() && neutral.Int() == 0) || (neutral.IsFloat() && neutral.Float() == 0.0)
	if !isZero {
		return false
	}
	// mapFn must not contain inner scans
	if containsScan(scanNode[6]) {
		return false
	}
	return true
}

// containsScan returns true if the expression contains a scan/scan_order/etc. call.
func containsScan(expr scm.Scmer) bool {
	if expr.IsProc() {
		return containsScan(expr.Proc().Body)
	}
	if !expr.IsSlice() {
		return false
	}
	items := expr.Slice()
	if len(items) >= 1 && items[0].IsSymbol() {
		sym := items[0].String()
		if sym == "scan" || sym == "scan_order" || sym == "scalar_scan" || sym == "scalar_scan_order" {
			return true
		}
	}
	for _, item := range items {
		if containsScan(item) {
			return true
		}
	}
	return false
}

// extractDeltaExpr transforms a scan's mapFn body by substituting parameter
// references (symbols or NthLocalVar) with (get_assoc dictSym "col") to
// reference OLD/NEW trigger dicts.
func extractDeltaExpr(mapFn scm.Scmer, dictSym string) scm.Scmer {
	var params []scm.Scmer
	var body scm.Scmer
	if mapFn.IsProc() {
		proc := mapFn.Proc()
		if proc.Params.IsSlice() {
			params = proc.Params.Slice()
		}
		body = proc.Body
	} else if mapFn.IsSlice() {
		items := mapFn.Slice()
		if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "lambda" {
			if items[1].IsSlice() {
				params = items[1].Slice()
			}
			body = items[2]
		}
	}
	if body.IsNil() {
		return scm.NewNil()
	}
	if len(params) == 0 {
		// No parameters (e.g. COUNT(*) mapFn = (lambda () 1)) — body is the delta
		return body
	}
	// Build substitution maps: symbol name -> expr AND NthLocalVar index -> expr
	// Compiled Procs use NthLocalVar for param references; uncompiled lambdas use symbols.
	symSubs := make(map[string]scm.Scmer, len(params))
	idxSubs := make(map[int]scm.Scmer, len(params))
	for i, p := range params {
		if p.IsSymbol() {
			name := p.String()
			// Extract column name: "tblvar.col" -> "col"
			col := name
			if dot := strings.LastIndex(name, "."); dot >= 0 {
				col = name[dot+1:]
			}
			expr := fkGetAssocExpr(dictSym, col)
			symSubs[name] = expr
			idxSubs[i] = expr
		}
	}
	return substituteParamRefs(body, symSubs, idxSubs)
}

// substituteParamRefs replaces symbol references and NthLocalVar references in
// an AST according to the given maps.
func substituteParamRefs(expr scm.Scmer, symSubs map[string]scm.Scmer, idxSubs map[int]scm.Scmer) scm.Scmer {
	if expr.IsSymbol() {
		if sub, ok := symSubs[expr.String()]; ok {
			return sub
		}
		return expr
	}
	if expr.IsNthLocalVar() {
		if sub, ok := idxSubs[int(expr.NthLocalVar())]; ok {
			return sub
		}
		return expr
	}
	if expr.IsSlice() {
		items := expr.Slice()
		result := make([]scm.Scmer, len(items))
		for i, item := range items {
			result[i] = substituteParamRefs(item, symSubs, idxSubs)
		}
		return scm.NewSlice(result)
	}
	return expr
}

// buildKeytableScanFilter constructs the filter column list, filter params, and
// filter body for scanning a keytable by join key values from a trigger dict.
// Shared by buildInvalidateScan and buildIncrementScan.
func buildKeytableScanFilter(targetTable string, srcCols, inputCols []string, dictSym string) (filterColElems []scm.Scmer, filterParams []scm.Scmer, filterBody scm.Scmer) {
	filterColElems = make([]scm.Scmer, 1+len(inputCols))
	filterColElems[0] = scm.NewSymbol("list")
	for i, col := range inputCols {
		filterColElems[1+i] = scm.NewString(col)
	}
	filterParams = make([]scm.Scmer, len(inputCols))
	for i, col := range inputCols {
		filterParams[i] = scm.NewSymbol(targetTable + "." + col)
	}
	paramSyms := make([]scm.Scmer, len(inputCols))
	getAssocExprs := make([]scm.Scmer, len(srcCols))
	for i := range inputCols {
		paramSyms[i] = scm.NewSymbol(targetTable + "." + inputCols[i])
		getAssocExprs[i] = fkGetAssocExpr(dictSym, srcCols[i])
	}
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
	return
}

// buildIncrementScan builds a scan expression that walks the keytable, matches rows
// by join key values from a trigger dict (OLD or NEW), and calls the $increment:
// closure to update the proxy's cached value in-place. No shard rebuild needed.
func buildIncrementScan(targetSchema, targetTable, colName string, srcCols, inputCols []string, dictSym string, deltaExpr scm.Scmer, negate bool) scm.Scmer {
	filterColElems, filterParams, filterBody := buildKeytableScanFilter(targetTable, srcCols, inputCols, dictSym)

	// Compute value expression: deltaExpr or (- 0 deltaExpr) for negation
	var valueExpr scm.Scmer
	if negate {
		valueExpr = scm.NewSlice([]scm.Scmer{scm.NewSymbol("-"), scm.NewInt(0), deltaExpr})
	} else {
		valueExpr = deltaExpr
	}

	// Result columns: (list "$increment:colName")
	resultCols := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("$increment:" + colName)})

	// Map function: (lambda ($incr) ($incr valueExpr))
	mapFn := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("$incr")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("$incr"), valueExpr}),
	})

	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewString(targetSchema), scm.NewString(targetTable),
		scm.NewSlice(filterColElems),
		scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(filterParams)}, filterBody)),
		resultCols,
		mapFn,
		scm.NewSymbol("+"), scm.NewInt(0), scm.NewNil(), scm.NewBool(false),
	})
}

// buildIncrementalBody constructs the trigger body for incremental aggregate
// updates. For AfterInsert it adds the delta, for AfterUpdate it subtracts
// OLD and adds NEW. For AfterDelete it falls back to selective invalidation
// to ensure proper group removal when the last row in a group is deleted.
func buildIncrementalBody(targetSchema, targetTable, colName string, srcCols, inputCols []string, mapFn scm.Scmer, timing TriggerTiming) scm.Scmer {
	switch timing {
	case AfterInsert:
		deltaExpr := extractDeltaExpr(mapFn, "NEW")
		return buildIncrementScan(targetSchema, targetTable, colName, srcCols, inputCols, "NEW", deltaExpr, false)
	case AfterUpdate:
		// Build runtime check: if group key unchanged → $increment, else → invalidate.
		// When the group key changes, keytable rows may be created/deleted by the
		// cleanup trigger, making the old $increment targets stale. Full invalidation
		// lets the next query recompute correctly.
		oldDelta := extractDeltaExpr(mapFn, "OLD")
		newDelta := extractDeltaExpr(mapFn, "NEW")
		incrementBody := scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			buildIncrementScan(targetSchema, targetTable, colName, srcCols, inputCols, "OLD", oldDelta, true),
			buildIncrementScan(targetSchema, targetTable, colName, srcCols, inputCols, "NEW", newDelta, false),
		})
		invalidateBody := scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("invalidatecolumn"),
			scm.NewString(targetSchema),
			scm.NewString(targetTable),
			scm.NewString(colName),
		})
		// Build (and (equal? (get_assoc "OLD" "srcCol1") (get_assoc "NEW" "srcCol1")) ...)
		keyEqualChecks := make([]scm.Scmer, len(srcCols))
		for i, col := range srcCols {
			keyEqualChecks[i] = scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("equal?"),
				fkGetAssocExpr("OLD", col),
				fkGetAssocExpr("NEW", col),
			})
		}
		var keyUnchanged scm.Scmer
		if len(keyEqualChecks) == 1 {
			keyUnchanged = keyEqualChecks[0]
		} else {
			parts := make([]scm.Scmer, 1+len(keyEqualChecks))
			parts[0] = scm.NewSymbol("and")
			copy(parts[1:], keyEqualChecks)
			keyUnchanged = scm.NewSlice(parts)
		}
		return scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("if"),
			keyUnchanged,
			incrementBody,
			invalidateBody,
		})
	case AfterDelete:
		// Incremental subtraction: subtract OLD delta from cached aggregate.
		// The COUNT column on the keytable naturally tracks group membership;
		// when COUNT reaches 0, HAVING (COUNT > 0) excludes the empty group.
		// The keytable cleanup trigger (priority 90) removes the row entirely.
		oldDelta := extractDeltaExpr(mapFn, "OLD")
		return buildIncrementScan(targetSchema, targetTable, colName, srcCols, inputCols, "OLD", oldDelta, true)
	default:
		// Fallback: full invalidation.
		return scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("invalidatecolumn"),
			scm.NewString(targetSchema),
			scm.NewString(targetTable),
			scm.NewString(colName),
		})
	}
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

		// Check if this scan is an additive aggregate eligible for incremental update
		var scanNode []scm.Scmer
		incremental := false
		if selective {
			scanNode = findScanNode(computor, ref.schema, ref.table)
			if scanNode != nil && isAdditiveAggregate(scanNode) {
				incremental = true
			}
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
			if !exists {
				var body scm.Scmer
				if incremental {
					body = buildIncrementalBody(targetSchema, t.Name, name, ref.srcCols, ref.inputCols, scanNode[6], timing)
				} else {
					// Full invalidation: correct for non-additive aggregates.
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
		// Only for internal temp tables (dot-prefixed) — never drop user base tables
		if strings.HasPrefix(t.Name, ".") {
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

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func boolSlicesEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
