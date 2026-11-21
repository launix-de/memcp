/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"sync"
	"sync/atomic"

	"github.com/launix-de/memcp/scm"
)

// relationScope owns native identities, never SQL names or syntax. Its capability
// stays in the connection/controller; queries and table serialization use the
// existing native table catalog and coherence resources.
type relationScope struct {
	mu             sync.Mutex
	closeMu        sync.Mutex
	cond           *sync.Cond
	owner          *relationScopeOwner
	parent         *relationScope
	closing        bool
	active         int
	tables         map[*table]struct{}
	children       []*relationScope
	dataPolicy     relationDataPolicy
	dataWriter     atomic.Bool
	dataOperations map[*relationDataOperation]struct{}

	transactionalSchema bool
	schemaSlots         atomic.Pointer[relationSchemaSlots]
	schemaClosed        atomic.Bool
	schemaDecisions     int
}
type relationScopeOwner struct {
	mu     sync.Mutex
	scopes []*relationScope
	closed bool
}

const relationScopeKey = "__memcp_relation_scopes"

// A namespace-local sentinel identifies the connection through delegating
// accessors as well: a wrapper function is not a different owner.
var relationScopeNamespace = scm.NewAny(new(byte))

func relationScopeFromValue(value scm.Scmer) *relationScope {
	scope, ok := value.Any().(*relationScope)
	if !ok || scope == nil {
		panic("expected a relation scope capability")
	}
	return scope
}
func relationScopeOwnerFor(session scm.Scmer) *relationScopeOwner {
	value := scm.Apply(session, scm.NewString("get_or_compute_scoped"), relationScopeNamespace, scm.NewString(relationScopeKey), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.NewAny(&relationScopeOwner{})
	}))
	owner, ok := value.Any().(*relationScopeOwner)
	if !ok {
		panic("invalid relation scope owner")
	}
	return owner
}

// Admission for construction or a native result-reader lease, once per
// operator/shard acquisition. The SAME close barrier seals admission before
// waiting; no row loop registers and no closing thread holds a shard lock.
func (scope *relationScope) beginUse() func() {
	scope.mu.Lock()
	if scope.closing {
		scope.mu.Unlock()
		panic("relation scope is closed")
	}
	scope.active++
	scope.mu.Unlock()
	return func() { scope.mu.Lock(); scope.active--; scope.cond.Broadcast(); scope.mu.Unlock() }
}
func (scope *relationScope) adopt(table *table) {
	scope.mu.Lock()
	scope.tables[table] = struct{}{}
	scope.mu.Unlock()
}
func (scope *relationScope) owns(table *table) bool {
	scope.mu.Lock()
	defer scope.mu.Unlock()
	for owned := range scope.tables {
		if owned.RuntimeIdentity == table.RuntimeIdentity {
			return true
		}
	}
	return false
}

// Close lets the existing transaction resource owner retire a scope only
// after its row decision and mutation leases have finished. Failed retirement
// remains retryable through that SAME resource and connection scope owner.
func (scope *relationScope) Close() error {
	scope.close()
	return nil
}

