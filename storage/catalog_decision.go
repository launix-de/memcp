/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "context"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Kinds distinguish declaration admission from SQL spelling. Defaults and
// immutable callable sources can join local ACID rows; enforcement declarations
// retain their earlier validation boundary, and private tables their own owner.
type catalogDeclarationKind uint8

const (
	catalogCreateKind catalogDeclarationKind = iota
	catalogDefaultKind
	catalogSourceKind
	catalogTableDropKind
	catalogForeignDropKind
	catalogPublicKeyKind
	catalogForeignReplacementKind
	catalogEnforcementKind
	catalogColumnKind
	catalogStatementTriggerKind
)

// Identity is allocated before the first unpublished declaration. It is part of
// the existing native transaction capability, not a name-based permission.
type catalogIdentity struct {
	database     *database
	latePublic   bool // local existing ACID owner; no mode, snapshot or WAL identity replacement
	lateCreates  bool // private generations join the original local ACID row decision
	Incarnation  string
	Database     string
	Decision     string
	Installation string
	Namespace    string
}

type catalogCommit struct {
	database             *database
	access               WriteAccess
	record               *catalogDecisionRecord
	prepared             bool
	participantsPrepared bool
	publicPublication    *publicForeignPublication
	appendDDL            []*table
}

func (tx *transactionState) commitAuthorityID() string {
	if identity := tx.catalogIdentity.Load(); identity != nil {
		return (&catalogDecisionRecord{Decision: identity.Decision, Installation: identity.Installation, Namespace: identity.Namespace, Incarnation: identity.Incarnation}).authorityID()
	}
	return tx.durableID()
}

// Existing local ACID effects can share the private catalog decision. Completed
// CS public reads can enter its clean owner; visible CS effects cannot be
// converted into an atomic schema/data publication.
func (tx *TxContext) beginCatalogCreate(db *database) {
	tx.mu.Lock()
	late := tx.catalogIdentity.Load() == nil && tx.Mode == TxACID &&
		(tx.rowAccessed.Load() || len(tx.shardTransactions()) != 0 || len(tx.participants) != 0 || tx.readViews.Load() != nil)
	tx.mu.Unlock()
	if late {
		tx.getOrComputeResource(lateCreateResourceKey, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			lease := db.admitLocalCatalog(false)
			accepted := false
			defer func() {
				if !accepted {
					_ = lease.Close()
				}
			}()
			tx.beginCatalogDeclaration(db, catalogCreateKind)
			accepted = true
			return scm.NewAny(lease)
		}))
	}
	tx.beginCatalogDeclaration(db, catalogCreateKind)
}

// Caller owns the existing construction scope and transaction mutexes. A
// completed public read has no row decision to convert. Actual CS undo journals
// survive savepoint/rebuild changes until the transaction's terminal decision;
// participantEpoch likewise remembers remote writer enrollment after settlement.
func (tx *TxContext) catalogReadOnlyLocked(sessionLocks bool) bool {
	if sessionLocks || tx.State != TxActive || tx.decisionStarted || tx.rollbackOnly.Load() ||
		(tx.Mode != TxCursorStability && tx.Mode != TxACID) || tx.prepared != nil || tx.catalogCommit != nil ||
		tx.serializable.Load() != nil || tx.coordinator != nil || tx.remoteObject != nil || tx.remoteInvocation ||
		tx.replicationActive || tx.replicationWriting || tx.relationData != nil || tx.mutationOperation != nil ||
		tx.shards.Load() != nil || tx.readViews.Load() != nil || tx.visibilityProjections.Load() != nil ||
		len(tx.participants) != 0 || tx.participantEpoch != 0 || len(tx.endpoints) != 0 || len(tx.writeHeld) != 0 ||
		len(tx.repartitionDeletes) != 0 || tx.recoveryNeedsDatabaseDecision.Load() ||
		tx.remoteActive != 0 || tx.migrations != 0 || tx.mutationOperations != 0 || tx.checkedMutation != nil ||
		tx.mutationSavepoints != 0 || tx.mutationAdmissionClosed || tx.relationDataOrigins != 0 ||
		len(tx.relationSchemaChanges) != 0 || len(tx.relationSchemaRetirements) != 0 ||
		len(tx.queryColumns) != 0 || len(tx.queryWorkReleases) != 0 || len(tx.queryReaderReleases) != 0 {
		return false
	}
	touched := false
	tx.touchedShards.Range(func(_, _ any) bool { touched = true; return false })
	tx.recoveryDecisionDatabases.Range(func(_, _ any) bool { touched = true; return false })
	if touched {
		return false
	}
	return tx.resourceScope().catalogReadOnlyLocked()
}

