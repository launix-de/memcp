/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import (
	"math"
	"math/big"
)

// numberPower keeps integral powers exact while they fit the native integer
// carrier. Wider and fractional results use ordinary floating arithmetic;
// callers own their language's domains, widths and overflow contracts.
func numberPower(base, exponent Scmer) Scmer {
	if base.IsNil() || exponent.IsNil() {
		return NewNil()
	}
	if base.IsInt() && exponent.IsInt() && exponent.Int() >= 0 {
		result, factor := big.NewInt(1), big.NewInt(base.Int())
		for remaining := exponent.Int(); remaining > 0; remaining >>= 1 {
			if remaining&1 != 0 {
				result.Mul(result, factor)
				if !result.IsInt64() {
					return NewFloat(math.Pow(base.Float(), exponent.Float()))
				}
			}
			if remaining > 1 {
				factor.Mul(factor, factor)
				if !factor.IsInt64() {
					return NewFloat(math.Pow(base.Float(), exponent.Float()))
				}
			}
		}
		return NewInt(result.Int64())
	}
	return NewFloat(math.Pow(base.Float(), exponent.Float()))
}

func initNumberPower() {
	Declare(&Globalenv, &Declaration{
		Name: "number_power",
		Fn:   func(a ...Scmer) Scmer { return numberPower(a[0], a[1]) },
		Type: &TypeDescriptor{Kind: "func", Description: "raises a number to a power, preserving exact integral results within the native integer range",
			Params: []*TypeDescriptor{{Kind: "number", Label: "base"}, {Kind: "number", Label: "exponent"}},
			Return: &TypeDescriptor{Kind: "number"}, Const: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				return jitEmitGeneratedCallBoundary(ctx, declarations["number_power"], sourceArgs, args, result)
			},
			JITVirtualArgs: true, JITInlineCost: 65535,
		},
	})
}
