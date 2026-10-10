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

import (
	"strings"
	"time"
)

var allowedDateFormats = []string{
	"2006-01-02 15:04:05.000000",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"06-01-02 15:04:05.000000",
	"06-01-02 15:04:05",
	"06-01-02 15:04",
	"06-01-02",
}

// mysqlZeroDateUnix is outside Go's practical SQL date range and fits exactly
// into Scmer's signed 45-bit date payload. It preserves MySQL's zero date
// without conflating it with the Unix epoch.
const mysqlZeroDateUnix int64 = -(1 << 44)

func isMySQLZeroDate(s string) bool {
	if s == "0000-00-00" || s == "0000-00-00 00:00:00" {
		return true
	}
	if !strings.HasPrefix(s, "0000-00-00 00:00:00.") {
		return false
	}
	fraction := strings.TrimPrefix(s, "0000-00-00 00:00:00.")
	return fraction != "" && strings.Trim(fraction, "0") == ""
}

// ParseDateString tries to parse a date/datetime string using the allowed formats.
// Returns the Unix timestamp and true on success, or 0 and false on failure.
func ParseDateString(s string) (int64, bool) {
	if isMySQLZeroDate(s) {
		return mysqlZeroDateUnix, true
	}
	for _, format := range allowedDateFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// toTime converts a Scmer value (tagDate, int, float, or string) to time.Time.
func toTime(v Scmer) (time.Time, bool) {
	if v.IsNil() {
		return time.Time{}, false
	}
	switch v.GetTag() {
	case tagDate:
		return time.Unix(v.Int(), 0).UTC(), true
	case tagInt:
		return time.Unix(v.Int(), 0).UTC(), true
	case tagFloat:
		return time.Unix(int64(v.Float()), 0).UTC(), true
	case tagString, tagSymbol, tagCString, tagBString:
		if ts, ok := ParseDateString(v.String()); ok {
			return time.Unix(ts, 0).UTC(), true
		}
		return time.Time{}, false
	default:
		return time.Unix(v.Int(), 0).UTC(), true
	}
}

func sqlTemporalOutput(value Scmer, sqlType string, timezone Scmer) Scmer {
	if value.IsNil() {
		return NewNil()
	}
	t, ok := toTime(value)
	if !ok {
		return value
	}
	if t.Unix() == mysqlZeroDateUnix {
		switch strings.ToUpper(sqlType) {
		case "DATE":
			return NewString("0000-00-00")
		case "DATETIME", "TIMESTAMP":
			return NewString("0000-00-00 00:00:00")
		default:
			return value
		}
	}
	loc, err := ResolveLocation(timezone.String())
	if err != nil {
		loc = time.UTC
	}
	switch strings.ToUpper(sqlType) {
	case "DATE":
		return NewString(t.UTC().Format("2006-01-02"))
	case "DATETIME", "TIMESTAMP":
		if value.GetTag() == tagDate {
			return NewString(DateToDisplay(value, loc))
		}
		return NewString(t.In(loc).Format("2006-01-02 15:04:05"))
	default:
		return value
	}
}

func init_date() {
	// string functions
	DeclareTitle("Date")

	Declare(&Globalenv, &Declaration{
		Name: "sql_temporal_output",

		Fn: func(a ...Scmer) Scmer {
			return sqlTemporalOutput(a[0], a[1].String(), a[2])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "formats a temporal SQL result according to its compiler-tracked declared type",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "type-flexible temporal value"},
				{Kind: "string", Label: "sql_type", Description: "declared SQL temporal type"},
				{Kind: "string", Label: "timezone", Description: "explicit session timezone"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sql_temporal_output"]
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
				d3 := d1
				ctx.SyncDesc(&d3)
				if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d3 = tmpScalar
				}
				d3 = JITPrepareScmerGoArg(ctx, d3)
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair && d3.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
				d4 := args[2]
				d4.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d2.Imm)
					ptrWord, _ := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d2.Imm.String())))
					d2 = tmpPair
				} else if d2.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d2)
					d2 = tmpPair
				}
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (sqlTemporalOutput arg1)")
				}
				d4 = JITPrepareScmerGoArg(ctx, d4)
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d2)
				ctx.SyncDesc(&d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(sqlTemporalOutput), []JITValueDesc{d0, d2, d4}, 2)
				d5.NoHeapPointer = false
				ctx.BindReg(d5.Reg, &d5)
				ctx.BindReg(d5.Reg2, &d5)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d4)
				if d5.Loc == LocImm {
					if result.Loc == LocAny {
						return d5
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d5)
				if d5.Loc == LocRegPair || d5.Loc == LocStackPair || d5.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d5, &result)
					result.Type = d5.Type
				} else {
					switch d5.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d5)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d5)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d5)
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
			JITInlineCost: 9,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "now",

		Fn: func(a ...Scmer) Scmer {
			return NewDate(time.Now().Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current date/time",
			Return: &TypeDescriptor{Kind: "date"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["now"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(time.Now), []JITValueDesc{}, 3)
				d0.NoHeapPointer = false
				ctx.BindReg(d0.Reg, &d0)
				ctx.BindReg(d0.Reg2, &d0)
				ctx.BindReg(d0.Reg3, &d0)
				d0 = JITPrepareGoSliceArg(ctx, d0)
				if d0.Loc != LocRegTriple && d0.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d0}, 1)
				d1.NoHeapPointer = true
				ctx.BindReg(d1.Reg, &d1)
				ctx.FreeDesc(&d0)
				if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d1}, 2)
				d2.NoHeapPointer = false
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
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
			JITVirtualArgs: true,
			JITInlineCost:  4,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nanotime",

		Fn: func(a ...Scmer) Scmer {
			return NewInt(time.Now().UnixNano())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a monotonic nanosecond timestamp for benchmarking (not wall-clock)",
			Return: &TypeDescriptor{Kind: "int"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["nanotime"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(time.Now), []JITValueDesc{}, 3)
				d0.NoHeapPointer = false
				ctx.BindReg(d0.Reg, &d0)
				ctx.BindReg(d0.Reg2, &d0)
				ctx.BindReg(d0.Reg3, &d0)
				d0 = JITPrepareGoSliceArg(ctx, d0)
				if d0.Loc != LocRegTriple && d0.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).UnixNano arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UnixNano), []JITValueDesc{d0}, 1)
				d1.NoHeapPointer = true
				ctx.BindReg(d1.Reg, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d1.Loc == LocImm {
					ctx.EmitMakeInt(result, d1)
				} else {
					ctx.EmitMakeInt(result, d1)
					ctx.FreeReg(d1.Reg)
				}
				result.Type = tagInt
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  4,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "current_date",

		Fn: func(a ...Scmer) Scmer {
			timezone := "UTC"
			if len(a) > 0 {
				timezone = a[0].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			now := time.Now().In(loc)
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current date (midnight in session timezone)",
			Params: []*TypeDescriptor{{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "date"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["current_date"]
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d43 JITValueDesc
				_ = d43
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
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
				var bbs [5]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() > 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d3.Reg, 0)
						d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d4)
					}
					ctx.FreeDesc(&d3)
					d5 = d4
					ctx.EnsureDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							return bbs[1].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl6 := ctx.ReserveLabel()
					ctx.EmitJump(d5.Condition, lbl2)
					ctx.EmitJmp(lbl6)
					ctx.FreeDesc(&d4)
					snap6 := d1
					snap7 := d2
					snap8 := d3
					snap9 := d4
					snap10 := d5
					alloc11 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl6)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc11)
					d1 = snap6
					d2 = snap7
					d3 = snap8
					d4 = snap9
					d5 = snap10
					if !bbs[2].Rendered {
						snap12 := d1
						snap13 := d2
						snap14 := d3
						snap15 := d4
						snap16 := d5
						alloc17 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc17)
						d1 = snap12
						d2 = snap13
						d3 = snap14
						d4 = snap15
						d5 = snap16
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d18 = args[0]
					d18.ID = 0
					d20 = d18
					ctx.SyncDesc(&d20)
					if d20.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d20.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d20.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d20 = tmpScalar
					}
					d20 = JITPrepareScmerGoArg(ctx, d20)
					if d20.Loc != LocRegPair && d20.Loc != LocStackPair && d20.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d20}, 2)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					ctx.SyncDesc(&d19)
					if d19.Loc == LocReg || d19.Loc == LocFPReg {
						ctx.ProtectReg(d19.Reg)
					} else if d19.Loc == LocRegPair {
						ctx.ProtectReg(d19.Reg)
						ctx.ProtectReg(d19.Reg2)
					}
					d21 = d19
					if d21.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d21)
					if d21.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d21, int32(bbs[2].PhiBase)+int32(0), 2)
					} else if d21.Loc == LocInputPair {
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
					} else if d21.Loc == LocRegPair || d21.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreToStack(d21, int32(bbs[2].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
					}
					if d19.Loc == LocReg || d19.Loc == LocFPReg {
						ctx.UnprotectReg(d19.Reg)
					} else if d19.Loc == LocRegPair {
						ctx.UnprotectReg(d19.Reg)
						ctx.UnprotectReg(d19.Reg2)
					}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d1.Imm.String())))
						d1 = tmpPair
					} else if d1.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d1)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d1)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d1)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d1)
						d1 = tmpPair
					}
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d1)
					callResults22 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d1}, []uint8{1, 2}, []uint8{1, 3})
					d23 = callResults22[0]
					_ = d23
					d24 = callResults22[1]
					_ = d24
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d24.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d24)
						if d24.Loc != LocReg && d24.Loc != LocRegPair && d24.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitCmpRegImm32(d24.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d25)
					}
					ctx.FreeDesc(&d24)
					d26 = d25
					ctx.EnsureDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d27 = d23
						if d27.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d27)
						ctx.EmitStoreToStack(d27, int32(bbs[4].PhiBase)+int32(0))
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
						return bbs[4].Render()
					}
					lbl7 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl7)
					snap28 := d1
					snap29 := d2
					snap30 := d3
					snap31 := d4
					snap32 := d5
					snap33 := d18
					snap34 := d19
					snap35 := d20
					snap36 := d21
					snap37 := d23
					snap38 := d24
					snap39 := d25
					snap40 := d26
					snap41 := d27
					alloc42 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.SyncDesc(&d23)
					if d23.Loc == LocReg || d23.Loc == LocFPReg {
						ctx.ProtectReg(d23.Reg)
					} else if d23.Loc == LocRegPair {
						ctx.ProtectReg(d23.Reg)
						ctx.ProtectReg(d23.Reg2)
					}
					d43 = d23
					if d43.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d43)
					ctx.EmitStoreToStack(d43, int32(bbs[4].PhiBase)+int32(0))
					if d23.Loc == LocReg || d23.Loc == LocFPReg {
						ctx.UnprotectReg(d23.Reg)
					} else if d23.Loc == LocRegPair {
						ctx.UnprotectReg(d23.Reg)
						ctx.UnprotectReg(d23.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc42)
					d1 = snap28
					d2 = snap29
					d3 = snap30
					d4 = snap31
					d5 = snap32
					d18 = snap33
					d19 = snap34
					d20 = snap35
					d21 = snap36
					d23 = snap37
					d24 = snap38
					d25 = snap39
					d26 = snap40
					d27 = snap41
					if !bbs[4].Rendered {
						snap44 := d1
						snap45 := d2
						snap46 := d3
						snap47 := d4
						snap48 := d5
						snap49 := d18
						snap50 := d19
						snap51 := d20
						snap52 := d21
						snap53 := d23
						snap54 := d24
						snap55 := d25
						snap56 := d26
						snap57 := d27
						snap58 := d43
						alloc59 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc59)
						d1 = snap44
						d2 = snap45
						d3 = snap46
						d4 = snap47
						d5 = snap48
						d18 = snap49
						d19 = snap50
						d20 = snap51
						d21 = snap52
						d23 = snap53
						d24 = snap54
						d25 = snap55
						d26 = snap56
						d27 = snap57
						d43 = snap58
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d25)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d60 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.StabilizeDescForControlFlow(&d60)
					ctx.SyncDesc(&d60)
					if d60.Loc == LocReg || d60.Loc == LocFPReg {
						ctx.ProtectReg(d60.Reg)
					} else if d60.Loc == LocRegPair {
						ctx.ProtectReg(d60.Reg)
						ctx.ProtectReg(d60.Reg2)
					}
					d61 = d60
					if d61.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreToStack(d61, int32(bbs[4].PhiBase)+int32(0))
					if d60.Loc == LocReg || d60.Loc == LocFPReg {
						ctx.UnprotectReg(d60.Reg)
					} else if d60.Loc == LocRegPair {
						ctx.UnprotectReg(d60.Reg)
						ctx.UnprotectReg(d60.Reg2)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d62 = ctx.EmitGoCallScalar(GoFuncAddr(time.Now), []JITValueDesc{}, 3)
					d62.NoHeapPointer = false
					ctx.BindReg(d62.Reg, &d62)
					ctx.BindReg(d62.Reg2, &d62)
					ctx.BindReg(d62.Reg3, &d62)
					d62 = JITPrepareGoSliceArg(ctx, d62)
					if d62.Loc != LocRegTriple && d62.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d62)
					ctx.SyncDesc(&d2)
					d63 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d62, d2}, 3)
					d63.NoHeapPointer = false
					ctx.BindReg(d63.Reg, &d63)
					ctx.BindReg(d63.Reg2, &d63)
					ctx.BindReg(d63.Reg3, &d63)
					ctx.FreeDesc(&d62)
					ctx.FreeDesc(&d2)
					d63 = JITPrepareGoSliceArg(ctx, d63)
					if d63.Loc != LocRegTriple && d63.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d63)
					d64 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d63}, 1)
					d64.NoHeapPointer = true
					ctx.BindReg(d64.Reg, &d64)
					d63 = JITPrepareGoSliceArg(ctx, d63)
					if d63.Loc != LocRegTriple && d63.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d63)
					d65 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d63}, 1)
					d65.NoHeapPointer = true
					ctx.BindReg(d65.Reg, &d65)
					d63 = JITPrepareGoSliceArg(ctx, d63)
					if d63.Loc != LocRegTriple && d63.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d63)
					d66 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d63}, 1)
					d66.NoHeapPointer = true
					ctx.BindReg(d66.Reg, &d66)
					ctx.FreeDesc(&d63)
					d67 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d64.Loc == LocRegPair || d64.Loc == LocStackPair || d64.Loc == LocRegTriple || d64.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d65.Loc == LocRegPair || d65.Loc == LocStackPair || d65.Loc == LocRegTriple || d65.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d66.Loc == LocRegPair || d66.Loc == LocStackPair || d66.Loc == LocRegTriple || d66.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d68.Loc == LocRegPair || d68.Loc == LocStackPair || d68.Loc == LocRegTriple || d68.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d69.Loc == LocRegPair || d69.Loc == LocStackPair || d69.Loc == LocRegTriple || d69.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d71.Loc == LocRegPair || d71.Loc == LocStackPair || d71.Loc == LocRegTriple || d71.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d67.Loc == LocRegPair || d67.Loc == LocStackPair || d67.Loc == LocRegTriple || d67.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d64)
					ctx.SyncDesc(&d65)
					ctx.SyncDesc(&d66)
					ctx.SyncDesc(&d68)
					ctx.SyncDesc(&d69)
					ctx.SyncDesc(&d70)
					ctx.SyncDesc(&d71)
					ctx.SyncDesc(&d67)
					d72 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d64, d65, d66, d68, d69, d70, d71, d67}, 3)
					d72.NoHeapPointer = false
					ctx.BindReg(d72.Reg, &d72)
					ctx.BindReg(d72.Reg2, &d72)
					ctx.BindReg(d72.Reg3, &d72)
					ctx.FreeDesc(&d68)
					ctx.FreeDesc(&d69)
					ctx.FreeDesc(&d70)
					ctx.FreeDesc(&d71)
					ctx.FreeDesc(&d64)
					ctx.FreeDesc(&d65)
					ctx.FreeDesc(&d66)
					ctx.FreeDesc(&d67)
					d72 = JITPrepareGoSliceArg(ctx, d72)
					if d72.Loc != LocRegTriple && d72.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d72)
					d73 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d72}, 1)
					d73.NoHeapPointer = true
					ctx.BindReg(d73.Reg, &d73)
					ctx.FreeDesc(&d72)
					if d73.Loc == LocRegPair || d73.Loc == LocStackPair || d73.Loc == LocRegTriple || d73.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d73)
					d74 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d73}, 2)
					d74.NoHeapPointer = false
					ctx.BindReg(d74.Reg, &d74)
					ctx.BindReg(d74.Reg2, &d74)
					ctx.FreeDesc(&d73)
					ctx.SyncDesc(&d74)
					if d74.Loc == LocRegPair || d74.Loc == LocStackPair || d74.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d74, &result)
						result.Type = d74.Type
					} else {
						switch d74.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d74)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d74)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d74)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d74, &result)
							result.Type = d74.Type
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
			JITVirtualArgs: true,
			JITInlineCost:  26,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parse_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			if a[0].GetTag() == tagDate {
				return a[0]
			}
			if a[0].IsInt() || a[0].IsFloat() {
				return NewDate(a[0].Int())
			}
			if ts, ok := ParseDateString(a[0].String()); ok {
				return NewDate(ts)
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses a date from a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "values to parse"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["parse_date"]
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
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
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
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
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
				var bbs [10]BBDescriptor
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
					d11 = d10
					d11.ID = 0
					d12 = ctx.EmitGetTagDesc(&d11, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0x10))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d12.Reg, 16)
						d13 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d13)
					}
					ctx.FreeDesc(&d12)
					d14 = d13
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d14.Loc == LocImm {
						if d14.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d14.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d13)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap15 := d0
						snap16 := d1
						snap17 := d2
						snap18 := d3
						snap19 := d9
						snap20 := d10
						snap21 := d11
						snap22 := d12
						snap23 := d13
						snap24 := d14
						alloc25 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc25)
						d0 = snap15
						d1 = snap16
						d2 = snap17
						d3 = snap18
						d9 = snap19
						d10 = snap20
						d11 = snap21
						d12 = snap22
						d13 = snap23
						d14 = snap24
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
					d26 = args[0]
					d26.ID = 0
					ctx.SyncDesc(&d26)
					if d26.Loc == LocRegPair || d26.Loc == LocStackPair || d26.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d26, &result)
						result.Type = d26.Type
					} else {
						switch d26.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d26)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d26)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d26)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d26, &result)
							result.Type = d26.Type
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
					d27 = args[0]
					d27.ID = 0
					d29 = d27
					d29.ID = 0
					d28 = ctx.EmitTagEqualsBorrowed(&d29, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d27)
					d30 = d28
					ctx.EnsureDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap31 := d0
						snap32 := d1
						snap33 := d2
						snap34 := d3
						snap35 := d9
						snap36 := d10
						snap37 := d11
						snap38 := d12
						snap39 := d13
						snap40 := d14
						snap41 := d26
						snap42 := d27
						snap43 := d28
						snap44 := d29
						snap45 := d30
						alloc46 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc46)
						d0 = snap31
						d1 = snap32
						d2 = snap33
						d3 = snap34
						d9 = snap35
						d10 = snap36
						d11 = snap37
						d12 = snap38
						d13 = snap39
						d14 = snap40
						d26 = snap41
						d27 = snap42
						d28 = snap43
						d29 = snap44
						d30 = snap45
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d28)
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
					d47 = args[0]
					d47.ID = 0
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
					ctx.FreeDesc(&d47)
					if d48.Loc == LocRegPair || d48.Loc == LocStackPair || d48.Loc == LocRegTriple || d48.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d48)
					d49 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d48}, 2)
					d49.NoHeapPointer = false
					ctx.BindReg(d49.Reg, &d49)
					ctx.BindReg(d49.Reg2, &d49)
					ctx.FreeDesc(&d48)
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
					d50 = args[0]
					d50.ID = 0
					d52 = d50
					ctx.SyncDesc(&d52)
					if d52.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d52.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d52.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d52 = tmpScalar
					}
					d52 = JITPrepareScmerGoArg(ctx, d52)
					if d52.Loc != LocRegPair && d52.Loc != LocStackPair && d52.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d52}, 2)
					ctx.FreeDesc(&d50)
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d51.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d51.Imm)
						ptrWord, _ := d51.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d51.Imm.String())))
						d51 = tmpPair
					} else if d51.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d51.Type, Reg: ctx.AllocRegExcept(d51.Reg), Reg2: ctx.AllocRegExcept(d51.Reg)}
						switch d51.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d51)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d51)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d51)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d51)
						d51 = tmpPair
					}
					if d51.Loc != LocRegPair && d51.Loc != LocStackPair && d51.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ParseDateString arg0)")
					}
					ctx.SyncDesc(&d51)
					callResults53 := JITEmitGoCallResults(ctx, GoFuncAddr(ParseDateString), []JITValueDesc{d51}, []uint8{1, 1}, []uint8{0, 0})
					d54 = callResults53[0]
					_ = d54
					d55 = callResults53[1]
					_ = d55
					ctx.StabilizeDescForControlFlow(&d54)
					d56 = d55
					ctx.EnsureDesc(&d56)
					if d56.Loc != LocImm && d56.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d56.Loc == LocImm {
						if d56.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitCmpRegImm32(d56.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap57 := d0
						snap58 := d1
						snap59 := d2
						snap60 := d3
						snap61 := d9
						snap62 := d10
						snap63 := d11
						snap64 := d12
						snap65 := d13
						snap66 := d14
						snap67 := d26
						snap68 := d27
						snap69 := d28
						snap70 := d29
						snap71 := d30
						snap72 := d47
						snap73 := d48
						snap74 := d49
						snap75 := d50
						snap76 := d51
						snap77 := d52
						snap78 := d54
						snap79 := d55
						snap80 := d56
						alloc81 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc81)
						d0 = snap57
						d1 = snap58
						d2 = snap59
						d3 = snap60
						d9 = snap61
						d10 = snap62
						d11 = snap63
						d12 = snap64
						d13 = snap65
						d14 = snap66
						d26 = snap67
						d27 = snap68
						d28 = snap69
						d29 = snap70
						d30 = snap71
						d47 = snap72
						d48 = snap73
						d49 = snap74
						d50 = snap75
						d51 = snap76
						d52 = snap77
						d54 = snap78
						d55 = snap79
						d56 = snap80
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d55)
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
					d82 = args[0]
					d82.ID = 0
					d84 = d82
					d84.ID = 0
					d83 = ctx.EmitTagEqualsBorrowed(&d84, tagFloat, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d82)
					d85 = d83
					ctx.EnsureDesc(&d85)
					if d85.Loc != LocImm && d85.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d85.Loc == LocImm {
						if d85.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d85.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap86 := d0
						snap87 := d1
						snap88 := d2
						snap89 := d3
						snap90 := d9
						snap91 := d10
						snap92 := d11
						snap93 := d12
						snap94 := d13
						snap95 := d14
						snap96 := d26
						snap97 := d27
						snap98 := d28
						snap99 := d29
						snap100 := d30
						snap101 := d47
						snap102 := d48
						snap103 := d49
						snap104 := d50
						snap105 := d51
						snap106 := d52
						snap107 := d54
						snap108 := d55
						snap109 := d56
						snap110 := d82
						snap111 := d83
						snap112 := d84
						snap113 := d85
						alloc114 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc114)
						d0 = snap86
						d1 = snap87
						d2 = snap88
						d3 = snap89
						d9 = snap90
						d10 = snap91
						d11 = snap92
						d12 = snap93
						d13 = snap94
						d14 = snap95
						d26 = snap96
						d27 = snap97
						d28 = snap98
						d29 = snap99
						d30 = snap100
						d47 = snap101
						d48 = snap102
						d49 = snap103
						d50 = snap104
						d51 = snap105
						d52 = snap106
						d54 = snap107
						d55 = snap108
						d56 = snap109
						d82 = snap110
						d83 = snap111
						d84 = snap112
						d85 = snap113
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d83)
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
					if d54.Loc == LocRegPair || d54.Loc == LocStackPair || d54.Loc == LocRegTriple || d54.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d54)
					d115 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d54}, 2)
					d115.NoHeapPointer = false
					ctx.BindReg(d115.Reg, &d115)
					ctx.BindReg(d115.Reg2, &d115)
					ctx.SyncDesc(&d115)
					if d115.Loc == LocRegPair || d115.Loc == LocStackPair || d115.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d115, &result)
						result.Type = d115.Type
					} else {
						switch d115.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d115)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d115)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d115)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d115, &result)
							result.Type = d115.Type
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
					ctx.ReclaimUntrackedRegs()
					d116 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d116)
					if d116.Loc == LocRegPair || d116.Loc == LocStackPair || d116.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d116, &result)
						result.Type = d116.Type
					} else {
						switch d116.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d116)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d116)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d116)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d116, &result)
							result.Type = d116.Type
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
			JITVirtualArgs: true,
			JITInlineCost:  38,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "format_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			timezone := "UTC"
			if len(a) > 2 {
				timezone = a[2].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			t = t.In(loc)
			return NewString(formatDateMySQL(t, String(a[1])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "formats a unix timestamp, date, or datetime string into a date string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "timestamp", Description: "unix timestamp, date, or datetime string"}, &TypeDescriptor{Kind: "string", Label: "format", Description: "MySQL-style format string (e.g. %Y-%m-%d %H:%i:%s)"}, &TypeDescriptor{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["format_date"]
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
				var d15 JITValueDesc
				_ = d15
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
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
				var d103 JITValueDesc
				_ = d103
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
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
				var bbs [9]BBDescriptor
				bbs[6].PhiBase = int32(phiBase0) + int32(0)
				bbs[8].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					d5 = d3
					d5.ID = 0
					d4 = ctx.EmitTagEqualsBorrowed(&d5, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d3)
					d6 = d4
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						snap12 := d6
						alloc13 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc13)
						d1 = snap7
						d2 = snap8
						d3 = snap9
						d4 = snap10
						d5 = snap11
						d6 = snap12
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d4)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d14 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d14, &result)
						result.Type = d14.Type
					} else {
						switch d14.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d14)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d14)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d14)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d14, &result)
							result.Type = d14.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d15 = args[0]
					d15.ID = 0
					d15 = JITPrepareScmerGoArg(ctx, d15)
					ctx.SyncDesc(&d15)
					callResults16 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d15}, []uint8{3, 1}, []uint8{4, 0})
					d17 = callResults16[0]
					_ = d17
					d18 = callResults16[1]
					_ = d18
					ctx.FreeDesc(&d15)
					ctx.StabilizeDescForControlFlow(&d17)
					d19 = d18
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d4
						snap24 := d5
						snap25 := d6
						snap26 := d14
						snap27 := d15
						snap28 := d17
						snap29 := d18
						snap30 := d19
						alloc31 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc31)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d4 = snap23
						d5 = snap24
						d6 = snap25
						d14 = snap26
						d15 = snap27
						d17 = snap28
						d18 = snap29
						d19 = snap30
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d32 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d32)
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d32, &result)
						result.Type = d32.Type
					} else {
						switch d32.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d32)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d32)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d32)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d32, &result)
							result.Type = d32.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d33.Reg, 2)
						d34 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d34)
					}
					ctx.FreeDesc(&d33)
					d35 = d34
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							return bbs[5].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[6].PhiBase)+int32(0))
						return bbs[6].Render()
					}
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d35.Condition, lbl6)
					ctx.EmitJmp(lbl10)
					ctx.FreeDesc(&d34)
					snap36 := d1
					snap37 := d2
					snap38 := d3
					snap39 := d4
					snap40 := d5
					snap41 := d6
					snap42 := d14
					snap43 := d15
					snap44 := d17
					snap45 := d18
					snap46 := d19
					snap47 := d32
					snap48 := d33
					snap49 := d34
					snap50 := d35
					alloc51 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[6].PhiBase)+int32(0))
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc51)
					d1 = snap36
					d2 = snap37
					d3 = snap38
					d4 = snap39
					d5 = snap40
					d6 = snap41
					d14 = snap42
					d15 = snap43
					d17 = snap44
					d18 = snap45
					d19 = snap46
					d32 = snap47
					d33 = snap48
					d34 = snap49
					d35 = snap50
					if !bbs[6].Rendered {
						snap52 := d1
						snap53 := d2
						snap54 := d3
						snap55 := d4
						snap56 := d5
						snap57 := d6
						snap58 := d14
						snap59 := d15
						snap60 := d17
						snap61 := d18
						snap62 := d19
						snap63 := d32
						snap64 := d33
						snap65 := d34
						snap66 := d35
						alloc67 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc67)
						d1 = snap52
						d2 = snap53
						d3 = snap54
						d4 = snap55
						d5 = snap56
						d6 = snap57
						d14 = snap58
						d15 = snap59
						d17 = snap60
						d18 = snap61
						d19 = snap62
						d32 = snap63
						d33 = snap64
						d34 = snap65
						d35 = snap66
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d68 = args[2]
					d68.ID = 0
					d70 = d68
					ctx.SyncDesc(&d70)
					if d70.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d70.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d70.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d70 = tmpScalar
					}
					d70 = JITPrepareScmerGoArg(ctx, d70)
					if d70.Loc != LocRegPair && d70.Loc != LocStackPair && d70.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d70}, 2)
					ctx.StabilizeDescForControlFlow(&d69)
					ctx.FreeDesc(&d68)
					ctx.SyncDesc(&d69)
					if d69.Loc == LocReg || d69.Loc == LocFPReg {
						ctx.ProtectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.ProtectReg(d69.Reg)
						ctx.ProtectReg(d69.Reg2)
					}
					d71 = d69
					if d71.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d71)
					if d71.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d71, int32(bbs[6].PhiBase)+int32(0), 2)
					} else if d71.Loc == LocInputPair {
						ctx.EnsureDesc(&d71)
						ctx.EmitStoreScmerToStack(d71, int32(bbs[6].PhiBase)+int32(0))
					} else if d71.Loc == LocRegPair || d71.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d71, int32(bbs[6].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d71)
						ctx.EmitStoreToStack(d71, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
					}
					if d69.Loc == LocReg || d69.Loc == LocFPReg {
						ctx.UnprotectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.UnprotectReg(d69.Reg)
						ctx.UnprotectReg(d69.Reg2)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d1.Imm.String())))
						d1 = tmpPair
					} else if d1.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d1)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d1)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d1)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d1)
						d1 = tmpPair
					}
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d1)
					callResults72 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d1}, []uint8{1, 2}, []uint8{1, 3})
					d73 = callResults72[0]
					_ = d73
					d74 = callResults72[1]
					_ = d74
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d73)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d74.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d74)
						if d74.Loc != LocReg && d74.Loc != LocRegPair && d74.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d74.Reg)
						ctx.EmitCmpRegImm32(d74.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d75 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d75)
					}
					ctx.FreeDesc(&d74)
					d76 = d75
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							return bbs[7].Render()
						}
						ctx.SyncDesc(&d73)
						if d73.Loc == LocReg || d73.Loc == LocFPReg {
							ctx.ProtectReg(d73.Reg)
						} else if d73.Loc == LocRegPair {
							ctx.ProtectReg(d73.Reg)
							ctx.ProtectReg(d73.Reg2)
						}
						d77 = d73
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreToStack(d77, int32(bbs[8].PhiBase)+int32(0))
						if d73.Loc == LocReg || d73.Loc == LocFPReg {
							ctx.UnprotectReg(d73.Reg)
						} else if d73.Loc == LocRegPair {
							ctx.UnprotectReg(d73.Reg)
							ctx.UnprotectReg(d73.Reg2)
						}
						return bbs[8].Render()
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl11)
					snap78 := d1
					snap79 := d2
					snap80 := d3
					snap81 := d4
					snap82 := d5
					snap83 := d6
					snap84 := d14
					snap85 := d15
					snap86 := d17
					snap87 := d18
					snap88 := d19
					snap89 := d32
					snap90 := d33
					snap91 := d34
					snap92 := d35
					snap93 := d68
					snap94 := d69
					snap95 := d70
					snap96 := d71
					snap97 := d73
					snap98 := d74
					snap99 := d75
					snap100 := d76
					snap101 := d77
					alloc102 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.SyncDesc(&d73)
					if d73.Loc == LocReg || d73.Loc == LocFPReg {
						ctx.ProtectReg(d73.Reg)
					} else if d73.Loc == LocRegPair {
						ctx.ProtectReg(d73.Reg)
						ctx.ProtectReg(d73.Reg2)
					}
					d103 = d73
					if d103.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d103)
					ctx.EmitStoreToStack(d103, int32(bbs[8].PhiBase)+int32(0))
					if d73.Loc == LocReg || d73.Loc == LocFPReg {
						ctx.UnprotectReg(d73.Reg)
					} else if d73.Loc == LocRegPair {
						ctx.UnprotectReg(d73.Reg)
						ctx.UnprotectReg(d73.Reg2)
					}
					ctx.EmitJmp(lbl9)
					ctx.RestoreAllocState(alloc102)
					d1 = snap78
					d2 = snap79
					d3 = snap80
					d4 = snap81
					d5 = snap82
					d6 = snap83
					d14 = snap84
					d15 = snap85
					d17 = snap86
					d18 = snap87
					d19 = snap88
					d32 = snap89
					d33 = snap90
					d34 = snap91
					d35 = snap92
					d68 = snap93
					d69 = snap94
					d70 = snap95
					d71 = snap96
					d73 = snap97
					d74 = snap98
					d75 = snap99
					d76 = snap100
					d77 = snap101
					if !bbs[8].Rendered {
						snap104 := d1
						snap105 := d2
						snap106 := d3
						snap107 := d4
						snap108 := d5
						snap109 := d6
						snap110 := d14
						snap111 := d15
						snap112 := d17
						snap113 := d18
						snap114 := d19
						snap115 := d32
						snap116 := d33
						snap117 := d34
						snap118 := d35
						snap119 := d68
						snap120 := d69
						snap121 := d70
						snap122 := d71
						snap123 := d73
						snap124 := d74
						snap125 := d75
						snap126 := d76
						snap127 := d77
						snap128 := d103
						alloc129 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc129)
						d1 = snap104
						d2 = snap105
						d3 = snap106
						d4 = snap107
						d5 = snap108
						d6 = snap109
						d14 = snap110
						d15 = snap111
						d17 = snap112
						d18 = snap113
						d19 = snap114
						d32 = snap115
						d33 = snap116
						d34 = snap117
						d35 = snap118
						d68 = snap119
						d69 = snap120
						d70 = snap121
						d71 = snap122
						d73 = snap123
						d74 = snap124
						d75 = snap125
						d76 = snap126
						d77 = snap127
						d103 = snap128
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d75)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d130 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.StabilizeDescForControlFlow(&d130)
					ctx.SyncDesc(&d130)
					if d130.Loc == LocReg || d130.Loc == LocFPReg {
						ctx.ProtectReg(d130.Reg)
					} else if d130.Loc == LocRegPair {
						ctx.ProtectReg(d130.Reg)
						ctx.ProtectReg(d130.Reg2)
					}
					d131 = d130
					if d131.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d131)
					ctx.EmitStoreToStack(d131, int32(bbs[8].PhiBase)+int32(0))
					if d130.Loc == LocReg || d130.Loc == LocFPReg {
						ctx.UnprotectReg(d130.Reg)
					} else if d130.Loc == LocRegPair {
						ctx.UnprotectReg(d130.Reg)
						ctx.UnprotectReg(d130.Reg2)
					}
					return bbs[8].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d17 = JITPrepareGoSliceArg(ctx, d17)
					if d17.Loc != LocRegTriple && d17.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d17)
					ctx.SyncDesc(&d2)
					d132 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d17, d2}, 3)
					d132.NoHeapPointer = false
					ctx.BindReg(d132.Reg, &d132)
					ctx.BindReg(d132.Reg2, &d132)
					ctx.BindReg(d132.Reg3, &d132)
					ctx.FreeDesc(&d2)
					d133 = args[1]
					d133.ID = 0
					d135 = d133
					ctx.SyncDesc(&d135)
					if d135.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d135.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d135.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d135 = tmpScalar
					}
					d135 = JITPrepareScmerGoArg(ctx, d135)
					if d135.Loc != LocRegPair && d135.Loc != LocStackPair && d135.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d134 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d135}, 2)
					ctx.FreeDesc(&d133)
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (formatDateMySQL arg0)")
					}
					ctx.EnsureDesc(&d134)
					if d134.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d134.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d134.Imm)
						ptrWord, _ := d134.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d134.Imm.String())))
						d134 = tmpPair
					} else if d134.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d134.Type, Reg: ctx.AllocRegExcept(d134.Reg), Reg2: ctx.AllocRegExcept(d134.Reg)}
						switch d134.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d134)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d134)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d134)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d134)
						d134 = tmpPair
					}
					if d134.Loc != LocRegPair && d134.Loc != LocStackPair && d134.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (formatDateMySQL arg1)")
					}
					ctx.SyncDesc(&d132)
					ctx.SyncDesc(&d134)
					d136 = ctx.EmitGoCallScalar(GoFuncAddr(formatDateMySQL), []JITValueDesc{d132, d134}, 2)
					d136.NoHeapPointer = false
					ctx.BindReg(d136.Reg, &d136)
					ctx.BindReg(d136.Reg2, &d136)
					ctx.FreeDesc(&d132)
					ctx.EnsureDesc(&d136)
					d137 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d136}, 2)
					ctx.EmitMovPairToResult(&d137, &result)
					result.Type = tagString
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
			JITInlineCost:  37,
		},
	})

	// EXTRACT(field FROM expr) - implemented as extract_date(expr, field)
	Declare(&Globalenv, &Declaration{
		Name: "extract_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			timezone := "UTC"
			if len(a) > 2 {
				timezone = a[2].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			t = t.In(loc)
			field := strings.ToUpper(a[1].String())
			switch field {
			case "YEAR":
				return NewInt(int64(t.Year()))
			case "MONTH":
				return NewInt(int64(t.Month()))
			case "DAY":
				return NewInt(int64(t.Day()))
			case "HOUR":
				return NewInt(int64(t.Hour()))
			case "MINUTE":
				return NewInt(int64(t.Minute()))
			case "SECOND":
				return NewInt(int64(t.Second()))
			case "QUARTER":
				return NewInt(int64((int(t.Month())-1)/3 + 1))
			case "WEEK":
				_, week := t.ISOWeek()
				return NewInt(int64(week))
			case "DAYOFWEEK":
				// MySQL: 1=Sunday, 2=Monday, ..., 7=Saturday
				return NewInt(int64(t.Weekday()) + 1)
			case "WEEKDAY":
				// MySQL WEEKDAY: 0=Monday, 1=Tuesday, ..., 6=Sunday
				return NewInt(int64((t.Weekday() + 6) % 7))
			default:
				panic("unknown EXTRACT field: " + field)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "extracts a date field (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, QUARTER, WEEK, DAYOFWEEK, WEEKDAY) from a date value",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date value"}, &TypeDescriptor{Kind: "string", Label: "field", Description: "field name: YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, QUARTER, WEEK, DAYOFWEEK, WEEKDAY"}, &TypeDescriptor{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["extract_date"]
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
				var d15 JITValueDesc
				_ = d15
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
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
				var d103 JITValueDesc
				_ = d103
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				var d178 JITValueDesc
				_ = d178
				var d179 JITValueDesc
				_ = d179
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d183 JITValueDesc
				_ = d183
				var d184 JITValueDesc
				_ = d184
				var d185 JITValueDesc
				_ = d185
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d235 JITValueDesc
				_ = d235
				var d236 JITValueDesc
				_ = d236
				var d237 JITValueDesc
				_ = d237
				var d238 JITValueDesc
				_ = d238
				var d239 JITValueDesc
				_ = d239
				var d240 JITValueDesc
				_ = d240
				var d241 JITValueDesc
				_ = d241
				var d296 JITValueDesc
				_ = d296
				var d297 JITValueDesc
				_ = d297
				var d298 JITValueDesc
				_ = d298
				var d299 JITValueDesc
				_ = d299
				var d300 JITValueDesc
				_ = d300
				var d301 JITValueDesc
				_ = d301
				var d302 JITValueDesc
				_ = d302
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d368 JITValueDesc
				_ = d368
				var d369 JITValueDesc
				_ = d369
				var d370 JITValueDesc
				_ = d370
				var d439 JITValueDesc
				_ = d439
				var d440 JITValueDesc
				_ = d440
				var d441 JITValueDesc
				_ = d441
				var d442 JITValueDesc
				_ = d442
				var d443 JITValueDesc
				_ = d443
				var d444 JITValueDesc
				_ = d444
				var d445 JITValueDesc
				_ = d445
				var d521 JITValueDesc
				_ = d521
				var d522 JITValueDesc
				_ = d522
				var d523 JITValueDesc
				_ = d523
				var d524 JITValueDesc
				_ = d524
				var d525 JITValueDesc
				_ = d525
				var d526 JITValueDesc
				_ = d526
				var d527 JITValueDesc
				_ = d527
				var d528 JITValueDesc
				_ = d528
				var d529 JITValueDesc
				_ = d529
				var d530 JITValueDesc
				_ = d530
				var d617 JITValueDesc
				_ = d617
				var d618 JITValueDesc
				_ = d618
				var d619 JITValueDesc
				_ = d619
				var d620 JITValueDesc
				_ = d620
				var d621 JITValueDesc
				_ = d621
				var d622 JITValueDesc
				_ = d622
				var d623 JITValueDesc
				_ = d623
				var d624 JITValueDesc
				_ = d624
				var d718 JITValueDesc
				_ = d718
				var d719 JITValueDesc
				_ = d719
				var d721 JITValueDesc
				_ = d721
				var d722 JITValueDesc
				_ = d722
				var d723 JITValueDesc
				_ = d723
				var d724 JITValueDesc
				_ = d724
				var d725 JITValueDesc
				_ = d725
				var d726 JITValueDesc
				_ = d726
				var d828 JITValueDesc
				_ = d828
				var d829 JITValueDesc
				_ = d829
				var d830 JITValueDesc
				_ = d830
				var d831 JITValueDesc
				_ = d831
				var d832 JITValueDesc
				_ = d832
				var d833 JITValueDesc
				_ = d833
				var d834 JITValueDesc
				_ = d834
				var d835 JITValueDesc
				_ = d835
				var d836 JITValueDesc
				_ = d836
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
				var bbs [29]BBDescriptor
				bbs[6].PhiBase = int32(phiBase0) + int32(0)
				bbs[8].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					d5 = d3
					d5.ID = 0
					d4 = ctx.EmitTagEqualsBorrowed(&d5, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d3)
					d6 = d4
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						snap12 := d6
						alloc13 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc13)
						d1 = snap7
						d2 = snap8
						d3 = snap9
						d4 = snap10
						d5 = snap11
						d6 = snap12
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d4)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d14 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d14, &result)
						result.Type = d14.Type
					} else {
						switch d14.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d14)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d14)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d14)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d14, &result)
							result.Type = d14.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d15 = args[0]
					d15.ID = 0
					d15 = JITPrepareScmerGoArg(ctx, d15)
					ctx.SyncDesc(&d15)
					callResults16 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d15}, []uint8{3, 1}, []uint8{4, 0})
					d17 = callResults16[0]
					_ = d17
					d18 = callResults16[1]
					_ = d18
					ctx.FreeDesc(&d15)
					ctx.StabilizeDescForControlFlow(&d17)
					d19 = d18
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d4
						snap24 := d5
						snap25 := d6
						snap26 := d14
						snap27 := d15
						snap28 := d17
						snap29 := d18
						snap30 := d19
						alloc31 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc31)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d4 = snap23
						d5 = snap24
						d6 = snap25
						d14 = snap26
						d15 = snap27
						d17 = snap28
						d18 = snap29
						d19 = snap30
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d32 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d32)
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d32, &result)
						result.Type = d32.Type
					} else {
						switch d32.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d32)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d32)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d32)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d32, &result)
							result.Type = d32.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d33.Reg, 2)
						d34 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d34)
					}
					ctx.FreeDesc(&d33)
					d35 = d34
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							return bbs[5].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[6].PhiBase)+int32(0))
						return bbs[6].Render()
					}
					lbl30 := ctx.ReserveLabel()
					ctx.EmitJump(d35.Condition, lbl6)
					ctx.EmitJmp(lbl30)
					ctx.FreeDesc(&d34)
					snap36 := d1
					snap37 := d2
					snap38 := d3
					snap39 := d4
					snap40 := d5
					snap41 := d6
					snap42 := d14
					snap43 := d15
					snap44 := d17
					snap45 := d18
					snap46 := d19
					snap47 := d32
					snap48 := d33
					snap49 := d34
					snap50 := d35
					alloc51 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl30)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[6].PhiBase)+int32(0))
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc51)
					d1 = snap36
					d2 = snap37
					d3 = snap38
					d4 = snap39
					d5 = snap40
					d6 = snap41
					d14 = snap42
					d15 = snap43
					d17 = snap44
					d18 = snap45
					d19 = snap46
					d32 = snap47
					d33 = snap48
					d34 = snap49
					d35 = snap50
					if !bbs[6].Rendered {
						snap52 := d1
						snap53 := d2
						snap54 := d3
						snap55 := d4
						snap56 := d5
						snap57 := d6
						snap58 := d14
						snap59 := d15
						snap60 := d17
						snap61 := d18
						snap62 := d19
						snap63 := d32
						snap64 := d33
						snap65 := d34
						snap66 := d35
						alloc67 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc67)
						d1 = snap52
						d2 = snap53
						d3 = snap54
						d4 = snap55
						d5 = snap56
						d6 = snap57
						d14 = snap58
						d15 = snap59
						d17 = snap60
						d18 = snap61
						d19 = snap62
						d32 = snap63
						d33 = snap64
						d34 = snap65
						d35 = snap66
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d68 = args[2]
					d68.ID = 0
					d70 = d68
					ctx.SyncDesc(&d70)
					if d70.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d70.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d70.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d70 = tmpScalar
					}
					d70 = JITPrepareScmerGoArg(ctx, d70)
					if d70.Loc != LocRegPair && d70.Loc != LocStackPair && d70.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d70}, 2)
					ctx.StabilizeDescForControlFlow(&d69)
					ctx.FreeDesc(&d68)
					ctx.SyncDesc(&d69)
					if d69.Loc == LocReg || d69.Loc == LocFPReg {
						ctx.ProtectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.ProtectReg(d69.Reg)
						ctx.ProtectReg(d69.Reg2)
					}
					d71 = d69
					if d71.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d71)
					if d71.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d71, int32(bbs[6].PhiBase)+int32(0), 2)
					} else if d71.Loc == LocInputPair {
						ctx.EnsureDesc(&d71)
						ctx.EmitStoreScmerToStack(d71, int32(bbs[6].PhiBase)+int32(0))
					} else if d71.Loc == LocRegPair || d71.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d71, int32(bbs[6].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d71)
						ctx.EmitStoreToStack(d71, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
					}
					if d69.Loc == LocReg || d69.Loc == LocFPReg {
						ctx.UnprotectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.UnprotectReg(d69.Reg)
						ctx.UnprotectReg(d69.Reg2)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d1.Imm.String())))
						d1 = tmpPair
					} else if d1.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d1)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d1)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d1)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d1)
						d1 = tmpPair
					}
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d1)
					callResults72 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d1}, []uint8{1, 2}, []uint8{1, 3})
					d73 = callResults72[0]
					_ = d73
					d74 = callResults72[1]
					_ = d74
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d73)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d74.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d74)
						if d74.Loc != LocReg && d74.Loc != LocRegPair && d74.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d74.Reg)
						ctx.EmitCmpRegImm32(d74.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d75 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d75)
					}
					ctx.FreeDesc(&d74)
					d76 = d75
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							return bbs[7].Render()
						}
						ctx.SyncDesc(&d73)
						if d73.Loc == LocReg || d73.Loc == LocFPReg {
							ctx.ProtectReg(d73.Reg)
						} else if d73.Loc == LocRegPair {
							ctx.ProtectReg(d73.Reg)
							ctx.ProtectReg(d73.Reg2)
						}
						d77 = d73
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreToStack(d77, int32(bbs[8].PhiBase)+int32(0))
						if d73.Loc == LocReg || d73.Loc == LocFPReg {
							ctx.UnprotectReg(d73.Reg)
						} else if d73.Loc == LocRegPair {
							ctx.UnprotectReg(d73.Reg)
							ctx.UnprotectReg(d73.Reg2)
						}
						return bbs[8].Render()
					}
					lbl31 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl31)
					snap78 := d1
					snap79 := d2
					snap80 := d3
					snap81 := d4
					snap82 := d5
					snap83 := d6
					snap84 := d14
					snap85 := d15
					snap86 := d17
					snap87 := d18
					snap88 := d19
					snap89 := d32
					snap90 := d33
					snap91 := d34
					snap92 := d35
					snap93 := d68
					snap94 := d69
					snap95 := d70
					snap96 := d71
					snap97 := d73
					snap98 := d74
					snap99 := d75
					snap100 := d76
					snap101 := d77
					alloc102 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl31)
					ctx.SyncDesc(&d73)
					if d73.Loc == LocReg || d73.Loc == LocFPReg {
						ctx.ProtectReg(d73.Reg)
					} else if d73.Loc == LocRegPair {
						ctx.ProtectReg(d73.Reg)
						ctx.ProtectReg(d73.Reg2)
					}
					d103 = d73
					if d103.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d103)
					ctx.EmitStoreToStack(d103, int32(bbs[8].PhiBase)+int32(0))
					if d73.Loc == LocReg || d73.Loc == LocFPReg {
						ctx.UnprotectReg(d73.Reg)
					} else if d73.Loc == LocRegPair {
						ctx.UnprotectReg(d73.Reg)
						ctx.UnprotectReg(d73.Reg2)
					}
					ctx.EmitJmp(lbl9)
					ctx.RestoreAllocState(alloc102)
					d1 = snap78
					d2 = snap79
					d3 = snap80
					d4 = snap81
					d5 = snap82
					d6 = snap83
					d14 = snap84
					d15 = snap85
					d17 = snap86
					d18 = snap87
					d19 = snap88
					d32 = snap89
					d33 = snap90
					d34 = snap91
					d35 = snap92
					d68 = snap93
					d69 = snap94
					d70 = snap95
					d71 = snap96
					d73 = snap97
					d74 = snap98
					d75 = snap99
					d76 = snap100
					d77 = snap101
					if !bbs[8].Rendered {
						snap104 := d1
						snap105 := d2
						snap106 := d3
						snap107 := d4
						snap108 := d5
						snap109 := d6
						snap110 := d14
						snap111 := d15
						snap112 := d17
						snap113 := d18
						snap114 := d19
						snap115 := d32
						snap116 := d33
						snap117 := d34
						snap118 := d35
						snap119 := d68
						snap120 := d69
						snap121 := d70
						snap122 := d71
						snap123 := d73
						snap124 := d74
						snap125 := d75
						snap126 := d76
						snap127 := d77
						snap128 := d103
						alloc129 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc129)
						d1 = snap104
						d2 = snap105
						d3 = snap106
						d4 = snap107
						d5 = snap108
						d6 = snap109
						d14 = snap110
						d15 = snap111
						d17 = snap112
						d18 = snap113
						d19 = snap114
						d32 = snap115
						d33 = snap116
						d34 = snap117
						d35 = snap118
						d68 = snap119
						d69 = snap120
						d70 = snap121
						d71 = snap122
						d73 = snap123
						d74 = snap124
						d75 = snap125
						d76 = snap126
						d77 = snap127
						d103 = snap128
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d75)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d130 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.StabilizeDescForControlFlow(&d130)
					ctx.SyncDesc(&d130)
					if d130.Loc == LocReg || d130.Loc == LocFPReg {
						ctx.ProtectReg(d130.Reg)
					} else if d130.Loc == LocRegPair {
						ctx.ProtectReg(d130.Reg)
						ctx.ProtectReg(d130.Reg2)
					}
					d131 = d130
					if d131.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d131)
					ctx.EmitStoreToStack(d131, int32(bbs[8].PhiBase)+int32(0))
					if d130.Loc == LocReg || d130.Loc == LocFPReg {
						ctx.UnprotectReg(d130.Reg)
					} else if d130.Loc == LocRegPair {
						ctx.UnprotectReg(d130.Reg)
						ctx.UnprotectReg(d130.Reg2)
					}
					return bbs[8].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d17 = JITPrepareGoSliceArg(ctx, d17)
					if d17.Loc != LocRegTriple && d17.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d17)
					ctx.SyncDesc(&d2)
					d132 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d17, d2}, 3)
					d132.NoHeapPointer = false
					ctx.BindReg(d132.Reg, &d132)
					ctx.BindReg(d132.Reg2, &d132)
					ctx.BindReg(d132.Reg3, &d132)
					ctx.StabilizeDescForControlFlow(&d132)
					ctx.FreeDesc(&d2)
					d133 = args[1]
					d133.ID = 0
					d135 = d133
					ctx.SyncDesc(&d135)
					if d135.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d135.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d135.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d135 = tmpScalar
					}
					d135 = JITPrepareScmerGoArg(ctx, d135)
					if d135.Loc != LocRegPair && d135.Loc != LocStackPair && d135.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d134 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d135}, 2)
					ctx.FreeDesc(&d133)
					ctx.EnsureDesc(&d134)
					if d134.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d134.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d134.Imm)
						ptrWord, _ := d134.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d134.Imm.String())))
						d134 = tmpPair
					} else if d134.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d134.Type, Reg: ctx.AllocRegExcept(d134.Reg), Reg2: ctx.AllocRegExcept(d134.Reg)}
						switch d134.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d134)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d134)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d134)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d134)
						d134 = tmpPair
					}
					if d134.Loc != LocRegPair && d134.Loc != LocStackPair && d134.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d134)
					d136 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d134}, 2)
					d136.NoHeapPointer = false
					ctx.BindReg(d136.Reg, &d136)
					ctx.BindReg(d136.Reg2, &d136)
					ctx.StabilizeDescForControlFlow(&d136)
					ctx.EnsureDesc(&d136)
					d137 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					if d137.Loc == LocImm {
						ctx.TrackImm(d137.Imm)
						ptrWord, _ := d137.Imm.RawWords()
						d138 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d138.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d138.Reg2, uint64(len(d137.Imm.String())))
						ctx.BindReg(d138.Reg, &d138)
						ctx.BindReg(d138.Reg2, &d138)
					} else {
						d138 = d137
					}
					d139 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d138}, 1)
					ctx.EmitAndRegImm32(d139.Reg, 1)
					d139.Type = tagBool
					ctx.BindReg(d139.Reg, &d139)
					d140 = d139
					ctx.EnsureDesc(&d140)
					if d140.Loc != LocImm && d140.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d140.Loc == LocImm {
						if d140.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitCmpRegImm32(d140.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap141 := d1
						snap142 := d2
						snap143 := d3
						snap144 := d4
						snap145 := d5
						snap146 := d6
						snap147 := d14
						snap148 := d15
						snap149 := d17
						snap150 := d18
						snap151 := d19
						snap152 := d32
						snap153 := d33
						snap154 := d34
						snap155 := d35
						snap156 := d68
						snap157 := d69
						snap158 := d70
						snap159 := d71
						snap160 := d73
						snap161 := d74
						snap162 := d75
						snap163 := d76
						snap164 := d77
						snap165 := d103
						snap166 := d130
						snap167 := d131
						snap168 := d132
						snap169 := d133
						snap170 := d134
						snap171 := d135
						snap172 := d136
						snap173 := d137
						snap174 := d138
						snap175 := d139
						snap176 := d140
						alloc177 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc177)
						d1 = snap141
						d2 = snap142
						d3 = snap143
						d4 = snap144
						d5 = snap145
						d6 = snap146
						d14 = snap147
						d15 = snap148
						d17 = snap149
						d18 = snap150
						d19 = snap151
						d32 = snap152
						d33 = snap153
						d34 = snap154
						d35 = snap155
						d68 = snap156
						d69 = snap157
						d70 = snap158
						d71 = snap159
						d73 = snap160
						d74 = snap161
						d75 = snap162
						d76 = snap163
						d77 = snap164
						d103 = snap165
						d130 = snap166
						d131 = snap167
						d132 = snap168
						d133 = snap169
						d134 = snap170
						d135 = snap171
						d136 = snap172
						d137 = snap173
						d138 = snap174
						d139 = snap175
						d140 = snap176
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					ctx.FreeDesc(&d139)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d132)
					d178 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d132}, 1)
					d178.NoHeapPointer = true
					ctx.BindReg(d178.Reg, &d178)
					ctx.EnsureDesc(&d178)
					ctx.EnsureDesc(&d178)
					ctx.EnsureDesc(&d178)
					if d178.Loc == LocImm {
						ctx.EmitMakeInt(result, d178)
					} else {
						ctx.EmitMovToReg(result.Reg2, d178)
						d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d180)
						if d178.Loc == LocReg && d178.Reg != result.Reg2 {
							ctx.FreeReg(d178.Reg)
						}
					}
					result.Type = tagInt
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d132)
					d181 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d132}, 1)
					d181.NoHeapPointer = true
					ctx.BindReg(d181.Reg, &d181)
					ctx.EnsureDesc(&d181)
					ctx.EnsureDesc(&d181)
					ctx.EnsureDesc(&d181)
					if d181.Loc == LocImm {
						ctx.EmitMakeInt(result, d181)
					} else {
						ctx.EmitMovToReg(result.Reg2, d181)
						d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d183)
						if d181.Loc == LocReg && d181.Reg != result.Reg2 {
							ctx.FreeReg(d181.Reg)
						}
					}
					result.Type = tagInt
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d184 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					if d184.Loc == LocImm {
						ctx.TrackImm(d184.Imm)
						ptrWord, _ := d184.Imm.RawWords()
						d185 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d185.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d185.Reg2, uint64(len(d184.Imm.String())))
						ctx.BindReg(d185.Reg, &d185)
						ctx.BindReg(d185.Reg2, &d185)
					} else {
						d185 = d184
					}
					d186 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d185}, 1)
					ctx.EmitAndRegImm32(d186.Reg, 1)
					d186.Type = tagBool
					ctx.BindReg(d186.Reg, &d186)
					d187 = d186
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocImm && d187.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d187.Loc == LocImm {
						if d187.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[13].Render()
					}
					ctx.EmitCmpRegImm32(d187.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
					}
					ctx.FlushRegisterMoves()
					if !bbs[13].Rendered {
						snap188 := d1
						snap189 := d2
						snap190 := d3
						snap191 := d4
						snap192 := d5
						snap193 := d6
						snap194 := d14
						snap195 := d15
						snap196 := d17
						snap197 := d18
						snap198 := d19
						snap199 := d32
						snap200 := d33
						snap201 := d34
						snap202 := d35
						snap203 := d68
						snap204 := d69
						snap205 := d70
						snap206 := d71
						snap207 := d73
						snap208 := d74
						snap209 := d75
						snap210 := d76
						snap211 := d77
						snap212 := d103
						snap213 := d130
						snap214 := d131
						snap215 := d132
						snap216 := d133
						snap217 := d134
						snap218 := d135
						snap219 := d136
						snap220 := d137
						snap221 := d138
						snap222 := d139
						snap223 := d140
						snap224 := d178
						snap225 := d179
						snap226 := d180
						snap227 := d181
						snap228 := d182
						snap229 := d183
						snap230 := d184
						snap231 := d185
						snap232 := d186
						snap233 := d187
						alloc234 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc234)
						d1 = snap188
						d2 = snap189
						d3 = snap190
						d4 = snap191
						d5 = snap192
						d6 = snap193
						d14 = snap194
						d15 = snap195
						d17 = snap196
						d18 = snap197
						d19 = snap198
						d32 = snap199
						d33 = snap200
						d34 = snap201
						d35 = snap202
						d68 = snap203
						d69 = snap204
						d70 = snap205
						d71 = snap206
						d73 = snap207
						d74 = snap208
						d75 = snap209
						d76 = snap210
						d77 = snap211
						d103 = snap212
						d130 = snap213
						d131 = snap214
						d132 = snap215
						d133 = snap216
						d134 = snap217
						d135 = snap218
						d136 = snap219
						d137 = snap220
						d138 = snap221
						d139 = snap222
						d140 = snap223
						d178 = snap224
						d179 = snap225
						d180 = snap226
						d181 = snap227
						d182 = snap228
						d183 = snap229
						d184 = snap230
						d185 = snap231
						d186 = snap232
						d187 = snap233
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					ctx.FreeDesc(&d186)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d132)
					d235 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d132}, 1)
					d235.NoHeapPointer = true
					ctx.BindReg(d235.Reg, &d235)
					ctx.EnsureDesc(&d235)
					ctx.EnsureDesc(&d235)
					ctx.EnsureDesc(&d235)
					if d235.Loc == LocImm {
						ctx.EmitMakeInt(result, d235)
					} else {
						ctx.EmitMovToReg(result.Reg2, d235)
						d237 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d237)
						if d235.Loc == LocReg && d235.Reg != result.Reg2 {
							ctx.FreeReg(d235.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d238 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
					if d238.Loc == LocImm {
						ctx.TrackImm(d238.Imm)
						ptrWord, _ := d238.Imm.RawWords()
						d239 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d239.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d239.Reg2, uint64(len(d238.Imm.String())))
						ctx.BindReg(d239.Reg, &d239)
						ctx.BindReg(d239.Reg2, &d239)
					} else {
						d239 = d238
					}
					d240 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d239}, 1)
					ctx.EmitAndRegImm32(d240.Reg, 1)
					d240.Type = tagBool
					ctx.BindReg(d240.Reg, &d240)
					d241 = d240
					ctx.EnsureDesc(&d241)
					if d241.Loc != LocImm && d241.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d241.Loc == LocImm {
						if d241.Imm.Bool() {
							return bbs[12].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitCmpRegImm32(d241.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap242 := d1
						snap243 := d2
						snap244 := d3
						snap245 := d4
						snap246 := d5
						snap247 := d6
						snap248 := d14
						snap249 := d15
						snap250 := d17
						snap251 := d18
						snap252 := d19
						snap253 := d32
						snap254 := d33
						snap255 := d34
						snap256 := d35
						snap257 := d68
						snap258 := d69
						snap259 := d70
						snap260 := d71
						snap261 := d73
						snap262 := d74
						snap263 := d75
						snap264 := d76
						snap265 := d77
						snap266 := d103
						snap267 := d130
						snap268 := d131
						snap269 := d132
						snap270 := d133
						snap271 := d134
						snap272 := d135
						snap273 := d136
						snap274 := d137
						snap275 := d138
						snap276 := d139
						snap277 := d140
						snap278 := d178
						snap279 := d179
						snap280 := d180
						snap281 := d181
						snap282 := d182
						snap283 := d183
						snap284 := d184
						snap285 := d185
						snap286 := d186
						snap287 := d187
						snap288 := d235
						snap289 := d236
						snap290 := d237
						snap291 := d238
						snap292 := d239
						snap293 := d240
						snap294 := d241
						alloc295 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc295)
						d1 = snap242
						d2 = snap243
						d3 = snap244
						d4 = snap245
						d5 = snap246
						d6 = snap247
						d14 = snap248
						d15 = snap249
						d17 = snap250
						d18 = snap251
						d19 = snap252
						d32 = snap253
						d33 = snap254
						d34 = snap255
						d35 = snap256
						d68 = snap257
						d69 = snap258
						d70 = snap259
						d71 = snap260
						d73 = snap261
						d74 = snap262
						d75 = snap263
						d76 = snap264
						d77 = snap265
						d103 = snap266
						d130 = snap267
						d131 = snap268
						d132 = snap269
						d133 = snap270
						d134 = snap271
						d135 = snap272
						d136 = snap273
						d137 = snap274
						d138 = snap275
						d139 = snap276
						d140 = snap277
						d178 = snap278
						d179 = snap279
						d180 = snap280
						d181 = snap281
						d182 = snap282
						d183 = snap283
						d184 = snap284
						d185 = snap285
						d186 = snap286
						d187 = snap287
						d235 = snap288
						d236 = snap289
						d237 = snap290
						d238 = snap291
						d239 = snap292
						d240 = snap293
						d241 = snap294
					}
					if !bbs[12].Rendered {
						return bbs[12].Render()
					}
					return result
					ctx.FreeDesc(&d240)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d132)
					d296 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d132}, 1)
					d296.NoHeapPointer = true
					ctx.BindReg(d296.Reg, &d296)
					ctx.EnsureDesc(&d296)
					ctx.EnsureDesc(&d296)
					ctx.EnsureDesc(&d296)
					if d296.Loc == LocImm {
						ctx.EmitMakeInt(result, d296)
					} else {
						ctx.EmitMovToReg(result.Reg2, d296)
						d298 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d298)
						if d296.Loc == LocReg && d296.Reg != result.Reg2 {
							ctx.FreeReg(d296.Reg)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d299 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					if d299.Loc == LocImm {
						ctx.TrackImm(d299.Imm)
						ptrWord, _ := d299.Imm.RawWords()
						d300 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d300.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d300.Reg2, uint64(len(d299.Imm.String())))
						ctx.BindReg(d300.Reg, &d300)
						ctx.BindReg(d300.Reg2, &d300)
					} else {
						d300 = d299
					}
					d301 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d300}, 1)
					ctx.EmitAndRegImm32(d301.Reg, 1)
					d301.Type = tagBool
					ctx.BindReg(d301.Reg, &d301)
					d302 = d301
					ctx.EnsureDesc(&d302)
					if d302.Loc != LocImm && d302.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d302.Loc == LocImm {
						if d302.Imm.Bool() {
							return bbs[14].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitCmpRegImm32(d302.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap303 := d1
						snap304 := d2
						snap305 := d3
						snap306 := d4
						snap307 := d5
						snap308 := d6
						snap309 := d14
						snap310 := d15
						snap311 := d17
						snap312 := d18
						snap313 := d19
						snap314 := d32
						snap315 := d33
						snap316 := d34
						snap317 := d35
						snap318 := d68
						snap319 := d69
						snap320 := d70
						snap321 := d71
						snap322 := d73
						snap323 := d74
						snap324 := d75
						snap325 := d76
						snap326 := d77
						snap327 := d103
						snap328 := d130
						snap329 := d131
						snap330 := d132
						snap331 := d133
						snap332 := d134
						snap333 := d135
						snap334 := d136
						snap335 := d137
						snap336 := d138
						snap337 := d139
						snap338 := d140
						snap339 := d178
						snap340 := d179
						snap341 := d180
						snap342 := d181
						snap343 := d182
						snap344 := d183
						snap345 := d184
						snap346 := d185
						snap347 := d186
						snap348 := d187
						snap349 := d235
						snap350 := d236
						snap351 := d237
						snap352 := d238
						snap353 := d239
						snap354 := d240
						snap355 := d241
						snap356 := d296
						snap357 := d297
						snap358 := d298
						snap359 := d299
						snap360 := d300
						snap361 := d301
						snap362 := d302
						alloc363 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc363)
						d1 = snap303
						d2 = snap304
						d3 = snap305
						d4 = snap306
						d5 = snap307
						d6 = snap308
						d14 = snap309
						d15 = snap310
						d17 = snap311
						d18 = snap312
						d19 = snap313
						d32 = snap314
						d33 = snap315
						d34 = snap316
						d35 = snap317
						d68 = snap318
						d69 = snap319
						d70 = snap320
						d71 = snap321
						d73 = snap322
						d74 = snap323
						d75 = snap324
						d76 = snap325
						d77 = snap326
						d103 = snap327
						d130 = snap328
						d131 = snap329
						d132 = snap330
						d133 = snap331
						d134 = snap332
						d135 = snap333
						d136 = snap334
						d137 = snap335
						d138 = snap336
						d139 = snap337
						d140 = snap338
						d178 = snap339
						d179 = snap340
						d180 = snap341
						d181 = snap342
						d182 = snap343
						d183 = snap344
						d184 = snap345
						d185 = snap346
						d186 = snap347
						d187 = snap348
						d235 = snap349
						d236 = snap350
						d237 = snap351
						d238 = snap352
						d239 = snap353
						d240 = snap354
						d241 = snap355
						d296 = snap356
						d297 = snap357
						d298 = snap358
						d299 = snap359
						d300 = snap360
						d301 = snap361
						d302 = snap362
					}
					if !bbs[14].Rendered {
						return bbs[14].Render()
					}
					return result
					ctx.FreeDesc(&d301)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d132)
					d364 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d132}, 1)
					d364.NoHeapPointer = true
					ctx.BindReg(d364.Reg, &d364)
					ctx.EnsureDesc(&d364)
					ctx.EnsureDesc(&d364)
					ctx.EnsureDesc(&d364)
					if d364.Loc == LocImm {
						ctx.EmitMakeInt(result, d364)
					} else {
						ctx.EmitMovToReg(result.Reg2, d364)
						d366 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d366)
						if d364.Loc == LocReg && d364.Reg != result.Reg2 {
							ctx.FreeReg(d364.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d367 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					if d367.Loc == LocImm {
						ctx.TrackImm(d367.Imm)
						ptrWord, _ := d367.Imm.RawWords()
						d368 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d368.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d368.Reg2, uint64(len(d367.Imm.String())))
						ctx.BindReg(d368.Reg, &d368)
						ctx.BindReg(d368.Reg2, &d368)
					} else {
						d368 = d367
					}
					d369 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d368}, 1)
					ctx.EmitAndRegImm32(d369.Reg, 1)
					d369.Type = tagBool
					ctx.BindReg(d369.Reg, &d369)
					d370 = d369
					ctx.EnsureDesc(&d370)
					if d370.Loc != LocImm && d370.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d370.Loc == LocImm {
						if d370.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d370.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap371 := d1
						snap372 := d2
						snap373 := d3
						snap374 := d4
						snap375 := d5
						snap376 := d6
						snap377 := d14
						snap378 := d15
						snap379 := d17
						snap380 := d18
						snap381 := d19
						snap382 := d32
						snap383 := d33
						snap384 := d34
						snap385 := d35
						snap386 := d68
						snap387 := d69
						snap388 := d70
						snap389 := d71
						snap390 := d73
						snap391 := d74
						snap392 := d75
						snap393 := d76
						snap394 := d77
						snap395 := d103
						snap396 := d130
						snap397 := d131
						snap398 := d132
						snap399 := d133
						snap400 := d134
						snap401 := d135
						snap402 := d136
						snap403 := d137
						snap404 := d138
						snap405 := d139
						snap406 := d140
						snap407 := d178
						snap408 := d179
						snap409 := d180
						snap410 := d181
						snap411 := d182
						snap412 := d183
						snap413 := d184
						snap414 := d185
						snap415 := d186
						snap416 := d187
						snap417 := d235
						snap418 := d236
						snap419 := d237
						snap420 := d238
						snap421 := d239
						snap422 := d240
						snap423 := d241
						snap424 := d296
						snap425 := d297
						snap426 := d298
						snap427 := d299
						snap428 := d300
						snap429 := d301
						snap430 := d302
						snap431 := d364
						snap432 := d365
						snap433 := d366
						snap434 := d367
						snap435 := d368
						snap436 := d369
						snap437 := d370
						alloc438 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc438)
						d1 = snap371
						d2 = snap372
						d3 = snap373
						d4 = snap374
						d5 = snap375
						d6 = snap376
						d14 = snap377
						d15 = snap378
						d17 = snap379
						d18 = snap380
						d19 = snap381
						d32 = snap382
						d33 = snap383
						d34 = snap384
						d35 = snap385
						d68 = snap386
						d69 = snap387
						d70 = snap388
						d71 = snap389
						d73 = snap390
						d74 = snap391
						d75 = snap392
						d76 = snap393
						d77 = snap394
						d103 = snap395
						d130 = snap396
						d131 = snap397
						d132 = snap398
						d133 = snap399
						d134 = snap400
						d135 = snap401
						d136 = snap402
						d137 = snap403
						d138 = snap404
						d139 = snap405
						d140 = snap406
						d178 = snap407
						d179 = snap408
						d180 = snap409
						d181 = snap410
						d182 = snap411
						d183 = snap412
						d184 = snap413
						d185 = snap414
						d186 = snap415
						d187 = snap416
						d235 = snap417
						d236 = snap418
						d237 = snap419
						d238 = snap420
						d239 = snap421
						d240 = snap422
						d241 = snap423
						d296 = snap424
						d297 = snap425
						d298 = snap426
						d299 = snap427
						d300 = snap428
						d301 = snap429
						d302 = snap430
						d364 = snap431
						d365 = snap432
						d366 = snap433
						d367 = snap434
						d368 = snap435
						d369 = snap436
						d370 = snap437
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d369)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d132)
					d439 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d132}, 1)
					d439.NoHeapPointer = true
					ctx.BindReg(d439.Reg, &d439)
					ctx.EnsureDesc(&d439)
					ctx.EnsureDesc(&d439)
					ctx.EnsureDesc(&d439)
					if d439.Loc == LocImm {
						ctx.EmitMakeInt(result, d439)
					} else {
						ctx.EmitMovToReg(result.Reg2, d439)
						d441 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d441)
						if d439.Loc == LocReg && d439.Reg != result.Reg2 {
							ctx.FreeReg(d439.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d442 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					if d442.Loc == LocImm {
						ctx.TrackImm(d442.Imm)
						ptrWord, _ := d442.Imm.RawWords()
						d443 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d443.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d443.Reg2, uint64(len(d442.Imm.String())))
						ctx.BindReg(d443.Reg, &d443)
						ctx.BindReg(d443.Reg2, &d443)
					} else {
						d443 = d442
					}
					d444 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d443}, 1)
					ctx.EmitAndRegImm32(d444.Reg, 1)
					d444.Type = tagBool
					ctx.BindReg(d444.Reg, &d444)
					d445 = d444
					ctx.EnsureDesc(&d445)
					if d445.Loc != LocImm && d445.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d445.Loc == LocImm {
						if d445.Imm.Bool() {
							return bbs[18].Render()
						}
						return bbs[21].Render()
					}
					ctx.EmitCmpRegImm32(d445.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					if bbs[21].Rendered {
						ctx.EmitJmp(lbl22)
					}
					ctx.FlushRegisterMoves()
					if !bbs[21].Rendered {
						snap446 := d1
						snap447 := d2
						snap448 := d3
						snap449 := d4
						snap450 := d5
						snap451 := d6
						snap452 := d14
						snap453 := d15
						snap454 := d17
						snap455 := d18
						snap456 := d19
						snap457 := d32
						snap458 := d33
						snap459 := d34
						snap460 := d35
						snap461 := d68
						snap462 := d69
						snap463 := d70
						snap464 := d71
						snap465 := d73
						snap466 := d74
						snap467 := d75
						snap468 := d76
						snap469 := d77
						snap470 := d103
						snap471 := d130
						snap472 := d131
						snap473 := d132
						snap474 := d133
						snap475 := d134
						snap476 := d135
						snap477 := d136
						snap478 := d137
						snap479 := d138
						snap480 := d139
						snap481 := d140
						snap482 := d178
						snap483 := d179
						snap484 := d180
						snap485 := d181
						snap486 := d182
						snap487 := d183
						snap488 := d184
						snap489 := d185
						snap490 := d186
						snap491 := d187
						snap492 := d235
						snap493 := d236
						snap494 := d237
						snap495 := d238
						snap496 := d239
						snap497 := d240
						snap498 := d241
						snap499 := d296
						snap500 := d297
						snap501 := d298
						snap502 := d299
						snap503 := d300
						snap504 := d301
						snap505 := d302
						snap506 := d364
						snap507 := d365
						snap508 := d366
						snap509 := d367
						snap510 := d368
						snap511 := d369
						snap512 := d370
						snap513 := d439
						snap514 := d440
						snap515 := d441
						snap516 := d442
						snap517 := d443
						snap518 := d444
						snap519 := d445
						alloc520 := ctx.SnapshotAllocState()
						bbs[21].Render()
						ctx.RestoreAllocState(alloc520)
						d1 = snap446
						d2 = snap447
						d3 = snap448
						d4 = snap449
						d5 = snap450
						d6 = snap451
						d14 = snap452
						d15 = snap453
						d17 = snap454
						d18 = snap455
						d19 = snap456
						d32 = snap457
						d33 = snap458
						d34 = snap459
						d35 = snap460
						d68 = snap461
						d69 = snap462
						d70 = snap463
						d71 = snap464
						d73 = snap465
						d74 = snap466
						d75 = snap467
						d76 = snap468
						d77 = snap469
						d103 = snap470
						d130 = snap471
						d131 = snap472
						d132 = snap473
						d133 = snap474
						d134 = snap475
						d135 = snap476
						d136 = snap477
						d137 = snap478
						d138 = snap479
						d139 = snap480
						d140 = snap481
						d178 = snap482
						d179 = snap483
						d180 = snap484
						d181 = snap485
						d182 = snap486
						d183 = snap487
						d184 = snap488
						d185 = snap489
						d186 = snap490
						d187 = snap491
						d235 = snap492
						d236 = snap493
						d237 = snap494
						d238 = snap495
						d239 = snap496
						d240 = snap497
						d241 = snap498
						d296 = snap499
						d297 = snap500
						d298 = snap501
						d299 = snap502
						d300 = snap503
						d301 = snap504
						d302 = snap505
						d364 = snap506
						d365 = snap507
						d366 = snap508
						d367 = snap509
						d368 = snap510
						d369 = snap511
						d370 = snap512
						d439 = snap513
						d440 = snap514
						d441 = snap515
						d442 = snap516
						d443 = snap517
						d444 = snap518
						d445 = snap519
					}
					if !bbs[18].Rendered {
						return bbs[18].Render()
					}
					return result
					ctx.FreeDesc(&d444)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d132)
					d521 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d132}, 1)
					d521.NoHeapPointer = true
					ctx.BindReg(d521.Reg, &d521)
					ctx.EnsureDesc(&d521)
					ctx.FreeDesc(&d521)
					ctx.EnsureDesc(&d521)
					ctx.EnsureDesc(&d521)
					if d521.Loc == LocImm {
						d522 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d521.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d521.Reg)
						ctx.EmitMovRegReg(scratch, d521.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d522 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d522)
					}
					if d522.Loc == LocReg && d521.Loc == LocReg && d522.Reg == d521.Reg {
						ctx.TransferReg(d521.Reg)
						d521.Loc = LocNone
					}
					ctx.FreeDesc(&d521)
					ctx.EnsureDesc(&d522)
					if d522.Loc == LocImm {
						d523 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d522.Imm.Int() / 3)}
					} else {
						ctx.EmitIdivRegImm(d522.Reg, 3)
						d523 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d522.Reg}
						ctx.BindReg(d522.Reg, &d523)
					}
					if d523.Loc == LocReg && d522.Loc == LocReg && d523.Reg == d522.Reg {
						ctx.TransferReg(d522.Reg)
						d522.Loc = LocNone
					}
					ctx.FreeDesc(&d522)
					ctx.EnsureDesc(&d523)
					ctx.EnsureDesc(&d523)
					if d523.Loc == LocImm {
						d524 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d523.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d523.Reg)
						ctx.EmitMovRegReg(scratch, d523.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d524 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d524)
					}
					if d524.Loc == LocReg && d523.Loc == LocReg && d524.Reg == d523.Reg {
						ctx.TransferReg(d523.Reg)
						d523.Loc = LocNone
					}
					ctx.FreeDesc(&d523)
					ctx.EnsureDesc(&d524)
					ctx.EnsureDesc(&d524)
					ctx.EnsureDesc(&d524)
					if d524.Loc == LocImm {
						ctx.EmitMakeInt(result, d524)
					} else {
						ctx.EmitMovToReg(result.Reg2, d524)
						d526 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d526)
						if d524.Loc == LocReg && d524.Reg != result.Reg2 {
							ctx.FreeReg(d524.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d527 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("QUARTER")}
					if d527.Loc == LocImm {
						ctx.TrackImm(d527.Imm)
						ptrWord, _ := d527.Imm.RawWords()
						d528 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d528.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d528.Reg2, uint64(len(d527.Imm.String())))
						ctx.BindReg(d528.Reg, &d528)
						ctx.BindReg(d528.Reg2, &d528)
					} else {
						d528 = d527
					}
					d529 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d528}, 1)
					ctx.EmitAndRegImm32(d529.Reg, 1)
					d529.Type = tagBool
					ctx.BindReg(d529.Reg, &d529)
					d530 = d529
					ctx.EnsureDesc(&d530)
					if d530.Loc != LocImm && d530.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d530.Loc == LocImm {
						if d530.Imm.Bool() {
							return bbs[20].Render()
						}
						return bbs[23].Render()
					}
					ctx.EmitCmpRegImm32(d530.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					if bbs[23].Rendered {
						ctx.EmitJmp(lbl24)
					}
					ctx.FlushRegisterMoves()
					if !bbs[23].Rendered {
						snap531 := d1
						snap532 := d2
						snap533 := d3
						snap534 := d4
						snap535 := d5
						snap536 := d6
						snap537 := d14
						snap538 := d15
						snap539 := d17
						snap540 := d18
						snap541 := d19
						snap542 := d32
						snap543 := d33
						snap544 := d34
						snap545 := d35
						snap546 := d68
						snap547 := d69
						snap548 := d70
						snap549 := d71
						snap550 := d73
						snap551 := d74
						snap552 := d75
						snap553 := d76
						snap554 := d77
						snap555 := d103
						snap556 := d130
						snap557 := d131
						snap558 := d132
						snap559 := d133
						snap560 := d134
						snap561 := d135
						snap562 := d136
						snap563 := d137
						snap564 := d138
						snap565 := d139
						snap566 := d140
						snap567 := d178
						snap568 := d179
						snap569 := d180
						snap570 := d181
						snap571 := d182
						snap572 := d183
						snap573 := d184
						snap574 := d185
						snap575 := d186
						snap576 := d187
						snap577 := d235
						snap578 := d236
						snap579 := d237
						snap580 := d238
						snap581 := d239
						snap582 := d240
						snap583 := d241
						snap584 := d296
						snap585 := d297
						snap586 := d298
						snap587 := d299
						snap588 := d300
						snap589 := d301
						snap590 := d302
						snap591 := d364
						snap592 := d365
						snap593 := d366
						snap594 := d367
						snap595 := d368
						snap596 := d369
						snap597 := d370
						snap598 := d439
						snap599 := d440
						snap600 := d441
						snap601 := d442
						snap602 := d443
						snap603 := d444
						snap604 := d445
						snap605 := d521
						snap606 := d522
						snap607 := d523
						snap608 := d524
						snap609 := d525
						snap610 := d526
						snap611 := d527
						snap612 := d528
						snap613 := d529
						snap614 := d530
						alloc615 := ctx.SnapshotAllocState()
						bbs[23].Render()
						ctx.RestoreAllocState(alloc615)
						d1 = snap531
						d2 = snap532
						d3 = snap533
						d4 = snap534
						d5 = snap535
						d6 = snap536
						d14 = snap537
						d15 = snap538
						d17 = snap539
						d18 = snap540
						d19 = snap541
						d32 = snap542
						d33 = snap543
						d34 = snap544
						d35 = snap545
						d68 = snap546
						d69 = snap547
						d70 = snap548
						d71 = snap549
						d73 = snap550
						d74 = snap551
						d75 = snap552
						d76 = snap553
						d77 = snap554
						d103 = snap555
						d130 = snap556
						d131 = snap557
						d132 = snap558
						d133 = snap559
						d134 = snap560
						d135 = snap561
						d136 = snap562
						d137 = snap563
						d138 = snap564
						d139 = snap565
						d140 = snap566
						d178 = snap567
						d179 = snap568
						d180 = snap569
						d181 = snap570
						d182 = snap571
						d183 = snap572
						d184 = snap573
						d185 = snap574
						d186 = snap575
						d187 = snap576
						d235 = snap577
						d236 = snap578
						d237 = snap579
						d238 = snap580
						d239 = snap581
						d240 = snap582
						d241 = snap583
						d296 = snap584
						d297 = snap585
						d298 = snap586
						d299 = snap587
						d300 = snap588
						d301 = snap589
						d302 = snap590
						d364 = snap591
						d365 = snap592
						d366 = snap593
						d367 = snap594
						d368 = snap595
						d369 = snap596
						d370 = snap597
						d439 = snap598
						d440 = snap599
						d441 = snap600
						d442 = snap601
						d443 = snap602
						d444 = snap603
						d445 = snap604
						d521 = snap605
						d522 = snap606
						d523 = snap607
						d524 = snap608
						d525 = snap609
						d526 = snap610
						d527 = snap611
						d528 = snap612
						d529 = snap613
						d530 = snap614
					}
					if !bbs[20].Rendered {
						return bbs[20].Render()
					}
					return result
					ctx.FreeDesc(&d529)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).ISOWeek arg0)")
					}
					ctx.SyncDesc(&d132)
					callResults616 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).ISOWeek), []JITValueDesc{d132}, []uint8{1, 1}, []uint8{0, 0})
					d617 = callResults616[0]
					_ = d617
					d618 = callResults616[1]
					_ = d618
					ctx.EnsureDesc(&d618)
					ctx.EnsureDesc(&d618)
					ctx.EnsureDesc(&d618)
					if d618.Loc == LocImm {
						ctx.EmitMakeInt(result, d618)
					} else {
						ctx.EmitMovToReg(result.Reg2, d618)
						d620 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d620)
						if d618.Loc == LocReg && d618.Reg != result.Reg2 {
							ctx.FreeReg(d618.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d621 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					if d621.Loc == LocImm {
						ctx.TrackImm(d621.Imm)
						ptrWord, _ := d621.Imm.RawWords()
						d622 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d622.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d622.Reg2, uint64(len(d621.Imm.String())))
						ctx.BindReg(d622.Reg, &d622)
						ctx.BindReg(d622.Reg2, &d622)
					} else {
						d622 = d621
					}
					d623 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d622}, 1)
					ctx.EmitAndRegImm32(d623.Reg, 1)
					d623.Type = tagBool
					ctx.BindReg(d623.Reg, &d623)
					d624 = d623
					ctx.EnsureDesc(&d624)
					if d624.Loc != LocImm && d624.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d624.Loc == LocImm {
						if d624.Imm.Bool() {
							return bbs[22].Render()
						}
						return bbs[25].Render()
					}
					ctx.EmitCmpRegImm32(d624.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl23)
					if bbs[25].Rendered {
						ctx.EmitJmp(lbl26)
					}
					ctx.FlushRegisterMoves()
					if !bbs[25].Rendered {
						snap625 := d1
						snap626 := d2
						snap627 := d3
						snap628 := d4
						snap629 := d5
						snap630 := d6
						snap631 := d14
						snap632 := d15
						snap633 := d17
						snap634 := d18
						snap635 := d19
						snap636 := d32
						snap637 := d33
						snap638 := d34
						snap639 := d35
						snap640 := d68
						snap641 := d69
						snap642 := d70
						snap643 := d71
						snap644 := d73
						snap645 := d74
						snap646 := d75
						snap647 := d76
						snap648 := d77
						snap649 := d103
						snap650 := d130
						snap651 := d131
						snap652 := d132
						snap653 := d133
						snap654 := d134
						snap655 := d135
						snap656 := d136
						snap657 := d137
						snap658 := d138
						snap659 := d139
						snap660 := d140
						snap661 := d178
						snap662 := d179
						snap663 := d180
						snap664 := d181
						snap665 := d182
						snap666 := d183
						snap667 := d184
						snap668 := d185
						snap669 := d186
						snap670 := d187
						snap671 := d235
						snap672 := d236
						snap673 := d237
						snap674 := d238
						snap675 := d239
						snap676 := d240
						snap677 := d241
						snap678 := d296
						snap679 := d297
						snap680 := d298
						snap681 := d299
						snap682 := d300
						snap683 := d301
						snap684 := d302
						snap685 := d364
						snap686 := d365
						snap687 := d366
						snap688 := d367
						snap689 := d368
						snap690 := d369
						snap691 := d370
						snap692 := d439
						snap693 := d440
						snap694 := d441
						snap695 := d442
						snap696 := d443
						snap697 := d444
						snap698 := d445
						snap699 := d521
						snap700 := d522
						snap701 := d523
						snap702 := d524
						snap703 := d525
						snap704 := d526
						snap705 := d527
						snap706 := d528
						snap707 := d529
						snap708 := d530
						snap709 := d617
						snap710 := d618
						snap711 := d619
						snap712 := d620
						snap713 := d621
						snap714 := d622
						snap715 := d623
						snap716 := d624
						alloc717 := ctx.SnapshotAllocState()
						bbs[25].Render()
						ctx.RestoreAllocState(alloc717)
						d1 = snap625
						d2 = snap626
						d3 = snap627
						d4 = snap628
						d5 = snap629
						d6 = snap630
						d14 = snap631
						d15 = snap632
						d17 = snap633
						d18 = snap634
						d19 = snap635
						d32 = snap636
						d33 = snap637
						d34 = snap638
						d35 = snap639
						d68 = snap640
						d69 = snap641
						d70 = snap642
						d71 = snap643
						d73 = snap644
						d74 = snap645
						d75 = snap646
						d76 = snap647
						d77 = snap648
						d103 = snap649
						d130 = snap650
						d131 = snap651
						d132 = snap652
						d133 = snap653
						d134 = snap654
						d135 = snap655
						d136 = snap656
						d137 = snap657
						d138 = snap658
						d139 = snap659
						d140 = snap660
						d178 = snap661
						d179 = snap662
						d180 = snap663
						d181 = snap664
						d182 = snap665
						d183 = snap666
						d184 = snap667
						d185 = snap668
						d186 = snap669
						d187 = snap670
						d235 = snap671
						d236 = snap672
						d237 = snap673
						d238 = snap674
						d239 = snap675
						d240 = snap676
						d241 = snap677
						d296 = snap678
						d297 = snap679
						d298 = snap680
						d299 = snap681
						d300 = snap682
						d301 = snap683
						d302 = snap684
						d364 = snap685
						d365 = snap686
						d366 = snap687
						d367 = snap688
						d368 = snap689
						d369 = snap690
						d370 = snap691
						d439 = snap692
						d440 = snap693
						d441 = snap694
						d442 = snap695
						d443 = snap696
						d444 = snap697
						d445 = snap698
						d521 = snap699
						d522 = snap700
						d523 = snap701
						d524 = snap702
						d525 = snap703
						d526 = snap704
						d527 = snap705
						d528 = snap706
						d529 = snap707
						d530 = snap708
						d617 = snap709
						d618 = snap710
						d619 = snap711
						d620 = snap712
						d621 = snap713
						d622 = snap714
						d623 = snap715
						d624 = snap716
					}
					if !bbs[22].Rendered {
						return bbs[22].Render()
					}
					return result
					ctx.FreeDesc(&d623)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Weekday arg0)")
					}
					ctx.SyncDesc(&d132)
					d718 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Weekday), []JITValueDesc{d132}, 1)
					d718.NoHeapPointer = true
					ctx.BindReg(d718.Reg, &d718)
					ctx.EnsureDesc(&d718)
					ctx.EnsureDesc(&d718)
					ctx.EnsureDesc(&d718)
					resultTarget720 := false
					_ = resultTarget720
					ctx.EnsureDesc(&d718)
					if d718.Loc == LocImm {
						d721 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d718.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if result.Loc == LocRegPair && result.Reg2 != d718.Reg {
							scratch = result.Reg2
							resultTarget720 = true
						} else {
							scratch = ctx.AllocRegExcept(d718.Reg)
						}
						ctx.EmitMovRegReg(scratch, d718.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d721 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d721)
					}
					if d721.Loc == LocReg && d718.Loc == LocReg && d721.Reg == d718.Reg {
						ctx.TransferReg(d718.Reg)
						d718.Loc = LocNone
					}
					if resultTarget720 && d721.Loc == LocReg {
						ctx.BindReg(result.Reg2, &result)
					}
					ctx.FreeDesc(&d718)
					ctx.EnsureDesc(&d721)
					if d721.Loc == LocImm {
						ctx.EmitMakeInt(result, d721)
					} else {
						ctx.EmitMovToReg(result.Reg2, d721)
						d722 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d722)
						if d721.Loc == LocReg && d721.Reg != result.Reg2 {
							ctx.FreeReg(d721.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d723 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAYOFWEEK")}
					if d723.Loc == LocImm {
						ctx.TrackImm(d723.Imm)
						ptrWord, _ := d723.Imm.RawWords()
						d724 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d724.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d724.Reg2, uint64(len(d723.Imm.String())))
						ctx.BindReg(d724.Reg, &d724)
						ctx.BindReg(d724.Reg2, &d724)
					} else {
						d724 = d723
					}
					d725 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d724}, 1)
					ctx.EmitAndRegImm32(d725.Reg, 1)
					d725.Type = tagBool
					ctx.BindReg(d725.Reg, &d725)
					d726 = d725
					ctx.EnsureDesc(&d726)
					if d726.Loc != LocImm && d726.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d726.Loc == LocImm {
						if d726.Imm.Bool() {
							return bbs[24].Render()
						}
						return bbs[27].Render()
					}
					ctx.EmitCmpRegImm32(d726.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl25)
					if bbs[27].Rendered {
						ctx.EmitJmp(lbl28)
					}
					ctx.FlushRegisterMoves()
					if !bbs[27].Rendered {
						snap727 := d1
						snap728 := d2
						snap729 := d3
						snap730 := d4
						snap731 := d5
						snap732 := d6
						snap733 := d14
						snap734 := d15
						snap735 := d17
						snap736 := d18
						snap737 := d19
						snap738 := d32
						snap739 := d33
						snap740 := d34
						snap741 := d35
						snap742 := d68
						snap743 := d69
						snap744 := d70
						snap745 := d71
						snap746 := d73
						snap747 := d74
						snap748 := d75
						snap749 := d76
						snap750 := d77
						snap751 := d103
						snap752 := d130
						snap753 := d131
						snap754 := d132
						snap755 := d133
						snap756 := d134
						snap757 := d135
						snap758 := d136
						snap759 := d137
						snap760 := d138
						snap761 := d139
						snap762 := d140
						snap763 := d178
						snap764 := d179
						snap765 := d180
						snap766 := d181
						snap767 := d182
						snap768 := d183
						snap769 := d184
						snap770 := d185
						snap771 := d186
						snap772 := d187
						snap773 := d235
						snap774 := d236
						snap775 := d237
						snap776 := d238
						snap777 := d239
						snap778 := d240
						snap779 := d241
						snap780 := d296
						snap781 := d297
						snap782 := d298
						snap783 := d299
						snap784 := d300
						snap785 := d301
						snap786 := d302
						snap787 := d364
						snap788 := d365
						snap789 := d366
						snap790 := d367
						snap791 := d368
						snap792 := d369
						snap793 := d370
						snap794 := d439
						snap795 := d440
						snap796 := d441
						snap797 := d442
						snap798 := d443
						snap799 := d444
						snap800 := d445
						snap801 := d521
						snap802 := d522
						snap803 := d523
						snap804 := d524
						snap805 := d525
						snap806 := d526
						snap807 := d527
						snap808 := d528
						snap809 := d529
						snap810 := d530
						snap811 := d617
						snap812 := d618
						snap813 := d619
						snap814 := d620
						snap815 := d621
						snap816 := d622
						snap817 := d623
						snap818 := d624
						snap819 := d718
						snap820 := d719
						snap821 := d721
						snap822 := d722
						snap823 := d723
						snap824 := d724
						snap825 := d725
						snap826 := d726
						alloc827 := ctx.SnapshotAllocState()
						bbs[27].Render()
						ctx.RestoreAllocState(alloc827)
						d1 = snap727
						d2 = snap728
						d3 = snap729
						d4 = snap730
						d5 = snap731
						d6 = snap732
						d14 = snap733
						d15 = snap734
						d17 = snap735
						d18 = snap736
						d19 = snap737
						d32 = snap738
						d33 = snap739
						d34 = snap740
						d35 = snap741
						d68 = snap742
						d69 = snap743
						d70 = snap744
						d71 = snap745
						d73 = snap746
						d74 = snap747
						d75 = snap748
						d76 = snap749
						d77 = snap750
						d103 = snap751
						d130 = snap752
						d131 = snap753
						d132 = snap754
						d133 = snap755
						d134 = snap756
						d135 = snap757
						d136 = snap758
						d137 = snap759
						d138 = snap760
						d139 = snap761
						d140 = snap762
						d178 = snap763
						d179 = snap764
						d180 = snap765
						d181 = snap766
						d182 = snap767
						d183 = snap768
						d184 = snap769
						d185 = snap770
						d186 = snap771
						d187 = snap772
						d235 = snap773
						d236 = snap774
						d237 = snap775
						d238 = snap776
						d239 = snap777
						d240 = snap778
						d241 = snap779
						d296 = snap780
						d297 = snap781
						d298 = snap782
						d299 = snap783
						d300 = snap784
						d301 = snap785
						d302 = snap786
						d364 = snap787
						d365 = snap788
						d366 = snap789
						d367 = snap790
						d368 = snap791
						d369 = snap792
						d370 = snap793
						d439 = snap794
						d440 = snap795
						d441 = snap796
						d442 = snap797
						d443 = snap798
						d444 = snap799
						d445 = snap800
						d521 = snap801
						d522 = snap802
						d523 = snap803
						d524 = snap804
						d525 = snap805
						d526 = snap806
						d527 = snap807
						d528 = snap808
						d529 = snap809
						d530 = snap810
						d617 = snap811
						d618 = snap812
						d619 = snap813
						d620 = snap814
						d621 = snap815
						d622 = snap816
						d623 = snap817
						d624 = snap818
						d718 = snap819
						d719 = snap820
						d721 = snap821
						d722 = snap822
						d723 = snap823
						d724 = snap824
						d725 = snap825
						d726 = snap826
					}
					if !bbs[24].Rendered {
						return bbs[24].Render()
					}
					return result
					ctx.FreeDesc(&d725)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareGoSliceArg(ctx, d132)
					if d132.Loc != LocRegTriple && d132.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Weekday arg0)")
					}
					ctx.SyncDesc(&d132)
					d828 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Weekday), []JITValueDesc{d132}, 1)
					d828.NoHeapPointer = true
					ctx.BindReg(d828.Reg, &d828)
					ctx.EnsureDesc(&d828)
					ctx.EnsureDesc(&d828)
					if d828.Loc == LocImm {
						d829 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d828.Imm.Int() + 6)}
					} else {
						scratch := ctx.AllocRegExcept(d828.Reg)
						ctx.EmitMovRegReg(scratch, d828.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 6)
						d829 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d829)
					}
					if d829.Loc == LocReg && d828.Loc == LocReg && d829.Reg == d828.Reg {
						ctx.TransferReg(d828.Reg)
						d828.Loc = LocNone
					}
					ctx.FreeDesc(&d828)
					ctx.EnsureDesc(&d829)
					if d829.Loc == LocImm {
						d830 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d829.Imm.Int() % 7)}
					} else {
						ctx.EmitIremRegImm(d829.Reg, 7)
						d830 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d829.Reg}
						ctx.BindReg(d829.Reg, &d830)
					}
					if d830.Loc == LocReg && d829.Loc == LocReg && d830.Reg == d829.Reg {
						ctx.TransferReg(d829.Reg)
						d829.Loc = LocNone
					}
					ctx.FreeDesc(&d829)
					ctx.EnsureDesc(&d830)
					ctx.EnsureDesc(&d830)
					ctx.EnsureDesc(&d830)
					if d830.Loc == LocImm {
						ctx.EmitMakeInt(result, d830)
					} else {
						ctx.EmitMovToReg(result.Reg2, d830)
						d832 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d832)
						if d830.Loc == LocReg && d830.Reg != result.Reg2 {
							ctx.FreeReg(d830.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d833 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEKDAY")}
					if d833.Loc == LocImm {
						ctx.TrackImm(d833.Imm)
						ptrWord, _ := d833.Imm.RawWords()
						d834 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d834.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d834.Reg2, uint64(len(d833.Imm.String())))
						ctx.BindReg(d834.Reg, &d834)
						ctx.BindReg(d834.Reg2, &d834)
					} else {
						d834 = d833
					}
					d835 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d136, d834}, 1)
					ctx.EmitAndRegImm32(d835.Reg, 1)
					d835.Type = tagBool
					ctx.BindReg(d835.Reg, &d835)
					d836 = d835
					ctx.EnsureDesc(&d836)
					if d836.Loc != LocImm && d836.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d836.Loc == LocImm {
						if d836.Imm.Bool() {
							return bbs[26].Render()
						}
						return bbs[28].Render()
					}
					ctx.EmitCmpRegImm32(d836.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl27)
					if bbs[28].Rendered {
						ctx.EmitJmp(lbl29)
					}
					ctx.FlushRegisterMoves()
					if !bbs[28].Rendered {
						snap837 := d1
						snap838 := d2
						snap839 := d3
						snap840 := d4
						snap841 := d5
						snap842 := d6
						snap843 := d14
						snap844 := d15
						snap845 := d17
						snap846 := d18
						snap847 := d19
						snap848 := d32
						snap849 := d33
						snap850 := d34
						snap851 := d35
						snap852 := d68
						snap853 := d69
						snap854 := d70
						snap855 := d71
						snap856 := d73
						snap857 := d74
						snap858 := d75
						snap859 := d76
						snap860 := d77
						snap861 := d103
						snap862 := d130
						snap863 := d131
						snap864 := d132
						snap865 := d133
						snap866 := d134
						snap867 := d135
						snap868 := d136
						snap869 := d137
						snap870 := d138
						snap871 := d139
						snap872 := d140
						snap873 := d178
						snap874 := d179
						snap875 := d180
						snap876 := d181
						snap877 := d182
						snap878 := d183
						snap879 := d184
						snap880 := d185
						snap881 := d186
						snap882 := d187
						snap883 := d235
						snap884 := d236
						snap885 := d237
						snap886 := d238
						snap887 := d239
						snap888 := d240
						snap889 := d241
						snap890 := d296
						snap891 := d297
						snap892 := d298
						snap893 := d299
						snap894 := d300
						snap895 := d301
						snap896 := d302
						snap897 := d364
						snap898 := d365
						snap899 := d366
						snap900 := d367
						snap901 := d368
						snap902 := d369
						snap903 := d370
						snap904 := d439
						snap905 := d440
						snap906 := d441
						snap907 := d442
						snap908 := d443
						snap909 := d444
						snap910 := d445
						snap911 := d521
						snap912 := d522
						snap913 := d523
						snap914 := d524
						snap915 := d525
						snap916 := d526
						snap917 := d527
						snap918 := d528
						snap919 := d529
						snap920 := d530
						snap921 := d617
						snap922 := d618
						snap923 := d619
						snap924 := d620
						snap925 := d621
						snap926 := d622
						snap927 := d623
						snap928 := d624
						snap929 := d718
						snap930 := d719
						snap931 := d721
						snap932 := d722
						snap933 := d723
						snap934 := d724
						snap935 := d725
						snap936 := d726
						snap937 := d828
						snap938 := d829
						snap939 := d830
						snap940 := d831
						snap941 := d832
						snap942 := d833
						snap943 := d834
						snap944 := d835
						snap945 := d836
						alloc946 := ctx.SnapshotAllocState()
						bbs[28].Render()
						ctx.RestoreAllocState(alloc946)
						d1 = snap837
						d2 = snap838
						d3 = snap839
						d4 = snap840
						d5 = snap841
						d6 = snap842
						d14 = snap843
						d15 = snap844
						d17 = snap845
						d18 = snap846
						d19 = snap847
						d32 = snap848
						d33 = snap849
						d34 = snap850
						d35 = snap851
						d68 = snap852
						d69 = snap853
						d70 = snap854
						d71 = snap855
						d73 = snap856
						d74 = snap857
						d75 = snap858
						d76 = snap859
						d77 = snap860
						d103 = snap861
						d130 = snap862
						d131 = snap863
						d132 = snap864
						d133 = snap865
						d134 = snap866
						d135 = snap867
						d136 = snap868
						d137 = snap869
						d138 = snap870
						d139 = snap871
						d140 = snap872
						d178 = snap873
						d179 = snap874
						d180 = snap875
						d181 = snap876
						d182 = snap877
						d183 = snap878
						d184 = snap879
						d185 = snap880
						d186 = snap881
						d187 = snap882
						d235 = snap883
						d236 = snap884
						d237 = snap885
						d238 = snap886
						d239 = snap887
						d240 = snap888
						d241 = snap889
						d296 = snap890
						d297 = snap891
						d298 = snap892
						d299 = snap893
						d300 = snap894
						d301 = snap895
						d302 = snap896
						d364 = snap897
						d365 = snap898
						d366 = snap899
						d367 = snap900
						d368 = snap901
						d369 = snap902
						d370 = snap903
						d439 = snap904
						d440 = snap905
						d441 = snap906
						d442 = snap907
						d443 = snap908
						d444 = snap909
						d445 = snap910
						d521 = snap911
						d522 = snap912
						d523 = snap913
						d524 = snap914
						d525 = snap915
						d526 = snap916
						d527 = snap917
						d528 = snap918
						d529 = snap919
						d530 = snap920
						d617 = snap921
						d618 = snap922
						d619 = snap923
						d620 = snap924
						d621 = snap925
						d622 = snap926
						d623 = snap927
						d624 = snap928
						d718 = snap929
						d719 = snap930
						d721 = snap931
						d722 = snap932
						d723 = snap933
						d724 = snap934
						d725 = snap935
						d726 = snap936
						d828 = snap937
						d829 = snap938
						d830 = snap939
						d831 = snap940
						d832 = snap941
						d833 = snap942
						d834 = snap943
						d835 = snap944
						d836 = snap945
					}
					if !bbs[26].Rendered {
						return bbs[26].Render()
					}
					return result
					ctx.FreeDesc(&d835)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["extract_date"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
			JITInlineCost:  107,
		},
	})

	// DATE_ADD(expr, interval_seconds)
	Declare(&Globalenv, &Declaration{
		Name: "date_add",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			amount := int(a[1].Int())
			unit := strings.ToUpper(a[2].String())
			switch unit {
			case "SECOND":
				t = t.Add(time.Duration(amount) * time.Second)
			case "MINUTE":
				t = t.Add(time.Duration(amount) * time.Minute)
			case "HOUR":
				t = t.Add(time.Duration(amount) * time.Hour)
			case "DAY":
				t = t.AddDate(0, 0, amount)
			case "WEEK":
				t = t.AddDate(0, 0, amount*7)
			case "MONTH":
				t = t.AddDate(0, amount, 0)
			case "YEAR":
				t = t.AddDate(amount, 0, 0)
			default:
				panic("unknown DATE_ADD unit: " + unit)
			}
			return NewDate(t.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "adds an interval to a date value",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date value"}, &TypeDescriptor{Kind: "int", Label: "amount", Description: "interval amount"}, &TypeDescriptor{Kind: "string", Label: "unit", Description: "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["date_add"]
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
				var d32 JITValueDesc
				_ = d32
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
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
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d173 JITValueDesc
				_ = d173
				var d174 JITValueDesc
				_ = d174
				var d175 JITValueDesc
				_ = d175
				var d176 JITValueDesc
				_ = d176
				var d177 JITValueDesc
				_ = d177
				var d178 JITValueDesc
				_ = d178
				var d179 JITValueDesc
				_ = d179
				var d180 JITValueDesc
				_ = d180
				var d234 JITValueDesc
				_ = d234
				var d235 JITValueDesc
				_ = d235
				var d236 JITValueDesc
				_ = d236
				var d237 JITValueDesc
				_ = d237
				var d238 JITValueDesc
				_ = d238
				var d239 JITValueDesc
				_ = d239
				var d240 JITValueDesc
				_ = d240
				var d241 JITValueDesc
				_ = d241
				var d303 JITValueDesc
				_ = d303
				var d304 JITValueDesc
				_ = d304
				var d305 JITValueDesc
				_ = d305
				var d306 JITValueDesc
				_ = d306
				var d307 JITValueDesc
				_ = d307
				var d308 JITValueDesc
				_ = d308
				var d309 JITValueDesc
				_ = d309
				var d310 JITValueDesc
				_ = d310
				var d311 JITValueDesc
				_ = d311
				var d382 JITValueDesc
				_ = d382
				var d383 JITValueDesc
				_ = d383
				var d384 JITValueDesc
				_ = d384
				var d385 JITValueDesc
				_ = d385
				var d386 JITValueDesc
				_ = d386
				var d387 JITValueDesc
				_ = d387
				var d388 JITValueDesc
				_ = d388
				var d389 JITValueDesc
				_ = d389
				var d468 JITValueDesc
				_ = d468
				var d469 JITValueDesc
				_ = d469
				var d470 JITValueDesc
				_ = d470
				var d471 JITValueDesc
				_ = d471
				var d472 JITValueDesc
				_ = d472
				var d473 JITValueDesc
				_ = d473
				var d474 JITValueDesc
				_ = d474
				var d475 JITValueDesc
				_ = d475
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
				var bbs [22]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
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
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						snap10 := d5
						alloc11 := ctx.SnapshotAllocState()
						bbs[4].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					ctx.SyncDesc(&d13)
					callResults14 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d13}, []uint8{3, 1}, []uint8{4, 0})
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
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
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
						bbs[5].Render()
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
					if !bbs[6].Rendered {
						return bbs[6].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d29 = args[2]
					d29.ID = 0
					d31 = d29
					d31.ID = 0
					d30 = ctx.EmitTagEqualsBorrowed(&d31, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d29)
					d32 = d30
					ctx.EnsureDesc(&d32)
					if d32.Loc != LocImm && d32.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d32.Loc == LocImm {
						if d32.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d32.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap33 := d1
						snap34 := d2
						snap35 := d3
						snap36 := d4
						snap37 := d5
						snap38 := d12
						snap39 := d13
						snap40 := d15
						snap41 := d16
						snap42 := d17
						snap43 := d29
						snap44 := d30
						snap45 := d31
						snap46 := d32
						alloc47 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc47)
						d1 = snap33
						d2 = snap34
						d3 = snap35
						d4 = snap36
						d5 = snap37
						d12 = snap38
						d13 = snap39
						d15 = snap40
						d16 = snap41
						d17 = snap42
						d29 = snap43
						d30 = snap44
						d31 = snap45
						d32 = snap46
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d30)
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
					ctx.ReclaimUntrackedRegs()
					d48 = args[1]
					d48.ID = 0
					d50 = d48
					d50.ID = 0
					d49 = ctx.EmitTagEqualsBorrowed(&d50, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d48)
					d51 = d49
					ctx.EnsureDesc(&d51)
					if d51.Loc != LocImm && d51.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d51.Loc == LocImm {
						if d51.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d51.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap52 := d1
						snap53 := d2
						snap54 := d3
						snap55 := d4
						snap56 := d5
						snap57 := d12
						snap58 := d13
						snap59 := d15
						snap60 := d16
						snap61 := d17
						snap62 := d29
						snap63 := d30
						snap64 := d31
						snap65 := d32
						snap66 := d48
						snap67 := d49
						snap68 := d50
						snap69 := d51
						alloc70 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc70)
						d1 = snap52
						d2 = snap53
						d3 = snap54
						d4 = snap55
						d5 = snap56
						d12 = snap57
						d13 = snap58
						d15 = snap59
						d16 = snap60
						d17 = snap61
						d29 = snap62
						d30 = snap63
						d31 = snap64
						d32 = snap65
						d48 = snap66
						d49 = snap67
						d50 = snap68
						d51 = snap69
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d49)
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
					ctx.ReclaimUntrackedRegs()
					d71 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
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
					ctx.ReclaimUntrackedRegs()
					d72 = args[1]
					d72.ID = 0
					if d72.Loc == LocImm {
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d72.Imm.Int())}
					} else if d72.Type == tagInt && d72.Loc == LocRegPair {
						ctx.FreeReg(d72.Reg)
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg2}
						ctx.BindReg(d72.Reg2, &d73)
						ctx.BindReg(d72.Reg2, &d73)
					} else if d72.Type == tagInt && d72.Loc == LocReg {
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg}
						ctx.BindReg(d72.Reg, &d73)
						ctx.BindReg(d72.Reg, &d73)
					} else {
						d73 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d72}, 1)
						d73.Type = tagInt
						ctx.BindReg(d73.Reg, &d73)
					}
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.StabilizeDescForControlFlow(&d73)
					d75 = args[2]
					d75.ID = 0
					d77 = d75
					ctx.SyncDesc(&d77)
					if d77.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d77.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d77.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d77 = tmpScalar
					}
					d77 = JITPrepareScmerGoArg(ctx, d77)
					if d77.Loc != LocRegPair && d77.Loc != LocStackPair && d77.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d76 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d77}, 2)
					ctx.FreeDesc(&d75)
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d76.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d76.Imm)
						ptrWord, _ := d76.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d76.Imm.String())))
						d76 = tmpPair
					} else if d76.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d76.Type, Reg: ctx.AllocRegExcept(d76.Reg), Reg2: ctx.AllocRegExcept(d76.Reg)}
						switch d76.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d76)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d76)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d76)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d76)
						d76 = tmpPair
					}
					if d76.Loc != LocRegPair && d76.Loc != LocStackPair && d76.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d76)
					d78 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d76}, 2)
					d78.NoHeapPointer = false
					ctx.BindReg(d78.Reg, &d78)
					ctx.BindReg(d78.Reg2, &d78)
					ctx.StabilizeDescForControlFlow(&d78)
					ctx.EnsureDesc(&d78)
					d79 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					if d79.Loc == LocImm {
						ctx.TrackImm(d79.Imm)
						ptrWord, _ := d79.Imm.RawWords()
						d80 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d80.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d80.Reg2, uint64(len(d79.Imm.String())))
						ctx.BindReg(d80.Reg, &d80)
						ctx.BindReg(d80.Reg2, &d80)
					} else {
						d80 = d79
					}
					d81 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d80}, 1)
					ctx.EmitAndRegImm32(d81.Reg, 1)
					d81.Type = tagBool
					ctx.BindReg(d81.Reg, &d81)
					d82 = d81
					ctx.EnsureDesc(&d82)
					if d82.Loc != LocImm && d82.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d82.Loc == LocImm {
						if d82.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[10].Render()
					}
					ctx.EmitCmpRegImm32(d82.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
					}
					ctx.FlushRegisterMoves()
					if !bbs[10].Rendered {
						snap83 := d1
						snap84 := d2
						snap85 := d3
						snap86 := d4
						snap87 := d5
						snap88 := d12
						snap89 := d13
						snap90 := d15
						snap91 := d16
						snap92 := d17
						snap93 := d29
						snap94 := d30
						snap95 := d31
						snap96 := d32
						snap97 := d48
						snap98 := d49
						snap99 := d50
						snap100 := d51
						snap101 := d71
						snap102 := d72
						snap103 := d73
						snap104 := d74
						snap105 := d75
						snap106 := d76
						snap107 := d77
						snap108 := d78
						snap109 := d79
						snap110 := d80
						snap111 := d81
						snap112 := d82
						alloc113 := ctx.SnapshotAllocState()
						bbs[10].Render()
						ctx.RestoreAllocState(alloc113)
						d1 = snap83
						d2 = snap84
						d3 = snap85
						d4 = snap86
						d5 = snap87
						d12 = snap88
						d13 = snap89
						d15 = snap90
						d16 = snap91
						d17 = snap92
						d29 = snap93
						d30 = snap94
						d31 = snap95
						d32 = snap96
						d48 = snap97
						d49 = snap98
						d50 = snap99
						d51 = snap100
						d71 = snap101
						d72 = snap102
						d73 = snap103
						d74 = snap104
						d75 = snap105
						d76 = snap106
						d77 = snap107
						d78 = snap108
						d79 = snap109
						d80 = snap110
						d81 = snap111
						d82 = snap112
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d81)
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
					ctx.ReclaimUntrackedRegs()
					d1 = JITPrepareGoSliceArg(ctx, d1)
					if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d1)
					d114 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d1}, 1)
					d114.NoHeapPointer = true
					ctx.BindReg(d114.Reg, &d114)
					ctx.FreeDesc(&d1)
					if d114.Loc == LocRegPair || d114.Loc == LocStackPair || d114.Loc == LocRegTriple || d114.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d114)
					d115 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d114}, 2)
					d115.NoHeapPointer = false
					ctx.BindReg(d115.Reg, &d115)
					ctx.BindReg(d115.Reg2, &d115)
					ctx.FreeDesc(&d114)
					ctx.SyncDesc(&d115)
					if d115.Loc == LocRegPair || d115.Loc == LocStackPair || d115.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d115, &result)
						result.Type = d115.Type
					} else {
						switch d115.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d115)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d115)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d115)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d115, &result)
							result.Type = d115.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						d117 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() * 1000000000)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d73.Reg, 1000000000)
						d117 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d117)
					}
					if d117.Loc == LocReg && d73.Loc == LocReg && d117.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Add arg0)")
					}
					if d117.Loc == LocRegPair || d117.Loc == LocStackPair || d117.Loc == LocRegTriple || d117.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d117)
					d118 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Add), []JITValueDesc{d15, d117}, 3)
					d118.NoHeapPointer = false
					ctx.BindReg(d118.Reg, &d118)
					ctx.BindReg(d118.Reg2, &d118)
					ctx.BindReg(d118.Reg3, &d118)
					ctx.StabilizeDescForControlFlow(&d118)
					ctx.FreeDesc(&d117)
					ctx.SyncDesc(&d118)
					if d118.Loc == LocReg || d118.Loc == LocFPReg {
						ctx.ProtectReg(d118.Reg)
					} else if d118.Loc == LocRegPair {
						ctx.ProtectReg(d118.Reg)
						ctx.ProtectReg(d118.Reg2)
					}
					d119 = d118
					if d119.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d119)
					if d119.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d119, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d119.Loc == LocInputPair {
						ctx.EnsureDesc(&d119)
						ctx.EmitStoreScmerToStack(d119, int32(bbs[7].PhiBase)+int32(0))
					} else if d119.Loc == LocRegPair || d119.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d119, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d119)
						ctx.EmitStoreToStack(d119, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d118.Loc == LocReg || d118.Loc == LocFPReg {
						ctx.UnprotectReg(d118.Reg)
					} else if d118.Loc == LocRegPair {
						ctx.UnprotectReg(d118.Reg)
						ctx.UnprotectReg(d118.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						d121 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() * 60000000000)}
					} else {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegReg(scratch, d73.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, 60000000000)
						d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d121)
					}
					if d121.Loc == LocReg && d73.Loc == LocReg && d121.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Add arg0)")
					}
					if d121.Loc == LocRegPair || d121.Loc == LocStackPair || d121.Loc == LocRegTriple || d121.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d121)
					d122 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Add), []JITValueDesc{d15, d121}, 3)
					d122.NoHeapPointer = false
					ctx.BindReg(d122.Reg, &d122)
					ctx.BindReg(d122.Reg2, &d122)
					ctx.BindReg(d122.Reg3, &d122)
					ctx.StabilizeDescForControlFlow(&d122)
					ctx.FreeDesc(&d121)
					ctx.SyncDesc(&d122)
					if d122.Loc == LocReg || d122.Loc == LocFPReg {
						ctx.ProtectReg(d122.Reg)
					} else if d122.Loc == LocRegPair {
						ctx.ProtectReg(d122.Reg)
						ctx.ProtectReg(d122.Reg2)
					}
					d123 = d122
					if d123.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d123)
					if d123.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d123, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d123.Loc == LocInputPair {
						ctx.EnsureDesc(&d123)
						ctx.EmitStoreScmerToStack(d123, int32(bbs[7].PhiBase)+int32(0))
					} else if d123.Loc == LocRegPair || d123.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d123, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d123)
						ctx.EmitStoreToStack(d123, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d122.Loc == LocReg || d122.Loc == LocFPReg {
						ctx.UnprotectReg(d122.Reg)
					} else if d122.Loc == LocRegPair {
						ctx.UnprotectReg(d122.Reg)
						ctx.UnprotectReg(d122.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d124 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					if d124.Loc == LocImm {
						ctx.TrackImm(d124.Imm)
						ptrWord, _ := d124.Imm.RawWords()
						d125 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d125.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d125.Reg2, uint64(len(d124.Imm.String())))
						ctx.BindReg(d125.Reg, &d125)
						ctx.BindReg(d125.Reg2, &d125)
					} else {
						d125 = d124
					}
					d126 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d125}, 1)
					ctx.EmitAndRegImm32(d126.Reg, 1)
					d126.Type = tagBool
					ctx.BindReg(d126.Reg, &d126)
					d127 = d126
					ctx.EnsureDesc(&d127)
					if d127.Loc != LocImm && d127.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d127.Loc == LocImm {
						if d127.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitCmpRegImm32(d127.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap128 := d1
						snap129 := d2
						snap130 := d3
						snap131 := d4
						snap132 := d5
						snap133 := d12
						snap134 := d13
						snap135 := d15
						snap136 := d16
						snap137 := d17
						snap138 := d29
						snap139 := d30
						snap140 := d31
						snap141 := d32
						snap142 := d48
						snap143 := d49
						snap144 := d50
						snap145 := d51
						snap146 := d71
						snap147 := d72
						snap148 := d73
						snap149 := d74
						snap150 := d75
						snap151 := d76
						snap152 := d77
						snap153 := d78
						snap154 := d79
						snap155 := d80
						snap156 := d81
						snap157 := d82
						snap158 := d114
						snap159 := d115
						snap160 := d116
						snap161 := d117
						snap162 := d118
						snap163 := d119
						snap164 := d120
						snap165 := d121
						snap166 := d122
						snap167 := d123
						snap168 := d124
						snap169 := d125
						snap170 := d126
						snap171 := d127
						alloc172 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc172)
						d1 = snap128
						d2 = snap129
						d3 = snap130
						d4 = snap131
						d5 = snap132
						d12 = snap133
						d13 = snap134
						d15 = snap135
						d16 = snap136
						d17 = snap137
						d29 = snap138
						d30 = snap139
						d31 = snap140
						d32 = snap141
						d48 = snap142
						d49 = snap143
						d50 = snap144
						d51 = snap145
						d71 = snap146
						d72 = snap147
						d73 = snap148
						d74 = snap149
						d75 = snap150
						d76 = snap151
						d77 = snap152
						d78 = snap153
						d79 = snap154
						d80 = snap155
						d81 = snap156
						d82 = snap157
						d114 = snap158
						d115 = snap159
						d116 = snap160
						d117 = snap161
						d118 = snap162
						d119 = snap163
						d120 = snap164
						d121 = snap165
						d122 = snap166
						d123 = snap167
						d124 = snap168
						d125 = snap169
						d126 = snap170
						d127 = snap171
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					ctx.FreeDesc(&d126)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						d174 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() * 3600000000000)}
					} else {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegReg(scratch, d73.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, 3600000000000)
						d174 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d174)
					}
					if d174.Loc == LocReg && d73.Loc == LocReg && d174.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Add arg0)")
					}
					if d174.Loc == LocRegPair || d174.Loc == LocStackPair || d174.Loc == LocRegTriple || d174.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d174)
					d175 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Add), []JITValueDesc{d15, d174}, 3)
					d175.NoHeapPointer = false
					ctx.BindReg(d175.Reg, &d175)
					ctx.BindReg(d175.Reg2, &d175)
					ctx.BindReg(d175.Reg3, &d175)
					ctx.StabilizeDescForControlFlow(&d175)
					ctx.FreeDesc(&d174)
					ctx.SyncDesc(&d175)
					if d175.Loc == LocReg || d175.Loc == LocFPReg {
						ctx.ProtectReg(d175.Reg)
					} else if d175.Loc == LocRegPair {
						ctx.ProtectReg(d175.Reg)
						ctx.ProtectReg(d175.Reg2)
					}
					d176 = d175
					if d176.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d176)
					if d176.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d176, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d176.Loc == LocInputPair {
						ctx.EnsureDesc(&d176)
						ctx.EmitStoreScmerToStack(d176, int32(bbs[7].PhiBase)+int32(0))
					} else if d176.Loc == LocRegPair || d176.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d176, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d176)
						ctx.EmitStoreToStack(d176, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d175.Loc == LocReg || d175.Loc == LocFPReg {
						ctx.UnprotectReg(d175.Reg)
					} else if d175.Loc == LocRegPair {
						ctx.UnprotectReg(d175.Reg)
						ctx.UnprotectReg(d175.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d177 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					if d177.Loc == LocImm {
						ctx.TrackImm(d177.Imm)
						ptrWord, _ := d177.Imm.RawWords()
						d178 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d178.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d178.Reg2, uint64(len(d177.Imm.String())))
						ctx.BindReg(d178.Reg, &d178)
						ctx.BindReg(d178.Reg2, &d178)
					} else {
						d178 = d177
					}
					d179 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d178}, 1)
					ctx.EmitAndRegImm32(d179.Reg, 1)
					d179.Type = tagBool
					ctx.BindReg(d179.Reg, &d179)
					d180 = d179
					ctx.EnsureDesc(&d180)
					if d180.Loc != LocImm && d180.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d180.Loc == LocImm {
						if d180.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[14].Render()
					}
					ctx.EmitCmpRegImm32(d180.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
					}
					ctx.FlushRegisterMoves()
					if !bbs[14].Rendered {
						snap181 := d1
						snap182 := d2
						snap183 := d3
						snap184 := d4
						snap185 := d5
						snap186 := d12
						snap187 := d13
						snap188 := d15
						snap189 := d16
						snap190 := d17
						snap191 := d29
						snap192 := d30
						snap193 := d31
						snap194 := d32
						snap195 := d48
						snap196 := d49
						snap197 := d50
						snap198 := d51
						snap199 := d71
						snap200 := d72
						snap201 := d73
						snap202 := d74
						snap203 := d75
						snap204 := d76
						snap205 := d77
						snap206 := d78
						snap207 := d79
						snap208 := d80
						snap209 := d81
						snap210 := d82
						snap211 := d114
						snap212 := d115
						snap213 := d116
						snap214 := d117
						snap215 := d118
						snap216 := d119
						snap217 := d120
						snap218 := d121
						snap219 := d122
						snap220 := d123
						snap221 := d124
						snap222 := d125
						snap223 := d126
						snap224 := d127
						snap225 := d173
						snap226 := d174
						snap227 := d175
						snap228 := d176
						snap229 := d177
						snap230 := d178
						snap231 := d179
						snap232 := d180
						alloc233 := ctx.SnapshotAllocState()
						bbs[14].Render()
						ctx.RestoreAllocState(alloc233)
						d1 = snap181
						d2 = snap182
						d3 = snap183
						d4 = snap184
						d5 = snap185
						d12 = snap186
						d13 = snap187
						d15 = snap188
						d16 = snap189
						d17 = snap190
						d29 = snap191
						d30 = snap192
						d31 = snap193
						d32 = snap194
						d48 = snap195
						d49 = snap196
						d50 = snap197
						d51 = snap198
						d71 = snap199
						d72 = snap200
						d73 = snap201
						d74 = snap202
						d75 = snap203
						d76 = snap204
						d77 = snap205
						d78 = snap206
						d79 = snap207
						d80 = snap208
						d81 = snap209
						d82 = snap210
						d114 = snap211
						d115 = snap212
						d116 = snap213
						d117 = snap214
						d118 = snap215
						d119 = snap216
						d120 = snap217
						d121 = snap218
						d122 = snap219
						d123 = snap220
						d124 = snap221
						d125 = snap222
						d126 = snap223
						d127 = snap224
						d173 = snap225
						d174 = snap226
						d175 = snap227
						d176 = snap228
						d177 = snap229
						d178 = snap230
						d179 = snap231
						d180 = snap232
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					ctx.FreeDesc(&d179)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					d234 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d234.Loc == LocRegPair || d234.Loc == LocStackPair || d234.Loc == LocRegTriple || d234.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d235 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d235.Loc == LocRegPair || d235.Loc == LocStackPair || d235.Loc == LocRegTriple || d235.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d73.Loc == LocRegPair || d73.Loc == LocStackPair || d73.Loc == LocRegTriple || d73.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d234)
					ctx.SyncDesc(&d235)
					ctx.SyncDesc(&d73)
					d236 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d234, d235, d73}, 3)
					d236.NoHeapPointer = false
					ctx.BindReg(d236.Reg, &d236)
					ctx.BindReg(d236.Reg2, &d236)
					ctx.BindReg(d236.Reg3, &d236)
					ctx.FreeDesc(&d234)
					ctx.FreeDesc(&d235)
					ctx.StabilizeDescForControlFlow(&d236)
					ctx.SyncDesc(&d236)
					if d236.Loc == LocReg || d236.Loc == LocFPReg {
						ctx.ProtectReg(d236.Reg)
					} else if d236.Loc == LocRegPair {
						ctx.ProtectReg(d236.Reg)
						ctx.ProtectReg(d236.Reg2)
					}
					d237 = d236
					if d237.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d237)
					if d237.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d237, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d237.Loc == LocInputPair {
						ctx.EnsureDesc(&d237)
						ctx.EmitStoreScmerToStack(d237, int32(bbs[7].PhiBase)+int32(0))
					} else if d237.Loc == LocRegPair || d237.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d237, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d237)
						ctx.EmitStoreToStack(d237, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d236.Loc == LocReg || d236.Loc == LocFPReg {
						ctx.UnprotectReg(d236.Reg)
					} else if d236.Loc == LocRegPair {
						ctx.UnprotectReg(d236.Reg)
						ctx.UnprotectReg(d236.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d238 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
					if d238.Loc == LocImm {
						ctx.TrackImm(d238.Imm)
						ptrWord, _ := d238.Imm.RawWords()
						d239 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d239.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d239.Reg2, uint64(len(d238.Imm.String())))
						ctx.BindReg(d239.Reg, &d239)
						ctx.BindReg(d239.Reg2, &d239)
					} else {
						d239 = d238
					}
					d240 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d239}, 1)
					ctx.EmitAndRegImm32(d240.Reg, 1)
					d240.Type = tagBool
					ctx.BindReg(d240.Reg, &d240)
					d241 = d240
					ctx.EnsureDesc(&d241)
					if d241.Loc != LocImm && d241.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d241.Loc == LocImm {
						if d241.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[16].Render()
					}
					ctx.EmitCmpRegImm32(d241.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
					}
					ctx.FlushRegisterMoves()
					if !bbs[16].Rendered {
						snap242 := d1
						snap243 := d2
						snap244 := d3
						snap245 := d4
						snap246 := d5
						snap247 := d12
						snap248 := d13
						snap249 := d15
						snap250 := d16
						snap251 := d17
						snap252 := d29
						snap253 := d30
						snap254 := d31
						snap255 := d32
						snap256 := d48
						snap257 := d49
						snap258 := d50
						snap259 := d51
						snap260 := d71
						snap261 := d72
						snap262 := d73
						snap263 := d74
						snap264 := d75
						snap265 := d76
						snap266 := d77
						snap267 := d78
						snap268 := d79
						snap269 := d80
						snap270 := d81
						snap271 := d82
						snap272 := d114
						snap273 := d115
						snap274 := d116
						snap275 := d117
						snap276 := d118
						snap277 := d119
						snap278 := d120
						snap279 := d121
						snap280 := d122
						snap281 := d123
						snap282 := d124
						snap283 := d125
						snap284 := d126
						snap285 := d127
						snap286 := d173
						snap287 := d174
						snap288 := d175
						snap289 := d176
						snap290 := d177
						snap291 := d178
						snap292 := d179
						snap293 := d180
						snap294 := d234
						snap295 := d235
						snap296 := d236
						snap297 := d237
						snap298 := d238
						snap299 := d239
						snap300 := d240
						snap301 := d241
						alloc302 := ctx.SnapshotAllocState()
						bbs[16].Render()
						ctx.RestoreAllocState(alloc302)
						d1 = snap242
						d2 = snap243
						d3 = snap244
						d4 = snap245
						d5 = snap246
						d12 = snap247
						d13 = snap248
						d15 = snap249
						d16 = snap250
						d17 = snap251
						d29 = snap252
						d30 = snap253
						d31 = snap254
						d32 = snap255
						d48 = snap256
						d49 = snap257
						d50 = snap258
						d51 = snap259
						d71 = snap260
						d72 = snap261
						d73 = snap262
						d74 = snap263
						d75 = snap264
						d76 = snap265
						d77 = snap266
						d78 = snap267
						d79 = snap268
						d80 = snap269
						d81 = snap270
						d82 = snap271
						d114 = snap272
						d115 = snap273
						d116 = snap274
						d117 = snap275
						d118 = snap276
						d119 = snap277
						d120 = snap278
						d121 = snap279
						d122 = snap280
						d123 = snap281
						d124 = snap282
						d125 = snap283
						d126 = snap284
						d127 = snap285
						d173 = snap286
						d174 = snap287
						d175 = snap288
						d176 = snap289
						d177 = snap290
						d178 = snap291
						d179 = snap292
						d180 = snap293
						d234 = snap294
						d235 = snap295
						d236 = snap296
						d237 = snap297
						d238 = snap298
						d239 = snap299
						d240 = snap300
						d241 = snap301
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					ctx.FreeDesc(&d240)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						d303 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() * 7)}
					} else {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegReg(scratch, d73.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, 7)
						d303 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d303)
					}
					if d303.Loc == LocReg && d73.Loc == LocReg && d303.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					d304 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d304.Loc == LocRegPair || d304.Loc == LocStackPair || d304.Loc == LocRegTriple || d304.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d305 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d305.Loc == LocRegPair || d305.Loc == LocStackPair || d305.Loc == LocRegTriple || d305.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d303.Loc == LocRegPair || d303.Loc == LocStackPair || d303.Loc == LocRegTriple || d303.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d304)
					ctx.SyncDesc(&d305)
					ctx.SyncDesc(&d303)
					d306 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d304, d305, d303}, 3)
					d306.NoHeapPointer = false
					ctx.BindReg(d306.Reg, &d306)
					ctx.BindReg(d306.Reg2, &d306)
					ctx.BindReg(d306.Reg3, &d306)
					ctx.FreeDesc(&d304)
					ctx.FreeDesc(&d305)
					ctx.StabilizeDescForControlFlow(&d306)
					ctx.FreeDesc(&d303)
					ctx.SyncDesc(&d306)
					if d306.Loc == LocReg || d306.Loc == LocFPReg {
						ctx.ProtectReg(d306.Reg)
					} else if d306.Loc == LocRegPair {
						ctx.ProtectReg(d306.Reg)
						ctx.ProtectReg(d306.Reg2)
					}
					d307 = d306
					if d307.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d307)
					if d307.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d307, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d307.Loc == LocInputPair {
						ctx.EnsureDesc(&d307)
						ctx.EmitStoreScmerToStack(d307, int32(bbs[7].PhiBase)+int32(0))
					} else if d307.Loc == LocRegPair || d307.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d307, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d307)
						ctx.EmitStoreToStack(d307, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d306.Loc == LocReg || d306.Loc == LocFPReg {
						ctx.UnprotectReg(d306.Reg)
					} else if d306.Loc == LocRegPair {
						ctx.UnprotectReg(d306.Reg)
						ctx.UnprotectReg(d306.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d308 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					if d308.Loc == LocImm {
						ctx.TrackImm(d308.Imm)
						ptrWord, _ := d308.Imm.RawWords()
						d309 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d309.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d309.Reg2, uint64(len(d308.Imm.String())))
						ctx.BindReg(d309.Reg, &d309)
						ctx.BindReg(d309.Reg2, &d309)
					} else {
						d309 = d308
					}
					d310 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d309}, 1)
					ctx.EmitAndRegImm32(d310.Reg, 1)
					d310.Type = tagBool
					ctx.BindReg(d310.Reg, &d310)
					d311 = d310
					ctx.EnsureDesc(&d311)
					if d311.Loc != LocImm && d311.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d311.Loc == LocImm {
						if d311.Imm.Bool() {
							return bbs[15].Render()
						}
						return bbs[18].Render()
					}
					ctx.EmitCmpRegImm32(d311.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
					}
					ctx.FlushRegisterMoves()
					if !bbs[18].Rendered {
						snap312 := d1
						snap313 := d2
						snap314 := d3
						snap315 := d4
						snap316 := d5
						snap317 := d12
						snap318 := d13
						snap319 := d15
						snap320 := d16
						snap321 := d17
						snap322 := d29
						snap323 := d30
						snap324 := d31
						snap325 := d32
						snap326 := d48
						snap327 := d49
						snap328 := d50
						snap329 := d51
						snap330 := d71
						snap331 := d72
						snap332 := d73
						snap333 := d74
						snap334 := d75
						snap335 := d76
						snap336 := d77
						snap337 := d78
						snap338 := d79
						snap339 := d80
						snap340 := d81
						snap341 := d82
						snap342 := d114
						snap343 := d115
						snap344 := d116
						snap345 := d117
						snap346 := d118
						snap347 := d119
						snap348 := d120
						snap349 := d121
						snap350 := d122
						snap351 := d123
						snap352 := d124
						snap353 := d125
						snap354 := d126
						snap355 := d127
						snap356 := d173
						snap357 := d174
						snap358 := d175
						snap359 := d176
						snap360 := d177
						snap361 := d178
						snap362 := d179
						snap363 := d180
						snap364 := d234
						snap365 := d235
						snap366 := d236
						snap367 := d237
						snap368 := d238
						snap369 := d239
						snap370 := d240
						snap371 := d241
						snap372 := d303
						snap373 := d304
						snap374 := d305
						snap375 := d306
						snap376 := d307
						snap377 := d308
						snap378 := d309
						snap379 := d310
						snap380 := d311
						alloc381 := ctx.SnapshotAllocState()
						bbs[18].Render()
						ctx.RestoreAllocState(alloc381)
						d1 = snap312
						d2 = snap313
						d3 = snap314
						d4 = snap315
						d5 = snap316
						d12 = snap317
						d13 = snap318
						d15 = snap319
						d16 = snap320
						d17 = snap321
						d29 = snap322
						d30 = snap323
						d31 = snap324
						d32 = snap325
						d48 = snap326
						d49 = snap327
						d50 = snap328
						d51 = snap329
						d71 = snap330
						d72 = snap331
						d73 = snap332
						d74 = snap333
						d75 = snap334
						d76 = snap335
						d77 = snap336
						d78 = snap337
						d79 = snap338
						d80 = snap339
						d81 = snap340
						d82 = snap341
						d114 = snap342
						d115 = snap343
						d116 = snap344
						d117 = snap345
						d118 = snap346
						d119 = snap347
						d120 = snap348
						d121 = snap349
						d122 = snap350
						d123 = snap351
						d124 = snap352
						d125 = snap353
						d126 = snap354
						d127 = snap355
						d173 = snap356
						d174 = snap357
						d175 = snap358
						d176 = snap359
						d177 = snap360
						d178 = snap361
						d179 = snap362
						d180 = snap363
						d234 = snap364
						d235 = snap365
						d236 = snap366
						d237 = snap367
						d238 = snap368
						d239 = snap369
						d240 = snap370
						d241 = snap371
						d303 = snap372
						d304 = snap373
						d305 = snap374
						d306 = snap375
						d307 = snap376
						d308 = snap377
						d309 = snap378
						d310 = snap379
						d311 = snap380
					}
					if !bbs[15].Rendered {
						return bbs[15].Render()
					}
					return result
					ctx.FreeDesc(&d310)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					d382 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d382.Loc == LocRegPair || d382.Loc == LocStackPair || d382.Loc == LocRegTriple || d382.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d73.Loc == LocRegPair || d73.Loc == LocStackPair || d73.Loc == LocRegTriple || d73.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d383 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d383.Loc == LocRegPair || d383.Loc == LocStackPair || d383.Loc == LocRegTriple || d383.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d382)
					ctx.SyncDesc(&d73)
					ctx.SyncDesc(&d383)
					d384 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d382, d73, d383}, 3)
					d384.NoHeapPointer = false
					ctx.BindReg(d384.Reg, &d384)
					ctx.BindReg(d384.Reg2, &d384)
					ctx.BindReg(d384.Reg3, &d384)
					ctx.FreeDesc(&d382)
					ctx.FreeDesc(&d383)
					ctx.StabilizeDescForControlFlow(&d384)
					ctx.SyncDesc(&d384)
					if d384.Loc == LocReg || d384.Loc == LocFPReg {
						ctx.ProtectReg(d384.Reg)
					} else if d384.Loc == LocRegPair {
						ctx.ProtectReg(d384.Reg)
						ctx.ProtectReg(d384.Reg2)
					}
					d385 = d384
					if d385.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d385)
					if d385.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d385, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d385.Loc == LocInputPair {
						ctx.EnsureDesc(&d385)
						ctx.EmitStoreScmerToStack(d385, int32(bbs[7].PhiBase)+int32(0))
					} else if d385.Loc == LocRegPair || d385.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d385, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d385)
						ctx.EmitStoreToStack(d385, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d384.Loc == LocReg || d384.Loc == LocFPReg {
						ctx.UnprotectReg(d384.Reg)
					} else if d384.Loc == LocRegPair {
						ctx.UnprotectReg(d384.Reg)
						ctx.UnprotectReg(d384.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d386 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					if d386.Loc == LocImm {
						ctx.TrackImm(d386.Imm)
						ptrWord, _ := d386.Imm.RawWords()
						d387 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d387.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d387.Reg2, uint64(len(d386.Imm.String())))
						ctx.BindReg(d387.Reg, &d387)
						ctx.BindReg(d387.Reg2, &d387)
					} else {
						d387 = d386
					}
					d388 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d387}, 1)
					ctx.EmitAndRegImm32(d388.Reg, 1)
					d388.Type = tagBool
					ctx.BindReg(d388.Reg, &d388)
					d389 = d388
					ctx.EnsureDesc(&d389)
					if d389.Loc != LocImm && d389.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d389.Loc == LocImm {
						if d389.Imm.Bool() {
							return bbs[17].Render()
						}
						return bbs[20].Render()
					}
					ctx.EmitCmpRegImm32(d389.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
					}
					ctx.FlushRegisterMoves()
					if !bbs[20].Rendered {
						snap390 := d1
						snap391 := d2
						snap392 := d3
						snap393 := d4
						snap394 := d5
						snap395 := d12
						snap396 := d13
						snap397 := d15
						snap398 := d16
						snap399 := d17
						snap400 := d29
						snap401 := d30
						snap402 := d31
						snap403 := d32
						snap404 := d48
						snap405 := d49
						snap406 := d50
						snap407 := d51
						snap408 := d71
						snap409 := d72
						snap410 := d73
						snap411 := d74
						snap412 := d75
						snap413 := d76
						snap414 := d77
						snap415 := d78
						snap416 := d79
						snap417 := d80
						snap418 := d81
						snap419 := d82
						snap420 := d114
						snap421 := d115
						snap422 := d116
						snap423 := d117
						snap424 := d118
						snap425 := d119
						snap426 := d120
						snap427 := d121
						snap428 := d122
						snap429 := d123
						snap430 := d124
						snap431 := d125
						snap432 := d126
						snap433 := d127
						snap434 := d173
						snap435 := d174
						snap436 := d175
						snap437 := d176
						snap438 := d177
						snap439 := d178
						snap440 := d179
						snap441 := d180
						snap442 := d234
						snap443 := d235
						snap444 := d236
						snap445 := d237
						snap446 := d238
						snap447 := d239
						snap448 := d240
						snap449 := d241
						snap450 := d303
						snap451 := d304
						snap452 := d305
						snap453 := d306
						snap454 := d307
						snap455 := d308
						snap456 := d309
						snap457 := d310
						snap458 := d311
						snap459 := d382
						snap460 := d383
						snap461 := d384
						snap462 := d385
						snap463 := d386
						snap464 := d387
						snap465 := d388
						snap466 := d389
						alloc467 := ctx.SnapshotAllocState()
						bbs[20].Render()
						ctx.RestoreAllocState(alloc467)
						d1 = snap390
						d2 = snap391
						d3 = snap392
						d4 = snap393
						d5 = snap394
						d12 = snap395
						d13 = snap396
						d15 = snap397
						d16 = snap398
						d17 = snap399
						d29 = snap400
						d30 = snap401
						d31 = snap402
						d32 = snap403
						d48 = snap404
						d49 = snap405
						d50 = snap406
						d51 = snap407
						d71 = snap408
						d72 = snap409
						d73 = snap410
						d74 = snap411
						d75 = snap412
						d76 = snap413
						d77 = snap414
						d78 = snap415
						d79 = snap416
						d80 = snap417
						d81 = snap418
						d82 = snap419
						d114 = snap420
						d115 = snap421
						d116 = snap422
						d117 = snap423
						d118 = snap424
						d119 = snap425
						d120 = snap426
						d121 = snap427
						d122 = snap428
						d123 = snap429
						d124 = snap430
						d125 = snap431
						d126 = snap432
						d127 = snap433
						d173 = snap434
						d174 = snap435
						d175 = snap436
						d176 = snap437
						d177 = snap438
						d178 = snap439
						d179 = snap440
						d180 = snap441
						d234 = snap442
						d235 = snap443
						d236 = snap444
						d237 = snap445
						d238 = snap446
						d239 = snap447
						d240 = snap448
						d241 = snap449
						d303 = snap450
						d304 = snap451
						d305 = snap452
						d306 = snap453
						d307 = snap454
						d308 = snap455
						d309 = snap456
						d310 = snap457
						d311 = snap458
						d382 = snap459
						d383 = snap460
						d384 = snap461
						d385 = snap462
						d386 = snap463
						d387 = snap464
						d388 = snap465
						d389 = snap466
					}
					if !bbs[17].Rendered {
						return bbs[17].Render()
					}
					return result
					ctx.FreeDesc(&d388)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					if d73.Loc == LocRegPair || d73.Loc == LocStackPair || d73.Loc == LocRegTriple || d73.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d468 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d468.Loc == LocRegPair || d468.Loc == LocStackPair || d468.Loc == LocRegTriple || d468.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d469 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d469.Loc == LocRegPair || d469.Loc == LocStackPair || d469.Loc == LocRegTriple || d469.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d73)
					ctx.SyncDesc(&d468)
					ctx.SyncDesc(&d469)
					d470 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d73, d468, d469}, 3)
					d470.NoHeapPointer = false
					ctx.BindReg(d470.Reg, &d470)
					ctx.BindReg(d470.Reg2, &d470)
					ctx.BindReg(d470.Reg3, &d470)
					ctx.FreeDesc(&d468)
					ctx.FreeDesc(&d469)
					ctx.StabilizeDescForControlFlow(&d470)
					ctx.FreeDesc(&d73)
					ctx.SyncDesc(&d470)
					if d470.Loc == LocReg || d470.Loc == LocFPReg {
						ctx.ProtectReg(d470.Reg)
					} else if d470.Loc == LocRegPair {
						ctx.ProtectReg(d470.Reg)
						ctx.ProtectReg(d470.Reg2)
					}
					d471 = d470
					if d471.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d471)
					if d471.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d471, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d471.Loc == LocInputPair {
						ctx.EnsureDesc(&d471)
						ctx.EmitStoreScmerToStack(d471, int32(bbs[7].PhiBase)+int32(0))
					} else if d471.Loc == LocRegPair || d471.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d471, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d471)
						ctx.EmitStoreToStack(d471, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d470.Loc == LocReg || d470.Loc == LocFPReg {
						ctx.UnprotectReg(d470.Reg)
					} else if d470.Loc == LocRegPair {
						ctx.UnprotectReg(d470.Reg)
						ctx.UnprotectReg(d470.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d472 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					if d472.Loc == LocImm {
						ctx.TrackImm(d472.Imm)
						ptrWord, _ := d472.Imm.RawWords()
						d473 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d473.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d473.Reg2, uint64(len(d472.Imm.String())))
						ctx.BindReg(d473.Reg, &d473)
						ctx.BindReg(d473.Reg2, &d473)
					} else {
						d473 = d472
					}
					d474 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d473}, 1)
					ctx.EmitAndRegImm32(d474.Reg, 1)
					d474.Type = tagBool
					ctx.BindReg(d474.Reg, &d474)
					d475 = d474
					ctx.EnsureDesc(&d475)
					if d475.Loc != LocImm && d475.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d475.Loc == LocImm {
						if d475.Imm.Bool() {
							return bbs[19].Render()
						}
						return bbs[21].Render()
					}
					ctx.EmitCmpRegImm32(d475.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					if bbs[21].Rendered {
						ctx.EmitJmp(lbl22)
					}
					ctx.FlushRegisterMoves()
					if !bbs[21].Rendered {
						snap476 := d1
						snap477 := d2
						snap478 := d3
						snap479 := d4
						snap480 := d5
						snap481 := d12
						snap482 := d13
						snap483 := d15
						snap484 := d16
						snap485 := d17
						snap486 := d29
						snap487 := d30
						snap488 := d31
						snap489 := d32
						snap490 := d48
						snap491 := d49
						snap492 := d50
						snap493 := d51
						snap494 := d71
						snap495 := d72
						snap496 := d73
						snap497 := d74
						snap498 := d75
						snap499 := d76
						snap500 := d77
						snap501 := d78
						snap502 := d79
						snap503 := d80
						snap504 := d81
						snap505 := d82
						snap506 := d114
						snap507 := d115
						snap508 := d116
						snap509 := d117
						snap510 := d118
						snap511 := d119
						snap512 := d120
						snap513 := d121
						snap514 := d122
						snap515 := d123
						snap516 := d124
						snap517 := d125
						snap518 := d126
						snap519 := d127
						snap520 := d173
						snap521 := d174
						snap522 := d175
						snap523 := d176
						snap524 := d177
						snap525 := d178
						snap526 := d179
						snap527 := d180
						snap528 := d234
						snap529 := d235
						snap530 := d236
						snap531 := d237
						snap532 := d238
						snap533 := d239
						snap534 := d240
						snap535 := d241
						snap536 := d303
						snap537 := d304
						snap538 := d305
						snap539 := d306
						snap540 := d307
						snap541 := d308
						snap542 := d309
						snap543 := d310
						snap544 := d311
						snap545 := d382
						snap546 := d383
						snap547 := d384
						snap548 := d385
						snap549 := d386
						snap550 := d387
						snap551 := d388
						snap552 := d389
						snap553 := d468
						snap554 := d469
						snap555 := d470
						snap556 := d471
						snap557 := d472
						snap558 := d473
						snap559 := d474
						snap560 := d475
						alloc561 := ctx.SnapshotAllocState()
						bbs[21].Render()
						ctx.RestoreAllocState(alloc561)
						d1 = snap476
						d2 = snap477
						d3 = snap478
						d4 = snap479
						d5 = snap480
						d12 = snap481
						d13 = snap482
						d15 = snap483
						d16 = snap484
						d17 = snap485
						d29 = snap486
						d30 = snap487
						d31 = snap488
						d32 = snap489
						d48 = snap490
						d49 = snap491
						d50 = snap492
						d51 = snap493
						d71 = snap494
						d72 = snap495
						d73 = snap496
						d74 = snap497
						d75 = snap498
						d76 = snap499
						d77 = snap500
						d78 = snap501
						d79 = snap502
						d80 = snap503
						d81 = snap504
						d82 = snap505
						d114 = snap506
						d115 = snap507
						d116 = snap508
						d117 = snap509
						d118 = snap510
						d119 = snap511
						d120 = snap512
						d121 = snap513
						d122 = snap514
						d123 = snap515
						d124 = snap516
						d125 = snap517
						d126 = snap518
						d127 = snap519
						d173 = snap520
						d174 = snap521
						d175 = snap522
						d176 = snap523
						d177 = snap524
						d178 = snap525
						d179 = snap526
						d180 = snap527
						d234 = snap528
						d235 = snap529
						d236 = snap530
						d237 = snap531
						d238 = snap532
						d239 = snap533
						d240 = snap534
						d241 = snap535
						d303 = snap536
						d304 = snap537
						d305 = snap538
						d306 = snap539
						d307 = snap540
						d308 = snap541
						d309 = snap542
						d310 = snap543
						d311 = snap544
						d382 = snap545
						d383 = snap546
						d384 = snap547
						d385 = snap548
						d386 = snap549
						d387 = snap550
						d388 = snap551
						d389 = snap552
						d468 = snap553
						d469 = snap554
						d470 = snap555
						d471 = snap556
						d472 = snap557
						d473 = snap558
						d474 = snap559
						d475 = snap560
					}
					if !bbs[19].Rendered {
						return bbs[19].Render()
					}
					return result
					ctx.FreeDesc(&d474)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["date_add"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
			JITInlineCost:  72,
		},
	})

	// DATE_SUB(expr, amount, unit)
	Declare(&Globalenv, &Declaration{
		Name: "date_sub",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			amount := int(a[1].Int())
			unit := strings.ToUpper(a[2].String())
			switch unit {
			case "SECOND":
				t = t.Add(-time.Duration(amount) * time.Second)
			case "MINUTE":
				t = t.Add(-time.Duration(amount) * time.Minute)
			case "HOUR":
				t = t.Add(-time.Duration(amount) * time.Hour)
			case "DAY":
				t = t.AddDate(0, 0, -amount)
			case "WEEK":
				t = t.AddDate(0, 0, -amount*7)
			case "MONTH":
				t = t.AddDate(0, -amount, 0)
			case "YEAR":
				t = t.AddDate(-amount, 0, 0)
			default:
				panic("unknown DATE_SUB unit: " + unit)
			}
			return NewDate(t.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "subtracts an interval from a date value",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date value"}, &TypeDescriptor{Kind: "int", Label: "amount", Description: "interval amount"}, &TypeDescriptor{Kind: "string", Label: "unit", Description: "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["date_sub"]
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
				var d32 JITValueDesc
				_ = d32
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
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
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
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
				var d177 JITValueDesc
				_ = d177
				var d178 JITValueDesc
				_ = d178
				var d179 JITValueDesc
				_ = d179
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d183 JITValueDesc
				_ = d183
				var d184 JITValueDesc
				_ = d184
				var d185 JITValueDesc
				_ = d185
				var d242 JITValueDesc
				_ = d242
				var d243 JITValueDesc
				_ = d243
				var d244 JITValueDesc
				_ = d244
				var d245 JITValueDesc
				_ = d245
				var d246 JITValueDesc
				_ = d246
				var d247 JITValueDesc
				_ = d247
				var d248 JITValueDesc
				_ = d248
				var d249 JITValueDesc
				_ = d249
				var d250 JITValueDesc
				_ = d250
				var d316 JITValueDesc
				_ = d316
				var d317 JITValueDesc
				_ = d317
				var d318 JITValueDesc
				_ = d318
				var d319 JITValueDesc
				_ = d319
				var d320 JITValueDesc
				_ = d320
				var d321 JITValueDesc
				_ = d321
				var d322 JITValueDesc
				_ = d322
				var d323 JITValueDesc
				_ = d323
				var d324 JITValueDesc
				_ = d324
				var d325 JITValueDesc
				_ = d325
				var d401 JITValueDesc
				_ = d401
				var d402 JITValueDesc
				_ = d402
				var d403 JITValueDesc
				_ = d403
				var d404 JITValueDesc
				_ = d404
				var d405 JITValueDesc
				_ = d405
				var d406 JITValueDesc
				_ = d406
				var d407 JITValueDesc
				_ = d407
				var d408 JITValueDesc
				_ = d408
				var d409 JITValueDesc
				_ = d409
				var d494 JITValueDesc
				_ = d494
				var d495 JITValueDesc
				_ = d495
				var d496 JITValueDesc
				_ = d496
				var d497 JITValueDesc
				_ = d497
				var d498 JITValueDesc
				_ = d498
				var d499 JITValueDesc
				_ = d499
				var d500 JITValueDesc
				_ = d500
				var d501 JITValueDesc
				_ = d501
				var d502 JITValueDesc
				_ = d502
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
				var bbs [22]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
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
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						snap10 := d5
						alloc11 := ctx.SnapshotAllocState()
						bbs[4].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					ctx.SyncDesc(&d13)
					callResults14 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d13}, []uint8{3, 1}, []uint8{4, 0})
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
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
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
						bbs[5].Render()
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
					if !bbs[6].Rendered {
						return bbs[6].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d29 = args[2]
					d29.ID = 0
					d31 = d29
					d31.ID = 0
					d30 = ctx.EmitTagEqualsBorrowed(&d31, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d29)
					d32 = d30
					ctx.EnsureDesc(&d32)
					if d32.Loc != LocImm && d32.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d32.Loc == LocImm {
						if d32.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d32.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap33 := d1
						snap34 := d2
						snap35 := d3
						snap36 := d4
						snap37 := d5
						snap38 := d12
						snap39 := d13
						snap40 := d15
						snap41 := d16
						snap42 := d17
						snap43 := d29
						snap44 := d30
						snap45 := d31
						snap46 := d32
						alloc47 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc47)
						d1 = snap33
						d2 = snap34
						d3 = snap35
						d4 = snap36
						d5 = snap37
						d12 = snap38
						d13 = snap39
						d15 = snap40
						d16 = snap41
						d17 = snap42
						d29 = snap43
						d30 = snap44
						d31 = snap45
						d32 = snap46
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d30)
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
					ctx.ReclaimUntrackedRegs()
					d48 = args[1]
					d48.ID = 0
					d50 = d48
					d50.ID = 0
					d49 = ctx.EmitTagEqualsBorrowed(&d50, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d48)
					d51 = d49
					ctx.EnsureDesc(&d51)
					if d51.Loc != LocImm && d51.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d51.Loc == LocImm {
						if d51.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d51.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap52 := d1
						snap53 := d2
						snap54 := d3
						snap55 := d4
						snap56 := d5
						snap57 := d12
						snap58 := d13
						snap59 := d15
						snap60 := d16
						snap61 := d17
						snap62 := d29
						snap63 := d30
						snap64 := d31
						snap65 := d32
						snap66 := d48
						snap67 := d49
						snap68 := d50
						snap69 := d51
						alloc70 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc70)
						d1 = snap52
						d2 = snap53
						d3 = snap54
						d4 = snap55
						d5 = snap56
						d12 = snap57
						d13 = snap58
						d15 = snap59
						d16 = snap60
						d17 = snap61
						d29 = snap62
						d30 = snap63
						d31 = snap64
						d32 = snap65
						d48 = snap66
						d49 = snap67
						d50 = snap68
						d51 = snap69
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d49)
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
					ctx.ReclaimUntrackedRegs()
					d71 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
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
					ctx.ReclaimUntrackedRegs()
					d72 = args[1]
					d72.ID = 0
					if d72.Loc == LocImm {
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d72.Imm.Int())}
					} else if d72.Type == tagInt && d72.Loc == LocRegPair {
						ctx.FreeReg(d72.Reg)
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg2}
						ctx.BindReg(d72.Reg2, &d73)
						ctx.BindReg(d72.Reg2, &d73)
					} else if d72.Type == tagInt && d72.Loc == LocReg {
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg}
						ctx.BindReg(d72.Reg, &d73)
						ctx.BindReg(d72.Reg, &d73)
					} else {
						d73 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d72}, 1)
						d73.Type = tagInt
						ctx.BindReg(d73.Reg, &d73)
					}
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.StabilizeDescForControlFlow(&d73)
					d75 = args[2]
					d75.ID = 0
					d77 = d75
					ctx.SyncDesc(&d77)
					if d77.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d77.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d77.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d77 = tmpScalar
					}
					d77 = JITPrepareScmerGoArg(ctx, d77)
					if d77.Loc != LocRegPair && d77.Loc != LocStackPair && d77.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d76 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d77}, 2)
					ctx.FreeDesc(&d75)
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d76.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d76.Imm)
						ptrWord, _ := d76.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d76.Imm.String())))
						d76 = tmpPair
					} else if d76.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d76.Type, Reg: ctx.AllocRegExcept(d76.Reg), Reg2: ctx.AllocRegExcept(d76.Reg)}
						switch d76.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d76)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d76)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d76)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d76)
						d76 = tmpPair
					}
					if d76.Loc != LocRegPair && d76.Loc != LocStackPair && d76.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d76)
					d78 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d76}, 2)
					d78.NoHeapPointer = false
					ctx.BindReg(d78.Reg, &d78)
					ctx.BindReg(d78.Reg2, &d78)
					ctx.StabilizeDescForControlFlow(&d78)
					ctx.EnsureDesc(&d78)
					d79 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					if d79.Loc == LocImm {
						ctx.TrackImm(d79.Imm)
						ptrWord, _ := d79.Imm.RawWords()
						d80 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d80.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d80.Reg2, uint64(len(d79.Imm.String())))
						ctx.BindReg(d80.Reg, &d80)
						ctx.BindReg(d80.Reg2, &d80)
					} else {
						d80 = d79
					}
					d81 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d80}, 1)
					ctx.EmitAndRegImm32(d81.Reg, 1)
					d81.Type = tagBool
					ctx.BindReg(d81.Reg, &d81)
					d82 = d81
					ctx.EnsureDesc(&d82)
					if d82.Loc != LocImm && d82.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d82.Loc == LocImm {
						if d82.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[10].Render()
					}
					ctx.EmitCmpRegImm32(d82.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
					}
					ctx.FlushRegisterMoves()
					if !bbs[10].Rendered {
						snap83 := d1
						snap84 := d2
						snap85 := d3
						snap86 := d4
						snap87 := d5
						snap88 := d12
						snap89 := d13
						snap90 := d15
						snap91 := d16
						snap92 := d17
						snap93 := d29
						snap94 := d30
						snap95 := d31
						snap96 := d32
						snap97 := d48
						snap98 := d49
						snap99 := d50
						snap100 := d51
						snap101 := d71
						snap102 := d72
						snap103 := d73
						snap104 := d74
						snap105 := d75
						snap106 := d76
						snap107 := d77
						snap108 := d78
						snap109 := d79
						snap110 := d80
						snap111 := d81
						snap112 := d82
						alloc113 := ctx.SnapshotAllocState()
						bbs[10].Render()
						ctx.RestoreAllocState(alloc113)
						d1 = snap83
						d2 = snap84
						d3 = snap85
						d4 = snap86
						d5 = snap87
						d12 = snap88
						d13 = snap89
						d15 = snap90
						d16 = snap91
						d17 = snap92
						d29 = snap93
						d30 = snap94
						d31 = snap95
						d32 = snap96
						d48 = snap97
						d49 = snap98
						d50 = snap99
						d51 = snap100
						d71 = snap101
						d72 = snap102
						d73 = snap103
						d74 = snap104
						d75 = snap105
						d76 = snap106
						d77 = snap107
						d78 = snap108
						d79 = snap109
						d80 = snap110
						d81 = snap111
						d82 = snap112
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d81)
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
					ctx.ReclaimUntrackedRegs()
					d1 = JITPrepareGoSliceArg(ctx, d1)
					if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d1)
					d114 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d1}, 1)
					d114.NoHeapPointer = true
					ctx.BindReg(d114.Reg, &d114)
					ctx.FreeDesc(&d1)
					if d114.Loc == LocRegPair || d114.Loc == LocStackPair || d114.Loc == LocRegTriple || d114.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d114)
					d115 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d114}, 2)
					d115.NoHeapPointer = false
					ctx.BindReg(d115.Reg, &d115)
					ctx.BindReg(d115.Reg2, &d115)
					ctx.FreeDesc(&d114)
					ctx.SyncDesc(&d115)
					if d115.Loc == LocRegPair || d115.Loc == LocStackPair || d115.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d115, &result)
						result.Type = d115.Type
					} else {
						switch d115.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d115)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d115)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d115)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d115, &result)
							result.Type = d115.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d117 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d117 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r0 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r0, 0)
							ctx.EmitSubFloat64(r0, d73.Reg)
							d117 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
							ctx.BindReg(r0, &d117)
						} else {
							r1 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r1, 0)
							ctx.EmitSubInt64(r1, d73.Reg)
							d117 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
							ctx.BindReg(r1, &d117)
						}
					}
					ctx.EnsureDesc(&d117)
					ctx.EnsureDesc(&d117)
					if d117.Loc == LocImm {
						d118 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d117.Imm.Int() * 1000000000)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d117.Reg, 1000000000)
						d118 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d117.Reg}
						ctx.BindReg(d117.Reg, &d118)
					}
					if d118.Loc == LocReg && d117.Loc == LocReg && d118.Reg == d117.Reg {
						ctx.TransferReg(d117.Reg)
						d117.Loc = LocNone
					}
					ctx.FreeDesc(&d117)
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Add arg0)")
					}
					if d118.Loc == LocRegPair || d118.Loc == LocStackPair || d118.Loc == LocRegTriple || d118.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d118)
					d119 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Add), []JITValueDesc{d15, d118}, 3)
					d119.NoHeapPointer = false
					ctx.BindReg(d119.Reg, &d119)
					ctx.BindReg(d119.Reg2, &d119)
					ctx.BindReg(d119.Reg3, &d119)
					ctx.StabilizeDescForControlFlow(&d119)
					ctx.FreeDesc(&d118)
					ctx.SyncDesc(&d119)
					if d119.Loc == LocReg || d119.Loc == LocFPReg {
						ctx.ProtectReg(d119.Reg)
					} else if d119.Loc == LocRegPair {
						ctx.ProtectReg(d119.Reg)
						ctx.ProtectReg(d119.Reg2)
					}
					d120 = d119
					if d120.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d120)
					if d120.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d120, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d120.Loc == LocInputPair {
						ctx.EnsureDesc(&d120)
						ctx.EmitStoreScmerToStack(d120, int32(bbs[7].PhiBase)+int32(0))
					} else if d120.Loc == LocRegPair || d120.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d120, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d120)
						ctx.EmitStoreToStack(d120, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d119.Loc == LocReg || d119.Loc == LocFPReg {
						ctx.UnprotectReg(d119.Reg)
					} else if d119.Loc == LocRegPair {
						ctx.UnprotectReg(d119.Reg)
						ctx.UnprotectReg(d119.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d122 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r2 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r2, 0)
							ctx.EmitSubFloat64(r2, d73.Reg)
							d122 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
							ctx.BindReg(r2, &d122)
						} else {
							r3 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r3, 0)
							ctx.EmitSubInt64(r3, d73.Reg)
							d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
							ctx.BindReg(r3, &d122)
						}
					}
					ctx.EnsureDesc(&d122)
					ctx.EnsureDesc(&d122)
					if d122.Loc == LocImm {
						d123 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d122.Imm.Int() * 60000000000)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d122.Reg, 60000000000)
						d123 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d122.Reg}
						ctx.BindReg(d122.Reg, &d123)
					}
					if d123.Loc == LocReg && d122.Loc == LocReg && d123.Reg == d122.Reg {
						ctx.TransferReg(d122.Reg)
						d122.Loc = LocNone
					}
					ctx.FreeDesc(&d122)
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Add arg0)")
					}
					if d123.Loc == LocRegPair || d123.Loc == LocStackPair || d123.Loc == LocRegTriple || d123.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d123)
					d124 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Add), []JITValueDesc{d15, d123}, 3)
					d124.NoHeapPointer = false
					ctx.BindReg(d124.Reg, &d124)
					ctx.BindReg(d124.Reg2, &d124)
					ctx.BindReg(d124.Reg3, &d124)
					ctx.StabilizeDescForControlFlow(&d124)
					ctx.FreeDesc(&d123)
					ctx.SyncDesc(&d124)
					if d124.Loc == LocReg || d124.Loc == LocFPReg {
						ctx.ProtectReg(d124.Reg)
					} else if d124.Loc == LocRegPair {
						ctx.ProtectReg(d124.Reg)
						ctx.ProtectReg(d124.Reg2)
					}
					d125 = d124
					if d125.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d125)
					if d125.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d125, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d125.Loc == LocInputPair {
						ctx.EnsureDesc(&d125)
						ctx.EmitStoreScmerToStack(d125, int32(bbs[7].PhiBase)+int32(0))
					} else if d125.Loc == LocRegPair || d125.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d125, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d125)
						ctx.EmitStoreToStack(d125, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d124.Loc == LocReg || d124.Loc == LocFPReg {
						ctx.UnprotectReg(d124.Reg)
					} else if d124.Loc == LocRegPair {
						ctx.UnprotectReg(d124.Reg)
						ctx.UnprotectReg(d124.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d126 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					if d126.Loc == LocImm {
						ctx.TrackImm(d126.Imm)
						ptrWord, _ := d126.Imm.RawWords()
						d127 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d127.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d127.Reg2, uint64(len(d126.Imm.String())))
						ctx.BindReg(d127.Reg, &d127)
						ctx.BindReg(d127.Reg2, &d127)
					} else {
						d127 = d126
					}
					d128 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d127}, 1)
					ctx.EmitAndRegImm32(d128.Reg, 1)
					d128.Type = tagBool
					ctx.BindReg(d128.Reg, &d128)
					d129 = d128
					ctx.EnsureDesc(&d129)
					if d129.Loc != LocImm && d129.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d129.Loc == LocImm {
						if d129.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitCmpRegImm32(d129.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap130 := d1
						snap131 := d2
						snap132 := d3
						snap133 := d4
						snap134 := d5
						snap135 := d12
						snap136 := d13
						snap137 := d15
						snap138 := d16
						snap139 := d17
						snap140 := d29
						snap141 := d30
						snap142 := d31
						snap143 := d32
						snap144 := d48
						snap145 := d49
						snap146 := d50
						snap147 := d51
						snap148 := d71
						snap149 := d72
						snap150 := d73
						snap151 := d74
						snap152 := d75
						snap153 := d76
						snap154 := d77
						snap155 := d78
						snap156 := d79
						snap157 := d80
						snap158 := d81
						snap159 := d82
						snap160 := d114
						snap161 := d115
						snap162 := d116
						snap163 := d117
						snap164 := d118
						snap165 := d119
						snap166 := d120
						snap167 := d121
						snap168 := d122
						snap169 := d123
						snap170 := d124
						snap171 := d125
						snap172 := d126
						snap173 := d127
						snap174 := d128
						snap175 := d129
						alloc176 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc176)
						d1 = snap130
						d2 = snap131
						d3 = snap132
						d4 = snap133
						d5 = snap134
						d12 = snap135
						d13 = snap136
						d15 = snap137
						d16 = snap138
						d17 = snap139
						d29 = snap140
						d30 = snap141
						d31 = snap142
						d32 = snap143
						d48 = snap144
						d49 = snap145
						d50 = snap146
						d51 = snap147
						d71 = snap148
						d72 = snap149
						d73 = snap150
						d74 = snap151
						d75 = snap152
						d76 = snap153
						d77 = snap154
						d78 = snap155
						d79 = snap156
						d80 = snap157
						d81 = snap158
						d82 = snap159
						d114 = snap160
						d115 = snap161
						d116 = snap162
						d117 = snap163
						d118 = snap164
						d119 = snap165
						d120 = snap166
						d121 = snap167
						d122 = snap168
						d123 = snap169
						d124 = snap170
						d125 = snap171
						d126 = snap172
						d127 = snap173
						d128 = snap174
						d129 = snap175
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					ctx.FreeDesc(&d128)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d178 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d178 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r4 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r4, 0)
							ctx.EmitSubFloat64(r4, d73.Reg)
							d178 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
							ctx.BindReg(r4, &d178)
						} else {
							r5 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r5, 0)
							ctx.EmitSubInt64(r5, d73.Reg)
							d178 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
							ctx.BindReg(r5, &d178)
						}
					}
					ctx.EnsureDesc(&d178)
					ctx.EnsureDesc(&d178)
					if d178.Loc == LocImm {
						d179 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d178.Imm.Int() * 3600000000000)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d178.Reg, 3600000000000)
						d179 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d178.Reg}
						ctx.BindReg(d178.Reg, &d179)
					}
					if d179.Loc == LocReg && d178.Loc == LocReg && d179.Reg == d178.Reg {
						ctx.TransferReg(d178.Reg)
						d178.Loc = LocNone
					}
					ctx.FreeDesc(&d178)
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Add arg0)")
					}
					if d179.Loc == LocRegPair || d179.Loc == LocStackPair || d179.Loc == LocRegTriple || d179.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d179)
					d180 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Add), []JITValueDesc{d15, d179}, 3)
					d180.NoHeapPointer = false
					ctx.BindReg(d180.Reg, &d180)
					ctx.BindReg(d180.Reg2, &d180)
					ctx.BindReg(d180.Reg3, &d180)
					ctx.StabilizeDescForControlFlow(&d180)
					ctx.FreeDesc(&d179)
					ctx.SyncDesc(&d180)
					if d180.Loc == LocReg || d180.Loc == LocFPReg {
						ctx.ProtectReg(d180.Reg)
					} else if d180.Loc == LocRegPair {
						ctx.ProtectReg(d180.Reg)
						ctx.ProtectReg(d180.Reg2)
					}
					d181 = d180
					if d181.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d181)
					if d181.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d181, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d181.Loc == LocInputPair {
						ctx.EnsureDesc(&d181)
						ctx.EmitStoreScmerToStack(d181, int32(bbs[7].PhiBase)+int32(0))
					} else if d181.Loc == LocRegPair || d181.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d181, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d181)
						ctx.EmitStoreToStack(d181, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d180.Loc == LocReg || d180.Loc == LocFPReg {
						ctx.UnprotectReg(d180.Reg)
					} else if d180.Loc == LocRegPair {
						ctx.UnprotectReg(d180.Reg)
						ctx.UnprotectReg(d180.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d182 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					if d182.Loc == LocImm {
						ctx.TrackImm(d182.Imm)
						ptrWord, _ := d182.Imm.RawWords()
						d183 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d183.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d183.Reg2, uint64(len(d182.Imm.String())))
						ctx.BindReg(d183.Reg, &d183)
						ctx.BindReg(d183.Reg2, &d183)
					} else {
						d183 = d182
					}
					d184 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d183}, 1)
					ctx.EmitAndRegImm32(d184.Reg, 1)
					d184.Type = tagBool
					ctx.BindReg(d184.Reg, &d184)
					d185 = d184
					ctx.EnsureDesc(&d185)
					if d185.Loc != LocImm && d185.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d185.Loc == LocImm {
						if d185.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[14].Render()
					}
					ctx.EmitCmpRegImm32(d185.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
					}
					ctx.FlushRegisterMoves()
					if !bbs[14].Rendered {
						snap186 := d1
						snap187 := d2
						snap188 := d3
						snap189 := d4
						snap190 := d5
						snap191 := d12
						snap192 := d13
						snap193 := d15
						snap194 := d16
						snap195 := d17
						snap196 := d29
						snap197 := d30
						snap198 := d31
						snap199 := d32
						snap200 := d48
						snap201 := d49
						snap202 := d50
						snap203 := d51
						snap204 := d71
						snap205 := d72
						snap206 := d73
						snap207 := d74
						snap208 := d75
						snap209 := d76
						snap210 := d77
						snap211 := d78
						snap212 := d79
						snap213 := d80
						snap214 := d81
						snap215 := d82
						snap216 := d114
						snap217 := d115
						snap218 := d116
						snap219 := d117
						snap220 := d118
						snap221 := d119
						snap222 := d120
						snap223 := d121
						snap224 := d122
						snap225 := d123
						snap226 := d124
						snap227 := d125
						snap228 := d126
						snap229 := d127
						snap230 := d128
						snap231 := d129
						snap232 := d177
						snap233 := d178
						snap234 := d179
						snap235 := d180
						snap236 := d181
						snap237 := d182
						snap238 := d183
						snap239 := d184
						snap240 := d185
						alloc241 := ctx.SnapshotAllocState()
						bbs[14].Render()
						ctx.RestoreAllocState(alloc241)
						d1 = snap186
						d2 = snap187
						d3 = snap188
						d4 = snap189
						d5 = snap190
						d12 = snap191
						d13 = snap192
						d15 = snap193
						d16 = snap194
						d17 = snap195
						d29 = snap196
						d30 = snap197
						d31 = snap198
						d32 = snap199
						d48 = snap200
						d49 = snap201
						d50 = snap202
						d51 = snap203
						d71 = snap204
						d72 = snap205
						d73 = snap206
						d74 = snap207
						d75 = snap208
						d76 = snap209
						d77 = snap210
						d78 = snap211
						d79 = snap212
						d80 = snap213
						d81 = snap214
						d82 = snap215
						d114 = snap216
						d115 = snap217
						d116 = snap218
						d117 = snap219
						d118 = snap220
						d119 = snap221
						d120 = snap222
						d121 = snap223
						d122 = snap224
						d123 = snap225
						d124 = snap226
						d125 = snap227
						d126 = snap228
						d127 = snap229
						d128 = snap230
						d129 = snap231
						d177 = snap232
						d178 = snap233
						d179 = snap234
						d180 = snap235
						d181 = snap236
						d182 = snap237
						d183 = snap238
						d184 = snap239
						d185 = snap240
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					ctx.FreeDesc(&d184)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d242 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d242 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r6 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r6, 0)
							ctx.EmitSubFloat64(r6, d73.Reg)
							d242 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
							ctx.BindReg(r6, &d242)
						} else {
							r7 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r7, 0)
							ctx.EmitSubInt64(r7, d73.Reg)
							d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
							ctx.BindReg(r7, &d242)
						}
					}
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					d243 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d243.Loc == LocRegPair || d243.Loc == LocStackPair || d243.Loc == LocRegTriple || d243.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d244 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d244.Loc == LocRegPair || d244.Loc == LocStackPair || d244.Loc == LocRegTriple || d244.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d242.Loc == LocRegPair || d242.Loc == LocStackPair || d242.Loc == LocRegTriple || d242.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d243)
					ctx.SyncDesc(&d244)
					ctx.SyncDesc(&d242)
					d245 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d243, d244, d242}, 3)
					d245.NoHeapPointer = false
					ctx.BindReg(d245.Reg, &d245)
					ctx.BindReg(d245.Reg2, &d245)
					ctx.BindReg(d245.Reg3, &d245)
					ctx.FreeDesc(&d243)
					ctx.FreeDesc(&d244)
					ctx.StabilizeDescForControlFlow(&d245)
					ctx.FreeDesc(&d242)
					ctx.SyncDesc(&d245)
					if d245.Loc == LocReg || d245.Loc == LocFPReg {
						ctx.ProtectReg(d245.Reg)
					} else if d245.Loc == LocRegPair {
						ctx.ProtectReg(d245.Reg)
						ctx.ProtectReg(d245.Reg2)
					}
					d246 = d245
					if d246.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d246)
					if d246.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d246, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d246.Loc == LocInputPair {
						ctx.EnsureDesc(&d246)
						ctx.EmitStoreScmerToStack(d246, int32(bbs[7].PhiBase)+int32(0))
					} else if d246.Loc == LocRegPair || d246.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d246, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d246)
						ctx.EmitStoreToStack(d246, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d245.Loc == LocReg || d245.Loc == LocFPReg {
						ctx.UnprotectReg(d245.Reg)
					} else if d245.Loc == LocRegPair {
						ctx.UnprotectReg(d245.Reg)
						ctx.UnprotectReg(d245.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d247 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
					if d247.Loc == LocImm {
						ctx.TrackImm(d247.Imm)
						ptrWord, _ := d247.Imm.RawWords()
						d248 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d248.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d248.Reg2, uint64(len(d247.Imm.String())))
						ctx.BindReg(d248.Reg, &d248)
						ctx.BindReg(d248.Reg2, &d248)
					} else {
						d248 = d247
					}
					d249 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d248}, 1)
					ctx.EmitAndRegImm32(d249.Reg, 1)
					d249.Type = tagBool
					ctx.BindReg(d249.Reg, &d249)
					d250 = d249
					ctx.EnsureDesc(&d250)
					if d250.Loc != LocImm && d250.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d250.Loc == LocImm {
						if d250.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[16].Render()
					}
					ctx.EmitCmpRegImm32(d250.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
					}
					ctx.FlushRegisterMoves()
					if !bbs[16].Rendered {
						snap251 := d1
						snap252 := d2
						snap253 := d3
						snap254 := d4
						snap255 := d5
						snap256 := d12
						snap257 := d13
						snap258 := d15
						snap259 := d16
						snap260 := d17
						snap261 := d29
						snap262 := d30
						snap263 := d31
						snap264 := d32
						snap265 := d48
						snap266 := d49
						snap267 := d50
						snap268 := d51
						snap269 := d71
						snap270 := d72
						snap271 := d73
						snap272 := d74
						snap273 := d75
						snap274 := d76
						snap275 := d77
						snap276 := d78
						snap277 := d79
						snap278 := d80
						snap279 := d81
						snap280 := d82
						snap281 := d114
						snap282 := d115
						snap283 := d116
						snap284 := d117
						snap285 := d118
						snap286 := d119
						snap287 := d120
						snap288 := d121
						snap289 := d122
						snap290 := d123
						snap291 := d124
						snap292 := d125
						snap293 := d126
						snap294 := d127
						snap295 := d128
						snap296 := d129
						snap297 := d177
						snap298 := d178
						snap299 := d179
						snap300 := d180
						snap301 := d181
						snap302 := d182
						snap303 := d183
						snap304 := d184
						snap305 := d185
						snap306 := d242
						snap307 := d243
						snap308 := d244
						snap309 := d245
						snap310 := d246
						snap311 := d247
						snap312 := d248
						snap313 := d249
						snap314 := d250
						alloc315 := ctx.SnapshotAllocState()
						bbs[16].Render()
						ctx.RestoreAllocState(alloc315)
						d1 = snap251
						d2 = snap252
						d3 = snap253
						d4 = snap254
						d5 = snap255
						d12 = snap256
						d13 = snap257
						d15 = snap258
						d16 = snap259
						d17 = snap260
						d29 = snap261
						d30 = snap262
						d31 = snap263
						d32 = snap264
						d48 = snap265
						d49 = snap266
						d50 = snap267
						d51 = snap268
						d71 = snap269
						d72 = snap270
						d73 = snap271
						d74 = snap272
						d75 = snap273
						d76 = snap274
						d77 = snap275
						d78 = snap276
						d79 = snap277
						d80 = snap278
						d81 = snap279
						d82 = snap280
						d114 = snap281
						d115 = snap282
						d116 = snap283
						d117 = snap284
						d118 = snap285
						d119 = snap286
						d120 = snap287
						d121 = snap288
						d122 = snap289
						d123 = snap290
						d124 = snap291
						d125 = snap292
						d126 = snap293
						d127 = snap294
						d128 = snap295
						d129 = snap296
						d177 = snap297
						d178 = snap298
						d179 = snap299
						d180 = snap300
						d181 = snap301
						d182 = snap302
						d183 = snap303
						d184 = snap304
						d185 = snap305
						d242 = snap306
						d243 = snap307
						d244 = snap308
						d245 = snap309
						d246 = snap310
						d247 = snap311
						d248 = snap312
						d249 = snap313
						d250 = snap314
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					ctx.FreeDesc(&d249)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d316 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d316 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r8 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r8, 0)
							ctx.EmitSubFloat64(r8, d73.Reg)
							d316 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
							ctx.BindReg(r8, &d316)
						} else {
							r9 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r9, 0)
							ctx.EmitSubInt64(r9, d73.Reg)
							d316 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
							ctx.BindReg(r9, &d316)
						}
					}
					ctx.EnsureDesc(&d316)
					ctx.EnsureDesc(&d316)
					if d316.Loc == LocImm {
						d317 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d316.Imm.Int() * 7)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d316.Reg, 7)
						d317 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d316.Reg}
						ctx.BindReg(d316.Reg, &d317)
					}
					if d317.Loc == LocReg && d316.Loc == LocReg && d317.Reg == d316.Reg {
						ctx.TransferReg(d316.Reg)
						d316.Loc = LocNone
					}
					ctx.FreeDesc(&d316)
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					d318 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d318.Loc == LocRegPair || d318.Loc == LocStackPair || d318.Loc == LocRegTriple || d318.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d319 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d319.Loc == LocRegPair || d319.Loc == LocStackPair || d319.Loc == LocRegTriple || d319.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d317.Loc == LocRegPair || d317.Loc == LocStackPair || d317.Loc == LocRegTriple || d317.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d318)
					ctx.SyncDesc(&d319)
					ctx.SyncDesc(&d317)
					d320 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d318, d319, d317}, 3)
					d320.NoHeapPointer = false
					ctx.BindReg(d320.Reg, &d320)
					ctx.BindReg(d320.Reg2, &d320)
					ctx.BindReg(d320.Reg3, &d320)
					ctx.FreeDesc(&d318)
					ctx.FreeDesc(&d319)
					ctx.StabilizeDescForControlFlow(&d320)
					ctx.FreeDesc(&d317)
					ctx.SyncDesc(&d320)
					if d320.Loc == LocReg || d320.Loc == LocFPReg {
						ctx.ProtectReg(d320.Reg)
					} else if d320.Loc == LocRegPair {
						ctx.ProtectReg(d320.Reg)
						ctx.ProtectReg(d320.Reg2)
					}
					d321 = d320
					if d321.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d321)
					if d321.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d321, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d321.Loc == LocInputPair {
						ctx.EnsureDesc(&d321)
						ctx.EmitStoreScmerToStack(d321, int32(bbs[7].PhiBase)+int32(0))
					} else if d321.Loc == LocRegPair || d321.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d321, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d321)
						ctx.EmitStoreToStack(d321, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d320.Loc == LocReg || d320.Loc == LocFPReg {
						ctx.UnprotectReg(d320.Reg)
					} else if d320.Loc == LocRegPair {
						ctx.UnprotectReg(d320.Reg)
						ctx.UnprotectReg(d320.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d322 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					if d322.Loc == LocImm {
						ctx.TrackImm(d322.Imm)
						ptrWord, _ := d322.Imm.RawWords()
						d323 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d323.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d323.Reg2, uint64(len(d322.Imm.String())))
						ctx.BindReg(d323.Reg, &d323)
						ctx.BindReg(d323.Reg2, &d323)
					} else {
						d323 = d322
					}
					d324 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d323}, 1)
					ctx.EmitAndRegImm32(d324.Reg, 1)
					d324.Type = tagBool
					ctx.BindReg(d324.Reg, &d324)
					d325 = d324
					ctx.EnsureDesc(&d325)
					if d325.Loc != LocImm && d325.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d325.Loc == LocImm {
						if d325.Imm.Bool() {
							return bbs[15].Render()
						}
						return bbs[18].Render()
					}
					ctx.EmitCmpRegImm32(d325.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
					}
					ctx.FlushRegisterMoves()
					if !bbs[18].Rendered {
						snap326 := d1
						snap327 := d2
						snap328 := d3
						snap329 := d4
						snap330 := d5
						snap331 := d12
						snap332 := d13
						snap333 := d15
						snap334 := d16
						snap335 := d17
						snap336 := d29
						snap337 := d30
						snap338 := d31
						snap339 := d32
						snap340 := d48
						snap341 := d49
						snap342 := d50
						snap343 := d51
						snap344 := d71
						snap345 := d72
						snap346 := d73
						snap347 := d74
						snap348 := d75
						snap349 := d76
						snap350 := d77
						snap351 := d78
						snap352 := d79
						snap353 := d80
						snap354 := d81
						snap355 := d82
						snap356 := d114
						snap357 := d115
						snap358 := d116
						snap359 := d117
						snap360 := d118
						snap361 := d119
						snap362 := d120
						snap363 := d121
						snap364 := d122
						snap365 := d123
						snap366 := d124
						snap367 := d125
						snap368 := d126
						snap369 := d127
						snap370 := d128
						snap371 := d129
						snap372 := d177
						snap373 := d178
						snap374 := d179
						snap375 := d180
						snap376 := d181
						snap377 := d182
						snap378 := d183
						snap379 := d184
						snap380 := d185
						snap381 := d242
						snap382 := d243
						snap383 := d244
						snap384 := d245
						snap385 := d246
						snap386 := d247
						snap387 := d248
						snap388 := d249
						snap389 := d250
						snap390 := d316
						snap391 := d317
						snap392 := d318
						snap393 := d319
						snap394 := d320
						snap395 := d321
						snap396 := d322
						snap397 := d323
						snap398 := d324
						snap399 := d325
						alloc400 := ctx.SnapshotAllocState()
						bbs[18].Render()
						ctx.RestoreAllocState(alloc400)
						d1 = snap326
						d2 = snap327
						d3 = snap328
						d4 = snap329
						d5 = snap330
						d12 = snap331
						d13 = snap332
						d15 = snap333
						d16 = snap334
						d17 = snap335
						d29 = snap336
						d30 = snap337
						d31 = snap338
						d32 = snap339
						d48 = snap340
						d49 = snap341
						d50 = snap342
						d51 = snap343
						d71 = snap344
						d72 = snap345
						d73 = snap346
						d74 = snap347
						d75 = snap348
						d76 = snap349
						d77 = snap350
						d78 = snap351
						d79 = snap352
						d80 = snap353
						d81 = snap354
						d82 = snap355
						d114 = snap356
						d115 = snap357
						d116 = snap358
						d117 = snap359
						d118 = snap360
						d119 = snap361
						d120 = snap362
						d121 = snap363
						d122 = snap364
						d123 = snap365
						d124 = snap366
						d125 = snap367
						d126 = snap368
						d127 = snap369
						d128 = snap370
						d129 = snap371
						d177 = snap372
						d178 = snap373
						d179 = snap374
						d180 = snap375
						d181 = snap376
						d182 = snap377
						d183 = snap378
						d184 = snap379
						d185 = snap380
						d242 = snap381
						d243 = snap382
						d244 = snap383
						d245 = snap384
						d246 = snap385
						d247 = snap386
						d248 = snap387
						d249 = snap388
						d250 = snap389
						d316 = snap390
						d317 = snap391
						d318 = snap392
						d319 = snap393
						d320 = snap394
						d321 = snap395
						d322 = snap396
						d323 = snap397
						d324 = snap398
						d325 = snap399
					}
					if !bbs[15].Rendered {
						return bbs[15].Render()
					}
					return result
					ctx.FreeDesc(&d324)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d401 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d401 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r10 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r10, 0)
							ctx.EmitSubFloat64(r10, d73.Reg)
							d401 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r10}
							ctx.BindReg(r10, &d401)
						} else {
							r11 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r11, 0)
							ctx.EmitSubInt64(r11, d73.Reg)
							d401 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
							ctx.BindReg(r11, &d401)
						}
					}
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					d402 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d402.Loc == LocRegPair || d402.Loc == LocStackPair || d402.Loc == LocRegTriple || d402.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d401.Loc == LocRegPair || d401.Loc == LocStackPair || d401.Loc == LocRegTriple || d401.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d403 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d403.Loc == LocRegPair || d403.Loc == LocStackPair || d403.Loc == LocRegTriple || d403.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d402)
					ctx.SyncDesc(&d401)
					ctx.SyncDesc(&d403)
					d404 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d402, d401, d403}, 3)
					d404.NoHeapPointer = false
					ctx.BindReg(d404.Reg, &d404)
					ctx.BindReg(d404.Reg2, &d404)
					ctx.BindReg(d404.Reg3, &d404)
					ctx.FreeDesc(&d402)
					ctx.FreeDesc(&d403)
					ctx.StabilizeDescForControlFlow(&d404)
					ctx.FreeDesc(&d401)
					ctx.SyncDesc(&d404)
					if d404.Loc == LocReg || d404.Loc == LocFPReg {
						ctx.ProtectReg(d404.Reg)
					} else if d404.Loc == LocRegPair {
						ctx.ProtectReg(d404.Reg)
						ctx.ProtectReg(d404.Reg2)
					}
					d405 = d404
					if d405.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d405)
					if d405.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d405, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d405.Loc == LocInputPair {
						ctx.EnsureDesc(&d405)
						ctx.EmitStoreScmerToStack(d405, int32(bbs[7].PhiBase)+int32(0))
					} else if d405.Loc == LocRegPair || d405.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d405, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d405)
						ctx.EmitStoreToStack(d405, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d404.Loc == LocReg || d404.Loc == LocFPReg {
						ctx.UnprotectReg(d404.Reg)
					} else if d404.Loc == LocRegPair {
						ctx.UnprotectReg(d404.Reg)
						ctx.UnprotectReg(d404.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d406 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					if d406.Loc == LocImm {
						ctx.TrackImm(d406.Imm)
						ptrWord, _ := d406.Imm.RawWords()
						d407 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d407.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d407.Reg2, uint64(len(d406.Imm.String())))
						ctx.BindReg(d407.Reg, &d407)
						ctx.BindReg(d407.Reg2, &d407)
					} else {
						d407 = d406
					}
					d408 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d407}, 1)
					ctx.EmitAndRegImm32(d408.Reg, 1)
					d408.Type = tagBool
					ctx.BindReg(d408.Reg, &d408)
					d409 = d408
					ctx.EnsureDesc(&d409)
					if d409.Loc != LocImm && d409.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d409.Loc == LocImm {
						if d409.Imm.Bool() {
							return bbs[17].Render()
						}
						return bbs[20].Render()
					}
					ctx.EmitCmpRegImm32(d409.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
					}
					ctx.FlushRegisterMoves()
					if !bbs[20].Rendered {
						snap410 := d1
						snap411 := d2
						snap412 := d3
						snap413 := d4
						snap414 := d5
						snap415 := d12
						snap416 := d13
						snap417 := d15
						snap418 := d16
						snap419 := d17
						snap420 := d29
						snap421 := d30
						snap422 := d31
						snap423 := d32
						snap424 := d48
						snap425 := d49
						snap426 := d50
						snap427 := d51
						snap428 := d71
						snap429 := d72
						snap430 := d73
						snap431 := d74
						snap432 := d75
						snap433 := d76
						snap434 := d77
						snap435 := d78
						snap436 := d79
						snap437 := d80
						snap438 := d81
						snap439 := d82
						snap440 := d114
						snap441 := d115
						snap442 := d116
						snap443 := d117
						snap444 := d118
						snap445 := d119
						snap446 := d120
						snap447 := d121
						snap448 := d122
						snap449 := d123
						snap450 := d124
						snap451 := d125
						snap452 := d126
						snap453 := d127
						snap454 := d128
						snap455 := d129
						snap456 := d177
						snap457 := d178
						snap458 := d179
						snap459 := d180
						snap460 := d181
						snap461 := d182
						snap462 := d183
						snap463 := d184
						snap464 := d185
						snap465 := d242
						snap466 := d243
						snap467 := d244
						snap468 := d245
						snap469 := d246
						snap470 := d247
						snap471 := d248
						snap472 := d249
						snap473 := d250
						snap474 := d316
						snap475 := d317
						snap476 := d318
						snap477 := d319
						snap478 := d320
						snap479 := d321
						snap480 := d322
						snap481 := d323
						snap482 := d324
						snap483 := d325
						snap484 := d401
						snap485 := d402
						snap486 := d403
						snap487 := d404
						snap488 := d405
						snap489 := d406
						snap490 := d407
						snap491 := d408
						snap492 := d409
						alloc493 := ctx.SnapshotAllocState()
						bbs[20].Render()
						ctx.RestoreAllocState(alloc493)
						d1 = snap410
						d2 = snap411
						d3 = snap412
						d4 = snap413
						d5 = snap414
						d12 = snap415
						d13 = snap416
						d15 = snap417
						d16 = snap418
						d17 = snap419
						d29 = snap420
						d30 = snap421
						d31 = snap422
						d32 = snap423
						d48 = snap424
						d49 = snap425
						d50 = snap426
						d51 = snap427
						d71 = snap428
						d72 = snap429
						d73 = snap430
						d74 = snap431
						d75 = snap432
						d76 = snap433
						d77 = snap434
						d78 = snap435
						d79 = snap436
						d80 = snap437
						d81 = snap438
						d82 = snap439
						d114 = snap440
						d115 = snap441
						d116 = snap442
						d117 = snap443
						d118 = snap444
						d119 = snap445
						d120 = snap446
						d121 = snap447
						d122 = snap448
						d123 = snap449
						d124 = snap450
						d125 = snap451
						d126 = snap452
						d127 = snap453
						d128 = snap454
						d129 = snap455
						d177 = snap456
						d178 = snap457
						d179 = snap458
						d180 = snap459
						d181 = snap460
						d182 = snap461
						d183 = snap462
						d184 = snap463
						d185 = snap464
						d242 = snap465
						d243 = snap466
						d244 = snap467
						d245 = snap468
						d246 = snap469
						d247 = snap470
						d248 = snap471
						d249 = snap472
						d250 = snap473
						d316 = snap474
						d317 = snap475
						d318 = snap476
						d319 = snap477
						d320 = snap478
						d321 = snap479
						d322 = snap480
						d323 = snap481
						d324 = snap482
						d325 = snap483
						d401 = snap484
						d402 = snap485
						d403 = snap486
						d404 = snap487
						d405 = snap488
						d406 = snap489
						d407 = snap490
						d408 = snap491
						d409 = snap492
					}
					if !bbs[17].Rendered {
						return bbs[17].Render()
					}
					return result
					ctx.FreeDesc(&d408)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						if d73.Type == tagFloat {
							d494 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d73.Imm.Float())}
						} else {
							d494 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d73.Imm.Int())}
						}
					} else {
						if d73.Type == tagFloat {
							r12 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r12, 0)
							ctx.EmitSubFloat64(r12, d73.Reg)
							d494 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
							ctx.BindReg(r12, &d494)
						} else {
							r13 := ctx.AllocRegExcept(d73.Reg)
							ctx.EmitMovRegImm64(r13, 0)
							ctx.EmitSubInt64(r13, d73.Reg)
							d494 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
							ctx.BindReg(r13, &d494)
						}
					}
					ctx.FreeDesc(&d73)
					d15 = JITPrepareGoSliceArg(ctx, d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).AddDate arg0)")
					}
					if d494.Loc == LocRegPair || d494.Loc == LocStackPair || d494.Loc == LocRegTriple || d494.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d495 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d495.Loc == LocRegPair || d495.Loc == LocStackPair || d495.Loc == LocRegTriple || d495.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d496 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d496.Loc == LocRegPair || d496.Loc == LocStackPair || d496.Loc == LocRegTriple || d496.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d494)
					ctx.SyncDesc(&d495)
					ctx.SyncDesc(&d496)
					d497 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).AddDate), []JITValueDesc{d15, d494, d495, d496}, 3)
					d497.NoHeapPointer = false
					ctx.BindReg(d497.Reg, &d497)
					ctx.BindReg(d497.Reg2, &d497)
					ctx.BindReg(d497.Reg3, &d497)
					ctx.FreeDesc(&d495)
					ctx.FreeDesc(&d496)
					ctx.StabilizeDescForControlFlow(&d497)
					ctx.FreeDesc(&d494)
					ctx.SyncDesc(&d497)
					if d497.Loc == LocReg || d497.Loc == LocFPReg {
						ctx.ProtectReg(d497.Reg)
					} else if d497.Loc == LocRegPair {
						ctx.ProtectReg(d497.Reg)
						ctx.ProtectReg(d497.Reg2)
					}
					d498 = d497
					if d498.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d498)
					if d498.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d498, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d498.Loc == LocInputPair {
						ctx.EnsureDesc(&d498)
						ctx.EmitStoreScmerToStack(d498, int32(bbs[7].PhiBase)+int32(0))
					} else if d498.Loc == LocRegPair || d498.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d498, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d498)
						ctx.EmitStoreToStack(d498, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d497.Loc == LocReg || d497.Loc == LocFPReg {
						ctx.UnprotectReg(d497.Reg)
					} else if d497.Loc == LocRegPair {
						ctx.UnprotectReg(d497.Reg)
						ctx.UnprotectReg(d497.Reg2)
					}
					return bbs[7].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d499 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					if d499.Loc == LocImm {
						ctx.TrackImm(d499.Imm)
						ptrWord, _ := d499.Imm.RawWords()
						d500 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d500.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d500.Reg2, uint64(len(d499.Imm.String())))
						ctx.BindReg(d500.Reg, &d500)
						ctx.BindReg(d500.Reg2, &d500)
					} else {
						d500 = d499
					}
					d501 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d78, d500}, 1)
					ctx.EmitAndRegImm32(d501.Reg, 1)
					d501.Type = tagBool
					ctx.BindReg(d501.Reg, &d501)
					d502 = d501
					ctx.EnsureDesc(&d502)
					if d502.Loc != LocImm && d502.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d502.Loc == LocImm {
						if d502.Imm.Bool() {
							return bbs[19].Render()
						}
						return bbs[21].Render()
					}
					ctx.EmitCmpRegImm32(d502.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					if bbs[21].Rendered {
						ctx.EmitJmp(lbl22)
					}
					ctx.FlushRegisterMoves()
					if !bbs[21].Rendered {
						snap503 := d1
						snap504 := d2
						snap505 := d3
						snap506 := d4
						snap507 := d5
						snap508 := d12
						snap509 := d13
						snap510 := d15
						snap511 := d16
						snap512 := d17
						snap513 := d29
						snap514 := d30
						snap515 := d31
						snap516 := d32
						snap517 := d48
						snap518 := d49
						snap519 := d50
						snap520 := d51
						snap521 := d71
						snap522 := d72
						snap523 := d73
						snap524 := d74
						snap525 := d75
						snap526 := d76
						snap527 := d77
						snap528 := d78
						snap529 := d79
						snap530 := d80
						snap531 := d81
						snap532 := d82
						snap533 := d114
						snap534 := d115
						snap535 := d116
						snap536 := d117
						snap537 := d118
						snap538 := d119
						snap539 := d120
						snap540 := d121
						snap541 := d122
						snap542 := d123
						snap543 := d124
						snap544 := d125
						snap545 := d126
						snap546 := d127
						snap547 := d128
						snap548 := d129
						snap549 := d177
						snap550 := d178
						snap551 := d179
						snap552 := d180
						snap553 := d181
						snap554 := d182
						snap555 := d183
						snap556 := d184
						snap557 := d185
						snap558 := d242
						snap559 := d243
						snap560 := d244
						snap561 := d245
						snap562 := d246
						snap563 := d247
						snap564 := d248
						snap565 := d249
						snap566 := d250
						snap567 := d316
						snap568 := d317
						snap569 := d318
						snap570 := d319
						snap571 := d320
						snap572 := d321
						snap573 := d322
						snap574 := d323
						snap575 := d324
						snap576 := d325
						snap577 := d401
						snap578 := d402
						snap579 := d403
						snap580 := d404
						snap581 := d405
						snap582 := d406
						snap583 := d407
						snap584 := d408
						snap585 := d409
						snap586 := d494
						snap587 := d495
						snap588 := d496
						snap589 := d497
						snap590 := d498
						snap591 := d499
						snap592 := d500
						snap593 := d501
						snap594 := d502
						alloc595 := ctx.SnapshotAllocState()
						bbs[21].Render()
						ctx.RestoreAllocState(alloc595)
						d1 = snap503
						d2 = snap504
						d3 = snap505
						d4 = snap506
						d5 = snap507
						d12 = snap508
						d13 = snap509
						d15 = snap510
						d16 = snap511
						d17 = snap512
						d29 = snap513
						d30 = snap514
						d31 = snap515
						d32 = snap516
						d48 = snap517
						d49 = snap518
						d50 = snap519
						d51 = snap520
						d71 = snap521
						d72 = snap522
						d73 = snap523
						d74 = snap524
						d75 = snap525
						d76 = snap526
						d77 = snap527
						d78 = snap528
						d79 = snap529
						d80 = snap530
						d81 = snap531
						d82 = snap532
						d114 = snap533
						d115 = snap534
						d116 = snap535
						d117 = snap536
						d118 = snap537
						d119 = snap538
						d120 = snap539
						d121 = snap540
						d122 = snap541
						d123 = snap542
						d124 = snap543
						d125 = snap544
						d126 = snap545
						d127 = snap546
						d128 = snap547
						d129 = snap548
						d177 = snap549
						d178 = snap550
						d179 = snap551
						d180 = snap552
						d181 = snap553
						d182 = snap554
						d183 = snap555
						d184 = snap556
						d185 = snap557
						d242 = snap558
						d243 = snap559
						d244 = snap560
						d245 = snap561
						d246 = snap562
						d247 = snap563
						d248 = snap564
						d249 = snap565
						d250 = snap566
						d316 = snap567
						d317 = snap568
						d318 = snap569
						d319 = snap570
						d320 = snap571
						d321 = snap572
						d322 = snap573
						d323 = snap574
						d324 = snap575
						d325 = snap576
						d401 = snap577
						d402 = snap578
						d403 = snap579
						d404 = snap580
						d405 = snap581
						d406 = snap582
						d407 = snap583
						d408 = snap584
						d409 = snap585
						d494 = snap586
						d495 = snap587
						d496 = snap588
						d497 = snap589
						d498 = snap590
						d499 = snap591
						d500 = snap592
						d501 = snap593
						d502 = snap594
					}
					if !bbs[19].Rendered {
						return bbs[19].Render()
					}
					return result
					ctx.FreeDesc(&d501)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["date_sub"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
			JITInlineCost:  79,
		},
	})

	// DATE(expr) - truncate to date only (midnight)
	Declare(&Globalenv, &Declaration{
		Name: "date_trunc_day",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "truncates a datetime to date (midnight UTC)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date/datetime value"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["date_trunc_day"]
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
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
					d10 = JITPrepareScmerGoArg(ctx, d10)
					ctx.SyncDesc(&d10)
					callResults11 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d10}, []uint8{3, 1}, []uint8{4, 0})
					d12 = callResults11[0]
					_ = d12
					d13 = callResults11[1]
					_ = d13
					ctx.FreeDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d12)
					d14 = d13
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d14.Loc == LocImm {
						if d14.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap15 := d0
						snap16 := d1
						snap17 := d2
						snap18 := d3
						snap19 := d9
						snap20 := d10
						snap21 := d12
						snap22 := d13
						snap23 := d14
						alloc24 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc24)
						d0 = snap15
						d1 = snap16
						d2 = snap17
						d3 = snap18
						d9 = snap19
						d10 = snap20
						d12 = snap21
						d13 = snap22
						d14 = snap23
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d13)
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
					d25 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d25)
					if d25.Loc == LocRegPair || d25.Loc == LocStackPair || d25.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d25, &result)
						result.Type = d25.Type
					} else {
						switch d25.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d25)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d25)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d25)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d25, &result)
							result.Type = d25.Type
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
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d12)
					d26 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d12}, 1)
					d26.NoHeapPointer = true
					ctx.BindReg(d26.Reg, &d26)
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d12)
					d27 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d12}, 1)
					d27.NoHeapPointer = true
					ctx.BindReg(d27.Reg, &d27)
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d12)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d12}, 1)
					d28.NoHeapPointer = true
					ctx.BindReg(d28.Reg, &d28)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d26.Loc == LocRegPair || d26.Loc == LocStackPair || d26.Loc == LocRegTriple || d26.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocRegTriple || d28.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocRegTriple || d31.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocRegTriple || d32.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d33.Loc == LocRegPair || d33.Loc == LocStackPair || d33.Loc == LocRegTriple || d33.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d29.Loc == LocRegPair || d29.Loc == LocStackPair || d29.Loc == LocRegTriple || d29.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					ctx.SyncDesc(&d28)
					ctx.SyncDesc(&d30)
					ctx.SyncDesc(&d31)
					ctx.SyncDesc(&d32)
					ctx.SyncDesc(&d33)
					ctx.SyncDesc(&d29)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d26, d27, d28, d30, d31, d32, d33, d29}, 3)
					d34.NoHeapPointer = false
					ctx.BindReg(d34.Reg, &d34)
					ctx.BindReg(d34.Reg2, &d34)
					ctx.BindReg(d34.Reg3, &d34)
					ctx.FreeDesc(&d30)
					ctx.FreeDesc(&d31)
					ctx.FreeDesc(&d32)
					ctx.FreeDesc(&d33)
					ctx.FreeDesc(&d26)
					ctx.FreeDesc(&d27)
					ctx.FreeDesc(&d28)
					ctx.FreeDesc(&d29)
					d34 = JITPrepareGoSliceArg(ctx, d34)
					if d34.Loc != LocRegTriple && d34.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d34)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d34}, 1)
					d35.NoHeapPointer = true
					ctx.BindReg(d35.Reg, &d35)
					ctx.FreeDesc(&d34)
					if d35.Loc == LocRegPair || d35.Loc == LocStackPair || d35.Loc == LocRegTriple || d35.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d35)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d35}, 2)
					d36.NoHeapPointer = false
					ctx.BindReg(d36.Reg, &d36)
					ctx.BindReg(d36.Reg2, &d36)
					ctx.FreeDesc(&d35)
					ctx.SyncDesc(&d36)
					if d36.Loc == LocRegPair || d36.Loc == LocStackPair || d36.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d36, &result)
						result.Type = d36.Type
					} else {
						switch d36.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d36)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d36)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d36)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d36, &result)
							result.Type = d36.Type
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
			JITVirtualArgs: true,
			JITInlineCost:  22,
		},
	})

	// TIMESTAMPDIFF(unit, datetime1, datetime2) - returns datetime2 - datetime1 in the given unit
	Declare(&Globalenv, &Declaration{
		Name: "timestampdiff",

		Fn: func(a ...Scmer) Scmer {
			if a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			t1, ok1 := toTime(a[1])
			t2, ok2 := toTime(a[2])
			if !ok1 || !ok2 {
				return NewNil()
			}
			unit := strings.ToUpper(a[0].String())
			switch unit {
			case "MICROSECOND":
				return NewInt(t2.Sub(t1).Microseconds())
			case "SECOND":
				return NewInt(int64(t2.Sub(t1).Seconds()))
			case "MINUTE":
				return NewInt(int64(t2.Sub(t1).Minutes()))
			case "HOUR":
				return NewInt(int64(t2.Sub(t1).Hours()))
			case "DAY":
				d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
				d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
				return NewInt(int64(d2.Sub(d1).Hours() / 24))
			case "WEEK":
				d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
				d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
				return NewInt(int64(d2.Sub(d1).Hours() / 24 / 7))
			case "MONTH":
				years := int64(t2.Year() - t1.Year())
				months := int64(t2.Month() - t1.Month())
				total := years*12 + months
				// adjust if day of month hasn't been reached yet
				if t2.Day() < t1.Day() {
					total--
				}
				return NewInt(total)
			case "QUARTER":
				years := int64(t2.Year() - t1.Year())
				months := int64(t2.Month() - t1.Month())
				total := years*12 + months
				if t2.Day() < t1.Day() {
					total--
				}
				return NewInt(total / 3)
			case "YEAR":
				years := int64(t2.Year() - t1.Year())
				// adjust if month/day hasn't been reached yet
				if t2.Month() < t1.Month() || (t2.Month() == t1.Month() && t2.Day() < t1.Day()) {
					years--
				}
				return NewInt(years)
			default:
				panic("unknown TIMESTAMPDIFF unit: " + unit)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the difference between two datetime values in the specified unit (datetime2 - datetime1)",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "unit", Description: "unit: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR"},
				{Kind: "any", Label: "datetime1", Description: "first datetime value"},
				{Kind: "any", Label: "datetime2", Description: "second datetime value"},
			},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["timestampdiff"]
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
				var d17 JITValueDesc
				_ = d17
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
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
				var d104 JITValueDesc
				_ = d104
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				var d141 JITValueDesc
				_ = d141
				var d142 JITValueDesc
				_ = d142
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
				var d191 JITValueDesc
				_ = d191
				var d192 JITValueDesc
				_ = d192
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d244 JITValueDesc
				_ = d244
				var d245 JITValueDesc
				_ = d245
				var d246 JITValueDesc
				_ = d246
				var d247 JITValueDesc
				_ = d247
				var d248 JITValueDesc
				_ = d248
				var d249 JITValueDesc
				_ = d249
				var d250 JITValueDesc
				_ = d250
				var d251 JITValueDesc
				_ = d251
				var d309 JITValueDesc
				_ = d309
				var d310 JITValueDesc
				_ = d310
				var d311 JITValueDesc
				_ = d311
				var d312 JITValueDesc
				_ = d312
				var d313 JITValueDesc
				_ = d313
				var d314 JITValueDesc
				_ = d314
				var d315 JITValueDesc
				_ = d315
				var d316 JITValueDesc
				_ = d316
				var d317 JITValueDesc
				_ = d317
				var d318 JITValueDesc
				_ = d318
				var d319 JITValueDesc
				_ = d319
				var d320 JITValueDesc
				_ = d320
				var d321 JITValueDesc
				_ = d321
				var d322 JITValueDesc
				_ = d322
				var d323 JITValueDesc
				_ = d323
				var d324 JITValueDesc
				_ = d324
				var d325 JITValueDesc
				_ = d325
				var d326 JITValueDesc
				_ = d326
				var d327 JITValueDesc
				_ = d327
				var d328 JITValueDesc
				_ = d328
				var d329 JITValueDesc
				_ = d329
				var d330 JITValueDesc
				_ = d330
				var d331 JITValueDesc
				_ = d331
				var d332 JITValueDesc
				_ = d332
				var d333 JITValueDesc
				_ = d333
				var d334 JITValueDesc
				_ = d334
				var d335 JITValueDesc
				_ = d335
				var d420 JITValueDesc
				_ = d420
				var d421 JITValueDesc
				_ = d421
				var d422 JITValueDesc
				_ = d422
				var d423 JITValueDesc
				_ = d423
				var d424 JITValueDesc
				_ = d424
				var d425 JITValueDesc
				_ = d425
				var d426 JITValueDesc
				_ = d426
				var d427 JITValueDesc
				_ = d427
				var d428 JITValueDesc
				_ = d428
				var d429 JITValueDesc
				_ = d429
				var d430 JITValueDesc
				_ = d430
				var d431 JITValueDesc
				_ = d431
				var d432 JITValueDesc
				_ = d432
				var d433 JITValueDesc
				_ = d433
				var d434 JITValueDesc
				_ = d434
				var d435 JITValueDesc
				_ = d435
				var d436 JITValueDesc
				_ = d436
				var d437 JITValueDesc
				_ = d437
				var d438 JITValueDesc
				_ = d438
				var d439 JITValueDesc
				_ = d439
				var d440 JITValueDesc
				_ = d440
				var d441 JITValueDesc
				_ = d441
				var d442 JITValueDesc
				_ = d442
				var d443 JITValueDesc
				_ = d443
				var d444 JITValueDesc
				_ = d444
				var d445 JITValueDesc
				_ = d445
				var d446 JITValueDesc
				_ = d446
				var d447 JITValueDesc
				_ = d447
				var d560 JITValueDesc
				_ = d560
				var d561 JITValueDesc
				_ = d561
				var d562 JITValueDesc
				_ = d562
				var d563 JITValueDesc
				_ = d563
				var d564 JITValueDesc
				_ = d564
				var d565 JITValueDesc
				_ = d565
				var d566 JITValueDesc
				_ = d566
				var d567 JITValueDesc
				_ = d567
				var d568 JITValueDesc
				_ = d568
				var d569 JITValueDesc
				_ = d569
				var d570 JITValueDesc
				_ = d570
				var d571 JITValueDesc
				_ = d571
				var d572 JITValueDesc
				_ = d572
				var d573 JITValueDesc
				_ = d573
				var d574 JITValueDesc
				_ = d574
				var d702 JITValueDesc
				_ = d702
				var d831 JITValueDesc
				_ = d831
				var d832 JITValueDesc
				_ = d832
				var d833 JITValueDesc
				_ = d833
				var d834 JITValueDesc
				_ = d834
				var d967 JITValueDesc
				_ = d967
				var d968 JITValueDesc
				_ = d968
				var d969 JITValueDesc
				_ = d969
				var d970 JITValueDesc
				_ = d970
				var d971 JITValueDesc
				_ = d971
				var d972 JITValueDesc
				_ = d972
				var d973 JITValueDesc
				_ = d973
				var d974 JITValueDesc
				_ = d974
				var d975 JITValueDesc
				_ = d975
				var d976 JITValueDesc
				_ = d976
				var d977 JITValueDesc
				_ = d977
				var d978 JITValueDesc
				_ = d978
				var d979 JITValueDesc
				_ = d979
				var d980 JITValueDesc
				_ = d980
				var d981 JITValueDesc
				_ = d981
				var d1129 JITValueDesc
				_ = d1129
				var d1278 JITValueDesc
				_ = d1278
				var d1279 JITValueDesc
				_ = d1279
				var d1280 JITValueDesc
				_ = d1280
				var d1281 JITValueDesc
				_ = d1281
				var d1434 JITValueDesc
				_ = d1434
				var d1435 JITValueDesc
				_ = d1435
				var d1436 JITValueDesc
				_ = d1436
				var d1437 JITValueDesc
				_ = d1437
				var d1438 JITValueDesc
				_ = d1438
				var d1439 JITValueDesc
				_ = d1439
				var d1440 JITValueDesc
				_ = d1440
				var d1441 JITValueDesc
				_ = d1441
				var d1442 JITValueDesc
				_ = d1442
				var d1443 JITValueDesc
				_ = d1443
				var d1606 JITValueDesc
				_ = d1606
				var d1607 JITValueDesc
				_ = d1607
				var d1608 JITValueDesc
				_ = d1608
				var d1609 JITValueDesc
				_ = d1609
				var d1776 JITValueDesc
				_ = d1776
				var d1778 JITValueDesc
				_ = d1778
				var d1779 JITValueDesc
				_ = d1779
				var d1780 JITValueDesc
				_ = d1780
				var d1781 JITValueDesc
				_ = d1781
				var d1782 JITValueDesc
				_ = d1782
				var d1783 JITValueDesc
				_ = d1783
				var d1784 JITValueDesc
				_ = d1784
				var d1785 JITValueDesc
				_ = d1785
				var d1786 JITValueDesc
				_ = d1786
				var d1963 JITValueDesc
				_ = d1963
				var d2141 JITValueDesc
				_ = d2141
				var d2142 JITValueDesc
				_ = d2142
				var d2143 JITValueDesc
				_ = d2143
				var d2144 JITValueDesc
				_ = d2144
				var d2145 JITValueDesc
				_ = d2145
				var d2328 JITValueDesc
				_ = d2328
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
				var bbs [33]BBDescriptor
				bbs[23].PhiBase = int32(phiBase0) + int32(0)
				bbs[27].PhiBase = int32(phiBase0) + int32(16)
				bbs[30].PhiBase = int32(phiBase0) + int32(32)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
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
					ctx.ReclaimUntrackedRegs()
					d4 = args[1]
					d4.ID = 0
					d6 = d4
					d6.ID = 0
					d5 = ctx.EmitTagEqualsBorrowed(&d6, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d4)
					d7 = d5
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d7.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
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
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d5)
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
					ctx.ReclaimUntrackedRegs()
					d16 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d16, &result)
						result.Type = d16.Type
					} else {
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d16)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d16)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d16)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d16, &result)
							result.Type = d16.Type
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
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d17 = args[1]
					d17.ID = 0
					d17 = JITPrepareScmerGoArg(ctx, d17)
					ctx.SyncDesc(&d17)
					callResults18 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d17}, []uint8{3, 1}, []uint8{4, 0})
					d19 = callResults18[0]
					_ = d19
					d20 = callResults18[1]
					_ = d20
					ctx.FreeDesc(&d17)
					ctx.StabilizeDescForControlFlow(&d19)
					d21 = args[2]
					d21.ID = 0
					d21 = JITPrepareScmerGoArg(ctx, d21)
					ctx.SyncDesc(&d21)
					callResults22 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d21}, []uint8{3, 1}, []uint8{4, 0})
					d23 = callResults22[0]
					_ = d23
					d24 = callResults22[1]
					_ = d24
					ctx.FreeDesc(&d21)
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.StabilizeDescForControlFlow(&d24)
					d25 = d20
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap26 := d1
						snap27 := d2
						snap28 := d3
						snap29 := d4
						snap30 := d5
						snap31 := d6
						snap32 := d7
						snap33 := d16
						snap34 := d17
						snap35 := d19
						snap36 := d20
						snap37 := d21
						snap38 := d23
						snap39 := d24
						snap40 := d25
						alloc41 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc41)
						d1 = snap26
						d2 = snap27
						d3 = snap28
						d4 = snap29
						d5 = snap30
						d6 = snap31
						d7 = snap32
						d16 = snap33
						d17 = snap34
						d19 = snap35
						d20 = snap36
						d21 = snap37
						d23 = snap38
						d24 = snap39
						d25 = snap40
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d20)
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
					ctx.ReclaimUntrackedRegs()
					d42 = args[2]
					d42.ID = 0
					d44 = d42
					d44.ID = 0
					d43 = ctx.EmitTagEqualsBorrowed(&d44, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d42)
					d45 = d43
					ctx.EnsureDesc(&d45)
					if d45.Loc != LocImm && d45.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d45.Loc == LocImm {
						if d45.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d45.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap46 := d1
						snap47 := d2
						snap48 := d3
						snap49 := d4
						snap50 := d5
						snap51 := d6
						snap52 := d7
						snap53 := d16
						snap54 := d17
						snap55 := d19
						snap56 := d20
						snap57 := d21
						snap58 := d23
						snap59 := d24
						snap60 := d25
						snap61 := d42
						snap62 := d43
						snap63 := d44
						snap64 := d45
						alloc65 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc65)
						d1 = snap46
						d2 = snap47
						d3 = snap48
						d4 = snap49
						d5 = snap50
						d6 = snap51
						d7 = snap52
						d16 = snap53
						d17 = snap54
						d19 = snap55
						d20 = snap56
						d21 = snap57
						d23 = snap58
						d24 = snap59
						d25 = snap60
						d42 = snap61
						d43 = snap62
						d44 = snap63
						d45 = snap64
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d43)
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
					ctx.ReclaimUntrackedRegs()
					d66 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d66)
					if d66.Loc == LocRegPair || d66.Loc == LocStackPair || d66.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d66, &result)
						result.Type = d66.Type
					} else {
						switch d66.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d66)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d66)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d66)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d66, &result)
							result.Type = d66.Type
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
					ctx.ReclaimUntrackedRegs()
					d67 = args[0]
					d67.ID = 0
					d69 = d67
					ctx.SyncDesc(&d69)
					if d69.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d69.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d69.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d69 = tmpScalar
					}
					d69 = JITPrepareScmerGoArg(ctx, d69)
					if d69.Loc != LocRegPair && d69.Loc != LocStackPair && d69.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d68 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d69}, 2)
					ctx.FreeDesc(&d67)
					ctx.EnsureDesc(&d68)
					if d68.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d68.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d68.Imm)
						ptrWord, _ := d68.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d68.Imm.String())))
						d68 = tmpPair
					} else if d68.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d68.Type, Reg: ctx.AllocRegExcept(d68.Reg), Reg2: ctx.AllocRegExcept(d68.Reg)}
						switch d68.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d68)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d68)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d68)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d68)
						d68 = tmpPair
					}
					if d68.Loc != LocRegPair && d68.Loc != LocStackPair && d68.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d68)
					d70 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d68}, 2)
					d70.NoHeapPointer = false
					ctx.BindReg(d70.Reg, &d70)
					ctx.BindReg(d70.Reg2, &d70)
					ctx.StabilizeDescForControlFlow(&d70)
					ctx.EnsureDesc(&d70)
					d71 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MICROSECOND")}
					if d71.Loc == LocImm {
						ctx.TrackImm(d71.Imm)
						ptrWord, _ := d71.Imm.RawWords()
						d72 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d72.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d72.Reg2, uint64(len(d71.Imm.String())))
						ctx.BindReg(d72.Reg, &d72)
						ctx.BindReg(d72.Reg2, &d72)
					} else {
						d72 = d71
					}
					d73 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d72}, 1)
					ctx.EmitAndRegImm32(d73.Reg, 1)
					d73.Type = tagBool
					ctx.BindReg(d73.Reg, &d73)
					d74 = d73
					ctx.EnsureDesc(&d74)
					if d74.Loc != LocImm && d74.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d74.Loc == LocImm {
						if d74.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitCmpRegImm32(d74.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap75 := d1
						snap76 := d2
						snap77 := d3
						snap78 := d4
						snap79 := d5
						snap80 := d6
						snap81 := d7
						snap82 := d16
						snap83 := d17
						snap84 := d19
						snap85 := d20
						snap86 := d21
						snap87 := d23
						snap88 := d24
						snap89 := d25
						snap90 := d42
						snap91 := d43
						snap92 := d44
						snap93 := d45
						snap94 := d66
						snap95 := d67
						snap96 := d68
						snap97 := d69
						snap98 := d70
						snap99 := d71
						snap100 := d72
						snap101 := d73
						snap102 := d74
						alloc103 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc103)
						d1 = snap75
						d2 = snap76
						d3 = snap77
						d4 = snap78
						d5 = snap79
						d6 = snap80
						d7 = snap81
						d16 = snap82
						d17 = snap83
						d19 = snap84
						d20 = snap85
						d21 = snap86
						d23 = snap87
						d24 = snap88
						d25 = snap89
						d42 = snap90
						d43 = snap91
						d44 = snap92
						d45 = snap93
						d66 = snap94
						d67 = snap95
						d68 = snap96
						d69 = snap97
						d70 = snap98
						d71 = snap99
						d72 = snap100
						d73 = snap101
						d74 = snap102
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d73)
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
					ctx.ReclaimUntrackedRegs()
					d104 = d24
					ctx.EnsureDesc(&d104)
					if d104.Loc != LocImm && d104.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d104.Loc == LocImm {
						if d104.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d104.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap105 := d1
						snap106 := d2
						snap107 := d3
						snap108 := d4
						snap109 := d5
						snap110 := d6
						snap111 := d7
						snap112 := d16
						snap113 := d17
						snap114 := d19
						snap115 := d20
						snap116 := d21
						snap117 := d23
						snap118 := d24
						snap119 := d25
						snap120 := d42
						snap121 := d43
						snap122 := d44
						snap123 := d45
						snap124 := d66
						snap125 := d67
						snap126 := d68
						snap127 := d69
						snap128 := d70
						snap129 := d71
						snap130 := d72
						snap131 := d73
						snap132 := d74
						snap133 := d104
						alloc134 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc134)
						d1 = snap105
						d2 = snap106
						d3 = snap107
						d4 = snap108
						d5 = snap109
						d6 = snap110
						d7 = snap111
						d16 = snap112
						d17 = snap113
						d19 = snap114
						d20 = snap115
						d21 = snap116
						d23 = snap117
						d24 = snap118
						d25 = snap119
						d42 = snap120
						d43 = snap121
						d44 = snap122
						d45 = snap123
						d66 = snap124
						d67 = snap125
						d68 = snap126
						d69 = snap127
						d70 = snap128
						d71 = snap129
						d72 = snap130
						d73 = snap131
						d74 = snap132
						d104 = snap133
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d19)
					d135 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d23, d19}, 1)
					d135.NoHeapPointer = true
					ctx.BindReg(d135.Reg, &d135)
					if d135.Loc == LocRegPair || d135.Loc == LocStackPair || d135.Loc == LocRegTriple || d135.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d135)
					d136 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Microseconds), []JITValueDesc{d135}, 1)
					d136.NoHeapPointer = true
					ctx.BindReg(d136.Reg, &d136)
					ctx.FreeDesc(&d135)
					ctx.EnsureDesc(&d136)
					if d136.Loc == LocImm {
						ctx.EmitMakeInt(result, d136)
					} else {
						ctx.EmitMovToReg(result.Reg2, d136)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d137)
						if d136.Loc == LocReg && d136.Reg != result.Reg2 {
							ctx.FreeReg(d136.Reg)
						}
					}
					result.Type = tagInt
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
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d19)
					d138 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d23, d19}, 1)
					d138.NoHeapPointer = true
					ctx.BindReg(d138.Reg, &d138)
					if d138.Loc == LocRegPair || d138.Loc == LocStackPair || d138.Loc == LocRegTriple || d138.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d138)
					d139 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Seconds), []JITValueDesc{d138}, 1)
					d139.NoHeapPointer = true
					ctx.BindReg(d139.Reg, &d139)
					ctx.FreeDesc(&d138)
					ctx.EnsureDesc(&d139)
					ctx.EnsureDesc(&d139)
					if d139.Loc == LocImm {
						d140 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d139.Imm.Float()))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r0, d139.Reg)
						d140 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d140)
					}
					ctx.FreeDesc(&d139)
					ctx.EnsureDesc(&d140)
					if d140.Loc == LocImm {
						ctx.EmitMakeInt(result, d140)
					} else {
						ctx.EmitMovToReg(result.Reg2, d140)
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d141)
						if d140.Loc == LocReg && d140.Reg != result.Reg2 {
							ctx.FreeReg(d140.Reg)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d142 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					if d142.Loc == LocImm {
						ctx.TrackImm(d142.Imm)
						ptrWord, _ := d142.Imm.RawWords()
						d143 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d143.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d143.Reg2, uint64(len(d142.Imm.String())))
						ctx.BindReg(d143.Reg, &d143)
						ctx.BindReg(d143.Reg2, &d143)
					} else {
						d143 = d142
					}
					d144 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d143}, 1)
					ctx.EmitAndRegImm32(d144.Reg, 1)
					d144.Type = tagBool
					ctx.BindReg(d144.Reg, &d144)
					d145 = d144
					ctx.EnsureDesc(&d145)
					if d145.Loc != LocImm && d145.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d145.Loc == LocImm {
						if d145.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitCmpRegImm32(d145.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap146 := d1
						snap147 := d2
						snap148 := d3
						snap149 := d4
						snap150 := d5
						snap151 := d6
						snap152 := d7
						snap153 := d16
						snap154 := d17
						snap155 := d19
						snap156 := d20
						snap157 := d21
						snap158 := d23
						snap159 := d24
						snap160 := d25
						snap161 := d42
						snap162 := d43
						snap163 := d44
						snap164 := d45
						snap165 := d66
						snap166 := d67
						snap167 := d68
						snap168 := d69
						snap169 := d70
						snap170 := d71
						snap171 := d72
						snap172 := d73
						snap173 := d74
						snap174 := d104
						snap175 := d135
						snap176 := d136
						snap177 := d137
						snap178 := d138
						snap179 := d139
						snap180 := d140
						snap181 := d141
						snap182 := d142
						snap183 := d143
						snap184 := d144
						snap185 := d145
						alloc186 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc186)
						d1 = snap146
						d2 = snap147
						d3 = snap148
						d4 = snap149
						d5 = snap150
						d6 = snap151
						d7 = snap152
						d16 = snap153
						d17 = snap154
						d19 = snap155
						d20 = snap156
						d21 = snap157
						d23 = snap158
						d24 = snap159
						d25 = snap160
						d42 = snap161
						d43 = snap162
						d44 = snap163
						d45 = snap164
						d66 = snap165
						d67 = snap166
						d68 = snap167
						d69 = snap168
						d70 = snap169
						d71 = snap170
						d72 = snap171
						d73 = snap172
						d74 = snap173
						d104 = snap174
						d135 = snap175
						d136 = snap176
						d137 = snap177
						d138 = snap178
						d139 = snap179
						d140 = snap180
						d141 = snap181
						d142 = snap182
						d143 = snap183
						d144 = snap184
						d145 = snap185
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d144)
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
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d19)
					d187 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d23, d19}, 1)
					d187.NoHeapPointer = true
					ctx.BindReg(d187.Reg, &d187)
					if d187.Loc == LocRegPair || d187.Loc == LocStackPair || d187.Loc == LocRegTriple || d187.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d187)
					d188 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Minutes), []JITValueDesc{d187}, 1)
					d188.NoHeapPointer = true
					ctx.BindReg(d188.Reg, &d188)
					ctx.FreeDesc(&d187)
					ctx.EnsureDesc(&d188)
					ctx.EnsureDesc(&d188)
					if d188.Loc == LocImm {
						d189 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d188.Imm.Float()))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r1, d188.Reg)
						d189 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d189)
					}
					ctx.FreeDesc(&d188)
					ctx.EnsureDesc(&d189)
					if d189.Loc == LocImm {
						ctx.EmitMakeInt(result, d189)
					} else {
						ctx.EmitMovToReg(result.Reg2, d189)
						d190 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d190)
						if d189.Loc == LocReg && d189.Reg != result.Reg2 {
							ctx.FreeReg(d189.Reg)
						}
					}
					result.Type = tagInt
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d191 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					if d191.Loc == LocImm {
						ctx.TrackImm(d191.Imm)
						ptrWord, _ := d191.Imm.RawWords()
						d192 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d192.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d192.Reg2, uint64(len(d191.Imm.String())))
						ctx.BindReg(d192.Reg, &d192)
						ctx.BindReg(d192.Reg2, &d192)
					} else {
						d192 = d191
					}
					d193 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d192}, 1)
					ctx.EmitAndRegImm32(d193.Reg, 1)
					d193.Type = tagBool
					ctx.BindReg(d193.Reg, &d193)
					d194 = d193
					ctx.EnsureDesc(&d194)
					if d194.Loc != LocImm && d194.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d194.Loc == LocImm {
						if d194.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[13].Render()
					}
					ctx.EmitCmpRegImm32(d194.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
					}
					ctx.FlushRegisterMoves()
					if !bbs[13].Rendered {
						snap195 := d1
						snap196 := d2
						snap197 := d3
						snap198 := d4
						snap199 := d5
						snap200 := d6
						snap201 := d7
						snap202 := d16
						snap203 := d17
						snap204 := d19
						snap205 := d20
						snap206 := d21
						snap207 := d23
						snap208 := d24
						snap209 := d25
						snap210 := d42
						snap211 := d43
						snap212 := d44
						snap213 := d45
						snap214 := d66
						snap215 := d67
						snap216 := d68
						snap217 := d69
						snap218 := d70
						snap219 := d71
						snap220 := d72
						snap221 := d73
						snap222 := d74
						snap223 := d104
						snap224 := d135
						snap225 := d136
						snap226 := d137
						snap227 := d138
						snap228 := d139
						snap229 := d140
						snap230 := d141
						snap231 := d142
						snap232 := d143
						snap233 := d144
						snap234 := d145
						snap235 := d187
						snap236 := d188
						snap237 := d189
						snap238 := d190
						snap239 := d191
						snap240 := d192
						snap241 := d193
						snap242 := d194
						alloc243 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc243)
						d1 = snap195
						d2 = snap196
						d3 = snap197
						d4 = snap198
						d5 = snap199
						d6 = snap200
						d7 = snap201
						d16 = snap202
						d17 = snap203
						d19 = snap204
						d20 = snap205
						d21 = snap206
						d23 = snap207
						d24 = snap208
						d25 = snap209
						d42 = snap210
						d43 = snap211
						d44 = snap212
						d45 = snap213
						d66 = snap214
						d67 = snap215
						d68 = snap216
						d69 = snap217
						d70 = snap218
						d71 = snap219
						d72 = snap220
						d73 = snap221
						d74 = snap222
						d104 = snap223
						d135 = snap224
						d136 = snap225
						d137 = snap226
						d138 = snap227
						d139 = snap228
						d140 = snap229
						d141 = snap230
						d142 = snap231
						d143 = snap232
						d144 = snap233
						d145 = snap234
						d187 = snap235
						d188 = snap236
						d189 = snap237
						d190 = snap238
						d191 = snap239
						d192 = snap240
						d193 = snap241
						d194 = snap242
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					ctx.FreeDesc(&d193)
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
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d19)
					d244 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d23, d19}, 1)
					d244.NoHeapPointer = true
					ctx.BindReg(d244.Reg, &d244)
					if d244.Loc == LocRegPair || d244.Loc == LocStackPair || d244.Loc == LocRegTriple || d244.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d244)
					d245 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d244}, 1)
					d245.NoHeapPointer = true
					ctx.BindReg(d245.Reg, &d245)
					ctx.FreeDesc(&d244)
					ctx.EnsureDesc(&d245)
					ctx.EnsureDesc(&d245)
					if d245.Loc == LocImm {
						d246 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d245.Imm.Float()))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r2, d245.Reg)
						d246 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d246)
					}
					ctx.FreeDesc(&d245)
					ctx.EnsureDesc(&d246)
					if d246.Loc == LocImm {
						ctx.EmitMakeInt(result, d246)
					} else {
						ctx.EmitMovToReg(result.Reg2, d246)
						d247 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d247)
						if d246.Loc == LocReg && d246.Reg != result.Reg2 {
							ctx.FreeReg(d246.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d248 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					if d248.Loc == LocImm {
						ctx.TrackImm(d248.Imm)
						ptrWord, _ := d248.Imm.RawWords()
						d249 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d249.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d249.Reg2, uint64(len(d248.Imm.String())))
						ctx.BindReg(d249.Reg, &d249)
						ctx.BindReg(d249.Reg2, &d249)
					} else {
						d249 = d248
					}
					d250 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d249}, 1)
					ctx.EmitAndRegImm32(d250.Reg, 1)
					d250.Type = tagBool
					ctx.BindReg(d250.Reg, &d250)
					d251 = d250
					ctx.EnsureDesc(&d251)
					if d251.Loc != LocImm && d251.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d251.Loc == LocImm {
						if d251.Imm.Bool() {
							return bbs[12].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitCmpRegImm32(d251.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap252 := d1
						snap253 := d2
						snap254 := d3
						snap255 := d4
						snap256 := d5
						snap257 := d6
						snap258 := d7
						snap259 := d16
						snap260 := d17
						snap261 := d19
						snap262 := d20
						snap263 := d21
						snap264 := d23
						snap265 := d24
						snap266 := d25
						snap267 := d42
						snap268 := d43
						snap269 := d44
						snap270 := d45
						snap271 := d66
						snap272 := d67
						snap273 := d68
						snap274 := d69
						snap275 := d70
						snap276 := d71
						snap277 := d72
						snap278 := d73
						snap279 := d74
						snap280 := d104
						snap281 := d135
						snap282 := d136
						snap283 := d137
						snap284 := d138
						snap285 := d139
						snap286 := d140
						snap287 := d141
						snap288 := d142
						snap289 := d143
						snap290 := d144
						snap291 := d145
						snap292 := d187
						snap293 := d188
						snap294 := d189
						snap295 := d190
						snap296 := d191
						snap297 := d192
						snap298 := d193
						snap299 := d194
						snap300 := d244
						snap301 := d245
						snap302 := d246
						snap303 := d247
						snap304 := d248
						snap305 := d249
						snap306 := d250
						snap307 := d251
						alloc308 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc308)
						d1 = snap252
						d2 = snap253
						d3 = snap254
						d4 = snap255
						d5 = snap256
						d6 = snap257
						d7 = snap258
						d16 = snap259
						d17 = snap260
						d19 = snap261
						d20 = snap262
						d21 = snap263
						d23 = snap264
						d24 = snap265
						d25 = snap266
						d42 = snap267
						d43 = snap268
						d44 = snap269
						d45 = snap270
						d66 = snap271
						d67 = snap272
						d68 = snap273
						d69 = snap274
						d70 = snap275
						d71 = snap276
						d72 = snap277
						d73 = snap278
						d74 = snap279
						d104 = snap280
						d135 = snap281
						d136 = snap282
						d137 = snap283
						d138 = snap284
						d139 = snap285
						d140 = snap286
						d141 = snap287
						d142 = snap288
						d143 = snap289
						d144 = snap290
						d145 = snap291
						d187 = snap292
						d188 = snap293
						d189 = snap294
						d190 = snap295
						d191 = snap296
						d192 = snap297
						d193 = snap298
						d194 = snap299
						d244 = snap300
						d245 = snap301
						d246 = snap302
						d247 = snap303
						d248 = snap304
						d249 = snap305
						d250 = snap306
						d251 = snap307
					}
					if !bbs[12].Rendered {
						return bbs[12].Render()
					}
					return result
					ctx.FreeDesc(&d250)
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
					ctx.ReclaimUntrackedRegs()
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d19)
					d309 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d19}, 1)
					d309.NoHeapPointer = true
					ctx.BindReg(d309.Reg, &d309)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d19)
					d310 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d19}, 1)
					d310.NoHeapPointer = true
					ctx.BindReg(d310.Reg, &d310)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d19)
					d311 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d19}, 1)
					d311.NoHeapPointer = true
					ctx.BindReg(d311.Reg, &d311)
					d312 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d309.Loc == LocRegPair || d309.Loc == LocStackPair || d309.Loc == LocRegTriple || d309.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d310.Loc == LocRegPair || d310.Loc == LocStackPair || d310.Loc == LocRegTriple || d310.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d311.Loc == LocRegPair || d311.Loc == LocStackPair || d311.Loc == LocRegTriple || d311.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d313 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d313.Loc == LocRegPair || d313.Loc == LocStackPair || d313.Loc == LocRegTriple || d313.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d314 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d314.Loc == LocRegPair || d314.Loc == LocStackPair || d314.Loc == LocRegTriple || d314.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d315 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d315.Loc == LocRegPair || d315.Loc == LocStackPair || d315.Loc == LocRegTriple || d315.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d316 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d316.Loc == LocRegPair || d316.Loc == LocStackPair || d316.Loc == LocRegTriple || d316.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d312.Loc == LocRegPair || d312.Loc == LocStackPair || d312.Loc == LocRegTriple || d312.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d309)
					ctx.SyncDesc(&d310)
					ctx.SyncDesc(&d311)
					ctx.SyncDesc(&d313)
					ctx.SyncDesc(&d314)
					ctx.SyncDesc(&d315)
					ctx.SyncDesc(&d316)
					ctx.SyncDesc(&d312)
					d317 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d309, d310, d311, d313, d314, d315, d316, d312}, 3)
					d317.NoHeapPointer = false
					ctx.BindReg(d317.Reg, &d317)
					ctx.BindReg(d317.Reg2, &d317)
					ctx.BindReg(d317.Reg3, &d317)
					ctx.FreeDesc(&d313)
					ctx.FreeDesc(&d314)
					ctx.FreeDesc(&d315)
					ctx.FreeDesc(&d316)
					ctx.FreeDesc(&d309)
					ctx.FreeDesc(&d310)
					ctx.FreeDesc(&d311)
					ctx.FreeDesc(&d312)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d23)
					d318 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d23}, 1)
					d318.NoHeapPointer = true
					ctx.BindReg(d318.Reg, &d318)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d23)
					d319 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d23}, 1)
					d319.NoHeapPointer = true
					ctx.BindReg(d319.Reg, &d319)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d23)
					d320 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d23}, 1)
					d320.NoHeapPointer = true
					ctx.BindReg(d320.Reg, &d320)
					d321 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d318.Loc == LocRegPair || d318.Loc == LocStackPair || d318.Loc == LocRegTriple || d318.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d319.Loc == LocRegPair || d319.Loc == LocStackPair || d319.Loc == LocRegTriple || d319.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d320.Loc == LocRegPair || d320.Loc == LocStackPair || d320.Loc == LocRegTriple || d320.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d322 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d322.Loc == LocRegPair || d322.Loc == LocStackPair || d322.Loc == LocRegTriple || d322.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d323 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d323.Loc == LocRegPair || d323.Loc == LocStackPair || d323.Loc == LocRegTriple || d323.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d324 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d324.Loc == LocRegPair || d324.Loc == LocStackPair || d324.Loc == LocRegTriple || d324.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d325 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d325.Loc == LocRegPair || d325.Loc == LocStackPair || d325.Loc == LocRegTriple || d325.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d321.Loc == LocRegPair || d321.Loc == LocStackPair || d321.Loc == LocRegTriple || d321.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d318)
					ctx.SyncDesc(&d319)
					ctx.SyncDesc(&d320)
					ctx.SyncDesc(&d322)
					ctx.SyncDesc(&d323)
					ctx.SyncDesc(&d324)
					ctx.SyncDesc(&d325)
					ctx.SyncDesc(&d321)
					d326 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d318, d319, d320, d322, d323, d324, d325, d321}, 3)
					d326.NoHeapPointer = false
					ctx.BindReg(d326.Reg, &d326)
					ctx.BindReg(d326.Reg2, &d326)
					ctx.BindReg(d326.Reg3, &d326)
					ctx.FreeDesc(&d322)
					ctx.FreeDesc(&d323)
					ctx.FreeDesc(&d324)
					ctx.FreeDesc(&d325)
					ctx.FreeDesc(&d318)
					ctx.FreeDesc(&d319)
					ctx.FreeDesc(&d320)
					ctx.FreeDesc(&d321)
					d326 = JITPrepareGoSliceArg(ctx, d326)
					if d326.Loc != LocRegTriple && d326.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d317 = JITPrepareGoSliceArg(ctx, d317)
					if d317.Loc != LocRegTriple && d317.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d326)
					ctx.SyncDesc(&d317)
					d327 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d326, d317}, 1)
					d327.NoHeapPointer = true
					ctx.BindReg(d327.Reg, &d327)
					ctx.FreeDesc(&d326)
					ctx.FreeDesc(&d317)
					if d327.Loc == LocRegPair || d327.Loc == LocStackPair || d327.Loc == LocRegTriple || d327.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d327)
					d328 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d327}, 1)
					d328.NoHeapPointer = true
					ctx.BindReg(d328.Reg, &d328)
					ctx.FreeDesc(&d327)
					ctx.EnsureDesc(&d328)
					if d328.Loc == LocImm {
						d329 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d328.Imm.Float() / 24)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4627448617123184640))
						ctx.EmitDivFloat64(d328.Reg, ctx.ScratchReg)
						d329 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d328.Reg}
						ctx.BindReg(d328.Reg, &d329)
					}
					if d329.Loc == LocReg && d328.Loc == LocReg && d329.Reg == d328.Reg {
						ctx.TransferReg(d328.Reg)
						d328.Loc = LocNone
					}
					ctx.FreeDesc(&d328)
					ctx.EnsureDesc(&d329)
					ctx.EnsureDesc(&d329)
					if d329.Loc == LocImm {
						d330 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d329.Imm.Float()))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r3, d329.Reg)
						d330 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d330)
					}
					ctx.FreeDesc(&d329)
					ctx.EnsureDesc(&d330)
					if d330.Loc == LocImm {
						ctx.EmitMakeInt(result, d330)
					} else {
						ctx.EmitMovToReg(result.Reg2, d330)
						d331 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d331)
						if d330.Loc == LocReg && d330.Reg != result.Reg2 {
							ctx.FreeReg(d330.Reg)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d332 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
					if d332.Loc == LocImm {
						ctx.TrackImm(d332.Imm)
						ptrWord, _ := d332.Imm.RawWords()
						d333 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d333.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d333.Reg2, uint64(len(d332.Imm.String())))
						ctx.BindReg(d333.Reg, &d333)
						ctx.BindReg(d333.Reg2, &d333)
					} else {
						d333 = d332
					}
					d334 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d333}, 1)
					ctx.EmitAndRegImm32(d334.Reg, 1)
					d334.Type = tagBool
					ctx.BindReg(d334.Reg, &d334)
					d335 = d334
					ctx.EnsureDesc(&d335)
					if d335.Loc != LocImm && d335.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d335.Loc == LocImm {
						if d335.Imm.Bool() {
							return bbs[14].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitCmpRegImm32(d335.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap336 := d1
						snap337 := d2
						snap338 := d3
						snap339 := d4
						snap340 := d5
						snap341 := d6
						snap342 := d7
						snap343 := d16
						snap344 := d17
						snap345 := d19
						snap346 := d20
						snap347 := d21
						snap348 := d23
						snap349 := d24
						snap350 := d25
						snap351 := d42
						snap352 := d43
						snap353 := d44
						snap354 := d45
						snap355 := d66
						snap356 := d67
						snap357 := d68
						snap358 := d69
						snap359 := d70
						snap360 := d71
						snap361 := d72
						snap362 := d73
						snap363 := d74
						snap364 := d104
						snap365 := d135
						snap366 := d136
						snap367 := d137
						snap368 := d138
						snap369 := d139
						snap370 := d140
						snap371 := d141
						snap372 := d142
						snap373 := d143
						snap374 := d144
						snap375 := d145
						snap376 := d187
						snap377 := d188
						snap378 := d189
						snap379 := d190
						snap380 := d191
						snap381 := d192
						snap382 := d193
						snap383 := d194
						snap384 := d244
						snap385 := d245
						snap386 := d246
						snap387 := d247
						snap388 := d248
						snap389 := d249
						snap390 := d250
						snap391 := d251
						snap392 := d309
						snap393 := d310
						snap394 := d311
						snap395 := d312
						snap396 := d313
						snap397 := d314
						snap398 := d315
						snap399 := d316
						snap400 := d317
						snap401 := d318
						snap402 := d319
						snap403 := d320
						snap404 := d321
						snap405 := d322
						snap406 := d323
						snap407 := d324
						snap408 := d325
						snap409 := d326
						snap410 := d327
						snap411 := d328
						snap412 := d329
						snap413 := d330
						snap414 := d331
						snap415 := d332
						snap416 := d333
						snap417 := d334
						snap418 := d335
						alloc419 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc419)
						d1 = snap336
						d2 = snap337
						d3 = snap338
						d4 = snap339
						d5 = snap340
						d6 = snap341
						d7 = snap342
						d16 = snap343
						d17 = snap344
						d19 = snap345
						d20 = snap346
						d21 = snap347
						d23 = snap348
						d24 = snap349
						d25 = snap350
						d42 = snap351
						d43 = snap352
						d44 = snap353
						d45 = snap354
						d66 = snap355
						d67 = snap356
						d68 = snap357
						d69 = snap358
						d70 = snap359
						d71 = snap360
						d72 = snap361
						d73 = snap362
						d74 = snap363
						d104 = snap364
						d135 = snap365
						d136 = snap366
						d137 = snap367
						d138 = snap368
						d139 = snap369
						d140 = snap370
						d141 = snap371
						d142 = snap372
						d143 = snap373
						d144 = snap374
						d145 = snap375
						d187 = snap376
						d188 = snap377
						d189 = snap378
						d190 = snap379
						d191 = snap380
						d192 = snap381
						d193 = snap382
						d194 = snap383
						d244 = snap384
						d245 = snap385
						d246 = snap386
						d247 = snap387
						d248 = snap388
						d249 = snap389
						d250 = snap390
						d251 = snap391
						d309 = snap392
						d310 = snap393
						d311 = snap394
						d312 = snap395
						d313 = snap396
						d314 = snap397
						d315 = snap398
						d316 = snap399
						d317 = snap400
						d318 = snap401
						d319 = snap402
						d320 = snap403
						d321 = snap404
						d322 = snap405
						d323 = snap406
						d324 = snap407
						d325 = snap408
						d326 = snap409
						d327 = snap410
						d328 = snap411
						d329 = snap412
						d330 = snap413
						d331 = snap414
						d332 = snap415
						d333 = snap416
						d334 = snap417
						d335 = snap418
					}
					if !bbs[14].Rendered {
						return bbs[14].Render()
					}
					return result
					ctx.FreeDesc(&d334)
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
					ctx.ReclaimUntrackedRegs()
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d19)
					d420 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d19}, 1)
					d420.NoHeapPointer = true
					ctx.BindReg(d420.Reg, &d420)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d19)
					d421 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d19}, 1)
					d421.NoHeapPointer = true
					ctx.BindReg(d421.Reg, &d421)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d19)
					d422 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d19}, 1)
					d422.NoHeapPointer = true
					ctx.BindReg(d422.Reg, &d422)
					d423 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d420.Loc == LocRegPair || d420.Loc == LocStackPair || d420.Loc == LocRegTriple || d420.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d421.Loc == LocRegPair || d421.Loc == LocStackPair || d421.Loc == LocRegTriple || d421.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d422.Loc == LocRegPair || d422.Loc == LocStackPair || d422.Loc == LocRegTriple || d422.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d424 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d424.Loc == LocRegPair || d424.Loc == LocStackPair || d424.Loc == LocRegTriple || d424.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d425 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d425.Loc == LocRegPair || d425.Loc == LocStackPair || d425.Loc == LocRegTriple || d425.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d426 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d426.Loc == LocRegPair || d426.Loc == LocStackPair || d426.Loc == LocRegTriple || d426.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d427 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d427.Loc == LocRegPair || d427.Loc == LocStackPair || d427.Loc == LocRegTriple || d427.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d423.Loc == LocRegPair || d423.Loc == LocStackPair || d423.Loc == LocRegTriple || d423.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d420)
					ctx.SyncDesc(&d421)
					ctx.SyncDesc(&d422)
					ctx.SyncDesc(&d424)
					ctx.SyncDesc(&d425)
					ctx.SyncDesc(&d426)
					ctx.SyncDesc(&d427)
					ctx.SyncDesc(&d423)
					d428 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d420, d421, d422, d424, d425, d426, d427, d423}, 3)
					d428.NoHeapPointer = false
					ctx.BindReg(d428.Reg, &d428)
					ctx.BindReg(d428.Reg2, &d428)
					ctx.BindReg(d428.Reg3, &d428)
					ctx.FreeDesc(&d424)
					ctx.FreeDesc(&d425)
					ctx.FreeDesc(&d426)
					ctx.FreeDesc(&d427)
					ctx.FreeDesc(&d420)
					ctx.FreeDesc(&d421)
					ctx.FreeDesc(&d422)
					ctx.FreeDesc(&d423)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d23)
					d429 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d23}, 1)
					d429.NoHeapPointer = true
					ctx.BindReg(d429.Reg, &d429)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d23)
					d430 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d23}, 1)
					d430.NoHeapPointer = true
					ctx.BindReg(d430.Reg, &d430)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d23)
					d431 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d23}, 1)
					d431.NoHeapPointer = true
					ctx.BindReg(d431.Reg, &d431)
					d432 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d429.Loc == LocRegPair || d429.Loc == LocStackPair || d429.Loc == LocRegTriple || d429.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d430.Loc == LocRegPair || d430.Loc == LocStackPair || d430.Loc == LocRegTriple || d430.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d431.Loc == LocRegPair || d431.Loc == LocStackPair || d431.Loc == LocRegTriple || d431.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d433 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d433.Loc == LocRegPair || d433.Loc == LocStackPair || d433.Loc == LocRegTriple || d433.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d434 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d434.Loc == LocRegPair || d434.Loc == LocStackPair || d434.Loc == LocRegTriple || d434.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d435 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d435.Loc == LocRegPair || d435.Loc == LocStackPair || d435.Loc == LocRegTriple || d435.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d436 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d436.Loc == LocRegPair || d436.Loc == LocStackPair || d436.Loc == LocRegTriple || d436.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d432.Loc == LocRegPair || d432.Loc == LocStackPair || d432.Loc == LocRegTriple || d432.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d429)
					ctx.SyncDesc(&d430)
					ctx.SyncDesc(&d431)
					ctx.SyncDesc(&d433)
					ctx.SyncDesc(&d434)
					ctx.SyncDesc(&d435)
					ctx.SyncDesc(&d436)
					ctx.SyncDesc(&d432)
					d437 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d429, d430, d431, d433, d434, d435, d436, d432}, 3)
					d437.NoHeapPointer = false
					ctx.BindReg(d437.Reg, &d437)
					ctx.BindReg(d437.Reg2, &d437)
					ctx.BindReg(d437.Reg3, &d437)
					ctx.FreeDesc(&d433)
					ctx.FreeDesc(&d434)
					ctx.FreeDesc(&d435)
					ctx.FreeDesc(&d436)
					ctx.FreeDesc(&d429)
					ctx.FreeDesc(&d430)
					ctx.FreeDesc(&d431)
					ctx.FreeDesc(&d432)
					d437 = JITPrepareGoSliceArg(ctx, d437)
					if d437.Loc != LocRegTriple && d437.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d428 = JITPrepareGoSliceArg(ctx, d428)
					if d428.Loc != LocRegTriple && d428.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d437)
					ctx.SyncDesc(&d428)
					d438 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d437, d428}, 1)
					d438.NoHeapPointer = true
					ctx.BindReg(d438.Reg, &d438)
					ctx.FreeDesc(&d437)
					ctx.FreeDesc(&d428)
					if d438.Loc == LocRegPair || d438.Loc == LocStackPair || d438.Loc == LocRegTriple || d438.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d438)
					d439 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d438}, 1)
					d439.NoHeapPointer = true
					ctx.BindReg(d439.Reg, &d439)
					ctx.FreeDesc(&d438)
					ctx.EnsureDesc(&d439)
					if d439.Loc == LocImm {
						d440 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d439.Imm.Float() / 24)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4627448617123184640))
						ctx.EmitDivFloat64(d439.Reg, ctx.ScratchReg)
						d440 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d439.Reg}
						ctx.BindReg(d439.Reg, &d440)
					}
					if d440.Loc == LocReg && d439.Loc == LocReg && d440.Reg == d439.Reg {
						ctx.TransferReg(d439.Reg)
						d439.Loc = LocNone
					}
					ctx.FreeDesc(&d439)
					ctx.EnsureDesc(&d440)
					if d440.Loc == LocImm {
						d441 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d440.Imm.Float() / 7)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4619567317775286272))
						ctx.EmitDivFloat64(d440.Reg, ctx.ScratchReg)
						d441 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d440.Reg}
						ctx.BindReg(d440.Reg, &d441)
					}
					if d441.Loc == LocReg && d440.Loc == LocReg && d441.Reg == d440.Reg {
						ctx.TransferReg(d440.Reg)
						d440.Loc = LocNone
					}
					ctx.FreeDesc(&d440)
					ctx.EnsureDesc(&d441)
					ctx.EnsureDesc(&d441)
					if d441.Loc == LocImm {
						d442 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d441.Imm.Float()))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r4, d441.Reg)
						d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d442)
					}
					ctx.FreeDesc(&d441)
					ctx.EnsureDesc(&d442)
					if d442.Loc == LocImm {
						ctx.EmitMakeInt(result, d442)
					} else {
						ctx.EmitMovToReg(result.Reg2, d442)
						d443 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d443)
						if d442.Loc == LocReg && d442.Reg != result.Reg2 {
							ctx.FreeReg(d442.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d444 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					if d444.Loc == LocImm {
						ctx.TrackImm(d444.Imm)
						ptrWord, _ := d444.Imm.RawWords()
						d445 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d445.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d445.Reg2, uint64(len(d444.Imm.String())))
						ctx.BindReg(d445.Reg, &d445)
						ctx.BindReg(d445.Reg2, &d445)
					} else {
						d445 = d444
					}
					d446 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d445}, 1)
					ctx.EmitAndRegImm32(d446.Reg, 1)
					d446.Type = tagBool
					ctx.BindReg(d446.Reg, &d446)
					d447 = d446
					ctx.EnsureDesc(&d447)
					if d447.Loc != LocImm && d447.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d447.Loc == LocImm {
						if d447.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d447.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap448 := d1
						snap449 := d2
						snap450 := d3
						snap451 := d4
						snap452 := d5
						snap453 := d6
						snap454 := d7
						snap455 := d16
						snap456 := d17
						snap457 := d19
						snap458 := d20
						snap459 := d21
						snap460 := d23
						snap461 := d24
						snap462 := d25
						snap463 := d42
						snap464 := d43
						snap465 := d44
						snap466 := d45
						snap467 := d66
						snap468 := d67
						snap469 := d68
						snap470 := d69
						snap471 := d70
						snap472 := d71
						snap473 := d72
						snap474 := d73
						snap475 := d74
						snap476 := d104
						snap477 := d135
						snap478 := d136
						snap479 := d137
						snap480 := d138
						snap481 := d139
						snap482 := d140
						snap483 := d141
						snap484 := d142
						snap485 := d143
						snap486 := d144
						snap487 := d145
						snap488 := d187
						snap489 := d188
						snap490 := d189
						snap491 := d190
						snap492 := d191
						snap493 := d192
						snap494 := d193
						snap495 := d194
						snap496 := d244
						snap497 := d245
						snap498 := d246
						snap499 := d247
						snap500 := d248
						snap501 := d249
						snap502 := d250
						snap503 := d251
						snap504 := d309
						snap505 := d310
						snap506 := d311
						snap507 := d312
						snap508 := d313
						snap509 := d314
						snap510 := d315
						snap511 := d316
						snap512 := d317
						snap513 := d318
						snap514 := d319
						snap515 := d320
						snap516 := d321
						snap517 := d322
						snap518 := d323
						snap519 := d324
						snap520 := d325
						snap521 := d326
						snap522 := d327
						snap523 := d328
						snap524 := d329
						snap525 := d330
						snap526 := d331
						snap527 := d332
						snap528 := d333
						snap529 := d334
						snap530 := d335
						snap531 := d420
						snap532 := d421
						snap533 := d422
						snap534 := d423
						snap535 := d424
						snap536 := d425
						snap537 := d426
						snap538 := d427
						snap539 := d428
						snap540 := d429
						snap541 := d430
						snap542 := d431
						snap543 := d432
						snap544 := d433
						snap545 := d434
						snap546 := d435
						snap547 := d436
						snap548 := d437
						snap549 := d438
						snap550 := d439
						snap551 := d440
						snap552 := d441
						snap553 := d442
						snap554 := d443
						snap555 := d444
						snap556 := d445
						snap557 := d446
						snap558 := d447
						alloc559 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc559)
						d1 = snap448
						d2 = snap449
						d3 = snap450
						d4 = snap451
						d5 = snap452
						d6 = snap453
						d7 = snap454
						d16 = snap455
						d17 = snap456
						d19 = snap457
						d20 = snap458
						d21 = snap459
						d23 = snap460
						d24 = snap461
						d25 = snap462
						d42 = snap463
						d43 = snap464
						d44 = snap465
						d45 = snap466
						d66 = snap467
						d67 = snap468
						d68 = snap469
						d69 = snap470
						d70 = snap471
						d71 = snap472
						d72 = snap473
						d73 = snap474
						d74 = snap475
						d104 = snap476
						d135 = snap477
						d136 = snap478
						d137 = snap479
						d138 = snap480
						d139 = snap481
						d140 = snap482
						d141 = snap483
						d142 = snap484
						d143 = snap485
						d144 = snap486
						d145 = snap487
						d187 = snap488
						d188 = snap489
						d189 = snap490
						d190 = snap491
						d191 = snap492
						d192 = snap493
						d193 = snap494
						d194 = snap495
						d244 = snap496
						d245 = snap497
						d246 = snap498
						d247 = snap499
						d248 = snap500
						d249 = snap501
						d250 = snap502
						d251 = snap503
						d309 = snap504
						d310 = snap505
						d311 = snap506
						d312 = snap507
						d313 = snap508
						d314 = snap509
						d315 = snap510
						d316 = snap511
						d317 = snap512
						d318 = snap513
						d319 = snap514
						d320 = snap515
						d321 = snap516
						d322 = snap517
						d323 = snap518
						d324 = snap519
						d325 = snap520
						d326 = snap521
						d327 = snap522
						d328 = snap523
						d329 = snap524
						d330 = snap525
						d331 = snap526
						d332 = snap527
						d333 = snap528
						d334 = snap529
						d335 = snap530
						d420 = snap531
						d421 = snap532
						d422 = snap533
						d423 = snap534
						d424 = snap535
						d425 = snap536
						d426 = snap537
						d427 = snap538
						d428 = snap539
						d429 = snap540
						d430 = snap541
						d431 = snap542
						d432 = snap543
						d433 = snap544
						d434 = snap545
						d435 = snap546
						d436 = snap547
						d437 = snap548
						d438 = snap549
						d439 = snap550
						d440 = snap551
						d441 = snap552
						d442 = snap553
						d443 = snap554
						d444 = snap555
						d445 = snap556
						d446 = snap557
						d447 = snap558
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d446)
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
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d23)
					d560 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d23}, 1)
					d560.NoHeapPointer = true
					ctx.BindReg(d560.Reg, &d560)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d19)
					d561 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d19}, 1)
					d561.NoHeapPointer = true
					ctx.BindReg(d561.Reg, &d561)
					ctx.EnsureDesc(&d560)
					ctx.EnsureDesc(&d561)
					ctx.SyncDesc(&d560)
					ctx.SyncDesc(&d561)
					if d560.Loc == LocImm && d561.Loc == LocImm {
						d562 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d560.Imm.Int() - d561.Imm.Int())}
					} else if d561.Loc == LocImm && d561.Imm.Int() == 0 {
						ctx.EnsureDesc(&d560)
						r5 := ctx.AllocRegExcept(d560.Reg)
						ctx.EmitMovRegReg(r5, d560.Reg)
						d562 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d562)
					} else if d560.Loc == LocImm {
						ctx.EnsureDesc(&d561)
						scratch := ctx.AllocRegExcept(d561.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d560.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d561)
						d562 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d562)
					} else if d561.Loc == LocImm {
						ctx.EnsureDesc(&d560)
						scratch := ctx.AllocRegExcept(d560.Reg)
						ctx.EmitMovRegReg(scratch, d560.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d561.Imm.Int())
						d562 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d562)
					} else {
						ctx.EnsureDesc(&d560)
						ctx.SyncDesc(&d561)
						r6 := ctx.AllocRegExceptOperand(&d561, d560.Reg)
						ctx.EmitMovRegReg(r6, d560.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r6, &d561)
						d562 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d562)
					}
					if d562.Loc == LocReg && d560.Loc == LocReg && d562.Reg == d560.Reg {
						ctx.TransferReg(d560.Reg)
						d560.Loc = LocNone
					}
					ctx.FreeDesc(&d560)
					ctx.FreeDesc(&d561)
					ctx.EnsureDesc(&d562)
					ctx.EnsureDesc(&d562)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d23)
					d564 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d23}, 1)
					d564.NoHeapPointer = true
					ctx.BindReg(d564.Reg, &d564)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d19)
					d565 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d19}, 1)
					d565.NoHeapPointer = true
					ctx.BindReg(d565.Reg, &d565)
					ctx.EnsureDesc(&d564)
					ctx.EnsureDesc(&d565)
					ctx.SyncDesc(&d564)
					ctx.SyncDesc(&d565)
					if d564.Loc == LocImm && d565.Loc == LocImm {
						d566 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d564.Imm.Int() - d565.Imm.Int())}
					} else if d565.Loc == LocImm && d565.Imm.Int() == 0 {
						ctx.EnsureDesc(&d564)
						r7 := ctx.AllocRegExcept(d564.Reg)
						ctx.EmitMovRegReg(r7, d564.Reg)
						d566 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d566)
					} else if d564.Loc == LocImm {
						ctx.EnsureDesc(&d565)
						scratch := ctx.AllocRegExcept(d565.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d564.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d565)
						d566 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d566)
					} else if d565.Loc == LocImm {
						ctx.EnsureDesc(&d564)
						scratch := ctx.AllocRegExcept(d564.Reg)
						ctx.EmitMovRegReg(scratch, d564.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d565.Imm.Int())
						d566 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d566)
					} else {
						ctx.EnsureDesc(&d564)
						ctx.SyncDesc(&d565)
						r8 := ctx.AllocRegExceptOperand(&d565, d564.Reg)
						ctx.EmitMovRegReg(r8, d564.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r8, &d565)
						d566 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d566)
					}
					if d566.Loc == LocReg && d564.Loc == LocReg && d566.Reg == d564.Reg {
						ctx.TransferReg(d564.Reg)
						d564.Loc = LocNone
					}
					ctx.FreeDesc(&d564)
					ctx.FreeDesc(&d565)
					ctx.EnsureDesc(&d566)
					ctx.EnsureDesc(&d566)
					ctx.EnsureDesc(&d562)
					ctx.EnsureDesc(&d562)
					if d562.Loc == LocImm {
						d568 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d562.Imm.Int() * 12)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d562.Reg, 12)
						d568 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d562.Reg}
						ctx.BindReg(d562.Reg, &d568)
					}
					if d568.Loc == LocReg && d562.Loc == LocReg && d568.Reg == d562.Reg {
						ctx.TransferReg(d562.Reg)
						d562.Loc = LocNone
					}
					ctx.FreeDesc(&d562)
					ctx.EnsureDesc(&d568)
					ctx.EnsureDesc(&d566)
					ctx.SyncDesc(&d568)
					ctx.SyncDesc(&d566)
					if d568.Loc == LocImm && d566.Loc == LocImm {
						d569 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d568.Imm.Int() + d566.Imm.Int())}
					} else if d566.Loc == LocImm && d566.Imm.Int() == 0 {
						ctx.EnsureDesc(&d568)
						r9 := ctx.AllocRegExcept(d568.Reg)
						ctx.EmitMovRegReg(r9, d568.Reg)
						d569 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d569)
					} else if d568.Loc == LocImm && d568.Imm.Int() == 0 {
						ctx.EnsureDesc(&d566)
						d569 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d566.Reg}
						ctx.BindReg(d566.Reg, &d569)
					} else if d568.Loc == LocImm {
						ctx.EnsureDesc(&d566)
						scratch := ctx.AllocRegExcept(d566.Reg)
						ctx.EmitMovRegReg(scratch, d566.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d568.Imm.Int())
						d569 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d569)
					} else if d566.Loc == LocImm {
						ctx.EnsureDesc(&d568)
						scratch := ctx.AllocRegExcept(d568.Reg)
						ctx.EmitMovRegReg(scratch, d568.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d566.Imm.Int())
						d569 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d569)
					} else {
						ctx.EnsureDesc(&d568)
						ctx.SyncDesc(&d566)
						r10 := ctx.AllocRegExceptOperand(&d566, d568.Reg)
						ctx.EmitMovRegReg(r10, d568.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r10, &d566)
						d569 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d569)
					}
					if d569.Loc == LocReg && d568.Loc == LocReg && d569.Reg == d568.Reg {
						ctx.TransferReg(d568.Reg)
						d568.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d569)
					ctx.FreeDesc(&d568)
					ctx.FreeDesc(&d566)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d23)
					d570 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d23}, 1)
					d570.NoHeapPointer = true
					ctx.BindReg(d570.Reg, &d570)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d19)
					d571 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d19}, 1)
					d571.NoHeapPointer = true
					ctx.BindReg(d571.Reg, &d571)
					ctx.EnsureDesc(&d570)
					ctx.EnsureDesc(&d571)
					ctx.EnsureDescsTogether(&d570, &d571)
					if d570.Loc == LocImm && d571.Loc == LocImm {
						d572 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d570.Imm.Int() < d571.Imm.Int())}
					} else if d571.Loc == LocImm {
						r11 := ctx.AllocReg()
						if d571.Imm.Int() >= -2147483648 && d571.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d570.Reg, int32(d571.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d571.Imm.Int()))
							ctx.EmitCmpInt64(d570.Reg, ctx.ScratchReg)
						}
						d572 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
						ctx.BindReg(r11, &d572)
					} else if d570.Loc == LocImm {
						r12 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d570.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d571.Reg)
						d572 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
						ctx.BindReg(r12, &d572)
					} else {
						r13 := ctx.AllocReg()
						ctx.EmitCmpInt64(d570.Reg, d571.Reg)
						d572 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d572)
					}
					ctx.FreeDesc(&d570)
					ctx.FreeDesc(&d571)
					d573 = d572
					ctx.EnsureDesc(&d573)
					if d573.Loc != LocImm && d573.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d573.Loc == LocImm {
						if d573.Imm.Bool() {
							return bbs[22].Render()
						}
						ctx.SyncDesc(&d569)
						if d569.Loc == LocReg || d569.Loc == LocFPReg {
							ctx.ProtectReg(d569.Reg)
						} else if d569.Loc == LocRegPair {
							ctx.ProtectReg(d569.Reg)
							ctx.ProtectReg(d569.Reg2)
						}
						d574 = d569
						if d574.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d574)
						ctx.EmitStoreToStack(d574, int32(bbs[23].PhiBase)+int32(0))
						if d569.Loc == LocReg || d569.Loc == LocFPReg {
							ctx.UnprotectReg(d569.Reg)
						} else if d569.Loc == LocRegPair {
							ctx.UnprotectReg(d569.Reg)
							ctx.UnprotectReg(d569.Reg2)
						}
						return bbs[23].Render()
					}
					lbl34 := ctx.ReserveLabel()
					ctx.EmitJump(d573.Condition, lbl23)
					ctx.EmitJmp(lbl34)
					ctx.FreeDesc(&d572)
					snap575 := d1
					snap576 := d2
					snap577 := d3
					snap578 := d4
					snap579 := d5
					snap580 := d6
					snap581 := d7
					snap582 := d16
					snap583 := d17
					snap584 := d19
					snap585 := d20
					snap586 := d21
					snap587 := d23
					snap588 := d24
					snap589 := d25
					snap590 := d42
					snap591 := d43
					snap592 := d44
					snap593 := d45
					snap594 := d66
					snap595 := d67
					snap596 := d68
					snap597 := d69
					snap598 := d70
					snap599 := d71
					snap600 := d72
					snap601 := d73
					snap602 := d74
					snap603 := d104
					snap604 := d135
					snap605 := d136
					snap606 := d137
					snap607 := d138
					snap608 := d139
					snap609 := d140
					snap610 := d141
					snap611 := d142
					snap612 := d143
					snap613 := d144
					snap614 := d145
					snap615 := d187
					snap616 := d188
					snap617 := d189
					snap618 := d190
					snap619 := d191
					snap620 := d192
					snap621 := d193
					snap622 := d194
					snap623 := d244
					snap624 := d245
					snap625 := d246
					snap626 := d247
					snap627 := d248
					snap628 := d249
					snap629 := d250
					snap630 := d251
					snap631 := d309
					snap632 := d310
					snap633 := d311
					snap634 := d312
					snap635 := d313
					snap636 := d314
					snap637 := d315
					snap638 := d316
					snap639 := d317
					snap640 := d318
					snap641 := d319
					snap642 := d320
					snap643 := d321
					snap644 := d322
					snap645 := d323
					snap646 := d324
					snap647 := d325
					snap648 := d326
					snap649 := d327
					snap650 := d328
					snap651 := d329
					snap652 := d330
					snap653 := d331
					snap654 := d332
					snap655 := d333
					snap656 := d334
					snap657 := d335
					snap658 := d420
					snap659 := d421
					snap660 := d422
					snap661 := d423
					snap662 := d424
					snap663 := d425
					snap664 := d426
					snap665 := d427
					snap666 := d428
					snap667 := d429
					snap668 := d430
					snap669 := d431
					snap670 := d432
					snap671 := d433
					snap672 := d434
					snap673 := d435
					snap674 := d436
					snap675 := d437
					snap676 := d438
					snap677 := d439
					snap678 := d440
					snap679 := d441
					snap680 := d442
					snap681 := d443
					snap682 := d444
					snap683 := d445
					snap684 := d446
					snap685 := d447
					snap686 := d560
					snap687 := d561
					snap688 := d562
					snap689 := d563
					snap690 := d564
					snap691 := d565
					snap692 := d566
					snap693 := d567
					snap694 := d568
					snap695 := d569
					snap696 := d570
					snap697 := d571
					snap698 := d572
					snap699 := d573
					snap700 := d574
					alloc701 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl34)
					ctx.SyncDesc(&d569)
					if d569.Loc == LocReg || d569.Loc == LocFPReg {
						ctx.ProtectReg(d569.Reg)
					} else if d569.Loc == LocRegPair {
						ctx.ProtectReg(d569.Reg)
						ctx.ProtectReg(d569.Reg2)
					}
					d702 = d569
					if d702.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d702)
					ctx.EmitStoreToStack(d702, int32(bbs[23].PhiBase)+int32(0))
					if d569.Loc == LocReg || d569.Loc == LocFPReg {
						ctx.UnprotectReg(d569.Reg)
					} else if d569.Loc == LocRegPair {
						ctx.UnprotectReg(d569.Reg)
						ctx.UnprotectReg(d569.Reg2)
					}
					ctx.EmitJmp(lbl24)
					ctx.RestoreAllocState(alloc701)
					d1 = snap575
					d2 = snap576
					d3 = snap577
					d4 = snap578
					d5 = snap579
					d6 = snap580
					d7 = snap581
					d16 = snap582
					d17 = snap583
					d19 = snap584
					d20 = snap585
					d21 = snap586
					d23 = snap587
					d24 = snap588
					d25 = snap589
					d42 = snap590
					d43 = snap591
					d44 = snap592
					d45 = snap593
					d66 = snap594
					d67 = snap595
					d68 = snap596
					d69 = snap597
					d70 = snap598
					d71 = snap599
					d72 = snap600
					d73 = snap601
					d74 = snap602
					d104 = snap603
					d135 = snap604
					d136 = snap605
					d137 = snap606
					d138 = snap607
					d139 = snap608
					d140 = snap609
					d141 = snap610
					d142 = snap611
					d143 = snap612
					d144 = snap613
					d145 = snap614
					d187 = snap615
					d188 = snap616
					d189 = snap617
					d190 = snap618
					d191 = snap619
					d192 = snap620
					d193 = snap621
					d194 = snap622
					d244 = snap623
					d245 = snap624
					d246 = snap625
					d247 = snap626
					d248 = snap627
					d249 = snap628
					d250 = snap629
					d251 = snap630
					d309 = snap631
					d310 = snap632
					d311 = snap633
					d312 = snap634
					d313 = snap635
					d314 = snap636
					d315 = snap637
					d316 = snap638
					d317 = snap639
					d318 = snap640
					d319 = snap641
					d320 = snap642
					d321 = snap643
					d322 = snap644
					d323 = snap645
					d324 = snap646
					d325 = snap647
					d326 = snap648
					d327 = snap649
					d328 = snap650
					d329 = snap651
					d330 = snap652
					d331 = snap653
					d332 = snap654
					d333 = snap655
					d334 = snap656
					d335 = snap657
					d420 = snap658
					d421 = snap659
					d422 = snap660
					d423 = snap661
					d424 = snap662
					d425 = snap663
					d426 = snap664
					d427 = snap665
					d428 = snap666
					d429 = snap667
					d430 = snap668
					d431 = snap669
					d432 = snap670
					d433 = snap671
					d434 = snap672
					d435 = snap673
					d436 = snap674
					d437 = snap675
					d438 = snap676
					d439 = snap677
					d440 = snap678
					d441 = snap679
					d442 = snap680
					d443 = snap681
					d444 = snap682
					d445 = snap683
					d446 = snap684
					d447 = snap685
					d560 = snap686
					d561 = snap687
					d562 = snap688
					d563 = snap689
					d564 = snap690
					d565 = snap691
					d566 = snap692
					d567 = snap693
					d568 = snap694
					d569 = snap695
					d570 = snap696
					d571 = snap697
					d572 = snap698
					d573 = snap699
					d574 = snap700
					if !bbs[23].Rendered {
						snap703 := d1
						snap704 := d2
						snap705 := d3
						snap706 := d4
						snap707 := d5
						snap708 := d6
						snap709 := d7
						snap710 := d16
						snap711 := d17
						snap712 := d19
						snap713 := d20
						snap714 := d21
						snap715 := d23
						snap716 := d24
						snap717 := d25
						snap718 := d42
						snap719 := d43
						snap720 := d44
						snap721 := d45
						snap722 := d66
						snap723 := d67
						snap724 := d68
						snap725 := d69
						snap726 := d70
						snap727 := d71
						snap728 := d72
						snap729 := d73
						snap730 := d74
						snap731 := d104
						snap732 := d135
						snap733 := d136
						snap734 := d137
						snap735 := d138
						snap736 := d139
						snap737 := d140
						snap738 := d141
						snap739 := d142
						snap740 := d143
						snap741 := d144
						snap742 := d145
						snap743 := d187
						snap744 := d188
						snap745 := d189
						snap746 := d190
						snap747 := d191
						snap748 := d192
						snap749 := d193
						snap750 := d194
						snap751 := d244
						snap752 := d245
						snap753 := d246
						snap754 := d247
						snap755 := d248
						snap756 := d249
						snap757 := d250
						snap758 := d251
						snap759 := d309
						snap760 := d310
						snap761 := d311
						snap762 := d312
						snap763 := d313
						snap764 := d314
						snap765 := d315
						snap766 := d316
						snap767 := d317
						snap768 := d318
						snap769 := d319
						snap770 := d320
						snap771 := d321
						snap772 := d322
						snap773 := d323
						snap774 := d324
						snap775 := d325
						snap776 := d326
						snap777 := d327
						snap778 := d328
						snap779 := d329
						snap780 := d330
						snap781 := d331
						snap782 := d332
						snap783 := d333
						snap784 := d334
						snap785 := d335
						snap786 := d420
						snap787 := d421
						snap788 := d422
						snap789 := d423
						snap790 := d424
						snap791 := d425
						snap792 := d426
						snap793 := d427
						snap794 := d428
						snap795 := d429
						snap796 := d430
						snap797 := d431
						snap798 := d432
						snap799 := d433
						snap800 := d434
						snap801 := d435
						snap802 := d436
						snap803 := d437
						snap804 := d438
						snap805 := d439
						snap806 := d440
						snap807 := d441
						snap808 := d442
						snap809 := d443
						snap810 := d444
						snap811 := d445
						snap812 := d446
						snap813 := d447
						snap814 := d560
						snap815 := d561
						snap816 := d562
						snap817 := d563
						snap818 := d564
						snap819 := d565
						snap820 := d566
						snap821 := d567
						snap822 := d568
						snap823 := d569
						snap824 := d570
						snap825 := d571
						snap826 := d572
						snap827 := d573
						snap828 := d574
						snap829 := d702
						alloc830 := ctx.SnapshotAllocState()
						bbs[23].Render()
						ctx.RestoreAllocState(alloc830)
						d1 = snap703
						d2 = snap704
						d3 = snap705
						d4 = snap706
						d5 = snap707
						d6 = snap708
						d7 = snap709
						d16 = snap710
						d17 = snap711
						d19 = snap712
						d20 = snap713
						d21 = snap714
						d23 = snap715
						d24 = snap716
						d25 = snap717
						d42 = snap718
						d43 = snap719
						d44 = snap720
						d45 = snap721
						d66 = snap722
						d67 = snap723
						d68 = snap724
						d69 = snap725
						d70 = snap726
						d71 = snap727
						d72 = snap728
						d73 = snap729
						d74 = snap730
						d104 = snap731
						d135 = snap732
						d136 = snap733
						d137 = snap734
						d138 = snap735
						d139 = snap736
						d140 = snap737
						d141 = snap738
						d142 = snap739
						d143 = snap740
						d144 = snap741
						d145 = snap742
						d187 = snap743
						d188 = snap744
						d189 = snap745
						d190 = snap746
						d191 = snap747
						d192 = snap748
						d193 = snap749
						d194 = snap750
						d244 = snap751
						d245 = snap752
						d246 = snap753
						d247 = snap754
						d248 = snap755
						d249 = snap756
						d250 = snap757
						d251 = snap758
						d309 = snap759
						d310 = snap760
						d311 = snap761
						d312 = snap762
						d313 = snap763
						d314 = snap764
						d315 = snap765
						d316 = snap766
						d317 = snap767
						d318 = snap768
						d319 = snap769
						d320 = snap770
						d321 = snap771
						d322 = snap772
						d323 = snap773
						d324 = snap774
						d325 = snap775
						d326 = snap776
						d327 = snap777
						d328 = snap778
						d329 = snap779
						d330 = snap780
						d331 = snap781
						d332 = snap782
						d333 = snap783
						d334 = snap784
						d335 = snap785
						d420 = snap786
						d421 = snap787
						d422 = snap788
						d423 = snap789
						d424 = snap790
						d425 = snap791
						d426 = snap792
						d427 = snap793
						d428 = snap794
						d429 = snap795
						d430 = snap796
						d431 = snap797
						d432 = snap798
						d433 = snap799
						d434 = snap800
						d435 = snap801
						d436 = snap802
						d437 = snap803
						d438 = snap804
						d439 = snap805
						d440 = snap806
						d441 = snap807
						d442 = snap808
						d443 = snap809
						d444 = snap810
						d445 = snap811
						d446 = snap812
						d447 = snap813
						d560 = snap814
						d561 = snap815
						d562 = snap816
						d563 = snap817
						d564 = snap818
						d565 = snap819
						d566 = snap820
						d567 = snap821
						d568 = snap822
						d569 = snap823
						d570 = snap824
						d571 = snap825
						d572 = snap826
						d573 = snap827
						d574 = snap828
						d702 = snap829
					}
					if !bbs[22].Rendered {
						return bbs[22].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d831 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					if d831.Loc == LocImm {
						ctx.TrackImm(d831.Imm)
						ptrWord, _ := d831.Imm.RawWords()
						d832 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d832.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d832.Reg2, uint64(len(d831.Imm.String())))
						ctx.BindReg(d832.Reg, &d832)
						ctx.BindReg(d832.Reg2, &d832)
					} else {
						d832 = d831
					}
					d833 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d832}, 1)
					ctx.EmitAndRegImm32(d833.Reg, 1)
					d833.Type = tagBool
					ctx.BindReg(d833.Reg, &d833)
					d834 = d833
					ctx.EnsureDesc(&d834)
					if d834.Loc != LocImm && d834.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d834.Loc == LocImm {
						if d834.Imm.Bool() {
							return bbs[18].Render()
						}
						return bbs[21].Render()
					}
					ctx.EmitCmpRegImm32(d834.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					if bbs[21].Rendered {
						ctx.EmitJmp(lbl22)
					}
					ctx.FlushRegisterMoves()
					if !bbs[21].Rendered {
						snap835 := d1
						snap836 := d2
						snap837 := d3
						snap838 := d4
						snap839 := d5
						snap840 := d6
						snap841 := d7
						snap842 := d16
						snap843 := d17
						snap844 := d19
						snap845 := d20
						snap846 := d21
						snap847 := d23
						snap848 := d24
						snap849 := d25
						snap850 := d42
						snap851 := d43
						snap852 := d44
						snap853 := d45
						snap854 := d66
						snap855 := d67
						snap856 := d68
						snap857 := d69
						snap858 := d70
						snap859 := d71
						snap860 := d72
						snap861 := d73
						snap862 := d74
						snap863 := d104
						snap864 := d135
						snap865 := d136
						snap866 := d137
						snap867 := d138
						snap868 := d139
						snap869 := d140
						snap870 := d141
						snap871 := d142
						snap872 := d143
						snap873 := d144
						snap874 := d145
						snap875 := d187
						snap876 := d188
						snap877 := d189
						snap878 := d190
						snap879 := d191
						snap880 := d192
						snap881 := d193
						snap882 := d194
						snap883 := d244
						snap884 := d245
						snap885 := d246
						snap886 := d247
						snap887 := d248
						snap888 := d249
						snap889 := d250
						snap890 := d251
						snap891 := d309
						snap892 := d310
						snap893 := d311
						snap894 := d312
						snap895 := d313
						snap896 := d314
						snap897 := d315
						snap898 := d316
						snap899 := d317
						snap900 := d318
						snap901 := d319
						snap902 := d320
						snap903 := d321
						snap904 := d322
						snap905 := d323
						snap906 := d324
						snap907 := d325
						snap908 := d326
						snap909 := d327
						snap910 := d328
						snap911 := d329
						snap912 := d330
						snap913 := d331
						snap914 := d332
						snap915 := d333
						snap916 := d334
						snap917 := d335
						snap918 := d420
						snap919 := d421
						snap920 := d422
						snap921 := d423
						snap922 := d424
						snap923 := d425
						snap924 := d426
						snap925 := d427
						snap926 := d428
						snap927 := d429
						snap928 := d430
						snap929 := d431
						snap930 := d432
						snap931 := d433
						snap932 := d434
						snap933 := d435
						snap934 := d436
						snap935 := d437
						snap936 := d438
						snap937 := d439
						snap938 := d440
						snap939 := d441
						snap940 := d442
						snap941 := d443
						snap942 := d444
						snap943 := d445
						snap944 := d446
						snap945 := d447
						snap946 := d560
						snap947 := d561
						snap948 := d562
						snap949 := d563
						snap950 := d564
						snap951 := d565
						snap952 := d566
						snap953 := d567
						snap954 := d568
						snap955 := d569
						snap956 := d570
						snap957 := d571
						snap958 := d572
						snap959 := d573
						snap960 := d574
						snap961 := d702
						snap962 := d831
						snap963 := d832
						snap964 := d833
						snap965 := d834
						alloc966 := ctx.SnapshotAllocState()
						bbs[21].Render()
						ctx.RestoreAllocState(alloc966)
						d1 = snap835
						d2 = snap836
						d3 = snap837
						d4 = snap838
						d5 = snap839
						d6 = snap840
						d7 = snap841
						d16 = snap842
						d17 = snap843
						d19 = snap844
						d20 = snap845
						d21 = snap846
						d23 = snap847
						d24 = snap848
						d25 = snap849
						d42 = snap850
						d43 = snap851
						d44 = snap852
						d45 = snap853
						d66 = snap854
						d67 = snap855
						d68 = snap856
						d69 = snap857
						d70 = snap858
						d71 = snap859
						d72 = snap860
						d73 = snap861
						d74 = snap862
						d104 = snap863
						d135 = snap864
						d136 = snap865
						d137 = snap866
						d138 = snap867
						d139 = snap868
						d140 = snap869
						d141 = snap870
						d142 = snap871
						d143 = snap872
						d144 = snap873
						d145 = snap874
						d187 = snap875
						d188 = snap876
						d189 = snap877
						d190 = snap878
						d191 = snap879
						d192 = snap880
						d193 = snap881
						d194 = snap882
						d244 = snap883
						d245 = snap884
						d246 = snap885
						d247 = snap886
						d248 = snap887
						d249 = snap888
						d250 = snap889
						d251 = snap890
						d309 = snap891
						d310 = snap892
						d311 = snap893
						d312 = snap894
						d313 = snap895
						d314 = snap896
						d315 = snap897
						d316 = snap898
						d317 = snap899
						d318 = snap900
						d319 = snap901
						d320 = snap902
						d321 = snap903
						d322 = snap904
						d323 = snap905
						d324 = snap906
						d325 = snap907
						d326 = snap908
						d327 = snap909
						d328 = snap910
						d329 = snap911
						d330 = snap912
						d331 = snap913
						d332 = snap914
						d333 = snap915
						d334 = snap916
						d335 = snap917
						d420 = snap918
						d421 = snap919
						d422 = snap920
						d423 = snap921
						d424 = snap922
						d425 = snap923
						d426 = snap924
						d427 = snap925
						d428 = snap926
						d429 = snap927
						d430 = snap928
						d431 = snap929
						d432 = snap930
						d433 = snap931
						d434 = snap932
						d435 = snap933
						d436 = snap934
						d437 = snap935
						d438 = snap936
						d439 = snap937
						d440 = snap938
						d441 = snap939
						d442 = snap940
						d443 = snap941
						d444 = snap942
						d445 = snap943
						d446 = snap944
						d447 = snap945
						d560 = snap946
						d561 = snap947
						d562 = snap948
						d563 = snap949
						d564 = snap950
						d565 = snap951
						d566 = snap952
						d567 = snap953
						d568 = snap954
						d569 = snap955
						d570 = snap956
						d571 = snap957
						d572 = snap958
						d573 = snap959
						d574 = snap960
						d702 = snap961
						d831 = snap962
						d832 = snap963
						d833 = snap964
						d834 = snap965
					}
					if !bbs[18].Rendered {
						return bbs[18].Render()
					}
					return result
					ctx.FreeDesc(&d833)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d23)
					d967 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d23}, 1)
					d967.NoHeapPointer = true
					ctx.BindReg(d967.Reg, &d967)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d19)
					d968 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d19}, 1)
					d968.NoHeapPointer = true
					ctx.BindReg(d968.Reg, &d968)
					ctx.EnsureDesc(&d967)
					ctx.EnsureDesc(&d968)
					ctx.SyncDesc(&d967)
					ctx.SyncDesc(&d968)
					if d967.Loc == LocImm && d968.Loc == LocImm {
						d969 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d967.Imm.Int() - d968.Imm.Int())}
					} else if d968.Loc == LocImm && d968.Imm.Int() == 0 {
						ctx.EnsureDesc(&d967)
						r14 := ctx.AllocRegExcept(d967.Reg)
						ctx.EmitMovRegReg(r14, d967.Reg)
						d969 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d969)
					} else if d967.Loc == LocImm {
						ctx.EnsureDesc(&d968)
						scratch := ctx.AllocRegExcept(d968.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d967.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d968)
						d969 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d969)
					} else if d968.Loc == LocImm {
						ctx.EnsureDesc(&d967)
						scratch := ctx.AllocRegExcept(d967.Reg)
						ctx.EmitMovRegReg(scratch, d967.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d968.Imm.Int())
						d969 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d969)
					} else {
						ctx.EnsureDesc(&d967)
						ctx.SyncDesc(&d968)
						r15 := ctx.AllocRegExceptOperand(&d968, d967.Reg)
						ctx.EmitMovRegReg(r15, d967.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r15, &d968)
						d969 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d969)
					}
					if d969.Loc == LocReg && d967.Loc == LocReg && d969.Reg == d967.Reg {
						ctx.TransferReg(d967.Reg)
						d967.Loc = LocNone
					}
					ctx.FreeDesc(&d967)
					ctx.FreeDesc(&d968)
					ctx.EnsureDesc(&d969)
					ctx.EnsureDesc(&d969)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d23)
					d971 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d23}, 1)
					d971.NoHeapPointer = true
					ctx.BindReg(d971.Reg, &d971)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d19)
					d972 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d19}, 1)
					d972.NoHeapPointer = true
					ctx.BindReg(d972.Reg, &d972)
					ctx.EnsureDesc(&d971)
					ctx.EnsureDesc(&d972)
					ctx.SyncDesc(&d971)
					ctx.SyncDesc(&d972)
					if d971.Loc == LocImm && d972.Loc == LocImm {
						d973 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d971.Imm.Int() - d972.Imm.Int())}
					} else if d972.Loc == LocImm && d972.Imm.Int() == 0 {
						ctx.EnsureDesc(&d971)
						r16 := ctx.AllocRegExcept(d971.Reg)
						ctx.EmitMovRegReg(r16, d971.Reg)
						d973 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d973)
					} else if d971.Loc == LocImm {
						ctx.EnsureDesc(&d972)
						scratch := ctx.AllocRegExcept(d972.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d971.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d972)
						d973 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d973)
					} else if d972.Loc == LocImm {
						ctx.EnsureDesc(&d971)
						scratch := ctx.AllocRegExcept(d971.Reg)
						ctx.EmitMovRegReg(scratch, d971.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d972.Imm.Int())
						d973 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d973)
					} else {
						ctx.EnsureDesc(&d971)
						ctx.SyncDesc(&d972)
						r17 := ctx.AllocRegExceptOperand(&d972, d971.Reg)
						ctx.EmitMovRegReg(r17, d971.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r17, &d972)
						d973 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d973)
					}
					if d973.Loc == LocReg && d971.Loc == LocReg && d973.Reg == d971.Reg {
						ctx.TransferReg(d971.Reg)
						d971.Loc = LocNone
					}
					ctx.FreeDesc(&d971)
					ctx.FreeDesc(&d972)
					ctx.EnsureDesc(&d973)
					ctx.EnsureDesc(&d973)
					ctx.EnsureDesc(&d969)
					ctx.EnsureDesc(&d969)
					if d969.Loc == LocImm {
						d975 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d969.Imm.Int() * 12)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d969.Reg, 12)
						d975 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d969.Reg}
						ctx.BindReg(d969.Reg, &d975)
					}
					if d975.Loc == LocReg && d969.Loc == LocReg && d975.Reg == d969.Reg {
						ctx.TransferReg(d969.Reg)
						d969.Loc = LocNone
					}
					ctx.FreeDesc(&d969)
					ctx.EnsureDesc(&d975)
					ctx.EnsureDesc(&d973)
					ctx.SyncDesc(&d975)
					ctx.SyncDesc(&d973)
					if d975.Loc == LocImm && d973.Loc == LocImm {
						d976 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d975.Imm.Int() + d973.Imm.Int())}
					} else if d973.Loc == LocImm && d973.Imm.Int() == 0 {
						ctx.EnsureDesc(&d975)
						r18 := ctx.AllocRegExcept(d975.Reg)
						ctx.EmitMovRegReg(r18, d975.Reg)
						d976 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d976)
					} else if d975.Loc == LocImm && d975.Imm.Int() == 0 {
						ctx.EnsureDesc(&d973)
						d976 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d973.Reg}
						ctx.BindReg(d973.Reg, &d976)
					} else if d975.Loc == LocImm {
						ctx.EnsureDesc(&d973)
						scratch := ctx.AllocRegExcept(d973.Reg)
						ctx.EmitMovRegReg(scratch, d973.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d975.Imm.Int())
						d976 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d976)
					} else if d973.Loc == LocImm {
						ctx.EnsureDesc(&d975)
						scratch := ctx.AllocRegExcept(d975.Reg)
						ctx.EmitMovRegReg(scratch, d975.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d973.Imm.Int())
						d976 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d976)
					} else {
						ctx.EnsureDesc(&d975)
						ctx.SyncDesc(&d973)
						r19 := ctx.AllocRegExceptOperand(&d973, d975.Reg)
						ctx.EmitMovRegReg(r19, d975.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r19, &d973)
						d976 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d976)
					}
					if d976.Loc == LocReg && d975.Loc == LocReg && d976.Reg == d975.Reg {
						ctx.TransferReg(d975.Reg)
						d975.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d976)
					ctx.FreeDesc(&d975)
					ctx.FreeDesc(&d973)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d23)
					d977 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d23}, 1)
					d977.NoHeapPointer = true
					ctx.BindReg(d977.Reg, &d977)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d19)
					d978 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d19}, 1)
					d978.NoHeapPointer = true
					ctx.BindReg(d978.Reg, &d978)
					ctx.EnsureDesc(&d977)
					ctx.EnsureDesc(&d978)
					ctx.EnsureDescsTogether(&d977, &d978)
					if d977.Loc == LocImm && d978.Loc == LocImm {
						d979 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d977.Imm.Int() < d978.Imm.Int())}
					} else if d978.Loc == LocImm {
						r20 := ctx.AllocReg()
						if d978.Imm.Int() >= -2147483648 && d978.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d977.Reg, int32(d978.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d978.Imm.Int()))
							ctx.EmitCmpInt64(d977.Reg, ctx.ScratchReg)
						}
						d979 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d979)
					} else if d977.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d977.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d978.Reg)
						d979 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedLess}
						ctx.BindReg(r21, &d979)
					} else {
						r22 := ctx.AllocReg()
						ctx.EmitCmpInt64(d977.Reg, d978.Reg)
						d979 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d979)
					}
					ctx.FreeDesc(&d977)
					ctx.FreeDesc(&d978)
					d980 = d979
					ctx.EnsureDesc(&d980)
					if d980.Loc != LocImm && d980.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d980.Loc == LocImm {
						if d980.Imm.Bool() {
							return bbs[26].Render()
						}
						ctx.SyncDesc(&d976)
						if d976.Loc == LocReg || d976.Loc == LocFPReg {
							ctx.ProtectReg(d976.Reg)
						} else if d976.Loc == LocRegPair {
							ctx.ProtectReg(d976.Reg)
							ctx.ProtectReg(d976.Reg2)
						}
						d981 = d976
						if d981.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d981)
						ctx.EmitStoreToStack(d981, int32(bbs[27].PhiBase)+int32(0))
						if d976.Loc == LocReg || d976.Loc == LocFPReg {
							ctx.UnprotectReg(d976.Reg)
						} else if d976.Loc == LocRegPair {
							ctx.UnprotectReg(d976.Reg)
							ctx.UnprotectReg(d976.Reg2)
						}
						return bbs[27].Render()
					}
					lbl35 := ctx.ReserveLabel()
					ctx.EmitJump(d980.Condition, lbl27)
					ctx.EmitJmp(lbl35)
					ctx.FreeDesc(&d979)
					snap982 := d1
					snap983 := d2
					snap984 := d3
					snap985 := d4
					snap986 := d5
					snap987 := d6
					snap988 := d7
					snap989 := d16
					snap990 := d17
					snap991 := d19
					snap992 := d20
					snap993 := d21
					snap994 := d23
					snap995 := d24
					snap996 := d25
					snap997 := d42
					snap998 := d43
					snap999 := d44
					snap1000 := d45
					snap1001 := d66
					snap1002 := d67
					snap1003 := d68
					snap1004 := d69
					snap1005 := d70
					snap1006 := d71
					snap1007 := d72
					snap1008 := d73
					snap1009 := d74
					snap1010 := d104
					snap1011 := d135
					snap1012 := d136
					snap1013 := d137
					snap1014 := d138
					snap1015 := d139
					snap1016 := d140
					snap1017 := d141
					snap1018 := d142
					snap1019 := d143
					snap1020 := d144
					snap1021 := d145
					snap1022 := d187
					snap1023 := d188
					snap1024 := d189
					snap1025 := d190
					snap1026 := d191
					snap1027 := d192
					snap1028 := d193
					snap1029 := d194
					snap1030 := d244
					snap1031 := d245
					snap1032 := d246
					snap1033 := d247
					snap1034 := d248
					snap1035 := d249
					snap1036 := d250
					snap1037 := d251
					snap1038 := d309
					snap1039 := d310
					snap1040 := d311
					snap1041 := d312
					snap1042 := d313
					snap1043 := d314
					snap1044 := d315
					snap1045 := d316
					snap1046 := d317
					snap1047 := d318
					snap1048 := d319
					snap1049 := d320
					snap1050 := d321
					snap1051 := d322
					snap1052 := d323
					snap1053 := d324
					snap1054 := d325
					snap1055 := d326
					snap1056 := d327
					snap1057 := d328
					snap1058 := d329
					snap1059 := d330
					snap1060 := d331
					snap1061 := d332
					snap1062 := d333
					snap1063 := d334
					snap1064 := d335
					snap1065 := d420
					snap1066 := d421
					snap1067 := d422
					snap1068 := d423
					snap1069 := d424
					snap1070 := d425
					snap1071 := d426
					snap1072 := d427
					snap1073 := d428
					snap1074 := d429
					snap1075 := d430
					snap1076 := d431
					snap1077 := d432
					snap1078 := d433
					snap1079 := d434
					snap1080 := d435
					snap1081 := d436
					snap1082 := d437
					snap1083 := d438
					snap1084 := d439
					snap1085 := d440
					snap1086 := d441
					snap1087 := d442
					snap1088 := d443
					snap1089 := d444
					snap1090 := d445
					snap1091 := d446
					snap1092 := d447
					snap1093 := d560
					snap1094 := d561
					snap1095 := d562
					snap1096 := d563
					snap1097 := d564
					snap1098 := d565
					snap1099 := d566
					snap1100 := d567
					snap1101 := d568
					snap1102 := d569
					snap1103 := d570
					snap1104 := d571
					snap1105 := d572
					snap1106 := d573
					snap1107 := d574
					snap1108 := d702
					snap1109 := d831
					snap1110 := d832
					snap1111 := d833
					snap1112 := d834
					snap1113 := d967
					snap1114 := d968
					snap1115 := d969
					snap1116 := d970
					snap1117 := d971
					snap1118 := d972
					snap1119 := d973
					snap1120 := d974
					snap1121 := d975
					snap1122 := d976
					snap1123 := d977
					snap1124 := d978
					snap1125 := d979
					snap1126 := d980
					snap1127 := d981
					alloc1128 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl35)
					ctx.SyncDesc(&d976)
					if d976.Loc == LocReg || d976.Loc == LocFPReg {
						ctx.ProtectReg(d976.Reg)
					} else if d976.Loc == LocRegPair {
						ctx.ProtectReg(d976.Reg)
						ctx.ProtectReg(d976.Reg2)
					}
					d1129 = d976
					if d1129.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d1129)
					ctx.EmitStoreToStack(d1129, int32(bbs[27].PhiBase)+int32(0))
					if d976.Loc == LocReg || d976.Loc == LocFPReg {
						ctx.UnprotectReg(d976.Reg)
					} else if d976.Loc == LocRegPair {
						ctx.UnprotectReg(d976.Reg)
						ctx.UnprotectReg(d976.Reg2)
					}
					ctx.EmitJmp(lbl28)
					ctx.RestoreAllocState(alloc1128)
					d1 = snap982
					d2 = snap983
					d3 = snap984
					d4 = snap985
					d5 = snap986
					d6 = snap987
					d7 = snap988
					d16 = snap989
					d17 = snap990
					d19 = snap991
					d20 = snap992
					d21 = snap993
					d23 = snap994
					d24 = snap995
					d25 = snap996
					d42 = snap997
					d43 = snap998
					d44 = snap999
					d45 = snap1000
					d66 = snap1001
					d67 = snap1002
					d68 = snap1003
					d69 = snap1004
					d70 = snap1005
					d71 = snap1006
					d72 = snap1007
					d73 = snap1008
					d74 = snap1009
					d104 = snap1010
					d135 = snap1011
					d136 = snap1012
					d137 = snap1013
					d138 = snap1014
					d139 = snap1015
					d140 = snap1016
					d141 = snap1017
					d142 = snap1018
					d143 = snap1019
					d144 = snap1020
					d145 = snap1021
					d187 = snap1022
					d188 = snap1023
					d189 = snap1024
					d190 = snap1025
					d191 = snap1026
					d192 = snap1027
					d193 = snap1028
					d194 = snap1029
					d244 = snap1030
					d245 = snap1031
					d246 = snap1032
					d247 = snap1033
					d248 = snap1034
					d249 = snap1035
					d250 = snap1036
					d251 = snap1037
					d309 = snap1038
					d310 = snap1039
					d311 = snap1040
					d312 = snap1041
					d313 = snap1042
					d314 = snap1043
					d315 = snap1044
					d316 = snap1045
					d317 = snap1046
					d318 = snap1047
					d319 = snap1048
					d320 = snap1049
					d321 = snap1050
					d322 = snap1051
					d323 = snap1052
					d324 = snap1053
					d325 = snap1054
					d326 = snap1055
					d327 = snap1056
					d328 = snap1057
					d329 = snap1058
					d330 = snap1059
					d331 = snap1060
					d332 = snap1061
					d333 = snap1062
					d334 = snap1063
					d335 = snap1064
					d420 = snap1065
					d421 = snap1066
					d422 = snap1067
					d423 = snap1068
					d424 = snap1069
					d425 = snap1070
					d426 = snap1071
					d427 = snap1072
					d428 = snap1073
					d429 = snap1074
					d430 = snap1075
					d431 = snap1076
					d432 = snap1077
					d433 = snap1078
					d434 = snap1079
					d435 = snap1080
					d436 = snap1081
					d437 = snap1082
					d438 = snap1083
					d439 = snap1084
					d440 = snap1085
					d441 = snap1086
					d442 = snap1087
					d443 = snap1088
					d444 = snap1089
					d445 = snap1090
					d446 = snap1091
					d447 = snap1092
					d560 = snap1093
					d561 = snap1094
					d562 = snap1095
					d563 = snap1096
					d564 = snap1097
					d565 = snap1098
					d566 = snap1099
					d567 = snap1100
					d568 = snap1101
					d569 = snap1102
					d570 = snap1103
					d571 = snap1104
					d572 = snap1105
					d573 = snap1106
					d574 = snap1107
					d702 = snap1108
					d831 = snap1109
					d832 = snap1110
					d833 = snap1111
					d834 = snap1112
					d967 = snap1113
					d968 = snap1114
					d969 = snap1115
					d970 = snap1116
					d971 = snap1117
					d972 = snap1118
					d973 = snap1119
					d974 = snap1120
					d975 = snap1121
					d976 = snap1122
					d977 = snap1123
					d978 = snap1124
					d979 = snap1125
					d980 = snap1126
					d981 = snap1127
					if !bbs[27].Rendered {
						snap1130 := d1
						snap1131 := d2
						snap1132 := d3
						snap1133 := d4
						snap1134 := d5
						snap1135 := d6
						snap1136 := d7
						snap1137 := d16
						snap1138 := d17
						snap1139 := d19
						snap1140 := d20
						snap1141 := d21
						snap1142 := d23
						snap1143 := d24
						snap1144 := d25
						snap1145 := d42
						snap1146 := d43
						snap1147 := d44
						snap1148 := d45
						snap1149 := d66
						snap1150 := d67
						snap1151 := d68
						snap1152 := d69
						snap1153 := d70
						snap1154 := d71
						snap1155 := d72
						snap1156 := d73
						snap1157 := d74
						snap1158 := d104
						snap1159 := d135
						snap1160 := d136
						snap1161 := d137
						snap1162 := d138
						snap1163 := d139
						snap1164 := d140
						snap1165 := d141
						snap1166 := d142
						snap1167 := d143
						snap1168 := d144
						snap1169 := d145
						snap1170 := d187
						snap1171 := d188
						snap1172 := d189
						snap1173 := d190
						snap1174 := d191
						snap1175 := d192
						snap1176 := d193
						snap1177 := d194
						snap1178 := d244
						snap1179 := d245
						snap1180 := d246
						snap1181 := d247
						snap1182 := d248
						snap1183 := d249
						snap1184 := d250
						snap1185 := d251
						snap1186 := d309
						snap1187 := d310
						snap1188 := d311
						snap1189 := d312
						snap1190 := d313
						snap1191 := d314
						snap1192 := d315
						snap1193 := d316
						snap1194 := d317
						snap1195 := d318
						snap1196 := d319
						snap1197 := d320
						snap1198 := d321
						snap1199 := d322
						snap1200 := d323
						snap1201 := d324
						snap1202 := d325
						snap1203 := d326
						snap1204 := d327
						snap1205 := d328
						snap1206 := d329
						snap1207 := d330
						snap1208 := d331
						snap1209 := d332
						snap1210 := d333
						snap1211 := d334
						snap1212 := d335
						snap1213 := d420
						snap1214 := d421
						snap1215 := d422
						snap1216 := d423
						snap1217 := d424
						snap1218 := d425
						snap1219 := d426
						snap1220 := d427
						snap1221 := d428
						snap1222 := d429
						snap1223 := d430
						snap1224 := d431
						snap1225 := d432
						snap1226 := d433
						snap1227 := d434
						snap1228 := d435
						snap1229 := d436
						snap1230 := d437
						snap1231 := d438
						snap1232 := d439
						snap1233 := d440
						snap1234 := d441
						snap1235 := d442
						snap1236 := d443
						snap1237 := d444
						snap1238 := d445
						snap1239 := d446
						snap1240 := d447
						snap1241 := d560
						snap1242 := d561
						snap1243 := d562
						snap1244 := d563
						snap1245 := d564
						snap1246 := d565
						snap1247 := d566
						snap1248 := d567
						snap1249 := d568
						snap1250 := d569
						snap1251 := d570
						snap1252 := d571
						snap1253 := d572
						snap1254 := d573
						snap1255 := d574
						snap1256 := d702
						snap1257 := d831
						snap1258 := d832
						snap1259 := d833
						snap1260 := d834
						snap1261 := d967
						snap1262 := d968
						snap1263 := d969
						snap1264 := d970
						snap1265 := d971
						snap1266 := d972
						snap1267 := d973
						snap1268 := d974
						snap1269 := d975
						snap1270 := d976
						snap1271 := d977
						snap1272 := d978
						snap1273 := d979
						snap1274 := d980
						snap1275 := d981
						snap1276 := d1129
						alloc1277 := ctx.SnapshotAllocState()
						bbs[27].Render()
						ctx.RestoreAllocState(alloc1277)
						d1 = snap1130
						d2 = snap1131
						d3 = snap1132
						d4 = snap1133
						d5 = snap1134
						d6 = snap1135
						d7 = snap1136
						d16 = snap1137
						d17 = snap1138
						d19 = snap1139
						d20 = snap1140
						d21 = snap1141
						d23 = snap1142
						d24 = snap1143
						d25 = snap1144
						d42 = snap1145
						d43 = snap1146
						d44 = snap1147
						d45 = snap1148
						d66 = snap1149
						d67 = snap1150
						d68 = snap1151
						d69 = snap1152
						d70 = snap1153
						d71 = snap1154
						d72 = snap1155
						d73 = snap1156
						d74 = snap1157
						d104 = snap1158
						d135 = snap1159
						d136 = snap1160
						d137 = snap1161
						d138 = snap1162
						d139 = snap1163
						d140 = snap1164
						d141 = snap1165
						d142 = snap1166
						d143 = snap1167
						d144 = snap1168
						d145 = snap1169
						d187 = snap1170
						d188 = snap1171
						d189 = snap1172
						d190 = snap1173
						d191 = snap1174
						d192 = snap1175
						d193 = snap1176
						d194 = snap1177
						d244 = snap1178
						d245 = snap1179
						d246 = snap1180
						d247 = snap1181
						d248 = snap1182
						d249 = snap1183
						d250 = snap1184
						d251 = snap1185
						d309 = snap1186
						d310 = snap1187
						d311 = snap1188
						d312 = snap1189
						d313 = snap1190
						d314 = snap1191
						d315 = snap1192
						d316 = snap1193
						d317 = snap1194
						d318 = snap1195
						d319 = snap1196
						d320 = snap1197
						d321 = snap1198
						d322 = snap1199
						d323 = snap1200
						d324 = snap1201
						d325 = snap1202
						d326 = snap1203
						d327 = snap1204
						d328 = snap1205
						d329 = snap1206
						d330 = snap1207
						d331 = snap1208
						d332 = snap1209
						d333 = snap1210
						d334 = snap1211
						d335 = snap1212
						d420 = snap1213
						d421 = snap1214
						d422 = snap1215
						d423 = snap1216
						d424 = snap1217
						d425 = snap1218
						d426 = snap1219
						d427 = snap1220
						d428 = snap1221
						d429 = snap1222
						d430 = snap1223
						d431 = snap1224
						d432 = snap1225
						d433 = snap1226
						d434 = snap1227
						d435 = snap1228
						d436 = snap1229
						d437 = snap1230
						d438 = snap1231
						d439 = snap1232
						d440 = snap1233
						d441 = snap1234
						d442 = snap1235
						d443 = snap1236
						d444 = snap1237
						d445 = snap1238
						d446 = snap1239
						d447 = snap1240
						d560 = snap1241
						d561 = snap1242
						d562 = snap1243
						d563 = snap1244
						d564 = snap1245
						d565 = snap1246
						d566 = snap1247
						d567 = snap1248
						d568 = snap1249
						d569 = snap1250
						d570 = snap1251
						d571 = snap1252
						d572 = snap1253
						d573 = snap1254
						d574 = snap1255
						d702 = snap1256
						d831 = snap1257
						d832 = snap1258
						d833 = snap1259
						d834 = snap1260
						d967 = snap1261
						d968 = snap1262
						d969 = snap1263
						d970 = snap1264
						d971 = snap1265
						d972 = snap1266
						d973 = snap1267
						d974 = snap1268
						d975 = snap1269
						d976 = snap1270
						d977 = snap1271
						d978 = snap1272
						d979 = snap1273
						d980 = snap1274
						d981 = snap1275
						d1129 = snap1276
					}
					if !bbs[26].Rendered {
						return bbs[26].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d1278 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("QUARTER")}
					if d1278.Loc == LocImm {
						ctx.TrackImm(d1278.Imm)
						ptrWord, _ := d1278.Imm.RawWords()
						d1279 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d1279.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d1279.Reg2, uint64(len(d1278.Imm.String())))
						ctx.BindReg(d1279.Reg, &d1279)
						ctx.BindReg(d1279.Reg2, &d1279)
					} else {
						d1279 = d1278
					}
					d1280 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d1279}, 1)
					ctx.EmitAndRegImm32(d1280.Reg, 1)
					d1280.Type = tagBool
					ctx.BindReg(d1280.Reg, &d1280)
					d1281 = d1280
					ctx.EnsureDesc(&d1281)
					if d1281.Loc != LocImm && d1281.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d1281.Loc == LocImm {
						if d1281.Imm.Bool() {
							return bbs[20].Render()
						}
						return bbs[25].Render()
					}
					ctx.EmitCmpRegImm32(d1281.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					if bbs[25].Rendered {
						ctx.EmitJmp(lbl26)
					}
					ctx.FlushRegisterMoves()
					if !bbs[25].Rendered {
						snap1282 := d1
						snap1283 := d2
						snap1284 := d3
						snap1285 := d4
						snap1286 := d5
						snap1287 := d6
						snap1288 := d7
						snap1289 := d16
						snap1290 := d17
						snap1291 := d19
						snap1292 := d20
						snap1293 := d21
						snap1294 := d23
						snap1295 := d24
						snap1296 := d25
						snap1297 := d42
						snap1298 := d43
						snap1299 := d44
						snap1300 := d45
						snap1301 := d66
						snap1302 := d67
						snap1303 := d68
						snap1304 := d69
						snap1305 := d70
						snap1306 := d71
						snap1307 := d72
						snap1308 := d73
						snap1309 := d74
						snap1310 := d104
						snap1311 := d135
						snap1312 := d136
						snap1313 := d137
						snap1314 := d138
						snap1315 := d139
						snap1316 := d140
						snap1317 := d141
						snap1318 := d142
						snap1319 := d143
						snap1320 := d144
						snap1321 := d145
						snap1322 := d187
						snap1323 := d188
						snap1324 := d189
						snap1325 := d190
						snap1326 := d191
						snap1327 := d192
						snap1328 := d193
						snap1329 := d194
						snap1330 := d244
						snap1331 := d245
						snap1332 := d246
						snap1333 := d247
						snap1334 := d248
						snap1335 := d249
						snap1336 := d250
						snap1337 := d251
						snap1338 := d309
						snap1339 := d310
						snap1340 := d311
						snap1341 := d312
						snap1342 := d313
						snap1343 := d314
						snap1344 := d315
						snap1345 := d316
						snap1346 := d317
						snap1347 := d318
						snap1348 := d319
						snap1349 := d320
						snap1350 := d321
						snap1351 := d322
						snap1352 := d323
						snap1353 := d324
						snap1354 := d325
						snap1355 := d326
						snap1356 := d327
						snap1357 := d328
						snap1358 := d329
						snap1359 := d330
						snap1360 := d331
						snap1361 := d332
						snap1362 := d333
						snap1363 := d334
						snap1364 := d335
						snap1365 := d420
						snap1366 := d421
						snap1367 := d422
						snap1368 := d423
						snap1369 := d424
						snap1370 := d425
						snap1371 := d426
						snap1372 := d427
						snap1373 := d428
						snap1374 := d429
						snap1375 := d430
						snap1376 := d431
						snap1377 := d432
						snap1378 := d433
						snap1379 := d434
						snap1380 := d435
						snap1381 := d436
						snap1382 := d437
						snap1383 := d438
						snap1384 := d439
						snap1385 := d440
						snap1386 := d441
						snap1387 := d442
						snap1388 := d443
						snap1389 := d444
						snap1390 := d445
						snap1391 := d446
						snap1392 := d447
						snap1393 := d560
						snap1394 := d561
						snap1395 := d562
						snap1396 := d563
						snap1397 := d564
						snap1398 := d565
						snap1399 := d566
						snap1400 := d567
						snap1401 := d568
						snap1402 := d569
						snap1403 := d570
						snap1404 := d571
						snap1405 := d572
						snap1406 := d573
						snap1407 := d574
						snap1408 := d702
						snap1409 := d831
						snap1410 := d832
						snap1411 := d833
						snap1412 := d834
						snap1413 := d967
						snap1414 := d968
						snap1415 := d969
						snap1416 := d970
						snap1417 := d971
						snap1418 := d972
						snap1419 := d973
						snap1420 := d974
						snap1421 := d975
						snap1422 := d976
						snap1423 := d977
						snap1424 := d978
						snap1425 := d979
						snap1426 := d980
						snap1427 := d981
						snap1428 := d1129
						snap1429 := d1278
						snap1430 := d1279
						snap1431 := d1280
						snap1432 := d1281
						alloc1433 := ctx.SnapshotAllocState()
						bbs[25].Render()
						ctx.RestoreAllocState(alloc1433)
						d1 = snap1282
						d2 = snap1283
						d3 = snap1284
						d4 = snap1285
						d5 = snap1286
						d6 = snap1287
						d7 = snap1288
						d16 = snap1289
						d17 = snap1290
						d19 = snap1291
						d20 = snap1292
						d21 = snap1293
						d23 = snap1294
						d24 = snap1295
						d25 = snap1296
						d42 = snap1297
						d43 = snap1298
						d44 = snap1299
						d45 = snap1300
						d66 = snap1301
						d67 = snap1302
						d68 = snap1303
						d69 = snap1304
						d70 = snap1305
						d71 = snap1306
						d72 = snap1307
						d73 = snap1308
						d74 = snap1309
						d104 = snap1310
						d135 = snap1311
						d136 = snap1312
						d137 = snap1313
						d138 = snap1314
						d139 = snap1315
						d140 = snap1316
						d141 = snap1317
						d142 = snap1318
						d143 = snap1319
						d144 = snap1320
						d145 = snap1321
						d187 = snap1322
						d188 = snap1323
						d189 = snap1324
						d190 = snap1325
						d191 = snap1326
						d192 = snap1327
						d193 = snap1328
						d194 = snap1329
						d244 = snap1330
						d245 = snap1331
						d246 = snap1332
						d247 = snap1333
						d248 = snap1334
						d249 = snap1335
						d250 = snap1336
						d251 = snap1337
						d309 = snap1338
						d310 = snap1339
						d311 = snap1340
						d312 = snap1341
						d313 = snap1342
						d314 = snap1343
						d315 = snap1344
						d316 = snap1345
						d317 = snap1346
						d318 = snap1347
						d319 = snap1348
						d320 = snap1349
						d321 = snap1350
						d322 = snap1351
						d323 = snap1352
						d324 = snap1353
						d325 = snap1354
						d326 = snap1355
						d327 = snap1356
						d328 = snap1357
						d329 = snap1358
						d330 = snap1359
						d331 = snap1360
						d332 = snap1361
						d333 = snap1362
						d334 = snap1363
						d335 = snap1364
						d420 = snap1365
						d421 = snap1366
						d422 = snap1367
						d423 = snap1368
						d424 = snap1369
						d425 = snap1370
						d426 = snap1371
						d427 = snap1372
						d428 = snap1373
						d429 = snap1374
						d430 = snap1375
						d431 = snap1376
						d432 = snap1377
						d433 = snap1378
						d434 = snap1379
						d435 = snap1380
						d436 = snap1381
						d437 = snap1382
						d438 = snap1383
						d439 = snap1384
						d440 = snap1385
						d441 = snap1386
						d442 = snap1387
						d443 = snap1388
						d444 = snap1389
						d445 = snap1390
						d446 = snap1391
						d447 = snap1392
						d560 = snap1393
						d561 = snap1394
						d562 = snap1395
						d563 = snap1396
						d564 = snap1397
						d565 = snap1398
						d566 = snap1399
						d567 = snap1400
						d568 = snap1401
						d569 = snap1402
						d570 = snap1403
						d571 = snap1404
						d572 = snap1405
						d573 = snap1406
						d574 = snap1407
						d702 = snap1408
						d831 = snap1409
						d832 = snap1410
						d833 = snap1411
						d834 = snap1412
						d967 = snap1413
						d968 = snap1414
						d969 = snap1415
						d970 = snap1416
						d971 = snap1417
						d972 = snap1418
						d973 = snap1419
						d974 = snap1420
						d975 = snap1421
						d976 = snap1422
						d977 = snap1423
						d978 = snap1424
						d979 = snap1425
						d980 = snap1426
						d981 = snap1427
						d1129 = snap1428
						d1278 = snap1429
						d1279 = snap1430
						d1280 = snap1431
						d1281 = snap1432
					}
					if !bbs[20].Rendered {
						return bbs[20].Render()
					}
					return result
					ctx.FreeDesc(&d1280)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d569)
					ctx.EnsureDesc(&d569)
					if d569.Loc == LocImm {
						d1434 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d569.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d569.Reg)
						ctx.EmitMovRegReg(scratch, d569.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d1434 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1434)
					}
					if d1434.Loc == LocReg && d569.Loc == LocReg && d1434.Reg == d569.Reg {
						ctx.TransferReg(d569.Reg)
						d569.Loc = LocNone
					}
					ctx.EnsureDesc(&d1434)
					ctx.EmitStoreToStack(d1434, int32(bbs[23].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d1434)
					return bbs[23].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						ctx.EmitMakeInt(result, d1)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1)
						d1435 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d1435)
						if d1.Loc == LocReg && d1.Reg != result.Reg2 {
							ctx.FreeReg(d1.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d23)
					d1436 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d23}, 1)
					d1436.NoHeapPointer = true
					ctx.BindReg(d1436.Reg, &d1436)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d19)
					d1437 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d19}, 1)
					d1437.NoHeapPointer = true
					ctx.BindReg(d1437.Reg, &d1437)
					ctx.EnsureDesc(&d1436)
					ctx.EnsureDesc(&d1437)
					ctx.SyncDesc(&d1436)
					ctx.SyncDesc(&d1437)
					if d1436.Loc == LocImm && d1437.Loc == LocImm {
						d1438 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1436.Imm.Int() - d1437.Imm.Int())}
					} else if d1437.Loc == LocImm && d1437.Imm.Int() == 0 {
						ctx.EnsureDesc(&d1436)
						r23 := ctx.AllocRegExcept(d1436.Reg)
						ctx.EmitMovRegReg(r23, d1436.Reg)
						d1438 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r23}
						ctx.BindReg(r23, &d1438)
					} else if d1436.Loc == LocImm {
						ctx.EnsureDesc(&d1437)
						scratch := ctx.AllocRegExcept(d1437.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1436.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d1437)
						d1438 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1438)
					} else if d1437.Loc == LocImm {
						ctx.EnsureDesc(&d1436)
						scratch := ctx.AllocRegExcept(d1436.Reg)
						ctx.EmitMovRegReg(scratch, d1436.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d1437.Imm.Int())
						d1438 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1438)
					} else {
						ctx.EnsureDesc(&d1436)
						ctx.SyncDesc(&d1437)
						r24 := ctx.AllocRegExceptOperand(&d1437, d1436.Reg)
						ctx.EmitMovRegReg(r24, d1436.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r24, &d1437)
						d1438 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
						ctx.BindReg(r24, &d1438)
					}
					if d1438.Loc == LocReg && d1436.Loc == LocReg && d1438.Reg == d1436.Reg {
						ctx.TransferReg(d1436.Reg)
						d1436.Loc = LocNone
					}
					ctx.FreeDesc(&d1436)
					ctx.FreeDesc(&d1437)
					ctx.EnsureDesc(&d1438)
					ctx.EnsureDesc(&d1438)
					ctx.StabilizeDescForControlFlow(&d1438)
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d23)
					d1440 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d23}, 1)
					d1440.NoHeapPointer = true
					ctx.BindReg(d1440.Reg, &d1440)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d19)
					d1441 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d19}, 1)
					d1441.NoHeapPointer = true
					ctx.BindReg(d1441.Reg, &d1441)
					ctx.EnsureDesc(&d1440)
					ctx.EnsureDesc(&d1441)
					ctx.EnsureDescsTogether(&d1440, &d1441)
					if d1440.Loc == LocImm && d1441.Loc == LocImm {
						d1442 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1440.Imm.Int() < d1441.Imm.Int())}
					} else if d1441.Loc == LocImm {
						r25 := ctx.AllocReg()
						if d1441.Imm.Int() >= -2147483648 && d1441.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1440.Reg, int32(d1441.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1441.Imm.Int()))
							ctx.EmitCmpInt64(d1440.Reg, ctx.ScratchReg)
						}
						d1442 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedLess}
						ctx.BindReg(r25, &d1442)
					} else if d1440.Loc == LocImm {
						r26 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1440.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d1441.Reg)
						d1442 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedLess}
						ctx.BindReg(r26, &d1442)
					} else {
						r27 := ctx.AllocReg()
						ctx.EmitCmpInt64(d1440.Reg, d1441.Reg)
						d1442 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
						ctx.BindReg(r27, &d1442)
					}
					ctx.FreeDesc(&d1440)
					ctx.FreeDesc(&d1441)
					d1443 = d1442
					ctx.EnsureDesc(&d1443)
					if d1443.Loc != LocImm && d1443.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d1443.Loc == LocImm {
						if d1443.Imm.Bool() {
							return bbs[29].Render()
						}
						return bbs[31].Render()
					}
					ctx.EmitJump(d1443.Condition, lbl30)
					if bbs[31].Rendered {
						ctx.EmitJmp(lbl32)
					}
					ctx.FreeDesc(&d1442)
					ctx.FlushRegisterMoves()
					if !bbs[31].Rendered {
						snap1444 := d1
						snap1445 := d2
						snap1446 := d3
						snap1447 := d4
						snap1448 := d5
						snap1449 := d6
						snap1450 := d7
						snap1451 := d16
						snap1452 := d17
						snap1453 := d19
						snap1454 := d20
						snap1455 := d21
						snap1456 := d23
						snap1457 := d24
						snap1458 := d25
						snap1459 := d42
						snap1460 := d43
						snap1461 := d44
						snap1462 := d45
						snap1463 := d66
						snap1464 := d67
						snap1465 := d68
						snap1466 := d69
						snap1467 := d70
						snap1468 := d71
						snap1469 := d72
						snap1470 := d73
						snap1471 := d74
						snap1472 := d104
						snap1473 := d135
						snap1474 := d136
						snap1475 := d137
						snap1476 := d138
						snap1477 := d139
						snap1478 := d140
						snap1479 := d141
						snap1480 := d142
						snap1481 := d143
						snap1482 := d144
						snap1483 := d145
						snap1484 := d187
						snap1485 := d188
						snap1486 := d189
						snap1487 := d190
						snap1488 := d191
						snap1489 := d192
						snap1490 := d193
						snap1491 := d194
						snap1492 := d244
						snap1493 := d245
						snap1494 := d246
						snap1495 := d247
						snap1496 := d248
						snap1497 := d249
						snap1498 := d250
						snap1499 := d251
						snap1500 := d309
						snap1501 := d310
						snap1502 := d311
						snap1503 := d312
						snap1504 := d313
						snap1505 := d314
						snap1506 := d315
						snap1507 := d316
						snap1508 := d317
						snap1509 := d318
						snap1510 := d319
						snap1511 := d320
						snap1512 := d321
						snap1513 := d322
						snap1514 := d323
						snap1515 := d324
						snap1516 := d325
						snap1517 := d326
						snap1518 := d327
						snap1519 := d328
						snap1520 := d329
						snap1521 := d330
						snap1522 := d331
						snap1523 := d332
						snap1524 := d333
						snap1525 := d334
						snap1526 := d335
						snap1527 := d420
						snap1528 := d421
						snap1529 := d422
						snap1530 := d423
						snap1531 := d424
						snap1532 := d425
						snap1533 := d426
						snap1534 := d427
						snap1535 := d428
						snap1536 := d429
						snap1537 := d430
						snap1538 := d431
						snap1539 := d432
						snap1540 := d433
						snap1541 := d434
						snap1542 := d435
						snap1543 := d436
						snap1544 := d437
						snap1545 := d438
						snap1546 := d439
						snap1547 := d440
						snap1548 := d441
						snap1549 := d442
						snap1550 := d443
						snap1551 := d444
						snap1552 := d445
						snap1553 := d446
						snap1554 := d447
						snap1555 := d560
						snap1556 := d561
						snap1557 := d562
						snap1558 := d563
						snap1559 := d564
						snap1560 := d565
						snap1561 := d566
						snap1562 := d567
						snap1563 := d568
						snap1564 := d569
						snap1565 := d570
						snap1566 := d571
						snap1567 := d572
						snap1568 := d573
						snap1569 := d574
						snap1570 := d702
						snap1571 := d831
						snap1572 := d832
						snap1573 := d833
						snap1574 := d834
						snap1575 := d967
						snap1576 := d968
						snap1577 := d969
						snap1578 := d970
						snap1579 := d971
						snap1580 := d972
						snap1581 := d973
						snap1582 := d974
						snap1583 := d975
						snap1584 := d976
						snap1585 := d977
						snap1586 := d978
						snap1587 := d979
						snap1588 := d980
						snap1589 := d981
						snap1590 := d1129
						snap1591 := d1278
						snap1592 := d1279
						snap1593 := d1280
						snap1594 := d1281
						snap1595 := d1434
						snap1596 := d1435
						snap1597 := d1436
						snap1598 := d1437
						snap1599 := d1438
						snap1600 := d1439
						snap1601 := d1440
						snap1602 := d1441
						snap1603 := d1442
						snap1604 := d1443
						alloc1605 := ctx.SnapshotAllocState()
						bbs[31].Render()
						ctx.RestoreAllocState(alloc1605)
						d1 = snap1444
						d2 = snap1445
						d3 = snap1446
						d4 = snap1447
						d5 = snap1448
						d6 = snap1449
						d7 = snap1450
						d16 = snap1451
						d17 = snap1452
						d19 = snap1453
						d20 = snap1454
						d21 = snap1455
						d23 = snap1456
						d24 = snap1457
						d25 = snap1458
						d42 = snap1459
						d43 = snap1460
						d44 = snap1461
						d45 = snap1462
						d66 = snap1463
						d67 = snap1464
						d68 = snap1465
						d69 = snap1466
						d70 = snap1467
						d71 = snap1468
						d72 = snap1469
						d73 = snap1470
						d74 = snap1471
						d104 = snap1472
						d135 = snap1473
						d136 = snap1474
						d137 = snap1475
						d138 = snap1476
						d139 = snap1477
						d140 = snap1478
						d141 = snap1479
						d142 = snap1480
						d143 = snap1481
						d144 = snap1482
						d145 = snap1483
						d187 = snap1484
						d188 = snap1485
						d189 = snap1486
						d190 = snap1487
						d191 = snap1488
						d192 = snap1489
						d193 = snap1490
						d194 = snap1491
						d244 = snap1492
						d245 = snap1493
						d246 = snap1494
						d247 = snap1495
						d248 = snap1496
						d249 = snap1497
						d250 = snap1498
						d251 = snap1499
						d309 = snap1500
						d310 = snap1501
						d311 = snap1502
						d312 = snap1503
						d313 = snap1504
						d314 = snap1505
						d315 = snap1506
						d316 = snap1507
						d317 = snap1508
						d318 = snap1509
						d319 = snap1510
						d320 = snap1511
						d321 = snap1512
						d322 = snap1513
						d323 = snap1514
						d324 = snap1515
						d325 = snap1516
						d326 = snap1517
						d327 = snap1518
						d328 = snap1519
						d329 = snap1520
						d330 = snap1521
						d331 = snap1522
						d332 = snap1523
						d333 = snap1524
						d334 = snap1525
						d335 = snap1526
						d420 = snap1527
						d421 = snap1528
						d422 = snap1529
						d423 = snap1530
						d424 = snap1531
						d425 = snap1532
						d426 = snap1533
						d427 = snap1534
						d428 = snap1535
						d429 = snap1536
						d430 = snap1537
						d431 = snap1538
						d432 = snap1539
						d433 = snap1540
						d434 = snap1541
						d435 = snap1542
						d436 = snap1543
						d437 = snap1544
						d438 = snap1545
						d439 = snap1546
						d440 = snap1547
						d441 = snap1548
						d442 = snap1549
						d443 = snap1550
						d444 = snap1551
						d445 = snap1552
						d446 = snap1553
						d447 = snap1554
						d560 = snap1555
						d561 = snap1556
						d562 = snap1557
						d563 = snap1558
						d564 = snap1559
						d565 = snap1560
						d566 = snap1561
						d567 = snap1562
						d568 = snap1563
						d569 = snap1564
						d570 = snap1565
						d571 = snap1566
						d572 = snap1567
						d573 = snap1568
						d574 = snap1569
						d702 = snap1570
						d831 = snap1571
						d832 = snap1572
						d833 = snap1573
						d834 = snap1574
						d967 = snap1575
						d968 = snap1576
						d969 = snap1577
						d970 = snap1578
						d971 = snap1579
						d972 = snap1580
						d973 = snap1581
						d974 = snap1582
						d975 = snap1583
						d976 = snap1584
						d977 = snap1585
						d978 = snap1586
						d979 = snap1587
						d980 = snap1588
						d981 = snap1589
						d1129 = snap1590
						d1278 = snap1591
						d1279 = snap1592
						d1280 = snap1593
						d1281 = snap1594
						d1434 = snap1595
						d1435 = snap1596
						d1436 = snap1597
						d1437 = snap1598
						d1438 = snap1599
						d1439 = snap1600
						d1440 = snap1601
						d1441 = snap1602
						d1442 = snap1603
						d1443 = snap1604
					}
					if !bbs[29].Rendered {
						return bbs[29].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					d1606 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					if d1606.Loc == LocImm {
						ctx.TrackImm(d1606.Imm)
						ptrWord, _ := d1606.Imm.RawWords()
						d1607 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d1607.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d1607.Reg2, uint64(len(d1606.Imm.String())))
						ctx.BindReg(d1607.Reg, &d1607)
						ctx.BindReg(d1607.Reg2, &d1607)
					} else {
						d1607 = d1606
					}
					d1608 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d70, d1607}, 1)
					ctx.EmitAndRegImm32(d1608.Reg, 1)
					d1608.Type = tagBool
					ctx.BindReg(d1608.Reg, &d1608)
					d1609 = d1608
					ctx.EnsureDesc(&d1609)
					if d1609.Loc != LocImm && d1609.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d1609.Loc == LocImm {
						if d1609.Imm.Bool() {
							return bbs[24].Render()
						}
						return bbs[28].Render()
					}
					ctx.EmitCmpRegImm32(d1609.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl25)
					if bbs[28].Rendered {
						ctx.EmitJmp(lbl29)
					}
					ctx.FlushRegisterMoves()
					if !bbs[28].Rendered {
						snap1610 := d1
						snap1611 := d2
						snap1612 := d3
						snap1613 := d4
						snap1614 := d5
						snap1615 := d6
						snap1616 := d7
						snap1617 := d16
						snap1618 := d17
						snap1619 := d19
						snap1620 := d20
						snap1621 := d21
						snap1622 := d23
						snap1623 := d24
						snap1624 := d25
						snap1625 := d42
						snap1626 := d43
						snap1627 := d44
						snap1628 := d45
						snap1629 := d66
						snap1630 := d67
						snap1631 := d68
						snap1632 := d69
						snap1633 := d70
						snap1634 := d71
						snap1635 := d72
						snap1636 := d73
						snap1637 := d74
						snap1638 := d104
						snap1639 := d135
						snap1640 := d136
						snap1641 := d137
						snap1642 := d138
						snap1643 := d139
						snap1644 := d140
						snap1645 := d141
						snap1646 := d142
						snap1647 := d143
						snap1648 := d144
						snap1649 := d145
						snap1650 := d187
						snap1651 := d188
						snap1652 := d189
						snap1653 := d190
						snap1654 := d191
						snap1655 := d192
						snap1656 := d193
						snap1657 := d194
						snap1658 := d244
						snap1659 := d245
						snap1660 := d246
						snap1661 := d247
						snap1662 := d248
						snap1663 := d249
						snap1664 := d250
						snap1665 := d251
						snap1666 := d309
						snap1667 := d310
						snap1668 := d311
						snap1669 := d312
						snap1670 := d313
						snap1671 := d314
						snap1672 := d315
						snap1673 := d316
						snap1674 := d317
						snap1675 := d318
						snap1676 := d319
						snap1677 := d320
						snap1678 := d321
						snap1679 := d322
						snap1680 := d323
						snap1681 := d324
						snap1682 := d325
						snap1683 := d326
						snap1684 := d327
						snap1685 := d328
						snap1686 := d329
						snap1687 := d330
						snap1688 := d331
						snap1689 := d332
						snap1690 := d333
						snap1691 := d334
						snap1692 := d335
						snap1693 := d420
						snap1694 := d421
						snap1695 := d422
						snap1696 := d423
						snap1697 := d424
						snap1698 := d425
						snap1699 := d426
						snap1700 := d427
						snap1701 := d428
						snap1702 := d429
						snap1703 := d430
						snap1704 := d431
						snap1705 := d432
						snap1706 := d433
						snap1707 := d434
						snap1708 := d435
						snap1709 := d436
						snap1710 := d437
						snap1711 := d438
						snap1712 := d439
						snap1713 := d440
						snap1714 := d441
						snap1715 := d442
						snap1716 := d443
						snap1717 := d444
						snap1718 := d445
						snap1719 := d446
						snap1720 := d447
						snap1721 := d560
						snap1722 := d561
						snap1723 := d562
						snap1724 := d563
						snap1725 := d564
						snap1726 := d565
						snap1727 := d566
						snap1728 := d567
						snap1729 := d568
						snap1730 := d569
						snap1731 := d570
						snap1732 := d571
						snap1733 := d572
						snap1734 := d573
						snap1735 := d574
						snap1736 := d702
						snap1737 := d831
						snap1738 := d832
						snap1739 := d833
						snap1740 := d834
						snap1741 := d967
						snap1742 := d968
						snap1743 := d969
						snap1744 := d970
						snap1745 := d971
						snap1746 := d972
						snap1747 := d973
						snap1748 := d974
						snap1749 := d975
						snap1750 := d976
						snap1751 := d977
						snap1752 := d978
						snap1753 := d979
						snap1754 := d980
						snap1755 := d981
						snap1756 := d1129
						snap1757 := d1278
						snap1758 := d1279
						snap1759 := d1280
						snap1760 := d1281
						snap1761 := d1434
						snap1762 := d1435
						snap1763 := d1436
						snap1764 := d1437
						snap1765 := d1438
						snap1766 := d1439
						snap1767 := d1440
						snap1768 := d1441
						snap1769 := d1442
						snap1770 := d1443
						snap1771 := d1606
						snap1772 := d1607
						snap1773 := d1608
						snap1774 := d1609
						alloc1775 := ctx.SnapshotAllocState()
						bbs[28].Render()
						ctx.RestoreAllocState(alloc1775)
						d1 = snap1610
						d2 = snap1611
						d3 = snap1612
						d4 = snap1613
						d5 = snap1614
						d6 = snap1615
						d7 = snap1616
						d16 = snap1617
						d17 = snap1618
						d19 = snap1619
						d20 = snap1620
						d21 = snap1621
						d23 = snap1622
						d24 = snap1623
						d25 = snap1624
						d42 = snap1625
						d43 = snap1626
						d44 = snap1627
						d45 = snap1628
						d66 = snap1629
						d67 = snap1630
						d68 = snap1631
						d69 = snap1632
						d70 = snap1633
						d71 = snap1634
						d72 = snap1635
						d73 = snap1636
						d74 = snap1637
						d104 = snap1638
						d135 = snap1639
						d136 = snap1640
						d137 = snap1641
						d138 = snap1642
						d139 = snap1643
						d140 = snap1644
						d141 = snap1645
						d142 = snap1646
						d143 = snap1647
						d144 = snap1648
						d145 = snap1649
						d187 = snap1650
						d188 = snap1651
						d189 = snap1652
						d190 = snap1653
						d191 = snap1654
						d192 = snap1655
						d193 = snap1656
						d194 = snap1657
						d244 = snap1658
						d245 = snap1659
						d246 = snap1660
						d247 = snap1661
						d248 = snap1662
						d249 = snap1663
						d250 = snap1664
						d251 = snap1665
						d309 = snap1666
						d310 = snap1667
						d311 = snap1668
						d312 = snap1669
						d313 = snap1670
						d314 = snap1671
						d315 = snap1672
						d316 = snap1673
						d317 = snap1674
						d318 = snap1675
						d319 = snap1676
						d320 = snap1677
						d321 = snap1678
						d322 = snap1679
						d323 = snap1680
						d324 = snap1681
						d325 = snap1682
						d326 = snap1683
						d327 = snap1684
						d328 = snap1685
						d329 = snap1686
						d330 = snap1687
						d331 = snap1688
						d332 = snap1689
						d333 = snap1690
						d334 = snap1691
						d335 = snap1692
						d420 = snap1693
						d421 = snap1694
						d422 = snap1695
						d423 = snap1696
						d424 = snap1697
						d425 = snap1698
						d426 = snap1699
						d427 = snap1700
						d428 = snap1701
						d429 = snap1702
						d430 = snap1703
						d431 = snap1704
						d432 = snap1705
						d433 = snap1706
						d434 = snap1707
						d435 = snap1708
						d436 = snap1709
						d437 = snap1710
						d438 = snap1711
						d439 = snap1712
						d440 = snap1713
						d441 = snap1714
						d442 = snap1715
						d443 = snap1716
						d444 = snap1717
						d445 = snap1718
						d446 = snap1719
						d447 = snap1720
						d560 = snap1721
						d561 = snap1722
						d562 = snap1723
						d563 = snap1724
						d564 = snap1725
						d565 = snap1726
						d566 = snap1727
						d567 = snap1728
						d568 = snap1729
						d569 = snap1730
						d570 = snap1731
						d571 = snap1732
						d572 = snap1733
						d573 = snap1734
						d574 = snap1735
						d702 = snap1736
						d831 = snap1737
						d832 = snap1738
						d833 = snap1739
						d834 = snap1740
						d967 = snap1741
						d968 = snap1742
						d969 = snap1743
						d970 = snap1744
						d971 = snap1745
						d972 = snap1746
						d973 = snap1747
						d974 = snap1748
						d975 = snap1749
						d976 = snap1750
						d977 = snap1751
						d978 = snap1752
						d979 = snap1753
						d980 = snap1754
						d981 = snap1755
						d1129 = snap1756
						d1278 = snap1757
						d1279 = snap1758
						d1280 = snap1759
						d1281 = snap1760
						d1434 = snap1761
						d1435 = snap1762
						d1436 = snap1763
						d1437 = snap1764
						d1438 = snap1765
						d1439 = snap1766
						d1440 = snap1767
						d1441 = snap1768
						d1442 = snap1769
						d1443 = snap1770
						d1606 = snap1771
						d1607 = snap1772
						d1608 = snap1773
						d1609 = snap1774
					}
					if !bbs[24].Rendered {
						return bbs[24].Render()
					}
					return result
					ctx.FreeDesc(&d1608)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d976)
					ctx.EnsureDesc(&d976)
					if d976.Loc == LocImm {
						d1776 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d976.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d976.Reg)
						ctx.EmitMovRegReg(scratch, d976.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d1776 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1776)
					}
					if d1776.Loc == LocReg && d976.Loc == LocReg && d1776.Reg == d976.Reg {
						ctx.TransferReg(d976.Reg)
						d976.Loc = LocNone
					}
					ctx.EnsureDesc(&d1776)
					ctx.EmitStoreToStack(d1776, int32(bbs[27].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d1776)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					resultTarget1777 := false
					_ = resultTarget1777
					if d2.Loc == LocImm {
						d1778 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() / 3)}
					} else {
						r28 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(r28, d2.Reg)
						ctx.EmitIdivRegImm(r28, 3)
						d1778 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r28}
						ctx.BindReg(r28, &d1778)
					}
					if d1778.Loc == LocReg && d2.Loc == LocReg && d1778.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d1778)
					if d1778.Loc == LocImm {
						ctx.EmitMakeInt(result, d1778)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1778)
						d1779 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d1779)
						if d1778.Loc == LocReg && d1778.Reg != result.Reg2 {
							ctx.FreeReg(d1778.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["timestampdiff"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1438)
					ctx.EnsureDesc(&d1438)
					if d1438.Loc == LocImm {
						d1780 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1438.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1438.Reg)
						ctx.EmitMovRegReg(scratch, d1438.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d1780 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1780)
					}
					if d1780.Loc == LocReg && d1438.Loc == LocReg && d1780.Reg == d1438.Reg {
						ctx.TransferReg(d1438.Reg)
						d1438.Loc = LocNone
					}
					ctx.EnsureDesc(&d1780)
					ctx.EmitStoreToStack(d1780, int32(bbs[30].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d1780)
					return bbs[30].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						ctx.EmitMakeInt(result, d3)
					} else {
						ctx.EmitMovToReg(result.Reg2, d3)
						d1781 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d1781)
						if d3.Loc == LocReg && d3.Reg != result.Reg2 {
							ctx.FreeReg(d3.Reg)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d23)
					d1782 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d23}, 1)
					d1782.NoHeapPointer = true
					ctx.BindReg(d1782.Reg, &d1782)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d19)
					d1783 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d19}, 1)
					d1783.NoHeapPointer = true
					ctx.BindReg(d1783.Reg, &d1783)
					ctx.EnsureDesc(&d1782)
					ctx.EnsureDesc(&d1783)
					ctx.EnsureDescsTogether(&d1782, &d1783)
					if d1782.Loc == LocImm && d1783.Loc == LocImm {
						d1784 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1782.Imm.Int() == d1783.Imm.Int())}
					} else if d1783.Loc == LocImm {
						r29 := ctx.AllocReg()
						if d1783.Imm.Int() >= -2147483648 && d1783.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1782.Reg, int32(d1783.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1783.Imm.Int()))
							ctx.EmitCmpInt64(d1782.Reg, ctx.ScratchReg)
						}
						d1784 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondEqual}
						ctx.BindReg(r29, &d1784)
					} else if d1782.Loc == LocImm {
						r30 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1782.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d1783.Reg)
						d1784 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondEqual}
						ctx.BindReg(r30, &d1784)
					} else {
						r31 := ctx.AllocReg()
						ctx.EmitCmpInt64(d1782.Reg, d1783.Reg)
						d1784 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r31, Condition: CondEqual}
						ctx.BindReg(r31, &d1784)
					}
					ctx.FreeDesc(&d1782)
					ctx.FreeDesc(&d1783)
					d1785 = d1784
					ctx.EnsureDesc(&d1785)
					if d1785.Loc != LocImm && d1785.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d1785.Loc == LocImm {
						if d1785.Imm.Bool() {
							return bbs[32].Render()
						}
						ctx.SyncDesc(&d1438)
						if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
							ctx.ProtectReg(d1438.Reg)
						} else if d1438.Loc == LocRegPair {
							ctx.ProtectReg(d1438.Reg)
							ctx.ProtectReg(d1438.Reg2)
						}
						d1786 = d1438
						if d1786.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d1786)
						ctx.EmitStoreToStack(d1786, int32(bbs[30].PhiBase)+int32(0))
						if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
							ctx.UnprotectReg(d1438.Reg)
						} else if d1438.Loc == LocRegPair {
							ctx.UnprotectReg(d1438.Reg)
							ctx.UnprotectReg(d1438.Reg2)
						}
						return bbs[30].Render()
					}
					lbl36 := ctx.ReserveLabel()
					ctx.EmitJump(d1785.Condition, lbl33)
					ctx.EmitJmp(lbl36)
					ctx.FreeDesc(&d1784)
					snap1787 := d1
					snap1788 := d2
					snap1789 := d3
					snap1790 := d4
					snap1791 := d5
					snap1792 := d6
					snap1793 := d7
					snap1794 := d16
					snap1795 := d17
					snap1796 := d19
					snap1797 := d20
					snap1798 := d21
					snap1799 := d23
					snap1800 := d24
					snap1801 := d25
					snap1802 := d42
					snap1803 := d43
					snap1804 := d44
					snap1805 := d45
					snap1806 := d66
					snap1807 := d67
					snap1808 := d68
					snap1809 := d69
					snap1810 := d70
					snap1811 := d71
					snap1812 := d72
					snap1813 := d73
					snap1814 := d74
					snap1815 := d104
					snap1816 := d135
					snap1817 := d136
					snap1818 := d137
					snap1819 := d138
					snap1820 := d139
					snap1821 := d140
					snap1822 := d141
					snap1823 := d142
					snap1824 := d143
					snap1825 := d144
					snap1826 := d145
					snap1827 := d187
					snap1828 := d188
					snap1829 := d189
					snap1830 := d190
					snap1831 := d191
					snap1832 := d192
					snap1833 := d193
					snap1834 := d194
					snap1835 := d244
					snap1836 := d245
					snap1837 := d246
					snap1838 := d247
					snap1839 := d248
					snap1840 := d249
					snap1841 := d250
					snap1842 := d251
					snap1843 := d309
					snap1844 := d310
					snap1845 := d311
					snap1846 := d312
					snap1847 := d313
					snap1848 := d314
					snap1849 := d315
					snap1850 := d316
					snap1851 := d317
					snap1852 := d318
					snap1853 := d319
					snap1854 := d320
					snap1855 := d321
					snap1856 := d322
					snap1857 := d323
					snap1858 := d324
					snap1859 := d325
					snap1860 := d326
					snap1861 := d327
					snap1862 := d328
					snap1863 := d329
					snap1864 := d330
					snap1865 := d331
					snap1866 := d332
					snap1867 := d333
					snap1868 := d334
					snap1869 := d335
					snap1870 := d420
					snap1871 := d421
					snap1872 := d422
					snap1873 := d423
					snap1874 := d424
					snap1875 := d425
					snap1876 := d426
					snap1877 := d427
					snap1878 := d428
					snap1879 := d429
					snap1880 := d430
					snap1881 := d431
					snap1882 := d432
					snap1883 := d433
					snap1884 := d434
					snap1885 := d435
					snap1886 := d436
					snap1887 := d437
					snap1888 := d438
					snap1889 := d439
					snap1890 := d440
					snap1891 := d441
					snap1892 := d442
					snap1893 := d443
					snap1894 := d444
					snap1895 := d445
					snap1896 := d446
					snap1897 := d447
					snap1898 := d560
					snap1899 := d561
					snap1900 := d562
					snap1901 := d563
					snap1902 := d564
					snap1903 := d565
					snap1904 := d566
					snap1905 := d567
					snap1906 := d568
					snap1907 := d569
					snap1908 := d570
					snap1909 := d571
					snap1910 := d572
					snap1911 := d573
					snap1912 := d574
					snap1913 := d702
					snap1914 := d831
					snap1915 := d832
					snap1916 := d833
					snap1917 := d834
					snap1918 := d967
					snap1919 := d968
					snap1920 := d969
					snap1921 := d970
					snap1922 := d971
					snap1923 := d972
					snap1924 := d973
					snap1925 := d974
					snap1926 := d975
					snap1927 := d976
					snap1928 := d977
					snap1929 := d978
					snap1930 := d979
					snap1931 := d980
					snap1932 := d981
					snap1933 := d1129
					snap1934 := d1278
					snap1935 := d1279
					snap1936 := d1280
					snap1937 := d1281
					snap1938 := d1434
					snap1939 := d1435
					snap1940 := d1436
					snap1941 := d1437
					snap1942 := d1438
					snap1943 := d1439
					snap1944 := d1440
					snap1945 := d1441
					snap1946 := d1442
					snap1947 := d1443
					snap1948 := d1606
					snap1949 := d1607
					snap1950 := d1608
					snap1951 := d1609
					snap1952 := d1776
					snap1953 := d1778
					snap1954 := d1779
					snap1955 := d1780
					snap1956 := d1781
					snap1957 := d1782
					snap1958 := d1783
					snap1959 := d1784
					snap1960 := d1785
					snap1961 := d1786
					alloc1962 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl36)
					ctx.SyncDesc(&d1438)
					if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
						ctx.ProtectReg(d1438.Reg)
					} else if d1438.Loc == LocRegPair {
						ctx.ProtectReg(d1438.Reg)
						ctx.ProtectReg(d1438.Reg2)
					}
					d1963 = d1438
					if d1963.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d1963)
					ctx.EmitStoreToStack(d1963, int32(bbs[30].PhiBase)+int32(0))
					if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
						ctx.UnprotectReg(d1438.Reg)
					} else if d1438.Loc == LocRegPair {
						ctx.UnprotectReg(d1438.Reg)
						ctx.UnprotectReg(d1438.Reg2)
					}
					ctx.EmitJmp(lbl31)
					ctx.RestoreAllocState(alloc1962)
					d1 = snap1787
					d2 = snap1788
					d3 = snap1789
					d4 = snap1790
					d5 = snap1791
					d6 = snap1792
					d7 = snap1793
					d16 = snap1794
					d17 = snap1795
					d19 = snap1796
					d20 = snap1797
					d21 = snap1798
					d23 = snap1799
					d24 = snap1800
					d25 = snap1801
					d42 = snap1802
					d43 = snap1803
					d44 = snap1804
					d45 = snap1805
					d66 = snap1806
					d67 = snap1807
					d68 = snap1808
					d69 = snap1809
					d70 = snap1810
					d71 = snap1811
					d72 = snap1812
					d73 = snap1813
					d74 = snap1814
					d104 = snap1815
					d135 = snap1816
					d136 = snap1817
					d137 = snap1818
					d138 = snap1819
					d139 = snap1820
					d140 = snap1821
					d141 = snap1822
					d142 = snap1823
					d143 = snap1824
					d144 = snap1825
					d145 = snap1826
					d187 = snap1827
					d188 = snap1828
					d189 = snap1829
					d190 = snap1830
					d191 = snap1831
					d192 = snap1832
					d193 = snap1833
					d194 = snap1834
					d244 = snap1835
					d245 = snap1836
					d246 = snap1837
					d247 = snap1838
					d248 = snap1839
					d249 = snap1840
					d250 = snap1841
					d251 = snap1842
					d309 = snap1843
					d310 = snap1844
					d311 = snap1845
					d312 = snap1846
					d313 = snap1847
					d314 = snap1848
					d315 = snap1849
					d316 = snap1850
					d317 = snap1851
					d318 = snap1852
					d319 = snap1853
					d320 = snap1854
					d321 = snap1855
					d322 = snap1856
					d323 = snap1857
					d324 = snap1858
					d325 = snap1859
					d326 = snap1860
					d327 = snap1861
					d328 = snap1862
					d329 = snap1863
					d330 = snap1864
					d331 = snap1865
					d332 = snap1866
					d333 = snap1867
					d334 = snap1868
					d335 = snap1869
					d420 = snap1870
					d421 = snap1871
					d422 = snap1872
					d423 = snap1873
					d424 = snap1874
					d425 = snap1875
					d426 = snap1876
					d427 = snap1877
					d428 = snap1878
					d429 = snap1879
					d430 = snap1880
					d431 = snap1881
					d432 = snap1882
					d433 = snap1883
					d434 = snap1884
					d435 = snap1885
					d436 = snap1886
					d437 = snap1887
					d438 = snap1888
					d439 = snap1889
					d440 = snap1890
					d441 = snap1891
					d442 = snap1892
					d443 = snap1893
					d444 = snap1894
					d445 = snap1895
					d446 = snap1896
					d447 = snap1897
					d560 = snap1898
					d561 = snap1899
					d562 = snap1900
					d563 = snap1901
					d564 = snap1902
					d565 = snap1903
					d566 = snap1904
					d567 = snap1905
					d568 = snap1906
					d569 = snap1907
					d570 = snap1908
					d571 = snap1909
					d572 = snap1910
					d573 = snap1911
					d574 = snap1912
					d702 = snap1913
					d831 = snap1914
					d832 = snap1915
					d833 = snap1916
					d834 = snap1917
					d967 = snap1918
					d968 = snap1919
					d969 = snap1920
					d970 = snap1921
					d971 = snap1922
					d972 = snap1923
					d973 = snap1924
					d974 = snap1925
					d975 = snap1926
					d976 = snap1927
					d977 = snap1928
					d978 = snap1929
					d979 = snap1930
					d980 = snap1931
					d981 = snap1932
					d1129 = snap1933
					d1278 = snap1934
					d1279 = snap1935
					d1280 = snap1936
					d1281 = snap1937
					d1434 = snap1938
					d1435 = snap1939
					d1436 = snap1940
					d1437 = snap1941
					d1438 = snap1942
					d1439 = snap1943
					d1440 = snap1944
					d1441 = snap1945
					d1442 = snap1946
					d1443 = snap1947
					d1606 = snap1948
					d1607 = snap1949
					d1608 = snap1950
					d1609 = snap1951
					d1776 = snap1952
					d1778 = snap1953
					d1779 = snap1954
					d1780 = snap1955
					d1781 = snap1956
					d1782 = snap1957
					d1783 = snap1958
					d1784 = snap1959
					d1785 = snap1960
					d1786 = snap1961
					if !bbs[30].Rendered {
						snap1964 := d1
						snap1965 := d2
						snap1966 := d3
						snap1967 := d4
						snap1968 := d5
						snap1969 := d6
						snap1970 := d7
						snap1971 := d16
						snap1972 := d17
						snap1973 := d19
						snap1974 := d20
						snap1975 := d21
						snap1976 := d23
						snap1977 := d24
						snap1978 := d25
						snap1979 := d42
						snap1980 := d43
						snap1981 := d44
						snap1982 := d45
						snap1983 := d66
						snap1984 := d67
						snap1985 := d68
						snap1986 := d69
						snap1987 := d70
						snap1988 := d71
						snap1989 := d72
						snap1990 := d73
						snap1991 := d74
						snap1992 := d104
						snap1993 := d135
						snap1994 := d136
						snap1995 := d137
						snap1996 := d138
						snap1997 := d139
						snap1998 := d140
						snap1999 := d141
						snap2000 := d142
						snap2001 := d143
						snap2002 := d144
						snap2003 := d145
						snap2004 := d187
						snap2005 := d188
						snap2006 := d189
						snap2007 := d190
						snap2008 := d191
						snap2009 := d192
						snap2010 := d193
						snap2011 := d194
						snap2012 := d244
						snap2013 := d245
						snap2014 := d246
						snap2015 := d247
						snap2016 := d248
						snap2017 := d249
						snap2018 := d250
						snap2019 := d251
						snap2020 := d309
						snap2021 := d310
						snap2022 := d311
						snap2023 := d312
						snap2024 := d313
						snap2025 := d314
						snap2026 := d315
						snap2027 := d316
						snap2028 := d317
						snap2029 := d318
						snap2030 := d319
						snap2031 := d320
						snap2032 := d321
						snap2033 := d322
						snap2034 := d323
						snap2035 := d324
						snap2036 := d325
						snap2037 := d326
						snap2038 := d327
						snap2039 := d328
						snap2040 := d329
						snap2041 := d330
						snap2042 := d331
						snap2043 := d332
						snap2044 := d333
						snap2045 := d334
						snap2046 := d335
						snap2047 := d420
						snap2048 := d421
						snap2049 := d422
						snap2050 := d423
						snap2051 := d424
						snap2052 := d425
						snap2053 := d426
						snap2054 := d427
						snap2055 := d428
						snap2056 := d429
						snap2057 := d430
						snap2058 := d431
						snap2059 := d432
						snap2060 := d433
						snap2061 := d434
						snap2062 := d435
						snap2063 := d436
						snap2064 := d437
						snap2065 := d438
						snap2066 := d439
						snap2067 := d440
						snap2068 := d441
						snap2069 := d442
						snap2070 := d443
						snap2071 := d444
						snap2072 := d445
						snap2073 := d446
						snap2074 := d447
						snap2075 := d560
						snap2076 := d561
						snap2077 := d562
						snap2078 := d563
						snap2079 := d564
						snap2080 := d565
						snap2081 := d566
						snap2082 := d567
						snap2083 := d568
						snap2084 := d569
						snap2085 := d570
						snap2086 := d571
						snap2087 := d572
						snap2088 := d573
						snap2089 := d574
						snap2090 := d702
						snap2091 := d831
						snap2092 := d832
						snap2093 := d833
						snap2094 := d834
						snap2095 := d967
						snap2096 := d968
						snap2097 := d969
						snap2098 := d970
						snap2099 := d971
						snap2100 := d972
						snap2101 := d973
						snap2102 := d974
						snap2103 := d975
						snap2104 := d976
						snap2105 := d977
						snap2106 := d978
						snap2107 := d979
						snap2108 := d980
						snap2109 := d981
						snap2110 := d1129
						snap2111 := d1278
						snap2112 := d1279
						snap2113 := d1280
						snap2114 := d1281
						snap2115 := d1434
						snap2116 := d1435
						snap2117 := d1436
						snap2118 := d1437
						snap2119 := d1438
						snap2120 := d1439
						snap2121 := d1440
						snap2122 := d1441
						snap2123 := d1442
						snap2124 := d1443
						snap2125 := d1606
						snap2126 := d1607
						snap2127 := d1608
						snap2128 := d1609
						snap2129 := d1776
						snap2130 := d1778
						snap2131 := d1779
						snap2132 := d1780
						snap2133 := d1781
						snap2134 := d1782
						snap2135 := d1783
						snap2136 := d1784
						snap2137 := d1785
						snap2138 := d1786
						snap2139 := d1963
						alloc2140 := ctx.SnapshotAllocState()
						bbs[30].Render()
						ctx.RestoreAllocState(alloc2140)
						d1 = snap1964
						d2 = snap1965
						d3 = snap1966
						d4 = snap1967
						d5 = snap1968
						d6 = snap1969
						d7 = snap1970
						d16 = snap1971
						d17 = snap1972
						d19 = snap1973
						d20 = snap1974
						d21 = snap1975
						d23 = snap1976
						d24 = snap1977
						d25 = snap1978
						d42 = snap1979
						d43 = snap1980
						d44 = snap1981
						d45 = snap1982
						d66 = snap1983
						d67 = snap1984
						d68 = snap1985
						d69 = snap1986
						d70 = snap1987
						d71 = snap1988
						d72 = snap1989
						d73 = snap1990
						d74 = snap1991
						d104 = snap1992
						d135 = snap1993
						d136 = snap1994
						d137 = snap1995
						d138 = snap1996
						d139 = snap1997
						d140 = snap1998
						d141 = snap1999
						d142 = snap2000
						d143 = snap2001
						d144 = snap2002
						d145 = snap2003
						d187 = snap2004
						d188 = snap2005
						d189 = snap2006
						d190 = snap2007
						d191 = snap2008
						d192 = snap2009
						d193 = snap2010
						d194 = snap2011
						d244 = snap2012
						d245 = snap2013
						d246 = snap2014
						d247 = snap2015
						d248 = snap2016
						d249 = snap2017
						d250 = snap2018
						d251 = snap2019
						d309 = snap2020
						d310 = snap2021
						d311 = snap2022
						d312 = snap2023
						d313 = snap2024
						d314 = snap2025
						d315 = snap2026
						d316 = snap2027
						d317 = snap2028
						d318 = snap2029
						d319 = snap2030
						d320 = snap2031
						d321 = snap2032
						d322 = snap2033
						d323 = snap2034
						d324 = snap2035
						d325 = snap2036
						d326 = snap2037
						d327 = snap2038
						d328 = snap2039
						d329 = snap2040
						d330 = snap2041
						d331 = snap2042
						d332 = snap2043
						d333 = snap2044
						d334 = snap2045
						d335 = snap2046
						d420 = snap2047
						d421 = snap2048
						d422 = snap2049
						d423 = snap2050
						d424 = snap2051
						d425 = snap2052
						d426 = snap2053
						d427 = snap2054
						d428 = snap2055
						d429 = snap2056
						d430 = snap2057
						d431 = snap2058
						d432 = snap2059
						d433 = snap2060
						d434 = snap2061
						d435 = snap2062
						d436 = snap2063
						d437 = snap2064
						d438 = snap2065
						d439 = snap2066
						d440 = snap2067
						d441 = snap2068
						d442 = snap2069
						d443 = snap2070
						d444 = snap2071
						d445 = snap2072
						d446 = snap2073
						d447 = snap2074
						d560 = snap2075
						d561 = snap2076
						d562 = snap2077
						d563 = snap2078
						d564 = snap2079
						d565 = snap2080
						d566 = snap2081
						d567 = snap2082
						d568 = snap2083
						d569 = snap2084
						d570 = snap2085
						d571 = snap2086
						d572 = snap2087
						d573 = snap2088
						d574 = snap2089
						d702 = snap2090
						d831 = snap2091
						d832 = snap2092
						d833 = snap2093
						d834 = snap2094
						d967 = snap2095
						d968 = snap2096
						d969 = snap2097
						d970 = snap2098
						d971 = snap2099
						d972 = snap2100
						d973 = snap2101
						d974 = snap2102
						d975 = snap2103
						d976 = snap2104
						d977 = snap2105
						d978 = snap2106
						d979 = snap2107
						d980 = snap2108
						d981 = snap2109
						d1129 = snap2110
						d1278 = snap2111
						d1279 = snap2112
						d1280 = snap2113
						d1281 = snap2114
						d1434 = snap2115
						d1435 = snap2116
						d1436 = snap2117
						d1437 = snap2118
						d1438 = snap2119
						d1439 = snap2120
						d1440 = snap2121
						d1441 = snap2122
						d1442 = snap2123
						d1443 = snap2124
						d1606 = snap2125
						d1607 = snap2126
						d1608 = snap2127
						d1609 = snap2128
						d1776 = snap2129
						d1778 = snap2130
						d1779 = snap2131
						d1780 = snap2132
						d1781 = snap2133
						d1782 = snap2134
						d1783 = snap2135
						d1784 = snap2136
						d1785 = snap2137
						d1786 = snap2138
						d1963 = snap2139
					}
					if !bbs[32].Rendered {
						return bbs[32].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d23 = JITPrepareGoSliceArg(ctx, d23)
					if d23.Loc != LocRegTriple && d23.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d23)
					d2141 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d23}, 1)
					d2141.NoHeapPointer = true
					ctx.BindReg(d2141.Reg, &d2141)
					d19 = JITPrepareGoSliceArg(ctx, d19)
					if d19.Loc != LocRegTriple && d19.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d19)
					d2142 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d19}, 1)
					d2142.NoHeapPointer = true
					ctx.BindReg(d2142.Reg, &d2142)
					ctx.EnsureDesc(&d2141)
					ctx.EnsureDesc(&d2142)
					ctx.EnsureDescsTogether(&d2141, &d2142)
					if d2141.Loc == LocImm && d2142.Loc == LocImm {
						d2143 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2141.Imm.Int() < d2142.Imm.Int())}
					} else if d2142.Loc == LocImm {
						r32 := ctx.AllocReg()
						if d2142.Imm.Int() >= -2147483648 && d2142.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2141.Reg, int32(d2142.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2142.Imm.Int()))
							ctx.EmitCmpInt64(d2141.Reg, ctx.ScratchReg)
						}
						d2143 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedLess}
						ctx.BindReg(r32, &d2143)
					} else if d2141.Loc == LocImm {
						r33 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2141.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2142.Reg)
						d2143 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondSignedLess}
						ctx.BindReg(r33, &d2143)
					} else {
						r34 := ctx.AllocReg()
						ctx.EmitCmpInt64(d2141.Reg, d2142.Reg)
						d2143 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r34, Condition: CondSignedLess}
						ctx.BindReg(r34, &d2143)
					}
					ctx.FreeDesc(&d2141)
					ctx.FreeDesc(&d2142)
					d2144 = d2143
					ctx.EnsureDesc(&d2144)
					if d2144.Loc != LocImm && d2144.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d2144.Loc == LocImm {
						if d2144.Imm.Bool() {
							return bbs[29].Render()
						}
						ctx.SyncDesc(&d1438)
						if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
							ctx.ProtectReg(d1438.Reg)
						} else if d1438.Loc == LocRegPair {
							ctx.ProtectReg(d1438.Reg)
							ctx.ProtectReg(d1438.Reg2)
						}
						d2145 = d1438
						if d2145.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d2145)
						ctx.EmitStoreToStack(d2145, int32(bbs[30].PhiBase)+int32(0))
						if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
							ctx.UnprotectReg(d1438.Reg)
						} else if d1438.Loc == LocRegPair {
							ctx.UnprotectReg(d1438.Reg)
							ctx.UnprotectReg(d1438.Reg2)
						}
						return bbs[30].Render()
					}
					lbl37 := ctx.ReserveLabel()
					ctx.EmitJump(d2144.Condition, lbl30)
					ctx.EmitJmp(lbl37)
					ctx.FreeDesc(&d2143)
					snap2146 := d1
					snap2147 := d2
					snap2148 := d3
					snap2149 := d4
					snap2150 := d5
					snap2151 := d6
					snap2152 := d7
					snap2153 := d16
					snap2154 := d17
					snap2155 := d19
					snap2156 := d20
					snap2157 := d21
					snap2158 := d23
					snap2159 := d24
					snap2160 := d25
					snap2161 := d42
					snap2162 := d43
					snap2163 := d44
					snap2164 := d45
					snap2165 := d66
					snap2166 := d67
					snap2167 := d68
					snap2168 := d69
					snap2169 := d70
					snap2170 := d71
					snap2171 := d72
					snap2172 := d73
					snap2173 := d74
					snap2174 := d104
					snap2175 := d135
					snap2176 := d136
					snap2177 := d137
					snap2178 := d138
					snap2179 := d139
					snap2180 := d140
					snap2181 := d141
					snap2182 := d142
					snap2183 := d143
					snap2184 := d144
					snap2185 := d145
					snap2186 := d187
					snap2187 := d188
					snap2188 := d189
					snap2189 := d190
					snap2190 := d191
					snap2191 := d192
					snap2192 := d193
					snap2193 := d194
					snap2194 := d244
					snap2195 := d245
					snap2196 := d246
					snap2197 := d247
					snap2198 := d248
					snap2199 := d249
					snap2200 := d250
					snap2201 := d251
					snap2202 := d309
					snap2203 := d310
					snap2204 := d311
					snap2205 := d312
					snap2206 := d313
					snap2207 := d314
					snap2208 := d315
					snap2209 := d316
					snap2210 := d317
					snap2211 := d318
					snap2212 := d319
					snap2213 := d320
					snap2214 := d321
					snap2215 := d322
					snap2216 := d323
					snap2217 := d324
					snap2218 := d325
					snap2219 := d326
					snap2220 := d327
					snap2221 := d328
					snap2222 := d329
					snap2223 := d330
					snap2224 := d331
					snap2225 := d332
					snap2226 := d333
					snap2227 := d334
					snap2228 := d335
					snap2229 := d420
					snap2230 := d421
					snap2231 := d422
					snap2232 := d423
					snap2233 := d424
					snap2234 := d425
					snap2235 := d426
					snap2236 := d427
					snap2237 := d428
					snap2238 := d429
					snap2239 := d430
					snap2240 := d431
					snap2241 := d432
					snap2242 := d433
					snap2243 := d434
					snap2244 := d435
					snap2245 := d436
					snap2246 := d437
					snap2247 := d438
					snap2248 := d439
					snap2249 := d440
					snap2250 := d441
					snap2251 := d442
					snap2252 := d443
					snap2253 := d444
					snap2254 := d445
					snap2255 := d446
					snap2256 := d447
					snap2257 := d560
					snap2258 := d561
					snap2259 := d562
					snap2260 := d563
					snap2261 := d564
					snap2262 := d565
					snap2263 := d566
					snap2264 := d567
					snap2265 := d568
					snap2266 := d569
					snap2267 := d570
					snap2268 := d571
					snap2269 := d572
					snap2270 := d573
					snap2271 := d574
					snap2272 := d702
					snap2273 := d831
					snap2274 := d832
					snap2275 := d833
					snap2276 := d834
					snap2277 := d967
					snap2278 := d968
					snap2279 := d969
					snap2280 := d970
					snap2281 := d971
					snap2282 := d972
					snap2283 := d973
					snap2284 := d974
					snap2285 := d975
					snap2286 := d976
					snap2287 := d977
					snap2288 := d978
					snap2289 := d979
					snap2290 := d980
					snap2291 := d981
					snap2292 := d1129
					snap2293 := d1278
					snap2294 := d1279
					snap2295 := d1280
					snap2296 := d1281
					snap2297 := d1434
					snap2298 := d1435
					snap2299 := d1436
					snap2300 := d1437
					snap2301 := d1438
					snap2302 := d1439
					snap2303 := d1440
					snap2304 := d1441
					snap2305 := d1442
					snap2306 := d1443
					snap2307 := d1606
					snap2308 := d1607
					snap2309 := d1608
					snap2310 := d1609
					snap2311 := d1776
					snap2312 := d1778
					snap2313 := d1779
					snap2314 := d1780
					snap2315 := d1781
					snap2316 := d1782
					snap2317 := d1783
					snap2318 := d1784
					snap2319 := d1785
					snap2320 := d1786
					snap2321 := d1963
					snap2322 := d2141
					snap2323 := d2142
					snap2324 := d2143
					snap2325 := d2144
					snap2326 := d2145
					alloc2327 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl37)
					ctx.SyncDesc(&d1438)
					if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
						ctx.ProtectReg(d1438.Reg)
					} else if d1438.Loc == LocRegPair {
						ctx.ProtectReg(d1438.Reg)
						ctx.ProtectReg(d1438.Reg2)
					}
					d2328 = d1438
					if d2328.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d2328)
					ctx.EmitStoreToStack(d2328, int32(bbs[30].PhiBase)+int32(0))
					if d1438.Loc == LocReg || d1438.Loc == LocFPReg {
						ctx.UnprotectReg(d1438.Reg)
					} else if d1438.Loc == LocRegPair {
						ctx.UnprotectReg(d1438.Reg)
						ctx.UnprotectReg(d1438.Reg2)
					}
					ctx.EmitJmp(lbl31)
					ctx.RestoreAllocState(alloc2327)
					d1 = snap2146
					d2 = snap2147
					d3 = snap2148
					d4 = snap2149
					d5 = snap2150
					d6 = snap2151
					d7 = snap2152
					d16 = snap2153
					d17 = snap2154
					d19 = snap2155
					d20 = snap2156
					d21 = snap2157
					d23 = snap2158
					d24 = snap2159
					d25 = snap2160
					d42 = snap2161
					d43 = snap2162
					d44 = snap2163
					d45 = snap2164
					d66 = snap2165
					d67 = snap2166
					d68 = snap2167
					d69 = snap2168
					d70 = snap2169
					d71 = snap2170
					d72 = snap2171
					d73 = snap2172
					d74 = snap2173
					d104 = snap2174
					d135 = snap2175
					d136 = snap2176
					d137 = snap2177
					d138 = snap2178
					d139 = snap2179
					d140 = snap2180
					d141 = snap2181
					d142 = snap2182
					d143 = snap2183
					d144 = snap2184
					d145 = snap2185
					d187 = snap2186
					d188 = snap2187
					d189 = snap2188
					d190 = snap2189
					d191 = snap2190
					d192 = snap2191
					d193 = snap2192
					d194 = snap2193
					d244 = snap2194
					d245 = snap2195
					d246 = snap2196
					d247 = snap2197
					d248 = snap2198
					d249 = snap2199
					d250 = snap2200
					d251 = snap2201
					d309 = snap2202
					d310 = snap2203
					d311 = snap2204
					d312 = snap2205
					d313 = snap2206
					d314 = snap2207
					d315 = snap2208
					d316 = snap2209
					d317 = snap2210
					d318 = snap2211
					d319 = snap2212
					d320 = snap2213
					d321 = snap2214
					d322 = snap2215
					d323 = snap2216
					d324 = snap2217
					d325 = snap2218
					d326 = snap2219
					d327 = snap2220
					d328 = snap2221
					d329 = snap2222
					d330 = snap2223
					d331 = snap2224
					d332 = snap2225
					d333 = snap2226
					d334 = snap2227
					d335 = snap2228
					d420 = snap2229
					d421 = snap2230
					d422 = snap2231
					d423 = snap2232
					d424 = snap2233
					d425 = snap2234
					d426 = snap2235
					d427 = snap2236
					d428 = snap2237
					d429 = snap2238
					d430 = snap2239
					d431 = snap2240
					d432 = snap2241
					d433 = snap2242
					d434 = snap2243
					d435 = snap2244
					d436 = snap2245
					d437 = snap2246
					d438 = snap2247
					d439 = snap2248
					d440 = snap2249
					d441 = snap2250
					d442 = snap2251
					d443 = snap2252
					d444 = snap2253
					d445 = snap2254
					d446 = snap2255
					d447 = snap2256
					d560 = snap2257
					d561 = snap2258
					d562 = snap2259
					d563 = snap2260
					d564 = snap2261
					d565 = snap2262
					d566 = snap2263
					d567 = snap2264
					d568 = snap2265
					d569 = snap2266
					d570 = snap2267
					d571 = snap2268
					d572 = snap2269
					d573 = snap2270
					d574 = snap2271
					d702 = snap2272
					d831 = snap2273
					d832 = snap2274
					d833 = snap2275
					d834 = snap2276
					d967 = snap2277
					d968 = snap2278
					d969 = snap2279
					d970 = snap2280
					d971 = snap2281
					d972 = snap2282
					d973 = snap2283
					d974 = snap2284
					d975 = snap2285
					d976 = snap2286
					d977 = snap2287
					d978 = snap2288
					d979 = snap2289
					d980 = snap2290
					d981 = snap2291
					d1129 = snap2292
					d1278 = snap2293
					d1279 = snap2294
					d1280 = snap2295
					d1281 = snap2296
					d1434 = snap2297
					d1435 = snap2298
					d1436 = snap2299
					d1437 = snap2300
					d1438 = snap2301
					d1439 = snap2302
					d1440 = snap2303
					d1441 = snap2304
					d1442 = snap2305
					d1443 = snap2306
					d1606 = snap2307
					d1607 = snap2308
					d1608 = snap2309
					d1609 = snap2310
					d1776 = snap2311
					d1778 = snap2312
					d1779 = snap2313
					d1780 = snap2314
					d1781 = snap2315
					d1782 = snap2316
					d1783 = snap2317
					d1784 = snap2318
					d1785 = snap2319
					d1786 = snap2320
					d1963 = snap2321
					d2141 = snap2322
					d2142 = snap2323
					d2143 = snap2324
					d2144 = snap2325
					d2145 = snap2326
					if !bbs[30].Rendered {
						snap2329 := d1
						snap2330 := d2
						snap2331 := d3
						snap2332 := d4
						snap2333 := d5
						snap2334 := d6
						snap2335 := d7
						snap2336 := d16
						snap2337 := d17
						snap2338 := d19
						snap2339 := d20
						snap2340 := d21
						snap2341 := d23
						snap2342 := d24
						snap2343 := d25
						snap2344 := d42
						snap2345 := d43
						snap2346 := d44
						snap2347 := d45
						snap2348 := d66
						snap2349 := d67
						snap2350 := d68
						snap2351 := d69
						snap2352 := d70
						snap2353 := d71
						snap2354 := d72
						snap2355 := d73
						snap2356 := d74
						snap2357 := d104
						snap2358 := d135
						snap2359 := d136
						snap2360 := d137
						snap2361 := d138
						snap2362 := d139
						snap2363 := d140
						snap2364 := d141
						snap2365 := d142
						snap2366 := d143
						snap2367 := d144
						snap2368 := d145
						snap2369 := d187
						snap2370 := d188
						snap2371 := d189
						snap2372 := d190
						snap2373 := d191
						snap2374 := d192
						snap2375 := d193
						snap2376 := d194
						snap2377 := d244
						snap2378 := d245
						snap2379 := d246
						snap2380 := d247
						snap2381 := d248
						snap2382 := d249
						snap2383 := d250
						snap2384 := d251
						snap2385 := d309
						snap2386 := d310
						snap2387 := d311
						snap2388 := d312
						snap2389 := d313
						snap2390 := d314
						snap2391 := d315
						snap2392 := d316
						snap2393 := d317
						snap2394 := d318
						snap2395 := d319
						snap2396 := d320
						snap2397 := d321
						snap2398 := d322
						snap2399 := d323
						snap2400 := d324
						snap2401 := d325
						snap2402 := d326
						snap2403 := d327
						snap2404 := d328
						snap2405 := d329
						snap2406 := d330
						snap2407 := d331
						snap2408 := d332
						snap2409 := d333
						snap2410 := d334
						snap2411 := d335
						snap2412 := d420
						snap2413 := d421
						snap2414 := d422
						snap2415 := d423
						snap2416 := d424
						snap2417 := d425
						snap2418 := d426
						snap2419 := d427
						snap2420 := d428
						snap2421 := d429
						snap2422 := d430
						snap2423 := d431
						snap2424 := d432
						snap2425 := d433
						snap2426 := d434
						snap2427 := d435
						snap2428 := d436
						snap2429 := d437
						snap2430 := d438
						snap2431 := d439
						snap2432 := d440
						snap2433 := d441
						snap2434 := d442
						snap2435 := d443
						snap2436 := d444
						snap2437 := d445
						snap2438 := d446
						snap2439 := d447
						snap2440 := d560
						snap2441 := d561
						snap2442 := d562
						snap2443 := d563
						snap2444 := d564
						snap2445 := d565
						snap2446 := d566
						snap2447 := d567
						snap2448 := d568
						snap2449 := d569
						snap2450 := d570
						snap2451 := d571
						snap2452 := d572
						snap2453 := d573
						snap2454 := d574
						snap2455 := d702
						snap2456 := d831
						snap2457 := d832
						snap2458 := d833
						snap2459 := d834
						snap2460 := d967
						snap2461 := d968
						snap2462 := d969
						snap2463 := d970
						snap2464 := d971
						snap2465 := d972
						snap2466 := d973
						snap2467 := d974
						snap2468 := d975
						snap2469 := d976
						snap2470 := d977
						snap2471 := d978
						snap2472 := d979
						snap2473 := d980
						snap2474 := d981
						snap2475 := d1129
						snap2476 := d1278
						snap2477 := d1279
						snap2478 := d1280
						snap2479 := d1281
						snap2480 := d1434
						snap2481 := d1435
						snap2482 := d1436
						snap2483 := d1437
						snap2484 := d1438
						snap2485 := d1439
						snap2486 := d1440
						snap2487 := d1441
						snap2488 := d1442
						snap2489 := d1443
						snap2490 := d1606
						snap2491 := d1607
						snap2492 := d1608
						snap2493 := d1609
						snap2494 := d1776
						snap2495 := d1778
						snap2496 := d1779
						snap2497 := d1780
						snap2498 := d1781
						snap2499 := d1782
						snap2500 := d1783
						snap2501 := d1784
						snap2502 := d1785
						snap2503 := d1786
						snap2504 := d1963
						snap2505 := d2141
						snap2506 := d2142
						snap2507 := d2143
						snap2508 := d2144
						snap2509 := d2145
						snap2510 := d2328
						alloc2511 := ctx.SnapshotAllocState()
						bbs[30].Render()
						ctx.RestoreAllocState(alloc2511)
						d1 = snap2329
						d2 = snap2330
						d3 = snap2331
						d4 = snap2332
						d5 = snap2333
						d6 = snap2334
						d7 = snap2335
						d16 = snap2336
						d17 = snap2337
						d19 = snap2338
						d20 = snap2339
						d21 = snap2340
						d23 = snap2341
						d24 = snap2342
						d25 = snap2343
						d42 = snap2344
						d43 = snap2345
						d44 = snap2346
						d45 = snap2347
						d66 = snap2348
						d67 = snap2349
						d68 = snap2350
						d69 = snap2351
						d70 = snap2352
						d71 = snap2353
						d72 = snap2354
						d73 = snap2355
						d74 = snap2356
						d104 = snap2357
						d135 = snap2358
						d136 = snap2359
						d137 = snap2360
						d138 = snap2361
						d139 = snap2362
						d140 = snap2363
						d141 = snap2364
						d142 = snap2365
						d143 = snap2366
						d144 = snap2367
						d145 = snap2368
						d187 = snap2369
						d188 = snap2370
						d189 = snap2371
						d190 = snap2372
						d191 = snap2373
						d192 = snap2374
						d193 = snap2375
						d194 = snap2376
						d244 = snap2377
						d245 = snap2378
						d246 = snap2379
						d247 = snap2380
						d248 = snap2381
						d249 = snap2382
						d250 = snap2383
						d251 = snap2384
						d309 = snap2385
						d310 = snap2386
						d311 = snap2387
						d312 = snap2388
						d313 = snap2389
						d314 = snap2390
						d315 = snap2391
						d316 = snap2392
						d317 = snap2393
						d318 = snap2394
						d319 = snap2395
						d320 = snap2396
						d321 = snap2397
						d322 = snap2398
						d323 = snap2399
						d324 = snap2400
						d325 = snap2401
						d326 = snap2402
						d327 = snap2403
						d328 = snap2404
						d329 = snap2405
						d330 = snap2406
						d331 = snap2407
						d332 = snap2408
						d333 = snap2409
						d334 = snap2410
						d335 = snap2411
						d420 = snap2412
						d421 = snap2413
						d422 = snap2414
						d423 = snap2415
						d424 = snap2416
						d425 = snap2417
						d426 = snap2418
						d427 = snap2419
						d428 = snap2420
						d429 = snap2421
						d430 = snap2422
						d431 = snap2423
						d432 = snap2424
						d433 = snap2425
						d434 = snap2426
						d435 = snap2427
						d436 = snap2428
						d437 = snap2429
						d438 = snap2430
						d439 = snap2431
						d440 = snap2432
						d441 = snap2433
						d442 = snap2434
						d443 = snap2435
						d444 = snap2436
						d445 = snap2437
						d446 = snap2438
						d447 = snap2439
						d560 = snap2440
						d561 = snap2441
						d562 = snap2442
						d563 = snap2443
						d564 = snap2444
						d565 = snap2445
						d566 = snap2446
						d567 = snap2447
						d568 = snap2448
						d569 = snap2449
						d570 = snap2450
						d571 = snap2451
						d572 = snap2452
						d573 = snap2453
						d574 = snap2454
						d702 = snap2455
						d831 = snap2456
						d832 = snap2457
						d833 = snap2458
						d834 = snap2459
						d967 = snap2460
						d968 = snap2461
						d969 = snap2462
						d970 = snap2463
						d971 = snap2464
						d972 = snap2465
						d973 = snap2466
						d974 = snap2467
						d975 = snap2468
						d976 = snap2469
						d977 = snap2470
						d978 = snap2471
						d979 = snap2472
						d980 = snap2473
						d981 = snap2474
						d1129 = snap2475
						d1278 = snap2476
						d1279 = snap2477
						d1280 = snap2478
						d1281 = snap2479
						d1434 = snap2480
						d1435 = snap2481
						d1436 = snap2482
						d1437 = snap2483
						d1438 = snap2484
						d1439 = snap2485
						d1440 = snap2486
						d1441 = snap2487
						d1442 = snap2488
						d1443 = snap2489
						d1606 = snap2490
						d1607 = snap2491
						d1608 = snap2492
						d1609 = snap2493
						d1776 = snap2494
						d1778 = snap2495
						d1779 = snap2496
						d1780 = snap2497
						d1781 = snap2498
						d1782 = snap2499
						d1783 = snap2500
						d1784 = snap2501
						d1785 = snap2502
						d1786 = snap2503
						d1963 = snap2504
						d2141 = snap2505
						d2142 = snap2506
						d2143 = snap2507
						d2144 = snap2508
						d2145 = snap2509
						d2328 = snap2510
					}
					if !bbs[29].Rendered {
						return bbs[29].Render()
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
			JITVirtualArgs: true,
			JITInlineCost:  161,
		},
	})

	// DATEDIFF(date1, date2) - returns number of days between two dates
	Declare(&Globalenv, &Declaration{
		Name: "datediff",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			t1, ok1 := toTime(a[0])
			t2, ok2 := toTime(a[1])
			if !ok1 || !ok2 {
				return NewNil()
			}
			d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
			d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
			days := int64(d1.Sub(d2).Hours() / 24)
			return NewInt(days)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns number of days between two dates (date1 - date2)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "date1", Description: "first date"}, &TypeDescriptor{Kind: "any", Label: "date2", Description: "second date"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["datediff"]
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
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
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
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
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
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
				var bbs [7]BBDescriptor
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
					d10 = JITPrepareScmerGoArg(ctx, d10)
					ctx.SyncDesc(&d10)
					callResults11 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d10}, []uint8{3, 1}, []uint8{4, 0})
					d12 = callResults11[0]
					_ = d12
					d13 = callResults11[1]
					_ = d13
					ctx.FreeDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d12)
					d14 = args[1]
					d14.ID = 0
					d14 = JITPrepareScmerGoArg(ctx, d14)
					ctx.SyncDesc(&d14)
					callResults15 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d14}, []uint8{3, 1}, []uint8{4, 0})
					d16 = callResults15[0]
					_ = d16
					d17 = callResults15[1]
					_ = d17
					ctx.FreeDesc(&d14)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.StabilizeDescForControlFlow(&d17)
					d18 = d13
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap19 := d0
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d9
						snap24 := d10
						snap25 := d12
						snap26 := d13
						snap27 := d14
						snap28 := d16
						snap29 := d17
						snap30 := d18
						alloc31 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc31)
						d0 = snap19
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d9 = snap23
						d10 = snap24
						d12 = snap25
						d13 = snap26
						d14 = snap27
						d16 = snap28
						d17 = snap29
						d18 = snap30
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d13)
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
					d32 = args[1]
					d32.ID = 0
					d34 = d32
					d34.ID = 0
					d33 = ctx.EmitTagEqualsBorrowed(&d34, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d32)
					d35 = d33
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d35.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap36 := d0
						snap37 := d1
						snap38 := d2
						snap39 := d3
						snap40 := d9
						snap41 := d10
						snap42 := d12
						snap43 := d13
						snap44 := d14
						snap45 := d16
						snap46 := d17
						snap47 := d18
						snap48 := d32
						snap49 := d33
						snap50 := d34
						snap51 := d35
						alloc52 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc52)
						d0 = snap36
						d1 = snap37
						d2 = snap38
						d3 = snap39
						d9 = snap40
						d10 = snap41
						d12 = snap42
						d13 = snap43
						d14 = snap44
						d16 = snap45
						d17 = snap46
						d18 = snap47
						d32 = snap48
						d33 = snap49
						d34 = snap50
						d35 = snap51
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d33)
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
					d53 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d53)
					if d53.Loc == LocRegPair || d53.Loc == LocStackPair || d53.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d53, &result)
						result.Type = d53.Type
					} else {
						switch d53.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d53)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d53)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d53)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d53, &result)
							result.Type = d53.Type
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
					ctx.ReclaimUntrackedRegs()
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d12)
					d54 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d12}, 1)
					d54.NoHeapPointer = true
					ctx.BindReg(d54.Reg, &d54)
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d12)
					d55 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d12}, 1)
					d55.NoHeapPointer = true
					ctx.BindReg(d55.Reg, &d55)
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d12)
					d56 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d12}, 1)
					d56.NoHeapPointer = true
					ctx.BindReg(d56.Reg, &d56)
					d57 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d54.Loc == LocRegPair || d54.Loc == LocStackPair || d54.Loc == LocRegTriple || d54.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d55.Loc == LocRegPair || d55.Loc == LocStackPair || d55.Loc == LocRegTriple || d55.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d56.Loc == LocRegPair || d56.Loc == LocStackPair || d56.Loc == LocRegTriple || d56.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocRegTriple || d58.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d59.Loc == LocRegPair || d59.Loc == LocStackPair || d59.Loc == LocRegTriple || d59.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d60 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d60.Loc == LocRegPair || d60.Loc == LocStackPair || d60.Loc == LocRegTriple || d60.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d61 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d61.Loc == LocRegPair || d61.Loc == LocStackPair || d61.Loc == LocRegTriple || d61.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d57.Loc == LocRegPair || d57.Loc == LocStackPair || d57.Loc == LocRegTriple || d57.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d54)
					ctx.SyncDesc(&d55)
					ctx.SyncDesc(&d56)
					ctx.SyncDesc(&d58)
					ctx.SyncDesc(&d59)
					ctx.SyncDesc(&d60)
					ctx.SyncDesc(&d61)
					ctx.SyncDesc(&d57)
					d62 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d54, d55, d56, d58, d59, d60, d61, d57}, 3)
					d62.NoHeapPointer = false
					ctx.BindReg(d62.Reg, &d62)
					ctx.BindReg(d62.Reg2, &d62)
					ctx.BindReg(d62.Reg3, &d62)
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d59)
					ctx.FreeDesc(&d60)
					ctx.FreeDesc(&d61)
					ctx.FreeDesc(&d54)
					ctx.FreeDesc(&d55)
					ctx.FreeDesc(&d56)
					ctx.FreeDesc(&d57)
					d16 = JITPrepareGoSliceArg(ctx, d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d16)
					d63 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d16}, 1)
					d63.NoHeapPointer = true
					ctx.BindReg(d63.Reg, &d63)
					d16 = JITPrepareGoSliceArg(ctx, d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d16)
					d64 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d16}, 1)
					d64.NoHeapPointer = true
					ctx.BindReg(d64.Reg, &d64)
					d16 = JITPrepareGoSliceArg(ctx, d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d16)
					d65 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d16}, 1)
					d65.NoHeapPointer = true
					ctx.BindReg(d65.Reg, &d65)
					d66 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d63.Loc == LocRegPair || d63.Loc == LocStackPair || d63.Loc == LocRegTriple || d63.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d64.Loc == LocRegPair || d64.Loc == LocStackPair || d64.Loc == LocRegTriple || d64.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d65.Loc == LocRegPair || d65.Loc == LocStackPair || d65.Loc == LocRegTriple || d65.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d67 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d67.Loc == LocRegPair || d67.Loc == LocStackPair || d67.Loc == LocRegTriple || d67.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d68.Loc == LocRegPair || d68.Loc == LocStackPair || d68.Loc == LocRegTriple || d68.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d69.Loc == LocRegPair || d69.Loc == LocStackPair || d69.Loc == LocRegTriple || d69.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d66.Loc == LocRegPair || d66.Loc == LocStackPair || d66.Loc == LocRegTriple || d66.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d63)
					ctx.SyncDesc(&d64)
					ctx.SyncDesc(&d65)
					ctx.SyncDesc(&d67)
					ctx.SyncDesc(&d68)
					ctx.SyncDesc(&d69)
					ctx.SyncDesc(&d70)
					ctx.SyncDesc(&d66)
					d71 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d63, d64, d65, d67, d68, d69, d70, d66}, 3)
					d71.NoHeapPointer = false
					ctx.BindReg(d71.Reg, &d71)
					ctx.BindReg(d71.Reg2, &d71)
					ctx.BindReg(d71.Reg3, &d71)
					ctx.FreeDesc(&d67)
					ctx.FreeDesc(&d68)
					ctx.FreeDesc(&d69)
					ctx.FreeDesc(&d70)
					ctx.FreeDesc(&d63)
					ctx.FreeDesc(&d64)
					ctx.FreeDesc(&d65)
					ctx.FreeDesc(&d66)
					d62 = JITPrepareGoSliceArg(ctx, d62)
					if d62.Loc != LocRegTriple && d62.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d71 = JITPrepareGoSliceArg(ctx, d71)
					if d71.Loc != LocRegTriple && d71.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d62)
					ctx.SyncDesc(&d71)
					d72 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d62, d71}, 1)
					d72.NoHeapPointer = true
					ctx.BindReg(d72.Reg, &d72)
					ctx.FreeDesc(&d62)
					ctx.FreeDesc(&d71)
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocRegTriple || d72.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d72)
					d73 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d72}, 1)
					d73.NoHeapPointer = true
					ctx.BindReg(d73.Reg, &d73)
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d73.Imm.Float() / 24)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4627448617123184640))
						ctx.EmitDivFloat64(d73.Reg, ctx.ScratchReg)
						d74 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d74)
					}
					if d74.Loc == LocReg && d73.Loc == LocReg && d74.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					ctx.FreeDesc(&d73)
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d74.Imm.Float()))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r0, d74.Reg)
						d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d75)
					}
					ctx.FreeDesc(&d74)
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
					ctx.ReclaimUntrackedRegs()
					d77 = d17
					ctx.EnsureDesc(&d77)
					if d77.Loc != LocImm && d77.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d77.Loc == LocImm {
						if d77.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d77.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap78 := d0
						snap79 := d1
						snap80 := d2
						snap81 := d3
						snap82 := d9
						snap83 := d10
						snap84 := d12
						snap85 := d13
						snap86 := d14
						snap87 := d16
						snap88 := d17
						snap89 := d18
						snap90 := d32
						snap91 := d33
						snap92 := d34
						snap93 := d35
						snap94 := d53
						snap95 := d54
						snap96 := d55
						snap97 := d56
						snap98 := d57
						snap99 := d58
						snap100 := d59
						snap101 := d60
						snap102 := d61
						snap103 := d62
						snap104 := d63
						snap105 := d64
						snap106 := d65
						snap107 := d66
						snap108 := d67
						snap109 := d68
						snap110 := d69
						snap111 := d70
						snap112 := d71
						snap113 := d72
						snap114 := d73
						snap115 := d74
						snap116 := d75
						snap117 := d76
						snap118 := d77
						alloc119 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc119)
						d0 = snap78
						d1 = snap79
						d2 = snap80
						d3 = snap81
						d9 = snap82
						d10 = snap83
						d12 = snap84
						d13 = snap85
						d14 = snap86
						d16 = snap87
						d17 = snap88
						d18 = snap89
						d32 = snap90
						d33 = snap91
						d34 = snap92
						d35 = snap93
						d53 = snap94
						d54 = snap95
						d55 = snap96
						d56 = snap97
						d57 = snap98
						d58 = snap99
						d59 = snap100
						d60 = snap101
						d61 = snap102
						d62 = snap103
						d63 = snap104
						d64 = snap105
						d65 = snap106
						d66 = snap107
						d67 = snap108
						d68 = snap109
						d69 = snap110
						d70 = snap111
						d71 = snap112
						d72 = snap113
						d73 = snap114
						d74 = snap115
						d75 = snap116
						d76 = snap117
						d77 = snap118
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
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
			JITVirtualArgs: true,
			JITInlineCost:  40,
		},
	})

	// STR_TO_DATE(str, format) - parse string with MySQL format to date
	Declare(&Globalenv, &Declaration{
		Name: "str_to_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			// convert MySQL format to Go format
			mysqlFmt := a[1].String()
			goFmt := mysqlFormatToGo(mysqlFmt)
			if t, err := time.Parse(goFmt, a[0].String()); err == nil {
				return NewDate(t.Unix())
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses a string with MySQL format specifiers to a date",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "date string"}, &TypeDescriptor{Kind: "string", Label: "format", Description: "MySQL format string (e.g. %Y-%m-%d)"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["str_to_date"]
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
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
					d10 = args[1]
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
						panic("jit: generic call arg expects 2-word value (mysqlFormatToGo arg0)")
					}
					ctx.SyncDesc(&d11)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(mysqlFormatToGo), []JITValueDesc{d11}, 2)
					d13.NoHeapPointer = false
					ctx.BindReg(d13.Reg, &d13)
					ctx.BindReg(d13.Reg2, &d13)
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
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
						panic("jit: generic call arg expects 2-word value (time.Parse arg0)")
					}
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d15.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d15.Imm)
						ptrWord, _ := d15.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d15.Imm.String())))
						d15 = tmpPair
					} else if d15.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d15.Type, Reg: ctx.AllocRegExcept(d15.Reg), Reg2: ctx.AllocRegExcept(d15.Reg)}
						switch d15.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d15)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d15)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d15)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d15)
						d15 = tmpPair
					}
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (time.Parse arg1)")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d15)
					callResults17 := JITEmitGoCallResults(ctx, GoFuncAddr(time.Parse), []JITValueDesc{d13, d15}, []uint8{3, 2}, []uint8{4, 3})
					d18 = callResults17[0]
					_ = d18
					d19 = callResults17[1]
					_ = d19
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc != LocReg && d19.Loc != LocRegPair && d19.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 0)
						ctx.EmitSetcc(r0, CondEqual)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d20)
					}
					ctx.FreeDesc(&d19)
					d21 = d20
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
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
						snap34 := d18
						snap35 := d19
						snap36 := d20
						snap37 := d21
						alloc38 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc38)
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
						d18 = snap34
						d19 = snap35
						d20 = snap36
						d21 = snap37
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d20)
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
					d18 = JITPrepareGoSliceArg(ctx, d18)
					if d18.Loc != LocRegTriple && d18.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d18)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d18}, 1)
					d39.NoHeapPointer = true
					ctx.BindReg(d39.Reg, &d39)
					if d39.Loc == LocRegPair || d39.Loc == LocStackPair || d39.Loc == LocRegTriple || d39.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d39)
					d40 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d39}, 2)
					d40.NoHeapPointer = false
					ctx.BindReg(d40.Reg, &d40)
					ctx.BindReg(d40.Reg2, &d40)
					ctx.FreeDesc(&d39)
					ctx.SyncDesc(&d40)
					if d40.Loc == LocRegPair || d40.Loc == LocStackPair || d40.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d40, &result)
						result.Type = d40.Type
					} else {
						switch d40.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d40)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d40)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d40)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d40, &result)
							result.Type = d40.Type
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
					d41 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d41)
					if d41.Loc == LocRegPair || d41.Loc == LocStackPair || d41.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d41, &result)
						result.Type = d41.Type
					} else {
						switch d41.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d41)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d41)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d41)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d41, &result)
							result.Type = d41.Type
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
			JITVirtualArgs: true,
			JITInlineCost:  23,
		},
	})
}

// mysqlFormatToGo converts a MySQL date format string to a Go time format string.
func mysqlFormatToGo(format string) string {
	var buf strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'Y':
				buf.WriteString("2006")
			case 'y':
				buf.WriteString("06")
			case 'm':
				buf.WriteString("01")
			case 'd':
				buf.WriteString("02")
			case 'H':
				buf.WriteString("15")
			case 'i':
				buf.WriteString("04")
			case 's':
				buf.WriteString("05")
			case '%':
				buf.WriteByte('%')
			default:
				buf.WriteByte(format[i+1])
			}
			i++
		} else {
			buf.WriteByte(format[i])
		}
	}
	return buf.String()
}
