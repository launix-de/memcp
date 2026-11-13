/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// nativeMutationOperation belongs to one outer mutation producer. It is not a
// transaction registry or a row-reader usage counter. The eventual producer
// factory must establish the binding and all target rights before exposing it.
type nativeMutationOperation struct {
	closeMu           sync.Mutex
	mu                sync.Mutex
	changed           *sync.Cond
	state             *transactionState
	invocation        *TxContext
	closed            bool
	activeCalls       int
	openFrames        int
	producerDone      bool
	localOrigin       bool
	statement         *nativeMutationStatement
	completionFailure any
	undoOwner         *nativeMutationOperation // synchronous child borrows its outer statement undo
	effects           map[*table]*nativeMutationEffect
	effectOrder       []*nativeMutationEffect
	effectsAttempted  bool
	effectScope       *relationScope
	completing        bool
	completionRoot    *nativeMutationOperation // private completion-derived child admission
	abortAllowed      bool
	completionColumns scm.Scmer // immutable producer facts, ONLY on a completion invocation
	replacementTarget *table    // direct completion invocation only; not inherited by nested producers
	replacementEvent  nativeMutationEvent
	transitionEffects bool
	rowTriggerChild   *nativeMutationOperation
	parent            *nativeMutationOperation // immutable; sibling links use parent.mu
	firstChild        *nativeMutationOperation
	nextSibling       *nativeMutationOperation
	previousSibling   *nativeMutationOperation
	linked            bool
	workDrained       bool
}

// newNativeMutationOperationLocked requires state.mu and an UNPUBLISHED binding
// whose state pointer cannot be reset. It deliberately does not snapshot a
// reusable executor: that admission/reset protocol belongs to the caller.
// No production caller is enabled until that protocol and target setup exist.
func newNativeMutationOperationLocked(state *transactionState, binding *TxContext) *nativeMutationOperation {
	if state == nil || binding == nil || binding.transactionState != state || binding.mutationOperation != nil {
		panic("native mutation requires an independent state-bound invocation")
	}
	if state.mutationAdmissionClosed || state.State != TxActive || state.decisionStarted {
		panic("native transaction is not admitting mutation producers")
	}
	operation := &nativeMutationOperation{state: state, invocation: binding}
	operation.changed = sync.NewCond(&operation.mu)
	binding.mutationOperation = operation
	state.mutationOperations++
	return operation
}

// A rowframe owns the borrowed callback bodies, including their row/UNIQUE/batch
// payload. Escaped wrappers retain only this header and an alias index. The owner
// pointer is cleared at retirement; the registry-free header then retains no Tx.
type nativeMutationFrame struct {
	owner  atomic.Pointer[nativeMutationOperation]
	closed bool                 // protected by the loaded owner's mu
	active bool                 // a particular borrowed row/guard cannot be mutated concurrently
	scope  *nativeMutationScope // immutable header pointer, retired with this frame
	bodies []func(*nativeMutationScope, ...scm.Scmer) scm.Scmer
}

func (operation *nativeMutationOperation) newFrame() *nativeMutationFrame {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.closed {
		panic("native mutation producer has finished")
	}
	frame := &nativeMutationFrame{}
	frame.scope = newNativeMutationScopeLocked(operation, nil, frame, false)
	frame.owner.Store(operation)
	operation.openFrames++
	return frame
}

// bind aliases share admission/exclusivity even when their bodies differ
// ($update and a stored-column $set). Neither wrapper captures the body or Tx.
func (frame *nativeMutationFrame) bind(body func(*nativeMutationScope, ...scm.Scmer) scm.Scmer) func(...scm.Scmer) scm.Scmer {
	operation := frame.owner.Load()
	if operation == nil || body == nil {
		panic("native mutation callback has no live body")
	}
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.closed || frame.closed {
		panic("native mutation callback has finished")
	}
	index := len(frame.bodies)
	frame.bodies = append(frame.bodies, body)
	return func(values ...scm.Scmer) scm.Scmer { return frame.call(index, values...) }
}

