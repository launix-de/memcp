/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestInsertIdentityFrontendSelection(t *testing.T) {
	oldBasepath, oldShardSize := Basepath, Settings.ShardSize
	Basepath, Settings.ShardSize = t.TempDir(), 2
	defer func() { Basepath, Settings.ShardSize = oldBasepath, oldShardSize }()
	Init(scm.Globalenv)
	const database = "identity_frontends"
	CreateDatabase(database, false)
	defer DropDatabase(database, false)
	tbl, _ := CreateTable(database, "items", Memory, false)
	defer func() {
		// ShardSize=2 starts an overflow rebuild. Join its maintenance owner
		// before DROP and TempDir cleanup so a late schema save cannot recreate
		// the test directory or observe restored process-wide settings.
		tbl.maintenanceMu.Lock()
		tbl.maintenanceMu.Unlock()
	}()
	tbl.CreateColumn("id", "BIGINT", nil, []scm.Scmer{scm.NewString("auto_increment"), scm.NewBool(true)})
	tbl.CreateColumn("value", "INT", nil, nil)
	insert := scm.Globalenv.Vars[scm.Symbol("insert")]
	columns := scm.NewSlice([]scm.Scmer{scm.NewString("value")})
	rows := scm.NewSlice([]scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewInt(1)}), scm.NewSlice([]scm.Scmer{scm.NewInt(2)}), scm.NewSlice([]scm.Scmer{scm.NewInt(3)})})
	var observed []int64
	callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		observed = append(observed, args[0].Int())
		return scm.NewBool(true)
	})
	scm.Apply(insert, NewTableScmer(tbl), columns, rows, scm.NewSlice(nil), scm.NewNil(), scm.NewBool(false), callback)
	if len(observed) != 1 || observed[0] != 1 {
		t.Fatal("default frontend lost first identity semantics", observed)
	}
	observed = nil
	scm.Apply(insert, NewTableScmer(tbl), columns, rows, scm.NewSlice(nil), scm.NewNil(), scm.NewBool(false), callback, scm.NewNil(), scm.NewBool(true))
	if len(observed) != 2 || observed[0] != 5 || observed[1] != 6 {
		t.Fatal("last identity did not follow this invocation's chunk ranges", observed)
	}
	tbl.mu.Lock()
	atomic.StoreUint64(&tbl.Auto_increment, 9007199254740993)
	tbl.mu.Unlock()
	observed = nil
	scm.Apply(insert, NewTableScmer(tbl), columns, rows, scm.NewSlice(nil), scm.NewNil(), scm.NewBool(false), callback, scm.NewNil(), scm.NewBool(true))
	if observed[len(observed)-1] != 9007199254740996 {
		t.Fatal("identity passed through a lossy floating point value", observed)
	}
}

func TestInsertIdentityConcurrentReservationOwnership(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	const database = "identity_reservations"
	CreateDatabase(database, false)
	defer databases.Remove(database)
	tbl, _ := CreateTable(database, "items", Memory, false)
	tbl.CreateColumn("id", "BIGINT", nil, []scm.Scmer{scm.NewString("auto_increment"), scm.NewBool(true)})
	tbl.CreateColumn("value", "INT", nil, nil)
	var writers sync.WaitGroup
	var mu sync.Mutex
	seen := make(map[int64]bool)
	for i := 0; i < 8; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}, {scm.NewInt(3)}}, nil, scm.NewNil(), false, func(first, last int64) {
				if last-first != 2 {
					t.Error("reservation mixed concurrent invocations", first, last)
				}
				mu.Lock()
				defer mu.Unlock()
				for id := first; id <= last; id++ {
					if seen[id] {
						t.Error("reservation reused another invocation's identity", id)
					}
					seen[id] = true
				}
			})
		}()
	}
	writers.Wait()
	if len(seen) != 24 {
		t.Fatal("incomplete identity ranges", seen)
	}
}

