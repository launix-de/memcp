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
import "strings"
import "github.com/launix-de/memcp/storage/mosi"

type replicaRecord struct {
	Forwards       []shardForward                `json:"forwards,omitempty"`
	Handoff        string                        `json:"handoff,omitempty"`
	InitialClaimed bool                          `json:"initial_claimed,omitempty"`
	Epoch          uint64                        `json:"epoch"`
	State          string                        `json:"state"`
	Owner          string                        `json:"owner,omitempty"`
	Readers        map[string]bool               `json:"readers,omitempty"`
	Resource       mosiResource                  `json:"resource"`
	Content        string                        `json:"content,omitempty"`
	Requests       map[string]replicaRequestMark `json:"requests,omitempty"`
}

type replicaRequestMark struct {
	Sequence  uint64 `json:"sequence"`
	Write     bool   `json:"write"`
	ID        string `json:"id,omitempty"`
	Completed bool   `json:"completed,omitempty"`
}

type replicaDirectory struct {
	Records     map[string]replicaRecord `json:"records"`
	Users       map[string]replicaAuth   `json:"users"`
	Retired     map[string]bool          `json:"retired"`
	Initialized map[string]bool          `json:"initialized,omitempty"`
	Sequence    uint64                   `json:"sequence,omitempty"`
}

// updateRecord publishes one immutable runtime directory entry. MOSI rights,
// ownership, epochs and request sequences are never written to a backend.
func (r *replicationManager) updateRecord(resource mosiResource, update func(*replicaRecord) bool) replicaRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.directory.Records[r.resourceKey(resource)]
	if record.State == "" {
		record.Resource, record.State = resource, "IO"
	}
	record.Readers, record.Requests = maps.Clone(record.Readers), maps.Clone(record.Requests)
	if update(&record) {
		record.Resource = r.canonicalResource(resource)
		r.directory.Records[r.resourceKey(resource)] = record
	}
	return record
}

