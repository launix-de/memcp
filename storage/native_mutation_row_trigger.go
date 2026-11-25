/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"

// Select all synchronous callback bindings at the existing event boundary.
// Acquire/Release are the existing target pins; moving their release to the
// event unwind also releases a later callback's pin if an earlier one fails.
// GetTriggers already returns event-owned headers; declaration inputs instead
// supply immutable shared headers and need a bounded event-owned copy. Neither
// case adds a registry or per-callback warm locking. Cold construction is unlocked.
func nativeRowTriggerInvocations(tx *TxContext, triggers []TriggerDescription, immutable bool) ([]TriggerDescription, func()) {
	if len(triggers) == 0 {
		return triggers, func() {}
	}
	if immutable {
		triggers = append([]TriggerDescription(nil), triggers...)
	}
	release := func() {
		for i := range triggers {
			trigger := &triggers[i]
			if !trigger.Func.IsNil() && !trigger.Async {
				trigger.releaseTarget()
			}
			trigger.invocation = nil
		}
	}
	acquired := 0
	accepted := false
	defer func() {
		if !accepted {
			// Only the completed prefix owns pins if Acquire itself panicked.
			for i := range triggers[:acquired] {
				trigger := &triggers[i]
				if !trigger.Func.IsNil() && !trigger.Async {
					trigger.releaseTarget()
				}
			}
		}
	}()
	for i := range triggers {
		trigger := &triggers[i]
		if !trigger.Func.IsNil() && !trigger.Async && !trigger.acquireTarget(tx) {
			trigger.Func = scm.NewNil()
		}
		trigger.invocation = tx
		acquired++
	}
	if tx == nil || tx.mutationOperation == nil {
		accepted = true
		return triggers, release
	}
	operation := tx.mutationOperation
	operation.mu.Lock()
	if operation.closed || operation.producerDone || operation.invocation != tx {
		operation.mu.Unlock()
		panic("native row trigger invocation has finished")
	}
	missing := false
	for i := range triggers {
		trigger := &triggers[i]
		target := trigger.cacheMutationTarget()
		if trigger.Func.IsNil() || trigger.Async ||
			!operation.transitionEffects && target == nil && operation.cacheInput == nil {
			continue
		}
		if child := operation.rowTriggerInputs[target]; child != nil {
			trigger.invocation = child.invocation
		} else {
			trigger.invocation = nil
			missing = true
		}
	}
	operation.mu.Unlock()
	if missing {
		for i := range triggers {
			trigger := &triggers[i]
			if !trigger.Func.IsNil() && !trigger.Async && trigger.invocation == nil {
				trigger.invocation = nativeRowTriggerInvocation(tx, trigger.cacheMutationTarget())
			}
		}
	}
	accepted = true
	return triggers, release
}

// Row callbacks are separate native invocations, never the base statement's
// row-event producer. Otherwise a synchronous INSERT from a row callback into
// the same target could be misattributed to the original INSERT's transitions.
// One unattributed child per actual input belongs to the entire operation (not
// each source row). Ordinary callbacks cannot inherit a sibling's cache input.
// Real nested SQL DML opens its own explicitly attributed producer beneath it.
func nativeRowTriggerInvocation(tx *TxContext, target *table) *TxContext {
	if tx == nil || tx.mutationOperation == nil {
		return tx
	}
	operation := tx.mutationOperation
	operation.mu.Lock()
	if !operation.transitionEffects && target == nil && operation.cacheInput == nil {
		operation.mu.Unlock()
		return tx
	}
	if operation.closed || operation.producerDone || operation.invocation != tx {
		operation.mu.Unlock()
		panic("native row trigger invocation has finished")
	}
	if child := operation.rowTriggerInputs[target]; child != nil {
		binding := child.invocation // creator/closer owns this entire producer
		operation.mu.Unlock()
		return binding
	}
	operation.mu.Unlock()
	var input *nativeCacheMutationTarget
	if target != nil {
		input = target.retainCacheMutationTarget(tx)
	}
	child, err := newNestedNativeMutationBinding(tx, tx.Session)
	if err != nil {
		panic(err)
	}
	// Constructor-only assignment. The child has not escaped to a callback;
	// cleanup cannot release it before its producer is finished below.
	child.cacheInput = input
	operation.mu.Lock()
	winner := operation.rowTriggerInputs[target]
	if winner == nil {
		if operation.rowTriggerInputs == nil {
			operation.rowTriggerInputs = make(map[*table]*nativeMutationOperation)
		}
		operation.rowTriggerInputs[target], winner = child, child
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
	for {
		operation.mu.Lock()
		var target *table
		var child *nativeMutationOperation
		for target, child = range operation.rowTriggerInputs {
			break
		}
		operation.mu.Unlock()
		if child == nil {
			return
		}
		child.finishProducer()
		operation.mu.Lock()
		delete(operation.rowTriggerInputs, target)
		operation.mu.Unlock()
	}
}
