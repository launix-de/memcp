/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"github.com/launix-de/memcp/storage/mosi"
)

// catalogPartitionMethod partitions system_schema relations by database.
// Pivots hold every attached database name ordered by (case fold, name);
// partition i belongs to database Pivots[i], with files on the system backend.
const catalogPartitionMethod = "database-v1"

func catalogPivotLess(a, b string) bool {
	foldedA, foldedB := scm.CaseFoldKey(a), scm.CaseFoldKey(b)
	if foldedA != foldedB {
		return foldedA < foldedB
	}
	return a < b
}

// catalogPartitionInterval returns the partitions whose database name equals
// value under case folding. An unknown name yields an empty interval.
func catalogPartitionInterval(d shardDimension, value scm.Scmer) (int, int) {
	key := scm.CaseFoldKey(scm.String(value))
	lo := sort.Search(len(d.Pivots), func(i int) bool { return scm.CaseFoldKey(scm.String(d.Pivots[i])) >= key })
	hi := lo
	for hi < len(d.Pivots) && scm.CaseFoldKey(scm.String(d.Pivots[hi])) == key {
		hi++
	}
	return lo, hi - 1
}

func catalogPartitionIndex(d shardDimension, value scm.Scmer) int {
	lo, hi := catalogPartitionInterval(d, value)
	for i := lo; i <= hi; i++ {
		if scm.String(d.Pivots[i]) == scm.String(value) {
			return i
		}
	}
	panic("database " + scm.String(value) + " has no system_schema partition")
}

// catalogRoot is the "catalog" member of schema.json: the database's own
// catalog shards. Readers that predate it ignore the member.
type catalogRoot struct {
	Version  int               `json:"version"`
	Revision string            `json:"revision,omitempty"`
	Shards   map[string]string `json:"shards"`
	Forwards map[string]string `json:"forwards,omitempty"`
}

// databaseCatalog owns authoritative definition shards and their last durable
// checkpoint. Go tables/columns are local runtime projections of these rows.
type databaseCatalog struct {
	rootReady atomic.Bool
	revision  atomic.Pointer[string]
	// Guarded by database.catalogMu: the shards named by the newest root
	// handed to the schema.json commit queue, and files awaiting removal.
	committed   [catalogRelationCount]*storageShard
	committedFP [catalogRelationCount][32]byte
	generation  uint64
	retired     []retiredCatalogShard
	forwards    map[string]string

	// Guarded by database.schemalock: the queryable partition. It changes
	// by DML, so triggers of planner caches over system_schema observe it.
	live   [catalogRelationCount]*storageShard
	liveFP [catalogRelationCount][32]byte
	// Remote MOSI publication advances this too, independently of schemalock.
	liveVersion atomic.Uint64
}

type retiredCatalogShard struct {
	shard      *storageShard
	generation uint64
	drained    <-chan struct{}
}

var systemSchemaOnce sync.Once
var systemSchemaDB *database

// Collection validates an epoch after enumerating candidates and collecting
// roots. Publications create fresh UUIDs, so a later publication cannot adopt
// an unowned candidate. Backend I/O never holds a global publication lock.
var catalogPublicationEpoch atomic.Uint64
var catalogPublications atomic.Int64

// Retiring generations remain GC roots until native topology readers drain.
var retiringCatalogGenerations sync.Map // UUID -> *storageShard

// systemSchemaTables are the native catalog relations. Each database contributes
// one isolated partition to each relation; checkpoints use these same tables.
var systemSchemaTables [catalogRelationCount]*table

// systemSchema is resolvable by name but absent from the database directory:
// it is never persisted, listed, rebuilt, cleaned or replicated as a database.
func systemSchema() *database {
	systemSchemaOnce.Do(func() {
		db := newDatabase()
		db.Name = systemSchemaName
		db.srState = SHARED
		db.loadOnce.Do(func() {})
		db.transactionOnce.Do(func() {})
		newRelation := func(relation *catalogRelation) *table {
			t := new(table)
			t.DroppedColumns = make(map[string]bool)
			t.schema = db
			t.Name = relation.name
			t.PersistencyMode = Safe
			for _, field := range relation.columns() {
				c := &column{Name: field.column, Typ: field.kind.sqlType(), AllowNull: true}
				c.UpdateSanitizer()
				t.Columns = append(t.Columns, c)
			}
			t.ShardMode = ShardModePartition
			t.PShards = []*storageShard{}
			t.PDimensions = []shardDimension{{Method: catalogPartitionMethod, Column: "schema_name", Pivots: []scm.Scmer{}}}
			t.publishTopologyLocked()
			t.publishShowColumnsSnapshot(true)
			return t
		}
		for i := range catalogRelations {
			relation := &catalogRelations[i]
			systemSchemaTables[i] = newRelation(relation)
			db.tables.Set(relation.name, systemSchemaTables[i])
		}
		db.publishTableNamesLocked()
		systemSchemaDB = db
	})
	return systemSchemaDB
}

func isSystemSchemaName(name string, ignoreCase bool) bool {
	if ignoreCase {
		return scm.CaseFoldKey(name) == scm.CaseFoldKey(systemSchemaName)
	}
	return name == systemSchemaName
}

func catalogRelationColumns(relation catalogRelationID) []string {
	fields := catalogRelations[relation].columns()
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = field.column
	}
	return names
}

// publishCatalogShards replaces (or with nil removes) one database's live
// partition of the relations whose entry is set.
func publishCatalogShards(name string, shards *[catalogRelationCount]*storageShard) {
	for i, t := range systemSchemaTables {
		if shards != nil && shards[i] == nil {
			continue
		}
		t.mu.Lock()
		dimension := t.PDimensions[0]
		pivots := append([]scm.Scmer(nil), dimension.Pivots...)
		current := append([]*storageShard(nil), t.PShards...)
		position := sort.Search(len(pivots), func(j int) bool { return !catalogPivotLess(scm.String(pivots[j]), name) })
		exists := position < len(pivots) && scm.String(pivots[position]) == name
		switch {
		case shards == nil && !exists:
			t.mu.Unlock()
			continue
		case shards == nil:
			pivots = append(pivots[:position], pivots[position+1:]...)
			current = append(current[:position], current[position+1:]...)
		case exists:
			current[position] = shards[i]
		default:
			pivots = append(pivots[:position], append([]scm.Scmer{scm.NewString(name)}, pivots[position:]...)...)
			current = append(current[:position], append([]*storageShard{shards[i]}, current[position:]...)...)
		}
		dimension.Pivots = pivots
		dimension.NumPartitions = len(pivots)
		t.PDimensions = []shardDimension{dimension}
		t.PShards = current
		t.publishTopologyLocked()
		t.mu.Unlock()
	}
}

