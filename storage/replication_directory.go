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
	_, err := c.rights.Grant(packet.ID, mosi.ParsePhase(packet.State), func() error {
		r.mu.Lock()
		defer r.mu.Unlock()
		record := r.journal.Records[packet.Resource.key()]
		record.Resource, record.State, record.Epoch, record.Owner = packet.Resource, packet.State, packet.Epoch, packet.Owner
		if packet.State == "M" {
			record.InitialClaimed = true
		}
		if packet.Generation != "" {
			record.Generation = packet.Generation
		}
		r.journal.Records[packet.Resource.key()] = record
		return r.saveLocked()
	})
	return err
}

func (r *replicationManager) invalidateResource(ctx context.Context, source string, packet mosiPacket) error {
	if r.master || source != "" {
		return fmt.Errorf("invalidation from non-master")
	}
	c := r.resourceCell(packet.Resource)
	if err := replicationLock(ctx, &c.gate); err != nil {
		return err
	}
	defer c.gate.Unlock()
	if err := r.localRecord(packet.Resource, mosi.IO, ""); err != nil {
		return err
	}
	r.publishLocal(packet.Resource, mosi.IO)
	return r.peer(source).post(mosiPacket{Kind: "ack", Resource: packet.Resource, ID: packet.ID})
}

func (r *replicationManager) offerResource(ctx context.Context, source string, packet mosiPacket) error {
	if r.master || source != "" {
		return fmt.Errorf("recall from non-master")
	}
	releasePersistence, err := replicaPersistenceRead(packet.Resource)
	if err != nil {
		return err
	}
	defer releasePersistence()
	c := r.resourceCell(packet.Resource)
	if err := replicationLock(ctx, &c.gate); err != nil {
		return err
	}
	defer c.gate.Unlock()
	r.mu.Lock()
	record := r.journal.Records[packet.Resource.key()]
	r.mu.Unlock()
	if record.Epoch != packet.Epoch {
		return fmt.Errorf("stale recall")
	}
	generation, err := r.prepareResource(packet.Resource)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	phase := mosi.ParsePhase(packet.State)
	if err := r.localRecord(packet.Resource, phase, generation); err != nil {
		return err
	}
	r.publishLocal(packet.Resource, phase)
	packet.Generation = generation
	releasePersistence()
	return r.sendResource(r.peer(source), packet)
}

func (r *replicationManager) coordinate(ctx context.Context, requester string, packet mosiPacket) error {
	if !r.master {
		return fmt.Errorf("resource request to non-master")
	}
	if requester != r.id {
		if err := r.authorizePacket(packet); err != nil {
			return err
		}
	}
	c := r.resourceCell(packet.Resource)
	c.directory.Lock()
	defer c.directory.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// Catalog loading may itself declare shards and acquire r.mu.
	localShard := c.current.Load()
	if localShard == nil && packet.Resource.Shard != "" {
		localShard = r.findShard(packet.Resource)
	}
	r.mu.Lock()
	record, exists := r.journal.Records[packet.Resource.key()]
	if !exists {
		record = replicaRecord{Resource: packet.Resource, Owner: r.id, State: "M", Epoch: 1}
		if packet.Resource.Shard != "" {
			if shard := localShard; shard != nil {
				record.Generation = shard.uuid.String()
				c.current.Store(shard)
			}
		}
	}
	marks := make(map[string]replicaRequestMark, len(record.Requests))
	for id, mark := range record.Requests {
		marks[id] = mark
	}
	record.Requests = marks
	mark := marks[requester]
	write := packet.Kind == "write" || packet.Kind == "begin"
	if packet.Sequence < mark.Sequence || (packet.Sequence == mark.Sequence && mark.Write && !write) {
		r.mu.Unlock()
		return nil
	}
	marks[requester] = replicaRequestMark{Sequence: packet.Sequence, Write: write}
	originalOwner := record.Owner
	r.journal.Records[packet.Resource.key()] = record
	if err := r.saveLocked(); err != nil {
		r.mu.Unlock()
		return err
	}
	// The durable request mark now shares the journal's reader directory.
	// Handovers keep a private successor until their final publication: another
	// resource's checkpoint may serialize the journal while network steps run.
	readers := make(map[string]bool, len(record.Readers))
	for id, present := range record.Readers {
		readers[id] = present
	}
	record.Readers = readers
	r.mu.Unlock()
	if record.Owner != requester {
		oldState := "IO"
		if !write {
			oldState = "O"
		}
		if record.Owner == r.id {
			releasePersistence, err := replicaPersistenceRead(packet.Resource)
			if err != nil {
				return err
			}
			defer releasePersistence()
			if err := replicationLock(ctx, &c.gate); err != nil {
				return err
			}
			generation, err := r.prepareResource(packet.Resource)
			if err == nil {
				err = r.localRecord(packet.Resource, mosi.ParsePhase(oldState), generation)
			}
			if err == nil {
				r.publishLocal(packet.Resource, mosi.ParsePhase(oldState))
				record.Generation = generation
			}
			c.gate.Unlock()
			releasePersistence()
			if err != nil {
				return err
			}
		} else {
			if err := r.step(ctx, record.Owner, mosiPacket{Kind: "recall", ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch, State: oldState}); err != nil {
				return err
			}
			if shard := c.current.Load(); shard != nil {
				record.Generation = shard.uuid.String()
			}
		}
	}
	if write {
		for reader := range record.Readers {
			if reader == requester {
				continue
			}
			if reader == r.id {
				if err := replicationLock(ctx, &c.gate); err != nil {
					return err
				}
				r.publishLocal(packet.Resource, mosi.IO)
				c.gate.Unlock()
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
	// The destination durably installs the complete generation before the
	// directory commits its new owner. A crash during transfer leaves ownership
	// at the old modifier, never at an incomplete destination copy.
	if requester != r.id && requester != originalOwner {
		if err := r.transmitResource(ctx, requester, mosiPacket{ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch, Generation: record.Generation}); err != nil {
			return err
		}
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
	r.mu.Lock()
	r.journal.Records[packet.Resource.key()] = record
	err := r.saveLocked()
	r.mu.Unlock()
	if err != nil {
		return err
	}
	if requester == r.id {
		_, err := c.rights.Grant(packet.ID, mosi.ParsePhase(grantState), func() error { return nil })
		return err
	}
	peer := r.peer(requester)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	r.publishLocal(packet.Resource, mosi.ParsePhase(record.State))
	return peer.post(mosiPacket{Kind: "grant", ID: packet.ID, Resource: packet.Resource, Epoch: record.Epoch, State: grantState, Generation: record.Generation, Owner: record.Owner})
}
