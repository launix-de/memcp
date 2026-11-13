/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "strconv"
import "strings"
import "time"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

func nativeCreateTable(a ...scm.Scmer) scm.Scmer {
	ifnotexists := len(a) > 4 && scm.ToBool(a[4])
	currentTx := scm.NewNil()
	if len(a) > 5 {
		currentTx = a[5]
	}
	tx := scmerToTxContext(currentTx)
	db := GetDatabase(scm.String(a[0]))
	if db == nil {
		panic("database " + scm.String(a[0]) + " does not exist")
	}
	db.ensureLoaded()
	tblName := scm.String(a[1])

	// Software contract:
	// createtable(..., ifnotexists=true) is the hot-path guard used by
	// planner-generated keytable/prejoin initializers. When the table is
	// already present, the fast path must stay extremely cheap:
	// - no schema save
	// - no table-definition rebuild
	// - no options/columns parsing
	//
	// We still confirm under schemalock after the optimistic probe so the
	// result stays race-free under concurrent creators.
	if ifnotexists {
		if existing := db.tables.Get(tblName); existing != nil {
			if !existing.Ephemeral && (existing.OnInit.Load() != nil || existing.onInitComplete.Load()) && scmerToTxContext(currentTx).retainQueryTable(existing) {
				// This also reruns persisted oninit for data-empty MEMORY/CACHE
				// tables after restart. The local barrier affects no other table.
				existing.awaitCreationInitialization(currentTx)
				atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
				return scm.NewBool(false)
			}
			// Legacy CACHE/MEMORY schemas did not persist oninit. Parse this
			// invocation so its current closed callback can repair the table.
		}
	}
	// parse options only after the fast existing-table probe
	options := mustScmerSlice(a[3], "options")
	var autoIncrement uint64
	engine := Settings.DefaultEngine
	collation := ""
	charset := ""
	comment := ""
	nameIgnoreCase := false
	oninit := scm.NewNil()
	ephemeral := false
	var relationOwner *relationScope
	scopeKey := ""
	for i := 0; i+1 < len(options); i += 2 {
		key := scm.String(options[i])
		val := options[i+1]
		switch key {
		case "engine":
			engine = scm.String(val)
		case "ephemeral":
			ephemeral = scm.ToBool(val)
		case "scope_key":
			scopeKey = scm.String(val)
		case "scope":
			relationOwner = relationScopeFromValue(val)
		case "collation":
			collation = scm.String(val)
		case "charset":
			charset = scm.String(val)
		case "comment":
			comment = scm.String(val)
		case "name_ignorecase":
			nameIgnoreCase = scm.ToBool(val)
		case "auto_increment":
			nextAutoIncrement, _ := strconv.ParseUint(scm.String(val), 0, 64)
			if nextAutoIncrement > 0 {
				autoIncrement = nextAutoIncrement - 1
			}
		case "oninit":
			oninit = val
		default:
			panic("unknown option: " + key)
		}
	}
	// Inline self references have no joint CREATE/FK validation owner yet.
	// Reject before allocating the table, opening logs or publishing callbacks.
	for _, definition := range mustScmerSlice(a[2], "columns") {
		fields := mustScmerSlice(definition, "column definition")
		if len(fields) > 3 && scm.String(fields[0]) == "foreign" {
			parent := scm.String(fields[3])
			if parent == tblName || (nameIgnoreCase && scm.CaseFoldKey(parent) == scm.CaseFoldKey(tblName)) {
				panic("inline self foreign declarations require a separate transactional ALTER")
			}
		}
	}
	pm := parsePersistencyMode(engine)
	// Catalog publication is currently independent of transaction row
	// decisions. Reject new persistent declarations before taking schema
	// ownership, opening shard logs, or invoking initialization callbacks.
	// Query caches and scope-owned RAM relations have separate native
	// lifetime contracts; materialization must retain those contracts.
	queryCache := pm == Cache && strings.HasPrefix(tblName, ".")
	scopedMemory := ephemeral && pm == Memory && relationOwner != nil
	construction := tx != nil && !queryCache && !scopedMemory
	staged := construction && !tx.autoCommit
	if construction {
		if tx.publicCatalogEntered.Load() {
			panic("public and private catalog declarations cannot be mixed")
		}
		done := tx.beginCatalogConstruction(false)
		defer done()
		// A public phase can enter between the optimistic check and admission.
		// Its phase store shares this same resource-scope owner.
		if tx.publicCatalogEntered.Load() {
			panic("public and private catalog declarations cannot be mixed")
		}
	}
	if staged && tx.coordinator != nil {
		return tx.forwardCatalogCreate(a)
	}
	if staged {
		if existing := db.GetTableForTx(tblName, nameIgnoreCase, tx); existing != nil && ifnotexists {
			return scm.NewBool(false)
		}
		if !oninit.IsNil() || len(getCreateTableTriggers(db.Name, tblName)) != 0 {
			panic("transactional table creation with initialization callbacks is not supported")
		}
		for _, coldef := range mustScmerSlice(a[2], "columns") {
			def := mustScmerSlice(coldef, "column definition")
			if len(def) != 0 && scm.String(def[0]) == "foreign" {
				panic("transactional foreign-key creation is not supported")
			}
		}
		for _, existing := range db.tables.GetAll() {
			for _, foreign := range existing.Foreign {
				if foreign.Tbl2 == tblName {
					panic("transactional forward foreign-key creation is not supported")
				}
			}
		}
	}
	if ephemeral {
		if pm != Memory || relationOwner == nil {
			panic("ephemeral relations require MEMORY and a relation scope")
		}
		finish := relationOwner.beginUse()
		defer finish()
	} else if relationOwner != nil {
		panic("relation scope requires ephemeral MEMORY")
	}
	var scopedChange *relationSchemaChange
	scopedComplete := false
	if relationOwner != nil && relationOwner.transactionalSchema {
		if !oninit.IsNil() || len(getCreateTableTriggers(db.Name, tblName)) != 0 {
			panic("transactional relation schemas require declarations without initialization callbacks")
		}
		for _, definition := range mustScmerSlice(a[2], "columns") {
			fields := mustScmerSlice(definition, "column definition")
			if len(fields) > 0 && scm.String(fields[0]) == "foreign" {
				panic("transactional relation schemas do not cross foreign catalog owners")
			}
		}
		change, finish := relationOwner.beginSchemaChange(scopeKey, tx, true, ifnotexists)
		defer finish()
		if change == nil {
			return scm.NewBool(false)
		}
		scopedChange = change
		defer func() {
			if !scopedComplete {
				change.cancel()
			}
		}()
	} else if scopeKey != "" {
		panic("relation scope slot requires transactional schema policy")
	}
	if relationOwner != nil && relationOwner.dataPolicy == relationIndependentData {
		if !oninit.IsNil() || len(getCreateTableTriggers(db.Name, tblName)) != 0 || activeReplication(db) != nil || sharedClusterDatabase(db.Name) {
			panic("independent private data requires a local MEMORY relation without external initialization")
		}
		for _, definition := range mustScmerSlice(a[2], "columns") {
			fields := mustScmerSlice(definition, "column definition")
			if len(fields) != 0 && scm.String(fields[0]) == "foreign" {
				panic("independent private data does not cross foreign data owners")
			}
		}
	}
	// Acquire schema ownership before native lifecycle locks or opening
	// unpublished shard logs. A remote schema grant may need to install
	// metadata through that same lifecycle boundary.
	var schemaAccess WriteAccess
	if !(pm == Cache && strings.HasPrefix(tblName, ".")) {
		schemaAccess = db.GetWrite(nil)
		defer func() { schemaAccess.Release() }()
	}
	if staged {
		// Recheck under the native schema publication capability. No concurrent
		// public forward-FK declaration may be silently mutated by this profile.
		if !oninit.IsNil() || len(getCreateTableTriggers(db.Name, tblName)) != 0 {
			panic("transactional table creation with initialization callbacks is not supported")
		}
		for _, existing := range db.tables.GetAll() {
			for _, foreign := range existing.Foreign {
				if foreign.Tbl2 == tblName {
					panic("transactional forward foreign-key creation is not supported")
				}
			}
		}
		tx.beginCatalogCreate(db)
	}
	db.persistenceLifecycle.RLock()
	persistenceLifecycleLocked := true
	unlockPersistenceLifecycle := func() {
		if persistenceLifecycleLocked {
			persistenceLifecycleLocked = false
			db.persistenceLifecycle.RUnlock()
		}
	}
	defer unlockPersistenceLifecycle()

	newTable := new(table)
	newTable.schema = db
	newTable.Name = tblName
	newTable.RuntimeIdentity = uuid.NewString()
	published := false
	if staged {
		identity := tx.catalogIdentity.Load()
		newTable.CreatorTxID = tx.durableID()
		newTable.CatalogDecisionID = identity.Decision
		newTable.CreatorIncarnation = identity.Incarnation
		newTable.CreatorNamespace = identity.Namespace
		newTable.creator.Store(tx.transactionState)
		defer func() {
			if !published {
				for _, shard := range newTable.ActiveShards() {
					discardUnpublishedShard(shard)
				}
			}
		}()
	}
	newTable.PersistencyMode = pm
	newTable.Ephemeral = ephemeral
	newTable.relationLifetime = relationOwner
	if scopedChange != nil {
		newTable.ScopedSchema = true
		newTable.CreatorTxID = tx.durableID()
		newTable.creator.Store(tx.transactionState)
		scopedChange.created = newTable
	}
	if relationOwner != nil && relationOwner.dataPolicy == relationIndependentData {
		newTable.dataOwner = relationOwner
	}
	newTable.ShardMode = ShardModeFree
	newTable.lastAccessed = uint64(time.Now().UnixNano())
	newTable.Shards = make([]*storageShard, 1)
	newTable.Shards[0] = NewShard(newTable)
	newTable.publishTopologyLocked()
	newTable.Collation = collation
	newTable.Charset = charset
	newTable.Comment = comment
	newTable.Auto_increment = autoIncrement
	if !oninit.IsNil() {
		closedOnInit := scm.CloseProcedure(oninit)
		newTable.OnInit.Store(&closedOnInit)
	}

	for _, coldef := range mustScmerSlice(a[2], "columns") {
		def := mustScmerSlice(coldef, "column definition")
		if len(def) == 0 {
			continue
		}
		head := scm.String(def[0])
		switch head {
		case "unique":
			cols := scmerSliceToStrings(mustScmerSlice(def[2], "unique columns"))
			key := uniqueKey{Id: scm.String(def[1]), Cols: cols}
			if len(def) > 3 {
				key = uniqueKeyOptions(key, def[3])
			}
			newTable.Unique = append(newTable.Unique, key)
		case "foreign":
			cols1 := scmerSliceToStrings(mustScmerSlice(def[2], "foreign cols1"))
			cols2 := scmerSliceToStrings(mustScmerSlice(def[4], "foreign cols2"))
			var updatemode foreignKeyMode
			if len(def) > 5 {
				updatemode = getForeignKeyMode(def[5])
			}
			var deletemode foreignKeyMode
			if len(def) > 6 {
				deletemode = getForeignKeyMode(def[6])
			}
			newTable.Foreign = append(newTable.Foreign, ownForeignPayload(foreignKey{
				Id:         scm.String(def[1]),
				Tbl1:       newTable.Name,
				Cols1:      cols1,
				Tbl2:       scm.String(def[3]),
				Cols2:      cols2,
				Updatemode: updatemode,
				Deletemode: deletemode,
			}))
		case "column":
			colname := scm.String(def[1])
			typename := scm.String(def[2])
			dimVals := mustScmerSlice(def[3], "column dimensions")
			dimensions := make([]int, len(dimVals))
			for i, d := range dimVals {
				dimensions[i] = scm.ToInt(d)
			}
			typeparams := mustScmerSlice(def[4], "column typeparams")
			column, ok := newTable.createColumnLocked(colname, typename, dimensions, typeparams, false)
			if !ok {
				panic("column " + newTable.Name + "." + colname + " already exists")
			}
			if len(def) != 5 && len(def) != 7 {
				panic("column declaration requires five or seven fields")
			}
			if len(def) == 7 {
				for i := 0; i+1 < len(typeparams); i += 2 {
					if scm.String(typeparams[i]) == "default" || scm.String(typeparams[i]) == "default_expression" {
						panic("declared computation cannot have a default")
					}
				}
				declareColumnComputor(column, def[5], def[6])
			}
		default:
			panic("unknown column definition: " + head)
		}
	}
	newTable.bindDeclaredComputations()
	validateUniqueConstraintNames(newTable.Unique)
	newTable.publishShowColumnsSnapshot(true)

	existing := func() *table {
		db.schemalock.Lock()
		resolved := false
		defer func() {
			// Folded lookup can reject an ambiguous namespace. Only a
			// successful lookup hands schema ownership to publication below.
			if !resolved {
				db.schemalock.Unlock()
			}
		}()
		existing := db.tables.Get(tblName)
		if nameIgnoreCase {
			key := scm.CaseFoldKey(tblName)
			existing = nil
			for _, candidate := range db.tables.GetAll() {
				if scm.CaseFoldKey(candidate.Name) == key {
					if existing != nil && existing != candidate {
						panic("ambiguous table name: " + tblName)
					}
					existing = candidate
				}
			}
		}
		resolved = true
		return existing
	}()
	if existing != nil {
		if scopedChange != nil {
			db.schemalock.Unlock()
			panic("scoped schema declaration requires a fresh physical generation")
		}
		if !existing.visibleIn(tx, db.catalogView().version) {
			db.schemalock.Unlock()
			panic("table name is reserved by another native transaction")
		}
		if existing.Ephemeral && (relationOwner == nil || !relationOwner.owns(existing)) {
			db.schemalock.Unlock()
			panic("relation scope does not own the existing relation")
		}
		if !scmerToTxContext(currentTx).retainQueryTable(existing) {
			db.schemalock.Unlock()
			panic("cache table retired during creation")
		}
		// Keep the hot ifnotexists path free of schema saves. Planner-created
		// helper tables deliberately re-issue createtable on every query; if
		// the table already exists, "created=false" is the only signal the
		// caller needs to skip collect/materialization work.
		atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
		if existing.OnInit.Load() == nil && !oninit.IsNil() {
			closedOnInit := scm.CloseProcedure(oninit)
			existing.OnInit.Store(&closedOnInit)
			db.saveLockedAndUnlock(schemaSaveBuffered)
		} else {
			db.schemalock.Unlock()
		}
		unlockPersistenceLifecycle()
		// The competing creator may have published the table after our
		// optimistic probe. Its table-local barrier includes oninit.
		existing.awaitCreationInitialization(currentTx)
		if !ifnotexists {
			panic("Table " + tblName + " already exists")
		}
		return scm.NewBool(false)
	}

	// Lock before publication. Every if-not-exists observer therefore waits
	// until oninit and registered create-table triggers have both completed.
	for _, c := range newTable.Columns {
		if c.DefaultName != "" && (newTable.hasDeclarationName(c.DefaultName, c, true) || db.hasDeclarationNameLocked(c.DefaultName, nil)) {
			db.schemalock.Unlock()
			panic("declaration name already exists: " + c.DefaultName)
		}
	}
	for _, foreign := range newTable.Foreign {
		if reserved := db.tables.Get(foreign.Tbl2); reserved != nil && !reserved.committedAt(db.catalogView().version) {
			db.schemalock.Unlock()
			panic("foreign-key target is reserved by a native transaction")
		}
	}
	// Retain the unpublished carrier before cache registration can offer
	// it for eviction. The caller still has initialization work to do.
	scmerToTxContext(currentTx).retainQueryTable(newTable)
	newTable.creationMu.Lock()
	if staged || scopedChange != nil {
		createTableTriggerMu.Lock()
		for _, registration := range Settings.CreateTableTriggers {
			if registration.Schema == db.Name && registration.Table == tblName {
				createTableTriggerMu.Unlock()
				newTable.creationMu.Unlock()
				db.schemalock.Unlock()
				panic("transactional table creation with initialization callbacks is not supported")
			}
		}
	}
	if prev := db.tables.Set(tblName, newTable); prev != nil {
		if staged || scopedChange != nil {
			createTableTriggerMu.Unlock()
		}
		newTable.creationMu.Unlock()
		db.schemalock.Unlock()
		panic("Table " + tblName + " already exists")
	}
	published = true
	if staged {
		tx.appendCatalogIntent(newTable)
	}
	if !newTable.isHiddenFromShowTables() {
		db.publishTableNamesLocked()
	}
	if staged || scopedChange != nil {
		createTableTriggerMu.Unlock()
	}
	if relationOwner != nil {
		relationOwner.adopt(newTable)
	}
	if scopedChange != nil {
		scopedChange.publish(newTable)
	}

	if !staged {
		for _, fk := range newTable.Foreign {
			if t2 := newTable.schema.GetTable(fk.Tbl2); t2 != nil {
				t2.Foreign = append(t2.Foreign, sharedForeignMirror(fk))
				installFKTriggers(newTable.schema, newTable, t2, fk)
				t2.publishShowColumnsSnapshot(true)
			}
		}
		// add constraints that are added onto us (forward-declared FKs)
		forwardedConstraints := false
		for _, t2 := range newTable.schema.tables.GetAll() {
			if t2 != newTable {
				for _, foreign := range t2.Foreign {
					if foreign.Tbl2 == newTable.Name {
						newTable.Foreign = append(newTable.Foreign, sharedForeignMirror(foreign))
						forwardedConstraints = true
						installFKTriggers(newTable.schema, t2, newTable, foreign)
					}
				}
			}
		}
		// Only new incoming declarations need another metadata publication.
		if forwardedConstraints {
			newTable.publishShowColumnsSnapshot(true)
		}
	}

	mode := newTable.schemaSaveMode()
	if newTable.isEphemeralQueryTable() {
		mode = schemaSaveBuffered
	}
	if staged {
		db.schemalock.Unlock()
	} else {
		db.saveLockedAndUnlock(mode)
	}
	// Dependency/oninit callbacks acquire their own schema rights.
	// End publication ownership before invoking nested DDL.
	schemaAccess.Release()
	schemaAccess = WriteAccess{}
	unlockPersistenceLifecycle()
	if !staged {
		registerCreatedTable(newTable)
	}
	var thisCallPanic any
	func() {
		defer func() {
			thisCallPanic = recover()
			// A transient per-query cancellation ("query killed") is
			// not a genuine initialization failure: caching it as
			// creationPanic would permanently break this table for
			// every future caller, even though a fresh attempt would
			// very likely succeed. Only deterministic OnInit/trigger
			// failures are cached and replayed to later callers.
			if thisCallPanic != nil && thisCallPanic != "query killed" {
				newTable.creationPanic = thisCallPanic
			}
			// Publish the shared outcome before releasing concurrent creators.
			newTable.creationMu.Unlock()
		}()
		if !oninit.IsNil() {
			scm.Apply(oninit, currentTx)
		}
		newTable.onInitComplete.Store(true)
		if !staged && scopedChange == nil {
			executeRegisteredCreateTableTriggers(newTable, nil)
		}
	}()
	if thisCallPanic != nil {
		panic(thisCallPanic)
	}
	scopedComplete = true
	return scm.NewBool(true)
}