func (frame *nativeMutationFrame) call(index int, values ...scm.Scmer) scm.Scmer {
	return frame.scope.invoke(func(scope *nativeMutationScope, arguments ...scm.Scmer) scm.Scmer {
		operation := scope.owner.Load()
		operation.mu.Lock()
		// The admitted call keeps this array until its entire child scope ends.
		// A simultaneous frame closer seals first and waits for that same borrow.
		body := frame.bodies[index]
		operation.mu.Unlock()
		return body(scope, arguments...)
	}, values...)
}

// close must run before the producer returns UNIQUE rights, batch counters,
// pooled storage or a borrowed shard lock. It never holds a mutation lock while
// waiting for the admitted call. It is not callable by its own callback body.
func (frame *nativeMutationFrame) close() {
	operation := frame.owner.Load()
	if operation == nil {
		return
	}
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if frame.owner.Load() != operation {
		return
	}
	frame.closed = true
	for frame.active {
		operation.changed.Wait()
	}
	// Wait releases mu. Another closer may have retired the frame while both
	// waited for the same call; only the still-owning closer releases it once.
	if frame.owner.Load() != operation {
		return
	}
	frame.scope.closeLocked(operation)
	clear(frame.bodies)
	frame.bodies = nil
	frame.owner.Store(nil)
	operation.openFrames--
}

// sealAndDrain precedes invocation work/reader cleanup. New writer calls fail;
// existing calls finish without this mutex held. QueryWork does not reopen it.
func (operation *nativeMutationOperation) sealAndDrain() *TxContext {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	operation.closed = true
	for !operation.producerDone || operation.activeCalls != 0 {
		operation.changed.Wait()
	}
	return operation.invocation
}

// The actual producer calls this in its unwind defer BEFORE trying cleanup.
// A parent query closer may already be waiting in close; it must not retire a
// still-running inline INSERT merely because there is no exposed row callback.
func (operation *nativeMutationOperation) finishProducer() {
	operation.mu.Lock()
	operation.producerDone = true
	operation.changed.Broadcast()
	operation.mu.Unlock()
	operation.close()
}

// Parent query cleanup and the producer's own defer may race. Only one owns
// work/reader retirement. Every existing native QueryWork join drains before
// reporting its error; endStatement attempts all joins/releases before panic.
// Rows/guards must have retired their frames before producerDone is published.
func (operation *nativeMutationOperation) close() {
	operation.closeMu.Lock()
	defer operation.closeMu.Unlock()
	operation.closeOwned()
}

// Existing recovery controls may resume only an already sealed, fully drained
// outer undo. TryLock avoids waiting on the currently executing cleanup/trigger
// itself. No new producer/frame admission is reopened by this continuation.
func (operation *nativeMutationOperation) resumeDrained() bool {
	if !operation.closeMu.TryLock() {
		return false
	}
	defer operation.closeMu.Unlock()
	operation.mu.Lock()
	state := operation.state
	if state == nil {
		operation.mu.Unlock()
		return true
	}
	state.mu.Lock()
	ready := operation.closed && operation.producerDone && operation.activeCalls == 0 && operation.openFrames == 0 && operation.workDrained &&
		state.checkedMutation == operation && state.mutationOperations >= 1 &&
		(state.State == TxActive || state.State == TxAborted && operation.statement != nil && operation.statement.abort) && !state.decisionStarted
	state.mu.Unlock()
	ready = ready && operation.childrenDrainedLocked()
	if ready && operation.effectScope != nil {
		operation.effectScope.mu.Lock()
		ready = operation.effectScope.active == 0
		operation.effectScope.mu.Unlock()
	}
	operation.mu.Unlock()
	if !ready {
		return false
	}
	operation.closeOwned()
	return true
}

