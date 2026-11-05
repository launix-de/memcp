/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "os"
import "fmt"
import "math"
import "sync"
import "time"
import "unsafe"
import "os/exec"
import "runtime"
import "strconv"
import "strings"
import "testing"
import "sync/atomic"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func TestInsertRoutesAroundBusyFreeShard(t *testing.T) {
	oldCPUs := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(oldCPUs)
	tbl := setupScanParallelTestTable(t, "tinsertbusyfree")
	shard := tbl.ActiveShards()[0]
	shard.mu.Lock()
	locked := true
	firstDone := make(chan any, 1)
	secondDone := make(chan any, 1)
	insert := func(id int64, done chan<- any) {
		defer func() { done <- recover() }()
		tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(id)}}, nil, scm.NewNil(), false, nil)
	}
	go insert(1, firstDone)
	defer func() {
		if locked {
			shard.mu.Unlock()
		}
		if caught := <-firstDone; caught != nil {
			t.Errorf("first INSERT: %v", caught)
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for shard.pendingInsertRows.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("first writer never reserved its batch")
		}
		runtime.Gosched()
	}
	go insert(2, secondDone)
	select {
	case caught := <-secondDone:
		if caught != nil {
			t.Fatalf("second INSERT: %v", caught)
		}
	case <-time.After(3 * time.Second):
		shard.mu.Unlock()
		locked = false
		<-secondDone
		t.Fatal("independent INSERT waited for the busy shard")
	}
	if len(tbl.ActiveShards()) != 2 {
		t.Fatalf("expected two writer lanes, got %d", len(tbl.ActiveShards()))
	}
	for _, s := range tbl.ActiveShards() {
		if s != shard && s.pendingInsertRows.Load() != 0 {
			t.Fatal("completed writer retained its reservation")
		}
	}
}

func TestInsertUniqueChecksAllowIndependentKeys(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinsertparallelkeys")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	first := [][]scm.Scmer{{scm.NewInt(1)}}
	firstMask := tbl.guardUniqueWrites([]string{"id"}, first, false, 0, nil).mask
	var second [][]scm.Scmer
	for i := int64(101); i < 1000; i++ {
		rows := [][]scm.Scmer{{scm.NewInt(i)}}
		if !tbl.guardUniqueWrites([]string{"id"}, rows, false, 0, nil).mask.intersects(firstMask) {
			second = rows
			break
		}
	}
	if second == nil {
		t.Fatal("could not find an independent UNIQUE stripe")
	}
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		tbl.ProcessUniqueCollision([]string{"id"}, first, false, func(_ [][]scm.Scmer, _ *uniqueWriteGuard) { close(entered); <-resume }, nil, nil, 0, nil, nil, nil)
	}()
	<-entered
	defer func() { close(resume); <-done }()
	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		tbl.ProcessUniqueCollision([]string{"id"}, second, false, func(_ [][]scm.Scmer, _ *uniqueWriteGuard) {}, nil, nil, 0, nil, nil, nil)
	}()
	select {
	case <-secondDone:
	case <-time.After(3 * time.Second):
		t.Fatal("independent UNIQUE key waited for a table-wide guard")
	}
}

func TestInsertConcurrentSameUniqueKey(t *testing.T) {
	for _, partitioned := range []bool{false, true} {
		t.Run(fmt.Sprint(partitioned), func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tinsertduplicate")
			tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
			if partitioned {
				tbl.ShardMode = ShardModePartition
				tbl.PDimensions = []shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}}}
				tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
				tbl.publishTopologyLocked()
			}
			var accepted atomic.Int64
			var workers sync.WaitGroup
			start := make(chan struct{})
			for i := 0; i < 16; i++ {
				workers.Add(1)
				go func() {
					defer workers.Done()
					defer func() {
						if caught := recover(); caught != nil && !strings.Contains(fmt.Sprint(caught), "Duplicate entry") {
							t.Errorf("INSERT: %v", caught)
						}
					}()
					<-start
					tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil)
					accepted.Add(1)
				}()
			}
			close(start)
			workers.Wait()
			if accepted.Load() != 1 {
				t.Fatalf("accepted %d equal keys", accepted.Load())
			}
			if tbl.CountExact() != 1 {
				t.Fatal("duplicate row became visible")
			}
			for _, shard := range tbl.ActiveShards() {
				if shard.pendingInsertRows.Load() != 0 {
					t.Fatal("failed writer retained its reservation")
				}
			}
		})
	}
}

// Exercise logical UNIQUE identity independently of its physical partition.
// An extra partition dimension may place equal keys in different shards.
func setupUniqueContentionTable(t *testing.T, layout string) *table {
	t.Helper()
	layout = strings.TrimSuffix(layout, "-main")
	tbl := setupScanParallelTestTable(t, "tuniquecontention")
	tbl.CreateColumn("bucket", "INT", nil, nil)
	tbl.CreateColumn("val", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	if layout != "free" {
		tbl.ShardMode = ShardModePartition
		switch layout {
		case "unique":
			tbl.PDimensions = []shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}}}
		case "extra":
			tbl.PDimensions = []shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}}, {Column: "bucket", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(1)}}}
		case "unrelated":
			tbl.PDimensions = []shardDimension{{Column: "bucket", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(1)}}}
		}
		count := 1
		for _, dimension := range tbl.PDimensions {
			count *= dimension.NumPartitions
		}
		tbl.PShards = make([]*storageShard, count)
		for i := range tbl.PShards {
			tbl.PShards[i] = NewShard(tbl)
		}
		tbl.publishTopologyLocked()
	}
	return tbl
}

func TestInsertUniqueContention(t *testing.T) {
	previous := runtime.GOMAXPROCS(32)
	defer runtime.GOMAXPROCS(previous)
	for _, layout := range []string{"free", "unique", "extra", "unrelated"} {
		t.Run(layout, func(t *testing.T) {
			tbl := setupUniqueContentionTable(t, layout)
			const workers, keys = 64, 8
			var accepted [keys]atomic.Int64
			var collisions atomic.Int64
			start := make(chan struct{})
			var done sync.WaitGroup
			for worker := 0; worker < workers; worker++ {
				done.Add(1)
				go func(worker int) {
					defer done.Done()
					defer func() {
						if failure := recover(); failure != nil {
							t.Errorf("INSERT: %v", failure)
						}
					}()
					<-start
					for key := 0; key < keys; key++ {
						inserted := tbl.Insert([]string{"id", "bucket", "val"}, [][]scm.Scmer{{scm.NewInt(int64(key + 1)), scm.NewInt(int64(worker % 2)), scm.NewInt(int64(worker))}}, nil,
							scm.NewFunc(func(...scm.Scmer) scm.Scmer { collisions.Add(1); return scm.NewInt(0) }), false, nil)
						accepted[key].Add(int64(inserted))
					}
				}(worker)
			}
			close(start)
			done.Wait()
			for key := range accepted {
				if n := accepted[key].Load(); n != 1 {
					t.Errorf("key %d: %d winners, want 1", key+1, n)
				}
			}
			if n := collisions.Load(); n != (workers-1)*keys {
				t.Errorf("collisions=%d, want %d", n, (workers-1)*keys)
			}
			if n := tbl.CountExact(); n != keys {
				t.Errorf("rows=%d, want %d", n, keys)
			}
		})
	}
}

