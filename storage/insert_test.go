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

import "fmt"
import "math"
import "sync"
import "time"
import "runtime"
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
	firstMask := tbl.guardUniqueWrites([]string{"id"}, first, false, 0).mask
	var second [][]scm.Scmer
	for i := int64(101); i < 1000; i++ {
		rows := [][]scm.Scmer{{scm.NewInt(i)}}
		if tbl.guardUniqueWrites([]string{"id"}, rows, false, 0).mask&firstMask == 0 {
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
		tbl.ProcessUniqueCollision([]string{"id"}, first, false, func(_ [][]scm.Scmer, _ func() bool) { close(entered); <-resume }, nil, nil, 0, nil)
	}()
	<-entered
	defer func() { close(resume); <-done }()
	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		tbl.ProcessUniqueCollision([]string{"id"}, second, false, func(_ [][]scm.Scmer, _ func() bool) {}, nil, nil, 0, nil)
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

func TestInsertCollisionCallbackRechecksEarlierKeys(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinsertcallbackrecheck")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	columns := []string{"id", "name"}
	tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("taken")}}, nil, scm.NewNil(), false, nil)
	var collisions []string
	tbl.ProcessUniqueCollision(columns, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("taken")}, {scm.NewInt(3), scm.NewString("free")}}, false,
		func(rows [][]scm.Scmer, _ func() bool) { t.Fatal("accepted key added by collision callback") }, nil,
		func(key string, _ []scm.Scmer) {
			collisions = append(collisions, key)
			if key == "name" {
				tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(3), scm.NewString("other")}}, nil, scm.NewNil(), false, nil)
			}
		}, 0, nil)
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
		shard.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(int64(i + 1)), scm.NewString("durable")}}, false, nil, nil, false, nil)
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
		release := shard.GetRead(nil)
		shard.mu.RLock()
		for i := range shard.inserts {
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
