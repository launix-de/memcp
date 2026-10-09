/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: AGPL-3.0-or-later */

package storage

import "sort"
import "github.com/launix-de/memcp/scm"

// columnInitialValue describes only the rows present when this column was
// added to one persisted shard generation. The optional persisted extents
// are constructed under caller-owned DDL, maintenance and schema ownership;
// the map and its values are immutable afterwards. Rebuilt generations have
// different UUIDs and persist their actual column values instead.
type columnInitialValue struct {
	Value     scm.Scmer `json:"value"`
	MainRows  uint32    `json:"main_rows"`
	DeltaRows uint32    `json:"delta_rows"`
}

func (t *table) columnDeclarations() []*column {
	if snapshot := t.columnNamesSnapshot.Load(); snapshot != nil {
		return snapshot.declarations
	}
	// Only private constructors and unpublished test tables lack a snapshot.
	return t.Columns
}

// initializeColumnRowsLocked runs at the explicit ADD boundary. The caller
// owns ddlMu, maintenanceMu and schemalock. It acquires shard rights and ordered
// shard locks, retained until the caller attaches the complete declaration,
// so an overlapping INSERT cannot publish an uninitialized new-column slot.
// No ordinary reader or INSERT needs to inspect these initialization records.
func (t *table) initializeColumnRowsLocked(c *column, fillExisting, unique bool) (release func()) {
	value := scm.NewNil()
	if c.hasDefault() {
		declared := c.defaultValue()
		if c.sanitizer != nil {
			declared = c.sanitizer(declared)
		}
		if c.DefaultExpression == "" {
			c.Default = declared
		}
		if fillExisting {
			value = declared
		}
	}
	t.mu.Lock()
	shards := make([]*storageShard, 0, len(t.Shards)+len(t.PShards))
	seen := make(map[*storageShard]bool)
	for _, group := range [][]*storageShard{t.Shards, t.PShards} {
		for _, shard := range group {
			if shard != nil && !seen[shard] {
				seen[shard] = true
				shards = append(shards, shard)
			}
		}
	}
	// The caller owns schemalock, ddlMu and maintenanceMu: shard append and maintenance
	// publication cannot replace this directory. Release the topology mutex
	// before taking shard locks; ID allocation takes it from a shard batch.
	t.mu.Unlock()
	sort.Slice(shards, func(i, j int) bool { return shards[i].uuid.String() < shards[j].uuid.String() })
	locked := 0
	releases := make([]func(), 0, len(shards))
	release = func() {
		for i := locked - 1; i >= 0; i-- {
			shards[i].mu.Unlock()
		}
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	complete := false
	defer func() {
		if !complete {
			release()
		}
	}()
	var visibleRows uint64
	authoritative := make(map[*storageShard]bool)
	for _, shard := range t.ActiveShards() {
		authoritative[shard] = true
	}
	for _, shard := range shards {
		releases = append(releases, shard.GetExclusive())
		shard.ensureMainCount(false, nil)
	}
	for _, shard := range shards {
		shard.mu.Lock()
		locked++
		rows := uint64(shard.main_count) + uint64(len(shard.inserts))
		if deleted := uint64(shard.deletions.Count()); authoritative[shard] && rows > deleted {
			visibleRows += rows - deleted
		}
		if value.IsNil() && !c.AllowNull && (rows > uint64(shard.deletions.Count()) || shard.rollbackProtected.Count() != 0) {
			panic("column " + c.Name + " cannot be NULL for existing rows")
		}
	}
	if unique && !value.IsNil() && visibleRows > 1 {
		panic("new unique column has duplicate values in existing rows")
	}
	// Allocate complete delta replacements before changing any shared state.
	positions := make([]int, len(shards))
	rows := make([][][]scm.Scmer, len(shards))
	initial := make(map[string]columnInitialValue, len(shards))
	for i, shard := range shards {
		if shard.main_count == 0 && len(shard.inserts) == 0 {
			positions[i] = -1
			continue
		}
		position, exists := shard.deltaColumns[c.Name]
		if !exists {
			position = len(shard.deltaColumns)
		}
		positions[i] = position
		rows[i] = make([][]scm.Scmer, len(shard.inserts))
		for j, old := range shard.inserts {
			width := len(old)
			if width <= position {
				width = position + 1
			}
			row := make([]scm.Scmer, width)
			copy(row, old)
			row[position] = value
			rows[i][j] = row
		}
		initial[shard.uuid.String()] = columnInitialValue{value, shard.main_count, uint32(len(shard.inserts))}
	}
	if len(initial) != 0 {
		c.InitialValues = initial
	}
	for i, shard := range shards {
		if positions[i] == -1 {
			shard.columns[c.Name] = new(StorageSparse)
			continue
		}
		shard.columns[c.Name] = &StorageConst{value: value, count: uint64(shard.main_count)}
		shard.deltaColumns[c.Name] = positions[i]
		shard.inserts = rows[i]
	}
	complete = true
	return release
}

// initialColumnValues applies only to recovery of the captured generation and
// prefix. Presence in a newer WAL row, including an explicit NULL, is untouched.
func (s *storageShard) initialColumnValues() map[string]columnInitialValue {
	result := make(map[string]columnInitialValue)
	for _, c := range s.t.columnDeclarations() {
		if initial, exists := c.InitialValues[s.uuid.String()]; exists {
			result[c.Name] = initial
		}
	}
	return result
}
