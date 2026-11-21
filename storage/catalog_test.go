/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"github.com/launix-de/memcp/storage/mosi"
)

// normalizedSchemaJSON decodes a schema document through the legacy loader
// and encodes it again. Empty and absent collections compare equal.
func normalizedSchemaJSON(t *testing.T, document []byte) any {
	t.Helper()
	db := newDatabase()
	if err := json.Unmarshal(document, db); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	encoded, err := json.Marshal(db)
	if err != nil {
		t.Fatalf("encode schema: %v", err)
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatal(err)
	}
	var normalize func(any) any
	normalize = func(value any) any {
		switch typed := value.(type) {
		case []any:
			if len(typed) == 0 {
				return nil
			}
			for i := range typed {
				typed[i] = normalize(typed[i])
			}
		case map[string]any:
			for key, item := range typed {
				typed[key] = normalize(item)
			}
		}
		return value
	}
	return normalize(value)
}

const catalogRoundTripDocument = `{
  "name": "shop",
  "tables": {
    "orders": {
      "Name": "orders",
      "Columns": [
        {"Name": "id", "Typ": "int", "Typdimensions": null, "PartitioningScore": 4, "AutoIncrement": true, "Default": null, "DefaultExpression": "", "OnUpdate": null, "AllowNull": false, "IsTemp": false, "Collation": "", "Comment": "", "OrcFilterFn": null, "OrcMapReduceFn": null, "OrcReduceInit": null},
        {"Name": "state", "Typ": "enum", "EnumValues": ["new", "it's done"], "Typdimensions": null, "PartitioningScore": 0, "AutoIncrement": false, "Default": "new", "DefaultExpression": "", "default_name": "DF_state", "OnUpdate": null, "AllowNull": true, "IsTemp": false, "Collation": "utf8mb4_bin", "Comment": "lifecycle", "OrcFilterFn": null, "OrcMapReduceFn": null, "OrcReduceInit": null},
        {"Name": "total", "Typ": "decimal", "Typdimensions": [12, 2], "PartitioningScore": 0, "AutoIncrement": false, "Default": 1.5, "DefaultExpression": "", "OnUpdate": null, "AllowNull": true, "IsTemp": false, "Collation": "", "Comment": "", "initial_value": 0, "initial_values": {"00000000-0000-0000-0000-000000000001": {"Rows": 3}}, "OrcFilterFn": null, "OrcMapReduceFn": null, "OrcReduceInit": null},
        {"Name": "running", "Typ": "int", "Typdimensions": null, "ComputorInputCols": ["total"], "ComputorDependencies": ["shop", "orders"], "PartitioningScore": 0, "AutoIncrement": false, "Default": null, "DefaultExpression": "", "OnUpdate": null, "AllowNull": true, "IsTemp": true, "Collation": "", "Comment": "", "OrcSortCols": ["id"], "OrcSortDirs": [true], "OrcPartitionCount": 0, "OrcMapCols": ["total"], "OrcFilterFn": null, "OrcMapReduceFn": ["lambda", ["acc", "set", "total"], ["set", ["+", "acc", "total"]]], "OrcReduceInit": 0}
      ],
      "dropped_columns": {"legacy": true},
      "Unique": [{"Id": "PRIMARY", "Cols": ["id"]}, {"Id": "state_total", "Cols": ["state", "total"], "constraint_name": "UQ_state", "nulls_not_distinct": true}],
      "Foreign": [{"Id": "fk_customer", "Tbl1": "orders", "Cols1": ["id"], "Tbl2": "customers", "Cols2": ["order_id"], "Updatemode": 1, "Deletemode": 2}],
      "Triggers": [{"name": "audit", "timing": "after_insert", "func": ["lambda", ["OLD", "NEW"], ["print", "NEW"]], "source": "INSERT INTO log VALUES (NEW.id)", "language": "sql", "priority": 3, "async": true}],
      "PersistencyMode": "logged",
      "oninit": ["lambda", [], true],
      "Auto_increment": 41,
      "Collation": "utf8mb4_general_ci",
      "Charset": "utf8mb4",
      "Comment": "all orders",
      "planner_row_estimate": 1200,
      "ShardMode": "freeshard",
      "Shards": ["00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"],
      "PShards": null,
      "PDimensions": null,
      "shard_forwards": {"00000000-0000-0000-0000-000000000003": "00000000-0000-0000-0000-000000000001"}
    },
    "customers": {
      "Name": "customers",
      "Columns": [
        {"Name": "order_id", "Typ": "int", "Typdimensions": null, "PartitioningScore": 0, "AutoIncrement": false, "Default": null, "DefaultExpression": "CURRENT_TIMESTAMP", "OnUpdate": "now", "AllowNull": true, "IsTemp": false, "Collation": "", "Comment": "", "OrcFilterFn": null, "OrcMapReduceFn": null, "OrcReduceInit": null}
      ],
      "dropped_columns": {},
      "Unique": null,
      "Foreign": [{"Id": "fk_customer", "Tbl1": "orders", "Cols1": ["id"], "Tbl2": "customers", "Cols2": ["order_id"], "Updatemode": 1, "Deletemode": 2}],
      "Triggers": null,
      "PersistencyMode": "safe",
      "Auto_increment": 0,
      "Collation": "",
      "Charset": "",
      "Comment": "",
      "planner_row_estimate": 0,
      "ShardMode": "partition",
      "Shards": null,
      "PShards": ["00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000005"],
      "PDimensions": [{"Column": "order_id", "NumPartitions": 2, "Pivots": [10]}]
    }
  },
  "procedures": [
    {"name": "refill", "source": "SELECT 1", "language": "sql", "parameters": [["amount", "int"]]},
    {"name": "audit", "schema_only": true, "source": "(lambda () 1)", "language": "scm", "parameters": null}
  ]
}`

func TestCatalogNativeProposalsMatchLegacyDefinitions(t *testing.T) {
	db := newDatabase()
	if err := json.Unmarshal([]byte(catalogRoundTripDocument), db); err != nil {
		t.Fatal(err)
	}
	legacy := catalogRowsFromDocument(marshalCatalogValue(db))
	var native catalogRows
	for _, table := range db.tables.GetAll() {
		table.schema = db
		proposal := catalogRowsFromTable(table)
		for i := catalogTables; i < catalogProcedures; i++ {
			native[i] = append(native[i], proposal[i]...)
		}
	}
	for i := catalogTables; i < catalogProcedures; i++ {
		keys := func(rows [][]scm.Scmer) []string {
			result := make([]string, len(rows))
			for j, row := range rows {
				result[j] = catalogRowKey(catalogRelations[i].columns(), row)
			}
			sort.Strings(result)
			return result
		}
		if !reflect.DeepEqual(keys(legacy[i]), keys(native[i])) {
			t.Fatalf("native %s proposals differ from legacy definitions", catalogRelations[i].name)
		}
	}
}

