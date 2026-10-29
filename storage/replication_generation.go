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

import "sync/atomic"

import "io"
import "fmt"
import "sync"
import "bytes"
import "strings"
import "reflect"
import "runtime"
import "context"
import "crypto/sha256"
import "encoding/json"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/storage/mosi"

// The receiver owns one private backend writer per active transfer. Packet
// memory is bounded by the connection queue; schema JSON uses the existing
// byte-slice persistence API, while column/blob bytes go directly to writers.
type replicaImport struct {
	header mosiPacket
	engine PersistenceEngine
	// Set at unpublished construction from completed local data placement.
	// Duplicate immutable columns are consumed without reopening their files.
	knownContent bool
	superseded   bool
	writer       io.WriteCloser
	schema       bytes.Buffer
	routes       bytes.Buffer
	objectKind   string
	objectName   string
}

type replicaStep struct {
	peer     string
	resource mosiResource
	epoch    uint64
	content  string
	done     chan struct{}
	once     sync.Once
	err      error
}

// Source preparation follows native lifecycle -> MOSI handover ownership.
// The caller releases this capability before sending packets; the idempotent
// local release also supports deferred cleanup of preparation failures.
func replicaPersistenceRead(resource mosiResource) (func(), error) {
	if resource.Shard == "" {
		return func() {}, nil
	}
	db := databases.Get(resource.Database)
	if db == nil {
		return nil, fmt.Errorf("missing shard database")
	}
	db.persistenceLifecycle.RLock()
	released := false
	return func() {
		if !released {
			released = true
			db.persistenceLifecycle.RUnlock()
		}
	}, nil
}

// A request ID names one operation, not just a peer/resource pair. An old
// epoch or a different installed content cannot complete its acknowledgement.
func (step *replicaStep) matches(source string, packet mosiPacket) bool {
	return step.peer == source && step.resource == packet.Resource && step.epoch == packet.Epoch && (step.content == "" || step.content == packet.Content)
}

func (step *replicaStep) finish(err error) { step.once.Do(func() { step.err = err; close(step.done) }) }

// publishLocal changes only execution permission. In particular, handing M to
// an attached slave leaves the master's outward clusterstate unchanged. The
// wire adapter must explicitly publish an outer revocation AFTER this call.
func (r *replicationManager) publishLocal(resource mosiResource, phase mosi.Phase) {
	c := r.resourceCell(resource)
	if phase == mosi.IO && resource.Shard != "" {
		// Cached analytic results must stop being reusable before a remote
		// writer changes this source. This is publication work, not a scan/read
		// counter, and causes ordinary cache initialization to acquire GetRead.
		if shard := c.current.Load(); shard != nil {
			invalidateDependentCaches(shard.t)
		}
	}
	c.ownstate.Publish(phase)
}

func (r *replicationManager) localRecord(resource mosiResource, phase mosi.Phase, content string) {
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Resource, record.State = resource, phase.String()
		if content != "" {
			record.Content = content
		}
		return true
	})
}

// establishAuthority publishes an unpublished resource from the existing
// declaration/recovery proof. record.State is the LOCAL permission, not the
// aggregate rights of the master's domain. A known remote owner still belongs
// to that root domain; an unknown backup owner establishes no outward rights.
//
// This root/leaf mapping is the current replication adapter only. Cluster join
// must supply real outward grants instead of deriving M from the master role.
// Recursive groups apply this same pair at each boundary; peer routing and
// directory ownership belong to their adapters, never to query admission.
func (r *replicationManager) establishAuthority(c *replicaCell, record replicaRecord) {
	own := mosi.ParsePhase(record.State)
	outer := own
	if r.master {
		outer = mosi.IO
		if record.Owner != "" {
			outer = mosi.M
		}
	}
	c.clusterstate.Establish(outer)
	c.ownstate.Establish(own)
}

