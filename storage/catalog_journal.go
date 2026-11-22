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
const catalogCheckDropJournalPrefix = "catalog-v8/"
const catalogTableDropJournalPrefix = "catalog-v9/"
const catalogParticipantJournalPrefix = "catalog-v10/"
const catalogEnforcementParticipantJournalPrefix = "catalog-v11/"
const catalogForeignDropJournalPrefix = "catalog-v12/"
const catalogForeignHistoryJournalPrefix = "catalog-v13/"
const catalogColumnDeclarationJournalPrefix = "catalog-v14/"
const catalogNativeDeclarationJournalPrefix = "catalog-v15/"
const catalogColumnAppendJournalPrefix = "catalog-v16/"
const catalogColumnPropertyJournalPrefix = "catalog-v17/"
const catalogForeignValidationJournalPrefix = "catalog-v18/"
const catalogColumnInitializationJournalPrefix = "catalog-v19/"
const catalogColumnPropertyClaimJournalPrefix = "catalog-v20/"
const catalogOwnedDeclarationJournalPrefix = "catalog-v22/"
const catalogNameDeclarationJournalPrefix = "catalog-v23/"

type catalogDecisionRecord struct {
	encoded          string
	sourceFrameBytes uint
	sourceFrameOwner *database

	Format                 int                                  `json:"format"`
	Kind                   string                               `json:"kind"`
	Database               string                               `json:"database"`
	Incarnation            string                               `json:"incarnation"`
	Transaction            string                               `json:"transaction"`
	Decision               string                               `json:"decision"`
	Installation           string                               `json:"installation"`
	Boot                   string                               `json:"boot"`
	Namespace              string                               `json:"namespace,omitempty"`
	Version                uint64                               `json:"version"`
	Distributed            bool                                 `json:"distributed,omitempty"`
	AuthorityStorage       string                               `json:"authority_storage,omitempty"`
	Participants           []catalogParticipantIdentity         `json:"participants,omitempty"`
	TableDrops             []catalogTableDropDeclaration        `json:"table_drops,omitempty"`
	Tables                 []*table                             `json:"tables"`
	NativeDeclarations     []catalogNativeDeclaration           `json:"native_declarations,omitempty"`
	ForeignHistory         []catalogForeignAction               `json:"foreign_history,omitempty"`
	ForeignDrops           []catalogForeignDeclaration          `json:"foreign_drops,omitempty"`
	ForeignKeys            []catalogForeignDeclaration          `json:"foreign_keys,omitempty"`
	Sources                []catalogSourceDeclaration           `json:"sources,omitempty"`
	Checks                 []catalogCheckDeclaration            `json:"checks,omitempty"`
	CheckDrops             []catalogCheckDeclaration            `json:"check_drops,omitempty"`
	StatementTriggers      []catalogStatementTriggerDeclaration `json:"statement_triggers,omitempty"`
	ColumnDefaults         []catalogColumnDefaultDeclaration    `json:"column_defaults,omitempty"`
	ColumnDeclarations     []catalogColumnDeclaration           `json:"column_declarations,omitempty"`
	ColumnAppends          []catalogColumnAppend                `json:"column_appends,omitempty"`
	ColumnPropertyRemovals []catalogColumnPropertyRemoval       `json:"column_property_removals,omitempty"`
}

// Participant-bearing versions use the same original-T enrollment and settlement owner.
func (record *catalogDecisionRecord) hasRowParticipants() bool {
	return record.Format == 10 || record.Format == 11 || (record.Format == 12 || record.Format == 13 || record.Format == 14 || record.Format == 15 || record.Format == 16 || record.Format == 17 || record.Format == 18 || record.Format == 19 || record.Format == 20 || (record.Format == 22 || record.Format == 23)) && len(record.Participants) > 1
}

func (record *catalogDecisionRecord) hasPublicDeltas() bool {
	return record.hasEnforcementDeltas() || len(record.Sources) != 0 || len(record.ColumnDefaults) != 0 || len(record.ColumnDeclarations) != 0 || len(record.ColumnAppends) != 0 || len(record.ColumnPropertyRemovals) != 0
}

