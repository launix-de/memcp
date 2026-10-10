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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // embed IANA timezone database
)

// tzAbbrevMap maps common timezone abbreviations to IANA zone names.
// Used as fallback when time.LoadLocation fails for an abbreviation.
var tzAbbrevMap = map[string]string{
	"UTC": "UTC", "GMT": "UTC",
	"CET": "Europe/Paris", "CEST": "Europe/Paris",
	"WET": "Europe/Lisbon", "WEST": "Europe/Lisbon",
	"EET": "Europe/Helsinki", "EEST": "Europe/Helsinki",
	"MSK": "Europe/Moscow",
	"EST": "America/New_York", "EDT": "America/New_York",
	"CST": "America/Chicago", "CDT": "America/Chicago",
	"MST": "America/Denver", "MDT": "America/Denver",
	"PST": "America/Los_Angeles", "PDT": "America/Los_Angeles",
	"AKST": "America/Anchorage", "AKDT": "America/Anchorage",
	"HST":    "Pacific/Honolulu",
	"IST":    "Asia/Kolkata",
	"JST":    "Asia/Tokyo",
	"KST":    "Asia/Seoul",
	"CST_CN": "Asia/Shanghai",
	"AEST":   "Australia/Sydney", "AEDT": "Australia/Sydney",
	"NZST": "Pacific/Auckland", "NZDT": "Pacific/Auckland",
}

// tzLocationCache caches resolved *time.Location values by name to avoid repeated IANA parsing.
var tzLocationCache sync.Map // map[string]*time.Location

// ResolveLocation resolves a timezone name string to a *time.Location.
// Accepts: "UTC", "SYSTEM", "+HH:MM" / "-HH:MM" offsets, IANA names, abbreviations.
// Results are cached to avoid repeated parsing of the embedded IANA timezone database.
func ResolveLocation(name string) (*time.Location, error) {
	if v, ok := tzLocationCache.Load(name); ok {
		return v.(*time.Location), nil
	}
	loc, err := resolveLocationUncached(name)
	if err == nil {
		tzLocationCache.Store(name, loc)
	}
	return loc, err
}

func resolveLocationUncached(name string) (*time.Location, error) {
	switch strings.ToUpper(name) {
	case "UTC", "UTC+0", "UTC-0", "+00:00", "-00:00", "+0:00", "-0:00":
		return time.UTC, nil
	case "SYSTEM", "LOCAL":
		return time.Local, nil
	}
	// Fixed offset: +HH:MM or -HH:MM
	if len(name) >= 3 && (name[0] == '+' || name[0] == '-') {
		loc, err := parseFixedOffset(name)
		if err == nil {
			return loc, nil
		}
	}
	// IANA named zone
	if loc, err := time.LoadLocation(name); err == nil {
		return loc, nil
	}
	// Abbreviation fallback
	if iana, ok := tzAbbrevMap[strings.ToUpper(name)]; ok {
		return time.LoadLocation(iana)
	}
	return nil, fmt.Errorf("unknown timezone: %q", name)
}

// parseFixedOffset parses "+HH:MM", "+H:MM", or "+HH" into a fixed-offset location.
func parseFixedOffset(s string) (*time.Location, error) {
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	s = s[1:]
	var h, m int
	var err error
	switch {
	case len(s) == 5 && s[2] == ':':
		h, err = strconv.Atoi(s[0:2])
		if err == nil {
			m, err = strconv.Atoi(s[3:5])
		}
	case len(s) == 4 && s[2] == ':':
		h, err = strconv.Atoi(s[0:2])
		if err == nil {
			m, err = strconv.Atoi(s[3:4])
		}
	case len(s) == 2:
		h, err = strconv.Atoi(s)
	default:
		return nil, fmt.Errorf("cannot parse offset %q", s)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot parse offset %q: %w", s, err)
	}
	offset := sign * (h*3600 + m*60)
	name := fmt.Sprintf("%+03d:%02d", sign*h, m)
	return time.FixedZone(name, offset), nil
}

// GetSessionLocation resolves time_zone from an explicitly passed Scheme session.
func GetSessionLocation(sessionScmer Scmer) *time.Location {
	tz := Apply(sessionScmer, NewString("time_zone"))
	if tz.IsNil() {
		return time.UTC
	}
	loc, err := ResolveLocation(tz.String())
	if err != nil {
		return time.UTC
	}
	return loc
}

// DateToDisplay formats a tagDate Scmer value for display, respecting zone_id and session TZ.
// If the value's zone_id != 0, displays in that zone; otherwise uses sessionLoc.
func DateToDisplay(v Scmer, sessionLoc *time.Location) string {
	unix := TagDateDecodeUnix(auxVal(v.aux))
	if unix == mysqlZeroDateUnix {
		return "0000-00-00 00:00:00"
	}
	zoneID := TagDateDecodeZone(auxVal(v.aux))
	loc := sessionLoc
	if loc == nil {
		loc = time.UTC
	}
	if zoneID != 0 {
		// zone_id is set — look up via GlobalZoneRegistry (set at startup from system.timezones).
		// For now: use UTC (zone registry is populated later in the implementation).
		// TODO: look up zone by ID from zone registry
		loc = time.UTC
	}
	return time.Unix(unix, 0).In(loc).Format("2006-01-02 15:04:05")
}

