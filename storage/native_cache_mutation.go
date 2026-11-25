/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// Only an actual pinned trigger dispatch installs this input on a native child.
// The original transaction resource retains the physical cache generation until
// its SAME decision, including work drain and statement/savepoint restoration.
// A public SQL target, cache name or matching Session supplies no such input.
type nativeCacheMutationTarget struct {
	target *table
	owner  *transactionState
	bytes  uint
}

// Existing computed-column dependencies also pin public tables. Their value
// invalidation retains the ordinary callback profile; a lifetime pin alone
// must never grant cross-database row access to a public endpoint.
func (trigger TriggerDescription) cacheMutationTarget() *table {
	if target := trigger.cacheTarget; target != nil && target.isEphemeralQueryTable() {
		return target
	}
	return nil
}

// The existing resource owner calls a successful closer once. No callback or
// child releases the transaction's retained pin independently.
func (input *nativeCacheMutationTarget) Close() error {
	input.target.releaseCacheUse()
	input.target.schema.adjustMetadataMemory(input.bytes, 0)
	return nil
}

func (t *table) retainCacheMutationTarget(tx *TxContext) *nativeCacheMutationTarget {
	operation := tx.mutationOperation
	if operation == nil || !operation.localOrigin || tx.remoteObject != nil || tx.remoteInvocation || tx.coordinator != nil {
		panic("cache maintenance requires its original local native producer")
	}
	key := fmt.Sprintf("native-cache-mutation/%p", t)
	value := tx.getOrComputeResource(key, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		// This mode lease is also owned by the original resource scope. Shared
		// and replication authority are rejected before any cache row effect.
		tx.admitStandaloneMutation(t.schema)
		if !t.isEphemeralQueryTable() || !t.acquireCacheUse() {
			panic("cache maintenance target has retired")
		}
		if GetDatabase(t.schema.Name) != t.schema || t.schema.tables.Get(t.Name) != t {
			t.releaseCacheUse()
			panic("cache maintenance requires its current physical target")
		}
		bytes := uint(unsafe.Sizeof(nativeCacheMutationTarget{})) + uint(unsafe.Sizeof(transactionResource{})) + uint(len(key)) + 128
		t.schema.adjustMetadataMemory(0, bytes)
		return scm.NewAny(&nativeCacheMutationTarget{t, tx.transactionState, bytes})
	}))
	input, ok := value.Any().(*nativeCacheMutationTarget)
	if !ok || input.target != t || input.owner != tx.transactionState {
		panic("cache maintenance input belongs to another native owner")
	}
	return input
}

// Registration publishes this complete runtime target with the callback, under
// the existing source header owner. insertTriggerLocked records the reverse
// edge before exposing the callback; no create-then-rebind window is permitted.
func (trigger *TriggerDescription) bindCacheTarget(target *table) func() {
	if !trigger.Hidden || trigger.Source != "" || trigger.Language != "" || trigger.Async ||
		trigger.Timing == BeforeInsert || trigger.Timing == BeforeUpdate || trigger.Timing == BeforeDelete ||
		!target.isEphemeralQueryTable() {
		panic("cache trigger registration requires a compiled hidden AFTER program and an actual cache target")
	}
	if !target.acquireCacheUse() {
		panic("cache trigger target has retired")
	}
	if GetDatabase(target.schema.Name) != target.schema || target.schema.tables.Get(target.Name) != target {
		target.releaseCacheUse()
		panic("cache trigger requires its current physical target")
	}
	trigger.cacheTarget = target
	trigger.Acquire, trigger.Release = target.acquireCacheUseForTrigger, target.releaseCacheUse
	return target.releaseCacheUse
}
