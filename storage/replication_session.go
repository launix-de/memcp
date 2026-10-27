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
import "weak"
import "github.com/launix-de/memcp/scm"

// Reads use the engine's ordinary immutable shard/catalog snapshots. They do
// not register admission, increment a counter, hold a lock, or rebuild/checkpoint.
// Native transaction generations retain uncommitted writes. The MOSI local
// writer is batch-scoped; neither read-only nor idle transactions hold a gate.
func replicationEnter(tx *TxContext, session scm.Scmer) {
	replicationEnterMode(tx, session, false)
}

func replicationLeave(tx *TxContext) {
	tx.replicationWriting, tx.replicationActive = false, false
}

func replicaCanRead(state *string) bool {
	return state != nil && (*state == "M" || *state == "O" || *state == "S")
}

func replicationEnterMode(tx *TxContext, session scm.Scmer, write bool) {
	r := replication.Load()
	if r == nil {
		return
	}
	name := session.Func()(replicationSchemaArgument...).String()
	// Shared snapshots have their own S-only domain. Opening a local query
	// must not subscribe/claim the same namespace through the replication parent.
	if !validReplicaDatabase(name) || sharedClusterDatabase(name) {
		return
	}
	r.firstDatabase(name, session)
	tx.replicationActive = true
	if write {
		tx.replicationWriting = true
	}
}

// Called before a parsed write plan executes, including on plan-cache hits.
func replicationWrite(tx *TxContext) {
	if replication.Load() != nil && tx != nil {
		tx.replicationWriting = true
	}
}

func replicationCheckWriteTable(t *table, tx *TxContext) {
	if !validReplicaDatabase(t.schema.Name) || t.isEphemeralQueryTable() || activeReplication(t.schema) == nil {
		return
	}
	if t.schema.tables.Get(t.Name) != t {
		panic("replication schema changed; retry statement")
	}
}

// Weak retirement metadata is updated only at content publication. A query
// retaining a historical shard keeps its files alive without registering reads.
func replicationRetainsCatalogs(name string) bool {
	if sharedClusterDatabase(name) {
		return true
	} // cluster-wide historical-generation GC is not a local pointer test
	if r := replication.Load(); r != nil {
		if r.hasTransfers(name) {
			return true
		}
		if cells := r.cellSnapshot.Load(); cells != nil {
			for _, cell := range *cells {
				resource := cell.resource.Load()
				if resource == nil || resource.Database != name {
					continue
				}
				if history := cell.retiredShards.Load(); history != nil {
					for _, previous := range *history {
						if previous.Value() != nil {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (c *replicaCell) retainShard(shard *storageShard) {
	if shard == nil {
		return
	}
	for {
		previous := c.retiredShards.Load()
		next := make([]weak.Pointer[storageShard], 0)
		if previous != nil {
			for _, entry := range *previous {
				if entry.Value() == shard {
					return
				}
				if entry.Value() != nil {
					next = append(next, entry)
				}
			}
		}
		next = append(next, weak.Make(shard))
		if c.retiredShards.CompareAndSwap(previous, &next) {
			return
		}
	}
}

func replicationOpenDatabase(user, database string) (err error) {
	if replication.Load() == nil || !validReplicaDatabase(database) || sharedClusterDatabase(database) {
		return nil
	}
	tx := &TxContext{}
	defer func() {
		if tx.replicationActive {
			replicationLeave(tx)
		}
		if value := recover(); value != nil {
			err = fmt.Errorf("%v", value)
		}
	}()
	session := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		switch a[0].String() {
		case "username":
			return scm.NewString(user)
		case "schema":
			return scm.NewString(database)
		}
		return scm.NewNil()
	})
	replicationEnter(tx, session)
	return nil
}

func replicationSessionClosed(session scm.Scmer) {
	if replication.Load() == nil {
		return
	}
	value := session.Func()(scm.NewString("__memcp_tx"))
	if value.IsNil() {
		return
	}
	tx, ok := value.Any().(*TxContext)
	if !ok {
		return
	}
	tx.queryMu.Lock()
	defer tx.queryMu.Unlock()
	defer func() {
		if tx.replicationActive {
			replicationLeave(tx)
			tx.replicationActive = false
		}
		if err := recover(); err != nil {
			scm.PrintError(err)
		}
	}()
	if tx.State == TxActive {
		tx.Rollback()
	}
}
