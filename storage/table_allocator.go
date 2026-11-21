/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "math"
import "io"
import "unsafe"
import "sort"
import "bytes"
import "strconv"
import "strings"
import "encoding/json"
import "crypto/sha256"
import "encoding/base64"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Table allocation is independent of row visibility and of SQL transaction
// decisions. Issued records an actual supplied/generated ID input; HighAllocated
// separately proves that the highwater came from a forward counter reservation. The complete immutable observation has one UUID owner. Historical
// Auto_increment headers carry only High; missing provenance stays unknown.
// Version 1 is also used by the schema field, never by a catalog decision.
type tableAllocatorState struct {
	Version       int    `json:"version"`
	Table         string `json:"table"`
	StartKnown    bool   `json:"start_known"`
	Start         uint64 `json:"start"`
	IssuedKnown   bool   `json:"issued_known"`
	Issued        bool   `json:"issued"`
	HighAllocated bool   `json:"high_allocated"`
	High          uint64 `json:"high"`
}

type LogEntryTableAllocator struct{ state tableAllocatorState }

func (s tableAllocatorState) validate() error {
	id, err := uuid.Parse(s.Table)
	if err != nil || id == uuid.Nil || id.String() != s.Table || s.Version != 1 ||
		(s.StartKnown && (s.Start == 0 || s.Start > math.MaxInt64)) ||
		(!s.StartKnown && s.Start != 0) || (s.Issued && !s.IssuedKnown) ||
		(s.HighAllocated && (!s.IssuedKnown || !s.Issued || s.High == 0 || (s.StartKnown && s.High < s.Start))) ||
		(s.StartKnown && s.High < s.Start-1) ||
		(s.IssuedKnown && !s.Issued && (!s.StartKnown || s.High != s.Start-1)) {
		return fmt.Errorf("invalid native table allocator state")
	}
	return nil
}

// Versioned provenance is required data: reject omitted/null fields rather
// than interpreting zero-value booleans as historical allocator facts.
func (state *tableAllocatorState) UnmarshalJSON(data []byte) error {
	type plain tableAllocatorState
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, name := range []string{"version", "table", "start_known", "start", "issued_known", "issued", "high_allocated", "high"} {
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("missing native allocator provenance field: %s", name)
		}
	}
	var decoded plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*state = tableAllocatorState(decoded)
	return state.validate()
}

func encodeTableAllocatorRecord(entry LogEntryTableAllocator) string {
	if err := entry.state.validate(); err != nil {
		panic(err)
	}
	raw, err := json.Marshal(entry.state)
	if err != nil {
		panic(err)
	}
	checksum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + fmt.Sprintf("%x", checksum[:])
}

func decodeTableAllocatorRecord(encoded string) LogEntryTableAllocator {
	payload, checksum, found := strings.Cut(encoded, ".")
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if !found || err != nil || strings.Contains(checksum, ".") {
		panic("invalid native table allocator frame")
	}
	actual := sha256.Sum256(raw)
	if checksum != fmt.Sprintf("%x", actual[:]) {
		panic("corrupt native table allocator frame")
	}
	var state tableAllocatorState
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		panic("invalid native table allocator record")
	}
	if err := state.validate(); err != nil {
		panic(err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		panic("trailing native allocator frame payload")
	}
	return LogEntryTableAllocator{state}
}

// Recovery combines actual monotone facts, not wall-clock order or a second
// sequence counter. A table UUID has one immutable configured start. An issued
// high is proven only by an actual reservation record at that high; raising a
// legacy/advisory high alone never manufactures that proof.
func mergeTableAllocator(previous, incoming tableAllocatorState) (tableAllocatorState, error) {
	if err := previous.validate(); err != nil {
		return previous, err
	}
	if err := incoming.validate(); err != nil {
		return previous, err
	}
	if previous.Table != incoming.Table || (previous.StartKnown && incoming.StartKnown && previous.Start != incoming.Start) {
		return previous, fmt.Errorf("conflicting native table allocator provenance")
	}
	next := previous
	if !next.StartKnown && incoming.StartKnown {
		next.StartKnown, next.Start = true, incoming.Start
	}
	if incoming.High > previous.High {
		next.High, next.HighAllocated = incoming.High, incoming.HighAllocated
	} else if incoming.High == previous.High {
		next.HighAllocated = previous.HighAllocated || incoming.HighAllocated
	}
	next.Issued = previous.Issued || incoming.Issued
	next.IssuedKnown = next.Issued || (previous.IssuedKnown && incoming.IssuedKnown)
	if err := next.validate(); err != nil {
		return previous, err
	}
	return next, nil
}

