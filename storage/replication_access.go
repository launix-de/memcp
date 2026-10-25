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
import "sort"
import "sync"
import "weak"
import "time"
import "context"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

type replicaCell struct {
	// gate protects uncommitted local writes against ownership transfer.
	// Analytics never participate: their existing snapshots remain valid.
	gate              sync.RWMutex // write transactions only; reads use existing snapshots
	rights            mosi.State
	resource          atomic.Pointer[mosiResource]
	current           atomic.Pointer[storageShard]
	bootstrap         sync.Mutex
	initialized       atomic.Bool
	bootstrapStarted  bool
	directory         sync.Mutex
	checkpoint        sync.Mutex
	importPublication sync.Mutex // schema cell: import registration/publication versus GC
	acquire           sync.Mutex
	retiredShards     atomic.Pointer[[]weak.Pointer[storageShard]]
}

func (table *table) schemaWrite() (*table, WriteAccess) {
	if table.isEphemeralQueryTable() {
		return table, WriteAccess{}
	}
	access := table.schema.GetWrite(nil)
	complete := false
	defer func() {
		if !complete {
			access.Release()
		}
	}()
	if r := replication.Load(); r != nil && validReplicaDatabase(table.schema.Name) {
		current := table.schema.tables.Get(table.Name)
		if current == nil {
			panic("replication schema changed; retry statement")
		}
		table = current
		// Schema M revokes foreign schema readers. Acquire the table's current
		// generations before changing declarations, so a schema import cannot
		// replace a remote modifier's untransferred deltas with an old backup.
		for _, shard := range table.ActiveShards() {
			if shard != nil {
				r.obtain(*r.shardCell(shard).resource.Load(), mosi.M, "")
			}
		}
	}
	complete = true
	return table, access
}

func (r *replicationManager) requestSequence(flight *mosi.Flight) uint64 {
	if serial := flight.Sequence(); serial != 0 {
		return serial
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if serial := flight.Sequence(); serial != 0 {
		return serial
	}
	r.directory.Sequence++
	flight.SetSequence(r.directory.Sequence)
	return r.directory.Sequence
}

func replicationUser(tx *TxContext) string {
	if tx != nil && !tx.Session.IsNil() {
		return tx.Session.Func()(replicationUsernameArgument...).String()
	}
	return ""
}

func (r *replicationManager) proof(user, database string) (string, replicaAuth, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user != "" {
		auth, exists := r.directory.Users[user]
		if exists && (auth.Admin || auth.Databases[database]) {
			return user, auth, nil
		}
		return "", replicaAuth{}, fmt.Errorf("database access denied")
	}
	for name, auth := range r.directory.Users {
		if auth.Admin || auth.Databases[database] {
			return name, auth, nil
		}
	}
	return "", replicaAuth{}, fmt.Errorf("no authenticated user for database")
}

func (r *replicationManager) obtain(resource mosiResource, wanted mosi.Phase, user string) *replicaCell {
	if r.retired.Load() {
		panic("replica retired by administrator")
	}
	c := r.resourceCell(resource)
	phase := c.rights.Phase()
	if phase == mosi.M || wanted != mosi.M && phase.Readable() {
		return c
	}
	if r.config != nil && ((!r.master && !r.ready.Load()) || !r.resourceKnown(resource)) {
		panic(mosi.NetworkError)
	}
	// A master owns previously undelegated local resources. Declare that fact
	// before publishing M; all further changes use normal messages.
	if r.master && !c.rights.Initialized() {
		c.acquire.Lock()
		if !c.rights.Initialized() {
			record := r.updateRecord(resource, func(record *replicaRecord) bool {
				if record.Owner != "" {
					return false
				}
				record.Resource, record.State, record.Owner, record.Epoch = resource, "M", r.id, 1
				return true
			})
			c.rights.RestoreBackend(mosi.ParsePhase(record.State))
		}
		c.acquire.Unlock()
	}
	for {
		publication, action := c.rights.Request(wanted)
		if action == mosi.Ready {
			return c
		}
		if action == mosi.Send || action == mosi.Upgrade {
			serial := r.requestSequence(publication.Flight())
			var err error
			packet := mosiPacket{Kind: "read", ID: publication.Flight().ID(), Resource: resource, Sequence: serial}
			if publication.Phase() == mosi.MR {
				packet.Kind = "write"
			}
			if r.master {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
					defer cancel()
					if err := r.coordinate(ctx, r.id, packet); err != nil {
						fmt.Println("replication acquisition failed:", packet.Kind, packet.Resource.key(), err)
						c.rights.Fail(packet.ID, err)
					}
				}()
			} else {
				var auth replicaAuth
				packet.User, auth, err = r.proof(user, resource.Database)
				packet.Hash = auth.Hash
				peer := r.peer("")
				if peer == nil {
					err = fmt.Errorf(mosi.NetworkError)
				} else if err == nil {
					err = peer.post(packet)
				}
			}
			if err != nil {
				c.rights.Fail(publication.Flight().ID(), err)
			}
		}
		var disconnected <-chan struct{}
		if !r.master {
			if peer := r.peer(""); peer != nil {
				disconnected = peer.done
			} else {
				c.rights.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError))
			}
		}
		select {
		case <-publication.Flight().Done():
			if publication.Flight().Err() != nil {
				panic(mosi.NetworkError)
			}
		case <-disconnected:
			c.rights.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError))
			panic(mosi.NetworkError)
		case <-r.done:
			c.rights.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError))
			panic(mosi.NetworkError)
		}
	}
}

