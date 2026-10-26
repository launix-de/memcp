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
import "hash/fnv"
import "context"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/storage/mosi"

// A database's directory location is pinned in RAM after agreement with every
// known participant. It is a routing choice, not database-X: schema and every
// logical shard retain independent MOSI cells/requests. New peers discover an
// existing home; joining never rehashes live rights. A missing witness/home
// fails closed rather than authorizing a second directory on a timeout.
// This small-mesh agreement will later travel over URI-addressed neighbour hops.
type clusterHomeRequest struct{ Database, Home string }

func (cluster *nodeCluster) home(ctx context.Context, name string) (string, error) {
	if value, ok := cluster.homes.Load(name); ok {
		return value.(string), nil
	}
	c := cluster.manager.resourceCell(mosiResource{Database: name})
	c.bootstrap.Lock()
	defer c.bootstrap.Unlock()
	if value, ok := cluster.homes.Load(name); ok {
		return value.(string), nil
	}
	view := cluster.view.Load()
	if cluster.seed != "" && len(view.members) == 1 {
		return "", fmt.Errorf(mosi.NetworkError)
	}
	if len(view.links) != len(view.members)-1 {
		return "", fmt.Errorf(mosi.NetworkError)
	}
	candidate := ""
	for _, link := range view.links {
		var answer string
		if err := link.peer.call(ctx, "cluster.home", clusterHomeRequest{Database: name}, &answer); err != nil {
			return "", err
		}
		if answer != "" {
			if candidate != "" && candidate != answer {
				return "", fmt.Errorf("conflicting cluster directory witnesses")
			}
			candidate = answer
		}
	}
	if candidate == "" {
		// Rendezvous placement distributes independent database directories. It is
		// consulted only for a previously unseen DB, never to revoke an offline owner.
		var best uint64
		for _, member := range view.members {
			hash := fnv.New64a()
			_, _ = hash.Write([]byte(name + "\x00" + member.ID))
			score := hash.Sum64()
			if candidate == "" || score > best {
				candidate, best = member.ID, score
			}
		}
	}
	for _, link := range view.links {
		var answer string
		if err := link.peer.call(ctx, "cluster.home", clusterHomeRequest{name, candidate}, &answer); err != nil {
			return "", err
		}
		if answer != candidate {
			return "", fmt.Errorf("conflicting cluster directory witnesses")
		}
	}
	actual, loaded := cluster.homes.LoadOrStore(name, candidate)
	if loaded && actual.(string) != candidate {
		return "", fmt.Errorf("conflicting cluster directory witnesses")
	}
	return candidate, nil
}
func (cluster *nodeCluster) homeMessage(request clusterHomeRequest) (string, error) {
	if !cluster.databases[request.Database] {
		return "", fmt.Errorf("unknown cluster database")
	}
	if request.Home == "" {
		if value, ok := cluster.homes.Load(request.Database); ok {
			return value.(string), nil
		}
		return "", nil
	}
	if validateClusterURI(request.Home) != nil {
		return "", fmt.Errorf("invalid directory URI")
	}
	known := false
	for _, member := range cluster.view.Load().members {
		if member.ID == request.Home {
			known = true
			break
		}
	}
	if !known {
		return "", fmt.Errorf("unknown directory URI")
	}
	actual, _ := cluster.homes.LoadOrStore(request.Database, request.Home)
	return actual.(string), nil
}
func (cluster *nodeCluster) peer(id string) *mosiPeer {
	for _, link := range cluster.view.Load().links {
		if link.member.ID == id {
			return link.peer
		}
	}
	return nil
}
func (cluster *nodeCluster) obtain(resource mosiResource, wanted mosi.Phase) *replicaCell {
	r := cluster.manager
	c := r.resourceCell(resource)
	if phase := c.ownstate.Phase(); phase == mosi.M || wanted != mosi.M && phase.Readable() {
		return c
	}
	ctx, cancel := context.WithTimeout(cluster.ctx, 30*time.Minute)
	defer cancel()
	home, err := cluster.home(ctx, resource.Database)
	if err != nil {
		panic(mosi.NetworkError)
	}
	for {
		publication, action := c.ownstate.Request(wanted)
		if action == mosi.Ready {
			return c
		}
		if action == mosi.Send || action == mosi.Upgrade {
			packet := mosiPacket{Kind: "read", Resource: resource, ID: publication.Flight().ID(), Sequence: r.requestSequence(publication.Flight())}
			if publication.Phase() == mosi.MR {
				packet.Kind = "write"
			}
			if home == r.id {
				go func() {
					if failure := r.coordinate(ctx, r.id, packet); failure != nil {
						c.ownstate.Fail(packet.ID, failure, wanted)
					}
				}()
			} else {
				peer := cluster.peer(home)
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
		if home != r.id {
			if peer := cluster.peer(home); peer != nil {
				disconnected = peer.done
			} else {
				c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), wanted)
			}
		}
		select {
		case <-publication.Flight().Done():
			if publication.Flight().Err() != nil {
				panic(mosi.NetworkError)
			}
		case <-ctx.Done():
			c.ownstate.Fail(publication.Flight().ID(), ctx.Err(), wanted)
			panic(mosi.NetworkError)
		case <-disconnected:
			c.ownstate.Fail(publication.Flight().ID(), fmt.Errorf(mosi.NetworkError), wanted)
			panic(mosi.NetworkError)
		}
	}
}