// All callers own transactionMu or the unpublished authority-log loader.
func (db *database) acceptTableAllocator(state tableAllocatorState) error {
	if err := state.validate(); err != nil {
		return err
	}
	if !state.Issued {
		return fmt.Errorf("table allocator WAL must record an actual reservation")
	}
	if db.tableAllocators == nil {
		db.tableAllocators = make(map[string]tableAllocatorState)
	}
	previous, exists := db.tableAllocators[state.Table]
	if exists {
		var err error
		state, err = mergeTableAllocator(previous, state)
		if err != nil {
			return err
		}
	} else {
		db.adjustMetadataMemory(0, tableAllocatorRecordMemory(state))
	}
	db.tableAllocators[state.Table] = state
	return nil
}

func tableAllocatorRecordMemory(state tableAllocatorState) uint {
	return uint(unsafe.Sizeof(state)) + 64 + uint(len(state.Table))
}

func (db *database) tableAllocatorLogEntries() []interface{} {
	// Inventory belongs to the existing cold checkpoint/MOVE owner. A private
	// CREATE carries its current allocator in its eventual catalog envelope;
	// before acceptance there is no durable table whose allocator must survive.
	// Retired UUIDs therefore need no tombstone or ever-growing second catalog.
	live := make(map[string]bool)
	version := db.catalogView().version
	for _, t := range db.tables.GetAll() {
		if t.committedAt(version) && !t.droppedAt(version) {
			live[t.RuntimeIdentity] = true
		}
	}
	retired := false
	for id, state := range db.tableAllocators {
		if !live[id] {
			retired = true
			delete(db.tableAllocators, id)
			db.adjustMetadataMemory(tableAllocatorRecordMemory(state), 0)
		}
	}
	if retired {
		retained := make(map[string]tableAllocatorState, len(db.tableAllocators))
		for id, state := range db.tableAllocators {
			retained[id] = state
		}
		db.tableAllocators = retained // release retired bucket capacity at its cold owner
	}
	ids := make([]string, 0, len(db.tableAllocators))
	for id := range db.tableAllocators {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	entries := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, LogEntryTableAllocator{db.tableAllocators[id]})
	}
	return entries
}

func (t *table) initializeNewAllocator(start uint64) {
	state := &tableAllocatorState{Version: 1, Table: t.RuntimeIdentity, StartKnown: true, Start: start,
		IssuedKnown: true, High: start - 1}
	if err := state.validate(); err != nil {
		panic(err)
	}
	t.Allocator = state
	t.allocator.Store(state)
}

func (t *table) allocatorSnapshot() *tableAllocatorState {
	if state := t.allocator.Load(); state != nil {
		return state
	}
	return t.Allocator // immutable schema-import provenance, never inferred
}

// Row-WAL recovery and remote schema imports only merge the old highwater.
// They cannot establish a seed or claim that a live reservation took place.
func (t *table) mergeRecoveredAutoIncrement(high uint64) {
	for {
		previous := atomic.LoadUint64(&t.Auto_increment)
		if high <= previous || atomic.CompareAndSwapUint64(&t.Auto_increment, previous, high) {
			break
		}
	}
	for {
		previous := t.allocator.Load()
		if previous == nil || high <= previous.High {
			return
		}
		next := *previous
		next.High, next.HighAllocated = high, false
		next.IssuedKnown = previous.Issued
		if t.allocator.CompareAndSwap(previous, &next) {
			return
		}
	}
}

// The native allocation boundary may recover legacy row WAL once, before target
// mutation rights. A getter never loads rows to guess historical provenance.
func (t *table) prepareAllocatorRecovery() {
	if t.allocatorRecovery.Load() {
		return
	}
	snapshot := t.allocatorSnapshot()
	if snapshot != nil {
		if err := snapshot.validate(); err != nil || snapshot.Table != t.RuntimeIdentity {
			panic("invalid native table allocator schema provenance")
		}
	} else if t.RuntimeIdentity != "" {
		if err := (tableAllocatorState{Version: 1, Table: t.RuntimeIdentity}).validate(); err != nil {
			panic(err)
		}
	}
	if snapshot == nil || !snapshot.IssuedKnown {
		for _, shard := range t.ActiveShards() {
			access := shard.GetRead(nil, false)
			access.Release()
		}
	}
	// A failed native acquisition leaves the producer incomplete and retryable.
	// Concurrent first callers may share native loads; no failure can publish a
	// false completion witness or let the next allocation skip recovery.
	t.allocatorRecovery.Store(true)
}

