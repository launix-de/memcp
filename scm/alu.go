/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package scm

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"strings"
	"unsafe"
)

// jitEmitCompare builds a JIT emitter for comparison operators (<, <=, >, >=).
// goFn computes the compile-time result from two Scmer values (for constant folding).
// swap=true swaps operands for the register path (> is CMP(b,a)).
// intCc/floatCc are condition codes for signed int CMP / unsigned float UCOMISD.
func jitEmitCompare(swap bool, intCc, floatCc byte) func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	return func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		a, b := args[0], args[1]
		// Constant fold using the same semantics as the Go function
		if a.Loc == LocImm && b.Loc == LocImm {
			var val bool
			switch intCc {
			case CcL: // < : Less(a, b)
				if swap {
					val = Less(b.Imm, a.Imm)
				} else {
					val = Less(a.Imm, b.Imm)
				}
			case CcLE: // <= : !Less(b, a)
				val = !Less(b.Imm, a.Imm)
			case CcGE: // >= : !Less(a, b)
				val = !Less(a.Imm, b.Imm)
			}
			r := JITValueDesc{Loc: LocImm, Imm: NewBool(val)}
			if result.Loc == LocAny {
				return r
			}
			ctx.EnsureResultRegPair(&result)
			ctx.W.EmitMakeBool(result, r)
			return result
		}
		if swap {
			a, b = b, a
		}
		ctx.MaterializeToRegPair(&a)
		ctx.MaterializeToRegPair(&b)
		ctx.EnsureResultRegPair(&result)
		boolReg := ctx.AllocReg()
		lblNotBothInt := ctx.W.ReserveLabel()
		lblDone := ctx.W.ReserveLabel()
		// Check both int
		ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
		ctx.W.EmitCmpInt64(a.Reg, RegR11)
		ctx.W.EmitJcc(CcNE, lblNotBothInt)
		ctx.W.EmitCmpInt64(b.Reg, RegR11)
		ctx.W.EmitJcc(CcNE, lblNotBothInt)
		// Both int: CMP a.aux, b.aux
		ctx.W.EmitCmpInt64(a.Reg2, b.Reg2)
		ctx.W.EmitSetcc(boolReg, intCc)
		ctx.W.EmitJmp(lblDone)
		// Float path
		ctx.W.MarkLabel(lblNotBothInt)
		ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
		lblF0 := ctx.W.ReserveLabel()
		lblG0 := ctx.W.ReserveLabel()
		ctx.W.EmitCmpInt64(a.Reg, RegR11)
		ctx.W.EmitJcc(CcNE, lblF0)
		ctx.W.EmitCvtInt64ToFloat64(RegX0, a.Reg2)
		ctx.W.EmitJmp(lblG0)
		ctx.W.MarkLabel(lblF0)
		ctx.W.emitMovqGprToXmm(RegX0, a.Reg2)
		ctx.W.MarkLabel(lblG0)
		lblF1 := ctx.W.ReserveLabel()
		lblG1 := ctx.W.ReserveLabel()
		ctx.W.EmitCmpInt64(b.Reg, RegR11)
		ctx.W.EmitJcc(CcNE, lblF1)
		ctx.W.EmitCvtInt64ToFloat64(RegX1, b.Reg2)
		ctx.W.EmitJmp(lblG1)
		ctx.W.MarkLabel(lblF1)
		ctx.W.emitMovqGprToXmm(RegX1, b.Reg2)
		ctx.W.MarkLabel(lblG1)
		ctx.W.EmitUcomisd(RegX0, RegX1)
		ctx.W.EmitSetcc(boolReg, floatCc)
		ctx.W.MarkLabel(lblDone)
		ctx.FreeDesc(&a)
		ctx.FreeDesc(&b)
		src := JITValueDesc{Loc: LocReg, Reg: boolReg}
		ctx.W.EmitMakeBool(result, src)
		ctx.FreeReg(boolReg)
		return result
	}
}

