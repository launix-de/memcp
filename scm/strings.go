/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "io"
import "fmt"
import "html"
import "regexp"
import "strings"
import "net/url"
import "encoding/json"
import "encoding/base64"
import "encoding/hex"
import crand "crypto/rand"
import "golang.org/x/text/collate"
import "golang.org/x/text/language"
import "sync"
import "reflect"

// Collation metadata registry for stable serialization of comparator closures.
// Keyed by function pointer.
var collateRegistry sync.Map // map[uintptr]struct{Collation string; Reverse bool}

// (no additional globals needed)

// LookupCollate returns (collation, reverse, ok) for a previously built collate closure.
func LookupCollate(fn func(...Scmer) Scmer) (string, bool, bool) {
	if fn == nil {
		return "", false, false
	}
	if v, ok := collateRegistry.Load(reflect.ValueOf(fn).Pointer()); ok {
		m := v.(struct {
			Collation string
			Reverse   bool
		})
		return m.Collation, m.Reverse, true
	}
	return "", false, false
}

/* SQL LIKE operator implementation on strings */
func StrLike(str, pattern string) bool {
	for {
		// boundary check
		if len(pattern) == 0 {
			if len(str) == 0 {
				// we finished matching
				return true
			} else {
				// pattern is consumed but no string left: no match
				return false
			}
		}
		// now str[0] and pattern[0] are assured to exist
		if pattern[0] == '%' { // wildcard
			pattern = pattern[1:]
			if pattern == "" {
				return true // string ends with wildcard
			}
			// otherwise: match against all possible endings
			for i := len(str) - 1; i >= 0; i-- { // run from right to left to be as greedy and performant as possible
				if str[i] == pattern[0] {
					// check if this caracter matches the rest
					if StrLike(str[i:], pattern) {
						return true // we found a match with this position as continuation
					}
				}
			}
			return false // no continuation found
		} else {
			if len(str) > 0 && (pattern[0] == '_' || pattern[0] == str[0]) {
				// match -> move one character forward
				pattern = pattern[1:]
				str = str[1:]
			} else {
				// mismatch -> we're out
				return false
			}
		}
	}
}

func TransformFromJSON(a_ any) Scmer {
	switch a := a_.(type) {
	case map[string]any:
		// decode binary strings encoded by MarshalJSON
		if b64, ok := a["bytes"]; ok && len(a) == 1 {
			if s, ok := b64.(string); ok {
				if raw, err := base64.StdEncoding.DecodeString(s); err == nil {
					return NewString(string(raw))
				}
			}
		}
		result := make([]Scmer, 0, len(a)*2)
		for k, v := range a {
			result = append(result, NewString(k), TransformFromJSON(v))
		}
		return NewSlice(result)
	case []any:
		result := make([]Scmer, len(a))
		for i, v := range a {
			result[i] = TransformFromJSON(v)
		}
		return NewSlice(result)
	default:
		return FromAny(a_)
	}
}

