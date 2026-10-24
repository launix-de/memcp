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

import "io"
import "fmt"
import "time"
import "bytes"
import "context"
import "compress/gzip"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

const replicaPacketBytes = 64 * 1024

type replicaResource struct {
	Database string `json:"database"`
	Table    string `json:"table,omitempty"`
	Shard    string `json:"shard,omitempty"`
}

func (resource replicaResource) key() string {
	if resource.Shard == "" {
		return resource.Database
	}
	return resource.Database + "/" + resource.Shard
}

func (resource replicaResource) valid() bool {
	if !validReplicaDatabase(resource.Database) {
		return false
	}
	if resource.Shard == "" {
		return resource.Table == ""
	}
	_, err := uuid.Parse(resource.Shard)
	return err == nil && resource.Table != ""
}

// Packet kinds select their header fields: read/write carry user proof;
// recall/invalidate carry an epoch; begin/object/chunk/end install one resource;
// grant/failure/ack complete a state transition. A chunk is independently gzip
// compressed and bounded, so queues never retain a whole shard or column.
type replicaPacket struct {
	Kind       string          `json:"kind"`
	ID         string          `json:"id"`
	Resource   replicaResource `json:"resource"`
	Epoch      uint64          `json:"epoch,omitempty"`
	State      string          `json:"state,omitempty"`
	Generation string          `json:"generation,omitempty"`
	Object     string          `json:"object,omitempty"`
	ObjectKind string          `json:"object_kind,omitempty"`
	User       string          `json:"user,omitempty"`
	Hash       string          `json:"hash,omitempty"`
	Payload    []byte          `json:"payload,omitempty"`
	Error      string          `json:"error,omitempty"`
	Owner      string          `json:"owner,omitempty"`
	Sequence   uint64          `json:"sequence,omitempty"`
}

func (peer *replicationPeer) post(packet replicaPacket) error {
	return peer.send(replicationMessage{Packet: &packet})
}

