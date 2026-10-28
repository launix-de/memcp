/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
import "strings"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func TestHashPartitionRepresentationAndPersistence(t *testing.T) {
	text := "550e8400-e29b-41d4-a716-446655440000"
	packed := scm.ParseUUID(scm.NewString(text))
	d := shardDimension{Column: "id", NumPartitions: 17, Method: "hash-uuid-v1"}
	want := computeShardDimensionIndex(d, packed)
	if hash, ok := partitionHash(d.Method, packed); !ok || hash != 5032308396471380868 || want != 16 {
		t.Fatal("permanent hash-uuid-v1 mapping changed")
	}
	for _, value := range []scm.Scmer{scm.NewString(text), scm.NewString(strings.ToUpper(text)), scm.NewString(strings.ReplaceAll(text, "-", ""))} {
		if got := computeShardDimensionIndex(d, value); got != want {
			t.Fatalf("representation routed to %d, want %d", got, want)
		}
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var restored shardDimension
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if got := computeShardDimensionIndex(restored, packed); got != want {
		t.Fatal("persisted routing changed")
	}
	for _, data := range []string{
		`{"Column":"id","NumPartitions":2,"Method":"unknown"}`,
		`{"Column":"id","NumPartitions":0,"Method":"hash-int-v1"}`,
		`{"Column":"id","NumPartitions":2}`,
	} {
		if json.Unmarshal([]byte(data), &restored) == nil {
			t.Fatalf("accepted corrupt dimension %s", data)
		}
	}
	if err := json.Unmarshal([]byte(`{"Column":"id","NumPartitions":1}`), &restored); err != nil {
		t.Fatalf("legacy range: %v", err)
	}
}

func TestHashPartitionAutoIncrementRoutesGeneratedRows(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "thashauto", 0)
	tbl.Columns[0].AutoIncrement = true
	tbl.Columns[0].UpdateSanitizer()
	tbl.publishShowColumnsSnapshot(true)
	if !tbl.beginManualRepartition() {
		t.Fatal("maintenance not claimed")
	}
	d := shardDimension{Column: "id", NumPartitions: 8, Method: "hash-int-v1"}
	tbl.repartition([]shardDimension{d})
	calls, first := 0, int64(0)
	rows := make([][]scm.Scmer, 128)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewString("payload")}
	}
	tbl.Insert([]string{"payload"}, rows, nil, scm.NewNil(), false, func(id int64) { calls++; first = id })
	if calls != 1 || first != 1 {
		t.Fatalf("ID callback calls=%d first=%d", calls, first)
	}
	seen := make(map[int64]bool)
	used := 0
	for bucket, shard := range tbl.ActiveShards() {
		shard.mu.RLock()
		if len(shard.inserts) > 0 {
			used++
		}
		idx := shard.deltaColumns["id"]
		for _, row := range shard.inserts {
			id := row[idx]
			if id.Int() < 1 || id.Int() > 128 || seen[id.Int()] || computeShardDimensionIndex(d, id) != bucket {
				t.Fatalf("invalid or misrouted ID %v in bucket %d", id, bucket)
			}
			seen[id.Int()] = true
		}
		shard.mu.RUnlock()
	}
	if len(seen) != 128 || used < 2 {
		t.Fatalf("generated IDs=%d occupied buckets=%d", len(seen), used)
	}
}

func TestHashPartitionPruning(t *testing.T) {
	d := shardDimension{Column: "id", NumPartitions: 8, Method: "hash-int-v1"}
	shards := make([]*storageShard, 8)
	for i := range shards {
		shards[i] = &storageShard{}
	}
	for _, tc := range []struct {
		name     string
		boundary analyzedBoundary
		single   bool
	}{
		{"integer", analyzedBoundary{col: "id", matcher: EqualMatcher, lower: scm.NewInt(123), upper: scm.NewInt(123), lowerInclusive: true, upperInclusive: true}, true},
		{"null", analyzedBoundary{col: "id", matcher: EqualMatcher, lower: scm.NewNil(), upper: scm.NewNil(), lowerInclusive: true, upperInclusive: true}, true},
		{"range", analyzedBoundary{col: "id", matcher: RangeMatcher, lower: scm.NewInt(1), upper: scm.NewInt(9), lowerInclusive: true, upperInclusive: true}, false},
		{"coercion", analyzedBoundary{col: "id", matcher: EqualMatcher, lower: scm.NewFloat(123), upper: scm.NewFloat(123), lowerInclusive: true, upperInclusive: true}, false},
		{"in", analyzedBoundary{col: "id", matcher: InMatcher, lower: scm.NewSlice([]scm.Scmer{scm.NewInt(123)})}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access := runtimeScanAccess(analyzedBoundaries{tc.boundary})
			got := collectRelevantShards([]shardDimension{d}, access, shards)
			if tc.single {
				bucket := computeShardDimensionIndex(d, tc.boundary.lower)
				if len(got) != 1 || got[0] != shards[bucket] {
					t.Fatal("equality did not select its hash bucket")
				}
			} else if len(got) != len(shards) {
				t.Fatal("non-equality incorrectly pruned hash buckets")
			}
			_, single := singleRelevantShard([]shardDimension{d}, access, shards)
			if single != tc.single {
				t.Fatal("count and collection pruning disagree")
			}
		})
	}
}