func TestInsertCollisionReadModifyWriteContention(t *testing.T) { testInsertCollisionRMW(t, false) }

func TestInsertCollisionReadModifyWriteBeforeTrigger(t *testing.T) { testInsertCollisionRMW(t, true) }

func testInsertCollisionRMW(t *testing.T, before bool) {
	previous := runtime.GOMAXPROCS(32)
	defer runtime.GOMAXPROCS(previous)
	for _, layout := range []string{"free", "unique", "extra", "unrelated", "free-main", "unique-main", "extra-main", "unrelated-main"} {
		t.Run(layout, func(t *testing.T) {
			tbl := setupUniqueContentionTable(t, layout)
			columns := []string{"id", "bucket", "val"}
			tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(0), scm.NewInt(0)}}, nil, scm.NewNil(), false, nil)
			if strings.HasSuffix(layout, "-main") {
				if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
					t.Fatal(result)
				}
				var mainRows uint32
				for _, shard := range tbl.ActiveShards() {
					mainRows += shard.main_count
				}
				if mainRows == 0 {
					t.Fatal("fixture did not produce a main row")
				}
			}
			var beforeCalls atomic.Int64
			if before {
				get, set := scm.Globalenv.Vars["get_assoc"], scm.Globalenv.Vars["set_assoc"]
				tbl.AddTrigger(TriggerDescription{Name: "increment_before_update", Timing: BeforeUpdate, Func: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
					if !args[3].IsNil() {
						panic("native BEFORE acquired a synthetic transaction")
					}
					beforeCalls.Add(1)
					old := scm.Apply(get, args[0], scm.NewString("val"))
					return scm.Apply(set, args[1], scm.NewString("val"), scm.NewInt(old.Int()+1))
				})})
			}
			const workers, repetitions = 64, 16
			var calls, applied atomic.Int64
			callback := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				calls.Add(1)
				old := a[0].Int()
				// Widen the read/write interleaving without relying on a timer.
				runtime.Gosched()
				result := scm.Apply(a[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(old + 1)}))
				if result.Bool() {
					applied.Add(1)
				}
				return result
			})
			start := make(chan struct{})
			var done sync.WaitGroup
			for worker := 0; worker < workers; worker++ {
				done.Add(1)
				go func(worker int) {
					defer done.Done()
					defer func() {
						if failure := recover(); failure != nil {
							t.Errorf("UPSERT: %v", failure)
						}
					}()
					<-start
					for i := 0; i < repetitions; i++ {
						tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(int64(worker % 2)), scm.NewInt(0)}}, []string{"val", "$update"}, callback, false, nil)
					}
				}(worker)
			}
			close(start)
			done.Wait()
			var total int64
			for _, shard := range tbl.ActiveShards() {
				access := shard.GetRead(nil, false)
				reader := access.shard.ColumnReaderTx(nil, "val", false, ReadAuto)
				for id := uint32(0); id < access.shard.main_count+uint32(len(access.shard.deltaRows())); id++ {
					if !access.shard.deletions.Get(uint(id)) {
						total += reader.GetValue(id).Int()
					}
				}
				access.Release()
			}
			const expected = workers * repetitions
			if before && beforeCalls.Load() != expected {
				t.Fatalf("BEFORE executions=%d, want %d", beforeCalls.Load(), expected)
			}
			if calls.Load() != expected || applied.Load() != expected || total != expected || tbl.CountExact() != 1 {
				t.Fatalf("callbacks=%d updates=%d value=%d rows=%d; want %d increments on one row", calls.Load(), applied.Load(), total, tbl.CountExact(), expected)
			}
		})
	}
}

func TestInsertCollisionDifferentUniqueKeysShareRow(t *testing.T) {
	previous := runtime.GOMAXPROCS(32)
	defer runtime.GOMAXPROCS(previous)
	tbl := setupScanParallelTestTable(t, "tuniquecollisionaliases")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.CreateColumn("code", "TEXT", nil, nil)
	tbl.CreateColumn("val", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}, {Id: "code", Cols: []string{"code"}}}
	columns := []string{"id", "name", "code", "val"}
	tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("same-name"), scm.NewString("same-code"), scm.NewInt(0)}}, nil, scm.NewNil(), false, nil)
	const workers = 64
	start := make(chan struct{})
	var done sync.WaitGroup
	var applied atomic.Int64
	for worker := 0; worker < workers; worker++ {
		done.Add(1)
		go func(worker int) {
			defer done.Done()
			defer func() {
				if failure := recover(); failure != nil {
					t.Errorf("UPSERT: %v", failure)
				}
			}()
			<-start
			name, code := fmt.Sprint("name-", worker), fmt.Sprint("code-", worker)
			if worker%2 == 0 {
				name = "same-name"
			} else {
				code = "same-code"
			}
			tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(int64(worker + 2)), scm.NewString(name), scm.NewString(code), scm.NewInt(0)}}, []string{"val", "$update"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				runtime.Gosched()
				result := scm.Apply(args[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(args[0].Int() + 1)}))
				if result.Bool() {
					applied.Add(1)
				}
				return result
			}), false, nil)
		}(worker)
	}
	close(start)
	done.Wait()
	shard := tbl.ActiveShards()[0]
	reader := shard.ColumnReaderTx(nil, "val", false, ReadAuto)
	var total int64
	for id := uint32(0); id < shard.main_count+uint32(len(shard.deltaRows())); id++ {
		if !shard.deletions.Get(uint(id)) {
			total += reader.GetValue(id).Int()
		}
	}
	if total != workers || applied.Load() != workers || tbl.CountExact() != 1 {
		t.Fatalf("value=%d applied=%d rows=%d; want %d increments on one row", total, applied.Load(), tbl.CountExact(), workers)
	}
}

