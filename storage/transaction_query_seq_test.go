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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func TestTransactionVisibilityDoesNotWaitForDirectoryWriter(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_tx_visibility_directory", 4)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	tx.AddToDeleteMask(shard, 1)
	tx.mu.Lock()
	visible := make(chan bool, 1)
	go func() { visible <- tx.IsVisible(shard, 1) }()
	blocked := false
	select {
	case value := <-visible:
		if value {
			t.Error("transaction's own deletion became visible")
		}
	case <-time.After(time.Second):
		blocked = true
		t.Error("row visibility waited for the transaction directory writer")
	}
	tx.mu.Unlock()
	if blocked {
		<-visible
	}
}

func TestQueryCleanupJoinsAllWorkBeforeReleasingReaders(t *testing.T) {
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(nil, 1, "cleanup ordering")
	joined := 0
	tx.retainQueryReader(func() {
		if joined != 2 {
			t.Error("physical reader released before all work joined")
		}
		joined++
	})
	tx.RetainQueryWork(func() { joined++; panic("output failed") })
	tx.RetainQueryWork(func() { joined++ })
	var failure any
	func() {
		defer func() { failure = recover() }()
		tx.endQuery(1)
	}()
	if fmt.Sprint(failure) != "output failed" || joined != 3 || tx.queryActive.Load() {
		t.Fatalf("incomplete cleanup: failure=%v joined=%d active=%v", failure, joined, tx.queryActive.Load())
	}
}

func TestQuerySeqFromTxUsesExplicitTransactionStatement(t *testing.T) {
	const cachedSeq uint64 = 17
	tx := NewTxContext(TxCursorStability)
	tx.querySeq.Store(cachedSeq)

	if got := querySeqFromTx(tx); got != cachedSeq {
		t.Fatalf("transaction lost its explicit statement sequence: got %d, want %d", got, cachedSeq)
	}
}

func TestSessionStateFromTxUsesOnlyExplicitTransaction(t *testing.T) {
	ss := &scm.SessionState{ID: 23}
	tx := NewTxContext(TxCursorStability)
	tx.SessionState = ss
	if got := SessionStateFromTx(nil); got != nil {
		t.Fatalf("nil transaction unexpectedly resolved session: got %p", got)
	}
	if got := SessionStateFromTx(tx); got != ss {
		t.Fatalf("transaction lost explicit query session: got %p, want %p", got, ss)
	}
}

func TestTransactionInvocationsShareMutationsAndKeepIndependentCleanup(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tnative_invocation_state", 4)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	first, second := tx.newInvocation(), tx.newInvocation()
	firstSession, secondSession := &scm.SessionState{}, &scm.SessionState{}
	firstSeq := firstSession.BeginQuery("Cluster", "first native invocation")
	secondSeq := secondSession.BeginQuery("Cluster", "second native invocation")
	first.beginQuery(firstSession, firstSeq, "first native invocation")
	second.beginQuery(secondSession, secondSeq, "second native invocation")
	defer second.endQuery(secondSeq)
	var cleaned bool
	first.RetainQueryWork(func() { cleaned = true })
	firstWrite := shard.UpdateFunction(0, false, false, first)
	firstWrite(scm.NewSlice([]scm.Scmer{scm.NewString("payload"), scm.NewString("changed")}))
	other := NewTxContext(TxACID)
	if tx.IsVisible(shard, 0) || second.IsVisible(shard, 0) || !other.IsVisible(shard, 0) {
		t.Fatal("native invocations lost their shared delete overlay or leaked it")
	}
	first.endQuery(firstSeq)
	if !cleaned || !second.queryActive.Load() || second.queryInfo.Load() == nil || tx.State != TxActive {
		t.Fatal("one invocation cleaned up another invocation or its transaction")
	}
	if first.ID != second.ID || first.durableID() != tx.durableID() {
		t.Fatal("native invocation created a new transaction identity")
	}
	tx.Rollback()
	if !other.IsVisible(shard, 0) || first.State != TxAborted || shard.activeTransactions.Load() != 0 {
		t.Fatal("rollback did not cover the invocation's native mutation")
	}
	oldID := first.ID
	tx.reset(TxCursorStability)
	if first.ID != oldID || first.State != TxAborted || tx.ID == oldID {
		t.Fatal("session reuse changed a previous invocation's transaction identity")
	}
}