func TestHashPartitionMixedRangePruning(t *testing.T) {
	dimensions := []shardDimension{
		{Column: "id", NumPartitions: 8, Method: "hash-int-v1"},
		{Column: "day", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}},
	}
	shards := make([]*storageShard, 16)
	for i := range shards {
		shards[i] = &storageShard{}
	}
	point := analyzedBoundary{col: "id", matcher: EqualMatcher, lower: scm.NewInt(123), upper: scm.NewInt(123), lowerInclusive: true, upperInclusive: true}
	day := analyzedBoundary{col: "day", matcher: RangeMatcher, lower: scm.NewInt(11), lowerInclusive: true}
	got := collectRelevantShards(dimensions, runtimeScanAccess(analyzedBoundaries{day}), shards)
	if len(got) != 8 {
		t.Fatalf("range dimension selected %d shards, want 8", len(got))
	}
	for i, shard := range got {
		if shard != shards[i*2+1] {
			t.Fatal("hash dimension corrupted independent range proof")
		}
	}
	got = collectRelevantShards(dimensions, runtimeScanAccess(analyzedBoundaries{point, day}), shards)
	want := computeShardDimensionIndex(dimensions[0], point.lower)*2 + 1
	if len(got) != 1 || got[0] != shards[want] {
		t.Fatal("mixed point/range routing disagrees with insertion")
	}
}

func TestRepartitionChoosesMethodPerColumn(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "thashchoice", 64)
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	id, payload := tbl.Columns[0], tbl.Columns[1]
	id.partitioningTraffic.Store(100)
	id.partitioningRangeTraffic.Store(0)
	payload.partitioningTraffic.Store(100)
	payload.partitioningRangeTraffic.Store(100)
	if d := tbl.preferredShardDimension(id.Name, 8); d.Method != "hash-int-v1" || d.NumPartitions != 8 {
		t.Fatalf("point column: %+v", d)
	}
	if d := tbl.preferredShardDimension(payload.Name, 8); d.Method != "" || d.NumPartitions < 2 {
		t.Fatalf("range column: %+v", d)
	}
	candidates, change := tbl.proposerepartition(Settings.ShardSize * 16)
	if !change || len(candidates) != 2 {
		t.Fatalf("rebuild proposal: change=%v dimensions=%+v", change, candidates)
	}
	for _, d := range candidates {
		if d.Column == id.Name && d.Method != "hash-int-v1" || d.Column == payload.Name && d.Method != "" {
			t.Fatalf("rebuild did not decide independently per column: %+v", candidates)
		}
	}
	id.Typ = "UUID"
	if d := tbl.preferredShardDimension(id.Name, 8); d.Method != "hash-uuid-v1" {
		t.Fatalf("UUID point column: %+v", d)
	}
	id.Typ = "TEXT"
	if d := tbl.preferredShardDimension(id.Name, 8); d.Method != "" {
		t.Fatalf("unsupported domain selected hash: %+v", d)
	}
}

func TestHashRepartitionConcurrentDeleteSurvivesReload(t *testing.T) {
	const rows = 256
	tbl, persistence := createDurabilityTestTable(t, "thashdeletereload", rows)
	blocking := &blockingColumnWritePersistence{PersistenceEngine: persistence, entered: make(chan struct{}), release: make(chan struct{})}
	tbl.schema.persistence = blocking
	if !tbl.beginManualRepartition() {
		t.Fatal("maintenance not claimed")
	}
	d := shardDimension{Column: "id", NumPartitions: 8, Method: "hash-int-v1"}
	done := make(chan struct{})
	go func() { tbl.repartition([]shardDimension{d}); close(done) }()
	<-blocking.entered
	waitForRepartitionDualWrite(t, tbl)
	old := tbl.repartitionSources.Load().shards[0]
	if !scm.ToBool(old.UpdateFunction(0, false, false, nil)()) {
		t.Fatal("delete failed")
	}
	close(blocking.release)
	<-done
	reloaded := reloadTableFromPersistence(t, "thashdeletereload", persistence)
	if restored := reloaded.preferredShardDimension("id", 8); restored.Method != d.Method {
		t.Fatal("unobserved rebuild changed persisted hash method")
	}
	if got := reloaded.Count(); got != rows-1 {
		t.Fatalf("reloaded count %d, want %d", got, rows-1)
	}
	seen := make(map[int64]bool)
	for bucket, shard := range reloaded.ActiveShards() {
		access := shard.GetRead(nil)
		reader := shard.getColumnStorageOrPanic("id", false, nil)
		for row := uint32(0); row < shard.main_count; row++ {
			if shard.deletions.Get(uint(row)) {
				continue
			}
			v := reader.GetValue(row)
			if computeShardDimensionIndex(d, v) != bucket {
				t.Fatal("row in wrong bucket after reload")
			}
			if seen[v.Int()] {
				t.Fatal("duplicate row after reload")
			}
			seen[v.Int()] = true
		}
		access.Release()
	}
	if len(seen) != rows-1 {
		t.Fatalf("only %d distinct rows survived", len(seen))
	}
}