func (scope *relationScope) close() {
	scope.closeMu.Lock()
	defer scope.closeMu.Unlock()
	scope.mu.Lock()
	scope.closing = true
	scope.schemaClosed.Store(true)
	if scope.dataPolicy == relationIndependentData && scope.active != 0 {
		scope.mu.Unlock()
		panic("private data scope still owns active operator inputs")
	}
	for scope.active != 0 {
		scope.cond.Wait()
	}
	children := append([]*relationScope(nil), scope.children...)
	tables := make([]*table, 0, len(scope.tables))
	for table := range scope.tables {
		tables = append(tables, table)
	}
	scope.mu.Unlock()
	var first any
	attempt := func(fn func()) (ok bool) {
		defer func() {
			if cause := recover(); cause != nil {
				if first == nil {
					first = cause
				}
				ok = false
			}
		}()
		fn()
		return true
	}
	for i := len(children) - 1; i >= 0; i-- {
		attempt(children[i].close)
	}
	// Private data decisions retire before native maintenance drops their
	// identities. Their source transaction is a distinct borrowed authority.
	if !attempt(scope.retireDataOperations) {
		panic(first)
	}
	scope.mu.Lock()
	pendingSchema := scope.schemaDecisions != 0
	scope.mu.Unlock()
	if pendingSchema {
		if first != nil {
			panic(first)
		}
		return
	}
	for _, table := range tables {
		if attempt(func() {
			// A database or same-named relation may have been explicitly replaced.
			// The closed capability can only retire its original live incarnation.
			if db := GetDatabase(table.schema.Name); db == table.schema {
				dropTableIdentity(db.Name, table.Name, true, table)
			}
		}) {
			scope.mu.Lock()
			delete(scope.tables, table)
			scope.mu.Unlock()
		}
	}
	if first != nil {
		panic(first)
	}
	// A retained closed capability must not retain the retired MEMORY rows
	// through its former slot catalog. Pending decisions return before this point.
	scope.schemaSlots.Store(nil)
	// Closed module/reset scopes do not accumulate on a pooled connection.
	scope.owner.mu.Lock()
	for i, owned := range scope.owner.scopes {
		if owned == scope {
			scope.owner.scopes = append(scope.owner.scopes[:i], scope.owner.scopes[i+1:]...)
			break
		}
	}
	scope.owner.mu.Unlock()
	if scope.parent != nil {
		scope.parent.mu.Lock()
		for i, child := range scope.parent.children {
			if child == scope {
				scope.parent.children = append(scope.parent.children[:i], scope.parent.children[i+1:]...)
				break
			}
		}
		scope.parent.mu.Unlock()
	}
}

// A disconnected session may leave an accepted decision to the existing
// preparation finisher. Retain its final cleanup with that native decision.
type transactionSessionClose struct {
	executor *TxContext
	session  scm.Scmer
}

// Caller owns finalizationMu. Failed scope retirement remains in the request
// and is retried by the same prepared-decision finisher, never another worker.
func (tx *TxContext) finishRequestedSessionClose() {
	request := tx.sessionClose
	if request == nil || tx.State == TxActive || tx.decisionPending() {
		return
	}
	request.executor.finishSessionClose(request.session)
	tx.sessionClose = nil
}

func closeSessionResources(session scm.Scmer) {
	// Transaction rollback releases reader/writer ownership before owned relation
	// retirement. This ordering also applies without replication being configured.
	value := scm.Apply(session, scm.NewString("__memcp_tx"))
	if tx, ok := value.Any().(*TxContext); ok && tx != nil {
		// Close must fail before queryMu when called by its own active mutation.
		// The attempt owns capture/close against reset while waiting for a foreign
		// request to unwind, without holding bindingMu during that wait.
		attempt, err := tx.acquireMutationDecisionAttempt("session close")
		if err != nil {
			panic(err)
		}
		tx.queryMu.Lock()
		defer tx.queryMu.Unlock()
		if err := attempt.runDecision(func(decision *TxContext) error {
			if decision.decisionPending() {
				if decision.sessionClose == nil {
					decision.sessionClose = &transactionSessionClose{executor: tx, session: session}
				}
				return decision.commit()
			}
			if decision.sessionClose != nil {
				return decision.finishResourceCommit()
			}
			defer func() {
				if decision.State != TxActive && !decision.decisionPending() {
					tx.finishSessionClose(session)
				}
			}()
			if decision.State == TxActive {
				decision.rollback()
			}
			return nil
		}); err != nil {
			panic(err)
		}
		return
	}
	closeSessionRelationScopes(session)
}

