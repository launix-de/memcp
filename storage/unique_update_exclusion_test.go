/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"github.com/launix-de/memcp/storage/mosi"
)

func TestUniqueProbeExclusionContinuesAfterItsExactPhysicalRecord(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_exact_exclusion")
	// Construct equal data before declaring any key: the kernel must still
	// report the genuinely different record after omitting one exact source.
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}, {scm.NewInt(7)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	access := exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(7)})
	first, found := shard.GetRecordidForUnique(access, nil, true, nil)
	if !found {
		t.Fatal("unexcluded positive control has no equal row")
	}
	exclude := &recMapTarget{shard: shard, recid: first}
	if other, found := shard.GetRecordidForUnique(access, nil, true, exclude); !found || other == first {
		t.Fatal("skipping the first candidate hid a genuinely different equal record")
	}
	// A pointer/record pair from a different generation is not a wildcard,
	// even if a descriptive UUID was copied into that private test handle.
	another := &storageShard{t: tbl, uuid: shard.uuid}
	if got, found := shard.GetRecordidForUnique(access, nil, true, &recMapTarget{shard: another, recid: first}); !found || got != first {
		t.Fatal("another physical generation suppressed this source's record")
	}
	requirePanic(t, func() {
		shard.GetRecordidForUnique(access, nil, true, &recMapTarget{shard: shard, recid: ^uint32(0)})
	})
}

func TestRepeatedUniqueUpdatesPreserveOriginalRowsAndRealCollisions(t *testing.T) {
	for _, mode := range []string{"nil", "cs", "acid"} {
		for _, rebuilt := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/rebuilt=%t", mode, rebuilt), func(t *testing.T) {
				tbl := setupScanParallelTestTable(t, "t_unique_repeat_original")
				tbl.CreateColumn("value", "INT", nil, nil)
				tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "UQ_value", Cols: []string{"value"}}}
				tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewInt(10)}, {scm.NewInt(2), scm.NewInt(20)}}, nil, scm.NewNil(), false, nil)
				if rebuilt {
					if result := RebuildTable(tbl, true, false, nil); result.err != nil {
						t.Fatal(result.err)
					}
				}
				shard := tbl.ActiveShards()[0]
				var tx *TxContext
				if mode != "nil" {
					kind := TxCursorStability
					if mode == "acid" {
						kind = TxACID
					}
					tx = NewTxContext(kind)
					defer func() {
						if tx.State == TxActive {
							tx.Rollback()
						}
					}()
				}
				locate := func(id int64, reader *TxContext) uint32 {
					t.Helper()
					recid, found := shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(id)}), reader, false, nil)
					if !found {
						t.Fatalf("original row %d disappeared", id)
					}
					return recid
				}
				value := func(id int64, reader *TxContext) int64 {
					recid := locate(id, reader)
					column := shard.ColumnReaderTx(reader, "value", false, ReadAuto)
					defer ReleaseColumnReader(column)
					return column.GetValue(recid).Int()
				}
				update := func(next int64) {
					t.Helper()
					recid := locate(1, tx)
					if !scm.ToBool(shard.UpdateFunction(recid, false, false, tx, nil)(scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(next)}))) {
						t.Fatal("original physical update did not execute")
					}
				}
				update(11)
				update(12) // unchanged PRIMARY must not collide with staged self
				if got := value(1, tx); got != 12 || value(2, tx) != 20 {
					t.Fatal("sequential replacements lost their complete original rows")
				}
				observer := int64(12)
				if mode == "acid" {
					observer = 10
				}
				if value(1, nil) != observer || value(2, nil) != 20 {
					t.Fatal("self exclusion changed ordinary observer visibility")
				}
				var failure any
				func() { defer func() { failure = recover() }(); update(20) }()
				if failure == nil || !strings.Contains(fmt.Sprint(failure), "Duplicate entry") || value(1, tx) != 12 || value(2, tx) != 20 {
					t.Fatalf("genuine other-row duplicate lost its original result/state: %v", failure)
				}
				if tx != nil {
					sp := tx.CreateSavepoint()
					update(13)
					tx.RollbackToSavepoint(sp)
					if value(1, tx) != 12 || value(2, tx) != 20 {
						t.Fatal("savepoint rewind lost the previous staged replacement")
					}
					update(14)
					tx.Rollback()
					if value(1, nil) != 10 || value(2, nil) != 20 || tbl.CountExact() != 2 {
						t.Fatal("full rollback did not restore both original rows")
					}
				} else {
					update(14)
					if value(1, nil) != 14 || value(2, nil) != 20 || tbl.CountExact() != 2 {
						t.Fatal("nil-TX replacement changed a different row")
					}
				}
			})
		}
	}
}

func TestReservedUniqueProbeExclusionHasExactNativeCoordinates(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_reserved_exclusion")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	tbl.Shards = append(tbl.Shards, NewShard(tbl))
	tbl.publishTopologyLocked()
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	for _, part := range tbl.ActiveShards() {
		cluster.declareShard(part, true)
	}
	columns, row := []string{"id"}, []scm.Scmer{scm.NewInt(7)}
	guard := tbl.guardUniqueWrites(columns, [][]scm.Scmer{row}, false, 0, nil, nil)
	defer guard.release()
	set := tbl.reserveUniqueDomain(columns, [][]scm.Scmer{row}, false, nil, nil, &guard)
	defer set.close(false)
	part := set.nativeShard(shard)
	if part == nil || set.local[shard.uuid.String()] != part {
		t.Fatal("actual native reservation did not bind the expected group member")
	}
	old := []scm.Scmer{scm.NewString("probe"), scm.NewString(shard.uuid.String()), uniqueStringArray(columns), scm.NewSlice(row), uniqueStringArray(nil)}
	if value, err := set.dispatchUniqueCommand(context.Background(), old); err != nil || !value.Slice()[0].Bool() {
		t.Fatalf("old strict probe lost its ordinary positive result: %v", err)
	}
	exclusion := []scm.Scmer{scm.NewString(tbl.RuntimeIdentity), scm.NewString(shard.uuid.String()), scm.NewInt(0)}
	probe := append(append([]scm.Scmer(nil), old...), scm.NewSlice(exclusion))
	if value, err := set.dispatchUniqueCommand(context.Background(), probe); err != nil || value.Slice()[0].Bool() {
		t.Fatalf("exact reserved self exclusion did not omit its source: %v", err)
	}
	for _, bad := range []scm.Scmer{
		scm.NewNil(),
		scm.NewSlice([]scm.Scmer{scm.NewString(uuid.NewString()), exclusion[1], exclusion[2]}),
		scm.NewSlice([]scm.Scmer{exclusion[0], scm.NewString(uuid.NewString()), exclusion[2]}),
		scm.NewSlice([]scm.Scmer{exclusion[0], exclusion[1], scm.NewInt(-1)}),
		scm.NewSlice([]scm.Scmer{exclusion[0], exclusion[1], scm.NewInt(1 << 33)}),
	} {
		if _, err := set.dispatchUniqueCommand(context.Background(), append(append([]scm.Scmer(nil), old...), bad)); err == nil {
			t.Fatal("reserved probe accepted foreign or malformed exclusion authority")
		}
	}
	requirePanic(t, func() {
		set.dispatchUniqueCommand(context.Background(), append(append([]scm.Scmer(nil), old...), scm.NewSlice([]scm.Scmer{exclusion[0], exclusion[1], scm.NewInt(1)})))
	})
	set.close(false) // drain actual native writers before an independent observer
	if tbl.CountExact() != 1 {
		t.Fatal("read-only exclusion changed reserved table rows")
	}
}
