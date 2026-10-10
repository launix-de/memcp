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
	"math/big"
	"strconv"
	"strings"
)

func sqlLiteralArithmetic(a, b Scmer, subtract bool) Scmer {
	left, leftOK := new(big.Rat).SetString(strconv.FormatFloat(a.Float(), 'g', -1, 64))
	right, rightOK := new(big.Rat).SetString(strconv.FormatFloat(b.Float(), 'g', -1, 64))
	if !leftOK || !rightOK {
		if subtract {
			return NewFloat(a.Float() - b.Float())
		}
		return NewFloat(a.Float() + b.Float())
	}
	if subtract {
		left.Sub(left, right)
	} else {
		left.Add(left, right)
	}
	result, _ := left.Float64()
	return NewFloat(result)
}

func roundSQLDecimalOutput(value float64, scale int) float64 {
	factor := math.Pow10(scale)
	scaled := value * factor
	if math.IsNaN(scaled) || math.IsInf(scaled, 0) {
		return value
	}

	// Exact DECIMAL half values can land one binary ULP to either side after
	// arithmetic. Snap only that representation error to the half boundary;
	// values farther away retain normal half-away-from-zero rounding.
	half := math.Round(scaled*2) / 2
	ulpUp := math.Abs(math.Nextafter(scaled, math.Inf(1)) - scaled)
	ulpDown := math.Abs(scaled - math.Nextafter(scaled, math.Inf(-1)))
	tolerance := 2 * math.Max(ulpUp, ulpDown)
	if math.Abs(scaled-half) <= tolerance {
		scaled = half
	}
	return math.Round(scaled) / factor
}

func init_alu() {
	// string functions
	DeclareTitle("Arithmetic / Logic")

	Declare(&Globalenv, &Declaration{
		Name: "int?",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagInt)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells if the value is a integer",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["int?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := d0
				d1.ID = 0
				d2 := ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d2)
				resultTarget3 := false
				_ = resultTarget3
				var d4 JITValueDesc
				if d2.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x4))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d2.Reg, 4)
					d4 = ctx.DeferBooleanFlags(r0, CondEqual)
					ctx.BindReg(r0, &d4)
				}
				ctx.FreeDesc(&d2)
				ctx.SyncDesc(&d4)
				if ctx.hasBooleanFlags(d4) {
					return d4
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d4.Loc == LocImm {
					ctx.EmitMakeBool(result, d4)
				} else {
					ctx.EmitMakeBool(result, d4)
					if !resultTarget3 {
						ctx.FreeReg(d4.Reg)
					}
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "number?",

		Fn: func(a ...Scmer) Scmer {
			tag := a[0].GetTag()
			return NewBool(tag == tagFloat || tag == tagInt || tag == tagDate)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells if the value is a number",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["number?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [4]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d3 = d2
					d3.ID = 0
					d4 = ctx.EmitGetTagDesc(&d3, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x3))}
					} else {
						r0 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpRegImm32(d4.Reg, 3)
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d5)
					}
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					lbl5 := ctx.ReserveLabel()
					ctx.EmitJump(d6.Condition, lbl5)
					ctx.EmitJmp(lbl4)
					ctx.FreeDesc(&d5)
					snap7 := d1
					snap8 := d2
					snap9 := d3
					snap10 := d4
					snap11 := d5
					snap12 := d6
					alloc13 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl5)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc13)
					d1 = snap7
					d2 = snap8
					d3 = snap9
					d4 = snap10
					d5 = snap11
					d6 = snap12
					if !bbs[2].Rendered {
						snap14 := d1
						snap15 := d2
						snap16 := d3
						snap17 := d4
						snap18 := d5
						snap19 := d6
						alloc20 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc20)
						d1 = snap14
						d2 = snap15
						d3 = snap16
						d4 = snap17
						d5 = snap18
						d6 = snap19
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x10))}
					} else {
						ctx.EmitCmpRegImm32(d4.Reg, 16)
						r1 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitSetcc(r1, CondEqual)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d21)
					}
					ctx.EnsureDesc(&d21)
					ctx.EmitStoreToStack(d21, int32(bbs[2].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d21)
					return bbs[2].Render()
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d1)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1) {
						return d1
					}
					if d1.Loc == LocImm {
						ctx.EmitMakeBool(result, d1)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d22)
						if d1.Loc == LocReg && d1.Reg != result.Reg2 {
							ctx.FreeReg(d1.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x4))}
					} else {
						r2 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpRegImm32(d4.Reg, 4)
						d23 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d23)
					}
					d24 = d23
					ctx.EnsureDesc(&d24)
					if d24.Loc != LocImm && d24.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d24.Loc == LocImm {
						if d24.Imm.Bool() {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
							return bbs[2].Render()
						}
						return bbs[1].Render()
					}
					lbl6 := ctx.ReserveLabel()
					ctx.EmitJump(d24.Condition, lbl6)
					ctx.EmitJmp(lbl2)
					ctx.FreeDesc(&d23)
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d4
					snap29 := d5
					snap30 := d6
					snap31 := d21
					snap32 := d22
					snap33 := d23
					snap34 := d24
					alloc35 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl6)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc35)
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d6 = snap30
					d21 = snap31
					d22 = snap32
					d23 = snap33
					d24 = snap34
					if !bbs[2].Rendered {
						snap36 := d1
						snap37 := d2
						snap38 := d3
						snap39 := d4
						snap40 := d5
						snap41 := d6
						snap42 := d21
						snap43 := d22
						snap44 := d23
						snap45 := d24
						alloc46 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc46)
						d1 = snap36
						d2 = snap37
						d3 = snap38
						d4 = snap39
						d5 = snap40
						d6 = snap41
						d21 = snap42
						d22 = snap43
						d23 = snap44
						d24 = snap45
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 12,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "symbol?",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagSymbol)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells if the value is a symbol",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["symbol?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := d0
				d1.ID = 0
				d2 := ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d2)
				resultTarget3 := false
				_ = resultTarget3
				var d4 JITValueDesc
				if d2.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x2))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d2.Reg, 2)
					d4 = ctx.DeferBooleanFlags(r0, CondEqual)
					ctx.BindReg(r0, &d4)
				}
				ctx.FreeDesc(&d2)
				ctx.SyncDesc(&d4)
				if ctx.hasBooleanFlags(d4) {
					return d4
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d4.Loc == LocImm {
					ctx.EmitMakeBool(result, d4)
				} else {
					ctx.EmitMakeBool(result, d4)
					if !resultTarget3 {
						ctx.FreeReg(d4.Reg)
					}
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "+",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "adds two or more numbers",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value...", Description: "values to add", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["+"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d5 JITValueDesc
				_ = d5
				var dynamicArgOff6 int32
				var dynamicArgOff7 int32
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var dynamicArgOff61 int32
				var dynamicArgOff62 int32
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(64))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [12]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				bbs[9].PhiBase = int32(phiBase0) + int32(32)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				d4 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
				_ = d4
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(16))
					return bbs[3].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						idx := int(d2.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d5 = args[idx]
						d5.ID = 0
					} else {
						ctx.EnsureDesc(&d2)
						dynamicArgOff6 = ctx.AllocStack(16)
						ctx.ProtectReg(d2.Reg)
						lbl13 := ctx.ReserveLabel()
						lbl14 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d2.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl14)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d2.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff6))
							ctx.EmitJmp(lbl13)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff6))
						ctx.MarkLabel(lbl13)
						ctx.UnprotectReg(d2.Reg)
						d5 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff6), Rooted: true}
					}
					dynamicArgOff7 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d5, int32(dynamicArgOff7))
					ctx.FreeDesc(&d5)
					d5 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff7), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d5)
					d9 = d5
					d9.ID = 0
					d8 = ctx.EmitTagEqualsBorrowed(&d9, tagInt, JITValueDesc{Loc: LocAny})
					d10 = d8
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d1
						snap12 := d2
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d8
						snap17 := d9
						snap18 := d10
						alloc19 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc19)
						d1 = snap11
						d2 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d8 = snap16
						d9 = snap17
						d10 = snap18
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d8)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDescsTogether(&d2, &d20)
					if d2.Loc == LocImm && d20.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == d20.Imm.Int())}
					} else if d20.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d2.Reg)
						if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d20.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d20.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, ctx.ScratchReg)
						}
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d21)
					} else if d2.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d20.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d21)
					} else {
						r2 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d20.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d21)
					}
					ctx.FreeDesc(&d20)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d22.Condition, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d21)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap23 := d1
						snap24 := d2
						snap25 := d3
						snap26 := d4
						snap27 := d5
						snap28 := d8
						snap29 := d9
						snap30 := d10
						snap31 := d20
						snap32 := d21
						snap33 := d22
						alloc34 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc34)
						d1 = snap23
						d2 = snap24
						d3 = snap25
						d4 = snap26
						d5 = snap27
						d8 = snap28
						d9 = snap29
						d10 = snap30
						d20 = snap31
						d21 = snap32
						d22 = snap33
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDescsTogether(&d2, &d35)
					if d2.Loc == LocImm && d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d35.Imm.Int())}
					} else if d35.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d2.Reg)
						if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d35.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d35.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, ctx.ScratchReg)
						}
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d36)
					} else if d2.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d35.Reg)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d36)
					} else {
						r5 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d35.Reg)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d36)
					}
					ctx.FreeDesc(&d35)
					d37 = d36
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocImm && d37.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d37.Loc == LocImm {
						if d37.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d37.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d36)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap38 := d1
						snap39 := d2
						snap40 := d3
						snap41 := d4
						snap42 := d5
						snap43 := d8
						snap44 := d9
						snap45 := d10
						snap46 := d20
						snap47 := d21
						snap48 := d22
						snap49 := d35
						snap50 := d36
						snap51 := d37
						alloc52 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc52)
						d1 = snap38
						d2 = snap39
						d3 = snap40
						d4 = snap41
						d5 = snap42
						d8 = snap43
						d9 = snap44
						d10 = snap45
						d20 = snap46
						d21 = snap47
						d22 = snap48
						d35 = snap49
						d36 = snap50
						d37 = snap51
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					if d5.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int())}
					} else if d5.Type == tagInt && d5.Loc == LocRegPair {
						ctx.FreeReg(d5.Reg)
						d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2}
						ctx.BindReg(d5.Reg2, &d53)
						ctx.BindReg(d5.Reg2, &d53)
					} else if d5.Type == tagInt && d5.Loc == LocReg {
						d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
						ctx.BindReg(d5.Reg, &d53)
						ctx.BindReg(d5.Reg, &d53)
					} else {
						d53 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d5}, 1)
						d53.Type = tagInt
						ctx.BindReg(d53.Reg, &d53)
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d53)
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d53)
					if d1.Loc == LocImm && d53.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d53.Imm.Int())}
					} else if d53.Loc == LocImm && d53.Imm.Int() == 0 {
						ctx.EnsureDesc(&d1)
						r6 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r6, d1.Reg)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d54)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						ctx.EnsureDesc(&d53)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg}
						ctx.BindReg(d53.Reg, &d54)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d53)
						scratch := ctx.AllocRegExcept(d53.Reg)
						ctx.EmitMovRegReg(scratch, d53.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d1.Imm.Int())
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d54)
					} else if d53.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d53.Imm.Int())
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d54)
					} else {
						ctx.EnsureDesc(&d1)
						ctx.SyncDesc(&d53)
						r7 := ctx.AllocRegExceptOperand(&d53, d1.Reg)
						ctx.EmitMovRegReg(r7, d1.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r7, &d53)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d54)
					}
					if d54.Loc == LocReg && d1.Loc == LocReg && d54.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d54)
					ctx.EmitStoreToStack(d54, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d54)
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d55)
					}
					if d55.Loc == LocReg && d2.Loc == LocReg && d55.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d55)
					ctx.EmitStoreToStack(d55, int32(bbs[3].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d55)
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						ctx.EmitMakeInt(result, d1)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1)
						d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d56)
						if d1.Loc == LocReg && d1.Reg != result.Reg2 {
							ctx.FreeReg(d1.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d1.Imm.Int()))}
					} else {
						var r8 Reg
						r8 = ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r8, d1.Reg)
						ctx.EmitInt64ToFloatBits(r8)
						d57 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
						ctx.BindReg(r8, &d57)
					}
					ctx.StabilizeDescForControlFlow(&d57)
					ctx.SyncDesc(&d2)
					if d2.Loc == LocReg || d2.Loc == LocFPReg {
						ctx.ProtectReg(d2.Reg)
					} else if d2.Loc == LocRegPair {
						ctx.ProtectReg(d2.Reg)
						ctx.ProtectReg(d2.Reg2)
					}
					ctx.SyncDesc(&d57)
					if d57.Loc == LocReg || d57.Loc == LocFPReg {
						ctx.ProtectReg(d57.Reg)
					} else if d57.Loc == LocRegPair {
						ctx.ProtectReg(d57.Reg)
						ctx.ProtectReg(d57.Reg2)
					}
					d58 = d2
					if d58.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d58)
					ctx.EmitStoreToStack(d58, int32(bbs[9].PhiBase)+int32(0))
					d59 = d57
					if d59.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(bbs[9].PhiBase)+int32(16))
					if d2.Loc == LocReg || d2.Loc == LocFPReg {
						ctx.UnprotectReg(d2.Reg)
					} else if d2.Loc == LocRegPair {
						ctx.UnprotectReg(d2.Reg)
						ctx.UnprotectReg(d2.Reg2)
					}
					if d57.Loc == LocReg || d57.Loc == LocFPReg {
						ctx.UnprotectReg(d57.Reg)
					} else if d57.Loc == LocRegPair {
						ctx.UnprotectReg(d57.Reg)
						ctx.UnprotectReg(d57.Reg2)
					}
					return bbs[9].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						idx := int(d3.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d60 = args[idx]
						d60.ID = 0
					} else {
						ctx.EnsureDesc(&d3)
						dynamicArgOff61 = ctx.AllocStack(16)
						ctx.ProtectReg(d3.Reg)
						lbl15 := ctx.ReserveLabel()
						lbl16 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d3.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl16)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d3.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff61))
							ctx.EmitJmp(lbl15)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl16)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff61))
						ctx.MarkLabel(lbl15)
						ctx.UnprotectReg(d3.Reg)
						d60 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff61), Rooted: true}
					}
					dynamicArgOff62 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d60, int32(dynamicArgOff62))
					ctx.FreeDesc(&d60)
					d60 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff62), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d60)
					d64 = d60
					d64.ID = 0
					d63 = ctx.EmitTagEqualsBorrowed(&d64, tagNil, JITValueDesc{Loc: LocAny})
					d65 = d63
					ctx.EnsureDesc(&d65)
					if d65.Loc != LocImm && d65.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d65.Loc == LocImm {
						if d65.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitCmpRegImm32(d65.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap66 := d1
						snap67 := d2
						snap68 := d3
						snap69 := d4
						snap70 := d5
						snap71 := d8
						snap72 := d9
						snap73 := d10
						snap74 := d20
						snap75 := d21
						snap76 := d22
						snap77 := d35
						snap78 := d36
						snap79 := d37
						snap80 := d53
						snap81 := d54
						snap82 := d55
						snap83 := d56
						snap84 := d57
						snap85 := d58
						snap86 := d59
						snap87 := d60
						snap88 := d63
						snap89 := d64
						snap90 := d65
						alloc91 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc91)
						d1 = snap66
						d2 = snap67
						d3 = snap68
						d4 = snap69
						d5 = snap70
						d8 = snap71
						d9 = snap72
						d10 = snap73
						d20 = snap74
						d21 = snap75
						d22 = snap76
						d35 = snap77
						d36 = snap78
						d37 = snap79
						d53 = snap80
						d54 = snap81
						d55 = snap82
						d56 = snap83
						d57 = snap84
						d58 = snap85
						d59 = snap86
						d60 = snap87
						d63 = snap88
						d64 = snap89
						d65 = snap90
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					ctx.FreeDesc(&d63)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						ctx.EmitMakeFloat(result, d4)
					} else {
						ctx.EmitMovToReg(result.Reg2, d4)
						d92 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d92)
						if d4.Loc == LocReg && d4.Reg != result.Reg2 {
							ctx.FreeReg(d4.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.StabilizeDescForControlFlow(&d4)
					d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d93)
					ctx.EnsureDescsTogether(&d3, &d93)
					if d3.Loc == LocImm && d93.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d93.Imm.Int())}
					} else if d93.Loc == LocImm {
						r9 := ctx.AllocRegExcept(d3.Reg)
						if d93.Imm.Int() >= -2147483648 && d93.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d93.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d93.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d94 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
						ctx.BindReg(r9, &d94)
					} else if d3.Loc == LocImm {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d93.Reg)
						d94 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
						ctx.BindReg(r10, &d94)
					} else {
						r11 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d93.Reg)
						d94 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
						ctx.BindReg(r11, &d94)
					}
					ctx.FreeDesc(&d93)
					d95 = d94
					ctx.EnsureDesc(&d95)
					if d95.Loc != LocImm && d95.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d95.Loc == LocImm {
						if d95.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d95.Condition, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d94)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap96 := d1
						snap97 := d2
						snap98 := d3
						snap99 := d4
						snap100 := d5
						snap101 := d8
						snap102 := d9
						snap103 := d10
						snap104 := d20
						snap105 := d21
						snap106 := d22
						snap107 := d35
						snap108 := d36
						snap109 := d37
						snap110 := d53
						snap111 := d54
						snap112 := d55
						snap113 := d56
						snap114 := d57
						snap115 := d58
						snap116 := d59
						snap117 := d60
						snap118 := d63
						snap119 := d64
						snap120 := d65
						snap121 := d92
						snap122 := d93
						snap123 := d94
						snap124 := d95
						alloc125 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc125)
						d1 = snap96
						d2 = snap97
						d3 = snap98
						d4 = snap99
						d5 = snap100
						d8 = snap101
						d9 = snap102
						d10 = snap103
						d20 = snap104
						d21 = snap105
						d22 = snap106
						d35 = snap107
						d36 = snap108
						d37 = snap109
						d53 = snap110
						d54 = snap111
						d55 = snap112
						d56 = snap113
						d57 = snap114
						d58 = snap115
						d59 = snap116
						d60 = snap117
						d63 = snap118
						d64 = snap119
						d65 = snap120
						d92 = snap121
						d93 = snap122
						d94 = snap123
						d95 = snap124
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d126 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d126)
					if d126.Loc == LocRegPair || d126.Loc == LocStackPair || d126.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d126, &result)
						result.Type = d126.Type
					} else {
						switch d126.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d126)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d126)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d126)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d126, &result)
							result.Type = d126.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d60)
					d127 = ctx.EmitFloatDesc(d60)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d127)
					ctx.EnsureDescsTogether(&d4, &d127)
					if d4.Loc == LocImm && d127.Loc == LocImm {
						d128 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() + d127.Imm.Float())}
					} else if d4.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d127.Reg)
						_, xBits := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d127.Reg)
						d128 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d128)
					} else if d127.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						_, yBits := d127.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitAddFloat64(scratch, ctx.ScratchReg)
						d128 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d128)
					} else {
						r12 := ctx.AllocRegExcept(d4.Reg, d127.Reg)
						ctx.EmitMovRegReg(r12, d4.Reg)
						ctx.EmitAddFloat64(r12, d127.Reg)
						d128 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d128)
					}
					if d128.Loc == LocReg && d4.Loc == LocReg && d128.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d128)
					ctx.EmitStoreToStack(d128, int32(bbs[9].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d128)
					ctx.FreeDesc(&d127)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d129)
					}
					if d129.Loc == LocReg && d3.Loc == LocReg && d129.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d129)
					ctx.EmitStoreToStack(d129, int32(bbs[9].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d129)
					return bbs[9].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 38,
		},
		Optimize: optimizeAssociative,
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_sum_reduce",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return a[1]
			}
			if a[1].IsNil() {
				return a[0]
			}
			if a[0].IsInt() && a[1].IsInt() {
				return NewInt(a[0].Int() + a[1].Int())
			}
			return NewFloat(a[0].Float() + a[1].Float())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "adds two SQL SUM values while treating NULL as the empty aggregate identity",
			Params: []*TypeDescriptor{
				{Kind: "number|nil", Label: "left", Description: "partial sum"},
				{Kind: "number|nil", Label: "right", Description: "next value"},
			},
			Return: &TypeDescriptor{Kind: "number|nil"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_sum_reduce"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [8]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = args[1]
					d9.ID = 0
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					d12 = d10
					d12.ID = 0
					d11 = ctx.EmitTagEqualsBorrowed(&d12, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d10)
					d13 = d11
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d13.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap14 := d0
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d9
						snap19 := d10
						snap20 := d11
						snap21 := d12
						snap22 := d13
						alloc23 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc23)
						d0 = snap14
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d9 = snap18
						d10 = snap19
						d11 = snap20
						d12 = snap21
						d13 = snap22
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d24 = args[0]
					d24.ID = 0
					ctx.SyncDesc(&d24)
					if d24.Loc == LocRegPair || d24.Loc == LocStackPair || d24.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d24, &result)
						result.Type = d24.Type
					} else {
						switch d24.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d24)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d24)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d24)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d24, &result)
							result.Type = d24.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d25 = args[0]
					d25.ID = 0
					d27 = d25
					d27.ID = 0
					d26 = ctx.EmitTagEqualsBorrowed(&d27, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d25)
					d28 = d26
					ctx.EnsureDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d28.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap29 := d0
						snap30 := d1
						snap31 := d2
						snap32 := d3
						snap33 := d9
						snap34 := d10
						snap35 := d11
						snap36 := d12
						snap37 := d13
						snap38 := d24
						snap39 := d25
						snap40 := d26
						snap41 := d27
						snap42 := d28
						alloc43 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc43)
						d0 = snap29
						d1 = snap30
						d2 = snap31
						d3 = snap32
						d9 = snap33
						d10 = snap34
						d11 = snap35
						d12 = snap36
						d13 = snap37
						d24 = snap38
						d25 = snap39
						d26 = snap40
						d27 = snap41
						d28 = snap42
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d26)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d44 = args[0]
					d44.ID = 0
					if d44.Loc == LocImm {
						d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int())}
					} else if d44.Type == tagInt && d44.Loc == LocRegPair {
						ctx.FreeReg(d44.Reg)
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2}
						ctx.BindReg(d44.Reg2, &d45)
						ctx.BindReg(d44.Reg2, &d45)
					} else if d44.Type == tagInt && d44.Loc == LocReg {
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg}
						ctx.BindReg(d44.Reg, &d45)
						ctx.BindReg(d44.Reg, &d45)
					} else {
						d45 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d44}, 1)
						d45.Type = tagInt
						ctx.BindReg(d45.Reg, &d45)
					}
					ctx.FreeDesc(&d44)
					d46 = args[1]
					d46.ID = 0
					if d46.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d46.Imm.Int())}
					} else if d46.Type == tagInt && d46.Loc == LocRegPair {
						ctx.FreeReg(d46.Reg)
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2}
						ctx.BindReg(d46.Reg2, &d47)
						ctx.BindReg(d46.Reg2, &d47)
					} else if d46.Type == tagInt && d46.Loc == LocReg {
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg}
						ctx.BindReg(d46.Reg, &d47)
						ctx.BindReg(d46.Reg, &d47)
					} else {
						d47 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d46}, 1)
						d47.Type = tagInt
						ctx.BindReg(d47.Reg, &d47)
					}
					ctx.FreeDesc(&d46)
					ctx.EnsureDesc(&d45)
					resultTarget48 := false
					_ = resultTarget48
					ctx.EnsureDesc(&d47)
					ctx.SyncDesc(&d45)
					ctx.SyncDesc(&d47)
					if d45.Loc == LocImm && d47.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d45.Imm.Int() + d47.Imm.Int())}
					} else if d47.Loc == LocImm && d47.Imm.Int() == 0 {
						ctx.EnsureDesc(&d45)
						var r0 Reg
						if result.Loc == LocRegPair && result.Reg2 != d45.Reg {
							r0 = result.Reg2
							resultTarget48 = true
						} else {
							r0 = ctx.AllocRegExcept(d45.Reg)
						}
						ctx.EmitMovRegReg(r0, d45.Reg)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d49)
					} else if d45.Loc == LocImm && d45.Imm.Int() == 0 {
						ctx.EnsureDesc(&d47)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d49)
					} else if d45.Loc == LocImm {
						ctx.EnsureDesc(&d47)
						var scratch Reg
						if result.Loc == LocRegPair && result.Reg2 != d47.Reg {
							scratch = result.Reg2
							resultTarget48 = true
						} else {
							scratch = ctx.AllocRegExcept(d47.Reg)
						}
						ctx.EmitMovRegReg(scratch, d47.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d45.Imm.Int())
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					} else if d47.Loc == LocImm {
						ctx.EnsureDesc(&d45)
						var scratch Reg
						if result.Loc == LocRegPair && result.Reg2 != d45.Reg {
							scratch = result.Reg2
							resultTarget48 = true
						} else {
							scratch = ctx.AllocRegExcept(d45.Reg)
						}
						ctx.EmitMovRegReg(scratch, d45.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d47.Imm.Int())
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					} else {
						ctx.EnsureDesc(&d45)
						ctx.SyncDesc(&d47)
						var r1 Reg
						if result.Loc == LocRegPair && (d47.Loc != LocReg || result.Reg2 != d47.Reg) && result.Reg2 != d45.Reg {
							r1 = result.Reg2
							resultTarget48 = true
						} else {
							r1 = ctx.AllocRegExceptOperand(&d47, d45.Reg)
						}
						ctx.EmitMovRegReg(r1, d45.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r1, &d47)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d49)
					}
					if d49.Loc == LocReg && d45.Loc == LocReg && d49.Reg == d45.Reg {
						ctx.TransferReg(d45.Reg)
						d45.Loc = LocNone
					}
					if resultTarget48 && d49.Loc == LocReg {
						ctx.BindReg(result.Reg2, &result)
					}
					ctx.FreeDesc(&d45)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d49)
					if d49.Loc == LocImm {
						ctx.EmitMakeInt(result, d49)
					} else {
						ctx.EmitMovToReg(result.Reg2, d49)
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d50)
						if d49.Loc == LocReg && d49.Reg != result.Reg2 {
							ctx.FreeReg(d49.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d51 = args[0]
					d51.ID = 0
					d52 = ctx.EmitFloatDesc(d51)
					ctx.FreeDesc(&d51)
					d53 = args[1]
					d53.ID = 0
					d54 = ctx.EmitFloatDesc(d53)
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d52)
					resultTarget55 := false
					_ = resultTarget55
					ctx.EnsureDesc(&d54)
					ctx.EnsureDescsTogether(&d52, &d54)
					if d52.Loc == LocImm && d54.Loc == LocImm {
						d56 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d52.Imm.Float() + d54.Imm.Float())}
					} else if d52.Loc == LocImm {
						var scratch Reg
						if result.Loc == LocRegPair && result.Reg2 != d54.Reg {
							scratch = result.Reg2
							resultTarget55 = true
						} else {
							scratch = ctx.AllocRegExcept(d54.Reg)
						}
						_, xBits := d52.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d54.Reg)
						d56 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d56)
					} else if d54.Loc == LocImm {
						var scratch Reg
						if result.Loc == LocRegPair && result.Reg2 != d52.Reg {
							scratch = result.Reg2
							resultTarget55 = true
						} else {
							scratch = ctx.AllocRegExcept(d52.Reg)
						}
						ctx.EmitMovRegReg(scratch, d52.Reg)
						_, yBits := d54.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitAddFloat64(scratch, ctx.ScratchReg)
						d56 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d56)
					} else {
						var r2 Reg
						if result.Loc == LocRegPair && result.Reg2 != d52.Reg && result.Reg2 != d54.Reg {
							r2 = result.Reg2
							resultTarget55 = true
						} else {
							r2 = ctx.AllocRegExcept(d52.Reg, d54.Reg)
						}
						ctx.EmitMovRegReg(r2, d52.Reg)
						ctx.EmitAddFloat64(r2, d54.Reg)
						d56 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d56)
					}
					if d56.Loc == LocReg && d52.Loc == LocReg && d56.Reg == d52.Reg {
						ctx.TransferReg(d52.Reg)
						d52.Loc = LocNone
					}
					if resultTarget55 && d56.Loc == LocReg {
						ctx.BindReg(result.Reg2, &result)
					}
					ctx.FreeDesc(&d52)
					ctx.FreeDesc(&d54)
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocImm {
						ctx.EmitMakeFloat(result, d56)
					} else {
						ctx.EmitMovToReg(result.Reg2, d56)
						d57 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d57)
						if d56.Loc == LocReg && d56.Reg != result.Reg2 {
							ctx.FreeReg(d56.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d58 = args[1]
					d58.ID = 0
					d60 = d58
					d60.ID = 0
					d59 = ctx.EmitTagEqualsBorrowed(&d60, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d58)
					d61 = d59
					ctx.EnsureDesc(&d61)
					if d61.Loc != LocImm && d61.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d61.Loc == LocImm {
						if d61.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d61.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap62 := d0
						snap63 := d1
						snap64 := d2
						snap65 := d3
						snap66 := d9
						snap67 := d10
						snap68 := d11
						snap69 := d12
						snap70 := d13
						snap71 := d24
						snap72 := d25
						snap73 := d26
						snap74 := d27
						snap75 := d28
						snap76 := d44
						snap77 := d45
						snap78 := d46
						snap79 := d47
						snap80 := d49
						snap81 := d50
						snap82 := d51
						snap83 := d52
						snap84 := d53
						snap85 := d54
						snap86 := d56
						snap87 := d57
						snap88 := d58
						snap89 := d59
						snap90 := d60
						snap91 := d61
						alloc92 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc92)
						d0 = snap62
						d1 = snap63
						d2 = snap64
						d3 = snap65
						d9 = snap66
						d10 = snap67
						d11 = snap68
						d12 = snap69
						d13 = snap70
						d24 = snap71
						d25 = snap72
						d26 = snap73
						d27 = snap74
						d28 = snap75
						d44 = snap76
						d45 = snap77
						d46 = snap78
						d47 = snap79
						d49 = snap80
						d50 = snap81
						d51 = snap82
						d52 = snap83
						d53 = snap84
						d54 = snap85
						d56 = snap86
						d57 = snap87
						d58 = snap88
						d59 = snap89
						d60 = snap90
						d61 = snap91
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d59)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 40,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "-",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "subtracts two or more numbers from the first one",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value...", Description: "values", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["-"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d24 JITValueDesc
				_ = d24
				var dynamicArgOff25 int32
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d47 JITValueDesc
				_ = d47
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d99 JITValueDesc
				_ = d99
				var dynamicArgOff100 int32
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d185 JITValueDesc
				_ = d185
				var dynamicArgOff186 int32
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var d233 JITValueDesc
				_ = d233
				var d234 JITValueDesc
				_ = d234
				var d235 JITValueDesc
				_ = d235
				var d236 JITValueDesc
				_ = d236
				var d237 JITValueDesc
				_ = d237
				var dynamicArgOff238 int32
				var d239 JITValueDesc
				_ = d239
				var d240 JITValueDesc
				_ = d240
				var d241 JITValueDesc
				_ = d241
				var d242 JITValueDesc
				_ = d242
				var d243 JITValueDesc
				_ = d243
				var d244 JITValueDesc
				_ = d244
				var d245 JITValueDesc
				_ = d245
				var d301 JITValueDesc
				_ = d301
				var d302 JITValueDesc
				_ = d302
				var d303 JITValueDesc
				_ = d303
				var d362 JITValueDesc
				_ = d362
				var dynamicArgOff363 int32
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(112))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [19]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[9].PhiBase = int32(phiBase0) + int32(16)
				bbs[15].PhiBase = int32(phiBase0) + int32(48)
				bbs[16].PhiBase = int32(phiBase0) + int32(80)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				d4 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
				_ = d4
				d5 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
				_ = d5
				d6 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
				_ = d6
				d7 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
				_ = d7
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_0_14 := int32(-1)
				_ = bbpos_0_14
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_0_15 := int32(-1)
				_ = bbpos_0_15
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_0_16 := int32(-1)
				_ = bbpos_0_16
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_0_17 := int32(-1)
				_ = bbpos_0_17
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_0_18 := int32(-1)
				_ = bbpos_0_18
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d9)
					}
					if d9.Loc == LocReg && d1.Loc == LocReg && d9.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDescsTogether(&d9, &d8)
					if d9.Loc == LocImm && d8.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < d8.Imm.Int())}
					} else if d8.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d9.Reg)
						if d8.Imm.Int() >= -2147483648 && d8.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d9.Reg, int32(d8.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d8.Imm.Int()))
							ctx.EmitCmpInt64(d9.Reg, ctx.ScratchReg)
						}
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d10)
					} else if d9.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d9.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d8.Reg)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d10)
					} else {
						r2 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitCmpInt64(d9.Reg, d8.Reg)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d10)
					}
					d11 = d10
					ctx.EnsureDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d11.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d10)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap12 := d1
						snap13 := d2
						snap14 := d3
						snap15 := d4
						snap16 := d5
						snap17 := d6
						snap18 := d7
						snap19 := d8
						snap20 := d9
						snap21 := d10
						snap22 := d11
						alloc23 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc23)
						d1 = snap12
						d2 = snap13
						d3 = snap14
						d4 = snap15
						d5 = snap16
						d6 = snap17
						d7 = snap18
						d8 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						idx := int(d9.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d24 = args[idx]
						d24.ID = 0
					} else {
						ctx.EnsureDesc(&d9)
						dynamicArgOff25 = ctx.AllocStack(16)
						ctx.ProtectReg(d9.Reg)
						lbl20 := ctx.ReserveLabel()
						lbl21 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d9.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl21)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d9.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff25))
							ctx.EmitJmp(lbl20)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl21)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff25))
						ctx.MarkLabel(lbl20)
						ctx.UnprotectReg(d9.Reg)
						d24 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff25), Rooted: true}
					}
					d27 = d24
					d27.ID = 0
					d26 = ctx.EmitTagEqualsBorrowed(&d27, tagNil, JITValueDesc{Loc: LocAny})
					d28 = d26
					ctx.EnsureDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							return bbs[4].Render()
						}
						ctx.SyncDesc(&d9)
						if d9.Loc == LocReg || d9.Loc == LocFPReg {
							ctx.ProtectReg(d9.Reg)
						} else if d9.Loc == LocRegPair {
							ctx.ProtectReg(d9.Reg)
							ctx.ProtectReg(d9.Reg2)
						}
						d29 = d9
						if d29.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d29)
						ctx.EmitStoreToStack(d29, int32(bbs[1].PhiBase)+int32(0))
						if d9.Loc == LocReg || d9.Loc == LocFPReg {
							ctx.UnprotectReg(d9.Reg)
						} else if d9.Loc == LocRegPair {
							ctx.UnprotectReg(d9.Reg)
							ctx.UnprotectReg(d9.Reg2)
						}
						return bbs[1].Render()
					}
					lbl22 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d28.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					ctx.EmitJmp(lbl22)
					snap30 := d1
					snap31 := d2
					snap32 := d3
					snap33 := d4
					snap34 := d5
					snap35 := d6
					snap36 := d7
					snap37 := d8
					snap38 := d9
					snap39 := d10
					snap40 := d11
					snap41 := d24
					snap42 := d26
					snap43 := d27
					snap44 := d28
					snap45 := d29
					alloc46 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl22)
					ctx.SyncDesc(&d9)
					if d9.Loc == LocReg || d9.Loc == LocFPReg {
						ctx.ProtectReg(d9.Reg)
					} else if d9.Loc == LocRegPair {
						ctx.ProtectReg(d9.Reg)
						ctx.ProtectReg(d9.Reg2)
					}
					d47 = d9
					if d47.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d47)
					ctx.EmitStoreToStack(d47, int32(bbs[1].PhiBase)+int32(0))
					if d9.Loc == LocReg || d9.Loc == LocFPReg {
						ctx.UnprotectReg(d9.Reg)
					} else if d9.Loc == LocRegPair {
						ctx.UnprotectReg(d9.Reg)
						ctx.UnprotectReg(d9.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc46)
					d1 = snap30
					d2 = snap31
					d3 = snap32
					d4 = snap33
					d5 = snap34
					d6 = snap35
					d7 = snap36
					d8 = snap37
					d9 = snap38
					d10 = snap39
					d11 = snap40
					d24 = snap41
					d26 = snap42
					d27 = snap43
					d28 = snap44
					d29 = snap45
					if !bbs[1].Rendered {
						snap48 := d1
						snap49 := d2
						snap50 := d3
						snap51 := d4
						snap52 := d5
						snap53 := d6
						snap54 := d7
						snap55 := d8
						snap56 := d9
						snap57 := d10
						snap58 := d11
						snap59 := d24
						snap60 := d26
						snap61 := d27
						snap62 := d28
						snap63 := d29
						snap64 := d47
						alloc65 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc65)
						d1 = snap48
						d2 = snap49
						d3 = snap50
						d4 = snap51
						d5 = snap52
						d6 = snap53
						d7 = snap54
						d8 = snap55
						d9 = snap56
						d10 = snap57
						d11 = snap58
						d24 = snap59
						d26 = snap60
						d27 = snap61
						d28 = snap62
						d29 = snap63
						d47 = snap64
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d26)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					d66 = args[0]
					d66.ID = 0
					d68 = d66
					d68.ID = 0
					d67 = ctx.EmitTagEqualsBorrowed(&d68, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d66)
					d69 = d67
					ctx.EnsureDesc(&d69)
					if d69.Loc != LocImm && d69.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d69.Loc == LocImm {
						if d69.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d69.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap70 := d1
						snap71 := d2
						snap72 := d3
						snap73 := d4
						snap74 := d5
						snap75 := d6
						snap76 := d7
						snap77 := d8
						snap78 := d9
						snap79 := d10
						snap80 := d11
						snap81 := d24
						snap82 := d26
						snap83 := d27
						snap84 := d28
						snap85 := d29
						snap86 := d47
						snap87 := d66
						snap88 := d67
						snap89 := d68
						snap90 := d69
						alloc91 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc91)
						d1 = snap70
						d2 = snap71
						d3 = snap72
						d4 = snap73
						d5 = snap74
						d6 = snap75
						d7 = snap76
						d8 = snap77
						d9 = snap78
						d10 = snap79
						d11 = snap80
						d24 = snap81
						d26 = snap82
						d27 = snap83
						d28 = snap84
						d29 = snap85
						d47 = snap86
						d66 = snap87
						d67 = snap88
						d68 = snap89
						d69 = snap90
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d67)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					d92 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d92)
					if d92.Loc == LocRegPair || d92.Loc == LocStackPair || d92.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d92, &result)
						result.Type = d92.Type
					} else {
						switch d92.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d92)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d92)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d92)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d92, &result)
							result.Type = d92.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					d93 = args[0]
					d93.ID = 0
					if d93.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d93.Imm.Int())}
					} else if d93.Type == tagInt && d93.Loc == LocRegPair {
						ctx.FreeReg(d93.Reg)
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d93.Reg2}
						ctx.BindReg(d93.Reg2, &d94)
						ctx.BindReg(d93.Reg2, &d94)
					} else if d93.Type == tagInt && d93.Loc == LocReg {
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d93.Reg}
						ctx.BindReg(d93.Reg, &d94)
						ctx.BindReg(d93.Reg, &d94)
					} else {
						d94 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d93}, 1)
						d94.Type = tagInt
						ctx.BindReg(d94.Reg, &d94)
					}
					ctx.StabilizeDescForControlFlow(&d94)
					ctx.FreeDesc(&d93)
					ctx.SyncDesc(&d94)
					if d94.Loc == LocReg || d94.Loc == LocFPReg {
						ctx.ProtectReg(d94.Reg)
					} else if d94.Loc == LocRegPair {
						ctx.ProtectReg(d94.Reg)
						ctx.ProtectReg(d94.Reg2)
					}
					d95 = d94
					if d95.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d95)
					ctx.EmitStoreToStack(d95, int32(bbs[9].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[9].PhiBase)+int32(16))
					if d94.Loc == LocReg || d94.Loc == LocFPReg {
						ctx.UnprotectReg(d94.Reg)
					} else if d94.Loc == LocRegPair {
						ctx.UnprotectReg(d94.Reg)
						ctx.UnprotectReg(d94.Reg2)
					}
					return bbs[9].Render()
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					d96 = args[0]
					d96.ID = 0
					d97 = ctx.EmitFloatDesc(d96)
					ctx.StabilizeDescForControlFlow(&d97)
					ctx.FreeDesc(&d96)
					ctx.SyncDesc(&d97)
					if d97.Loc == LocReg || d97.Loc == LocFPReg {
						ctx.ProtectReg(d97.Reg)
					} else if d97.Loc == LocRegPair {
						ctx.ProtectReg(d97.Reg)
						ctx.ProtectReg(d97.Reg2)
					}
					d98 = d97
					if d98.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d98)
					ctx.EmitStoreToStack(d98, int32(bbs[16].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[16].PhiBase)+int32(16))
					if d97.Loc == LocReg || d97.Loc == LocFPReg {
						ctx.UnprotectReg(d97.Reg)
					} else if d97.Loc == LocRegPair {
						ctx.UnprotectReg(d97.Reg)
						ctx.UnprotectReg(d97.Reg2)
					}
					return bbs[16].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						idx := int(d3.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d99 = args[idx]
						d99.ID = 0
					} else {
						ctx.EnsureDesc(&d3)
						dynamicArgOff100 = ctx.AllocStack(16)
						ctx.ProtectReg(d3.Reg)
						lbl23 := ctx.ReserveLabel()
						lbl24 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d3.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl24)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d3.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff100))
							ctx.EmitJmp(lbl23)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl24)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff100))
						ctx.MarkLabel(lbl23)
						ctx.UnprotectReg(d3.Reg)
						d99 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff100), Rooted: true}
					}
					if d99.Loc == LocImm {
						d101 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d99.Imm.Int())}
					} else if d99.Type == tagInt && d99.Loc == LocRegPair {
						ctx.FreeReg(d99.Reg)
						d101 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d99.Reg2}
						ctx.BindReg(d99.Reg2, &d101)
						ctx.BindReg(d99.Reg2, &d101)
					} else if d99.Type == tagInt && d99.Loc == LocReg {
						d101 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d99.Reg}
						ctx.BindReg(d99.Reg, &d101)
						ctx.BindReg(d99.Reg, &d101)
					} else {
						d101 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d99}, 1)
						d101.Type = tagInt
						ctx.BindReg(d101.Reg, &d101)
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d101)
					ctx.SyncDesc(&d2)
					ctx.SyncDesc(&d101)
					if d2.Loc == LocImm && d101.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() - d101.Imm.Int())}
					} else if d101.Loc == LocImm && d101.Imm.Int() == 0 {
						ctx.EnsureDesc(&d2)
						r3 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(r3, d2.Reg)
						d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d102)
					} else if d2.Loc == LocImm {
						ctx.EnsureDesc(&d101)
						scratch := ctx.AllocRegExcept(d101.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d101)
						d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d102)
					} else if d101.Loc == LocImm {
						ctx.EnsureDesc(&d2)
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d101.Imm.Int())
						d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d102)
					} else {
						ctx.EnsureDesc(&d2)
						ctx.SyncDesc(&d101)
						r4 := ctx.AllocRegExceptOperand(&d101, d2.Reg)
						ctx.EmitMovRegReg(r4, d2.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r4, &d101)
						d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d102)
					}
					if d102.Loc == LocReg && d2.Loc == LocReg && d102.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d102)
					ctx.EmitStoreToStack(d102, int32(bbs[9].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d102)
					ctx.FreeDesc(&d101)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d103)
					}
					if d103.Loc == LocReg && d3.Loc == LocReg && d103.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d103)
					ctx.EmitStoreToStack(d103, int32(bbs[9].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d103)
					return bbs[9].Render()
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d104)
					ctx.EnsureDescsTogether(&d3, &d104)
					if d3.Loc == LocImm && d104.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() == d104.Imm.Int())}
					} else if d104.Loc == LocImm {
						r5 := ctx.AllocRegExcept(d3.Reg)
						if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d104.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d104.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d105 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
						ctx.BindReg(r5, &d105)
					} else if d3.Loc == LocImm {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d104.Reg)
						d105 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
						ctx.BindReg(r6, &d105)
					} else {
						r7 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d104.Reg)
						d105 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
						ctx.BindReg(r7, &d105)
					}
					ctx.FreeDesc(&d104)
					d106 = d105
					ctx.EnsureDesc(&d106)
					if d106.Loc != LocImm && d106.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d106.Loc == LocImm {
						if d106.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitJump(d106.Condition, lbl12)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d105)
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap107 := d1
						snap108 := d2
						snap109 := d3
						snap110 := d4
						snap111 := d5
						snap112 := d6
						snap113 := d7
						snap114 := d8
						snap115 := d9
						snap116 := d10
						snap117 := d11
						snap118 := d24
						snap119 := d26
						snap120 := d27
						snap121 := d28
						snap122 := d29
						snap123 := d47
						snap124 := d66
						snap125 := d67
						snap126 := d68
						snap127 := d69
						snap128 := d92
						snap129 := d93
						snap130 := d94
						snap131 := d95
						snap132 := d96
						snap133 := d97
						snap134 := d98
						snap135 := d99
						snap136 := d101
						snap137 := d102
						snap138 := d103
						snap139 := d104
						snap140 := d105
						snap141 := d106
						alloc142 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc142)
						d1 = snap107
						d2 = snap108
						d3 = snap109
						d4 = snap110
						d5 = snap111
						d6 = snap112
						d7 = snap113
						d8 = snap114
						d9 = snap115
						d10 = snap116
						d11 = snap117
						d24 = snap118
						d26 = snap119
						d27 = snap120
						d28 = snap121
						d29 = snap122
						d47 = snap123
						d66 = snap124
						d67 = snap125
						d68 = snap126
						d69 = snap127
						d92 = snap128
						d93 = snap129
						d94 = snap130
						d95 = snap131
						d96 = snap132
						d97 = snap133
						d98 = snap134
						d99 = snap135
						d101 = snap136
						d102 = snap137
						d103 = snap138
						d104 = snap139
						d105 = snap140
						d106 = snap141
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.StabilizeDescForControlFlow(&d3)
					d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDescsTogether(&d3, &d143)
					if d3.Loc == LocImm && d143.Loc == LocImm {
						d144 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d143.Imm.Int())}
					} else if d143.Loc == LocImm {
						r8 := ctx.AllocRegExcept(d3.Reg)
						if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d143.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d143.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d144 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedLess}
						ctx.BindReg(r8, &d144)
					} else if d3.Loc == LocImm {
						r9 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d143.Reg)
						d144 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
						ctx.BindReg(r9, &d144)
					} else {
						r10 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d143.Reg)
						d144 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
						ctx.BindReg(r10, &d144)
					}
					ctx.FreeDesc(&d143)
					d145 = d144
					ctx.EnsureDesc(&d145)
					if d145.Loc != LocImm && d145.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d145.Loc == LocImm {
						if d145.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d145.Condition, lbl11)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d144)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap146 := d1
						snap147 := d2
						snap148 := d3
						snap149 := d4
						snap150 := d5
						snap151 := d6
						snap152 := d7
						snap153 := d8
						snap154 := d9
						snap155 := d10
						snap156 := d11
						snap157 := d24
						snap158 := d26
						snap159 := d27
						snap160 := d28
						snap161 := d29
						snap162 := d47
						snap163 := d66
						snap164 := d67
						snap165 := d68
						snap166 := d69
						snap167 := d92
						snap168 := d93
						snap169 := d94
						snap170 := d95
						snap171 := d96
						snap172 := d97
						snap173 := d98
						snap174 := d99
						snap175 := d101
						snap176 := d102
						snap177 := d103
						snap178 := d104
						snap179 := d105
						snap180 := d106
						snap181 := d143
						snap182 := d144
						snap183 := d145
						alloc184 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc184)
						d1 = snap146
						d2 = snap147
						d3 = snap148
						d4 = snap149
						d5 = snap150
						d6 = snap151
						d7 = snap152
						d8 = snap153
						d9 = snap154
						d10 = snap155
						d11 = snap156
						d24 = snap157
						d26 = snap158
						d27 = snap159
						d28 = snap160
						d29 = snap161
						d47 = snap162
						d66 = snap163
						d67 = snap164
						d68 = snap165
						d69 = snap166
						d92 = snap167
						d93 = snap168
						d94 = snap169
						d95 = snap170
						d96 = snap171
						d97 = snap172
						d98 = snap173
						d99 = snap174
						d101 = snap175
						d102 = snap176
						d103 = snap177
						d104 = snap178
						d105 = snap179
						d106 = snap180
						d143 = snap181
						d144 = snap182
						d145 = snap183
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						idx := int(d3.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d185 = args[idx]
						d185.ID = 0
					} else {
						ctx.EnsureDesc(&d3)
						dynamicArgOff186 = ctx.AllocStack(16)
						ctx.ProtectReg(d3.Reg)
						lbl25 := ctx.ReserveLabel()
						lbl26 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d3.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl26)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d3.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff186))
							ctx.EmitJmp(lbl25)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl26)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff186))
						ctx.MarkLabel(lbl25)
						ctx.UnprotectReg(d3.Reg)
						d185 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff186), Rooted: true}
					}
					d188 = d185
					d188.ID = 0
					d187 = ctx.EmitTagEqualsBorrowed(&d188, tagInt, JITValueDesc{Loc: LocAny})
					d189 = d187
					ctx.EnsureDesc(&d189)
					if d189.Loc != LocImm && d189.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d189.Loc == LocImm {
						if d189.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitCmpRegImm32(d189.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap190 := d1
						snap191 := d2
						snap192 := d3
						snap193 := d4
						snap194 := d5
						snap195 := d6
						snap196 := d7
						snap197 := d8
						snap198 := d9
						snap199 := d10
						snap200 := d11
						snap201 := d24
						snap202 := d26
						snap203 := d27
						snap204 := d28
						snap205 := d29
						snap206 := d47
						snap207 := d66
						snap208 := d67
						snap209 := d68
						snap210 := d69
						snap211 := d92
						snap212 := d93
						snap213 := d94
						snap214 := d95
						snap215 := d96
						snap216 := d97
						snap217 := d98
						snap218 := d99
						snap219 := d101
						snap220 := d102
						snap221 := d103
						snap222 := d104
						snap223 := d105
						snap224 := d106
						snap225 := d143
						snap226 := d144
						snap227 := d145
						snap228 := d185
						snap229 := d187
						snap230 := d188
						snap231 := d189
						alloc232 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc232)
						d1 = snap190
						d2 = snap191
						d3 = snap192
						d4 = snap193
						d5 = snap194
						d6 = snap195
						d7 = snap196
						d8 = snap197
						d9 = snap198
						d10 = snap199
						d11 = snap200
						d24 = snap201
						d26 = snap202
						d27 = snap203
						d28 = snap204
						d29 = snap205
						d47 = snap206
						d66 = snap207
						d67 = snap208
						d68 = snap209
						d69 = snap210
						d92 = snap211
						d93 = snap212
						d94 = snap213
						d95 = snap214
						d96 = snap215
						d97 = snap216
						d98 = snap217
						d99 = snap218
						d101 = snap219
						d102 = snap220
						d103 = snap221
						d104 = snap222
						d105 = snap223
						d106 = snap224
						d143 = snap225
						d144 = snap226
						d145 = snap227
						d185 = snap228
						d187 = snap229
						d188 = snap230
						d189 = snap231
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d187)
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						ctx.EmitMakeInt(result, d2)
					} else {
						ctx.EmitMovToReg(result.Reg2, d2)
						d233 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d233)
						if d2.Loc == LocReg && d2.Reg != result.Reg2 {
							ctx.FreeReg(d2.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d234 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d2.Imm.Int()))}
					} else {
						var r11 Reg
						r11 = ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(r11, d2.Reg)
						ctx.EmitInt64ToFloatBits(r11)
						d234 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r11}
						ctx.BindReg(r11, &d234)
					}
					ctx.StabilizeDescForControlFlow(&d234)
					ctx.SyncDesc(&d3)
					if d3.Loc == LocReg || d3.Loc == LocFPReg {
						ctx.ProtectReg(d3.Reg)
					} else if d3.Loc == LocRegPair {
						ctx.ProtectReg(d3.Reg)
						ctx.ProtectReg(d3.Reg2)
					}
					ctx.SyncDesc(&d234)
					if d234.Loc == LocReg || d234.Loc == LocFPReg {
						ctx.ProtectReg(d234.Reg)
					} else if d234.Loc == LocRegPair {
						ctx.ProtectReg(d234.Reg)
						ctx.ProtectReg(d234.Reg2)
					}
					d235 = d3
					if d235.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d235)
					ctx.EmitStoreToStack(d235, int32(bbs[15].PhiBase)+int32(0))
					d236 = d234
					if d236.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d236)
					ctx.EmitStoreToStack(d236, int32(bbs[15].PhiBase)+int32(16))
					if d3.Loc == LocReg || d3.Loc == LocFPReg {
						ctx.UnprotectReg(d3.Reg)
					} else if d3.Loc == LocRegPair {
						ctx.UnprotectReg(d3.Reg)
						ctx.UnprotectReg(d3.Reg2)
					}
					if d234.Loc == LocReg || d234.Loc == LocFPReg {
						ctx.UnprotectReg(d234.Reg)
					} else if d234.Loc == LocRegPair {
						ctx.UnprotectReg(d234.Reg)
						ctx.UnprotectReg(d234.Reg2)
					}
					return bbs[15].Render()
					return result
				}
				bbs[13].Render = func() JITValueDesc {
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
						return result
					}
					bbs[13].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						idx := int(d4.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d237 = args[idx]
						d237.ID = 0
					} else {
						ctx.EnsureDesc(&d4)
						dynamicArgOff238 = ctx.AllocStack(16)
						ctx.ProtectReg(d4.Reg)
						lbl27 := ctx.ReserveLabel()
						lbl28 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d4.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl28)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d4.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff238))
							ctx.EmitJmp(lbl27)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl28)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff238))
						ctx.MarkLabel(lbl27)
						ctx.UnprotectReg(d4.Reg)
						d237 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff238), Rooted: true}
					}
					d239 = ctx.EmitFloatDesc(d237)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d239)
					ctx.EnsureDescsTogether(&d5, &d239)
					if d5.Loc == LocImm && d239.Loc == LocImm {
						d240 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() - d239.Imm.Float())}
					} else if d5.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d239.Reg)
						_, xBits := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitSubFloat64(scratch, d239.Reg)
						d240 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d240)
					} else if d239.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(scratch, d5.Reg)
						_, yBits := d239.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitSubFloat64(scratch, ctx.ScratchReg)
						d240 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d240)
					} else {
						r12 := ctx.AllocRegExcept(d5.Reg, d239.Reg)
						ctx.EmitMovRegReg(r12, d5.Reg)
						ctx.EmitSubFloat64(r12, d239.Reg)
						d240 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d240)
					}
					if d240.Loc == LocReg && d5.Loc == LocReg && d240.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.EnsureDesc(&d240)
					ctx.EmitStoreToStack(d240, int32(bbs[15].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d240)
					ctx.FreeDesc(&d239)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d241 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d241 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d241)
					}
					if d241.Loc == LocReg && d4.Loc == LocReg && d241.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d241)
					ctx.EmitStoreToStack(d241, int32(bbs[15].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d241)
					return bbs[15].Render()
					return result
				}
				bbs[14].Render = func() JITValueDesc {
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
						return result
					}
					bbs[14].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						ctx.EmitMakeFloat(result, d5)
					} else {
						ctx.EmitMovToReg(result.Reg2, d5)
						d242 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d242)
						if d5.Loc == LocReg && d5.Reg != result.Reg2 {
							ctx.FreeReg(d5.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[15].Render = func() JITValueDesc {
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
						return result
					}
					bbs[15].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d5)
					d243 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d243)
					ctx.EnsureDescsTogether(&d4, &d243)
					if d4.Loc == LocImm && d243.Loc == LocImm {
						d244 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d243.Imm.Int())}
					} else if d243.Loc == LocImm {
						r13 := ctx.AllocRegExcept(d4.Reg)
						if d243.Imm.Int() >= -2147483648 && d243.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d4.Reg, int32(d243.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d243.Imm.Int()))
							ctx.EmitCmpInt64(d4.Reg, ctx.ScratchReg)
						}
						d244 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d244)
					} else if d4.Loc == LocImm {
						r14 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d243.Reg)
						d244 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d244)
					} else {
						r15 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpInt64(d4.Reg, d243.Reg)
						d244 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedLess}
						ctx.BindReg(r15, &d244)
					}
					ctx.FreeDesc(&d243)
					d245 = d244
					ctx.EnsureDesc(&d245)
					if d245.Loc != LocImm && d245.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d245.Loc == LocImm {
						if d245.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[14].Render()
					}
					ctx.EmitJump(d245.Condition, lbl14)
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
					}
					ctx.FreeDesc(&d244)
					ctx.FlushRegisterMoves()
					if !bbs[14].Rendered {
						snap246 := d1
						snap247 := d2
						snap248 := d3
						snap249 := d4
						snap250 := d5
						snap251 := d6
						snap252 := d7
						snap253 := d8
						snap254 := d9
						snap255 := d10
						snap256 := d11
						snap257 := d24
						snap258 := d26
						snap259 := d27
						snap260 := d28
						snap261 := d29
						snap262 := d47
						snap263 := d66
						snap264 := d67
						snap265 := d68
						snap266 := d69
						snap267 := d92
						snap268 := d93
						snap269 := d94
						snap270 := d95
						snap271 := d96
						snap272 := d97
						snap273 := d98
						snap274 := d99
						snap275 := d101
						snap276 := d102
						snap277 := d103
						snap278 := d104
						snap279 := d105
						snap280 := d106
						snap281 := d143
						snap282 := d144
						snap283 := d145
						snap284 := d185
						snap285 := d187
						snap286 := d188
						snap287 := d189
						snap288 := d233
						snap289 := d234
						snap290 := d235
						snap291 := d236
						snap292 := d237
						snap293 := d239
						snap294 := d240
						snap295 := d241
						snap296 := d242
						snap297 := d243
						snap298 := d244
						snap299 := d245
						alloc300 := ctx.SnapshotAllocState()
						bbs[14].Render()
						ctx.RestoreAllocState(alloc300)
						d1 = snap246
						d2 = snap247
						d3 = snap248
						d4 = snap249
						d5 = snap250
						d6 = snap251
						d7 = snap252
						d8 = snap253
						d9 = snap254
						d10 = snap255
						d11 = snap256
						d24 = snap257
						d26 = snap258
						d27 = snap259
						d28 = snap260
						d29 = snap261
						d47 = snap262
						d66 = snap263
						d67 = snap264
						d68 = snap265
						d69 = snap266
						d92 = snap267
						d93 = snap268
						d94 = snap269
						d95 = snap270
						d96 = snap271
						d97 = snap272
						d98 = snap273
						d99 = snap274
						d101 = snap275
						d102 = snap276
						d103 = snap277
						d104 = snap278
						d105 = snap279
						d106 = snap280
						d143 = snap281
						d144 = snap282
						d145 = snap283
						d185 = snap284
						d187 = snap285
						d188 = snap286
						d189 = snap287
						d233 = snap288
						d234 = snap289
						d235 = snap290
						d236 = snap291
						d237 = snap292
						d239 = snap293
						d240 = snap294
						d241 = snap295
						d242 = snap296
						d243 = snap297
						d244 = snap298
						d245 = snap299
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					return result
				}
				bbs[16].Render = func() JITValueDesc {
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
						return result
					}
					bbs[16].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d6)
					ctx.StabilizeDescForControlFlow(&d7)
					d301 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d301)
					ctx.EnsureDescsTogether(&d7, &d301)
					if d7.Loc == LocImm && d301.Loc == LocImm {
						d302 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.Int() < d301.Imm.Int())}
					} else if d301.Loc == LocImm {
						r16 := ctx.AllocRegExcept(d7.Reg)
						if d301.Imm.Int() >= -2147483648 && d301.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d7.Reg, int32(d301.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d301.Imm.Int()))
							ctx.EmitCmpInt64(d7.Reg, ctx.ScratchReg)
						}
						d302 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
						ctx.BindReg(r16, &d302)
					} else if d7.Loc == LocImm {
						r17 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d7.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d301.Reg)
						d302 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedLess}
						ctx.BindReg(r17, &d302)
					} else {
						r18 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpInt64(d7.Reg, d301.Reg)
						d302 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
						ctx.BindReg(r18, &d302)
					}
					ctx.FreeDesc(&d301)
					d303 = d302
					ctx.EnsureDesc(&d303)
					if d303.Loc != LocImm && d303.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d303.Loc == LocImm {
						if d303.Imm.Bool() {
							return bbs[17].Render()
						}
						return bbs[18].Render()
					}
					ctx.EmitJump(d303.Condition, lbl18)
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
					}
					ctx.FreeDesc(&d302)
					ctx.FlushRegisterMoves()
					if !bbs[18].Rendered {
						snap304 := d1
						snap305 := d2
						snap306 := d3
						snap307 := d4
						snap308 := d5
						snap309 := d6
						snap310 := d7
						snap311 := d8
						snap312 := d9
						snap313 := d10
						snap314 := d11
						snap315 := d24
						snap316 := d26
						snap317 := d27
						snap318 := d28
						snap319 := d29
						snap320 := d47
						snap321 := d66
						snap322 := d67
						snap323 := d68
						snap324 := d69
						snap325 := d92
						snap326 := d93
						snap327 := d94
						snap328 := d95
						snap329 := d96
						snap330 := d97
						snap331 := d98
						snap332 := d99
						snap333 := d101
						snap334 := d102
						snap335 := d103
						snap336 := d104
						snap337 := d105
						snap338 := d106
						snap339 := d143
						snap340 := d144
						snap341 := d145
						snap342 := d185
						snap343 := d187
						snap344 := d188
						snap345 := d189
						snap346 := d233
						snap347 := d234
						snap348 := d235
						snap349 := d236
						snap350 := d237
						snap351 := d239
						snap352 := d240
						snap353 := d241
						snap354 := d242
						snap355 := d243
						snap356 := d244
						snap357 := d245
						snap358 := d301
						snap359 := d302
						snap360 := d303
						alloc361 := ctx.SnapshotAllocState()
						bbs[18].Render()
						ctx.RestoreAllocState(alloc361)
						d1 = snap304
						d2 = snap305
						d3 = snap306
						d4 = snap307
						d5 = snap308
						d6 = snap309
						d7 = snap310
						d8 = snap311
						d9 = snap312
						d10 = snap313
						d11 = snap314
						d24 = snap315
						d26 = snap316
						d27 = snap317
						d28 = snap318
						d29 = snap319
						d47 = snap320
						d66 = snap321
						d67 = snap322
						d68 = snap323
						d69 = snap324
						d92 = snap325
						d93 = snap326
						d94 = snap327
						d95 = snap328
						d96 = snap329
						d97 = snap330
						d98 = snap331
						d99 = snap332
						d101 = snap333
						d102 = snap334
						d103 = snap335
						d104 = snap336
						d105 = snap337
						d106 = snap338
						d143 = snap339
						d144 = snap340
						d145 = snap341
						d185 = snap342
						d187 = snap343
						d188 = snap344
						d189 = snap345
						d233 = snap346
						d234 = snap347
						d235 = snap348
						d236 = snap349
						d237 = snap350
						d239 = snap351
						d240 = snap352
						d241 = snap353
						d242 = snap354
						d243 = snap355
						d244 = snap356
						d245 = snap357
						d301 = snap358
						d302 = snap359
						d303 = snap360
					}
					if !bbs[17].Rendered {
						return bbs[17].Render()
					}
					return result
					return result
				}
				bbs[17].Render = func() JITValueDesc {
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
						return result
					}
					bbs[17].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						idx := int(d7.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d362 = args[idx]
						d362.ID = 0
					} else {
						ctx.EnsureDesc(&d7)
						dynamicArgOff363 = ctx.AllocStack(16)
						ctx.ProtectReg(d7.Reg)
						lbl29 := ctx.ReserveLabel()
						lbl30 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d7.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl30)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d7.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff363))
							ctx.EmitJmp(lbl29)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl30)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff363))
						ctx.MarkLabel(lbl29)
						ctx.UnprotectReg(d7.Reg)
						d362 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff363), Rooted: true}
					}
					d364 = ctx.EmitFloatDesc(d362)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d364)
					ctx.EnsureDescsTogether(&d6, &d364)
					if d6.Loc == LocImm && d364.Loc == LocImm {
						d365 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d6.Imm.Float() - d364.Imm.Float())}
					} else if d6.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d364.Reg)
						_, xBits := d6.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitSubFloat64(scratch, d364.Reg)
						d365 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d365)
					} else if d364.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitMovRegReg(scratch, d6.Reg)
						_, yBits := d364.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitSubFloat64(scratch, ctx.ScratchReg)
						d365 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d365)
					} else {
						r19 := ctx.AllocRegExcept(d6.Reg, d364.Reg)
						ctx.EmitMovRegReg(r19, d6.Reg)
						ctx.EmitSubFloat64(r19, d364.Reg)
						d365 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r19}
						ctx.BindReg(r19, &d365)
					}
					if d365.Loc == LocReg && d6.Loc == LocReg && d365.Reg == d6.Reg {
						ctx.TransferReg(d6.Reg)
						d6.Loc = LocNone
					}
					ctx.EnsureDesc(&d365)
					ctx.EmitStoreToStack(d365, int32(bbs[16].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d365)
					ctx.FreeDesc(&d364)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d366 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d7.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegReg(scratch, d7.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d366 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d366)
					}
					if d366.Loc == LocReg && d7.Loc == LocReg && d366.Reg == d7.Reg {
						ctx.TransferReg(d7.Reg)
						d7.Loc = LocNone
					}
					ctx.EnsureDesc(&d366)
					ctx.EmitStoreToStack(d366, int32(bbs[16].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d366)
					return bbs[16].Render()
					return result
				}
				bbs[18].Render = func() JITValueDesc {
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
						return result
					}
					bbs[18].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(96)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						ctx.EmitMakeFloat(result, d6)
					} else {
						ctx.EmitMovToReg(result.Reg2, d6)
						d367 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d367)
						if d6.Loc == LocReg && d6.Reg != result.Reg2 {
							ctx.FreeReg(d6.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 72,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_add_numeric_literals",

		Fn: func(a ...Scmer) Scmer {
			return sqlLiteralArithmetic(a[0], a[1], false)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "adds two SQL numeric literals using their exact decimal spellings",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "a", Description: "left literal"},
				{Kind: "number", Label: "b", Description: "right literal"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_add_numeric_literals"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 = JITPrepareScmerGoArg(ctx, d1)
				d2 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d2)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(sqlLiteralArithmetic), []JITValueDesc{d0, d1, d2}, 2)
				d3.NoHeapPointer = false
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.FreeDesc(&d2)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				if d3.Loc == LocImm {
					if result.Loc == LocAny {
						return d3
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d3)
				if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d3, &result)
					result.Type = d3.Type
				} else {
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d3)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d3)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d3)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_sub_numeric_literals",

		Fn: func(a ...Scmer) Scmer {
			return sqlLiteralArithmetic(a[0], a[1], true)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "subtracts two SQL numeric literals using their exact decimal spellings",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "a", Description: "left literal"},
				{Kind: "number", Label: "b", Description: "right literal"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_sub_numeric_literals"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 = JITPrepareScmerGoArg(ctx, d1)
				d2 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d2)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(sqlLiteralArithmetic), []JITValueDesc{d0, d1, d2}, 2)
				d3.NoHeapPointer = false
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.FreeDesc(&d2)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				if d3.Loc == LocImm {
					if result.Loc == LocAny {
						return d3
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d3)
				if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d3, &result)
					result.Type = d3.Type
				} else {
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d3)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d3)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d3)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "*",

		Fn: func(a ...Scmer) Scmer {
			// Integral-valued floats retain the existing integer result whenever
			// the integer prefixes fit. Detect a floating operand before computing
			// any product, so overflow before a later float cannot corrupt it.
			hasFloat := false
			allIntegral := true
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
				if v.IsFloat() {
					hasFloat = true
					if v.Float() != math.Trunc(v.Float()) {
						allIntegral = false
					}
				} else if !v.IsInt() {
					allIntegral = false
				}
			}
			prodInt := int64(1)
			i := 0
			for ; i < len(a); i++ {
				v := a[i]
				var factor int64
				if v.IsInt() {
					factor = v.Int()
				} else if v.IsFloat() {
					f := v.Float()
					if f != math.Trunc(f) || f < -0x1p63 || f >= 0x1p63 {
						break
					}
					factor = int64(f)
				} else {
					break
				}
				product := prodInt * factor
				if hasFloat && factor != 0 &&
					((factor == -1 && prodInt == math.MinInt64) || product/factor != prodInt) {
					break
				}
				prodInt = product
			}
			if i == len(a) {
				return NewInt(prodInt)
			}
			// Recompute from the operands: the integral prefix may have exceeded
			// int64 even though its floating product is finite.
			prodFloat := 1.0
			for _, v := range a {
				prodFloat *= v.Float()
			}
			if allIntegral && prodFloat >= -0x1p63 && prodFloat < 0x1p63 {
				return NewInt(int64(prodFloat))
			}
			return NewFloat(prodFloat)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "multiplies two or more numbers",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value...", Description: "values", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["*"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d26 JITValueDesc
				_ = d26
				var dynamicArgOff27 int32
				var dynamicArgOff28 int32
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d107 JITValueDesc
				_ = d107
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d172 JITValueDesc
				_ = d172
				var d206 JITValueDesc
				_ = d206
				var d207 JITValueDesc
				_ = d207
				var d208 JITValueDesc
				_ = d208
				var dynamicArgOff209 int32
				var dynamicArgOff210 int32
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d253 JITValueDesc
				_ = d253
				var d254 JITValueDesc
				_ = d254
				var d255 JITValueDesc
				_ = d255
				var d298 JITValueDesc
				_ = d298
				var d299 JITValueDesc
				_ = d299
				var d300 JITValueDesc
				_ = d300
				var d346 JITValueDesc
				_ = d346
				var d347 JITValueDesc
				_ = d347
				var d348 JITValueDesc
				_ = d348
				var d349 JITValueDesc
				_ = d349
				var d399 JITValueDesc
				_ = d399
				var d400 JITValueDesc
				_ = d400
				var d401 JITValueDesc
				_ = d401
				var d454 JITValueDesc
				_ = d454
				var d455 JITValueDesc
				_ = d455
				var d456 JITValueDesc
				_ = d456
				var d457 JITValueDesc
				_ = d457
				var d514 JITValueDesc
				_ = d514
				var d515 JITValueDesc
				_ = d515
				var d516 JITValueDesc
				_ = d516
				var d517 JITValueDesc
				_ = d517
				var d578 JITValueDesc
				_ = d578
				var d579 JITValueDesc
				_ = d579
				var d642 JITValueDesc
				_ = d642
				var d643 JITValueDesc
				_ = d643
				var d644 JITValueDesc
				_ = d644
				var d645 JITValueDesc
				_ = d645
				var d712 JITValueDesc
				_ = d712
				var d713 JITValueDesc
				_ = d713
				var d782 JITValueDesc
				_ = d782
				var d783 JITValueDesc
				_ = d783
				var d784 JITValueDesc
				_ = d784
				var d856 JITValueDesc
				_ = d856
				var d857 JITValueDesc
				_ = d857
				var d931 JITValueDesc
				_ = d931
				var d932 JITValueDesc
				_ = d932
				var d933 JITValueDesc
				_ = d933
				var d934 JITValueDesc
				_ = d934
				var d935 JITValueDesc
				_ = d935
				var d1014 JITValueDesc
				_ = d1014
				var dynamicArgOff1015 int32
				var d1016 JITValueDesc
				_ = d1016
				var d1017 JITValueDesc
				_ = d1017
				var d1018 JITValueDesc
				_ = d1018
				var d1019 JITValueDesc
				_ = d1019
				var d1103 JITValueDesc
				_ = d1103
				var d1104 JITValueDesc
				_ = d1104
				var d1105 JITValueDesc
				_ = d1105
				var d1106 JITValueDesc
				_ = d1106
				var d1107 JITValueDesc
				_ = d1107
				var d1196 JITValueDesc
				_ = d1196
				var d1197 JITValueDesc
				_ = d1197
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(128))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [34]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[12].PhiBase = int32(phiBase0) + int32(48)
				bbs[14].PhiBase = int32(phiBase0) + int32(80)
				bbs[27].PhiBase = int32(phiBase0) + int32(96)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				d4 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
				_ = d4
				d5 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
				_ = d5
				d6 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
				_ = d6
				d7 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
				_ = d7
				d8 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
				_ = d8
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_0_14 := int32(-1)
				_ = bbpos_0_14
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_0_15 := int32(-1)
				_ = bbpos_0_15
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_0_16 := int32(-1)
				_ = bbpos_0_16
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_0_17 := int32(-1)
				_ = bbpos_0_17
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_0_18 := int32(-1)
				_ = bbpos_0_18
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_0_19 := int32(-1)
				_ = bbpos_0_19
				lbl20 := ctx.ReserveLabel()
				_ = lbl20
				bbpos_0_20 := int32(-1)
				_ = bbpos_0_20
				lbl21 := ctx.ReserveLabel()
				_ = lbl21
				bbpos_0_21 := int32(-1)
				_ = bbpos_0_21
				lbl22 := ctx.ReserveLabel()
				_ = lbl22
				bbpos_0_22 := int32(-1)
				_ = bbpos_0_22
				lbl23 := ctx.ReserveLabel()
				_ = lbl23
				bbpos_0_23 := int32(-1)
				_ = bbpos_0_23
				lbl24 := ctx.ReserveLabel()
				_ = lbl24
				bbpos_0_24 := int32(-1)
				_ = bbpos_0_24
				lbl25 := ctx.ReserveLabel()
				_ = lbl25
				bbpos_0_25 := int32(-1)
				_ = bbpos_0_25
				lbl26 := ctx.ReserveLabel()
				_ = lbl26
				bbpos_0_26 := int32(-1)
				_ = bbpos_0_26
				lbl27 := ctx.ReserveLabel()
				_ = lbl27
				bbpos_0_27 := int32(-1)
				_ = bbpos_0_27
				lbl28 := ctx.ReserveLabel()
				_ = lbl28
				bbpos_0_28 := int32(-1)
				_ = bbpos_0_28
				lbl29 := ctx.ReserveLabel()
				_ = lbl29
				bbpos_0_29 := int32(-1)
				_ = bbpos_0_29
				lbl30 := ctx.ReserveLabel()
				_ = lbl30
				bbpos_0_30 := int32(-1)
				_ = bbpos_0_30
				lbl31 := ctx.ReserveLabel()
				_ = lbl31
				bbpos_0_31 := int32(-1)
				_ = bbpos_0_31
				lbl32 := ctx.ReserveLabel()
				_ = lbl32
				bbpos_0_32 := int32(-1)
				_ = bbpos_0_32
				lbl33 := ctx.ReserveLabel()
				_ = lbl33
				bbpos_0_33 := int32(-1)
				_ = bbpos_0_33
				lbl34 := ctx.ReserveLabel()
				_ = lbl34
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[1].PhiBase)+int32(16))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(32))
					return bbs[1].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d10)
					}
					if d10.Loc == LocReg && d3.Loc == LocReg && d10.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDescsTogether(&d10, &d9)
					if d10.Loc == LocImm && d9.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() < d9.Imm.Int())}
					} else if d9.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d10.Reg)
						if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d10.Reg, int32(d9.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d9.Imm.Int()))
							ctx.EmitCmpInt64(d10.Reg, ctx.ScratchReg)
						}
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d11)
					} else if d10.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d10.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d9.Reg)
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d11)
					} else {
						r2 := ctx.AllocRegExcept(d10.Reg)
						ctx.EmitCmpInt64(d10.Reg, d9.Reg)
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d11)
					}
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d12.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d11)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap13 := d1
						snap14 := d2
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d6
						snap19 := d7
						snap20 := d8
						snap21 := d9
						snap22 := d10
						snap23 := d11
						snap24 := d12
						alloc25 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc25)
						d1 = snap13
						d2 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d6 = snap18
						d7 = snap19
						d8 = snap20
						d9 = snap21
						d10 = snap22
						d11 = snap23
						d12 = snap24
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocImm {
						idx := int(d10.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d26 = args[idx]
						d26.ID = 0
					} else {
						ctx.EnsureDesc(&d10)
						dynamicArgOff27 = ctx.AllocStack(16)
						ctx.ProtectReg(d10.Reg)
						lbl35 := ctx.ReserveLabel()
						lbl36 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d10.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl36)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d10.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff27))
							ctx.EmitJmp(lbl35)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl36)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff27))
						ctx.MarkLabel(lbl35)
						ctx.UnprotectReg(d10.Reg)
						d26 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff27), Rooted: true}
					}
					dynamicArgOff28 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d26, int32(dynamicArgOff28))
					ctx.FreeDesc(&d26)
					d26 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff28), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d26)
					d30 = d26
					d30.ID = 0
					d29 = ctx.EmitTagEqualsBorrowed(&d30, tagNil, JITValueDesc{Loc: LocAny})
					d31 = d29
					ctx.EnsureDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d31.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap32 := d1
						snap33 := d2
						snap34 := d3
						snap35 := d4
						snap36 := d5
						snap37 := d6
						snap38 := d7
						snap39 := d8
						snap40 := d9
						snap41 := d10
						snap42 := d11
						snap43 := d12
						snap44 := d26
						snap45 := d29
						snap46 := d30
						snap47 := d31
						alloc48 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc48)
						d1 = snap32
						d2 = snap33
						d3 = snap34
						d4 = snap35
						d5 = snap36
						d6 = snap37
						d7 = snap38
						d8 = snap39
						d9 = snap40
						d10 = snap41
						d11 = snap42
						d12 = snap43
						d26 = snap44
						d29 = snap45
						d30 = snap46
						d31 = snap47
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d29)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[12].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[12].PhiBase)+int32(16))
					return bbs[12].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					d49 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d49)
					if d49.Loc == LocRegPair || d49.Loc == LocStackPair || d49.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d49, &result)
						result.Type = d49.Type
					} else {
						switch d49.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d49)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d49)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d49)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d49, &result)
							result.Type = d49.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d26)
					d51 = d26
					d51.ID = 0
					d50 = ctx.EmitTagEqualsBorrowed(&d51, tagFloat, JITValueDesc{Loc: LocAny})
					d52 = d50
					ctx.EnsureDesc(&d52)
					if d52.Loc != LocImm && d52.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d52.Loc == LocImm {
						if d52.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d52.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap53 := d1
						snap54 := d2
						snap55 := d3
						snap56 := d4
						snap57 := d5
						snap58 := d6
						snap59 := d7
						snap60 := d8
						snap61 := d9
						snap62 := d10
						snap63 := d11
						snap64 := d12
						snap65 := d26
						snap66 := d29
						snap67 := d30
						snap68 := d31
						snap69 := d49
						snap70 := d50
						snap71 := d51
						snap72 := d52
						alloc73 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc73)
						d1 = snap53
						d2 = snap54
						d3 = snap55
						d4 = snap56
						d5 = snap57
						d6 = snap58
						d7 = snap59
						d8 = snap60
						d9 = snap61
						d10 = snap62
						d11 = snap63
						d12 = snap64
						d26 = snap65
						d29 = snap66
						d30 = snap67
						d31 = snap68
						d49 = snap69
						d50 = snap70
						d51 = snap71
						d52 = snap72
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d50)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d26)
					d74 = ctx.EmitFloatDesc(d26)
					d75 = ctx.EmitFloatDesc(d26)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d75.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d75)
						var truncSrc Reg
						if d75.Loc == LocRegPair {
							ctx.FreeReg(d75.Reg)
							truncSrc = d75.Reg2
						} else {
							truncSrc = d75.Reg
						}
						truncInt := ctx.AllocRegExcept(truncSrc)
						ctx.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
						ctx.EmitInt64ToFloatBits(truncInt)
						d76 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
						ctx.BindReg(truncInt, &d76)
						ctx.BindReg(truncInt, &d76)
					}
					ctx.FreeDesc(&d75)
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d76)
					ctx.EnsureDescsTogether(&d74, &d76)
					if d74.Loc == LocImm && d76.Loc == LocImm {
						d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d74.Imm.Float() != d76.Imm.Float())}
					} else if d76.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d74.Reg)
						_, yBits := d76.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitCmpFloat64Setcc(r3, d74.Reg, ctx.ScratchReg, CondNotEqual)
						d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d77)
					} else if d74.Loc == LocImm {
						r4 := ctx.AllocRegExcept(d76.Reg)
						_, xBits := d74.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
						ctx.EmitCmpFloat64Setcc(r4, ctx.ScratchReg, d76.Reg, CondNotEqual)
						d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d77)
					} else {
						r5 := ctx.AllocRegExcept(d74.Reg, d76.Reg)
						ctx.EmitCmpFloat64Setcc(r5, d74.Reg, d76.Reg, CondNotEqual)
						d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d77)
					}
					ctx.FreeDesc(&d74)
					ctx.FreeDesc(&d76)
					d78 = d77
					ctx.EnsureDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							return bbs[8].Render()
						}
						ctx.SyncDesc(&d10)
						if d10.Loc == LocReg || d10.Loc == LocFPReg {
							ctx.ProtectReg(d10.Reg)
						} else if d10.Loc == LocRegPair {
							ctx.ProtectReg(d10.Reg)
							ctx.ProtectReg(d10.Reg2)
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[1].PhiBase)+int32(0))
						d79 = d10
						if d79.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d79)
						ctx.EmitStoreToStack(d79, int32(bbs[1].PhiBase)+int32(32))
						if d10.Loc == LocReg || d10.Loc == LocFPReg {
							ctx.UnprotectReg(d10.Reg)
						} else if d10.Loc == LocRegPair {
							ctx.UnprotectReg(d10.Reg)
							ctx.UnprotectReg(d10.Reg2)
						}
						return bbs[1].Render()
					}
					lbl37 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d78.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl37)
					snap80 := d1
					snap81 := d2
					snap82 := d3
					snap83 := d4
					snap84 := d5
					snap85 := d6
					snap86 := d7
					snap87 := d8
					snap88 := d9
					snap89 := d10
					snap90 := d11
					snap91 := d12
					snap92 := d26
					snap93 := d29
					snap94 := d30
					snap95 := d31
					snap96 := d49
					snap97 := d50
					snap98 := d51
					snap99 := d52
					snap100 := d74
					snap101 := d75
					snap102 := d76
					snap103 := d77
					snap104 := d78
					snap105 := d79
					alloc106 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl37)
					ctx.SyncDesc(&d10)
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[1].PhiBase)+int32(0))
					d107 = d10
					if d107.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d107)
					ctx.EmitStoreToStack(d107, int32(bbs[1].PhiBase)+int32(32))
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc106)
					d1 = snap80
					d2 = snap81
					d3 = snap82
					d4 = snap83
					d5 = snap84
					d6 = snap85
					d7 = snap86
					d8 = snap87
					d9 = snap88
					d10 = snap89
					d11 = snap90
					d12 = snap91
					d26 = snap92
					d29 = snap93
					d30 = snap94
					d31 = snap95
					d49 = snap96
					d50 = snap97
					d51 = snap98
					d52 = snap99
					d74 = snap100
					d75 = snap101
					d76 = snap102
					d77 = snap103
					d78 = snap104
					d79 = snap105
					if !bbs[1].Rendered {
						snap108 := d1
						snap109 := d2
						snap110 := d3
						snap111 := d4
						snap112 := d5
						snap113 := d6
						snap114 := d7
						snap115 := d8
						snap116 := d9
						snap117 := d10
						snap118 := d11
						snap119 := d12
						snap120 := d26
						snap121 := d29
						snap122 := d30
						snap123 := d31
						snap124 := d49
						snap125 := d50
						snap126 := d51
						snap127 := d52
						snap128 := d74
						snap129 := d75
						snap130 := d76
						snap131 := d77
						snap132 := d78
						snap133 := d79
						snap134 := d107
						alloc135 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc135)
						d1 = snap108
						d2 = snap109
						d3 = snap110
						d4 = snap111
						d5 = snap112
						d6 = snap113
						d7 = snap114
						d8 = snap115
						d9 = snap116
						d10 = snap117
						d11 = snap118
						d12 = snap119
						d26 = snap120
						d29 = snap121
						d30 = snap122
						d31 = snap123
						d49 = snap124
						d50 = snap125
						d51 = snap126
						d52 = snap127
						d74 = snap128
						d75 = snap129
						d76 = snap130
						d77 = snap131
						d78 = snap132
						d79 = snap133
						d107 = snap134
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d77)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d26)
					d137 = d26
					d137.ID = 0
					d136 = ctx.EmitTagEqualsBorrowed(&d137, tagInt, JITValueDesc{Loc: LocAny})
					d138 = d136
					ctx.EnsureDesc(&d138)
					if d138.Loc != LocImm && d138.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d138.Loc == LocImm {
						if d138.Imm.Bool() {
							ctx.SyncDesc(&d10)
							if d10.Loc == LocReg || d10.Loc == LocFPReg {
								ctx.ProtectReg(d10.Reg)
							} else if d10.Loc == LocRegPair {
								ctx.ProtectReg(d10.Reg)
								ctx.ProtectReg(d10.Reg2)
							}
							d139 = d10
							if d139.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d139)
							ctx.EmitStoreToStack(d139, int32(bbs[1].PhiBase)+int32(32))
							if d10.Loc == LocReg || d10.Loc == LocFPReg {
								ctx.UnprotectReg(d10.Reg)
							} else if d10.Loc == LocRegPair {
								ctx.UnprotectReg(d10.Reg)
								ctx.UnprotectReg(d10.Reg2)
							}
							return bbs[1].Render()
						}
						return bbs[9].Render()
					}
					lbl38 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d138.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl38)
					ctx.EmitJmp(lbl10)
					snap140 := d1
					snap141 := d2
					snap142 := d3
					snap143 := d4
					snap144 := d5
					snap145 := d6
					snap146 := d7
					snap147 := d8
					snap148 := d9
					snap149 := d10
					snap150 := d11
					snap151 := d12
					snap152 := d26
					snap153 := d29
					snap154 := d30
					snap155 := d31
					snap156 := d49
					snap157 := d50
					snap158 := d51
					snap159 := d52
					snap160 := d74
					snap161 := d75
					snap162 := d76
					snap163 := d77
					snap164 := d78
					snap165 := d79
					snap166 := d107
					snap167 := d136
					snap168 := d137
					snap169 := d138
					snap170 := d139
					alloc171 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl38)
					ctx.SyncDesc(&d10)
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d172 = d10
					if d172.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d172)
					ctx.EmitStoreToStack(d172, int32(bbs[1].PhiBase)+int32(32))
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc171)
					d1 = snap140
					d2 = snap141
					d3 = snap142
					d4 = snap143
					d5 = snap144
					d6 = snap145
					d7 = snap146
					d8 = snap147
					d9 = snap148
					d10 = snap149
					d11 = snap150
					d12 = snap151
					d26 = snap152
					d29 = snap153
					d30 = snap154
					d31 = snap155
					d49 = snap156
					d50 = snap157
					d51 = snap158
					d52 = snap159
					d74 = snap160
					d75 = snap161
					d76 = snap162
					d77 = snap163
					d78 = snap164
					d79 = snap165
					d107 = snap166
					d136 = snap167
					d137 = snap168
					d138 = snap169
					d139 = snap170
					if !bbs[1].Rendered {
						snap173 := d1
						snap174 := d2
						snap175 := d3
						snap176 := d4
						snap177 := d5
						snap178 := d6
						snap179 := d7
						snap180 := d8
						snap181 := d9
						snap182 := d10
						snap183 := d11
						snap184 := d12
						snap185 := d26
						snap186 := d29
						snap187 := d30
						snap188 := d31
						snap189 := d49
						snap190 := d50
						snap191 := d51
						snap192 := d52
						snap193 := d74
						snap194 := d75
						snap195 := d76
						snap196 := d77
						snap197 := d78
						snap198 := d79
						snap199 := d107
						snap200 := d136
						snap201 := d137
						snap202 := d138
						snap203 := d139
						snap204 := d172
						alloc205 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc205)
						d1 = snap173
						d2 = snap174
						d3 = snap175
						d4 = snap176
						d5 = snap177
						d6 = snap178
						d7 = snap179
						d8 = snap180
						d9 = snap181
						d10 = snap182
						d11 = snap183
						d12 = snap184
						d26 = snap185
						d29 = snap186
						d30 = snap187
						d31 = snap188
						d49 = snap189
						d50 = snap190
						d51 = snap191
						d52 = snap192
						d74 = snap193
						d75 = snap194
						d76 = snap195
						d77 = snap196
						d78 = snap197
						d79 = snap198
						d107 = snap199
						d136 = snap200
						d137 = snap201
						d138 = snap202
						d139 = snap203
						d172 = snap204
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					ctx.FreeDesc(&d136)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d10)
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[1].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
					d206 = d10
					if d206.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d206)
					ctx.EmitStoreToStack(d206, int32(bbs[1].PhiBase)+int32(32))
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
					return bbs[1].Render()
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d10)
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
					d207 = d10
					if d207.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d207)
					ctx.EmitStoreToStack(d207, int32(bbs[1].PhiBase)+int32(32))
					if d10.Loc == LocReg || d10.Loc == LocFPReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
					return bbs[1].Render()
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						idx := int(d5.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d208 = args[idx]
						d208.ID = 0
					} else {
						ctx.EnsureDesc(&d5)
						dynamicArgOff209 = ctx.AllocStack(16)
						ctx.ProtectReg(d5.Reg)
						lbl39 := ctx.ReserveLabel()
						lbl40 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d5.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl40)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d5.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff209))
							ctx.EmitJmp(lbl39)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl40)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff209))
						ctx.MarkLabel(lbl39)
						ctx.UnprotectReg(d5.Reg)
						d208 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff209), Rooted: true}
					}
					dynamicArgOff210 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d208, int32(dynamicArgOff210))
					ctx.FreeDesc(&d208)
					d208 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff210), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d208)
					d212 = d208
					d212.ID = 0
					d211 = ctx.EmitTagEqualsBorrowed(&d212, tagInt, JITValueDesc{Loc: LocAny})
					d213 = d211
					ctx.EnsureDesc(&d213)
					if d213.Loc != LocImm && d213.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d213.Loc == LocImm {
						if d213.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitCmpRegImm32(d213.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap214 := d1
						snap215 := d2
						snap216 := d3
						snap217 := d4
						snap218 := d5
						snap219 := d6
						snap220 := d7
						snap221 := d8
						snap222 := d9
						snap223 := d10
						snap224 := d11
						snap225 := d12
						snap226 := d26
						snap227 := d29
						snap228 := d30
						snap229 := d31
						snap230 := d49
						snap231 := d50
						snap232 := d51
						snap233 := d52
						snap234 := d74
						snap235 := d75
						snap236 := d76
						snap237 := d77
						snap238 := d78
						snap239 := d79
						snap240 := d107
						snap241 := d136
						snap242 := d137
						snap243 := d138
						snap244 := d139
						snap245 := d172
						snap246 := d206
						snap247 := d207
						snap248 := d208
						snap249 := d211
						snap250 := d212
						snap251 := d213
						alloc252 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc252)
						d1 = snap214
						d2 = snap215
						d3 = snap216
						d4 = snap217
						d5 = snap218
						d6 = snap219
						d7 = snap220
						d8 = snap221
						d9 = snap222
						d10 = snap223
						d11 = snap224
						d12 = snap225
						d26 = snap226
						d29 = snap227
						d30 = snap228
						d31 = snap229
						d49 = snap230
						d50 = snap231
						d51 = snap232
						d52 = snap233
						d74 = snap234
						d75 = snap235
						d76 = snap236
						d77 = snap237
						d78 = snap238
						d79 = snap239
						d107 = snap240
						d136 = snap241
						d137 = snap242
						d138 = snap243
						d139 = snap244
						d172 = snap245
						d206 = snap246
						d207 = snap247
						d208 = snap248
						d211 = snap249
						d212 = snap250
						d213 = snap251
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					ctx.FreeDesc(&d211)
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					d253 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d253)
					ctx.EnsureDescsTogether(&d5, &d253)
					if d5.Loc == LocImm && d253.Loc == LocImm {
						d254 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() == d253.Imm.Int())}
					} else if d253.Loc == LocImm {
						r6 := ctx.AllocRegExcept(d5.Reg)
						if d253.Imm.Int() >= -2147483648 && d253.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d5.Reg, int32(d253.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d253.Imm.Int()))
							ctx.EmitCmpInt64(d5.Reg, ctx.ScratchReg)
						}
						d254 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
						ctx.BindReg(r6, &d254)
					} else if d5.Loc == LocImm {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d5.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d253.Reg)
						d254 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
						ctx.BindReg(r7, &d254)
					} else {
						r8 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpInt64(d5.Reg, d253.Reg)
						d254 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondEqual}
						ctx.BindReg(r8, &d254)
					}
					ctx.FreeDesc(&d253)
					d255 = d254
					ctx.EnsureDesc(&d255)
					if d255.Loc != LocImm && d255.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d255.Loc == LocImm {
						if d255.Imm.Bool() {
							return bbs[25].Render()
						}
						return bbs[26].Render()
					}
					ctx.EmitJump(d255.Condition, lbl26)
					if bbs[26].Rendered {
						ctx.EmitJmp(lbl27)
					}
					ctx.FreeDesc(&d254)
					ctx.FlushRegisterMoves()
					if !bbs[26].Rendered {
						snap256 := d1
						snap257 := d2
						snap258 := d3
						snap259 := d4
						snap260 := d5
						snap261 := d6
						snap262 := d7
						snap263 := d8
						snap264 := d9
						snap265 := d10
						snap266 := d11
						snap267 := d12
						snap268 := d26
						snap269 := d29
						snap270 := d30
						snap271 := d31
						snap272 := d49
						snap273 := d50
						snap274 := d51
						snap275 := d52
						snap276 := d74
						snap277 := d75
						snap278 := d76
						snap279 := d77
						snap280 := d78
						snap281 := d79
						snap282 := d107
						snap283 := d136
						snap284 := d137
						snap285 := d138
						snap286 := d139
						snap287 := d172
						snap288 := d206
						snap289 := d207
						snap290 := d208
						snap291 := d211
						snap292 := d212
						snap293 := d213
						snap294 := d253
						snap295 := d254
						snap296 := d255
						alloc297 := ctx.SnapshotAllocState()
						bbs[26].Render()
						ctx.RestoreAllocState(alloc297)
						d1 = snap256
						d2 = snap257
						d3 = snap258
						d4 = snap259
						d5 = snap260
						d6 = snap261
						d7 = snap262
						d8 = snap263
						d9 = snap264
						d10 = snap265
						d11 = snap266
						d12 = snap267
						d26 = snap268
						d29 = snap269
						d30 = snap270
						d31 = snap271
						d49 = snap272
						d50 = snap273
						d51 = snap274
						d52 = snap275
						d74 = snap276
						d75 = snap277
						d76 = snap278
						d77 = snap279
						d78 = snap280
						d79 = snap281
						d107 = snap282
						d136 = snap283
						d137 = snap284
						d138 = snap285
						d139 = snap286
						d172 = snap287
						d206 = snap288
						d207 = snap289
						d208 = snap290
						d211 = snap291
						d212 = snap292
						d213 = snap293
						d253 = snap294
						d254 = snap295
						d255 = snap296
					}
					if !bbs[25].Rendered {
						return bbs[25].Render()
					}
					return result
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d5)
					d298 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d298)
					ctx.EnsureDescsTogether(&d5, &d298)
					if d5.Loc == LocImm && d298.Loc == LocImm {
						d299 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() < d298.Imm.Int())}
					} else if d298.Loc == LocImm {
						r9 := ctx.AllocRegExcept(d5.Reg)
						if d298.Imm.Int() >= -2147483648 && d298.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d5.Reg, int32(d298.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d298.Imm.Int()))
							ctx.EmitCmpInt64(d5.Reg, ctx.ScratchReg)
						}
						d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
						ctx.BindReg(r9, &d299)
					} else if d5.Loc == LocImm {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d5.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d298.Reg)
						d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
						ctx.BindReg(r10, &d299)
					} else {
						r11 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpInt64(d5.Reg, d298.Reg)
						d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
						ctx.BindReg(r11, &d299)
					}
					ctx.FreeDesc(&d298)
					d300 = d299
					ctx.EnsureDesc(&d300)
					if d300.Loc != LocImm && d300.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d300.Loc == LocImm {
						if d300.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitJump(d300.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d299)
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap301 := d1
						snap302 := d2
						snap303 := d3
						snap304 := d4
						snap305 := d5
						snap306 := d6
						snap307 := d7
						snap308 := d8
						snap309 := d9
						snap310 := d10
						snap311 := d11
						snap312 := d12
						snap313 := d26
						snap314 := d29
						snap315 := d30
						snap316 := d31
						snap317 := d49
						snap318 := d50
						snap319 := d51
						snap320 := d52
						snap321 := d74
						snap322 := d75
						snap323 := d76
						snap324 := d77
						snap325 := d78
						snap326 := d79
						snap327 := d107
						snap328 := d136
						snap329 := d137
						snap330 := d138
						snap331 := d139
						snap332 := d172
						snap333 := d206
						snap334 := d207
						snap335 := d208
						snap336 := d211
						snap337 := d212
						snap338 := d213
						snap339 := d253
						snap340 := d254
						snap341 := d255
						snap342 := d298
						snap343 := d299
						snap344 := d300
						alloc345 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc345)
						d1 = snap301
						d2 = snap302
						d3 = snap303
						d4 = snap304
						d5 = snap305
						d6 = snap306
						d7 = snap307
						d8 = snap308
						d9 = snap309
						d10 = snap310
						d11 = snap311
						d12 = snap312
						d26 = snap313
						d29 = snap314
						d30 = snap315
						d31 = snap316
						d49 = snap317
						d50 = snap318
						d51 = snap319
						d52 = snap320
						d74 = snap321
						d75 = snap322
						d76 = snap323
						d77 = snap324
						d78 = snap325
						d79 = snap326
						d107 = snap327
						d136 = snap328
						d137 = snap329
						d138 = snap330
						d139 = snap331
						d172 = snap332
						d206 = snap333
						d207 = snap334
						d208 = snap335
						d211 = snap336
						d212 = snap337
						d213 = snap338
						d253 = snap339
						d254 = snap340
						d255 = snap341
						d298 = snap342
						d299 = snap343
						d300 = snap344
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					return result
				}
				bbs[13].Render = func() JITValueDesc {
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
						return result
					}
					bbs[13].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d208)
					if d208.Loc == LocImm {
						d346 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d208.Imm.Int())}
					} else if d208.Type == tagInt && d208.Loc == LocRegPair {
						ctx.FreeReg(d208.Reg)
						d346 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d208.Reg2}
						ctx.BindReg(d208.Reg2, &d346)
						ctx.BindReg(d208.Reg2, &d346)
					} else if d208.Type == tagInt && d208.Loc == LocReg {
						d346 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d208.Reg}
						ctx.BindReg(d208.Reg, &d346)
						ctx.BindReg(d208.Reg, &d346)
					} else {
						d346 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d208}, 1)
						d346.Type = tagInt
						ctx.BindReg(d346.Reg, &d346)
					}
					ctx.StabilizeDescForControlFlow(&d346)
					ctx.SyncDesc(&d346)
					if d346.Loc == LocReg || d346.Loc == LocFPReg {
						ctx.ProtectReg(d346.Reg)
					} else if d346.Loc == LocRegPair {
						ctx.ProtectReg(d346.Reg)
						ctx.ProtectReg(d346.Reg2)
					}
					d347 = d346
					if d347.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d347)
					ctx.EmitStoreToStack(d347, int32(bbs[14].PhiBase)+int32(0))
					if d346.Loc == LocReg || d346.Loc == LocFPReg {
						ctx.UnprotectReg(d346.Reg)
					} else if d346.Loc == LocRegPair {
						ctx.UnprotectReg(d346.Reg)
						ctx.UnprotectReg(d346.Reg2)
					}
					return bbs[14].Render()
					return result
				}
				bbs[14].Render = func() JITValueDesc {
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
						return result
					}
					bbs[14].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d6)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d6)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d6)
					if d4.Loc == LocImm && d6.Loc == LocImm {
						d348 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() * d6.Imm.Int())}
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d6)
						scratch := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitMovRegReg(scratch, d6.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d4.Imm.Int())
						d348 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d348)
					} else if d6.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d6.Imm.Int())
						d348 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d348)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d6)
						r12 := ctx.AllocRegExceptOperand(&d6, d4.Reg)
						ctx.EmitMovRegReg(r12, d4.Reg)
						ctx.EmitIntBinary(JITIntMul, 64, r12, &d6)
						d348 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
						ctx.BindReg(r12, &d348)
					}
					if d348.Loc == LocReg && d4.Loc == LocReg && d348.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d348)
					d349 = d1
					ctx.EnsureDesc(&d349)
					if d349.Loc != LocImm && d349.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d349.Loc == LocImm {
						if d349.Imm.Bool() {
							return bbs[22].Render()
						}
						return bbs[20].Render()
					}
					ctx.EmitCmpRegImm32(d349.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl23)
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
					}
					ctx.FlushRegisterMoves()
					if !bbs[20].Rendered {
						snap350 := d1
						snap351 := d2
						snap352 := d3
						snap353 := d4
						snap354 := d5
						snap355 := d6
						snap356 := d7
						snap357 := d8
						snap358 := d9
						snap359 := d10
						snap360 := d11
						snap361 := d12
						snap362 := d26
						snap363 := d29
						snap364 := d30
						snap365 := d31
						snap366 := d49
						snap367 := d50
						snap368 := d51
						snap369 := d52
						snap370 := d74
						snap371 := d75
						snap372 := d76
						snap373 := d77
						snap374 := d78
						snap375 := d79
						snap376 := d107
						snap377 := d136
						snap378 := d137
						snap379 := d138
						snap380 := d139
						snap381 := d172
						snap382 := d206
						snap383 := d207
						snap384 := d208
						snap385 := d211
						snap386 := d212
						snap387 := d213
						snap388 := d253
						snap389 := d254
						snap390 := d255
						snap391 := d298
						snap392 := d299
						snap393 := d300
						snap394 := d346
						snap395 := d347
						snap396 := d348
						snap397 := d349
						alloc398 := ctx.SnapshotAllocState()
						bbs[20].Render()
						ctx.RestoreAllocState(alloc398)
						d1 = snap350
						d2 = snap351
						d3 = snap352
						d4 = snap353
						d5 = snap354
						d6 = snap355
						d7 = snap356
						d8 = snap357
						d9 = snap358
						d10 = snap359
						d11 = snap360
						d12 = snap361
						d26 = snap362
						d29 = snap363
						d30 = snap364
						d31 = snap365
						d49 = snap366
						d50 = snap367
						d51 = snap368
						d52 = snap369
						d74 = snap370
						d75 = snap371
						d76 = snap372
						d77 = snap373
						d78 = snap374
						d79 = snap375
						d107 = snap376
						d136 = snap377
						d137 = snap378
						d138 = snap379
						d139 = snap380
						d172 = snap381
						d206 = snap382
						d207 = snap383
						d208 = snap384
						d211 = snap385
						d212 = snap386
						d213 = snap387
						d253 = snap388
						d254 = snap389
						d255 = snap390
						d298 = snap391
						d299 = snap392
						d300 = snap393
						d346 = snap394
						d347 = snap395
						d348 = snap396
						d349 = snap397
					}
					if !bbs[22].Rendered {
						return bbs[22].Render()
					}
					return result
					return result
				}
				bbs[15].Render = func() JITValueDesc {
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
						return result
					}
					bbs[15].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d208)
					d400 = d208
					d400.ID = 0
					d399 = ctx.EmitTagEqualsBorrowed(&d400, tagFloat, JITValueDesc{Loc: LocAny})
					d401 = d399
					ctx.EnsureDesc(&d401)
					if d401.Loc != LocImm && d401.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d401.Loc == LocImm {
						if d401.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitCmpRegImm32(d401.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap402 := d1
						snap403 := d2
						snap404 := d3
						snap405 := d4
						snap406 := d5
						snap407 := d6
						snap408 := d7
						snap409 := d8
						snap410 := d9
						snap411 := d10
						snap412 := d11
						snap413 := d12
						snap414 := d26
						snap415 := d29
						snap416 := d30
						snap417 := d31
						snap418 := d49
						snap419 := d50
						snap420 := d51
						snap421 := d52
						snap422 := d74
						snap423 := d75
						snap424 := d76
						snap425 := d77
						snap426 := d78
						snap427 := d79
						snap428 := d107
						snap429 := d136
						snap430 := d137
						snap431 := d138
						snap432 := d139
						snap433 := d172
						snap434 := d206
						snap435 := d207
						snap436 := d208
						snap437 := d211
						snap438 := d212
						snap439 := d213
						snap440 := d253
						snap441 := d254
						snap442 := d255
						snap443 := d298
						snap444 := d299
						snap445 := d300
						snap446 := d346
						snap447 := d347
						snap448 := d348
						snap449 := d349
						snap450 := d399
						snap451 := d400
						snap452 := d401
						alloc453 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc453)
						d1 = snap402
						d2 = snap403
						d3 = snap404
						d4 = snap405
						d5 = snap406
						d6 = snap407
						d7 = snap408
						d8 = snap409
						d9 = snap410
						d10 = snap411
						d11 = snap412
						d12 = snap413
						d26 = snap414
						d29 = snap415
						d30 = snap416
						d31 = snap417
						d49 = snap418
						d50 = snap419
						d51 = snap420
						d52 = snap421
						d74 = snap422
						d75 = snap423
						d76 = snap424
						d77 = snap425
						d78 = snap426
						d79 = snap427
						d107 = snap428
						d136 = snap429
						d137 = snap430
						d138 = snap431
						d139 = snap432
						d172 = snap433
						d206 = snap434
						d207 = snap435
						d208 = snap436
						d211 = snap437
						d212 = snap438
						d213 = snap439
						d253 = snap440
						d254 = snap441
						d255 = snap442
						d298 = snap443
						d299 = snap444
						d300 = snap445
						d346 = snap446
						d347 = snap447
						d348 = snap448
						d349 = snap449
						d399 = snap450
						d400 = snap451
						d401 = snap452
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d399)
					return result
				}
				bbs[16].Render = func() JITValueDesc {
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
						return result
					}
					bbs[16].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d208)
					d454 = ctx.EmitFloatDesc(d208)
					ctx.StabilizeDescForControlFlow(&d454)
					ctx.EnsureDesc(&d454)
					if d454.Loc == LocImm {
						d455 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d454.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d454)
						var truncSrc Reg
						if d454.Loc == LocRegPair {
							ctx.FreeReg(d454.Reg)
							truncSrc = d454.Reg2
						} else {
							truncSrc = d454.Reg
						}
						truncInt := ctx.AllocRegExcept(truncSrc)
						ctx.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
						ctx.EmitInt64ToFloatBits(truncInt)
						d455 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
						ctx.BindReg(truncInt, &d455)
						ctx.BindReg(truncInt, &d455)
					}
					ctx.EnsureDesc(&d454)
					ctx.EnsureDesc(&d455)
					ctx.EnsureDescsTogether(&d454, &d455)
					if d454.Loc == LocImm && d455.Loc == LocImm {
						d456 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d454.Imm.Float() != d455.Imm.Float())}
					} else if d455.Loc == LocImm {
						r13 := ctx.AllocRegExcept(d454.Reg)
						_, yBits := d455.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitCmpFloat64Setcc(r13, d454.Reg, ctx.ScratchReg, CondNotEqual)
						d456 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
						ctx.BindReg(r13, &d456)
					} else if d454.Loc == LocImm {
						r14 := ctx.AllocRegExcept(d455.Reg)
						_, xBits := d454.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
						ctx.EmitCmpFloat64Setcc(r14, ctx.ScratchReg, d455.Reg, CondNotEqual)
						d456 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d456)
					} else {
						r15 := ctx.AllocRegExcept(d454.Reg, d455.Reg)
						ctx.EmitCmpFloat64Setcc(r15, d454.Reg, d455.Reg, CondNotEqual)
						d456 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
						ctx.BindReg(r15, &d456)
					}
					ctx.FreeDesc(&d455)
					d457 = d456
					ctx.EnsureDesc(&d457)
					if d457.Loc != LocImm && d457.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d457.Loc == LocImm {
						if d457.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d457.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap458 := d1
						snap459 := d2
						snap460 := d3
						snap461 := d4
						snap462 := d5
						snap463 := d6
						snap464 := d7
						snap465 := d8
						snap466 := d9
						snap467 := d10
						snap468 := d11
						snap469 := d12
						snap470 := d26
						snap471 := d29
						snap472 := d30
						snap473 := d31
						snap474 := d49
						snap475 := d50
						snap476 := d51
						snap477 := d52
						snap478 := d74
						snap479 := d75
						snap480 := d76
						snap481 := d77
						snap482 := d78
						snap483 := d79
						snap484 := d107
						snap485 := d136
						snap486 := d137
						snap487 := d138
						snap488 := d139
						snap489 := d172
						snap490 := d206
						snap491 := d207
						snap492 := d208
						snap493 := d211
						snap494 := d212
						snap495 := d213
						snap496 := d253
						snap497 := d254
						snap498 := d255
						snap499 := d298
						snap500 := d299
						snap501 := d300
						snap502 := d346
						snap503 := d347
						snap504 := d348
						snap505 := d349
						snap506 := d399
						snap507 := d400
						snap508 := d401
						snap509 := d454
						snap510 := d455
						snap511 := d456
						snap512 := d457
						alloc513 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc513)
						d1 = snap458
						d2 = snap459
						d3 = snap460
						d4 = snap461
						d5 = snap462
						d6 = snap463
						d7 = snap464
						d8 = snap465
						d9 = snap466
						d10 = snap467
						d11 = snap468
						d12 = snap469
						d26 = snap470
						d29 = snap471
						d30 = snap472
						d31 = snap473
						d49 = snap474
						d50 = snap475
						d51 = snap476
						d52 = snap477
						d74 = snap478
						d75 = snap479
						d76 = snap480
						d77 = snap481
						d78 = snap482
						d79 = snap483
						d107 = snap484
						d136 = snap485
						d137 = snap486
						d138 = snap487
						d139 = snap488
						d172 = snap489
						d206 = snap490
						d207 = snap491
						d208 = snap492
						d211 = snap493
						d212 = snap494
						d213 = snap495
						d253 = snap496
						d254 = snap497
						d255 = snap498
						d298 = snap499
						d299 = snap500
						d300 = snap501
						d346 = snap502
						d347 = snap503
						d348 = snap504
						d349 = snap505
						d399 = snap506
						d400 = snap507
						d401 = snap508
						d454 = snap509
						d455 = snap510
						d456 = snap511
						d457 = snap512
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					ctx.FreeDesc(&d456)
					return result
				}
				bbs[17].Render = func() JITValueDesc {
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
						return result
					}
					bbs[17].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d454)
					ctx.EnsureDesc(&d454)
					if d454.Loc == LocImm {
						d514 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d454.Imm.Float()))}
					} else {
						r16 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r16, d454.Reg)
						d514 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d514)
					}
					ctx.StabilizeDescForControlFlow(&d514)
					ctx.SyncDesc(&d514)
					if d514.Loc == LocReg || d514.Loc == LocFPReg {
						ctx.ProtectReg(d514.Reg)
					} else if d514.Loc == LocRegPair {
						ctx.ProtectReg(d514.Reg)
						ctx.ProtectReg(d514.Reg2)
					}
					d515 = d514
					if d515.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d515)
					ctx.EmitStoreToStack(d515, int32(bbs[14].PhiBase)+int32(0))
					if d514.Loc == LocReg || d514.Loc == LocFPReg {
						ctx.UnprotectReg(d514.Reg)
					} else if d514.Loc == LocRegPair {
						ctx.UnprotectReg(d514.Reg)
						ctx.UnprotectReg(d514.Reg2)
					}
					return bbs[14].Render()
					return result
				}
				bbs[18].Render = func() JITValueDesc {
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
						return result
					}
					bbs[18].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d454)
					if d454.Loc == LocImm {
						d516 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d454.Imm.Float() >= 9.223372036854776e+18)}
					} else {
						r17 := ctx.AllocRegExcept(d454.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4890909195324358656))
						ctx.EmitCmpFloat64(d454.Reg, ctx.ScratchReg)
						d516 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondUnsignedAboveOrEqual}
						ctx.BindReg(r17, &d516)
					}
					d517 = d516
					ctx.EnsureDesc(&d517)
					if d517.Loc != LocImm && d517.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d517.Loc == LocImm {
						if d517.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitJump(d517.Condition, lbl12)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d516)
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap518 := d1
						snap519 := d2
						snap520 := d3
						snap521 := d4
						snap522 := d5
						snap523 := d6
						snap524 := d7
						snap525 := d8
						snap526 := d9
						snap527 := d10
						snap528 := d11
						snap529 := d12
						snap530 := d26
						snap531 := d29
						snap532 := d30
						snap533 := d31
						snap534 := d49
						snap535 := d50
						snap536 := d51
						snap537 := d52
						snap538 := d74
						snap539 := d75
						snap540 := d76
						snap541 := d77
						snap542 := d78
						snap543 := d79
						snap544 := d107
						snap545 := d136
						snap546 := d137
						snap547 := d138
						snap548 := d139
						snap549 := d172
						snap550 := d206
						snap551 := d207
						snap552 := d208
						snap553 := d211
						snap554 := d212
						snap555 := d213
						snap556 := d253
						snap557 := d254
						snap558 := d255
						snap559 := d298
						snap560 := d299
						snap561 := d300
						snap562 := d346
						snap563 := d347
						snap564 := d348
						snap565 := d349
						snap566 := d399
						snap567 := d400
						snap568 := d401
						snap569 := d454
						snap570 := d455
						snap571 := d456
						snap572 := d457
						snap573 := d514
						snap574 := d515
						snap575 := d516
						snap576 := d517
						alloc577 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc577)
						d1 = snap518
						d2 = snap519
						d3 = snap520
						d4 = snap521
						d5 = snap522
						d6 = snap523
						d7 = snap524
						d8 = snap525
						d9 = snap526
						d10 = snap527
						d11 = snap528
						d12 = snap529
						d26 = snap530
						d29 = snap531
						d30 = snap532
						d31 = snap533
						d49 = snap534
						d50 = snap535
						d51 = snap536
						d52 = snap537
						d74 = snap538
						d75 = snap539
						d76 = snap540
						d77 = snap541
						d78 = snap542
						d79 = snap543
						d107 = snap544
						d136 = snap545
						d137 = snap546
						d138 = snap547
						d139 = snap548
						d172 = snap549
						d206 = snap550
						d207 = snap551
						d208 = snap552
						d211 = snap553
						d212 = snap554
						d213 = snap555
						d253 = snap556
						d254 = snap557
						d255 = snap558
						d298 = snap559
						d299 = snap560
						d300 = snap561
						d346 = snap562
						d347 = snap563
						d348 = snap564
						d349 = snap565
						d399 = snap566
						d400 = snap567
						d401 = snap568
						d454 = snap569
						d455 = snap570
						d456 = snap571
						d457 = snap572
						d514 = snap573
						d515 = snap574
						d516 = snap575
						d517 = snap576
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[19].Render = func() JITValueDesc {
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
						return result
					}
					bbs[19].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d454)
					if d454.Loc == LocImm {
						d578 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d454.Imm.Float() < -9.223372036854776e+18)}
					} else {
						r18 := ctx.AllocRegExcept(d454.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(14114281232179134464))
						ctx.EmitCmpFloat64(ctx.ScratchReg, d454.Reg)
						d578 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondUnsignedAbove}
						ctx.BindReg(r18, &d578)
					}
					d579 = d578
					ctx.EnsureDesc(&d579)
					if d579.Loc != LocImm && d579.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d579.Loc == LocImm {
						if d579.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[18].Render()
					}
					ctx.EmitJump(d579.Condition, lbl12)
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
					}
					ctx.FreeDesc(&d578)
					ctx.FlushRegisterMoves()
					if !bbs[18].Rendered {
						snap580 := d1
						snap581 := d2
						snap582 := d3
						snap583 := d4
						snap584 := d5
						snap585 := d6
						snap586 := d7
						snap587 := d8
						snap588 := d9
						snap589 := d10
						snap590 := d11
						snap591 := d12
						snap592 := d26
						snap593 := d29
						snap594 := d30
						snap595 := d31
						snap596 := d49
						snap597 := d50
						snap598 := d51
						snap599 := d52
						snap600 := d74
						snap601 := d75
						snap602 := d76
						snap603 := d77
						snap604 := d78
						snap605 := d79
						snap606 := d107
						snap607 := d136
						snap608 := d137
						snap609 := d138
						snap610 := d139
						snap611 := d172
						snap612 := d206
						snap613 := d207
						snap614 := d208
						snap615 := d211
						snap616 := d212
						snap617 := d213
						snap618 := d253
						snap619 := d254
						snap620 := d255
						snap621 := d298
						snap622 := d299
						snap623 := d300
						snap624 := d346
						snap625 := d347
						snap626 := d348
						snap627 := d349
						snap628 := d399
						snap629 := d400
						snap630 := d401
						snap631 := d454
						snap632 := d455
						snap633 := d456
						snap634 := d457
						snap635 := d514
						snap636 := d515
						snap637 := d516
						snap638 := d517
						snap639 := d578
						snap640 := d579
						alloc641 := ctx.SnapshotAllocState()
						bbs[18].Render()
						ctx.RestoreAllocState(alloc641)
						d1 = snap580
						d2 = snap581
						d3 = snap582
						d4 = snap583
						d5 = snap584
						d6 = snap585
						d7 = snap586
						d8 = snap587
						d9 = snap588
						d10 = snap589
						d11 = snap590
						d12 = snap591
						d26 = snap592
						d29 = snap593
						d30 = snap594
						d31 = snap595
						d49 = snap596
						d50 = snap597
						d51 = snap598
						d52 = snap599
						d74 = snap600
						d75 = snap601
						d76 = snap602
						d77 = snap603
						d78 = snap604
						d79 = snap605
						d107 = snap606
						d136 = snap607
						d137 = snap608
						d138 = snap609
						d139 = snap610
						d172 = snap611
						d206 = snap612
						d207 = snap613
						d208 = snap614
						d211 = snap615
						d212 = snap616
						d213 = snap617
						d253 = snap618
						d254 = snap619
						d255 = snap620
						d298 = snap621
						d299 = snap622
						d300 = snap623
						d346 = snap624
						d347 = snap625
						d348 = snap626
						d349 = snap627
						d399 = snap628
						d400 = snap629
						d401 = snap630
						d454 = snap631
						d455 = snap632
						d456 = snap633
						d457 = snap634
						d514 = snap635
						d515 = snap636
						d516 = snap637
						d517 = snap638
						d578 = snap639
						d579 = snap640
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[20].Render = func() JITValueDesc {
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
						return result
					}
					bbs[20].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_20 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl21)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d642 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(scratch, d5.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d642 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d642)
					}
					if d642.Loc == LocReg && d5.Loc == LocReg && d642.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.EnsureDesc(&d642)
					ctx.EmitStoreToStack(d642, int32(bbs[12].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d642)
					ctx.SyncDesc(&d348)
					if d348.Loc == LocReg || d348.Loc == LocFPReg {
						ctx.ProtectReg(d348.Reg)
					} else if d348.Loc == LocRegPair {
						ctx.ProtectReg(d348.Reg)
						ctx.ProtectReg(d348.Reg2)
					}
					d643 = d348
					if d643.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d643)
					ctx.EmitStoreToStack(d643, int32(bbs[12].PhiBase)+int32(0))
					if d348.Loc == LocReg || d348.Loc == LocFPReg {
						ctx.UnprotectReg(d348.Reg)
					} else if d348.Loc == LocRegPair {
						ctx.UnprotectReg(d348.Reg)
						ctx.UnprotectReg(d348.Reg2)
					}
					return bbs[12].Render()
					return result
				}
				bbs[21].Render = func() JITValueDesc {
					if bbs[21].Rendered {
						ctx.EmitJmp(lbl22)
						return result
					}
					bbs[21].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_21 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d644 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() == -1)}
					} else {
						r19 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, -1)
						d644 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondEqual}
						ctx.BindReg(r19, &d644)
					}
					d645 = d644
					ctx.EnsureDesc(&d645)
					if d645.Loc != LocImm && d645.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d645.Loc == LocImm {
						if d645.Imm.Bool() {
							return bbs[24].Render()
						}
						return bbs[23].Render()
					}
					ctx.EmitJump(d645.Condition, lbl25)
					if bbs[23].Rendered {
						ctx.EmitJmp(lbl24)
					}
					ctx.FreeDesc(&d644)
					ctx.FlushRegisterMoves()
					if !bbs[23].Rendered {
						snap646 := d1
						snap647 := d2
						snap648 := d3
						snap649 := d4
						snap650 := d5
						snap651 := d6
						snap652 := d7
						snap653 := d8
						snap654 := d9
						snap655 := d10
						snap656 := d11
						snap657 := d12
						snap658 := d26
						snap659 := d29
						snap660 := d30
						snap661 := d31
						snap662 := d49
						snap663 := d50
						snap664 := d51
						snap665 := d52
						snap666 := d74
						snap667 := d75
						snap668 := d76
						snap669 := d77
						snap670 := d78
						snap671 := d79
						snap672 := d107
						snap673 := d136
						snap674 := d137
						snap675 := d138
						snap676 := d139
						snap677 := d172
						snap678 := d206
						snap679 := d207
						snap680 := d208
						snap681 := d211
						snap682 := d212
						snap683 := d213
						snap684 := d253
						snap685 := d254
						snap686 := d255
						snap687 := d298
						snap688 := d299
						snap689 := d300
						snap690 := d346
						snap691 := d347
						snap692 := d348
						snap693 := d349
						snap694 := d399
						snap695 := d400
						snap696 := d401
						snap697 := d454
						snap698 := d455
						snap699 := d456
						snap700 := d457
						snap701 := d514
						snap702 := d515
						snap703 := d516
						snap704 := d517
						snap705 := d578
						snap706 := d579
						snap707 := d642
						snap708 := d643
						snap709 := d644
						snap710 := d645
						alloc711 := ctx.SnapshotAllocState()
						bbs[23].Render()
						ctx.RestoreAllocState(alloc711)
						d1 = snap646
						d2 = snap647
						d3 = snap648
						d4 = snap649
						d5 = snap650
						d6 = snap651
						d7 = snap652
						d8 = snap653
						d9 = snap654
						d10 = snap655
						d11 = snap656
						d12 = snap657
						d26 = snap658
						d29 = snap659
						d30 = snap660
						d31 = snap661
						d49 = snap662
						d50 = snap663
						d51 = snap664
						d52 = snap665
						d74 = snap666
						d75 = snap667
						d76 = snap668
						d77 = snap669
						d78 = snap670
						d79 = snap671
						d107 = snap672
						d136 = snap673
						d137 = snap674
						d138 = snap675
						d139 = snap676
						d172 = snap677
						d206 = snap678
						d207 = snap679
						d208 = snap680
						d211 = snap681
						d212 = snap682
						d213 = snap683
						d253 = snap684
						d254 = snap685
						d255 = snap686
						d298 = snap687
						d299 = snap688
						d300 = snap689
						d346 = snap690
						d347 = snap691
						d348 = snap692
						d349 = snap693
						d399 = snap694
						d400 = snap695
						d401 = snap696
						d454 = snap697
						d455 = snap698
						d456 = snap699
						d457 = snap700
						d514 = snap701
						d515 = snap702
						d516 = snap703
						d517 = snap704
						d578 = snap705
						d579 = snap706
						d642 = snap707
						d643 = snap708
						d644 = snap709
						d645 = snap710
					}
					if !bbs[24].Rendered {
						return bbs[24].Render()
					}
					return result
					return result
				}
				bbs[22].Render = func() JITValueDesc {
					if bbs[22].Rendered {
						ctx.EmitJmp(lbl23)
						return result
					}
					bbs[22].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_22 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d712 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() != 0)}
					} else {
						r20 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 0)
						d712 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondNotEqual}
						ctx.BindReg(r20, &d712)
					}
					d713 = d712
					ctx.EnsureDesc(&d713)
					if d713.Loc != LocImm && d713.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d713.Loc == LocImm {
						if d713.Imm.Bool() {
							return bbs[21].Render()
						}
						return bbs[20].Render()
					}
					ctx.EmitJump(d713.Condition, lbl22)
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
					}
					ctx.FreeDesc(&d712)
					ctx.FlushRegisterMoves()
					if !bbs[20].Rendered {
						snap714 := d1
						snap715 := d2
						snap716 := d3
						snap717 := d4
						snap718 := d5
						snap719 := d6
						snap720 := d7
						snap721 := d8
						snap722 := d9
						snap723 := d10
						snap724 := d11
						snap725 := d12
						snap726 := d26
						snap727 := d29
						snap728 := d30
						snap729 := d31
						snap730 := d49
						snap731 := d50
						snap732 := d51
						snap733 := d52
						snap734 := d74
						snap735 := d75
						snap736 := d76
						snap737 := d77
						snap738 := d78
						snap739 := d79
						snap740 := d107
						snap741 := d136
						snap742 := d137
						snap743 := d138
						snap744 := d139
						snap745 := d172
						snap746 := d206
						snap747 := d207
						snap748 := d208
						snap749 := d211
						snap750 := d212
						snap751 := d213
						snap752 := d253
						snap753 := d254
						snap754 := d255
						snap755 := d298
						snap756 := d299
						snap757 := d300
						snap758 := d346
						snap759 := d347
						snap760 := d348
						snap761 := d349
						snap762 := d399
						snap763 := d400
						snap764 := d401
						snap765 := d454
						snap766 := d455
						snap767 := d456
						snap768 := d457
						snap769 := d514
						snap770 := d515
						snap771 := d516
						snap772 := d517
						snap773 := d578
						snap774 := d579
						snap775 := d642
						snap776 := d643
						snap777 := d644
						snap778 := d645
						snap779 := d712
						snap780 := d713
						alloc781 := ctx.SnapshotAllocState()
						bbs[20].Render()
						ctx.RestoreAllocState(alloc781)
						d1 = snap714
						d2 = snap715
						d3 = snap716
						d4 = snap717
						d5 = snap718
						d6 = snap719
						d7 = snap720
						d8 = snap721
						d9 = snap722
						d10 = snap723
						d11 = snap724
						d12 = snap725
						d26 = snap726
						d29 = snap727
						d30 = snap728
						d31 = snap729
						d49 = snap730
						d50 = snap731
						d51 = snap732
						d52 = snap733
						d74 = snap734
						d75 = snap735
						d76 = snap736
						d77 = snap737
						d78 = snap738
						d79 = snap739
						d107 = snap740
						d136 = snap741
						d137 = snap742
						d138 = snap743
						d139 = snap744
						d172 = snap745
						d206 = snap746
						d207 = snap747
						d208 = snap748
						d211 = snap749
						d212 = snap750
						d213 = snap751
						d253 = snap752
						d254 = snap753
						d255 = snap754
						d298 = snap755
						d299 = snap756
						d300 = snap757
						d346 = snap758
						d347 = snap759
						d348 = snap760
						d349 = snap761
						d399 = snap762
						d400 = snap763
						d401 = snap764
						d454 = snap765
						d455 = snap766
						d456 = snap767
						d457 = snap768
						d514 = snap769
						d515 = snap770
						d516 = snap771
						d517 = snap772
						d578 = snap773
						d579 = snap774
						d642 = snap775
						d643 = snap776
						d644 = snap777
						d645 = snap778
						d712 = snap779
						d713 = snap780
					}
					if !bbs[21].Rendered {
						return bbs[21].Render()
					}
					return result
					return result
				}
				bbs[23].Render = func() JITValueDesc {
					if bbs[23].Rendered {
						ctx.EmitJmp(lbl24)
						return result
					}
					bbs[23].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_23 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d348)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDescsTogether(&d348, &d6)
					if d348.Loc == LocImm && d6.Loc == LocImm {
						d782 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d348.Imm.Int() / d6.Imm.Int())}
					} else {
						d782 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntDiv), []JITValueDesc{d348, d6}, 1)
					}
					if d782.Loc == LocReg && d348.Loc == LocReg && d782.Reg == d348.Reg {
						ctx.TransferReg(d348.Reg)
						d348.Loc = LocNone
					}
					ctx.EnsureDesc(&d782)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDescsTogether(&d782, &d4)
					if d782.Loc == LocImm && d4.Loc == LocImm {
						d783 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d782.Imm.Int() != d4.Imm.Int())}
					} else if d4.Loc == LocImm {
						r21 := ctx.AllocReg()
						if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d782.Reg, int32(d4.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitCmpInt64(d782.Reg, ctx.ScratchReg)
						}
						d783 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondNotEqual}
						ctx.BindReg(r21, &d783)
					} else if d782.Loc == LocImm {
						r22 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d782.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d4.Reg)
						d783 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondNotEqual}
						ctx.BindReg(r22, &d783)
					} else {
						r23 := ctx.AllocReg()
						ctx.EmitCmpInt64(d782.Reg, d4.Reg)
						d783 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondNotEqual}
						ctx.BindReg(r23, &d783)
					}
					ctx.FreeDesc(&d782)
					d784 = d783
					ctx.EnsureDesc(&d784)
					if d784.Loc != LocImm && d784.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d784.Loc == LocImm {
						if d784.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[20].Render()
					}
					ctx.EmitJump(d784.Condition, lbl12)
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
					}
					ctx.FreeDesc(&d783)
					ctx.FlushRegisterMoves()
					if !bbs[20].Rendered {
						snap785 := d1
						snap786 := d2
						snap787 := d3
						snap788 := d4
						snap789 := d5
						snap790 := d6
						snap791 := d7
						snap792 := d8
						snap793 := d9
						snap794 := d10
						snap795 := d11
						snap796 := d12
						snap797 := d26
						snap798 := d29
						snap799 := d30
						snap800 := d31
						snap801 := d49
						snap802 := d50
						snap803 := d51
						snap804 := d52
						snap805 := d74
						snap806 := d75
						snap807 := d76
						snap808 := d77
						snap809 := d78
						snap810 := d79
						snap811 := d107
						snap812 := d136
						snap813 := d137
						snap814 := d138
						snap815 := d139
						snap816 := d172
						snap817 := d206
						snap818 := d207
						snap819 := d208
						snap820 := d211
						snap821 := d212
						snap822 := d213
						snap823 := d253
						snap824 := d254
						snap825 := d255
						snap826 := d298
						snap827 := d299
						snap828 := d300
						snap829 := d346
						snap830 := d347
						snap831 := d348
						snap832 := d349
						snap833 := d399
						snap834 := d400
						snap835 := d401
						snap836 := d454
						snap837 := d455
						snap838 := d456
						snap839 := d457
						snap840 := d514
						snap841 := d515
						snap842 := d516
						snap843 := d517
						snap844 := d578
						snap845 := d579
						snap846 := d642
						snap847 := d643
						snap848 := d644
						snap849 := d645
						snap850 := d712
						snap851 := d713
						snap852 := d782
						snap853 := d783
						snap854 := d784
						alloc855 := ctx.SnapshotAllocState()
						bbs[20].Render()
						ctx.RestoreAllocState(alloc855)
						d1 = snap785
						d2 = snap786
						d3 = snap787
						d4 = snap788
						d5 = snap789
						d6 = snap790
						d7 = snap791
						d8 = snap792
						d9 = snap793
						d10 = snap794
						d11 = snap795
						d12 = snap796
						d26 = snap797
						d29 = snap798
						d30 = snap799
						d31 = snap800
						d49 = snap801
						d50 = snap802
						d51 = snap803
						d52 = snap804
						d74 = snap805
						d75 = snap806
						d76 = snap807
						d77 = snap808
						d78 = snap809
						d79 = snap810
						d107 = snap811
						d136 = snap812
						d137 = snap813
						d138 = snap814
						d139 = snap815
						d172 = snap816
						d206 = snap817
						d207 = snap818
						d208 = snap819
						d211 = snap820
						d212 = snap821
						d213 = snap822
						d253 = snap823
						d254 = snap824
						d255 = snap825
						d298 = snap826
						d299 = snap827
						d300 = snap828
						d346 = snap829
						d347 = snap830
						d348 = snap831
						d349 = snap832
						d399 = snap833
						d400 = snap834
						d401 = snap835
						d454 = snap836
						d455 = snap837
						d456 = snap838
						d457 = snap839
						d514 = snap840
						d515 = snap841
						d516 = snap842
						d517 = snap843
						d578 = snap844
						d579 = snap845
						d642 = snap846
						d643 = snap847
						d644 = snap848
						d645 = snap849
						d712 = snap850
						d713 = snap851
						d782 = snap852
						d783 = snap853
						d784 = snap854
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[24].Render = func() JITValueDesc {
					if bbs[24].Rendered {
						ctx.EmitJmp(lbl25)
						return result
					}
					bbs[24].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_24 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl25)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d856 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() == -9223372036854775808)}
					} else {
						r24 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0x8000000000000000)
						ctx.EmitCmpInt64(d4.Reg, ctx.ScratchReg)
						d856 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondEqual}
						ctx.BindReg(r24, &d856)
					}
					d857 = d856
					ctx.EnsureDesc(&d857)
					if d857.Loc != LocImm && d857.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d857.Loc == LocImm {
						if d857.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[23].Render()
					}
					ctx.EmitJump(d857.Condition, lbl12)
					if bbs[23].Rendered {
						ctx.EmitJmp(lbl24)
					}
					ctx.FreeDesc(&d856)
					ctx.FlushRegisterMoves()
					if !bbs[23].Rendered {
						snap858 := d1
						snap859 := d2
						snap860 := d3
						snap861 := d4
						snap862 := d5
						snap863 := d6
						snap864 := d7
						snap865 := d8
						snap866 := d9
						snap867 := d10
						snap868 := d11
						snap869 := d12
						snap870 := d26
						snap871 := d29
						snap872 := d30
						snap873 := d31
						snap874 := d49
						snap875 := d50
						snap876 := d51
						snap877 := d52
						snap878 := d74
						snap879 := d75
						snap880 := d76
						snap881 := d77
						snap882 := d78
						snap883 := d79
						snap884 := d107
						snap885 := d136
						snap886 := d137
						snap887 := d138
						snap888 := d139
						snap889 := d172
						snap890 := d206
						snap891 := d207
						snap892 := d208
						snap893 := d211
						snap894 := d212
						snap895 := d213
						snap896 := d253
						snap897 := d254
						snap898 := d255
						snap899 := d298
						snap900 := d299
						snap901 := d300
						snap902 := d346
						snap903 := d347
						snap904 := d348
						snap905 := d349
						snap906 := d399
						snap907 := d400
						snap908 := d401
						snap909 := d454
						snap910 := d455
						snap911 := d456
						snap912 := d457
						snap913 := d514
						snap914 := d515
						snap915 := d516
						snap916 := d517
						snap917 := d578
						snap918 := d579
						snap919 := d642
						snap920 := d643
						snap921 := d644
						snap922 := d645
						snap923 := d712
						snap924 := d713
						snap925 := d782
						snap926 := d783
						snap927 := d784
						snap928 := d856
						snap929 := d857
						alloc930 := ctx.SnapshotAllocState()
						bbs[23].Render()
						ctx.RestoreAllocState(alloc930)
						d1 = snap858
						d2 = snap859
						d3 = snap860
						d4 = snap861
						d5 = snap862
						d6 = snap863
						d7 = snap864
						d8 = snap865
						d9 = snap866
						d10 = snap867
						d11 = snap868
						d12 = snap869
						d26 = snap870
						d29 = snap871
						d30 = snap872
						d31 = snap873
						d49 = snap874
						d50 = snap875
						d51 = snap876
						d52 = snap877
						d74 = snap878
						d75 = snap879
						d76 = snap880
						d77 = snap881
						d78 = snap882
						d79 = snap883
						d107 = snap884
						d136 = snap885
						d137 = snap886
						d138 = snap887
						d139 = snap888
						d172 = snap889
						d206 = snap890
						d207 = snap891
						d208 = snap892
						d211 = snap893
						d212 = snap894
						d213 = snap895
						d253 = snap896
						d254 = snap897
						d255 = snap898
						d298 = snap899
						d299 = snap900
						d300 = snap901
						d346 = snap902
						d347 = snap903
						d348 = snap904
						d349 = snap905
						d399 = snap906
						d400 = snap907
						d401 = snap908
						d454 = snap909
						d455 = snap910
						d456 = snap911
						d457 = snap912
						d514 = snap913
						d515 = snap914
						d516 = snap915
						d517 = snap916
						d578 = snap917
						d579 = snap918
						d642 = snap919
						d643 = snap920
						d644 = snap921
						d645 = snap922
						d712 = snap923
						d713 = snap924
						d782 = snap925
						d783 = snap926
						d784 = snap927
						d856 = snap928
						d857 = snap929
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[25].Render = func() JITValueDesc {
					if bbs[25].Rendered {
						ctx.EmitJmp(lbl26)
						return result
					}
					bbs[25].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_25 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						ctx.EmitMakeInt(result, d4)
					} else {
						ctx.EmitMovToReg(result.Reg2, d4)
						d931 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d931)
						if d4.Loc == LocReg && d4.Reg != result.Reg2 {
							ctx.FreeReg(d4.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[26].Render = func() JITValueDesc {
					if bbs[26].Rendered {
						ctx.EmitJmp(lbl27)
						return result
					}
					bbs[26].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_26 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					d932 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d932)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(1)}, int32(bbs[27].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[27].PhiBase)+int32(16))
					return bbs[27].Render()
					return result
				}
				bbs[27].Render = func() JITValueDesc {
					if bbs[27].Rendered {
						ctx.EmitJmp(lbl28)
						return result
					}
					bbs[27].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_27 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl28)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d933 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitMovRegReg(scratch, d8.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d933 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d933)
					}
					if d933.Loc == LocReg && d8.Loc == LocReg && d933.Reg == d8.Reg {
						ctx.TransferReg(d8.Reg)
						d8.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d933)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d933)
					ctx.EnsureDesc(&d932)
					ctx.EnsureDescsTogether(&d933, &d932)
					if d933.Loc == LocImm && d932.Loc == LocImm {
						d934 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d933.Imm.Int() < d932.Imm.Int())}
					} else if d932.Loc == LocImm {
						r25 := ctx.AllocRegExcept(d933.Reg)
						if d932.Imm.Int() >= -2147483648 && d932.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d933.Reg, int32(d932.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d932.Imm.Int()))
							ctx.EmitCmpInt64(d933.Reg, ctx.ScratchReg)
						}
						d934 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedLess}
						ctx.BindReg(r25, &d934)
					} else if d933.Loc == LocImm {
						r26 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d933.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d932.Reg)
						d934 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedLess}
						ctx.BindReg(r26, &d934)
					} else {
						r27 := ctx.AllocRegExcept(d933.Reg)
						ctx.EmitCmpInt64(d933.Reg, d932.Reg)
						d934 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
						ctx.BindReg(r27, &d934)
					}
					d935 = d934
					ctx.EnsureDesc(&d935)
					if d935.Loc != LocImm && d935.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d935.Loc == LocImm {
						if d935.Imm.Bool() {
							return bbs[28].Render()
						}
						return bbs[29].Render()
					}
					ctx.EmitJump(d935.Condition, lbl29)
					if bbs[29].Rendered {
						ctx.EmitJmp(lbl30)
					}
					ctx.FreeDesc(&d934)
					ctx.FlushRegisterMoves()
					if !bbs[29].Rendered {
						snap936 := d1
						snap937 := d2
						snap938 := d3
						snap939 := d4
						snap940 := d5
						snap941 := d6
						snap942 := d7
						snap943 := d8
						snap944 := d9
						snap945 := d10
						snap946 := d11
						snap947 := d12
						snap948 := d26
						snap949 := d29
						snap950 := d30
						snap951 := d31
						snap952 := d49
						snap953 := d50
						snap954 := d51
						snap955 := d52
						snap956 := d74
						snap957 := d75
						snap958 := d76
						snap959 := d77
						snap960 := d78
						snap961 := d79
						snap962 := d107
						snap963 := d136
						snap964 := d137
						snap965 := d138
						snap966 := d139
						snap967 := d172
						snap968 := d206
						snap969 := d207
						snap970 := d208
						snap971 := d211
						snap972 := d212
						snap973 := d213
						snap974 := d253
						snap975 := d254
						snap976 := d255
						snap977 := d298
						snap978 := d299
						snap979 := d300
						snap980 := d346
						snap981 := d347
						snap982 := d348
						snap983 := d349
						snap984 := d399
						snap985 := d400
						snap986 := d401
						snap987 := d454
						snap988 := d455
						snap989 := d456
						snap990 := d457
						snap991 := d514
						snap992 := d515
						snap993 := d516
						snap994 := d517
						snap995 := d578
						snap996 := d579
						snap997 := d642
						snap998 := d643
						snap999 := d644
						snap1000 := d645
						snap1001 := d712
						snap1002 := d713
						snap1003 := d782
						snap1004 := d783
						snap1005 := d784
						snap1006 := d856
						snap1007 := d857
						snap1008 := d931
						snap1009 := d932
						snap1010 := d933
						snap1011 := d934
						snap1012 := d935
						alloc1013 := ctx.SnapshotAllocState()
						bbs[29].Render()
						ctx.RestoreAllocState(alloc1013)
						d1 = snap936
						d2 = snap937
						d3 = snap938
						d4 = snap939
						d5 = snap940
						d6 = snap941
						d7 = snap942
						d8 = snap943
						d9 = snap944
						d10 = snap945
						d11 = snap946
						d12 = snap947
						d26 = snap948
						d29 = snap949
						d30 = snap950
						d31 = snap951
						d49 = snap952
						d50 = snap953
						d51 = snap954
						d52 = snap955
						d74 = snap956
						d75 = snap957
						d76 = snap958
						d77 = snap959
						d78 = snap960
						d79 = snap961
						d107 = snap962
						d136 = snap963
						d137 = snap964
						d138 = snap965
						d139 = snap966
						d172 = snap967
						d206 = snap968
						d207 = snap969
						d208 = snap970
						d211 = snap971
						d212 = snap972
						d213 = snap973
						d253 = snap974
						d254 = snap975
						d255 = snap976
						d298 = snap977
						d299 = snap978
						d300 = snap979
						d346 = snap980
						d347 = snap981
						d348 = snap982
						d349 = snap983
						d399 = snap984
						d400 = snap985
						d401 = snap986
						d454 = snap987
						d455 = snap988
						d456 = snap989
						d457 = snap990
						d514 = snap991
						d515 = snap992
						d516 = snap993
						d517 = snap994
						d578 = snap995
						d579 = snap996
						d642 = snap997
						d643 = snap998
						d644 = snap999
						d645 = snap1000
						d712 = snap1001
						d713 = snap1002
						d782 = snap1003
						d783 = snap1004
						d784 = snap1005
						d856 = snap1006
						d857 = snap1007
						d931 = snap1008
						d932 = snap1009
						d933 = snap1010
						d934 = snap1011
						d935 = snap1012
					}
					if !bbs[28].Rendered {
						return bbs[28].Render()
					}
					return result
					return result
				}
				bbs[28].Render = func() JITValueDesc {
					if bbs[28].Rendered {
						ctx.EmitJmp(lbl29)
						return result
					}
					bbs[28].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_28 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl29)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d933)
					if d933.Loc == LocImm {
						idx := int(d933.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d1014 = args[idx]
						d1014.ID = 0
					} else {
						ctx.EnsureDesc(&d933)
						dynamicArgOff1015 = ctx.AllocStack(16)
						ctx.ProtectReg(d933.Reg)
						lbl41 := ctx.ReserveLabel()
						lbl42 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d933.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl42)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d933.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff1015))
							ctx.EmitJmp(lbl41)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl42)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff1015))
						ctx.MarkLabel(lbl41)
						ctx.UnprotectReg(d933.Reg)
						d1014 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff1015), Rooted: true}
					}
					d1016 = ctx.EmitFloatDesc(d1014)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d1016)
					ctx.EnsureDescsTogether(&d7, &d1016)
					if d7.Loc == LocImm && d1016.Loc == LocImm {
						d1017 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d7.Imm.Float() * d1016.Imm.Float())}
					} else if d7.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1016.Reg)
						_, xBits := d7.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d1016.Reg)
						d1017 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d1017)
					} else if d1016.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegReg(scratch, d7.Reg)
						_, yBits := d1016.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitMulFloat64(scratch, ctx.ScratchReg)
						d1017 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d1017)
					} else {
						r28 := ctx.AllocRegExcept(d7.Reg, d1016.Reg)
						ctx.EmitMovRegReg(r28, d7.Reg)
						ctx.EmitMulFloat64(r28, d1016.Reg)
						d1017 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r28}
						ctx.BindReg(r28, &d1017)
					}
					if d1017.Loc == LocReg && d7.Loc == LocReg && d1017.Reg == d7.Reg {
						ctx.TransferReg(d7.Reg)
						d7.Loc = LocNone
					}
					ctx.EnsureDesc(&d1017)
					ctx.EmitStoreToStack(d1017, int32(bbs[27].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d1017)
					ctx.FreeDesc(&d1016)
					ctx.SyncDesc(&d933)
					if d933.Loc == LocReg || d933.Loc == LocFPReg {
						ctx.ProtectReg(d933.Reg)
					} else if d933.Loc == LocRegPair {
						ctx.ProtectReg(d933.Reg)
						ctx.ProtectReg(d933.Reg2)
					}
					d1018 = d933
					if d1018.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d1018)
					ctx.EmitStoreToStack(d1018, int32(bbs[27].PhiBase)+int32(16))
					if d933.Loc == LocReg || d933.Loc == LocFPReg {
						ctx.UnprotectReg(d933.Reg)
					} else if d933.Loc == LocRegPair {
						ctx.UnprotectReg(d933.Reg)
						ctx.UnprotectReg(d933.Reg2)
					}
					return bbs[27].Render()
					return result
				}
				bbs[29].Render = func() JITValueDesc {
					if bbs[29].Rendered {
						ctx.EmitJmp(lbl30)
						return result
					}
					bbs[29].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_29 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl30)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					d1019 = d2
					ctx.EnsureDesc(&d1019)
					if d1019.Loc != LocImm && d1019.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d1019.Loc == LocImm {
						if d1019.Imm.Bool() {
							return bbs[33].Render()
						}
						return bbs[31].Render()
					}
					ctx.EmitCmpRegImm32(d1019.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl34)
					if bbs[31].Rendered {
						ctx.EmitJmp(lbl32)
					}
					ctx.FlushRegisterMoves()
					if !bbs[31].Rendered {
						snap1020 := d1
						snap1021 := d2
						snap1022 := d3
						snap1023 := d4
						snap1024 := d5
						snap1025 := d6
						snap1026 := d7
						snap1027 := d8
						snap1028 := d9
						snap1029 := d10
						snap1030 := d11
						snap1031 := d12
						snap1032 := d26
						snap1033 := d29
						snap1034 := d30
						snap1035 := d31
						snap1036 := d49
						snap1037 := d50
						snap1038 := d51
						snap1039 := d52
						snap1040 := d74
						snap1041 := d75
						snap1042 := d76
						snap1043 := d77
						snap1044 := d78
						snap1045 := d79
						snap1046 := d107
						snap1047 := d136
						snap1048 := d137
						snap1049 := d138
						snap1050 := d139
						snap1051 := d172
						snap1052 := d206
						snap1053 := d207
						snap1054 := d208
						snap1055 := d211
						snap1056 := d212
						snap1057 := d213
						snap1058 := d253
						snap1059 := d254
						snap1060 := d255
						snap1061 := d298
						snap1062 := d299
						snap1063 := d300
						snap1064 := d346
						snap1065 := d347
						snap1066 := d348
						snap1067 := d349
						snap1068 := d399
						snap1069 := d400
						snap1070 := d401
						snap1071 := d454
						snap1072 := d455
						snap1073 := d456
						snap1074 := d457
						snap1075 := d514
						snap1076 := d515
						snap1077 := d516
						snap1078 := d517
						snap1079 := d578
						snap1080 := d579
						snap1081 := d642
						snap1082 := d643
						snap1083 := d644
						snap1084 := d645
						snap1085 := d712
						snap1086 := d713
						snap1087 := d782
						snap1088 := d783
						snap1089 := d784
						snap1090 := d856
						snap1091 := d857
						snap1092 := d931
						snap1093 := d932
						snap1094 := d933
						snap1095 := d934
						snap1096 := d935
						snap1097 := d1014
						snap1098 := d1016
						snap1099 := d1017
						snap1100 := d1018
						snap1101 := d1019
						alloc1102 := ctx.SnapshotAllocState()
						bbs[31].Render()
						ctx.RestoreAllocState(alloc1102)
						d1 = snap1020
						d2 = snap1021
						d3 = snap1022
						d4 = snap1023
						d5 = snap1024
						d6 = snap1025
						d7 = snap1026
						d8 = snap1027
						d9 = snap1028
						d10 = snap1029
						d11 = snap1030
						d12 = snap1031
						d26 = snap1032
						d29 = snap1033
						d30 = snap1034
						d31 = snap1035
						d49 = snap1036
						d50 = snap1037
						d51 = snap1038
						d52 = snap1039
						d74 = snap1040
						d75 = snap1041
						d76 = snap1042
						d77 = snap1043
						d78 = snap1044
						d79 = snap1045
						d107 = snap1046
						d136 = snap1047
						d137 = snap1048
						d138 = snap1049
						d139 = snap1050
						d172 = snap1051
						d206 = snap1052
						d207 = snap1053
						d208 = snap1054
						d211 = snap1055
						d212 = snap1056
						d213 = snap1057
						d253 = snap1058
						d254 = snap1059
						d255 = snap1060
						d298 = snap1061
						d299 = snap1062
						d300 = snap1063
						d346 = snap1064
						d347 = snap1065
						d348 = snap1066
						d349 = snap1067
						d399 = snap1068
						d400 = snap1069
						d401 = snap1070
						d454 = snap1071
						d455 = snap1072
						d456 = snap1073
						d457 = snap1074
						d514 = snap1075
						d515 = snap1076
						d516 = snap1077
						d517 = snap1078
						d578 = snap1079
						d579 = snap1080
						d642 = snap1081
						d643 = snap1082
						d644 = snap1083
						d645 = snap1084
						d712 = snap1085
						d713 = snap1086
						d782 = snap1087
						d783 = snap1088
						d784 = snap1089
						d856 = snap1090
						d857 = snap1091
						d931 = snap1092
						d932 = snap1093
						d933 = snap1094
						d934 = snap1095
						d935 = snap1096
						d1014 = snap1097
						d1016 = snap1098
						d1017 = snap1099
						d1018 = snap1100
						d1019 = snap1101
					}
					if !bbs[33].Rendered {
						return bbs[33].Render()
					}
					return result
					return result
				}
				bbs[30].Render = func() JITValueDesc {
					if bbs[30].Rendered {
						ctx.EmitJmp(lbl31)
						return result
					}
					bbs[30].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_30 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl31)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d1103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d7.Imm.Float()))}
					} else {
						r29 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r29, d7.Reg)
						d1103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r29}
						ctx.BindReg(r29, &d1103)
					}
					ctx.EnsureDesc(&d1103)
					if d1103.Loc == LocImm {
						ctx.EmitMakeInt(result, d1103)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1103)
						d1104 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d1104)
						if d1103.Loc == LocReg && d1103.Reg != result.Reg2 {
							ctx.FreeReg(d1103.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[31].Render = func() JITValueDesc {
					if bbs[31].Rendered {
						ctx.EmitJmp(lbl32)
						return result
					}
					bbs[31].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_31 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl32)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						ctx.EmitMakeFloat(result, d7)
					} else {
						ctx.EmitMovToReg(result.Reg2, d7)
						d1105 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d1105)
						if d7.Loc == LocReg && d7.Reg != result.Reg2 {
							ctx.FreeReg(d7.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[32].Render = func() JITValueDesc {
					if bbs[32].Rendered {
						ctx.EmitJmp(lbl33)
						return result
					}
					bbs[32].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_32 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl33)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d1106 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.Float() < 9.223372036854776e+18)}
					} else {
						r30 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4890909195324358656))
						ctx.EmitCmpFloat64(ctx.ScratchReg, d7.Reg)
						d1106 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondUnsignedAbove}
						ctx.BindReg(r30, &d1106)
					}
					d1107 = d1106
					ctx.EnsureDesc(&d1107)
					if d1107.Loc != LocImm && d1107.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d1107.Loc == LocImm {
						if d1107.Imm.Bool() {
							return bbs[30].Render()
						}
						return bbs[31].Render()
					}
					ctx.EmitJump(d1107.Condition, lbl31)
					if bbs[31].Rendered {
						ctx.EmitJmp(lbl32)
					}
					ctx.FreeDesc(&d1106)
					ctx.FlushRegisterMoves()
					if !bbs[31].Rendered {
						snap1108 := d1
						snap1109 := d2
						snap1110 := d3
						snap1111 := d4
						snap1112 := d5
						snap1113 := d6
						snap1114 := d7
						snap1115 := d8
						snap1116 := d9
						snap1117 := d10
						snap1118 := d11
						snap1119 := d12
						snap1120 := d26
						snap1121 := d29
						snap1122 := d30
						snap1123 := d31
						snap1124 := d49
						snap1125 := d50
						snap1126 := d51
						snap1127 := d52
						snap1128 := d74
						snap1129 := d75
						snap1130 := d76
						snap1131 := d77
						snap1132 := d78
						snap1133 := d79
						snap1134 := d107
						snap1135 := d136
						snap1136 := d137
						snap1137 := d138
						snap1138 := d139
						snap1139 := d172
						snap1140 := d206
						snap1141 := d207
						snap1142 := d208
						snap1143 := d211
						snap1144 := d212
						snap1145 := d213
						snap1146 := d253
						snap1147 := d254
						snap1148 := d255
						snap1149 := d298
						snap1150 := d299
						snap1151 := d300
						snap1152 := d346
						snap1153 := d347
						snap1154 := d348
						snap1155 := d349
						snap1156 := d399
						snap1157 := d400
						snap1158 := d401
						snap1159 := d454
						snap1160 := d455
						snap1161 := d456
						snap1162 := d457
						snap1163 := d514
						snap1164 := d515
						snap1165 := d516
						snap1166 := d517
						snap1167 := d578
						snap1168 := d579
						snap1169 := d642
						snap1170 := d643
						snap1171 := d644
						snap1172 := d645
						snap1173 := d712
						snap1174 := d713
						snap1175 := d782
						snap1176 := d783
						snap1177 := d784
						snap1178 := d856
						snap1179 := d857
						snap1180 := d931
						snap1181 := d932
						snap1182 := d933
						snap1183 := d934
						snap1184 := d935
						snap1185 := d1014
						snap1186 := d1016
						snap1187 := d1017
						snap1188 := d1018
						snap1189 := d1019
						snap1190 := d1103
						snap1191 := d1104
						snap1192 := d1105
						snap1193 := d1106
						snap1194 := d1107
						alloc1195 := ctx.SnapshotAllocState()
						bbs[31].Render()
						ctx.RestoreAllocState(alloc1195)
						d1 = snap1108
						d2 = snap1109
						d3 = snap1110
						d4 = snap1111
						d5 = snap1112
						d6 = snap1113
						d7 = snap1114
						d8 = snap1115
						d9 = snap1116
						d10 = snap1117
						d11 = snap1118
						d12 = snap1119
						d26 = snap1120
						d29 = snap1121
						d30 = snap1122
						d31 = snap1123
						d49 = snap1124
						d50 = snap1125
						d51 = snap1126
						d52 = snap1127
						d74 = snap1128
						d75 = snap1129
						d76 = snap1130
						d77 = snap1131
						d78 = snap1132
						d79 = snap1133
						d107 = snap1134
						d136 = snap1135
						d137 = snap1136
						d138 = snap1137
						d139 = snap1138
						d172 = snap1139
						d206 = snap1140
						d207 = snap1141
						d208 = snap1142
						d211 = snap1143
						d212 = snap1144
						d213 = snap1145
						d253 = snap1146
						d254 = snap1147
						d255 = snap1148
						d298 = snap1149
						d299 = snap1150
						d300 = snap1151
						d346 = snap1152
						d347 = snap1153
						d348 = snap1154
						d349 = snap1155
						d399 = snap1156
						d400 = snap1157
						d401 = snap1158
						d454 = snap1159
						d455 = snap1160
						d456 = snap1161
						d457 = snap1162
						d514 = snap1163
						d515 = snap1164
						d516 = snap1165
						d517 = snap1166
						d578 = snap1167
						d579 = snap1168
						d642 = snap1169
						d643 = snap1170
						d644 = snap1171
						d645 = snap1172
						d712 = snap1173
						d713 = snap1174
						d782 = snap1175
						d783 = snap1176
						d784 = snap1177
						d856 = snap1178
						d857 = snap1179
						d931 = snap1180
						d932 = snap1181
						d933 = snap1182
						d934 = snap1183
						d935 = snap1184
						d1014 = snap1185
						d1016 = snap1186
						d1017 = snap1187
						d1018 = snap1188
						d1019 = snap1189
						d1103 = snap1190
						d1104 = snap1191
						d1105 = snap1192
						d1106 = snap1193
						d1107 = snap1194
					}
					if !bbs[30].Rendered {
						return bbs[30].Render()
					}
					return result
					return result
				}
				bbs[33].Render = func() JITValueDesc {
					if bbs[33].Rendered {
						ctx.EmitJmp(lbl34)
						return result
					}
					bbs[33].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_33 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl34)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d1196 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.Float() >= -9.223372036854776e+18)}
					} else {
						r31 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(14114281232179134464))
						ctx.EmitCmpFloat64(d7.Reg, ctx.ScratchReg)
						d1196 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r31, Condition: CondUnsignedAboveOrEqual}
						ctx.BindReg(r31, &d1196)
					}
					d1197 = d1196
					ctx.EnsureDesc(&d1197)
					if d1197.Loc != LocImm && d1197.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d1197.Loc == LocImm {
						if d1197.Imm.Bool() {
							return bbs[32].Render()
						}
						return bbs[31].Render()
					}
					ctx.EmitJump(d1197.Condition, lbl33)
					if bbs[31].Rendered {
						ctx.EmitJmp(lbl32)
					}
					ctx.FreeDesc(&d1196)
					ctx.FlushRegisterMoves()
					if !bbs[31].Rendered {
						snap1198 := d1
						snap1199 := d2
						snap1200 := d3
						snap1201 := d4
						snap1202 := d5
						snap1203 := d6
						snap1204 := d7
						snap1205 := d8
						snap1206 := d9
						snap1207 := d10
						snap1208 := d11
						snap1209 := d12
						snap1210 := d26
						snap1211 := d29
						snap1212 := d30
						snap1213 := d31
						snap1214 := d49
						snap1215 := d50
						snap1216 := d51
						snap1217 := d52
						snap1218 := d74
						snap1219 := d75
						snap1220 := d76
						snap1221 := d77
						snap1222 := d78
						snap1223 := d79
						snap1224 := d107
						snap1225 := d136
						snap1226 := d137
						snap1227 := d138
						snap1228 := d139
						snap1229 := d172
						snap1230 := d206
						snap1231 := d207
						snap1232 := d208
						snap1233 := d211
						snap1234 := d212
						snap1235 := d213
						snap1236 := d253
						snap1237 := d254
						snap1238 := d255
						snap1239 := d298
						snap1240 := d299
						snap1241 := d300
						snap1242 := d346
						snap1243 := d347
						snap1244 := d348
						snap1245 := d349
						snap1246 := d399
						snap1247 := d400
						snap1248 := d401
						snap1249 := d454
						snap1250 := d455
						snap1251 := d456
						snap1252 := d457
						snap1253 := d514
						snap1254 := d515
						snap1255 := d516
						snap1256 := d517
						snap1257 := d578
						snap1258 := d579
						snap1259 := d642
						snap1260 := d643
						snap1261 := d644
						snap1262 := d645
						snap1263 := d712
						snap1264 := d713
						snap1265 := d782
						snap1266 := d783
						snap1267 := d784
						snap1268 := d856
						snap1269 := d857
						snap1270 := d931
						snap1271 := d932
						snap1272 := d933
						snap1273 := d934
						snap1274 := d935
						snap1275 := d1014
						snap1276 := d1016
						snap1277 := d1017
						snap1278 := d1018
						snap1279 := d1019
						snap1280 := d1103
						snap1281 := d1104
						snap1282 := d1105
						snap1283 := d1106
						snap1284 := d1107
						snap1285 := d1196
						snap1286 := d1197
						alloc1287 := ctx.SnapshotAllocState()
						bbs[31].Render()
						ctx.RestoreAllocState(alloc1287)
						d1 = snap1198
						d2 = snap1199
						d3 = snap1200
						d4 = snap1201
						d5 = snap1202
						d6 = snap1203
						d7 = snap1204
						d8 = snap1205
						d9 = snap1206
						d10 = snap1207
						d11 = snap1208
						d12 = snap1209
						d26 = snap1210
						d29 = snap1211
						d30 = snap1212
						d31 = snap1213
						d49 = snap1214
						d50 = snap1215
						d51 = snap1216
						d52 = snap1217
						d74 = snap1218
						d75 = snap1219
						d76 = snap1220
						d77 = snap1221
						d78 = snap1222
						d79 = snap1223
						d107 = snap1224
						d136 = snap1225
						d137 = snap1226
						d138 = snap1227
						d139 = snap1228
						d172 = snap1229
						d206 = snap1230
						d207 = snap1231
						d208 = snap1232
						d211 = snap1233
						d212 = snap1234
						d213 = snap1235
						d253 = snap1236
						d254 = snap1237
						d255 = snap1238
						d298 = snap1239
						d299 = snap1240
						d300 = snap1241
						d346 = snap1242
						d347 = snap1243
						d348 = snap1244
						d349 = snap1245
						d399 = snap1246
						d400 = snap1247
						d401 = snap1248
						d454 = snap1249
						d455 = snap1250
						d456 = snap1251
						d457 = snap1252
						d514 = snap1253
						d515 = snap1254
						d516 = snap1255
						d517 = snap1256
						d578 = snap1257
						d579 = snap1258
						d642 = snap1259
						d643 = snap1260
						d644 = snap1261
						d645 = snap1262
						d712 = snap1263
						d713 = snap1264
						d782 = snap1265
						d783 = snap1266
						d784 = snap1267
						d856 = snap1268
						d857 = snap1269
						d931 = snap1270
						d932 = snap1271
						d933 = snap1272
						d934 = snap1273
						d935 = snap1274
						d1014 = snap1275
						d1016 = snap1276
						d1017 = snap1277
						d1018 = snap1278
						d1019 = snap1279
						d1103 = snap1280
						d1104 = snap1281
						d1105 = snap1282
						d1106 = snap1283
						d1107 = snap1284
						d1196 = snap1285
						d1197 = snap1286
					}
					if !bbs[32].Rendered {
						return bbs[32].Render()
					}
					return result
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 90,
		},
		Optimize: optimizeAssociative,
	})
	Declare(&Globalenv, &Declaration{
		Name: "/",

		Fn: func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			v := a[0].Float()
			for _, i := range a[1:] {
				d := i.Float()
				if d == 0 {
					panic("division by zero")
				}
				v /= d
			}
			return NewFloat(v)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "divides two or more numbers from the first one; division by zero raises an error",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value...", Description: "values", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["/"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d16 JITValueDesc
				_ = d16
				var dynamicArgOff17 int32
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d35 JITValueDesc
				_ = d35
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d80 JITValueDesc
				_ = d80
				var dynamicArgOff81 int32
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [10]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[5].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d5)
					}
					if d5.Loc == LocReg && d1.Loc == LocReg && d5.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDescsTogether(&d5, &d4)
					if d5.Loc == LocImm && d4.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() < d4.Imm.Int())}
					} else if d4.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d5.Reg)
						if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d5.Reg, int32(d4.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitCmpInt64(d5.Reg, ctx.ScratchReg)
						}
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d6)
					} else if d5.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d5.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d4.Reg)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d6)
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpInt64(d5.Reg, d4.Reg)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d6)
					}
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d7.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d6)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap8 := d1
						snap9 := d2
						snap10 := d3
						snap11 := d4
						snap12 := d5
						snap13 := d6
						snap14 := d7
						alloc15 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc15)
						d1 = snap8
						d2 = snap9
						d3 = snap10
						d4 = snap11
						d5 = snap12
						d6 = snap13
						d7 = snap14
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						idx := int(d5.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d16 = args[idx]
						d16.ID = 0
					} else {
						ctx.EnsureDesc(&d5)
						dynamicArgOff17 = ctx.AllocStack(16)
						ctx.ProtectReg(d5.Reg)
						lbl11 := ctx.ReserveLabel()
						lbl12 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d5.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl12)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d5.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff17))
							ctx.EmitJmp(lbl11)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff17))
						ctx.MarkLabel(lbl11)
						ctx.UnprotectReg(d5.Reg)
						d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff17), Rooted: true}
					}
					d19 = d16
					d19.ID = 0
					d18 = ctx.EmitTagEqualsBorrowed(&d19, tagNil, JITValueDesc{Loc: LocAny})
					d20 = d18
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[4].Render()
						}
						ctx.SyncDesc(&d5)
						if d5.Loc == LocReg || d5.Loc == LocFPReg {
							ctx.ProtectReg(d5.Reg)
						} else if d5.Loc == LocRegPair {
							ctx.ProtectReg(d5.Reg)
							ctx.ProtectReg(d5.Reg2)
						}
						d21 = d5
						if d21.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
						if d5.Loc == LocReg || d5.Loc == LocFPReg {
							ctx.UnprotectReg(d5.Reg)
						} else if d5.Loc == LocRegPair {
							ctx.UnprotectReg(d5.Reg)
							ctx.UnprotectReg(d5.Reg2)
						}
						return bbs[1].Render()
					}
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					ctx.EmitJmp(lbl13)
					snap22 := d1
					snap23 := d2
					snap24 := d3
					snap25 := d4
					snap26 := d5
					snap27 := d6
					snap28 := d7
					snap29 := d16
					snap30 := d18
					snap31 := d19
					snap32 := d20
					snap33 := d21
					alloc34 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl13)
					ctx.SyncDesc(&d5)
					if d5.Loc == LocReg || d5.Loc == LocFPReg {
						ctx.ProtectReg(d5.Reg)
					} else if d5.Loc == LocRegPair {
						ctx.ProtectReg(d5.Reg)
						ctx.ProtectReg(d5.Reg2)
					}
					d35 = d5
					if d35.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d35)
					ctx.EmitStoreToStack(d35, int32(bbs[1].PhiBase)+int32(0))
					if d5.Loc == LocReg || d5.Loc == LocFPReg {
						ctx.UnprotectReg(d5.Reg)
					} else if d5.Loc == LocRegPair {
						ctx.UnprotectReg(d5.Reg)
						ctx.UnprotectReg(d5.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc34)
					d1 = snap22
					d2 = snap23
					d3 = snap24
					d4 = snap25
					d5 = snap26
					d6 = snap27
					d7 = snap28
					d16 = snap29
					d18 = snap30
					d19 = snap31
					d20 = snap32
					d21 = snap33
					if !bbs[1].Rendered {
						snap36 := d1
						snap37 := d2
						snap38 := d3
						snap39 := d4
						snap40 := d5
						snap41 := d6
						snap42 := d7
						snap43 := d16
						snap44 := d18
						snap45 := d19
						snap46 := d20
						snap47 := d21
						snap48 := d35
						alloc49 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc49)
						d1 = snap36
						d2 = snap37
						d3 = snap38
						d4 = snap39
						d5 = snap40
						d6 = snap41
						d7 = snap42
						d16 = snap43
						d18 = snap44
						d19 = snap45
						d20 = snap46
						d21 = snap47
						d35 = snap48
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d18)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d50 = args[0]
					d50.ID = 0
					d51 = ctx.EmitFloatDesc(d50)
					ctx.StabilizeDescForControlFlow(&d51)
					ctx.FreeDesc(&d50)
					d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args) - 1))}
					ctx.StabilizeDescForControlFlow(&d52)
					ctx.SyncDesc(&d51)
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.ProtectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.ProtectReg(d51.Reg)
						ctx.ProtectReg(d51.Reg2)
					}
					d53 = d51
					if d53.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreToStack(d53, int32(bbs[5].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[5].PhiBase)+int32(16))
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.UnprotectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.UnprotectReg(d51.Reg)
						ctx.UnprotectReg(d51.Reg2)
					}
					return bbs[5].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d54 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d54)
					if d54.Loc == LocRegPair || d54.Loc == LocStackPair || d54.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d54, &result)
						result.Type = d54.Type
					} else {
						switch d54.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d54)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d54)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d54)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d54, &result)
							result.Type = d54.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d55)
					}
					if d55.Loc == LocReg && d3.Loc == LocReg && d55.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d55)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d52)
					ctx.EnsureDescsTogether(&d55, &d52)
					if d55.Loc == LocImm && d52.Loc == LocImm {
						d56 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d55.Imm.Int() < d52.Imm.Int())}
					} else if d52.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d55.Reg)
						if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d55.Reg, int32(d52.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d52.Imm.Int()))
							ctx.EmitCmpInt64(d55.Reg, ctx.ScratchReg)
						}
						d56 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d56)
					} else if d55.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d55.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d52.Reg)
						d56 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d56)
					} else {
						r5 := ctx.AllocRegExcept(d55.Reg)
						ctx.EmitCmpInt64(d55.Reg, d52.Reg)
						d56 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d56)
					}
					d57 = d56
					ctx.EnsureDesc(&d57)
					if d57.Loc != LocImm && d57.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d57.Loc == LocImm {
						if d57.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitJump(d57.Condition, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d56)
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap58 := d1
						snap59 := d2
						snap60 := d3
						snap61 := d4
						snap62 := d5
						snap63 := d6
						snap64 := d7
						snap65 := d16
						snap66 := d18
						snap67 := d19
						snap68 := d20
						snap69 := d21
						snap70 := d35
						snap71 := d50
						snap72 := d51
						snap73 := d52
						snap74 := d53
						snap75 := d54
						snap76 := d55
						snap77 := d56
						snap78 := d57
						alloc79 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc79)
						d1 = snap58
						d2 = snap59
						d3 = snap60
						d4 = snap61
						d5 = snap62
						d6 = snap63
						d7 = snap64
						d16 = snap65
						d18 = snap66
						d19 = snap67
						d20 = snap68
						d21 = snap69
						d35 = snap70
						d50 = snap71
						d51 = snap72
						d52 = snap73
						d53 = snap74
						d54 = snap75
						d55 = snap76
						d56 = snap77
						d57 = snap78
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						idx := int(d55.Imm.Int()) + 1
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d80 = args[idx]
						d80.ID = 0
					} else {
						ctx.EnsureDesc(&d55)
						dynamicArgOff81 = ctx.AllocStack(16)
						ctx.ProtectReg(d55.Reg)
						lbl14 := ctx.ReserveLabel()
						lbl15 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d55.Reg, int32(len(args)-1))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl15)
						for i := 1; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d55.Reg, int32(i-1))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff81))
							ctx.EmitJmp(lbl14)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl15)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff81))
						ctx.MarkLabel(lbl14)
						ctx.UnprotectReg(d55.Reg)
						d80 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff81), Rooted: true}
					}
					d82 = ctx.EmitFloatDesc(d80)
					ctx.StabilizeDescForControlFlow(&d82)
					ctx.EnsureDesc(&d82)
					if d82.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d82.Imm.Float() == 0)}
					} else {
						r6 := ctx.AllocRegExcept(d82.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(0))
						ctx.EmitCmpFloat64Setcc(r6, d82.Reg, ctx.ScratchReg, CondEqual)
						d83 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
						ctx.BindReg(r6, &d83)
					}
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitCmpRegImm32(d84.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap85 := d1
						snap86 := d2
						snap87 := d3
						snap88 := d4
						snap89 := d5
						snap90 := d6
						snap91 := d7
						snap92 := d16
						snap93 := d18
						snap94 := d19
						snap95 := d20
						snap96 := d21
						snap97 := d35
						snap98 := d50
						snap99 := d51
						snap100 := d52
						snap101 := d53
						snap102 := d54
						snap103 := d55
						snap104 := d56
						snap105 := d57
						snap106 := d80
						snap107 := d82
						snap108 := d83
						snap109 := d84
						alloc110 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc110)
						d1 = snap85
						d2 = snap86
						d3 = snap87
						d4 = snap88
						d5 = snap89
						d6 = snap90
						d7 = snap91
						d16 = snap92
						d18 = snap93
						d19 = snap94
						d20 = snap95
						d21 = snap96
						d35 = snap97
						d50 = snap98
						d51 = snap99
						d52 = snap100
						d53 = snap101
						d54 = snap102
						d55 = snap103
						d56 = snap104
						d57 = snap105
						d80 = snap106
						d82 = snap107
						d83 = snap108
						d84 = snap109
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d83)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						ctx.EmitMakeFloat(result, d2)
					} else {
						ctx.EmitMovToReg(result.Reg2, d2)
						d111 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d111)
						if d2.Loc == LocReg && d2.Reg != result.Reg2 {
							ctx.FreeReg(d2.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["/"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d82)
					ctx.EnsureDescsTogether(&d2, &d82)
					if d2.Loc == LocImm && d82.Loc == LocImm {
						d112 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d2.Imm.Float() / d82.Imm.Float())}
					} else if d2.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d82.Reg)
						_, xBits := d2.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d82.Reg)
						d112 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d112)
					} else if d82.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						_, yBits := d82.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitDivFloat64(scratch, ctx.ScratchReg)
						d112 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d112)
					} else {
						r7 := ctx.AllocRegExcept(d2.Reg, d82.Reg)
						ctx.EmitMovRegReg(r7, d2.Reg)
						ctx.EmitDivFloat64(r7, d82.Reg)
						d112 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d112)
					}
					if d112.Loc == LocReg && d2.Loc == LocReg && d112.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d112)
					ctx.EmitStoreToStack(d112, int32(bbs[5].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d112)
					ctx.SyncDesc(&d55)
					if d55.Loc == LocReg || d55.Loc == LocFPReg {
						ctx.ProtectReg(d55.Reg)
					} else if d55.Loc == LocRegPair {
						ctx.ProtectReg(d55.Reg)
						ctx.ProtectReg(d55.Reg2)
					}
					d113 = d55
					if d113.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d113)
					ctx.EmitStoreToStack(d113, int32(bbs[5].PhiBase)+int32(16))
					if d55.Loc == LocReg || d55.Loc == LocFPReg {
						ctx.UnprotectReg(d55.Reg)
					} else if d55.Loc == LocRegPair {
						ctx.UnprotectReg(d55.Reg)
						ctx.UnprotectReg(d55.Reg2)
					}
					return bbs[5].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  34,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "div_null",

		Fn: func(a ...Scmer) Scmer {
			// SQL division semantics: a nil operand or division by zero yields NULL
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			v := a[0].Float()
			for _, i := range a[1:] {
				d := i.Float()
				if d == 0 {
					return NewNil()
				}
				v /= d
			}
			return NewFloat(v)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "divides two or more numbers, yielding NULL on a nil operand or division by zero",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value...", Description: "values", Variadic: true},
			},
			Return:        &TypeDescriptor{Kind: "number"},
			Const:         true,
			JITInlineCost: 34,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["div_null"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d16 JITValueDesc
				_ = d16
				var dynamicArgOff17 int32
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d35 JITValueDesc
				_ = d35
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d80 JITValueDesc
				_ = d80
				var dynamicArgOff81 int32
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [10]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[5].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d5)
					}
					if d5.Loc == LocReg && d1.Loc == LocReg && d5.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDescsTogether(&d5, &d4)
					if d5.Loc == LocImm && d4.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() < d4.Imm.Int())}
					} else if d4.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d5.Reg)
						if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d5.Reg, int32(d4.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitCmpInt64(d5.Reg, ctx.ScratchReg)
						}
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d6)
					} else if d5.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d5.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d4.Reg)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d6)
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpInt64(d5.Reg, d4.Reg)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d6)
					}
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d7.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d6)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap8 := d1
						snap9 := d2
						snap10 := d3
						snap11 := d4
						snap12 := d5
						snap13 := d6
						snap14 := d7
						alloc15 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc15)
						d1 = snap8
						d2 = snap9
						d3 = snap10
						d4 = snap11
						d5 = snap12
						d6 = snap13
						d7 = snap14
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						idx := int(d5.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d16 = args[idx]
						d16.ID = 0
					} else {
						ctx.EnsureDesc(&d5)
						dynamicArgOff17 = ctx.AllocStack(16)
						ctx.ProtectReg(d5.Reg)
						lbl11 := ctx.ReserveLabel()
						lbl12 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d5.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl12)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d5.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff17))
							ctx.EmitJmp(lbl11)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff17))
						ctx.MarkLabel(lbl11)
						ctx.UnprotectReg(d5.Reg)
						d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff17), Rooted: true}
					}
					d19 = d16
					d19.ID = 0
					d18 = ctx.EmitTagEqualsBorrowed(&d19, tagNil, JITValueDesc{Loc: LocAny})
					d20 = d18
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[4].Render()
						}
						ctx.SyncDesc(&d5)
						if d5.Loc == LocReg || d5.Loc == LocFPReg {
							ctx.ProtectReg(d5.Reg)
						} else if d5.Loc == LocRegPair {
							ctx.ProtectReg(d5.Reg)
							ctx.ProtectReg(d5.Reg2)
						}
						d21 = d5
						if d21.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
						if d5.Loc == LocReg || d5.Loc == LocFPReg {
							ctx.UnprotectReg(d5.Reg)
						} else if d5.Loc == LocRegPair {
							ctx.UnprotectReg(d5.Reg)
							ctx.UnprotectReg(d5.Reg2)
						}
						return bbs[1].Render()
					}
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					ctx.EmitJmp(lbl13)
					snap22 := d1
					snap23 := d2
					snap24 := d3
					snap25 := d4
					snap26 := d5
					snap27 := d6
					snap28 := d7
					snap29 := d16
					snap30 := d18
					snap31 := d19
					snap32 := d20
					snap33 := d21
					alloc34 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl13)
					ctx.SyncDesc(&d5)
					if d5.Loc == LocReg || d5.Loc == LocFPReg {
						ctx.ProtectReg(d5.Reg)
					} else if d5.Loc == LocRegPair {
						ctx.ProtectReg(d5.Reg)
						ctx.ProtectReg(d5.Reg2)
					}
					d35 = d5
					if d35.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d35)
					ctx.EmitStoreToStack(d35, int32(bbs[1].PhiBase)+int32(0))
					if d5.Loc == LocReg || d5.Loc == LocFPReg {
						ctx.UnprotectReg(d5.Reg)
					} else if d5.Loc == LocRegPair {
						ctx.UnprotectReg(d5.Reg)
						ctx.UnprotectReg(d5.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc34)
					d1 = snap22
					d2 = snap23
					d3 = snap24
					d4 = snap25
					d5 = snap26
					d6 = snap27
					d7 = snap28
					d16 = snap29
					d18 = snap30
					d19 = snap31
					d20 = snap32
					d21 = snap33
					if !bbs[1].Rendered {
						snap36 := d1
						snap37 := d2
						snap38 := d3
						snap39 := d4
						snap40 := d5
						snap41 := d6
						snap42 := d7
						snap43 := d16
						snap44 := d18
						snap45 := d19
						snap46 := d20
						snap47 := d21
						snap48 := d35
						alloc49 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc49)
						d1 = snap36
						d2 = snap37
						d3 = snap38
						d4 = snap39
						d5 = snap40
						d6 = snap41
						d7 = snap42
						d16 = snap43
						d18 = snap44
						d19 = snap45
						d20 = snap46
						d21 = snap47
						d35 = snap48
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d18)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d50 = args[0]
					d50.ID = 0
					d51 = ctx.EmitFloatDesc(d50)
					ctx.StabilizeDescForControlFlow(&d51)
					ctx.FreeDesc(&d50)
					d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args) - 1))}
					ctx.StabilizeDescForControlFlow(&d52)
					ctx.SyncDesc(&d51)
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.ProtectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.ProtectReg(d51.Reg)
						ctx.ProtectReg(d51.Reg2)
					}
					d53 = d51
					if d53.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreToStack(d53, int32(bbs[5].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[5].PhiBase)+int32(16))
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.UnprotectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.UnprotectReg(d51.Reg)
						ctx.UnprotectReg(d51.Reg2)
					}
					return bbs[5].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d54 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d54)
					if d54.Loc == LocRegPair || d54.Loc == LocStackPair || d54.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d54, &result)
						result.Type = d54.Type
					} else {
						switch d54.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d54)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d54)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d54)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d54, &result)
							result.Type = d54.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d55)
					}
					if d55.Loc == LocReg && d3.Loc == LocReg && d55.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d55)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d52)
					ctx.EnsureDescsTogether(&d55, &d52)
					if d55.Loc == LocImm && d52.Loc == LocImm {
						d56 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d55.Imm.Int() < d52.Imm.Int())}
					} else if d52.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d55.Reg)
						if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d55.Reg, int32(d52.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d52.Imm.Int()))
							ctx.EmitCmpInt64(d55.Reg, ctx.ScratchReg)
						}
						d56 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d56)
					} else if d55.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d55.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d52.Reg)
						d56 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d56)
					} else {
						r5 := ctx.AllocRegExcept(d55.Reg)
						ctx.EmitCmpInt64(d55.Reg, d52.Reg)
						d56 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d56)
					}
					d57 = d56
					ctx.EnsureDesc(&d57)
					if d57.Loc != LocImm && d57.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d57.Loc == LocImm {
						if d57.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitJump(d57.Condition, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d56)
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap58 := d1
						snap59 := d2
						snap60 := d3
						snap61 := d4
						snap62 := d5
						snap63 := d6
						snap64 := d7
						snap65 := d16
						snap66 := d18
						snap67 := d19
						snap68 := d20
						snap69 := d21
						snap70 := d35
						snap71 := d50
						snap72 := d51
						snap73 := d52
						snap74 := d53
						snap75 := d54
						snap76 := d55
						snap77 := d56
						snap78 := d57
						alloc79 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc79)
						d1 = snap58
						d2 = snap59
						d3 = snap60
						d4 = snap61
						d5 = snap62
						d6 = snap63
						d7 = snap64
						d16 = snap65
						d18 = snap66
						d19 = snap67
						d20 = snap68
						d21 = snap69
						d35 = snap70
						d50 = snap71
						d51 = snap72
						d52 = snap73
						d53 = snap74
						d54 = snap75
						d55 = snap76
						d56 = snap77
						d57 = snap78
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						idx := int(d55.Imm.Int()) + 1
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d80 = args[idx]
						d80.ID = 0
					} else {
						ctx.EnsureDesc(&d55)
						dynamicArgOff81 = ctx.AllocStack(16)
						ctx.ProtectReg(d55.Reg)
						lbl14 := ctx.ReserveLabel()
						lbl15 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d55.Reg, int32(len(args)-1))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl15)
						for i := 1; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d55.Reg, int32(i-1))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff81))
							ctx.EmitJmp(lbl14)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl15)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff81))
						ctx.MarkLabel(lbl14)
						ctx.UnprotectReg(d55.Reg)
						d80 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff81), Rooted: true}
					}
					d82 = ctx.EmitFloatDesc(d80)
					ctx.StabilizeDescForControlFlow(&d82)
					ctx.EnsureDesc(&d82)
					if d82.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d82.Imm.Float() == 0)}
					} else {
						r6 := ctx.AllocRegExcept(d82.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(0))
						ctx.EmitCmpFloat64Setcc(r6, d82.Reg, ctx.ScratchReg, CondEqual)
						d83 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
						ctx.BindReg(r6, &d83)
					}
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitCmpRegImm32(d84.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap85 := d1
						snap86 := d2
						snap87 := d3
						snap88 := d4
						snap89 := d5
						snap90 := d6
						snap91 := d7
						snap92 := d16
						snap93 := d18
						snap94 := d19
						snap95 := d20
						snap96 := d21
						snap97 := d35
						snap98 := d50
						snap99 := d51
						snap100 := d52
						snap101 := d53
						snap102 := d54
						snap103 := d55
						snap104 := d56
						snap105 := d57
						snap106 := d80
						snap107 := d82
						snap108 := d83
						snap109 := d84
						alloc110 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc110)
						d1 = snap85
						d2 = snap86
						d3 = snap87
						d4 = snap88
						d5 = snap89
						d6 = snap90
						d7 = snap91
						d16 = snap92
						d18 = snap93
						d19 = snap94
						d20 = snap95
						d21 = snap96
						d35 = snap97
						d50 = snap98
						d51 = snap99
						d52 = snap100
						d53 = snap101
						d54 = snap102
						d55 = snap103
						d56 = snap104
						d57 = snap105
						d80 = snap106
						d82 = snap107
						d83 = snap108
						d84 = snap109
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d83)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						ctx.EmitMakeFloat(result, d2)
					} else {
						ctx.EmitMovToReg(result.Reg2, d2)
						d111 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d111)
						if d2.Loc == LocReg && d2.Reg != result.Reg2 {
							ctx.FreeReg(d2.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d112 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d112)
					if d112.Loc == LocRegPair || d112.Loc == LocStackPair || d112.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d112, &result)
						result.Type = d112.Type
					} else {
						switch d112.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d112)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d112)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d112)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d112, &result)
							result.Type = d112.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d82)
					ctx.EnsureDescsTogether(&d2, &d82)
					if d2.Loc == LocImm && d82.Loc == LocImm {
						d113 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d2.Imm.Float() / d82.Imm.Float())}
					} else if d2.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d82.Reg)
						_, xBits := d2.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d82.Reg)
						d113 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d113)
					} else if d82.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						_, yBits := d82.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitDivFloat64(scratch, ctx.ScratchReg)
						d113 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d113)
					} else {
						r7 := ctx.AllocRegExcept(d2.Reg, d82.Reg)
						ctx.EmitMovRegReg(r7, d2.Reg)
						ctx.EmitDivFloat64(r7, d82.Reg)
						d113 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d113)
					}
					if d113.Loc == LocReg && d2.Loc == LocReg && d113.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d113)
					ctx.EmitStoreToStack(d113, int32(bbs[5].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d113)
					ctx.SyncDesc(&d55)
					if d55.Loc == LocReg || d55.Loc == LocFPReg {
						ctx.ProtectReg(d55.Reg)
					} else if d55.Loc == LocRegPair {
						ctx.ProtectReg(d55.Reg)
						ctx.ProtectReg(d55.Reg2)
					}
					d114 = d55
					if d114.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d114)
					ctx.EmitStoreToStack(d114, int32(bbs[5].PhiBase)+int32(16))
					if d55.Loc == LocReg || d55.Loc == LocFPReg {
						ctx.UnprotectReg(d55.Reg)
					} else if d55.Loc == LocRegPair {
						ctx.UnprotectReg(d55.Reg)
						ctx.UnprotectReg(d55.Reg2)
					}
					return bbs[5].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "intdiv",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			if a[0].IsInt() && a[1].IsInt() {
				divisor := a[1].Int()
				if divisor == 0 || a[0].Int() == math.MinInt64 && divisor == -1 {
					return NewNil()
				}
				return NewInt(a[0].Int() / divisor)
			}
			divisor := a[1].Float()
			if divisor == 0 {
				return NewNil()
			}
			quotient := math.Trunc(a[0].Float() / divisor)
			if math.IsNaN(quotient) || quotient < math.MinInt64 || quotient >= -float64(math.MinInt64) {
				return NewNil()
			}
			return NewInt(int64(quotient))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "divides two numbers and truncates the quotient toward zero",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "dividend", Description: "value to divide"},
				{Kind: "number", Label: "divisor", Description: "value to divide by"},
			},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["intdiv"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d204 JITValueDesc
				_ = d204
				var d205 JITValueDesc
				_ = d205
				var d206 JITValueDesc
				_ = d206
				var d207 JITValueDesc
				_ = d207
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
				var d211 JITValueDesc
				_ = d211
				var d256 JITValueDesc
				_ = d256
				var d257 JITValueDesc
				_ = d257
				var d258 JITValueDesc
				_ = d258
				var d259 JITValueDesc
				_ = d259
				var d260 JITValueDesc
				_ = d260
				var d310 JITValueDesc
				_ = d310
				var d311 JITValueDesc
				_ = d311
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [17]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_0_14 := int32(-1)
				_ = bbpos_0_14
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_0_15 := int32(-1)
				_ = bbpos_0_15
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_0_16 := int32(-1)
				_ = bbpos_0_16
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d12 = d10
					d12.ID = 0
					d11 = ctx.EmitTagEqualsBorrowed(&d12, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d10)
					d13 = d11
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d13.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap14 := d0
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d9
						snap19 := d10
						snap20 := d11
						snap21 := d12
						snap22 := d13
						alloc23 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc23)
						d0 = snap14
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d9 = snap18
						d10 = snap19
						d11 = snap20
						d12 = snap21
						d13 = snap22
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d24 = args[1]
					d24.ID = 0
					d26 = d24
					d26.ID = 0
					d25 = ctx.EmitTagEqualsBorrowed(&d26, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d24)
					d27 = d25
					ctx.EnsureDesc(&d27)
					if d27.Loc != LocImm && d27.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d27.Loc == LocImm {
						if d27.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d27.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap28 := d0
						snap29 := d1
						snap30 := d2
						snap31 := d3
						snap32 := d9
						snap33 := d10
						snap34 := d11
						snap35 := d12
						snap36 := d13
						snap37 := d24
						snap38 := d25
						snap39 := d26
						snap40 := d27
						alloc41 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc41)
						d0 = snap28
						d1 = snap29
						d2 = snap30
						d3 = snap31
						d9 = snap32
						d10 = snap33
						d11 = snap34
						d12 = snap35
						d13 = snap36
						d24 = snap37
						d25 = snap38
						d26 = snap39
						d27 = snap40
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d25)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d42 = args[1]
					d42.ID = 0
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int())}
					} else if d42.Type == tagInt && d42.Loc == LocRegPair {
						ctx.FreeReg(d42.Reg)
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg2}
						ctx.BindReg(d42.Reg2, &d43)
						ctx.BindReg(d42.Reg2, &d43)
					} else if d42.Type == tagInt && d42.Loc == LocReg {
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
						ctx.BindReg(d42.Reg, &d43)
						ctx.BindReg(d42.Reg, &d43)
					} else {
						d43 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d42}, 1)
						d43.Type = tagInt
						ctx.BindReg(d43.Reg, &d43)
					}
					ctx.StabilizeDescForControlFlow(&d43)
					ctx.FreeDesc(&d42)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d43.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocRegExcept(d43.Reg)
						ctx.EmitCmpRegImm32(d43.Reg, 0)
						d44 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d44)
					}
					d45 = d44
					ctx.EnsureDesc(&d45)
					if d45.Loc != LocImm && d45.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d45.Loc == LocImm {
						if d45.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitJump(d45.Condition, lbl8)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d44)
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap46 := d0
						snap47 := d1
						snap48 := d2
						snap49 := d3
						snap50 := d9
						snap51 := d10
						snap52 := d11
						snap53 := d12
						snap54 := d13
						snap55 := d24
						snap56 := d25
						snap57 := d26
						snap58 := d27
						snap59 := d42
						snap60 := d43
						snap61 := d44
						snap62 := d45
						alloc63 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc63)
						d0 = snap46
						d1 = snap47
						d2 = snap48
						d3 = snap49
						d9 = snap50
						d10 = snap51
						d11 = snap52
						d12 = snap53
						d13 = snap54
						d24 = snap55
						d25 = snap56
						d26 = snap57
						d27 = snap58
						d42 = snap59
						d43 = snap60
						d44 = snap61
						d45 = snap62
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d64 = args[1]
					d64.ID = 0
					d65 = ctx.EmitFloatDesc(d64)
					ctx.StabilizeDescForControlFlow(&d65)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d65.Imm.Float() == 0)}
					} else {
						r1 := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(0))
						ctx.EmitCmpFloat64Setcc(r1, d65.Reg, ctx.ScratchReg, CondEqual)
						d66 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d66)
					}
					d67 = d66
					ctx.EnsureDesc(&d67)
					if d67.Loc != LocImm && d67.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d67.Loc == LocImm {
						if d67.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitCmpRegImm32(d67.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap68 := d0
						snap69 := d1
						snap70 := d2
						snap71 := d3
						snap72 := d9
						snap73 := d10
						snap74 := d11
						snap75 := d12
						snap76 := d13
						snap77 := d24
						snap78 := d25
						snap79 := d26
						snap80 := d27
						snap81 := d42
						snap82 := d43
						snap83 := d44
						snap84 := d45
						snap85 := d64
						snap86 := d65
						snap87 := d66
						snap88 := d67
						alloc89 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc89)
						d0 = snap68
						d1 = snap69
						d2 = snap70
						d3 = snap71
						d9 = snap72
						d10 = snap73
						d11 = snap74
						d12 = snap75
						d13 = snap76
						d24 = snap77
						d25 = snap78
						d26 = snap79
						d27 = snap80
						d42 = snap81
						d43 = snap82
						d44 = snap83
						d45 = snap84
						d64 = snap85
						d65 = snap86
						d66 = snap87
						d67 = snap88
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					ctx.FreeDesc(&d66)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d90 = args[1]
					d90.ID = 0
					d92 = d90
					d92.ID = 0
					d91 = ctx.EmitTagEqualsBorrowed(&d92, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d90)
					d93 = d91
					ctx.EnsureDesc(&d93)
					if d93.Loc != LocImm && d93.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d93.Loc == LocImm {
						if d93.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d93.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap94 := d0
						snap95 := d1
						snap96 := d2
						snap97 := d3
						snap98 := d9
						snap99 := d10
						snap100 := d11
						snap101 := d12
						snap102 := d13
						snap103 := d24
						snap104 := d25
						snap105 := d26
						snap106 := d27
						snap107 := d42
						snap108 := d43
						snap109 := d44
						snap110 := d45
						snap111 := d64
						snap112 := d65
						snap113 := d66
						snap114 := d67
						snap115 := d90
						snap116 := d91
						snap117 := d92
						snap118 := d93
						alloc119 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc119)
						d0 = snap94
						d1 = snap95
						d2 = snap96
						d3 = snap97
						d9 = snap98
						d10 = snap99
						d11 = snap100
						d12 = snap101
						d13 = snap102
						d24 = snap103
						d25 = snap104
						d26 = snap105
						d27 = snap106
						d42 = snap107
						d43 = snap108
						d44 = snap109
						d45 = snap110
						d64 = snap111
						d65 = snap112
						d66 = snap113
						d67 = snap114
						d90 = snap115
						d91 = snap116
						d92 = snap117
						d93 = snap118
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d91)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d120 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d120)
					if d120.Loc == LocRegPair || d120.Loc == LocStackPair || d120.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d120, &result)
						result.Type = d120.Type
					} else {
						switch d120.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d120)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d120)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d120)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d120, &result)
							result.Type = d120.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d121 = args[0]
					d121.ID = 0
					if d121.Loc == LocImm {
						d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d121.Imm.Int())}
					} else if d121.Type == tagInt && d121.Loc == LocRegPair {
						ctx.FreeReg(d121.Reg)
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d121.Reg2}
						ctx.BindReg(d121.Reg2, &d122)
						ctx.BindReg(d121.Reg2, &d122)
					} else if d121.Type == tagInt && d121.Loc == LocReg {
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d121.Reg}
						ctx.BindReg(d121.Reg, &d122)
						ctx.BindReg(d121.Reg, &d122)
					} else {
						d122 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d121}, 1)
						d122.Type = tagInt
						ctx.BindReg(d122.Reg, &d122)
					}
					ctx.FreeDesc(&d121)
					ctx.EnsureDesc(&d122)
					resultTarget123 := false
					_ = resultTarget123
					ctx.EnsureDesc(&d43)
					ctx.EnsureDescsTogether(&d122, &d43)
					if d122.Loc == LocImm && d43.Loc == LocImm {
						d124 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d122.Imm.Int() / d43.Imm.Int())}
					} else {
						d124 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntDiv), []JITValueDesc{d122, d43}, 1)
					}
					if d124.Loc == LocReg && d122.Loc == LocReg && d124.Reg == d122.Reg {
						ctx.TransferReg(d122.Reg)
						d122.Loc = LocNone
					}
					ctx.FreeDesc(&d122)
					ctx.EnsureDesc(&d124)
					if d124.Loc == LocImm {
						ctx.EmitMakeInt(result, d124)
					} else {
						ctx.EmitMovToReg(result.Reg2, d124)
						d125 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d125)
						if d124.Loc == LocReg && d124.Reg != result.Reg2 {
							ctx.FreeReg(d124.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d126 = args[0]
					d126.ID = 0
					if d126.Loc == LocImm {
						d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d126.Imm.Int())}
					} else if d126.Type == tagInt && d126.Loc == LocRegPair {
						ctx.FreeReg(d126.Reg)
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2}
						ctx.BindReg(d126.Reg2, &d127)
						ctx.BindReg(d126.Reg2, &d127)
					} else if d126.Type == tagInt && d126.Loc == LocReg {
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg}
						ctx.BindReg(d126.Reg, &d127)
						ctx.BindReg(d126.Reg, &d127)
					} else {
						d127 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d126}, 1)
						d127.Type = tagInt
						ctx.BindReg(d127.Reg, &d127)
					}
					ctx.FreeDesc(&d126)
					ctx.EnsureDesc(&d127)
					if d127.Loc == LocImm {
						d128 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d127.Imm.Int() == -9223372036854775808)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0x8000000000000000)
						ctx.EmitCmpInt64(d127.Reg, ctx.ScratchReg)
						d128 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d128)
					}
					ctx.FreeDesc(&d127)
					d129 = d128
					ctx.EnsureDesc(&d129)
					if d129.Loc != LocImm && d129.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d129.Loc == LocImm {
						if d129.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d129.Condition, lbl11)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d128)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap130 := d0
						snap131 := d1
						snap132 := d2
						snap133 := d3
						snap134 := d9
						snap135 := d10
						snap136 := d11
						snap137 := d12
						snap138 := d13
						snap139 := d24
						snap140 := d25
						snap141 := d26
						snap142 := d27
						snap143 := d42
						snap144 := d43
						snap145 := d44
						snap146 := d45
						snap147 := d64
						snap148 := d65
						snap149 := d66
						snap150 := d67
						snap151 := d90
						snap152 := d91
						snap153 := d92
						snap154 := d93
						snap155 := d120
						snap156 := d121
						snap157 := d122
						snap158 := d124
						snap159 := d125
						snap160 := d126
						snap161 := d127
						snap162 := d128
						snap163 := d129
						alloc164 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc164)
						d0 = snap130
						d1 = snap131
						d2 = snap132
						d3 = snap133
						d9 = snap134
						d10 = snap135
						d11 = snap136
						d12 = snap137
						d13 = snap138
						d24 = snap139
						d25 = snap140
						d26 = snap141
						d27 = snap142
						d42 = snap143
						d43 = snap144
						d44 = snap145
						d45 = snap146
						d64 = snap147
						d65 = snap148
						d66 = snap149
						d67 = snap150
						d90 = snap151
						d91 = snap152
						d92 = snap153
						d93 = snap154
						d120 = snap155
						d121 = snap156
						d122 = snap157
						d124 = snap158
						d125 = snap159
						d126 = snap160
						d127 = snap161
						d128 = snap162
						d129 = snap163
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						d165 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d43.Imm.Int() == -1)}
					} else {
						r3 := ctx.AllocRegExcept(d43.Reg)
						ctx.EmitCmpRegImm32(d43.Reg, -1)
						d165 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d165)
					}
					d166 = d165
					ctx.EnsureDesc(&d166)
					if d166.Loc != LocImm && d166.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d166.Loc == LocImm {
						if d166.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d166.Condition, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d165)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap167 := d0
						snap168 := d1
						snap169 := d2
						snap170 := d3
						snap171 := d9
						snap172 := d10
						snap173 := d11
						snap174 := d12
						snap175 := d13
						snap176 := d24
						snap177 := d25
						snap178 := d26
						snap179 := d27
						snap180 := d42
						snap181 := d43
						snap182 := d44
						snap183 := d45
						snap184 := d64
						snap185 := d65
						snap186 := d66
						snap187 := d67
						snap188 := d90
						snap189 := d91
						snap190 := d92
						snap191 := d93
						snap192 := d120
						snap193 := d121
						snap194 := d122
						snap195 := d124
						snap196 := d125
						snap197 := d126
						snap198 := d127
						snap199 := d128
						snap200 := d129
						snap201 := d165
						snap202 := d166
						alloc203 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc203)
						d0 = snap167
						d1 = snap168
						d2 = snap169
						d3 = snap170
						d9 = snap171
						d10 = snap172
						d11 = snap173
						d12 = snap174
						d13 = snap175
						d24 = snap176
						d25 = snap177
						d26 = snap178
						d27 = snap179
						d42 = snap180
						d43 = snap181
						d44 = snap182
						d45 = snap183
						d64 = snap184
						d65 = snap185
						d66 = snap186
						d67 = snap187
						d90 = snap188
						d91 = snap189
						d92 = snap190
						d93 = snap191
						d120 = snap192
						d121 = snap193
						d122 = snap194
						d124 = snap195
						d125 = snap196
						d126 = snap197
						d127 = snap198
						d128 = snap199
						d129 = snap200
						d165 = snap201
						d166 = snap202
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d204 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d204)
					if d204.Loc == LocRegPair || d204.Loc == LocStackPair || d204.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d204, &result)
						result.Type = d204.Type
					} else {
						switch d204.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d204)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d204)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d204)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d204, &result)
							result.Type = d204.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d205 = args[0]
					d205.ID = 0
					d206 = ctx.EmitFloatDesc(d205)
					ctx.FreeDesc(&d205)
					ctx.EnsureDesc(&d206)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDescsTogether(&d206, &d65)
					if d206.Loc == LocImm && d65.Loc == LocImm {
						d207 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d206.Imm.Float() / d65.Imm.Float())}
					} else if d206.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d65.Reg)
						_, xBits := d206.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d65.Reg)
						d207 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d207)
					} else if d65.Loc == LocImm {
						_, yBits := d65.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitDivFloat64(d206.Reg, ctx.ScratchReg)
						d207 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d206.Reg}
						ctx.BindReg(d206.Reg, &d207)
					} else {
						ctx.EmitDivFloat64(d206.Reg, d65.Reg)
						d207 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d206.Reg}
						ctx.BindReg(d206.Reg, &d207)
					}
					if d207.Loc == LocReg && d206.Loc == LocReg && d207.Reg == d206.Reg {
						ctx.TransferReg(d206.Reg)
						d206.Loc = LocNone
					}
					ctx.FreeDesc(&d206)
					ctx.EnsureDesc(&d207)
					if d207.Loc == LocImm {
						d208 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d207.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d207)
						var truncSrc Reg
						if d207.Loc == LocRegPair {
							ctx.FreeReg(d207.Reg)
							truncSrc = d207.Reg2
						} else {
							truncSrc = d207.Reg
						}
						truncInt := ctx.AllocRegExcept(truncSrc)
						ctx.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
						ctx.EmitInt64ToFloatBits(truncInt)
						d208 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
						ctx.BindReg(truncInt, &d208)
						ctx.BindReg(truncInt, &d208)
					}
					ctx.StabilizeDescForControlFlow(&d208)
					ctx.FreeDesc(&d207)
					ctx.EnsureDesc(&d208)
					if d208.Loc == LocImm {
						d209 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d208.Imm.Float() != d208.Imm.Float())}
					} else {
						ctx.EnsureDesc(&d208)
						nanSource210 := d208.Reg
						if d208.Loc == LocRegPair {
							nanSource210 = d208.Reg2
						}
						r4 := ctx.AllocRegExcept(nanSource210)
						ctx.EmitCmpFloat64(nanSource210, nanSource210)
						d209 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondParity}
						ctx.BindReg(r4, &d209)
					}
					d211 = d209
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocImm && d211.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d211.Loc == LocImm {
						if d211.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[16].Render()
					}
					ctx.EmitJump(d211.Condition, lbl14)
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
					}
					ctx.FreeDesc(&d209)
					ctx.FlushRegisterMoves()
					if !bbs[16].Rendered {
						snap212 := d0
						snap213 := d1
						snap214 := d2
						snap215 := d3
						snap216 := d9
						snap217 := d10
						snap218 := d11
						snap219 := d12
						snap220 := d13
						snap221 := d24
						snap222 := d25
						snap223 := d26
						snap224 := d27
						snap225 := d42
						snap226 := d43
						snap227 := d44
						snap228 := d45
						snap229 := d64
						snap230 := d65
						snap231 := d66
						snap232 := d67
						snap233 := d90
						snap234 := d91
						snap235 := d92
						snap236 := d93
						snap237 := d120
						snap238 := d121
						snap239 := d122
						snap240 := d124
						snap241 := d125
						snap242 := d126
						snap243 := d127
						snap244 := d128
						snap245 := d129
						snap246 := d165
						snap247 := d166
						snap248 := d204
						snap249 := d205
						snap250 := d206
						snap251 := d207
						snap252 := d208
						snap253 := d209
						snap254 := d211
						alloc255 := ctx.SnapshotAllocState()
						bbs[16].Render()
						ctx.RestoreAllocState(alloc255)
						d0 = snap212
						d1 = snap213
						d2 = snap214
						d3 = snap215
						d9 = snap216
						d10 = snap217
						d11 = snap218
						d12 = snap219
						d13 = snap220
						d24 = snap221
						d25 = snap222
						d26 = snap223
						d27 = snap224
						d42 = snap225
						d43 = snap226
						d44 = snap227
						d45 = snap228
						d64 = snap229
						d65 = snap230
						d66 = snap231
						d67 = snap232
						d90 = snap233
						d91 = snap234
						d92 = snap235
						d93 = snap236
						d120 = snap237
						d121 = snap238
						d122 = snap239
						d124 = snap240
						d125 = snap241
						d126 = snap242
						d127 = snap243
						d128 = snap244
						d129 = snap245
						d165 = snap246
						d166 = snap247
						d204 = snap248
						d205 = snap249
						d206 = snap250
						d207 = snap251
						d208 = snap252
						d209 = snap253
						d211 = snap254
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					return result
				}
				bbs[13].Render = func() JITValueDesc {
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
						return result
					}
					bbs[13].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d256 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d256)
					if d256.Loc == LocRegPair || d256.Loc == LocStackPair || d256.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d256, &result)
						result.Type = d256.Type
					} else {
						switch d256.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d256)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d256)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d256)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d256, &result)
							result.Type = d256.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[14].Render = func() JITValueDesc {
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
						return result
					}
					bbs[14].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d208)
					ctx.EnsureDesc(&d208)
					if d208.Loc == LocImm {
						d257 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d208.Imm.Float()))}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r5, d208.Reg)
						d257 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d257)
					}
					ctx.EnsureDesc(&d257)
					if d257.Loc == LocImm {
						ctx.EmitMakeInt(result, d257)
					} else {
						ctx.EmitMovToReg(result.Reg2, d257)
						d258 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d258)
						if d257.Loc == LocReg && d257.Reg != result.Reg2 {
							ctx.FreeReg(d257.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[15].Render = func() JITValueDesc {
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
						return result
					}
					bbs[15].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d208)
					if d208.Loc == LocImm {
						d259 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d208.Imm.Float() >= 9.223372036854776e+18)}
					} else {
						r6 := ctx.AllocRegExcept(d208.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4890909195324358656))
						ctx.EmitCmpFloat64(d208.Reg, ctx.ScratchReg)
						d259 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondUnsignedAboveOrEqual}
						ctx.BindReg(r6, &d259)
					}
					d260 = d259
					ctx.EnsureDesc(&d260)
					if d260.Loc != LocImm && d260.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d260.Loc == LocImm {
						if d260.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[14].Render()
					}
					ctx.EmitJump(d260.Condition, lbl14)
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
					}
					ctx.FreeDesc(&d259)
					ctx.FlushRegisterMoves()
					if !bbs[14].Rendered {
						snap261 := d0
						snap262 := d1
						snap263 := d2
						snap264 := d3
						snap265 := d9
						snap266 := d10
						snap267 := d11
						snap268 := d12
						snap269 := d13
						snap270 := d24
						snap271 := d25
						snap272 := d26
						snap273 := d27
						snap274 := d42
						snap275 := d43
						snap276 := d44
						snap277 := d45
						snap278 := d64
						snap279 := d65
						snap280 := d66
						snap281 := d67
						snap282 := d90
						snap283 := d91
						snap284 := d92
						snap285 := d93
						snap286 := d120
						snap287 := d121
						snap288 := d122
						snap289 := d124
						snap290 := d125
						snap291 := d126
						snap292 := d127
						snap293 := d128
						snap294 := d129
						snap295 := d165
						snap296 := d166
						snap297 := d204
						snap298 := d205
						snap299 := d206
						snap300 := d207
						snap301 := d208
						snap302 := d209
						snap303 := d211
						snap304 := d256
						snap305 := d257
						snap306 := d258
						snap307 := d259
						snap308 := d260
						alloc309 := ctx.SnapshotAllocState()
						bbs[14].Render()
						ctx.RestoreAllocState(alloc309)
						d0 = snap261
						d1 = snap262
						d2 = snap263
						d3 = snap264
						d9 = snap265
						d10 = snap266
						d11 = snap267
						d12 = snap268
						d13 = snap269
						d24 = snap270
						d25 = snap271
						d26 = snap272
						d27 = snap273
						d42 = snap274
						d43 = snap275
						d44 = snap276
						d45 = snap277
						d64 = snap278
						d65 = snap279
						d66 = snap280
						d67 = snap281
						d90 = snap282
						d91 = snap283
						d92 = snap284
						d93 = snap285
						d120 = snap286
						d121 = snap287
						d122 = snap288
						d124 = snap289
						d125 = snap290
						d126 = snap291
						d127 = snap292
						d128 = snap293
						d129 = snap294
						d165 = snap295
						d166 = snap296
						d204 = snap297
						d205 = snap298
						d206 = snap299
						d207 = snap300
						d208 = snap301
						d209 = snap302
						d211 = snap303
						d256 = snap304
						d257 = snap305
						d258 = snap306
						d259 = snap307
						d260 = snap308
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					return result
				}
				bbs[16].Render = func() JITValueDesc {
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
						return result
					}
					bbs[16].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d208)
					if d208.Loc == LocImm {
						d310 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d208.Imm.Float() < -9.223372036854776e+18)}
					} else {
						r7 := ctx.AllocRegExcept(d208.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(14114281232179134464))
						ctx.EmitCmpFloat64(ctx.ScratchReg, d208.Reg)
						d310 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondUnsignedAbove}
						ctx.BindReg(r7, &d310)
					}
					d311 = d310
					ctx.EnsureDesc(&d311)
					if d311.Loc != LocImm && d311.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d311.Loc == LocImm {
						if d311.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitJump(d311.Condition, lbl14)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FreeDesc(&d310)
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap312 := d0
						snap313 := d1
						snap314 := d2
						snap315 := d3
						snap316 := d9
						snap317 := d10
						snap318 := d11
						snap319 := d12
						snap320 := d13
						snap321 := d24
						snap322 := d25
						snap323 := d26
						snap324 := d27
						snap325 := d42
						snap326 := d43
						snap327 := d44
						snap328 := d45
						snap329 := d64
						snap330 := d65
						snap331 := d66
						snap332 := d67
						snap333 := d90
						snap334 := d91
						snap335 := d92
						snap336 := d93
						snap337 := d120
						snap338 := d121
						snap339 := d122
						snap340 := d124
						snap341 := d125
						snap342 := d126
						snap343 := d127
						snap344 := d128
						snap345 := d129
						snap346 := d165
						snap347 := d166
						snap348 := d204
						snap349 := d205
						snap350 := d206
						snap351 := d207
						snap352 := d208
						snap353 := d209
						snap354 := d211
						snap355 := d256
						snap356 := d257
						snap357 := d258
						snap358 := d259
						snap359 := d260
						snap360 := d310
						snap361 := d311
						alloc362 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc362)
						d0 = snap312
						d1 = snap313
						d2 = snap314
						d3 = snap315
						d9 = snap316
						d10 = snap317
						d11 = snap318
						d12 = snap319
						d13 = snap320
						d24 = snap321
						d25 = snap322
						d26 = snap323
						d27 = snap324
						d42 = snap325
						d43 = snap326
						d44 = snap327
						d45 = snap328
						d64 = snap329
						d65 = snap330
						d66 = snap331
						d67 = snap332
						d90 = snap333
						d91 = snap334
						d92 = snap335
						d93 = snap336
						d120 = snap337
						d121 = snap338
						d122 = snap339
						d124 = snap340
						d125 = snap341
						d126 = snap342
						d127 = snap343
						d128 = snap344
						d129 = snap345
						d165 = snap346
						d166 = snap347
						d204 = snap348
						d205 = snap349
						d206 = snap350
						d207 = snap351
						d208 = snap352
						d209 = snap353
						d211 = snap354
						d256 = snap355
						d257 = snap356
						d258 = snap357
						d259 = snap358
						d260 = snap359
						d310 = snap360
						d311 = snap361
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 61,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mod",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			if a[0].IsInt() && a[1].IsInt() {
				b := a[1].Int()
				if b == 0 {
					return NewNil()
				}
				return NewInt(a[0].Int() % b)
			}
			b := a[1].Float()
			if b == 0 {
				return NewNil()
			}
			return NewFloat(float64(int64(a[0].Float()) % int64(b)))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the remainder of integer division (modulo)",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "a", Description: "dividend"},
				{Kind: "number", Label: "b", Description: "divisor"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["mod"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [11]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d12 = d10
					d12.ID = 0
					d11 = ctx.EmitTagEqualsBorrowed(&d12, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d10)
					d13 = d11
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d13.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap14 := d0
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d9
						snap19 := d10
						snap20 := d11
						snap21 := d12
						snap22 := d13
						alloc23 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc23)
						d0 = snap14
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d9 = snap18
						d10 = snap19
						d11 = snap20
						d12 = snap21
						d13 = snap22
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d24 = args[1]
					d24.ID = 0
					d26 = d24
					d26.ID = 0
					d25 = ctx.EmitTagEqualsBorrowed(&d26, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d24)
					d27 = d25
					ctx.EnsureDesc(&d27)
					if d27.Loc != LocImm && d27.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d27.Loc == LocImm {
						if d27.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d27.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap28 := d0
						snap29 := d1
						snap30 := d2
						snap31 := d3
						snap32 := d9
						snap33 := d10
						snap34 := d11
						snap35 := d12
						snap36 := d13
						snap37 := d24
						snap38 := d25
						snap39 := d26
						snap40 := d27
						alloc41 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc41)
						d0 = snap28
						d1 = snap29
						d2 = snap30
						d3 = snap31
						d9 = snap32
						d10 = snap33
						d11 = snap34
						d12 = snap35
						d13 = snap36
						d24 = snap37
						d25 = snap38
						d26 = snap39
						d27 = snap40
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d25)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d42 = args[1]
					d42.ID = 0
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int())}
					} else if d42.Type == tagInt && d42.Loc == LocRegPair {
						ctx.FreeReg(d42.Reg)
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg2}
						ctx.BindReg(d42.Reg2, &d43)
						ctx.BindReg(d42.Reg2, &d43)
					} else if d42.Type == tagInt && d42.Loc == LocReg {
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
						ctx.BindReg(d42.Reg, &d43)
						ctx.BindReg(d42.Reg, &d43)
					} else {
						d43 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d42}, 1)
						d43.Type = tagInt
						ctx.BindReg(d43.Reg, &d43)
					}
					ctx.StabilizeDescForControlFlow(&d43)
					ctx.FreeDesc(&d42)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d43.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocRegExcept(d43.Reg)
						ctx.EmitCmpRegImm32(d43.Reg, 0)
						d44 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d44)
					}
					d45 = d44
					ctx.EnsureDesc(&d45)
					if d45.Loc != LocImm && d45.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d45.Loc == LocImm {
						if d45.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d45.Condition, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d44)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap46 := d0
						snap47 := d1
						snap48 := d2
						snap49 := d3
						snap50 := d9
						snap51 := d10
						snap52 := d11
						snap53 := d12
						snap54 := d13
						snap55 := d24
						snap56 := d25
						snap57 := d26
						snap58 := d27
						snap59 := d42
						snap60 := d43
						snap61 := d44
						snap62 := d45
						alloc63 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc63)
						d0 = snap46
						d1 = snap47
						d2 = snap48
						d3 = snap49
						d9 = snap50
						d10 = snap51
						d11 = snap52
						d12 = snap53
						d13 = snap54
						d24 = snap55
						d25 = snap56
						d26 = snap57
						d27 = snap58
						d42 = snap59
						d43 = snap60
						d44 = snap61
						d45 = snap62
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d64 = args[1]
					d64.ID = 0
					d65 = ctx.EmitFloatDesc(d64)
					ctx.StabilizeDescForControlFlow(&d65)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d65.Imm.Float() == 0)}
					} else {
						r1 := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(0))
						ctx.EmitCmpFloat64Setcc(r1, d65.Reg, ctx.ScratchReg, CondEqual)
						d66 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d66)
					}
					d67 = d66
					ctx.EnsureDesc(&d67)
					if d67.Loc != LocImm && d67.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d67.Loc == LocImm {
						if d67.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[10].Render()
					}
					ctx.EmitCmpRegImm32(d67.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
					}
					ctx.FlushRegisterMoves()
					if !bbs[10].Rendered {
						snap68 := d0
						snap69 := d1
						snap70 := d2
						snap71 := d3
						snap72 := d9
						snap73 := d10
						snap74 := d11
						snap75 := d12
						snap76 := d13
						snap77 := d24
						snap78 := d25
						snap79 := d26
						snap80 := d27
						snap81 := d42
						snap82 := d43
						snap83 := d44
						snap84 := d45
						snap85 := d64
						snap86 := d65
						snap87 := d66
						snap88 := d67
						alloc89 := ctx.SnapshotAllocState()
						bbs[10].Render()
						ctx.RestoreAllocState(alloc89)
						d0 = snap68
						d1 = snap69
						d2 = snap70
						d3 = snap71
						d9 = snap72
						d10 = snap73
						d11 = snap74
						d12 = snap75
						d13 = snap76
						d24 = snap77
						d25 = snap78
						d26 = snap79
						d27 = snap80
						d42 = snap81
						d43 = snap82
						d44 = snap83
						d45 = snap84
						d64 = snap85
						d65 = snap86
						d66 = snap87
						d67 = snap88
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					ctx.FreeDesc(&d66)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d90 = args[1]
					d90.ID = 0
					d92 = d90
					d92.ID = 0
					d91 = ctx.EmitTagEqualsBorrowed(&d92, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d90)
					d93 = d91
					ctx.EnsureDesc(&d93)
					if d93.Loc != LocImm && d93.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d93.Loc == LocImm {
						if d93.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d93.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap94 := d0
						snap95 := d1
						snap96 := d2
						snap97 := d3
						snap98 := d9
						snap99 := d10
						snap100 := d11
						snap101 := d12
						snap102 := d13
						snap103 := d24
						snap104 := d25
						snap105 := d26
						snap106 := d27
						snap107 := d42
						snap108 := d43
						snap109 := d44
						snap110 := d45
						snap111 := d64
						snap112 := d65
						snap113 := d66
						snap114 := d67
						snap115 := d90
						snap116 := d91
						snap117 := d92
						snap118 := d93
						alloc119 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc119)
						d0 = snap94
						d1 = snap95
						d2 = snap96
						d3 = snap97
						d9 = snap98
						d10 = snap99
						d11 = snap100
						d12 = snap101
						d13 = snap102
						d24 = snap103
						d25 = snap104
						d26 = snap105
						d27 = snap106
						d42 = snap107
						d43 = snap108
						d44 = snap109
						d45 = snap110
						d64 = snap111
						d65 = snap112
						d66 = snap113
						d67 = snap114
						d90 = snap115
						d91 = snap116
						d92 = snap117
						d93 = snap118
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d91)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d120 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d120)
					if d120.Loc == LocRegPair || d120.Loc == LocStackPair || d120.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d120, &result)
						result.Type = d120.Type
					} else {
						switch d120.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d120)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d120)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d120)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d120, &result)
							result.Type = d120.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d121 = args[0]
					d121.ID = 0
					if d121.Loc == LocImm {
						d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d121.Imm.Int())}
					} else if d121.Type == tagInt && d121.Loc == LocRegPair {
						ctx.FreeReg(d121.Reg)
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d121.Reg2}
						ctx.BindReg(d121.Reg2, &d122)
						ctx.BindReg(d121.Reg2, &d122)
					} else if d121.Type == tagInt && d121.Loc == LocReg {
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d121.Reg}
						ctx.BindReg(d121.Reg, &d122)
						ctx.BindReg(d121.Reg, &d122)
					} else {
						d122 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d121}, 1)
						d122.Type = tagInt
						ctx.BindReg(d122.Reg, &d122)
					}
					ctx.FreeDesc(&d121)
					ctx.EnsureDesc(&d122)
					resultTarget123 := false
					_ = resultTarget123
					ctx.EnsureDesc(&d43)
					ctx.EnsureDescsTogether(&d122, &d43)
					if d122.Loc == LocImm && d43.Loc == LocImm {
						d124 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d122.Imm.Int() % d43.Imm.Int())}
					} else {
						d124 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d122, d43}, 1)
					}
					if d124.Loc == LocReg && d122.Loc == LocReg && d124.Reg == d122.Reg {
						ctx.TransferReg(d122.Reg)
						d122.Loc = LocNone
					}
					ctx.FreeDesc(&d122)
					ctx.EnsureDesc(&d124)
					if d124.Loc == LocImm {
						ctx.EmitMakeInt(result, d124)
					} else {
						ctx.EmitMovToReg(result.Reg2, d124)
						d125 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d125)
						if d124.Loc == LocReg && d124.Reg != result.Reg2 {
							ctx.FreeReg(d124.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d126 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d126)
					if d126.Loc == LocRegPair || d126.Loc == LocStackPair || d126.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d126, &result)
						result.Type = d126.Type
					} else {
						switch d126.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d126)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d126)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d126)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d126, &result)
							result.Type = d126.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d127 = args[0]
					d127.ID = 0
					d128 = ctx.EmitFloatDesc(d127)
					ctx.FreeDesc(&d127)
					ctx.EnsureDesc(&d128)
					ctx.EnsureDesc(&d128)
					if d128.Loc == LocImm {
						d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d128.Imm.Float()))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r2, d128.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d129)
					}
					ctx.FreeDesc(&d128)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d130 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.Imm.Float()))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r3, d65.Reg)
						d130 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d130)
					}
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d130)
					ctx.EnsureDescsTogether(&d129, &d130)
					if d129.Loc == LocImm && d130.Loc == LocImm {
						d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d129.Imm.Int() % d130.Imm.Int())}
					} else {
						d131 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d129, d130}, 1)
					}
					if d131.Loc == LocReg && d129.Loc == LocReg && d131.Reg == d129.Reg {
						ctx.TransferReg(d129.Reg)
						d129.Loc = LocNone
					}
					ctx.FreeDesc(&d129)
					ctx.FreeDesc(&d130)
					ctx.EnsureDesc(&d131)
					ctx.EnsureDesc(&d131)
					if d131.Loc == LocImm {
						d132 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d131.Imm.Int()))}
					} else {
						var r4 Reg
						r4 = d131.Reg
						d131.Loc = LocNone
						ctx.EmitInt64ToFloatBits(r4)
						d132 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d132)
					}
					ctx.FreeDesc(&d131)
					ctx.EnsureDesc(&d132)
					if d132.Loc == LocImm {
						ctx.EmitMakeFloat(result, d132)
					} else {
						ctx.EmitMovToReg(result.Reg2, d132)
						d133 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d133)
						if d132.Loc == LocReg && d132.Reg != result.Reg2 {
							ctx.FreeReg(d132.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 47,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name:     "<=",
		Optimize: optimizeOrderedComparison,

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			return NewBool(!Less(a[1], a[0]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compares two numbers or strings; returns nil if either value is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "first value"},
				{Kind: "any", Label: "b", Description: "second value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["<="]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [4]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					d11 = args[0]
					d11.ID = 0
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d11)
					d12 = jitEmitLess(ctx, []JITValueDesc{d10, d11}, JITValueDesc{Loc: LocAny})
					d12.Type = tagBool
					ctx.FreeDesc(&d10)
					ctx.FreeDesc(&d11)
					ctx.SyncDesc(&d12)
					if d12.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d12.Imm.Bool())}
					} else if ctx.hasBooleanFlags(d12) {
						d13 = d12
						d13.ID = 0
						d13.Condition = InvertJITCondition(d13.Condition)
						ctx.lazyFlags.Condition = d13.Condition
						ctx.BindReg(d13.Reg, &d13)
						d12.Loc = LocNone
					} else {
						ctx.EnsureDesc(&d12)
						negReg := ctx.AllocReg()
						if d12.Loc == LocRegPair {
							ctx.EmitMovRegReg(negReg, d12.Reg2)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d13)
						} else if d12.Loc == LocReg {
							ctx.EmitMovRegReg(negReg, d12.Reg)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d13)
						} else {
							panic("UnOp ! unsupported source location")
						}
					}
					ctx.FreeDesc(&d12)
					ctx.SyncDesc(&d13)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d13) {
						return d13
					}
					if d13.Loc == LocImm {
						ctx.EmitMakeBool(result, d13)
					} else {
						ctx.EmitMovToReg(result.Reg2, d13)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d14)
						if d13.Loc == LocReg && d13.Reg != result.Reg2 {
							ctx.FreeReg(d13.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d15 = args[1]
					d15.ID = 0
					d17 = d15
					d17.ID = 0
					d16 = ctx.EmitTagEqualsBorrowed(&d17, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d15)
					d18 = d16
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap19 := d0
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d9
						snap24 := d10
						snap25 := d11
						snap26 := d12
						snap27 := d13
						snap28 := d14
						snap29 := d15
						snap30 := d16
						snap31 := d17
						snap32 := d18
						alloc33 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc33)
						d0 = snap19
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d9 = snap23
						d10 = snap24
						d11 = snap25
						d12 = snap26
						d13 = snap27
						d14 = snap28
						d15 = snap29
						d16 = snap30
						d17 = snap31
						d18 = snap32
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d16)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 18,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name:     "<",
		Optimize: optimizeOrderedComparison,

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			return NewBool(Less(a[0], a[1]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compares two numbers or strings; returns nil if either value is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "first value"},
				{Kind: "any", Label: "b", Description: "second value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["<"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [4]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d11 = args[1]
					d11.ID = 0
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d11)
					d13 = JITValueDesc{Loc: LocAny}
					resultTarget14 := result.Loc == LocRegPair
					if resultTarget14 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
					}
					d12 = jitEmitLess(ctx, []JITValueDesc{d10, d11}, d13)
					d12.Type = tagBool
					ctx.FreeDesc(&d10)
					ctx.FreeDesc(&d11)
					ctx.SyncDesc(&d12)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d12) {
						return d12
					}
					if d12.Loc == LocImm {
						ctx.EmitMakeBool(result, d12)
					} else {
						ctx.EmitMovToReg(result.Reg2, d12)
						d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d15)
						if d12.Loc == LocReg && d12.Reg != result.Reg2 {
							ctx.FreeReg(d12.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d16 = args[1]
					d16.ID = 0
					d18 = d16
					d18.ID = 0
					d17 = ctx.EmitTagEqualsBorrowed(&d18, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d16)
					d19 = d17
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap20 := d0
						snap21 := d1
						snap22 := d2
						snap23 := d3
						snap24 := d9
						snap25 := d10
						snap26 := d11
						snap27 := d12
						snap28 := d13
						snap29 := d15
						snap30 := d16
						snap31 := d17
						snap32 := d18
						snap33 := d19
						alloc34 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc34)
						d0 = snap20
						d1 = snap21
						d2 = snap22
						d3 = snap23
						d9 = snap24
						d10 = snap25
						d11 = snap26
						d12 = snap27
						d13 = snap28
						d15 = snap29
						d16 = snap30
						d17 = snap31
						d18 = snap32
						d19 = snap33
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 17,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name:     ">",
		Optimize: optimizeOrderedComparison,

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			return NewBool(Less(a[1], a[0]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compares two numbers or strings; returns nil if either value is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "first value"},
				{Kind: "any", Label: "b", Description: "second value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations[">"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [4]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					d11 = args[0]
					d11.ID = 0
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d11)
					d13 = JITValueDesc{Loc: LocAny}
					resultTarget14 := result.Loc == LocRegPair
					if resultTarget14 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
					}
					d12 = jitEmitLess(ctx, []JITValueDesc{d10, d11}, d13)
					d12.Type = tagBool
					ctx.FreeDesc(&d10)
					ctx.FreeDesc(&d11)
					ctx.SyncDesc(&d12)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d12) {
						return d12
					}
					if d12.Loc == LocImm {
						ctx.EmitMakeBool(result, d12)
					} else {
						ctx.EmitMovToReg(result.Reg2, d12)
						d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d15)
						if d12.Loc == LocReg && d12.Reg != result.Reg2 {
							ctx.FreeReg(d12.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d16 = args[1]
					d16.ID = 0
					d18 = d16
					d18.ID = 0
					d17 = ctx.EmitTagEqualsBorrowed(&d18, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d16)
					d19 = d17
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap20 := d0
						snap21 := d1
						snap22 := d2
						snap23 := d3
						snap24 := d9
						snap25 := d10
						snap26 := d11
						snap27 := d12
						snap28 := d13
						snap29 := d15
						snap30 := d16
						snap31 := d17
						snap32 := d18
						snap33 := d19
						alloc34 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc34)
						d0 = snap20
						d1 = snap21
						d2 = snap22
						d3 = snap23
						d9 = snap24
						d10 = snap25
						d11 = snap26
						d12 = snap27
						d13 = snap28
						d15 = snap29
						d16 = snap30
						d17 = snap31
						d18 = snap32
						d19 = snap33
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 17,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name:     ">=",
		Optimize: optimizeOrderedComparison,

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			return NewBool(!Less(a[0], a[1]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compares two numbers or strings; returns nil if either value is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "first value"},
				{Kind: "any", Label: "b", Description: "second value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations[">="]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [4]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d11 = args[1]
					d11.ID = 0
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d11)
					d12 = jitEmitLess(ctx, []JITValueDesc{d10, d11}, JITValueDesc{Loc: LocAny})
					d12.Type = tagBool
					ctx.FreeDesc(&d10)
					ctx.FreeDesc(&d11)
					ctx.SyncDesc(&d12)
					if d12.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d12.Imm.Bool())}
					} else if ctx.hasBooleanFlags(d12) {
						d13 = d12
						d13.ID = 0
						d13.Condition = InvertJITCondition(d13.Condition)
						ctx.lazyFlags.Condition = d13.Condition
						ctx.BindReg(d13.Reg, &d13)
						d12.Loc = LocNone
					} else {
						ctx.EnsureDesc(&d12)
						negReg := ctx.AllocReg()
						if d12.Loc == LocRegPair {
							ctx.EmitMovRegReg(negReg, d12.Reg2)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d13)
						} else if d12.Loc == LocReg {
							ctx.EmitMovRegReg(negReg, d12.Reg)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d13)
						} else {
							panic("UnOp ! unsupported source location")
						}
					}
					ctx.FreeDesc(&d12)
					ctx.SyncDesc(&d13)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d13) {
						return d13
					}
					if d13.Loc == LocImm {
						ctx.EmitMakeBool(result, d13)
					} else {
						ctx.EmitMovToReg(result.Reg2, d13)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d14)
						if d13.Loc == LocReg && d13.Reg != result.Reg2 {
							ctx.FreeReg(d13.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d15 = args[1]
					d15.ID = 0
					d17 = d15
					d17.ID = 0
					d16 = ctx.EmitTagEqualsBorrowed(&d17, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d15)
					d18 = d16
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap19 := d0
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d9
						snap24 := d10
						snap25 := d11
						snap26 := d12
						snap27 := d13
						snap28 := d14
						snap29 := d15
						snap30 := d16
						snap31 := d17
						snap32 := d18
						alloc33 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc33)
						d0 = snap19
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d9 = snap23
						d10 = snap24
						d11 = snap25
						d12 = snap26
						d13 = snap27
						d14 = snap28
						d15 = snap29
						d16 = snap30
						d17 = snap31
						d18 = snap32
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d16)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 18,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "equal?",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(Equal(a[0], a[1]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compares two values of the same type, (equal? nil nil) is true",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "first value"},
				{Kind: "any", Label: "b", Description: "second value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["equal?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d0, d1}, 1)
				d2.NoHeapPointer = true
				ctx.EmitAndRegImm32(d2.Reg, 1)
				d2.Type = tagBool
				ctx.BindReg(d2.Reg, &d2)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				ctx.SyncDesc(&d2)
				if ctx.hasBooleanFlags(d2) {
					return d2
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d2.Loc == LocImm {
					ctx.EmitMakeBool(result, d2)
				} else {
					ctx.EmitMakeBool(result, d2)
					ctx.FreeReg(d2.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 7,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "equal??",

		Fn: func(a ...Scmer) Scmer {
			return EqualSQL(a[0], a[1])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "first value"},
				{Kind: "any", Label: "b", Description: "second value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["equal??"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(EqualSQL), []JITValueDesc{d0, d1}, 2)
				d2.NoHeapPointer = false
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				if d2.Loc == LocImm {
					if result.Loc == LocAny {
						return d2
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d2)
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d2, &result)
					result.Type = d2.Type
				} else {
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d2)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d2)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d2)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_not",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewBool(!a[0].Bool())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "negates a SQL predicate while preserving nil as UNKNOWN",
			Params: []*TypeDescriptor{{Kind: "any", Label: "value", Description: "SQL predicate value"}},
			// SQL NOT is nullable: UNKNOWN remains UNKNOWN. Advertising a concrete
			// bool lets expression optimization replace it with two-valued `not`.
			Return: &TypeDescriptor{Kind: "bool|nil"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_not"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [3]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d12 = d10
					d12.ID = 0
					d11 = ctx.EmitBoolDesc(&d12, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d10)
					ctx.SyncDesc(&d11)
					if d11.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d11.Imm.Bool())}
					} else if ctx.hasBooleanFlags(d11) {
						d13 = d11
						d13.ID = 0
						d13.Condition = InvertJITCondition(d13.Condition)
						ctx.lazyFlags.Condition = d13.Condition
						ctx.BindReg(d13.Reg, &d13)
						d11.Loc = LocNone
					} else {
						ctx.EnsureDesc(&d11)
						negReg := ctx.AllocReg()
						if d11.Loc == LocRegPair {
							ctx.EmitMovRegReg(negReg, d11.Reg2)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d13)
						} else if d11.Loc == LocReg {
							ctx.EmitMovRegReg(negReg, d11.Reg)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d13)
						} else {
							panic("UnOp ! unsupported source location")
						}
					}
					ctx.FreeDesc(&d11)
					ctx.SyncDesc(&d13)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d13) {
						return d13
					}
					if d13.Loc == LocImm {
						ctx.EmitMakeBool(result, d13)
					} else {
						ctx.EmitMovToReg(result.Reg2, d13)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d14)
						if d13.Loc == LocReg && d13.Reg != result.Reg2 {
							ctx.FreeReg(d13.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 12,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "equal_collate",

		Fn: func(a ...Scmer) Scmer {
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
			return equalCollatedValues(a[0], a[1], coll)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "performs SQL equality with a specified collation (e.g. *_ci case-insensitive, *_bin case-sensitive); returns nil if either arg is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "left side"},
				{Kind: "any", Label: "b", Description: "right side"},
				{Kind: "string", Label: "collation", Description: "collation name"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["equal_collate"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d229 JITValueDesc
				_ = d229
				var d230 JITValueDesc
				_ = d230
				var d231 JITValueDesc
				_ = d231
				var d232 JITValueDesc
				_ = d232
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [11]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[2]
					d10.ID = 0
					d12 = d10
					ctx.SyncDesc(&d12)
					if d12.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d12.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d12.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d12 = tmpScalar
					}
					d12 = JITPrepareScmerGoArg(ctx, d12)
					if d12.Loc != LocRegPair && d12.Loc != LocStackPair && d12.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d12}, 2)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d11.Imm)
						ptrWord, _ := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d11.Imm.String())))
						d11 = tmpPair
					} else if d11.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
						switch d11.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d11)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d11)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d11)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d11)
						d11 = tmpPair
					}
					if d11.Loc != LocRegPair && d11.Loc != LocStackPair && d11.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d11)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d11}, 2)
					d13.NoHeapPointer = false
					ctx.BindReg(d13.Reg, &d13)
					ctx.BindReg(d13.Reg2, &d13)
					ctx.StabilizeDescForControlFlow(&d13)
					d14 = args[0]
					d14.ID = 0
					d15 = d14
					d15.ID = 0
					d16 = ctx.EmitGetTagDesc(&d15, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d14)
					d17 = args[1]
					d17.ID = 0
					d18 = d17
					d18.ID = 0
					d19 = ctx.EmitGetTagDesc(&d18, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x1))}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 1)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d20)
					}
					d21 = d20
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitJump(d21.Condition, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d20)
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap22 := d0
						snap23 := d1
						snap24 := d2
						snap25 := d3
						snap26 := d9
						snap27 := d10
						snap28 := d11
						snap29 := d12
						snap30 := d13
						snap31 := d14
						snap32 := d15
						snap33 := d16
						snap34 := d17
						snap35 := d18
						snap36 := d19
						snap37 := d20
						snap38 := d21
						alloc39 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc39)
						d0 = snap22
						d1 = snap23
						d2 = snap24
						d3 = snap25
						d9 = snap26
						d10 = snap27
						d11 = snap28
						d12 = snap29
						d13 = snap30
						d14 = snap31
						d15 = snap32
						d16 = snap33
						d17 = snap34
						d18 = snap35
						d19 = snap36
						d20 = snap37
						d21 = snap38
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d40 = args[1]
					d40.ID = 0
					d42 = d40
					d42.ID = 0
					d41 = ctx.EmitTagEqualsBorrowed(&d42, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d40)
					d43 = d41
					ctx.EnsureDesc(&d43)
					if d43.Loc != LocImm && d43.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d43.Loc == LocImm {
						if d43.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d43.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap44 := d0
						snap45 := d1
						snap46 := d2
						snap47 := d3
						snap48 := d9
						snap49 := d10
						snap50 := d11
						snap51 := d12
						snap52 := d13
						snap53 := d14
						snap54 := d15
						snap55 := d16
						snap56 := d17
						snap57 := d18
						snap58 := d19
						snap59 := d20
						snap60 := d21
						snap61 := d40
						snap62 := d41
						snap63 := d42
						snap64 := d43
						alloc65 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc65)
						d0 = snap44
						d1 = snap45
						d2 = snap46
						d3 = snap47
						d9 = snap48
						d10 = snap49
						d11 = snap50
						d12 = snap51
						d13 = snap52
						d14 = snap53
						d15 = snap54
						d16 = snap55
						d17 = snap56
						d18 = snap57
						d19 = snap58
						d20 = snap59
						d21 = snap60
						d40 = snap61
						d41 = snap62
						d42 = snap63
						d43 = snap64
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d41)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d66 = args[0]
					d66.ID = 0
					d68 = d66
					ctx.SyncDesc(&d68)
					if d68.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d68.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d68.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d68 = tmpScalar
					}
					d68 = JITPrepareScmerGoArg(ctx, d68)
					if d68.Loc != LocRegPair && d68.Loc != LocStackPair && d68.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d67 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d68}, 2)
					ctx.StabilizeDescForControlFlow(&d67)
					ctx.FreeDesc(&d66)
					d69 = args[1]
					d69.ID = 0
					d71 = d69
					ctx.SyncDesc(&d71)
					if d71.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d71.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d71.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d71 = tmpScalar
					}
					d71 = JITPrepareScmerGoArg(ctx, d71)
					if d71.Loc != LocRegPair && d71.Loc != LocStackPair && d71.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d70 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d71}, 2)
					ctx.StabilizeDescForControlFlow(&d70)
					ctx.FreeDesc(&d69)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d13.Imm)
						ptrWord, _ := d13.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d13.Imm.String())))
						d13 = tmpPair
					} else if d13.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocRegExcept(d13.Reg), Reg2: ctx.AllocRegExcept(d13.Reg)}
						switch d13.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d13)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d13)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d13)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d13)
						d13 = tmpPair
					}
					if d13.Loc != LocRegPair && d13.Loc != LocStackPair && d13.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Contains arg0)")
					}
					d72 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("_ci")}
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d72.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d72.Imm)
						ptrWord, _ := d72.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d72.Imm.String())))
						d72 = tmpPair
					} else if d72.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d72.Type, Reg: ctx.AllocRegExcept(d72.Reg), Reg2: ctx.AllocRegExcept(d72.Reg)}
						switch d72.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d72)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d72)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d72)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d72)
						d72 = tmpPair
					}
					if d72.Loc != LocRegPair && d72.Loc != LocStackPair && d72.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Contains arg1)")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d72)
					d73 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Contains), []JITValueDesc{d13, d72}, 1)
					d73.NoHeapPointer = true
					ctx.EmitAndRegImm32(d73.Reg, 1)
					d73.Type = tagBool
					ctx.BindReg(d73.Reg, &d73)
					ctx.FreeDesc(&d72)
					d74 = d73
					ctx.EnsureDesc(&d74)
					if d74.Loc != LocImm && d74.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d74.Loc == LocImm {
						if d74.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[10].Render()
					}
					ctx.EmitCmpRegImm32(d74.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
					}
					ctx.FlushRegisterMoves()
					if !bbs[10].Rendered {
						snap75 := d0
						snap76 := d1
						snap77 := d2
						snap78 := d3
						snap79 := d9
						snap80 := d10
						snap81 := d11
						snap82 := d12
						snap83 := d13
						snap84 := d14
						snap85 := d15
						snap86 := d16
						snap87 := d17
						snap88 := d18
						snap89 := d19
						snap90 := d20
						snap91 := d21
						snap92 := d40
						snap93 := d41
						snap94 := d42
						snap95 := d43
						snap96 := d66
						snap97 := d67
						snap98 := d68
						snap99 := d69
						snap100 := d70
						snap101 := d71
						snap102 := d72
						snap103 := d73
						snap104 := d74
						alloc105 := ctx.SnapshotAllocState()
						bbs[10].Render()
						ctx.RestoreAllocState(alloc105)
						d0 = snap75
						d1 = snap76
						d2 = snap77
						d3 = snap78
						d9 = snap79
						d10 = snap80
						d11 = snap81
						d12 = snap82
						d13 = snap83
						d14 = snap84
						d15 = snap85
						d16 = snap86
						d17 = snap87
						d18 = snap88
						d19 = snap89
						d20 = snap90
						d21 = snap91
						d40 = snap92
						d41 = snap93
						d42 = snap94
						d43 = snap95
						d66 = snap96
						d67 = snap97
						d68 = snap98
						d69 = snap99
						d70 = snap100
						d71 = snap101
						d72 = snap102
						d73 = snap103
						d74 = snap104
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					ctx.FreeDesc(&d73)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d106 = args[0]
					d106.ID = 0
					d107 = args[1]
					d107.ID = 0
					d106 = JITPrepareScmerGoArg(ctx, d106)
					d107 = JITPrepareScmerGoArg(ctx, d107)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d13.Imm)
						ptrWord, _ := d13.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d13.Imm.String())))
						d13 = tmpPair
					} else if d13.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocRegExcept(d13.Reg), Reg2: ctx.AllocRegExcept(d13.Reg)}
						switch d13.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d13)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d13)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d13)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d13)
						d13 = tmpPair
					}
					if d13.Loc != LocRegPair && d13.Loc != LocStackPair && d13.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (equalCollatedValues arg2)")
					}
					ctx.SyncDesc(&d106)
					ctx.SyncDesc(&d107)
					ctx.SyncDesc(&d13)
					d108 = ctx.EmitGoCallScalar(GoFuncAddr(equalCollatedValues), []JITValueDesc{d106, d107, d13}, 2)
					d108.NoHeapPointer = false
					ctx.BindReg(d108.Reg, &d108)
					ctx.BindReg(d108.Reg2, &d108)
					ctx.FreeDesc(&d106)
					ctx.FreeDesc(&d107)
					ctx.SyncDesc(&d108)
					if d108.Loc == LocRegPair || d108.Loc == LocStackPair || d108.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d108, &result)
						result.Type = d108.Type
					} else {
						switch d108.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d108)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d108)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d108)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d108, &result)
							result.Type = d108.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d109 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d19.Imm.Int()) == uint64(0x1))}
					} else {
						r1 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 1)
						d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d109)
					}
					d110 = d109
					ctx.EnsureDesc(&d110)
					if d110.Loc != LocImm && d110.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d110.Loc == LocImm {
						if d110.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d110.Condition, lbl5)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d109)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap111 := d0
						snap112 := d1
						snap113 := d2
						snap114 := d3
						snap115 := d9
						snap116 := d10
						snap117 := d11
						snap118 := d12
						snap119 := d13
						snap120 := d14
						snap121 := d15
						snap122 := d16
						snap123 := d17
						snap124 := d18
						snap125 := d19
						snap126 := d20
						snap127 := d21
						snap128 := d40
						snap129 := d41
						snap130 := d42
						snap131 := d43
						snap132 := d66
						snap133 := d67
						snap134 := d68
						snap135 := d69
						snap136 := d70
						snap137 := d71
						snap138 := d72
						snap139 := d73
						snap140 := d74
						snap141 := d106
						snap142 := d107
						snap143 := d108
						snap144 := d109
						snap145 := d110
						alloc146 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc146)
						d0 = snap111
						d1 = snap112
						d2 = snap113
						d3 = snap114
						d9 = snap115
						d10 = snap116
						d11 = snap117
						d12 = snap118
						d13 = snap119
						d14 = snap120
						d15 = snap121
						d16 = snap122
						d17 = snap123
						d18 = snap124
						d19 = snap125
						d20 = snap126
						d21 = snap127
						d40 = snap128
						d41 = snap129
						d42 = snap130
						d43 = snap131
						d66 = snap132
						d67 = snap133
						d68 = snap134
						d69 = snap135
						d70 = snap136
						d71 = snap137
						d72 = snap138
						d73 = snap139
						d74 = snap140
						d106 = snap141
						d107 = snap142
						d108 = snap143
						d109 = snap144
						d110 = snap145
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d147 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x2))}
					} else {
						r2 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 2)
						d147 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d147)
					}
					d148 = d147
					ctx.EnsureDesc(&d148)
					if d148.Loc != LocImm && d148.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d148.Loc == LocImm {
						if d148.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d148.Condition, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d147)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap149 := d0
						snap150 := d1
						snap151 := d2
						snap152 := d3
						snap153 := d9
						snap154 := d10
						snap155 := d11
						snap156 := d12
						snap157 := d13
						snap158 := d14
						snap159 := d15
						snap160 := d16
						snap161 := d17
						snap162 := d18
						snap163 := d19
						snap164 := d20
						snap165 := d21
						snap166 := d40
						snap167 := d41
						snap168 := d42
						snap169 := d43
						snap170 := d66
						snap171 := d67
						snap172 := d68
						snap173 := d69
						snap174 := d70
						snap175 := d71
						snap176 := d72
						snap177 := d73
						snap178 := d74
						snap179 := d106
						snap180 := d107
						snap181 := d108
						snap182 := d109
						snap183 := d110
						snap184 := d147
						snap185 := d148
						alloc186 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc186)
						d0 = snap149
						d1 = snap150
						d2 = snap151
						d3 = snap152
						d9 = snap153
						d10 = snap154
						d11 = snap155
						d12 = snap156
						d13 = snap157
						d14 = snap158
						d15 = snap159
						d16 = snap160
						d17 = snap161
						d18 = snap162
						d19 = snap163
						d20 = snap164
						d21 = snap165
						d40 = snap166
						d41 = snap167
						d42 = snap168
						d43 = snap169
						d66 = snap170
						d67 = snap171
						d68 = snap172
						d69 = snap173
						d70 = snap174
						d71 = snap175
						d72 = snap176
						d73 = snap177
						d74 = snap178
						d106 = snap179
						d107 = snap180
						d108 = snap181
						d109 = snap182
						d110 = snap183
						d147 = snap184
						d148 = snap185
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d187 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d19.Imm.Int()) == uint64(0x2))}
					} else {
						r3 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 2)
						d187 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d187)
					}
					d188 = d187
					ctx.EnsureDesc(&d188)
					if d188.Loc != LocImm && d188.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d188.Loc == LocImm {
						if d188.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d188.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d187)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap189 := d0
						snap190 := d1
						snap191 := d2
						snap192 := d3
						snap193 := d9
						snap194 := d10
						snap195 := d11
						snap196 := d12
						snap197 := d13
						snap198 := d14
						snap199 := d15
						snap200 := d16
						snap201 := d17
						snap202 := d18
						snap203 := d19
						snap204 := d20
						snap205 := d21
						snap206 := d40
						snap207 := d41
						snap208 := d42
						snap209 := d43
						snap210 := d66
						snap211 := d67
						snap212 := d68
						snap213 := d69
						snap214 := d70
						snap215 := d71
						snap216 := d72
						snap217 := d73
						snap218 := d74
						snap219 := d106
						snap220 := d107
						snap221 := d108
						snap222 := d109
						snap223 := d110
						snap224 := d147
						snap225 := d148
						snap226 := d187
						snap227 := d188
						alloc228 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc228)
						d0 = snap189
						d1 = snap190
						d2 = snap191
						d3 = snap192
						d9 = snap193
						d10 = snap194
						d11 = snap195
						d12 = snap196
						d13 = snap197
						d14 = snap198
						d15 = snap199
						d16 = snap200
						d17 = snap201
						d18 = snap202
						d19 = snap203
						d20 = snap204
						d21 = snap205
						d40 = snap206
						d41 = snap207
						d42 = snap208
						d43 = snap209
						d66 = snap210
						d67 = snap211
						d68 = snap212
						d69 = snap213
						d70 = snap214
						d71 = snap215
						d72 = snap216
						d73 = snap217
						d74 = snap218
						d106 = snap219
						d107 = snap220
						d108 = snap221
						d109 = snap222
						d110 = snap223
						d147 = snap224
						d148 = snap225
						d187 = snap226
						d188 = snap227
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d67)
					if d67.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d67.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d67.Imm)
						ptrWord, _ := d67.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d67.Imm.String())))
						d67 = tmpPair
					} else if d67.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d67.Type, Reg: ctx.AllocRegExcept(d67.Reg), Reg2: ctx.AllocRegExcept(d67.Reg)}
						switch d67.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d67)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d67)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d67)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d67)
						d67 = tmpPair
					}
					if d67.Loc != LocRegPair && d67.Loc != LocStackPair && d67.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.EqualFold arg0)")
					}
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d70.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d70.Imm)
						ptrWord, _ := d70.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d70.Imm.String())))
						d70 = tmpPair
					} else if d70.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d70.Type, Reg: ctx.AllocRegExcept(d70.Reg), Reg2: ctx.AllocRegExcept(d70.Reg)}
						switch d70.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d70)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d70)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d70)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d70)
						d70 = tmpPair
					}
					if d70.Loc != LocRegPair && d70.Loc != LocStackPair && d70.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.EqualFold arg1)")
					}
					ctx.SyncDesc(&d67)
					ctx.SyncDesc(&d70)
					d229 = ctx.EmitGoCallScalar(GoFuncAddr(strings.EqualFold), []JITValueDesc{d67, d70}, 1)
					d229.NoHeapPointer = true
					ctx.EmitAndRegImm32(d229.Reg, 1)
					d229.Type = tagBool
					ctx.BindReg(d229.Reg, &d229)
					ctx.SyncDesc(&d229)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d229) {
						return d229
					}
					if d229.Loc == LocImm {
						ctx.EmitMakeBool(result, d229)
					} else {
						ctx.EmitMovToReg(result.Reg2, d229)
						d230 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d230)
						if d229.Loc == LocReg && d229.Reg != result.Reg2 {
							ctx.FreeReg(d229.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d67)
					ctx.EnsureDesc(&d70)
					d231 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d67, d70}, 1)
					ctx.EmitAndRegImm32(d231.Reg, 1)
					d231.Type = tagBool
					ctx.BindReg(d231.Reg, &d231)
					ctx.SyncDesc(&d231)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d231) {
						return d231
					}
					if d231.Loc == LocImm {
						ctx.EmitMakeBool(result, d231)
					} else {
						ctx.EmitMovToReg(result.Reg2, d231)
						d232 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d232)
						if d231.Loc == LocReg && d231.Reg != result.Reg2 {
							ctx.FreeReg(d231.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  48,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "notequal_collate",

		Fn: func(a ...Scmer) Scmer {
			r := Globalenv.Vars["equal_collate"].Func()(a[0], a[1], a[2])
			if r.IsNil() {
				return r
			}
			return NewBool(!r.Bool())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "performs SQL inequality with a specified collation; returns nil if either arg is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "a", Description: "left side"},
				{Kind: "any", Label: "b", Description: "right side"},
				{Kind: "string", Label: "collation", Description: "collation name"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["notequal_collate"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var stackArray6 int32
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [3]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					globalLookup0 := Globalenv.Vars[Symbol("equal_collate")]
					ctx.TrackImm(globalLookup0)
					d1 = JITValueDesc{Loc: LocImm, Type: globalLookup0.GetTag(), Imm: globalLookup0, Rooted: true}
					d1 = JITPrepareScmerGoArg(ctx, d1)
					ctx.SyncDesc(&d1)
					d2 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Func), []JITValueDesc{d1}, 1)
					d2.NoHeapPointer = false
					ctx.BindReg(d2.Reg, &d2)
					d3 = args[0]
					d3.ID = 0
					d4 = args[1]
					d4.ID = 0
					d5 = args[2]
					d5.ID = 0
					stackArray6 = ctx.AllocStack(int32(48))
					_ = stackArray6
					ctx.SyncDesc(&d3)
					ctx.EmitStoreScmerToStack(d3, int32(stackArray6)+int32(0))
					ctx.FreeDesc(&d3)
					ctx.SyncDesc(&d4)
					ctx.EmitStoreScmerToStack(d4, int32(stackArray6)+int32(16))
					ctx.FreeDesc(&d4)
					ctx.SyncDesc(&d5)
					ctx.EmitStoreScmerToStack(d5, int32(stackArray6)+int32(32))
					ctx.FreeDesc(&d5)
					d7 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d7
					r0 := ctx.AllocReg()
					r1 := ctx.AllocRegExcept(r0)
					r2 := ctx.AllocRegExcept(r0, r1)
					d8 = JITValueDesc{Loc: LocRegTriple, Type: tagSlice, Reg: r0, Reg2: r1, Reg3: r2}
					ctx.BindReg(r0, &d8)
					ctx.BindReg(r1, &d8)
					ctx.BindReg(r2, &d8)
					ctx.EmitLeaRegMem(d8.Reg, ctx.StackReg, int32(stackArray6))
					ctx.EmitMovRegImm64(d8.Reg2, uint64(3))
					ctx.EmitMovRegImm64(d8.Reg3, uint64(3))
					d9 = ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeGoFunctionSlice), []JITValueDesc{d2, d8}, 2)
					ctx.StabilizeDescForControlFlow(&d9)
					d11 = d9
					d11.ID = 0
					d10 = ctx.EmitTagEqualsBorrowed(&d11, tagNil, JITValueDesc{Loc: LocAny})
					d12 = d10
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap13 := d1
						snap14 := d2
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d7
						snap19 := d8
						snap20 := d9
						snap21 := d10
						snap22 := d11
						snap23 := d12
						alloc24 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc24)
						d1 = snap13
						d2 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d7 = snap18
						d8 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
						d12 = snap23
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d10)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d26 = d9
					d26.ID = 0
					d25 = ctx.EmitBoolDesc(&d26, JITValueDesc{Loc: LocAny})
					ctx.SyncDesc(&d25)
					if d25.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d25.Imm.Bool())}
					} else if ctx.hasBooleanFlags(d25) {
						d27 = d25
						d27.ID = 0
						d27.Condition = InvertJITCondition(d27.Condition)
						ctx.lazyFlags.Condition = d27.Condition
						ctx.BindReg(d27.Reg, &d27)
						d25.Loc = LocNone
					} else {
						ctx.EnsureDesc(&d25)
						negReg := ctx.AllocReg()
						if d25.Loc == LocRegPair {
							ctx.EmitMovRegReg(negReg, d25.Reg2)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d27 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d27)
						} else if d25.Loc == LocReg {
							ctx.EmitMovRegReg(negReg, d25.Reg)
							ctx.EmitAndRegImm32(negReg, 1)
							ctx.EmitCmpRegImm32(negReg, 0)
							ctx.EmitSetcc(negReg, CondEqual)
							d27 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
							ctx.BindReg(negReg, &d27)
						} else {
							panic("UnOp ! unsupported source location")
						}
					}
					ctx.FreeDesc(&d25)
					ctx.SyncDesc(&d27)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d27) {
						return d27
					}
					if d27.Loc == LocImm {
						ctx.EmitMakeBool(result, d27)
					} else {
						ctx.EmitMovToReg(result.Reg2, d27)
						d28 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d28)
						if d27.Loc == LocReg && d27.Reg != result.Reg2 {
							ctx.FreeReg(d27.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      26,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "!",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "negates the boolean value",
			Params: []*TypeDescriptor{
				{Kind: "bool", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["!"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				d2.ID = 0
				d1 := ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d0)
				ctx.SyncDesc(&d1)
				var d3 JITValueDesc
				if d1.Loc == LocImm {
					d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
				} else if ctx.hasBooleanFlags(d1) {
					d3 = d1
					d3.ID = 0
					d3.Condition = InvertJITCondition(d3.Condition)
					ctx.lazyFlags.Condition = d3.Condition
					ctx.BindReg(d3.Reg, &d3)
					d1.Loc = LocNone
				} else {
					ctx.EnsureDesc(&d1)
					negReg := ctx.AllocReg()
					if d1.Loc == LocRegPair {
						ctx.EmitMovRegReg(negReg, d1.Reg2)
						ctx.EmitAndRegImm32(negReg, 1)
						ctx.EmitCmpRegImm32(negReg, 0)
						ctx.EmitSetcc(negReg, CondEqual)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
						ctx.BindReg(negReg, &d3)
					} else if d1.Loc == LocReg {
						ctx.EmitMovRegReg(negReg, d1.Reg)
						ctx.EmitAndRegImm32(negReg, 1)
						ctx.EmitCmpRegImm32(negReg, 0)
						ctx.EmitSetcc(negReg, CondEqual)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
						ctx.BindReg(negReg, &d3)
					} else {
						panic("UnOp ! unsupported source location")
					}
				}
				ctx.FreeDesc(&d1)
				ctx.SyncDesc(&d3)
				if ctx.hasBooleanFlags(d3) {
					return d3
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d3.Loc == LocImm {
					ctx.EmitMakeBool(result, d3)
				} else {
					ctx.EmitMakeBool(result, d3)
					ctx.FreeReg(d3.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "not",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "negates the boolean value",
			Params: []*TypeDescriptor{
				{Kind: "bool", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["not"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				d2.ID = 0
				d1 := ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d0)
				ctx.SyncDesc(&d1)
				var d3 JITValueDesc
				if d1.Loc == LocImm {
					d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
				} else if ctx.hasBooleanFlags(d1) {
					d3 = d1
					d3.ID = 0
					d3.Condition = InvertJITCondition(d3.Condition)
					ctx.lazyFlags.Condition = d3.Condition
					ctx.BindReg(d3.Reg, &d3)
					d1.Loc = LocNone
				} else {
					ctx.EnsureDesc(&d1)
					negReg := ctx.AllocReg()
					if d1.Loc == LocRegPair {
						ctx.EmitMovRegReg(negReg, d1.Reg2)
						ctx.EmitAndRegImm32(negReg, 1)
						ctx.EmitCmpRegImm32(negReg, 0)
						ctx.EmitSetcc(negReg, CondEqual)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
						ctx.BindReg(negReg, &d3)
					} else if d1.Loc == LocReg {
						ctx.EmitMovRegReg(negReg, d1.Reg)
						ctx.EmitAndRegImm32(negReg, 1)
						ctx.EmitCmpRegImm32(negReg, 0)
						ctx.EmitSetcc(negReg, CondEqual)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
						ctx.BindReg(negReg, &d3)
					} else {
						panic("UnOp ! unsupported source location")
					}
				}
				ctx.FreeDesc(&d1)
				ctx.SyncDesc(&d3)
				if ctx.hasBooleanFlags(d3) {
					return d3
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d3.Loc == LocImm {
					ctx.EmitMakeBool(result, d3)
				} else {
					ctx.EmitMakeBool(result, d3)
					ctx.FreeReg(d3.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nil?",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(a[0].IsNil())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns true if value is nil",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["nil?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				d2.ID = 0
				d1 := ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d0)
				ctx.SyncDesc(&d1)
				if ctx.hasBooleanFlags(d1) {
					return d1
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d1.Loc == LocImm {
					ctx.EmitMakeBool(result, d1)
				} else {
					ctx.EmitMakeBool(result, d1)
					ctx.FreeReg(d1.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 5,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "min",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "returns the smallest value",
			Params: []*TypeDescriptor{
				{Kind: "number|string", Label: "value...", Description: "value", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "number|string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["min"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d14 JITValueDesc
				_ = d14
				var dynamicArgOff15 int32
				var dynamicArgOff16 int32
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d54 JITValueDesc
				_ = d54
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d101 JITValueDesc
				_ = d101
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [8]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(16))
					return bbs[1].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d4)
					}
					if d4.Loc == LocReg && d2.Loc == LocReg && d4.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDescsTogether(&d4, &d3)
					if d4.Loc == LocImm && d3.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d3.Imm.Int())}
					} else if d3.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d4.Reg)
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d4.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitCmpInt64(d4.Reg, ctx.ScratchReg)
						}
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d5)
					} else if d4.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d3.Reg)
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d5)
					} else {
						r2 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpInt64(d4.Reg, d3.Reg)
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d5)
					}
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d6.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d5)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						snap12 := d6
						alloc13 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc13)
						d1 = snap7
						d2 = snap8
						d3 = snap9
						d4 = snap10
						d5 = snap11
						d6 = snap12
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						idx := int(d4.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d14 = args[idx]
						d14.ID = 0
					} else {
						ctx.EnsureDesc(&d4)
						dynamicArgOff15 = ctx.AllocStack(16)
						ctx.ProtectReg(d4.Reg)
						lbl9 := ctx.ReserveLabel()
						lbl10 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d4.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl10)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d4.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff15))
							ctx.EmitJmp(lbl9)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl10)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff15))
						ctx.MarkLabel(lbl9)
						ctx.UnprotectReg(d4.Reg)
						d14 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff15), Rooted: true}
					}
					dynamicArgOff16 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d14, int32(dynamicArgOff16))
					ctx.FreeDesc(&d14)
					d14 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff16), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d14)
					d18 = d1
					d18.ID = 0
					d17 = ctx.EmitTagEqualsBorrowed(&d18, tagNil, JITValueDesc{Loc: LocAny})
					d19 = d17
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d4
						snap24 := d5
						snap25 := d6
						snap26 := d14
						snap27 := d17
						snap28 := d18
						snap29 := d19
						alloc30 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc30)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d4 = snap23
						d5 = snap24
						d6 = snap25
						d14 = snap26
						d17 = snap27
						d18 = snap28
						d19 = snap29
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d1, &result)
						result.Type = d1.Type
					} else {
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d1)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d1)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d1)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d1, &result)
							result.Type = d1.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d31 = d14
					if d31.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d31)
					if d31.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d31, int32(bbs[1].PhiBase)+int32(0), 2)
					} else if d31.Loc == LocInputPair {
						ctx.EnsureDesc(&d31)
						ctx.EmitStoreScmerToStack(d31, int32(bbs[1].PhiBase)+int32(0))
					} else if d31.Loc == LocRegPair || d31.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d31, int32(bbs[1].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d31)
						ctx.EmitStoreToStack(d31, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
					}
					d32 = d4
					if d32.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d32)
					ctx.EmitStoreToStack(d32, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					return bbs[1].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d14)
					d34 = d14
					d34.ID = 0
					d33 = ctx.EmitTagEqualsBorrowed(&d34, tagNil, JITValueDesc{Loc: LocAny})
					d35 = d33
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							ctx.SyncDesc(&d4)
							if d4.Loc == LocReg || d4.Loc == LocFPReg {
								ctx.ProtectReg(d4.Reg)
							} else if d4.Loc == LocRegPair {
								ctx.ProtectReg(d4.Reg)
								ctx.ProtectReg(d4.Reg2)
							}
							d36 = d4
							if d36.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d36)
							ctx.EmitStoreToStack(d36, int32(bbs[1].PhiBase)+int32(16))
							if d4.Loc == LocReg || d4.Loc == LocFPReg {
								ctx.UnprotectReg(d4.Reg)
							} else if d4.Loc == LocRegPair {
								ctx.UnprotectReg(d4.Reg)
								ctx.UnprotectReg(d4.Reg2)
							}
							return bbs[1].Render()
						}
						return bbs[7].Render()
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d35.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl8)
					snap37 := d1
					snap38 := d2
					snap39 := d3
					snap40 := d4
					snap41 := d5
					snap42 := d6
					snap43 := d14
					snap44 := d17
					snap45 := d18
					snap46 := d19
					snap47 := d31
					snap48 := d32
					snap49 := d33
					snap50 := d34
					snap51 := d35
					snap52 := d36
					alloc53 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					d54 = d4
					if d54.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d54)
					ctx.EmitStoreToStack(d54, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc53)
					d1 = snap37
					d2 = snap38
					d3 = snap39
					d4 = snap40
					d5 = snap41
					d6 = snap42
					d14 = snap43
					d17 = snap44
					d18 = snap45
					d19 = snap46
					d31 = snap47
					d32 = snap48
					d33 = snap49
					d34 = snap50
					d35 = snap51
					d36 = snap52
					if !bbs[1].Rendered {
						snap55 := d1
						snap56 := d2
						snap57 := d3
						snap58 := d4
						snap59 := d5
						snap60 := d6
						snap61 := d14
						snap62 := d17
						snap63 := d18
						snap64 := d19
						snap65 := d31
						snap66 := d32
						snap67 := d33
						snap68 := d34
						snap69 := d35
						snap70 := d36
						snap71 := d54
						alloc72 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc72)
						d1 = snap55
						d2 = snap56
						d3 = snap57
						d4 = snap58
						d5 = snap59
						d6 = snap60
						d14 = snap61
						d17 = snap62
						d18 = snap63
						d19 = snap64
						d31 = snap65
						d32 = snap66
						d33 = snap67
						d34 = snap68
						d35 = snap69
						d36 = snap70
						d54 = snap71
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d33)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d73 = d14
					if d73.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d73)
					if d73.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d73, int32(bbs[1].PhiBase)+int32(0), 2)
					} else if d73.Loc == LocInputPair {
						ctx.EnsureDesc(&d73)
						ctx.EmitStoreScmerToStack(d73, int32(bbs[1].PhiBase)+int32(0))
					} else if d73.Loc == LocRegPair || d73.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d73, int32(bbs[1].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d73)
						ctx.EmitStoreToStack(d73, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
					}
					d74 = d4
					if d74.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d74)
					ctx.EmitStoreToStack(d74, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					return bbs[1].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d14)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d1)
					d75 = jitEmitLess(ctx, []JITValueDesc{d14, d1}, JITValueDesc{Loc: LocAny})
					d75.Type = tagBool
					d76 = d75
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							return bbs[6].Render()
						}
						ctx.SyncDesc(&d4)
						if d4.Loc == LocReg || d4.Loc == LocFPReg {
							ctx.ProtectReg(d4.Reg)
						} else if d4.Loc == LocRegPair {
							ctx.ProtectReg(d4.Reg)
							ctx.ProtectReg(d4.Reg2)
						}
						d77 = d4
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreToStack(d77, int32(bbs[1].PhiBase)+int32(16))
						if d4.Loc == LocReg || d4.Loc == LocFPReg {
							ctx.UnprotectReg(d4.Reg)
						} else if d4.Loc == LocRegPair {
							ctx.UnprotectReg(d4.Reg)
							ctx.UnprotectReg(d4.Reg2)
						}
						return bbs[1].Render()
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl12)
					snap78 := d1
					snap79 := d2
					snap80 := d3
					snap81 := d4
					snap82 := d5
					snap83 := d6
					snap84 := d14
					snap85 := d17
					snap86 := d18
					snap87 := d19
					snap88 := d31
					snap89 := d32
					snap90 := d33
					snap91 := d34
					snap92 := d35
					snap93 := d36
					snap94 := d54
					snap95 := d73
					snap96 := d74
					snap97 := d75
					snap98 := d76
					snap99 := d77
					alloc100 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					d101 = d4
					if d101.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d101)
					ctx.EmitStoreToStack(d101, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc100)
					d1 = snap78
					d2 = snap79
					d3 = snap80
					d4 = snap81
					d5 = snap82
					d6 = snap83
					d14 = snap84
					d17 = snap85
					d18 = snap86
					d19 = snap87
					d31 = snap88
					d32 = snap89
					d33 = snap90
					d34 = snap91
					d35 = snap92
					d36 = snap93
					d54 = snap94
					d73 = snap95
					d74 = snap96
					d75 = snap97
					d76 = snap98
					d77 = snap99
					if !bbs[1].Rendered {
						snap102 := d1
						snap103 := d2
						snap104 := d3
						snap105 := d4
						snap106 := d5
						snap107 := d6
						snap108 := d14
						snap109 := d17
						snap110 := d18
						snap111 := d19
						snap112 := d31
						snap113 := d32
						snap114 := d33
						snap115 := d34
						snap116 := d35
						snap117 := d36
						snap118 := d54
						snap119 := d73
						snap120 := d74
						snap121 := d75
						snap122 := d76
						snap123 := d77
						snap124 := d101
						alloc125 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc125)
						d1 = snap102
						d2 = snap103
						d3 = snap104
						d4 = snap105
						d5 = snap106
						d6 = snap107
						d14 = snap108
						d17 = snap109
						d18 = snap110
						d19 = snap111
						d31 = snap112
						d32 = snap113
						d33 = snap114
						d34 = snap115
						d35 = snap116
						d36 = snap117
						d54 = snap118
						d73 = snap119
						d74 = snap120
						d75 = snap121
						d76 = snap122
						d77 = snap123
						d101 = snap124
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d75)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 18,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "max",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "returns the highest value",
			Params: []*TypeDescriptor{
				{Kind: "number|string", Label: "value...", Description: "value", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "number|string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["max"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d14 JITValueDesc
				_ = d14
				var dynamicArgOff15 int32
				var dynamicArgOff16 int32
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d54 JITValueDesc
				_ = d54
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d101 JITValueDesc
				_ = d101
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [8]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(16))
					return bbs[1].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d4)
					}
					if d4.Loc == LocReg && d2.Loc == LocReg && d4.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDescsTogether(&d4, &d3)
					if d4.Loc == LocImm && d3.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d3.Imm.Int())}
					} else if d3.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d4.Reg)
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d4.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitCmpInt64(d4.Reg, ctx.ScratchReg)
						}
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d5)
					} else if d4.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d3.Reg)
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d5)
					} else {
						r2 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpInt64(d4.Reg, d3.Reg)
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d5)
					}
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d6.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d5)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						snap12 := d6
						alloc13 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc13)
						d1 = snap7
						d2 = snap8
						d3 = snap9
						d4 = snap10
						d5 = snap11
						d6 = snap12
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						idx := int(d4.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d14 = args[idx]
						d14.ID = 0
					} else {
						ctx.EnsureDesc(&d4)
						dynamicArgOff15 = ctx.AllocStack(16)
						ctx.ProtectReg(d4.Reg)
						lbl9 := ctx.ReserveLabel()
						lbl10 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d4.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl10)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d4.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff15))
							ctx.EmitJmp(lbl9)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl10)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff15))
						ctx.MarkLabel(lbl9)
						ctx.UnprotectReg(d4.Reg)
						d14 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff15), Rooted: true}
					}
					dynamicArgOff16 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d14, int32(dynamicArgOff16))
					ctx.FreeDesc(&d14)
					d14 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff16), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d14)
					d18 = d1
					d18.ID = 0
					d17 = ctx.EmitTagEqualsBorrowed(&d18, tagNil, JITValueDesc{Loc: LocAny})
					d19 = d17
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d4
						snap24 := d5
						snap25 := d6
						snap26 := d14
						snap27 := d17
						snap28 := d18
						snap29 := d19
						alloc30 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc30)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d4 = snap23
						d5 = snap24
						d6 = snap25
						d14 = snap26
						d17 = snap27
						d18 = snap28
						d19 = snap29
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d1, &result)
						result.Type = d1.Type
					} else {
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d1)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d1)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d1)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d1, &result)
							result.Type = d1.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d31 = d14
					if d31.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d31)
					if d31.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d31, int32(bbs[1].PhiBase)+int32(0), 2)
					} else if d31.Loc == LocInputPair {
						ctx.EnsureDesc(&d31)
						ctx.EmitStoreScmerToStack(d31, int32(bbs[1].PhiBase)+int32(0))
					} else if d31.Loc == LocRegPair || d31.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d31, int32(bbs[1].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d31)
						ctx.EmitStoreToStack(d31, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
					}
					d32 = d4
					if d32.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d32)
					ctx.EmitStoreToStack(d32, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					return bbs[1].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d14)
					d34 = d14
					d34.ID = 0
					d33 = ctx.EmitTagEqualsBorrowed(&d34, tagNil, JITValueDesc{Loc: LocAny})
					d35 = d33
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							ctx.SyncDesc(&d4)
							if d4.Loc == LocReg || d4.Loc == LocFPReg {
								ctx.ProtectReg(d4.Reg)
							} else if d4.Loc == LocRegPair {
								ctx.ProtectReg(d4.Reg)
								ctx.ProtectReg(d4.Reg2)
							}
							d36 = d4
							if d36.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d36)
							ctx.EmitStoreToStack(d36, int32(bbs[1].PhiBase)+int32(16))
							if d4.Loc == LocReg || d4.Loc == LocFPReg {
								ctx.UnprotectReg(d4.Reg)
							} else if d4.Loc == LocRegPair {
								ctx.UnprotectReg(d4.Reg)
								ctx.UnprotectReg(d4.Reg2)
							}
							return bbs[1].Render()
						}
						return bbs[7].Render()
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d35.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl8)
					snap37 := d1
					snap38 := d2
					snap39 := d3
					snap40 := d4
					snap41 := d5
					snap42 := d6
					snap43 := d14
					snap44 := d17
					snap45 := d18
					snap46 := d19
					snap47 := d31
					snap48 := d32
					snap49 := d33
					snap50 := d34
					snap51 := d35
					snap52 := d36
					alloc53 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					d54 = d4
					if d54.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d54)
					ctx.EmitStoreToStack(d54, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc53)
					d1 = snap37
					d2 = snap38
					d3 = snap39
					d4 = snap40
					d5 = snap41
					d6 = snap42
					d14 = snap43
					d17 = snap44
					d18 = snap45
					d19 = snap46
					d31 = snap47
					d32 = snap48
					d33 = snap49
					d34 = snap50
					d35 = snap51
					d36 = snap52
					if !bbs[1].Rendered {
						snap55 := d1
						snap56 := d2
						snap57 := d3
						snap58 := d4
						snap59 := d5
						snap60 := d6
						snap61 := d14
						snap62 := d17
						snap63 := d18
						snap64 := d19
						snap65 := d31
						snap66 := d32
						snap67 := d33
						snap68 := d34
						snap69 := d35
						snap70 := d36
						snap71 := d54
						alloc72 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc72)
						d1 = snap55
						d2 = snap56
						d3 = snap57
						d4 = snap58
						d5 = snap59
						d6 = snap60
						d14 = snap61
						d17 = snap62
						d18 = snap63
						d19 = snap64
						d31 = snap65
						d32 = snap66
						d33 = snap67
						d34 = snap68
						d35 = snap69
						d36 = snap70
						d54 = snap71
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d33)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d73 = d14
					if d73.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d73)
					if d73.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d73, int32(bbs[1].PhiBase)+int32(0), 2)
					} else if d73.Loc == LocInputPair {
						ctx.EnsureDesc(&d73)
						ctx.EmitStoreScmerToStack(d73, int32(bbs[1].PhiBase)+int32(0))
					} else if d73.Loc == LocRegPair || d73.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d73, int32(bbs[1].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d73)
						ctx.EmitStoreToStack(d73, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
					}
					d74 = d4
					if d74.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d74)
					ctx.EmitStoreToStack(d74, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					return bbs[1].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d14)
					ctx.StabilizeDescForControlFlow(&d14)
					d75 = jitEmitLess(ctx, []JITValueDesc{d1, d14}, JITValueDesc{Loc: LocAny})
					d75.Type = tagBool
					d76 = d75
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							return bbs[6].Render()
						}
						ctx.SyncDesc(&d4)
						if d4.Loc == LocReg || d4.Loc == LocFPReg {
							ctx.ProtectReg(d4.Reg)
						} else if d4.Loc == LocRegPair {
							ctx.ProtectReg(d4.Reg)
							ctx.ProtectReg(d4.Reg2)
						}
						d77 = d4
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreToStack(d77, int32(bbs[1].PhiBase)+int32(16))
						if d4.Loc == LocReg || d4.Loc == LocFPReg {
							ctx.UnprotectReg(d4.Reg)
						} else if d4.Loc == LocRegPair {
							ctx.UnprotectReg(d4.Reg)
							ctx.UnprotectReg(d4.Reg2)
						}
						return bbs[1].Render()
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl12)
					snap78 := d1
					snap79 := d2
					snap80 := d3
					snap81 := d4
					snap82 := d5
					snap83 := d6
					snap84 := d14
					snap85 := d17
					snap86 := d18
					snap87 := d19
					snap88 := d31
					snap89 := d32
					snap90 := d33
					snap91 := d34
					snap92 := d35
					snap93 := d36
					snap94 := d54
					snap95 := d73
					snap96 := d74
					snap97 := d75
					snap98 := d76
					snap99 := d77
					alloc100 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d4)
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.ProtectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.ProtectReg(d4.Reg)
						ctx.ProtectReg(d4.Reg2)
					}
					d101 = d4
					if d101.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d101)
					ctx.EmitStoreToStack(d101, int32(bbs[1].PhiBase)+int32(16))
					if d4.Loc == LocReg || d4.Loc == LocFPReg {
						ctx.UnprotectReg(d4.Reg)
					} else if d4.Loc == LocRegPair {
						ctx.UnprotectReg(d4.Reg)
						ctx.UnprotectReg(d4.Reg2)
					}
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc100)
					d1 = snap78
					d2 = snap79
					d3 = snap80
					d4 = snap81
					d5 = snap82
					d6 = snap83
					d14 = snap84
					d17 = snap85
					d18 = snap86
					d19 = snap87
					d31 = snap88
					d32 = snap89
					d33 = snap90
					d34 = snap91
					d35 = snap92
					d36 = snap93
					d54 = snap94
					d73 = snap95
					d74 = snap96
					d75 = snap97
					d76 = snap98
					d77 = snap99
					if !bbs[1].Rendered {
						snap102 := d1
						snap103 := d2
						snap104 := d3
						snap105 := d4
						snap106 := d5
						snap107 := d6
						snap108 := d14
						snap109 := d17
						snap110 := d18
						snap111 := d19
						snap112 := d31
						snap113 := d32
						snap114 := d33
						snap115 := d34
						snap116 := d35
						snap117 := d36
						snap118 := d54
						snap119 := d73
						snap120 := d74
						snap121 := d75
						snap122 := d76
						snap123 := d77
						snap124 := d101
						alloc125 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc125)
						d1 = snap102
						d2 = snap103
						d3 = snap104
						d4 = snap105
						d5 = snap106
						d6 = snap107
						d14 = snap108
						d17 = snap109
						d18 = snap110
						d19 = snap111
						d31 = snap112
						d32 = snap113
						d33 = snap114
						d34 = snap115
						d35 = snap116
						d36 = snap117
						d54 = snap118
						d73 = snap119
						d74 = snap120
						d75 = snap121
						d76 = snap122
						d77 = snap123
						d101 = snap124
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d75)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 18,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "floor",

		Fn: func(a ...Scmer) Scmer {
			return NewFloat(math.Floor(a[0].Float()))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "rounds the number down",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["floor"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := ctx.EmitFloatDesc(d0)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				var d2 JITValueDesc
				if d1.Loc == LocImm {
					d2 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Floor(d1.Imm.Float()))}
				} else {
					ctx.EnsureDesc(&d1)
					var d3 JITValueDesc
					if d1.Loc == LocRegPair {
						ctx.FreeReg(d1.Reg)
						d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d3)
						ctx.BindReg(d1.Reg2, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
						ctx.BindReg(d1.Reg, &d3)
						ctx.BindReg(d1.Reg, &d3)
					}
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(JITFloorBits), []JITValueDesc{d3}, 1)
					d2.Type = tagFloat
					ctx.BindReg(d2.Reg, &d2)
				}
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d2.Loc == LocImm {
					ctx.EmitMakeFloat(result, d2)
				} else {
					ctx.EmitMakeFloat(result, d2)
					ctx.FreeReg(d2.Reg)
				}
				result.Type = tagFloat
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "ceil",

		Fn: func(a ...Scmer) Scmer {
			return NewFloat(math.Ceil(a[0].Float()))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "rounds the number up",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["ceil"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := ctx.EmitFloatDesc(d0)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				var d2 JITValueDesc
				if d1.Loc == LocImm {
					d2 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Ceil(d1.Imm.Float()))}
				} else {
					ctx.EnsureDesc(&d1)
					var d3 JITValueDesc
					if d1.Loc == LocRegPair {
						ctx.FreeReg(d1.Reg)
						d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d3)
						ctx.BindReg(d1.Reg2, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
						ctx.BindReg(d1.Reg, &d3)
						ctx.BindReg(d1.Reg, &d3)
					}
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(JITCeilBits), []JITValueDesc{d3}, 1)
					d2.Type = tagFloat
					ctx.BindReg(d2.Reg, &d2)
				}
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d2.Loc == LocImm {
					ctx.EmitMakeFloat(result, d2)
				} else {
					ctx.EmitMakeFloat(result, d2)
					ctx.FreeReg(d2.Reg)
				}
				result.Type = tagFloat
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "round",

		Fn: func(a ...Scmer) Scmer {
			return NewFloat(math.Round(a[0].Float()))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "rounds the number",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["round"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := ctx.EmitFloatDesc(d0)
				ctx.FreeDesc(&d0)
				if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(math.Round), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = true
				ctx.BindReg(d2.Reg, &d2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d2.Loc == LocImm {
					ctx.EmitMakeFloat(result, d2)
				} else {
					ctx.EmitMakeFloat(result, d2)
					ctx.FreeReg(d2.Reg)
				}
				result.Type = tagFloat
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_decimal_output",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			scale := ToInt(a[1])
			if scale < 0 || scale > 15 {
				return a[0]
			}
			value := a[0].Float()
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return a[0]
			}
			return NewFloat(roundSQLDecimalOutput(value, scale))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "normalizes a DECIMAL-derived result to its declared output scale",
			Params: []*TypeDescriptor{
				{Kind: "number|nil", Label: "value", Description: "DECIMAL-derived value"},
				{Kind: "int", Label: "scale", Description: "declared decimal scale"},
			},
			Return: &TypeDescriptor{Kind: "number|nil"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_decimal_output"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d33 JITValueDesc
				_ = d33
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var phiBase75 int32
				_ = phiBase75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [9]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d11.Imm.Int())}
					} else if d11.Type == tagInt && d11.Loc == LocRegPair {
						ctx.FreeReg(d11.Reg)
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg2}
						ctx.BindReg(d11.Reg2, &d12)
						ctx.BindReg(d11.Reg2, &d12)
					} else if d11.Type == tagInt && d11.Loc == LocReg {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg}
						ctx.BindReg(d11.Reg, &d12)
						ctx.BindReg(d11.Reg, &d12)
					} else {
						d12 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d11}, 1)
						d12.Type = tagInt
						ctx.BindReg(d12.Reg, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpRegImm32(d12.Reg, 0)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d14)
					}
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d15.Condition, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d14)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d2
						snap19 := d3
						snap20 := d9
						snap21 := d10
						snap22 := d11
						snap23 := d12
						snap24 := d13
						snap25 := d14
						snap26 := d15
						alloc27 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc27)
						d0 = snap16
						d1 = snap17
						d2 = snap18
						d3 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
						d12 = snap23
						d13 = snap24
						d14 = snap25
						d15 = snap26
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d28 = args[0]
					d28.ID = 0
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d28, &result)
						result.Type = d28.Type
					} else {
						switch d28.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d28)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d28)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d28)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d28, &result)
							result.Type = d28.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
					d29.ID = 0
					d30 = ctx.EmitFloatDesc(d29)
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d30.Imm.Float() != d30.Imm.Float())}
					} else {
						ctx.EnsureDesc(&d30)
						nanSource32 := d30.Reg
						if d30.Loc == LocRegPair {
							nanSource32 = d30.Reg2
						}
						r1 := ctx.AllocRegExcept(nanSource32)
						ctx.EmitCmpFloat64(nanSource32, nanSource32)
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondParity}
						ctx.BindReg(r1, &d31)
					}
					d33 = d31
					ctx.EnsureDesc(&d33)
					if d33.Loc != LocImm && d33.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d33.Loc == LocImm {
						if d33.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d33.Condition, lbl7)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d31)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap34 := d0
						snap35 := d1
						snap36 := d2
						snap37 := d3
						snap38 := d9
						snap39 := d10
						snap40 := d11
						snap41 := d12
						snap42 := d13
						snap43 := d14
						snap44 := d15
						snap45 := d28
						snap46 := d29
						snap47 := d30
						snap48 := d31
						snap49 := d33
						alloc50 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc50)
						d0 = snap34
						d1 = snap35
						d2 = snap36
						d3 = snap37
						d9 = snap38
						d10 = snap39
						d11 = snap40
						d12 = snap41
						d13 = snap42
						d14 = snap43
						d15 = snap44
						d28 = snap45
						d29 = snap46
						d30 = snap47
						d31 = snap48
						d33 = snap49
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d51 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() > 15)}
					} else {
						r2 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpRegImm32(d12.Reg, 15)
						d51 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreater}
						ctx.BindReg(r2, &d51)
					}
					d52 = d51
					ctx.EnsureDesc(&d52)
					if d52.Loc != LocImm && d52.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d52.Loc == LocImm {
						if d52.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d52.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d51)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap53 := d0
						snap54 := d1
						snap55 := d2
						snap56 := d3
						snap57 := d9
						snap58 := d10
						snap59 := d11
						snap60 := d12
						snap61 := d13
						snap62 := d14
						snap63 := d15
						snap64 := d28
						snap65 := d29
						snap66 := d30
						snap67 := d31
						snap68 := d33
						snap69 := d51
						snap70 := d52
						alloc71 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc71)
						d0 = snap53
						d1 = snap54
						d2 = snap55
						d3 = snap56
						d9 = snap57
						d10 = snap58
						d11 = snap59
						d12 = snap60
						d13 = snap61
						d14 = snap62
						d15 = snap63
						d28 = snap64
						d29 = snap65
						d30 = snap66
						d31 = snap67
						d33 = snap68
						d51 = snap69
						d52 = snap70
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d72 = args[0]
					d72.ID = 0
					ctx.SyncDesc(&d72)
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d72, &result)
						result.Type = d72.Type
					} else {
						switch d72.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d72)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d72)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d72)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d72, &result)
							result.Type = d72.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d12)
					d73 = d30
					_ = d73
					ctx.StabilizeDescForControlFlow(&d73)
					d74 = d12
					_ = d74
					ctx.StabilizeDescForControlFlow(&d74)
					phiBase75 = ctx.AllocStack(int32(16))
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					_ = d76
					lbl11 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_2_5 := int32(-1)
					_ = bbpos_2_5
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d74.Loc == LocRegPair || d74.Loc == LocStackPair || d74.Loc == LocRegTriple || d74.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d74)
					d77 = ctx.EmitGoCallScalar(GoFuncAddr(math.Pow10), []JITValueDesc{d74}, 1)
					d77.NoHeapPointer = true
					ctx.BindReg(d77.Reg, &d77)
					ctx.StabilizeDescForControlFlow(&d77)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d77)
					ctx.EnsureDescsTogether(&d73, &d77)
					if d73.Loc == LocImm && d77.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d73.Imm.Float() * d77.Imm.Float())}
					} else if d73.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d77.Reg)
						_, xBits := d73.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d77.Reg)
						d78 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d78)
					} else if d77.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegReg(scratch, d73.Reg)
						_, yBits := d77.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitMulFloat64(scratch, ctx.ScratchReg)
						d78 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d78)
					} else {
						r3 := ctx.AllocRegExcept(d73.Reg, d77.Reg)
						ctx.EmitMovRegReg(r3, d73.Reg)
						ctx.EmitMulFloat64(r3, d77.Reg)
						d78 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r3}
						ctx.BindReg(r3, &d78)
					}
					if d78.Loc == LocReg && d73.Loc == LocReg && d78.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d78)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					if d78.Loc == LocImm {
						d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d78.Imm.Float() != d78.Imm.Float())}
					} else {
						ctx.EnsureDesc(&d78)
						nanSource80 := d78.Reg
						if d78.Loc == LocRegPair {
							nanSource80 = d78.Reg2
						}
						r4 := ctx.AllocRegExcept(nanSource80)
						ctx.EmitCmpFloat64(nanSource80, nanSource80)
						d79 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondParity}
						ctx.BindReg(r4, &d79)
					}
					ctx.ReclaimUntrackedRegs()
					d81 = d79
					ctx.EnsureDesc(&d81)
					if d81.Loc != LocImm && d81.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					if d81.Loc == LocImm {
						if d81.Imm.Bool() {
							ctx.MarkLabel(lbl18)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl15)
						}
					} else {
						ctx.EmitJump(d81.Condition, lbl18)
						ctx.EmitJmp(lbl19)
						ctx.FreeDesc(&d79)
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl15)
					}
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d78.Loc == LocRegPair || d78.Loc == LocStackPair || d78.Loc == LocRegTriple || d78.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d82.Loc == LocRegPair || d82.Loc == LocStackPair || d82.Loc == LocRegTriple || d82.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d82)
					d83 = ctx.EmitGoCallScalar(GoFuncAddr(math.IsInf), []JITValueDesc{d78, d82}, 1)
					d83.NoHeapPointer = true
					ctx.EmitAndRegImm32(d83.Reg, 1)
					d83.Type = tagBool
					ctx.BindReg(d83.Reg, &d83)
					ctx.FreeDesc(&d82)
					ctx.ReclaimUntrackedRegs()
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl21)
							ctx.EmitJmp(lbl14)
						}
					} else {
						ctx.EmitCmpRegImm32(d84.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl20)
						ctx.EmitJmp(lbl21)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d83)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					if d78.Loc == LocImm {
						d85 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d78.Imm.Float() * 2)}
					} else {
						scratch := ctx.AllocRegExcept(d78.Reg)
						ctx.EmitMovRegReg(scratch, d78.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4611686018427387904))
						ctx.EmitMulFloat64(scratch, ctx.ScratchReg)
						d85 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d85)
					}
					if d85.Loc == LocReg && d78.Loc == LocReg && d85.Reg == d78.Reg {
						ctx.TransferReg(d78.Reg)
						d78.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					if d85.Loc == LocRegPair || d85.Loc == LocStackPair || d85.Loc == LocRegTriple || d85.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d85)
					d86 = ctx.EmitGoCallScalar(GoFuncAddr(math.Round), []JITValueDesc{d85}, 1)
					d86.NoHeapPointer = true
					ctx.BindReg(d86.Reg, &d86)
					ctx.FreeDesc(&d85)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d86)
					if d86.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d86.Imm.Float() / 2)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4611686018427387904))
						ctx.EmitDivFloat64(d86.Reg, ctx.ScratchReg)
						d87 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d86.Reg}
						ctx.BindReg(d86.Reg, &d87)
					}
					if d87.Loc == LocReg && d86.Loc == LocReg && d87.Reg == d86.Reg {
						ctx.TransferReg(d86.Reg)
						d86.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d87)
					ctx.FreeDesc(&d86)
					ctx.ReclaimUntrackedRegs()
					d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d88.Loc == LocRegPair || d88.Loc == LocStackPair || d88.Loc == LocRegTriple || d88.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d88)
					d89 = ctx.EmitGoCallScalar(GoFuncAddr(math.Inf), []JITValueDesc{d88}, 1)
					d89.NoHeapPointer = true
					ctx.BindReg(d89.Reg, &d89)
					ctx.FreeDesc(&d88)
					ctx.ReclaimUntrackedRegs()
					if d78.Loc == LocRegPair || d78.Loc == LocStackPair || d78.Loc == LocRegTriple || d78.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d89.Loc == LocRegPair || d89.Loc == LocStackPair || d89.Loc == LocRegTriple || d89.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d89)
					d90 = ctx.EmitGoCallScalar(GoFuncAddr(math.Nextafter), []JITValueDesc{d78, d89}, 1)
					d90.NoHeapPointer = true
					ctx.BindReg(d90.Reg, &d90)
					ctx.FreeDesc(&d89)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d78)
					ctx.EnsureDescsTogether(&d90, &d78)
					if d90.Loc == LocImm && d78.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d90.Imm.Float() - d78.Imm.Float())}
					} else if d90.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d78.Reg)
						_, xBits := d90.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitSubFloat64(scratch, d78.Reg)
						d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d91)
					} else if d78.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d90.Reg)
						ctx.EmitMovRegReg(scratch, d90.Reg)
						_, yBits := d78.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitSubFloat64(scratch, ctx.ScratchReg)
						d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d91)
					} else {
						r5 := ctx.AllocRegExcept(d90.Reg, d78.Reg)
						ctx.EmitMovRegReg(r5, d90.Reg)
						ctx.EmitSubFloat64(r5, d78.Reg)
						d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r5}
						ctx.BindReg(r5, &d91)
					}
					if d91.Loc == LocReg && d90.Loc == LocReg && d91.Reg == d90.Reg {
						ctx.TransferReg(d90.Reg)
						d90.Loc = LocNone
					}
					ctx.FreeDesc(&d90)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d91)
					if d91.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Abs(d91.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d91)
						if d91.Loc == LocRegPair {
							ctx.FreeReg(d91.Reg)
							d93 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d91.Reg2}
							ctx.BindReg(d91.Reg2, &d93)
							ctx.BindReg(d91.Reg2, &d93)
						} else {
							d93 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d91.Reg}
							ctx.BindReg(d91.Reg, &d93)
							ctx.BindReg(d91.Reg, &d93)
						}
						d92 = ctx.EmitGoCallScalar(GoFuncAddr(JITAbsBits), []JITValueDesc{d93}, 1)
						d92.Type = tagFloat
						ctx.BindReg(d92.Reg, &d92)
					}
					ctx.FreeDesc(&d91)
					ctx.ReclaimUntrackedRegs()
					d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d94.Loc == LocRegPair || d94.Loc == LocStackPair || d94.Loc == LocRegTriple || d94.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d94)
					d95 = ctx.EmitGoCallScalar(GoFuncAddr(math.Inf), []JITValueDesc{d94}, 1)
					d95.NoHeapPointer = true
					ctx.BindReg(d95.Reg, &d95)
					ctx.FreeDesc(&d94)
					ctx.ReclaimUntrackedRegs()
					if d78.Loc == LocRegPair || d78.Loc == LocStackPair || d78.Loc == LocRegTriple || d78.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d95.Loc == LocRegPair || d95.Loc == LocStackPair || d95.Loc == LocRegTriple || d95.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d95)
					d96 = ctx.EmitGoCallScalar(GoFuncAddr(math.Nextafter), []JITValueDesc{d78, d95}, 1)
					d96.NoHeapPointer = true
					ctx.BindReg(d96.Reg, &d96)
					ctx.FreeDesc(&d95)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d96)
					ctx.EnsureDescsTogether(&d78, &d96)
					if d78.Loc == LocImm && d96.Loc == LocImm {
						d97 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d78.Imm.Float() - d96.Imm.Float())}
					} else if d78.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d96.Reg)
						_, xBits := d78.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitSubFloat64(scratch, d96.Reg)
						d97 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d97)
					} else if d96.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d78.Reg)
						ctx.EmitMovRegReg(scratch, d78.Reg)
						_, yBits := d96.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitSubFloat64(scratch, ctx.ScratchReg)
						d97 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d97)
					} else {
						r6 := ctx.AllocRegExcept(d78.Reg, d96.Reg)
						ctx.EmitMovRegReg(r6, d78.Reg)
						ctx.EmitSubFloat64(r6, d96.Reg)
						d97 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
						ctx.BindReg(r6, &d97)
					}
					if d97.Loc == LocReg && d78.Loc == LocReg && d97.Reg == d78.Reg {
						ctx.TransferReg(d78.Reg)
						d78.Loc = LocNone
					}
					ctx.FreeDesc(&d96)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					if d97.Loc == LocImm {
						d98 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Abs(d97.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d97)
						if d97.Loc == LocRegPair {
							ctx.FreeReg(d97.Reg)
							d99 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d97.Reg2}
							ctx.BindReg(d97.Reg2, &d99)
							ctx.BindReg(d97.Reg2, &d99)
						} else {
							d99 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d97.Reg}
							ctx.BindReg(d97.Reg, &d99)
							ctx.BindReg(d97.Reg, &d99)
						}
						d98 = ctx.EmitGoCallScalar(GoFuncAddr(JITAbsBits), []JITValueDesc{d99}, 1)
						d98.Type = tagFloat
						ctx.BindReg(d98.Reg, &d98)
					}
					ctx.FreeDesc(&d97)
					ctx.ReclaimUntrackedRegs()
					if d92.Loc == LocRegPair || d92.Loc == LocStackPair || d92.Loc == LocRegTriple || d92.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d98.Loc == LocRegPair || d98.Loc == LocStackPair || d98.Loc == LocRegTriple || d98.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d92)
					ctx.SyncDesc(&d98)
					d100 = ctx.EmitGoCallScalar(GoFuncAddr(math.Max), []JITValueDesc{d92, d98}, 1)
					d100.NoHeapPointer = true
					ctx.BindReg(d100.Reg, &d100)
					ctx.FreeDesc(&d92)
					ctx.FreeDesc(&d98)
					ctx.ReclaimUntrackedRegs()
					d101 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(2)}
					ctx.EnsureDesc(&d100)
					ctx.EnsureDescsTogether(&d101, &d100)
					if d101.Loc == LocImm && d100.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d101.Imm.Float() * d100.Imm.Float())}
					} else if d101.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d100.Reg)
						_, xBits := d101.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d100.Reg)
						d102 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d102)
					} else if d100.Loc == LocImm {
						_, yBits := d100.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitMulFloat64(d101.Reg, ctx.ScratchReg)
						d102 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d101.Reg}
						ctx.BindReg(d101.Reg, &d102)
					} else {
						ctx.EmitMulFloat64(d101.Reg, d100.Reg)
						d102 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d101.Reg}
						ctx.BindReg(d101.Reg, &d102)
					}
					if d102.Loc == LocReg && d101.Loc == LocReg && d102.Reg == d101.Reg {
						ctx.TransferReg(d101.Reg)
						d101.Loc = LocNone
					}
					ctx.FreeDesc(&d100)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d87)
					ctx.EnsureDescsTogether(&d78, &d87)
					if d78.Loc == LocImm && d87.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d78.Imm.Float() - d87.Imm.Float())}
					} else if d78.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d87.Reg)
						_, xBits := d78.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitSubFloat64(scratch, d87.Reg)
						d103 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d103)
					} else if d87.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d78.Reg)
						ctx.EmitMovRegReg(scratch, d78.Reg)
						_, yBits := d87.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitSubFloat64(scratch, ctx.ScratchReg)
						d103 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d103)
					} else {
						r7 := ctx.AllocRegExcept(d78.Reg, d87.Reg)
						ctx.EmitMovRegReg(r7, d78.Reg)
						ctx.EmitSubFloat64(r7, d87.Reg)
						d103 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d103)
					}
					if d103.Loc == LocReg && d78.Loc == LocReg && d103.Reg == d78.Reg {
						ctx.TransferReg(d78.Reg)
						d78.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm {
						d104 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Abs(d103.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d103)
						if d103.Loc == LocRegPair {
							ctx.FreeReg(d103.Reg)
							d105 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d103.Reg2}
							ctx.BindReg(d103.Reg2, &d105)
							ctx.BindReg(d103.Reg2, &d105)
						} else {
							d105 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d103.Reg}
							ctx.BindReg(d103.Reg, &d105)
							ctx.BindReg(d103.Reg, &d105)
						}
						d104 = ctx.EmitGoCallScalar(GoFuncAddr(JITAbsBits), []JITValueDesc{d105}, 1)
						d104.Type = tagFloat
						ctx.BindReg(d104.Reg, &d104)
					}
					ctx.FreeDesc(&d103)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					ctx.EnsureDesc(&d102)
					ctx.EnsureDescsTogether(&d104, &d102)
					if d104.Loc == LocImm && d102.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d104.Imm.Float() <= d102.Imm.Float())}
					} else if d102.Loc == LocImm {
						r8 := ctx.AllocRegExcept(d104.Reg)
						_, yBits := d102.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitCmpFloat64(ctx.ScratchReg, d104.Reg)
						d106 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondUnsignedAboveOrEqual}
						ctx.BindReg(r8, &d106)
					} else if d104.Loc == LocImm {
						r9 := ctx.AllocRegExcept(d102.Reg)
						_, xBits := d104.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
						ctx.EmitCmpFloat64(d102.Reg, ctx.ScratchReg)
						d106 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondUnsignedAboveOrEqual}
						ctx.BindReg(r9, &d106)
					} else {
						r10 := ctx.AllocRegExcept(d104.Reg, d102.Reg)
						ctx.EmitCmpFloat64(d102.Reg, d104.Reg)
						d106 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondUnsignedAboveOrEqual}
						ctx.BindReg(r10, &d106)
					}
					ctx.FreeDesc(&d104)
					ctx.FreeDesc(&d102)
					ctx.ReclaimUntrackedRegs()
					d107 = d106
					ctx.EnsureDesc(&d107)
					if d107.Loc != LocImm && d107.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					if d107.Loc == LocImm {
						if d107.Imm.Bool() {
							ctx.MarkLabel(lbl22)
							ctx.EmitJmp(lbl16)
						} else {
							ctx.MarkLabel(lbl23)
							ctx.SyncDesc(&d78)
							if d78.Loc == LocReg || d78.Loc == LocFPReg {
								ctx.ProtectReg(d78.Reg)
							} else if d78.Loc == LocRegPair {
								ctx.ProtectReg(d78.Reg)
								ctx.ProtectReg(d78.Reg2)
							}
							d108 = d78
							if d108.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d108)
							ctx.EmitStoreToStack(d108, int32(phiBase75)+int32(0))
							if d78.Loc == LocReg || d78.Loc == LocFPReg {
								ctx.UnprotectReg(d78.Reg)
							} else if d78.Loc == LocRegPair {
								ctx.UnprotectReg(d78.Reg)
								ctx.UnprotectReg(d78.Reg2)
							}
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitJump(d107.Condition, lbl22)
						ctx.EmitJmp(lbl23)
						ctx.FreeDesc(&d106)
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl23)
						ctx.SyncDesc(&d78)
						if d78.Loc == LocReg || d78.Loc == LocFPReg {
							ctx.ProtectReg(d78.Reg)
						} else if d78.Loc == LocRegPair {
							ctx.ProtectReg(d78.Reg)
							ctx.ProtectReg(d78.Reg2)
						}
						d109 = d78
						if d109.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d109)
						ctx.EmitStoreToStack(d109, int32(phiBase75)+int32(0))
						if d78.Loc == LocReg || d78.Loc == LocFPReg {
							ctx.UnprotectReg(d78.Reg)
						} else if d78.Loc == LocRegPair {
							ctx.UnprotectReg(d78.Reg)
							ctx.UnprotectReg(d78.Reg2)
						}
						ctx.EmitJmp(lbl17)
					}
					bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d76.Loc == LocRegPair || d76.Loc == LocStackPair || d76.Loc == LocRegTriple || d76.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d76)
					d110 = ctx.EmitGoCallScalar(GoFuncAddr(math.Round), []JITValueDesc{d76}, 1)
					d110.NoHeapPointer = true
					ctx.BindReg(d110.Reg, &d110)
					ctx.FreeDesc(&d76)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EnsureDesc(&d77)
					ctx.EnsureDescsTogether(&d110, &d77)
					if d110.Loc == LocImm && d77.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d110.Imm.Float() / d77.Imm.Float())}
					} else if d110.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d77.Reg)
						_, xBits := d110.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d77.Reg)
						d111 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d111)
					} else if d77.Loc == LocImm {
						_, yBits := d77.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitDivFloat64(d110.Reg, ctx.ScratchReg)
						d111 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d110.Reg}
						ctx.BindReg(d110.Reg, &d111)
					} else {
						ctx.EmitDivFloat64(d110.Reg, d77.Reg)
						d111 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d110.Reg}
						ctx.BindReg(d110.Reg, &d111)
					}
					if d111.Loc == LocReg && d110.Loc == LocReg && d111.Reg == d110.Reg {
						ctx.TransferReg(d110.Reg)
						d110.Loc = LocNone
					}
					ctx.FreeDesc(&d110)
					ctx.ReclaimUntrackedRegs()
					r11 := ctx.AllocReg()
					ctx.EnsureDesc(&d111)
					ctx.EnsureDesc(&d111)
					if d111.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r11, d111)
					}
					ctx.EmitJmp(lbl11)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r11, d73)
					}
					ctx.EmitJmp(lbl11)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d76 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase75) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d87)
					if d87.Loc == LocReg || d87.Loc == LocFPReg {
						ctx.ProtectReg(d87.Reg)
					} else if d87.Loc == LocRegPair {
						ctx.ProtectReg(d87.Reg)
						ctx.ProtectReg(d87.Reg2)
					}
					d112 = d87
					if d112.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d112)
					ctx.EmitStoreToStack(d112, int32(phiBase75)+int32(0))
					if d87.Loc == LocReg || d87.Loc == LocFPReg {
						ctx.UnprotectReg(d87.Reg)
					} else if d87.Loc == LocRegPair {
						ctx.UnprotectReg(d87.Reg)
						ctx.UnprotectReg(d87.Reg2)
					}
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl11)
					d113 = JITValueDesc{Loc: LocReg, Reg: r11}
					ctx.BindReg(r11, &d113)
					ctx.BindReg(r11, &d113)
					ctx.EnsureDesc(&d113)
					if d113.Loc == LocImm {
						ctx.EmitMakeFloat(result, d113)
					} else {
						ctx.EmitMovToReg(result.Reg2, d113)
						d114 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d114)
						if d113.Loc == LocReg && d113.Reg != result.Reg2 {
							ctx.FreeReg(d113.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d115 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d115.Loc == LocRegPair || d115.Loc == LocStackPair || d115.Loc == LocRegTriple || d115.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d30)
					ctx.SyncDesc(&d115)
					d116 = ctx.EmitGoCallScalar(GoFuncAddr(math.IsInf), []JITValueDesc{d30, d115}, 1)
					d116.NoHeapPointer = true
					ctx.EmitAndRegImm32(d116.Reg, 1)
					d116.Type = tagBool
					ctx.BindReg(d116.Reg, &d116)
					ctx.FreeDesc(&d115)
					d117 = d116
					ctx.EnsureDesc(&d117)
					if d117.Loc != LocImm && d117.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d117.Loc == LocImm {
						if d117.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d117.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap118 := d0
						snap119 := d1
						snap120 := d2
						snap121 := d3
						snap122 := d9
						snap123 := d10
						snap124 := d11
						snap125 := d12
						snap126 := d13
						snap127 := d14
						snap128 := d15
						snap129 := d28
						snap130 := d29
						snap131 := d30
						snap132 := d31
						snap133 := d33
						snap134 := d51
						snap135 := d52
						snap136 := d72
						snap137 := d73
						snap138 := d74
						snap139 := d76
						snap140 := d77
						snap141 := d78
						snap142 := d79
						snap143 := d81
						snap144 := d82
						snap145 := d83
						snap146 := d84
						snap147 := d85
						snap148 := d86
						snap149 := d87
						snap150 := d88
						snap151 := d89
						snap152 := d90
						snap153 := d91
						snap154 := d92
						snap155 := d93
						snap156 := d94
						snap157 := d95
						snap158 := d96
						snap159 := d97
						snap160 := d98
						snap161 := d99
						snap162 := d100
						snap163 := d101
						snap164 := d102
						snap165 := d103
						snap166 := d104
						snap167 := d105
						snap168 := d106
						snap169 := d107
						snap170 := d108
						snap171 := d109
						snap172 := d110
						snap173 := d111
						snap174 := d112
						snap175 := d113
						snap176 := d114
						snap177 := d115
						snap178 := d116
						snap179 := d117
						alloc180 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc180)
						d0 = snap118
						d1 = snap119
						d2 = snap120
						d3 = snap121
						d9 = snap122
						d10 = snap123
						d11 = snap124
						d12 = snap125
						d13 = snap126
						d14 = snap127
						d15 = snap128
						d28 = snap129
						d29 = snap130
						d30 = snap131
						d31 = snap132
						d33 = snap133
						d51 = snap134
						d52 = snap135
						d72 = snap136
						d73 = snap137
						d74 = snap138
						d76 = snap139
						d77 = snap140
						d78 = snap141
						d79 = snap142
						d81 = snap143
						d82 = snap144
						d83 = snap145
						d84 = snap146
						d85 = snap147
						d86 = snap148
						d87 = snap149
						d88 = snap150
						d89 = snap151
						d90 = snap152
						d91 = snap153
						d92 = snap154
						d93 = snap155
						d94 = snap156
						d95 = snap157
						d96 = snap158
						d97 = snap159
						d98 = snap160
						d99 = snap161
						d100 = snap162
						d101 = snap163
						d102 = snap164
						d103 = snap165
						d104 = snap166
						d105 = snap167
						d106 = snap168
						d107 = snap169
						d108 = snap170
						d109 = snap171
						d110 = snap172
						d111 = snap173
						d112 = snap174
						d113 = snap175
						d114 = snap176
						d115 = snap177
						d116 = snap178
						d117 = snap179
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d116)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 61,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_abs",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "SQL ABS(): returns absolute value, NULL-safe",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_abs"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d30 JITValueDesc
				_ = d30
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [8]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d4 = d2
					d4.ID = 0
					d3 = ctx.EmitTagEqualsBorrowed(&d4, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d2)
					d5 = d3
					ctx.EnsureDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						snap10 := d5
						alloc11 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc11)
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
						d5 = snap10
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d3)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d12 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d12)
					if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d12, &result)
						result.Type = d12.Type
					} else {
						switch d12.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d12)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d12)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d12)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d12, &result)
							result.Type = d12.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d13 = args[0]
					d13.ID = 0
					d14 = ctx.EmitFloatDesc(d13)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Float() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d14.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(0))
						ctx.EmitCmpFloat64(ctx.ScratchReg, d14.Reg)
						d15 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondUnsignedAbove}
						ctx.BindReg(r0, &d15)
					}
					d16 = d15
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.SyncDesc(&d14)
						if d14.Loc == LocReg || d14.Loc == LocFPReg {
							ctx.ProtectReg(d14.Reg)
						} else if d14.Loc == LocRegPair {
							ctx.ProtectReg(d14.Reg)
							ctx.ProtectReg(d14.Reg2)
						}
						d17 = d14
						if d17.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d17)
						ctx.EmitStoreToStack(d17, int32(bbs[4].PhiBase)+int32(0))
						if d14.Loc == LocReg || d14.Loc == LocFPReg {
							ctx.UnprotectReg(d14.Reg)
						} else if d14.Loc == LocRegPair {
							ctx.UnprotectReg(d14.Reg)
							ctx.UnprotectReg(d14.Reg2)
						}
						return bbs[4].Render()
					}
					lbl9 := ctx.ReserveLabel()
					ctx.EmitJump(d16.Condition, lbl4)
					ctx.EmitJmp(lbl9)
					ctx.FreeDesc(&d15)
					snap18 := d1
					snap19 := d2
					snap20 := d3
					snap21 := d4
					snap22 := d5
					snap23 := d12
					snap24 := d13
					snap25 := d14
					snap26 := d15
					snap27 := d16
					snap28 := d17
					alloc29 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d30 = d14
					if d30.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d30)
					ctx.EmitStoreToStack(d30, int32(bbs[4].PhiBase)+int32(0))
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc29)
					d1 = snap18
					d2 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					d12 = snap23
					d13 = snap24
					d14 = snap25
					d15 = snap26
					d16 = snap27
					d17 = snap28
					if !bbs[4].Rendered {
						snap31 := d1
						snap32 := d2
						snap33 := d3
						snap34 := d4
						snap35 := d5
						snap36 := d12
						snap37 := d13
						snap38 := d14
						snap39 := d15
						snap40 := d16
						snap41 := d17
						snap42 := d30
						alloc43 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc43)
						d1 = snap31
						d2 = snap32
						d3 = snap33
						d4 = snap34
						d5 = snap35
						d12 = snap36
						d13 = snap37
						d14 = snap38
						d15 = snap39
						d16 = snap40
						d17 = snap41
						d30 = snap42
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						if d14.Type == tagFloat {
							d44 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d14.Imm.Float())}
						} else {
							d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d14.Imm.Int())}
						}
					} else {
						if d14.Type == tagFloat {
							r1 := ctx.AllocRegExcept(d14.Reg)
							ctx.EmitMovRegImm64(r1, 0)
							ctx.EmitSubFloat64(r1, d14.Reg)
							d44 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
							ctx.BindReg(r1, &d44)
						} else {
							r2 := ctx.AllocRegExcept(d14.Reg)
							ctx.EmitMovRegImm64(r2, 0)
							ctx.EmitSubInt64(r2, d14.Reg)
							d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
							ctx.BindReg(r2, &d44)
						}
					}
					ctx.StabilizeDescForControlFlow(&d44)
					ctx.SyncDesc(&d44)
					if d44.Loc == LocReg || d44.Loc == LocFPReg {
						ctx.ProtectReg(d44.Reg)
					} else if d44.Loc == LocRegPair {
						ctx.ProtectReg(d44.Reg)
						ctx.ProtectReg(d44.Reg2)
					}
					d45 = d44
					if d45.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d45)
					ctx.EmitStoreToStack(d45, int32(bbs[4].PhiBase)+int32(0))
					if d44.Loc == LocReg || d44.Loc == LocFPReg {
						ctx.UnprotectReg(d44.Reg)
					} else if d44.Loc == LocRegPair {
						ctx.UnprotectReg(d44.Reg)
						ctx.UnprotectReg(d44.Reg2)
					}
					return bbs[4].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d46 = args[0]
					d46.ID = 0
					ctx.EnsureDesc(&d46)
					d47 = d46
					_ = d47
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d47.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int())}
					} else if d47.Type == tagInt && d47.Loc == LocRegPair {
						ctx.FreeReg(d47.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg2}
						ctx.BindReg(d47.Reg2, &d48)
						ctx.BindReg(d47.Reg2, &d48)
					} else if d47.Type == tagInt && d47.Loc == LocReg {
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d48)
						ctx.BindReg(d47.Reg, &d48)
					} else {
						d48 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d47}, 1)
						d48.Type = tagInt
						ctx.BindReg(d48.Reg, &d48)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					ctx.FreeDesc(&d46)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.Imm.Float()))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r3, d1.Reg)
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d50)
					}
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d50)
					ctx.EnsureDescsTogether(&d48, &d50)
					if d48.Loc == LocImm && d50.Loc == LocImm {
						d51 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d48.Imm.Int() == d50.Imm.Int())}
					} else if d50.Loc == LocImm {
						r4 := ctx.AllocReg()
						if d50.Imm.Int() >= -2147483648 && d50.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d48.Reg, int32(d50.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d50.Imm.Int()))
							ctx.EmitCmpInt64(d48.Reg, ctx.ScratchReg)
						}
						d51 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
						ctx.BindReg(r4, &d51)
					} else if d48.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d48.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d50.Reg)
						d51 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
						ctx.BindReg(r5, &d51)
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitCmpInt64(d48.Reg, d50.Reg)
						d51 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
						ctx.BindReg(r6, &d51)
					}
					ctx.FreeDesc(&d48)
					ctx.FreeDesc(&d50)
					d52 = d51
					ctx.EnsureDesc(&d52)
					if d52.Loc != LocImm && d52.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d52.Loc == LocImm {
						if d52.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d52.Condition, lbl8)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d51)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap53 := d1
						snap54 := d2
						snap55 := d3
						snap56 := d4
						snap57 := d5
						snap58 := d12
						snap59 := d13
						snap60 := d14
						snap61 := d15
						snap62 := d16
						snap63 := d17
						snap64 := d30
						snap65 := d44
						snap66 := d45
						snap67 := d46
						snap68 := d47
						snap69 := d48
						snap70 := d49
						snap71 := d50
						snap72 := d51
						snap73 := d52
						alloc74 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc74)
						d1 = snap53
						d2 = snap54
						d3 = snap55
						d4 = snap56
						d5 = snap57
						d12 = snap58
						d13 = snap59
						d14 = snap60
						d15 = snap61
						d16 = snap62
						d17 = snap63
						d30 = snap64
						d44 = snap65
						d45 = snap66
						d46 = snap67
						d47 = snap68
						d48 = snap69
						d49 = snap70
						d50 = snap71
						d51 = snap72
						d52 = snap73
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.Imm.Float()))}
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r7, d1.Reg)
						d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d75)
					}
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						ctx.EmitMakeInt(result, d75)
					} else {
						ctx.EmitMovToReg(result.Reg2, d75)
						d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d76)
						if d75.Loc == LocReg && d75.Reg != result.Reg2 {
							ctx.FreeReg(d75.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						ctx.EmitMakeFloat(result, d1)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1)
						d77 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d77)
						if d1.Loc == LocReg && d1.Reg != result.Reg2 {
							ctx.FreeReg(d1.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d78 = args[0]
					d78.ID = 0
					d79 = ctx.EmitFloatDesc(d78)
					ctx.FreeDesc(&d78)
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d79, &d1)
					if d79.Loc == LocImm && d1.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d79.Imm.Float() == d1.Imm.Float())}
					} else if d1.Loc == LocImm {
						r8 := ctx.AllocRegExcept(d79.Reg)
						_, yBits := d1.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
						ctx.EmitCmpFloat64Setcc(r8, d79.Reg, ctx.ScratchReg, CondEqual)
						d80 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d80)
					} else if d79.Loc == LocImm {
						r9 := ctx.AllocRegExcept(d1.Reg)
						_, xBits := d79.Imm.RawWords()
						ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
						ctx.EmitCmpFloat64Setcc(r9, ctx.ScratchReg, d1.Reg, CondEqual)
						d80 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d80)
					} else {
						r10 := ctx.AllocRegExcept(d79.Reg, d1.Reg)
						ctx.EmitCmpFloat64Setcc(r10, d79.Reg, d1.Reg, CondEqual)
						d80 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d80)
					}
					ctx.FreeDesc(&d79)
					d81 = d80
					ctx.EnsureDesc(&d81)
					if d81.Loc != LocImm && d81.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d81.Loc == LocImm {
						if d81.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d81.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap82 := d1
						snap83 := d2
						snap84 := d3
						snap85 := d4
						snap86 := d5
						snap87 := d12
						snap88 := d13
						snap89 := d14
						snap90 := d15
						snap91 := d16
						snap92 := d17
						snap93 := d30
						snap94 := d44
						snap95 := d45
						snap96 := d46
						snap97 := d47
						snap98 := d48
						snap99 := d49
						snap100 := d50
						snap101 := d51
						snap102 := d52
						snap103 := d75
						snap104 := d76
						snap105 := d77
						snap106 := d78
						snap107 := d79
						snap108 := d80
						snap109 := d81
						alloc110 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc110)
						d1 = snap82
						d2 = snap83
						d3 = snap84
						d4 = snap85
						d5 = snap86
						d12 = snap87
						d13 = snap88
						d14 = snap89
						d15 = snap90
						d16 = snap91
						d17 = snap92
						d30 = snap93
						d44 = snap94
						d45 = snap95
						d46 = snap96
						d47 = snap97
						d48 = snap98
						d49 = snap99
						d50 = snap100
						d51 = snap101
						d52 = snap102
						d75 = snap103
						d76 = snap104
						d77 = snap105
						d78 = snap106
						d79 = snap107
						d80 = snap108
						d81 = snap109
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d80)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 33,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sqrt",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				return NewNil()
			}
			return NewFloat(math.Sqrt(v))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the square root of a number",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "value", Description: "value"},
			},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sqrt"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [5]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d11 = ctx.EmitFloatDesc(d10)
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Float() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(0))
						ctx.EmitCmpFloat64(ctx.ScratchReg, d11.Reg)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondUnsignedAbove}
						ctx.BindReg(r0, &d12)
					}
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d13.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d12)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap14 := d0
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d9
						snap19 := d10
						snap20 := d11
						snap21 := d12
						snap22 := d13
						alloc23 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc23)
						d0 = snap14
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d9 = snap18
						d10 = snap19
						d11 = snap20
						d12 = snap21
						d13 = snap22
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d24 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d24)
					if d24.Loc == LocRegPair || d24.Loc == LocStackPair || d24.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d24, &result)
						result.Type = d24.Type
					} else {
						switch d24.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d24)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d24)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d24)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d24, &result)
							result.Type = d24.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d11.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d11)
						if d11.Loc == LocRegPair {
							ctx.FreeReg(d11.Reg)
							d26 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d11.Reg2}
							ctx.BindReg(d11.Reg2, &d26)
							ctx.BindReg(d11.Reg2, &d26)
						} else {
							d26 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d11.Reg}
							ctx.BindReg(d11.Reg, &d26)
							ctx.BindReg(d11.Reg, &d26)
						}
						d25 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d26}, 1)
						d25.Type = tagFloat
						ctx.BindReg(d25.Reg, &d25)
					}
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						ctx.EmitMakeFloat(result, d25)
					} else {
						ctx.EmitMovToReg(result.Reg2, d25)
						d27 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d27)
						if d25.Loc == LocReg && d25.Reg != result.Reg2 {
							ctx.FreeReg(d25.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 16,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_rand",

		Fn: func(a ...Scmer) Scmer {
			var buf [8]byte
			if _, err := crand.Read(buf[:]); err != nil {
				panic("sql_rand: " + err.Error())
			}
			// 53 random bits map exactly into float64 mantissa range.
			u := binary.LittleEndian.Uint64(buf[:]) >> 11
			return NewFloat(float64(u) / (1 << 53))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "SQL RAND(): returns a random float in [0,1)",
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				declaration := declarations["sql_rand"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
	})
}
