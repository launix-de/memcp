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
import "sync/atomic"
import "github.com/google/uuid"

// These are coherence states, independent of COLD/SHARED/WRITE, which describe
// local materialization. Requested states never confer a read or write right.
type mosiPhase uint8

const (
	mosiIO mosiPhase = iota
	mosiM
	mosiO
	mosiS
	mosiIS
	mosiMR
	mosiOR
	mosiSR
)

func (phase mosiPhase) String() string {
	switch phase {
	case mosiIO:
		return "IO"
	case mosiM:
		return "M"
	case mosiO:
		return "O"
	case mosiS:
		return "S"
	case mosiIS:
		return "IS"
	case mosiMR:
		return "MR"
	case mosiOR:
		return "OR"
	case mosiSR:
		return "SR"
	}
	panic("invalid MOSI state")
}

func parseMosiPhase(value string) mosiPhase {
	// Old journals used I only for remote acquisition, never backend authority.
	if value == "I" {
		return mosiIO
	}
	for phase := mosiIO; phase <= mosiSR; phase++ {
		if phase.String() == value {
			return phase
		}
	}
	panic("invalid MOSI state: " + value)
}

func (phase mosiPhase) readable() bool  { return phase == mosiM || phase == mosiO || phase == mosiS }
func (phase mosiPhase) requested() bool { return phase >= mosiMR && phase <= mosiSR }

type mosiFlight struct {
	id     string
	done   chan struct{}
	serial atomic.Uint64 // assigned durably by the transport before first enqueue
	// The transition owner writes err once before closing done.
	err error
}

// A state publication is immutable. The flight belongs to that requested
// state, including its OR/SR -> MR successors; it is not a second authority.
type mosiPublication struct {
	phase      mosiPhase
	backend    mosiPhase // immutable durable permission proving an IS generation is current
	flight     *mosiFlight
	installing bool
}

var mosiStable = [...]mosiPublication{{phase: mosiIO}, {phase: mosiM}, {phase: mosiO}, {phase: mosiS}, {phase: mosiIS}}

// mosiState is embedded in each independently coherent schema or shard.
// Authorized reads perform one atomic load, without registration or CAS.
type mosiState struct {
	value atomic.Pointer[mosiPublication]
}

func (state *mosiState) snapshot() *mosiPublication {
	if current := state.value.Load(); current != nil {
		return current
	}
	return &mosiStable[mosiIO]
}

func (state *mosiState) phase() mosiPhase { return state.snapshot().phase }

// initialize is restricted to unpublished construction and durable recovery.
// A requested state is never a recoverable authority after a process restart.
func (state *mosiState) initialize(phase mosiPhase) {
	if phase.requested() {
		phase = mosiIO
	}
	if phase == mosiIS {
		panic("IS requires durable backend authority")
	}
	state.value.Store(&mosiStable[phase])
}

// Backend-only generations are always current. The durable permission is
// retained so loading a shared copy cannot accidentally grant write ownership.
// A backup belonging to a remote modifier always restores as IO, even if files
// are present locally. COLD remains the independent RAM materialization state.
func (state *mosiState) restoreBackend(phase mosiPhase) {
	if phase.readable() {
		state.value.Store(&mosiPublication{phase: mosiIS, backend: phase})
		return
	}
	state.initialize(phase)
}

// The existing eviction owner has verified that no deltas remain and that the
// current persisted generation can be reloaded. Never resurrect IO or a pending
// acquisition when eviction crosses a network invalidation.
func (state *mosiState) backendOnly() {
	for {
		current := state.snapshot()
		if !current.phase.readable() {
			return
		}
		next := &mosiPublication{phase: mosiIS, backend: current.phase}
		if state.value.CompareAndSwap(current, next) {
			return
		}
	}
}

type mosiAction uint8

const (
	mosiReady mosiAction = iota
	mosiWait
	mosiSend
	mosiUpgrade
)

