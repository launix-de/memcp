/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "bytes"
import "encoding/json"
import "maps"
import "unsafe"
import "unicode/utf8"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// V19 reuses the native ADD historical-prefix contract. Future DEFAULT changes
// never rewrite Value/Extents. No SQL expression or callback is executable here.
type catalogColumnInitialization struct {
	Default     scm.Scmer                      `json:"default"`
	DefaultName string                         `json:"default_name,omitempty"`
	Value       scm.Scmer                      `json:"value"`
	Extents     map[string]columnInitialExtent `json:"extents"`
}

// These V19 scalar fields use the existing durable-value codec, rather than
// public JSON's calendar text conversion. Old catalog domains are untouched.
func (d catalogColumnInitialization) MarshalJSON() ([]byte, error) {
	def, err := scm.MarshalPersistentJSON(d.Default)
	if err != nil {
		return nil, err
	}
	value, err := scm.MarshalPersistentJSON(d.Value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Default     json.RawMessage                `json:"default"`
		DefaultName string                         `json:"default_name,omitempty"`
		Value       json.RawMessage                `json:"value"`
		Extents     map[string]columnInitialExtent `json:"extents"`
	}{def, d.DefaultName, value, d.Extents})
}

func (d *catalogColumnInitialization) UnmarshalJSON(data []byte) error {
	var payload struct {
		Default     json.RawMessage                `json:"default"`
		DefaultName string                         `json:"default_name,omitempty"`
		Value       json.RawMessage                `json:"value"`
		Extents     map[string]columnInitialExtent `json:"extents"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return err
	}
	if len(payload.Default) == 0 || len(payload.Value) == 0 {
		return fmt.Errorf("initialized column requires explicit durable scalar values")
	}
	if err := scm.UnmarshalPersistentJSON(payload.Default, &d.Default); err != nil {
		return err
	}
	if err := scm.UnmarshalPersistentJSON(payload.Value, &d.Value); err != nil {
		return err
	}
	d.DefaultName, d.Extents = payload.DefaultName, payload.Extents
	return nil
}

func (d *catalogColumnInitialization) containers() uint {
	if d == nil {
		return 0
	}
	bytes := uint(unsafe.Sizeof(*d))
	if d.Extents != nil {
		bytes += 64 + 64*uint(len(d.Extents))
	}
	return bytes
}

func (d *catalogColumnInitialization) memory() uint {
	if d == nil {
		return 0
	}
	bytes := d.containers() + uint(len(d.DefaultName)) + scm.ComputeSize(d.Default) + scm.ComputeSize(d.Value) - 2*uint(unsafe.Sizeof(scm.Scmer{}))
	for id := range d.Extents {
		bytes += uint(len(id))
	}
	return bytes
}

// These are native self-contained scalar values, not frontend syntax.
// Date/UUID keep their existing persistent codecs; callable/container values
// cannot become a historical initializer or executable default here.
func validateColumnInitialScalar(value scm.Scmer) error {
	if value.IsDate() || value.IsUUID() {
		return nil
	}
	return validateColumnProperty("column initializer", value)
}

func validateColumnInitialization(d catalogColumnAppend) error {
	i := d.Initialization
	if i == nil {
		return nil
	}
	if !utf8.ValidString(i.DefaultName) || i.Extents == nil {
		return fmt.Errorf("invalid initialized column declaration")
	}
	if err := validateColumnInitialScalar(i.Default); err != nil {
		return err
	}
	if err := validateColumnInitialScalar(i.Value); err != nil {
		return err
	}
	if !d.Type.Nullable && i.Value.IsNil() {
		return fmt.Errorf("required initialized column has NULL history")
	}
	for id := range i.Extents {
		if identity, err := uuid.Parse(id); err != nil || identity == uuid.Nil || identity.String() != id {
			return fmt.Errorf("invalid initialized column generation")
		}
	}
	return nil
}

// Caller owns the validated declaration lease and the existing DDL write side.
// Rebuild preserves pending rows/forwarded journals before relinquishing DDL;
// after this acquisition no affected generation can be rebuilt until decision.
func captureColumnInitialization(tx *TxContext, t *table, c *column, fillExisting bool) *catalogColumnInitialization {
	value := scm.NewNil()
	if fillExisting {
		value = c.Default
	}
	if c.sanitizer != nil {
		value = c.sanitizer(value)
	}
	if err := validateColumnInitialScalar(value); err != nil {
		panic(err)
	}
	initial := make(map[string]columnInitialExtent)
	seen := make(map[*storageShard]bool)
	var capture func(*storageShard)
	capture = func(s *storageShard) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		if s.t != t {
			panic("initialized append lost its native endpoint")
		}
		access := s.GetRead(tx, false)
		defer access.Release()
		s = access.shard
		s.ensureMainCount(false, tx)
		s.mu.RLock()
		initial[s.uuid.String()] = columnInitialExtent{s.main_count, uint32(len(s.deltaRows()))}
		s.mu.RUnlock()
		if next := s.loadNext(); next != nil && s.nextReady.Load() {
			capture(next)
		}
	}
	t.mu.Lock()
	shards := append(append([]*storageShard(nil), t.Shards...), t.PShards...)
	t.mu.Unlock()
	for _, s := range shards {
		capture(s)
	}
	tx.mu.Lock()
	transactions := tx.shardTransactions()
	tx.mu.Unlock()
	visited := make(map[*storageShardTransaction]bool)
	var journals func(*storageShardTransaction)
	journals = func(st *storageShardTransaction) {
		if st == nil || visited[st] {
			return
		}
		visited[st] = true
		if st.clusterShard == nil {
			panic("initialized append requires native row journal identity")
		}
		if st.clusterShard.t != t {
			return
		}
		capture(st.clusterShard)
		st.mu.Lock()
		extent, found := initial[st.clusterShard.uuid.String()]
		covered := found && st.remoteObject == nil
		check := func(recid uint32) {
			covered = covered && uint64(recid) < uint64(extent.MainRows)+uint64(extent.DeltaRows)
		}
		for _, recid := range st.InsertRecids {
			check(recid)
		}
		for _, recid := range st.DeletedRecids {
			check(recid)
		}
		for _, recid := range st.DeleteRecids {
			check(recid)
		}
		for _, change := range st.UndeleteRecids {
			check(change.recid)
		}
		endpoint := st.endpoint
		st.mu.Unlock()
		if !covered {
			panic("initialized append does not cover its actual original row journal")
		}
		journals(endpoint)
		journals(st.forwarded.Load())
	}
	for s, st := range transactions {
		if s.t == t {
			journals(st)
		}
	}
	c.InitialValue, c.InitialValues = value, initial
	return &catalogColumnInitialization{c.Default, c.DefaultName, value, maps.Clone(initial)}
}

func (tx *TxContext) retainedColumnDDL(t *table) bool {
	if tx != nil {
		if layout := tx.scopedRelationLayout(t); layout != nil {
			return layout.lease.ddlTable.Load() == t
		}
	}
	if changes := tx.publicCatalogChanges(); changes != nil {
		changes.mu.Lock()
		defer changes.mu.Unlock()
		if lease := changes.leases[t]; lease != nil {
			return lease.ddlTable.Load() == t
		}
	}
	return false
}

// Native readers and index setup borrow this immutable invocation view once.
// It is never a public source/column registration or a transaction in a cache.
func (tx *TxContext) pendingAppendedColumns(t *table) map[string]*column {
	if tx != nil && tx.transactionState != nil && tx.State == TxActive {
		if changes := tx.publicCatalogChanges(); changes != nil {
			if views := changes.views.Load(); views != nil {
				return views.appendedColumns[t]
			}
		}
	}
	return nil
}

// The caller supplies its already-bound immutable descriptor, never a fresh
// per-row transaction/catalog lookup. The captured original prefix is immutable.
type nativeColumnInitialValue struct {
	columnRecoveryValue
	main ColumnStorage
}

type nativeColumnInitialInput struct {
	fields map[string]nativeColumnInitialValue
}

func (s *storageShard) getDeltaInput(idx int, name string, input *nativeColumnInitialInput) scm.Scmer {
	// Every captured row predates this fresh logical declaration. Aborted ADD
	// slots are not value provenance. Later RecIDs retain explicit values/NULL.
	if value, ok := initialColumnValue(input, name, idx); ok {
		return value
	}
	return s.getDelta(idx, name)
}

func (s *storageShard) initializedColumnInput(input *nativeTableInput) *nativeColumnInitialInput {
	if input == nil || input.table != s.t {
		panic("column input belongs to another native table")
	}
	return s.initializedColumnNames(input.names)
}

func (s *storageShard) initializedColumnNames(names *tableColumnNamesSnapshot) *nativeColumnInitialInput {
	if names != nil {
		if input := names.initialByShard[s.uuid]; input != nil {
			return input
		}
		return names.appendedInitialInput
	}
	return nil
}

func initialColumnValue(input *nativeColumnInitialInput, name string, delta int) (scm.Scmer, bool) {
	if input != nil {
		if c, ok := input.fields[name]; ok && delta >= 0 && uint64(delta) < uint64(c.DeltaRows) {
			return c.Value, true
		}
	}
	return scm.NewNil(), false
}

func initialMainStorage(c *column, id string) ColumnStorage {
	if c != nil {
		if initial, ok := c.InitialValues[id]; ok {
			value := c.InitialValue
			if c.recoverPersisted != nil && !value.IsNil() {
				value = c.recoverPersisted(value)
			}
			return &StorageConst{value: value, count: uint64(initial.MainRows)}
		}
	}
	return new(StorageSparse)
}

func usesInitialColumn(input *nativeColumnInitialInput, name string) bool {
	if input == nil {
		return false
	}
	_, found := input.fields[name]
	return found
}
