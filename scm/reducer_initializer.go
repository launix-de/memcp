/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

// reducerInitialValue is the same creation-time value/factory boundary used
// by native reducers. It must not be constant-folded: a factory belongs to
// each execution of its reducer, even when the callable itself is immutable.
func reducerInitialValue(a ...Scmer) Scmer {
	return ApplyOnce(a[0], nil)
}

func initReducerInitializer() {
	Declare(&Globalenv, &Declaration{
		Name: "reducer_initial_value",
		Fn:   reducerInitialValue,
		Type: &TypeDescriptor{Kind: "func", Description: "Initialize a reducer from a plain value or zero-argument factory", HasSideEffects: true,
			Params: []*TypeDescriptor{{Kind: "any", Label: "value_or_factory"}},
			Return: &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitReducerInitializer(ctx, args[0])
			},
			JITVirtualArgs: true,
		},
	})
}
