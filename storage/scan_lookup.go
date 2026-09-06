/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/launix-de/memcp/scm"
)

const scalarSubselectOverflow = "scalar subselect returned more than one row"

type scanLookupConsumer uint8

const (
	scanLookupExists scanLookupConsumer = iota
	scanLookupValue
	scanLookupMap
)

// scanLookupPlan is a non-owning view over the planner-emitted [schema]
// [values] pair. Both slices refer directly to cached-plan or NoEscape storage.
type scanLookupPlan struct {
	access   scanAccess
	mapCols  []scm.Scmer
	mapper   scm.Scmer
	consumer scanLookupConsumer
}

func executeCompiledScanLookup(t *table, currentTx *TxContext, schemaValue, valuesValue scm.Scmer) scm.Scmer {
	schema := mustScmerSlice(schemaValue, "scan_lookup schema")
	values := mustScmerSlice(valuesValue, "scan_lookup values")
	meta := scanAccessSchemaMeta{}
	validHeader := false
	if len(schema) > 0 {
		meta, validHeader = decodeScanAccessHeader(schema[0])
	}
	// The overwhelmingly common authentication and scalar-subquery shapes use
	// fixed schema offsets. Keep validation and generic multidimensional binding
	// out of this path so a cached plan adds no allocation or decoder dispatch.
	if len(values) == 1 && len(schema) >= scanAccessSchemaHeaderSize+scanAccessBoundaryStride &&
		validHeader && meta.count == 1 && schema[scanAccessSchemaHeaderSize].IsCustom(TagScanBoundary) {
		boundary := ScanBoundaryFromScmer(schema[scanAccessSchemaHeaderSize])
		if boundary.Analyzer() != EqualMatcher || boundary.LowerSlot() != 0 || boundary.UpperSlot() != 0 ||
			!boundary.LowerInclusive() || !boundary.UpperInclusive() {
			return t.executeScanLookup(currentTx, parseScanLookupPlanSlices(schema, values))
		}
		if values[0].IsNil() {
			return scanLookupMiss(meta.consumer != "exists")
		}
		projectionAt := scanAccessSchemaHeaderSize + scanAccessBoundaryStride
		access := scanAccess{schema: schema, values: values, compiledCount: 1}
		switch meta.consumer {
		case "exists":
			if len(schema) == projectionAt && meta.projections == 0 {
				return t.scanLookupOne(currentTx, access, "", false)
			}
		case "value":
			if len(schema) == projectionAt+1 && meta.projections == 1 {
				return t.scanLookupOne(currentTx, access, schema[projectionAt].String(), true)
			}
		}
	}
	return t.executeScanLookup(currentTx, parseScanLookupPlanSlices(schema, values))
}

func parseScanLookupPlan(schemaValue, valuesValue scm.Scmer) scanLookupPlan {
	schema := mustScmerSlice(schemaValue, "scan_lookup schema")
	values := mustScmerSlice(valuesValue, "scan_lookup values")
	return parseScanLookupPlanSlices(schema, values)
}

func parseScanLookupPlanSlices(schema, values []scm.Scmer) scanLookupPlan {
	access, valid := scanAccessFromScheme(scm.NewSlice(schema), values, nil)
	if !valid {
		panic("scan_lookup needs a scan_access schema")
	}
	meta, _ := decodeScanAccessHeader(schema[0])
	matchCount := meta.count
	if matchCount <= 0 {
		panic("scan_lookup schema has an invalid match-column count")
	}
	for i := 0; i < matchCount; i++ {
		if !matcherKindEqual(access.boundaryAnalyzer(i), EqualMatcher) || !access.boundaryLowerInclusive(i) ||
			!access.boundaryUpperInclusive(i) || !boundaryValueEqual(access.boundValue(i, false), access.boundValue(i, true)) {
			panic("scan_lookup requires exact equality access entries")
		}
	}
	projectionCount := meta.projections
	projectionAt := scanAccessSchemaHeaderSize + matchCount*scanAccessBoundaryStride

	plan := scanLookupPlan{
		access:  access,
		mapCols: schema[projectionAt : projectionAt+projectionCount],
	}
	switch meta.consumer {
	case "exists":
		plan.consumer = scanLookupExists
		if projectionCount != 0 {
			panic("scan_lookup exists schema must not project columns")
		}
	case "value":
		plan.consumer = scanLookupValue
		if projectionCount != 1 {
			panic("scan_lookup value schema needs exactly one projection column")
		}
	case "map":
		plan.consumer = scanLookupMap
		mapperSlot := meta.mapperSlot
		if mapperSlot < 0 || mapperSlot >= len(values) {
			panic("scan_lookup map values need one mapper after the match values")
		}
		plan.mapper = values[mapperSlot]
	default:
		panic("scan_lookup schema has an unknown consumer")
	}
	return plan
}