func init_timezone() {
	DeclareTitle("Timezone")

	// UNIX_TIMESTAMP(): returns current unix timestamp as integer
	// UNIX_TIMESTAMP(dt): converts datetime string to unix timestamp integer
	Declare(&Globalenv, &Declaration{
		Name: "unix_timestamp",

		Fn: func(a ...Scmer) Scmer {
			if len(a) == 0 {
				return NewInt(time.Now().Unix())
			}
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			return NewInt(t.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a unix timestamp (integer seconds since epoch)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "dt", Description: "optional datetime value to convert", Optional: true}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["unix_timestamp"]
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
				var d13 JITValueDesc
				_ = d13
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
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
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						d1 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d2.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d1)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
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
					ctx.ReclaimUntrackedRegs()
					d7 = ctx.EmitGoCallScalar(GoFuncAddr(time.Now), []JITValueDesc{}, 3)
					d7.NoHeapPointer = false
					ctx.BindReg(d7.Reg, &d7)
					ctx.BindReg(d7.Reg2, &d7)
					ctx.BindReg(d7.Reg3, &d7)
					d7 = JITPrepareGoSliceArg(ctx, d7)
					if d7.Loc != LocRegTriple && d7.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d7)
					d8 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d7}, 1)
					d8.NoHeapPointer = true
					ctx.BindReg(d8.Reg, &d8)
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						ctx.EmitMakeInt(result, d8)
					} else {
						ctx.EmitMovToReg(result.Reg2, d8)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d9)
						if d8.Loc == LocReg && d8.Reg != result.Reg2 {
							ctx.FreeReg(d8.Reg)
						}
					}
					result.Type = tagInt
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
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						snap21 := d11
						snap22 := d12
						snap23 := d13
						alloc24 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc24)
						d0 = snap14
						d1 = snap15
						d2 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
						d11 = snap21
						d12 = snap22
						d13 = snap23
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
					d26 = args[0]
					d26.ID = 0
					d26 = JITPrepareScmerGoArg(ctx, d26)
					ctx.SyncDesc(&d26)
					callResults27 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d26}, []uint8{3, 1}, []uint8{4, 0})
					d28 = callResults27[0]
					_ = d28
					d29 = callResults27[1]
					_ = d29
					ctx.FreeDesc(&d26)
					ctx.StabilizeDescForControlFlow(&d28)
					d30 = d29
					ctx.EnsureDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap31 := d0
						snap32 := d1
						snap33 := d2
						snap34 := d7
						snap35 := d8
						snap36 := d9
						snap37 := d10
						snap38 := d11
						snap39 := d12
						snap40 := d13
						snap41 := d25
						snap42 := d26
						snap43 := d28
						snap44 := d29
						snap45 := d30
						alloc46 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc46)
						d0 = snap31
						d1 = snap32
						d2 = snap33
						d7 = snap34
						d8 = snap35
						d9 = snap36
						d10 = snap37
						d11 = snap38
						d12 = snap39
						d13 = snap40
						d25 = snap41
						d26 = snap42
						d28 = snap43
						d29 = snap44
						d30 = snap45
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d29)
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
					d47 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d47)
					if d47.Loc == LocRegPair || d47.Loc == LocStackPair || d47.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d47, &result)
						result.Type = d47.Type
					} else {
						switch d47.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d47)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d47)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d47)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d47, &result)
							result.Type = d47.Type
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
					d28 = JITPrepareGoSliceArg(ctx, d28)
					if d28.Loc != LocRegTriple && d28.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d28)
					d48 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d28}, 1)
					d48.NoHeapPointer = true
					ctx.BindReg(d48.Reg, &d48)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						ctx.EmitMakeInt(result, d48)
					} else {
						ctx.EmitMovToReg(result.Reg2, d48)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d49)
						if d48.Loc == LocReg && d48.Reg != result.Reg2 {
							ctx.FreeReg(d48.Reg)
						}
					}
					result.Type = tagInt
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
			JITInlineCost:  24,
		},
	})

	// system_time_zone: returns the OS-level timezone name
	Declare(&Globalenv, &Declaration{
		Name: "system_time_zone",

		Fn: func(a ...Scmer) Scmer {
			return NewString(time.Local.String())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the operating system's local timezone name",
			Return: &TypeDescriptor{Kind: "string"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["system_time_zone"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.Local }), nil, 1)
				if d0.Loc == LocRegPair || d0.Loc == LocStackPair || d0.Loc == LocRegTriple || d0.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((*time.Location).String), []JITValueDesc{d0}, 2)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d1}, 2)
				if result.Loc == LocAny {
					return d2
				}
				ctx.EmitMovPairToResult(&d2, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 4,
		},
	})

	// CONVERT_TZ(dt, from_tz, to_tz)
	Declare(&Globalenv, &Declaration{
		Name: "convert_tz",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			fromLoc, err := ResolveLocation(a[1].String())
			if err != nil {
				return NewNil()
			}
			toLoc, err := ResolveLocation(a[2].String())
			if err != nil {
				return NewNil()
			}
			// parse the input as a wall-clock time in fromLoc
			var t time.Time
			switch a[0].GetTag() {
			case tagDate:
				// tagDate stores a naive UTC unix (wall-clock as UTC); reinterpret as local in fromLoc
				wall := time.Unix(a[0].Int(), 0).UTC()
				t = time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), 0, fromLoc)
			default:
				unix, ok := parseDateStringInLoc(a[0].String(), fromLoc)
				if !ok {
					return NewNil()
				}
				t = time.Unix(unix, 0)
			}
			// convert to target zone; encode result as naive UTC (wall-clock in toLoc stored as UTC)
			tInTo := t.In(toLoc)
			naive := time.Date(tInTo.Year(), tInTo.Month(), tInTo.Day(), tInTo.Hour(), tInTo.Minute(), tInTo.Second(), 0, time.UTC)
			return NewDate(naive.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "converts a datetime from one timezone to another",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "dt", Description: "datetime value"}, &TypeDescriptor{Kind: "string", Label: "from_tz", Description: "source timezone"}, &TypeDescriptor{Kind: "string", Label: "to_tz", Description: "target timezone"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["convert_tz"]
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
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
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
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
				var d170 JITValueDesc
				_ = d170
				var d171 JITValueDesc
				_ = d171
				var d172 JITValueDesc
				_ = d172
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
				var d194 JITValueDesc
				_ = d194
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d265 JITValueDesc
				_ = d265
				var d266 JITValueDesc
				_ = d266
				var d267 JITValueDesc
				_ = d267
				var d268 JITValueDesc
				_ = d268
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
				var bbs [14]BBDescriptor
				bbs[9].PhiBase = int32(phiBase0) + int32(0)
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
					d13 = args[1]
					d13.ID = 0
					d15 = d13
					ctx.SyncDesc(&d15)
					if d15.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d15.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d15.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d15 = tmpScalar
					}
					d15 = JITPrepareScmerGoArg(ctx, d15)
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d15}, 2)
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d14.Imm)
						ptrWord, _ := d14.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d14.Imm.String())))
						d14 = tmpPair
					} else if d14.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocRegExcept(d14.Reg), Reg2: ctx.AllocRegExcept(d14.Reg)}
						switch d14.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d14)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d14)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d14)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d14)
						d14 = tmpPair
					}
					if d14.Loc != LocRegPair && d14.Loc != LocStackPair && d14.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d14)
					callResults16 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d14}, []uint8{1, 2}, []uint8{1, 3})
					d17 = callResults16[0]
					_ = d17
					d18 = callResults16[1]
					_ = d18
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d18.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d18)
						if d18.Loc != LocReg && d18.Loc != LocRegPair && d18.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpRegImm32(d18.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d19)
					}
					ctx.FreeDesc(&d18)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap21 := d1
						snap22 := d2
						snap23 := d3
						snap24 := d4
						snap25 := d5
						snap26 := d12
						snap27 := d13
						snap28 := d14
						snap29 := d15
						snap30 := d17
						snap31 := d18
						snap32 := d19
						snap33 := d20
						alloc34 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc34)
						d1 = snap21
						d2 = snap22
						d3 = snap23
						d4 = snap24
						d5 = snap25
						d12 = snap26
						d13 = snap27
						d14 = snap28
						d15 = snap29
						d17 = snap30
						d18 = snap31
						d19 = snap32
						d20 = snap33
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d19)
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
					d35 = args[2]
					d35.ID = 0
					d37 = d35
					d37.ID = 0
					d36 = ctx.EmitTagEqualsBorrowed(&d37, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d35)
					d38 = d36
					ctx.EnsureDesc(&d38)
					if d38.Loc != LocImm && d38.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d38.Loc == LocImm {
						if d38.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d38.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap39 := d1
						snap40 := d2
						snap41 := d3
						snap42 := d4
						snap43 := d5
						snap44 := d12
						snap45 := d13
						snap46 := d14
						snap47 := d15
						snap48 := d17
						snap49 := d18
						snap50 := d19
						snap51 := d20
						snap52 := d35
						snap53 := d36
						snap54 := d37
						snap55 := d38
						alloc56 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc56)
						d1 = snap39
						d2 = snap40
						d3 = snap41
						d4 = snap42
						d5 = snap43
						d12 = snap44
						d13 = snap45
						d14 = snap46
						d15 = snap47
						d17 = snap48
						d18 = snap49
						d19 = snap50
						d20 = snap51
						d35 = snap52
						d36 = snap53
						d37 = snap54
						d38 = snap55
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d36)
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
					d57 = args[1]
					d57.ID = 0
					d59 = d57
					d59.ID = 0
					d58 = ctx.EmitTagEqualsBorrowed(&d59, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d57)
					d60 = d58
					ctx.EnsureDesc(&d60)
					if d60.Loc != LocImm && d60.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d60.Loc == LocImm {
						if d60.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d60.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap61 := d1
						snap62 := d2
						snap63 := d3
						snap64 := d4
						snap65 := d5
						snap66 := d12
						snap67 := d13
						snap68 := d14
						snap69 := d15
						snap70 := d17
						snap71 := d18
						snap72 := d19
						snap73 := d20
						snap74 := d35
						snap75 := d36
						snap76 := d37
						snap77 := d38
						snap78 := d57
						snap79 := d58
						snap80 := d59
						snap81 := d60
						alloc82 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc82)
						d1 = snap61
						d2 = snap62
						d3 = snap63
						d4 = snap64
						d5 = snap65
						d12 = snap66
						d13 = snap67
						d14 = snap68
						d15 = snap69
						d17 = snap70
						d18 = snap71
						d19 = snap72
						d20 = snap73
						d35 = snap74
						d36 = snap75
						d37 = snap76
						d38 = snap77
						d57 = snap78
						d58 = snap79
						d59 = snap80
						d60 = snap81
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d58)
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
					d83 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d83)
					if d83.Loc == LocRegPair || d83.Loc == LocStackPair || d83.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d83, &result)
						result.Type = d83.Type
					} else {
						switch d83.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d83)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d83)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d83)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d83, &result)
							result.Type = d83.Type
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
					d84 = args[2]
					d84.ID = 0
					d86 = d84
					ctx.SyncDesc(&d86)
					if d86.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d86.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d86.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d86 = tmpScalar
					}
					d86 = JITPrepareScmerGoArg(ctx, d86)
					if d86.Loc != LocRegPair && d86.Loc != LocStackPair && d86.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d85 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d86}, 2)
					ctx.FreeDesc(&d84)
					ctx.EnsureDesc(&d85)
					if d85.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d85.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d85.Imm)
						ptrWord, _ := d85.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d85.Imm.String())))
						d85 = tmpPair
					} else if d85.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d85.Type, Reg: ctx.AllocRegExcept(d85.Reg), Reg2: ctx.AllocRegExcept(d85.Reg)}
						switch d85.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d85)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d85)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d85)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d85)
						d85 = tmpPair
					}
					if d85.Loc != LocRegPair && d85.Loc != LocStackPair && d85.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d85)
					callResults87 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d85}, []uint8{1, 2}, []uint8{1, 3})
					d88 = callResults87[0]
					_ = d88
					d89 = callResults87[1]
					_ = d89
					ctx.StabilizeDescForControlFlow(&d88)
					ctx.EnsureDesc(&d89)
					if d89.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d89.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d89)
						if d89.Loc != LocReg && d89.Loc != LocRegPair && d89.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d89.Reg)
						ctx.EmitCmpRegImm32(d89.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d90 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d90)
					}
					ctx.FreeDesc(&d89)
					d91 = d90
					ctx.EnsureDesc(&d91)
					if d91.Loc != LocImm && d91.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d91.Loc == LocImm {
						if d91.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitCmpRegImm32(d91.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap92 := d1
						snap93 := d2
						snap94 := d3
						snap95 := d4
						snap96 := d5
						snap97 := d12
						snap98 := d13
						snap99 := d14
						snap100 := d15
						snap101 := d17
						snap102 := d18
						snap103 := d19
						snap104 := d20
						snap105 := d35
						snap106 := d36
						snap107 := d37
						snap108 := d38
						snap109 := d57
						snap110 := d58
						snap111 := d59
						snap112 := d60
						snap113 := d83
						snap114 := d84
						snap115 := d85
						snap116 := d86
						snap117 := d88
						snap118 := d89
						snap119 := d90
						snap120 := d91
						alloc121 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc121)
						d1 = snap92
						d2 = snap93
						d3 = snap94
						d4 = snap95
						d5 = snap96
						d12 = snap97
						d13 = snap98
						d14 = snap99
						d15 = snap100
						d17 = snap101
						d18 = snap102
						d19 = snap103
						d20 = snap104
						d35 = snap105
						d36 = snap106
						d37 = snap107
						d38 = snap108
						d57 = snap109
						d58 = snap110
						d59 = snap111
						d60 = snap112
						d83 = snap113
						d84 = snap114
						d85 = snap115
						d86 = snap116
						d88 = snap117
						d89 = snap118
						d90 = snap119
						d91 = snap120
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d90)
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
					d122 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d122)
					if d122.Loc == LocRegPair || d122.Loc == LocStackPair || d122.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d122, &result)
						result.Type = d122.Type
					} else {
						switch d122.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d122)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d122)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d122)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d122, &result)
							result.Type = d122.Type
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
					d123 = args[0]
					d123.ID = 0
					d124 = d123
					d124.ID = 0
					d125 = ctx.EmitGetTagDesc(&d124, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d123)
					ctx.EnsureDesc(&d125)
					if d125.Loc == LocImm {
						d126 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d125.Imm.Int()) == uint64(0x10))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d125.Reg, 16)
						d126 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d126)
					}
					ctx.FreeDesc(&d125)
					d127 = d126
					ctx.EnsureDesc(&d127)
					if d127.Loc != LocImm && d127.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d127.Loc == LocImm {
						if d127.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitJump(d127.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d126)
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap128 := d1
						snap129 := d2
						snap130 := d3
						snap131 := d4
						snap132 := d5
						snap133 := d12
						snap134 := d13
						snap135 := d14
						snap136 := d15
						snap137 := d17
						snap138 := d18
						snap139 := d19
						snap140 := d20
						snap141 := d35
						snap142 := d36
						snap143 := d37
						snap144 := d38
						snap145 := d57
						snap146 := d58
						snap147 := d59
						snap148 := d60
						snap149 := d83
						snap150 := d84
						snap151 := d85
						snap152 := d86
						snap153 := d88
						snap154 := d89
						snap155 := d90
						snap156 := d91
						snap157 := d122
						snap158 := d123
						snap159 := d124
						snap160 := d125
						snap161 := d126
						snap162 := d127
						alloc163 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc163)
						d1 = snap128
						d2 = snap129
						d3 = snap130
						d4 = snap131
						d5 = snap132
						d12 = snap133
						d13 = snap134
						d14 = snap135
						d15 = snap136
						d17 = snap137
						d18 = snap138
						d19 = snap139
						d20 = snap140
						d35 = snap141
						d36 = snap142
						d37 = snap143
						d38 = snap144
						d57 = snap145
						d58 = snap146
						d59 = snap147
						d60 = snap148
						d83 = snap149
						d84 = snap150
						d85 = snap151
						d86 = snap152
						d88 = snap153
						d89 = snap154
						d90 = snap155
						d91 = snap156
						d122 = snap157
						d123 = snap158
						d124 = snap159
						d125 = snap160
						d126 = snap161
						d127 = snap162
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d1 = JITPrepareGoSliceArg(ctx, d1)
					if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					if d88.Loc == LocRegPair || d88.Loc == LocStackPair || d88.Loc == LocRegTriple || d88.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d88)
					d164 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d1, d88}, 3)
					d164.NoHeapPointer = false
					ctx.BindReg(d164.Reg, &d164)
					ctx.BindReg(d164.Reg2, &d164)
					ctx.BindReg(d164.Reg3, &d164)
					ctx.FreeDesc(&d1)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d164)
					d165 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d164}, 1)
					d165.NoHeapPointer = true
					ctx.BindReg(d165.Reg, &d165)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d164)
					d166 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d164}, 1)
					d166.NoHeapPointer = true
					ctx.BindReg(d166.Reg, &d166)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d164)
					d167 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d164}, 1)
					d167.NoHeapPointer = true
					ctx.BindReg(d167.Reg, &d167)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d164)
					d168 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d164}, 1)
					d168.NoHeapPointer = true
					ctx.BindReg(d168.Reg, &d168)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d164)
					d169 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d164}, 1)
					d169.NoHeapPointer = true
					ctx.BindReg(d169.Reg, &d169)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d164)
					d170 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d164}, 1)
					d170.NoHeapPointer = true
					ctx.BindReg(d170.Reg, &d170)
					ctx.FreeDesc(&d164)
					d171 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d165.Loc == LocRegPair || d165.Loc == LocStackPair || d165.Loc == LocRegTriple || d165.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d166.Loc == LocRegPair || d166.Loc == LocStackPair || d166.Loc == LocRegTriple || d166.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d167.Loc == LocRegPair || d167.Loc == LocStackPair || d167.Loc == LocRegTriple || d167.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d168.Loc == LocRegPair || d168.Loc == LocStackPair || d168.Loc == LocRegTriple || d168.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d169.Loc == LocRegPair || d169.Loc == LocStackPair || d169.Loc == LocRegTriple || d169.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d170.Loc == LocRegPair || d170.Loc == LocStackPair || d170.Loc == LocRegTriple || d170.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d172 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d172.Loc == LocRegPair || d172.Loc == LocStackPair || d172.Loc == LocRegTriple || d172.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d171.Loc == LocRegPair || d171.Loc == LocStackPair || d171.Loc == LocRegTriple || d171.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d165)
					ctx.SyncDesc(&d166)
					ctx.SyncDesc(&d167)
					ctx.SyncDesc(&d168)
					ctx.SyncDesc(&d169)
					ctx.SyncDesc(&d170)
					ctx.SyncDesc(&d172)
					ctx.SyncDesc(&d171)
					d173 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d165, d166, d167, d168, d169, d170, d172, d171}, 3)
					d173.NoHeapPointer = false
					ctx.BindReg(d173.Reg, &d173)
					ctx.BindReg(d173.Reg2, &d173)
					ctx.BindReg(d173.Reg3, &d173)
					ctx.FreeDesc(&d172)
					ctx.FreeDesc(&d165)
					ctx.FreeDesc(&d166)
					ctx.FreeDesc(&d167)
					ctx.FreeDesc(&d168)
					ctx.FreeDesc(&d169)
					ctx.FreeDesc(&d170)
					ctx.FreeDesc(&d171)
					d173 = JITPrepareGoSliceArg(ctx, d173)
					if d173.Loc != LocRegTriple && d173.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d173)
					d174 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d173}, 1)
					d174.NoHeapPointer = true
					ctx.BindReg(d174.Reg, &d174)
					ctx.FreeDesc(&d173)
					if d174.Loc == LocRegPair || d174.Loc == LocStackPair || d174.Loc == LocRegTriple || d174.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d174)
					d175 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d174}, 2)
					d175.NoHeapPointer = false
					ctx.BindReg(d175.Reg, &d175)
					ctx.BindReg(d175.Reg2, &d175)
					ctx.FreeDesc(&d174)
					ctx.SyncDesc(&d175)
					if d175.Loc == LocRegPair || d175.Loc == LocStackPair || d175.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d175, &result)
						result.Type = d175.Type
					} else {
						switch d175.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d175)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d175)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d175)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d175, &result)
							result.Type = d175.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d176 = args[0]
					d176.ID = 0
					if d176.Loc == LocImm {
						d177 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d176.Imm.Int())}
					} else if d176.Type == tagInt && d176.Loc == LocRegPair {
						ctx.FreeReg(d176.Reg)
						d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d176.Reg2}
						ctx.BindReg(d176.Reg2, &d177)
						ctx.BindReg(d176.Reg2, &d177)
					} else if d176.Type == tagInt && d176.Loc == LocReg {
						d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d176.Reg}
						ctx.BindReg(d176.Reg, &d177)
						ctx.BindReg(d176.Reg, &d177)
					} else {
						d177 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d176}, 1)
						d177.Type = tagInt
						ctx.BindReg(d177.Reg, &d177)
					}
					ctx.FreeDesc(&d176)
					if d177.Loc == LocRegPair || d177.Loc == LocStackPair || d177.Loc == LocRegTriple || d177.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d178 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d178.Loc == LocRegPair || d178.Loc == LocStackPair || d178.Loc == LocRegTriple || d178.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d177)
					ctx.SyncDesc(&d178)
					d179 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d177, d178}, 3)
					d179.NoHeapPointer = false
					ctx.BindReg(d179.Reg, &d179)
					ctx.BindReg(d179.Reg2, &d179)
					ctx.BindReg(d179.Reg3, &d179)
					ctx.FreeDesc(&d178)
					ctx.FreeDesc(&d177)
					d179 = JITPrepareGoSliceArg(ctx, d179)
					if d179.Loc != LocRegTriple && d179.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d179)
					d180 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d179}, 3)
					d180.NoHeapPointer = false
					ctx.BindReg(d180.Reg, &d180)
					ctx.BindReg(d180.Reg2, &d180)
					ctx.BindReg(d180.Reg3, &d180)
					ctx.FreeDesc(&d179)
					d180 = JITPrepareGoSliceArg(ctx, d180)
					if d180.Loc != LocRegTriple && d180.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d180)
					d181 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d180}, 1)
					d181.NoHeapPointer = true
					ctx.BindReg(d181.Reg, &d181)
					d180 = JITPrepareGoSliceArg(ctx, d180)
					if d180.Loc != LocRegTriple && d180.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d180)
					d182 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d180}, 1)
					d182.NoHeapPointer = true
					ctx.BindReg(d182.Reg, &d182)
					d180 = JITPrepareGoSliceArg(ctx, d180)
					if d180.Loc != LocRegTriple && d180.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d180)
					d183 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d180}, 1)
					d183.NoHeapPointer = true
					ctx.BindReg(d183.Reg, &d183)
					d180 = JITPrepareGoSliceArg(ctx, d180)
					if d180.Loc != LocRegTriple && d180.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d180)
					d184 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d180}, 1)
					d184.NoHeapPointer = true
					ctx.BindReg(d184.Reg, &d184)
					d180 = JITPrepareGoSliceArg(ctx, d180)
					if d180.Loc != LocRegTriple && d180.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d180)
					d185 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d180}, 1)
					d185.NoHeapPointer = true
					ctx.BindReg(d185.Reg, &d185)
					d180 = JITPrepareGoSliceArg(ctx, d180)
					if d180.Loc != LocRegTriple && d180.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d180)
					d186 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d180}, 1)
					d186.NoHeapPointer = true
					ctx.BindReg(d186.Reg, &d186)
					ctx.FreeDesc(&d180)
					if d181.Loc == LocRegPair || d181.Loc == LocStackPair || d181.Loc == LocRegTriple || d181.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d182.Loc == LocRegPair || d182.Loc == LocStackPair || d182.Loc == LocRegTriple || d182.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d183.Loc == LocRegPair || d183.Loc == LocStackPair || d183.Loc == LocRegTriple || d183.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d184.Loc == LocRegPair || d184.Loc == LocStackPair || d184.Loc == LocRegTriple || d184.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d185.Loc == LocRegPair || d185.Loc == LocStackPair || d185.Loc == LocRegTriple || d185.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d186.Loc == LocRegPair || d186.Loc == LocStackPair || d186.Loc == LocRegTriple || d186.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d187 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d187.Loc == LocRegPair || d187.Loc == LocStackPair || d187.Loc == LocRegTriple || d187.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d17.Loc == LocRegPair || d17.Loc == LocStackPair || d17.Loc == LocRegTriple || d17.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d181)
					ctx.SyncDesc(&d182)
					ctx.SyncDesc(&d183)
					ctx.SyncDesc(&d184)
					ctx.SyncDesc(&d185)
					ctx.SyncDesc(&d186)
					ctx.SyncDesc(&d187)
					ctx.SyncDesc(&d17)
					d188 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d181, d182, d183, d184, d185, d186, d187, d17}, 3)
					d188.NoHeapPointer = false
					ctx.BindReg(d188.Reg, &d188)
					ctx.BindReg(d188.Reg2, &d188)
					ctx.BindReg(d188.Reg3, &d188)
					ctx.FreeDesc(&d187)
					ctx.StabilizeDescForControlFlow(&d188)
					ctx.FreeDesc(&d181)
					ctx.FreeDesc(&d182)
					ctx.FreeDesc(&d183)
					ctx.FreeDesc(&d184)
					ctx.FreeDesc(&d185)
					ctx.FreeDesc(&d186)
					ctx.SyncDesc(&d188)
					if d188.Loc == LocReg || d188.Loc == LocFPReg {
						ctx.ProtectReg(d188.Reg)
					} else if d188.Loc == LocRegPair {
						ctx.ProtectReg(d188.Reg)
						ctx.ProtectReg(d188.Reg2)
					}
					d189 = d188
					if d189.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d189)
					if d189.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d189, int32(bbs[9].PhiBase)+int32(0), 2)
					} else if d189.Loc == LocInputPair {
						ctx.EnsureDesc(&d189)
						ctx.EmitStoreScmerToStack(d189, int32(bbs[9].PhiBase)+int32(0))
					} else if d189.Loc == LocRegPair || d189.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d189, int32(bbs[9].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d189)
						ctx.EmitStoreToStack(d189, int32(bbs[9].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
					}
					if d188.Loc == LocReg || d188.Loc == LocFPReg {
						ctx.UnprotectReg(d188.Reg)
					} else if d188.Loc == LocRegPair {
						ctx.UnprotectReg(d188.Reg)
						ctx.UnprotectReg(d188.Reg2)
					}
					return bbs[9].Render()
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
					d190 = args[0]
					d190.ID = 0
					d192 = d190
					ctx.SyncDesc(&d192)
					if d192.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d192.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d192.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d192 = tmpScalar
					}
					d192 = JITPrepareScmerGoArg(ctx, d192)
					if d192.Loc != LocRegPair && d192.Loc != LocStackPair && d192.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d191 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d192}, 2)
					ctx.FreeDesc(&d190)
					ctx.EnsureDesc(&d191)
					if d191.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d191.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d191.Imm)
						ptrWord, _ := d191.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d191.Imm.String())))
						d191 = tmpPair
					} else if d191.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d191.Type, Reg: ctx.AllocRegExcept(d191.Reg), Reg2: ctx.AllocRegExcept(d191.Reg)}
						switch d191.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d191)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d191)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d191)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d191)
						d191 = tmpPair
					}
					if d191.Loc != LocRegPair && d191.Loc != LocStackPair && d191.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (parseDateStringInLoc arg0)")
					}
					if d17.Loc == LocRegPair || d17.Loc == LocStackPair || d17.Loc == LocRegTriple || d17.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d191)
					ctx.SyncDesc(&d17)
					callResults193 := JITEmitGoCallResults(ctx, GoFuncAddr(parseDateStringInLoc), []JITValueDesc{d191, d17}, []uint8{1, 1}, []uint8{0, 0})
					d194 = callResults193[0]
					_ = d194
					d195 = callResults193[1]
					_ = d195
					ctx.StabilizeDescForControlFlow(&d194)
					d196 = d195
					ctx.EnsureDesc(&d196)
					if d196.Loc != LocImm && d196.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d196.Loc == LocImm {
						if d196.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitCmpRegImm32(d196.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap197 := d1
						snap198 := d2
						snap199 := d3
						snap200 := d4
						snap201 := d5
						snap202 := d12
						snap203 := d13
						snap204 := d14
						snap205 := d15
						snap206 := d17
						snap207 := d18
						snap208 := d19
						snap209 := d20
						snap210 := d35
						snap211 := d36
						snap212 := d37
						snap213 := d38
						snap214 := d57
						snap215 := d58
						snap216 := d59
						snap217 := d60
						snap218 := d83
						snap219 := d84
						snap220 := d85
						snap221 := d86
						snap222 := d88
						snap223 := d89
						snap224 := d90
						snap225 := d91
						snap226 := d122
						snap227 := d123
						snap228 := d124
						snap229 := d125
						snap230 := d126
						snap231 := d127
						snap232 := d164
						snap233 := d165
						snap234 := d166
						snap235 := d167
						snap236 := d168
						snap237 := d169
						snap238 := d170
						snap239 := d171
						snap240 := d172
						snap241 := d173
						snap242 := d174
						snap243 := d175
						snap244 := d176
						snap245 := d177
						snap246 := d178
						snap247 := d179
						snap248 := d180
						snap249 := d181
						snap250 := d182
						snap251 := d183
						snap252 := d184
						snap253 := d185
						snap254 := d186
						snap255 := d187
						snap256 := d188
						snap257 := d189
						snap258 := d190
						snap259 := d191
						snap260 := d192
						snap261 := d194
						snap262 := d195
						snap263 := d196
						alloc264 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc264)
						d1 = snap197
						d2 = snap198
						d3 = snap199
						d4 = snap200
						d5 = snap201
						d12 = snap202
						d13 = snap203
						d14 = snap204
						d15 = snap205
						d17 = snap206
						d18 = snap207
						d19 = snap208
						d20 = snap209
						d35 = snap210
						d36 = snap211
						d37 = snap212
						d38 = snap213
						d57 = snap214
						d58 = snap215
						d59 = snap216
						d60 = snap217
						d83 = snap218
						d84 = snap219
						d85 = snap220
						d86 = snap221
						d88 = snap222
						d89 = snap223
						d90 = snap224
						d91 = snap225
						d122 = snap226
						d123 = snap227
						d124 = snap228
						d125 = snap229
						d126 = snap230
						d127 = snap231
						d164 = snap232
						d165 = snap233
						d166 = snap234
						d167 = snap235
						d168 = snap236
						d169 = snap237
						d170 = snap238
						d171 = snap239
						d172 = snap240
						d173 = snap241
						d174 = snap242
						d175 = snap243
						d176 = snap244
						d177 = snap245
						d178 = snap246
						d179 = snap247
						d180 = snap248
						d181 = snap249
						d182 = snap250
						d183 = snap251
						d184 = snap252
						d185 = snap253
						d186 = snap254
						d187 = snap255
						d188 = snap256
						d189 = snap257
						d190 = snap258
						d191 = snap259
						d192 = snap260
						d194 = snap261
						d195 = snap262
						d196 = snap263
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					ctx.FreeDesc(&d195)
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
					d265 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d265)
					if d265.Loc == LocRegPair || d265.Loc == LocStackPair || d265.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d265, &result)
						result.Type = d265.Type
					} else {
						switch d265.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d265)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d265)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d265)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d265, &result)
							result.Type = d265.Type
						}
					}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					if d194.Loc == LocRegPair || d194.Loc == LocStackPair || d194.Loc == LocRegTriple || d194.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d266 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d266.Loc == LocRegPair || d266.Loc == LocStackPair || d266.Loc == LocRegTriple || d266.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d194)
					ctx.SyncDesc(&d266)
					d267 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d194, d266}, 3)
					d267.NoHeapPointer = false
					ctx.BindReg(d267.Reg, &d267)
					ctx.BindReg(d267.Reg2, &d267)
					ctx.BindReg(d267.Reg3, &d267)
					ctx.FreeDesc(&d266)
					ctx.StabilizeDescForControlFlow(&d267)
					ctx.SyncDesc(&d267)
					if d267.Loc == LocReg || d267.Loc == LocFPReg {
						ctx.ProtectReg(d267.Reg)
					} else if d267.Loc == LocRegPair {
						ctx.ProtectReg(d267.Reg)
						ctx.ProtectReg(d267.Reg2)
					}
					d268 = d267
					if d268.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d268)
					if d268.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d268, int32(bbs[9].PhiBase)+int32(0), 2)
					} else if d268.Loc == LocInputPair {
						ctx.EnsureDesc(&d268)
						ctx.EmitStoreScmerToStack(d268, int32(bbs[9].PhiBase)+int32(0))
					} else if d268.Loc == LocRegPair || d268.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d268, int32(bbs[9].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d268)
						ctx.EmitStoreToStack(d268, int32(bbs[9].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
					}
					if d267.Loc == LocReg || d267.Loc == LocFPReg {
						ctx.UnprotectReg(d267.Reg)
					} else if d267.Loc == LocRegPair {
						ctx.UnprotectReg(d267.Reg)
						ctx.UnprotectReg(d267.Reg2)
					}
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
			JITVirtualArgs: true,
			JITInlineCost:  76,
		},
	})

	// FROM_UNIXTIME(unix_ts [, format])
	Declare(&Globalenv, &Declaration{
		Name: "from_unixtime",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			unix := a[0].Int()
			timezone := "UTC"
			if len(a) > 2 {
				timezone = a[2].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			if len(a) > 1 && !a[1].IsNil() {
				// with format string: return string
				t := time.Unix(unix, 0).In(loc)
				return NewString(formatDateMySQL(t, a[1].String()))
			}
			return NewDate(unix)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "converts a unix timestamp to a datetime in the session timezone",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", Label: "unix_ts", Description: "unix timestamp (seconds since epoch)"}, &TypeDescriptor{Kind: "string", Label: "format", Description: "optional MySQL format string", Optional: true}, &TypeDescriptor{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["from_unixtime"]
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
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
				var d78 JITValueDesc
				_ = d78
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
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
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
				var bbs [10]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				bbs[6].PhiBase = int32(phiBase0) + int32(16)
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
					if d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int())}
					} else if d15.Type == tagInt && d15.Loc == LocRegPair {
						ctx.FreeReg(d15.Reg)
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg2}
						ctx.BindReg(d15.Reg2, &d16)
						ctx.BindReg(d15.Reg2, &d16)
					} else if d15.Type == tagInt && d15.Loc == LocReg {
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg}
						ctx.BindReg(d15.Reg, &d16)
						ctx.BindReg(d15.Reg, &d16)
					} else {
						d16 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d15}, 1)
						d16.Type = tagInt
						ctx.BindReg(d16.Reg, &d16)
					}
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d15)
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d17.Reg, 2)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d18)
					}
					ctx.FreeDesc(&d17)
					d19 = d18
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[4].PhiBase)+int32(0))
						return bbs[4].Render()
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitJump(d19.Condition, lbl4)
					ctx.EmitJmp(lbl11)
					ctx.FreeDesc(&d18)
					snap20 := d1
					snap21 := d2
					snap22 := d3
					snap23 := d4
					snap24 := d5
					snap25 := d6
					snap26 := d14
					snap27 := d15
					snap28 := d16
					snap29 := d17
					snap30 := d18
					snap31 := d19
					alloc32 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc32)
					d1 = snap20
					d2 = snap21
					d3 = snap22
					d4 = snap23
					d5 = snap24
					d6 = snap25
					d14 = snap26
					d15 = snap27
					d16 = snap28
					d17 = snap29
					d18 = snap30
					d19 = snap31
					if !bbs[4].Rendered {
						snap33 := d1
						snap34 := d2
						snap35 := d3
						snap36 := d4
						snap37 := d5
						snap38 := d6
						snap39 := d14
						snap40 := d15
						snap41 := d16
						snap42 := d17
						snap43 := d18
						snap44 := d19
						alloc45 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc45)
						d1 = snap33
						d2 = snap34
						d3 = snap35
						d4 = snap36
						d5 = snap37
						d6 = snap38
						d14 = snap39
						d15 = snap40
						d16 = snap41
						d17 = snap42
						d18 = snap43
						d19 = snap44
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d46 = args[2]
					d46.ID = 0
					d48 = d46
					ctx.SyncDesc(&d48)
					if d48.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d48.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d48.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d48 = tmpScalar
					}
					d48 = JITPrepareScmerGoArg(ctx, d48)
					if d48.Loc != LocRegPair && d48.Loc != LocStackPair && d48.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d47 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d48}, 2)
					ctx.StabilizeDescForControlFlow(&d47)
					ctx.FreeDesc(&d46)
					ctx.SyncDesc(&d47)
					if d47.Loc == LocReg || d47.Loc == LocFPReg {
						ctx.ProtectReg(d47.Reg)
					} else if d47.Loc == LocRegPair {
						ctx.ProtectReg(d47.Reg)
						ctx.ProtectReg(d47.Reg2)
					}
					d49 = d47
					if d49.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d49)
					if d49.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d49, int32(bbs[4].PhiBase)+int32(0), 2)
					} else if d49.Loc == LocInputPair {
						ctx.EnsureDesc(&d49)
						ctx.EmitStoreScmerToStack(d49, int32(bbs[4].PhiBase)+int32(0))
					} else if d49.Loc == LocRegPair || d49.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d49, int32(bbs[4].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d49)
						ctx.EmitStoreToStack(d49, int32(bbs[4].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
					}
					if d47.Loc == LocReg || d47.Loc == LocFPReg {
						ctx.UnprotectReg(d47.Reg)
					} else if d47.Loc == LocRegPair {
						ctx.UnprotectReg(d47.Reg)
						ctx.UnprotectReg(d47.Reg2)
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
					callResults50 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d1}, []uint8{1, 2}, []uint8{1, 3})
					d51 = callResults50[0]
					_ = d51
					d52 = callResults50[1]
					_ = d52
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d51)
					ctx.EnsureDesc(&d52)
					if d52.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d52.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d52)
						if d52.Loc != LocReg && d52.Loc != LocRegPair && d52.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d52.Reg)
						ctx.EmitCmpRegImm32(d52.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d53)
					}
					ctx.FreeDesc(&d52)
					d54 = d53
					ctx.EnsureDesc(&d54)
					if d54.Loc != LocImm && d54.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d54.Loc == LocImm {
						if d54.Imm.Bool() {
							return bbs[5].Render()
						}
						ctx.SyncDesc(&d51)
						if d51.Loc == LocReg || d51.Loc == LocFPReg {
							ctx.ProtectReg(d51.Reg)
						} else if d51.Loc == LocRegPair {
							ctx.ProtectReg(d51.Reg)
							ctx.ProtectReg(d51.Reg2)
						}
						d55 = d51
						if d55.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d55)
						ctx.EmitStoreToStack(d55, int32(bbs[6].PhiBase)+int32(0))
						if d51.Loc == LocReg || d51.Loc == LocFPReg {
							ctx.UnprotectReg(d51.Reg)
						} else if d51.Loc == LocRegPair {
							ctx.UnprotectReg(d51.Reg)
							ctx.UnprotectReg(d51.Reg2)
						}
						return bbs[6].Render()
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d54.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					ctx.EmitJmp(lbl12)
					snap56 := d1
					snap57 := d2
					snap58 := d3
					snap59 := d4
					snap60 := d5
					snap61 := d6
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d18
					snap67 := d19
					snap68 := d46
					snap69 := d47
					snap70 := d48
					snap71 := d49
					snap72 := d51
					snap73 := d52
					snap74 := d53
					snap75 := d54
					snap76 := d55
					alloc77 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d51)
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.ProtectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.ProtectReg(d51.Reg)
						ctx.ProtectReg(d51.Reg2)
					}
					d78 = d51
					if d78.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d78)
					ctx.EmitStoreToStack(d78, int32(bbs[6].PhiBase)+int32(0))
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.UnprotectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.UnprotectReg(d51.Reg)
						ctx.UnprotectReg(d51.Reg2)
					}
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc77)
					d1 = snap56
					d2 = snap57
					d3 = snap58
					d4 = snap59
					d5 = snap60
					d6 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d18 = snap66
					d19 = snap67
					d46 = snap68
					d47 = snap69
					d48 = snap70
					d49 = snap71
					d51 = snap72
					d52 = snap73
					d53 = snap74
					d54 = snap75
					d55 = snap76
					if !bbs[6].Rendered {
						snap79 := d1
						snap80 := d2
						snap81 := d3
						snap82 := d4
						snap83 := d5
						snap84 := d6
						snap85 := d14
						snap86 := d15
						snap87 := d16
						snap88 := d17
						snap89 := d18
						snap90 := d19
						snap91 := d46
						snap92 := d47
						snap93 := d48
						snap94 := d49
						snap95 := d51
						snap96 := d52
						snap97 := d53
						snap98 := d54
						snap99 := d55
						snap100 := d78
						alloc101 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc101)
						d1 = snap79
						d2 = snap80
						d3 = snap81
						d4 = snap82
						d5 = snap83
						d6 = snap84
						d14 = snap85
						d15 = snap86
						d16 = snap87
						d17 = snap88
						d18 = snap89
						d19 = snap90
						d46 = snap91
						d47 = snap92
						d48 = snap93
						d49 = snap94
						d51 = snap95
						d52 = snap96
						d53 = snap97
						d54 = snap98
						d55 = snap99
						d78 = snap100
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d53)
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
					d102 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.StabilizeDescForControlFlow(&d102)
					ctx.SyncDesc(&d102)
					if d102.Loc == LocReg || d102.Loc == LocFPReg {
						ctx.ProtectReg(d102.Reg)
					} else if d102.Loc == LocRegPair {
						ctx.ProtectReg(d102.Reg)
						ctx.ProtectReg(d102.Reg2)
					}
					d103 = d102
					if d103.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d103)
					ctx.EmitStoreToStack(d103, int32(bbs[6].PhiBase)+int32(0))
					if d102.Loc == LocReg || d102.Loc == LocFPReg {
						ctx.UnprotectReg(d102.Reg)
					} else if d102.Loc == LocRegPair {
						ctx.UnprotectReg(d102.Reg)
						ctx.UnprotectReg(d102.Reg2)
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
					ctx.StabilizeDescForControlFlow(&d2)
					d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d104)
					if d104.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d104.Imm.Int() > 1)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d104.Reg, 1)
						d105 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreater}
						ctx.BindReg(r2, &d105)
					}
					ctx.FreeDesc(&d104)
					d106 = d105
					ctx.EnsureDesc(&d106)
					if d106.Loc != LocImm && d106.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d106.Loc == LocImm {
						if d106.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d106.Condition, lbl10)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d105)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap107 := d1
						snap108 := d2
						snap109 := d3
						snap110 := d4
						snap111 := d5
						snap112 := d6
						snap113 := d14
						snap114 := d15
						snap115 := d16
						snap116 := d17
						snap117 := d18
						snap118 := d19
						snap119 := d46
						snap120 := d47
						snap121 := d48
						snap122 := d49
						snap123 := d51
						snap124 := d52
						snap125 := d53
						snap126 := d54
						snap127 := d55
						snap128 := d78
						snap129 := d102
						snap130 := d103
						snap131 := d104
						snap132 := d105
						snap133 := d106
						alloc134 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc134)
						d1 = snap107
						d2 = snap108
						d3 = snap109
						d4 = snap110
						d5 = snap111
						d6 = snap112
						d14 = snap113
						d15 = snap114
						d16 = snap115
						d17 = snap116
						d18 = snap117
						d19 = snap118
						d46 = snap119
						d47 = snap120
						d48 = snap121
						d49 = snap122
						d51 = snap123
						d52 = snap124
						d53 = snap125
						d54 = snap126
						d55 = snap127
						d78 = snap128
						d102 = snap129
						d103 = snap130
						d104 = snap131
						d105 = snap132
						d106 = snap133
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d135 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d135.Loc == LocRegPair || d135.Loc == LocStackPair || d135.Loc == LocRegTriple || d135.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d135)
					d136 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d16, d135}, 3)
					d136.NoHeapPointer = false
					ctx.BindReg(d136.Reg, &d136)
					ctx.BindReg(d136.Reg2, &d136)
					ctx.BindReg(d136.Reg3, &d136)
					ctx.FreeDesc(&d135)
					d136 = JITPrepareGoSliceArg(ctx, d136)
					if d136.Loc != LocRegTriple && d136.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d136)
					ctx.SyncDesc(&d2)
					d137 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d136, d2}, 3)
					d137.NoHeapPointer = false
					ctx.BindReg(d137.Reg, &d137)
					ctx.BindReg(d137.Reg2, &d137)
					ctx.BindReg(d137.Reg3, &d137)
					ctx.FreeDesc(&d136)
					d138 = args[1]
					d138.ID = 0
					d140 = d138
					ctx.SyncDesc(&d140)
					if d140.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d140.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d140.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d140 = tmpScalar
					}
					d140 = JITPrepareScmerGoArg(ctx, d140)
					if d140.Loc != LocRegPair && d140.Loc != LocStackPair && d140.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d139 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d140}, 2)
					ctx.FreeDesc(&d138)
					d137 = JITPrepareGoSliceArg(ctx, d137)
					if d137.Loc != LocRegTriple && d137.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (formatDateMySQL arg0)")
					}
					ctx.EnsureDesc(&d139)
					if d139.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d139.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d139.Imm)
						ptrWord, _ := d139.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d139.Imm.String())))
						d139 = tmpPair
					} else if d139.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d139.Type, Reg: ctx.AllocRegExcept(d139.Reg), Reg2: ctx.AllocRegExcept(d139.Reg)}
						switch d139.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d139)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d139)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d139)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d139)
						d139 = tmpPair
					}
					if d139.Loc != LocRegPair && d139.Loc != LocStackPair && d139.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (formatDateMySQL arg1)")
					}
					ctx.SyncDesc(&d137)
					ctx.SyncDesc(&d139)
					d141 = ctx.EmitGoCallScalar(GoFuncAddr(formatDateMySQL), []JITValueDesc{d137, d139}, 2)
					d141.NoHeapPointer = false
					ctx.BindReg(d141.Reg, &d141)
					ctx.BindReg(d141.Reg2, &d141)
					ctx.FreeDesc(&d137)
					ctx.EnsureDesc(&d141)
					d142 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d141}, 2)
					ctx.EmitMovPairToResult(&d142, &result)
					result.Type = tagString
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					d143 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d16}, 2)
					d143.NoHeapPointer = false
					ctx.BindReg(d143.Reg, &d143)
					ctx.BindReg(d143.Reg2, &d143)
					ctx.SyncDesc(&d143)
					if d143.Loc == LocRegPair || d143.Loc == LocStackPair || d143.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d143, &result)
						result.Type = d143.Type
					} else {
						switch d143.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d143)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d143)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d143)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d143, &result)
							result.Type = d143.Type
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d144 = args[1]
					d144.ID = 0
					d146 = d144
					d146.ID = 0
					d145 = ctx.EmitTagEqualsBorrowed(&d146, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d144)
					d147 = d145
					ctx.EnsureDesc(&d147)
					if d147.Loc != LocImm && d147.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d147.Loc == LocImm {
						if d147.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d147.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap148 := d1
						snap149 := d2
						snap150 := d3
						snap151 := d4
						snap152 := d5
						snap153 := d6
						snap154 := d14
						snap155 := d15
						snap156 := d16
						snap157 := d17
						snap158 := d18
						snap159 := d19
						snap160 := d46
						snap161 := d47
						snap162 := d48
						snap163 := d49
						snap164 := d51
						snap165 := d52
						snap166 := d53
						snap167 := d54
						snap168 := d55
						snap169 := d78
						snap170 := d102
						snap171 := d103
						snap172 := d104
						snap173 := d105
						snap174 := d106
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
						snap186 := d146
						snap187 := d147
						alloc188 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc188)
						d1 = snap148
						d2 = snap149
						d3 = snap150
						d4 = snap151
						d5 = snap152
						d6 = snap153
						d14 = snap154
						d15 = snap155
						d16 = snap156
						d17 = snap157
						d18 = snap158
						d19 = snap159
						d46 = snap160
						d47 = snap161
						d48 = snap162
						d49 = snap163
						d51 = snap164
						d52 = snap165
						d53 = snap166
						d54 = snap167
						d55 = snap168
						d78 = snap169
						d102 = snap170
						d103 = snap171
						d104 = snap172
						d105 = snap173
						d106 = snap174
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
						d146 = snap186
						d147 = snap187
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d145)
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
			JITInlineCost:  42,
		},
	})

	// UTC_TIMESTAMP()
	Declare(&Globalenv, &Declaration{
		Name: "utc_timestamp",

		Fn: func(a ...Scmer) Scmer {
			return NewDate(time.Now().UTC().Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current UTC datetime",
			Return: &TypeDescriptor{Kind: "date"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["utc_timestamp"]
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
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d0}, 3)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.BindReg(d1.Reg3, &d1)
				ctx.FreeDesc(&d0)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = true
				ctx.BindReg(d2.Reg, &d2)
				ctx.FreeDesc(&d1)
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d2)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d2}, 2)
				d3.NoHeapPointer = false
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.FreeDesc(&d2)
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
			JITVirtualArgs: true,
			JITInlineCost:  5,
		},
	})

	// UTC_DATE()
	Declare(&Globalenv, &Declaration{
		Name: "utc_date",

		Fn: func(a ...Scmer) Scmer {
			now := time.Now().UTC()
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current UTC date (midnight)",
			Return: &TypeDescriptor{Kind: "date"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["utc_date"]
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
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d0}, 3)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.BindReg(d1.Reg3, &d1)
				ctx.FreeDesc(&d0)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = true
				ctx.BindReg(d2.Reg, &d2)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d1}, 1)
				d3.NoHeapPointer = true
				ctx.BindReg(d3.Reg, &d3)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
				}
				ctx.SyncDesc(&d1)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d1}, 1)
				d4.NoHeapPointer = true
				ctx.BindReg(d4.Reg, &d4)
				ctx.FreeDesc(&d1)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocRegTriple || d4.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d6 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				if d6.Loc == LocRegPair || d6.Loc == LocStackPair || d6.Loc == LocRegTriple || d6.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d7 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocRegTriple || d7.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d8 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d9 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocRegTriple || d9.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d5.Loc == LocRegPair || d5.Loc == LocStackPair || d5.Loc == LocRegTriple || d5.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d2)
				ctx.SyncDesc(&d3)
				ctx.SyncDesc(&d4)
				ctx.SyncDesc(&d6)
				ctx.SyncDesc(&d7)
				ctx.SyncDesc(&d8)
				ctx.SyncDesc(&d9)
				ctx.SyncDesc(&d5)
				d10 := ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d2, d3, d4, d6, d7, d8, d9, d5}, 3)
				d10.NoHeapPointer = false
				ctx.BindReg(d10.Reg, &d10)
				ctx.BindReg(d10.Reg2, &d10)
				ctx.BindReg(d10.Reg3, &d10)
				ctx.FreeDesc(&d6)
				ctx.FreeDesc(&d7)
				ctx.FreeDesc(&d8)
				ctx.FreeDesc(&d9)
				ctx.FreeDesc(&d2)
				ctx.FreeDesc(&d3)
				ctx.FreeDesc(&d4)
				ctx.FreeDesc(&d5)
				d10 = JITPrepareGoSliceArg(ctx, d10)
				if d10.Loc != LocRegTriple && d10.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d10)
				d11 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d10}, 1)
				d11.NoHeapPointer = true
				ctx.BindReg(d11.Reg, &d11)
				ctx.FreeDesc(&d10)
				if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d11)
				d12 := ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d11}, 2)
				d12.NoHeapPointer = false
				ctx.BindReg(d12.Reg, &d12)
				ctx.BindReg(d12.Reg2, &d12)
				ctx.FreeDesc(&d11)
				if d12.Loc == LocImm {
					if result.Loc == LocAny {
						return d12
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
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
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  10,
		},
	})

	// UTC_TIME()
	Declare(&Globalenv, &Declaration{
		Name: "utc_time",

		Fn: func(a ...Scmer) Scmer {
			now := time.Now().UTC()
			// Return as seconds since midnight
			seconds := int64(now.Hour()*3600 + now.Minute()*60 + now.Second())
			return NewDate(seconds)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current UTC time (as a datetime at epoch date)",
			Return: &TypeDescriptor{Kind: "date"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["utc_time"]
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
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d0}, 3)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.BindReg(d1.Reg3, &d1)
				ctx.FreeDesc(&d0)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = true
				ctx.BindReg(d2.Reg, &d2)
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				var d3 JITValueDesc
				if d2.Loc == LocImm {
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() * 3600)}
				} else {
					ctx.EmitIntBinaryImm(JITIntMul, 64, d2.Reg, 3600)
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
					ctx.BindReg(d2.Reg, &d3)
				}
				if d3.Loc == LocReg && d2.Loc == LocReg && d3.Reg == d2.Reg {
					ctx.TransferReg(d2.Reg)
					d2.Loc = LocNone
				}
				ctx.FreeDesc(&d2)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
				}
				ctx.SyncDesc(&d1)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d1}, 1)
				d4.NoHeapPointer = true
				ctx.BindReg(d4.Reg, &d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				var d5 JITValueDesc
				if d4.Loc == LocImm {
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() * 60)}
				} else {
					ctx.EmitIntBinaryImm(JITIntMul, 64, d4.Reg, 60)
					d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
					ctx.BindReg(d4.Reg, &d5)
				}
				if d5.Loc == LocReg && d4.Loc == LocReg && d5.Reg == d4.Reg {
					ctx.TransferReg(d4.Reg)
					d4.Loc = LocNone
				}
				ctx.FreeDesc(&d4)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d5)
				ctx.SyncDesc(&d3)
				ctx.SyncDesc(&d5)
				var d6 JITValueDesc
				if d3.Loc == LocImm && d5.Loc == LocImm {
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d5.Imm.Int())}
				} else if d5.Loc == LocImm && d5.Imm.Int() == 0 {
					ctx.EnsureDesc(&d3)
					r0 := ctx.AllocRegExcept(d3.Reg)
					ctx.EmitMovRegReg(r0, d3.Reg)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
					ctx.BindReg(r0, &d6)
				} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
					ctx.EnsureDesc(&d5)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
					ctx.BindReg(d5.Reg, &d6)
				} else if d3.Loc == LocImm {
					ctx.EnsureDesc(&d5)
					scratch := ctx.AllocRegExcept(d5.Reg)
					ctx.EmitMovRegReg(scratch, d5.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d6)
				} else if d5.Loc == LocImm {
					ctx.EnsureDesc(&d3)
					scratch := ctx.AllocRegExcept(d3.Reg)
					ctx.EmitMovRegReg(scratch, d3.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d5.Imm.Int())
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d6)
				} else {
					ctx.EnsureDesc(&d3)
					ctx.SyncDesc(&d5)
					r1 := ctx.AllocRegExceptOperand(&d5, d3.Reg)
					ctx.EmitMovRegReg(r1, d3.Reg)
					ctx.EmitIntBinary(JITIntAdd, 64, r1, &d5)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
					ctx.BindReg(r1, &d6)
				}
				if d6.Loc == LocReg && d3.Loc == LocReg && d6.Reg == d3.Reg {
					ctx.TransferReg(d3.Reg)
					d3.Loc = LocNone
				}
				ctx.FreeDesc(&d3)
				ctx.FreeDesc(&d5)
				d1 = JITPrepareGoSliceArg(ctx, d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
				}
				ctx.SyncDesc(&d1)
				d7 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d1}, 1)
				d7.NoHeapPointer = true
				ctx.BindReg(d7.Reg, &d7)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d7)
				ctx.SyncDesc(&d6)
				ctx.SyncDesc(&d7)
				var d8 JITValueDesc
				if d6.Loc == LocImm && d7.Loc == LocImm {
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + d7.Imm.Int())}
				} else if d7.Loc == LocImm && d7.Imm.Int() == 0 {
					ctx.EnsureDesc(&d6)
					r2 := ctx.AllocRegExcept(d6.Reg)
					ctx.EmitMovRegReg(r2, d6.Reg)
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d8)
				} else if d6.Loc == LocImm && d6.Imm.Int() == 0 {
					ctx.EnsureDesc(&d7)
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg}
					ctx.BindReg(d7.Reg, &d8)
				} else if d6.Loc == LocImm {
					ctx.EnsureDesc(&d7)
					scratch := ctx.AllocRegExcept(d7.Reg)
					ctx.EmitMovRegReg(scratch, d7.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d6.Imm.Int())
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d8)
				} else if d7.Loc == LocImm {
					ctx.EnsureDesc(&d6)
					scratch := ctx.AllocRegExcept(d6.Reg)
					ctx.EmitMovRegReg(scratch, d6.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d7.Imm.Int())
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d8)
				} else {
					ctx.EnsureDesc(&d6)
					ctx.SyncDesc(&d7)
					r3 := ctx.AllocRegExceptOperand(&d7, d6.Reg)
					ctx.EmitMovRegReg(r3, d6.Reg)
					ctx.EmitIntBinary(JITIntAdd, 64, r3, &d7)
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
					ctx.BindReg(r3, &d8)
				}
				if d8.Loc == LocReg && d6.Loc == LocReg && d8.Reg == d6.Reg {
					ctx.TransferReg(d6.Reg)
					d6.Loc = LocNone
				}
				ctx.FreeDesc(&d6)
				ctx.FreeDesc(&d7)
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d8)
				if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d8)
				d10 := ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d8}, 2)
				d10.NoHeapPointer = false
				ctx.BindReg(d10.Reg, &d10)
				ctx.BindReg(d10.Reg2, &d10)
				ctx.FreeDesc(&d8)
				if d10.Loc == LocImm {
					if result.Loc == LocAny {
						return d10
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d10)
				if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d10, &result)
					result.Type = d10.Type
				} else {
					switch d10.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d10)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d10)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d10)
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
			JITInlineCost:  12,
		},
	})

	// SYSDATE() — re-evaluated on every call (unlike NOW() which is constant per query)
	Declare(&Globalenv, &Declaration{
		Name: "sysdate",

		Fn: func(a ...Scmer) Scmer {
			return NewDate(time.Now().Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current datetime (re-evaluated per call, unlike now())",
			Return: &TypeDescriptor{Kind: "date"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["sysdate"]
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

	// AT_TIME_ZONE(dt, zone): PostgreSQL AT TIME ZONE operator implementation.
	// If dt has zone_id=0 (TIMESTAMP without TZ): interpret as local time in zone → return UTC.
	// If dt has zone_id!=0 (TIMESTAMPTZ): convert UTC moment to local time in zone → return as-is.
	Declare(&Globalenv, &Declaration{
		Name: "at_time_zone",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			toLoc, err := ResolveLocation(a[1].String())
			if err != nil {
				return NewNil()
			}
			var unix int64
			zoneID := 0
			if a[0].GetTag() == tagDate {
				unix = TagDateDecodeUnix(auxVal(a[0].aux))
				zoneID = TagDateDecodeZone(auxVal(a[0].aux))
			} else {
				unix = a[0].Int()
			}
			if zoneID == 0 {
				// TIMESTAMP without TZ: the stored unix is a wall-clock time (UTC-interpreted).
				// Reinterpret it as local time in toLoc and return UTC.
				wall := time.Unix(unix, 0).UTC()
				local := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), 0, toLoc)
				return NewDate(local.UTC().Unix())
			}
			// TIMESTAMPTZ: convert the absolute UTC moment to the target zone's wall clock.
			utcTime := time.Unix(unix, 0).In(toLoc)
			// Return the local wall-clock reading as a "naive" UTC timestamp (zone_id=0)
			naive := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), utcTime.Hour(), utcTime.Minute(), utcTime.Second(), 0, time.UTC)
			return NewDate(naive.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "PostgreSQL AT TIME ZONE operator: converts between timezones",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "dt", Description: "datetime value"}, &TypeDescriptor{Kind: "string", Label: "zone", Description: "target timezone"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["at_time_zone"]
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
				var d22 JITValueDesc
				_ = d22
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
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
				var d159 JITValueDesc
				_ = d159
				var d160 JITValueDesc
				_ = d160
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
				var d170 JITValueDesc
				_ = d170
				var d171 JITValueDesc
				_ = d171
				var d172 JITValueDesc
				_ = d172
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
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
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
				var bbs [11]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
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
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d15 = args[1]
					d15.ID = 0
					d17 = d15
					ctx.SyncDesc(&d17)
					if d17.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d17.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d17.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d17 = tmpScalar
					}
					d17 = JITPrepareScmerGoArg(ctx, d17)
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d17}, 2)
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d16.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d16.Imm)
						ptrWord, _ := d16.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d16.Imm.String())))
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d16.Type, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d16)
					callResults18 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d16}, []uint8{1, 2}, []uint8{1, 3})
					d19 = callResults18[0]
					_ = d19
					d20 = callResults18[1]
					_ = d20
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d20)
						if d20.Loc != LocReg && d20.Loc != LocRegPair && d20.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpRegImm32(d20.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d21)
					}
					ctx.FreeDesc(&d20)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap23 := d1
						snap24 := d2
						snap25 := d3
						snap26 := d4
						snap27 := d5
						snap28 := d6
						snap29 := d14
						snap30 := d15
						snap31 := d16
						snap32 := d17
						snap33 := d19
						snap34 := d20
						snap35 := d21
						snap36 := d22
						alloc37 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc37)
						d1 = snap23
						d2 = snap24
						d3 = snap25
						d4 = snap26
						d5 = snap27
						d6 = snap28
						d14 = snap29
						d15 = snap30
						d16 = snap31
						d17 = snap32
						d19 = snap33
						d20 = snap34
						d21 = snap35
						d22 = snap36
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d21)
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
					ctx.ReclaimUntrackedRegs()
					d38 = args[1]
					d38.ID = 0
					d40 = d38
					d40.ID = 0
					d39 = ctx.EmitTagEqualsBorrowed(&d40, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d38)
					d41 = d39
					ctx.EnsureDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d41.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap42 := d1
						snap43 := d2
						snap44 := d3
						snap45 := d4
						snap46 := d5
						snap47 := d6
						snap48 := d14
						snap49 := d15
						snap50 := d16
						snap51 := d17
						snap52 := d19
						snap53 := d20
						snap54 := d21
						snap55 := d22
						snap56 := d38
						snap57 := d39
						snap58 := d40
						snap59 := d41
						alloc60 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc60)
						d1 = snap42
						d2 = snap43
						d3 = snap44
						d4 = snap45
						d5 = snap46
						d6 = snap47
						d14 = snap48
						d15 = snap49
						d16 = snap50
						d17 = snap51
						d19 = snap52
						d20 = snap53
						d21 = snap54
						d22 = snap55
						d38 = snap56
						d39 = snap57
						d40 = snap58
						d41 = snap59
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d39)
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
					ctx.ReclaimUntrackedRegs()
					d61 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d61)
					if d61.Loc == LocRegPair || d61.Loc == LocStackPair || d61.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d61, &result)
						result.Type = d61.Type
					} else {
						switch d61.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d61)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d61)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d61)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d61, &result)
							result.Type = d61.Type
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
					ctx.ReclaimUntrackedRegs()
					d62 = args[0]
					d62.ID = 0
					d63 = d62
					d63.ID = 0
					d64 = ctx.EmitGetTagDesc(&d63, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d62)
					ctx.EnsureDesc(&d64)
					if d64.Loc == LocImm {
						d65 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d64.Imm.Int()) == uint64(0x10))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d64.Reg, 16)
						d65 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d65)
					}
					ctx.FreeDesc(&d64)
					d66 = d65
					ctx.EnsureDesc(&d66)
					if d66.Loc != LocImm && d66.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d66.Loc == LocImm {
						if d66.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d66.Condition, lbl7)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d65)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap67 := d1
						snap68 := d2
						snap69 := d3
						snap70 := d4
						snap71 := d5
						snap72 := d6
						snap73 := d14
						snap74 := d15
						snap75 := d16
						snap76 := d17
						snap77 := d19
						snap78 := d20
						snap79 := d21
						snap80 := d22
						snap81 := d38
						snap82 := d39
						snap83 := d40
						snap84 := d41
						snap85 := d61
						snap86 := d62
						snap87 := d63
						snap88 := d64
						snap89 := d65
						snap90 := d66
						alloc91 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc91)
						d1 = snap67
						d2 = snap68
						d3 = snap69
						d4 = snap70
						d5 = snap71
						d6 = snap72
						d14 = snap73
						d15 = snap74
						d16 = snap75
						d17 = snap76
						d19 = snap77
						d20 = snap78
						d21 = snap79
						d22 = snap80
						d38 = snap81
						d39 = snap82
						d40 = snap83
						d41 = snap84
						d61 = snap85
						d62 = snap86
						d63 = snap87
						d64 = snap88
						d65 = snap89
						d66 = snap90
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
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d92 = args[0]
					d92.ID = 0
					ctx.EnsureDesc(&d92)
					if d92.Loc == LocImm {
						_, auxWord := d92.Imm.RawWords()
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d92.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r2 := ctx.AllocReg()
						ctx.EmitMovRegReg(r2, d92.Reg2)
						d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d93)
					}
					ctx.EnsureDesc(&d93)
					d94 = d93
					_ = d94
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d94)
					if d94.Loc == LocImm {
						d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d94.Imm.Int()) >> 8))}
					} else {
						r3 := ctx.AllocRegExcept(d94.Reg)
						ctx.EmitMovRegReg(r3, d94.Reg)
						ctx.EmitShrRegImm8(r3, 8)
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d95)
					}
					if d95.Loc == LocReg && d94.Loc == LocReg && d95.Reg == d94.Reg {
						ctx.TransferReg(d94.Reg)
						d94.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d95)
					ctx.FreeDesc(&d93)
					ctx.EnsureDesc(&d95)
					d96 = d95
					_ = d96
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d96)
					if d96.Loc == LocImm {
						d97 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d96.Imm.Int() & 35184372088831)}
					} else {
						r4 := ctx.AllocRegExcept(d96.Reg)
						ctx.EmitMovRegReg(r4, d96.Reg)
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0x1fffffffffff)
						ctx.EmitAndInt64(r4, ctx.ScratchReg)
						d97 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d97)
					}
					if d97.Loc == LocReg && d96.Loc == LocReg && d97.Reg == d96.Reg {
						ctx.TransferReg(d96.Reg)
						d96.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					if d97.Loc == LocImm {
						d98 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d97.Imm.Int()) << 19))}
					} else {
						ctx.EmitShlRegImm8(d97.Reg, 19)
						d98 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d97.Reg}
						ctx.BindReg(d97.Reg, &d98)
					}
					if d98.Loc == LocReg && d97.Loc == LocReg && d98.Reg == d97.Reg {
						ctx.TransferReg(d97.Reg)
						d97.Loc = LocNone
					}
					ctx.FreeDesc(&d97)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d98)
					if d98.Loc == LocImm {
						d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d98.Imm.Int()))))}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegReg(r5, d98.Reg)
						d99 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d99)
					}
					ctx.FreeDesc(&d98)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d99)
					if d99.Loc == LocImm {
						d100 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d99.Imm.Int()) >> 19))}
					} else {
						ctx.EmitShrRegImm8(d99.Reg, 19)
						d100 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d99.Reg}
						ctx.BindReg(d99.Reg, &d100)
					}
					if d100.Loc == LocReg && d99.Loc == LocReg && d100.Reg == d99.Reg {
						ctx.TransferReg(d99.Reg)
						d99.Loc = LocNone
					}
					ctx.FreeDesc(&d99)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.StabilizeDescForControlFlow(&d100)
					ctx.FreeDesc(&d95)
					d101 = args[0]
					d101.ID = 0
					ctx.EnsureDesc(&d101)
					if d101.Loc == LocImm {
						_, auxWord := d101.Imm.RawWords()
						d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d101.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r6 := ctx.AllocReg()
						ctx.EmitMovRegReg(r6, d101.Reg2)
						d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d102)
					}
					ctx.EnsureDesc(&d102)
					d103 = d102
					_ = d103
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm {
						d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d103.Imm.Int()) >> 8))}
					} else {
						r7 := ctx.AllocRegExcept(d103.Reg)
						ctx.EmitMovRegReg(r7, d103.Reg)
						ctx.EmitShrRegImm8(r7, 8)
						d104 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d104)
					}
					if d104.Loc == LocReg && d103.Loc == LocReg && d104.Reg == d103.Reg {
						ctx.TransferReg(d103.Reg)
						d103.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					ctx.FreeDesc(&d102)
					ctx.EnsureDesc(&d104)
					d105 = d104
					_ = d105
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d105.Imm.Int()) >> 45))}
					} else {
						r8 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r8, d105.Reg)
						ctx.EmitShrRegImm8(r8, 45)
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d106)
					}
					if d106.Loc == LocReg && d105.Loc == LocReg && d106.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					if d106.Loc == LocImm {
						d107 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d106.Imm.Int() & 2047)}
					} else {
						ctx.EmitAndRegImm32(d106.Reg, int32(2047))
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d106.Reg}
						ctx.BindReg(d106.Reg, &d107)
					}
					if d107.Loc == LocReg && d106.Loc == LocReg && d107.Reg == d106.Reg {
						ctx.TransferReg(d106.Reg)
						d106.Loc = LocNone
					}
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d107)
					ctx.EnsureDesc(&d107)
					if d107.Loc == LocImm {
						d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d107.Imm.Int()))))}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitMovRegReg(r9, d107.Reg)
						d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d108)
					}
					ctx.FreeDesc(&d107)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d108)
					ctx.StabilizeDescForControlFlow(&d108)
					ctx.FreeDesc(&d104)
					ctx.SyncDesc(&d100)
					if d100.Loc == LocReg || d100.Loc == LocFPReg {
						ctx.ProtectReg(d100.Reg)
					} else if d100.Loc == LocRegPair {
						ctx.ProtectReg(d100.Reg)
						ctx.ProtectReg(d100.Reg2)
					}
					ctx.SyncDesc(&d108)
					if d108.Loc == LocReg || d108.Loc == LocFPReg {
						ctx.ProtectReg(d108.Reg)
					} else if d108.Loc == LocRegPair {
						ctx.ProtectReg(d108.Reg)
						ctx.ProtectReg(d108.Reg2)
					}
					d109 = d100
					if d109.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d109)
					ctx.EmitStoreToStack(d109, int32(bbs[7].PhiBase)+int32(0))
					d110 = d108
					if d110.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d110)
					ctx.EmitStoreToStack(d110, int32(bbs[7].PhiBase)+int32(16))
					if d100.Loc == LocReg || d100.Loc == LocFPReg {
						ctx.UnprotectReg(d100.Reg)
					} else if d100.Loc == LocRegPair {
						ctx.UnprotectReg(d100.Reg)
						ctx.UnprotectReg(d100.Reg2)
					}
					if d108.Loc == LocReg || d108.Loc == LocFPReg {
						ctx.UnprotectReg(d108.Reg)
					} else if d108.Loc == LocRegPair {
						ctx.UnprotectReg(d108.Reg)
						ctx.UnprotectReg(d108.Reg2)
					}
					return bbs[7].Render()
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
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 0)}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						d111 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondEqual}
						ctx.BindReg(r10, &d111)
					}
					ctx.FreeDesc(&d2)
					d112 = d111
					ctx.EnsureDesc(&d112)
					if d112.Loc != LocImm && d112.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d112.Loc == LocImm {
						if d112.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[10].Render()
					}
					ctx.EmitJump(d112.Condition, lbl10)
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
					}
					ctx.FreeDesc(&d111)
					ctx.FlushRegisterMoves()
					if !bbs[10].Rendered {
						snap113 := d1
						snap114 := d2
						snap115 := d3
						snap116 := d4
						snap117 := d5
						snap118 := d6
						snap119 := d14
						snap120 := d15
						snap121 := d16
						snap122 := d17
						snap123 := d19
						snap124 := d20
						snap125 := d21
						snap126 := d22
						snap127 := d38
						snap128 := d39
						snap129 := d40
						snap130 := d41
						snap131 := d61
						snap132 := d62
						snap133 := d63
						snap134 := d64
						snap135 := d65
						snap136 := d66
						snap137 := d92
						snap138 := d93
						snap139 := d94
						snap140 := d95
						snap141 := d96
						snap142 := d97
						snap143 := d98
						snap144 := d99
						snap145 := d100
						snap146 := d101
						snap147 := d102
						snap148 := d103
						snap149 := d104
						snap150 := d105
						snap151 := d106
						snap152 := d107
						snap153 := d108
						snap154 := d109
						snap155 := d110
						snap156 := d111
						snap157 := d112
						alloc158 := ctx.SnapshotAllocState()
						bbs[10].Render()
						ctx.RestoreAllocState(alloc158)
						d1 = snap113
						d2 = snap114
						d3 = snap115
						d4 = snap116
						d5 = snap117
						d6 = snap118
						d14 = snap119
						d15 = snap120
						d16 = snap121
						d17 = snap122
						d19 = snap123
						d20 = snap124
						d21 = snap125
						d22 = snap126
						d38 = snap127
						d39 = snap128
						d40 = snap129
						d41 = snap130
						d61 = snap131
						d62 = snap132
						d63 = snap133
						d64 = snap134
						d65 = snap135
						d66 = snap136
						d92 = snap137
						d93 = snap138
						d94 = snap139
						d95 = snap140
						d96 = snap141
						d97 = snap142
						d98 = snap143
						d99 = snap144
						d100 = snap145
						d101 = snap146
						d102 = snap147
						d103 = snap148
						d104 = snap149
						d105 = snap150
						d106 = snap151
						d107 = snap152
						d108 = snap153
						d109 = snap154
						d110 = snap155
						d111 = snap156
						d112 = snap157
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d159 = args[0]
					d159.ID = 0
					if d159.Loc == LocImm {
						d160 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d159.Imm.Int())}
					} else if d159.Type == tagInt && d159.Loc == LocRegPair {
						ctx.FreeReg(d159.Reg)
						d160 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d159.Reg2}
						ctx.BindReg(d159.Reg2, &d160)
						ctx.BindReg(d159.Reg2, &d160)
					} else if d159.Type == tagInt && d159.Loc == LocReg {
						d160 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d159.Reg}
						ctx.BindReg(d159.Reg, &d160)
						ctx.BindReg(d159.Reg, &d160)
					} else {
						d160 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d159}, 1)
						d160.Type = tagInt
						ctx.BindReg(d160.Reg, &d160)
					}
					ctx.StabilizeDescForControlFlow(&d160)
					ctx.FreeDesc(&d159)
					ctx.SyncDesc(&d160)
					if d160.Loc == LocReg || d160.Loc == LocFPReg {
						ctx.ProtectReg(d160.Reg)
					} else if d160.Loc == LocRegPair {
						ctx.ProtectReg(d160.Reg)
						ctx.ProtectReg(d160.Reg2)
					}
					d161 = d160
					if d161.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d161)
					ctx.EmitStoreToStack(d161, int32(bbs[7].PhiBase)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
					if d160.Loc == LocReg || d160.Loc == LocFPReg {
						ctx.UnprotectReg(d160.Reg)
					} else if d160.Loc == LocRegPair {
						ctx.UnprotectReg(d160.Reg)
						ctx.UnprotectReg(d160.Reg2)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d162 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d162.Loc == LocRegPair || d162.Loc == LocStackPair || d162.Loc == LocRegTriple || d162.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d162)
					d163 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d1, d162}, 3)
					d163.NoHeapPointer = false
					ctx.BindReg(d163.Reg, &d163)
					ctx.BindReg(d163.Reg2, &d163)
					ctx.BindReg(d163.Reg3, &d163)
					ctx.FreeDesc(&d162)
					d163 = JITPrepareGoSliceArg(ctx, d163)
					if d163.Loc != LocRegTriple && d163.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d163)
					d164 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d163}, 3)
					d164.NoHeapPointer = false
					ctx.BindReg(d164.Reg, &d164)
					ctx.BindReg(d164.Reg2, &d164)
					ctx.BindReg(d164.Reg3, &d164)
					ctx.FreeDesc(&d163)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d164)
					d165 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d164}, 1)
					d165.NoHeapPointer = true
					ctx.BindReg(d165.Reg, &d165)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d164)
					d166 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d164}, 1)
					d166.NoHeapPointer = true
					ctx.BindReg(d166.Reg, &d166)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d164)
					d167 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d164}, 1)
					d167.NoHeapPointer = true
					ctx.BindReg(d167.Reg, &d167)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d164)
					d168 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d164}, 1)
					d168.NoHeapPointer = true
					ctx.BindReg(d168.Reg, &d168)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d164)
					d169 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d164}, 1)
					d169.NoHeapPointer = true
					ctx.BindReg(d169.Reg, &d169)
					d164 = JITPrepareGoSliceArg(ctx, d164)
					if d164.Loc != LocRegTriple && d164.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d164)
					d170 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d164}, 1)
					d170.NoHeapPointer = true
					ctx.BindReg(d170.Reg, &d170)
					ctx.FreeDesc(&d164)
					if d165.Loc == LocRegPair || d165.Loc == LocStackPair || d165.Loc == LocRegTriple || d165.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d166.Loc == LocRegPair || d166.Loc == LocStackPair || d166.Loc == LocRegTriple || d166.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d167.Loc == LocRegPair || d167.Loc == LocStackPair || d167.Loc == LocRegTriple || d167.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d168.Loc == LocRegPair || d168.Loc == LocStackPair || d168.Loc == LocRegTriple || d168.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d169.Loc == LocRegPair || d169.Loc == LocStackPair || d169.Loc == LocRegTriple || d169.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d170.Loc == LocRegPair || d170.Loc == LocStackPair || d170.Loc == LocRegTriple || d170.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d171 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d171.Loc == LocRegPair || d171.Loc == LocStackPair || d171.Loc == LocRegTriple || d171.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d19.Loc == LocRegPair || d19.Loc == LocStackPair || d19.Loc == LocRegTriple || d19.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d165)
					ctx.SyncDesc(&d166)
					ctx.SyncDesc(&d167)
					ctx.SyncDesc(&d168)
					ctx.SyncDesc(&d169)
					ctx.SyncDesc(&d170)
					ctx.SyncDesc(&d171)
					ctx.SyncDesc(&d19)
					d172 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d165, d166, d167, d168, d169, d170, d171, d19}, 3)
					d172.NoHeapPointer = false
					ctx.BindReg(d172.Reg, &d172)
					ctx.BindReg(d172.Reg2, &d172)
					ctx.BindReg(d172.Reg3, &d172)
					ctx.FreeDesc(&d171)
					ctx.FreeDesc(&d165)
					ctx.FreeDesc(&d166)
					ctx.FreeDesc(&d167)
					ctx.FreeDesc(&d168)
					ctx.FreeDesc(&d169)
					ctx.FreeDesc(&d170)
					d172 = JITPrepareGoSliceArg(ctx, d172)
					if d172.Loc != LocRegTriple && d172.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d172)
					d173 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d172}, 3)
					d173.NoHeapPointer = false
					ctx.BindReg(d173.Reg, &d173)
					ctx.BindReg(d173.Reg2, &d173)
					ctx.BindReg(d173.Reg3, &d173)
					ctx.FreeDesc(&d172)
					d173 = JITPrepareGoSliceArg(ctx, d173)
					if d173.Loc != LocRegTriple && d173.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d173)
					d174 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d173}, 1)
					d174.NoHeapPointer = true
					ctx.BindReg(d174.Reg, &d174)
					ctx.FreeDesc(&d173)
					if d174.Loc == LocRegPair || d174.Loc == LocStackPair || d174.Loc == LocRegTriple || d174.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d174)
					d175 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d174}, 2)
					d175.NoHeapPointer = false
					ctx.BindReg(d175.Reg, &d175)
					ctx.BindReg(d175.Reg2, &d175)
					ctx.FreeDesc(&d174)
					ctx.SyncDesc(&d175)
					if d175.Loc == LocRegPair || d175.Loc == LocStackPair || d175.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d175, &result)
						result.Type = d175.Type
					} else {
						switch d175.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d175)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d175)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d175)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d175, &result)
							result.Type = d175.Type
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d176 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d176.Loc == LocRegPair || d176.Loc == LocStackPair || d176.Loc == LocRegTriple || d176.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d176)
					d177 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d1, d176}, 3)
					d177.NoHeapPointer = false
					ctx.BindReg(d177.Reg, &d177)
					ctx.BindReg(d177.Reg2, &d177)
					ctx.BindReg(d177.Reg3, &d177)
					ctx.FreeDesc(&d176)
					d177 = JITPrepareGoSliceArg(ctx, d177)
					if d177.Loc != LocRegTriple && d177.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					if d19.Loc == LocRegPair || d19.Loc == LocStackPair || d19.Loc == LocRegTriple || d19.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d177)
					ctx.SyncDesc(&d19)
					d178 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d177, d19}, 3)
					d178.NoHeapPointer = false
					ctx.BindReg(d178.Reg, &d178)
					ctx.BindReg(d178.Reg2, &d178)
					ctx.BindReg(d178.Reg3, &d178)
					ctx.FreeDesc(&d177)
					d178 = JITPrepareGoSliceArg(ctx, d178)
					if d178.Loc != LocRegTriple && d178.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d178)
					d179 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d178}, 1)
					d179.NoHeapPointer = true
					ctx.BindReg(d179.Reg, &d179)
					d178 = JITPrepareGoSliceArg(ctx, d178)
					if d178.Loc != LocRegTriple && d178.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d178)
					d180 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d178}, 1)
					d180.NoHeapPointer = true
					ctx.BindReg(d180.Reg, &d180)
					d178 = JITPrepareGoSliceArg(ctx, d178)
					if d178.Loc != LocRegTriple && d178.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d178)
					d181 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d178}, 1)
					d181.NoHeapPointer = true
					ctx.BindReg(d181.Reg, &d181)
					d178 = JITPrepareGoSliceArg(ctx, d178)
					if d178.Loc != LocRegTriple && d178.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d178)
					d182 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d178}, 1)
					d182.NoHeapPointer = true
					ctx.BindReg(d182.Reg, &d182)
					d178 = JITPrepareGoSliceArg(ctx, d178)
					if d178.Loc != LocRegTriple && d178.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d178)
					d183 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d178}, 1)
					d183.NoHeapPointer = true
					ctx.BindReg(d183.Reg, &d183)
					d178 = JITPrepareGoSliceArg(ctx, d178)
					if d178.Loc != LocRegTriple && d178.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d178)
					d184 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d178}, 1)
					d184.NoHeapPointer = true
					ctx.BindReg(d184.Reg, &d184)
					ctx.FreeDesc(&d178)
					d185 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					if d179.Loc == LocRegPair || d179.Loc == LocStackPair || d179.Loc == LocRegTriple || d179.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d180.Loc == LocRegPair || d180.Loc == LocStackPair || d180.Loc == LocRegTriple || d180.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d181.Loc == LocRegPair || d181.Loc == LocStackPair || d181.Loc == LocRegTriple || d181.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d182.Loc == LocRegPair || d182.Loc == LocStackPair || d182.Loc == LocRegTriple || d182.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d183.Loc == LocRegPair || d183.Loc == LocStackPair || d183.Loc == LocRegTriple || d183.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d184.Loc == LocRegPair || d184.Loc == LocStackPair || d184.Loc == LocRegTriple || d184.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d186 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d186.Loc == LocRegPair || d186.Loc == LocStackPair || d186.Loc == LocRegTriple || d186.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d185.Loc == LocRegPair || d185.Loc == LocStackPair || d185.Loc == LocRegTriple || d185.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d179)
					ctx.SyncDesc(&d180)
					ctx.SyncDesc(&d181)
					ctx.SyncDesc(&d182)
					ctx.SyncDesc(&d183)
					ctx.SyncDesc(&d184)
					ctx.SyncDesc(&d186)
					ctx.SyncDesc(&d185)
					d187 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d179, d180, d181, d182, d183, d184, d186, d185}, 3)
					d187.NoHeapPointer = false
					ctx.BindReg(d187.Reg, &d187)
					ctx.BindReg(d187.Reg2, &d187)
					ctx.BindReg(d187.Reg3, &d187)
					ctx.FreeDesc(&d186)
					ctx.FreeDesc(&d179)
					ctx.FreeDesc(&d180)
					ctx.FreeDesc(&d181)
					ctx.FreeDesc(&d182)
					ctx.FreeDesc(&d183)
					ctx.FreeDesc(&d184)
					ctx.FreeDesc(&d185)
					d187 = JITPrepareGoSliceArg(ctx, d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d187)
					d188 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d187}, 1)
					d188.NoHeapPointer = true
					ctx.BindReg(d188.Reg, &d188)
					ctx.FreeDesc(&d187)
					if d188.Loc == LocRegPair || d188.Loc == LocStackPair || d188.Loc == LocRegTriple || d188.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d188)
					d189 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d188}, 2)
					d189.NoHeapPointer = false
					ctx.BindReg(d189.Reg, &d189)
					ctx.BindReg(d189.Reg2, &d189)
					ctx.FreeDesc(&d188)
					ctx.SyncDesc(&d189)
					if d189.Loc == LocRegPair || d189.Loc == LocStackPair || d189.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d189, &result)
						result.Type = d189.Type
					} else {
						switch d189.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d189)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d189)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d189)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d189, &result)
							result.Type = d189.Type
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
			JITInlineCost:  83,
		},
	})

	// TIMESTAMPDIFF(unit, dt1, dt2)
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
			unit := strings.ToUpper(String(a[0]))
			diff := t2.Sub(t1)
			switch unit {
			case "SECOND":
				return NewInt(int64(diff.Seconds()))
			case "MINUTE":
				return NewInt(int64(diff.Minutes()))
			case "HOUR":
				return NewInt(int64(diff.Hours()))
			case "DAY":
				return NewInt(int64(diff.Hours() / 24))
			case "WEEK":
				return NewInt(int64(diff.Hours() / (24 * 7)))
			case "MONTH":
				y1, m1, _ := t1.Date()
				y2, m2, _ := t2.Date()
				return NewInt(int64((y2-y1)*12 + int(m2-m1)))
			case "YEAR":
				y1, _, _ := t1.Date()
				y2, _, _ := t2.Date()
				return NewInt(int64(y2 - y1))
			default:
				return NewNil() // unknown unit → NULL (MySQL compatible)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the difference between two datetimes in the given unit",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "unit", Description: "SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, YEAR"}, &TypeDescriptor{Kind: "any", Label: "dt1", Description: "first datetime"}, &TypeDescriptor{Kind: "any", Label: "dt2", Description: "second datetime"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["timestampdiff"]
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
				var d90 JITValueDesc
				_ = d90
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
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
				var d170 JITValueDesc
				_ = d170
				var d171 JITValueDesc
				_ = d171
				var d172 JITValueDesc
				_ = d172
				var d173 JITValueDesc
				_ = d173
				var d219 JITValueDesc
				_ = d219
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d222 JITValueDesc
				_ = d222
				var d223 JITValueDesc
				_ = d223
				var d224 JITValueDesc
				_ = d224
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d280 JITValueDesc
				_ = d280
				var d281 JITValueDesc
				_ = d281
				var d282 JITValueDesc
				_ = d282
				var d283 JITValueDesc
				_ = d283
				var d284 JITValueDesc
				_ = d284
				var d285 JITValueDesc
				_ = d285
				var d286 JITValueDesc
				_ = d286
				var d287 JITValueDesc
				_ = d287
				var d350 JITValueDesc
				_ = d350
				var d351 JITValueDesc
				_ = d351
				var d352 JITValueDesc
				_ = d352
				var d354 JITValueDesc
				_ = d354
				var d355 JITValueDesc
				_ = d355
				var d356 JITValueDesc
				_ = d356
				var d357 JITValueDesc
				_ = d357
				var d358 JITValueDesc
				_ = d358
				var d359 JITValueDesc
				_ = d359
				var d360 JITValueDesc
				_ = d360
				var d361 JITValueDesc
				_ = d361
				var d362 JITValueDesc
				_ = d362
				var d363 JITValueDesc
				_ = d363
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d445 JITValueDesc
				_ = d445
				var d446 JITValueDesc
				_ = d446
				var d447 JITValueDesc
				_ = d447
				var d449 JITValueDesc
				_ = d449
				var d450 JITValueDesc
				_ = d450
				var d451 JITValueDesc
				_ = d451
				var d452 JITValueDesc
				_ = d452
				var d453 JITValueDesc
				_ = d453
				var d454 JITValueDesc
				_ = d454
				var d455 JITValueDesc
				_ = d455
				var d456 JITValueDesc
				_ = d456
				var d457 JITValueDesc
				_ = d457
				var d458 JITValueDesc
				_ = d458
				var d549 JITValueDesc
				_ = d549
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
				var bbs [21]BBDescriptor
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
					d0 = args[1]
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
					d10 = JITPrepareScmerGoArg(ctx, d10)
					ctx.SyncDesc(&d10)
					callResults11 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d10}, []uint8{3, 1}, []uint8{4, 0})
					d12 = callResults11[0]
					_ = d12
					d13 = callResults11[1]
					_ = d13
					ctx.FreeDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d12)
					d14 = args[2]
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
					d32 = args[2]
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
					d54 = args[0]
					d54.ID = 0
					d56 = d54
					ctx.SyncDesc(&d56)
					if d56.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d56.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d56.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d56 = tmpScalar
					}
					d56 = JITPrepareScmerGoArg(ctx, d56)
					if d56.Loc != LocRegPair && d56.Loc != LocStackPair && d56.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d55 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d56}, 2)
					ctx.FreeDesc(&d54)
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d55.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d55.Imm)
						ptrWord, _ := d55.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d55.Imm.String())))
						d55 = tmpPair
					} else if d55.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d55.Type, Reg: ctx.AllocRegExcept(d55.Reg), Reg2: ctx.AllocRegExcept(d55.Reg)}
						switch d55.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d55)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d55)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d55)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d55)
						d55 = tmpPair
					}
					if d55.Loc != LocRegPair && d55.Loc != LocStackPair && d55.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d55)
					d57 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d55}, 2)
					d57.NoHeapPointer = false
					ctx.BindReg(d57.Reg, &d57)
					ctx.BindReg(d57.Reg2, &d57)
					ctx.StabilizeDescForControlFlow(&d57)
					d16 = JITPrepareGoSliceArg(ctx, d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d12)
					d58 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d16, d12}, 1)
					d58.NoHeapPointer = true
					ctx.BindReg(d58.Reg, &d58)
					ctx.StabilizeDescForControlFlow(&d58)
					ctx.EnsureDesc(&d57)
					d59 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					if d59.Loc == LocImm {
						ctx.TrackImm(d59.Imm)
						ptrWord, _ := d59.Imm.RawWords()
						d60 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d60.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d60.Reg2, uint64(len(d59.Imm.String())))
						ctx.BindReg(d60.Reg, &d60)
						ctx.BindReg(d60.Reg2, &d60)
					} else {
						d60 = d59
					}
					d61 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d60}, 1)
					ctx.EmitAndRegImm32(d61.Reg, 1)
					d61.Type = tagBool
					ctx.BindReg(d61.Reg, &d61)
					d62 = d61
					ctx.EnsureDesc(&d62)
					if d62.Loc != LocImm && d62.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d62.Loc == LocImm {
						if d62.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitCmpRegImm32(d62.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap63 := d0
						snap64 := d1
						snap65 := d2
						snap66 := d3
						snap67 := d9
						snap68 := d10
						snap69 := d12
						snap70 := d13
						snap71 := d14
						snap72 := d16
						snap73 := d17
						snap74 := d18
						snap75 := d32
						snap76 := d33
						snap77 := d34
						snap78 := d35
						snap79 := d53
						snap80 := d54
						snap81 := d55
						snap82 := d56
						snap83 := d57
						snap84 := d58
						snap85 := d59
						snap86 := d60
						snap87 := d61
						snap88 := d62
						alloc89 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc89)
						d0 = snap63
						d1 = snap64
						d2 = snap65
						d3 = snap66
						d9 = snap67
						d10 = snap68
						d12 = snap69
						d13 = snap70
						d14 = snap71
						d16 = snap72
						d17 = snap73
						d18 = snap74
						d32 = snap75
						d33 = snap76
						d34 = snap77
						d35 = snap78
						d53 = snap79
						d54 = snap80
						d55 = snap81
						d56 = snap82
						d57 = snap83
						d58 = snap84
						d59 = snap85
						d60 = snap86
						d61 = snap87
						d62 = snap88
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					ctx.FreeDesc(&d61)
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
					d90 = d17
					ctx.EnsureDesc(&d90)
					if d90.Loc != LocImm && d90.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d90.Loc == LocImm {
						if d90.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d90.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap91 := d0
						snap92 := d1
						snap93 := d2
						snap94 := d3
						snap95 := d9
						snap96 := d10
						snap97 := d12
						snap98 := d13
						snap99 := d14
						snap100 := d16
						snap101 := d17
						snap102 := d18
						snap103 := d32
						snap104 := d33
						snap105 := d34
						snap106 := d35
						snap107 := d53
						snap108 := d54
						snap109 := d55
						snap110 := d56
						snap111 := d57
						snap112 := d58
						snap113 := d59
						snap114 := d60
						snap115 := d61
						snap116 := d62
						snap117 := d90
						alloc118 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc118)
						d0 = snap91
						d1 = snap92
						d2 = snap93
						d3 = snap94
						d9 = snap95
						d10 = snap96
						d12 = snap97
						d13 = snap98
						d14 = snap99
						d16 = snap100
						d17 = snap101
						d18 = snap102
						d32 = snap103
						d33 = snap104
						d34 = snap105
						d35 = snap106
						d53 = snap107
						d54 = snap108
						d55 = snap109
						d56 = snap110
						d57 = snap111
						d58 = snap112
						d59 = snap113
						d60 = snap114
						d61 = snap115
						d62 = snap116
						d90 = snap117
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
					ctx.ReclaimUntrackedRegs()
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocRegTriple || d58.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d58)
					d119 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Seconds), []JITValueDesc{d58}, 1)
					d119.NoHeapPointer = true
					ctx.BindReg(d119.Reg, &d119)
					ctx.EnsureDesc(&d119)
					ctx.EnsureDesc(&d119)
					if d119.Loc == LocImm {
						d120 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d119.Imm.Float()))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r0, d119.Reg)
						d120 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d120)
					}
					ctx.FreeDesc(&d119)
					ctx.EnsureDesc(&d120)
					if d120.Loc == LocImm {
						ctx.EmitMakeInt(result, d120)
					} else {
						ctx.EmitMovToReg(result.Reg2, d120)
						d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d121)
						if d120.Loc == LocReg && d120.Reg != result.Reg2 {
							ctx.FreeReg(d120.Reg)
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
					ctx.ReclaimUntrackedRegs()
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocRegTriple || d58.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d58)
					d122 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Minutes), []JITValueDesc{d58}, 1)
					d122.NoHeapPointer = true
					ctx.BindReg(d122.Reg, &d122)
					ctx.EnsureDesc(&d122)
					ctx.EnsureDesc(&d122)
					if d122.Loc == LocImm {
						d123 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d122.Imm.Float()))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r1, d122.Reg)
						d123 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d123)
					}
					ctx.FreeDesc(&d122)
					ctx.EnsureDesc(&d123)
					if d123.Loc == LocImm {
						ctx.EmitMakeInt(result, d123)
					} else {
						ctx.EmitMovToReg(result.Reg2, d123)
						d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d124)
						if d123.Loc == LocReg && d123.Reg != result.Reg2 {
							ctx.FreeReg(d123.Reg)
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
					ctx.EnsureDesc(&d57)
					d125 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					if d125.Loc == LocImm {
						ctx.TrackImm(d125.Imm)
						ptrWord, _ := d125.Imm.RawWords()
						d126 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d126.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d126.Reg2, uint64(len(d125.Imm.String())))
						ctx.BindReg(d126.Reg, &d126)
						ctx.BindReg(d126.Reg2, &d126)
					} else {
						d126 = d125
					}
					d127 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d126}, 1)
					ctx.EmitAndRegImm32(d127.Reg, 1)
					d127.Type = tagBool
					ctx.BindReg(d127.Reg, &d127)
					d128 = d127
					ctx.EnsureDesc(&d128)
					if d128.Loc != LocImm && d128.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d128.Loc == LocImm {
						if d128.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitCmpRegImm32(d128.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap129 := d0
						snap130 := d1
						snap131 := d2
						snap132 := d3
						snap133 := d9
						snap134 := d10
						snap135 := d12
						snap136 := d13
						snap137 := d14
						snap138 := d16
						snap139 := d17
						snap140 := d18
						snap141 := d32
						snap142 := d33
						snap143 := d34
						snap144 := d35
						snap145 := d53
						snap146 := d54
						snap147 := d55
						snap148 := d56
						snap149 := d57
						snap150 := d58
						snap151 := d59
						snap152 := d60
						snap153 := d61
						snap154 := d62
						snap155 := d90
						snap156 := d119
						snap157 := d120
						snap158 := d121
						snap159 := d122
						snap160 := d123
						snap161 := d124
						snap162 := d125
						snap163 := d126
						snap164 := d127
						snap165 := d128
						alloc166 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc166)
						d0 = snap129
						d1 = snap130
						d2 = snap131
						d3 = snap132
						d9 = snap133
						d10 = snap134
						d12 = snap135
						d13 = snap136
						d14 = snap137
						d16 = snap138
						d17 = snap139
						d18 = snap140
						d32 = snap141
						d33 = snap142
						d34 = snap143
						d35 = snap144
						d53 = snap145
						d54 = snap146
						d55 = snap147
						d56 = snap148
						d57 = snap149
						d58 = snap150
						d59 = snap151
						d60 = snap152
						d61 = snap153
						d62 = snap154
						d90 = snap155
						d119 = snap156
						d120 = snap157
						d121 = snap158
						d122 = snap159
						d123 = snap160
						d124 = snap161
						d125 = snap162
						d126 = snap163
						d127 = snap164
						d128 = snap165
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d127)
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
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocRegTriple || d58.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d58)
					d167 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d58}, 1)
					d167.NoHeapPointer = true
					ctx.BindReg(d167.Reg, &d167)
					ctx.EnsureDesc(&d167)
					ctx.EnsureDesc(&d167)
					if d167.Loc == LocImm {
						d168 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d167.Imm.Float()))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r2, d167.Reg)
						d168 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d168)
					}
					ctx.FreeDesc(&d167)
					ctx.EnsureDesc(&d168)
					if d168.Loc == LocImm {
						ctx.EmitMakeInt(result, d168)
					} else {
						ctx.EmitMovToReg(result.Reg2, d168)
						d169 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d169)
						if d168.Loc == LocReg && d168.Reg != result.Reg2 {
							ctx.FreeReg(d168.Reg)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d57)
					d170 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					if d170.Loc == LocImm {
						ctx.TrackImm(d170.Imm)
						ptrWord, _ := d170.Imm.RawWords()
						d171 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d171.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d171.Reg2, uint64(len(d170.Imm.String())))
						ctx.BindReg(d171.Reg, &d171)
						ctx.BindReg(d171.Reg2, &d171)
					} else {
						d171 = d170
					}
					d172 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d171}, 1)
					ctx.EmitAndRegImm32(d172.Reg, 1)
					d172.Type = tagBool
					ctx.BindReg(d172.Reg, &d172)
					d173 = d172
					ctx.EnsureDesc(&d173)
					if d173.Loc != LocImm && d173.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d173.Loc == LocImm {
						if d173.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[13].Render()
					}
					ctx.EmitCmpRegImm32(d173.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
					}
					ctx.FlushRegisterMoves()
					if !bbs[13].Rendered {
						snap174 := d0
						snap175 := d1
						snap176 := d2
						snap177 := d3
						snap178 := d9
						snap179 := d10
						snap180 := d12
						snap181 := d13
						snap182 := d14
						snap183 := d16
						snap184 := d17
						snap185 := d18
						snap186 := d32
						snap187 := d33
						snap188 := d34
						snap189 := d35
						snap190 := d53
						snap191 := d54
						snap192 := d55
						snap193 := d56
						snap194 := d57
						snap195 := d58
						snap196 := d59
						snap197 := d60
						snap198 := d61
						snap199 := d62
						snap200 := d90
						snap201 := d119
						snap202 := d120
						snap203 := d121
						snap204 := d122
						snap205 := d123
						snap206 := d124
						snap207 := d125
						snap208 := d126
						snap209 := d127
						snap210 := d128
						snap211 := d167
						snap212 := d168
						snap213 := d169
						snap214 := d170
						snap215 := d171
						snap216 := d172
						snap217 := d173
						alloc218 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc218)
						d0 = snap174
						d1 = snap175
						d2 = snap176
						d3 = snap177
						d9 = snap178
						d10 = snap179
						d12 = snap180
						d13 = snap181
						d14 = snap182
						d16 = snap183
						d17 = snap184
						d18 = snap185
						d32 = snap186
						d33 = snap187
						d34 = snap188
						d35 = snap189
						d53 = snap190
						d54 = snap191
						d55 = snap192
						d56 = snap193
						d57 = snap194
						d58 = snap195
						d59 = snap196
						d60 = snap197
						d61 = snap198
						d62 = snap199
						d90 = snap200
						d119 = snap201
						d120 = snap202
						d121 = snap203
						d122 = snap204
						d123 = snap205
						d124 = snap206
						d125 = snap207
						d126 = snap208
						d127 = snap209
						d128 = snap210
						d167 = snap211
						d168 = snap212
						d169 = snap213
						d170 = snap214
						d171 = snap215
						d172 = snap216
						d173 = snap217
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					ctx.FreeDesc(&d172)
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
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocRegTriple || d58.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d58)
					d219 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d58}, 1)
					d219.NoHeapPointer = true
					ctx.BindReg(d219.Reg, &d219)
					ctx.EnsureDesc(&d219)
					if d219.Loc == LocImm {
						d220 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d219.Imm.Float() / 24)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4627448617123184640))
						ctx.EmitDivFloat64(d219.Reg, ctx.ScratchReg)
						d220 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d219.Reg}
						ctx.BindReg(d219.Reg, &d220)
					}
					if d220.Loc == LocReg && d219.Loc == LocReg && d220.Reg == d219.Reg {
						ctx.TransferReg(d219.Reg)
						d219.Loc = LocNone
					}
					ctx.FreeDesc(&d219)
					ctx.EnsureDesc(&d220)
					ctx.EnsureDesc(&d220)
					if d220.Loc == LocImm {
						d221 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d220.Imm.Float()))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r3, d220.Reg)
						d221 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d221)
					}
					ctx.FreeDesc(&d220)
					ctx.EnsureDesc(&d221)
					if d221.Loc == LocImm {
						ctx.EmitMakeInt(result, d221)
					} else {
						ctx.EmitMovToReg(result.Reg2, d221)
						d222 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d222)
						if d221.Loc == LocReg && d221.Reg != result.Reg2 {
							ctx.FreeReg(d221.Reg)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d57)
					d223 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
					if d223.Loc == LocImm {
						ctx.TrackImm(d223.Imm)
						ptrWord, _ := d223.Imm.RawWords()
						d224 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d224.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d224.Reg2, uint64(len(d223.Imm.String())))
						ctx.BindReg(d224.Reg, &d224)
						ctx.BindReg(d224.Reg2, &d224)
					} else {
						d224 = d223
					}
					d225 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d224}, 1)
					ctx.EmitAndRegImm32(d225.Reg, 1)
					d225.Type = tagBool
					ctx.BindReg(d225.Reg, &d225)
					d226 = d225
					ctx.EnsureDesc(&d226)
					if d226.Loc != LocImm && d226.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d226.Loc == LocImm {
						if d226.Imm.Bool() {
							return bbs[12].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitCmpRegImm32(d226.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap227 := d0
						snap228 := d1
						snap229 := d2
						snap230 := d3
						snap231 := d9
						snap232 := d10
						snap233 := d12
						snap234 := d13
						snap235 := d14
						snap236 := d16
						snap237 := d17
						snap238 := d18
						snap239 := d32
						snap240 := d33
						snap241 := d34
						snap242 := d35
						snap243 := d53
						snap244 := d54
						snap245 := d55
						snap246 := d56
						snap247 := d57
						snap248 := d58
						snap249 := d59
						snap250 := d60
						snap251 := d61
						snap252 := d62
						snap253 := d90
						snap254 := d119
						snap255 := d120
						snap256 := d121
						snap257 := d122
						snap258 := d123
						snap259 := d124
						snap260 := d125
						snap261 := d126
						snap262 := d127
						snap263 := d128
						snap264 := d167
						snap265 := d168
						snap266 := d169
						snap267 := d170
						snap268 := d171
						snap269 := d172
						snap270 := d173
						snap271 := d219
						snap272 := d220
						snap273 := d221
						snap274 := d222
						snap275 := d223
						snap276 := d224
						snap277 := d225
						snap278 := d226
						alloc279 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc279)
						d0 = snap227
						d1 = snap228
						d2 = snap229
						d3 = snap230
						d9 = snap231
						d10 = snap232
						d12 = snap233
						d13 = snap234
						d14 = snap235
						d16 = snap236
						d17 = snap237
						d18 = snap238
						d32 = snap239
						d33 = snap240
						d34 = snap241
						d35 = snap242
						d53 = snap243
						d54 = snap244
						d55 = snap245
						d56 = snap246
						d57 = snap247
						d58 = snap248
						d59 = snap249
						d60 = snap250
						d61 = snap251
						d62 = snap252
						d90 = snap253
						d119 = snap254
						d120 = snap255
						d121 = snap256
						d122 = snap257
						d123 = snap258
						d124 = snap259
						d125 = snap260
						d126 = snap261
						d127 = snap262
						d128 = snap263
						d167 = snap264
						d168 = snap265
						d169 = snap266
						d170 = snap267
						d171 = snap268
						d172 = snap269
						d173 = snap270
						d219 = snap271
						d220 = snap272
						d221 = snap273
						d222 = snap274
						d223 = snap275
						d224 = snap276
						d225 = snap277
						d226 = snap278
					}
					if !bbs[12].Rendered {
						return bbs[12].Render()
					}
					return result
					ctx.FreeDesc(&d225)
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
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocRegTriple || d58.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d58)
					d280 = ctx.EmitGoCallScalar(GoFuncAddr((time.Duration).Hours), []JITValueDesc{d58}, 1)
					d280.NoHeapPointer = true
					ctx.BindReg(d280.Reg, &d280)
					ctx.EnsureDesc(&d280)
					if d280.Loc == LocImm {
						d281 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d280.Imm.Float() / 168)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(4640114991075164160))
						ctx.EmitDivFloat64(d280.Reg, ctx.ScratchReg)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d280.Reg}
						ctx.BindReg(d280.Reg, &d281)
					}
					if d281.Loc == LocReg && d280.Loc == LocReg && d281.Reg == d280.Reg {
						ctx.TransferReg(d280.Reg)
						d280.Loc = LocNone
					}
					ctx.FreeDesc(&d280)
					ctx.EnsureDesc(&d281)
					ctx.EnsureDesc(&d281)
					if d281.Loc == LocImm {
						d282 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d281.Imm.Float()))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r4, d281.Reg)
						d282 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d282)
					}
					ctx.FreeDesc(&d281)
					ctx.EnsureDesc(&d282)
					if d282.Loc == LocImm {
						ctx.EmitMakeInt(result, d282)
					} else {
						ctx.EmitMovToReg(result.Reg2, d282)
						d283 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d283)
						if d282.Loc == LocReg && d282.Reg != result.Reg2 {
							ctx.FreeReg(d282.Reg)
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
					ctx.EnsureDesc(&d57)
					d284 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					if d284.Loc == LocImm {
						ctx.TrackImm(d284.Imm)
						ptrWord, _ := d284.Imm.RawWords()
						d285 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d285.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d285.Reg2, uint64(len(d284.Imm.String())))
						ctx.BindReg(d285.Reg, &d285)
						ctx.BindReg(d285.Reg2, &d285)
					} else {
						d285 = d284
					}
					d286 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d285}, 1)
					ctx.EmitAndRegImm32(d286.Reg, 1)
					d286.Type = tagBool
					ctx.BindReg(d286.Reg, &d286)
					d287 = d286
					ctx.EnsureDesc(&d287)
					if d287.Loc != LocImm && d287.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d287.Loc == LocImm {
						if d287.Imm.Bool() {
							return bbs[14].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitCmpRegImm32(d287.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap288 := d0
						snap289 := d1
						snap290 := d2
						snap291 := d3
						snap292 := d9
						snap293 := d10
						snap294 := d12
						snap295 := d13
						snap296 := d14
						snap297 := d16
						snap298 := d17
						snap299 := d18
						snap300 := d32
						snap301 := d33
						snap302 := d34
						snap303 := d35
						snap304 := d53
						snap305 := d54
						snap306 := d55
						snap307 := d56
						snap308 := d57
						snap309 := d58
						snap310 := d59
						snap311 := d60
						snap312 := d61
						snap313 := d62
						snap314 := d90
						snap315 := d119
						snap316 := d120
						snap317 := d121
						snap318 := d122
						snap319 := d123
						snap320 := d124
						snap321 := d125
						snap322 := d126
						snap323 := d127
						snap324 := d128
						snap325 := d167
						snap326 := d168
						snap327 := d169
						snap328 := d170
						snap329 := d171
						snap330 := d172
						snap331 := d173
						snap332 := d219
						snap333 := d220
						snap334 := d221
						snap335 := d222
						snap336 := d223
						snap337 := d224
						snap338 := d225
						snap339 := d226
						snap340 := d280
						snap341 := d281
						snap342 := d282
						snap343 := d283
						snap344 := d284
						snap345 := d285
						snap346 := d286
						snap347 := d287
						alloc348 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc348)
						d0 = snap288
						d1 = snap289
						d2 = snap290
						d3 = snap291
						d9 = snap292
						d10 = snap293
						d12 = snap294
						d13 = snap295
						d14 = snap296
						d16 = snap297
						d17 = snap298
						d18 = snap299
						d32 = snap300
						d33 = snap301
						d34 = snap302
						d35 = snap303
						d53 = snap304
						d54 = snap305
						d55 = snap306
						d56 = snap307
						d57 = snap308
						d58 = snap309
						d59 = snap310
						d60 = snap311
						d61 = snap312
						d62 = snap313
						d90 = snap314
						d119 = snap315
						d120 = snap316
						d121 = snap317
						d122 = snap318
						d123 = snap319
						d124 = snap320
						d125 = snap321
						d126 = snap322
						d127 = snap323
						d128 = snap324
						d167 = snap325
						d168 = snap326
						d169 = snap327
						d170 = snap328
						d171 = snap329
						d172 = snap330
						d173 = snap331
						d219 = snap332
						d220 = snap333
						d221 = snap334
						d222 = snap335
						d223 = snap336
						d224 = snap337
						d225 = snap338
						d226 = snap339
						d280 = snap340
						d281 = snap341
						d282 = snap342
						d283 = snap343
						d284 = snap344
						d285 = snap345
						d286 = snap346
						d287 = snap347
					}
					if !bbs[14].Rendered {
						return bbs[14].Render()
					}
					return result
					ctx.FreeDesc(&d286)
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
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d12)
					callResults349 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d12}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d350 = callResults349[0]
					_ = d350
					d351 = callResults349[1]
					_ = d351
					d352 = callResults349[2]
					_ = d352
					d16 = JITPrepareGoSliceArg(ctx, d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d16)
					callResults353 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d16}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d354 = callResults353[0]
					_ = d354
					d355 = callResults353[1]
					_ = d355
					d356 = callResults353[2]
					_ = d356
					ctx.EnsureDesc(&d354)
					ctx.EnsureDesc(&d350)
					ctx.SyncDesc(&d354)
					ctx.SyncDesc(&d350)
					if d354.Loc == LocImm && d350.Loc == LocImm {
						d357 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d354.Imm.Int() - d350.Imm.Int())}
					} else if d350.Loc == LocImm && d350.Imm.Int() == 0 {
						ctx.EnsureDesc(&d354)
						r5 := ctx.AllocRegExcept(d354.Reg)
						ctx.EmitMovRegReg(r5, d354.Reg)
						d357 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d357)
					} else if d354.Loc == LocImm {
						ctx.EnsureDesc(&d350)
						scratch := ctx.AllocRegExcept(d350.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d354.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d350)
						d357 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d357)
					} else if d350.Loc == LocImm {
						ctx.EnsureDesc(&d354)
						scratch := ctx.AllocRegExcept(d354.Reg)
						ctx.EmitMovRegReg(scratch, d354.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d350.Imm.Int())
						d357 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d357)
					} else {
						ctx.EnsureDesc(&d354)
						ctx.SyncDesc(&d350)
						r6 := ctx.AllocRegExceptOperand(&d350, d354.Reg)
						ctx.EmitMovRegReg(r6, d354.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r6, &d350)
						d357 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d357)
					}
					if d357.Loc == LocReg && d354.Loc == LocReg && d357.Reg == d354.Reg {
						ctx.TransferReg(d354.Reg)
						d354.Loc = LocNone
					}
					ctx.FreeDesc(&d354)
					ctx.FreeDesc(&d350)
					ctx.EnsureDesc(&d357)
					ctx.EnsureDesc(&d357)
					if d357.Loc == LocImm {
						d358 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d357.Imm.Int() * 12)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d357.Reg, 12)
						d358 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d357.Reg}
						ctx.BindReg(d357.Reg, &d358)
					}
					if d358.Loc == LocReg && d357.Loc == LocReg && d358.Reg == d357.Reg {
						ctx.TransferReg(d357.Reg)
						d357.Loc = LocNone
					}
					ctx.FreeDesc(&d357)
					ctx.EnsureDesc(&d355)
					ctx.EnsureDesc(&d351)
					ctx.SyncDesc(&d355)
					ctx.SyncDesc(&d351)
					if d355.Loc == LocImm && d351.Loc == LocImm {
						d359 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d355.Imm.Int() - d351.Imm.Int())}
					} else if d351.Loc == LocImm && d351.Imm.Int() == 0 {
						ctx.EnsureDesc(&d355)
						r7 := ctx.AllocRegExcept(d355.Reg)
						ctx.EmitMovRegReg(r7, d355.Reg)
						d359 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d359)
					} else if d355.Loc == LocImm {
						ctx.EnsureDesc(&d351)
						scratch := ctx.AllocRegExcept(d351.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d355.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d351)
						d359 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d359)
					} else if d351.Loc == LocImm {
						ctx.EnsureDesc(&d355)
						scratch := ctx.AllocRegExcept(d355.Reg)
						ctx.EmitMovRegReg(scratch, d355.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d351.Imm.Int())
						d359 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d359)
					} else {
						ctx.EnsureDesc(&d355)
						ctx.SyncDesc(&d351)
						r8 := ctx.AllocRegExceptOperand(&d351, d355.Reg)
						ctx.EmitMovRegReg(r8, d355.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r8, &d351)
						d359 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d359)
					}
					if d359.Loc == LocReg && d355.Loc == LocReg && d359.Reg == d355.Reg {
						ctx.TransferReg(d355.Reg)
						d355.Loc = LocNone
					}
					ctx.FreeDesc(&d355)
					ctx.FreeDesc(&d351)
					ctx.EnsureDesc(&d359)
					ctx.FreeDesc(&d359)
					ctx.EnsureDesc(&d358)
					ctx.EnsureDesc(&d359)
					ctx.SyncDesc(&d358)
					ctx.SyncDesc(&d359)
					if d358.Loc == LocImm && d359.Loc == LocImm {
						d360 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d358.Imm.Int() + d359.Imm.Int())}
					} else if d359.Loc == LocImm && d359.Imm.Int() == 0 {
						ctx.EnsureDesc(&d358)
						r9 := ctx.AllocRegExcept(d358.Reg)
						ctx.EmitMovRegReg(r9, d358.Reg)
						d360 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d360)
					} else if d358.Loc == LocImm && d358.Imm.Int() == 0 {
						ctx.EnsureDesc(&d359)
						d360 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d359.Reg}
						ctx.BindReg(d359.Reg, &d360)
					} else if d358.Loc == LocImm {
						ctx.EnsureDesc(&d359)
						scratch := ctx.AllocRegExcept(d359.Reg)
						ctx.EmitMovRegReg(scratch, d359.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d358.Imm.Int())
						d360 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d360)
					} else if d359.Loc == LocImm {
						ctx.EnsureDesc(&d358)
						scratch := ctx.AllocRegExcept(d358.Reg)
						ctx.EmitMovRegReg(scratch, d358.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d359.Imm.Int())
						d360 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d360)
					} else {
						ctx.EnsureDesc(&d358)
						ctx.SyncDesc(&d359)
						r10 := ctx.AllocRegExceptOperand(&d359, d358.Reg)
						ctx.EmitMovRegReg(r10, d358.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r10, &d359)
						d360 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d360)
					}
					if d360.Loc == LocReg && d358.Loc == LocReg && d360.Reg == d358.Reg {
						ctx.TransferReg(d358.Reg)
						d358.Loc = LocNone
					}
					ctx.FreeDesc(&d358)
					ctx.FreeDesc(&d359)
					ctx.EnsureDesc(&d360)
					ctx.EnsureDesc(&d360)
					ctx.EnsureDesc(&d360)
					if d360.Loc == LocImm {
						ctx.EmitMakeInt(result, d360)
					} else {
						ctx.EmitMovToReg(result.Reg2, d360)
						d362 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d362)
						if d360.Loc == LocReg && d360.Reg != result.Reg2 {
							ctx.FreeReg(d360.Reg)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d57)
					d363 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					if d363.Loc == LocImm {
						ctx.TrackImm(d363.Imm)
						ptrWord, _ := d363.Imm.RawWords()
						d364 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d364.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d364.Reg2, uint64(len(d363.Imm.String())))
						ctx.BindReg(d364.Reg, &d364)
						ctx.BindReg(d364.Reg2, &d364)
					} else {
						d364 = d363
					}
					d365 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d364}, 1)
					ctx.EmitAndRegImm32(d365.Reg, 1)
					d365.Type = tagBool
					ctx.BindReg(d365.Reg, &d365)
					d366 = d365
					ctx.EnsureDesc(&d366)
					if d366.Loc != LocImm && d366.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d366.Loc == LocImm {
						if d366.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d366.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap367 := d0
						snap368 := d1
						snap369 := d2
						snap370 := d3
						snap371 := d9
						snap372 := d10
						snap373 := d12
						snap374 := d13
						snap375 := d14
						snap376 := d16
						snap377 := d17
						snap378 := d18
						snap379 := d32
						snap380 := d33
						snap381 := d34
						snap382 := d35
						snap383 := d53
						snap384 := d54
						snap385 := d55
						snap386 := d56
						snap387 := d57
						snap388 := d58
						snap389 := d59
						snap390 := d60
						snap391 := d61
						snap392 := d62
						snap393 := d90
						snap394 := d119
						snap395 := d120
						snap396 := d121
						snap397 := d122
						snap398 := d123
						snap399 := d124
						snap400 := d125
						snap401 := d126
						snap402 := d127
						snap403 := d128
						snap404 := d167
						snap405 := d168
						snap406 := d169
						snap407 := d170
						snap408 := d171
						snap409 := d172
						snap410 := d173
						snap411 := d219
						snap412 := d220
						snap413 := d221
						snap414 := d222
						snap415 := d223
						snap416 := d224
						snap417 := d225
						snap418 := d226
						snap419 := d280
						snap420 := d281
						snap421 := d282
						snap422 := d283
						snap423 := d284
						snap424 := d285
						snap425 := d286
						snap426 := d287
						snap427 := d350
						snap428 := d351
						snap429 := d352
						snap430 := d354
						snap431 := d355
						snap432 := d356
						snap433 := d357
						snap434 := d358
						snap435 := d359
						snap436 := d360
						snap437 := d361
						snap438 := d362
						snap439 := d363
						snap440 := d364
						snap441 := d365
						snap442 := d366
						alloc443 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc443)
						d0 = snap367
						d1 = snap368
						d2 = snap369
						d3 = snap370
						d9 = snap371
						d10 = snap372
						d12 = snap373
						d13 = snap374
						d14 = snap375
						d16 = snap376
						d17 = snap377
						d18 = snap378
						d32 = snap379
						d33 = snap380
						d34 = snap381
						d35 = snap382
						d53 = snap383
						d54 = snap384
						d55 = snap385
						d56 = snap386
						d57 = snap387
						d58 = snap388
						d59 = snap389
						d60 = snap390
						d61 = snap391
						d62 = snap392
						d90 = snap393
						d119 = snap394
						d120 = snap395
						d121 = snap396
						d122 = snap397
						d123 = snap398
						d124 = snap399
						d125 = snap400
						d126 = snap401
						d127 = snap402
						d128 = snap403
						d167 = snap404
						d168 = snap405
						d169 = snap406
						d170 = snap407
						d171 = snap408
						d172 = snap409
						d173 = snap410
						d219 = snap411
						d220 = snap412
						d221 = snap413
						d222 = snap414
						d223 = snap415
						d224 = snap416
						d225 = snap417
						d226 = snap418
						d280 = snap419
						d281 = snap420
						d282 = snap421
						d283 = snap422
						d284 = snap423
						d285 = snap424
						d286 = snap425
						d287 = snap426
						d350 = snap427
						d351 = snap428
						d352 = snap429
						d354 = snap430
						d355 = snap431
						d356 = snap432
						d357 = snap433
						d358 = snap434
						d359 = snap435
						d360 = snap436
						d361 = snap437
						d362 = snap438
						d363 = snap439
						d364 = snap440
						d365 = snap441
						d366 = snap442
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d365)
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
					ctx.ReclaimUntrackedRegs()
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d12)
					callResults444 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d12}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d445 = callResults444[0]
					_ = d445
					d446 = callResults444[1]
					_ = d446
					d447 = callResults444[2]
					_ = d447
					d16 = JITPrepareGoSliceArg(ctx, d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d16)
					callResults448 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d16}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d449 = callResults448[0]
					_ = d449
					d450 = callResults448[1]
					_ = d450
					d451 = callResults448[2]
					_ = d451
					ctx.EnsureDesc(&d449)
					ctx.EnsureDesc(&d445)
					ctx.SyncDesc(&d449)
					ctx.SyncDesc(&d445)
					if d449.Loc == LocImm && d445.Loc == LocImm {
						d452 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d449.Imm.Int() - d445.Imm.Int())}
					} else if d445.Loc == LocImm && d445.Imm.Int() == 0 {
						ctx.EnsureDesc(&d449)
						r11 := ctx.AllocRegExcept(d449.Reg)
						ctx.EmitMovRegReg(r11, d449.Reg)
						d452 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d452)
					} else if d449.Loc == LocImm {
						ctx.EnsureDesc(&d445)
						scratch := ctx.AllocRegExcept(d445.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d449.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d445)
						d452 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d452)
					} else if d445.Loc == LocImm {
						ctx.EnsureDesc(&d449)
						scratch := ctx.AllocRegExcept(d449.Reg)
						ctx.EmitMovRegReg(scratch, d449.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d445.Imm.Int())
						d452 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d452)
					} else {
						ctx.EnsureDesc(&d449)
						ctx.SyncDesc(&d445)
						r12 := ctx.AllocRegExceptOperand(&d445, d449.Reg)
						ctx.EmitMovRegReg(r12, d449.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r12, &d445)
						d452 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
						ctx.BindReg(r12, &d452)
					}
					if d452.Loc == LocReg && d449.Loc == LocReg && d452.Reg == d449.Reg {
						ctx.TransferReg(d449.Reg)
						d449.Loc = LocNone
					}
					ctx.FreeDesc(&d449)
					ctx.FreeDesc(&d445)
					ctx.EnsureDesc(&d452)
					ctx.EnsureDesc(&d452)
					ctx.EnsureDesc(&d452)
					if d452.Loc == LocImm {
						ctx.EmitMakeInt(result, d452)
					} else {
						ctx.EmitMovToReg(result.Reg2, d452)
						d454 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d454)
						if d452.Loc == LocReg && d452.Reg != result.Reg2 {
							ctx.FreeReg(d452.Reg)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d57)
					d455 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					if d455.Loc == LocImm {
						ctx.TrackImm(d455.Imm)
						ptrWord, _ := d455.Imm.RawWords()
						d456 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d456.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d456.Reg2, uint64(len(d455.Imm.String())))
						ctx.BindReg(d456.Reg, &d456)
						ctx.BindReg(d456.Reg2, &d456)
					} else {
						d456 = d455
					}
					d457 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d57, d456}, 1)
					ctx.EmitAndRegImm32(d457.Reg, 1)
					d457.Type = tagBool
					ctx.BindReg(d457.Reg, &d457)
					d458 = d457
					ctx.EnsureDesc(&d458)
					if d458.Loc != LocImm && d458.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d458.Loc == LocImm {
						if d458.Imm.Bool() {
							return bbs[18].Render()
						}
						return bbs[20].Render()
					}
					ctx.EmitCmpRegImm32(d458.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					if bbs[20].Rendered {
						ctx.EmitJmp(lbl21)
					}
					ctx.FlushRegisterMoves()
					if !bbs[20].Rendered {
						snap459 := d0
						snap460 := d1
						snap461 := d2
						snap462 := d3
						snap463 := d9
						snap464 := d10
						snap465 := d12
						snap466 := d13
						snap467 := d14
						snap468 := d16
						snap469 := d17
						snap470 := d18
						snap471 := d32
						snap472 := d33
						snap473 := d34
						snap474 := d35
						snap475 := d53
						snap476 := d54
						snap477 := d55
						snap478 := d56
						snap479 := d57
						snap480 := d58
						snap481 := d59
						snap482 := d60
						snap483 := d61
						snap484 := d62
						snap485 := d90
						snap486 := d119
						snap487 := d120
						snap488 := d121
						snap489 := d122
						snap490 := d123
						snap491 := d124
						snap492 := d125
						snap493 := d126
						snap494 := d127
						snap495 := d128
						snap496 := d167
						snap497 := d168
						snap498 := d169
						snap499 := d170
						snap500 := d171
						snap501 := d172
						snap502 := d173
						snap503 := d219
						snap504 := d220
						snap505 := d221
						snap506 := d222
						snap507 := d223
						snap508 := d224
						snap509 := d225
						snap510 := d226
						snap511 := d280
						snap512 := d281
						snap513 := d282
						snap514 := d283
						snap515 := d284
						snap516 := d285
						snap517 := d286
						snap518 := d287
						snap519 := d350
						snap520 := d351
						snap521 := d352
						snap522 := d354
						snap523 := d355
						snap524 := d356
						snap525 := d357
						snap526 := d358
						snap527 := d359
						snap528 := d360
						snap529 := d361
						snap530 := d362
						snap531 := d363
						snap532 := d364
						snap533 := d365
						snap534 := d366
						snap535 := d445
						snap536 := d446
						snap537 := d447
						snap538 := d449
						snap539 := d450
						snap540 := d451
						snap541 := d452
						snap542 := d453
						snap543 := d454
						snap544 := d455
						snap545 := d456
						snap546 := d457
						snap547 := d458
						alloc548 := ctx.SnapshotAllocState()
						bbs[20].Render()
						ctx.RestoreAllocState(alloc548)
						d0 = snap459
						d1 = snap460
						d2 = snap461
						d3 = snap462
						d9 = snap463
						d10 = snap464
						d12 = snap465
						d13 = snap466
						d14 = snap467
						d16 = snap468
						d17 = snap469
						d18 = snap470
						d32 = snap471
						d33 = snap472
						d34 = snap473
						d35 = snap474
						d53 = snap475
						d54 = snap476
						d55 = snap477
						d56 = snap478
						d57 = snap479
						d58 = snap480
						d59 = snap481
						d60 = snap482
						d61 = snap483
						d62 = snap484
						d90 = snap485
						d119 = snap486
						d120 = snap487
						d121 = snap488
						d122 = snap489
						d123 = snap490
						d124 = snap491
						d125 = snap492
						d126 = snap493
						d127 = snap494
						d128 = snap495
						d167 = snap496
						d168 = snap497
						d169 = snap498
						d170 = snap499
						d171 = snap500
						d172 = snap501
						d173 = snap502
						d219 = snap503
						d220 = snap504
						d221 = snap505
						d222 = snap506
						d223 = snap507
						d224 = snap508
						d225 = snap509
						d226 = snap510
						d280 = snap511
						d281 = snap512
						d282 = snap513
						d283 = snap514
						d284 = snap515
						d285 = snap516
						d286 = snap517
						d287 = snap518
						d350 = snap519
						d351 = snap520
						d352 = snap521
						d354 = snap522
						d355 = snap523
						d356 = snap524
						d357 = snap525
						d358 = snap526
						d359 = snap527
						d360 = snap528
						d361 = snap529
						d362 = snap530
						d363 = snap531
						d364 = snap532
						d365 = snap533
						d366 = snap534
						d445 = snap535
						d446 = snap536
						d447 = snap537
						d449 = snap538
						d450 = snap539
						d451 = snap540
						d452 = snap541
						d453 = snap542
						d454 = snap543
						d455 = snap544
						d456 = snap545
						d457 = snap546
						d458 = snap547
					}
					if !bbs[18].Rendered {
						return bbs[18].Render()
					}
					return result
					ctx.FreeDesc(&d457)
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
					ctx.ReclaimUntrackedRegs()
					d549 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d549)
					if d549.Loc == LocRegPair || d549.Loc == LocStackPair || d549.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d549, &result)
						result.Type = d549.Type
					} else {
						switch d549.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d549)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d549)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d549)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d549, &result)
							result.Type = d549.Type
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
			JITInlineCost:  95,
		},
	})
}

