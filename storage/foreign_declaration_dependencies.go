/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "slices"
import "sort"

// Published foreign programs consume these exact native endpoint declarations.
// Callers own schema publication; an incompatible endpoint change needs a joint
// declaration replacement, rather than leaving the installed programs behind.
func (t *table) foreignColumnDependency(name string) *foreignKey {
	for index := range t.Foreign {
		foreign := &t.Foreign[index]
		if foreign.Tbl1 == t.Name && slices.Contains(foreign.Cols1, name) ||
			foreign.Tbl2 == t.Name && slices.Contains(foreign.Cols2, name) {
			return foreign
		}
	}
	return nil
}

func (t *table) requireNoConstraintColumnChange(name string) {
	if dependency := t.checkColumnDependency(name); dependency != nil {
		panic("column declaration is required by check constraint " + dependency.Id + ": " + name)
	}
	if dependency := t.foreignColumnDependency(name); dependency != nil {
		panic("column declaration is required by foreign key " + dependency.Id + ": " + name)
	}
}

// A foreign key names its ordered parent columns, not a particular UNIQUE ID.
// Removing a redundant declaration is valid while another exact native key
// continues to provide that same lookup/uniqueness contract.
func (t *table) foreignUniqueDependency(removed int) *foreignKey {
	key := t.Unique[removed]
	for index := range t.Foreign {
		foreign := &t.Foreign[index]
		if foreign.Tbl2 != t.Name || !slices.Equal(foreign.Cols2, key.Cols) {
			continue
		}
		supported := false
		for index, other := range t.Unique {
			if index != removed && slices.Equal(other.Cols, foreign.Cols2) {
				supported = true
				break
			}
		}
		if !supported {
			return foreign
		}
	}
	return nil
}

// DROP owns schema publication. Retire the dropped child's declarations at
// both native endpoints before saving that same catalog. A surviving child
// prevents dropping its parent; DROP never silently removes its constraint.
// All admission and DDL acquisitions precede the first metadata effect.
func (t *table) retireDroppedForeignDeclarationsLocked() (func(), schemaSaveMode) {
	mode := t.schemaSaveMode()
	if len(t.Foreign) == 0 {
		return func() {}, mode
	}
	db := t.schema
	foreign := append([]foreignKey(nil), t.Foreign...)
	endpoints := []*table{t}
	seen := map[*table]bool{t: true}
	for _, fk := range foreign {
		if fk.Tbl2 == t.Name && fk.Tbl1 != t.Name && db.tables.Get(fk.Tbl1) != nil {
			panic("table is required by foreign key " + fk.Id + ": " + t.Name)
		}
		if fk.Tbl1 != t.Name {
			continue
		}
		if parent := db.tables.Get(fk.Tbl2); parent != nil && !seen[parent] {
			seen[parent] = true
			endpoints = append(endpoints, parent)
		}
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Name < endpoints[j].Name })
	var releases []func()
	var locked []*table
	release := func() {
		for i := len(locked) - 1; i >= 0; i-- {
			locked[i].ddlMu.Unlock()
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
	for _, endpoint := range endpoints {
		if endpoint != t {
			requireTableMaintenance(db.Name, endpoint.Name, maintenanceAlter)
			releases = append(releases, endpoint.admitNativeMaintenance())
			mode = min(mode, endpoint.schemaSaveMode())
		}
		// Other maintenance can hold DDL before schema. Never wait for that
		// inverse prefix while holding the schema publication owner.
		if !endpoint.ddlMu.TryLock() {
			panic("foreign declaration conflict: endpoint maintenance is active; retry statement")
		}
		locked = append(locked, endpoint)
	}
	for _, fk := range foreign {
		if fk.Tbl1 != t.Name {
			continue
		}
		parent := db.tables.Get(fk.Tbl2)
		removeFKTriggers(t, parent, fk)
		removeForeignMetadata(t, fk)
		if parent != nil && parent != t {
			removeForeignMetadata(parent, fk)
		}
	}
	complete = true
	return release, mode
}
