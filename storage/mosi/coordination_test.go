/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package mosi

import "context"
import "runtime"
import "testing"
import "time"

func waitCoordination(t *testing.T, state *State, previous *Coordination) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for state.coordination.Load() == previous {
		if time.Now().After(deadline) {
			t.Fatal("request did not enqueue")
		}
		runtime.Gosched()
	}
}

func TestCoordinationFIFOAndIndependentResources(t *testing.T) {
	var state, other State
	state.Initialize(M)
	first := state.TryCoordinate()
	second := make(chan *Coordination, 1)
	go func() { claim, _ := state.Coordinate(context.Background()); second <- claim }()
	waitCoordination(t, &state, first)
	secondTail := state.coordination.Load()
	third := make(chan *Coordination, 1)
	go func() { claim, _ := state.Coordinate(context.Background()); third <- claim }()
	waitCoordination(t, &state, secondTail)
	if state.TryCoordinate() != nil {
		t.Fatal("local request overtook a queued remote transition")
	}
	independent := other.TryCoordinate()
	if independent == nil || !state.ReadReady() {
		t.Fatal("coordination blocked another item or an authorized reader")
	}
	independent.Release()
	first.Release()
	select {
	case claim := <-second:
		select {
		case <-third:
			t.Fatal("third request overtook second")
		default:
		}
		claim.Release()
	case <-time.After(time.Second):
		t.Fatal("second request starved")
	}
	select {
	case claim := <-third:
		claim.Release()
	case <-time.After(time.Second):
		t.Fatal("third request starved")
	}
	last := state.TryCoordinate()
	if last == nil {
		t.Fatal("last transition did not release queue")
	}
	last.Release()
}

func TestCoordinationCancellationPreservesQueue(t *testing.T) {
	var state State
	first := state.TryCoordinate()
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan error, 1)
	go func() { _, err := state.Coordinate(ctx); cancelled <- err }()
	waitCoordination(t, &state, first)
	tail := state.coordination.Load()
	last := make(chan *Coordination, 1)
	go func() { claim, _ := state.Coordinate(context.Background()); last <- claim }()
	waitCoordination(t, &state, tail)
	cancel()
	if err := <-cancelled; err != context.Canceled {
		t.Fatal(err)
	}
	select {
	case <-last:
		t.Fatal("cancelled waiter detached the active predecessor")
	default:
	}
	first.Release()
	select {
	case claim := <-last:
		claim.Release()
	case <-time.After(time.Second):
		t.Fatal("cancelled waiter stranded successor")
	}
	claim := state.TryCoordinate()
	if claim == nil {
		t.Fatal("queue retained a cancelled request")
	}
	claim.Release()
}