func TestInsertCollisionMutationPanicReleasesKeys(t *testing.T) {
	tbl := setupUniqueContentionTable(t, "free")
	columns := []string{"id", "bucket", "val"}
	row := [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(0), scm.NewInt(0)}}
	tbl.Insert(columns, row, nil, scm.NewNil(), false, nil)
	var caught any
	func() {
		defer func() { caught = recover() }()
		tbl.Insert(columns, row, []string{"val", "$update"}, scm.NewFunc(func(...scm.Scmer) scm.Scmer { panic("collision mutation failed") }), false, nil)
	}()
	if caught != "collision mutation failed" {
		t.Fatalf("panic=%v", caught)
	}
	assertUniqueWriteLocksReleased(t, tbl)
	if got := tbl.Insert(columns, row, []string{"val", "$update"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.Apply(a[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(a[0].Int() + 1)}))
	}), false, nil); got != 2 {
		t.Fatalf("subsequent update=%d", got)
	}
}

func TestInsertCollisionKeyChangeConflictRestoresRow(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquekeychangeconflict")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	columns := []string{"id", "name"}
	row := [][]scm.Scmer{{scm.NewInt(7), scm.NewString("original")}}
	tbl.Insert(columns, row, nil, scm.NewNil(), false, nil)
	original := tbl.guardUniqueWrites(columns, row, false, 0, nil)
	var name string
	var extra uniqueWriteGuard
	for i := 0; i < 1000; i++ {
		name = fmt.Sprint("replacement-", i)
		needed := tbl.guardUniqueWrites(columns, [][]scm.Scmer{{scm.NewInt(7), scm.NewString(name)}}, false, 0, nil)
		if !original.mask.contains(needed.mask) {
			var mask uniqueWriteMask
			for word := 0; word < len(needed.mask.words)+len(needed.mask.extra); word++ {
				mask.addWord(word, needed.mask.word(word)&^original.mask.word(word))
			}
			extra = uniqueWriteGuard{locks: original.locks, mask: mask}
			break
		}
	}
	if extra.mask.empty() {
		t.Fatal("no independent key stripe")
	}
	extra.lock()
	defer extra.unlock()
	calls := 0
	callback := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		calls++
		return scm.Apply(a[0], scm.NewSlice([]scm.Scmer{scm.NewString("name"), scm.NewString(name)}))
	})
	var caught any
	func() {
		defer func() { caught = recover() }()
		tbl.Insert(columns, row, []string{"$update"}, callback, false, nil)
	}()
	if caught == nil || !strings.Contains(fmt.Sprint(caught), "retry transaction") || calls != 1 {
		t.Fatalf("failure=%v callbacks=%d", caught, calls)
	}
	extra.unlock()
	assertUniqueWriteLocksReleased(t, tbl)
	if tbl.CountExact() != 1 {
		t.Fatal("failed key change hid the original row")
	}
	if got := tbl.Insert(columns, row, []string{"$update"}, callback, false, nil); got != 2 || calls != 2 || tbl.CountExact() != 1 {
		t.Fatalf("retry affected=%d callbacks=%d rows=%d", got, calls, tbl.CountExact())
	}
}

func TestInsertCollisionIndependentKeyProgress(t *testing.T) {
	tbl := setupUniqueContentionTable(t, "free")
	columns := []string{"id", "bucket", "val"}
	firstRow := [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(0), scm.NewInt(0)}}
	firstMask := tbl.guardUniqueWrites(columns, firstRow, false, 0, nil).mask
	var secondRow [][]scm.Scmer
	for id := int64(8); id < 1000; id++ {
		row := [][]scm.Scmer{{scm.NewInt(id), scm.NewInt(0), scm.NewInt(0)}}
		if !tbl.guardUniqueWrites(columns, row, false, 0, nil).mask.intersects(firstMask) {
			secondRow = row
			break
		}
	}
	if secondRow == nil {
		t.Fatal("no independent key stripe")
	}
	tbl.Insert(columns, append(firstRow, secondRow...), nil, scm.NewNil(), false, nil)
	entered, resume, firstDone := make(chan struct{}), make(chan struct{}), make(chan any, 1)
	release := sync.OnceFunc(func() { close(resume) })
	go func() {
		defer func() { firstDone <- recover() }()
		tbl.Insert(columns, firstRow, []string{"val", "$update"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			close(entered)
			<-resume
			return scm.Apply(a[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(a[0].Int() + 1)}))
		}), false, nil)
	}()
	<-entered
	defer func() {
		release()
		if failure := <-firstDone; failure != nil {
			t.Errorf("first callback: %v", failure)
		}
	}()
	secondDone := make(chan any, 1)
	go func() {
		defer func() { secondDone <- recover() }()
		tbl.Insert(columns, secondRow, []string{"val", "$update"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			return scm.Apply(a[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(a[0].Int() + 1)}))
		}), false, nil)
	}()
	select {
	case failure := <-secondDone:
		if failure != nil {
			t.Fatal(failure)
		}
	case <-time.After(3 * time.Second):
		release()
		<-secondDone
		t.Fatal("independent key waited for blocked collision callback")
	}
}

func TestInsertCollisionCallbackRechecksEarlierKeys(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinsertcallbackrecheck")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	columns := []string{"id", "name"}
	tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("taken")}}, nil, scm.NewNil(), false, nil)
	var collisions []string
	tbl.ProcessUniqueCollision(columns, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("taken")}, {scm.NewInt(3), scm.NewString("free")}}, false,
		func(rows [][]scm.Scmer, _ *uniqueWriteGuard) { t.Fatal("accepted key added by collision callback") }, nil,
		func(key string, _ []scm.Scmer) {
			collisions = append(collisions, key)
			if key == "name" {
				tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(3), scm.NewString("other")}}, nil, scm.NewNil(), false, nil)
			}
		}, 0, nil, nil, nil)
	if fmt.Sprint(collisions) != "[name PRIMARY]" {
		t.Fatalf("collisions=%v", collisions)
	}
	assertUniqueWriteLocksReleased(t, tbl)
}

func TestInsertUniqueGuardCoversCoercingEquality(t *testing.T) {
	values := []scm.Scmer{scm.NewNil(), scm.NewBool(false), scm.NewBool(true), scm.NewInt(0), scm.NewInt(1), scm.NewInt(9), scm.NewInt(1000), scm.NewInt(9007199254740993), scm.NewFloat(9.5), scm.NewFloat(1), scm.NewFloat(0.1), scm.NewDate(1700000000), scm.NewString("9x"), scm.NewString("09"), scm.NewString("9.5"), scm.NewString("1e3"), scm.NewString("9007199254740993"), scm.NewString("name"), scm.NewSymbol("name"), scm.NewString("2023-11-14 22:13:20"), scm.NewString("")}
	for _, a := range values {
		for _, b := range values {
			if !scm.Equal(a, b) && !scm.Equal(b, a) {
				continue
			}
			ah, ap := uniqueGuardHash(a)
			bh, bp := uniqueGuardHash(b)
			if ap && bp && ah != bh {
				t.Errorf("equal values %s / %s use disjoint guards", scm.String(a), scm.String(b))
			}
		}
	}
}