func (t *table) executeScanLookup(currentTx *TxContext, plan scanLookupPlan) scm.Scmer {
	matchCount := plan.access.len()
	for i := 0; i < matchCount; i++ {
		value := plan.access.boundValue(i, false)
		if value.IsNil() {
			return scanLookupMiss(plan.consumer != scanLookupExists)
		}
	}
	if matchCount == 1 && plan.consumer != scanLookupMap {
		resultCol := ""
		if plan.consumer == scanLookupValue {
			resultCol = plan.mapCols[0].String()
		}
		return t.scanLookupOne(
			currentTx,
			plan.access,
			resultCol,
			plan.consumer == scanLookupValue,
		)
	}
	if matchCount == 1 && plan.consumer == scanLookupMap {
		mapCols := scmerSliceToStrings(plan.mapCols)
		mappedValues, matches := t.scanLookupMapOne(
			currentTx, plan.access, mapCols)
		if matches > 1 {
			panic(scalarSubselectOverflow)
		}
		if matches == 0 {
			return scm.NewNil()
		}
		mapProgram := scm.PrepareSerialProc(plan.mapper)
		return mapProgram.Call(mappedValues)
	}
	switch plan.consumer {
	case scanLookupExists:
		return t.scanLookup(currentTx, plan.access, "", false)
	case scanLookupValue:
		return t.scanLookup(currentTx, plan.access, plan.mapCols[0].String(), true)
	case scanLookupMap:
		mapCols := scmerSliceToStrings(plan.mapCols)
		mapProgram := scm.PrepareSerialProc(plan.mapper)
		return t.scanLookupMap(currentTx, plan.access, mapCols, &mapProgram)
	default:
		panic("invalid scan_lookup consumer")
	}
}

type scanLookupMapReader struct {
	reader   ColumnReader
	computed bool
}

// scanLookup probes an exact index prefix. Omitting resultCol turns it into a
// lightweight existence check; scalar value lookups stop after the second
// visible match to enforce scalar-subselect cardinality.
func (t *table) scanLookup(currentTx *TxContext, access scanAccess, resultCol string, returnValue bool) scm.Scmer {
	matchCount := access.len()
	// Keep the dominant authentication and scalar-subselect case free of the
	// per-dimension reader slices needed by composite probes.
	if matchCount == 1 {
		if access.boundValue(0, false).IsNil() {
			return scanLookupMiss(returnValue)
		}
		return t.scanLookupOne(currentTx, access, resultCol, returnValue)
	}
	for i := 0; i < matchCount; i++ {
		if access.boundValue(i, false).IsNil() {
			return scanLookupMiss(returnValue)
		}
	}
	return t.scanLookupMany(currentTx, access, resultCol, returnValue)
}

func validateScanLookupDimensions(columnCount, valueCount int) {
	if columnCount == 0 || columnCount != valueCount {
		panic(fmt.Sprintf("scan_lookup needs equally sized non-empty match column and value lists, got %d and %d", columnCount, valueCount))
	}
}

func scanLookupMiss(returnValue bool) scm.Scmer {
	if returnValue {
		return scm.NewNil()
	}
	return scm.NewBool(false)
}

// scanLookupMap keeps row materialization inside the point probe, but invokes
// the mapper only after global scalar cardinality has been validated and all
// shard locks have been released.
func (t *table) scanLookupMap(currentTx *TxContext, access scanAccess, mapCols []string, mapProgram *scm.SerialProc) scm.Scmer {
	for i := 0; i < access.len(); i++ {
		if access.boundValue(i, false).IsNil() {
			return scm.NewNil()
		}
	}
	var values []scm.Scmer
	var matches int
	if access.len() == 1 {
		values, matches = t.scanLookupMapOne(currentTx, access, mapCols)
	} else {
		values, matches = t.scanLookupMapMany(currentTx, access, mapCols)
	}
	if matches > 1 {
		panic(scalarSubselectOverflow)
	}
	if matches == 0 {
		return scm.NewNil()
	}
	return mapProgram.Call(values)
}

