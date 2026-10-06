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

	"github.com/launix-de/memcp/scm"
)

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