func init_alu() {
	// string functions
	DeclareTitle("Arithmetic / Logic")

	Declare(&Globalenv, &Declaration{
		"int?", "tells if the value is a integer",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagInt)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			d0 := args[0]
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Imm: NewInt(int64(d0.Imm.GetTag()))}
			} else {
				d1.Reg = ctx.AllocReg()
				d1.Loc = LocReg
				ctx.W.EmitGetTag(d1.Reg, d0.Reg, d0.Reg2)
				ctx.FreeDesc(&d0)
			}
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Imm: NewBool(d1.Imm.Int() == 4)}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(d1.Reg, CcE)
				d2 = JITValueDesc{Loc: LocReg, Reg: d1.Reg}
			}
			if d2.Loc == LocImm {
				if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: d2.Imm} }
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"number?", "tells if the value is a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			tag := a[0].GetTag()
			return NewBool(tag == tagFloat || tag == tagInt || tag == tagDate)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			a := args[0]
			if a.Loc == LocImm {
				tag := a.Imm.GetTag()
				r := JITValueDesc{Loc: LocImm, Imm: NewBool(tag == tagFloat || tag == tagInt || tag == tagDate)}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeBool(result, r)
				return result
			}
			// Register path: GetTag → check tagFloat(3), tagInt(4), tagDate(15)
			tagReg := ctx.AllocReg()
			ctx.W.EmitGetTag(tagReg, a.Reg, a.Reg2)
			ctx.FreeDesc(&a)
			lblYes := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagFloat))
			ctx.W.EmitJcc(CcE, lblYes)
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagInt))
			ctx.W.EmitJcc(CcE, lblYes)
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagDate))
			ctx.W.EmitJcc(CcE, lblYes)
			// Not a number
			ctx.W.emitXorReg(tagReg) // tagReg = 0 (false)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblYes)
			ctx.W.EmitMovRegImm64(tagReg, 1) // tagReg = 1 (true)
			ctx.W.MarkLabel(lblDone)
			src := JITValueDesc{Loc: LocReg, Reg: tagReg}
			ctx.EnsureResultRegPair(&result)
			ctx.W.EmitMakeBool(result, src)
			ctx.FreeReg(tagReg)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"+", "adds two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values to add", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Fast path: accumulate ints until first non-int, then promote to float if needed
			var sumInt int64
			i := 0
			for i < len(a) {
				v := a[i]
				if v.IsInt() {
					sumInt += v.Int()
					i++
					continue
				}
				break
			}
			if i == len(a) {
				return NewInt(sumInt)
			}
			// Promote to float and continue
			sumFloat := float64(sumInt)
			for ; i < len(a); i++ {
				v := a[i]
				if v.IsNil() {
					return NewNil()
				}
				sumFloat += v.Float()
			}
			return NewFloat(sumFloat)
		},
		true, false, &TypeDescriptor{Optimize: optimizeAssociative},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			// Constant fold: all LocImm
			allImm := true
			for _, a := range args {
				if a.Loc != LocImm {
					allImm = false
					break
				}
			}
			if allImm {
				var sumInt int64
				j := 0
				for j < len(args) {
					if args[j].Imm.IsInt() {
						sumInt += args[j].Imm.Int()
						j++
						continue
					}
					break
				}
				if j == len(args) {
					r := JITValueDesc{Loc: LocImm, Imm: NewInt(sumInt)}
					if result.Loc == LocAny {
						return r
					}
					ctx.EnsureResultRegPair(&result)
					ctx.W.EmitMakeInt(result, r)
					return result
				}
				sumFloat := float64(sumInt)
				for ; j < len(args); j++ {
					if args[j].Imm.IsNil() {
						r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
						if result.Loc == LocAny {
							return r
						}
						ctx.EnsureResultRegPair(&result)
						ctx.W.EmitMakeNil(result)
						return result
					}
					sumFloat += args[j].Imm.Float()
				}
				r := JITValueDesc{Loc: LocImm, Imm: NewFloat(sumFloat)}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeFloat(result, r)
				return result
			}
			// Register path: 2-arg fast path with int/float dispatch
			if len(args) != 2 {
				// Variadic > 2: materialize and chain pairwise
				// For now, only handle 2 args; fall back for more
				return JITValueDesc{}
			}
			a, b := args[0], args[1]
			// Check for LocImm nil short-circuit
			if a.Loc == LocImm && a.Imm.IsNil() {
				ctx.FreeDesc(&b)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			if b.Loc == LocImm && b.Imm.IsNil() {
				ctx.FreeDesc(&a)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			// Materialize any remaining LocImm to registers
			ctx.MaterializeToRegPair(&a)
			ctx.MaterializeToRegPair(&b)
			ctx.EnsureResultRegPair(&result)
			// Labels for branching
			lblNotBothInt := ctx.W.ReserveLabel()
			lblNilResult := ctx.W.ReserveLabel()
			lblFloat0 := ctx.W.ReserveLabel()
			lblGotFloat0 := ctx.W.ReserveLabel()
			lblFloat1 := ctx.W.ReserveLabel()
			lblGotFloat1 := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			// Check both are int
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			// Both int: ADD aux values
			ctx.W.EmitAddInt64(a.Reg2, b.Reg2)
			ctx.W.emitMovRegReg(result.Reg, RegR11) // R11 = &scmerIntSentinel
			ctx.W.emitMovRegReg(result.Reg2, a.Reg2)
			ctx.W.EmitJmp(lblDone)
			// Not both int: check for nil
			ctx.W.MarkLabel(lblNotBothInt)
			ctx.W.EmitTestRegReg(a.Reg)
			ctx.W.EmitJcc(CcNE, lblFloat0) // ptr != nil → not nil
			ctx.W.EmitTestRegReg(a.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult) // ptr == nil && aux == 0 → nil
			// a has ptr=nil but aux!=0 (bool/etc) → treat as float
			ctx.W.MarkLabel(lblFloat0)
			// Check nil for b
			ctx.W.EmitTestRegReg(b.Reg)
			ctx.W.EmitJcc(CcNE, lblFloat1)
			ctx.W.EmitTestRegReg(b.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblFloat1)
			// Float path: convert both to float64 and add
			// arg0: int → CVTSI2SDQ, float → MOVQ bits
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblGotFloat0)
			ctx.W.EmitCvtInt64ToFloat64(RegX0, a.Reg2)
			ctx.W.EmitJmp(lblGotFloat1) // skip the MOVQ
			ctx.W.MarkLabel(lblGotFloat0)
			ctx.W.emitMovqGprToXmm(RegX0, a.Reg2) // float bits → XMM
			ctx.W.MarkLabel(lblGotFloat1)
			// arg1: same dispatch
			lblCvt1 := ctx.W.ReserveLabel()
			lblGotBoth := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblCvt1)
			ctx.W.EmitCvtInt64ToFloat64(RegX1, b.Reg2)
			ctx.W.EmitJmp(lblGotBoth)
			ctx.W.MarkLabel(lblCvt1)
			ctx.W.emitMovqGprToXmm(RegX1, b.Reg2)
			ctx.W.MarkLabel(lblGotBoth)
			// ADDSD
			ctx.W.EmitAddFloat64(RegX0, RegX1)
			// Construct NewFloat
			ctx.W.EmitMovRegImm64(result.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
			ctx.W.emitMovqXmmToGpr(result.Reg2, RegX0)
			ctx.W.EmitJmp(lblDone)
			// Nil result
			ctx.W.MarkLabel(lblNilResult)
			ctx.W.emitXorReg(result.Reg)
			ctx.W.emitXorReg(result.Reg2)
			ctx.W.MarkLabel(lblDone)
			// Free input registers
			ctx.FreeDesc(&a)
			ctx.FreeDesc(&b)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"-", "subtracts two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			// Int-first, then promote to float if needed
			if a[0].IsInt() {
				diffInt := a[0].Int()
				i := 1
				for i < len(a) && a[i].IsInt() {
					diffInt -= a[i].Int()
					i++
				}
				if i == len(a) {
					return NewInt(diffInt)
				}
				diffFloat := float64(diffInt)
				for ; i < len(a); i++ {
					diffFloat -= a[i].Float()
				}
				return NewFloat(diffFloat)
			}
			// Float mode from the start
			diffFloat := a[0].Float()
			for i := 1; i < len(a); i++ {
				diffFloat -= a[i].Float()
			}
			return NewFloat(diffFloat)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			// Constant fold
			allImm := true
			for _, a := range args {
				if a.Loc != LocImm {
					allImm = false
					break
				}
			}
			if allImm {
				// Check nil
				for _, a := range args {
					if a.Imm.IsNil() {
						r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
						if result.Loc == LocAny {
							return r
						}
						ctx.EnsureResultRegPair(&result)
						ctx.W.EmitMakeNil(result)
						return result
					}
				}
				if args[0].Imm.IsInt() {
					diff := args[0].Imm.Int()
					allInt := true
					for i := 1; i < len(args); i++ {
						if args[i].Imm.IsInt() {
							diff -= args[i].Imm.Int()
						} else {
							allInt = false
							break
						}
					}
					if allInt {
						r := JITValueDesc{Loc: LocImm, Imm: NewInt(diff)}
						if result.Loc == LocAny {
							return r
						}
						ctx.EnsureResultRegPair(&result)
						ctx.W.EmitMakeInt(result, r)
						return result
					}
				}
				diffFloat := args[0].Imm.Float()
				for i := 1; i < len(args); i++ {
					diffFloat -= args[i].Imm.Float()
				}
				r := JITValueDesc{Loc: LocImm, Imm: NewFloat(diffFloat)}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeFloat(result, r)
				return result
			}
			if len(args) != 2 {
				return JITValueDesc{}
			}
			a, b := args[0], args[1]
			// Nil short-circuit
			if a.Loc == LocImm && a.Imm.IsNil() {
				ctx.FreeDesc(&b)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			if b.Loc == LocImm && b.Imm.IsNil() {
				ctx.FreeDesc(&a)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			ctx.MaterializeToRegPair(&a)
			ctx.MaterializeToRegPair(&b)
			ctx.EnsureResultRegPair(&result)
			lblNotBothInt := ctx.W.ReserveLabel()
			lblNilResult := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			// Check both int
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			// Both int: SUB
			ctx.W.EmitSubInt64(a.Reg2, b.Reg2)
			ctx.W.emitMovRegReg(result.Reg, RegR11)
			ctx.W.emitMovRegReg(result.Reg2, a.Reg2)
			ctx.W.EmitJmp(lblDone)
			// Not both int
			ctx.W.MarkLabel(lblNotBothInt)
			// Nil checks
			ctx.W.EmitTestRegReg(a.Reg)
			lblA := ctx.W.ReserveLabel()
			ctx.W.EmitJcc(CcNE, lblA)
			ctx.W.EmitTestRegReg(a.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblA)
			ctx.W.EmitTestRegReg(b.Reg)
			lblB := ctx.W.ReserveLabel()
			ctx.W.EmitJcc(CcNE, lblB)
			ctx.W.EmitTestRegReg(b.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblB)
			// Float path
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			lblF0 := ctx.W.ReserveLabel()
			lblG0 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblF0)
			ctx.W.EmitCvtInt64ToFloat64(RegX0, a.Reg2)
			ctx.W.EmitJmp(lblG0)
			ctx.W.MarkLabel(lblF0)
			ctx.W.emitMovqGprToXmm(RegX0, a.Reg2)
			ctx.W.MarkLabel(lblG0)
			lblF1 := ctx.W.ReserveLabel()
			lblG1 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblF1)
			ctx.W.EmitCvtInt64ToFloat64(RegX1, b.Reg2)
			ctx.W.EmitJmp(lblG1)
			ctx.W.MarkLabel(lblF1)
			ctx.W.emitMovqGprToXmm(RegX1, b.Reg2)
			ctx.W.MarkLabel(lblG1)
			ctx.W.EmitSubFloat64(RegX0, RegX1)
			ctx.W.EmitMovRegImm64(result.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
			ctx.W.emitMovqXmmToGpr(result.Reg2, RegX0)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNilResult)
			ctx.W.emitXorReg(result.Reg)
			ctx.W.emitXorReg(result.Reg2)
			ctx.W.MarkLabel(lblDone)
			ctx.FreeDesc(&a)
			ctx.FreeDesc(&b)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"*", "multiplies two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit (SQL-style): if any arg is nil, result is nil
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			// Try integer mode: treat float operands with zero fractional part as integers
			prodInt := int64(1)
			i := 0
			for ; i < len(a); i++ {
				v := a[i]
				if v.IsInt() {
					prodInt *= v.Int()
					continue
				}
				if v.IsFloat() {
					f := v.Float()
					if f == math.Trunc(f) {
						prodInt *= int64(f)
						continue
					}
				}
				break // non-integer number encountered -> switch to float mode
			}
			if i == len(a) {
				return NewInt(prodInt)
			}
			// Float mode: include any prior integer product and continue in float
			prodFloat := float64(prodInt)
			for ; i < len(a); i++ {
				prodFloat *= a[i].Float()
			}
			return NewFloat(prodFloat)
		},
		true, false, &TypeDescriptor{Optimize: optimizeAssociative},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			allImm := true
			for _, a := range args {
				if a.Loc != LocImm {
					allImm = false
					break
				}
			}
			if allImm {
				for _, a := range args {
					if a.Imm.IsNil() {
						r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
						if result.Loc == LocAny {
							return r
						}
						ctx.EnsureResultRegPair(&result)
						ctx.W.EmitMakeNil(result)
						return result
					}
				}
				prod := int64(1)
				allInt := true
				for _, a := range args {
					if a.Imm.IsInt() {
						prod *= a.Imm.Int()
					} else {
						allInt = false
						break
					}
				}
				if allInt {
					r := JITValueDesc{Loc: LocImm, Imm: NewInt(prod)}
					if result.Loc == LocAny {
						return r
					}
					ctx.EnsureResultRegPair(&result)
					ctx.W.EmitMakeInt(result, r)
					return result
				}
				prodFloat := float64(1)
				for _, a := range args {
					prodFloat *= a.Imm.Float()
				}
				r := JITValueDesc{Loc: LocImm, Imm: NewFloat(prodFloat)}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeFloat(result, r)
				return result
			}
			if len(args) != 2 {
				return JITValueDesc{}
			}
			a, b := args[0], args[1]
			if a.Loc == LocImm && a.Imm.IsNil() {
				ctx.FreeDesc(&b)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			if b.Loc == LocImm && b.Imm.IsNil() {
				ctx.FreeDesc(&a)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			ctx.MaterializeToRegPair(&a)
			ctx.MaterializeToRegPair(&b)
			ctx.EnsureResultRegPair(&result)
			lblNotBothInt := ctx.W.ReserveLabel()
			lblNilResult := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			// Both int: IMUL
			ctx.W.EmitImulInt64(a.Reg2, b.Reg2)
			ctx.W.emitMovRegReg(result.Reg, RegR11)
			ctx.W.emitMovRegReg(result.Reg2, a.Reg2)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNotBothInt)
			// Nil checks
			ctx.W.EmitTestRegReg(a.Reg)
			lblA := ctx.W.ReserveLabel()
			ctx.W.EmitJcc(CcNE, lblA)
			ctx.W.EmitTestRegReg(a.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblA)
			ctx.W.EmitTestRegReg(b.Reg)
			lblB := ctx.W.ReserveLabel()
			ctx.W.EmitJcc(CcNE, lblB)
			ctx.W.EmitTestRegReg(b.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblB)
			// Float path
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			lblF0 := ctx.W.ReserveLabel()
			lblG0 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblF0)
			ctx.W.EmitCvtInt64ToFloat64(RegX0, a.Reg2)
			ctx.W.EmitJmp(lblG0)
			ctx.W.MarkLabel(lblF0)
			ctx.W.emitMovqGprToXmm(RegX0, a.Reg2)
			ctx.W.MarkLabel(lblG0)
			lblF1 := ctx.W.ReserveLabel()
			lblG1 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblF1)
			ctx.W.EmitCvtInt64ToFloat64(RegX1, b.Reg2)
			ctx.W.EmitJmp(lblG1)
			ctx.W.MarkLabel(lblF1)
			ctx.W.emitMovqGprToXmm(RegX1, b.Reg2)
			ctx.W.MarkLabel(lblG1)
			ctx.W.EmitMulFloat64(RegX0, RegX1)
			ctx.W.EmitMovRegImm64(result.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
			ctx.W.emitMovqXmmToGpr(result.Reg2, RegX0)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNilResult)
			ctx.W.emitXorReg(result.Reg)
			ctx.W.emitXorReg(result.Reg2)
			ctx.W.MarkLabel(lblDone)
			ctx.FreeDesc(&a)
			ctx.FreeDesc(&b)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"/", "divides two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			v := a[0].Float()
			for _, i := range a[1:] {
				v /= i.Float()
			}
			return NewFloat(v)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			allImm := true
			for _, a := range args {
				if a.Loc != LocImm {
					allImm = false
					break
				}
			}
			if allImm {
				for _, a := range args {
					if a.Imm.IsNil() {
						r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
						if result.Loc == LocAny {
							return r
						}
						ctx.EnsureResultRegPair(&result)
						ctx.W.EmitMakeNil(result)
						return result
					}
				}
				v := args[0].Imm.Float()
				for i := 1; i < len(args); i++ {
					v /= args[i].Imm.Float()
				}
				r := JITValueDesc{Loc: LocImm, Imm: NewFloat(v)}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeFloat(result, r)
				return result
			}
			if len(args) != 2 {
				return JITValueDesc{}
			}
			a, b := args[0], args[1]
			if (a.Loc == LocImm && a.Imm.IsNil()) || (b.Loc == LocImm && b.Imm.IsNil()) {
				ctx.FreeDesc(&a)
				ctx.FreeDesc(&b)
				r := JITValueDesc{Loc: LocImm, Imm: NewNil()}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeNil(result)
				return result
			}
			ctx.MaterializeToRegPair(&a)
			ctx.MaterializeToRegPair(&b)
			ctx.EnsureResultRegPair(&result)
			lblNilResult := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			// Nil checks
			ctx.W.EmitTestRegReg(a.Reg)
			lblA := ctx.W.ReserveLabel()
			ctx.W.EmitJcc(CcNE, lblA)
			ctx.W.EmitTestRegReg(a.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblA)
			ctx.W.EmitTestRegReg(b.Reg)
			lblB := ctx.W.ReserveLabel()
			ctx.W.EmitJcc(CcNE, lblB)
			ctx.W.EmitTestRegReg(b.Reg2)
			ctx.W.EmitJcc(CcE, lblNilResult)
			ctx.W.MarkLabel(lblB)
			// Always float: convert both to float64
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			lblF0 := ctx.W.ReserveLabel()
			lblG0 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblF0)
			ctx.W.EmitCvtInt64ToFloat64(RegX0, a.Reg2)
			ctx.W.EmitJmp(lblG0)
			ctx.W.MarkLabel(lblF0)
			ctx.W.emitMovqGprToXmm(RegX0, a.Reg2)
			ctx.W.MarkLabel(lblG0)
			lblF1 := ctx.W.ReserveLabel()
			lblG1 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblF1)
			ctx.W.EmitCvtInt64ToFloat64(RegX1, b.Reg2)
			ctx.W.EmitJmp(lblG1)
			ctx.W.MarkLabel(lblF1)
			ctx.W.emitMovqGprToXmm(RegX1, b.Reg2)
			ctx.W.MarkLabel(lblG1)
			ctx.W.EmitDivFloat64(RegX0, RegX1)
			ctx.W.EmitMovRegImm64(result.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
			ctx.W.emitMovqXmmToGpr(result.Reg2, RegX0)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNilResult)
			ctx.W.emitXorReg(result.Reg)
			ctx.W.emitXorReg(result.Reg2)
			ctx.W.MarkLabel(lblDone)
			ctx.FreeDesc(&a)
			ctx.FreeDesc(&b)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"<=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!Less(a[1], a[0]))
		},
		true, false, nil,
		jitEmitCompare(false, CcLE, CcAE),
	})
	Declare(&Globalenv, &Declaration{
		"<", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Less(a[0], a[1]))
		},
		true, false, nil,
		jitEmitCompare(false, CcL, CcB),
	})
	Declare(&Globalenv, &Declaration{
		">", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Less(a[1], a[0]))
		},
		true, false, nil,
		jitEmitCompare(true, CcL, CcB),
	})
	Declare(&Globalenv, &Declaration{
		">=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!Less(a[0], a[1]))
		},
		true, false, nil,
		jitEmitCompare(false, CcGE, CcAE),
	})
	Declare(&Globalenv, &Declaration{
		"equal?", "compares two values of the same type, (equal? nil nil) is true",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Equal(a[0], a[1]))
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			a, b := args[0], args[1]
			if a.Loc == LocImm && b.Loc == LocImm {
				r := JITValueDesc{Loc: LocImm, Imm: NewBool(Equal(a.Imm, b.Imm))}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeBool(result, r)
				return result
			}
			// Register path: compare both int fast, fallback to aux comparison
			ctx.MaterializeToRegPair(&a)
			ctx.MaterializeToRegPair(&b)
			ctx.EnsureResultRegPair(&result)
			boolReg := ctx.AllocReg()
			lblDone := ctx.W.ReserveLabel()
			lblNotSamePtr := ctx.W.ReserveLabel()
			// Fast: if both ptr and aux match, values are equal
			ctx.W.EmitCmpInt64(a.Reg, b.Reg)
			ctx.W.EmitJcc(CcNE, lblNotSamePtr)
			ctx.W.EmitCmpInt64(a.Reg2, b.Reg2)
			ctx.W.EmitSetcc(boolReg, CcE)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNotSamePtr)
			// Both int sentinel → compare aux (int values)
			lblNotBothInt := ctx.W.ReserveLabel()
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothInt)
			ctx.W.EmitCmpInt64(a.Reg2, b.Reg2)
			ctx.W.EmitSetcc(boolReg, CcE)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNotBothInt)
			// Both float sentinel → compare as float
			lblNotBothFloat := ctx.W.ReserveLabel()
			ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
			ctx.W.EmitCmpInt64(a.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothFloat)
			ctx.W.EmitCmpInt64(b.Reg, RegR11)
			ctx.W.EmitJcc(CcNE, lblNotBothFloat)
			ctx.W.emitMovqGprToXmm(RegX0, a.Reg2)
			ctx.W.emitMovqGprToXmm(RegX1, b.Reg2)
			ctx.W.EmitUcomisd(RegX0, RegX1)
			ctx.W.EmitSetcc(boolReg, CcE)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblNotBothFloat)
			// Mixed types or complex: conservative false (not perfectly correct but
			// handles the hot path of same-type numeric comparisons)
			ctx.W.emitXorReg(boolReg)
			ctx.W.MarkLabel(lblDone)
			ctx.FreeDesc(&a)
			ctx.FreeDesc(&b)
			src := JITValueDesc{Loc: LocReg, Reg: boolReg}
			ctx.W.EmitMakeBool(result, src)
			ctx.FreeReg(boolReg)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"equal??", "performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return EqualSQL(a[0], a[1])
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			a, b := args[0], args[1]
			if a.Loc == LocImm && b.Loc == LocImm {
				r := JITValueDesc{Loc: LocImm, Imm: EqualSQL(a.Imm, b.Imm)}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				tag := r.Imm.GetTag()
				if tag == tagNil {
					ctx.W.EmitMakeNil(result)
				} else {
					ctx.W.EmitMakeBool(result, r)
				}
				return result
			}
			// No register path — too complex (SQL semantics with nil propagation)
			return JITValueDesc{}
		},
	})
	Declare(&Globalenv, &Declaration{
		"equal_collate", "performs SQL equality with a specified collation (e.g. *_ci case-insensitive, *_bin case-sensitive); returns nil if either arg is nil",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"a", "any", "left side", nil},
			DeclarationParameter{"b", "any", "right side", nil},
			DeclarationParameter{"collation", "string", "collation name", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			coll := strings.ToLower(String(a[2]))
			ta := a[0].GetTag()
			tb := a[1].GetTag()
			if (ta == tagString || ta == tagSymbol) && (tb == tagString || tb == tagSymbol) {
				as := a[0].String()
				bs := a[1].String()
				if strings.Contains(coll, "_ci") {
					return NewBool(strings.EqualFold(as, bs))
				}
				return NewBool(as == bs)
			}
			return EqualSQL(a[0], a[1])
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).IsNil(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"notequal_collate", "performs SQL inequality with a specified collation; returns nil if either arg is nil",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"a", "any", "left side", nil},
			DeclarationParameter{"b", "any", "right side", nil},
			DeclarationParameter{"collation", "string", "collation name", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			r := Globalenv.Vars["equal_collate"].Func()(a[0], a[1], a[2])
			if r.IsNil() {
				return r
			}
			return NewBool(!r.Bool())
		},
		true, false, nil,
		nil /* TODO: FieldAddr: &Globalenv.Vars [#0] */,
	})
	Declare(&Globalenv, &Declaration{
		"!", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			a := args[0]
			if a.Loc == LocImm {
				r := JITValueDesc{Loc: LocImm, Imm: NewBool(!a.Imm.Bool())}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeBool(result, r)
				return result
			}
			// Register path: tag dispatch for !Bool()
			// tagNil → true, tagBool → flip low bit, else → false
			tagReg := ctx.AllocReg()
			ctx.W.EmitGetTag(tagReg, a.Reg, a.Reg2)
			boolReg := a.Reg // reuse ptr reg for result bool
			lblIsBool := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			// Check nil first
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagNil))
			ctx.FreeReg(tagReg)
			ctx.W.EmitSetcc(boolReg, CcE) // boolReg = 1 if nil (nil is falsy, !false = true)
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagBool))
			ctx.W.EmitJcc(CcE, lblIsBool)
			// For nil: boolReg already has the right value (1)
			// For non-nil non-bool: truthy → !true = false
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagNil))
			ctx.W.EmitSetcc(boolReg, CcE) // 1 if nil, 0 otherwise
			ctx.W.EmitJmp(lblDone)
			// Bool path: extract low bit of aux, flip it
			ctx.W.MarkLabel(lblIsBool)
			ctx.W.emitMovRegReg(boolReg, a.Reg2) // boolReg = aux
			ctx.W.emitAndRegImm32(boolReg, 1)     // isolate low bit
			ctx.W.EmitXorRegImm8(boolReg, 1)      // flip
			ctx.W.MarkLabel(lblDone)
			ctx.FreeReg(a.Reg2) // free aux reg
			src := JITValueDesc{Loc: LocReg, Reg: boolReg}
			ctx.EnsureResultRegPair(&result)
			ctx.W.EmitMakeBool(result, src)
			ctx.FreeReg(boolReg)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"not", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			a := args[0]
			if a.Loc == LocImm {
				r := JITValueDesc{Loc: LocImm, Imm: NewBool(!a.Imm.Bool())}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeBool(result, r)
				return result
			}
			tagReg := ctx.AllocReg()
			ctx.W.EmitGetTag(tagReg, a.Reg, a.Reg2)
			boolReg := a.Reg
			lblIsBool := ctx.W.ReserveLabel()
			lblDone := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagBool))
			ctx.W.EmitJcc(CcE, lblIsBool)
			ctx.FreeReg(tagReg)
			// nil → true, else → false
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagNil))
			ctx.W.EmitSetcc(boolReg, CcE)
			ctx.W.EmitJmp(lblDone)
			ctx.W.MarkLabel(lblIsBool)
			ctx.W.emitMovRegReg(boolReg, a.Reg2)
			ctx.W.emitAndRegImm32(boolReg, 1)
			ctx.W.EmitXorRegImm8(boolReg, 1)
			ctx.W.MarkLabel(lblDone)
			ctx.FreeReg(a.Reg2)
			src := JITValueDesc{Loc: LocReg, Reg: boolReg}
			ctx.EnsureResultRegPair(&result)
			ctx.W.EmitMakeBool(result, src)
			ctx.FreeReg(boolReg)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"nil?", "returns true if value is nil",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].IsNil())
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			a := args[0]
			if a.Loc == LocImm {
				r := JITValueDesc{Loc: LocImm, Imm: NewBool(a.Imm.IsNil())}
				if result.Loc == LocAny {
					return r
				}
				ctx.EnsureResultRegPair(&result)
				ctx.W.EmitMakeBool(result, r)
				return result
			}
			tagReg := ctx.AllocReg()
			ctx.W.EmitGetTag(tagReg, a.Reg, a.Reg2)
			ctx.FreeDesc(&a)
			ctx.W.EmitCmpRegImm32(tagReg, int32(tagNil))
			ctx.W.EmitSetcc(tagReg, CcE)
			src := JITValueDesc{Loc: LocReg, Reg: tagReg}
			ctx.EnsureResultRegPair(&result)
			ctx.W.EmitMakeBool(result, src)
			ctx.FreeReg(tagReg)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"min", "returns the smallest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value", nil},
		}, "number|string",
		func(a ...Scmer) Scmer {
			var result Scmer
			for _, v := range a {
				if result.IsNil() {
					result = v
				} else if !v.IsNil() && Less(v, result) {
					result = v
				}
			}
			return result
		},
		true, false, nil,
		nil /* TODO: dynamic call: len(a) */,
	})
	Declare(&Globalenv, &Declaration{
		"max", "returns the highest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value", nil},
		}, "number|string",
		func(a ...Scmer) Scmer {
			var result Scmer
			for _, v := range a {
				if result.IsNil() {
					result = v
				} else if !v.IsNil() && Less(result, v) {
					result = v
				}
			}
			return result
		},
		true, false, nil,
		nil /* TODO: dynamic call: len(a) */,
	})
	Declare(&Globalenv, &Declaration{
		"floor", "rounds the number down",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Floor(a[0].Float()))
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).Float(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"ceil", "rounds the number up",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Ceil(a[0].Float()))
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).Float(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"round", "rounds the number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Round(a[0].Float()))
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).Float(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"sql_abs", "SQL ABS(): returns absolute value, NULL-safe",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				v = -v
			}
			// preserve int type
			if ToInt(a[0]) == int(v) && a[0].Float() == v {
				return NewInt(int64(v))
			}
			return NewFloat(v)
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).IsNil(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"sqrt", "returns the square root of a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				return NewNil()
			}
			return NewFloat(math.Sqrt(v))
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).IsNil(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"sql_rand", "SQL RAND(): returns a random float in [0,1)",
		0, 0,
		[]DeclarationParameter{}, "number",
		func(a ...Scmer) Scmer {
			var buf [8]byte
			if _, err := crand.Read(buf[:]); err != nil {
				panic("sql_rand: " + err.Error())
			}
			// 53 random bits map exactly into float64 mantissa range.
			u := binary.LittleEndian.Uint64(buf[:]) >> 11
			return NewFloat(float64(u) / (1 << 53))
		},
		true, false, nil,
		nil /* TODO: Alloc: new [8]byte (buf) */,
	})
}