func (t *table) scanLookupMapOne(currentTx *TxContext, access scanAccess, mapCols []string) ([]scm.Scmer, int) {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	lookupCol := access.boundaryColumn(0)
	touchTempColumns(t, []string{lookupCol}, mapCols)

	var mu sync.Mutex
	var values []scm.Scmer
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, access, func(shard *storageShard, solo bool) {
		defer func() {
			if recovered := recover(); recovered != nil {
				mu.Lock()
				if panicValue == nil {
					panicValue = recovered
				}
				mu.Unlock()
			}
		}()
		if ss := SessionStateFromTx(currentTx); ss != nil && ss.IsKilledSeq(querySeqFromTx(currentTx)) {
			panic("query killed")
		}
		localValues, count := shard.scanLookupMapOne(access, mapCols, currentTx)
		if count == 0 {
			return
		}
		if solo {
			values, matches = localValues, count
			return
		}
		mu.Lock()
		if matches == 0 {
			values = localValues
		}
		matches += count
		mu.Unlock()
	})
	if done != nil {
		<-done
	}
	if panicValue != nil {
		panic(panicValue)
	}
	return values, matches
}

func (t *storageShard) scanLookupMapOne(access scanAccess, mapCols []string, currentTx *TxContext) ([]scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupCol := access.boundaryColumn(0)
	lookupValue := access.boundValue(0, false)
	lookupReader := newCachedColumnReaderTx(t.getColumnStorageOrPanic(lookupCol, false, currentTx), currentTx)
	var fixedMapReaders [8]scanLookupMapReader
	mapReaders := fixedMapReaders[:]
	if len(mapCols) <= len(fixedMapReaders) {
		mapReaders = mapReaders[:len(mapCols)]
	} else {
		mapReaders = make([]scanLookupMapReader, len(mapCols))
	}
	t.prepareScanLookupMapReaders(mapCols, mapReaders, currentTx)

	var values []scm.Scmer
	matches := 0
	t.mu.RLock()
	defer t.mu.RUnlock()
	mainCount := t.main_count
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	var ids [8]uint32
	t.iterateIndexForce(currentTx, access, len(t.inserts), ids[:], true, func(batch []uint32) bool {
		for _, recid := range batch {
			var actual scm.Scmer
			if recid < mainCount {
				actual = lookupReader.GetValue(recid)
			} else {
				actual = t.getDelta(int(recid-mainCount), lookupCol)
			}
			if !scm.Equal(actual, lookupValue) {
				continue
			}
			if acidMode {
				if !currentTx.IsVisible(t, recid) {
					continue
				}
			} else if t.deletions.Get(uint(recid)) {
				continue
			}
			matches++
			if matches == 1 {
				values = t.scanLookupMapValues(recid, mainCount, mapCols, mapReaders)
			}
			if matches == 2 {
				return false
			}
		}
		return true
	})
	return values, matches
}

func (t *storageShard) prepareScanLookupMapReaders(mapCols []string, readers []scanLookupMapReader, currentTx *TxContext) {
	for i, col := range mapCols {
		storage := t.getColumnStorageOrPanic(col, false, currentTx)
		readers[i].reader = newCachedColumnReaderTx(storage, currentTx)
		_, readers[i].computed = storage.(*StorageComputeProxy)
	}
}

func (t *storageShard) scanLookupMapValues(recid, mainCount uint32, mapCols []string, readers []scanLookupMapReader) []scm.Scmer {
	values := make([]scm.Scmer, len(mapCols))
	for i, col := range mapCols {
		if recid < mainCount || readers[i].computed {
			values[i] = readers[i].reader.GetValue(recid)
		} else {
			values[i] = t.getDelta(int(recid-mainCount), col)
		}
	}
	return values
}

func (t *table) scanLookupOne(currentTx *TxContext, access scanAccess, resultCol string, returnValue bool) scm.Scmer {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	lookupCol := access.boundaryColumn(0)
	var resultCols []string
	if returnValue {
		resultCols = []string{resultCol}
	}
	touchTempColumns(t, []string{lookupCol}, resultCols)

	state := struct {
		mu         sync.Mutex
		result     scm.Scmer
		matches    int
		panicValue any
	}{result: scm.NewNil()}
	done := t.iterateShardsParallel(currentTx, access, func(shard *storageShard, solo bool) {
		defer func() {
			if recovered := recover(); recovered != nil {
				state.mu.Lock()
				if state.panicValue == nil {
					state.panicValue = recovered
				}
				state.mu.Unlock()
			}
		}()
		if ss := SessionStateFromTx(currentTx); ss != nil && ss.IsKilledSeq(querySeqFromTx(currentTx)) {
			panic("query killed")
		}
		value, count := shard.scanLookupOne(access, resultCol, returnValue, currentTx)
		if count == 0 {
			return
		}
		if solo {
			state.result = value
			state.matches = count
			return
		}
		state.mu.Lock()
		if state.matches == 0 {
			state.result = value
		}
		state.matches += count
		state.mu.Unlock()
	})
	if done != nil {
		<-done
	}
	if state.panicValue != nil {
		panic(state.panicValue)
	}
	if state.matches > 1 {
		panic(scalarSubselectOverflow)
	}
	if !returnValue {
		return scm.NewBool(state.matches != 0)
	}
	return state.result
}

