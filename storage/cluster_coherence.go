/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "fmt"
import "time"
import "context"
import "encoding/json"
import "github.com/launix-de/memcp/storage/mosi"

// Owner routing, not a permanent database directory: only an O/M holder
// coordinates its shard and keeps the S-reader set. Other nodes retain a URI
// hint only for resources they actually touched. A stale hint forwards a request
// to the new owner; discovery broadcasts only when no hint exists. Small meshes
// route directly; From/To URIs preserve the contract for future neighbour hops.
func (cluster *nodeCluster) peer(id string) *mosiPeer {
	for _, link := range cluster.view.Load().links {
		if link.member.ID == id {
			return link.peer
		}
	}
	return nil
}
func (cluster *nodeCluster) bootstrapCell(resource mosiResource) {
	if _, ok := cluster.bootstrapResources[resource.key()]; !ok {
		return
	}
	c := cluster.manager.resourceCell(resource)
	if c.ownstate.Initialized() {
		return
	}
	c.acquire.Lock()
	defer c.acquire.Unlock()
	if c.ownstate.Initialized() {
		return
	}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Resource, record.Owner, record.State, record.Epoch = resource, cluster.self.ID, "M", 1
		record.Readers = map[string]bool{}
		return true
	})
	c.clusterstate.Establish(mosi.M)
	c.ownstate.Establish(mosi.M)
}
func (cluster *nodeCluster) locate(ctx context.Context, resource mosiResource) (string, error) {
	r := cluster.manager
	cluster.bootstrapCell(resource)
	own := cluster.report(resource)
	phase := mosi.ParsePhase(own.State)
	if phase == mosi.M || phase == mosi.O {
		return r.id, nil
	}
	if own.Owner != "" && own.Owner != r.id {
		return own.Owner, nil
	}
	view := cluster.view.Load()
	if cluster.seed != "" && len(view.members) == 1 {
		return "", fmt.Errorf(mosi.NetworkError)
	}
	owner := ""
	var epoch uint64
	authoritative := false
	reports := map[string]replicaRecord{r.id: own}
	for _, link := range view.links {
		var answer replicaRecord
		if err := cluster.service.control(ctx, link.peer, "cluster.report", resource, &answer); err != nil {
			return "", err
		}
		reports[link.member.ID] = answer
		actual := answer.State == "M" || answer.State == "O"
		if actual {
			if authoritative && owner != link.member.ID {
				return "", fmt.Errorf("conflicting cluster owners")
			}
			owner, epoch, authoritative = link.member.ID, answer.Epoch, true
		} else if !authoritative && answer.Owner != "" && answer.Epoch >= epoch {
			owner, epoch = answer.Owner, answer.Epoch
		}
	}
	if owner == "" {
		return "", fmt.Errorf(mosi.NetworkError)
	}
	if owner == r.id && !authoritative {
		// Recover a restarted owner only from a complete live roster. An unreachable
		// permission holder never disappears because of a timeout or membership hint.
		if len(view.links) != len(view.members)-1 {
			return "", fmt.Errorf(mosi.NetworkError)
		}
		readers := map[string]bool{}
		for id, record := range reports {
			if record.Handoff != "" {
				return "", fmt.Errorf("interrupted owner handover")
			}
			if record.Epoch > epoch && record.Owner != "" && record.Owner != r.id {
				return "", fmt.Errorf("newer foreign ownership")
			}
			if record.State == "S" {
				readers[id] = true
			}
		}
		state := mosi.M
		if len(readers) > 0 {
			state = mosi.O
		}
		r.updateRecord(resource, func(record *replicaRecord) bool {
			record.Resource, record.Owner, record.State, record.Epoch = resource, r.id, state.String(), max(epoch, 1)
			record.Readers = readers
			return true
		})
		c := r.resourceCell(resource)
		c.clusterstate.Establish(state)
		c.ownstate.Establish(state)
	} else {
		r.updateRecord(resource, func(record *replicaRecord) bool {
			record.Resource, record.Owner, record.Epoch = resource, owner, epoch
			record.Readers = nil
			return true
		})
	}
	return owner, nil
}
func (cluster *nodeCluster) obtain(parent context.Context, resource mosiResource, wanted mosi.Phase) *replicaCell {
	r := cluster.manager
	c := r.resourceCell(resource)
	if phase := c.ownstate.Phase(); phase == mosi.M || wanted != mosi.M && phase.Readable() {
		return c
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()
	owner, err := cluster.locate(ctx, resource)
	if err != nil {
		fmt.Println("cluster owner lookup failed:", resource.key(), err)
		panic(mosi.NetworkError)
	}
	for {
		publication, action := c.ownstate.Request(wanted)
		if action == mosi.Ready {
			return c
		}
		if action == mosi.Send || action == mosi.Upgrade {
			packet := mosiPacket{Kind: "read", Resource: resource, From: r.id, To: owner, ID: publication.Flight().ID(), Sequence: r.requestSequence(publication.Flight())}
			if publication.Phase() == mosi.MR {
				packet.Kind = "write"
			}
			if owner == r.id {
				go func() {
					// The shared MR/SR flight belongs to the protocol. One waiter
					// cancelling cannot revoke every other reader's acquisition.
					operation, finish := context.WithTimeout(cluster.ctx, 30*time.Minute)
					defer finish()
					if failure := r.coordinate(operation, r.id, packet); failure != nil {
						c.ownstate.Fail(packet.ID, failure, wanted)
					}
				}()
			} else {
				peer := cluster.peer(owner)
				if peer == nil {
					err = fmt.Errorf(mosi.NetworkError)
				} else {
					err = peer.post(packet)
				}
				if err != nil {
					c.ownstate.Fail(packet.ID, err, wanted)
				}
			}
		}
		var disconnected <-chan struct{}
		if owner != r.id {
			if peer := cluster.peer(owner); peer != nil {
				disconnected = peer.done
			} else {
				c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), wanted)
			}
		}
		select {
		case <-publication.Flight().Done():
			if publication.Flight().Err() != nil {
				fmt.Println("cluster acquisition failed:", resource.key(), publication.Flight().Err())
				panic(mosi.NetworkError)
			}
		case <-ctx.Done():
			// Leave the shared flight alive for other waiters and the node
			// reader to complete. Cancellation is local to this invocation.
			panic(ctx.Err())
		case <-disconnected:
			c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), wanted)
			panic(mosi.NetworkError)
		}
	}
}

