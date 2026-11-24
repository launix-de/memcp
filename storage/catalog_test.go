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
	"sync"
	"sync/atomic"
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

func TestCatalogRuntimeFilteringKeepsChildOrdinalsDense(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-runtime-ordinals", 0)
	db := tbl.schema
	tbl.Columns = []*column{
		{Name: "runtime_first", IsTemp: true, RuntimeOnly: true},
		{Name: "id", Typ: "INT"},
		{Name: "runtime_middle", IsTemp: true, RuntimeOnly: true},
		{Name: "value", Typ: "INT"},
	}
	tbl.Triggers = []TriggerDescription{
		{Name: ".prejoin:runtime"},
		{Name: "user_first"},
		{Name: ".cache:runtime"},
		{Name: "user_second"},
	}
	rows := catalogRowsFromTable(tbl)
	// Legacy JSON and native reload both require dense child ordinals. Runtime
	// programs can precede durable fields after a query followed by user DDL.
	document := catalogDocumentFromRows(db.Name, &rows)
	var restored catalogDocument
	if err := json.Unmarshal(document, &restored); err != nil {
		t.Fatal(err)
	}
	var definition struct {
		Columns  []struct{ Name string }
		Triggers []struct{ Name string }
	}
	if err := json.Unmarshal(restored.Tables[tbl.Name], &definition); err != nil {
		t.Fatal(err)
	}
	if len(definition.Columns) != 2 || definition.Columns[0].Name != "id" || definition.Columns[1].Name != "value" {
		t.Fatalf("runtime filtering changed durable columns: %+v", definition.Columns)
	}
	if len(definition.Triggers) != 2 || definition.Triggers[0].Name != "user_first" || definition.Triggers[1].Name != "user_second" {
		t.Fatalf("runtime filtering changed durable triggers: %+v", definition.Triggers)
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

func TestCatalogExportPreservesRawDefaultsAndLiteralJSONText(t *testing.T) {
	document := []byte(`{"name":"shop","tables":{"values":{"Name":"values","Columns":[
		{"Name":"a","Comment":"true","Default":true},
		{"Name":"b","Comment":"[1,\"x\"]","Default":[1,"x"]},
		{"Name":"c","Comment":"true","Default":true}
	]}}}`)
	rows := catalogRowsFromDocument(document)
	var decoded catalogDocument
	if err := json.Unmarshal(catalogDocumentFromRows("shop", &rows), &decoded); err != nil {
		t.Fatal(err)
	}
	var record struct {
		Columns []struct {
			Comment string
			Default any
		}
	}
	if err := json.Unmarshal(decoded.Tables["values"], &record); err != nil {
		t.Fatal(err)
	}
	for i, expected := range []any{true, []any{float64(1), "x"}, true} {
		if !reflect.DeepEqual(record.Columns[i].Default, expected) {
			t.Fatalf("default %d changed JSON type: %v", i, record.Columns[i].Default)
		}
	}
	if record.Columns[0].Comment != "true" || record.Columns[1].Comment != `[1,"x"]` || record.Columns[2].Comment != "true" {
		t.Fatal("literal JSON text was interpreted as a raw default", record.Columns)
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
	parent, _ := CreateTable(nil, schema, "parent", Safe, false)
	parent.CreateColumn(nil, "id", "int", nil, []scm.Scmer{scm.NewString("primary"), scm.NewBool(true)})
	parent.CreateColumn(nil, "label", "varchar", []int{20}, []scm.Scmer{scm.NewString("comment"), scm.NewString("shown name")})
	child, _ := CreateTable(nil, schema, "child", Safe, false)
	child.CreateColumn(nil, "id", "int", nil, nil)
	child.CreateColumn(nil, "parent_id", "int", nil, nil)
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
	if published == nil || published.uuid != incoming.uuid || published.home() != catalogBackendDatabase() || published.catalogPrivate.Load() {
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
			before := db.catalogRowsLocked(nil)
			db.schemalock.Unlock()
			cache := db.newTable(".grp:local", Cache)
			cache.RuntimeOnly = true
			db.tables.Set(cache.Name, cache)
			compute := &column{Name: "local_compute", Typ: "INT", IsTemp: true, RuntimeOnly: true, Computor: scm.NewSymbol("local_runtime_helper")}
			tbl.Columns = append(tbl.Columns, compute)
			db.schemalock.Lock()
			db.proposeCatalogTablesLocked(nil, tbl, cache)
			rows := db.catalogRowsLocked(nil)
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
	cache := db.newTable(".grp:checkpoint_scope", Cache)
	cache.RuntimeOnly = true
	db.tables.Set(cache.Name, cache)
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
	db.save(nil, nil, source)
	after = readCatalogRoot(db.persistence.ReadSchema())
	if after.Shards["tables"] == before.Shards["tables"] {
		t.Fatal("source checkpoint lost its advisory statistics")
	}
}

func TestCatalogRuntimeCacheRebuildDoesNotPublishLegacyMigration(t *testing.T) {
	source, engine := createDurabilityTestTable(t, "catalog_runtime_cache_legacy", 0)
	legacy := stripCatalogRoot(engine.ReadSchema())
	engine.WriteSchema(legacy)
	loaded := reloadTableFromPersistence(t, source.schema.Name, engine)
	db := loaded.schema
	databases.Set(db.Name, db)
	defer databases.Set(source.schema.Name, source.schema)
	cache := db.newTable(".grp:legacy", Cache)
	cache.RuntimeOnly = true
	db.tables.Set(cache.Name, cache)
	counting := &countingSchemaWritePersistence{PersistenceEngine: db.persistence}
	db.persistence = counting
	if result := RebuildTable(cache, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	DropTable(nil, db.Name, cache.Name, false)
	if counting.calls.Load() != 0 || string(engine.ReadSchema()) != string(legacy) {
		t.Fatal("physical query cache published a durable legacy catalog migration")
	}
	// A real source checkpoint still migrates and dual-writes the native root.
	db.save(nil, nil, loaded)
	if readCatalogRoot(engine.ReadSchema()) == nil {
		t.Fatal("source checkpoint did not persist its native catalog")
	}
}

func TestCatalogReplicaSchemaBackupPreservesRemoteAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog_passive_schema_backup", 0)
	db := tbl.schema
	data := db.persistence.ReadSchema()
	r := replicaRecoveryFixture()
	owner := uuid.NewString()
	for _, shard := range db.catalog.committed {
		cell := r.declareShard(shard, owner, false)
		cell.ownstate.Publish(mosi.IO)
		cell.clusterstate.Publish(mosi.M)
		r.updateRecord(shard.replicationResource(), func(record *replicaRecord) bool {
			record.State, record.Epoch = "IO", 7
			return true
		})
	}
	previous := replication.Swap(r)
	defer replication.Store(previous)
	// There is deliberately no connection for recalling M. Saving admitted
	// immutable files must not acquire query permissions for a safety copy.
	if err := r.installSchema(owner, db.Name, db.persistence, data, true); err != nil {
		t.Fatal(err)
	}
	for _, shard := range db.catalog.live {
		cell := r.shardCell(shard)
		record := r.directory.Records[r.resourceKey(shard.replicationResource())]
		if cell.ownstate.Phase() != mosi.IO || record.Owner != owner || record.Epoch != 7 {
			t.Fatalf("backup changed native authority: %+v", record)
		}
	}
	if readCatalogRoot(db.persistence.ReadSchema()) == nil {
		t.Fatal("passive backup lost the native root")
	}
}

func TestCatalogDataPublicationDoesNotCheckpointDefinitions(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog_passive_data_publication", 3)
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	original := tbl.ActiveShards()[0]
	received := original.rebuild(true)
	before := string(tbl.schema.persistence.ReadSchema())
	r := replicaRecoveryFixture()
	owner := uuid.NewString()
	r.schemaShards(tbl.schema, owner, false)
	cell := r.shardCell(original)
	cell.ownstate.Publish(mosi.S)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	// Catalog authority is remote and cannot be recalled without a connection.
	// Receiving one data generation is still a legal runtime publication.
	if err := r.replaceShard(original.replicationResource(), original, received); err != nil {
		t.Fatal(err)
	}
	if got := string(tbl.schema.persistence.ReadSchema()); got != before {
		t.Fatal("data receipt changed the independent catalog checkpoint")
	}
	if current := tbl.ActiveShards()[0]; current != received {
		t.Fatal("data receipt did not publish the accepted native generation")
	}
	for _, shard := range tbl.schema.replicationShards() {
		if shard.catalogOwner != "" && r.shardCell(shard).ownstate.Phase() != mosi.IO {
			t.Fatal("data receipt acquired independent catalog rights")
		}
	}
}

func TestCatalogReplicaTablesIntroduceDataAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-new-data-authority", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	introduced := db.newTable("introduced", Safe)
	rows := db.catalogRowsLocked(nil)
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
	rows = db.catalogRowsLocked(nil)
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
	tbl.CreateColumn(nil, "new_definition", "INT", nil, nil)
	db.save(nil, nil) // force a physical checkpoint while the prior export lives
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
	db.editCatalogLocked(nil, &proposal, 1<<catalogTables, []string{tbl.Name})
	rows := db.catalogRowsLocked(nil)
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
	db.editCatalogLocked(nil, &rows, 1<<catalogTables, nil)
	accepted := db.catalogRowsLocked(nil)
	db.materializeCatalogTableLocked(tbl, &accepted)
	if tbl.OnInit.Load() != nil {
		t.Fatal("removed catalog initializer remained in the runtime view")
	}
}

func TestCatalogCacheRebuildDoesNotCreateBlobCatalog(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-cache-checkpoint", 1)
	db := tbl.schema
	cache, _ := CreateTable(nil, db.Name, ".projection_cache", Cache, false)
	cache.CreateColumn(nil, "value", "INT", nil, nil)
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
	db.save(nil, nil)
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
	rows := db.catalogRowsLocked(nil)
	db.materializeCatalogTableLocked(tbl, &rows)
	db.schemalock.Unlock()
	if tbl.Comment != "" || tbl.Columns[1].Comment != "" {
		t.Fatal("runtime projection did not follow the catalog")
	}
}

func TestCatalogRejectsRetainedMutationCallback(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-callback-authority", 0)
	shard := tbl.schema.catalog.live[catalogColumns]
	_, before := catalogShardRecords(nil, shard, catalogRelationColumns(catalogColumns), nil)
	callback := shard.UpdateFunction(0, true, false, nil, nil)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a retained catalog mutation callback was writable")
			}
		}()
		callback()
	}()
	_, after := catalogShardRecords(nil, shard, catalogRelationColumns(catalogColumns), nil)
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
	_, rows := catalogShardRecords(nil, cold.catalog.live[catalogColumns], catalogRelationColumns(catalogColumns), nil)
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
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	systemSchemaTables[catalogColumns].scanTopology(tx, access)
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

func TestCatalogCleanupRetainsAnchorBeforeSnapshotPublication(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "catalog-private-snapshot-cleanup", 0)
	db := tbl.schema
	root := readCatalogRoot(engine.ReadSchema())
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = db.Name, engine, COLD
	// Snapshot decoding has consumed the bootstrap slot but has not published
	// its Go descriptors. Only the durable anchor currently owns these files.
	cold.catalogBootstrap.Do(func() {})
	databases.Set(cold.Name, cold)
	publishCatalogShards(db.Name, nil)
	defer func() {
		databases.Set(db.Name, db)
		publishCatalogShards(db.Name, &db.catalog.live)
	}()
	cleanShards(catalogBackendDatabase())
	for relation, id := range root.Shards {
		reader := catalogBackendDatabase().persistence.ReadColumn(id, "schema_name")
		missing, ok := reader.(interface{ Missing() bool })
		absent := ok && missing.Missing()
		reader.Close()
		if absent {
			t.Fatalf("cleanup deleted anchored %s before snapshot publication", relation)
		}
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
	tbl.CreateColumn(nil, "later", "INT", nil, nil)
	db.save(nil, nil) // compact while the old topology still has its reader
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
	// Let cleanup reach the held schema boundary. Checkpoint publication must
	// make progress without waiting for a collector holding a global gate.
	for range 256 {
		runtime.Gosched()
	}
	db.commitSchemaLocked(nil, true, db.schemalock.Unlock)
	<-done
}

type delayedCatalogCleanupPersistence struct {
	PersistenceEngine
	phase   string
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *delayedCatalogCleanupPersistence) wait(phase string) {
	if p.phase == phase {
		p.once.Do(func() { close(p.entered); <-p.release })
	}
}
func (p *delayedCatalogCleanupPersistence) ReadSchema() []byte {
	p.wait("anchor")
	return p.PersistenceEngine.ReadSchema()
}
func (p *delayedCatalogCleanupPersistence) WalkShardFiles(fn func(string)) {
	p.wait("walk")
	p.PersistenceEngine.WalkShardFiles(fn)
}
func (p *delayedCatalogCleanupPersistence) DeleteShardFile(name string) {
	p.wait("delete")
	p.PersistenceEngine.DeleteShardFile(name)
}

func TestCatalogSlowCleanupDoesNotBlockUnrelatedDDL(t *testing.T) {
	for _, phase := range []string{"walk", "anchor", "delete"} {
		t.Run(phase, func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "catalog-slow-cleanup-"+phase, 0)
			home := catalogBackendDatabase()
			target := home
			if phase == "anchor" {
				target = tbl.schema
			}
			physical := target.persistence
			delay := &delayedCatalogCleanupPersistence{PersistenceEngine: physical, phase: phase, entered: make(chan struct{}), release: make(chan struct{})}
			if phase == "delete" {
				writer := home.persistence.WriteColumn(uuid.NewString(), "orphan")
				writer.Write([]byte("unowned"))
				writer.Close()
			}
			target.persistence = delay
			cleanupDone := make(chan struct{})
			go func() { defer close(cleanupDone); cleanShards(home) }()
			release := sync.OnceFunc(func() { close(delay.release) })
			defer func() {
				release()
				<-cleanupDone
				target.persistence = physical
			}()
			select {
			case <-delay.entered:
			case <-time.After(2 * time.Second):
				t.Fatal("cleanup did not reach delayed backend operation")
			}
			done := make(chan struct{})
			go func() { defer close(done); tbl.CreateColumn(nil, "added", "INT", nil, nil) }()
			defer func() { release(); <-done }()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("unrelated DDL waited for cleanup backend I/O")
			}
			root := readCatalogRoot(physical.ReadSchema())
			if target != tbl.schema {
				root = readCatalogRoot(tbl.schema.persistence.ReadSchema())
			}
			if root == nil {
				t.Fatal("DDL failed to publish its durable catalog root")
			}
			release()
			<-cleanupDone
			// The new root was published while cleanup was blocked. Collection
			// must retain its entire payload, including the added column.
			cold := newDatabase()
			cold.Name, cold.persistence, cold.srState = tbl.schema.Name, tbl.schema.persistence, COLD
			defer cold.closeTransactionLog()
			cold.ensureLoaded()
			restored := cold.GetTable(tbl.Name)
			if restored == nil || len(restored.Columns) != 3 || restored.Columns[2].Name != "added" {
				t.Fatal("cleanup removed files from the concurrently published catalog")
			}
		})
	}
}