func TestInsertIdentityDeclaredRangeRejectsBeforePublication(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	const database = "identity_declared_ranges"
	CreateDatabase(database, false)
	defer databases.Remove(database)
	for _, spec := range []struct {
		name string
		typ  string
		dims []int
		max  uint64
	}{
		{"tiny", "TINYINT", nil, 255},
		{"small", "SMALLINT", nil, 32767},
		{"int", "INT", nil, 2147483647},
		{"big", "BIGINT", nil, 9223372036854775807},
		{"decimal", "NUMERIC", []int{2, 0}, 99},
	} {
		t.Run(spec.name, func(t *testing.T) {
			tbl, _ := CreateTable(database, spec.name, Memory, false)
			tbl.CreateColumn("id", spec.typ, spec.dims, []scm.Scmer{scm.NewString("auto_increment"), scm.NewBool(true), scm.NewString("allocator_max"), scm.NewInt(int64(spec.max)), scm.NewString("null"), scm.NewBool(false)})
			tbl.CreateColumn("value", "INT", nil, nil)
			tbl.mu.Lock()
			atomic.StoreUint64(&tbl.Auto_increment, spec.max-1)
			tbl.mu.Unlock()
			called := false
			func() {
				defer func() {
					if recover() == nil {
						t.Error("out-of-range identity batch succeeded")
					}
				}()
				tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}}, nil, scm.NewNil(), false, func(first, last int64) { called = true })
			}()
			tbl.mu.Lock()
			counter := atomic.LoadUint64(&tbl.Auto_increment)
			tbl.mu.Unlock()
			if called || counter != spec.max-1 || tbl.CountEstimate() != 0 {
				t.Fatal("failed allocation published a counter, callback or row", counter, called)
			}
			var last int64
			tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(3)}}, nil, scm.NewNil(), false, func(first, generated int64) { last = generated })
			if last != int64(spec.max) || tbl.CountEstimate() != 1 {
				t.Fatal("valid boundary identity was not recovered after failure", last)
			}
		})
	}
}

func TestInsertIdentityEncodedScalarWALRecovery(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const database = "identity_exact_wal"
	CreateDatabase(database, false)
	defer databases.Remove(database)
	tbl, _ := CreateTable(database, "items", Safe, false)
	scm.RegisterExactPrimitives()
	tbl.CreateColumn("id", "ANY", nil, []scm.Scmer{
		scm.NewString("auto_increment"), scm.NewBool(true),
		scm.NewString("allocator_max"), scm.NewInt(math.MaxInt64),
		scm.NewString("allocator_value"), scm.EvalAll("allocator reader", `(lambda (value) (coefficient_to_integer value 0 false))`, &scm.Globalenv),
		scm.NewString("allocator_encode"), scm.EvalAll("allocator writer", `(lambda (value) (integer_to_coefficient value))`, &scm.Globalenv),
		scm.NewString("null"), scm.NewBool(false),
	})
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(10)}, {scm.NewInt(20)}}, nil, scm.NewNil(), false, nil)
	const explicit = int64(9007199254740993)
	negative := scm.IntegerToCoefficient(scm.NewInt(-5))
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{negative, scm.NewInt(5)}}, nil, scm.NewNil(), false, nil)
	id := scm.IntegerToCoefficient(scm.NewInt(explicit))
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{id, scm.NewInt(30)}}, nil, scm.NewNil(), false, nil)
	tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(40)}}, nil, scm.NewNil(), false, nil)
	for _, shard := range tbl.ActiveShards() {
		if shard.logfile != nil {
			shard.logfile.Close() // Retain WAL without rebuilding its delta rows.
		}
	}
	restored := reloadTableFromPersistence(t, database, tbl.schema.persistence)
	if restored.Count() != 5 || atomic.LoadUint64(&restored.Auto_increment) != uint64(explicit+1) {
		t.Fatal("decimal identity WAL lost rows or allocator precision", restored.Count(), atomic.LoadUint64(&restored.Auto_increment))
	}
	var generated int64
	restored.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(50)}}, nil, scm.NewNil(), false, func(first, last int64) { generated = last })
	if generated != explicit+2 || restored.Count() != 6 {
		t.Fatal("restored decimal allocator reused or rounded its next ID", generated)
	}
	if result := RebuildTable(restored, true, false); strings.Contains(result, "errors:") {
		t.Fatal("decimal identity rebuild failed", result)
	}
	for _, shard := range restored.ActiveShards() {
		if shard.logfile != nil {
			shard.logfile.Close()
		}
	}
	rebuilt := reloadTableFromPersistence(t, database, restored.schema.persistence)
	if rebuilt.Count() != 6 || atomic.LoadUint64(&rebuilt.Auto_increment) != uint64(explicit+2) {
		t.Fatal("rebuilt decimal identity lost rows or allocator precision", rebuilt.Count(), atomic.LoadUint64(&rebuilt.Auto_increment))
	}
	rebuilt.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(60)}}, nil, scm.NewNil(), false, func(first, last int64) { generated = last })
	if generated != explicit+3 || rebuilt.Count() != 7 {
		t.Fatal("rebuilt decimal allocator reused or rounded its next ID", generated)
	}
}