func (cluster *nodeCluster) report(resource mosiResource) replicaRecord {
	cluster.bootstrapCell(resource)
	if _, known := cluster.states.Load(resource.key()); !known {
		return replicaRecord{Resource: resource, State: "IO"}
	}
	r := cluster.manager
	r.mu.Lock()
	record := r.directory.Records[r.recordKey(resource)]
	r.mu.Unlock()
	c := r.resourceCell(resource)
	publication := c.ownstate.Snapshot()
	phase := publication.Phase()
	if phase == mosi.IS {
		phase = publication.Backend()
	}
	if phase.Requested() {
		phase = publication.Retained()
	}
	record.State = phase.String()
	record.Resource = resource
	return record
}

func (cluster *nodeCluster) receive(source string, packet *mosiPacket, connection *mosiPeer) error {
	if !cluster.databases[packet.Resource.Database] {
		return fmt.Errorf("resource outside cluster domain")
	}
	if packet.From != "" {
		source = packet.From
	}
	if packet.To != "" && packet.To != cluster.self.ID {
		peer := cluster.peer(packet.To)
		if peer == nil {
			return fmt.Errorf(mosi.NetworkError)
		}
		return peer.post(*packet)
	}
	r := cluster.manager
	switch packet.Kind {
	case "reference":
		return cluster.installReference(source, *packet)
	case "read", "write":
		copy := *packet
		connection.workers.Add(1)
		go func() {
			defer connection.workers.Done()
			ctx, cancel := mosiPacketContext(connection)
			defer cancel()
			var err error
			func() {
				defer func() {
					if failure := recover(); failure != nil {
						err = fmt.Errorf("%v", failure)
					}
				}()
				owner, lookupErr := cluster.locate(ctx, copy.Resource)
				if lookupErr != nil {
					err = lookupErr
					return
				}
				if owner == r.id {
					err = r.coordinate(ctx, source, copy)
					return
				}
				peer := cluster.peer(owner)
				if peer == nil {
					err = fmt.Errorf(mosi.NetworkError)
					return
				}
				copy.From, copy.To = source, owner
				err = peer.post(copy)
			}()
			if err != nil {
				fmt.Println("cluster request failed:", copy.Kind, copy.Resource.key(), err)
				state := "S"
				if copy.Kind == "write" {
					state = "M"
				}
				if peer := cluster.peer(source); peer != nil {
					_ = peer.post(mosiPacket{Kind: "failure", ID: copy.ID, Resource: copy.Resource, State: state, Error: err.Error()})
				}
			}
		}()
		return nil
	}
	return r.receivePacket(source, packet)
}
func (cluster *nodeCluster) declareShard(shard *storageShard, local bool) *replicaCell {
	r := cluster.manager
	resource := mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()}
	c := r.resourceCell(resource)
	c.current.CompareAndSwap(nil, shard)
	shard.coherence.CompareAndSwap(nil, c)
	if !local || c.ownstate.Initialized() {
		return c
	}
	// New shards are created under schema M, and immediately belong to their
	// creator. There is no registration at a permanent seed/directory server.
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Resource, record.Owner, record.State, record.Epoch, record.Generation = resource, r.id, "M", 1, shard.uuid.String()
		record.Readers = map[string]bool{}
		return true
	})
	c.clusterstate.Establish(mosi.M)
	c.ownstate.Establish(mosi.M)
	return c
}
func (cluster *nodeCluster) installReference(source string, packet mosiPacket) error {
	r := cluster.manager
	c := r.resourceCell(packet.Resource)
	if packet.Epoch < cluster.report(packet.Resource).Epoch {
		return fmt.Errorf("stale shared reference")
	}
	var expected *replicaStep
	if value, exists := r.steps.Load(packet.ID); exists {
		expected = value.(*replicaStep)
		if !expected.matches(source, packet) {
			return fmt.Errorf("unexpected shared reference source")
		}
	}
	if expected == nil && !c.ownstate.Phase().Requested() {
		// An O holder may publish a rebuilt generation to its known S readers
		// without a new read request. Keep their existing permission and install
		// only an update from the recorded owner at this exact authority epoch.
		r.mu.Lock()
		record := r.directory.Records[r.recordKey(packet.Resource)]
		r.mu.Unlock()
		if c.ownstate.Phase() != mosi.S || record.Owner != source || record.Epoch != packet.Epoch {
			return fmt.Errorf("unsolicited shared reference for %s in %s", packet.Resource.key(), c.ownstate.Phase())
		}
	}
	db := GetDatabase(packet.Resource.Database)
	if db == nil {
		return fmt.Errorf("missing shared database")
	}
	if packet.Resource.Shard == "" {
		data := db.persistence.ReadSchema()
		generation, err := replicaSchemaGeneration(data)
		if err != nil || generation != packet.Generation {
			return fmt.Errorf("shared schema changed")
		}
		if err := r.installSchema(source, db.Name, db.persistence, data); err != nil {
			return err
		}
	} else {
		if err := db.importClusterCommits(source); err != nil {
			return err
		}
		if err := r.finishImport(source, &replicaImport{header: packet, engine: db.persistence}); err != nil {
			return err
		}
	}
	r.updateRecord(packet.Resource, func(record *replicaRecord) bool { record.Generation = packet.Generation; return true })
	if expected != nil {
		expected.finish(nil)
		return nil
	}
	peer := r.peer(source)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	return peer.post(mosiPacket{Kind: "ack", ID: packet.ID, Resource: packet.Resource, Epoch: packet.Epoch, Generation: packet.Generation})
}