// The existing scope mutex protects completed native capabilities as well as
// unfinished factories. A writer can be retained before its first row journal.
func (scope *transactionResources) catalogReadOnlyLocked() bool {
	if scope.closed || scope.deciding {
		return false
	}
	for _, entry := range scope.entries {
		select {
		case <-entry.done:
		default:
			return false // unfinished native resource authority is not a read proof
		}
		if lease, ok := entry.value.Any().(*tableWriteLease); ok && !lease.declaration {
			return false // preserve an escaped native writer, even before its first row
		}
	}
	return true
}

func (tx *TxContext) beginCatalogDeclaration(db *database, kind catalogDeclarationKind) {
	public := kind == catalogDefaultKind || kind == catalogSourceKind || kind == catalogTableDropKind || kind == catalogForeignDropKind || kind == catalogForeignReplacementKind || kind == catalogPublicKeyKind || kind == catalogColumnKind || kind == catalogStatementTriggerKind
	privateCreate := kind == catalogCreateKind
	if tx.hasTableDrops() && (!public || kind == catalogStatementTriggerKind) {
		panic("table removal phase admits only default or source declarations")
	}
	// Construction admission owns effect policy. Merely preparing a private
	// catalog identity is not a declaration mutation: validation-only producers
	// mark their effect at the sole successful amendment publication.
	// Validate before allocating a new native cluster identity or changing a
	// legacy schema checkpoint: rejected operations have no catalog effects.
	scope := tx.resourceScope()
	lock := func() { scope.mu.Lock(); tx.mu.Lock() }
	unlock := func() { tx.mu.Unlock(); scope.mu.Unlock() }
	sessionLocks := tx.SessionState.HasLocks()
	lock()
	if tx.State != TxActive || tx.decisionStarted {
		unlock()
		panic("transaction is not open for catalog changes")
	}
	identity := tx.catalogIdentity.Load()
	readOnly := tx.catalogReadOnlyLocked(sessionLocks)
	if tx.Mode == TxCursorStability && !readOnly {
		unlock()
		panic("transactional catalog declarations must precede native row access")
	}
	dirty := false
	if identity == nil {
		dirty = tx.rowAccessed.Load() || len(tx.shardTransactions()) != 0 || len(tx.participants) != 0 || tx.readViews.Load() != nil
	}
	// Only prior native data admission enters this mixed row/declaration
	// profile. Clean declaration owners retain their enforcement boundary.
	late := (public || privateCreate) && tx.Mode == TxACID && identity == nil && dirty && !readOnly
	if tx.rowAccessed.Load() && !tx.publicCatalogEntered.Load() && !late &&
		!(identity != nil && identity.lateCreates && privateCreate) && !readOnly {
		unlock()
		panic("transactional catalog declarations must precede native row access")
	}
	if identity != nil {
		valid := identity.Database == db.Name && identity.Incarnation == db.incarnationID() && identity.database == db
		allowed := (!identity.latePublic || public) && (!identity.lateCreates || privateCreate)
		var entryError error
		if valid && allowed {
			if identity.latePublic {
				entryError = tx.validateLatePublicOwnerLocked(db, false)
			} else if identity.lateCreates {
				entryError = tx.validateLateCreateOwnerLocked(db, false)
			}
		}
		unlock()
		if !valid {
			panic("transactional catalog creation is limited to its original database")
		}
		if !allowed {
			if identity.latePublic {
				panic("late catalog phase admits only default or source declarations")
			}
			panic("late private catalog phase admits only new table declarations")
		}
		if entryError != nil {
			panic(entryError)
		}
		return
	}
	if late {
		var entryError error
		if privateCreate {
			entryError = tx.validateLateCreateOwnerLocked(db, true)
		} else {
			entryError = tx.validateLatePublicOwnerLocked(db, true)
		}
		if entryError != nil {
			unlock()
			panic(entryError)
		}
	}
	unlock()
	if dirty && !late && !readOnly {
		panic("transactional catalog creation must precede native row access")
	}
	if sharedClusterDatabase(db.Name) {
		tx.ensureClusterIdentity()
	}
	installation, err := localNodeIdentity()
	if err != nil {
		panic(err)
	}
	namespace := ""
	if sharedClusterDatabase(db.Name) {
		namespace, err = persistedCatalogNamespace(sharedCommitBackend())
		if err != nil {
			panic(err)
		}
	}
	if db.incarnationID() == "" {
		db.schemalock.Lock()
		changed := db.incarnationID() == ""
		if changed {
			id := uuid.NewString()
			db.incarnation.Store(&id)
		}
		db.schemalock.Unlock()
		if changed {
			db.save()
		}
	}
	sessionLocks = tx.SessionState.HasLocks()
	lock()
	defer unlock()
	if tx.State != TxActive || tx.decisionStarted {
		panic("transaction is not open for catalog changes")
	}
	identity = tx.catalogIdentity.Load()
	readOnly = tx.catalogReadOnlyLocked(sessionLocks)
	if tx.Mode == TxCursorStability && !readOnly {
		panic("transactional catalog declarations must precede native row access")
	}
	if tx.rowAccessed.Load() && !tx.publicCatalogEntered.Load() && !late &&
		!(identity != nil && identity.lateCreates && privateCreate) && !readOnly {
		panic("transactional catalog declarations must precede native row access")
	}
	if identity != nil {
		if identity.Database != db.Name || identity.Incarnation != db.incarnationID() || identity.database != db {
			panic("transactional catalog creation is limited to its original database")
		}
		if identity.latePublic && !public {
			panic("late catalog phase admits only default or source declarations")
		}
		if identity.lateCreates && !privateCreate {
			panic("late private catalog phase admits only new table declarations")
		}
		if identity.latePublic {
			if err := tx.validateLatePublicOwnerLocked(db, false); err != nil {
				panic(err)
			}
		} else if identity.lateCreates {
			if err := tx.validateLateCreateOwnerLocked(db, false); err != nil {
				panic(err)
			}
		}
		return
	}
	if late {
		var entryError error
		if privateCreate {
			entryError = tx.validateLateCreateOwnerLocked(db, true)
		} else {
			entryError = tx.validateLatePublicOwnerLocked(db, true)
		}
		if entryError != nil {
			panic(entryError)
		}
	} else if !readOnly && (tx.rowAccessed.Load() || len(tx.shardTransactions()) != 0 || len(tx.participants) != 0 || tx.readViews.Load() != nil) {
		panic("transactional catalog creation must precede native row access")
	}
	if !late {
		// Promotion retains the original binding's decision metadata. Public
		// row reads are current-committed; no read history is discarded.
		tx.Mode = TxACID
	}
	tx.catalogIdentity.Store(&catalogIdentity{database: db, latePublic: late && public, lateCreates: late && privateCreate, Incarnation: db.incarnationID(), Database: db.Name, Decision: uuid.NewString(), Installation: installation, Namespace: namespace})
}

