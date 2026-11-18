/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/launix-de/memcp/scm"
)

// system_schema is the relational form of every database's declarations.
// Each catalog relation is one table, partitioned by database with the
// "database-v1" method: partition i is database Pivots[i]. Catalog files
// use the system backend. There is no central shard list; a
// database's schema.json root names its own catalog shards.
//
// Catalog rows are authoritative. Legacy schema.json records are imported
// once and exported for old readers. Scalars keep their native
// type, scalar lists become list cells and everything else, including code,
// is one JSON value per cell. A database document reconstructed from the rows
// is decoded by the same loader as a legacy schema.json.
const systemSchemaName = "system_schema"

// catalogFormatVersion is stored in the schema.json root. A reader accepts
// every version up to its own; newer catalogs fall back to the legacy part.
const catalogFormatVersion = 2

type catalogKind uint8

const (
	catalogText catalogKind = iota
	catalogInt
	catalogBool
	catalogFloat
	catalogList
	catalogJSON
)

func (kind catalogKind) sqlType() string {
	switch kind {
	case catalogInt:
		return "BIGINT"
	case catalogBool:
		return "BOOLEAN"
	case catalogFloat:
		return "DOUBLE"
	}
	return "TEXT"
}

// catalogField maps one key of a legacy JSON record to a catalog column.
type catalogField struct {
	key    string
	column string
	kind   catalogKind
}

type catalogRelationID int

const (
	catalogDatabases catalogRelationID = iota
	catalogTables
	catalogColumns
	catalogKeys
	catalogKeyColumns
	catalogForeignKeys
	catalogForeignKeyColumns
	catalogTriggers
	catalogProcedures
	catalogRelationCount
)

// catalogRelation describes one system_schema table. Identity columns follow
// schema_name; they order child records and are not part of the record.
type catalogRelation struct {
	name     string
	identity []catalogField
	fields   []catalogField
}

func (r *catalogRelation) columns() []catalogField {
	result := []catalogField{{column: "schema_name", kind: catalogText}}
	result = append(result, r.identity...)
	return append(result, r.fields...)
}

var catalogRelations = func() [catalogRelationCount]catalogRelation {
	text := func(name string) catalogField { return catalogField{column: name, kind: catalogText} }
	ordinal := func(name string) catalogField { return catalogField{column: name, kind: catalogInt} }
	owner := []catalogField{text("table_name"), ordinal("ordinal")}
	return [catalogRelationCount]catalogRelation{
		catalogDatabases: {name: "databases"},
		catalogTables: {name: "tables", fields: catalogRecordFields(reflect.TypeOf(persistedTable{}),
			map[string]catalogKind{"Shards": catalogList, "PShards": catalogList},
			"Columns", "Unique", "Foreign", "Triggers")},
		catalogColumns: {name: "columns", identity: owner, fields: catalogRecordFields(reflect.TypeOf(persistedColumn{}), nil)},
		catalogKeys:    {name: "keys", identity: owner, fields: catalogRecordFields(reflect.TypeOf(uniqueKey{}), nil, "Cols")},
		catalogKeyColumns: {name: "key_columns", identity: []catalogField{
			text("table_name"), ordinal("key_ordinal"), ordinal("ordinal"), text("column_name")}},
		catalogForeignKeys: {name: "foreign_keys", identity: owner, fields: catalogRecordFields(reflect.TypeOf(foreignKey{}), nil, "Cols1", "Cols2")},
		catalogForeignKeyColumns: {name: "foreign_key_columns", identity: []catalogField{
			text("table_name"), ordinal("foreign_key_ordinal"), ordinal("ordinal"), text("column1"), text("column2")}},
		catalogTriggers:   {name: "triggers", identity: owner, fields: catalogRecordFields(reflect.TypeOf(TriggerDescription{}), nil)},
		catalogProcedures: {name: "procedures", identity: []catalogField{ordinal("ordinal")}, fields: catalogRecordFields(reflect.TypeOf(ProcedureDescription{}), nil)},
	}
}()