// buildDiskShard writes rows as a new relation shard on the system backend.
func buildDiskShard(db *database, relation catalogRelationID, rows [][]scm.Scmer, durable bool) *storageShard {
	shard := newCatalogShard(db, relation, rows)
	shard.catalogHome = catalogBackendDatabase()
	shard.catalogDurable = durable
	complete := false
	defer func() {
		if !complete {
			discardUnpublishedShard(shard)
		}
	}()
	// The source already is a dense private snapshot: it needs neither row-ID
	// translation nor a live rebuild's forwarding/transaction handover.
	for _, name := range catalogRelationColumns(relation) {
		values := shard.columns[name].(*StorageSCMER).values
		column := buildColumnStorage(shard.main_count, shard.home(), true, func(emit func(uint32, scm.Scmer)) {
			for i, value := range values {
				emit(uint32(i), value)
			}
		})
		shard.columns[name] = column
		writer := shard.home().persistence.WriteColumn(shard.uuid.String(), name)
		column.Serialize(writer)
		finishColumnWrite(writer, durable)
	}
	writeBlobManifest(shard)
	shard.catalogDurable = false
	shard.registerCachedStorage()
	complete = true
	return shard
}

// loadDiskShard opens a persisted catalog shard of db.
func loadDiskShard(owner, db *database, relation catalogRelationID, id uuid.UUID, version int, snapshot bool) *storageShard {
	// Opening a historical root must not publish a stale stub over an already
	// installed native successor. Bootstrap resolves identity only; GetRead
	// below still acquires the existing MOSI permission before reading payload.
	r := replication.Load()
	if sharedClusterDatabase(owner.Name) {
		r = nil
		if cluster := activeNodeCluster(); cluster != nil {
			r = cluster.manager
		}
	}
	if r != nil && validReplicaDatabase(owner.Name) {
		resource := mosiResource{Database: owner.Name, Table: catalogResourcePrefix + catalogRelations[relation].name, Shard: id.String()}
		if cell := r.lookupCell(resource); cell != nil {
			if current := cell.publishedShard(); current != nil && current.catalogOwner == owner.Name {
				if !snapshot {
					return current
				}
				// Backup admission reads a private descriptor for the completed
				// content, including an admitted rebuild alias, never its live rights.
				if current.uuid.String() == r.canonicalResource(resource).Shard {
					id = current.uuid
				}
			}
		}
	}
	shard := &storageShard{t: systemSchemaTables[relation], uuid: id, catalogHome: db, catalogOwner: owner.Name, catalogVersion: version}
	shard.catalogPrivate.Store(snapshot)
	shard.columns = make(map[string]ColumnStorage)
	shard.deltaColumns = make(map[string]int)
	shard.deletions.Reset()
	shard.setState(COLD)
	return shard
}

// catalogShardRecords reads the visible rows of a catalog shard with their
// record ids, in recid order.
func catalogShardRecords(tx *TxContext, shard *storageShard, columns []string, tableOwners []string) (recids []uint32, rows [][]scm.Scmer) {
	access := shard.GetRead(tx, false)
	defer access.Release()
	shard = access.shard
	if tableOwners != nil {
		ownerColumn := "table_name"
		if shard.t.Name == catalogRelations[catalogTables].name {
			ownerColumn = "name"
		}
		selected := make(map[uint32]bool)
		for _, owner := range tableOwners {
			value := scm.NewString(owner)
			bounds := runtimeScanAccess(analyzedBoundaries{{col: ownerColumn, matcher: EqualMatcher, lower: value, upper: value, lowerInclusive: true, upperInclusive: true}})
			part := shard.collectRecSet(bounds, []string{ownerColumn}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
				return scm.NewBool(values[0].String() == owner)
			}), tx, SessionStateFromTx(tx))
			part.forEachRange(func(base, count uint32) bool {
				for id := base; id < base+count; id++ {
					selected[id] = true
				}
				return true
			})
		}
		for id := range selected {
			recids = append(recids, id)
		}
		sort.Slice(recids, func(i, j int) bool { return recids[i] < recids[j] })
	} else {
		total := shard.main_count + uint32(len(shard.deltaRows()))
		for id := uint32(0); id < total; id++ {
			visible := !shard.deletions.Get(uint(id))
			if tx != nil {
				visible = tx.IsVisible(shard, id)
			}
			if visible {
				recids = append(recids, id)
			}
		}
	}
	if len(recids) == 0 {
		return
	}
	mainRows := shard.main_count
	delta := shard.deltaRows()
	width := len(columns)
	values := make([]scm.Scmer, len(recids)*width)
	mainEnd := sort.Search(len(recids), func(i int) bool { return recids[i] >= mainRows })
	readers := make([]ColumnReader, width)
	defer releaseColumnReaders(readers)
	deltaPositions := make([]int, width)
	shard.mu.RLock()
	for i, name := range columns {
		position, present := shard.deltaColumns[name]
		deltaPositions[i] = -1
		if present {
			deltaPositions[i] = position
		}
	}
	shard.mu.RUnlock()
	for i, name := range columns {
		if mainEnd != 0 {
			storage := shard.getColumnStorageOrPanic(name, false, tx)
			readers[i] = newCachedColumnReaderTx(storage, tx, false, ReadCompressed)
			readers[i].GetValueMulti(recids[:mainEnd], values[i:], width)
		}
		if position := deltaPositions[i]; position >= 0 {
			for j := mainEnd; j < len(recids); j++ {
				row := delta[recids[j]-mainRows]
				if position < len(row) {
					values[j*width+i] = row[position]
				}
			}
		}
	}
	for i := range recids {
		rows = append(rows, values[i*width:(i+1)*width])
	}
	return
}

// catalogRowKey identifies a row by its kind-normalized encoding, so rows
// read back from compressed storage compare equal to declaration edits.
func catalogRowKey(fields []catalogField, row []scm.Scmer) string {
	return string(appendCatalogRowKey(nil, fields, row))
}

func appendCatalogRowKey(dst []byte, fields []catalogField, row []scm.Scmer) []byte {
	for i, field := range fields {
		cell := row[i]
		if cell.IsNil() {
			dst = append(dst, 0)
			continue
		}
		dst = append(dst, 1)
		if field.kind == catalogBool {
			if scm.ToBool(cell) {
				dst = append(dst, 1)
			} else {
				dst = append(dst, 0)
			}
			continue
		}
		length := len(dst)
		dst = append(dst, 0, 0, 0, 0, 0, 0, 0, 0)
		switch field.kind {
		case catalogText, catalogJSON:
			dst = append(dst, cell.String()...)
		case catalogInt:
			dst = strconv.AppendInt(dst, cell.Int(), 10)
		case catalogFloat:
			dst = strconv.AppendFloat(dst, cell.Float(), 'g', -1, 64)
		case catalogList:
			dst = append(dst, catalogRaw(field.kind, cell)...)
		}
		binary.LittleEndian.PutUint64(dst[length:], uint64(len(dst)-length-8))
	}
	return dst
}

func catalogFingerprint(relation catalogRelationID, rows [][]scm.Scmer) [32]byte {
	fields := catalogRelations[relation].columns()
	hash := sha256.New()
	var length [8]byte
	var key []byte
	for _, row := range rows {
		key = appendCatalogRowKey(key[:0], fields, row)
		binary.LittleEndian.PutUint64(length[:], uint64(len(key)))
		hash.Write(length[:])
		hash.Write(key)
	}
	var result [32]byte
	hash.Sum(result[:0])
	return result
}

