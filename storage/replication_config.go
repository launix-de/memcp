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
import "fmt"
import "sort"
import "path/filepath"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Only configuration and completed data-placement metadata cross persistence.
// No MOSI phase, owner, reader set, request, epoch or sequence appears here.
type replicaBackup struct {
	Forwards []shardForward `json:"forwards,omitempty"`
	Resource mosiResource   `json:"resource"`
	Content  string         `json:"content"`
}

type replicaConfigEntry struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name"`
	Auth   *replicaAuth   `json:"auth,omitempty"`
	Backup *replicaBackup `json:"backup,omitempty"`
	Value  bool           `json:"value,omitempty"`
}

// Multiple placement updates use one native transaction so replacement cannot
// leave two competing completed-copy records after a crash.
func (r *replicationManager) writeConfig(entries ...replicaConfigEntry) (err error) {
	var tx *TxContext
	if len(entries) > 1 {
		tx = NewTxContext(TxACID)
		defer tx.Rollback()
	}
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication configuration: %v", failure)
		}
	}()
	for _, entry := range entries {
		data, failure := json.Marshal(entry)
		if failure != nil {
			return failure
		}
		update := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			return scm.Apply(a[0], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewString(string(data))}))
		})
		r.config.Insert([]string{"node", "key", "value"}, [][]scm.Scmer{{scm.NewString(r.id), scm.NewString(entry.Kind + "/" + entry.Name), scm.NewString(string(data))}}, []string{"$update"}, update, false, nil, tx)
	}
	if tx != nil {
		return tx.Commit()
	}
	return nil
}

// Old completed-copy records may contain the removed logical resource ID.
// Only their actual file UUID and explicit forwarding survive the upgrade;
// none of these metadata establish a permission.
func (backup *replicaBackup) UnmarshalJSON(data []byte) error {
	type fields replicaBackup
	var entry struct {
		*fields
		Legacy string `json:"generation"`
	}
	entry.fields = (*fields)(backup)
	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}
	if backup.Content == "" {
		backup.Content = entry.Legacy
	}
	if backup.Resource.Shard != "" && backup.Resource.Shard != backup.Content {
		actual := backup.Resource
		actual.Shard = backup.Content
		backup.Forwards = append(backup.Forwards, shardForward{backup.Resource, actual})
		backup.Resource = actual
	}
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
	var placements []replicaBackup
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
			if !entry.Backup.Resource.valid() {
				return fmt.Errorf("invalid backup placement")
			}
			if _, err := uuid.Parse(entry.Backup.Content); entry.Backup.Resource.Shard != "" && err != nil {
				return err
			}
			placements = append(placements, *entry.Backup)
		default:
			return fmt.Errorf("unknown replication configuration kind %q", entry.Kind)
		}
	}

	for _, backup := range placements {
		for _, route := range backup.Forwards {
			r.forwardResource(route.From, route.To)
		}
	}
	for _, backup := range placements {
		r.backups[r.resourceKey(backup.Resource)] = backup
	}

	return nil
}

// saveBackup records a completed copy by its actual shard UUID (or a schema
// digest). Routing metadata describes old-to-new UUIDs, never MOSI permissions.
func (r *replicationManager) saveBackup(resource mosiResource, content string) error {
	actual := resource
	if resource.Shard != "" {
		if _, err := uuid.Parse(content); err != nil {
			return err
		}
		actual.Shard = content
		if r.canonicalResource(resource) != r.canonicalResource(actual) {
			r.forwardResource(resource, actual)
		}
	}
	backup := replicaBackup{Resource: actual, Content: content, Forwards: r.shardRoutes(resource)}
	r.mu.Lock()
	previous, exists := r.backups[r.resourceKey(resource)]
	r.mu.Unlock()
	if r.config != nil {
		entries := []replicaConfigEntry{{Kind: "backup", Name: actual.key(), Backup: &backup}}
		if exists && previous.Resource.key() != actual.key() {
			entries = append(entries, replicaConfigEntry{Kind: "backup", Name: previous.Resource.key()})
		}
		if err := r.writeConfig(entries...); err != nil {
			return err
		}
	}
	r.mu.Lock()
	if r.backups == nil {
		r.backups = make(map[string]replicaBackup)
	}
	r.backups[r.resourceKey(resource)] = backup
	r.mu.Unlock()
	return nil
}

// Promotion removes a safety-copy placement only after the ordinary schema
// points at the installed data and every foreign read/write permission is gone.
func (r *replicationManager) clearBackup(resource mosiResource) error {
	r.mu.Lock()
	backup, exists := r.backups[r.resourceKey(resource)]
	r.mu.Unlock()
	if !exists {
		return nil
	}
	if r.config != nil {
		// Native UUID merging can coalesce independently completed copies into
		// one runtime cache entry. Promotion must retire all persisted placements
		// of that same chain, otherwise an older copy requests recovery again
		// after restart. Keep unrelated resources and node identities intact.
		keys := map[string]bool{backup.Resource.key(): true, r.canonicalResource(resource).key(): true}
		if resource.Shard != "" {
			for _, route := range r.shardRoutes(resource) {
				keys[route.From.key()] = true
			}
		}
		names := make([]string, 0, len(keys))
		for name := range keys {
			names = append(names, name)
		}
		sort.Strings(names)
		entries := make([]replicaConfigEntry, len(names))
		for i, name := range names {
			entries[i] = replicaConfigEntry{Kind: "backup", Name: name}
		}
		if err := r.writeConfig(entries...); err != nil {
			return err
		}
	}
	r.mu.Lock()
	delete(r.backups, r.resourceKey(resource))
	r.mu.Unlock()
	return nil
}