func TestWithAutocommitReusesParkedTransactionAndClearsQueryState(t *testing.T) {
	session := scm.NewSession()
	ss := &scm.SessionState{}
	var first, second *TxContext
	tbl, col := &table{}, &column{IsTemp: true}

	run := func(query string, dst **TxContext) {
		seq := ss.BeginQuery("Query", query)
		defer ss.EndQuery(seq, "Sleep", "")
		WithAutocommit(session, ss, seq, query, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			tx := scmerToTxContext(a[0])
			*dst = tx
			if !tx.retainQueryColumn(tbl, col) {
				t.Fatal("could not retain the prepared statement dependency")
			}
			if !tx.queryActive.Load() {
				t.Fatal("transaction is not marked active inside query")
			}
			if info := tx.queryInfo.Load(); info == nil || *info != query {
				t.Fatalf("transaction query text = %v, want %q", info, query)
			}
			return scm.NewBool(true)
		}))
		if tbl.cacheUsers != 0 || col.cacheUsers != 0 || (*dst).queryColumns != nil {
			t.Fatal("autocommit retained prepared cache columns after execution")
		}
		if (*dst).queryActive.Load() || (*dst).querySeq.Load() != 0 || (*dst).queryInfo.Load() != nil {
			t.Fatal("parked transaction retained finished query state")
		}
	}

	run("SELECT 1", &first)
	run("SELECT 2", &second)
	if first != second {
		t.Fatal("autocommit allocated a new transaction instead of reusing the session object")
	}
}

func TestTransactionChoosesQueryLocalCacheWhenSharedRecipesAreUnsafe(t *testing.T) {
	if txRequiresQueryLocalCache(nil) {
		t.Fatal("nil transaction disabled shared caches")
	}
	tx := NewTxContext(TxCursorStability)
	tx.autoCommit = true
	if txRequiresQueryLocalCache(tx) {
		t.Fatal("ordinary autocommit transaction disabled shared caches")
	}

	tx.autoCommit = false
	if !txRequiresQueryLocalCache(tx) {
		t.Fatal("explicit transaction reused a shared cache")
	}
	tx.autoCommit = true
	tx.Mode = TxACID
	if !txRequiresQueryLocalCache(tx) {
		t.Fatal("ACID transaction reused a shared cache")
	}

	tx.Mode = TxCursorStability
	tx.SessionState = &scm.SessionState{}
	tx.SessionState.AddLock(func() {})
	defer tx.SessionState.ReleaseAllLocks()
	if !txRequiresQueryLocalCache(tx) {
		t.Fatal("table-lock owner reused a shared cache")
	}
}

func TestContributionReadVersionRejectsOverlappingWriters(t *testing.T) {
	tbl := &table{}
	before := tbl.contributionReadVersion().Slice()
	entered, finish, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		tbl.beginContributionMutation()
		close(entered)
		<-finish
		tbl.endContributionMutation()
		close(done)
	}()
	<-entered
	tbl.beginContributionMutation()
	if !tbl.contributionReadVersion().IsNil() {
		t.Error("active writers published a reusable read version")
	}
	close(finish)
	<-done
	if !tbl.contributionReadVersion().IsNil() {
		t.Error("one completed writer hid another active writer")
	}
	tbl.endContributionMutation()
	after := tbl.contributionReadVersion().Slice()
	if before[0].Int() != after[0].Int() || before[1].Int() == after[1].Int() {
		t.Fatal("completed mutations must retain identity and advance revision")
	}
	replacement := (&table{}).contributionReadVersion().Slice()
	if replacement[0].Int() == after[0].Int() {
		t.Fatal("replacement table reused a prior identity")
	}
}

