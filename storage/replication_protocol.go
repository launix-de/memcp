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
import "time"
import "context"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func mosiPacketContext(peer *mosiPeer) (context.Context, context.CancelFunc) {
	return context.WithTimeout(peer.ctx, 30*time.Minute)
}

// applyResourcePacket runs in the resource's ordered packet worker, never the
// network reader. Installation completes before a following grant is admitted.
// Operations awaiting remote replies start their own control workers: their
// acknowledgements must remain dispatchable on this same resource queue.
func (r *replicationManager) applyResourcePacket(source string, packet *mosiPacket) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication packet: %v", failure)
		}
	}()
	if !packet.Resource.valid() || packet.ID == "" {
		return fmt.Errorf("invalid resource packet")
	}
	if packet.Resource == globalResource && r.cluster == nil {
		return fmt.Errorf("global resource requires an authenticated cluster peer")
	}
	if r.master {
		r.mu.Lock()
		retired := r.directory.Retired[source]
		r.mu.Unlock()
		if retired {
			return fmt.Errorf("replica retired by administrator")
		}
	}
	peer := r.peer(source)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	switch packet.Kind {
	case "read", "write", "recall", "invalidate":
		copy := *packet
		peer.workers.Add(1)
		go func() {
			defer peer.workers.Done()
			ctx, cancel := mosiPacketContext(peer)
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
				fmt.Println("replication request failed:", copy.Kind, copy.Resource.key(), requestErr)
				state := copy.State
				if copy.Kind == "read" {
					state = "S"
				} else if copy.Kind == "write" {
					state = "M"
				}
				_ = peer.post(mosiPacket{Kind: "failure", ID: copy.ID, Resource: copy.Resource, Epoch: copy.Epoch, Content: copy.Content, State: state, Error: requestErr.Error()})
			}
		}()
		return nil
	case "failure", "ack":
		if value, exists := r.steps.Load(packet.ID); exists {
			step := value.(*replicaStep)
			if !step.matches(source, *packet) {
				return fmt.Errorf("unexpected acknowledgement")
			}
			var failure error
			if packet.Error != "" {
				failure = fmt.Errorf("%s", packet.Error)
			}
			step.finish(failure)
		} else if packet.Kind == "failure" {
			rights := &r.resourceCell(packet.Resource).ownstate
			// Read and write-upgrade workers share a flight. A late read failure
			// cannot cancel the stronger request already sent for that flight.
			requested := mosi.IO
			if packet.State == "S" {
				requested = mosi.S
			}
			rights.Fail(packet.ID, fmt.Errorf(mosi.NetworkError), requested)
		}
		return nil
	case "grant":
		if r.cluster == nil && (r.master || source != "") {
			return fmt.Errorf("grant from non-master")
		}
		return r.acceptGrant(*packet)
	case "begin":
		if err := r.validateForwards(source, *packet); err != nil {
			return err
		}
		if r.master {
			if value, expected := r.steps.Load(packet.ID); expected {
				step := value.(*replicaStep)
				if !step.matches(source, *packet) {
					return fmt.Errorf("unexpected transfer source")
				}
			} else {
				r.mu.Lock()
				record := r.directory.Records[r.resourceKey(packet.Resource)]
				r.mu.Unlock()
				if record.Owner != source || record.Epoch != packet.Epoch {
					return fmt.Errorf("backup from non-owner: %s epoch %d, directory epoch %d, owner matches %t", packet.Resource.key(), packet.Epoch, record.Epoch, record.Owner == source)
				}
				if err := r.authorizePacket(*packet); err != nil {
					return err
				}
			}
		} else {
			if source != "" {
				return fmt.Errorf("transfer from non-master")
			}
			r.mu.Lock()
			epoch := r.directory.Records[r.resourceKey(packet.Resource)].Epoch
			r.mu.Unlock()
			if packet.Epoch < epoch {
				return fmt.Errorf("stale transfer epoch")
			}
		}
		key := source + "/" + packet.ID
		if _, exists := r.imports.Load(key); exists {
			return fmt.Errorf("duplicate transfer")
		}
		transfer, err := r.startImport(*packet)
		if err != nil {
			return err
		}
		for _, route := range packet.Forwards {
			r.forwardResource(route.From, route.To)
		}
		publication := r.resourceCell(mosiResource{Database: packet.Resource.Database})
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
		if packet.Resource != transfer.header.Resource || packet.Epoch != transfer.header.Epoch || packet.Content != transfer.header.Content {
			return fmt.Errorf("transfer header changed")
		}
		if packet.Kind != "end" {
			return transfer.packet(*packet)
		}
		if err := func() error {
			if packet.Resource.Shard != "" {
				// All bytes are received. Final data publication shares existing cold
				// lifecycle ownership with schema installation; no lease spans a stream
				// or a network wait, and granted query reads never acquire this lease.
				db := databases.Get(packet.Resource.Database)
				if db == nil {
					return fmt.Errorf("missing import database")
				}
				db.persistenceLifecycle.RLock()
				defer db.persistenceLifecycle.RUnlock()
				publication := r.resourceCell(mosiResource{Database: packet.Resource.Database})
				publication.importPublication.Lock()
				defer publication.importPublication.Unlock()
			}
			defer r.imports.Delete(key)
			defer transfer.abort()
			if r.master {
				r.mu.Lock()
				record := r.directory.Records[r.resourceKey(packet.Resource)]
				r.mu.Unlock()
				if record.Owner != source || record.Epoch != packet.Epoch {
					return fmt.Errorf("stale transfer epoch")
				}
			}
			if err := func() error {
				if !r.master {
					// The reader owns staged bytes; acquire serializes only their final
					// publication with grant/revocation workers, never ordinary query reads.
					cell := r.resourceCell(packet.Resource)
					cell.acquire.Lock()
					defer cell.acquire.Unlock()
					r.mu.Lock()
					epoch := r.directory.Records[r.resourceKey(packet.Resource)].Epoch
					r.mu.Unlock()
					if packet.Epoch < epoch {
						return fmt.Errorf("stale transfer epoch")
					}
				}
				if err := r.finishImport(source, transfer); err != nil {
					return err
				}
				if !transfer.superseded {
					return r.saveBackup(packet.Resource, packet.Content)
				}
				return nil
			}(); err != nil {
				return err
			}
			if r.master && !transfer.superseded {
				r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
					record.Content = packet.Content
					return true
				})
			}
			return nil
		}(); err != nil {
			return err
		}
		if !r.master {
			return peer.post(mosiPacket{Kind: "ack", ID: packet.ID, Resource: packet.Resource, Epoch: packet.Epoch, Content: packet.Content})
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

func (r *replicationManager) authorizePacket(packet mosiPacket) error {
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

// The protocol entry resolves its domain object and delegates. Storage/auth,
// import publication and MOSI transitions remain in the object's engine adapter;
// the network reader neither executes them nor knows shard/schema policy.
func (r *replicationManager) receivePacket(source string, packet *mosiPacket) error {
	if !packet.Resource.valid() || packet.ID == "" {
		return fmt.Errorf("invalid resource packet")
	}
	object := r.resourceCell(packet.Resource)
	_, err := object.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "packet", Payload: replicaPacketInvocation{source, packet}})
	return err
}

type replicaPacketInvocation struct {
	source string
	packet *mosiPacket
}

func (cell *replicaCell) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	invocation, ok := request.Payload.(replicaPacketInvocation)
	if request.Method != "packet" || !ok || cell.engine == nil {
		return nil, fmt.Errorf("invalid native coherence request")
	}
	return nil, cell.engine.applyResourcePacket(invocation.source, invocation.packet)
}
