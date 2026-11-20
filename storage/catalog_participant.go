/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "io"
import "sort"
import "unsafe"
import "strings"
import "strconv"
import "encoding/json"
import "encoding/base64"
import "github.com/google/uuid"

const catalogParticipantPrefix = "catalog-participant-v10/"

func validLocalCatalogTransaction(transaction, boot string) bool {
	parts := strings.Split(transaction, "/")
	if len(parts) != 2 || parts[0] != boot {
		return false
	}
	n, err := strconv.ParseUint(parts[1], 10, 64)
	return err == nil && n != 0
}

// The logical database incarnation survives a legitimate storage move. The
// authority's physical address is additionally fenced until local settlement.
type catalogParticipantIdentity struct {
	Database    string `json:"database"`
	Incarnation string `json:"incarnation"`
}

type catalogParticipantRecord struct {
	Version      int                        `json:"version"`
	State        string                     `json:"state"`
	Transaction  string                     `json:"transaction"`
	Authority    catalogParticipantIdentity `json:"authority"`
	Storage      string                     `json:"storage"`
	Decision     string                     `json:"decision"`
	Installation string                     `json:"installation"`
	Boot         string                     `json:"boot"`
	Participant  catalogParticipantIdentity `json:"participant"`
}

func (record *catalogParticipantRecord) authorityID() string {
	return (&catalogDecisionRecord{Incarnation: record.Authority.Incarnation, Installation: record.Installation, Decision: record.Decision}).authorityID()
}

func (record *catalogParticipantRecord) encoded() string {
	data, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	return catalogParticipantPrefix + base64.RawURLEncoding.EncodeToString(data)
}

func validCatalogParticipant(identity catalogParticipantIdentity) bool {
	id, err := uuid.Parse(identity.Incarnation)
	return validDatabaseName(identity.Database) && err == nil && id != uuid.Nil
}

func decodeCatalogParticipant(value string) (*catalogParticipantRecord, error) {
	if !strings.HasPrefix(value, catalogParticipantPrefix) {
		return nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, catalogParticipantPrefix))
	if err != nil {
		return nil, fmt.Errorf("invalid native participant frame: %w", err)
	}
	var record catalogParticipantRecord
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&record); err != nil {
		return nil, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("invalid trailing native participant payload")
	}
	if record.Version != 10 || record.Storage == "" || !validCatalogParticipant(record.Authority) || !validCatalogParticipant(record.Participant) ||
		(record.State != "prepare" && record.State != "commit" && record.State != "abort") {
		return nil, fmt.Errorf("invalid native participant identity or state")
	}
	for _, value := range []string{record.Decision, record.Installation, record.Boot} {
		if id, err := uuid.Parse(value); err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("invalid native participant authority")
		}
	}
	if !validLocalCatalogTransaction(record.Transaction, record.Boot) {
		return nil, fmt.Errorf("invalid native participant transaction")
	}
	return &record, nil
}

// All callers own transactionMu, or the unpublished transactionOnce producer.
func (db *database) acceptCatalogParticipant(record *catalogParticipantRecord, committed map[string]uint64, generation uint64) error {
	if record.Participant.Database != db.Name {
		return fmt.Errorf("native participant belongs to another database")
	}
	if db.catalogParticipants == nil {
		db.catalogParticipants = make(map[string]*catalogParticipantRecord)
	}
	if previous := db.catalogParticipants[record.Transaction]; previous != nil {
		before, after := *previous, *record
		before.State, after.State = "", ""
		if before != after || previous.State != "prepare" && previous.State != record.State {
			return fmt.Errorf("conflicting native participant decision")
		}
	} else {
		bytes := uint(unsafe.Sizeof(*record)) + uint(len(record.State)+len(record.Transaction)+len(record.Authority.Database)+len(record.Authority.Incarnation)+
			len(record.Storage)+len(record.Decision)+len(record.Installation)+len(record.Boot)+len(record.Participant.Database)+len(record.Participant.Incarnation)) + 64
		db.adjustMetadataMemory(0, bytes)
	}
	db.catalogParticipants[record.Transaction] = record
	return nil
}

