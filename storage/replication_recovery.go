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
import "sync"
import "context"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

type replicaRecovery struct {
	Boot      string          `json:"boot"`
	Resources []replicaRecord `json:"resources"`
}

// A retained backup proves data placement, never local authority. A restarted
// master must hear from a live authenticated participant before using it.
func (r *replicationManager) resourceKnown(resource mosiResource) bool {
	if !r.master {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, exists := r.directory.Records[r.resourceKey(resource)]
	if exists && record.Owner == "" {
		return false
	}
	if _, backup := r.backups[r.resourceKey(resource)]; !backup {
		return true
	}
	return record.Owner != "" && r.confirmed[r.resourceKey(resource)] == record.Owner
}

func (r *replicationManager) recoveryInventory(auth replicaAuth) []replicaRecord {
	allowed := func(name string) bool { return !sharedClusterDatabase(name) && (auth.Admin || auth.Databases[name]) }
	resources := make(map[string]replicaRecord)
	// Catalogs describe existing local data only. Loading them establishes no
	// coherence permission, even when the backend contains a complete content.
	for _, db := range databases.GetAll() {
		if !validReplicaDatabase(db.Name) || !allowed(db.Name) {
			continue
		}
		db.ensureLoaded()
		schema := mosiResource{Database: db.Name}
		content, err := r.schemaDigest(db.persistence.ReadSchema())
		if err != nil {
			panic(err)
		}
		resources[schema.key()] = replicaRecord{Resource: schema, Content: content}
		for _, table := range db.tables.GetAll() {
			if table.isEphemeralQueryTable() {
				continue
			}
			r.restoreTableRoutes(table)
			for _, shard := range table.ActiveShards() {
				if shard == nil {
					continue
				}
				name := ""
				if published := table.publishedName.Load(); published != nil {
					name = *published
				} else {
					name = table.Name
				}
				resource := r.canonicalResource(mosiResource{Database: db.Name, Table: name, Shard: shard.uuid.String()})
				resources[resource.key()] = replicaRecord{Resource: resource, Content: shard.uuid.String()}
			}
		}
	}
	r.mu.Lock()
	for key, record := range r.directory.Records {
		if !allowed(record.Resource.Database) {
			continue
		}
		inventory, active := resources[key]
		if !active {
			continue
		}
		record.Resource = inventory.Resource
		content := inventory.Content
		cell := r.cells[key]
		if cell == nil {
			continue
		}
		// This inventory is sent OUTWARD, so delegated authority must not disappear
		// just because the local executor is IO. Today only leaf slaves call this
		// adapter: their record owner/readers/epoch already belong to the upstream
		// domain. A grouped slave must supply that domain's directory record here,
		// never expose its private child-owner directory to the enclosing master.
		publication := cell.clusterstate.Snapshot()
		phase := publication.Phase()
		if phase == mosi.IS {
			phase = publication.Backend()
		}
		if phase.Requested() {
			phase = publication.Retained()
			if !phase.Readable() {
				phase = mosi.IO
			}
		}
		record.State = phase.String()
		record.Readers = maps.Clone(record.Readers)
		record.Requests = nil
		if record.Resource.Shard != "" {
			record.Forwards = r.shardRoutes(record.Resource)
		}
		if current := cell.current.Load(); current != nil {
			content = current.uuid.String()
		}
		if content != "" {
			record.Content = content
		}
		resources[key] = record
	}
	r.mu.Unlock()
	result := make([]replicaRecord, 0, len(resources))
	for _, resource := range resources {
		result = append(result, resource)
	}
	return result
}

func (r *replicationManager) recoverCached(peer *mosiPeer) {
	r.mu.Lock()
	users := maps.Clone(r.directory.Users)
	r.mu.Unlock()
	for user, auth := range users {
		if err := r.recoverConnection(peer, user, auth); err != nil {
			scm.PrintError(err)
		}
	}
}

func (r *replicationManager) recoverConnection(peer *mosiPeer, user string, auth replicaAuth) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication recovery: %v", failure)
		}
	}()
	r.recoveryMu.Lock()
	defer r.recoveryMu.Unlock()
	if r.recoveredUsers[user] == peer {
		return nil
	}
	inventory := r.recoveryInventory(auth)
	ctx, cancel := mosiPacketContext(peer)
	defer cancel()
	// Inventory crosses the existing bounded transport one resource at a time.
	// A fixed worker count avoids both giant JSON frames and a goroutine per shard.
	requests := inventory
	if len(requests) == 0 {
		requests = []replicaRecord{{}}
	}
	replies := make([]replicaRecovery, len(requests))
	var next atomic.Uint64
	var pending sync.WaitGroup
	var failureOnce sync.Once
	var failure error
	for worker := 0; worker < min(8, len(requests)); worker++ {
		pending.Add(1)
		go func() {
			defer pending.Done()
			for {
				index := int(next.Add(1) - 1)
				if index >= len(requests) || ctx.Err() != nil {
					return
				}
				req := replicaRequest{Boot: r.boot, User: user, Hash: auth.Hash}
				if requests[index].Resource.Database != "" {
					req.Resources = []replicaRecord{requests[index]}
				}
				if err := peer.call(ctx, "hello", req, &replies[index]); err != nil {
					failureOnce.Do(func() { failure = err; cancel() })
					return
				}
			}
		}()
	}
	pending.Wait()
	if failure != nil {
		return failure
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	reply := replicaRecovery{}
	for _, part := range replies {
		if _, err := uuid.Parse(part.Boot); err != nil {
			return fmt.Errorf("invalid master incarnation")
		}
		reply.Boot = part.Boot
		reply.Resources = append(reply.Resources, part.Resources...)
	}
	for _, record := range reply.Resources {
		if !record.Resource.valid() {
			return fmt.Errorf("invalid recovery resource")
		}
		for _, route := range record.Forwards {
			r.forwardResource(route.From, route.To)
		}
		cell := r.resourceCell(record.Resource)
		r.mu.Lock()
		if _, exists := r.directory.Records[r.resourceKey(record.Resource)]; !exists {
			r.directory.Records[r.resourceKey(record.Resource)] = record
		}
		r.mu.Unlock()
		// Recover outer permission before local admission. Establish is one-shot:
		// a late hello cannot undo a newer wire grant, recall or pending request.
		r.establishAuthority(cell, record)

	}
	complete := make(map[string]bool)
	for _, record := range reply.Resources {
		if record.Resource.Shard == "" {
			complete[record.Resource.Database] = record.InitialClaimed
		}
	}
	for _, record := range reply.Resources {
		if !record.InitialClaimed {
			complete[record.Resource.Database] = false
		}
	}
	for name, initialized := range complete {
		r.mu.Lock()
		initialized = initialized || r.directory.Initialized[name]
		r.directory.Initialized[name] = initialized
		r.mu.Unlock()
		if initialized {
			r.resourceCell(mosiResource{Database: name}).initialized.Store(true)
		}
	}
	if r.recoveredUsers == nil {
		r.recoveredUsers = make(map[string]*mosiPeer)
	}
	r.recoveredUsers[user] = peer
	r.mu.Lock()
	completeConnection := true
	for cachedUser := range r.directory.Users {
		if r.recoveredUsers[cachedUser] != peer {
			completeConnection = false
			break
		}
	}
	r.mu.Unlock()
	if completeConnection {
		if err := peer.call(ctx, "hello_done", replicaRequest{Boot: r.boot, User: user, Hash: auth.Hash}, nil); err != nil {
			delete(r.recoveredUsers, user)
			return err
		}
	}
	r.ready.Store(true)
	go func() {
		names := make(map[string]bool)
		for _, record := range reply.Resources {
			if record.State == "M" {
				names[record.Resource.Database] = true
			}
		}
		for name := range names {
			r.backupDatabase(name, user)
		}
	}()
	return nil
}

