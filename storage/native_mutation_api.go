/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"

// The caller supplies actual targets once at the OUTER producer, before source
// scans/defaults/identity allocation. INSERT SELECT and bulk sinks borrow the
// resulting invocation; they do not create one producer per emitted source row.
func withTableMutation(targets []*table, descriptor nativeMutationDescriptor, parent *TxContext, session, producer scm.Scmer) scm.Scmer {
	owner := nativeMutationDataOwner(targets)
	if parent != nil && parent.relationData != nil {
		coordinates, finish := parent.relationData.borrow()
		defer finish()
		if owner == nil {
			execution := len(targets) != 0
			for _, target := range targets {
				execution = execution && target.callerExecutionRelation()
			}
			if execution {
				for _, target := range targets {
					if !target.isEphemeralQueryTable() && (len(target.Foreign) != 0 || target.synchronousRowMutationEvents.Load() != 0 || target.statementTriggerEvents.Load() != 0) {
						panic("private source execution cannot cross external caller row effects")
					}
				}
				return withNativeTableMutation(targets, descriptor, coordinates.public, session,
					coordinates.operation.callerProducer(coordinates, targets, producer))
			}
		}
		if owner != coordinates.owner || !coordinates.writable {
			panic("native mutation crosses its private data producer")
		}
		data := coordinates.data
		if parent.transactionState == data.transactionState {
			data = parent
		}
		return withNativeTableMutation(targets, descriptor, data, session,
			coordinates.operation.producer(coordinates.public, targets, producer))
	}
	if owner != nil {
		return owner.mutateData(targets, descriptor, parent, session, producer, false)
	}
	return withNativeTableMutation(targets, descriptor, parent, session, producer)
}

