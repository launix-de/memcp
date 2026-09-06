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
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d0)
					var d1 JITValueDesc
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						ctx.EmitSetcc(r0, CondEqual)
						d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							if ps.General {
							}
							ps3 := PhiState{General: ps.General}
							ps3.OverlayValues = make([]JITValueDesc, 3)
							ps3.OverlayValues[0] = d0
							ps3.OverlayValues[1] = d1
							ps3.OverlayValues[2] = d2
							return bbs[1].RenderPS(ps3)
						}
						if ps.General {
						}
						ps4 := PhiState{General: ps.General}
						ps4.OverlayValues = make([]JITValueDesc, 3)
						ps4.OverlayValues[0] = d0
						ps4.OverlayValues[1] = d1
						ps4.OverlayValues[2] = d2
						return bbs[2].RenderPS(ps4)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl3)
					ps5 := PhiState{General: true}
					ps5.OverlayValues = make([]JITValueDesc, 3)
					ps5.OverlayValues[0] = d0
					ps5.OverlayValues[1] = d1
					ps5.OverlayValues[2] = d2
					ps6 := PhiState{General: true}
					ps6.OverlayValues = make([]JITValueDesc, 3)
					ps6.OverlayValues[0] = d0
					ps6.OverlayValues[1] = d1
					ps6.OverlayValues[2] = d2
					snap7 := d0
					snap8 := d1
					snap9 := d2
					alloc10 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps6)
					}
					ctx.RestoreAllocState(alloc10)
					d0 = snap7
					d1 = snap8
					d2 = snap9
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps5)
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(time.Now), []JITValueDesc{}, 3)
					d11.NoHeapPointer = false
					ctx.BindReg(d11.Reg, &d11)
					ctx.BindReg(d11.Reg2, &d11)
					ctx.BindReg(d11.Reg3, &d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc != LocRegTriple && d11.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d11)
					d12 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d11}, 1)
					d12.NoHeapPointer = true
					ctx.BindReg(d12.Reg, &d12)
					ctx.FreeDesc(&d11)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						ctx.EmitMakeInt(result, d12)
					} else {
						ctx.EmitMovToReg(result.Reg2, d12)
						d13 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d13)
						if d12.Loc == LocReg && d12.Reg != result.Reg2 {
							ctx.FreeReg(d12.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					ctx.ReclaimUntrackedRegs()
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					d16.ID = 0
					d15 = ctx.EmitTagEqualsBorrowed(&d16, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d14)
					d17 = d15
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							if ps.General {
							}
							ps18 := PhiState{General: ps.General}
							ps18.OverlayValues = make([]JITValueDesc, 18)
							ps18.OverlayValues[0] = d0
							ps18.OverlayValues[1] = d1
							ps18.OverlayValues[2] = d2
							ps18.OverlayValues[11] = d11
							ps18.OverlayValues[12] = d12
							ps18.OverlayValues[13] = d13
							ps18.OverlayValues[14] = d14
							ps18.OverlayValues[15] = d15
							ps18.OverlayValues[16] = d16
							ps18.OverlayValues[17] = d17
							return bbs[3].RenderPS(ps18)
						}
						if ps.General {
						}
						ps19 := PhiState{General: ps.General}
						ps19.OverlayValues = make([]JITValueDesc, 18)
						ps19.OverlayValues[0] = d0
						ps19.OverlayValues[1] = d1
						ps19.OverlayValues[2] = d2
						ps19.OverlayValues[11] = d11
						ps19.OverlayValues[12] = d12
						ps19.OverlayValues[13] = d13
						ps19.OverlayValues[14] = d14
						ps19.OverlayValues[15] = d15
						ps19.OverlayValues[16] = d16
						ps19.OverlayValues[17] = d17
						return bbs[4].RenderPS(ps19)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl5)
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 18)
					ps20.OverlayValues[0] = d0
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					ps20.OverlayValues[17] = d17
					ps21 := PhiState{General: true}
					ps21.OverlayValues = make([]JITValueDesc, 18)
					ps21.OverlayValues[0] = d0
					ps21.OverlayValues[1] = d1
					ps21.OverlayValues[2] = d2
					ps21.OverlayValues[11] = d11
					ps21.OverlayValues[12] = d12
					ps21.OverlayValues[13] = d13
					ps21.OverlayValues[14] = d14
					ps21.OverlayValues[15] = d15
					ps21.OverlayValues[16] = d16
					ps21.OverlayValues[17] = d17
					snap22 := d0
					snap23 := d1
					snap24 := d2
					snap25 := d11
					snap26 := d12
					snap27 := d13
					snap28 := d14
					snap29 := d15
					snap30 := d16
					snap31 := d17
					alloc32 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps21)
					}
					ctx.RestoreAllocState(alloc32)
					d0 = snap22
					d1 = snap23
					d2 = snap24
					d11 = snap25
					d12 = snap26
					d13 = snap27
					d14 = snap28
					d15 = snap29
					d16 = snap30
					d17 = snap31
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps20)
					}
					return result
					ctx.FreeDesc(&d15)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					ctx.ReclaimUntrackedRegs()
					d33 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d33)
					if d33.Loc == LocRegPair || d33.Loc == LocStackPair || d33.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d33, &result)
						result.Type = d33.Type
					} else {
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d33)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d33)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d33)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d33, &result)
							result.Type = d33.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					ctx.ReclaimUntrackedRegs()
					d34 = args[0]
					d34.ID = 0
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					d34 = JITPrepareScmerGoArg(ctx, d34)
					ctx.SyncDesc(&d34)
					callResults35 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d34}, []uint8{3, 1}, []uint8{4, 0})
					d36 = callResults35[0]
					_ = d36
					d37 = callResults35[1]
					_ = d37
					ctx.FreeDesc(&d34)
					ctx.StabilizeDescForControlFlow(&d36)
					d38 = d37
					ctx.EnsureDesc(&d38)
					if d38.Loc != LocImm && d38.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d38.Loc == LocImm {
						if d38.Imm.Bool() {
							if ps.General {
							}
							ps39 := PhiState{General: ps.General}
							ps39.OverlayValues = make([]JITValueDesc, 39)
							ps39.OverlayValues[0] = d0
							ps39.OverlayValues[1] = d1
							ps39.OverlayValues[2] = d2
							ps39.OverlayValues[11] = d11
							ps39.OverlayValues[12] = d12
							ps39.OverlayValues[13] = d13
							ps39.OverlayValues[14] = d14
							ps39.OverlayValues[15] = d15
							ps39.OverlayValues[16] = d16
							ps39.OverlayValues[17] = d17
							ps39.OverlayValues[33] = d33
							ps39.OverlayValues[34] = d34
							ps39.OverlayValues[36] = d36
							ps39.OverlayValues[37] = d37
							ps39.OverlayValues[38] = d38
							return bbs[6].RenderPS(ps39)
						}
						if ps.General {
						}
						ps40 := PhiState{General: ps.General}
						ps40.OverlayValues = make([]JITValueDesc, 39)
						ps40.OverlayValues[0] = d0
						ps40.OverlayValues[1] = d1
						ps40.OverlayValues[2] = d2
						ps40.OverlayValues[11] = d11
						ps40.OverlayValues[12] = d12
						ps40.OverlayValues[13] = d13
						ps40.OverlayValues[14] = d14
						ps40.OverlayValues[15] = d15
						ps40.OverlayValues[16] = d16
						ps40.OverlayValues[17] = d17
						ps40.OverlayValues[33] = d33
						ps40.OverlayValues[34] = d34
						ps40.OverlayValues[36] = d36
						ps40.OverlayValues[37] = d37
						ps40.OverlayValues[38] = d38
						return bbs[5].RenderPS(ps40)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d38.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl6)
					ps41 := PhiState{General: true}
					ps41.OverlayValues = make([]JITValueDesc, 39)
					ps41.OverlayValues[0] = d0
					ps41.OverlayValues[1] = d1
					ps41.OverlayValues[2] = d2
					ps41.OverlayValues[11] = d11
					ps41.OverlayValues[12] = d12
					ps41.OverlayValues[13] = d13
					ps41.OverlayValues[14] = d14
					ps41.OverlayValues[15] = d15
					ps41.OverlayValues[16] = d16
					ps41.OverlayValues[17] = d17
					ps41.OverlayValues[33] = d33
					ps41.OverlayValues[34] = d34
					ps41.OverlayValues[36] = d36
					ps41.OverlayValues[37] = d37
					ps41.OverlayValues[38] = d38
					ps42 := PhiState{General: true}
					ps42.OverlayValues = make([]JITValueDesc, 39)
					ps42.OverlayValues[0] = d0
					ps42.OverlayValues[1] = d1
					ps42.OverlayValues[2] = d2
					ps42.OverlayValues[11] = d11
					ps42.OverlayValues[12] = d12
					ps42.OverlayValues[13] = d13
					ps42.OverlayValues[14] = d14
					ps42.OverlayValues[15] = d15
					ps42.OverlayValues[16] = d16
					ps42.OverlayValues[17] = d17
					ps42.OverlayValues[33] = d33
					ps42.OverlayValues[34] = d34
					ps42.OverlayValues[36] = d36
					ps42.OverlayValues[37] = d37
					ps42.OverlayValues[38] = d38
					snap43 := d0
					snap44 := d1
					snap45 := d2
					snap46 := d11
					snap47 := d12
					snap48 := d13
					snap49 := d14
					snap50 := d15
					snap51 := d16
					snap52 := d17
					snap53 := d33
					snap54 := d34
					snap55 := d36
					snap56 := d37
					snap57 := d38
					alloc58 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps42)
					}
					ctx.RestoreAllocState(alloc58)
					d0 = snap43
					d1 = snap44
					d2 = snap45
					d11 = snap46
					d12 = snap47
					d13 = snap48
					d14 = snap49
					d15 = snap50
					d16 = snap51
					d17 = snap52
					d33 = snap53
					d34 = snap54
					d36 = snap55
					d37 = snap56
					d38 = snap57
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps41)
					}
					return result
					ctx.FreeDesc(&d37)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					ctx.ReclaimUntrackedRegs()
					d59 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d59)
					if d59.Loc == LocRegPair || d59.Loc == LocStackPair || d59.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d59, &result)
						result.Type = d59.Type
					} else {
						switch d59.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d59)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d59)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d59)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d59, &result)
							result.Type = d59.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					if d36.Loc != LocRegTriple && d36.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d36)
					d60 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d36}, 1)
					d60.NoHeapPointer = true
					ctx.BindReg(d60.Reg, &d60)
					ctx.FreeDesc(&d36)
					ctx.EnsureDesc(&d60)
					if d60.Loc == LocImm {
						ctx.EmitMakeInt(result, d60)
					} else {
						ctx.EmitMovToReg(result.Reg2, d60)
						d61 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d61)
						if d60.Loc == LocReg && d60.Reg != result.Reg2 {
							ctx.FreeReg(d60.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				ps62 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps62)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
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
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d198 JITValueDesc
				_ = d198
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d203 JITValueDesc
				_ = d203
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
				var d210 JITValueDesc
				_ = d210
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d217 JITValueDesc
				_ = d217
				var d219 JITValueDesc
				_ = d219
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d295 JITValueDesc
				_ = d295
				var d296 JITValueDesc
				_ = d296
				var d297 JITValueDesc
				_ = d297
				var d298 JITValueDesc
				_ = d298
				var d300 JITValueDesc
				_ = d300
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [14]BBDescriptor
				bbs[9].PhiBase = int32(phiBase0) + int32(0)
				bbs[9].PhiCount = uint16(1)
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
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
							if ps.General {
							}
							ps6 := PhiState{General: ps.General}
							ps6.OverlayValues = make([]JITValueDesc, 6)
							ps6.OverlayValues[1] = d1
							ps6.OverlayValues[2] = d2
							ps6.OverlayValues[3] = d3
							ps6.OverlayValues[4] = d4
							ps6.OverlayValues[5] = d5
							return bbs[1].RenderPS(ps6)
						}
						if ps.General {
						}
						ps7 := PhiState{General: ps.General}
						ps7.OverlayValues = make([]JITValueDesc, 6)
						ps7.OverlayValues[1] = d1
						ps7.OverlayValues[2] = d2
						ps7.OverlayValues[3] = d3
						ps7.OverlayValues[4] = d4
						ps7.OverlayValues[5] = d5
						return bbs[4].RenderPS(ps7)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl5)
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 6)
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[2] = d2
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					ps8.OverlayValues[5] = d5
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 6)
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					snap10 := d1
					snap11 := d2
					snap12 := d3
					snap13 := d4
					snap14 := d5
					alloc15 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps9)
					}
					ctx.RestoreAllocState(alloc15)
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps8)
					}
					return result
					ctx.FreeDesc(&d3)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
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
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					ctx.ReclaimUntrackedRegs()
					d17 = args[1]
					d17.ID = 0
					d19 = d17
					ctx.SyncDesc(&d19)
					if d19.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d19.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d19.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d19 = tmpScalar
					}
					d19 = JITPrepareScmerGoArg(ctx, d19)
					if d19.Loc != LocRegPair && d19.Loc != LocStackPair && d19.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d19}, 2)
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d18.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d18.Imm)
						ptrWord, _ := d18.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d18.Imm.String())))
						d18 = tmpPair
					} else if d18.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d18.Type, Reg: ctx.AllocRegExcept(d18.Reg), Reg2: ctx.AllocRegExcept(d18.Reg)}
						switch d18.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d18)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d18)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d18)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d18)
						d18 = tmpPair
					}
					if d18.Loc != LocRegPair && d18.Loc != LocStackPair && d18.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d18)
					callResults20 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d18}, []uint8{1, 2}, []uint8{1, 3})
					d21 = callResults20[0]
					_ = d21
					d22 = callResults20[1]
					_ = d22
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.EnsureDesc(&d22)
					var d23 JITValueDesc
					if d22.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d22.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d22)
						if d22.Loc != LocReg && d22.Loc != LocRegPair && d22.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d22.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d23)
					}
					ctx.FreeDesc(&d22)
					d24 = d23
					ctx.EnsureDesc(&d24)
					if d24.Loc != LocImm && d24.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d24.Loc == LocImm {
						if d24.Imm.Bool() {
							if ps.General {
							}
							ps25 := PhiState{General: ps.General}
							ps25.OverlayValues = make([]JITValueDesc, 25)
							ps25.OverlayValues[1] = d1
							ps25.OverlayValues[2] = d2
							ps25.OverlayValues[3] = d3
							ps25.OverlayValues[4] = d4
							ps25.OverlayValues[5] = d5
							ps25.OverlayValues[16] = d16
							ps25.OverlayValues[17] = d17
							ps25.OverlayValues[18] = d18
							ps25.OverlayValues[19] = d19
							ps25.OverlayValues[21] = d21
							ps25.OverlayValues[22] = d22
							ps25.OverlayValues[23] = d23
							ps25.OverlayValues[24] = d24
							return bbs[5].RenderPS(ps25)
						}
						if ps.General {
						}
						ps26 := PhiState{General: ps.General}
						ps26.OverlayValues = make([]JITValueDesc, 25)
						ps26.OverlayValues[1] = d1
						ps26.OverlayValues[2] = d2
						ps26.OverlayValues[3] = d3
						ps26.OverlayValues[4] = d4
						ps26.OverlayValues[5] = d5
						ps26.OverlayValues[16] = d16
						ps26.OverlayValues[17] = d17
						ps26.OverlayValues[18] = d18
						ps26.OverlayValues[19] = d19
						ps26.OverlayValues[21] = d21
						ps26.OverlayValues[22] = d22
						ps26.OverlayValues[23] = d23
						ps26.OverlayValues[24] = d24
						return bbs[6].RenderPS(ps26)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d24.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl7)
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 25)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[17] = d17
					ps27.OverlayValues[18] = d18
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					ps27.OverlayValues[24] = d24
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 25)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[18] = d18
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[24] = d24
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d16
					snap35 := d17
					snap36 := d18
					snap37 := d19
					snap38 := d21
					snap39 := d22
					snap40 := d23
					snap41 := d24
					alloc42 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps28)
					}
					ctx.RestoreAllocState(alloc42)
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d16 = snap34
					d17 = snap35
					d18 = snap36
					d19 = snap37
					d21 = snap38
					d22 = snap39
					d23 = snap40
					d24 = snap41
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps27)
					}
					return result
					ctx.FreeDesc(&d23)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					ctx.ReclaimUntrackedRegs()
					d43 = args[2]
					d43.ID = 0
					d45 = d43
					d45.ID = 0
					d44 = ctx.EmitTagEqualsBorrowed(&d45, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d43)
					d46 = d44
					ctx.EnsureDesc(&d46)
					if d46.Loc != LocImm && d46.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d46.Loc == LocImm {
						if d46.Imm.Bool() {
							if ps.General {
							}
							ps47 := PhiState{General: ps.General}
							ps47.OverlayValues = make([]JITValueDesc, 47)
							ps47.OverlayValues[1] = d1
							ps47.OverlayValues[2] = d2
							ps47.OverlayValues[3] = d3
							ps47.OverlayValues[4] = d4
							ps47.OverlayValues[5] = d5
							ps47.OverlayValues[16] = d16
							ps47.OverlayValues[17] = d17
							ps47.OverlayValues[18] = d18
							ps47.OverlayValues[19] = d19
							ps47.OverlayValues[21] = d21
							ps47.OverlayValues[22] = d22
							ps47.OverlayValues[23] = d23
							ps47.OverlayValues[24] = d24
							ps47.OverlayValues[43] = d43
							ps47.OverlayValues[44] = d44
							ps47.OverlayValues[45] = d45
							ps47.OverlayValues[46] = d46
							return bbs[1].RenderPS(ps47)
						}
						if ps.General {
						}
						ps48 := PhiState{General: ps.General}
						ps48.OverlayValues = make([]JITValueDesc, 47)
						ps48.OverlayValues[1] = d1
						ps48.OverlayValues[2] = d2
						ps48.OverlayValues[3] = d3
						ps48.OverlayValues[4] = d4
						ps48.OverlayValues[5] = d5
						ps48.OverlayValues[16] = d16
						ps48.OverlayValues[17] = d17
						ps48.OverlayValues[18] = d18
						ps48.OverlayValues[19] = d19
						ps48.OverlayValues[21] = d21
						ps48.OverlayValues[22] = d22
						ps48.OverlayValues[23] = d23
						ps48.OverlayValues[24] = d24
						ps48.OverlayValues[43] = d43
						ps48.OverlayValues[44] = d44
						ps48.OverlayValues[45] = d45
						ps48.OverlayValues[46] = d46
						return bbs[2].RenderPS(ps48)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d46.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl3)
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 47)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[16] = d16
					ps49.OverlayValues[17] = d17
					ps49.OverlayValues[18] = d18
					ps49.OverlayValues[19] = d19
					ps49.OverlayValues[21] = d21
					ps49.OverlayValues[22] = d22
					ps49.OverlayValues[23] = d23
					ps49.OverlayValues[24] = d24
					ps49.OverlayValues[43] = d43
					ps49.OverlayValues[44] = d44
					ps49.OverlayValues[45] = d45
					ps49.OverlayValues[46] = d46
					ps50 := PhiState{General: true}
					ps50.OverlayValues = make([]JITValueDesc, 47)
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[4] = d4
					ps50.OverlayValues[5] = d5
					ps50.OverlayValues[16] = d16
					ps50.OverlayValues[17] = d17
					ps50.OverlayValues[18] = d18
					ps50.OverlayValues[19] = d19
					ps50.OverlayValues[21] = d21
					ps50.OverlayValues[22] = d22
					ps50.OverlayValues[23] = d23
					ps50.OverlayValues[24] = d24
					ps50.OverlayValues[43] = d43
					ps50.OverlayValues[44] = d44
					ps50.OverlayValues[45] = d45
					ps50.OverlayValues[46] = d46
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d16
					snap57 := d17
					snap58 := d18
					snap59 := d19
					snap60 := d21
					snap61 := d22
					snap62 := d23
					snap63 := d24
					snap64 := d43
					snap65 := d44
					snap66 := d45
					snap67 := d46
					alloc68 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps50)
					}
					ctx.RestoreAllocState(alloc68)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d16 = snap56
					d17 = snap57
					d18 = snap58
					d19 = snap59
					d21 = snap60
					d22 = snap61
					d23 = snap62
					d24 = snap63
					d43 = snap64
					d44 = snap65
					d45 = snap66
					d46 = snap67
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps49)
					}
					return result
					ctx.FreeDesc(&d44)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					ctx.ReclaimUntrackedRegs()
					d69 = args[1]
					d69.ID = 0
					d71 = d69
					d71.ID = 0
					d70 = ctx.EmitTagEqualsBorrowed(&d71, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d69)
					d72 = d70
					ctx.EnsureDesc(&d72)
					if d72.Loc != LocImm && d72.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d72.Loc == LocImm {
						if d72.Imm.Bool() {
							if ps.General {
							}
							ps73 := PhiState{General: ps.General}
							ps73.OverlayValues = make([]JITValueDesc, 73)
							ps73.OverlayValues[1] = d1
							ps73.OverlayValues[2] = d2
							ps73.OverlayValues[3] = d3
							ps73.OverlayValues[4] = d4
							ps73.OverlayValues[5] = d5
							ps73.OverlayValues[16] = d16
							ps73.OverlayValues[17] = d17
							ps73.OverlayValues[18] = d18
							ps73.OverlayValues[19] = d19
							ps73.OverlayValues[21] = d21
							ps73.OverlayValues[22] = d22
							ps73.OverlayValues[23] = d23
							ps73.OverlayValues[24] = d24
							ps73.OverlayValues[43] = d43
							ps73.OverlayValues[44] = d44
							ps73.OverlayValues[45] = d45
							ps73.OverlayValues[46] = d46
							ps73.OverlayValues[69] = d69
							ps73.OverlayValues[70] = d70
							ps73.OverlayValues[71] = d71
							ps73.OverlayValues[72] = d72
							return bbs[1].RenderPS(ps73)
						}
						if ps.General {
						}
						ps74 := PhiState{General: ps.General}
						ps74.OverlayValues = make([]JITValueDesc, 73)
						ps74.OverlayValues[1] = d1
						ps74.OverlayValues[2] = d2
						ps74.OverlayValues[3] = d3
						ps74.OverlayValues[4] = d4
						ps74.OverlayValues[5] = d5
						ps74.OverlayValues[16] = d16
						ps74.OverlayValues[17] = d17
						ps74.OverlayValues[18] = d18
						ps74.OverlayValues[19] = d19
						ps74.OverlayValues[21] = d21
						ps74.OverlayValues[22] = d22
						ps74.OverlayValues[23] = d23
						ps74.OverlayValues[24] = d24
						ps74.OverlayValues[43] = d43
						ps74.OverlayValues[44] = d44
						ps74.OverlayValues[45] = d45
						ps74.OverlayValues[46] = d46
						ps74.OverlayValues[69] = d69
						ps74.OverlayValues[70] = d70
						ps74.OverlayValues[71] = d71
						ps74.OverlayValues[72] = d72
						return bbs[3].RenderPS(ps74)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d72.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl4)
					ps75 := PhiState{General: true}
					ps75.OverlayValues = make([]JITValueDesc, 73)
					ps75.OverlayValues[1] = d1
					ps75.OverlayValues[2] = d2
					ps75.OverlayValues[3] = d3
					ps75.OverlayValues[4] = d4
					ps75.OverlayValues[5] = d5
					ps75.OverlayValues[16] = d16
					ps75.OverlayValues[17] = d17
					ps75.OverlayValues[18] = d18
					ps75.OverlayValues[19] = d19
					ps75.OverlayValues[21] = d21
					ps75.OverlayValues[22] = d22
					ps75.OverlayValues[23] = d23
					ps75.OverlayValues[24] = d24
					ps75.OverlayValues[43] = d43
					ps75.OverlayValues[44] = d44
					ps75.OverlayValues[45] = d45
					ps75.OverlayValues[46] = d46
					ps75.OverlayValues[69] = d69
					ps75.OverlayValues[70] = d70
					ps75.OverlayValues[71] = d71
					ps75.OverlayValues[72] = d72
					ps76 := PhiState{General: true}
					ps76.OverlayValues = make([]JITValueDesc, 73)
					ps76.OverlayValues[1] = d1
					ps76.OverlayValues[2] = d2
					ps76.OverlayValues[3] = d3
					ps76.OverlayValues[4] = d4
					ps76.OverlayValues[5] = d5
					ps76.OverlayValues[16] = d16
					ps76.OverlayValues[17] = d17
					ps76.OverlayValues[18] = d18
					ps76.OverlayValues[19] = d19
					ps76.OverlayValues[21] = d21
					ps76.OverlayValues[22] = d22
					ps76.OverlayValues[23] = d23
					ps76.OverlayValues[24] = d24
					ps76.OverlayValues[43] = d43
					ps76.OverlayValues[44] = d44
					ps76.OverlayValues[45] = d45
					ps76.OverlayValues[46] = d46
					ps76.OverlayValues[69] = d69
					ps76.OverlayValues[70] = d70
					ps76.OverlayValues[71] = d71
					ps76.OverlayValues[72] = d72
					snap77 := d1
					snap78 := d2
					snap79 := d3
					snap80 := d4
					snap81 := d5
					snap82 := d16
					snap83 := d17
					snap84 := d18
					snap85 := d19
					snap86 := d21
					snap87 := d22
					snap88 := d23
					snap89 := d24
					snap90 := d43
					snap91 := d44
					snap92 := d45
					snap93 := d46
					snap94 := d69
					snap95 := d70
					snap96 := d71
					snap97 := d72
					alloc98 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps76)
					}
					ctx.RestoreAllocState(alloc98)
					d1 = snap77
					d2 = snap78
					d3 = snap79
					d4 = snap80
					d5 = snap81
					d16 = snap82
					d17 = snap83
					d18 = snap84
					d19 = snap85
					d21 = snap86
					d22 = snap87
					d23 = snap88
					d24 = snap89
					d43 = snap90
					d44 = snap91
					d45 = snap92
					d46 = snap93
					d69 = snap94
					d70 = snap95
					d71 = snap96
					d72 = snap97
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps75)
					}
					return result
					ctx.FreeDesc(&d70)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					ctx.ReclaimUntrackedRegs()
					d99 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d99)
					if d99.Loc == LocRegPair || d99.Loc == LocStackPair || d99.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d99, &result)
						result.Type = d99.Type
					} else {
						switch d99.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d99)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d99)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d99)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d99, &result)
							result.Type = d99.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					ctx.ReclaimUntrackedRegs()
					d100 = args[2]
					d100.ID = 0
					d102 = d100
					ctx.SyncDesc(&d102)
					if d102.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d102.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d102.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d102 = tmpScalar
					}
					d102 = JITPrepareScmerGoArg(ctx, d102)
					if d102.Loc != LocRegPair && d102.Loc != LocStackPair && d102.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d101 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d102}, 2)
					ctx.FreeDesc(&d100)
					ctx.EnsureDesc(&d101)
					ctx.EnsureDesc(&d101)
					ctx.EnsureDesc(&d101)
					if d101.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d101.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d101.Imm)
						ptrWord, _ := d101.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d101.Imm.String())))
						d101 = tmpPair
					} else if d101.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d101.Type, Reg: ctx.AllocRegExcept(d101.Reg), Reg2: ctx.AllocRegExcept(d101.Reg)}
						switch d101.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d101)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d101)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d101)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d101)
						d101 = tmpPair
					}
					if d101.Loc != LocRegPair && d101.Loc != LocStackPair && d101.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d101)
					callResults103 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d101}, []uint8{1, 2}, []uint8{1, 3})
					d104 = callResults103[0]
					_ = d104
					d105 = callResults103[1]
					_ = d105
					ctx.StabilizeDescForControlFlow(&d104)
					ctx.EnsureDesc(&d105)
					var d106 JITValueDesc
					if d105.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d105.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d105)
						if d105.Loc != LocReg && d105.Loc != LocRegPair && d105.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d105.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d106 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d106)
					}
					ctx.FreeDesc(&d105)
					d107 = d106
					ctx.EnsureDesc(&d107)
					if d107.Loc != LocImm && d107.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d107.Loc == LocImm {
						if d107.Imm.Bool() {
							if ps.General {
							}
							ps108 := PhiState{General: ps.General}
							ps108.OverlayValues = make([]JITValueDesc, 108)
							ps108.OverlayValues[1] = d1
							ps108.OverlayValues[2] = d2
							ps108.OverlayValues[3] = d3
							ps108.OverlayValues[4] = d4
							ps108.OverlayValues[5] = d5
							ps108.OverlayValues[16] = d16
							ps108.OverlayValues[17] = d17
							ps108.OverlayValues[18] = d18
							ps108.OverlayValues[19] = d19
							ps108.OverlayValues[21] = d21
							ps108.OverlayValues[22] = d22
							ps108.OverlayValues[23] = d23
							ps108.OverlayValues[24] = d24
							ps108.OverlayValues[43] = d43
							ps108.OverlayValues[44] = d44
							ps108.OverlayValues[45] = d45
							ps108.OverlayValues[46] = d46
							ps108.OverlayValues[69] = d69
							ps108.OverlayValues[70] = d70
							ps108.OverlayValues[71] = d71
							ps108.OverlayValues[72] = d72
							ps108.OverlayValues[99] = d99
							ps108.OverlayValues[100] = d100
							ps108.OverlayValues[101] = d101
							ps108.OverlayValues[102] = d102
							ps108.OverlayValues[104] = d104
							ps108.OverlayValues[105] = d105
							ps108.OverlayValues[106] = d106
							ps108.OverlayValues[107] = d107
							return bbs[7].RenderPS(ps108)
						}
						if ps.General {
						}
						ps109 := PhiState{General: ps.General}
						ps109.OverlayValues = make([]JITValueDesc, 108)
						ps109.OverlayValues[1] = d1
						ps109.OverlayValues[2] = d2
						ps109.OverlayValues[3] = d3
						ps109.OverlayValues[4] = d4
						ps109.OverlayValues[5] = d5
						ps109.OverlayValues[16] = d16
						ps109.OverlayValues[17] = d17
						ps109.OverlayValues[18] = d18
						ps109.OverlayValues[19] = d19
						ps109.OverlayValues[21] = d21
						ps109.OverlayValues[22] = d22
						ps109.OverlayValues[23] = d23
						ps109.OverlayValues[24] = d24
						ps109.OverlayValues[43] = d43
						ps109.OverlayValues[44] = d44
						ps109.OverlayValues[45] = d45
						ps109.OverlayValues[46] = d46
						ps109.OverlayValues[69] = d69
						ps109.OverlayValues[70] = d70
						ps109.OverlayValues[71] = d71
						ps109.OverlayValues[72] = d72
						ps109.OverlayValues[99] = d99
						ps109.OverlayValues[100] = d100
						ps109.OverlayValues[101] = d101
						ps109.OverlayValues[102] = d102
						ps109.OverlayValues[104] = d104
						ps109.OverlayValues[105] = d105
						ps109.OverlayValues[106] = d106
						ps109.OverlayValues[107] = d107
						return bbs[8].RenderPS(ps109)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d107.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl23)
					ctx.EmitJmp(lbl24)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl9)
					ps110 := PhiState{General: true}
					ps110.OverlayValues = make([]JITValueDesc, 108)
					ps110.OverlayValues[1] = d1
					ps110.OverlayValues[2] = d2
					ps110.OverlayValues[3] = d3
					ps110.OverlayValues[4] = d4
					ps110.OverlayValues[5] = d5
					ps110.OverlayValues[16] = d16
					ps110.OverlayValues[17] = d17
					ps110.OverlayValues[18] = d18
					ps110.OverlayValues[19] = d19
					ps110.OverlayValues[21] = d21
					ps110.OverlayValues[22] = d22
					ps110.OverlayValues[23] = d23
					ps110.OverlayValues[24] = d24
					ps110.OverlayValues[43] = d43
					ps110.OverlayValues[44] = d44
					ps110.OverlayValues[45] = d45
					ps110.OverlayValues[46] = d46
					ps110.OverlayValues[69] = d69
					ps110.OverlayValues[70] = d70
					ps110.OverlayValues[71] = d71
					ps110.OverlayValues[72] = d72
					ps110.OverlayValues[99] = d99
					ps110.OverlayValues[100] = d100
					ps110.OverlayValues[101] = d101
					ps110.OverlayValues[102] = d102
					ps110.OverlayValues[104] = d104
					ps110.OverlayValues[105] = d105
					ps110.OverlayValues[106] = d106
					ps110.OverlayValues[107] = d107
					ps111 := PhiState{General: true}
					ps111.OverlayValues = make([]JITValueDesc, 108)
					ps111.OverlayValues[1] = d1
					ps111.OverlayValues[2] = d2
					ps111.OverlayValues[3] = d3
					ps111.OverlayValues[4] = d4
					ps111.OverlayValues[5] = d5
					ps111.OverlayValues[16] = d16
					ps111.OverlayValues[17] = d17
					ps111.OverlayValues[18] = d18
					ps111.OverlayValues[19] = d19
					ps111.OverlayValues[21] = d21
					ps111.OverlayValues[22] = d22
					ps111.OverlayValues[23] = d23
					ps111.OverlayValues[24] = d24
					ps111.OverlayValues[43] = d43
					ps111.OverlayValues[44] = d44
					ps111.OverlayValues[45] = d45
					ps111.OverlayValues[46] = d46
					ps111.OverlayValues[69] = d69
					ps111.OverlayValues[70] = d70
					ps111.OverlayValues[71] = d71
					ps111.OverlayValues[72] = d72
					ps111.OverlayValues[99] = d99
					ps111.OverlayValues[100] = d100
					ps111.OverlayValues[101] = d101
					ps111.OverlayValues[102] = d102
					ps111.OverlayValues[104] = d104
					ps111.OverlayValues[105] = d105
					ps111.OverlayValues[106] = d106
					ps111.OverlayValues[107] = d107
					snap112 := d1
					snap113 := d2
					snap114 := d3
					snap115 := d4
					snap116 := d5
					snap117 := d16
					snap118 := d17
					snap119 := d18
					snap120 := d19
					snap121 := d21
					snap122 := d22
					snap123 := d23
					snap124 := d24
					snap125 := d43
					snap126 := d44
					snap127 := d45
					snap128 := d46
					snap129 := d69
					snap130 := d70
					snap131 := d71
					snap132 := d72
					snap133 := d99
					snap134 := d100
					snap135 := d101
					snap136 := d102
					snap137 := d104
					snap138 := d105
					snap139 := d106
					snap140 := d107
					alloc141 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps111)
					}
					ctx.RestoreAllocState(alloc141)
					d1 = snap112
					d2 = snap113
					d3 = snap114
					d4 = snap115
					d5 = snap116
					d16 = snap117
					d17 = snap118
					d18 = snap119
					d19 = snap120
					d21 = snap121
					d22 = snap122
					d23 = snap123
					d24 = snap124
					d43 = snap125
					d44 = snap126
					d45 = snap127
					d46 = snap128
					d69 = snap129
					d70 = snap130
					d71 = snap131
					d72 = snap132
					d99 = snap133
					d100 = snap134
					d101 = snap135
					d102 = snap136
					d104 = snap137
					d105 = snap138
					d106 = snap139
					d107 = snap140
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps110)
					}
					return result
					ctx.FreeDesc(&d106)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					ctx.ReclaimUntrackedRegs()
					d142 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d142)
					if d142.Loc == LocRegPair || d142.Loc == LocStackPair || d142.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d142, &result)
						result.Type = d142.Type
					} else {
						switch d142.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d142)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d142)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d142)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d142, &result)
							result.Type = d142.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					ctx.ReclaimUntrackedRegs()
					d143 = args[0]
					d143.ID = 0
					d144 = ctx.EmitGetTagDesc(&d143, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d143)
					ctx.EnsureDesc(&d144)
					var d145 JITValueDesc
					if d144.Loc == LocImm {
						d145 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d144.Imm.Int()) == uint64(0x10))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d144.Reg, 16)
						ctx.EmitSetcc(r2, CondEqual)
						d145 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d145)
					}
					ctx.FreeDesc(&d144)
					d146 = d145
					ctx.EnsureDesc(&d146)
					if d146.Loc != LocImm && d146.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d146.Loc == LocImm {
						if d146.Imm.Bool() {
							if ps.General {
							}
							ps147 := PhiState{General: ps.General}
							ps147.OverlayValues = make([]JITValueDesc, 147)
							ps147.OverlayValues[1] = d1
							ps147.OverlayValues[2] = d2
							ps147.OverlayValues[3] = d3
							ps147.OverlayValues[4] = d4
							ps147.OverlayValues[5] = d5
							ps147.OverlayValues[16] = d16
							ps147.OverlayValues[17] = d17
							ps147.OverlayValues[18] = d18
							ps147.OverlayValues[19] = d19
							ps147.OverlayValues[21] = d21
							ps147.OverlayValues[22] = d22
							ps147.OverlayValues[23] = d23
							ps147.OverlayValues[24] = d24
							ps147.OverlayValues[43] = d43
							ps147.OverlayValues[44] = d44
							ps147.OverlayValues[45] = d45
							ps147.OverlayValues[46] = d46
							ps147.OverlayValues[69] = d69
							ps147.OverlayValues[70] = d70
							ps147.OverlayValues[71] = d71
							ps147.OverlayValues[72] = d72
							ps147.OverlayValues[99] = d99
							ps147.OverlayValues[100] = d100
							ps147.OverlayValues[101] = d101
							ps147.OverlayValues[102] = d102
							ps147.OverlayValues[104] = d104
							ps147.OverlayValues[105] = d105
							ps147.OverlayValues[106] = d106
							ps147.OverlayValues[107] = d107
							ps147.OverlayValues[142] = d142
							ps147.OverlayValues[143] = d143
							ps147.OverlayValues[144] = d144
							ps147.OverlayValues[145] = d145
							ps147.OverlayValues[146] = d146
							return bbs[10].RenderPS(ps147)
						}
						if ps.General {
						}
						ps148 := PhiState{General: ps.General}
						ps148.OverlayValues = make([]JITValueDesc, 147)
						ps148.OverlayValues[1] = d1
						ps148.OverlayValues[2] = d2
						ps148.OverlayValues[3] = d3
						ps148.OverlayValues[4] = d4
						ps148.OverlayValues[5] = d5
						ps148.OverlayValues[16] = d16
						ps148.OverlayValues[17] = d17
						ps148.OverlayValues[18] = d18
						ps148.OverlayValues[19] = d19
						ps148.OverlayValues[21] = d21
						ps148.OverlayValues[22] = d22
						ps148.OverlayValues[23] = d23
						ps148.OverlayValues[24] = d24
						ps148.OverlayValues[43] = d43
						ps148.OverlayValues[44] = d44
						ps148.OverlayValues[45] = d45
						ps148.OverlayValues[46] = d46
						ps148.OverlayValues[69] = d69
						ps148.OverlayValues[70] = d70
						ps148.OverlayValues[71] = d71
						ps148.OverlayValues[72] = d72
						ps148.OverlayValues[99] = d99
						ps148.OverlayValues[100] = d100
						ps148.OverlayValues[101] = d101
						ps148.OverlayValues[102] = d102
						ps148.OverlayValues[104] = d104
						ps148.OverlayValues[105] = d105
						ps148.OverlayValues[106] = d106
						ps148.OverlayValues[107] = d107
						ps148.OverlayValues[142] = d142
						ps148.OverlayValues[143] = d143
						ps148.OverlayValues[144] = d144
						ps148.OverlayValues[145] = d145
						ps148.OverlayValues[146] = d146
						return bbs[11].RenderPS(ps148)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d146.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl25)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl12)
					ps149 := PhiState{General: true}
					ps149.OverlayValues = make([]JITValueDesc, 147)
					ps149.OverlayValues[1] = d1
					ps149.OverlayValues[2] = d2
					ps149.OverlayValues[3] = d3
					ps149.OverlayValues[4] = d4
					ps149.OverlayValues[5] = d5
					ps149.OverlayValues[16] = d16
					ps149.OverlayValues[17] = d17
					ps149.OverlayValues[18] = d18
					ps149.OverlayValues[19] = d19
					ps149.OverlayValues[21] = d21
					ps149.OverlayValues[22] = d22
					ps149.OverlayValues[23] = d23
					ps149.OverlayValues[24] = d24
					ps149.OverlayValues[43] = d43
					ps149.OverlayValues[44] = d44
					ps149.OverlayValues[45] = d45
					ps149.OverlayValues[46] = d46
					ps149.OverlayValues[69] = d69
					ps149.OverlayValues[70] = d70
					ps149.OverlayValues[71] = d71
					ps149.OverlayValues[72] = d72
					ps149.OverlayValues[99] = d99
					ps149.OverlayValues[100] = d100
					ps149.OverlayValues[101] = d101
					ps149.OverlayValues[102] = d102
					ps149.OverlayValues[104] = d104
					ps149.OverlayValues[105] = d105
					ps149.OverlayValues[106] = d106
					ps149.OverlayValues[107] = d107
					ps149.OverlayValues[142] = d142
					ps149.OverlayValues[143] = d143
					ps149.OverlayValues[144] = d144
					ps149.OverlayValues[145] = d145
					ps149.OverlayValues[146] = d146
					ps150 := PhiState{General: true}
					ps150.OverlayValues = make([]JITValueDesc, 147)
					ps150.OverlayValues[1] = d1
					ps150.OverlayValues[2] = d2
					ps150.OverlayValues[3] = d3
					ps150.OverlayValues[4] = d4
					ps150.OverlayValues[5] = d5
					ps150.OverlayValues[16] = d16
					ps150.OverlayValues[17] = d17
					ps150.OverlayValues[18] = d18
					ps150.OverlayValues[19] = d19
					ps150.OverlayValues[21] = d21
					ps150.OverlayValues[22] = d22
					ps150.OverlayValues[23] = d23
					ps150.OverlayValues[24] = d24
					ps150.OverlayValues[43] = d43
					ps150.OverlayValues[44] = d44
					ps150.OverlayValues[45] = d45
					ps150.OverlayValues[46] = d46
					ps150.OverlayValues[69] = d69
					ps150.OverlayValues[70] = d70
					ps150.OverlayValues[71] = d71
					ps150.OverlayValues[72] = d72
					ps150.OverlayValues[99] = d99
					ps150.OverlayValues[100] = d100
					ps150.OverlayValues[101] = d101
					ps150.OverlayValues[102] = d102
					ps150.OverlayValues[104] = d104
					ps150.OverlayValues[105] = d105
					ps150.OverlayValues[106] = d106
					ps150.OverlayValues[107] = d107
					ps150.OverlayValues[142] = d142
					ps150.OverlayValues[143] = d143
					ps150.OverlayValues[144] = d144
					ps150.OverlayValues[145] = d145
					ps150.OverlayValues[146] = d146
					snap151 := d1
					snap152 := d2
					snap153 := d3
					snap154 := d4
					snap155 := d5
					snap156 := d16
					snap157 := d17
					snap158 := d18
					snap159 := d19
					snap160 := d21
					snap161 := d22
					snap162 := d23
					snap163 := d24
					snap164 := d43
					snap165 := d44
					snap166 := d45
					snap167 := d46
					snap168 := d69
					snap169 := d70
					snap170 := d71
					snap171 := d72
					snap172 := d99
					snap173 := d100
					snap174 := d101
					snap175 := d102
					snap176 := d104
					snap177 := d105
					snap178 := d106
					snap179 := d107
					snap180 := d142
					snap181 := d143
					snap182 := d144
					snap183 := d145
					snap184 := d146
					alloc185 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps150)
					}
					ctx.RestoreAllocState(alloc185)
					d1 = snap151
					d2 = snap152
					d3 = snap153
					d4 = snap154
					d5 = snap155
					d16 = snap156
					d17 = snap157
					d18 = snap158
					d19 = snap159
					d21 = snap160
					d22 = snap161
					d23 = snap162
					d24 = snap163
					d43 = snap164
					d44 = snap165
					d45 = snap166
					d46 = snap167
					d69 = snap168
					d70 = snap169
					d71 = snap170
					d72 = snap171
					d99 = snap172
					d100 = snap173
					d101 = snap174
					d102 = snap175
					d104 = snap176
					d105 = snap177
					d106 = snap178
					d107 = snap179
					d142 = snap180
					d143 = snap181
					d144 = snap182
					d145 = snap183
					d146 = snap184
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps149)
					}
					return result
					ctx.FreeDesc(&d145)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d186 := ps.PhiValues[0]
							ctx.EnsureDesc(&d186)
							ctx.EmitStoreScmerToStack(d186, int32(bbs[9].PhiBase)+int32(0))
						}
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					ctx.EnsureDesc(&d104)
					ctx.EnsureDesc(&d104)
					if d104.Loc == LocRegPair || d104.Loc == LocStackPair || d104.Loc == LocRegTriple || d104.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d104)
					d187 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d1, d104}, 3)
					d187.NoHeapPointer = false
					ctx.BindReg(d187.Reg, &d187)
					ctx.BindReg(d187.Reg2, &d187)
					ctx.BindReg(d187.Reg3, &d187)
					ctx.FreeDesc(&d1)
					ctx.FreeDesc(&d104)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d187)
					d188 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d187}, 1)
					d188.NoHeapPointer = true
					ctx.BindReg(d188.Reg, &d188)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d187)
					d189 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d187}, 1)
					d189.NoHeapPointer = true
					ctx.BindReg(d189.Reg, &d189)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d187)
					d190 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d187}, 1)
					d190.NoHeapPointer = true
					ctx.BindReg(d190.Reg, &d190)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d187)
					d191 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d187}, 1)
					d191.NoHeapPointer = true
					ctx.BindReg(d191.Reg, &d191)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d187)
					d192 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d187}, 1)
					d192.NoHeapPointer = true
					ctx.BindReg(d192.Reg, &d192)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocRegTriple && d187.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d187)
					d193 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d187}, 1)
					d193.NoHeapPointer = true
					ctx.BindReg(d193.Reg, &d193)
					ctx.FreeDesc(&d187)
					d194 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.EnsureDesc(&d188)
					ctx.EnsureDesc(&d188)
					if d188.Loc == LocRegPair || d188.Loc == LocStackPair || d188.Loc == LocRegTriple || d188.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d189)
					ctx.EnsureDesc(&d189)
					if d189.Loc == LocRegPair || d189.Loc == LocStackPair || d189.Loc == LocRegTriple || d189.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d190)
					ctx.EnsureDesc(&d190)
					if d190.Loc == LocRegPair || d190.Loc == LocStackPair || d190.Loc == LocRegTriple || d190.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d191)
					ctx.EnsureDesc(&d191)
					if d191.Loc == LocRegPair || d191.Loc == LocStackPair || d191.Loc == LocRegTriple || d191.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d192)
					ctx.EnsureDesc(&d192)
					if d192.Loc == LocRegPair || d192.Loc == LocStackPair || d192.Loc == LocRegTriple || d192.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d193)
					ctx.EnsureDesc(&d193)
					if d193.Loc == LocRegPair || d193.Loc == LocStackPair || d193.Loc == LocRegTriple || d193.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d195 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d195.Loc == LocRegPair || d195.Loc == LocStackPair || d195.Loc == LocRegTriple || d195.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d194)
					ctx.EnsureDesc(&d194)
					if d194.Loc == LocRegPair || d194.Loc == LocStackPair || d194.Loc == LocRegTriple || d194.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d188)
					ctx.SyncDesc(&d189)
					ctx.SyncDesc(&d190)
					ctx.SyncDesc(&d191)
					ctx.SyncDesc(&d192)
					ctx.SyncDesc(&d193)
					ctx.SyncDesc(&d195)
					ctx.SyncDesc(&d194)
					d196 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d188, d189, d190, d191, d192, d193, d195, d194}, 3)
					d196.NoHeapPointer = false
					ctx.BindReg(d196.Reg, &d196)
					ctx.BindReg(d196.Reg2, &d196)
					ctx.BindReg(d196.Reg3, &d196)
					ctx.FreeDesc(&d195)
					ctx.FreeDesc(&d188)
					ctx.FreeDesc(&d189)
					ctx.FreeDesc(&d190)
					ctx.FreeDesc(&d191)
					ctx.FreeDesc(&d192)
					ctx.FreeDesc(&d193)
					ctx.FreeDesc(&d194)
					ctx.EnsureDesc(&d196)
					ctx.EnsureDesc(&d196)
					ctx.EnsureDesc(&d196)
					if d196.Loc != LocRegTriple && d196.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d196)
					d197 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d196}, 1)
					d197.NoHeapPointer = true
					ctx.BindReg(d197.Reg, &d197)
					ctx.FreeDesc(&d196)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc == LocRegPair || d197.Loc == LocStackPair || d197.Loc == LocRegTriple || d197.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d197)
					d198 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d197}, 2)
					d198.NoHeapPointer = false
					ctx.BindReg(d198.Reg, &d198)
					ctx.BindReg(d198.Reg2, &d198)
					ctx.FreeDesc(&d197)
					ctx.SyncDesc(&d198)
					if d198.Loc == LocRegPair || d198.Loc == LocStackPair || d198.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d198, &result)
						result.Type = d198.Type
					} else {
						switch d198.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d198)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d198)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d198)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d198, &result)
							result.Type = d198.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					ctx.ReclaimUntrackedRegs()
					d199 = args[0]
					d199.ID = 0
					var d200 JITValueDesc
					if d199.Loc == LocImm {
						d200 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d199.Imm.Int())}
					} else if d199.Type == tagInt && d199.Loc == LocRegPair {
						ctx.FreeReg(d199.Reg)
						d200 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d199.Reg2}
						ctx.BindReg(d199.Reg2, &d200)
						ctx.BindReg(d199.Reg2, &d200)
					} else if d199.Type == tagInt && d199.Loc == LocReg {
						d200 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d199.Reg}
						ctx.BindReg(d199.Reg, &d200)
						ctx.BindReg(d199.Reg, &d200)
					} else {
						d200 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d199}, 1)
						d200.Type = tagInt
						ctx.BindReg(d200.Reg, &d200)
					}
					ctx.FreeDesc(&d199)
					ctx.EnsureDesc(&d200)
					ctx.EnsureDesc(&d200)
					if d200.Loc == LocRegPair || d200.Loc == LocStackPair || d200.Loc == LocRegTriple || d200.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d201 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d201.Loc == LocRegPair || d201.Loc == LocStackPair || d201.Loc == LocRegTriple || d201.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d200)
					ctx.SyncDesc(&d201)
					d202 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d200, d201}, 3)
					d202.NoHeapPointer = false
					ctx.BindReg(d202.Reg, &d202)
					ctx.BindReg(d202.Reg2, &d202)
					ctx.BindReg(d202.Reg3, &d202)
					ctx.FreeDesc(&d201)
					ctx.FreeDesc(&d200)
					ctx.EnsureDesc(&d202)
					ctx.EnsureDesc(&d202)
					ctx.EnsureDesc(&d202)
					if d202.Loc != LocRegTriple && d202.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d202)
					d203 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d202}, 3)
					d203.NoHeapPointer = false
					ctx.BindReg(d203.Reg, &d203)
					ctx.BindReg(d203.Reg2, &d203)
					ctx.BindReg(d203.Reg3, &d203)
					ctx.FreeDesc(&d202)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc != LocRegTriple && d203.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d203)
					d204 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d203}, 1)
					d204.NoHeapPointer = true
					ctx.BindReg(d204.Reg, &d204)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc != LocRegTriple && d203.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d203)
					d205 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d203}, 1)
					d205.NoHeapPointer = true
					ctx.BindReg(d205.Reg, &d205)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc != LocRegTriple && d203.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d203)
					d206 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d203}, 1)
					d206.NoHeapPointer = true
					ctx.BindReg(d206.Reg, &d206)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc != LocRegTriple && d203.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d203)
					d207 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d203}, 1)
					d207.NoHeapPointer = true
					ctx.BindReg(d207.Reg, &d207)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc != LocRegTriple && d203.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d203)
					d208 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d203}, 1)
					d208.NoHeapPointer = true
					ctx.BindReg(d208.Reg, &d208)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc != LocRegTriple && d203.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d203)
					d209 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d203}, 1)
					d209.NoHeapPointer = true
					ctx.BindReg(d209.Reg, &d209)
					ctx.FreeDesc(&d203)
					ctx.EnsureDesc(&d204)
					ctx.EnsureDesc(&d204)
					if d204.Loc == LocRegPair || d204.Loc == LocStackPair || d204.Loc == LocRegTriple || d204.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d205)
					ctx.EnsureDesc(&d205)
					if d205.Loc == LocRegPair || d205.Loc == LocStackPair || d205.Loc == LocRegTriple || d205.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d206)
					ctx.EnsureDesc(&d206)
					if d206.Loc == LocRegPair || d206.Loc == LocStackPair || d206.Loc == LocRegTriple || d206.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d207)
					ctx.EnsureDesc(&d207)
					if d207.Loc == LocRegPair || d207.Loc == LocStackPair || d207.Loc == LocRegTriple || d207.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d208)
					ctx.EnsureDesc(&d208)
					if d208.Loc == LocRegPair || d208.Loc == LocStackPair || d208.Loc == LocRegTriple || d208.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d209)
					ctx.EnsureDesc(&d209)
					if d209.Loc == LocRegPair || d209.Loc == LocStackPair || d209.Loc == LocRegTriple || d209.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d210 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d210.Loc == LocRegPair || d210.Loc == LocStackPair || d210.Loc == LocRegTriple || d210.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d204)
					ctx.SyncDesc(&d205)
					ctx.SyncDesc(&d206)
					ctx.SyncDesc(&d207)
					ctx.SyncDesc(&d208)
					ctx.SyncDesc(&d209)
					ctx.SyncDesc(&d210)
					ctx.SyncDesc(&d21)
					d211 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d204, d205, d206, d207, d208, d209, d210, d21}, 3)
					d211.NoHeapPointer = false
					ctx.BindReg(d211.Reg, &d211)
					ctx.BindReg(d211.Reg2, &d211)
					ctx.BindReg(d211.Reg3, &d211)
					ctx.FreeDesc(&d210)
					ctx.StabilizeDescForControlFlow(&d211)
					ctx.FreeDesc(&d204)
					ctx.FreeDesc(&d205)
					ctx.FreeDesc(&d206)
					ctx.FreeDesc(&d207)
					ctx.FreeDesc(&d208)
					ctx.FreeDesc(&d209)
					if ps.General {
						ctx.SyncDesc(&d211)
						if d211.Loc == LocReg {
							ctx.ProtectReg(d211.Reg)
						} else if d211.Loc == LocRegPair {
							ctx.ProtectReg(d211.Reg)
							ctx.ProtectReg(d211.Reg2)
						}
						d212 = d211
						if d212.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d212)
						if d212.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d212, int32(bbs[9].PhiBase)+int32(0), 2)
						} else if d212.Loc == LocInputPair {
							ctx.EnsureDesc(&d212)
							ctx.EmitStoreScmerToStack(d212, int32(bbs[9].PhiBase)+int32(0))
						} else if d212.Loc == LocRegPair || d212.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d212, int32(bbs[9].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d212)
							ctx.EmitStoreToStack(d212, int32(bbs[9].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
						}
						if d211.Loc == LocReg {
							ctx.UnprotectReg(d211.Reg)
						} else if d211.Loc == LocRegPair {
							ctx.UnprotectReg(d211.Reg)
							ctx.UnprotectReg(d211.Reg2)
						}
					}
					ps213 := PhiState{General: ps.General}
					ps213.OverlayValues = make([]JITValueDesc, 213)
					ps213.OverlayValues[1] = d1
					ps213.OverlayValues[2] = d2
					ps213.OverlayValues[3] = d3
					ps213.OverlayValues[4] = d4
					ps213.OverlayValues[5] = d5
					ps213.OverlayValues[16] = d16
					ps213.OverlayValues[17] = d17
					ps213.OverlayValues[18] = d18
					ps213.OverlayValues[19] = d19
					ps213.OverlayValues[21] = d21
					ps213.OverlayValues[22] = d22
					ps213.OverlayValues[23] = d23
					ps213.OverlayValues[24] = d24
					ps213.OverlayValues[43] = d43
					ps213.OverlayValues[44] = d44
					ps213.OverlayValues[45] = d45
					ps213.OverlayValues[46] = d46
					ps213.OverlayValues[69] = d69
					ps213.OverlayValues[70] = d70
					ps213.OverlayValues[71] = d71
					ps213.OverlayValues[72] = d72
					ps213.OverlayValues[99] = d99
					ps213.OverlayValues[100] = d100
					ps213.OverlayValues[101] = d101
					ps213.OverlayValues[102] = d102
					ps213.OverlayValues[104] = d104
					ps213.OverlayValues[105] = d105
					ps213.OverlayValues[106] = d106
					ps213.OverlayValues[107] = d107
					ps213.OverlayValues[142] = d142
					ps213.OverlayValues[143] = d143
					ps213.OverlayValues[144] = d144
					ps213.OverlayValues[145] = d145
					ps213.OverlayValues[146] = d146
					ps213.OverlayValues[186] = d186
					ps213.OverlayValues[187] = d187
					ps213.OverlayValues[188] = d188
					ps213.OverlayValues[189] = d189
					ps213.OverlayValues[190] = d190
					ps213.OverlayValues[191] = d191
					ps213.OverlayValues[192] = d192
					ps213.OverlayValues[193] = d193
					ps213.OverlayValues[194] = d194
					ps213.OverlayValues[195] = d195
					ps213.OverlayValues[196] = d196
					ps213.OverlayValues[197] = d197
					ps213.OverlayValues[198] = d198
					ps213.OverlayValues[199] = d199
					ps213.OverlayValues[200] = d200
					ps213.OverlayValues[201] = d201
					ps213.OverlayValues[202] = d202
					ps213.OverlayValues[203] = d203
					ps213.OverlayValues[204] = d204
					ps213.OverlayValues[205] = d205
					ps213.OverlayValues[206] = d206
					ps213.OverlayValues[207] = d207
					ps213.OverlayValues[208] = d208
					ps213.OverlayValues[209] = d209
					ps213.OverlayValues[210] = d210
					ps213.OverlayValues[211] = d211
					ps213.OverlayValues[212] = d212
					ps213.PhiValues = make([]JITValueDesc, 1)
					d214 = d211
					ps213.PhiValues[0] = d214
					if ps213.General && bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					return bbs[9].RenderPS(ps213)
					return result
				}
				bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[11].VisitCount >= 0 {
							ps.General = true
							return bbs[11].RenderPS(ps)
						}
					}
					bbs[11].VisitCount++
					if ps.General {
						if bbs[11].Rendered {
							ctx.EmitJmp(lbl12)
							return result
						}
						bbs[11].Rendered = true
						bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_11 = bbs[11].Address
						ctx.MarkLabel(lbl12)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					ctx.ReclaimUntrackedRegs()
					d215 = args[0]
					d215.ID = 0
					d217 = d215
					ctx.SyncDesc(&d217)
					if d217.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d217.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d217.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d217 = tmpScalar
					}
					d217 = JITPrepareScmerGoArg(ctx, d217)
					if d217.Loc != LocRegPair && d217.Loc != LocStackPair && d217.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d216 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d217}, 2)
					ctx.FreeDesc(&d215)
					ctx.EnsureDesc(&d216)
					ctx.EnsureDesc(&d216)
					ctx.EnsureDesc(&d216)
					if d216.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d216.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d216.Imm)
						ptrWord, _ := d216.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d216.Imm.String())))
						d216 = tmpPair
					} else if d216.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d216.Type, Reg: ctx.AllocRegExcept(d216.Reg), Reg2: ctx.AllocRegExcept(d216.Reg)}
						switch d216.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d216)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d216)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d216)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d216)
						d216 = tmpPair
					}
					if d216.Loc != LocRegPair && d216.Loc != LocStackPair && d216.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (parseDateStringInLoc arg0)")
					}
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d216)
					ctx.SyncDesc(&d21)
					callResults218 := JITEmitGoCallResults(ctx, GoFuncAddr(parseDateStringInLoc), []JITValueDesc{d216, d21}, []uint8{1, 1}, []uint8{0, 0})
					d219 = callResults218[0]
					_ = d219
					d220 = callResults218[1]
					_ = d220
					ctx.FreeDesc(&d21)
					ctx.StabilizeDescForControlFlow(&d219)
					d221 = d220
					ctx.EnsureDesc(&d221)
					if d221.Loc != LocImm && d221.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d221.Loc == LocImm {
						if d221.Imm.Bool() {
							if ps.General {
							}
							ps222 := PhiState{General: ps.General}
							ps222.OverlayValues = make([]JITValueDesc, 222)
							ps222.OverlayValues[1] = d1
							ps222.OverlayValues[2] = d2
							ps222.OverlayValues[3] = d3
							ps222.OverlayValues[4] = d4
							ps222.OverlayValues[5] = d5
							ps222.OverlayValues[16] = d16
							ps222.OverlayValues[17] = d17
							ps222.OverlayValues[18] = d18
							ps222.OverlayValues[19] = d19
							ps222.OverlayValues[21] = d21
							ps222.OverlayValues[22] = d22
							ps222.OverlayValues[23] = d23
							ps222.OverlayValues[24] = d24
							ps222.OverlayValues[43] = d43
							ps222.OverlayValues[44] = d44
							ps222.OverlayValues[45] = d45
							ps222.OverlayValues[46] = d46
							ps222.OverlayValues[69] = d69
							ps222.OverlayValues[70] = d70
							ps222.OverlayValues[71] = d71
							ps222.OverlayValues[72] = d72
							ps222.OverlayValues[99] = d99
							ps222.OverlayValues[100] = d100
							ps222.OverlayValues[101] = d101
							ps222.OverlayValues[102] = d102
							ps222.OverlayValues[104] = d104
							ps222.OverlayValues[105] = d105
							ps222.OverlayValues[106] = d106
							ps222.OverlayValues[107] = d107
							ps222.OverlayValues[142] = d142
							ps222.OverlayValues[143] = d143
							ps222.OverlayValues[144] = d144
							ps222.OverlayValues[145] = d145
							ps222.OverlayValues[146] = d146
							ps222.OverlayValues[186] = d186
							ps222.OverlayValues[187] = d187
							ps222.OverlayValues[188] = d188
							ps222.OverlayValues[189] = d189
							ps222.OverlayValues[190] = d190
							ps222.OverlayValues[191] = d191
							ps222.OverlayValues[192] = d192
							ps222.OverlayValues[193] = d193
							ps222.OverlayValues[194] = d194
							ps222.OverlayValues[195] = d195
							ps222.OverlayValues[196] = d196
							ps222.OverlayValues[197] = d197
							ps222.OverlayValues[198] = d198
							ps222.OverlayValues[199] = d199
							ps222.OverlayValues[200] = d200
							ps222.OverlayValues[201] = d201
							ps222.OverlayValues[202] = d202
							ps222.OverlayValues[203] = d203
							ps222.OverlayValues[204] = d204
							ps222.OverlayValues[205] = d205
							ps222.OverlayValues[206] = d206
							ps222.OverlayValues[207] = d207
							ps222.OverlayValues[208] = d208
							ps222.OverlayValues[209] = d209
							ps222.OverlayValues[210] = d210
							ps222.OverlayValues[211] = d211
							ps222.OverlayValues[212] = d212
							ps222.OverlayValues[214] = d214
							ps222.OverlayValues[215] = d215
							ps222.OverlayValues[216] = d216
							ps222.OverlayValues[217] = d217
							ps222.OverlayValues[219] = d219
							ps222.OverlayValues[220] = d220
							ps222.OverlayValues[221] = d221
							return bbs[13].RenderPS(ps222)
						}
						if ps.General {
						}
						ps223 := PhiState{General: ps.General}
						ps223.OverlayValues = make([]JITValueDesc, 222)
						ps223.OverlayValues[1] = d1
						ps223.OverlayValues[2] = d2
						ps223.OverlayValues[3] = d3
						ps223.OverlayValues[4] = d4
						ps223.OverlayValues[5] = d5
						ps223.OverlayValues[16] = d16
						ps223.OverlayValues[17] = d17
						ps223.OverlayValues[18] = d18
						ps223.OverlayValues[19] = d19
						ps223.OverlayValues[21] = d21
						ps223.OverlayValues[22] = d22
						ps223.OverlayValues[23] = d23
						ps223.OverlayValues[24] = d24
						ps223.OverlayValues[43] = d43
						ps223.OverlayValues[44] = d44
						ps223.OverlayValues[45] = d45
						ps223.OverlayValues[46] = d46
						ps223.OverlayValues[69] = d69
						ps223.OverlayValues[70] = d70
						ps223.OverlayValues[71] = d71
						ps223.OverlayValues[72] = d72
						ps223.OverlayValues[99] = d99
						ps223.OverlayValues[100] = d100
						ps223.OverlayValues[101] = d101
						ps223.OverlayValues[102] = d102
						ps223.OverlayValues[104] = d104
						ps223.OverlayValues[105] = d105
						ps223.OverlayValues[106] = d106
						ps223.OverlayValues[107] = d107
						ps223.OverlayValues[142] = d142
						ps223.OverlayValues[143] = d143
						ps223.OverlayValues[144] = d144
						ps223.OverlayValues[145] = d145
						ps223.OverlayValues[146] = d146
						ps223.OverlayValues[186] = d186
						ps223.OverlayValues[187] = d187
						ps223.OverlayValues[188] = d188
						ps223.OverlayValues[189] = d189
						ps223.OverlayValues[190] = d190
						ps223.OverlayValues[191] = d191
						ps223.OverlayValues[192] = d192
						ps223.OverlayValues[193] = d193
						ps223.OverlayValues[194] = d194
						ps223.OverlayValues[195] = d195
						ps223.OverlayValues[196] = d196
						ps223.OverlayValues[197] = d197
						ps223.OverlayValues[198] = d198
						ps223.OverlayValues[199] = d199
						ps223.OverlayValues[200] = d200
						ps223.OverlayValues[201] = d201
						ps223.OverlayValues[202] = d202
						ps223.OverlayValues[203] = d203
						ps223.OverlayValues[204] = d204
						ps223.OverlayValues[205] = d205
						ps223.OverlayValues[206] = d206
						ps223.OverlayValues[207] = d207
						ps223.OverlayValues[208] = d208
						ps223.OverlayValues[209] = d209
						ps223.OverlayValues[210] = d210
						ps223.OverlayValues[211] = d211
						ps223.OverlayValues[212] = d212
						ps223.OverlayValues[214] = d214
						ps223.OverlayValues[215] = d215
						ps223.OverlayValues[216] = d216
						ps223.OverlayValues[217] = d217
						ps223.OverlayValues[219] = d219
						ps223.OverlayValues[220] = d220
						ps223.OverlayValues[221] = d221
						return bbs[12].RenderPS(ps223)
					}
					if !ps.General {
						ps.General = true
						return bbs[11].RenderPS(ps)
					}
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d221.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl27)
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl13)
					ps224 := PhiState{General: true}
					ps224.OverlayValues = make([]JITValueDesc, 222)
					ps224.OverlayValues[1] = d1
					ps224.OverlayValues[2] = d2
					ps224.OverlayValues[3] = d3
					ps224.OverlayValues[4] = d4
					ps224.OverlayValues[5] = d5
					ps224.OverlayValues[16] = d16
					ps224.OverlayValues[17] = d17
					ps224.OverlayValues[18] = d18
					ps224.OverlayValues[19] = d19
					ps224.OverlayValues[21] = d21
					ps224.OverlayValues[22] = d22
					ps224.OverlayValues[23] = d23
					ps224.OverlayValues[24] = d24
					ps224.OverlayValues[43] = d43
					ps224.OverlayValues[44] = d44
					ps224.OverlayValues[45] = d45
					ps224.OverlayValues[46] = d46
					ps224.OverlayValues[69] = d69
					ps224.OverlayValues[70] = d70
					ps224.OverlayValues[71] = d71
					ps224.OverlayValues[72] = d72
					ps224.OverlayValues[99] = d99
					ps224.OverlayValues[100] = d100
					ps224.OverlayValues[101] = d101
					ps224.OverlayValues[102] = d102
					ps224.OverlayValues[104] = d104
					ps224.OverlayValues[105] = d105
					ps224.OverlayValues[106] = d106
					ps224.OverlayValues[107] = d107
					ps224.OverlayValues[142] = d142
					ps224.OverlayValues[143] = d143
					ps224.OverlayValues[144] = d144
					ps224.OverlayValues[145] = d145
					ps224.OverlayValues[146] = d146
					ps224.OverlayValues[186] = d186
					ps224.OverlayValues[187] = d187
					ps224.OverlayValues[188] = d188
					ps224.OverlayValues[189] = d189
					ps224.OverlayValues[190] = d190
					ps224.OverlayValues[191] = d191
					ps224.OverlayValues[192] = d192
					ps224.OverlayValues[193] = d193
					ps224.OverlayValues[194] = d194
					ps224.OverlayValues[195] = d195
					ps224.OverlayValues[196] = d196
					ps224.OverlayValues[197] = d197
					ps224.OverlayValues[198] = d198
					ps224.OverlayValues[199] = d199
					ps224.OverlayValues[200] = d200
					ps224.OverlayValues[201] = d201
					ps224.OverlayValues[202] = d202
					ps224.OverlayValues[203] = d203
					ps224.OverlayValues[204] = d204
					ps224.OverlayValues[205] = d205
					ps224.OverlayValues[206] = d206
					ps224.OverlayValues[207] = d207
					ps224.OverlayValues[208] = d208
					ps224.OverlayValues[209] = d209
					ps224.OverlayValues[210] = d210
					ps224.OverlayValues[211] = d211
					ps224.OverlayValues[212] = d212
					ps224.OverlayValues[214] = d214
					ps224.OverlayValues[215] = d215
					ps224.OverlayValues[216] = d216
					ps224.OverlayValues[217] = d217
					ps224.OverlayValues[219] = d219
					ps224.OverlayValues[220] = d220
					ps224.OverlayValues[221] = d221
					ps225 := PhiState{General: true}
					ps225.OverlayValues = make([]JITValueDesc, 222)
					ps225.OverlayValues[1] = d1
					ps225.OverlayValues[2] = d2
					ps225.OverlayValues[3] = d3
					ps225.OverlayValues[4] = d4
					ps225.OverlayValues[5] = d5
					ps225.OverlayValues[16] = d16
					ps225.OverlayValues[17] = d17
					ps225.OverlayValues[18] = d18
					ps225.OverlayValues[19] = d19
					ps225.OverlayValues[21] = d21
					ps225.OverlayValues[22] = d22
					ps225.OverlayValues[23] = d23
					ps225.OverlayValues[24] = d24
					ps225.OverlayValues[43] = d43
					ps225.OverlayValues[44] = d44
					ps225.OverlayValues[45] = d45
					ps225.OverlayValues[46] = d46
					ps225.OverlayValues[69] = d69
					ps225.OverlayValues[70] = d70
					ps225.OverlayValues[71] = d71
					ps225.OverlayValues[72] = d72
					ps225.OverlayValues[99] = d99
					ps225.OverlayValues[100] = d100
					ps225.OverlayValues[101] = d101
					ps225.OverlayValues[102] = d102
					ps225.OverlayValues[104] = d104
					ps225.OverlayValues[105] = d105
					ps225.OverlayValues[106] = d106
					ps225.OverlayValues[107] = d107
					ps225.OverlayValues[142] = d142
					ps225.OverlayValues[143] = d143
					ps225.OverlayValues[144] = d144
					ps225.OverlayValues[145] = d145
					ps225.OverlayValues[146] = d146
					ps225.OverlayValues[186] = d186
					ps225.OverlayValues[187] = d187
					ps225.OverlayValues[188] = d188
					ps225.OverlayValues[189] = d189
					ps225.OverlayValues[190] = d190
					ps225.OverlayValues[191] = d191
					ps225.OverlayValues[192] = d192
					ps225.OverlayValues[193] = d193
					ps225.OverlayValues[194] = d194
					ps225.OverlayValues[195] = d195
					ps225.OverlayValues[196] = d196
					ps225.OverlayValues[197] = d197
					ps225.OverlayValues[198] = d198
					ps225.OverlayValues[199] = d199
					ps225.OverlayValues[200] = d200
					ps225.OverlayValues[201] = d201
					ps225.OverlayValues[202] = d202
					ps225.OverlayValues[203] = d203
					ps225.OverlayValues[204] = d204
					ps225.OverlayValues[205] = d205
					ps225.OverlayValues[206] = d206
					ps225.OverlayValues[207] = d207
					ps225.OverlayValues[208] = d208
					ps225.OverlayValues[209] = d209
					ps225.OverlayValues[210] = d210
					ps225.OverlayValues[211] = d211
					ps225.OverlayValues[212] = d212
					ps225.OverlayValues[214] = d214
					ps225.OverlayValues[215] = d215
					ps225.OverlayValues[216] = d216
					ps225.OverlayValues[217] = d217
					ps225.OverlayValues[219] = d219
					ps225.OverlayValues[220] = d220
					ps225.OverlayValues[221] = d221
					snap226 := d1
					snap227 := d2
					snap228 := d3
					snap229 := d4
					snap230 := d5
					snap231 := d16
					snap232 := d17
					snap233 := d18
					snap234 := d19
					snap235 := d21
					snap236 := d22
					snap237 := d23
					snap238 := d24
					snap239 := d43
					snap240 := d44
					snap241 := d45
					snap242 := d46
					snap243 := d69
					snap244 := d70
					snap245 := d71
					snap246 := d72
					snap247 := d99
					snap248 := d100
					snap249 := d101
					snap250 := d102
					snap251 := d104
					snap252 := d105
					snap253 := d106
					snap254 := d107
					snap255 := d142
					snap256 := d143
					snap257 := d144
					snap258 := d145
					snap259 := d146
					snap260 := d186
					snap261 := d187
					snap262 := d188
					snap263 := d189
					snap264 := d190
					snap265 := d191
					snap266 := d192
					snap267 := d193
					snap268 := d194
					snap269 := d195
					snap270 := d196
					snap271 := d197
					snap272 := d198
					snap273 := d199
					snap274 := d200
					snap275 := d201
					snap276 := d202
					snap277 := d203
					snap278 := d204
					snap279 := d205
					snap280 := d206
					snap281 := d207
					snap282 := d208
					snap283 := d209
					snap284 := d210
					snap285 := d211
					snap286 := d212
					snap287 := d214
					snap288 := d215
					snap289 := d216
					snap290 := d217
					snap291 := d219
					snap292 := d220
					snap293 := d221
					alloc294 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps225)
					}
					ctx.RestoreAllocState(alloc294)
					d1 = snap226
					d2 = snap227
					d3 = snap228
					d4 = snap229
					d5 = snap230
					d16 = snap231
					d17 = snap232
					d18 = snap233
					d19 = snap234
					d21 = snap235
					d22 = snap236
					d23 = snap237
					d24 = snap238
					d43 = snap239
					d44 = snap240
					d45 = snap241
					d46 = snap242
					d69 = snap243
					d70 = snap244
					d71 = snap245
					d72 = snap246
					d99 = snap247
					d100 = snap248
					d101 = snap249
					d102 = snap250
					d104 = snap251
					d105 = snap252
					d106 = snap253
					d107 = snap254
					d142 = snap255
					d143 = snap256
					d144 = snap257
					d145 = snap258
					d146 = snap259
					d186 = snap260
					d187 = snap261
					d188 = snap262
					d189 = snap263
					d190 = snap264
					d191 = snap265
					d192 = snap266
					d193 = snap267
					d194 = snap268
					d195 = snap269
					d196 = snap270
					d197 = snap271
					d198 = snap272
					d199 = snap273
					d200 = snap274
					d201 = snap275
					d202 = snap276
					d203 = snap277
					d204 = snap278
					d205 = snap279
					d206 = snap280
					d207 = snap281
					d208 = snap282
					d209 = snap283
					d210 = snap284
					d211 = snap285
					d212 = snap286
					d214 = snap287
					d215 = snap288
					d216 = snap289
					d217 = snap290
					d219 = snap291
					d220 = snap292
					d221 = snap293
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps224)
					}
					return result
					ctx.FreeDesc(&d220)
					return result
				}
				bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[12].VisitCount >= 0 {
							ps.General = true
							return bbs[12].RenderPS(ps)
						}
					}
					bbs[12].VisitCount++
					if ps.General {
						if bbs[12].Rendered {
							ctx.EmitJmp(lbl13)
							return result
						}
						bbs[12].Rendered = true
						bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_12 = bbs[12].Address
						ctx.MarkLabel(lbl13)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					ctx.ReclaimUntrackedRegs()
					d295 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d295)
					if d295.Loc == LocRegPair || d295.Loc == LocStackPair || d295.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d295, &result)
						result.Type = d295.Type
					} else {
						switch d295.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d295)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d295)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d295)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d295, &result)
							result.Type = d295.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[13].VisitCount >= 0 {
							ps.General = true
							return bbs[13].RenderPS(ps)
						}
					}
					bbs[13].VisitCount++
					if ps.General {
						if bbs[13].Rendered {
							ctx.EmitJmp(lbl14)
							return result
						}
						bbs[13].Rendered = true
						bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_13 = bbs[13].Address
						ctx.MarkLabel(lbl14)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d219)
					ctx.EnsureDesc(&d219)
					if d219.Loc == LocRegPair || d219.Loc == LocStackPair || d219.Loc == LocRegTriple || d219.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d296 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d296.Loc == LocRegPair || d296.Loc == LocStackPair || d296.Loc == LocRegTriple || d296.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d219)
					ctx.SyncDesc(&d296)
					d297 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d219, d296}, 3)
					d297.NoHeapPointer = false
					ctx.BindReg(d297.Reg, &d297)
					ctx.BindReg(d297.Reg2, &d297)
					ctx.BindReg(d297.Reg3, &d297)
					ctx.FreeDesc(&d296)
					ctx.StabilizeDescForControlFlow(&d297)
					ctx.FreeDesc(&d219)
					if ps.General {
						ctx.SyncDesc(&d297)
						if d297.Loc == LocReg {
							ctx.ProtectReg(d297.Reg)
						} else if d297.Loc == LocRegPair {
							ctx.ProtectReg(d297.Reg)
							ctx.ProtectReg(d297.Reg2)
						}
						d298 = d297
						if d298.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d298)
						if d298.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d298, int32(bbs[9].PhiBase)+int32(0), 2)
						} else if d298.Loc == LocInputPair {
							ctx.EnsureDesc(&d298)
							ctx.EmitStoreScmerToStack(d298, int32(bbs[9].PhiBase)+int32(0))
						} else if d298.Loc == LocRegPair || d298.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d298, int32(bbs[9].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d298)
							ctx.EmitStoreToStack(d298, int32(bbs[9].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
						}
						if d297.Loc == LocReg {
							ctx.UnprotectReg(d297.Reg)
						} else if d297.Loc == LocRegPair {
							ctx.UnprotectReg(d297.Reg)
							ctx.UnprotectReg(d297.Reg2)
						}
					}
					ps299 := PhiState{General: ps.General}
					ps299.OverlayValues = make([]JITValueDesc, 299)
					ps299.OverlayValues[1] = d1
					ps299.OverlayValues[2] = d2
					ps299.OverlayValues[3] = d3
					ps299.OverlayValues[4] = d4
					ps299.OverlayValues[5] = d5
					ps299.OverlayValues[16] = d16
					ps299.OverlayValues[17] = d17
					ps299.OverlayValues[18] = d18
					ps299.OverlayValues[19] = d19
					ps299.OverlayValues[21] = d21
					ps299.OverlayValues[22] = d22
					ps299.OverlayValues[23] = d23
					ps299.OverlayValues[24] = d24
					ps299.OverlayValues[43] = d43
					ps299.OverlayValues[44] = d44
					ps299.OverlayValues[45] = d45
					ps299.OverlayValues[46] = d46
					ps299.OverlayValues[69] = d69
					ps299.OverlayValues[70] = d70
					ps299.OverlayValues[71] = d71
					ps299.OverlayValues[72] = d72
					ps299.OverlayValues[99] = d99
					ps299.OverlayValues[100] = d100
					ps299.OverlayValues[101] = d101
					ps299.OverlayValues[102] = d102
					ps299.OverlayValues[104] = d104
					ps299.OverlayValues[105] = d105
					ps299.OverlayValues[106] = d106
					ps299.OverlayValues[107] = d107
					ps299.OverlayValues[142] = d142
					ps299.OverlayValues[143] = d143
					ps299.OverlayValues[144] = d144
					ps299.OverlayValues[145] = d145
					ps299.OverlayValues[146] = d146
					ps299.OverlayValues[186] = d186
					ps299.OverlayValues[187] = d187
					ps299.OverlayValues[188] = d188
					ps299.OverlayValues[189] = d189
					ps299.OverlayValues[190] = d190
					ps299.OverlayValues[191] = d191
					ps299.OverlayValues[192] = d192
					ps299.OverlayValues[193] = d193
					ps299.OverlayValues[194] = d194
					ps299.OverlayValues[195] = d195
					ps299.OverlayValues[196] = d196
					ps299.OverlayValues[197] = d197
					ps299.OverlayValues[198] = d198
					ps299.OverlayValues[199] = d199
					ps299.OverlayValues[200] = d200
					ps299.OverlayValues[201] = d201
					ps299.OverlayValues[202] = d202
					ps299.OverlayValues[203] = d203
					ps299.OverlayValues[204] = d204
					ps299.OverlayValues[205] = d205
					ps299.OverlayValues[206] = d206
					ps299.OverlayValues[207] = d207
					ps299.OverlayValues[208] = d208
					ps299.OverlayValues[209] = d209
					ps299.OverlayValues[210] = d210
					ps299.OverlayValues[211] = d211
					ps299.OverlayValues[212] = d212
					ps299.OverlayValues[214] = d214
					ps299.OverlayValues[215] = d215
					ps299.OverlayValues[216] = d216
					ps299.OverlayValues[217] = d217
					ps299.OverlayValues[219] = d219
					ps299.OverlayValues[220] = d220
					ps299.OverlayValues[221] = d221
					ps299.OverlayValues[295] = d295
					ps299.OverlayValues[296] = d296
					ps299.OverlayValues[297] = d297
					ps299.OverlayValues[298] = d298
					ps299.PhiValues = make([]JITValueDesc, 1)
					d300 = d297
					ps299.PhiValues[0] = d300
					if ps299.General && bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					return bbs[9].RenderPS(ps299)
					return result
				}
				ps301 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps301)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d26 JITValueDesc
				_ = d26
				var d29 JITValueDesc
				_ = d29
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d58 JITValueDesc
				_ = d58
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d65 JITValueDesc
				_ = d65
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
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
				var d106 JITValueDesc
				_ = d106
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
				var d154 JITValueDesc
				_ = d154
				var d155 JITValueDesc
				_ = d155
				var d156 JITValueDesc
				_ = d156
				var d157 JITValueDesc
				_ = d157
				var d158 JITValueDesc
				_ = d158
				var d159 JITValueDesc
				_ = d159
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				var bbs [10]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiCount = uint16(1)
				bbs[6].PhiBase = int32(phiBase0) + int32(16)
				bbs[6].PhiCount = uint16(1)
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
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
							if ps.General {
							}
							ps7 := PhiState{General: ps.General}
							ps7.OverlayValues = make([]JITValueDesc, 7)
							ps7.OverlayValues[1] = d1
							ps7.OverlayValues[2] = d2
							ps7.OverlayValues[3] = d3
							ps7.OverlayValues[4] = d4
							ps7.OverlayValues[5] = d5
							ps7.OverlayValues[6] = d6
							return bbs[1].RenderPS(ps7)
						}
						if ps.General {
						}
						ps8 := PhiState{General: ps.General}
						ps8.OverlayValues = make([]JITValueDesc, 7)
						ps8.OverlayValues[1] = d1
						ps8.OverlayValues[2] = d2
						ps8.OverlayValues[3] = d3
						ps8.OverlayValues[4] = d4
						ps8.OverlayValues[5] = d5
						ps8.OverlayValues[6] = d6
						return bbs[2].RenderPS(ps8)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl3)
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 7)
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					ps9.OverlayValues[6] = d6
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 7)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					snap11 := d1
					snap12 := d2
					snap13 := d3
					snap14 := d4
					snap15 := d5
					snap16 := d6
					alloc17 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc17)
					d1 = snap11
					d2 = snap12
					d3 = snap13
					d4 = snap14
					d5 = snap15
					d6 = snap16
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps9)
					}
					return result
					ctx.FreeDesc(&d4)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					ctx.ReclaimUntrackedRegs()
					d18 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d18, &result)
						result.Type = d18.Type
					} else {
						switch d18.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d18)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d18)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d18)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d18, &result)
							result.Type = d18.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					ctx.ReclaimUntrackedRegs()
					d19 = args[0]
					d19.ID = 0
					var d20 JITValueDesc
					if d19.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int())}
					} else if d19.Type == tagInt && d19.Loc == LocRegPair {
						ctx.FreeReg(d19.Reg)
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2}
						ctx.BindReg(d19.Reg2, &d20)
						ctx.BindReg(d19.Reg2, &d20)
					} else if d19.Type == tagInt && d19.Loc == LocReg {
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg}
						ctx.BindReg(d19.Reg, &d20)
						ctx.BindReg(d19.Reg, &d20)
					} else {
						d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d19}, 1)
						d20.Type = tagInt
						ctx.BindReg(d20.Reg, &d20)
					}
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d19)
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d21)
					var d22 JITValueDesc
					if d21.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d21.Reg, 2)
						ctx.EmitSetcc(r0, CondSignedGreater)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d22)
					}
					ctx.FreeDesc(&d21)
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							if ps.General {
							}
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[2] = d2
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[4] = d4
							ps24.OverlayValues[5] = d5
							ps24.OverlayValues[6] = d6
							ps24.OverlayValues[18] = d18
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[3].RenderPS(ps24)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[2] = d2
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[4] = d4
						ps25.OverlayValues[5] = d5
						ps25.OverlayValues[6] = d6
						ps25.OverlayValues[18] = d18
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						ps25.PhiValues = make([]JITValueDesc, 1)
						d26 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}
						ps25.PhiValues[0] = d26
						return bbs[4].RenderPS(ps25)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl14)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 27)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[6] = d6
					ps27.OverlayValues[18] = d18
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					ps27.OverlayValues[26] = d26
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 27)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[6] = d6
					ps28.OverlayValues[18] = d18
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[26] = d26
					ps28.PhiValues = make([]JITValueDesc, 1)
					d29 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}
					ps28.PhiValues[0] = d29
					snap30 := d1
					snap31 := d2
					snap32 := d3
					snap33 := d4
					snap34 := d5
					snap35 := d6
					snap36 := d18
					snap37 := d19
					snap38 := d20
					snap39 := d21
					snap40 := d22
					snap41 := d23
					snap42 := d26
					snap43 := d29
					alloc44 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps28)
					}
					ctx.RestoreAllocState(alloc44)
					d1 = snap30
					d2 = snap31
					d3 = snap32
					d4 = snap33
					d5 = snap34
					d6 = snap35
					d18 = snap36
					d19 = snap37
					d20 = snap38
					d21 = snap39
					d22 = snap40
					d23 = snap41
					d26 = snap42
					d29 = snap43
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps27)
					}
					return result
					ctx.FreeDesc(&d22)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					ctx.ReclaimUntrackedRegs()
					d45 = args[2]
					d45.ID = 0
					d47 = d45
					ctx.SyncDesc(&d47)
					if d47.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d47.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d47.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d47 = tmpScalar
					}
					d47 = JITPrepareScmerGoArg(ctx, d47)
					if d47.Loc != LocRegPair && d47.Loc != LocStackPair && d47.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d46 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d47}, 2)
					ctx.StabilizeDescForControlFlow(&d46)
					ctx.FreeDesc(&d45)
					if ps.General {
						ctx.SyncDesc(&d46)
						if d46.Loc == LocReg {
							ctx.ProtectReg(d46.Reg)
						} else if d46.Loc == LocRegPair {
							ctx.ProtectReg(d46.Reg)
							ctx.ProtectReg(d46.Reg2)
						}
						d48 = d46
						if d48.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d48)
						if d48.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d48, int32(bbs[4].PhiBase)+int32(0), 2)
						} else if d48.Loc == LocInputPair {
							ctx.EnsureDesc(&d48)
							ctx.EmitStoreScmerToStack(d48, int32(bbs[4].PhiBase)+int32(0))
						} else if d48.Loc == LocRegPair || d48.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d48, int32(bbs[4].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d48)
							ctx.EmitStoreToStack(d48, int32(bbs[4].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
						}
						if d46.Loc == LocReg {
							ctx.UnprotectReg(d46.Reg)
						} else if d46.Loc == LocRegPair {
							ctx.UnprotectReg(d46.Reg)
							ctx.UnprotectReg(d46.Reg2)
						}
					}
					ps49 := PhiState{General: ps.General}
					ps49.OverlayValues = make([]JITValueDesc, 49)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[6] = d6
					ps49.OverlayValues[18] = d18
					ps49.OverlayValues[19] = d19
					ps49.OverlayValues[20] = d20
					ps49.OverlayValues[21] = d21
					ps49.OverlayValues[22] = d22
					ps49.OverlayValues[23] = d23
					ps49.OverlayValues[26] = d26
					ps49.OverlayValues[29] = d29
					ps49.OverlayValues[45] = d45
					ps49.OverlayValues[46] = d46
					ps49.OverlayValues[47] = d47
					ps49.OverlayValues[48] = d48
					ps49.PhiValues = make([]JITValueDesc, 1)
					d50 = d46
					ps49.PhiValues[0] = d50
					if ps49.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps49)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d51 := ps.PhiValues[0]
							ctx.EnsureDesc(&d51)
							ctx.EmitStoreScmerToStack(d51, int32(bbs[4].PhiBase)+int32(0))
						}
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
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
					callResults52 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d1}, []uint8{1, 2}, []uint8{1, 3})
					d53 = callResults52[0]
					_ = d53
					d54 = callResults52[1]
					_ = d54
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d53)
					ctx.EnsureDesc(&d54)
					var d55 JITValueDesc
					if d54.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d54.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d54)
						if d54.Loc != LocReg && d54.Loc != LocRegPair && d54.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d54.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d55 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d55)
					}
					ctx.FreeDesc(&d54)
					d56 = d55
					ctx.EnsureDesc(&d56)
					if d56.Loc != LocImm && d56.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d56.Loc == LocImm {
						if d56.Imm.Bool() {
							if ps.General {
							}
							ps57 := PhiState{General: ps.General}
							ps57.OverlayValues = make([]JITValueDesc, 57)
							ps57.OverlayValues[1] = d1
							ps57.OverlayValues[2] = d2
							ps57.OverlayValues[3] = d3
							ps57.OverlayValues[4] = d4
							ps57.OverlayValues[5] = d5
							ps57.OverlayValues[6] = d6
							ps57.OverlayValues[18] = d18
							ps57.OverlayValues[19] = d19
							ps57.OverlayValues[20] = d20
							ps57.OverlayValues[21] = d21
							ps57.OverlayValues[22] = d22
							ps57.OverlayValues[23] = d23
							ps57.OverlayValues[26] = d26
							ps57.OverlayValues[29] = d29
							ps57.OverlayValues[45] = d45
							ps57.OverlayValues[46] = d46
							ps57.OverlayValues[47] = d47
							ps57.OverlayValues[48] = d48
							ps57.OverlayValues[50] = d50
							ps57.OverlayValues[51] = d51
							ps57.OverlayValues[53] = d53
							ps57.OverlayValues[54] = d54
							ps57.OverlayValues[55] = d55
							ps57.OverlayValues[56] = d56
							return bbs[5].RenderPS(ps57)
						}
						if ps.General {
							ctx.SyncDesc(&d53)
							if d53.Loc == LocReg {
								ctx.ProtectReg(d53.Reg)
							} else if d53.Loc == LocRegPair {
								ctx.ProtectReg(d53.Reg)
								ctx.ProtectReg(d53.Reg2)
							}
							d58 = d53
							if d58.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d58)
							ctx.EmitStoreToStack(d58, int32(bbs[6].PhiBase)+int32(0))
							if d53.Loc == LocReg {
								ctx.UnprotectReg(d53.Reg)
							} else if d53.Loc == LocRegPair {
								ctx.UnprotectReg(d53.Reg)
								ctx.UnprotectReg(d53.Reg2)
							}
						}
						ps59 := PhiState{General: ps.General}
						ps59.OverlayValues = make([]JITValueDesc, 59)
						ps59.OverlayValues[1] = d1
						ps59.OverlayValues[2] = d2
						ps59.OverlayValues[3] = d3
						ps59.OverlayValues[4] = d4
						ps59.OverlayValues[5] = d5
						ps59.OverlayValues[6] = d6
						ps59.OverlayValues[18] = d18
						ps59.OverlayValues[19] = d19
						ps59.OverlayValues[20] = d20
						ps59.OverlayValues[21] = d21
						ps59.OverlayValues[22] = d22
						ps59.OverlayValues[23] = d23
						ps59.OverlayValues[26] = d26
						ps59.OverlayValues[29] = d29
						ps59.OverlayValues[45] = d45
						ps59.OverlayValues[46] = d46
						ps59.OverlayValues[47] = d47
						ps59.OverlayValues[48] = d48
						ps59.OverlayValues[50] = d50
						ps59.OverlayValues[51] = d51
						ps59.OverlayValues[53] = d53
						ps59.OverlayValues[54] = d54
						ps59.OverlayValues[55] = d55
						ps59.OverlayValues[56] = d56
						ps59.OverlayValues[58] = d58
						ps59.PhiValues = make([]JITValueDesc, 1)
						d60 = d53
						ps59.PhiValues[0] = d60
						return bbs[6].RenderPS(ps59)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d61 := ps.PhiValues[0]
							ctx.EnsureDesc(&d61)
							ctx.EmitStoreScmerToStack(d61, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d56.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl16)
					ctx.SyncDesc(&d53)
					if d53.Loc == LocReg {
						ctx.ProtectReg(d53.Reg)
					} else if d53.Loc == LocRegPair {
						ctx.ProtectReg(d53.Reg)
						ctx.ProtectReg(d53.Reg2)
					}
					d62 = d53
					if d62.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d62)
					ctx.EmitStoreToStack(d62, int32(bbs[6].PhiBase)+int32(0))
					if d53.Loc == LocReg {
						ctx.UnprotectReg(d53.Reg)
					} else if d53.Loc == LocRegPair {
						ctx.UnprotectReg(d53.Reg)
						ctx.UnprotectReg(d53.Reg2)
					}
					ctx.EmitJmp(lbl7)
					ps63 := PhiState{General: true}
					ps63.OverlayValues = make([]JITValueDesc, 63)
					ps63.OverlayValues[1] = d1
					ps63.OverlayValues[2] = d2
					ps63.OverlayValues[3] = d3
					ps63.OverlayValues[4] = d4
					ps63.OverlayValues[5] = d5
					ps63.OverlayValues[6] = d6
					ps63.OverlayValues[18] = d18
					ps63.OverlayValues[19] = d19
					ps63.OverlayValues[20] = d20
					ps63.OverlayValues[21] = d21
					ps63.OverlayValues[22] = d22
					ps63.OverlayValues[23] = d23
					ps63.OverlayValues[26] = d26
					ps63.OverlayValues[29] = d29
					ps63.OverlayValues[45] = d45
					ps63.OverlayValues[46] = d46
					ps63.OverlayValues[47] = d47
					ps63.OverlayValues[48] = d48
					ps63.OverlayValues[50] = d50
					ps63.OverlayValues[51] = d51
					ps63.OverlayValues[53] = d53
					ps63.OverlayValues[54] = d54
					ps63.OverlayValues[55] = d55
					ps63.OverlayValues[56] = d56
					ps63.OverlayValues[58] = d58
					ps63.OverlayValues[60] = d60
					ps63.OverlayValues[61] = d61
					ps63.OverlayValues[62] = d62
					ps64 := PhiState{General: true}
					ps64.OverlayValues = make([]JITValueDesc, 63)
					ps64.OverlayValues[1] = d1
					ps64.OverlayValues[2] = d2
					ps64.OverlayValues[3] = d3
					ps64.OverlayValues[4] = d4
					ps64.OverlayValues[5] = d5
					ps64.OverlayValues[6] = d6
					ps64.OverlayValues[18] = d18
					ps64.OverlayValues[19] = d19
					ps64.OverlayValues[20] = d20
					ps64.OverlayValues[21] = d21
					ps64.OverlayValues[22] = d22
					ps64.OverlayValues[23] = d23
					ps64.OverlayValues[26] = d26
					ps64.OverlayValues[29] = d29
					ps64.OverlayValues[45] = d45
					ps64.OverlayValues[46] = d46
					ps64.OverlayValues[47] = d47
					ps64.OverlayValues[48] = d48
					ps64.OverlayValues[50] = d50
					ps64.OverlayValues[51] = d51
					ps64.OverlayValues[53] = d53
					ps64.OverlayValues[54] = d54
					ps64.OverlayValues[55] = d55
					ps64.OverlayValues[56] = d56
					ps64.OverlayValues[58] = d58
					ps64.OverlayValues[60] = d60
					ps64.OverlayValues[61] = d61
					ps64.OverlayValues[62] = d62
					ps64.PhiValues = make([]JITValueDesc, 1)
					d65 = d53
					ps64.PhiValues[0] = d65
					snap66 := d1
					snap67 := d2
					snap68 := d3
					snap69 := d4
					snap70 := d5
					snap71 := d6
					snap72 := d18
					snap73 := d19
					snap74 := d20
					snap75 := d21
					snap76 := d22
					snap77 := d23
					snap78 := d26
					snap79 := d29
					snap80 := d45
					snap81 := d46
					snap82 := d47
					snap83 := d48
					snap84 := d50
					snap85 := d51
					snap86 := d53
					snap87 := d54
					snap88 := d55
					snap89 := d56
					snap90 := d58
					snap91 := d60
					snap92 := d61
					snap93 := d62
					snap94 := d65
					alloc95 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps64)
					}
					ctx.RestoreAllocState(alloc95)
					d1 = snap66
					d2 = snap67
					d3 = snap68
					d4 = snap69
					d5 = snap70
					d6 = snap71
					d18 = snap72
					d19 = snap73
					d20 = snap74
					d21 = snap75
					d22 = snap76
					d23 = snap77
					d26 = snap78
					d29 = snap79
					d45 = snap80
					d46 = snap81
					d47 = snap82
					d48 = snap83
					d50 = snap84
					d51 = snap85
					d53 = snap86
					d54 = snap87
					d55 = snap88
					d56 = snap89
					d58 = snap90
					d60 = snap91
					d61 = snap92
					d62 = snap93
					d65 = snap94
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps63)
					}
					return result
					ctx.FreeDesc(&d55)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					ctx.ReclaimUntrackedRegs()
					d96 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.StabilizeDescForControlFlow(&d96)
					if ps.General {
						ctx.SyncDesc(&d96)
						if d96.Loc == LocReg {
							ctx.ProtectReg(d96.Reg)
						} else if d96.Loc == LocRegPair {
							ctx.ProtectReg(d96.Reg)
							ctx.ProtectReg(d96.Reg2)
						}
						d97 = d96
						if d97.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d97)
						ctx.EmitStoreToStack(d97, int32(bbs[6].PhiBase)+int32(0))
						if d96.Loc == LocReg {
							ctx.UnprotectReg(d96.Reg)
						} else if d96.Loc == LocRegPair {
							ctx.UnprotectReg(d96.Reg)
							ctx.UnprotectReg(d96.Reg2)
						}
					}
					ps98 := PhiState{General: ps.General}
					ps98.OverlayValues = make([]JITValueDesc, 98)
					ps98.OverlayValues[1] = d1
					ps98.OverlayValues[2] = d2
					ps98.OverlayValues[3] = d3
					ps98.OverlayValues[4] = d4
					ps98.OverlayValues[5] = d5
					ps98.OverlayValues[6] = d6
					ps98.OverlayValues[18] = d18
					ps98.OverlayValues[19] = d19
					ps98.OverlayValues[20] = d20
					ps98.OverlayValues[21] = d21
					ps98.OverlayValues[22] = d22
					ps98.OverlayValues[23] = d23
					ps98.OverlayValues[26] = d26
					ps98.OverlayValues[29] = d29
					ps98.OverlayValues[45] = d45
					ps98.OverlayValues[46] = d46
					ps98.OverlayValues[47] = d47
					ps98.OverlayValues[48] = d48
					ps98.OverlayValues[50] = d50
					ps98.OverlayValues[51] = d51
					ps98.OverlayValues[53] = d53
					ps98.OverlayValues[54] = d54
					ps98.OverlayValues[55] = d55
					ps98.OverlayValues[56] = d56
					ps98.OverlayValues[58] = d58
					ps98.OverlayValues[60] = d60
					ps98.OverlayValues[61] = d61
					ps98.OverlayValues[62] = d62
					ps98.OverlayValues[65] = d65
					ps98.OverlayValues[96] = d96
					ps98.OverlayValues[97] = d97
					ps98.PhiValues = make([]JITValueDesc, 1)
					d99 = d96
					ps98.PhiValues[0] = d99
					if ps98.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps98)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d100 := ps.PhiValues[0]
							ctx.EnsureDesc(&d100)
							ctx.EmitStoreToStack(d100, int32(bbs[6].PhiBase)+int32(0))
						}
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					d101 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d101)
					var d102 JITValueDesc
					if d101.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d101.Imm.Int() > 1)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d101.Reg, 1)
						ctx.EmitSetcc(r2, CondSignedGreater)
						d102 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d102)
					}
					ctx.FreeDesc(&d101)
					d103 = d102
					ctx.EnsureDesc(&d103)
					if d103.Loc != LocImm && d103.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d103.Loc == LocImm {
						if d103.Imm.Bool() {
							if ps.General {
							}
							ps104 := PhiState{General: ps.General}
							ps104.OverlayValues = make([]JITValueDesc, 104)
							ps104.OverlayValues[1] = d1
							ps104.OverlayValues[2] = d2
							ps104.OverlayValues[3] = d3
							ps104.OverlayValues[4] = d4
							ps104.OverlayValues[5] = d5
							ps104.OverlayValues[6] = d6
							ps104.OverlayValues[18] = d18
							ps104.OverlayValues[19] = d19
							ps104.OverlayValues[20] = d20
							ps104.OverlayValues[21] = d21
							ps104.OverlayValues[22] = d22
							ps104.OverlayValues[23] = d23
							ps104.OverlayValues[26] = d26
							ps104.OverlayValues[29] = d29
							ps104.OverlayValues[45] = d45
							ps104.OverlayValues[46] = d46
							ps104.OverlayValues[47] = d47
							ps104.OverlayValues[48] = d48
							ps104.OverlayValues[50] = d50
							ps104.OverlayValues[51] = d51
							ps104.OverlayValues[53] = d53
							ps104.OverlayValues[54] = d54
							ps104.OverlayValues[55] = d55
							ps104.OverlayValues[56] = d56
							ps104.OverlayValues[58] = d58
							ps104.OverlayValues[60] = d60
							ps104.OverlayValues[61] = d61
							ps104.OverlayValues[62] = d62
							ps104.OverlayValues[65] = d65
							ps104.OverlayValues[96] = d96
							ps104.OverlayValues[97] = d97
							ps104.OverlayValues[99] = d99
							ps104.OverlayValues[100] = d100
							ps104.OverlayValues[101] = d101
							ps104.OverlayValues[102] = d102
							ps104.OverlayValues[103] = d103
							return bbs[9].RenderPS(ps104)
						}
						if ps.General {
						}
						ps105 := PhiState{General: ps.General}
						ps105.OverlayValues = make([]JITValueDesc, 104)
						ps105.OverlayValues[1] = d1
						ps105.OverlayValues[2] = d2
						ps105.OverlayValues[3] = d3
						ps105.OverlayValues[4] = d4
						ps105.OverlayValues[5] = d5
						ps105.OverlayValues[6] = d6
						ps105.OverlayValues[18] = d18
						ps105.OverlayValues[19] = d19
						ps105.OverlayValues[20] = d20
						ps105.OverlayValues[21] = d21
						ps105.OverlayValues[22] = d22
						ps105.OverlayValues[23] = d23
						ps105.OverlayValues[26] = d26
						ps105.OverlayValues[29] = d29
						ps105.OverlayValues[45] = d45
						ps105.OverlayValues[46] = d46
						ps105.OverlayValues[47] = d47
						ps105.OverlayValues[48] = d48
						ps105.OverlayValues[50] = d50
						ps105.OverlayValues[51] = d51
						ps105.OverlayValues[53] = d53
						ps105.OverlayValues[54] = d54
						ps105.OverlayValues[55] = d55
						ps105.OverlayValues[56] = d56
						ps105.OverlayValues[58] = d58
						ps105.OverlayValues[60] = d60
						ps105.OverlayValues[61] = d61
						ps105.OverlayValues[62] = d62
						ps105.OverlayValues[65] = d65
						ps105.OverlayValues[96] = d96
						ps105.OverlayValues[97] = d97
						ps105.OverlayValues[99] = d99
						ps105.OverlayValues[100] = d100
						ps105.OverlayValues[101] = d101
						ps105.OverlayValues[102] = d102
						ps105.OverlayValues[103] = d103
						return bbs[8].RenderPS(ps105)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d106 := ps.PhiValues[0]
							ctx.EnsureDesc(&d106)
							ctx.EmitStoreToStack(d106, int32(bbs[6].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d103.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl9)
					ps107 := PhiState{General: true}
					ps107.OverlayValues = make([]JITValueDesc, 107)
					ps107.OverlayValues[1] = d1
					ps107.OverlayValues[2] = d2
					ps107.OverlayValues[3] = d3
					ps107.OverlayValues[4] = d4
					ps107.OverlayValues[5] = d5
					ps107.OverlayValues[6] = d6
					ps107.OverlayValues[18] = d18
					ps107.OverlayValues[19] = d19
					ps107.OverlayValues[20] = d20
					ps107.OverlayValues[21] = d21
					ps107.OverlayValues[22] = d22
					ps107.OverlayValues[23] = d23
					ps107.OverlayValues[26] = d26
					ps107.OverlayValues[29] = d29
					ps107.OverlayValues[45] = d45
					ps107.OverlayValues[46] = d46
					ps107.OverlayValues[47] = d47
					ps107.OverlayValues[48] = d48
					ps107.OverlayValues[50] = d50
					ps107.OverlayValues[51] = d51
					ps107.OverlayValues[53] = d53
					ps107.OverlayValues[54] = d54
					ps107.OverlayValues[55] = d55
					ps107.OverlayValues[56] = d56
					ps107.OverlayValues[58] = d58
					ps107.OverlayValues[60] = d60
					ps107.OverlayValues[61] = d61
					ps107.OverlayValues[62] = d62
					ps107.OverlayValues[65] = d65
					ps107.OverlayValues[96] = d96
					ps107.OverlayValues[97] = d97
					ps107.OverlayValues[99] = d99
					ps107.OverlayValues[100] = d100
					ps107.OverlayValues[101] = d101
					ps107.OverlayValues[102] = d102
					ps107.OverlayValues[103] = d103
					ps107.OverlayValues[106] = d106
					ps108 := PhiState{General: true}
					ps108.OverlayValues = make([]JITValueDesc, 107)
					ps108.OverlayValues[1] = d1
					ps108.OverlayValues[2] = d2
					ps108.OverlayValues[3] = d3
					ps108.OverlayValues[4] = d4
					ps108.OverlayValues[5] = d5
					ps108.OverlayValues[6] = d6
					ps108.OverlayValues[18] = d18
					ps108.OverlayValues[19] = d19
					ps108.OverlayValues[20] = d20
					ps108.OverlayValues[21] = d21
					ps108.OverlayValues[22] = d22
					ps108.OverlayValues[23] = d23
					ps108.OverlayValues[26] = d26
					ps108.OverlayValues[29] = d29
					ps108.OverlayValues[45] = d45
					ps108.OverlayValues[46] = d46
					ps108.OverlayValues[47] = d47
					ps108.OverlayValues[48] = d48
					ps108.OverlayValues[50] = d50
					ps108.OverlayValues[51] = d51
					ps108.OverlayValues[53] = d53
					ps108.OverlayValues[54] = d54
					ps108.OverlayValues[55] = d55
					ps108.OverlayValues[56] = d56
					ps108.OverlayValues[58] = d58
					ps108.OverlayValues[60] = d60
					ps108.OverlayValues[61] = d61
					ps108.OverlayValues[62] = d62
					ps108.OverlayValues[65] = d65
					ps108.OverlayValues[96] = d96
					ps108.OverlayValues[97] = d97
					ps108.OverlayValues[99] = d99
					ps108.OverlayValues[100] = d100
					ps108.OverlayValues[101] = d101
					ps108.OverlayValues[102] = d102
					ps108.OverlayValues[103] = d103
					ps108.OverlayValues[106] = d106
					snap109 := d1
					snap110 := d2
					snap111 := d3
					snap112 := d4
					snap113 := d5
					snap114 := d6
					snap115 := d18
					snap116 := d19
					snap117 := d20
					snap118 := d21
					snap119 := d22
					snap120 := d23
					snap121 := d26
					snap122 := d29
					snap123 := d45
					snap124 := d46
					snap125 := d47
					snap126 := d48
					snap127 := d50
					snap128 := d51
					snap129 := d53
					snap130 := d54
					snap131 := d55
					snap132 := d56
					snap133 := d58
					snap134 := d60
					snap135 := d61
					snap136 := d62
					snap137 := d65
					snap138 := d96
					snap139 := d97
					snap140 := d99
					snap141 := d100
					snap142 := d101
					snap143 := d102
					snap144 := d103
					snap145 := d106
					alloc146 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps108)
					}
					ctx.RestoreAllocState(alloc146)
					d1 = snap109
					d2 = snap110
					d3 = snap111
					d4 = snap112
					d5 = snap113
					d6 = snap114
					d18 = snap115
					d19 = snap116
					d20 = snap117
					d21 = snap118
					d22 = snap119
					d23 = snap120
					d26 = snap121
					d29 = snap122
					d45 = snap123
					d46 = snap124
					d47 = snap125
					d48 = snap126
					d50 = snap127
					d51 = snap128
					d53 = snap129
					d54 = snap130
					d55 = snap131
					d56 = snap132
					d58 = snap133
					d60 = snap134
					d61 = snap135
					d62 = snap136
					d65 = snap137
					d96 = snap138
					d97 = snap139
					d99 = snap140
					d100 = snap141
					d101 = snap142
					d102 = snap143
					d103 = snap144
					d106 = snap145
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps107)
					}
					return result
					ctx.FreeDesc(&d102)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocRegPair || d20.Loc == LocStackPair || d20.Loc == LocRegTriple || d20.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d147.Loc == LocRegPair || d147.Loc == LocStackPair || d147.Loc == LocRegTriple || d147.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d147)
					d148 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d20, d147}, 3)
					d148.NoHeapPointer = false
					ctx.BindReg(d148.Reg, &d148)
					ctx.BindReg(d148.Reg2, &d148)
					ctx.BindReg(d148.Reg3, &d148)
					ctx.FreeDesc(&d147)
					ctx.EnsureDesc(&d148)
					ctx.EnsureDesc(&d148)
					ctx.EnsureDesc(&d148)
					if d148.Loc != LocRegTriple && d148.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d148)
					ctx.SyncDesc(&d2)
					d149 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d148, d2}, 3)
					d149.NoHeapPointer = false
					ctx.BindReg(d149.Reg, &d149)
					ctx.BindReg(d149.Reg2, &d149)
					ctx.BindReg(d149.Reg3, &d149)
					ctx.FreeDesc(&d148)
					ctx.FreeDesc(&d2)
					d150 = args[1]
					d150.ID = 0
					d152 = d150
					ctx.SyncDesc(&d152)
					if d152.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d152.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d152.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d152 = tmpScalar
					}
					d152 = JITPrepareScmerGoArg(ctx, d152)
					if d152.Loc != LocRegPair && d152.Loc != LocStackPair && d152.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d151 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d152}, 2)
					ctx.FreeDesc(&d150)
					ctx.EnsureDesc(&d149)
					ctx.EnsureDesc(&d149)
					ctx.EnsureDesc(&d149)
					if d149.Loc != LocRegTriple && d149.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (formatDateMySQL arg0)")
					}
					ctx.EnsureDesc(&d151)
					ctx.EnsureDesc(&d151)
					ctx.EnsureDesc(&d151)
					if d151.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d151.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d151.Imm)
						ptrWord, _ := d151.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d151.Imm.String())))
						d151 = tmpPair
					} else if d151.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d151.Type, Reg: ctx.AllocRegExcept(d151.Reg), Reg2: ctx.AllocRegExcept(d151.Reg)}
						switch d151.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d151)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d151)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d151)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d151)
						d151 = tmpPair
					}
					if d151.Loc != LocRegPair && d151.Loc != LocStackPair && d151.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (formatDateMySQL arg1)")
					}
					ctx.SyncDesc(&d149)
					ctx.SyncDesc(&d151)
					d153 = ctx.EmitGoCallScalar(GoFuncAddr(formatDateMySQL), []JITValueDesc{d149, d151}, 2)
					d153.NoHeapPointer = false
					ctx.BindReg(d153.Reg, &d153)
					ctx.BindReg(d153.Reg2, &d153)
					ctx.FreeDesc(&d149)
					ctx.EnsureDesc(&d153)
					d154 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d153}, 2)
					ctx.EmitMovPairToResult(&d154, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocRegPair || d20.Loc == LocStackPair || d20.Loc == LocRegTriple || d20.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d20)
					d155 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d20}, 2)
					d155.NoHeapPointer = false
					ctx.BindReg(d155.Reg, &d155)
					ctx.BindReg(d155.Reg2, &d155)
					ctx.FreeDesc(&d20)
					ctx.SyncDesc(&d155)
					if d155.Loc == LocRegPair || d155.Loc == LocStackPair || d155.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d155, &result)
						result.Type = d155.Type
					} else {
						switch d155.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d155)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d155)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d155)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d155, &result)
							result.Type = d155.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					ctx.ReclaimUntrackedRegs()
					d156 = args[1]
					d156.ID = 0
					d158 = d156
					d158.ID = 0
					d157 = ctx.EmitTagEqualsBorrowed(&d158, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d156)
					d159 = d157
					ctx.EnsureDesc(&d159)
					if d159.Loc != LocImm && d159.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d159.Loc == LocImm {
						if d159.Imm.Bool() {
							if ps.General {
							}
							ps160 := PhiState{General: ps.General}
							ps160.OverlayValues = make([]JITValueDesc, 160)
							ps160.OverlayValues[1] = d1
							ps160.OverlayValues[2] = d2
							ps160.OverlayValues[3] = d3
							ps160.OverlayValues[4] = d4
							ps160.OverlayValues[5] = d5
							ps160.OverlayValues[6] = d6
							ps160.OverlayValues[18] = d18
							ps160.OverlayValues[19] = d19
							ps160.OverlayValues[20] = d20
							ps160.OverlayValues[21] = d21
							ps160.OverlayValues[22] = d22
							ps160.OverlayValues[23] = d23
							ps160.OverlayValues[26] = d26
							ps160.OverlayValues[29] = d29
							ps160.OverlayValues[45] = d45
							ps160.OverlayValues[46] = d46
							ps160.OverlayValues[47] = d47
							ps160.OverlayValues[48] = d48
							ps160.OverlayValues[50] = d50
							ps160.OverlayValues[51] = d51
							ps160.OverlayValues[53] = d53
							ps160.OverlayValues[54] = d54
							ps160.OverlayValues[55] = d55
							ps160.OverlayValues[56] = d56
							ps160.OverlayValues[58] = d58
							ps160.OverlayValues[60] = d60
							ps160.OverlayValues[61] = d61
							ps160.OverlayValues[62] = d62
							ps160.OverlayValues[65] = d65
							ps160.OverlayValues[96] = d96
							ps160.OverlayValues[97] = d97
							ps160.OverlayValues[99] = d99
							ps160.OverlayValues[100] = d100
							ps160.OverlayValues[101] = d101
							ps160.OverlayValues[102] = d102
							ps160.OverlayValues[103] = d103
							ps160.OverlayValues[106] = d106
							ps160.OverlayValues[147] = d147
							ps160.OverlayValues[148] = d148
							ps160.OverlayValues[149] = d149
							ps160.OverlayValues[150] = d150
							ps160.OverlayValues[151] = d151
							ps160.OverlayValues[152] = d152
							ps160.OverlayValues[153] = d153
							ps160.OverlayValues[154] = d154
							ps160.OverlayValues[155] = d155
							ps160.OverlayValues[156] = d156
							ps160.OverlayValues[157] = d157
							ps160.OverlayValues[158] = d158
							ps160.OverlayValues[159] = d159
							return bbs[8].RenderPS(ps160)
						}
						if ps.General {
						}
						ps161 := PhiState{General: ps.General}
						ps161.OverlayValues = make([]JITValueDesc, 160)
						ps161.OverlayValues[1] = d1
						ps161.OverlayValues[2] = d2
						ps161.OverlayValues[3] = d3
						ps161.OverlayValues[4] = d4
						ps161.OverlayValues[5] = d5
						ps161.OverlayValues[6] = d6
						ps161.OverlayValues[18] = d18
						ps161.OverlayValues[19] = d19
						ps161.OverlayValues[20] = d20
						ps161.OverlayValues[21] = d21
						ps161.OverlayValues[22] = d22
						ps161.OverlayValues[23] = d23
						ps161.OverlayValues[26] = d26
						ps161.OverlayValues[29] = d29
						ps161.OverlayValues[45] = d45
						ps161.OverlayValues[46] = d46
						ps161.OverlayValues[47] = d47
						ps161.OverlayValues[48] = d48
						ps161.OverlayValues[50] = d50
						ps161.OverlayValues[51] = d51
						ps161.OverlayValues[53] = d53
						ps161.OverlayValues[54] = d54
						ps161.OverlayValues[55] = d55
						ps161.OverlayValues[56] = d56
						ps161.OverlayValues[58] = d58
						ps161.OverlayValues[60] = d60
						ps161.OverlayValues[61] = d61
						ps161.OverlayValues[62] = d62
						ps161.OverlayValues[65] = d65
						ps161.OverlayValues[96] = d96
						ps161.OverlayValues[97] = d97
						ps161.OverlayValues[99] = d99
						ps161.OverlayValues[100] = d100
						ps161.OverlayValues[101] = d101
						ps161.OverlayValues[102] = d102
						ps161.OverlayValues[103] = d103
						ps161.OverlayValues[106] = d106
						ps161.OverlayValues[147] = d147
						ps161.OverlayValues[148] = d148
						ps161.OverlayValues[149] = d149
						ps161.OverlayValues[150] = d150
						ps161.OverlayValues[151] = d151
						ps161.OverlayValues[152] = d152
						ps161.OverlayValues[153] = d153
						ps161.OverlayValues[154] = d154
						ps161.OverlayValues[155] = d155
						ps161.OverlayValues[156] = d156
						ps161.OverlayValues[157] = d157
						ps161.OverlayValues[158] = d158
						ps161.OverlayValues[159] = d159
						return bbs[7].RenderPS(ps161)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d159.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl8)
					ps162 := PhiState{General: true}
					ps162.OverlayValues = make([]JITValueDesc, 160)
					ps162.OverlayValues[1] = d1
					ps162.OverlayValues[2] = d2
					ps162.OverlayValues[3] = d3
					ps162.OverlayValues[4] = d4
					ps162.OverlayValues[5] = d5
					ps162.OverlayValues[6] = d6
					ps162.OverlayValues[18] = d18
					ps162.OverlayValues[19] = d19
					ps162.OverlayValues[20] = d20
					ps162.OverlayValues[21] = d21
					ps162.OverlayValues[22] = d22
					ps162.OverlayValues[23] = d23
					ps162.OverlayValues[26] = d26
					ps162.OverlayValues[29] = d29
					ps162.OverlayValues[45] = d45
					ps162.OverlayValues[46] = d46
					ps162.OverlayValues[47] = d47
					ps162.OverlayValues[48] = d48
					ps162.OverlayValues[50] = d50
					ps162.OverlayValues[51] = d51
					ps162.OverlayValues[53] = d53
					ps162.OverlayValues[54] = d54
					ps162.OverlayValues[55] = d55
					ps162.OverlayValues[56] = d56
					ps162.OverlayValues[58] = d58
					ps162.OverlayValues[60] = d60
					ps162.OverlayValues[61] = d61
					ps162.OverlayValues[62] = d62
					ps162.OverlayValues[65] = d65
					ps162.OverlayValues[96] = d96
					ps162.OverlayValues[97] = d97
					ps162.OverlayValues[99] = d99
					ps162.OverlayValues[100] = d100
					ps162.OverlayValues[101] = d101
					ps162.OverlayValues[102] = d102
					ps162.OverlayValues[103] = d103
					ps162.OverlayValues[106] = d106
					ps162.OverlayValues[147] = d147
					ps162.OverlayValues[148] = d148
					ps162.OverlayValues[149] = d149
					ps162.OverlayValues[150] = d150
					ps162.OverlayValues[151] = d151
					ps162.OverlayValues[152] = d152
					ps162.OverlayValues[153] = d153
					ps162.OverlayValues[154] = d154
					ps162.OverlayValues[155] = d155
					ps162.OverlayValues[156] = d156
					ps162.OverlayValues[157] = d157
					ps162.OverlayValues[158] = d158
					ps162.OverlayValues[159] = d159
					ps163 := PhiState{General: true}
					ps163.OverlayValues = make([]JITValueDesc, 160)
					ps163.OverlayValues[1] = d1
					ps163.OverlayValues[2] = d2
					ps163.OverlayValues[3] = d3
					ps163.OverlayValues[4] = d4
					ps163.OverlayValues[5] = d5
					ps163.OverlayValues[6] = d6
					ps163.OverlayValues[18] = d18
					ps163.OverlayValues[19] = d19
					ps163.OverlayValues[20] = d20
					ps163.OverlayValues[21] = d21
					ps163.OverlayValues[22] = d22
					ps163.OverlayValues[23] = d23
					ps163.OverlayValues[26] = d26
					ps163.OverlayValues[29] = d29
					ps163.OverlayValues[45] = d45
					ps163.OverlayValues[46] = d46
					ps163.OverlayValues[47] = d47
					ps163.OverlayValues[48] = d48
					ps163.OverlayValues[50] = d50
					ps163.OverlayValues[51] = d51
					ps163.OverlayValues[53] = d53
					ps163.OverlayValues[54] = d54
					ps163.OverlayValues[55] = d55
					ps163.OverlayValues[56] = d56
					ps163.OverlayValues[58] = d58
					ps163.OverlayValues[60] = d60
					ps163.OverlayValues[61] = d61
					ps163.OverlayValues[62] = d62
					ps163.OverlayValues[65] = d65
					ps163.OverlayValues[96] = d96
					ps163.OverlayValues[97] = d97
					ps163.OverlayValues[99] = d99
					ps163.OverlayValues[100] = d100
					ps163.OverlayValues[101] = d101
					ps163.OverlayValues[102] = d102
					ps163.OverlayValues[103] = d103
					ps163.OverlayValues[106] = d106
					ps163.OverlayValues[147] = d147
					ps163.OverlayValues[148] = d148
					ps163.OverlayValues[149] = d149
					ps163.OverlayValues[150] = d150
					ps163.OverlayValues[151] = d151
					ps163.OverlayValues[152] = d152
					ps163.OverlayValues[153] = d153
					ps163.OverlayValues[154] = d154
					ps163.OverlayValues[155] = d155
					ps163.OverlayValues[156] = d156
					ps163.OverlayValues[157] = d157
					ps163.OverlayValues[158] = d158
					ps163.OverlayValues[159] = d159
					snap164 := d1
					snap165 := d2
					snap166 := d3
					snap167 := d4
					snap168 := d5
					snap169 := d6
					snap170 := d18
					snap171 := d19
					snap172 := d20
					snap173 := d21
					snap174 := d22
					snap175 := d23
					snap176 := d26
					snap177 := d29
					snap178 := d45
					snap179 := d46
					snap180 := d47
					snap181 := d48
					snap182 := d50
					snap183 := d51
					snap184 := d53
					snap185 := d54
					snap186 := d55
					snap187 := d56
					snap188 := d58
					snap189 := d60
					snap190 := d61
					snap191 := d62
					snap192 := d65
					snap193 := d96
					snap194 := d97
					snap195 := d99
					snap196 := d100
					snap197 := d101
					snap198 := d102
					snap199 := d103
					snap200 := d106
					snap201 := d147
					snap202 := d148
					snap203 := d149
					snap204 := d150
					snap205 := d151
					snap206 := d152
					snap207 := d153
					snap208 := d154
					snap209 := d155
					snap210 := d156
					snap211 := d157
					snap212 := d158
					snap213 := d159
					alloc214 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps163)
					}
					ctx.RestoreAllocState(alloc214)
					d1 = snap164
					d2 = snap165
					d3 = snap166
					d4 = snap167
					d5 = snap168
					d6 = snap169
					d18 = snap170
					d19 = snap171
					d20 = snap172
					d21 = snap173
					d22 = snap174
					d23 = snap175
					d26 = snap176
					d29 = snap177
					d45 = snap178
					d46 = snap179
					d47 = snap180
					d48 = snap181
					d50 = snap182
					d51 = snap183
					d53 = snap184
					d54 = snap185
					d55 = snap186
					d56 = snap187
					d58 = snap188
					d60 = snap189
					d61 = snap190
					d62 = snap191
					d65 = snap192
					d96 = snap193
					d97 = snap194
					d99 = snap195
					d100 = snap196
					d101 = snap197
					d102 = snap198
					d103 = snap199
					d106 = snap200
					d147 = snap201
					d148 = snap202
					d149 = snap203
					d150 = snap204
					d151 = snap205
					d152 = snap206
					d153 = snap207
					d154 = snap208
					d155 = snap209
					d156 = snap210
					d157 = snap211
					d158 = snap212
					d159 = snap213
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps162)
					}
					return result
					ctx.FreeDesc(&d157)
					return result
				}
				ps215 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps215)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
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
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = true
				ctx.BindReg(d2.Reg, &d2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
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
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = true
				ctx.BindReg(d2.Reg, &d2)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d1}, 1)
				d3.NoHeapPointer = true
				ctx.BindReg(d3.Reg, &d3)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
				}
				ctx.SyncDesc(&d1)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d1}, 1)
				d4.NoHeapPointer = true
				ctx.BindReg(d4.Reg, &d4)
				ctx.FreeDesc(&d1)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
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
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d5)
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
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				if d10.Loc != LocRegTriple && d10.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d10)
				d11 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d10}, 1)
				d11.NoHeapPointer = true
				ctx.BindReg(d11.Reg, &d11)
				ctx.FreeDesc(&d10)
				ctx.EnsureDesc(&d11)
				ctx.EnsureDesc(&d11)
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
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
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
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
					ctx.EmitImulRegImm32(d2.Reg, int32(3600))
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
					ctx.BindReg(d2.Reg, &d3)
				}
				if d3.Loc == LocReg && d2.Loc == LocReg && d3.Reg == d2.Reg {
					ctx.TransferReg(d2.Reg)
					d2.Loc = LocNone
				}
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
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
					ctx.EmitImulRegImm32(d4.Reg, int32(60))
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
				ctx.EnsureDescsTogether(&d3, &d5)
				var d6 JITValueDesc
				if d3.Loc == LocImm && d5.Loc == LocImm {
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d5.Imm.Int())}
				} else if d5.Loc == LocImm && d5.Imm.Int() == 0 {
					r0 := ctx.AllocRegExcept(d3.Reg)
					ctx.EmitMovRegReg(r0, d3.Reg)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
					ctx.BindReg(r0, &d6)
				} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
					ctx.BindReg(d5.Reg, &d6)
				} else if d3.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d5.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
					ctx.EmitAddInt64(scratch, d5.Reg)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d6)
				} else if d5.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d3.Reg)
					ctx.EmitMovRegReg(scratch, d3.Reg)
					if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(scratch, int32(d5.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
						ctx.EmitAddInt64(scratch, RegR11)
					}
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d6)
				} else {
					r1 := ctx.AllocRegExcept(d3.Reg, d5.Reg)
					ctx.EmitMovRegReg(r1, d3.Reg)
					ctx.EmitAddInt64(r1, d5.Reg)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
					ctx.BindReg(r1, &d6)
				}
				if d6.Loc == LocReg && d3.Loc == LocReg && d6.Reg == d3.Reg {
					ctx.TransferReg(d3.Reg)
					d3.Loc = LocNone
				}
				ctx.FreeDesc(&d3)
				ctx.FreeDesc(&d5)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
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
				ctx.EnsureDescsTogether(&d6, &d7)
				var d8 JITValueDesc
				if d6.Loc == LocImm && d7.Loc == LocImm {
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + d7.Imm.Int())}
				} else if d7.Loc == LocImm && d7.Imm.Int() == 0 {
					r2 := ctx.AllocRegExcept(d6.Reg)
					ctx.EmitMovRegReg(r2, d6.Reg)
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d8)
				} else if d6.Loc == LocImm && d6.Imm.Int() == 0 {
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg}
					ctx.BindReg(d7.Reg, &d8)
				} else if d6.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d7.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d6.Imm.Int()))
					ctx.EmitAddInt64(scratch, d7.Reg)
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d8)
				} else if d7.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d6.Reg)
					ctx.EmitMovRegReg(scratch, d6.Reg)
					if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(scratch, int32(d7.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
						ctx.EmitAddInt64(scratch, RegR11)
					}
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d8)
				} else {
					r3 := ctx.AllocRegExcept(d6.Reg, d7.Reg)
					ctx.EmitMovRegReg(r3, d6.Reg)
					ctx.EmitAddInt64(r3, d7.Reg)
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if d0.Loc != LocRegTriple && d0.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d0}, 1)
				d1.NoHeapPointer = true
				ctx.BindReg(d1.Reg, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
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
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
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
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
				var d191 JITValueDesc
				_ = d191
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d198 JITValueDesc
				_ = d198
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d203 JITValueDesc
				_ = d203
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
				var d210 JITValueDesc
				_ = d210
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d217 JITValueDesc
				_ = d217
				var d218 JITValueDesc
				_ = d218
				var d219 JITValueDesc
				_ = d219
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d222 JITValueDesc
				_ = d222
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				var bbs [11]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(2)
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
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
							if ps.General {
							}
							ps7 := PhiState{General: ps.General}
							ps7.OverlayValues = make([]JITValueDesc, 7)
							ps7.OverlayValues[1] = d1
							ps7.OverlayValues[2] = d2
							ps7.OverlayValues[3] = d3
							ps7.OverlayValues[4] = d4
							ps7.OverlayValues[5] = d5
							ps7.OverlayValues[6] = d6
							return bbs[1].RenderPS(ps7)
						}
						if ps.General {
						}
						ps8 := PhiState{General: ps.General}
						ps8.OverlayValues = make([]JITValueDesc, 7)
						ps8.OverlayValues[1] = d1
						ps8.OverlayValues[2] = d2
						ps8.OverlayValues[3] = d3
						ps8.OverlayValues[4] = d4
						ps8.OverlayValues[5] = d5
						ps8.OverlayValues[6] = d6
						return bbs[3].RenderPS(ps8)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl4)
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 7)
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					ps9.OverlayValues[6] = d6
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 7)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					snap11 := d1
					snap12 := d2
					snap13 := d3
					snap14 := d4
					snap15 := d5
					snap16 := d6
					alloc17 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc17)
					d1 = snap11
					d2 = snap12
					d3 = snap13
					d4 = snap14
					d5 = snap15
					d6 = snap16
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps9)
					}
					return result
					ctx.FreeDesc(&d4)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					ctx.ReclaimUntrackedRegs()
					d18 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d18, &result)
						result.Type = d18.Type
					} else {
						switch d18.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d18)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d18)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d18)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d18, &result)
							result.Type = d18.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					ctx.ReclaimUntrackedRegs()
					d19 = args[1]
					d19.ID = 0
					d21 = d19
					ctx.SyncDesc(&d21)
					if d21.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d21.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d21.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d21 = tmpScalar
					}
					d21 = JITPrepareScmerGoArg(ctx, d21)
					if d21.Loc != LocRegPair && d21.Loc != LocStackPair && d21.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d21}, 2)
					ctx.FreeDesc(&d19)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d20.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d20.Imm)
						ptrWord, _ := d20.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d20.Imm.String())))
						d20 = tmpPair
					} else if d20.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d20.Type, Reg: ctx.AllocRegExcept(d20.Reg), Reg2: ctx.AllocRegExcept(d20.Reg)}
						switch d20.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d20)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d20)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d20)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d20)
						d20 = tmpPair
					}
					if d20.Loc != LocRegPair && d20.Loc != LocStackPair && d20.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d20)
					callResults22 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d20}, []uint8{1, 2}, []uint8{1, 3})
					d23 = callResults22[0]
					_ = d23
					d24 = callResults22[1]
					_ = d24
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.EnsureDesc(&d24)
					var d25 JITValueDesc
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d24.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d24)
						if d24.Loc != LocReg && d24.Loc != LocRegPair && d24.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d24.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d25)
					}
					ctx.FreeDesc(&d24)
					d26 = d25
					ctx.EnsureDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							if ps.General {
							}
							ps27 := PhiState{General: ps.General}
							ps27.OverlayValues = make([]JITValueDesc, 27)
							ps27.OverlayValues[1] = d1
							ps27.OverlayValues[2] = d2
							ps27.OverlayValues[3] = d3
							ps27.OverlayValues[4] = d4
							ps27.OverlayValues[5] = d5
							ps27.OverlayValues[6] = d6
							ps27.OverlayValues[18] = d18
							ps27.OverlayValues[19] = d19
							ps27.OverlayValues[20] = d20
							ps27.OverlayValues[21] = d21
							ps27.OverlayValues[23] = d23
							ps27.OverlayValues[24] = d24
							ps27.OverlayValues[25] = d25
							ps27.OverlayValues[26] = d26
							return bbs[4].RenderPS(ps27)
						}
						if ps.General {
						}
						ps28 := PhiState{General: ps.General}
						ps28.OverlayValues = make([]JITValueDesc, 27)
						ps28.OverlayValues[1] = d1
						ps28.OverlayValues[2] = d2
						ps28.OverlayValues[3] = d3
						ps28.OverlayValues[4] = d4
						ps28.OverlayValues[5] = d5
						ps28.OverlayValues[6] = d6
						ps28.OverlayValues[18] = d18
						ps28.OverlayValues[19] = d19
						ps28.OverlayValues[20] = d20
						ps28.OverlayValues[21] = d21
						ps28.OverlayValues[23] = d23
						ps28.OverlayValues[24] = d24
						ps28.OverlayValues[25] = d25
						ps28.OverlayValues[26] = d26
						return bbs[5].RenderPS(ps28)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl6)
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 27)
					ps29.OverlayValues[1] = d1
					ps29.OverlayValues[2] = d2
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[6] = d6
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[20] = d20
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[25] = d25
					ps29.OverlayValues[26] = d26
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 27)
					ps30.OverlayValues[1] = d1
					ps30.OverlayValues[2] = d2
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[18] = d18
					ps30.OverlayValues[19] = d19
					ps30.OverlayValues[20] = d20
					ps30.OverlayValues[21] = d21
					ps30.OverlayValues[23] = d23
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[26] = d26
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d4
					snap35 := d5
					snap36 := d6
					snap37 := d18
					snap38 := d19
					snap39 := d20
					snap40 := d21
					snap41 := d23
					snap42 := d24
					snap43 := d25
					snap44 := d26
					alloc45 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc45)
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d4 = snap34
					d5 = snap35
					d6 = snap36
					d18 = snap37
					d19 = snap38
					d20 = snap39
					d21 = snap40
					d23 = snap41
					d24 = snap42
					d25 = snap43
					d26 = snap44
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps29)
					}
					return result
					ctx.FreeDesc(&d25)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					ctx.ReclaimUntrackedRegs()
					d46 = args[1]
					d46.ID = 0
					d48 = d46
					d48.ID = 0
					d47 = ctx.EmitTagEqualsBorrowed(&d48, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d46)
					d49 = d47
					ctx.EnsureDesc(&d49)
					if d49.Loc != LocImm && d49.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d49.Loc == LocImm {
						if d49.Imm.Bool() {
							if ps.General {
							}
							ps50 := PhiState{General: ps.General}
							ps50.OverlayValues = make([]JITValueDesc, 50)
							ps50.OverlayValues[1] = d1
							ps50.OverlayValues[2] = d2
							ps50.OverlayValues[3] = d3
							ps50.OverlayValues[4] = d4
							ps50.OverlayValues[5] = d5
							ps50.OverlayValues[6] = d6
							ps50.OverlayValues[18] = d18
							ps50.OverlayValues[19] = d19
							ps50.OverlayValues[20] = d20
							ps50.OverlayValues[21] = d21
							ps50.OverlayValues[23] = d23
							ps50.OverlayValues[24] = d24
							ps50.OverlayValues[25] = d25
							ps50.OverlayValues[26] = d26
							ps50.OverlayValues[46] = d46
							ps50.OverlayValues[47] = d47
							ps50.OverlayValues[48] = d48
							ps50.OverlayValues[49] = d49
							return bbs[1].RenderPS(ps50)
						}
						if ps.General {
						}
						ps51 := PhiState{General: ps.General}
						ps51.OverlayValues = make([]JITValueDesc, 50)
						ps51.OverlayValues[1] = d1
						ps51.OverlayValues[2] = d2
						ps51.OverlayValues[3] = d3
						ps51.OverlayValues[4] = d4
						ps51.OverlayValues[5] = d5
						ps51.OverlayValues[6] = d6
						ps51.OverlayValues[18] = d18
						ps51.OverlayValues[19] = d19
						ps51.OverlayValues[20] = d20
						ps51.OverlayValues[21] = d21
						ps51.OverlayValues[23] = d23
						ps51.OverlayValues[24] = d24
						ps51.OverlayValues[25] = d25
						ps51.OverlayValues[26] = d26
						ps51.OverlayValues[46] = d46
						ps51.OverlayValues[47] = d47
						ps51.OverlayValues[48] = d48
						ps51.OverlayValues[49] = d49
						return bbs[2].RenderPS(ps51)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d49.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl3)
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 50)
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[20] = d20
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[26] = d26
					ps52.OverlayValues[46] = d46
					ps52.OverlayValues[47] = d47
					ps52.OverlayValues[48] = d48
					ps52.OverlayValues[49] = d49
					ps53 := PhiState{General: true}
					ps53.OverlayValues = make([]JITValueDesc, 50)
					ps53.OverlayValues[1] = d1
					ps53.OverlayValues[2] = d2
					ps53.OverlayValues[3] = d3
					ps53.OverlayValues[4] = d4
					ps53.OverlayValues[5] = d5
					ps53.OverlayValues[6] = d6
					ps53.OverlayValues[18] = d18
					ps53.OverlayValues[19] = d19
					ps53.OverlayValues[20] = d20
					ps53.OverlayValues[21] = d21
					ps53.OverlayValues[23] = d23
					ps53.OverlayValues[24] = d24
					ps53.OverlayValues[25] = d25
					ps53.OverlayValues[26] = d26
					ps53.OverlayValues[46] = d46
					ps53.OverlayValues[47] = d47
					ps53.OverlayValues[48] = d48
					ps53.OverlayValues[49] = d49
					snap54 := d1
					snap55 := d2
					snap56 := d3
					snap57 := d4
					snap58 := d5
					snap59 := d6
					snap60 := d18
					snap61 := d19
					snap62 := d20
					snap63 := d21
					snap64 := d23
					snap65 := d24
					snap66 := d25
					snap67 := d26
					snap68 := d46
					snap69 := d47
					snap70 := d48
					snap71 := d49
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps53)
					}
					ctx.RestoreAllocState(alloc72)
					d1 = snap54
					d2 = snap55
					d3 = snap56
					d4 = snap57
					d5 = snap58
					d6 = snap59
					d18 = snap60
					d19 = snap61
					d20 = snap62
					d21 = snap63
					d23 = snap64
					d24 = snap65
					d25 = snap66
					d26 = snap67
					d46 = snap68
					d47 = snap69
					d48 = snap70
					d49 = snap71
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps52)
					}
					return result
					ctx.FreeDesc(&d47)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					ctx.ReclaimUntrackedRegs()
					d73 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d73)
					if d73.Loc == LocRegPair || d73.Loc == LocStackPair || d73.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d73, &result)
						result.Type = d73.Type
					} else {
						switch d73.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d73)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d73)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d73)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d73, &result)
							result.Type = d73.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					ctx.ReclaimUntrackedRegs()
					d74 = args[0]
					d74.ID = 0
					d75 = ctx.EmitGetTagDesc(&d74, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d74)
					ctx.EnsureDesc(&d75)
					var d76 JITValueDesc
					if d75.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d75.Imm.Int()) == uint64(0x10))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d75.Reg, 16)
						ctx.EmitSetcc(r1, CondEqual)
						d76 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d76)
					}
					ctx.FreeDesc(&d75)
					d77 = d76
					ctx.EnsureDesc(&d77)
					if d77.Loc != LocImm && d77.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d77.Loc == LocImm {
						if d77.Imm.Bool() {
							if ps.General {
							}
							ps78 := PhiState{General: ps.General}
							ps78.OverlayValues = make([]JITValueDesc, 78)
							ps78.OverlayValues[1] = d1
							ps78.OverlayValues[2] = d2
							ps78.OverlayValues[3] = d3
							ps78.OverlayValues[4] = d4
							ps78.OverlayValues[5] = d5
							ps78.OverlayValues[6] = d6
							ps78.OverlayValues[18] = d18
							ps78.OverlayValues[19] = d19
							ps78.OverlayValues[20] = d20
							ps78.OverlayValues[21] = d21
							ps78.OverlayValues[23] = d23
							ps78.OverlayValues[24] = d24
							ps78.OverlayValues[25] = d25
							ps78.OverlayValues[26] = d26
							ps78.OverlayValues[46] = d46
							ps78.OverlayValues[47] = d47
							ps78.OverlayValues[48] = d48
							ps78.OverlayValues[49] = d49
							ps78.OverlayValues[73] = d73
							ps78.OverlayValues[74] = d74
							ps78.OverlayValues[75] = d75
							ps78.OverlayValues[76] = d76
							ps78.OverlayValues[77] = d77
							return bbs[6].RenderPS(ps78)
						}
						if ps.General {
						}
						ps79 := PhiState{General: ps.General}
						ps79.OverlayValues = make([]JITValueDesc, 78)
						ps79.OverlayValues[1] = d1
						ps79.OverlayValues[2] = d2
						ps79.OverlayValues[3] = d3
						ps79.OverlayValues[4] = d4
						ps79.OverlayValues[5] = d5
						ps79.OverlayValues[6] = d6
						ps79.OverlayValues[18] = d18
						ps79.OverlayValues[19] = d19
						ps79.OverlayValues[20] = d20
						ps79.OverlayValues[21] = d21
						ps79.OverlayValues[23] = d23
						ps79.OverlayValues[24] = d24
						ps79.OverlayValues[25] = d25
						ps79.OverlayValues[26] = d26
						ps79.OverlayValues[46] = d46
						ps79.OverlayValues[47] = d47
						ps79.OverlayValues[48] = d48
						ps79.OverlayValues[49] = d49
						ps79.OverlayValues[73] = d73
						ps79.OverlayValues[74] = d74
						ps79.OverlayValues[75] = d75
						ps79.OverlayValues[76] = d76
						ps79.OverlayValues[77] = d77
						return bbs[8].RenderPS(ps79)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d77.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl9)
					ps80 := PhiState{General: true}
					ps80.OverlayValues = make([]JITValueDesc, 78)
					ps80.OverlayValues[1] = d1
					ps80.OverlayValues[2] = d2
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[4] = d4
					ps80.OverlayValues[5] = d5
					ps80.OverlayValues[6] = d6
					ps80.OverlayValues[18] = d18
					ps80.OverlayValues[19] = d19
					ps80.OverlayValues[20] = d20
					ps80.OverlayValues[21] = d21
					ps80.OverlayValues[23] = d23
					ps80.OverlayValues[24] = d24
					ps80.OverlayValues[25] = d25
					ps80.OverlayValues[26] = d26
					ps80.OverlayValues[46] = d46
					ps80.OverlayValues[47] = d47
					ps80.OverlayValues[48] = d48
					ps80.OverlayValues[49] = d49
					ps80.OverlayValues[73] = d73
					ps80.OverlayValues[74] = d74
					ps80.OverlayValues[75] = d75
					ps80.OverlayValues[76] = d76
					ps80.OverlayValues[77] = d77
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 78)
					ps81.OverlayValues[1] = d1
					ps81.OverlayValues[2] = d2
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[4] = d4
					ps81.OverlayValues[5] = d5
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[18] = d18
					ps81.OverlayValues[19] = d19
					ps81.OverlayValues[20] = d20
					ps81.OverlayValues[21] = d21
					ps81.OverlayValues[23] = d23
					ps81.OverlayValues[24] = d24
					ps81.OverlayValues[25] = d25
					ps81.OverlayValues[26] = d26
					ps81.OverlayValues[46] = d46
					ps81.OverlayValues[47] = d47
					ps81.OverlayValues[48] = d48
					ps81.OverlayValues[49] = d49
					ps81.OverlayValues[73] = d73
					ps81.OverlayValues[74] = d74
					ps81.OverlayValues[75] = d75
					ps81.OverlayValues[76] = d76
					ps81.OverlayValues[77] = d77
					snap82 := d1
					snap83 := d2
					snap84 := d3
					snap85 := d4
					snap86 := d5
					snap87 := d6
					snap88 := d18
					snap89 := d19
					snap90 := d20
					snap91 := d21
					snap92 := d23
					snap93 := d24
					snap94 := d25
					snap95 := d26
					snap96 := d46
					snap97 := d47
					snap98 := d48
					snap99 := d49
					snap100 := d73
					snap101 := d74
					snap102 := d75
					snap103 := d76
					snap104 := d77
					alloc105 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps81)
					}
					ctx.RestoreAllocState(alloc105)
					d1 = snap82
					d2 = snap83
					d3 = snap84
					d4 = snap85
					d5 = snap86
					d6 = snap87
					d18 = snap88
					d19 = snap89
					d20 = snap90
					d21 = snap91
					d23 = snap92
					d24 = snap93
					d25 = snap94
					d26 = snap95
					d46 = snap96
					d47 = snap97
					d48 = snap98
					d49 = snap99
					d73 = snap100
					d74 = snap101
					d75 = snap102
					d76 = snap103
					d77 = snap104
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps80)
					}
					return result
					ctx.FreeDesc(&d76)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					ctx.ReclaimUntrackedRegs()
					d106 = args[0]
					d106.ID = 0
					var d107 JITValueDesc
					ctx.EnsureDesc(&d106)
					if d106.Loc == LocImm {
						_, auxWord := d106.Imm.RawWords()
						d107 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d106.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r2 := ctx.AllocReg()
						ctx.EmitMovRegReg(r2, d106.Reg2)
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d107)
					}
					ctx.EnsureDesc(&d107)
					d108 = d107
					_ = d108
					ctx.StabilizeDescForControlFlow(&d108)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d108)
					var d109 JITValueDesc
					if d108.Loc == LocImm {
						d109 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d108.Imm.Int()) >> 8))}
					} else {
						r3 := ctx.AllocRegExcept(d108.Reg)
						ctx.EmitMovRegReg(r3, d108.Reg)
						ctx.EmitShrRegImm8(r3, 8)
						d109 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d109)
					}
					if d109.Loc == LocReg && d108.Loc == LocReg && d109.Reg == d108.Reg {
						ctx.TransferReg(d108.Reg)
						d108.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d109)
					ctx.FreeDesc(&d107)
					ctx.EnsureDesc(&d109)
					d110 = d109
					_ = d110
					ctx.StabilizeDescForControlFlow(&d110)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl21 := ctx.ReserveLabel()
					_ = lbl21
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl21)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					var d111 JITValueDesc
					if d110.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d110.Imm.Int() & 35184372088831)}
					} else {
						r4 := ctx.AllocRegExcept(d110.Reg)
						ctx.EmitMovRegReg(r4, d110.Reg)
						ctx.EmitMovRegImm64(RegR11, 0x1fffffffffff)
						ctx.EmitAndInt64(r4, RegR11)
						d111 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d111)
					}
					if d111.Loc == LocReg && d110.Loc == LocReg && d111.Reg == d110.Reg {
						ctx.TransferReg(d110.Reg)
						d110.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d111)
					var d112 JITValueDesc
					if d111.Loc == LocImm {
						d112 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d111.Imm.Int()) << 19))}
					} else {
						ctx.EmitShlRegImm8(d111.Reg, 19)
						d112 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d111.Reg}
						ctx.BindReg(d111.Reg, &d112)
					}
					if d112.Loc == LocReg && d111.Loc == LocReg && d112.Reg == d111.Reg {
						ctx.TransferReg(d111.Reg)
						d111.Loc = LocNone
					}
					ctx.FreeDesc(&d111)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d112)
					ctx.EnsureDesc(&d112)
					var d113 JITValueDesc
					if d112.Loc == LocImm {
						d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d112.Imm.Int()))))}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegReg(r5, d112.Reg)
						d113 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d113)
					}
					ctx.FreeDesc(&d112)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d113)
					var d114 JITValueDesc
					if d113.Loc == LocImm {
						d114 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d113.Imm.Int()) >> 19))}
					} else {
						ctx.EmitShrRegImm8(d113.Reg, 19)
						d114 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d113.Reg}
						ctx.BindReg(d113.Reg, &d114)
					}
					if d114.Loc == LocReg && d113.Loc == LocReg && d114.Reg == d113.Reg {
						ctx.TransferReg(d113.Reg)
						d113.Loc = LocNone
					}
					ctx.FreeDesc(&d113)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d114)
					ctx.StabilizeDescForControlFlow(&d114)
					ctx.FreeDesc(&d109)
					d115 = args[0]
					d115.ID = 0
					var d116 JITValueDesc
					ctx.EnsureDesc(&d115)
					if d115.Loc == LocImm {
						_, auxWord := d115.Imm.RawWords()
						d116 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d115.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r6 := ctx.AllocReg()
						ctx.EmitMovRegReg(r6, d115.Reg2)
						d116 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d116)
					}
					ctx.EnsureDesc(&d116)
					d117 = d116
					_ = d117
					ctx.StabilizeDescForControlFlow(&d117)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl22 := ctx.ReserveLabel()
					_ = lbl22
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d117)
					var d118 JITValueDesc
					if d117.Loc == LocImm {
						d118 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d117.Imm.Int()) >> 8))}
					} else {
						r7 := ctx.AllocRegExcept(d117.Reg)
						ctx.EmitMovRegReg(r7, d117.Reg)
						ctx.EmitShrRegImm8(r7, 8)
						d118 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d118)
					}
					if d118.Loc == LocReg && d117.Loc == LocReg && d118.Reg == d117.Reg {
						ctx.TransferReg(d117.Reg)
						d117.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d118)
					ctx.FreeDesc(&d116)
					ctx.EnsureDesc(&d118)
					d119 = d118
					_ = d119
					ctx.StabilizeDescForControlFlow(&d119)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d119)
					var d120 JITValueDesc
					if d119.Loc == LocImm {
						d120 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d119.Imm.Int()) >> 45))}
					} else {
						r8 := ctx.AllocRegExcept(d119.Reg)
						ctx.EmitMovRegReg(r8, d119.Reg)
						ctx.EmitShrRegImm8(r8, 45)
						d120 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d120)
					}
					if d120.Loc == LocReg && d119.Loc == LocReg && d120.Reg == d119.Reg {
						ctx.TransferReg(d119.Reg)
						d119.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d120)
					var d121 JITValueDesc
					if d120.Loc == LocImm {
						d121 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d120.Imm.Int() & 2047)}
					} else {
						ctx.EmitAndRegImm32(d120.Reg, int32(2047))
						d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d120.Reg}
						ctx.BindReg(d120.Reg, &d121)
					}
					if d121.Loc == LocReg && d120.Loc == LocReg && d121.Reg == d120.Reg {
						ctx.TransferReg(d120.Reg)
						d120.Loc = LocNone
					}
					ctx.FreeDesc(&d120)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d121)
					ctx.EnsureDesc(&d121)
					var d122 JITValueDesc
					if d121.Loc == LocImm {
						d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d121.Imm.Int()))))}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitMovRegReg(r9, d121.Reg)
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d122)
					}
					ctx.FreeDesc(&d121)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d122)
					ctx.StabilizeDescForControlFlow(&d122)
					ctx.FreeDesc(&d118)
					if ps.General {
						ctx.SyncDesc(&d114)
						if d114.Loc == LocReg {
							ctx.ProtectReg(d114.Reg)
						} else if d114.Loc == LocRegPair {
							ctx.ProtectReg(d114.Reg)
							ctx.ProtectReg(d114.Reg2)
						}
						ctx.SyncDesc(&d122)
						if d122.Loc == LocReg {
							ctx.ProtectReg(d122.Reg)
						} else if d122.Loc == LocRegPair {
							ctx.ProtectReg(d122.Reg)
							ctx.ProtectReg(d122.Reg2)
						}
						d123 = d114
						if d123.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d123)
						ctx.EmitStoreToStack(d123, int32(bbs[7].PhiBase)+int32(0))
						d124 = d122
						if d124.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d124)
						ctx.EmitStoreToStack(d124, int32(bbs[7].PhiBase)+int32(16))
						if d114.Loc == LocReg {
							ctx.UnprotectReg(d114.Reg)
						} else if d114.Loc == LocRegPair {
							ctx.UnprotectReg(d114.Reg)
							ctx.UnprotectReg(d114.Reg2)
						}
						if d122.Loc == LocReg {
							ctx.UnprotectReg(d122.Reg)
						} else if d122.Loc == LocRegPair {
							ctx.UnprotectReg(d122.Reg)
							ctx.UnprotectReg(d122.Reg2)
						}
					}
					ps125 := PhiState{General: ps.General}
					ps125.OverlayValues = make([]JITValueDesc, 125)
					ps125.OverlayValues[1] = d1
					ps125.OverlayValues[2] = d2
					ps125.OverlayValues[3] = d3
					ps125.OverlayValues[4] = d4
					ps125.OverlayValues[5] = d5
					ps125.OverlayValues[6] = d6
					ps125.OverlayValues[18] = d18
					ps125.OverlayValues[19] = d19
					ps125.OverlayValues[20] = d20
					ps125.OverlayValues[21] = d21
					ps125.OverlayValues[23] = d23
					ps125.OverlayValues[24] = d24
					ps125.OverlayValues[25] = d25
					ps125.OverlayValues[26] = d26
					ps125.OverlayValues[46] = d46
					ps125.OverlayValues[47] = d47
					ps125.OverlayValues[48] = d48
					ps125.OverlayValues[49] = d49
					ps125.OverlayValues[73] = d73
					ps125.OverlayValues[74] = d74
					ps125.OverlayValues[75] = d75
					ps125.OverlayValues[76] = d76
					ps125.OverlayValues[77] = d77
					ps125.OverlayValues[106] = d106
					ps125.OverlayValues[107] = d107
					ps125.OverlayValues[108] = d108
					ps125.OverlayValues[109] = d109
					ps125.OverlayValues[110] = d110
					ps125.OverlayValues[111] = d111
					ps125.OverlayValues[112] = d112
					ps125.OverlayValues[113] = d113
					ps125.OverlayValues[114] = d114
					ps125.OverlayValues[115] = d115
					ps125.OverlayValues[116] = d116
					ps125.OverlayValues[117] = d117
					ps125.OverlayValues[118] = d118
					ps125.OverlayValues[119] = d119
					ps125.OverlayValues[120] = d120
					ps125.OverlayValues[121] = d121
					ps125.OverlayValues[122] = d122
					ps125.OverlayValues[123] = d123
					ps125.OverlayValues[124] = d124
					ps125.PhiValues = make([]JITValueDesc, 2)
					d126 = d114
					ps125.PhiValues[0] = d126
					d127 = d122
					ps125.PhiValues[1] = d127
					if ps125.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps125)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d128 := ps.PhiValues[0]
							ctx.EnsureDesc(&d128)
							ctx.EmitStoreToStack(d128, int32(bbs[7].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d129 := ps.PhiValues[1]
							ctx.EnsureDesc(&d129)
							ctx.EmitStoreToStack(d129, int32(bbs[7].PhiBase)+int32(16))
						}
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d2 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d2)
					var d130 JITValueDesc
					if d2.Loc == LocImm {
						d130 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 0)}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r10, CondEqual)
						d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d130)
					}
					ctx.FreeDesc(&d2)
					d131 = d130
					ctx.EnsureDesc(&d131)
					if d131.Loc != LocImm && d131.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d131.Loc == LocImm {
						if d131.Imm.Bool() {
							if ps.General {
							}
							ps132 := PhiState{General: ps.General}
							ps132.OverlayValues = make([]JITValueDesc, 132)
							ps132.OverlayValues[1] = d1
							ps132.OverlayValues[2] = d2
							ps132.OverlayValues[3] = d3
							ps132.OverlayValues[4] = d4
							ps132.OverlayValues[5] = d5
							ps132.OverlayValues[6] = d6
							ps132.OverlayValues[18] = d18
							ps132.OverlayValues[19] = d19
							ps132.OverlayValues[20] = d20
							ps132.OverlayValues[21] = d21
							ps132.OverlayValues[23] = d23
							ps132.OverlayValues[24] = d24
							ps132.OverlayValues[25] = d25
							ps132.OverlayValues[26] = d26
							ps132.OverlayValues[46] = d46
							ps132.OverlayValues[47] = d47
							ps132.OverlayValues[48] = d48
							ps132.OverlayValues[49] = d49
							ps132.OverlayValues[73] = d73
							ps132.OverlayValues[74] = d74
							ps132.OverlayValues[75] = d75
							ps132.OverlayValues[76] = d76
							ps132.OverlayValues[77] = d77
							ps132.OverlayValues[106] = d106
							ps132.OverlayValues[107] = d107
							ps132.OverlayValues[108] = d108
							ps132.OverlayValues[109] = d109
							ps132.OverlayValues[110] = d110
							ps132.OverlayValues[111] = d111
							ps132.OverlayValues[112] = d112
							ps132.OverlayValues[113] = d113
							ps132.OverlayValues[114] = d114
							ps132.OverlayValues[115] = d115
							ps132.OverlayValues[116] = d116
							ps132.OverlayValues[117] = d117
							ps132.OverlayValues[118] = d118
							ps132.OverlayValues[119] = d119
							ps132.OverlayValues[120] = d120
							ps132.OverlayValues[121] = d121
							ps132.OverlayValues[122] = d122
							ps132.OverlayValues[123] = d123
							ps132.OverlayValues[124] = d124
							ps132.OverlayValues[126] = d126
							ps132.OverlayValues[127] = d127
							ps132.OverlayValues[128] = d128
							ps132.OverlayValues[129] = d129
							ps132.OverlayValues[130] = d130
							ps132.OverlayValues[131] = d131
							return bbs[9].RenderPS(ps132)
						}
						if ps.General {
						}
						ps133 := PhiState{General: ps.General}
						ps133.OverlayValues = make([]JITValueDesc, 132)
						ps133.OverlayValues[1] = d1
						ps133.OverlayValues[2] = d2
						ps133.OverlayValues[3] = d3
						ps133.OverlayValues[4] = d4
						ps133.OverlayValues[5] = d5
						ps133.OverlayValues[6] = d6
						ps133.OverlayValues[18] = d18
						ps133.OverlayValues[19] = d19
						ps133.OverlayValues[20] = d20
						ps133.OverlayValues[21] = d21
						ps133.OverlayValues[23] = d23
						ps133.OverlayValues[24] = d24
						ps133.OverlayValues[25] = d25
						ps133.OverlayValues[26] = d26
						ps133.OverlayValues[46] = d46
						ps133.OverlayValues[47] = d47
						ps133.OverlayValues[48] = d48
						ps133.OverlayValues[49] = d49
						ps133.OverlayValues[73] = d73
						ps133.OverlayValues[74] = d74
						ps133.OverlayValues[75] = d75
						ps133.OverlayValues[76] = d76
						ps133.OverlayValues[77] = d77
						ps133.OverlayValues[106] = d106
						ps133.OverlayValues[107] = d107
						ps133.OverlayValues[108] = d108
						ps133.OverlayValues[109] = d109
						ps133.OverlayValues[110] = d110
						ps133.OverlayValues[111] = d111
						ps133.OverlayValues[112] = d112
						ps133.OverlayValues[113] = d113
						ps133.OverlayValues[114] = d114
						ps133.OverlayValues[115] = d115
						ps133.OverlayValues[116] = d116
						ps133.OverlayValues[117] = d117
						ps133.OverlayValues[118] = d118
						ps133.OverlayValues[119] = d119
						ps133.OverlayValues[120] = d120
						ps133.OverlayValues[121] = d121
						ps133.OverlayValues[122] = d122
						ps133.OverlayValues[123] = d123
						ps133.OverlayValues[124] = d124
						ps133.OverlayValues[126] = d126
						ps133.OverlayValues[127] = d127
						ps133.OverlayValues[128] = d128
						ps133.OverlayValues[129] = d129
						ps133.OverlayValues[130] = d130
						ps133.OverlayValues[131] = d131
						return bbs[10].RenderPS(ps133)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d134 := ps.PhiValues[0]
							ctx.EnsureDesc(&d134)
							ctx.EmitStoreToStack(d134, int32(bbs[7].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d135 := ps.PhiValues[1]
							ctx.EnsureDesc(&d135)
							ctx.EmitStoreToStack(d135, int32(bbs[7].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d131.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl11)
					ps136 := PhiState{General: true}
					ps136.OverlayValues = make([]JITValueDesc, 136)
					ps136.OverlayValues[1] = d1
					ps136.OverlayValues[2] = d2
					ps136.OverlayValues[3] = d3
					ps136.OverlayValues[4] = d4
					ps136.OverlayValues[5] = d5
					ps136.OverlayValues[6] = d6
					ps136.OverlayValues[18] = d18
					ps136.OverlayValues[19] = d19
					ps136.OverlayValues[20] = d20
					ps136.OverlayValues[21] = d21
					ps136.OverlayValues[23] = d23
					ps136.OverlayValues[24] = d24
					ps136.OverlayValues[25] = d25
					ps136.OverlayValues[26] = d26
					ps136.OverlayValues[46] = d46
					ps136.OverlayValues[47] = d47
					ps136.OverlayValues[48] = d48
					ps136.OverlayValues[49] = d49
					ps136.OverlayValues[73] = d73
					ps136.OverlayValues[74] = d74
					ps136.OverlayValues[75] = d75
					ps136.OverlayValues[76] = d76
					ps136.OverlayValues[77] = d77
					ps136.OverlayValues[106] = d106
					ps136.OverlayValues[107] = d107
					ps136.OverlayValues[108] = d108
					ps136.OverlayValues[109] = d109
					ps136.OverlayValues[110] = d110
					ps136.OverlayValues[111] = d111
					ps136.OverlayValues[112] = d112
					ps136.OverlayValues[113] = d113
					ps136.OverlayValues[114] = d114
					ps136.OverlayValues[115] = d115
					ps136.OverlayValues[116] = d116
					ps136.OverlayValues[117] = d117
					ps136.OverlayValues[118] = d118
					ps136.OverlayValues[119] = d119
					ps136.OverlayValues[120] = d120
					ps136.OverlayValues[121] = d121
					ps136.OverlayValues[122] = d122
					ps136.OverlayValues[123] = d123
					ps136.OverlayValues[124] = d124
					ps136.OverlayValues[126] = d126
					ps136.OverlayValues[127] = d127
					ps136.OverlayValues[128] = d128
					ps136.OverlayValues[129] = d129
					ps136.OverlayValues[130] = d130
					ps136.OverlayValues[131] = d131
					ps136.OverlayValues[134] = d134
					ps136.OverlayValues[135] = d135
					ps137 := PhiState{General: true}
					ps137.OverlayValues = make([]JITValueDesc, 136)
					ps137.OverlayValues[1] = d1
					ps137.OverlayValues[2] = d2
					ps137.OverlayValues[3] = d3
					ps137.OverlayValues[4] = d4
					ps137.OverlayValues[5] = d5
					ps137.OverlayValues[6] = d6
					ps137.OverlayValues[18] = d18
					ps137.OverlayValues[19] = d19
					ps137.OverlayValues[20] = d20
					ps137.OverlayValues[21] = d21
					ps137.OverlayValues[23] = d23
					ps137.OverlayValues[24] = d24
					ps137.OverlayValues[25] = d25
					ps137.OverlayValues[26] = d26
					ps137.OverlayValues[46] = d46
					ps137.OverlayValues[47] = d47
					ps137.OverlayValues[48] = d48
					ps137.OverlayValues[49] = d49
					ps137.OverlayValues[73] = d73
					ps137.OverlayValues[74] = d74
					ps137.OverlayValues[75] = d75
					ps137.OverlayValues[76] = d76
					ps137.OverlayValues[77] = d77
					ps137.OverlayValues[106] = d106
					ps137.OverlayValues[107] = d107
					ps137.OverlayValues[108] = d108
					ps137.OverlayValues[109] = d109
					ps137.OverlayValues[110] = d110
					ps137.OverlayValues[111] = d111
					ps137.OverlayValues[112] = d112
					ps137.OverlayValues[113] = d113
					ps137.OverlayValues[114] = d114
					ps137.OverlayValues[115] = d115
					ps137.OverlayValues[116] = d116
					ps137.OverlayValues[117] = d117
					ps137.OverlayValues[118] = d118
					ps137.OverlayValues[119] = d119
					ps137.OverlayValues[120] = d120
					ps137.OverlayValues[121] = d121
					ps137.OverlayValues[122] = d122
					ps137.OverlayValues[123] = d123
					ps137.OverlayValues[124] = d124
					ps137.OverlayValues[126] = d126
					ps137.OverlayValues[127] = d127
					ps137.OverlayValues[128] = d128
					ps137.OverlayValues[129] = d129
					ps137.OverlayValues[130] = d130
					ps137.OverlayValues[131] = d131
					ps137.OverlayValues[134] = d134
					ps137.OverlayValues[135] = d135
					snap138 := d1
					snap139 := d2
					snap140 := d3
					snap141 := d4
					snap142 := d5
					snap143 := d6
					snap144 := d18
					snap145 := d19
					snap146 := d20
					snap147 := d21
					snap148 := d23
					snap149 := d24
					snap150 := d25
					snap151 := d26
					snap152 := d46
					snap153 := d47
					snap154 := d48
					snap155 := d49
					snap156 := d73
					snap157 := d74
					snap158 := d75
					snap159 := d76
					snap160 := d77
					snap161 := d106
					snap162 := d107
					snap163 := d108
					snap164 := d109
					snap165 := d110
					snap166 := d111
					snap167 := d112
					snap168 := d113
					snap169 := d114
					snap170 := d115
					snap171 := d116
					snap172 := d117
					snap173 := d118
					snap174 := d119
					snap175 := d120
					snap176 := d121
					snap177 := d122
					snap178 := d123
					snap179 := d124
					snap180 := d126
					snap181 := d127
					snap182 := d128
					snap183 := d129
					snap184 := d130
					snap185 := d131
					snap186 := d134
					snap187 := d135
					alloc188 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps137)
					}
					ctx.RestoreAllocState(alloc188)
					d1 = snap138
					d2 = snap139
					d3 = snap140
					d4 = snap141
					d5 = snap142
					d6 = snap143
					d18 = snap144
					d19 = snap145
					d20 = snap146
					d21 = snap147
					d23 = snap148
					d24 = snap149
					d25 = snap150
					d26 = snap151
					d46 = snap152
					d47 = snap153
					d48 = snap154
					d49 = snap155
					d73 = snap156
					d74 = snap157
					d75 = snap158
					d76 = snap159
					d77 = snap160
					d106 = snap161
					d107 = snap162
					d108 = snap163
					d109 = snap164
					d110 = snap165
					d111 = snap166
					d112 = snap167
					d113 = snap168
					d114 = snap169
					d115 = snap170
					d116 = snap171
					d117 = snap172
					d118 = snap173
					d119 = snap174
					d120 = snap175
					d121 = snap176
					d122 = snap177
					d123 = snap178
					d124 = snap179
					d126 = snap180
					d127 = snap181
					d128 = snap182
					d129 = snap183
					d130 = snap184
					d131 = snap185
					d134 = snap186
					d135 = snap187
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps136)
					}
					return result
					ctx.FreeDesc(&d130)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					ctx.ReclaimUntrackedRegs()
					d189 = args[0]
					d189.ID = 0
					var d190 JITValueDesc
					if d189.Loc == LocImm {
						d190 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d189.Imm.Int())}
					} else if d189.Type == tagInt && d189.Loc == LocRegPair {
						ctx.FreeReg(d189.Reg)
						d190 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d189.Reg2}
						ctx.BindReg(d189.Reg2, &d190)
						ctx.BindReg(d189.Reg2, &d190)
					} else if d189.Type == tagInt && d189.Loc == LocReg {
						d190 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d189.Reg}
						ctx.BindReg(d189.Reg, &d190)
						ctx.BindReg(d189.Reg, &d190)
					} else {
						d190 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d189}, 1)
						d190.Type = tagInt
						ctx.BindReg(d190.Reg, &d190)
					}
					ctx.StabilizeDescForControlFlow(&d190)
					ctx.FreeDesc(&d189)
					if ps.General {
						ctx.SyncDesc(&d190)
						if d190.Loc == LocReg {
							ctx.ProtectReg(d190.Reg)
						} else if d190.Loc == LocRegPair {
							ctx.ProtectReg(d190.Reg)
							ctx.ProtectReg(d190.Reg2)
						}
						d191 = d190
						if d191.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d191)
						ctx.EmitStoreToStack(d191, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
						if d190.Loc == LocReg {
							ctx.UnprotectReg(d190.Reg)
						} else if d190.Loc == LocRegPair {
							ctx.UnprotectReg(d190.Reg)
							ctx.UnprotectReg(d190.Reg2)
						}
					}
					ps192 := PhiState{General: ps.General}
					ps192.OverlayValues = make([]JITValueDesc, 192)
					ps192.OverlayValues[1] = d1
					ps192.OverlayValues[2] = d2
					ps192.OverlayValues[3] = d3
					ps192.OverlayValues[4] = d4
					ps192.OverlayValues[5] = d5
					ps192.OverlayValues[6] = d6
					ps192.OverlayValues[18] = d18
					ps192.OverlayValues[19] = d19
					ps192.OverlayValues[20] = d20
					ps192.OverlayValues[21] = d21
					ps192.OverlayValues[23] = d23
					ps192.OverlayValues[24] = d24
					ps192.OverlayValues[25] = d25
					ps192.OverlayValues[26] = d26
					ps192.OverlayValues[46] = d46
					ps192.OverlayValues[47] = d47
					ps192.OverlayValues[48] = d48
					ps192.OverlayValues[49] = d49
					ps192.OverlayValues[73] = d73
					ps192.OverlayValues[74] = d74
					ps192.OverlayValues[75] = d75
					ps192.OverlayValues[76] = d76
					ps192.OverlayValues[77] = d77
					ps192.OverlayValues[106] = d106
					ps192.OverlayValues[107] = d107
					ps192.OverlayValues[108] = d108
					ps192.OverlayValues[109] = d109
					ps192.OverlayValues[110] = d110
					ps192.OverlayValues[111] = d111
					ps192.OverlayValues[112] = d112
					ps192.OverlayValues[113] = d113
					ps192.OverlayValues[114] = d114
					ps192.OverlayValues[115] = d115
					ps192.OverlayValues[116] = d116
					ps192.OverlayValues[117] = d117
					ps192.OverlayValues[118] = d118
					ps192.OverlayValues[119] = d119
					ps192.OverlayValues[120] = d120
					ps192.OverlayValues[121] = d121
					ps192.OverlayValues[122] = d122
					ps192.OverlayValues[123] = d123
					ps192.OverlayValues[124] = d124
					ps192.OverlayValues[126] = d126
					ps192.OverlayValues[127] = d127
					ps192.OverlayValues[128] = d128
					ps192.OverlayValues[129] = d129
					ps192.OverlayValues[130] = d130
					ps192.OverlayValues[131] = d131
					ps192.OverlayValues[134] = d134
					ps192.OverlayValues[135] = d135
					ps192.OverlayValues[189] = d189
					ps192.OverlayValues[190] = d190
					ps192.OverlayValues[191] = d191
					ps192.PhiValues = make([]JITValueDesc, 2)
					d193 = d190
					ps192.PhiValues[0] = d193
					d194 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps192.PhiValues[1] = d194
					if ps192.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps192)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d195 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d195.Loc == LocRegPair || d195.Loc == LocStackPair || d195.Loc == LocRegTriple || d195.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d195)
					d196 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d1, d195}, 3)
					d196.NoHeapPointer = false
					ctx.BindReg(d196.Reg, &d196)
					ctx.BindReg(d196.Reg2, &d196)
					ctx.BindReg(d196.Reg3, &d196)
					ctx.FreeDesc(&d195)
					ctx.EnsureDesc(&d196)
					ctx.EnsureDesc(&d196)
					ctx.EnsureDesc(&d196)
					if d196.Loc != LocRegTriple && d196.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d196)
					d197 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d196}, 3)
					d197.NoHeapPointer = false
					ctx.BindReg(d197.Reg, &d197)
					ctx.BindReg(d197.Reg2, &d197)
					ctx.BindReg(d197.Reg3, &d197)
					ctx.FreeDesc(&d196)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocRegTriple && d197.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d197)
					d198 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d197}, 1)
					d198.NoHeapPointer = true
					ctx.BindReg(d198.Reg, &d198)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocRegTriple && d197.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d197)
					d199 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d197}, 1)
					d199.NoHeapPointer = true
					ctx.BindReg(d199.Reg, &d199)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocRegTriple && d197.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d197)
					d200 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d197}, 1)
					d200.NoHeapPointer = true
					ctx.BindReg(d200.Reg, &d200)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocRegTriple && d197.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d197)
					d201 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d197}, 1)
					d201.NoHeapPointer = true
					ctx.BindReg(d201.Reg, &d201)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocRegTriple && d197.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d197)
					d202 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d197}, 1)
					d202.NoHeapPointer = true
					ctx.BindReg(d202.Reg, &d202)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocRegTriple && d197.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d197)
					d203 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d197}, 1)
					d203.NoHeapPointer = true
					ctx.BindReg(d203.Reg, &d203)
					ctx.FreeDesc(&d197)
					ctx.EnsureDesc(&d198)
					ctx.EnsureDesc(&d198)
					if d198.Loc == LocRegPair || d198.Loc == LocStackPair || d198.Loc == LocRegTriple || d198.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d199)
					ctx.EnsureDesc(&d199)
					if d199.Loc == LocRegPair || d199.Loc == LocStackPair || d199.Loc == LocRegTriple || d199.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d200)
					ctx.EnsureDesc(&d200)
					if d200.Loc == LocRegPair || d200.Loc == LocStackPair || d200.Loc == LocRegTriple || d200.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d201)
					ctx.EnsureDesc(&d201)
					if d201.Loc == LocRegPair || d201.Loc == LocStackPair || d201.Loc == LocRegTriple || d201.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d202)
					ctx.EnsureDesc(&d202)
					if d202.Loc == LocRegPair || d202.Loc == LocStackPair || d202.Loc == LocRegTriple || d202.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc == LocRegPair || d203.Loc == LocStackPair || d203.Loc == LocRegTriple || d203.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d204 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d204.Loc == LocRegPair || d204.Loc == LocStackPair || d204.Loc == LocRegTriple || d204.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d198)
					ctx.SyncDesc(&d199)
					ctx.SyncDesc(&d200)
					ctx.SyncDesc(&d201)
					ctx.SyncDesc(&d202)
					ctx.SyncDesc(&d203)
					ctx.SyncDesc(&d204)
					ctx.SyncDesc(&d23)
					d205 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d198, d199, d200, d201, d202, d203, d204, d23}, 3)
					d205.NoHeapPointer = false
					ctx.BindReg(d205.Reg, &d205)
					ctx.BindReg(d205.Reg2, &d205)
					ctx.BindReg(d205.Reg3, &d205)
					ctx.FreeDesc(&d204)
					ctx.FreeDesc(&d198)
					ctx.FreeDesc(&d199)
					ctx.FreeDesc(&d200)
					ctx.FreeDesc(&d201)
					ctx.FreeDesc(&d202)
					ctx.FreeDesc(&d203)
					ctx.EnsureDesc(&d205)
					ctx.EnsureDesc(&d205)
					ctx.EnsureDesc(&d205)
					if d205.Loc != LocRegTriple && d205.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d205)
					d206 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d205}, 3)
					d206.NoHeapPointer = false
					ctx.BindReg(d206.Reg, &d206)
					ctx.BindReg(d206.Reg2, &d206)
					ctx.BindReg(d206.Reg3, &d206)
					ctx.FreeDesc(&d205)
					ctx.EnsureDesc(&d206)
					ctx.EnsureDesc(&d206)
					ctx.EnsureDesc(&d206)
					if d206.Loc != LocRegTriple && d206.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d206)
					d207 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d206}, 1)
					d207.NoHeapPointer = true
					ctx.BindReg(d207.Reg, &d207)
					ctx.FreeDesc(&d206)
					ctx.EnsureDesc(&d207)
					ctx.EnsureDesc(&d207)
					if d207.Loc == LocRegPair || d207.Loc == LocStackPair || d207.Loc == LocRegTriple || d207.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d207)
					d208 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d207}, 2)
					d208.NoHeapPointer = false
					ctx.BindReg(d208.Reg, &d208)
					ctx.BindReg(d208.Reg2, &d208)
					ctx.FreeDesc(&d207)
					ctx.SyncDesc(&d208)
					if d208.Loc == LocRegPair || d208.Loc == LocStackPair || d208.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d208, &result)
						result.Type = d208.Type
					} else {
						switch d208.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d208)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d208)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d208)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d208, &result)
							result.Type = d208.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d209 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d209.Loc == LocRegPair || d209.Loc == LocStackPair || d209.Loc == LocRegTriple || d209.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d209)
					d210 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d1, d209}, 3)
					d210.NoHeapPointer = false
					ctx.BindReg(d210.Reg, &d210)
					ctx.BindReg(d210.Reg2, &d210)
					ctx.BindReg(d210.Reg3, &d210)
					ctx.FreeDesc(&d209)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d210)
					ctx.EnsureDesc(&d210)
					ctx.EnsureDesc(&d210)
					if d210.Loc != LocRegTriple && d210.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d210)
					ctx.SyncDesc(&d23)
					d211 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d210, d23}, 3)
					d211.NoHeapPointer = false
					ctx.BindReg(d211.Reg, &d211)
					ctx.BindReg(d211.Reg2, &d211)
					ctx.BindReg(d211.Reg3, &d211)
					ctx.FreeDesc(&d210)
					ctx.FreeDesc(&d23)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocRegTriple && d211.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d211)
					d212 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d211}, 1)
					d212.NoHeapPointer = true
					ctx.BindReg(d212.Reg, &d212)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocRegTriple && d211.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d211)
					d213 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d211}, 1)
					d213.NoHeapPointer = true
					ctx.BindReg(d213.Reg, &d213)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocRegTriple && d211.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d211)
					d214 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d211}, 1)
					d214.NoHeapPointer = true
					ctx.BindReg(d214.Reg, &d214)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocRegTriple && d211.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d211)
					d215 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d211}, 1)
					d215.NoHeapPointer = true
					ctx.BindReg(d215.Reg, &d215)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocRegTriple && d211.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d211)
					d216 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d211}, 1)
					d216.NoHeapPointer = true
					ctx.BindReg(d216.Reg, &d216)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					if d211.Loc != LocRegTriple && d211.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d211)
					d217 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d211}, 1)
					d217.NoHeapPointer = true
					ctx.BindReg(d217.Reg, &d217)
					ctx.FreeDesc(&d211)
					d218 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.EnsureDesc(&d212)
					ctx.EnsureDesc(&d212)
					if d212.Loc == LocRegPair || d212.Loc == LocStackPair || d212.Loc == LocRegTriple || d212.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d213)
					ctx.EnsureDesc(&d213)
					if d213.Loc == LocRegPair || d213.Loc == LocStackPair || d213.Loc == LocRegTriple || d213.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d214)
					ctx.EnsureDesc(&d214)
					if d214.Loc == LocRegPair || d214.Loc == LocStackPair || d214.Loc == LocRegTriple || d214.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d215)
					ctx.EnsureDesc(&d215)
					if d215.Loc == LocRegPair || d215.Loc == LocStackPair || d215.Loc == LocRegTriple || d215.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d216)
					ctx.EnsureDesc(&d216)
					if d216.Loc == LocRegPair || d216.Loc == LocStackPair || d216.Loc == LocRegTriple || d216.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d217)
					ctx.EnsureDesc(&d217)
					if d217.Loc == LocRegPair || d217.Loc == LocStackPair || d217.Loc == LocRegTriple || d217.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d219 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d219.Loc == LocRegPair || d219.Loc == LocStackPair || d219.Loc == LocRegTriple || d219.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d218)
					ctx.EnsureDesc(&d218)
					if d218.Loc == LocRegPair || d218.Loc == LocStackPair || d218.Loc == LocRegTriple || d218.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d212)
					ctx.SyncDesc(&d213)
					ctx.SyncDesc(&d214)
					ctx.SyncDesc(&d215)
					ctx.SyncDesc(&d216)
					ctx.SyncDesc(&d217)
					ctx.SyncDesc(&d219)
					ctx.SyncDesc(&d218)
					d220 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d212, d213, d214, d215, d216, d217, d219, d218}, 3)
					d220.NoHeapPointer = false
					ctx.BindReg(d220.Reg, &d220)
					ctx.BindReg(d220.Reg2, &d220)
					ctx.BindReg(d220.Reg3, &d220)
					ctx.FreeDesc(&d219)
					ctx.FreeDesc(&d212)
					ctx.FreeDesc(&d213)
					ctx.FreeDesc(&d214)
					ctx.FreeDesc(&d215)
					ctx.FreeDesc(&d216)
					ctx.FreeDesc(&d217)
					ctx.FreeDesc(&d218)
					ctx.EnsureDesc(&d220)
					ctx.EnsureDesc(&d220)
					ctx.EnsureDesc(&d220)
					if d220.Loc != LocRegTriple && d220.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d220)
					d221 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d220}, 1)
					d221.NoHeapPointer = true
					ctx.BindReg(d221.Reg, &d221)
					ctx.FreeDesc(&d220)
					ctx.EnsureDesc(&d221)
					ctx.EnsureDesc(&d221)
					if d221.Loc == LocRegPair || d221.Loc == LocStackPair || d221.Loc == LocRegTriple || d221.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d221)
					d222 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d221}, 2)
					d222.NoHeapPointer = false
					ctx.BindReg(d222.Reg, &d222)
					ctx.BindReg(d222.Reg2, &d222)
					ctx.FreeDesc(&d221)
					ctx.SyncDesc(&d222)
					if d222.Loc == LocRegPair || d222.Loc == LocStackPair || d222.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d222, &result)
						result.Type = d222.Type
					} else {
						switch d222.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d222)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d222)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d222)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d222, &result)
							result.Type = d222.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps223 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps223)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
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
				var d106 JITValueDesc
				_ = d106
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
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
				var d154 JITValueDesc
				_ = d154
				var d155 JITValueDesc
				_ = d155
				var d156 JITValueDesc
				_ = d156
				var d157 JITValueDesc
				_ = d157
				var d158 JITValueDesc
				_ = d158
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d217 JITValueDesc
				_ = d217
				var d218 JITValueDesc
				_ = d218
				var d219 JITValueDesc
				_ = d219
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d222 JITValueDesc
				_ = d222
				var d287 JITValueDesc
				_ = d287
				var d288 JITValueDesc
				_ = d288
				var d289 JITValueDesc
				_ = d289
				var d290 JITValueDesc
				_ = d290
				var d291 JITValueDesc
				_ = d291
				var d292 JITValueDesc
				_ = d292
				var d293 JITValueDesc
				_ = d293
				var d294 JITValueDesc
				_ = d294
				var d295 JITValueDesc
				_ = d295
				var d296 JITValueDesc
				_ = d296
				var d297 JITValueDesc
				_ = d297
				var d298 JITValueDesc
				_ = d298
				var d299 JITValueDesc
				_ = d299
				var d377 JITValueDesc
				_ = d377
				var d378 JITValueDesc
				_ = d378
				var d379 JITValueDesc
				_ = d379
				var d380 JITValueDesc
				_ = d380
				var d381 JITValueDesc
				_ = d381
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
				var d481 JITValueDesc
				_ = d481
				var d482 JITValueDesc
				_ = d482
				var d483 JITValueDesc
				_ = d483
				var d485 JITValueDesc
				_ = d485
				var d486 JITValueDesc
				_ = d486
				var d487 JITValueDesc
				_ = d487
				var d488 JITValueDesc
				_ = d488
				var d489 JITValueDesc
				_ = d489
				var d490 JITValueDesc
				_ = d490
				var d491 JITValueDesc
				_ = d491
				var d492 JITValueDesc
				_ = d492
				var d493 JITValueDesc
				_ = d493
				var d494 JITValueDesc
				_ = d494
				var d495 JITValueDesc
				_ = d495
				var d496 JITValueDesc
				_ = d496
				var d497 JITValueDesc
				_ = d497
				var d605 JITValueDesc
				_ = d605
				var d606 JITValueDesc
				_ = d606
				var d607 JITValueDesc
				_ = d607
				var d609 JITValueDesc
				_ = d609
				var d610 JITValueDesc
				_ = d610
				var d611 JITValueDesc
				_ = d611
				var d612 JITValueDesc
				_ = d612
				var d613 JITValueDesc
				_ = d613
				var d614 JITValueDesc
				_ = d614
				var d615 JITValueDesc
				_ = d615
				var d616 JITValueDesc
				_ = d616
				var d617 JITValueDesc
				_ = d617
				var d618 JITValueDesc
				_ = d618
				var d738 JITValueDesc
				_ = d738
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
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
							if ps.General {
							}
							ps4 := PhiState{General: ps.General}
							ps4.OverlayValues = make([]JITValueDesc, 4)
							ps4.OverlayValues[0] = d0
							ps4.OverlayValues[1] = d1
							ps4.OverlayValues[2] = d2
							ps4.OverlayValues[3] = d3
							return bbs[1].RenderPS(ps4)
						}
						if ps.General {
						}
						ps5 := PhiState{General: ps.General}
						ps5.OverlayValues = make([]JITValueDesc, 4)
						ps5.OverlayValues[0] = d0
						ps5.OverlayValues[1] = d1
						ps5.OverlayValues[2] = d2
						ps5.OverlayValues[3] = d3
						return bbs[3].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl4)
					ps6 := PhiState{General: true}
					ps6.OverlayValues = make([]JITValueDesc, 4)
					ps6.OverlayValues[0] = d0
					ps6.OverlayValues[1] = d1
					ps6.OverlayValues[2] = d2
					ps6.OverlayValues[3] = d3
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 4)
					ps7.OverlayValues[0] = d0
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					snap8 := d0
					snap9 := d1
					snap10 := d2
					snap11 := d3
					alloc12 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps7)
					}
					ctx.RestoreAllocState(alloc12)
					d0 = snap8
					d1 = snap9
					d2 = snap10
					d3 = snap11
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps6)
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d13, &result)
						result.Type = d13.Type
					} else {
						switch d13.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d13)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d13)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d13)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d13, &result)
							result.Type = d13.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					ctx.ReclaimUntrackedRegs()
					d14 = args[1]
					d14.ID = 0
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					d14 = JITPrepareScmerGoArg(ctx, d14)
					ctx.SyncDesc(&d14)
					callResults15 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d14}, []uint8{3, 1}, []uint8{4, 0})
					d16 = callResults15[0]
					_ = d16
					d17 = callResults15[1]
					_ = d17
					ctx.FreeDesc(&d14)
					ctx.StabilizeDescForControlFlow(&d16)
					d18 = args[2]
					d18.ID = 0
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					d18 = JITPrepareScmerGoArg(ctx, d18)
					ctx.SyncDesc(&d18)
					callResults19 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d18}, []uint8{3, 1}, []uint8{4, 0})
					d20 = callResults19[0]
					_ = d20
					d21 = callResults19[1]
					_ = d21
					ctx.FreeDesc(&d18)
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.StabilizeDescForControlFlow(&d21)
					d22 = d17
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							if ps.General {
							}
							ps23 := PhiState{General: ps.General}
							ps23.OverlayValues = make([]JITValueDesc, 23)
							ps23.OverlayValues[0] = d0
							ps23.OverlayValues[1] = d1
							ps23.OverlayValues[2] = d2
							ps23.OverlayValues[3] = d3
							ps23.OverlayValues[13] = d13
							ps23.OverlayValues[14] = d14
							ps23.OverlayValues[16] = d16
							ps23.OverlayValues[17] = d17
							ps23.OverlayValues[18] = d18
							ps23.OverlayValues[20] = d20
							ps23.OverlayValues[21] = d21
							ps23.OverlayValues[22] = d22
							return bbs[6].RenderPS(ps23)
						}
						if ps.General {
						}
						ps24 := PhiState{General: ps.General}
						ps24.OverlayValues = make([]JITValueDesc, 23)
						ps24.OverlayValues[0] = d0
						ps24.OverlayValues[1] = d1
						ps24.OverlayValues[2] = d2
						ps24.OverlayValues[3] = d3
						ps24.OverlayValues[13] = d13
						ps24.OverlayValues[14] = d14
						ps24.OverlayValues[16] = d16
						ps24.OverlayValues[17] = d17
						ps24.OverlayValues[18] = d18
						ps24.OverlayValues[20] = d20
						ps24.OverlayValues[21] = d21
						ps24.OverlayValues[22] = d22
						return bbs[4].RenderPS(ps24)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl5)
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 23)
					ps25.OverlayValues[0] = d0
					ps25.OverlayValues[1] = d1
					ps25.OverlayValues[2] = d2
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[13] = d13
					ps25.OverlayValues[14] = d14
					ps25.OverlayValues[16] = d16
					ps25.OverlayValues[17] = d17
					ps25.OverlayValues[18] = d18
					ps25.OverlayValues[20] = d20
					ps25.OverlayValues[21] = d21
					ps25.OverlayValues[22] = d22
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 23)
					ps26.OverlayValues[0] = d0
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[2] = d2
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[13] = d13
					ps26.OverlayValues[14] = d14
					ps26.OverlayValues[16] = d16
					ps26.OverlayValues[17] = d17
					ps26.OverlayValues[18] = d18
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					snap27 := d0
					snap28 := d1
					snap29 := d2
					snap30 := d3
					snap31 := d13
					snap32 := d14
					snap33 := d16
					snap34 := d17
					snap35 := d18
					snap36 := d20
					snap37 := d21
					snap38 := d22
					alloc39 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps26)
					}
					ctx.RestoreAllocState(alloc39)
					d0 = snap27
					d1 = snap28
					d2 = snap29
					d3 = snap30
					d13 = snap31
					d14 = snap32
					d16 = snap33
					d17 = snap34
					d18 = snap35
					d20 = snap36
					d21 = snap37
					d22 = snap38
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps25)
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					ctx.ReclaimUntrackedRegs()
					d40 = args[2]
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
							if ps.General {
							}
							ps44 := PhiState{General: ps.General}
							ps44.OverlayValues = make([]JITValueDesc, 44)
							ps44.OverlayValues[0] = d0
							ps44.OverlayValues[1] = d1
							ps44.OverlayValues[2] = d2
							ps44.OverlayValues[3] = d3
							ps44.OverlayValues[13] = d13
							ps44.OverlayValues[14] = d14
							ps44.OverlayValues[16] = d16
							ps44.OverlayValues[17] = d17
							ps44.OverlayValues[18] = d18
							ps44.OverlayValues[20] = d20
							ps44.OverlayValues[21] = d21
							ps44.OverlayValues[22] = d22
							ps44.OverlayValues[40] = d40
							ps44.OverlayValues[41] = d41
							ps44.OverlayValues[42] = d42
							ps44.OverlayValues[43] = d43
							return bbs[1].RenderPS(ps44)
						}
						if ps.General {
						}
						ps45 := PhiState{General: ps.General}
						ps45.OverlayValues = make([]JITValueDesc, 44)
						ps45.OverlayValues[0] = d0
						ps45.OverlayValues[1] = d1
						ps45.OverlayValues[2] = d2
						ps45.OverlayValues[3] = d3
						ps45.OverlayValues[13] = d13
						ps45.OverlayValues[14] = d14
						ps45.OverlayValues[16] = d16
						ps45.OverlayValues[17] = d17
						ps45.OverlayValues[18] = d18
						ps45.OverlayValues[20] = d20
						ps45.OverlayValues[21] = d21
						ps45.OverlayValues[22] = d22
						ps45.OverlayValues[40] = d40
						ps45.OverlayValues[41] = d41
						ps45.OverlayValues[42] = d42
						ps45.OverlayValues[43] = d43
						return bbs[2].RenderPS(ps45)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d43.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl3)
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 44)
					ps46.OverlayValues[0] = d0
					ps46.OverlayValues[1] = d1
					ps46.OverlayValues[2] = d2
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[16] = d16
					ps46.OverlayValues[17] = d17
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[21] = d21
					ps46.OverlayValues[22] = d22
					ps46.OverlayValues[40] = d40
					ps46.OverlayValues[41] = d41
					ps46.OverlayValues[42] = d42
					ps46.OverlayValues[43] = d43
					ps47 := PhiState{General: true}
					ps47.OverlayValues = make([]JITValueDesc, 44)
					ps47.OverlayValues[0] = d0
					ps47.OverlayValues[1] = d1
					ps47.OverlayValues[2] = d2
					ps47.OverlayValues[3] = d3
					ps47.OverlayValues[13] = d13
					ps47.OverlayValues[14] = d14
					ps47.OverlayValues[16] = d16
					ps47.OverlayValues[17] = d17
					ps47.OverlayValues[18] = d18
					ps47.OverlayValues[20] = d20
					ps47.OverlayValues[21] = d21
					ps47.OverlayValues[22] = d22
					ps47.OverlayValues[40] = d40
					ps47.OverlayValues[41] = d41
					ps47.OverlayValues[42] = d42
					ps47.OverlayValues[43] = d43
					snap48 := d0
					snap49 := d1
					snap50 := d2
					snap51 := d3
					snap52 := d13
					snap53 := d14
					snap54 := d16
					snap55 := d17
					snap56 := d18
					snap57 := d20
					snap58 := d21
					snap59 := d22
					snap60 := d40
					snap61 := d41
					snap62 := d42
					snap63 := d43
					alloc64 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps47)
					}
					ctx.RestoreAllocState(alloc64)
					d0 = snap48
					d1 = snap49
					d2 = snap50
					d3 = snap51
					d13 = snap52
					d14 = snap53
					d16 = snap54
					d17 = snap55
					d18 = snap56
					d20 = snap57
					d21 = snap58
					d22 = snap59
					d40 = snap60
					d41 = snap61
					d42 = snap62
					d43 = snap63
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps46)
					}
					return result
					ctx.FreeDesc(&d41)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					ctx.ReclaimUntrackedRegs()
					d65 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d65)
					if d65.Loc == LocRegPair || d65.Loc == LocStackPair || d65.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d65, &result)
						result.Type = d65.Type
					} else {
						switch d65.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d65)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d65)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d65)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d65, &result)
							result.Type = d65.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
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
					ctx.FreeDesc(&d66)
					ctx.EnsureDesc(&d67)
					ctx.EnsureDesc(&d67)
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
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d67)
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d67}, 2)
					d69.NoHeapPointer = false
					ctx.BindReg(d69.Reg, &d69)
					ctx.BindReg(d69.Reg2, &d69)
					ctx.StabilizeDescForControlFlow(&d69)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocRegTriple && d20.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d16)
					d70 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d20, d16}, 1)
					d70.NoHeapPointer = true
					ctx.BindReg(d70.Reg, &d70)
					ctx.StabilizeDescForControlFlow(&d70)
					ctx.EnsureDesc(&d69)
					d71 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					var d72 JITValueDesc
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
					d73 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d72}, 1)
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
							if ps.General {
							}
							ps75 := PhiState{General: ps.General}
							ps75.OverlayValues = make([]JITValueDesc, 75)
							ps75.OverlayValues[0] = d0
							ps75.OverlayValues[1] = d1
							ps75.OverlayValues[2] = d2
							ps75.OverlayValues[3] = d3
							ps75.OverlayValues[13] = d13
							ps75.OverlayValues[14] = d14
							ps75.OverlayValues[16] = d16
							ps75.OverlayValues[17] = d17
							ps75.OverlayValues[18] = d18
							ps75.OverlayValues[20] = d20
							ps75.OverlayValues[21] = d21
							ps75.OverlayValues[22] = d22
							ps75.OverlayValues[40] = d40
							ps75.OverlayValues[41] = d41
							ps75.OverlayValues[42] = d42
							ps75.OverlayValues[43] = d43
							ps75.OverlayValues[65] = d65
							ps75.OverlayValues[66] = d66
							ps75.OverlayValues[67] = d67
							ps75.OverlayValues[68] = d68
							ps75.OverlayValues[69] = d69
							ps75.OverlayValues[70] = d70
							ps75.OverlayValues[71] = d71
							ps75.OverlayValues[72] = d72
							ps75.OverlayValues[73] = d73
							ps75.OverlayValues[74] = d74
							return bbs[7].RenderPS(ps75)
						}
						if ps.General {
						}
						ps76 := PhiState{General: ps.General}
						ps76.OverlayValues = make([]JITValueDesc, 75)
						ps76.OverlayValues[0] = d0
						ps76.OverlayValues[1] = d1
						ps76.OverlayValues[2] = d2
						ps76.OverlayValues[3] = d3
						ps76.OverlayValues[13] = d13
						ps76.OverlayValues[14] = d14
						ps76.OverlayValues[16] = d16
						ps76.OverlayValues[17] = d17
						ps76.OverlayValues[18] = d18
						ps76.OverlayValues[20] = d20
						ps76.OverlayValues[21] = d21
						ps76.OverlayValues[22] = d22
						ps76.OverlayValues[40] = d40
						ps76.OverlayValues[41] = d41
						ps76.OverlayValues[42] = d42
						ps76.OverlayValues[43] = d43
						ps76.OverlayValues[65] = d65
						ps76.OverlayValues[66] = d66
						ps76.OverlayValues[67] = d67
						ps76.OverlayValues[68] = d68
						ps76.OverlayValues[69] = d69
						ps76.OverlayValues[70] = d70
						ps76.OverlayValues[71] = d71
						ps76.OverlayValues[72] = d72
						ps76.OverlayValues[73] = d73
						ps76.OverlayValues[74] = d74
						return bbs[9].RenderPS(ps76)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d74.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl28)
					ctx.EmitJmp(lbl29)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl10)
					ps77 := PhiState{General: true}
					ps77.OverlayValues = make([]JITValueDesc, 75)
					ps77.OverlayValues[0] = d0
					ps77.OverlayValues[1] = d1
					ps77.OverlayValues[2] = d2
					ps77.OverlayValues[3] = d3
					ps77.OverlayValues[13] = d13
					ps77.OverlayValues[14] = d14
					ps77.OverlayValues[16] = d16
					ps77.OverlayValues[17] = d17
					ps77.OverlayValues[18] = d18
					ps77.OverlayValues[20] = d20
					ps77.OverlayValues[21] = d21
					ps77.OverlayValues[22] = d22
					ps77.OverlayValues[40] = d40
					ps77.OverlayValues[41] = d41
					ps77.OverlayValues[42] = d42
					ps77.OverlayValues[43] = d43
					ps77.OverlayValues[65] = d65
					ps77.OverlayValues[66] = d66
					ps77.OverlayValues[67] = d67
					ps77.OverlayValues[68] = d68
					ps77.OverlayValues[69] = d69
					ps77.OverlayValues[70] = d70
					ps77.OverlayValues[71] = d71
					ps77.OverlayValues[72] = d72
					ps77.OverlayValues[73] = d73
					ps77.OverlayValues[74] = d74
					ps78 := PhiState{General: true}
					ps78.OverlayValues = make([]JITValueDesc, 75)
					ps78.OverlayValues[0] = d0
					ps78.OverlayValues[1] = d1
					ps78.OverlayValues[2] = d2
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[13] = d13
					ps78.OverlayValues[14] = d14
					ps78.OverlayValues[16] = d16
					ps78.OverlayValues[17] = d17
					ps78.OverlayValues[18] = d18
					ps78.OverlayValues[20] = d20
					ps78.OverlayValues[21] = d21
					ps78.OverlayValues[22] = d22
					ps78.OverlayValues[40] = d40
					ps78.OverlayValues[41] = d41
					ps78.OverlayValues[42] = d42
					ps78.OverlayValues[43] = d43
					ps78.OverlayValues[65] = d65
					ps78.OverlayValues[66] = d66
					ps78.OverlayValues[67] = d67
					ps78.OverlayValues[68] = d68
					ps78.OverlayValues[69] = d69
					ps78.OverlayValues[70] = d70
					ps78.OverlayValues[71] = d71
					ps78.OverlayValues[72] = d72
					ps78.OverlayValues[73] = d73
					ps78.OverlayValues[74] = d74
					snap79 := d0
					snap80 := d1
					snap81 := d2
					snap82 := d3
					snap83 := d13
					snap84 := d14
					snap85 := d16
					snap86 := d17
					snap87 := d18
					snap88 := d20
					snap89 := d21
					snap90 := d22
					snap91 := d40
					snap92 := d41
					snap93 := d42
					snap94 := d43
					snap95 := d65
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
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps78)
					}
					ctx.RestoreAllocState(alloc105)
					d0 = snap79
					d1 = snap80
					d2 = snap81
					d3 = snap82
					d13 = snap83
					d14 = snap84
					d16 = snap85
					d17 = snap86
					d18 = snap87
					d20 = snap88
					d21 = snap89
					d22 = snap90
					d40 = snap91
					d41 = snap92
					d42 = snap93
					d43 = snap94
					d65 = snap95
					d66 = snap96
					d67 = snap97
					d68 = snap98
					d69 = snap99
					d70 = snap100
					d71 = snap101
					d72 = snap102
					d73 = snap103
					d74 = snap104
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps77)
					}
					return result
					ctx.FreeDesc(&d73)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					ctx.ReclaimUntrackedRegs()
					d106 = d21
					ctx.EnsureDesc(&d106)
					if d106.Loc != LocImm && d106.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d106.Loc == LocImm {
						if d106.Imm.Bool() {
							if ps.General {
							}
							ps107 := PhiState{General: ps.General}
							ps107.OverlayValues = make([]JITValueDesc, 107)
							ps107.OverlayValues[0] = d0
							ps107.OverlayValues[1] = d1
							ps107.OverlayValues[2] = d2
							ps107.OverlayValues[3] = d3
							ps107.OverlayValues[13] = d13
							ps107.OverlayValues[14] = d14
							ps107.OverlayValues[16] = d16
							ps107.OverlayValues[17] = d17
							ps107.OverlayValues[18] = d18
							ps107.OverlayValues[20] = d20
							ps107.OverlayValues[21] = d21
							ps107.OverlayValues[22] = d22
							ps107.OverlayValues[40] = d40
							ps107.OverlayValues[41] = d41
							ps107.OverlayValues[42] = d42
							ps107.OverlayValues[43] = d43
							ps107.OverlayValues[65] = d65
							ps107.OverlayValues[66] = d66
							ps107.OverlayValues[67] = d67
							ps107.OverlayValues[68] = d68
							ps107.OverlayValues[69] = d69
							ps107.OverlayValues[70] = d70
							ps107.OverlayValues[71] = d71
							ps107.OverlayValues[72] = d72
							ps107.OverlayValues[73] = d73
							ps107.OverlayValues[74] = d74
							ps107.OverlayValues[106] = d106
							return bbs[5].RenderPS(ps107)
						}
						if ps.General {
						}
						ps108 := PhiState{General: ps.General}
						ps108.OverlayValues = make([]JITValueDesc, 107)
						ps108.OverlayValues[0] = d0
						ps108.OverlayValues[1] = d1
						ps108.OverlayValues[2] = d2
						ps108.OverlayValues[3] = d3
						ps108.OverlayValues[13] = d13
						ps108.OverlayValues[14] = d14
						ps108.OverlayValues[16] = d16
						ps108.OverlayValues[17] = d17
						ps108.OverlayValues[18] = d18
						ps108.OverlayValues[20] = d20
						ps108.OverlayValues[21] = d21
						ps108.OverlayValues[22] = d22
						ps108.OverlayValues[40] = d40
						ps108.OverlayValues[41] = d41
						ps108.OverlayValues[42] = d42
						ps108.OverlayValues[43] = d43
						ps108.OverlayValues[65] = d65
						ps108.OverlayValues[66] = d66
						ps108.OverlayValues[67] = d67
						ps108.OverlayValues[68] = d68
						ps108.OverlayValues[69] = d69
						ps108.OverlayValues[70] = d70
						ps108.OverlayValues[71] = d71
						ps108.OverlayValues[72] = d72
						ps108.OverlayValues[73] = d73
						ps108.OverlayValues[74] = d74
						ps108.OverlayValues[106] = d106
						return bbs[4].RenderPS(ps108)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl30 := ctx.ReserveLabel()
					lbl31 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d106.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl30)
					ctx.EmitJmp(lbl31)
					ctx.MarkLabel(lbl30)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl5)
					ps109 := PhiState{General: true}
					ps109.OverlayValues = make([]JITValueDesc, 107)
					ps109.OverlayValues[0] = d0
					ps109.OverlayValues[1] = d1
					ps109.OverlayValues[2] = d2
					ps109.OverlayValues[3] = d3
					ps109.OverlayValues[13] = d13
					ps109.OverlayValues[14] = d14
					ps109.OverlayValues[16] = d16
					ps109.OverlayValues[17] = d17
					ps109.OverlayValues[18] = d18
					ps109.OverlayValues[20] = d20
					ps109.OverlayValues[21] = d21
					ps109.OverlayValues[22] = d22
					ps109.OverlayValues[40] = d40
					ps109.OverlayValues[41] = d41
					ps109.OverlayValues[42] = d42
					ps109.OverlayValues[43] = d43
					ps109.OverlayValues[65] = d65
					ps109.OverlayValues[66] = d66
					ps109.OverlayValues[67] = d67
					ps109.OverlayValues[68] = d68
					ps109.OverlayValues[69] = d69
					ps109.OverlayValues[70] = d70
					ps109.OverlayValues[71] = d71
					ps109.OverlayValues[72] = d72
					ps109.OverlayValues[73] = d73
					ps109.OverlayValues[74] = d74
					ps109.OverlayValues[106] = d106
					ps110 := PhiState{General: true}
					ps110.OverlayValues = make([]JITValueDesc, 107)
					ps110.OverlayValues[0] = d0
					ps110.OverlayValues[1] = d1
					ps110.OverlayValues[2] = d2
					ps110.OverlayValues[3] = d3
					ps110.OverlayValues[13] = d13
					ps110.OverlayValues[14] = d14
					ps110.OverlayValues[16] = d16
					ps110.OverlayValues[17] = d17
					ps110.OverlayValues[18] = d18
					ps110.OverlayValues[20] = d20
					ps110.OverlayValues[21] = d21
					ps110.OverlayValues[22] = d22
					ps110.OverlayValues[40] = d40
					ps110.OverlayValues[41] = d41
					ps110.OverlayValues[42] = d42
					ps110.OverlayValues[43] = d43
					ps110.OverlayValues[65] = d65
					ps110.OverlayValues[66] = d66
					ps110.OverlayValues[67] = d67
					ps110.OverlayValues[68] = d68
					ps110.OverlayValues[69] = d69
					ps110.OverlayValues[70] = d70
					ps110.OverlayValues[71] = d71
					ps110.OverlayValues[72] = d72
					ps110.OverlayValues[73] = d73
					ps110.OverlayValues[74] = d74
					ps110.OverlayValues[106] = d106
					snap111 := d0
					snap112 := d1
					snap113 := d2
					snap114 := d3
					snap115 := d13
					snap116 := d14
					snap117 := d16
					snap118 := d17
					snap119 := d18
					snap120 := d20
					snap121 := d21
					snap122 := d22
					snap123 := d40
					snap124 := d41
					snap125 := d42
					snap126 := d43
					snap127 := d65
					snap128 := d66
					snap129 := d67
					snap130 := d68
					snap131 := d69
					snap132 := d70
					snap133 := d71
					snap134 := d72
					snap135 := d73
					snap136 := d74
					snap137 := d106
					alloc138 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps110)
					}
					ctx.RestoreAllocState(alloc138)
					d0 = snap111
					d1 = snap112
					d2 = snap113
					d3 = snap114
					d13 = snap115
					d14 = snap116
					d16 = snap117
					d17 = snap118
					d18 = snap119
					d20 = snap120
					d21 = snap121
					d22 = snap122
					d40 = snap123
					d41 = snap124
					d42 = snap125
					d43 = snap126
					d65 = snap127
					d66 = snap128
					d67 = snap129
					d68 = snap130
					d69 = snap131
					d70 = snap132
					d71 = snap133
					d72 = snap134
					d73 = snap135
					d74 = snap136
					d106 = snap137
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps109)
					}
					return result
					ctx.FreeDesc(&d21)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl32 := ctx.ReserveLabel()
					_ = lbl32
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl32)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d139 JITValueDesc
					if d70.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() / 1000000000)}
					} else {
						r0 := ctx.AllocRegExcept(d70.Reg)
						ctx.EmitMovRegReg(r0, d70.Reg)
						ctx.EmitIdivRegImm(r0, 1000000000)
						d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d139)
					}
					if d139.Loc == LocReg && d70.Loc == LocReg && d139.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d140 JITValueDesc
					if d70.Loc == LocImm {
						d140 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() % 1000000000)}
					} else {
						ctx.EmitIremRegImm(d70.Reg, 1000000000)
						d140 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d70.Reg}
						ctx.BindReg(d70.Reg, &d140)
					}
					if d140.Loc == LocReg && d70.Loc == LocReg && d140.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d139)
					ctx.EnsureDesc(&d139)
					var d141 JITValueDesc
					if d139.Loc == LocImm {
						d141 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d139.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d139.Reg)
						d141 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d139.Reg}
						ctx.BindReg(d139.Reg, &d141)
					}
					ctx.FreeDesc(&d139)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d140)
					ctx.EnsureDesc(&d140)
					var d142 JITValueDesc
					if d140.Loc == LocImm {
						d142 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d140.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d140.Reg)
						d142 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d140.Reg}
						ctx.BindReg(d140.Reg, &d142)
					}
					ctx.FreeDesc(&d140)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d142)
					var d143 JITValueDesc
					if d142.Loc == LocImm {
						d143 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d142.Imm.Float() / 1e+09)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4741671816366391296))
						ctx.EmitDivFloat64(d142.Reg, RegR11)
						d143 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d142.Reg}
						ctx.BindReg(d142.Reg, &d143)
					}
					if d143.Loc == LocReg && d142.Loc == LocReg && d143.Reg == d142.Reg {
						ctx.TransferReg(d142.Reg)
						d142.Loc = LocNone
					}
					ctx.FreeDesc(&d142)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d141)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDescsTogether(&d141, &d143)
					var d144 JITValueDesc
					if d141.Loc == LocImm && d143.Loc == LocImm {
						d144 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d141.Imm.Float() + d143.Imm.Float())}
					} else if d141.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d143.Reg)
						_, xBits := d141.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d143.Reg)
						d144 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d144)
					} else if d143.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d141.Reg)
						ctx.EmitMovRegReg(scratch, d141.Reg)
						_, yBits := d143.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d144 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d144)
					} else {
						r1 := ctx.AllocRegExcept(d141.Reg, d143.Reg)
						ctx.EmitMovRegReg(r1, d141.Reg)
						ctx.EmitAddFloat64(r1, d143.Reg)
						d144 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d144)
					}
					if d144.Loc == LocReg && d141.Loc == LocReg && d144.Reg == d141.Reg {
						ctx.TransferReg(d141.Reg)
						d141.Loc = LocNone
					}
					ctx.FreeDesc(&d141)
					ctx.FreeDesc(&d143)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d144)
					var d145 JITValueDesc
					if d144.Loc == LocImm {
						d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d144.Imm.Float()))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r2, d144.Reg)
						d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d145)
					}
					ctx.FreeDesc(&d144)
					ctx.EnsureDesc(&d145)
					if d145.Loc == LocImm {
						ctx.EmitMakeInt(result, d145)
					} else {
						ctx.EmitMovToReg(result.Reg2, d145)
						d146 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d146)
						if d145.Loc == LocReg && d145.Reg != result.Reg2 {
							ctx.FreeReg(d145.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl33 := ctx.ReserveLabel()
					_ = lbl33
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl33)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d147 JITValueDesc
					if d70.Loc == LocImm {
						d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() / 60000000000)}
					} else {
						r3 := ctx.AllocRegExcept(d70.Reg)
						ctx.EmitMovRegReg(r3, d70.Reg)
						ctx.EmitIdivRegImm(r3, 60000000000)
						d147 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d147)
					}
					if d147.Loc == LocReg && d70.Loc == LocReg && d147.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d148 JITValueDesc
					if d70.Loc == LocImm {
						d148 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() % 60000000000)}
					} else {
						ctx.EmitIremRegImm(d70.Reg, 60000000000)
						d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d70.Reg}
						ctx.BindReg(d70.Reg, &d148)
					}
					if d148.Loc == LocReg && d70.Loc == LocReg && d148.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d147)
					ctx.EnsureDesc(&d147)
					var d149 JITValueDesc
					if d147.Loc == LocImm {
						d149 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d147.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d147.Reg)
						d149 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d147.Reg}
						ctx.BindReg(d147.Reg, &d149)
					}
					ctx.FreeDesc(&d147)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d148)
					ctx.EnsureDesc(&d148)
					var d150 JITValueDesc
					if d148.Loc == LocImm {
						d150 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d148.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d148.Reg)
						d150 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d148.Reg}
						ctx.BindReg(d148.Reg, &d150)
					}
					ctx.FreeDesc(&d148)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d150)
					var d151 JITValueDesc
					if d150.Loc == LocImm {
						d151 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d150.Imm.Float() / 6e+10)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4768169126130614272))
						ctx.EmitDivFloat64(d150.Reg, RegR11)
						d151 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d150.Reg}
						ctx.BindReg(d150.Reg, &d151)
					}
					if d151.Loc == LocReg && d150.Loc == LocReg && d151.Reg == d150.Reg {
						ctx.TransferReg(d150.Reg)
						d150.Loc = LocNone
					}
					ctx.FreeDesc(&d150)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d149)
					ctx.EnsureDesc(&d151)
					ctx.EnsureDescsTogether(&d149, &d151)
					var d152 JITValueDesc
					if d149.Loc == LocImm && d151.Loc == LocImm {
						d152 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d149.Imm.Float() + d151.Imm.Float())}
					} else if d149.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d151.Reg)
						_, xBits := d149.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d151.Reg)
						d152 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d152)
					} else if d151.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d149.Reg)
						ctx.EmitMovRegReg(scratch, d149.Reg)
						_, yBits := d151.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d152 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d152)
					} else {
						r4 := ctx.AllocRegExcept(d149.Reg, d151.Reg)
						ctx.EmitMovRegReg(r4, d149.Reg)
						ctx.EmitAddFloat64(r4, d151.Reg)
						d152 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d152)
					}
					if d152.Loc == LocReg && d149.Loc == LocReg && d152.Reg == d149.Reg {
						ctx.TransferReg(d149.Reg)
						d149.Loc = LocNone
					}
					ctx.FreeDesc(&d149)
					ctx.FreeDesc(&d151)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d152)
					ctx.EnsureDesc(&d152)
					ctx.EnsureDesc(&d152)
					var d153 JITValueDesc
					if d152.Loc == LocImm {
						d153 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d152.Imm.Float()))}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r5, d152.Reg)
						d153 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d153)
					}
					ctx.FreeDesc(&d152)
					ctx.EnsureDesc(&d153)
					if d153.Loc == LocImm {
						ctx.EmitMakeInt(result, d153)
					} else {
						ctx.EmitMovToReg(result.Reg2, d153)
						d154 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d154)
						if d153.Loc == LocReg && d153.Reg != result.Reg2 {
							ctx.FreeReg(d153.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					d155 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					var d156 JITValueDesc
					if d155.Loc == LocImm {
						ctx.TrackImm(d155.Imm)
						ptrWord, _ := d155.Imm.RawWords()
						d156 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d156.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d156.Reg2, uint64(len(d155.Imm.String())))
						ctx.BindReg(d156.Reg, &d156)
						ctx.BindReg(d156.Reg2, &d156)
					} else {
						d156 = d155
					}
					d157 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d156}, 1)
					ctx.EmitAndRegImm32(d157.Reg, 1)
					d157.Type = tagBool
					ctx.BindReg(d157.Reg, &d157)
					d158 = d157
					ctx.EnsureDesc(&d158)
					if d158.Loc != LocImm && d158.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d158.Loc == LocImm {
						if d158.Imm.Bool() {
							if ps.General {
							}
							ps159 := PhiState{General: ps.General}
							ps159.OverlayValues = make([]JITValueDesc, 159)
							ps159.OverlayValues[0] = d0
							ps159.OverlayValues[1] = d1
							ps159.OverlayValues[2] = d2
							ps159.OverlayValues[3] = d3
							ps159.OverlayValues[13] = d13
							ps159.OverlayValues[14] = d14
							ps159.OverlayValues[16] = d16
							ps159.OverlayValues[17] = d17
							ps159.OverlayValues[18] = d18
							ps159.OverlayValues[20] = d20
							ps159.OverlayValues[21] = d21
							ps159.OverlayValues[22] = d22
							ps159.OverlayValues[40] = d40
							ps159.OverlayValues[41] = d41
							ps159.OverlayValues[42] = d42
							ps159.OverlayValues[43] = d43
							ps159.OverlayValues[65] = d65
							ps159.OverlayValues[66] = d66
							ps159.OverlayValues[67] = d67
							ps159.OverlayValues[68] = d68
							ps159.OverlayValues[69] = d69
							ps159.OverlayValues[70] = d70
							ps159.OverlayValues[71] = d71
							ps159.OverlayValues[72] = d72
							ps159.OverlayValues[73] = d73
							ps159.OverlayValues[74] = d74
							ps159.OverlayValues[106] = d106
							ps159.OverlayValues[139] = d139
							ps159.OverlayValues[140] = d140
							ps159.OverlayValues[141] = d141
							ps159.OverlayValues[142] = d142
							ps159.OverlayValues[143] = d143
							ps159.OverlayValues[144] = d144
							ps159.OverlayValues[145] = d145
							ps159.OverlayValues[146] = d146
							ps159.OverlayValues[147] = d147
							ps159.OverlayValues[148] = d148
							ps159.OverlayValues[149] = d149
							ps159.OverlayValues[150] = d150
							ps159.OverlayValues[151] = d151
							ps159.OverlayValues[152] = d152
							ps159.OverlayValues[153] = d153
							ps159.OverlayValues[154] = d154
							ps159.OverlayValues[155] = d155
							ps159.OverlayValues[156] = d156
							ps159.OverlayValues[157] = d157
							ps159.OverlayValues[158] = d158
							return bbs[8].RenderPS(ps159)
						}
						if ps.General {
						}
						ps160 := PhiState{General: ps.General}
						ps160.OverlayValues = make([]JITValueDesc, 159)
						ps160.OverlayValues[0] = d0
						ps160.OverlayValues[1] = d1
						ps160.OverlayValues[2] = d2
						ps160.OverlayValues[3] = d3
						ps160.OverlayValues[13] = d13
						ps160.OverlayValues[14] = d14
						ps160.OverlayValues[16] = d16
						ps160.OverlayValues[17] = d17
						ps160.OverlayValues[18] = d18
						ps160.OverlayValues[20] = d20
						ps160.OverlayValues[21] = d21
						ps160.OverlayValues[22] = d22
						ps160.OverlayValues[40] = d40
						ps160.OverlayValues[41] = d41
						ps160.OverlayValues[42] = d42
						ps160.OverlayValues[43] = d43
						ps160.OverlayValues[65] = d65
						ps160.OverlayValues[66] = d66
						ps160.OverlayValues[67] = d67
						ps160.OverlayValues[68] = d68
						ps160.OverlayValues[69] = d69
						ps160.OverlayValues[70] = d70
						ps160.OverlayValues[71] = d71
						ps160.OverlayValues[72] = d72
						ps160.OverlayValues[73] = d73
						ps160.OverlayValues[74] = d74
						ps160.OverlayValues[106] = d106
						ps160.OverlayValues[139] = d139
						ps160.OverlayValues[140] = d140
						ps160.OverlayValues[141] = d141
						ps160.OverlayValues[142] = d142
						ps160.OverlayValues[143] = d143
						ps160.OverlayValues[144] = d144
						ps160.OverlayValues[145] = d145
						ps160.OverlayValues[146] = d146
						ps160.OverlayValues[147] = d147
						ps160.OverlayValues[148] = d148
						ps160.OverlayValues[149] = d149
						ps160.OverlayValues[150] = d150
						ps160.OverlayValues[151] = d151
						ps160.OverlayValues[152] = d152
						ps160.OverlayValues[153] = d153
						ps160.OverlayValues[154] = d154
						ps160.OverlayValues[155] = d155
						ps160.OverlayValues[156] = d156
						ps160.OverlayValues[157] = d157
						ps160.OverlayValues[158] = d158
						return bbs[11].RenderPS(ps160)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl34 := ctx.ReserveLabel()
					lbl35 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d158.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl34)
					ctx.EmitJmp(lbl35)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl12)
					ps161 := PhiState{General: true}
					ps161.OverlayValues = make([]JITValueDesc, 159)
					ps161.OverlayValues[0] = d0
					ps161.OverlayValues[1] = d1
					ps161.OverlayValues[2] = d2
					ps161.OverlayValues[3] = d3
					ps161.OverlayValues[13] = d13
					ps161.OverlayValues[14] = d14
					ps161.OverlayValues[16] = d16
					ps161.OverlayValues[17] = d17
					ps161.OverlayValues[18] = d18
					ps161.OverlayValues[20] = d20
					ps161.OverlayValues[21] = d21
					ps161.OverlayValues[22] = d22
					ps161.OverlayValues[40] = d40
					ps161.OverlayValues[41] = d41
					ps161.OverlayValues[42] = d42
					ps161.OverlayValues[43] = d43
					ps161.OverlayValues[65] = d65
					ps161.OverlayValues[66] = d66
					ps161.OverlayValues[67] = d67
					ps161.OverlayValues[68] = d68
					ps161.OverlayValues[69] = d69
					ps161.OverlayValues[70] = d70
					ps161.OverlayValues[71] = d71
					ps161.OverlayValues[72] = d72
					ps161.OverlayValues[73] = d73
					ps161.OverlayValues[74] = d74
					ps161.OverlayValues[106] = d106
					ps161.OverlayValues[139] = d139
					ps161.OverlayValues[140] = d140
					ps161.OverlayValues[141] = d141
					ps161.OverlayValues[142] = d142
					ps161.OverlayValues[143] = d143
					ps161.OverlayValues[144] = d144
					ps161.OverlayValues[145] = d145
					ps161.OverlayValues[146] = d146
					ps161.OverlayValues[147] = d147
					ps161.OverlayValues[148] = d148
					ps161.OverlayValues[149] = d149
					ps161.OverlayValues[150] = d150
					ps161.OverlayValues[151] = d151
					ps161.OverlayValues[152] = d152
					ps161.OverlayValues[153] = d153
					ps161.OverlayValues[154] = d154
					ps161.OverlayValues[155] = d155
					ps161.OverlayValues[156] = d156
					ps161.OverlayValues[157] = d157
					ps161.OverlayValues[158] = d158
					ps162 := PhiState{General: true}
					ps162.OverlayValues = make([]JITValueDesc, 159)
					ps162.OverlayValues[0] = d0
					ps162.OverlayValues[1] = d1
					ps162.OverlayValues[2] = d2
					ps162.OverlayValues[3] = d3
					ps162.OverlayValues[13] = d13
					ps162.OverlayValues[14] = d14
					ps162.OverlayValues[16] = d16
					ps162.OverlayValues[17] = d17
					ps162.OverlayValues[18] = d18
					ps162.OverlayValues[20] = d20
					ps162.OverlayValues[21] = d21
					ps162.OverlayValues[22] = d22
					ps162.OverlayValues[40] = d40
					ps162.OverlayValues[41] = d41
					ps162.OverlayValues[42] = d42
					ps162.OverlayValues[43] = d43
					ps162.OverlayValues[65] = d65
					ps162.OverlayValues[66] = d66
					ps162.OverlayValues[67] = d67
					ps162.OverlayValues[68] = d68
					ps162.OverlayValues[69] = d69
					ps162.OverlayValues[70] = d70
					ps162.OverlayValues[71] = d71
					ps162.OverlayValues[72] = d72
					ps162.OverlayValues[73] = d73
					ps162.OverlayValues[74] = d74
					ps162.OverlayValues[106] = d106
					ps162.OverlayValues[139] = d139
					ps162.OverlayValues[140] = d140
					ps162.OverlayValues[141] = d141
					ps162.OverlayValues[142] = d142
					ps162.OverlayValues[143] = d143
					ps162.OverlayValues[144] = d144
					ps162.OverlayValues[145] = d145
					ps162.OverlayValues[146] = d146
					ps162.OverlayValues[147] = d147
					ps162.OverlayValues[148] = d148
					ps162.OverlayValues[149] = d149
					ps162.OverlayValues[150] = d150
					ps162.OverlayValues[151] = d151
					ps162.OverlayValues[152] = d152
					ps162.OverlayValues[153] = d153
					ps162.OverlayValues[154] = d154
					ps162.OverlayValues[155] = d155
					ps162.OverlayValues[156] = d156
					ps162.OverlayValues[157] = d157
					ps162.OverlayValues[158] = d158
					snap163 := d0
					snap164 := d1
					snap165 := d2
					snap166 := d3
					snap167 := d13
					snap168 := d14
					snap169 := d16
					snap170 := d17
					snap171 := d18
					snap172 := d20
					snap173 := d21
					snap174 := d22
					snap175 := d40
					snap176 := d41
					snap177 := d42
					snap178 := d43
					snap179 := d65
					snap180 := d66
					snap181 := d67
					snap182 := d68
					snap183 := d69
					snap184 := d70
					snap185 := d71
					snap186 := d72
					snap187 := d73
					snap188 := d74
					snap189 := d106
					snap190 := d139
					snap191 := d140
					snap192 := d141
					snap193 := d142
					snap194 := d143
					snap195 := d144
					snap196 := d145
					snap197 := d146
					snap198 := d147
					snap199 := d148
					snap200 := d149
					snap201 := d150
					snap202 := d151
					snap203 := d152
					snap204 := d153
					snap205 := d154
					snap206 := d155
					snap207 := d156
					snap208 := d157
					snap209 := d158
					alloc210 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps162)
					}
					ctx.RestoreAllocState(alloc210)
					d0 = snap163
					d1 = snap164
					d2 = snap165
					d3 = snap166
					d13 = snap167
					d14 = snap168
					d16 = snap169
					d17 = snap170
					d18 = snap171
					d20 = snap172
					d21 = snap173
					d22 = snap174
					d40 = snap175
					d41 = snap176
					d42 = snap177
					d43 = snap178
					d65 = snap179
					d66 = snap180
					d67 = snap181
					d68 = snap182
					d69 = snap183
					d70 = snap184
					d71 = snap185
					d72 = snap186
					d73 = snap187
					d74 = snap188
					d106 = snap189
					d139 = snap190
					d140 = snap191
					d141 = snap192
					d142 = snap193
					d143 = snap194
					d144 = snap195
					d145 = snap196
					d146 = snap197
					d147 = snap198
					d148 = snap199
					d149 = snap200
					d150 = snap201
					d151 = snap202
					d152 = snap203
					d153 = snap204
					d154 = snap205
					d155 = snap206
					d156 = snap207
					d157 = snap208
					d158 = snap209
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps161)
					}
					return result
					ctx.FreeDesc(&d157)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl36 := ctx.ReserveLabel()
					_ = lbl36
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl36)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d211 JITValueDesc
					if d70.Loc == LocImm {
						d211 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() / 3600000000000)}
					} else {
						r6 := ctx.AllocRegExcept(d70.Reg)
						ctx.EmitMovRegReg(r6, d70.Reg)
						ctx.EmitIdivRegImm(r6, 3600000000000)
						d211 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d211)
					}
					if d211.Loc == LocReg && d70.Loc == LocReg && d211.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d212 JITValueDesc
					if d70.Loc == LocImm {
						d212 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() % 3600000000000)}
					} else {
						ctx.EmitIremRegImm(d70.Reg, 3600000000000)
						d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d70.Reg}
						ctx.BindReg(d70.Reg, &d212)
					}
					if d212.Loc == LocReg && d70.Loc == LocReg && d212.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					var d213 JITValueDesc
					if d211.Loc == LocImm {
						d213 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d211.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d211.Reg)
						d213 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d211.Reg}
						ctx.BindReg(d211.Reg, &d213)
					}
					ctx.FreeDesc(&d211)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d212)
					ctx.EnsureDesc(&d212)
					var d214 JITValueDesc
					if d212.Loc == LocImm {
						d214 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d212.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d212.Reg)
						d214 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d212.Reg}
						ctx.BindReg(d212.Reg, &d214)
					}
					ctx.FreeDesc(&d212)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d214)
					var d215 JITValueDesc
					if d214.Loc == LocImm {
						d215 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d214.Imm.Float() / 3.6e+12)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4794699203894837248))
						ctx.EmitDivFloat64(d214.Reg, RegR11)
						d215 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d214.Reg}
						ctx.BindReg(d214.Reg, &d215)
					}
					if d215.Loc == LocReg && d214.Loc == LocReg && d215.Reg == d214.Reg {
						ctx.TransferReg(d214.Reg)
						d214.Loc = LocNone
					}
					ctx.FreeDesc(&d214)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d213)
					ctx.EnsureDesc(&d215)
					ctx.EnsureDescsTogether(&d213, &d215)
					var d216 JITValueDesc
					if d213.Loc == LocImm && d215.Loc == LocImm {
						d216 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d213.Imm.Float() + d215.Imm.Float())}
					} else if d213.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d215.Reg)
						_, xBits := d213.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d215.Reg)
						d216 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d216)
					} else if d215.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d213.Reg)
						ctx.EmitMovRegReg(scratch, d213.Reg)
						_, yBits := d215.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d216 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d216)
					} else {
						r7 := ctx.AllocRegExcept(d213.Reg, d215.Reg)
						ctx.EmitMovRegReg(r7, d213.Reg)
						ctx.EmitAddFloat64(r7, d215.Reg)
						d216 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d216)
					}
					if d216.Loc == LocReg && d213.Loc == LocReg && d216.Reg == d213.Reg {
						ctx.TransferReg(d213.Reg)
						d213.Loc = LocNone
					}
					ctx.FreeDesc(&d213)
					ctx.FreeDesc(&d215)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d216)
					ctx.EnsureDesc(&d216)
					ctx.EnsureDesc(&d216)
					var d217 JITValueDesc
					if d216.Loc == LocImm {
						d217 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d216.Imm.Float()))}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r8, d216.Reg)
						d217 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d217)
					}
					ctx.FreeDesc(&d216)
					ctx.EnsureDesc(&d217)
					if d217.Loc == LocImm {
						ctx.EmitMakeInt(result, d217)
					} else {
						ctx.EmitMovToReg(result.Reg2, d217)
						d218 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d218)
						if d217.Loc == LocReg && d217.Reg != result.Reg2 {
							ctx.FreeReg(d217.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[11].VisitCount >= 0 {
							ps.General = true
							return bbs[11].RenderPS(ps)
						}
					}
					bbs[11].VisitCount++
					if ps.General {
						if bbs[11].Rendered {
							ctx.EmitJmp(lbl12)
							return result
						}
						bbs[11].Rendered = true
						bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_11 = bbs[11].Address
						ctx.MarkLabel(lbl12)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					d219 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					var d220 JITValueDesc
					if d219.Loc == LocImm {
						ctx.TrackImm(d219.Imm)
						ptrWord, _ := d219.Imm.RawWords()
						d220 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d220.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d220.Reg2, uint64(len(d219.Imm.String())))
						ctx.BindReg(d220.Reg, &d220)
						ctx.BindReg(d220.Reg2, &d220)
					} else {
						d220 = d219
					}
					d221 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d220}, 1)
					ctx.EmitAndRegImm32(d221.Reg, 1)
					d221.Type = tagBool
					ctx.BindReg(d221.Reg, &d221)
					d222 = d221
					ctx.EnsureDesc(&d222)
					if d222.Loc != LocImm && d222.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d222.Loc == LocImm {
						if d222.Imm.Bool() {
							if ps.General {
							}
							ps223 := PhiState{General: ps.General}
							ps223.OverlayValues = make([]JITValueDesc, 223)
							ps223.OverlayValues[0] = d0
							ps223.OverlayValues[1] = d1
							ps223.OverlayValues[2] = d2
							ps223.OverlayValues[3] = d3
							ps223.OverlayValues[13] = d13
							ps223.OverlayValues[14] = d14
							ps223.OverlayValues[16] = d16
							ps223.OverlayValues[17] = d17
							ps223.OverlayValues[18] = d18
							ps223.OverlayValues[20] = d20
							ps223.OverlayValues[21] = d21
							ps223.OverlayValues[22] = d22
							ps223.OverlayValues[40] = d40
							ps223.OverlayValues[41] = d41
							ps223.OverlayValues[42] = d42
							ps223.OverlayValues[43] = d43
							ps223.OverlayValues[65] = d65
							ps223.OverlayValues[66] = d66
							ps223.OverlayValues[67] = d67
							ps223.OverlayValues[68] = d68
							ps223.OverlayValues[69] = d69
							ps223.OverlayValues[70] = d70
							ps223.OverlayValues[71] = d71
							ps223.OverlayValues[72] = d72
							ps223.OverlayValues[73] = d73
							ps223.OverlayValues[74] = d74
							ps223.OverlayValues[106] = d106
							ps223.OverlayValues[139] = d139
							ps223.OverlayValues[140] = d140
							ps223.OverlayValues[141] = d141
							ps223.OverlayValues[142] = d142
							ps223.OverlayValues[143] = d143
							ps223.OverlayValues[144] = d144
							ps223.OverlayValues[145] = d145
							ps223.OverlayValues[146] = d146
							ps223.OverlayValues[147] = d147
							ps223.OverlayValues[148] = d148
							ps223.OverlayValues[149] = d149
							ps223.OverlayValues[150] = d150
							ps223.OverlayValues[151] = d151
							ps223.OverlayValues[152] = d152
							ps223.OverlayValues[153] = d153
							ps223.OverlayValues[154] = d154
							ps223.OverlayValues[155] = d155
							ps223.OverlayValues[156] = d156
							ps223.OverlayValues[157] = d157
							ps223.OverlayValues[158] = d158
							ps223.OverlayValues[211] = d211
							ps223.OverlayValues[212] = d212
							ps223.OverlayValues[213] = d213
							ps223.OverlayValues[214] = d214
							ps223.OverlayValues[215] = d215
							ps223.OverlayValues[216] = d216
							ps223.OverlayValues[217] = d217
							ps223.OverlayValues[218] = d218
							ps223.OverlayValues[219] = d219
							ps223.OverlayValues[220] = d220
							ps223.OverlayValues[221] = d221
							ps223.OverlayValues[222] = d222
							return bbs[10].RenderPS(ps223)
						}
						if ps.General {
						}
						ps224 := PhiState{General: ps.General}
						ps224.OverlayValues = make([]JITValueDesc, 223)
						ps224.OverlayValues[0] = d0
						ps224.OverlayValues[1] = d1
						ps224.OverlayValues[2] = d2
						ps224.OverlayValues[3] = d3
						ps224.OverlayValues[13] = d13
						ps224.OverlayValues[14] = d14
						ps224.OverlayValues[16] = d16
						ps224.OverlayValues[17] = d17
						ps224.OverlayValues[18] = d18
						ps224.OverlayValues[20] = d20
						ps224.OverlayValues[21] = d21
						ps224.OverlayValues[22] = d22
						ps224.OverlayValues[40] = d40
						ps224.OverlayValues[41] = d41
						ps224.OverlayValues[42] = d42
						ps224.OverlayValues[43] = d43
						ps224.OverlayValues[65] = d65
						ps224.OverlayValues[66] = d66
						ps224.OverlayValues[67] = d67
						ps224.OverlayValues[68] = d68
						ps224.OverlayValues[69] = d69
						ps224.OverlayValues[70] = d70
						ps224.OverlayValues[71] = d71
						ps224.OverlayValues[72] = d72
						ps224.OverlayValues[73] = d73
						ps224.OverlayValues[74] = d74
						ps224.OverlayValues[106] = d106
						ps224.OverlayValues[139] = d139
						ps224.OverlayValues[140] = d140
						ps224.OverlayValues[141] = d141
						ps224.OverlayValues[142] = d142
						ps224.OverlayValues[143] = d143
						ps224.OverlayValues[144] = d144
						ps224.OverlayValues[145] = d145
						ps224.OverlayValues[146] = d146
						ps224.OverlayValues[147] = d147
						ps224.OverlayValues[148] = d148
						ps224.OverlayValues[149] = d149
						ps224.OverlayValues[150] = d150
						ps224.OverlayValues[151] = d151
						ps224.OverlayValues[152] = d152
						ps224.OverlayValues[153] = d153
						ps224.OverlayValues[154] = d154
						ps224.OverlayValues[155] = d155
						ps224.OverlayValues[156] = d156
						ps224.OverlayValues[157] = d157
						ps224.OverlayValues[158] = d158
						ps224.OverlayValues[211] = d211
						ps224.OverlayValues[212] = d212
						ps224.OverlayValues[213] = d213
						ps224.OverlayValues[214] = d214
						ps224.OverlayValues[215] = d215
						ps224.OverlayValues[216] = d216
						ps224.OverlayValues[217] = d217
						ps224.OverlayValues[218] = d218
						ps224.OverlayValues[219] = d219
						ps224.OverlayValues[220] = d220
						ps224.OverlayValues[221] = d221
						ps224.OverlayValues[222] = d222
						return bbs[13].RenderPS(ps224)
					}
					if !ps.General {
						ps.General = true
						return bbs[11].RenderPS(ps)
					}
					lbl37 := ctx.ReserveLabel()
					lbl38 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d222.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl37)
					ctx.EmitJmp(lbl38)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl38)
					ctx.EmitJmp(lbl14)
					ps225 := PhiState{General: true}
					ps225.OverlayValues = make([]JITValueDesc, 223)
					ps225.OverlayValues[0] = d0
					ps225.OverlayValues[1] = d1
					ps225.OverlayValues[2] = d2
					ps225.OverlayValues[3] = d3
					ps225.OverlayValues[13] = d13
					ps225.OverlayValues[14] = d14
					ps225.OverlayValues[16] = d16
					ps225.OverlayValues[17] = d17
					ps225.OverlayValues[18] = d18
					ps225.OverlayValues[20] = d20
					ps225.OverlayValues[21] = d21
					ps225.OverlayValues[22] = d22
					ps225.OverlayValues[40] = d40
					ps225.OverlayValues[41] = d41
					ps225.OverlayValues[42] = d42
					ps225.OverlayValues[43] = d43
					ps225.OverlayValues[65] = d65
					ps225.OverlayValues[66] = d66
					ps225.OverlayValues[67] = d67
					ps225.OverlayValues[68] = d68
					ps225.OverlayValues[69] = d69
					ps225.OverlayValues[70] = d70
					ps225.OverlayValues[71] = d71
					ps225.OverlayValues[72] = d72
					ps225.OverlayValues[73] = d73
					ps225.OverlayValues[74] = d74
					ps225.OverlayValues[106] = d106
					ps225.OverlayValues[139] = d139
					ps225.OverlayValues[140] = d140
					ps225.OverlayValues[141] = d141
					ps225.OverlayValues[142] = d142
					ps225.OverlayValues[143] = d143
					ps225.OverlayValues[144] = d144
					ps225.OverlayValues[145] = d145
					ps225.OverlayValues[146] = d146
					ps225.OverlayValues[147] = d147
					ps225.OverlayValues[148] = d148
					ps225.OverlayValues[149] = d149
					ps225.OverlayValues[150] = d150
					ps225.OverlayValues[151] = d151
					ps225.OverlayValues[152] = d152
					ps225.OverlayValues[153] = d153
					ps225.OverlayValues[154] = d154
					ps225.OverlayValues[155] = d155
					ps225.OverlayValues[156] = d156
					ps225.OverlayValues[157] = d157
					ps225.OverlayValues[158] = d158
					ps225.OverlayValues[211] = d211
					ps225.OverlayValues[212] = d212
					ps225.OverlayValues[213] = d213
					ps225.OverlayValues[214] = d214
					ps225.OverlayValues[215] = d215
					ps225.OverlayValues[216] = d216
					ps225.OverlayValues[217] = d217
					ps225.OverlayValues[218] = d218
					ps225.OverlayValues[219] = d219
					ps225.OverlayValues[220] = d220
					ps225.OverlayValues[221] = d221
					ps225.OverlayValues[222] = d222
					ps226 := PhiState{General: true}
					ps226.OverlayValues = make([]JITValueDesc, 223)
					ps226.OverlayValues[0] = d0
					ps226.OverlayValues[1] = d1
					ps226.OverlayValues[2] = d2
					ps226.OverlayValues[3] = d3
					ps226.OverlayValues[13] = d13
					ps226.OverlayValues[14] = d14
					ps226.OverlayValues[16] = d16
					ps226.OverlayValues[17] = d17
					ps226.OverlayValues[18] = d18
					ps226.OverlayValues[20] = d20
					ps226.OverlayValues[21] = d21
					ps226.OverlayValues[22] = d22
					ps226.OverlayValues[40] = d40
					ps226.OverlayValues[41] = d41
					ps226.OverlayValues[42] = d42
					ps226.OverlayValues[43] = d43
					ps226.OverlayValues[65] = d65
					ps226.OverlayValues[66] = d66
					ps226.OverlayValues[67] = d67
					ps226.OverlayValues[68] = d68
					ps226.OverlayValues[69] = d69
					ps226.OverlayValues[70] = d70
					ps226.OverlayValues[71] = d71
					ps226.OverlayValues[72] = d72
					ps226.OverlayValues[73] = d73
					ps226.OverlayValues[74] = d74
					ps226.OverlayValues[106] = d106
					ps226.OverlayValues[139] = d139
					ps226.OverlayValues[140] = d140
					ps226.OverlayValues[141] = d141
					ps226.OverlayValues[142] = d142
					ps226.OverlayValues[143] = d143
					ps226.OverlayValues[144] = d144
					ps226.OverlayValues[145] = d145
					ps226.OverlayValues[146] = d146
					ps226.OverlayValues[147] = d147
					ps226.OverlayValues[148] = d148
					ps226.OverlayValues[149] = d149
					ps226.OverlayValues[150] = d150
					ps226.OverlayValues[151] = d151
					ps226.OverlayValues[152] = d152
					ps226.OverlayValues[153] = d153
					ps226.OverlayValues[154] = d154
					ps226.OverlayValues[155] = d155
					ps226.OverlayValues[156] = d156
					ps226.OverlayValues[157] = d157
					ps226.OverlayValues[158] = d158
					ps226.OverlayValues[211] = d211
					ps226.OverlayValues[212] = d212
					ps226.OverlayValues[213] = d213
					ps226.OverlayValues[214] = d214
					ps226.OverlayValues[215] = d215
					ps226.OverlayValues[216] = d216
					ps226.OverlayValues[217] = d217
					ps226.OverlayValues[218] = d218
					ps226.OverlayValues[219] = d219
					ps226.OverlayValues[220] = d220
					ps226.OverlayValues[221] = d221
					ps226.OverlayValues[222] = d222
					snap227 := d0
					snap228 := d1
					snap229 := d2
					snap230 := d3
					snap231 := d13
					snap232 := d14
					snap233 := d16
					snap234 := d17
					snap235 := d18
					snap236 := d20
					snap237 := d21
					snap238 := d22
					snap239 := d40
					snap240 := d41
					snap241 := d42
					snap242 := d43
					snap243 := d65
					snap244 := d66
					snap245 := d67
					snap246 := d68
					snap247 := d69
					snap248 := d70
					snap249 := d71
					snap250 := d72
					snap251 := d73
					snap252 := d74
					snap253 := d106
					snap254 := d139
					snap255 := d140
					snap256 := d141
					snap257 := d142
					snap258 := d143
					snap259 := d144
					snap260 := d145
					snap261 := d146
					snap262 := d147
					snap263 := d148
					snap264 := d149
					snap265 := d150
					snap266 := d151
					snap267 := d152
					snap268 := d153
					snap269 := d154
					snap270 := d155
					snap271 := d156
					snap272 := d157
					snap273 := d158
					snap274 := d211
					snap275 := d212
					snap276 := d213
					snap277 := d214
					snap278 := d215
					snap279 := d216
					snap280 := d217
					snap281 := d218
					snap282 := d219
					snap283 := d220
					snap284 := d221
					snap285 := d222
					alloc286 := ctx.SnapshotAllocState()
					if !bbs[13].Rendered {
						bbs[13].RenderPS(ps226)
					}
					ctx.RestoreAllocState(alloc286)
					d0 = snap227
					d1 = snap228
					d2 = snap229
					d3 = snap230
					d13 = snap231
					d14 = snap232
					d16 = snap233
					d17 = snap234
					d18 = snap235
					d20 = snap236
					d21 = snap237
					d22 = snap238
					d40 = snap239
					d41 = snap240
					d42 = snap241
					d43 = snap242
					d65 = snap243
					d66 = snap244
					d67 = snap245
					d68 = snap246
					d69 = snap247
					d70 = snap248
					d71 = snap249
					d72 = snap250
					d73 = snap251
					d74 = snap252
					d106 = snap253
					d139 = snap254
					d140 = snap255
					d141 = snap256
					d142 = snap257
					d143 = snap258
					d144 = snap259
					d145 = snap260
					d146 = snap261
					d147 = snap262
					d148 = snap263
					d149 = snap264
					d150 = snap265
					d151 = snap266
					d152 = snap267
					d153 = snap268
					d154 = snap269
					d155 = snap270
					d156 = snap271
					d157 = snap272
					d158 = snap273
					d211 = snap274
					d212 = snap275
					d213 = snap276
					d214 = snap277
					d215 = snap278
					d216 = snap279
					d217 = snap280
					d218 = snap281
					d219 = snap282
					d220 = snap283
					d221 = snap284
					d222 = snap285
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps225)
					}
					return result
					ctx.FreeDesc(&d221)
					return result
				}
				bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[12].VisitCount >= 0 {
							ps.General = true
							return bbs[12].RenderPS(ps)
						}
					}
					bbs[12].VisitCount++
					if ps.General {
						if bbs[12].Rendered {
							ctx.EmitJmp(lbl13)
							return result
						}
						bbs[12].Rendered = true
						bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_12 = bbs[12].Address
						ctx.MarkLabel(lbl13)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl39 := ctx.ReserveLabel()
					_ = lbl39
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl39)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d287 JITValueDesc
					if d70.Loc == LocImm {
						d287 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() / 3600000000000)}
					} else {
						r9 := ctx.AllocRegExcept(d70.Reg)
						ctx.EmitMovRegReg(r9, d70.Reg)
						ctx.EmitIdivRegImm(r9, 3600000000000)
						d287 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d287)
					}
					if d287.Loc == LocReg && d70.Loc == LocReg && d287.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d288 JITValueDesc
					if d70.Loc == LocImm {
						d288 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() % 3600000000000)}
					} else {
						ctx.EmitIremRegImm(d70.Reg, 3600000000000)
						d288 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d70.Reg}
						ctx.BindReg(d70.Reg, &d288)
					}
					if d288.Loc == LocReg && d70.Loc == LocReg && d288.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d287)
					ctx.EnsureDesc(&d287)
					var d289 JITValueDesc
					if d287.Loc == LocImm {
						d289 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d287.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d287.Reg)
						d289 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d287.Reg}
						ctx.BindReg(d287.Reg, &d289)
					}
					ctx.FreeDesc(&d287)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d288)
					ctx.EnsureDesc(&d288)
					var d290 JITValueDesc
					if d288.Loc == LocImm {
						d290 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d288.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d288.Reg)
						d290 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d288.Reg}
						ctx.BindReg(d288.Reg, &d290)
					}
					ctx.FreeDesc(&d288)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d290)
					var d291 JITValueDesc
					if d290.Loc == LocImm {
						d291 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d290.Imm.Float() / 3.6e+12)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4794699203894837248))
						ctx.EmitDivFloat64(d290.Reg, RegR11)
						d291 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d290.Reg}
						ctx.BindReg(d290.Reg, &d291)
					}
					if d291.Loc == LocReg && d290.Loc == LocReg && d291.Reg == d290.Reg {
						ctx.TransferReg(d290.Reg)
						d290.Loc = LocNone
					}
					ctx.FreeDesc(&d290)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d289)
					ctx.EnsureDesc(&d291)
					ctx.EnsureDescsTogether(&d289, &d291)
					var d292 JITValueDesc
					if d289.Loc == LocImm && d291.Loc == LocImm {
						d292 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d289.Imm.Float() + d291.Imm.Float())}
					} else if d289.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d291.Reg)
						_, xBits := d289.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d291.Reg)
						d292 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d292)
					} else if d291.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d289.Reg)
						ctx.EmitMovRegReg(scratch, d289.Reg)
						_, yBits := d291.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d292 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d292)
					} else {
						r10 := ctx.AllocRegExcept(d289.Reg, d291.Reg)
						ctx.EmitMovRegReg(r10, d289.Reg)
						ctx.EmitAddFloat64(r10, d291.Reg)
						d292 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r10}
						ctx.BindReg(r10, &d292)
					}
					if d292.Loc == LocReg && d289.Loc == LocReg && d292.Reg == d289.Reg {
						ctx.TransferReg(d289.Reg)
						d289.Loc = LocNone
					}
					ctx.FreeDesc(&d289)
					ctx.FreeDesc(&d291)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d292)
					ctx.EnsureDesc(&d292)
					var d293 JITValueDesc
					if d292.Loc == LocImm {
						d293 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d292.Imm.Float() / 24)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4627448617123184640))
						ctx.EmitDivFloat64(d292.Reg, RegR11)
						d293 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d292.Reg}
						ctx.BindReg(d292.Reg, &d293)
					}
					if d293.Loc == LocReg && d292.Loc == LocReg && d293.Reg == d292.Reg {
						ctx.TransferReg(d292.Reg)
						d292.Loc = LocNone
					}
					ctx.FreeDesc(&d292)
					ctx.EnsureDesc(&d293)
					ctx.EnsureDesc(&d293)
					var d294 JITValueDesc
					if d293.Loc == LocImm {
						d294 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d293.Imm.Float()))}
					} else {
						r11 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r11, d293.Reg)
						d294 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d294)
					}
					ctx.FreeDesc(&d293)
					ctx.EnsureDesc(&d294)
					if d294.Loc == LocImm {
						ctx.EmitMakeInt(result, d294)
					} else {
						ctx.EmitMovToReg(result.Reg2, d294)
						d295 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d295)
						if d294.Loc == LocReg && d294.Reg != result.Reg2 {
							ctx.FreeReg(d294.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[13].VisitCount >= 0 {
							ps.General = true
							return bbs[13].RenderPS(ps)
						}
					}
					bbs[13].VisitCount++
					if ps.General {
						if bbs[13].Rendered {
							ctx.EmitJmp(lbl14)
							return result
						}
						bbs[13].Rendered = true
						bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_13 = bbs[13].Address
						ctx.MarkLabel(lbl14)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					d296 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
					var d297 JITValueDesc
					if d296.Loc == LocImm {
						ctx.TrackImm(d296.Imm)
						ptrWord, _ := d296.Imm.RawWords()
						d297 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d297.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d297.Reg2, uint64(len(d296.Imm.String())))
						ctx.BindReg(d297.Reg, &d297)
						ctx.BindReg(d297.Reg2, &d297)
					} else {
						d297 = d296
					}
					d298 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d297}, 1)
					ctx.EmitAndRegImm32(d298.Reg, 1)
					d298.Type = tagBool
					ctx.BindReg(d298.Reg, &d298)
					d299 = d298
					ctx.EnsureDesc(&d299)
					if d299.Loc != LocImm && d299.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d299.Loc == LocImm {
						if d299.Imm.Bool() {
							if ps.General {
							}
							ps300 := PhiState{General: ps.General}
							ps300.OverlayValues = make([]JITValueDesc, 300)
							ps300.OverlayValues[0] = d0
							ps300.OverlayValues[1] = d1
							ps300.OverlayValues[2] = d2
							ps300.OverlayValues[3] = d3
							ps300.OverlayValues[13] = d13
							ps300.OverlayValues[14] = d14
							ps300.OverlayValues[16] = d16
							ps300.OverlayValues[17] = d17
							ps300.OverlayValues[18] = d18
							ps300.OverlayValues[20] = d20
							ps300.OverlayValues[21] = d21
							ps300.OverlayValues[22] = d22
							ps300.OverlayValues[40] = d40
							ps300.OverlayValues[41] = d41
							ps300.OverlayValues[42] = d42
							ps300.OverlayValues[43] = d43
							ps300.OverlayValues[65] = d65
							ps300.OverlayValues[66] = d66
							ps300.OverlayValues[67] = d67
							ps300.OverlayValues[68] = d68
							ps300.OverlayValues[69] = d69
							ps300.OverlayValues[70] = d70
							ps300.OverlayValues[71] = d71
							ps300.OverlayValues[72] = d72
							ps300.OverlayValues[73] = d73
							ps300.OverlayValues[74] = d74
							ps300.OverlayValues[106] = d106
							ps300.OverlayValues[139] = d139
							ps300.OverlayValues[140] = d140
							ps300.OverlayValues[141] = d141
							ps300.OverlayValues[142] = d142
							ps300.OverlayValues[143] = d143
							ps300.OverlayValues[144] = d144
							ps300.OverlayValues[145] = d145
							ps300.OverlayValues[146] = d146
							ps300.OverlayValues[147] = d147
							ps300.OverlayValues[148] = d148
							ps300.OverlayValues[149] = d149
							ps300.OverlayValues[150] = d150
							ps300.OverlayValues[151] = d151
							ps300.OverlayValues[152] = d152
							ps300.OverlayValues[153] = d153
							ps300.OverlayValues[154] = d154
							ps300.OverlayValues[155] = d155
							ps300.OverlayValues[156] = d156
							ps300.OverlayValues[157] = d157
							ps300.OverlayValues[158] = d158
							ps300.OverlayValues[211] = d211
							ps300.OverlayValues[212] = d212
							ps300.OverlayValues[213] = d213
							ps300.OverlayValues[214] = d214
							ps300.OverlayValues[215] = d215
							ps300.OverlayValues[216] = d216
							ps300.OverlayValues[217] = d217
							ps300.OverlayValues[218] = d218
							ps300.OverlayValues[219] = d219
							ps300.OverlayValues[220] = d220
							ps300.OverlayValues[221] = d221
							ps300.OverlayValues[222] = d222
							ps300.OverlayValues[287] = d287
							ps300.OverlayValues[288] = d288
							ps300.OverlayValues[289] = d289
							ps300.OverlayValues[290] = d290
							ps300.OverlayValues[291] = d291
							ps300.OverlayValues[292] = d292
							ps300.OverlayValues[293] = d293
							ps300.OverlayValues[294] = d294
							ps300.OverlayValues[295] = d295
							ps300.OverlayValues[296] = d296
							ps300.OverlayValues[297] = d297
							ps300.OverlayValues[298] = d298
							ps300.OverlayValues[299] = d299
							return bbs[12].RenderPS(ps300)
						}
						if ps.General {
						}
						ps301 := PhiState{General: ps.General}
						ps301.OverlayValues = make([]JITValueDesc, 300)
						ps301.OverlayValues[0] = d0
						ps301.OverlayValues[1] = d1
						ps301.OverlayValues[2] = d2
						ps301.OverlayValues[3] = d3
						ps301.OverlayValues[13] = d13
						ps301.OverlayValues[14] = d14
						ps301.OverlayValues[16] = d16
						ps301.OverlayValues[17] = d17
						ps301.OverlayValues[18] = d18
						ps301.OverlayValues[20] = d20
						ps301.OverlayValues[21] = d21
						ps301.OverlayValues[22] = d22
						ps301.OverlayValues[40] = d40
						ps301.OverlayValues[41] = d41
						ps301.OverlayValues[42] = d42
						ps301.OverlayValues[43] = d43
						ps301.OverlayValues[65] = d65
						ps301.OverlayValues[66] = d66
						ps301.OverlayValues[67] = d67
						ps301.OverlayValues[68] = d68
						ps301.OverlayValues[69] = d69
						ps301.OverlayValues[70] = d70
						ps301.OverlayValues[71] = d71
						ps301.OverlayValues[72] = d72
						ps301.OverlayValues[73] = d73
						ps301.OverlayValues[74] = d74
						ps301.OverlayValues[106] = d106
						ps301.OverlayValues[139] = d139
						ps301.OverlayValues[140] = d140
						ps301.OverlayValues[141] = d141
						ps301.OverlayValues[142] = d142
						ps301.OverlayValues[143] = d143
						ps301.OverlayValues[144] = d144
						ps301.OverlayValues[145] = d145
						ps301.OverlayValues[146] = d146
						ps301.OverlayValues[147] = d147
						ps301.OverlayValues[148] = d148
						ps301.OverlayValues[149] = d149
						ps301.OverlayValues[150] = d150
						ps301.OverlayValues[151] = d151
						ps301.OverlayValues[152] = d152
						ps301.OverlayValues[153] = d153
						ps301.OverlayValues[154] = d154
						ps301.OverlayValues[155] = d155
						ps301.OverlayValues[156] = d156
						ps301.OverlayValues[157] = d157
						ps301.OverlayValues[158] = d158
						ps301.OverlayValues[211] = d211
						ps301.OverlayValues[212] = d212
						ps301.OverlayValues[213] = d213
						ps301.OverlayValues[214] = d214
						ps301.OverlayValues[215] = d215
						ps301.OverlayValues[216] = d216
						ps301.OverlayValues[217] = d217
						ps301.OverlayValues[218] = d218
						ps301.OverlayValues[219] = d219
						ps301.OverlayValues[220] = d220
						ps301.OverlayValues[221] = d221
						ps301.OverlayValues[222] = d222
						ps301.OverlayValues[287] = d287
						ps301.OverlayValues[288] = d288
						ps301.OverlayValues[289] = d289
						ps301.OverlayValues[290] = d290
						ps301.OverlayValues[291] = d291
						ps301.OverlayValues[292] = d292
						ps301.OverlayValues[293] = d293
						ps301.OverlayValues[294] = d294
						ps301.OverlayValues[295] = d295
						ps301.OverlayValues[296] = d296
						ps301.OverlayValues[297] = d297
						ps301.OverlayValues[298] = d298
						ps301.OverlayValues[299] = d299
						return bbs[15].RenderPS(ps301)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					lbl40 := ctx.ReserveLabel()
					lbl41 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d299.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl40)
					ctx.EmitJmp(lbl41)
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl41)
					ctx.EmitJmp(lbl16)
					ps302 := PhiState{General: true}
					ps302.OverlayValues = make([]JITValueDesc, 300)
					ps302.OverlayValues[0] = d0
					ps302.OverlayValues[1] = d1
					ps302.OverlayValues[2] = d2
					ps302.OverlayValues[3] = d3
					ps302.OverlayValues[13] = d13
					ps302.OverlayValues[14] = d14
					ps302.OverlayValues[16] = d16
					ps302.OverlayValues[17] = d17
					ps302.OverlayValues[18] = d18
					ps302.OverlayValues[20] = d20
					ps302.OverlayValues[21] = d21
					ps302.OverlayValues[22] = d22
					ps302.OverlayValues[40] = d40
					ps302.OverlayValues[41] = d41
					ps302.OverlayValues[42] = d42
					ps302.OverlayValues[43] = d43
					ps302.OverlayValues[65] = d65
					ps302.OverlayValues[66] = d66
					ps302.OverlayValues[67] = d67
					ps302.OverlayValues[68] = d68
					ps302.OverlayValues[69] = d69
					ps302.OverlayValues[70] = d70
					ps302.OverlayValues[71] = d71
					ps302.OverlayValues[72] = d72
					ps302.OverlayValues[73] = d73
					ps302.OverlayValues[74] = d74
					ps302.OverlayValues[106] = d106
					ps302.OverlayValues[139] = d139
					ps302.OverlayValues[140] = d140
					ps302.OverlayValues[141] = d141
					ps302.OverlayValues[142] = d142
					ps302.OverlayValues[143] = d143
					ps302.OverlayValues[144] = d144
					ps302.OverlayValues[145] = d145
					ps302.OverlayValues[146] = d146
					ps302.OverlayValues[147] = d147
					ps302.OverlayValues[148] = d148
					ps302.OverlayValues[149] = d149
					ps302.OverlayValues[150] = d150
					ps302.OverlayValues[151] = d151
					ps302.OverlayValues[152] = d152
					ps302.OverlayValues[153] = d153
					ps302.OverlayValues[154] = d154
					ps302.OverlayValues[155] = d155
					ps302.OverlayValues[156] = d156
					ps302.OverlayValues[157] = d157
					ps302.OverlayValues[158] = d158
					ps302.OverlayValues[211] = d211
					ps302.OverlayValues[212] = d212
					ps302.OverlayValues[213] = d213
					ps302.OverlayValues[214] = d214
					ps302.OverlayValues[215] = d215
					ps302.OverlayValues[216] = d216
					ps302.OverlayValues[217] = d217
					ps302.OverlayValues[218] = d218
					ps302.OverlayValues[219] = d219
					ps302.OverlayValues[220] = d220
					ps302.OverlayValues[221] = d221
					ps302.OverlayValues[222] = d222
					ps302.OverlayValues[287] = d287
					ps302.OverlayValues[288] = d288
					ps302.OverlayValues[289] = d289
					ps302.OverlayValues[290] = d290
					ps302.OverlayValues[291] = d291
					ps302.OverlayValues[292] = d292
					ps302.OverlayValues[293] = d293
					ps302.OverlayValues[294] = d294
					ps302.OverlayValues[295] = d295
					ps302.OverlayValues[296] = d296
					ps302.OverlayValues[297] = d297
					ps302.OverlayValues[298] = d298
					ps302.OverlayValues[299] = d299
					ps303 := PhiState{General: true}
					ps303.OverlayValues = make([]JITValueDesc, 300)
					ps303.OverlayValues[0] = d0
					ps303.OverlayValues[1] = d1
					ps303.OverlayValues[2] = d2
					ps303.OverlayValues[3] = d3
					ps303.OverlayValues[13] = d13
					ps303.OverlayValues[14] = d14
					ps303.OverlayValues[16] = d16
					ps303.OverlayValues[17] = d17
					ps303.OverlayValues[18] = d18
					ps303.OverlayValues[20] = d20
					ps303.OverlayValues[21] = d21
					ps303.OverlayValues[22] = d22
					ps303.OverlayValues[40] = d40
					ps303.OverlayValues[41] = d41
					ps303.OverlayValues[42] = d42
					ps303.OverlayValues[43] = d43
					ps303.OverlayValues[65] = d65
					ps303.OverlayValues[66] = d66
					ps303.OverlayValues[67] = d67
					ps303.OverlayValues[68] = d68
					ps303.OverlayValues[69] = d69
					ps303.OverlayValues[70] = d70
					ps303.OverlayValues[71] = d71
					ps303.OverlayValues[72] = d72
					ps303.OverlayValues[73] = d73
					ps303.OverlayValues[74] = d74
					ps303.OverlayValues[106] = d106
					ps303.OverlayValues[139] = d139
					ps303.OverlayValues[140] = d140
					ps303.OverlayValues[141] = d141
					ps303.OverlayValues[142] = d142
					ps303.OverlayValues[143] = d143
					ps303.OverlayValues[144] = d144
					ps303.OverlayValues[145] = d145
					ps303.OverlayValues[146] = d146
					ps303.OverlayValues[147] = d147
					ps303.OverlayValues[148] = d148
					ps303.OverlayValues[149] = d149
					ps303.OverlayValues[150] = d150
					ps303.OverlayValues[151] = d151
					ps303.OverlayValues[152] = d152
					ps303.OverlayValues[153] = d153
					ps303.OverlayValues[154] = d154
					ps303.OverlayValues[155] = d155
					ps303.OverlayValues[156] = d156
					ps303.OverlayValues[157] = d157
					ps303.OverlayValues[158] = d158
					ps303.OverlayValues[211] = d211
					ps303.OverlayValues[212] = d212
					ps303.OverlayValues[213] = d213
					ps303.OverlayValues[214] = d214
					ps303.OverlayValues[215] = d215
					ps303.OverlayValues[216] = d216
					ps303.OverlayValues[217] = d217
					ps303.OverlayValues[218] = d218
					ps303.OverlayValues[219] = d219
					ps303.OverlayValues[220] = d220
					ps303.OverlayValues[221] = d221
					ps303.OverlayValues[222] = d222
					ps303.OverlayValues[287] = d287
					ps303.OverlayValues[288] = d288
					ps303.OverlayValues[289] = d289
					ps303.OverlayValues[290] = d290
					ps303.OverlayValues[291] = d291
					ps303.OverlayValues[292] = d292
					ps303.OverlayValues[293] = d293
					ps303.OverlayValues[294] = d294
					ps303.OverlayValues[295] = d295
					ps303.OverlayValues[296] = d296
					ps303.OverlayValues[297] = d297
					ps303.OverlayValues[298] = d298
					ps303.OverlayValues[299] = d299
					snap304 := d0
					snap305 := d1
					snap306 := d2
					snap307 := d3
					snap308 := d13
					snap309 := d14
					snap310 := d16
					snap311 := d17
					snap312 := d18
					snap313 := d20
					snap314 := d21
					snap315 := d22
					snap316 := d40
					snap317 := d41
					snap318 := d42
					snap319 := d43
					snap320 := d65
					snap321 := d66
					snap322 := d67
					snap323 := d68
					snap324 := d69
					snap325 := d70
					snap326 := d71
					snap327 := d72
					snap328 := d73
					snap329 := d74
					snap330 := d106
					snap331 := d139
					snap332 := d140
					snap333 := d141
					snap334 := d142
					snap335 := d143
					snap336 := d144
					snap337 := d145
					snap338 := d146
					snap339 := d147
					snap340 := d148
					snap341 := d149
					snap342 := d150
					snap343 := d151
					snap344 := d152
					snap345 := d153
					snap346 := d154
					snap347 := d155
					snap348 := d156
					snap349 := d157
					snap350 := d158
					snap351 := d211
					snap352 := d212
					snap353 := d213
					snap354 := d214
					snap355 := d215
					snap356 := d216
					snap357 := d217
					snap358 := d218
					snap359 := d219
					snap360 := d220
					snap361 := d221
					snap362 := d222
					snap363 := d287
					snap364 := d288
					snap365 := d289
					snap366 := d290
					snap367 := d291
					snap368 := d292
					snap369 := d293
					snap370 := d294
					snap371 := d295
					snap372 := d296
					snap373 := d297
					snap374 := d298
					snap375 := d299
					alloc376 := ctx.SnapshotAllocState()
					if !bbs[15].Rendered {
						bbs[15].RenderPS(ps303)
					}
					ctx.RestoreAllocState(alloc376)
					d0 = snap304
					d1 = snap305
					d2 = snap306
					d3 = snap307
					d13 = snap308
					d14 = snap309
					d16 = snap310
					d17 = snap311
					d18 = snap312
					d20 = snap313
					d21 = snap314
					d22 = snap315
					d40 = snap316
					d41 = snap317
					d42 = snap318
					d43 = snap319
					d65 = snap320
					d66 = snap321
					d67 = snap322
					d68 = snap323
					d69 = snap324
					d70 = snap325
					d71 = snap326
					d72 = snap327
					d73 = snap328
					d74 = snap329
					d106 = snap330
					d139 = snap331
					d140 = snap332
					d141 = snap333
					d142 = snap334
					d143 = snap335
					d144 = snap336
					d145 = snap337
					d146 = snap338
					d147 = snap339
					d148 = snap340
					d149 = snap341
					d150 = snap342
					d151 = snap343
					d152 = snap344
					d153 = snap345
					d154 = snap346
					d155 = snap347
					d156 = snap348
					d157 = snap349
					d158 = snap350
					d211 = snap351
					d212 = snap352
					d213 = snap353
					d214 = snap354
					d215 = snap355
					d216 = snap356
					d217 = snap357
					d218 = snap358
					d219 = snap359
					d220 = snap360
					d221 = snap361
					d222 = snap362
					d287 = snap363
					d288 = snap364
					d289 = snap365
					d290 = snap366
					d291 = snap367
					d292 = snap368
					d293 = snap369
					d294 = snap370
					d295 = snap371
					d296 = snap372
					d297 = snap373
					d298 = snap374
					d299 = snap375
					if !bbs[12].Rendered {
						return bbs[12].RenderPS(ps302)
					}
					return result
					ctx.FreeDesc(&d298)
					return result
				}
				bbs[14].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[14].VisitCount >= 0 {
							ps.General = true
							return bbs[14].RenderPS(ps)
						}
					}
					bbs[14].VisitCount++
					if ps.General {
						if bbs[14].Rendered {
							ctx.EmitJmp(lbl15)
							return result
						}
						bbs[14].Rendered = true
						bbs[14].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_14 = bbs[14].Address
						ctx.MarkLabel(lbl15)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					bbpos_5_0 := int32(-1)
					_ = bbpos_5_0
					lbl42 := ctx.ReserveLabel()
					_ = lbl42
					bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl42)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d377 JITValueDesc
					if d70.Loc == LocImm {
						d377 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() / 3600000000000)}
					} else {
						r12 := ctx.AllocRegExcept(d70.Reg)
						ctx.EmitMovRegReg(r12, d70.Reg)
						ctx.EmitIdivRegImm(r12, 3600000000000)
						d377 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
						ctx.BindReg(r12, &d377)
					}
					if d377.Loc == LocReg && d70.Loc == LocReg && d377.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d378 JITValueDesc
					if d70.Loc == LocImm {
						d378 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() % 3600000000000)}
					} else {
						ctx.EmitIremRegImm(d70.Reg, 3600000000000)
						d378 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d70.Reg}
						ctx.BindReg(d70.Reg, &d378)
					}
					if d378.Loc == LocReg && d70.Loc == LocReg && d378.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d377)
					ctx.EnsureDesc(&d377)
					var d379 JITValueDesc
					if d377.Loc == LocImm {
						d379 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d377.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d377.Reg)
						d379 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d377.Reg}
						ctx.BindReg(d377.Reg, &d379)
					}
					ctx.FreeDesc(&d377)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d378)
					ctx.EnsureDesc(&d378)
					var d380 JITValueDesc
					if d378.Loc == LocImm {
						d380 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d378.Imm.Int()))}
					} else {
						ctx.EmitCvtInt64ToFloat64(RegX0, d378.Reg)
						d380 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d378.Reg}
						ctx.BindReg(d378.Reg, &d380)
					}
					ctx.FreeDesc(&d378)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d380)
					var d381 JITValueDesc
					if d380.Loc == LocImm {
						d381 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d380.Imm.Float() / 3.6e+12)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4794699203894837248))
						ctx.EmitDivFloat64(d380.Reg, RegR11)
						d381 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d380.Reg}
						ctx.BindReg(d380.Reg, &d381)
					}
					if d381.Loc == LocReg && d380.Loc == LocReg && d381.Reg == d380.Reg {
						ctx.TransferReg(d380.Reg)
						d380.Loc = LocNone
					}
					ctx.FreeDesc(&d380)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d379)
					ctx.EnsureDesc(&d381)
					ctx.EnsureDescsTogether(&d379, &d381)
					var d382 JITValueDesc
					if d379.Loc == LocImm && d381.Loc == LocImm {
						d382 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d379.Imm.Float() + d381.Imm.Float())}
					} else if d379.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d381.Reg)
						_, xBits := d379.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d381.Reg)
						d382 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d382)
					} else if d381.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d379.Reg)
						ctx.EmitMovRegReg(scratch, d379.Reg)
						_, yBits := d381.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d382 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d382)
					} else {
						r13 := ctx.AllocRegExcept(d379.Reg, d381.Reg)
						ctx.EmitMovRegReg(r13, d379.Reg)
						ctx.EmitAddFloat64(r13, d381.Reg)
						d382 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r13}
						ctx.BindReg(r13, &d382)
					}
					if d382.Loc == LocReg && d379.Loc == LocReg && d382.Reg == d379.Reg {
						ctx.TransferReg(d379.Reg)
						d379.Loc = LocNone
					}
					ctx.FreeDesc(&d379)
					ctx.FreeDesc(&d381)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d382)
					ctx.FreeDesc(&d70)
					ctx.EnsureDesc(&d382)
					var d383 JITValueDesc
					if d382.Loc == LocImm {
						d383 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d382.Imm.Float() / 168)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4640114991075164160))
						ctx.EmitDivFloat64(d382.Reg, RegR11)
						d383 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d382.Reg}
						ctx.BindReg(d382.Reg, &d383)
					}
					if d383.Loc == LocReg && d382.Loc == LocReg && d383.Reg == d382.Reg {
						ctx.TransferReg(d382.Reg)
						d382.Loc = LocNone
					}
					ctx.FreeDesc(&d382)
					ctx.EnsureDesc(&d383)
					ctx.EnsureDesc(&d383)
					var d384 JITValueDesc
					if d383.Loc == LocImm {
						d384 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d383.Imm.Float()))}
					} else {
						r14 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r14, d383.Reg)
						d384 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d384)
					}
					ctx.FreeDesc(&d383)
					ctx.EnsureDesc(&d384)
					if d384.Loc == LocImm {
						ctx.EmitMakeInt(result, d384)
					} else {
						ctx.EmitMovToReg(result.Reg2, d384)
						d385 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d385)
						if d384.Loc == LocReg && d384.Reg != result.Reg2 {
							ctx.FreeReg(d384.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[15].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[15].VisitCount >= 0 {
							ps.General = true
							return bbs[15].RenderPS(ps)
						}
					}
					bbs[15].VisitCount++
					if ps.General {
						if bbs[15].Rendered {
							ctx.EmitJmp(lbl16)
							return result
						}
						bbs[15].Rendered = true
						bbs[15].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_15 = bbs[15].Address
						ctx.MarkLabel(lbl16)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != LocNone {
						d380 = ps.OverlayValues[380]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != LocNone {
						d382 = ps.OverlayValues[382]
					}
					if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != LocNone {
						d383 = ps.OverlayValues[383]
					}
					if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != LocNone {
						d384 = ps.OverlayValues[384]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					d386 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					var d387 JITValueDesc
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
					d388 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d387}, 1)
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
							if ps.General {
							}
							ps390 := PhiState{General: ps.General}
							ps390.OverlayValues = make([]JITValueDesc, 390)
							ps390.OverlayValues[0] = d0
							ps390.OverlayValues[1] = d1
							ps390.OverlayValues[2] = d2
							ps390.OverlayValues[3] = d3
							ps390.OverlayValues[13] = d13
							ps390.OverlayValues[14] = d14
							ps390.OverlayValues[16] = d16
							ps390.OverlayValues[17] = d17
							ps390.OverlayValues[18] = d18
							ps390.OverlayValues[20] = d20
							ps390.OverlayValues[21] = d21
							ps390.OverlayValues[22] = d22
							ps390.OverlayValues[40] = d40
							ps390.OverlayValues[41] = d41
							ps390.OverlayValues[42] = d42
							ps390.OverlayValues[43] = d43
							ps390.OverlayValues[65] = d65
							ps390.OverlayValues[66] = d66
							ps390.OverlayValues[67] = d67
							ps390.OverlayValues[68] = d68
							ps390.OverlayValues[69] = d69
							ps390.OverlayValues[70] = d70
							ps390.OverlayValues[71] = d71
							ps390.OverlayValues[72] = d72
							ps390.OverlayValues[73] = d73
							ps390.OverlayValues[74] = d74
							ps390.OverlayValues[106] = d106
							ps390.OverlayValues[139] = d139
							ps390.OverlayValues[140] = d140
							ps390.OverlayValues[141] = d141
							ps390.OverlayValues[142] = d142
							ps390.OverlayValues[143] = d143
							ps390.OverlayValues[144] = d144
							ps390.OverlayValues[145] = d145
							ps390.OverlayValues[146] = d146
							ps390.OverlayValues[147] = d147
							ps390.OverlayValues[148] = d148
							ps390.OverlayValues[149] = d149
							ps390.OverlayValues[150] = d150
							ps390.OverlayValues[151] = d151
							ps390.OverlayValues[152] = d152
							ps390.OverlayValues[153] = d153
							ps390.OverlayValues[154] = d154
							ps390.OverlayValues[155] = d155
							ps390.OverlayValues[156] = d156
							ps390.OverlayValues[157] = d157
							ps390.OverlayValues[158] = d158
							ps390.OverlayValues[211] = d211
							ps390.OverlayValues[212] = d212
							ps390.OverlayValues[213] = d213
							ps390.OverlayValues[214] = d214
							ps390.OverlayValues[215] = d215
							ps390.OverlayValues[216] = d216
							ps390.OverlayValues[217] = d217
							ps390.OverlayValues[218] = d218
							ps390.OverlayValues[219] = d219
							ps390.OverlayValues[220] = d220
							ps390.OverlayValues[221] = d221
							ps390.OverlayValues[222] = d222
							ps390.OverlayValues[287] = d287
							ps390.OverlayValues[288] = d288
							ps390.OverlayValues[289] = d289
							ps390.OverlayValues[290] = d290
							ps390.OverlayValues[291] = d291
							ps390.OverlayValues[292] = d292
							ps390.OverlayValues[293] = d293
							ps390.OverlayValues[294] = d294
							ps390.OverlayValues[295] = d295
							ps390.OverlayValues[296] = d296
							ps390.OverlayValues[297] = d297
							ps390.OverlayValues[298] = d298
							ps390.OverlayValues[299] = d299
							ps390.OverlayValues[377] = d377
							ps390.OverlayValues[378] = d378
							ps390.OverlayValues[379] = d379
							ps390.OverlayValues[380] = d380
							ps390.OverlayValues[381] = d381
							ps390.OverlayValues[382] = d382
							ps390.OverlayValues[383] = d383
							ps390.OverlayValues[384] = d384
							ps390.OverlayValues[385] = d385
							ps390.OverlayValues[386] = d386
							ps390.OverlayValues[387] = d387
							ps390.OverlayValues[388] = d388
							ps390.OverlayValues[389] = d389
							return bbs[14].RenderPS(ps390)
						}
						if ps.General {
						}
						ps391 := PhiState{General: ps.General}
						ps391.OverlayValues = make([]JITValueDesc, 390)
						ps391.OverlayValues[0] = d0
						ps391.OverlayValues[1] = d1
						ps391.OverlayValues[2] = d2
						ps391.OverlayValues[3] = d3
						ps391.OverlayValues[13] = d13
						ps391.OverlayValues[14] = d14
						ps391.OverlayValues[16] = d16
						ps391.OverlayValues[17] = d17
						ps391.OverlayValues[18] = d18
						ps391.OverlayValues[20] = d20
						ps391.OverlayValues[21] = d21
						ps391.OverlayValues[22] = d22
						ps391.OverlayValues[40] = d40
						ps391.OverlayValues[41] = d41
						ps391.OverlayValues[42] = d42
						ps391.OverlayValues[43] = d43
						ps391.OverlayValues[65] = d65
						ps391.OverlayValues[66] = d66
						ps391.OverlayValues[67] = d67
						ps391.OverlayValues[68] = d68
						ps391.OverlayValues[69] = d69
						ps391.OverlayValues[70] = d70
						ps391.OverlayValues[71] = d71
						ps391.OverlayValues[72] = d72
						ps391.OverlayValues[73] = d73
						ps391.OverlayValues[74] = d74
						ps391.OverlayValues[106] = d106
						ps391.OverlayValues[139] = d139
						ps391.OverlayValues[140] = d140
						ps391.OverlayValues[141] = d141
						ps391.OverlayValues[142] = d142
						ps391.OverlayValues[143] = d143
						ps391.OverlayValues[144] = d144
						ps391.OverlayValues[145] = d145
						ps391.OverlayValues[146] = d146
						ps391.OverlayValues[147] = d147
						ps391.OverlayValues[148] = d148
						ps391.OverlayValues[149] = d149
						ps391.OverlayValues[150] = d150
						ps391.OverlayValues[151] = d151
						ps391.OverlayValues[152] = d152
						ps391.OverlayValues[153] = d153
						ps391.OverlayValues[154] = d154
						ps391.OverlayValues[155] = d155
						ps391.OverlayValues[156] = d156
						ps391.OverlayValues[157] = d157
						ps391.OverlayValues[158] = d158
						ps391.OverlayValues[211] = d211
						ps391.OverlayValues[212] = d212
						ps391.OverlayValues[213] = d213
						ps391.OverlayValues[214] = d214
						ps391.OverlayValues[215] = d215
						ps391.OverlayValues[216] = d216
						ps391.OverlayValues[217] = d217
						ps391.OverlayValues[218] = d218
						ps391.OverlayValues[219] = d219
						ps391.OverlayValues[220] = d220
						ps391.OverlayValues[221] = d221
						ps391.OverlayValues[222] = d222
						ps391.OverlayValues[287] = d287
						ps391.OverlayValues[288] = d288
						ps391.OverlayValues[289] = d289
						ps391.OverlayValues[290] = d290
						ps391.OverlayValues[291] = d291
						ps391.OverlayValues[292] = d292
						ps391.OverlayValues[293] = d293
						ps391.OverlayValues[294] = d294
						ps391.OverlayValues[295] = d295
						ps391.OverlayValues[296] = d296
						ps391.OverlayValues[297] = d297
						ps391.OverlayValues[298] = d298
						ps391.OverlayValues[299] = d299
						ps391.OverlayValues[377] = d377
						ps391.OverlayValues[378] = d378
						ps391.OverlayValues[379] = d379
						ps391.OverlayValues[380] = d380
						ps391.OverlayValues[381] = d381
						ps391.OverlayValues[382] = d382
						ps391.OverlayValues[383] = d383
						ps391.OverlayValues[384] = d384
						ps391.OverlayValues[385] = d385
						ps391.OverlayValues[386] = d386
						ps391.OverlayValues[387] = d387
						ps391.OverlayValues[388] = d388
						ps391.OverlayValues[389] = d389
						return bbs[17].RenderPS(ps391)
					}
					if !ps.General {
						ps.General = true
						return bbs[15].RenderPS(ps)
					}
					lbl43 := ctx.ReserveLabel()
					lbl44 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d389.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl43)
					ctx.EmitJmp(lbl44)
					ctx.MarkLabel(lbl43)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl44)
					ctx.EmitJmp(lbl18)
					ps392 := PhiState{General: true}
					ps392.OverlayValues = make([]JITValueDesc, 390)
					ps392.OverlayValues[0] = d0
					ps392.OverlayValues[1] = d1
					ps392.OverlayValues[2] = d2
					ps392.OverlayValues[3] = d3
					ps392.OverlayValues[13] = d13
					ps392.OverlayValues[14] = d14
					ps392.OverlayValues[16] = d16
					ps392.OverlayValues[17] = d17
					ps392.OverlayValues[18] = d18
					ps392.OverlayValues[20] = d20
					ps392.OverlayValues[21] = d21
					ps392.OverlayValues[22] = d22
					ps392.OverlayValues[40] = d40
					ps392.OverlayValues[41] = d41
					ps392.OverlayValues[42] = d42
					ps392.OverlayValues[43] = d43
					ps392.OverlayValues[65] = d65
					ps392.OverlayValues[66] = d66
					ps392.OverlayValues[67] = d67
					ps392.OverlayValues[68] = d68
					ps392.OverlayValues[69] = d69
					ps392.OverlayValues[70] = d70
					ps392.OverlayValues[71] = d71
					ps392.OverlayValues[72] = d72
					ps392.OverlayValues[73] = d73
					ps392.OverlayValues[74] = d74
					ps392.OverlayValues[106] = d106
					ps392.OverlayValues[139] = d139
					ps392.OverlayValues[140] = d140
					ps392.OverlayValues[141] = d141
					ps392.OverlayValues[142] = d142
					ps392.OverlayValues[143] = d143
					ps392.OverlayValues[144] = d144
					ps392.OverlayValues[145] = d145
					ps392.OverlayValues[146] = d146
					ps392.OverlayValues[147] = d147
					ps392.OverlayValues[148] = d148
					ps392.OverlayValues[149] = d149
					ps392.OverlayValues[150] = d150
					ps392.OverlayValues[151] = d151
					ps392.OverlayValues[152] = d152
					ps392.OverlayValues[153] = d153
					ps392.OverlayValues[154] = d154
					ps392.OverlayValues[155] = d155
					ps392.OverlayValues[156] = d156
					ps392.OverlayValues[157] = d157
					ps392.OverlayValues[158] = d158
					ps392.OverlayValues[211] = d211
					ps392.OverlayValues[212] = d212
					ps392.OverlayValues[213] = d213
					ps392.OverlayValues[214] = d214
					ps392.OverlayValues[215] = d215
					ps392.OverlayValues[216] = d216
					ps392.OverlayValues[217] = d217
					ps392.OverlayValues[218] = d218
					ps392.OverlayValues[219] = d219
					ps392.OverlayValues[220] = d220
					ps392.OverlayValues[221] = d221
					ps392.OverlayValues[222] = d222
					ps392.OverlayValues[287] = d287
					ps392.OverlayValues[288] = d288
					ps392.OverlayValues[289] = d289
					ps392.OverlayValues[290] = d290
					ps392.OverlayValues[291] = d291
					ps392.OverlayValues[292] = d292
					ps392.OverlayValues[293] = d293
					ps392.OverlayValues[294] = d294
					ps392.OverlayValues[295] = d295
					ps392.OverlayValues[296] = d296
					ps392.OverlayValues[297] = d297
					ps392.OverlayValues[298] = d298
					ps392.OverlayValues[299] = d299
					ps392.OverlayValues[377] = d377
					ps392.OverlayValues[378] = d378
					ps392.OverlayValues[379] = d379
					ps392.OverlayValues[380] = d380
					ps392.OverlayValues[381] = d381
					ps392.OverlayValues[382] = d382
					ps392.OverlayValues[383] = d383
					ps392.OverlayValues[384] = d384
					ps392.OverlayValues[385] = d385
					ps392.OverlayValues[386] = d386
					ps392.OverlayValues[387] = d387
					ps392.OverlayValues[388] = d388
					ps392.OverlayValues[389] = d389
					ps393 := PhiState{General: true}
					ps393.OverlayValues = make([]JITValueDesc, 390)
					ps393.OverlayValues[0] = d0
					ps393.OverlayValues[1] = d1
					ps393.OverlayValues[2] = d2
					ps393.OverlayValues[3] = d3
					ps393.OverlayValues[13] = d13
					ps393.OverlayValues[14] = d14
					ps393.OverlayValues[16] = d16
					ps393.OverlayValues[17] = d17
					ps393.OverlayValues[18] = d18
					ps393.OverlayValues[20] = d20
					ps393.OverlayValues[21] = d21
					ps393.OverlayValues[22] = d22
					ps393.OverlayValues[40] = d40
					ps393.OverlayValues[41] = d41
					ps393.OverlayValues[42] = d42
					ps393.OverlayValues[43] = d43
					ps393.OverlayValues[65] = d65
					ps393.OverlayValues[66] = d66
					ps393.OverlayValues[67] = d67
					ps393.OverlayValues[68] = d68
					ps393.OverlayValues[69] = d69
					ps393.OverlayValues[70] = d70
					ps393.OverlayValues[71] = d71
					ps393.OverlayValues[72] = d72
					ps393.OverlayValues[73] = d73
					ps393.OverlayValues[74] = d74
					ps393.OverlayValues[106] = d106
					ps393.OverlayValues[139] = d139
					ps393.OverlayValues[140] = d140
					ps393.OverlayValues[141] = d141
					ps393.OverlayValues[142] = d142
					ps393.OverlayValues[143] = d143
					ps393.OverlayValues[144] = d144
					ps393.OverlayValues[145] = d145
					ps393.OverlayValues[146] = d146
					ps393.OverlayValues[147] = d147
					ps393.OverlayValues[148] = d148
					ps393.OverlayValues[149] = d149
					ps393.OverlayValues[150] = d150
					ps393.OverlayValues[151] = d151
					ps393.OverlayValues[152] = d152
					ps393.OverlayValues[153] = d153
					ps393.OverlayValues[154] = d154
					ps393.OverlayValues[155] = d155
					ps393.OverlayValues[156] = d156
					ps393.OverlayValues[157] = d157
					ps393.OverlayValues[158] = d158
					ps393.OverlayValues[211] = d211
					ps393.OverlayValues[212] = d212
					ps393.OverlayValues[213] = d213
					ps393.OverlayValues[214] = d214
					ps393.OverlayValues[215] = d215
					ps393.OverlayValues[216] = d216
					ps393.OverlayValues[217] = d217
					ps393.OverlayValues[218] = d218
					ps393.OverlayValues[219] = d219
					ps393.OverlayValues[220] = d220
					ps393.OverlayValues[221] = d221
					ps393.OverlayValues[222] = d222
					ps393.OverlayValues[287] = d287
					ps393.OverlayValues[288] = d288
					ps393.OverlayValues[289] = d289
					ps393.OverlayValues[290] = d290
					ps393.OverlayValues[291] = d291
					ps393.OverlayValues[292] = d292
					ps393.OverlayValues[293] = d293
					ps393.OverlayValues[294] = d294
					ps393.OverlayValues[295] = d295
					ps393.OverlayValues[296] = d296
					ps393.OverlayValues[297] = d297
					ps393.OverlayValues[298] = d298
					ps393.OverlayValues[299] = d299
					ps393.OverlayValues[377] = d377
					ps393.OverlayValues[378] = d378
					ps393.OverlayValues[379] = d379
					ps393.OverlayValues[380] = d380
					ps393.OverlayValues[381] = d381
					ps393.OverlayValues[382] = d382
					ps393.OverlayValues[383] = d383
					ps393.OverlayValues[384] = d384
					ps393.OverlayValues[385] = d385
					ps393.OverlayValues[386] = d386
					ps393.OverlayValues[387] = d387
					ps393.OverlayValues[388] = d388
					ps393.OverlayValues[389] = d389
					snap394 := d0
					snap395 := d1
					snap396 := d2
					snap397 := d3
					snap398 := d13
					snap399 := d14
					snap400 := d16
					snap401 := d17
					snap402 := d18
					snap403 := d20
					snap404 := d21
					snap405 := d22
					snap406 := d40
					snap407 := d41
					snap408 := d42
					snap409 := d43
					snap410 := d65
					snap411 := d66
					snap412 := d67
					snap413 := d68
					snap414 := d69
					snap415 := d70
					snap416 := d71
					snap417 := d72
					snap418 := d73
					snap419 := d74
					snap420 := d106
					snap421 := d139
					snap422 := d140
					snap423 := d141
					snap424 := d142
					snap425 := d143
					snap426 := d144
					snap427 := d145
					snap428 := d146
					snap429 := d147
					snap430 := d148
					snap431 := d149
					snap432 := d150
					snap433 := d151
					snap434 := d152
					snap435 := d153
					snap436 := d154
					snap437 := d155
					snap438 := d156
					snap439 := d157
					snap440 := d158
					snap441 := d211
					snap442 := d212
					snap443 := d213
					snap444 := d214
					snap445 := d215
					snap446 := d216
					snap447 := d217
					snap448 := d218
					snap449 := d219
					snap450 := d220
					snap451 := d221
					snap452 := d222
					snap453 := d287
					snap454 := d288
					snap455 := d289
					snap456 := d290
					snap457 := d291
					snap458 := d292
					snap459 := d293
					snap460 := d294
					snap461 := d295
					snap462 := d296
					snap463 := d297
					snap464 := d298
					snap465 := d299
					snap466 := d377
					snap467 := d378
					snap468 := d379
					snap469 := d380
					snap470 := d381
					snap471 := d382
					snap472 := d383
					snap473 := d384
					snap474 := d385
					snap475 := d386
					snap476 := d387
					snap477 := d388
					snap478 := d389
					alloc479 := ctx.SnapshotAllocState()
					if !bbs[17].Rendered {
						bbs[17].RenderPS(ps393)
					}
					ctx.RestoreAllocState(alloc479)
					d0 = snap394
					d1 = snap395
					d2 = snap396
					d3 = snap397
					d13 = snap398
					d14 = snap399
					d16 = snap400
					d17 = snap401
					d18 = snap402
					d20 = snap403
					d21 = snap404
					d22 = snap405
					d40 = snap406
					d41 = snap407
					d42 = snap408
					d43 = snap409
					d65 = snap410
					d66 = snap411
					d67 = snap412
					d68 = snap413
					d69 = snap414
					d70 = snap415
					d71 = snap416
					d72 = snap417
					d73 = snap418
					d74 = snap419
					d106 = snap420
					d139 = snap421
					d140 = snap422
					d141 = snap423
					d142 = snap424
					d143 = snap425
					d144 = snap426
					d145 = snap427
					d146 = snap428
					d147 = snap429
					d148 = snap430
					d149 = snap431
					d150 = snap432
					d151 = snap433
					d152 = snap434
					d153 = snap435
					d154 = snap436
					d155 = snap437
					d156 = snap438
					d157 = snap439
					d158 = snap440
					d211 = snap441
					d212 = snap442
					d213 = snap443
					d214 = snap444
					d215 = snap445
					d216 = snap446
					d217 = snap447
					d218 = snap448
					d219 = snap449
					d220 = snap450
					d221 = snap451
					d222 = snap452
					d287 = snap453
					d288 = snap454
					d289 = snap455
					d290 = snap456
					d291 = snap457
					d292 = snap458
					d293 = snap459
					d294 = snap460
					d295 = snap461
					d296 = snap462
					d297 = snap463
					d298 = snap464
					d299 = snap465
					d377 = snap466
					d378 = snap467
					d379 = snap468
					d380 = snap469
					d381 = snap470
					d382 = snap471
					d383 = snap472
					d384 = snap473
					d385 = snap474
					d386 = snap475
					d387 = snap476
					d388 = snap477
					d389 = snap478
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps392)
					}
					return result
					ctx.FreeDesc(&d388)
					return result
				}
				bbs[16].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[16].VisitCount >= 0 {
							ps.General = true
							return bbs[16].RenderPS(ps)
						}
					}
					bbs[16].VisitCount++
					if ps.General {
						if bbs[16].Rendered {
							ctx.EmitJmp(lbl17)
							return result
						}
						bbs[16].Rendered = true
						bbs[16].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_16 = bbs[16].Address
						ctx.MarkLabel(lbl17)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != LocNone {
						d380 = ps.OverlayValues[380]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != LocNone {
						d382 = ps.OverlayValues[382]
					}
					if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != LocNone {
						d383 = ps.OverlayValues[383]
					}
					if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != LocNone {
						d384 = ps.OverlayValues[384]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d16)
					callResults480 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d16}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d481 = callResults480[0]
					_ = d481
					d482 = callResults480[1]
					_ = d482
					d483 = callResults480[2]
					_ = d483
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocRegTriple && d20.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d20)
					callResults484 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d20}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d485 = callResults484[0]
					_ = d485
					d486 = callResults484[1]
					_ = d486
					d487 = callResults484[2]
					_ = d487
					ctx.EnsureDesc(&d485)
					ctx.EnsureDesc(&d481)
					ctx.EnsureDescsTogether(&d485, &d481)
					var d488 JITValueDesc
					if d485.Loc == LocImm && d481.Loc == LocImm {
						d488 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d485.Imm.Int() - d481.Imm.Int())}
					} else if d481.Loc == LocImm && d481.Imm.Int() == 0 {
						r15 := ctx.AllocRegExcept(d485.Reg)
						ctx.EmitMovRegReg(r15, d485.Reg)
						d488 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d488)
					} else if d485.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d481.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d485.Imm.Int()))
						ctx.EmitSubInt64(scratch, d481.Reg)
						d488 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d488)
					} else if d481.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d485.Reg)
						ctx.EmitMovRegReg(scratch, d485.Reg)
						if d481.Imm.Int() >= -2147483648 && d481.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d481.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d481.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d488 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d488)
					} else {
						r16 := ctx.AllocRegExcept(d485.Reg, d481.Reg)
						ctx.EmitMovRegReg(r16, d485.Reg)
						ctx.EmitSubInt64(r16, d481.Reg)
						d488 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d488)
					}
					if d488.Loc == LocReg && d485.Loc == LocReg && d488.Reg == d485.Reg {
						ctx.TransferReg(d485.Reg)
						d485.Loc = LocNone
					}
					ctx.FreeDesc(&d485)
					ctx.FreeDesc(&d481)
					ctx.EnsureDesc(&d488)
					ctx.EnsureDesc(&d488)
					var d489 JITValueDesc
					if d488.Loc == LocImm {
						d489 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d488.Imm.Int() * 12)}
					} else {
						ctx.EmitImulRegImm32(d488.Reg, int32(12))
						d489 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d488.Reg}
						ctx.BindReg(d488.Reg, &d489)
					}
					if d489.Loc == LocReg && d488.Loc == LocReg && d489.Reg == d488.Reg {
						ctx.TransferReg(d488.Reg)
						d488.Loc = LocNone
					}
					ctx.FreeDesc(&d488)
					ctx.EnsureDesc(&d486)
					ctx.EnsureDesc(&d482)
					ctx.EnsureDescsTogether(&d486, &d482)
					var d490 JITValueDesc
					if d486.Loc == LocImm && d482.Loc == LocImm {
						d490 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d486.Imm.Int() - d482.Imm.Int())}
					} else if d482.Loc == LocImm && d482.Imm.Int() == 0 {
						r17 := ctx.AllocRegExcept(d486.Reg)
						ctx.EmitMovRegReg(r17, d486.Reg)
						d490 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d490)
					} else if d486.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d482.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d486.Imm.Int()))
						ctx.EmitSubInt64(scratch, d482.Reg)
						d490 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d490)
					} else if d482.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d486.Reg)
						ctx.EmitMovRegReg(scratch, d486.Reg)
						if d482.Imm.Int() >= -2147483648 && d482.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d482.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d482.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d490 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d490)
					} else {
						r18 := ctx.AllocRegExcept(d486.Reg, d482.Reg)
						ctx.EmitMovRegReg(r18, d486.Reg)
						ctx.EmitSubInt64(r18, d482.Reg)
						d490 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d490)
					}
					if d490.Loc == LocReg && d486.Loc == LocReg && d490.Reg == d486.Reg {
						ctx.TransferReg(d486.Reg)
						d486.Loc = LocNone
					}
					ctx.FreeDesc(&d486)
					ctx.FreeDesc(&d482)
					ctx.EnsureDesc(&d490)
					ctx.FreeDesc(&d490)
					ctx.EnsureDesc(&d489)
					ctx.EnsureDesc(&d490)
					ctx.EnsureDescsTogether(&d489, &d490)
					var d491 JITValueDesc
					if d489.Loc == LocImm && d490.Loc == LocImm {
						d491 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d489.Imm.Int() + d490.Imm.Int())}
					} else if d490.Loc == LocImm && d490.Imm.Int() == 0 {
						r19 := ctx.AllocRegExcept(d489.Reg)
						ctx.EmitMovRegReg(r19, d489.Reg)
						d491 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d491)
					} else if d489.Loc == LocImm && d489.Imm.Int() == 0 {
						d491 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d490.Reg}
						ctx.BindReg(d490.Reg, &d491)
					} else if d489.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d490.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d489.Imm.Int()))
						ctx.EmitAddInt64(scratch, d490.Reg)
						d491 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d491)
					} else if d490.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d489.Reg)
						ctx.EmitMovRegReg(scratch, d489.Reg)
						if d490.Imm.Int() >= -2147483648 && d490.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d490.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d490.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d491 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d491)
					} else {
						r20 := ctx.AllocRegExcept(d489.Reg, d490.Reg)
						ctx.EmitMovRegReg(r20, d489.Reg)
						ctx.EmitAddInt64(r20, d490.Reg)
						d491 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
						ctx.BindReg(r20, &d491)
					}
					if d491.Loc == LocReg && d489.Loc == LocReg && d491.Reg == d489.Reg {
						ctx.TransferReg(d489.Reg)
						d489.Loc = LocNone
					}
					ctx.FreeDesc(&d489)
					ctx.FreeDesc(&d490)
					ctx.EnsureDesc(&d491)
					ctx.EnsureDesc(&d491)
					ctx.EnsureDesc(&d491)
					if d491.Loc == LocImm {
						ctx.EmitMakeInt(result, d491)
					} else {
						ctx.EmitMovToReg(result.Reg2, d491)
						d493 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d493)
						if d491.Loc == LocReg && d491.Reg != result.Reg2 {
							ctx.FreeReg(d491.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[17].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[17].VisitCount >= 0 {
							ps.General = true
							return bbs[17].RenderPS(ps)
						}
					}
					bbs[17].VisitCount++
					if ps.General {
						if bbs[17].Rendered {
							ctx.EmitJmp(lbl18)
							return result
						}
						bbs[17].Rendered = true
						bbs[17].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_17 = bbs[17].Address
						ctx.MarkLabel(lbl18)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != LocNone {
						d380 = ps.OverlayValues[380]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != LocNone {
						d382 = ps.OverlayValues[382]
					}
					if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != LocNone {
						d383 = ps.OverlayValues[383]
					}
					if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != LocNone {
						d384 = ps.OverlayValues[384]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != LocNone {
						d481 = ps.OverlayValues[481]
					}
					if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != LocNone {
						d482 = ps.OverlayValues[482]
					}
					if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != LocNone {
						d483 = ps.OverlayValues[483]
					}
					if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != LocNone {
						d485 = ps.OverlayValues[485]
					}
					if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != LocNone {
						d486 = ps.OverlayValues[486]
					}
					if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != LocNone {
						d487 = ps.OverlayValues[487]
					}
					if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != LocNone {
						d488 = ps.OverlayValues[488]
					}
					if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != LocNone {
						d489 = ps.OverlayValues[489]
					}
					if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != LocNone {
						d490 = ps.OverlayValues[490]
					}
					if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != LocNone {
						d491 = ps.OverlayValues[491]
					}
					if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != LocNone {
						d492 = ps.OverlayValues[492]
					}
					if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != LocNone {
						d493 = ps.OverlayValues[493]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					d494 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					var d495 JITValueDesc
					if d494.Loc == LocImm {
						ctx.TrackImm(d494.Imm)
						ptrWord, _ := d494.Imm.RawWords()
						d495 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d495.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d495.Reg2, uint64(len(d494.Imm.String())))
						ctx.BindReg(d495.Reg, &d495)
						ctx.BindReg(d495.Reg2, &d495)
					} else {
						d495 = d494
					}
					d496 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d495}, 1)
					ctx.EmitAndRegImm32(d496.Reg, 1)
					d496.Type = tagBool
					ctx.BindReg(d496.Reg, &d496)
					d497 = d496
					ctx.EnsureDesc(&d497)
					if d497.Loc != LocImm && d497.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d497.Loc == LocImm {
						if d497.Imm.Bool() {
							if ps.General {
							}
							ps498 := PhiState{General: ps.General}
							ps498.OverlayValues = make([]JITValueDesc, 498)
							ps498.OverlayValues[0] = d0
							ps498.OverlayValues[1] = d1
							ps498.OverlayValues[2] = d2
							ps498.OverlayValues[3] = d3
							ps498.OverlayValues[13] = d13
							ps498.OverlayValues[14] = d14
							ps498.OverlayValues[16] = d16
							ps498.OverlayValues[17] = d17
							ps498.OverlayValues[18] = d18
							ps498.OverlayValues[20] = d20
							ps498.OverlayValues[21] = d21
							ps498.OverlayValues[22] = d22
							ps498.OverlayValues[40] = d40
							ps498.OverlayValues[41] = d41
							ps498.OverlayValues[42] = d42
							ps498.OverlayValues[43] = d43
							ps498.OverlayValues[65] = d65
							ps498.OverlayValues[66] = d66
							ps498.OverlayValues[67] = d67
							ps498.OverlayValues[68] = d68
							ps498.OverlayValues[69] = d69
							ps498.OverlayValues[70] = d70
							ps498.OverlayValues[71] = d71
							ps498.OverlayValues[72] = d72
							ps498.OverlayValues[73] = d73
							ps498.OverlayValues[74] = d74
							ps498.OverlayValues[106] = d106
							ps498.OverlayValues[139] = d139
							ps498.OverlayValues[140] = d140
							ps498.OverlayValues[141] = d141
							ps498.OverlayValues[142] = d142
							ps498.OverlayValues[143] = d143
							ps498.OverlayValues[144] = d144
							ps498.OverlayValues[145] = d145
							ps498.OverlayValues[146] = d146
							ps498.OverlayValues[147] = d147
							ps498.OverlayValues[148] = d148
							ps498.OverlayValues[149] = d149
							ps498.OverlayValues[150] = d150
							ps498.OverlayValues[151] = d151
							ps498.OverlayValues[152] = d152
							ps498.OverlayValues[153] = d153
							ps498.OverlayValues[154] = d154
							ps498.OverlayValues[155] = d155
							ps498.OverlayValues[156] = d156
							ps498.OverlayValues[157] = d157
							ps498.OverlayValues[158] = d158
							ps498.OverlayValues[211] = d211
							ps498.OverlayValues[212] = d212
							ps498.OverlayValues[213] = d213
							ps498.OverlayValues[214] = d214
							ps498.OverlayValues[215] = d215
							ps498.OverlayValues[216] = d216
							ps498.OverlayValues[217] = d217
							ps498.OverlayValues[218] = d218
							ps498.OverlayValues[219] = d219
							ps498.OverlayValues[220] = d220
							ps498.OverlayValues[221] = d221
							ps498.OverlayValues[222] = d222
							ps498.OverlayValues[287] = d287
							ps498.OverlayValues[288] = d288
							ps498.OverlayValues[289] = d289
							ps498.OverlayValues[290] = d290
							ps498.OverlayValues[291] = d291
							ps498.OverlayValues[292] = d292
							ps498.OverlayValues[293] = d293
							ps498.OverlayValues[294] = d294
							ps498.OverlayValues[295] = d295
							ps498.OverlayValues[296] = d296
							ps498.OverlayValues[297] = d297
							ps498.OverlayValues[298] = d298
							ps498.OverlayValues[299] = d299
							ps498.OverlayValues[377] = d377
							ps498.OverlayValues[378] = d378
							ps498.OverlayValues[379] = d379
							ps498.OverlayValues[380] = d380
							ps498.OverlayValues[381] = d381
							ps498.OverlayValues[382] = d382
							ps498.OverlayValues[383] = d383
							ps498.OverlayValues[384] = d384
							ps498.OverlayValues[385] = d385
							ps498.OverlayValues[386] = d386
							ps498.OverlayValues[387] = d387
							ps498.OverlayValues[388] = d388
							ps498.OverlayValues[389] = d389
							ps498.OverlayValues[481] = d481
							ps498.OverlayValues[482] = d482
							ps498.OverlayValues[483] = d483
							ps498.OverlayValues[485] = d485
							ps498.OverlayValues[486] = d486
							ps498.OverlayValues[487] = d487
							ps498.OverlayValues[488] = d488
							ps498.OverlayValues[489] = d489
							ps498.OverlayValues[490] = d490
							ps498.OverlayValues[491] = d491
							ps498.OverlayValues[492] = d492
							ps498.OverlayValues[493] = d493
							ps498.OverlayValues[494] = d494
							ps498.OverlayValues[495] = d495
							ps498.OverlayValues[496] = d496
							ps498.OverlayValues[497] = d497
							return bbs[16].RenderPS(ps498)
						}
						if ps.General {
						}
						ps499 := PhiState{General: ps.General}
						ps499.OverlayValues = make([]JITValueDesc, 498)
						ps499.OverlayValues[0] = d0
						ps499.OverlayValues[1] = d1
						ps499.OverlayValues[2] = d2
						ps499.OverlayValues[3] = d3
						ps499.OverlayValues[13] = d13
						ps499.OverlayValues[14] = d14
						ps499.OverlayValues[16] = d16
						ps499.OverlayValues[17] = d17
						ps499.OverlayValues[18] = d18
						ps499.OverlayValues[20] = d20
						ps499.OverlayValues[21] = d21
						ps499.OverlayValues[22] = d22
						ps499.OverlayValues[40] = d40
						ps499.OverlayValues[41] = d41
						ps499.OverlayValues[42] = d42
						ps499.OverlayValues[43] = d43
						ps499.OverlayValues[65] = d65
						ps499.OverlayValues[66] = d66
						ps499.OverlayValues[67] = d67
						ps499.OverlayValues[68] = d68
						ps499.OverlayValues[69] = d69
						ps499.OverlayValues[70] = d70
						ps499.OverlayValues[71] = d71
						ps499.OverlayValues[72] = d72
						ps499.OverlayValues[73] = d73
						ps499.OverlayValues[74] = d74
						ps499.OverlayValues[106] = d106
						ps499.OverlayValues[139] = d139
						ps499.OverlayValues[140] = d140
						ps499.OverlayValues[141] = d141
						ps499.OverlayValues[142] = d142
						ps499.OverlayValues[143] = d143
						ps499.OverlayValues[144] = d144
						ps499.OverlayValues[145] = d145
						ps499.OverlayValues[146] = d146
						ps499.OverlayValues[147] = d147
						ps499.OverlayValues[148] = d148
						ps499.OverlayValues[149] = d149
						ps499.OverlayValues[150] = d150
						ps499.OverlayValues[151] = d151
						ps499.OverlayValues[152] = d152
						ps499.OverlayValues[153] = d153
						ps499.OverlayValues[154] = d154
						ps499.OverlayValues[155] = d155
						ps499.OverlayValues[156] = d156
						ps499.OverlayValues[157] = d157
						ps499.OverlayValues[158] = d158
						ps499.OverlayValues[211] = d211
						ps499.OverlayValues[212] = d212
						ps499.OverlayValues[213] = d213
						ps499.OverlayValues[214] = d214
						ps499.OverlayValues[215] = d215
						ps499.OverlayValues[216] = d216
						ps499.OverlayValues[217] = d217
						ps499.OverlayValues[218] = d218
						ps499.OverlayValues[219] = d219
						ps499.OverlayValues[220] = d220
						ps499.OverlayValues[221] = d221
						ps499.OverlayValues[222] = d222
						ps499.OverlayValues[287] = d287
						ps499.OverlayValues[288] = d288
						ps499.OverlayValues[289] = d289
						ps499.OverlayValues[290] = d290
						ps499.OverlayValues[291] = d291
						ps499.OverlayValues[292] = d292
						ps499.OverlayValues[293] = d293
						ps499.OverlayValues[294] = d294
						ps499.OverlayValues[295] = d295
						ps499.OverlayValues[296] = d296
						ps499.OverlayValues[297] = d297
						ps499.OverlayValues[298] = d298
						ps499.OverlayValues[299] = d299
						ps499.OverlayValues[377] = d377
						ps499.OverlayValues[378] = d378
						ps499.OverlayValues[379] = d379
						ps499.OverlayValues[380] = d380
						ps499.OverlayValues[381] = d381
						ps499.OverlayValues[382] = d382
						ps499.OverlayValues[383] = d383
						ps499.OverlayValues[384] = d384
						ps499.OverlayValues[385] = d385
						ps499.OverlayValues[386] = d386
						ps499.OverlayValues[387] = d387
						ps499.OverlayValues[388] = d388
						ps499.OverlayValues[389] = d389
						ps499.OverlayValues[481] = d481
						ps499.OverlayValues[482] = d482
						ps499.OverlayValues[483] = d483
						ps499.OverlayValues[485] = d485
						ps499.OverlayValues[486] = d486
						ps499.OverlayValues[487] = d487
						ps499.OverlayValues[488] = d488
						ps499.OverlayValues[489] = d489
						ps499.OverlayValues[490] = d490
						ps499.OverlayValues[491] = d491
						ps499.OverlayValues[492] = d492
						ps499.OverlayValues[493] = d493
						ps499.OverlayValues[494] = d494
						ps499.OverlayValues[495] = d495
						ps499.OverlayValues[496] = d496
						ps499.OverlayValues[497] = d497
						return bbs[19].RenderPS(ps499)
					}
					if !ps.General {
						ps.General = true
						return bbs[17].RenderPS(ps)
					}
					lbl45 := ctx.ReserveLabel()
					lbl46 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d497.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl45)
					ctx.EmitJmp(lbl46)
					ctx.MarkLabel(lbl45)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl46)
					ctx.EmitJmp(lbl20)
					ps500 := PhiState{General: true}
					ps500.OverlayValues = make([]JITValueDesc, 498)
					ps500.OverlayValues[0] = d0
					ps500.OverlayValues[1] = d1
					ps500.OverlayValues[2] = d2
					ps500.OverlayValues[3] = d3
					ps500.OverlayValues[13] = d13
					ps500.OverlayValues[14] = d14
					ps500.OverlayValues[16] = d16
					ps500.OverlayValues[17] = d17
					ps500.OverlayValues[18] = d18
					ps500.OverlayValues[20] = d20
					ps500.OverlayValues[21] = d21
					ps500.OverlayValues[22] = d22
					ps500.OverlayValues[40] = d40
					ps500.OverlayValues[41] = d41
					ps500.OverlayValues[42] = d42
					ps500.OverlayValues[43] = d43
					ps500.OverlayValues[65] = d65
					ps500.OverlayValues[66] = d66
					ps500.OverlayValues[67] = d67
					ps500.OverlayValues[68] = d68
					ps500.OverlayValues[69] = d69
					ps500.OverlayValues[70] = d70
					ps500.OverlayValues[71] = d71
					ps500.OverlayValues[72] = d72
					ps500.OverlayValues[73] = d73
					ps500.OverlayValues[74] = d74
					ps500.OverlayValues[106] = d106
					ps500.OverlayValues[139] = d139
					ps500.OverlayValues[140] = d140
					ps500.OverlayValues[141] = d141
					ps500.OverlayValues[142] = d142
					ps500.OverlayValues[143] = d143
					ps500.OverlayValues[144] = d144
					ps500.OverlayValues[145] = d145
					ps500.OverlayValues[146] = d146
					ps500.OverlayValues[147] = d147
					ps500.OverlayValues[148] = d148
					ps500.OverlayValues[149] = d149
					ps500.OverlayValues[150] = d150
					ps500.OverlayValues[151] = d151
					ps500.OverlayValues[152] = d152
					ps500.OverlayValues[153] = d153
					ps500.OverlayValues[154] = d154
					ps500.OverlayValues[155] = d155
					ps500.OverlayValues[156] = d156
					ps500.OverlayValues[157] = d157
					ps500.OverlayValues[158] = d158
					ps500.OverlayValues[211] = d211
					ps500.OverlayValues[212] = d212
					ps500.OverlayValues[213] = d213
					ps500.OverlayValues[214] = d214
					ps500.OverlayValues[215] = d215
					ps500.OverlayValues[216] = d216
					ps500.OverlayValues[217] = d217
					ps500.OverlayValues[218] = d218
					ps500.OverlayValues[219] = d219
					ps500.OverlayValues[220] = d220
					ps500.OverlayValues[221] = d221
					ps500.OverlayValues[222] = d222
					ps500.OverlayValues[287] = d287
					ps500.OverlayValues[288] = d288
					ps500.OverlayValues[289] = d289
					ps500.OverlayValues[290] = d290
					ps500.OverlayValues[291] = d291
					ps500.OverlayValues[292] = d292
					ps500.OverlayValues[293] = d293
					ps500.OverlayValues[294] = d294
					ps500.OverlayValues[295] = d295
					ps500.OverlayValues[296] = d296
					ps500.OverlayValues[297] = d297
					ps500.OverlayValues[298] = d298
					ps500.OverlayValues[299] = d299
					ps500.OverlayValues[377] = d377
					ps500.OverlayValues[378] = d378
					ps500.OverlayValues[379] = d379
					ps500.OverlayValues[380] = d380
					ps500.OverlayValues[381] = d381
					ps500.OverlayValues[382] = d382
					ps500.OverlayValues[383] = d383
					ps500.OverlayValues[384] = d384
					ps500.OverlayValues[385] = d385
					ps500.OverlayValues[386] = d386
					ps500.OverlayValues[387] = d387
					ps500.OverlayValues[388] = d388
					ps500.OverlayValues[389] = d389
					ps500.OverlayValues[481] = d481
					ps500.OverlayValues[482] = d482
					ps500.OverlayValues[483] = d483
					ps500.OverlayValues[485] = d485
					ps500.OverlayValues[486] = d486
					ps500.OverlayValues[487] = d487
					ps500.OverlayValues[488] = d488
					ps500.OverlayValues[489] = d489
					ps500.OverlayValues[490] = d490
					ps500.OverlayValues[491] = d491
					ps500.OverlayValues[492] = d492
					ps500.OverlayValues[493] = d493
					ps500.OverlayValues[494] = d494
					ps500.OverlayValues[495] = d495
					ps500.OverlayValues[496] = d496
					ps500.OverlayValues[497] = d497
					ps501 := PhiState{General: true}
					ps501.OverlayValues = make([]JITValueDesc, 498)
					ps501.OverlayValues[0] = d0
					ps501.OverlayValues[1] = d1
					ps501.OverlayValues[2] = d2
					ps501.OverlayValues[3] = d3
					ps501.OverlayValues[13] = d13
					ps501.OverlayValues[14] = d14
					ps501.OverlayValues[16] = d16
					ps501.OverlayValues[17] = d17
					ps501.OverlayValues[18] = d18
					ps501.OverlayValues[20] = d20
					ps501.OverlayValues[21] = d21
					ps501.OverlayValues[22] = d22
					ps501.OverlayValues[40] = d40
					ps501.OverlayValues[41] = d41
					ps501.OverlayValues[42] = d42
					ps501.OverlayValues[43] = d43
					ps501.OverlayValues[65] = d65
					ps501.OverlayValues[66] = d66
					ps501.OverlayValues[67] = d67
					ps501.OverlayValues[68] = d68
					ps501.OverlayValues[69] = d69
					ps501.OverlayValues[70] = d70
					ps501.OverlayValues[71] = d71
					ps501.OverlayValues[72] = d72
					ps501.OverlayValues[73] = d73
					ps501.OverlayValues[74] = d74
					ps501.OverlayValues[106] = d106
					ps501.OverlayValues[139] = d139
					ps501.OverlayValues[140] = d140
					ps501.OverlayValues[141] = d141
					ps501.OverlayValues[142] = d142
					ps501.OverlayValues[143] = d143
					ps501.OverlayValues[144] = d144
					ps501.OverlayValues[145] = d145
					ps501.OverlayValues[146] = d146
					ps501.OverlayValues[147] = d147
					ps501.OverlayValues[148] = d148
					ps501.OverlayValues[149] = d149
					ps501.OverlayValues[150] = d150
					ps501.OverlayValues[151] = d151
					ps501.OverlayValues[152] = d152
					ps501.OverlayValues[153] = d153
					ps501.OverlayValues[154] = d154
					ps501.OverlayValues[155] = d155
					ps501.OverlayValues[156] = d156
					ps501.OverlayValues[157] = d157
					ps501.OverlayValues[158] = d158
					ps501.OverlayValues[211] = d211
					ps501.OverlayValues[212] = d212
					ps501.OverlayValues[213] = d213
					ps501.OverlayValues[214] = d214
					ps501.OverlayValues[215] = d215
					ps501.OverlayValues[216] = d216
					ps501.OverlayValues[217] = d217
					ps501.OverlayValues[218] = d218
					ps501.OverlayValues[219] = d219
					ps501.OverlayValues[220] = d220
					ps501.OverlayValues[221] = d221
					ps501.OverlayValues[222] = d222
					ps501.OverlayValues[287] = d287
					ps501.OverlayValues[288] = d288
					ps501.OverlayValues[289] = d289
					ps501.OverlayValues[290] = d290
					ps501.OverlayValues[291] = d291
					ps501.OverlayValues[292] = d292
					ps501.OverlayValues[293] = d293
					ps501.OverlayValues[294] = d294
					ps501.OverlayValues[295] = d295
					ps501.OverlayValues[296] = d296
					ps501.OverlayValues[297] = d297
					ps501.OverlayValues[298] = d298
					ps501.OverlayValues[299] = d299
					ps501.OverlayValues[377] = d377
					ps501.OverlayValues[378] = d378
					ps501.OverlayValues[379] = d379
					ps501.OverlayValues[380] = d380
					ps501.OverlayValues[381] = d381
					ps501.OverlayValues[382] = d382
					ps501.OverlayValues[383] = d383
					ps501.OverlayValues[384] = d384
					ps501.OverlayValues[385] = d385
					ps501.OverlayValues[386] = d386
					ps501.OverlayValues[387] = d387
					ps501.OverlayValues[388] = d388
					ps501.OverlayValues[389] = d389
					ps501.OverlayValues[481] = d481
					ps501.OverlayValues[482] = d482
					ps501.OverlayValues[483] = d483
					ps501.OverlayValues[485] = d485
					ps501.OverlayValues[486] = d486
					ps501.OverlayValues[487] = d487
					ps501.OverlayValues[488] = d488
					ps501.OverlayValues[489] = d489
					ps501.OverlayValues[490] = d490
					ps501.OverlayValues[491] = d491
					ps501.OverlayValues[492] = d492
					ps501.OverlayValues[493] = d493
					ps501.OverlayValues[494] = d494
					ps501.OverlayValues[495] = d495
					ps501.OverlayValues[496] = d496
					ps501.OverlayValues[497] = d497
					snap502 := d0
					snap503 := d1
					snap504 := d2
					snap505 := d3
					snap506 := d13
					snap507 := d14
					snap508 := d16
					snap509 := d17
					snap510 := d18
					snap511 := d20
					snap512 := d21
					snap513 := d22
					snap514 := d40
					snap515 := d41
					snap516 := d42
					snap517 := d43
					snap518 := d65
					snap519 := d66
					snap520 := d67
					snap521 := d68
					snap522 := d69
					snap523 := d70
					snap524 := d71
					snap525 := d72
					snap526 := d73
					snap527 := d74
					snap528 := d106
					snap529 := d139
					snap530 := d140
					snap531 := d141
					snap532 := d142
					snap533 := d143
					snap534 := d144
					snap535 := d145
					snap536 := d146
					snap537 := d147
					snap538 := d148
					snap539 := d149
					snap540 := d150
					snap541 := d151
					snap542 := d152
					snap543 := d153
					snap544 := d154
					snap545 := d155
					snap546 := d156
					snap547 := d157
					snap548 := d158
					snap549 := d211
					snap550 := d212
					snap551 := d213
					snap552 := d214
					snap553 := d215
					snap554 := d216
					snap555 := d217
					snap556 := d218
					snap557 := d219
					snap558 := d220
					snap559 := d221
					snap560 := d222
					snap561 := d287
					snap562 := d288
					snap563 := d289
					snap564 := d290
					snap565 := d291
					snap566 := d292
					snap567 := d293
					snap568 := d294
					snap569 := d295
					snap570 := d296
					snap571 := d297
					snap572 := d298
					snap573 := d299
					snap574 := d377
					snap575 := d378
					snap576 := d379
					snap577 := d380
					snap578 := d381
					snap579 := d382
					snap580 := d383
					snap581 := d384
					snap582 := d385
					snap583 := d386
					snap584 := d387
					snap585 := d388
					snap586 := d389
					snap587 := d481
					snap588 := d482
					snap589 := d483
					snap590 := d485
					snap591 := d486
					snap592 := d487
					snap593 := d488
					snap594 := d489
					snap595 := d490
					snap596 := d491
					snap597 := d492
					snap598 := d493
					snap599 := d494
					snap600 := d495
					snap601 := d496
					snap602 := d497
					alloc603 := ctx.SnapshotAllocState()
					if !bbs[19].Rendered {
						bbs[19].RenderPS(ps501)
					}
					ctx.RestoreAllocState(alloc603)
					d0 = snap502
					d1 = snap503
					d2 = snap504
					d3 = snap505
					d13 = snap506
					d14 = snap507
					d16 = snap508
					d17 = snap509
					d18 = snap510
					d20 = snap511
					d21 = snap512
					d22 = snap513
					d40 = snap514
					d41 = snap515
					d42 = snap516
					d43 = snap517
					d65 = snap518
					d66 = snap519
					d67 = snap520
					d68 = snap521
					d69 = snap522
					d70 = snap523
					d71 = snap524
					d72 = snap525
					d73 = snap526
					d74 = snap527
					d106 = snap528
					d139 = snap529
					d140 = snap530
					d141 = snap531
					d142 = snap532
					d143 = snap533
					d144 = snap534
					d145 = snap535
					d146 = snap536
					d147 = snap537
					d148 = snap538
					d149 = snap539
					d150 = snap540
					d151 = snap541
					d152 = snap542
					d153 = snap543
					d154 = snap544
					d155 = snap545
					d156 = snap546
					d157 = snap547
					d158 = snap548
					d211 = snap549
					d212 = snap550
					d213 = snap551
					d214 = snap552
					d215 = snap553
					d216 = snap554
					d217 = snap555
					d218 = snap556
					d219 = snap557
					d220 = snap558
					d221 = snap559
					d222 = snap560
					d287 = snap561
					d288 = snap562
					d289 = snap563
					d290 = snap564
					d291 = snap565
					d292 = snap566
					d293 = snap567
					d294 = snap568
					d295 = snap569
					d296 = snap570
					d297 = snap571
					d298 = snap572
					d299 = snap573
					d377 = snap574
					d378 = snap575
					d379 = snap576
					d380 = snap577
					d381 = snap578
					d382 = snap579
					d383 = snap580
					d384 = snap581
					d385 = snap582
					d386 = snap583
					d387 = snap584
					d388 = snap585
					d389 = snap586
					d481 = snap587
					d482 = snap588
					d483 = snap589
					d485 = snap590
					d486 = snap591
					d487 = snap592
					d488 = snap593
					d489 = snap594
					d490 = snap595
					d491 = snap596
					d492 = snap597
					d493 = snap598
					d494 = snap599
					d495 = snap600
					d496 = snap601
					d497 = snap602
					if !bbs[16].Rendered {
						return bbs[16].RenderPS(ps500)
					}
					return result
					ctx.FreeDesc(&d496)
					return result
				}
				bbs[18].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[18].VisitCount >= 0 {
							ps.General = true
							return bbs[18].RenderPS(ps)
						}
					}
					bbs[18].VisitCount++
					if ps.General {
						if bbs[18].Rendered {
							ctx.EmitJmp(lbl19)
							return result
						}
						bbs[18].Rendered = true
						bbs[18].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_18 = bbs[18].Address
						ctx.MarkLabel(lbl19)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != LocNone {
						d380 = ps.OverlayValues[380]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != LocNone {
						d382 = ps.OverlayValues[382]
					}
					if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != LocNone {
						d383 = ps.OverlayValues[383]
					}
					if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != LocNone {
						d384 = ps.OverlayValues[384]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != LocNone {
						d481 = ps.OverlayValues[481]
					}
					if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != LocNone {
						d482 = ps.OverlayValues[482]
					}
					if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != LocNone {
						d483 = ps.OverlayValues[483]
					}
					if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != LocNone {
						d485 = ps.OverlayValues[485]
					}
					if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != LocNone {
						d486 = ps.OverlayValues[486]
					}
					if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != LocNone {
						d487 = ps.OverlayValues[487]
					}
					if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != LocNone {
						d488 = ps.OverlayValues[488]
					}
					if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != LocNone {
						d489 = ps.OverlayValues[489]
					}
					if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != LocNone {
						d490 = ps.OverlayValues[490]
					}
					if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != LocNone {
						d491 = ps.OverlayValues[491]
					}
					if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != LocNone {
						d492 = ps.OverlayValues[492]
					}
					if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != LocNone {
						d493 = ps.OverlayValues[493]
					}
					if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != LocNone {
						d494 = ps.OverlayValues[494]
					}
					if len(ps.OverlayValues) > 495 && ps.OverlayValues[495].Loc != LocNone {
						d495 = ps.OverlayValues[495]
					}
					if len(ps.OverlayValues) > 496 && ps.OverlayValues[496].Loc != LocNone {
						d496 = ps.OverlayValues[496]
					}
					if len(ps.OverlayValues) > 497 && ps.OverlayValues[497].Loc != LocNone {
						d497 = ps.OverlayValues[497]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocRegTriple && d16.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d16)
					callResults604 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d16}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d605 = callResults604[0]
					_ = d605
					d606 = callResults604[1]
					_ = d606
					d607 = callResults604[2]
					_ = d607
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocRegTriple && d20.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d20)
					callResults608 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d20}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d609 = callResults608[0]
					_ = d609
					d610 = callResults608[1]
					_ = d610
					d611 = callResults608[2]
					_ = d611
					ctx.FreeDesc(&d20)
					ctx.EnsureDesc(&d609)
					ctx.EnsureDesc(&d605)
					ctx.EnsureDescsTogether(&d609, &d605)
					var d612 JITValueDesc
					if d609.Loc == LocImm && d605.Loc == LocImm {
						d612 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d609.Imm.Int() - d605.Imm.Int())}
					} else if d605.Loc == LocImm && d605.Imm.Int() == 0 {
						r21 := ctx.AllocRegExcept(d609.Reg)
						ctx.EmitMovRegReg(r21, d609.Reg)
						d612 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
						ctx.BindReg(r21, &d612)
					} else if d609.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d605.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d609.Imm.Int()))
						ctx.EmitSubInt64(scratch, d605.Reg)
						d612 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d612)
					} else if d605.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d609.Reg)
						ctx.EmitMovRegReg(scratch, d609.Reg)
						if d605.Imm.Int() >= -2147483648 && d605.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d605.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d605.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d612 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d612)
					} else {
						r22 := ctx.AllocRegExcept(d609.Reg, d605.Reg)
						ctx.EmitMovRegReg(r22, d609.Reg)
						ctx.EmitSubInt64(r22, d605.Reg)
						d612 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r22}
						ctx.BindReg(r22, &d612)
					}
					if d612.Loc == LocReg && d609.Loc == LocReg && d612.Reg == d609.Reg {
						ctx.TransferReg(d609.Reg)
						d609.Loc = LocNone
					}
					ctx.FreeDesc(&d609)
					ctx.FreeDesc(&d605)
					ctx.EnsureDesc(&d612)
					ctx.EnsureDesc(&d612)
					ctx.EnsureDesc(&d612)
					if d612.Loc == LocImm {
						ctx.EmitMakeInt(result, d612)
					} else {
						ctx.EmitMovToReg(result.Reg2, d612)
						d614 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d614)
						if d612.Loc == LocReg && d612.Reg != result.Reg2 {
							ctx.FreeReg(d612.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[19].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[19].VisitCount >= 0 {
							ps.General = true
							return bbs[19].RenderPS(ps)
						}
					}
					bbs[19].VisitCount++
					if ps.General {
						if bbs[19].Rendered {
							ctx.EmitJmp(lbl20)
							return result
						}
						bbs[19].Rendered = true
						bbs[19].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_19 = bbs[19].Address
						ctx.MarkLabel(lbl20)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != LocNone {
						d380 = ps.OverlayValues[380]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != LocNone {
						d382 = ps.OverlayValues[382]
					}
					if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != LocNone {
						d383 = ps.OverlayValues[383]
					}
					if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != LocNone {
						d384 = ps.OverlayValues[384]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != LocNone {
						d481 = ps.OverlayValues[481]
					}
					if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != LocNone {
						d482 = ps.OverlayValues[482]
					}
					if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != LocNone {
						d483 = ps.OverlayValues[483]
					}
					if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != LocNone {
						d485 = ps.OverlayValues[485]
					}
					if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != LocNone {
						d486 = ps.OverlayValues[486]
					}
					if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != LocNone {
						d487 = ps.OverlayValues[487]
					}
					if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != LocNone {
						d488 = ps.OverlayValues[488]
					}
					if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != LocNone {
						d489 = ps.OverlayValues[489]
					}
					if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != LocNone {
						d490 = ps.OverlayValues[490]
					}
					if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != LocNone {
						d491 = ps.OverlayValues[491]
					}
					if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != LocNone {
						d492 = ps.OverlayValues[492]
					}
					if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != LocNone {
						d493 = ps.OverlayValues[493]
					}
					if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != LocNone {
						d494 = ps.OverlayValues[494]
					}
					if len(ps.OverlayValues) > 495 && ps.OverlayValues[495].Loc != LocNone {
						d495 = ps.OverlayValues[495]
					}
					if len(ps.OverlayValues) > 496 && ps.OverlayValues[496].Loc != LocNone {
						d496 = ps.OverlayValues[496]
					}
					if len(ps.OverlayValues) > 497 && ps.OverlayValues[497].Loc != LocNone {
						d497 = ps.OverlayValues[497]
					}
					if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != LocNone {
						d605 = ps.OverlayValues[605]
					}
					if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != LocNone {
						d606 = ps.OverlayValues[606]
					}
					if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != LocNone {
						d607 = ps.OverlayValues[607]
					}
					if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != LocNone {
						d609 = ps.OverlayValues[609]
					}
					if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != LocNone {
						d610 = ps.OverlayValues[610]
					}
					if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != LocNone {
						d611 = ps.OverlayValues[611]
					}
					if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != LocNone {
						d612 = ps.OverlayValues[612]
					}
					if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != LocNone {
						d613 = ps.OverlayValues[613]
					}
					if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != LocNone {
						d614 = ps.OverlayValues[614]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					d615 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					var d616 JITValueDesc
					if d615.Loc == LocImm {
						ctx.TrackImm(d615.Imm)
						ptrWord, _ := d615.Imm.RawWords()
						d616 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d616.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d616.Reg2, uint64(len(d615.Imm.String())))
						ctx.BindReg(d616.Reg, &d616)
						ctx.BindReg(d616.Reg2, &d616)
					} else {
						d616 = d615
					}
					d617 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d69, d616}, 1)
					ctx.EmitAndRegImm32(d617.Reg, 1)
					d617.Type = tagBool
					ctx.BindReg(d617.Reg, &d617)
					d618 = d617
					ctx.EnsureDesc(&d618)
					if d618.Loc != LocImm && d618.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d618.Loc == LocImm {
						if d618.Imm.Bool() {
							if ps.General {
							}
							ps619 := PhiState{General: ps.General}
							ps619.OverlayValues = make([]JITValueDesc, 619)
							ps619.OverlayValues[0] = d0
							ps619.OverlayValues[1] = d1
							ps619.OverlayValues[2] = d2
							ps619.OverlayValues[3] = d3
							ps619.OverlayValues[13] = d13
							ps619.OverlayValues[14] = d14
							ps619.OverlayValues[16] = d16
							ps619.OverlayValues[17] = d17
							ps619.OverlayValues[18] = d18
							ps619.OverlayValues[20] = d20
							ps619.OverlayValues[21] = d21
							ps619.OverlayValues[22] = d22
							ps619.OverlayValues[40] = d40
							ps619.OverlayValues[41] = d41
							ps619.OverlayValues[42] = d42
							ps619.OverlayValues[43] = d43
							ps619.OverlayValues[65] = d65
							ps619.OverlayValues[66] = d66
							ps619.OverlayValues[67] = d67
							ps619.OverlayValues[68] = d68
							ps619.OverlayValues[69] = d69
							ps619.OverlayValues[70] = d70
							ps619.OverlayValues[71] = d71
							ps619.OverlayValues[72] = d72
							ps619.OverlayValues[73] = d73
							ps619.OverlayValues[74] = d74
							ps619.OverlayValues[106] = d106
							ps619.OverlayValues[139] = d139
							ps619.OverlayValues[140] = d140
							ps619.OverlayValues[141] = d141
							ps619.OverlayValues[142] = d142
							ps619.OverlayValues[143] = d143
							ps619.OverlayValues[144] = d144
							ps619.OverlayValues[145] = d145
							ps619.OverlayValues[146] = d146
							ps619.OverlayValues[147] = d147
							ps619.OverlayValues[148] = d148
							ps619.OverlayValues[149] = d149
							ps619.OverlayValues[150] = d150
							ps619.OverlayValues[151] = d151
							ps619.OverlayValues[152] = d152
							ps619.OverlayValues[153] = d153
							ps619.OverlayValues[154] = d154
							ps619.OverlayValues[155] = d155
							ps619.OverlayValues[156] = d156
							ps619.OverlayValues[157] = d157
							ps619.OverlayValues[158] = d158
							ps619.OverlayValues[211] = d211
							ps619.OverlayValues[212] = d212
							ps619.OverlayValues[213] = d213
							ps619.OverlayValues[214] = d214
							ps619.OverlayValues[215] = d215
							ps619.OverlayValues[216] = d216
							ps619.OverlayValues[217] = d217
							ps619.OverlayValues[218] = d218
							ps619.OverlayValues[219] = d219
							ps619.OverlayValues[220] = d220
							ps619.OverlayValues[221] = d221
							ps619.OverlayValues[222] = d222
							ps619.OverlayValues[287] = d287
							ps619.OverlayValues[288] = d288
							ps619.OverlayValues[289] = d289
							ps619.OverlayValues[290] = d290
							ps619.OverlayValues[291] = d291
							ps619.OverlayValues[292] = d292
							ps619.OverlayValues[293] = d293
							ps619.OverlayValues[294] = d294
							ps619.OverlayValues[295] = d295
							ps619.OverlayValues[296] = d296
							ps619.OverlayValues[297] = d297
							ps619.OverlayValues[298] = d298
							ps619.OverlayValues[299] = d299
							ps619.OverlayValues[377] = d377
							ps619.OverlayValues[378] = d378
							ps619.OverlayValues[379] = d379
							ps619.OverlayValues[380] = d380
							ps619.OverlayValues[381] = d381
							ps619.OverlayValues[382] = d382
							ps619.OverlayValues[383] = d383
							ps619.OverlayValues[384] = d384
							ps619.OverlayValues[385] = d385
							ps619.OverlayValues[386] = d386
							ps619.OverlayValues[387] = d387
							ps619.OverlayValues[388] = d388
							ps619.OverlayValues[389] = d389
							ps619.OverlayValues[481] = d481
							ps619.OverlayValues[482] = d482
							ps619.OverlayValues[483] = d483
							ps619.OverlayValues[485] = d485
							ps619.OverlayValues[486] = d486
							ps619.OverlayValues[487] = d487
							ps619.OverlayValues[488] = d488
							ps619.OverlayValues[489] = d489
							ps619.OverlayValues[490] = d490
							ps619.OverlayValues[491] = d491
							ps619.OverlayValues[492] = d492
							ps619.OverlayValues[493] = d493
							ps619.OverlayValues[494] = d494
							ps619.OverlayValues[495] = d495
							ps619.OverlayValues[496] = d496
							ps619.OverlayValues[497] = d497
							ps619.OverlayValues[605] = d605
							ps619.OverlayValues[606] = d606
							ps619.OverlayValues[607] = d607
							ps619.OverlayValues[609] = d609
							ps619.OverlayValues[610] = d610
							ps619.OverlayValues[611] = d611
							ps619.OverlayValues[612] = d612
							ps619.OverlayValues[613] = d613
							ps619.OverlayValues[614] = d614
							ps619.OverlayValues[615] = d615
							ps619.OverlayValues[616] = d616
							ps619.OverlayValues[617] = d617
							ps619.OverlayValues[618] = d618
							return bbs[18].RenderPS(ps619)
						}
						if ps.General {
						}
						ps620 := PhiState{General: ps.General}
						ps620.OverlayValues = make([]JITValueDesc, 619)
						ps620.OverlayValues[0] = d0
						ps620.OverlayValues[1] = d1
						ps620.OverlayValues[2] = d2
						ps620.OverlayValues[3] = d3
						ps620.OverlayValues[13] = d13
						ps620.OverlayValues[14] = d14
						ps620.OverlayValues[16] = d16
						ps620.OverlayValues[17] = d17
						ps620.OverlayValues[18] = d18
						ps620.OverlayValues[20] = d20
						ps620.OverlayValues[21] = d21
						ps620.OverlayValues[22] = d22
						ps620.OverlayValues[40] = d40
						ps620.OverlayValues[41] = d41
						ps620.OverlayValues[42] = d42
						ps620.OverlayValues[43] = d43
						ps620.OverlayValues[65] = d65
						ps620.OverlayValues[66] = d66
						ps620.OverlayValues[67] = d67
						ps620.OverlayValues[68] = d68
						ps620.OverlayValues[69] = d69
						ps620.OverlayValues[70] = d70
						ps620.OverlayValues[71] = d71
						ps620.OverlayValues[72] = d72
						ps620.OverlayValues[73] = d73
						ps620.OverlayValues[74] = d74
						ps620.OverlayValues[106] = d106
						ps620.OverlayValues[139] = d139
						ps620.OverlayValues[140] = d140
						ps620.OverlayValues[141] = d141
						ps620.OverlayValues[142] = d142
						ps620.OverlayValues[143] = d143
						ps620.OverlayValues[144] = d144
						ps620.OverlayValues[145] = d145
						ps620.OverlayValues[146] = d146
						ps620.OverlayValues[147] = d147
						ps620.OverlayValues[148] = d148
						ps620.OverlayValues[149] = d149
						ps620.OverlayValues[150] = d150
						ps620.OverlayValues[151] = d151
						ps620.OverlayValues[152] = d152
						ps620.OverlayValues[153] = d153
						ps620.OverlayValues[154] = d154
						ps620.OverlayValues[155] = d155
						ps620.OverlayValues[156] = d156
						ps620.OverlayValues[157] = d157
						ps620.OverlayValues[158] = d158
						ps620.OverlayValues[211] = d211
						ps620.OverlayValues[212] = d212
						ps620.OverlayValues[213] = d213
						ps620.OverlayValues[214] = d214
						ps620.OverlayValues[215] = d215
						ps620.OverlayValues[216] = d216
						ps620.OverlayValues[217] = d217
						ps620.OverlayValues[218] = d218
						ps620.OverlayValues[219] = d219
						ps620.OverlayValues[220] = d220
						ps620.OverlayValues[221] = d221
						ps620.OverlayValues[222] = d222
						ps620.OverlayValues[287] = d287
						ps620.OverlayValues[288] = d288
						ps620.OverlayValues[289] = d289
						ps620.OverlayValues[290] = d290
						ps620.OverlayValues[291] = d291
						ps620.OverlayValues[292] = d292
						ps620.OverlayValues[293] = d293
						ps620.OverlayValues[294] = d294
						ps620.OverlayValues[295] = d295
						ps620.OverlayValues[296] = d296
						ps620.OverlayValues[297] = d297
						ps620.OverlayValues[298] = d298
						ps620.OverlayValues[299] = d299
						ps620.OverlayValues[377] = d377
						ps620.OverlayValues[378] = d378
						ps620.OverlayValues[379] = d379
						ps620.OverlayValues[380] = d380
						ps620.OverlayValues[381] = d381
						ps620.OverlayValues[382] = d382
						ps620.OverlayValues[383] = d383
						ps620.OverlayValues[384] = d384
						ps620.OverlayValues[385] = d385
						ps620.OverlayValues[386] = d386
						ps620.OverlayValues[387] = d387
						ps620.OverlayValues[388] = d388
						ps620.OverlayValues[389] = d389
						ps620.OverlayValues[481] = d481
						ps620.OverlayValues[482] = d482
						ps620.OverlayValues[483] = d483
						ps620.OverlayValues[485] = d485
						ps620.OverlayValues[486] = d486
						ps620.OverlayValues[487] = d487
						ps620.OverlayValues[488] = d488
						ps620.OverlayValues[489] = d489
						ps620.OverlayValues[490] = d490
						ps620.OverlayValues[491] = d491
						ps620.OverlayValues[492] = d492
						ps620.OverlayValues[493] = d493
						ps620.OverlayValues[494] = d494
						ps620.OverlayValues[495] = d495
						ps620.OverlayValues[496] = d496
						ps620.OverlayValues[497] = d497
						ps620.OverlayValues[605] = d605
						ps620.OverlayValues[606] = d606
						ps620.OverlayValues[607] = d607
						ps620.OverlayValues[609] = d609
						ps620.OverlayValues[610] = d610
						ps620.OverlayValues[611] = d611
						ps620.OverlayValues[612] = d612
						ps620.OverlayValues[613] = d613
						ps620.OverlayValues[614] = d614
						ps620.OverlayValues[615] = d615
						ps620.OverlayValues[616] = d616
						ps620.OverlayValues[617] = d617
						ps620.OverlayValues[618] = d618
						return bbs[20].RenderPS(ps620)
					}
					if !ps.General {
						ps.General = true
						return bbs[19].RenderPS(ps)
					}
					lbl47 := ctx.ReserveLabel()
					lbl48 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d618.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl47)
					ctx.EmitJmp(lbl48)
					ctx.MarkLabel(lbl47)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl21)
					ps621 := PhiState{General: true}
					ps621.OverlayValues = make([]JITValueDesc, 619)
					ps621.OverlayValues[0] = d0
					ps621.OverlayValues[1] = d1
					ps621.OverlayValues[2] = d2
					ps621.OverlayValues[3] = d3
					ps621.OverlayValues[13] = d13
					ps621.OverlayValues[14] = d14
					ps621.OverlayValues[16] = d16
					ps621.OverlayValues[17] = d17
					ps621.OverlayValues[18] = d18
					ps621.OverlayValues[20] = d20
					ps621.OverlayValues[21] = d21
					ps621.OverlayValues[22] = d22
					ps621.OverlayValues[40] = d40
					ps621.OverlayValues[41] = d41
					ps621.OverlayValues[42] = d42
					ps621.OverlayValues[43] = d43
					ps621.OverlayValues[65] = d65
					ps621.OverlayValues[66] = d66
					ps621.OverlayValues[67] = d67
					ps621.OverlayValues[68] = d68
					ps621.OverlayValues[69] = d69
					ps621.OverlayValues[70] = d70
					ps621.OverlayValues[71] = d71
					ps621.OverlayValues[72] = d72
					ps621.OverlayValues[73] = d73
					ps621.OverlayValues[74] = d74
					ps621.OverlayValues[106] = d106
					ps621.OverlayValues[139] = d139
					ps621.OverlayValues[140] = d140
					ps621.OverlayValues[141] = d141
					ps621.OverlayValues[142] = d142
					ps621.OverlayValues[143] = d143
					ps621.OverlayValues[144] = d144
					ps621.OverlayValues[145] = d145
					ps621.OverlayValues[146] = d146
					ps621.OverlayValues[147] = d147
					ps621.OverlayValues[148] = d148
					ps621.OverlayValues[149] = d149
					ps621.OverlayValues[150] = d150
					ps621.OverlayValues[151] = d151
					ps621.OverlayValues[152] = d152
					ps621.OverlayValues[153] = d153
					ps621.OverlayValues[154] = d154
					ps621.OverlayValues[155] = d155
					ps621.OverlayValues[156] = d156
					ps621.OverlayValues[157] = d157
					ps621.OverlayValues[158] = d158
					ps621.OverlayValues[211] = d211
					ps621.OverlayValues[212] = d212
					ps621.OverlayValues[213] = d213
					ps621.OverlayValues[214] = d214
					ps621.OverlayValues[215] = d215
					ps621.OverlayValues[216] = d216
					ps621.OverlayValues[217] = d217
					ps621.OverlayValues[218] = d218
					ps621.OverlayValues[219] = d219
					ps621.OverlayValues[220] = d220
					ps621.OverlayValues[221] = d221
					ps621.OverlayValues[222] = d222
					ps621.OverlayValues[287] = d287
					ps621.OverlayValues[288] = d288
					ps621.OverlayValues[289] = d289
					ps621.OverlayValues[290] = d290
					ps621.OverlayValues[291] = d291
					ps621.OverlayValues[292] = d292
					ps621.OverlayValues[293] = d293
					ps621.OverlayValues[294] = d294
					ps621.OverlayValues[295] = d295
					ps621.OverlayValues[296] = d296
					ps621.OverlayValues[297] = d297
					ps621.OverlayValues[298] = d298
					ps621.OverlayValues[299] = d299
					ps621.OverlayValues[377] = d377
					ps621.OverlayValues[378] = d378
					ps621.OverlayValues[379] = d379
					ps621.OverlayValues[380] = d380
					ps621.OverlayValues[381] = d381
					ps621.OverlayValues[382] = d382
					ps621.OverlayValues[383] = d383
					ps621.OverlayValues[384] = d384
					ps621.OverlayValues[385] = d385
					ps621.OverlayValues[386] = d386
					ps621.OverlayValues[387] = d387
					ps621.OverlayValues[388] = d388
					ps621.OverlayValues[389] = d389
					ps621.OverlayValues[481] = d481
					ps621.OverlayValues[482] = d482
					ps621.OverlayValues[483] = d483
					ps621.OverlayValues[485] = d485
					ps621.OverlayValues[486] = d486
					ps621.OverlayValues[487] = d487
					ps621.OverlayValues[488] = d488
					ps621.OverlayValues[489] = d489
					ps621.OverlayValues[490] = d490
					ps621.OverlayValues[491] = d491
					ps621.OverlayValues[492] = d492
					ps621.OverlayValues[493] = d493
					ps621.OverlayValues[494] = d494
					ps621.OverlayValues[495] = d495
					ps621.OverlayValues[496] = d496
					ps621.OverlayValues[497] = d497
					ps621.OverlayValues[605] = d605
					ps621.OverlayValues[606] = d606
					ps621.OverlayValues[607] = d607
					ps621.OverlayValues[609] = d609
					ps621.OverlayValues[610] = d610
					ps621.OverlayValues[611] = d611
					ps621.OverlayValues[612] = d612
					ps621.OverlayValues[613] = d613
					ps621.OverlayValues[614] = d614
					ps621.OverlayValues[615] = d615
					ps621.OverlayValues[616] = d616
					ps621.OverlayValues[617] = d617
					ps621.OverlayValues[618] = d618
					ps622 := PhiState{General: true}
					ps622.OverlayValues = make([]JITValueDesc, 619)
					ps622.OverlayValues[0] = d0
					ps622.OverlayValues[1] = d1
					ps622.OverlayValues[2] = d2
					ps622.OverlayValues[3] = d3
					ps622.OverlayValues[13] = d13
					ps622.OverlayValues[14] = d14
					ps622.OverlayValues[16] = d16
					ps622.OverlayValues[17] = d17
					ps622.OverlayValues[18] = d18
					ps622.OverlayValues[20] = d20
					ps622.OverlayValues[21] = d21
					ps622.OverlayValues[22] = d22
					ps622.OverlayValues[40] = d40
					ps622.OverlayValues[41] = d41
					ps622.OverlayValues[42] = d42
					ps622.OverlayValues[43] = d43
					ps622.OverlayValues[65] = d65
					ps622.OverlayValues[66] = d66
					ps622.OverlayValues[67] = d67
					ps622.OverlayValues[68] = d68
					ps622.OverlayValues[69] = d69
					ps622.OverlayValues[70] = d70
					ps622.OverlayValues[71] = d71
					ps622.OverlayValues[72] = d72
					ps622.OverlayValues[73] = d73
					ps622.OverlayValues[74] = d74
					ps622.OverlayValues[106] = d106
					ps622.OverlayValues[139] = d139
					ps622.OverlayValues[140] = d140
					ps622.OverlayValues[141] = d141
					ps622.OverlayValues[142] = d142
					ps622.OverlayValues[143] = d143
					ps622.OverlayValues[144] = d144
					ps622.OverlayValues[145] = d145
					ps622.OverlayValues[146] = d146
					ps622.OverlayValues[147] = d147
					ps622.OverlayValues[148] = d148
					ps622.OverlayValues[149] = d149
					ps622.OverlayValues[150] = d150
					ps622.OverlayValues[151] = d151
					ps622.OverlayValues[152] = d152
					ps622.OverlayValues[153] = d153
					ps622.OverlayValues[154] = d154
					ps622.OverlayValues[155] = d155
					ps622.OverlayValues[156] = d156
					ps622.OverlayValues[157] = d157
					ps622.OverlayValues[158] = d158
					ps622.OverlayValues[211] = d211
					ps622.OverlayValues[212] = d212
					ps622.OverlayValues[213] = d213
					ps622.OverlayValues[214] = d214
					ps622.OverlayValues[215] = d215
					ps622.OverlayValues[216] = d216
					ps622.OverlayValues[217] = d217
					ps622.OverlayValues[218] = d218
					ps622.OverlayValues[219] = d219
					ps622.OverlayValues[220] = d220
					ps622.OverlayValues[221] = d221
					ps622.OverlayValues[222] = d222
					ps622.OverlayValues[287] = d287
					ps622.OverlayValues[288] = d288
					ps622.OverlayValues[289] = d289
					ps622.OverlayValues[290] = d290
					ps622.OverlayValues[291] = d291
					ps622.OverlayValues[292] = d292
					ps622.OverlayValues[293] = d293
					ps622.OverlayValues[294] = d294
					ps622.OverlayValues[295] = d295
					ps622.OverlayValues[296] = d296
					ps622.OverlayValues[297] = d297
					ps622.OverlayValues[298] = d298
					ps622.OverlayValues[299] = d299
					ps622.OverlayValues[377] = d377
					ps622.OverlayValues[378] = d378
					ps622.OverlayValues[379] = d379
					ps622.OverlayValues[380] = d380
					ps622.OverlayValues[381] = d381
					ps622.OverlayValues[382] = d382
					ps622.OverlayValues[383] = d383
					ps622.OverlayValues[384] = d384
					ps622.OverlayValues[385] = d385
					ps622.OverlayValues[386] = d386
					ps622.OverlayValues[387] = d387
					ps622.OverlayValues[388] = d388
					ps622.OverlayValues[389] = d389
					ps622.OverlayValues[481] = d481
					ps622.OverlayValues[482] = d482
					ps622.OverlayValues[483] = d483
					ps622.OverlayValues[485] = d485
					ps622.OverlayValues[486] = d486
					ps622.OverlayValues[487] = d487
					ps622.OverlayValues[488] = d488
					ps622.OverlayValues[489] = d489
					ps622.OverlayValues[490] = d490
					ps622.OverlayValues[491] = d491
					ps622.OverlayValues[492] = d492
					ps622.OverlayValues[493] = d493
					ps622.OverlayValues[494] = d494
					ps622.OverlayValues[495] = d495
					ps622.OverlayValues[496] = d496
					ps622.OverlayValues[497] = d497
					ps622.OverlayValues[605] = d605
					ps622.OverlayValues[606] = d606
					ps622.OverlayValues[607] = d607
					ps622.OverlayValues[609] = d609
					ps622.OverlayValues[610] = d610
					ps622.OverlayValues[611] = d611
					ps622.OverlayValues[612] = d612
					ps622.OverlayValues[613] = d613
					ps622.OverlayValues[614] = d614
					ps622.OverlayValues[615] = d615
					ps622.OverlayValues[616] = d616
					ps622.OverlayValues[617] = d617
					ps622.OverlayValues[618] = d618
					snap623 := d0
					snap624 := d1
					snap625 := d2
					snap626 := d3
					snap627 := d13
					snap628 := d14
					snap629 := d16
					snap630 := d17
					snap631 := d18
					snap632 := d20
					snap633 := d21
					snap634 := d22
					snap635 := d40
					snap636 := d41
					snap637 := d42
					snap638 := d43
					snap639 := d65
					snap640 := d66
					snap641 := d67
					snap642 := d68
					snap643 := d69
					snap644 := d70
					snap645 := d71
					snap646 := d72
					snap647 := d73
					snap648 := d74
					snap649 := d106
					snap650 := d139
					snap651 := d140
					snap652 := d141
					snap653 := d142
					snap654 := d143
					snap655 := d144
					snap656 := d145
					snap657 := d146
					snap658 := d147
					snap659 := d148
					snap660 := d149
					snap661 := d150
					snap662 := d151
					snap663 := d152
					snap664 := d153
					snap665 := d154
					snap666 := d155
					snap667 := d156
					snap668 := d157
					snap669 := d158
					snap670 := d211
					snap671 := d212
					snap672 := d213
					snap673 := d214
					snap674 := d215
					snap675 := d216
					snap676 := d217
					snap677 := d218
					snap678 := d219
					snap679 := d220
					snap680 := d221
					snap681 := d222
					snap682 := d287
					snap683 := d288
					snap684 := d289
					snap685 := d290
					snap686 := d291
					snap687 := d292
					snap688 := d293
					snap689 := d294
					snap690 := d295
					snap691 := d296
					snap692 := d297
					snap693 := d298
					snap694 := d299
					snap695 := d377
					snap696 := d378
					snap697 := d379
					snap698 := d380
					snap699 := d381
					snap700 := d382
					snap701 := d383
					snap702 := d384
					snap703 := d385
					snap704 := d386
					snap705 := d387
					snap706 := d388
					snap707 := d389
					snap708 := d481
					snap709 := d482
					snap710 := d483
					snap711 := d485
					snap712 := d486
					snap713 := d487
					snap714 := d488
					snap715 := d489
					snap716 := d490
					snap717 := d491
					snap718 := d492
					snap719 := d493
					snap720 := d494
					snap721 := d495
					snap722 := d496
					snap723 := d497
					snap724 := d605
					snap725 := d606
					snap726 := d607
					snap727 := d609
					snap728 := d610
					snap729 := d611
					snap730 := d612
					snap731 := d613
					snap732 := d614
					snap733 := d615
					snap734 := d616
					snap735 := d617
					snap736 := d618
					alloc737 := ctx.SnapshotAllocState()
					if !bbs[20].Rendered {
						bbs[20].RenderPS(ps622)
					}
					ctx.RestoreAllocState(alloc737)
					d0 = snap623
					d1 = snap624
					d2 = snap625
					d3 = snap626
					d13 = snap627
					d14 = snap628
					d16 = snap629
					d17 = snap630
					d18 = snap631
					d20 = snap632
					d21 = snap633
					d22 = snap634
					d40 = snap635
					d41 = snap636
					d42 = snap637
					d43 = snap638
					d65 = snap639
					d66 = snap640
					d67 = snap641
					d68 = snap642
					d69 = snap643
					d70 = snap644
					d71 = snap645
					d72 = snap646
					d73 = snap647
					d74 = snap648
					d106 = snap649
					d139 = snap650
					d140 = snap651
					d141 = snap652
					d142 = snap653
					d143 = snap654
					d144 = snap655
					d145 = snap656
					d146 = snap657
					d147 = snap658
					d148 = snap659
					d149 = snap660
					d150 = snap661
					d151 = snap662
					d152 = snap663
					d153 = snap664
					d154 = snap665
					d155 = snap666
					d156 = snap667
					d157 = snap668
					d158 = snap669
					d211 = snap670
					d212 = snap671
					d213 = snap672
					d214 = snap673
					d215 = snap674
					d216 = snap675
					d217 = snap676
					d218 = snap677
					d219 = snap678
					d220 = snap679
					d221 = snap680
					d222 = snap681
					d287 = snap682
					d288 = snap683
					d289 = snap684
					d290 = snap685
					d291 = snap686
					d292 = snap687
					d293 = snap688
					d294 = snap689
					d295 = snap690
					d296 = snap691
					d297 = snap692
					d298 = snap693
					d299 = snap694
					d377 = snap695
					d378 = snap696
					d379 = snap697
					d380 = snap698
					d381 = snap699
					d382 = snap700
					d383 = snap701
					d384 = snap702
					d385 = snap703
					d386 = snap704
					d387 = snap705
					d388 = snap706
					d389 = snap707
					d481 = snap708
					d482 = snap709
					d483 = snap710
					d485 = snap711
					d486 = snap712
					d487 = snap713
					d488 = snap714
					d489 = snap715
					d490 = snap716
					d491 = snap717
					d492 = snap718
					d493 = snap719
					d494 = snap720
					d495 = snap721
					d496 = snap722
					d497 = snap723
					d605 = snap724
					d606 = snap725
					d607 = snap726
					d609 = snap727
					d610 = snap728
					d611 = snap729
					d612 = snap730
					d613 = snap731
					d614 = snap732
					d615 = snap733
					d616 = snap734
					d617 = snap735
					d618 = snap736
					if !bbs[18].Rendered {
						return bbs[18].RenderPS(ps621)
					}
					return result
					ctx.FreeDesc(&d617)
					return result
				}
				bbs[20].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[20].VisitCount >= 0 {
							ps.General = true
							return bbs[20].RenderPS(ps)
						}
					}
					bbs[20].VisitCount++
					if ps.General {
						if bbs[20].Rendered {
							ctx.EmitJmp(lbl21)
							return result
						}
						bbs[20].Rendered = true
						bbs[20].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_20 = bbs[20].Address
						ctx.MarkLabel(lbl21)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					if len(ps.OverlayValues) > 219 && ps.OverlayValues[219].Loc != LocNone {
						d219 = ps.OverlayValues[219]
					}
					if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != LocNone {
						d220 = ps.OverlayValues[220]
					}
					if len(ps.OverlayValues) > 221 && ps.OverlayValues[221].Loc != LocNone {
						d221 = ps.OverlayValues[221]
					}
					if len(ps.OverlayValues) > 222 && ps.OverlayValues[222].Loc != LocNone {
						d222 = ps.OverlayValues[222]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != LocNone {
						d380 = ps.OverlayValues[380]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != LocNone {
						d382 = ps.OverlayValues[382]
					}
					if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != LocNone {
						d383 = ps.OverlayValues[383]
					}
					if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != LocNone {
						d384 = ps.OverlayValues[384]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 481 && ps.OverlayValues[481].Loc != LocNone {
						d481 = ps.OverlayValues[481]
					}
					if len(ps.OverlayValues) > 482 && ps.OverlayValues[482].Loc != LocNone {
						d482 = ps.OverlayValues[482]
					}
					if len(ps.OverlayValues) > 483 && ps.OverlayValues[483].Loc != LocNone {
						d483 = ps.OverlayValues[483]
					}
					if len(ps.OverlayValues) > 485 && ps.OverlayValues[485].Loc != LocNone {
						d485 = ps.OverlayValues[485]
					}
					if len(ps.OverlayValues) > 486 && ps.OverlayValues[486].Loc != LocNone {
						d486 = ps.OverlayValues[486]
					}
					if len(ps.OverlayValues) > 487 && ps.OverlayValues[487].Loc != LocNone {
						d487 = ps.OverlayValues[487]
					}
					if len(ps.OverlayValues) > 488 && ps.OverlayValues[488].Loc != LocNone {
						d488 = ps.OverlayValues[488]
					}
					if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != LocNone {
						d489 = ps.OverlayValues[489]
					}
					if len(ps.OverlayValues) > 490 && ps.OverlayValues[490].Loc != LocNone {
						d490 = ps.OverlayValues[490]
					}
					if len(ps.OverlayValues) > 491 && ps.OverlayValues[491].Loc != LocNone {
						d491 = ps.OverlayValues[491]
					}
					if len(ps.OverlayValues) > 492 && ps.OverlayValues[492].Loc != LocNone {
						d492 = ps.OverlayValues[492]
					}
					if len(ps.OverlayValues) > 493 && ps.OverlayValues[493].Loc != LocNone {
						d493 = ps.OverlayValues[493]
					}
					if len(ps.OverlayValues) > 494 && ps.OverlayValues[494].Loc != LocNone {
						d494 = ps.OverlayValues[494]
					}
					if len(ps.OverlayValues) > 495 && ps.OverlayValues[495].Loc != LocNone {
						d495 = ps.OverlayValues[495]
					}
					if len(ps.OverlayValues) > 496 && ps.OverlayValues[496].Loc != LocNone {
						d496 = ps.OverlayValues[496]
					}
					if len(ps.OverlayValues) > 497 && ps.OverlayValues[497].Loc != LocNone {
						d497 = ps.OverlayValues[497]
					}
					if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != LocNone {
						d605 = ps.OverlayValues[605]
					}
					if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != LocNone {
						d606 = ps.OverlayValues[606]
					}
					if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != LocNone {
						d607 = ps.OverlayValues[607]
					}
					if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != LocNone {
						d609 = ps.OverlayValues[609]
					}
					if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != LocNone {
						d610 = ps.OverlayValues[610]
					}
					if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != LocNone {
						d611 = ps.OverlayValues[611]
					}
					if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != LocNone {
						d612 = ps.OverlayValues[612]
					}
					if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != LocNone {
						d613 = ps.OverlayValues[613]
					}
					if len(ps.OverlayValues) > 614 && ps.OverlayValues[614].Loc != LocNone {
						d614 = ps.OverlayValues[614]
					}
					if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != LocNone {
						d615 = ps.OverlayValues[615]
					}
					if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != LocNone {
						d616 = ps.OverlayValues[616]
					}
					if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != LocNone {
						d617 = ps.OverlayValues[617]
					}
					if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != LocNone {
						d618 = ps.OverlayValues[618]
					}
					ctx.ReclaimUntrackedRegs()
					d738 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d738)
					if d738.Loc == LocRegPair || d738.Loc == LocStackPair || d738.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d738, &result)
						result.Type = d738.Type
					} else {
						switch d738.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d738)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d738)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d738)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d738, &result)
							result.Type = d738.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps739 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps739)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  130,
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
