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
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestQuerySeqFromTxCachesOnlyAutocommitStatement(t *testing.T) {
	const cachedSeq uint64 = 17
	const statementSeq uint64 = 42
	tx := NewTxContext(TxCursorStability)
	tx.querySeq.Store(cachedSeq)

	scm.SetValues(map[string]any{"querySeq": statementSeq, "context": context.Background()}, func() {
		if got := querySeqFromTx(tx); got != statementSeq {
			t.Fatalf("explicit transaction reused another statement's sequence: got %d, want %d", got, statementSeq)
		}

		tx.autoCommit = true
		if got := querySeqFromTx(tx); got != cachedSeq {
			t.Fatalf("autocommit transaction did not reuse its statement sequence: got %d, want %d", got, cachedSeq)
		}
	})
}
