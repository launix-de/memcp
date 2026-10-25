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
		r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
			record.Resource, record.State, record.Epoch, record.Owner = packet.Resource, packet.State, packet.Epoch, packet.Owner
			record.Readers = packet.Readers
			record.Handoff = ""
			if packet.State == "M" {
				record.InitialClaimed = true
			}
			if packet.Generation != "" {
				record.Generation = packet.Generation
			}
			return true
		})
		return nil
	})
	return err
}

func (r *replicationManager) invalidateResource(ctx context.Context, source string, packet mosiPacket) error {
	if r.master || source != "" {
		return fmt.Errorf("invalidation from non-master")
	}
	c := r.resourceCell(packet.Resource)
	handover, err := c.rights.BeginHandover(ctx, c.drainWrites)
	if err != nil {
		return err
	}
	defer handover.Release()
	r.localRecord(packet.Resource, mosi.IO, "")
	r.publishLocal(packet.Resource, mosi.IO)
	return r.peer(source).post(mosiPacket{Kind: "ack", Resource: packet.Resource, ID: packet.ID})
}

func (r *replicationManager) offerResource(ctx context.Context, source string, packet mosiPacket) error {
	if r.master || source != "" {
		return fmt.Errorf("recall from non-master")
	}
	c := r.resourceCell(packet.Resource)
	// Older safety-copy packets must be enqueued before this ownership offer.
	// Waiting for a sender never closes local write admission.
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	releasePersistence, err := replicaPersistenceRead(packet.Resource)
	if err != nil {
		return err
	}
	defer releasePersistence()
	handover, err := c.rights.BeginHandover(ctx, c.drainWrites)
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
	record := r.directory.Records[packet.Resource.key()]
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
	r.localRecord(packet.Resource, phase, generation)
	r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
		record.Readers = packet.Readers
		record.Handoff = ""
		if phase == mosi.IO {
			record.Handoff = packet.Owner
		}
		return true
	})
	r.publishLocal(packet.Resource, phase)
	packet.Generation = generation
	transfer, err := r.captureResource(packet.Resource, generation)
	if err != nil {
		return err
	}
	handover.Release()
	locked = false
	releasePersistence()
	return r.sendResource(r.peer(source), packet, transfer)
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
	if !r.resourceKnown(packet.Resource) {
		return fmt.Errorf(mosi.NetworkError)
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
	write := packet.Kind == "write" || packet.Kind == "begin"
	stale := false
	record := r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
		if record.Owner == "" {
			record.Resource, record.Owner, record.State, record.Epoch = packet.Resource, r.id, "M", 1
			if localShard != nil && packet.Resource.Shard != "" {
				record.Generation = localShard.uuid.String()
				c.current.Store(localShard)
			}
		}
		mark := record.Requests[requester]
		if packet.Sequence < mark.Sequence || (packet.Sequence == mark.Sequence && mark.Write && !write) {
			stale = true
			return false
		}
		if record.Requests == nil {
			record.Requests = make(map[string]replicaRequestMark)
		}
		record.Requests[requester] = replicaRequestMark{Sequence: packet.Sequence, Write: write}
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
			releasePersistence, err := replicaPersistenceRead(packet.Resource)
			if err != nil {
				return err
			}
			defer releasePersistence()
			handover, err := c.rights.BeginHandover(ctx, c.drainWrites)
			if err != nil {
				return err
			}
			generation, err := r.prepareResource(packet.Resource)
			if err == nil && requester != r.id {
				err = r.saveBackup(packet.Resource, generation)
			}
			if err == nil {
				r.localRecord(packet.Resource, mosi.ParsePhase(oldState), generation)
			}
			if err == nil {
				r.publishLocal(packet.Resource, mosi.ParsePhase(oldState))
				record.Generation = generation
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
			record.Generation = r.directory.Records[packet.Resource.key()].Generation
			r.mu.Unlock()
		}
	}
	if write {
		for reader := range record.Readers {
			if reader == requester {
				continue
			}
			if reader == r.id {
				handover, err := c.rights.BeginHandover(ctx, c.drainWrites)
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
	r.updateRecord(packet.Resource, func(current *replicaRecord) bool {
		*current = record
		if r.master {
			if r.confirmed == nil {
				r.confirmed = make(map[string]string)
			}
			r.confirmed[packet.Resource.key()] = record.Owner
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
		_, err := c.rights.Grant(packet.ID, mosi.ParsePhase(grantState), func() error { return nil })
		return err
	}
	peer := r.peer(requester)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	r.publishLocal(packet.Resource, mosi.ParsePhase(record.State))
	return peer.post(mosiPacket{Kind: "grant", ID: packet.ID, Resource: packet.Resource, Epoch: record.Epoch, State: grantState, Generation: record.Generation, Owner: record.Owner, Readers: record.Readers})
}
