/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

func releaseNoRelationData() {}

type relationDataPolicy uint8

const (
	relationCallerData relationDataPolicy = iota
	relationIndependentData
)

// One execution view keeps public source authority separate from the native
// private data invocation. The borrow is per operator input, never per row.
// Close drains admitted operators and clears every execution coordinate; an
// escaped facade retains only this sealed header, not a session or transaction.
type relationDataView struct {
	mu           sync.Mutex
	cond         *sync.Cond
	coordinates  *relationDataCoordinates
	active       int
	closing      bool
	insertInputs atomic.Pointer[relationDataInsertInputs]
}

// These targets belong to one admitted native producer. A streaming INSERT
// resolves its sink once from this immutable input binding; borrowMutation on
// that exact native invocation still owns every writer call and rejects escape.
type relationDataInsertInputs struct {
	targets map[*table]*TxContext
}

func (view *relationDataView) bindInsertInputs(targets []*table, data *TxContext) {
	if data.mutationOperation == nil {
		return
	}
	inputs := &relationDataInsertInputs{targets: make(map[*table]*TxContext, len(targets))}
	for _, target := range targets {
		inputs.targets[target] = data
	}
	view.insertInputs.Store(inputs)
}

type relationDataCoordinates struct {
	operation *relationDataOperation
	owner     *relationScope
	public    *TxContext
	read      *TxContext // same public state, short invocation-owned reader/work lists
	data      *TxContext
	writable  bool
}

func newRelationDataView(owner *relationScope, public, data *TxContext, writable bool) *relationDataView {
	view := &relationDataView{coordinates: &relationDataCoordinates{owner: owner, public: public, data: data, writable: writable}}
	view.cond = sync.NewCond(&view.mu)
	return view
}

func (view *relationDataView) borrow() (*relationDataCoordinates, func()) {
	view.mu.Lock()
	if view.closing || view.coordinates == nil {
		view.mu.Unlock()
		panic("private relation execution view has finished")
	}
	view.active++
	coordinates := view.coordinates
	view.mu.Unlock()
	var once sync.Once
	return coordinates, func() {
		once.Do(func() {
			view.mu.Lock()
			view.active--
			view.cond.Broadcast()
			view.mu.Unlock()
		})
	}
}

func (view *relationDataView) close() {
	view.mu.Lock()
	defer view.mu.Unlock()
	view.closing = true
	view.insertInputs.Store(nil)
	for view.active != 0 {
		view.cond.Wait()
	}
	view.coordinates = nil
}

// This counter borrows the actual public transaction while a separate native
// data operation consumes its sources. It is not another mutation or catalog
// construction counter. Public decisions cannot replace its source authority.
func borrowRelationDataOrigin(tx *TxContext) func() {
	if tx == nil || tx.transactionState == nil || !tx.queryActive.Load() {
		panic("private relation data requires an active query origin")
	}
	tx.requireMutationBinding()
	tx.bindingMu.Lock()
	defer tx.bindingMu.Unlock()
	if tx.bindingAttempt != nil {
		panic(newNativeMutationBusyError(tx.transactionState, "private data origin"))
	}
	state := tx.transactionState
	resources := state.resourceScope()
	resources.mu.Lock()
	defer resources.mu.Unlock()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.State != TxActive || state.decisionStarted || state.mutationAdmissionClosed || resources.deciding || resources.closed {
		panic(newNativeMutationBusyError(state, "private data origin"))
	}
	state.relationDataOrigins++
	var once sync.Once
	return func() { once.Do(func() { state.mu.Lock(); state.relationDataOrigins--; state.mu.Unlock() }) }
}

func copyRelationExecution(public *TxContext, state *transactionState) *TxContext {
	context := &TxContext{transactionState: state, Session: public.Session, SessionState: public.SessionState,
		invocationContext: public.invocationContext}
	if state == public.transactionState {
		context.remoteObject, context.remoteInvocation = public.remoteObject, public.remoteInvocation
	}
	context.querySeq.Store(public.querySeq.Load())
	context.queryInfo.Store(public.queryInfo.Load())
	context.statementGeneration.Store(statementGenerationCounter.Add(1))
	context.queryActive.Store(true)
	context.fanoutOwner = public
	if public.fanoutOwner != nil {
		context.fanoutOwner = public.fanoutOwner
	}
	return context
}

