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
import "sync"
import "strings"
import "reflect"
import "encoding/binary"
import "bytes"
import "context"
import "encoding/json"
import "github.com/google/uuid"

// The receiver owns one private backend writer per active transfer. Packet
// memory is bounded by the connection queue; schema JSON uses the existing
// byte-slice persistence API, while column/blob bytes go directly to writers.
type replicaImport struct {
	header     replicaPacket
	engine     PersistenceEngine
	writer     io.WriteCloser
	schema     bytes.Buffer
	objectKind string
	objectName string
}

type replicaStep struct {
	peer     string
	resource replicaResource
	done     chan struct{}
	once     sync.Once
	err      error
}

func (step *replicaStep) finish(err error) { step.once.Do(func() { step.err = err; close(step.done) }) }

func (r *replicationManager) resourceCell(resource replicaResource) *replicaCell {
	c := r.cell(resource.key())
	if c.resource.Load() == nil {
		publication := resource
		c.resource.CompareAndSwap(nil, &publication)
	}
	return c
}

func (r *replicationManager) publishLocal(resource replicaResource, phase mosiPhase) {
	c := r.resourceCell(resource)
	if phase == mosiIO && resource.Shard != "" {
		// Cached analytic results must stop being reusable before a remote
		// writer changes this source. This is publication work, not a scan/read
		// counter, and causes ordinary cache initialization to acquire GetRead.
		if shard := c.current.Load(); shard != nil {
			invalidateReplicaCaches(shard.t)
		}
	}
	c.rights.publish(phase)
}

// Mark runtime dependency targets without acquiring their shard/proxy locks or
// executing SQL triggers. A cache query can itself be waiting for network data;
// the reader must acknowledge revocation without waiting for that query.
func invalidateReplicaCaches(source *table) {
	queue := []*table{source}
	seen := make(map[*table]bool)
	for len(queue) > 0 {
		target := queue[0]
		queue = queue[1:]
		if seen[target] {
			continue
		}
		seen[target] = true
		target.replicationCacheEpoch.Add(1)
		target.cacheGeneration.Add(1)
		target.beginContributionMutation()
		target.endContributionMutation()
		target.mu.Lock()
		for _, trigger := range target.Triggers {
			if trigger.cacheTarget != nil {
				queue = append(queue, trigger.cacheTarget)
			}
		}
		target.mu.Unlock()
	}
}

func (r *replicationManager) localRecord(resource replicaResource, phase mosiPhase, generation string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.journal.Records[resource.key()]
	record.Resource, record.State = resource, phase.String()
	if generation != "" {
		record.Generation = generation
	}
	r.journal.Records[resource.key()] = record
	return r.saveLocked()
}

func (r *replicationManager) declareShard(shard *storageShard, owner string, local bool) *replicaCell {
	resource := replicaResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.logicalID().String()}
	c := r.resourceCell(resource)
	c.current.CompareAndSwap(nil, shard)
	shard.coherence.CompareAndSwap(nil, c)
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.journal.Records[resource.key()]; !exists {
		phase := mosiIO
		if local {
			phase = mosiM
		}
		generation := ""
		if local {
			generation = shard.uuid.String()
		}
		r.journal.Records[resource.key()] = replicaRecord{Resource: resource, State: phase.String(), Owner: owner, Epoch: 1, Generation: generation}
		if err := r.saveLocked(); err != nil {
			panic(err)
		}
		c.rights.initialize(phase)
	}
	return c
}

func (r *replicationManager) schemaShards(db *database, owner string, local bool) {
	for _, table := range db.tables.GetAll() {
		if table.isEphemeralQueryTable() {
			continue
		}
		for _, shard := range table.ActiveShards() {
			if shard != nil {
				r.declareShard(shard, owner, local)
			}
		}
	}
}

