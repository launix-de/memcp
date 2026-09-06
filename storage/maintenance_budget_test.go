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
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func TestRAMBudgetBlocksBeforeStartingExcessWork(t *testing.T) {
	var budget RAMBudget
	budget.SetCapacity(2 * minimumMaintenanceReservation)
	first := budget.Acquire(2 * minimumMaintenanceReservation)

	started := make(chan *RAMLease, 1)
	go func() { started <- budget.Acquire(minimumMaintenanceReservation) }()
	select {
	case lease := <-started:
		lease.Release()
		t.Fatal("excess maintenance work acquired an exhausted budget")
	case <-time.After(20 * time.Millisecond):
	}

	first.Release()
	select {
	case lease := <-started:
		lease.Release()
	case <-time.After(time.Second):
		t.Fatal("maintenance work did not resume after memory was released")
	}
}

func TestRAMBudgetOversizedWorkRunsAlone(t *testing.T) {
	var budget RAMBudget
	budget.SetCapacity(minimumMaintenanceReservation)
	lease := budget.Acquire(8 * minimumMaintenanceReservation)
	stat := budget.Stat()
	if stat.Used != minimumMaintenanceReservation {
		t.Fatalf("oversized reservation = %d, want capacity %d", stat.Used, minimumMaintenanceReservation)
	}
	lease.Release()
}

func TestRAMLeaseContainsChildReservations(t *testing.T) {
	var budget RAMBudget
	budget.SetCapacity(4 * minimumMaintenanceReservation)
	parent := budget.Acquire(3 * minimumMaintenanceReservation)
	child := parent.Acquire(2 * minimumMaintenanceReservation)

	if got := budget.Stat().Used; got != 3*minimumMaintenanceReservation {
		t.Fatalf("child reservation changed global usage to %d", got)
	}
	if got := parent.children.Stat().Used; got != 2*minimumMaintenanceReservation {
		t.Fatalf("local child usage = %d", got)
	}
	child.Release()
	parent.Release()
}

func TestMaintenanceEstimateScalesPublishedStatistics(t *testing.T) {
	tbl := &table{}
	tbl.PlannerRowEstimate.value.Store(200_000)
	tbl.showColumnsSnapshot.Store(&tableShowColumnsSnapshot{
		metadata: &tableShowColumnsSnapshotMetadata{
			statistics: &tableStatisticsSnapshot{rowCount: 100_000, sizeBytes: 10_000_000},
		},
	})

	// 10 MB scales to 20 MB for the newer row estimate; rebuild/repartition
	// metadata adds another 48 bytes for each of the 200k rows.
	if got, want := estimateTableMaintenanceBytes(tbl), int64(29_600_000); got != want {
		t.Fatalf("maintenance estimate = %d, want %d", got, want)
	}
}

func TestTotalMemoryBytesHonorsCgroupV2Limit(t *testing.T) {
	limit := cgroupMemoryLimitBytes()
	if limit == 0 {
		t.Skip("no finite cgroup v2 memory limit")
	}
	if got := totalMemoryBytes(); got > limit {
		t.Fatalf("available process memory = %d, exceeds cgroup limit %d", got, limit)
	}
}

func BenchmarkMaintenanceEstimatePublishedRead(b *testing.B) {
	tbl := &table{}
	tbl.PlannerRowEstimate.value.Store(200_000)
	tbl.showColumnsSnapshot.Store(&tableShowColumnsSnapshot{
		metadata: &tableShowColumnsSnapshotMetadata{
			statistics: &tableStatisticsSnapshot{rowCount: 100_000, sizeBytes: 10_000_000},
		},
	})
	b.ReportAllocs()
	for b.Loop() {
		tableStatisticsBenchmarkSink.sizeBytes = estimateTableMaintenanceBytes(tbl)
	}
}

func BenchmarkRAMBudgetUncontended(b *testing.B) {
	var budget RAMBudget
	budget.SetCapacity(1 << 30)
	b.ReportAllocs()
	for b.Loop() {
		lease := budget.Acquire(minimumMaintenanceReservation)
		lease.Release()
	}
}

func BenchmarkBudgetedTableRebuild(b *testing.B) {
	oldBasepath := Basepath
	Basepath = b.TempDir()
	b.Cleanup(func() { Basepath = oldBasepath })
	dbName := fmt.Sprintf("maintenance-budget-bench-%d", time.Now().UnixNano())
	CreateDatabase(dbName, false)
	b.Cleanup(func() { databases.Remove(dbName) })
	tbl, _ := CreateTable(dbName, "items", Memory, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("tenant_id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)
	rows := make([][]scm.Scmer, 10_000)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i % 128)), scm.NewString(fmt.Sprintf("payload-%d", i))}
	}
	tbl.Insert([]string{"id", "tenant_id", "payload"}, rows, nil, scm.NewNil(), false, nil)

	previousCapacity := GlobalMaintenanceRAMBudget.Stat().Capacity
	GlobalMaintenanceRAMBudget.SetCapacity(64 << 20)
	b.Cleanup(func() { GlobalMaintenanceRAMBudget.SetCapacity(previousCapacity) })
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result := tbl.schema.rebuild(true, false, false, tbl)
		if len(result.errors) != 0 {
			b.Fatalf("rebuild errors: %v", result.errors)
		}
	}
}