func clearRelationExecution(context *TxContext) {
	context.Session, context.SessionState, context.transactionState = scm.NewNil(), nil, nil
	context.invocationContext, context.fanoutOwner, context.remoteObject = nil, nil, nil
	context.remoteInvocation = false
	context.queryInfo.Store(nil)
	context.querySeq.Store(0)
	context.statementGeneration.Store(0)
	context.queryActive.Store(false)
	context.remoteBindings.Store(nil)
}

// Scope-owned pending decisions are bounded by live inputs and failed cleanup
// continuations. Successful operations leave no state in the lexical owner.
type relationDataOperation struct {
	mu            sync.Mutex
	viewsMu       sync.Mutex
	owner         *relationScope
	context       *TxContext
	views         []*relationDataView
	facades       []*TxContext
	dataBindings  []*TxContext
	commit        bool
	drained       bool
	retired       bool
	releaseOrigin func()
	releaseWriter func()
}

func (scope *relationScope) newDataOperation(public *TxContext, commit bool, releaseOrigin, releaseWriter func()) *relationDataOperation {
	if scope.dataPolicy != relationIndependentData || public.Session.IsNil() || relationScopeOwnerFor(public.Session) != scope.owner {
		panic("private relation data belongs to another scope owner")
	}
	context := NewTxContext(TxCursorStability)
	context = copyRelationExecution(public, context.transactionState)
	context.autoCommit = true
	operation := &relationDataOperation{owner: scope, context: context, commit: commit, releaseOrigin: releaseOrigin, releaseWriter: releaseWriter}
	view := newRelationDataView(scope, public, context, commit)
	view.coordinates.operation = operation
	context.relationData = view
	read := copyRelationExecution(public, public.transactionState)
	read.relationData = view
	view.coordinates.read = read
	operation.views = append(operation.views, view)
	operation.facades = append(operation.facades, read)
	operation.dataBindings = append(operation.dataBindings, context)
	scope.mu.Lock()
	if scope.dataOperations == nil {
		scope.dataOperations = make(map[*relationDataOperation]struct{})
	}
	scope.dataOperations[operation] = struct{}{}
	scope.mu.Unlock()
	return operation
}

func (operation *relationDataOperation) retire() {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.retired {
		return
	}
	context := operation.context
	if !operation.drained {
		operation.viewsMu.Lock()
		facades := append([]*TxContext(nil), operation.facades...)
		operation.viewsMu.Unlock()
		for _, facade := range facades {
			facade.endStatement()
		}
		context.endStatement()
	}
	operation.viewsMu.Lock()
	defer operation.viewsMu.Unlock()
	if !operation.drained {
		for _, view := range operation.views {
			view.close()
		}
		for _, facade := range operation.facades {
			clearRelationExecution(facade)
		}
		operation.drained = true
	}
	// The lexical owner, rather than the connection's transaction marker,
	// retains a failed private decision. Expression execution has finished;
	// this controller must never publish a caller recovery marker on retry.
	context.Session = scm.NewNil()
	context.relationData = nil // only this private decision owner finalizes its state
	if operation.commit {
		if err := context.Commit(); err != nil {
			panic(err)
		}
	} else {
		context.Rollback()
	}
	context.relationData = operation.views[0] // sealed capability header, including after retirement
	for _, binding := range operation.dataBindings {
		clearRelationExecution(binding)
	}
	operation.owner.mu.Lock()
	delete(operation.owner.dataOperations, operation)
	operation.owner.mu.Unlock()
	operation.releaseOrigin()
	if operation.releaseWriter != nil {
		operation.releaseWriter()
	}
	operation.releaseOrigin, operation.releaseWriter = nil, nil
	operation.context, operation.views, operation.facades, operation.dataBindings, operation.retired = nil, nil, nil, nil, true
}