func TestUniqueTypedHashMatchesEquality(t *testing.T) {
	for _, definition := range []struct{ typ, collation string }{
		{"VARCHAR", "bin"}, {"VARCHAR", "utf8mb4_general_ci"}, {"VARCHAR", "utf8mb4_unicode_520_ci"},
		{"INT", "bin"}, {"DOUBLE", "bin"},
	} {
		t.Run(definition.typ+"-"+definition.collation, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "t_unique_typed_hash")
			tbl.CreateColumn("value", definition.typ, []int{40}, nil)
			column := tbl.Columns[len(tbl.Columns)-1]
			column.Collation = definition.collation
			column.UpdateSanitizer()
			values := []scm.Scmer{scm.NewNil(), scm.NewBool(false), scm.NewBool(true), scm.NewInt(0), scm.NewInt(3), scm.NewFloat(math.Copysign(0, -1)), scm.NewFloat(3), scm.NewString("03"), scm.NewString("3")}
			if definition.typ == "VARCHAR" {
				for _, value := range []string{"sail", "ſail", "K", "K", "Σ", "σ", "ς", "I", "i", "İ", "ı"} {
					values = append(values, scm.NewString(value))
				}
			}
			equal := tbl.uniqueColumnEqual("value")
			for _, left := range values {
				for _, right := range values {
					if !equal(left, right) {
						continue
					}
					a, ap := tbl.uniqueColumnHash("value", left)
					b, bp := tbl.uniqueColumnHash("value", right)
					if !ap || !bp || a != b {
						t.Errorf("equal typed values %s / %s have hashes %x / %x (precise %v / %v)", scm.String(left), scm.String(right), a, b, ap, bp)
					}
				}
			}
		})
	}
}

func TestInsertRoutingReservationsDoNotOverbook(t *testing.T) {
	shard := &storageShard{}
	releases := make(chan func(), 32)
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if release, ok := reserveInsertRows(shard, 1, 10, false, -1); ok {
				releases <- release
			}
		}()
	}
	workers.Wait()
	close(releases)
	if shard.pendingInsertRows.Load() != 10 {
		t.Fatalf("reserved %d rows, expected capacity 10", shard.pendingInsertRows.Load())
	}
	for release := range releases {
		release()
	}
	if shard.pendingInsertRows.Load() != 0 {
		t.Fatal("reservations not released")
	}
}

func TestInsertBalancedShardsSurviveWALReplay(t *testing.T) {
	oldCPUs := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(oldCPUs)
	tbl, persistence := createDurabilityTestTable(t, "tinsertbalancedrecovery", 0)
	topology := tbl.pinActiveTopology()
	defer func() { topology.releaseOperation() }()
	first, releaseFirst, _ := tbl.routeFreeInsert(topology, 1)
	defer releaseFirst()
	_, _, next := tbl.routeFreeInsert(topology, 1)
	if next == topology || next == nil {
		t.Fatal("busy routing did not publish a writer lane")
	}
	topology.releaseOperation()
	topology = tbl.pinActiveTopology()
	second, releaseSecond, _ := tbl.routeFreeInsert(topology, 1)
	defer releaseSecond()
	if first == second {
		t.Fatal("writers share a shard")
	}
	for i, shard := range []*storageShard{first, second} {
		release := shard.GetWrite(nil).Release
		shard.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(int64(i + 1)), scm.NewString("durable")}}, false, nil, nil, false, nil, nil)
		release()
	}
	reloaded := reloadTableFromPersistence(t, "tinsertbalancedrecovery", persistence)
	if len(reloaded.ActiveShards()) != 2 {
		t.Fatal("durable schema lost a writer lane")
	}
	if reloaded.CountExact() != 2 {
		t.Fatalf("WAL replay restored %d rows, expected 2", reloaded.CountExact())
	}
}

func TestInsertConcurrentAutoIncrementReservations(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinsertatomicids")
	tbl.Columns[0].AutoIncrement = true
	var workers sync.WaitGroup
	for w := 0; w < 4; w++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := 0; i < 64; i++ {
				tbl.Insert(nil, [][]scm.Scmer{{}}, nil, scm.NewNil(), false, nil)
			}
		}()
	}
	workers.Wait()
	if last := atomic.LoadUint64(&tbl.Auto_increment); last != 256 {
		t.Fatalf("counter=%d, expected 256", last)
	}
	seen := make(map[int64]bool)
	for _, shard := range tbl.ActiveShards() {
		release := shard.GetRead(nil, false)
		shard.mu.RLock()
		for i := range shard.deltaRows() {
			id := shard.getDelta(i, "id").Int()
			if seen[id] {
				t.Errorf("duplicate AUTO_INCREMENT id %d", id)
			}
			seen[id] = true
		}
		shard.mu.RUnlock()
		release.Release()
	}
	if len(seen) != 256 {
		t.Fatalf("generated %d distinct IDs", len(seen))
	}
	if base := tbl.reserveAutoIncrement(1000, 2); base != 1000 {
		t.Fatalf("explicit ID range starts at %d", base)
	}
	if base := tbl.reserveAutoIncrement(900, 1); base != 1002 {
		t.Fatalf("counter moved backwards to %d", base)
	}
}

func TestInsertPartitioningTrafficSchemaCheckpoint(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinserttrafficschema")
	tbl.Columns[0].PartitioningScore = 2
	for i := 0; i < 3; i++ {
		tbl.AddPartitioningScore([]string{"id"}, false)
	}
	data, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint struct {
		Columns []struct{ PartitioningScore int }
	}
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.Columns[0].PartitioningScore != 5 {
		t.Fatalf("persisted score=%d", checkpoint.Columns[0].PartitioningScore)
	}
	if tbl.Columns[0].PartitioningScore != 2 {
		t.Fatal("checkpoint mutated declaration-owned score")
	}
}

func TestInsertSQLLockWaitDoesNotBlockOwnerUniqueKey(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinsertlockowner")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	owner := &scm.SessionState{}
	tbl.tableLockOwner.Store(owner)
	tbl.tableLockState.Store(-1)
	unlocked := false
	defer func() {
		if !unlocked {
			tbl.unlockTableWrite()
		}
	}()
	waiter := scm.RegisterSession("root", "insert-test", tbl.schema.Name)
	defer scm.UnregisterSession(waiter.ID)
	waiterTx := NewTxContext(TxCursorStability)
	waiterTx.SessionState = waiter
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil, waiterTx)
	}()
	// The waiter must reach the lock condition without reserving the UNIQUE key.
	deadline := time.Now().Add(2 * time.Second)