func (t *table) loadAllocatorAuthorityLocked() tableAllocatorState {
	state := tableAllocatorState{Version: 1, Table: t.RuntimeIdentity, High: atomic.LoadUint64(&t.Auto_increment)}
	if snapshot := t.allocatorSnapshot(); snapshot != nil {
		state = *snapshot
		if err := state.validate(); err != nil || state.Table != t.RuntimeIdentity {
			panic("invalid native table allocator schema provenance")
		}
	}
	if accepted, exists := t.schema.tableAllocators[t.RuntimeIdentity]; exists {
		var err error
		state, err = mergeTableAllocator(state, accepted)
		if err != nil {
			panic(err)
		}
	}
	if high := atomic.LoadUint64(&t.Auto_increment); high > state.High {
		state.High, state.HighAllocated = high, false
		// High-only import/replay does not prove virgin or issued state.
		state.IssuedKnown = state.Issued
	}
	return state
}

// This pure arithmetic is shared by RAM-only CAS and durable WAL publication.
// Inputs come from validated immutable producers; there is no row/catalog work.
func tableAllocatorReservation(previous tableAllocatorState, high, explicit, generated uint64, observedExplicit bool) (uint64, tableAllocatorState, bool) {
	if high > previous.High {
		previous.High, previous.HighAllocated = high, false
		previous.IssuedKnown = previous.Issued
	}
	base := max(previous.High, explicit)
	if generated > 0 && (base > math.MaxInt64 || generated > math.MaxInt64-base) {
		panic("generated ID block exceeds native integer capacity")
	}
	if generated == 0 && base == previous.High && (!observedExplicit || previous.Issued) {
		return base, previous, false
	}
	next := previous
	next.High = base + generated
	next.IssuedKnown, next.Issued = true, true
	// A nonadvancing explicit input proves an observed ID, but not a
	// highwater allocated by this counter. Preserve that distinction as data;
	// each frontend owns its identity/sequence interpretation.
	next.HighAllocated = generated > 0 || explicit > previous.High || (previous.Issued && previous.HighAllocated)
	return base, next, true
}

// No-WAL engines retain per-table CAS admission. They must not acquire a
// database-global log/mutex merely to publish volatile allocator provenance.
func (t *table) reserveVolatileAutoIncrement(explicit, generated uint64, observedExplicit bool) uint64 {
	for {
		current := t.allocator.Load()
		previous := tableAllocatorState{Version: 1, Table: t.RuntimeIdentity}
		if current != nil {
			previous = *current
		} else if t.Allocator != nil {
			previous = *t.Allocator
		}
		base, next, changed := tableAllocatorReservation(previous, atomic.LoadUint64(&t.Auto_increment), explicit, generated, observedExplicit)
		if !changed {
			return base
		}
		if t.allocator.CompareAndSwap(current, &next) {
			t.mergeRecoveredAutoIncrement(next.High)
			return base
		}
	}
}

// Called only at a table's batch preparation / explicit reservation owner.
// Callers release physical shard locks before entry; ordinary INSERT prepares
// before target write/UNIQUE ownership. Lifecycle admission excludes MOVE/Drop.
func (t *table) reserveLocalAutoIncrement(explicit, generated uint64, observedExplicit bool) uint64 {
	t.prepareAllocatorRecovery()
	db := t.schema
	if failure := db.allocatorFailure.Load(); failure != nil {
		panic(*failure)
	}
	if t.RuntimeIdentity == "" {
		// Historical imported schemas may have no table UUID. Preserve their
		// native highwater CAS, but never manufacture allocator provenance or
		// attach a UUID-bound reservation frame to a name-only generation.
		for {
			high := atomic.LoadUint64(&t.Auto_increment)
			base, next, changed := tableAllocatorReservation(tableAllocatorState{High: high}, high, explicit, generated, false)
			if !changed || atomic.CompareAndSwapUint64(&t.Auto_increment, high, next.High) {
				return base
			}
		}
	}
	// Already supplied/reserved IDs add no facts. The immutable accepted
	// witness makes this a linearizable no-op, including under concurrent WAL.
	if snapshot := t.allocatorSnapshot(); generated == 0 && snapshot != nil && explicit <= snapshot.High && (!observedExplicit || snapshot.Issued) {
		return snapshot.High
	}
	if t.PersistencyMode != Safe && t.PersistencyMode != Logged {
		return t.reserveVolatileAutoIncrement(explicit, generated, observedExplicit)
	}
	if !db.persistenceLifecycle.TryRLock() {
		panic("database maintenance in progress; retry table ID reservation")
	}
	defer db.persistenceLifecycle.RUnlock()
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if failure := db.allocatorFailure.Load(); failure != nil {
		panic(*failure)
	}
	previous := t.loadAllocatorAuthorityLocked()
	var base uint64
	var next tableAllocatorState
	for {
		high := atomic.LoadUint64(&t.Auto_increment)
		var changed bool
		base, next, changed = tableAllocatorReservation(previous, high, explicit, generated, observedExplicit)
		if !changed {
			return base
		}
		if atomic.CompareAndSwapUint64(&t.Auto_increment, high, next.High) {
			break
		}
	}
	// Consume before I/O: an ambiguous failure may have persisted the frame.
	// Further allocation fails closed until recovery re-establishes authority.
	if err := persistenceCall(func() {
		db.transactionLog.Write(LogEntryTableAllocator{next})
		db.transactionLog.Flush(t.PersistencyMode == Safe)
	}); err != nil {
		db.allocatorFailure.Store(&err)
		panic(err)
	}
	if err := db.acceptTableAllocator(next); err != nil {
		panic(err)
	}
	// Lock-free observers see only accepted issued/high facts. The compatibility
	// highwater consumed the range before I/O but never grants that proof.
	t.allocator.Store(&next)
	return base
}

