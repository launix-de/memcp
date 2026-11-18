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
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
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
	Version int               `json:"version"`
	Shards  map[string]string `json:"shards"`
}

// databaseCatalog owns authoritative definition shards and their last durable
// checkpoint. Go tables/columns are local runtime projections of these rows.
type databaseCatalog struct {
	// Guarded by database.catalogMu: the shards named by the newest root
	// handed to the schema.json commit queue, and files awaiting removal.
	committed   [catalogRelationCount]*storageShard
	committedFP [catalogRelationCount][32]byte
	generation  uint64
	retired     []retiredCatalogShard

	// Guarded by database.schemalock: the queryable partition. It changes
	// by DML, so triggers of planner caches over system_schema observe it.
	live        [catalogRelationCount]*storageShard
	liveFP      [catalogRelationCount][32]byte
	liveVersion uint64
}

type retiredCatalogShard struct {
	shard      *storageShard
	generation uint64
	drained    <-chan struct{}
}

var systemSchemaOnce sync.Once
var systemSchemaDB *database

// Serializes orphan collection with publication of cross-database catalog files.
var catalogPersistenceMu sync.RWMutex

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
			t.PersistencyMode = Sloppy
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
	rebuilt := shard.rebuild(true)
	rebuilt.catalogDurable = false
	return rebuilt
}

// loadDiskShard opens a persisted catalog shard of db.
func loadDiskShard(db *database, relation catalogRelationID, id uuid.UUID) *storageShard {
	shard := &storageShard{t: systemSchemaTables[relation], uuid: id, catalogHome: db}
	shard.columns = make(map[string]ColumnStorage)
	shard.deltaColumns = make(map[string]int)
	shard.deletions.Reset()
	shard.setState(COLD)
	return shard
}