func TestCatalogRenameUsesOneNativeDecision(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-rename-decision", 0)
	db := tbl.schema
	oldname := tbl.Name
	before := atomic.LoadUint64(&GlobalCommitEpoch)
	RenameTable(nil, db.Name, oldname, "renamed")
	if got := atomic.LoadUint64(&GlobalCommitEpoch); got != before+1 {
		t.Fatalf("rename published %d native decisions, want one", got-before)
	}
	if db.GetTable(oldname) != nil || db.GetTable("renamed") != tbl {
		t.Fatal("RAM directory does not match rename")
	}
	rows := db.catalogRowsLocked(nil)
	for i := catalogTables; i < catalogProcedures; i++ {
		for _, row := range rows[i] {
			if i == catalogTables {
				if catalogTableName(row) != "renamed" {
					t.Fatal("catalog table retained old name")
				}
			} else if row[1].String() != "renamed" {
				t.Fatal("catalog child retained old owner")
			}
		}
	}
}

func TestCatalogSizeEstimateDoesNotAcquireDataLocks(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-size-estimate", 8192)
	fn := scm.Globalenv.Vars[scm.Symbol("table_size_estimate")]
	for _, statistics := range []*tableStatisticsSnapshot{nil, {rowCount: 8192, sizeBytes: 123456}} {
		snapshot := tbl.buildShowColumnsSnapshot(8192, nil)
		snapshot.metadata.statistics = statistics
		tbl.showColumnsSnapshot.Store(snapshot)
		// A metadata scalar must complete while every data and definition
		// mutex is unavailable. Diagnostic memory accounting cannot do that.
		tbl.schema.schemalock.Lock()
		tbl.mu.Lock()
		shards := tbl.ActiveShards()
		for _, shard := range shards {
			shard.mu.Lock()
		}
		result := make(chan scm.Scmer, 1)
		go func() { result <- scm.Apply(fn, NewTableScmer(tbl)) }()
		var value scm.Scmer
		completed := false
		select {
		case value = <-result:
			completed = true
		case <-time.After(time.Second):
		}
		for _, shard := range shards {
			shard.mu.Unlock()
		}
		tbl.mu.Unlock()
		tbl.schema.schemalock.Unlock()
		if !completed {
			<-result
			t.Fatal("size estimate acquired a data or definition lock")
		}
		if statistics == nil {
			if !value.IsNil() {
				t.Fatalf("unknown size must remain unknown, got %v", value)
			}
		} else if scm.ToInt(value) != 123456 {
			t.Fatalf("size estimate = %v, want published estimate 123456", value)
		}
	}
}