func (r *replicationManager) findShard(resource replicaResource) *storageShard {
	db := databases.Get(resource.Database)
	if db == nil {
		return nil
	}
	db.ensureLoaded()
	table := db.tables.Get(resource.Table)
	if table == nil {
		return nil
	}
	for _, shard := range table.ActiveShards() {
		if shard != nil && shard.logicalID().String() == resource.Shard {
			return shard
		}
	}
	return nil
}

func (r *replicationManager) shardCell(shard *storageShard) *replicaCell {
	if c := shard.coherence.Load(); c != nil {
		return c
	}
	resource := replicaResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.logicalID().String()}
	c := r.resourceCell(resource)
	c.current.CompareAndSwap(nil, shard)
	shard.coherence.CompareAndSwap(nil, c)
	return c
}

// prepareResource rebuilds exactly one logical shard, never its siblings. Its
// write handover gate is already owned and transactions have completed. The
// existing rebuild forwards mutations and retains old reader generations.
func (r *replicationManager) prepareResource(resource replicaResource) (generation string, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication prepare: %v", failure)
		}
	}()
	db := databases.Get(resource.Database)
	if db == nil {
		return "", fmt.Errorf("database does not exist")
	}
	db.ensureLoaded()
	if resource.Shard == "" {
		r.schemaShards(db, r.id, true)
		db.save()
		return "", nil
	}
	c := r.resourceCell(resource)
	shard := c.current.Load()
	if shard == nil {
		shard = r.findShard(resource)
	}
	if shard == nil {
		return "", fmt.Errorf("shard does not exist")
	}
	table := shard.t
	lease := GlobalMaintenanceRAMBudget.Acquire(estimateTableMaintenanceBytes(table))
	defer lease.Release()
	table.maintenanceMu.Lock()
	defer table.maintenanceMu.Unlock()
	shard.ensureLoaded() // replay a recovered WAL before testing delta-free state
	force := false
	for _, column := range shard.t.columnDeclarations() {
		if !column.Computor.IsNil() {
			continue
		}
		reader := db.persistence.ReadColumn(shard.uuid.String(), column.Name)
		if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
			force = true
		}
		reader.Close()
	}
	rebuilt := shard.rebuild(force)
	if rebuilt != shard {
		if err := r.replaceShard(resource, shard, rebuilt); err != nil {
			return "", err
		}
	}
	if table.PersistencyMode != Memory && table.PersistencyMode != Cache && table.Name != ".blobs" {
		if _, valid := readBlobManifest(db.persistence.ReadColumn(rebuilt.uuid.String(), blobManifestColumn)); !valid {
			if _, valid := backfillBlobManifest(rebuilt, true); !valid {
				return "", fmt.Errorf("invalid shard blob manifest")
			}
		}
	}
	c.current.Store(rebuilt)
	return rebuilt.uuid.String(), nil
}

// replaceShard publishes a point update to one table's immutable topology.
// Old pointers and files remain owned by their existing snapshots; replication
// does not mutate payload or introduce a second lock for that old generation.
func (r *replicationManager) replaceShard(resource replicaResource, previous, replacement *storageShard) error {
	db := databases.Get(resource.Database)
	if db == nil {
		return fmt.Errorf("missing database for shard patch")
	}
	table := db.tables.Get(resource.Table)
	if table == nil {
		return fmt.Errorf("missing table for shard patch")
	}
	cell := r.resourceCell(resource)
	if previous != replacement {
		cell.retainGeneration(previous)
	}
	table.mu.Lock()
	replace := func(shards []*storageShard) []*storageShard {
		result := append([]*storageShard(nil), shards...)
		for i, shard := range result {
			if shard == previous || (shard != nil && shard.logicalID().String() == resource.Shard) {
				result[i] = replacement
			}
		}
		return result
	}
	table.Shards, table.PShards = replace(table.Shards), replace(table.PShards)
	table.publishTopologyLocked()
	table.mu.Unlock()
	db.save()
	return nil
}