// The data-owner selector above supplies one actual native transaction. This
// owner keeps the existing write admission, statement Undo and callback ABI.
func withNativeTableMutation(targets []*table, descriptor nativeMutationDescriptor, parent *TxContext, session, producer scm.Scmer) scm.Scmer {
	var callerInputs map[*relationScope]bool
	for _, target := range targets {
		if scope := target.callerRelationScope(); scope != nil && !callerInputs[scope] {
			leave := target.beginCallerInput(parent)
			defer leave()
			if callerInputs == nil {
				callerInputs = make(map[*relationScope]bool)
			}
			callerInputs[scope] = true
		}
	}
	checked := parent != nil && parent.mutationOperation != nil
	checked = checked || parent != nil && parent.relationData != nil && nativeMutationDataOwner(targets) != nil
	var declarationSource *publicCatalogViews
	var declarationTarget *table
	var db *database
	var multipleDatabases bool
	localStatementTargets := true
	storedStatementTarget := false
	uniqueTargets := make([]*table, 0, len(targets))
	seen := make(map[*table]struct{}, len(targets))
	for _, target := range targets {
		if target == nil {
			panic("native mutation target does not exist")
		}
		if _, repeated := seen[target]; repeated {
			continue
		}
		seen[target] = struct{}{}
		uniqueTargets = append(uniqueTargets, target)
		// Acquire the existing writer boundary BEFORE observing hasCheck. An
		// ADD CHECK cannot publish between this observation and the first sink.
		if parent != nil && parent.scopedRelationLayout(target) != nil {
			// Only the actual checked invocation below may borrow the pending
			// layout's retained DDL lease. The raw parent grants no row rights.
			parent.requireScopedRelationOwner(target)
			checked = true
		} else if source := parent.privateMutationSource(target); source != nil {
			if declarationTarget != nil && declarationTarget != target {
				panic("private declaration input requires one checked native target")
			}
			if !source.mutationEventAllowed(target, descriptor.event) {
				panic("private declaration input requires a checked native INSERT or UPDATE producer")
			}
			declarationSource, declarationTarget = source, target
			checked = true
		} else if len(targets) == 1 && parent.catalogCreationRowSource(target) {
			// This source needs the actual prepared statement/input below. The
			// raw parent still has no grant to mutate this committed endpoint.
			declarationTarget, checked = target, true
		} else {
			release := target.admitNativeWrite(parent)
			defer release()
		}
		if target.dataOwner != nil {
			validateIndependentDataTarget(target)
		}
		checked = checked || target.hasCheckConstraints.Load() || target.statementTriggerEvents.Load() != 0
		// Lexical caller data is retired before its public transaction ends.
		// Its SQL mutation callbacks therefore need the existing producer/frame
		// owner, rather than a raw grant retained until the public decision.
		checked = checked || parent != nil && target.callerRelationScope() != nil
		// Synchronous callbacks can reach a checked/statement-triggered target
		// that is not named by this producer. Own the native savepoint BEFORE the
		// first outer write, not only when that later target happens to open one.
		// Distributed and nil-Tx legacy profiles are unchanged.
		localStatementTarget := parent != nil && parent.mutationOperation == nil &&
			(parent.Mode == TxCursorStability || parent.Mode == TxACID) &&
			activeReplication(target.schema) == nil && !sharedClusterDatabase(target.schema.Name) && !target.isEphemeralQueryTable()
		localStatementTargets = localStatementTargets && localStatementTarget
		// Caller-owned execution scratch does not itself require public Undo.
		// Transactional scoped declarations (including SQL temporary tables)
		// remain stored-data targets. Keep all explicit CHECK/trigger/parent
		// owners, and mixed producers with any real stored-data target.
		storedStatementTarget = storedStatementTarget || !target.callerExecutionRelation() || target.ScopedSchema
		if localStatementTarget {
			// Ordinary writer admission excludes staged FK publication.
			// The existing declaration counter also closes legacy FK and user-trigger
			// publication until this SAME producer returns; system cache
			// churn remains on its own owner and never selects this profile.
			definitions := target.borrowTriggerDeclarations(parent)
			defer definitions.Close()
			events := target.synchronousRowMutationEvents.Load() | target.cascadingForeignMutationEvents.Load()
			checked = checked || events&(1<<descriptor.event) != 0
		}
		if db == nil {
			db = target.schema
		} else if db != target.schema {
			multipleDatabases = true
		}
	}
	targets = uniqueTargets
	// The existing local statement admission owns ordinary ACID errors too,
	// not only CHECK/trigger errors. Selection grants no mode authority: the
	// same prepareStatement resource still admits and pins the actual database
	// before any producer effect. Unbound, distributed, multi-database and
	// query-scratch profiles retain their existing native execution owners.
	if parent != nil && parent.Mode == TxACID && localStatementTargets && storedStatementTarget && !multipleDatabases && db != nil &&
		!session.IsNil() && (session.IsProc() || session.IsNativeFunc() || session.IsJIT()) {
		checked = true
	}
	if !checked {
		// Preserve the ordinary CS execution/decision owner. This local runtime
		// input records an existing producer contract, not checked statement
		// admission or another transaction. Distributed operators retain their
		// existing native protocol rather than receiving unencoded authority.
		if parent != nil && descriptor.event == nativeMutationUpdate && parent.remoteObject == nil &&
			!parent.remoteInvocation && parent.coordinator == nil && parent.relationData == nil {
			local := true
			for _, target := range targets {
				local = local && target.dataOwner == nil && activeReplication(target.schema) == nil && !sharedClusterSource(target)
			}
			if local {
				input := &nativeMutationInput{execution: parent, owner: parent.transactionState, targets: targets, event: descriptor.event}
				return scm.Apply(producer, scm.NewAny(input))
			}
		}
		return scm.Apply(producer, txContextScmer(parent))
	}
	if parent == nil || db == nil || multipleDatabases {
		panic("CHECK statement requires native targets in one local database")
	}
	if session.IsNil() || (!session.IsProc() && !session.IsNativeFunc() && !session.IsJIT()) {
		panic("native mutation requires its actual invocation session")
	}
	operation, err := newNativeMutationBinding(parent, session)
	if err != nil {
		panic(err)
	}
	if len(callerInputs) != 0 {
		operation.callerInputs.Store(&callerInputs)
	}
	return operation.runProducer(func(binding *TxContext) scm.Scmer {
		for _, target := range targets {
			if target.ScopedSchema {
				// The supplied execution session can differ from the parent's.
				// Borrowing its scope above cannot authorize a foreign invocation.
				binding.requireScopedRelationOwner(target)
			}
		}
		if operation.undoOwner == nil {
			operation.beginStatement(db)
		}
		for _, target := range targets {
			operation.admitStatementTable(target)
		}
		if operation.undoOwner == nil {
			operation.prepareStatement()
		}
		if declarationTarget != nil {
			binding.declarationInput = declarationTarget.bindMutationInput(binding, declarationSource, descriptor.event)
			if declarationSource == nil {
				// A public baseline borrows no declaration lease. Retain its real
				// ordinary writer through the same original transaction decision.
				release := declarationTarget.admitNativeWrite(binding)
				defer release()
			}
		}
		physicalTargets := make([]*table, 0, len(targets))
		for _, target := range targets {
			// Only the actual replacement body may issue its ordinary same-target
			// operation. The new child receives no such authority of its own, so
			// row/AFTER callbacks cannot accidentally suppress another replacement.
			bypassReplacement := parent.mutationOperation != nil &&
				parent.mutationOperation.replacementTarget == target && parent.mutationOperation.replacementEvent == descriptor.event
			programs := target.statementTriggerProgramsFor(descriptor.event, binding, bypassReplacement)
			operation.declareEffect(target, descriptor, programs)
			if len(programs) == 0 || !programs[0].replacement {
				physicalTargets = append(physicalTargets, target)
			}
		}
		operation.rejectStatementCascades(physicalTargets, descriptor)
		return scm.Apply(producer, txContextScmer(binding))
	})
}