func init_strings() {
	// string functions
	DeclareTitle("Strings")

	Declare(&Globalenv, &Declaration{
		"string?", "tells if the value is a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].IsString())
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			d2.ID = 0
			d1 = ctx.EmitTagEqualsBorrowed(&d2, tagString, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.ResolveFixups()
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
			}
			argPinned3 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned3 = append(argPinned3, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned3 = append(argPinned3, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned3 = append(argPinned3, ai.Reg2)
					}
				}
			}
			ps4 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps4)
			for _, r := range argPinned3 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"concat", "concatenates stringable values and returns a string",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values to concat", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			var sb strings.Builder
			for _, s := range a {
				if stream, ok := s.Any().(io.Reader); ok {
					_, _ = io.Copy(&sb, stream)
				} else {
					sb.WriteString(String(s))
				}
			}
			return NewString(sb.String())
		}, true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t82[0:int] (x=t82 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t82[0:int] (x=t82 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t78[0:int] (x=t78 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t78[0:int] (x=t78 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t78[0:int] (x=t78 marker="_alloc" isDesc=false goVar=) */ /* TODO: ChangeType: changetype Symbol <- string (t25) */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
	})
	Declare(&Globalenv, &Declaration{
		"substr", "returns a substring (0-based index)",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to cut", nil},
			DeclarationParameter{"start", "number", "first character index (0-based)", nil},
			DeclarationParameter{"len", "number", "optional length", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			s := String(a[0])
			i := ToInt(a[1])
			if len(a) > 2 {
				return NewString(s[i : i+ToInt(a[2])])
			}
			return NewString(s[i:])
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
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
			var d6 JITValueDesc
			_ = d6
			var d7 JITValueDesc
			_ = d7
			var d21 JITValueDesc
			_ = d21
			var d22 JITValueDesc
			_ = d22
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [3]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			d3 = args[1]
			d3.ID = 0
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d3.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d3 = tmpPair
			} else if d3.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
				switch d3.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d3)
				d3 = tmpPair
			}
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d3}, 1)
			ctx.FreeDesc(&d3)
			d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d5)
			var d6 JITValueDesc
			if d5.Loc == LocImm {
				d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() > 2)}
			} else {
				r0 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d5.Reg, 2)
				ctx.EmitSetcc(r0, CcG)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d6)
			}
			ctx.FreeDesc(&d5)
			d7 = d6
			ctx.EnsureDesc(&d7)
			if d7.Loc != LocImm && d7.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
			ps8 := PhiState{General: ps.General}
			ps8.OverlayValues = make([]JITValueDesc, 8)
			ps8.OverlayValues[0] = d0
			ps8.OverlayValues[1] = d1
			ps8.OverlayValues[2] = d2
			ps8.OverlayValues[3] = d3
			ps8.OverlayValues[4] = d4
			ps8.OverlayValues[5] = d5
			ps8.OverlayValues[6] = d6
			ps8.OverlayValues[7] = d7
					return bbs[1].RenderPS(ps8)
				}
			ps9 := PhiState{General: ps.General}
			ps9.OverlayValues = make([]JITValueDesc, 8)
			ps9.OverlayValues[0] = d0
			ps9.OverlayValues[1] = d1
			ps9.OverlayValues[2] = d2
			ps9.OverlayValues[3] = d3
			ps9.OverlayValues[4] = d4
			ps9.OverlayValues[5] = d5
			ps9.OverlayValues[6] = d6
			ps9.OverlayValues[7] = d7
				return bbs[2].RenderPS(ps9)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl4 := ctx.ReserveLabel()
			lbl5 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d7.Reg, 0)
			ctx.EmitJcc(CcNE, lbl4)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl4)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl5)
			ctx.EmitJmp(lbl3)
			ps10 := PhiState{General: true}
			ps10.OverlayValues = make([]JITValueDesc, 8)
			ps10.OverlayValues[0] = d0
			ps10.OverlayValues[1] = d1
			ps10.OverlayValues[2] = d2
			ps10.OverlayValues[3] = d3
			ps10.OverlayValues[4] = d4
			ps10.OverlayValues[5] = d5
			ps10.OverlayValues[6] = d6
			ps10.OverlayValues[7] = d7
			ps11 := PhiState{General: true}
			ps11.OverlayValues = make([]JITValueDesc, 8)
			ps11.OverlayValues[0] = d0
			ps11.OverlayValues[1] = d1
			ps11.OverlayValues[2] = d2
			ps11.OverlayValues[3] = d3
			ps11.OverlayValues[4] = d4
			ps11.OverlayValues[5] = d5
			ps11.OverlayValues[6] = d6
			ps11.OverlayValues[7] = d7
			snap12 := d0
			snap13 := d1
			snap14 := d2
			snap15 := d3
			snap16 := d4
			snap17 := d5
			snap18 := d6
			snap19 := d7
			alloc20 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps11)
			}
			ctx.RestoreAllocState(alloc20)
			d0 = snap12
			d1 = snap13
			d2 = snap14
			d3 = snap15
			d4 = snap16
			d5 = snap17
			d6 = snap18
			d7 = snap19
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps10)
			}
			return result
			ctx.FreeDesc(&d6)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			ctx.ReclaimUntrackedRegs()
			d21 = args[2]
			d21.ID = 0
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d21.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d21.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d21)
				} else if d21.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d21)
				} else if d21.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d21)
				} else if d21.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d21.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d21 = tmpPair
			} else if d21.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d21.Type, Reg: ctx.AllocRegExcept(d21.Reg), Reg2: ctx.AllocRegExcept(d21.Reg)}
				switch d21.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d21)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d21)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d21)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d21)
				d21 = tmpPair
			}
			if d21.Loc != LocRegPair && d21.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d22 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d21}, 1)
			ctx.FreeDesc(&d21)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d22)
			var d23 JITValueDesc
			if d4.Loc == LocImm && d22.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d22.Imm.Int())}
			} else if d22.Loc == LocImm && d22.Imm.Int() == 0 {
				r1 := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(r1, d4.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
				ctx.BindReg(r1, &d23)
			} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.EmitAddInt64(scratch, d22.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(scratch, d4.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r2 := ctx.AllocRegExcept(d4.Reg, d22.Reg)
				ctx.EmitMovRegReg(r2, d4.Reg)
				ctx.EmitAddInt64(r2, d22.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
				ctx.BindReg(r2, &d23)
			}
			if d23.Loc == LocReg && d4.Loc == LocReg && d23.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d23)
			var d25 JITValueDesc
			if d23.Loc == LocImm && d4.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() - d4.Imm.Int())}
			} else {
				r3 := ctx.AllocReg()
				if d23.Loc == LocImm {
					ctx.EmitMovRegImm64(r3, uint64(d23.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r3, d23.Reg)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitSubInt64(r3, RegR11)
				} else {
					ctx.EmitSubInt64(r3, d4.Reg)
				}
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
				ctx.BindReg(r3, &d25)
			}
			var d26 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d4.Imm.Int())}
			} else {
				r4 := ctx.AllocReg()
				if d1.Loc == LocImm {
					ctx.EmitMovRegImm64(r4, uint64(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r4, d1.Reg)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitAddInt64(r4, RegR11)
				} else {
					ctx.EmitAddInt64(r4, d4.Reg)
				}
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
				ctx.BindReg(r4, &d26)
			}
			var d27 JITValueDesc
			r5 := ctx.AllocReg()
			r6 := ctx.AllocReg()
			if d26.Loc == LocImm {
				ctx.EmitMovRegImm64(r5, uint64(d26.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r5, d26.Reg)
				ctx.FreeReg(d26.Reg)
			}
			if d25.Loc == LocImm {
				ctx.EmitMovRegImm64(r6, uint64(d25.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r6, d25.Reg)
				ctx.FreeReg(d25.Reg)
			}
			d27 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
			ctx.BindReg(r5, &d27)
			ctx.BindReg(r6, &d27)
			ctx.FreeDesc(&d23)
			d28 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d27}, 2)
			ctx.EmitMovPairToResult(&d28, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			var d29 JITValueDesc
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair {
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
				ctx.BindReg(d1.Reg2, &d29)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d29)
			var d31 JITValueDesc
			if d29.Loc == LocImm && d4.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int() - d4.Imm.Int())}
			} else {
				r7 := ctx.AllocReg()
				if d29.Loc == LocImm {
					ctx.EmitMovRegImm64(r7, uint64(d29.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r7, d29.Reg)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitSubInt64(r7, RegR11)
				} else {
					ctx.EmitSubInt64(r7, d4.Reg)
				}
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
				ctx.BindReg(r7, &d31)
			}
			var d32 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d4.Imm.Int())}
			} else {
				r8 := ctx.AllocReg()
				if d1.Loc == LocImm {
					ctx.EmitMovRegImm64(r8, uint64(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r8, d1.Reg)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitAddInt64(r8, RegR11)
				} else {
					ctx.EmitAddInt64(r8, d4.Reg)
				}
				d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d32)
			}
			var d33 JITValueDesc
			r9 := ctx.AllocReg()
			r10 := ctx.AllocReg()
			if d32.Loc == LocImm {
				ctx.EmitMovRegImm64(r9, uint64(d32.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r9, d32.Reg)
				ctx.FreeReg(d32.Reg)
			}
			if d31.Loc == LocImm {
				ctx.EmitMovRegImm64(r10, uint64(d31.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r10, d31.Reg)
				ctx.FreeReg(d31.Reg)
			}
			d33 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
			ctx.BindReg(r9, &d33)
			ctx.BindReg(r10, &d33)
			ctx.FreeDesc(&d4)
			d34 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d33}, 2)
			ctx.EmitMovPairToResult(&d34, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned35 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned35 = append(argPinned35, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned35 = append(argPinned35, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned35 = append(argPinned35, ai.Reg2)
					}
				}
			}
			ps36 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps36)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned35 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"sql_substr", "SQL SUBSTR/SUBSTRING with 1-based index and bounds checking",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to cut", nil},
			DeclarationParameter{"start", "number", "first character position (1-based)", nil},
			DeclarationParameter{"len", "number", "optional length", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			s := String(a[0])
			slen := len(s)
			start := ToInt(a[1]) - 1 // convert 1-based to 0-based
			if start < 0 {
				start = 0
			}
			if start >= slen {
				return NewString("")
			}
			if len(a) > 2 {
				n := ToInt(a[2])
				if start+n > slen {
					n = slen - start
				}
				if n < 0 {
					return NewString("")
				}
				return NewString(s[start : start+n])
			}
			return NewString(s[start:])
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			var d5 JITValueDesc
			_ = d5
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
			var d29 JITValueDesc
			_ = d29
			var d30 JITValueDesc
			_ = d30
			var d33 JITValueDesc
			_ = d33
			var d55 JITValueDesc
			_ = d55
			var d56 JITValueDesc
			_ = d56
			var d57 JITValueDesc
			_ = d57
			var d58 JITValueDesc
			_ = d58
			var d61 JITValueDesc
			_ = d61
			var d89 JITValueDesc
			_ = d89
			var d90 JITValueDesc
			_ = d90
			var d91 JITValueDesc
			_ = d91
			var d92 JITValueDesc
			_ = d92
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
			var d132 JITValueDesc
			_ = d132
			var d134 JITValueDesc
			_ = d134
			var d135 JITValueDesc
			_ = d135
			var d138 JITValueDesc
			_ = d138
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
			var d186 JITValueDesc
			_ = d186
			var d187 JITValueDesc
			_ = d187
			var d188 JITValueDesc
			_ = d188
			var d189 JITValueDesc
			_ = d189
			var d192 JITValueDesc
			_ = d192
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
			var d252 JITValueDesc
			_ = d252
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			var bbs [13]BBDescriptor
			bbs[4].PhiBase = int32(0)
			bbs[4].PhiCount = uint16(1)
			bbs[10].PhiBase = int32(16)
			bbs[10].PhiCount = uint16(1)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.ReserveLabel()
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl9 := ctx.ReserveLabel()
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			lbl10 := ctx.ReserveLabel()
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.ReserveLabel()
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			lbl13 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			ps6 := PhiState{General: ps.General}
			ps6.OverlayValues = make([]JITValueDesc, 6)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
			ps6.OverlayValues[5] = d5
					return bbs[1].RenderPS(ps6)
				}
			ps7 := PhiState{General: ps.General}
			ps7.OverlayValues = make([]JITValueDesc, 6)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps7.OverlayValues[5] = d5
				return bbs[2].RenderPS(ps7)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl14 := ctx.ReserveLabel()
			lbl15 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d5.Reg, 0)
			ctx.EmitJcc(CcNE, lbl14)
			ctx.EmitJmp(lbl15)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl3)
			ps8 := PhiState{General: true}
			ps8.OverlayValues = make([]JITValueDesc, 6)
			ps8.OverlayValues[0] = d0
			ps8.OverlayValues[1] = d1
			ps8.OverlayValues[2] = d2
			ps8.OverlayValues[3] = d3
			ps8.OverlayValues[4] = d4
			ps8.OverlayValues[5] = d5
			ps9 := PhiState{General: true}
			ps9.OverlayValues = make([]JITValueDesc, 6)
			ps9.OverlayValues[0] = d0
			ps9.OverlayValues[1] = d1
			ps9.OverlayValues[2] = d2
			ps9.OverlayValues[3] = d3
			ps9.OverlayValues[4] = d4
			ps9.OverlayValues[5] = d5
			snap10 := d0
			snap11 := d1
			snap12 := d2
			snap13 := d3
			snap14 := d4
			snap15 := d5
			alloc16 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps9)
			}
			ctx.RestoreAllocState(alloc16)
			d0 = snap10
			d1 = snap11
			d2 = snap12
			d3 = snap13
			d4 = snap14
			d5 = snap15
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps8)
			}
			return result
			ctx.FreeDesc(&d3)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			d17 = args[0]
			d17.ID = 0
			d19 = d17
			ctx.EnsureDesc(&d19)
			if d19.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d19.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d19)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d19)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d19)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d19.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d19 = tmpPair
			} else if d19.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d19.Reg), Reg2: ctx.AllocRegExcept(d19.Reg)}
				switch d19.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d19)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d19)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d19)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d19)
				d19 = tmpPair
			} else if d19.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d19.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d19.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d19 = tmpPair
			}
			if d19.Loc != LocRegPair && d19.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d18 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d19}, 2)
			ctx.FreeDesc(&d17)
			var d20 JITValueDesc
			if d18.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d18.Imm.String())))}
			} else {
				ctx.EnsureDesc(&d18)
				if d18.Loc == LocRegPair {
					d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d18.Reg2}
					ctx.BindReg(d18.Reg2, &d20)
					ctx.BindReg(d18.Reg2, &d20)
				} else if d18.Loc == LocReg {
					d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d18.Reg}
					ctx.BindReg(d18.Reg, &d20)
					ctx.BindReg(d18.Reg, &d20)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			d21 = args[1]
			d21.ID = 0
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d21.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d21.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d21)
				} else if d21.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d21)
				} else if d21.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d21)
				} else if d21.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d21.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d21 = tmpPair
			} else if d21.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d21.Type, Reg: ctx.AllocRegExcept(d21.Reg), Reg2: ctx.AllocRegExcept(d21.Reg)}
				switch d21.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d21)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d21)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d21)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d21)
				d21 = tmpPair
			}
			if d21.Loc != LocRegPair && d21.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d22 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d21}, 1)
			ctx.FreeDesc(&d21)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			var d23 JITValueDesc
			if d22.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d22.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.EmitMovRegReg(scratch, d22.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			}
			if d23.Loc == LocReg && d22.Loc == LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.EnsureDesc(&d23)
			var d24 JITValueDesc
			if d23.Loc == LocImm {
				d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d23.Reg)
				ctx.EmitCmpRegImm32(d23.Reg, 0)
				ctx.EmitSetcc(r1, CcL)
				d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d24)
			}
			d25 = d24
			ctx.EnsureDesc(&d25)
			if d25.Loc != LocImm && d25.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d25.Loc == LocImm {
				if d25.Imm.Bool() {
			ps26 := PhiState{General: ps.General}
			ps26.OverlayValues = make([]JITValueDesc, 26)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[17] = d17
			ps26.OverlayValues[18] = d18
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			ps26.OverlayValues[23] = d23
			ps26.OverlayValues[24] = d24
			ps26.OverlayValues[25] = d25
					return bbs[3].RenderPS(ps26)
				}
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocReg {
				ctx.ProtectReg(d23.Reg)
			} else if d23.Loc == LocRegPair {
				ctx.ProtectReg(d23.Reg)
				ctx.ProtectReg(d23.Reg2)
			}
			d27 = d23
			if d27.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, int32(bbs[4].PhiBase)+int32(0))
			if d23.Loc == LocReg {
				ctx.UnprotectReg(d23.Reg)
			} else if d23.Loc == LocRegPair {
				ctx.UnprotectReg(d23.Reg)
				ctx.UnprotectReg(d23.Reg2)
			}
			ps28 := PhiState{General: ps.General}
			ps28.OverlayValues = make([]JITValueDesc, 28)
			ps28.OverlayValues[0] = d0
			ps28.OverlayValues[1] = d1
			ps28.OverlayValues[2] = d2
			ps28.OverlayValues[3] = d3
			ps28.OverlayValues[4] = d4
			ps28.OverlayValues[5] = d5
			ps28.OverlayValues[17] = d17
			ps28.OverlayValues[18] = d18
			ps28.OverlayValues[19] = d19
			ps28.OverlayValues[20] = d20
			ps28.OverlayValues[21] = d21
			ps28.OverlayValues[22] = d22
			ps28.OverlayValues[23] = d23
			ps28.OverlayValues[24] = d24
			ps28.OverlayValues[25] = d25
			ps28.OverlayValues[27] = d27
			ps28.PhiValues = make([]JITValueDesc, 1)
			d29 = d23
			ps28.PhiValues[0] = d29
				return bbs[4].RenderPS(ps28)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d25.Reg, 0)
			ctx.EmitJcc(CcNE, lbl16)
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl17)
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocReg {
				ctx.ProtectReg(d23.Reg)
			} else if d23.Loc == LocRegPair {
				ctx.ProtectReg(d23.Reg)
				ctx.ProtectReg(d23.Reg2)
			}
			d30 = d23
			if d30.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, int32(bbs[4].PhiBase)+int32(0))
			if d23.Loc == LocReg {
				ctx.UnprotectReg(d23.Reg)
			} else if d23.Loc == LocRegPair {
				ctx.UnprotectReg(d23.Reg)
				ctx.UnprotectReg(d23.Reg2)
			}
			ctx.EmitJmp(lbl5)
			ps31 := PhiState{General: true}
			ps31.OverlayValues = make([]JITValueDesc, 31)
			ps31.OverlayValues[0] = d0
			ps31.OverlayValues[1] = d1
			ps31.OverlayValues[2] = d2
			ps31.OverlayValues[3] = d3
			ps31.OverlayValues[4] = d4
			ps31.OverlayValues[5] = d5
			ps31.OverlayValues[17] = d17
			ps31.OverlayValues[18] = d18
			ps31.OverlayValues[19] = d19
			ps31.OverlayValues[20] = d20
			ps31.OverlayValues[21] = d21
			ps31.OverlayValues[22] = d22
			ps31.OverlayValues[23] = d23
			ps31.OverlayValues[24] = d24
			ps31.OverlayValues[25] = d25
			ps31.OverlayValues[27] = d27
			ps31.OverlayValues[29] = d29
			ps31.OverlayValues[30] = d30
			ps32 := PhiState{General: true}
			ps32.OverlayValues = make([]JITValueDesc, 31)
			ps32.OverlayValues[0] = d0
			ps32.OverlayValues[1] = d1
			ps32.OverlayValues[2] = d2
			ps32.OverlayValues[3] = d3
			ps32.OverlayValues[4] = d4
			ps32.OverlayValues[5] = d5
			ps32.OverlayValues[17] = d17
			ps32.OverlayValues[18] = d18
			ps32.OverlayValues[19] = d19
			ps32.OverlayValues[20] = d20
			ps32.OverlayValues[21] = d21
			ps32.OverlayValues[22] = d22
			ps32.OverlayValues[23] = d23
			ps32.OverlayValues[24] = d24
			ps32.OverlayValues[25] = d25
			ps32.OverlayValues[27] = d27
			ps32.OverlayValues[29] = d29
			ps32.OverlayValues[30] = d30
			ps32.PhiValues = make([]JITValueDesc, 1)
			d33 = d23
			ps32.PhiValues[0] = d33
			snap34 := d0
			snap35 := d1
			snap36 := d2
			snap37 := d3
			snap38 := d4
			snap39 := d5
			snap40 := d17
			snap41 := d18
			snap42 := d19
			snap43 := d20
			snap44 := d21
			snap45 := d22
			snap46 := d23
			snap47 := d24
			snap48 := d25
			snap49 := d27
			snap50 := d29
			snap51 := d30
			snap52 := d33
			alloc53 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps32)
			}
			ctx.RestoreAllocState(alloc53)
			d0 = snap34
			d1 = snap35
			d2 = snap36
			d3 = snap37
			d4 = snap38
			d5 = snap39
			d17 = snap40
			d18 = snap41
			d19 = snap42
			d20 = snap43
			d21 = snap44
			d22 = snap45
			d23 = snap46
			d24 = snap47
			d25 = snap48
			d27 = snap49
			d29 = snap50
			d30 = snap51
			d33 = snap52
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps31)
			}
			return result
			ctx.FreeDesc(&d24)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
			ps54 := PhiState{General: ps.General}
			ps54.OverlayValues = make([]JITValueDesc, 34)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[2] = d2
			ps54.OverlayValues[3] = d3
			ps54.OverlayValues[4] = d4
			ps54.OverlayValues[5] = d5
			ps54.OverlayValues[17] = d17
			ps54.OverlayValues[18] = d18
			ps54.OverlayValues[19] = d19
			ps54.OverlayValues[20] = d20
			ps54.OverlayValues[21] = d21
			ps54.OverlayValues[22] = d22
			ps54.OverlayValues[23] = d23
			ps54.OverlayValues[24] = d24
			ps54.OverlayValues[25] = d25
			ps54.OverlayValues[27] = d27
			ps54.OverlayValues[29] = d29
			ps54.OverlayValues[30] = d30
			ps54.OverlayValues[33] = d33
			ps54.PhiValues = make([]JITValueDesc, 1)
			d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps54.PhiValues[0] = d55
			if ps54.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps54)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d56 := ps.PhiValues[0]
					ctx.EnsureDesc(&d56)
					ctx.EmitStoreToStack(d56, int32(bbs[4].PhiBase)+int32(0))
				}
				if bbs[4].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			var d57 JITValueDesc
			if d0.Loc == LocImm && d20.Loc == LocImm {
				d57 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() >= d20.Imm.Int())}
			} else if d20.Loc == LocImm {
				r2 := ctx.AllocRegExcept(d0.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d0.Reg, int32(d20.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
					ctx.EmitCmpInt64(d0.Reg, RegR11)
				}
				ctx.EmitSetcc(r2, CcGE)
				d57 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d57)
			} else if d0.Loc == LocImm {
				r3 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d20.Reg)
				ctx.EmitSetcc(r3, CcGE)
				d57 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d57)
			} else {
				r4 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitCmpInt64(d0.Reg, d20.Reg)
				ctx.EmitSetcc(r4, CcGE)
				d57 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d57)
			}
			d58 = d57
			ctx.EnsureDesc(&d58)
			if d58.Loc != LocImm && d58.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d58.Loc == LocImm {
				if d58.Imm.Bool() {
			ps59 := PhiState{General: ps.General}
			ps59.OverlayValues = make([]JITValueDesc, 59)
			ps59.OverlayValues[0] = d0
			ps59.OverlayValues[1] = d1
			ps59.OverlayValues[2] = d2
			ps59.OverlayValues[3] = d3
			ps59.OverlayValues[4] = d4
			ps59.OverlayValues[5] = d5
			ps59.OverlayValues[17] = d17
			ps59.OverlayValues[18] = d18
			ps59.OverlayValues[19] = d19
			ps59.OverlayValues[20] = d20
			ps59.OverlayValues[21] = d21
			ps59.OverlayValues[22] = d22
			ps59.OverlayValues[23] = d23
			ps59.OverlayValues[24] = d24
			ps59.OverlayValues[25] = d25
			ps59.OverlayValues[27] = d27
			ps59.OverlayValues[29] = d29
			ps59.OverlayValues[30] = d30
			ps59.OverlayValues[33] = d33
			ps59.OverlayValues[55] = d55
			ps59.OverlayValues[56] = d56
			ps59.OverlayValues[57] = d57
			ps59.OverlayValues[58] = d58
					return bbs[5].RenderPS(ps59)
				}
			ps60 := PhiState{General: ps.General}
			ps60.OverlayValues = make([]JITValueDesc, 59)
			ps60.OverlayValues[0] = d0
			ps60.OverlayValues[1] = d1
			ps60.OverlayValues[2] = d2
			ps60.OverlayValues[3] = d3
			ps60.OverlayValues[4] = d4
			ps60.OverlayValues[5] = d5
			ps60.OverlayValues[17] = d17
			ps60.OverlayValues[18] = d18
			ps60.OverlayValues[19] = d19
			ps60.OverlayValues[20] = d20
			ps60.OverlayValues[21] = d21
			ps60.OverlayValues[22] = d22
			ps60.OverlayValues[23] = d23
			ps60.OverlayValues[24] = d24
			ps60.OverlayValues[25] = d25
			ps60.OverlayValues[27] = d27
			ps60.OverlayValues[29] = d29
			ps60.OverlayValues[30] = d30
			ps60.OverlayValues[33] = d33
			ps60.OverlayValues[55] = d55
			ps60.OverlayValues[56] = d56
			ps60.OverlayValues[57] = d57
			ps60.OverlayValues[58] = d58
				return bbs[6].RenderPS(ps60)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d61 := ps.PhiValues[0]
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreToStack(d61, int32(bbs[4].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl18 := ctx.ReserveLabel()
			lbl19 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d58.Reg, 0)
			ctx.EmitJcc(CcNE, lbl18)
			ctx.EmitJmp(lbl19)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl7)
			ps62 := PhiState{General: true}
			ps62.OverlayValues = make([]JITValueDesc, 62)
			ps62.OverlayValues[0] = d0
			ps62.OverlayValues[1] = d1
			ps62.OverlayValues[2] = d2
			ps62.OverlayValues[3] = d3
			ps62.OverlayValues[4] = d4
			ps62.OverlayValues[5] = d5
			ps62.OverlayValues[17] = d17
			ps62.OverlayValues[18] = d18
			ps62.OverlayValues[19] = d19
			ps62.OverlayValues[20] = d20
			ps62.OverlayValues[21] = d21
			ps62.OverlayValues[22] = d22
			ps62.OverlayValues[23] = d23
			ps62.OverlayValues[24] = d24
			ps62.OverlayValues[25] = d25
			ps62.OverlayValues[27] = d27
			ps62.OverlayValues[29] = d29
			ps62.OverlayValues[30] = d30
			ps62.OverlayValues[33] = d33
			ps62.OverlayValues[55] = d55
			ps62.OverlayValues[56] = d56
			ps62.OverlayValues[57] = d57
			ps62.OverlayValues[58] = d58
			ps62.OverlayValues[61] = d61
			ps63 := PhiState{General: true}
			ps63.OverlayValues = make([]JITValueDesc, 62)
			ps63.OverlayValues[0] = d0
			ps63.OverlayValues[1] = d1
			ps63.OverlayValues[2] = d2
			ps63.OverlayValues[3] = d3
			ps63.OverlayValues[4] = d4
			ps63.OverlayValues[5] = d5
			ps63.OverlayValues[17] = d17
			ps63.OverlayValues[18] = d18
			ps63.OverlayValues[19] = d19
			ps63.OverlayValues[20] = d20
			ps63.OverlayValues[21] = d21
			ps63.OverlayValues[22] = d22
			ps63.OverlayValues[23] = d23
			ps63.OverlayValues[24] = d24
			ps63.OverlayValues[25] = d25
			ps63.OverlayValues[27] = d27
			ps63.OverlayValues[29] = d29
			ps63.OverlayValues[30] = d30
			ps63.OverlayValues[33] = d33
			ps63.OverlayValues[55] = d55
			ps63.OverlayValues[56] = d56
			ps63.OverlayValues[57] = d57
			ps63.OverlayValues[58] = d58
			ps63.OverlayValues[61] = d61
			snap64 := d0
			snap65 := d1
			snap66 := d2
			snap67 := d3
			snap68 := d4
			snap69 := d5
			snap70 := d17
			snap71 := d18
			snap72 := d19
			snap73 := d20
			snap74 := d21
			snap75 := d22
			snap76 := d23
			snap77 := d24
			snap78 := d25
			snap79 := d27
			snap80 := d29
			snap81 := d30
			snap82 := d33
			snap83 := d55
			snap84 := d56
			snap85 := d57
			snap86 := d58
			snap87 := d61
			alloc88 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps63)
			}
			ctx.RestoreAllocState(alloc88)
			d0 = snap64
			d1 = snap65
			d2 = snap66
			d3 = snap67
			d4 = snap68
			d5 = snap69
			d17 = snap70
			d18 = snap71
			d19 = snap72
			d20 = snap73
			d21 = snap74
			d22 = snap75
			d23 = snap76
			d24 = snap77
			d25 = snap78
			d27 = snap79
			d29 = snap80
			d30 = snap81
			d33 = snap82
			d55 = snap83
			d56 = snap84
			d57 = snap85
			d58 = snap86
			d61 = snap87
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps62)
			}
			return result
			ctx.FreeDesc(&d57)
			return result
			}
			bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			ctx.ReclaimUntrackedRegs()
			d89 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d89, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			ctx.ReclaimUntrackedRegs()
			d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d90)
			var d91 JITValueDesc
			if d90.Loc == LocImm {
				d91 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d90.Imm.Int() > 2)}
			} else {
				r5 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d90.Reg, 2)
				ctx.EmitSetcc(r5, CcG)
				d91 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d91)
			}
			ctx.FreeDesc(&d90)
			d92 = d91
			ctx.EnsureDesc(&d92)
			if d92.Loc != LocImm && d92.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d92.Loc == LocImm {
				if d92.Imm.Bool() {
			ps93 := PhiState{General: ps.General}
			ps93.OverlayValues = make([]JITValueDesc, 93)
			ps93.OverlayValues[0] = d0
			ps93.OverlayValues[1] = d1
			ps93.OverlayValues[2] = d2
			ps93.OverlayValues[3] = d3
			ps93.OverlayValues[4] = d4
			ps93.OverlayValues[5] = d5
			ps93.OverlayValues[17] = d17
			ps93.OverlayValues[18] = d18
			ps93.OverlayValues[19] = d19
			ps93.OverlayValues[20] = d20
			ps93.OverlayValues[21] = d21
			ps93.OverlayValues[22] = d22
			ps93.OverlayValues[23] = d23
			ps93.OverlayValues[24] = d24
			ps93.OverlayValues[25] = d25
			ps93.OverlayValues[27] = d27
			ps93.OverlayValues[29] = d29
			ps93.OverlayValues[30] = d30
			ps93.OverlayValues[33] = d33
			ps93.OverlayValues[55] = d55
			ps93.OverlayValues[56] = d56
			ps93.OverlayValues[57] = d57
			ps93.OverlayValues[58] = d58
			ps93.OverlayValues[61] = d61
			ps93.OverlayValues[89] = d89
			ps93.OverlayValues[90] = d90
			ps93.OverlayValues[91] = d91
			ps93.OverlayValues[92] = d92
					return bbs[7].RenderPS(ps93)
				}
			ps94 := PhiState{General: ps.General}
			ps94.OverlayValues = make([]JITValueDesc, 93)
			ps94.OverlayValues[0] = d0
			ps94.OverlayValues[1] = d1
			ps94.OverlayValues[2] = d2
			ps94.OverlayValues[3] = d3
			ps94.OverlayValues[4] = d4
			ps94.OverlayValues[5] = d5
			ps94.OverlayValues[17] = d17
			ps94.OverlayValues[18] = d18
			ps94.OverlayValues[19] = d19
			ps94.OverlayValues[20] = d20
			ps94.OverlayValues[21] = d21
			ps94.OverlayValues[22] = d22
			ps94.OverlayValues[23] = d23
			ps94.OverlayValues[24] = d24
			ps94.OverlayValues[25] = d25
			ps94.OverlayValues[27] = d27
			ps94.OverlayValues[29] = d29
			ps94.OverlayValues[30] = d30
			ps94.OverlayValues[33] = d33
			ps94.OverlayValues[55] = d55
			ps94.OverlayValues[56] = d56
			ps94.OverlayValues[57] = d57
			ps94.OverlayValues[58] = d58
			ps94.OverlayValues[61] = d61
			ps94.OverlayValues[89] = d89
			ps94.OverlayValues[90] = d90
			ps94.OverlayValues[91] = d91
			ps94.OverlayValues[92] = d92
				return bbs[8].RenderPS(ps94)
			}
			if !ps.General {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d92.Reg, 0)
			ctx.EmitJcc(CcNE, lbl20)
			ctx.EmitJmp(lbl21)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl9)
			ps95 := PhiState{General: true}
			ps95.OverlayValues = make([]JITValueDesc, 93)
			ps95.OverlayValues[0] = d0
			ps95.OverlayValues[1] = d1
			ps95.OverlayValues[2] = d2
			ps95.OverlayValues[3] = d3
			ps95.OverlayValues[4] = d4
			ps95.OverlayValues[5] = d5
			ps95.OverlayValues[17] = d17
			ps95.OverlayValues[18] = d18
			ps95.OverlayValues[19] = d19
			ps95.OverlayValues[20] = d20
			ps95.OverlayValues[21] = d21
			ps95.OverlayValues[22] = d22
			ps95.OverlayValues[23] = d23
			ps95.OverlayValues[24] = d24
			ps95.OverlayValues[25] = d25
			ps95.OverlayValues[27] = d27
			ps95.OverlayValues[29] = d29
			ps95.OverlayValues[30] = d30
			ps95.OverlayValues[33] = d33
			ps95.OverlayValues[55] = d55
			ps95.OverlayValues[56] = d56
			ps95.OverlayValues[57] = d57
			ps95.OverlayValues[58] = d58
			ps95.OverlayValues[61] = d61
			ps95.OverlayValues[89] = d89
			ps95.OverlayValues[90] = d90
			ps95.OverlayValues[91] = d91
			ps95.OverlayValues[92] = d92
			ps96 := PhiState{General: true}
			ps96.OverlayValues = make([]JITValueDesc, 93)
			ps96.OverlayValues[0] = d0
			ps96.OverlayValues[1] = d1
			ps96.OverlayValues[2] = d2
			ps96.OverlayValues[3] = d3
			ps96.OverlayValues[4] = d4
			ps96.OverlayValues[5] = d5
			ps96.OverlayValues[17] = d17
			ps96.OverlayValues[18] = d18
			ps96.OverlayValues[19] = d19
			ps96.OverlayValues[20] = d20
			ps96.OverlayValues[21] = d21
			ps96.OverlayValues[22] = d22
			ps96.OverlayValues[23] = d23
			ps96.OverlayValues[24] = d24
			ps96.OverlayValues[25] = d25
			ps96.OverlayValues[27] = d27
			ps96.OverlayValues[29] = d29
			ps96.OverlayValues[30] = d30
			ps96.OverlayValues[33] = d33
			ps96.OverlayValues[55] = d55
			ps96.OverlayValues[56] = d56
			ps96.OverlayValues[57] = d57
			ps96.OverlayValues[58] = d58
			ps96.OverlayValues[61] = d61
			ps96.OverlayValues[89] = d89
			ps96.OverlayValues[90] = d90
			ps96.OverlayValues[91] = d91
			ps96.OverlayValues[92] = d92
			snap97 := d0
			snap98 := d1
			snap99 := d2
			snap100 := d3
			snap101 := d4
			snap102 := d5
			snap103 := d17
			snap104 := d18
			snap105 := d19
			snap106 := d20
			snap107 := d21
			snap108 := d22
			snap109 := d23
			snap110 := d24
			snap111 := d25
			snap112 := d27
			snap113 := d29
			snap114 := d30
			snap115 := d33
			snap116 := d55
			snap117 := d56
			snap118 := d57
			snap119 := d58
			snap120 := d61
			snap121 := d89
			snap122 := d90
			snap123 := d91
			snap124 := d92
			alloc125 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps96)
			}
			ctx.RestoreAllocState(alloc125)
			d0 = snap97
			d1 = snap98
			d2 = snap99
			d3 = snap100
			d4 = snap101
			d5 = snap102
			d17 = snap103
			d18 = snap104
			d19 = snap105
			d20 = snap106
			d21 = snap107
			d22 = snap108
			d23 = snap109
			d24 = snap110
			d25 = snap111
			d27 = snap112
			d29 = snap113
			d30 = snap114
			d33 = snap115
			d55 = snap116
			d56 = snap117
			d57 = snap118
			d58 = snap119
			d61 = snap120
			d89 = snap121
			d90 = snap122
			d91 = snap123
			d92 = snap124
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps95)
			}
			return result
			ctx.FreeDesc(&d91)
			return result
			}
			bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
			}
			ctx.ReclaimUntrackedRegs()
			d126 = args[2]
			d126.ID = 0
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d126)
			if d126.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d126.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d126.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d126)
				} else if d126.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d126)
				} else if d126.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d126)
				} else if d126.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d126.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d126 = tmpPair
			} else if d126.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d126.Type, Reg: ctx.AllocRegExcept(d126.Reg), Reg2: ctx.AllocRegExcept(d126.Reg)}
				switch d126.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d126)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d126)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d126)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d126)
				d126 = tmpPair
			}
			if d126.Loc != LocRegPair && d126.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d127 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d126}, 1)
			ctx.FreeDesc(&d126)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d127)
			var d128 JITValueDesc
			if d0.Loc == LocImm && d127.Loc == LocImm {
				d128 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d127.Imm.Int())}
			} else if d127.Loc == LocImm && d127.Imm.Int() == 0 {
				r6 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r6, d0.Reg)
				d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
				ctx.BindReg(r6, &d128)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d127.Reg}
				ctx.BindReg(d127.Reg, &d128)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d127.Reg)
				d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else if d127.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d127.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else {
				r7 := ctx.AllocRegExcept(d0.Reg, d127.Reg)
				ctx.EmitMovRegReg(r7, d0.Reg)
				ctx.EmitAddInt64(r7, d127.Reg)
				d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
				ctx.BindReg(r7, &d128)
			}
			if d128.Loc == LocReg && d0.Loc == LocReg && d128.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d20)
			var d129 JITValueDesc
			if d128.Loc == LocImm && d20.Loc == LocImm {
				d129 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d128.Imm.Int() > d20.Imm.Int())}
			} else if d20.Loc == LocImm {
				r8 := ctx.AllocReg()
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d128.Reg, int32(d20.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
					ctx.EmitCmpInt64(d128.Reg, RegR11)
				}
				ctx.EmitSetcc(r8, CcG)
				d129 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d129)
			} else if d128.Loc == LocImm {
				r9 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d128.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d20.Reg)
				ctx.EmitSetcc(r9, CcG)
				d129 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d129)
			} else {
				r10 := ctx.AllocReg()
				ctx.EmitCmpInt64(d128.Reg, d20.Reg)
				ctx.EmitSetcc(r10, CcG)
				d129 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d129)
			}
			ctx.FreeDesc(&d128)
			d130 = d129
			ctx.EnsureDesc(&d130)
			if d130.Loc != LocImm && d130.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d130.Loc == LocImm {
				if d130.Imm.Bool() {
			ps131 := PhiState{General: ps.General}
			ps131.OverlayValues = make([]JITValueDesc, 131)
			ps131.OverlayValues[0] = d0
			ps131.OverlayValues[1] = d1
			ps131.OverlayValues[2] = d2
			ps131.OverlayValues[3] = d3
			ps131.OverlayValues[4] = d4
			ps131.OverlayValues[5] = d5
			ps131.OverlayValues[17] = d17
			ps131.OverlayValues[18] = d18
			ps131.OverlayValues[19] = d19
			ps131.OverlayValues[20] = d20
			ps131.OverlayValues[21] = d21
			ps131.OverlayValues[22] = d22
			ps131.OverlayValues[23] = d23
			ps131.OverlayValues[24] = d24
			ps131.OverlayValues[25] = d25
			ps131.OverlayValues[27] = d27
			ps131.OverlayValues[29] = d29
			ps131.OverlayValues[30] = d30
			ps131.OverlayValues[33] = d33
			ps131.OverlayValues[55] = d55
			ps131.OverlayValues[56] = d56
			ps131.OverlayValues[57] = d57
			ps131.OverlayValues[58] = d58
			ps131.OverlayValues[61] = d61
			ps131.OverlayValues[89] = d89
			ps131.OverlayValues[90] = d90
			ps131.OverlayValues[91] = d91
			ps131.OverlayValues[92] = d92
			ps131.OverlayValues[126] = d126
			ps131.OverlayValues[127] = d127
			ps131.OverlayValues[128] = d128
			ps131.OverlayValues[129] = d129
			ps131.OverlayValues[130] = d130
					return bbs[9].RenderPS(ps131)
				}
			ctx.EnsureDesc(&d127)
			if d127.Loc == LocReg {
				ctx.ProtectReg(d127.Reg)
			} else if d127.Loc == LocRegPair {
				ctx.ProtectReg(d127.Reg)
				ctx.ProtectReg(d127.Reg2)
			}
			d132 = d127
			if d132.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d132)
			ctx.EmitStoreToStack(d132, int32(bbs[10].PhiBase)+int32(0))
			if d127.Loc == LocReg {
				ctx.UnprotectReg(d127.Reg)
			} else if d127.Loc == LocRegPair {
				ctx.UnprotectReg(d127.Reg)
				ctx.UnprotectReg(d127.Reg2)
			}
			ps133 := PhiState{General: ps.General}
			ps133.OverlayValues = make([]JITValueDesc, 133)
			ps133.OverlayValues[0] = d0
			ps133.OverlayValues[1] = d1
			ps133.OverlayValues[2] = d2
			ps133.OverlayValues[3] = d3
			ps133.OverlayValues[4] = d4
			ps133.OverlayValues[5] = d5
			ps133.OverlayValues[17] = d17
			ps133.OverlayValues[18] = d18
			ps133.OverlayValues[19] = d19
			ps133.OverlayValues[20] = d20
			ps133.OverlayValues[21] = d21
			ps133.OverlayValues[22] = d22
			ps133.OverlayValues[23] = d23
			ps133.OverlayValues[24] = d24
			ps133.OverlayValues[25] = d25
			ps133.OverlayValues[27] = d27
			ps133.OverlayValues[29] = d29
			ps133.OverlayValues[30] = d30
			ps133.OverlayValues[33] = d33
			ps133.OverlayValues[55] = d55
			ps133.OverlayValues[56] = d56
			ps133.OverlayValues[57] = d57
			ps133.OverlayValues[58] = d58
			ps133.OverlayValues[61] = d61
			ps133.OverlayValues[89] = d89
			ps133.OverlayValues[90] = d90
			ps133.OverlayValues[91] = d91
			ps133.OverlayValues[92] = d92
			ps133.OverlayValues[126] = d126
			ps133.OverlayValues[127] = d127
			ps133.OverlayValues[128] = d128
			ps133.OverlayValues[129] = d129
			ps133.OverlayValues[130] = d130
			ps133.OverlayValues[132] = d132
			ps133.PhiValues = make([]JITValueDesc, 1)
			d134 = d127
			ps133.PhiValues[0] = d134
				return bbs[10].RenderPS(ps133)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl22 := ctx.ReserveLabel()
			lbl23 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d130.Reg, 0)
			ctx.EmitJcc(CcNE, lbl22)
			ctx.EmitJmp(lbl23)
			ctx.MarkLabel(lbl22)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl23)
			ctx.EnsureDesc(&d127)
			if d127.Loc == LocReg {
				ctx.ProtectReg(d127.Reg)
			} else if d127.Loc == LocRegPair {
				ctx.ProtectReg(d127.Reg)
				ctx.ProtectReg(d127.Reg2)
			}
			d135 = d127
			if d135.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d135)
			ctx.EmitStoreToStack(d135, int32(bbs[10].PhiBase)+int32(0))
			if d127.Loc == LocReg {
				ctx.UnprotectReg(d127.Reg)
			} else if d127.Loc == LocRegPair {
				ctx.UnprotectReg(d127.Reg)
				ctx.UnprotectReg(d127.Reg2)
			}
			ctx.EmitJmp(lbl11)
			ps136 := PhiState{General: true}
			ps136.OverlayValues = make([]JITValueDesc, 136)
			ps136.OverlayValues[0] = d0
			ps136.OverlayValues[1] = d1
			ps136.OverlayValues[2] = d2
			ps136.OverlayValues[3] = d3
			ps136.OverlayValues[4] = d4
			ps136.OverlayValues[5] = d5
			ps136.OverlayValues[17] = d17
			ps136.OverlayValues[18] = d18
			ps136.OverlayValues[19] = d19
			ps136.OverlayValues[20] = d20
			ps136.OverlayValues[21] = d21
			ps136.OverlayValues[22] = d22
			ps136.OverlayValues[23] = d23
			ps136.OverlayValues[24] = d24
			ps136.OverlayValues[25] = d25
			ps136.OverlayValues[27] = d27
			ps136.OverlayValues[29] = d29
			ps136.OverlayValues[30] = d30
			ps136.OverlayValues[33] = d33
			ps136.OverlayValues[55] = d55
			ps136.OverlayValues[56] = d56
			ps136.OverlayValues[57] = d57
			ps136.OverlayValues[58] = d58
			ps136.OverlayValues[61] = d61
			ps136.OverlayValues[89] = d89
			ps136.OverlayValues[90] = d90
			ps136.OverlayValues[91] = d91
			ps136.OverlayValues[92] = d92
			ps136.OverlayValues[126] = d126
			ps136.OverlayValues[127] = d127
			ps136.OverlayValues[128] = d128
			ps136.OverlayValues[129] = d129
			ps136.OverlayValues[130] = d130
			ps136.OverlayValues[132] = d132
			ps136.OverlayValues[134] = d134
			ps136.OverlayValues[135] = d135
			ps137 := PhiState{General: true}
			ps137.OverlayValues = make([]JITValueDesc, 136)
			ps137.OverlayValues[0] = d0
			ps137.OverlayValues[1] = d1
			ps137.OverlayValues[2] = d2
			ps137.OverlayValues[3] = d3
			ps137.OverlayValues[4] = d4
			ps137.OverlayValues[5] = d5
			ps137.OverlayValues[17] = d17
			ps137.OverlayValues[18] = d18
			ps137.OverlayValues[19] = d19
			ps137.OverlayValues[20] = d20
			ps137.OverlayValues[21] = d21
			ps137.OverlayValues[22] = d22
			ps137.OverlayValues[23] = d23
			ps137.OverlayValues[24] = d24
			ps137.OverlayValues[25] = d25
			ps137.OverlayValues[27] = d27
			ps137.OverlayValues[29] = d29
			ps137.OverlayValues[30] = d30
			ps137.OverlayValues[33] = d33
			ps137.OverlayValues[55] = d55
			ps137.OverlayValues[56] = d56
			ps137.OverlayValues[57] = d57
			ps137.OverlayValues[58] = d58
			ps137.OverlayValues[61] = d61
			ps137.OverlayValues[89] = d89
			ps137.OverlayValues[90] = d90
			ps137.OverlayValues[91] = d91
			ps137.OverlayValues[92] = d92
			ps137.OverlayValues[126] = d126
			ps137.OverlayValues[127] = d127
			ps137.OverlayValues[128] = d128
			ps137.OverlayValues[129] = d129
			ps137.OverlayValues[130] = d130
			ps137.OverlayValues[132] = d132
			ps137.OverlayValues[134] = d134
			ps137.OverlayValues[135] = d135
			ps137.PhiValues = make([]JITValueDesc, 1)
			d138 = d127
			ps137.PhiValues[0] = d138
			snap139 := d0
			snap140 := d1
			snap141 := d2
			snap142 := d3
			snap143 := d4
			snap144 := d5
			snap145 := d17
			snap146 := d18
			snap147 := d19
			snap148 := d20
			snap149 := d21
			snap150 := d22
			snap151 := d23
			snap152 := d24
			snap153 := d25
			snap154 := d27
			snap155 := d29
			snap156 := d30
			snap157 := d33
			snap158 := d55
			snap159 := d56
			snap160 := d57
			snap161 := d58
			snap162 := d61
			snap163 := d89
			snap164 := d90
			snap165 := d91
			snap166 := d92
			snap167 := d126
			snap168 := d127
			snap169 := d128
			snap170 := d129
			snap171 := d130
			snap172 := d132
			snap173 := d134
			snap174 := d135
			snap175 := d138
			alloc176 := ctx.SnapshotAllocState()
			if !bbs[10].Rendered {
				bbs[10].RenderPS(ps137)
			}
			ctx.RestoreAllocState(alloc176)
			d0 = snap139
			d1 = snap140
			d2 = snap141
			d3 = snap142
			d4 = snap143
			d5 = snap144
			d17 = snap145
			d18 = snap146
			d19 = snap147
			d20 = snap148
			d21 = snap149
			d22 = snap150
			d23 = snap151
			d24 = snap152
			d25 = snap153
			d27 = snap154
			d29 = snap155
			d30 = snap156
			d33 = snap157
			d55 = snap158
			d56 = snap159
			d57 = snap160
			d58 = snap161
			d61 = snap162
			d89 = snap163
			d90 = snap164
			d91 = snap165
			d92 = snap166
			d126 = snap167
			d127 = snap168
			d128 = snap169
			d129 = snap170
			d130 = snap171
			d132 = snap172
			d134 = snap173
			d135 = snap174
			d138 = snap175
			if !bbs[9].Rendered {
				return bbs[9].RenderPS(ps136)
			}
			return result
			ctx.FreeDesc(&d129)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
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
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
				d138 = ps.OverlayValues[138]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			var d177 JITValueDesc
			ctx.EnsureDesc(&d18)
			if d18.Loc == LocRegPair {
				d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d18.Reg2}
				ctx.BindReg(d18.Reg2, &d177)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d177)
			var d179 JITValueDesc
			if d177.Loc == LocImm && d0.Loc == LocImm {
				d179 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d177.Imm.Int() - d0.Imm.Int())}
			} else {
				r11 := ctx.AllocReg()
				if d177.Loc == LocImm {
					ctx.EmitMovRegImm64(r11, uint64(d177.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r11, d177.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitSubInt64(r11, RegR11)
				} else {
					ctx.EmitSubInt64(r11, d0.Reg)
				}
				d179 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d179)
			}
			var d180 JITValueDesc
			if d18.Loc == LocImm && d0.Loc == LocImm {
				d180 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d18.Imm.Int() + d0.Imm.Int())}
			} else {
				r12 := ctx.AllocReg()
				if d18.Loc == LocImm {
					ctx.EmitMovRegImm64(r12, uint64(d18.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r12, d18.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitAddInt64(r12, RegR11)
				} else {
					ctx.EmitAddInt64(r12, d0.Reg)
				}
				d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d180)
			}
			var d181 JITValueDesc
			r13 := ctx.AllocReg()
			r14 := ctx.AllocReg()
			if d180.Loc == LocImm {
				ctx.EmitMovRegImm64(r13, uint64(d180.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r13, d180.Reg)
				ctx.FreeReg(d180.Reg)
			}
			if d179.Loc == LocImm {
				ctx.EmitMovRegImm64(r14, uint64(d179.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r14, d179.Reg)
				ctx.FreeReg(d179.Reg)
			}
			d181 = JITValueDesc{Loc: LocRegPair, Reg: r13, Reg2: r14}
			ctx.BindReg(r13, &d181)
			ctx.BindReg(r14, &d181)
			d182 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d181}, 2)
			ctx.EmitMovPairToResult(&d182, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[9].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
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
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
				d138 = ps.OverlayValues[138]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			var d183 JITValueDesc
			if d20.Loc == LocImm && d0.Loc == LocImm {
				d183 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int() - d0.Imm.Int())}
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				r15 := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(r15, d20.Reg)
				d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
				ctx.BindReg(r15, &d183)
			} else if d20.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.EmitSubInt64(scratch, d0.Reg)
				d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(scratch, d20.Reg)
				if d0.Imm.Int() >= -2147483648 && d0.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d0.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitSubInt64(scratch, RegR11)
				}
				d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else {
				r16 := ctx.AllocRegExcept(d20.Reg, d0.Reg)
				ctx.EmitMovRegReg(r16, d20.Reg)
				ctx.EmitSubInt64(r16, d0.Reg)
				d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
				ctx.BindReg(r16, &d183)
			}
			if d183.Loc == LocReg && d20.Loc == LocReg && d183.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = LocNone
			}
			ctx.FreeDesc(&d20)
			ctx.EnsureDesc(&d183)
			if d183.Loc == LocReg {
				ctx.ProtectReg(d183.Reg)
			} else if d183.Loc == LocRegPair {
				ctx.ProtectReg(d183.Reg)
				ctx.ProtectReg(d183.Reg2)
			}
			d184 = d183
			if d184.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d184)
			ctx.EmitStoreToStack(d184, int32(bbs[10].PhiBase)+int32(0))
			if d183.Loc == LocReg {
				ctx.UnprotectReg(d183.Reg)
			} else if d183.Loc == LocRegPair {
				ctx.UnprotectReg(d183.Reg)
				ctx.UnprotectReg(d183.Reg2)
			}
			ps185 := PhiState{General: ps.General}
			ps185.OverlayValues = make([]JITValueDesc, 185)
			ps185.OverlayValues[0] = d0
			ps185.OverlayValues[1] = d1
			ps185.OverlayValues[2] = d2
			ps185.OverlayValues[3] = d3
			ps185.OverlayValues[4] = d4
			ps185.OverlayValues[5] = d5
			ps185.OverlayValues[17] = d17
			ps185.OverlayValues[18] = d18
			ps185.OverlayValues[19] = d19
			ps185.OverlayValues[20] = d20
			ps185.OverlayValues[21] = d21
			ps185.OverlayValues[22] = d22
			ps185.OverlayValues[23] = d23
			ps185.OverlayValues[24] = d24
			ps185.OverlayValues[25] = d25
			ps185.OverlayValues[27] = d27
			ps185.OverlayValues[29] = d29
			ps185.OverlayValues[30] = d30
			ps185.OverlayValues[33] = d33
			ps185.OverlayValues[55] = d55
			ps185.OverlayValues[56] = d56
			ps185.OverlayValues[57] = d57
			ps185.OverlayValues[58] = d58
			ps185.OverlayValues[61] = d61
			ps185.OverlayValues[89] = d89
			ps185.OverlayValues[90] = d90
			ps185.OverlayValues[91] = d91
			ps185.OverlayValues[92] = d92
			ps185.OverlayValues[126] = d126
			ps185.OverlayValues[127] = d127
			ps185.OverlayValues[128] = d128
			ps185.OverlayValues[129] = d129
			ps185.OverlayValues[130] = d130
			ps185.OverlayValues[132] = d132
			ps185.OverlayValues[134] = d134
			ps185.OverlayValues[135] = d135
			ps185.OverlayValues[138] = d138
			ps185.OverlayValues[177] = d177
			ps185.OverlayValues[178] = d178
			ps185.OverlayValues[179] = d179
			ps185.OverlayValues[180] = d180
			ps185.OverlayValues[181] = d181
			ps185.OverlayValues[182] = d182
			ps185.OverlayValues[183] = d183
			ps185.OverlayValues[184] = d184
			ps185.PhiValues = make([]JITValueDesc, 1)
			d186 = d183
			ps185.PhiValues[0] = d186
			if ps185.General && bbs[10].Rendered {
				ctx.EmitJmp(lbl11)
				return result
			}
			return bbs[10].RenderPS(ps185)
			return result
			}
			bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d187 := ps.PhiValues[0]
					ctx.EnsureDesc(&d187)
					ctx.EmitStoreToStack(d187, int32(bbs[10].PhiBase)+int32(0))
				}
				if bbs[10].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
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
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
				d138 = ps.OverlayValues[138]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			var d188 JITValueDesc
			if d1.Loc == LocImm {
				d188 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < 0)}
			} else {
				r17 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitCmpRegImm32(d1.Reg, 0)
				ctx.EmitSetcc(r17, CcL)
				d188 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d188)
			}
			d189 = d188
			ctx.EnsureDesc(&d189)
			if d189.Loc != LocImm && d189.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d189.Loc == LocImm {
				if d189.Imm.Bool() {
			ps190 := PhiState{General: ps.General}
			ps190.OverlayValues = make([]JITValueDesc, 190)
			ps190.OverlayValues[0] = d0
			ps190.OverlayValues[1] = d1
			ps190.OverlayValues[2] = d2
			ps190.OverlayValues[3] = d3
			ps190.OverlayValues[4] = d4
			ps190.OverlayValues[5] = d5
			ps190.OverlayValues[17] = d17
			ps190.OverlayValues[18] = d18
			ps190.OverlayValues[19] = d19
			ps190.OverlayValues[20] = d20
			ps190.OverlayValues[21] = d21
			ps190.OverlayValues[22] = d22
			ps190.OverlayValues[23] = d23
			ps190.OverlayValues[24] = d24
			ps190.OverlayValues[25] = d25
			ps190.OverlayValues[27] = d27
			ps190.OverlayValues[29] = d29
			ps190.OverlayValues[30] = d30
			ps190.OverlayValues[33] = d33
			ps190.OverlayValues[55] = d55
			ps190.OverlayValues[56] = d56
			ps190.OverlayValues[57] = d57
			ps190.OverlayValues[58] = d58
			ps190.OverlayValues[61] = d61
			ps190.OverlayValues[89] = d89
			ps190.OverlayValues[90] = d90
			ps190.OverlayValues[91] = d91
			ps190.OverlayValues[92] = d92
			ps190.OverlayValues[126] = d126
			ps190.OverlayValues[127] = d127
			ps190.OverlayValues[128] = d128
			ps190.OverlayValues[129] = d129
			ps190.OverlayValues[130] = d130
			ps190.OverlayValues[132] = d132
			ps190.OverlayValues[134] = d134
			ps190.OverlayValues[135] = d135
			ps190.OverlayValues[138] = d138
			ps190.OverlayValues[177] = d177
			ps190.OverlayValues[178] = d178
			ps190.OverlayValues[179] = d179
			ps190.OverlayValues[180] = d180
			ps190.OverlayValues[181] = d181
			ps190.OverlayValues[182] = d182
			ps190.OverlayValues[183] = d183
			ps190.OverlayValues[184] = d184
			ps190.OverlayValues[186] = d186
			ps190.OverlayValues[187] = d187
			ps190.OverlayValues[188] = d188
			ps190.OverlayValues[189] = d189
					return bbs[11].RenderPS(ps190)
				}
			ps191 := PhiState{General: ps.General}
			ps191.OverlayValues = make([]JITValueDesc, 190)
			ps191.OverlayValues[0] = d0
			ps191.OverlayValues[1] = d1
			ps191.OverlayValues[2] = d2
			ps191.OverlayValues[3] = d3
			ps191.OverlayValues[4] = d4
			ps191.OverlayValues[5] = d5
			ps191.OverlayValues[17] = d17
			ps191.OverlayValues[18] = d18
			ps191.OverlayValues[19] = d19
			ps191.OverlayValues[20] = d20
			ps191.OverlayValues[21] = d21
			ps191.OverlayValues[22] = d22
			ps191.OverlayValues[23] = d23
			ps191.OverlayValues[24] = d24
			ps191.OverlayValues[25] = d25
			ps191.OverlayValues[27] = d27
			ps191.OverlayValues[29] = d29
			ps191.OverlayValues[30] = d30
			ps191.OverlayValues[33] = d33
			ps191.OverlayValues[55] = d55
			ps191.OverlayValues[56] = d56
			ps191.OverlayValues[57] = d57
			ps191.OverlayValues[58] = d58
			ps191.OverlayValues[61] = d61
			ps191.OverlayValues[89] = d89
			ps191.OverlayValues[90] = d90
			ps191.OverlayValues[91] = d91
			ps191.OverlayValues[92] = d92
			ps191.OverlayValues[126] = d126
			ps191.OverlayValues[127] = d127
			ps191.OverlayValues[128] = d128
			ps191.OverlayValues[129] = d129
			ps191.OverlayValues[130] = d130
			ps191.OverlayValues[132] = d132
			ps191.OverlayValues[134] = d134
			ps191.OverlayValues[135] = d135
			ps191.OverlayValues[138] = d138
			ps191.OverlayValues[177] = d177
			ps191.OverlayValues[178] = d178
			ps191.OverlayValues[179] = d179
			ps191.OverlayValues[180] = d180
			ps191.OverlayValues[181] = d181
			ps191.OverlayValues[182] = d182
			ps191.OverlayValues[183] = d183
			ps191.OverlayValues[184] = d184
			ps191.OverlayValues[186] = d186
			ps191.OverlayValues[187] = d187
			ps191.OverlayValues[188] = d188
			ps191.OverlayValues[189] = d189
				return bbs[12].RenderPS(ps191)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d192 := ps.PhiValues[0]
					ctx.EnsureDesc(&d192)
					ctx.EmitStoreToStack(d192, int32(bbs[10].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[10].RenderPS(ps)
			}
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d189.Reg, 0)
			ctx.EmitJcc(CcNE, lbl24)
			ctx.EmitJmp(lbl25)
			ctx.MarkLabel(lbl24)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl25)
			ctx.EmitJmp(lbl13)
			ps193 := PhiState{General: true}
			ps193.OverlayValues = make([]JITValueDesc, 193)
			ps193.OverlayValues[0] = d0
			ps193.OverlayValues[1] = d1
			ps193.OverlayValues[2] = d2
			ps193.OverlayValues[3] = d3
			ps193.OverlayValues[4] = d4
			ps193.OverlayValues[5] = d5
			ps193.OverlayValues[17] = d17
			ps193.OverlayValues[18] = d18
			ps193.OverlayValues[19] = d19
			ps193.OverlayValues[20] = d20
			ps193.OverlayValues[21] = d21
			ps193.OverlayValues[22] = d22
			ps193.OverlayValues[23] = d23
			ps193.OverlayValues[24] = d24
			ps193.OverlayValues[25] = d25
			ps193.OverlayValues[27] = d27
			ps193.OverlayValues[29] = d29
			ps193.OverlayValues[30] = d30
			ps193.OverlayValues[33] = d33
			ps193.OverlayValues[55] = d55
			ps193.OverlayValues[56] = d56
			ps193.OverlayValues[57] = d57
			ps193.OverlayValues[58] = d58
			ps193.OverlayValues[61] = d61
			ps193.OverlayValues[89] = d89
			ps193.OverlayValues[90] = d90
			ps193.OverlayValues[91] = d91
			ps193.OverlayValues[92] = d92
			ps193.OverlayValues[126] = d126
			ps193.OverlayValues[127] = d127
			ps193.OverlayValues[128] = d128
			ps193.OverlayValues[129] = d129
			ps193.OverlayValues[130] = d130
			ps193.OverlayValues[132] = d132
			ps193.OverlayValues[134] = d134
			ps193.OverlayValues[135] = d135
			ps193.OverlayValues[138] = d138
			ps193.OverlayValues[177] = d177
			ps193.OverlayValues[178] = d178
			ps193.OverlayValues[179] = d179
			ps193.OverlayValues[180] = d180
			ps193.OverlayValues[181] = d181
			ps193.OverlayValues[182] = d182
			ps193.OverlayValues[183] = d183
			ps193.OverlayValues[184] = d184
			ps193.OverlayValues[186] = d186
			ps193.OverlayValues[187] = d187
			ps193.OverlayValues[188] = d188
			ps193.OverlayValues[189] = d189
			ps193.OverlayValues[192] = d192
			ps194 := PhiState{General: true}
			ps194.OverlayValues = make([]JITValueDesc, 193)
			ps194.OverlayValues[0] = d0
			ps194.OverlayValues[1] = d1
			ps194.OverlayValues[2] = d2
			ps194.OverlayValues[3] = d3
			ps194.OverlayValues[4] = d4
			ps194.OverlayValues[5] = d5
			ps194.OverlayValues[17] = d17
			ps194.OverlayValues[18] = d18
			ps194.OverlayValues[19] = d19
			ps194.OverlayValues[20] = d20
			ps194.OverlayValues[21] = d21
			ps194.OverlayValues[22] = d22
			ps194.OverlayValues[23] = d23
			ps194.OverlayValues[24] = d24
			ps194.OverlayValues[25] = d25
			ps194.OverlayValues[27] = d27
			ps194.OverlayValues[29] = d29
			ps194.OverlayValues[30] = d30
			ps194.OverlayValues[33] = d33
			ps194.OverlayValues[55] = d55
			ps194.OverlayValues[56] = d56
			ps194.OverlayValues[57] = d57
			ps194.OverlayValues[58] = d58
			ps194.OverlayValues[61] = d61
			ps194.OverlayValues[89] = d89
			ps194.OverlayValues[90] = d90
			ps194.OverlayValues[91] = d91
			ps194.OverlayValues[92] = d92
			ps194.OverlayValues[126] = d126
			ps194.OverlayValues[127] = d127
			ps194.OverlayValues[128] = d128
			ps194.OverlayValues[129] = d129
			ps194.OverlayValues[130] = d130
			ps194.OverlayValues[132] = d132
			ps194.OverlayValues[134] = d134
			ps194.OverlayValues[135] = d135
			ps194.OverlayValues[138] = d138
			ps194.OverlayValues[177] = d177
			ps194.OverlayValues[178] = d178
			ps194.OverlayValues[179] = d179
			ps194.OverlayValues[180] = d180
			ps194.OverlayValues[181] = d181
			ps194.OverlayValues[182] = d182
			ps194.OverlayValues[183] = d183
			ps194.OverlayValues[184] = d184
			ps194.OverlayValues[186] = d186
			ps194.OverlayValues[187] = d187
			ps194.OverlayValues[188] = d188
			ps194.OverlayValues[189] = d189
			ps194.OverlayValues[192] = d192
			snap195 := d0
			snap196 := d1
			snap197 := d2
			snap198 := d3
			snap199 := d4
			snap200 := d5
			snap201 := d17
			snap202 := d18
			snap203 := d19
			snap204 := d20
			snap205 := d21
			snap206 := d22
			snap207 := d23
			snap208 := d24
			snap209 := d25
			snap210 := d27
			snap211 := d29
			snap212 := d30
			snap213 := d33
			snap214 := d55
			snap215 := d56
			snap216 := d57
			snap217 := d58
			snap218 := d61
			snap219 := d89
			snap220 := d90
			snap221 := d91
			snap222 := d92
			snap223 := d126
			snap224 := d127
			snap225 := d128
			snap226 := d129
			snap227 := d130
			snap228 := d132
			snap229 := d134
			snap230 := d135
			snap231 := d138
			snap232 := d177
			snap233 := d178
			snap234 := d179
			snap235 := d180
			snap236 := d181
			snap237 := d182
			snap238 := d183
			snap239 := d184
			snap240 := d186
			snap241 := d187
			snap242 := d188
			snap243 := d189
			snap244 := d192
			alloc245 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps194)
			}
			ctx.RestoreAllocState(alloc245)
			d0 = snap195
			d1 = snap196
			d2 = snap197
			d3 = snap198
			d4 = snap199
			d5 = snap200
			d17 = snap201
			d18 = snap202
			d19 = snap203
			d20 = snap204
			d21 = snap205
			d22 = snap206
			d23 = snap207
			d24 = snap208
			d25 = snap209
			d27 = snap210
			d29 = snap211
			d30 = snap212
			d33 = snap213
			d55 = snap214
			d56 = snap215
			d57 = snap216
			d58 = snap217
			d61 = snap218
			d89 = snap219
			d90 = snap220
			d91 = snap221
			d92 = snap222
			d126 = snap223
			d127 = snap224
			d128 = snap225
			d129 = snap226
			d130 = snap227
			d132 = snap228
			d134 = snap229
			d135 = snap230
			d138 = snap231
			d177 = snap232
			d178 = snap233
			d179 = snap234
			d180 = snap235
			d181 = snap236
			d182 = snap237
			d183 = snap238
			d184 = snap239
			d186 = snap240
			d187 = snap241
			d188 = snap242
			d189 = snap243
			d192 = snap244
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps193)
			}
			return result
			ctx.FreeDesc(&d188)
			return result
			}
			bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[11].VisitCount >= 2 {
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
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
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
				d138 = ps.OverlayValues[138]
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
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
				d192 = ps.OverlayValues[192]
			}
			ctx.ReclaimUntrackedRegs()
			d246 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d246, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[12].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
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
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
				d132 = ps.OverlayValues[132]
			}
			if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
				d134 = ps.OverlayValues[134]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
				d138 = ps.OverlayValues[138]
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
			if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
				d192 = ps.OverlayValues[192]
			}
			if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
				d246 = ps.OverlayValues[246]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d247 JITValueDesc
			if d0.Loc == LocImm && d1.Loc == LocImm {
				d247 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d1.Imm.Int())}
			} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
				r18 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r18, d0.Reg)
				d247 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
				ctx.BindReg(r18, &d247)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d247 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d247)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d1.Reg)
				d247 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d247)
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d247 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d247)
			} else {
				r19 := ctx.AllocRegExcept(d0.Reg, d1.Reg)
				ctx.EmitMovRegReg(r19, d0.Reg)
				ctx.EmitAddInt64(r19, d1.Reg)
				d247 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
				ctx.BindReg(r19, &d247)
			}
			if d247.Loc == LocReg && d0.Loc == LocReg && d247.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d247)
			var d249 JITValueDesc
			if d247.Loc == LocImm && d0.Loc == LocImm {
				d249 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d247.Imm.Int() - d0.Imm.Int())}
			} else {
				r20 := ctx.AllocReg()
				if d247.Loc == LocImm {
					ctx.EmitMovRegImm64(r20, uint64(d247.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r20, d247.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitSubInt64(r20, RegR11)
				} else {
					ctx.EmitSubInt64(r20, d0.Reg)
				}
				d249 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
				ctx.BindReg(r20, &d249)
			}
			var d250 JITValueDesc
			if d18.Loc == LocImm && d0.Loc == LocImm {
				d250 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d18.Imm.Int() + d0.Imm.Int())}
			} else {
				r21 := ctx.AllocReg()
				if d18.Loc == LocImm {
					ctx.EmitMovRegImm64(r21, uint64(d18.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r21, d18.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitAddInt64(r21, RegR11)
				} else {
					ctx.EmitAddInt64(r21, d0.Reg)
				}
				d250 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
				ctx.BindReg(r21, &d250)
			}
			var d251 JITValueDesc
			r22 := ctx.AllocReg()
			r23 := ctx.AllocReg()
			if d250.Loc == LocImm {
				ctx.EmitMovRegImm64(r22, uint64(d250.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r22, d250.Reg)
				ctx.FreeReg(d250.Reg)
			}
			if d249.Loc == LocImm {
				ctx.EmitMovRegImm64(r23, uint64(d249.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r23, d249.Reg)
				ctx.FreeReg(d249.Reg)
			}
			d251 = JITValueDesc{Loc: LocRegPair, Reg: r22, Reg2: r23}
			ctx.BindReg(r22, &d251)
			ctx.BindReg(r23, &d251)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d247)
			d252 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d251}, 2)
			ctx.EmitMovPairToResult(&d252, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned253 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned253 = append(argPinned253, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned253 = append(argPinned253, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned253 = append(argPinned253, ai.Reg2)
					}
				}
			}
			ps254 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps254)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(32))
			ctx.EmitAddRSP32(int32(32))
			for _, r := range argPinned253 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"simplify", "turns a stringable input value in the easiest-most value (e.g. turn strings into numbers if they are numeric",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to simplify", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			// turn string to number or so
			return Simplify(String(a[0]))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Simplify arg0)")
			}
			d3 = ctx.EmitGoCallScalar(GoFuncAddr(Simplify), []JITValueDesc{d1}, 2)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocRegPair {
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
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"strlen", "returns the length of a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "int",
		func(a ...Scmer) Scmer {
			return NewInt(int64(len(String(a[0]))))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d1.Imm.String())))}
			} else {
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocRegPair {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
					ctx.BindReg(d1.Reg2, &d3)
					ctx.BindReg(d1.Reg2, &d3)
				} else if d1.Loc == LocReg {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
					ctx.BindReg(d1.Reg, &d3)
					ctx.BindReg(d1.Reg, &d3)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.EmitMakeInt(result, d3)
			} else {
				ctx.EmitMakeInt(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagInt
			return result
			return result
			}
			argPinned5 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned5 = append(argPinned5, ai.Reg2)
					}
				}
			}
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
			for _, r := range argPinned5 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"strlike", "matches the string against a wildcard pattern (SQL compliant)",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
			DeclarationParameter{"pattern", "string", "pattern with % and _ in them", nil},
			DeclarationParameter{"collation", "string", "collation in which to compare them", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			value := String(a[0])
			pattern := String(a[1])
			collation := "utf8mb4_general_ci"
			if len(a) > 2 {
				collation = strings.ToLower(String(a[2]))
			}
			if strings.Contains(collation, "_ci") {
				value = strings.ToLower(value)
				pattern = strings.ToLower(pattern)
			}
			return NewBool(StrLike(value, pattern))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			var d5 JITValueDesc
			_ = d5
			var d6 JITValueDesc
			_ = d6
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
			var d14 JITValueDesc
			_ = d14
			var d17 JITValueDesc
			_ = d17
			var d33 JITValueDesc
			_ = d33
			var d34 JITValueDesc
			_ = d34
			var d35 JITValueDesc
			_ = d35
			var d36 JITValueDesc
			_ = d36
			var d37 JITValueDesc
			_ = d37
			var d39 JITValueDesc
			_ = d39
			var d40 JITValueDesc
			_ = d40
			var d41 JITValueDesc
			_ = d41
			var d42 JITValueDesc
			_ = d42
			var d43 JITValueDesc
			_ = d43
			var d45 JITValueDesc
			_ = d45
			var d46 JITValueDesc
			_ = d46
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
			var d55 JITValueDesc
			_ = d55
			var d56 JITValueDesc
			_ = d56
			var d91 JITValueDesc
			_ = d91
			var d92 JITValueDesc
			_ = d92
			var d93 JITValueDesc
			_ = d93
			var d94 JITValueDesc
			_ = d94
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(16)}
			d2 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(32)}
			var bbs [5]BBDescriptor
			bbs[2].PhiBase = int32(0)
			bbs[2].PhiCount = uint16(1)
			bbs[4].PhiBase = int32(16)
			bbs[4].PhiCount = uint16(2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			ctx.EnsureDesc(&d5)
			if d5.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d5.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d5)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d5.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d5 = tmpPair
			} else if d5.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
				switch d5.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d5)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d5)
				d5 = tmpPair
			} else if d5.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d5.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d5.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d5 = tmpPair
			}
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d5}, 2)
			ctx.FreeDesc(&d3)
			d6 = args[1]
			d6.ID = 0
			d8 = d6
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d8.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d8)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d8)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d8)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d8.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d8 = tmpPair
			} else if d8.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d8.Reg), Reg2: ctx.AllocRegExcept(d8.Reg)}
				switch d8.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d8)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d8)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d8)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d8)
				d8 = tmpPair
			} else if d8.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d8.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d8.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d8 = tmpPair
			}
			if d8.Loc != LocRegPair && d8.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d7 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d8}, 2)
			ctx.FreeDesc(&d6)
			d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d9)
			var d10 JITValueDesc
			if d9.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() > 2)}
			} else {
				r1 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d9.Reg, 2)
				ctx.EmitSetcc(r1, CcG)
				d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d10)
			}
			ctx.FreeDesc(&d9)
			d11 = d10
			ctx.EnsureDesc(&d11)
			if d11.Loc != LocImm && d11.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d11.Loc == LocImm {
				if d11.Imm.Bool() {
			ps12 := PhiState{General: ps.General}
			ps12.OverlayValues = make([]JITValueDesc, 12)
			ps12.OverlayValues[0] = d0
			ps12.OverlayValues[1] = d1
			ps12.OverlayValues[2] = d2
			ps12.OverlayValues[3] = d3
			ps12.OverlayValues[4] = d4
			ps12.OverlayValues[5] = d5
			ps12.OverlayValues[6] = d6
			ps12.OverlayValues[7] = d7
			ps12.OverlayValues[8] = d8
			ps12.OverlayValues[9] = d9
			ps12.OverlayValues[10] = d10
			ps12.OverlayValues[11] = d11
					return bbs[1].RenderPS(ps12)
				}
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[2].PhiBase)+int32(0))
			ps13 := PhiState{General: ps.General}
			ps13.OverlayValues = make([]JITValueDesc, 12)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[6] = d6
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			ps13.PhiValues = make([]JITValueDesc, 1)
			d14 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
			ps13.PhiValues[0] = d14
				return bbs[2].RenderPS(ps13)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl6 := ctx.ReserveLabel()
			lbl7 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d11.Reg, 0)
			ctx.EmitJcc(CcNE, lbl6)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl6)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl7)
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[2].PhiBase)+int32(0))
			ctx.EmitJmp(lbl3)
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 15)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[14] = d14
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 15)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[5] = d5
			ps16.OverlayValues[6] = d6
			ps16.OverlayValues[7] = d7
			ps16.OverlayValues[8] = d8
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[14] = d14
			ps16.PhiValues = make([]JITValueDesc, 1)
			d17 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
			ps16.PhiValues[0] = d17
			snap18 := d0
			snap19 := d1
			snap20 := d2
			snap21 := d3
			snap22 := d4
			snap23 := d5
			snap24 := d6
			snap25 := d7
			snap26 := d8
			snap27 := d9
			snap28 := d10
			snap29 := d11
			snap30 := d14
			snap31 := d17
			alloc32 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc32)
			d0 = snap18
			d1 = snap19
			d2 = snap20
			d3 = snap21
			d4 = snap22
			d5 = snap23
			d6 = snap24
			d7 = snap25
			d8 = snap26
			d9 = snap27
			d10 = snap28
			d11 = snap29
			d14 = snap30
			d17 = snap31
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps15)
			}
			return result
			ctx.FreeDesc(&d10)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
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
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(32)}
			d0 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
				d17 = ps.OverlayValues[17]
			}
			ctx.ReclaimUntrackedRegs()
			d33 = args[2]
			d33.ID = 0
			d35 = d33
			ctx.EnsureDesc(&d35)
			if d35.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d35.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d35)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d35)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d35)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d35.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d35 = tmpPair
			} else if d35.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d35.Reg), Reg2: ctx.AllocRegExcept(d35.Reg)}
				switch d35.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d35)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d35)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d35)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d35)
				d35 = tmpPair
			} else if d35.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d35.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d35.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d35 = tmpPair
			}
			if d35.Loc != LocRegPair && d35.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d34 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d35}, 2)
			ctx.FreeDesc(&d33)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d34)
			if d34.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d34.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d34.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d34)
				} else if d34.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d34)
				} else if d34.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d34)
				} else if d34.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d34.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d34 = tmpPair
			} else if d34.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d34.Type, Reg: ctx.AllocRegExcept(d34.Reg), Reg2: ctx.AllocRegExcept(d34.Reg)}
				switch d34.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d34)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d34)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d34)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d34)
				d34 = tmpPair
			}
			if d34.Loc != LocRegPair && d34.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
			}
			d36 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d34}, 2)
			ctx.EnsureDesc(&d36)
			if d36.Loc == LocReg {
				ctx.ProtectReg(d36.Reg)
			} else if d36.Loc == LocRegPair {
				ctx.ProtectReg(d36.Reg)
				ctx.ProtectReg(d36.Reg2)
			}
			d37 = d36
			if d37.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			if d37.Loc == LocRegPair || d37.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d37, int32(bbs[2].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d37, int32(bbs[2].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
			}
			if d36.Loc == LocReg {
				ctx.UnprotectReg(d36.Reg)
			} else if d36.Loc == LocRegPair {
				ctx.UnprotectReg(d36.Reg)
				ctx.UnprotectReg(d36.Reg2)
			}
			ps38 := PhiState{General: ps.General}
			ps38.OverlayValues = make([]JITValueDesc, 38)
			ps38.OverlayValues[0] = d0
			ps38.OverlayValues[1] = d1
			ps38.OverlayValues[2] = d2
			ps38.OverlayValues[3] = d3
			ps38.OverlayValues[4] = d4
			ps38.OverlayValues[5] = d5
			ps38.OverlayValues[6] = d6
			ps38.OverlayValues[7] = d7
			ps38.OverlayValues[8] = d8
			ps38.OverlayValues[9] = d9
			ps38.OverlayValues[10] = d10
			ps38.OverlayValues[11] = d11
			ps38.OverlayValues[14] = d14
			ps38.OverlayValues[17] = d17
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			ps38.OverlayValues[35] = d35
			ps38.OverlayValues[36] = d36
			ps38.OverlayValues[37] = d37
			ps38.PhiValues = make([]JITValueDesc, 1)
			d39 = d36
			ps38.PhiValues[0] = d39
			if ps38.General && bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps38)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d40 := ps.PhiValues[0]
					ctx.EnsureDesc(&d40)
					ctx.EmitStoreScmerToStack(d40, int32(bbs[2].PhiBase)+int32(0))
				}
				if bbs[2].VisitCount >= 2 {
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
			d2 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(32)}
			d0 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.Contains arg0)")
			}
			d41 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("_ci")}
			ctx.EnsureDesc(&d41)
			if d41.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d41.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d41)
				} else if d41.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d41)
				} else if d41.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d41)
				} else if d41.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d41.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d41 = tmpPair
			} else if d41.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocRegExcept(d41.Reg), Reg2: ctx.AllocRegExcept(d41.Reg)}
				switch d41.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d41)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d41)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d41)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d41)
				d41 = tmpPair
			}
			if d41.Loc != LocRegPair && d41.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.Contains arg1)")
			}
			d42 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Contains), []JITValueDesc{d0, d41}, 1)
			ctx.FreeDesc(&d0)
			d43 = d42
			ctx.EnsureDesc(&d43)
			if d43.Loc != LocImm && d43.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d43.Loc == LocImm {
				if d43.Imm.Bool() {
			ps44 := PhiState{General: ps.General}
			ps44.OverlayValues = make([]JITValueDesc, 44)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[3] = d3
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[6] = d6
			ps44.OverlayValues[7] = d7
			ps44.OverlayValues[8] = d8
			ps44.OverlayValues[9] = d9
			ps44.OverlayValues[10] = d10
			ps44.OverlayValues[11] = d11
			ps44.OverlayValues[14] = d14
			ps44.OverlayValues[17] = d17
			ps44.OverlayValues[33] = d33
			ps44.OverlayValues[34] = d34
			ps44.OverlayValues[35] = d35
			ps44.OverlayValues[36] = d36
			ps44.OverlayValues[37] = d37
			ps44.OverlayValues[39] = d39
			ps44.OverlayValues[40] = d40
			ps44.OverlayValues[41] = d41
			ps44.OverlayValues[42] = d42
			ps44.OverlayValues[43] = d43
					return bbs[3].RenderPS(ps44)
				}
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocReg {
				ctx.ProtectReg(d4.Reg)
			} else if d4.Loc == LocRegPair {
				ctx.ProtectReg(d4.Reg)
				ctx.ProtectReg(d4.Reg2)
			}
			ctx.EnsureDesc(&d7)
			if d7.Loc == LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d45 = d4
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			if d45.Loc == LocRegPair || d45.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d45, int32(bbs[4].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d45, int32(bbs[4].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
			}
			d46 = d7
			if d46.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d46)
			if d46.Loc == LocRegPair || d46.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d46, int32(bbs[4].PhiBase)+int32(16))
			} else {
				ctx.EmitStoreToStack(d46, int32(bbs[4].PhiBase)+int32(16))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(16))+8)
			}
			if d4.Loc == LocReg {
				ctx.UnprotectReg(d4.Reg)
			} else if d4.Loc == LocRegPair {
				ctx.UnprotectReg(d4.Reg)
				ctx.UnprotectReg(d4.Reg2)
			}
			if d7.Loc == LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
			ps47 := PhiState{General: ps.General}
			ps47.OverlayValues = make([]JITValueDesc, 47)
			ps47.OverlayValues[0] = d0
			ps47.OverlayValues[1] = d1
			ps47.OverlayValues[2] = d2
			ps47.OverlayValues[3] = d3
			ps47.OverlayValues[4] = d4
			ps47.OverlayValues[5] = d5
			ps47.OverlayValues[6] = d6
			ps47.OverlayValues[7] = d7
			ps47.OverlayValues[8] = d8
			ps47.OverlayValues[9] = d9
			ps47.OverlayValues[10] = d10
			ps47.OverlayValues[11] = d11
			ps47.OverlayValues[14] = d14
			ps47.OverlayValues[17] = d17
			ps47.OverlayValues[33] = d33
			ps47.OverlayValues[34] = d34
			ps47.OverlayValues[35] = d35
			ps47.OverlayValues[36] = d36
			ps47.OverlayValues[37] = d37
			ps47.OverlayValues[39] = d39
			ps47.OverlayValues[40] = d40
			ps47.OverlayValues[41] = d41
			ps47.OverlayValues[42] = d42
			ps47.OverlayValues[43] = d43
			ps47.OverlayValues[45] = d45
			ps47.OverlayValues[46] = d46
			ps47.PhiValues = make([]JITValueDesc, 2)
			d48 = d4
			ps47.PhiValues[0] = d48
			d49 = d7
			ps47.PhiValues[1] = d49
				return bbs[4].RenderPS(ps47)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d50 := ps.PhiValues[0]
					ctx.EnsureDesc(&d50)
					ctx.EmitStoreScmerToStack(d50, int32(bbs[2].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d43.Reg, 0)
			ctx.EmitJcc(CcNE, lbl8)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocReg {
				ctx.ProtectReg(d4.Reg)
			} else if d4.Loc == LocRegPair {
				ctx.ProtectReg(d4.Reg)
				ctx.ProtectReg(d4.Reg2)
			}
			ctx.EnsureDesc(&d7)
			if d7.Loc == LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d51 = d4
			if d51.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d51)
			if d51.Loc == LocRegPair || d51.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d51, int32(bbs[4].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d51, int32(bbs[4].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
			}
			d52 = d7
			if d52.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d52)
			if d52.Loc == LocRegPair || d52.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d52, int32(bbs[4].PhiBase)+int32(16))
			} else {
				ctx.EmitStoreToStack(d52, int32(bbs[4].PhiBase)+int32(16))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(16))+8)
			}
			if d4.Loc == LocReg {
				ctx.UnprotectReg(d4.Reg)
			} else if d4.Loc == LocRegPair {
				ctx.UnprotectReg(d4.Reg)
				ctx.UnprotectReg(d4.Reg2)
			}
			if d7.Loc == LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
			ctx.EmitJmp(lbl5)
			ps53 := PhiState{General: true}
			ps53.OverlayValues = make([]JITValueDesc, 53)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
			ps53.OverlayValues[2] = d2
			ps53.OverlayValues[3] = d3
			ps53.OverlayValues[4] = d4
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[6] = d6
			ps53.OverlayValues[7] = d7
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[9] = d9
			ps53.OverlayValues[10] = d10
			ps53.OverlayValues[11] = d11
			ps53.OverlayValues[14] = d14
			ps53.OverlayValues[17] = d17
			ps53.OverlayValues[33] = d33
			ps53.OverlayValues[34] = d34
			ps53.OverlayValues[35] = d35
			ps53.OverlayValues[36] = d36
			ps53.OverlayValues[37] = d37
			ps53.OverlayValues[39] = d39
			ps53.OverlayValues[40] = d40
			ps53.OverlayValues[41] = d41
			ps53.OverlayValues[42] = d42
			ps53.OverlayValues[43] = d43
			ps53.OverlayValues[45] = d45
			ps53.OverlayValues[46] = d46
			ps53.OverlayValues[48] = d48
			ps53.OverlayValues[49] = d49
			ps53.OverlayValues[50] = d50
			ps53.OverlayValues[51] = d51
			ps53.OverlayValues[52] = d52
			ps54 := PhiState{General: true}
			ps54.OverlayValues = make([]JITValueDesc, 53)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[2] = d2
			ps54.OverlayValues[3] = d3
			ps54.OverlayValues[4] = d4
			ps54.OverlayValues[5] = d5
			ps54.OverlayValues[6] = d6
			ps54.OverlayValues[7] = d7
			ps54.OverlayValues[8] = d8
			ps54.OverlayValues[9] = d9
			ps54.OverlayValues[10] = d10
			ps54.OverlayValues[11] = d11
			ps54.OverlayValues[14] = d14
			ps54.OverlayValues[17] = d17
			ps54.OverlayValues[33] = d33
			ps54.OverlayValues[34] = d34
			ps54.OverlayValues[35] = d35
			ps54.OverlayValues[36] = d36
			ps54.OverlayValues[37] = d37
			ps54.OverlayValues[39] = d39
			ps54.OverlayValues[40] = d40
			ps54.OverlayValues[41] = d41
			ps54.OverlayValues[42] = d42
			ps54.OverlayValues[43] = d43
			ps54.OverlayValues[45] = d45
			ps54.OverlayValues[46] = d46
			ps54.OverlayValues[48] = d48
			ps54.OverlayValues[49] = d49
			ps54.OverlayValues[50] = d50
			ps54.OverlayValues[51] = d51
			ps54.OverlayValues[52] = d52
			ps54.PhiValues = make([]JITValueDesc, 2)
			d55 = d4
			ps54.PhiValues[0] = d55
			d56 = d7
			ps54.PhiValues[1] = d56
			snap57 := d0
			snap58 := d1
			snap59 := d2
			snap60 := d3
			snap61 := d4
			snap62 := d5
			snap63 := d6
			snap64 := d7
			snap65 := d8
			snap66 := d9
			snap67 := d10
			snap68 := d11
			snap69 := d14
			snap70 := d17
			snap71 := d33
			snap72 := d34
			snap73 := d35
			snap74 := d36
			snap75 := d37
			snap76 := d39
			snap77 := d40
			snap78 := d41
			snap79 := d42
			snap80 := d43
			snap81 := d45
			snap82 := d46
			snap83 := d48
			snap84 := d49
			snap85 := d50
			snap86 := d51
			snap87 := d52
			snap88 := d55
			snap89 := d56
			alloc90 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps54)
			}
			ctx.RestoreAllocState(alloc90)
			d0 = snap57
			d1 = snap58
			d2 = snap59
			d3 = snap60
			d4 = snap61
			d5 = snap62
			d6 = snap63
			d7 = snap64
			d8 = snap65
			d9 = snap66
			d10 = snap67
			d11 = snap68
			d14 = snap69
			d17 = snap70
			d33 = snap71
			d34 = snap72
			d35 = snap73
			d36 = snap74
			d37 = snap75
			d39 = snap76
			d40 = snap77
			d41 = snap78
			d42 = snap79
			d43 = snap80
			d45 = snap81
			d46 = snap82
			d48 = snap83
			d49 = snap84
			d50 = snap85
			d51 = snap86
			d52 = snap87
			d55 = snap88
			d56 = snap89
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps53)
			}
			return result
			ctx.FreeDesc(&d42)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
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
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d4.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d4.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d4 = tmpPair
			} else if d4.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
				switch d4.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d4)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d4)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d4)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d4)
				d4 = tmpPair
			}
			if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
			}
			d91 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d4}, 2)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d7)
			if d7.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d7.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d7.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d7)
				} else if d7.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d7)
				} else if d7.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d7)
				} else if d7.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d7.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d7 = tmpPair
			} else if d7.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d7.Type, Reg: ctx.AllocRegExcept(d7.Reg), Reg2: ctx.AllocRegExcept(d7.Reg)}
				switch d7.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d7)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d7)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d7)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d7)
				d7 = tmpPair
			}
			if d7.Loc != LocRegPair && d7.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
			}
			d92 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d7}, 2)
			ctx.EnsureDesc(&d91)
			if d91.Loc == LocReg {
				ctx.ProtectReg(d91.Reg)
			} else if d91.Loc == LocRegPair {
				ctx.ProtectReg(d91.Reg)
				ctx.ProtectReg(d91.Reg2)
			}
			ctx.EnsureDesc(&d92)
			if d92.Loc == LocReg {
				ctx.ProtectReg(d92.Reg)
			} else if d92.Loc == LocRegPair {
				ctx.ProtectReg(d92.Reg)
				ctx.ProtectReg(d92.Reg2)
			}
			d93 = d91
			if d93.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d93)
			if d93.Loc == LocRegPair || d93.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d93, int32(bbs[4].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d93, int32(bbs[4].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
			}
			d94 = d92
			if d94.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d94)
			if d94.Loc == LocRegPair || d94.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d94, int32(bbs[4].PhiBase)+int32(16))
			} else {
				ctx.EmitStoreToStack(d94, int32(bbs[4].PhiBase)+int32(16))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(16))+8)
			}
			if d91.Loc == LocReg {
				ctx.UnprotectReg(d91.Reg)
			} else if d91.Loc == LocRegPair {
				ctx.UnprotectReg(d91.Reg)
				ctx.UnprotectReg(d91.Reg2)
			}
			if d92.Loc == LocReg {
				ctx.UnprotectReg(d92.Reg)
			} else if d92.Loc == LocRegPair {
				ctx.UnprotectReg(d92.Reg)
				ctx.UnprotectReg(d92.Reg2)
			}
			ps95 := PhiState{General: ps.General}
			ps95.OverlayValues = make([]JITValueDesc, 95)
			ps95.OverlayValues[0] = d0
			ps95.OverlayValues[1] = d1
			ps95.OverlayValues[2] = d2
			ps95.OverlayValues[3] = d3
			ps95.OverlayValues[4] = d4
			ps95.OverlayValues[5] = d5
			ps95.OverlayValues[6] = d6
			ps95.OverlayValues[7] = d7
			ps95.OverlayValues[8] = d8
			ps95.OverlayValues[9] = d9
			ps95.OverlayValues[10] = d10
			ps95.OverlayValues[11] = d11
			ps95.OverlayValues[14] = d14
			ps95.OverlayValues[17] = d17
			ps95.OverlayValues[33] = d33
			ps95.OverlayValues[34] = d34
			ps95.OverlayValues[35] = d35
			ps95.OverlayValues[36] = d36
			ps95.OverlayValues[37] = d37
			ps95.OverlayValues[39] = d39
			ps95.OverlayValues[40] = d40
			ps95.OverlayValues[41] = d41
			ps95.OverlayValues[42] = d42
			ps95.OverlayValues[43] = d43
			ps95.OverlayValues[45] = d45
			ps95.OverlayValues[46] = d46
			ps95.OverlayValues[48] = d48
			ps95.OverlayValues[49] = d49
			ps95.OverlayValues[50] = d50
			ps95.OverlayValues[51] = d51
			ps95.OverlayValues[52] = d52
			ps95.OverlayValues[55] = d55
			ps95.OverlayValues[56] = d56
			ps95.OverlayValues[91] = d91
			ps95.OverlayValues[92] = d92
			ps95.OverlayValues[93] = d93
			ps95.OverlayValues[94] = d94
			ps95.PhiValues = make([]JITValueDesc, 2)
			d96 = d91
			ps95.PhiValues[0] = d96
			d97 = d92
			ps95.PhiValues[1] = d97
			if ps95.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps95)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d98 := ps.PhiValues[0]
					ctx.EnsureDesc(&d98)
					ctx.EmitStoreScmerToStack(d98, int32(bbs[4].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d99 := ps.PhiValues[1]
					ctx.EnsureDesc(&d99)
					ctx.EmitStoreScmerToStack(d99, int32(bbs[4].PhiBase)+int32(16))
				}
				if bbs[4].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
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
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
				d91 = ps.OverlayValues[91]
			}
			if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
				d92 = ps.OverlayValues[92]
			}
			if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
				d93 = ps.OverlayValues[93]
			}
			if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
				d94 = ps.OverlayValues[94]
			}
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
				d96 = ps.OverlayValues[96]
			}
			if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
				d97 = ps.OverlayValues[97]
			}
			if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
				d98 = ps.OverlayValues[98]
			}
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d2 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (StrLike arg0)")
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d2.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d2)
				} else if d2.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d2)
				} else if d2.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d2)
				} else if d2.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (StrLike arg1)")
			}
			d100 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d1, d2}, 1)
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d100)
			ctx.EmitMakeBool(result, d100)
			if d100.Loc == LocReg { ctx.FreeReg(d100.Reg) }
			result.Type = tagBool
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned101 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned101 = append(argPinned101, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned101 = append(argPinned101, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned101 = append(argPinned101, ai.Reg2)
					}
				}
			}
			ps102 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps102)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(48))
			ctx.EmitAddRSP32(int32(48))
			for _, r := range argPinned101 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"strlike_cs", "matches the string against a wildcard pattern (case-sensitive)",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
			DeclarationParameter{"pattern", "string", "pattern with % and _ in them", nil},
			DeclarationParameter{"collation", "string", "ignored (present for parser compatibility)", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(StrLike(String(a[0]), String(a[1])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
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
			var d6 JITValueDesc
			_ = d6
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			d3 = args[1]
			d3.ID = 0
			d5 = d3
			ctx.EnsureDesc(&d5)
			if d5.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d5.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d5)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d5.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d5 = tmpPair
			} else if d5.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
				switch d5.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d5)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d5)
				d5 = tmpPair
			} else if d5.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d5.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d5.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d5 = tmpPair
			}
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d5}, 2)
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (StrLike arg0)")
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d4.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d4.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d4 = tmpPair
			} else if d4.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
				switch d4.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d4)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d4)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d4)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d4)
				d4 = tmpPair
			}
			if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (StrLike arg1)")
			}
			d6 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d1, d4}, 1)
			ctx.EnsureDesc(&d6)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d6.Loc == LocImm {
				ctx.EmitMakeBool(result, d6)
			} else {
				ctx.EmitMakeBool(result, d6)
				ctx.FreeReg(d6.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			argPinned7 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned7 = append(argPinned7, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned7 = append(argPinned7, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned7 = append(argPinned7, ai.Reg2)
					}
				}
			}
			ps8 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps8)
			for _, r := range argPinned7 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"toLower", "turns a string into lower case",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(strings.ToLower(String(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
			}
			d3 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d1}, 2)
			ctx.ResolveFixups()
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned5 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned5 = append(argPinned5, ai.Reg2)
					}
				}
			}
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
			for _, r := range argPinned5 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
	})
	Declare(&Globalenv, &Declaration{
		"toUpper", "turns a string into upper case",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(strings.ToUpper(String(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
			}
			d3 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d1}, 2)
			ctx.ResolveFixups()
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned5 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned5 = append(argPinned5, ai.Reg2)
					}
				}
			}
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
			for _, r := range argPinned5 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
	})
	Declare(&Globalenv, &Declaration{
		"replace", "replaces all occurances in a string with another string",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"s", "string", "input string", nil},
			DeclarationParameter{"find", "string", "search string", nil},
			DeclarationParameter{"replace", "string", "replace string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(strings.ReplaceAll(String(a[0]), String(a[1]), String(a[2])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
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
			var d6 JITValueDesc
			_ = d6
			var d7 JITValueDesc
			_ = d7
			var d8 JITValueDesc
			_ = d8
			var d9 JITValueDesc
			_ = d9
			var d10 JITValueDesc
			_ = d10
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			d3 = args[1]
			d3.ID = 0
			d5 = d3
			ctx.EnsureDesc(&d5)
			if d5.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d5.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d5)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d5.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d5 = tmpPair
			} else if d5.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
				switch d5.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d5)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d5)
				d5 = tmpPair
			} else if d5.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d5.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d5.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d5 = tmpPair
			}
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d5}, 2)
			ctx.FreeDesc(&d3)
			d6 = args[2]
			d6.ID = 0
			d8 = d6
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d8.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d8)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d8)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d8)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d8.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d8 = tmpPair
			} else if d8.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d8.Reg), Reg2: ctx.AllocRegExcept(d8.Reg)}
				switch d8.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d8)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d8)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d8)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d8)
				d8 = tmpPair
			} else if d8.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d8.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d8.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d8 = tmpPair
			}
			if d8.Loc != LocRegPair && d8.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d7 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d8}, 2)
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg0)")
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d4.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d4)
				} else if d4.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d4.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d4 = tmpPair
			} else if d4.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
				switch d4.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d4)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d4)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d4)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d4)
				d4 = tmpPair
			}
			if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg1)")
			}
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d7)
			if d7.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d7.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d7.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d7)
				} else if d7.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d7)
				} else if d7.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d7)
				} else if d7.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d7.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d7 = tmpPair
			} else if d7.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d7.Type, Reg: ctx.AllocRegExcept(d7.Reg), Reg2: ctx.AllocRegExcept(d7.Reg)}
				switch d7.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d7)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d7)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d7)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d7)
				d7 = tmpPair
			}
			if d7.Loc != LocRegPair && d7.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg2)")
			}
			d9 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ReplaceAll), []JITValueDesc{d1, d4, d7}, 2)
			ctx.ResolveFixups()
			d10 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d9}, 2)
			if result.Loc == LocAny { return d10 }
			ctx.EmitMovPairToResult(&d10, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned11 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned11 = append(argPinned11, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned11 = append(argPinned11, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned11 = append(argPinned11, ai.Reg2)
					}
				}
			}
			ps12 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps12)
			for _, r := range argPinned11 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */
	})
	Declare(&Globalenv, &Declaration{
		"strtrim", "trims whitespace from both ends of a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(strings.TrimSpace(String(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
			}
			d3 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d1}, 2)
			ctx.ResolveFixups()
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned5 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned5 = append(argPinned5, ai.Reg2)
					}
				}
			}
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
			for _, r := range argPinned5 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */
	})
	Declare(&Globalenv, &Declaration{
		"strltrim", "trims whitespace from the left of a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(strings.TrimLeft(String(a[0]), " \t\n\r"))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
			}
			d3 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d3.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d3 = tmpPair
			} else if d3.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
				switch d3.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d3)
				d3 = tmpPair
			}
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
			}
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d1, d3}, 2)
			ctx.ResolveFixups()
			d5 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
			if result.Loc == LocAny { return d5 }
			ctx.EmitMovPairToResult(&d5, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned6 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned6 = append(argPinned6, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned6 = append(argPinned6, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned6 = append(argPinned6, ai.Reg2)
					}
				}
			}
			ps7 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps7)
			for _, r := range argPinned6 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
	})
	Declare(&Globalenv, &Declaration{
		"strrtrim", "trims whitespace from the right of a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(strings.TrimRight(String(a[0]), " \t\n\r"))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
			}
			d3 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d3.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d3 = tmpPair
			} else if d3.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
				switch d3.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d3)
				d3 = tmpPair
			}
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
			}
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d1, d3}, 2)
			ctx.ResolveFixups()
			d5 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
			if result.Loc == LocAny { return d5 }
			ctx.EmitMovPairToResult(&d5, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned6 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned6 = append(argPinned6, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned6 = append(argPinned6, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned6 = append(argPinned6, ai.Reg2)
					}
				}
			}
			ps7 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps7)
			for _, r := range argPinned6 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
	})
	// SQL-level NULL-safe wrappers for TRIM/LTRIM/RTRIM
	Declare(&Globalenv, &Declaration{
		"sql_trim", "SQL TRIM(): NULL-safe trim of whitespace from both ends",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewString(strings.TrimSpace(String(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d17 JITValueDesc
			_ = d17
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [3]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			ps4 := PhiState{General: ps.General}
			ps4.OverlayValues = make([]JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
					return bbs[1].RenderPS(ps4)
				}
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 4)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
				return bbs[2].RenderPS(ps5)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl4 := ctx.ReserveLabel()
			lbl5 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			ctx.EmitJcc(CcNE, lbl4)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl4)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl5)
			ctx.EmitJmp(lbl3)
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
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
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
				if bbs[1].VisitCount >= 2 {
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			ctx.ReclaimUntrackedRegs()
			d13 = args[0]
			d13.ID = 0
			d15 = d13
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d15.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d15)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d15)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d15)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d15.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d15 = tmpPair
			} else if d15.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d15.Reg), Reg2: ctx.AllocRegExcept(d15.Reg)}
				switch d15.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d15)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d15)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d15)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d15)
				d15 = tmpPair
			} else if d15.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d15.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d15.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d15 = tmpPair
			}
			if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d15}, 2)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d14)
			if d14.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d14.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d14.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d14.Loc != LocRegPair && d14.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
			}
			d16 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d14}, 2)
			d17 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d16}, 2)
			ctx.EmitMovPairToResult(&d17, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned18 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned18 = append(argPinned18, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned18 = append(argPinned18, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned18 = append(argPinned18, ai.Reg2)
					}
				}
			}
			ps19 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps19)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned18 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */
	})
	Declare(&Globalenv, &Declaration{
		"sql_ltrim", "SQL LTRIM(): NULL-safe trim of whitespace from left",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewString(strings.TrimLeft(String(a[0]), " \t\n\r"))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d17 JITValueDesc
			_ = d17
			var d18 JITValueDesc
			_ = d18
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [3]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			ps4 := PhiState{General: ps.General}
			ps4.OverlayValues = make([]JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
					return bbs[1].RenderPS(ps4)
				}
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 4)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
				return bbs[2].RenderPS(ps5)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl4 := ctx.ReserveLabel()
			lbl5 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			ctx.EmitJcc(CcNE, lbl4)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl4)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl5)
			ctx.EmitJmp(lbl3)
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
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
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
				if bbs[1].VisitCount >= 2 {
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			ctx.ReclaimUntrackedRegs()
			d13 = args[0]
			d13.ID = 0
			d15 = d13
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d15.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d15)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d15)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d15)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d15.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d15 = tmpPair
			} else if d15.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d15.Reg), Reg2: ctx.AllocRegExcept(d15.Reg)}
				switch d15.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d15)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d15)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d15)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d15)
				d15 = tmpPair
			} else if d15.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d15.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d15.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d15 = tmpPair
			}
			if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d15}, 2)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d14)
			if d14.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d14.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d14.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d14.Loc != LocRegPair && d14.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
			}
			d16 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d16)
			if d16.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d16.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d16.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d16)
				} else if d16.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d16)
				} else if d16.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d16)
				} else if d16.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d16.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
			}
			d17 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d14, d16}, 2)
			d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d17}, 2)
			ctx.EmitMovPairToResult(&d18, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned19 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned19 = append(argPinned19, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned19 = append(argPinned19, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned19 = append(argPinned19, ai.Reg2)
					}
				}
			}
			ps20 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps20)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned19 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
	})
	Declare(&Globalenv, &Declaration{
		"sql_rtrim", "SQL RTRIM(): NULL-safe trim of whitespace from right",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewString(strings.TrimRight(String(a[0]), " \t\n\r"))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d17 JITValueDesc
			_ = d17
			var d18 JITValueDesc
			_ = d18
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [3]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			ps4 := PhiState{General: ps.General}
			ps4.OverlayValues = make([]JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
					return bbs[1].RenderPS(ps4)
				}
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 4)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
				return bbs[2].RenderPS(ps5)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl4 := ctx.ReserveLabel()
			lbl5 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			ctx.EmitJcc(CcNE, lbl4)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl4)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl5)
			ctx.EmitJmp(lbl3)
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
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
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
				if bbs[1].VisitCount >= 2 {
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			ctx.ReclaimUntrackedRegs()
			d13 = args[0]
			d13.ID = 0
			d15 = d13
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d15.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d15)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d15)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d15)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d15.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d15 = tmpPair
			} else if d15.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d15.Reg), Reg2: ctx.AllocRegExcept(d15.Reg)}
				switch d15.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d15)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d15)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d15)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d15)
				d15 = tmpPair
			} else if d15.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d15.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d15.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d15 = tmpPair
			}
			if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d15}, 2)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d14)
			if d14.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d14.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d14)
				} else if d14.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d14.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d14.Loc != LocRegPair && d14.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
			}
			d16 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d16)
			if d16.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d16.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d16.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d16)
				} else if d16.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d16)
				} else if d16.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d16)
				} else if d16.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d16.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
			}
			d17 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d14, d16}, 2)
			d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d17}, 2)
			ctx.EmitMovPairToResult(&d18, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned19 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned19 = append(argPinned19, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned19 = append(argPinned19, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned19 = append(argPinned19, ai.Reg2)
					}
				}
			}
			ps20 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps20)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned19 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
	})
	Declare(&Globalenv, &Declaration{
		"split", "splits a string using a separator or space",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
			DeclarationParameter{"separator", "string", "(optional) parameter, defaults to \" \"", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			split := " "
			if len(a) > 1 {
				split = String(a[1])
			}
			ar := strings.Split(String(a[0]), split)
			result := make([]Scmer, len(ar))
			for i, v := range ar {
				result[i] = NewString(v)
			}
			return NewSlice(result)
		}, true, false, nil,
		nil /* TODO: MakeSlice: make []string t12 t12 */, /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t2 t2 */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
	})

	Declare(&Globalenv, &Declaration{
		"string_repeat", "repeats a string n times",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to repeat", nil},
			DeclarationParameter{"count", "number", "number of repetitions", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			n := ToInt(a[1])
			if n <= 0 {
				return NewString("")
			}
			return NewString(strings.Repeat(String(a[0]), int(n)))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [5]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			ps4 := PhiState{General: ps.General}
			ps4.OverlayValues = make([]JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
					return bbs[1].RenderPS(ps4)
				}
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 4)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
				return bbs[2].RenderPS(ps5)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl6 := ctx.ReserveLabel()
			lbl7 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			ctx.EmitJcc(CcNE, lbl6)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl6)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl7)
			ctx.EmitJmp(lbl3)
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
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
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
				if bbs[1].VisitCount >= 2 {
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			ctx.ReclaimUntrackedRegs()
			d13 = args[1]
			d13.ID = 0
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d13)
			if d13.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d13.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d13)
				} else if d13.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d13)
				} else if d13.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d13)
				} else if d13.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d13.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d13.Loc != LocRegPair && d13.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d13}, 1)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			var d15 JITValueDesc
			if d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Int() <= 0)}
			} else {
				r0 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitCmpRegImm32(d14.Reg, 0)
				ctx.EmitSetcc(r0, CcLE)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d15)
			}
			d16 = d15
			ctx.EnsureDesc(&d16)
			if d16.Loc != LocImm && d16.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d16.Loc == LocImm {
				if d16.Imm.Bool() {
			ps17 := PhiState{General: ps.General}
			ps17.OverlayValues = make([]JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[13] = d13
			ps17.OverlayValues[14] = d14
			ps17.OverlayValues[15] = d15
			ps17.OverlayValues[16] = d16
					return bbs[3].RenderPS(ps17)
				}
			ps18 := PhiState{General: ps.General}
			ps18.OverlayValues = make([]JITValueDesc, 17)
			ps18.OverlayValues[0] = d0
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
			ps18.OverlayValues[3] = d3
			ps18.OverlayValues[13] = d13
			ps18.OverlayValues[14] = d14
			ps18.OverlayValues[15] = d15
			ps18.OverlayValues[16] = d16
				return bbs[4].RenderPS(ps18)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d16.Reg, 0)
			ctx.EmitJcc(CcNE, lbl8)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl5)
			ps19 := PhiState{General: true}
			ps19.OverlayValues = make([]JITValueDesc, 17)
			ps19.OverlayValues[0] = d0
			ps19.OverlayValues[1] = d1
			ps19.OverlayValues[2] = d2
			ps19.OverlayValues[3] = d3
			ps19.OverlayValues[13] = d13
			ps19.OverlayValues[14] = d14
			ps19.OverlayValues[15] = d15
			ps19.OverlayValues[16] = d16
			ps20 := PhiState{General: true}
			ps20.OverlayValues = make([]JITValueDesc, 17)
			ps20.OverlayValues[0] = d0
			ps20.OverlayValues[1] = d1
			ps20.OverlayValues[2] = d2
			ps20.OverlayValues[3] = d3
			ps20.OverlayValues[13] = d13
			ps20.OverlayValues[14] = d14
			ps20.OverlayValues[15] = d15
			ps20.OverlayValues[16] = d16
			snap21 := d0
			snap22 := d1
			snap23 := d2
			snap24 := d3
			snap25 := d13
			snap26 := d14
			snap27 := d15
			snap28 := d16
			alloc29 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps20)
			}
			ctx.RestoreAllocState(alloc29)
			d0 = snap21
			d1 = snap22
			d2 = snap23
			d3 = snap24
			d13 = snap25
			d14 = snap26
			d15 = snap27
			d16 = snap28
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps19)
			}
			return result
			ctx.FreeDesc(&d15)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
			}
			ctx.ReclaimUntrackedRegs()
			d30 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d30, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			ctx.ReclaimUntrackedRegs()
			d31 = args[0]
			d31.ID = 0
			d33 = d31
			ctx.EnsureDesc(&d33)
			if d33.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d33.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d33)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d33)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d33)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d33.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d33 = tmpPair
			} else if d33.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d33.Reg), Reg2: ctx.AllocRegExcept(d33.Reg)}
				switch d33.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d33)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d33)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d33)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d33)
				d33 = tmpPair
			} else if d33.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d33.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d33.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d33 = tmpPair
			}
			if d33.Loc != LocRegPair && d33.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d32 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d33}, 2)
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d32)
			if d32.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d32.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d32.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d32)
				} else if d32.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d32)
				} else if d32.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d32)
				} else if d32.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d32.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d32 = tmpPair
			} else if d32.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d32.Type, Reg: ctx.AllocRegExcept(d32.Reg), Reg2: ctx.AllocRegExcept(d32.Reg)}
				switch d32.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d32)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d32)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d32)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d32)
				d32 = tmpPair
			}
			if d32.Loc != LocRegPair && d32.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.Repeat arg0)")
			}
			ctx.EnsureDesc(&d14)
			if d14.Loc == LocRegPair || d14.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d34 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Repeat), []JITValueDesc{d32, d14}, 2)
			ctx.FreeDesc(&d14)
			d35 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d34}, 2)
			ctx.EmitMovPairToResult(&d35, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned36 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned36 = append(argPinned36, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned36 = append(argPinned36, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned36 = append(argPinned36, ai.Reg2)
					}
				}
			}
			ps37 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps37)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned36 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */
	})

	/* comparison */
	collation_re := regexp.MustCompile("^([^_]+_)?(.+?)$") // caracterset_language_case
	Declare(&Globalenv, &Declaration{
		"collate", "returns the `<` operator for a given collation. MemCP allows natural sorting of numeric literals.",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"collation", "string", "collation string of the form LANG or LANG_cs or LANG_ci where LANG is a BCP 47 code, for compatibility to MySQL, a CHARSET_ prefix is allowed and ignored as well as the aliases bin, danish, general, german1, german2, spanish and swedish are allowed for language codes", nil},
			DeclarationParameter{"reverse", "bool", "whether to reverse the order like in ORDER BY DESC", nil},
		}, "func",
		func(a ...Scmer) Scmer {
			collation := String(a[0])
			ci := false
			if strings.HasSuffix(collation, "_ci") {
				ci = true
				collation = collation[:len(collation)-3]
			} else if strings.HasSuffix(collation, "_cs") {
				collation = collation[:len(collation)-3]
			}
			if m := collation_re.FindStringSubmatch(collation); m != nil {
				if m[2] == "bin" { // binary
					// Return closures that compare raw UTF-8 byte order; register for serialization
					if len(a) > 1 && ToBool(a[1]) {
						f := func(a ...Scmer) Scmer { return GreaterScm(a...) }
						collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
							Collation string
							Reverse   bool
						}{Collation: String(a[0]), Reverse: true})
						return NewFunc(f)
					}
					f := func(a ...Scmer) Scmer { return LessScm(a...) }
					collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
						Collation string
						Reverse   bool
					}{Collation: String(a[0]), Reverse: false})
					return NewFunc(f)
				}
				base := m[2]
				// Special-case MySQL-style "general" to simple case-insensitive first-letter ordering
				if strings.Contains(base, "general") {
					reverse := len(a) > 1 && ToBool(a[1])
					// general_ci heuristic:
					// - ASCII letters sort before non-ASCII always (both ASC and DESC).
					// - Treat leading "aa" as non-ASCII class to place after ASCII group in ASC and after ASCII even in DESC.
					// - Within ASCII, compare by lowercase first letter; tie-break by case-insensitive string compare.
					classify := func(s string) (isASCII bool, key byte, folded string) {
						if s == "" {
							return true, 0, s
						}
						sl := strings.ToLower(s)
						// map leading "aa" to non-ASCII class
						if len(sl) >= 2 && sl[0] == 'a' && sl[1] == 'a' {
							return false, 0, sl
						}
						b := sl[0]
						// check ASCII letter
						if b >= 'a' && b <= 'z' && (s[0] < 128) {
							return true, b, sl
						}
						return false, 0, sl
					}
					if reverse {
						f := func(a ...Scmer) Scmer {
							as := String(a[0])
							bs := String(a[1])
							aAsc, ak, af := classify(as)
							bAsc, bk, bf := classify(bs)
							var res bool
							if aAsc != bAsc {
								// ASCII ranks above non-ASCII for DESC too
								res = aAsc && !bAsc
							} else if aAsc { // both ASCII letters: reverse letter order
								if ak != bk {
									res = ak > bk
								} else {
									res = af > bf
								}
							} else {
								// both non-ASCII: keep stable fallback
								res = as > bs
							}
							return NewBool(res)
						}
						collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
							Collation string
							Reverse   bool
						}{Collation: String(a[0]), Reverse: true})
						return NewFunc(f)
					}
					f := func(a ...Scmer) Scmer {
						as := String(a[0])
						bs := String(a[1])
						aAsc, ak, af := classify(as)
						bAsc, bk, bf := classify(bs)
						var res bool
						if aAsc != bAsc {
							// ASCII first for ASC
							res = aAsc && !bAsc
						} else if aAsc { // both ASCII letters
							if ak != bk {
								res = ak < bk
							} else {
								res = af < bf
							}
						} else {
							// both non-ASCII: leave at end
							res = as < bs
						}
						return NewBool(res)
					}
					collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
						Collation string
						Reverse   bool
					}{Collation: String(a[0]), Reverse: false})
					return NewFunc(f)
				}
				tag, err := language.Parse(base) // treat as BCP 47
				if err != nil {
					// language not detected, try one of the aliases
					switch m[2] {
					case "danish":
						tag = language.Danish
					case "german1":
						tag = language.German
					case "german2":
						tag = language.German
					case "spanish":
						tag = language.Spanish
					case "swedish":
						tag = language.Swedish
					default:
						tag = language.Danish // default to danish for general-like collations (aa -> å semantics)
					}
				}
				var c *collate.Collator
				// the following options are available:
				// IgnoreCase -> when string ends with _ci
				// IgnoreDiacritics -> o == ö
				// IgnoreWidth: half width == width
				// Numeric -> sort numbers correctly
				if ci {
					c = collate.New(tag, collate.Numeric, collate.IgnoreCase)
				} else {
					c = collate.New(tag, collate.Numeric)
				}

				// return a LESS function specialized to that language and register for serialization
				reverse := len(a) > 1 && ToBool(a[1])
				if reverse {
					f := func(a ...Scmer) Scmer {
						var res bool
						// numeric fallback when both operands are numbers
						if (a[0].IsInt() || a[0].IsFloat()) && (a[1].IsInt() || a[1].IsFloat()) {
							res = ToFloat(a[0]) > ToFloat(a[1])
						}
						if !res {
							res = c.CompareString(String(a[0]), String(a[1])) == 1
						}
						return NewBool(res)
					}
					collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
						Collation string
						Reverse   bool
					}{Collation: String(a[0]), Reverse: true})
					return NewFunc(f)
				}
				f := func(a ...Scmer) Scmer {
					// numeric fallback when both operands are numbers
					if (a[0].IsInt() || a[0].IsFloat()) && (a[1].IsInt() || a[1].IsFloat()) {
						return NewBool(ToFloat(a[0]) < ToFloat(a[1]))
					}
					return NewBool(c.CompareString(String(a[0]), String(a[1])) == -1)
				}
				collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
					Collation string
					Reverse   bool
				}{Collation: String(a[0]), Reverse: false})
				return NewFunc(f)
			} else {
				if len(a) > 1 && ToBool(a[1]) {
					return NewFunc(GreaterScm)
				}
				return NewFunc(LessScm)
			}
		}, true, false, nil,
		nil /* TODO: Slice on non-desc: slice t0[:0:int] */, /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */
	})

	/* escaping functions similar to PHP */
	Declare(&Globalenv, &Declaration{
		"htmlentities", "escapes the string for use in HTML",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(html.EscapeString(String(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (html.EscapeString arg0)")
			}
			d3 = ctx.EmitGoCallScalar(GoFuncAddr(html.EscapeString), []JITValueDesc{d1}, 2)
			ctx.ResolveFixups()
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned5 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned5 = append(argPinned5, ai.Reg2)
					}
				}
			}
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
			for _, r := range argPinned5 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */
	})
	Declare(&Globalenv, &Declaration{
		"urlencode", "encodes a string according to URI coding schema",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to encode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(url.QueryEscape(String(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d2.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			} else if d2.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (url.QueryEscape arg0)")
			}
			d3 = ctx.EmitGoCallScalar(GoFuncAddr(url.QueryEscape), []JITValueDesc{d1}, 2)
			ctx.ResolveFixups()
			d4 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
			result.Type = tagString
			return result
			return result
			}
			argPinned5 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned5 = append(argPinned5, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned5 = append(argPinned5, ai.Reg2)
					}
				}
			}
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
			for _, r := range argPinned5 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */
	})
	Declare(&Globalenv, &Declaration{
		"urldecode", "decodes a string according to URI coding schema",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to decode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			result, err := url.QueryUnescape(String(a[0]))
			if err != nil {
				panic("error while decoding URL: " + fmt.Sprint(err))
			}
			return NewString(result)
		}, true, false, nil,
		nil /* TODO: FieldAddr on non-receiver: &b.addr [#0] */, /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */
	})
	Declare(&Globalenv, &Declaration{
		"json_encode", "encodes a value in JSON, treats lists as lists",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to encode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			b, err := json.Marshal(a[0])
			if err != nil {
				panic(err)
			}
			return NewString(string(b))
		}, true, false, nil,
		nil /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */, /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */
	})
	Declare(&Globalenv, &Declaration{
		"json_encode_assoc", "encodes a value in JSON, treats lists as associative arrays",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to encode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			// Build a Go structure where assoc lists (even-length lists or FastDict)
			// are represented as map[string]any, and leaf values remain Scmer so
			// Scmer.MarshalJSON applies for nested values.
			var transform func(Scmer) any
			transform = func(val Scmer) any {
				if val.IsSlice() {
					v := val.Slice()
					result := make(map[string]any)
					for i := 0; i < len(v)-1; i += 2 {
						result[String(v[i])] = transform(v[i+1])
					}
					return result
				}
				if val.IsFastDict() {
					fd := val.FastDict()
					result := make(map[string]any)
					if fd != nil {
						for i := 0; i < len(fd.Pairs)-1; i += 2 {
							result[String(fd.Pairs[i])] = transform(fd.Pairs[i+1])
						}
					}
					return result
				}
				// Keep as Scmer so its MarshalJSON semantics apply
				return val
			}
			b, err := json.Marshal(transform(a[0]))
			if err != nil {
				panic(err)
			}
			return NewString(string(b))
		}, true, false, nil,
		nil /* TODO: MakeClosure binding not an alloc-stored value */, /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */
	})
	Declare(&Globalenv, &Declaration{
		"json_decode", "parses JSON into a map",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to decode", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			var result any
			err := json.Unmarshal([]byte(String(a[0])), &result)
			if err != nil {
				panic(err)
			}
			return TransformFromJSON(result)
		}, true, false, nil,
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
	})

	Declare(&Globalenv, &Declaration{
		"base64_encode", "encodes a string as Base64 (standard encoding)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "binary string to encode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(base64.StdEncoding.EncodeToString([]byte(String(a[0]))))
		}, true, false, nil,
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
	})
	Declare(&Globalenv, &Declaration{
		"base64_decode", "decodes a Base64 string (standard encoding)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "base64-encoded string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			decoded, err := base64.StdEncoding.DecodeString(String(a[0]))
			if err != nil {
				panic("error while decoding base64: " + fmt.Sprint(err))
			}
			return NewString(string(decoded))
		}, true, false, nil,
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */
	})
	sql_escapings := regexp.MustCompile("\\\\[\\\\'\"nr0]")
	Declare(&Globalenv, &Declaration{
		"sql_unescape", "unescapes the inner part of a sql string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to decode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			input := String(a[0])
			out := sql_escapings.ReplaceAllStringFunc(input, func(m string) string {
				switch m {
				case "\\\\":
					return "\\"
				case "\\'":
					return "'"
				case "\\\"":
					return "\""
				case "\\n":
					return "\n"
				case "\\r":
					return "\r"
				case "\\0":
					return string([]byte{0})
				}
				return m
			})
			return NewString(out)
		}, true, false, nil,
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */
	})
	Declare(&Globalenv, &Declaration{
		"bin2hex", "turns binary data into hex with lowercase letters",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to decode", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			input := String(a[0])
			result := make([]byte, 2*len(input))
			hexmap := "0123456789abcdef"
			for i := 0; i < len(input); i++ {
				result[2*i] = hexmap[input[i]/16]
				result[2*i+1] = hexmap[input[i]%16]
			}
			return NewString(string(result))
		}, true, false, nil,
		nil /* TODO: MakeSlice: make []byte t4 t4 */, /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */
	})
	Declare(&Globalenv, &Declaration{
		"hex2bin", "decodes a hex string into binary data",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "hex string (even length)", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			decoded, err := hex.DecodeString(String(a[0]))
			if err != nil {
				panic("error while decoding hex: " + fmt.Sprint(err))
			}
			return NewString(string(decoded))
		}, true, false, nil,
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */
	})

	Declare(&Globalenv, &Declaration{
		"randomBytes", "returns a string with numBytes cryptographically secure random bytes",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"numBytes", "number", "number of random bytes", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			n := ToInt(a[0])
			if n < 0 {
				panic("randomBytes: numBytes must be non-negative")
			}
			buf := make([]byte, n)
			if n > 0 {
				if _, err := crand.Read(buf); err != nil {
					panic("error generating random bytes: " + fmt.Sprint(err))
				}
			}
			return NewString(string(buf))
		}, true, false, nil,
		nil /* TODO: MakeSlice: make []byte t2 t2 */, /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */
	})

	Declare(&Globalenv, &Declaration{
		"regexp_replace", "replaces matches of a regex pattern in a string",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"str", "string", "input string", nil},
			DeclarationParameter{"pattern", "string", "regex pattern", nil},
			DeclarationParameter{"replacement", "string", "replacement string", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			re, err := regexp.Compile(String(a[1]))
			if err != nil {
				panic("regexp_replace: invalid pattern: " + err.Error())
			}
			return NewString(re.ReplaceAllString(String(a[0]), String(a[2])))
		}, true, false, &TypeDescriptor{Optimize: optimizeRegexpReplace},
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
	})

	Declare(&Globalenv, &Declaration{
		"regexp_test", "tests if a string matches a regex pattern, returns true/false",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"str", "string", "input string", nil},
			DeclarationParameter{"pattern", "string", "regex pattern", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			re, err := regexp.Compile(String(a[1]))
			if err != nil {
				panic("regexp_test: invalid pattern: " + err.Error())
			}
			return NewBool(re.MatchString(String(a[0])))
		}, true, false, &TypeDescriptor{Optimize: optimizeRegexpTest},
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
	})

}

