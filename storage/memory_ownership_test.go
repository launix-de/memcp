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
	"io"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

type memoryPublicationPersistence struct {
	PersistenceEngine
	manifestReached chan struct{}
	resume          chan struct{}
}

func (p *memoryPublicationPersistence) WriteColumn(shard, column string) io.WriteCloser {
	if column == blobManifestColumn {
		close(p.manifestReached)
		<-p.resume
	}
	return p.PersistenceEngine.WriteColumn(shard, column)
}

func TestRebuildMemoryTransferPrecedesCommitForwarding(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	CreateDatabase("trampublication", false)
	defer databases.Remove("trampublication")
	tbl, _ := CreateTable("trampublication", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.Shards[0]
	col := &column{}
	shard.mu.Lock()
	shard.tempColumnBytes = map[*column]int64{col: 128}
	shard.mu.Unlock()
	p := &memoryPublicationPersistence{tbl.schema.persistence, make(chan struct{}), make(chan struct{})}
	tbl.schema.persistence = p
	done := make(chan *storageShard, 1)
	go func() { done <- shard.rebuild(true) }()
	<-p.manifestReached
	// This is the source lock held by commit's syncNextVisibilityLocked.
	// Once nextReady is published, commit may wait for the successor lock;
	// rebuild must release that lock without ever reacquiring this one.
	shard.mu.Lock()
	if !shard.nextReady.Load() {
		t.Error("fixture did not reach commit-forwarding publication")
	}
	close(p.resume)
	select {
	case rebuilt := <-done:
		shard.mu.Unlock()
		rebuilt.mu.RLock()
		got := rebuilt.tempColumnBytes[col]
		rebuilt.mu.RUnlock()
		if got != 128 {
			t.Fatalf("successor lost temporary-column accounting: %d", got)
		}
	case <-time.After(time.Second):
		shard.mu.Unlock()
		<-done
		t.Fatal("rebuild reacquired source lock after enabling commit forwarding")
	}
}

func TestBlobSizeDoesNotWaitForReader(t *testing.T) {
	cache := &blobRAMCache{}
	want := cache.ComputeSize()
	cache.mu.Lock()
	done := make(chan uint, 1)
	go func() { done <- cache.ComputeSize() }()
	select {
	case got := <-done:
		cache.mu.Unlock()
		if got != want {
			t.Fatalf("fixed owner bytes changed: got %d, want %d", got, want)
		}
	case <-time.After(time.Second):
		cache.mu.Unlock()
		<-done
		t.Fatal("memory display waits for a blob reader that may wait for CacheManager")
	}
}

func TestPersistedInternalTableIsNotRegisteredAsTempKeytable(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")
	before := GlobalCache.Stat()
	blobs := db.ensureBlobTable()
	after := GlobalCache.Stat()
	defer func() {
		GlobalCache.Remove(blobs)
		for _, shard := range blobs.ActiveShards() {
			GlobalCache.Remove(shard)
		}
	}()

	if after.CountByType[TypeTempKeytable] != before.CountByType[TypeTempKeytable] {
		t.Fatalf("persisted internal table registered as temp keytable: before=%d after=%d",
			before.CountByType[TypeTempKeytable], after.CountByType[TypeTempKeytable])
	}
	if after.CountByType[TypeShard] != before.CountByType[TypeShard]+1 {
		t.Fatalf("persisted internal shard has no durable owner: before=%d after=%d",
			before.CountByType[TypeShard], after.CountByType[TypeShard])
	}
}

func TestShardMemoryExcludesSeparatelyOwnedTempColumn(t *testing.T) {
	table := &table{Columns: []*column{
		{Name: "base"},
		{Name: "cached", IsTemp: true},
	}}
	shard := &storageShard{
		t:            table,
		columns:      map[string]ColumnStorage{"base": &StorageConst{value: scm.NewInt(1), count: 1}, "cached": nil},
		deltaColumns: make(map[string]int),
		srState:      SHARED,
	}
	before := shard.exclusiveSize()
	shard.columns["cached"] = &StorageConst{value: scm.NewString("separately-owned"), count: 1}
	after := shard.exclusiveSize()
	if after != before {
		t.Fatalf("shard owns separately accounted temp-column bytes: before=%d after=%d", before, after)
	}
}

func TestShardMemoryExcludesMaterializedCompressedDictionary(t *testing.T) {
	strings := &StorageString{
		compressedDict: []byte("compressed"),
		compressed:     true,
	}
	shard := &storageShard{
		columns:      map[string]ColumnStorage{"value": strings},
		deltaColumns: make(map[string]int),
		srState:      SHARED,
	}
	before := shard.exclusiveSize()
	strings.dictionary = "materialized-dictionary"
	if after := shard.exclusiveSize(); after != before {
		t.Fatalf("shard size grew with separately owned dictionary: before=%d after=%d", before, after)
	}
}

func TestOwnedMemoryExcludesNestedMaterializedDictionaries(t *testing.T) {
	base := &StorageString{compressedDict: []byte("compressed"), compressed: true}
	blob := &OverlayBlob{Base: base}
	before := ownedColumnMemory(blob)
	base.dictionary = "materialized-under-blob"
	if after := ownedColumnMemory(blob); after != before {
		t.Fatalf("blob owner includes separately weighted nested dictionary: before=%d after=%d", before, after)
	}

	prefix := &StoragePrefix{values: StorageString{compressed: true, dictionary: "materialized-under-prefix"}}
	if got, want := materializedDictionaryMemory(prefix), uint(len(prefix.values.dictionary)); got != want {
		t.Fatalf("nested prefix dictionary ownership = %d, want %d", got, want)
	}

	proxy := &StorageComputeProxy{main: base, delta: make(map[uint32]scm.Scmer)}
	if got, want := materializedDictionaryMemory(proxy), uint(len(base.dictionary)); got != want {
		t.Fatalf("nested compute-proxy dictionary ownership = %d, want %d", got, want)
	}
}

func TestShardMemoryExcludesSeparatelyOwnedIndex(t *testing.T) {
	shard := &storageShard{
		columns:      make(map[string]ColumnStorage),
		deltaColumns: make(map[string]int),
		srState:      SHARED,
		Indexes:      make([]*StorageIndex, 1), // reserve the parent-owned pointer slot
	}
	before := shard.exclusiveSize()
	idx := &StorageIndex{}
	idx.baseState.mainIndexes.initValuesUInt32(1024, 0, 1023)
	shard.Indexes[0] = idx
	after := shard.exclusiveSize()
	if after != before {
		t.Fatalf("shard owns separately accounted index bytes: before=%d after=%d index=%d",
			before, after, idx.ComputeSize())
	}
}

func TestCacheManagerSetSizeIsAbsolute(t *testing.T) {
	manager := new(CacheManager)
	manager.Init(0, 0)
	defer manager.Stop()

	pointer := new(int)
	manager.AddItem(pointer, 10, TypeCacheEntry,
		func(any, *[numEvictableTypes]int64) bool { return true },
		func(any) time.Time { return time.Time{} }, nil)
	manager.SetSize(pointer, 25)
	manager.SetSize(pointer, 25)
	stat := manager.Stat()
	if stat.CurrentMemory != 25 {
		t.Fatalf("absolute size update accumulated: got %d want 25", stat.CurrentMemory)
	}
}

func TestDisjointOwnershipDoesNotChangeEvictionWeights(t *testing.T) {
	want := [numEvictableTypes]int64{20, 1, 20, 2, 20, 20}
	if evictableWeights != want {
		t.Fatalf("eviction weights changed with accounting ownership: got %v want %v", evictableWeights, want)
	}
}

func TestScmerCustomHandleDoesNotClaimOwnedPayload(t *testing.T) {
	// Custom handles are non-owning pointers. Their payload belongs to a table,
	// RecSet, or another explicitly accounted owner and must not be traversed by
	// generic AST/cache accounting.
	tbl := new(table)
	if got := scm.ComputeSize(NewTableScmer(tbl)); got != 16 {
		t.Fatalf("custom handle size = %d, want one Scmer slot", got)
	}
}

func TestShardInclusiveSizeAndBusyChildEviction(t *testing.T) {
	defer setupGCTest(t)()
	shard := &storageShard{columns: make(map[string]ColumnStorage), srState: SHARED}
	index := &StorageIndex{}
	index.baseState.mainIndexes.initValuesUInt32(1024, 0, 1023)
	shard.Indexes = []*StorageIndex{index}
	if got, want := shard.ComputeSize(), shard.exclusiveSize()+index.ComputeSize(); got != want {
		t.Fatalf("inclusive shard size %d != base plus index %d", got, want)
	}
	// Registration goes through the manager; the failed TryLock below returns
	// before cleanup touches the manager's single-owner map.
	GlobalCache.AddItem(index, int64(index.ComputeSize()), TypeIndex, indexCleanup, indexLastUsed, indexGetScore)
	index.mu.Lock()
	// Cleanup must not deregister the index when its lock cannot be acquired.
	if shardCleanup(shard, nil) {
		t.Fatal("busy index must prevent full shard release")
	}
	index.mu.Unlock()
	if got := GlobalCache.Stat().CountByType[TypeIndex]; got < 1 {
		t.Fatal("busy index disappeared from memory ledger")
	}
	GlobalCache.Remove(index)
}

func TestShardFullOfferAndReleaseCountChildrenExactlyOnce(t *testing.T) {
	defer setupGCTest(t)()
	shard := &storageShard{columns: make(map[string]ColumnStorage), srState: SHARED}
	index := &StorageIndex{}
	index.baseState.mainIndexes.initValuesUInt32(1024, 0, 1023)
	shard.Indexes = []*StorageIndex{index}
	GlobalCache.AddItem(index, int64(index.ComputeSize()), TypeIndex, indexCleanup, indexLastUsed, indexGetScore)
	GlobalCache.AddItem(shard, int64(shard.exclusiveSize()), TypeShard, shardCleanup, shardLastUsed, nil)
	// Stop the worker before exercising its single-owner internals directly.
	GlobalCache.Stop()
	defer resumeOwnershipTestCache()
	parent := GlobalCache.itemMap[shard]
	offer := shard.evictionOffer(parent.size)
	want := parent.size + GlobalCache.itemMap[index].size
	if offer.partialBytes != 0 || offer.fullBytes != want {
		t.Fatalf("parent offer %+v, want full union %d", offer, want)
	}
	var byType [numEvictableTypes]int64
	freed := GlobalCache.applyEviction(evictionCandidate{item: parent, offer: offer}, evictFull, &byType)
	if freed != want || byType[TypeShard]+byType[TypeIndex] != want {
		t.Fatalf("release counted wrong: actual=%d types=%v want=%d", freed, byType, want)
	}
	if GlobalCache.itemMap[index] != nil || GlobalCache.itemMap[shard] != nil {
		t.Fatal("evicted owner remains registered")
	}
}

func TestChildCacheCleanupRetainsResidentTempColumnAccounting(t *testing.T) {
	defer setupGCTest(t)()
	col := &column{}
	shard := &storageShard{
		columns:         make(map[string]ColumnStorage),
		tempColumnBytes: map[*column]int64{col: 512},
	}
	GlobalCache.AddItem(col, 512, TypeTempColumn,
		func(any, *[numEvictableTypes]int64) bool { return true },
		func(any) time.Time { return time.Time{} }, nil)
	GlobalCache.Stop()
	defer resumeOwnershipTestCache()
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if !shard.evictChildrenLocked(nil) {
		t.Fatal("empty child-cache cleanup failed")
	}
	// Another shard can still veto a table eviction. No payload has been
	// detached here, so its separate registration must retain all its bytes.
	if got := GlobalCache.itemMap[col].size; got != 512 || shard.tempColumnBytes[col] != 512 {
		t.Fatalf("resident payload prematurely subtracted: registration=%d", got)
	}
	shard.releaseTempColumnBytesLocked(nil)
	if got := GlobalCache.itemMap[col].size; got != 0 {
		t.Fatalf("detached shard portion remains charged: %d", got)
	}
}

// Tests pause the single-owner worker to inspect its ledger without racing it.
// Resume with the same registrations so subsequent tests retain their owners.
func resumeOwnershipTestCache() {
	GlobalCache.opChan = make(chan cacheOp, 1024)
	GlobalCache.runDone = make(chan struct{})
	GlobalCache.stopped.Store(false)
	go GlobalCache.run()
}
