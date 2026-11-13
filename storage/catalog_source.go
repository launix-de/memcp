/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"sort"
	"unicode/utf8"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// Native source metadata uses the existing catalog decision, not a row table or
// a second transaction registry. Nil Definition is an actual name deletion.
type catalogSourceDeclaration struct {
	Name       string                `json:"name"`
	Definition *ProcedureDescription `json:"definition,omitempty"`
}

type catalogSourceAmendment struct {
	name    string
	before  *ProcedureDescription
	current *ProcedureDescription
}

// One point lookup selects the caller's pending definition or the committed
// predecessor. Repeated replacements never form a row-time predecessor chain.
func (db *database) visibleProcedure(name string, tx *TxContext) *ProcedureDescription {
	for {
		view := db.catalogView()
		definition := db.procedures.Get(name)
		if definition == nil {
			return nil
		}
		version := definition.publicationVersion.Load()
		visible := definition.pendingDecision == "" || version != 0 && version <= view.version
		own := tx != nil && tx.State == TxActive && tx.pendingCatalogKey() == definition.pendingDecision
		if own && definition.sourceReservation {
			return definition.previousCommitted.Load()
		}
		if visible || own {
			if definition.removed {
				return nil
			}
			return definition
		}
		if previous := definition.previousCommitted.Load(); previous != nil {
			return previous
		}
		// Close may have retired the predecessor after catalog publication.
		// Do not turn an earlier sampled version into a false missing result.
		if db.catalogView().version == view.version {
			return nil
		}
	}
}

func (definition *ProcedureDescription) pendingSource(version uint64) bool {
	return definition != nil && definition.pendingDecision != "" &&
		(definition.publicationVersion.Load() == 0 || definition.publicationVersion.Load() > version)
}

func (tx *TxContext) stageCatalogSource(db *database, definition *ProcedureDescription, name string, replace, mustExist, ignore bool) bool {
	if tx == nil || tx.transactionState == nil {
		panic("source declaration requires an active native transaction")
	}
	finish := tx.beginCatalogConstruction(false)
	defer finish()
	if tx.State != TxActive || tx.decisionStarted {
		panic("source declaration requires an active native transaction")
	}
	if tx.coordinator != nil || tx.catalogIntents.Load() != nil {
		panic("source declarations cannot join a private or distributed catalog transaction")
	}
	if name == "" || !utf8.ValidString(name) {
		panic("source declaration requires a nonempty UTF-8 name")
	}
	if definition != nil && (!utf8.ValidString(definition.Source) || !utf8.ValidString(definition.Language)) {
		panic("source declaration requires UTF-8 source and language")
	}
	return tx.stagePublicCatalogDeclaration(db, func(changes *publicCatalogChanges) bool {
		return changes.stageSource(tx, definition, name, replace, mustExist, ignore)
	})
}

func (changes *publicCatalogChanges) stageSource(tx *TxContext, definition *ProcedureDescription, name string, replace, mustExist, ignore bool) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	if changes.owner != tx.transactionState || changes.db != GetDatabase(changes.db.Name) {
		panic("source declaration lost its native database owner")
	}
	db := changes.db
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	current := db.procedures.Get(name)
	if current.pendingSource(db.catalogView().version) && current.pendingDecision != tx.pendingCatalogKey() {
		panic("source declaration name is reserved by another native transaction: " + name)
	}
	visible := db.visibleProcedure(name, tx)
	if mustExist && visible == nil {
		panic("procedure does not exist: " + name)
	}
	if definition == nil && visible == nil {
		if ignore {
			return false
		}
		panic("procedure does not exist: " + name)
	}
	if definition != nil && visible != nil && !replace {
		panic("procedure already exists: " + name)
	}
	if changes.sourceHeads == nil {
		changes.sourceHeads = make(map[string]*ProcedureDescription)
		changes.sourceBaselines = make(map[string]*ProcedureDescription)
		changes.sourceReservations = make(map[string]*ProcedureDescription)
	}
	baseline, staged := changes.sourceBaselines[name]
	if !staged {
		baseline = db.visibleProcedure(name, nil)
	}
	if definition == nil {
		definition = &ProcedureDescription{Name: name, removed: true}
		if err := definition.initializeSourceMemory(); err != nil {
			panic(err)
		}
	}
	definition.pendingDecision = tx.pendingCatalogKey()
	definition.previousCommitted.Store(baseline)
	amendment := &catalogSourceAmendment{name: name, before: changes.sourceHeads[name], current: definition}
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{source: amendment})
	// Every allocated private definition is retained by its exact history entry.
	definition.retainSourceMemory(db)
	if !staged {
		reservation := &ProcedureDescription{Name: name, pendingDecision: tx.pendingCatalogKey(), sourceReservation: true}
		reservation.previousCommitted.Store(baseline)
		if err := reservation.initializeSourceMemory(); err != nil {
			panic(err)
		}
		reservation.retainSourceMemory(db)
		changes.sourceReservations[name] = reservation
		changes.sourceBaselines[name] = baseline
		if baseline != nil {
			baseline.retainSourceMemory(db)
		}
	}
	changes.sourceHeads[name] = definition
	changes.sourceHighWater = max(changes.sourceHighWater, len(changes.sourceBaselines))
	changes.sourceHistoryBytes += uint(unsafe.Sizeof(*amendment))
	tx.catalogAmendments.Store(&history)
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	db.procedures.Set(name, definition)
	return true
}

