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
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

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
		columns:      map[string]ColumnStorage{"base": &StorageConst{value: scm.NewInt(1), count: 1}},
		deltaColumns: make(map[string]int),
		srState:      SHARED,
	}
	before := shard.ComputeSize()
	shard.columns["cached"] = &StorageConst{value: scm.NewString("separately-owned"), count: 1}
	after := shard.ComputeSize()
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
	before := shard.ComputeSize()
	strings.dictionary = "materialized-dictionary"
	if after := shard.ComputeSize(); after != before {
		t.Fatalf("shard size grew with separately owned dictionary: before=%d after=%d", before, after)
	}
}

func TestShardMemoryExcludesSeparatelyOwnedIndex(t *testing.T) {
	shard := &storageShard{
		columns:      make(map[string]ColumnStorage),
		deltaColumns: make(map[string]int),
		srState:      SHARED,
	}
	before := shard.ComputeSize()
	idx := &StorageIndex{}
	idx.baseState.mainIndexes.initValuesUInt32(1024, 0, 1023)
	shard.Indexes = []*StorageIndex{idx}
	after := shard.ComputeSize()
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

func TestScmerCustomHandleDoesNotClaimOwnedPayload(t *testing.T) {
	// Custom handles are non-owning pointers. Their payload belongs to a table,
	// RecSet, or another explicitly accounted owner and must not be traversed by
	// generic AST/cache accounting.
	tbl := new(table)
	if got := scm.ComputeSize(NewTableScmer(tbl)); got != 16 {
		t.Fatalf("custom handle size = %d, want one Scmer slot", got)
	}
}

func TestComputeProxySizeIncludesSessionVariants(t *testing.T) {
	proxy := &StorageComputeProxy{
		delta:    make(map[uint32]scm.Scmer),
		variants: make(map[string]*storageComputeVariant),
	}
	before := proxy.ComputeSize()
	variant := newStorageComputeVariant(1)
	variant.delta[0] = scm.NewString("variant-owned-value")
	proxy.variants["tenant"] = variant
	after := proxy.ComputeSize()
	if after <= before {
		t.Fatalf("session variant is absent from proxy ownership: before=%d after=%d", before, after)
	}
}