func initMutationBuiltins(en *scm.Env) {
	initMutationAbortBuiltins(en)

	scm.Declare(en, &scm.Declaration{Name: "with_independent_mutation_outcome", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 5 || !a[0].IsSlice() {
			panic("with_independent_mutation_outcome requires targets, event, transaction, session and producer")
		}
		targets := make([]*table, len(a[0].Slice()))
		for i, value := range a[0].Slice() {
			targets[i] = TableFromScmer(value)
		}
		parent := scmerToTxContext(a[2])
		owner := nativeMutationDataOwner(targets)
		if owner == nil || parent == nil || parent.relationData != nil {
			panic("independent mutation outcome requires its actual private relation and caller origin")
		}
		return owner.mutateData(targets, parseNativeMutationDescriptor(a[1]), parent, a[3], a[4], true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "runs a boolean/value producer with independent sink and public-source executions, returning outcome and data metadata after native drain and decision",
		Params: []*scm.TypeDescriptor{{Kind: "list", Label: "targets"}, {Kind: "list", Label: "event"}, {Kind: "any", Label: "tx"},
			{Kind: "any", Label: "session"}, {Kind: "func", Label: "producer"}},
		Return: &scm.TypeDescriptor{Kind: "list"}}})
	scm.Declare(en, &scm.Declaration{Name: "with_table_mutation", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 5 || !a[0].IsSlice() {
			panic("with_table_mutation requires target list, event, transaction, session and producer")
		}
		targets := make([]*table, len(a[0].Slice()))
		for i, value := range a[0].Slice() {
			targets[i] = TableFromScmer(value)
		}
		return withTableMutation(targets, parseNativeMutationDescriptor(a[1]), scmerToTxContext(a[2]), a[3], a[4])
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "owns one native mutation producer, its invocation and checked statement undo",
		Params:      []*scm.TypeDescriptor{{Kind: "list", Label: "targets"}, {Kind: "list", Label: "event"}, {Kind: "any", Label: "tx"}, {Kind: "any", Label: "session"}, {Kind: "func", Label: "producer"}},
		Return:      &scm.TypeDescriptor{Kind: "any"}}})
}