func TestCatalogEditsFollowOriginalTransaction(t *testing.T) {
	for _, mode := range []TxMode{TxACID, TxCursorStability} {
		for _, commit := range []bool{false, true} {
			for _, indexed := range []bool{false, true} {
				t.Run(fmt.Sprintf("mode-%d-commit-%t-indexed-%t", mode, commit, indexed), func(t *testing.T) {
					tbl, _ := createDurabilityTestTable(t, "catalog-original-transaction", 1)
					db := tbl.schema
					before := db.catalogRowsLocked(nil)
					tx := NewTxContext(mode)
					defer func() {
						if tx.State == TxActive {
							tx.Rollback()
						}
					}()
					id, epoch := tx.ID, tx.SnapshotEpoch
					// Catalog admission must join prior native work without resetting
					// its identity, snapshot, overlays or undo journal.
					tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("before-ddl")}}, nil, scm.NewNil(), false, nil, tx)
					comment, name := -1, -1
					for i, field := range catalogRelations[catalogColumns].columns() {
						if field.key == "Comment" {
							comment = i
						}
						if field.key == "Name" {
							name = i
						}
					}
					if comment < 0 || name < 0 {
						t.Fatal("column comment is missing from native relation")
					}
					idColumn := func(rows catalogRows) []scm.Scmer {
						for _, row := range rows[catalogColumns] {
							if row[name].String() == "id" {
								return row
							}
						}
						var definitions []string
						for _, row := range rows[catalogColumns] {
							definitions = append(definitions, fmt.Sprintf("%v:%v", row[name], row[comment]))
						}
						t.Fatalf("id column disappeared: %v", definitions)
						return nil
					}
					edit := func(value string) {
						db.schemalock.Lock()
						defer db.schemalock.Unlock()
						rows := db.catalogRowsLocked(tx)
						idColumn(rows)[comment] = scm.NewString(value)
						var owners []string
						if indexed {
							owners = []string{tbl.Name}
						}
						if !db.editCatalogLocked(tx, &rows, 1<<catalogColumns, owners) {
							t.Fatal("catalog mutation was not staged")
						}
						if tx.State != TxActive || tx.ID != id || tx.SnapshotEpoch != epoch {
							t.Fatal("catalog editor finalized or replaced the original transaction")
						}
					}
					edit("first")
					sp := tx.CreateSavepoint()
					edit("second")
					edit("third")
					tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(3), scm.NewString("after-savepoint")}}, nil, scm.NewNil(), false, nil, tx)
					tx.RollbackToSavepoint(sp)
					own := db.catalogRowsLocked(tx)
					if got := idColumn(own)[comment].String(); got != "first" || len(own[catalogColumns]) != len(before[catalogColumns]) {
						t.Fatalf("savepoint restored wrong definitions: %q, %d rows", got, len(own[catalogColumns]))
					}
					_, ownData := catalogShardRecords(tx, tbl.ActiveShards()[0], []string{"id", "payload"}, nil)
					wantData := [][]scm.Scmer{{scm.NewInt(1), scm.NewString("row-00000001")}, {scm.NewInt(2), scm.NewString("before-ddl")}}
					if !reflect.DeepEqual(ownData, wantData) {
						t.Fatalf("savepoint lost original data journal: %v", ownData)
					}
					if commit {
						if err := tx.Commit(); err != nil {
							t.Fatal(err)
						}
					} else {
						tx.Rollback()
						wantData = wantData[:1]
					}
					accepted := db.catalogRowsLocked(nil)
					want := before
					if commit {
						want = own
					}
					for i := range catalogRelations {
						if catalogFingerprint(catalogRelationID(i), accepted[i]) != catalogFingerprint(catalogRelationID(i), want[i]) {
							t.Fatalf("relation %s did not follow original decision", catalogRelations[i].name)
						}
					}
					_, acceptedData := catalogShardRecords(nil, tbl.ActiveShards()[0], []string{"id", "payload"}, nil)
					if !reflect.DeepEqual(acceptedData, wantData) {
						t.Fatalf("data and catalog followed different decisions: %v", acceptedData)
					}
					// No compatibility export or catalog checkpoint has run since
					// these edits. Both definitions and data must recover through
					// their existing WALs and the same tenant commit authority.
					cold := newDatabase()
					cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
					previous := databases.Set(db.Name, cold)
					defer func() {
						databases.Set(db.Name, previous)
						publishCatalogShards(db.Name, &db.catalog.live)
						cold.closeTransactionLog()
					}()
					cold.ensureLoaded()
					recovered := cold.catalogRowsLocked(nil)
					for i := range catalogRelations {
						if catalogFingerprint(catalogRelationID(i), recovered[i]) != catalogFingerprint(catalogRelationID(i), want[i]) {
							t.Fatalf("relation %s recovered a different transaction decision", catalogRelations[i].name)
						}
					}
					restored := cold.GetTable(tbl.Name)
					_, recoveredData := catalogShardRecords(nil, restored.ActiveShards()[0], []string{"id", "payload"}, nil)
					if !reflect.DeepEqual(recoveredData, wantData) {
						t.Fatalf("data and catalog recovered different decisions: %v", recoveredData)
					}
				})
			}
		}
	}
}

