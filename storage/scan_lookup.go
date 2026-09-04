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
	"sync"

	"github.com/launix-de/memcp/scm"
)

const scalarSubselectOverflow = "scalar subselect returned more than one row"

// scanLookup reads one scalar through an equality-prefix index probe. It stops
// after the second visible exact match because that is sufficient to enforce
// scalar-subselect cardinality without constructing a scan reducer.
func (t *table) scanLookup(currentTx *TxContext, lookupCol string, lookupValue scm.Scmer, resultCol string) scm.Scmer {
	if t.hasTableLock() {
		t.waitTableLock(SessionStateFromTx(currentTx), querySeqFromTx(currentTx), false)
	}
	touchTempColumns(t, []string{lookupCol}, []string{resultCol})

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
		value, count := shard.scanLookup(boundary, lookupValue, resultCol, currentTx)
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
	return result
}

func (t *storageShard) scanLookup(boundary columnboundaries, lookupValue scm.Scmer, resultCol string, currentTx *TxContext) (scm.Scmer, int) {
	t.ensureLoaded()
	t.ensureMainCount(false)
	lookupStorage := t.getColumnStorageOrPanic(boundary.col, false, currentTx)
	resultStorage := t.getColumnStorageOrPanic(resultCol, false, currentTx)
	lookupReader := newCachedColumnReaderTx(lookupStorage, currentTx)
	resultReader := newCachedColumnReaderTx(resultStorage, currentTx)
	_, resultComputed := resultStorage.(*StorageComputeProxy)

	bounds := []columnboundaries{boundary}
	lower := []scm.Scmer{lookupValue}
	result := scm.NewNil()
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
			if matches == 1 {
				if recid < mainCount || resultComputed {
					result = resultReader.GetValue(recid)
				} else {
					result = t.getDelta(int(recid-mainCount), resultCol)
				}
			}
			if matches == 2 {
				return false
			}
		}
		return true
	})
	return result, matches
}