// optimizeRegexpReplace precompiles the regex when the pattern argument is a constant string.
// This avoids calling regexp.Compile() on every invocation at runtime.
func optimizeRegexpReplace(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Optimize all arguments first
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if td != nil && td.Const {
		return result, td // already constant-folded
	}
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 4 {
		return result, td
	}
	// Check if the pattern (arg 2, index 2) is a constant string
	if !rv[2].IsString() {
		return result, td
	}
	pattern := rv[2].String()
	re, err := regexp.Compile(pattern)
	if err != nil {
		return result, td // let runtime handle the error
	}
	// Replace call with a precompiled closure
	compiled := NewFunc(func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return NewNil()
		}
		return NewString(re.ReplaceAllString(String(a[0]), String(a[1])))
	})
	// Rewrite: (regexp_replace str pattern repl) -> (compiled_fn str repl)
	return NewSlice([]Scmer{compiled, rv[1], rv[3]}), td
}

// optimizeRegexpTest precompiles the regex when the pattern argument is a constant string.
func optimizeRegexpTest(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if td != nil && td.Const {
		return result, td
	}
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 3 {
		return result, td
	}
	// Check if the pattern (arg 2, index 2) is a constant string
	if !rv[2].IsString() {
		return result, td
	}
	pattern := rv[2].String()
	re, err := regexp.Compile(pattern)
	if err != nil {
		return result, td
	}
	compiled := NewFunc(func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return NewNil()
		}
		return NewBool(re.MatchString(String(a[0])))
	})
	// Rewrite: (regexp_test str pattern) -> (compiled_fn str)
	return NewSlice([]Scmer{compiled, rv[1]}), td
}