func (r *replicationManager) sendResource(peer *replicationPeer, header replicaPacket) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication transfer: %v", failure)
		}
	}()
	db := databases.Get(header.Resource.Database)
	if db == nil {
		return fmt.Errorf("missing database")
	}
	header.Kind = "begin"
	if err := peer.post(header); err != nil {
		return err
	}
	if header.Resource.Shard == "" {
		if err := peer.sendObject(header, "schema", "", io.NopCloser(bytes.NewReader(db.persistence.ReadSchema()))); err != nil {
			return err
		}
	} else {
		shard := r.resourceCell(header.Resource).current.Load()
		if shard == nil {
			return fmt.Errorf("missing shard generation")
		}
		header.Generation = shard.uuid.String()
		volatile := shard.t.PersistencyMode == Memory || shard.t.PersistencyMode == Cache
		for _, column := range shard.t.columnDeclarations() {
			if column.Computor.IsNil() {
				var reader io.ReadCloser
				if volatile {
					reader = replicaColumnStream(shard.getColumnStorageOrPanic(column.Name, false, nil))
				} else {
					reader = db.persistence.ReadColumn(header.Generation, column.Name)
				}
				if err := peer.sendObject(header, "column", column.Name, reader); err != nil {
					return err
				}
			}
		}
		manifest := db.persistence.ReadColumn(header.Generation, blobManifestColumn)
		if missing, ok := manifest.(interface{ Missing() bool }); !ok || !missing.Missing() {
			if err := peer.sendObject(header, "column", blobManifestColumn, manifest); err != nil {
				return err
			}
		} else {
			manifest.Close()
		}
		references := make(map[string]struct{})
		if volatile {
			for _, column := range shard.t.columnDeclarations() {
				if column.Computor.IsNil() {
					appendColumnBlobReferences(shard.getColumnStorageOrPanic(column.Name, false, nil), references, shard.main_count)
				}
			}
		} else if shard.t.Name != ".blobs" {
			var valid bool
			references, valid = readBlobManifest(db.persistence.ReadColumn(header.Generation, blobManifestColumn))
			if !valid {
				return fmt.Errorf("invalid shard blob manifest")
			}
		}
		for hash := range references {
			if err := peer.sendObject(header, "blob", hash, db.persistence.ReadBlob(hash)); err != nil {
				return err
			}
		}
	}
	header.Kind = "end"
	return peer.post(header)
}

// Volatile table modes have no committed column file. Serialize an immutable
// rebuilt generation through a pipe, retaining the same bounded packet path.
func replicaColumnStream(column ColumnStorage) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				writer.CloseWithError(fmt.Errorf("column serialization: %v", failure))
			} else {
				writer.Close()
			}
		}()
		column.Serialize(writer)
	}()
	return reader
}

func (r *replicationManager) startImport(packet replicaPacket) (*replicaImport, error) {
	db := databases.Get(packet.Resource.Database)
	var engine PersistenceEngine
	if db != nil {
		engine = db.persistence
	} else {
		engine = createPersistenceFromConfig(packet.Resource.Database, databaseBackendConfig(packet.Resource.Database))
	}
	if engine == nil {
		return nil, fmt.Errorf("invalid local backend")
	}
	return &replicaImport{header: packet, engine: engine}, nil
}

func (transfer *replicaImport) abort() {
	if transfer.writer != nil {
		abortPersistenceWrite(transfer.writer)
		transfer.writer = nil
	}
}

