/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "maps"
import "sync"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Transition data is ordinary native table/shard storage. The operation owns
// its construction; it is not a second evaluator, transaction journal or row
// list exposed to Scheme. Only the bounded worker input buffers use row tuples.
// The completion caller must publish AFTER every writer has finished and retire
// AFTER its readers/children have drained. These primitives do not expose an API.
type nativeTransitionRelation struct {
	table      *table
	columns    []string
	positions  []int
	sourceSize int
	writers    []*nativeTransitionWriter
	ready      bool
	published  bool
	access     atomic.Pointer[nativeTransitionReadAccess]
	retired    atomic.Bool
}

// Each actual source shard owns one bounded buffer under its EXISTING write
// lock. Detached full chunks flush after that lock is released; flushMu
// serializes destination appends/rotation without holding source locks. Register once per actual source
// shard under operation.mu, never create a shard for each INSERT SELECT row.
type nativeTransitionWriter struct {
	relation   *nativeTransitionRelation
	shard      *storageShard
	shards     []*storageShard
	compressed int
	flushMu    sync.Mutex
	rows       [][]scm.Scmer
	finished   bool
}

const nativeTransitionBufferRows = 128

// Caller holds the source's actual statement writer lease. Retain only its
// immutable descriptive contract; defaults, generators and cache/computation
// authority must never execute again while copying an already admitted value.
func newNativeTransitionRelation(source *table) *nativeTransitionRelation {
	relation := &nativeTransitionRelation{}
	target := &table{schema: source.schema, Name: ".transition:" + uuid.NewString(), RuntimeIdentity: uuid.NewString(),
		PersistencyMode: Memory, Ephemeral: true, mutationReadOnly: true, ShardMode: ShardModeFree, Collation: source.Collation, Charset: source.Charset}
	relation.table = target
	target.transitionRelation = relation
	declarations := source.columnDeclarations()
	relation.sourceSize = len(declarations)
	for position, current := range declarations {
		if current.IsTemp {
			continue
		}
		if isRuntimeComputedColumn(current) {
			panic("native statement transitions require materialized stored columns")
		}
		target.Columns = append(target.Columns, &column{Name: current.Name, Typ: current.Typ,
			Typdimensions: append([]int(nil), current.Typdimensions...), EnumValues: append([]string(nil), current.EnumValues...),
			AllowNull: current.AllowNull, Collation: current.Collation, DeclarationProperties: maps.Clone(current.DeclarationProperties)})
		relation.columns = append(relation.columns, current.Name)
		relation.positions = append(relation.positions, position)
	}
	target.publishShowColumnsSnapshot(true)
	return relation
}

// Registration runs once per native worker, not for every row. The operation's
// setup lock protects this unpublished topology; the worker alone mutates its
// shard until the same operation's writer drain completes.
func (relation *nativeTransitionRelation) newWriter() *nativeTransitionWriter {
	if relation.ready || relation.published {
		panic("native transition collection has finished")
	}
	writer := &nativeTransitionWriter{relation: relation, shard: NewShard(relation.table)}
	relation.writers = append(relation.writers, writer)
	return writer
}

// capture copies the tuple while its actual source reader/rowframe is alive.
// Scmer values are immutable and carry their own representation references;
// no source recid, ColumnReader, pooled row slice or transaction is retained.
// Flush at the mutation caller's POST-UNLOCK point, never in frame.close.
func (writer *nativeTransitionWriter) capture(row dataset) [][]scm.Scmer {
	if writer.finished || len(row) != writer.relation.sourceSize || len(writer.rows) >= nativeTransitionBufferRows {
		panic("invalid native transition writer input or flush boundary")
	}
	owned := make([]scm.Scmer, len(writer.relation.positions))
	for index, position := range writer.relation.positions {
		owned[index] = row[position]
	}
	writer.rows = append(writer.rows, owned)
	if len(writer.rows) < nativeTransitionBufferRows {
		return nil
	}
	chunk := writer.rows
	writer.rows = nil
	return chunk
}

// The writer owns this unpublished shard exclusively. Use the existing native
// insertion kernel, with an exact declaration snapshot and NO user/default/ID/
// WAL/trigger execution. Its payload then has the ordinary native row/index
// accounting and is read by ordinary scan operators at completion.
func (writer *nativeTransitionWriter) flush(chunk [][]scm.Scmer, guard *uniqueWriteGuard) {
	if len(chunk) == 0 {
		return
	}
	defer clear(chunk)
	pending := func() bool {
		writer.flushMu.Lock()
		defer writer.flushMu.Unlock()
		if writer.shard == nil {
			writer.shard = NewShard(writer.relation.table)
		}
		shard := writer.shard
		func() {
			shard.mu.Lock()
			defer shard.mu.Unlock()
			shard.insertDataset(writer.relation.columns, chunk, nil, writer.relation.table.columnNamesSnapshot.Load(), nil)
		}()
		// Rotate the PRIVATE destination at the usual native shard boundary.
		// The source lock is released, but this caller may still own UNIQUE
		// keys. Compression belongs to the real guard-release owner below.
		if shard.plannerDeltaRows.Load() >= uint64(Settings.ShardSize) {
			writer.sealCurrent()
		}
		return writer.compressed < len(writer.shards)
	}()
	if pending {
		if guard != nil {
			guard.deferTransitionCompression(writer)
		} else {
			writer.compressReady()
		}
	}
}

// Caller owns flushMu, or the actual operation has drained every writer.
func (writer *nativeTransitionWriter) sealCurrent() {
	if writer.shard != nil {
		writer.shards = append(writer.shards, writer.shard)
		writer.shard = nil
	}
}