func TestContributionReadVersionTracksDMLAndVisibility(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tcontributionview")
	stamp := func() int64 {
		value := tbl.contributionReadVersion()
		if value.IsNil() {
			t.Fatal("completed operation left an active writer")
		}
		return value.Slice()[1].Int()
	}
	changed := func(before int64, operation string) {
		if stamp() == before {
			t.Fatalf("%s retained an obsolete read version", operation)
		}
	}
	before := stamp()
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	changed(before, "insert")
	for _, commit := range []bool{false, true} {
		tx := NewTxContext(TxACID)
		sp := tx.CreateSavepoint()
		tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(2)}}, nil, scm.NewNil(), false, nil, tx)
		before = stamp()
		tx.RollbackToSavepoint(sp)
		changed(before, "savepoint rollback")
		tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(3)}}, nil, scm.NewNil(), false, nil, tx)
		before = stamp()
		if commit {
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
		} else {
			tx.Rollback()
		}
		changed(before, "transaction visibility publication")
	}
	before = stamp()
	tbl.mu.Lock()
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	changed(before, "topology publication")
}

// A prepared computed column must survive unrelated cache pressure until every
// consumer in the statement finishes. Explicit transactions release per query.
func TestPreparedQueryColumnSurvivesEvictionUntilStatementCompletes(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(fmt.Sprint("panic=", panics), func(t *testing.T) {
			tbl := &table{}
			col := &column{IsTemp: true}
			tx := NewTxContext(TxCursorStability)
			tx.beginQuery(nil, 41, "prepared cache consumer")
			func() {
				defer func() { _ = recover() }()
				defer tx.endQuery(41)
				tx.retainQueryColumn(tbl, col)
				tx.retainQueryColumn(tbl, col)
				if tbl.cacheUsers != 1 || col.cacheUsers != 1 {
					t.Fatalf("duplicate prepare changed pin counts: table=%d column=%d", tbl.cacheUsers, col.cacheUsers)
				}
				if tbl.beginCacheEviction() || col.beginCacheEviction() {
					t.Fatal("cache pressure retired a prepared statement dependency")
				}
				if panics {
					panic("consumer failure")
				}
			}()
			if tbl.cacheUsers != 0 || col.cacheUsers != 0 || tx.queryColumns != nil {
				t.Fatal("finished query retained cache pins")
			}
			if !tbl.beginCacheEviction() || !col.beginCacheEviction() {
				t.Fatal("finished statement still blocks cache eviction")
			}
		})
	}
}

func TestParkedTransactionDoesNotRetainQueryColumn(t *testing.T) {
	tbl, col := &table{}, &column{IsTemp: true}
	tx := NewTxContext(TxCursorStability)
	tx.retainQueryColumn(tbl, col)
	if tx.queryColumns != nil || tbl.cacheUsers != 0 || col.cacheUsers != 0 {
		t.Fatal("parked transaction retained a statement dependency")
	}
}

func TestQueryRequestPreservesCancellationAndReusesStatementContext(t *testing.T) {
	session := scm.NewSession()
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Query", "several statements")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss.SetQueryContext(seq, ctx)
	defer ss.EndQuery(seq, "Sleep", "")
	var original *TxContext
	var firstGeneration uint64
	tbl, col := &table{}, &column{IsTemp: true}
	WithQueryRequest(session, ss, seq, "several statements", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		statement := a[0]
		for i := 0; i < 2; i++ {
			scm.Apply(statement, scm.NewString("statement"), scm.NewBool(false), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
				tx := scmerToTxContext(values[0])
				if original == nil {
					original = tx
					firstGeneration = tx.statementGeneration.Load()
				} else {
					if tx != original {
						t.Fatal("request allocated another transaction record")
					}
					if tx.statementGeneration.Load() == firstGeneration {
						t.Fatal("statement reused its predecessor's cache scope")
					}
				}
				if querySeqFromTx(tx) != seq {
					t.Fatal("statement changed request cancellation identity")
				}
				if !tx.retainQueryColumn(tbl, col) {
					t.Fatal("statement dependency was not retained")
				}
				return scm.NewBool(true)
			}))
			if ss.QueryContext(seq) != ctx {
				t.Fatal("statement ended request cancellation context")
			}
			if original.queryActive.Load() || original.statementGeneration.Load() != 0 || tbl.cacheUsers != 0 || col.cacheUsers != 0 {
				t.Fatal("completed statement retained its physical ownership")
			}
		}
		return scm.NewBool(true)
	}))
	if ss.QueryContext(seq) != nil || original.querySeq.Load() != 0 {
		t.Fatal("finished request retained cancellation ownership")
	}
}

