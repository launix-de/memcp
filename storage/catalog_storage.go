/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
)

// catalogPartitionMethod partitions system_schema relations by database.
// Pivots hold every attached database name ordered by (case fold, name);
// partition i lives on the backend of database Pivots[i].
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
	Version int               `json:"version"`
	Shards  map[string]string `json:"shards"`
}

// databaseCatalog is one database's part of system_schema. Its disk
// generation and its live partition are separate shards with equal rows.
type databaseCatalog struct {
	// Guarded by database.catalogMu: the shards named by the newest root
	// handed to the schema.json commit queue, and files awaiting removal.
	committed   [catalogRelationCount]*storageShard
	committedFP [catalogRelationCount][32]byte
	generation  uint64
	retired     []retiredCatalogShard

	// Guarded by database.catalogLiveMu: the queryable partition. It changes
	// by DML, so triggers of planner caches over system_schema observe it.
	live         [catalogRelationCount]*storageShard
	liveFP       [catalogRelationCount][32]byte
	liveSnapshot uint64
}

type retiredCatalogShard struct {
	shard      *storageShard
	generation uint64
}

var systemSchemaOnce sync.Once
var systemSchemaDB *database

// systemSchemaTables are the queryable relations: memory tables whose
// partition i holds the rows of database Pivots[i], including the transient
// handle column. catalogDiskTables are unpublished twins without handles; their
// shards are the files a schema.json root names.
var systemSchemaTables [catalogRelationCount]*table
var catalogDiskTables [catalogRelationCount]*table