func (record *catalogDecisionRecord) hasEnforcementDeltas() bool {
	return len(record.NativeDeclarations) > 0 || len(record.ForeignHistory) > 0 || len(record.ForeignDrops) > 0 || len(record.ForeignKeys) > 0 || len(record.Checks) > 0 || len(record.CheckDrops) > 0 || len(record.StatementTriggers) > 0
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
	if record.Format == 23 {
		prefix = catalogNameDeclarationJournalPrefix
	} else if record.Format == 22 {
		prefix = catalogOwnedDeclarationJournalPrefix
	} else if record.Format == 20 {
		prefix = catalogColumnPropertyClaimJournalPrefix
	} else if record.Format == 19 {
		prefix = catalogColumnInitializationJournalPrefix
	} else if record.Format == 18 {
		prefix = catalogForeignValidationJournalPrefix
	} else if record.Format == 17 {
		prefix = catalogColumnPropertyJournalPrefix
	} else if record.Format == 16 {
		prefix = catalogColumnAppendJournalPrefix
	} else if record.Format == 15 {
		prefix = catalogNativeDeclarationJournalPrefix
	} else if record.Format == 14 {
		prefix = catalogColumnDeclarationJournalPrefix
	} else if record.Format == 13 {
		prefix = catalogForeignHistoryJournalPrefix
	} else if record.Format == 12 {
		prefix = catalogForeignDropJournalPrefix
	} else if record.Format == 11 {
		prefix = catalogEnforcementParticipantJournalPrefix
	} else if record.Format == 10 {
		prefix = catalogParticipantJournalPrefix
	} else if record.Format == 9 {
		prefix = catalogTableDropJournalPrefix
	} else if record.Format == 8 {
		prefix = catalogCheckDropJournalPrefix
	} else if record.Format == 7 {
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
	if strings.HasPrefix(value, catalogNameDeclarationJournalPrefix) {
		prefix = catalogNameDeclarationJournalPrefix
	} else if strings.HasPrefix(value, catalogOwnedDeclarationJournalPrefix) {
		prefix = catalogOwnedDeclarationJournalPrefix
	} else if strings.HasPrefix(value, catalogColumnPropertyClaimJournalPrefix) {
		prefix = catalogColumnPropertyClaimJournalPrefix
	} else if strings.HasPrefix(value, catalogColumnInitializationJournalPrefix) {
		prefix = catalogColumnInitializationJournalPrefix
	} else if strings.HasPrefix(value, catalogForeignValidationJournalPrefix) {
		prefix = catalogForeignValidationJournalPrefix
	} else if strings.HasPrefix(value, catalogColumnPropertyJournalPrefix) {
		prefix = catalogColumnPropertyJournalPrefix
	} else if strings.HasPrefix(value, catalogColumnAppendJournalPrefix) {
		prefix = catalogColumnAppendJournalPrefix
	} else if strings.HasPrefix(value, catalogNativeDeclarationJournalPrefix) {
		prefix = catalogNativeDeclarationJournalPrefix
	} else if strings.HasPrefix(value, catalogColumnDeclarationJournalPrefix) {
		prefix = catalogColumnDeclarationJournalPrefix
	} else if strings.HasPrefix(value, catalogForeignHistoryJournalPrefix) {
		prefix = catalogForeignHistoryJournalPrefix
	} else if strings.HasPrefix(value, catalogForeignDropJournalPrefix) {
		prefix = catalogForeignDropJournalPrefix
	} else if strings.HasPrefix(value, catalogEnforcementParticipantJournalPrefix) {
		prefix = catalogEnforcementParticipantJournalPrefix
	} else if strings.HasPrefix(value, catalogParticipantJournalPrefix) {
		prefix = catalogParticipantJournalPrefix
	} else if strings.HasPrefix(value, catalogTableDropJournalPrefix) {
		prefix = catalogTableDropJournalPrefix
	} else if strings.HasPrefix(value, catalogCheckDropJournalPrefix) {
		prefix = catalogCheckDropJournalPrefix
	} else if strings.HasPrefix(value, catalogStatementReplacementJournalPrefix) {
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
	if prefix == catalogParticipantJournalPrefix || prefix == catalogEnforcementParticipantJournalPrefix || prefix == catalogForeignDropJournalPrefix || prefix == catalogForeignHistoryJournalPrefix || prefix == catalogColumnDeclarationJournalPrefix || prefix == catalogNativeDeclarationJournalPrefix || prefix == catalogColumnPropertyJournalPrefix || prefix == catalogColumnAppendJournalPrefix || prefix == catalogForeignValidationJournalPrefix || prefix == catalogColumnInitializationJournalPrefix || prefix == catalogColumnPropertyClaimJournalPrefix || prefix == catalogOwnedDeclarationJournalPrefix || prefix == catalogNameDeclarationJournalPrefix {
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, err
		}
	}
	// Historical domains never acquire the new state bit, including an explicit
	// false spelling. The schema decoder records presence without changing its
	// old field/default semantics; only V18 and its explicit newer domains own
	// this durable declaration fact.
	if present, _ := record.foreignValidationFacts(); record.Format != 18 && record.Format != 19 && record.Format != 20 && record.Format != 22 && record.Format != 23 && present {
		return nil, fmt.Errorf("foreign validation state requires catalog version 18")
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
	// CHECK removals carry predecessor authority only in their new envelope.
	// Historical v1-v7 readers keep precisely their earlier accepted domains.
	validShape = validShape && len(record.CheckDrops) == 0 || record.Format == 8 && prefix == catalogCheckDropJournalPrefix && len(record.Tables) == 0 && len(record.CheckDrops) > 0
	// v1-v8 never acquire table-removal semantics. v9 shares only the
	// already existing DEFAULT/SOURCE participants and original row decision.
	validShape = validShape && len(record.TableDrops) == 0 || record.Format == 9 && prefix == catalogTableDropJournalPrefix && len(record.TableDrops) > 0 && len(record.Tables) == 0 && len(record.ForeignKeys) == 0 && len(record.Checks) == 0 && len(record.CheckDrops) == 0 && len(record.StatementTriggers) == 0
	// Cross-database recovery authority is exclusive to its new domain. None
	// of the historical catalog versions acquire participant semantics.
	validShape = validShape && len(record.Participants) == 0 && record.AuthorityStorage == "" ||
		record.Format == 10 && prefix == catalogParticipantJournalPrefix && record.Namespace == "" && record.AuthorityStorage != "" &&
			len(record.Participants) > 1 && len(record.Tables) == 0 && len(record.ForeignKeys) == 0 && len(record.Checks) == 0 &&
			len(record.CheckDrops) == 0 && len(record.StatementTriggers) == 0 &&
			(len(record.Sources) > 0 || len(record.ColumnDefaults) > 0 || len(record.TableDrops) > 0)
	// v1-v10 retain their exact domains. Enforcement plus independently
	// journaled local row databases requires this new strict authority envelope.
	validShape = validShape || record.Format == 11 && prefix == catalogEnforcementParticipantJournalPrefix &&
		record.Namespace == "" && record.AuthorityStorage != "" && len(record.Participants) > 1 &&
		len(record.Tables) == 0 && len(record.TableDrops) == 0 &&
		record.hasEnforcementDeltas()
	// Foreign retirement is new authority. Historical domains do not interpret
	// even an empty explicit retirement field; v12 owns its complete profile.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for key, raw := range fields {
		if strings.EqualFold(key, "column_appends") && record.Format != 23 {
			var appends []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &appends); err != nil {
				return nil, err
			}
			for _, d := range appends {
				for name := range d {
					if strings.EqualFold(name, "name_binding") {
						return nil, fmt.Errorf("appended name bindings require catalog version 23")
					}
				}
			}
		}
		if strings.EqualFold(key, "native_declarations") && record.Format != 23 {
			var actions []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &actions); err != nil {
				return nil, err
			}
			for _, action := range actions {
				for field := range action {
					if strings.EqualFold(field, "column_rename") || strings.EqualFold(field, "table_rename") {
						return nil, fmt.Errorf("native name actions require catalog version 23")
					}
				}
			}
		}
		if (record.Format == 22 || record.Format == 23) && (strings.EqualFold(key, "foreign_keys") || strings.EqualFold(key, "foreign_drops") || strings.EqualFold(key, "foreign_history") || record.Format == 22 && strings.EqualFold(key, "table_drops")) {
			return nil, fmt.Errorf("owned declarations require their single ordered native history")
		}
		if strings.EqualFold(key, "column_property_removals") && (record.Format == 20 || (record.Format == 22 || record.Format == 23) && record.hasColumnPropertyClaimActions()) {
			return nil, fmt.Errorf("ordered column properties cannot use an unordered removal field")
		}
		if strings.EqualFold(key, "native_declarations") && record.Format != 20 && record.Format != 22 && record.Format != 23 {
			var actions []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &actions); err != nil {
				return nil, err
			}
			for _, action := range actions {
				for field := range action {
					if strings.EqualFold(field, "column_property") {
						return nil, fmt.Errorf("column property claims require catalog version 20")
					}
				}
			}
		}
		if strings.EqualFold(key, "column_appends") && record.Format != 16 && record.Format != 17 && record.Format != 18 && record.Format != 19 && record.Format != 20 && record.Format != 22 && record.Format != 23 {
			return nil, fmt.Errorf("nullable appends require their native journal version")
		}
		if strings.EqualFold(key, "column_property_removals") && record.Format != 17 && record.Format != 18 && record.Format != 19 && record.Format != 22 && record.Format != 23 {
			return nil, fmt.Errorf("column property removal requires its native journal version")
		}
		if strings.EqualFold(key, "native_declarations") && record.Format != 15 && record.Format != 16 && record.Format != 17 && record.Format != 18 && record.Format != 19 && record.Format != 20 && record.Format != 22 && record.Format != 23 {
			return nil, fmt.Errorf("ordered native declarations require their journal version")
		}
		if strings.EqualFold(key, "foreign_history") && record.Format != 13 {
			return nil, fmt.Errorf("foreign history requires its native journal version")
		}
		if strings.EqualFold(key, "column_declarations") && record.Format != 14 && record.Format != 15 && record.Format != 16 && record.Format != 17 && record.Format != 18 && record.Format != 19 && record.Format != 20 && record.Format != 22 && record.Format != 23 {
			return nil, fmt.Errorf("column declarations require their native journal version")
		}
		if strings.EqualFold(key, "foreign_drops") && record.Format != 12 {
			return nil, fmt.Errorf("foreign removal requires its native journal version")
		}
	}
	validShape = validShape && len(record.ForeignDrops) == 0 || record.Format == 12 && prefix == catalogForeignDropJournalPrefix &&
		len(record.ForeignDrops) > 0 && record.Namespace == "" && validLocalCatalogTransaction(record.Transaction, record.Boot) && len(record.Tables) == 0 && len(record.TableDrops) == 0 &&
		((len(record.Participants) == 0 && record.AuthorityStorage == "") ||
			(record.Namespace == "" && record.AuthorityStorage != "" && len(record.Participants) > 1))
	validShape = validShape && len(record.ForeignHistory) == 0 || record.Format == 13 && prefix == catalogForeignHistoryJournalPrefix &&
		len(record.ForeignHistory) > 0 && len(record.ForeignKeys) == 0 && len(record.ForeignDrops) == 0 && len(record.Tables) == 0 && len(record.TableDrops) == 0 &&
		record.Namespace == "" && validLocalCatalogTransaction(record.Transaction, record.Boot) &&
		((len(record.Participants) == 0 && record.AuthorityStorage == "") || (record.AuthorityStorage != "" && len(record.Participants) > 1))
	// New versions have one declaration authority: V14 adds only same-slot
	// column contracts; V15 composes those contracts with its ordered key/FK
	// program. Earlier version domains above remain unchanged.
	switch record.Format {
	case 14, 15, 16, 17, 18, 19, 20:
		validShape = record.Namespace == "" && validLocalCatalogTransaction(record.Transaction, record.Boot) &&
			len(record.Tables) == 0 && len(record.TableDrops) == 0 && len(record.ForeignKeys) == 0 &&
			len(record.ForeignDrops) == 0 && len(record.ForeignHistory) == 0 &&
			((len(record.Participants) == 0 && record.AuthorityStorage == "") || (record.AuthorityStorage != "" && len(record.Participants) > 1))
		if record.Format == 14 {
			validShape = validShape && prefix == catalogColumnDeclarationJournalPrefix && len(record.ColumnDeclarations) > 0 && len(record.NativeDeclarations) == 0
		} else if record.Format == 15 {
			validShape = validShape && prefix == catalogNativeDeclarationJournalPrefix && len(record.NativeDeclarations) > 0
		} else if record.Format == 16 {
			validShape = validShape && prefix == catalogColumnAppendJournalPrefix && len(record.ColumnAppends) > 0
		} else if record.Format == 17 {
			validShape = validShape && prefix == catalogColumnPropertyJournalPrefix && len(record.ColumnPropertyRemovals) > 0
		} else if record.Format == 20 {
			validShape = validShape && prefix == catalogColumnPropertyClaimJournalPrefix && len(record.NativeDeclarations) > 0 && record.hasColumnPropertyClaimActions()
		} else if record.Format == 18 {
			_, untrusted := record.foreignValidationFacts()
			validShape = validShape && prefix == catalogForeignValidationJournalPrefix && len(record.NativeDeclarations) > 0 && untrusted
		} else {
			initialized := false
			for _, d := range record.ColumnAppends {
				initialized = initialized || d.Initialization != nil
			}
			validShape = validShape && prefix == catalogColumnInitializationJournalPrefix && initialized
		}
	default:
		validShape = validShape && len(record.ColumnDeclarations) == 0 && len(record.NativeDeclarations) == 0
	}
	if record.Format == 22 {
		validShape = prefix == catalogOwnedDeclarationJournalPrefix && record.Namespace == "" &&
			validLocalCatalogTransaction(record.Transaction, record.Boot) && len(record.Tables) > 0 && record.hasPublicDeltas() &&
			len(record.TableDrops) == 0 && len(record.ForeignKeys) == 0 && len(record.ForeignDrops) == 0 && len(record.ForeignHistory) == 0 &&
			((len(record.Participants) == 0 && record.AuthorityStorage == "") || (record.AuthorityStorage != "" && len(record.Participants) > 1))
	}
	if record.Format == 23 {
		validShape = prefix == catalogNameDeclarationJournalPrefix && record.Namespace == "" &&
			validLocalCatalogTransaction(record.Transaction, record.Boot) && (record.hasNameDeclarationActions() || record.hasOwnedKeyDeclarationActions() || len(record.Tables) > 0 && len(record.TableDrops) > 0) &&
			len(record.ForeignKeys) == 0 && len(record.ForeignDrops) == 0 && len(record.ForeignHistory) == 0 &&
			((len(record.Participants) == 0 && record.AuthorityStorage == "") || (record.AuthorityStorage != "" && len(record.Participants) > 1))
	}
	if record.Format != 23 && record.hasOwnedKeyDeclarationActions() {
		return nil, fmt.Errorf("owned data-phase key actions require catalog version 23")
	}
	if record.Format != 16 && record.Format != 17 && record.Format != 18 && record.Format != 19 && record.Format != 20 && record.Format != 22 && record.Format != 23 {
		validShape = validShape && len(record.ColumnAppends) == 0
	}
	// New initialization authority is never inferred in old domains, even for
	// an explicit null field. V20 composes the V19 recipe with ordered property
	// claims; nullable-only V16 recipes retain their old shape.
	if record.Format != 19 && record.Format != 20 && record.Format != 22 && record.Format != 23 {
		for key, raw := range fields {
			if strings.EqualFold(key, "column_appends") {
				var appends []map[string]json.RawMessage
				if err := json.Unmarshal(raw, &appends); err != nil {
					return nil, err
				}
				for _, d := range appends {
					for name := range d {
						if strings.EqualFold(name, "initialization") {
							return nil, fmt.Errorf("initialized append requires its native journal version")
						}
					}
				}
			}
		}
	}
	if err := validateColumnAppendShape(record.ColumnAppends); err != nil {
		return nil, err
	}
	if err := validateColumnPropertyRemovalShape(record.ColumnPropertyRemovals); err != nil {
		return nil, err
	}
	if err := record.validateColumnPropertyProfile(); err != nil {
		return nil, err
	}
	if len(record.NativeDeclarations) > 0 {
		if err := validateNativeDeclarationShape(record.NativeDeclarations); err != nil {
			return nil, err
		}
	}
	if len(record.ForeignHistory) > 0 {
		if err := validateForeignActionOrder(record.ForeignHistory); err != nil {
			return nil, err
		}
	}
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
	participantNames := make(map[string]bool)
	for _, participant := range record.Participants {
		if !validCatalogParticipant(participant) || participantNames[participant.Database] {
			return nil, fmt.Errorf("invalid native catalog participant inventory")
		}
		participantNames[participant.Database] = true
		if participant.Database == record.Database && participant.Incarnation != record.Incarnation {
			return nil, fmt.Errorf("catalog participant authority incarnation mismatch")
		}
	}
	if record.hasRowParticipants() && (!participantNames[record.Database] || !validLocalCatalogTransaction(record.Transaction, record.Boot)) {
		return nil, fmt.Errorf("cross-database catalog decision requires a local transaction")
	}
	names := make(map[string]bool)
	ownedIdentities := make(map[string]bool)
	for _, table := range record.Tables {
		if table == nil || len(table.Checks) != 0 || table.Name == "" || names[table.Name] || table.CreatorTxID != record.Transaction || table.CatalogDecisionID != record.Decision || table.CreatorIncarnation != record.Incarnation || table.CreatorNamespace != record.Namespace || table.CatalogVersion.Load() != record.Version {
			return nil, fmt.Errorf("invalid catalog declaration identity")
		}
		if _, err := uuid.Parse(table.RuntimeIdentity); err != nil {
			return nil, fmt.Errorf("invalid catalog table identity")
		}
		if record.Format != 23 {
			if table.rawVisibleName() != table.Name {
				return nil, fmt.Errorf("bound table names require catalog version 23")
			}
			for _, c := range table.Columns {
				if c.visibleName != "" {
					return nil, fmt.Errorf("bound column names require catalog version 23")
				}
			}
		}
		if record.Format == 22 || record.Format == 23 {
			// The new joint envelope has exactly one ordered FK authority.
			// Created generations must also be distinct from one another.
			identity, _ := uuid.Parse(table.RuntimeIdentity)
			if identity == uuid.Nil || ownedIdentities[identity.String()] || len(table.Foreign) != 0 {
				return nil, fmt.Errorf("invalid owned catalog generation")
			}
			ownedIdentities[identity.String()] = true
		}
		for _, trigger := range table.Triggers {
			if trigger.Timing == ReplaceStatement {
				return nil, fmt.Errorf("native statement replacement requires its declaration envelope")
			}
		}
		names[table.Name] = true
	}
	for _, drop := range record.TableDrops {
		if drop.Name == "" || names[drop.Name] {
			return nil, fmt.Errorf("invalid catalog table removal")
		}
		if identity, err := uuid.Parse(drop.Identity); err != nil || identity == uuid.Nil {
			return nil, fmt.Errorf("invalid catalog table removal identity")
		}
		names[drop.Name] = true
	}
	nameKey := strings.ToLower
	if record.Format == 13 || record.Format == 14 || record.Format == 15 || record.Format == 16 || record.Format == 17 || record.Format == 18 || record.Format == 19 || record.Format == 20 || record.Format == 22 || record.Format == 23 {
		nameKey = scm.CaseFoldKey
	}
	foreignNames := make(map[string]bool)
	for _, declaration := range append(append([]catalogForeignDeclaration(nil), record.ForeignKeys...), record.ForeignDrops...) {
		fk := declaration.Foreign
		if fk.Id == "" || fk.Tbl1 == "" || fk.Tbl2 == "" || len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) || fk.Updatemode > SETNULL || fk.Deletemode > SETNULL || foreignNames[nameKey(fk.Id)] {
			return nil, fmt.Errorf("invalid native foreign declaration delta")
		}
		// A self delta is one native endpoint with two RESTRICT roles. The
		// ordinary apply/replay owner still proves that current endpoint.
		if fk.Tbl1 == fk.Tbl2 && (fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT || declaration.ChildIdentity != declaration.ParentIdentity) {
			return nil, fmt.Errorf("invalid native self foreign declaration delta")
		}
		foreignNames[nameKey(fk.Id)] = true
		for _, identity := range []string{declaration.ChildIdentity, declaration.ParentIdentity} {
			if _, err := uuid.Parse(identity); err != nil {
				return nil, fmt.Errorf("invalid native foreign endpoint identity")
			}
		}
	}

	for _, action := range record.ForeignHistory {
		declaration := action.Declaration
		fk := declaration.Foreign
		if fk.Id == "" || fk.Tbl1 == "" || fk.Tbl2 == "" || len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) || fk.Updatemode > SETNULL || fk.Deletemode > SETNULL {
			return nil, fmt.Errorf("invalid native foreign history recipe")
		}
		if fk.Tbl1 == fk.Tbl2 && (fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT || declaration.ChildIdentity != declaration.ParentIdentity) {
			return nil, fmt.Errorf("invalid native self foreign history")
		}
		for _, identity := range []string{declaration.ChildIdentity, declaration.ParentIdentity} {
			if _, err := uuid.Parse(identity); err != nil {
				return nil, fmt.Errorf("invalid foreign history endpoint")
			}
		}
		foreignNames[nameKey(fk.Id)] = true
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
		name := nameKey(declaration.Check.Id)
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
	alteredColumns := make(map[string]bool)
	for _, declaration := range record.ColumnDeclarations {
		key := declaration.Table + "\x00" + declaration.Column
		if declaration.Table == "" || declaration.Column == "" || !utf8.ValidString(declaration.Table) || !utf8.ValidString(declaration.Column) || alteredColumns[key] {
			return nil, fmt.Errorf("invalid native column declaration delta")
		}
		if _, err := uuid.Parse(declaration.Identity); err != nil {
			return nil, fmt.Errorf("invalid native column endpoint identity")
		}
		if declaration.Before.Type == "" || !utf8.ValidString(declaration.Before.Type) || !utf8.ValidString(declaration.Before.Collation) ||
			!declaration.After.compatibleWith(declaration.Before) {
			return nil, fmt.Errorf("invalid native same-slot column declaration")
		}
		alteredColumns[key] = true
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
			name := nameKey(declaration.Name)
			if foreignNames[name] {
				return nil, fmt.Errorf("duplicate native constraint declaration delta")
			}
			foreignNames[name] = true
		}
		defaultColumns[key] = true
	}
	for _, declaration := range record.CheckDrops {
		if declaration.Table == "" || !utf8.ValidString(declaration.Table) || declaration.Check == nil {
			return nil, fmt.Errorf("invalid native check removal delta")
		}
		if err := declaration.Check.validate(); err != nil {
			return nil, err
		}
		if _, err := uuid.Parse(declaration.Identity); err != nil {
			return nil, fmt.Errorf("invalid native check removal endpoint identity")
		}
		name := nameKey(declaration.Check.Id)
		if foreignNames[name] {
			return nil, fmt.Errorf("duplicate native constraint removal delta")
		}
		foreignNames[name] = true
	}
	if record.Format == 19 || record.Format == 20 || record.Format == 22 || record.Format == 23 {
		overridden := make(map[string]bool)
		for _, d := range record.ColumnDefaults {
			overridden[d.Table+"\x00"+d.Identity+"\x00"+d.Column] = true
		}
		for _, d := range record.ColumnAppends {
			if d.Initialization != nil && d.Initialization.DefaultName != "" && !overridden[d.Table+"\x00"+d.Identity+"\x00"+d.Name] {
				name := nameKey(d.Initialization.DefaultName)
				if foreignNames[name] {
					return nil, fmt.Errorf("duplicate initialized column default name")
				}
				foreignNames[name] = true
			}
		}
	}
	if record.Format == 15 || record.Format == 16 || record.Format == 17 || record.Format == 18 || record.Format == 19 || record.Format == 20 || record.Format == 22 || record.Format == 23 {
		// Compare only final active names with the other retained declaration
		// families; canceled/replaced native actions stay inert ordered history.
		keys := make(map[string]*catalogKeyDeclaration)
		foreign := make(map[catalogForeignIdentity]*catalogForeignDeclaration)
		for _, action := range record.NativeDeclarations {
			switch action.Operation {
			case "key_add":
				keys[action.Key.Identity+"/"+scm.CaseFoldKey(action.Key.Key.Id)] = action.Key
			case "key_drop":
				delete(keys, action.Key.Identity+"/"+scm.CaseFoldKey(action.Key.Key.Id))
			case "foreign_add":
				foreign[foreignIdentity(*action.Foreign)] = action.Foreign
			case "foreign_drop":
				delete(foreign, foreignIdentity(*action.Foreign))
			}
		}
		reserve := func(name string) bool {
			if name == "" {
				return true
			}
			key := scm.CaseFoldKey(name)
			if foreignNames[key] {
				return false
			}
			foreignNames[key] = true
			return true
		}
		for _, key := range keys {
			if !reserve(key.Key.ConstraintName) {
				return nil, fmt.Errorf("duplicate native key constraint declaration")
			}
		}
		for _, fk := range foreign {
			if !reserve(fk.Foreign.Id) {
				return nil, fmt.Errorf("duplicate native foreign constraint declaration")
			}
		}
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
	participant, err := decodeCatalogParticipant(value)
	if err != nil {
		return err
	}
	if participant != nil {
		return db.acceptCatalogParticipant(participant, committed, generation)
	}
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
	if err := db.settleCatalogParticipantDecisions(false); err != nil {
		panic(err)
	}
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
			// A later accepted record can amend this recovered CREATE before
			// the loader's final descriptor/program reconstruction pass.
			db.initializeRecoveryColumnMetadata(table)
			db.tables.Set(table.Name, table)
		}
		db.recoverColumnAppends(declarations.ColumnAppends)
		if len(declarations.ColumnDeclarations) != 0 {
			db.recoverColumnDefaultDeclarations(declarations.ColumnDefaults)
			db.recoverColumnDeclarations(declarations.ColumnDeclarations)
		}
		db.recoverNativeDeclarations(declarations.NativeDeclarations, catalogNativeCurrent, recoveredForeignEndpointScope(db, declarations))
		db.recoverPublicForeignHistory(declarations.ForeignHistory)
		db.recoverPublicForeignDrops(declarations.ForeignDrops, catalogForeignEndpointScope{})
		db.recoverPublicForeignDeclarations(declarations.ForeignKeys, catalogForeignEndpointScope{})
		db.recoverPublicCheckDeclarations(declarations.Checks)
		db.recoverPublicCheckDrops(declarations.CheckDrops)
		db.recoverSourceDeclarations(declarations.Sources)
		db.recoverStatementTriggerDeclarations(declarations.StatementTriggers)
		if len(declarations.ColumnDeclarations) == 0 {
			db.recoverColumnDefaultDeclarations(declarations.ColumnDefaults)
		}
		db.recoverColumnPropertyRemovals(declarations.ColumnPropertyRemovals)
		db.recoverTableDrops(declarations.TableDrops)
		db.catalogVersion = record.Version
	}
	db.catalogProofVersion.Store(db.catalogVersion)
}

