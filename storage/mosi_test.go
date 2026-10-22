/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "fmt"
import "sync"
import "testing"
import "sync/atomic"

func TestMosiConcurrentReadersSendOnce(t *testing.T) {
	var state mosiState
	var sent atomic.Int32
	flights := make(chan *mosiFlight, 128)
	var readers sync.WaitGroup
	for i := 0; i < cap(flights); i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			publication, action := state.request(mosiS)
			if action == mosiSend {
				sent.Add(1)
			} else if action != mosiWait {
				t.Errorf("unexpected action %d", action)
			}
			flights <- publication.flight
		}()
	}
	readers.Wait()
	close(flights)
	if sent.Load() != 1 || state.phase() != mosiSR {
		t.Fatalf("sends=%d state=%s", sent.Load(), state.phase())
	}
	current := state.snapshot()
	for flight := range flights {
		if flight != current.flight {
			t.Fatal("readers do not share the resource's request")
		}
	}
	if current.phase.readable() {
		t.Fatal("SR conferred a read right")
	}
	installed := false
	if accepted, err := state.grant(current.flight.id, mosiS, func() error {
		select {
		case <-current.flight.done:
			t.Fatal("reader released before installation")
		default:
		}
		if state.phase().readable() {
			t.Fatal("rights published before durable installation")
		}
		installed = true
		return nil
	}); !accepted || err != nil {
		t.Fatalf("grant: %v %v", accepted, err)
	}
	<-current.flight.done
	if !installed || state.phase() != mosiS {
		t.Fatal("missing installed shared generation")
	}
}

func TestMosiReadUpgradeRejectsDelayedGrant(t *testing.T) {
	for _, wanted := range []mosiPhase{mosiO, mosiS} {
		t.Run(wanted.String(), func(t *testing.T) {
			var state mosiState
			read, action := state.request(wanted)
			if action != mosiSend {
				t.Fatal(action)
			}
			write, action := state.request(mosiM)
			if action != mosiUpgrade || write.phase != mosiMR || write.flight != read.flight {
				t.Fatal("read request was not upgraded in place")
			}
			for i := 0; i < 64; i++ {
				if _, action := state.request(mosiM); action != mosiWait {
					t.Fatal("duplicate M request")
				}
				if _, action := state.request(mosiS); action != mosiWait {
					t.Fatal("MR granted read access")
				}
			}
			accepted, err := state.grant(read.flight.id, wanted, func() error { t.Fatal("stale read grant persisted"); return nil })
			if accepted || err != nil || state.phase() != mosiMR {
				t.Fatal("read grant undid MR")
			}
			select {
			case <-write.flight.done:
				t.Fatal("read grant released M waiter")
			default:
			}
			if accepted, err := state.grant(write.flight.id, mosiM, func() error { return nil }); !accepted || err != nil {
				t.Fatal(accepted, err)
			}
			<-write.flight.done
			if state.phase() != mosiM {
				t.Fatal("missing M")
			}
			if accepted, _ := state.grant(read.flight.id, wanted, func() error { t.Fatal("duplicate grant persisted"); return nil }); accepted {
				t.Fatal("late grant accepted")
			}
		})
	}
}

func TestMosiAuthorizedReadsDoNotPublish(t *testing.T) {
	for _, phase := range []mosiPhase{mosiM, mosiO, mosiS} {
		var state mosiState
		state.initialize(phase)
		before := state.value.Load()
		var readers sync.WaitGroup
		for i := 0; i < 32; i++ {
			readers.Add(1)
			go func() {
				defer readers.Done()
				for j := 0; j < 1000; j++ {
					if _, action := state.request(mosiS); action != mosiReady {
						t.Error("read requested additional rights")
						return
					}
				}
			}()
		}
		readers.Wait()
		if state.value.Load() != before {
			t.Fatal("authorized readers republished state")
		}
	}
}