func (record *catalogDecisionRecord) participantRecord(identity catalogParticipantIdentity) *catalogParticipantRecord {
	return &catalogParticipantRecord{Version: 10, State: "prepare", Transaction: record.Transaction,
		Authority: catalogParticipantIdentity{Database: record.Database, Incarnation: record.Incarnation}, Storage: record.AuthorityStorage,
		Decision: record.Decision, Installation: record.Installation, Boot: record.Boot, Participant: identity}
}

// Inspect only the durable identity of a registered backend. In particular this
// does NOT invoke ensureLoaded on another database while recovering this one.
func (db *database) durableIncarnation() (string, error) {
	var header struct {
		Incarnation string `json:"incarnation"`
	}
	if err := json.Unmarshal(db.persistence.ReadSchema(), &header); err != nil {
		return "", err
	}
	return header.Incarnation, nil
}

func (db *database) writeCatalogParticipant(record *catalogParticipantRecord) error {
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if previous := db.catalogParticipants[record.Transaction]; previous != nil {
		before, after := *previous, *record
		before.State, after.State = "", ""
		if before != after {
			return fmt.Errorf("native participant authority changed")
		}
		if previous.State == record.State {
			return nil
		}
		if previous.State != "prepare" {
			return fmt.Errorf("native participant already settled differently")
		}
	}
	return persistenceCall(func() {
		db.transactionLog.Write(LogEntryCommit{txID: record.encoded()})
		db.transactionLog.Flush(true)
		if err := db.acceptCatalogParticipant(record, db.committedTx, db.transactionGen); err != nil {
			panic(err)
		}
	})
}