func (r *replicationManager) declareShard(shard *storageShard, owner string, local bool) *replicaCell {
	r.restoreTableRoutes(shard.t)
	if r.cluster != nil {
		return r.cluster.declareShard(shard, local)
	}
	resource := r.shardResource(shard)
	c := r.resourceCell(resource)
	c.current.CompareAndSwap(nil, shard)
	shard.coherence.CompareAndSwap(nil, c)
	initialized := false
	record := r.updateRecord(resource, func(record *replicaRecord) bool {
		// Schema loading may retain predecessor files while data is in flight.
		// Update its table name without reverting an already forwarded UUID.
		publication := *c.resource.Load()
		publication.Table = resource.Table
		c.resource.Store(&publication)
		if record.Owner != "" {
			changed := record.Resource != resource
			record.Resource = resource
			if r.master && !r.restoring[resource.Database] {
				if r.confirmed == nil {
					r.confirmed = make(map[string]string)
				}
				r.confirmed[r.resourceKey(resource)] = record.Owner
			}
			return changed
		}
		record.Resource, record.Owner, record.Epoch = resource, owner, 1
		record.State = "IO"
		if local {
			record.State, record.Content = "M", shard.uuid.String()
		}
		if r.master {
			schemaKey := mosiResource{Database: resource.Database}.key()
			known := !r.restoring[resource.Database] || r.confirmed[schemaKey] != "" && (owner == r.id || r.peers[owner].recovered()) && len(r.recoveryReports[resource.key()]) == 0
			if !known {
				record.Owner, record.State = "", "IO"
			} else {
				if r.confirmed == nil {
					r.confirmed = make(map[string]string)
				}
				r.confirmed[r.resourceKey(resource)] = record.Owner
			}
		}
		initialized = true
		return true
	})
	if initialized {
		r.establishAuthority(c, record)
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

func (r *replicationManager) findShard(resource mosiResource) *storageShard {
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
		if shard != nil && r.resourceKey(mosiResource{Database: resource.Database, Table: resource.Table, Shard: shard.uuid.String()}) == r.resourceKey(resource) {
			return shard
		}
	}
	return nil
}

func (r *replicationManager) shardCell(shard *storageShard) *replicaCell {
	if c := shard.coherence.Load(); c != nil {
		return c.resolved()
	}
	r.restoreTableRoutes(shard.t)
	resource := mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()}
	c := r.resourceCell(resource)
	c.current.CompareAndSwap(nil, shard)
	shard.coherence.CompareAndSwap(nil, c)
	return c
}