// Initialization uses the same per-resource state machine as ordinary access.
// After schema M arrives, shard M requests proceed independently: a query waits
// only for the shard it needs. Durable per-item claims resume interrupted setup
// without claiming back shards that have already become shared elsewhere.
func (r *replicationManager) firstDatabase(name string, session scm.Scmer) {
	if r.retired.Load() {
		panic("replica retired by administrator")
	}
	resource := mosiResource{Database: name}
	c := r.resourceCell(resource)
	if r.master || c.initialized.Load() {
		if !c.rights.Phase().Readable() {
			r.obtain(resource, mosi.S, session.Func()(replicationUsernameArgument...).String())
		}
		return
	}
	user := session.Func()(replicationUsernameArgument...).String()
	c.bootstrap.Lock()
	defer c.bootstrap.Unlock()
	if c.initialized.Load() || c.bootstrapStarted {
		r.obtain(resource, mosi.S, user)
		return
	}
	r.mu.Lock()
	claimed := r.directory.Records[resource.key()].InitialClaimed
	r.mu.Unlock()
	wanted := mosi.M
	if claimed {
		wanted = mosi.S
	}
	r.obtain(resource, wanted, user)
	db := databases.Get(name)
	if db == nil {
		panic("missing initial schema")
	}
	resources := make([]mosiResource, 0)
	for _, table := range db.tables.GetAll() {
		if table.isEphemeralQueryTable() {
			continue
		}
		for _, shard := range table.ActiveShards() {
			if shard == nil {
				continue
			}
			item := mosiResource{Database: name, Table: table.Name, Shard: shard.logicalID().String()}
			r.mu.Lock()
			claimed := r.directory.Records[item.key()].InitialClaimed
			r.mu.Unlock()
			if !claimed {
				resources = append(resources, item)
			}
		}
	}
	c.bootstrapStarted = true
	go func() {
		var pending sync.WaitGroup
		var failed atomic.Bool
		for _, item := range resources {
			pending.Add(1)
			go func(item mosiResource) {
				defer pending.Done()
				defer func() {
					if recover() != nil {
						failed.Store(true)
					}
				}()
				r.obtain(item, mosi.M, user)
			}(item)
		}
		pending.Wait()
		c.bootstrap.Lock()
		defer c.bootstrap.Unlock()
		c.bootstrapStarted = false
		if failed.Load() {
			return
		}
		r.mu.Lock()
		if r.directory.Initialized == nil {
			r.directory.Initialized = make(map[string]bool)
		}
		r.directory.Initialized[name] = true
		r.mu.Unlock()
		c.initialized.Store(true)
	}()
}

