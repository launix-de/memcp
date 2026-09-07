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
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
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
					ctx.EmitJump(d2.Condition, lbl2)
					snap5 := d0
					snap6 := d1
					snap7 := d2
					alloc8 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc8)
					d0 = snap5
					d1 = snap6
					d2 = snap7
					ctx.RestoreAllocState(alloc8)
					d0 = snap5
					d1 = snap6
					d2 = snap7
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 3)
					ps9.OverlayValues[0] = d0
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 3)
					ps10.OverlayValues[0] = d0
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					snap11 := d0
					snap12 := d1
					snap13 := d2
					alloc14 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc14)
					d0 = snap11
					d1 = snap12
					d2 = snap13
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps9)
					}
					return result
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
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(time.Now), []JITValueDesc{}, 3)
					d15.NoHeapPointer = false
					ctx.BindReg(d15.Reg, &d15)
					ctx.BindReg(d15.Reg2, &d15)
					ctx.BindReg(d15.Reg3, &d15)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d15}, 1)
					d16.NoHeapPointer = true
					ctx.BindReg(d16.Reg, &d16)
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						ctx.EmitMakeInt(result, d16)
					} else {
						ctx.EmitMovToReg(result.Reg2, d16)
						d17 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d17)
						if d16.Loc == LocReg && d16.Reg != result.Reg2 {
							ctx.FreeReg(d16.Reg)
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
					d18 = args[0]
					d18.ID = 0
					d20 = d18
					d20.ID = 0
					d19 = ctx.EmitTagEqualsBorrowed(&d20, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d18)
					d21 = d19
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							if ps.General {
							}
							ps22 := PhiState{General: ps.General}
							ps22.OverlayValues = make([]JITValueDesc, 22)
							ps22.OverlayValues[0] = d0
							ps22.OverlayValues[1] = d1
							ps22.OverlayValues[2] = d2
							ps22.OverlayValues[15] = d15
							ps22.OverlayValues[16] = d16
							ps22.OverlayValues[17] = d17
							ps22.OverlayValues[18] = d18
							ps22.OverlayValues[19] = d19
							ps22.OverlayValues[20] = d20
							ps22.OverlayValues[21] = d21
							return bbs[3].RenderPS(ps22)
						}
						if ps.General {
						}
						ps23 := PhiState{General: ps.General}
						ps23.OverlayValues = make([]JITValueDesc, 22)
						ps23.OverlayValues[0] = d0
						ps23.OverlayValues[1] = d1
						ps23.OverlayValues[2] = d2
						ps23.OverlayValues[15] = d15
						ps23.OverlayValues[16] = d16
						ps23.OverlayValues[17] = d17
						ps23.OverlayValues[18] = d18
						ps23.OverlayValues[19] = d19
						ps23.OverlayValues[20] = d20
						ps23.OverlayValues[21] = d21
						return bbs[4].RenderPS(ps23)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					snap24 := d0
					snap25 := d1
					snap26 := d2
					snap27 := d15
					snap28 := d16
					snap29 := d17
					snap30 := d18
					snap31 := d19
					snap32 := d20
					snap33 := d21
					alloc34 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc34)
					d0 = snap24
					d1 = snap25
					d2 = snap26
					d15 = snap27
					d16 = snap28
					d17 = snap29
					d18 = snap30
					d19 = snap31
					d20 = snap32
					d21 = snap33
					ctx.RestoreAllocState(alloc34)
					d0 = snap24
					d1 = snap25
					d2 = snap26
					d15 = snap27
					d16 = snap28
					d17 = snap29
					d18 = snap30
					d19 = snap31
					d20 = snap32
					d21 = snap33
					ps35 := PhiState{General: true}
					ps35.OverlayValues = make([]JITValueDesc, 22)
					ps35.OverlayValues[0] = d0
					ps35.OverlayValues[1] = d1
					ps35.OverlayValues[2] = d2
					ps35.OverlayValues[15] = d15
					ps35.OverlayValues[16] = d16
					ps35.OverlayValues[17] = d17
					ps35.OverlayValues[18] = d18
					ps35.OverlayValues[19] = d19
					ps35.OverlayValues[20] = d20
					ps35.OverlayValues[21] = d21
					ps36 := PhiState{General: true}
					ps36.OverlayValues = make([]JITValueDesc, 22)
					ps36.OverlayValues[0] = d0
					ps36.OverlayValues[1] = d1
					ps36.OverlayValues[2] = d2
					ps36.OverlayValues[15] = d15
					ps36.OverlayValues[16] = d16
					ps36.OverlayValues[17] = d17
					ps36.OverlayValues[18] = d18
					ps36.OverlayValues[19] = d19
					ps36.OverlayValues[20] = d20
					ps36.OverlayValues[21] = d21
					snap37 := d0
					snap38 := d1
					snap39 := d2
					snap40 := d15
					snap41 := d16
					snap42 := d17
					snap43 := d18
					snap44 := d19
					snap45 := d20
					snap46 := d21
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps36)
					}
					ctx.RestoreAllocState(alloc47)
					d0 = snap37
					d1 = snap38
					d2 = snap39
					d15 = snap40
					d16 = snap41
					d17 = snap42
					d18 = snap43
					d19 = snap44
					d20 = snap45
					d21 = snap46
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps35)
					}
					return result
					ctx.FreeDesc(&d19)
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					ctx.ReclaimUntrackedRegs()
					d48 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d48)
					if d48.Loc == LocRegPair || d48.Loc == LocStackPair || d48.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d48, &result)
						result.Type = d48.Type
					} else {
						switch d48.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d48)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d48)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d48)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d48, &result)
							result.Type = d48.Type
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					ctx.ReclaimUntrackedRegs()
					d49 = args[0]
					d49.ID = 0
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d49)
					d49 = JITPrepareScmerGoArg(ctx, d49)
					ctx.SyncDesc(&d49)
					callResults50 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d49}, []uint8{3, 1}, []uint8{4, 0})
					d51 = callResults50[0]
					_ = d51
					d52 = callResults50[1]
					_ = d52
					ctx.FreeDesc(&d49)
					ctx.StabilizeDescForControlFlow(&d51)
					d53 = d52
					ctx.EnsureDesc(&d53)
					if d53.Loc != LocImm && d53.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d53.Loc == LocImm {
						if d53.Imm.Bool() {
							if ps.General {
							}
							ps54 := PhiState{General: ps.General}
							ps54.OverlayValues = make([]JITValueDesc, 54)
							ps54.OverlayValues[0] = d0
							ps54.OverlayValues[1] = d1
							ps54.OverlayValues[2] = d2
							ps54.OverlayValues[15] = d15
							ps54.OverlayValues[16] = d16
							ps54.OverlayValues[17] = d17
							ps54.OverlayValues[18] = d18
							ps54.OverlayValues[19] = d19
							ps54.OverlayValues[20] = d20
							ps54.OverlayValues[21] = d21
							ps54.OverlayValues[48] = d48
							ps54.OverlayValues[49] = d49
							ps54.OverlayValues[51] = d51
							ps54.OverlayValues[52] = d52
							ps54.OverlayValues[53] = d53
							return bbs[6].RenderPS(ps54)
						}
						if ps.General {
						}
						ps55 := PhiState{General: ps.General}
						ps55.OverlayValues = make([]JITValueDesc, 54)
						ps55.OverlayValues[0] = d0
						ps55.OverlayValues[1] = d1
						ps55.OverlayValues[2] = d2
						ps55.OverlayValues[15] = d15
						ps55.OverlayValues[16] = d16
						ps55.OverlayValues[17] = d17
						ps55.OverlayValues[18] = d18
						ps55.OverlayValues[19] = d19
						ps55.OverlayValues[20] = d20
						ps55.OverlayValues[21] = d21
						ps55.OverlayValues[48] = d48
						ps55.OverlayValues[49] = d49
						ps55.OverlayValues[51] = d51
						ps55.OverlayValues[52] = d52
						ps55.OverlayValues[53] = d53
						return bbs[5].RenderPS(ps55)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d53.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					snap56 := d0
					snap57 := d1
					snap58 := d2
					snap59 := d15
					snap60 := d16
					snap61 := d17
					snap62 := d18
					snap63 := d19
					snap64 := d20
					snap65 := d21
					snap66 := d48
					snap67 := d49
					snap68 := d51
					snap69 := d52
					snap70 := d53
					alloc71 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc71)
					d0 = snap56
					d1 = snap57
					d2 = snap58
					d15 = snap59
					d16 = snap60
					d17 = snap61
					d18 = snap62
					d19 = snap63
					d20 = snap64
					d21 = snap65
					d48 = snap66
					d49 = snap67
					d51 = snap68
					d52 = snap69
					d53 = snap70
					ctx.RestoreAllocState(alloc71)
					d0 = snap56
					d1 = snap57
					d2 = snap58
					d15 = snap59
					d16 = snap60
					d17 = snap61
					d18 = snap62
					d19 = snap63
					d20 = snap64
					d21 = snap65
					d48 = snap66
					d49 = snap67
					d51 = snap68
					d52 = snap69
					d53 = snap70
					ps72 := PhiState{General: true}
					ps72.OverlayValues = make([]JITValueDesc, 54)
					ps72.OverlayValues[0] = d0
					ps72.OverlayValues[1] = d1
					ps72.OverlayValues[2] = d2
					ps72.OverlayValues[15] = d15
					ps72.OverlayValues[16] = d16
					ps72.OverlayValues[17] = d17
					ps72.OverlayValues[18] = d18
					ps72.OverlayValues[19] = d19
					ps72.OverlayValues[20] = d20
					ps72.OverlayValues[21] = d21
					ps72.OverlayValues[48] = d48
					ps72.OverlayValues[49] = d49
					ps72.OverlayValues[51] = d51
					ps72.OverlayValues[52] = d52
					ps72.OverlayValues[53] = d53
					ps73 := PhiState{General: true}
					ps73.OverlayValues = make([]JITValueDesc, 54)
					ps73.OverlayValues[0] = d0
					ps73.OverlayValues[1] = d1
					ps73.OverlayValues[2] = d2
					ps73.OverlayValues[15] = d15
					ps73.OverlayValues[16] = d16
					ps73.OverlayValues[17] = d17
					ps73.OverlayValues[18] = d18
					ps73.OverlayValues[19] = d19
					ps73.OverlayValues[20] = d20
					ps73.OverlayValues[21] = d21
					ps73.OverlayValues[48] = d48
					ps73.OverlayValues[49] = d49
					ps73.OverlayValues[51] = d51
					ps73.OverlayValues[52] = d52
					ps73.OverlayValues[53] = d53
					snap74 := d0
					snap75 := d1
					snap76 := d2
					snap77 := d15
					snap78 := d16
					snap79 := d17
					snap80 := d18
					snap81 := d19
					snap82 := d20
					snap83 := d21
					snap84 := d48
					snap85 := d49
					snap86 := d51
					snap87 := d52
					snap88 := d53
					alloc89 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps73)
					}
					ctx.RestoreAllocState(alloc89)
					d0 = snap74
					d1 = snap75
					d2 = snap76
					d15 = snap77
					d16 = snap78
					d17 = snap79
					d18 = snap80
					d19 = snap81
					d20 = snap82
					d21 = snap83
					d48 = snap84
					d49 = snap85
					d51 = snap86
					d52 = snap87
					d53 = snap88
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps72)
					}
					return result
					ctx.FreeDesc(&d52)
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					ctx.ReclaimUntrackedRegs()
					d90 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d90)
					if d90.Loc == LocRegPair || d90.Loc == LocStackPair || d90.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d90, &result)
						result.Type = d90.Type
					} else {
						switch d90.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d90)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d90)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d90)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d90, &result)
							result.Type = d90.Type
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					if d51.Loc != LocRegTriple && d51.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d51)
					d91 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d51}, 1)
					d91.NoHeapPointer = true
					ctx.BindReg(d91.Reg, &d91)
					ctx.EnsureDesc(&d91)
					if d91.Loc == LocImm {
						ctx.EmitMakeInt(result, d91)
					} else {
						ctx.EmitMovToReg(result.Reg2, d91)
						d92 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d92)
						if d91.Loc == LocReg && d91.Reg != result.Reg2 {
							ctx.FreeReg(d91.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				ps93 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps93)
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
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d159 JITValueDesc
				_ = d159
				var d160 JITValueDesc
				_ = d160
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d232 JITValueDesc
				_ = d232
				var d233 JITValueDesc
				_ = d233
				var d234 JITValueDesc
				_ = d234
				var d235 JITValueDesc
				_ = d235
				var d236 JITValueDesc
				_ = d236
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
				var d336 JITValueDesc
				_ = d336
				var d337 JITValueDesc
				_ = d337
				var d339 JITValueDesc
				_ = d339
				var d340 JITValueDesc
				_ = d340
				var d341 JITValueDesc
				_ = d341
				var d342 JITValueDesc
				_ = d342
				var d344 JITValueDesc
				_ = d344
				var d345 JITValueDesc
				_ = d345
				var d346 JITValueDesc
				_ = d346
				var d489 JITValueDesc
				_ = d489
				var d490 JITValueDesc
				_ = d490
				var d491 JITValueDesc
				_ = d491
				var d492 JITValueDesc
				_ = d492
				var d494 JITValueDesc
				_ = d494
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
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap8 := d1
					snap9 := d2
					snap10 := d3
					snap11 := d4
					snap12 := d5
					alloc13 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc13)
					d1 = snap8
					d2 = snap9
					d3 = snap10
					d4 = snap11
					d5 = snap12
					ctx.RestoreAllocState(alloc13)
					d1 = snap8
					d2 = snap9
					d3 = snap10
					d4 = snap11
					d5 = snap12
					ps14 := PhiState{General: true}
					ps14.OverlayValues = make([]JITValueDesc, 6)
					ps14.OverlayValues[1] = d1
					ps14.OverlayValues[2] = d2
					ps14.OverlayValues[3] = d3
					ps14.OverlayValues[4] = d4
					ps14.OverlayValues[5] = d5
					ps15 := PhiState{General: true}
					ps15.OverlayValues = make([]JITValueDesc, 6)
					ps15.OverlayValues[1] = d1
					ps15.OverlayValues[2] = d2
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					snap16 := d1
					snap17 := d2
					snap18 := d3
					snap19 := d4
					snap20 := d5
					alloc21 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps15)
					}
					ctx.RestoreAllocState(alloc21)
					d1 = snap16
					d2 = snap17
					d3 = snap18
					d4 = snap19
					d5 = snap20
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps14)
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
					d22 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d22)
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d22, &result)
						result.Type = d22.Type
					} else {
						switch d22.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d22)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d22)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d22)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d22, &result)
							result.Type = d22.Type
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					ctx.ReclaimUntrackedRegs()
					d23 = args[1]
					d23.ID = 0
					d25 = d23
					ctx.SyncDesc(&d25)
					if d25.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d25.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d25.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d25 = tmpScalar
					}
					d25 = JITPrepareScmerGoArg(ctx, d25)
					if d25.Loc != LocRegPair && d25.Loc != LocStackPair && d25.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d25}, 2)
					ctx.FreeDesc(&d23)
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d24.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d24.Imm)
						ptrWord, _ := d24.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d24.Imm.String())))
						d24 = tmpPair
					} else if d24.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d24.Type, Reg: ctx.AllocRegExcept(d24.Reg), Reg2: ctx.AllocRegExcept(d24.Reg)}
						switch d24.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d24)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d24)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d24)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d24)
						d24 = tmpPair
					}
					if d24.Loc != LocRegPair && d24.Loc != LocStackPair && d24.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d24)
					callResults26 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d24}, []uint8{1, 2}, []uint8{1, 3})
					d27 = callResults26[0]
					_ = d27
					d28 = callResults26[1]
					_ = d28
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.EnsureDesc(&d28)
					var d29 JITValueDesc
					if d28.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d28.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d28)
						if d28.Loc != LocReg && d28.Loc != LocRegPair && d28.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d28.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d29)
					}
					ctx.FreeDesc(&d28)
					d30 = d29
					ctx.EnsureDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							if ps.General {
							}
							ps31 := PhiState{General: ps.General}
							ps31.OverlayValues = make([]JITValueDesc, 31)
							ps31.OverlayValues[1] = d1
							ps31.OverlayValues[2] = d2
							ps31.OverlayValues[3] = d3
							ps31.OverlayValues[4] = d4
							ps31.OverlayValues[5] = d5
							ps31.OverlayValues[22] = d22
							ps31.OverlayValues[23] = d23
							ps31.OverlayValues[24] = d24
							ps31.OverlayValues[25] = d25
							ps31.OverlayValues[27] = d27
							ps31.OverlayValues[28] = d28
							ps31.OverlayValues[29] = d29
							ps31.OverlayValues[30] = d30
							return bbs[5].RenderPS(ps31)
						}
						if ps.General {
						}
						ps32 := PhiState{General: ps.General}
						ps32.OverlayValues = make([]JITValueDesc, 31)
						ps32.OverlayValues[1] = d1
						ps32.OverlayValues[2] = d2
						ps32.OverlayValues[3] = d3
						ps32.OverlayValues[4] = d4
						ps32.OverlayValues[5] = d5
						ps32.OverlayValues[22] = d22
						ps32.OverlayValues[23] = d23
						ps32.OverlayValues[24] = d24
						ps32.OverlayValues[25] = d25
						ps32.OverlayValues[27] = d27
						ps32.OverlayValues[28] = d28
						ps32.OverlayValues[29] = d29
						ps32.OverlayValues[30] = d30
						return bbs[6].RenderPS(ps32)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					snap33 := d1
					snap34 := d2
					snap35 := d3
					snap36 := d4
					snap37 := d5
					snap38 := d22
					snap39 := d23
					snap40 := d24
					snap41 := d25
					snap42 := d27
					snap43 := d28
					snap44 := d29
					snap45 := d30
					alloc46 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc46)
					d1 = snap33
					d2 = snap34
					d3 = snap35
					d4 = snap36
					d5 = snap37
					d22 = snap38
					d23 = snap39
					d24 = snap40
					d25 = snap41
					d27 = snap42
					d28 = snap43
					d29 = snap44
					d30 = snap45
					ctx.RestoreAllocState(alloc46)
					d1 = snap33
					d2 = snap34
					d3 = snap35
					d4 = snap36
					d5 = snap37
					d22 = snap38
					d23 = snap39
					d24 = snap40
					d25 = snap41
					d27 = snap42
					d28 = snap43
					d29 = snap44
					d30 = snap45
					ps47 := PhiState{General: true}
					ps47.OverlayValues = make([]JITValueDesc, 31)
					ps47.OverlayValues[1] = d1
					ps47.OverlayValues[2] = d2
					ps47.OverlayValues[3] = d3
					ps47.OverlayValues[4] = d4
					ps47.OverlayValues[5] = d5
					ps47.OverlayValues[22] = d22
					ps47.OverlayValues[23] = d23
					ps47.OverlayValues[24] = d24
					ps47.OverlayValues[25] = d25
					ps47.OverlayValues[27] = d27
					ps47.OverlayValues[28] = d28
					ps47.OverlayValues[29] = d29
					ps47.OverlayValues[30] = d30
					ps48 := PhiState{General: true}
					ps48.OverlayValues = make([]JITValueDesc, 31)
					ps48.OverlayValues[1] = d1
					ps48.OverlayValues[2] = d2
					ps48.OverlayValues[3] = d3
					ps48.OverlayValues[4] = d4
					ps48.OverlayValues[5] = d5
					ps48.OverlayValues[22] = d22
					ps48.OverlayValues[23] = d23
					ps48.OverlayValues[24] = d24
					ps48.OverlayValues[25] = d25
					ps48.OverlayValues[27] = d27
					ps48.OverlayValues[28] = d28
					ps48.OverlayValues[29] = d29
					ps48.OverlayValues[30] = d30
					snap49 := d1
					snap50 := d2
					snap51 := d3
					snap52 := d4
					snap53 := d5
					snap54 := d22
					snap55 := d23
					snap56 := d24
					snap57 := d25
					snap58 := d27
					snap59 := d28
					snap60 := d29
					snap61 := d30
					alloc62 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps48)
					}
					ctx.RestoreAllocState(alloc62)
					d1 = snap49
					d2 = snap50
					d3 = snap51
					d4 = snap52
					d5 = snap53
					d22 = snap54
					d23 = snap55
					d24 = snap56
					d25 = snap57
					d27 = snap58
					d28 = snap59
					d29 = snap60
					d30 = snap61
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps47)
					}
					return result
					ctx.FreeDesc(&d29)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					ctx.ReclaimUntrackedRegs()
					d63 = args[2]
					d63.ID = 0
					d65 = d63
					d65.ID = 0
					d64 = ctx.EmitTagEqualsBorrowed(&d65, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d63)
					d66 = d64
					ctx.EnsureDesc(&d66)
					if d66.Loc != LocImm && d66.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d66.Loc == LocImm {
						if d66.Imm.Bool() {
							if ps.General {
							}
							ps67 := PhiState{General: ps.General}
							ps67.OverlayValues = make([]JITValueDesc, 67)
							ps67.OverlayValues[1] = d1
							ps67.OverlayValues[2] = d2
							ps67.OverlayValues[3] = d3
							ps67.OverlayValues[4] = d4
							ps67.OverlayValues[5] = d5
							ps67.OverlayValues[22] = d22
							ps67.OverlayValues[23] = d23
							ps67.OverlayValues[24] = d24
							ps67.OverlayValues[25] = d25
							ps67.OverlayValues[27] = d27
							ps67.OverlayValues[28] = d28
							ps67.OverlayValues[29] = d29
							ps67.OverlayValues[30] = d30
							ps67.OverlayValues[63] = d63
							ps67.OverlayValues[64] = d64
							ps67.OverlayValues[65] = d65
							ps67.OverlayValues[66] = d66
							return bbs[1].RenderPS(ps67)
						}
						if ps.General {
						}
						ps68 := PhiState{General: ps.General}
						ps68.OverlayValues = make([]JITValueDesc, 67)
						ps68.OverlayValues[1] = d1
						ps68.OverlayValues[2] = d2
						ps68.OverlayValues[3] = d3
						ps68.OverlayValues[4] = d4
						ps68.OverlayValues[5] = d5
						ps68.OverlayValues[22] = d22
						ps68.OverlayValues[23] = d23
						ps68.OverlayValues[24] = d24
						ps68.OverlayValues[25] = d25
						ps68.OverlayValues[27] = d27
						ps68.OverlayValues[28] = d28
						ps68.OverlayValues[29] = d29
						ps68.OverlayValues[30] = d30
						ps68.OverlayValues[63] = d63
						ps68.OverlayValues[64] = d64
						ps68.OverlayValues[65] = d65
						ps68.OverlayValues[66] = d66
						return bbs[2].RenderPS(ps68)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d66.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap69 := d1
					snap70 := d2
					snap71 := d3
					snap72 := d4
					snap73 := d5
					snap74 := d22
					snap75 := d23
					snap76 := d24
					snap77 := d25
					snap78 := d27
					snap79 := d28
					snap80 := d29
					snap81 := d30
					snap82 := d63
					snap83 := d64
					snap84 := d65
					snap85 := d66
					alloc86 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc86)
					d1 = snap69
					d2 = snap70
					d3 = snap71
					d4 = snap72
					d5 = snap73
					d22 = snap74
					d23 = snap75
					d24 = snap76
					d25 = snap77
					d27 = snap78
					d28 = snap79
					d29 = snap80
					d30 = snap81
					d63 = snap82
					d64 = snap83
					d65 = snap84
					d66 = snap85
					ctx.RestoreAllocState(alloc86)
					d1 = snap69
					d2 = snap70
					d3 = snap71
					d4 = snap72
					d5 = snap73
					d22 = snap74
					d23 = snap75
					d24 = snap76
					d25 = snap77
					d27 = snap78
					d28 = snap79
					d29 = snap80
					d30 = snap81
					d63 = snap82
					d64 = snap83
					d65 = snap84
					d66 = snap85
					ps87 := PhiState{General: true}
					ps87.OverlayValues = make([]JITValueDesc, 67)
					ps87.OverlayValues[1] = d1
					ps87.OverlayValues[2] = d2
					ps87.OverlayValues[3] = d3
					ps87.OverlayValues[4] = d4
					ps87.OverlayValues[5] = d5
					ps87.OverlayValues[22] = d22
					ps87.OverlayValues[23] = d23
					ps87.OverlayValues[24] = d24
					ps87.OverlayValues[25] = d25
					ps87.OverlayValues[27] = d27
					ps87.OverlayValues[28] = d28
					ps87.OverlayValues[29] = d29
					ps87.OverlayValues[30] = d30
					ps87.OverlayValues[63] = d63
					ps87.OverlayValues[64] = d64
					ps87.OverlayValues[65] = d65
					ps87.OverlayValues[66] = d66
					ps88 := PhiState{General: true}
					ps88.OverlayValues = make([]JITValueDesc, 67)
					ps88.OverlayValues[1] = d1
					ps88.OverlayValues[2] = d2
					ps88.OverlayValues[3] = d3
					ps88.OverlayValues[4] = d4
					ps88.OverlayValues[5] = d5
					ps88.OverlayValues[22] = d22
					ps88.OverlayValues[23] = d23
					ps88.OverlayValues[24] = d24
					ps88.OverlayValues[25] = d25
					ps88.OverlayValues[27] = d27
					ps88.OverlayValues[28] = d28
					ps88.OverlayValues[29] = d29
					ps88.OverlayValues[30] = d30
					ps88.OverlayValues[63] = d63
					ps88.OverlayValues[64] = d64
					ps88.OverlayValues[65] = d65
					ps88.OverlayValues[66] = d66
					snap89 := d1
					snap90 := d2
					snap91 := d3
					snap92 := d4
					snap93 := d5
					snap94 := d22
					snap95 := d23
					snap96 := d24
					snap97 := d25
					snap98 := d27
					snap99 := d28
					snap100 := d29
					snap101 := d30
					snap102 := d63
					snap103 := d64
					snap104 := d65
					snap105 := d66
					alloc106 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps88)
					}
					ctx.RestoreAllocState(alloc106)
					d1 = snap89
					d2 = snap90
					d3 = snap91
					d4 = snap92
					d5 = snap93
					d22 = snap94
					d23 = snap95
					d24 = snap96
					d25 = snap97
					d27 = snap98
					d28 = snap99
					d29 = snap100
					d30 = snap101
					d63 = snap102
					d64 = snap103
					d65 = snap104
					d66 = snap105
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps87)
					}
					return result
					ctx.FreeDesc(&d64)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					d107 = args[1]
					d107.ID = 0
					d109 = d107
					d109.ID = 0
					d108 = ctx.EmitTagEqualsBorrowed(&d109, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d107)
					d110 = d108
					ctx.EnsureDesc(&d110)
					if d110.Loc != LocImm && d110.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d110.Loc == LocImm {
						if d110.Imm.Bool() {
							if ps.General {
							}
							ps111 := PhiState{General: ps.General}
							ps111.OverlayValues = make([]JITValueDesc, 111)
							ps111.OverlayValues[1] = d1
							ps111.OverlayValues[2] = d2
							ps111.OverlayValues[3] = d3
							ps111.OverlayValues[4] = d4
							ps111.OverlayValues[5] = d5
							ps111.OverlayValues[22] = d22
							ps111.OverlayValues[23] = d23
							ps111.OverlayValues[24] = d24
							ps111.OverlayValues[25] = d25
							ps111.OverlayValues[27] = d27
							ps111.OverlayValues[28] = d28
							ps111.OverlayValues[29] = d29
							ps111.OverlayValues[30] = d30
							ps111.OverlayValues[63] = d63
							ps111.OverlayValues[64] = d64
							ps111.OverlayValues[65] = d65
							ps111.OverlayValues[66] = d66
							ps111.OverlayValues[107] = d107
							ps111.OverlayValues[108] = d108
							ps111.OverlayValues[109] = d109
							ps111.OverlayValues[110] = d110
							return bbs[1].RenderPS(ps111)
						}
						if ps.General {
						}
						ps112 := PhiState{General: ps.General}
						ps112.OverlayValues = make([]JITValueDesc, 111)
						ps112.OverlayValues[1] = d1
						ps112.OverlayValues[2] = d2
						ps112.OverlayValues[3] = d3
						ps112.OverlayValues[4] = d4
						ps112.OverlayValues[5] = d5
						ps112.OverlayValues[22] = d22
						ps112.OverlayValues[23] = d23
						ps112.OverlayValues[24] = d24
						ps112.OverlayValues[25] = d25
						ps112.OverlayValues[27] = d27
						ps112.OverlayValues[28] = d28
						ps112.OverlayValues[29] = d29
						ps112.OverlayValues[30] = d30
						ps112.OverlayValues[63] = d63
						ps112.OverlayValues[64] = d64
						ps112.OverlayValues[65] = d65
						ps112.OverlayValues[66] = d66
						ps112.OverlayValues[107] = d107
						ps112.OverlayValues[108] = d108
						ps112.OverlayValues[109] = d109
						ps112.OverlayValues[110] = d110
						return bbs[3].RenderPS(ps112)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d110.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap113 := d1
					snap114 := d2
					snap115 := d3
					snap116 := d4
					snap117 := d5
					snap118 := d22
					snap119 := d23
					snap120 := d24
					snap121 := d25
					snap122 := d27
					snap123 := d28
					snap124 := d29
					snap125 := d30
					snap126 := d63
					snap127 := d64
					snap128 := d65
					snap129 := d66
					snap130 := d107
					snap131 := d108
					snap132 := d109
					snap133 := d110
					alloc134 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc134)
					d1 = snap113
					d2 = snap114
					d3 = snap115
					d4 = snap116
					d5 = snap117
					d22 = snap118
					d23 = snap119
					d24 = snap120
					d25 = snap121
					d27 = snap122
					d28 = snap123
					d29 = snap124
					d30 = snap125
					d63 = snap126
					d64 = snap127
					d65 = snap128
					d66 = snap129
					d107 = snap130
					d108 = snap131
					d109 = snap132
					d110 = snap133
					ctx.RestoreAllocState(alloc134)
					d1 = snap113
					d2 = snap114
					d3 = snap115
					d4 = snap116
					d5 = snap117
					d22 = snap118
					d23 = snap119
					d24 = snap120
					d25 = snap121
					d27 = snap122
					d28 = snap123
					d29 = snap124
					d30 = snap125
					d63 = snap126
					d64 = snap127
					d65 = snap128
					d66 = snap129
					d107 = snap130
					d108 = snap131
					d109 = snap132
					d110 = snap133
					ps135 := PhiState{General: true}
					ps135.OverlayValues = make([]JITValueDesc, 111)
					ps135.OverlayValues[1] = d1
					ps135.OverlayValues[2] = d2
					ps135.OverlayValues[3] = d3
					ps135.OverlayValues[4] = d4
					ps135.OverlayValues[5] = d5
					ps135.OverlayValues[22] = d22
					ps135.OverlayValues[23] = d23
					ps135.OverlayValues[24] = d24
					ps135.OverlayValues[25] = d25
					ps135.OverlayValues[27] = d27
					ps135.OverlayValues[28] = d28
					ps135.OverlayValues[29] = d29
					ps135.OverlayValues[30] = d30
					ps135.OverlayValues[63] = d63
					ps135.OverlayValues[64] = d64
					ps135.OverlayValues[65] = d65
					ps135.OverlayValues[66] = d66
					ps135.OverlayValues[107] = d107
					ps135.OverlayValues[108] = d108
					ps135.OverlayValues[109] = d109
					ps135.OverlayValues[110] = d110
					ps136 := PhiState{General: true}
					ps136.OverlayValues = make([]JITValueDesc, 111)
					ps136.OverlayValues[1] = d1
					ps136.OverlayValues[2] = d2
					ps136.OverlayValues[3] = d3
					ps136.OverlayValues[4] = d4
					ps136.OverlayValues[5] = d5
					ps136.OverlayValues[22] = d22
					ps136.OverlayValues[23] = d23
					ps136.OverlayValues[24] = d24
					ps136.OverlayValues[25] = d25
					ps136.OverlayValues[27] = d27
					ps136.OverlayValues[28] = d28
					ps136.OverlayValues[29] = d29
					ps136.OverlayValues[30] = d30
					ps136.OverlayValues[63] = d63
					ps136.OverlayValues[64] = d64
					ps136.OverlayValues[65] = d65
					ps136.OverlayValues[66] = d66
					ps136.OverlayValues[107] = d107
					ps136.OverlayValues[108] = d108
					ps136.OverlayValues[109] = d109
					ps136.OverlayValues[110] = d110
					snap137 := d1
					snap138 := d2
					snap139 := d3
					snap140 := d4
					snap141 := d5
					snap142 := d22
					snap143 := d23
					snap144 := d24
					snap145 := d25
					snap146 := d27
					snap147 := d28
					snap148 := d29
					snap149 := d30
					snap150 := d63
					snap151 := d64
					snap152 := d65
					snap153 := d66
					snap154 := d107
					snap155 := d108
					snap156 := d109
					snap157 := d110
					alloc158 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps136)
					}
					ctx.RestoreAllocState(alloc158)
					d1 = snap137
					d2 = snap138
					d3 = snap139
					d4 = snap140
					d5 = snap141
					d22 = snap142
					d23 = snap143
					d24 = snap144
					d25 = snap145
					d27 = snap146
					d28 = snap147
					d29 = snap148
					d30 = snap149
					d63 = snap150
					d64 = snap151
					d65 = snap152
					d66 = snap153
					d107 = snap154
					d108 = snap155
					d109 = snap156
					d110 = snap157
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps135)
					}
					return result
					ctx.FreeDesc(&d108)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					ctx.ReclaimUntrackedRegs()
					d159 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d159)
					if d159.Loc == LocRegPair || d159.Loc == LocStackPair || d159.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d159, &result)
						result.Type = d159.Type
					} else {
						switch d159.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d159)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d159)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d159)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d159, &result)
							result.Type = d159.Type
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					ctx.ReclaimUntrackedRegs()
					d160 = args[2]
					d160.ID = 0
					d162 = d160
					ctx.SyncDesc(&d162)
					if d162.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d162.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d162.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d162 = tmpScalar
					}
					d162 = JITPrepareScmerGoArg(ctx, d162)
					if d162.Loc != LocRegPair && d162.Loc != LocStackPair && d162.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d161 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d162}, 2)
					ctx.FreeDesc(&d160)
					ctx.EnsureDesc(&d161)
					ctx.EnsureDesc(&d161)
					ctx.EnsureDesc(&d161)
					if d161.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d161.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d161.Imm)
						ptrWord, _ := d161.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d161.Imm.String())))
						d161 = tmpPair
					} else if d161.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d161.Type, Reg: ctx.AllocRegExcept(d161.Reg), Reg2: ctx.AllocRegExcept(d161.Reg)}
						switch d161.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d161)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d161)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d161)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d161)
						d161 = tmpPair
					}
					if d161.Loc != LocRegPair && d161.Loc != LocStackPair && d161.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d161)
					callResults163 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d161}, []uint8{1, 2}, []uint8{1, 3})
					d164 = callResults163[0]
					_ = d164
					d165 = callResults163[1]
					_ = d165
					ctx.StabilizeDescForControlFlow(&d164)
					ctx.EnsureDesc(&d165)
					var d166 JITValueDesc
					if d165.Loc == LocImm {
						d166 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d165.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d165)
						if d165.Loc != LocReg && d165.Loc != LocRegPair && d165.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d165.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d166 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d166)
					}
					ctx.FreeDesc(&d165)
					d167 = d166
					ctx.EnsureDesc(&d167)
					if d167.Loc != LocImm && d167.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d167.Loc == LocImm {
						if d167.Imm.Bool() {
							if ps.General {
							}
							ps168 := PhiState{General: ps.General}
							ps168.OverlayValues = make([]JITValueDesc, 168)
							ps168.OverlayValues[1] = d1
							ps168.OverlayValues[2] = d2
							ps168.OverlayValues[3] = d3
							ps168.OverlayValues[4] = d4
							ps168.OverlayValues[5] = d5
							ps168.OverlayValues[22] = d22
							ps168.OverlayValues[23] = d23
							ps168.OverlayValues[24] = d24
							ps168.OverlayValues[25] = d25
							ps168.OverlayValues[27] = d27
							ps168.OverlayValues[28] = d28
							ps168.OverlayValues[29] = d29
							ps168.OverlayValues[30] = d30
							ps168.OverlayValues[63] = d63
							ps168.OverlayValues[64] = d64
							ps168.OverlayValues[65] = d65
							ps168.OverlayValues[66] = d66
							ps168.OverlayValues[107] = d107
							ps168.OverlayValues[108] = d108
							ps168.OverlayValues[109] = d109
							ps168.OverlayValues[110] = d110
							ps168.OverlayValues[159] = d159
							ps168.OverlayValues[160] = d160
							ps168.OverlayValues[161] = d161
							ps168.OverlayValues[162] = d162
							ps168.OverlayValues[164] = d164
							ps168.OverlayValues[165] = d165
							ps168.OverlayValues[166] = d166
							ps168.OverlayValues[167] = d167
							return bbs[7].RenderPS(ps168)
						}
						if ps.General {
						}
						ps169 := PhiState{General: ps.General}
						ps169.OverlayValues = make([]JITValueDesc, 168)
						ps169.OverlayValues[1] = d1
						ps169.OverlayValues[2] = d2
						ps169.OverlayValues[3] = d3
						ps169.OverlayValues[4] = d4
						ps169.OverlayValues[5] = d5
						ps169.OverlayValues[22] = d22
						ps169.OverlayValues[23] = d23
						ps169.OverlayValues[24] = d24
						ps169.OverlayValues[25] = d25
						ps169.OverlayValues[27] = d27
						ps169.OverlayValues[28] = d28
						ps169.OverlayValues[29] = d29
						ps169.OverlayValues[30] = d30
						ps169.OverlayValues[63] = d63
						ps169.OverlayValues[64] = d64
						ps169.OverlayValues[65] = d65
						ps169.OverlayValues[66] = d66
						ps169.OverlayValues[107] = d107
						ps169.OverlayValues[108] = d108
						ps169.OverlayValues[109] = d109
						ps169.OverlayValues[110] = d110
						ps169.OverlayValues[159] = d159
						ps169.OverlayValues[160] = d160
						ps169.OverlayValues[161] = d161
						ps169.OverlayValues[162] = d162
						ps169.OverlayValues[164] = d164
						ps169.OverlayValues[165] = d165
						ps169.OverlayValues[166] = d166
						ps169.OverlayValues[167] = d167
						return bbs[8].RenderPS(ps169)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d167.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					snap170 := d1
					snap171 := d2
					snap172 := d3
					snap173 := d4
					snap174 := d5
					snap175 := d22
					snap176 := d23
					snap177 := d24
					snap178 := d25
					snap179 := d27
					snap180 := d28
					snap181 := d29
					snap182 := d30
					snap183 := d63
					snap184 := d64
					snap185 := d65
					snap186 := d66
					snap187 := d107
					snap188 := d108
					snap189 := d109
					snap190 := d110
					snap191 := d159
					snap192 := d160
					snap193 := d161
					snap194 := d162
					snap195 := d164
					snap196 := d165
					snap197 := d166
					snap198 := d167
					alloc199 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc199)
					d1 = snap170
					d2 = snap171
					d3 = snap172
					d4 = snap173
					d5 = snap174
					d22 = snap175
					d23 = snap176
					d24 = snap177
					d25 = snap178
					d27 = snap179
					d28 = snap180
					d29 = snap181
					d30 = snap182
					d63 = snap183
					d64 = snap184
					d65 = snap185
					d66 = snap186
					d107 = snap187
					d108 = snap188
					d109 = snap189
					d110 = snap190
					d159 = snap191
					d160 = snap192
					d161 = snap193
					d162 = snap194
					d164 = snap195
					d165 = snap196
					d166 = snap197
					d167 = snap198
					ctx.RestoreAllocState(alloc199)
					d1 = snap170
					d2 = snap171
					d3 = snap172
					d4 = snap173
					d5 = snap174
					d22 = snap175
					d23 = snap176
					d24 = snap177
					d25 = snap178
					d27 = snap179
					d28 = snap180
					d29 = snap181
					d30 = snap182
					d63 = snap183
					d64 = snap184
					d65 = snap185
					d66 = snap186
					d107 = snap187
					d108 = snap188
					d109 = snap189
					d110 = snap190
					d159 = snap191
					d160 = snap192
					d161 = snap193
					d162 = snap194
					d164 = snap195
					d165 = snap196
					d166 = snap197
					d167 = snap198
					ps200 := PhiState{General: true}
					ps200.OverlayValues = make([]JITValueDesc, 168)
					ps200.OverlayValues[1] = d1
					ps200.OverlayValues[2] = d2
					ps200.OverlayValues[3] = d3
					ps200.OverlayValues[4] = d4
					ps200.OverlayValues[5] = d5
					ps200.OverlayValues[22] = d22
					ps200.OverlayValues[23] = d23
					ps200.OverlayValues[24] = d24
					ps200.OverlayValues[25] = d25
					ps200.OverlayValues[27] = d27
					ps200.OverlayValues[28] = d28
					ps200.OverlayValues[29] = d29
					ps200.OverlayValues[30] = d30
					ps200.OverlayValues[63] = d63
					ps200.OverlayValues[64] = d64
					ps200.OverlayValues[65] = d65
					ps200.OverlayValues[66] = d66
					ps200.OverlayValues[107] = d107
					ps200.OverlayValues[108] = d108
					ps200.OverlayValues[109] = d109
					ps200.OverlayValues[110] = d110
					ps200.OverlayValues[159] = d159
					ps200.OverlayValues[160] = d160
					ps200.OverlayValues[161] = d161
					ps200.OverlayValues[162] = d162
					ps200.OverlayValues[164] = d164
					ps200.OverlayValues[165] = d165
					ps200.OverlayValues[166] = d166
					ps200.OverlayValues[167] = d167
					ps201 := PhiState{General: true}
					ps201.OverlayValues = make([]JITValueDesc, 168)
					ps201.OverlayValues[1] = d1
					ps201.OverlayValues[2] = d2
					ps201.OverlayValues[3] = d3
					ps201.OverlayValues[4] = d4
					ps201.OverlayValues[5] = d5
					ps201.OverlayValues[22] = d22
					ps201.OverlayValues[23] = d23
					ps201.OverlayValues[24] = d24
					ps201.OverlayValues[25] = d25
					ps201.OverlayValues[27] = d27
					ps201.OverlayValues[28] = d28
					ps201.OverlayValues[29] = d29
					ps201.OverlayValues[30] = d30
					ps201.OverlayValues[63] = d63
					ps201.OverlayValues[64] = d64
					ps201.OverlayValues[65] = d65
					ps201.OverlayValues[66] = d66
					ps201.OverlayValues[107] = d107
					ps201.OverlayValues[108] = d108
					ps201.OverlayValues[109] = d109
					ps201.OverlayValues[110] = d110
					ps201.OverlayValues[159] = d159
					ps201.OverlayValues[160] = d160
					ps201.OverlayValues[161] = d161
					ps201.OverlayValues[162] = d162
					ps201.OverlayValues[164] = d164
					ps201.OverlayValues[165] = d165
					ps201.OverlayValues[166] = d166
					ps201.OverlayValues[167] = d167
					snap202 := d1
					snap203 := d2
					snap204 := d3
					snap205 := d4
					snap206 := d5
					snap207 := d22
					snap208 := d23
					snap209 := d24
					snap210 := d25
					snap211 := d27
					snap212 := d28
					snap213 := d29
					snap214 := d30
					snap215 := d63
					snap216 := d64
					snap217 := d65
					snap218 := d66
					snap219 := d107
					snap220 := d108
					snap221 := d109
					snap222 := d110
					snap223 := d159
					snap224 := d160
					snap225 := d161
					snap226 := d162
					snap227 := d164
					snap228 := d165
					snap229 := d166
					snap230 := d167
					alloc231 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps201)
					}
					ctx.RestoreAllocState(alloc231)
					d1 = snap202
					d2 = snap203
					d3 = snap204
					d4 = snap205
					d5 = snap206
					d22 = snap207
					d23 = snap208
					d24 = snap209
					d25 = snap210
					d27 = snap211
					d28 = snap212
					d29 = snap213
					d30 = snap214
					d63 = snap215
					d64 = snap216
					d65 = snap217
					d66 = snap218
					d107 = snap219
					d108 = snap220
					d109 = snap221
					d110 = snap222
					d159 = snap223
					d160 = snap224
					d161 = snap225
					d162 = snap226
					d164 = snap227
					d165 = snap228
					d166 = snap229
					d167 = snap230
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps200)
					}
					return result
					ctx.FreeDesc(&d166)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					ctx.ReclaimUntrackedRegs()
					d232 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d232)
					if d232.Loc == LocRegPair || d232.Loc == LocStackPair || d232.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d232, &result)
						result.Type = d232.Type
					} else {
						switch d232.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d232)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d232)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d232)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d232, &result)
							result.Type = d232.Type
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					ctx.ReclaimUntrackedRegs()
					d233 = args[0]
					d233.ID = 0
					d234 = ctx.EmitGetTagDesc(&d233, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d233)
					ctx.EnsureDesc(&d234)
					var d235 JITValueDesc
					if d234.Loc == LocImm {
						d235 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d234.Imm.Int()) == uint64(0x10))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d234.Reg, 16)
						d235 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d235)
					}
					ctx.FreeDesc(&d234)
					d236 = d235
					ctx.EnsureDesc(&d236)
					if d236.Loc != LocImm && d236.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d236.Loc == LocImm {
						if d236.Imm.Bool() {
							if ps.General {
							}
							ps237 := PhiState{General: ps.General}
							ps237.OverlayValues = make([]JITValueDesc, 237)
							ps237.OverlayValues[1] = d1
							ps237.OverlayValues[2] = d2
							ps237.OverlayValues[3] = d3
							ps237.OverlayValues[4] = d4
							ps237.OverlayValues[5] = d5
							ps237.OverlayValues[22] = d22
							ps237.OverlayValues[23] = d23
							ps237.OverlayValues[24] = d24
							ps237.OverlayValues[25] = d25
							ps237.OverlayValues[27] = d27
							ps237.OverlayValues[28] = d28
							ps237.OverlayValues[29] = d29
							ps237.OverlayValues[30] = d30
							ps237.OverlayValues[63] = d63
							ps237.OverlayValues[64] = d64
							ps237.OverlayValues[65] = d65
							ps237.OverlayValues[66] = d66
							ps237.OverlayValues[107] = d107
							ps237.OverlayValues[108] = d108
							ps237.OverlayValues[109] = d109
							ps237.OverlayValues[110] = d110
							ps237.OverlayValues[159] = d159
							ps237.OverlayValues[160] = d160
							ps237.OverlayValues[161] = d161
							ps237.OverlayValues[162] = d162
							ps237.OverlayValues[164] = d164
							ps237.OverlayValues[165] = d165
							ps237.OverlayValues[166] = d166
							ps237.OverlayValues[167] = d167
							ps237.OverlayValues[232] = d232
							ps237.OverlayValues[233] = d233
							ps237.OverlayValues[234] = d234
							ps237.OverlayValues[235] = d235
							ps237.OverlayValues[236] = d236
							return bbs[10].RenderPS(ps237)
						}
						if ps.General {
						}
						ps238 := PhiState{General: ps.General}
						ps238.OverlayValues = make([]JITValueDesc, 237)
						ps238.OverlayValues[1] = d1
						ps238.OverlayValues[2] = d2
						ps238.OverlayValues[3] = d3
						ps238.OverlayValues[4] = d4
						ps238.OverlayValues[5] = d5
						ps238.OverlayValues[22] = d22
						ps238.OverlayValues[23] = d23
						ps238.OverlayValues[24] = d24
						ps238.OverlayValues[25] = d25
						ps238.OverlayValues[27] = d27
						ps238.OverlayValues[28] = d28
						ps238.OverlayValues[29] = d29
						ps238.OverlayValues[30] = d30
						ps238.OverlayValues[63] = d63
						ps238.OverlayValues[64] = d64
						ps238.OverlayValues[65] = d65
						ps238.OverlayValues[66] = d66
						ps238.OverlayValues[107] = d107
						ps238.OverlayValues[108] = d108
						ps238.OverlayValues[109] = d109
						ps238.OverlayValues[110] = d110
						ps238.OverlayValues[159] = d159
						ps238.OverlayValues[160] = d160
						ps238.OverlayValues[161] = d161
						ps238.OverlayValues[162] = d162
						ps238.OverlayValues[164] = d164
						ps238.OverlayValues[165] = d165
						ps238.OverlayValues[166] = d166
						ps238.OverlayValues[167] = d167
						ps238.OverlayValues[232] = d232
						ps238.OverlayValues[233] = d233
						ps238.OverlayValues[234] = d234
						ps238.OverlayValues[235] = d235
						ps238.OverlayValues[236] = d236
						return bbs[11].RenderPS(ps238)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					ctx.EmitJump(d236.Condition, lbl11)
					snap239 := d1
					snap240 := d2
					snap241 := d3
					snap242 := d4
					snap243 := d5
					snap244 := d22
					snap245 := d23
					snap246 := d24
					snap247 := d25
					snap248 := d27
					snap249 := d28
					snap250 := d29
					snap251 := d30
					snap252 := d63
					snap253 := d64
					snap254 := d65
					snap255 := d66
					snap256 := d107
					snap257 := d108
					snap258 := d109
					snap259 := d110
					snap260 := d159
					snap261 := d160
					snap262 := d161
					snap263 := d162
					snap264 := d164
					snap265 := d165
					snap266 := d166
					snap267 := d167
					snap268 := d232
					snap269 := d233
					snap270 := d234
					snap271 := d235
					snap272 := d236
					alloc273 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc273)
					d1 = snap239
					d2 = snap240
					d3 = snap241
					d4 = snap242
					d5 = snap243
					d22 = snap244
					d23 = snap245
					d24 = snap246
					d25 = snap247
					d27 = snap248
					d28 = snap249
					d29 = snap250
					d30 = snap251
					d63 = snap252
					d64 = snap253
					d65 = snap254
					d66 = snap255
					d107 = snap256
					d108 = snap257
					d109 = snap258
					d110 = snap259
					d159 = snap260
					d160 = snap261
					d161 = snap262
					d162 = snap263
					d164 = snap264
					d165 = snap265
					d166 = snap266
					d167 = snap267
					d232 = snap268
					d233 = snap269
					d234 = snap270
					d235 = snap271
					d236 = snap272
					ctx.RestoreAllocState(alloc273)
					d1 = snap239
					d2 = snap240
					d3 = snap241
					d4 = snap242
					d5 = snap243
					d22 = snap244
					d23 = snap245
					d24 = snap246
					d25 = snap247
					d27 = snap248
					d28 = snap249
					d29 = snap250
					d30 = snap251
					d63 = snap252
					d64 = snap253
					d65 = snap254
					d66 = snap255
					d107 = snap256
					d108 = snap257
					d109 = snap258
					d110 = snap259
					d159 = snap260
					d160 = snap261
					d161 = snap262
					d162 = snap263
					d164 = snap264
					d165 = snap265
					d166 = snap266
					d167 = snap267
					d232 = snap268
					d233 = snap269
					d234 = snap270
					d235 = snap271
					d236 = snap272
					ps274 := PhiState{General: true}
					ps274.OverlayValues = make([]JITValueDesc, 237)
					ps274.OverlayValues[1] = d1
					ps274.OverlayValues[2] = d2
					ps274.OverlayValues[3] = d3
					ps274.OverlayValues[4] = d4
					ps274.OverlayValues[5] = d5
					ps274.OverlayValues[22] = d22
					ps274.OverlayValues[23] = d23
					ps274.OverlayValues[24] = d24
					ps274.OverlayValues[25] = d25
					ps274.OverlayValues[27] = d27
					ps274.OverlayValues[28] = d28
					ps274.OverlayValues[29] = d29
					ps274.OverlayValues[30] = d30
					ps274.OverlayValues[63] = d63
					ps274.OverlayValues[64] = d64
					ps274.OverlayValues[65] = d65
					ps274.OverlayValues[66] = d66
					ps274.OverlayValues[107] = d107
					ps274.OverlayValues[108] = d108
					ps274.OverlayValues[109] = d109
					ps274.OverlayValues[110] = d110
					ps274.OverlayValues[159] = d159
					ps274.OverlayValues[160] = d160
					ps274.OverlayValues[161] = d161
					ps274.OverlayValues[162] = d162
					ps274.OverlayValues[164] = d164
					ps274.OverlayValues[165] = d165
					ps274.OverlayValues[166] = d166
					ps274.OverlayValues[167] = d167
					ps274.OverlayValues[232] = d232
					ps274.OverlayValues[233] = d233
					ps274.OverlayValues[234] = d234
					ps274.OverlayValues[235] = d235
					ps274.OverlayValues[236] = d236
					ps275 := PhiState{General: true}
					ps275.OverlayValues = make([]JITValueDesc, 237)
					ps275.OverlayValues[1] = d1
					ps275.OverlayValues[2] = d2
					ps275.OverlayValues[3] = d3
					ps275.OverlayValues[4] = d4
					ps275.OverlayValues[5] = d5
					ps275.OverlayValues[22] = d22
					ps275.OverlayValues[23] = d23
					ps275.OverlayValues[24] = d24
					ps275.OverlayValues[25] = d25
					ps275.OverlayValues[27] = d27
					ps275.OverlayValues[28] = d28
					ps275.OverlayValues[29] = d29
					ps275.OverlayValues[30] = d30
					ps275.OverlayValues[63] = d63
					ps275.OverlayValues[64] = d64
					ps275.OverlayValues[65] = d65
					ps275.OverlayValues[66] = d66
					ps275.OverlayValues[107] = d107
					ps275.OverlayValues[108] = d108
					ps275.OverlayValues[109] = d109
					ps275.OverlayValues[110] = d110
					ps275.OverlayValues[159] = d159
					ps275.OverlayValues[160] = d160
					ps275.OverlayValues[161] = d161
					ps275.OverlayValues[162] = d162
					ps275.OverlayValues[164] = d164
					ps275.OverlayValues[165] = d165
					ps275.OverlayValues[166] = d166
					ps275.OverlayValues[167] = d167
					ps275.OverlayValues[232] = d232
					ps275.OverlayValues[233] = d233
					ps275.OverlayValues[234] = d234
					ps275.OverlayValues[235] = d235
					ps275.OverlayValues[236] = d236
					snap276 := d1
					snap277 := d2
					snap278 := d3
					snap279 := d4
					snap280 := d5
					snap281 := d22
					snap282 := d23
					snap283 := d24
					snap284 := d25
					snap285 := d27
					snap286 := d28
					snap287 := d29
					snap288 := d30
					snap289 := d63
					snap290 := d64
					snap291 := d65
					snap292 := d66
					snap293 := d107
					snap294 := d108
					snap295 := d109
					snap296 := d110
					snap297 := d159
					snap298 := d160
					snap299 := d161
					snap300 := d162
					snap301 := d164
					snap302 := d165
					snap303 := d166
					snap304 := d167
					snap305 := d232
					snap306 := d233
					snap307 := d234
					snap308 := d235
					snap309 := d236
					alloc310 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps275)
					}
					ctx.RestoreAllocState(alloc310)
					d1 = snap276
					d2 = snap277
					d3 = snap278
					d4 = snap279
					d5 = snap280
					d22 = snap281
					d23 = snap282
					d24 = snap283
					d25 = snap284
					d27 = snap285
					d28 = snap286
					d29 = snap287
					d30 = snap288
					d63 = snap289
					d64 = snap290
					d65 = snap291
					d66 = snap292
					d107 = snap293
					d108 = snap294
					d109 = snap295
					d110 = snap296
					d159 = snap297
					d160 = snap298
					d161 = snap299
					d162 = snap300
					d164 = snap301
					d165 = snap302
					d166 = snap303
					d167 = snap304
					d232 = snap305
					d233 = snap306
					d234 = snap307
					d235 = snap308
					d236 = snap309
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps274)
					}
					return result
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d311 := ps.PhiValues[0]
							ctx.EnsureDesc(&d311)
							ctx.EmitStoreScmerToStack(d311, int32(bbs[9].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
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
					ctx.EnsureDesc(&d164)
					ctx.EnsureDesc(&d164)
					if d164.Loc == LocRegPair || d164.Loc == LocStackPair || d164.Loc == LocRegTriple || d164.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d164)
					d312 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d1, d164}, 3)
					d312.NoHeapPointer = false
					ctx.BindReg(d312.Reg, &d312)
					ctx.BindReg(d312.Reg2, &d312)
					ctx.BindReg(d312.Reg3, &d312)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d312)
					d313 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d312}, 1)
					d313.NoHeapPointer = true
					ctx.BindReg(d313.Reg, &d313)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d312)
					d314 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d312}, 1)
					d314.NoHeapPointer = true
					ctx.BindReg(d314.Reg, &d314)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d312)
					d315 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d312}, 1)
					d315.NoHeapPointer = true
					ctx.BindReg(d315.Reg, &d315)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d312)
					d316 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d312}, 1)
					d316.NoHeapPointer = true
					ctx.BindReg(d316.Reg, &d316)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d312)
					d317 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d312}, 1)
					d317.NoHeapPointer = true
					ctx.BindReg(d317.Reg, &d317)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d312)
					d318 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d312}, 1)
					d318.NoHeapPointer = true
					ctx.BindReg(d318.Reg, &d318)
					ctx.FreeDesc(&d312)
					d319 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc == LocRegPair || d313.Loc == LocStackPair || d313.Loc == LocRegTriple || d313.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d314)
					ctx.EnsureDesc(&d314)
					if d314.Loc == LocRegPair || d314.Loc == LocStackPair || d314.Loc == LocRegTriple || d314.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d315)
					ctx.EnsureDesc(&d315)
					if d315.Loc == LocRegPair || d315.Loc == LocStackPair || d315.Loc == LocRegTriple || d315.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d316)
					ctx.EnsureDesc(&d316)
					if d316.Loc == LocRegPair || d316.Loc == LocStackPair || d316.Loc == LocRegTriple || d316.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d317)
					ctx.EnsureDesc(&d317)
					if d317.Loc == LocRegPair || d317.Loc == LocStackPair || d317.Loc == LocRegTriple || d317.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d318)
					ctx.EnsureDesc(&d318)
					if d318.Loc == LocRegPair || d318.Loc == LocStackPair || d318.Loc == LocRegTriple || d318.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d320 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d320.Loc == LocRegPair || d320.Loc == LocStackPair || d320.Loc == LocRegTriple || d320.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d319)
					ctx.EnsureDesc(&d319)
					if d319.Loc == LocRegPair || d319.Loc == LocStackPair || d319.Loc == LocRegTriple || d319.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d313)
					ctx.SyncDesc(&d314)
					ctx.SyncDesc(&d315)
					ctx.SyncDesc(&d316)
					ctx.SyncDesc(&d317)
					ctx.SyncDesc(&d318)
					ctx.SyncDesc(&d320)
					ctx.SyncDesc(&d319)
					d321 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d313, d314, d315, d316, d317, d318, d320, d319}, 3)
					d321.NoHeapPointer = false
					ctx.BindReg(d321.Reg, &d321)
					ctx.BindReg(d321.Reg2, &d321)
					ctx.BindReg(d321.Reg3, &d321)
					ctx.FreeDesc(&d320)
					ctx.FreeDesc(&d313)
					ctx.FreeDesc(&d314)
					ctx.FreeDesc(&d315)
					ctx.FreeDesc(&d316)
					ctx.FreeDesc(&d317)
					ctx.FreeDesc(&d318)
					ctx.FreeDesc(&d319)
					ctx.EnsureDesc(&d321)
					ctx.EnsureDesc(&d321)
					ctx.EnsureDesc(&d321)
					if d321.Loc != LocRegTriple && d321.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d321)
					d322 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d321}, 1)
					d322.NoHeapPointer = true
					ctx.BindReg(d322.Reg, &d322)
					ctx.FreeDesc(&d321)
					ctx.EnsureDesc(&d322)
					ctx.EnsureDesc(&d322)
					if d322.Loc == LocRegPair || d322.Loc == LocStackPair || d322.Loc == LocRegTriple || d322.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d322)
					d323 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d322}, 2)
					d323.NoHeapPointer = false
					ctx.BindReg(d323.Reg, &d323)
					ctx.BindReg(d323.Reg2, &d323)
					ctx.FreeDesc(&d322)
					ctx.SyncDesc(&d323)
					if d323.Loc == LocRegPair || d323.Loc == LocStackPair || d323.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d323, &result)
						result.Type = d323.Type
					} else {
						switch d323.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d323)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d323)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d323)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d323, &result)
							result.Type = d323.Type
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
					}
					if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != LocNone {
						d312 = ps.OverlayValues[312]
					}
					if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != LocNone {
						d313 = ps.OverlayValues[313]
					}
					if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != LocNone {
						d314 = ps.OverlayValues[314]
					}
					if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != LocNone {
						d315 = ps.OverlayValues[315]
					}
					if len(ps.OverlayValues) > 316 && ps.OverlayValues[316].Loc != LocNone {
						d316 = ps.OverlayValues[316]
					}
					if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != LocNone {
						d317 = ps.OverlayValues[317]
					}
					if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != LocNone {
						d318 = ps.OverlayValues[318]
					}
					if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != LocNone {
						d319 = ps.OverlayValues[319]
					}
					if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != LocNone {
						d320 = ps.OverlayValues[320]
					}
					if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != LocNone {
						d321 = ps.OverlayValues[321]
					}
					if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != LocNone {
						d322 = ps.OverlayValues[322]
					}
					if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != LocNone {
						d323 = ps.OverlayValues[323]
					}
					ctx.ReclaimUntrackedRegs()
					d324 = args[0]
					d324.ID = 0
					var d325 JITValueDesc
					if d324.Loc == LocImm {
						d325 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d324.Imm.Int())}
					} else if d324.Type == tagInt && d324.Loc == LocRegPair {
						ctx.FreeReg(d324.Reg)
						d325 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d324.Reg2}
						ctx.BindReg(d324.Reg2, &d325)
						ctx.BindReg(d324.Reg2, &d325)
					} else if d324.Type == tagInt && d324.Loc == LocReg {
						d325 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d324.Reg}
						ctx.BindReg(d324.Reg, &d325)
						ctx.BindReg(d324.Reg, &d325)
					} else {
						d325 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d324}, 1)
						d325.Type = tagInt
						ctx.BindReg(d325.Reg, &d325)
					}
					ctx.FreeDesc(&d324)
					ctx.EnsureDesc(&d325)
					ctx.EnsureDesc(&d325)
					if d325.Loc == LocRegPair || d325.Loc == LocStackPair || d325.Loc == LocRegTriple || d325.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d326 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d326.Loc == LocRegPair || d326.Loc == LocStackPair || d326.Loc == LocRegTriple || d326.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d325)
					ctx.SyncDesc(&d326)
					d327 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d325, d326}, 3)
					d327.NoHeapPointer = false
					ctx.BindReg(d327.Reg, &d327)
					ctx.BindReg(d327.Reg2, &d327)
					ctx.BindReg(d327.Reg3, &d327)
					ctx.FreeDesc(&d326)
					ctx.FreeDesc(&d325)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d327)
					d328 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d327}, 3)
					d328.NoHeapPointer = false
					ctx.BindReg(d328.Reg, &d328)
					ctx.BindReg(d328.Reg2, &d328)
					ctx.BindReg(d328.Reg3, &d328)
					ctx.FreeDesc(&d327)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc != LocRegTriple && d328.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d328)
					d329 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d328}, 1)
					d329.NoHeapPointer = true
					ctx.BindReg(d329.Reg, &d329)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc != LocRegTriple && d328.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d328)
					d330 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d328}, 1)
					d330.NoHeapPointer = true
					ctx.BindReg(d330.Reg, &d330)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc != LocRegTriple && d328.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d328)
					d331 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d328}, 1)
					d331.NoHeapPointer = true
					ctx.BindReg(d331.Reg, &d331)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc != LocRegTriple && d328.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d328)
					d332 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d328}, 1)
					d332.NoHeapPointer = true
					ctx.BindReg(d332.Reg, &d332)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc != LocRegTriple && d328.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d328)
					d333 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d328}, 1)
					d333.NoHeapPointer = true
					ctx.BindReg(d333.Reg, &d333)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc != LocRegTriple && d328.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d328)
					d334 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d328}, 1)
					d334.NoHeapPointer = true
					ctx.BindReg(d334.Reg, &d334)
					ctx.FreeDesc(&d328)
					ctx.EnsureDesc(&d329)
					ctx.EnsureDesc(&d329)
					if d329.Loc == LocRegPair || d329.Loc == LocStackPair || d329.Loc == LocRegTriple || d329.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d330)
					ctx.EnsureDesc(&d330)
					if d330.Loc == LocRegPair || d330.Loc == LocStackPair || d330.Loc == LocRegTriple || d330.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d331)
					ctx.EnsureDesc(&d331)
					if d331.Loc == LocRegPair || d331.Loc == LocStackPair || d331.Loc == LocRegTriple || d331.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d332)
					ctx.EnsureDesc(&d332)
					if d332.Loc == LocRegPair || d332.Loc == LocStackPair || d332.Loc == LocRegTriple || d332.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d333)
					ctx.EnsureDesc(&d333)
					if d333.Loc == LocRegPair || d333.Loc == LocStackPair || d333.Loc == LocRegTriple || d333.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d334)
					ctx.EnsureDesc(&d334)
					if d334.Loc == LocRegPair || d334.Loc == LocStackPair || d334.Loc == LocRegTriple || d334.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d335 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d335.Loc == LocRegPair || d335.Loc == LocStackPair || d335.Loc == LocRegTriple || d335.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d329)
					ctx.SyncDesc(&d330)
					ctx.SyncDesc(&d331)
					ctx.SyncDesc(&d332)
					ctx.SyncDesc(&d333)
					ctx.SyncDesc(&d334)
					ctx.SyncDesc(&d335)
					ctx.SyncDesc(&d27)
					d336 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d329, d330, d331, d332, d333, d334, d335, d27}, 3)
					d336.NoHeapPointer = false
					ctx.BindReg(d336.Reg, &d336)
					ctx.BindReg(d336.Reg2, &d336)
					ctx.BindReg(d336.Reg3, &d336)
					ctx.FreeDesc(&d335)
					ctx.StabilizeDescForControlFlow(&d336)
					ctx.FreeDesc(&d329)
					ctx.FreeDesc(&d330)
					ctx.FreeDesc(&d331)
					ctx.FreeDesc(&d332)
					ctx.FreeDesc(&d333)
					ctx.FreeDesc(&d334)
					if ps.General {
						ctx.SyncDesc(&d336)
						if d336.Loc == LocReg {
							ctx.ProtectReg(d336.Reg)
						} else if d336.Loc == LocRegPair {
							ctx.ProtectReg(d336.Reg)
							ctx.ProtectReg(d336.Reg2)
						}
						d337 = d336
						if d337.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d337)
						if d337.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d337, int32(bbs[9].PhiBase)+int32(0), 2)
						} else if d337.Loc == LocInputPair {
							ctx.EnsureDesc(&d337)
							ctx.EmitStoreScmerToStack(d337, int32(bbs[9].PhiBase)+int32(0))
						} else if d337.Loc == LocRegPair || d337.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d337, int32(bbs[9].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d337)
							ctx.EmitStoreToStack(d337, int32(bbs[9].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
						}
						if d336.Loc == LocReg {
							ctx.UnprotectReg(d336.Reg)
						} else if d336.Loc == LocRegPair {
							ctx.UnprotectReg(d336.Reg)
							ctx.UnprotectReg(d336.Reg2)
						}
					}
					ps338 := PhiState{General: ps.General}
					ps338.OverlayValues = make([]JITValueDesc, 338)
					ps338.OverlayValues[1] = d1
					ps338.OverlayValues[2] = d2
					ps338.OverlayValues[3] = d3
					ps338.OverlayValues[4] = d4
					ps338.OverlayValues[5] = d5
					ps338.OverlayValues[22] = d22
					ps338.OverlayValues[23] = d23
					ps338.OverlayValues[24] = d24
					ps338.OverlayValues[25] = d25
					ps338.OverlayValues[27] = d27
					ps338.OverlayValues[28] = d28
					ps338.OverlayValues[29] = d29
					ps338.OverlayValues[30] = d30
					ps338.OverlayValues[63] = d63
					ps338.OverlayValues[64] = d64
					ps338.OverlayValues[65] = d65
					ps338.OverlayValues[66] = d66
					ps338.OverlayValues[107] = d107
					ps338.OverlayValues[108] = d108
					ps338.OverlayValues[109] = d109
					ps338.OverlayValues[110] = d110
					ps338.OverlayValues[159] = d159
					ps338.OverlayValues[160] = d160
					ps338.OverlayValues[161] = d161
					ps338.OverlayValues[162] = d162
					ps338.OverlayValues[164] = d164
					ps338.OverlayValues[165] = d165
					ps338.OverlayValues[166] = d166
					ps338.OverlayValues[167] = d167
					ps338.OverlayValues[232] = d232
					ps338.OverlayValues[233] = d233
					ps338.OverlayValues[234] = d234
					ps338.OverlayValues[235] = d235
					ps338.OverlayValues[236] = d236
					ps338.OverlayValues[311] = d311
					ps338.OverlayValues[312] = d312
					ps338.OverlayValues[313] = d313
					ps338.OverlayValues[314] = d314
					ps338.OverlayValues[315] = d315
					ps338.OverlayValues[316] = d316
					ps338.OverlayValues[317] = d317
					ps338.OverlayValues[318] = d318
					ps338.OverlayValues[319] = d319
					ps338.OverlayValues[320] = d320
					ps338.OverlayValues[321] = d321
					ps338.OverlayValues[322] = d322
					ps338.OverlayValues[323] = d323
					ps338.OverlayValues[324] = d324
					ps338.OverlayValues[325] = d325
					ps338.OverlayValues[326] = d326
					ps338.OverlayValues[327] = d327
					ps338.OverlayValues[328] = d328
					ps338.OverlayValues[329] = d329
					ps338.OverlayValues[330] = d330
					ps338.OverlayValues[331] = d331
					ps338.OverlayValues[332] = d332
					ps338.OverlayValues[333] = d333
					ps338.OverlayValues[334] = d334
					ps338.OverlayValues[335] = d335
					ps338.OverlayValues[336] = d336
					ps338.OverlayValues[337] = d337
					ps338.PhiValues = make([]JITValueDesc, 1)
					d339 = d336
					ps338.PhiValues[0] = d339
					if ps338.General && bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					return bbs[9].RenderPS(ps338)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
					}
					if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != LocNone {
						d312 = ps.OverlayValues[312]
					}
					if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != LocNone {
						d313 = ps.OverlayValues[313]
					}
					if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != LocNone {
						d314 = ps.OverlayValues[314]
					}
					if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != LocNone {
						d315 = ps.OverlayValues[315]
					}
					if len(ps.OverlayValues) > 316 && ps.OverlayValues[316].Loc != LocNone {
						d316 = ps.OverlayValues[316]
					}
					if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != LocNone {
						d317 = ps.OverlayValues[317]
					}
					if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != LocNone {
						d318 = ps.OverlayValues[318]
					}
					if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != LocNone {
						d319 = ps.OverlayValues[319]
					}
					if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != LocNone {
						d320 = ps.OverlayValues[320]
					}
					if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != LocNone {
						d321 = ps.OverlayValues[321]
					}
					if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != LocNone {
						d322 = ps.OverlayValues[322]
					}
					if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != LocNone {
						d323 = ps.OverlayValues[323]
					}
					if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != LocNone {
						d324 = ps.OverlayValues[324]
					}
					if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
						d325 = ps.OverlayValues[325]
					}
					if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
						d326 = ps.OverlayValues[326]
					}
					if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
						d327 = ps.OverlayValues[327]
					}
					if len(ps.OverlayValues) > 328 && ps.OverlayValues[328].Loc != LocNone {
						d328 = ps.OverlayValues[328]
					}
					if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
						d329 = ps.OverlayValues[329]
					}
					if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
						d330 = ps.OverlayValues[330]
					}
					if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
						d331 = ps.OverlayValues[331]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					ctx.ReclaimUntrackedRegs()
					d340 = args[0]
					d340.ID = 0
					d342 = d340
					ctx.SyncDesc(&d342)
					if d342.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d342.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d342.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d342 = tmpScalar
					}
					d342 = JITPrepareScmerGoArg(ctx, d342)
					if d342.Loc != LocRegPair && d342.Loc != LocStackPair && d342.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d341 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d342}, 2)
					ctx.FreeDesc(&d340)
					ctx.EnsureDesc(&d341)
					ctx.EnsureDesc(&d341)
					ctx.EnsureDesc(&d341)
					if d341.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d341.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d341.Imm)
						ptrWord, _ := d341.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d341.Imm.String())))
						d341 = tmpPair
					} else if d341.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d341.Type, Reg: ctx.AllocRegExcept(d341.Reg), Reg2: ctx.AllocRegExcept(d341.Reg)}
						switch d341.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d341)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d341)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d341)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d341)
						d341 = tmpPair
					}
					if d341.Loc != LocRegPair && d341.Loc != LocStackPair && d341.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (parseDateStringInLoc arg0)")
					}
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d341)
					ctx.SyncDesc(&d27)
					callResults343 := JITEmitGoCallResults(ctx, GoFuncAddr(parseDateStringInLoc), []JITValueDesc{d341, d27}, []uint8{1, 1}, []uint8{0, 0})
					d344 = callResults343[0]
					_ = d344
					d345 = callResults343[1]
					_ = d345
					ctx.StabilizeDescForControlFlow(&d344)
					d346 = d345
					ctx.EnsureDesc(&d346)
					if d346.Loc != LocImm && d346.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d346.Loc == LocImm {
						if d346.Imm.Bool() {
							if ps.General {
							}
							ps347 := PhiState{General: ps.General}
							ps347.OverlayValues = make([]JITValueDesc, 347)
							ps347.OverlayValues[1] = d1
							ps347.OverlayValues[2] = d2
							ps347.OverlayValues[3] = d3
							ps347.OverlayValues[4] = d4
							ps347.OverlayValues[5] = d5
							ps347.OverlayValues[22] = d22
							ps347.OverlayValues[23] = d23
							ps347.OverlayValues[24] = d24
							ps347.OverlayValues[25] = d25
							ps347.OverlayValues[27] = d27
							ps347.OverlayValues[28] = d28
							ps347.OverlayValues[29] = d29
							ps347.OverlayValues[30] = d30
							ps347.OverlayValues[63] = d63
							ps347.OverlayValues[64] = d64
							ps347.OverlayValues[65] = d65
							ps347.OverlayValues[66] = d66
							ps347.OverlayValues[107] = d107
							ps347.OverlayValues[108] = d108
							ps347.OverlayValues[109] = d109
							ps347.OverlayValues[110] = d110
							ps347.OverlayValues[159] = d159
							ps347.OverlayValues[160] = d160
							ps347.OverlayValues[161] = d161
							ps347.OverlayValues[162] = d162
							ps347.OverlayValues[164] = d164
							ps347.OverlayValues[165] = d165
							ps347.OverlayValues[166] = d166
							ps347.OverlayValues[167] = d167
							ps347.OverlayValues[232] = d232
							ps347.OverlayValues[233] = d233
							ps347.OverlayValues[234] = d234
							ps347.OverlayValues[235] = d235
							ps347.OverlayValues[236] = d236
							ps347.OverlayValues[311] = d311
							ps347.OverlayValues[312] = d312
							ps347.OverlayValues[313] = d313
							ps347.OverlayValues[314] = d314
							ps347.OverlayValues[315] = d315
							ps347.OverlayValues[316] = d316
							ps347.OverlayValues[317] = d317
							ps347.OverlayValues[318] = d318
							ps347.OverlayValues[319] = d319
							ps347.OverlayValues[320] = d320
							ps347.OverlayValues[321] = d321
							ps347.OverlayValues[322] = d322
							ps347.OverlayValues[323] = d323
							ps347.OverlayValues[324] = d324
							ps347.OverlayValues[325] = d325
							ps347.OverlayValues[326] = d326
							ps347.OverlayValues[327] = d327
							ps347.OverlayValues[328] = d328
							ps347.OverlayValues[329] = d329
							ps347.OverlayValues[330] = d330
							ps347.OverlayValues[331] = d331
							ps347.OverlayValues[332] = d332
							ps347.OverlayValues[333] = d333
							ps347.OverlayValues[334] = d334
							ps347.OverlayValues[335] = d335
							ps347.OverlayValues[336] = d336
							ps347.OverlayValues[337] = d337
							ps347.OverlayValues[339] = d339
							ps347.OverlayValues[340] = d340
							ps347.OverlayValues[341] = d341
							ps347.OverlayValues[342] = d342
							ps347.OverlayValues[344] = d344
							ps347.OverlayValues[345] = d345
							ps347.OverlayValues[346] = d346
							return bbs[13].RenderPS(ps347)
						}
						if ps.General {
						}
						ps348 := PhiState{General: ps.General}
						ps348.OverlayValues = make([]JITValueDesc, 347)
						ps348.OverlayValues[1] = d1
						ps348.OverlayValues[2] = d2
						ps348.OverlayValues[3] = d3
						ps348.OverlayValues[4] = d4
						ps348.OverlayValues[5] = d5
						ps348.OverlayValues[22] = d22
						ps348.OverlayValues[23] = d23
						ps348.OverlayValues[24] = d24
						ps348.OverlayValues[25] = d25
						ps348.OverlayValues[27] = d27
						ps348.OverlayValues[28] = d28
						ps348.OverlayValues[29] = d29
						ps348.OverlayValues[30] = d30
						ps348.OverlayValues[63] = d63
						ps348.OverlayValues[64] = d64
						ps348.OverlayValues[65] = d65
						ps348.OverlayValues[66] = d66
						ps348.OverlayValues[107] = d107
						ps348.OverlayValues[108] = d108
						ps348.OverlayValues[109] = d109
						ps348.OverlayValues[110] = d110
						ps348.OverlayValues[159] = d159
						ps348.OverlayValues[160] = d160
						ps348.OverlayValues[161] = d161
						ps348.OverlayValues[162] = d162
						ps348.OverlayValues[164] = d164
						ps348.OverlayValues[165] = d165
						ps348.OverlayValues[166] = d166
						ps348.OverlayValues[167] = d167
						ps348.OverlayValues[232] = d232
						ps348.OverlayValues[233] = d233
						ps348.OverlayValues[234] = d234
						ps348.OverlayValues[235] = d235
						ps348.OverlayValues[236] = d236
						ps348.OverlayValues[311] = d311
						ps348.OverlayValues[312] = d312
						ps348.OverlayValues[313] = d313
						ps348.OverlayValues[314] = d314
						ps348.OverlayValues[315] = d315
						ps348.OverlayValues[316] = d316
						ps348.OverlayValues[317] = d317
						ps348.OverlayValues[318] = d318
						ps348.OverlayValues[319] = d319
						ps348.OverlayValues[320] = d320
						ps348.OverlayValues[321] = d321
						ps348.OverlayValues[322] = d322
						ps348.OverlayValues[323] = d323
						ps348.OverlayValues[324] = d324
						ps348.OverlayValues[325] = d325
						ps348.OverlayValues[326] = d326
						ps348.OverlayValues[327] = d327
						ps348.OverlayValues[328] = d328
						ps348.OverlayValues[329] = d329
						ps348.OverlayValues[330] = d330
						ps348.OverlayValues[331] = d331
						ps348.OverlayValues[332] = d332
						ps348.OverlayValues[333] = d333
						ps348.OverlayValues[334] = d334
						ps348.OverlayValues[335] = d335
						ps348.OverlayValues[336] = d336
						ps348.OverlayValues[337] = d337
						ps348.OverlayValues[339] = d339
						ps348.OverlayValues[340] = d340
						ps348.OverlayValues[341] = d341
						ps348.OverlayValues[342] = d342
						ps348.OverlayValues[344] = d344
						ps348.OverlayValues[345] = d345
						ps348.OverlayValues[346] = d346
						return bbs[12].RenderPS(ps348)
					}
					if !ps.General {
						ps.General = true
						return bbs[11].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d346.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					snap349 := d1
					snap350 := d2
					snap351 := d3
					snap352 := d4
					snap353 := d5
					snap354 := d22
					snap355 := d23
					snap356 := d24
					snap357 := d25
					snap358 := d27
					snap359 := d28
					snap360 := d29
					snap361 := d30
					snap362 := d63
					snap363 := d64
					snap364 := d65
					snap365 := d66
					snap366 := d107
					snap367 := d108
					snap368 := d109
					snap369 := d110
					snap370 := d159
					snap371 := d160
					snap372 := d161
					snap373 := d162
					snap374 := d164
					snap375 := d165
					snap376 := d166
					snap377 := d167
					snap378 := d232
					snap379 := d233
					snap380 := d234
					snap381 := d235
					snap382 := d236
					snap383 := d311
					snap384 := d312
					snap385 := d313
					snap386 := d314
					snap387 := d315
					snap388 := d316
					snap389 := d317
					snap390 := d318
					snap391 := d319
					snap392 := d320
					snap393 := d321
					snap394 := d322
					snap395 := d323
					snap396 := d324
					snap397 := d325
					snap398 := d326
					snap399 := d327
					snap400 := d328
					snap401 := d329
					snap402 := d330
					snap403 := d331
					snap404 := d332
					snap405 := d333
					snap406 := d334
					snap407 := d335
					snap408 := d336
					snap409 := d337
					snap410 := d339
					snap411 := d340
					snap412 := d341
					snap413 := d342
					snap414 := d344
					snap415 := d345
					snap416 := d346
					alloc417 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc417)
					d1 = snap349
					d2 = snap350
					d3 = snap351
					d4 = snap352
					d5 = snap353
					d22 = snap354
					d23 = snap355
					d24 = snap356
					d25 = snap357
					d27 = snap358
					d28 = snap359
					d29 = snap360
					d30 = snap361
					d63 = snap362
					d64 = snap363
					d65 = snap364
					d66 = snap365
					d107 = snap366
					d108 = snap367
					d109 = snap368
					d110 = snap369
					d159 = snap370
					d160 = snap371
					d161 = snap372
					d162 = snap373
					d164 = snap374
					d165 = snap375
					d166 = snap376
					d167 = snap377
					d232 = snap378
					d233 = snap379
					d234 = snap380
					d235 = snap381
					d236 = snap382
					d311 = snap383
					d312 = snap384
					d313 = snap385
					d314 = snap386
					d315 = snap387
					d316 = snap388
					d317 = snap389
					d318 = snap390
					d319 = snap391
					d320 = snap392
					d321 = snap393
					d322 = snap394
					d323 = snap395
					d324 = snap396
					d325 = snap397
					d326 = snap398
					d327 = snap399
					d328 = snap400
					d329 = snap401
					d330 = snap402
					d331 = snap403
					d332 = snap404
					d333 = snap405
					d334 = snap406
					d335 = snap407
					d336 = snap408
					d337 = snap409
					d339 = snap410
					d340 = snap411
					d341 = snap412
					d342 = snap413
					d344 = snap414
					d345 = snap415
					d346 = snap416
					ctx.RestoreAllocState(alloc417)
					d1 = snap349
					d2 = snap350
					d3 = snap351
					d4 = snap352
					d5 = snap353
					d22 = snap354
					d23 = snap355
					d24 = snap356
					d25 = snap357
					d27 = snap358
					d28 = snap359
					d29 = snap360
					d30 = snap361
					d63 = snap362
					d64 = snap363
					d65 = snap364
					d66 = snap365
					d107 = snap366
					d108 = snap367
					d109 = snap368
					d110 = snap369
					d159 = snap370
					d160 = snap371
					d161 = snap372
					d162 = snap373
					d164 = snap374
					d165 = snap375
					d166 = snap376
					d167 = snap377
					d232 = snap378
					d233 = snap379
					d234 = snap380
					d235 = snap381
					d236 = snap382
					d311 = snap383
					d312 = snap384
					d313 = snap385
					d314 = snap386
					d315 = snap387
					d316 = snap388
					d317 = snap389
					d318 = snap390
					d319 = snap391
					d320 = snap392
					d321 = snap393
					d322 = snap394
					d323 = snap395
					d324 = snap396
					d325 = snap397
					d326 = snap398
					d327 = snap399
					d328 = snap400
					d329 = snap401
					d330 = snap402
					d331 = snap403
					d332 = snap404
					d333 = snap405
					d334 = snap406
					d335 = snap407
					d336 = snap408
					d337 = snap409
					d339 = snap410
					d340 = snap411
					d341 = snap412
					d342 = snap413
					d344 = snap414
					d345 = snap415
					d346 = snap416
					ps418 := PhiState{General: true}
					ps418.OverlayValues = make([]JITValueDesc, 347)
					ps418.OverlayValues[1] = d1
					ps418.OverlayValues[2] = d2
					ps418.OverlayValues[3] = d3
					ps418.OverlayValues[4] = d4
					ps418.OverlayValues[5] = d5
					ps418.OverlayValues[22] = d22
					ps418.OverlayValues[23] = d23
					ps418.OverlayValues[24] = d24
					ps418.OverlayValues[25] = d25
					ps418.OverlayValues[27] = d27
					ps418.OverlayValues[28] = d28
					ps418.OverlayValues[29] = d29
					ps418.OverlayValues[30] = d30
					ps418.OverlayValues[63] = d63
					ps418.OverlayValues[64] = d64
					ps418.OverlayValues[65] = d65
					ps418.OverlayValues[66] = d66
					ps418.OverlayValues[107] = d107
					ps418.OverlayValues[108] = d108
					ps418.OverlayValues[109] = d109
					ps418.OverlayValues[110] = d110
					ps418.OverlayValues[159] = d159
					ps418.OverlayValues[160] = d160
					ps418.OverlayValues[161] = d161
					ps418.OverlayValues[162] = d162
					ps418.OverlayValues[164] = d164
					ps418.OverlayValues[165] = d165
					ps418.OverlayValues[166] = d166
					ps418.OverlayValues[167] = d167
					ps418.OverlayValues[232] = d232
					ps418.OverlayValues[233] = d233
					ps418.OverlayValues[234] = d234
					ps418.OverlayValues[235] = d235
					ps418.OverlayValues[236] = d236
					ps418.OverlayValues[311] = d311
					ps418.OverlayValues[312] = d312
					ps418.OverlayValues[313] = d313
					ps418.OverlayValues[314] = d314
					ps418.OverlayValues[315] = d315
					ps418.OverlayValues[316] = d316
					ps418.OverlayValues[317] = d317
					ps418.OverlayValues[318] = d318
					ps418.OverlayValues[319] = d319
					ps418.OverlayValues[320] = d320
					ps418.OverlayValues[321] = d321
					ps418.OverlayValues[322] = d322
					ps418.OverlayValues[323] = d323
					ps418.OverlayValues[324] = d324
					ps418.OverlayValues[325] = d325
					ps418.OverlayValues[326] = d326
					ps418.OverlayValues[327] = d327
					ps418.OverlayValues[328] = d328
					ps418.OverlayValues[329] = d329
					ps418.OverlayValues[330] = d330
					ps418.OverlayValues[331] = d331
					ps418.OverlayValues[332] = d332
					ps418.OverlayValues[333] = d333
					ps418.OverlayValues[334] = d334
					ps418.OverlayValues[335] = d335
					ps418.OverlayValues[336] = d336
					ps418.OverlayValues[337] = d337
					ps418.OverlayValues[339] = d339
					ps418.OverlayValues[340] = d340
					ps418.OverlayValues[341] = d341
					ps418.OverlayValues[342] = d342
					ps418.OverlayValues[344] = d344
					ps418.OverlayValues[345] = d345
					ps418.OverlayValues[346] = d346
					ps419 := PhiState{General: true}
					ps419.OverlayValues = make([]JITValueDesc, 347)
					ps419.OverlayValues[1] = d1
					ps419.OverlayValues[2] = d2
					ps419.OverlayValues[3] = d3
					ps419.OverlayValues[4] = d4
					ps419.OverlayValues[5] = d5
					ps419.OverlayValues[22] = d22
					ps419.OverlayValues[23] = d23
					ps419.OverlayValues[24] = d24
					ps419.OverlayValues[25] = d25
					ps419.OverlayValues[27] = d27
					ps419.OverlayValues[28] = d28
					ps419.OverlayValues[29] = d29
					ps419.OverlayValues[30] = d30
					ps419.OverlayValues[63] = d63
					ps419.OverlayValues[64] = d64
					ps419.OverlayValues[65] = d65
					ps419.OverlayValues[66] = d66
					ps419.OverlayValues[107] = d107
					ps419.OverlayValues[108] = d108
					ps419.OverlayValues[109] = d109
					ps419.OverlayValues[110] = d110
					ps419.OverlayValues[159] = d159
					ps419.OverlayValues[160] = d160
					ps419.OverlayValues[161] = d161
					ps419.OverlayValues[162] = d162
					ps419.OverlayValues[164] = d164
					ps419.OverlayValues[165] = d165
					ps419.OverlayValues[166] = d166
					ps419.OverlayValues[167] = d167
					ps419.OverlayValues[232] = d232
					ps419.OverlayValues[233] = d233
					ps419.OverlayValues[234] = d234
					ps419.OverlayValues[235] = d235
					ps419.OverlayValues[236] = d236
					ps419.OverlayValues[311] = d311
					ps419.OverlayValues[312] = d312
					ps419.OverlayValues[313] = d313
					ps419.OverlayValues[314] = d314
					ps419.OverlayValues[315] = d315
					ps419.OverlayValues[316] = d316
					ps419.OverlayValues[317] = d317
					ps419.OverlayValues[318] = d318
					ps419.OverlayValues[319] = d319
					ps419.OverlayValues[320] = d320
					ps419.OverlayValues[321] = d321
					ps419.OverlayValues[322] = d322
					ps419.OverlayValues[323] = d323
					ps419.OverlayValues[324] = d324
					ps419.OverlayValues[325] = d325
					ps419.OverlayValues[326] = d326
					ps419.OverlayValues[327] = d327
					ps419.OverlayValues[328] = d328
					ps419.OverlayValues[329] = d329
					ps419.OverlayValues[330] = d330
					ps419.OverlayValues[331] = d331
					ps419.OverlayValues[332] = d332
					ps419.OverlayValues[333] = d333
					ps419.OverlayValues[334] = d334
					ps419.OverlayValues[335] = d335
					ps419.OverlayValues[336] = d336
					ps419.OverlayValues[337] = d337
					ps419.OverlayValues[339] = d339
					ps419.OverlayValues[340] = d340
					ps419.OverlayValues[341] = d341
					ps419.OverlayValues[342] = d342
					ps419.OverlayValues[344] = d344
					ps419.OverlayValues[345] = d345
					ps419.OverlayValues[346] = d346
					snap420 := d1
					snap421 := d2
					snap422 := d3
					snap423 := d4
					snap424 := d5
					snap425 := d22
					snap426 := d23
					snap427 := d24
					snap428 := d25
					snap429 := d27
					snap430 := d28
					snap431 := d29
					snap432 := d30
					snap433 := d63
					snap434 := d64
					snap435 := d65
					snap436 := d66
					snap437 := d107
					snap438 := d108
					snap439 := d109
					snap440 := d110
					snap441 := d159
					snap442 := d160
					snap443 := d161
					snap444 := d162
					snap445 := d164
					snap446 := d165
					snap447 := d166
					snap448 := d167
					snap449 := d232
					snap450 := d233
					snap451 := d234
					snap452 := d235
					snap453 := d236
					snap454 := d311
					snap455 := d312
					snap456 := d313
					snap457 := d314
					snap458 := d315
					snap459 := d316
					snap460 := d317
					snap461 := d318
					snap462 := d319
					snap463 := d320
					snap464 := d321
					snap465 := d322
					snap466 := d323
					snap467 := d324
					snap468 := d325
					snap469 := d326
					snap470 := d327
					snap471 := d328
					snap472 := d329
					snap473 := d330
					snap474 := d331
					snap475 := d332
					snap476 := d333
					snap477 := d334
					snap478 := d335
					snap479 := d336
					snap480 := d337
					snap481 := d339
					snap482 := d340
					snap483 := d341
					snap484 := d342
					snap485 := d344
					snap486 := d345
					snap487 := d346
					alloc488 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps419)
					}
					ctx.RestoreAllocState(alloc488)
					d1 = snap420
					d2 = snap421
					d3 = snap422
					d4 = snap423
					d5 = snap424
					d22 = snap425
					d23 = snap426
					d24 = snap427
					d25 = snap428
					d27 = snap429
					d28 = snap430
					d29 = snap431
					d30 = snap432
					d63 = snap433
					d64 = snap434
					d65 = snap435
					d66 = snap436
					d107 = snap437
					d108 = snap438
					d109 = snap439
					d110 = snap440
					d159 = snap441
					d160 = snap442
					d161 = snap443
					d162 = snap444
					d164 = snap445
					d165 = snap446
					d166 = snap447
					d167 = snap448
					d232 = snap449
					d233 = snap450
					d234 = snap451
					d235 = snap452
					d236 = snap453
					d311 = snap454
					d312 = snap455
					d313 = snap456
					d314 = snap457
					d315 = snap458
					d316 = snap459
					d317 = snap460
					d318 = snap461
					d319 = snap462
					d320 = snap463
					d321 = snap464
					d322 = snap465
					d323 = snap466
					d324 = snap467
					d325 = snap468
					d326 = snap469
					d327 = snap470
					d328 = snap471
					d329 = snap472
					d330 = snap473
					d331 = snap474
					d332 = snap475
					d333 = snap476
					d334 = snap477
					d335 = snap478
					d336 = snap479
					d337 = snap480
					d339 = snap481
					d340 = snap482
					d341 = snap483
					d342 = snap484
					d344 = snap485
					d345 = snap486
					d346 = snap487
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps418)
					}
					return result
					ctx.FreeDesc(&d345)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
					}
					if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != LocNone {
						d312 = ps.OverlayValues[312]
					}
					if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != LocNone {
						d313 = ps.OverlayValues[313]
					}
					if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != LocNone {
						d314 = ps.OverlayValues[314]
					}
					if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != LocNone {
						d315 = ps.OverlayValues[315]
					}
					if len(ps.OverlayValues) > 316 && ps.OverlayValues[316].Loc != LocNone {
						d316 = ps.OverlayValues[316]
					}
					if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != LocNone {
						d317 = ps.OverlayValues[317]
					}
					if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != LocNone {
						d318 = ps.OverlayValues[318]
					}
					if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != LocNone {
						d319 = ps.OverlayValues[319]
					}
					if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != LocNone {
						d320 = ps.OverlayValues[320]
					}
					if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != LocNone {
						d321 = ps.OverlayValues[321]
					}
					if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != LocNone {
						d322 = ps.OverlayValues[322]
					}
					if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != LocNone {
						d323 = ps.OverlayValues[323]
					}
					if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != LocNone {
						d324 = ps.OverlayValues[324]
					}
					if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
						d325 = ps.OverlayValues[325]
					}
					if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
						d326 = ps.OverlayValues[326]
					}
					if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
						d327 = ps.OverlayValues[327]
					}
					if len(ps.OverlayValues) > 328 && ps.OverlayValues[328].Loc != LocNone {
						d328 = ps.OverlayValues[328]
					}
					if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
						d329 = ps.OverlayValues[329]
					}
					if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
						d330 = ps.OverlayValues[330]
					}
					if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
						d331 = ps.OverlayValues[331]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
						d340 = ps.OverlayValues[340]
					}
					if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
						d341 = ps.OverlayValues[341]
					}
					if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
						d342 = ps.OverlayValues[342]
					}
					if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
						d344 = ps.OverlayValues[344]
					}
					if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
						d345 = ps.OverlayValues[345]
					}
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					ctx.ReclaimUntrackedRegs()
					d489 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d489)
					if d489.Loc == LocRegPair || d489.Loc == LocStackPair || d489.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d489, &result)
						result.Type = d489.Type
					} else {
						switch d489.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d489)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d489)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d489)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d489, &result)
							result.Type = d489.Type
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
					}
					if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != LocNone {
						d312 = ps.OverlayValues[312]
					}
					if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != LocNone {
						d313 = ps.OverlayValues[313]
					}
					if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != LocNone {
						d314 = ps.OverlayValues[314]
					}
					if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != LocNone {
						d315 = ps.OverlayValues[315]
					}
					if len(ps.OverlayValues) > 316 && ps.OverlayValues[316].Loc != LocNone {
						d316 = ps.OverlayValues[316]
					}
					if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != LocNone {
						d317 = ps.OverlayValues[317]
					}
					if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != LocNone {
						d318 = ps.OverlayValues[318]
					}
					if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != LocNone {
						d319 = ps.OverlayValues[319]
					}
					if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != LocNone {
						d320 = ps.OverlayValues[320]
					}
					if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != LocNone {
						d321 = ps.OverlayValues[321]
					}
					if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != LocNone {
						d322 = ps.OverlayValues[322]
					}
					if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != LocNone {
						d323 = ps.OverlayValues[323]
					}
					if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != LocNone {
						d324 = ps.OverlayValues[324]
					}
					if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
						d325 = ps.OverlayValues[325]
					}
					if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
						d326 = ps.OverlayValues[326]
					}
					if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
						d327 = ps.OverlayValues[327]
					}
					if len(ps.OverlayValues) > 328 && ps.OverlayValues[328].Loc != LocNone {
						d328 = ps.OverlayValues[328]
					}
					if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
						d329 = ps.OverlayValues[329]
					}
					if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
						d330 = ps.OverlayValues[330]
					}
					if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
						d331 = ps.OverlayValues[331]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
						d340 = ps.OverlayValues[340]
					}
					if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
						d341 = ps.OverlayValues[341]
					}
					if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
						d342 = ps.OverlayValues[342]
					}
					if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
						d344 = ps.OverlayValues[344]
					}
					if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
						d345 = ps.OverlayValues[345]
					}
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					if len(ps.OverlayValues) > 489 && ps.OverlayValues[489].Loc != LocNone {
						d489 = ps.OverlayValues[489]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d344)
					ctx.EnsureDesc(&d344)
					if d344.Loc == LocRegPair || d344.Loc == LocStackPair || d344.Loc == LocRegTriple || d344.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d490 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d490.Loc == LocRegPair || d490.Loc == LocStackPair || d490.Loc == LocRegTriple || d490.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d344)
					ctx.SyncDesc(&d490)
					d491 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d344, d490}, 3)
					d491.NoHeapPointer = false
					ctx.BindReg(d491.Reg, &d491)
					ctx.BindReg(d491.Reg2, &d491)
					ctx.BindReg(d491.Reg3, &d491)
					ctx.FreeDesc(&d490)
					ctx.StabilizeDescForControlFlow(&d491)
					if ps.General {
						ctx.SyncDesc(&d491)
						if d491.Loc == LocReg {
							ctx.ProtectReg(d491.Reg)
						} else if d491.Loc == LocRegPair {
							ctx.ProtectReg(d491.Reg)
							ctx.ProtectReg(d491.Reg2)
						}
						d492 = d491
						if d492.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d492)
						if d492.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d492, int32(bbs[9].PhiBase)+int32(0), 2)
						} else if d492.Loc == LocInputPair {
							ctx.EnsureDesc(&d492)
							ctx.EmitStoreScmerToStack(d492, int32(bbs[9].PhiBase)+int32(0))
						} else if d492.Loc == LocRegPair || d492.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d492, int32(bbs[9].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d492)
							ctx.EmitStoreToStack(d492, int32(bbs[9].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
						}
						if d491.Loc == LocReg {
							ctx.UnprotectReg(d491.Reg)
						} else if d491.Loc == LocRegPair {
							ctx.UnprotectReg(d491.Reg)
							ctx.UnprotectReg(d491.Reg2)
						}
					}
					ps493 := PhiState{General: ps.General}
					ps493.OverlayValues = make([]JITValueDesc, 493)
					ps493.OverlayValues[1] = d1
					ps493.OverlayValues[2] = d2
					ps493.OverlayValues[3] = d3
					ps493.OverlayValues[4] = d4
					ps493.OverlayValues[5] = d5
					ps493.OverlayValues[22] = d22
					ps493.OverlayValues[23] = d23
					ps493.OverlayValues[24] = d24
					ps493.OverlayValues[25] = d25
					ps493.OverlayValues[27] = d27
					ps493.OverlayValues[28] = d28
					ps493.OverlayValues[29] = d29
					ps493.OverlayValues[30] = d30
					ps493.OverlayValues[63] = d63
					ps493.OverlayValues[64] = d64
					ps493.OverlayValues[65] = d65
					ps493.OverlayValues[66] = d66
					ps493.OverlayValues[107] = d107
					ps493.OverlayValues[108] = d108
					ps493.OverlayValues[109] = d109
					ps493.OverlayValues[110] = d110
					ps493.OverlayValues[159] = d159
					ps493.OverlayValues[160] = d160
					ps493.OverlayValues[161] = d161
					ps493.OverlayValues[162] = d162
					ps493.OverlayValues[164] = d164
					ps493.OverlayValues[165] = d165
					ps493.OverlayValues[166] = d166
					ps493.OverlayValues[167] = d167
					ps493.OverlayValues[232] = d232
					ps493.OverlayValues[233] = d233
					ps493.OverlayValues[234] = d234
					ps493.OverlayValues[235] = d235
					ps493.OverlayValues[236] = d236
					ps493.OverlayValues[311] = d311
					ps493.OverlayValues[312] = d312
					ps493.OverlayValues[313] = d313
					ps493.OverlayValues[314] = d314
					ps493.OverlayValues[315] = d315
					ps493.OverlayValues[316] = d316
					ps493.OverlayValues[317] = d317
					ps493.OverlayValues[318] = d318
					ps493.OverlayValues[319] = d319
					ps493.OverlayValues[320] = d320
					ps493.OverlayValues[321] = d321
					ps493.OverlayValues[322] = d322
					ps493.OverlayValues[323] = d323
					ps493.OverlayValues[324] = d324
					ps493.OverlayValues[325] = d325
					ps493.OverlayValues[326] = d326
					ps493.OverlayValues[327] = d327
					ps493.OverlayValues[328] = d328
					ps493.OverlayValues[329] = d329
					ps493.OverlayValues[330] = d330
					ps493.OverlayValues[331] = d331
					ps493.OverlayValues[332] = d332
					ps493.OverlayValues[333] = d333
					ps493.OverlayValues[334] = d334
					ps493.OverlayValues[335] = d335
					ps493.OverlayValues[336] = d336
					ps493.OverlayValues[337] = d337
					ps493.OverlayValues[339] = d339
					ps493.OverlayValues[340] = d340
					ps493.OverlayValues[341] = d341
					ps493.OverlayValues[342] = d342
					ps493.OverlayValues[344] = d344
					ps493.OverlayValues[345] = d345
					ps493.OverlayValues[346] = d346
					ps493.OverlayValues[489] = d489
					ps493.OverlayValues[490] = d490
					ps493.OverlayValues[491] = d491
					ps493.OverlayValues[492] = d492
					ps493.PhiValues = make([]JITValueDesc, 1)
					d494 = d491
					ps493.PhiValues[0] = d494
					if ps493.General && bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					return bbs[9].RenderPS(ps493)
					return result
				}
				ps495 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps495)
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
				var d33 JITValueDesc
				_ = d33
				var d50 JITValueDesc
				_ = d50
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
				var d72 JITValueDesc
				_ = d72
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d79 JITValueDesc
				_ = d79
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d111 JITValueDesc
				_ = d111
				var d114 JITValueDesc
				_ = d114
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
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
				var d155 JITValueDesc
				_ = d155
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
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					snap13 := d5
					snap14 := d6
					alloc15 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc15)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ctx.RestoreAllocState(alloc15)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ps16 := PhiState{General: true}
					ps16.OverlayValues = make([]JITValueDesc, 7)
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					ps16.OverlayValues[6] = d6
					ps17 := PhiState{General: true}
					ps17.OverlayValues = make([]JITValueDesc, 7)
					ps17.OverlayValues[1] = d1
					ps17.OverlayValues[2] = d2
					ps17.OverlayValues[3] = d3
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					snap18 := d1
					snap19 := d2
					snap20 := d3
					snap21 := d4
					snap22 := d5
					snap23 := d6
					alloc24 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps17)
					}
					ctx.RestoreAllocState(alloc24)
					d1 = snap18
					d2 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					d6 = snap23
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps16)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					var d27 JITValueDesc
					if d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d26.Imm.Int())}
					} else if d26.Type == tagInt && d26.Loc == LocRegPair {
						ctx.FreeReg(d26.Reg)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg2}
						ctx.BindReg(d26.Reg2, &d27)
						ctx.BindReg(d26.Reg2, &d27)
					} else if d26.Type == tagInt && d26.Loc == LocReg {
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg}
						ctx.BindReg(d26.Reg, &d27)
						ctx.BindReg(d26.Reg, &d27)
					} else {
						d27 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d26}, 1)
						d27.Type = tagInt
						ctx.BindReg(d27.Reg, &d27)
					}
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.FreeDesc(&d26)
					d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d28)
					var d29 JITValueDesc
					if d28.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d28.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d28.Reg, 2)
						d29 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d29)
					}
					ctx.FreeDesc(&d28)
					d30 = d29
					ctx.EnsureDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							if ps.General {
							}
							ps31 := PhiState{General: ps.General}
							ps31.OverlayValues = make([]JITValueDesc, 31)
							ps31.OverlayValues[1] = d1
							ps31.OverlayValues[2] = d2
							ps31.OverlayValues[3] = d3
							ps31.OverlayValues[4] = d4
							ps31.OverlayValues[5] = d5
							ps31.OverlayValues[6] = d6
							ps31.OverlayValues[25] = d25
							ps31.OverlayValues[26] = d26
							ps31.OverlayValues[27] = d27
							ps31.OverlayValues[28] = d28
							ps31.OverlayValues[29] = d29
							ps31.OverlayValues[30] = d30
							return bbs[3].RenderPS(ps31)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps32 := PhiState{General: ps.General}
						ps32.OverlayValues = make([]JITValueDesc, 31)
						ps32.OverlayValues[1] = d1
						ps32.OverlayValues[2] = d2
						ps32.OverlayValues[3] = d3
						ps32.OverlayValues[4] = d4
						ps32.OverlayValues[5] = d5
						ps32.OverlayValues[6] = d6
						ps32.OverlayValues[25] = d25
						ps32.OverlayValues[26] = d26
						ps32.OverlayValues[27] = d27
						ps32.OverlayValues[28] = d28
						ps32.OverlayValues[29] = d29
						ps32.OverlayValues[30] = d30
						ps32.PhiValues = make([]JITValueDesc, 1)
						d33 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}
						ps32.PhiValues[0] = d33
						return bbs[4].RenderPS(ps32)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitJump(d30.Condition, lbl4)
					ctx.EmitJmp(lbl11)
					snap34 := d1
					snap35 := d2
					snap36 := d3
					snap37 := d4
					snap38 := d5
					snap39 := d6
					snap40 := d25
					snap41 := d26
					snap42 := d27
					snap43 := d28
					snap44 := d29
					snap45 := d30
					snap46 := d33
					alloc47 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc47)
					d1 = snap34
					d2 = snap35
					d3 = snap36
					d4 = snap37
					d5 = snap38
					d6 = snap39
					d25 = snap40
					d26 = snap41
					d27 = snap42
					d28 = snap43
					d29 = snap44
					d30 = snap45
					d33 = snap46
					ctx.MarkLabel(lbl11)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc47)
					d1 = snap34
					d2 = snap35
					d3 = snap36
					d4 = snap37
					d5 = snap38
					d6 = snap39
					d25 = snap40
					d26 = snap41
					d27 = snap42
					d28 = snap43
					d29 = snap44
					d30 = snap45
					d33 = snap46
					ps48 := PhiState{General: true}
					ps48.OverlayValues = make([]JITValueDesc, 34)
					ps48.OverlayValues[1] = d1
					ps48.OverlayValues[2] = d2
					ps48.OverlayValues[3] = d3
					ps48.OverlayValues[4] = d4
					ps48.OverlayValues[5] = d5
					ps48.OverlayValues[6] = d6
					ps48.OverlayValues[25] = d25
					ps48.OverlayValues[26] = d26
					ps48.OverlayValues[27] = d27
					ps48.OverlayValues[28] = d28
					ps48.OverlayValues[29] = d29
					ps48.OverlayValues[30] = d30
					ps48.OverlayValues[33] = d33
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 34)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[6] = d6
					ps49.OverlayValues[25] = d25
					ps49.OverlayValues[26] = d26
					ps49.OverlayValues[27] = d27
					ps49.OverlayValues[28] = d28
					ps49.OverlayValues[29] = d29
					ps49.OverlayValues[30] = d30
					ps49.OverlayValues[33] = d33
					ps49.PhiValues = make([]JITValueDesc, 1)
					d50 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("UTC")}
					ps49.PhiValues[0] = d50
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d25
					snap58 := d26
					snap59 := d27
					snap60 := d28
					snap61 := d29
					snap62 := d30
					snap63 := d33
					snap64 := d50
					alloc65 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps49)
					}
					ctx.RestoreAllocState(alloc65)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d25 = snap57
					d26 = snap58
					d27 = snap59
					d28 = snap60
					d29 = snap61
					d30 = snap62
					d33 = snap63
					d50 = snap64
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps48)
					}
					return result
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					ctx.ReclaimUntrackedRegs()
					d66 = args[2]
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
					if ps.General {
						ctx.SyncDesc(&d67)
						if d67.Loc == LocReg {
							ctx.ProtectReg(d67.Reg)
						} else if d67.Loc == LocRegPair {
							ctx.ProtectReg(d67.Reg)
							ctx.ProtectReg(d67.Reg2)
						}
						d69 = d67
						if d69.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d69)
						if d69.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d69, int32(bbs[4].PhiBase)+int32(0), 2)
						} else if d69.Loc == LocInputPair {
							ctx.EnsureDesc(&d69)
							ctx.EmitStoreScmerToStack(d69, int32(bbs[4].PhiBase)+int32(0))
						} else if d69.Loc == LocRegPair || d69.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d69, int32(bbs[4].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d69)
							ctx.EmitStoreToStack(d69, int32(bbs[4].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
						}
						if d67.Loc == LocReg {
							ctx.UnprotectReg(d67.Reg)
						} else if d67.Loc == LocRegPair {
							ctx.UnprotectReg(d67.Reg)
							ctx.UnprotectReg(d67.Reg2)
						}
					}
					ps70 := PhiState{General: ps.General}
					ps70.OverlayValues = make([]JITValueDesc, 70)
					ps70.OverlayValues[1] = d1
					ps70.OverlayValues[2] = d2
					ps70.OverlayValues[3] = d3
					ps70.OverlayValues[4] = d4
					ps70.OverlayValues[5] = d5
					ps70.OverlayValues[6] = d6
					ps70.OverlayValues[25] = d25
					ps70.OverlayValues[26] = d26
					ps70.OverlayValues[27] = d27
					ps70.OverlayValues[28] = d28
					ps70.OverlayValues[29] = d29
					ps70.OverlayValues[30] = d30
					ps70.OverlayValues[33] = d33
					ps70.OverlayValues[50] = d50
					ps70.OverlayValues[66] = d66
					ps70.OverlayValues[67] = d67
					ps70.OverlayValues[68] = d68
					ps70.OverlayValues[69] = d69
					ps70.PhiValues = make([]JITValueDesc, 1)
					d71 = d67
					ps70.PhiValues[0] = d71
					if ps70.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps70)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d72 := ps.PhiValues[0]
							ctx.EnsureDesc(&d72)
							ctx.EmitStoreScmerToStack(d72, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					callResults73 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d1}, []uint8{1, 2}, []uint8{1, 3})
					d74 = callResults73[0]
					_ = d74
					d75 = callResults73[1]
					_ = d75
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d74)
					ctx.EnsureDesc(&d75)
					var d76 JITValueDesc
					if d75.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d75.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d75)
						if d75.Loc != LocReg && d75.Loc != LocRegPair && d75.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d75.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
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
							ps78.OverlayValues[25] = d25
							ps78.OverlayValues[26] = d26
							ps78.OverlayValues[27] = d27
							ps78.OverlayValues[28] = d28
							ps78.OverlayValues[29] = d29
							ps78.OverlayValues[30] = d30
							ps78.OverlayValues[33] = d33
							ps78.OverlayValues[50] = d50
							ps78.OverlayValues[66] = d66
							ps78.OverlayValues[67] = d67
							ps78.OverlayValues[68] = d68
							ps78.OverlayValues[69] = d69
							ps78.OverlayValues[71] = d71
							ps78.OverlayValues[72] = d72
							ps78.OverlayValues[74] = d74
							ps78.OverlayValues[75] = d75
							ps78.OverlayValues[76] = d76
							ps78.OverlayValues[77] = d77
							return bbs[5].RenderPS(ps78)
						}
						if ps.General {
							ctx.SyncDesc(&d74)
							if d74.Loc == LocReg {
								ctx.ProtectReg(d74.Reg)
							} else if d74.Loc == LocRegPair {
								ctx.ProtectReg(d74.Reg)
								ctx.ProtectReg(d74.Reg2)
							}
							d79 = d74
							if d79.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d79)
							ctx.EmitStoreToStack(d79, int32(bbs[6].PhiBase)+int32(0))
							if d74.Loc == LocReg {
								ctx.UnprotectReg(d74.Reg)
							} else if d74.Loc == LocRegPair {
								ctx.UnprotectReg(d74.Reg)
								ctx.UnprotectReg(d74.Reg2)
							}
						}
						ps80 := PhiState{General: ps.General}
						ps80.OverlayValues = make([]JITValueDesc, 80)
						ps80.OverlayValues[1] = d1
						ps80.OverlayValues[2] = d2
						ps80.OverlayValues[3] = d3
						ps80.OverlayValues[4] = d4
						ps80.OverlayValues[5] = d5
						ps80.OverlayValues[6] = d6
						ps80.OverlayValues[25] = d25
						ps80.OverlayValues[26] = d26
						ps80.OverlayValues[27] = d27
						ps80.OverlayValues[28] = d28
						ps80.OverlayValues[29] = d29
						ps80.OverlayValues[30] = d30
						ps80.OverlayValues[33] = d33
						ps80.OverlayValues[50] = d50
						ps80.OverlayValues[66] = d66
						ps80.OverlayValues[67] = d67
						ps80.OverlayValues[68] = d68
						ps80.OverlayValues[69] = d69
						ps80.OverlayValues[71] = d71
						ps80.OverlayValues[72] = d72
						ps80.OverlayValues[74] = d74
						ps80.OverlayValues[75] = d75
						ps80.OverlayValues[76] = d76
						ps80.OverlayValues[77] = d77
						ps80.OverlayValues[79] = d79
						ps80.PhiValues = make([]JITValueDesc, 1)
						d81 = d74
						ps80.PhiValues[0] = d81
						return bbs[6].RenderPS(ps80)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d82 := ps.PhiValues[0]
							ctx.EnsureDesc(&d82)
							ctx.EmitStoreScmerToStack(d82, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d77.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					ctx.EmitJmp(lbl12)
					snap83 := d1
					snap84 := d2
					snap85 := d3
					snap86 := d4
					snap87 := d5
					snap88 := d6
					snap89 := d25
					snap90 := d26
					snap91 := d27
					snap92 := d28
					snap93 := d29
					snap94 := d30
					snap95 := d33
					snap96 := d50
					snap97 := d66
					snap98 := d67
					snap99 := d68
					snap100 := d69
					snap101 := d71
					snap102 := d72
					snap103 := d74
					snap104 := d75
					snap105 := d76
					snap106 := d77
					snap107 := d79
					snap108 := d81
					snap109 := d82
					alloc110 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc110)
					d1 = snap83
					d2 = snap84
					d3 = snap85
					d4 = snap86
					d5 = snap87
					d6 = snap88
					d25 = snap89
					d26 = snap90
					d27 = snap91
					d28 = snap92
					d29 = snap93
					d30 = snap94
					d33 = snap95
					d50 = snap96
					d66 = snap97
					d67 = snap98
					d68 = snap99
					d69 = snap100
					d71 = snap101
					d72 = snap102
					d74 = snap103
					d75 = snap104
					d76 = snap105
					d77 = snap106
					d79 = snap107
					d81 = snap108
					d82 = snap109
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d74)
					if d74.Loc == LocReg {
						ctx.ProtectReg(d74.Reg)
					} else if d74.Loc == LocRegPair {
						ctx.ProtectReg(d74.Reg)
						ctx.ProtectReg(d74.Reg2)
					}
					d111 = d74
					if d111.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d111)
					ctx.EmitStoreToStack(d111, int32(bbs[6].PhiBase)+int32(0))
					if d74.Loc == LocReg {
						ctx.UnprotectReg(d74.Reg)
					} else if d74.Loc == LocRegPair {
						ctx.UnprotectReg(d74.Reg)
						ctx.UnprotectReg(d74.Reg2)
					}
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc110)
					d1 = snap83
					d2 = snap84
					d3 = snap85
					d4 = snap86
					d5 = snap87
					d6 = snap88
					d25 = snap89
					d26 = snap90
					d27 = snap91
					d28 = snap92
					d29 = snap93
					d30 = snap94
					d33 = snap95
					d50 = snap96
					d66 = snap97
					d67 = snap98
					d68 = snap99
					d69 = snap100
					d71 = snap101
					d72 = snap102
					d74 = snap103
					d75 = snap104
					d76 = snap105
					d77 = snap106
					d79 = snap107
					d81 = snap108
					d82 = snap109
					ps112 := PhiState{General: true}
					ps112.OverlayValues = make([]JITValueDesc, 112)
					ps112.OverlayValues[1] = d1
					ps112.OverlayValues[2] = d2
					ps112.OverlayValues[3] = d3
					ps112.OverlayValues[4] = d4
					ps112.OverlayValues[5] = d5
					ps112.OverlayValues[6] = d6
					ps112.OverlayValues[25] = d25
					ps112.OverlayValues[26] = d26
					ps112.OverlayValues[27] = d27
					ps112.OverlayValues[28] = d28
					ps112.OverlayValues[29] = d29
					ps112.OverlayValues[30] = d30
					ps112.OverlayValues[33] = d33
					ps112.OverlayValues[50] = d50
					ps112.OverlayValues[66] = d66
					ps112.OverlayValues[67] = d67
					ps112.OverlayValues[68] = d68
					ps112.OverlayValues[69] = d69
					ps112.OverlayValues[71] = d71
					ps112.OverlayValues[72] = d72
					ps112.OverlayValues[74] = d74
					ps112.OverlayValues[75] = d75
					ps112.OverlayValues[76] = d76
					ps112.OverlayValues[77] = d77
					ps112.OverlayValues[79] = d79
					ps112.OverlayValues[81] = d81
					ps112.OverlayValues[82] = d82
					ps112.OverlayValues[111] = d111
					ps113 := PhiState{General: true}
					ps113.OverlayValues = make([]JITValueDesc, 112)
					ps113.OverlayValues[1] = d1
					ps113.OverlayValues[2] = d2
					ps113.OverlayValues[3] = d3
					ps113.OverlayValues[4] = d4
					ps113.OverlayValues[5] = d5
					ps113.OverlayValues[6] = d6
					ps113.OverlayValues[25] = d25
					ps113.OverlayValues[26] = d26
					ps113.OverlayValues[27] = d27
					ps113.OverlayValues[28] = d28
					ps113.OverlayValues[29] = d29
					ps113.OverlayValues[30] = d30
					ps113.OverlayValues[33] = d33
					ps113.OverlayValues[50] = d50
					ps113.OverlayValues[66] = d66
					ps113.OverlayValues[67] = d67
					ps113.OverlayValues[68] = d68
					ps113.OverlayValues[69] = d69
					ps113.OverlayValues[71] = d71
					ps113.OverlayValues[72] = d72
					ps113.OverlayValues[74] = d74
					ps113.OverlayValues[75] = d75
					ps113.OverlayValues[76] = d76
					ps113.OverlayValues[77] = d77
					ps113.OverlayValues[79] = d79
					ps113.OverlayValues[81] = d81
					ps113.OverlayValues[82] = d82
					ps113.OverlayValues[111] = d111
					ps113.PhiValues = make([]JITValueDesc, 1)
					d114 = d74
					ps113.PhiValues[0] = d114
					snap115 := d1
					snap116 := d2
					snap117 := d3
					snap118 := d4
					snap119 := d5
					snap120 := d6
					snap121 := d25
					snap122 := d26
					snap123 := d27
					snap124 := d28
					snap125 := d29
					snap126 := d30
					snap127 := d33
					snap128 := d50
					snap129 := d66
					snap130 := d67
					snap131 := d68
					snap132 := d69
					snap133 := d71
					snap134 := d72
					snap135 := d74
					snap136 := d75
					snap137 := d76
					snap138 := d77
					snap139 := d79
					snap140 := d81
					snap141 := d82
					snap142 := d111
					snap143 := d114
					alloc144 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps113)
					}
					ctx.RestoreAllocState(alloc144)
					d1 = snap115
					d2 = snap116
					d3 = snap117
					d4 = snap118
					d5 = snap119
					d6 = snap120
					d25 = snap121
					d26 = snap122
					d27 = snap123
					d28 = snap124
					d29 = snap125
					d30 = snap126
					d33 = snap127
					d50 = snap128
					d66 = snap129
					d67 = snap130
					d68 = snap131
					d69 = snap132
					d71 = snap133
					d72 = snap134
					d74 = snap135
					d75 = snap136
					d76 = snap137
					d77 = snap138
					d79 = snap139
					d81 = snap140
					d82 = snap141
					d111 = snap142
					d114 = snap143
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps112)
					}
					return result
					ctx.FreeDesc(&d76)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					ctx.ReclaimUntrackedRegs()
					d145 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.StabilizeDescForControlFlow(&d145)
					if ps.General {
						ctx.SyncDesc(&d145)
						if d145.Loc == LocReg {
							ctx.ProtectReg(d145.Reg)
						} else if d145.Loc == LocRegPair {
							ctx.ProtectReg(d145.Reg)
							ctx.ProtectReg(d145.Reg2)
						}
						d146 = d145
						if d146.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d146)
						ctx.EmitStoreToStack(d146, int32(bbs[6].PhiBase)+int32(0))
						if d145.Loc == LocReg {
							ctx.UnprotectReg(d145.Reg)
						} else if d145.Loc == LocRegPair {
							ctx.UnprotectReg(d145.Reg)
							ctx.UnprotectReg(d145.Reg2)
						}
					}
					ps147 := PhiState{General: ps.General}
					ps147.OverlayValues = make([]JITValueDesc, 147)
					ps147.OverlayValues[1] = d1
					ps147.OverlayValues[2] = d2
					ps147.OverlayValues[3] = d3
					ps147.OverlayValues[4] = d4
					ps147.OverlayValues[5] = d5
					ps147.OverlayValues[6] = d6
					ps147.OverlayValues[25] = d25
					ps147.OverlayValues[26] = d26
					ps147.OverlayValues[27] = d27
					ps147.OverlayValues[28] = d28
					ps147.OverlayValues[29] = d29
					ps147.OverlayValues[30] = d30
					ps147.OverlayValues[33] = d33
					ps147.OverlayValues[50] = d50
					ps147.OverlayValues[66] = d66
					ps147.OverlayValues[67] = d67
					ps147.OverlayValues[68] = d68
					ps147.OverlayValues[69] = d69
					ps147.OverlayValues[71] = d71
					ps147.OverlayValues[72] = d72
					ps147.OverlayValues[74] = d74
					ps147.OverlayValues[75] = d75
					ps147.OverlayValues[76] = d76
					ps147.OverlayValues[77] = d77
					ps147.OverlayValues[79] = d79
					ps147.OverlayValues[81] = d81
					ps147.OverlayValues[82] = d82
					ps147.OverlayValues[111] = d111
					ps147.OverlayValues[114] = d114
					ps147.OverlayValues[145] = d145
					ps147.OverlayValues[146] = d146
					ps147.PhiValues = make([]JITValueDesc, 1)
					d148 = d145
					ps147.PhiValues[0] = d148
					if ps147.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps147)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d149 := ps.PhiValues[0]
							ctx.EnsureDesc(&d149)
							ctx.EmitStoreToStack(d149, int32(bbs[6].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					d150 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d150)
					var d151 JITValueDesc
					if d150.Loc == LocImm {
						d151 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d150.Imm.Int() > 1)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d150.Reg, 1)
						d151 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreater}
						ctx.BindReg(r2, &d151)
					}
					ctx.FreeDesc(&d150)
					d152 = d151
					ctx.EnsureDesc(&d152)
					if d152.Loc != LocImm && d152.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d152.Loc == LocImm {
						if d152.Imm.Bool() {
							if ps.General {
							}
							ps153 := PhiState{General: ps.General}
							ps153.OverlayValues = make([]JITValueDesc, 153)
							ps153.OverlayValues[1] = d1
							ps153.OverlayValues[2] = d2
							ps153.OverlayValues[3] = d3
							ps153.OverlayValues[4] = d4
							ps153.OverlayValues[5] = d5
							ps153.OverlayValues[6] = d6
							ps153.OverlayValues[25] = d25
							ps153.OverlayValues[26] = d26
							ps153.OverlayValues[27] = d27
							ps153.OverlayValues[28] = d28
							ps153.OverlayValues[29] = d29
							ps153.OverlayValues[30] = d30
							ps153.OverlayValues[33] = d33
							ps153.OverlayValues[50] = d50
							ps153.OverlayValues[66] = d66
							ps153.OverlayValues[67] = d67
							ps153.OverlayValues[68] = d68
							ps153.OverlayValues[69] = d69
							ps153.OverlayValues[71] = d71
							ps153.OverlayValues[72] = d72
							ps153.OverlayValues[74] = d74
							ps153.OverlayValues[75] = d75
							ps153.OverlayValues[76] = d76
							ps153.OverlayValues[77] = d77
							ps153.OverlayValues[79] = d79
							ps153.OverlayValues[81] = d81
							ps153.OverlayValues[82] = d82
							ps153.OverlayValues[111] = d111
							ps153.OverlayValues[114] = d114
							ps153.OverlayValues[145] = d145
							ps153.OverlayValues[146] = d146
							ps153.OverlayValues[148] = d148
							ps153.OverlayValues[149] = d149
							ps153.OverlayValues[150] = d150
							ps153.OverlayValues[151] = d151
							ps153.OverlayValues[152] = d152
							return bbs[9].RenderPS(ps153)
						}
						if ps.General {
						}
						ps154 := PhiState{General: ps.General}
						ps154.OverlayValues = make([]JITValueDesc, 153)
						ps154.OverlayValues[1] = d1
						ps154.OverlayValues[2] = d2
						ps154.OverlayValues[3] = d3
						ps154.OverlayValues[4] = d4
						ps154.OverlayValues[5] = d5
						ps154.OverlayValues[6] = d6
						ps154.OverlayValues[25] = d25
						ps154.OverlayValues[26] = d26
						ps154.OverlayValues[27] = d27
						ps154.OverlayValues[28] = d28
						ps154.OverlayValues[29] = d29
						ps154.OverlayValues[30] = d30
						ps154.OverlayValues[33] = d33
						ps154.OverlayValues[50] = d50
						ps154.OverlayValues[66] = d66
						ps154.OverlayValues[67] = d67
						ps154.OverlayValues[68] = d68
						ps154.OverlayValues[69] = d69
						ps154.OverlayValues[71] = d71
						ps154.OverlayValues[72] = d72
						ps154.OverlayValues[74] = d74
						ps154.OverlayValues[75] = d75
						ps154.OverlayValues[76] = d76
						ps154.OverlayValues[77] = d77
						ps154.OverlayValues[79] = d79
						ps154.OverlayValues[81] = d81
						ps154.OverlayValues[82] = d82
						ps154.OverlayValues[111] = d111
						ps154.OverlayValues[114] = d114
						ps154.OverlayValues[145] = d145
						ps154.OverlayValues[146] = d146
						ps154.OverlayValues[148] = d148
						ps154.OverlayValues[149] = d149
						ps154.OverlayValues[150] = d150
						ps154.OverlayValues[151] = d151
						ps154.OverlayValues[152] = d152
						return bbs[8].RenderPS(ps154)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d155 := ps.PhiValues[0]
							ctx.EnsureDesc(&d155)
							ctx.EmitStoreToStack(d155, int32(bbs[6].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d152.Condition, lbl10)
					snap156 := d1
					snap157 := d2
					snap158 := d3
					snap159 := d4
					snap160 := d5
					snap161 := d6
					snap162 := d25
					snap163 := d26
					snap164 := d27
					snap165 := d28
					snap166 := d29
					snap167 := d30
					snap168 := d33
					snap169 := d50
					snap170 := d66
					snap171 := d67
					snap172 := d68
					snap173 := d69
					snap174 := d71
					snap175 := d72
					snap176 := d74
					snap177 := d75
					snap178 := d76
					snap179 := d77
					snap180 := d79
					snap181 := d81
					snap182 := d82
					snap183 := d111
					snap184 := d114
					snap185 := d145
					snap186 := d146
					snap187 := d148
					snap188 := d149
					snap189 := d150
					snap190 := d151
					snap191 := d152
					snap192 := d155
					alloc193 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc193)
					d1 = snap156
					d2 = snap157
					d3 = snap158
					d4 = snap159
					d5 = snap160
					d6 = snap161
					d25 = snap162
					d26 = snap163
					d27 = snap164
					d28 = snap165
					d29 = snap166
					d30 = snap167
					d33 = snap168
					d50 = snap169
					d66 = snap170
					d67 = snap171
					d68 = snap172
					d69 = snap173
					d71 = snap174
					d72 = snap175
					d74 = snap176
					d75 = snap177
					d76 = snap178
					d77 = snap179
					d79 = snap180
					d81 = snap181
					d82 = snap182
					d111 = snap183
					d114 = snap184
					d145 = snap185
					d146 = snap186
					d148 = snap187
					d149 = snap188
					d150 = snap189
					d151 = snap190
					d152 = snap191
					d155 = snap192
					ctx.RestoreAllocState(alloc193)
					d1 = snap156
					d2 = snap157
					d3 = snap158
					d4 = snap159
					d5 = snap160
					d6 = snap161
					d25 = snap162
					d26 = snap163
					d27 = snap164
					d28 = snap165
					d29 = snap166
					d30 = snap167
					d33 = snap168
					d50 = snap169
					d66 = snap170
					d67 = snap171
					d68 = snap172
					d69 = snap173
					d71 = snap174
					d72 = snap175
					d74 = snap176
					d75 = snap177
					d76 = snap178
					d77 = snap179
					d79 = snap180
					d81 = snap181
					d82 = snap182
					d111 = snap183
					d114 = snap184
					d145 = snap185
					d146 = snap186
					d148 = snap187
					d149 = snap188
					d150 = snap189
					d151 = snap190
					d152 = snap191
					d155 = snap192
					ps194 := PhiState{General: true}
					ps194.OverlayValues = make([]JITValueDesc, 156)
					ps194.OverlayValues[1] = d1
					ps194.OverlayValues[2] = d2
					ps194.OverlayValues[3] = d3
					ps194.OverlayValues[4] = d4
					ps194.OverlayValues[5] = d5
					ps194.OverlayValues[6] = d6
					ps194.OverlayValues[25] = d25
					ps194.OverlayValues[26] = d26
					ps194.OverlayValues[27] = d27
					ps194.OverlayValues[28] = d28
					ps194.OverlayValues[29] = d29
					ps194.OverlayValues[30] = d30
					ps194.OverlayValues[33] = d33
					ps194.OverlayValues[50] = d50
					ps194.OverlayValues[66] = d66
					ps194.OverlayValues[67] = d67
					ps194.OverlayValues[68] = d68
					ps194.OverlayValues[69] = d69
					ps194.OverlayValues[71] = d71
					ps194.OverlayValues[72] = d72
					ps194.OverlayValues[74] = d74
					ps194.OverlayValues[75] = d75
					ps194.OverlayValues[76] = d76
					ps194.OverlayValues[77] = d77
					ps194.OverlayValues[79] = d79
					ps194.OverlayValues[81] = d81
					ps194.OverlayValues[82] = d82
					ps194.OverlayValues[111] = d111
					ps194.OverlayValues[114] = d114
					ps194.OverlayValues[145] = d145
					ps194.OverlayValues[146] = d146
					ps194.OverlayValues[148] = d148
					ps194.OverlayValues[149] = d149
					ps194.OverlayValues[150] = d150
					ps194.OverlayValues[151] = d151
					ps194.OverlayValues[152] = d152
					ps194.OverlayValues[155] = d155
					ps195 := PhiState{General: true}
					ps195.OverlayValues = make([]JITValueDesc, 156)
					ps195.OverlayValues[1] = d1
					ps195.OverlayValues[2] = d2
					ps195.OverlayValues[3] = d3
					ps195.OverlayValues[4] = d4
					ps195.OverlayValues[5] = d5
					ps195.OverlayValues[6] = d6
					ps195.OverlayValues[25] = d25
					ps195.OverlayValues[26] = d26
					ps195.OverlayValues[27] = d27
					ps195.OverlayValues[28] = d28
					ps195.OverlayValues[29] = d29
					ps195.OverlayValues[30] = d30
					ps195.OverlayValues[33] = d33
					ps195.OverlayValues[50] = d50
					ps195.OverlayValues[66] = d66
					ps195.OverlayValues[67] = d67
					ps195.OverlayValues[68] = d68
					ps195.OverlayValues[69] = d69
					ps195.OverlayValues[71] = d71
					ps195.OverlayValues[72] = d72
					ps195.OverlayValues[74] = d74
					ps195.OverlayValues[75] = d75
					ps195.OverlayValues[76] = d76
					ps195.OverlayValues[77] = d77
					ps195.OverlayValues[79] = d79
					ps195.OverlayValues[81] = d81
					ps195.OverlayValues[82] = d82
					ps195.OverlayValues[111] = d111
					ps195.OverlayValues[114] = d114
					ps195.OverlayValues[145] = d145
					ps195.OverlayValues[146] = d146
					ps195.OverlayValues[148] = d148
					ps195.OverlayValues[149] = d149
					ps195.OverlayValues[150] = d150
					ps195.OverlayValues[151] = d151
					ps195.OverlayValues[152] = d152
					ps195.OverlayValues[155] = d155
					snap196 := d1
					snap197 := d2
					snap198 := d3
					snap199 := d4
					snap200 := d5
					snap201 := d6
					snap202 := d25
					snap203 := d26
					snap204 := d27
					snap205 := d28
					snap206 := d29
					snap207 := d30
					snap208 := d33
					snap209 := d50
					snap210 := d66
					snap211 := d67
					snap212 := d68
					snap213 := d69
					snap214 := d71
					snap215 := d72
					snap216 := d74
					snap217 := d75
					snap218 := d76
					snap219 := d77
					snap220 := d79
					snap221 := d81
					snap222 := d82
					snap223 := d111
					snap224 := d114
					snap225 := d145
					snap226 := d146
					snap227 := d148
					snap228 := d149
					snap229 := d150
					snap230 := d151
					snap231 := d152
					snap232 := d155
					alloc233 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps195)
					}
					ctx.RestoreAllocState(alloc233)
					d1 = snap196
					d2 = snap197
					d3 = snap198
					d4 = snap199
					d5 = snap200
					d6 = snap201
					d25 = snap202
					d26 = snap203
					d27 = snap204
					d28 = snap205
					d29 = snap206
					d30 = snap207
					d33 = snap208
					d50 = snap209
					d66 = snap210
					d67 = snap211
					d68 = snap212
					d69 = snap213
					d71 = snap214
					d72 = snap215
					d74 = snap216
					d75 = snap217
					d76 = snap218
					d77 = snap219
					d79 = snap220
					d81 = snap221
					d82 = snap222
					d111 = snap223
					d114 = snap224
					d145 = snap225
					d146 = snap226
					d148 = snap227
					d149 = snap228
					d150 = snap229
					d151 = snap230
					d152 = snap231
					d155 = snap232
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps194)
					}
					return result
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
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
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d234 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d234.Loc == LocRegPair || d234.Loc == LocStackPair || d234.Loc == LocRegTriple || d234.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d27)
					ctx.SyncDesc(&d234)
					d235 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d27, d234}, 3)
					d235.NoHeapPointer = false
					ctx.BindReg(d235.Reg, &d235)
					ctx.BindReg(d235.Reg2, &d235)
					ctx.BindReg(d235.Reg3, &d235)
					ctx.FreeDesc(&d234)
					ctx.EnsureDesc(&d235)
					ctx.EnsureDesc(&d235)
					ctx.EnsureDesc(&d235)
					if d235.Loc != LocRegTriple && d235.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d235)
					ctx.SyncDesc(&d2)
					d236 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d235, d2}, 3)
					d236.NoHeapPointer = false
					ctx.BindReg(d236.Reg, &d236)
					ctx.BindReg(d236.Reg2, &d236)
					ctx.BindReg(d236.Reg3, &d236)
					ctx.FreeDesc(&d235)
					d237 = args[1]
					d237.ID = 0
					d239 = d237
					ctx.SyncDesc(&d239)
					if d239.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d239.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d239.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d239 = tmpScalar
					}
					d239 = JITPrepareScmerGoArg(ctx, d239)
					if d239.Loc != LocRegPair && d239.Loc != LocStackPair && d239.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d238 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d239}, 2)
					ctx.FreeDesc(&d237)
					ctx.EnsureDesc(&d236)
					ctx.EnsureDesc(&d236)
					ctx.EnsureDesc(&d236)
					if d236.Loc != LocRegTriple && d236.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (formatDateMySQL arg0)")
					}
					ctx.EnsureDesc(&d238)
					ctx.EnsureDesc(&d238)
					ctx.EnsureDesc(&d238)
					if d238.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d238.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d238.Imm)
						ptrWord, _ := d238.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d238.Imm.String())))
						d238 = tmpPair
					} else if d238.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d238.Type, Reg: ctx.AllocRegExcept(d238.Reg), Reg2: ctx.AllocRegExcept(d238.Reg)}
						switch d238.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d238)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d238)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d238)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d238)
						d238 = tmpPair
					}
					if d238.Loc != LocRegPair && d238.Loc != LocStackPair && d238.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (formatDateMySQL arg1)")
					}
					ctx.SyncDesc(&d236)
					ctx.SyncDesc(&d238)
					d240 = ctx.EmitGoCallScalar(GoFuncAddr(formatDateMySQL), []JITValueDesc{d236, d238}, 2)
					d240.NoHeapPointer = false
					ctx.BindReg(d240.Reg, &d240)
					ctx.BindReg(d240.Reg2, &d240)
					ctx.FreeDesc(&d236)
					ctx.EnsureDesc(&d240)
					d241 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d240}, 2)
					ctx.EmitMovPairToResult(&d241, &result)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
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
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d27)
					d242 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d27}, 2)
					d242.NoHeapPointer = false
					ctx.BindReg(d242.Reg, &d242)
					ctx.BindReg(d242.Reg2, &d242)
					ctx.SyncDesc(&d242)
					if d242.Loc == LocRegPair || d242.Loc == LocStackPair || d242.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d242, &result)
						result.Type = d242.Type
					} else {
						switch d242.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d242)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d242)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d242)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d242, &result)
							result.Type = d242.Type
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
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
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					ctx.ReclaimUntrackedRegs()
					d243 = args[1]
					d243.ID = 0
					d245 = d243
					d245.ID = 0
					d244 = ctx.EmitTagEqualsBorrowed(&d245, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d243)
					d246 = d244
					ctx.EnsureDesc(&d246)
					if d246.Loc != LocImm && d246.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d246.Loc == LocImm {
						if d246.Imm.Bool() {
							if ps.General {
							}
							ps247 := PhiState{General: ps.General}
							ps247.OverlayValues = make([]JITValueDesc, 247)
							ps247.OverlayValues[1] = d1
							ps247.OverlayValues[2] = d2
							ps247.OverlayValues[3] = d3
							ps247.OverlayValues[4] = d4
							ps247.OverlayValues[5] = d5
							ps247.OverlayValues[6] = d6
							ps247.OverlayValues[25] = d25
							ps247.OverlayValues[26] = d26
							ps247.OverlayValues[27] = d27
							ps247.OverlayValues[28] = d28
							ps247.OverlayValues[29] = d29
							ps247.OverlayValues[30] = d30
							ps247.OverlayValues[33] = d33
							ps247.OverlayValues[50] = d50
							ps247.OverlayValues[66] = d66
							ps247.OverlayValues[67] = d67
							ps247.OverlayValues[68] = d68
							ps247.OverlayValues[69] = d69
							ps247.OverlayValues[71] = d71
							ps247.OverlayValues[72] = d72
							ps247.OverlayValues[74] = d74
							ps247.OverlayValues[75] = d75
							ps247.OverlayValues[76] = d76
							ps247.OverlayValues[77] = d77
							ps247.OverlayValues[79] = d79
							ps247.OverlayValues[81] = d81
							ps247.OverlayValues[82] = d82
							ps247.OverlayValues[111] = d111
							ps247.OverlayValues[114] = d114
							ps247.OverlayValues[145] = d145
							ps247.OverlayValues[146] = d146
							ps247.OverlayValues[148] = d148
							ps247.OverlayValues[149] = d149
							ps247.OverlayValues[150] = d150
							ps247.OverlayValues[151] = d151
							ps247.OverlayValues[152] = d152
							ps247.OverlayValues[155] = d155
							ps247.OverlayValues[234] = d234
							ps247.OverlayValues[235] = d235
							ps247.OverlayValues[236] = d236
							ps247.OverlayValues[237] = d237
							ps247.OverlayValues[238] = d238
							ps247.OverlayValues[239] = d239
							ps247.OverlayValues[240] = d240
							ps247.OverlayValues[241] = d241
							ps247.OverlayValues[242] = d242
							ps247.OverlayValues[243] = d243
							ps247.OverlayValues[244] = d244
							ps247.OverlayValues[245] = d245
							ps247.OverlayValues[246] = d246
							return bbs[8].RenderPS(ps247)
						}
						if ps.General {
						}
						ps248 := PhiState{General: ps.General}
						ps248.OverlayValues = make([]JITValueDesc, 247)
						ps248.OverlayValues[1] = d1
						ps248.OverlayValues[2] = d2
						ps248.OverlayValues[3] = d3
						ps248.OverlayValues[4] = d4
						ps248.OverlayValues[5] = d5
						ps248.OverlayValues[6] = d6
						ps248.OverlayValues[25] = d25
						ps248.OverlayValues[26] = d26
						ps248.OverlayValues[27] = d27
						ps248.OverlayValues[28] = d28
						ps248.OverlayValues[29] = d29
						ps248.OverlayValues[30] = d30
						ps248.OverlayValues[33] = d33
						ps248.OverlayValues[50] = d50
						ps248.OverlayValues[66] = d66
						ps248.OverlayValues[67] = d67
						ps248.OverlayValues[68] = d68
						ps248.OverlayValues[69] = d69
						ps248.OverlayValues[71] = d71
						ps248.OverlayValues[72] = d72
						ps248.OverlayValues[74] = d74
						ps248.OverlayValues[75] = d75
						ps248.OverlayValues[76] = d76
						ps248.OverlayValues[77] = d77
						ps248.OverlayValues[79] = d79
						ps248.OverlayValues[81] = d81
						ps248.OverlayValues[82] = d82
						ps248.OverlayValues[111] = d111
						ps248.OverlayValues[114] = d114
						ps248.OverlayValues[145] = d145
						ps248.OverlayValues[146] = d146
						ps248.OverlayValues[148] = d148
						ps248.OverlayValues[149] = d149
						ps248.OverlayValues[150] = d150
						ps248.OverlayValues[151] = d151
						ps248.OverlayValues[152] = d152
						ps248.OverlayValues[155] = d155
						ps248.OverlayValues[234] = d234
						ps248.OverlayValues[235] = d235
						ps248.OverlayValues[236] = d236
						ps248.OverlayValues[237] = d237
						ps248.OverlayValues[238] = d238
						ps248.OverlayValues[239] = d239
						ps248.OverlayValues[240] = d240
						ps248.OverlayValues[241] = d241
						ps248.OverlayValues[242] = d242
						ps248.OverlayValues[243] = d243
						ps248.OverlayValues[244] = d244
						ps248.OverlayValues[245] = d245
						ps248.OverlayValues[246] = d246
						return bbs[7].RenderPS(ps248)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d246.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					snap249 := d1
					snap250 := d2
					snap251 := d3
					snap252 := d4
					snap253 := d5
					snap254 := d6
					snap255 := d25
					snap256 := d26
					snap257 := d27
					snap258 := d28
					snap259 := d29
					snap260 := d30
					snap261 := d33
					snap262 := d50
					snap263 := d66
					snap264 := d67
					snap265 := d68
					snap266 := d69
					snap267 := d71
					snap268 := d72
					snap269 := d74
					snap270 := d75
					snap271 := d76
					snap272 := d77
					snap273 := d79
					snap274 := d81
					snap275 := d82
					snap276 := d111
					snap277 := d114
					snap278 := d145
					snap279 := d146
					snap280 := d148
					snap281 := d149
					snap282 := d150
					snap283 := d151
					snap284 := d152
					snap285 := d155
					snap286 := d234
					snap287 := d235
					snap288 := d236
					snap289 := d237
					snap290 := d238
					snap291 := d239
					snap292 := d240
					snap293 := d241
					snap294 := d242
					snap295 := d243
					snap296 := d244
					snap297 := d245
					snap298 := d246
					alloc299 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc299)
					d1 = snap249
					d2 = snap250
					d3 = snap251
					d4 = snap252
					d5 = snap253
					d6 = snap254
					d25 = snap255
					d26 = snap256
					d27 = snap257
					d28 = snap258
					d29 = snap259
					d30 = snap260
					d33 = snap261
					d50 = snap262
					d66 = snap263
					d67 = snap264
					d68 = snap265
					d69 = snap266
					d71 = snap267
					d72 = snap268
					d74 = snap269
					d75 = snap270
					d76 = snap271
					d77 = snap272
					d79 = snap273
					d81 = snap274
					d82 = snap275
					d111 = snap276
					d114 = snap277
					d145 = snap278
					d146 = snap279
					d148 = snap280
					d149 = snap281
					d150 = snap282
					d151 = snap283
					d152 = snap284
					d155 = snap285
					d234 = snap286
					d235 = snap287
					d236 = snap288
					d237 = snap289
					d238 = snap290
					d239 = snap291
					d240 = snap292
					d241 = snap293
					d242 = snap294
					d243 = snap295
					d244 = snap296
					d245 = snap297
					d246 = snap298
					ctx.RestoreAllocState(alloc299)
					d1 = snap249
					d2 = snap250
					d3 = snap251
					d4 = snap252
					d5 = snap253
					d6 = snap254
					d25 = snap255
					d26 = snap256
					d27 = snap257
					d28 = snap258
					d29 = snap259
					d30 = snap260
					d33 = snap261
					d50 = snap262
					d66 = snap263
					d67 = snap264
					d68 = snap265
					d69 = snap266
					d71 = snap267
					d72 = snap268
					d74 = snap269
					d75 = snap270
					d76 = snap271
					d77 = snap272
					d79 = snap273
					d81 = snap274
					d82 = snap275
					d111 = snap276
					d114 = snap277
					d145 = snap278
					d146 = snap279
					d148 = snap280
					d149 = snap281
					d150 = snap282
					d151 = snap283
					d152 = snap284
					d155 = snap285
					d234 = snap286
					d235 = snap287
					d236 = snap288
					d237 = snap289
					d238 = snap290
					d239 = snap291
					d240 = snap292
					d241 = snap293
					d242 = snap294
					d243 = snap295
					d244 = snap296
					d245 = snap297
					d246 = snap298
					ps300 := PhiState{General: true}
					ps300.OverlayValues = make([]JITValueDesc, 247)
					ps300.OverlayValues[1] = d1
					ps300.OverlayValues[2] = d2
					ps300.OverlayValues[3] = d3
					ps300.OverlayValues[4] = d4
					ps300.OverlayValues[5] = d5
					ps300.OverlayValues[6] = d6
					ps300.OverlayValues[25] = d25
					ps300.OverlayValues[26] = d26
					ps300.OverlayValues[27] = d27
					ps300.OverlayValues[28] = d28
					ps300.OverlayValues[29] = d29
					ps300.OverlayValues[30] = d30
					ps300.OverlayValues[33] = d33
					ps300.OverlayValues[50] = d50
					ps300.OverlayValues[66] = d66
					ps300.OverlayValues[67] = d67
					ps300.OverlayValues[68] = d68
					ps300.OverlayValues[69] = d69
					ps300.OverlayValues[71] = d71
					ps300.OverlayValues[72] = d72
					ps300.OverlayValues[74] = d74
					ps300.OverlayValues[75] = d75
					ps300.OverlayValues[76] = d76
					ps300.OverlayValues[77] = d77
					ps300.OverlayValues[79] = d79
					ps300.OverlayValues[81] = d81
					ps300.OverlayValues[82] = d82
					ps300.OverlayValues[111] = d111
					ps300.OverlayValues[114] = d114
					ps300.OverlayValues[145] = d145
					ps300.OverlayValues[146] = d146
					ps300.OverlayValues[148] = d148
					ps300.OverlayValues[149] = d149
					ps300.OverlayValues[150] = d150
					ps300.OverlayValues[151] = d151
					ps300.OverlayValues[152] = d152
					ps300.OverlayValues[155] = d155
					ps300.OverlayValues[234] = d234
					ps300.OverlayValues[235] = d235
					ps300.OverlayValues[236] = d236
					ps300.OverlayValues[237] = d237
					ps300.OverlayValues[238] = d238
					ps300.OverlayValues[239] = d239
					ps300.OverlayValues[240] = d240
					ps300.OverlayValues[241] = d241
					ps300.OverlayValues[242] = d242
					ps300.OverlayValues[243] = d243
					ps300.OverlayValues[244] = d244
					ps300.OverlayValues[245] = d245
					ps300.OverlayValues[246] = d246
					ps301 := PhiState{General: true}
					ps301.OverlayValues = make([]JITValueDesc, 247)
					ps301.OverlayValues[1] = d1
					ps301.OverlayValues[2] = d2
					ps301.OverlayValues[3] = d3
					ps301.OverlayValues[4] = d4
					ps301.OverlayValues[5] = d5
					ps301.OverlayValues[6] = d6
					ps301.OverlayValues[25] = d25
					ps301.OverlayValues[26] = d26
					ps301.OverlayValues[27] = d27
					ps301.OverlayValues[28] = d28
					ps301.OverlayValues[29] = d29
					ps301.OverlayValues[30] = d30
					ps301.OverlayValues[33] = d33
					ps301.OverlayValues[50] = d50
					ps301.OverlayValues[66] = d66
					ps301.OverlayValues[67] = d67
					ps301.OverlayValues[68] = d68
					ps301.OverlayValues[69] = d69
					ps301.OverlayValues[71] = d71
					ps301.OverlayValues[72] = d72
					ps301.OverlayValues[74] = d74
					ps301.OverlayValues[75] = d75
					ps301.OverlayValues[76] = d76
					ps301.OverlayValues[77] = d77
					ps301.OverlayValues[79] = d79
					ps301.OverlayValues[81] = d81
					ps301.OverlayValues[82] = d82
					ps301.OverlayValues[111] = d111
					ps301.OverlayValues[114] = d114
					ps301.OverlayValues[145] = d145
					ps301.OverlayValues[146] = d146
					ps301.OverlayValues[148] = d148
					ps301.OverlayValues[149] = d149
					ps301.OverlayValues[150] = d150
					ps301.OverlayValues[151] = d151
					ps301.OverlayValues[152] = d152
					ps301.OverlayValues[155] = d155
					ps301.OverlayValues[234] = d234
					ps301.OverlayValues[235] = d235
					ps301.OverlayValues[236] = d236
					ps301.OverlayValues[237] = d237
					ps301.OverlayValues[238] = d238
					ps301.OverlayValues[239] = d239
					ps301.OverlayValues[240] = d240
					ps301.OverlayValues[241] = d241
					ps301.OverlayValues[242] = d242
					ps301.OverlayValues[243] = d243
					ps301.OverlayValues[244] = d244
					ps301.OverlayValues[245] = d245
					ps301.OverlayValues[246] = d246
					snap302 := d1
					snap303 := d2
					snap304 := d3
					snap305 := d4
					snap306 := d5
					snap307 := d6
					snap308 := d25
					snap309 := d26
					snap310 := d27
					snap311 := d28
					snap312 := d29
					snap313 := d30
					snap314 := d33
					snap315 := d50
					snap316 := d66
					snap317 := d67
					snap318 := d68
					snap319 := d69
					snap320 := d71
					snap321 := d72
					snap322 := d74
					snap323 := d75
					snap324 := d76
					snap325 := d77
					snap326 := d79
					snap327 := d81
					snap328 := d82
					snap329 := d111
					snap330 := d114
					snap331 := d145
					snap332 := d146
					snap333 := d148
					snap334 := d149
					snap335 := d150
					snap336 := d151
					snap337 := d152
					snap338 := d155
					snap339 := d234
					snap340 := d235
					snap341 := d236
					snap342 := d237
					snap343 := d238
					snap344 := d239
					snap345 := d240
					snap346 := d241
					snap347 := d242
					snap348 := d243
					snap349 := d244
					snap350 := d245
					snap351 := d246
					alloc352 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps301)
					}
					ctx.RestoreAllocState(alloc352)
					d1 = snap302
					d2 = snap303
					d3 = snap304
					d4 = snap305
					d5 = snap306
					d6 = snap307
					d25 = snap308
					d26 = snap309
					d27 = snap310
					d28 = snap311
					d29 = snap312
					d30 = snap313
					d33 = snap314
					d50 = snap315
					d66 = snap316
					d67 = snap317
					d68 = snap318
					d69 = snap319
					d71 = snap320
					d72 = snap321
					d74 = snap322
					d75 = snap323
					d76 = snap324
					d77 = snap325
					d79 = snap326
					d81 = snap327
					d82 = snap328
					d111 = snap329
					d114 = snap330
					d145 = snap331
					d146 = snap332
					d148 = snap333
					d149 = snap334
					d150 = snap335
					d151 = snap336
					d152 = snap337
					d155 = snap338
					d234 = snap339
					d235 = snap340
					d236 = snap341
					d237 = snap342
					d238 = snap343
					d239 = snap344
					d240 = snap345
					d241 = snap346
					d242 = snap347
					d243 = snap348
					d244 = snap349
					d245 = snap350
					d246 = snap351
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps300)
					}
					return result
					ctx.FreeDesc(&d244)
					return result
				}
				ps353 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps353)
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
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
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
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d305 JITValueDesc
				_ = d305
				var d306 JITValueDesc
				_ = d306
				var d307 JITValueDesc
				_ = d307
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
				var d336 JITValueDesc
				_ = d336
				var d337 JITValueDesc
				_ = d337
				var d338 JITValueDesc
				_ = d338
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
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					snap13 := d5
					snap14 := d6
					alloc15 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc15)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ctx.RestoreAllocState(alloc15)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ps16 := PhiState{General: true}
					ps16.OverlayValues = make([]JITValueDesc, 7)
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					ps16.OverlayValues[6] = d6
					ps17 := PhiState{General: true}
					ps17.OverlayValues = make([]JITValueDesc, 7)
					ps17.OverlayValues[1] = d1
					ps17.OverlayValues[2] = d2
					ps17.OverlayValues[3] = d3
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					snap18 := d1
					snap19 := d2
					snap20 := d3
					snap21 := d4
					snap22 := d5
					snap23 := d6
					alloc24 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps17)
					}
					ctx.RestoreAllocState(alloc24)
					d1 = snap18
					d2 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					d6 = snap23
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps16)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					ctx.ReclaimUntrackedRegs()
					d26 = args[1]
					d26.ID = 0
					d28 = d26
					ctx.SyncDesc(&d28)
					if d28.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d28.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d28.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d28 = tmpScalar
					}
					d28 = JITPrepareScmerGoArg(ctx, d28)
					if d28.Loc != LocRegPair && d28.Loc != LocStackPair && d28.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d27 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d28}, 2)
					ctx.FreeDesc(&d26)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d27.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d27.Imm)
						ptrWord, _ := d27.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d27.Imm.String())))
						d27 = tmpPair
					} else if d27.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d27.Type, Reg: ctx.AllocRegExcept(d27.Reg), Reg2: ctx.AllocRegExcept(d27.Reg)}
						switch d27.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d27)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d27)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d27)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d27)
						d27 = tmpPair
					}
					if d27.Loc != LocRegPair && d27.Loc != LocStackPair && d27.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ResolveLocation arg0)")
					}
					ctx.SyncDesc(&d27)
					callResults29 := JITEmitGoCallResults(ctx, GoFuncAddr(ResolveLocation), []JITValueDesc{d27}, []uint8{1, 2}, []uint8{1, 3})
					d30 = callResults29[0]
					_ = d30
					d31 = callResults29[1]
					_ = d31
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.EnsureDesc(&d31)
					var d32 JITValueDesc
					if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d31)
						if d31.Loc != LocReg && d31.Loc != LocRegPair && d31.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d31.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d32)
					}
					ctx.FreeDesc(&d31)
					d33 = d32
					ctx.EnsureDesc(&d33)
					if d33.Loc != LocImm && d33.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d33.Loc == LocImm {
						if d33.Imm.Bool() {
							if ps.General {
							}
							ps34 := PhiState{General: ps.General}
							ps34.OverlayValues = make([]JITValueDesc, 34)
							ps34.OverlayValues[1] = d1
							ps34.OverlayValues[2] = d2
							ps34.OverlayValues[3] = d3
							ps34.OverlayValues[4] = d4
							ps34.OverlayValues[5] = d5
							ps34.OverlayValues[6] = d6
							ps34.OverlayValues[25] = d25
							ps34.OverlayValues[26] = d26
							ps34.OverlayValues[27] = d27
							ps34.OverlayValues[28] = d28
							ps34.OverlayValues[30] = d30
							ps34.OverlayValues[31] = d31
							ps34.OverlayValues[32] = d32
							ps34.OverlayValues[33] = d33
							return bbs[4].RenderPS(ps34)
						}
						if ps.General {
						}
						ps35 := PhiState{General: ps.General}
						ps35.OverlayValues = make([]JITValueDesc, 34)
						ps35.OverlayValues[1] = d1
						ps35.OverlayValues[2] = d2
						ps35.OverlayValues[3] = d3
						ps35.OverlayValues[4] = d4
						ps35.OverlayValues[5] = d5
						ps35.OverlayValues[6] = d6
						ps35.OverlayValues[25] = d25
						ps35.OverlayValues[26] = d26
						ps35.OverlayValues[27] = d27
						ps35.OverlayValues[28] = d28
						ps35.OverlayValues[30] = d30
						ps35.OverlayValues[31] = d31
						ps35.OverlayValues[32] = d32
						ps35.OverlayValues[33] = d33
						return bbs[5].RenderPS(ps35)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d33.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					snap36 := d1
					snap37 := d2
					snap38 := d3
					snap39 := d4
					snap40 := d5
					snap41 := d6
					snap42 := d25
					snap43 := d26
					snap44 := d27
					snap45 := d28
					snap46 := d30
					snap47 := d31
					snap48 := d32
					snap49 := d33
					alloc50 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc50)
					d1 = snap36
					d2 = snap37
					d3 = snap38
					d4 = snap39
					d5 = snap40
					d6 = snap41
					d25 = snap42
					d26 = snap43
					d27 = snap44
					d28 = snap45
					d30 = snap46
					d31 = snap47
					d32 = snap48
					d33 = snap49
					ctx.RestoreAllocState(alloc50)
					d1 = snap36
					d2 = snap37
					d3 = snap38
					d4 = snap39
					d5 = snap40
					d6 = snap41
					d25 = snap42
					d26 = snap43
					d27 = snap44
					d28 = snap45
					d30 = snap46
					d31 = snap47
					d32 = snap48
					d33 = snap49
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 34)
					ps51.OverlayValues[1] = d1
					ps51.OverlayValues[2] = d2
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[6] = d6
					ps51.OverlayValues[25] = d25
					ps51.OverlayValues[26] = d26
					ps51.OverlayValues[27] = d27
					ps51.OverlayValues[28] = d28
					ps51.OverlayValues[30] = d30
					ps51.OverlayValues[31] = d31
					ps51.OverlayValues[32] = d32
					ps51.OverlayValues[33] = d33
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 34)
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[26] = d26
					ps52.OverlayValues[27] = d27
					ps52.OverlayValues[28] = d28
					ps52.OverlayValues[30] = d30
					ps52.OverlayValues[31] = d31
					ps52.OverlayValues[32] = d32
					ps52.OverlayValues[33] = d33
					snap53 := d1
					snap54 := d2
					snap55 := d3
					snap56 := d4
					snap57 := d5
					snap58 := d6
					snap59 := d25
					snap60 := d26
					snap61 := d27
					snap62 := d28
					snap63 := d30
					snap64 := d31
					snap65 := d32
					snap66 := d33
					alloc67 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc67)
					d1 = snap53
					d2 = snap54
					d3 = snap55
					d4 = snap56
					d5 = snap57
					d6 = snap58
					d25 = snap59
					d26 = snap60
					d27 = snap61
					d28 = snap62
					d30 = snap63
					d31 = snap64
					d32 = snap65
					d33 = snap66
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps51)
					}
					return result
					ctx.FreeDesc(&d32)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					ctx.ReclaimUntrackedRegs()
					d68 = args[1]
					d68.ID = 0
					d70 = d68
					d70.ID = 0
					d69 = ctx.EmitTagEqualsBorrowed(&d70, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d68)
					d71 = d69
					ctx.EnsureDesc(&d71)
					if d71.Loc != LocImm && d71.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d71.Loc == LocImm {
						if d71.Imm.Bool() {
							if ps.General {
							}
							ps72 := PhiState{General: ps.General}
							ps72.OverlayValues = make([]JITValueDesc, 72)
							ps72.OverlayValues[1] = d1
							ps72.OverlayValues[2] = d2
							ps72.OverlayValues[3] = d3
							ps72.OverlayValues[4] = d4
							ps72.OverlayValues[5] = d5
							ps72.OverlayValues[6] = d6
							ps72.OverlayValues[25] = d25
							ps72.OverlayValues[26] = d26
							ps72.OverlayValues[27] = d27
							ps72.OverlayValues[28] = d28
							ps72.OverlayValues[30] = d30
							ps72.OverlayValues[31] = d31
							ps72.OverlayValues[32] = d32
							ps72.OverlayValues[33] = d33
							ps72.OverlayValues[68] = d68
							ps72.OverlayValues[69] = d69
							ps72.OverlayValues[70] = d70
							ps72.OverlayValues[71] = d71
							return bbs[1].RenderPS(ps72)
						}
						if ps.General {
						}
						ps73 := PhiState{General: ps.General}
						ps73.OverlayValues = make([]JITValueDesc, 72)
						ps73.OverlayValues[1] = d1
						ps73.OverlayValues[2] = d2
						ps73.OverlayValues[3] = d3
						ps73.OverlayValues[4] = d4
						ps73.OverlayValues[5] = d5
						ps73.OverlayValues[6] = d6
						ps73.OverlayValues[25] = d25
						ps73.OverlayValues[26] = d26
						ps73.OverlayValues[27] = d27
						ps73.OverlayValues[28] = d28
						ps73.OverlayValues[30] = d30
						ps73.OverlayValues[31] = d31
						ps73.OverlayValues[32] = d32
						ps73.OverlayValues[33] = d33
						ps73.OverlayValues[68] = d68
						ps73.OverlayValues[69] = d69
						ps73.OverlayValues[70] = d70
						ps73.OverlayValues[71] = d71
						return bbs[2].RenderPS(ps73)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d71.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap74 := d1
					snap75 := d2
					snap76 := d3
					snap77 := d4
					snap78 := d5
					snap79 := d6
					snap80 := d25
					snap81 := d26
					snap82 := d27
					snap83 := d28
					snap84 := d30
					snap85 := d31
					snap86 := d32
					snap87 := d33
					snap88 := d68
					snap89 := d69
					snap90 := d70
					snap91 := d71
					alloc92 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc92)
					d1 = snap74
					d2 = snap75
					d3 = snap76
					d4 = snap77
					d5 = snap78
					d6 = snap79
					d25 = snap80
					d26 = snap81
					d27 = snap82
					d28 = snap83
					d30 = snap84
					d31 = snap85
					d32 = snap86
					d33 = snap87
					d68 = snap88
					d69 = snap89
					d70 = snap90
					d71 = snap91
					ctx.RestoreAllocState(alloc92)
					d1 = snap74
					d2 = snap75
					d3 = snap76
					d4 = snap77
					d5 = snap78
					d6 = snap79
					d25 = snap80
					d26 = snap81
					d27 = snap82
					d28 = snap83
					d30 = snap84
					d31 = snap85
					d32 = snap86
					d33 = snap87
					d68 = snap88
					d69 = snap89
					d70 = snap90
					d71 = snap91
					ps93 := PhiState{General: true}
					ps93.OverlayValues = make([]JITValueDesc, 72)
					ps93.OverlayValues[1] = d1
					ps93.OverlayValues[2] = d2
					ps93.OverlayValues[3] = d3
					ps93.OverlayValues[4] = d4
					ps93.OverlayValues[5] = d5
					ps93.OverlayValues[6] = d6
					ps93.OverlayValues[25] = d25
					ps93.OverlayValues[26] = d26
					ps93.OverlayValues[27] = d27
					ps93.OverlayValues[28] = d28
					ps93.OverlayValues[30] = d30
					ps93.OverlayValues[31] = d31
					ps93.OverlayValues[32] = d32
					ps93.OverlayValues[33] = d33
					ps93.OverlayValues[68] = d68
					ps93.OverlayValues[69] = d69
					ps93.OverlayValues[70] = d70
					ps93.OverlayValues[71] = d71
					ps94 := PhiState{General: true}
					ps94.OverlayValues = make([]JITValueDesc, 72)
					ps94.OverlayValues[1] = d1
					ps94.OverlayValues[2] = d2
					ps94.OverlayValues[3] = d3
					ps94.OverlayValues[4] = d4
					ps94.OverlayValues[5] = d5
					ps94.OverlayValues[6] = d6
					ps94.OverlayValues[25] = d25
					ps94.OverlayValues[26] = d26
					ps94.OverlayValues[27] = d27
					ps94.OverlayValues[28] = d28
					ps94.OverlayValues[30] = d30
					ps94.OverlayValues[31] = d31
					ps94.OverlayValues[32] = d32
					ps94.OverlayValues[33] = d33
					ps94.OverlayValues[68] = d68
					ps94.OverlayValues[69] = d69
					ps94.OverlayValues[70] = d70
					ps94.OverlayValues[71] = d71
					snap95 := d1
					snap96 := d2
					snap97 := d3
					snap98 := d4
					snap99 := d5
					snap100 := d6
					snap101 := d25
					snap102 := d26
					snap103 := d27
					snap104 := d28
					snap105 := d30
					snap106 := d31
					snap107 := d32
					snap108 := d33
					snap109 := d68
					snap110 := d69
					snap111 := d70
					snap112 := d71
					alloc113 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps94)
					}
					ctx.RestoreAllocState(alloc113)
					d1 = snap95
					d2 = snap96
					d3 = snap97
					d4 = snap98
					d5 = snap99
					d6 = snap100
					d25 = snap101
					d26 = snap102
					d27 = snap103
					d28 = snap104
					d30 = snap105
					d31 = snap106
					d32 = snap107
					d33 = snap108
					d68 = snap109
					d69 = snap110
					d70 = snap111
					d71 = snap112
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps93)
					}
					return result
					ctx.FreeDesc(&d69)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					ctx.ReclaimUntrackedRegs()
					d114 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d114)
					if d114.Loc == LocRegPair || d114.Loc == LocStackPair || d114.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d114, &result)
						result.Type = d114.Type
					} else {
						switch d114.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d114)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d114)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d114)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d114, &result)
							result.Type = d114.Type
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					ctx.ReclaimUntrackedRegs()
					d115 = args[0]
					d115.ID = 0
					d116 = ctx.EmitGetTagDesc(&d115, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d115)
					ctx.EnsureDesc(&d116)
					var d117 JITValueDesc
					if d116.Loc == LocImm {
						d117 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d116.Imm.Int()) == uint64(0x10))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d116.Reg, 16)
						d117 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d117)
					}
					ctx.FreeDesc(&d116)
					d118 = d117
					ctx.EnsureDesc(&d118)
					if d118.Loc != LocImm && d118.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d118.Loc == LocImm {
						if d118.Imm.Bool() {
							if ps.General {
							}
							ps119 := PhiState{General: ps.General}
							ps119.OverlayValues = make([]JITValueDesc, 119)
							ps119.OverlayValues[1] = d1
							ps119.OverlayValues[2] = d2
							ps119.OverlayValues[3] = d3
							ps119.OverlayValues[4] = d4
							ps119.OverlayValues[5] = d5
							ps119.OverlayValues[6] = d6
							ps119.OverlayValues[25] = d25
							ps119.OverlayValues[26] = d26
							ps119.OverlayValues[27] = d27
							ps119.OverlayValues[28] = d28
							ps119.OverlayValues[30] = d30
							ps119.OverlayValues[31] = d31
							ps119.OverlayValues[32] = d32
							ps119.OverlayValues[33] = d33
							ps119.OverlayValues[68] = d68
							ps119.OverlayValues[69] = d69
							ps119.OverlayValues[70] = d70
							ps119.OverlayValues[71] = d71
							ps119.OverlayValues[114] = d114
							ps119.OverlayValues[115] = d115
							ps119.OverlayValues[116] = d116
							ps119.OverlayValues[117] = d117
							ps119.OverlayValues[118] = d118
							return bbs[6].RenderPS(ps119)
						}
						if ps.General {
						}
						ps120 := PhiState{General: ps.General}
						ps120.OverlayValues = make([]JITValueDesc, 119)
						ps120.OverlayValues[1] = d1
						ps120.OverlayValues[2] = d2
						ps120.OverlayValues[3] = d3
						ps120.OverlayValues[4] = d4
						ps120.OverlayValues[5] = d5
						ps120.OverlayValues[6] = d6
						ps120.OverlayValues[25] = d25
						ps120.OverlayValues[26] = d26
						ps120.OverlayValues[27] = d27
						ps120.OverlayValues[28] = d28
						ps120.OverlayValues[30] = d30
						ps120.OverlayValues[31] = d31
						ps120.OverlayValues[32] = d32
						ps120.OverlayValues[33] = d33
						ps120.OverlayValues[68] = d68
						ps120.OverlayValues[69] = d69
						ps120.OverlayValues[70] = d70
						ps120.OverlayValues[71] = d71
						ps120.OverlayValues[114] = d114
						ps120.OverlayValues[115] = d115
						ps120.OverlayValues[116] = d116
						ps120.OverlayValues[117] = d117
						ps120.OverlayValues[118] = d118
						return bbs[8].RenderPS(ps120)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					ctx.EmitJump(d118.Condition, lbl7)
					snap121 := d1
					snap122 := d2
					snap123 := d3
					snap124 := d4
					snap125 := d5
					snap126 := d6
					snap127 := d25
					snap128 := d26
					snap129 := d27
					snap130 := d28
					snap131 := d30
					snap132 := d31
					snap133 := d32
					snap134 := d33
					snap135 := d68
					snap136 := d69
					snap137 := d70
					snap138 := d71
					snap139 := d114
					snap140 := d115
					snap141 := d116
					snap142 := d117
					snap143 := d118
					alloc144 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc144)
					d1 = snap121
					d2 = snap122
					d3 = snap123
					d4 = snap124
					d5 = snap125
					d6 = snap126
					d25 = snap127
					d26 = snap128
					d27 = snap129
					d28 = snap130
					d30 = snap131
					d31 = snap132
					d32 = snap133
					d33 = snap134
					d68 = snap135
					d69 = snap136
					d70 = snap137
					d71 = snap138
					d114 = snap139
					d115 = snap140
					d116 = snap141
					d117 = snap142
					d118 = snap143
					ctx.RestoreAllocState(alloc144)
					d1 = snap121
					d2 = snap122
					d3 = snap123
					d4 = snap124
					d5 = snap125
					d6 = snap126
					d25 = snap127
					d26 = snap128
					d27 = snap129
					d28 = snap130
					d30 = snap131
					d31 = snap132
					d32 = snap133
					d33 = snap134
					d68 = snap135
					d69 = snap136
					d70 = snap137
					d71 = snap138
					d114 = snap139
					d115 = snap140
					d116 = snap141
					d117 = snap142
					d118 = snap143
					ps145 := PhiState{General: true}
					ps145.OverlayValues = make([]JITValueDesc, 119)
					ps145.OverlayValues[1] = d1
					ps145.OverlayValues[2] = d2
					ps145.OverlayValues[3] = d3
					ps145.OverlayValues[4] = d4
					ps145.OverlayValues[5] = d5
					ps145.OverlayValues[6] = d6
					ps145.OverlayValues[25] = d25
					ps145.OverlayValues[26] = d26
					ps145.OverlayValues[27] = d27
					ps145.OverlayValues[28] = d28
					ps145.OverlayValues[30] = d30
					ps145.OverlayValues[31] = d31
					ps145.OverlayValues[32] = d32
					ps145.OverlayValues[33] = d33
					ps145.OverlayValues[68] = d68
					ps145.OverlayValues[69] = d69
					ps145.OverlayValues[70] = d70
					ps145.OverlayValues[71] = d71
					ps145.OverlayValues[114] = d114
					ps145.OverlayValues[115] = d115
					ps145.OverlayValues[116] = d116
					ps145.OverlayValues[117] = d117
					ps145.OverlayValues[118] = d118
					ps146 := PhiState{General: true}
					ps146.OverlayValues = make([]JITValueDesc, 119)
					ps146.OverlayValues[1] = d1
					ps146.OverlayValues[2] = d2
					ps146.OverlayValues[3] = d3
					ps146.OverlayValues[4] = d4
					ps146.OverlayValues[5] = d5
					ps146.OverlayValues[6] = d6
					ps146.OverlayValues[25] = d25
					ps146.OverlayValues[26] = d26
					ps146.OverlayValues[27] = d27
					ps146.OverlayValues[28] = d28
					ps146.OverlayValues[30] = d30
					ps146.OverlayValues[31] = d31
					ps146.OverlayValues[32] = d32
					ps146.OverlayValues[33] = d33
					ps146.OverlayValues[68] = d68
					ps146.OverlayValues[69] = d69
					ps146.OverlayValues[70] = d70
					ps146.OverlayValues[71] = d71
					ps146.OverlayValues[114] = d114
					ps146.OverlayValues[115] = d115
					ps146.OverlayValues[116] = d116
					ps146.OverlayValues[117] = d117
					ps146.OverlayValues[118] = d118
					snap147 := d1
					snap148 := d2
					snap149 := d3
					snap150 := d4
					snap151 := d5
					snap152 := d6
					snap153 := d25
					snap154 := d26
					snap155 := d27
					snap156 := d28
					snap157 := d30
					snap158 := d31
					snap159 := d32
					snap160 := d33
					snap161 := d68
					snap162 := d69
					snap163 := d70
					snap164 := d71
					snap165 := d114
					snap166 := d115
					snap167 := d116
					snap168 := d117
					snap169 := d118
					alloc170 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps146)
					}
					ctx.RestoreAllocState(alloc170)
					d1 = snap147
					d2 = snap148
					d3 = snap149
					d4 = snap150
					d5 = snap151
					d6 = snap152
					d25 = snap153
					d26 = snap154
					d27 = snap155
					d28 = snap156
					d30 = snap157
					d31 = snap158
					d32 = snap159
					d33 = snap160
					d68 = snap161
					d69 = snap162
					d70 = snap163
					d71 = snap164
					d114 = snap165
					d115 = snap166
					d116 = snap167
					d117 = snap168
					d118 = snap169
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps145)
					}
					return result
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					ctx.ReclaimUntrackedRegs()
					d171 = args[0]
					d171.ID = 0
					var d172 JITValueDesc
					ctx.EnsureDesc(&d171)
					if d171.Loc == LocImm {
						_, auxWord := d171.Imm.RawWords()
						d172 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d171.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r2 := ctx.AllocReg()
						ctx.EmitMovRegReg(r2, d171.Reg2)
						d172 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d172)
					}
					ctx.EnsureDesc(&d172)
					d173 = d172
					_ = d173
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d173)
					var d174 JITValueDesc
					if d173.Loc == LocImm {
						d174 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d173.Imm.Int()) >> 8))}
					} else {
						r3 := ctx.AllocRegExcept(d173.Reg)
						ctx.EmitMovRegReg(r3, d173.Reg)
						ctx.EmitShrRegImm8(r3, 8)
						d174 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d174)
					}
					if d174.Loc == LocReg && d173.Loc == LocReg && d174.Reg == d173.Reg {
						ctx.TransferReg(d173.Reg)
						d173.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d174)
					ctx.FreeDesc(&d172)
					ctx.EnsureDesc(&d174)
					d175 = d174
					_ = d175
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d175)
					var d176 JITValueDesc
					if d175.Loc == LocImm {
						d176 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d175.Imm.Int() & 35184372088831)}
					} else {
						r4 := ctx.AllocRegExcept(d175.Reg)
						ctx.EmitMovRegReg(r4, d175.Reg)
						ctx.EmitMovRegImm64(RegR11, 0x1fffffffffff)
						ctx.EmitAndInt64(r4, RegR11)
						d176 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d176)
					}
					if d176.Loc == LocReg && d175.Loc == LocReg && d176.Reg == d175.Reg {
						ctx.TransferReg(d175.Reg)
						d175.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d176)
					var d177 JITValueDesc
					if d176.Loc == LocImm {
						d177 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d176.Imm.Int()) << 19))}
					} else {
						ctx.EmitShlRegImm8(d176.Reg, 19)
						d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d176.Reg}
						ctx.BindReg(d176.Reg, &d177)
					}
					if d177.Loc == LocReg && d176.Loc == LocReg && d177.Reg == d176.Reg {
						ctx.TransferReg(d176.Reg)
						d176.Loc = LocNone
					}
					ctx.FreeDesc(&d176)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d177)
					ctx.EnsureDesc(&d177)
					var d178 JITValueDesc
					if d177.Loc == LocImm {
						d178 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d177.Imm.Int()))))}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegReg(r5, d177.Reg)
						d178 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d178)
					}
					ctx.FreeDesc(&d177)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d178)
					var d179 JITValueDesc
					if d178.Loc == LocImm {
						d179 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d178.Imm.Int()) >> 19))}
					} else {
						ctx.EmitShrRegImm8(d178.Reg, 19)
						d179 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d178.Reg}
						ctx.BindReg(d178.Reg, &d179)
					}
					if d179.Loc == LocReg && d178.Loc == LocReg && d179.Reg == d178.Reg {
						ctx.TransferReg(d178.Reg)
						d178.Loc = LocNone
					}
					ctx.FreeDesc(&d178)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d179)
					ctx.StabilizeDescForControlFlow(&d179)
					ctx.FreeDesc(&d174)
					d180 = args[0]
					d180.ID = 0
					var d181 JITValueDesc
					ctx.EnsureDesc(&d180)
					if d180.Loc == LocImm {
						_, auxWord := d180.Imm.RawWords()
						d181 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d180.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r6 := ctx.AllocReg()
						ctx.EmitMovRegReg(r6, d180.Reg2)
						d181 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d181)
					}
					ctx.EnsureDesc(&d181)
					d182 = d181
					_ = d182
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d182)
					var d183 JITValueDesc
					if d182.Loc == LocImm {
						d183 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d182.Imm.Int()) >> 8))}
					} else {
						r7 := ctx.AllocRegExcept(d182.Reg)
						ctx.EmitMovRegReg(r7, d182.Reg)
						ctx.EmitShrRegImm8(r7, 8)
						d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d183)
					}
					if d183.Loc == LocReg && d182.Loc == LocReg && d183.Reg == d182.Reg {
						ctx.TransferReg(d182.Reg)
						d182.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d183)
					ctx.FreeDesc(&d181)
					ctx.EnsureDesc(&d183)
					d184 = d183
					_ = d184
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d184)
					var d185 JITValueDesc
					if d184.Loc == LocImm {
						d185 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d184.Imm.Int()) >> 45))}
					} else {
						r8 := ctx.AllocRegExcept(d184.Reg)
						ctx.EmitMovRegReg(r8, d184.Reg)
						ctx.EmitShrRegImm8(r8, 45)
						d185 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d185)
					}
					if d185.Loc == LocReg && d184.Loc == LocReg && d185.Reg == d184.Reg {
						ctx.TransferReg(d184.Reg)
						d184.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d185)
					var d186 JITValueDesc
					if d185.Loc == LocImm {
						d186 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d185.Imm.Int() & 2047)}
					} else {
						ctx.EmitAndRegImm32(d185.Reg, int32(2047))
						d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d185.Reg}
						ctx.BindReg(d185.Reg, &d186)
					}
					if d186.Loc == LocReg && d185.Loc == LocReg && d186.Reg == d185.Reg {
						ctx.TransferReg(d185.Reg)
						d185.Loc = LocNone
					}
					ctx.FreeDesc(&d185)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d186)
					ctx.EnsureDesc(&d186)
					var d187 JITValueDesc
					if d186.Loc == LocImm {
						d187 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d186.Imm.Int()))))}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitMovRegReg(r9, d186.Reg)
						d187 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d187)
					}
					ctx.FreeDesc(&d186)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d187)
					ctx.StabilizeDescForControlFlow(&d187)
					ctx.FreeDesc(&d183)
					if ps.General {
						ctx.SyncDesc(&d179)
						if d179.Loc == LocReg {
							ctx.ProtectReg(d179.Reg)
						} else if d179.Loc == LocRegPair {
							ctx.ProtectReg(d179.Reg)
							ctx.ProtectReg(d179.Reg2)
						}
						ctx.SyncDesc(&d187)
						if d187.Loc == LocReg {
							ctx.ProtectReg(d187.Reg)
						} else if d187.Loc == LocRegPair {
							ctx.ProtectReg(d187.Reg)
							ctx.ProtectReg(d187.Reg2)
						}
						d188 = d179
						if d188.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d188)
						ctx.EmitStoreToStack(d188, int32(bbs[7].PhiBase)+int32(0))
						d189 = d187
						if d189.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d189)
						ctx.EmitStoreToStack(d189, int32(bbs[7].PhiBase)+int32(16))
						if d179.Loc == LocReg {
							ctx.UnprotectReg(d179.Reg)
						} else if d179.Loc == LocRegPair {
							ctx.UnprotectReg(d179.Reg)
							ctx.UnprotectReg(d179.Reg2)
						}
						if d187.Loc == LocReg {
							ctx.UnprotectReg(d187.Reg)
						} else if d187.Loc == LocRegPair {
							ctx.UnprotectReg(d187.Reg)
							ctx.UnprotectReg(d187.Reg2)
						}
					}
					ps190 := PhiState{General: ps.General}
					ps190.OverlayValues = make([]JITValueDesc, 190)
					ps190.OverlayValues[1] = d1
					ps190.OverlayValues[2] = d2
					ps190.OverlayValues[3] = d3
					ps190.OverlayValues[4] = d4
					ps190.OverlayValues[5] = d5
					ps190.OverlayValues[6] = d6
					ps190.OverlayValues[25] = d25
					ps190.OverlayValues[26] = d26
					ps190.OverlayValues[27] = d27
					ps190.OverlayValues[28] = d28
					ps190.OverlayValues[30] = d30
					ps190.OverlayValues[31] = d31
					ps190.OverlayValues[32] = d32
					ps190.OverlayValues[33] = d33
					ps190.OverlayValues[68] = d68
					ps190.OverlayValues[69] = d69
					ps190.OverlayValues[70] = d70
					ps190.OverlayValues[71] = d71
					ps190.OverlayValues[114] = d114
					ps190.OverlayValues[115] = d115
					ps190.OverlayValues[116] = d116
					ps190.OverlayValues[117] = d117
					ps190.OverlayValues[118] = d118
					ps190.OverlayValues[171] = d171
					ps190.OverlayValues[172] = d172
					ps190.OverlayValues[173] = d173
					ps190.OverlayValues[174] = d174
					ps190.OverlayValues[175] = d175
					ps190.OverlayValues[176] = d176
					ps190.OverlayValues[177] = d177
					ps190.OverlayValues[178] = d178
					ps190.OverlayValues[179] = d179
					ps190.OverlayValues[180] = d180
					ps190.OverlayValues[181] = d181
					ps190.OverlayValues[182] = d182
					ps190.OverlayValues[183] = d183
					ps190.OverlayValues[184] = d184
					ps190.OverlayValues[185] = d185
					ps190.OverlayValues[186] = d186
					ps190.OverlayValues[187] = d187
					ps190.OverlayValues[188] = d188
					ps190.OverlayValues[189] = d189
					ps190.PhiValues = make([]JITValueDesc, 2)
					d191 = d179
					ps190.PhiValues[0] = d191
					d192 = d187
					ps190.PhiValues[1] = d192
					if ps190.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps190)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d193 := ps.PhiValues[0]
							ctx.EnsureDesc(&d193)
							ctx.EmitStoreToStack(d193, int32(bbs[7].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d194 := ps.PhiValues[1]
							ctx.EnsureDesc(&d194)
							ctx.EmitStoreToStack(d194, int32(bbs[7].PhiBase)+int32(16))
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d2 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d2)
					var d195 JITValueDesc
					if d2.Loc == LocImm {
						d195 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 0)}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						d195 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondEqual}
						ctx.BindReg(r10, &d195)
					}
					ctx.FreeDesc(&d2)
					d196 = d195
					ctx.EnsureDesc(&d196)
					if d196.Loc != LocImm && d196.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d196.Loc == LocImm {
						if d196.Imm.Bool() {
							if ps.General {
							}
							ps197 := PhiState{General: ps.General}
							ps197.OverlayValues = make([]JITValueDesc, 197)
							ps197.OverlayValues[1] = d1
							ps197.OverlayValues[2] = d2
							ps197.OverlayValues[3] = d3
							ps197.OverlayValues[4] = d4
							ps197.OverlayValues[5] = d5
							ps197.OverlayValues[6] = d6
							ps197.OverlayValues[25] = d25
							ps197.OverlayValues[26] = d26
							ps197.OverlayValues[27] = d27
							ps197.OverlayValues[28] = d28
							ps197.OverlayValues[30] = d30
							ps197.OverlayValues[31] = d31
							ps197.OverlayValues[32] = d32
							ps197.OverlayValues[33] = d33
							ps197.OverlayValues[68] = d68
							ps197.OverlayValues[69] = d69
							ps197.OverlayValues[70] = d70
							ps197.OverlayValues[71] = d71
							ps197.OverlayValues[114] = d114
							ps197.OverlayValues[115] = d115
							ps197.OverlayValues[116] = d116
							ps197.OverlayValues[117] = d117
							ps197.OverlayValues[118] = d118
							ps197.OverlayValues[171] = d171
							ps197.OverlayValues[172] = d172
							ps197.OverlayValues[173] = d173
							ps197.OverlayValues[174] = d174
							ps197.OverlayValues[175] = d175
							ps197.OverlayValues[176] = d176
							ps197.OverlayValues[177] = d177
							ps197.OverlayValues[178] = d178
							ps197.OverlayValues[179] = d179
							ps197.OverlayValues[180] = d180
							ps197.OverlayValues[181] = d181
							ps197.OverlayValues[182] = d182
							ps197.OverlayValues[183] = d183
							ps197.OverlayValues[184] = d184
							ps197.OverlayValues[185] = d185
							ps197.OverlayValues[186] = d186
							ps197.OverlayValues[187] = d187
							ps197.OverlayValues[188] = d188
							ps197.OverlayValues[189] = d189
							ps197.OverlayValues[191] = d191
							ps197.OverlayValues[192] = d192
							ps197.OverlayValues[193] = d193
							ps197.OverlayValues[194] = d194
							ps197.OverlayValues[195] = d195
							ps197.OverlayValues[196] = d196
							return bbs[9].RenderPS(ps197)
						}
						if ps.General {
						}
						ps198 := PhiState{General: ps.General}
						ps198.OverlayValues = make([]JITValueDesc, 197)
						ps198.OverlayValues[1] = d1
						ps198.OverlayValues[2] = d2
						ps198.OverlayValues[3] = d3
						ps198.OverlayValues[4] = d4
						ps198.OverlayValues[5] = d5
						ps198.OverlayValues[6] = d6
						ps198.OverlayValues[25] = d25
						ps198.OverlayValues[26] = d26
						ps198.OverlayValues[27] = d27
						ps198.OverlayValues[28] = d28
						ps198.OverlayValues[30] = d30
						ps198.OverlayValues[31] = d31
						ps198.OverlayValues[32] = d32
						ps198.OverlayValues[33] = d33
						ps198.OverlayValues[68] = d68
						ps198.OverlayValues[69] = d69
						ps198.OverlayValues[70] = d70
						ps198.OverlayValues[71] = d71
						ps198.OverlayValues[114] = d114
						ps198.OverlayValues[115] = d115
						ps198.OverlayValues[116] = d116
						ps198.OverlayValues[117] = d117
						ps198.OverlayValues[118] = d118
						ps198.OverlayValues[171] = d171
						ps198.OverlayValues[172] = d172
						ps198.OverlayValues[173] = d173
						ps198.OverlayValues[174] = d174
						ps198.OverlayValues[175] = d175
						ps198.OverlayValues[176] = d176
						ps198.OverlayValues[177] = d177
						ps198.OverlayValues[178] = d178
						ps198.OverlayValues[179] = d179
						ps198.OverlayValues[180] = d180
						ps198.OverlayValues[181] = d181
						ps198.OverlayValues[182] = d182
						ps198.OverlayValues[183] = d183
						ps198.OverlayValues[184] = d184
						ps198.OverlayValues[185] = d185
						ps198.OverlayValues[186] = d186
						ps198.OverlayValues[187] = d187
						ps198.OverlayValues[188] = d188
						ps198.OverlayValues[189] = d189
						ps198.OverlayValues[191] = d191
						ps198.OverlayValues[192] = d192
						ps198.OverlayValues[193] = d193
						ps198.OverlayValues[194] = d194
						ps198.OverlayValues[195] = d195
						ps198.OverlayValues[196] = d196
						return bbs[10].RenderPS(ps198)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d199 := ps.PhiValues[0]
							ctx.EnsureDesc(&d199)
							ctx.EmitStoreToStack(d199, int32(bbs[7].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d200 := ps.PhiValues[1]
							ctx.EnsureDesc(&d200)
							ctx.EmitStoreToStack(d200, int32(bbs[7].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					ctx.EmitJump(d196.Condition, lbl10)
					snap201 := d1
					snap202 := d2
					snap203 := d3
					snap204 := d4
					snap205 := d5
					snap206 := d6
					snap207 := d25
					snap208 := d26
					snap209 := d27
					snap210 := d28
					snap211 := d30
					snap212 := d31
					snap213 := d32
					snap214 := d33
					snap215 := d68
					snap216 := d69
					snap217 := d70
					snap218 := d71
					snap219 := d114
					snap220 := d115
					snap221 := d116
					snap222 := d117
					snap223 := d118
					snap224 := d171
					snap225 := d172
					snap226 := d173
					snap227 := d174
					snap228 := d175
					snap229 := d176
					snap230 := d177
					snap231 := d178
					snap232 := d179
					snap233 := d180
					snap234 := d181
					snap235 := d182
					snap236 := d183
					snap237 := d184
					snap238 := d185
					snap239 := d186
					snap240 := d187
					snap241 := d188
					snap242 := d189
					snap243 := d191
					snap244 := d192
					snap245 := d193
					snap246 := d194
					snap247 := d195
					snap248 := d196
					snap249 := d199
					snap250 := d200
					alloc251 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc251)
					d1 = snap201
					d2 = snap202
					d3 = snap203
					d4 = snap204
					d5 = snap205
					d6 = snap206
					d25 = snap207
					d26 = snap208
					d27 = snap209
					d28 = snap210
					d30 = snap211
					d31 = snap212
					d32 = snap213
					d33 = snap214
					d68 = snap215
					d69 = snap216
					d70 = snap217
					d71 = snap218
					d114 = snap219
					d115 = snap220
					d116 = snap221
					d117 = snap222
					d118 = snap223
					d171 = snap224
					d172 = snap225
					d173 = snap226
					d174 = snap227
					d175 = snap228
					d176 = snap229
					d177 = snap230
					d178 = snap231
					d179 = snap232
					d180 = snap233
					d181 = snap234
					d182 = snap235
					d183 = snap236
					d184 = snap237
					d185 = snap238
					d186 = snap239
					d187 = snap240
					d188 = snap241
					d189 = snap242
					d191 = snap243
					d192 = snap244
					d193 = snap245
					d194 = snap246
					d195 = snap247
					d196 = snap248
					d199 = snap249
					d200 = snap250
					ctx.RestoreAllocState(alloc251)
					d1 = snap201
					d2 = snap202
					d3 = snap203
					d4 = snap204
					d5 = snap205
					d6 = snap206
					d25 = snap207
					d26 = snap208
					d27 = snap209
					d28 = snap210
					d30 = snap211
					d31 = snap212
					d32 = snap213
					d33 = snap214
					d68 = snap215
					d69 = snap216
					d70 = snap217
					d71 = snap218
					d114 = snap219
					d115 = snap220
					d116 = snap221
					d117 = snap222
					d118 = snap223
					d171 = snap224
					d172 = snap225
					d173 = snap226
					d174 = snap227
					d175 = snap228
					d176 = snap229
					d177 = snap230
					d178 = snap231
					d179 = snap232
					d180 = snap233
					d181 = snap234
					d182 = snap235
					d183 = snap236
					d184 = snap237
					d185 = snap238
					d186 = snap239
					d187 = snap240
					d188 = snap241
					d189 = snap242
					d191 = snap243
					d192 = snap244
					d193 = snap245
					d194 = snap246
					d195 = snap247
					d196 = snap248
					d199 = snap249
					d200 = snap250
					ps252 := PhiState{General: true}
					ps252.OverlayValues = make([]JITValueDesc, 201)
					ps252.OverlayValues[1] = d1
					ps252.OverlayValues[2] = d2
					ps252.OverlayValues[3] = d3
					ps252.OverlayValues[4] = d4
					ps252.OverlayValues[5] = d5
					ps252.OverlayValues[6] = d6
					ps252.OverlayValues[25] = d25
					ps252.OverlayValues[26] = d26
					ps252.OverlayValues[27] = d27
					ps252.OverlayValues[28] = d28
					ps252.OverlayValues[30] = d30
					ps252.OverlayValues[31] = d31
					ps252.OverlayValues[32] = d32
					ps252.OverlayValues[33] = d33
					ps252.OverlayValues[68] = d68
					ps252.OverlayValues[69] = d69
					ps252.OverlayValues[70] = d70
					ps252.OverlayValues[71] = d71
					ps252.OverlayValues[114] = d114
					ps252.OverlayValues[115] = d115
					ps252.OverlayValues[116] = d116
					ps252.OverlayValues[117] = d117
					ps252.OverlayValues[118] = d118
					ps252.OverlayValues[171] = d171
					ps252.OverlayValues[172] = d172
					ps252.OverlayValues[173] = d173
					ps252.OverlayValues[174] = d174
					ps252.OverlayValues[175] = d175
					ps252.OverlayValues[176] = d176
					ps252.OverlayValues[177] = d177
					ps252.OverlayValues[178] = d178
					ps252.OverlayValues[179] = d179
					ps252.OverlayValues[180] = d180
					ps252.OverlayValues[181] = d181
					ps252.OverlayValues[182] = d182
					ps252.OverlayValues[183] = d183
					ps252.OverlayValues[184] = d184
					ps252.OverlayValues[185] = d185
					ps252.OverlayValues[186] = d186
					ps252.OverlayValues[187] = d187
					ps252.OverlayValues[188] = d188
					ps252.OverlayValues[189] = d189
					ps252.OverlayValues[191] = d191
					ps252.OverlayValues[192] = d192
					ps252.OverlayValues[193] = d193
					ps252.OverlayValues[194] = d194
					ps252.OverlayValues[195] = d195
					ps252.OverlayValues[196] = d196
					ps252.OverlayValues[199] = d199
					ps252.OverlayValues[200] = d200
					ps253 := PhiState{General: true}
					ps253.OverlayValues = make([]JITValueDesc, 201)
					ps253.OverlayValues[1] = d1
					ps253.OverlayValues[2] = d2
					ps253.OverlayValues[3] = d3
					ps253.OverlayValues[4] = d4
					ps253.OverlayValues[5] = d5
					ps253.OverlayValues[6] = d6
					ps253.OverlayValues[25] = d25
					ps253.OverlayValues[26] = d26
					ps253.OverlayValues[27] = d27
					ps253.OverlayValues[28] = d28
					ps253.OverlayValues[30] = d30
					ps253.OverlayValues[31] = d31
					ps253.OverlayValues[32] = d32
					ps253.OverlayValues[33] = d33
					ps253.OverlayValues[68] = d68
					ps253.OverlayValues[69] = d69
					ps253.OverlayValues[70] = d70
					ps253.OverlayValues[71] = d71
					ps253.OverlayValues[114] = d114
					ps253.OverlayValues[115] = d115
					ps253.OverlayValues[116] = d116
					ps253.OverlayValues[117] = d117
					ps253.OverlayValues[118] = d118
					ps253.OverlayValues[171] = d171
					ps253.OverlayValues[172] = d172
					ps253.OverlayValues[173] = d173
					ps253.OverlayValues[174] = d174
					ps253.OverlayValues[175] = d175
					ps253.OverlayValues[176] = d176
					ps253.OverlayValues[177] = d177
					ps253.OverlayValues[178] = d178
					ps253.OverlayValues[179] = d179
					ps253.OverlayValues[180] = d180
					ps253.OverlayValues[181] = d181
					ps253.OverlayValues[182] = d182
					ps253.OverlayValues[183] = d183
					ps253.OverlayValues[184] = d184
					ps253.OverlayValues[185] = d185
					ps253.OverlayValues[186] = d186
					ps253.OverlayValues[187] = d187
					ps253.OverlayValues[188] = d188
					ps253.OverlayValues[189] = d189
					ps253.OverlayValues[191] = d191
					ps253.OverlayValues[192] = d192
					ps253.OverlayValues[193] = d193
					ps253.OverlayValues[194] = d194
					ps253.OverlayValues[195] = d195
					ps253.OverlayValues[196] = d196
					ps253.OverlayValues[199] = d199
					ps253.OverlayValues[200] = d200
					snap254 := d1
					snap255 := d2
					snap256 := d3
					snap257 := d4
					snap258 := d5
					snap259 := d6
					snap260 := d25
					snap261 := d26
					snap262 := d27
					snap263 := d28
					snap264 := d30
					snap265 := d31
					snap266 := d32
					snap267 := d33
					snap268 := d68
					snap269 := d69
					snap270 := d70
					snap271 := d71
					snap272 := d114
					snap273 := d115
					snap274 := d116
					snap275 := d117
					snap276 := d118
					snap277 := d171
					snap278 := d172
					snap279 := d173
					snap280 := d174
					snap281 := d175
					snap282 := d176
					snap283 := d177
					snap284 := d178
					snap285 := d179
					snap286 := d180
					snap287 := d181
					snap288 := d182
					snap289 := d183
					snap290 := d184
					snap291 := d185
					snap292 := d186
					snap293 := d187
					snap294 := d188
					snap295 := d189
					snap296 := d191
					snap297 := d192
					snap298 := d193
					snap299 := d194
					snap300 := d195
					snap301 := d196
					snap302 := d199
					snap303 := d200
					alloc304 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps253)
					}
					ctx.RestoreAllocState(alloc304)
					d1 = snap254
					d2 = snap255
					d3 = snap256
					d4 = snap257
					d5 = snap258
					d6 = snap259
					d25 = snap260
					d26 = snap261
					d27 = snap262
					d28 = snap263
					d30 = snap264
					d31 = snap265
					d32 = snap266
					d33 = snap267
					d68 = snap268
					d69 = snap269
					d70 = snap270
					d71 = snap271
					d114 = snap272
					d115 = snap273
					d116 = snap274
					d117 = snap275
					d118 = snap276
					d171 = snap277
					d172 = snap278
					d173 = snap279
					d174 = snap280
					d175 = snap281
					d176 = snap282
					d177 = snap283
					d178 = snap284
					d179 = snap285
					d180 = snap286
					d181 = snap287
					d182 = snap288
					d183 = snap289
					d184 = snap290
					d185 = snap291
					d186 = snap292
					d187 = snap293
					d188 = snap294
					d189 = snap295
					d191 = snap296
					d192 = snap297
					d193 = snap298
					d194 = snap299
					d195 = snap300
					d196 = snap301
					d199 = snap302
					d200 = snap303
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps252)
					}
					return result
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
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
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					ctx.ReclaimUntrackedRegs()
					d305 = args[0]
					d305.ID = 0
					var d306 JITValueDesc
					if d305.Loc == LocImm {
						d306 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d305.Imm.Int())}
					} else if d305.Type == tagInt && d305.Loc == LocRegPair {
						ctx.FreeReg(d305.Reg)
						d306 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d305.Reg2}
						ctx.BindReg(d305.Reg2, &d306)
						ctx.BindReg(d305.Reg2, &d306)
					} else if d305.Type == tagInt && d305.Loc == LocReg {
						d306 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d305.Reg}
						ctx.BindReg(d305.Reg, &d306)
						ctx.BindReg(d305.Reg, &d306)
					} else {
						d306 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d305}, 1)
						d306.Type = tagInt
						ctx.BindReg(d306.Reg, &d306)
					}
					ctx.StabilizeDescForControlFlow(&d306)
					ctx.FreeDesc(&d305)
					if ps.General {
						ctx.SyncDesc(&d306)
						if d306.Loc == LocReg {
							ctx.ProtectReg(d306.Reg)
						} else if d306.Loc == LocRegPair {
							ctx.ProtectReg(d306.Reg)
							ctx.ProtectReg(d306.Reg2)
						}
						d307 = d306
						if d307.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d307)
						ctx.EmitStoreToStack(d307, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
						if d306.Loc == LocReg {
							ctx.UnprotectReg(d306.Reg)
						} else if d306.Loc == LocRegPair {
							ctx.UnprotectReg(d306.Reg)
							ctx.UnprotectReg(d306.Reg2)
						}
					}
					ps308 := PhiState{General: ps.General}
					ps308.OverlayValues = make([]JITValueDesc, 308)
					ps308.OverlayValues[1] = d1
					ps308.OverlayValues[2] = d2
					ps308.OverlayValues[3] = d3
					ps308.OverlayValues[4] = d4
					ps308.OverlayValues[5] = d5
					ps308.OverlayValues[6] = d6
					ps308.OverlayValues[25] = d25
					ps308.OverlayValues[26] = d26
					ps308.OverlayValues[27] = d27
					ps308.OverlayValues[28] = d28
					ps308.OverlayValues[30] = d30
					ps308.OverlayValues[31] = d31
					ps308.OverlayValues[32] = d32
					ps308.OverlayValues[33] = d33
					ps308.OverlayValues[68] = d68
					ps308.OverlayValues[69] = d69
					ps308.OverlayValues[70] = d70
					ps308.OverlayValues[71] = d71
					ps308.OverlayValues[114] = d114
					ps308.OverlayValues[115] = d115
					ps308.OverlayValues[116] = d116
					ps308.OverlayValues[117] = d117
					ps308.OverlayValues[118] = d118
					ps308.OverlayValues[171] = d171
					ps308.OverlayValues[172] = d172
					ps308.OverlayValues[173] = d173
					ps308.OverlayValues[174] = d174
					ps308.OverlayValues[175] = d175
					ps308.OverlayValues[176] = d176
					ps308.OverlayValues[177] = d177
					ps308.OverlayValues[178] = d178
					ps308.OverlayValues[179] = d179
					ps308.OverlayValues[180] = d180
					ps308.OverlayValues[181] = d181
					ps308.OverlayValues[182] = d182
					ps308.OverlayValues[183] = d183
					ps308.OverlayValues[184] = d184
					ps308.OverlayValues[185] = d185
					ps308.OverlayValues[186] = d186
					ps308.OverlayValues[187] = d187
					ps308.OverlayValues[188] = d188
					ps308.OverlayValues[189] = d189
					ps308.OverlayValues[191] = d191
					ps308.OverlayValues[192] = d192
					ps308.OverlayValues[193] = d193
					ps308.OverlayValues[194] = d194
					ps308.OverlayValues[195] = d195
					ps308.OverlayValues[196] = d196
					ps308.OverlayValues[199] = d199
					ps308.OverlayValues[200] = d200
					ps308.OverlayValues[305] = d305
					ps308.OverlayValues[306] = d306
					ps308.OverlayValues[307] = d307
					ps308.PhiValues = make([]JITValueDesc, 2)
					d309 = d306
					ps308.PhiValues[0] = d309
					d310 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps308.PhiValues[1] = d310
					if ps308.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps308)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
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
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != LocNone {
						d306 = ps.OverlayValues[306]
					}
					if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != LocNone {
						d307 = ps.OverlayValues[307]
					}
					if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != LocNone {
						d309 = ps.OverlayValues[309]
					}
					if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != LocNone {
						d310 = ps.OverlayValues[310]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d311 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d311.Loc == LocRegPair || d311.Loc == LocStackPair || d311.Loc == LocRegTriple || d311.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d311)
					d312 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d1, d311}, 3)
					d312.NoHeapPointer = false
					ctx.BindReg(d312.Reg, &d312)
					ctx.BindReg(d312.Reg2, &d312)
					ctx.BindReg(d312.Reg3, &d312)
					ctx.FreeDesc(&d311)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc != LocRegTriple && d312.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d312)
					d313 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d312}, 3)
					d313.NoHeapPointer = false
					ctx.BindReg(d313.Reg, &d313)
					ctx.BindReg(d313.Reg2, &d313)
					ctx.BindReg(d313.Reg3, &d313)
					ctx.FreeDesc(&d312)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc != LocRegTriple && d313.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d313)
					d314 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d313}, 1)
					d314.NoHeapPointer = true
					ctx.BindReg(d314.Reg, &d314)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc != LocRegTriple && d313.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d313)
					d315 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d313}, 1)
					d315.NoHeapPointer = true
					ctx.BindReg(d315.Reg, &d315)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc != LocRegTriple && d313.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d313)
					d316 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d313}, 1)
					d316.NoHeapPointer = true
					ctx.BindReg(d316.Reg, &d316)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc != LocRegTriple && d313.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d313)
					d317 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d313}, 1)
					d317.NoHeapPointer = true
					ctx.BindReg(d317.Reg, &d317)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc != LocRegTriple && d313.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d313)
					d318 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d313}, 1)
					d318.NoHeapPointer = true
					ctx.BindReg(d318.Reg, &d318)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					ctx.EnsureDesc(&d313)
					if d313.Loc != LocRegTriple && d313.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d313)
					d319 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d313}, 1)
					d319.NoHeapPointer = true
					ctx.BindReg(d319.Reg, &d319)
					ctx.FreeDesc(&d313)
					ctx.EnsureDesc(&d314)
					ctx.EnsureDesc(&d314)
					if d314.Loc == LocRegPair || d314.Loc == LocStackPair || d314.Loc == LocRegTriple || d314.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d315)
					ctx.EnsureDesc(&d315)
					if d315.Loc == LocRegPair || d315.Loc == LocStackPair || d315.Loc == LocRegTriple || d315.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d316)
					ctx.EnsureDesc(&d316)
					if d316.Loc == LocRegPair || d316.Loc == LocStackPair || d316.Loc == LocRegTriple || d316.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d317)
					ctx.EnsureDesc(&d317)
					if d317.Loc == LocRegPair || d317.Loc == LocStackPair || d317.Loc == LocRegTriple || d317.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d318)
					ctx.EnsureDesc(&d318)
					if d318.Loc == LocRegPair || d318.Loc == LocStackPair || d318.Loc == LocRegTriple || d318.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d319)
					ctx.EnsureDesc(&d319)
					if d319.Loc == LocRegPair || d319.Loc == LocStackPair || d319.Loc == LocRegTriple || d319.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d320 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d320.Loc == LocRegPair || d320.Loc == LocStackPair || d320.Loc == LocRegTriple || d320.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d314)
					ctx.SyncDesc(&d315)
					ctx.SyncDesc(&d316)
					ctx.SyncDesc(&d317)
					ctx.SyncDesc(&d318)
					ctx.SyncDesc(&d319)
					ctx.SyncDesc(&d320)
					ctx.SyncDesc(&d30)
					d321 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d314, d315, d316, d317, d318, d319, d320, d30}, 3)
					d321.NoHeapPointer = false
					ctx.BindReg(d321.Reg, &d321)
					ctx.BindReg(d321.Reg2, &d321)
					ctx.BindReg(d321.Reg3, &d321)
					ctx.FreeDesc(&d320)
					ctx.FreeDesc(&d314)
					ctx.FreeDesc(&d315)
					ctx.FreeDesc(&d316)
					ctx.FreeDesc(&d317)
					ctx.FreeDesc(&d318)
					ctx.FreeDesc(&d319)
					ctx.EnsureDesc(&d321)
					ctx.EnsureDesc(&d321)
					ctx.EnsureDesc(&d321)
					if d321.Loc != LocRegTriple && d321.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).UTC arg0)")
					}
					ctx.SyncDesc(&d321)
					d322 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).UTC), []JITValueDesc{d321}, 3)
					d322.NoHeapPointer = false
					ctx.BindReg(d322.Reg, &d322)
					ctx.BindReg(d322.Reg2, &d322)
					ctx.BindReg(d322.Reg3, &d322)
					ctx.FreeDesc(&d321)
					ctx.EnsureDesc(&d322)
					ctx.EnsureDesc(&d322)
					ctx.EnsureDesc(&d322)
					if d322.Loc != LocRegTriple && d322.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d322)
					d323 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d322}, 1)
					d323.NoHeapPointer = true
					ctx.BindReg(d323.Reg, &d323)
					ctx.FreeDesc(&d322)
					ctx.EnsureDesc(&d323)
					ctx.EnsureDesc(&d323)
					if d323.Loc == LocRegPair || d323.Loc == LocStackPair || d323.Loc == LocRegTriple || d323.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d323)
					d324 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d323}, 2)
					d324.NoHeapPointer = false
					ctx.BindReg(d324.Reg, &d324)
					ctx.BindReg(d324.Reg2, &d324)
					ctx.FreeDesc(&d323)
					ctx.SyncDesc(&d324)
					if d324.Loc == LocRegPair || d324.Loc == LocStackPair || d324.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d324, &result)
						result.Type = d324.Type
					} else {
						switch d324.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d324)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d324)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d324)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d324, &result)
							result.Type = d324.Type
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
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
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != LocNone {
						d306 = ps.OverlayValues[306]
					}
					if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != LocNone {
						d307 = ps.OverlayValues[307]
					}
					if len(ps.OverlayValues) > 309 && ps.OverlayValues[309].Loc != LocNone {
						d309 = ps.OverlayValues[309]
					}
					if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != LocNone {
						d310 = ps.OverlayValues[310]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
					}
					if len(ps.OverlayValues) > 312 && ps.OverlayValues[312].Loc != LocNone {
						d312 = ps.OverlayValues[312]
					}
					if len(ps.OverlayValues) > 313 && ps.OverlayValues[313].Loc != LocNone {
						d313 = ps.OverlayValues[313]
					}
					if len(ps.OverlayValues) > 314 && ps.OverlayValues[314].Loc != LocNone {
						d314 = ps.OverlayValues[314]
					}
					if len(ps.OverlayValues) > 315 && ps.OverlayValues[315].Loc != LocNone {
						d315 = ps.OverlayValues[315]
					}
					if len(ps.OverlayValues) > 316 && ps.OverlayValues[316].Loc != LocNone {
						d316 = ps.OverlayValues[316]
					}
					if len(ps.OverlayValues) > 317 && ps.OverlayValues[317].Loc != LocNone {
						d317 = ps.OverlayValues[317]
					}
					if len(ps.OverlayValues) > 318 && ps.OverlayValues[318].Loc != LocNone {
						d318 = ps.OverlayValues[318]
					}
					if len(ps.OverlayValues) > 319 && ps.OverlayValues[319].Loc != LocNone {
						d319 = ps.OverlayValues[319]
					}
					if len(ps.OverlayValues) > 320 && ps.OverlayValues[320].Loc != LocNone {
						d320 = ps.OverlayValues[320]
					}
					if len(ps.OverlayValues) > 321 && ps.OverlayValues[321].Loc != LocNone {
						d321 = ps.OverlayValues[321]
					}
					if len(ps.OverlayValues) > 322 && ps.OverlayValues[322].Loc != LocNone {
						d322 = ps.OverlayValues[322]
					}
					if len(ps.OverlayValues) > 323 && ps.OverlayValues[323].Loc != LocNone {
						d323 = ps.OverlayValues[323]
					}
					if len(ps.OverlayValues) > 324 && ps.OverlayValues[324].Loc != LocNone {
						d324 = ps.OverlayValues[324]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d325 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d325.Loc == LocRegPair || d325.Loc == LocStackPair || d325.Loc == LocRegTriple || d325.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d325)
					d326 = ctx.EmitGoCallScalar(GoFuncAddr(time.Unix), []JITValueDesc{d1, d325}, 3)
					d326.NoHeapPointer = false
					ctx.BindReg(d326.Reg, &d326)
					ctx.BindReg(d326.Reg2, &d326)
					ctx.BindReg(d326.Reg3, &d326)
					ctx.FreeDesc(&d325)
					ctx.EnsureDesc(&d326)
					ctx.EnsureDesc(&d326)
					ctx.EnsureDesc(&d326)
					if d326.Loc != LocRegTriple && d326.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).In arg0)")
					}
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d326)
					ctx.SyncDesc(&d30)
					d327 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).In), []JITValueDesc{d326, d30}, 3)
					d327.NoHeapPointer = false
					ctx.BindReg(d327.Reg, &d327)
					ctx.BindReg(d327.Reg2, &d327)
					ctx.BindReg(d327.Reg3, &d327)
					ctx.FreeDesc(&d326)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Year arg0)")
					}
					ctx.SyncDesc(&d327)
					d328 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Year), []JITValueDesc{d327}, 1)
					d328.NoHeapPointer = true
					ctx.BindReg(d328.Reg, &d328)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Month arg0)")
					}
					ctx.SyncDesc(&d327)
					d329 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Month), []JITValueDesc{d327}, 1)
					d329.NoHeapPointer = true
					ctx.BindReg(d329.Reg, &d329)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Day arg0)")
					}
					ctx.SyncDesc(&d327)
					d330 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Day), []JITValueDesc{d327}, 1)
					d330.NoHeapPointer = true
					ctx.BindReg(d330.Reg, &d330)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Hour arg0)")
					}
					ctx.SyncDesc(&d327)
					d331 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Hour), []JITValueDesc{d327}, 1)
					d331.NoHeapPointer = true
					ctx.BindReg(d331.Reg, &d331)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Minute arg0)")
					}
					ctx.SyncDesc(&d327)
					d332 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Minute), []JITValueDesc{d327}, 1)
					d332.NoHeapPointer = true
					ctx.BindReg(d332.Reg, &d332)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					ctx.EnsureDesc(&d327)
					if d327.Loc != LocRegTriple && d327.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Second arg0)")
					}
					ctx.SyncDesc(&d327)
					d333 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Second), []JITValueDesc{d327}, 1)
					d333.NoHeapPointer = true
					ctx.BindReg(d333.Reg, &d333)
					ctx.FreeDesc(&d327)
					d334 = ctx.EmitGoCallScalar(GoFuncAddr(func() *time.Location { return time.UTC }), nil, 1)
					ctx.EnsureDesc(&d328)
					ctx.EnsureDesc(&d328)
					if d328.Loc == LocRegPair || d328.Loc == LocStackPair || d328.Loc == LocRegTriple || d328.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d329)
					ctx.EnsureDesc(&d329)
					if d329.Loc == LocRegPair || d329.Loc == LocStackPair || d329.Loc == LocRegTriple || d329.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d330)
					ctx.EnsureDesc(&d330)
					if d330.Loc == LocRegPair || d330.Loc == LocStackPair || d330.Loc == LocRegTriple || d330.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d331)
					ctx.EnsureDesc(&d331)
					if d331.Loc == LocRegPair || d331.Loc == LocStackPair || d331.Loc == LocRegTriple || d331.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d332)
					ctx.EnsureDesc(&d332)
					if d332.Loc == LocRegPair || d332.Loc == LocStackPair || d332.Loc == LocRegTriple || d332.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d333)
					ctx.EnsureDesc(&d333)
					if d333.Loc == LocRegPair || d333.Loc == LocStackPair || d333.Loc == LocRegTriple || d333.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d335 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d335.Loc == LocRegPair || d335.Loc == LocStackPair || d335.Loc == LocRegTriple || d335.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d334)
					ctx.EnsureDesc(&d334)
					if d334.Loc == LocRegPair || d334.Loc == LocStackPair || d334.Loc == LocRegTriple || d334.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d328)
					ctx.SyncDesc(&d329)
					ctx.SyncDesc(&d330)
					ctx.SyncDesc(&d331)
					ctx.SyncDesc(&d332)
					ctx.SyncDesc(&d333)
					ctx.SyncDesc(&d335)
					ctx.SyncDesc(&d334)
					d336 = ctx.EmitGoCallScalar(GoFuncAddr(time.Date), []JITValueDesc{d328, d329, d330, d331, d332, d333, d335, d334}, 3)
					d336.NoHeapPointer = false
					ctx.BindReg(d336.Reg, &d336)
					ctx.BindReg(d336.Reg2, &d336)
					ctx.BindReg(d336.Reg3, &d336)
					ctx.FreeDesc(&d335)
					ctx.FreeDesc(&d328)
					ctx.FreeDesc(&d329)
					ctx.FreeDesc(&d330)
					ctx.FreeDesc(&d331)
					ctx.FreeDesc(&d332)
					ctx.FreeDesc(&d333)
					ctx.FreeDesc(&d334)
					ctx.EnsureDesc(&d336)
					ctx.EnsureDesc(&d336)
					ctx.EnsureDesc(&d336)
					if d336.Loc != LocRegTriple && d336.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Unix arg0)")
					}
					ctx.SyncDesc(&d336)
					d337 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Unix), []JITValueDesc{d336}, 1)
					d337.NoHeapPointer = true
					ctx.BindReg(d337.Reg, &d337)
					ctx.FreeDesc(&d336)
					ctx.EnsureDesc(&d337)
					ctx.EnsureDesc(&d337)
					if d337.Loc == LocRegPair || d337.Loc == LocStackPair || d337.Loc == LocRegTriple || d337.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d337)
					d338 = ctx.EmitGoCallScalar(GoFuncAddr(NewDate), []JITValueDesc{d337}, 2)
					d338.NoHeapPointer = false
					ctx.BindReg(d338.Reg, &d338)
					ctx.BindReg(d338.Reg2, &d338)
					ctx.FreeDesc(&d337)
					ctx.SyncDesc(&d338)
					if d338.Loc == LocRegPair || d338.Loc == LocStackPair || d338.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d338, &result)
						result.Type = d338.Type
					} else {
						switch d338.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d338)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d338)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d338)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d338, &result)
							result.Type = d338.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps339 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps339)
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
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
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
				var d168 JITValueDesc
				_ = d168
				var d229 JITValueDesc
				_ = d229
				var d230 JITValueDesc
				_ = d230
				var d231 JITValueDesc
				_ = d231
				var d232 JITValueDesc
				_ = d232
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
				var d238 JITValueDesc
				_ = d238
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
				var d246 JITValueDesc
				_ = d246
				var d247 JITValueDesc
				_ = d247
				var d248 JITValueDesc
				_ = d248
				var d349 JITValueDesc
				_ = d349
				var d350 JITValueDesc
				_ = d350
				var d351 JITValueDesc
				_ = d351
				var d352 JITValueDesc
				_ = d352
				var d353 JITValueDesc
				_ = d353
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
				var d648 JITValueDesc
				_ = d648
				var d649 JITValueDesc
				_ = d649
				var d650 JITValueDesc
				_ = d650
				var d651 JITValueDesc
				_ = d651
				var d652 JITValueDesc
				_ = d652
				var d653 JITValueDesc
				_ = d653
				var d654 JITValueDesc
				_ = d654
				var d655 JITValueDesc
				_ = d655
				var d656 JITValueDesc
				_ = d656
				var d657 JITValueDesc
				_ = d657
				var d658 JITValueDesc
				_ = d658
				var d659 JITValueDesc
				_ = d659
				var d660 JITValueDesc
				_ = d660
				var d838 JITValueDesc
				_ = d838
				var d839 JITValueDesc
				_ = d839
				var d840 JITValueDesc
				_ = d840
				var d842 JITValueDesc
				_ = d842
				var d843 JITValueDesc
				_ = d843
				var d844 JITValueDesc
				_ = d844
				var d845 JITValueDesc
				_ = d845
				var d846 JITValueDesc
				_ = d846
				var d847 JITValueDesc
				_ = d847
				var d848 JITValueDesc
				_ = d848
				var d849 JITValueDesc
				_ = d849
				var d850 JITValueDesc
				_ = d850
				var d851 JITValueDesc
				_ = d851
				var d852 JITValueDesc
				_ = d852
				var d853 JITValueDesc
				_ = d853
				var d854 JITValueDesc
				_ = d854
				var d1064 JITValueDesc
				_ = d1064
				var d1065 JITValueDesc
				_ = d1065
				var d1066 JITValueDesc
				_ = d1066
				var d1068 JITValueDesc
				_ = d1068
				var d1069 JITValueDesc
				_ = d1069
				var d1070 JITValueDesc
				_ = d1070
				var d1071 JITValueDesc
				_ = d1071
				var d1072 JITValueDesc
				_ = d1072
				var d1073 JITValueDesc
				_ = d1073
				var d1074 JITValueDesc
				_ = d1074
				var d1075 JITValueDesc
				_ = d1075
				var d1076 JITValueDesc
				_ = d1076
				var d1077 JITValueDesc
				_ = d1077
				var d1312 JITValueDesc
				_ = d1312
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
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 4)
					ps11.OverlayValues[0] = d0
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[3] = d3
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 4)
					ps12.OverlayValues[0] = d0
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					snap13 := d0
					snap14 := d1
					snap15 := d2
					snap16 := d3
					alloc17 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps12)
					}
					ctx.RestoreAllocState(alloc17)
					d0 = snap13
					d1 = snap14
					d2 = snap15
					d3 = snap16
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps11)
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					ctx.ReclaimUntrackedRegs()
					d19 = args[1]
					d19.ID = 0
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d19)
					d19 = JITPrepareScmerGoArg(ctx, d19)
					ctx.SyncDesc(&d19)
					callResults20 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d19}, []uint8{3, 1}, []uint8{4, 0})
					d21 = callResults20[0]
					_ = d21
					d22 = callResults20[1]
					_ = d22
					ctx.FreeDesc(&d19)
					ctx.StabilizeDescForControlFlow(&d21)
					d23 = args[2]
					d23.ID = 0
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					d23 = JITPrepareScmerGoArg(ctx, d23)
					ctx.SyncDesc(&d23)
					callResults24 := JITEmitGoCallResults(ctx, GoFuncAddr(toTime), []JITValueDesc{d23}, []uint8{3, 1}, []uint8{4, 0})
					d25 = callResults24[0]
					_ = d25
					d26 = callResults24[1]
					_ = d26
					ctx.FreeDesc(&d23)
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.StabilizeDescForControlFlow(&d26)
					d27 = d22
					ctx.EnsureDesc(&d27)
					if d27.Loc != LocImm && d27.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d27.Loc == LocImm {
						if d27.Imm.Bool() {
							if ps.General {
							}
							ps28 := PhiState{General: ps.General}
							ps28.OverlayValues = make([]JITValueDesc, 28)
							ps28.OverlayValues[0] = d0
							ps28.OverlayValues[1] = d1
							ps28.OverlayValues[2] = d2
							ps28.OverlayValues[3] = d3
							ps28.OverlayValues[18] = d18
							ps28.OverlayValues[19] = d19
							ps28.OverlayValues[21] = d21
							ps28.OverlayValues[22] = d22
							ps28.OverlayValues[23] = d23
							ps28.OverlayValues[25] = d25
							ps28.OverlayValues[26] = d26
							ps28.OverlayValues[27] = d27
							return bbs[6].RenderPS(ps28)
						}
						if ps.General {
						}
						ps29 := PhiState{General: ps.General}
						ps29.OverlayValues = make([]JITValueDesc, 28)
						ps29.OverlayValues[0] = d0
						ps29.OverlayValues[1] = d1
						ps29.OverlayValues[2] = d2
						ps29.OverlayValues[3] = d3
						ps29.OverlayValues[18] = d18
						ps29.OverlayValues[19] = d19
						ps29.OverlayValues[21] = d21
						ps29.OverlayValues[22] = d22
						ps29.OverlayValues[23] = d23
						ps29.OverlayValues[25] = d25
						ps29.OverlayValues[26] = d26
						ps29.OverlayValues[27] = d27
						return bbs[4].RenderPS(ps29)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d27.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					snap30 := d0
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d18
					snap35 := d19
					snap36 := d21
					snap37 := d22
					snap38 := d23
					snap39 := d25
					snap40 := d26
					snap41 := d27
					alloc42 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc42)
					d0 = snap30
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d18 = snap34
					d19 = snap35
					d21 = snap36
					d22 = snap37
					d23 = snap38
					d25 = snap39
					d26 = snap40
					d27 = snap41
					ctx.RestoreAllocState(alloc42)
					d0 = snap30
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d18 = snap34
					d19 = snap35
					d21 = snap36
					d22 = snap37
					d23 = snap38
					d25 = snap39
					d26 = snap40
					d27 = snap41
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 28)
					ps43.OverlayValues[0] = d0
					ps43.OverlayValues[1] = d1
					ps43.OverlayValues[2] = d2
					ps43.OverlayValues[3] = d3
					ps43.OverlayValues[18] = d18
					ps43.OverlayValues[19] = d19
					ps43.OverlayValues[21] = d21
					ps43.OverlayValues[22] = d22
					ps43.OverlayValues[23] = d23
					ps43.OverlayValues[25] = d25
					ps43.OverlayValues[26] = d26
					ps43.OverlayValues[27] = d27
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 28)
					ps44.OverlayValues[0] = d0
					ps44.OverlayValues[1] = d1
					ps44.OverlayValues[2] = d2
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[18] = d18
					ps44.OverlayValues[19] = d19
					ps44.OverlayValues[21] = d21
					ps44.OverlayValues[22] = d22
					ps44.OverlayValues[23] = d23
					ps44.OverlayValues[25] = d25
					ps44.OverlayValues[26] = d26
					ps44.OverlayValues[27] = d27
					snap45 := d0
					snap46 := d1
					snap47 := d2
					snap48 := d3
					snap49 := d18
					snap50 := d19
					snap51 := d21
					snap52 := d22
					snap53 := d23
					snap54 := d25
					snap55 := d26
					snap56 := d27
					alloc57 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps44)
					}
					ctx.RestoreAllocState(alloc57)
					d0 = snap45
					d1 = snap46
					d2 = snap47
					d3 = snap48
					d18 = snap49
					d19 = snap50
					d21 = snap51
					d22 = snap52
					d23 = snap53
					d25 = snap54
					d26 = snap55
					d27 = snap56
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps43)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					ctx.ReclaimUntrackedRegs()
					d58 = args[2]
					d58.ID = 0
					d60 = d58
					d60.ID = 0
					d59 = ctx.EmitTagEqualsBorrowed(&d60, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d58)
					d61 = d59
					ctx.EnsureDesc(&d61)
					if d61.Loc != LocImm && d61.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d61.Loc == LocImm {
						if d61.Imm.Bool() {
							if ps.General {
							}
							ps62 := PhiState{General: ps.General}
							ps62.OverlayValues = make([]JITValueDesc, 62)
							ps62.OverlayValues[0] = d0
							ps62.OverlayValues[1] = d1
							ps62.OverlayValues[2] = d2
							ps62.OverlayValues[3] = d3
							ps62.OverlayValues[18] = d18
							ps62.OverlayValues[19] = d19
							ps62.OverlayValues[21] = d21
							ps62.OverlayValues[22] = d22
							ps62.OverlayValues[23] = d23
							ps62.OverlayValues[25] = d25
							ps62.OverlayValues[26] = d26
							ps62.OverlayValues[27] = d27
							ps62.OverlayValues[58] = d58
							ps62.OverlayValues[59] = d59
							ps62.OverlayValues[60] = d60
							ps62.OverlayValues[61] = d61
							return bbs[1].RenderPS(ps62)
						}
						if ps.General {
						}
						ps63 := PhiState{General: ps.General}
						ps63.OverlayValues = make([]JITValueDesc, 62)
						ps63.OverlayValues[0] = d0
						ps63.OverlayValues[1] = d1
						ps63.OverlayValues[2] = d2
						ps63.OverlayValues[3] = d3
						ps63.OverlayValues[18] = d18
						ps63.OverlayValues[19] = d19
						ps63.OverlayValues[21] = d21
						ps63.OverlayValues[22] = d22
						ps63.OverlayValues[23] = d23
						ps63.OverlayValues[25] = d25
						ps63.OverlayValues[26] = d26
						ps63.OverlayValues[27] = d27
						ps63.OverlayValues[58] = d58
						ps63.OverlayValues[59] = d59
						ps63.OverlayValues[60] = d60
						ps63.OverlayValues[61] = d61
						return bbs[2].RenderPS(ps63)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d61.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					snap64 := d0
					snap65 := d1
					snap66 := d2
					snap67 := d3
					snap68 := d18
					snap69 := d19
					snap70 := d21
					snap71 := d22
					snap72 := d23
					snap73 := d25
					snap74 := d26
					snap75 := d27
					snap76 := d58
					snap77 := d59
					snap78 := d60
					snap79 := d61
					alloc80 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc80)
					d0 = snap64
					d1 = snap65
					d2 = snap66
					d3 = snap67
					d18 = snap68
					d19 = snap69
					d21 = snap70
					d22 = snap71
					d23 = snap72
					d25 = snap73
					d26 = snap74
					d27 = snap75
					d58 = snap76
					d59 = snap77
					d60 = snap78
					d61 = snap79
					ctx.RestoreAllocState(alloc80)
					d0 = snap64
					d1 = snap65
					d2 = snap66
					d3 = snap67
					d18 = snap68
					d19 = snap69
					d21 = snap70
					d22 = snap71
					d23 = snap72
					d25 = snap73
					d26 = snap74
					d27 = snap75
					d58 = snap76
					d59 = snap77
					d60 = snap78
					d61 = snap79
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 62)
					ps81.OverlayValues[0] = d0
					ps81.OverlayValues[1] = d1
					ps81.OverlayValues[2] = d2
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[18] = d18
					ps81.OverlayValues[19] = d19
					ps81.OverlayValues[21] = d21
					ps81.OverlayValues[22] = d22
					ps81.OverlayValues[23] = d23
					ps81.OverlayValues[25] = d25
					ps81.OverlayValues[26] = d26
					ps81.OverlayValues[27] = d27
					ps81.OverlayValues[58] = d58
					ps81.OverlayValues[59] = d59
					ps81.OverlayValues[60] = d60
					ps81.OverlayValues[61] = d61
					ps82 := PhiState{General: true}
					ps82.OverlayValues = make([]JITValueDesc, 62)
					ps82.OverlayValues[0] = d0
					ps82.OverlayValues[1] = d1
					ps82.OverlayValues[2] = d2
					ps82.OverlayValues[3] = d3
					ps82.OverlayValues[18] = d18
					ps82.OverlayValues[19] = d19
					ps82.OverlayValues[21] = d21
					ps82.OverlayValues[22] = d22
					ps82.OverlayValues[23] = d23
					ps82.OverlayValues[25] = d25
					ps82.OverlayValues[26] = d26
					ps82.OverlayValues[27] = d27
					ps82.OverlayValues[58] = d58
					ps82.OverlayValues[59] = d59
					ps82.OverlayValues[60] = d60
					ps82.OverlayValues[61] = d61
					snap83 := d0
					snap84 := d1
					snap85 := d2
					snap86 := d3
					snap87 := d18
					snap88 := d19
					snap89 := d21
					snap90 := d22
					snap91 := d23
					snap92 := d25
					snap93 := d26
					snap94 := d27
					snap95 := d58
					snap96 := d59
					snap97 := d60
					snap98 := d61
					alloc99 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps82)
					}
					ctx.RestoreAllocState(alloc99)
					d0 = snap83
					d1 = snap84
					d2 = snap85
					d3 = snap86
					d18 = snap87
					d19 = snap88
					d21 = snap89
					d22 = snap90
					d23 = snap91
					d25 = snap92
					d26 = snap93
					d27 = snap94
					d58 = snap95
					d59 = snap96
					d60 = snap97
					d61 = snap98
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps81)
					}
					return result
					ctx.FreeDesc(&d59)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					ctx.ReclaimUntrackedRegs()
					d100 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d100)
					if d100.Loc == LocRegPair || d100.Loc == LocStackPair || d100.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d100, &result)
						result.Type = d100.Type
					} else {
						switch d100.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d100)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d100)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d100)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d100, &result)
							result.Type = d100.Type
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					ctx.ReclaimUntrackedRegs()
					d101 = args[0]
					d101.ID = 0
					d103 = d101
					ctx.SyncDesc(&d103)
					if d103.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d103.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d103.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d103 = tmpScalar
					}
					d103 = JITPrepareScmerGoArg(ctx, d103)
					if d103.Loc != LocRegPair && d103.Loc != LocStackPair && d103.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d102 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d103}, 2)
					ctx.FreeDesc(&d101)
					ctx.EnsureDesc(&d102)
					ctx.EnsureDesc(&d102)
					ctx.EnsureDesc(&d102)
					if d102.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d102.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d102.Imm)
						ptrWord, _ := d102.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d102.Imm.String())))
						d102 = tmpPair
					} else if d102.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d102.Type, Reg: ctx.AllocRegExcept(d102.Reg), Reg2: ctx.AllocRegExcept(d102.Reg)}
						switch d102.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d102)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d102)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d102)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d102)
						d102 = tmpPair
					}
					if d102.Loc != LocRegPair && d102.Loc != LocStackPair && d102.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d102)
					d104 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d102}, 2)
					d104.NoHeapPointer = false
					ctx.BindReg(d104.Reg, &d104)
					ctx.BindReg(d104.Reg2, &d104)
					ctx.StabilizeDescForControlFlow(&d104)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocRegTriple && d25.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg0)")
					}
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocRegTriple && d21.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Sub arg1)")
					}
					ctx.SyncDesc(&d25)
					ctx.SyncDesc(&d21)
					d105 = ctx.EmitGoCallScalar(GoFuncAddr((time.Time).Sub), []JITValueDesc{d25, d21}, 1)
					d105.NoHeapPointer = true
					ctx.BindReg(d105.Reg, &d105)
					ctx.StabilizeDescForControlFlow(&d105)
					ctx.EnsureDesc(&d104)
					d106 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("SECOND")}
					var d107 JITValueDesc
					if d106.Loc == LocImm {
						ctx.TrackImm(d106.Imm)
						ptrWord, _ := d106.Imm.RawWords()
						d107 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d107.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d107.Reg2, uint64(len(d106.Imm.String())))
						ctx.BindReg(d107.Reg, &d107)
						ctx.BindReg(d107.Reg2, &d107)
					} else {
						d107 = d106
					}
					d108 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d107}, 1)
					ctx.EmitAndRegImm32(d108.Reg, 1)
					d108.Type = tagBool
					ctx.BindReg(d108.Reg, &d108)
					d109 = d108
					ctx.EnsureDesc(&d109)
					if d109.Loc != LocImm && d109.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d109.Loc == LocImm {
						if d109.Imm.Bool() {
							if ps.General {
							}
							ps110 := PhiState{General: ps.General}
							ps110.OverlayValues = make([]JITValueDesc, 110)
							ps110.OverlayValues[0] = d0
							ps110.OverlayValues[1] = d1
							ps110.OverlayValues[2] = d2
							ps110.OverlayValues[3] = d3
							ps110.OverlayValues[18] = d18
							ps110.OverlayValues[19] = d19
							ps110.OverlayValues[21] = d21
							ps110.OverlayValues[22] = d22
							ps110.OverlayValues[23] = d23
							ps110.OverlayValues[25] = d25
							ps110.OverlayValues[26] = d26
							ps110.OverlayValues[27] = d27
							ps110.OverlayValues[58] = d58
							ps110.OverlayValues[59] = d59
							ps110.OverlayValues[60] = d60
							ps110.OverlayValues[61] = d61
							ps110.OverlayValues[100] = d100
							ps110.OverlayValues[101] = d101
							ps110.OverlayValues[102] = d102
							ps110.OverlayValues[103] = d103
							ps110.OverlayValues[104] = d104
							ps110.OverlayValues[105] = d105
							ps110.OverlayValues[106] = d106
							ps110.OverlayValues[107] = d107
							ps110.OverlayValues[108] = d108
							ps110.OverlayValues[109] = d109
							return bbs[7].RenderPS(ps110)
						}
						if ps.General {
						}
						ps111 := PhiState{General: ps.General}
						ps111.OverlayValues = make([]JITValueDesc, 110)
						ps111.OverlayValues[0] = d0
						ps111.OverlayValues[1] = d1
						ps111.OverlayValues[2] = d2
						ps111.OverlayValues[3] = d3
						ps111.OverlayValues[18] = d18
						ps111.OverlayValues[19] = d19
						ps111.OverlayValues[21] = d21
						ps111.OverlayValues[22] = d22
						ps111.OverlayValues[23] = d23
						ps111.OverlayValues[25] = d25
						ps111.OverlayValues[26] = d26
						ps111.OverlayValues[27] = d27
						ps111.OverlayValues[58] = d58
						ps111.OverlayValues[59] = d59
						ps111.OverlayValues[60] = d60
						ps111.OverlayValues[61] = d61
						ps111.OverlayValues[100] = d100
						ps111.OverlayValues[101] = d101
						ps111.OverlayValues[102] = d102
						ps111.OverlayValues[103] = d103
						ps111.OverlayValues[104] = d104
						ps111.OverlayValues[105] = d105
						ps111.OverlayValues[106] = d106
						ps111.OverlayValues[107] = d107
						ps111.OverlayValues[108] = d108
						ps111.OverlayValues[109] = d109
						return bbs[9].RenderPS(ps111)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d109.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					snap112 := d0
					snap113 := d1
					snap114 := d2
					snap115 := d3
					snap116 := d18
					snap117 := d19
					snap118 := d21
					snap119 := d22
					snap120 := d23
					snap121 := d25
					snap122 := d26
					snap123 := d27
					snap124 := d58
					snap125 := d59
					snap126 := d60
					snap127 := d61
					snap128 := d100
					snap129 := d101
					snap130 := d102
					snap131 := d103
					snap132 := d104
					snap133 := d105
					snap134 := d106
					snap135 := d107
					snap136 := d108
					snap137 := d109
					alloc138 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc138)
					d0 = snap112
					d1 = snap113
					d2 = snap114
					d3 = snap115
					d18 = snap116
					d19 = snap117
					d21 = snap118
					d22 = snap119
					d23 = snap120
					d25 = snap121
					d26 = snap122
					d27 = snap123
					d58 = snap124
					d59 = snap125
					d60 = snap126
					d61 = snap127
					d100 = snap128
					d101 = snap129
					d102 = snap130
					d103 = snap131
					d104 = snap132
					d105 = snap133
					d106 = snap134
					d107 = snap135
					d108 = snap136
					d109 = snap137
					ctx.RestoreAllocState(alloc138)
					d0 = snap112
					d1 = snap113
					d2 = snap114
					d3 = snap115
					d18 = snap116
					d19 = snap117
					d21 = snap118
					d22 = snap119
					d23 = snap120
					d25 = snap121
					d26 = snap122
					d27 = snap123
					d58 = snap124
					d59 = snap125
					d60 = snap126
					d61 = snap127
					d100 = snap128
					d101 = snap129
					d102 = snap130
					d103 = snap131
					d104 = snap132
					d105 = snap133
					d106 = snap134
					d107 = snap135
					d108 = snap136
					d109 = snap137
					ps139 := PhiState{General: true}
					ps139.OverlayValues = make([]JITValueDesc, 110)
					ps139.OverlayValues[0] = d0
					ps139.OverlayValues[1] = d1
					ps139.OverlayValues[2] = d2
					ps139.OverlayValues[3] = d3
					ps139.OverlayValues[18] = d18
					ps139.OverlayValues[19] = d19
					ps139.OverlayValues[21] = d21
					ps139.OverlayValues[22] = d22
					ps139.OverlayValues[23] = d23
					ps139.OverlayValues[25] = d25
					ps139.OverlayValues[26] = d26
					ps139.OverlayValues[27] = d27
					ps139.OverlayValues[58] = d58
					ps139.OverlayValues[59] = d59
					ps139.OverlayValues[60] = d60
					ps139.OverlayValues[61] = d61
					ps139.OverlayValues[100] = d100
					ps139.OverlayValues[101] = d101
					ps139.OverlayValues[102] = d102
					ps139.OverlayValues[103] = d103
					ps139.OverlayValues[104] = d104
					ps139.OverlayValues[105] = d105
					ps139.OverlayValues[106] = d106
					ps139.OverlayValues[107] = d107
					ps139.OverlayValues[108] = d108
					ps139.OverlayValues[109] = d109
					ps140 := PhiState{General: true}
					ps140.OverlayValues = make([]JITValueDesc, 110)
					ps140.OverlayValues[0] = d0
					ps140.OverlayValues[1] = d1
					ps140.OverlayValues[2] = d2
					ps140.OverlayValues[3] = d3
					ps140.OverlayValues[18] = d18
					ps140.OverlayValues[19] = d19
					ps140.OverlayValues[21] = d21
					ps140.OverlayValues[22] = d22
					ps140.OverlayValues[23] = d23
					ps140.OverlayValues[25] = d25
					ps140.OverlayValues[26] = d26
					ps140.OverlayValues[27] = d27
					ps140.OverlayValues[58] = d58
					ps140.OverlayValues[59] = d59
					ps140.OverlayValues[60] = d60
					ps140.OverlayValues[61] = d61
					ps140.OverlayValues[100] = d100
					ps140.OverlayValues[101] = d101
					ps140.OverlayValues[102] = d102
					ps140.OverlayValues[103] = d103
					ps140.OverlayValues[104] = d104
					ps140.OverlayValues[105] = d105
					ps140.OverlayValues[106] = d106
					ps140.OverlayValues[107] = d107
					ps140.OverlayValues[108] = d108
					ps140.OverlayValues[109] = d109
					snap141 := d0
					snap142 := d1
					snap143 := d2
					snap144 := d3
					snap145 := d18
					snap146 := d19
					snap147 := d21
					snap148 := d22
					snap149 := d23
					snap150 := d25
					snap151 := d26
					snap152 := d27
					snap153 := d58
					snap154 := d59
					snap155 := d60
					snap156 := d61
					snap157 := d100
					snap158 := d101
					snap159 := d102
					snap160 := d103
					snap161 := d104
					snap162 := d105
					snap163 := d106
					snap164 := d107
					snap165 := d108
					snap166 := d109
					alloc167 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps140)
					}
					ctx.RestoreAllocState(alloc167)
					d0 = snap141
					d1 = snap142
					d2 = snap143
					d3 = snap144
					d18 = snap145
					d19 = snap146
					d21 = snap147
					d22 = snap148
					d23 = snap149
					d25 = snap150
					d26 = snap151
					d27 = snap152
					d58 = snap153
					d59 = snap154
					d60 = snap155
					d61 = snap156
					d100 = snap157
					d101 = snap158
					d102 = snap159
					d103 = snap160
					d104 = snap161
					d105 = snap162
					d106 = snap163
					d107 = snap164
					d108 = snap165
					d109 = snap166
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps139)
					}
					return result
					ctx.FreeDesc(&d108)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					ctx.ReclaimUntrackedRegs()
					d168 = d26
					ctx.EnsureDesc(&d168)
					if d168.Loc != LocImm && d168.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d168.Loc == LocImm {
						if d168.Imm.Bool() {
							if ps.General {
							}
							ps169 := PhiState{General: ps.General}
							ps169.OverlayValues = make([]JITValueDesc, 169)
							ps169.OverlayValues[0] = d0
							ps169.OverlayValues[1] = d1
							ps169.OverlayValues[2] = d2
							ps169.OverlayValues[3] = d3
							ps169.OverlayValues[18] = d18
							ps169.OverlayValues[19] = d19
							ps169.OverlayValues[21] = d21
							ps169.OverlayValues[22] = d22
							ps169.OverlayValues[23] = d23
							ps169.OverlayValues[25] = d25
							ps169.OverlayValues[26] = d26
							ps169.OverlayValues[27] = d27
							ps169.OverlayValues[58] = d58
							ps169.OverlayValues[59] = d59
							ps169.OverlayValues[60] = d60
							ps169.OverlayValues[61] = d61
							ps169.OverlayValues[100] = d100
							ps169.OverlayValues[101] = d101
							ps169.OverlayValues[102] = d102
							ps169.OverlayValues[103] = d103
							ps169.OverlayValues[104] = d104
							ps169.OverlayValues[105] = d105
							ps169.OverlayValues[106] = d106
							ps169.OverlayValues[107] = d107
							ps169.OverlayValues[108] = d108
							ps169.OverlayValues[109] = d109
							ps169.OverlayValues[168] = d168
							return bbs[5].RenderPS(ps169)
						}
						if ps.General {
						}
						ps170 := PhiState{General: ps.General}
						ps170.OverlayValues = make([]JITValueDesc, 169)
						ps170.OverlayValues[0] = d0
						ps170.OverlayValues[1] = d1
						ps170.OverlayValues[2] = d2
						ps170.OverlayValues[3] = d3
						ps170.OverlayValues[18] = d18
						ps170.OverlayValues[19] = d19
						ps170.OverlayValues[21] = d21
						ps170.OverlayValues[22] = d22
						ps170.OverlayValues[23] = d23
						ps170.OverlayValues[25] = d25
						ps170.OverlayValues[26] = d26
						ps170.OverlayValues[27] = d27
						ps170.OverlayValues[58] = d58
						ps170.OverlayValues[59] = d59
						ps170.OverlayValues[60] = d60
						ps170.OverlayValues[61] = d61
						ps170.OverlayValues[100] = d100
						ps170.OverlayValues[101] = d101
						ps170.OverlayValues[102] = d102
						ps170.OverlayValues[103] = d103
						ps170.OverlayValues[104] = d104
						ps170.OverlayValues[105] = d105
						ps170.OverlayValues[106] = d106
						ps170.OverlayValues[107] = d107
						ps170.OverlayValues[108] = d108
						ps170.OverlayValues[109] = d109
						ps170.OverlayValues[168] = d168
						return bbs[4].RenderPS(ps170)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d168.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					snap171 := d0
					snap172 := d1
					snap173 := d2
					snap174 := d3
					snap175 := d18
					snap176 := d19
					snap177 := d21
					snap178 := d22
					snap179 := d23
					snap180 := d25
					snap181 := d26
					snap182 := d27
					snap183 := d58
					snap184 := d59
					snap185 := d60
					snap186 := d61
					snap187 := d100
					snap188 := d101
					snap189 := d102
					snap190 := d103
					snap191 := d104
					snap192 := d105
					snap193 := d106
					snap194 := d107
					snap195 := d108
					snap196 := d109
					snap197 := d168
					alloc198 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc198)
					d0 = snap171
					d1 = snap172
					d2 = snap173
					d3 = snap174
					d18 = snap175
					d19 = snap176
					d21 = snap177
					d22 = snap178
					d23 = snap179
					d25 = snap180
					d26 = snap181
					d27 = snap182
					d58 = snap183
					d59 = snap184
					d60 = snap185
					d61 = snap186
					d100 = snap187
					d101 = snap188
					d102 = snap189
					d103 = snap190
					d104 = snap191
					d105 = snap192
					d106 = snap193
					d107 = snap194
					d108 = snap195
					d109 = snap196
					d168 = snap197
					ctx.RestoreAllocState(alloc198)
					d0 = snap171
					d1 = snap172
					d2 = snap173
					d3 = snap174
					d18 = snap175
					d19 = snap176
					d21 = snap177
					d22 = snap178
					d23 = snap179
					d25 = snap180
					d26 = snap181
					d27 = snap182
					d58 = snap183
					d59 = snap184
					d60 = snap185
					d61 = snap186
					d100 = snap187
					d101 = snap188
					d102 = snap189
					d103 = snap190
					d104 = snap191
					d105 = snap192
					d106 = snap193
					d107 = snap194
					d108 = snap195
					d109 = snap196
					d168 = snap197
					ps199 := PhiState{General: true}
					ps199.OverlayValues = make([]JITValueDesc, 169)
					ps199.OverlayValues[0] = d0
					ps199.OverlayValues[1] = d1
					ps199.OverlayValues[2] = d2
					ps199.OverlayValues[3] = d3
					ps199.OverlayValues[18] = d18
					ps199.OverlayValues[19] = d19
					ps199.OverlayValues[21] = d21
					ps199.OverlayValues[22] = d22
					ps199.OverlayValues[23] = d23
					ps199.OverlayValues[25] = d25
					ps199.OverlayValues[26] = d26
					ps199.OverlayValues[27] = d27
					ps199.OverlayValues[58] = d58
					ps199.OverlayValues[59] = d59
					ps199.OverlayValues[60] = d60
					ps199.OverlayValues[61] = d61
					ps199.OverlayValues[100] = d100
					ps199.OverlayValues[101] = d101
					ps199.OverlayValues[102] = d102
					ps199.OverlayValues[103] = d103
					ps199.OverlayValues[104] = d104
					ps199.OverlayValues[105] = d105
					ps199.OverlayValues[106] = d106
					ps199.OverlayValues[107] = d107
					ps199.OverlayValues[108] = d108
					ps199.OverlayValues[109] = d109
					ps199.OverlayValues[168] = d168
					ps200 := PhiState{General: true}
					ps200.OverlayValues = make([]JITValueDesc, 169)
					ps200.OverlayValues[0] = d0
					ps200.OverlayValues[1] = d1
					ps200.OverlayValues[2] = d2
					ps200.OverlayValues[3] = d3
					ps200.OverlayValues[18] = d18
					ps200.OverlayValues[19] = d19
					ps200.OverlayValues[21] = d21
					ps200.OverlayValues[22] = d22
					ps200.OverlayValues[23] = d23
					ps200.OverlayValues[25] = d25
					ps200.OverlayValues[26] = d26
					ps200.OverlayValues[27] = d27
					ps200.OverlayValues[58] = d58
					ps200.OverlayValues[59] = d59
					ps200.OverlayValues[60] = d60
					ps200.OverlayValues[61] = d61
					ps200.OverlayValues[100] = d100
					ps200.OverlayValues[101] = d101
					ps200.OverlayValues[102] = d102
					ps200.OverlayValues[103] = d103
					ps200.OverlayValues[104] = d104
					ps200.OverlayValues[105] = d105
					ps200.OverlayValues[106] = d106
					ps200.OverlayValues[107] = d107
					ps200.OverlayValues[108] = d108
					ps200.OverlayValues[109] = d109
					ps200.OverlayValues[168] = d168
					snap201 := d0
					snap202 := d1
					snap203 := d2
					snap204 := d3
					snap205 := d18
					snap206 := d19
					snap207 := d21
					snap208 := d22
					snap209 := d23
					snap210 := d25
					snap211 := d26
					snap212 := d27
					snap213 := d58
					snap214 := d59
					snap215 := d60
					snap216 := d61
					snap217 := d100
					snap218 := d101
					snap219 := d102
					snap220 := d103
					snap221 := d104
					snap222 := d105
					snap223 := d106
					snap224 := d107
					snap225 := d108
					snap226 := d109
					snap227 := d168
					alloc228 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps200)
					}
					ctx.RestoreAllocState(alloc228)
					d0 = snap201
					d1 = snap202
					d2 = snap203
					d3 = snap204
					d18 = snap205
					d19 = snap206
					d21 = snap207
					d22 = snap208
					d23 = snap209
					d25 = snap210
					d26 = snap211
					d27 = snap212
					d58 = snap213
					d59 = snap214
					d60 = snap215
					d61 = snap216
					d100 = snap217
					d101 = snap218
					d102 = snap219
					d103 = snap220
					d104 = snap221
					d105 = snap222
					d106 = snap223
					d107 = snap224
					d108 = snap225
					d109 = snap226
					d168 = snap227
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps199)
					}
					return result
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl22 := ctx.ReserveLabel()
					_ = lbl22
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d229 JITValueDesc
					if d105.Loc == LocImm {
						d229 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() / 1000000000)}
					} else {
						r0 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r0, d105.Reg)
						ctx.EmitIdivRegImm(r0, 1000000000)
						d229 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d229)
					}
					if d229.Loc == LocReg && d105.Loc == LocReg && d229.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d230 JITValueDesc
					if d105.Loc == LocImm {
						d230 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() % 1000000000)}
					} else {
						ctx.EmitIremRegImm(d105.Reg, 1000000000)
						d230 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d105.Reg}
						ctx.BindReg(d105.Reg, &d230)
					}
					if d230.Loc == LocReg && d105.Loc == LocReg && d230.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d229)
					ctx.EnsureDesc(&d229)
					var d231 JITValueDesc
					if d229.Loc == LocImm {
						d231 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d229.Imm.Int()))}
					} else {
						r1 := ctx.AllocRegExcept(d229.Reg)
						ctx.EmitMovRegReg(r1, d229.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r1)
						d231 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d231)
					}
					ctx.FreeDesc(&d229)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d230)
					ctx.EnsureDesc(&d230)
					var d232 JITValueDesc
					if d230.Loc == LocImm {
						d232 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d230.Imm.Int()))}
					} else {
						r2 := ctx.AllocRegExcept(d230.Reg)
						ctx.EmitMovRegReg(r2, d230.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r2)
						d232 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d232)
					}
					ctx.FreeDesc(&d230)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d232)
					var d233 JITValueDesc
					if d232.Loc == LocImm {
						d233 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d232.Imm.Float() / 1e+09)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4741671816366391296))
						ctx.EmitDivFloat64(d232.Reg, RegR11)
						d233 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d232.Reg}
						ctx.BindReg(d232.Reg, &d233)
					}
					if d233.Loc == LocReg && d232.Loc == LocReg && d233.Reg == d232.Reg {
						ctx.TransferReg(d232.Reg)
						d232.Loc = LocNone
					}
					ctx.FreeDesc(&d232)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d231)
					ctx.EnsureDesc(&d233)
					ctx.EnsureDescsTogether(&d231, &d233)
					var d234 JITValueDesc
					if d231.Loc == LocImm && d233.Loc == LocImm {
						d234 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d231.Imm.Float() + d233.Imm.Float())}
					} else if d231.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d233.Reg)
						_, xBits := d231.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d233.Reg)
						d234 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d234)
					} else if d233.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d231.Reg)
						ctx.EmitMovRegReg(scratch, d231.Reg)
						_, yBits := d233.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d234 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d234)
					} else {
						r3 := ctx.AllocRegExcept(d231.Reg, d233.Reg)
						ctx.EmitMovRegReg(r3, d231.Reg)
						ctx.EmitAddFloat64(r3, d233.Reg)
						d234 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r3}
						ctx.BindReg(r3, &d234)
					}
					if d234.Loc == LocReg && d231.Loc == LocReg && d234.Reg == d231.Reg {
						ctx.TransferReg(d231.Reg)
						d231.Loc = LocNone
					}
					ctx.FreeDesc(&d231)
					ctx.FreeDesc(&d233)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d234)
					ctx.EnsureDesc(&d234)
					ctx.EnsureDesc(&d234)
					var d235 JITValueDesc
					if d234.Loc == LocImm {
						d235 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d234.Imm.Float()))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r4, d234.Reg)
						d235 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d235)
					}
					ctx.FreeDesc(&d234)
					ctx.EnsureDesc(&d235)
					if d235.Loc == LocImm {
						ctx.EmitMakeInt(result, d235)
					} else {
						ctx.EmitMovToReg(result.Reg2, d235)
						d236 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d236)
						if d235.Loc == LocReg && d235.Reg != result.Reg2 {
							ctx.FreeReg(d235.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d237 JITValueDesc
					if d105.Loc == LocImm {
						d237 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() / 60000000000)}
					} else {
						r5 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r5, d105.Reg)
						ctx.EmitIdivRegImm(r5, 60000000000)
						d237 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d237)
					}
					if d237.Loc == LocReg && d105.Loc == LocReg && d237.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d238 JITValueDesc
					if d105.Loc == LocImm {
						d238 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() % 60000000000)}
					} else {
						ctx.EmitIremRegImm(d105.Reg, 60000000000)
						d238 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d105.Reg}
						ctx.BindReg(d105.Reg, &d238)
					}
					if d238.Loc == LocReg && d105.Loc == LocReg && d238.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d237)
					ctx.EnsureDesc(&d237)
					var d239 JITValueDesc
					if d237.Loc == LocImm {
						d239 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d237.Imm.Int()))}
					} else {
						r6 := ctx.AllocRegExcept(d237.Reg)
						ctx.EmitMovRegReg(r6, d237.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r6)
						d239 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
						ctx.BindReg(r6, &d239)
					}
					ctx.FreeDesc(&d237)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d238)
					ctx.EnsureDesc(&d238)
					var d240 JITValueDesc
					if d238.Loc == LocImm {
						d240 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d238.Imm.Int()))}
					} else {
						r7 := ctx.AllocRegExcept(d238.Reg)
						ctx.EmitMovRegReg(r7, d238.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r7)
						d240 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d240)
					}
					ctx.FreeDesc(&d238)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d240)
					var d241 JITValueDesc
					if d240.Loc == LocImm {
						d241 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d240.Imm.Float() / 6e+10)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4768169126130614272))
						ctx.EmitDivFloat64(d240.Reg, RegR11)
						d241 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d240.Reg}
						ctx.BindReg(d240.Reg, &d241)
					}
					if d241.Loc == LocReg && d240.Loc == LocReg && d241.Reg == d240.Reg {
						ctx.TransferReg(d240.Reg)
						d240.Loc = LocNone
					}
					ctx.FreeDesc(&d240)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d239)
					ctx.EnsureDesc(&d241)
					ctx.EnsureDescsTogether(&d239, &d241)
					var d242 JITValueDesc
					if d239.Loc == LocImm && d241.Loc == LocImm {
						d242 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d239.Imm.Float() + d241.Imm.Float())}
					} else if d239.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d241.Reg)
						_, xBits := d239.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d241.Reg)
						d242 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d242)
					} else if d241.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d239.Reg)
						ctx.EmitMovRegReg(scratch, d239.Reg)
						_, yBits := d241.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d242 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d242)
					} else {
						r8 := ctx.AllocRegExcept(d239.Reg, d241.Reg)
						ctx.EmitMovRegReg(r8, d239.Reg)
						ctx.EmitAddFloat64(r8, d241.Reg)
						d242 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
						ctx.BindReg(r8, &d242)
					}
					if d242.Loc == LocReg && d239.Loc == LocReg && d242.Reg == d239.Reg {
						ctx.TransferReg(d239.Reg)
						d239.Loc = LocNone
					}
					ctx.FreeDesc(&d239)
					ctx.FreeDesc(&d241)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d242)
					ctx.EnsureDesc(&d242)
					ctx.EnsureDesc(&d242)
					var d243 JITValueDesc
					if d242.Loc == LocImm {
						d243 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d242.Imm.Float()))}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r9, d242.Reg)
						d243 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d243)
					}
					ctx.FreeDesc(&d242)
					ctx.EnsureDesc(&d243)
					if d243.Loc == LocImm {
						ctx.EmitMakeInt(result, d243)
					} else {
						ctx.EmitMovToReg(result.Reg2, d243)
						d244 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d244)
						if d243.Loc == LocReg && d243.Reg != result.Reg2 {
							ctx.FreeReg(d243.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					d245 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MINUTE")}
					var d246 JITValueDesc
					if d245.Loc == LocImm {
						ctx.TrackImm(d245.Imm)
						ptrWord, _ := d245.Imm.RawWords()
						d246 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d246.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d246.Reg2, uint64(len(d245.Imm.String())))
						ctx.BindReg(d246.Reg, &d246)
						ctx.BindReg(d246.Reg2, &d246)
					} else {
						d246 = d245
					}
					d247 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d246}, 1)
					ctx.EmitAndRegImm32(d247.Reg, 1)
					d247.Type = tagBool
					ctx.BindReg(d247.Reg, &d247)
					d248 = d247
					ctx.EnsureDesc(&d248)
					if d248.Loc != LocImm && d248.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d248.Loc == LocImm {
						if d248.Imm.Bool() {
							if ps.General {
							}
							ps249 := PhiState{General: ps.General}
							ps249.OverlayValues = make([]JITValueDesc, 249)
							ps249.OverlayValues[0] = d0
							ps249.OverlayValues[1] = d1
							ps249.OverlayValues[2] = d2
							ps249.OverlayValues[3] = d3
							ps249.OverlayValues[18] = d18
							ps249.OverlayValues[19] = d19
							ps249.OverlayValues[21] = d21
							ps249.OverlayValues[22] = d22
							ps249.OverlayValues[23] = d23
							ps249.OverlayValues[25] = d25
							ps249.OverlayValues[26] = d26
							ps249.OverlayValues[27] = d27
							ps249.OverlayValues[58] = d58
							ps249.OverlayValues[59] = d59
							ps249.OverlayValues[60] = d60
							ps249.OverlayValues[61] = d61
							ps249.OverlayValues[100] = d100
							ps249.OverlayValues[101] = d101
							ps249.OverlayValues[102] = d102
							ps249.OverlayValues[103] = d103
							ps249.OverlayValues[104] = d104
							ps249.OverlayValues[105] = d105
							ps249.OverlayValues[106] = d106
							ps249.OverlayValues[107] = d107
							ps249.OverlayValues[108] = d108
							ps249.OverlayValues[109] = d109
							ps249.OverlayValues[168] = d168
							ps249.OverlayValues[229] = d229
							ps249.OverlayValues[230] = d230
							ps249.OverlayValues[231] = d231
							ps249.OverlayValues[232] = d232
							ps249.OverlayValues[233] = d233
							ps249.OverlayValues[234] = d234
							ps249.OverlayValues[235] = d235
							ps249.OverlayValues[236] = d236
							ps249.OverlayValues[237] = d237
							ps249.OverlayValues[238] = d238
							ps249.OverlayValues[239] = d239
							ps249.OverlayValues[240] = d240
							ps249.OverlayValues[241] = d241
							ps249.OverlayValues[242] = d242
							ps249.OverlayValues[243] = d243
							ps249.OverlayValues[244] = d244
							ps249.OverlayValues[245] = d245
							ps249.OverlayValues[246] = d246
							ps249.OverlayValues[247] = d247
							ps249.OverlayValues[248] = d248
							return bbs[8].RenderPS(ps249)
						}
						if ps.General {
						}
						ps250 := PhiState{General: ps.General}
						ps250.OverlayValues = make([]JITValueDesc, 249)
						ps250.OverlayValues[0] = d0
						ps250.OverlayValues[1] = d1
						ps250.OverlayValues[2] = d2
						ps250.OverlayValues[3] = d3
						ps250.OverlayValues[18] = d18
						ps250.OverlayValues[19] = d19
						ps250.OverlayValues[21] = d21
						ps250.OverlayValues[22] = d22
						ps250.OverlayValues[23] = d23
						ps250.OverlayValues[25] = d25
						ps250.OverlayValues[26] = d26
						ps250.OverlayValues[27] = d27
						ps250.OverlayValues[58] = d58
						ps250.OverlayValues[59] = d59
						ps250.OverlayValues[60] = d60
						ps250.OverlayValues[61] = d61
						ps250.OverlayValues[100] = d100
						ps250.OverlayValues[101] = d101
						ps250.OverlayValues[102] = d102
						ps250.OverlayValues[103] = d103
						ps250.OverlayValues[104] = d104
						ps250.OverlayValues[105] = d105
						ps250.OverlayValues[106] = d106
						ps250.OverlayValues[107] = d107
						ps250.OverlayValues[108] = d108
						ps250.OverlayValues[109] = d109
						ps250.OverlayValues[168] = d168
						ps250.OverlayValues[229] = d229
						ps250.OverlayValues[230] = d230
						ps250.OverlayValues[231] = d231
						ps250.OverlayValues[232] = d232
						ps250.OverlayValues[233] = d233
						ps250.OverlayValues[234] = d234
						ps250.OverlayValues[235] = d235
						ps250.OverlayValues[236] = d236
						ps250.OverlayValues[237] = d237
						ps250.OverlayValues[238] = d238
						ps250.OverlayValues[239] = d239
						ps250.OverlayValues[240] = d240
						ps250.OverlayValues[241] = d241
						ps250.OverlayValues[242] = d242
						ps250.OverlayValues[243] = d243
						ps250.OverlayValues[244] = d244
						ps250.OverlayValues[245] = d245
						ps250.OverlayValues[246] = d246
						ps250.OverlayValues[247] = d247
						ps250.OverlayValues[248] = d248
						return bbs[11].RenderPS(ps250)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d248.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					snap251 := d0
					snap252 := d1
					snap253 := d2
					snap254 := d3
					snap255 := d18
					snap256 := d19
					snap257 := d21
					snap258 := d22
					snap259 := d23
					snap260 := d25
					snap261 := d26
					snap262 := d27
					snap263 := d58
					snap264 := d59
					snap265 := d60
					snap266 := d61
					snap267 := d100
					snap268 := d101
					snap269 := d102
					snap270 := d103
					snap271 := d104
					snap272 := d105
					snap273 := d106
					snap274 := d107
					snap275 := d108
					snap276 := d109
					snap277 := d168
					snap278 := d229
					snap279 := d230
					snap280 := d231
					snap281 := d232
					snap282 := d233
					snap283 := d234
					snap284 := d235
					snap285 := d236
					snap286 := d237
					snap287 := d238
					snap288 := d239
					snap289 := d240
					snap290 := d241
					snap291 := d242
					snap292 := d243
					snap293 := d244
					snap294 := d245
					snap295 := d246
					snap296 := d247
					snap297 := d248
					alloc298 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc298)
					d0 = snap251
					d1 = snap252
					d2 = snap253
					d3 = snap254
					d18 = snap255
					d19 = snap256
					d21 = snap257
					d22 = snap258
					d23 = snap259
					d25 = snap260
					d26 = snap261
					d27 = snap262
					d58 = snap263
					d59 = snap264
					d60 = snap265
					d61 = snap266
					d100 = snap267
					d101 = snap268
					d102 = snap269
					d103 = snap270
					d104 = snap271
					d105 = snap272
					d106 = snap273
					d107 = snap274
					d108 = snap275
					d109 = snap276
					d168 = snap277
					d229 = snap278
					d230 = snap279
					d231 = snap280
					d232 = snap281
					d233 = snap282
					d234 = snap283
					d235 = snap284
					d236 = snap285
					d237 = snap286
					d238 = snap287
					d239 = snap288
					d240 = snap289
					d241 = snap290
					d242 = snap291
					d243 = snap292
					d244 = snap293
					d245 = snap294
					d246 = snap295
					d247 = snap296
					d248 = snap297
					ctx.RestoreAllocState(alloc298)
					d0 = snap251
					d1 = snap252
					d2 = snap253
					d3 = snap254
					d18 = snap255
					d19 = snap256
					d21 = snap257
					d22 = snap258
					d23 = snap259
					d25 = snap260
					d26 = snap261
					d27 = snap262
					d58 = snap263
					d59 = snap264
					d60 = snap265
					d61 = snap266
					d100 = snap267
					d101 = snap268
					d102 = snap269
					d103 = snap270
					d104 = snap271
					d105 = snap272
					d106 = snap273
					d107 = snap274
					d108 = snap275
					d109 = snap276
					d168 = snap277
					d229 = snap278
					d230 = snap279
					d231 = snap280
					d232 = snap281
					d233 = snap282
					d234 = snap283
					d235 = snap284
					d236 = snap285
					d237 = snap286
					d238 = snap287
					d239 = snap288
					d240 = snap289
					d241 = snap290
					d242 = snap291
					d243 = snap292
					d244 = snap293
					d245 = snap294
					d246 = snap295
					d247 = snap296
					d248 = snap297
					ps299 := PhiState{General: true}
					ps299.OverlayValues = make([]JITValueDesc, 249)
					ps299.OverlayValues[0] = d0
					ps299.OverlayValues[1] = d1
					ps299.OverlayValues[2] = d2
					ps299.OverlayValues[3] = d3
					ps299.OverlayValues[18] = d18
					ps299.OverlayValues[19] = d19
					ps299.OverlayValues[21] = d21
					ps299.OverlayValues[22] = d22
					ps299.OverlayValues[23] = d23
					ps299.OverlayValues[25] = d25
					ps299.OverlayValues[26] = d26
					ps299.OverlayValues[27] = d27
					ps299.OverlayValues[58] = d58
					ps299.OverlayValues[59] = d59
					ps299.OverlayValues[60] = d60
					ps299.OverlayValues[61] = d61
					ps299.OverlayValues[100] = d100
					ps299.OverlayValues[101] = d101
					ps299.OverlayValues[102] = d102
					ps299.OverlayValues[103] = d103
					ps299.OverlayValues[104] = d104
					ps299.OverlayValues[105] = d105
					ps299.OverlayValues[106] = d106
					ps299.OverlayValues[107] = d107
					ps299.OverlayValues[108] = d108
					ps299.OverlayValues[109] = d109
					ps299.OverlayValues[168] = d168
					ps299.OverlayValues[229] = d229
					ps299.OverlayValues[230] = d230
					ps299.OverlayValues[231] = d231
					ps299.OverlayValues[232] = d232
					ps299.OverlayValues[233] = d233
					ps299.OverlayValues[234] = d234
					ps299.OverlayValues[235] = d235
					ps299.OverlayValues[236] = d236
					ps299.OverlayValues[237] = d237
					ps299.OverlayValues[238] = d238
					ps299.OverlayValues[239] = d239
					ps299.OverlayValues[240] = d240
					ps299.OverlayValues[241] = d241
					ps299.OverlayValues[242] = d242
					ps299.OverlayValues[243] = d243
					ps299.OverlayValues[244] = d244
					ps299.OverlayValues[245] = d245
					ps299.OverlayValues[246] = d246
					ps299.OverlayValues[247] = d247
					ps299.OverlayValues[248] = d248
					ps300 := PhiState{General: true}
					ps300.OverlayValues = make([]JITValueDesc, 249)
					ps300.OverlayValues[0] = d0
					ps300.OverlayValues[1] = d1
					ps300.OverlayValues[2] = d2
					ps300.OverlayValues[3] = d3
					ps300.OverlayValues[18] = d18
					ps300.OverlayValues[19] = d19
					ps300.OverlayValues[21] = d21
					ps300.OverlayValues[22] = d22
					ps300.OverlayValues[23] = d23
					ps300.OverlayValues[25] = d25
					ps300.OverlayValues[26] = d26
					ps300.OverlayValues[27] = d27
					ps300.OverlayValues[58] = d58
					ps300.OverlayValues[59] = d59
					ps300.OverlayValues[60] = d60
					ps300.OverlayValues[61] = d61
					ps300.OverlayValues[100] = d100
					ps300.OverlayValues[101] = d101
					ps300.OverlayValues[102] = d102
					ps300.OverlayValues[103] = d103
					ps300.OverlayValues[104] = d104
					ps300.OverlayValues[105] = d105
					ps300.OverlayValues[106] = d106
					ps300.OverlayValues[107] = d107
					ps300.OverlayValues[108] = d108
					ps300.OverlayValues[109] = d109
					ps300.OverlayValues[168] = d168
					ps300.OverlayValues[229] = d229
					ps300.OverlayValues[230] = d230
					ps300.OverlayValues[231] = d231
					ps300.OverlayValues[232] = d232
					ps300.OverlayValues[233] = d233
					ps300.OverlayValues[234] = d234
					ps300.OverlayValues[235] = d235
					ps300.OverlayValues[236] = d236
					ps300.OverlayValues[237] = d237
					ps300.OverlayValues[238] = d238
					ps300.OverlayValues[239] = d239
					ps300.OverlayValues[240] = d240
					ps300.OverlayValues[241] = d241
					ps300.OverlayValues[242] = d242
					ps300.OverlayValues[243] = d243
					ps300.OverlayValues[244] = d244
					ps300.OverlayValues[245] = d245
					ps300.OverlayValues[246] = d246
					ps300.OverlayValues[247] = d247
					ps300.OverlayValues[248] = d248
					snap301 := d0
					snap302 := d1
					snap303 := d2
					snap304 := d3
					snap305 := d18
					snap306 := d19
					snap307 := d21
					snap308 := d22
					snap309 := d23
					snap310 := d25
					snap311 := d26
					snap312 := d27
					snap313 := d58
					snap314 := d59
					snap315 := d60
					snap316 := d61
					snap317 := d100
					snap318 := d101
					snap319 := d102
					snap320 := d103
					snap321 := d104
					snap322 := d105
					snap323 := d106
					snap324 := d107
					snap325 := d108
					snap326 := d109
					snap327 := d168
					snap328 := d229
					snap329 := d230
					snap330 := d231
					snap331 := d232
					snap332 := d233
					snap333 := d234
					snap334 := d235
					snap335 := d236
					snap336 := d237
					snap337 := d238
					snap338 := d239
					snap339 := d240
					snap340 := d241
					snap341 := d242
					snap342 := d243
					snap343 := d244
					snap344 := d245
					snap345 := d246
					snap346 := d247
					snap347 := d248
					alloc348 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps300)
					}
					ctx.RestoreAllocState(alloc348)
					d0 = snap301
					d1 = snap302
					d2 = snap303
					d3 = snap304
					d18 = snap305
					d19 = snap306
					d21 = snap307
					d22 = snap308
					d23 = snap309
					d25 = snap310
					d26 = snap311
					d27 = snap312
					d58 = snap313
					d59 = snap314
					d60 = snap315
					d61 = snap316
					d100 = snap317
					d101 = snap318
					d102 = snap319
					d103 = snap320
					d104 = snap321
					d105 = snap322
					d106 = snap323
					d107 = snap324
					d108 = snap325
					d109 = snap326
					d168 = snap327
					d229 = snap328
					d230 = snap329
					d231 = snap330
					d232 = snap331
					d233 = snap332
					d234 = snap333
					d235 = snap334
					d236 = snap335
					d237 = snap336
					d238 = snap337
					d239 = snap338
					d240 = snap339
					d241 = snap340
					d242 = snap341
					d243 = snap342
					d244 = snap343
					d245 = snap344
					d246 = snap345
					d247 = snap346
					d248 = snap347
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps299)
					}
					return result
					ctx.FreeDesc(&d247)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl24 := ctx.ReserveLabel()
					_ = lbl24
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d349 JITValueDesc
					if d105.Loc == LocImm {
						d349 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() / 3600000000000)}
					} else {
						r10 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r10, d105.Reg)
						ctx.EmitIdivRegImm(r10, 3600000000000)
						d349 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d349)
					}
					if d349.Loc == LocReg && d105.Loc == LocReg && d349.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d350 JITValueDesc
					if d105.Loc == LocImm {
						d350 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() % 3600000000000)}
					} else {
						ctx.EmitIremRegImm(d105.Reg, 3600000000000)
						d350 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d105.Reg}
						ctx.BindReg(d105.Reg, &d350)
					}
					if d350.Loc == LocReg && d105.Loc == LocReg && d350.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d349)
					ctx.EnsureDesc(&d349)
					var d351 JITValueDesc
					if d349.Loc == LocImm {
						d351 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d349.Imm.Int()))}
					} else {
						r11 := ctx.AllocRegExcept(d349.Reg)
						ctx.EmitMovRegReg(r11, d349.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r11)
						d351 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r11}
						ctx.BindReg(r11, &d351)
					}
					ctx.FreeDesc(&d349)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d350)
					ctx.EnsureDesc(&d350)
					var d352 JITValueDesc
					if d350.Loc == LocImm {
						d352 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d350.Imm.Int()))}
					} else {
						r12 := ctx.AllocRegExcept(d350.Reg)
						ctx.EmitMovRegReg(r12, d350.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r12)
						d352 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d352)
					}
					ctx.FreeDesc(&d350)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d352)
					var d353 JITValueDesc
					if d352.Loc == LocImm {
						d353 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d352.Imm.Float() / 3.6e+12)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4794699203894837248))
						ctx.EmitDivFloat64(d352.Reg, RegR11)
						d353 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d352.Reg}
						ctx.BindReg(d352.Reg, &d353)
					}
					if d353.Loc == LocReg && d352.Loc == LocReg && d353.Reg == d352.Reg {
						ctx.TransferReg(d352.Reg)
						d352.Loc = LocNone
					}
					ctx.FreeDesc(&d352)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d351)
					ctx.EnsureDesc(&d353)
					ctx.EnsureDescsTogether(&d351, &d353)
					var d354 JITValueDesc
					if d351.Loc == LocImm && d353.Loc == LocImm {
						d354 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d351.Imm.Float() + d353.Imm.Float())}
					} else if d351.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d353.Reg)
						_, xBits := d351.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d353.Reg)
						d354 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d354)
					} else if d353.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d351.Reg)
						ctx.EmitMovRegReg(scratch, d351.Reg)
						_, yBits := d353.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d354 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d354)
					} else {
						r13 := ctx.AllocRegExcept(d351.Reg, d353.Reg)
						ctx.EmitMovRegReg(r13, d351.Reg)
						ctx.EmitAddFloat64(r13, d353.Reg)
						d354 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r13}
						ctx.BindReg(r13, &d354)
					}
					if d354.Loc == LocReg && d351.Loc == LocReg && d354.Reg == d351.Reg {
						ctx.TransferReg(d351.Reg)
						d351.Loc = LocNone
					}
					ctx.FreeDesc(&d351)
					ctx.FreeDesc(&d353)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d354)
					ctx.EnsureDesc(&d354)
					ctx.EnsureDesc(&d354)
					var d355 JITValueDesc
					if d354.Loc == LocImm {
						d355 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d354.Imm.Float()))}
					} else {
						r14 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r14, d354.Reg)
						d355 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d355)
					}
					ctx.FreeDesc(&d354)
					ctx.EnsureDesc(&d355)
					if d355.Loc == LocImm {
						ctx.EmitMakeInt(result, d355)
					} else {
						ctx.EmitMovToReg(result.Reg2, d355)
						d356 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d356)
						if d355.Loc == LocReg && d355.Reg != result.Reg2 {
							ctx.FreeReg(d355.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					d357 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("HOUR")}
					var d358 JITValueDesc
					if d357.Loc == LocImm {
						ctx.TrackImm(d357.Imm)
						ptrWord, _ := d357.Imm.RawWords()
						d358 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d358.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d358.Reg2, uint64(len(d357.Imm.String())))
						ctx.BindReg(d358.Reg, &d358)
						ctx.BindReg(d358.Reg2, &d358)
					} else {
						d358 = d357
					}
					d359 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d358}, 1)
					ctx.EmitAndRegImm32(d359.Reg, 1)
					d359.Type = tagBool
					ctx.BindReg(d359.Reg, &d359)
					d360 = d359
					ctx.EnsureDesc(&d360)
					if d360.Loc != LocImm && d360.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d360.Loc == LocImm {
						if d360.Imm.Bool() {
							if ps.General {
							}
							ps361 := PhiState{General: ps.General}
							ps361.OverlayValues = make([]JITValueDesc, 361)
							ps361.OverlayValues[0] = d0
							ps361.OverlayValues[1] = d1
							ps361.OverlayValues[2] = d2
							ps361.OverlayValues[3] = d3
							ps361.OverlayValues[18] = d18
							ps361.OverlayValues[19] = d19
							ps361.OverlayValues[21] = d21
							ps361.OverlayValues[22] = d22
							ps361.OverlayValues[23] = d23
							ps361.OverlayValues[25] = d25
							ps361.OverlayValues[26] = d26
							ps361.OverlayValues[27] = d27
							ps361.OverlayValues[58] = d58
							ps361.OverlayValues[59] = d59
							ps361.OverlayValues[60] = d60
							ps361.OverlayValues[61] = d61
							ps361.OverlayValues[100] = d100
							ps361.OverlayValues[101] = d101
							ps361.OverlayValues[102] = d102
							ps361.OverlayValues[103] = d103
							ps361.OverlayValues[104] = d104
							ps361.OverlayValues[105] = d105
							ps361.OverlayValues[106] = d106
							ps361.OverlayValues[107] = d107
							ps361.OverlayValues[108] = d108
							ps361.OverlayValues[109] = d109
							ps361.OverlayValues[168] = d168
							ps361.OverlayValues[229] = d229
							ps361.OverlayValues[230] = d230
							ps361.OverlayValues[231] = d231
							ps361.OverlayValues[232] = d232
							ps361.OverlayValues[233] = d233
							ps361.OverlayValues[234] = d234
							ps361.OverlayValues[235] = d235
							ps361.OverlayValues[236] = d236
							ps361.OverlayValues[237] = d237
							ps361.OverlayValues[238] = d238
							ps361.OverlayValues[239] = d239
							ps361.OverlayValues[240] = d240
							ps361.OverlayValues[241] = d241
							ps361.OverlayValues[242] = d242
							ps361.OverlayValues[243] = d243
							ps361.OverlayValues[244] = d244
							ps361.OverlayValues[245] = d245
							ps361.OverlayValues[246] = d246
							ps361.OverlayValues[247] = d247
							ps361.OverlayValues[248] = d248
							ps361.OverlayValues[349] = d349
							ps361.OverlayValues[350] = d350
							ps361.OverlayValues[351] = d351
							ps361.OverlayValues[352] = d352
							ps361.OverlayValues[353] = d353
							ps361.OverlayValues[354] = d354
							ps361.OverlayValues[355] = d355
							ps361.OverlayValues[356] = d356
							ps361.OverlayValues[357] = d357
							ps361.OverlayValues[358] = d358
							ps361.OverlayValues[359] = d359
							ps361.OverlayValues[360] = d360
							return bbs[10].RenderPS(ps361)
						}
						if ps.General {
						}
						ps362 := PhiState{General: ps.General}
						ps362.OverlayValues = make([]JITValueDesc, 361)
						ps362.OverlayValues[0] = d0
						ps362.OverlayValues[1] = d1
						ps362.OverlayValues[2] = d2
						ps362.OverlayValues[3] = d3
						ps362.OverlayValues[18] = d18
						ps362.OverlayValues[19] = d19
						ps362.OverlayValues[21] = d21
						ps362.OverlayValues[22] = d22
						ps362.OverlayValues[23] = d23
						ps362.OverlayValues[25] = d25
						ps362.OverlayValues[26] = d26
						ps362.OverlayValues[27] = d27
						ps362.OverlayValues[58] = d58
						ps362.OverlayValues[59] = d59
						ps362.OverlayValues[60] = d60
						ps362.OverlayValues[61] = d61
						ps362.OverlayValues[100] = d100
						ps362.OverlayValues[101] = d101
						ps362.OverlayValues[102] = d102
						ps362.OverlayValues[103] = d103
						ps362.OverlayValues[104] = d104
						ps362.OverlayValues[105] = d105
						ps362.OverlayValues[106] = d106
						ps362.OverlayValues[107] = d107
						ps362.OverlayValues[108] = d108
						ps362.OverlayValues[109] = d109
						ps362.OverlayValues[168] = d168
						ps362.OverlayValues[229] = d229
						ps362.OverlayValues[230] = d230
						ps362.OverlayValues[231] = d231
						ps362.OverlayValues[232] = d232
						ps362.OverlayValues[233] = d233
						ps362.OverlayValues[234] = d234
						ps362.OverlayValues[235] = d235
						ps362.OverlayValues[236] = d236
						ps362.OverlayValues[237] = d237
						ps362.OverlayValues[238] = d238
						ps362.OverlayValues[239] = d239
						ps362.OverlayValues[240] = d240
						ps362.OverlayValues[241] = d241
						ps362.OverlayValues[242] = d242
						ps362.OverlayValues[243] = d243
						ps362.OverlayValues[244] = d244
						ps362.OverlayValues[245] = d245
						ps362.OverlayValues[246] = d246
						ps362.OverlayValues[247] = d247
						ps362.OverlayValues[248] = d248
						ps362.OverlayValues[349] = d349
						ps362.OverlayValues[350] = d350
						ps362.OverlayValues[351] = d351
						ps362.OverlayValues[352] = d352
						ps362.OverlayValues[353] = d353
						ps362.OverlayValues[354] = d354
						ps362.OverlayValues[355] = d355
						ps362.OverlayValues[356] = d356
						ps362.OverlayValues[357] = d357
						ps362.OverlayValues[358] = d358
						ps362.OverlayValues[359] = d359
						ps362.OverlayValues[360] = d360
						return bbs[13].RenderPS(ps362)
					}
					if !ps.General {
						ps.General = true
						return bbs[11].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d360.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					snap363 := d0
					snap364 := d1
					snap365 := d2
					snap366 := d3
					snap367 := d18
					snap368 := d19
					snap369 := d21
					snap370 := d22
					snap371 := d23
					snap372 := d25
					snap373 := d26
					snap374 := d27
					snap375 := d58
					snap376 := d59
					snap377 := d60
					snap378 := d61
					snap379 := d100
					snap380 := d101
					snap381 := d102
					snap382 := d103
					snap383 := d104
					snap384 := d105
					snap385 := d106
					snap386 := d107
					snap387 := d108
					snap388 := d109
					snap389 := d168
					snap390 := d229
					snap391 := d230
					snap392 := d231
					snap393 := d232
					snap394 := d233
					snap395 := d234
					snap396 := d235
					snap397 := d236
					snap398 := d237
					snap399 := d238
					snap400 := d239
					snap401 := d240
					snap402 := d241
					snap403 := d242
					snap404 := d243
					snap405 := d244
					snap406 := d245
					snap407 := d246
					snap408 := d247
					snap409 := d248
					snap410 := d349
					snap411 := d350
					snap412 := d351
					snap413 := d352
					snap414 := d353
					snap415 := d354
					snap416 := d355
					snap417 := d356
					snap418 := d357
					snap419 := d358
					snap420 := d359
					snap421 := d360
					alloc422 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc422)
					d0 = snap363
					d1 = snap364
					d2 = snap365
					d3 = snap366
					d18 = snap367
					d19 = snap368
					d21 = snap369
					d22 = snap370
					d23 = snap371
					d25 = snap372
					d26 = snap373
					d27 = snap374
					d58 = snap375
					d59 = snap376
					d60 = snap377
					d61 = snap378
					d100 = snap379
					d101 = snap380
					d102 = snap381
					d103 = snap382
					d104 = snap383
					d105 = snap384
					d106 = snap385
					d107 = snap386
					d108 = snap387
					d109 = snap388
					d168 = snap389
					d229 = snap390
					d230 = snap391
					d231 = snap392
					d232 = snap393
					d233 = snap394
					d234 = snap395
					d235 = snap396
					d236 = snap397
					d237 = snap398
					d238 = snap399
					d239 = snap400
					d240 = snap401
					d241 = snap402
					d242 = snap403
					d243 = snap404
					d244 = snap405
					d245 = snap406
					d246 = snap407
					d247 = snap408
					d248 = snap409
					d349 = snap410
					d350 = snap411
					d351 = snap412
					d352 = snap413
					d353 = snap414
					d354 = snap415
					d355 = snap416
					d356 = snap417
					d357 = snap418
					d358 = snap419
					d359 = snap420
					d360 = snap421
					ctx.RestoreAllocState(alloc422)
					d0 = snap363
					d1 = snap364
					d2 = snap365
					d3 = snap366
					d18 = snap367
					d19 = snap368
					d21 = snap369
					d22 = snap370
					d23 = snap371
					d25 = snap372
					d26 = snap373
					d27 = snap374
					d58 = snap375
					d59 = snap376
					d60 = snap377
					d61 = snap378
					d100 = snap379
					d101 = snap380
					d102 = snap381
					d103 = snap382
					d104 = snap383
					d105 = snap384
					d106 = snap385
					d107 = snap386
					d108 = snap387
					d109 = snap388
					d168 = snap389
					d229 = snap390
					d230 = snap391
					d231 = snap392
					d232 = snap393
					d233 = snap394
					d234 = snap395
					d235 = snap396
					d236 = snap397
					d237 = snap398
					d238 = snap399
					d239 = snap400
					d240 = snap401
					d241 = snap402
					d242 = snap403
					d243 = snap404
					d244 = snap405
					d245 = snap406
					d246 = snap407
					d247 = snap408
					d248 = snap409
					d349 = snap410
					d350 = snap411
					d351 = snap412
					d352 = snap413
					d353 = snap414
					d354 = snap415
					d355 = snap416
					d356 = snap417
					d357 = snap418
					d358 = snap419
					d359 = snap420
					d360 = snap421
					ps423 := PhiState{General: true}
					ps423.OverlayValues = make([]JITValueDesc, 361)
					ps423.OverlayValues[0] = d0
					ps423.OverlayValues[1] = d1
					ps423.OverlayValues[2] = d2
					ps423.OverlayValues[3] = d3
					ps423.OverlayValues[18] = d18
					ps423.OverlayValues[19] = d19
					ps423.OverlayValues[21] = d21
					ps423.OverlayValues[22] = d22
					ps423.OverlayValues[23] = d23
					ps423.OverlayValues[25] = d25
					ps423.OverlayValues[26] = d26
					ps423.OverlayValues[27] = d27
					ps423.OverlayValues[58] = d58
					ps423.OverlayValues[59] = d59
					ps423.OverlayValues[60] = d60
					ps423.OverlayValues[61] = d61
					ps423.OverlayValues[100] = d100
					ps423.OverlayValues[101] = d101
					ps423.OverlayValues[102] = d102
					ps423.OverlayValues[103] = d103
					ps423.OverlayValues[104] = d104
					ps423.OverlayValues[105] = d105
					ps423.OverlayValues[106] = d106
					ps423.OverlayValues[107] = d107
					ps423.OverlayValues[108] = d108
					ps423.OverlayValues[109] = d109
					ps423.OverlayValues[168] = d168
					ps423.OverlayValues[229] = d229
					ps423.OverlayValues[230] = d230
					ps423.OverlayValues[231] = d231
					ps423.OverlayValues[232] = d232
					ps423.OverlayValues[233] = d233
					ps423.OverlayValues[234] = d234
					ps423.OverlayValues[235] = d235
					ps423.OverlayValues[236] = d236
					ps423.OverlayValues[237] = d237
					ps423.OverlayValues[238] = d238
					ps423.OverlayValues[239] = d239
					ps423.OverlayValues[240] = d240
					ps423.OverlayValues[241] = d241
					ps423.OverlayValues[242] = d242
					ps423.OverlayValues[243] = d243
					ps423.OverlayValues[244] = d244
					ps423.OverlayValues[245] = d245
					ps423.OverlayValues[246] = d246
					ps423.OverlayValues[247] = d247
					ps423.OverlayValues[248] = d248
					ps423.OverlayValues[349] = d349
					ps423.OverlayValues[350] = d350
					ps423.OverlayValues[351] = d351
					ps423.OverlayValues[352] = d352
					ps423.OverlayValues[353] = d353
					ps423.OverlayValues[354] = d354
					ps423.OverlayValues[355] = d355
					ps423.OverlayValues[356] = d356
					ps423.OverlayValues[357] = d357
					ps423.OverlayValues[358] = d358
					ps423.OverlayValues[359] = d359
					ps423.OverlayValues[360] = d360
					ps424 := PhiState{General: true}
					ps424.OverlayValues = make([]JITValueDesc, 361)
					ps424.OverlayValues[0] = d0
					ps424.OverlayValues[1] = d1
					ps424.OverlayValues[2] = d2
					ps424.OverlayValues[3] = d3
					ps424.OverlayValues[18] = d18
					ps424.OverlayValues[19] = d19
					ps424.OverlayValues[21] = d21
					ps424.OverlayValues[22] = d22
					ps424.OverlayValues[23] = d23
					ps424.OverlayValues[25] = d25
					ps424.OverlayValues[26] = d26
					ps424.OverlayValues[27] = d27
					ps424.OverlayValues[58] = d58
					ps424.OverlayValues[59] = d59
					ps424.OverlayValues[60] = d60
					ps424.OverlayValues[61] = d61
					ps424.OverlayValues[100] = d100
					ps424.OverlayValues[101] = d101
					ps424.OverlayValues[102] = d102
					ps424.OverlayValues[103] = d103
					ps424.OverlayValues[104] = d104
					ps424.OverlayValues[105] = d105
					ps424.OverlayValues[106] = d106
					ps424.OverlayValues[107] = d107
					ps424.OverlayValues[108] = d108
					ps424.OverlayValues[109] = d109
					ps424.OverlayValues[168] = d168
					ps424.OverlayValues[229] = d229
					ps424.OverlayValues[230] = d230
					ps424.OverlayValues[231] = d231
					ps424.OverlayValues[232] = d232
					ps424.OverlayValues[233] = d233
					ps424.OverlayValues[234] = d234
					ps424.OverlayValues[235] = d235
					ps424.OverlayValues[236] = d236
					ps424.OverlayValues[237] = d237
					ps424.OverlayValues[238] = d238
					ps424.OverlayValues[239] = d239
					ps424.OverlayValues[240] = d240
					ps424.OverlayValues[241] = d241
					ps424.OverlayValues[242] = d242
					ps424.OverlayValues[243] = d243
					ps424.OverlayValues[244] = d244
					ps424.OverlayValues[245] = d245
					ps424.OverlayValues[246] = d246
					ps424.OverlayValues[247] = d247
					ps424.OverlayValues[248] = d248
					ps424.OverlayValues[349] = d349
					ps424.OverlayValues[350] = d350
					ps424.OverlayValues[351] = d351
					ps424.OverlayValues[352] = d352
					ps424.OverlayValues[353] = d353
					ps424.OverlayValues[354] = d354
					ps424.OverlayValues[355] = d355
					ps424.OverlayValues[356] = d356
					ps424.OverlayValues[357] = d357
					ps424.OverlayValues[358] = d358
					ps424.OverlayValues[359] = d359
					ps424.OverlayValues[360] = d360
					snap425 := d0
					snap426 := d1
					snap427 := d2
					snap428 := d3
					snap429 := d18
					snap430 := d19
					snap431 := d21
					snap432 := d22
					snap433 := d23
					snap434 := d25
					snap435 := d26
					snap436 := d27
					snap437 := d58
					snap438 := d59
					snap439 := d60
					snap440 := d61
					snap441 := d100
					snap442 := d101
					snap443 := d102
					snap444 := d103
					snap445 := d104
					snap446 := d105
					snap447 := d106
					snap448 := d107
					snap449 := d108
					snap450 := d109
					snap451 := d168
					snap452 := d229
					snap453 := d230
					snap454 := d231
					snap455 := d232
					snap456 := d233
					snap457 := d234
					snap458 := d235
					snap459 := d236
					snap460 := d237
					snap461 := d238
					snap462 := d239
					snap463 := d240
					snap464 := d241
					snap465 := d242
					snap466 := d243
					snap467 := d244
					snap468 := d245
					snap469 := d246
					snap470 := d247
					snap471 := d248
					snap472 := d349
					snap473 := d350
					snap474 := d351
					snap475 := d352
					snap476 := d353
					snap477 := d354
					snap478 := d355
					snap479 := d356
					snap480 := d357
					snap481 := d358
					snap482 := d359
					snap483 := d360
					alloc484 := ctx.SnapshotAllocState()
					if !bbs[13].Rendered {
						bbs[13].RenderPS(ps424)
					}
					ctx.RestoreAllocState(alloc484)
					d0 = snap425
					d1 = snap426
					d2 = snap427
					d3 = snap428
					d18 = snap429
					d19 = snap430
					d21 = snap431
					d22 = snap432
					d23 = snap433
					d25 = snap434
					d26 = snap435
					d27 = snap436
					d58 = snap437
					d59 = snap438
					d60 = snap439
					d61 = snap440
					d100 = snap441
					d101 = snap442
					d102 = snap443
					d103 = snap444
					d104 = snap445
					d105 = snap446
					d106 = snap447
					d107 = snap448
					d108 = snap449
					d109 = snap450
					d168 = snap451
					d229 = snap452
					d230 = snap453
					d231 = snap454
					d232 = snap455
					d233 = snap456
					d234 = snap457
					d235 = snap458
					d236 = snap459
					d237 = snap460
					d238 = snap461
					d239 = snap462
					d240 = snap463
					d241 = snap464
					d242 = snap465
					d243 = snap466
					d244 = snap467
					d245 = snap468
					d246 = snap469
					d247 = snap470
					d248 = snap471
					d349 = snap472
					d350 = snap473
					d351 = snap474
					d352 = snap475
					d353 = snap476
					d354 = snap477
					d355 = snap478
					d356 = snap479
					d357 = snap480
					d358 = snap481
					d359 = snap482
					d360 = snap483
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps423)
					}
					return result
					ctx.FreeDesc(&d359)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl25 := ctx.ReserveLabel()
					_ = lbl25
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl25)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d485 JITValueDesc
					if d105.Loc == LocImm {
						d485 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() / 3600000000000)}
					} else {
						r15 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r15, d105.Reg)
						ctx.EmitIdivRegImm(r15, 3600000000000)
						d485 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d485)
					}
					if d485.Loc == LocReg && d105.Loc == LocReg && d485.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d486 JITValueDesc
					if d105.Loc == LocImm {
						d486 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() % 3600000000000)}
					} else {
						ctx.EmitIremRegImm(d105.Reg, 3600000000000)
						d486 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d105.Reg}
						ctx.BindReg(d105.Reg, &d486)
					}
					if d486.Loc == LocReg && d105.Loc == LocReg && d486.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d485)
					ctx.EnsureDesc(&d485)
					var d487 JITValueDesc
					if d485.Loc == LocImm {
						d487 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d485.Imm.Int()))}
					} else {
						r16 := ctx.AllocRegExcept(d485.Reg)
						ctx.EmitMovRegReg(r16, d485.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r16)
						d487 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
						ctx.BindReg(r16, &d487)
					}
					ctx.FreeDesc(&d485)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d486)
					ctx.EnsureDesc(&d486)
					var d488 JITValueDesc
					if d486.Loc == LocImm {
						d488 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d486.Imm.Int()))}
					} else {
						r17 := ctx.AllocRegExcept(d486.Reg)
						ctx.EmitMovRegReg(r17, d486.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r17)
						d488 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r17}
						ctx.BindReg(r17, &d488)
					}
					ctx.FreeDesc(&d486)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d488)
					var d489 JITValueDesc
					if d488.Loc == LocImm {
						d489 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d488.Imm.Float() / 3.6e+12)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4794699203894837248))
						ctx.EmitDivFloat64(d488.Reg, RegR11)
						d489 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d488.Reg}
						ctx.BindReg(d488.Reg, &d489)
					}
					if d489.Loc == LocReg && d488.Loc == LocReg && d489.Reg == d488.Reg {
						ctx.TransferReg(d488.Reg)
						d488.Loc = LocNone
					}
					ctx.FreeDesc(&d488)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d487)
					ctx.EnsureDesc(&d489)
					ctx.EnsureDescsTogether(&d487, &d489)
					var d490 JITValueDesc
					if d487.Loc == LocImm && d489.Loc == LocImm {
						d490 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d487.Imm.Float() + d489.Imm.Float())}
					} else if d487.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d489.Reg)
						_, xBits := d487.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d489.Reg)
						d490 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d490)
					} else if d489.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d487.Reg)
						ctx.EmitMovRegReg(scratch, d487.Reg)
						_, yBits := d489.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d490 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d490)
					} else {
						r18 := ctx.AllocRegExcept(d487.Reg, d489.Reg)
						ctx.EmitMovRegReg(r18, d487.Reg)
						ctx.EmitAddFloat64(r18, d489.Reg)
						d490 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r18}
						ctx.BindReg(r18, &d490)
					}
					if d490.Loc == LocReg && d487.Loc == LocReg && d490.Reg == d487.Reg {
						ctx.TransferReg(d487.Reg)
						d487.Loc = LocNone
					}
					ctx.FreeDesc(&d487)
					ctx.FreeDesc(&d489)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d490)
					ctx.EnsureDesc(&d490)
					var d491 JITValueDesc
					if d490.Loc == LocImm {
						d491 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d490.Imm.Float() / 24)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4627448617123184640))
						ctx.EmitDivFloat64(d490.Reg, RegR11)
						d491 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d490.Reg}
						ctx.BindReg(d490.Reg, &d491)
					}
					if d491.Loc == LocReg && d490.Loc == LocReg && d491.Reg == d490.Reg {
						ctx.TransferReg(d490.Reg)
						d490.Loc = LocNone
					}
					ctx.FreeDesc(&d490)
					ctx.EnsureDesc(&d491)
					ctx.EnsureDesc(&d491)
					var d492 JITValueDesc
					if d491.Loc == LocImm {
						d492 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d491.Imm.Float()))}
					} else {
						r19 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r19, d491.Reg)
						d492 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d492)
					}
					ctx.FreeDesc(&d491)
					ctx.EnsureDesc(&d492)
					if d492.Loc == LocImm {
						ctx.EmitMakeInt(result, d492)
					} else {
						ctx.EmitMovToReg(result.Reg2, d492)
						d493 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d493)
						if d492.Loc == LocReg && d492.Reg != result.Reg2 {
							ctx.FreeReg(d492.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					ctx.EnsureDesc(&d104)
					d494 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DAY")}
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
					d496 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d495}, 1)
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
							ps498.OverlayValues[18] = d18
							ps498.OverlayValues[19] = d19
							ps498.OverlayValues[21] = d21
							ps498.OverlayValues[22] = d22
							ps498.OverlayValues[23] = d23
							ps498.OverlayValues[25] = d25
							ps498.OverlayValues[26] = d26
							ps498.OverlayValues[27] = d27
							ps498.OverlayValues[58] = d58
							ps498.OverlayValues[59] = d59
							ps498.OverlayValues[60] = d60
							ps498.OverlayValues[61] = d61
							ps498.OverlayValues[100] = d100
							ps498.OverlayValues[101] = d101
							ps498.OverlayValues[102] = d102
							ps498.OverlayValues[103] = d103
							ps498.OverlayValues[104] = d104
							ps498.OverlayValues[105] = d105
							ps498.OverlayValues[106] = d106
							ps498.OverlayValues[107] = d107
							ps498.OverlayValues[108] = d108
							ps498.OverlayValues[109] = d109
							ps498.OverlayValues[168] = d168
							ps498.OverlayValues[229] = d229
							ps498.OverlayValues[230] = d230
							ps498.OverlayValues[231] = d231
							ps498.OverlayValues[232] = d232
							ps498.OverlayValues[233] = d233
							ps498.OverlayValues[234] = d234
							ps498.OverlayValues[235] = d235
							ps498.OverlayValues[236] = d236
							ps498.OverlayValues[237] = d237
							ps498.OverlayValues[238] = d238
							ps498.OverlayValues[239] = d239
							ps498.OverlayValues[240] = d240
							ps498.OverlayValues[241] = d241
							ps498.OverlayValues[242] = d242
							ps498.OverlayValues[243] = d243
							ps498.OverlayValues[244] = d244
							ps498.OverlayValues[245] = d245
							ps498.OverlayValues[246] = d246
							ps498.OverlayValues[247] = d247
							ps498.OverlayValues[248] = d248
							ps498.OverlayValues[349] = d349
							ps498.OverlayValues[350] = d350
							ps498.OverlayValues[351] = d351
							ps498.OverlayValues[352] = d352
							ps498.OverlayValues[353] = d353
							ps498.OverlayValues[354] = d354
							ps498.OverlayValues[355] = d355
							ps498.OverlayValues[356] = d356
							ps498.OverlayValues[357] = d357
							ps498.OverlayValues[358] = d358
							ps498.OverlayValues[359] = d359
							ps498.OverlayValues[360] = d360
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
							return bbs[12].RenderPS(ps498)
						}
						if ps.General {
						}
						ps499 := PhiState{General: ps.General}
						ps499.OverlayValues = make([]JITValueDesc, 498)
						ps499.OverlayValues[0] = d0
						ps499.OverlayValues[1] = d1
						ps499.OverlayValues[2] = d2
						ps499.OverlayValues[3] = d3
						ps499.OverlayValues[18] = d18
						ps499.OverlayValues[19] = d19
						ps499.OverlayValues[21] = d21
						ps499.OverlayValues[22] = d22
						ps499.OverlayValues[23] = d23
						ps499.OverlayValues[25] = d25
						ps499.OverlayValues[26] = d26
						ps499.OverlayValues[27] = d27
						ps499.OverlayValues[58] = d58
						ps499.OverlayValues[59] = d59
						ps499.OverlayValues[60] = d60
						ps499.OverlayValues[61] = d61
						ps499.OverlayValues[100] = d100
						ps499.OverlayValues[101] = d101
						ps499.OverlayValues[102] = d102
						ps499.OverlayValues[103] = d103
						ps499.OverlayValues[104] = d104
						ps499.OverlayValues[105] = d105
						ps499.OverlayValues[106] = d106
						ps499.OverlayValues[107] = d107
						ps499.OverlayValues[108] = d108
						ps499.OverlayValues[109] = d109
						ps499.OverlayValues[168] = d168
						ps499.OverlayValues[229] = d229
						ps499.OverlayValues[230] = d230
						ps499.OverlayValues[231] = d231
						ps499.OverlayValues[232] = d232
						ps499.OverlayValues[233] = d233
						ps499.OverlayValues[234] = d234
						ps499.OverlayValues[235] = d235
						ps499.OverlayValues[236] = d236
						ps499.OverlayValues[237] = d237
						ps499.OverlayValues[238] = d238
						ps499.OverlayValues[239] = d239
						ps499.OverlayValues[240] = d240
						ps499.OverlayValues[241] = d241
						ps499.OverlayValues[242] = d242
						ps499.OverlayValues[243] = d243
						ps499.OverlayValues[244] = d244
						ps499.OverlayValues[245] = d245
						ps499.OverlayValues[246] = d246
						ps499.OverlayValues[247] = d247
						ps499.OverlayValues[248] = d248
						ps499.OverlayValues[349] = d349
						ps499.OverlayValues[350] = d350
						ps499.OverlayValues[351] = d351
						ps499.OverlayValues[352] = d352
						ps499.OverlayValues[353] = d353
						ps499.OverlayValues[354] = d354
						ps499.OverlayValues[355] = d355
						ps499.OverlayValues[356] = d356
						ps499.OverlayValues[357] = d357
						ps499.OverlayValues[358] = d358
						ps499.OverlayValues[359] = d359
						ps499.OverlayValues[360] = d360
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
						return bbs[15].RenderPS(ps499)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d497.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					snap500 := d0
					snap501 := d1
					snap502 := d2
					snap503 := d3
					snap504 := d18
					snap505 := d19
					snap506 := d21
					snap507 := d22
					snap508 := d23
					snap509 := d25
					snap510 := d26
					snap511 := d27
					snap512 := d58
					snap513 := d59
					snap514 := d60
					snap515 := d61
					snap516 := d100
					snap517 := d101
					snap518 := d102
					snap519 := d103
					snap520 := d104
					snap521 := d105
					snap522 := d106
					snap523 := d107
					snap524 := d108
					snap525 := d109
					snap526 := d168
					snap527 := d229
					snap528 := d230
					snap529 := d231
					snap530 := d232
					snap531 := d233
					snap532 := d234
					snap533 := d235
					snap534 := d236
					snap535 := d237
					snap536 := d238
					snap537 := d239
					snap538 := d240
					snap539 := d241
					snap540 := d242
					snap541 := d243
					snap542 := d244
					snap543 := d245
					snap544 := d246
					snap545 := d247
					snap546 := d248
					snap547 := d349
					snap548 := d350
					snap549 := d351
					snap550 := d352
					snap551 := d353
					snap552 := d354
					snap553 := d355
					snap554 := d356
					snap555 := d357
					snap556 := d358
					snap557 := d359
					snap558 := d360
					snap559 := d485
					snap560 := d486
					snap561 := d487
					snap562 := d488
					snap563 := d489
					snap564 := d490
					snap565 := d491
					snap566 := d492
					snap567 := d493
					snap568 := d494
					snap569 := d495
					snap570 := d496
					snap571 := d497
					alloc572 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc572)
					d0 = snap500
					d1 = snap501
					d2 = snap502
					d3 = snap503
					d18 = snap504
					d19 = snap505
					d21 = snap506
					d22 = snap507
					d23 = snap508
					d25 = snap509
					d26 = snap510
					d27 = snap511
					d58 = snap512
					d59 = snap513
					d60 = snap514
					d61 = snap515
					d100 = snap516
					d101 = snap517
					d102 = snap518
					d103 = snap519
					d104 = snap520
					d105 = snap521
					d106 = snap522
					d107 = snap523
					d108 = snap524
					d109 = snap525
					d168 = snap526
					d229 = snap527
					d230 = snap528
					d231 = snap529
					d232 = snap530
					d233 = snap531
					d234 = snap532
					d235 = snap533
					d236 = snap534
					d237 = snap535
					d238 = snap536
					d239 = snap537
					d240 = snap538
					d241 = snap539
					d242 = snap540
					d243 = snap541
					d244 = snap542
					d245 = snap543
					d246 = snap544
					d247 = snap545
					d248 = snap546
					d349 = snap547
					d350 = snap548
					d351 = snap549
					d352 = snap550
					d353 = snap551
					d354 = snap552
					d355 = snap553
					d356 = snap554
					d357 = snap555
					d358 = snap556
					d359 = snap557
					d360 = snap558
					d485 = snap559
					d486 = snap560
					d487 = snap561
					d488 = snap562
					d489 = snap563
					d490 = snap564
					d491 = snap565
					d492 = snap566
					d493 = snap567
					d494 = snap568
					d495 = snap569
					d496 = snap570
					d497 = snap571
					ctx.RestoreAllocState(alloc572)
					d0 = snap500
					d1 = snap501
					d2 = snap502
					d3 = snap503
					d18 = snap504
					d19 = snap505
					d21 = snap506
					d22 = snap507
					d23 = snap508
					d25 = snap509
					d26 = snap510
					d27 = snap511
					d58 = snap512
					d59 = snap513
					d60 = snap514
					d61 = snap515
					d100 = snap516
					d101 = snap517
					d102 = snap518
					d103 = snap519
					d104 = snap520
					d105 = snap521
					d106 = snap522
					d107 = snap523
					d108 = snap524
					d109 = snap525
					d168 = snap526
					d229 = snap527
					d230 = snap528
					d231 = snap529
					d232 = snap530
					d233 = snap531
					d234 = snap532
					d235 = snap533
					d236 = snap534
					d237 = snap535
					d238 = snap536
					d239 = snap537
					d240 = snap538
					d241 = snap539
					d242 = snap540
					d243 = snap541
					d244 = snap542
					d245 = snap543
					d246 = snap544
					d247 = snap545
					d248 = snap546
					d349 = snap547
					d350 = snap548
					d351 = snap549
					d352 = snap550
					d353 = snap551
					d354 = snap552
					d355 = snap553
					d356 = snap554
					d357 = snap555
					d358 = snap556
					d359 = snap557
					d360 = snap558
					d485 = snap559
					d486 = snap560
					d487 = snap561
					d488 = snap562
					d489 = snap563
					d490 = snap564
					d491 = snap565
					d492 = snap566
					d493 = snap567
					d494 = snap568
					d495 = snap569
					d496 = snap570
					d497 = snap571
					ps573 := PhiState{General: true}
					ps573.OverlayValues = make([]JITValueDesc, 498)
					ps573.OverlayValues[0] = d0
					ps573.OverlayValues[1] = d1
					ps573.OverlayValues[2] = d2
					ps573.OverlayValues[3] = d3
					ps573.OverlayValues[18] = d18
					ps573.OverlayValues[19] = d19
					ps573.OverlayValues[21] = d21
					ps573.OverlayValues[22] = d22
					ps573.OverlayValues[23] = d23
					ps573.OverlayValues[25] = d25
					ps573.OverlayValues[26] = d26
					ps573.OverlayValues[27] = d27
					ps573.OverlayValues[58] = d58
					ps573.OverlayValues[59] = d59
					ps573.OverlayValues[60] = d60
					ps573.OverlayValues[61] = d61
					ps573.OverlayValues[100] = d100
					ps573.OverlayValues[101] = d101
					ps573.OverlayValues[102] = d102
					ps573.OverlayValues[103] = d103
					ps573.OverlayValues[104] = d104
					ps573.OverlayValues[105] = d105
					ps573.OverlayValues[106] = d106
					ps573.OverlayValues[107] = d107
					ps573.OverlayValues[108] = d108
					ps573.OverlayValues[109] = d109
					ps573.OverlayValues[168] = d168
					ps573.OverlayValues[229] = d229
					ps573.OverlayValues[230] = d230
					ps573.OverlayValues[231] = d231
					ps573.OverlayValues[232] = d232
					ps573.OverlayValues[233] = d233
					ps573.OverlayValues[234] = d234
					ps573.OverlayValues[235] = d235
					ps573.OverlayValues[236] = d236
					ps573.OverlayValues[237] = d237
					ps573.OverlayValues[238] = d238
					ps573.OverlayValues[239] = d239
					ps573.OverlayValues[240] = d240
					ps573.OverlayValues[241] = d241
					ps573.OverlayValues[242] = d242
					ps573.OverlayValues[243] = d243
					ps573.OverlayValues[244] = d244
					ps573.OverlayValues[245] = d245
					ps573.OverlayValues[246] = d246
					ps573.OverlayValues[247] = d247
					ps573.OverlayValues[248] = d248
					ps573.OverlayValues[349] = d349
					ps573.OverlayValues[350] = d350
					ps573.OverlayValues[351] = d351
					ps573.OverlayValues[352] = d352
					ps573.OverlayValues[353] = d353
					ps573.OverlayValues[354] = d354
					ps573.OverlayValues[355] = d355
					ps573.OverlayValues[356] = d356
					ps573.OverlayValues[357] = d357
					ps573.OverlayValues[358] = d358
					ps573.OverlayValues[359] = d359
					ps573.OverlayValues[360] = d360
					ps573.OverlayValues[485] = d485
					ps573.OverlayValues[486] = d486
					ps573.OverlayValues[487] = d487
					ps573.OverlayValues[488] = d488
					ps573.OverlayValues[489] = d489
					ps573.OverlayValues[490] = d490
					ps573.OverlayValues[491] = d491
					ps573.OverlayValues[492] = d492
					ps573.OverlayValues[493] = d493
					ps573.OverlayValues[494] = d494
					ps573.OverlayValues[495] = d495
					ps573.OverlayValues[496] = d496
					ps573.OverlayValues[497] = d497
					ps574 := PhiState{General: true}
					ps574.OverlayValues = make([]JITValueDesc, 498)
					ps574.OverlayValues[0] = d0
					ps574.OverlayValues[1] = d1
					ps574.OverlayValues[2] = d2
					ps574.OverlayValues[3] = d3
					ps574.OverlayValues[18] = d18
					ps574.OverlayValues[19] = d19
					ps574.OverlayValues[21] = d21
					ps574.OverlayValues[22] = d22
					ps574.OverlayValues[23] = d23
					ps574.OverlayValues[25] = d25
					ps574.OverlayValues[26] = d26
					ps574.OverlayValues[27] = d27
					ps574.OverlayValues[58] = d58
					ps574.OverlayValues[59] = d59
					ps574.OverlayValues[60] = d60
					ps574.OverlayValues[61] = d61
					ps574.OverlayValues[100] = d100
					ps574.OverlayValues[101] = d101
					ps574.OverlayValues[102] = d102
					ps574.OverlayValues[103] = d103
					ps574.OverlayValues[104] = d104
					ps574.OverlayValues[105] = d105
					ps574.OverlayValues[106] = d106
					ps574.OverlayValues[107] = d107
					ps574.OverlayValues[108] = d108
					ps574.OverlayValues[109] = d109
					ps574.OverlayValues[168] = d168
					ps574.OverlayValues[229] = d229
					ps574.OverlayValues[230] = d230
					ps574.OverlayValues[231] = d231
					ps574.OverlayValues[232] = d232
					ps574.OverlayValues[233] = d233
					ps574.OverlayValues[234] = d234
					ps574.OverlayValues[235] = d235
					ps574.OverlayValues[236] = d236
					ps574.OverlayValues[237] = d237
					ps574.OverlayValues[238] = d238
					ps574.OverlayValues[239] = d239
					ps574.OverlayValues[240] = d240
					ps574.OverlayValues[241] = d241
					ps574.OverlayValues[242] = d242
					ps574.OverlayValues[243] = d243
					ps574.OverlayValues[244] = d244
					ps574.OverlayValues[245] = d245
					ps574.OverlayValues[246] = d246
					ps574.OverlayValues[247] = d247
					ps574.OverlayValues[248] = d248
					ps574.OverlayValues[349] = d349
					ps574.OverlayValues[350] = d350
					ps574.OverlayValues[351] = d351
					ps574.OverlayValues[352] = d352
					ps574.OverlayValues[353] = d353
					ps574.OverlayValues[354] = d354
					ps574.OverlayValues[355] = d355
					ps574.OverlayValues[356] = d356
					ps574.OverlayValues[357] = d357
					ps574.OverlayValues[358] = d358
					ps574.OverlayValues[359] = d359
					ps574.OverlayValues[360] = d360
					ps574.OverlayValues[485] = d485
					ps574.OverlayValues[486] = d486
					ps574.OverlayValues[487] = d487
					ps574.OverlayValues[488] = d488
					ps574.OverlayValues[489] = d489
					ps574.OverlayValues[490] = d490
					ps574.OverlayValues[491] = d491
					ps574.OverlayValues[492] = d492
					ps574.OverlayValues[493] = d493
					ps574.OverlayValues[494] = d494
					ps574.OverlayValues[495] = d495
					ps574.OverlayValues[496] = d496
					ps574.OverlayValues[497] = d497
					snap575 := d0
					snap576 := d1
					snap577 := d2
					snap578 := d3
					snap579 := d18
					snap580 := d19
					snap581 := d21
					snap582 := d22
					snap583 := d23
					snap584 := d25
					snap585 := d26
					snap586 := d27
					snap587 := d58
					snap588 := d59
					snap589 := d60
					snap590 := d61
					snap591 := d100
					snap592 := d101
					snap593 := d102
					snap594 := d103
					snap595 := d104
					snap596 := d105
					snap597 := d106
					snap598 := d107
					snap599 := d108
					snap600 := d109
					snap601 := d168
					snap602 := d229
					snap603 := d230
					snap604 := d231
					snap605 := d232
					snap606 := d233
					snap607 := d234
					snap608 := d235
					snap609 := d236
					snap610 := d237
					snap611 := d238
					snap612 := d239
					snap613 := d240
					snap614 := d241
					snap615 := d242
					snap616 := d243
					snap617 := d244
					snap618 := d245
					snap619 := d246
					snap620 := d247
					snap621 := d248
					snap622 := d349
					snap623 := d350
					snap624 := d351
					snap625 := d352
					snap626 := d353
					snap627 := d354
					snap628 := d355
					snap629 := d356
					snap630 := d357
					snap631 := d358
					snap632 := d359
					snap633 := d360
					snap634 := d485
					snap635 := d486
					snap636 := d487
					snap637 := d488
					snap638 := d489
					snap639 := d490
					snap640 := d491
					snap641 := d492
					snap642 := d493
					snap643 := d494
					snap644 := d495
					snap645 := d496
					snap646 := d497
					alloc647 := ctx.SnapshotAllocState()
					if !bbs[15].Rendered {
						bbs[15].RenderPS(ps574)
					}
					ctx.RestoreAllocState(alloc647)
					d0 = snap575
					d1 = snap576
					d2 = snap577
					d3 = snap578
					d18 = snap579
					d19 = snap580
					d21 = snap581
					d22 = snap582
					d23 = snap583
					d25 = snap584
					d26 = snap585
					d27 = snap586
					d58 = snap587
					d59 = snap588
					d60 = snap589
					d61 = snap590
					d100 = snap591
					d101 = snap592
					d102 = snap593
					d103 = snap594
					d104 = snap595
					d105 = snap596
					d106 = snap597
					d107 = snap598
					d108 = snap599
					d109 = snap600
					d168 = snap601
					d229 = snap602
					d230 = snap603
					d231 = snap604
					d232 = snap605
					d233 = snap606
					d234 = snap607
					d235 = snap608
					d236 = snap609
					d237 = snap610
					d238 = snap611
					d239 = snap612
					d240 = snap613
					d241 = snap614
					d242 = snap615
					d243 = snap616
					d244 = snap617
					d245 = snap618
					d246 = snap619
					d247 = snap620
					d248 = snap621
					d349 = snap622
					d350 = snap623
					d351 = snap624
					d352 = snap625
					d353 = snap626
					d354 = snap627
					d355 = snap628
					d356 = snap629
					d357 = snap630
					d358 = snap631
					d359 = snap632
					d360 = snap633
					d485 = snap634
					d486 = snap635
					d487 = snap636
					d488 = snap637
					d489 = snap638
					d490 = snap639
					d491 = snap640
					d492 = snap641
					d493 = snap642
					d494 = snap643
					d495 = snap644
					d496 = snap645
					d497 = snap646
					if !bbs[12].Rendered {
						return bbs[12].RenderPS(ps573)
					}
					return result
					ctx.FreeDesc(&d496)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					ctx.EnsureDesc(&d105)
					bbpos_5_0 := int32(-1)
					_ = bbpos_5_0
					lbl26 := ctx.ReserveLabel()
					_ = lbl26
					bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d648 JITValueDesc
					if d105.Loc == LocImm {
						d648 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() / 3600000000000)}
					} else {
						r20 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r20, d105.Reg)
						ctx.EmitIdivRegImm(r20, 3600000000000)
						d648 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
						ctx.BindReg(r20, &d648)
					}
					if d648.Loc == LocReg && d105.Loc == LocReg && d648.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					var d649 JITValueDesc
					if d105.Loc == LocImm {
						d649 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() % 3600000000000)}
					} else {
						ctx.EmitIremRegImm(d105.Reg, 3600000000000)
						d649 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d105.Reg}
						ctx.BindReg(d105.Reg, &d649)
					}
					if d649.Loc == LocReg && d105.Loc == LocReg && d649.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d648)
					ctx.EnsureDesc(&d648)
					var d650 JITValueDesc
					if d648.Loc == LocImm {
						d650 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d648.Imm.Int()))}
					} else {
						r21 := ctx.AllocRegExcept(d648.Reg)
						ctx.EmitMovRegReg(r21, d648.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r21)
						d650 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r21}
						ctx.BindReg(r21, &d650)
					}
					ctx.FreeDesc(&d648)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d649)
					ctx.EnsureDesc(&d649)
					var d651 JITValueDesc
					if d649.Loc == LocImm {
						d651 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d649.Imm.Int()))}
					} else {
						r22 := ctx.AllocRegExcept(d649.Reg)
						ctx.EmitMovRegReg(r22, d649.Reg)
						ctx.EmitCvtInt64ToFloat64(RegX0, r22)
						d651 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r22}
						ctx.BindReg(r22, &d651)
					}
					ctx.FreeDesc(&d649)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d651)
					var d652 JITValueDesc
					if d651.Loc == LocImm {
						d652 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d651.Imm.Float() / 3.6e+12)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4794699203894837248))
						ctx.EmitDivFloat64(d651.Reg, RegR11)
						d652 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d651.Reg}
						ctx.BindReg(d651.Reg, &d652)
					}
					if d652.Loc == LocReg && d651.Loc == LocReg && d652.Reg == d651.Reg {
						ctx.TransferReg(d651.Reg)
						d651.Loc = LocNone
					}
					ctx.FreeDesc(&d651)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d650)
					ctx.EnsureDesc(&d652)
					ctx.EnsureDescsTogether(&d650, &d652)
					var d653 JITValueDesc
					if d650.Loc == LocImm && d652.Loc == LocImm {
						d653 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d650.Imm.Float() + d652.Imm.Float())}
					} else if d650.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d652.Reg)
						_, xBits := d650.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d652.Reg)
						d653 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d653)
					} else if d652.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d650.Reg)
						ctx.EmitMovRegReg(scratch, d650.Reg)
						_, yBits := d652.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d653 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d653)
					} else {
						r23 := ctx.AllocRegExcept(d650.Reg, d652.Reg)
						ctx.EmitMovRegReg(r23, d650.Reg)
						ctx.EmitAddFloat64(r23, d652.Reg)
						d653 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r23}
						ctx.BindReg(r23, &d653)
					}
					if d653.Loc == LocReg && d650.Loc == LocReg && d653.Reg == d650.Reg {
						ctx.TransferReg(d650.Reg)
						d650.Loc = LocNone
					}
					ctx.FreeDesc(&d650)
					ctx.FreeDesc(&d652)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d653)
					ctx.EnsureDesc(&d653)
					var d654 JITValueDesc
					if d653.Loc == LocImm {
						d654 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d653.Imm.Float() / 168)}
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(4640114991075164160))
						ctx.EmitDivFloat64(d653.Reg, RegR11)
						d654 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d653.Reg}
						ctx.BindReg(d653.Reg, &d654)
					}
					if d654.Loc == LocReg && d653.Loc == LocReg && d654.Reg == d653.Reg {
						ctx.TransferReg(d653.Reg)
						d653.Loc = LocNone
					}
					ctx.FreeDesc(&d653)
					ctx.EnsureDesc(&d654)
					ctx.EnsureDesc(&d654)
					var d655 JITValueDesc
					if d654.Loc == LocImm {
						d655 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d654.Imm.Float()))}
					} else {
						r24 := ctx.AllocReg()
						ctx.EmitCvtFloatBitsToInt64(r24, d654.Reg)
						d655 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
						ctx.BindReg(r24, &d655)
					}
					ctx.FreeDesc(&d654)
					ctx.EnsureDesc(&d655)
					if d655.Loc == LocImm {
						ctx.EmitMakeInt(result, d655)
					} else {
						ctx.EmitMovToReg(result.Reg2, d655)
						d656 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d656)
						if d655.Loc == LocReg && d655.Reg != result.Reg2 {
							ctx.FreeReg(d655.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != LocNone {
						d648 = ps.OverlayValues[648]
					}
					if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != LocNone {
						d649 = ps.OverlayValues[649]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					d657 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("WEEK")}
					var d658 JITValueDesc
					if d657.Loc == LocImm {
						ctx.TrackImm(d657.Imm)
						ptrWord, _ := d657.Imm.RawWords()
						d658 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d658.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d658.Reg2, uint64(len(d657.Imm.String())))
						ctx.BindReg(d658.Reg, &d658)
						ctx.BindReg(d658.Reg2, &d658)
					} else {
						d658 = d657
					}
					d659 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d658}, 1)
					ctx.EmitAndRegImm32(d659.Reg, 1)
					d659.Type = tagBool
					ctx.BindReg(d659.Reg, &d659)
					d660 = d659
					ctx.EnsureDesc(&d660)
					if d660.Loc != LocImm && d660.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d660.Loc == LocImm {
						if d660.Imm.Bool() {
							if ps.General {
							}
							ps661 := PhiState{General: ps.General}
							ps661.OverlayValues = make([]JITValueDesc, 661)
							ps661.OverlayValues[0] = d0
							ps661.OverlayValues[1] = d1
							ps661.OverlayValues[2] = d2
							ps661.OverlayValues[3] = d3
							ps661.OverlayValues[18] = d18
							ps661.OverlayValues[19] = d19
							ps661.OverlayValues[21] = d21
							ps661.OverlayValues[22] = d22
							ps661.OverlayValues[23] = d23
							ps661.OverlayValues[25] = d25
							ps661.OverlayValues[26] = d26
							ps661.OverlayValues[27] = d27
							ps661.OverlayValues[58] = d58
							ps661.OverlayValues[59] = d59
							ps661.OverlayValues[60] = d60
							ps661.OverlayValues[61] = d61
							ps661.OverlayValues[100] = d100
							ps661.OverlayValues[101] = d101
							ps661.OverlayValues[102] = d102
							ps661.OverlayValues[103] = d103
							ps661.OverlayValues[104] = d104
							ps661.OverlayValues[105] = d105
							ps661.OverlayValues[106] = d106
							ps661.OverlayValues[107] = d107
							ps661.OverlayValues[108] = d108
							ps661.OverlayValues[109] = d109
							ps661.OverlayValues[168] = d168
							ps661.OverlayValues[229] = d229
							ps661.OverlayValues[230] = d230
							ps661.OverlayValues[231] = d231
							ps661.OverlayValues[232] = d232
							ps661.OverlayValues[233] = d233
							ps661.OverlayValues[234] = d234
							ps661.OverlayValues[235] = d235
							ps661.OverlayValues[236] = d236
							ps661.OverlayValues[237] = d237
							ps661.OverlayValues[238] = d238
							ps661.OverlayValues[239] = d239
							ps661.OverlayValues[240] = d240
							ps661.OverlayValues[241] = d241
							ps661.OverlayValues[242] = d242
							ps661.OverlayValues[243] = d243
							ps661.OverlayValues[244] = d244
							ps661.OverlayValues[245] = d245
							ps661.OverlayValues[246] = d246
							ps661.OverlayValues[247] = d247
							ps661.OverlayValues[248] = d248
							ps661.OverlayValues[349] = d349
							ps661.OverlayValues[350] = d350
							ps661.OverlayValues[351] = d351
							ps661.OverlayValues[352] = d352
							ps661.OverlayValues[353] = d353
							ps661.OverlayValues[354] = d354
							ps661.OverlayValues[355] = d355
							ps661.OverlayValues[356] = d356
							ps661.OverlayValues[357] = d357
							ps661.OverlayValues[358] = d358
							ps661.OverlayValues[359] = d359
							ps661.OverlayValues[360] = d360
							ps661.OverlayValues[485] = d485
							ps661.OverlayValues[486] = d486
							ps661.OverlayValues[487] = d487
							ps661.OverlayValues[488] = d488
							ps661.OverlayValues[489] = d489
							ps661.OverlayValues[490] = d490
							ps661.OverlayValues[491] = d491
							ps661.OverlayValues[492] = d492
							ps661.OverlayValues[493] = d493
							ps661.OverlayValues[494] = d494
							ps661.OverlayValues[495] = d495
							ps661.OverlayValues[496] = d496
							ps661.OverlayValues[497] = d497
							ps661.OverlayValues[648] = d648
							ps661.OverlayValues[649] = d649
							ps661.OverlayValues[650] = d650
							ps661.OverlayValues[651] = d651
							ps661.OverlayValues[652] = d652
							ps661.OverlayValues[653] = d653
							ps661.OverlayValues[654] = d654
							ps661.OverlayValues[655] = d655
							ps661.OverlayValues[656] = d656
							ps661.OverlayValues[657] = d657
							ps661.OverlayValues[658] = d658
							ps661.OverlayValues[659] = d659
							ps661.OverlayValues[660] = d660
							return bbs[14].RenderPS(ps661)
						}
						if ps.General {
						}
						ps662 := PhiState{General: ps.General}
						ps662.OverlayValues = make([]JITValueDesc, 661)
						ps662.OverlayValues[0] = d0
						ps662.OverlayValues[1] = d1
						ps662.OverlayValues[2] = d2
						ps662.OverlayValues[3] = d3
						ps662.OverlayValues[18] = d18
						ps662.OverlayValues[19] = d19
						ps662.OverlayValues[21] = d21
						ps662.OverlayValues[22] = d22
						ps662.OverlayValues[23] = d23
						ps662.OverlayValues[25] = d25
						ps662.OverlayValues[26] = d26
						ps662.OverlayValues[27] = d27
						ps662.OverlayValues[58] = d58
						ps662.OverlayValues[59] = d59
						ps662.OverlayValues[60] = d60
						ps662.OverlayValues[61] = d61
						ps662.OverlayValues[100] = d100
						ps662.OverlayValues[101] = d101
						ps662.OverlayValues[102] = d102
						ps662.OverlayValues[103] = d103
						ps662.OverlayValues[104] = d104
						ps662.OverlayValues[105] = d105
						ps662.OverlayValues[106] = d106
						ps662.OverlayValues[107] = d107
						ps662.OverlayValues[108] = d108
						ps662.OverlayValues[109] = d109
						ps662.OverlayValues[168] = d168
						ps662.OverlayValues[229] = d229
						ps662.OverlayValues[230] = d230
						ps662.OverlayValues[231] = d231
						ps662.OverlayValues[232] = d232
						ps662.OverlayValues[233] = d233
						ps662.OverlayValues[234] = d234
						ps662.OverlayValues[235] = d235
						ps662.OverlayValues[236] = d236
						ps662.OverlayValues[237] = d237
						ps662.OverlayValues[238] = d238
						ps662.OverlayValues[239] = d239
						ps662.OverlayValues[240] = d240
						ps662.OverlayValues[241] = d241
						ps662.OverlayValues[242] = d242
						ps662.OverlayValues[243] = d243
						ps662.OverlayValues[244] = d244
						ps662.OverlayValues[245] = d245
						ps662.OverlayValues[246] = d246
						ps662.OverlayValues[247] = d247
						ps662.OverlayValues[248] = d248
						ps662.OverlayValues[349] = d349
						ps662.OverlayValues[350] = d350
						ps662.OverlayValues[351] = d351
						ps662.OverlayValues[352] = d352
						ps662.OverlayValues[353] = d353
						ps662.OverlayValues[354] = d354
						ps662.OverlayValues[355] = d355
						ps662.OverlayValues[356] = d356
						ps662.OverlayValues[357] = d357
						ps662.OverlayValues[358] = d358
						ps662.OverlayValues[359] = d359
						ps662.OverlayValues[360] = d360
						ps662.OverlayValues[485] = d485
						ps662.OverlayValues[486] = d486
						ps662.OverlayValues[487] = d487
						ps662.OverlayValues[488] = d488
						ps662.OverlayValues[489] = d489
						ps662.OverlayValues[490] = d490
						ps662.OverlayValues[491] = d491
						ps662.OverlayValues[492] = d492
						ps662.OverlayValues[493] = d493
						ps662.OverlayValues[494] = d494
						ps662.OverlayValues[495] = d495
						ps662.OverlayValues[496] = d496
						ps662.OverlayValues[497] = d497
						ps662.OverlayValues[648] = d648
						ps662.OverlayValues[649] = d649
						ps662.OverlayValues[650] = d650
						ps662.OverlayValues[651] = d651
						ps662.OverlayValues[652] = d652
						ps662.OverlayValues[653] = d653
						ps662.OverlayValues[654] = d654
						ps662.OverlayValues[655] = d655
						ps662.OverlayValues[656] = d656
						ps662.OverlayValues[657] = d657
						ps662.OverlayValues[658] = d658
						ps662.OverlayValues[659] = d659
						ps662.OverlayValues[660] = d660
						return bbs[17].RenderPS(ps662)
					}
					if !ps.General {
						ps.General = true
						return bbs[15].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d660.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					snap663 := d0
					snap664 := d1
					snap665 := d2
					snap666 := d3
					snap667 := d18
					snap668 := d19
					snap669 := d21
					snap670 := d22
					snap671 := d23
					snap672 := d25
					snap673 := d26
					snap674 := d27
					snap675 := d58
					snap676 := d59
					snap677 := d60
					snap678 := d61
					snap679 := d100
					snap680 := d101
					snap681 := d102
					snap682 := d103
					snap683 := d104
					snap684 := d105
					snap685 := d106
					snap686 := d107
					snap687 := d108
					snap688 := d109
					snap689 := d168
					snap690 := d229
					snap691 := d230
					snap692 := d231
					snap693 := d232
					snap694 := d233
					snap695 := d234
					snap696 := d235
					snap697 := d236
					snap698 := d237
					snap699 := d238
					snap700 := d239
					snap701 := d240
					snap702 := d241
					snap703 := d242
					snap704 := d243
					snap705 := d244
					snap706 := d245
					snap707 := d246
					snap708 := d247
					snap709 := d248
					snap710 := d349
					snap711 := d350
					snap712 := d351
					snap713 := d352
					snap714 := d353
					snap715 := d354
					snap716 := d355
					snap717 := d356
					snap718 := d357
					snap719 := d358
					snap720 := d359
					snap721 := d360
					snap722 := d485
					snap723 := d486
					snap724 := d487
					snap725 := d488
					snap726 := d489
					snap727 := d490
					snap728 := d491
					snap729 := d492
					snap730 := d493
					snap731 := d494
					snap732 := d495
					snap733 := d496
					snap734 := d497
					snap735 := d648
					snap736 := d649
					snap737 := d650
					snap738 := d651
					snap739 := d652
					snap740 := d653
					snap741 := d654
					snap742 := d655
					snap743 := d656
					snap744 := d657
					snap745 := d658
					snap746 := d659
					snap747 := d660
					alloc748 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc748)
					d0 = snap663
					d1 = snap664
					d2 = snap665
					d3 = snap666
					d18 = snap667
					d19 = snap668
					d21 = snap669
					d22 = snap670
					d23 = snap671
					d25 = snap672
					d26 = snap673
					d27 = snap674
					d58 = snap675
					d59 = snap676
					d60 = snap677
					d61 = snap678
					d100 = snap679
					d101 = snap680
					d102 = snap681
					d103 = snap682
					d104 = snap683
					d105 = snap684
					d106 = snap685
					d107 = snap686
					d108 = snap687
					d109 = snap688
					d168 = snap689
					d229 = snap690
					d230 = snap691
					d231 = snap692
					d232 = snap693
					d233 = snap694
					d234 = snap695
					d235 = snap696
					d236 = snap697
					d237 = snap698
					d238 = snap699
					d239 = snap700
					d240 = snap701
					d241 = snap702
					d242 = snap703
					d243 = snap704
					d244 = snap705
					d245 = snap706
					d246 = snap707
					d247 = snap708
					d248 = snap709
					d349 = snap710
					d350 = snap711
					d351 = snap712
					d352 = snap713
					d353 = snap714
					d354 = snap715
					d355 = snap716
					d356 = snap717
					d357 = snap718
					d358 = snap719
					d359 = snap720
					d360 = snap721
					d485 = snap722
					d486 = snap723
					d487 = snap724
					d488 = snap725
					d489 = snap726
					d490 = snap727
					d491 = snap728
					d492 = snap729
					d493 = snap730
					d494 = snap731
					d495 = snap732
					d496 = snap733
					d497 = snap734
					d648 = snap735
					d649 = snap736
					d650 = snap737
					d651 = snap738
					d652 = snap739
					d653 = snap740
					d654 = snap741
					d655 = snap742
					d656 = snap743
					d657 = snap744
					d658 = snap745
					d659 = snap746
					d660 = snap747
					ctx.RestoreAllocState(alloc748)
					d0 = snap663
					d1 = snap664
					d2 = snap665
					d3 = snap666
					d18 = snap667
					d19 = snap668
					d21 = snap669
					d22 = snap670
					d23 = snap671
					d25 = snap672
					d26 = snap673
					d27 = snap674
					d58 = snap675
					d59 = snap676
					d60 = snap677
					d61 = snap678
					d100 = snap679
					d101 = snap680
					d102 = snap681
					d103 = snap682
					d104 = snap683
					d105 = snap684
					d106 = snap685
					d107 = snap686
					d108 = snap687
					d109 = snap688
					d168 = snap689
					d229 = snap690
					d230 = snap691
					d231 = snap692
					d232 = snap693
					d233 = snap694
					d234 = snap695
					d235 = snap696
					d236 = snap697
					d237 = snap698
					d238 = snap699
					d239 = snap700
					d240 = snap701
					d241 = snap702
					d242 = snap703
					d243 = snap704
					d244 = snap705
					d245 = snap706
					d246 = snap707
					d247 = snap708
					d248 = snap709
					d349 = snap710
					d350 = snap711
					d351 = snap712
					d352 = snap713
					d353 = snap714
					d354 = snap715
					d355 = snap716
					d356 = snap717
					d357 = snap718
					d358 = snap719
					d359 = snap720
					d360 = snap721
					d485 = snap722
					d486 = snap723
					d487 = snap724
					d488 = snap725
					d489 = snap726
					d490 = snap727
					d491 = snap728
					d492 = snap729
					d493 = snap730
					d494 = snap731
					d495 = snap732
					d496 = snap733
					d497 = snap734
					d648 = snap735
					d649 = snap736
					d650 = snap737
					d651 = snap738
					d652 = snap739
					d653 = snap740
					d654 = snap741
					d655 = snap742
					d656 = snap743
					d657 = snap744
					d658 = snap745
					d659 = snap746
					d660 = snap747
					ps749 := PhiState{General: true}
					ps749.OverlayValues = make([]JITValueDesc, 661)
					ps749.OverlayValues[0] = d0
					ps749.OverlayValues[1] = d1
					ps749.OverlayValues[2] = d2
					ps749.OverlayValues[3] = d3
					ps749.OverlayValues[18] = d18
					ps749.OverlayValues[19] = d19
					ps749.OverlayValues[21] = d21
					ps749.OverlayValues[22] = d22
					ps749.OverlayValues[23] = d23
					ps749.OverlayValues[25] = d25
					ps749.OverlayValues[26] = d26
					ps749.OverlayValues[27] = d27
					ps749.OverlayValues[58] = d58
					ps749.OverlayValues[59] = d59
					ps749.OverlayValues[60] = d60
					ps749.OverlayValues[61] = d61
					ps749.OverlayValues[100] = d100
					ps749.OverlayValues[101] = d101
					ps749.OverlayValues[102] = d102
					ps749.OverlayValues[103] = d103
					ps749.OverlayValues[104] = d104
					ps749.OverlayValues[105] = d105
					ps749.OverlayValues[106] = d106
					ps749.OverlayValues[107] = d107
					ps749.OverlayValues[108] = d108
					ps749.OverlayValues[109] = d109
					ps749.OverlayValues[168] = d168
					ps749.OverlayValues[229] = d229
					ps749.OverlayValues[230] = d230
					ps749.OverlayValues[231] = d231
					ps749.OverlayValues[232] = d232
					ps749.OverlayValues[233] = d233
					ps749.OverlayValues[234] = d234
					ps749.OverlayValues[235] = d235
					ps749.OverlayValues[236] = d236
					ps749.OverlayValues[237] = d237
					ps749.OverlayValues[238] = d238
					ps749.OverlayValues[239] = d239
					ps749.OverlayValues[240] = d240
					ps749.OverlayValues[241] = d241
					ps749.OverlayValues[242] = d242
					ps749.OverlayValues[243] = d243
					ps749.OverlayValues[244] = d244
					ps749.OverlayValues[245] = d245
					ps749.OverlayValues[246] = d246
					ps749.OverlayValues[247] = d247
					ps749.OverlayValues[248] = d248
					ps749.OverlayValues[349] = d349
					ps749.OverlayValues[350] = d350
					ps749.OverlayValues[351] = d351
					ps749.OverlayValues[352] = d352
					ps749.OverlayValues[353] = d353
					ps749.OverlayValues[354] = d354
					ps749.OverlayValues[355] = d355
					ps749.OverlayValues[356] = d356
					ps749.OverlayValues[357] = d357
					ps749.OverlayValues[358] = d358
					ps749.OverlayValues[359] = d359
					ps749.OverlayValues[360] = d360
					ps749.OverlayValues[485] = d485
					ps749.OverlayValues[486] = d486
					ps749.OverlayValues[487] = d487
					ps749.OverlayValues[488] = d488
					ps749.OverlayValues[489] = d489
					ps749.OverlayValues[490] = d490
					ps749.OverlayValues[491] = d491
					ps749.OverlayValues[492] = d492
					ps749.OverlayValues[493] = d493
					ps749.OverlayValues[494] = d494
					ps749.OverlayValues[495] = d495
					ps749.OverlayValues[496] = d496
					ps749.OverlayValues[497] = d497
					ps749.OverlayValues[648] = d648
					ps749.OverlayValues[649] = d649
					ps749.OverlayValues[650] = d650
					ps749.OverlayValues[651] = d651
					ps749.OverlayValues[652] = d652
					ps749.OverlayValues[653] = d653
					ps749.OverlayValues[654] = d654
					ps749.OverlayValues[655] = d655
					ps749.OverlayValues[656] = d656
					ps749.OverlayValues[657] = d657
					ps749.OverlayValues[658] = d658
					ps749.OverlayValues[659] = d659
					ps749.OverlayValues[660] = d660
					ps750 := PhiState{General: true}
					ps750.OverlayValues = make([]JITValueDesc, 661)
					ps750.OverlayValues[0] = d0
					ps750.OverlayValues[1] = d1
					ps750.OverlayValues[2] = d2
					ps750.OverlayValues[3] = d3
					ps750.OverlayValues[18] = d18
					ps750.OverlayValues[19] = d19
					ps750.OverlayValues[21] = d21
					ps750.OverlayValues[22] = d22
					ps750.OverlayValues[23] = d23
					ps750.OverlayValues[25] = d25
					ps750.OverlayValues[26] = d26
					ps750.OverlayValues[27] = d27
					ps750.OverlayValues[58] = d58
					ps750.OverlayValues[59] = d59
					ps750.OverlayValues[60] = d60
					ps750.OverlayValues[61] = d61
					ps750.OverlayValues[100] = d100
					ps750.OverlayValues[101] = d101
					ps750.OverlayValues[102] = d102
					ps750.OverlayValues[103] = d103
					ps750.OverlayValues[104] = d104
					ps750.OverlayValues[105] = d105
					ps750.OverlayValues[106] = d106
					ps750.OverlayValues[107] = d107
					ps750.OverlayValues[108] = d108
					ps750.OverlayValues[109] = d109
					ps750.OverlayValues[168] = d168
					ps750.OverlayValues[229] = d229
					ps750.OverlayValues[230] = d230
					ps750.OverlayValues[231] = d231
					ps750.OverlayValues[232] = d232
					ps750.OverlayValues[233] = d233
					ps750.OverlayValues[234] = d234
					ps750.OverlayValues[235] = d235
					ps750.OverlayValues[236] = d236
					ps750.OverlayValues[237] = d237
					ps750.OverlayValues[238] = d238
					ps750.OverlayValues[239] = d239
					ps750.OverlayValues[240] = d240
					ps750.OverlayValues[241] = d241
					ps750.OverlayValues[242] = d242
					ps750.OverlayValues[243] = d243
					ps750.OverlayValues[244] = d244
					ps750.OverlayValues[245] = d245
					ps750.OverlayValues[246] = d246
					ps750.OverlayValues[247] = d247
					ps750.OverlayValues[248] = d248
					ps750.OverlayValues[349] = d349
					ps750.OverlayValues[350] = d350
					ps750.OverlayValues[351] = d351
					ps750.OverlayValues[352] = d352
					ps750.OverlayValues[353] = d353
					ps750.OverlayValues[354] = d354
					ps750.OverlayValues[355] = d355
					ps750.OverlayValues[356] = d356
					ps750.OverlayValues[357] = d357
					ps750.OverlayValues[358] = d358
					ps750.OverlayValues[359] = d359
					ps750.OverlayValues[360] = d360
					ps750.OverlayValues[485] = d485
					ps750.OverlayValues[486] = d486
					ps750.OverlayValues[487] = d487
					ps750.OverlayValues[488] = d488
					ps750.OverlayValues[489] = d489
					ps750.OverlayValues[490] = d490
					ps750.OverlayValues[491] = d491
					ps750.OverlayValues[492] = d492
					ps750.OverlayValues[493] = d493
					ps750.OverlayValues[494] = d494
					ps750.OverlayValues[495] = d495
					ps750.OverlayValues[496] = d496
					ps750.OverlayValues[497] = d497
					ps750.OverlayValues[648] = d648
					ps750.OverlayValues[649] = d649
					ps750.OverlayValues[650] = d650
					ps750.OverlayValues[651] = d651
					ps750.OverlayValues[652] = d652
					ps750.OverlayValues[653] = d653
					ps750.OverlayValues[654] = d654
					ps750.OverlayValues[655] = d655
					ps750.OverlayValues[656] = d656
					ps750.OverlayValues[657] = d657
					ps750.OverlayValues[658] = d658
					ps750.OverlayValues[659] = d659
					ps750.OverlayValues[660] = d660
					snap751 := d0
					snap752 := d1
					snap753 := d2
					snap754 := d3
					snap755 := d18
					snap756 := d19
					snap757 := d21
					snap758 := d22
					snap759 := d23
					snap760 := d25
					snap761 := d26
					snap762 := d27
					snap763 := d58
					snap764 := d59
					snap765 := d60
					snap766 := d61
					snap767 := d100
					snap768 := d101
					snap769 := d102
					snap770 := d103
					snap771 := d104
					snap772 := d105
					snap773 := d106
					snap774 := d107
					snap775 := d108
					snap776 := d109
					snap777 := d168
					snap778 := d229
					snap779 := d230
					snap780 := d231
					snap781 := d232
					snap782 := d233
					snap783 := d234
					snap784 := d235
					snap785 := d236
					snap786 := d237
					snap787 := d238
					snap788 := d239
					snap789 := d240
					snap790 := d241
					snap791 := d242
					snap792 := d243
					snap793 := d244
					snap794 := d245
					snap795 := d246
					snap796 := d247
					snap797 := d248
					snap798 := d349
					snap799 := d350
					snap800 := d351
					snap801 := d352
					snap802 := d353
					snap803 := d354
					snap804 := d355
					snap805 := d356
					snap806 := d357
					snap807 := d358
					snap808 := d359
					snap809 := d360
					snap810 := d485
					snap811 := d486
					snap812 := d487
					snap813 := d488
					snap814 := d489
					snap815 := d490
					snap816 := d491
					snap817 := d492
					snap818 := d493
					snap819 := d494
					snap820 := d495
					snap821 := d496
					snap822 := d497
					snap823 := d648
					snap824 := d649
					snap825 := d650
					snap826 := d651
					snap827 := d652
					snap828 := d653
					snap829 := d654
					snap830 := d655
					snap831 := d656
					snap832 := d657
					snap833 := d658
					snap834 := d659
					snap835 := d660
					alloc836 := ctx.SnapshotAllocState()
					if !bbs[17].Rendered {
						bbs[17].RenderPS(ps750)
					}
					ctx.RestoreAllocState(alloc836)
					d0 = snap751
					d1 = snap752
					d2 = snap753
					d3 = snap754
					d18 = snap755
					d19 = snap756
					d21 = snap757
					d22 = snap758
					d23 = snap759
					d25 = snap760
					d26 = snap761
					d27 = snap762
					d58 = snap763
					d59 = snap764
					d60 = snap765
					d61 = snap766
					d100 = snap767
					d101 = snap768
					d102 = snap769
					d103 = snap770
					d104 = snap771
					d105 = snap772
					d106 = snap773
					d107 = snap774
					d108 = snap775
					d109 = snap776
					d168 = snap777
					d229 = snap778
					d230 = snap779
					d231 = snap780
					d232 = snap781
					d233 = snap782
					d234 = snap783
					d235 = snap784
					d236 = snap785
					d237 = snap786
					d238 = snap787
					d239 = snap788
					d240 = snap789
					d241 = snap790
					d242 = snap791
					d243 = snap792
					d244 = snap793
					d245 = snap794
					d246 = snap795
					d247 = snap796
					d248 = snap797
					d349 = snap798
					d350 = snap799
					d351 = snap800
					d352 = snap801
					d353 = snap802
					d354 = snap803
					d355 = snap804
					d356 = snap805
					d357 = snap806
					d358 = snap807
					d359 = snap808
					d360 = snap809
					d485 = snap810
					d486 = snap811
					d487 = snap812
					d488 = snap813
					d489 = snap814
					d490 = snap815
					d491 = snap816
					d492 = snap817
					d493 = snap818
					d494 = snap819
					d495 = snap820
					d496 = snap821
					d497 = snap822
					d648 = snap823
					d649 = snap824
					d650 = snap825
					d651 = snap826
					d652 = snap827
					d653 = snap828
					d654 = snap829
					d655 = snap830
					d656 = snap831
					d657 = snap832
					d658 = snap833
					d659 = snap834
					d660 = snap835
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps749)
					}
					return result
					ctx.FreeDesc(&d659)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != LocNone {
						d648 = ps.OverlayValues[648]
					}
					if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != LocNone {
						d649 = ps.OverlayValues[649]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocRegTriple && d21.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d21)
					callResults837 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d21}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d838 = callResults837[0]
					_ = d838
					d839 = callResults837[1]
					_ = d839
					d840 = callResults837[2]
					_ = d840
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocRegTriple && d25.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d25)
					callResults841 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d25}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d842 = callResults841[0]
					_ = d842
					d843 = callResults841[1]
					_ = d843
					d844 = callResults841[2]
					_ = d844
					ctx.EnsureDesc(&d842)
					ctx.EnsureDesc(&d838)
					ctx.EnsureDescsTogether(&d842, &d838)
					var d845 JITValueDesc
					if d842.Loc == LocImm && d838.Loc == LocImm {
						d845 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d842.Imm.Int() - d838.Imm.Int())}
					} else if d838.Loc == LocImm && d838.Imm.Int() == 0 {
						r25 := ctx.AllocRegExcept(d842.Reg)
						ctx.EmitMovRegReg(r25, d842.Reg)
						d845 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d845)
					} else if d842.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d838.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d842.Imm.Int()))
						ctx.EmitSubInt64(scratch, d838.Reg)
						d845 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d845)
					} else if d838.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d842.Reg)
						ctx.EmitMovRegReg(scratch, d842.Reg)
						if d838.Imm.Int() >= -2147483648 && d838.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d838.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d838.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d845 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d845)
					} else {
						r26 := ctx.AllocRegExcept(d842.Reg, d838.Reg)
						ctx.EmitMovRegReg(r26, d842.Reg)
						ctx.EmitSubInt64(r26, d838.Reg)
						d845 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
						ctx.BindReg(r26, &d845)
					}
					if d845.Loc == LocReg && d842.Loc == LocReg && d845.Reg == d842.Reg {
						ctx.TransferReg(d842.Reg)
						d842.Loc = LocNone
					}
					ctx.FreeDesc(&d842)
					ctx.FreeDesc(&d838)
					ctx.EnsureDesc(&d845)
					ctx.EnsureDesc(&d845)
					var d846 JITValueDesc
					if d845.Loc == LocImm {
						d846 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d845.Imm.Int() * 12)}
					} else {
						ctx.EmitImulRegImm32(d845.Reg, int32(12))
						d846 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d845.Reg}
						ctx.BindReg(d845.Reg, &d846)
					}
					if d846.Loc == LocReg && d845.Loc == LocReg && d846.Reg == d845.Reg {
						ctx.TransferReg(d845.Reg)
						d845.Loc = LocNone
					}
					ctx.FreeDesc(&d845)
					ctx.EnsureDesc(&d843)
					ctx.EnsureDesc(&d839)
					ctx.EnsureDescsTogether(&d843, &d839)
					var d847 JITValueDesc
					if d843.Loc == LocImm && d839.Loc == LocImm {
						d847 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d843.Imm.Int() - d839.Imm.Int())}
					} else if d839.Loc == LocImm && d839.Imm.Int() == 0 {
						r27 := ctx.AllocRegExcept(d843.Reg)
						ctx.EmitMovRegReg(r27, d843.Reg)
						d847 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
						ctx.BindReg(r27, &d847)
					} else if d843.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d839.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d843.Imm.Int()))
						ctx.EmitSubInt64(scratch, d839.Reg)
						d847 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d847)
					} else if d839.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d843.Reg)
						ctx.EmitMovRegReg(scratch, d843.Reg)
						if d839.Imm.Int() >= -2147483648 && d839.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d839.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d839.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d847 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d847)
					} else {
						r28 := ctx.AllocRegExcept(d843.Reg, d839.Reg)
						ctx.EmitMovRegReg(r28, d843.Reg)
						ctx.EmitSubInt64(r28, d839.Reg)
						d847 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r28}
						ctx.BindReg(r28, &d847)
					}
					if d847.Loc == LocReg && d843.Loc == LocReg && d847.Reg == d843.Reg {
						ctx.TransferReg(d843.Reg)
						d843.Loc = LocNone
					}
					ctx.FreeDesc(&d843)
					ctx.FreeDesc(&d839)
					ctx.EnsureDesc(&d847)
					ctx.FreeDesc(&d847)
					ctx.EnsureDesc(&d846)
					ctx.EnsureDesc(&d847)
					ctx.EnsureDescsTogether(&d846, &d847)
					var d848 JITValueDesc
					if d846.Loc == LocImm && d847.Loc == LocImm {
						d848 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d846.Imm.Int() + d847.Imm.Int())}
					} else if d847.Loc == LocImm && d847.Imm.Int() == 0 {
						r29 := ctx.AllocRegExcept(d846.Reg)
						ctx.EmitMovRegReg(r29, d846.Reg)
						d848 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r29}
						ctx.BindReg(r29, &d848)
					} else if d846.Loc == LocImm && d846.Imm.Int() == 0 {
						d848 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d847.Reg}
						ctx.BindReg(d847.Reg, &d848)
					} else if d846.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d847.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d846.Imm.Int()))
						ctx.EmitAddInt64(scratch, d847.Reg)
						d848 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d848)
					} else if d847.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d846.Reg)
						ctx.EmitMovRegReg(scratch, d846.Reg)
						if d847.Imm.Int() >= -2147483648 && d847.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d847.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d847.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d848 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d848)
					} else {
						r30 := ctx.AllocRegExcept(d846.Reg, d847.Reg)
						ctx.EmitMovRegReg(r30, d846.Reg)
						ctx.EmitAddInt64(r30, d847.Reg)
						d848 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
						ctx.BindReg(r30, &d848)
					}
					if d848.Loc == LocReg && d846.Loc == LocReg && d848.Reg == d846.Reg {
						ctx.TransferReg(d846.Reg)
						d846.Loc = LocNone
					}
					ctx.FreeDesc(&d846)
					ctx.FreeDesc(&d847)
					ctx.EnsureDesc(&d848)
					ctx.EnsureDesc(&d848)
					ctx.EnsureDesc(&d848)
					if d848.Loc == LocImm {
						ctx.EmitMakeInt(result, d848)
					} else {
						ctx.EmitMovToReg(result.Reg2, d848)
						d850 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d850)
						if d848.Loc == LocReg && d848.Reg != result.Reg2 {
							ctx.FreeReg(d848.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != LocNone {
						d648 = ps.OverlayValues[648]
					}
					if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != LocNone {
						d649 = ps.OverlayValues[649]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != LocNone {
						d838 = ps.OverlayValues[838]
					}
					if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != LocNone {
						d839 = ps.OverlayValues[839]
					}
					if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != LocNone {
						d840 = ps.OverlayValues[840]
					}
					if len(ps.OverlayValues) > 842 && ps.OverlayValues[842].Loc != LocNone {
						d842 = ps.OverlayValues[842]
					}
					if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != LocNone {
						d843 = ps.OverlayValues[843]
					}
					if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != LocNone {
						d844 = ps.OverlayValues[844]
					}
					if len(ps.OverlayValues) > 845 && ps.OverlayValues[845].Loc != LocNone {
						d845 = ps.OverlayValues[845]
					}
					if len(ps.OverlayValues) > 846 && ps.OverlayValues[846].Loc != LocNone {
						d846 = ps.OverlayValues[846]
					}
					if len(ps.OverlayValues) > 847 && ps.OverlayValues[847].Loc != LocNone {
						d847 = ps.OverlayValues[847]
					}
					if len(ps.OverlayValues) > 848 && ps.OverlayValues[848].Loc != LocNone {
						d848 = ps.OverlayValues[848]
					}
					if len(ps.OverlayValues) > 849 && ps.OverlayValues[849].Loc != LocNone {
						d849 = ps.OverlayValues[849]
					}
					if len(ps.OverlayValues) > 850 && ps.OverlayValues[850].Loc != LocNone {
						d850 = ps.OverlayValues[850]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					d851 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("MONTH")}
					var d852 JITValueDesc
					if d851.Loc == LocImm {
						ctx.TrackImm(d851.Imm)
						ptrWord, _ := d851.Imm.RawWords()
						d852 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d852.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d852.Reg2, uint64(len(d851.Imm.String())))
						ctx.BindReg(d852.Reg, &d852)
						ctx.BindReg(d852.Reg2, &d852)
					} else {
						d852 = d851
					}
					d853 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d852}, 1)
					ctx.EmitAndRegImm32(d853.Reg, 1)
					d853.Type = tagBool
					ctx.BindReg(d853.Reg, &d853)
					d854 = d853
					ctx.EnsureDesc(&d854)
					if d854.Loc != LocImm && d854.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d854.Loc == LocImm {
						if d854.Imm.Bool() {
							if ps.General {
							}
							ps855 := PhiState{General: ps.General}
							ps855.OverlayValues = make([]JITValueDesc, 855)
							ps855.OverlayValues[0] = d0
							ps855.OverlayValues[1] = d1
							ps855.OverlayValues[2] = d2
							ps855.OverlayValues[3] = d3
							ps855.OverlayValues[18] = d18
							ps855.OverlayValues[19] = d19
							ps855.OverlayValues[21] = d21
							ps855.OverlayValues[22] = d22
							ps855.OverlayValues[23] = d23
							ps855.OverlayValues[25] = d25
							ps855.OverlayValues[26] = d26
							ps855.OverlayValues[27] = d27
							ps855.OverlayValues[58] = d58
							ps855.OverlayValues[59] = d59
							ps855.OverlayValues[60] = d60
							ps855.OverlayValues[61] = d61
							ps855.OverlayValues[100] = d100
							ps855.OverlayValues[101] = d101
							ps855.OverlayValues[102] = d102
							ps855.OverlayValues[103] = d103
							ps855.OverlayValues[104] = d104
							ps855.OverlayValues[105] = d105
							ps855.OverlayValues[106] = d106
							ps855.OverlayValues[107] = d107
							ps855.OverlayValues[108] = d108
							ps855.OverlayValues[109] = d109
							ps855.OverlayValues[168] = d168
							ps855.OverlayValues[229] = d229
							ps855.OverlayValues[230] = d230
							ps855.OverlayValues[231] = d231
							ps855.OverlayValues[232] = d232
							ps855.OverlayValues[233] = d233
							ps855.OverlayValues[234] = d234
							ps855.OverlayValues[235] = d235
							ps855.OverlayValues[236] = d236
							ps855.OverlayValues[237] = d237
							ps855.OverlayValues[238] = d238
							ps855.OverlayValues[239] = d239
							ps855.OverlayValues[240] = d240
							ps855.OverlayValues[241] = d241
							ps855.OverlayValues[242] = d242
							ps855.OverlayValues[243] = d243
							ps855.OverlayValues[244] = d244
							ps855.OverlayValues[245] = d245
							ps855.OverlayValues[246] = d246
							ps855.OverlayValues[247] = d247
							ps855.OverlayValues[248] = d248
							ps855.OverlayValues[349] = d349
							ps855.OverlayValues[350] = d350
							ps855.OverlayValues[351] = d351
							ps855.OverlayValues[352] = d352
							ps855.OverlayValues[353] = d353
							ps855.OverlayValues[354] = d354
							ps855.OverlayValues[355] = d355
							ps855.OverlayValues[356] = d356
							ps855.OverlayValues[357] = d357
							ps855.OverlayValues[358] = d358
							ps855.OverlayValues[359] = d359
							ps855.OverlayValues[360] = d360
							ps855.OverlayValues[485] = d485
							ps855.OverlayValues[486] = d486
							ps855.OverlayValues[487] = d487
							ps855.OverlayValues[488] = d488
							ps855.OverlayValues[489] = d489
							ps855.OverlayValues[490] = d490
							ps855.OverlayValues[491] = d491
							ps855.OverlayValues[492] = d492
							ps855.OverlayValues[493] = d493
							ps855.OverlayValues[494] = d494
							ps855.OverlayValues[495] = d495
							ps855.OverlayValues[496] = d496
							ps855.OverlayValues[497] = d497
							ps855.OverlayValues[648] = d648
							ps855.OverlayValues[649] = d649
							ps855.OverlayValues[650] = d650
							ps855.OverlayValues[651] = d651
							ps855.OverlayValues[652] = d652
							ps855.OverlayValues[653] = d653
							ps855.OverlayValues[654] = d654
							ps855.OverlayValues[655] = d655
							ps855.OverlayValues[656] = d656
							ps855.OverlayValues[657] = d657
							ps855.OverlayValues[658] = d658
							ps855.OverlayValues[659] = d659
							ps855.OverlayValues[660] = d660
							ps855.OverlayValues[838] = d838
							ps855.OverlayValues[839] = d839
							ps855.OverlayValues[840] = d840
							ps855.OverlayValues[842] = d842
							ps855.OverlayValues[843] = d843
							ps855.OverlayValues[844] = d844
							ps855.OverlayValues[845] = d845
							ps855.OverlayValues[846] = d846
							ps855.OverlayValues[847] = d847
							ps855.OverlayValues[848] = d848
							ps855.OverlayValues[849] = d849
							ps855.OverlayValues[850] = d850
							ps855.OverlayValues[851] = d851
							ps855.OverlayValues[852] = d852
							ps855.OverlayValues[853] = d853
							ps855.OverlayValues[854] = d854
							return bbs[16].RenderPS(ps855)
						}
						if ps.General {
						}
						ps856 := PhiState{General: ps.General}
						ps856.OverlayValues = make([]JITValueDesc, 855)
						ps856.OverlayValues[0] = d0
						ps856.OverlayValues[1] = d1
						ps856.OverlayValues[2] = d2
						ps856.OverlayValues[3] = d3
						ps856.OverlayValues[18] = d18
						ps856.OverlayValues[19] = d19
						ps856.OverlayValues[21] = d21
						ps856.OverlayValues[22] = d22
						ps856.OverlayValues[23] = d23
						ps856.OverlayValues[25] = d25
						ps856.OverlayValues[26] = d26
						ps856.OverlayValues[27] = d27
						ps856.OverlayValues[58] = d58
						ps856.OverlayValues[59] = d59
						ps856.OverlayValues[60] = d60
						ps856.OverlayValues[61] = d61
						ps856.OverlayValues[100] = d100
						ps856.OverlayValues[101] = d101
						ps856.OverlayValues[102] = d102
						ps856.OverlayValues[103] = d103
						ps856.OverlayValues[104] = d104
						ps856.OverlayValues[105] = d105
						ps856.OverlayValues[106] = d106
						ps856.OverlayValues[107] = d107
						ps856.OverlayValues[108] = d108
						ps856.OverlayValues[109] = d109
						ps856.OverlayValues[168] = d168
						ps856.OverlayValues[229] = d229
						ps856.OverlayValues[230] = d230
						ps856.OverlayValues[231] = d231
						ps856.OverlayValues[232] = d232
						ps856.OverlayValues[233] = d233
						ps856.OverlayValues[234] = d234
						ps856.OverlayValues[235] = d235
						ps856.OverlayValues[236] = d236
						ps856.OverlayValues[237] = d237
						ps856.OverlayValues[238] = d238
						ps856.OverlayValues[239] = d239
						ps856.OverlayValues[240] = d240
						ps856.OverlayValues[241] = d241
						ps856.OverlayValues[242] = d242
						ps856.OverlayValues[243] = d243
						ps856.OverlayValues[244] = d244
						ps856.OverlayValues[245] = d245
						ps856.OverlayValues[246] = d246
						ps856.OverlayValues[247] = d247
						ps856.OverlayValues[248] = d248
						ps856.OverlayValues[349] = d349
						ps856.OverlayValues[350] = d350
						ps856.OverlayValues[351] = d351
						ps856.OverlayValues[352] = d352
						ps856.OverlayValues[353] = d353
						ps856.OverlayValues[354] = d354
						ps856.OverlayValues[355] = d355
						ps856.OverlayValues[356] = d356
						ps856.OverlayValues[357] = d357
						ps856.OverlayValues[358] = d358
						ps856.OverlayValues[359] = d359
						ps856.OverlayValues[360] = d360
						ps856.OverlayValues[485] = d485
						ps856.OverlayValues[486] = d486
						ps856.OverlayValues[487] = d487
						ps856.OverlayValues[488] = d488
						ps856.OverlayValues[489] = d489
						ps856.OverlayValues[490] = d490
						ps856.OverlayValues[491] = d491
						ps856.OverlayValues[492] = d492
						ps856.OverlayValues[493] = d493
						ps856.OverlayValues[494] = d494
						ps856.OverlayValues[495] = d495
						ps856.OverlayValues[496] = d496
						ps856.OverlayValues[497] = d497
						ps856.OverlayValues[648] = d648
						ps856.OverlayValues[649] = d649
						ps856.OverlayValues[650] = d650
						ps856.OverlayValues[651] = d651
						ps856.OverlayValues[652] = d652
						ps856.OverlayValues[653] = d653
						ps856.OverlayValues[654] = d654
						ps856.OverlayValues[655] = d655
						ps856.OverlayValues[656] = d656
						ps856.OverlayValues[657] = d657
						ps856.OverlayValues[658] = d658
						ps856.OverlayValues[659] = d659
						ps856.OverlayValues[660] = d660
						ps856.OverlayValues[838] = d838
						ps856.OverlayValues[839] = d839
						ps856.OverlayValues[840] = d840
						ps856.OverlayValues[842] = d842
						ps856.OverlayValues[843] = d843
						ps856.OverlayValues[844] = d844
						ps856.OverlayValues[845] = d845
						ps856.OverlayValues[846] = d846
						ps856.OverlayValues[847] = d847
						ps856.OverlayValues[848] = d848
						ps856.OverlayValues[849] = d849
						ps856.OverlayValues[850] = d850
						ps856.OverlayValues[851] = d851
						ps856.OverlayValues[852] = d852
						ps856.OverlayValues[853] = d853
						ps856.OverlayValues[854] = d854
						return bbs[19].RenderPS(ps856)
					}
					if !ps.General {
						ps.General = true
						return bbs[17].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d854.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					snap857 := d0
					snap858 := d1
					snap859 := d2
					snap860 := d3
					snap861 := d18
					snap862 := d19
					snap863 := d21
					snap864 := d22
					snap865 := d23
					snap866 := d25
					snap867 := d26
					snap868 := d27
					snap869 := d58
					snap870 := d59
					snap871 := d60
					snap872 := d61
					snap873 := d100
					snap874 := d101
					snap875 := d102
					snap876 := d103
					snap877 := d104
					snap878 := d105
					snap879 := d106
					snap880 := d107
					snap881 := d108
					snap882 := d109
					snap883 := d168
					snap884 := d229
					snap885 := d230
					snap886 := d231
					snap887 := d232
					snap888 := d233
					snap889 := d234
					snap890 := d235
					snap891 := d236
					snap892 := d237
					snap893 := d238
					snap894 := d239
					snap895 := d240
					snap896 := d241
					snap897 := d242
					snap898 := d243
					snap899 := d244
					snap900 := d245
					snap901 := d246
					snap902 := d247
					snap903 := d248
					snap904 := d349
					snap905 := d350
					snap906 := d351
					snap907 := d352
					snap908 := d353
					snap909 := d354
					snap910 := d355
					snap911 := d356
					snap912 := d357
					snap913 := d358
					snap914 := d359
					snap915 := d360
					snap916 := d485
					snap917 := d486
					snap918 := d487
					snap919 := d488
					snap920 := d489
					snap921 := d490
					snap922 := d491
					snap923 := d492
					snap924 := d493
					snap925 := d494
					snap926 := d495
					snap927 := d496
					snap928 := d497
					snap929 := d648
					snap930 := d649
					snap931 := d650
					snap932 := d651
					snap933 := d652
					snap934 := d653
					snap935 := d654
					snap936 := d655
					snap937 := d656
					snap938 := d657
					snap939 := d658
					snap940 := d659
					snap941 := d660
					snap942 := d838
					snap943 := d839
					snap944 := d840
					snap945 := d842
					snap946 := d843
					snap947 := d844
					snap948 := d845
					snap949 := d846
					snap950 := d847
					snap951 := d848
					snap952 := d849
					snap953 := d850
					snap954 := d851
					snap955 := d852
					snap956 := d853
					snap957 := d854
					alloc958 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc958)
					d0 = snap857
					d1 = snap858
					d2 = snap859
					d3 = snap860
					d18 = snap861
					d19 = snap862
					d21 = snap863
					d22 = snap864
					d23 = snap865
					d25 = snap866
					d26 = snap867
					d27 = snap868
					d58 = snap869
					d59 = snap870
					d60 = snap871
					d61 = snap872
					d100 = snap873
					d101 = snap874
					d102 = snap875
					d103 = snap876
					d104 = snap877
					d105 = snap878
					d106 = snap879
					d107 = snap880
					d108 = snap881
					d109 = snap882
					d168 = snap883
					d229 = snap884
					d230 = snap885
					d231 = snap886
					d232 = snap887
					d233 = snap888
					d234 = snap889
					d235 = snap890
					d236 = snap891
					d237 = snap892
					d238 = snap893
					d239 = snap894
					d240 = snap895
					d241 = snap896
					d242 = snap897
					d243 = snap898
					d244 = snap899
					d245 = snap900
					d246 = snap901
					d247 = snap902
					d248 = snap903
					d349 = snap904
					d350 = snap905
					d351 = snap906
					d352 = snap907
					d353 = snap908
					d354 = snap909
					d355 = snap910
					d356 = snap911
					d357 = snap912
					d358 = snap913
					d359 = snap914
					d360 = snap915
					d485 = snap916
					d486 = snap917
					d487 = snap918
					d488 = snap919
					d489 = snap920
					d490 = snap921
					d491 = snap922
					d492 = snap923
					d493 = snap924
					d494 = snap925
					d495 = snap926
					d496 = snap927
					d497 = snap928
					d648 = snap929
					d649 = snap930
					d650 = snap931
					d651 = snap932
					d652 = snap933
					d653 = snap934
					d654 = snap935
					d655 = snap936
					d656 = snap937
					d657 = snap938
					d658 = snap939
					d659 = snap940
					d660 = snap941
					d838 = snap942
					d839 = snap943
					d840 = snap944
					d842 = snap945
					d843 = snap946
					d844 = snap947
					d845 = snap948
					d846 = snap949
					d847 = snap950
					d848 = snap951
					d849 = snap952
					d850 = snap953
					d851 = snap954
					d852 = snap955
					d853 = snap956
					d854 = snap957
					ctx.RestoreAllocState(alloc958)
					d0 = snap857
					d1 = snap858
					d2 = snap859
					d3 = snap860
					d18 = snap861
					d19 = snap862
					d21 = snap863
					d22 = snap864
					d23 = snap865
					d25 = snap866
					d26 = snap867
					d27 = snap868
					d58 = snap869
					d59 = snap870
					d60 = snap871
					d61 = snap872
					d100 = snap873
					d101 = snap874
					d102 = snap875
					d103 = snap876
					d104 = snap877
					d105 = snap878
					d106 = snap879
					d107 = snap880
					d108 = snap881
					d109 = snap882
					d168 = snap883
					d229 = snap884
					d230 = snap885
					d231 = snap886
					d232 = snap887
					d233 = snap888
					d234 = snap889
					d235 = snap890
					d236 = snap891
					d237 = snap892
					d238 = snap893
					d239 = snap894
					d240 = snap895
					d241 = snap896
					d242 = snap897
					d243 = snap898
					d244 = snap899
					d245 = snap900
					d246 = snap901
					d247 = snap902
					d248 = snap903
					d349 = snap904
					d350 = snap905
					d351 = snap906
					d352 = snap907
					d353 = snap908
					d354 = snap909
					d355 = snap910
					d356 = snap911
					d357 = snap912
					d358 = snap913
					d359 = snap914
					d360 = snap915
					d485 = snap916
					d486 = snap917
					d487 = snap918
					d488 = snap919
					d489 = snap920
					d490 = snap921
					d491 = snap922
					d492 = snap923
					d493 = snap924
					d494 = snap925
					d495 = snap926
					d496 = snap927
					d497 = snap928
					d648 = snap929
					d649 = snap930
					d650 = snap931
					d651 = snap932
					d652 = snap933
					d653 = snap934
					d654 = snap935
					d655 = snap936
					d656 = snap937
					d657 = snap938
					d658 = snap939
					d659 = snap940
					d660 = snap941
					d838 = snap942
					d839 = snap943
					d840 = snap944
					d842 = snap945
					d843 = snap946
					d844 = snap947
					d845 = snap948
					d846 = snap949
					d847 = snap950
					d848 = snap951
					d849 = snap952
					d850 = snap953
					d851 = snap954
					d852 = snap955
					d853 = snap956
					d854 = snap957
					ps959 := PhiState{General: true}
					ps959.OverlayValues = make([]JITValueDesc, 855)
					ps959.OverlayValues[0] = d0
					ps959.OverlayValues[1] = d1
					ps959.OverlayValues[2] = d2
					ps959.OverlayValues[3] = d3
					ps959.OverlayValues[18] = d18
					ps959.OverlayValues[19] = d19
					ps959.OverlayValues[21] = d21
					ps959.OverlayValues[22] = d22
					ps959.OverlayValues[23] = d23
					ps959.OverlayValues[25] = d25
					ps959.OverlayValues[26] = d26
					ps959.OverlayValues[27] = d27
					ps959.OverlayValues[58] = d58
					ps959.OverlayValues[59] = d59
					ps959.OverlayValues[60] = d60
					ps959.OverlayValues[61] = d61
					ps959.OverlayValues[100] = d100
					ps959.OverlayValues[101] = d101
					ps959.OverlayValues[102] = d102
					ps959.OverlayValues[103] = d103
					ps959.OverlayValues[104] = d104
					ps959.OverlayValues[105] = d105
					ps959.OverlayValues[106] = d106
					ps959.OverlayValues[107] = d107
					ps959.OverlayValues[108] = d108
					ps959.OverlayValues[109] = d109
					ps959.OverlayValues[168] = d168
					ps959.OverlayValues[229] = d229
					ps959.OverlayValues[230] = d230
					ps959.OverlayValues[231] = d231
					ps959.OverlayValues[232] = d232
					ps959.OverlayValues[233] = d233
					ps959.OverlayValues[234] = d234
					ps959.OverlayValues[235] = d235
					ps959.OverlayValues[236] = d236
					ps959.OverlayValues[237] = d237
					ps959.OverlayValues[238] = d238
					ps959.OverlayValues[239] = d239
					ps959.OverlayValues[240] = d240
					ps959.OverlayValues[241] = d241
					ps959.OverlayValues[242] = d242
					ps959.OverlayValues[243] = d243
					ps959.OverlayValues[244] = d244
					ps959.OverlayValues[245] = d245
					ps959.OverlayValues[246] = d246
					ps959.OverlayValues[247] = d247
					ps959.OverlayValues[248] = d248
					ps959.OverlayValues[349] = d349
					ps959.OverlayValues[350] = d350
					ps959.OverlayValues[351] = d351
					ps959.OverlayValues[352] = d352
					ps959.OverlayValues[353] = d353
					ps959.OverlayValues[354] = d354
					ps959.OverlayValues[355] = d355
					ps959.OverlayValues[356] = d356
					ps959.OverlayValues[357] = d357
					ps959.OverlayValues[358] = d358
					ps959.OverlayValues[359] = d359
					ps959.OverlayValues[360] = d360
					ps959.OverlayValues[485] = d485
					ps959.OverlayValues[486] = d486
					ps959.OverlayValues[487] = d487
					ps959.OverlayValues[488] = d488
					ps959.OverlayValues[489] = d489
					ps959.OverlayValues[490] = d490
					ps959.OverlayValues[491] = d491
					ps959.OverlayValues[492] = d492
					ps959.OverlayValues[493] = d493
					ps959.OverlayValues[494] = d494
					ps959.OverlayValues[495] = d495
					ps959.OverlayValues[496] = d496
					ps959.OverlayValues[497] = d497
					ps959.OverlayValues[648] = d648
					ps959.OverlayValues[649] = d649
					ps959.OverlayValues[650] = d650
					ps959.OverlayValues[651] = d651
					ps959.OverlayValues[652] = d652
					ps959.OverlayValues[653] = d653
					ps959.OverlayValues[654] = d654
					ps959.OverlayValues[655] = d655
					ps959.OverlayValues[656] = d656
					ps959.OverlayValues[657] = d657
					ps959.OverlayValues[658] = d658
					ps959.OverlayValues[659] = d659
					ps959.OverlayValues[660] = d660
					ps959.OverlayValues[838] = d838
					ps959.OverlayValues[839] = d839
					ps959.OverlayValues[840] = d840
					ps959.OverlayValues[842] = d842
					ps959.OverlayValues[843] = d843
					ps959.OverlayValues[844] = d844
					ps959.OverlayValues[845] = d845
					ps959.OverlayValues[846] = d846
					ps959.OverlayValues[847] = d847
					ps959.OverlayValues[848] = d848
					ps959.OverlayValues[849] = d849
					ps959.OverlayValues[850] = d850
					ps959.OverlayValues[851] = d851
					ps959.OverlayValues[852] = d852
					ps959.OverlayValues[853] = d853
					ps959.OverlayValues[854] = d854
					ps960 := PhiState{General: true}
					ps960.OverlayValues = make([]JITValueDesc, 855)
					ps960.OverlayValues[0] = d0
					ps960.OverlayValues[1] = d1
					ps960.OverlayValues[2] = d2
					ps960.OverlayValues[3] = d3
					ps960.OverlayValues[18] = d18
					ps960.OverlayValues[19] = d19
					ps960.OverlayValues[21] = d21
					ps960.OverlayValues[22] = d22
					ps960.OverlayValues[23] = d23
					ps960.OverlayValues[25] = d25
					ps960.OverlayValues[26] = d26
					ps960.OverlayValues[27] = d27
					ps960.OverlayValues[58] = d58
					ps960.OverlayValues[59] = d59
					ps960.OverlayValues[60] = d60
					ps960.OverlayValues[61] = d61
					ps960.OverlayValues[100] = d100
					ps960.OverlayValues[101] = d101
					ps960.OverlayValues[102] = d102
					ps960.OverlayValues[103] = d103
					ps960.OverlayValues[104] = d104
					ps960.OverlayValues[105] = d105
					ps960.OverlayValues[106] = d106
					ps960.OverlayValues[107] = d107
					ps960.OverlayValues[108] = d108
					ps960.OverlayValues[109] = d109
					ps960.OverlayValues[168] = d168
					ps960.OverlayValues[229] = d229
					ps960.OverlayValues[230] = d230
					ps960.OverlayValues[231] = d231
					ps960.OverlayValues[232] = d232
					ps960.OverlayValues[233] = d233
					ps960.OverlayValues[234] = d234
					ps960.OverlayValues[235] = d235
					ps960.OverlayValues[236] = d236
					ps960.OverlayValues[237] = d237
					ps960.OverlayValues[238] = d238
					ps960.OverlayValues[239] = d239
					ps960.OverlayValues[240] = d240
					ps960.OverlayValues[241] = d241
					ps960.OverlayValues[242] = d242
					ps960.OverlayValues[243] = d243
					ps960.OverlayValues[244] = d244
					ps960.OverlayValues[245] = d245
					ps960.OverlayValues[246] = d246
					ps960.OverlayValues[247] = d247
					ps960.OverlayValues[248] = d248
					ps960.OverlayValues[349] = d349
					ps960.OverlayValues[350] = d350
					ps960.OverlayValues[351] = d351
					ps960.OverlayValues[352] = d352
					ps960.OverlayValues[353] = d353
					ps960.OverlayValues[354] = d354
					ps960.OverlayValues[355] = d355
					ps960.OverlayValues[356] = d356
					ps960.OverlayValues[357] = d357
					ps960.OverlayValues[358] = d358
					ps960.OverlayValues[359] = d359
					ps960.OverlayValues[360] = d360
					ps960.OverlayValues[485] = d485
					ps960.OverlayValues[486] = d486
					ps960.OverlayValues[487] = d487
					ps960.OverlayValues[488] = d488
					ps960.OverlayValues[489] = d489
					ps960.OverlayValues[490] = d490
					ps960.OverlayValues[491] = d491
					ps960.OverlayValues[492] = d492
					ps960.OverlayValues[493] = d493
					ps960.OverlayValues[494] = d494
					ps960.OverlayValues[495] = d495
					ps960.OverlayValues[496] = d496
					ps960.OverlayValues[497] = d497
					ps960.OverlayValues[648] = d648
					ps960.OverlayValues[649] = d649
					ps960.OverlayValues[650] = d650
					ps960.OverlayValues[651] = d651
					ps960.OverlayValues[652] = d652
					ps960.OverlayValues[653] = d653
					ps960.OverlayValues[654] = d654
					ps960.OverlayValues[655] = d655
					ps960.OverlayValues[656] = d656
					ps960.OverlayValues[657] = d657
					ps960.OverlayValues[658] = d658
					ps960.OverlayValues[659] = d659
					ps960.OverlayValues[660] = d660
					ps960.OverlayValues[838] = d838
					ps960.OverlayValues[839] = d839
					ps960.OverlayValues[840] = d840
					ps960.OverlayValues[842] = d842
					ps960.OverlayValues[843] = d843
					ps960.OverlayValues[844] = d844
					ps960.OverlayValues[845] = d845
					ps960.OverlayValues[846] = d846
					ps960.OverlayValues[847] = d847
					ps960.OverlayValues[848] = d848
					ps960.OverlayValues[849] = d849
					ps960.OverlayValues[850] = d850
					ps960.OverlayValues[851] = d851
					ps960.OverlayValues[852] = d852
					ps960.OverlayValues[853] = d853
					ps960.OverlayValues[854] = d854
					snap961 := d0
					snap962 := d1
					snap963 := d2
					snap964 := d3
					snap965 := d18
					snap966 := d19
					snap967 := d21
					snap968 := d22
					snap969 := d23
					snap970 := d25
					snap971 := d26
					snap972 := d27
					snap973 := d58
					snap974 := d59
					snap975 := d60
					snap976 := d61
					snap977 := d100
					snap978 := d101
					snap979 := d102
					snap980 := d103
					snap981 := d104
					snap982 := d105
					snap983 := d106
					snap984 := d107
					snap985 := d108
					snap986 := d109
					snap987 := d168
					snap988 := d229
					snap989 := d230
					snap990 := d231
					snap991 := d232
					snap992 := d233
					snap993 := d234
					snap994 := d235
					snap995 := d236
					snap996 := d237
					snap997 := d238
					snap998 := d239
					snap999 := d240
					snap1000 := d241
					snap1001 := d242
					snap1002 := d243
					snap1003 := d244
					snap1004 := d245
					snap1005 := d246
					snap1006 := d247
					snap1007 := d248
					snap1008 := d349
					snap1009 := d350
					snap1010 := d351
					snap1011 := d352
					snap1012 := d353
					snap1013 := d354
					snap1014 := d355
					snap1015 := d356
					snap1016 := d357
					snap1017 := d358
					snap1018 := d359
					snap1019 := d360
					snap1020 := d485
					snap1021 := d486
					snap1022 := d487
					snap1023 := d488
					snap1024 := d489
					snap1025 := d490
					snap1026 := d491
					snap1027 := d492
					snap1028 := d493
					snap1029 := d494
					snap1030 := d495
					snap1031 := d496
					snap1032 := d497
					snap1033 := d648
					snap1034 := d649
					snap1035 := d650
					snap1036 := d651
					snap1037 := d652
					snap1038 := d653
					snap1039 := d654
					snap1040 := d655
					snap1041 := d656
					snap1042 := d657
					snap1043 := d658
					snap1044 := d659
					snap1045 := d660
					snap1046 := d838
					snap1047 := d839
					snap1048 := d840
					snap1049 := d842
					snap1050 := d843
					snap1051 := d844
					snap1052 := d845
					snap1053 := d846
					snap1054 := d847
					snap1055 := d848
					snap1056 := d849
					snap1057 := d850
					snap1058 := d851
					snap1059 := d852
					snap1060 := d853
					snap1061 := d854
					alloc1062 := ctx.SnapshotAllocState()
					if !bbs[19].Rendered {
						bbs[19].RenderPS(ps960)
					}
					ctx.RestoreAllocState(alloc1062)
					d0 = snap961
					d1 = snap962
					d2 = snap963
					d3 = snap964
					d18 = snap965
					d19 = snap966
					d21 = snap967
					d22 = snap968
					d23 = snap969
					d25 = snap970
					d26 = snap971
					d27 = snap972
					d58 = snap973
					d59 = snap974
					d60 = snap975
					d61 = snap976
					d100 = snap977
					d101 = snap978
					d102 = snap979
					d103 = snap980
					d104 = snap981
					d105 = snap982
					d106 = snap983
					d107 = snap984
					d108 = snap985
					d109 = snap986
					d168 = snap987
					d229 = snap988
					d230 = snap989
					d231 = snap990
					d232 = snap991
					d233 = snap992
					d234 = snap993
					d235 = snap994
					d236 = snap995
					d237 = snap996
					d238 = snap997
					d239 = snap998
					d240 = snap999
					d241 = snap1000
					d242 = snap1001
					d243 = snap1002
					d244 = snap1003
					d245 = snap1004
					d246 = snap1005
					d247 = snap1006
					d248 = snap1007
					d349 = snap1008
					d350 = snap1009
					d351 = snap1010
					d352 = snap1011
					d353 = snap1012
					d354 = snap1013
					d355 = snap1014
					d356 = snap1015
					d357 = snap1016
					d358 = snap1017
					d359 = snap1018
					d360 = snap1019
					d485 = snap1020
					d486 = snap1021
					d487 = snap1022
					d488 = snap1023
					d489 = snap1024
					d490 = snap1025
					d491 = snap1026
					d492 = snap1027
					d493 = snap1028
					d494 = snap1029
					d495 = snap1030
					d496 = snap1031
					d497 = snap1032
					d648 = snap1033
					d649 = snap1034
					d650 = snap1035
					d651 = snap1036
					d652 = snap1037
					d653 = snap1038
					d654 = snap1039
					d655 = snap1040
					d656 = snap1041
					d657 = snap1042
					d658 = snap1043
					d659 = snap1044
					d660 = snap1045
					d838 = snap1046
					d839 = snap1047
					d840 = snap1048
					d842 = snap1049
					d843 = snap1050
					d844 = snap1051
					d845 = snap1052
					d846 = snap1053
					d847 = snap1054
					d848 = snap1055
					d849 = snap1056
					d850 = snap1057
					d851 = snap1058
					d852 = snap1059
					d853 = snap1060
					d854 = snap1061
					if !bbs[16].Rendered {
						return bbs[16].RenderPS(ps959)
					}
					return result
					ctx.FreeDesc(&d853)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != LocNone {
						d648 = ps.OverlayValues[648]
					}
					if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != LocNone {
						d649 = ps.OverlayValues[649]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != LocNone {
						d838 = ps.OverlayValues[838]
					}
					if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != LocNone {
						d839 = ps.OverlayValues[839]
					}
					if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != LocNone {
						d840 = ps.OverlayValues[840]
					}
					if len(ps.OverlayValues) > 842 && ps.OverlayValues[842].Loc != LocNone {
						d842 = ps.OverlayValues[842]
					}
					if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != LocNone {
						d843 = ps.OverlayValues[843]
					}
					if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != LocNone {
						d844 = ps.OverlayValues[844]
					}
					if len(ps.OverlayValues) > 845 && ps.OverlayValues[845].Loc != LocNone {
						d845 = ps.OverlayValues[845]
					}
					if len(ps.OverlayValues) > 846 && ps.OverlayValues[846].Loc != LocNone {
						d846 = ps.OverlayValues[846]
					}
					if len(ps.OverlayValues) > 847 && ps.OverlayValues[847].Loc != LocNone {
						d847 = ps.OverlayValues[847]
					}
					if len(ps.OverlayValues) > 848 && ps.OverlayValues[848].Loc != LocNone {
						d848 = ps.OverlayValues[848]
					}
					if len(ps.OverlayValues) > 849 && ps.OverlayValues[849].Loc != LocNone {
						d849 = ps.OverlayValues[849]
					}
					if len(ps.OverlayValues) > 850 && ps.OverlayValues[850].Loc != LocNone {
						d850 = ps.OverlayValues[850]
					}
					if len(ps.OverlayValues) > 851 && ps.OverlayValues[851].Loc != LocNone {
						d851 = ps.OverlayValues[851]
					}
					if len(ps.OverlayValues) > 852 && ps.OverlayValues[852].Loc != LocNone {
						d852 = ps.OverlayValues[852]
					}
					if len(ps.OverlayValues) > 853 && ps.OverlayValues[853].Loc != LocNone {
						d853 = ps.OverlayValues[853]
					}
					if len(ps.OverlayValues) > 854 && ps.OverlayValues[854].Loc != LocNone {
						d854 = ps.OverlayValues[854]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocRegTriple && d21.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d21)
					callResults1063 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d21}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d1064 = callResults1063[0]
					_ = d1064
					d1065 = callResults1063[1]
					_ = d1065
					d1066 = callResults1063[2]
					_ = d1066
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocRegTriple && d25.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((time.Time).Date arg0)")
					}
					ctx.SyncDesc(&d25)
					callResults1067 := JITEmitGoCallResults(ctx, GoFuncAddr((time.Time).Date), []JITValueDesc{d25}, []uint8{1, 1, 1}, []uint8{0, 0, 0})
					d1068 = callResults1067[0]
					_ = d1068
					d1069 = callResults1067[1]
					_ = d1069
					d1070 = callResults1067[2]
					_ = d1070
					ctx.EnsureDesc(&d1068)
					ctx.EnsureDesc(&d1064)
					ctx.EnsureDescsTogether(&d1068, &d1064)
					var d1071 JITValueDesc
					if d1068.Loc == LocImm && d1064.Loc == LocImm {
						d1071 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1068.Imm.Int() - d1064.Imm.Int())}
					} else if d1064.Loc == LocImm && d1064.Imm.Int() == 0 {
						r31 := ctx.AllocRegExcept(d1068.Reg)
						ctx.EmitMovRegReg(r31, d1068.Reg)
						d1071 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r31}
						ctx.BindReg(r31, &d1071)
					} else if d1068.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1064.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1068.Imm.Int()))
						ctx.EmitSubInt64(scratch, d1064.Reg)
						d1071 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1071)
					} else if d1064.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1068.Reg)
						ctx.EmitMovRegReg(scratch, d1068.Reg)
						if d1064.Imm.Int() >= -2147483648 && d1064.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d1064.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1064.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d1071 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d1071)
					} else {
						r32 := ctx.AllocRegExcept(d1068.Reg, d1064.Reg)
						ctx.EmitMovRegReg(r32, d1068.Reg)
						ctx.EmitSubInt64(r32, d1064.Reg)
						d1071 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r32}
						ctx.BindReg(r32, &d1071)
					}
					if d1071.Loc == LocReg && d1068.Loc == LocReg && d1071.Reg == d1068.Reg {
						ctx.TransferReg(d1068.Reg)
						d1068.Loc = LocNone
					}
					ctx.FreeDesc(&d1068)
					ctx.FreeDesc(&d1064)
					ctx.EnsureDesc(&d1071)
					ctx.EnsureDesc(&d1071)
					ctx.EnsureDesc(&d1071)
					if d1071.Loc == LocImm {
						ctx.EmitMakeInt(result, d1071)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1071)
						d1073 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d1073)
						if d1071.Loc == LocReg && d1071.Reg != result.Reg2 {
							ctx.FreeReg(d1071.Reg)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != LocNone {
						d648 = ps.OverlayValues[648]
					}
					if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != LocNone {
						d649 = ps.OverlayValues[649]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != LocNone {
						d838 = ps.OverlayValues[838]
					}
					if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != LocNone {
						d839 = ps.OverlayValues[839]
					}
					if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != LocNone {
						d840 = ps.OverlayValues[840]
					}
					if len(ps.OverlayValues) > 842 && ps.OverlayValues[842].Loc != LocNone {
						d842 = ps.OverlayValues[842]
					}
					if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != LocNone {
						d843 = ps.OverlayValues[843]
					}
					if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != LocNone {
						d844 = ps.OverlayValues[844]
					}
					if len(ps.OverlayValues) > 845 && ps.OverlayValues[845].Loc != LocNone {
						d845 = ps.OverlayValues[845]
					}
					if len(ps.OverlayValues) > 846 && ps.OverlayValues[846].Loc != LocNone {
						d846 = ps.OverlayValues[846]
					}
					if len(ps.OverlayValues) > 847 && ps.OverlayValues[847].Loc != LocNone {
						d847 = ps.OverlayValues[847]
					}
					if len(ps.OverlayValues) > 848 && ps.OverlayValues[848].Loc != LocNone {
						d848 = ps.OverlayValues[848]
					}
					if len(ps.OverlayValues) > 849 && ps.OverlayValues[849].Loc != LocNone {
						d849 = ps.OverlayValues[849]
					}
					if len(ps.OverlayValues) > 850 && ps.OverlayValues[850].Loc != LocNone {
						d850 = ps.OverlayValues[850]
					}
					if len(ps.OverlayValues) > 851 && ps.OverlayValues[851].Loc != LocNone {
						d851 = ps.OverlayValues[851]
					}
					if len(ps.OverlayValues) > 852 && ps.OverlayValues[852].Loc != LocNone {
						d852 = ps.OverlayValues[852]
					}
					if len(ps.OverlayValues) > 853 && ps.OverlayValues[853].Loc != LocNone {
						d853 = ps.OverlayValues[853]
					}
					if len(ps.OverlayValues) > 854 && ps.OverlayValues[854].Loc != LocNone {
						d854 = ps.OverlayValues[854]
					}
					if len(ps.OverlayValues) > 1064 && ps.OverlayValues[1064].Loc != LocNone {
						d1064 = ps.OverlayValues[1064]
					}
					if len(ps.OverlayValues) > 1065 && ps.OverlayValues[1065].Loc != LocNone {
						d1065 = ps.OverlayValues[1065]
					}
					if len(ps.OverlayValues) > 1066 && ps.OverlayValues[1066].Loc != LocNone {
						d1066 = ps.OverlayValues[1066]
					}
					if len(ps.OverlayValues) > 1068 && ps.OverlayValues[1068].Loc != LocNone {
						d1068 = ps.OverlayValues[1068]
					}
					if len(ps.OverlayValues) > 1069 && ps.OverlayValues[1069].Loc != LocNone {
						d1069 = ps.OverlayValues[1069]
					}
					if len(ps.OverlayValues) > 1070 && ps.OverlayValues[1070].Loc != LocNone {
						d1070 = ps.OverlayValues[1070]
					}
					if len(ps.OverlayValues) > 1071 && ps.OverlayValues[1071].Loc != LocNone {
						d1071 = ps.OverlayValues[1071]
					}
					if len(ps.OverlayValues) > 1072 && ps.OverlayValues[1072].Loc != LocNone {
						d1072 = ps.OverlayValues[1072]
					}
					if len(ps.OverlayValues) > 1073 && ps.OverlayValues[1073].Loc != LocNone {
						d1073 = ps.OverlayValues[1073]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d104)
					d1074 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("YEAR")}
					var d1075 JITValueDesc
					if d1074.Loc == LocImm {
						ctx.TrackImm(d1074.Imm)
						ptrWord, _ := d1074.Imm.RawWords()
						d1075 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d1075.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d1075.Reg2, uint64(len(d1074.Imm.String())))
						ctx.BindReg(d1075.Reg, &d1075)
						ctx.BindReg(d1075.Reg2, &d1075)
					} else {
						d1075 = d1074
					}
					d1076 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d104, d1075}, 1)
					ctx.EmitAndRegImm32(d1076.Reg, 1)
					d1076.Type = tagBool
					ctx.BindReg(d1076.Reg, &d1076)
					d1077 = d1076
					ctx.EnsureDesc(&d1077)
					if d1077.Loc != LocImm && d1077.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d1077.Loc == LocImm {
						if d1077.Imm.Bool() {
							if ps.General {
							}
							ps1078 := PhiState{General: ps.General}
							ps1078.OverlayValues = make([]JITValueDesc, 1078)
							ps1078.OverlayValues[0] = d0
							ps1078.OverlayValues[1] = d1
							ps1078.OverlayValues[2] = d2
							ps1078.OverlayValues[3] = d3
							ps1078.OverlayValues[18] = d18
							ps1078.OverlayValues[19] = d19
							ps1078.OverlayValues[21] = d21
							ps1078.OverlayValues[22] = d22
							ps1078.OverlayValues[23] = d23
							ps1078.OverlayValues[25] = d25
							ps1078.OverlayValues[26] = d26
							ps1078.OverlayValues[27] = d27
							ps1078.OverlayValues[58] = d58
							ps1078.OverlayValues[59] = d59
							ps1078.OverlayValues[60] = d60
							ps1078.OverlayValues[61] = d61
							ps1078.OverlayValues[100] = d100
							ps1078.OverlayValues[101] = d101
							ps1078.OverlayValues[102] = d102
							ps1078.OverlayValues[103] = d103
							ps1078.OverlayValues[104] = d104
							ps1078.OverlayValues[105] = d105
							ps1078.OverlayValues[106] = d106
							ps1078.OverlayValues[107] = d107
							ps1078.OverlayValues[108] = d108
							ps1078.OverlayValues[109] = d109
							ps1078.OverlayValues[168] = d168
							ps1078.OverlayValues[229] = d229
							ps1078.OverlayValues[230] = d230
							ps1078.OverlayValues[231] = d231
							ps1078.OverlayValues[232] = d232
							ps1078.OverlayValues[233] = d233
							ps1078.OverlayValues[234] = d234
							ps1078.OverlayValues[235] = d235
							ps1078.OverlayValues[236] = d236
							ps1078.OverlayValues[237] = d237
							ps1078.OverlayValues[238] = d238
							ps1078.OverlayValues[239] = d239
							ps1078.OverlayValues[240] = d240
							ps1078.OverlayValues[241] = d241
							ps1078.OverlayValues[242] = d242
							ps1078.OverlayValues[243] = d243
							ps1078.OverlayValues[244] = d244
							ps1078.OverlayValues[245] = d245
							ps1078.OverlayValues[246] = d246
							ps1078.OverlayValues[247] = d247
							ps1078.OverlayValues[248] = d248
							ps1078.OverlayValues[349] = d349
							ps1078.OverlayValues[350] = d350
							ps1078.OverlayValues[351] = d351
							ps1078.OverlayValues[352] = d352
							ps1078.OverlayValues[353] = d353
							ps1078.OverlayValues[354] = d354
							ps1078.OverlayValues[355] = d355
							ps1078.OverlayValues[356] = d356
							ps1078.OverlayValues[357] = d357
							ps1078.OverlayValues[358] = d358
							ps1078.OverlayValues[359] = d359
							ps1078.OverlayValues[360] = d360
							ps1078.OverlayValues[485] = d485
							ps1078.OverlayValues[486] = d486
							ps1078.OverlayValues[487] = d487
							ps1078.OverlayValues[488] = d488
							ps1078.OverlayValues[489] = d489
							ps1078.OverlayValues[490] = d490
							ps1078.OverlayValues[491] = d491
							ps1078.OverlayValues[492] = d492
							ps1078.OverlayValues[493] = d493
							ps1078.OverlayValues[494] = d494
							ps1078.OverlayValues[495] = d495
							ps1078.OverlayValues[496] = d496
							ps1078.OverlayValues[497] = d497
							ps1078.OverlayValues[648] = d648
							ps1078.OverlayValues[649] = d649
							ps1078.OverlayValues[650] = d650
							ps1078.OverlayValues[651] = d651
							ps1078.OverlayValues[652] = d652
							ps1078.OverlayValues[653] = d653
							ps1078.OverlayValues[654] = d654
							ps1078.OverlayValues[655] = d655
							ps1078.OverlayValues[656] = d656
							ps1078.OverlayValues[657] = d657
							ps1078.OverlayValues[658] = d658
							ps1078.OverlayValues[659] = d659
							ps1078.OverlayValues[660] = d660
							ps1078.OverlayValues[838] = d838
							ps1078.OverlayValues[839] = d839
							ps1078.OverlayValues[840] = d840
							ps1078.OverlayValues[842] = d842
							ps1078.OverlayValues[843] = d843
							ps1078.OverlayValues[844] = d844
							ps1078.OverlayValues[845] = d845
							ps1078.OverlayValues[846] = d846
							ps1078.OverlayValues[847] = d847
							ps1078.OverlayValues[848] = d848
							ps1078.OverlayValues[849] = d849
							ps1078.OverlayValues[850] = d850
							ps1078.OverlayValues[851] = d851
							ps1078.OverlayValues[852] = d852
							ps1078.OverlayValues[853] = d853
							ps1078.OverlayValues[854] = d854
							ps1078.OverlayValues[1064] = d1064
							ps1078.OverlayValues[1065] = d1065
							ps1078.OverlayValues[1066] = d1066
							ps1078.OverlayValues[1068] = d1068
							ps1078.OverlayValues[1069] = d1069
							ps1078.OverlayValues[1070] = d1070
							ps1078.OverlayValues[1071] = d1071
							ps1078.OverlayValues[1072] = d1072
							ps1078.OverlayValues[1073] = d1073
							ps1078.OverlayValues[1074] = d1074
							ps1078.OverlayValues[1075] = d1075
							ps1078.OverlayValues[1076] = d1076
							ps1078.OverlayValues[1077] = d1077
							return bbs[18].RenderPS(ps1078)
						}
						if ps.General {
						}
						ps1079 := PhiState{General: ps.General}
						ps1079.OverlayValues = make([]JITValueDesc, 1078)
						ps1079.OverlayValues[0] = d0
						ps1079.OverlayValues[1] = d1
						ps1079.OverlayValues[2] = d2
						ps1079.OverlayValues[3] = d3
						ps1079.OverlayValues[18] = d18
						ps1079.OverlayValues[19] = d19
						ps1079.OverlayValues[21] = d21
						ps1079.OverlayValues[22] = d22
						ps1079.OverlayValues[23] = d23
						ps1079.OverlayValues[25] = d25
						ps1079.OverlayValues[26] = d26
						ps1079.OverlayValues[27] = d27
						ps1079.OverlayValues[58] = d58
						ps1079.OverlayValues[59] = d59
						ps1079.OverlayValues[60] = d60
						ps1079.OverlayValues[61] = d61
						ps1079.OverlayValues[100] = d100
						ps1079.OverlayValues[101] = d101
						ps1079.OverlayValues[102] = d102
						ps1079.OverlayValues[103] = d103
						ps1079.OverlayValues[104] = d104
						ps1079.OverlayValues[105] = d105
						ps1079.OverlayValues[106] = d106
						ps1079.OverlayValues[107] = d107
						ps1079.OverlayValues[108] = d108
						ps1079.OverlayValues[109] = d109
						ps1079.OverlayValues[168] = d168
						ps1079.OverlayValues[229] = d229
						ps1079.OverlayValues[230] = d230
						ps1079.OverlayValues[231] = d231
						ps1079.OverlayValues[232] = d232
						ps1079.OverlayValues[233] = d233
						ps1079.OverlayValues[234] = d234
						ps1079.OverlayValues[235] = d235
						ps1079.OverlayValues[236] = d236
						ps1079.OverlayValues[237] = d237
						ps1079.OverlayValues[238] = d238
						ps1079.OverlayValues[239] = d239
						ps1079.OverlayValues[240] = d240
						ps1079.OverlayValues[241] = d241
						ps1079.OverlayValues[242] = d242
						ps1079.OverlayValues[243] = d243
						ps1079.OverlayValues[244] = d244
						ps1079.OverlayValues[245] = d245
						ps1079.OverlayValues[246] = d246
						ps1079.OverlayValues[247] = d247
						ps1079.OverlayValues[248] = d248
						ps1079.OverlayValues[349] = d349
						ps1079.OverlayValues[350] = d350
						ps1079.OverlayValues[351] = d351
						ps1079.OverlayValues[352] = d352
						ps1079.OverlayValues[353] = d353
						ps1079.OverlayValues[354] = d354
						ps1079.OverlayValues[355] = d355
						ps1079.OverlayValues[356] = d356
						ps1079.OverlayValues[357] = d357
						ps1079.OverlayValues[358] = d358
						ps1079.OverlayValues[359] = d359
						ps1079.OverlayValues[360] = d360
						ps1079.OverlayValues[485] = d485
						ps1079.OverlayValues[486] = d486
						ps1079.OverlayValues[487] = d487
						ps1079.OverlayValues[488] = d488
						ps1079.OverlayValues[489] = d489
						ps1079.OverlayValues[490] = d490
						ps1079.OverlayValues[491] = d491
						ps1079.OverlayValues[492] = d492
						ps1079.OverlayValues[493] = d493
						ps1079.OverlayValues[494] = d494
						ps1079.OverlayValues[495] = d495
						ps1079.OverlayValues[496] = d496
						ps1079.OverlayValues[497] = d497
						ps1079.OverlayValues[648] = d648
						ps1079.OverlayValues[649] = d649
						ps1079.OverlayValues[650] = d650
						ps1079.OverlayValues[651] = d651
						ps1079.OverlayValues[652] = d652
						ps1079.OverlayValues[653] = d653
						ps1079.OverlayValues[654] = d654
						ps1079.OverlayValues[655] = d655
						ps1079.OverlayValues[656] = d656
						ps1079.OverlayValues[657] = d657
						ps1079.OverlayValues[658] = d658
						ps1079.OverlayValues[659] = d659
						ps1079.OverlayValues[660] = d660
						ps1079.OverlayValues[838] = d838
						ps1079.OverlayValues[839] = d839
						ps1079.OverlayValues[840] = d840
						ps1079.OverlayValues[842] = d842
						ps1079.OverlayValues[843] = d843
						ps1079.OverlayValues[844] = d844
						ps1079.OverlayValues[845] = d845
						ps1079.OverlayValues[846] = d846
						ps1079.OverlayValues[847] = d847
						ps1079.OverlayValues[848] = d848
						ps1079.OverlayValues[849] = d849
						ps1079.OverlayValues[850] = d850
						ps1079.OverlayValues[851] = d851
						ps1079.OverlayValues[852] = d852
						ps1079.OverlayValues[853] = d853
						ps1079.OverlayValues[854] = d854
						ps1079.OverlayValues[1064] = d1064
						ps1079.OverlayValues[1065] = d1065
						ps1079.OverlayValues[1066] = d1066
						ps1079.OverlayValues[1068] = d1068
						ps1079.OverlayValues[1069] = d1069
						ps1079.OverlayValues[1070] = d1070
						ps1079.OverlayValues[1071] = d1071
						ps1079.OverlayValues[1072] = d1072
						ps1079.OverlayValues[1073] = d1073
						ps1079.OverlayValues[1074] = d1074
						ps1079.OverlayValues[1075] = d1075
						ps1079.OverlayValues[1076] = d1076
						ps1079.OverlayValues[1077] = d1077
						return bbs[20].RenderPS(ps1079)
					}
					if !ps.General {
						ps.General = true
						return bbs[19].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d1077.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					snap1080 := d0
					snap1081 := d1
					snap1082 := d2
					snap1083 := d3
					snap1084 := d18
					snap1085 := d19
					snap1086 := d21
					snap1087 := d22
					snap1088 := d23
					snap1089 := d25
					snap1090 := d26
					snap1091 := d27
					snap1092 := d58
					snap1093 := d59
					snap1094 := d60
					snap1095 := d61
					snap1096 := d100
					snap1097 := d101
					snap1098 := d102
					snap1099 := d103
					snap1100 := d104
					snap1101 := d105
					snap1102 := d106
					snap1103 := d107
					snap1104 := d108
					snap1105 := d109
					snap1106 := d168
					snap1107 := d229
					snap1108 := d230
					snap1109 := d231
					snap1110 := d232
					snap1111 := d233
					snap1112 := d234
					snap1113 := d235
					snap1114 := d236
					snap1115 := d237
					snap1116 := d238
					snap1117 := d239
					snap1118 := d240
					snap1119 := d241
					snap1120 := d242
					snap1121 := d243
					snap1122 := d244
					snap1123 := d245
					snap1124 := d246
					snap1125 := d247
					snap1126 := d248
					snap1127 := d349
					snap1128 := d350
					snap1129 := d351
					snap1130 := d352
					snap1131 := d353
					snap1132 := d354
					snap1133 := d355
					snap1134 := d356
					snap1135 := d357
					snap1136 := d358
					snap1137 := d359
					snap1138 := d360
					snap1139 := d485
					snap1140 := d486
					snap1141 := d487
					snap1142 := d488
					snap1143 := d489
					snap1144 := d490
					snap1145 := d491
					snap1146 := d492
					snap1147 := d493
					snap1148 := d494
					snap1149 := d495
					snap1150 := d496
					snap1151 := d497
					snap1152 := d648
					snap1153 := d649
					snap1154 := d650
					snap1155 := d651
					snap1156 := d652
					snap1157 := d653
					snap1158 := d654
					snap1159 := d655
					snap1160 := d656
					snap1161 := d657
					snap1162 := d658
					snap1163 := d659
					snap1164 := d660
					snap1165 := d838
					snap1166 := d839
					snap1167 := d840
					snap1168 := d842
					snap1169 := d843
					snap1170 := d844
					snap1171 := d845
					snap1172 := d846
					snap1173 := d847
					snap1174 := d848
					snap1175 := d849
					snap1176 := d850
					snap1177 := d851
					snap1178 := d852
					snap1179 := d853
					snap1180 := d854
					snap1181 := d1064
					snap1182 := d1065
					snap1183 := d1066
					snap1184 := d1068
					snap1185 := d1069
					snap1186 := d1070
					snap1187 := d1071
					snap1188 := d1072
					snap1189 := d1073
					snap1190 := d1074
					snap1191 := d1075
					snap1192 := d1076
					snap1193 := d1077
					alloc1194 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc1194)
					d0 = snap1080
					d1 = snap1081
					d2 = snap1082
					d3 = snap1083
					d18 = snap1084
					d19 = snap1085
					d21 = snap1086
					d22 = snap1087
					d23 = snap1088
					d25 = snap1089
					d26 = snap1090
					d27 = snap1091
					d58 = snap1092
					d59 = snap1093
					d60 = snap1094
					d61 = snap1095
					d100 = snap1096
					d101 = snap1097
					d102 = snap1098
					d103 = snap1099
					d104 = snap1100
					d105 = snap1101
					d106 = snap1102
					d107 = snap1103
					d108 = snap1104
					d109 = snap1105
					d168 = snap1106
					d229 = snap1107
					d230 = snap1108
					d231 = snap1109
					d232 = snap1110
					d233 = snap1111
					d234 = snap1112
					d235 = snap1113
					d236 = snap1114
					d237 = snap1115
					d238 = snap1116
					d239 = snap1117
					d240 = snap1118
					d241 = snap1119
					d242 = snap1120
					d243 = snap1121
					d244 = snap1122
					d245 = snap1123
					d246 = snap1124
					d247 = snap1125
					d248 = snap1126
					d349 = snap1127
					d350 = snap1128
					d351 = snap1129
					d352 = snap1130
					d353 = snap1131
					d354 = snap1132
					d355 = snap1133
					d356 = snap1134
					d357 = snap1135
					d358 = snap1136
					d359 = snap1137
					d360 = snap1138
					d485 = snap1139
					d486 = snap1140
					d487 = snap1141
					d488 = snap1142
					d489 = snap1143
					d490 = snap1144
					d491 = snap1145
					d492 = snap1146
					d493 = snap1147
					d494 = snap1148
					d495 = snap1149
					d496 = snap1150
					d497 = snap1151
					d648 = snap1152
					d649 = snap1153
					d650 = snap1154
					d651 = snap1155
					d652 = snap1156
					d653 = snap1157
					d654 = snap1158
					d655 = snap1159
					d656 = snap1160
					d657 = snap1161
					d658 = snap1162
					d659 = snap1163
					d660 = snap1164
					d838 = snap1165
					d839 = snap1166
					d840 = snap1167
					d842 = snap1168
					d843 = snap1169
					d844 = snap1170
					d845 = snap1171
					d846 = snap1172
					d847 = snap1173
					d848 = snap1174
					d849 = snap1175
					d850 = snap1176
					d851 = snap1177
					d852 = snap1178
					d853 = snap1179
					d854 = snap1180
					d1064 = snap1181
					d1065 = snap1182
					d1066 = snap1183
					d1068 = snap1184
					d1069 = snap1185
					d1070 = snap1186
					d1071 = snap1187
					d1072 = snap1188
					d1073 = snap1189
					d1074 = snap1190
					d1075 = snap1191
					d1076 = snap1192
					d1077 = snap1193
					ctx.RestoreAllocState(alloc1194)
					d0 = snap1080
					d1 = snap1081
					d2 = snap1082
					d3 = snap1083
					d18 = snap1084
					d19 = snap1085
					d21 = snap1086
					d22 = snap1087
					d23 = snap1088
					d25 = snap1089
					d26 = snap1090
					d27 = snap1091
					d58 = snap1092
					d59 = snap1093
					d60 = snap1094
					d61 = snap1095
					d100 = snap1096
					d101 = snap1097
					d102 = snap1098
					d103 = snap1099
					d104 = snap1100
					d105 = snap1101
					d106 = snap1102
					d107 = snap1103
					d108 = snap1104
					d109 = snap1105
					d168 = snap1106
					d229 = snap1107
					d230 = snap1108
					d231 = snap1109
					d232 = snap1110
					d233 = snap1111
					d234 = snap1112
					d235 = snap1113
					d236 = snap1114
					d237 = snap1115
					d238 = snap1116
					d239 = snap1117
					d240 = snap1118
					d241 = snap1119
					d242 = snap1120
					d243 = snap1121
					d244 = snap1122
					d245 = snap1123
					d246 = snap1124
					d247 = snap1125
					d248 = snap1126
					d349 = snap1127
					d350 = snap1128
					d351 = snap1129
					d352 = snap1130
					d353 = snap1131
					d354 = snap1132
					d355 = snap1133
					d356 = snap1134
					d357 = snap1135
					d358 = snap1136
					d359 = snap1137
					d360 = snap1138
					d485 = snap1139
					d486 = snap1140
					d487 = snap1141
					d488 = snap1142
					d489 = snap1143
					d490 = snap1144
					d491 = snap1145
					d492 = snap1146
					d493 = snap1147
					d494 = snap1148
					d495 = snap1149
					d496 = snap1150
					d497 = snap1151
					d648 = snap1152
					d649 = snap1153
					d650 = snap1154
					d651 = snap1155
					d652 = snap1156
					d653 = snap1157
					d654 = snap1158
					d655 = snap1159
					d656 = snap1160
					d657 = snap1161
					d658 = snap1162
					d659 = snap1163
					d660 = snap1164
					d838 = snap1165
					d839 = snap1166
					d840 = snap1167
					d842 = snap1168
					d843 = snap1169
					d844 = snap1170
					d845 = snap1171
					d846 = snap1172
					d847 = snap1173
					d848 = snap1174
					d849 = snap1175
					d850 = snap1176
					d851 = snap1177
					d852 = snap1178
					d853 = snap1179
					d854 = snap1180
					d1064 = snap1181
					d1065 = snap1182
					d1066 = snap1183
					d1068 = snap1184
					d1069 = snap1185
					d1070 = snap1186
					d1071 = snap1187
					d1072 = snap1188
					d1073 = snap1189
					d1074 = snap1190
					d1075 = snap1191
					d1076 = snap1192
					d1077 = snap1193
					ps1195 := PhiState{General: true}
					ps1195.OverlayValues = make([]JITValueDesc, 1078)
					ps1195.OverlayValues[0] = d0
					ps1195.OverlayValues[1] = d1
					ps1195.OverlayValues[2] = d2
					ps1195.OverlayValues[3] = d3
					ps1195.OverlayValues[18] = d18
					ps1195.OverlayValues[19] = d19
					ps1195.OverlayValues[21] = d21
					ps1195.OverlayValues[22] = d22
					ps1195.OverlayValues[23] = d23
					ps1195.OverlayValues[25] = d25
					ps1195.OverlayValues[26] = d26
					ps1195.OverlayValues[27] = d27
					ps1195.OverlayValues[58] = d58
					ps1195.OverlayValues[59] = d59
					ps1195.OverlayValues[60] = d60
					ps1195.OverlayValues[61] = d61
					ps1195.OverlayValues[100] = d100
					ps1195.OverlayValues[101] = d101
					ps1195.OverlayValues[102] = d102
					ps1195.OverlayValues[103] = d103
					ps1195.OverlayValues[104] = d104
					ps1195.OverlayValues[105] = d105
					ps1195.OverlayValues[106] = d106
					ps1195.OverlayValues[107] = d107
					ps1195.OverlayValues[108] = d108
					ps1195.OverlayValues[109] = d109
					ps1195.OverlayValues[168] = d168
					ps1195.OverlayValues[229] = d229
					ps1195.OverlayValues[230] = d230
					ps1195.OverlayValues[231] = d231
					ps1195.OverlayValues[232] = d232
					ps1195.OverlayValues[233] = d233
					ps1195.OverlayValues[234] = d234
					ps1195.OverlayValues[235] = d235
					ps1195.OverlayValues[236] = d236
					ps1195.OverlayValues[237] = d237
					ps1195.OverlayValues[238] = d238
					ps1195.OverlayValues[239] = d239
					ps1195.OverlayValues[240] = d240
					ps1195.OverlayValues[241] = d241
					ps1195.OverlayValues[242] = d242
					ps1195.OverlayValues[243] = d243
					ps1195.OverlayValues[244] = d244
					ps1195.OverlayValues[245] = d245
					ps1195.OverlayValues[246] = d246
					ps1195.OverlayValues[247] = d247
					ps1195.OverlayValues[248] = d248
					ps1195.OverlayValues[349] = d349
					ps1195.OverlayValues[350] = d350
					ps1195.OverlayValues[351] = d351
					ps1195.OverlayValues[352] = d352
					ps1195.OverlayValues[353] = d353
					ps1195.OverlayValues[354] = d354
					ps1195.OverlayValues[355] = d355
					ps1195.OverlayValues[356] = d356
					ps1195.OverlayValues[357] = d357
					ps1195.OverlayValues[358] = d358
					ps1195.OverlayValues[359] = d359
					ps1195.OverlayValues[360] = d360
					ps1195.OverlayValues[485] = d485
					ps1195.OverlayValues[486] = d486
					ps1195.OverlayValues[487] = d487
					ps1195.OverlayValues[488] = d488
					ps1195.OverlayValues[489] = d489
					ps1195.OverlayValues[490] = d490
					ps1195.OverlayValues[491] = d491
					ps1195.OverlayValues[492] = d492
					ps1195.OverlayValues[493] = d493
					ps1195.OverlayValues[494] = d494
					ps1195.OverlayValues[495] = d495
					ps1195.OverlayValues[496] = d496
					ps1195.OverlayValues[497] = d497
					ps1195.OverlayValues[648] = d648
					ps1195.OverlayValues[649] = d649
					ps1195.OverlayValues[650] = d650
					ps1195.OverlayValues[651] = d651
					ps1195.OverlayValues[652] = d652
					ps1195.OverlayValues[653] = d653
					ps1195.OverlayValues[654] = d654
					ps1195.OverlayValues[655] = d655
					ps1195.OverlayValues[656] = d656
					ps1195.OverlayValues[657] = d657
					ps1195.OverlayValues[658] = d658
					ps1195.OverlayValues[659] = d659
					ps1195.OverlayValues[660] = d660
					ps1195.OverlayValues[838] = d838
					ps1195.OverlayValues[839] = d839
					ps1195.OverlayValues[840] = d840
					ps1195.OverlayValues[842] = d842
					ps1195.OverlayValues[843] = d843
					ps1195.OverlayValues[844] = d844
					ps1195.OverlayValues[845] = d845
					ps1195.OverlayValues[846] = d846
					ps1195.OverlayValues[847] = d847
					ps1195.OverlayValues[848] = d848
					ps1195.OverlayValues[849] = d849
					ps1195.OverlayValues[850] = d850
					ps1195.OverlayValues[851] = d851
					ps1195.OverlayValues[852] = d852
					ps1195.OverlayValues[853] = d853
					ps1195.OverlayValues[854] = d854
					ps1195.OverlayValues[1064] = d1064
					ps1195.OverlayValues[1065] = d1065
					ps1195.OverlayValues[1066] = d1066
					ps1195.OverlayValues[1068] = d1068
					ps1195.OverlayValues[1069] = d1069
					ps1195.OverlayValues[1070] = d1070
					ps1195.OverlayValues[1071] = d1071
					ps1195.OverlayValues[1072] = d1072
					ps1195.OverlayValues[1073] = d1073
					ps1195.OverlayValues[1074] = d1074
					ps1195.OverlayValues[1075] = d1075
					ps1195.OverlayValues[1076] = d1076
					ps1195.OverlayValues[1077] = d1077
					ps1196 := PhiState{General: true}
					ps1196.OverlayValues = make([]JITValueDesc, 1078)
					ps1196.OverlayValues[0] = d0
					ps1196.OverlayValues[1] = d1
					ps1196.OverlayValues[2] = d2
					ps1196.OverlayValues[3] = d3
					ps1196.OverlayValues[18] = d18
					ps1196.OverlayValues[19] = d19
					ps1196.OverlayValues[21] = d21
					ps1196.OverlayValues[22] = d22
					ps1196.OverlayValues[23] = d23
					ps1196.OverlayValues[25] = d25
					ps1196.OverlayValues[26] = d26
					ps1196.OverlayValues[27] = d27
					ps1196.OverlayValues[58] = d58
					ps1196.OverlayValues[59] = d59
					ps1196.OverlayValues[60] = d60
					ps1196.OverlayValues[61] = d61
					ps1196.OverlayValues[100] = d100
					ps1196.OverlayValues[101] = d101
					ps1196.OverlayValues[102] = d102
					ps1196.OverlayValues[103] = d103
					ps1196.OverlayValues[104] = d104
					ps1196.OverlayValues[105] = d105
					ps1196.OverlayValues[106] = d106
					ps1196.OverlayValues[107] = d107
					ps1196.OverlayValues[108] = d108
					ps1196.OverlayValues[109] = d109
					ps1196.OverlayValues[168] = d168
					ps1196.OverlayValues[229] = d229
					ps1196.OverlayValues[230] = d230
					ps1196.OverlayValues[231] = d231
					ps1196.OverlayValues[232] = d232
					ps1196.OverlayValues[233] = d233
					ps1196.OverlayValues[234] = d234
					ps1196.OverlayValues[235] = d235
					ps1196.OverlayValues[236] = d236
					ps1196.OverlayValues[237] = d237
					ps1196.OverlayValues[238] = d238
					ps1196.OverlayValues[239] = d239
					ps1196.OverlayValues[240] = d240
					ps1196.OverlayValues[241] = d241
					ps1196.OverlayValues[242] = d242
					ps1196.OverlayValues[243] = d243
					ps1196.OverlayValues[244] = d244
					ps1196.OverlayValues[245] = d245
					ps1196.OverlayValues[246] = d246
					ps1196.OverlayValues[247] = d247
					ps1196.OverlayValues[248] = d248
					ps1196.OverlayValues[349] = d349
					ps1196.OverlayValues[350] = d350
					ps1196.OverlayValues[351] = d351
					ps1196.OverlayValues[352] = d352
					ps1196.OverlayValues[353] = d353
					ps1196.OverlayValues[354] = d354
					ps1196.OverlayValues[355] = d355
					ps1196.OverlayValues[356] = d356
					ps1196.OverlayValues[357] = d357
					ps1196.OverlayValues[358] = d358
					ps1196.OverlayValues[359] = d359
					ps1196.OverlayValues[360] = d360
					ps1196.OverlayValues[485] = d485
					ps1196.OverlayValues[486] = d486
					ps1196.OverlayValues[487] = d487
					ps1196.OverlayValues[488] = d488
					ps1196.OverlayValues[489] = d489
					ps1196.OverlayValues[490] = d490
					ps1196.OverlayValues[491] = d491
					ps1196.OverlayValues[492] = d492
					ps1196.OverlayValues[493] = d493
					ps1196.OverlayValues[494] = d494
					ps1196.OverlayValues[495] = d495
					ps1196.OverlayValues[496] = d496
					ps1196.OverlayValues[497] = d497
					ps1196.OverlayValues[648] = d648
					ps1196.OverlayValues[649] = d649
					ps1196.OverlayValues[650] = d650
					ps1196.OverlayValues[651] = d651
					ps1196.OverlayValues[652] = d652
					ps1196.OverlayValues[653] = d653
					ps1196.OverlayValues[654] = d654
					ps1196.OverlayValues[655] = d655
					ps1196.OverlayValues[656] = d656
					ps1196.OverlayValues[657] = d657
					ps1196.OverlayValues[658] = d658
					ps1196.OverlayValues[659] = d659
					ps1196.OverlayValues[660] = d660
					ps1196.OverlayValues[838] = d838
					ps1196.OverlayValues[839] = d839
					ps1196.OverlayValues[840] = d840
					ps1196.OverlayValues[842] = d842
					ps1196.OverlayValues[843] = d843
					ps1196.OverlayValues[844] = d844
					ps1196.OverlayValues[845] = d845
					ps1196.OverlayValues[846] = d846
					ps1196.OverlayValues[847] = d847
					ps1196.OverlayValues[848] = d848
					ps1196.OverlayValues[849] = d849
					ps1196.OverlayValues[850] = d850
					ps1196.OverlayValues[851] = d851
					ps1196.OverlayValues[852] = d852
					ps1196.OverlayValues[853] = d853
					ps1196.OverlayValues[854] = d854
					ps1196.OverlayValues[1064] = d1064
					ps1196.OverlayValues[1065] = d1065
					ps1196.OverlayValues[1066] = d1066
					ps1196.OverlayValues[1068] = d1068
					ps1196.OverlayValues[1069] = d1069
					ps1196.OverlayValues[1070] = d1070
					ps1196.OverlayValues[1071] = d1071
					ps1196.OverlayValues[1072] = d1072
					ps1196.OverlayValues[1073] = d1073
					ps1196.OverlayValues[1074] = d1074
					ps1196.OverlayValues[1075] = d1075
					ps1196.OverlayValues[1076] = d1076
					ps1196.OverlayValues[1077] = d1077
					snap1197 := d0
					snap1198 := d1
					snap1199 := d2
					snap1200 := d3
					snap1201 := d18
					snap1202 := d19
					snap1203 := d21
					snap1204 := d22
					snap1205 := d23
					snap1206 := d25
					snap1207 := d26
					snap1208 := d27
					snap1209 := d58
					snap1210 := d59
					snap1211 := d60
					snap1212 := d61
					snap1213 := d100
					snap1214 := d101
					snap1215 := d102
					snap1216 := d103
					snap1217 := d104
					snap1218 := d105
					snap1219 := d106
					snap1220 := d107
					snap1221 := d108
					snap1222 := d109
					snap1223 := d168
					snap1224 := d229
					snap1225 := d230
					snap1226 := d231
					snap1227 := d232
					snap1228 := d233
					snap1229 := d234
					snap1230 := d235
					snap1231 := d236
					snap1232 := d237
					snap1233 := d238
					snap1234 := d239
					snap1235 := d240
					snap1236 := d241
					snap1237 := d242
					snap1238 := d243
					snap1239 := d244
					snap1240 := d245
					snap1241 := d246
					snap1242 := d247
					snap1243 := d248
					snap1244 := d349
					snap1245 := d350
					snap1246 := d351
					snap1247 := d352
					snap1248 := d353
					snap1249 := d354
					snap1250 := d355
					snap1251 := d356
					snap1252 := d357
					snap1253 := d358
					snap1254 := d359
					snap1255 := d360
					snap1256 := d485
					snap1257 := d486
					snap1258 := d487
					snap1259 := d488
					snap1260 := d489
					snap1261 := d490
					snap1262 := d491
					snap1263 := d492
					snap1264 := d493
					snap1265 := d494
					snap1266 := d495
					snap1267 := d496
					snap1268 := d497
					snap1269 := d648
					snap1270 := d649
					snap1271 := d650
					snap1272 := d651
					snap1273 := d652
					snap1274 := d653
					snap1275 := d654
					snap1276 := d655
					snap1277 := d656
					snap1278 := d657
					snap1279 := d658
					snap1280 := d659
					snap1281 := d660
					snap1282 := d838
					snap1283 := d839
					snap1284 := d840
					snap1285 := d842
					snap1286 := d843
					snap1287 := d844
					snap1288 := d845
					snap1289 := d846
					snap1290 := d847
					snap1291 := d848
					snap1292 := d849
					snap1293 := d850
					snap1294 := d851
					snap1295 := d852
					snap1296 := d853
					snap1297 := d854
					snap1298 := d1064
					snap1299 := d1065
					snap1300 := d1066
					snap1301 := d1068
					snap1302 := d1069
					snap1303 := d1070
					snap1304 := d1071
					snap1305 := d1072
					snap1306 := d1073
					snap1307 := d1074
					snap1308 := d1075
					snap1309 := d1076
					snap1310 := d1077
					alloc1311 := ctx.SnapshotAllocState()
					if !bbs[20].Rendered {
						bbs[20].RenderPS(ps1196)
					}
					ctx.RestoreAllocState(alloc1311)
					d0 = snap1197
					d1 = snap1198
					d2 = snap1199
					d3 = snap1200
					d18 = snap1201
					d19 = snap1202
					d21 = snap1203
					d22 = snap1204
					d23 = snap1205
					d25 = snap1206
					d26 = snap1207
					d27 = snap1208
					d58 = snap1209
					d59 = snap1210
					d60 = snap1211
					d61 = snap1212
					d100 = snap1213
					d101 = snap1214
					d102 = snap1215
					d103 = snap1216
					d104 = snap1217
					d105 = snap1218
					d106 = snap1219
					d107 = snap1220
					d108 = snap1221
					d109 = snap1222
					d168 = snap1223
					d229 = snap1224
					d230 = snap1225
					d231 = snap1226
					d232 = snap1227
					d233 = snap1228
					d234 = snap1229
					d235 = snap1230
					d236 = snap1231
					d237 = snap1232
					d238 = snap1233
					d239 = snap1234
					d240 = snap1235
					d241 = snap1236
					d242 = snap1237
					d243 = snap1238
					d244 = snap1239
					d245 = snap1240
					d246 = snap1241
					d247 = snap1242
					d248 = snap1243
					d349 = snap1244
					d350 = snap1245
					d351 = snap1246
					d352 = snap1247
					d353 = snap1248
					d354 = snap1249
					d355 = snap1250
					d356 = snap1251
					d357 = snap1252
					d358 = snap1253
					d359 = snap1254
					d360 = snap1255
					d485 = snap1256
					d486 = snap1257
					d487 = snap1258
					d488 = snap1259
					d489 = snap1260
					d490 = snap1261
					d491 = snap1262
					d492 = snap1263
					d493 = snap1264
					d494 = snap1265
					d495 = snap1266
					d496 = snap1267
					d497 = snap1268
					d648 = snap1269
					d649 = snap1270
					d650 = snap1271
					d651 = snap1272
					d652 = snap1273
					d653 = snap1274
					d654 = snap1275
					d655 = snap1276
					d656 = snap1277
					d657 = snap1278
					d658 = snap1279
					d659 = snap1280
					d660 = snap1281
					d838 = snap1282
					d839 = snap1283
					d840 = snap1284
					d842 = snap1285
					d843 = snap1286
					d844 = snap1287
					d845 = snap1288
					d846 = snap1289
					d847 = snap1290
					d848 = snap1291
					d849 = snap1292
					d850 = snap1293
					d851 = snap1294
					d852 = snap1295
					d853 = snap1296
					d854 = snap1297
					d1064 = snap1298
					d1065 = snap1299
					d1066 = snap1300
					d1068 = snap1301
					d1069 = snap1302
					d1070 = snap1303
					d1071 = snap1304
					d1072 = snap1305
					d1073 = snap1306
					d1074 = snap1307
					d1075 = snap1308
					d1076 = snap1309
					d1077 = snap1310
					if !bbs[18].Rendered {
						return bbs[18].RenderPS(ps1195)
					}
					return result
					ctx.FreeDesc(&d1076)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
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
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != LocNone {
						d230 = ps.OverlayValues[230]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != LocNone {
						d232 = ps.OverlayValues[232]
					}
					if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != LocNone {
						d233 = ps.OverlayValues[233]
					}
					if len(ps.OverlayValues) > 234 && ps.OverlayValues[234].Loc != LocNone {
						d234 = ps.OverlayValues[234]
					}
					if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != LocNone {
						d235 = ps.OverlayValues[235]
					}
					if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != LocNone {
						d236 = ps.OverlayValues[236]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != LocNone {
						d244 = ps.OverlayValues[244]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
						d352 = ps.OverlayValues[352]
					}
					if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
						d353 = ps.OverlayValues[353]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
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
					if len(ps.OverlayValues) > 648 && ps.OverlayValues[648].Loc != LocNone {
						d648 = ps.OverlayValues[648]
					}
					if len(ps.OverlayValues) > 649 && ps.OverlayValues[649].Loc != LocNone {
						d649 = ps.OverlayValues[649]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 838 && ps.OverlayValues[838].Loc != LocNone {
						d838 = ps.OverlayValues[838]
					}
					if len(ps.OverlayValues) > 839 && ps.OverlayValues[839].Loc != LocNone {
						d839 = ps.OverlayValues[839]
					}
					if len(ps.OverlayValues) > 840 && ps.OverlayValues[840].Loc != LocNone {
						d840 = ps.OverlayValues[840]
					}
					if len(ps.OverlayValues) > 842 && ps.OverlayValues[842].Loc != LocNone {
						d842 = ps.OverlayValues[842]
					}
					if len(ps.OverlayValues) > 843 && ps.OverlayValues[843].Loc != LocNone {
						d843 = ps.OverlayValues[843]
					}
					if len(ps.OverlayValues) > 844 && ps.OverlayValues[844].Loc != LocNone {
						d844 = ps.OverlayValues[844]
					}
					if len(ps.OverlayValues) > 845 && ps.OverlayValues[845].Loc != LocNone {
						d845 = ps.OverlayValues[845]
					}
					if len(ps.OverlayValues) > 846 && ps.OverlayValues[846].Loc != LocNone {
						d846 = ps.OverlayValues[846]
					}
					if len(ps.OverlayValues) > 847 && ps.OverlayValues[847].Loc != LocNone {
						d847 = ps.OverlayValues[847]
					}
					if len(ps.OverlayValues) > 848 && ps.OverlayValues[848].Loc != LocNone {
						d848 = ps.OverlayValues[848]
					}
					if len(ps.OverlayValues) > 849 && ps.OverlayValues[849].Loc != LocNone {
						d849 = ps.OverlayValues[849]
					}
					if len(ps.OverlayValues) > 850 && ps.OverlayValues[850].Loc != LocNone {
						d850 = ps.OverlayValues[850]
					}
					if len(ps.OverlayValues) > 851 && ps.OverlayValues[851].Loc != LocNone {
						d851 = ps.OverlayValues[851]
					}
					if len(ps.OverlayValues) > 852 && ps.OverlayValues[852].Loc != LocNone {
						d852 = ps.OverlayValues[852]
					}
					if len(ps.OverlayValues) > 853 && ps.OverlayValues[853].Loc != LocNone {
						d853 = ps.OverlayValues[853]
					}
					if len(ps.OverlayValues) > 854 && ps.OverlayValues[854].Loc != LocNone {
						d854 = ps.OverlayValues[854]
					}
					if len(ps.OverlayValues) > 1064 && ps.OverlayValues[1064].Loc != LocNone {
						d1064 = ps.OverlayValues[1064]
					}
					if len(ps.OverlayValues) > 1065 && ps.OverlayValues[1065].Loc != LocNone {
						d1065 = ps.OverlayValues[1065]
					}
					if len(ps.OverlayValues) > 1066 && ps.OverlayValues[1066].Loc != LocNone {
						d1066 = ps.OverlayValues[1066]
					}
					if len(ps.OverlayValues) > 1068 && ps.OverlayValues[1068].Loc != LocNone {
						d1068 = ps.OverlayValues[1068]
					}
					if len(ps.OverlayValues) > 1069 && ps.OverlayValues[1069].Loc != LocNone {
						d1069 = ps.OverlayValues[1069]
					}
					if len(ps.OverlayValues) > 1070 && ps.OverlayValues[1070].Loc != LocNone {
						d1070 = ps.OverlayValues[1070]
					}
					if len(ps.OverlayValues) > 1071 && ps.OverlayValues[1071].Loc != LocNone {
						d1071 = ps.OverlayValues[1071]
					}
					if len(ps.OverlayValues) > 1072 && ps.OverlayValues[1072].Loc != LocNone {
						d1072 = ps.OverlayValues[1072]
					}
					if len(ps.OverlayValues) > 1073 && ps.OverlayValues[1073].Loc != LocNone {
						d1073 = ps.OverlayValues[1073]
					}
					if len(ps.OverlayValues) > 1074 && ps.OverlayValues[1074].Loc != LocNone {
						d1074 = ps.OverlayValues[1074]
					}
					if len(ps.OverlayValues) > 1075 && ps.OverlayValues[1075].Loc != LocNone {
						d1075 = ps.OverlayValues[1075]
					}
					if len(ps.OverlayValues) > 1076 && ps.OverlayValues[1076].Loc != LocNone {
						d1076 = ps.OverlayValues[1076]
					}
					if len(ps.OverlayValues) > 1077 && ps.OverlayValues[1077].Loc != LocNone {
						d1077 = ps.OverlayValues[1077]
					}
					ctx.ReclaimUntrackedRegs()
					d1312 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d1312)
					if d1312.Loc == LocRegPair || d1312.Loc == LocStackPair || d1312.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d1312, &result)
						result.Type = d1312.Type
					} else {
						switch d1312.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d1312)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d1312)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d1312)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d1312, &result)
							result.Type = d1312.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps1313 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps1313)
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