func (t *storageShard) scanLookupOne(access scanAccess, resultCol string, returnValue bool, currentTx *TxContext) (scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupCol := access.boundaryColumn(0)
	lookupValue := access.boundValue(0, false)
	lookupStorage := t.getColumnStorageOrPanic(lookupCol, false, currentTx)
	lookupReader := newCachedColumnReaderTx(lookupStorage, currentTx)
	var resultReader ColumnReader
	resultComputed := false
	if returnValue {
		resultStorage := t.getColumnStorageOrPanic(resultCol, false, currentTx)
		resultReader = newCachedColumnReaderTx(resultStorage, currentTx)
		_, resultComputed = resultStorage.(*StorageComputeProxy)
	}

	result := scm.NewBool(false)
	matches := 0

	t.mu.RLock()
	defer t.mu.RUnlock()
	mainCount := t.main_count
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	var ids [8]uint32
	t.iterateIndexForce(currentTx, access, len(t.inserts), ids[:], true, func(batch []uint32) bool {
		for _, recid := range batch {
			var actual scm.Scmer
			if recid < mainCount {
				actual = lookupReader.GetValue(recid)
			} else {
				actual = t.getDelta(int(recid-mainCount), lookupCol)
			}
			if !scm.Equal(actual, lookupValue) {
				continue
			}
			if acidMode {
				if !currentTx.IsVisible(t, recid) {
					continue
				}
			} else if t.deletions.Get(uint(recid)) {
				continue
			}

			matches++
			if matches == 1 && returnValue {
				if recid < mainCount || resultComputed {
					result = resultReader.GetValue(recid)
				} else {
					result = t.getDelta(int(recid-mainCount), resultCol)
				}
			}
			if !returnValue || matches == 2 {
				return false
			}
		}
		return true
	})
	return result, matches
}

func scanLookupColumns(access scanAccess, inline []string) []string {
	columns := inline
	if access.len() <= len(columns) {
		columns = columns[:access.len()]
	} else {
		columns = make([]string, access.len())
	}
	for i := range columns {
		columns[i] = access.boundaryColumn(i)
	}
	return columns
}

func (t *table) scanLookupMany(currentTx *TxContext, access scanAccess, resultCol string, returnValue bool) scm.Scmer {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	var fixedLookupCols [8]string
	lookupCols := scanLookupColumns(access, fixedLookupCols[:])
	var resultCols []string
	if returnValue {
		resultCols = []string{resultCol}
	}
	touchTempColumns(t, lookupCols, resultCols)

	var mu sync.Mutex
	var stop atomic.Bool
	result := scm.NewBool(false)
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, access, func(shard *storageShard, solo bool) {
		if stop.Load() {
			return
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				mu.Lock()
				if panicValue == nil {
					panicValue = recovered
				}
				mu.Unlock()
				stop.Store(true)
			}
		}()
		if ss := SessionStateFromTx(currentTx); ss != nil && ss.IsKilledSeq(querySeqFromTx(currentTx)) {
			panic("query killed")
		}
		value, count := shard.scanLookupMany(access, resultCol, returnValue, currentTx, &stop)
		if count == 0 {
			return
		}
		if solo {
			result = value
			matches = count
			return
		}
		mu.Lock()
		if matches == 0 {
			result = value
		}
		matches += count
		if !returnValue || matches > 1 {
			stop.Store(true)
		}
		mu.Unlock()
	})
	if done != nil {
		<-done
	}
	if panicValue != nil {
		panic(panicValue)
	}
	if !returnValue {
		return scm.NewBool(matches != 0)
	}
	if matches > 1 {
		panic(scalarSubselectOverflow)
	}
	if matches == 0 {
		return scm.NewNil()
	}
	return result
}