func TestMosiFailureAndRecovery(t *testing.T) {
	var state mosiState
	first, _ := state.request(mosiS)
	state.fail(first.flight.id, fmt.Errorf(replicationNetworkError))
	<-first.flight.done
	if first.flight.err == nil || state.phase() != mosiIO {
		t.Fatal("network failure did not complete request")
	}
	second, _ := state.request(mosiM)
	if second.flight.id == first.flight.id {
		t.Fatal("request identity reused")
	}
	if accepted, _ := state.grant(first.flight.id, mosiM, func() error { t.Fatal("old response persisted"); return nil }); accepted {
		t.Fatal("old response accepted")
	}
	accepted, err := state.grant(second.flight.id, mosiM, func() error { return fmt.Errorf("disk full") })
	if !accepted || err == nil || state.phase() != mosiIO {
		t.Fatal("failed installation conferred rights")
	}
	<-second.flight.done
	if second.flight.err == nil {
		t.Fatal("persistence error not propagated")
	}
	state.initialize(mosiMR)
	if state.phase() != mosiIO {
		t.Fatal("recovered pending request as authority")
	}
}

// IS has a current backend generation with an already durable permission.
// Its first access restores that permission locally; IO must acquire remotely.
func TestMosiBackendAuthority(t *testing.T) {
	for _, durable := range []mosiPhase{mosiM, mosiO, mosiS} {
		t.Run(durable.String(), func(t *testing.T) {
			var state mosiState
			state.restoreBackend(durable)
			if state.phase() != mosiIS {
				t.Fatal("backend generation did not restore as IS")
			}
			publication, action := state.request(mosiS)
			if action != mosiReady || publication.phase != durable || publication.flight != nil {
				t.Fatal("current backend read attempted network acquisition")
			}
			before := state.value.Load()
			for i := 0; i < 100; i++ {
				state.request(mosiS)
			}
			if state.value.Load() != before {
				t.Fatal("warm reads wrote coherence state")
			}
			_, action = state.request(mosiM)
			if durable == mosiM {
				if action != mosiReady {
					t.Fatal("durable M lost its offline write right")
				}
			} else if action != mosiSend || state.phase() != mosiMR {
				t.Fatal("loading shared backend data invented exclusive ownership")
			}
		})
	}
	for _, value := range []string{"IO", "I", "MR", "OR", "SR"} {
		var state mosiState
		state.restoreBackend(parseMosiPhase(value))
		if state.phase() != mosiIO {
			t.Fatalf("%s recovered a backend authority", value)
		}
		if _, action := state.request(mosiS); action != mosiSend {
			t.Fatal("IO did not acquire from owner")
		}
	}
}

func TestMosiInvalidationRemovesBackendAuthority(t *testing.T) {
	var state mosiState
	state.restoreBackend(mosiM)
	state.publish(mosiIO)
	publication, action := state.request(mosiS)
	if action != mosiSend || publication.phase != mosiSR {
		t.Fatal("revoked backend generation remained locally readable")
	}
	state.fail(publication.flight.id, fmt.Errorf(replicationNetworkError))
	if state.phase() != mosiIO {
		t.Fatal("unreachable owner fell back to a backup")
	}
}

func TestMosiBackendEvictionDoesNotResurrectRevokedRights(t *testing.T) {
	for _, phase := range []mosiPhase{mosiM, mosiO, mosiS} {
		var state mosiState
		state.initialize(phase)
		state.backendOnly()
		if state.phase() != mosiIS || state.snapshot().backend != phase {
			t.Fatal("eviction lost backend permission")
		}
		state.publish(mosiIO)
		state.backendOnly()
		if state.phase() != mosiIO {
			t.Fatal("eviction resurrected an invalidated backup")
		}
		request, _ := state.request(mosiM)
		state.backendOnly()
		if state.snapshot() != request {
			t.Fatal("eviction erased pending acquisition")
		}
	}
}