// systemSchema is resolvable by name but absent from the database directory:
// it is never persisted, listed, rebuilt, cleaned or replicated as a database.
func systemSchema() *database {
	systemSchemaOnce.Do(func() {
		db := newDatabase()
		db.Name = systemSchemaName
		db.srState = SHARED
		db.loadOnce.Do(func() {})
		db.transactionOnce.Do(func() {})
		newRelation := func(relation *catalogRelation, mode PersistencyMode, handle bool) *table {
			t := new(table)
			t.DroppedColumns = make(map[string]bool)
			t.schema = db
			t.Name = relation.name
			t.PersistencyMode = mode
			for _, field := range relation.columns() {
				c := &column{Name: field.column, Typ: field.kind.sqlType(), AllowNull: true}
				c.UpdateSanitizer()
				t.Columns = append(t.Columns, c)
			}
			if handle {
				t.Columns = append(t.Columns, &column{Name: "handle", Typ: "ANY", AllowNull: true})
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
			systemSchemaTables[i] = newRelation(relation, Memory, relation.handle)
			catalogDiskTables[i] = newRelation(relation, Sloppy, false)
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

// catalogHandles resolves the RAM table of every row of relations with a
// handle column. The caller owns db.schemalock.
func catalogHandles(db *database, rows *catalogRows) (handles [catalogRelationCount][]scm.Scmer) {
	for i := range catalogRelations {
		if !catalogRelations[i].handle {
			continue
		}
		nameColumn := 0
		for j, field := range catalogRelations[i].columns() {
			if field.column == "name" {
				nameColumn = j
			}
		}
		handles[i] = make([]scm.Scmer, len(rows[i]))
		for j, row := range rows[i] {
			if t := db.tables.Get(scm.String(row[nameColumn])); t != nil {
				handles[i][j] = NewTableScmer(t)
			}
		}
	}
	return
}

// buildDiskShard writes rows as a new shard of relation on db's backend.
func buildDiskShard(db *database, relation catalogRelationID, rows [][]scm.Scmer, durable bool) *storageShard {
	shard := NewShard(catalogDiskTables[relation])
	shard.catalogHome = db
	if len(rows) != 0 {
		shard.Insert(catalogRelationColumns(relation), rows, false, nil, nil, false, nil, nil)
	}
	shard.catalogDurable = durable
	rebuilt := shard.rebuild(true)
	rebuilt.catalogDurable = false
	return rebuilt
}

// loadDiskShard opens a persisted catalog shard of db.
func loadDiskShard(db *database, relation catalogRelationID, id uuid.UUID) *storageShard {
	shard := &storageShard{t: catalogDiskTables[relation], uuid: id, catalogHome: db}
	shard.columns = make(map[string]ColumnStorage)
	shard.deltaColumns = make(map[string]int)
	shard.deletions.Reset()
	shard.setState(COLD)
	return shard
}

// catalogShardRecords reads the visible rows of a catalog shard with their
// record ids, in recid order.
func catalogShardRecords(shard *storageShard, columns []string) (recids []uint32, rows [][]scm.Scmer) {
	shard.ensureLoaded()
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.ensureMainCount(true, nil)
	total := shard.main_count + uint32(len(shard.deltaRows()))
	for recid := uint32(0); recid < total; recid++ {
		if shard.deletions.Get(uint(recid)) {
			continue
		}
		row := make([]scm.Scmer, len(columns))
		for i, column := range columns {
			row[i] = shard.rowValueByRecidLocked(recid, column)
		}
		recids = append(recids, recid)
		rows = append(rows, row)
	}
	return
}

// catalogRowKey identifies a row by its kind-normalized encoding, so rows
// read back from compressed storage compare equal to generated ones. A
// handle takes part by identity: a replaced RAM table needs a new row.
func catalogRowKey(fields []catalogField, row []scm.Scmer, handle scm.Scmer) string {
	var b strings.Builder
	for i, field := range fields {
		raw := catalogRaw(field.kind, row[i])
		fmt.Fprintf(&b, "%d:", len(raw))
		b.Write(raw)
	}
	if !handle.IsNil() {
		fmt.Fprintf(&b, "%p", TableFromScmer(handle))
	}
	return b.String()
}

func catalogFingerprint(relation catalogRelationID, rows [][]scm.Scmer, handles []scm.Scmer) [32]byte {
	fields := catalogRelations[relation].columns()
	hash := sha256.New()
	for i, row := range rows {
		handle := scm.NewNil()
		if handles != nil {
			handle = handles[i]
		}
		key := catalogRowKey(fields, row, handle)
		fmt.Fprintf(hash, "%d:%s", len(key), key)
	}
	var result [32]byte
	copy(result[:], hash.Sum(nil))
	return result
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

// stripCatalogRoot removes the node-local catalog root from a schema.json
// before it leaves this node. Receivers derive their own catalog shards.
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
func (db *database) loadCatalogDocument(root *catalogRoot) (document []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	systemSchema()
	var rows catalogRows
	var committed [catalogRelationCount]*storageShard
	var fingerprints [catalogRelationCount][32]byte
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		id, err := uuid.Parse(root.Shards[catalogRelations[i].name])
		if err != nil {
			return nil, fmt.Errorf("catalog relation %s: %v", catalogRelations[i].name, err)
		}
		committed[i] = loadDiskShard(db, relation, id)
		_, rows[i] = catalogShardRecords(committed[i], catalogRelationColumns(relation))
		for _, row := range rows[i] {
			if scm.String(row[0]) != db.Name {
				return nil, fmt.Errorf("catalog relation %s holds rows of database %s", catalogRelations[i].name, scm.String(row[0]))
			}
		}
		fingerprints[i] = catalogFingerprint(relation, rows[i], nil)
	}
	document = catalogDocumentFromRows(db.Name, &rows)
	db.catalogMu.Lock()
	db.catalog.committed = committed
	db.catalog.committedFP = fingerprints
	db.catalogMu.Unlock()
	return document, nil
}

// commitCatalog writes every relation whose rows differ from the committed
// generation and returns the root naming the new generation. The caller holds
// db.catalogMu and keeps it until the root has its place in the commit queue,
// so roots reach disk in generation order.
func (db *database) commitCatalog(rows *catalogRows, durable bool) (*catalogRoot, uint64) {
	state := &db.catalog
	state.generation++
	root := &catalogRoot{Version: catalogFormatVersion, Shards: make(map[string]string, catalogRelationCount)}
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		fingerprint := catalogFingerprint(relation, rows[i], nil)
		if state.committed[i] == nil || fingerprint != state.committedFP[i] {
			shard := buildDiskShard(db, relation, rows[i], durable)
			if previous := state.committed[i]; previous != nil {
				state.retired = append(state.retired, retiredCatalogShard{shard: previous, generation: state.generation})
			}
			state.committed[i] = shard
			state.committedFP[i] = fingerprint
		}
		root.Shards[catalogRelations[i].name] = state.committed[i].uuid.String()
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
			drained := make(chan struct{})
			close(drained) // disk shards are never published for scans
			removable = append(removable, replacedShardGeneration{shard: retired.shard, drained: drained})
		} else {
			kept = append(kept, retired)
		}
	}
	db.catalog.retired = kept
	db.catalogMu.Unlock()
	cleanupReplacedShardGenerations(removable, "system_schema of "+db.Name)
}

// applyLiveCatalog brings db's live partition to the rows of snapshot by DML,
// firing the triggers of planner caches over system_schema. A snapshot older
// than the applied one is ignored. When another goroutine (or a trigger of
// this one) holds the live partition, the catalog is marked stale instead.
func (db *database) applyLiveCatalog(rows *catalogRows, handles *[catalogRelationCount][]scm.Scmer, snapshot uint64) {
	if !db.catalogLiveMu.TryLock() {
		db.catalogStale.Store(true)
		return
	}
	defer db.catalogLiveMu.Unlock()
	state := &db.catalog
	if snapshot < state.liveSnapshot {
		return
	}
	state.liveSnapshot = snapshot
	var created [catalogRelationCount]*storageShard
	attach := false
	for i := range catalogRelations {
		if state.live[i] == nil {
			shard := NewShard(systemSchemaTables[i])
			shard.catalogHome = db
			state.live[i], created[i] = shard, shard
			attach = true
		}
	}
	if attach {
		publishCatalogShards(db.Name, &created)
	}
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		var relationHandles []scm.Scmer
		if handles != nil {
			relationHandles = handles[i]
		}
		fingerprint := catalogFingerprint(relation, rows[i], relationHandles)
		if !attach && fingerprint == state.liveFP[i] {
			continue
		}
		db.replaceLiveRows(relation, rows[i], relationHandles)
		state.liveFP[i] = fingerprint
	}
}

// replaceLiveRows deletes the live rows that are not wanted and inserts the
// missing ones. Unchanged rows keep their record ids.
func (db *database) replaceLiveRows(relation catalogRelationID, rows [][]scm.Scmer, handles []scm.Scmer) {
	shard := db.catalog.live[relation]
	fields := catalogRelations[relation].columns()
	columns := catalogRelationColumns(relation)
	if catalogRelations[relation].handle {
		columns = append(columns, "handle")
	}
	wanted := make(map[string]int, len(rows))
	for i, row := range rows {
		handle := scm.NewNil()
		if handles != nil {
			handle = handles[i]
		}
		wanted[catalogRowKey(fields, row, handle)]++
	}
	recids, existing := catalogShardRecords(shard, columns)
	for i, row := range existing {
		handle := scm.NewNil()
		if len(row) > len(fields) {
			handle = row[len(fields)]
		}
		key := catalogRowKey(fields, row, handle)
		if wanted[key] > 0 {
			wanted[key]--
			continue
		}
		shard.UpdateFunction(recids[i], true, false, nil, nil)()
	}
	var inserts [][]scm.Scmer
	for i, row := range rows {
		handle := scm.NewNil()
		if handles != nil {
			handle = handles[i]
		}
		key := catalogRowKey(fields, row, handle)
		if wanted[key] == 0 {
			continue
		}
		wanted[key]--
		if catalogRelations[relation].handle {
			row = append(append([]scm.Scmer(nil), row...), handle)
		}
		inserts = append(inserts, row)
	}
	if len(inserts) != 0 {
		shard.Insert(columns, inserts, false, nil, nil, false, nil, nil)
	}
	// Each schema change leaves deleted rows behind; compact in RAM.
	if stats := shard.statsSnapshot(); stats.deletions > 64 && int64(stats.deletions) > stats.rowCount() {
		compacted := shard.rebuild(false)
		if compacted != shard {
			db.catalog.live[relation] = compacted
			var publish [catalogRelationCount]*storageShard
			publish[relation] = compacted
			publishCatalogShards(db.Name, &publish)
		}
	}
}

// commitSchemaLocked writes the declarations and their system_schema rows.
// The caller owns schemalock; release gives up that ownership. The catalog
// generation lock keeps the commit order of the schema snapshots.
func (db *database) commitSchemaLocked(durable bool, release func()) {
	released := false
	defer func() {
		if !released {
			release()
		}
	}()
	// DEPRECATED: the legacy declaration document stays in schema.json during
	// the transition to system_schema. It is the backup older binaries read;
	// readers of this version prefer the catalog root written next to it.
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	// Buffered mutations after this snapshot must leave the catalog dirty.
	// They need the exclusive schema lock and therefore cannot race this store.
	db.schemaDirty.Store(false)
	db.catalogStale.Store(false)
	defer func() {
		if r := recover(); r != nil {
			db.schemaDirty.Store(true)
			db.catalogStale.Store(true)
			panic(r)
		}
	}()
	rows := catalogRowsFromDocument(jsonbytes)
	handles := catalogHandles(db, &rows)
	snapshot := db.catalogSnapshot.Add(1)
	systemSchema()
	db.catalogMu.Lock()
	released = true
	release()
	var root *catalogRoot
	var generation uint64
	func() {
		defer func() {
			if r := recover(); r != nil {
				db.catalogMu.Unlock()
				panic(r)
			}
		}()
		root, generation = db.commitCatalog(&rows, durable)
		jsonbytes = attachCatalogRoot(jsonbytes, root)
	}()
	db.commitSchemaSnapshot(jsonbytes, durable, db.catalogMu.Unlock)
	db.catalogCommitted(generation)
	db.applyLiveCatalog(&rows, &handles, snapshot)
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
	db.catalogLiveMu.Lock()
	defer db.catalogLiveMu.Unlock()
	if db.catalog.live[0] == nil {
		return
	}
	for i := range catalogRelations {
		db.replaceLiveRows(catalogRelationID(i), nil, nil)
	}
	db.catalog.live = [catalogRelationCount]*storageShard{}
	publishCatalogShards(db.Name, nil)
}

// resetCatalogPersistence follows a backend switch: the next save writes
// every catalog shard to the new backend, the old one goes with its storage.
func (db *database) resetCatalogPersistence() {
	db.catalogMu.Lock()
	defer db.catalogMu.Unlock()
	db.catalog.committed = [catalogRelationCount]*storageShard{}
	db.catalog.retired = nil
}

// catalogShardIDs lists the catalog shard files GC must keep for db: the
// committed generation and every retired one whose removal is pending.
func (db *database) catalogShardIDs() []string {
	db.catalogMu.Lock()
	defer db.catalogMu.Unlock()
	var ids []string
	for _, shard := range db.catalog.committed {
		if shard != nil {
			ids = append(ids, shard.uuid.String())
		}
	}
	for _, retired := range db.catalog.retired {
		ids = append(ids, retired.shard.uuid.String())
	}
	return ids
}

// refreshSystemSchema makes system_schema current before it is read: every
// database is loaded and buffered declarations reach the live rows. A
// database whose schema is being changed right now keeps its last rows.
func refreshSystemSchema() {
	for _, db := range databases.GetAll() {
		db.GetRead(nil)
		db.refreshCatalog()
	}
}

func (db *database) refreshCatalog() {
	if !db.catalogStale.Load() || db.srState == COLD {
		return
	}
	if !db.schemalock.TryRLock() {
		return
	}
	db.catalogStale.Store(false)
	document, err := json.Marshal(db)
	if err != nil {
		db.catalogStale.Store(true)
		db.schemalock.RUnlock()
		panic(err)
	}
	rows := catalogRowsFromDocument(document)
	handles := catalogHandles(db, &rows)
	snapshot := db.catalogSnapshot.Add(1)
	db.schemalock.RUnlock()
	db.applyLiveCatalog(&rows, &handles, snapshot)
}

func declareCatalogBuiltins(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{
		Name: "column_type_display",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			var dimensions []int
			var labels []string
			if a[1].IsSlice() {
				for _, dimension := range a[1].Slice() {
					dimensions = append(dimensions, int(scm.ToInt(dimension)))
				}
			}
			if a[2].IsSlice() {
				for _, label := range a[2].Slice() {
					labels = append(labels, scm.String(label))
				}
			}
			return scm.NewString(columnTypeDisplay(scm.String(a[0]), dimensions, labels))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "renders a column type as SHOW COLUMNS does, e.g. DECIMAL(12,2) or ENUM('a','b')",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "type"},
				{Kind: "list", Label: "dimensions"},
				{Kind: "list", Label: "enum_values"},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(en, &scm.Declaration{
		Name: "table_statistics",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if a[0].IsNil() {
				return scm.NewNil()
			}
			t := TableFromScmer(a[0])
			stats := t.statistics()
			return scm.NewSlice(append(showBuildMemory(t.residentMemory()).Slice(),
				scm.NewString("row_count"), scm.NewInt(stats.rowCount)))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "runtime statistics of a table handle, such as system_schema.tables.handle: row_count and the resident size_bytes breakdown",
			Params: []*scm.TypeDescriptor{{Kind: "table", Label: "table"}},
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
}
