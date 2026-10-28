/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sort"
import "context"
import "encoding/json"
import "github.com/launix-de/memcp/storage/mosi"

// Location is routing metadata in the existing MOSI resource directory. It is
// never a grant, a proof that backup files are current, or a second directory.
// Schema transfers carry bounded batches before installing their reference.
type clusterSchemaLocations struct {
	Header  mosiPacket      `json:"header"`
	Entries []replicaRecord `json:"entries"`
	After   string          `json:"after,omitempty"`
}

func locationRecord(record replicaRecord) replicaRecord {
	return replicaRecord{Resource: record.Resource, Owner: record.Owner, Epoch: record.Epoch, Content: record.Content}
}

// The schema handover's directory ownership serializes this snapshot with
// incoming location updates. Resource rights remain with their native cells.
func (cluster *nodeCluster) transferSchemaLocations(ctx context.Context, target string, header mosiPacket) error {
	for _, resource := range cluster.bootstrapResources {
		if resource.Database == header.Resource.Database {
			cluster.bootstrapCell(resource)
		}
	}
	r := cluster.manager
	r.mu.Lock()
	entries := make([]replicaRecord, 0)
	for _, record := range r.directory.Records {
		if record.Resource.Database == header.Resource.Database && record.Resource.Shard != "" && record.Owner != "" {
			entries = append(entries, locationRecord(record))
		}
	}
	r.mu.Unlock()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Resource.key() < entries[j].Resource.key() })
	peer := cluster.peer(target)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	for start := 0; start < len(entries); start += 64 {
		end := min(start+64, len(entries))
		request := clusterSchemaLocations{Header: header, Entries: entries[start:end]}
		if err := cluster.service.control(ctx, peer, "cluster.schema_locations", request, nil); err != nil {
			return err
		}
	}
	return nil
}

// A restarted schema owner recovers its RAM directory from the same live
// participants which prove its native authority. This is schema discovery,
// once per recovery, never a broadcast for each cold shard. Entries restore
// routes only; even a reported owner cannot grant backend permission here.
func (cluster *nodeCluster) recoverSchemaLocations(ctx context.Context, database string, links []*clusterLink) error {
	for _, link := range links {
		request := clusterSchemaLocations{Header: mosiPacket{Resource: mosiResource{Database: database}}}
		for {
			var page clusterSchemaLocations
			if err := cluster.service.control(ctx, link.peer, "cluster.schema_locations", request, &page); err != nil {
				return err
			}
			if page.Header.Resource != request.Header.Resource || len(page.Entries) > 64 {
				return fmt.Errorf("invalid schema location recovery")
			}
			for _, entry := range page.Entries {
				if !entry.Resource.valid() || entry.Resource.Database != database || entry.Resource.Shard == "" || entry.Resource.key() <= request.After || entry.Owner == "" || entry.Epoch == 0 {
					return fmt.Errorf("invalid recovered shard location")
				}
				cluster.installLocationHint(entry)
			}
			if len(page.Entries) < 64 {
				break
			}
			request.After = page.Entries[len(page.Entries)-1].Resource.key()
		}
	}
	return nil
}

func (cluster *nodeCluster) installSchemaLocations(source string, request clusterSchemaLocations) error {
	if !request.Header.Resource.valid() || request.Header.Resource.Shard != "" || !cluster.databases[request.Header.Resource.Database] || len(request.Entries) > 64 {
		return fmt.Errorf("invalid schema locations")
	}
	schema := cluster.manager.resourceCell(request.Header.Resource)
	schema.acquire.Lock()
	defer schema.acquire.Unlock()
	cluster.manager.mu.Lock()
	record := cluster.manager.directory.Records[request.Header.Resource.Database]
	cluster.manager.mu.Unlock()
	if _, err := cluster.referenceSource(source, request.Header, schema, record); err != nil {
		return err
	}
	for _, entry := range request.Entries {
		if !entry.Resource.valid() || entry.Resource.Shard == "" || entry.Resource.Database != request.Header.Resource.Database || entry.Owner == "" || entry.Epoch == 0 {
			return fmt.Errorf("invalid shard location")
		}
	}
	for _, entry := range request.Entries {
		cluster.installLocationHint(entry)
	}
	return nil
}

func (cluster *nodeCluster) installLocationHint(entry replicaRecord) {
	r := cluster.manager
	cell := r.resourceCell(entry.Resource).resolved()
	cell.acquire.Lock()
	defer cell.acquire.Unlock()
	r.updateRecord(entry.Resource, func(current *replicaRecord) bool {
		// Hints must never replace native grants, an active acquisition, or a newer
		// handover. Only the authoritative native path publishes local permissions.
		phase := cell.ownstate.Phase()
		if phase != mosi.IO || current.Epoch > entry.Epoch || current.Epoch == entry.Epoch && current.Owner != "" && current.Owner != entry.Owner {
			return false
		}
		current.Owner, current.Epoch, current.Content = entry.Owner, entry.Epoch, entry.Content
		return true
	})
}

// A rebuild changes physical UUIDs without moving the shard owner. A cold
// directory miss may consume the existing schema checkpoint's forwarding edges;
// this changes identity routing only, never MOSI permission or backend freshness.
func (cluster *nodeCluster) shardLocation(resource mosiResource) (replicaRecord, error) {
	r := cluster.manager
	lookup := func() replicaRecord {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.directory.Records[r.resourceKey(resource)]
	}
	entry := lookup()
	if entry.Owner == "" {
		db := GetDatabase(resource.Database)
		if db == nil {
			return entry, fmt.Errorf("unknown location schema")
		}
		access := db.GetRead(nil)
		defer access.Release()
		if err := r.restoreSchemaRoutes(db.persistence.ReadSchema()); err != nil {
			return entry, err
		}
		entry = lookup()
	}
	if entry.Owner == "" {
		return entry, fmt.Errorf("schema owner has no shard location")
	}
	return locationRecord(entry), nil
}

