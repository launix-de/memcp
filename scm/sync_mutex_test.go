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

// Each producer deliberately reuses its argument array. The merge must retain
// independent rows across batches and apply the window to complete SQL rows.
func TestOrderedProducerMergeOwnsRowsAndPreservesDuplicates(t *testing.T) {
	producers := make([]Scmer, 3)
	for stream := range producers {
		producers[stream] = NewFunc(func(a ...Scmer) Scmer {
			row := []Scmer{NewInt(0)}
			for i := 0; i < 140; i++ {
				row[0] = NewInt(int64(i))
				Apply(a[0], NewSlice(row))
			}
			return NewNil()
		})
	}
	var got []int64
	consumer := NewFunc(func(a ...Scmer) Scmer {
		got = append(got, asSlice(a[0], "row")[0].Int())
		return NewNil()
	})
	scanOrderMerge(NewNil(), NewSlice(producers), NewSlice([]Scmer{NewInt(0)}),
		NewSlice([]Scmer{NewFunc(LessScm)}), NewInt(2), NewInt(207), consumer)
	if len(got) != 207 {
		t.Fatalf("window length = %d", len(got))
	}
	for i, value := range got {
		if value != int64((i+2)/3) {
			t.Fatalf("row %d = %d", i, value)
		}
	}
}

func TestOrderedProducerMergeFailureJoinsEveryProducer(t *testing.T) {
	var finished atomic.Int32
	producer := NewFunc(func(a ...Scmer) Scmer {
		defer finished.Add(1)
		for i := 0; i < 300; i++ {
			Apply(a[0], NewSlice([]Scmer{NewInt(int64(i))}))
		}
		return NewNil()
	})
	consumer := NewFunc(func(_ ...Scmer) Scmer { panic("consumer failure") })
	func() {
		defer func() {
			if failure := recover(); failure != "consumer failure" {
				t.Fatalf("failure = %v", failure)
			}
		}()
		scanOrderMerge(NewNil(), NewSlice([]Scmer{producer, producer}),
			NewSlice([]Scmer{NewInt(0)}), NewSlice([]Scmer{NewFunc(LessScm)}), NewInt(0), NewInt(-1), consumer)
	}()
	if finished.Load() != 2 {
		t.Fatalf("unfinished producer: %d", finished.Load())
	}
}

func TestOrderedProducerMergePropagatesProducerFailure(t *testing.T) {
	defer func() {
		if failure := recover(); failure != "producer failure" {
			t.Fatalf("failure = %v", failure)
		}
	}()
	producer := NewFunc(func(_ ...Scmer) Scmer { panic("producer failure") })
	scanOrderMerge(NewNil(), NewSlice([]Scmer{producer}), NewSlice([]Scmer{NewInt(0)}),
		NewSlice([]Scmer{NewFunc(LessScm)}), NewInt(0), NewInt(-1), NewFunc(func(_ ...Scmer) Scmer { return NewNil() }))
}

func TestOrderedProducerMergeRejectsInvalidInput(t *testing.T) {
	for _, positions := range [][]Scmer{{NewInt(-1)}, {NewInt(1)}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid position accepted")
				}
			}()
			producer := NewFunc(func(a ...Scmer) Scmer {
				return Apply(a[0], NewSlice([]Scmer{NewInt(0)}))
			})
			scanOrderMerge(NewNil(), NewSlice([]Scmer{producer}), NewSlice(positions),
				NewSlice([]Scmer{NewFunc(LessScm)}), NewInt(0), NewInt(-1),
				NewFunc(func(_ ...Scmer) Scmer { return NewNil() }))
		}()
	}
}

func TestOrderedProducerMergeCancelledQueryJoinsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ss := &SessionState{}
	seq := ss.BeginQuery("Query", "ordered merge cancellation")
	ss.SetQueryContext(seq, ctx)
	defer ss.EndQuery(seq, "Sleep", "")
	var finished atomic.Bool
	producer := NewFunc(func(a ...Scmer) Scmer {
		defer finished.Store(true)
		for i := 0; i < 1000; i++ {
			if !ToBool(Apply(a[0], NewSlice([]Scmer{NewInt(int64(i))}))) {
				break
			}
		}
		return NewNil()
	})
	consumer := NewFunc(func(_ ...Scmer) Scmer { cancel(); return NewNil() })
	func() {
		defer func() {
			if failure := recover(); failure != context.Canceled {
				t.Fatalf("expected cancellation, got %v", failure)
			}
		}()
		scanOrderMerge(NewAny(&mutexTestTransaction{ss: ss, seq: seq}),
			NewSlice([]Scmer{producer}), NewSlice([]Scmer{NewInt(0)}),
			NewSlice([]Scmer{NewFunc(LessScm)}), NewInt(0), NewInt(-1), consumer)
	}()
	if !finished.Load() {
		t.Fatal("cancelled merge left its producer running")
	}
}