func (r *replicationManager) recoverPeer(ctx context.Context, id string, req replicaRequest, auth replicaAuth) (replicaRecovery, error) {
	reply := replicaRecovery{Boot: r.boot}
	if _, err := uuid.Parse(req.Boot); err != nil {
		return reply, fmt.Errorf("invalid slave incarnation")
	}
	for _, reported := range req.Resources {
		if !reported.Resource.valid() || sharedClusterDatabase(reported.Resource.Database) || (!auth.Admin && !auth.Databases[reported.Resource.Database]) {
			return reply, fmt.Errorf("unauthorized recovery resource")
		}
		switch reported.State {
		case "", "M", "O", "S", "IO":
		default:
			return reply, fmt.Errorf("invalid recovery phase")
		}
	}
	r.mu.Lock()
	oldBoot := r.peersBoot[id]
	r.peersBoot[id] = req.Boot
	r.mu.Unlock()
	if oldBoot != "" && oldBoot != req.Boot {
		r.mu.Lock()
		resources := make([]mosiResource, 0, len(r.directory.Records))
		for _, record := range r.directory.Records {
			resources = append(resources, record.Resource)
		}
		r.mu.Unlock()
		for _, resource := range resources {
			cell := r.resourceCell(resource)
			cell.directory.Lock()
			r.updateRecord(resource, func(record *replicaRecord) bool { delete(record.Requests, id); return true })
			cell.directory.Unlock()
		}
	}
	for _, reported := range req.Resources {
		recovered, exists, err := r.recoverResource(ctx, id, reported)
		if err != nil {
			return reply, err
		}
		if exists {
			reply.Resources = append(reply.Resources, recovered)
		}
	}

	return reply, nil
}

