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

var errClusterReadBusy = fmt.Errorf("cluster read acquisition would wait for a writer")

// tryRead acquires ordinary MOSI S without materializing a payload. A cache
// needs these rights even when its producer delegates all shard work. If a
// writer cannot relinquish M, the existing cache scope uses its direct producer
// instead of publishing a shared value from an unprotected source.
func (cluster *nodeCluster) tryRead(ctx context.Context, resource mosiResource) bool {
	c := cluster.manager.resourceCell(resource).resolved()
	if c.ownstate.ReadReady() {
		return true
	}
	restoreCommits := !c.ownstate.Initialized()
	owner, err := cluster.locate(ctx, resource)
	if err != nil {
		panic(err)
	}
	c = c.resolved()
	resource = *c.resource.Load()
	publication, action := c.ownstate.Request(mosi.S)
	if action == mosi.Ready {
		if restoreCommits && resource.Shard != "" {
			db := GetDatabase(resource.Database)
			if db == nil {
				panic(mosi.NetworkError)
			}
			if err := db.importClusterCommits(owner); err != nil {
				panic(err)
			}
		}
		return true
	}
	if action != mosi.Send {
		return false // another acquisition owns this flight; never wait for it
	}
	packet := mosiPacket{Kind: "read", Resource: resource, From: cluster.self.ID,
		To: owner, ID: publication.Flight().ID(), Sequence: cluster.manager.requestSequence(publication.Flight()),
		Boot: cluster.self.Boot, TryRead: true}
	granted := false
	if owner == cluster.self.ID {
		err = cluster.coordinate(ctx, cluster.self.ID, packet)
		granted = err == nil
		if err == errClusterReadBusy {
			err = nil
		}
	} else if peer := cluster.peer(owner); peer != nil {
		err = cluster.service.control(ctx, peer, "cluster.try_read", packet, &granted, true)
	} else {
		err = fmt.Errorf(mosi.NetworkError)
	}
	if err != nil {
		c.ownstate.Fail(packet.ID, err, mosi.S)
		panic(err)
	}
	if !granted || !c.resolved().ownstate.Phase().Readable() {
		c.ownstate.Fail(packet.ID, errClusterReadBusy, mosi.S)
		return false
	}
	return true
}

