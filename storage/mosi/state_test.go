/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package mosi_test

import "fmt"
import "sync"
import "testing"
import "sync/atomic"
import "github.com/launix-de/memcp/storage/mosi"

func TestMosiIndependentResources(t *testing.T) {
	var cluster, replication mosi.State
	outer, action := cluster.Request(mosi.M)
	if action != mosi.Send {
		t.Fatal("cluster resource did not start its own request")
	}
	inner, action := replication.Request(mosi.M)
	if action != mosi.Send || inner.Flight().ID() == outer.Flight().ID() {
		t.Fatal("independent resources shared a request identity")
	}
	if accepted, err := replication.Grant(outer.Flight().ID(), mosi.M, func() error {
		t.Fatal("another resource's response reached installation")
		return nil
	}); accepted || err != nil {
		t.Fatal("another resource's grant was accepted")
	}
	if accepted, err := cluster.Grant(outer.Flight().ID(), mosi.M, func() error { return nil }); !accepted || err != nil {
		t.Fatal("cluster grant was not accepted", err)
	}
	if cluster.Phase() != mosi.M || replication.Phase() != mosi.MR || outer.Phase() != mosi.MR {
		t.Fatal("one resource's publication changed another request or its immutable snapshot")
	}
	replication.Fail(inner.Flight().ID(), fmt.Errorf(mosi.NetworkError))
	<-inner.Flight().Done()
	if cluster.Phase() != mosi.M || inner.Flight().Err() == nil {
		t.Fatal("one resource's failure affected another resource")
	}
}

func TestMosiConcurrentReadersSendOnce(t *testing.T) {
	var state mosi.State
	if state.Initialized() || state.Snapshot().Phase() != mosi.IO || state.Initialized() {
		t.Fatal("reading a zero state established authority")
	}
	var sent atomic.Int32
	flights := make(chan *mosi.Flight, 128)
	var readers sync.WaitGroup
	for i := 0; i < cap(flights); i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			publication, action := state.Request(mosi.S)
			if action == mosi.Send {
				sent.Add(1)
			} else if action != mosi.Wait {
				t.Errorf("unexpected action %d", action)
			}
			flights <- publication.Flight()
		}()
	}
	readers.Wait()
	close(flights)
	if sent.Load() != 1 || state.Phase() != mosi.SR {
		t.Fatalf("sends=%d state=%s", sent.Load(), state.Phase())
	}
	current := state.Snapshot()
	current.Flight().SetSequence(42)
	for flight := range flights {
		if flight != current.Flight() || flight.Sequence() != 42 {
			t.Fatal("readers do not share the resource's request")
		}
	}
	if current.Phase().Readable() {
		t.Fatal("SR conferred a read right")
	}
	installed := false
	if accepted, err := state.Grant(current.Flight().ID(), mosi.S, func() error {
		select {
		case <-current.Flight().Done():
			t.Fatal("reader released before installation")
		default:
		}
		if state.Phase().Readable() {
			t.Fatal("rights published before durable installation")
		}
		installed = true
		return nil
	}); !accepted || err != nil {
		t.Fatalf("grant: %v %v", accepted, err)
	}
	<-current.Flight().Done()
	if !installed || state.Phase() != mosi.S {
		t.Fatal("missing installed shared generation")
	}
}

func TestMosiReadUpgradeRejectsDelayedGrant(t *testing.T) {
	for _, wanted := range []mosi.Phase{mosi.O, mosi.S} {
		t.Run(wanted.String(), func(t *testing.T) {
			var state mosi.State
			read, action := state.Request(wanted)
			if action != mosi.Send {
				t.Fatal(action)
			}
			write, action := state.Request(mosi.M)
			if action != mosi.Upgrade || write.Phase() != mosi.MR || write.Flight() != read.Flight() {
				t.Fatal("read request was not upgraded in place")
			}
			for i := 0; i < 64; i++ {
				if _, action := state.Request(mosi.M); action != mosi.Wait {
					t.Fatal("duplicate M request")
				}
				if _, action := state.Request(mosi.S); action != mosi.Wait {
					t.Fatal("MR granted read access")
				}
			}
			accepted, err := state.Grant(read.Flight().ID(), wanted, func() error { t.Fatal("stale read grant persisted"); return nil })
			if accepted || err != nil || state.Phase() != mosi.MR {
				t.Fatal("read grant undid MR")
			}
			select {
			case <-write.Flight().Done():
				t.Fatal("read grant released M waiter")
			default:
			}
			if accepted, err := state.Grant(write.Flight().ID(), mosi.M, func() error { return nil }); !accepted || err != nil {
				t.Fatal(accepted, err)
			}
			<-write.Flight().Done()
			if state.Phase() != mosi.M {
				t.Fatal("missing M")
			}
			if accepted, _ := state.Grant(read.Flight().ID(), wanted, func() error { t.Fatal("duplicate grant persisted"); return nil }); accepted {
				t.Fatal("late grant accepted")
			}
		})
	}
}