// prepareResource rebuilds exactly one logical shard, never its siblings. Its
// sealed MOSI handover is already owned and native transactions have drained. The
// existing rebuild forwards mutations and retains old reader generations.
func (r *replicationManager) prepareResource(resource mosiResource) (content string, err error) {
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
		if r.cluster == nil {
			r.schemaShards(db, r.id, true)
		}
		db.save()
		return r.schemaDigest(db.persistence.ReadSchema())
	}
	c := r.resourceCell(resource)
	shard := c.publishedShard()
	if shard == nil {
		shard = r.findShard(resource)
	}
	if shard == nil {
		return "", fmt.Errorf("shard does not exist")
	}
	table := shard.t
	if r.cluster != nil {
		c.current.CompareAndSwap(nil, shard)
		shard.coherence.CompareAndSwap(nil, c)
	}
	if r.cluster != nil && shard.state() == COLD {
		// Backend-only authority is current by invariant. There are no local
		// deltas to rebuild: transfer its reference and native commit authority.
		// In particular, a coordinator must not load a shard it is delegating.
		if table.PersistencyMode == Memory || table.PersistencyMode == Cache {
			return "", fmt.Errorf("volatile data needs a separate replication store")
		}
		return shard.uuid.String(), nil
	}
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
// does not mutate payload or introduce a second lock for that old content.
func (r *replicationManager) replaceShard(resource mosiResource, previous, replacement *storageShard) error {
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
		cell.retainShard(previous)
		r.forwardResource(resource, mosiResource{Database: resource.Database, Table: resource.Table, Shard: replacement.uuid.String()})
		cell = cell.resolved()
		replacement.coherence.Store(cell)
	}
	table.mu.Lock()
	replace := func(shards []*storageShard) []*storageShard {
		result := append([]*storageShard(nil), shards...)
		for i, shard := range result {
			if shard == previous || (shard != nil && r.resourceCell(mosiResource{Database: resource.Database, Table: resource.Table, Shard: shard.uuid.String()}) == cell) {
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

// replicaTransfer binds one immutable content before releasing source write
// ownership. The active outgoing registry protects its backend objects from
// reclamation; no writer lock is retained by the network.
type replicaTransfer struct {
	routes    []shardForward
	engine    PersistenceEngine
	schema    []byte
	shard     *storageShard
	columns   []*column
	content   string
	volatile  bool
	blobTable bool
	payload   ReadAccess // existing eviction lease, only for volatile streamed RAM
	manager   *replicationManager
	database  string
	released  bool // exclusively owned by the sending caller
}

// Caller owns the database persistence lifecycle while binding the content.
func (r *replicationManager) captureResource(resource mosiResource, content string) (*replicaTransfer, error) {
	db := databases.Get(resource.Database)
	if db == nil {
		return nil, fmt.Errorf("missing database")
	}
	transfer := &replicaTransfer{engine: db.persistence, content: content, routes: r.shardRoutes(resource)}
	complete := false
	defer func() {
		if !complete {
			transfer.release()
		}
	}()
	if resource.Shard == "" {
		transfer.schema = append([]byte(nil), db.persistence.ReadSchema()...)
		actual, err := r.schemaDigest(transfer.schema)
		if err != nil {
			return nil, err
		}
		if content != "" && actual != content {
			return nil, fmt.Errorf("schema content changed before transfer")
		}
		// A durable checkpoint can precede native active-topology publication
		// (overflow rebuild). Publish its exact UUID edges before a peer can
		// acquire one of the successor UUIDs advertised in this schema.
		if err := r.restoreSchemaRoutes(transfer.schema); err != nil {
			return nil, err
		}
		transfer.routes = r.shardRoutes(resource)
		transfer.content = actual
		complete = true
		return transfer, nil
	}
	transfer.manager, transfer.database = r, resource.Database
	r.transfers.Store(transfer, resource.Database)
	cell := r.resourceCell(resource)
	shard := cell.current.Load()
	if shard != nil && content != "" && shard.uuid.String() != content {
		shard = nil
		if history := cell.retiredShards.Load(); history != nil {
			for _, entry := range *history {
				if previous := entry.Value(); previous != nil && previous.uuid.String() == content {
					shard = previous
					break
				}
			}
		}
	}
	if shard == nil {
		return nil, fmt.Errorf("missing transfer content")
	}
	transfer.shard, transfer.content = shard, shard.uuid.String()
	transfer.columns = shard.t.columnDeclarations()
	transfer.volatile = shard.t.PersistencyMode == Memory || shard.t.PersistencyMode == Cache
	transfer.blobTable = shard.t.Name == ".blobs"
	if transfer.volatile {
		transfer.payload = shard.getRead(nil, true, false)
	}
	complete = true
	return transfer, nil
}

// The sender is the sole release owner. GC sees only the registry entry;
// ordinary queries and local write admission never inspect or update it.
func (transfer *replicaTransfer) release() {
	if transfer.released {
		return
	}
	transfer.released = true
	transfer.payload.Release()
	if transfer.manager != nil {
		transfer.manager.transfers.Delete(transfer)
	}
}

func (r *replicationManager) hasTransfers(database string) bool {
	found := false
	r.transfers.Range(func(_, value any) bool { found = value.(string) == database; return !found })
	return found
}

func (r *replicationManager) sendResource(peer *mosiPeer, header mosiPacket, transfer *replicaTransfer) (err error) {
	defer runtime.KeepAlive(transfer)
	defer transfer.release()

	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("replication transfer: %v", failure)
		}
	}()
	header.Content = transfer.content
	if header.Resource.Shard != "" {
		header.Forwards = transfer.routes
	}
	header.Kind = "begin"
	if err := peer.post(header); err != nil {
		return err
	}
	header.Forwards = nil // routes are introduced once, before streaming payload
	if header.Resource.Shard == "" {
		if len(transfer.routes) != 0 {
			routes, err := json.Marshal(transfer.routes)
			if err != nil {
				return err
			}
			if err := peer.sendObject(header, "routes", "", io.NopCloser(bytes.NewReader(routes))); err != nil {
				return err
			}
		}
		if err := peer.sendObject(header, "schema", "", io.NopCloser(bytes.NewReader(transfer.schema))); err != nil {
			return err
		}
	} else {
		shard := transfer.shard
		volatile := transfer.volatile
		for _, column := range transfer.columns {
			if column.Computor.IsNil() {
				var reader io.ReadCloser
				if volatile {
					reader = replicaColumnStream(shard.getColumnStorageOrPanic(column.Name, false, nil))
				} else {
					reader = transfer.engine.ReadColumn(header.Content, column.Name)
				}
				if err := peer.sendObject(header, "column", column.Name, reader); err != nil {
					return err
				}
			}
		}
		manifest := transfer.engine.ReadColumn(header.Content, blobManifestColumn)
		if missing, ok := manifest.(interface{ Missing() bool }); !ok || !missing.Missing() {
			if err := peer.sendObject(header, "column", blobManifestColumn, manifest); err != nil {
				return err
			}
		} else {
			manifest.Close()
		}
		references := make(map[string]struct{})
		if volatile {
			for _, column := range transfer.columns {
				if column.Computor.IsNil() {
					appendColumnBlobReferences(shard.getColumnStorageOrPanic(column.Name, false, nil), references, shard.main_count)
				}
			}
		} else if !transfer.blobTable {
			var valid bool
			references, valid = readBlobManifest(transfer.engine.ReadColumn(header.Content, blobManifestColumn))
			if !valid {
				return fmt.Errorf("invalid shard blob manifest")
			}
		}
		for hash := range references {
			if err := peer.sendObject(header, "blob", hash, transfer.engine.ReadBlob(hash)); err != nil {
				return err
			}
		}
	}
	header.Kind = "end"
	return peer.post(header)
}

// Volatile table modes have no committed column file. Serialize an immutable
// rebuilt content through a pipe, retaining the same bounded packet path.
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

func (r *replicationManager) startImport(packet mosiPacket) (*replicaImport, error) {
	db := databases.Get(packet.Resource.Database)
	var engine PersistenceEngine
	if db != nil {
		if packet.Resource.Shard != "" {
			// Complete lazy catalog publication before registering the import or
			// acquiring importPublication at completion. Startup GC may own the
			// same loadOnce; a discovered database is not yet a loaded schema.
			db.ensureLoaded()
		}
		engine = db.persistence
	} else {
		engine = createPersistenceFromConfig(packet.Resource.Database, databaseBackendConfig(packet.Resource.Database))
	}
	if engine == nil {
		return nil, fmt.Errorf("invalid local backend")
	}
	r.mu.Lock()
	known := packet.Resource.Shard != "" && packet.Content != "" && (r.directory.Records[r.resourceKey(packet.Resource)].Content == packet.Content || r.backups[r.resourceKey(packet.Resource)].Content == packet.Content)
	r.mu.Unlock()
	if cell := r.lookupCell(packet.Resource); !known && packet.Resource.Shard != "" && cell != nil {
		if history := cell.retiredShards.Load(); history != nil {
			for _, entry := range *history {
				if shard := entry.Value(); shard != nil && shard.uuid.String() == packet.Content {
					known = true
					break
				}
			}
		}
	}
	return &replicaImport{header: packet, engine: engine, knownContent: known}, nil
}

func (transfer *replicaImport) abort() {
	if transfer.writer != nil {
		abortPersistenceWrite(transfer.writer)
		transfer.writer = nil
	}
}

func (transfer *replicaImport) packet(packet mosiPacket) error {
	if packet.Resource != transfer.header.Resource || packet.Epoch != transfer.header.Epoch || packet.Content != transfer.header.Content {
		return fmt.Errorf("transfer header changed")
	}
	switch packet.Kind {
	case "object":
		if transfer.writer != nil || transfer.objectKind != "" {
			return fmt.Errorf("unfinished object")
		}
		transfer.objectKind, transfer.objectName = packet.ObjectKind, packet.Object
		switch packet.ObjectKind {
		case "schema", "routes":
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
			db.ensureLoaded()
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
			if _, err := uuid.Parse(packet.Content); err != nil {
				return err
			}
			if transfer.knownContent {
				// WriteColumn may truncate immediately: backends expect a private
				// new content. Reconnect backups can resend an already completed
				// UUID, whose immutable payload must survive even an aborted resend.
				// A volatile content may have no local column files yet; those
				// still need their first installation through the normal writer.
				exists, err := func() (bool, error) {
					existing := transfer.engine.ReadColumn(packet.Content, packet.Object)
					defer existing.Close()
					if missing, ok := existing.(interface{ Missing() bool }); ok && missing.Missing() {
						return false, nil
					}
					var marker [1]byte
					n, err := existing.Read(marker[:])
					if err != nil && err != io.EOF {
						return false, err
					}
					return n != 0, nil
				}()
				if err != nil {
					return err
				}
				if exists {
					return nil
				}
			}
			transfer.writer = transfer.engine.WriteColumn(packet.Content, packet.Object)
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
		data, err := decodeMosiChunk(packet)
		if err != nil {
			return err
		}
		if transfer.objectKind == "routes" {
			_, err = transfer.routes.Write(data)
		} else if transfer.objectKind == "schema" {
			_, err = transfer.schema.Write(data)
		} else if transfer.writer != nil {
			_, err = transfer.writer.Write(data)
		} else if !transfer.knownContent || transfer.objectKind != "column" {
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
		var routes []shardForward
		if transfer.routes.Len() != 0 {
			if err := json.Unmarshal(transfer.routes.Bytes(), &routes); err != nil {
				return err
			}
		}
		for _, route := range routes {
			if route.From.Database != resource.Database || route.To.Database != resource.Database {
				return fmt.Errorf("foreign schema forwarding")
			}
			r.forwardResource(route.From, route.To)
		}
		return r.installSchema(source, resource.Database, transfer.engine, transfer.schema.Bytes())
	}
	actual := resource
	actual.Shard = transfer.header.Content
	if source != r.id && r.canonicalResource(actual).Shard != actual.Shard {
		// A newer successor was already published while these bytes were in
		// flight. ACK the captured transfer without reverting current data.
		transfer.superseded = true
		return nil
	}
	db := databases.Get(resource.Database)
	if db == nil {
		return fmt.Errorf("schema must arrive before shard")
	}
	// Network transfers initialized this catalog before import registration;
	// local backup restoration also enters here without startImport.
	db.ensureLoaded()
	table := db.tables.Get(resource.Table)
	if table == nil {
		return fmt.Errorf("unknown shard table")
	}
	shard := new(storageShard)
	encoded, _ := json.Marshal(transfer.header.Content)
	if err := json.Unmarshal(encoded, shard); err != nil {
		return err
	}
	shard.t = table
	c := r.resourceCell(resource)
	publication := c.ownstate.Snapshot()
	readable := publication.Phase().Readable() || publication.Phase().Requested() && publication.Retained().Readable() || publication.Phase() == mosi.IS && publication.Backend().Readable()
	if current := c.current.Load(); current != nil && current.uuid == shard.uuid && current.t == table && readable {
		return nil
	}
	if table.PersistencyMode == Memory || table.PersistencyMode == Cache {
		// Ordinary volatile loaders deliberately start empty. A received
		// content instead instantiates its staged column streams privately,
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
	// database, preserving ongoing DML and runtime content ownership.
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
	if db != nil && r.cluster == nil {
		var err error
		data, err = r.replicaSchemaForLocalData(db, data)
		if err != nil {
			return err
		}
	}
	if r.cluster == nil {
		engine.WriteSchema(data)
	}
	replacement = newDatabase()
	replacement.Name, replacement.persistence, replacement.srState = name, engine, COLD
	if r.cluster != nil {
		replacement.persistence = clusterSchemaView{engine, data}
	}
	replacement.ensureLoaded()
	replacement.persistence = engine
	if db == nil {
		databases.Set(name, replacement)
		db = replacement
	} else {
		defer replacement.closeTransactionLog()
		for _, table := range db.tables.GetAll() {
			if r.cluster != nil && table.isEphemeralQueryTable() {
				continue
			}
			if replacement.tables.Get(table.Name) == nil {
				for _, shard := range table.ActiveShards() {
					if shard == nil {
						continue
					}
					cell := r.shardCell(shard)
					cell.retainShard(shard)
					cell.current.CompareAndSwap(shard, nil)
				}
				db.tables.Remove(table.Name)
			}
		}
		for _, table := range replacement.tables.GetAll() {
			r.restoreTableRoutes(table)
			previous := db.tables.Get(table.Name)
			if previous != nil && r.cluster != nil {
				merged, err := r.mergeClusterInsertTopology(previous, table)
				if err != nil {
					return err
				}
				if merged {
					continue
				}
			}
			if previous != nil && r.sameReplicaDeclaration(previous, table) {
				previous.reserveLocalAutoIncrement(atomic.LoadUint64(&table.Auto_increment), 0)
				if r.cluster != nil {
					for _, incoming := range table.ActiveShards() {
						if incoming == nil {
							continue
						}
						var old *storageShard
						for _, candidate := range previous.ActiveShards() {
							if candidate != nil && r.resourceKey(r.shardResource(candidate)) == r.resourceKey(r.shardResource(incoming)) {
								old = candidate
								break
							}
						}
						if old == nil {
							continue
						}
						resource := r.shardResource(old)
						cell := r.shardCell(old)
						if old != nil && old.uuid != incoming.uuid {
							if r.canonicalResource(r.shardResource(incoming)).Shard != incoming.uuid.String() {
								// A schema reference captured before the rebuild may
								// still name its predecessor. It cannot replace the
								// successor already installed through data coherence.
								continue
							}
							if cell.ownstate.Phase() == mosi.M {
								return fmt.Errorf("shared schema conflicts with a local modifier")
							}
							incoming.t = previous
							incoming.coherence.Store(cell)
							if err := r.replaceShard(resource, old, incoming); err != nil {
								return err
							}
							cell.current.Store(incoming)
						}
					}
				}
				continue
			}

			table.schema = db
			for _, shard := range table.ActiveShards() {
				if shard == nil {
					continue
				}
				cell := r.shardCell(shard)
				if old := cell.current.Load(); old != shard {
					cell.retainShard(old)
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
// table and detach its transaction, cache or content owners.
func (r *replicationManager) sameReplicaDeclaration(previous, incoming *table) bool {
	return bytes.Equal(r.replicaDeclaration(previous, true), r.replicaDeclaration(incoming, true))
}

func (r *replicationManager) replicaDeclaration(table *table, includeTopology bool) []byte {
	encoded, _ := json.Marshal(table)
	var fields map[string]json.RawMessage
	if json.Unmarshal(encoded, &fields) != nil {
		return encoded
	}
	for _, key := range []string{"Auto_increment", "planner_row_estimate", "key_frequencies", "filter_feedback", "shard_forwards"} {
		delete(fields, key)
	}
	for _, key := range []string{"Shards", "PShards"} {
		if !includeTopology {
			delete(fields, key)
			continue
		}
		var shards []*storageShard
		if json.Unmarshal(fields[key], &shards) == nil {
			identities := make([]string, len(shards))
			for i, shard := range shards {
				if shard != nil {
					identities[i] = r.canonicalResource(mosiResource{Database: table.schema.Name, Table: table.Name, Shard: shard.uuid.String()}).Shard
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

func (r *replicationManager) step(ctx context.Context, peerID string, packet mosiPacket) error {
	peer := r.peer(peerID)
	// Directory recovery may replace a connection after coordination captured
	// its epoch. Never send that operation on an unrecovered replacement.
	if peer == nil || !peer.recovered() {
		return fmt.Errorf("%s: %s step to %s for %s", mosi.NetworkError, packet.Kind, peerID, packet.Resource.key())
	}
	step := &replicaStep{peer: peerID, resource: packet.Resource, epoch: packet.Epoch, content: packet.Content, done: make(chan struct{})}
	r.steps.Store(packet.ID, step)
	defer r.steps.Delete(packet.ID)
	if err := peer.post(packet); err != nil {
		return fmt.Errorf("%s step to %s for %s: %w", packet.Kind, peerID, packet.Resource.key(), err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf("%s: %s step to %s for %s", mosi.NetworkError, packet.Kind, peerID, packet.Resource.key())
	case <-step.done:
		return step.err
	}
}

func (r *replicationManager) transmitResource(ctx context.Context, peerID string, header mosiPacket) error {
	if r.cluster != nil {
		if header.Resource.Shard == "" {
			if err := r.cluster.transferSchemaLocations(ctx, peerID, header); err != nil {
				return err
			}
		} else {
			// Preparation may have published a new physical UUID. Make its
			// current owner routable before the reference wakes the requester;
			// the final post-handover update alone leaves an observable gap.
			r.mu.Lock()
			entry := r.directory.Records[r.resourceKey(header.Resource)]
			r.mu.Unlock()
			if err := r.cluster.registerLocation(ctx, entry); err != nil {
				return err
			}
		}
		header.Kind = "reference"
		if header.Resource.Shard != "" {
			header.Forwards = r.shardRoutes(header.Resource)
		}
		return r.step(ctx, peerID, header)
	}
	peer := r.peer(peerID)
	if peer == nil {
		return fmt.Errorf(mosi.NetworkError)
	}
	release, err := replicaPersistenceRead(header.Resource)
	if err != nil {
		return err
	}
	transfer, err := r.captureResource(header.Resource, header.Content)
	release()
	if err != nil {
		return err
	}
	// Bind the ACK to the captured content even when the caller asked for
	// the current resource without already knowing its physical UUID/hash.
	header.Content = transfer.content
	step := &replicaStep{peer: peerID, resource: header.Resource, epoch: header.Epoch, content: header.Content, done: make(chan struct{})}
	r.steps.Store(header.ID, step)
	defer r.steps.Delete(header.ID)
	if err := r.sendResource(peer, header, transfer); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf(mosi.NetworkError)
	case <-step.done:
		return step.err
	}
}

// A completed backup can advance readers of the same immutable data. The
// resource directory serializes this point publication with ownership changes;
// receivers install the shard and topology before acknowledging it.
func (r *replicationManager) distributeRebuild(resource mosiResource) {
	cell := r.resourceCell(resource)
	cell.directory.Lock()
	defer cell.directory.Unlock()
	r.mu.Lock()
	record := r.directory.Records[r.resourceKey(resource)]
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
		ctx, cancel := mosiPacketContext(peer)
		_ = r.transmitResource(ctx, id, mosiPacket{ID: uuid.NewString(), Resource: resource, Epoch: record.Epoch, Content: record.Content})
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

// Completed data placement names safety copies independently of RAM ownership.
func replicationBackupRecords(db *database) []replicaRecord {
	r := replication.Load()
	if r == nil || !r.master {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var records []replicaRecord
	for _, backup := range r.backups {
		if backup.Resource.Database == db.Name {
			if table := db.tables.Get(backup.Resource.Table); table != nil && table.PersistencyMode != Memory && table.PersistencyMode != Cache {
				records = append(records, replicaRecord{Resource: backup.Resource, Content: backup.Content})
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
	if record.Content == "" {
		return fmt.Errorf("slave has no completed shard backup")
	}
	transfer := &replicaImport{header: mosiPacket{Resource: record.Resource, Content: record.Content}, engine: db.persistence}
	return r.finishImport(r.id, transfer)
}

// A schema transfer publishes declarations and logical topology. Physical UUIDs
// still name this backend's own data: a peer's schema must not redirect a local
// modifier away from its current content and WAL before data arrives.
func (r *replicationManager) replicaSchemaForLocalData(db *database, data []byte) ([]byte, error) {
	local := make(map[string]string)
	for _, table := range db.tables.GetAll() {
		for _, shard := range table.ActiveShards() {
			if shard != nil {
				local[r.resourceKey(mosiResource{Database: db.Name, Table: table.Name, Shard: shard.uuid.String()})] = shard.uuid.String()
			}
		}
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	var tables map[string]json.RawMessage
	if err := json.Unmarshal(document["tables"], &tables); err != nil {
		return nil, err
	}
	for name, encoded := range tables {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return nil, err
		}
		for _, key := range []string{"Shards", "PShards"} {
			if len(fields[key]) == 0 {
				continue
			}
			var shards []json.RawMessage
			if err := json.Unmarshal(fields[key], &shards); err != nil {
				return nil, err
			}
			for index, raw := range shards {
				if bytes.Equal(raw, []byte("null")) {
					continue
				}
				var identity storageShard
				if err := json.Unmarshal(raw, &identity); err != nil {
					return nil, err
				}
				if content, exists := local[r.resourceKey(mosiResource{Database: db.Name, Table: name, Shard: identity.uuid.String()})]; exists {
					replacement, err := json.Marshal(content)
					if err != nil {
						return nil, err
					}
					shards[index] = replacement
				}
			}
			fields[key], _ = json.Marshal(shards)
		}
		tables[name], _ = json.Marshal(fields)
	}
	document["tables"], _ = json.Marshal(tables)
	return json.Marshal(document)
}

// Schema coherence covers declarations, partitioning and writer-lane shape.
// Rebuild UUID forwarding is data-resource topology, not a schema mutation.
// Digest only the captured schema bytes; never consult a mutable routing view.
func (r *replicationManager) schemaDigest(data []byte) (string, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return "", err
	}
	var tables map[string]json.RawMessage
	if err := json.Unmarshal(document["tables"], &tables); err != nil {
		return "", err
	}
	for name, encoded := range tables {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return "", err
		}
		for _, key := range []string{"Auto_increment", "planner_row_estimate", "key_frequencies", "filter_feedback", "shard_forwards"} {
			delete(fields, key)
		}
		for _, key := range []string{"Shards", "PShards"} {
			if len(fields[key]) == 0 {
				continue
			}
			var shards []*storageShard
			if err := json.Unmarshal(fields[key], &shards); err != nil {
				return "", err
			}
			ids := make([]bool, len(shards))
			for i, shard := range shards {
				if shard != nil {
					ids[i] = true
				}
			}
			fields[key], _ = json.Marshal(ids)
		}
		var columns []map[string]json.RawMessage
		if err := json.Unmarshal(fields["Columns"], &columns); err != nil {
			return "", err
		}
		for _, column := range columns {
			delete(column, "PartitioningScore")
		}
		fields["Columns"], _ = json.Marshal(columns)
		tables[name], _ = json.Marshal(fields)
	}
	document["tables"], _ = json.Marshal(tables)
	normalized, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(normalized)), nil
}

// Complete local ordinary data can retain writes committed by the restarting
// master after a modifier surrendered ownership. Never choose an old slave copy
// over this content/WAL merely because the master's RAM directory was lost.
func (r *replicationManager) completeLocalShard(resource mosiResource) (string, error) {
	db := databases.Get(resource.Database)
	if db == nil {
		return "", nil
	}
	db.ensureLoaded()
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	if resource.Shard == "" {
		return r.schemaDigest(db.persistence.ReadSchema())
	}
	table := db.tables.Get(resource.Table)
	if table == nil || table.PersistencyMode == Memory || table.PersistencyMode == Cache {
		return "", nil
	}
	var current *storageShard
	for _, shard := range table.ActiveShards() {
		if shard != nil && shard.uuid.String() == resource.Shard {
			current = shard
			break
		}
	}
	if current == nil {
		return "", nil
	}
	content := current.uuid.String()
	for _, column := range table.columnDeclarations() {
		if !column.Computor.IsNil() {
			continue
		}
		reader := db.persistence.ReadColumn(content, column.Name)
		var marker [1]byte
		n, err := reader.Read(marker[:])
		reader.Close()
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			return "", nil
		}
	}
	if resource.Table != ".blobs" {
		references, valid := readBlobManifest(db.persistence.ReadColumn(content, blobManifestColumn))
		if !valid {
			return "", nil
		}
		for hash := range references {
			reader := db.persistence.ReadBlob(hash)
			if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
				reader.Close()
				return "", nil
			}
			var marker [1]byte
			_, err := reader.Read(marker[:])
			reader.Close()
			if err != nil && err != io.EOF {
				return "", err
			}
		}
	}
	return content, nil
}

// A source may checkpoint successor UUIDs before swapping its active topology.
// Apply only the routing metadata from that completed native checkpoint; no
// backend contents or former MOSI permission are proof of current authority.
func (r *replicationManager) restoreSchemaRoutes(data []byte) error {
	var schema struct {
		Name   string
		Tables map[string]struct {
			Forwards map[string]string `json:"shard_forwards"`
		}
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}
	for name, table := range schema.Tables {
		for from, to := range table.Forwards {
			r.forwardResource(mosiResource{Database: schema.Name, Table: name, Shard: from}, mosiResource{Database: schema.Name, Table: name, Shard: to})
		}
	}
	return nil
}
