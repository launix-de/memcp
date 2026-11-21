/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "github.com/launix-de/go-mysqlstack/sqldb"

// failureOriginalValue projects only the existing explicit diagnostic carrier.
// It does not traverse arbitrary error wrappers or confer recovery authority.
// Native completion decisions must continue to retain the original envelope.
func failureOriginalValue(value Scmer) Scmer {
	for value.GetTag() == tagAny {
		carrier, ok := value.Any().(interface{ OriginalFailure() any })
		if !ok {
			break
		}
		value = FromAny(carrier.OriginalFailure())
	}
	return value
}

// failureDiagnostic reads the actual typed backend diagnostic. Unknown runtime,
// cancellation, persistence and cleanup errors remain opaque; error strings are
// never inspected. Frontends own any interpretation of the code and SQLSTATE.
func failureDiagnostic(value Scmer) Scmer {
	value = failureOriginalValue(value)
	if value.GetTag() != tagAny {
		return NewNil()
	}
	diagnostic, ok := value.Any().(*sqldb.SQLError)
	if !ok || diagnostic == nil {
		return NewNil()
	}
	return NewSlice([]Scmer{
		NewString("code"), NewInt(int64(diagnostic.Num)),
		NewString("sqlstate"), NewString(diagnostic.State),
		NewString("message"), NewString(diagnostic.Message),
	})
}

func initFailureDiagnostic() {
	initFailureAnnotations()
	Declare(&Globalenv, &Declaration{
		Name: "failure_diagnostic",
		Fn: func(arguments ...Scmer) Scmer {
			return failureDiagnostic(arguments[0])
		},
		Type: &TypeDescriptor{Kind: "func", Const: true,
			Description: "reads code, SQLSTATE and message from an actual typed backend diagnostic after explicit original-failure projection; returns nil for other failures and grants no recovery authority",
			Params:      []*TypeDescriptor{{Kind: "any", Label: "failure"}},
			Return:      &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				return jitEmitGeneratedCallBoundary(ctx, declarations["failure_diagnostic"], sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
}