// First make the authority's complete inventory durable, then every child
// enrollment. Only the caller's subsequent single D can commit this program.
func (tx *TxContext) enrollCatalogParticipants(record *catalogDecisionRecord) error {
	for _, identity := range record.Participants {
		db := GetDatabase(identity.Database)
		if db == nil || db.incarnationID() != identity.Incarnation {
			return fmt.Errorf("native row participant was replaced")
		}
		if err := db.writeCatalogParticipant(record.participantRecord(identity)); err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxContext) settleCatalogParticipants() error {
	if tx.catalogCommit == nil || tx.catalogCommit.record == nil || !tx.catalogCommit.record.hasRowParticipants() {
		return nil
	}
	return tx.catalogCommit.database.settleCatalogParticipantInventory(tx.catalogCommit.record, true, tx.prepared.catalogDatabases)
}

// Existing lifecycle exclusion belongs to the authority caller. Each target is
// pinned without blocking or materializing its schema/shards. A terminal child
// frame is sufficient on its own after the authority is dropped or moved.
func (db *database) settleCatalogParticipantInventory(record *catalogDecisionRecord, committed bool, pinned map[*database]bool) error {
	for _, identity := range record.Participants {
		target := GetDatabase(identity.Database)
		if target == nil {
			continue
		} // a removed logical participant has no surviving rows
		settle := func() error {
			if target != db && !pinned[target] {
				if !target.persistenceLifecycle.TryRLock() {
					return fmt.Errorf("native participant storage is under maintenance; retry")
				}
				defer target.persistenceLifecycle.RUnlock()
			}
			incarnation, err := target.durableIncarnation()
			if err != nil {
				return err
			}
			if GetDatabase(identity.Database) != target || incarnation != identity.Incarnation {
				return nil
			}
			enrollment := record.participantRecord(identity)
			if committed {
				enrollment.State = "commit"
			} else {
				enrollment.State = "abort"
			}
			return target.writeCatalogParticipant(enrollment)
		}
		if err := settle(); err != nil {
			return err
		}
	}
	return nil
}

// Old boots cannot accept another decision. During live maintenance the caller
// must additionally own the authority's publication/maintenance exclusion.
func (db *database) settleCatalogParticipantDecisions(maintenance bool) error {
	db.initializeTransactionLog()
	db.transactionMu.RLock()
	var records []*catalogDecisionRecord
	for _, record := range db.catalogDecisions {
		if record.hasRowParticipants() && record.Incarnation == db.incarnationID() {
			records = append(records, record)
		}
	}
	db.transactionMu.RUnlock()
	for _, record := range records {
		// A legitimate authority move first settles its OWN inventory entry.
		// That exact terminal proof survives move/compaction without depending
		// on the former physical decision-log namespace.
		db.transactionMu.RLock()
		own := db.catalogParticipants[record.Transaction]
		db.transactionMu.RUnlock()
		committed, terminal := false, false
		if own != nil {
			expected, actual := *record.participantRecord(catalogParticipantIdentity{Database: db.Name, Incarnation: record.Incarnation}), *own
			expected.State, actual.State = "", ""
			if expected != actual {
				return fmt.Errorf("catalog authority has conflicting local enrollment")
			}
			terminal, committed = own.State != "prepare", own.State == "commit"
		}
		if !terminal {
			if record.AuthorityStorage != db.persistence.StorageIdentity() {
				return fmt.Errorf("unsettled catalog authority storage changed")
			}
			committed = readDistributedDecision(db.persistence, record.authorityID())
		}
		if !terminal && !committed && record.Boot == walBootID && !maintenance {
			continue
		}
		if err := db.settleCatalogParticipantInventory(record, committed, nil); err != nil {
			return err
		}
	}
	return nil
}

// D alone deliberately omits the original row T. The authority's durable
// prepare must therefore also attest this exact enrollment and its membership.
// ReplayLog is streaming and read-only; no other database is materialized.
func (db *database) verifyCatalogEnrollment(enrollment *catalogParticipantRecord) error {
	_, frames, _ := db.persistence.ReplayLog(transactionLogName, true)
	var matched bool
	var failure error
	for frame := range frames {
		commit, ok := frame.(LogEntryCommit)
		if !ok || (!strings.HasPrefix(commit.txID, catalogParticipantJournalPrefix) && !strings.HasPrefix(commit.txID, catalogEnforcementParticipantJournalPrefix) && !strings.HasPrefix(commit.txID, catalogForeignDropJournalPrefix) && !strings.HasPrefix(commit.txID, catalogForeignHistoryJournalPrefix) && !strings.HasPrefix(commit.txID, catalogColumnDeclarationJournalPrefix) && !strings.HasPrefix(commit.txID, catalogNativeDeclarationJournalPrefix) && !strings.HasPrefix(commit.txID, catalogColumnPropertyJournalPrefix) && !strings.HasPrefix(commit.txID, catalogColumnAppendJournalPrefix) && !strings.HasPrefix(commit.txID, catalogForeignValidationJournalPrefix)) {
			continue
		}
		record, err := decodeCatalogDecision(commit.txID)
		if err != nil {
			failure = err
			continue
		}
		if record.Decision != enrollment.Decision {
			continue
		}
		if record.Database != enrollment.Authority.Database || record.Incarnation != enrollment.Authority.Incarnation ||
			record.Transaction != enrollment.Transaction || record.Installation != enrollment.Installation ||
			record.Boot != enrollment.Boot || record.AuthorityStorage != enrollment.Storage {
			failure = fmt.Errorf("native participant does not match its authority preparation")
			continue
		}
		for _, participant := range record.Participants {
			matched = matched || participant == enrollment.Participant
		}
	}
	if failure != nil {
		return failure
	}
	if !matched {
		return fmt.Errorf("native participant has no authoritative enrollment")
	}
	return nil
}

func (db *database) catalogParticipantCommitted(record *catalogParticipantRecord) bool {
	if record.Participant.Incarnation != db.incarnationID() {
		return false
	}
	if record.State != "prepare" {
		return record.State == "commit"
	}
	authority := GetDatabase(record.Authority.Database)
	if authority == nil {
		panic("native participant lost its unsettled authority")
	}
	resolve := func() bool {
		if authority != db {
			if !authority.persistenceLifecycle.TryRLock() {
				panic("native participant authority is under maintenance; retry")
			}
			defer authority.persistenceLifecycle.RUnlock()
		}
		incarnation, err := authority.durableIncarnation()
		if err != nil {
			panic(err)
		}
		if GetDatabase(record.Authority.Database) != authority || incarnation != record.Authority.Incarnation || authority.persistence.StorageIdentity() != record.Storage {
			panic("native participant authority identity changed before settlement")
		}
		if err := authority.verifyCatalogEnrollment(record); err != nil {
			panic(err)
		}
		committed := readDistributedDecision(authority.persistence, record.authorityID())
		// Absence is not abort authority for an active preparation.
		if committed || record.Boot != walBootID {
			settled := *record
			if committed {
				settled.State = "commit"
			} else {
				settled.State = "abort"
			}
			if err := db.writeCatalogParticipant(&settled); err != nil {
				panic(err)
			}
		}
		return committed
	}
	return resolve()
}

func (db *database) catalogParticipantFrames() []interface{} {
	keys := make([]string, 0, len(db.catalogParticipants))
	for key := range db.catalogParticipants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	frames := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		frames = append(frames, LogEntryCommit{txID: db.catalogParticipants[key].encoded()})
	}
	return frames
}

