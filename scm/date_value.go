/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
package scm

import "math"
import "time"
import "unsafe"
import "strings"

// NewDateNanoWithZone retains the existing second/zone payload and allocates
// immutable subsecond storage only when the value has a fractional clock.
func NewDateNanoWithZone(seconds int64, nanosecond, zone int) Scmer {
	if seconds < -(1<<44) || seconds >= 1<<44 || nanosecond < 0 || nanosecond >= 1000000000 || zone < 0 || zone > 2047 {
		panic("date parts out of range")
	}
	value := NewDateWithZone(seconds, zone)
	if nanosecond != 0 {
		fraction := uint32(nanosecond)
		value.ptr = (*byte)(unsafe.Pointer(&fraction))
	}
	return value
}

func (value Scmer) DateNanosecond() int {
	if value.GetTag() != tagDate || value.ptr == nil {
		return 0
	}
	return int(*(*uint32)(unsafe.Pointer(value.ptr)))
}

func (value Scmer) DateZone() int { return TagDateDecodeZone(auxVal(value.aux)) }

// ParseDateValue preserves all supplied nanoseconds. The language-independent
// calendar parser rejects extra digits instead of silently discarding them.
func ParseDateValue(text string) (Scmer, bool) {
	if isMySQLZeroDate(text) {
		return NewDate(mysqlZeroDateUnix), true
	}
	if clock := strings.LastIndexByte(text, ':'); clock >= 0 {
		if dot := strings.IndexByte(text[clock:], '.'); dot >= 0 {
			fraction := text[clock+dot+1:]
			if len(fraction) > 9 {
				return NewNil(), false
			}
		}
	}
	for _, layout := range allowedDateFormats {
		if value, err := time.Parse(layout, text); err == nil {
			return NewDateNanoWithZone(value.Unix(), value.Nanosecond(), 0), true
		}
	}
	return NewNil(), false
}

// Both operands are native dates. Decode their known payload directly; the
// generic Int accessor takes its coercion path for every non-integer scalar.
func compareDates(left, right Scmer) int {
	leftSeconds := TagDateDecodeUnix(auxVal(left.aux))
	rightSeconds := TagDateDecodeUnix(auxVal(right.aux))
	if leftSeconds < rightSeconds {
		return -1
	}
	if leftSeconds > rightSeconds {
		return 1
	}
	leftNanos, rightNanos := left.DateNanosecond(), right.DateNanosecond()
	if leftNanos < rightNanos {
		return -1
	}
	if leftNanos > rightNanos {
		return 1
	}
	return 0
}