// Enter this declaration-only phase before publishing its first resource. The
// existing construction count orders it against private CREATE admission; the
// initial public entry rejects an unfinished competitor before FK effects.
func (tx *TxContext) enterPublicCatalogPhase() {
	scope := tx.resourceScope()
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if tx.publicCatalogEntered.Load() {
		return
	}
	if scope.closed || scope.deciding || scope.constructing != 1 {
		panic("public catalog phase conflicts with unfinished native construction; retry statement")
	}
	// A private constructor may have published its intent and retired its
	// construction count since the caller's optimistic mixed-profile check.
	// Its count retires under this same scope owner after appending the intent.
	if tx.catalogIntents.Load() != nil {
		panic("public and private catalog declarations cannot be mixed")
	}
	tx.publicCatalogEntered.Store(true)
	if identity := tx.catalogIdentity.Load(); identity != nil && identity.latePublic {
		scope.publicRowsEntered = true
	}
}

func (tx *TxContext) appendCatalogIntent(tbl *table) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	var intents []*table
	if previous := tx.catalogIntents.Load(); previous != nil {
		intents = append(intents, (*previous)...)
	}
	intents = append(intents, tbl)
	tx.catalogIntents.Store(&intents)
}

func (tx *TxContext) acquireCatalogPublication() {
	if !tx.hasCatalogIntents() || tx.catalogCommit != nil {
		return
	}
	identity := tx.catalogIdentity.Load()
	db := GetDatabase(identity.Database)
	if db == nil || db != identity.database || db.incarnationID() != identity.Incarnation {
		panic("transaction catalog database was removed or replaced")
	}
	var appendDDL []*table
	retained := false
	defer func() {
		if !retained {
			for _, held := range appendDDL {
				held.ddlMu.Unlock()
			}
		}
	}()
	for _, d := range tx.publicColumnAppends() {
		t := db.tables.Get(d.Table)
		if t == nil {
			panic("nullable append endpoint was removed")
		}
		if len(appendDDL) != 0 && appendDDL[len(appendDDL)-1] == t {
			continue
		}
		if tx.retainedColumnDDL(t) {
			continue
		}
		// Successful staging retained the validated lease and proved no own
		// endpoint rows. Decision admission has drained builders/producers, so this
		// wait holds no input that maintenance needs in order to finish.
		t.ddlMu.Lock()
		appendDDL = append(appendDDL, t)
	}
	access := db.GetWrite(tx)
	if GetDatabase(identity.Database) != db || db.incarnationID() != identity.Incarnation {
		access.Release()
		panic("transaction catalog database was replaced")
	}
	tx.catalogCommit = &catalogCommit{database: db, access: access, appendDDL: appendDDL}
	retained = true
}
func (tx *TxContext) releaseCatalogPublication() {
	if commit := tx.catalogCommit; commit != nil {
		tx.catalogCommit = nil
		commit.access.Release()
		for _, held := range commit.appendDDL {
			held.ddlMu.Unlock()
		}
	}
}

