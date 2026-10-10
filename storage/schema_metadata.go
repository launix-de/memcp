/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sort"
import "github.com/launix-de/memcp/scm"

// Frontend payloads share the existing schema lock and schema persistence.
// Table information is obtained only on explicit reads, using the normal SHOW
// definitions rather than publishing another catalog on every DDL save.
func (db *database) schemaMetadata() scm.Scmer {
	db.ensureLoaded()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	tables := db.tables.GetAll()
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	values := make([]scm.Scmer, 0, len(tables))
	for _, table := range tables {
		if !table.isHiddenFromShowTables() {
			values = append(values, showBuildMeta(db, table, true))
		}
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("revision"), scm.NewInt(int64(db.metadataRevision)),
		scm.NewString("value"), db.metadataValue,
		scm.NewString("tables"), scm.NewSlice(values),
	})
}

func metadataOptions(options scm.Scmer) (expected uint64, value scm.Scmer, guarded bool, update bool) {
	value = scm.NewNil()
	if options.IsNil() {
		return
	}
	items := mustScmerSlice(options, "schema publication")
	if len(items)%2 != 0 {
		panic("schema publication requires key/value pairs")
	}
	for i := 0; i < len(items); i += 2 {
		switch scm.String(items[i]) {
		case "revision":
			n := scm.ToInt(items[i+1])
			if n < 0 {
				panic("negative schema revision")
			}
			expected, guarded = uint64(n), true
		case "value":
			value, update = items[i+1], true
		default:
			panic("unknown schema publication option")
		}
	}
	if update && !guarded {
		panic("metadata publication requires an expected revision")
	}
	return
}

func (db *database) checkMetadataPublicationLocked(options scm.Scmer) error {
	expected, _, guarded, _ := metadataOptions(options)
	if guarded && expected != db.metadataRevision {
		return fmt.Errorf("stale schema metadata revision: expected %d, current %d", expected, db.metadataRevision)
	}
	return nil
}
func (db *database) applyMetadataPublicationLocked(options scm.Scmer) {
	_, value, _, update := metadataOptions(options)
	if update {
		db.metadataValue = value
	}
}

func initSchemaMetadataBuiltins(en scm.Env) {
	scm.Declare(&en, &scm.Declaration{Name: "schema_metadata", Fn: func(args ...scm.Scmer) scm.Scmer {
		db := GetDatabase(scm.String(args[0]))
		if db == nil {
			return scm.NewNil()
		}
		return db.schemaMetadata()
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "reads one immutable frontend metadata generation", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "database"}}, Return: &scm.TypeDescriptor{Kind: "list|nil"}}})
	scm.Declare(&en, &scm.Declaration{Name: "publish_schema_metadata", Fn: func(args ...scm.Scmer) scm.Scmer {
		db := GetDatabase(scm.String(args[0]))
		if db == nil {
			panic("database does not exist")
		}
		db.ensureLoaded()
		options := scm.NewSlice([]scm.Scmer{scm.NewString("revision"), args[1], scm.NewString("value"), args[2]})
		db.schemalock.Lock()
		if err := db.checkMetadataPublicationLocked(options); err != nil {
			db.schemalock.Unlock()
			panic(err)
		}
		db.applyMetadataPublicationLocked(options)
		db.saveLockedAndUnlock(schemaSaveFsync)
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "publishes opaque frontend metadata against an expected schema revision", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "database"}, {Kind: "int", Label: "revision"}, {Kind: "any", Label: "value"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}

// ddlPublication decodes no policy; the locked publication boundary validates it.
func ddlPublication(args []scm.Scmer, index int) scm.Scmer {
	if len(args) > index {
		return args[index]
	}
	return scm.NewNil()
}
