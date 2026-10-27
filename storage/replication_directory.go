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
import "context"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/storage/mosi"

func (r *replicationManager) acceptGrant(packet mosiPacket) error {
	c := r.resourceCell(packet.Resource)
	c.acquire.Lock()
	defer c.acquire.Unlock()
	publication := c.ownstate.Snapshot()
	if r.cluster != nil && packet.State == "S" && publication.Phase() == mosi.MR && publication.Flight().ID() == packet.ID {
		// A read grant superseded by this flight's write upgrade still proves
		// the reader epoch registered by its owner. Keep that control metadata
		// so a subsequent invalidation can revoke the registered reader; never
		// grant S or wake the MR waiters from this weaker response.
		r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
			if packet.Epoch < record.Epoch {
				return false
			}
			record.Resource, record.Owner, record.Epoch = packet.Resource, packet.Owner, packet.Epoch
			return true
		})
		return nil
	}
	_, err := c.ownstate.Grant(packet.ID, mosi.ParsePhase(packet.State), func() error {
		r.mu.Lock()
		current := r.directory.Records[r.resourceKey(packet.Resource)]
		r.mu.Unlock()
		if packet.Epoch < current.Epoch {
			return fmt.Errorf("stale grant epoch")
		}
		r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
			record.Resource, record.State, record.Epoch, record.Owner = packet.Resource, packet.State, packet.Epoch, packet.Owner
			record.Readers = packet.Readers
			record.Handoff = ""
			if packet.State == "M" {
				record.InitialClaimed = true
			}
			if packet.Content != "" {
				record.Content = packet.Content
			}
			return true
		})
		// The content is installed and the directory accepted this grant.
		// Publish outward permission before Grant wakes local query/write waiters.
		// A future delegating slave must grant its own descendants through their
		// directory; it must never treat this outer M as unconditional local M.
		c.clusterstate.Publish(mosi.ParsePhase(packet.State))
		return nil
	})
	return err
}

func (r *replicationManager) invalidateResource(ctx context.Context, source string, packet mosiPacket) error {
	if r.cluster == nil && (r.master || source != "") {
		return fmt.Errorf("invalidation from non-master")
	}
	c := r.resourceCell(packet.Resource)
	// Serialize only inbound authority publication against a newer grant. Local
	// read/write admission never takes acquire, and it is released before send.
	if err := func() error {
		c.acquire.Lock()
		defer c.acquire.Unlock()
		r.mu.Lock()
		epoch := r.directory.Records[r.resourceKey(packet.Resource)].Epoch
		r.mu.Unlock()
		if packet.Epoch > epoch {
			return fmt.Errorf("future invalidation epoch")
		}
		if packet.Epoch < epoch {
			return nil
		}
		handover, err := c.ownstate.BeginHandover(ctx, c.drainWrites)
		if err != nil {
			return err
		}
		defer handover.Release()
		r.localRecord(packet.Resource, mosi.IO, "")
		r.publishLocal(packet.Resource, mosi.IO)
		// No inward holder may retain stronger rights after an outward ACK.
		// Current slaves are leaves; a grouped slave must first recall its children.
		c.clusterstate.Publish(mosi.IO)
		return nil
	}(); err != nil {
		return err
	}
	peer := r.peer(source)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	return peer.post(mosiPacket{Kind: "ack", Resource: packet.Resource, ID: packet.ID, Epoch: packet.Epoch})
}