// Called after all native participant preparations have reached a stable epoch,
// while the same schema publication capability orders ordinary DDL and commits.
func (tx *TxContext) prepareCatalogDecision() error {
	if !tx.hasCatalogIntents() {
		return nil
	}
	commit := tx.catalogCommit
	if commit == nil {
		return fmt.Errorf("missing native catalog publication capability")
	}
	if commit.prepared {
		return nil
	}
	db := commit.database
	if GetDatabase(db.Name) != db || db.incarnationID() != tx.catalogIdentity.Load().Incarnation {
		return fmt.Errorf("transaction catalog database was replaced")
	}
	// Make the complete predecessor checkpoint durable before allocating the next
	// version; later DROP/CREATE generations cannot be resurrected by replay.
	if err := persistenceCall(func() { db.save() }); err != nil {
		return err
	}
	identity := tx.catalogIdentity.Load()
	tables := make([]*table, 0)
	db.schemalock.Lock()
	for _, original := range tx.catalogIntentTables() {
		current := db.tables.Get(original.Name)
		if current == nil || current.RuntimeIdentity != original.RuntimeIdentity || current.CreatorTxID != tx.durableID() {
			db.schemalock.Unlock()
			return fmt.Errorf("unpublished catalog generation was replaced")
		}
		if current.CatalogDecisionID != identity.Decision {
			db.schemalock.Unlock()
			return fmt.Errorf("catalog declaration belongs to another decision")
		}
		current.CatalogVersion.Store(db.catalogVersion + 1)
		tables = append(tables, current)
	}
	if err := tx.validatePublicDeclarations(db); err != nil {
		db.schemalock.Unlock()
		return err
	}
	record := &catalogDecisionRecord{Format: 1, Kind: "prepare", Database: db.Name, Incarnation: identity.Incarnation, Transaction: tx.durableID(), Decision: identity.Decision,
		Installation: identity.Installation, Boot: walBootID, Namespace: identity.Namespace, Version: db.catalogVersion + 1, Distributed: true, Tables: tables}
	record.NativeDeclarations = tx.publicNativeDeclarations()
	record.Sources = tx.publicSourceDeclarations()
	record.Checks = tx.publicCheckDeclarations()
	record.CheckDrops = tx.publicCheckDrops()
	record.TableDrops = tx.publicTableDrops()
	record.StatementTriggers = tx.publicStatementTriggerDeclarations()
	record.ColumnDefaults = tx.publicColumnDefaultDeclarations()
	record.ColumnDeclarations = tx.publicColumnDeclarations()
	record.ColumnPropertyRemovals = tx.publicColumnPropertyRemovals()
	record.ColumnAppends = tx.publicColumnAppends()
	if len(record.Sources) != 0 {
		record.Format = 3
	}
	if len(record.Checks) != 0 {
		record.Format = 4
	}
	if len(record.StatementTriggers) != 0 {
		record.Format = 5
	}
	if len(record.ColumnDefaults) != 0 {
		record.Format = 6
	}
	for _, trigger := range record.StatementTriggers {
		if trigger.Header != nil && trigger.Header.Timing == ReplaceStatement {
			record.Format = 7
			break
		}
	}
	if len(record.CheckDrops) != 0 {
		record.Format = 8
	}
	if len(record.TableDrops) != 0 {
		record.Format = 9
	}
	participants, participantError := tx.catalogRowParticipantInventory()
	if participantError != nil {
		db.schemalock.Unlock()
		return participantError
	}
	if len(participants) != 0 {
		record.Format = 10
		if record.hasEnforcementDeltas() {
			record.Format = 11
		}
		record.Participants = participants
		record.AuthorityStorage = db.persistence.StorageIdentity()
	}
	if len(record.ColumnDeclarations) > 0 {
		record.Format = 14
	}
	if len(record.NativeDeclarations) > 0 {
		record.Format = 15
	}
	if len(record.ColumnAppends) > 0 {
		record.Format = 16
	}
	if len(record.ColumnPropertyRemovals) > 0 {
		record.Format = 17
	}
	if _, untrusted := record.foreignValidationFacts(); untrusted {
		record.Format = 18
	}
	for _, d := range record.ColumnAppends {
		if d.Initialization != nil {
			record.Format = 19
			break
		}
	}
	if record.hasColumnPropertyClaimActions() {
		record.Format = 20
	}
	if len(record.ColumnAppends) != 0 || len(record.NativeDeclarations) != 0 || len(record.ColumnDeclarations) != 0 || len(record.ColumnPropertyRemovals) != 0 || len(record.Checks) != 0 || len(record.ColumnDefaults) != 0 || len(record.CheckDrops) != 0 {
		commit.publicPublication = db.preparePublicConstraintPublication(record.Checks, record.ColumnDefaults, record.ColumnDeclarations, record.ColumnAppends, record.ColumnPropertyRemovals, record.CheckDrops, record.NativeDeclarations, record.Version, tx.publicConstraintPrograms(), tx.publicCatalogChanges())
	}

	encoded := encodeCatalogDecision(record)
	db.schemalock.Unlock()
	frozen, err := decodeCatalogDecision(encoded)
	if err != nil {
		return err
	}
	if err := persistenceCall(func() {
		db.initializeTransactionLog()
		db.transactionMu.Lock()
		defer db.transactionMu.Unlock()
		log := db.transactionLog
		if sharedClusterDatabase(db.Name) {
			_, entries, handle := db.persistence.ReplayLog(transactionLogName, false)
			for range entries {
			}
			log = handle
			defer handle.Close()
		}
		log.Write(LogEntryCommit{txID: encoded})
		log.Flush(true)
		if db.catalogDecisions == nil {
			db.catalogDecisions = make(map[string]*catalogDecisionRecord)
		}
		frozen.retainSourceFrameMemory(db)
		db.catalogDecisions[identity.Decision] = frozen
	}); err != nil {
		return err
	}
	if err := tx.enrollCatalogParticipants(frozen); err != nil {
		return err
	}
	commit.record, commit.prepared = frozen, true
	return nil
}