// catalogRecordFields lists the JSON keys encoding/json writes for record,
// with the same embedding and shadowing rules. Fields named in children are
// stored in their own relation.
func catalogRecordFields(record reflect.Type, kinds map[string]catalogKind, children ...string) []catalogField {
	skip := make(map[string]bool, len(children))
	for _, child := range children {
		skip[child] = true
	}
	var result []catalogField
	seen := make(map[string]bool)
	var visit func(reflect.Type, bool)
	visit = func(record reflect.Type, embedded bool) {
		var nested []reflect.Type
		for i := 0; i < record.NumField(); i++ {
			field := record.Field(i)
			tag := field.Tag.Get("json")
			name, _, _ := strings.Cut(tag, ",")
			if tag == "-" {
				continue
			}
			if field.Anonymous && name == "" {
				embeddedType := field.Type
				if embeddedType.Kind() == reflect.Pointer {
					embeddedType = embeddedType.Elem()
				}
				if embeddedType.Kind() == reflect.Struct {
					nested = append(nested, embeddedType)
					continue
				}
			}
			if !field.IsExported() {
				continue
			}
			if name == "" {
				name = field.Name
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			if skip[name] {
				continue
			}
			kind, overridden := kinds[name]
			if !overridden {
				kind = catalogKindOf(field.Type)
			}
			result = append(result, catalogField{key: name, column: catalogColumnName(name), kind: kind})
		}
		// encoding/json gives shallower fields precedence over embedded ones.
		for _, embeddedType := range nested {
			visit(embeddedType, true)
		}
	}
	visit(record, false)
	return result
}

var scmerType = reflect.TypeOf(scm.Scmer{})

func catalogKindOf(typ reflect.Type) catalogKind {
	if typ == scmerType {
		return catalogJSON
	}
	if _, custom := reflect.New(typ).Interface().(json.Marshaler); custom {
		// Enumerations such as TriggerTiming encode as names: keep them text.
		if encoded, err := json.Marshal(reflect.New(typ).Interface()); err == nil && len(encoded) > 0 && encoded[0] == '"' {
			return catalogText
		}
		return catalogJSON
	}
	switch typ.Kind() {
	case reflect.String:
		return catalogText
	case reflect.Bool:
		return catalogBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return catalogInt
	case reflect.Float32, reflect.Float64:
		return catalogFloat
	case reflect.Slice:
		switch kind := catalogKindOf(typ.Elem()); kind {
		case catalogText, catalogBool, catalogInt, catalogFloat:
			return catalogList
		}
	}
	return catalogJSON
}

// catalogColumnName turns JSON keys such as "PersistencyMode" or
// "Auto_increment" into snake_case SQL identifiers.
func catalogColumnName(key string) string {
	var b strings.Builder
	runes := []rune(key)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && runes[i-1] != '_' {
			previousLower := unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if previousLower || (unicode.IsUpper(runes[i-1]) && nextLower) {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// catalogCell decodes one JSON value into the cell representation of kind.
// JSON null and absent keys are NULL.
func catalogCell(kind catalogKind, raw json.RawMessage) scm.Scmer {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return scm.NewNil()
	}
	switch kind {
	case catalogText:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			panic(err)
		}
		return scm.NewString(value)
	case catalogInt:
		var value json.Number
		if err := json.Unmarshal(raw, &value); err != nil {
			panic(err)
		}
		n, err := value.Int64()
		if err != nil {
			panic(err)
		}
		return scm.NewInt(n)
	case catalogBool:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			panic(err)
		}
		return scm.NewBool(value)
	case catalogFloat:
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil {
			panic(err)
		}
		return scm.NewFloat(value)
	case catalogList:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			panic(err)
		}
		values := make([]scm.Scmer, len(items))
		for i, item := range items {
			var value any
			decoder := json.NewDecoder(bytes.NewReader(item))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				panic(err)
			}
			values[i] = scm.TransformFromJSON(value)
		}
		return scm.NewSlice(values)
	}
	return scm.NewString(string(raw))
}