func (cluster *nodeCluster) handleTryRead(ctx context.Context, source string, data json.RawMessage) (any, error) {
	if err := cluster.requireMember(source); err != nil {
		return nil, err
	}
	var packet mosiPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, err
	}
	if !packet.Resource.valid() || !cluster.databases[packet.Resource.Database] || packet.Kind != "read" ||
		!packet.TryRead || packet.ID == "" || !cluster.requestIncarnation(packet.From, packet.Boot) {
		return nil, fmt.Errorf("invalid opportunistic cluster read")
	}
	owner, err := cluster.locate(ctx, packet.Resource)
	if err != nil {
		return nil, err
	}
	if owner != cluster.self.ID {
		peer := cluster.peer(owner)
		if peer == nil {
			return nil, fmt.Errorf(mosi.NetworkError)
		}
		var granted bool
		err := cluster.service.control(ctx, peer, "cluster.try_read", packet, &granted, true)
		return granted, err
	}
	err = cluster.coordinate(ctx, packet.From, packet)
	if err == errClusterReadBusy {
		return false, nil
	}
	return err == nil, err
}

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
	resource = cluster.manager.canonicalResource(resource)
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
	r.restoreResourceRoutes(resource)
	resource = r.canonicalResource(resource)
	cluster.bootstrapCell(resource)
	own := cluster.report(resource)
	phase := mosi.ParsePhase(own.State)
	if phase == mosi.M || phase == mosi.O {
		return r.id, nil
	}
	if own.Handoff != "" {
		return own.Handoff, nil
	}
	if own.Owner != "" && own.Owner != r.id && cluster.peerAvailable(own.Owner) {
		return own.Owner, nil
	}
	if resource.Shard != "" {
		// Restart recovery of our own former authority still needs authenticated
		// live reports. Ordinary cold placement consults only the schema owner.
		cell := r.resourceCell(resource)
		if own.Owner != r.id || cell.ownstate.Initialized() {
			return cluster.locateViaSchema(ctx, resource)
		}
	}
	view := cluster.view.Load()
	if cluster.seed != "" && len(view.members) == 1 {
		return "", fmt.Errorf(mosi.NetworkError)
	}
	owner := ""
	var epoch uint64
	authoritative := false
	reports := map[string]replicaRecord{r.id: own}
	complete := len(view.links) == len(view.members)-1
	for _, link := range view.links {
		var answer replicaRecord
		probe, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := cluster.service.control(probe, link.peer, "cluster.report", resource, &answer, true)
		cancel()
		if err != nil {
			complete = false
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue // discovery failure neither retires a peer nor grants rights
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
		return "", fmt.Errorf("%s: no live owner for %s", mosi.NetworkError, resource.key())
	}
	// Discovery spans replies from several live nodes. Ownership may have
	// advanced since our initial report. Re-sample routing only: this must never
	// establish a permission from IO or turn an in-flight handover into recovery.
	// An unchanged directory hint must not override a newer live peer report,
	// especially when the hinted owner is unavailable.
	latest := cluster.report(resource)
	latestPhase := mosi.ParsePhase(latest.State)
	if latestPhase == mosi.M || latestPhase == mosi.O {
		return r.id, nil
	}
	if latest.Handoff != "" {
		return latest.Handoff, nil
	}
	if latest.Owner != "" && latest.Owner != r.id && (latest.Owner != own.Owner || latest.Epoch > own.Epoch) {
		return latest.Owner, nil
	}
	if owner == r.id && !authoritative {
		// A live IO publication is a revocation, not a restarted owner. The
		// directory may still name us while a handover awaits installation ACK;
		// reconstructing M here would resurrect revoked rights and stale schema.
		if r.resourceCell(resource).ownstate.Initialized() {
			return "", fmt.Errorf("cluster ownership handover is incomplete")
		}
		// Recover a restarted owner only from a complete live roster. An unreachable
		// permission holder never disappears because of a timeout or membership hint.
		if !complete {
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
		if resource.Shard == "" {
			if err := cluster.recoverSchemaLocations(ctx, resource.Database, view.links); err != nil {
				return "", err
			}
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
		cell := r.resourceCell(resource)
		r.updateRecord(resource, func(record *replicaRecord) bool {
			// Discovery is advisory and spans network waits. A grant/handover
			// may have advanced after the final report too. Never overwrite its
			// owner or reader directory with that older observation.
			phase := cell.resolved().ownstate.Snapshot()
			permission := phase.Phase()
			if permission.Requested() {
				permission = phase.Retained()
			}
			if permission == mosi.M || permission == mosi.O {
				owner = r.id
				return false
			}
			if record.Handoff != "" {
				owner = record.Handoff
				return false
			}
			if record.Owner != own.Owner || record.Epoch != own.Epoch {
				if record.Owner != "" {
					owner = record.Owner
				}
				return false
			}
			record.Resource, record.Owner, record.Epoch = resource, owner, epoch
			return true
		})
	}
	return owner, nil
}
func (cluster *nodeCluster) obtain(parent context.Context, resource mosiResource, wanted mosi.Phase) *replicaCell {
	r := cluster.manager
	c := r.resourceCell(resource)
	if c.ownstate.ReadReady() && (wanted != mosi.M || c.ownstate.Phase() == mosi.M) {
		return c
	}
	restoreCommits := !c.ownstate.Initialized()
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()
	owner, err := cluster.locate(ctx, resource)
	if err != nil {
		fmt.Println("cluster owner lookup failed:", resource.key(), err)
		panic(mosi.NetworkError)
	}
	for {
		c = c.resolved()
		resource = *c.resource.Load()
		publication, action := c.ownstate.Request(wanted)
		if action == mosi.Ready {
			if restoreCommits && resource.Shard != "" {
				// Restart recovery may establish local authority without a new
				// reference packet. Restore native WAL commit proofs here too,
				// before any backend content can be materialized.
				db := GetDatabase(resource.Database)
				if db == nil || db.importClusterCommits(owner) != nil {
					panic(mosi.NetworkError)
				}
			}
			return c
		}
		if action == mosi.Send || action == mosi.Upgrade {
			packet := mosiPacket{Kind: "read", Resource: resource, From: r.id, To: owner, ID: publication.Flight().ID(), Sequence: r.requestSequence(publication.Flight()), Boot: cluster.self.Boot}
			if publication.Phase() == mosi.MR {
				packet.Kind = "write"
			}
			if owner == r.id {
				c := c
				go func() {
					// The shared MR/SR flight belongs to the protocol. One waiter
					// cancelling cannot revoke every other reader's acquisition.
					operation, finish := context.WithTimeout(cluster.ctx, 30*time.Minute)
					defer finish()
					if failure := cluster.coordinate(operation, r.id, packet); failure != nil {
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
		case <-c.forwardingDone():
			continue
		case <-publication.Flight().Done():
			if c.forward.Load() != nil {
				continue
			}
			if publication.Flight().Err() != nil {
				if publication.Flight().Err() == errClusterReadBusy {
					continue // a cache's declined try-read cannot fail a normal waiter
				}
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
	r := cluster.manager
	r.mu.Lock()
	// Routing publication owns identity changes. Sample existence, the current
	// routing binding and its permission under that same directory ownership;
	// separate lookup and permission reads can straddle an authority handover.
	c := r.cells[resource.key()]
	if c == nil {
		r.mu.Unlock()
		return replicaRecord{Resource: resource, State: "IO"}
	}
	c = c.resolved()
	resource = r.canonicalResource(resource)
	record := r.directory.Records[resource.key()]
	// Sample permission while the existing directory owner still protects the
	// routing hint. Copying an old record, unlocking, then observing new IO could
	// omit the successor published before that revocation and falsely report an
	// interrupted handover. This cold report adds no query admission lock.
	publication := c.ownstate.Snapshot()
	r.mu.Unlock()
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
		if packet.Boot == "" || !cluster.requestIncarnation(source, packet.Boot) {
			return fmt.Errorf("stale cluster request incarnation")
		}
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
					err = fmt.Errorf("cluster owner lookup: %w", lookupErr)
					return
				}
				if owner == r.id {
					err = cluster.coordinate(ctx, source, copy)
					if err != nil {
						err = fmt.Errorf("cluster coordinate: %w", err)
					}
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
		record.Resource, record.Owner, record.State, record.Epoch, record.Content = resource, r.id, "M", 1, shard.uuid.String()
		record.Readers = map[string]bool{}
		return true
	})
	c.clusterstate.Establish(mosi.M)
	c.ownstate.Establish(mosi.M)
	return c
}

// Called under the existing resource acquisition lock for both reference data
// and its schema location batches. Forwarded acquisitions may arrive from a
// newer owner than the requester's hint, at the newer native authority epoch.
func (cluster *nodeCluster) referenceSource(source string, packet mosiPacket, c *replicaCell, record replicaRecord) (*replicaStep, error) {
	r := cluster.manager
	if packet.Epoch < record.Epoch {
		return nil, fmt.Errorf("stale shared reference")
	}
	var expected *replicaStep
	if value, exists := r.steps.Load(packet.ID); exists {
		expected = value.(*replicaStep)
		if !expected.matches(source, packet) {
			return nil, fmt.Errorf("unexpected shared reference source")
		}
	}
	publication := c.ownstate.Snapshot()
	if expected == nil {
		if publication.Phase().Requested() {
			if publication.Flight().ID() != packet.ID || record.Owner != source && packet.Epoch <= record.Epoch {
				return nil, fmt.Errorf("unexpected shared reference request: resource=%s incoming=%s epoch=%d pending=%s epoch=%d owner=%s source=%s", packet.Resource.key(), packet.ID, packet.Epoch, publication.Flight().ID(), record.Epoch, record.Owner, source)
			}
		} else if publication.Phase() != mosi.S || record.Owner != source || record.Epoch != packet.Epoch {
			// Unsolicited O updates may refresh existing S readers only at the
			// exact recorded authority epoch. They never create a new grant.
			return nil, fmt.Errorf("unsolicited shared reference for %s in %s", packet.Resource.key(), publication.Phase())
		}
	}
	return expected, nil
}

func (cluster *nodeCluster) installReference(source string, packet mosiPacket) error {
	r := cluster.manager
	if !packet.Resource.valid() || packet.ID == "" {
		return fmt.Errorf("invalid shared reference")
	}
	if err := r.validateForwards(source, packet); err != nil {
		return err
	}
	// Find the already known authority without publishing the advertised edge.
	// A reference may name a successor which is not in our lookup directory yet.
	c := r.lookupCell(packet.Resource)
	if c == nil {
		for _, route := range packet.Forwards {
			if c = r.lookupCell(route.From); c != nil {
				break
			}
		}
	}
	if c == nil {
		return fmt.Errorf("unknown shared reference")
	}
	c.acquire.Lock()
	locked := true
	defer func() {
		if locked {
			c.acquire.Unlock()
		}
	}()
	r.mu.Lock()
	record := r.directory.Records[r.resourceKey(*c.resource.Load())]
	r.mu.Unlock()
	expected, err := cluster.referenceSource(source, packet, c, record)
	if err != nil {
		return err
	}
	if cluster.peer(source) == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	db := GetDatabase(packet.Resource.Database)
	if db == nil {
		return fmt.Errorf("missing shared database")
	}
	var data []byte
	if packet.Resource.Shard == "" {
		data = db.persistence.ReadSchema()
		content, err := r.schemaDigest(data)
		if err != nil || content != packet.Content {
			return fmt.Errorf("shared schema changed")
		}
	} else {
		actual := packet.Resource
		actual.Shard = packet.Content
		if !actual.valid() || db.tables.Get(packet.Resource.Table) == nil {
			return fmt.Errorf("invalid shared shard reference")
		}
		if err := db.importClusterCommits(source); err != nil {
			return err
		}
	}
	// Epoch, authenticated sender, flight and content have all been checked.
	// Publish the observed owner version before merging native UUID cells. A
	// successor may already contain an older invalidation, while this exact
	// predecessor flight is receiving a newer grant. Merging first would discard
	// the pending flight and then ACK a grant that could never install. An older
	// retained read proof does not survive advancement to this owner version.
	if packet.Epoch > record.Epoch {
		c.ownstate.Publish(mosi.IO)
		r.updateRecord(packet.Resource, func(current *replicaRecord) bool {
			current.Epoch = packet.Epoch
			return true
		})
	}
	// Rejected references above leave both lookup and active UUID unchanged.
	for _, route := range packet.Forwards {
		r.forwardResource(route.From, route.To)
	}
	if packet.Resource.Shard == "" {
		if err := r.installSchema(source, db.Name, db.persistence, data); err != nil {
			return err
		}
	} else {
		if err := r.finishImport(source, &replicaImport{header: packet, engine: db.persistence}); err != nil {
			return err
		}
	}
	r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
		record.Content = packet.Content
		// Installation registers the reader before ACK, including an S flight
		// already upgraded to MR. Its later S grant must retain this epoch.
		if packet.Epoch > record.Epoch {
			record.Epoch = packet.Epoch
		}
		return true
	})
	if expected == nil && packet.State != "" {
		if err := r.grantAcquired(c.resolved(), packet); err != nil {
			return err
		}
	}
	c.acquire.Unlock()
	locked = false
	if expected != nil {
		expected.finish(nil)
		return nil
	}
	peer := r.peer(source)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	return peer.post(mosiPacket{Kind: "ack", ID: packet.ID, Resource: packet.Resource, Epoch: packet.Epoch, Content: packet.Content})
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

func (cluster *nodeCluster) peerAvailable(id string) bool {
	if peer := cluster.peer(id); peer != nil {
		select {
		case <-peer.done:
		default:
			return true
		}
	}
	return false
}

func (cluster *nodeCluster) requestIncarnation(requester, boot string) bool {
	if requester == cluster.self.ID {
		return boot == cluster.self.Boot
	}
	if view := cluster.view.Load(); view != nil {
		for _, member := range view.members {
			if member.ID == requester {
				return member.Boot == boot
			}
		}
	}
	return false
}