// The durable native decision already authorizes every prepared row generation.
// Publish all names in one immutable view, then checkpoint under the same lease.
func (tx *TxContext) publishCatalogDecision() error {
	commit := tx.catalogCommit
	if commit == nil || !commit.prepared {
		return nil
	}
	db := commit.database
	if GetDatabase(db.Name) != db || db.incarnationID() != tx.catalogIdentity.Load().Incarnation {
		return fmt.Errorf("transaction catalog database was replaced")
	}
	db.schemalock.Lock()
	for _, original := range tx.catalogIntentTables() {
		current := db.tables.Get(original.Name)
		if current == nil || current.RuntimeIdentity != original.RuntimeIdentity {
			db.schemalock.Unlock()
			return fmt.Errorf("catalog decision lost a reserved identity")
		}
		current.CatalogVersion.Store(commit.record.Version)
	}
	if db.catalogVersion < commit.record.Version {
		if commit.publicPublication != nil {
			commit.publicPublication.apply()
		}
		if len(commit.record.Sources) != 0 {
			tx.publicCatalogChanges().publishSourcesLocked(commit.record.Version)
		}
		if len(commit.record.StatementTriggers) != 0 {
			tx.publicCatalogChanges().publishStatementTriggersLocked(commit.record.Version)
		}
	}
	tx.publishTableDropCutsLocked(commit.record.Version)
	db.catalogVersion = commit.record.Version
	db.publishTableNamesLocked()
	tx.detachTableDropsLocked(commit.record.Version)
	if len(commit.record.Sources) != 0 {
		tx.publicCatalogChanges().finishSourcePublicationLocked()
	}
	if len(commit.record.StatementTriggers) != 0 {
		tx.publicCatalogChanges().finishStatementTriggerPublicationLocked()
	}
	if commit.publicPublication != nil {
		commit.publicPublication.retirePrevious()
	}
	for _, original := range tx.catalogIntentTables() {
		original.creator.Store(nil)
		if current := db.tables.Get(original.Name); current != nil {
			current.creator.Store(nil)
		}
	}
	db.schemalock.Unlock()
	db.transactionMu.Lock()
	db.committedTx[tx.durableID()] = db.transactionGen
	db.transactionMu.Unlock()
	err := persistenceCall(func() { db.save() })
	for _, original := range tx.catalogIntentTables() {
		if current := db.tables.Get(original.Name); current != nil && current.RuntimeIdentity == original.RuntimeIdentity {
			registerCreatedTable(current)
		}
	}
	// Even a failed checkpoint cannot undo the already durable decision. Replay
	// contains both the complete declarations and their native commit proof.
	if err != nil && len(commit.record.TableDrops) != 0 {
		// Pending accepted DROP retains its original publication/resources and
		// lifecycle capability until retry durably excludes the payload.
		return err
	}
	tx.releaseCatalogPublication()
	if err == nil {
		tx.finishTableDrops()
	}
	return err
}