func (r *replicationManager) readShard(shard *storageShard, tx *TxContext) *storageShard {
	c := r.shardCell(shard)
	if !c.rights.Phase().Readable() {
		resource := *c.resource.Load()
		wanted := mosi.S
		if !r.master && !r.resourceCell(mosiResource{Database: resource.Database}).initialized.Load() {
			r.mu.Lock()
			claimed := r.directory.Records[resource.key()].InitialClaimed
			r.mu.Unlock()
			// A query may beat its bootstrap worker to this shard. Its first
			// acquisition still belongs to initialization and must request M;
			// it must not return an S grant while the initial M is unqueued.
			if !claimed {
				wanted = mosi.M
			}
		}
		r.obtain(resource, wanted, replicationUser(tx))
	}
	if current := c.current.Load(); current != nil {
		return current
	}
	return shard
}

// Write leases are transaction-owned and charged once per resource. They are
// absent from readers and row loops; the transfer waits for commit/rollback.
func (r *replicationManager) writeResource(resource mosiResource, tx *TxContext) (*replicaCell, func()) {
	c := r.resourceCell(resource)
	if tx != nil {
		tx.mu.Lock()
		owned := tx.replicationCells[c]
		tx.mu.Unlock()
		if owned {
			return c, func() {}
		}
	}
	// A statement which lost schema permission during DDL must reacquire it
	// before obtaining a new data write right. Existing write owners retain
	// their transaction lease; DDL waits for their committed shard handover.
	if resource.Shard != "" && tx != nil {
		r.obtain(mosiResource{Database: resource.Database}, mosi.S, replicationUser(tx))
	}
	for {
		r.obtain(resource, mosi.M, replicationUser(tx))
		c.gate.RLock()
		if c.rights.Phase() == mosi.M {
			break
		}
		c.gate.RUnlock()
	}
	if tx == nil {
		return c, c.gate.RUnlock
	}
	tx.mu.Lock()
	if tx.replicationCells == nil {
		tx.replicationCells = make(map[*replicaCell]bool)
	}
	if tx.replicationCells[c] {
		c.gate.RUnlock()
	} else {
		tx.replicationCells[c] = true
	}
	tx.replicationWriting = true
	tx.mu.Unlock()
	return c, func() {}
}

func (r *replicationManager) backupDatabase(name, user string) {
	defer func() {
		if failure := recover(); failure != nil {
			scm.PrintError(fmt.Errorf("replication backup deferred: %v", failure))
		}
	}()
	peer := r.peer("")
	if peer == nil {
		return
	}
	guard := r.resourceCell(mosiResource{Database: name})
	if !guard.checkpoint.TryLock() {
		return
	}
	defer guard.checkpoint.Unlock()
	user, auth, err := r.proof(user, name)
	if err != nil {
		return
	}
	r.mu.Lock()
	resources := make([]mosiResource, 0)
	for _, record := range r.directory.Records {
		if record.Resource.Database == name && record.State == "M" {
			resources = append(resources, record.Resource)
		}
	}
	r.mu.Unlock()
	sort.Slice(resources, func(i, j int) bool { return resources[i].key() < resources[j].key() })
	if guard.rights.Phase() == mosi.M {
		resources = append(resources, mosiResource{Database: name})
	}
	for _, resource := range resources {
		cell := r.resourceCell(resource)
		resource = *cell.resource.Load() // schema publication may have renamed its table
		func() {
			releasePersistence, err := replicaPersistenceRead(resource)
			if err != nil {
				scm.PrintError(err)
				return
			}
			defer releasePersistence()
			if !cell.gate.TryLock() {
				return
			}
			defer cell.gate.Unlock()
			if cell.rights.Phase() == mosi.IS {
				r.obtain(resource, mosi.M, user)
			}
			if cell.rights.Phase() != mosi.M {
				return
			}
			generation, err := r.prepareResource(resource)
			if err != nil {
				scm.PrintError(err)
				return
			}
			r.localRecord(resource, mosi.M, generation)
			r.mu.Lock()
			epoch := r.directory.Records[resource.key()].Epoch
			r.mu.Unlock()
			releasePersistence()
			if err := r.sendResource(peer, mosiPacket{ID: uuid.NewString(), Resource: resource, Epoch: epoch, Generation: generation, User: user, Hash: auth.Hash}); err != nil {
				scm.PrintError(err)
			}
		}()
	}
}