waiting:
	for {
		for _, row := range callBuiltin(t, "show_processlist", scm.NewBool(true)).Slice() {
			fields := row.Slice()
			if uint64(fields[1].Int()) == waiter.ID && fields[13].String() == "Waiting for table lock" {
				break waiting
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("waiter did not reach SQL lock")
		}
		runtime.Gosched()
	}
	tx := NewTxContext(TxCursorStability)
	tx.SessionState = owner
	ownerDone := make(chan any, 1)
	go func() {
		defer func() { ownerDone <- recover() }()
		tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil, tx)
		tx.Commit()
	}()
	select {
	case caught := <-ownerDone:
		if caught != nil {
			t.Fatalf("owner INSERT: %v", caught)
		}
	case <-time.After(2 * time.Second):
		tbl.unlockTableWrite()
		unlocked = true
		<-ownerDone
		<-done
		t.Fatal("lock waiter retained the owner's UNIQUE guard")
	}
	tbl.unlockTableWrite()
	unlocked = true
	select {
	case caught := <-done:
		if !strings.Contains(fmt.Sprint(caught), "Duplicate entry") {
			t.Fatalf("waiter=%v, expected collision with owner", caught)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not resume")
	}
}

// Recovery must reconstruct the same allocator high-water mark as live INSERT,
// while retaining every explicitly supplied row, including nonpositive IDs.
func TestInsertAutoIncrementWALReplayPreservesSignedIDs(t *testing.T) {
	for _, test := range []struct {
		name string
		ids  []int64
		last uint64
	}{
		{"negative", []int64{-1}, 0},
		{"minimum", []int64{math.MinInt64}, 0},
		{"zero", []int64{0}, 0},
		{"mixed", []int64{-1, 41, math.MinInt64, 0, 7}, 41},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "tautoidreplay" + test.name
			tbl, persistence := createDurabilityTestTable(t, name, 0)
			tbl.Columns[0].AutoIncrement = true
			tbl.publishShowColumnsSnapshot(false)
			tbl.schema.save()
			rows := make([][]scm.Scmer, len(test.ids))
			for i, id := range test.ids {
				rows[i] = []scm.Scmer{scm.NewInt(id), scm.NewString("explicit")}
			}
			tbl.Insert([]string{"id", "payload"}, rows, nil, scm.NewNil(), false, nil)
			if got := atomic.LoadUint64(&tbl.Auto_increment); got != test.last {
				t.Fatalf("live counter = %d, want %d", got, test.last)
			}

			for restart := 0; restart < 2; restart++ {
				tbl = reloadTableFromPersistence(t, name, persistence)
				wantLast := test.last + uint64(restart)
				if got := atomic.LoadUint64(&tbl.Auto_increment); got != wantLast {
					t.Fatalf("restart %d counter = %d, want %d", restart+1, got, wantLast)
				}
				wantRows := uint(len(rows) + restart)
				if got := tbl.CountExact(); got != wantRows {
					t.Fatalf("restart %d recovered %d rows, want %d", restart+1, got, wantRows)
				}
				var generated int64
				tbl.Insert([]string{"payload"}, [][]scm.Scmer{{scm.NewString("generated")}}, nil,
					scm.NewNil(), false, func(id int64) { generated = id })
				if generated != int64(wantLast+1) || tbl.CountExact() != wantRows+1 {
					t.Fatalf("restart %d generated %d with %d rows, want ID %d and %d rows",
						restart+1, generated, tbl.CountExact(), wantLast+1, wantRows+1)
				}
			}
		})
	}
}

func TestInsertCollisionAndUpdateShareAtomicRow(t *testing.T) {
	tbl := setupUniqueContentionTable(t, "free")
	cols := []string{"id", "bucket", "val"}
	row := [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(0), scm.NewInt(0)}}
	tbl.Insert(cols, row, nil, scm.NewNil(), false, nil)
	cb := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		old := a[0].Int()
		runtime.Gosched()
		return scm.Apply(a[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(old + 1)}))
	})
	scanCb := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { scm.Apply(cb, a[1:]...); return a[0] })
	trueFn := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	combine := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	var done sync.WaitGroup
	start := make(chan struct{})
	for worker := 0; worker < 2; worker++ {
		done.Add(1)
		go func(worker int) {
			defer done.Done()
			defer func() {
				if err := recover(); err != nil {
					t.Errorf("mutation: %v", err)
				}
			}()
			<-start
			for i := 0; i < 128; i++ {
				if worker == 0 {
					tbl.Insert(cols, row, []string{"val", "$update"}, cb, false, nil)
				} else {
					tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueFn, []string{"val", "$update"}, scanCb, scm.NewInt(0), combine, false)
				}
			}
		}(worker)
	}
	close(start)
	done.Wait()
	var total int64
	for _, s := range tbl.ActiveShards() {
		r := s.ColumnReaderTx(nil, "val", false, ReadAuto)
		for id := uint32(0); id < s.main_count+uint32(len(s.deltaRows())); id++ {
			if !s.deletions.Get(uint(id)) {
				total += r.GetValue(id).Int()
			}
		}
	}
	if total != 256 || tbl.CountExact() != 1 {
		t.Fatalf("mixed increments=%d rows=%d; want 256 on one row", total, tbl.CountExact())
	}
}

func TestUniqueCollatedIndexLookup(t *testing.T) {
	for _, pair := range [][2]string{{"Alpha", "ALPHA"}, {"ſail", "sail"}, {"Key", "key"}} {
		t.Run(pair[0], func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "t_unique_collated_lookup")
			tbl.CreateColumn("label", "VARCHAR", []int{40}, nil)
			col := tbl.Columns[len(tbl.Columns)-1]
			col.Collation = "utf8mb4_general_ci"
			col.UpdateSanitizer()
			tbl.Unique = []uniqueKey{{Id: "label", Cols: []string{"label"}}}
			tbl.Insert([]string{"label"}, [][]scm.Scmer{{scm.NewString(pair[0])}}, nil, scm.NewNil(), false, nil)
			for _, shard := range tbl.ActiveShards() {
				access := exactScanAccess(newExactScanAccessSchema([]string{"label"}), []scm.Scmer{scm.NewString(pair[1])})
				if _, found := shard.GetRecordidForUnique(access, nil, true); !found {
					t.Fatal("collated key absent from UNIQUE index")
				}
			}
		})
	}
}