// catalogRaw is the inverse of catalogCell. It encodes by the field kind, not
// by the storage representation a column happened to compress into.
func catalogRaw(kind catalogKind, cell scm.Scmer) json.RawMessage {
	if cell.IsNil() {
		return nil
	}
	var value any
	switch kind {
	case catalogText:
		value = cell.String()
	case catalogInt:
		value = scm.ToInt(cell)
	case catalogBool:
		value = scm.ToBool(cell)
	case catalogFloat:
		value = cell.Float()
	case catalogList:
		items := cell.Slice()
		raw := make([]json.RawMessage, len(items))
		for i, item := range items {
			encoded, err := json.Marshal(item)
			if err != nil {
				panic(err)
			}
			raw[i] = encoded
		}
		value = raw
	default:
		return json.RawMessage(cell.String())
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

// catalogRows holds one database's rows of every relation, in canonical order.
type catalogRows [catalogRelationCount][][]scm.Scmer

func (rows *catalogRows) add(relation catalogRelationID, schema string, identity []scm.Scmer, record map[string]json.RawMessage) {
	r := &catalogRelations[relation]
	row := make([]scm.Scmer, 0, 1+len(r.identity)+len(r.fields))
	row = append(row, scm.NewString(schema))
	row = append(row, identity...)
	for _, field := range r.fields {
		row = append(row, catalogCell(field.kind, record[field.key]))
	}
	rows[relation] = append(rows[relation], row)
}

type catalogDocument struct {
	Name       string                     `json:"name"`
	Tables     map[string]json.RawMessage `json:"tables"`
	Procedures []json.RawMessage          `json:"procedures,omitempty"`
}

func decodeCatalogRecords(raw json.RawMessage) []map[string]json.RawMessage {
	var records []map[string]json.RawMessage
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &records); err != nil {
			panic(err)
		}
	}
	return records
}

func decodeCatalogStrings(raw json.RawMessage) []string {
	var values []string
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &values); err != nil {
			panic(err)
		}
	}
	return values
}

// catalogRowsFromDocument splits a legacy schema.json document into rows.
func catalogRowsFromDocument(document []byte) (rows catalogRows) {
	var doc catalogDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		panic(err)
	}
	schema := doc.Name
	rows.add(catalogDatabases, schema, nil, nil)
	names := make([]string, 0, len(doc.Tables))
	for name := range doc.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(doc.Tables[name], &record); err != nil {
			panic(err)
		}
		if record["Name"] == nil {
			record["Name"], _ = json.Marshal(name)
		}
		rows.add(catalogTables, schema, nil, record)
		owner := scm.NewString(name)
		for i, column := range decodeCatalogRecords(record["Columns"]) {
			rows.add(catalogColumns, schema, []scm.Scmer{owner, scm.NewInt(int64(i))}, column)
		}
		for i, key := range decodeCatalogRecords(record["Unique"]) {
			rows.add(catalogKeys, schema, []scm.Scmer{owner, scm.NewInt(int64(i))}, key)
			for j, column := range decodeCatalogStrings(key["Cols"]) {
				rows.add(catalogKeyColumns, schema, []scm.Scmer{owner, scm.NewInt(int64(i)), scm.NewInt(int64(j)), scm.NewString(column)}, nil)
			}
		}
		for i, key := range decodeCatalogRecords(record["Foreign"]) {
			rows.add(catalogForeignKeys, schema, []scm.Scmer{owner, scm.NewInt(int64(i))}, key)
			local, remote := decodeCatalogStrings(key["Cols1"]), decodeCatalogStrings(key["Cols2"])
			for j := 0; j < max(len(local), len(remote)); j++ {
				column1, column2 := scm.NewNil(), scm.NewNil()
				if j < len(local) {
					column1 = scm.NewString(local[j])
				}
				if j < len(remote) {
					column2 = scm.NewString(remote[j])
				}
				rows.add(catalogForeignKeyColumns, schema, []scm.Scmer{owner, scm.NewInt(int64(i)), scm.NewInt(int64(j)), column1, column2}, nil)
			}
		}
		for i, trigger := range decodeCatalogRecords(record["Triggers"]) {
			rows.add(catalogTriggers, schema, []scm.Scmer{owner, scm.NewInt(int64(i))}, trigger)
		}
	}
	for i, procedure := range doc.Procedures {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(procedure, &record); err != nil {
			panic(err)
		}
		rows.add(catalogProcedures, schema, []scm.Scmer{scm.NewInt(int64(i))}, record)
	}
	return rows
}

// catalogRecord rebuilds the JSON record of one row.
func catalogRecord(relation catalogRelationID, row []scm.Scmer) map[string]json.RawMessage {
	r := &catalogRelations[relation]
	record := make(map[string]json.RawMessage, len(r.fields))
	offset := 1 + len(r.identity)
	for i, field := range r.fields {
		if raw := catalogRaw(field.kind, row[offset+i]); raw != nil {
			record[field.key] = raw
		}
	}
	return record
}