func (tx *TxContext) catalogDecisionBackend() PersistenceEngine {
	if identity := tx.catalogIdentity.Load(); identity != nil {
		if db := GetDatabase(identity.Database); db != nil && db.incarnationID() == identity.Incarnation {
			return db.persistence
		}
		return nil
	}
	return sharedCommitBackend()
}

// Native mutation admission runs once at the ordinary shard write boundary.
// Public row effects retain their admitted default/source profile; newly
// declared private tables retain their own catalog and row decision.
func (tx *TxContext) admitCatalogMutation(table *table) {
	tx.noteStatementMutation(table)
	if tx == nil {
		return
	}
	tx.requireMutationBinding()
	// Append has no private physical slot. This veto precedes every native
	// writer grant, including a checked KEY/FK input on the same endpoint.
	if len(tx.pendingNullColumns(table)) != 0 {
		panic("pending nullable append does not admit endpoint row mutation")
	}
	if tx.decisionStarted {
		panic("transaction commit outcome is unknown; retry COMMIT")
	}
	if identity := tx.catalogIdentity.Load(); identity != nil {
		// Scope-owned RAM relations retain their existing native transaction and
		// lifecycle semantics; they are never a durable public mutation target.
		if table.Ephemeral && table.PersistencyMode == Memory {
			return
		}
		scope := tx.resourceScope()
		scope.mu.Lock()
		busy := scope.constructing != 0 || scope.deciding
		if busy {
			scope.mu.Unlock()
			panic("catalog transaction is not open for native row mutation")
		}
		if identity.lateCreates {
			if !tx.lateCreateRowAllowed(table) {
				scope.mu.Unlock()
				panic("late private catalog changes require their original data database")
			}
			if tx.ownsPendingTable(table) {
				table.catalogDataEntered.Store(true)
			}
			tx.rowAccessed.Store(true)
			scope.publicRowsEntered = true
			scope.mu.Unlock()
			return
		}
		if tx.publicCatalogEntered.Load() {
			if !tx.publicColumnMutationAllowed(table) && tx.mutationInputFor(table) == nil && !tx.latePublicRowAllowed(table) {
				scope.mu.Unlock()
				panic("native metadata-only transaction does not admit row mutation")
			}
			// Declaration construction and this row-phase publication share
			// one admission boundary, including reuse of a default's lease.
			tx.rowAccessed.Store(true)
			scope.publicRowsEntered = true
			scope.mu.Unlock()
			return
		}
		if table.schema.Name != identity.Database || !tx.ownsPendingTable(table) {
			scope.mu.Unlock()
			panic("catalog transaction mutations require an owned unpublished table")
		}
		// Publish the row-phase boundary under the same admission as private
		// declaration construction, before defaults, triggers or ID allocation.
		tx.rowAccessed.Store(true)
		scope.mu.Unlock()
	}
}

func (tx *TxContext) discardCatalogIntents(keep int) {
	intents := tx.catalogIntents.Load()
	if intents == nil || len(*intents) <= keep {
		return
	}
	if tx.coordinator != nil {
		panic("private catalog retirement requires its native coordinator")
	}
	identity := tx.catalogIdentity.Load()
	db := GetDatabase(identity.Database)
	if db == nil || db != identity.database || db.incarnationID() != identity.Incarnation {
		return
	}
	access := db.GetWrite(nil)
	defer access.Release()
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.schemalock.Lock()
	var retired []replacedShardGeneration
	for _, original := range (*intents)[keep:] {
		table := db.tables.Get(original.Name)
		if table == nil || table.RuntimeIdentity != original.RuntimeIdentity || table.CreatorTxID != tx.durableID() {
			continue
		}
		db.tables.Remove(table.Name)
		topology := table.activeTopology()
		if topology != nil {
			drained := topology.retire()
			for _, shard := range table.ActiveShards() {
				if shard != nil {
					retired = append(retired, replacedShardGeneration{shard, drained, topology.retirementDependencies})
				}
			}
		}
	}
	remaining := append([]*table(nil), (*intents)[:keep]...)
	if len(remaining) == 0 {
		tx.catalogIntents.Store(nil)
	} else {
		tx.catalogIntents.Store(&remaining)
	}
	db.publishTableNamesLocked()
	db.schemalock.Unlock()
	if failures := cleanupUnpublishedCatalogGenerations(retired); len(failures) != 0 {
		panic(fmt.Errorf("native catalog rollback cleanup: %v", failures))
	}
}