func TestUniqueTextProbeDuringGlobalDefinitions(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_bootstrap_lookup")
	tbl.CreateColumn("label", "VARCHAR", []int{40}, nil)
	tbl.Unique = []uniqueKey{{Id: "label", Cols: []string{"label"}}}
	tbl.Insert([]string{"label"}, [][]scm.Scmer{{scm.NewString("key")}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	access := exactScanAccess(newExactScanAccessSchema([]string{"label"}), []scm.Scmer{scm.NewString("key")})
	// Replication starts while lib/main.scm is still defining unrelated globals.
	// Engine collation construction must not consult that mutable environment.
	const iterations = 4096
	name := scm.Symbol("unique_probe_bootstrap_definition")
	defer delete(scm.Globalenv.Vars, name)
	start, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		<-start
		for i := range iterations {
			scm.Globalenv.Vars[name] = scm.NewInt(int64(i))
		}
	}()
	close(start)
	for range iterations {
		if _, found := shard.GetRecordidForUnique(access, nil, true); !found {
			t.Error("global definitions changed UNIQUE membership")
			break
		}
	}
	<-finished
}

func TestUniqueProbePrunesKnownPartitionDimensions(t *testing.T) {
	tbl := setupUniqueContentionTable(t, "extra")
	topology := tbl.activeTopology()
	for _, tc := range []struct {
		columns []string
		values  []scm.Scmer
		indices []int
	}{
		{[]string{"id"}, []scm.Scmer{scm.NewInt(20)}, []int{2, 3}},
		{[]string{"bucket"}, []scm.Scmer{scm.NewInt(0)}, []int{0, 2}},
		{[]string{"id", "bucket"}, []scm.Scmer{scm.NewInt(20), scm.NewInt(0)}, []int{2}},
		{[]string{"val"}, []scm.Scmer{scm.NewInt(20)}, []int{0, 1, 2, 3}},
	} {
		selected := tbl.uniqueProbeShards(topology, tc.columns, tc.values)
		if len(selected) != len(tc.indices) {
			t.Fatalf("%v selected %d shards, want %v", tc.columns, len(selected), tc.indices)
		}
		for i, index := range tc.indices {
			if selected[i] != topology.shards[index] {
				t.Fatalf("%v selected wrong shard at %d", tc.columns, i)
			}
		}
	}
}

func TestUniqueCollatedPartitionProbePreservesOtherDimensions(t *testing.T) {
	tbl := setupUniqueContentionTable(t, "extra")
	tbl.CreateColumn("label", "VARCHAR", []int{40}, []scm.Scmer{scm.NewString("collate"), scm.NewString("utf8mb4_general_ci")})
	active := tbl.activeTopology()
	topology := tableShardTopology{mode: ShardModePartition, shards: active.shards, dimensions: []shardDimension{
		{Column: "label", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewString("M")}},
		active.dimensions[1],
	}}
	selected := tbl.uniqueProbeShards(&topology, []string{"label", "bucket"}, []scm.Scmer{scm.NewString("sail"), scm.NewInt(0)})
	if len(selected) != 2 || selected[0] != topology.shards[0] || selected[1] != topology.shards[2] {
		t.Fatal("collated key used raw string partitioning or lost the independent dimension")
	}
}

func TestUniqueNullRowAcquiresFirstConcreteKey(t *testing.T) {
	for _, main := range []bool{false, true} {
		t.Run(fmt.Sprint(main), func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "t_unique_null_to_concrete")
			tbl.CreateColumn("value", "INT", nil, nil)
			tbl.Unique = []uniqueKey{{Id: "value", Cols: []string{"value"}}}
			tbl.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewNil()}}, nil, scm.NewNil(), false, nil)
			if main {
				if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
					t.Fatal(result)
				}
			}
			tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
				scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
				[]string{"$update"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
					scm.Apply(values[1], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(7)}))
					return scm.NewInt(1)
				}), scm.NewInt(0), scm.Globalenv.Vars[scm.Symbol("+")], false)
			found := false
			for _, shard := range tbl.ActiveShards() {
				_, present := shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"value"}), []scm.Scmer{scm.NewInt(7)}), nil, true)
				found = found || present
			}
			if !found || tbl.CountExact() != 1 {
				t.Fatal("NULL-to-key update did not publish exactly one replacement")
			}
			guard := tbl.guardUniqueWrites([]string{"value"}, [][]scm.Scmer{{scm.NewInt(7)}}, false, 0, nil)
			if !guard.tryLock() {
				t.Fatal("NULL-to-key update retained its first stripe")
			}
			guard.unlock()
		})
	}
}

func TestUniqueDeleteForwardsBorrowedGuardToRebuild(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tuniquedeleteforward", 1)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	original := tbl.ActiveShards()[0]
	successor := original.rebuild(true)
	if successor == original {
		t.Fatal("fixture did not create a successor")
	}
	target, ok := original.translateNextRecid(0)
	if !ok {
		t.Fatal("rebuild omitted recid translation")
	}
	guard := tbl.guardUniqueWrites([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, false, 0, nil)
	guard.lock()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		original.UpdateFunction(0, false, false, nil, &guard)()
	}()
	select {
	case failure := <-done:
		guard.unlock()
		if failure != nil {
			t.Fatalf("DELETE forwarding: %v", failure)
		}
	case <-time.After(3 * time.Second):
		guard.unlock()
		<-done
		t.Fatal("successor DELETE reacquired its caller's UNIQUE stripe")
	}
	if !original.deletions.Get(0) || !successor.deletions.Get(uint(target)) {
		t.Fatal("DELETE did not reach both live generations")
	}
}

func TestUniqueNativeAfterInsertCanUpdatePublishedKey(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tuniqueafterinsert", 0)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.AddTrigger(TriggerDescription{Name: "update_inserted_key", Timing: AfterInsert, Func: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if !args[3].IsNil() {
			panic("native trigger acquired a synthetic transaction")
		}
		shard := tbl.ActiveShards()[0]
		access := exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(1)})
		id, found := shard.GetRecordidForUnique(access, nil, true)
		if !found {
			panic("AFTER trigger could not see the published row")
		}
		return shard.UpdateFunction(id, false, false, nil, nil)(scm.NewSlice([]scm.Scmer{scm.NewString("payload"), scm.NewString("after-insert")}))
	})})
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("before-trigger")}}, nil, scm.NewNil(), false, nil)
	}()
	select {
	case failure := <-done:
		if failure != nil {
			t.Fatalf("native AFTER INSERT: %v", failure)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("AFTER INSERT waited for its own key guard")
	}
	shard := tbl.ActiveShards()[0]
	id, found := shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(1)}), nil, true)
	if !found || shard.rowValueByRecidLocked(id, "payload").String() != "after-insert" {
		t.Fatal("native INSERT returned before its AFTER trigger's update")
	}
}