// parseDateStringInLoc parses a date string as a local time in loc.
func parseDateStringInLoc(s string, loc *time.Location) (int64, bool) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, fmt := range formats {
		if t, err := time.ParseInLocation(fmt, s, loc); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// formatDateMySQL formats a time.Time using MySQL format specifiers.
func formatDateMySQL(t time.Time, format string) string {
	var buf strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'Y':
				buf.WriteString(fmt.Sprintf("%04d", t.Year()))
			case 'y':
				buf.WriteString(fmt.Sprintf("%02d", t.Year()%100))
			case 'm':
				buf.WriteString(fmt.Sprintf("%02d", t.Month()))
			case 'd':
				buf.WriteString(fmt.Sprintf("%02d", t.Day()))
			case 'H':
				buf.WriteString(fmt.Sprintf("%02d", t.Hour()))
			case 'i':
				buf.WriteString(fmt.Sprintf("%02d", t.Minute()))
			case 's':
				buf.WriteString(fmt.Sprintf("%02d", t.Second()))
			case 'T':
				buf.WriteString(fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second()))
			case '%':
				buf.WriteByte('%')
			default:
				buf.WriteByte('%')
				buf.WriteByte(format[i+1])
			}
			i++
		} else {
			buf.WriteByte(format[i])
		}
	}
	return buf.String()
}