func initDateParts() {
	Declare(&Globalenv, &Declaration{Name: "date_from_parts", Fn: func(args ...Scmer) Scmer {
		for _, part := range args {
			if !part.IsInt() && !(part.IsFloat() && part.Float() == math.Trunc(part.Float())) {
				panic("date parts require whole numbers")
			}
		}
		return NewDateNanoWithZone(args[0].Int(), int(args[1].Int()), int(args[2].Int()))
	}, Type: &TypeDescriptor{
		Kind: "func", Const: true,
		Params:        []*TypeDescriptor{{Kind: "number"}, {Kind: "number"}, {Kind: "number"}},
		Return:        &TypeDescriptor{Kind: "date"},
		JITInlineCost: 32,
		JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			declaration := declarations["date_from_parts"]
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
			var dynamicArgOff13 int32
			var dynamicArgOff14 int32
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d17 JITValueDesc
			_ = d17
			var d18 JITValueDesc
			_ = d18
			var d30 JITValueDesc
			_ = d30
			var d43 JITValueDesc
			_ = d43
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
			var d53 JITValueDesc
			_ = d53
			var d54 JITValueDesc
			_ = d54
			var d79 JITValueDesc
			_ = d79
			var d80 JITValueDesc
			_ = d80
			var d81 JITValueDesc
			_ = d81
			var d82 JITValueDesc
			_ = d82
			var d83 JITValueDesc
			_ = d83
			var d84 JITValueDesc
			_ = d84
			var d115 JITValueDesc
			_ = d115
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
			var bbs [7]BBDescriptor
			bbs[1].PhiBase = int32(phiBase0) + int32(0)
			for i := range args {
				ctx.StabilizeDescForControlFlow(&args[i])
			}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				ctx.ReclaimUntrackedRegs()
				d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
				_ = d2
				ctx.StabilizeDescForControlFlow(&d2)
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
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocImm {
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d1.Reg)
					ctx.EmitMovRegReg(scratch, d1.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d3)
				}
				if d3.Loc == LocReg && d1.Loc == LocReg && d3.Reg == d1.Reg {
					ctx.TransferReg(d1.Reg)
					d1.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d3)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d2)
				ctx.EnsureDescsTogether(&d3, &d2)
				if d3.Loc == LocImm && d2.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d2.Imm.Int())}
				} else if d2.Loc == LocImm {
					r0 := ctx.AllocRegExcept(d3.Reg)
					if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d3.Reg, int32(d2.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
					}
					d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
					ctx.BindReg(r0, &d4)
				} else if d3.Loc == LocImm {
					r1 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
					d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
					ctx.BindReg(r1, &d4)
				} else {
					r2 := ctx.AllocRegExcept(d3.Reg)
					ctx.EmitCmpInt64(d3.Reg, d2.Reg)
					d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
					ctx.BindReg(r2, &d4)
				}
				d5 = d4
				ctx.SyncDesc(&d5)
				if d5.Loc != LocImm && d5.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				if d5.Loc == LocImm {
					if d5.Imm.Bool() {
						return bbs[2].Render()
					}
					return bbs[3].Render()
				}
				ctx.EmitJump(d5.Condition, lbl3)
				if bbs[3].Rendered {
					ctx.EmitJmp(lbl4)
				}
				ctx.FreeDesc(&d4)
				ctx.FlushRegisterMoves()
				if !bbs[3].Rendered {
					snap6 := d1
					snap7 := d2
					snap8 := d3
					snap9 := d4
					snap10 := d5
					alloc11 := ctx.SnapshotAllocState()
					bbs[3].Render()
					ctx.RestoreAllocState(alloc11)
					d1 = snap6
					d2 = snap7
					d3 = snap8
					d4 = snap9
					d5 = snap10
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
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					idx := int(d3.Imm.Int()) + 0
					if idx < 0 || idx >= len(args) {
						panic("jitgen: dynamic args index out of range")
					}
					d12 = args[idx]
					d12.ID = 0
				} else {
					ctx.EnsureDesc(&d3)
					dynamicArgOff13 = ctx.AllocStack(16)
					ctx.ProtectReg(d3.Reg)
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, int32(len(args)-0))
					ctx.EmitJump(CondUnsignedAboveOrEqual, lbl9)
					for i := 0; i < len(args); i++ {
						nextLbl := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d3.Reg, int32(i-0))
						ctx.EmitJump(CondNotEqual, nextLbl)
						ai := args[i]
						ai.ID = 0
						ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff13))
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(nextLbl)
					}
					ctx.MarkLabel(lbl9)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff13))
					ctx.MarkLabel(lbl8)
					ctx.UnprotectReg(d3.Reg)
					d12 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff13), Rooted: true}
				}
				dynamicArgOff14 = ctx.AllocStack(16)
				ctx.EmitStoreScmerToStack(d12, int32(dynamicArgOff14))
				ctx.FreeDesc(&d12)
				d12 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff14), Rooted: true}
				ctx.StabilizeDescForControlFlow(&d12)
				d16 = d12
				d16.ID = 0
				d15 = ctx.EmitTagEqualsBorrowed(&d16, tagInt, JITValueDesc{Loc: LocAny})
				d17 = d15
				ctx.EnsureDesc(&d17)
				if d17.Loc != LocImm && d17.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d17.Loc == LocImm {
					if d17.Imm.Bool() {
						ctx.SyncDesc(&d3)
						if d3.Loc == LocReg || d3.Loc == LocFPReg {
							ctx.ProtectReg(d3.Reg)
						} else if d3.Loc == LocRegPair {
							ctx.ProtectReg(d3.Reg)
							ctx.ProtectReg(d3.Reg2)
						}
						d18 = d3
						if d18.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d18)
						ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
						if d3.Loc == LocReg || d3.Loc == LocFPReg {
							ctx.UnprotectReg(d3.Reg)
						} else if d3.Loc == LocRegPair {
							ctx.UnprotectReg(d3.Reg)
							ctx.UnprotectReg(d3.Reg2)
						}
						return bbs[1].Render()
					}
					return bbs[5].Render()
				}
				lbl10 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d17.Reg, 0)
				ctx.EmitJump(CondNotEqual, lbl10)
				ctx.EmitJmp(lbl6)
				snap19 := d1
				snap20 := d2
				snap21 := d3
				snap22 := d4
				snap23 := d5
				snap24 := d12
				snap25 := d15
				snap26 := d16
				snap27 := d17
				snap28 := d18
				alloc29 := ctx.SnapshotAllocState()
				ctx.MarkLabel(lbl10)
				ctx.SyncDesc(&d3)
				if d3.Loc == LocReg || d3.Loc == LocFPReg {
					ctx.ProtectReg(d3.Reg)
				} else if d3.Loc == LocRegPair {
					ctx.ProtectReg(d3.Reg)
					ctx.ProtectReg(d3.Reg2)
				}
				d30 = d3
				if d30.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d30)
				ctx.EmitStoreToStack(d30, int32(bbs[1].PhiBase)+int32(0))
				if d3.Loc == LocReg || d3.Loc == LocFPReg {
					ctx.UnprotectReg(d3.Reg)
				} else if d3.Loc == LocRegPair {
					ctx.UnprotectReg(d3.Reg)
					ctx.UnprotectReg(d3.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ctx.RestoreAllocState(alloc29)
				d1 = snap19
				d2 = snap20
				d3 = snap21
				d4 = snap22
				d5 = snap23
				d12 = snap24
				d15 = snap25
				d16 = snap26
				d17 = snap27
				d18 = snap28
				if !bbs[1].Rendered {
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d4
					snap35 := d5
					snap36 := d12
					snap37 := d15
					snap38 := d16
					snap39 := d17
					snap40 := d18
					snap41 := d30
					alloc42 := ctx.SnapshotAllocState()
					bbs[1].Render()
					ctx.RestoreAllocState(alloc42)
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d4 = snap34
					d5 = snap35
					d12 = snap36
					d15 = snap37
					d16 = snap38
					d17 = snap39
					d18 = snap40
					d30 = snap41
				}
				if !bbs[5].Rendered {
					return bbs[5].Render()
				}
				return result
				ctx.FreeDesc(&d15)
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
				ctx.ReclaimUntrackedRegs()
				d43 = args[0]
				d43.ID = 0
				if d43.Loc != LocImm {
					ctx.EnsureDesc(&d43)
				}
				if d43.Loc == LocImm {
					d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d43.Imm.Int())}
				} else if d43.Type == tagInt && d43.Loc == LocRegPair {
					ctx.FreeReg(d43.Reg)
					d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d43.Reg2}
					ctx.BindReg(d43.Reg2, &d44)
					ctx.BindReg(d43.Reg2, &d44)
				} else if d43.Type == tagInt && d43.Loc == LocReg {
					d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d43.Reg}
					ctx.BindReg(d43.Reg, &d44)
					ctx.BindReg(d43.Reg, &d44)
				} else {
					boxed := JITPrepareScmerGoArg(ctx, d43)
					d44 = ctx.EmitGoCallScalar(jitGoABI_4fa10ac4a0b71afc, []JITValueDesc{boxed}, 1)
					d44.Type = tagInt
					ctx.BindReg(d44.Reg, &d44)
				}
				ctx.FreeDesc(&d43)
				d45 = args[1]
				d45.ID = 0
				if d45.Loc != LocImm {
					ctx.EnsureDesc(&d45)
				}
				if d45.Loc == LocImm {
					d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d45.Imm.Int())}
				} else if d45.Type == tagInt && d45.Loc == LocRegPair {
					ctx.FreeReg(d45.Reg)
					d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg2}
					ctx.BindReg(d45.Reg2, &d46)
					ctx.BindReg(d45.Reg2, &d46)
				} else if d45.Type == tagInt && d45.Loc == LocReg {
					d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg}
					ctx.BindReg(d45.Reg, &d46)
					ctx.BindReg(d45.Reg, &d46)
				} else {
					boxed := JITPrepareScmerGoArg(ctx, d45)
					d46 = ctx.EmitGoCallScalar(jitGoABI_4fa10ac4a0b71afc, []JITValueDesc{boxed}, 1)
					d46.Type = tagInt
					ctx.BindReg(d46.Reg, &d46)
				}
				ctx.FreeDesc(&d45)
				ctx.EnsureDesc(&d46)
				ctx.EnsureDesc(&d46)
				d48 = args[2]
				d48.ID = 0
				if d48.Loc != LocImm {
					ctx.EnsureDesc(&d48)
				}
				if d48.Loc == LocImm {
					d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d48.Imm.Int())}
				} else if d48.Type == tagInt && d48.Loc == LocRegPair {
					ctx.FreeReg(d48.Reg)
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg2}
					ctx.BindReg(d48.Reg2, &d49)
					ctx.BindReg(d48.Reg2, &d49)
				} else if d48.Type == tagInt && d48.Loc == LocReg {
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
					ctx.BindReg(d48.Reg, &d49)
					ctx.BindReg(d48.Reg, &d49)
				} else {
					boxed := JITPrepareScmerGoArg(ctx, d48)
					d49 = ctx.EmitGoCallScalar(jitGoABI_4fa10ac4a0b71afc, []JITValueDesc{boxed}, 1)
					d49.Type = tagInt
					ctx.BindReg(d49.Reg, &d49)
				}
				ctx.FreeDesc(&d48)
				ctx.EnsureDesc(&d49)
				ctx.EnsureDesc(&d49)
				if d44.Loc == LocRegPair || d44.Loc == LocStackPair || d44.Loc == LocRegTriple || d44.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d46.Loc == LocRegPair || d46.Loc == LocStackPair || d46.Loc == LocRegTriple || d46.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d49.Loc == LocRegPair || d49.Loc == LocStackPair || d49.Loc == LocRegTriple || d49.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d44)
				ctx.SyncDesc(&d46)
				ctx.SyncDesc(&d49)
				d51 = ctx.EmitGoCallScalar(GoFuncAddr(NewDateNanoWithZone), []JITValueDesc{d44, d46, d49}, 2)
				d51.NoHeapPointer = false
				ctx.BindReg(d51.Reg, &d51)
				ctx.BindReg(d51.Reg2, &d51)
				ctx.FreeDesc(&d44)
				ctx.FreeDesc(&d46)
				ctx.FreeDesc(&d49)
				ctx.SyncDesc(&d51)
				if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d51, &result)
					result.Type = d51.Type
				} else {
					switch d51.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d51)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d51)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d51)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						ctx.EmitMovPairToResult(&d51, &result)
						result.Type = d51.Type
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
				d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["date_from_parts"].Fn, args, result)
				ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d12)
				d53 = d12
				d53.ID = 0
				d52 = ctx.EmitTagEqualsBorrowed(&d53, tagFloat, JITValueDesc{Loc: LocAny})
				d54 = d52
				ctx.EnsureDesc(&d54)
				if d54.Loc != LocImm && d54.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d54.Loc == LocImm {
					if d54.Imm.Bool() {
						return bbs[6].Render()
					}
					return bbs[4].Render()
				}
				ctx.EmitCmpRegImm32(d54.Reg, 0)
				ctx.EmitJump(CondNotEqual, lbl7)
				if bbs[4].Rendered {
					ctx.EmitJmp(lbl5)
				}
				ctx.FlushRegisterMoves()
				if !bbs[4].Rendered {
					snap55 := d1
					snap56 := d2
					snap57 := d3
					snap58 := d4
					snap59 := d5
					snap60 := d12
					snap61 := d15
					snap62 := d16
					snap63 := d17
					snap64 := d18
					snap65 := d30
					snap66 := d43
					snap67 := d44
					snap68 := d45
					snap69 := d46
					snap70 := d47
					snap71 := d48
					snap72 := d49
					snap73 := d50
					snap74 := d51
					snap75 := d52
					snap76 := d53
					snap77 := d54
					alloc78 := ctx.SnapshotAllocState()
					bbs[4].Render()
					ctx.RestoreAllocState(alloc78)
					d1 = snap55
					d2 = snap56
					d3 = snap57
					d4 = snap58
					d5 = snap59
					d12 = snap60
					d15 = snap61
					d16 = snap62
					d17 = snap63
					d18 = snap64
					d30 = snap65
					d43 = snap66
					d44 = snap67
					d45 = snap68
					d46 = snap69
					d47 = snap70
					d48 = snap71
					d49 = snap72
					d50 = snap73
					d51 = snap74
					d52 = snap75
					d53 = snap76
					d54 = snap77
				}
				if !bbs[6].Rendered {
					return bbs[6].Render()
				}
				return result
				ctx.FreeDesc(&d52)
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
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d12)
				d79 = ctx.EmitFloatDesc(d12)
				d80 = ctx.EmitFloatDesc(d12)
				ctx.EnsureDesc(&d80)
				if d80.Loc == LocImm {
					d81 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d80.Imm.Float()))}
				} else {
					ctx.EnsureDesc(&d80)
					var truncSrc Reg
					if d80.Loc == LocRegPair {
						ctx.FreeReg(d80.Reg)
						truncSrc = d80.Reg2
					} else {
						truncSrc = d80.Reg
					}
					truncInt := ctx.AllocRegExcept(truncSrc)
					ctx.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
					ctx.EmitInt64ToFloatBits(truncInt)
					d81 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
					ctx.BindReg(truncInt, &d81)
					ctx.BindReg(truncInt, &d81)
				}
				ctx.FreeDesc(&d80)
				ctx.EnsureDesc(&d79)
				ctx.EnsureDesc(&d81)
				ctx.EnsureDescsTogether(&d79, &d81)
				if d79.Loc == LocImm && d81.Loc == LocImm {
					d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d79.Imm.Float() == d81.Imm.Float())}
				} else if d81.Loc == LocImm {
					r3 := ctx.AllocRegExcept(d79.Reg)
					_, yBits := d81.Imm.RawWords()
					ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
					ctx.EmitCmpFloat64Setcc(r3, d79.Reg, ctx.ScratchReg, CondEqual)
					d82 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
					ctx.BindReg(r3, &d82)
				} else if d79.Loc == LocImm {
					r4 := ctx.AllocRegExcept(d81.Reg)
					_, xBits := d79.Imm.RawWords()
					ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
					ctx.EmitCmpFloat64Setcc(r4, ctx.ScratchReg, d81.Reg, CondEqual)
					d82 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
					ctx.BindReg(r4, &d82)
				} else {
					r5 := ctx.AllocRegExcept(d79.Reg, d81.Reg)
					ctx.EmitCmpFloat64Setcc(r5, d79.Reg, d81.Reg, CondEqual)
					d82 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
					ctx.BindReg(r5, &d82)
				}
				ctx.FreeDesc(&d79)
				ctx.FreeDesc(&d81)
				d83 = d82
				ctx.EnsureDesc(&d83)
				if d83.Loc != LocImm && d83.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d83.Loc == LocImm {
					if d83.Imm.Bool() {
						ctx.SyncDesc(&d3)
						if d3.Loc == LocReg || d3.Loc == LocFPReg {
							ctx.ProtectReg(d3.Reg)
						} else if d3.Loc == LocRegPair {
							ctx.ProtectReg(d3.Reg)
							ctx.ProtectReg(d3.Reg2)
						}
						d84 = d3
						if d84.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d84)
						ctx.EmitStoreToStack(d84, int32(bbs[1].PhiBase)+int32(0))
						if d3.Loc == LocReg || d3.Loc == LocFPReg {
							ctx.UnprotectReg(d3.Reg)
						} else if d3.Loc == LocRegPair {
							ctx.UnprotectReg(d3.Reg)
							ctx.UnprotectReg(d3.Reg2)
						}
						return bbs[1].Render()
					}
					return bbs[4].Render()
				}
				lbl11 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d83.Reg, 0)
				ctx.EmitJump(CondNotEqual, lbl11)
				ctx.EmitJmp(lbl5)
				snap85 := d1
				snap86 := d2
				snap87 := d3
				snap88 := d4
				snap89 := d5
				snap90 := d12
				snap91 := d15
				snap92 := d16
				snap93 := d17
				snap94 := d18
				snap95 := d30
				snap96 := d43
				snap97 := d44
				snap98 := d45
				snap99 := d46
				snap100 := d47
				snap101 := d48
				snap102 := d49
				snap103 := d50
				snap104 := d51
				snap105 := d52
				snap106 := d53
				snap107 := d54
				snap108 := d79
				snap109 := d80
				snap110 := d81
				snap111 := d82
				snap112 := d83
				snap113 := d84
				alloc114 := ctx.SnapshotAllocState()
				ctx.MarkLabel(lbl11)
				ctx.SyncDesc(&d3)
				if d3.Loc == LocReg || d3.Loc == LocFPReg {
					ctx.ProtectReg(d3.Reg)
				} else if d3.Loc == LocRegPair {
					ctx.ProtectReg(d3.Reg)
					ctx.ProtectReg(d3.Reg2)
				}
				d115 = d3
				if d115.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d115)
				ctx.EmitStoreToStack(d115, int32(bbs[1].PhiBase)+int32(0))
				if d3.Loc == LocReg || d3.Loc == LocFPReg {
					ctx.UnprotectReg(d3.Reg)
				} else if d3.Loc == LocRegPair {
					ctx.UnprotectReg(d3.Reg)
					ctx.UnprotectReg(d3.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ctx.RestoreAllocState(alloc114)
				d1 = snap85
				d2 = snap86
				d3 = snap87
				d4 = snap88
				d5 = snap89
				d12 = snap90
				d15 = snap91
				d16 = snap92
				d17 = snap93
				d18 = snap94
				d30 = snap95
				d43 = snap96
				d44 = snap97
				d45 = snap98
				d46 = snap99
				d47 = snap100
				d48 = snap101
				d49 = snap102
				d50 = snap103
				d51 = snap104
				d52 = snap105
				d53 = snap106
				d54 = snap107
				d79 = snap108
				d80 = snap109
				d81 = snap110
				d82 = snap111
				d83 = snap112
				d84 = snap113
				if !bbs[1].Rendered {
					snap116 := d1
					snap117 := d2
					snap118 := d3
					snap119 := d4
					snap120 := d5
					snap121 := d12
					snap122 := d15
					snap123 := d16
					snap124 := d17
					snap125 := d18
					snap126 := d30
					snap127 := d43
					snap128 := d44
					snap129 := d45
					snap130 := d46
					snap131 := d47
					snap132 := d48
					snap133 := d49
					snap134 := d50
					snap135 := d51
					snap136 := d52
					snap137 := d53
					snap138 := d54
					snap139 := d79
					snap140 := d80
					snap141 := d81
					snap142 := d82
					snap143 := d83
					snap144 := d84
					snap145 := d115
					alloc146 := ctx.SnapshotAllocState()
					bbs[1].Render()
					ctx.RestoreAllocState(alloc146)
					d1 = snap116
					d2 = snap117
					d3 = snap118
					d4 = snap119
					d5 = snap120
					d12 = snap121
					d15 = snap122
					d16 = snap123
					d17 = snap124
					d18 = snap125
					d30 = snap126
					d43 = snap127
					d44 = snap128
					d45 = snap129
					d46 = snap130
					d47 = snap131
					d48 = snap132
					d49 = snap133
					d50 = snap134
					d51 = snap135
					d52 = snap136
					d53 = snap137
					d54 = snap138
					d79 = snap139
					d80 = snap140
					d81 = snap141
					d82 = snap142
					d83 = snap143
					d84 = snap144
					d115 = snap145
				}
				if !bbs[4].Rendered {
					return bbs[4].Render()
				}
				return result
				ctx.FreeDesc(&d82)
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
	}})
	Declare(&Globalenv, &Declaration{Name: "date_parts", Fn: func(args ...Scmer) Scmer {
		if args[0].IsNil() {
			return NewNil()
		}
		value, ok := toTime(args[0])
		if !ok {
			panic("invalid date value")
		}
		zone := 0
		if args[0].IsDate() {
			zone = args[0].DateZone()
		}
		return NewSlice([]Scmer{NewInt(value.Unix()), NewInt(int64(value.Nanosecond())), NewInt(int64(zone))})
	}, Type: &TypeDescriptor{
		Kind: "func", Const: true,
		Params:        []*TypeDescriptor{{Kind: "any"}},
		Return:        &TypeDescriptor{Kind: "list"},
		JITInlineCost: 40,
		JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			declaration := declarations["date_parts"]
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
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d17 JITValueDesc
			_ = d17
			var d29 JITValueDesc
			_ = d29
			var d30 JITValueDesc
			_ = d30
			var d31 JITValueDesc
			_ = d31
			var d60 JITValueDesc
			_ = d60
			var d61 JITValueDesc
			_ = d61
			var d62 JITValueDesc
			_ = d62
			var stackArray63 int32
			var d64 JITValueDesc
			_ = d64
			var d65 JITValueDesc
			_ = d65
			var d66 JITValueDesc
			_ = d66
			var d67 JITValueDesc
			_ = d67
			var d68 JITValueDesc
			_ = d68
			var d69 JITValueDesc
			_ = d69
			var d71 JITValueDesc
			_ = d71
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
			var bbs [7]BBDescriptor
			bbs[6].PhiBase = int32(phiBase0) + int32(0)
			for i := range args {
				ctx.StabilizeDescForControlFlow(&args[i])
			}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				d13 = args[0]
				d13.ID = 0
				d13 = JITPrepareScmerGoArg(ctx, d13)
				ctx.SyncDesc(&d13)
				callResults14 := JITEmitGoCallResults(ctx, jitGoABI_1f0b1572417b2f13, []JITValueDesc{d13})
				d15 = callResults14[0]
				_ = d15
				d16 = callResults14[1]
				_ = d16
				ctx.FreeDesc(&d13)
				ctx.StabilizeDescForControlFlow(&d15)
				d17 = d16
				ctx.EnsureDesc(&d17)
				if d17.Loc != LocImm && d17.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d17.Loc == LocImm {
					if d17.Imm.Bool() {
						return bbs[4].Render()
					}
					return bbs[3].Render()
				}
				ctx.EmitCmpRegImm32(d17.Reg, 0)
				ctx.EmitJump(CondNotEqual, lbl5)
				if bbs[3].Rendered {
					ctx.EmitJmp(lbl4)
				}
				ctx.FlushRegisterMoves()
				if !bbs[3].Rendered {
					snap18 := d1
					snap19 := d2
					snap20 := d3
					snap21 := d4
					snap22 := d5
					snap23 := d12
					snap24 := d13
					snap25 := d15
					snap26 := d16
					snap27 := d17
					alloc28 := ctx.SnapshotAllocState()
					bbs[3].Render()
					ctx.RestoreAllocState(alloc28)
					d1 = snap18
					d2 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					d12 = snap23
					d13 = snap24
					d15 = snap25
					d16 = snap26
					d17 = snap27
				}
				if !bbs[4].Rendered {
					return bbs[4].Render()
				}
				return result
				ctx.FreeDesc(&d16)
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
				ctx.ReclaimUntrackedRegs()
				_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["date_parts"].Fn, args, result)
				ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
				ctx.ReclaimUntrackedRegs()
				d29 = args[0]
				d29.ID = 0
				d29 = JITPrepareScmerGoArg(ctx, d29)
				ctx.SyncDesc(&d29)
				d30 = ctx.EmitGoCallScalar(jitGoABI_c63e10b67a3811e2, []JITValueDesc{d29}, 1)
				d30.NoHeapPointer = true
				ctx.EmitAndRegImm32(d30.Reg, 1)
				d30.Type = tagBool
				ctx.BindReg(d30.Reg, &d30)
				ctx.FreeDesc(&d29)
				d31 = d30
				ctx.EnsureDesc(&d31)
				if d31.Loc != LocImm && d31.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d31.Loc == LocImm {
					if d31.Imm.Bool() {
						return bbs[5].Render()
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
					return bbs[6].Render()
				}
				lbl8 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d31.Reg, 0)
				ctx.EmitJump(CondNotEqual, lbl6)
				ctx.EmitJmp(lbl8)
				snap32 := d1
				snap33 := d2
				snap34 := d3
				snap35 := d4
				snap36 := d5
				snap37 := d12
				snap38 := d13
				snap39 := d15
				snap40 := d16
				snap41 := d17
				snap42 := d29
				snap43 := d30
				snap44 := d31
				alloc45 := ctx.SnapshotAllocState()
				ctx.MarkLabel(lbl8)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
				ctx.EmitJmp(lbl7)
				ctx.RestoreAllocState(alloc45)
				d1 = snap32
				d2 = snap33
				d3 = snap34
				d4 = snap35
				d5 = snap36
				d12 = snap37
				d13 = snap38
				d15 = snap39
				d16 = snap40
				d17 = snap41
				d29 = snap42
				d30 = snap43
				d31 = snap44
				if !bbs[6].Rendered {
					snap46 := d1
					snap47 := d2
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d12
					snap52 := d13
					snap53 := d15
					snap54 := d16
					snap55 := d17
					snap56 := d29
					snap57 := d30
					snap58 := d31
					alloc59 := ctx.SnapshotAllocState()
					bbs[6].Render()
					ctx.RestoreAllocState(alloc59)
					d1 = snap46
					d2 = snap47
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d12 = snap51
					d13 = snap52
					d15 = snap53
					d16 = snap54
					d17 = snap55
					d29 = snap56
					d30 = snap57
					d31 = snap58
				}
				if !bbs[5].Rendered {
					return bbs[5].Render()
				}
				return result
				ctx.FreeDesc(&d30)
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
				ctx.ReclaimUntrackedRegs()
				d60 = args[0]
				d60.ID = 0
				d60 = JITPrepareScmerGoArg(ctx, d60)
				ctx.SyncDesc(&d60)
				d61 = ctx.EmitGoCallScalar(jitGoABI_ebf492feff5dfe92, []JITValueDesc{d60}, 1)
				d61.NoHeapPointer = true
				ctx.BindReg(d61.Reg, &d61)
				ctx.StabilizeDescForControlFlow(&d61)
				ctx.FreeDesc(&d60)
				ctx.SyncDesc(&d61)
				if d61.Loc == LocReg || d61.Loc == LocFPReg {
					ctx.ProtectReg(d61.Reg)
				} else if d61.Loc == LocRegPair {
					ctx.ProtectReg(d61.Reg)
					ctx.ProtectReg(d61.Reg2)
				}
				d62 = d61
				if d62.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d62)
				ctx.EmitStoreToStack(d62, int32(bbs[6].PhiBase)+int32(0))
				if d61.Loc == LocReg || d61.Loc == LocFPReg {
					ctx.UnprotectReg(d61.Reg)
				} else if d61.Loc == LocRegPair {
					ctx.UnprotectReg(d61.Reg)
					ctx.UnprotectReg(d61.Reg2)
				}
				return bbs[6].Render()
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
				ctx.ReclaimUntrackedRegs()
				stackArray63 = ctx.AllocStack(int32(48))
				_ = stackArray63
				d15 = JITPrepareGoSliceArg(ctx, d15)
				if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d15)
				d64 = ctx.EmitGoCallScalar(jitGoABI_350a7a69d55579f7, []JITValueDesc{d15}, 1)
				d64.NoHeapPointer = true
				ctx.BindReg(d64.Reg, &d64)
				ctx.EnsureDesc(&d64)
				ctx.SyncDesc(&d64)
				ctx.EnsureDesc(&d64)
				ctx.EmitStoreTypedScmerToStack(d64, tagInt, int32(stackArray63)+int32(0))
				d15 = JITPrepareGoSliceArg(ctx, d15)
				if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Nanosecond arg0)")
				}
				ctx.SyncDesc(&d15)
				d65 = ctx.EmitGoCallScalar(jitGoABI_c5b6c72358ad4b93, []JITValueDesc{d15}, 1)
				d65.NoHeapPointer = true
				ctx.BindReg(d65.Reg, &d65)
				ctx.EnsureDesc(&d65)
				ctx.EnsureDesc(&d65)
				ctx.EnsureDesc(&d65)
				ctx.SyncDesc(&d65)
				ctx.EnsureDesc(&d65)
				ctx.EmitStoreTypedScmerToStack(d65, tagInt, int32(stackArray63)+int32(16))
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.SyncDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EmitStoreTypedScmerToStack(d1, tagInt, int32(stackArray63)+int32(32))
				d68 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
				_ = d68
				r0 := ctx.AllocReg()
				r1 := ctx.AllocRegExcept(r0)
				r2 := ctx.AllocRegExcept(r0, r1)
				d69 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r0, Reg2: r1, Reg3: r2}
				ctx.BindReg(r0, &d69)
				ctx.BindReg(r1, &d69)
				ctx.BindReg(r2, &d69)
				ctx.BindReg(r0, &d69)
				ctx.BindReg(r1, &d69)
				ctx.BindReg(r2, &d69)
				ctx.EmitLeaRegMem(d69.Reg, ctx.StackReg, int32(stackArray63))
				ctx.EmitMovRegImm64(d69.Reg2, uint64(3))
				ctx.EmitMovRegImm64(d69.Reg3, uint64(3))
				callResults70 := JITEmitGoCallResults(ctx, jitGoABI_e904302330f969f9, []JITValueDesc{d69})
				d71 = callResults70[0]
				ctx.SyncDesc(&d71)
				if d71.Loc == LocRegPair || d71.Loc == LocStackPair || d71.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d71, &result)
					result.Type = d71.Type
				} else {
					switch d71.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d71)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d71)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d71)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						ctx.EmitMovPairToResult(&d71, &result)
						result.Type = d71.Type
					}
				}
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
	}})
}