func (r *replicationManager) offerResource(ctx context.Context, source string, packet mosiPacket) error {
	if r.cluster == nil && (r.master || source != "") {
		return fmt.Errorf("recall from non-master")
	}
	c := r.resourceCell(packet.Resource)
	// Older safety-copy packets must be enqueued before this ownership offer.
	// Waiting for a sender never closes local write admission.
	if r.cluster != nil && packet.Resource.Shard != "" {
		r.obtain(mosiResource{Database: packet.Resource.Database}, mosi.M, "", nil)
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	releasePersistence, err := replicaPersistenceRead(packet.Resource)
	if err != nil {
		return err
	}
	defer releasePersistence()
	c.acquire.Lock()
	authorityLocked := true
	defer func() {
		if authorityLocked {
			c.acquire.Unlock()
		}
	}()
	handover, err := c.ownstate.BeginHandover(ctx, c.drainWrites)
	if err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			handover.Release()
		}
	}()
	r.mu.Lock()
	record := r.directory.Records[r.resourceKey(packet.Resource)]
	r.mu.Unlock()
	if record.Epoch != packet.Epoch {
		return fmt.Errorf("stale recall")
	}
	content, err := r.prepareResource(packet.Resource)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	phase := mosi.ParsePhase(packet.State)
	r.localRecord(packet.Resource, phase, content)
	r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
		record.Readers = packet.Readers
		record.Handoff = ""
		if phase == mosi.IO {
			record.Handoff = packet.Owner
		}
		return true
	})
	r.publishLocal(packet.Resource, phase)
	// Local writes have drained before the enclosing permission is reduced.
	// The root master does not take this path when delegating to its slave.
	c.clusterstate.Publish(phase)
	packet.Content = content
	if r.cluster != nil {
		handover.Release()
		locked = false
		c.acquire.Unlock()
		authorityLocked = false
		releasePersistence()
		packet.Kind, packet.From, packet.To = "reference", r.id, source
		if packet.Resource.Shard != "" {
			packet.Forwards = r.shardRoutes(packet.Resource)
		}
		if current := c.current.Load(); current != nil && packet.Resource.Shard != "" {

		}
		peer := r.peer(source)
		if peer == nil {
			return fmt.Errorf(mosi.NetworkError)
		}
		return peer.post(packet)
	}
	transfer, err := r.captureResource(packet.Resource, content)
	if err != nil {
		return err
	}
	handover.Release()
	locked = false
	c.acquire.Unlock()
	authorityLocked = false
	releasePersistence()
	return r.sendResource(r.peer(source), packet, transfer)
}

