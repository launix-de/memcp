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

	run := func(query string, dst **TxContext) {
		seq := ss.BeginQuery("Query", query)
		defer ss.EndQuery(seq, "Sleep", "")
		WithAutocommit(session, ss, seq, query, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			tx := scmerToTxContext(a[0])
			*dst = tx
			if !tx.queryActive.Load() {
				t.Fatal("transaction is not marked active inside query")
			}
			if info := tx.queryInfo.Load(); info == nil || *info != query {
				t.Fatalf("transaction query text = %v, want %q", info, query)
			}
			return scm.NewBool(true)
		}))
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
