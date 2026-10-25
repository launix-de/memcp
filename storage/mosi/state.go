/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

// Package mosi implements resource coherence independently of routing, transport,
// persistence backends and SQL. Callers install complete data before publishing RAM-only permissions.
package mosi

import "fmt"
import "sync/atomic"
import "github.com/google/uuid"

// NetworkError is the shared protocol failure marker; it grants no fallback authority.
const NetworkError = "Cluster network error"

// Phase is a coherence permission, independent of RAM materialization.
type Phase uint8

const (
	IO Phase = iota
	M
	O
	S
	IS
	MR
	OR
	SR
)

func (phase Phase) String() string {
	switch phase {
	case IO:
		return "IO"
	case M:
		return "M"
	case O:
		return "O"
	case S:
		return "S"
	case IS:
		return "IS"
	case MR:
		return "MR"
	case OR:
		return "OR"
	case SR:
		return "SR"
	}
	panic("invalid MOSI state")
}

func ParsePhase(value string) Phase {
	// Old journals used I only for remote acquisition, never backend authority.
	if value == "I" {
		return IO
	}
	for phase := IO; phase <= SR; phase++ {
		if phase.String() == value {
			return phase
		}
	}
	panic("invalid MOSI state: " + value)
}

func (phase Phase) Readable() bool  { return phase == M || phase == O || phase == S }
func (phase Phase) Requested() bool { return phase >= MR && phase <= SR }

// Flight coordinates one request; upgrades retain its identity and waiters.
type Flight struct {
	id     string
	done   chan struct{}
	serial atomic.Uint64 // assigned by the transport before first enqueue; scoped to its process incarnation
	// The transition owner writes err once before closing done.
	err error
}

// Publication is immutable. The flight belongs to that requested
// state, including its OR/SR -> MR successors; it is not a second authority.
type Publication struct {
	phase      Phase
	prior      Phase // prior readable RAM permission, unless explicitly invalidated
	backend    Phase // immutable RAM permission proving an IS generation is current
	flight     *Flight
	installing bool
}

var stable = [...]Publication{{phase: IO}, {phase: M}, {phase: O}, {phase: S}, {phase: IS}}

// State is embedded in each independently coherent schema or shard.
// Authorized reads perform one atomic load, without registration or CAS.
type State struct {
	value atomic.Pointer[Publication]
}

func (state *State) Snapshot() *Publication {
	if current := state.value.Load(); current != nil {
		return current
	}
	return &stable[IO]
}

func (state *State) Phase() Phase { return state.Snapshot().phase }

// Initialize is restricted to unpublished construction and recovery from a live peer.
// A requested state is never a recoverable authority after a process restart.
func (state *State) Initialize(phase Phase) {
	if phase.Requested() {
		phase = IO
	}
	if phase == IS {
		panic("IS requires established backend authority")
	}
	state.value.Store(&stable[phase])
}

// Establish installs recovered live-peer authority only before any local
// request or permission was published. A late reconnect reply cannot overwrite
// a grant, invalidation, or an in-flight request on the new connection.
func (state *State) Establish(phase Phase) bool {
	if phase.Requested() || phase == IS {
		panic("recovery requires an established permission")
	}
	return state.value.CompareAndSwap(nil, &stable[phase])
}

// Backend-only generations are always current. The established RAM permission is
// retained so loading a shared copy cannot accidentally grant write ownership.
// A backup belonging to a remote modifier always restores as IO, even if files
// are present locally. COLD remains the independent RAM materialization state.
func (state *State) RestoreBackend(phase Phase) {
	if phase.Readable() {
		state.value.Store(&Publication{phase: IS, backend: phase})
		return
	}
	state.Initialize(phase)
}

// The existing eviction owner has verified that no deltas remain and that the
// current persisted generation can be reloaded. Never resurrect IO or a pending
// acquisition when eviction crosses a network invalidation.
func (state *State) BackendOnly() {
	for {
		current := state.Snapshot()
		if !current.phase.Readable() {
			return
		}
		next := &Publication{phase: IS, backend: current.phase}
		if state.value.CompareAndSwap(current, next) {
			return
		}
	}
}

type Action uint8

const (
	Ready Action = iota
	Wait
	Send
	Upgrade
)

