/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
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

func TestCatalogCheckpointCannotOverwriteDefinitionsFromRuntime(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "catalog-checkpoint-authority", 1)
	db := tbl.schema
	// Deliberately corrupt only the derived runtime view. A maintenance save
	// must export the admitted definition, rather than silently adopting this.
	tbl.Comment = "runtime-only divergence"
	tbl.Columns[1].Comment = "runtime column divergence"
	db.save()
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
	_, before := catalogShardRecords(shard, catalogRelationColumns(catalogColumns))
	callback := shard.UpdateFunction(0, true, false, nil, nil)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a retained catalog mutation callback was writable")
			}
		}()
		callback()
	}()
	_, after := catalogShardRecords(shard, catalogRelationColumns(catalogColumns))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected callback changed catalog definitions")
	}
}