// lockCatalogDecision does not hold schema publication while obtaining native
// prepare rights. Prepared admissions survive the first decision critical
// section, allowing schema M acquisition without blocking needed row handover.
func (tx *TxContext) lockCatalogDecision(ctx context.Context) (releaseDecision func(), decisionError error) {
	defer func() {
		if failure := recover(); failure != nil {
			// Native acquisition reports failures by panic. Return them through
			// the ordinary pre-decision cleanup path so prepared participants
			// and reservations are not abandoned by a removed database/peer.
			decisionError = fmt.Errorf("native catalog preparation: %v", failure)
		}
	}()
	if identity := tx.catalogIdentity.Load(); identity != nil && (identity.latePublic || identity.lateCreates) && !tx.decisionStarted {
		tx.mu.Lock()
		var admissionError error
		if tx.prepared == nil && tx.catalogCommit == nil {
			if identity.lateCreates {
				admissionError = tx.validateLateCreateOwnerLocked(identity.database, true)
			} else {
				admissionError = tx.validateLatePublicOwnerLocked(identity.database, true)
			}
		}
		tx.mu.Unlock()
		if admissionError != nil {
			return nil, admissionError
		}
	}
	if tx.decisionStarted && tx.catalogCommit != nil {
		tx.mu.Lock()
		directory := tx.shardTransactions()
		tx.mu.Unlock()
		shards := make([]*storageShard, 0, len(directory))
		for shard := range directory {
			shards = append(shards, shard)
		}
		release, ready := tx.claimDecisionRights(shards, false)
		if !ready {
			return nil, distributedOutcomeUnknown(fmt.Errorf("prepared native rights are not ready"))
		}
		if !tx.decisionMu.TryLock() {
			release()
			return nil, distributedOutcomeUnknown(fmt.Errorf("native decision is busy"))
		}
		if tx.migrations != 0 {
			tx.decisionMu.Unlock()
			release()
			return nil, distributedOutcomeUnknown(fmt.Errorf("native decision migration is pending"))
		}
		return func() { tx.decisionMu.Unlock(); release() }, nil
	}
	for {
		release, err := tx.lockDecision(ctx, true, nil)
		if err != nil {
			return nil, err
		}
		var epoch uint64
		var generations map[*storageShard]*storageShardTransaction
		preparationErr := func() (err error) {
			defer release()
			defer func() {
				if failure := recover(); failure != nil {
					tx.releasePreparation()
					panic(failure)
				}
			}()
			for {
				owners, version := tx.remoteParticipantSnapshot()
				if err = tx.prepareACID(ctx); err != nil {
					return err
				}
				keys := make([]string, 0, len(owners))
				for key := range owners {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					participant := owners[key]
					_, err = participant.executeClusterRequest(ctx, scm.ClusterRequest{Method: "prepare", Arguments: []scm.Scmer{scm.NewString(participant.decisionToken)}})
					if err != nil {
						return err
					}
				}
				_, latest := tx.remoteParticipantSnapshot()
				if version == latest {
					epoch = latest
					break
				}
			}
			tx.mu.Lock()
			generations = tx.shardTransactions()
			tx.mu.Unlock()
			return nil
		}()
		if preparationErr != nil {
			return nil, preparationErr
		}
		// GetWrite must run outside decisionMu: its ordinary MOSI acquisition may
		// install live metadata and move transaction endpoints.
		tx.acquireCatalogPublication()
		shards := make([]*storageShard, 0, len(generations))
		for shard := range generations {
			shards = append(shards, shard)
		}
		releaseRights, complete := tx.claimDecisionRights(shards, false)
		if !complete {
			tx.releaseCatalogPublication()
			continue
		}
		if !tx.decisionMu.TryLock() {
			releaseRights()
			tx.releaseCatalogPublication()
			continue
		}
		release = func() { tx.decisionMu.Unlock(); releaseRights() }

		tx.mu.Lock()
		stable := tx.migrations == 0 && epoch == tx.participantEpoch
		current := tx.shardTransactions()
		stable = stable && len(current) == len(generations)
		for shard, overlay := range current {
			stable = stable && generations[shard] == overlay
		}
		tx.mu.Unlock()
		if stable {
			tx.catalogCommit.participantsPrepared = true
			return release, nil
		}
		release()
		tx.releaseCatalogPublication()
	}
}