// closeMu is held through native work drain and the actual undo continuation.
func (operation *nativeMutationOperation) closeOwned() {
	invocation := operation.sealAndDrain()
	if invocation == nil {
		return
	}
	var workFailure any
	attemptWork := func(join func()) {
		defer func() {
			if failure := recover(); failure != nil && workFailure == nil {
				workFailure = failure
			}
		}()
		join()
	}
	attemptWork(operation.closeRowTriggerInvocation)
	// Existing endStatement attempts ALL registered work and reader releases.
	// Retained children remain independently reachable if their join failed.
	attemptWork(invocation.endStatement)
	operation.mu.Lock()
	operation.workDrained = true
	operation.mu.Unlock()
	if workFailure != nil {
		operation.failStatement(workFailure)
		operation.mu.Lock()
		retainedChild := operation.firstChild != nil
		operation.mu.Unlock()
		if retainedChild {
			panic(workFailure)
		}
	}
	operation.joinChildren()
	operation.completeEffects()
	aborting := operation.undoOwner == nil && operation.abortRequested()
	if aborting {
		// No live frame/producer can own this finalizer. Other session controls
		// either report Busy before decision or wait without holding our locks.
		invocation.finalizationMu.Lock()
		defer invocation.finalizationMu.Unlock()
	}
	// A failed undo keeps this SAME operation and all target leases alive for
	// the existing query cleanup retry. Never release the writer exclusion in
	// a defer that would also run after a partially restored statement.
	outcome := operation.finishStatement()
	operation.mu.Lock()
	operation.completionFailure = outcome
	operation.mu.Unlock()
	operation.release()
	if aborting {
		// The row rollback and native producer retirement are complete. Existing
		// resource cleanup now owns failures/retry, with no live callback rights.
		if failure := invocation.finishResourceCommit(); failure != nil {
			panic(failure)
		}
		if !invocation.Session.IsNil() {
			scm.Apply(invocation.Session, scm.NewString("transaction"), scm.NewNil())
		}
	}
	if workFailure != nil && outcome == nil {
		panic(workFailure)
	}
}

// Setup is inside the producer's failure boundary too: every admitted prefix
// must unwind before the frontend's rollback/session-close handler can run.
// This helper exposes no Scheme API until the complete caller port is present.
func (operation *nativeMutationOperation) runProducer(body func(*TxContext) scm.Scmer) scm.Scmer {
	var result scm.Scmer
	var failure any
	func() {
		defer func() { failure = recover() }()
		result = body(operation.invocation)
	}()
	if failure != nil {
		operation.failStatement(failure)
	}
	operation.finishProducer()
	operation.mu.Lock()
	outcome := operation.completionFailure
	operation.mu.Unlock()
	if outcome != nil {
		panic(outcome)
	}
	if failure != nil {
		panic(failure)
	}
	return result
}

// release follows SUCCESSFUL producer/frame/work/reader retirement. It does not
// join query work, reset an executor or close transaction resources. A cleanup
// failure must retain its existing owning continuation and retry before here.
func (operation *nativeMutationOperation) release() {
	operation.mu.Lock()
	if operation.state == nil {
		operation.mu.Unlock()
		return
	}
	if !operation.closed || operation.activeCalls != 0 || operation.openFrames != 0 || operation.firstChild != nil {
		operation.mu.Unlock()
		panic("native mutation producer still owns callback work")
	}
	state := operation.state
	state.mu.Lock()
	if state.mutationOperations <= 0 {
		state.mu.Unlock()
		operation.mu.Unlock()
		panic("native mutation producer lost its transaction owner")
	}
	state.mutationOperations--
	operation.invocation = nil
	operation.state = nil
	operation.undoOwner = nil
	operation.completionRoot = nil
	operation.completionColumns = scm.NewNil()
	operation.replacementTarget = nil
	operation.replacementEvent = 0
	operation.effects = nil
	operation.effectOrder = nil
	state.mu.Unlock()
	operation.mu.Unlock()
	if parent := operation.parent; parent != nil {
		parent.mu.Lock()
		parent.unlinkChildLocked(operation)
		parent.mu.Unlock()
	}
}
