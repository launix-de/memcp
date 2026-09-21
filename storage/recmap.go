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
	"sort"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// TagRecMap is the custom Scmer tag for query-local functional row mappings.
const TagRecMap = 104

// recMapTarget is a physical row identity valid for the query which built it.
// A nil shard denotes the SQL-NULL result of an outer/scalar-first lookup.
type recMapTarget struct {
	shard *storageShard
	recid uint32
}

// recMapShard stores only rows in the input RecSet. sourceRecIDs is sorted, so
// a narrow scan window costs O(window rows), not O(source table rows).
type recMapShard struct {
	sourceShard  *storageShard
	sourceRecIDs []uint32
	targets      []recMapTarget
}

// recMap is an immutable query-local source-row -> optional target-row carrier.
// Like recSet it retains no transaction or cancellation state and must never be
// persisted or reused by another query. Physical row identities can change on
// rebuild, so a persistent cache needs a separate generation contract.
type recMap struct {
	source *table
	target *table
	shards []recMapShard
	count  int64
}

func NewRecMapScmer(rm *recMap) scm.Scmer {
	return scm.NewCustom(TagRecMap, unsafe.Pointer(rm))
}

func RecMapFromScmer(value scm.Scmer) *recMap {
	return (*recMap)(value.Custom(TagRecMap))
}

func (r *recMap) String() string {
	if r == nil || r.source == nil || r.target == nil {
		return "(recmap nil)"
	}
	return fmt.Sprintf("(recmap %q.%q %q.%q %d)",
		r.source.schema.Name, r.source.Name, r.target.schema.Name, r.target.Name, r.count)
}

func (r *recMap) lookup(shard *storageShard, recid uint32) (recMapTarget, bool) {
	if r == nil || shard == nil {
		return recMapTarget{}, false
	}
	for i := range r.shards {
		part := &r.shards[i]
		if part.sourceShard != shard {
			continue
		}
		position := sort.Search(len(part.sourceRecIDs), func(i int) bool {
			return part.sourceRecIDs[i] >= recid
		})
		if position == len(part.sourceRecIDs) || part.sourceRecIDs[position] != recid {
			return recMapTarget{}, false
		}
		return part.targets[position], true
	}
	return recMapTarget{}, false
}

type recMapSourceRow struct {
	shard *storageShard
	recid uint32
	key   []scm.Scmer
}

func (r *recSet) collectRecMapSourceRows(currentTx *TxContext, sourceKeyCols []string) []recMapSourceRow {
	if r == nil || r.table == nil || r.count == 0 {
		return nil
	}
	rows := make([]recMapSourceRow, 0, r.count)
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)
	for partIndex := range r.shards {
		part := &r.shards[partIndex]
		if part.count == 0 {
			continue
		}
		shard := part.shard
		release := shard.acquireReadForScan(currentTx)
		func() {
			defer release()
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			shard.ensureLoaded()
			skipShardReadLock := shard.hasWriteOwnerForTx(currentTx)
			shard.ensureMainCount(skipShardReadLock)
			columns := make([]ColumnStorage, len(sourceKeyCols))
			for i, column := range sourceKeyCols {
				columns[i] = shard.getColumnStorageOrPanic(column, skipShardReadLock, currentTx)
			}
			if !skipShardReadLock {
				shard.mu.RLock()
				if shard.t.hasTableLock() {
					shard.mu.RUnlock()
					shard.t.waitTableLock(ss, querySeq, false)
					shard.mu.RLock()
				}
				defer shard.mu.RUnlock()
			}
			acidMode := currentTx != nil && currentTx.Mode == TxACID
			visibleUpper := shard.main_count + uint32(len(shard.inserts))
			part.forEachID(func(recid uint32) bool {
				if recid >= visibleUpper {
					return true
				}
				if acidMode {
					if !currentTx.IsVisible(shard, recid) {
						return true
					}
				} else if shard.deletions.Get(uint(recid)) {
					return true
				}
				key := make([]scm.Scmer, len(sourceKeyCols))
				for i, column := range sourceKeyCols {
					if recid < shard.main_count {
						key[i] = columns[i].GetValue(recid)
					} else if _, proxy := columns[i].(*StorageComputeProxy); proxy {
						key[i] = columns[i].GetValue(recid)
					} else {
						key[i] = shard.getDelta(int(recid-shard.main_count), column)
					}
				}
				rows = append(rows, recMapSourceRow{shard: shard, recid: recid, key: key})
				return true
			})
		}()
	}
	return rows
}

func recMapKeyHasNull(key []scm.Scmer) bool {
	for _, value := range key {
		if value.IsNil() {
			return true
		}
	}
	return false
}