// The existing WAL recovery map retains the immutable envelope and its decoded
// source payload until database retirement. Row-log compaction deliberately keeps
// these catalog records. Count that actual extra allocation, not only live names.
// Caller owns transactionMu or is constructing this database privately.
func (record *catalogDecisionRecord) retainSourceFrameMemory(db *database) {
	if len(record.Sources) == 0 && len(record.Checks) == 0 && len(record.StatementTriggers) == 0 && len(record.ColumnDefaults) == 0 && len(record.ColumnDeclarations) == 0 && len(record.ColumnPropertyRemovals) == 0 && len(record.ColumnAppends) == 0 && len(record.CheckDrops) == 0 && len(record.TableDrops) == 0 && len(record.ForeignDrops) == 0 && len(record.ForeignHistory) == 0 && len(record.NativeDeclarations) == 0 && !record.hasRowParticipants() || record.sourceFrameOwner != nil {
		return
	}
	bytes := uint(unsafe.Sizeof(*record)) + uint(len(record.encoded)) + uint(cap(record.Sources))*uint(unsafe.Sizeof(catalogSourceDeclaration{})) + uint(cap(record.Checks))*uint(unsafe.Sizeof(catalogCheckDeclaration{}))
	if record.Format == 11 || record.Format == 12 || record.Format == 13 || record.Format == 14 || record.Format == 15 || record.Format == 16 || record.Format == 17 || record.Format == 18 || record.Format == 19 || record.Format == 20 || record.Format == 22 || record.Format == 23 {
		bytes += uint(cap(record.ForeignKeys)) * uint(unsafe.Sizeof(catalogForeignDeclaration{}))
		for _, declaration := range record.ForeignKeys {
			bytes += uint(len(declaration.ChildIdentity)+len(declaration.ParentIdentity)) + declaration.Foreign.payloadFootprint
		}
	}
	bytes += uint(cap(record.ForeignDrops)) * uint(unsafe.Sizeof(catalogForeignDeclaration{}))
	for _, declaration := range record.ForeignDrops {
		bytes += uint(len(declaration.ChildIdentity)+len(declaration.ParentIdentity)) + declaration.Foreign.payloadFootprint
	}
	bytes += uint(cap(record.ForeignHistory)) * uint(unsafe.Sizeof(catalogForeignAction{}))
	for _, action := range record.ForeignHistory {
		d := action.Declaration
		bytes += uint(len(action.Operation)+len(d.ChildIdentity)+len(d.ParentIdentity)) + d.Foreign.payloadFootprint
	}
	bytes += uint(cap(record.NativeDeclarations)) * uint(unsafe.Sizeof(catalogNativeDeclaration{}))
	for _, action := range record.NativeDeclarations {
		bytes += uint(len(action.Operation))
		if action.Foreign != nil {
			d := action.Foreign
			bytes += uint(unsafe.Sizeof(*d)) + uint(len(d.ChildIdentity)+len(d.ParentIdentity)) + d.Foreign.payloadFootprint
		}
		if action.Key != nil {
			d := action.Key
			bytes += uint(unsafe.Sizeof(*d)) + uint(len(d.Table)+len(d.Identity)) + uniqueKeyPayloadBytes(d.Key)
		}
		if action.Rename != nil {
			bytes += action.Rename.memory()
		}
		if action.TableRename != nil {
			d := action.TableRename
			bytes += uint(unsafe.Sizeof(*d)) + uint(len(d.Storage)+len(d.Identity)+len(d.Before)+len(d.After))
		}
		if action.Property != nil {
			bytes += action.Property.memory()
		}
	}
	bytes += uint(len(record.AuthorityStorage)) + uint(cap(record.Participants))*uint(unsafe.Sizeof(catalogParticipantIdentity{}))
	for _, participant := range record.Participants {
		bytes += uint(len(participant.Database) + len(participant.Incarnation))
	}
	for _, source := range record.Sources {
		bytes += uint(len(source.Name))
		if source.Definition != nil {
			source.Definition.retainSourceMemory(db)
		}
	}
	for _, declaration := range record.Checks {
		bytes += uint(len(declaration.Table)+len(declaration.Identity)) + declaration.Check.payloadFootprint
	}
	bytes += uint(cap(record.TableDrops)) * uint(unsafe.Sizeof(catalogTableDropDeclaration{}))
	for _, drop := range record.TableDrops {
		bytes += uint(len(drop.Name) + len(drop.Identity))
	}
	bytes += uint(cap(record.CheckDrops)) * uint(unsafe.Sizeof(catalogCheckDeclaration{}))
	for _, declaration := range record.CheckDrops {
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
	bytes += uint(cap(record.ColumnAppends)-len(record.ColumnAppends)) * uint(unsafe.Sizeof(catalogColumnAppend{}))
	for i := range record.ColumnAppends {
		bytes += record.ColumnAppends[i].memory()
	}
	bytes += uint(cap(record.ColumnDeclarations)-len(record.ColumnDeclarations)) * uint(unsafe.Sizeof(catalogColumnDeclaration{}))
	for i := range record.ColumnDeclarations {
		bytes += record.ColumnDeclarations[i].memory()
	}
	bytes += uint(cap(record.ColumnPropertyRemovals)-len(record.ColumnPropertyRemovals)) * uint(unsafe.Sizeof(catalogColumnPropertyRemoval{}))
	for i := range record.ColumnPropertyRemovals {
		bytes += record.ColumnPropertyRemovals[i].memory()
	}
	record.sourceFrameOwner, record.sourceFrameBytes = db, bytes
	db.adjustMetadataMemory(0, bytes)
}