// BEGIN jitgen prepared Go ABI entries
var jitGoABI_1f0b1572417b2f13 uint64
var jitGoABI_350a7a69d55579f7 uint64
var jitGoABI_4fa10ac4a0b71afc uint64
var jitGoABI_c5b6c72358ad4b93 uint64
var jitGoABI_c63e10b67a3811e2 uint64
var jitGoABI_e904302330f969f9 uint64
var jitGoABI_ebf492feff5dfe92 uint64

func init() {
	jitGoABI_1f0b1572417b2f13 = GoFuncAddr(toTime)
	jitGoABI_350a7a69d55579f7 = GoFuncAddr((time.Time).Unix)
	jitGoABI_4fa10ac4a0b71afc = GoFuncAddr(Scmer.Int)
	jitGoABI_c5b6c72358ad4b93 = GoFuncAddr((time.Time).Nanosecond)
	jitGoABI_c63e10b67a3811e2 = GoFuncAddr((Scmer).IsDate)
	jitGoABI_e904302330f969f9 = GoFuncAddr(JITNewSliceCopy)
	jitGoABI_ebf492feff5dfe92 = GoFuncAddr((Scmer).DateZone)
}

// HasNativeDate selects the existing exact storage frame before native date
// identity, zone or fractional precision would be reduced to legacy JSON text.
func HasNativeDate(value Scmer) bool {
	value = value.WithoutSourceInfo()
	if value.IsDate() {
		return true
	}
	if value.IsProc() {
		return HasNativeDate(value.Proc().Params) || HasNativeDate(value.Proc().Body)
	}
	if value.IsJIT() {
		return HasNativeDate(value.JIT().Proc.Params) || HasNativeDate(value.JIT().Proc.Body)
	}
	if value.IsSlice() {
		for _, item := range value.Slice() {
			if HasNativeDate(item) {
				return true
			}
		}
	}
	if value.IsFastDict() && value.FastDict() != nil {
		for _, item := range value.FastDict().Pairs {
			if HasNativeDate(item) {
				return true
			}
		}
	}
	return false
}
