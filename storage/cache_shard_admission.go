/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

// shardRAMSnapshot is immutable runtime admission metadata, published by the
// existing cache actor. These are measured ledger bytes, not a second estimate.
// The percentage is a cold-load threshold; warm payload remains usable locally.
type shardRAMSnapshot struct {
	Allowed bool
	Budget  int64
}

func (cm *CacheManager) shardBudget() int64 {
	return cm.memoryBudget * int64(cm.shardPercent) / 100
}

func (cm *CacheManager) publishShardRAM() {
	limit := cm.shardBudget()
	allowed := limit > 0 && cm.sizeByType[TypeShard] < limit
	if old := cm.shardCapacity.Load(); old != nil && old.Allowed == allowed && old.Budget == limit {
		return
	}
	cm.shardCapacity.Store(&shardRAMSnapshot{Allowed: allowed, Budget: limit})
}
func (cm *CacheManager) canLoadClusterShard(tbl *table) bool {
	snapshot := cm.shardCapacity.Load()
	return snapshot != nil && snapshot.Allowed && (tbl == nil || !tbl.shardRAMFull.Load())
}

// Only the cache actor changes the ledger. Readers see an admission change only
// when a quota is crossed, rather than a shared byte-counter write per allocation.
func (cm *CacheManager) adjustTableShardMemory(item *softItem, delta int64) {
	if item.shardTable() == nil || delta == 0 {
		return
	}
	if cm.tableShardMemory == nil {
		cm.tableShardMemory = make(map[*table]int64)
	}
	tbl := item.shardTable()
	used := cm.tableShardMemory[tbl] + delta
	if used == 0 {
		delete(cm.tableShardMemory, tbl)
	} else {
		cm.tableShardMemory[tbl] = used
	}
	cm.publishTableShardRAM(tbl, used)
}

func (cm *CacheManager) publishTableShardRAM(tbl *table, used int64) {
	full := used > 0 && used >= cm.shardBudget()/3
	if tbl.shardRAMFull.Load() != full {
		tbl.shardRAMFull.Store(full)
	}
}

// A pinned shard cannot be evicted during admission. Once its last reader
// releases it, the actor retries reclamation at that shard boundary.
func (cm *CacheManager) reclaimShardRAM(tbl *table) {
	if cm.opChan != nil && !cm.stopped.Load() {
		cm.opChan <- cacheOp{shardTable: tbl}
	}
}
func (cm *CacheManager) SetShardRAMPercent(percent int) {
	if cm.opChan == nil || cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{shardPercent: percent, done: done}
	<-done
}