func (r *replicationManager) coordinate(ctx context.Context, requester string, packet mosiPacket) error {
	if r.cluster == nil && sharedClusterDatabase(packet.Resource.Database) {
		return fmt.Errorf("replication of a cluster-owned database requires an enclosing domain handover")
	}

	if !r.master {
		return fmt.Errorf("resource request to non-master")
	}
	if r.cluster == nil && requester != r.id {
		if err := r.authorizePacket(packet); err != nil {
			return err
		}
	}
	if r.cluster == nil && requester != r.id && packet.Resource.Shard != "" {
		if err := r.acceptRequestRoutes(requester, packet); err != nil {
			return err
		}
	}
	if !r.resourceKnown(packet.Resource) {
		return fmt.Errorf(mosi.NetworkError)
	}
	c := r.resourceCell(packet.Resource)
	c.directory.Lock()
	defer c.directory.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if r.cluster != nil {
		// Discovery may race an earlier handover while this request waits for
		// directory ownership. The old owner forwards to the successor; it must
		// never become a permanent coordinator recalling the actual owner.
		r.mu.Lock()
		owner := r.directory.Records[r.resourceKey(packet.Resource)].Owner
		r.mu.Unlock()
		if owner != "" && owner != r.id {
			peer := r.peer(owner)
			if peer == nil {
				return fmt.Errorf(mosi.NetworkError)
			}
			packet.From, packet.To = requester, owner
			return peer.post(packet)
		}
	}
	if r.cluster == nil {
		// A star master retains its outward root claim while delegating inward
		// to a slave. Cluster peers instead receive authority through owner
		// discovery and grants; merely coordinating cannot establish outer M.
		c.clusterstate.Establish(mosi.M)
	}
	// Catalog loading may itself declare shards and acquire r.mu.
	localShard := c.current.Load()
	if localShard == nil && packet.Resource.Shard != "" {
		localShard = r.findShard(packet.Resource)
	}
	write := packet.Kind == "write" || packet.Kind == "begin"
	stale := false
	record := r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
		if record.Owner == "" {
			record.Resource, record.Owner, record.State, record.Epoch = packet.Resource, r.id, "M", 1
			if localShard != nil && packet.Resource.Shard != "" {
				record.Content = localShard.uuid.String()
				c.current.Store(localShard)
			}
		}
		mark := record.Requests[requester]
		sameSequence := packet.Sequence == mark.Sequence
		differentFlight := mark.ID != "" && mark.ID != packet.ID
		completed := mark.Completed && mark.Write == write
		if packet.Sequence < mark.Sequence || sameSequence && (differentFlight || mark.Write && !write || completed) {
			stale = true
			return false
		}
		if record.Requests == nil {
			record.Requests = make(map[string]replicaRequestMark)
		}
		record.Requests[requester] = replicaRequestMark{Sequence: packet.Sequence, Write: write, ID: packet.ID}
		return true
	})
	if stale {
		return nil
	}
	originalOwner := record.Owner
	// The successor remains private while network handover runs.
	readers := make(map[string]bool, len(record.Readers))
	for id, present := range record.Readers {
		readers[id] = present
	}
	record.Readers = readers
	if record.Owner != requester {
		oldState := "IO"
		if !write {
			oldState = "O"
		}
		if record.Owner == r.id {
			if r.cluster != nil && packet.Resource.Shard != "" {
				r.obtain(mosiResource{Database: packet.Resource.Database}, mosi.M, "", nil)
			}
			releasePersistence, err := replicaPersistenceRead(packet.Resource)
			if err != nil {
				return err
			}
			defer releasePersistence()
			handover, err := c.ownstate.BeginHandover(ctx, c.drainWrites)
			if err != nil {
				return err
			}
			content, err := r.prepareResource(packet.Resource)
			if err == nil && requester != r.id && r.cluster == nil {
				err = r.saveBackup(packet.Resource, content)
			}
			if err == nil {
				r.localRecord(packet.Resource, mosi.ParsePhase(oldState), content)
			}
			if err == nil {
				if r.cluster != nil && write {
					// Route concurrent lookups toward the successor while its
					// installation ACK is pending. This is not a grant; local IO
					// remains revoked and the successor still waits for its M grant.
					r.updateRecord(packet.Resource, func(current *replicaRecord) bool {
						current.Handoff = requester
						return true
					})
				}
				r.publishLocal(packet.Resource, mosi.ParsePhase(oldState))
				record.Content = content
			}
			handover.Release()
			releasePersistence()
			if err != nil {
				return err
			}
		} else {
			knownReaders := make(map[string]bool, len(record.Readers)+1)
			for reader, present := range record.Readers {
				knownReaders[reader] = present
			}
			if !write {
				knownReaders[requester] = true
			}
			if err := r.step(ctx, record.Owner, mosiPacket{Kind: "recall", ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch, State: oldState, Readers: knownReaders, Owner: requester}); err != nil {
				return err
			}
			r.mu.Lock()
			record.Content = r.directory.Records[r.resourceKey(packet.Resource)].Content
			r.mu.Unlock()
		}
	}
	if write {
		for reader := range record.Readers {
			if reader == requester {
				continue
			}
			if reader == r.id {
				handover, err := c.ownstate.BeginHandover(ctx, c.drainWrites)
				if err != nil {
					return err
				}
				r.publishLocal(packet.Resource, mosi.IO)
				handover.Release()
			} else if err := r.step(ctx, reader, mosiPacket{Kind: "invalidate", ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch}); err != nil {
				return err
			}
		}
		record.Readers = make(map[string]bool)
		record.Owner, record.Epoch = requester, record.Epoch+1
	} else if requester != record.Owner {
		record.Readers[requester] = true
	}
	grantState := "S"
	if write {
		grantState = "M"
	} else if requester == record.Owner {
		grantState = "O"
	}
	// The destination durably installs the complete content before the
	// directory commits its new owner. A crash during transfer leaves ownership
	// at the old modifier, never at an incomplete destination copy.
	if requester != r.id && requester != originalOwner {
		if err := r.transmitResource(ctx, requester, mosiPacket{ID: packet.ID, Resource: packet.Resource, Epoch: record.Epoch, Content: record.Content}); err != nil {
			return err
		}
	}
	if r.cluster != nil && record.Owner != r.id {
		record.Readers = nil
	}
	record.State = "IO"
	if record.Owner == r.id {
		record.State = "O"
		if len(record.Readers) == 0 {
			record.State = "M"
		}
	} else if record.Readers[r.id] {
		record.State = "S"
	}
	r.updateRecord(packet.Resource, func(current *replicaRecord) bool {
		// updateRecord cloned the latest published request directory. Complete
		// only that private successor; the earlier pending record may be retained
		// by diagnostics or recovery while this handover was awaiting an ACK.
		mark := current.Requests[requester]
		mark.Completed = true
		current.Requests[requester] = mark
		record.Requests = current.Requests
		*current = record
		if r.master {
			if r.confirmed == nil {
				r.confirmed = make(map[string]string)
			}
			r.confirmed[r.resourceKey(packet.Resource)] = record.Owner
			if write && requester != r.id {
				if r.initialClaims == nil {
					r.initialClaims = make(map[string]map[string]bool)
				}
				if r.initialClaims[requester] == nil {
					r.initialClaims[requester] = make(map[string]bool)
				}
				r.initialClaims[requester][packet.Resource.key()] = true
			}
		}
		return true
	})
	if record.State == "M" {
		if err := r.clearBackup(packet.Resource); err != nil {
			return err
		}
	}
	if requester == r.id {
		_, err := c.ownstate.Grant(packet.ID, mosi.ParsePhase(grantState), func() error { return nil })
		return err
	}
	peer := r.peer(requester)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	r.publishLocal(packet.Resource, mosi.ParsePhase(record.State))
	return peer.post(mosiPacket{Kind: "grant", ID: packet.ID, Resource: packet.Resource, Epoch: record.Epoch, State: grantState, Content: record.Content, Owner: record.Owner, Readers: func() map[string]bool {
		if r.cluster != nil {
			return nil
		}
		return record.Readers
	}()})
}

