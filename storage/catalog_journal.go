/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
)

// Catalog envelopes use the existing checksummed native transaction frame.
// Legacy readers see an unrelated opaque identity, never the enclosed row ID.
// Only this database's validated native decoder may interpret its decision.
const catalogJournalPrefix = "catalog-v1/"
const catalogDeltaJournalPrefix = "catalog-v2/"
const catalogSourceJournalPrefix = "catalog-v3/"
const catalogCheckJournalPrefix = "catalog-v4/"
const catalogStatementTriggerJournalPrefix = "catalog-v5/"
const catalogColumnDefaultJournalPrefix = "catalog-v6/"
const catalogStatementReplacementJournalPrefix = "catalog-v7/"

type catalogDecisionRecord struct {
	encoded          string
	sourceFrameBytes uint
	sourceFrameOwner *database

	Format            int                                  `json:"format"`
	Kind              string                               `json:"kind"`
	Database          string                               `json:"database"`
	Incarnation       string                               `json:"incarnation"`
	Transaction       string                               `json:"transaction"`
	Decision          string                               `json:"decision"`
	Installation      string                               `json:"installation"`
	Boot              string                               `json:"boot"`
	Namespace         string                               `json:"namespace,omitempty"`
	Version           uint64                               `json:"version"`
	Distributed       bool                                 `json:"distributed,omitempty"`
	Tables            []*table                             `json:"tables"`
	ForeignKeys       []catalogForeignDeclaration          `json:"foreign_keys,omitempty"`
	Sources           []catalogSourceDeclaration           `json:"sources,omitempty"`
	Checks            []catalogCheckDeclaration            `json:"checks,omitempty"`
	StatementTriggers []catalogStatementTriggerDeclaration `json:"statement_triggers,omitempty"`
	ColumnDefaults    []catalogColumnDefaultDeclaration    `json:"column_defaults,omitempty"`
}

func (record *catalogDecisionRecord) authorityID() string {
	// Numeric cluster transaction IDs alone are not cross-installation proof.
	// This UUID-bound identity is the key in the existing decision stream.
	return "catalog-decision-v1/" + record.Namespace + "/" + record.Incarnation + "/" + record.Installation + "/" + record.Decision
}