func (t *storageShard) scanLookupMany(access scanAccess, resultCol string, returnValue bool, currentTx *TxContext, stop *atomic.Bool) (scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupReaders := make([]ColumnReader, access.len())
	for i := range lookupReaders {
		lookupReaders[i] = newCachedColumnReaderTx(t.getColumnStorageOrPanic(access.boundaryColumn(i), false, currentTx), currentTx)
	}
	var resultReader ColumnReader
	resultComputed := false
	if returnValue {
		resultStorage := t.getColumnStorageOrPanic(resultCol, false, currentTx)
		resultReader = newCachedColumnReaderTx(resultStorage, currentTx)
		_, resultComputed = resultStorage.(*StorageComputeProxy)
	}

	result := scm.NewBool(false)
	matches := 0
	t.mu.RLock()
	defer t.mu.RUnlock()
	mainCount := t.main_count
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	var ids [8]uint32
	t.iterateIndexForce(currentTx, access, len(t.inserts), ids[:], true, func(batch []uint32) bool {
		if stop.Load() {
			return false
		}
		for _, recid := range batch {
			exact := true
			for i := range lookupReaders {
				var actual scm.Scmer
				if recid < mainCount {
					actual = lookupReaders[i].GetValue(recid)
				} else {
					actual = t.getDelta(int(recid-mainCount), access.boundaryColumn(i))
				}
				if !scm.Equal(actual, access.boundValue(i, false)) {
					exact = false
					break
				}
			}
			if !exact {
				continue
			}
			if acidMode {
				if !currentTx.IsVisible(t, recid) {
					continue
				}
			} else if t.deletions.Get(uint(recid)) {
				continue
			}
			matches++
			if matches == 1 && returnValue {
				if recid < mainCount || resultComputed {
					result = resultReader.GetValue(recid)
				} else {
					result = t.getDelta(int(recid-mainCount), resultCol)
				}
			}
			if !returnValue || matches == 2 {
				return false
			}
		}
		return !stop.Load()
	})
	return result, matches
}

func (t *table) scanLookupMapMany(currentTx *TxContext, access scanAccess, mapCols []string) ([]scm.Scmer, int) {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	var fixedLookupCols [8]string
	lookupCols := scanLookupColumns(access, fixedLookupCols[:])
	touchTempColumns(t, lookupCols, mapCols)

	var mu sync.Mutex
	var stop atomic.Bool
	var values []scm.Scmer
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, access, func(shard *storageShard, solo bool) {
		if stop.Load() {
			return
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				mu.Lock()
				if panicValue == nil {
					panicValue = recovered
				}
				mu.Unlock()
				stop.Store(true)
			}
		}()
		if ss := SessionStateFromTx(currentTx); ss != nil && ss.IsKilledSeq(querySeqFromTx(currentTx)) {
			panic("query killed")
		}
		localValues, count := shard.scanLookupMapMany(access, mapCols, currentTx, &stop)
		if count == 0 {
			return
		}
		if solo {
			values, matches = localValues, count
			return
		}
		mu.Lock()
		if matches == 0 {
			values = localValues
		}
		matches += count
		if matches > 1 {
			stop.Store(true)
		}
		mu.Unlock()
	})
	if done != nil {
		<-done
	}
	if panicValue != nil {
		panic(panicValue)
	}
	return values, matches
}

func (t *storageShard) scanLookupMapMany(access scanAccess, mapCols []string, currentTx *TxContext, stop *atomic.Bool) ([]scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupReaders := make([]ColumnReader, access.len())
	for i := range lookupReaders {
		lookupReaders[i] = newCachedColumnReaderTx(t.getColumnStorageOrPanic(access.boundaryColumn(i), false, currentTx), currentTx)
	}
	var fixedMapReaders [8]scanLookupMapReader
	mapReaders := fixedMapReaders[:]
	if len(mapCols) <= len(fixedMapReaders) {
		mapReaders = mapReaders[:len(mapCols)]
	} else {
		mapReaders = make([]scanLookupMapReader, len(mapCols))
	}
	t.prepareScanLookupMapReaders(mapCols, mapReaders, currentTx)

	var values []scm.Scmer
	matches := 0
	t.mu.RLock()
	defer t.mu.RUnlock()
	mainCount := t.main_count
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	var ids [8]uint32
	t.iterateIndexForce(currentTx, access, len(t.inserts), ids[:], true, func(batch []uint32) bool {
		if stop.Load() {
			return false
		}
		for _, recid := range batch {
			exact := true
			for i := range lookupReaders {
				var actual scm.Scmer
				if recid < mainCount {
					actual = lookupReaders[i].GetValue(recid)
				} else {
					actual = t.getDelta(int(recid-mainCount), access.boundaryColumn(i))
				}
				if !scm.Equal(actual, access.boundValue(i, false)) {
					exact = false
					break
				}
			}
			if !exact {
				continue
			}
			if acidMode {
				if !currentTx.IsVisible(t, recid) {
					continue
				}
			} else if t.deletions.Get(uint(recid)) {
				continue
			}
			matches++
			if matches == 1 {
				values = t.scanLookupMapValues(recid, mainCount, mapCols, mapReaders)
			}
			if matches == 2 {
				return false
			}
		}
		return !stop.Load()
	})
	return values, matches
}
