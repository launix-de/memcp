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

	var mu sync.Mutex
	result := scm.NewNil()
	matches := 0
	var panicValue any
	done := t.iterateShardsParallel(currentTx, boundaries, func(shard *storageShard, solo bool) {
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
		value, count := shard.scanLookupOne(boundary, lookupValue, resultCol, returnValue, currentTx)
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
		mu.Unlock()
	})
	if done != nil {
		<-done
	}
	if panicValue != nil {
		panic(panicValue)
	}
	if matches > 1 {
		panic(scalarSubselectOverflow)
	}
	if !returnValue {
		return scm.NewBool(matches != 0)
	}
	return result
}

func (t *storageShard) scanLookupOne(boundary columnboundaries, lookupValue scm.Scmer, resultCol string, returnValue bool, currentTx *TxContext) (scm.Scmer, int) {
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
	t.iterateIndexForce(currentTx, bounds, lower, lookupValue, len(t.inserts), ids[:], true, func(batch []uint32) bool {
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
	done := t.iterateShardsParallel(currentTx, boundaries, func(shard *storageShard, solo bool) {
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
	t.iterateIndexForce(currentTx, bounds, lookupValues, lookupValues[len(lookupValues)-1], len(t.inserts), ids[:], true, func(batch []uint32) bool {
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