func (cluster *nodeCluster) locateViaSchema(ctx context.Context, resource mosiResource) (string, error) {
	schemaOwner, err := cluster.locate(ctx, mosiResource{Database: resource.Database})
	if err != nil {
		return "", err
	}
	if schemaOwner == cluster.self.ID {
		entry, err := cluster.shardLocation(resource)
		if err != nil {
			return "", err
		}
		if entry.Owner == cluster.self.ID {
			cell := cluster.manager.resourceCell(entry.Resource)
			phase := cell.ownstate.Phase()
			if phase == mosi.IO {
				if cell.ownstate.Initialized() {
					return "", fmt.Errorf("local invalid state has no backend authority")
				}
				return cluster.locate(ctx, entry.Resource) // explicit live-peer recovery
			}
		}
		return entry.Owner, nil
	}
	peer := cluster.peer(schemaOwner)
	if peer == nil {
		return "", fmt.Errorf(mosi.NetworkError)
	}
	var record replicaRecord
	if err := cluster.service.control(ctx, peer, "cluster.location", resource, &record); err != nil {
		return "", err
	}
	if !record.Resource.valid() || record.Resource.Database != resource.Database || record.Resource.Table != resource.Table || record.Owner == "" {
		return "", fmt.Errorf("invalid schema owner location")
	}
	cluster.installLocationHint(record)
	return record.Owner, nil
}

// Called after native handover has released its directory lock. The schema
// directory serializes registration against its own handover snapshot. A stale
// schema route forwards to its successor rather than recording at a former owner.
func (cluster *nodeCluster) registerLocation(ctx context.Context, entry replicaRecord) error {
	owner, err := cluster.locate(ctx, mosiResource{Database: entry.Resource.Database})
	if err != nil {
		return err
	}
	if owner != cluster.self.ID {
		peer := cluster.peer(owner)
		if peer == nil {
			return fmt.Errorf(mosi.NetworkError)
		}
		return cluster.service.control(ctx, peer, "cluster.location_update", locationRecord(entry), nil)
	}
	schema := cluster.manager.resourceCell(mosiResource{Database: entry.Resource.Database})
	schema.directory.Lock()
	cluster.manager.mu.Lock()
	currentOwner := cluster.manager.directory.Records[entry.Resource.Database].Owner
	cluster.manager.mu.Unlock()
	if currentOwner != cluster.self.ID {
		schema.directory.Unlock()
		return cluster.registerLocation(ctx, entry)
	}
	cluster.installLocationHint(entry)
	schema.directory.Unlock()
	return nil
}

func (cluster *nodeCluster) coordinate(ctx context.Context, source string, packet mosiPacket) error {
	if err := cluster.manager.coordinate(ctx, source, packet); err != nil {
		return err
	}
	if packet.Resource.Shard != "" {
		cluster.manager.mu.Lock()
		entry := cluster.manager.directory.Records[cluster.manager.resourceKey(packet.Resource)]
		cluster.manager.mu.Unlock()
		return cluster.registerLocation(ctx, entry)
	}
	return nil
}

func (cluster *nodeCluster) handleLocations(ctx context.Context, source, method string, data json.RawMessage) (any, error) {
	if err := cluster.requireMember(source); err != nil {
		return nil, err
	}
	switch method {
	case "cluster.schema_locations":
		var request clusterSchemaLocations
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		if request.Header.ID == "" {
			resource := request.Header.Resource
			if !resource.valid() || resource.Shard != "" || !cluster.databases[resource.Database] {
				return nil, fmt.Errorf("invalid schema location inventory")
			}
			r := cluster.manager
			r.mu.Lock()
			var entries []replicaRecord
			for _, record := range r.directory.Records {
				if record.Resource.Database == resource.Database && record.Resource.Shard != "" && record.Resource.key() > request.After && record.Owner != "" {
					entries = append(entries, locationRecord(record))
				}
			}
			r.mu.Unlock()
			sort.Slice(entries, func(i, j int) bool { return entries[i].Resource.key() < entries[j].Resource.key() })
			return clusterSchemaLocations{Header: request.Header, Entries: entries[:min(len(entries), 64)]}, nil
		}
		return nil, cluster.installSchemaLocations(source, request)
	case "cluster.location_update":
		var entry replicaRecord
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, err
		}
		if !entry.Resource.valid() || entry.Resource.Shard == "" || !cluster.databases[entry.Resource.Database] || entry.Owner == "" || entry.Epoch == 0 {
			return nil, fmt.Errorf("invalid shard location update")
		}
		return nil, cluster.registerLocation(ctx, entry)
	case "cluster.location":
		var resource mosiResource
		if err := json.Unmarshal(data, &resource); err != nil {
			return nil, err
		}
		if !resource.valid() || resource.Shard == "" || !cluster.databases[resource.Database] {
			return nil, fmt.Errorf("invalid shard location request")
		}
		// Schema discovery may broadcast. Per-shard routing consults exactly this
		// owner and, if needed, follows the existing resource-owner forwarding edge.
		owner, err := cluster.locate(ctx, mosiResource{Database: resource.Database})
		if err != nil {
			return nil, err
		}
		if owner != cluster.self.ID {
			peer := cluster.peer(owner)
			if peer == nil {
				return nil, fmt.Errorf(mosi.NetworkError)
			}
			var entry replicaRecord
			err := cluster.service.control(ctx, peer, method, resource, &entry)
			return entry, err
		}
		cluster.bootstrapCell(resource)
		return cluster.shardLocation(resource)
	}
	return nil, fmt.Errorf("unknown location method")
}