// Bootstrap is explicit initialization of a fresh unjoined installation, never
// recovery of a restarted node. Snapshot only the resources present at startup;
// new shards created later by another owner cannot be fabricated from filenames.
func (cluster *nodeCluster) prepareBootstrap(fresh bool) error {
	cluster.bootstrapResources = make(map[string]mosiResource)
	if !fresh || cluster.seed != "" {
		return nil
	}
	for name := range cluster.databases {
		db := GetDatabase(name)
		if db == nil {
			continue
		}
		schema := mosiResource{Database: name}
		cluster.bootstrapResources[schema.key()] = schema
		var snapshot database
		data := db.persistence.ReadSchema()
		if len(data) == 0 {
			continue
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return err
		}
		for _, table := range snapshot.tables.GetAll() {
			if table.isEphemeralQueryTable() {
				continue
			}
			for _, shard := range table.ActiveShards() {
				if shard != nil {
					resource := mosiResource{Database: name, Table: table.Name, Shard: shard.uuid.String()}
					cluster.bootstrapResources[resource.key()] = resource
				}
			}
		}
	}
	return nil
}

// Shared schema construction consumes exactly the publisher's captured JSON.
// This is an invocation-local read view of the existing backend, not another
// persistence backend or a second on-disk catalog. Receivers never rewrite it.
type clusterSchemaView struct {
	PersistenceEngine
	data []byte
}

func (view clusterSchemaView) ReadSchema() []byte { return view.data }

func (cluster *nodeCluster) initializeManager() {
	r := &replicationManager{cluster: cluster, master: true, id: cluster.self.ID, boot: cluster.self.Boot, done: make(chan struct{}), cells: map[string]*replicaCell{}, backups: map[string]replicaBackup{}, confirmed: map[string]string{}, initialClaims: map[string]map[string]bool{}, directory: replicaDirectory{Records: map[string]replicaRecord{}, Users: map[string]replicaAuth{}, Retired: map[string]bool{}, Initialized: map[string]bool{}}}
	cluster.manager = r
}

// A rebuild publishes a new shard UUID. Both native shard objects retain the
// same local permission cell while mutation forwarding is live; old UUIDs are
// routing aliases for in-flight operations, never independent writer grants.
func (cluster *nodeCluster) aliasShard(resource mosiResource, shard *storageShard) {
	r := cluster.manager
	c := r.resourceCell(resource)
	next := mosiResource{Database: resource.Database, Table: resource.Table, Shard: shard.uuid.String()}
	if current := c.resource.Load(); current != nil && current.Shard == next.Shard {
		cluster.states.Store(next.key(), c)
		return
	}
	if next.Shard == resource.Shard {
		return
	}
	oldKey := r.recordKey(resource)
	r.mu.Lock()
	record := r.directory.Records[oldKey]
	record.Resource = next
	r.directory.Records[next.key()] = record
	delete(r.directory.Records, oldKey)
	// Canonical identity and its directory entry publish under the existing
	// directory owner. A concurrent old-handle update must resolve the new key
	// after acquiring mu, rather than resurrecting a stale directory entry.
	cluster.states.Store(next.key(), c)
	c.resource.Store(&next)
	r.mu.Unlock()
}