func (transfer *replicaImport) packet(packet replicaPacket) error {
	if packet.Resource != transfer.header.Resource || packet.Epoch != transfer.header.Epoch || packet.Generation != transfer.header.Generation {
		return fmt.Errorf("transfer header changed")
	}
	switch packet.Kind {
	case "object":
		if transfer.writer != nil || transfer.objectKind != "" {
			return fmt.Errorf("unfinished object")
		}
		transfer.objectKind, transfer.objectName = packet.ObjectKind, packet.Object
		switch packet.ObjectKind {
		case "schema":
			if packet.Resource.Shard != "" {
				return fmt.Errorf("schema in shard transfer")
			}
		case "column":
			if packet.Resource.Shard == "" {
				return fmt.Errorf("column in schema transfer")
			}
			db := databases.Get(packet.Resource.Database)
			if db == nil {
				return fmt.Errorf("unknown column database")
			}
			table := db.tables.Get(packet.Resource.Table)
			if table == nil {
				return fmt.Errorf("unknown column table")
			}
			if packet.Object != blobManifestColumn {
				found := false
				for _, column := range table.columnDeclarations() {
					if column.Name == packet.Object && column.Computor.IsNil() {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("unknown transferred column")
				}
			}
			if _, err := uuid.Parse(packet.Generation); err != nil {
				return err
			}
			transfer.writer = transfer.engine.WriteColumn(packet.Generation, packet.Object)
		case "blob":
			if !validBlobHash(packet.Object) {
				return fmt.Errorf("invalid blob hash")
			}
			transfer.writer = transfer.engine.WriteBlob(packet.Object)
		default:
			return fmt.Errorf("invalid object kind")
		}
	case "chunk":
		if packet.ObjectKind != transfer.objectKind || packet.Object != transfer.objectName {
			return fmt.Errorf("object header changed")
		}
		data, err := decodeReplicaChunk(packet)
		if err != nil {
			return err
		}
		if transfer.objectKind == "schema" {
			_, err = transfer.schema.Write(data)
		} else if transfer.writer != nil {
			_, err = transfer.writer.Write(data)
		} else {
			return fmt.Errorf("chunk outside object")
		}
		return err
	case "object_end":
		if packet.ObjectKind != transfer.objectKind || packet.Object != transfer.objectName {
			return fmt.Errorf("object header changed")
		}
		if transfer.objectKind == "" {
			return fmt.Errorf("object end without object")
		}
		if transfer.writer != nil {
			finishColumnWrite(transfer.writer, true)
			transfer.writer = nil
		}
		transfer.objectKind = ""
	default:
		return fmt.Errorf("invalid transfer packet")
	}
	return nil
}

func validBlobHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}

func (r *replicationManager) finishImport(source string, transfer *replicaImport) error {
	if transfer.objectKind != "" {
		return fmt.Errorf("incomplete resource transfer")
	}
	resource := transfer.header.Resource
	if resource.Shard == "" {
		return r.installSchema(source, resource.Database, transfer.engine, transfer.schema.Bytes())
	}
	db := databases.Get(resource.Database)
	if db == nil {
		return fmt.Errorf("schema must arrive before shard")
	}
	table := db.tables.Get(resource.Table)
	if table == nil {
		return fmt.Errorf("unknown shard table")
	}
	shard := new(storageShard)
	encoded, _ := json.Marshal(struct {
		UUID    string `json:"uuid"`
		Lineage string `json:"mosi"`
	}{transfer.header.Generation, resource.Shard})
	if err := json.Unmarshal(encoded, shard); err != nil {
		return err
	}
	shard.t = table
	c := r.resourceCell(resource)
	if current := c.current.Load(); current != nil && current.uuid == shard.uuid && current.t == table && c.rights.phase().readable() {
		return nil
	}
	if table.PersistencyMode == Memory || table.PersistencyMode == Cache {
		// Ordinary volatile loaders deliberately start empty. A received
		// generation instead instantiates its staged column streams privately,
		// before publication, without changing backend or table-mode semantics.
		for _, column := range table.columnDeclarations() {
			if !column.Computor.IsNil() {
				shard.columns[column.Name] = nil
				continue
			}
			reader := transfer.engine.ReadColumn(shard.uuid.String(), column.Name)
			var magic uint8
			if err := binary.Read(reader, binary.LittleEndian, &magic); err != nil {
				reader.Close()
				return err
			}
			typ, valid := storages[magic]
			if !valid {
				reader.Close()
				return fmt.Errorf("invalid column storage type")
			}
			storage := reflect.New(typ).Interface().(ColumnStorage)
			count, valid := deserializeReferenceStorage(storage, reader)
			reader.Close()
			if !valid {
				return fmt.Errorf("invalid column stream")
			}
			if count > shard.main_count {
				shard.main_count = count
			}
			shard.columns[column.Name] = shard.attachColumnRuntime(column.Name, storage)
		}
		shard.plannerMainRows.Store(shard.main_count)
		shard.setState(WRITE)
	}
	shard.coherence.Store(c)
	previous := c.current.Load()
	if err := r.replaceShard(resource, previous, shard); err != nil {
		return err
	}
	c.current.Store(shard)
	return nil
}