func TestCatalogRowsRoundTripLegacyDocument(t *testing.T) {
	rows := catalogRowsFromDocument([]byte(catalogRoundTripDocument))
	counts := map[catalogRelationID]int{
		catalogDatabases: 1, catalogTables: 2, catalogColumns: 5, catalogKeys: 2, catalogKeyColumns: 3,
		catalogForeignKeys: 2, catalogForeignKeyColumns: 2, catalogTriggers: 1, catalogProcedures: 2,
	}
	for relation, want := range counts {
		if got := len(rows[relation]); got != want {
			t.Fatalf("%s has %d rows, want %d", catalogRelations[relation].name, got, want)
		}
	}
	// Reverse the row order: reconstruction places children by ordinal.
	for relation := range rows {
		for i, j := 0, len(rows[relation])-1; i < j; i, j = i+1, j-1 {
			rows[relation][i], rows[relation][j] = rows[relation][j], rows[relation][i]
		}
	}
	restored := catalogDocumentFromRows("shop", &rows)
	want := normalizedSchemaJSON(t, []byte(catalogRoundTripDocument))
	got := normalizedSchemaJSON(t, restored)
	if !reflect.DeepEqual(want, got) {
		wantJSON, _ := json.MarshalIndent(want, "", " ")
		gotJSON, _ := json.MarshalIndent(got, "", " ")
		t.Fatalf("catalog round trip changed the schema\nwant %s\ngot  %s", wantJSON, gotJSON)
	}
}

func TestCatalogRelationsCoverEveryPersistedField(t *testing.T) {
	// Every JSON key of the legacy records is a catalog column or a child
	// relation. A new persisted field automatically becomes a column.
	children := map[string]bool{"Columns": true, "Unique": true, "Foreign": true, "Triggers": true, "Cols": true, "Cols1": true, "Cols2": true}
	for relation, record := range map[catalogRelationID]any{
		catalogTables: persistedTable{}, catalogColumns: persistedColumn{}, catalogKeys: uniqueKey{},
		catalogForeignKeys: foreignKey{}, catalogTriggers: TriggerDescription{}, catalogProcedures: ProcedureDescription{},
	} {
		columns := map[string]bool{}
		for _, field := range catalogRelations[relation].fields {
			if columns[field.column] {
				t.Fatalf("%s maps two keys to column %s", catalogRelations[relation].name, field.column)
			}
			columns[field.column] = true
		}
		for _, field := range catalogRecordFields(reflect.TypeOf(record), nil) {
			if !children[field.key] && !columns[field.column] {
				t.Fatalf("%s lacks persisted field %s", catalogRelations[relation].name, field.key)
			}
		}
	}
	want := map[string]string{"PersistencyMode": "persistency_mode", "Auto_increment": "auto_increment", "PShards": "p_shards", "Cols1": "cols1", "ComputorInputCols": "computor_input_cols"}
	for key, column := range want {
		if got := catalogColumnName(key); got != column {
			t.Fatalf("catalogColumnName(%q) = %q, want %q", key, got, column)
		}
	}
}

func TestCatalogPersistsAndTakesPrecedenceOverLegacySchema(t *testing.T) {
	previousBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previousBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schema = "catalog-persistence-test"
	CreateDatabase(schema, false)
	defer DropDatabase(schema, true)
	parent, _ := CreateTable(schema, "parent", Safe, false)
	parent.CreateColumn("id", "int", nil, []scm.Scmer{scm.NewString("primary"), scm.NewBool(true)})
	parent.CreateColumn("label", "varchar", []int{20}, []scm.Scmer{scm.NewString("comment"), scm.NewString("shown name")})
	child, _ := CreateTable(schema, "child", Safe, false)
	child.CreateColumn("id", "int", nil, nil)
	child.CreateColumn("parent_id", "int", nil, nil)
	scm.Eval(scm.Read("catalog test", `(createforeignkey (table "`+schema+`" false "child" false) "fk_child_parent" '("parent_id") (table "`+schema+`" false "parent" false) '("id") "cascade" "restrict")`, true), &scm.Globalenv)

	db := GetDatabase(schema)
	stored := db.persistence.ReadSchema()
	root := readCatalogRoot(stored)
	if root == nil || len(root.Shards) != int(catalogRelationCount) {
		t.Fatalf("schema.json has no complete catalog root: %s", stored)
	}
	if !strings.Contains(string(stored), `"tables"`) {
		t.Fatal("schema.json lost the legacy declarations")
	}

	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = schema, db.persistence, COLD
	restored.ensureLoaded()
	original, _ := json.Marshal(db)
	reloaded, _ := json.Marshal(restored)
	if !reflect.DeepEqual(normalizedSchemaJSON(t, original), normalizedSchemaJSON(t, reloaded)) {
		t.Fatalf("catalog reload differs\noriginal %s\nreloaded %s", original, reloaded)
	}

	// The catalog takes precedence: a legacy part that disagrees is ignored.
	var document map[string]json.RawMessage
	if err := json.Unmarshal(stored, &document); err != nil {
		t.Fatal(err)
	}
	document["tables"] = json.RawMessage(`{}`)
	tampered, _ := json.Marshal(document)
	db.persistence.WriteSchema(tampered)
	preferred := newDatabase()
	preferred.Name, preferred.persistence, preferred.srState = schema, db.persistence, COLD
	preferred.ensureLoaded()
	if preferred.tables.Get("parent") == nil || preferred.tables.Get("child") == nil {
		t.Fatal("the legacy part took precedence over the catalog")
	}

	// Without a root the legacy part is authoritative.
	db.persistence.WriteSchema(stripCatalogRoot(stored))
	legacy := newDatabase()
	legacy.Name, legacy.persistence, legacy.srState = schema, db.persistence, COLD
	legacy.ensureLoaded()
	if legacy.tables.Get("parent") == nil {
		t.Fatal("legacy schema did not load")
	}
	db.persistence.WriteSchema(stored)
}