func TestSchemaDefinitionScopeExcludesConflictingDDL(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "schema-definition-scope", 1)
	reader, writer := NewTxContext(TxACID), NewTxContext(TxACID)
	defer func() {
		if reader.State == TxActive {
			reader.Rollback()
		}
		if writer.State == TxActive {
			writer.Rollback()
		}
	}()
	_, before := catalogShardRecords(reader, tbl.ActiveShards()[0], []string{"id", "payload"}, nil)
	var failure any
	func() {
		defer func() { failure = recover() }()
		tbl.CreateColumn(writer, "added", "INT", nil, nil)
	}()
	if failure == nil {
		t.Fatal("DDL changed a definition retained by another transaction")
	}
	writer.Rollback()
	_, after := catalogShardRecords(reader, tbl.ActiveShards()[0], []string{"id", "payload"}, nil)
	if !reflect.DeepEqual(before, after) || len(tbl.Columns) != 2 {
		t.Fatal("rejected DDL changed the reader's data or definition")
	}
	reader.Rollback()
	accepted := NewTxContext(TxACID)
	defer func() {
		if accepted.State == TxActive {
			accepted.Rollback()
		}
	}()
	if !tbl.CreateColumn(accepted, "added", "INT", nil, nil) {
		t.Fatal("released reader continued to exclude DDL")
	}
	if err := accepted.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(tbl.Columns) != 3 {
		t.Fatal("accepted DDL did not publish its definition")
	}
}

func TestParkedSchemaDefinitionAllowsMosiHandoverAndRenewsAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "schema-parked-mosi", 1)
	db := tbl.schema
	r := replicaRecoveryFixture()
	r.schemaShards(db, r.id, true)
	cell := r.resourceCell(mosiResource{Database: db.Name})
	cell.ownstate.Initialize(mosi.M)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	tx.readSchema(db)
	if handover := cell.ownstate.TryHandover(cell.writesDrained, true); handover != nil {
		handover.Release()
		t.Fatal("active statement relinquished schema authority")
	}
	tx.endStatement()
	handover := cell.ownstate.TryHandover(cell.writesDrained, true)
	if handover == nil {
		t.Fatal("parked transaction blocked physical schema handover")
	}
	handover.Release()
	if db.definitionMu.TryLock() {
		db.definitionMu.Unlock()
		t.Fatal("parking released the transaction's local definition")
	}
	tx.readSchema(db)
	if handover := cell.ownstate.TryHandover(cell.writesDrained, true); handover != nil {
		handover.Release()
		t.Fatal("resumed statement did not renew MOSI authority")
	}
	tx.endStatement()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !db.definitionMu.TryLock() {
		t.Fatal("commit retained the local definition")
	}
	db.definitionMu.Unlock()
}

func TestSchemaProjectionFollowsOriginalTransaction(t *testing.T) {
	for _, mode := range []TxMode{TxACID, TxCursorStability} {
		for _, commit := range []bool{false, true} {
			for _, savepoint := range []bool{false, true} {
				t.Run(fmt.Sprintf("mode-%d-commit-%t-savepoint-%t", mode, commit, savepoint), func(t *testing.T) {
					tbl, _ := createDurabilityTestTable(t, "schema-original-projection", 1)
					db := tbl.schema
					tx := NewTxContext(mode)
					defer func() {
						if tx.State == TxActive {
							tx.Rollback()
						}
					}()
					tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("prior-dml")}}, nil, scm.NewNil(), false, nil, tx)
					sp := tx.CreateSavepoint()
					if !tbl.CreateColumn(tx, "added", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(7)}) {
						t.Fatal("ADD did not create the original transaction's definition")
					}
					_, own := catalogShardRecords(tx, tbl.ActiveShards()[0], []string{"id", "payload", "added"}, nil)
					wantOwn := [][]scm.Scmer{{scm.NewInt(1), scm.NewString("row-00000001"), scm.NewInt(7)}, {scm.NewInt(2), scm.NewString("prior-dml"), scm.NewInt(7)}}
					if !reflect.DeepEqual(own, wantOwn) {
						t.Fatalf("data did not use its effective definition: %v", own)
					}
					if savepoint {
						tx.RollbackToSavepoint(sp)
						if len(tbl.Columns) != 2 {
							t.Fatal("savepoint retained the added RAM declaration")
						}
					}
					if commit {
						if err := tx.Commit(); err != nil {
							t.Fatal(err)
						}
					} else {
						tx.Rollback()
					}
					width, count := 2, 1
					if commit {
						count = 2
						if !savepoint {
							width = 3
						}
					}
					if len(tbl.Columns) != width {
						t.Fatalf("RAM has %d columns, want %d", len(tbl.Columns), width)
					}
					cold := newDatabase()
					cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
					previous := databases.Set(db.Name, cold)
					defer func() {
						databases.Set(db.Name, previous)
						publishCatalogShards(db.Name, &db.catalog.live)
						cold.closeTransactionLog()
					}()
					cold.ensureLoaded()
					restored := cold.GetTable(tbl.Name)
					if len(restored.Columns) != width {
						t.Fatal("restarted definitions disagree with the completed transaction")
					}
					columns := []string{"id", "payload"}
					if width == 3 {
						columns = append(columns, "added")
					}
					_, recovered := catalogShardRecords(nil, restored.ActiveShards()[0], columns, nil)
					want := make([][]scm.Scmer, count)
					for i := range want {
						want[i] = wantOwn[i][:width]
					}
					if !reflect.DeepEqual(recovered, want) {
						t.Fatalf("restart recovered mismatched data/schema: %v", recovered)
					}
				})
			}
		}
	}
}