func (scope *relationScope) retireDataOperations() {
	scope.mu.Lock()
	operations := make([]*relationDataOperation, 0, len(scope.dataOperations))
	for operation := range scope.dataOperations {
		operations = append(operations, operation)
	}
	scope.mu.Unlock()
	var first any
	for _, operation := range operations {
		func() {
			defer func() {
				if failure := recover(); failure != nil && first == nil {
					first = failure
				}
			}()
			operation.retire()
		}()
	}
	if first != nil {
		panic(first)
	}
}

// A read input gets a separate CS view only for an explicitly independent
// private relation. Nested native inputs reuse it; ordinary source callbacks
// keep their original lexical caller context and public transaction snapshot.
func (t *table) readDataContext(tx *TxContext) (*TxContext, func()) {
	if t.dataOwner == nil && (tx == nil || tx.relationData == nil) {
		return tx, releaseNoRelationData
	}
	finishView := func() {}
	ready := false
	defer func() {
		if !ready {
			finishView()
		}
	}()
	if tx != nil && tx.relationData != nil {
		coordinates, finish := tx.relationData.borrow()
		finishView = finish
		if t.dataOwner == coordinates.owner {
			data := coordinates.data
			if tx.transactionState == data.transactionState {
				data = tx
			}
			data.requireMutationBinding()
			ready = true
			return data, finish
		}
		if t.dataOwner == nil {
			read := coordinates.read
			if tx.transactionState == coordinates.public.transactionState {
				read = tx
			}
			ready = true
			return read, finish
		}
		tx = coordinates.public
	}
	if t.dataOwner == nil {
		ready = true
		return tx, finishView
	}
	finishScope := t.dataOwner.beginUse()
	if !t.dataOwner.owns(t) {
		finishScope()
		panic("private data relation is outside its actual scope")
	}
	var finishOrigin func()
	var operation *relationDataOperation
	transferred := false
	defer func() {
		if !transferred {
			if finishOrigin != nil {
				finishOrigin()
			}
			finishScope()
		}
	}()
	finishOrigin = borrowRelationDataOrigin(tx)
	operation = t.dataOwner.newDataOperation(tx, false, finishOrigin, nil)
	transferred = true
	ready = true
	var finishLease sync.Once
	return operation.context, func() {
		defer finishLease.Do(func() { finishView(); finishScope() })
		operation.retire()
	}
}

func (t *table) writeDataContext(tx *TxContext) (*TxContext, func()) {
	if tx == nil || tx.relationData == nil {
		if t.dataOwner != nil {
			panic("independent private writes require their native data producer")
		}
		return tx, func() {}
	}
	coordinates, finish := tx.relationData.borrow()
	ready := false
	defer func() {
		if !ready {
			finish()
		}
	}()
	// Existing execution scratch remains owned by the caller. This is an
	// actual native lifetime/engine classification, never a user SQL alias.
	// An explicit DML producer still rejects mixed logical write owners.
	if t.callerExecutionRelation() {
		coordinates.public.requireMutationBinding()
		ready = true
		return coordinates.public, finish
	}
	if t.dataOwner != coordinates.owner || !coordinates.writable {
		panic("private data mutation crosses its admitted owner")
	}
	data := coordinates.data
	if tx.transactionState == data.transactionState {
		data = tx
	}
	data.requireMutationBinding()
	ready = true
	return data, finish
}

// INSERT is also a per-row streaming sink. Its admitted target already names
// the exact checked invocation, whose existing borrowMutation closes/drains
// writer calls. Do not acquire another shared data-view lease for each row.
// Other inputs retain writeDataContext's ordinary operator admission, including
// its sealed-view rejection for an escaped or unmatched binding.
func (t *table) insertDataContext(tx *TxContext) (*TxContext, func()) {
	if tx != nil && tx.relationData != nil {
		if inputs := tx.relationData.insertInputs.Load(); inputs != nil {
			if data := inputs.targets[t]; data != nil {
				return data, releaseNoRelationData
			}
		}
	}
	return t.writeDataContext(tx)
}

// Ordinary scoped MEMORY and reconstructible execution tables keep their
// historical caller ownership, including scalar-function-local relations.
// Independent data never derives its authority from a SQL alias or spelling.
func (t *table) callerExecutionRelation() bool {
	return t.dataOwner == nil && (t.isEphemeralQueryTable() ||
		(t.Ephemeral && t.PersistencyMode == Memory && t.relationLifetime != nil &&
			t.relationLifetime.dataPolicy == relationCallerData && t.relationLifetime.owns(t)))
}