func TestCatalogReplicaAdmissionChecksNativeDeclarations(t *testing.T) {
	const schema = "catalog-client-admission"
	const language = "catalog-client-source-test"
	compiled := false
	registerTriggerLanguage(language, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		compiled = true
		return scm.NewNil()
	}), true)
	valid := []byte(`{"name":"catalog-client-admission","tables":{"items":{"Name":"items","Columns":[{"Name":"id","Typ":"INT","Default":7}],"Unique":[{"Name":"primary","Cols":["id"]}],"Foreign":[{"Cols1":["id"],"Cols2":["id"]}],"Triggers":[{"name":"source","timing":"before_insert","source":"SET NEW.id = 7","language":"catalog-client-source-test","schema_only":true}]}},"procedures":[{"name":"source_proc","source":"SELECT 7","language":"catalog-client-source-test","schema_only":true}]}`)
	rows := catalogRowsFromDocument(valid)
	for i := range catalogRelations {
		if len(rows[i]) == 0 {
			t.Fatalf("fixture lacks relation %s", catalogRelations[i].name)
		}
		if err := validateReplicaCatalogRows(schema, catalogRelationID(i), rows[i]); err != nil {
			t.Fatalf("valid %s: %v", catalogRelations[i].name, err)
		}
	}
	for name, fragment := range map[string]string{
		"initializer": `"oninit":[{"symbol":"lambda"},[],true]`,
		"column-code": `"Columns":[{"Name":"id","Typ":"INT","Default":{"symbol":"shutdown"}}]`,
		"temp-code":   `"Columns":[{"Name":"computed","Typ":"INT","IsTemp":true,"Computor":[{"symbol":"lambda"},[],true]}]`,
		"bad-name":    `"Columns":[{"Name":"../escape","Typ":"INT"}]`,
		"trigger":     `"Triggers":[{"name":"source","timing":"before_insert","source":"SET NEW.id = 7","language":"catalog-client-source-test"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			candidate := catalogRowsFromDocument([]byte(`{"name":"catalog-client-admission","tables":{"items":{"Name":"items",` + fragment + `}}}`))
			rejected := false
			for i := range catalogRelations {
				if err := validateReplicaCatalogRows(schema, catalogRelationID(i), candidate[i]); err != nil {
					rejected = true
				}
			}
			if !rejected {
				t.Fatal("unsafe native declaration was admitted")
			}
		})
	}
	columns := append([]scm.Scmer(nil), rows[catalogColumns][0]...)
	columns[2] = scm.NewInt(1 << 40)
	if err := validateReplicaCatalogRows(schema, catalogColumns, [][]scm.Scmer{columns}); err == nil {
		t.Fatal("unbounded native ordinal was admitted")
	}
	columns[2], columns[0] = scm.NewInt(0), scm.NewString("another-tenant")
	if err := validateReplicaCatalogRows(schema, catalogColumns, [][]scm.Scmer{columns}); err == nil {
		t.Fatal("foreign tenant catalog row was admitted")
	}
	left := append([]scm.Scmer(nil), rows[catalogColumns][0]...)
	right := append([]scm.Scmer(nil), rows[catalogColumns][0]...)
	if err := validateReplicaCatalogRows(schema, catalogColumns, [][]scm.Scmer{left, right}); err == nil {
		t.Fatal("duplicate native ordinal was admitted")
	}
	left[1], left[2], right[1] = scm.NewString("left"), scm.NewInt(1), scm.NewString("right")
	if err := validateReplicaCatalogRows(schema, catalogColumns, [][]scm.Scmer{left, right}); err == nil {
		t.Fatal("native ordinals with a gap in one table were admitted")
	}
	if compiled {
		t.Fatal("client declaration admission executed supplied code")
	}
}

func TestCatalogReplicaImportValidatesBeforePublication(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-client-import", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	source := db.catalog.live[catalogColumns]
	resource := source.replicationResource()
	cell := r.declareShard(source, "client", false)
	previous := cell.current.Load()
	proposal := catalogRowsFromTable(tbl)
	for index, field := range catalogRelations[catalogColumns].columns() {
		if field.key == "Name" {
			proposal[catalogColumns][0][index] = scm.NewString("../escape")
		}
	}
	invalid := buildDiskShard(db, catalogColumns, proposal[catalogColumns], true)
	transfer := &replicaImport{untrusted: true, header: mosiPacket{Resource: resource, Content: invalid.uuid.String()}, engine: invalid.home().persistence, home: invalid.home()}
	if err := r.finishImport("client", transfer); err == nil {
		t.Fatal("unsafe native catalog import was published")
	}
	if cell.current.Load() != previous || r.canonicalResource(resource) != resource {
		t.Fatal("rejected catalog import changed native payload or UUID authority")
	}
	validRows := catalogRowsFromTable(tbl)
	incoming := buildDiskShard(db, catalogColumns, validRows[catalogColumns], true)
	transfer.header.Content = incoming.uuid.String()
	home := incoming.home()
	physical := home.persistence
	barrier := &catalogNamespaceBarrier{PersistenceEngine: physical, before: func() { panic("namespace not durable") }}
	home.persistence = barrier
	defer func() { home.persistence = physical }()
	func() {
		defer func() {
			if failure := recover(); failure != "namespace not durable" {
				t.Errorf("catalog import did not propagate namespace durability failure: %v", failure)
			}
		}()
		if err := r.finishImport("client", transfer); err != nil {
			t.Fatal(err)
		}
	}()
	if cell.current.Load() != previous || r.canonicalResource(resource) != resource {
		t.Fatal("namespace durability failure published catalog payload or UUID authority")
	}
	barrier.before = func() {}
	if err := r.finishImport("client", transfer); err != nil {
		t.Fatal(err)
	}
	published := cell.resolved().current.Load()
	if published == nil || published.uuid != incoming.uuid || published.home() != catalogBackendDatabase() || published.catalogPrivate {
		t.Fatal("validated import did not publish native system-backend content")
	}
}

func TestCatalogReplicaTransferAdmitsSuccessorBeforePublishingRoutes(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "catalog-successor-admission", 0)
			db := tbl.schema
			r := replicaRecoveryFixture()
			source := db.catalog.live[catalogColumns]
			historical := source.replicationResource()
			cell := r.declareShard(source, "client", false)
			r.updateRecord(historical, func(record *replicaRecord) bool {
				record.Owner, record.Epoch = "client", 7
				return true
			})
			rows := catalogRowsFromTable(tbl)
			if !valid {
				for index, field := range catalogRelations[catalogColumns].columns() {
					if field.key == "Name" {
						rows[catalogColumns][0][index] = scm.NewString("../escape")
					}
				}
			}
			incoming := buildDiskShard(db, catalogColumns, rows[catalogColumns], true)
			current := incoming.replicationResource()
			packet := mosiPacket{Kind: "begin", ID: "successor", Resource: current, Epoch: 7, Content: current.Shard,
				Forwards: []shardForward{{From: historical, To: current}}}
			r.peers = map[string]*mosiPeer{"client": {outgoing: newMosiMailbox(), done: make(chan struct{})}}
			r.steps.Store(packet.ID, &replicaStep{peer: "client", resource: current, epoch: 7, content: current.Shard, done: make(chan struct{})})
			if err := r.applyResourcePacket("client", &packet); err != nil {
				t.Fatal("current modifier's predecessor proof was rejected", err)
			}
			if r.lookupCell(current) != nil || cell.current.Load() != source {
				t.Fatal("transfer header published an unvalidated catalog route")
			}
			// Follow-up packets do not repeat forwarding declarations. Completion
			// must check the admitted header's proof again, before installing data.
			packet.Kind, packet.Forwards = "end", nil
			err := r.applyResourcePacket("client", &packet)
			if !valid {
				if err == nil || !strings.Contains(err.Error(), "invalid replicated column name") {
					t.Fatalf("unsafe declaration did not reach native admission: %v", err)
				}
				if r.lookupCell(current) != nil || cell.current.Load() != source {
					t.Fatal("rejected successor changed accepted catalog authority")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.canonicalResource(historical) != current || cell.resolved().current.Load().uuid != incoming.uuid {
				t.Fatal("admitted successor did not inherit its predecessor's authority")
			}
		})
	}
}

func TestCatalogDefinitionProposalsKeepRuntimeCachesLocal(t *testing.T) {
	for _, mode := range []string{"standalone", "master", "slave"} {
		t.Run(mode, func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "catalog-local-cache-proposals", 0)
			db := tbl.schema
			var r *replicationManager
			if mode != "standalone" {
				r = replicaRecoveryFixture()
				r.master = mode == "master"
				r.schemaShards(db, r.id, true)
			}
			previous := replication.Swap(r)
			defer replication.Store(previous)
			db.schemalock.Lock()
			before := db.catalogRowsLocked()
			db.schemalock.Unlock()
			cache := db.newTable(".grp:local", Cache)
			db.tables.Set(cache.Name, cache)
			compute := &column{Name: "local_compute", Typ: "INT", IsTemp: true, RuntimeOnly: true, Computor: scm.NewSymbol("local_runtime_helper")}
			tbl.Columns = append(tbl.Columns, compute)
			db.schemalock.Lock()
			db.proposeCatalogTablesLocked(tbl, cache)
			rows := db.catalogRowsLocked()
			db.schemalock.Unlock()
			for _, row := range rows[catalogTables] {
				if catalogTableName(row) == cache.Name {
					t.Fatal("node-local cache became a canonical replicated declaration")
				}
			}
			for relation := range rows {
				if catalogFingerprint(catalogRelationID(relation), before[relation]) != catalogFingerprint(catalogRelationID(relation), rows[relation]) {
					t.Fatalf("runtime workspace changed canonical relation %s", catalogRelations[relation].name)
				}
				if err := validateReplicaCatalogRows(db.Name, catalogRelationID(relation), rows[relation]); err != nil {
					t.Fatal("runtime cache code reached native admission", err)
				}
			}
			if tbl.Columns[len(tbl.Columns)-1] != compute || db.tables.Get(cache.Name) != cache {
				t.Fatal("canonical declaration filtering destroyed node-local compute owners")
			}
		})
	}
}

func TestCatalogCacheRebuildLeavesUnrelatedStatisticsUncheckpointed(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "catalog_cache_checkpoint_scope", 0)
	db := source.schema
	cache, _ := CreateTable(db.Name, ".grp:checkpoint_scope", Cache, false)
	db.ensureBlobTable()
	before := readCatalogRoot(db.persistence.ReadSchema())
	if before == nil {
		t.Fatal("missing initial native catalog root")
	}
	// Query planning can update advisory statistics without changing a table's
	// physical generation. Rebuilding its query cache does not own that change.
	source.PlannerRowEstimate.value.Store(12345)
	if result := RebuildTable(cache, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	after := readCatalogRoot(db.persistence.ReadSchema())
	if after == nil || !reflect.DeepEqual(before.Shards, after.Shards) {
		t.Fatal("query cache rebuild rewrote native definition partitions")
	}
	// The source's own checkpoint must still record its new statistics.
	db.save(nil, source)
	after = readCatalogRoot(db.persistence.ReadSchema())
	if after.Shards["tables"] == before.Shards["tables"] {
		t.Fatal("source checkpoint lost its advisory statistics")
	}
}

func TestCatalogReplicaTablesIntroduceDataAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-new-data-authority", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	introduced := db.newTable("introduced", Safe)
	rows := db.catalogRowsLocked()
	proposal := catalogRowsFromTable(introduced)
	rows[catalogTables] = append(rows[catalogTables], proposal[catalogTables]...)
	incoming := buildDiskShard(db, catalogTables, rows[catalogTables], true)
	resource := db.catalog.live[catalogTables].replicationResource()
	transfer := &replicaImport{untrusted: true, header: mosiPacket{Resource: resource, Content: incoming.uuid.String()}, engine: incoming.home().persistence, home: incoming.home()}
	if err := r.finishImport("client", transfer); err != nil {
		t.Fatal(err)
	}
	data := introduced.Shards[0].replicationResource()
	record := r.directory.Records[r.resourceKey(data)]
	if record.Owner != "client" || record.Epoch != 1 || r.resourceCell(data).current.Load() != nil {
		t.Fatal("native declaration did not introduce data authority independently of its payload")
	}
	if db.tables.Get(introduced.Name) != nil {
		t.Fatal("catalog publication eagerly constructed a Go table")
	}
	previous := replication.Swap(r)
	defer replication.Store(previous)
	loaded := r.findShard(data)
	if loaded == nil || loaded.uuid != introduced.Shards[0].uuid || loaded.t != db.tables.Get(introduced.Name) {
		t.Fatal("native declaration did not materialize the requested table on demand")
	}
	// A local safety copy contains declarations but no live ownership proof.
	// Restoring it must not turn its previously unknown data UUIDs into rights.
	var backupOnly *table
	func() {
		previous := replication.Swap(nil)
		defer replication.Store(previous)
		backupOnly = db.newTable("backup_only", Safe)
	}()
	if r.lookupCell(backupOnly.Shards[0].replicationResource()) != nil {
		t.Fatal("backup fixture already registered data authority")
	}
	rows = db.catalogRowsLocked()
	proposal = catalogRowsFromTable(backupOnly)
	rows[catalogTables] = append(rows[catalogTables], proposal[catalogTables]...)
	incoming = buildDiskShard(db, catalogTables, rows[catalogTables], true)
	transfer.header.Content, transfer.untrusted = incoming.uuid.String(), false
	if err := r.finishImport(r.id, transfer); err != nil {
		t.Fatal(err)
	}
	if r.lookupCell(backupOnly.Shards[0].replicationResource()) != nil {
		t.Fatal("a restored catalog backup established data authority")
	}
}

func TestCatalogReplicaStagingProtectsPhysicalBackend(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-client-staging", 0)
	r := replicaRecoveryFixture()
	resource := tbl.schema.catalog.live[catalogColumns].replicationResource()
	content := uuid.NewString()
	transfer, err := r.startImport(mosiPacket{Resource: resource, Content: content})
	if err != nil {
		t.Fatal(err)
	}
	home := catalogBackendDatabase()
	if transfer.home != home || transfer.engine != home.persistence {
		t.Fatal("catalog staging escaped its physical system backend")
	}
	writer := transfer.engine.WriteColumn(content, "staging")
	if _, err := writer.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	finishColumnWrite(writer, true)
	r.imports.Store("client/staging", transfer)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	if _, deleted, complete := CleanDatabase(home); complete || deleted != 0 {
		t.Fatal("system cleanup ignored an active tenant catalog import")
	}
	reader := home.persistence.ReadColumn(content, "staging")
	var marker [1]byte
	_, err = io.ReadFull(reader, marker[:])
	reader.Close()
	if err != nil || marker[0] != 1 {
		t.Fatal("cleanup removed a partially received catalog column")
	}
	r.imports.Delete("client/staging")
}

func TestCatalogReplicaExportRetainsSystemFilesAcrossCheckpoint(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-client-export", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	schema := r.resourceCell(mosiResource{Database: db.Name})
	schema.ownstate.Establish(mosi.M)
	source := db.catalog.live[catalogColumns]
	resource := source.replicationResource()
	previous := replication.Swap(r)
	defer replication.Store(previous)
	release, err := replicaPersistenceRead(resource)
	if err != nil {
		t.Fatal(err)
	}
	transfer, err := r.captureResource(resource, resource.Shard)
	release()
	if err != nil {
		t.Fatal(err)
	}
	defer transfer.release()
	if !r.hasTransfers(db.Name) || !r.hasTransfers(source.home().Name) {
		t.Fatal("catalog export did not retain both logical and physical storage owners")
	}
	tbl.CreateColumn("new_definition", "INT", nil, nil)
	if db.catalog.live[catalogColumns].uuid == source.uuid {
		t.Fatal("fixture did not checkpoint a new catalog generation")
	}
	reader := source.home().persistence.ReadColumn(source.uuid.String(), "schema_name")
	var marker [1]byte
	_, err = io.ReadFull(reader, marker[:])
	reader.Close()
	if err != nil {
		t.Fatal("checkpoint removed a catalog column still owned by an export", err)
	}
	transfer.release()
	if r.hasTransfers(db.Name) || r.hasTransfers(source.home().Name) {
		t.Fatal("completed catalog export retained its storage owners")
	}
}

func TestCatalogMaterializationRetainsOnlyMatchingInitializer(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-initializer-projection", 0)
	db := tbl.schema
	initializer := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewInt(7)})
	tbl.OnInit.Store(&initializer)
	proposal := catalogRowsFromTable(tbl)
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	db.editCatalogLocked(&proposal, 1<<catalogTables, []string{tbl.Name})
	rows := db.catalogRowsLocked()
	before := catalogDocumentFromRows(db.Name, &rows)
	db.materializeCatalogTableLocked(tbl, &rows)
	if tbl.OnInit.Load() != &initializer {
		t.Fatal("equivalent catalog initializer lost its runtime binding")
	}
	if !reflect.DeepEqual(before, catalogDocumentFromRows(db.Name, &rows)) {
		t.Fatal("materialization modified the admitted rows")
	}
	diverged := scm.NewInt(99)
	tbl.OnInit.Store(&diverged)
	db.materializeCatalogTableLocked(tbl, &rows)
	if got := string(marshalCatalogValue(tbl.OnInit.Load())); got != string(marshalCatalogValue(initializer)) {
		t.Fatalf("runtime initializer overrode catalog authority: %s", got)
	}
	for _, row := range rows[catalogTables] {
		for index, field := range catalogRelations[catalogTables].columns() {
			if field.key == "oninit" {
				row[index] = scm.NewNil()
			}
		}
	}
	db.editCatalogLocked(&rows, 1<<catalogTables, nil)
	accepted := db.catalogRowsLocked()
	db.materializeCatalogTableLocked(tbl, &accepted)
	if tbl.OnInit.Load() != nil {
		t.Fatal("removed catalog initializer remained in the runtime view")
	}
}

func TestCatalogCacheRebuildDoesNotCreateBlobCatalog(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-cache-checkpoint", 1)
	db := tbl.schema
	cache, _ := CreateTable(db.Name, ".projection_cache", Cache, false)
	cache.CreateColumn("value", "INT", nil, nil)
	cache.Insert([]string{"value"}, [][]scm.Scmer{{scm.NewInt(42)}}, nil, scm.NewNil(), false, nil)
	if result := RebuildTable(cache, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	if db.tables.Get(".blobs") != nil {
		t.Fatal("cache compaction created an unused durable blob catalog")
	}
	if db.schemaDirty.Load() {
		t.Fatal("cache rebuild did not publish its schema checkpoint")
	}
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	if db.tables.Get(".blobs") == nil {
		t.Fatal("persistent table rebuild did not prepare its blob catalog")
	}
}

func TestCatalogCheckpointCannotOverwriteDefinitionsFromRuntime(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-checkpoint-authority", 1)
	db := tbl.schema
	// Deliberately corrupt only the derived runtime view. A maintenance save
	// must export the admitted definition, rather than silently adopting this.
	tbl.Comment = "runtime-only divergence"
	tbl.Columns[1].Comment = "runtime column divergence"
	db.save(nil)
	root := readCatalogRoot(db.persistence.ReadSchema())
	if root == nil || root.Version != catalogFormatVersion {
		t.Fatal("checkpoint did not preserve its catalog root")
	}
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = db.Name, db.persistence, COLD
	restored.ensureLoaded()
	definition := restored.tables.Get(tbl.Name)
	if definition.Comment != "" || definition.Columns[1].Comment != "" {
		t.Fatal("checkpoint adopted runtime declarations as catalog authority")
	}
	// Reconstructing a runtime projection repairs it from the tables as well.
	db.schemalock.Lock()
	rows := db.catalogRowsLocked()
	db.materializeCatalogTableLocked(tbl, &rows)
	db.schemalock.Unlock()
	if tbl.Comment != "" || tbl.Columns[1].Comment != "" {
		t.Fatal("runtime projection did not follow the catalog")
	}
}

func TestCatalogRejectsRetainedMutationCallback(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-callback-authority", 0)
	shard := tbl.schema.catalog.live[catalogColumns]
	_, before := catalogShardRecords(shard, catalogRelationColumns(catalogColumns), nil)
	callback := shard.UpdateFunction(0, true, false, nil, nil)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a retained catalog mutation callback was writable")
			}
		}()
		callback()
	}()
	_, after := catalogShardRecords(shard, catalogRelationColumns(catalogColumns), nil)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected callback changed catalog definitions")
	}
}

func TestCatalogRootNamesTheQueryableShards(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-queryable-generation", 0)
	db := tbl.schema
	root := readCatalogRoot(db.persistence.ReadSchema())
	for i, relation := range catalogRelations {
		shard := db.catalog.live[i]
		if shard.uuid.String() != root.Shards[relation.name] {
			t.Fatalf("%s query and disk roots name different generations", relation.name)
		}
		if shard.t != systemSchemaTables[i] || shard.home() != catalogBackendDatabase() {
			t.Fatalf("%s is not a native relation in the system backend", relation.name)
		}
	}
}

func TestCatalogScanLeavesRuntimeDefinitionsCold(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-lazy-runtime", 0)
	db := tbl.schema
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
	cold.bootstrapCatalog(nil)
	_, rows := catalogShardRecords(cold.catalog.live[catalogColumns], catalogRelationColumns(catalogColumns), nil)
	if len(rows) != 2 {
		t.Fatalf("catalog contains %d columns, want 2", len(rows))
	}
	if cold.srState != COLD || len(cold.tables.GetAll()) != 0 {
		t.Fatal("reading catalog rows materialized the database runtime")
	}
}

func TestCatalogRoutingOpensOnlySelectedAnchor(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-route-selected", 0)
	db := tbl.schema
	otherName := "catalog-route-unselected"
	CreateDatabase(otherName, false)
	other := GetDatabase(otherName)
	defer DropDatabase(otherName, true)
	selected := newDatabase()
	selected.Name, selected.persistence, selected.srState = db.Name, db.persistence, COLD
	unselected := newDatabase()
	unselected.Name, unselected.persistence, unselected.srState = other.Name, other.persistence, COLD
	databases.Set(db.Name, selected)
	databases.Set(other.Name, unselected)
	defer func() {
		databases.Set(db.Name, db)
		databases.Set(other.Name, other)
	}()
	for _, name := range []string{db.Name, other.Name} {
		publishCatalogShards(name, nil)
		publishCatalogDatabaseName(name)
	}
	value := scm.NewString(db.Name)
	access := runtimeScanAccess(analyzedBoundaries{{col: "schema_name", matcher: EqualMatcher,
		lower: value, upper: value, lowerInclusive: true, upperInclusive: true}})
	systemSchemaTables[catalogColumns].scanTopology(access)
	if selected.catalog.live[catalogColumns] == nil {
		t.Fatal("selected catalog partition was not opened")
	}
	if unselected.catalog.live[catalogColumns] != nil || unselected.srState != COLD {
		t.Fatal("a selective catalog scan opened an unrelated tenant")
	}
	if selected.srState != COLD || len(selected.tables.GetAll()) != 0 {
		t.Fatal("catalog routing materialized runtime table definitions")
	}
}

func TestCatalogCleanupRetainsPinnedGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-generation-reader", 0)
	db := tbl.schema
	old := db.catalog.live[catalogColumns]
	topology := systemSchemaTables[catalogColumns].activeTopology()
	if !topology.acquireOperation() {
		t.Fatal("could not retain the current catalog generation")
	}
	released := false
	defer func() {
		if !released {
			topology.releaseOperation()
		}
	}()
	CreateDatabase("catalog-generation-other", false)
	defer DropDatabase("catalog-generation-other", true)
	tbl.CreateColumn("later", "INT", nil, nil)
	if db.catalog.live[catalogColumns].uuid == old.uuid {
		t.Fatal("DDL did not publish a new queryable catalog generation")
	}
	CleanDatabase(catalogBackendDatabase())
	reader := old.home().persistence.ReadColumn(old.uuid.String(), "schema_name")
	if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
		t.Fatal("cleanup removed a catalog file before its reader drained")
	}
	reader.Close()
	topology.releaseOperation()
	released = true
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, retained := retiringCatalogGenerations.Load(old.uuid.String()); !retained {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("retired generation was not released after its reader drained")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestCatalogCleanupAndCheckpointMakeProgress(t *testing.T) {
	if os.Getenv("MEMCP_CATALOG_LOCK_CHILD") != "1" {
		binary, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, "-test.run=^TestCatalogCleanupAndCheckpointMakeProgress$", "-test.count=1")
		command.Env = append(os.Environ(), "MEMCP_CATALOG_LOCK_CHILD=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("cleanup/checkpoint failed to finish: %v (context: %v)\n%s", err, ctx.Err(), output)
		}
		return
	}
	tbl, _ := createDurabilityTestTable(t, "catalog-clean-checkpoint", 0)
	db := tbl.schema
	db.schemalock.Lock()
	done := make(chan struct{})
	go func() { cleanShards(db); close(done) }()
	// Let cleanup reach the held schema boundary. With the reversed order it
	// takes the persistence gate first; with the correct order it waits here.
	for range 256 {
		runtime.Gosched()
		if !catalogPersistenceMu.TryRLock() {
			break
		}
		catalogPersistenceMu.RUnlock()
	}
	db.commitSchemaLocked(true, db.schemalock.Unlock)
	<-done
}

func TestCatalogDoesNotFallbackToLegacyAfterMissingShardColumn(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-missing-column", 0)
	db := tbl.schema
	root := readCatalogRoot(db.persistence.ReadSchema())
	id := root.Shards["tables"]
	backend := catalogBackendDatabase().persistence
	reader := backend.ReadColumn(id, "schema_name")
	contents, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	backend.RemoveColumn(id, "schema_name")
	defer func() {
		writer := backend.WriteColumn(id, "schema_name")
		writer.Write(contents)
		writer.Close()
	}()
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
	defer cold.closeTransactionLog()
	defer func() {
		if recover() == nil {
			t.Fatal("missing authoritative catalog silently fell back to schema.json")
		}
	}()
	cold.ensureLoaded()
}

// Version 1 catalog roots used each tenant's backend. Reading those roots
// must not accidentally resolve their UUIDs on the new system backend.
func TestCatalogReadsTenantBackendRoot(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-v1-root", 1)
	db := tbl.schema
	db.schemalock.Lock()
	rows := db.catalogRowsLocked()
	db.schemalock.Unlock()
	root := &catalogRoot{Version: 1, Shards: make(map[string]string)}
	for i := range catalogRelations {
		shard := newCatalogShard(db, catalogRelationID(i), rows[i])
		persisted := shard.rebuild(true)
		root.Shards[catalogRelations[i].name] = persisted.uuid.String()
	}
	// A disagreeing compatibility document makes accidental fallback visible.
	db.persistence.WriteSchema(attachCatalogRoot([]byte(`{"name":"`+db.Name+`","tables":{}}`), root))
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = db.Name, db.persistence, COLD
	restored.ensureLoaded()
	if restored.tables.Get(tbl.Name) == nil {
		t.Fatal("tenant-backend catalog was not loaded")
	}
	restored.save(nil)
	rewritten := readCatalogRoot(db.persistence.ReadSchema())
	if rewritten == nil || rewritten.Version != catalogFormatVersion {
		t.Fatal("legacy catalog root was not upgraded on save")
	}
}

func TestCatalogMaterializationPreservesTriggerBeforeTargetBinding(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-trigger-binding", 0)
	called := false
	trigger := TriggerDescription{Name: ".prejoin:catalog-test", Timing: AfterInsert, Hidden: true,
		Func: scm.NewFunc(func(...scm.Scmer) scm.Scmer { called = true; return scm.NewNil() })}
	tbl.AddTrigger(trigger)
	tbl.schema.schemalock.Lock()
	tbl.schema.saveLockedAndUnlock(schemaSaveFsync, tbl)
	tbl.mu.Lock()
	restored := tbl.Triggers[0]
	tbl.mu.Unlock()
	if restored.Acquire != nil || restored.needsRegeneration {
		t.Fatal("materialization replaced a live trigger with a cold recovery stub")
	}
	scm.Apply(restored.Func)
	if !called {
		t.Fatal("materialization lost the live compiled callback")
	}
}

func TestCatalogPeerPublicationInitializesColdForeignAnchor(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "catalog-peer-publication", 0)
	accepted := stripCatalogRoot(engine.ReadSchema())
	// A shared tenant anchor may have been checkpointed by another node.
	// Its system-backend generations are intentionally absent on this node.
	root := &catalogRoot{Version: catalogFormatVersion, Shards: make(map[string]string)}
	for _, relation := range catalogRelations {
		root.Shards[relation.name] = uuid.NewString()
	}
	engine.WriteSchema(attachCatalogRoot(accepted, root))
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = tbl.schema.Name, engine, COLD
	databases.Set(cold.Name, cold)
	defer databases.Set(tbl.schema.Name, tbl.schema)
	r := replicaRecoveryFixture()
	r.cluster = &nodeCluster{manager: r, self: clusterMember{ID: r.id}}
	if err := r.installSchema(uuid.NewString(), cold.Name, engine, accepted); err != nil {
		t.Fatal(err)
	}
	if cold.GetTable(tbl.Name) == nil {
		t.Fatal("accepted peer schema did not initialize the cold database")
	}
	cold.schemalock.Lock()
	rows := cold.catalogRowsLocked()
	cold.schemalock.Unlock()
	if len(rows[catalogTables]) == 0 {
		t.Fatal("peer publication did not establish canonical catalog rows")
	}
}

func TestCatalogMosiReadUsesPublishedNativeGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-native-mosi", 0)
	db := tbl.schema
	db.schemalock.Lock()
	rows := db.catalogRowsLocked()
	db.schemalock.Unlock()
	source := db.catalog.live[catalogColumns]
	nameIndex := -1
	for i, field := range catalogRelations[catalogColumns].columns() {
		if field.key == "Name" {
			nameIndex = i
			break
		}
	}
	if nameIndex < 0 {
		t.Fatal("catalog column name field missing")
	}
	rows[catalogColumns][0][nameIndex] = scm.NewString("from-peer")
	received := newCatalogShard(db, catalogColumns, rows[catalogColumns])
	received.uuid = source.uuid
	received.catalogPrivate = false
	received.catalogHome = catalogBackendDatabase()
	r := replicaRecoveryFixture()
	cell := r.declareShard(source, r.id, true)
	if resource := *cell.resource.Load(); resource.Database != db.Name || resource.Table != catalogResourcePrefix+"columns" {
		t.Fatalf("catalog shard was registered outside its logical database: %+v", resource)
	}
	if source.home().Name != "system" {
		t.Fatal("catalog storage moved out of the system backend")
	}
	if err := r.replaceShard(source.replicationResource(), source, received); err != nil {
		t.Fatal(err)
	}
	cell.current.Store(received)
	old := replication.Swap(r)
	defer replication.Store(old)
	_, visible := catalogShardRecords(source, catalogRelationColumns(catalogColumns), nil)
	if len(visible) == 0 || visible[0][nameIndex].String() != "from-peer" {
		t.Fatalf("native catalog scan ignored MOSI generation: private=%v owner=%q manager=%v current=%p wanted=%p visible=%v", source.catalogPrivate, source.catalogOwner, source.replicationManager() != nil, cell.current.Load(), received, visible)
	}
}

func TestCatalogCheckpointPreservesNativeAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-native-checkpoint", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	old := replication.Swap(r)
	defer replication.Store(old)
	previous := db.catalog.live[catalogColumns]
	cell := r.shardCell(previous)
	resource := previous.replicationResource()
	historical := append([]byte(nil), db.persistence.ReadSchema()...)
	tbl.CreateColumn("added", "INT", nil, nil)
	current := db.catalog.live[catalogColumns]
	if previous.uuid == current.uuid {
		t.Fatal("checkpoint did not create a new catalog generation")
	}
	if r.resourceCell(current.replicationResource()) != cell.resolved() || cell.resolved().current.Load() != current {
		t.Fatal("checkpoint detached the native shard authority")
	}
	if r.canonicalResource(resource) != current.replicationResource() {
		t.Fatal("old catalog UUID did not forward to its checkpoint")
	}
	root := readCatalogRoot(db.persistence.ReadSchema())
	if root.Forwards[previous.uuid.String()] != current.uuid.String() {
		t.Fatal("checkpoint did not persist the native UUID forwarding edge")
	}
	_, visible := catalogShardRecords(previous, catalogRelationColumns(catalogColumns), nil)
	if len(visible) != len(tbl.Columns) {
		t.Fatal("old catalog stub did not read the new definition")
	}
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
	cold.bootstrapCatalog(historical)
	if installed := cell.resolved().publishedShard(); installed != current || cold.catalog.live[catalogColumns] != current {
		t.Fatal("historical root republished a predecessor over the acquired catalog generation")
	}
}

type catalogNamespaceBarrier struct {
	PersistenceEngine
	before func()
	calls  int
}

func (engine *catalogNamespaceBarrier) SyncNamespace() {
	engine.calls++
	engine.before()
	engine.PersistenceEngine.SyncNamespace()
}

func TestCatalogNamespaceDurabilityPrecedesTenantRoot(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "catalog-system-namespace-durability", 0)
	db, home := tbl.schema, catalogBackendDatabase()
	previous := append([]byte(nil), engine.ReadSchema()...)
	physical := home.persistence
	counting := &countingSchemaWritePersistence{PersistenceEngine: engine}
	db.persistence = counting
	defer func() { db.persistence = engine }()
	failure := "injected catalog namespace sync failure"
	barrier := &catalogNamespaceBarrier{PersistenceEngine: physical}
	barrier.before = func() {
		if counting.calls.Load() != 0 || string(engine.ReadSchema()) != string(previous) {
			t.Error("tenant root published before the system namespace durability barrier")
		}
		panic(failure)
	}
	home.persistence = barrier
	defer func() { home.persistence = physical }()
	func() {
		defer func() {
			if got := recover(); got != failure {
				t.Errorf("namespace barrier failure was not propagated: %v", got)
			}
		}()
		tbl.CreateColumn("after_barrier", "INT", nil, nil)
	}()
	if barrier.calls != 1 || counting.calls.Load() != 0 || string(engine.ReadSchema()) != string(previous) {
		t.Fatal("failed system namespace sync replaced the recoverable tenant root")
	}
	barrier.before = func() {
		if counting.calls.Load() != 0 {
			t.Error("retry published its root before syncing the system namespace")
		}
	}
	db.save(nil)
	if barrier.calls != 2 || counting.calls.Load() != 1 {
		t.Fatal("retry did not durably order the native namespace before its tenant root")
	}
	var accepted database
	if err := json.Unmarshal(engine.ReadSchema(), &accepted); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, column := range accepted.tables.Get(tbl.Name).Columns {
		found = found || column.Name == "after_barrier"
	}
	if !found {
		t.Fatal("retry omitted the accepted native column declaration")
	}
}

func TestCatalogCheckpointDoesNotReactivateHistoricalUUID(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "catalog-historical-checkpoint-uuid", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	r.resourceCell(mosiResource{Database: db.Name}).ownstate.Initialize(mosi.M)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	rows := db.catalogRowsLocked()
	initial := db.catalog.live[catalogColumns]
	cell := r.shardCell(initial)
	// A handover can advance immutable storage while retaining exactly the
	// same declarations. The last durable root still names its predecessor.
	successor := buildDiskShard(db, catalogColumns, rows[catalogColumns], true)
	if err := r.replaceCatalogShard(initial.replicationResource(), initial, successor); err != nil {
		t.Fatal(err)
	}
	db.save(nil)
	current := cell.resolved().current.Load()
	canonical := r.canonicalResource(initial.replicationResource())
	if current.uuid.String() != canonical.Shard {
		t.Fatalf("checkpoint reused historical payload %s under newer authority %s", current.uuid, canonical.Shard)
	}
	root := readCatalogRoot(engine.ReadSchema())
	if root.Shards[catalogRelations[catalogColumns].name] != current.uuid.String() {
		t.Fatal("durable root and native current publication disagree")
	}
	_, visible := catalogShardRecords(current, catalogRelationColumns(catalogColumns), nil)
	if catalogFingerprint(catalogColumns, visible) != catalogFingerprint(catalogColumns, rows[catalogColumns]) {
		t.Fatal("checkpoint changed equal declarations while advancing immutable identity")
	}
}

func TestCatalogCheckpointCannotReplaceReceivedGenerationDuringIO(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "catalog-received-during-checkpoint", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	r.resourceCell(mosiResource{Database: db.Name}).ownstate.Initialize(mosi.M)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	blocking := &blockingSchemaWritePersistence{PersistenceEngine: engine, entered: make(chan struct{}), release: make(chan struct{})}
	db.persistence = blocking
	defer func() { db.persistence = engine }()
	rows := db.catalogRowsLocked()
	var nameIndex int
	for index, field := range catalogRelations[catalogColumns].columns() {
		if field.key == "Name" {
			nameIndex = index
		}
	}
	rows[catalogColumns][0][nameIndex] = scm.NewString("checkpoint_definition")
	db.schemalock.Lock()
	db.editCatalogLocked(&rows, 1<<catalogColumns, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		db.commitSchemaLocked(true, db.schemalock.Unlock)
	}()
	defer func() {
		select {
		case <-blocking.release:
		default:
			close(blocking.release)
		}
		<-done
	}()
	<-blocking.entered
	rows[catalogColumns][0][nameIndex] = scm.NewString("received_definition")
	received := buildDiskShard(db, catalogColumns, rows[catalogColumns], true)
	live := db.catalog.live[catalogColumns]
	cell := r.shardCell(live)
	if err := r.replaceCatalogShard(live.replicationResource(), live, received); err != nil {
		t.Fatal(err)
	}
	close(blocking.release)
	<-done
	if cell.resolved().current.Load() != received {
		t.Fatal("checkpoint I/O completion replaced the newer received native generation")
	}
	root := readCatalogRoot(engine.ReadSchema())
	snapshot := received.replicationResource()
	snapshot.Shard = root.Shards[catalogRelations[catalogColumns].name]
	if r.canonicalResource(snapshot) != received.replicationResource() {
		t.Fatal("durable historical checkpoint did not resolve to the current native authority")
	}
}

func TestCatalogCheckpointDoesNotPublishDuringNativeHandover(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "catalog-checkpoint-during-handover", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	r.resourceCell(mosiResource{Database: db.Name}).ownstate.Initialize(mosi.M)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	rows := db.catalogRowsLocked()
	for index, field := range catalogRelations[catalogColumns].columns() {
		if field.key == "Name" {
			rows[catalogColumns][0][index] = scm.NewString("checkpoint_definition")
		}
	}
	db.schemalock.Lock()
	db.editCatalogLocked(&rows, 1<<catalogColumns, nil)
	live := db.catalog.live[catalogColumns]
	cell := r.shardCell(live)
	handover, err := cell.ownstate.BeginHandover(context.Background(), cell.drainWrites)
	if err != nil {
		db.schemalock.Unlock()
		t.Fatal(err)
	}
	defer handover.Release()
	db.commitSchemaLocked(true, db.schemalock.Unlock)
	if cell.resolved().current.Load() != live {
		t.Fatal("checkpoint changed the UUID/payload binding of an active native handover")
	}
	root := readCatalogRoot(engine.ReadSchema())
	if root.Shards[catalogRelations[catalogColumns].name] == live.uuid.String() {
		t.Fatal("fixture did not persist a distinct checkpoint generation")
	}
	transfer, err := r.captureResource(live.replicationResource(), live.uuid.String())
	if err != nil {
		t.Fatal(err)
	}
	defer transfer.release()
	for _, route := range transfer.routes {
		if route.To.Shard != transfer.content {
			t.Fatal("checkpoint changed advertised UUID routes away from the bound handover content")
		}
	}
}

func TestCatalogScopedEditsPreserveUnrelatedDefinitions(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-scoped-edits", 0)
	db := tbl.schema
	other, _ := CreateTable(db.Name, "unrelated", Safe, false)
	other.CreateColumn("kept", "TEXT", nil, []scm.Scmer{scm.NewString("default"), scm.NewString("unchanged <&> Gröẞe")})
	read := func() []string {
		var keys []string
		for i := catalogTables; i < catalogProcedures; i++ {
			_, rows := catalogShardRecords(db.catalog.live[i], catalogRelationColumns(i), []string{other.Name})
			for _, row := range rows {
				keys = append(keys, catalogRowKey(catalogRelations[i].columns(), row))
			}
		}
		sort.Strings(keys)
		return keys
	}
	before := read()
	tbl.AlterColumn("payload", "comment", scm.NewString("changed"))
	tbl.CreateColumn("temporary", "INT", nil, nil)
	tbl.DropColumn("temporary")
	if after := read(); len(before) == 0 || !reflect.DeepEqual(before, after) {
		t.Fatal("scoped catalog edits changed another table's authoritative rows")
	}
	_, rows := catalogShardRecords(db.catalog.live[catalogColumns], catalogRelationColumns(catalogColumns), []string{tbl.Name})
	if len(rows) != 2 {
		t.Fatalf("scoped add/drop left %d columns, want 2", len(rows))
	}
}