// An owner can rebuild while retaining M. Its next request can name the native
// successor before the coordinator has recalled its data. Learn that edge from
// the authenticated current owner, never invent local authority for an unknown
// UUID. Reader hints alone cannot redirect another participant's resource.
func (r *replicationManager) acceptRequestRoutes(source string, packet mosiPacket) error {
	if len(packet.Forwards) == 0 {
		return nil
	}
	for _, route := range packet.Forwards {
		if !route.From.valid() || route.From.Shard == "" || route.From.Database != packet.Resource.Database || route.From.Table != packet.Resource.Table || route.To != packet.Resource {
			return fmt.Errorf("invalid request forwarding")
		}
	}
	r.mu.Lock()
	target := r.canonicalResource(packet.Resource)
	known := r.directory.Records[target.key()].Owner != ""
	var authority *replicaCell
	if !known {
		for _, route := range packet.Forwards {
			from := r.canonicalResource(route.From)
			if record := r.directory.Records[from.key()]; record.Owner == source {
				authority = r.lookupCell(from)
				break
			}
		}
	}
	r.mu.Unlock()
	if known {
		return nil // an old request cannot reverse an already learned successor
	}
	if authority == nil {
		return fmt.Errorf("request forwarding from non-owner")
	}
	for _, route := range packet.Forwards {
		if cell := r.lookupCell(route.From); cell != nil && cell != authority {
			return fmt.Errorf("request forwarding crosses shard authorities")
		}
	}
	for _, route := range packet.Forwards {
		r.forwardResource(route.From, route.To)
	}
	return nil
}
