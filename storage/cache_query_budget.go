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
type shardRAMSnapshot struct{ Used, Limit int64 }

func (cm *CacheManager) publishShardRAM() {
	next := shardRAMSnapshot{cm.persistedMemory(), cm.memoryBudget * int64(cm.shardPercent) / 100}
	if old := cm.shardCapacity.Load(); old != nil && *old == next {
		return
	}
	cm.shardCapacity.Store(&next)
}
func (cm *CacheManager) canLoadClusterShard() bool {
	snapshot := cm.shardCapacity.Load()
	return snapshot != nil && snapshot.Limit > 0 && snapshot.Used < snapshot.Limit
}
func (cm *CacheManager) SetShardRAMPercent(percent int) {
	if cm.opChan == nil || cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{shardPercent: percent, done: done}
	<-done
}