// Directory recovery asks live participants for their published/retained rights.
// It neither restores MOSI from files nor promotes an unavailable owner. A
// partial handover is deliberately refused until its surviving participants
// can complete recovery; current backend bytes alone prove no permission.
func (cluster *nodeCluster) report(resource mosiResource) replicaRecord {
	r := cluster.manager
	r.mu.Lock()
	record := r.directory.Records[resource.key()]
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
func (cluster *nodeCluster) recoverDirectory(ctx context.Context, resource mosiResource) error {
	r := cluster.manager
	r.mu.Lock()
	known := r.directory.Records[resource.key()].Owner != ""
	r.mu.Unlock()
	if known {
		return nil
	}
	owner := ""
	generation := ""
	var epoch uint64
	readers := map[string]bool{}
	reports := map[string]replicaRecord{r.id: cluster.report(resource)}
	view := cluster.view.Load()
	if len(view.links) != len(view.members)-1 {
		return fmt.Errorf(mosi.NetworkError)
	}
	for _, link := range view.links {
		var record replicaRecord
		if err := link.peer.call(ctx, "cluster.report", resource, &record); err != nil {
			return err
		}
		reports[link.member.ID] = record
	}
	for id, record := range reports {
		if record.Handoff != "" {
			return fmt.Errorf("cluster handover recovery is incomplete")
		}
		if record.State == "M" || record.State == "O" {
			if owner != "" && owner != id {
				return fmt.Errorf("conflicting cluster owners")
			}
			owner, generation = id, record.Generation
		}
		if record.State == "S" {
			readers[id] = true
		}
		epoch = max(epoch, record.Epoch)
	}
	if owner == "" {
		// A remembered remote custodian must not disappear merely because its link
		// is down or its current publication is IO during an interrupted transfer.
		for _, record := range reports {
			if record.Owner != "" {
				return fmt.Errorf("cluster owner cannot yet be recovered")
			}
		}
		owner = r.id
	}
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Resource, record.Owner, record.Epoch, record.Generation = resource, owner, max(epoch, 1), generation
		record.Readers = readers
		record.State = "IO"
		if owner == r.id {
			record.State = "M"
		} else if readers[r.id] {
			record.State = "S"
		}
		return true
	})
	return nil
}
func (cluster *nodeCluster) receive(source string, packet *mosiPacket) error {
	r := cluster.manager
	if !cluster.databases[packet.Resource.Database] {
		return fmt.Errorf("resource outside cluster domain")
	}
	home, ok := cluster.homes.Load(packet.Resource.Database)
	if !ok {
		return fmt.Errorf("unknown cluster directory")
	}
	switch packet.Kind {
	case "reference":
		return cluster.installReference(source, *packet)
	case "declare":
		if home.(string) != r.id {
			return fmt.Errorf("declaration at wrong directory")
		}
		c := r.resourceCell(packet.Resource)
		c.directory.Lock()
		defer c.directory.Unlock()
		r.mu.Lock()
		schema := r.directory.Records[mosiResource{Database: packet.Resource.Database}.key()]
		r.mu.Unlock()
		if schema.Owner != source || len(schema.Readers) != 0 {
			return fmt.Errorf("new shard requires schema M")
		}
		if _, err := uuid.Parse(packet.Generation); err != nil {
			return err
		}
		r.updateRecord(packet.Resource, func(record *replicaRecord) bool {
			if record.Owner != "" {
				return false
			}
			record.Resource, record.Owner, record.State, record.Epoch, record.Generation = packet.Resource, source, "IO", 1, packet.Generation
			record.Readers = map[string]bool{}
			return true
		})
		c.clusterstate.Establish(mosi.M)
		return r.peer(source).post(mosiPacket{Kind: "ack", Resource: packet.Resource, ID: packet.ID, Generation: packet.Generation})
	case "read", "write":
		if home.(string) != r.id {
			return fmt.Errorf("request at wrong directory")
		}
	case "recall", "invalidate", "grant":
		if source != home.(string) {
			return fmt.Errorf("authority message from wrong directory")
		}
	}
	return r.receivePacket(source, packet)
}
func (cluster *nodeCluster) declareShard(shard *storageShard, local bool) *replicaCell {
	r := cluster.manager
	resource := mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.logicalID().String()}
	c := r.resourceCell(resource)
	c.current.CompareAndSwap(nil, shard)
	shard.coherence.CompareAndSwap(nil, c)
	if !local || c.ownstate.Initialized() {
		return c
	}
	ctx, cancel := context.WithTimeout(cluster.ctx, 30*time.Second)
	defer cancel()
	home, err := cluster.home(ctx, resource.Database)
	if err != nil {
		panic(err)
	}
	if home != r.id {
		if err := r.step(ctx, home, mosiPacket{Kind: "declare", ID: uuid.NewString(), Resource: resource, Generation: shard.uuid.String()}); err != nil {
			panic(err)
		}
	}
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
	home, _ := cluster.homes.Load(packet.Resource.Database)
	if source != home && r.id != home {
		return fmt.Errorf("reference outside directory handover")
	}
	if r.id == home && source != r.id {
		value, exists := r.steps.Load(packet.ID)
		if !exists || !value.(*replicaStep).matches(source, packet) {
			return fmt.Errorf("unsolicited owner reference")
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
			return fmt.Errorf("shared schema generation changed")
		}
		if err := r.installSchema(source, db.Name, db.persistence, data); err != nil {
			return err
		}
	} else {
		if err := r.finishImport(source, &replicaImport{header: packet, engine: db.persistence}); err != nil {
			return err
		}
	}
	r.updateRecord(packet.Resource, func(record *replicaRecord) bool { record.Generation = packet.Generation; return true })
	if value, exists := r.steps.Load(packet.ID); exists {
		value.(*replicaStep).finish(nil)
	}
	peer := r.peer(source)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	return peer.post(mosiPacket{Kind: "ack", ID: packet.ID, Resource: packet.Resource, Epoch: packet.Epoch, Generation: packet.Generation})
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

func decodeClusterHome(data json.RawMessage) (clusterHomeRequest, error) {
	var request clusterHomeRequest
	err := json.Unmarshal(data, &request)
	return request, err
}