// Reuse existing mode and persistence-lifecycle owners. Pins are per database
// and preparation, never per row. They survive an ambiguous accepted decision.
func (tx *TxContext) pinCatalogParticipantDatabases(preparation *acidPreparation, directory map[*storageShard]*storageShardTransaction) error {
	identity := tx.catalogIdentity.Load()
	changes := tx.publicCatalogChanges()
	if identity == nil || identity.lateCreates || changes == nil {
		return nil
	}
	views := changes.views.Load()
	if views == nil {
		return nil
	}
	if !localCatalogRowDatabase(identity.database) {
		return fmt.Errorf("catalog row participants require a local authority")
	}
	dbs := map[*database]bool{identity.database: true}
	for shard := range directory {
		if shard.t.PersistencyMode == Safe || shard.t.PersistencyMode == Logged {
			dbs[shard.t.schema] = true
		}
	}
	tx.recoveryDecisionDatabases.Range(func(key, _ any) bool { dbs[key.(*database)] = true; return true })
	if len(dbs) < 2 {
		return nil
	} // historical single-database profiles retain their exact owners
	for db := range dbs {
		if preparation.catalogDatabases[db] {
			continue
		}
		if !localCatalogRowDatabase(db) {
			return fmt.Errorf("catalog row participant is not local")
		}
		// A table DROP already owns this exact lifecycle through the existing
		// transaction resource. Borrow it instead of recursively reading the
		// writer-preferred RWMutex while CLEAN may be queued.
		scope := tx.resourceScope()
		scope.mu.Lock()
		entry := scope.entries[tableDropLifecycleResourceKey]
		borrowed := false
		if entry != nil {
			select {
			case <-entry.done:
				owner, ok := entry.value.Any().(*catalogTableDropLifecycle)
				borrowed = entry.failure == nil && ok && owner.db == db && !owner.transferred.Load()
			default:
			}
		}
		scope.mu.Unlock()
		if borrowed {
			if preparation.catalogDatabases == nil {
				preparation.catalogDatabases = make(map[*database]bool)
			}
			preparation.catalogDatabases[db] = true
			continue
		}
		if !db.persistenceLifecycle.TryRLock() {
			return fmt.Errorf("catalog row participant storage is busy; retry commit")
		}
		if err := func() error {
			accepted := false
			defer func() {
				if !accepted {
					db.persistenceLifecycle.RUnlock()
				}
			}()
			mode, err := db.tryAdmitCatalogParticipant()
			if err != nil {
				return err
			}
			if preparation.catalogDatabases == nil {
				preparation.catalogDatabases = make(map[*database]bool)
			}
			preparation.catalogDatabases[db] = true
			preparation.releases = append(preparation.releases, func() { _ = mode.Close(); db.persistenceLifecycle.RUnlock() })
			accepted = true
			return nil
		}(); err != nil {
			return err
		}
	}
	return nil
}

func (tx *TxContext) catalogRowParticipantInventory() ([]catalogParticipantIdentity, error) {
	identity := tx.catalogIdentity.Load()
	if tx.prepared == nil || len(tx.prepared.catalogDatabases) < 2 {
		return nil, nil
	}
	if identity == nil || identity.lateCreates || tx.coordinator != nil || len(tx.participants) != 0 || tx.clusterNumber.Load() != 0 {
		return nil, fmt.Errorf("cross-database catalog rows require an original local transaction")
	}
	participants := make([]catalogParticipantIdentity, 0, len(tx.prepared.catalogDatabases))
	for db := range tx.prepared.catalogDatabases {
		if !localCatalogRowDatabase(db) {
			return nil, fmt.Errorf("native catalog participant changed mode or identity")
		}
		participants = append(participants, catalogParticipantIdentity{Database: db.Name, Incarnation: db.incarnationID()})
	}
	sort.Slice(participants, func(i, j int) bool { return participants[i].Database < participants[j].Database })
	return participants, nil
}