// request linearizes both request suppression and OR/SR -> MR upgrades. The
// sender uses the flight ID on every packet; a crossed upgrade dominates an
// older read request with that ID at the directory as well as at the receiver.
func (state *State) Request(wanted Phase) (*Publication, Action) {
	if wanted != M && wanted != O && wanted != S {
		panic("invalid MOSI request")
	}
	for {
		old := state.value.Load()
		current := old
		if current == nil {
			current = &stable[IO]
		}
		if current.phase == IS {
			if !current.backend.Readable() {
				panic("IS without current backend authority")
			}
			// This only restores an already established permission. Actual payload
			// loading uses the existing local COLD lifecycle after acquisition.
			if !state.value.CompareAndSwap(old, &stable[current.backend]) {
				continue
			}
			continue
		}
		if current.phase == M || (wanted != M && current.phase.Readable()) {
			return current, Ready
		}
		if current.phase.Requested() {
			if wanted == M && current.phase != MR && !current.installing {
				next := &Publication{phase: MR, flight: current.flight, prior: current.prior}
				if !state.value.CompareAndSwap(old, next) {
					continue
				}
				return next, Upgrade
			}
			return current, Wait
		}
		phase := SR
		if wanted == M {
			phase = MR
		} else if wanted == O {
			phase = OR
		}
		next := &Publication{phase: phase, prior: current.phase, flight: &Flight{id: uuid.NewString(), done: make(chan struct{})}}
		if state.value.CompareAndSwap(old, next) {
			return next, Send
		}
	}
}

// grant is called by the reader after the last payload packet has been
// installed. Complete data installation precedes visibility and waking any waiter. Claiming
// installation closes the race with a concurrent upgrade; its caller retries
// after this grant if the stronger request did not linearize first.
func (state *State) Grant(id string, phase Phase, install func() error) (bool, error) {
	if !phase.Readable() {
		return false, fmt.Errorf("invalid MOSI grant")
	}
	for {
		current := state.Snapshot()
		if !current.phase.Requested() || current.flight.id != id || current.installing {
			return false, nil
		}
		if current.phase == MR && phase != M {
			return false, nil
		}
		claimed := &Publication{phase: current.phase, flight: current.flight, installing: true}
		if !state.value.CompareAndSwap(current, claimed) {
			continue
		}
		err := func() (err error) {
			defer func() {
				if failure := recover(); failure != nil {
					err = fmt.Errorf("MOSI publication: %v", failure)
				}
			}()
			return install()
		}()
		if err != nil {
			current.flight.err = err
			state.value.Store(&stable[IO])
		} else {
			state.value.Store(&stable[phase])
		}
		close(current.flight.done)
		return true, err
	}
}

func (state *State) Fail(id string, err error) {
	for {
		current := state.Snapshot()
		if !current.phase.Requested() || current.flight.id != id || current.installing {
			return
		}
		restore := IO
		if current.prior.Readable() {
			restore = current.prior
		}
		if state.value.CompareAndSwap(current, &stable[restore]) {
			current.flight.err = err
			close(current.flight.done)
			return
		}
	}
}

// publish is a directory-controlled revocation/downgrade, after write handover
// and local RAM state publication. Requested states already have I semantics;
// an unrelated revocation must not erase a queued acquisition.
func (state *State) Publish(phase Phase) {
	if phase.Requested() || phase == IS {
		panic("requested/IS state requires its authority publication")
	}
	for {
		current := state.value.Load()
		if current != nil && current.installing {
			<-current.flight.done
			continue
		}
		if current != nil && current.phase.Requested() {
			prior := current.prior
			if phase == IO {
				prior = IO
			} else if prior.Readable() {
				prior = phase
			}
			if prior == current.prior {
				return
			}
			next := &Publication{phase: current.phase, flight: current.flight, prior: prior}
			if !state.value.CompareAndSwap(current, next) {
				continue
			}
			return
		}
		if !state.value.CompareAndSwap(current, &stable[phase]) {
			continue
		}
		if current != nil && current.flight != nil {
			current.flight.err = fmt.Errorf(NetworkError)
			close(current.flight.done)
		}
		return
	}
}

// Initialized reports whether construction or recovery has published a state.
// A zero State has IO semantics but has not yet established ownership.
func (state *State) Initialized() bool { return state.value.Load() != nil }

// Phase returns the immutable publication's permission or pending request phase.
func (publication *Publication) Phase() Phase { return publication.phase }

// Backend returns the RAM permission retained by an IS publication.
func (publication *Publication) Backend() Phase { return publication.backend }

// Retained is a pending upgrade's unrevoked prior read permission. It is a
// reconnect proof, never query admission: requested phases still behave as IO.
func (publication *Publication) Retained() Phase { return publication.prior }

// Flight returns the request shared by waiters, or nil for stable permissions.
func (publication *Publication) Flight() *Flight { return publication.flight }

// ID identifies a request and remains unchanged across read-to-write upgrades.
func (flight *Flight) ID() string { return flight.id }

// Done closes after publication or failure; it synchronizes subsequent Err reads.
func (flight *Flight) Done() <-chan struct{} { return flight.done }

// Err must only be read after Done closes.
func (flight *Flight) Err() error { return flight.err }

// Sequence is assigned by the caller before enqueue, within one process incarnation.
func (flight *Flight) Sequence() uint64 { return flight.serial.Load() }

// SetSequence requires the caller's request-sequence ownership.
func (flight *Flight) SetSequence(sequence uint64) { flight.serial.Store(sequence) }