// The decision has retired before this tail releases connection-owned scopes.
// Attempt independent cleanup even if a preceding release fails.
func (tx *TxContext) finishSessionClose(session scm.Scmer) {
	var first any
	attempt := func(fn func()) {
		defer func() {
			if failure := recover(); failure != nil && first == nil {
				first = failure
			}
		}()
		fn()
	}
	if tx.SessionState != nil {
		attempt(tx.SessionState.ReleaseAllLocks)
	}
	if tx.replicationActive {
		attempt(func() { replicationLeave(tx); tx.replicationActive = false })
	}
	attempt(func() { closeSessionRelationScopes(session) })
	if first != nil {
		panic(first)
	}
}

func closeSessionRelationScopes(session scm.Scmer) {
	// The canonical owner belongs to hidden scoped state, never the public
	// session map carried to workers. Closing an unused session creates and
	// seals its empty owner too, preventing later relation construction.
	owner := relationScopeOwnerFor(session)
	owner.mu.Lock()
	owner.closed = true
	scopes := append([]*relationScope(nil), owner.scopes...)
	owner.mu.Unlock()
	var first any
	for i := len(scopes) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if failure := recover(); failure != nil && first == nil {
					first = failure
				}
			}()
			scopes[i].close()
		}()
	}
	if first != nil {
		panic(first)
	}
}

// Both public lexical scopes and native operation relations use this SAME
// connection-owned lifetime. Native callers supply a real session owner, not a
// detached catalog or a second table registry.
func newRelationScope(owner *relationScopeOwner, parent *relationScope, policy relationDataPolicy, transactionalSchema bool) *relationScope {
	scope := &relationScope{owner: owner, tables: make(map[*table]struct{}), dataPolicy: policy, transactionalSchema: transactionalSchema}
	scope.cond = sync.NewCond(&scope.mu)
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.closed {
		panic("relation scope owner is closed")
	}
	if parent != nil {
		parent.mu.Lock()
		defer parent.mu.Unlock()
		if parent.owner != owner || parent.closing {
			panic("invalid parent relation scope")
		}
		if parent.dataPolicy != policy || parent.transactionalSchema != transactionalSchema {
			panic("child relation scope must preserve its data and schema policies")
		}
		scope.parent = parent
		parent.children = append(parent.children, scope)
	}
	owner.scopes = append(owner.scopes, scope)
	return scope
}

func initRelationScopes(en *scm.Env) {
	initRelationSchema(en)
	scm.Declare(en, &scm.Declaration{Name: "new_relation_scope", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) < 1 || len(a) > 3 {
			panic("new_relation_scope requires session, optional parent and optional profile")
		}
		var parent *relationScope
		if len(a) > 1 && !a[1].IsNil() {
			parent = relationScopeFromValue(a[1])
		}
		policy := relationCallerData
		transactionalSchema := false
		if parent != nil {
			policy = parent.dataPolicy
			transactionalSchema = parent.transactionalSchema
		}
		if len(a) == 3 {
			if !a[2].IsString() {
				panic("relation profile must be caller, independent or transactional")
			}
			switch a[2].String() {
			case "caller":
				policy = relationCallerData
				transactionalSchema = false
			case "transactional":
				policy = relationCallerData
				transactionalSchema = true
			case "independent":
				policy = relationIndependentData
				transactionalSchema = false
			default:
				panic("unknown relation profile")
			}
		}
		return scm.NewAny(newRelationScope(relationScopeOwnerFor(a[0]), parent, policy, transactionalSchema))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "creates an opaque native relation lifetime owned by a connection session; independent MEMORY data uses separate native statement decisions; transactional scopes enlist lexical schema generations in the actual caller decision", Params: []*scm.TypeDescriptor{{Kind: "func", Label: "session"}, {Kind: "any", Label: "parent", Optional: true}, {Kind: "string", Label: "profile", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "close_relation_scope", Fn: func(a ...scm.Scmer) scm.Scmer {
		relationScopeFromValue(a[0]).close()
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "closes a native relation scope and retires only its owned native identities; repeat calls are safe", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "scope"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