func TestSchemaTransactionSurvivesPhysicalShardGrowth(t *testing.T) {
	previousSize := Settings.ShardSize
	Settings.ShardSize = 2
	defer func() { Settings.ShardSize = previousSize }()
	for _, mode := range []TxMode{TxACID, TxCursorStability} {
		for _, commit := range []bool{false, true} {
			t.Run(fmt.Sprintf("mode-%d-commit-%t", mode, commit), func(t *testing.T) {
				tbl, _ := createDurabilityTestTable(t, "schema-growing-transaction", 1)
				tx := NewTxContext(mode)
				defer func() {
					if tx.State == TxActive {
						tx.Rollback()
					}
					tbl.maintenanceMu.Lock()
					tbl.maintenanceMu.Unlock()
				}()
				tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("before-ddl")}}, nil, scm.NewNil(), false, nil, tx)
				tbl.CreateColumn(tx, "added", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(7)})
				tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(3), scm.NewString("after-ddl")}, {scm.NewInt(4), scm.NewString("after-ddl")}}, nil, scm.NewNil(), false, nil, tx)
				if commit {
					if err := tx.Commit(); err != nil {
						t.Fatal(err)
					}
				} else {
					tx.Rollback()
				}
				columns := []string{"id", "payload"}
				want := 1
				if commit {
					columns, want = append(columns, "added"), 4
				}
				check := func(current *table) {
					var rows [][]scm.Scmer
					for _, shard := range current.ActiveShards() {
						_, part := catalogShardRecords(nil, shard, columns, nil)
						rows = append(rows, part...)
					}
					sort.Slice(rows, func(i, j int) bool { return scm.ToInt(rows[i][0]) < scm.ToInt(rows[j][0]) })
					if len(rows) != want || len(current.Columns) != len(columns) {
						t.Fatalf("growth lost the accepted transaction: rows=%v columns=%v", rows, current.Columns)
					}
					for i, row := range rows {
						payload := "after-ddl"
						if i == 0 {
							payload = "row-00000001"
						} else if i == 1 {
							payload = "before-ddl"
						}
						expected := []scm.Scmer{scm.NewInt(int64(i + 1)), scm.NewString(payload)}
						if commit {
							expected = append(expected, scm.NewInt(7))
						}
						if !scm.Equal(scm.NewSlice(row), scm.NewSlice(expected)) {
							t.Fatalf("growth recovered an incorrect value: got=%v want=%v", row, expected)
						}
					}
				}
				check(tbl)
				db := tbl.schema
				cold := newDatabase()
				cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
				databases.Set(db.Name, cold)
				defer func() {
					databases.Set(db.Name, db)
					publishCatalogShards(db.Name, &db.catalog.live)
					cold.closeTransactionLog()
				}()
				cold.ensureLoaded()
				check(cold.GetTable(tbl.Name))
			})
		}
	}
}

func TestSchemaNamespaceFollowsOriginalTransaction(t *testing.T) {
	for _, mode := range []TxMode{TxACID, TxCursorStability} {
		for _, commit := range []bool{false, true} {
			for _, savepoint := range []bool{false, true} {
				for _, drop := range []bool{false, true} {
					t.Run(fmt.Sprintf("mode-%d-commit-%t-savepoint-%t-drop-%t", mode, commit, savepoint, drop), func(t *testing.T) {
						tbl, _ := createDurabilityTestTable(t, "schema-namespace-transaction", 1)
						db, original := tbl.schema, tbl.Name
						tx := NewTxContext(mode)
						defer func() {
							if tx.State == TxActive {
								tx.Rollback()
							}
						}()
						tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("before-ddl")}}, nil, scm.NewNil(), false, nil, tx)
						sp := tx.CreateSavepoint()
						if drop {
							DropTable(tx, db.Name, original, false)
						} else {
							RenameTable(tx, db.Name, original, "renamed")
						}
						if db.GetTable(original) != nil {
							t.Fatal("original name still resolves inside the schema transaction")
						}
						if savepoint {
							tx.RollbackToSavepoint(sp)
						}
						if commit {
							if err := tx.Commit(); err != nil {
								t.Fatal(err)
							}
						} else {
							tx.Rollback()
						}
						wantName := original
						if commit && !savepoint {
							wantName = "renamed"
							if drop {
								wantName = ""
							}
						}
						cold := newDatabase()
						cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
						previous := databases.Set(db.Name, cold)
						defer func() {
							databases.Set(db.Name, previous)
							publishCatalogShards(db.Name, &db.catalog.live)
							cold.closeTransactionLog()
						}()
						cold.ensureLoaded()
						for _, current := range []*database{db, cold} {
							for _, name := range []string{original, "renamed"} {
								relation := current.GetTable(name)
								if name != wantName {
									if relation != nil {
										t.Fatalf("unexpected relation %s after transaction/restart", name)
									}
									continue
								}
								if relation == nil {
									t.Fatalf("missing admitted relation %s after transaction/restart", name)
								}
								_, rows := catalogShardRecords(nil, relation.ActiveShards()[0], []string{"id", "payload"}, nil)
								want := [][]scm.Scmer{{scm.NewInt(1), scm.NewString("row-00000001")}}
								if commit {
									want = append(want, []scm.Scmer{scm.NewInt(2), scm.NewString("before-ddl")})
								}
								if !reflect.DeepEqual(rows, want) {
									t.Fatalf("data/schema decision mismatch: %v", rows)
								}
							}
						}
					})
				}
			}
		}
	}
}