func (r *replicationManager) installSchema(source, name string, engine PersistenceEngine, data []byte) error {
	replacement := newDatabase()
	if err := json.Unmarshal(data, replacement); err != nil {
		return err
	}
	if replacement.Name != name {
		return fmt.Errorf("schema identity mismatch")
	}
	// Construct metadata privately through the existing loader; no data shards
	// are read at this boundary. Existing unchanged tables stay in their live
	// database, preserving ongoing DML and runtime generation ownership.
	db := databases.Get(name)
	if db != nil {
		db.ensureLoaded()
		db.persistenceLifecycle.Lock()
		defer db.persistenceLifecycle.Unlock()
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		// A queued local checkpoint must finish before the incoming schema is
		// installed. Schema ownership prevents another old snapshot registering.
		db.saveMu.Lock()
		for db.saveInFlight {
			db.getSaveCond().Wait()
		}
		db.saveMu.Unlock()
	}
	engine.WriteSchema(data)
	replacement = newDatabase()
	replacement.Name, replacement.persistence, replacement.srState = name, engine, COLD
	replacement.ensureLoaded()
	if db == nil {
		databases.Set(name, replacement)
		db = replacement
	} else {
		defer replacement.closeTransactionLog()
		for _, table := range db.tables.GetAll() {
			if replacement.tables.Get(table.Name) == nil {
				for _, shard := range table.ActiveShards() {
					if shard == nil {
						continue
					}
					cell := r.shardCell(shard)
					cell.retainGeneration(shard)
					cell.current.CompareAndSwap(shard, nil)
				}
				db.tables.Remove(table.Name)
			}
		}
		for _, table := range replacement.tables.GetAll() {
			previous := db.tables.Get(table.Name)
			if previous != nil && sameReplicaDeclaration(previous, table) {
				continue
			}
			table.schema = db
			for _, shard := range table.ActiveShards() {
				if shard == nil {
					continue
				}
				cell := r.shardCell(shard)
				if old := cell.current.Load(); old != shard {
					cell.retainGeneration(old)
					cell.current.Store(shard)
				}
			}
			db.tables.Set(table.Name, table)
			if table.Name == ".blobs" {
				db.blobRefState().table.Store(table)
			}
		}
		for _, definition := range db.procedures.GetAll() {
			if replacement.procedures.Get(definition.Name) == nil {
				db.procedures.Remove(definition.Name)
			}
		}
		for _, definition := range replacement.procedures.GetAll() {
			db.procedures.Set(definition.Name, definition)
		}
		db.publishTableNamesLocked()
	}
	r.schemaShards(db, source, false)
	return nil
}

// Runtime feedback and physical rebuild UUIDs are independent of declaration
// identity. Receiving a metadata checkpoint must not replace an unchanged live
// table and detach its transaction, cache or generation owners.
func sameReplicaDeclaration(previous, incoming *table) bool {
	declaration := func(table *table) []byte {
		encoded, _ := json.Marshal(table)
		var fields map[string]json.RawMessage
		if json.Unmarshal(encoded, &fields) != nil {
			return encoded
		}
		for _, key := range []string{"Auto_increment", "planner_row_estimate", "key_frequencies", "filter_feedback"} {
			delete(fields, key)
		}
		for _, key := range []string{"Shards", "PShards"} {
			var shards []*storageShard
			if json.Unmarshal(fields[key], &shards) == nil {
				identities := make([]string, len(shards))
				for i, shard := range shards {
					if shard != nil {
						identities[i] = shard.logicalID().String()
					}
				}
				fields[key], _ = json.Marshal(identities)
			}
		}
		var columns []map[string]json.RawMessage
		if json.Unmarshal(fields["Columns"], &columns) == nil {
			for _, column := range columns {
				delete(column, "PartitioningScore")
			}
			fields["Columns"], _ = json.Marshal(columns)
		}
		result, _ := json.Marshal(fields)
		return result
	}
	return bytes.Equal(declaration(previous), declaration(incoming))
}