func (r *replicationManager) storeConfig(entry replicaConfigEntry) error {
	if entry.Kind != "users" && entry.Kind != "retired" {
		return fmt.Errorf("not persistent replication configuration: %s", entry.Kind)
	}
	if entry.Kind == "users" && entry.Auth == nil {
		return fmt.Errorf("missing saved user proof")
	}
	if r.config != nil {
		if err := r.writeConfig(entry); err != nil {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch entry.Kind {
	case "users":
		r.directory.Users[entry.Name] = *entry.Auth
	case "retired":
		r.directory.Retired[entry.Name] = entry.Value

	}
	return nil
}

func (r *replicationManager) cell(name string) *replicaCell {
	if snapshot := r.cellSnapshot.Load(); snapshot != nil {
		if c := (*snapshot)[name]; c != nil {
			return c.resolved()
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.cells[name]
	if c == nil {
		c = &replicaCell{}
		r.cells[name] = c
		snapshot := make(map[string]*replicaCell, len(r.cells))
		for name, cell := range r.cells {
			snapshot[name] = cell
		}
		r.cellSnapshot.Store(&snapshot)
	}
	return c
}

// canonicalResource follows the same UUID successor relation as shard.next.
// The immutable lookup snapshot contains only routing aliases, not payload
// references; old query owners alone retain old shard objects.
func (r *replicationManager) canonicalResource(resource mosiResource) mosiResource {
	if snapshot := r.cellSnapshot.Load(); snapshot != nil {
		if cell := (*snapshot)[resource.key()]; cell != nil {
			if current := cell.resolved().resource.Load(); current != nil {
				return *current
			}
		}
	}
	return resource
}

func (r *replicationManager) resourceKey(resource mosiResource) string {
	return r.canonicalResource(resource).key()
}

type shardForward struct {
	From mosiResource `json:"from"`
	To   mosiResource `json:"to"`
}

// forwardResource publishes a UUID route at rebuild/import publication, outside
// row loops. The successor borrows the existing authority and pending flight;
// M/O and reader directories are never cloned into independent writers.
func (r *replicationManager) forwardResource(from, to mosiResource) {
	if from == to {
		return
	}
	if !from.valid() || !to.valid() || from.Shard == "" || to.Shard == "" || from.Database != to.Database {
		panic("invalid shard forwarding")
	}
	cell := r.resourceCell(from)
	r.mu.Lock()
	defer r.mu.Unlock()
	cell = cell.resolved()
	to = r.canonicalResource(to)
	current := *cell.resource.Load()
	if current == to {
		return
	}
	if existing := r.cells[to.key()]; existing != nil && existing.resolved() != cell {
		existing = existing.resolved()
		// Prefer already granted authority to a placeholder/request. Otherwise
		// retain the destination flight and wake source waiters to retry there.
		primary, other := cell, existing
		if authorityRank(existing.ownstate.Phase()) > authorityRank(cell.ownstate.Phase()) || existing.ownstate.Phase().Requested() && cell.ownstate.Phase().Requested() {
			primary, other = existing, cell
		}
		if other.ownstate.Phase() == mosi.M && primary.ownstate.Phase() == mosi.M {
			panic("conflicting shard write authorities")
		}
		if shard := other.current.Load(); shard != nil {
			primary.current.CompareAndSwap(nil, shard)
		}
		if retired := other.retiredShards.Load(); retired != nil {
			for _, entry := range *retired {
				primary.retainShard(entry.Value())
			}
		}
		other.forwardingDone()
		other.forward.Store(primary)
		close(other.forwardDone)
		cell = primary
	}
	r.cells[to.key()] = cell

	if record, exists := r.directory.Records[current.key()]; exists {
		record.Resource = to
		if target, present := r.directory.Records[to.key()]; present {
			requests := maps.Clone(target.Requests)
			if requests == nil {
				requests = make(map[string]replicaRequestMark)
			}
			for peer, request := range record.Requests {
				if previous, exists := requests[peer]; !exists || previous.Sequence < request.Sequence {
					requests[peer] = request
				}
			}
			if target.Epoch > record.Epoch {
				record = target
			}
			record.Requests = requests
		}
		r.directory.Records[to.key()] = record
		delete(r.directory.Records, current.key())
	}
	if owner := r.confirmed[current.key()]; owner != "" {
		r.confirmed[to.key()] = owner
		delete(r.confirmed, current.key())
	}
	for _, claims := range r.initialClaims {
		if claims[current.key()] {
			claims[to.key()] = true
			delete(claims, current.key())
		}
	}
	if reports := r.recoveryReports[current.key()]; reports != nil {
		r.recoveryReports[to.key()] = reports
		delete(r.recoveryReports, current.key())
	}
	if backup, exists := r.backups[current.key()]; exists {
		r.backups[to.key()] = backup
		delete(r.backups, current.key())
	}
	cell.resource.Store(&to)
	snapshot := make(map[string]*replicaCell, len(r.cells))
	for key, value := range r.cells {
		snapshot[key] = value.resolved()
		r.cells[key] = snapshot[key]
	}
	r.cellSnapshot.Store(&snapshot)
}

// routes are captured with an outgoing transfer. They name actual old/new UUIDs
// so a peer can resolve a pre-rebuild stub without another logical identity.
func (r *replicationManager) shardRoutes(resource mosiResource) []shardForward {
	var routes []shardForward
	if snapshot := r.cellSnapshot.Load(); snapshot != nil {
		for key, cell := range *snapshot {
			cell = cell.resolved()
			to := cell.resource.Load()
			if to == nil || to.Database != resource.Database || to.Shard == "" || key == to.key() {
				continue
			}
			if resource.Shard != "" {
				bound := (*snapshot)[resource.key()]
				if bound == nil || cell != bound.resolved() {
					continue
				}
			}
			from := *to
			from.Shard = strings.TrimPrefix(key, to.Database+"/")
			routes = append(routes, shardForward{from, *to})
		}
	}
	return routes
}

// Incoming transfers may introduce a rebuild successor before its column bytes.
// Only the authenticated current modifier may redirect a known resource. The
// wire request keeps its original UUID so in-flight ACK matching stays exact.
func (r *replicationManager) validateForwards(source string, packet mosiPacket) error {
	if len(packet.Forwards) == 0 {
		return nil
	}
	for _, route := range packet.Forwards {
		if !route.From.valid() || !route.To.valid() || route.From.Shard == "" || route.To.Shard == "" || route.From.Database != packet.Resource.Database || route.From.Table != packet.Resource.Table || route.To.Database != packet.Resource.Database || route.To.Table != packet.Resource.Table || route.To.Shard != packet.Content {
			return fmt.Errorf("invalid transfer forwarding")
		}
	}
	// Cluster references validate their pending flight/owner in installReference;
	// trusted peer routing may legitimately arrive from a newer owner than our
	// URI hint. Replica clients instead need the current directory's proof.
	if r.cluster == nil && r.master {
		if value, expected := r.steps.Load(packet.ID); expected {
			if !value.(*replicaStep).matches(source, packet) {
				return fmt.Errorf("unexpected forwarding source")
			}
		} else if r.cluster == nil {
			if err := r.authorizePacket(packet); err != nil {
				return err
			}
		}
		r.restoreResourceRoutes(packet.Resource)
		allowed := false
		r.mu.Lock()
		for _, route := range packet.Forwards {
			record := r.directory.Records[r.resourceKey(route.From)]
			if record.Owner == source && record.Epoch == packet.Epoch {
				allowed = true
			}
		}
		r.mu.Unlock()
		if !allowed {
			return fmt.Errorf("forwarding from non-owner")
		}
	} else if r.cluster == nil && source != "" {
		return fmt.Errorf("forwarding from non-master")
	}
	return nil
}

func authorityRank(phase mosi.Phase) int {
	switch phase {
	case mosi.M:
		return 4
	case mosi.O:
		return 3
	case mosi.S:
		return 2
	}
	if phase.Requested() {
		return 1
	}
	return 0
}

func (r *replicationManager) shardResource(shard *storageShard) mosiResource {
	return mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()}
}
func (r *replicationManager) lookupCell(resource mosiResource) *replicaCell {
	if snapshot := r.cellSnapshot.Load(); snapshot != nil {
		if cell := (*snapshot)[resource.key()]; cell != nil {
			return cell.resolved()
		}
	}
	return nil
}

// UUID edges are native topology, not restored permissions. Loading them before
// peer reports prevents a stale stub from reversing a newer durable rebuild.
func (r *replicationManager) restoreTableRoutes(table *table) {
	if table == nil || table.schema == nil || table.schemaTopology.Load() == nil {
		return
	}
	table.replicaRoutes.Do(func() {
		if topology := table.schemaTopology.Load(); topology != nil {
			for from, to := range topology.forwards {
				r.forwardResource(mosiResource{Database: table.schema.Name, Table: table.Name, Shard: from}, mosiResource{Database: table.schema.Name, Table: table.Name, Shard: to})
			}
		}
	})
}
func (r *replicationManager) restoreResourceRoutes(resource mosiResource) {
	if db := databases.Get(resource.Database); db != nil {
		db.ensureLoaded()
		if resource.Shard == "" {
			for _, table := range db.tables.GetAll() {
				r.restoreTableRoutes(table)
			}
		} else if table := db.tables.Get(resource.Table); table != nil {
			r.restoreTableRoutes(table)
		}
	}
}
