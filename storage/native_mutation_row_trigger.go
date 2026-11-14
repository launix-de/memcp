/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// Row callbacks are separate native invocations, never the base statement's
// row-event producer. Otherwise a synchronous INSERT from a row callback into
// the same target could be misattributed to the original INSERT's transitions.
// One unattributed child belongs to the entire operation (not each source row).
// Real nested SQL DML opens its own explicitly attributed producer beneath it.
func nativeRowTriggerInvocation(tx *TxContext) *TxContext {
	if tx == nil || tx.mutationOperation == nil {
		return tx
	}
	operation := tx.mutationOperation
	operation.mu.Lock()
	if !operation.transitionEffects {
		operation.mu.Unlock()
		return tx
	}
	if operation.closed || operation.producerDone || operation.invocation != tx {
		operation.mu.Unlock()
		panic("native row trigger invocation has finished")
	}
	if child := operation.rowTriggerChild; child != nil {
		binding := child.invocation // creator/closer owns this entire producer
		operation.mu.Unlock()
		return binding
	}
	operation.mu.Unlock()
	child, err := newNestedNativeMutationBinding(tx, tx.Session)
	if err != nil {
		panic(err)
	}
	operation.mu.Lock()
	winner := operation.rowTriggerChild
	if winner == nil {
		operation.rowTriggerChild, winner = child, child
	}
	operation.mu.Unlock()
	if winner != child {
		// Concurrent native row callbacks may race on the first construction.
		// The loser owns no effects; retire its registered work/count normally.
		child.finishProducer()
	}
	return winner.invocation
}

// Parent close calls this after its actual producer/callback drain and BEFORE
// endStatement joins the parent's registered child cleanup. The child may have
// real escaped native calls, which are sealed and drained by its own owner.
func (operation *nativeMutationOperation) closeRowTriggerInvocation() {
	operation.mu.Lock()
	child := operation.rowTriggerChild
	operation.mu.Unlock()
	if child != nil {
		child.finishProducer()
		operation.mu.Lock()
		operation.rowTriggerChild = nil
		operation.mu.Unlock()
	}
}
