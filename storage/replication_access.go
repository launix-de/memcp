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
	// Each schema/logical shard has two independent authority slots. Both use
	// the same MOSI implementation; neither state is a persisted directory.
	//
	// clusterstate is this node's OUTWARD permission, including authority
	// delegated to its descendants. ownstate alone admits LOCAL engine access
	// and owns the existing active/closing M writer substate. For example, a
	// master can have clusterstate=M and ownstate=IO while its slave holds M.
	// Reading clusterstate must never authorize a local query or backend load.
	//
	// Today a replication master is the root of its domain (no cluster peers),
	// and a slave is a leaf whose two stable permissions agree. A future cluster
	// adapter must acquire the outward slot before delegating inward, and drain
	// all inward holders before acknowledging an outward recall. A child group
	// repeats this boundary; these are not two global flags for an entire tree.
	//
	// Publication raises clusterstate BEFORE raising ownstate, and lowers
	// ownstate BEFORE lowering clusterstate. Thus queries need only the existing
	// ownstate atomic read, with no second lock, CAS or reader registration.
	clusterstate      mosi.State
	ownstate          mosi.State
	resource          atomic.Pointer[mosiResource]
	current           atomic.Pointer[storageShard]
	bootstrap         sync.Mutex
	initialized       atomic.Bool
	bootstrapStarted  bool
	directory         sync.Mutex
	checkpoint        sync.Mutex // database backup order; never acquired by local readers/writers
	sendMu            sync.Mutex // source generation streams for this resource only
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
	if r := activeReplication(table.schema); r != nil && validReplicaDatabase(table.schema.Name) {
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
	if r.cluster != nil {
		return r.cluster.obtain(resource, wanted)
	}
	if r.retired.Load() {
		panic("replica retired by administrator")
	}
	c := r.resourceCell(resource)
	phase := c.ownstate.Phase()
	if phase == mosi.M || wanted != mosi.M && phase.Readable() {
		return c
	}
	if r.config != nil && ((!r.master && !r.ready.Load()) || !r.resourceKnown(resource)) {
		panic(mosi.NetworkError)
	}
	// A master owns previously undelegated local resources. Declare that fact
	// before publishing M; all further changes use normal messages.
	if r.master && !c.ownstate.Initialized() {
		c.acquire.Lock()
		if !c.ownstate.Initialized() {
			record := r.updateRecord(resource, func(record *replicaRecord) bool {
				if record.Owner != "" {
					return false
				}
				record.Resource, record.State, record.Owner, record.Epoch = resource, "M", r.id, 1
				return true
			})
			// This is root-domain bootstrap, not permission inferred from storage.
			// resourceKnown above prevents a retained backup from authorizing recovery.
			c.clusterstate.Establish(mosi.M)
			c.ownstate.RestoreBackend(mosi.ParsePhase(record.State))
		}
		c.acquire.Unlock()
	}
	for {
		publication, action := c.ownstate.Request(wanted)
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
						requested := mosi.S
						if packet.Kind == "write" {
							requested = mosi.M
						}
						c.ownstate.Fail(packet.ID, err, requested)
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
				c.ownstate.Fail(publication.Flight().ID(), err, mosi.IO)
			}
		}
		var disconnected <-chan struct{}
		if !r.master {
			if peer := r.peer(""); peer != nil {
				disconnected = peer.done
			} else {
				c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), mosi.IO)
			}
		}
		select {
		case <-publication.Flight().Done():
			if publication.Flight().Err() != nil {
				panic(mosi.NetworkError)
			}
		case <-disconnected:
			c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), mosi.IO)
			panic(mosi.NetworkError)
		case <-r.done:
			c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), mosi.IO)
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
		if !c.ownstate.Phase().Readable() {
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
	claimed := r.directory.Records[r.recordKey(resource)].InitialClaimed
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
	if !c.ownstate.Phase().Readable() {
		resource := *c.resource.Load()
		wanted := mosi.S
		if !r.master && !r.resourceCell(mosiResource{Database: resource.Database}).initialized.Load() {
			r.mu.Lock()
			claimed := r.directory.Records[r.recordKey(resource)].InitialClaimed
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

// Native transaction ownership already tracks every modified generation.
// Reuse that proof for handover; no transaction-wide replication lock, counter
// or duplicate per-transaction resource directory is maintained.
func (c *replicaCell) writesDrained() bool {
	if shard := c.current.Load(); shard != nil && shard.activeTransactions.Load() != 0 {
		return false
	}
	if previous := c.retiredShards.Load(); previous != nil {
		for _, entry := range *previous {
			if shard := entry.Value(); shard != nil && shard.activeTransactions.Load() != 0 {
				return false
			}
		}
	}
	return true
}

// Handover reuses the native transaction completion notification. Historical
// generations stay reachable while waiting; continuation batches can finish
// normally without registering another MOSI transaction lifetime.
func (c *replicaCell) drainWrites(ctx context.Context) error {
	groups := make(map[*table][]*storageShard)
	add := func(shard *storageShard) {
		if shard != nil {
			groups[shard.t] = append(groups[shard.t], shard)
		}
	}
	add(c.current.Load())
	if history := c.retiredShards.Load(); history != nil {
		for _, entry := range *history {
			add(entry.Value())
		}
	}
	for table, shards := range groups {
		if err := table.waitForTransactions(ctx, shards); err != nil {
			return err
		}
	}
	return nil
}

func (c *replicaCell) transactionContinues(tx *TxContext) bool {
	if tx == nil {
		return false
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	for shard := range tx.shards {
		if shard.coherence.Load() == c {
			return true
		}
	}
	return false
}

// Write admission is a local substate of M, once per batch. Independent writer
// transactions serialize; nested work may borrow its explicit transaction owner.
// Only handover consults the existing native transaction lifetime. Reads do not
// acquire a lease and network streaming never retains an active writer.
func (r *replicationManager) writeResource(resource mosiResource, tx *TxContext) (*replicaCell, func()) {
	c := r.resourceCell(resource)
	if resource.Shard != "" && tx != nil {
		schema := mosiResource{Database: resource.Database}
		// A foreign DDL owner may already hold schema M while waiting for our
		// native data transaction. Its continuation retains its old declarations
		// and must finish before that DDL, rather than deadlock reacquiring S.
		if !r.resourceCell(schema).ownstate.Phase().Readable() && !c.transactionContinues(tx) {
			r.obtain(schema, mosi.S, replicationUser(tx))
		}
	}
	var owner any
	if tx != nil {
		owner = tx
	}
	for {
		r.obtain(resource, mosi.M, replicationUser(tx))
		continuation := c.ownstate.ClosingWrites() && c.transactionContinues(tx)
		writer, wait := c.ownstate.TryWrite(owner, continuation)
		if writer != nil {

			return c, writer.Release
		}
		if wait != nil {
			select {
			case <-wait:
			case <-r.done:
				panic(mosi.NetworkError)
			}
		}
	}
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
		if record.Resource.Database == name && record.Resource.Shard != "" && record.State == "M" {
			resources = append(resources, record.Resource)
		}
	}
	r.mu.Unlock()
	sort.Slice(resources, func(i, j int) bool { return resources[i].key() < resources[j].key() })
	if guard.ownstate.Phase() == mosi.M {
		// Recovery may restore live schema rights before its RAM directory
		// record. Always introduce new tables before sending their data.
		schema := mosiResource{Database: name}
		resources = append([]mosiResource{schema}, resources...)
		resources = append(resources, schema)
	}
	for index, resource := range resources {
		cell := r.resourceCell(resource)
		resource = *cell.resource.Load() // schema publication may have renamed its table
		sent := func() bool {
			// Serialize generations sent for this resource, never local writes.
			if !cell.sendMu.TryLock() {
				return false
			}
			defer cell.sendMu.Unlock()
			releasePersistence, err := replicaPersistenceRead(resource)
			if err != nil {
				scm.PrintError(err)
				return false
			}
			defer releasePersistence()
			handover := cell.ownstate.TryHandover(cell.writesDrained)
			if handover == nil {
				return false
			}
			locked := true
			defer func() {
				if locked {
					handover.Release()
				}
			}()
			if cell.ownstate.Phase() == mosi.IS {
				r.obtain(resource, mosi.M, user)
			}
			if cell.ownstate.Phase() != mosi.M {
				return false
			}
			generation, err := r.prepareResource(resource)
			if err != nil {
				scm.PrintError(err)
				return false
			}
			r.localRecord(resource, mosi.M, generation)
			r.mu.Lock()
			epoch := r.directory.Records[r.recordKey(resource)].Epoch
			r.mu.Unlock()
			transfer, err := r.captureResource(resource, generation)
			if err != nil {
				scm.PrintError(err)
				return false
			}
			handover.Release()
			locked = false
			releasePersistence()
			if err := r.sendResource(peer, mosiPacket{ID: uuid.NewString(), Resource: resource, Epoch: epoch, Generation: generation, User: user, Hash: auth.Hash}, transfer); err != nil {
				scm.PrintError(err)
				return false
			}
			return true
		}()
		if index == 0 && resource.Shard == "" && !sent {
			return
		}
	}
}