func (peer *replicationPeer) sendObject(header replicaPacket, kind, name string, reader io.ReadCloser) error {
	defer reader.Close()
	header.Kind, header.ObjectKind, header.Object = "object", kind, name
	if err := peer.post(header); err != nil {
		return err
	}
	buffer := make([]byte, replicaPacketBytes)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			var packed bytes.Buffer
			compressor := gzip.NewWriter(&packed)
			if _, writeErr := compressor.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if closeErr := compressor.Close(); closeErr != nil {
				return closeErr
			}
			header.Kind, header.Payload = "chunk", packed.Bytes()
			if sendErr := peer.post(header); sendErr != nil {
				return sendErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	header.Kind, header.Payload = "object_end", nil
	return peer.post(header)
}

func decodeReplicaChunk(packet replicaPacket) ([]byte, error) {
	if len(packet.Payload) > replicaPacketBytes+1024 {
		return nil, fmt.Errorf("oversized replication packet")
	}
	reader, err := gzip.NewReader(bytes.NewReader(packet.Payload))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, replicaPacketBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > replicaPacketBytes {
		return nil, fmt.Errorf("oversized replication payload")
	}
	return data, nil
}

func replicaPacketContext(peer *replicationPeer) (context.Context, context.CancelFunc) {
	return context.WithTimeout(peer.ctx, 30*time.Minute)
}

// receivePacket runs in the connection reader. Payload installation and grant
// publication complete here. Requests which may wait for another node execute
// in workers, so the reader remains available for their replies.
func (r *replicationManager) receivePacket(source string, packet *replicaPacket) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication packet: %v", failure)
		}
	}()
	if !packet.Resource.valid() || packet.ID == "" {
		return fmt.Errorf("invalid resource packet")
	}
	if r.master {
		r.mu.Lock()
		retired := r.journal.Retired[source]
		r.mu.Unlock()
		if retired {
			return fmt.Errorf("replica retired by administrator")
		}
	}
	peer := r.peer(source)
	if peer == nil {
		return fmt.Errorf(replicationNetworkError)
	}
	switch packet.Kind {
	case "read", "write", "recall", "invalidate":
		copy := *packet
		peer.workers.Add(1)
		go func() {
			defer peer.workers.Done()
			ctx, cancel := replicaPacketContext(peer)
			defer cancel()
			var requestErr error
			func() {
				defer func() {
					if failure := recover(); failure != nil {
						requestErr = fmt.Errorf("%v", failure)
					}
				}()
				switch copy.Kind {
				case "read", "write":
					requestErr = r.coordinate(ctx, source, copy)
				case "recall":
					requestErr = r.offerResource(ctx, source, copy)
				case "invalidate":
					requestErr = r.invalidateResource(ctx, source, copy)
				}
			}()
			if requestErr != nil {
				_ = peer.post(replicaPacket{Kind: "failure", ID: copy.ID, Resource: copy.Resource, Error: requestErr.Error()})
			}
		}()
		return nil
	case "failure", "ack":
		if value, exists := r.steps.Load(packet.ID); exists {
			step := value.(*replicaStep)
			if step.peer != source || step.resource != packet.Resource {
				return fmt.Errorf("unexpected acknowledgement")
			}
			var failure error
			if packet.Error != "" {
				failure = fmt.Errorf("%s", packet.Error)
			}
			step.finish(failure)
		} else if packet.Kind == "failure" {
			r.resourceCell(packet.Resource).rights.fail(packet.ID, fmt.Errorf(replicationNetworkError))
		}
		return nil
	case "grant":
		if r.master || source != "" {
			return fmt.Errorf("grant from non-master")
		}
		return r.acceptGrant(*packet)
	case "begin":
		if r.master {
			if value, expected := r.steps.Load(packet.ID); expected {
				step := value.(*replicaStep)
				if step.peer != source || step.resource != packet.Resource {
					return fmt.Errorf("unexpected transfer source")
				}
			} else {
				r.mu.Lock()
				record := r.journal.Records[packet.Resource.key()]
				r.mu.Unlock()
				if record.Owner != source || record.Epoch != packet.Epoch {
					return fmt.Errorf("backup from non-owner")
				}
				if err := r.authorizePacket(*packet); err != nil {
					return err
				}
			}
		} else if source != "" {
			return fmt.Errorf("transfer from non-master")
		}
		key := source + "/" + packet.ID
		if _, exists := r.imports.Load(key); exists {
			return fmt.Errorf("duplicate transfer")
		}
		transfer, err := r.startImport(*packet)
		if err != nil {
			return err
		}
		publication := r.resourceCell(replicaResource{Database: packet.Resource.Database})
		publication.importPublication.Lock()
		r.imports.Store(key, transfer)
		publication.importPublication.Unlock()
		return nil
	case "object", "chunk", "object_end", "end":
		key := source + "/" + packet.ID
		value, exists := r.imports.Load(key)
		if !exists {
			return fmt.Errorf("packet without transfer")
		}
		transfer := value.(*replicaImport)
		if packet.Resource != transfer.header.Resource || packet.Epoch != transfer.header.Epoch || packet.Generation != transfer.header.Generation {
			return fmt.Errorf("transfer header changed")
		}
		if packet.Kind != "end" {
			return transfer.packet(*packet)
		}
		if packet.Resource.Shard != "" {
			publication := r.resourceCell(replicaResource{Database: packet.Resource.Database})
			publication.importPublication.Lock()
			defer publication.importPublication.Unlock()
		}
		defer r.imports.Delete(key)
		defer transfer.abort()
		if r.master {
			r.mu.Lock()
			record := r.journal.Records[packet.Resource.key()]
			r.mu.Unlock()
			if record.Owner != source || record.Epoch != packet.Epoch {
				return fmt.Errorf("stale transfer epoch")
			}
		}
		if err := r.finishImport(source, transfer); err != nil {
			return err
		}
		if r.master && packet.Resource.Shard != "" {
			r.mu.Lock()
			record := r.journal.Records[packet.Resource.key()]
			record.Generation = packet.Generation
			r.journal.Records[packet.Resource.key()] = record
			err := r.saveLocked()
			r.mu.Unlock()
			if err != nil {
				return err
			}
		}
		if !r.master {
			return peer.post(replicaPacket{Kind: "ack", ID: packet.ID, Resource: packet.Resource})
		}
		if value, expected := r.steps.Load(packet.ID); expected {
			value.(*replicaStep).finish(nil)
		} else if packet.Resource.Shard != "" {
			// A rebuild is a point update, not a database-wide metadata sync.
			go r.distributeRebuild(packet.Resource)
		}
		return nil
	default:
		return fmt.Errorf("unknown replication packet kind")
	}
}