func TestInsertIdentityConcurrentSchemaSnapshots(t *testing.T) {
	oldBasepath, oldShardSize := Basepath, Settings.ShardSize
	Basepath, Settings.ShardSize = t.TempDir(), 2048
	defer func() { Basepath, Settings.ShardSize = oldBasepath, oldShardSize }()
	Init(scm.Globalenv)
	const database = "identity_schema_snapshots"
	CreateDatabase(database, false)
	defer DropDatabase(database, false)
	tbl, _ := CreateTable(database, "items", Memory, false)
	tbl.CreateColumn("id", "BIGINT", nil, []scm.Scmer{scm.NewString("auto_increment"), scm.NewBool(true)})
	tbl.CreateColumn("value", "INT", nil, nil)
	const initial, batches = uint64(9007199254740993), 128
	atomic.StoreUint64(&tbl.Auto_increment, initial)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(5)
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < batches; i++ {
			tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(int64(i))}}, nil, scm.NewNil(), false, nil)
		}
	}()
	for i := 0; i < 4; i++ {
		go func() {
			defer workers.Done()
			<-start
			previous := initial
			for j := 0; j < batches; j++ {
				// Match the schema publisher's metadata ownership without taking
				// table mu while concurrent allocations reserve their own ranges.
				tbl.schema.schemalock.RLock()
				data, err := json.Marshal(tbl)
				tbl.schema.schemalock.RUnlock()
				if err != nil {
					t.Error(err)
					return
				}
				var snapshot struct {
					Counter uint64 `json:"Auto_increment"`
				}
				if err := json.Unmarshal(data, &snapshot); err != nil {
					t.Error(err)
					return
				}
				if snapshot.Counter < previous || snapshot.Counter > initial+batches {
					t.Error("schema snapshot lost or tore the allocator high-water value", snapshot.Counter, previous)
					return
				}
				previous = snapshot.Counter
			}
		}()
	}
	close(start)
	workers.Wait()
	if counter := atomic.LoadUint64(&tbl.Auto_increment); counter != initial+batches || tbl.Count() != batches {
		t.Fatal("concurrent schema snapshots changed identity reservations", counter, tbl.Count())
	}
}

func TestInsertIdentityLegacySchemaCounter(t *testing.T) {
	// The persisted integer field remains compatible even above signed and
	// floating-point ranges. JSON decoding is private before table publication.
	var tbl table
	if err := json.Unmarshal([]byte(`{"Name":"items","Auto_increment":18446744073709551615}`), &tbl); err != nil {
		t.Fatal(err)
	}
	if counter := atomic.LoadUint64(&tbl.Auto_increment); counter != ^uint64(0) {
		t.Fatal("legacy schema counter lost uint64 precision", counter)
	}
	data, err := json.Marshal(&tbl)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Counter uint64 `json:"Auto_increment"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil || snapshot.Counter != ^uint64(0) {
		t.Fatal("schema counter encoding changed", string(data), err)
	}
}

func TestInsertIdentityReseedPolicy(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	const database = "identity_reseed_policy"
	CreateDatabase(database, false)
	defer DropDatabase(database, false)
	alter := scm.Globalenv.Vars[scm.Symbol("altercolumn")]
	for _, strict := range []bool{false, true} {
		name := "generic"
		attrs := []scm.Scmer{scm.NewString("auto_increment"), scm.NewBool(true)}
		if strict {
			name = "declared"
			attrs = append(attrs, scm.NewString("allocator_max"), scm.NewInt(255))
		}
		tbl, _ := CreateTable(database, name, Memory, false)
		tbl.CreateColumn("id", "TINYINT", nil, attrs)
		tbl.CreateColumn("value", "INT", nil, nil)
		reseed := func(value int64) {
			scm.Apply(alter, NewTableScmer(tbl), scm.NewString("id"), scm.NewString("auto_increment"), scm.NewInt(value))
		}
		reseed(10)
		if strict {
			requirePanic(t, func() { reseed(2) })
			requirePanic(t, func() { reseed(256) })
			if counter := atomic.LoadUint64(&tbl.Auto_increment); counter != 10 {
				t.Fatal("rejected reseed changed the declared identity counter", counter)
			}
		} else {
			reseed(2)
		}
		var last int64
		tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, func(first, generated int64) { last = generated })
		wanted := int64(3)
		if strict {
			wanted = 11
		}
		if last != wanted {
			t.Fatal("reseed policy changed subsequent identity allocation", strict, last, wanted)
		}
	}
}