// projectRecMap builds only mappings for sourceDomain. Equal correlation keys
// share one batched target probe, retaining source multiplicity in the map.
// The first matching target implements scalar-first/LEFT semantics; callers
// requiring an ordered first row must provide an access path with that policy
// in a later, richer probe descriptor.
func projectRecMap(currentTx *TxContext, sourceDomain *recSet, sourceKeyCols []string, sourceKeyFn scm.Scmer, target *table, targetKeyCols []string) *recMap {
	if sourceDomain == nil || sourceDomain.table == nil {
		return &recMap{target: target}
	}
	if target == nil || len(sourceKeyCols) == 0 || len(targetKeyCols) == 0 ||
		sourceKeyFn.IsNil() && len(sourceKeyCols) != len(targetKeyCols) {
		panic("recmap_project_join: source and target keys must be non-empty and need equal widths without a key mapper")
	}
	rows := sourceDomain.collectRecMapSourceRows(currentTx, sourceKeyCols)
	if !sourceKeyFn.IsNil() {
		mapper := scm.PrepareSerialProc(sourceKeyFn)
		for i := range rows {
			mapped := mapper.Call(rows[i].key)
			if !mapped.IsSlice() || len(mapped.Slice()) != len(targetKeyCols) {
				panic("recmap_project_join: source key mapper must return one value per target key column")
			}
			rows[i].key = append([]scm.Scmer(nil), mapped.Slice()...)
		}
	}
	result := &recMap{source: sourceDomain.table, target: target, count: int64(len(rows))}
	if len(rows) == 0 {
		return result
	}

	order := make([]int, len(rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return compareProjectKey(rows[order[i]].key, rows[order[j]].key) < 0
	})
	uniqueKeys := recSetProjectKeys{width: len(targetKeyCols), values: make([]scm.Scmer, 0, len(rows)*len(targetKeyCols))}
	for i, rowIndex := range order {
		key := rows[rowIndex].key
		if recMapKeyHasNull(key) || (i > 0 && compareProjectKey(rows[order[i-1]].key, key) == 0) {
			continue
		}
		uniqueKeys.values = append(uniqueKeys.values, key...)
	}
	targetRows := target.projectJoinKeysToRecSet(
		currentTx, targetKeyCols, uniqueKeys, SessionStateFromTx(currentTx)).collectRecMapSourceRows(currentTx, targetKeyCols)
	sort.SliceStable(targetRows, func(i, j int) bool {
		return compareProjectKey(targetRows[i].key, targetRows[j].key) < 0
	})

	targets := make([]recMapTarget, len(rows))
	for targetIndex := range targetRows {
		targetRow := &targetRows[targetIndex]
		position := sort.Search(len(order), func(i int) bool {
			return compareProjectKey(rows[order[i]].key, targetRow.key) >= 0
		})
		if position == len(order) || compareProjectKey(rows[order[position]].key, targetRow.key) != 0 {
			continue
		}
		targetRef := recMapTarget{shard: targetRow.shard, recid: targetRow.recid}
		for position < len(order) && compareProjectKey(rows[order[position]].key, targetRow.key) == 0 {
			// The first row for a duplicate target key implements scalar-first.
			if targets[order[position]].shard == nil {
				targets[order[position]] = targetRef
			}
			position++
		}
	}

	for begin := 0; begin < len(rows); {
		end := begin + 1
		for end < len(rows) && rows[end].shard == rows[begin].shard {
			end++
		}
		part := recMapShard{
			sourceShard:  rows[begin].shard,
			sourceRecIDs: make([]uint32, end-begin),
			targets:      make([]recMapTarget, end-begin),
		}
		for i := begin; i < end; i++ {
			part.sourceRecIDs[i-begin] = rows[i].recid
			part.targets[i-begin] = targets[i]
		}
		result.shards = append(result.shards, part)
		begin = end
	}
	return result
}

func (r *recMap) image() *recSet {
	if r == nil || r.target == nil {
		return &recSet{}
	}
	result := &recSet{table: r.target}
	byShard := make(map[*storageShard][]uint32)
	for _, part := range r.shards {
		for _, target := range part.targets {
			if target.shard != nil {
				byShard[target.shard] = append(byShard[target.shard], target.recid)
			}
		}
	}
	for shard, ids := range byShard {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		write := 0
		for _, id := range ids {
			if write == 0 || ids[write-1] != id {
				ids[write] = id
				write++
			}
		}
		ids = ids[:write]
		release := shard.GetRead()
		shard.mu.RLock()
		universe := shard.main_count + uint32(len(shard.inserts))
		shard.mu.RUnlock()
		release()
		part := newRecSetShardFromSortedIDs(shard, universe, ids)
		result.count += part.count
		result.shards = append(result.shards, part)
	}
	return result
}

func composeRecMaps(left, right *recMap) *recMap {
	if left == nil || right == nil || left.target != right.source {
		panic("recmap_compose: left target must be right source")
	}
	result := &recMap{source: left.source, target: right.target, count: left.count}
	rightParts := make(map[*storageShard]*recMapShard, len(right.shards))
	for i := range right.shards {
		rightParts[right.shards[i].sourceShard] = &right.shards[i]
	}
	for _, leftPart := range left.shards {
		part := recMapShard{
			sourceShard:  leftPart.sourceShard,
			sourceRecIDs: append([]uint32(nil), leftPart.sourceRecIDs...),
			targets:      make([]recMapTarget, len(leftPart.targets)),
		}
		for i, intermediate := range leftPart.targets {
			if intermediate.shard == nil {
				continue
			}
			rightPart := rightParts[intermediate.shard]
			if rightPart == nil {
				continue
			}
			position := sort.Search(len(rightPart.sourceRecIDs), func(i int) bool {
				return rightPart.sourceRecIDs[i] >= intermediate.recid
			})
			if position < len(rightPart.sourceRecIDs) && rightPart.sourceRecIDs[position] == intermediate.recid {
				part.targets[i] = rightPart.targets[position]
			}
		}
		result.shards = append(result.shards, part)
	}
	return result
}