func encodeCatalogDecision(record *catalogDecisionRecord) string {
	data, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	prefix := catalogJournalPrefix
	if record.Format == 7 {
		prefix = catalogStatementReplacementJournalPrefix
	} else if record.Format == 6 {
		prefix = catalogColumnDefaultJournalPrefix
	} else if record.Format == 5 {
		prefix = catalogStatementTriggerJournalPrefix
	} else if record.Format == 2 {
		prefix = catalogDeltaJournalPrefix
	} else if record.Format == 4 {
		prefix = catalogCheckJournalPrefix
	} else if record.Format == 3 {
		prefix = catalogSourceJournalPrefix
	}
	return prefix + base64.RawURLEncoding.EncodeToString(data)
}
func decodeCatalogDecision(value string) (*catalogDecisionRecord, error) {
	prefix := catalogJournalPrefix
	if strings.HasPrefix(value, catalogStatementReplacementJournalPrefix) {
		prefix = catalogStatementReplacementJournalPrefix
	} else if strings.HasPrefix(value, catalogColumnDefaultJournalPrefix) {
		prefix = catalogColumnDefaultJournalPrefix
	} else if strings.HasPrefix(value, catalogStatementTriggerJournalPrefix) {
		prefix = catalogStatementTriggerJournalPrefix
	} else if strings.HasPrefix(value, catalogCheckJournalPrefix) {
		prefix = catalogCheckJournalPrefix
	} else if strings.HasPrefix(value, catalogSourceJournalPrefix) {
		prefix = catalogSourceJournalPrefix
	} else if strings.HasPrefix(value, catalogDeltaJournalPrefix) {
		prefix = catalogDeltaJournalPrefix
	} else if !strings.HasPrefix(value, catalogJournalPrefix) {
		return nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return nil, fmt.Errorf("invalid catalog journal envelope: %w", err)
	}
	var record catalogDecisionRecord
	if err = json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	validShape := (record.Format == 1 && prefix == catalogJournalPrefix && len(record.Tables) > 0 && len(record.ForeignKeys) == 0 && len(record.Sources) == 0 && len(record.Checks) == 0) || (record.Format == 2 && prefix == catalogDeltaJournalPrefix && len(record.Tables) == 0 && len(record.ForeignKeys) > 0 && len(record.Sources) == 0 && len(record.Checks) == 0) || (record.Format == 3 && prefix == catalogSourceJournalPrefix && len(record.Tables) == 0 && len(record.Sources) > 0 && len(record.Checks) == 0) || (record.Format == 4 && prefix == catalogCheckJournalPrefix && len(record.Tables) == 0 && len(record.Checks) > 0)
	// Permanent v1-v4 readers keep their exact accepted domains. New trigger
	// deltas require their own version instead of silently widening an old one.
	validShape = validShape && len(record.StatementTriggers) == 0 || record.Format == 5 && prefix == catalogStatementTriggerJournalPrefix && len(record.Tables) == 0 && len(record.StatementTriggers) > 0
	// Defaults join the same decision only in v6. Historical readers' exact
	// accepted record domains remain unchanged, including mixed source deltas.
	validShape = validShape && len(record.ColumnDefaults) == 0 || record.Format == 6 && prefix == catalogColumnDefaultJournalPrefix && len(record.Tables) == 0 && len(record.ColumnDefaults) > 0
	// Replacing a native producer is new durable authority, not an additional
	// role admitted by historical AFTER-statement envelopes. Mixed declaration
	// deltas remain in the same decision, but require the replacement version.
	replacement := false
	for _, trigger := range record.StatementTriggers {
		replacement = replacement || trigger.Header != nil && trigger.Header.Timing == ReplaceStatement
	}
	validShape = validShape && !replacement || record.Format == 7 && prefix == catalogStatementReplacementJournalPrefix && len(record.Tables) == 0 && replacement
	if !validShape || record.Version == 0 || record.Kind != "prepare" || !record.Distributed || !validDatabaseName(record.Database) {
		return nil, fmt.Errorf("invalid catalog decision")
	}
	for _, identity := range []string{record.Decision, record.Installation, record.Boot, record.Incarnation} {
		if _, err := uuid.Parse(identity); err != nil {
			return nil, fmt.Errorf("invalid catalog authority identity")
		}
	}
	parts := strings.Split(record.Transaction, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid catalog transaction identity")
	}
	if parts[0] == "cluster" {
		if record.Namespace == "" {
			return nil, fmt.Errorf("cluster catalog decision lacks its namespace")
		}
	} else if parts[0] != record.Boot {
		return nil, fmt.Errorf("catalog transaction belongs to another incarnation")
	}
	if n, err := strconv.ParseUint(parts[1], 10, 64); err != nil || n == 0 {
		return nil, fmt.Errorf("invalid catalog transaction sequence")
	}
	if record.Namespace != "" {
		if _, err := uuid.Parse(record.Namespace); err != nil {
			return nil, fmt.Errorf("invalid catalog namespace")
		}
	}
	names := make(map[string]bool)
	for _, table := range record.Tables {
		if table == nil || len(table.Checks) != 0 || table.Name == "" || names[table.Name] || table.CreatorTxID != record.Transaction || table.CatalogDecisionID != record.Decision || table.CreatorIncarnation != record.Incarnation || table.CreatorNamespace != record.Namespace || table.CatalogVersion.Load() != record.Version {
			return nil, fmt.Errorf("invalid catalog declaration identity")
		}
		if _, err := uuid.Parse(table.RuntimeIdentity); err != nil {
			return nil, fmt.Errorf("invalid catalog table identity")
		}
		for _, trigger := range table.Triggers {
			if trigger.Timing == ReplaceStatement {
				return nil, fmt.Errorf("native statement replacement requires its declaration envelope")
			}
		}
		names[table.Name] = true
	}
	foreignNames := make(map[string]bool)
	for _, declaration := range record.ForeignKeys {
		fk := declaration.Foreign
		if fk.Id == "" || fk.Tbl1 == "" || fk.Tbl2 == "" || len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) || fk.Updatemode > SETNULL || fk.Deletemode > SETNULL || foreignNames[strings.ToLower(fk.Id)] {
			return nil, fmt.Errorf("invalid native foreign declaration delta")
		}
		// A self delta is one native endpoint with two RESTRICT roles. The
		// ordinary apply/replay owner still proves that current endpoint.
		if fk.Tbl1 == fk.Tbl2 && (fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT || declaration.ChildIdentity != declaration.ParentIdentity) {
			return nil, fmt.Errorf("invalid native self foreign declaration delta")
		}
		foreignNames[strings.ToLower(fk.Id)] = true
		for _, identity := range []string{declaration.ChildIdentity, declaration.ParentIdentity} {
			if _, err := uuid.Parse(identity); err != nil {
				return nil, fmt.Errorf("invalid native foreign endpoint identity")
			}
		}
	}

	for _, declaration := range record.Checks {
		if declaration.Table == "" || !utf8.ValidString(declaration.Table) || declaration.Check == nil {
			return nil, fmt.Errorf("invalid native check declaration delta")
		}
		if err := declaration.Check.validate(); err != nil {
			return nil, err
		}
		if _, err := uuid.Parse(declaration.Identity); err != nil {
			return nil, fmt.Errorf("invalid native check endpoint identity")
		}
		name := strings.ToLower(declaration.Check.Id)
		if foreignNames[name] {
			return nil, fmt.Errorf("duplicate native constraint declaration delta")
		}
		foreignNames[name] = true
	}

	sourceNames := make(map[string]bool)
	for _, source := range record.Sources {
		if source.Name == "" || !utf8.ValidString(source.Name) || sourceNames[source.Name] {
			return nil, fmt.Errorf("invalid native source declaration delta")
		}
		sourceNames[source.Name] = true
		if definition := source.Definition; definition != nil {
			if definition.Name != source.Name || definition.Language == "" || definition.Source == "" || !utf8.ValidString(definition.Language) || !utf8.ValidString(definition.Source) {
				return nil, fmt.Errorf("invalid native source definition")
			}
			if err := definition.initializeSourceMemory(); err != nil {
				return nil, err
			}
		}
	}
	triggerNames := make(map[string]bool)
	for _, trigger := range record.StatementTriggers {
		if trigger.Table == "" || trigger.Name == "" || !utf8.ValidString(trigger.Table) || !utf8.ValidString(trigger.Name) || triggerNames[trigger.Name] {
			return nil, fmt.Errorf("invalid native statement trigger delta")
		}
		if _, err := uuid.Parse(trigger.Identity); err != nil {
			return nil, fmt.Errorf("invalid native statement trigger table identity")
		}
		if trigger.Header != nil {
			if trigger.Header.Name != trigger.Name {
				return nil, fmt.Errorf("native statement trigger name disagrees with its header")
			}
			if err := validateStatementTrigger(*trigger.Header); err != nil {
				return nil, err
			}
		}
		triggerNames[trigger.Name] = true
	}
	defaultColumns := make(map[string]bool)
	for _, declaration := range record.ColumnDefaults {
		key := declaration.Table + "\x00" + declaration.Column
		if declaration.Table == "" || declaration.Column == "" || !utf8.ValidString(declaration.Table) || !utf8.ValidString(declaration.Column) || !utf8.ValidString(declaration.Expression) || !utf8.ValidString(declaration.Name) || defaultColumns[key] {
			return nil, fmt.Errorf("invalid native default declaration delta")
		}
		if _, err := uuid.Parse(declaration.Identity); err != nil {
			return nil, fmt.Errorf("invalid native default endpoint identity")
		}
		if declaration.Deferred && (!declaration.Value.IsNil() || declaration.Expression == "") {
			return nil, fmt.Errorf("invalid native deferred default declaration")
		}
		if declaration.Name != "" {
			name := strings.ToLower(declaration.Name)
			if foreignNames[name] {
				return nil, fmt.Errorf("duplicate native constraint declaration delta")
			}
			foreignNames[name] = true
		}
		defaultColumns[key] = true
	}
	record.encoded = value
	return &record, nil
}

