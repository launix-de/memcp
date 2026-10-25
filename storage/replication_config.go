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
import "os"
import "path/filepath"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Only configuration and completed data-placement metadata cross persistence.
// No MOSI phase, owner, reader set, request, epoch or sequence appears here.
type replicaBackup struct {
	Resource   mosiResource `json:"resource"`
	Generation string       `json:"generation"`
}

type replicaConfigEntry struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name"`
	Auth   *replicaAuth   `json:"auth,omitempty"`
	Backup *replicaBackup `json:"backup,omitempty"`
	Value  bool           `json:"value,omitempty"`
}

func (r *replicationManager) writeConfig(entry replicaConfigEntry) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication configuration: %v", failure)
		}
	}()
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	update := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.Apply(a[0], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewString(string(data))}))
	})
	r.config.Insert([]string{"node", "key", "value"}, [][]scm.Scmer{{scm.NewString(r.id), scm.NewString(entry.Kind + "/" + entry.Name), scm.NewString(string(data))}}, []string{"$update"}, update, false, nil)
	return nil
}

func (r *replicationManager) openConfig() (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication configuration: %v", failure)
		}
	}()
	// A legacy private journal cannot be an ownership source. Refuse an
	// implicit upgrade which could turn an offline owner's files into local M.
	if _, err := os.Stat(filepath.Join(Basepath, ".replication-state")); err == nil {
		return fmt.Errorf("legacy replication journal present: explicit resynchronization with live owners is required")
	} else if !os.IsNotExist(err) {
		return err
	}
	t, _ := CreateTable("system", "replication_config", Safe, true)
	for _, name := range []string{"node", "key", "value"} {
		if _, exists := t.ResolveColumnName(name, false); !exists {
			t.CreateColumn(name, "text", nil, nil)
		}
	}
	createTableKey(t, uniqueKey{Id: "PRIMARY", Cols: []string{"node", "key"}}, nil)
	return r.loadConfig(t)
}

func (r *replicationManager) loadConfig(t *table) error {
	r.config = t
	collect := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append(append([]scm.Scmer(nil), a[0].Slice()...), a[1]))
	})
	combine := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append(append([]scm.Scmer(nil), a[0].Slice()...), a[1].Slice()...))
	})
	rows := t.scan(nil, scm.NewSlice([]scm.Scmer{newScanAccessHeader(1, scanAccessConsumerScan, 0, -1), newScanBoundarySpec("node", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false)}), []scm.Scmer{scm.NewString(r.id)}, nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{"value"}, collect, scm.NewSlice(nil), combine, false)
	for _, row := range rows.Slice() {
		var entry replicaConfigEntry
		if err := json.Unmarshal([]byte(row.String()), &entry); err != nil {
			return err
		}
		switch entry.Kind {
		case "users":
			if entry.Auth == nil {
				return fmt.Errorf("invalid saved login")
			}
			r.directory.Users[entry.Name] = *entry.Auth
		case "retired":
			r.directory.Retired[entry.Name] = entry.Value
		case "backup":
			if entry.Backup == nil {
				delete(r.backups, entry.Name)
				continue
			}
			if !entry.Backup.Resource.valid() || entry.Backup.Resource.key() != entry.Name {
				return fmt.Errorf("invalid backup placement")
			}
			if _, err := uuid.Parse(entry.Backup.Generation); entry.Backup.Resource.Shard != "" && err != nil {
				return err
			}
			r.backups[entry.Name] = *entry.Backup
		default:
			return fmt.Errorf("unknown replication configuration kind %q", entry.Kind)
		}
	}

	return nil
}

// saveBackup records a completed immutable data generation through ordinary
// storage. It conveys no permission; only reconnect reports can restore rights.
func (r *replicationManager) saveBackup(resource mosiResource, generation string) error {
	backup := replicaBackup{Resource: resource, Generation: generation}
	if r.config != nil {
		if err := r.writeConfig(replicaConfigEntry{Kind: "backup", Name: resource.key(), Backup: &backup}); err != nil {
			return err
		}
	}
	r.mu.Lock()
	if r.backups == nil {
		r.backups = make(map[string]replicaBackup)
	}
	r.backups[resource.key()] = backup
	r.mu.Unlock()
	return nil
}

// Promotion removes a safety-copy placement only after the ordinary schema
// points at the installed data and every foreign read/write permission is gone.
func (r *replicationManager) clearBackup(resource mosiResource) error {
	r.mu.Lock()
	_, exists := r.backups[resource.key()]
	r.mu.Unlock()
	if !exists {
		return nil
	}
	if r.config != nil {
		if err := r.writeConfig(replicaConfigEntry{Kind: "backup", Name: resource.key()}); err != nil {
			return err
		}
	}
	r.mu.Lock()
	delete(r.backups, resource.key())
	r.mu.Unlock()
	return nil
}
