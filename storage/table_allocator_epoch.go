/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "math"
import "strconv"
import "crypto/sha256"
import "encoding/json"

// A reservation carries actual signed endpoints. A predecessor of MinInt64
// does not exist; generation must never depend on unsigned wraparound.
type tableIDReservation struct {
	First      int64
	Last       int64
	LegacyBase uint64
}

type tableIDExplicit struct {
	High     uint64 // historical unsigned maintenance inputs remain readable
	Advance  bool   // an explicitly supplied native highwater, including zero
	Value    int64
	Observed bool
}

func tableAllocatorToken(state tableAllocatorState) string {
	encoded, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func legacyTableIDReservation(base, count uint64) tableIDReservation {
	r := tableIDReservation{LegacyBase: base}
	if count != 0 {
		r.First, r.Last = int64(base+1), int64(base+count)
	}
	return r
}

func epochTableIDReservation(previous tableAllocatorState, input tableIDExplicit, count uint64) (tableIDReservation, tableAllocatorState, bool) {
	if input.High > math.MaxInt64 {
		panic("native unsigned allocator input exceeds epoch capacity")
	}
	next := previous
	// Native maintenance's unsigned highwater input still means advancement,
	// not a reset. Its historical raw range cannot become signed generation.
	if !input.Observed && (input.High > 0 || input.Advance) {
		if next.HasNext && int64(input.High) >= next.Next {
			next.Current = int64(input.High)
			if next.Current == math.MaxInt64 {
				next.Next, next.HasNext = 0, false
			} else {
				next.Next = next.Current + 1
			}
			next.IssuedKnown, next.Issued = true, true
		}
		if input.High > next.High {
			next.High, next.HighAllocated = input.High, true
		}
	}
	if input.Observed {
		next.IssuedKnown, next.Issued = true, true
		if next.HasNext && input.Value >= next.Next {
			next.Current = input.Value
			if input.Value == math.MaxInt64 {
				next.Next, next.HasNext = 0, false
			} else {
				next.Next = input.Value + 1
			}
		}
		if input.High > next.High {
			next.High, next.HighAllocated = input.High, true
		}
	}
	r := tableIDReservation{LegacyBase: previous.High}
	if count != 0 {
		if !next.HasNext || count-1 > math.MaxInt64 || next.Next > math.MaxInt64-int64(count-1) {
			panic("generated ID block exceeds native integer capacity")
		}
		r.First, r.Last = next.Next, next.Next+int64(count-1)
		next.Current = r.Last
		if r.Last == math.MaxInt64 {
			next.Next, next.HasNext = 0, false
		} else {
			next.Next = r.Last + 1
		}
		next.IssuedKnown, next.Issued = true, true
		if r.Last > 0 && uint64(r.Last) >= next.High {
			next.High, next.HighAllocated = uint64(r.Last), true
		}
	}
	return r, next, next != previous
}

func allocatorSignedDecimal(text string) int64 {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil || strconv.FormatInt(value, 10) != text {
		panic("table allocator cursor requires canonical signed decimal integers")
	}
	return value
}

// The standalone block-reservation builtin has no row producer. Borrow its
// original transaction's existing construction/decision gate for the duration
// of that batch so a reset/savepoint/commit cannot overtake it. INSERT's actual
// native producer and endpoint admission already provide that exclusion.
func (tx *TxContext) admitAllocatorReservation() func() {
	if tx == nil {
		return func() {}
	}
	scope := tx.resourceScope()
	scope.mu.Lock()
	if scope.closed || scope.deciding || scope.constructing != 0 {
		scope.mu.Unlock()
		panic("native allocator reservation conflicts with transaction construction or decision")
	}
	scope.constructing++
	scope.mu.Unlock()
	return func() { scope.mu.Lock(); scope.constructing--; scope.mu.Unlock() }
}