func (operation *relationDataOperation) producer(public *TxContext, targets []*table, producer scm.Scmer) scm.Scmer {
	return scm.NewFunc(func(arguments ...scm.Scmer) scm.Scmer {
		data := scmerToTxContext(arguments[0])
		view := newRelationDataView(operation.owner, public, data, true)
		view.coordinates.operation = operation
		view.bindInsertInputs(targets, data)
		data.relationData = view
		facade := copyRelationExecution(public, public.transactionState)
		facade.relationData = view
		view.coordinates.read = facade
		operation.viewsMu.Lock()
		operation.views = append(operation.views, view)
		operation.facades = append(operation.facades, facade)
		operation.dataBindings = append(operation.dataBindings, data)
		operation.viewsMu.Unlock()
		defer facade.endStatement()
		return scm.Apply(producer, txContextScmer(facade))
	})
}

// A caller-owned execution relation (for example an ordinary scalar function's
// local MEMORY table) keeps its existing state and native Undo owner. Only its
// lexical reader facade shares the independent producer's private input route.
func (operation *relationDataOperation) callerProducer(parent *relationDataCoordinates, targets []*table, producer scm.Scmer) scm.Scmer {
	return scm.NewFunc(func(arguments ...scm.Scmer) scm.Scmer {
		public := scmerToTxContext(arguments[0])
		view := newRelationDataView(operation.owner, public, parent.data, parent.writable)
		view.coordinates.operation = operation
		view.bindInsertInputs(targets, public)
		facade := copyRelationExecution(public, public.transactionState)
		facade.relationData = view
		view.coordinates.read = facade
		operation.viewsMu.Lock()
		operation.views = append(operation.views, view)
		operation.facades = append(operation.facades, facade)
		operation.viewsMu.Unlock()
		defer facade.endStatement()
		return scm.Apply(producer, txContextScmer(facade))
	})
}

func (scope *relationScope) mutateData(targets []*table, descriptor nativeMutationDescriptor, public *TxContext, session, producer scm.Scmer) scm.Scmer {
	finishScope := scope.beginUse()
	defer finishScope()
	if !scope.dataWriter.CompareAndSwap(false, true) {
		panic("private data scope already has a mutation producer")
	}
	transferred := false
	defer func() {
		if !transferred {
			scope.dataWriter.Store(false)
		}
	}()
	finishOrigin := borrowRelationDataOrigin(public)
	defer func() {
		if !transferred {
			finishOrigin()
		}
	}()
	operation := scope.newDataOperation(public, true, finishOrigin, func() { scope.dataWriter.Store(false) })
	transferred = true
	var result scm.Scmer
	var failure any
	func() {
		defer func() { failure = recover() }()
		result = withNativeTableMutation(targets, descriptor, operation.context, session, operation.producer(public, targets, producer))
	}()
	if failure != nil {
		operation.commit = false
	}
	operation.retire()
	if failure != nil {
		panic(failure)
	}
	return result
}

func nativeMutationDataOwner(targets []*table) *relationScope {
	var owner *relationScope
	public := false
	for _, target := range targets {
		if target == nil {
			panic("native mutation target does not exist")
		}
		if target.dataOwner == nil {
			public = true
			continue
		}
		if owner != nil && owner != target.dataOwner {
			panic("native mutation crosses private data owners")
		}
		owner = target.dataOwner
		if !target.Ephemeral || target.PersistencyMode != Memory || !owner.owns(target) {
			panic("independent data mutation lost its actual private relation")
		}
	}
	if owner != nil && public {
		panic("native mutation mixes public and private data targets")
	}
	return owner
}

// Called after normal writer admission has fenced declaration changes.
func validateIndependentDataTarget(target *table) {
	if sharedClusterDatabase(target.schema.Name) || activeReplication(target.schema) != nil ||
		len(target.Foreign) != 0 || target.hasNonStatementTriggers.Load() || target.statementTriggerEvents.Load() != 0 {
		panic("independent data mutation requires its local private relation without external row effects")
	}
}
