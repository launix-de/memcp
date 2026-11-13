/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "unsafe"
import "sync/atomic"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

// Only the outside-lock public ADD and native7 producers capture this
// optional descriptor after buildFKTriggers finalizes fresh private programs.
// The inline legacy CREATE producer still compiles under schema ownership and
// generic lazy loaders have no outside-lock caller contract. Both remain
// descriptor-absent/container-only; this patch never sizes there.
// ComputeSize is an inclusive INITIAL
// footprint estimate: small shared symbols/scalar strings may be overestimated;
// global environments are not visited. Ordinary callbacks, future mutable/JIT
// growth and borrowed table/column/UNIQUE/catalog trees are outside this profile.
type initialFKProgramFootprint struct {
	bytes uint
}

// Native fresh records borrow endpoint/column string payloads, but own their
// new column-name vectors and declaration ID. A same-allocation parent mirror
// borrows those arrays; its zero fact prevents charging them a second time.
func ownForeignPayload(key foreignKey) foreignKey {
	key.payloadFootprint = uint(cap(key.Cols1)+cap(key.Cols2))*uint(unsafe.Sizeof("")) + uint(len(key.Id))
	return key
}

func sharedForeignMirror(key foreignKey) foreignKey {
	key.payloadFootprint = 0
	return key
}

// JSON allocates distinct strings/arrays for EACH occurrence, including old
// parent mirrors. Reconstruct those actual allocation facts before publication;
// accounting fields never appear in the durable format or equality contract.
func (key *foreignKey) UnmarshalJSON(data []byte) error {
	type declaration foreignKey
	var decoded declaration
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*key = ownForeignPayload(foreignKey(decoded))
	key.payloadFootprint += uint(len(key.Tbl1) + len(key.Tbl2))
	for _, columns := range [][]string{key.Cols1, key.Cols2} {
		for _, name := range columns {
			key.payloadFootprint += uint(len(name))
		}
	}
	return nil
}

// Caller owns schema mutation or an unpublished constructor/load. The numeric
// refresh visits record facts only, never ASTs or borrowed declaration trees.
func (t *table) refreshForeignMemoryLocked() {
	bytes := uint(cap(t.Foreign)) * uint(unsafe.Sizeof(foreignKey{}))
	var events uint32
	for _, key := range t.Foreign {
		if key.Tbl2 == t.Name {
			if key.Updatemode != RESTRICT {
				events |= 1 << nativeMutationUpdate
			}
			if key.Deletemode != RESTRICT {
				events |= 1 << nativeMutationDelete
			}
		}
		bytes += key.payloadFootprint
	}
	t.foreignMemoryBytes.Store(uint64(bytes))
	t.cascadingForeignMutationEvents.Store(events)
}

func captureForeignProgramFootprints(programs []foreignKeyTrigger) {
	for i := range programs {
		trigger := &programs[i].trigger
		trigger.initialFKFootprint = &initialFKProgramFootprint{bytes: uint(unsafe.Sizeof(initialFKProgramFootprint{})) +
			uint(len(trigger.Name)) + scm.ComputeSize(trigger.Func) - uint(unsafe.Sizeof(scm.Scmer{}))}
	}
}

// TriggerDescription owns its inline Scmer slots through the backing slice.
// Only the optional generated-program payload is additional. Every list producer
// refreshes this scalar under table.mu; readers never inspect procedure graphs.
func (t *table) refreshTriggerMemoryLocked() {
	bytes := uint(cap(t.Triggers)) * uint(unsafe.Sizeof(TriggerDescription{}))
	row := false
	var statementEvents uint32
	var events uint32
	for _, trigger := range t.Triggers {
		if !trigger.IsSystem && !trigger.Async {
			switch trigger.Timing {
			case BeforeInsert, AfterInsert:
				events |= 1 << nativeMutationInsert
			case BeforeUpdate, AfterUpdate:
				events |= 1 << nativeMutationUpdate
			case BeforeDelete, AfterDelete:
				events |= 1 << nativeMutationDelete
			}
		}
		if trigger.Timing.isStatement() {
			for _, name := range trigger.StatementEvents {
				statementEvents |= 1 << parseNativeMutationEvent(name)
			}
		} else {
			row = true
		}
		if trigger.initialFKFootprint != nil {
			bytes += trigger.initialFKFootprint.bytes
		}
	}
	t.triggerMemoryBytes.Store(uint64(bytes))
	t.statementTriggerEvents.Store(statementEvents)
	t.hasNonStatementTriggers.Store(row)
	t.synchronousRowMutationEvents.Store(events)
}

type publicCatalogFootprint struct{ bytes uint }

// Only the declaration token is required to validate a retained endpoint.
// Never retain its rendered statistics/planner graph through an old snapshot.
// These three old wrapper arrays remain public-owned until version publication;
// their column, UNIQUE and individual FK row payloads stay shared afterwards.
type publicForeignBaseline struct {
	declarations scm.Scmer
	wrapperBytes uint
}

