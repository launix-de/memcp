/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "errors"
import "github.com/launix-de/memcp/scm"

func initTransactionState(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{
		Name: "transaction_rollback_only_failure?",
		Fn: func(args ...scm.Scmer) scm.Scmer {
			cause := nativeFailureOriginalValue(args[0])
			failure, ok := cause.(error)
			var admission *rollbackOnlyViolation
			return scm.NewBool(ok && errors.As(failure, &admission))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "recognizes an actual native rollback-only admission failure; displayed error text grants no authority", Const: true,
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "failure"}}, Return: &scm.TypeDescriptor{Kind: "bool"}},
	})
	scm.Declare(en, &scm.Declaration{
		Name: "tx_state",
		Fn: func(args ...scm.Scmer) scm.Scmer {
			tx := scmerToTxContext(args[0])
			if tx == nil {
				return scm.NewString("none")
			}
			// Statement owners serialize execution; the native metadata mutex
			// also orders this observation with commit/rollback publication.
			tx.mu.Lock()
			state := tx.State
			rollbackOnly := tx.rollbackOnly.Load()
			deciding := tx.decisionPending()
			tx.mu.Unlock()
			if deciding {
				return scm.NewString("deciding")
			}
			switch state {
			case TxActive:
				if rollbackOnly {
					return scm.NewString("rollback-only")
				}
				return scm.NewString("active")
			case TxCommitted:
				return scm.NewString("committed")
			case TxAborted:
				return scm.NewString("aborted")
			default:
				panic("invalid native transaction state")
			}
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "observes an existing native transaction lifecycle without creating or resetting it", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "string"}},
	})
}
