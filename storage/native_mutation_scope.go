/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// A compact callback capability borrows an existing immutable invocation. It
// does not allocate/reset a TxContext per source row and does not change Session.
// The parent/child relation is explicit native call ownership, not caller Env,
// goroutine identity or a mutable currentFrame slot on the operation/transaction.
type nativeMutationScope struct {
	owner    atomic.Pointer[nativeMutationOperation]
	context  atomic.Pointer[TxContext]
	parent   *nativeMutationScope // owner's mu; cleared at retirement
	frame    *nativeMutationFrame // optional alias group, owner's mu
	admitted bool                 // continuation of an already admitted native call
	active   bool                 // actual writer call, never a reader usage counter
	closed   bool                 // owner's mu
}

// Caller owns operation.mu. All scopes use the producer's SAME actual state.
func newNativeMutationScopeLocked(operation *nativeMutationOperation, parent *nativeMutationScope, frame *nativeMutationFrame, admitted bool) *nativeMutationScope {
	scope := &nativeMutationScope{parent: parent, frame: frame, admitted: admitted}
	scope.context.Store(operation.invocation)
	scope.owner.Store(operation)
	return scope
}

// A getter can resolve immutable execution inputs without synchronization
// writes. This alone is NOT writer admission: every mutation must call invoke.
func (scope *nativeMutationScope) readContext() *TxContext {
	context := scope.context.Load()
	if context == nil {
		panic("native mutation execution scope has finished")
	}
	return context
}

func (scope *nativeMutationScope) String() string { return "native mutation execution scope" }

func (scope *nativeMutationScope) invoke(body func(*nativeMutationScope, ...scm.Scmer) scm.Scmer, values ...scm.Scmer) scm.Scmer {
	operation := scope.owner.Load()
	if operation == nil {
		panic("native mutation execution scope has finished")
	}
	operation.mu.Lock()
	if scope.closed || (operation.closed && !scope.admitted) || (scope.frame != nil && scope.frame.closed) {
		operation.mu.Unlock()
		panic("native mutation execution scope has finished")
	}
	if scope.active || (scope.frame != nil && scope.frame.active) {
		operation.mu.Unlock()
		panic("native mutation callback is already active")
	}
	scope.active = true
	if scope.frame != nil {
		scope.frame.active = true
	}
	operation.activeCalls++
	// A sealed producer may drain an admitted call without preventing that call
	// from completing its synchronous native/trigger continuations. This child
	// expires before the parent's borrow ends, so it cannot reopen the producer.
	child := newNativeMutationScopeLocked(operation, scope, nil, true)
	operation.mu.Unlock()
	defer func() {
		defer func() {
			operation.mu.Lock()
			scope.active = false
			if scope.frame != nil {
				scope.frame.active = false
			}
			operation.activeCalls--
			operation.changed.Broadcast()
			operation.mu.Unlock()
		}()
		child.close()
	}()
	return body(child, values...)
}

func (scope *nativeMutationScope) close() {
	operation := scope.owner.Load()
	if operation == nil {
		return
	}
	operation.mu.Lock()
	defer operation.mu.Unlock()
	scope.closeLocked(operation)
}

// The native creator closes a callback scope after its Scheme body returns,
// before resuming borrowed row/UNIQUE work. It must not close its own active call.
func (scope *nativeMutationScope) closeLocked(operation *nativeMutationOperation) {
	if scope.owner.Load() != operation {
		return
	}
	scope.closed = true
	for scope.active {
		operation.changed.Wait()
	}
	if scope.owner.Load() != operation {
		return
	}
	scope.context.Store(nil)
	scope.parent = nil
	scope.frame = nil
	scope.owner.Store(nil)
}