func foreignBaseline(snapshot *tableShowColumnsSnapshot) publicForeignBaseline {
	declarations := snapshot.metadata.declarations
	constraints := snapshot.metadata.constraints.Slice()
	bytes := uint(cap(declarations.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
	if snapshot.metadata.constraints != emptyTableConstraints {
		bytes += uint(cap(constraints)+cap(constraints[3].Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
		if len(constraints) == 6 {
			bytes += uint(cap(constraints[5].Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	return publicForeignBaseline{declarations: declarations, wrapperBytes: bytes}
}

// The first retained endpoint lease exposes this numeric publication once.
// It does not point back to a transaction, resource, callback or mutable graph.
// Program transfer and public trigger publication share schema publication;
// diagnostic memory observers already hold that owner's read lock.
type publicCatalogMemory struct {
	snapshot                atomic.Pointer[publicCatalogFootprint]
	publicationBytes        atomic.Uint64
	transferredProgramBytes atomic.Uint64
	transferredForeignBytes atomic.Uint64
	transferredCheckBytes   atomic.Uint64
	transferredDefaultBytes atomic.Uint64
	baselineWrapperBytes    atomic.Uint64
}

func (memory *publicCatalogMemory) bytes() uint {
	if memory == nil {
		return 0
	}
	if snapshot := memory.snapshot.Load(); snapshot != nil {
		return snapshot.bytes + uint(memory.publicationBytes.Load()) + uint(memory.baselineWrapperBytes.Load()) -
			uint(memory.transferredProgramBytes.Load()) - uint(memory.transferredForeignBytes.Load()) - uint(memory.transferredCheckBytes.Load()) - uint(memory.transferredDefaultBytes.Load())
	}
	return 0
}

// Map bucket capacity is implementation-dependent. Like existing metadata
// accounting this is an estimate, retaining a high-water allocation charge
// across deletes and sparse clones until the actual map is replaced or nilled.
func publicCatalogMapFootprint(highWater int) uint {
	if highWater == 0 {
		return 0
	}
	return 64 + 128*uint(max(8, highWater))
}

// Caller owns changes.mu. All recursive payload sizing happened at the private
// producer; refreshing the published descriptor uses only scalar capacities.
func (changes *publicCatalogChanges) refreshMemoryLocked() {
	// Resource/header ownership is database-native even with no table endpoint.
	// Shared directories are database-owned; endpoint snapshots/programs keep
	// their existing first-table payload owner.
	dbBytes := uint(unsafe.Sizeof(*changes)) + uint(unsafe.Sizeof(*changes.memory))
	dbBytes += uint(unsafe.Sizeof(transactionResource{})) + 128
	dbBytes += changes.historyBytes + changes.sourceHistoryBytes + changes.statementTriggerHistoryBytes
	dbBytes += 3 * publicCatalogMapFootprint(changes.sourceHighWater)
	if changes.sourceBaselines != nil {
		dbBytes += 3 * uint(unsafe.Sizeof(map[string]*ProcedureDescription{}))
	}
	if changes.statementTriggers != nil {
		dbBytes += publicCatalogMapFootprint(changes.statementTriggerHighWater) + uint(unsafe.Sizeof(changes.statementTriggers))
		dbBytes += uint(len(changes.statementTriggers)) * uint(unsafe.Sizeof(catalogStatementTriggerSlot{}))
	}
	if changes.modePin != nil {
		dbBytes += uint(unsafe.Sizeof(*changes.modePin))
	}
	// Shared state directories belong to the database even before the first
	// endpoint exists. Account allocated empty maps as containers, too.
	allocatedMap := func(highWater int, allocated bool) uint {
		if highWater != 0 {
			return publicCatalogMapFootprint(highWater)
		}
		if allocated {
			return 64
		}
		return 0
	}
	directories := allocatedMap(changes.leasesHighWater, changes.leases != nil) + allocatedMap(changes.baselinesHighWater, changes.baselines != nil)
	views := changes.views.Load()
	directories += allocatedMap(changes.viewsHighWater, views != nil && views.tables != nil)
	directories += allocatedMap(changes.defaultWritersHighWater, views != nil && views.defaultWriters != nil)
	if views != nil {
		directories += uint(unsafe.Sizeof(*views)) + uint(unsafe.Sizeof(views.defaultWriters))
	}
	directories += allocatedMap(changes.constraintStatesHighWater, changes.constraintStates != nil)
	directories += uint(unsafe.Sizeof(publicCatalogFootprint{})) + uint(unsafe.Sizeof(map[*table]*tableShowColumnsSnapshot{}))
	changes.db.adjustMetadataMemory(changes.dbMemoryBytes, dbBytes+directories)
	changes.dbMemoryBytes = dbBytes + directories
	bytes := uint(len(changes.leases))*uint(unsafe.Sizeof(tableWriteLease{})) + changes.amendmentBytes
	changes.memory.snapshot.Store(&publicCatalogFootprint{bytes: bytes})
}

// These are the NEW containers emitted by appendForeignMetadata, never a walk
// of its borrowed baseline declarations, columns, UNIQUE rows or prior FK rows.
func foreignMetadataContainers(snapshot *tableShowColumnsSnapshot, rows []scm.Scmer, all, declarations []scm.Scmer) uint {
	bytes := uint(unsafe.Sizeof(*snapshot)) + uint(unsafe.Sizeof(*snapshot.metadata))
	bytes += uint(cap(all)+cap(declarations)+cap(snapshot.metadata.constraints.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
	// Every appended private/publication snapshot produces its own directory.
	// History retains its previous private roots; charge each NEW map through
	// that same producer, without following borrowed row/ID payloads or adding
	// a second ledger. The public table uses this identical numeric estimate.
	if directory := snapshot.metadata.foreignByID; directory != nil {
		bytes += directory.metadataMemory()
	}
	for _, row := range rows {
		fields := row.Slice()
		bytes += uint(cap(fields)+cap(fields[5].Slice())+cap(fields[9].Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
	}
	return bytes
}

func foreignAmendmentFootprint(amendment *catalogForeignAmendment) uint {
	fk := amendment.declaration.Foreign
	bytes := uint(unsafe.Sizeof(*amendment)) + uint(cap(amendment.programs))*uint(unsafe.Sizeof(foreignKeyTrigger{}))
	bytes += fk.payloadFootprint
	for _, program := range amendment.programs {
		bytes += program.trigger.initialFKFootprint.bytes
	}
	return bytes
}