// catalogShardRecords reads the visible rows of a catalog shard with their
// record ids, in recid order.
func catalogShardRecords(shard *storageShard, columns []string) (recids []uint32, rows [][]scm.Scmer) {
	access := shard.GetRead(nil, false)
	defer access.Release()
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
// read back from compressed storage compare equal to declaration edits.
func catalogRowKey(fields []catalogField, row []scm.Scmer) string {
	var b strings.Builder
	for i, field := range fields {
		raw := catalogRaw(field.kind, row[i])
		fmt.Fprintf(&b, "%d:", len(raw))
		b.Write(raw)
	}
	return b.String()
}

func catalogFingerprint(relation catalogRelationID, rows [][]scm.Scmer) [32]byte {
	fields := catalogRelations[relation].columns()
	hash := sha256.New()
	for _, row := range rows {
		key := catalogRowKey(fields, row)
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
		home := db
		if root.Version >= 2 {
			home = catalogBackendDatabase()
		}
		committed[i] = db.catalog.live[i]
		if committed[i] == nil || committed[i].uuid != id || committed[i].home() != home {
			committed[i] = loadDiskShard(home, relation, id)
		}
		_, rows[i] = catalogShardRecords(committed[i], catalogRelationColumns(relation))
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
	document = catalogDocumentFromRows(db.Name, &rows)
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
func (db *database) commitCatalog(rows *catalogRows, durable bool) (*catalogRoot, uint64) {
	state := &db.catalog
	state.generation++
	root := &catalogRoot{Version: catalogFormatVersion, Shards: make(map[string]string, catalogRelationCount)}
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		fingerprint := catalogFingerprint(relation, rows[i])
		if state.committed[i] == nil || state.committed[i].home() != catalogBackendDatabase() || fingerprint != state.committedFP[i] {
			shard := buildDiskShard(db, relation, rows[i], durable)
			if previous := state.committed[i]; previous != nil {
				drained := closedCatalogGeneration
				if topology := previous.generation.Load(); topology != nil {
					drained = topology.drained
				}
				state.retired = append(state.retired, retiredCatalogShard{shard: previous, generation: state.generation, drained: drained})
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
func (db *database) catalogRowsLocked() (rows catalogRows) {
	for i, shard := range db.catalog.live {
		if shard != nil {
			_, rows[i] = catalogShardRecords(shard, catalogRelationColumns(catalogRelationID(i)))
		}
	}
	return
}

// newCatalogShard constructs an unpublished generation without executing user
// DML, WAL or cache callbacks. Only the catalog editor can call this constructor.
func newCatalogShard(db *database, relation catalogRelationID, rows [][]scm.Scmer) *storageShard {
	shard := NewShard(systemSchemaTables[relation])
	shard.catalogHome = db
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
func (db *database) editCatalogLocked(rows *catalogRows) bool {
	tx := NewTxContext(TxACID)
	defer func() {
		if tx.State == TxActive {
			tx.Rollback()
		}
	}()
	fingerprints := db.catalog.liveFP
	changed := false
	var created [catalogRelationCount]*storageShard
	for i := range catalogRelations {
		if db.catalog.live[i] == nil {
			created[i] = newCatalogShard(db, catalogRelationID(i), nil)
			db.catalog.live[i] = created[i]
		}
	}
	publishCatalogShards(db.Name, &created)
	for i := range catalogRelations {
		relation := catalogRelationID(i)
		fp := catalogFingerprint(relation, rows[i])
		if fp == db.catalog.liveFP[i] {
			continue
		}
		changed = true
		shard := db.catalog.live[i]
		fields := catalogRelations[i].columns()
		columns := catalogRelationColumns(relation)
		wanted := make(map[string]int, len(rows[i]))
		for _, row := range rows[i] {
			wanted[catalogRowKey(fields, row)]++
		}
		recids, existing := catalogShardRecords(shard, columns)
		for j, row := range existing {
			key := catalogRowKey(fields, row)
			if wanted[key] > 0 {
				wanted[key]--
				continue
			}
			shard.updateFunctionBatch(recids[j], true, false, nil, nil, tx, nil, true)()
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
		}
		fingerprints[i] = fp
	}
	if !changed {
		return false
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
	db.catalog.liveFP = fingerprints
	db.catalog.liveVersion++
	return true
}

// proposeCatalogTablesLocked admits explicit DDL proposals for named tables.
// Runtime objects are builders here, never an input to a save or catalog read.
// Only the named declarations are replaced; unrelated catalog rows are retained.
func (db *database) proposeCatalogTablesLocked(tables ...*table) {
	if db == systemSchemaDB {
		return
	}
	rows := db.catalogRowsLocked()
	for _, t := range tables {
		if t == nil || t.schema != db {
			continue
		}
		encoded, err := json.Marshal(t)
		if err != nil {
			panic(err)
		}
		doc := catalogDocument{Name: db.Name, Tables: map[string]json.RawMessage{t.Name: encoded}}
		proposal := catalogRowsFromDocument(marshalCatalogValue(doc))
		for i := catalogTables; i < catalogProcedures; i++ {
			kept := make([][]scm.Scmer, 0, len(rows[i])+len(proposal[i]))
			for _, row := range rows[i] {
				name := row[1].String()
				if i == catalogTables {
					name = catalogTableName(row)
				}
				if name != t.Name {
					kept = append(kept, row)
				}
			}
			rows[i] = append(kept, proposal[i]...)
		}
	}
	// A new compiled callback can share an unchanged persisted declaration.
	// Keep that runtime state without decoding the same canonical rows again.
	if !db.editCatalogLocked(&rows) {
		return
	}
	for _, t := range tables {
		if t != nil && t.schema == db {
			db.materializeCatalogTableLocked(t, &rows)
		}
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

func (db *database) dropCatalogTableLocked(name string) {
	rows := db.catalogRowsLocked()
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
	db.editCatalogLocked(&rows)
}

func (db *database) editCatalogProcedureLocked(name string, proposal *ProcedureDescription) *ProcedureDescription {
	rows := db.catalogRowsLocked()
	definitions := make([]map[string]json.RawMessage, 0, len(rows[catalogProcedures])+1)
	for _, row := range rows[catalogProcedures] {
		record := catalogRecord(catalogProcedures, row)
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
	db.editCatalogLocked(&rows)
	if proposal == nil {
		return nil
	}
	// Materialize the definition from the admitted row, retaining validated
	// process-local compiled code on this equivalent immutable declaration.
	for _, row := range rows[catalogProcedures] {
		var definition ProcedureDescription
		if err := json.Unmarshal(marshalCatalogValue(catalogRecord(catalogProcedures, row)), &definition); err != nil {
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
func (db *database) checkpointCatalogLocked() {
	rows := db.catalogRowsLocked()
	for _, row := range rows[catalogTables] {
		t := db.tables.Get(catalogTableName(row))
		if t == nil {
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
	db.editCatalogLocked(&rows)
}

// materializeCatalogTableLocked publishes a local runtime projection of the
// admitted rows. Matching column and trigger objects retain their compiled code
// and cache lifetimes. Definition values themselves are decoded from the catalog.
func (db *database) materializeCatalogTableLocked(t *table, rows *catalogRows) {
	var selected catalogRows
	for i := catalogTables; i < catalogProcedures; i++ {
		for _, row := range rows[i] {
			owner := row[1].String()
			if i == catalogTables {
				owner = catalogTableName(row)
			}
			if owner == t.Name {
				selected[i] = append(selected[i], row)
			}
		}
	}
	var document catalogDocument
	if err := json.Unmarshal(catalogDocumentFromRows(db.Name, &selected), &document); err != nil {
		panic(err)
	}
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
	t.Columns = definition.Columns
	t.Unique, t.Foreign = definition.Unique, definition.Foreign
	t.DroppedColumns = definition.DroppedColumns
	if t.PersistencyMode != definition.PersistencyMode {
		t.PersistencyMode = definition.PersistencyMode
	}
	t.Collation, t.Charset, t.Comment = definition.Collation, definition.Charset, definition.Comment
	if !bytes.Equal(marshalCatalogValue(t.OnInit.Load()), marshalCatalogValue(definition.OnInit.Load())) {
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
		if candidate.cacheTarget == nil && candidate.Acquire == nil {
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
func (db *database) commitSchemaLocked(durable bool, release func()) {
	released := false
	defer func() {
		if !released {
			release()
		}
	}()
	rows := db.catalogRowsLocked()
	version := db.catalog.liveVersion
	jsonbytes := catalogDocumentFromRows(db.Name, &rows)
	catalogPersistenceMu.RLock()
	defer catalogPersistenceMu.RUnlock()
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
		root, generation = db.commitCatalog(&rows, durable)
		committed = db.catalog.committed
		jsonbytes = attachCatalogRoot(jsonbytes, root)
	}()
	db.commitSchemaSnapshot(jsonbytes, durable, db.catalogMu.Unlock)
	db.schemalock.Lock()
	if db.catalog.liveVersion == version {
		db.catalog.live = committed
		publishCatalogShards(db.Name, &committed)
		db.schemaDirty.Store(false)
	}
	db.schemalock.Unlock()
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
	var ids []string
	owners := []*database{db}
	if db.Name == "system" {
		owners = databases.GetAll()
	}
	for _, owner := range owners {
		// System-backend deletion needs every tenant anchor as a proof. Opening
		// those roots creates only shard stubs, never Go runtime definitions.
		if db.Name == "system" && owner.srState == COLD {
			owner.bootstrapCatalog(nil)
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
	db.editCatalogLocked(&rows)
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
			for i, relation := range catalogRelations {
				id, err := uuid.Parse(root.Shards[relation.name])
				if err != nil {
					panic(err)
				}
				home := db
				if root.Version >= 2 {
					home = catalogBackendDatabase()
				}
				shard := loadDiskShard(home, catalogRelationID(i), id)
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
		db.editCatalogLocked(&rows)
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
func (t *table) scanTopology(access scanAccess) *tableShardTopology {
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
		if topology.shards[i] != nil {
			continue
		}
		if db := databases.Get(dimension.Pivots[i].String()); db != nil {
			db.bootstrapCatalog(nil)
		}
	}
	return t.activeTopology()
}

// Partition traffic is advisory runtime state. Equal declaration projections
// retain their native computed-column lifetimes and compiled value recipes.
func catalogColumnDefinitionEqual(a, b *column) bool {
	var left, right map[string]json.RawMessage
	if err := json.Unmarshal(marshalCatalogValue(a), &left); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(marshalCatalogValue(b), &right); err != nil {
		panic(err)
	}
	delete(left, "PartitioningScore")
	delete(right, "PartitioningScore")
	return bytes.Equal(marshalCatalogValue(left), marshalCatalogValue(right))
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
