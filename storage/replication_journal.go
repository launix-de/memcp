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

import "os"
import "encoding/json"
import "path/filepath"
import "github.com/launix-de/memcp/storage/mosi"

type replicaRecord struct {
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
	Sequence uint64 `json:"sequence"`
	Write    bool   `json:"write"`
}

type replicaJournal struct {
	Version     int                      `json:"version"`
	Records     map[string]replicaRecord `json:"records"`
	Users       map[string]replicaAuth   `json:"users"`
	Retired     map[string]bool          `json:"retired"`
	Initialized map[string]bool          `json:"initialized,omitempty"`
	Sequence    uint64                   `json:"sequence,omitempty"`
}

func writeReplicationJournal(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".replication-tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (r *replicationManager) saveLocked() error {
	data, err := json.Marshal(r.journal)
	if err != nil {
		return err
	}
	return writeReplicationJournal(filepath.Join(Basepath, ".replication-state"), data)
}

func (r *replicationManager) cell(name string) *replicaCell {
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

func (r *replicationManager) setLocal(name, state string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.journal.Records[name]
	record.State = state
	r.journal.Records[name] = record
	if err := r.saveLocked(); err != nil {
		return err
	}
	r.cells[name].rights.Publish(mosi.ParsePhase(state))
	return nil
}