// The same native global object's persisted Namespace scopes numeric cluster
// IDs. Read it during recovery without fabricating any MOSI permission.
func persistedCatalogNamespace(engine PersistenceEngine) (string, error) {
	if engine == nil {
		return "", fmt.Errorf("catalog namespace backend unavailable")
	}
	_, entries, _ := engine.ReplayLog(clusterGlobalLog, true)
	namespace := ""
	for entry := range entries {
		commit, ok := entry.(LogEntryCommit)
		if !ok {
			return "", fmt.Errorf("invalid native global namespace record")
		}
		value, err := decodeGlobal([]byte(commit.txID))
		if err != nil {
			return "", err
		}
		namespace = value.Namespace
	}
	if namespace == "" {
		return "", fmt.Errorf("catalog namespace has no durable identity")
	}
	return namespace, nil
}
func (db *database) validateCatalogAuthority(record *catalogDecisionRecord) error {
	if record.Database != db.Name {
		return fmt.Errorf("catalog decision belongs to another database")
	}
	if record.Namespace != "" {
		engine := sharedCommitBackend()
		if engine == nil {
			engine = db.persistence
		}
		namespace, err := persistedCatalogNamespace(engine)
		if err != nil {
			return err
		}
		if namespace != record.Namespace {
			return fmt.Errorf("catalog decision belongs to another native namespace")
		}
	}
	return nil
}