func TestMosiAuthorizedReadsDoNotPublish(t *testing.T) {
	for _, phase := range []mosi.Phase{mosi.M, mosi.O, mosi.S} {
		var state mosi.State
		state.Initialize(phase)
		before := state.Snapshot()
		var readers sync.WaitGroup
		for i := 0; i < 32; i++ {
			readers.Add(1)
			go func() {
				defer readers.Done()
				for j := 0; j < 1000; j++ {
					if _, action := state.Request(mosi.S); action != mosi.Ready {
						t.Error("read requested additional rights")
						return
					}
				}
			}()
		}
		readers.Wait()
		if state.Snapshot() != before {
			t.Fatal("authorized readers republished state")
		}
	}
}

func TestMosiFailureAndRecovery(t *testing.T) {
	var state mosi.State
	first, _ := state.Request(mosi.S)
	state.Fail(first.Flight().ID(), fmt.Errorf(mosi.NetworkError))
	<-first.Flight().Done()
	if first.Flight().Err() == nil || state.Phase() != mosi.IO {
		t.Fatal("network failure did not complete request")
	}
	second, _ := state.Request(mosi.M)
	if second.Flight().ID() == first.Flight().ID() {
		t.Fatal("request identity reused")
	}
	if accepted, _ := state.Grant(first.Flight().ID(), mosi.M, func() error { t.Fatal("old response persisted"); return nil }); accepted {
		t.Fatal("old response accepted")
	}
	accepted, err := state.Grant(second.Flight().ID(), mosi.M, func() error { return fmt.Errorf("disk full") })
	if !accepted || err == nil || state.Phase() != mosi.IO {
		t.Fatal("failed installation conferred rights")
	}
	<-second.Flight().Done()
	if second.Flight().Err() == nil {
		t.Fatal("persistence error not propagated")
	}
	state.Initialize(mosi.MR)
	if state.Phase() != mosi.IO {
		t.Fatal("recovered pending request as authority")
	}
}

// IS has a current backend generation with an already durable permission.
// Its first access restores that permission locally; mosi.IO must acquire remotely.
func TestMosiBackendAuthority(t *testing.T) {
	for _, durable := range []mosi.Phase{mosi.M, mosi.O, mosi.S} {
		t.Run(durable.String(), func(t *testing.T) {
			var state mosi.State
			state.RestoreBackend(durable)
			if state.Phase() != mosi.IS {
				t.Fatal("backend generation did not restore as IS")
			}
			publication, action := state.Request(mosi.S)
			if action != mosi.Ready || publication.Phase() != durable || publication.Flight() != nil {
				t.Fatal("current backend read attempted network acquisition")
			}
			before := state.Snapshot()
			for i := 0; i < 100; i++ {
				state.Request(mosi.S)
			}
			if state.Snapshot() != before {
				t.Fatal("warm reads wrote coherence state")
			}
			_, action = state.Request(mosi.M)
			if durable == mosi.M {
				if action != mosi.Ready {
					t.Fatal("durable M lost its offline write right")
				}
			} else if action != mosi.Send || state.Phase() != mosi.MR {
				t.Fatal("loading shared backend data invented exclusive ownership")
			}
		})
	}
	for _, value := range []string{"IO", "I", "MR", "OR", "SR"} {
		var state mosi.State
		state.RestoreBackend(mosi.ParsePhase(value))
		if state.Phase() != mosi.IO {
			t.Fatalf("%s recovered a backend authority", value)
		}
		if _, action := state.Request(mosi.S); action != mosi.Send {
			t.Fatal("IO did not acquire from owner")
		}
	}
}

func TestMosiInvalidationRemovesBackendAuthority(t *testing.T) {
	var state mosi.State
	state.RestoreBackend(mosi.M)
	state.Publish(mosi.IO)
	publication, action := state.Request(mosi.S)
	if action != mosi.Send || publication.Phase() != mosi.SR {
		t.Fatal("revoked backend generation remained locally readable")
	}
	state.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError))
	if state.Phase() != mosi.IO {
		t.Fatal("unreachable owner fell back to a backup")
	}
}

func TestMosiBackendEvictionDoesNotResurrectRevokedRights(t *testing.T) {
	for _, phase := range []mosi.Phase{mosi.M, mosi.O, mosi.S} {
		var state mosi.State
		state.Initialize(phase)
		state.BackendOnly()
		if state.Phase() != mosi.IS || state.Snapshot().Backend() != phase {
			t.Fatal("eviction lost backend permission")
		}
		state.Publish(mosi.IO)
		state.BackendOnly()
		if state.Phase() != mosi.IO {
			t.Fatal("eviction resurrected an invalidated backup")
		}
		request, _ := state.Request(mosi.M)
		state.BackendOnly()
		if state.Snapshot() != request {
			t.Fatal("eviction erased pending acquisition")
		}
	}
}