// This group owns its exact private generations until all native readers and
// transaction pins drain. It never uses the published-generation remover.
func cleanupUnpublishedCatalogGenerations(retired []replacedShardGeneration) []string {
	cleanup := func() []string {
		var failures []string
		for _, generation := range retired {
			generation.wait()
			if err := persistenceCall(func() {
				generation.shard.closeRetiredLog()
				discardUnpublishedShard(generation.shard)
			}); err != nil {
				failures = append(failures, err.Error())
			}
		}
		return failures
	}
	for _, generation := range retired {
		if !generation.ready() {
			go func() {
				for _, failure := range cleanup() {
					fmt.Println("native catalog private generation cleanup:", failure)
				}
			}()
			return nil
		}
	}
	return cleanup()
}

type catalogConstructionProfile uint8

const (
	catalogDeclarationConstruction catalogConstructionProfile = iota
	catalogEnforcementConstruction
	catalogValidationConstruction
	catalogStatementTriggerConstruction
)

// Catalog construction shares the native resource/decision admission. A native
// COMMIT cannot overtake a builder before its first intent is published.
// After public row mutation, enforcement may extend only endpoints already
// blocked by this owner's earlier enforcement declarations. Default-only leases
// grant row rights and cannot establish that protection. Read-only validation
// is not itself a row phase. Statement-program amendments instead retain
// their actual endpoint/input lease; their later producers capture the new
// definition without re-executing an earlier row phase.
func (tx *TxContext) beginCatalogConstruction(profile catalogConstructionProfile, endpoints ...*table) func() {
	switch profile {
	case catalogDeclarationConstruction, catalogEnforcementConstruction, catalogStatementTriggerConstruction:
		tx.noteStatementMutation(nil)
	case catalogValidationConstruction:
		// Admission itself changes no declaration. Validation retains the same
		// decision exclusion; only its actual private publication marks an effect.
		if tx.failurePolicyExecuting.Load() {
			panic(newNativeMutationBusyError(tx.transactionState, "failure policy mutation"))
		}
		if err := tx.requireCommittable(); err != nil {
			panic(err)
		}
	default:
		panic("invalid catalog construction profile")
	}
	scope := tx.resourceScope()
	scope.mu.Lock()
	if scope.closed || scope.deciding {
		scope.mu.Unlock()
		panic("native transaction is finishing")
	}
	for _, endpoint := range endpoints {
		if endpoint != nil {
			owner := &endpoint.writeAdmission
			owner.mu.Lock()
			busy := owner.statement != nil
			owner.mu.Unlock()
			if busy {
				scope.mu.Unlock()
				panic("native declaration input is in use by an admitted statement")
			}
		}
	}
	if profile == catalogEnforcementConstruction && scope.publicRowsEntered {
		protected := false
		if changes := tx.publicCatalogChanges(); changes != nil && len(endpoints) != 0 {
			if views := changes.views.Load(); views != nil {
				protected = true
				for _, t := range endpoints {
					_, blocked := views.blockedWriters[t]
					_, dropped := views.dropped[t]
					if t == nil || !blocked || dropped {
						protected = false
						break
					}
				}
				protected = protected && changes.owner == tx.transactionState
			}
		}
		if !protected {
			scope.mu.Unlock()
			panic("enforcement declarations must precede native row mutation")
		}
		// The immutable blocked directory is published only after retaining the
		// original endpoint leases. Under this same scope admission, the count
		// below excludes row writers, rewind and retirement until the ordinary
		// stage revalidates the actual generation and declaration baseline.
	}
	scope.constructing++
	scope.mu.Unlock()
	return func() { scope.mu.Lock(); scope.constructing--; scope.mu.Unlock() }
}
func (tx *TxContext) forwardCatalogCreate(arguments []scm.Scmer) scm.Scmer {
	args := append([]scm.Scmer(nil), arguments[:5]...)
	value, err := tx.coordinator.Call(scanQueryContext(tx, context.Background()), scm.ClusterRequest{Method: "catalog_create", Arguments: args})
	if err != nil {
		panic(err)
	}
	result := value.Slice()
	if len(result) != 2 {
		panic("invalid native catalog creation reply")
	}
	handle := *tx.coordinator
	handle.View = result[1]
	importTransactionState(handle)
	return result[0]
}

func (tx *TxContext) noteNativeRowAccess(table *table) {
	if table != nil {
		table.enterPendingCatalogData(tx)
	}
	if tx != nil && (table == nil || !table.Ephemeral || table.PersistencyMode != Memory) {
		tx.rowAccessed.Store(true)
	}
}

// Before acceptance, cancellation may abort preparation. An accepted native
// decision instead belongs to its transaction and must finish independently of
// the disconnected request that originally asked for COMMIT.
func (tx *TxContext) catalogDecisionContext(fallback context.Context) context.Context {
	if tx.decisionStarted && tx.catalogIdentity.Load() != nil {
		return fallback
	}
	return scanQueryContext(tx, fallback)
}
