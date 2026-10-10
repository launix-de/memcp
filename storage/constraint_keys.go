/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "github.com/launix-de/memcp/scm"

// constraintKeyBinding is invocation-owned. Only explicitly declared key
// recipes use it; normal constraints keep their existing raw-key path. The
// immutable computed-index descriptors reuse the engine's existing mapper ABI.
type constraintKeyBinding struct {
	columns     []string
	projected   []int
	recipes     []scm.Scmer
	schema      []scm.Scmer
	descriptors []scm.Scmer
}

func (t *table) bindConstraintKeys(columns []string, nullsEqual bool) *constraintKeyBinding {
	var projections map[string]scm.Scmer
	if snapshot := t.columnNamesSnapshot.Load(); snapshot != nil {
		projections = snapshot.keyProjections
	} else {
		// Only private unpublished declarations lack a column snapshot.
		for _, column := range t.Columns {
			if column.KeyProjection != nil {
				if projections == nil {
					projections = make(map[string]scm.Scmer)
				}
				projections[column.Name] = *column.KeyProjection
			}
		}
	}
	if len(projections) == 0 && !nullsEqual {
		return nil
	}
	var binding *constraintKeyBinding
	if nullsEqual {
		binding = &constraintKeyBinding{columns: columns, recipes: make([]scm.Scmer, len(columns)), schema: newExactScanAccessSchema(columns), descriptors: make([]scm.Scmer, len(columns))}
		for i, name := range columns {
			binding.schema[scanAccessSchemaHeaderSize+i] = newScanBoundarySpec(name, EqualMatcher, i, i, true, true, "", true, -1, nil, nil, "", false)
		}
	}
	for i, name := range columns {
		if recipe, found := projections[name]; found {
			if binding == nil {
				binding = &constraintKeyBinding{columns: columns, recipes: make([]scm.Scmer, len(columns)), schema: newExactScanAccessSchema(columns), descriptors: make([]scm.Scmer, len(columns))}
			}
			descriptor := compileComputedScanIndex(recipe, []string{name})
			binding.projected = append(binding.projected, i)
			binding.recipes[i] = recipe
			binding.descriptors[i] = descriptor
			binding.schema[scanAccessSchemaHeaderSize+i] = newScanBoundarySpec(name, EqualMatcher, i, i, true, true, "", nullsEqual, len(columns)+i, []string{name}, nil, "", false)
		}
	}
	return binding
}

func (b *constraintKeyBinding) project(values []scm.Scmer) {
	for _, i := range b.projected {
		if !values[i].IsNil() {
			values[i] = scm.Apply(b.recipes[i], values[i])
			if values[i].IsNil() {
				panic("constraint key recipe erased a nonempty value")
			}
		}
	}
}

func (b *constraintKeyBinding) access(keys []scm.Scmer) scanAccess {
	values := make([]scm.Scmer, len(keys)+len(b.descriptors))
	copy(values, keys)
	copy(values[len(keys):], b.descriptors)
	access, ok := scanAccessFromScheme(scm.NewSlice(b.schema), values, nil)
	if !ok {
		panic("invalid bound constraint key access")
	}
	return access
}

// condition chooses a projected predicate once, outside the scan. It retains
// immutable callbacks and keys, never serial reader state across shard workers.
func (b *constraintKeyBinding) condition(keys []scm.Scmer) scm.Scmer {
	return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		for i := range b.columns {
			if values[i].IsNil() {
				return scm.NewBool(false)
			}
		}
		projected := append([]scm.Scmer(nil), values...)
		b.project(projected)
		for i := range b.columns {
			if !scm.Equal(projected[i], keys[i]) {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
}

// lookup is separate from GetRecordidForUnique so raw constraints acquire no
// callback checks or mapped readers in their ordinary element loop.
func (b *constraintKeyBinding) lookup(shard *storageShard, keys []scm.Scmer, tx *TxContext) (result uint32, present bool) {
	release := shard.GetRead(tx)
	defer release()
	shard.ensureMainCount(false, tx)
	access := b.access(keys)
	shard.ensureScanAccessColumns(access, false, tx)
	readers := make([]ColumnStorage, len(b.columns))
	for i, name := range b.columns {
		readers[i] = shard.getColumnStorageOrPanic(name, false, tx)
	}
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	mainCount := shard.main_count
	var ids [8]uint32
	values := make([]scm.Scmer, len(b.columns))
	shard.iterateIndex(tx, access, len(shard.inserts), ids[:], 1, nil, func(batch []uint32) bool {
		for _, recid := range batch {
			if tx != nil && tx.Mode == TxACID {
				if !tx.IsVisible(shard, recid) {
					continue
				}
			} else if shard.deletions.Get(uint(recid)) {
				continue
			}
			for i, name := range b.columns {
				if recid < mainCount {
					values[i] = readers[i].GetValue(recid)
				} else {
					values[i] = shard.getDelta(int(recid-mainCount), name)
				}
			}
			b.project(values)
			matched := true
			for i := range values {
				if values[i].IsNil() != keys[i].IsNil() || !scm.Equal(values[i], keys[i]) {
					matched = false
					break
				}
			}
			if matched {
				result, present = recid, true
				return false
			}
		}
		return true
	})
	return
}