func (r *replicationManager) authorizePacket(packet replicaPacket) error {
	auth, err := r.authenticate(packet.User, packet.Hash, nil, nil, packet.Resource.Database)
	if err != nil {
		return err
	}
	if !auth.Admin && !auth.Databases[packet.Resource.Database] {
		return fmt.Errorf("database access denied")
	}
	if packet.Resource.Table != "" {
		policy := scm.Apply(r.policy, scm.NewString(packet.User))
		if !scm.Apply(policy, scm.NewString(packet.Resource.Database), scm.NewString(packet.Resource.Table), scm.NewBool(packet.Kind == "write" || packet.Kind == "begin")).Bool() {
			return fmt.Errorf("table access denied")
		}
	}
	return nil
}

func (r *replicationManager) acceptGrant(packet replicaPacket) error {
	c := r.resourceCell(packet.Resource)
	_, err := c.rights.grant(packet.ID, parseMosiPhase(packet.State), func() error {
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

func (r *replicationManager) invalidateResource(ctx context.Context, source string, packet replicaPacket) error {
	if r.master || source != "" {
		return fmt.Errorf("invalidation from non-master")
	}
	c := r.resourceCell(packet.Resource)
	if err := replicationLock(ctx, &c.gate); err != nil {
		return err
	}
	defer c.gate.Unlock()
	if err := r.localRecord(packet.Resource, mosiIO, ""); err != nil {
		return err
	}
	r.publishLocal(packet.Resource, mosiIO)
	return r.peer(source).post(replicaPacket{Kind: "ack", Resource: packet.Resource, ID: packet.ID})
}

func (r *replicationManager) offerResource(ctx context.Context, source string, packet replicaPacket) error {
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
	phase := parseMosiPhase(packet.State)
	if err := r.localRecord(packet.Resource, phase, generation); err != nil {
		return err
	}
	r.publishLocal(packet.Resource, phase)
	packet.Generation = generation
	releasePersistence()
	return r.sendResource(r.peer(source), packet)
}

func (r *replicationManager) coordinate(ctx context.Context, requester string, packet replicaPacket) error {
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
				err = r.localRecord(packet.Resource, parseMosiPhase(oldState), generation)
			}
			if err == nil {
				r.publishLocal(packet.Resource, parseMosiPhase(oldState))
				record.Generation = generation
			}
			c.gate.Unlock()
			releasePersistence()
			if err != nil {
				return err
			}
		} else {
			if err := r.step(ctx, record.Owner, replicaPacket{Kind: "recall", ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch, State: oldState}); err != nil {
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
				r.publishLocal(packet.Resource, mosiIO)
				c.gate.Unlock()
			} else if err := r.step(ctx, reader, replicaPacket{Kind: "invalidate", ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch}); err != nil {
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
		if err := r.transmitResource(ctx, requester, replicaPacket{ID: uuid.NewString(), Resource: packet.Resource, Epoch: record.Epoch, Generation: record.Generation}); err != nil {
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
		_, err := c.rights.grant(packet.ID, parseMosiPhase(grantState), func() error { return nil })
		return err
	}
	peer := r.peer(requester)
	if peer == nil {
		return fmt.Errorf(replicationNetworkError)
	}
	r.publishLocal(packet.Resource, parseMosiPhase(record.State))
	return peer.post(replicaPacket{Kind: "grant", ID: packet.ID, Resource: packet.Resource, Epoch: record.Epoch, State: grantState, Generation: record.Generation, Owner: record.Owner})
}
