/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

// The existing trigger declaration/compiler owner supplies an immutable native
// callable. SQL recipes, names, events and error policy remain with that owner;
// the mutation runtime only knows the native completion ABI4.
type nativeStatementTrigger struct {
	name        string
	replacement bool
	program     scm.Scmer // (OLD relation, NEW relation, invocation Session, invocation Tx)
}

type nativeMutationEffect struct {
	target      *table
	event       nativeMutationEvent
	columns     scm.Scmer
	programs    []nativeStatementTrigger
	replacement bool
	old, new    *nativeTransitionRelation
	writers     map[*storageShard]*nativeMutationEventWriter
}

// A source shard owns the tuple staging buffers under its existing writer lock.
// Maintenance forwarding and physical replacements receive no such writer, so
// their internal copies cannot become duplicate logical statement effects.
type nativeMutationEventWriter struct {
	event        nativeMutationEvent
	replacement  bool
	replacedRows NonLockingReadMap.NonBlockingBitMap // source recids, owned under this source shard's writer lock
	old, new     *nativeTransitionWriter
}

type nativeMutationEventChunk struct {
	old, new [][]scm.Scmer
}

// The outer target admission is held and no producer has been exposed. Take
// the program snapshot outside schema/shard/UNIQUE locks, then install exactly
// one event, including the zero-row case. Caller owns the operation setup.
func (operation *nativeMutationOperation) declareEffect(t *table, descriptor nativeMutationDescriptor, programs []nativeStatementTrigger) {
	values := make([]scm.Scmer, len(descriptor.columns))
	for index, column := range descriptor.columns {
		values[index] = scm.NewString(column)
	}
	effect := &nativeMutationEffect{target: t, event: descriptor.event, columns: scm.NewSlice(values), programs: programs}
	if len(programs) != 0 {
		effect.replacement = programs[0].replacement
		effect.old, effect.new = newNativeTransitionRelation(t), newNativeTransitionRelation(t)
	}
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.closed || operation.producerDone {
		panic("native mutation producer has finished")
	}
	if operation.effects == nil {
		operation.effects = make(map[*table]*nativeMutationEffect)
	}
	if operation.effects[t] != nil {
		panic("native mutation target was declared more than once")
	}
	operation.effects[t] = effect
	operation.transitionEffects = operation.transitionEffects || len(programs) != 0
	operation.effectOrder = append(operation.effectOrder, effect)
}

// Bind once at the actual mapper/physical INSERT setup, before acquiring the
// source shard lock. A map entry represents a real native shard, not a row or a
// transaction/session scratch value. All row capture then uses this direct
// writer pointer under the source's existing writer protocol.
func (operation *nativeMutationOperation) eventWriter(source *storageShard) *nativeMutationEventWriter {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.closed || operation.producerDone {
		panic("native mutation producer has finished")
	}
	effect := operation.effects[source.t]
	if effect == nil || len(effect.programs) == 0 {
		return nil
	}
	if writer := effect.writers[source]; writer != nil {
		return writer
	}
	writer := &nativeMutationEventWriter{event: effect.event, replacement: effect.replacement}
	if effect.event != nativeMutationInsert {
		writer.old = effect.old.newWriter()
	}
	if effect.event != nativeMutationDelete {
		writer.new = effect.new.newWriter()
	}
	if effect.writers == nil {
		effect.writers = make(map[*storageShard]*nativeMutationEventWriter)
	}
	effect.writers[source] = writer
	return writer
}

// Replacement candidates form the target row set, even if a joined producer
// visits one source recid repeatedly. Unlike a physical DELETE, capture cannot
// use the table's deletion mask for this ownership. The invocation owns this
// compact source bitmap until all candidate work and replacement children drain.
// Caller holds the source's existing write lock; no new row lock is introduced.
func (writer *nativeMutationEventWriter) captureReplacementDelete(recid uint32, old dataset) (nativeMutationEventChunk, bool) {
	if writer.replacedRows.Get(uint(recid)) {
		return nativeMutationEventChunk{}, false
	}
	writer.replacedRows.Set(uint(recid), true)
	return writer.capture(nativeMutationDelete, old, nil), true
}

// Caller holds the actual source shard write lock. Exact final OLD/NEW tuples
// are copied before its borrowed readers/recids can retire. Equal-value UPDATE
// is an application too; it must call here before its physical no-op skip.
func (writer *nativeMutationEventWriter) capture(event nativeMutationEvent, old, new dataset) nativeMutationEventChunk {
	if writer == nil {
		return nativeMutationEventChunk{}
	}
	if event != writer.event {
		panic("native mutation callback does not match its declared operation")
	}
	var chunk nativeMutationEventChunk
	if writer.old != nil {
		chunk.old = writer.old.capture(old)
	}
	if writer.new != nil {
		chunk.new = writer.new.capture(new)
	}
	return chunk
}

// This is the POST-UNLOCK caller seam, not nativeMutationFrame.close. Flushing
// a native chunk never borrows a row/UNIQUE right or leaves a source recid for
// the completion phase to reinterpret after deletion/rebuild.
func (writer *nativeMutationEventWriter) flush(chunk nativeMutationEventChunk, guard *uniqueWriteGuard) {
	if writer == nil {
		return
	}
	if writer.old != nil {
		writer.old.flush(chunk.old, guard)
	}
	if writer.new != nil {
		writer.new.flush(chunk.new, guard)
	}
}

// This check belongs to the actual native writer entry, before defaults,
// identities, user callbacks or row effects. A held/raw callback cannot acquire
// fresh standalone rights or silently omit a declared statement completion.
func (t *table) requireStatementTriggerMutation(tx *TxContext, event nativeMutationEvent) {
	if t.statementTriggerEvents.Load()&(1<<event) == 0 {
		return
	}
	if tx == nil || (tx.Mode != TxCursorStability && tx.Mode != TxACID) || tx.mutationOperation == nil {
		panic("statement-triggered mutation requires an outer native statement producer")
	}
	requireCheckInvocation(tx)
	tx.requireMutationBinding()
	operation := tx.mutationOperation
	operation.mu.Lock()
	effect := operation.effects[t]
	valid := effect != nil && effect.event == event
	operation.mu.Unlock()
	if !valid || !tx.hasStatementUndo() {
		panic("statement-triggered mutation has no matching outer native producer")
	}
}
