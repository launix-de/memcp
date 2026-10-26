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
import "maps"

type replicaRecord struct {
	Handoff        string                        `json:"handoff,omitempty"`
	InitialClaimed bool                          `json:"initial_claimed,omitempty"`
	Epoch          uint64                        `json:"epoch"`
	State          string                        `json:"state"`
	Owner          string                        `json:"owner,omitempty"`
	Readers        map[string]bool               `json:"readers,omitempty"`
	Resource       mosiResource                  `json:"resource"`
	Generation     string                        `json:"generation,omitempty"`
	Requests       map[string]replicaRequestMark `json:"requests,omitempty"`
}

type replicaRequestMark struct {
	Sequence  uint64 `json:"sequence"`
	Write     bool   `json:"write"`
	ID        string `json:"id,omitempty"`
	Completed bool   `json:"completed,omitempty"`
}

type replicaDirectory struct {
	Records     map[string]replicaRecord `json:"records"`
	Users       map[string]replicaAuth   `json:"users"`
	Retired     map[string]bool          `json:"retired"`
	Initialized map[string]bool          `json:"initialized,omitempty"`
	Sequence    uint64                   `json:"sequence,omitempty"`
}

// updateRecord publishes one immutable runtime directory entry. MOSI rights,
// ownership, epochs and request sequences are never written to a backend.
func (r *replicationManager) updateRecord(resource mosiResource, update func(*replicaRecord) bool) replicaRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.directory.Records[resource.key()]
	if record.State == "" {
		record.Resource, record.State = resource, "IO"
	}
	record.Readers, record.Requests = maps.Clone(record.Readers), maps.Clone(record.Requests)
	if update(&record) {
		r.directory.Records[resource.key()] = record
	}
	return record
}

func (r *replicationManager) storeConfig(entry replicaConfigEntry) error {
	if entry.Kind != "users" && entry.Kind != "retired" {
		return fmt.Errorf("not persistent replication configuration: %s", entry.Kind)
	}
	if entry.Kind == "users" && entry.Auth == nil {
		return fmt.Errorf("missing saved user proof")
	}
	if r.config != nil {
		if err := r.writeConfig(entry); err != nil {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch entry.Kind {
	case "users":
		r.directory.Users[entry.Name] = *entry.Auth
	case "retired":
		r.directory.Retired[entry.Name] = entry.Value

	}
	return nil
}

func (r *replicationManager) cell(name string) *replicaCell {
	if r.cluster != nil {
		if value, ok := r.cluster.states.Load(name); ok {
			return value.(*replicaCell)
		}
		value, _ := r.cluster.states.LoadOrStore(name, &replicaCell{})
		return value.(*replicaCell)
	}
	if snapshot := r.cellSnapshot.Load(); snapshot != nil {
		if c := (*snapshot)[name]; c != nil {
			return c
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.cells[name]
	if c == nil {
		c = &replicaCell{}
		r.cells[name] = c
		snapshot := make(map[string]*replicaCell, len(r.cells))
		for name, cell := range r.cells {
			snapshot[name] = cell
		}
		r.cellSnapshot.Store(&snapshot)
	}
	return c
}