// Run after the actual source shard AND UNIQUE writer have been released, or
// after the operation has drained every writer. Only private, sealed native
// generations are rebuilt; an active append lane never shares this payload.
// MEMORY retains its existing non-evictable policy. Pending generations may
// span one admitted input batch; this is not a total relation/RAM budget claim.
func (writer *nativeTransitionWriter) compressReady() {
	writer.flushMu.Lock()
	defer writer.flushMu.Unlock()
	for writer.compressed < len(writer.shards) {
		index := writer.compressed
		source := writer.shards[index]
		writer.shards[index] = source.rebuild(false)
		source.flushRebuildSuccessor(writer.shards[index])
		writer.compressed++
	}
}

// Only the operation's post-producer/frame/work drain may take the remaining
// partial chunk without the source shard lock. No source writer then survives.
func (writer *nativeTransitionWriter) finish() {
	if writer.finished {
		return
	}
	writer.flush(writer.rows, nil)
	writer.rows = nil
	writer.sealCurrent()
	writer.compressReady()
	writer.finished = true
}

// This runs after producer/frame/work drain, outside shard/UNIQUE locks. A
// zero-row event still publishes one normal empty shard. Relations have their
// own RuntimeIdentity and no durable schema entry (Ephemeral is already the
// existing schema serializer's exclusion contract).
func (relation *nativeTransitionRelation) prepare() {
	if relation.ready {
		return
	}
	for _, writer := range relation.writers {
		writer.finish()
		relation.table.Shards = append(relation.table.Shards, writer.shards...)
	}
	relation.writers = nil
	if len(relation.table.Shards) == 0 {
		relation.table.Shards = []*storageShard{NewShard(relation.table)}
	}
	relation.table.publishTopologyLocked()
	relation.table.onInitComplete.Store(true)
	relation.ready = true
}

// Publish only into the existing native catalog and relationScope. This does
// not run public CREATE callbacks or perform a schema save: the relation is
// native-owned transient result data, not a frontend CREATE declaration.
func (relation *nativeTransitionRelation) publish(scope *relationScope, root *nativeMutationOperation) {
	if !relation.ready || relation.published || scope == nil {
		panic("native transition publication requires completed collection and its scope")
	}
	finish := scope.beginUse()
	defer finish()
	target := relation.table
	db := target.schema
	access := db.GetWrite(nil)
	defer access.Release()
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	if GetDatabase(db.Name) != db || db.tables.Get(target.Name) != nil {
		panic("native transition database or identity changed before publication")
	}
	if previous := db.tables.Set(target.Name, target); previous != nil {
		panic("native transition identity collision")
	}
	scope.adopt(target)
	relation.access.Store(&nativeTransitionReadAccess{scope: scope, root: root})
	relation.published = true
}

// The descriptor is immutable and exists only while a completion may consume
// these results. Clone/serialization cannot manufacture a native invocation.
type nativeTransitionReadAccess struct {
	scope *relationScope
	root  *nativeMutationOperation
}

// ReadAccess owns this borrow until the actual shard reader returns. It closes
// the gap between an admission check and scope retirement without any counter
// or registry in a read row loop, and covers an escaped concurrent scan too.
func (relation *nativeTransitionRelation) borrowRead(tx *TxContext) *relationScope {
	// A private-data producer reads public sources through a short reader
	// facade. Only its borrowed actual source invocation can attest a native
	// transition relation; equal state pointers never manufacture that right.
	if tx != nil && tx.relationData != nil {
		coordinates, finish := tx.relationData.borrow()
		defer finish()
		tx = coordinates.public
	}
	access := relation.access.Load()
	if access == nil || tx == nil || tx.mutationOperation == nil {
		panic("native transition relation is outside its completion scope")
	}
	operation := tx.mutationOperation
	operation.mu.Lock()
	valid := operation.invocation == tx && operation.state == tx.transactionState && !operation.closed &&
		!operation.producerDone && operation.completionRoot == access.root
	operation.mu.Unlock()
	if !valid {
		panic("native transition reader has no live completion invocation")
	}
	// The scope re-attests closing under the SAME mutex used by close. A stale
	// descriptor cannot race past retirement merely because the pointer matched.
	access.scope.mu.Lock()
	defer access.scope.mu.Unlock()
	if access.scope.closing {
		panic("native transition relation is retiring")
	}
	access.scope.active++
	return access.scope
}

// Called after the scope has drained every native reader (or, for unpublished
// data, after the operation has drained every writer). Held table/frame headers
// may survive; they retain descriptions, never all retired transition rows.
func (relation *nativeTransitionRelation) clearPayload() {
	relation.access.Store(nil)
	clearShard := func(shard *storageShard) {
		var indexes []*StorageIndex
		func() {
			shard.mu.Lock()
			defer shard.mu.Unlock()
			shard.publishDeltaRows(nil)
			clear(shard.columns)
			indexes, shard.Indexes = shard.Indexes, nil
			// Published main cardinality remains an immutable generation fact.
			// Retirement revokes access; clearing payload does not republish it.
			shard.plannerDeltaRows.Store(0)
		}()
		for _, index := range indexes {
			GlobalCache.Remove(index)
		}
	}
	for _, shard := range relation.table.Shards {
		clearShard(shard)
	}
	relation.retired.Store(true)
	for _, writer := range relation.writers {
		if writer.shard != nil {
			clearShard(writer.shard)
		}
		for _, shard := range writer.shards {
			clearShard(shard)
		}
		for _, row := range writer.rows {
			clear(row)
		}
		writer.rows = nil
		writer.finished = true
	}
	relation.writers = nil
}