func TestSchemaEngineFollowsOriginalTransaction(t *testing.T) {
	equalRows := func(a, b [][]scm.Scmer) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !scm.Equal(scm.NewSlice(a[i]), scm.NewSlice(b[i])) {
				return false
			}
		}
		return true
	}
	for _, change := range [][2]PersistencyMode{{Safe, Memory}, {Memory, Safe}, {Sloppy, Safe}, {Safe, Logged}} {
		for _, mode := range []TxMode{TxACID, TxCursorStability} {
			for _, commit := range []bool{false, true} {
				t.Run(fmt.Sprintf("engine-%d-%d-tx-%d-commit-%t", change[0], change[1], mode, commit), func(t *testing.T) {
					tbl, _ := createDurabilityTestTable(t, "schema-engine-original-transaction", 1)
					tbl.ChangeEngine(nil, change[0])
					tx := NewTxContext(mode)
					defer func() {
						if tx.State == TxActive {
							tx.Rollback()
						}
					}()
					tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("prior-dml")}}, nil, scm.NewNil(), false, nil, tx)
					tbl.ChangeEngine(tx, change[1])
					if commit {
						if err := tx.Commit(); err != nil {
							t.Fatal(err)
						}
					} else {
						tx.Rollback()
					}
					wantMode, count := change[0], 1
					if commit {
						wantMode, count = change[1], 2
					}
					_, live := catalogShardRecords(nil, tbl.ActiveShards()[0], []string{"id", "payload"}, nil)
					want := [][]scm.Scmer{{scm.NewInt(1), scm.NewString("row-00000001")}}
					if commit {
						want = append(want, []scm.Scmer{scm.NewInt(2), scm.NewString("prior-dml")})
					}
					if tbl.PersistencyMode != wantMode || !equalRows(live, want) {
						t.Fatalf("engine/data rollback disagreement: %v/%v, want %v/%v", tbl.PersistencyMode, live, wantMode, want)
					}
					db := tbl.schema
					cold := newDatabase()
					cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
					previous := databases.Set(db.Name, cold)
					defer func() {
						databases.Set(db.Name, previous)
						publishCatalogShards(db.Name, &db.catalog.live)
						cold.closeTransactionLog()
					}()
					cold.ensureLoaded()
					restored := cold.GetTable(tbl.Name)
					_, rows := catalogShardRecords(nil, restored.ActiveShards()[0], []string{"id", "payload"}, nil)
					if wantMode != Safe && wantMode != Logged {
						count, want = 0, nil
					}
					if restored.PersistencyMode != wantMode || len(rows) != count || count != 0 && !equalRows(rows, want) {
						t.Fatalf("restarted engine/data disagreement: %v/%v, want %v/%v", restored.PersistencyMode, rows, wantMode, want)
					}
				})
			}
		}
	}
}

type catalogDecisionFailureLog struct {
	PersistenceLogfile
	flush bool
}

func (log *catalogDecisionFailureLog) Write(entry any) {
	if !log.flush {
		panic(&PersistenceFailure{Backend: "test", Operation: "decision.write", Err: fmt.Errorf("injected decision failure")})
	}
	log.PersistenceLogfile.Write(entry)
}

func (log *catalogDecisionFailureLog) Flush(durable bool) {
	if log.flush {
		panic(&PersistenceFailure{Backend: "test", Operation: "wal.flush", Err: fmt.Errorf("injected WAL failure")})
	}
	log.PersistenceLogfile.Flush(durable)
}

func TestSchemaAndDataUseOneCrossDatabaseDecision(t *testing.T) {
	for _, mode := range []TxMode{TxACID, TxCursorStability} {
		for _, failure := range []string{"none", "before-decision", "after-decision"} {
			t.Run(fmt.Sprintf("mode-%d-%s", mode, failure), func(t *testing.T) {
				first, _ := createDurabilityTestTable(t, "schema-decision-first", 1)
				CreateDatabase("schema-decision-second", false)
				second, _ := CreateTable(nil, "schema-decision-second", "items", Safe, false)
				second.CreateColumn(nil, "id", "INT", nil, nil)
				secondDB := second.schema
				defer func() {
					secondDB.closeTransactionLog()
					databases.Remove(secondDB.Name)
					publishCatalogShards(secondDB.Name, nil)
				}()
				tx := NewTxContext(mode)
				defer func() {
					if tx.State == TxActive {
						tx.Rollback()
					}
				}()
				first.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("prior-dml")}}, nil, scm.NewNil(), false, nil, tx)
				second.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil, tx)
				first.CreateColumn(tx, "added", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(9)})
				if failure == "before-decision" {
					shard := second.ActiveShards()[0]
					original := shard.logfile
					shard.logfile = &catalogDecisionFailureLog{PersistenceLogfile: original, flush: true}
					defer func() { shard.logfile = original }()
				}
				if failure == "after-decision" {
					secondDB.initializeTransactionLog()
					original := secondDB.transactionLog
					secondDB.transactionLog = &catalogDecisionFailureLog{PersistenceLogfile: original}
					defer func() { secondDB.transactionLog = original }()
				}
				err := tx.Commit()
				accepted := failure != "before-decision"
				if failure == "none" && err != nil || failure != "none" && err == nil {
					t.Fatalf("commit result for %s: %v", failure, err)
				}
				if commitOutcomeUnknown(err) != (failure == "after-decision") {
					t.Fatalf("incorrect native decision certainty: %v", err)
				}
				for _, original := range []*database{first.schema, secondDB} {
					cold := newDatabase()
					cold.Name, cold.persistence, cold.srState = original.Name, original.persistence, COLD
					databases.Set(original.Name, cold)
					func() {
						defer func() {
							databases.Set(original.Name, original)
							publishCatalogShards(original.Name, &original.catalog.live)
							cold.closeTransactionLog()
						}()
						cold.ensureLoaded()
						relation := cold.GetTable("items")
						columns, count := []string{"id"}, 0
						if original == first.schema {
							columns, count = []string{"id", "payload"}, 1
							if accepted {
								columns = append(columns, "added")
							}
						}
						if accepted {
							count++
						}
						_, rows := catalogShardRecords(nil, relation.ActiveShards()[0], columns, nil)
						if len(rows) != count || len(relation.Columns) != len(columns) {
							t.Fatalf("%s recovered rows/columns %v/%d; want %d/%d", original.Name, rows, len(relation.Columns), count, len(columns))
						}
						if accepted && original == first.schema && (rows[1][0].Int() != 2 || rows[1][1].String() != "prior-dml" || rows[0][2].Int() != 9 || rows[1][2].Int() != 9) {
							t.Fatalf("native data and column recovery disagreed: %v", rows)
						}
						if accepted && original == secondDB && rows[0][0].Int() != 7 {
							t.Fatalf("other participant lost the shared decision: %v", rows)
						}
					}()
				}
			})
		}
	}
}

func TestCatalogStableNativeIdentity(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-native-identity", 1)
	db := tbl.schema
	identity := tbl.NativeIdentity(nil)
	if identity.IsNil() {
		t.Fatal("new relation lacks its durable incarnation")
	}
	if _, err := uuid.Parse(identity.String()); err != nil {
		t.Fatal(err)
	}
	tx := NewTxContext(TxACID)
	RenameTable(tx, db.Name, tbl.Name, "renamed")
	if !scm.Equal(tbl.NativeIdentity(tx), identity) {
		t.Fatal("transactional rename changed the native incarnation")
	}
	tx.Rollback()
	if !scm.Equal(tbl.NativeIdentity(nil), identity) {
		t.Fatal("rollback changed the native incarnation")
	}
	RebuildTable(tbl, true, false)
	if !scm.Equal(tbl.NativeIdentity(nil), identity) {
		t.Fatal("physical shard rebuild changed the native incarnation")
	}
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
	databases.Set(db.Name, cold)
	defer func() {
		databases.Set(db.Name, db)
		publishCatalogShards(db.Name, &db.catalog.live)
		cold.closeTransactionLog()
	}()
	cold.ensureLoaded()
	loaded := cold.GetTable(tbl.Name)
	if !scm.Equal(loaded.NativeIdentity(nil), identity) {
		t.Fatal("restart changed the native incarnation")
	}
	DropTable(nil, cold.Name, loaded.Name, false)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("dropped handle supplied an authoritative identity")
			}
		}()
		loaded.NativeIdentity(nil)
	}()
	replacement, _ := CreateTable(nil, cold.Name, loaded.Name, Safe, false)
	if value := replacement.NativeIdentity(nil); value.IsNil() || scm.Equal(value, identity) {
		t.Fatal("replacement reused its predecessor's incarnation")
	}
}