func TestUniqueProbeDomainRetainsOnlyDistinctShards(t *testing.T) {
	const shardCount, rowCount = 128, 4096
	tbl := &table{Name: "bounded_domain", Unique: []uniqueKey{{Id: "key", Cols: []string{"key"}}}}
	shards := make([]*storageShard, shardCount)
	for i := range shards {
		shards[i] = &storageShard{t: tbl}
		shards[i].uuid[0] = byte(i + 1)
	}
	topology := &tableShardTopology{mode: ShardModeFree, shards: shards}
	rows := make([][]scm.Scmer, rowCount)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i))}
	}
	columns := []string{"key"}
	targets := []*storageShard{shards[0], nil, shards[0]}
	measurement := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			domain := tbl.uniqueProbeDomain(topology, columns, rows, false, targets)
			if len(domain) != shardCount {
				b.Fatalf("UNIQUE domain = %d, want %d", len(domain), shardCount)
			}
			for i, shard := range domain {
				if shard != shards[i] {
					b.Fatal("UNIQUE domain changed first-occurrence order")
				}
			}
		}
	})
	// The old rows×shards intermediate exceeded 16 MiB for this fixture. Allow
	// 256 KiB for the key tuples and the distinct routing set; fixture payload is
	// allocated before measurement and the helper neither loads nor claims it.
	t.Logf("%d rows over %d shards: %s, %d allocated bytes", rowCount, shardCount, measurement.String(), measurement.AllocedBytesPerOp())
	if allocated := measurement.AllocedBytesPerOp(); allocated > 256*1024 {
		t.Fatalf("UNIQUE domain allocated %d bytes for %d distinct shards", allocated, shardCount)
	}
}

func TestUniqueCoercingCollationHashPreservesEquality(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_coercing_collation_hash")
	tbl.CreateColumn("value", "ANY", nil, nil)
	declaration := tbl.uniqueColumnDeclaration("value")
	declaration.Collation = "utf8mb4_general_ci"
	values := []scm.Scmer{scm.NewNil(), scm.NewBool(false), scm.NewBool(true), scm.NewInt(0), scm.NewInt(3), scm.NewInt(1000), scm.NewInt(9007199254740993), scm.NewFloat(3), scm.NewFloat(3.5), scm.NewDate(1700000000)}
	for _, text := range []string{"", "0", "3", "03", "3.5", "3X", "3x", "1E3", "1e3", "9007199254740993", "K", "K", "Σ", "σ", "ς", "I", "i", "İ", "ı", "sail", "ſail", "false", "NaN", "nan", "Inf", "inf"} {
		values = append(values, scm.NewString(text), scm.NewSymbol(text))
	}
	for _, left := range values {
		for _, right := range values {
			if !uniqueColumnValuesEqual(declaration, left, right) && !uniqueColumnValuesEqual(declaration, right, left) {
				continue
			}
			a, ap := tbl.uniqueColumnHash("value", left)
			b, bp := tbl.uniqueColumnHash("value", right)
			if ap && bp && a != b {
				t.Errorf("equal coercing values %s / %s use disjoint guards", scm.String(left), scm.String(right))
			}
		}
	}
	for _, value := range []scm.Scmer{scm.NewInt(3), scm.NewString("K")} {
		if _, precise := tbl.uniqueColumnHash("value", value); !precise {
			t.Error("ordinary collated cache keys unnecessarily reserve every stripe")
		}
	}
}

// Previously every touched table retained 64 mutexes for the rest of its life.
// Sequentially touching many relations must leave no table-owned lock state.
func TestUniqueNodePoolReleasesIdleTables(t *testing.T) {
	pool := nodeUniqueWriteLocks()
	for i := 0; i < 4096; i++ {
		tbl := &table{Unique: []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}}
		guard := tbl.guardUniqueWrites([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, false, 0, nil)
		if guard.locks != pool {
			t.Fatal("table allocated a private lock pool")
		}
		guard.lock()
		guard.release()
	}
	// The shared pool contains mutexes only; it cannot retain a table owner.
	for i := range pool.stripes {
		if !pool.stripes[i].TryLock() {
			t.Fatalf("idle stripe %d retained", i)
		}
		pool.stripes[i].Unlock()
	}
	t.Logf("node pool: %d stripes, %d bytes of fixed synchronization state", len(pool.stripes), uintptr(len(pool.stripes))*unsafe.Sizeof(uniqueWriteStripe{})+unsafe.Sizeof(*pool))
}

// A global mutex held throughout a callback deadlocks this ordinary nested
// write when the two tables happen to hash into the same stripe.
func TestUniqueNodePoolNestedTriggerHashCollision(t *testing.T) {
	outer := setupScanParallelTestTable(t, "tunique_node_trigger")
	outer.CreateColumn("val", "INT", nil, nil)
	outer.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	inner, _ := CreateTable(outer.schema.Name, "trigger_target", Memory, false)
	inner.CreateColumn("id", "INT", nil, nil)
	inner.CreateColumn("val", "INT", nil, nil)
	inner.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	columns := []string{"id", "val"}
	outerRow := [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(0)}}
	outerMask := outer.guardUniqueWrites(columns, outerRow, false, 0, nil).mask
	var innerRow [][]scm.Scmer
	for id := int64(1); id <= int64(len(nodeUniqueWriteLocks().stripes))*64; id++ {
		row := [][]scm.Scmer{{scm.NewInt(id), scm.NewInt(0)}}
		if inner.guardUniqueWrites(columns, row, false, 0, nil).mask.intersects(outerMask) {
			innerRow = row
			break
		}
	}
	if innerRow == nil {
		t.Fatal("no colliding test key")
	}
	outer.Insert(columns, outerRow, nil, scm.NewNil(), false, nil)
	inner.Insert(columns, innerRow, nil, scm.NewNil(), false, nil)
	increment := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.Apply(args[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(args[0].Int() + 1)}))
	})
	outer.AddTrigger(TriggerDescription{Name: "update_other_table", Timing: BeforeUpdate, Func: scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		inner.Insert(columns, innerRow, []string{"val", "$update"}, increment, false, nil)
		return scm.NewNil()
	})})
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		outer.Insert(columns, outerRow, []string{"val", "$update"}, increment, false, nil)
	}()
	select {
	case failure := <-done:
		if failure != nil {
			t.Fatal(failure)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("nested trigger waited on another table's node stripe")
	}
	for _, tbl := range []*table{outer, inner} {
		shard := tbl.ActiveShards()[0]
		var total int64
		reader := shard.ColumnReaderTx(nil, "val", false, ReadAuto)
		for id := uint32(0); id < shard.main_count+uint32(len(shard.deltaRows())); id++ {
			if !shard.deletions.Get(uint(id)) {
				total += reader.GetValue(id).Int()
			}
		}
		ReleaseColumnReader(reader)
		if total != 1 || tbl.CountExact() != 1 {
			t.Fatalf("%s: value=%d rows=%d", tbl.Name, total, tbl.CountExact())
		}
		assertUniqueWriteLocksReleased(t, tbl)
	}
}

