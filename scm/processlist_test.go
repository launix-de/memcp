/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

package scm

import (
	"context"
	"testing"
	"time"
)

func TestSessionStateTouchUpdatesLastUsed(t *testing.T) {
	ss := RegisterSession("u", "h", "db")
	defer UnregisterSession(ss.ID)

	before := ss.LastUsedNano()
	time.Sleep(time.Millisecond)
	ss.Touch()
	after := ss.LastUsedNano()
	if after <= before {
		t.Fatalf("expected LastUsedNano to advance, before=%d after=%d", before, after)
	}
}

func TestEvictHTTPSessionReleasesLocksAndKillsQuery(t *testing.T) {
	ss := RegisterSession("u", "h", "db")
	key := "sid"
	httpStates.Store(key, ss)

	var unlocked int
	var cancelled bool
	ss.AddLock(func() { unlocked++ })
	seq := ss.BeginQuery("Query", "SELECT 1")
	ss.SetCancel(seq, func() { cancelled = true })

	if !EvictHTTPSession(key) {
		t.Fatalf("expected eviction to succeed")
	}
	if unlocked != 1 {
		t.Fatalf("expected one lock release, got %d", unlocked)
	}
	if !cancelled {
		t.Fatalf("expected active query to be cancelled")
	}
	if _, ok := processList.Load(ss.ID); ok {
		t.Fatalf("expected evicted session to be removed from process list")
	}
}

func TestKillQueryMarksOnlyCurrentGeneration(t *testing.T) {
	ss := RegisterSession("u", "h", "db")
	defer UnregisterSession(ss.ID)

	seq1 := ss.BeginQuery("Query", "SELECT 1")
	seq2 := ss.BeginQuery("Query", "SELECT 2")
	if !ss.KillQuery(seq1) {
		t.Fatalf("expected first query generation to be killable")
	}
	SetValues(map[string]any{"querySeq": seq1, "context": context.Background()}, func() {
		if !ss.IsKilled() {
			t.Fatalf("expected seq1 to be marked killed")
		}
	})
	SetValues(map[string]any{"querySeq": seq2, "context": context.Background()}, func() {
		if ss.IsKilled() {
			t.Fatalf("expected seq2 to remain alive")
		}
	})
	ss.EndQuery(seq1, "Sleep", "")
	ss.EndQuery(seq2, "Sleep", "")
}