// Called under transactionMu, or while constructing the database privately.
// Raw legacy records retain their original interpretation. A prepare frame does
// not become a commit merely because a remote row participant already finished.
func (db *database) acceptNativeCommitFrame(value string, committed map[string]uint64, generation uint64) error {
	record, err := decodeCatalogDecision(value)
	if err != nil {
		return err
	}
	if record == nil {
		committed[value] = generation
		return nil
	}
	if err := db.validateCatalogAuthority(record); err != nil {
		return err
	}
	if db.catalogDecisions == nil {
		db.catalogDecisions = make(map[string]*catalogDecisionRecord)
	}
	previous := db.catalogDecisions[record.Decision]
	if previous != nil {
		if previous.encoded != value {
			return fmt.Errorf("conflicting native catalog decision")
		}
		return nil
	}
	record.retainSourceFrameMemory(db)
	db.catalogDecisions[record.Decision] = record
	return nil
}

// Replay only durable decisions above the schema checkpoint. The publication
// boundary orders versions, and preparing a decision first hardens its complete
// predecessor schema. A later DROP therefore cannot resurrect an older CREATE.
func (db *database) recoverCatalogDecisions() {
	records := make([]*catalogDecisionRecord, 0)
	db.transactionMu.Lock()
	for _, record := range db.catalogDecisions {
		if record.Incarnation != db.incarnationID() {
			continue
		}
		committed := readDistributedDecision(db.persistence, record.authorityID())
		if committed {
			db.committedTx[record.Transaction] = 0
			records = append(records, record)
		}
	}
	db.transactionMu.Unlock()
	sort.Slice(records, func(i, j int) bool { return records[i].Version < records[j].Version })
	for _, record := range records {
		if record.Version <= db.catalogVersion {
			continue
		}
		if record.Version != db.catalogVersion+1 {
			panic("native catalog decision has no durable predecessor")
		}
		declarations, err := decodeCatalogDecision(record.encoded)
		if err != nil {
			panic(err)
		}
		for _, table := range declarations.Tables {
			if previous := db.tables.Get(table.Name); previous != nil && previous.RuntimeIdentity != table.RuntimeIdentity {
				panic("native catalog decision conflicts with its predecessor schema")
			}
			db.tables.Set(table.Name, table)
		}
		db.recoverPublicForeignDeclarations(declarations.ForeignKeys)
		db.recoverPublicCheckDeclarations(declarations.Checks)
		db.recoverSourceDeclarations(declarations.Sources)
		db.recoverStatementTriggerDeclarations(declarations.StatementTriggers)
		db.recoverColumnDefaultDeclarations(declarations.ColumnDefaults)
		db.catalogVersion = record.Version
	}
	db.catalogProofVersion.Store(db.catalogVersion)
}

// The existing WAL recovery map retains the immutable envelope and its decoded
// source payload until database retirement. Row-log compaction deliberately keeps
// these catalog records. Count that actual extra allocation, not only live names.
// Caller owns transactionMu or is constructing this database privately.
func (record *catalogDecisionRecord) retainSourceFrameMemory(db *database) {
	if len(record.Sources) == 0 && len(record.Checks) == 0 && len(record.StatementTriggers) == 0 && len(record.ColumnDefaults) == 0 || record.sourceFrameOwner != nil {
		return
	}
	bytes := uint(unsafe.Sizeof(*record)) + uint(len(record.encoded)) + uint(cap(record.Sources))*uint(unsafe.Sizeof(catalogSourceDeclaration{})) + uint(cap(record.Checks))*uint(unsafe.Sizeof(catalogCheckDeclaration{}))
	for _, source := range record.Sources {
		bytes += uint(len(source.Name))
		if source.Definition != nil {
			source.Definition.retainSourceMemory(db)
		}
	}
	for _, declaration := range record.Checks {
		bytes += uint(len(declaration.Table)+len(declaration.Identity)) + declaration.Check.payloadFootprint
	}
	bytes += uint(cap(record.StatementTriggers)) * uint(unsafe.Sizeof(catalogStatementTriggerDeclaration{}))
	for _, declaration := range record.StatementTriggers {
		bytes += uint(len(declaration.Table) + len(declaration.Identity) + len(declaration.Name))
		if declaration.Header != nil {
			bytes += uint(unsafe.Sizeof(*declaration.Header))
			declaration.Header.statementDefinition.memory.retain(db)
		}
	}
	bytes += uint(cap(record.ColumnDefaults)) * uint(unsafe.Sizeof(catalogColumnDefaultDeclaration{}))
	for _, declaration := range record.ColumnDefaults {
		bytes += uint(len(declaration.Table) + len(declaration.Identity) + len(declaration.Column) + len(declaration.Expression) + len(declaration.Name))
		bytes += scm.ComputeSize(declaration.Value) - uint(unsafe.Sizeof(scm.Scmer{}))
	}
	record.sourceFrameOwner, record.sourceFrameBytes = db, bytes
	db.adjustMetadataMemory(0, bytes)
}