func (r *replicationManager) step(ctx context.Context, peerID string, packet replicaPacket) error {
	peer := r.peer(peerID)
	if peer == nil {
		return fmt.Errorf(replicationNetworkError)
	}
	step := &replicaStep{peer: peerID, resource: packet.Resource, done: make(chan struct{})}
	r.steps.Store(packet.ID, step)
	defer r.steps.Delete(packet.ID)
	if err := peer.post(packet); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf(replicationNetworkError)
	case <-step.done:
		return step.err
	}
}

func (r *replicationManager) transmitResource(ctx context.Context, peerID string, header replicaPacket) error {
	peer := r.peer(peerID)
	if peer == nil {
		return fmt.Errorf(replicationNetworkError)
	}
	step := &replicaStep{peer: peerID, resource: header.Resource, done: make(chan struct{})}
	r.steps.Store(header.ID, step)
	defer r.steps.Delete(header.ID)
	if err := r.sendResource(peer, header); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf(replicationNetworkError)
	case <-step.done:
		return step.err
	}
}

// A completed backup can advance readers of the same immutable data. The
// resource directory serializes this point publication with ownership changes;
// receivers install the shard and topology before acknowledging it.
func (r *replicationManager) distributeRebuild(resource replicaResource) {
	cell := r.resourceCell(resource)
	cell.directory.Lock()
	defer cell.directory.Unlock()
	r.mu.Lock()
	record := r.journal.Records[resource.key()]
	readers := make([]string, 0, len(record.Readers))
	for id := range record.Readers {
		if id != r.id && id != record.Owner {
			readers = append(readers, id)
		}
	}
	r.mu.Unlock()
	for _, id := range readers {
		peer := r.peer(id)
		if peer == nil {
			continue
		}
		ctx, cancel := replicaPacketContext(peer)
		_ = r.transmitResource(ctx, id, replicaPacket{ID: uuid.NewString(), Resource: resource, Epoch: record.Epoch, Generation: record.Generation})
		cancel()
	}
}

// Called only after the connection reader and its handlers have stopped.
func (r *replicationManager) abortImports(source string) {
	r.imports.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), source+"/") {
			value.(*replicaImport).abort()
			r.imports.Delete(key)
		}
		return true
	})
}

// The journal's generation names a completed backup, not a UUID merely seen in
// a remote schema. Maintenance retains it until another complete transfer wins.
func replicationBackupRecords(db *database) []replicaRecord {
	r := replication.Load()
	if r == nil || !r.master {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var records []replicaRecord
	for _, record := range r.journal.Records {
		if record.Resource.Database == db.Name && record.Resource.Shard != "" && record.Owner != r.id && record.Generation != "" {
			if table := db.tables.Get(record.Resource.Table); table != nil && table.PersistencyMode != Memory && table.PersistencyMode != Cache {
				records = append(records, record)
			}
		}
	}
	return records
}

func (r *replicationManager) restoreBackup(record replicaRecord) error {
	if record.Resource.Shard == "" {
		return nil
	}
	db := databases.Get(record.Resource.Database)
	if db == nil {
		return fmt.Errorf("missing backup database")
	}
	table := db.tables.Get(record.Resource.Table)
	if table == nil {
		return nil
	} // resource was dropped through its schema
	if record.Generation == "" {
		return fmt.Errorf("slave has no completed shard backup")
	}
	transfer := &replicaImport{header: replicaPacket{Resource: record.Resource, Generation: record.Generation}, engine: db.persistence}
	return r.finishImport(r.id, transfer)
}