// The shared declaration owner has excluded another same-transaction producer.
// Schema publication checks retain the same concrete committed roots, not names.
func (changes *publicCatalogChanges) validateSourcesLocked() error {
	for name, baseline := range changes.sourceBaselines {
		current := changes.db.procedures.Get(name)
		expected := changes.sourceHeads[name]
		if expected == nil {
			expected = changes.sourceReservations[name]
		}
		if current != expected {
			return fmt.Errorf("source declaration reservation changed: %s", name)
		}
		if current != nil && current.previousCommitted.Load() != baseline {
			return fmt.Errorf("source declaration predecessor changed: %s", name)
		}
	}
	return nil
}

func (tx *TxContext) publicSourceDeclarations() []catalogSourceDeclaration {
	changes := tx.publicCatalogChanges()
	if changes == nil {
		return nil
	}
	changes.mu.Lock()
	defer changes.mu.Unlock()
	result := make([]catalogSourceDeclaration, 0, len(changes.sourceHeads))
	for name, definition := range changes.sourceHeads {
		var payload *ProcedureDescription
		if !definition.removed {
			payload = definition
		}
		result = append(result, catalogSourceDeclaration{Name: name, Definition: payload})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Catalog version publication is the single visibility boundary for all names.
// No compiler or program graph is evaluated/sized beneath this schema owner.
func (changes *publicCatalogChanges) publishSourcesLocked(version uint64) {
	for name, definition := range changes.sourceHeads {
		if !definition.removed {
			definition.retainSourceMemory(changes.db) // the public name now owns it
		}
		definition.publicationVersion.Store(version)
		if baseline := changes.sourceBaselines[name]; baseline != nil {
			baseline.releaseSourceMemory(changes.db) // private baseline still retains it
		}
	}
}

func (changes *publicCatalogChanges) finishSourcePublicationLocked() {
	for name, definition := range changes.sourceHeads {
		if definition.removed {
			changes.db.procedures.Remove(name)
		}
	}
}

// Restore one source amendment before releasing its private payload. The first
// name reservation remains until Close even at SP0, just like endpoint gates.
func (changes *publicCatalogChanges) restoreSourceLocked(amendment *catalogSourceAmendment) {
	if amendment.before != nil {
		changes.sourceHeads[amendment.name] = amendment.before
		changes.db.procedures.Set(amendment.name, amendment.before)
	} else {
		delete(changes.sourceHeads, amendment.name)
		changes.db.procedures.Set(amendment.name, changes.sourceReservations[amendment.name])
	}
	amendment.current.releaseSourceMemory(changes.db)
	changes.sourceHistoryBytes -= uint(unsafe.Sizeof(*amendment))
}

func (changes *publicCatalogChanges) closeSourcesLocked() {
	if len(changes.sourceBaselines) == 0 {
		return
	}
	access := changes.db.GetWrite(nil)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	committed := changes.owner.State == TxCommitted
	for name, baseline := range changes.sourceBaselines {
		head := changes.db.procedures.Get(name)
		if head != nil && head.pendingDecision == changes.owner.pendingCatalogKey() && (!committed || head.sourceReservation) {
			if baseline == nil {
				changes.db.procedures.Remove(name)
			} else {
				changes.db.procedures.Set(name, baseline)
			}
		}
		if committed && head != nil && !head.sourceReservation && head.pendingDecision == changes.owner.pendingCatalogKey() && head.publicationVersion.Load() != 0 {
			head.previousCommitted.Store(nil)
		}
		if baseline != nil {
			baseline.releaseSourceMemory(changes.db)
		}
		changes.sourceReservations[name].releaseSourceMemory(changes.db)
	}
	if history := changes.owner.catalogAmendments.Load(); history != nil {
		for _, amendment := range *history {
			if amendment.source != nil {
				amendment.source.current.releaseSourceMemory(changes.db)
			}
		}
	}
	changes.sourceHeads = nil
	changes.sourceReservations = nil
	changes.sourceBaselines = nil
}

func sourceTransaction(transactions []*TxContext) *TxContext {
	if len(transactions) == 0 {
		return nil
	}
	if len(transactions) != 1 {
		panic("source declaration expects one native transaction")
	}
	return transactions[0]
}

func sourceSCMTransaction(arguments []scm.Scmer, index int) *TxContext {
	if len(arguments) <= index {
		return nil
	}
	if len(arguments) != index+1 {
		panic("too many source declaration arguments")
	}
	return scmerToTxContext(arguments[index])
}

// Recovery owns a private decoded schema; source callables are regenerated only
// at the ordinary outside-lock language compiler boundary on later invocation.
func (db *database) recoverSourceDeclarations(declarations []catalogSourceDeclaration) {
	for _, source := range declarations {
		if previous := db.procedures.Get(source.Name); previous != nil {
			previous.releaseSourceMemory(db)
		}
		if source.Definition == nil {
			db.procedures.Remove(source.Name)
		} else {
			source.Definition.retainSourceMemory(db)
			db.procedures.Set(source.Name, source.Definition)
		}
	}
}

// Native declaration replacement is distinct from upsert. SQL frontends choose
// the mode, while the actual name-presence proof remains atomic with admission.
type sourceProcedureOptions struct {
	transaction *TxContext
	mustExist   bool
}

func sourceCreationOptions(options []sourceProcedureOptions) sourceProcedureOptions {
	if len(options) == 0 {
		return sourceProcedureOptions{}
	}
	if len(options) != 1 {
		panic("source declaration expects one option set")
	}
	return options[0]
}

func sourceSCMCreationOptions(arguments []scm.Scmer) sourceProcedureOptions {
	if len(arguments) > 8 {
		panic("too many source declaration arguments")
	}
	var result sourceProcedureOptions
	if len(arguments) > 6 {
		result.transaction = scmerToTxContext(arguments[6])
	}
	if len(arguments) > 7 {
		result.mustExist = scm.ToBool(arguments[7])
	}
	return result
}
