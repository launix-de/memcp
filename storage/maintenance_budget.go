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
	"math"
	"sync"
)

const (
	minimumMaintenanceReservation = int64(1 << 20)
	maintenanceBytesPerRow        = int64(48)
)

// RAMBudget limits the estimated temporary memory owned by maintenance work.
// It is deliberately separate from query fanout rights: queries remain latency
// oriented while rebuild/repartition workers may wait for memory. A request
// larger than the complete budget is admitted alone so mandatory maintenance
// cannot deadlock.
type RAMBudget struct {
	mu       sync.Mutex
	cond     *sync.Cond
	capacity int64
	used     int64
	peak     int64
}

type RAMBudgetStat struct {
	Capacity int64
	Used     int64
	Peak     int64
}

// RAMLease is one slice of a parent budget. Subtasks reserve from its local
// budget, keeping their lock traffic away from the global maintenance budget.
type RAMLease struct {
	parent   *RAMBudget
	size     int64
	children RAMBudget
	once     sync.Once
}

var GlobalMaintenanceRAMBudget RAMBudget

func (b *RAMBudget) conditionLocked() *sync.Cond {
	if b.cond == nil {
		b.cond = sync.NewCond(&b.mu)
	}
	return b.cond
}

func (b *RAMBudget) SetCapacity(capacity int64) {
	if capacity < 0 {
		capacity = 0
	}
	b.mu.Lock()
	b.capacity = capacity
	if b.cond != nil {
		b.cond.Broadcast()
	}
	b.mu.Unlock()
}

func (b *RAMBudget) Stat() RAMBudgetStat {
	b.mu.Lock()
	stat := RAMBudgetStat{Capacity: b.capacity, Used: b.used, Peak: b.peak}
	b.mu.Unlock()
	return stat
}

func normalizeMaintenanceReservation(size int64) int64 {
	if size < minimumMaintenanceReservation {
		return minimumMaintenanceReservation
	}
	return size
}

func (b *RAMBudget) Acquire(size int64) *RAMLease {
	size = normalizeMaintenanceReservation(size)
	b.mu.Lock()
	capacity := b.capacity
	if capacity <= 0 {
		b.mu.Unlock()
		return &RAMLease{}
	}
	if size > capacity {
		size = capacity
	}
	cond := b.conditionLocked()
	for b.used != 0 && size > capacity-b.used {
		cond.Wait()
		capacity = b.capacity
		if capacity <= 0 {
			b.mu.Unlock()
			return &RAMLease{}
		}
		if size > capacity {
			size = capacity
		}
	}
	b.used += size
	if b.used > b.peak {
		b.peak = b.used
	}
	b.mu.Unlock()

	lease := &RAMLease{parent: b, size: size}
	lease.children.SetCapacity(size)
	return lease
}

func (l *RAMLease) Release() {
	if l == nil || l.parent == nil {
		return
	}
	l.once.Do(func() {
		l.parent.mu.Lock()
		l.parent.used -= l.size
		if l.parent.cond != nil {
			l.parent.cond.Broadcast()
		}
		l.parent.mu.Unlock()
	})
}

func (l *RAMLease) Acquire(size int64) *RAMLease {
	if l == nil || l.parent == nil {
		return &RAMLease{}
	}
	return l.children.Acquire(size)
}

func (l *RAMLease) Capacity() int64 {
	if l == nil || l.size <= 0 {
		return math.MaxInt64
	}
	return l.size
}

func saturatingAdd(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

func saturatingMulDiv(value, multiplier, divisor int64) int64 {
	if value <= 0 || multiplier <= 0 || divisor <= 0 {
		return 0
	}
	if value > math.MaxInt64/multiplier {
		return math.MaxInt64
	}
	return value * multiplier / divisor
}

// estimateTableMaintenanceBytes is an O(1), lock-free estimate. In particular,
// it must never call ComputeSize, collectStatistics, load a shard, or inspect
// column storage. Published bytes are scaled with the fresher DML-maintained row
// estimate. The per-row allowance covers recid slices, masks and translation
// maps allocated by rebuild and repartition.
func estimateTableMaintenanceBytes(t *table) int64 {
	stats := t.statistics()
	rows := int64(t.CountEstimate())
	bytes := stats.sizeBytes
	if bytes > 0 && stats.rowCount > 0 && rows > 0 && rows != stats.rowCount {
		bytes = saturatingMulDiv(bytes, rows, stats.rowCount)
	}
	if bytes <= 0 {
		columns := int64(3)
		if snapshot := t.showColumnsSnapshot.Load(); snapshot != nil && snapshot.metadata != nil && snapshot.metadata.columns != nil {
			if count := int64(len(snapshot.metadata.columns.distinctEstimates)); count > columns {
				columns = count
			}
		}
		// A Scmer slot occupies 16 bytes. This fallback is used before the first
		// byte statistic is published and intentionally depends only on immutable
		// metadata, never on the live column storages.
		bytes = saturatingMulDiv(rows, 16*columns, 1)
	}
	return normalizeMaintenanceReservation(saturatingAdd(bytes, saturatingMulDiv(rows, maintenanceBytesPerRow, 1)))
}

func estimateShardMaintenanceBytes(tableBytes, shardRows, tableRows int64) int64 {
	if tableRows <= 0 || shardRows <= 0 {
		return minimumMaintenanceReservation
	}
	return normalizeMaintenanceReservation(saturatingMulDiv(tableBytes, shardRows, tableRows))
}

func estimatedShardRows(shard *storageShard) int64 {
	if shard == nil {
		return 0
	}
	return int64(shard.plannerMainRows.Load()) + int64(shard.plannerDeltaRows.Load())
}