// This is a derived declaration epoch, computed only while loading/exporting
// an entire catalog. Ordinary getters read its published scalar. Physical
// generations and advisory statistics do not change SQL definitions; row order
// is immaterial across compaction and peer transfers.
func catalogDefinitionRevision(rows *catalogRows) string {
	hash := sha256.New()
	var length [8]byte
	for relation, records := range rows {
		fields := catalogRelations[relation].columns()
		var selected []catalogField
		var positions []int
		for i, field := range fields {
			switch field.key {
			case "Shards", "PShards", "PDimensions", "ShardMode", "shard_forwards", "Auto_increment",
				"planner_row_estimate", "key_frequencies", "filter_feedback", "PartitioningScore":
				continue
			}
			selected = append(selected, field)
			positions = append(positions, i)
		}
		values := make([]scm.Scmer, len(positions))
		keys := make([][sha256.Size]byte, 0, len(records))
		var key []byte
		for _, row := range records {
			for i, position := range positions {
				values[i] = row[position]
			}
			key = appendCatalogRowKey(key[:0], selected, values)
			keys = append(keys, sha256.Sum256(key))
		}
		slices.SortFunc(keys, func(a, b [sha256.Size]byte) int { return bytes.Compare(a[:], b[:]) })
		binary.LittleEndian.PutUint64(length[:], uint64(len(keys)))
		hash.Write(length[:])
		for _, key := range keys {
			hash.Write(key[:])
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// readCatalogRoot extracts the catalog root of a schema.json, if any.
func readCatalogRoot(data []byte) *catalogRoot {
	var probe struct {
		Catalog *catalogRoot `json:"catalog"`
	}
	if len(data) == 0 || json.Unmarshal(data, &probe) != nil || probe.Catalog == nil {
		return nil
	}
	if probe.Catalog.Version < 1 || probe.Catalog.Version > catalogFormatVersion {
		fmt.Println("warning: unsupported system_schema catalog version", probe.Catalog.Version, "; using the legacy schema")
		return nil
	}
	return probe.Catalog
}

// stripCatalogRoot selects the legacy declaration envelope for legacy admission.
// Native transfers retain their roots and acquire the referenced shards.
func stripCatalogRoot(data []byte) []byte {
	if readCatalogRootMember(data) == nil {
		return data
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return data
	}
	delete(document, "catalog")
	stripped, err := json.Marshal(document)
	if err != nil {
		panic(err)
	}
	return stripped
}

func readCatalogRootMember(data []byte) json.RawMessage {
	var probe struct {
		Catalog json.RawMessage `json:"catalog"`
	}
	if len(data) == 0 || json.Unmarshal(data, &probe) != nil {
		return nil
	}
	return probe.Catalog
}

// loadCatalogDocument reads db's catalog shards named by root and returns
// the equivalent schema document. They become the committed generation.
func (db *database) loadCatalogDocument(root *catalogRoot, snapshot bool) (document []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	systemSchema()
	revision := root.Revision
	db.catalog.revision.Store(&revision)
	db.restoreCatalogRoutes(root)
	var rows catalogRows
	var committed [catalogRelationCount]*storageShard
	var fingerprints [catalogRelationCount][32]byte
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		id, err := uuid.Parse(root.Shards[catalogRelations[i].name])
		if err != nil {
			return nil, fmt.Errorf("catalog relation %s: %v", catalogRelations[i].name, err)
		}
		home := db
		if root.Version >= 2 {
			home = catalogBackendDatabase()
		}
		committed[i] = db.catalog.live[i]
		if snapshot || committed[i] == nil || committed[i].uuid != id || committed[i].home() != home {
			committed[i] = loadDiskShard(db, home, relation, id, root.Version, snapshot)
		}
		_, rows[i] = catalogShardRecords(nil, committed[i], catalogRelationColumns(relation), nil)
		if cell := committed[i].coherence.Load(); cell != nil {
			committed[i] = cell.resolved().publishedShard()
		}
		for _, row := range rows[i] {
			if scm.String(row[0]) != db.Name {
				return nil, fmt.Errorf("catalog relation %s holds rows of database %s", catalogRelations[i].name, scm.String(row[0]))
			}
		}
		fingerprints[i] = catalogFingerprint(relation, rows[i])
	}
	if len(rows[catalogDatabases]) != 1 {
		return nil, fmt.Errorf("catalog of %s lacks its database declaration", db.Name)
	}
	acceptedRevision := catalogDefinitionRevision(&rows)
	db.catalog.revision.Store(&acceptedRevision)
	document = catalogDocumentFromRows(db.Name, &rows)
	if snapshot {
		for _, shard := range committed {
			shard.catalogPrivate.Store(false) // published reads still require native MOSI rights
		}
	}
	db.catalogMu.Lock()
	db.catalog.committed = committed
	db.catalog.committedFP = fingerprints
	db.catalog.live = committed
	db.catalog.liveFP = fingerprints
	publishCatalogShards(db.Name, &committed)
	db.catalogMu.Unlock()
	return document, nil
}

// commitCatalog writes every relation whose rows differ from the committed
// generation and returns the root naming the new generation. The caller holds
// db.catalogMu and keeps it until the root has its place in the commit queue,
// so roots reach disk in generation order.
func (db *database) commitCatalog(rows *catalogRows, live [catalogRelationCount]*storageShard, durable, journaled bool) (*catalogRoot, uint64) {
	state := &db.catalog
	state.generation++
	root := &catalogRoot{Version: catalogFormatVersion, Shards: make(map[string]string, catalogRelationCount), Forwards: make(map[string]string)}
	if revision := db.catalog.revision.Load(); revision != nil {
		root.Revision = *revision
	}
	for from, to := range state.forwards {
		root.Forwards[from] = to
	}
	createdFiles := false
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		var manager *replicationManager
		if live[i] != nil {
			manager = live[i].replicationManager()
		}
		distributed := manager != nil
		fingerprint := catalogFingerprint(relation, rows[i])
		// Equal rows can still belong to a newer native generation, for example
		// after a remote handover. A retained checkpoint's UUID is immutable
		// history, not an available successor identity. Reusing it for active
		// publication would disagree with its already-forwarded MOSI resource.
		historical := distributed && state.committed[i] != nil && manager.canonicalResource(state.committed[i].replicationResource()) != state.committed[i].replicationResource()
		upgrade := (relation == catalogTables || relation == catalogDatabases) && state.committed[i] != nil && state.committed[i].catalogVersion < 3
		// An accepted native transaction already made its catalog WAL durable.
		// Keep that ordinary base+WAL generation instead of recompressing every
		// column after one metadata cell changes. Amortize compaction over a
		// quarter of the base, with a small-table floor to avoid UUID churn.
		keepJournal := false
		dirtyBase := len(live[i].deltaRows()) != 0 || int(live[i].main_count) != len(rows[i])
		if journaled && live[i] == state.committed[i] && !historical && !upgrade && live[i].home() == catalogBackendDatabase() {
			shard := live[i]
			delta := len(shard.deltaRows())
			dead := int(shard.main_count) + delta - len(rows[i])
			keepJournal = delta+dead < max(64, int(shard.main_count)/4)
		}
		// A standalone physical checkpoint also seals outstanding WAL content
		// before its caller may compact the tenant's transaction decision log.
		if state.committed[i] == nil || state.committed[i].home() != catalogBackendDatabase() || !keepJournal && fingerprint != state.committedFP[i] || !journaled && dirtyBase || historical || upgrade {
			shard := buildDiskShard(db, relation, rows[i], durable)
			createdFiles = true
			if previous := state.committed[i]; previous != nil {
				if distributed {
					root.Forwards[previous.uuid.String()] = shard.uuid.String()
				}
				for from, to := range root.Forwards {
					if to == previous.uuid.String() {
						root.Forwards[from] = shard.uuid.String()
					}
				}
				drained := closedCatalogGeneration
				if topology := previous.generation.Load(); topology != nil {
					drained = topology.drained
				}
				state.retired = append(state.retired, retiredCatalogShard{shard: previous, generation: state.generation, drained: drained})
			}
			state.committed[i] = shard
			state.committedFP[i] = fingerprint
		}
		if keepJournal {
			state.committedFP[i] = fingerprint
		}
		root.Shards[catalogRelations[i].name] = state.committed[i].uuid.String()
		if previous := live[i]; distributed && previous.uuid != state.committed[i].uuid {
			root.Forwards[previous.uuid.String()] = state.committed[i].uuid.String()
		}
	}
	state.forwards = root.Forwards
	if durable && createdFiles {
		// These files belong to the system backend; syncing the tenant's
		// schema.json directory cannot make their new entries durable.
		catalogBackendDatabase().persistence.SyncNamespace()
	}
	return root, state.generation
}

// catalogCommitted removes the files of catalog generations that no root
// at or after generation names. Call after that root's commit.
func (db *database) catalogCommitted(generation uint64) {
	db.catalogMu.Lock()
	var removable []replacedShardGeneration
	kept := db.catalog.retired[:0]
	for _, retired := range db.catalog.retired {
		if retired.generation <= generation {
			retiringCatalogGenerations.Store(retired.shard.uuid.String(), retired.shard)
			removable = append(removable, replacedShardGeneration{shard: retired.shard, drained: retired.drained})
		} else {
			kept = append(kept, retired)
		}
	}
	db.catalog.retired = kept
	db.catalogMu.Unlock()
	retireCatalogGenerations(removable)
}

// catalogRowsLocked reads the authoritative catalog partition. The schema owner
// serializes definition edits; ordinary scans keep their immutable shard generation.
func (db *database) catalogRowsLocked(tx *TxContext) (rows catalogRows) {
	for i, shard := range db.catalog.live {
		if shard != nil {
			_, rows[i] = catalogShardRecords(tx, shard, catalogRelationColumns(catalogRelationID(i)), nil)
		}
	}
	return
}

// Materialize a newly received declaration on demand. Native partitions can
// introduce tables without transferring the enclosing database namespace.
// Read/compile the private descriptor before entering the publication lock.
func (db *database) loadCatalogTable(name string) *table {
	if t := db.tables.Get(name); t != nil {
		return t
	}
	var rows catalogRows
	for i := catalogTables; i < catalogProcedures; i++ {
		topology := systemSchemaTables[i].activeTopology()
		position := catalogPartitionIndex(topology.dimensions[0], scm.NewString(db.Name))
		_, rows[i] = catalogShardRecords(nil, topology.shards[position], catalogRelationColumns(i), []string{name})
		if i == catalogTables && len(rows[i]) == 0 {
			return nil
		}
	}
	private := newDatabase()
	if err := json.Unmarshal(catalogDocumentFromRows(db.Name, &rows), private); err != nil {
		panic(err)
	}
	t := private.tables.Get(name)
	db.initializeTableAfterLoad(t)
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	if existing := db.tables.Get(name); existing != nil {
		return existing
	}
	db.tables.Set(name, t)
	if name == ".blobs" {
		db.blobRefState().table.Store(t)
	}
	db.publishTableNamesLocked()
	return t
}

// newCatalogShard constructs an unpublished generation without executing user
// DML, WAL or cache callbacks. Only the catalog editor can call this constructor.
func newCatalogShard(db *database, relation catalogRelationID, rows [][]scm.Scmer) *storageShard {
	shard := NewShard(systemSchemaTables[relation])
	shard.catalogHome = catalogBackendDatabase()
	shard.catalogOwner = db.Name
	shard.catalogVersion = catalogFormatVersion
	shard.catalogPrivate.Store(true)
	for i, name := range catalogRelationColumns(relation) {
		values := make([]scm.Scmer, len(rows))
		for j, row := range rows {
			values[j] = row[i]
		}
		shard.columns[name] = &StorageSCMER{values: values}
	}
	shard.main_count = uint32(len(rows))
	shard.plannerMainRows.Store(shard.main_count)
	return shard
}

// editCatalogLocked atomically applies definition edits through the native
// transaction owner. It never rebuilds definitions from the runtime directory.
// The caller owns schemalock.
func (db *database) editCatalogLocked(tx *TxContext, rows *catalogRows, relations uint16, tableOwners []string) bool {
	owned := tx == nil
	if owned {
		tx = NewTxContext(TxACID)
	} else if tx.State != TxActive || tx.decisionStarted {
		panic("catalog edits require an active original transaction")
	} else if !db.catalog.rootReady.Load() {
		panic("transactional catalog edits require an admitted base root")
	}
	var writes []WriteAccess
	defer func() {
		defer func() {
			for i := len(writes) - 1; i >= 0; i-- {
				writes[i].Release()
			}
		}()
		if owned && tx.State == TxActive {
			tx.Rollback()
		}
	}()
	fingerprints := db.catalog.liveFP
	changed := false
	var created [catalogRelationCount]*storageShard
	for i := range catalogRelations {
		if db.catalog.live[i] == nil {
			created[i] = newCatalogShard(db, catalogRelationID(i), nil)
			declareCatalogShard(created[i])
			db.catalog.live[i] = created[i]
		}
	}
	publishCatalogShards(db.Name, &created)
	for i := range catalogRelations {
		if relations&(1<<i) == 0 {
			continue
		}
		relation := catalogRelationID(i)
		var fp [32]byte
		if tableOwners == nil {
			fp = catalogFingerprint(relation, rows[i])
			current := db.catalog.live[i]
			unchangedGeneration := true
			if cell := current.coherence.Load(); cell != nil {
				unchangedGeneration = cell.resolved().publishedShard() == current
			}
			if owned && unchangedGeneration && fp == db.catalog.liveFP[i] {
				continue
			}
		}
		relationChanged := false
		access := db.catalog.live[i].GetWrite(tx)
		writes = append(writes, access)
		shard := access.shard
		db.catalog.live[i] = shard
		fields := catalogRelations[i].columns()
		columns := catalogRelationColumns(relation)
		wanted := make(map[string]int, len(rows[i]))
		for _, row := range rows[i] {
			wanted[catalogRowKey(fields, row)]++
		}
		recids, existing := catalogShardRecords(tx, shard, columns, tableOwners)
		for j, row := range existing {
			key := catalogRowKey(fields, row)
			if wanted[key] > 0 {
				wanted[key]--
				continue
			}
			shard.updateFunctionBatch(recids[j], true, false, nil, nil, tx, nil, true)()
			relationChanged = true
		}
		var inserts [][]scm.Scmer
		for _, row := range rows[i] {
			key := catalogRowKey(fields, row)
			if wanted[key] == 0 {
				continue
			}
			wanted[key]--
			inserts = append(inserts, row)
		}
		if len(inserts) > 0 {
			shard.insertValues(columns, inserts, false, nil, nil, false, tx, nil, true)
			relationChanged = true
		}
		// This batch invokes native update functions directly, without a scan
		// reducer's Close. Deletion-only edits must join the same WAL barrier
		// as inserts before the original transaction can accept its decision.
		if relationChanged {
			tx.RegisterTouchedShard(shard)
		}
		if !owned {
			// A staged edit cannot publish a committed-definition fingerprint.
			// The next checkpoint/edit must compare its accepted row view.
			fingerprints[i] = [32]byte{}
		} else if relationChanged || tableOwners == nil {
			fingerprints[i] = fp
		}
		changed = changed || relationChanged
	}
	if !changed {
		db.catalog.liveFP = fingerprints
		return false
	}
	if owned {
		if err := tx.Commit(); err != nil {
			panic(err)
		}
	}
	db.catalog.liveFP = fingerprints
	db.catalog.liveVersion.Add(1)
	return true
}

// proposeCatalogTablesLocked admits explicit DDL proposals for named tables.
// Runtime objects are builders here, never an input to a save or catalog read.
// Only the named declarations are replaced; unrelated catalog rows are retained.
func (db *database) proposeCatalogTablesLocked(tx *TxContext, tables ...*table) {
	if db == systemSchemaDB {
		return
	}
	var rows catalogRows
	var owners []string
	seen := make(map[string]bool)
	for _, t := range tables {
		if t == nil || t.schema != db || t.RuntimeOnly || seen[t.Name] {
			continue
		}
		seen[t.Name] = true
		owners = append(owners, t.Name)
		proposal := catalogRowsFromTable(t)
		proposal = db.catalogDefinitionProposal(&proposal)
		for i := catalogTables; i < catalogProcedures; i++ {
			rows[i] = append(rows[i], proposal[i]...)
		}
	}
	if len(owners) == 0 {
		return
	}
	// A new compiled callback can share an unchanged persisted declaration.
	// Keep that runtime state without decoding the same canonical rows again.
	if !db.editCatalogLocked(tx, &rows, catalogAllRelations&^(1<<catalogDatabases|1<<catalogProcedures), owners) {
		return
	}
	for _, t := range tables {
		if t != nil && t.schema == db {
			for _, row := range rows[catalogTables] {
				if catalogTableName(row) == t.Name {
					db.materializeCatalogTableLocked(t, &rows)
					break
				}
			}
		}
	}
}

// Planner workspaces are physical runtime objects, not logical declarations.
// Keep them local in standalone execution too: a cold query must not durably
// rewrite schema partitions to install its reconstructible cache program.
func (db *database) catalogDefinitionProposal(rows *catalogRows) catalogRows {
	r := replication.Load()
	if r == nil || sharedClusterDatabase(db.Name) || !validReplicaDatabase(db.Name) {
		return *rows
	}
	restricted := !r.master
	declarations, err := replicaSchemaDeclarations(catalogDocumentFromRows(db.Name, rows), restricted)
	if err != nil {
		panic(err)
	}
	return catalogRowsFromDocument(declarations)
}

func (db *database) prepareReplicaCatalog() {
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	rows := db.catalogRowsLocked(nil)
	rows = db.catalogDefinitionProposal(&rows)
	if !db.editCatalogLocked(nil, &rows, catalogAllRelations, nil) {
		return
	}
	for _, t := range db.tables.GetAll() {
		if !t.isEphemeralQueryTable() {
			db.materializeCatalogTableLocked(t, &rows)
		}
	}
	document := catalogRecordsFromRows(db.Name, &rows)
	for _, raw := range document.Procedures {
		var definition ProcedureDescription
		if err := json.Unmarshal(raw, &definition); err != nil {
			panic(err)
		}
		db.procedures.Set(definition.Name, &definition)
	}
}

func catalogTableName(row []scm.Scmer) string {
	for i, field := range catalogRelations[catalogTables].columns() {
		if field.key == "Name" {
			return row[i].String()
		}
	}
	panic("catalog tables lacks Name")
}

func (db *database) dropCatalogTableLocked(tx *TxContext, name string) {
	rows := db.catalogRowsLocked(tx)
	for i := catalogTables; i < catalogProcedures; i++ {
		kept := make([][]scm.Scmer, 0, len(rows[i]))
		for _, row := range rows[i] {
			owner := row[1].String()
			if i == catalogTables {
				owner = catalogTableName(row)
			}
			if owner != name {
				kept = append(kept, row)
			}
		}
		rows[i] = kept
	}
	db.editCatalogLocked(tx, &rows, catalogAllRelations, nil)
}

// Rename all owned definitions in one native decision. A reader's snapshot
// sees either complete name, including its columns, keys and triggers.
func (db *database) renameCatalogTableLocked(tx *TxContext, oldname, newname string) {
	rows := db.catalogRowsLocked(tx)
	for i := catalogTables; i < catalogProcedures; i++ {
		owner := 1
		if i == catalogTables {
			for j, field := range catalogRelations[i].columns() {
				if field.key == "Name" {
					owner = j
					break
				}
			}
		}
		for _, row := range rows[i] {
			if row[owner].String() == oldname {
				row[owner] = scm.NewString(newname)
			}
			if i == catalogForeignKeys {
				for index, field := range catalogRelations[i].columns() {
					if (field.key == "Tbl1" || field.key == "Tbl2") && row[index].String() == oldname {
						row[index] = scm.NewString(newname)
					}
				}
			}
		}
	}
	db.editCatalogLocked(tx, &rows, catalogAllRelations, nil)
}

func (db *database) editCatalogProcedureLocked(tx *TxContext, name string, proposal *ProcedureDescription) *ProcedureDescription {
	rows := db.catalogRowsLocked(tx)
	definitions := make([]map[string]json.RawMessage, 0, len(rows[catalogProcedures])+1)
	for _, row := range rows[catalogProcedures] {
		encoder := catalogJSONEncoder{strings: make(map[string]json.RawMessage)}
		record := encoder.record(catalogProcedures, row)
		var existing string
		if err := json.Unmarshal(record["name"], &existing); err != nil {
			panic(err)
		}
		if existing != name {
			definitions = append(definitions, record)
		}
	}
	if proposal != nil {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(marshalCatalogValue(proposal), &record); err != nil {
			panic(err)
		}
		definitions = append(definitions, record)
	}
	sort.Slice(definitions, func(i, j int) bool { return string(definitions[i]["name"]) < string(definitions[j]["name"]) })
	rows[catalogProcedures] = nil
	for i, definition := range definitions {
		rows.add(catalogProcedures, db.Name, []scm.Scmer{scm.NewInt(int64(i))}, definition)
	}
	rows = db.catalogDefinitionProposal(&rows)
	db.editCatalogLocked(tx, &rows, catalogAllRelations, nil)
	if proposal == nil {
		return nil
	}
	// Materialize the definition from the admitted row, retaining validated
	// process-local compiled code on this equivalent immutable declaration.
	for _, row := range rows[catalogProcedures] {
		var definition ProcedureDescription
		encoder := catalogJSONEncoder{strings: make(map[string]json.RawMessage)}
		if err := json.Unmarshal(encoder.appendRow(catalogProcedures, row), &definition); err != nil {
			panic(err)
		}
		if definition.Name == name {
			definition.compiled.Store(proposal.compiled.Load())
			return &definition
		}
	}
	panic("admitted procedure is missing")
}

// checkpointCatalogLocked updates only physical generations, advisory statistics
// and the ID allocator. User declarations always come from the catalog tables.
func (db *database) checkpointCatalogLocked(tx *TxContext, tables ...*table) {
	var owners []string
	if len(tables) != 0 {
		for _, t := range tables {
			if t != nil && t.schema == db && !t.RuntimeOnly {
				owners = append(owners, t.Name)
			}
		}
		if len(owners) == 0 {
			return
		}
	}
	var rows catalogRows
	if shard := db.catalog.live[catalogTables]; shard != nil {
		_, rows[catalogTables] = catalogShardRecords(tx, shard, catalogRelationColumns(catalogTables), owners)
	}
	for _, row := range rows[catalogTables] {
		t := db.tables.Get(catalogTableName(row))
		if t == nil {
			continue
		}
		if len(tables) != 0 && !slices.Contains(tables, t) {
			continue
		}
		topology := t.schemaTopology.Load()
		if topology == nil {
			continue
		}
		for i, field := range catalogRelations[catalogTables].columns() {
			var value any
			switch field.key {
			case "Shards":
				if topology.mode == ShardModeFree {
					value = topology.shards
				}
			case "PShards":
				if topology.mode == ShardModePartition {
					value = topology.shards
				}
			case "PDimensions":
				value = topology.dimensions
			case "ShardMode":
				mode := topology.mode
				value = &mode
			case "shard_forwards":
				value = topology.forwards
			case "Auto_increment":
				value = atomic.LoadUint64(&t.Auto_increment)
			case "planner_row_estimate":
				value = t.PlannerRowEstimate.value.Load()
			case "key_frequencies":
				value = t.persistKeyFrequencies()
			case "filter_feedback":
				value = t.persistFilterFeedback()
			default:
				continue
			}
			row[i] = catalogCell(field.kind, marshalCatalogValue(value))
		}
	}
	db.editCatalogLocked(tx, &rows, 1<<catalogTables, owners)
}

// materializeCatalogTableLocked publishes a local runtime projection of the
// admitted rows. Matching column and trigger objects retain their compiled code
// and cache lifetimes. Definition values themselves are decoded from the catalog.
func (db *database) materializeCatalogTableLocked(t *table, rows *catalogRows) {
	var selected catalogRows
	initializer := t.OnInit.Load()
	retainInitializer := false
	for i := catalogTables; i < catalogProcedures; i++ {
		for _, row := range rows[i] {
			owner := row[1].String()
			if i == catalogTables {
				owner = catalogTableName(row)
			}
			if owner == t.Name {
				if i == catalogTables && initializer != nil {
					for index, field := range catalogRelations[catalogTables].columns() {
						if field.key == "oninit" && row[index].IsString() && row[index].String() == string(marshalCatalogValue(initializer)) {
							// The compiled initializer already represents this admitted
							// cell exactly. Retain its runtime bindings without decoding
							// the same Scheme tree only to discard it afterwards.
							retainInitializer = true
							row = append([]scm.Scmer(nil), row...)
							row[index] = scm.NewNil()
							break
						}
					}
				}
				selected[i] = append(selected[i], row)
			}
		}
	}
	document := catalogRecordsFromRows(db.Name, &selected)
	var definition table
	if err := json.Unmarshal(document.Tables[t.Name], &definition); err != nil {
		panic(err)
	}
	for i, c := range definition.Columns {
		reused := false
		for _, candidate := range t.Columns {
			if candidate.Name != c.Name {
				continue
			}
			// Encoding compares declaration data only, never atomics, pointers,
			// runtime computors or compiled code ownership.
			if catalogColumnDefinitionEqual(candidate, c) {
				definition.Columns[i] = candidate
				reused = true
			}
			break
		}
		if !reused {
			definition.Columns[i].UpdateSanitizer()
		}
	}
	for _, candidate := range t.Columns {
		if !candidate.IsTemp {
			continue
		}
		found := false
		for _, declared := range definition.Columns {
			found = found || declared.Name == candidate.Name
		}
		if !found {
			// These are retained physical compute objects of this node, not
			// declarations supplied by the replicated catalog generation.
			definition.Columns = append(definition.Columns, candidate)
		}
	}
	t.Columns = definition.Columns
	t.RuntimeIdentity = definition.RuntimeIdentity
	t.Unique, t.Foreign = definition.Unique, definition.Foreign
	t.DroppedColumns = definition.DroppedColumns
	if t.PersistencyMode != definition.PersistencyMode {
		t.PersistencyMode = definition.PersistencyMode
	}
	t.Collation, t.Charset, t.Comment = definition.Collation, definition.Charset, definition.Comment
	if !retainInitializer {
		t.OnInit.Store(definition.OnInit.Load())
	}
	t.mu.Lock()
	for i, declaration := range definition.Triggers {
		for _, candidate := range t.Triggers {
			if candidate.Name == declaration.Name && catalogTriggerDefinitionEqual(candidate, declaration) {
				definition.Triggers[i] = candidate
				break
			}
		}
	}
	for _, candidate := range t.Triggers {
		if candidate.cacheTarget == nil && candidate.Acquire == nil && !restoredTriggerRequiresRuntimeTarget(candidate.Name) {
			continue
		}
		found := false
		for _, declared := range definition.Triggers {
			if declared.Name == candidate.Name {
				found = true
				break
			}
		}
		if !found {
			definition.Triggers = append(definition.Triggers, candidate)
		}
	}
	t.Triggers = definition.Triggers
	t.mu.Unlock()
	t.publishShowColumnsSnapshot(true)
}

// commitSchemaLocked exports the legacy document FROM the catalog tables and
// persists the same definitions in shard form. It never snapshots Go definitions.
func (db *database) commitSchemaLocked(tx *TxContext, durable bool, release func()) {
	released := false
	defer func() {
		if !released {
			release()
		}
	}()
	if db.schemaRecoveryRequired.Load() {
		panic(fmt.Errorf("schema checkpoint requires recovery of the uncertain original transaction"))
	}
	// A checkpoint can compact a generation received since the last local
	// save. Claim its native authority before publishing successor UUIDs;
	// namespace ownership alone never grants rights on these partitions.
	for i, shard := range db.catalog.live {
		if r := shard.replicationManager(); r != nil {
			cell := r.obtain(shard.replicationResource(), mosi.M, "", tx)
			if current := cell.publishedShard(); current != nil && current != shard {
				db.catalog.live[i] = current
				db.catalog.liveFP[i] = [32]byte{}
			}
		}
	}
	readTx := tx
	if tx != nil && tx.State == TxCommitted {
		readTx = nil // export accepted rows; the original decision is complete
	}
	rows := db.catalogRowsLocked(readTx)
	if tx == nil || tx.State == TxCommitted {
		revision := catalogDefinitionRevision(&rows)
		db.catalog.revision.Store(&revision)
	}
	version := db.catalog.liveVersion.Load()
	live := db.catalog.live
	jsonbytes := catalogDocumentFromRows(db.Name, &rows)
	catalogPublications.Add(1)
	catalogPublicationEpoch.Add(1)
	defer func() {
		catalogPublicationEpoch.Add(1)
		catalogPublications.Add(-1)
	}()
	db.catalogMu.Lock()
	released = true
	release()
	var root *catalogRoot
	var generation uint64
	var committed [catalogRelationCount]*storageShard
	func() {
		defer func() {
			if r := recover(); r != nil {
				db.catalogMu.Unlock()
				panic(r)
			}
		}()
		// Materializing accepted catalog rows must not overtake an earlier
		// uncertain tenant decision barrier. Its data and catalog WALs already
		// share this authority; make that authority durable before replacing
		// their replayable catalog generation with an unconditional snapshot.
		if tx == nil || tx.State != TxCommitted {
			func() {
				db.transactionMu.RLock()
				defer db.transactionMu.RUnlock()
				if db.transactionLog != nil {
					db.transactionLog.Flush(durable)
				}
			}()
		}
		root, generation = db.commitCatalog(&rows, live, durable, tx != nil && tx.State == TxCommitted)
		committed = db.catalog.committed
		jsonbytes = attachCatalogRoot(jsonbytes, root)
	}()
	db.commitSchemaSnapshot(jsonbytes, durable, db.catalogMu.Unlock)
	db.catalog.rootReady.Store(true)
	db.schemaExportPending.Store(false)
	db.schemalock.Lock()
	func() {
		defer db.schemalock.Unlock()
		// Use native writer admission for this short publication, so every
		// handover adapter drains it before binding payload and UUID routes.
		// Compression and I/O never retain these writer capabilities.
		var publications []*mosi.Writer
		defer func() {
			for _, writer := range publications {
				writer.Release()
			}
		}()
		publish := db.catalog.liveVersion.Load() == version
		if publish {
			for _, shard := range db.catalog.live {
				if cell := shard.coherence.Load(); cell != nil {
					cell = cell.resolved()
					writer, _ := cell.ownstate.TryWrite(nil, false, nil)
					if writer == nil {
						publish = false
						break
					}
					publications = append(publications, writer)
				}
			}
		}
		publish = publish && db.catalog.liveVersion.Load() == version
		if publish {
			db.catalog.liveFP = db.catalog.committedFP
			for i, shard := range committed {
				if previous := db.catalog.live[i]; previous != shard {
					if r := previous.replicationManager(); r != nil {
						if err := r.replaceCatalogShard(previous.replicationResource(), previous, shard); err != nil {
							panic(err)
						}
						shard.coherence.Load().resolved().current.Store(shard)
					} else {
						declareCatalogShard(shard)
					}
				}
			}
			db.restoreCatalogRoutes(root)
			db.catalog.live = committed
			publishCatalogShards(db.Name, &committed)
			db.schemaDirty.Store(false)
		} else {
			// A later local edit is still the live authority. The durable snapshot
			// is a historical UUID, so route it to that live cell until the newer
			// checkpoint publishes its own successor. Do not mint another owner.
			for i, snapshot := range committed {
				if live := db.catalog.live[i]; live != snapshot {
					if r := live.replicationManager(); r != nil {
						r.forwardResource(snapshot.replicationResource(), r.canonicalResource(live.replicationResource()))
					}
				}
			}
		}
	}()
	db.catalogCommitted(generation)
}

// attachCatalogRoot appends the catalog root to a schema.json document.
func attachCatalogRoot(document []byte, root *catalogRoot) []byte {
	if root == nil {
		return document
	}
	encoded, err := json.Marshal(root)
	if err != nil {
		panic(err)
	}
	trimmed := bytes.TrimRight(document, " \t\r\n")
	if len(trimmed) < 2 || trimmed[len(trimmed)-1] != '}' {
		panic("schema document is not a JSON object")
	}
	body := bytes.TrimRight(trimmed[:len(trimmed)-1], " \t\r\n")
	result := make([]byte, 0, len(document)+len(encoded)+16)
	result = append(result, body...)
	if body[len(body)-1] != '{' {
		result = append(result, ',')
	}
	result = append(result, "\n  \"catalog\": "...)
	result = append(result, encoded...)
	return append(result, "\n}"...)
}

// detachCatalog removes db's live partition when the database is dropped.
// Its rows leave by DML first, so caches over system_schema observe them.
func (db *database) detachCatalog() {
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	var retired []replacedShardGeneration
	db.catalogMu.Lock()
	seen := make(map[*storageShard]bool)
	for _, shard := range db.catalog.committed {
		if shard == nil || seen[shard] {
			continue
		}
		seen[shard] = true
		drained := closedCatalogGeneration
		if topology := shard.generation.Load(); topology != nil {
			drained = topology.drained
		}
		retiringCatalogGenerations.Store(shard.uuid.String(), shard)
		retired = append(retired, replacedShardGeneration{shard, drained})
	}
	db.catalogMu.Unlock()
	db.catalog.live = [catalogRelationCount]*storageShard{}
	publishCatalogShards(db.Name, nil)
	retireCatalogGenerations(retired)
}

// catalogShardIDs lists the catalog shard files GC must keep for db: the
// committed generation and every retired one whose removal is pending.
func (db *database) catalogShardIDs() []string {
	var ids []string
	// A peer can publish a received native partition before its schema root is
	// materialized locally. The native topology also owns those physical files.
	for _, relation := range systemSchemaTables {
		if relation != nil {
			for _, shard := range relation.activeTopology().shards {
				if shard != nil && shard.home() == db {
					ids = append(ids, shard.uuid.String())
				}
			}
		}
	}
	owners := []*database{db}
	if db.Name == "system" {
		owners = databases.GetAll()
	}
	for _, owner := range owners {
		// Durable anchors own their files independently of runtime loading.
		// A private snapshot can have completed catalogBootstrap before its
		// descriptors are published; an empty projection is no deletion proof.
		if owner.persistence != nil {
			if root := readCatalogRoot(owner.persistence.ReadSchema()); root != nil && (root.Version >= 2 && db.Name == "system" || root.Version == 1 && owner == db) {
				for _, id := range root.Shards {
					ids = append(ids, id)
				}
			}
		}
		owner.catalogMu.Lock()
		for _, shard := range owner.catalog.committed {
			if shard != nil && shard.home() == db {
				ids = append(ids, shard.uuid.String())
			}
		}
		for _, retired := range owner.catalog.retired {
			if retired.shard.home() == db {
				ids = append(ids, retired.shard.uuid.String())
			}
		}
		owner.catalogMu.Unlock()
	}
	retiringCatalogGenerations.Range(func(key, value any) bool {
		if value.(*storageShard).home() == db {
			ids = append(ids, key.(string))
		}
		return true
	})
	return ids
}

// initializeEmptyCatalog creates only this database's empty definitions. It is
// also used by CREATE DATABASE before any runtime table can exist.
func (db *database) initializeEmptyCatalog() {
	systemSchema()
	var rows catalogRows
	rows.add(catalogDatabases, db.Name, nil, nil)
	db.editCatalogLocked(nil, &rows, catalogAllRelations, nil)
}

func foreignCatalogTables(t *table) []*table {
	var result []*table
	for _, fk := range t.Foreign {
		if parent := t.schema.tables.Get(fk.Tbl2); parent != nil && parent != t {
			result = append(result, parent)
		}
	}
	return result
}

func requireCatalogDML(t *table) {
	if nativeCatalogRelation(t) {
		panic("schema catalog tables are read-only; use native DDL")
	}
}

// bootstrapCatalog attaches persisted shard stubs without loading runtime table
// definitions or user-data columns. A legacy document is imported once.
func (db *database) bootstrapCatalog(document []byte) {
	db.catalogBootstrap.Do(func() {
		if db.catalog.live[0] != nil {
			return
		}
		systemSchema()
		var data []byte
		if document != nil {
			data = document
		} else {
			data = db.persistence.ReadSchema()
		}
		if root := readCatalogRoot(data); root != nil {
			revision := root.Revision
			db.catalog.revision.Store(&revision)
			db.catalog.rootReady.Store(true)
			db.catalog.forwards = root.Forwards
			db.restoreCatalogRoutes(root)
			for i, relation := range catalogRelations {
				id, err := uuid.Parse(root.Shards[relation.name])
				if err != nil {
					panic(err)
				}
				home := db
				if root.Version >= 2 {
					home = catalogBackendDatabase()
				}
				shard := loadDiskShard(db, home, catalogRelationID(i), id, root.Version, false)
				db.catalog.live[i], db.catalog.committed[i] = shard, shard
			}
			publishCatalogShards(db.Name, &db.catalog.live)
			return
		}
		if len(data) == 0 {
			db.initializeEmptyCatalog()
			return
		}
		rows := catalogRowsFromDocument(data)
		db.editCatalogLocked(nil, &rows, catalogAllRelations, nil)
	})
}

func nativeCatalogRelation(t *table) bool {
	if t.schema != systemSchemaDB {
		return false
	}
	for _, relation := range systemSchemaTables {
		if t == relation {
			return true
		}
	}
	return false
}

// Catalog generations live in the unchanged backend of the system database.
// A tenant's schema.json remains its anchor and contains both legacy definitions
// and its own catalog UUIDs. Version 1 roots keep their original tenant backend.
func catalogBackendDatabase() *database {
	if db := databases.Get("system"); db != nil {
		return db
	}
	CreateDatabase("system", true)
	return databases.Get("system")
}

var closedCatalogGeneration <-chan struct{} = func() <-chan struct{} {
	closed := make(chan struct{})
	close(closed)
	return closed
}()

func retireCatalogGenerations(generations []replacedShardGeneration) {
	for _, generation := range generations {
		retiringCatalogGenerations.Store(generation.shard.uuid.String(), generation.shard)
		cleanup := func() {
			if failures := cleanupReplacedShardGenerations([]replacedShardGeneration{generation}, "schema catalog"); len(failures) == 0 {
				retiringCatalogGenerations.Delete(generation.shard.uuid.String())
			} else {
				for _, failure := range failures {
					fmt.Println(failure)
				}
			}
		}
		select {
		case <-generation.drained:
			cleanup()
		default:
			go func() { <-generation.drained; cleanup() }()
		}
	}
}

// publishCatalogDatabaseName reserves a keyed routing slot without reading a
// tenant's schema.json. Nil slots are resolved at the native scan boundary.
func publishCatalogDatabaseName(name string) {
	systemSchema()
	for _, t := range systemSchemaTables {
		t.mu.Lock()
		dimension := t.PDimensions[0]
		position := sort.Search(len(dimension.Pivots), func(j int) bool { return !catalogPivotLess(scm.String(dimension.Pivots[j]), name) })
		if position == len(dimension.Pivots) || scm.String(dimension.Pivots[position]) != name {
			pivots := append([]scm.Scmer(nil), dimension.Pivots...)
			pivots = append(pivots, scm.NewNil())
			copy(pivots[position+1:], pivots[position:])
			pivots[position] = scm.NewString(name)
			shards := append([]*storageShard(nil), t.PShards...)
			shards = append(shards, nil)
			copy(shards[position+1:], shards[position:])
			shards[position] = nil
			dimension.Pivots, dimension.NumPartitions = pivots, len(pivots)
			t.PDimensions, t.PShards = []shardDimension{dimension}, shards
			// Unknown cold cardinality is an estimate, never proof of emptiness.
			t.PlannerRowEstimate.value.Store(max(t.PlannerRowEstimate.value.Load(), uint64(256*len(pivots))))
			t.publishTopologyLocked()
		}
		t.mu.Unlock()
	}
}

// scanTopology resolves only the selected database keys before pinning native
// shard generations. All scan variants retain their existing access schema.
func (t *table) scanTopology(tx *TxContext, access scanAccess) *tableShardTopology {
	if !nativeCatalogRelation(t) {
		return t.activeTopology()
	}
	topology := t.activeTopology()
	dimension := topology.dimensions[0]
	minimum, maximum := 0, len(dimension.Pivots)-1
	for i := 0; i < access.len(); i++ {
		if access.boundaryColumn(i) != "schema_name" {
			continue
		}
		if lo, hi, usable := partitionAccessInterval(dimension, access, i); usable {
			minimum, maximum = lo, hi
			break
		}
	}
	for i := minimum; i <= maximum; i++ {
		// Ordinary metadata scans retain the logical tenant's effective
		// definitions. Internal catalog checkpointing uses only the fixed
		// system_schema layout and its native partition rights, so a sealed
		// schema handover never waits for its own declaration admission.
		db := databases.Get(dimension.Pivots[i].String())
		tx.readSchema(db)
		if topology.shards[i] != nil {
			continue
		}
		if db != nil {
			db.bootstrapCatalog(nil)
		}
	}
	return t.activeTopology()
}

// Partition traffic is advisory runtime state. Equal declaration projections
// retain their native computed-column lifetimes and compiled value recipes.
var catalogColumnFields = catalogRecordFields(reflect.TypeOf(column{}), nil, "PartitioningScore")

func catalogColumnDefinitionEqual(a, b *column) bool {
	left, right := make([]scm.Scmer, len(catalogColumnFields)), make([]scm.Scmer, len(catalogColumnFields))
	for i, field := range catalogColumnFields {
		left[i] = catalogFieldValue(field, reflect.ValueOf(a))
		right[i] = catalogFieldValue(field, reflect.ValueOf(b))
	}
	return catalogRowKey(catalogColumnFields, left) == catalogRowKey(catalogColumnFields, right)
}

// requires_target describes a process-local pin, not the trigger body. The
// legacy loader installs an unavailable pin for cache families, even while a
// new live trigger is still waiting for SetTriggerTarget. Keep its compiled
// callback and current lifecycle state when the admitted declaration matches.
func catalogTriggerDefinitionEqual(a, b TriggerDescription) bool {
	var left, right map[string]json.RawMessage
	if err := json.Unmarshal(marshalCatalogValue(a), &left); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(marshalCatalogValue(b), &right); err != nil {
		panic(err)
	}
	delete(left, "requires_target")
	delete(right, "requires_target")
	return bytes.Equal(marshalCatalogValue(left), marshalCatalogValue(right))
}