// Cold load/import publishes accepted provenance before the table can escape.
// Ordinary observations remain one immutable load and never load shard data.
func (db *database) restoreTableAllocator(t *table) {
	state := t.loadAllocatorAuthorityLocked()
	if t.Allocator != nil || db.tableAllocators[t.RuntimeIdentity].Version != 0 {
		t.allocator.Store(&state)
		t.mergeRecoveredAutoIncrement(state.High)
	}
}

func (t *table) localTableAllocator() tableAllocatorState {
	if failure := t.schema.allocatorFailure.Load(); failure != nil {
		panic(*failure)
	}
	if snapshot := t.allocatorSnapshot(); snapshot != nil {
		return *snapshot
	}
	return tableAllocatorState{Version: 1, Table: t.RuntimeIdentity, High: atomic.LoadUint64(&t.Auto_increment)}
}

// Return native facts only. In particular a virgin counter's High is Start-1;
// SQL identity functions, missing-object NULLs and numeric types belong in SCM.
func tableAllocatorObservation(state tableAllocatorState) scm.Scmer {
	start := scm.NewNil()
	if state.StartKnown {
		start = scm.NewString(strconv.FormatUint(state.Start, 10))
	}
	issued := scm.NewNil()
	if state.IssuedKnown {
		issued = scm.NewBool(state.Issued)
	}
	return scm.NewSlice([]scm.Scmer{scm.NewString("table_uuid"), scm.NewString(state.Table),
		scm.NewString("start"), start, scm.NewString("issued"), issued,
		scm.NewString("high"), scm.NewString(strconv.FormatUint(state.High, 10)),
		scm.NewString("high_allocated"), scm.NewBool(state.HighAllocated)})
}

// Handover carries the actual immutable publisher state, not guessed seed facts
// from the advisory highwater. Same-name replacement identities never merge.
func (t *table) mergeImportedAllocator(state tableAllocatorState, high uint64) {
	if state.Table != t.RuntimeIdentity {
		panic("table allocator import changed table generation")
	}
	if t.RuntimeIdentity == "" && t.allocatorSnapshot() == nil {
		// Accepted UUID-less predecessor schemas carry only the old highwater.
		// This import must not turn absent identity/start/issued fields into
		// an allocator-v1 authority, including for ordinary non-identity tables.
		if state.Version != 1 || state.StartKnown || state.Start != 0 || state.IssuedKnown || state.Issued || state.HighAllocated {
			panic("legacy table allocator import cannot carry provenance")
		}
		t.mergeRecoveredAutoIncrement(max(high, state.High))
		return
	}
	if err := state.validate(); err != nil {
		panic(err)
	}
	for {
		previous := t.allocator.Load()
		next := state
		if previous != nil {
			var err error
			next, err = mergeTableAllocator(*previous, state)
			if err != nil {
				panic(err)
			}
		}
		if high > next.High {
			next.High, next.HighAllocated = high, false
			next.IssuedKnown = next.Issued
		}
		if t.allocator.CompareAndSwap(previous, &next) {
			break
		}
	}
	t.mergeRecoveredAutoIncrement(high)
}

func (t *table) currentTableAllocator() tableAllocatorState {
	if t.RuntimeIdentity == "" {
		panic("table allocator current-value provenance is unavailable for a legacy table without native identity")
	}
	if !sharedClusterSource(t) {
		if activeReplication(t.schema) != nil {
			panic("table allocator current-value authority is not supported for independent replication writers")
		}
		return t.localTableAllocator()
	}
	cluster := activeNodeCluster()
	if cluster == nil {
		panic("table allocator authority unavailable")
	}
	reply, err := cluster.reserveIDs(cluster.ctx, clusterIDRequest{Database: t.schema.Name, Table: t.Name, Identity: t.RuntimeIdentity, Read: true})
	if err != nil {
		panic(err)
	}
	if err := reply.State.validate(); err != nil || reply.State.Table != t.RuntimeIdentity {
		panic("table allocator observation reply lacks its native generation provenance")
	}
	return reply.State
}