func (r *replicationManager) recoverResource(ctx context.Context, id string, reported replicaRecord) (replicaRecord, bool, error) {
	r.restoreResourceRoutes(reported.Resource)
	for _, route := range reported.Forwards {
		if route.From.Database != reported.Resource.Database || route.To != reported.Resource {
			return replicaRecord{}, false, fmt.Errorf("invalid recovery forwarding")
		}
		r.forwardResource(route.From, route.To)
	}
	resource := r.canonicalResource(reported.Resource)
	reported.Resource = resource
	cell := r.resourceCell(resource)
	cell.directory.Lock()
	defer cell.directory.Unlock()
	cell.acquire.Lock()
	defer cell.acquire.Unlock()
	if ctx.Err() != nil {
		return replicaRecord{}, false, ctx.Err()
	}
	localGeneration := ""
	if reported.State == "IO" && reported.Handoff == r.id {
		r.mu.Lock()
		_, backup := r.backups[r.resourceKey(resource)]
		r.mu.Unlock()
		if !backup {
			var err error
			localGeneration, err = r.completeLocalShard(resource)
			if err != nil {
				return replicaRecord{}, false, err
			}
		}
	}
	record, exists, claimed, err := r.mergeRecoveryReport(id, reported, localGeneration)
	if err != nil || !exists {
		return replicaRecord{}, false, err
	}
	local := mosi.ParsePhase(record.State)
	// A live, authenticated report identifies the root domain's custodian.
	// Previously unknown safety copies stay IO until this proof exists; recovery
	// never infers aggregate M merely from a file or a liveness timeout.
	if record.Owner != "" && r.resourceKnown(resource) {
		cell.clusterstate.Publish(mosi.M)
	} else {
		// A reader's report may name a modifier that has not confirmed recovery.
		// Preserve that routing evidence in the directory, but neither publish a
		// local S from its retained backup nor outrun the still-unknown outer slot.
		// A later confirmed owner is acquired through the ordinary GetRead path.
		local = mosi.IO
	}
	if !cell.ownstate.Initialized() {
		// A remote schema can advertise a UUID whose data never finished arriving.
		// Bind a readable recovered slot to its completed local copy before publishing
		// authority; ordinary COLD loading then uses the correct native content.
		if local.Readable() && resource.Shard != "" && record.Content != "" {
			if db := databases.Get(resource.Database); db != nil {
				db.ensureLoaded()
			}
			if err := r.restoreBackup(record); err != nil {
				return replicaRecord{}, false, err
			}
		}
		cell.ownstate.Establish(local)
	}
	state := "IO"
	if record.Owner == id {
		state = "M"
		if len(record.Readers) > 0 {
			state = "O"
		}
	} else if record.Readers[id] {
		state = "S"
	}
	record.State = state
	record.InitialClaimed = claimed
	record.Content = reported.Content
	record.Requests = nil
	if resource.Shard != "" {
		record.Forwards = r.shardRoutes(resource)
	}
	return record, true, nil
}