// request linearizes both request suppression and OR/SR -> MR upgrades. The
// sender uses the flight ID on every packet; a crossed upgrade dominates an
// older read request with that ID at the directory as well as at the receiver.
func (state *mosiState) request(wanted mosiPhase) (*mosiPublication, mosiAction) {
	if wanted != mosiM && wanted != mosiO && wanted != mosiS {
		panic("invalid MOSI request")
	}
	for {
		old := state.value.Load()
		current := old
		if current == nil {
			current = &mosiStable[mosiIO]
		}
		if current.phase == mosiIS {
			if !current.backend.readable() {
				panic("IS without current backend authority")
			}
			// This only restores an already durable permission. Actual payload
			// loading uses the existing local COLD lifecycle after acquisition.
			if !state.value.CompareAndSwap(old, &mosiStable[current.backend]) {
				continue
			}
			continue
		}
		if current.phase == mosiM || (wanted != mosiM && current.phase.readable()) {
			return current, mosiReady
		}
		if current.phase.requested() {
			if wanted == mosiM && current.phase != mosiMR && !current.installing {
				next := &mosiPublication{phase: mosiMR, flight: current.flight}
				if !state.value.CompareAndSwap(old, next) {
					continue
				}
				return next, mosiUpgrade
			}
			return current, mosiWait
		}
		phase := mosiSR
		if wanted == mosiM {
			phase = mosiMR
		} else if wanted == mosiO {
			phase = mosiOR
		}
		next := &mosiPublication{phase: phase, flight: &mosiFlight{id: uuid.NewString(), done: make(chan struct{})}}
		if state.value.CompareAndSwap(old, next) {
			return next, mosiSend
		}
	}
}

// grant is called by the reader after the last payload packet has been
// installed. Persistence precedes visibility and waking any waiter. Claiming
// installation closes the race with a concurrent upgrade; its caller retries
// after this grant if the stronger request did not linearize first.
func (state *mosiState) grant(id string, phase mosiPhase, persist func() error) (bool, error) {
	if !phase.readable() {
		return false, fmt.Errorf("invalid MOSI grant")
	}
	for {
		current := state.snapshot()
		if !current.phase.requested() || current.flight.id != id || current.installing {
			return false, nil
		}
		if current.phase == mosiMR && phase != mosiM {
			return false, nil
		}
		claimed := &mosiPublication{phase: current.phase, flight: current.flight, installing: true}
		if !state.value.CompareAndSwap(current, claimed) {
			continue
		}
		err := func() (err error) {
			defer func() {
				if failure := recover(); failure != nil {
					err = fmt.Errorf("MOSI publication: %v", failure)
				}
			}()
			return persist()
		}()
		if err != nil {
			current.flight.err = err
			state.value.Store(&mosiStable[mosiIO])
		} else {
			state.value.Store(&mosiStable[phase])
		}
		close(current.flight.done)
		return true, err
	}
}

func (state *mosiState) fail(id string, err error) {
	for {
		current := state.snapshot()
		if !current.phase.requested() || current.flight.id != id || current.installing {
			return
		}
		if state.value.CompareAndSwap(current, &mosiStable[mosiIO]) {
			current.flight.err = err
			close(current.flight.done)
			return
		}
	}
}

// publish is a directory-controlled revocation/downgrade, after write handover
// and durable state publication. Requested states already have I semantics;
// an unrelated revocation must not erase a queued acquisition.
func (state *mosiState) publish(phase mosiPhase) {
	if phase.requested() || phase == mosiIS {
		panic("requested/IS state requires its authority publication")
	}
	for {
		current := state.value.Load()
		if current != nil && current.installing {
			<-current.flight.done
			continue
		}
		if current != nil && current.phase.requested() {
			return
		}
		if !state.value.CompareAndSwap(current, &mosiStable[phase]) {
			continue
		}
		if current != nil && current.flight != nil {
			current.flight.err = fmt.Errorf(replicationNetworkError)
			close(current.flight.done)
		}
		return
	}
}
