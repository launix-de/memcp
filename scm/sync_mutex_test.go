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

package scm

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type mutexTestTransaction struct {
	ss  *SessionState
	seq uint64
}

func (tx *mutexTestTransaction) QuerySessionState() (*SessionState, uint64) {
	return tx.ss, tx.seq
}

func TestContextPassesExplicitSession(t *testing.T) {
	result := Context(NewFunc(func(a ...Scmer) Scmer {
		return Apply(a[0], NewString("key"), NewInt(7))
	}))
	if result.Int() != 7 {
		t.Fatalf("expected explicit context session result 7, got %v", result)
	}
}

func TestMutexWaitStopsWhenContextIsCancelled(t *testing.T) {
	lock := Apply(Globalenv.Vars[Symbol("mutex")])
	holderEntered := make(chan struct{})
	releaseHolder := make(chan struct{})
	holderDone := make(chan struct{})
	go func() {
		defer close(holderDone)
		Apply(lock, NewFunc(func(_ ...Scmer) Scmer {
			close(holderEntered)
			<-releaseHolder
			return NewBool(true)
		}))
	}()
	select {
	case <-holderEntered:
	case <-time.After(time.Second):
		t.Fatal("mutex holder did not enter")
	}

	waiterCtx, cancelWaiter := context.WithCancel(context.Background())
	ss := &SessionState{}
	seq := ss.BeginQuery("Query", "mutex wait")
	ss.SetQueryContext(seq, waiterCtx)
	defer ss.EndQuery(seq, "Sleep", "")
	var waiterRan atomic.Bool
	waiterDone := make(chan any, 1)
	go func() {
		defer func() { waiterDone <- recover() }()
		Apply(lock, NewAny(&mutexTestTransaction{ss: ss, seq: seq}), NewFunc(func(_ ...Scmer) Scmer {
			waiterRan.Store(true)
			return NewBool(true)
		}))
	}()
	cancelWaiter()
	select {
	case recovered := <-waiterDone:
		if recovered != context.Canceled {
			t.Fatalf("expected context cancellation, got %v", recovered)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled mutex waiter remained blocked")
	}
	if waiterRan.Load() {
		t.Fatal("cancelled mutex waiter executed its callback")
	}

	close(releaseHolder)
	select {
	case <-holderDone:
	case <-time.After(time.Second):
		t.Fatal("mutex holder did not finish")
	}
}