func TestCatalogVersionTwoUpgradesMissingNativeIdentity(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-v2-native-identity", 1)
	db := tbl.schema
	db.save(nil, nil) // the predecessor fixture has no format-3 mutation WAL
	var document map[string]json.RawMessage
	if err := json.Unmarshal(db.persistence.ReadSchema(), &document); err != nil {
		t.Fatal(err)
	}
	root := readCatalogRoot(db.persistence.ReadSchema())
	previous := root.Shards["tables"]
	root.Version = 2
	document["catalog"], _ = json.Marshal(root)
	legacy, _ := json.Marshal(document)
	db.persistence.WriteSchema(legacy)
	catalogBackendDatabase().persistence.RemoveColumn(previous, "runtime_identity")
	cold := newDatabase()
	cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
	databases.Set(db.Name, cold)
	defer func() {
		databases.Set(db.Name, db)
		publishCatalogShards(db.Name, &db.catalog.live)
		cold.closeTransactionLog()
	}()
	cold.ensureLoaded()
	if !cold.GetTable(tbl.Name).NativeIdentity(nil).IsNil() {
		t.Fatal("legacy catalog fabricated an incarnation or used the legacy JSON identity")
	}
	cold.save(nil, nil)
	upgraded := readCatalogRoot(cold.persistence.ReadSchema())
	if upgraded.Version != catalogFormatVersion || upgraded.Shards["tables"] == previous {
		t.Fatal("v3 root reused a base without the introduced native column")
	}
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = cold.Name, cold.persistence, COLD
	databases.Set(db.Name, restored)
	defer restored.closeTransactionLog()
	restored.ensureLoaded()
	if !restored.GetTable(tbl.Name).NativeIdentity(nil).IsNil() {
		t.Fatal("rewriting a legacy declaration assigned an invented identity")
	}
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
	rows := db.catalogRowsLocked(nil)
	db.schemalock.Unlock()
	root := &catalogRoot{Version: 1, Shards: make(map[string]string)}
	for i := range catalogRelations {
		shard := newCatalogShard(db, catalogRelationID(i), rows[i])
		// The legacy writer used the tenant backend. Modern constructors bind
		// system storage, so the historical fixture must choose its placement.
		shard.catalogHome = db
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
	restored.save(nil, nil)
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
	tbl.schema.saveLockedAndUnlock(nil, schemaSaveFsync, tbl)
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
	if err := r.installSchema(uuid.NewString(), cold.Name, engine, accepted, false); err != nil {
		t.Fatal(err)
	}
	if cold.GetTable(tbl.Name) == nil {
		t.Fatal("accepted peer schema did not initialize the cold database")
	}
	cold.schemalock.Lock()
	rows := cold.catalogRowsLocked(nil)
	cold.schemalock.Unlock()
	if len(rows[catalogTables]) == 0 {
		t.Fatal("peer publication did not establish canonical catalog rows")
	}
}

func TestCatalogMosiReadUsesPublishedNativeGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-native-mosi", 0)
	db := tbl.schema
	db.schemalock.Lock()
	rows := db.catalogRowsLocked(nil)
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
	received.catalogPrivate.Store(false)
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
	_, visible := catalogShardRecords(nil, source, catalogRelationColumns(catalogColumns), nil)
	if len(visible) == 0 || visible[0][nameIndex].String() != "from-peer" {
		t.Fatalf("native catalog scan ignored MOSI generation: private=%v owner=%q manager=%v current=%p wanted=%p visible=%v", source.catalogPrivate.Load(), source.catalogOwner, source.replicationManager() != nil, cell.current.Load(), received, visible)
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
	tbl.CreateColumn(nil, "added", "INT", nil, nil)
	db.save(nil, nil) // explicitly compact the accepted catalog WAL
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
	_, visible := catalogShardRecords(nil, previous, catalogRelationColumns(catalogColumns), nil)
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
	db.save(nil, nil) // the completed bases have not opened their first WAL yet
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
		tbl.CreateColumn(nil, "after_barrier", "INT", nil, nil)
	}()
	if barrier.calls != 1 || counting.calls.Load() != 0 || string(engine.ReadSchema()) != string(previous) {
		t.Fatal("failed system namespace sync replaced the recoverable tenant root")
	}
	if len(tbl.Columns) != 2 {
		t.Fatal("failed namespace admission retained the rejected RAM declaration")
	}
	_, rejected := catalogShardRecords(nil, db.catalog.live[catalogColumns], catalogRelationColumns(catalogColumns), []string{tbl.Name})
	if len(rejected) != 2 {
		t.Fatal("failed namespace admission retained the rejected catalog declaration")
	}
	barrier.before = func() {
		if counting.calls.Load() != 0 {
			t.Error("retry published its root before syncing the system namespace")
		}
	}
	tbl.CreateColumn(nil, "after_barrier", "INT", nil, nil)
	// First-use SAFE WALs for the table and column bases need namespace
	// barriers, including retrying the failed table-WAL namespace admission.
	if barrier.calls != 3 || counting.calls.Load() != 1 {
		t.Fatalf("retry did not durably order the native namespace before its tenant root: barriers=%d roots=%d", barrier.calls, counting.calls.Load())
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

func TestCatalogPublicationPreservesConcurrentCheckpointIdentity(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-publication-identity", 0)
	db := tbl.schema
	r := replicaRecoveryFixture()
	rows := db.catalogRowsLocked(nil)
	shard := buildDiskShard(db, catalogColumns, rows[catalogColumns], true)
	resource, home := shard.replicationResource(), shard.home()
	started, done := make(chan struct{}), make(chan bool, 1)
	go func() {
		close(started)
		consistent := true
		for range 10000 {
			consistent = consistent && shard.replicationResource() == resource && shard.home() == home
			_ = shard.replicationManager()
		}
		done <- consistent
	}()
	<-started
	for range 32 {
		if err := r.replaceCatalogShard(resource, nil, shard); err != nil {
			t.Fatal(err)
		}
	}
	if !<-done || shard.catalogPrivate.Load() {
		t.Fatal("native publication changed checkpoint identity or retained private authority")
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
	rows := db.catalogRowsLocked(nil)
	initial := db.catalog.live[catalogColumns]
	cell := r.shardCell(initial)
	// A handover can advance immutable storage while retaining exactly the
	// same declarations. The last durable root still names its predecessor.
	successor := buildDiskShard(db, catalogColumns, rows[catalogColumns], true)
	if err := r.replaceCatalogShard(initial.replicationResource(), initial, successor); err != nil {
		t.Fatal(err)
	}
	db.save(nil, nil)
	current := cell.resolved().current.Load()
	canonical := r.canonicalResource(initial.replicationResource())
	if current.uuid.String() != canonical.Shard {
		t.Fatalf("checkpoint reused historical payload %s under newer authority %s", current.uuid, canonical.Shard)
	}
	root := readCatalogRoot(engine.ReadSchema())
	if root.Shards[catalogRelations[catalogColumns].name] != current.uuid.String() {
		t.Fatal("durable root and native current publication disagree")
	}
	_, visible := catalogShardRecords(nil, current, catalogRelationColumns(catalogColumns), nil)
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
	rows := db.catalogRowsLocked(nil)
	var nameIndex int
	for index, field := range catalogRelations[catalogColumns].columns() {
		if field.key == "Name" {
			nameIndex = index
		}
	}
	rows[catalogColumns][0][nameIndex] = scm.NewString("checkpoint_definition")
	db.schemalock.Lock()
	db.editCatalogLocked(nil, &rows, 1<<catalogColumns, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		db.commitSchemaLocked(nil, true, db.schemalock.Unlock)
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
	rows := db.catalogRowsLocked(nil)
	for index, field := range catalogRelations[catalogColumns].columns() {
		if field.key == "Name" {
			rows[catalogColumns][0][index] = scm.NewString("checkpoint_definition")
		}
	}
	db.schemalock.Lock()
	db.editCatalogLocked(nil, &rows, 1<<catalogColumns, nil)
	live := db.catalog.live[catalogColumns]
	cell := r.shardCell(live)
	handover, err := cell.ownstate.BeginHandover(context.Background(), cell.drainWrites, true)
	if err != nil {
		db.schemalock.Unlock()
		t.Fatal(err)
	}
	defer handover.Release()
	db.commitSchemaLocked(nil, true, db.schemalock.Unlock)
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
	other, _ := CreateTable(nil, db.Name, "unrelated", Safe, false)
	other.CreateColumn(nil, "kept", "TEXT", nil, []scm.Scmer{scm.NewString("default"), scm.NewString("unchanged <&> Gröẞe")})
	read := func() []string {
		var keys []string
		for i := catalogTables; i < catalogProcedures; i++ {
			_, rows := catalogShardRecords(nil, db.catalog.live[i], catalogRelationColumns(i), []string{other.Name})
			for _, row := range rows {
				keys = append(keys, catalogRowKey(catalogRelations[i].columns(), row))
			}
		}
		sort.Strings(keys)
		return keys
	}
	before := read()
	tbl.AlterColumn(nil, "payload", "comment", scm.NewString("changed"))
	tbl.CreateColumn(nil, "temporary", "INT", nil, nil)
	tbl.DropColumn(nil, "temporary")
	if after := read(); len(before) == 0 || !reflect.DeepEqual(before, after) {
		t.Fatal("scoped catalog edits changed another table's authoritative rows")
	}
	_, rows := catalogShardRecords(nil, db.catalog.live[catalogColumns], catalogRelationColumns(catalogColumns), []string{tbl.Name})
	if len(rows) != 2 {
		t.Fatalf("scoped add/drop left %d columns, want 2", len(rows))
	}
}

func TestCatalogWalSurvivesRestartAndDecisionCompaction(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-wal-decision-compaction", 1)
	db := tbl.schema
	db.save(nil, nil)
	base := db.catalog.live[catalogColumns].uuid
	_, expected := catalogShardRecords(nil, tbl.ActiveShards()[0], []string{"id", "payload"}, nil)
	tbl.AlterColumn(nil, "payload", "comment", scm.NewString("durable catalog WAL"))
	if db.catalog.live[catalogColumns].uuid != base {
		t.Fatal("one metadata edit recompressed its complete catalog base")
	}
	checkCold := func() {
		t.Helper()
		cold := newDatabase()
		cold.Name, cold.persistence, cold.srState = db.Name, db.persistence, COLD
		databases.Set(db.Name, cold)
		defer func() {
			databases.Set(db.Name, db)
			publishCatalogShards(db.Name, &db.catalog.live)
			cold.closeTransactionLog()
		}()
		cold.ensureLoaded()
		loaded := cold.GetTable(tbl.Name)
		comment := scm.Apply(loaded.ShowColumn("payload", false), scm.NewString("Comment"))
		if !scm.Equal(comment, scm.NewString("durable catalog WAL")) {
			t.Fatal("cold catalog lost its accepted metadata")
		}
		_, actual := catalogShardRecords(nil, loaded.ActiveShards()[0], []string{"id", "payload"}, nil)
		if len(actual) != len(expected) {
			t.Fatal("restart changed user row count")
		}
		for i, row := range expected {
			if !scm.Equal(scm.NewSlice(row), scm.NewSlice(actual[i])) {
				t.Fatal("restart changed user values")
			}
		}
	}
	checkCold()
	if errors, complete := rebuildDatabaseAndCompact(db, true, false, false); len(errors) != 0 || !complete {
		t.Fatalf("physical checkpoint/decision compaction failed: %v, complete=%v", errors, complete)
	}
	if db.catalog.live[catalogColumns].uuid == base || len(db.catalog.live[catalogColumns].deltaRows()) != 0 {
		t.Fatal("decision-log compaction did not seal the catalog WAL first")
	}
	checkCold()
}

func TestCatalogCommentTouchesOnlyItsRelation(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-comment-write-scope", 0)
	tbl.schema.save(nil, nil)
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	tbl.AlterColumn(tx, "payload", "comment", scm.NewString("changed"))
	tx.prepareSchemaCommit()
	var relations []string
	tx.touchedShards.Range(func(key, _ any) bool {
		relations = append(relations, key.(*storageShard).t.Name)
		return true
	})
	sort.Strings(relations)
	if !reflect.DeepEqual(relations, []string{"columns"}) {
		t.Fatalf("one column comment changed unrelated catalog relations: %v", relations)
	}
}

func TestCatalogDeclarationEpochIgnoresPhysicalStateAndRowOrder(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-derived-declaration-epoch", 0)
	rows := tbl.schema.catalogRowsLocked(nil)
	before := catalogDefinitionRevision(&rows)
	for relation := range rows {
		for i, j := 0, len(rows[relation])-1; i < j; i, j = i+1, j-1 {
			rows[relation][i], rows[relation][j] = rows[relation][j], rows[relation][i]
		}
	}
	for i, field := range catalogRelations[catalogTables].columns() {
		if field.key == "Auto_increment" {
			rows[catalogTables][0][i] = scm.NewInt(999)
		}
		if field.key == "Shards" {
			rows[catalogTables][0][i] = scm.NewSlice([]scm.Scmer{scm.NewString(uuid.NewString())})
		}
	}
	if actual := catalogDefinitionRevision(&rows); actual != before {
		t.Fatal("physical checkpoint or row ordering changed the declaration epoch")
	}
	for i, field := range catalogRelations[catalogColumns].columns() {
		if field.key != "Comment" {
			continue
		}
		previous := rows[catalogColumns][0][i]
		rows[catalogColumns][0][i] = scm.NewString("new declaration")
		if catalogDefinitionRevision(&rows) == before {
			t.Fatal("column declaration did not change its epoch")
		}
		rows[catalogColumns][0][i] = previous
	}
	rows[catalogDatabases][0][1] = scm.NewString("frontend-owned-definition")
	if catalogDefinitionRevision(&rows) == before {
		t.Fatal("frontend-owned declaration did not change its epoch")
	}
}