func BenchmarkUniqueNodeThroughput(b *testing.B) {
	for _, count := range []int{1, 64} {
		b.Run(fmt.Sprintf("tables=%d", count), func(b *testing.B) {
			previous := Basepath
			Basepath = b.TempDir()
			b.Cleanup(func() { Basepath = previous; databases.Remove("tunique_node_bench") })
			Init(scm.Globalenv)
			LoadDatabases()
			CreateDatabase("tunique_node_bench", false)
			tables := make([]*table, count)
			columns := []string{"id", "val"}
			for i := range tables {
				tbl, _ := CreateTable("tunique_node_bench", fmt.Sprint("items_", i), Memory, false)
				tbl.CreateColumn("id", "INT", nil, nil)
				tbl.CreateColumn("val", "INT", nil, nil)
				tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
				rows := make([][]scm.Scmer, 64)
				for key := range rows {
					rows[key] = []scm.Scmer{scm.NewInt(int64(key + 1)), scm.NewInt(0)}
				}
				tbl.Insert(columns, rows, nil, scm.NewNil(), false, nil)
				tables[i] = tbl
			}
			increment := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				return scm.Apply(args[1], scm.NewSlice([]scm.Scmer{scm.NewString("val"), scm.NewInt(args[0].Int() + 1)}))
			})
			var next atomic.Uint64
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				worker := int(next.Add(1) - 1)
				tbl := tables[worker%len(tables)]
				row := [][]scm.Scmer{{scm.NewInt(int64(worker%64 + 1)), scm.NewInt(0)}}
				for pb.Next() {
					tbl.Insert(columns, row, []string{"val", "$update"}, increment, false, nil)
				}
			})
			b.StopTimer()
			var total int64
			for _, tbl := range tables {
				for _, shard := range tbl.ActiveShards() {
					reader := shard.ColumnReaderTx(nil, "val", false, ReadAuto)
					for id := uint32(0); id < shard.main_count+uint32(len(shard.deltaRows())); id++ {
						if !shard.deletions.Get(uint(id)) {
							total += reader.GetValue(id).Int()
						}
					}
					ReleaseColumnReader(reader)
				}
				if tbl.CountExact() != 64 {
					b.Fatalf("rows=%d, expected 64", tbl.CountExact())
				}
			}
			if total != int64(b.N) {
				b.Fatalf("lost increments: %d, expected %d", total, b.N)
			}
		})
	}
}

// Exercise complete real mutations with one stripe and a multiword pool in
// separate processes: replacing a live pool in-process would invalidate guards.
func TestUniqueNodePoolConfigured(t *testing.T) {
	if value := os.Getenv("MEMCP_UNIQUE_POOL_CHILD"); value != "" {
		factor, err := strconv.Atoi(value)
		if err != nil {
			t.Fatal(err)
		}
		Settings.UniqueScaleFactor = factor
		InitSettings()
		expected := 1
		if factor != 0 {
			expected = factor * runtime.GOMAXPROCS(0)
		}
		if len(nodeUniqueWriteLocks().stripes) != expected {
			t.Fatal("CPU multiplier not applied")
		}
		t.Run("atomic collisions", TestInsertCollisionReadModifyWriteContention)
		t.Run("atomic before collisions", TestInsertCollisionReadModifyWriteBeforeTrigger)
		t.Run("aliases", TestInsertCollisionDifferentUniqueKeysShareRow)
		t.Run("before nested update", TestUniqueNodePoolNestedTriggerHashCollision)
		t.Run("snapshot publication", TestUniqueSnapshotDrainsAdmittedWriter)
		t.Run("MOSI configuration collision", TestReplicationUniqueAcquisitionAllowsCollidingConfigWrite)
		t.Run("no retained table pools", TestUniqueNodePoolReleasesIdleTables)
		return
	}
	for _, config := range []struct{ factor, cpus string }{{"0", "3"}, {"8", "20"}} {
		t.Run(config.factor, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestUniqueNodePoolConfigured$", "-test.v", "-test.timeout=90s")
			command.Env = append(os.Environ(), "MEMCP_UNIQUE_POOL_CHILD="+config.factor, "GOMAXPROCS="+config.cpus)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("pool factor=%s: %v\n%s", config.factor, err, output)
			}
		})
	}
}

// READ publication must drain an admitted writer before a cache initializer
// can borrow the global pool. Both the first reader and adjacent READ owners
// must wait, while the admitted writer remains able to finish its INSERT.
func TestUniqueSnapshotDrainsAdmittedWriter(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tunique_snapshot_admission")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	columns := []string{"id"}
	rows := [][]scm.Scmer{{scm.NewInt(7)}}
	guard := tbl.guardUniqueWrites(columns, rows, false, 0, nil)
	guard.lock()
	defer guard.release()
	readers := make(chan func(), 2)
	for index := int64(1); index <= 2; index++ {
		go func() {
			readers <- acquireTableLock(tbl.schema.Name, tbl.Name, false, true, &scm.SessionState{}, 0)
		}()
		deadline := time.Now().Add(2 * time.Second)
		for tbl.tableLockState.Load() != index {
			if time.Now().After(deadline) {
				t.Fatal("READ ownership was not published")
			}
			runtime.Gosched()
		}
		select {
		case unlock := <-readers:
			unlock()
			t.Fatal("snapshot entered before the admitted writer finished")
		default:
		}
	}
	late := tbl.guardUniqueWrites(columns, rows, false, 0, nil)
	late.mask = uniqueWriteMask{} // no stripe contention can explain a refusal
	if late.tryLock() {
		late.release()
		t.Fatal("a new writer bypassed published READ ownership")
	}
	if tbl.tableLockWriters.Load() != 1 {
		t.Fatal("refused admission leaked a writer")
	}
	done := make(chan struct{})
	go func() {
		tbl.ActiveShards()[0].Insert(columns, rows, false, tbl.columnNamesSnapshot.Load(), nil, false, nil, &guard)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("admitted INSERT waited for the pending snapshot")
	}
	guard.release()
	for index := 0; index < 2; index++ {
		select {
		case unlock := <-readers:
			defer unlock()
		case <-time.After(2 * time.Second):
			t.Fatal("snapshot did not resume after writer departure")
		}
	}
	if tbl.CountExact() != 1 || tbl.tableLockWriters.Load() != 0 {
		t.Fatal("snapshot missed the completed INSERT or retained admission")
	}
	// A cache is a different relation, but still uses the same mutex. It must
	// make progress while these source READ locks remain held.
	cache := setupScanParallelTestTable(t, "tunique_snapshot_cache")
	cache.Unique = tbl.Unique
	for key := int64(1); ; key++ {
		candidate := [][]scm.Scmer{{scm.NewInt(key)}}
		if cache.guardUniqueWrites(columns, candidate, false, 0, nil).mask.intersects(guard.mask) {
			cache.Insert(columns, candidate, nil, scm.NewNil(), false, nil)
			break
		}
	}
}
