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
	// The overwhelmingly common authentication and scalar-subquery shapes use
	// fixed schema offsets. Keep validation and generic multidimensional binding
	// out of this path so a cached plan adds no allocation or decoder dispatch.
	if len(values) == 1 && len(schema) >= scanAccessSchemaHeaderSize+scanAccessBoundaryStride &&
		schema[0].String() == scanAccessSchemaName && scm.ToInt(schema[1]) == 1 &&
		schema[scanAccessSchemaHeaderSize].String() == "equal" &&
		scm.ToInt(schema[scanAccessSchemaHeaderSize+2]) == 0 &&
		scm.ToInt(schema[scanAccessSchemaHeaderSize+3]) == 0 &&
		scm.ToInt(schema[scanAccessSchemaHeaderSize+4]) == 3 {
		if values[0].IsNil() {
			return scanLookupMiss(schema[2].String() != "exists")
		}
		projectionAt := scanAccessSchemaHeaderSize + scanAccessBoundaryStride
		switch schema[2].String() {
		case "exists":
			if len(schema) == projectionAt && scm.ToInt(schema[3]) == 0 {
				return t.scanLookupOne(currentTx, schema[scanAccessSchemaHeaderSize+1].String(), values[0], "", false)
			}
		case "value":
			if len(schema) == projectionAt+1 && scm.ToInt(schema[3]) == 1 {
				return t.scanLookupOne(currentTx, schema[scanAccessSchemaHeaderSize+1].String(), values[0], schema[projectionAt].String(), true)
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
	matchCount := int(scm.ToInt(schema[1]))
	if matchCount <= 0 {
		panic("scan_lookup schema has an invalid match-column count")
	}
	for i := 0; i < matchCount; i++ {
		boundary := access.boundary(i)
		if !matcherKindEqual(boundary.matcher, EqualMatcher) || !boundary.lowerInclusive ||
			!boundary.upperInclusive || !boundaryValueEqual(boundary.lower, boundary.upper) {
			panic("scan_lookup requires exact equality access entries")
		}
	}
	projectionCount := int(scm.ToInt(schema[3]))
	projectionAt := scanAccessSchemaHeaderSize + matchCount*scanAccessBoundaryStride

	plan := scanLookupPlan{
		access:  access,
		mapCols: schema[projectionAt : projectionAt+projectionCount],
	}
	switch schema[2].String() {
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
		mapperSlot := int(scm.ToInt(schema[4]))
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
		value := plan.access.boundary(i).lower
		if value.IsNil() {
			return scanLookupMiss(plan.consumer != scanLookupExists)
		}
	}
	if matchCount == 1 && plan.consumer != scanLookupMap {
		boundary := plan.access.boundary(0)
		resultCol := ""
		if plan.consumer == scanLookupValue {
			resultCol = plan.mapCols[0].String()
		}
		return t.scanLookupOne(
			currentTx,
			boundary.col,
			boundary.lower,
			resultCol,
			plan.consumer == scanLookupValue,
		)
	}
	if matchCount == 1 && plan.consumer == scanLookupMap {
		boundary := plan.access.boundary(0)
		mapCols := scmerSliceToStrings(plan.mapCols)
		mappedValues, matches := t.scanLookupMapOne(
			currentTx, boundary.col, boundary.lower, mapCols)
		if matches > 1 {
			panic(scalarSubselectOverflow)
		}
		if matches == 0 {
			return scm.NewNil()
		}
		mapProgram := scm.PrepareSerialProc(plan.mapper)
		return mapProgram.Call(mappedValues)
	}
	lookupCols := make([]string, matchCount)
	lookupValues := make([]scm.Scmer, matchCount)
	for i := 0; i < matchCount; i++ {
		boundary := plan.access.boundary(i)
		lookupCols[i], lookupValues[i] = boundary.col, boundary.lower
	}
	switch plan.consumer {
	case scanLookupExists:
		return t.scanLookup(currentTx, lookupCols, lookupValues, "", false)
	case scanLookupValue:
		return t.scanLookup(currentTx, lookupCols, lookupValues, plan.mapCols[0].String(), true)
	case scanLookupMap:
		mapCols := scmerSliceToStrings(plan.mapCols)
		mapProgram := scm.PrepareSerialProc(plan.mapper)
		return t.scanLookupMap(currentTx, lookupCols, lookupValues, mapCols, &mapProgram)
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
func (t *table) scanLookup(currentTx *TxContext, lookupCols []string, lookupValues []scm.Scmer, resultCol string, returnValue bool) scm.Scmer {
	validateScanLookupDimensions(len(lookupCols), len(lookupValues))
	// Keep the dominant authentication and scalar-subselect case free of the
	// per-dimension reader slices needed by composite probes.
	if len(lookupCols) == 1 {
		if lookupValues[0].IsNil() {
			return scanLookupMiss(returnValue)
		}
		return t.scanLookupOne(currentTx, lookupCols[0], lookupValues[0], resultCol, returnValue)
	}
	for _, value := range lookupValues {
		if value.IsNil() {
			return scanLookupMiss(returnValue)
		}
	}
	return t.scanLookupMany(currentTx, lookupCols, lookupValues, resultCol, returnValue)
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
func (t *table) scanLookupMap(currentTx *TxContext, lookupCols []string, lookupValues []scm.Scmer, mapCols []string, mapProgram *scm.SerialProc) scm.Scmer {
	validateScanLookupDimensions(len(lookupCols), len(lookupValues))
	for _, value := range lookupValues {
		if value.IsNil() {
			return scm.NewNil()
		}
	}
	var values []scm.Scmer
	var matches int
	if len(lookupCols) == 1 {
		values, matches = t.scanLookupMapOne(currentTx, lookupCols[0], lookupValues[0], mapCols)
	} else {
		values, matches = t.scanLookupMapMany(currentTx, lookupCols, lookupValues, mapCols)
	}
	if matches > 1 {
		panic(scalarSubselectOverflow)
	}
	if matches == 0 {
		return scm.NewNil()
	}
	return mapProgram.Call(values)
}

func (t *table) scanLookupMapOne(currentTx *TxContext, lookupCol string, lookupValue scm.Scmer, mapCols []string) ([]scm.Scmer, int) {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	touchTempColumns(t, []string{lookupCol}, mapCols)
	boundary := columnboundaries{
		col: lookupCol, matcher: EqualMatcher,
		lower: lookupValue, lowerInclusive: true,
		upper: lookupValue, upperInclusive: true,
	}

	var mu sync.Mutex
	var values []scm.Scmer
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, scanAccess{suffix: boundaries{boundary}}, func(shard *storageShard, solo bool) {
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
		localValues, count := shard.scanLookupMapOne(boundary, lookupValue, mapCols, currentTx)
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

func (t *storageShard) scanLookupMapOne(boundary columnboundaries, lookupValue scm.Scmer, mapCols []string, currentTx *TxContext) ([]scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupReader := newCachedColumnReaderTx(t.getColumnStorageOrPanic(boundary.col, false, currentTx), currentTx)
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
	t.iterateIndexForce(currentTx, scanAccess{suffix: boundaries{boundary}}, []scm.Scmer{lookupValue}, lookupValue, len(t.inserts), ids[:], true, func(batch []uint32) bool {
		for _, recid := range batch {
			var actual scm.Scmer
			if recid < mainCount {
				actual = lookupReader.GetValue(recid)
			} else {
				actual = t.getDelta(int(recid-mainCount), boundary.col)
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

func (t *table) scanLookupOne(currentTx *TxContext, lookupCol string, lookupValue scm.Scmer, resultCol string, returnValue bool) scm.Scmer {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	var resultCols []string
	if returnValue {
		resultCols = []string{resultCol}
	}
	touchTempColumns(t, []string{lookupCol}, resultCols)

	boundary := columnboundaries{
		col:            lookupCol,
		matcher:        EqualMatcher,
		lower:          lookupValue,
		lowerInclusive: true,
		upper:          lookupValue,
		upperInclusive: true,
	}
	boundaries := []columnboundaries{boundary}

	state := struct {
		mu         sync.Mutex
		result     scm.Scmer
		matches    int
		panicValue any
	}{result: scm.NewNil()}
	done := t.iterateShardsParallel(currentTx, scanAccess{suffix: boundaries}, func(shard *storageShard, solo bool) {
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
		value, count := shard.scanLookupOne(lookupCol, lookupValue, resultCol, returnValue, currentTx)
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

func (t *storageShard) scanLookupOne(lookupCol string, lookupValue scm.Scmer, resultCol string, returnValue bool, currentTx *TxContext) (scm.Scmer, int) {
	boundary := columnboundaries{
		col:            lookupCol,
		matcher:        EqualMatcher,
		lower:          lookupValue,
		lowerInclusive: true,
		upper:          lookupValue,
		upperInclusive: true,
	}
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupStorage := t.getColumnStorageOrPanic(boundary.col, false, currentTx)
	lookupReader := newCachedColumnReaderTx(lookupStorage, currentTx)
	var resultReader ColumnReader
	resultComputed := false
	if returnValue {
		resultStorage := t.getColumnStorageOrPanic(resultCol, false, currentTx)
		resultReader = newCachedColumnReaderTx(resultStorage, currentTx)
		_, resultComputed = resultStorage.(*StorageComputeProxy)
	}

	bounds := []columnboundaries{boundary}
	lower := []scm.Scmer{lookupValue}
	result := scm.NewBool(false)
	matches := 0

	t.mu.RLock()
	defer t.mu.RUnlock()
	mainCount := t.main_count
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	var ids [8]uint32
	t.iterateIndexForce(currentTx, scanAccess{suffix: bounds}, lower, lookupValue, len(t.inserts), ids[:], true, func(batch []uint32) bool {
		for _, recid := range batch {
			var actual scm.Scmer
			if recid < mainCount {
				actual = lookupReader.GetValue(recid)
			} else {
				actual = t.getDelta(int(recid-mainCount), boundary.col)
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

func exactLookupBoundaries(cols []string, values []scm.Scmer) boundaries {
	result := make(boundaries, len(cols))
	for i, col := range cols {
		result[i] = columnboundaries{
			col: col, matcher: EqualMatcher,
			lower: values[i], lowerInclusive: true,
			upper: values[i], upperInclusive: true,
		}
	}
	return result
}

func (t *table) scanLookupMany(currentTx *TxContext, lookupCols []string, lookupValues []scm.Scmer, resultCol string, returnValue bool) scm.Scmer {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	var resultCols []string
	if returnValue {
		resultCols = []string{resultCol}
	}
	touchTempColumns(t, lookupCols, resultCols)
	boundaries := exactLookupBoundaries(lookupCols, lookupValues)

	var mu sync.Mutex
	var stop atomic.Bool
	result := scm.NewBool(false)
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, scanAccess{suffix: boundaries}, func(shard *storageShard, solo bool) {
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
		value, count := shard.scanLookupMany(boundaries, lookupValues, resultCol, returnValue, currentTx, &stop)
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

func (t *storageShard) scanLookupMany(bounds boundaries, lookupValues []scm.Scmer, resultCol string, returnValue bool, currentTx *TxContext, stop *atomic.Bool) (scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupReaders := make([]ColumnReader, len(bounds))
	for i, boundary := range bounds {
		lookupReaders[i] = newCachedColumnReaderTx(t.getColumnStorageOrPanic(boundary.col, false, currentTx), currentTx)
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
	t.iterateIndexForce(currentTx, scanAccess{suffix: bounds}, lookupValues, lookupValues[len(lookupValues)-1], len(t.inserts), ids[:], true, func(batch []uint32) bool {
		if stop.Load() {
			return false
		}
		for _, recid := range batch {
			exact := true
			for i, boundary := range bounds {
				var actual scm.Scmer
				if recid < mainCount {
					actual = lookupReaders[i].GetValue(recid)
				} else {
					actual = t.getDelta(int(recid-mainCount), boundary.col)
				}
				if !scm.Equal(actual, lookupValues[i]) {
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

func (t *table) scanLookupMapMany(currentTx *TxContext, lookupCols []string, lookupValues []scm.Scmer, mapCols []string) ([]scm.Scmer, int) {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	touchTempColumns(t, lookupCols, mapCols)
	boundaries := exactLookupBoundaries(lookupCols, lookupValues)

	var mu sync.Mutex
	var stop atomic.Bool
	var values []scm.Scmer
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, scanAccess{suffix: boundaries}, func(shard *storageShard, solo bool) {
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
		localValues, count := shard.scanLookupMapMany(boundaries, lookupValues, mapCols, currentTx, &stop)
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

func (t *storageShard) scanLookupMapMany(bounds boundaries, lookupValues []scm.Scmer, mapCols []string, currentTx *TxContext, stop *atomic.Bool) ([]scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupReaders := make([]ColumnReader, len(bounds))
	for i, boundary := range bounds {
		lookupReaders[i] = newCachedColumnReaderTx(t.getColumnStorageOrPanic(boundary.col, false, currentTx), currentTx)
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
	t.iterateIndexForce(currentTx, scanAccess{suffix: bounds}, lookupValues, lookupValues[len(lookupValues)-1], len(t.inserts), ids[:], true, func(batch []uint32) bool {
		if stop.Load() {
			return false
		}
		for _, recid := range batch {
			exact := true
			for i, boundary := range bounds {
				var actual scm.Scmer
				if recid < mainCount {
					actual = lookupReaders[i].GetValue(recid)
				} else {
					actual = t.getDelta(int(recid-mainCount), boundary.col)
				}
				if !scm.Equal(actual, lookupValues[i]) {
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