// Only immutable runtime metadata is merged while holding the manager mutex.
// Catalog loading, installation, backend writes and network waits happen outside.
func (r *replicationManager) mergeRecoveryReport(id string, reported replicaRecord, localGeneration string) (replicaRecord, bool, bool, error) {
	resource := reported.Resource
	r.mu.Lock()
	defer r.mu.Unlock()
	record, exists := r.directory.Records[r.resourceKey(resource)]
	if r.recoveryReports == nil {
		r.recoveryReports = make(map[string]map[string]replicaRecord)
	}
	if r.recoveryReports[resource.key()] == nil {
		r.recoveryReports[resource.key()] = make(map[string]replicaRecord)
	}
	r.recoveryReports[resource.key()][id] = reported
	if r.initialClaims == nil {
		r.initialClaims = make(map[string]map[string]bool)
	}
	if r.initialClaims[id] == nil {
		r.initialClaims[id] = make(map[string]bool)
	}
	if reported.InitialClaimed {
		r.initialClaims[id][resource.key()] = true
	}
	claimed := r.initialClaims[id][resource.key()]
	restoring := r.restoring[resource.Database]
	if restoring && reported.State != "" && reported.Epoch >= record.Epoch {
		phase := mosi.ParsePhase(reported.State)
		owner := reported.Owner
		owns := phase == mosi.M || phase == mosi.O
		if owns {
			if owner != "" && owner != id {
				return replicaRecord{}, false, false, fmt.Errorf("conflicting recovery owner")
			}
			owner = id
		}
		if record.Epoch == reported.Epoch && r.confirmed[r.resourceKey(resource)] != "" && owner != "" && record.Owner != owner {
			return replicaRecord{}, false, false, fmt.Errorf("conflicting recovery authority")
		}
		if owner != "" {
			if !exists || reported.Epoch > record.Epoch {
				record = replicaRecord{Resource: resource, Owner: owner, Epoch: reported.Epoch, State: "IO", Readers: make(map[string]bool)}
			}
			record.Owner = owner
			record.Readers = maps.Clone(record.Readers)
			if record.Readers == nil {
				record.Readers = make(map[string]bool)
			}
			for reader, present := range reported.Readers {
				record.Readers[reader] = present
			}
			if phase == mosi.S {
				record.Readers[id] = true
			}
			if owns || owner == r.id {
				r.confirmed[r.resourceKey(resource)] = owner
			}
			record.InitialClaimed = reported.InitialClaimed
			if backup, ok := r.backups[r.resourceKey(resource)]; ok {
				record.Content = backup.Content
			}
			if owner == r.id {
				record.State = "M"
				if len(record.Readers) > 0 {
					record.State = "O"
				}
			} else if record.Readers[r.id] {
				record.State = "S"
			}
			exists = true
		}
	}
	if restoring {
		for source, proof := range r.recoveryReports[resource.key()] {
			if proof.State != "IO" || proof.Handoff == "" || proof.Epoch < record.Epoch {
				continue
			}
			target := r.recoveryReports[resource.key()][proof.Handoff]
			if proof.Handoff != r.id && (target.State != "IO" || target.Owner != source || target.Epoch > proof.Epoch) {
				continue
			}
			owner := source
			if proof.Handoff == r.id && (r.backups[r.resourceKey(resource)].Content == proof.Content || localGeneration != "") {
				owner = r.id
			}
			record = replicaRecord{Resource: resource, Owner: owner, Epoch: proof.Epoch, State: "IO", Readers: maps.Clone(proof.Readers), Content: r.backups[r.resourceKey(resource)].Content, InitialClaimed: proof.InitialClaimed}
			if owner == r.id && localGeneration != "" {
				record.Content = localGeneration
			}
			delete(record.Readers, owner)
			delete(record.Readers, proof.Handoff)
			if owner == r.id {
				record.State = "M"
				if len(record.Readers) > 0 {
					record.State = "O"
				}
			}
			r.confirmed[r.resourceKey(resource)] = owner
			proof.Handoff = ""
			r.recoveryReports[resource.key()][source] = proof
			exists = true
		}
	}
	if exists {
		r.directory.Records[r.resourceKey(resource)] = record
	}
	return record, exists, claimed, nil
}