func TestQueryRequestCancellationStopsFollowingStatementAndRejectsEscapedExecutor(t *testing.T) {
	session := scm.NewSession()
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Query", "cancelled request")
	ctx, cancel := context.WithCancel(context.Background())
	ss.SetQueryContext(seq, ctx)
	defer ss.EndQuery(seq, "Sleep", "")
	var escaped scm.Scmer
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("cancelled request executed its following statement")
			}
		}()
		WithQueryRequest(session, ss, seq, "cancelled request", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			escaped = a[0]
			scm.Apply(escaped, scm.NewString("first"), scm.NewBool(false), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return scm.NewNil() }))
			cancel()
			return scm.Apply(escaped, scm.NewString("must not execute"), scm.NewBool(false), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
				t.Fatal("cancelled statement body ran")
				return scm.NewNil()
			}))
		}))
	}()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("escaped request executor remained callable")
			}
		}()
		scm.Apply(escaped, scm.NewString("escaped"), scm.NewBool(false), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return scm.NewNil() }))
	}()
	if ss.QueryContext(seq) != nil {
		t.Fatal("cancelled request retained its context")
	}
}

func TestQueryRequestRecoveryCapabilityAndGlobalStatementScopes(t *testing.T) {
	var previous uint64
	for i := 0; i < 2; i++ {
		session := scm.NewSession()
		WithQueryRequest(session, nil, 0, "request", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			return scm.Apply(a[0], scm.NewString("statement"), scm.NewBool(false), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
				tx := scmerToTxContext(values[0])
				if generation := tx.statementGeneration.Load(); generation == 0 || generation == previous {
					t.Fatal("distinct sessions reused statement scope")
				} else {
					previous = generation
				}
				return scm.NewNil()
			}))
		}))
	}
	session := scm.NewSession()
	tx := transactionForSession(session)
	tx.State = TxAborted
	session.Func()(scm.NewString("transaction"), scm.NewAny(tx))
	for _, recovery := range []bool{false, true} {
		func() {
			defer func() {
				if got := recover(); (got != nil) == recovery {
					t.Fatalf("recovery=%v unexpected outcome: %v", recovery, got)
				}
			}()
			WithQueryRequest(session, nil, 0, "request", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				return scm.Apply(a[0], scm.NewString("opaque statement text"), scm.NewBool(recovery), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
					tx.Rollback()
					session.Func()(scm.NewString("transaction"), scm.NewNil())
					return scm.NewBool(true)
				}))
			}))
		}()
	}
}

func TestStatementRetainsCacheCarrierBeforeColumnInitialization(t *testing.T) {
	tbl := &table{Name: ".grp:statement_carrier", PersistencyMode: Cache}
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(nil, 1, "cache initialization")
	if !tx.retainQueryTable(tbl) {
		t.Fatal("could not retain unpublished cache carrier")
	}
	if tbl.beginCacheEviction() {
		t.Fatal("evicted carrier before its column initialization")
	}
	tx.endStatement()
	if !tbl.beginCacheEviction() {
		t.Fatal("statement did not release cache carrier")
	}
	if tx.retainQueryTable(tbl) != true {
		t.Fatal("inactive statement unexpectedly acquired a pin")
	}
	tx.beginQuery(nil, 2, "retired carrier")
	if tx.retainQueryTable(tbl) {
		t.Fatal("retired cache carrier was reused")
	}
	tx.endStatement()
}