func catalogOrdinal(row []scm.Scmer, column int) int {
	return int(scm.ToInt(row[column]))
}

func marshalCatalogValue(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

// catalogDocumentFromRows is the inverse of catalogRowsFromDocument. Child
// records are placed by their ordinals, independent of row order.
func catalogDocumentFromRows(schema string, rows *catalogRows) []byte {
	type tableRecords struct {
		record   map[string]json.RawMessage
		children [4][]map[string]json.RawMessage
		keyCols  map[int][]string
		fkCols   map[int][2][]string
	}
	tables := make(map[string]*tableRecords)
	lookup := func(name string) *tableRecords {
		entry := tables[name]
		if entry == nil {
			panic("catalog row references unknown table " + schema + "." + name)
		}
		return entry
	}
	for _, row := range rows[catalogTables] {
		record := catalogRecord(catalogTables, row)
		var name string
		if err := json.Unmarshal(record["Name"], &name); err != nil {
			panic(err)
		}
		tables[name] = &tableRecords{record: record, keyCols: map[int][]string{}, fkCols: map[int][2][]string{}}
	}
	place := func(list *[]map[string]json.RawMessage, ordinal int, record map[string]json.RawMessage) {
		for len(*list) <= ordinal {
			*list = append(*list, nil)
		}
		(*list)[ordinal] = record
	}
	children := []catalogRelationID{catalogColumns, catalogKeys, catalogForeignKeys, catalogTriggers}
	for slot, relation := range children {
		for _, row := range rows[relation] {
			entry := lookup(row[1].String())
			place(&entry.children[slot], catalogOrdinal(row, 2), catalogRecord(relation, row))
		}
	}
	setList := func(cols []string, ordinal int, value string) []string {
		for len(cols) <= ordinal {
			cols = append(cols, "")
		}
		cols[ordinal] = value
		return cols
	}
	for _, row := range rows[catalogKeyColumns] {
		entry := lookup(row[1].String())
		key := catalogOrdinal(row, 2)
		entry.keyCols[key] = setList(entry.keyCols[key], catalogOrdinal(row, 3), row[4].String())
	}
	for _, row := range rows[catalogForeignKeyColumns] {
		entry := lookup(row[1].String())
		key, ordinal := catalogOrdinal(row, 2), catalogOrdinal(row, 3)
		pair := entry.fkCols[key]
		if !row[4].IsNil() {
			pair[0] = setList(pair[0], ordinal, row[4].String())
		}
		if !row[5].IsNil() {
			pair[1] = setList(pair[1], ordinal, row[5].String())
		}
		entry.fkCols[key] = pair
	}
	childKeys := []string{"Columns", "Unique", "Foreign", "Triggers"}
	doc := catalogDocument{Name: schema, Tables: make(map[string]json.RawMessage, len(tables))}
	for name, entry := range tables {
		for i, key := range entry.children[1] {
			if key != nil {
				key["Cols"] = marshalCatalogValue(entry.keyCols[i])
			}
		}
		for i, key := range entry.children[2] {
			if key != nil {
				pair := entry.fkCols[i]
				key["Cols1"], key["Cols2"] = marshalCatalogValue(pair[0]), marshalCatalogValue(pair[1])
			}
		}
		for slot, key := range childKeys {
			for i, child := range entry.children[slot] {
				if child == nil {
					panic(fmt.Sprintf("catalog of %s.%s misses %s record %d", schema, name, key, i))
				}
			}
			entry.record[key] = marshalCatalogValue(entry.children[slot])
		}
		doc.Tables[name] = marshalCatalogValue(entry.record)
	}
	procedures := make([]map[string]json.RawMessage, 0)
	for _, row := range rows[catalogProcedures] {
		place(&procedures, catalogOrdinal(row, 1), catalogRecord(catalogProcedures, row))
	}
	for _, procedure := range procedures {
		if procedure == nil {
			panic("catalog of " + schema + " misses a procedure record")
		}
		doc.Procedures = append(doc.Procedures, marshalCatalogValue(procedure))
	}
	return marshalCatalogValue(doc)
}
