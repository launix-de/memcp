/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "errors"
import "context"
import "runtime"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

// This is producer provenance, never an attestation that public statement Undo
// ran. It is created only inside the actual admitted independent producer; its
// exact pointer must survive the existing native completion before it can be
// projected into a returned outcome. No transaction or callback is retained.
type independentMutationProducerFailure struct{ cause scm.Scmer }

func (failure *independentMutationProducerFailure) Error() string        { return fmt.Sprint(failure.cause) }
func (failure *independentMutationProducerFailure) OriginalFailure() any { return failure.cause }

type independentMutationOutcome struct {
	failure             *independentMutationProducerFailure
	dataFailure         *sqldb.SQLError
	transactionID       uint64
	statementGeneration uint64
	sourceDrainFailure  any
}

// Data exceptions and integrity constraints are typed native data failures.
// Neither arbitrary Go errors nor error text grant this admission. In
// particular, restoration/cleanup, cancellation and terminal transfers must
// continue through their real native owners.
func independentMutationDataFailure(value any) *sqldb.SQLError {
	failure, ok := value.(error)
	if !ok {
		return nil
	}
	var persistence *PersistenceFailure
	var cleanup *transactionResourceCleanupError
	var busy *nativeMutationBusyError
	var terminal *nativeTransactionAbortTransfer
	var fatal runtime.Error
	if errors.Is(failure, context.Canceled) || errors.Is(failure, context.DeadlineExceeded) ||
		errors.As(failure, &persistence) || errors.As(failure, &cleanup) || errors.As(failure, &busy) ||
		errors.As(failure, &terminal) || errors.As(failure, &fatal) {
		return nil
	}
	var data *sqldb.SQLError
	if errors.As(failure, &data) && data != nil && len(data.State) == 5 && (data.State[:2] == "22" || data.State[:2] == "23") {
		return data
	}
	return nil
}

func (completion *independentMutationOutcome) producer(producer scm.Scmer, source *TxContext) scm.Scmer {
	return scm.NewFunc(func(arguments ...scm.Scmer) (result scm.Scmer) {
		if len(arguments) != 1 {
			panic("independent mutation requires its single admitted sink invocation")
		}
		// This actual public-source scope owns its work separately from the
		// private sink. Drain it before the private producer can complete/Undo;
		// later retire clears this same facade in the existing lifetime array.
		defer completion.finishSource(source)
		var cause any
		func() {
			defer func() { cause = recover() }()
			result = scm.Apply(producer, arguments[0], txContextScmer(source))
		}()
		if cause != nil {
			data := independentMutationDataFailure(cause)
			if data == nil {
				panic(cause)
			}
			completion.dataFailure = data
			completion.failure = &independentMutationProducerFailure{scm.FromAny(cause)}
			panic(completion.failure)
		}
		if !result.IsSlice() || len(result.Slice()) != 2 || !result.Slice()[0].IsBool() {
			panic("independent mutation producer requires a boolean/value outcome tuple")
		}
		parts := result.Slice()
		if !parts[0].Bool() {
			if parts[1].IsNil() {
				panic("independent mutation failure requires its original cause")
			}
			// The producer explicitly returned its language-level failure. The
			// native owner still performs its complete private rollback, not a
			// successful commit of the rows emitted before that outcome.
			completion.failure = &independentMutationProducerFailure{parts[1]}
			panic(completion.failure)
		}
		return parts[1]
	})
}

func (completion *independentMutationOutcome) ownsFailure(failure any) bool {
	if completion.failure == nil || completion.sourceDrainFailure != nil {
		return false
	}
	// Only the exact native completion envelope is accepted. Generic error
	// unwrapping here could accidentally erase a later cleanup failure.
	if restored, ok := failure.(*nativeStatementRestoredError); ok {
		if !restored.restoredBy(completion.transactionID, completion.statementGeneration) {
			return false
		}
		// A writer records its first data failure before its producer unwinds.
		// Accept only that exact positively admitted object, after this SAME
		// native statement has successfully completed its private Undo.
		if data, ok := restored.cause.(*sqldb.SQLError); ok && data == completion.dataFailure {
			return true
		}
		failure = restored.cause
	}
	producer, ok := failure.(*independentMutationProducerFailure)
	return ok && producer == completion.failure
}

// The failure object remains unchanged and throwable. Only its already typed,
// bounded data diagnostic is projected for a caller's language disposition;
// this descriptor carries no mutation or recovery authority.
func (completion *independentMutationOutcome) metadata() scm.Scmer {
	if completion.dataFailure == nil {
		return scm.NewNil()
	}
	return scm.NewSlice([]scm.Scmer{scm.NewString("sqlstate"), scm.NewString(completion.dataFailure.State),
		scm.NewString("number"), scm.NewInt(int64(completion.dataFailure.Num))})
}

// Source cleanup runs after the entered producer, outside its data/language
// failure admission. A later drain failure must not disappear behind the
// native writer's earlier first data failure. Keep that actual phase cause;
// no kind/message heuristic or synthetic restored fact selects its priority.
func (completion *independentMutationOutcome) finishSource(source *TxContext) {
	defer func() {
		if failure := recover(); failure != nil {
			completion.sourceDrainFailure = failure
			panic(failure)
		}
	}()
	source.endStatement()
	clearRelationExecution(source)
}

// The sole production caller invokes this only AFTER operation.retire has
// returned successfully. Retire itself remains outside every recovery here.
func (completion *independentMutationOutcome) afterRetire(result scm.Scmer, failure any) scm.Scmer {
	if completion.sourceDrainFailure != nil {
		panic(completion.sourceDrainFailure)
	}
	if failure != nil {
		if completion.ownsFailure(failure) {
			return scm.NewSlice([]scm.Scmer{scm.NewBool(false), completion.failure.cause, completion.metadata()})
		}
		panic(failure)
	}
	return scm.NewSlice([]scm.Scmer{scm.NewBool(true), result, scm.NewNil()})
}
