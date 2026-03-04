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
			ps3 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps3)
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
		nil /* TODO: IndexAddr on non-parameter: &t82[0:int] (x=t82 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t78[0:int] (x=t78 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t78[0:int] (x=t78 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t78[0:int] (x=t78 marker="_alloc" isDesc=false goVar=) */ /* TODO: ChangeType: changetype Symbol <- string (t25) */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
			snap12 := d1
			snap13 := d4
			alloc14 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps11)
			}
			ctx.RestoreAllocState(alloc14)
			d1 = snap12
			d4 = snap13
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
			d15 = args[2]
			d15.ID = 0
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d15.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d15.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d15)
				} else if d15.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d15)
				} else if d15.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d15)
				} else if d15.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d15.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d16 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d15}, 1)
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d16)
			var d17 JITValueDesc
			if d4.Loc == LocImm && d16.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d16.Imm.Int())}
			} else if d16.Loc == LocImm && d16.Imm.Int() == 0 {
				r1 := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(r1, d4.Reg)
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
				ctx.BindReg(r1, &d17)
			} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
				ctx.BindReg(d16.Reg, &d17)
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.EmitAddInt64(scratch, d16.Reg)
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else if d16.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(scratch, d4.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else {
				r2 := ctx.AllocRegExcept(d4.Reg, d16.Reg)
				ctx.EmitMovRegReg(r2, d4.Reg)
				ctx.EmitAddInt64(r2, d16.Reg)
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
				ctx.BindReg(r2, &d17)
			}
			if d17.Loc == LocReg && d4.Loc == LocReg && d17.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d17)
			var d19 JITValueDesc
			if d17.Loc == LocImm && d4.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d17.Imm.Int() - d4.Imm.Int())}
			} else {
				r3 := ctx.AllocReg()
				if d17.Loc == LocImm {
					ctx.EmitMovRegImm64(r3, uint64(d17.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r3, d17.Reg)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitSubInt64(r3, RegR11)
				} else {
					ctx.EmitSubInt64(r3, d4.Reg)
				}
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
				ctx.BindReg(r3, &d19)
			}
			var d20 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d4.Imm.Int())}
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
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
				ctx.BindReg(r4, &d20)
			}
			var d21 JITValueDesc
			r5 := ctx.AllocReg()
			r6 := ctx.AllocReg()
			if d20.Loc == LocImm {
				ctx.EmitMovRegImm64(r5, uint64(d20.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r5, d20.Reg)
				ctx.FreeReg(d20.Reg)
			}
			if d19.Loc == LocImm {
				ctx.EmitMovRegImm64(r6, uint64(d19.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r6, d19.Reg)
				ctx.FreeReg(d19.Reg)
			}
			d21 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
			ctx.BindReg(r5, &d21)
			ctx.BindReg(r6, &d21)
			ctx.FreeDesc(&d17)
			d22 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d21}, 2)
			ctx.EmitMovPairToResult(&d22, &result)
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
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			var d23 JITValueDesc
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair {
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
				ctx.BindReg(d1.Reg2, &d23)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d23)
			var d25 JITValueDesc
			if d23.Loc == LocImm && d4.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() - d4.Imm.Int())}
			} else {
				r7 := ctx.AllocReg()
				if d23.Loc == LocImm {
					ctx.EmitMovRegImm64(r7, uint64(d23.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r7, d23.Reg)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitSubInt64(r7, RegR11)
				} else {
					ctx.EmitSubInt64(r7, d4.Reg)
				}
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
				ctx.BindReg(r7, &d25)
			}
			var d26 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d4.Imm.Int())}
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
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d26)
			}
			var d27 JITValueDesc
			r9 := ctx.AllocReg()
			r10 := ctx.AllocReg()
			if d26.Loc == LocImm {
				ctx.EmitMovRegImm64(r9, uint64(d26.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r9, d26.Reg)
				ctx.FreeReg(d26.Reg)
			}
			if d25.Loc == LocImm {
				ctx.EmitMovRegImm64(r10, uint64(d25.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r10, d25.Reg)
				ctx.FreeReg(d25.Reg)
			}
			d27 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
			ctx.BindReg(r9, &d27)
			ctx.BindReg(r10, &d27)
			ctx.FreeDesc(&d4)
			d28 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d27}, 2)
			ctx.EmitMovPairToResult(&d28, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			ps29 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps29)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
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
			var d21 JITValueDesc
			_ = d21
			var d23 JITValueDesc
			_ = d23
			var d24 JITValueDesc
			_ = d24
			var d27 JITValueDesc
			_ = d27
			var d30 JITValueDesc
			_ = d30
			var d31 JITValueDesc
			_ = d31
			var d32 JITValueDesc
			_ = d32
			var d33 JITValueDesc
			_ = d33
			var d39 JITValueDesc
			_ = d39
			var d40 JITValueDesc
			_ = d40
			var d41 JITValueDesc
			_ = d41
			var d42 JITValueDesc
			_ = d42
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
			var d58 JITValueDesc
			_ = d58
			var d59 JITValueDesc
			_ = d59
			var d62 JITValueDesc
			_ = d62
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
			var d75 JITValueDesc
			_ = d75
			var d76 JITValueDesc
			_ = d76
			var d77 JITValueDesc
			_ = d77
			var d78 JITValueDesc
			_ = d78
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			var bbs [13]BBDescriptor
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			alloc10 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps9)
			}
			ctx.RestoreAllocState(alloc10)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			d11 = args[0]
			d11.ID = 0
			d13 = d11
			ctx.EnsureDesc(&d13)
			if d13.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d13.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d13)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d13)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d13)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d13.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d13 = tmpPair
			} else if d13.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d13.Reg), Reg2: ctx.AllocRegExcept(d13.Reg)}
				switch d13.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d13)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d13)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d13)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d13)
				d13 = tmpPair
			} else if d13.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d13.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d13.MemPtr))
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
				d13 = tmpPair
			}
			if d13.Loc != LocRegPair && d13.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d12 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d13}, 2)
			ctx.FreeDesc(&d11)
			var d14 JITValueDesc
			if d12.Loc == LocImm {
				d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d12.Imm.String())))}
			} else {
				ctx.EnsureDesc(&d12)
				if d12.Loc == LocRegPair {
					d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2}
					ctx.BindReg(d12.Reg2, &d14)
					ctx.BindReg(d12.Reg2, &d14)
				} else if d12.Loc == LocReg {
					d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg}
					ctx.BindReg(d12.Reg, &d14)
					ctx.BindReg(d12.Reg, &d14)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			d15 = args[1]
			d15.ID = 0
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d15.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d15.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d15)
				} else if d15.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d15)
				} else if d15.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d15)
				} else if d15.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d15.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d16 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d15}, 1)
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d16)
			var d17 JITValueDesc
			if d16.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.EmitMovRegReg(scratch, d16.Reg)
				ctx.EmitSubRegImm32(scratch, int32(1))
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			}
			if d17.Loc == LocReg && d16.Loc == LocReg && d17.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d17)
			var d18 JITValueDesc
			if d17.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitCmpRegImm32(d17.Reg, 0)
				ctx.EmitSetcc(r1, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d18)
			}
			d19 = d18
			ctx.EnsureDesc(&d19)
			if d19.Loc != LocImm && d19.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d19.Loc == LocImm {
				if d19.Imm.Bool() {
			ps20 := PhiState{General: ps.General}
			ps20.OverlayValues = make([]JITValueDesc, 20)
			ps20.OverlayValues[0] = d0
			ps20.OverlayValues[1] = d1
			ps20.OverlayValues[2] = d2
			ps20.OverlayValues[3] = d3
			ps20.OverlayValues[4] = d4
			ps20.OverlayValues[5] = d5
			ps20.OverlayValues[11] = d11
			ps20.OverlayValues[12] = d12
			ps20.OverlayValues[13] = d13
			ps20.OverlayValues[14] = d14
			ps20.OverlayValues[15] = d15
			ps20.OverlayValues[16] = d16
			ps20.OverlayValues[17] = d17
			ps20.OverlayValues[18] = d18
			ps20.OverlayValues[19] = d19
					return bbs[3].RenderPS(ps20)
				}
			d21 = d17
			if d21.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d21)
			ctx.EmitStoreToStack(d21, 0)
			ps22 := PhiState{General: ps.General}
			ps22.OverlayValues = make([]JITValueDesc, 22)
			ps22.OverlayValues[0] = d0
			ps22.OverlayValues[1] = d1
			ps22.OverlayValues[2] = d2
			ps22.OverlayValues[3] = d3
			ps22.OverlayValues[4] = d4
			ps22.OverlayValues[5] = d5
			ps22.OverlayValues[11] = d11
			ps22.OverlayValues[12] = d12
			ps22.OverlayValues[13] = d13
			ps22.OverlayValues[14] = d14
			ps22.OverlayValues[15] = d15
			ps22.OverlayValues[16] = d16
			ps22.OverlayValues[17] = d17
			ps22.OverlayValues[18] = d18
			ps22.OverlayValues[19] = d19
			ps22.OverlayValues[21] = d21
			ps22.PhiValues = make([]JITValueDesc, 1)
			d23 = d17
			ps22.PhiValues[0] = d23
				return bbs[4].RenderPS(ps22)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d19.Reg, 0)
			ctx.EmitJcc(CcNE, lbl16)
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl17)
			d24 = d17
			if d24.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			ctx.EmitStoreToStack(d24, 0)
			ctx.EmitJmp(lbl5)
			ps25 := PhiState{General: true}
			ps25.OverlayValues = make([]JITValueDesc, 25)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[12] = d12
			ps25.OverlayValues[13] = d13
			ps25.OverlayValues[14] = d14
			ps25.OverlayValues[15] = d15
			ps25.OverlayValues[16] = d16
			ps25.OverlayValues[17] = d17
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[19] = d19
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[23] = d23
			ps25.OverlayValues[24] = d24
			ps26 := PhiState{General: true}
			ps26.OverlayValues = make([]JITValueDesc, 25)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[11] = d11
			ps26.OverlayValues[12] = d12
			ps26.OverlayValues[13] = d13
			ps26.OverlayValues[14] = d14
			ps26.OverlayValues[15] = d15
			ps26.OverlayValues[16] = d16
			ps26.OverlayValues[17] = d17
			ps26.OverlayValues[18] = d18
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[23] = d23
			ps26.OverlayValues[24] = d24
			ps26.PhiValues = make([]JITValueDesc, 1)
			d27 = d17
			ps26.PhiValues[0] = d27
			alloc28 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps26)
			}
			ctx.RestoreAllocState(alloc28)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps25)
			}
			return result
			ctx.FreeDesc(&d18)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ps29 := PhiState{General: ps.General}
			ps29.OverlayValues = make([]JITValueDesc, 28)
			ps29.OverlayValues[0] = d0
			ps29.OverlayValues[1] = d1
			ps29.OverlayValues[2] = d2
			ps29.OverlayValues[3] = d3
			ps29.OverlayValues[4] = d4
			ps29.OverlayValues[5] = d5
			ps29.OverlayValues[11] = d11
			ps29.OverlayValues[12] = d12
			ps29.OverlayValues[13] = d13
			ps29.OverlayValues[14] = d14
			ps29.OverlayValues[15] = d15
			ps29.OverlayValues[16] = d16
			ps29.OverlayValues[17] = d17
			ps29.OverlayValues[18] = d18
			ps29.OverlayValues[19] = d19
			ps29.OverlayValues[21] = d21
			ps29.OverlayValues[23] = d23
			ps29.OverlayValues[24] = d24
			ps29.OverlayValues[27] = d27
			ps29.PhiValues = make([]JITValueDesc, 1)
			d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps29.PhiValues[0] = d30
			if ps29.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps29)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d31 := ps.PhiValues[0]
						ctx.EnsureDesc(&d31)
						ctx.EmitStoreToStack(d31, 0)
					}
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d14)
			var d32 JITValueDesc
			if d0.Loc == LocImm && d14.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() >= d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r2 := ctx.AllocRegExcept(d0.Reg)
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d0.Reg, int32(d14.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.EmitCmpInt64(d0.Reg, RegR11)
				}
				ctx.EmitSetcc(r2, CcGE)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d32)
			} else if d0.Loc == LocImm {
				r3 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d14.Reg)
				ctx.EmitSetcc(r3, CcGE)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d32)
			} else {
				r4 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitCmpInt64(d0.Reg, d14.Reg)
				ctx.EmitSetcc(r4, CcGE)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d32)
			}
			d33 = d32
			ctx.EnsureDesc(&d33)
			if d33.Loc != LocImm && d33.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
			ps34 := PhiState{General: ps.General}
			ps34.OverlayValues = make([]JITValueDesc, 34)
			ps34.OverlayValues[0] = d0
			ps34.OverlayValues[1] = d1
			ps34.OverlayValues[2] = d2
			ps34.OverlayValues[3] = d3
			ps34.OverlayValues[4] = d4
			ps34.OverlayValues[5] = d5
			ps34.OverlayValues[11] = d11
			ps34.OverlayValues[12] = d12
			ps34.OverlayValues[13] = d13
			ps34.OverlayValues[14] = d14
			ps34.OverlayValues[15] = d15
			ps34.OverlayValues[16] = d16
			ps34.OverlayValues[17] = d17
			ps34.OverlayValues[18] = d18
			ps34.OverlayValues[19] = d19
			ps34.OverlayValues[21] = d21
			ps34.OverlayValues[23] = d23
			ps34.OverlayValues[24] = d24
			ps34.OverlayValues[27] = d27
			ps34.OverlayValues[30] = d30
			ps34.OverlayValues[31] = d31
			ps34.OverlayValues[32] = d32
			ps34.OverlayValues[33] = d33
					return bbs[5].RenderPS(ps34)
				}
			ps35 := PhiState{General: ps.General}
			ps35.OverlayValues = make([]JITValueDesc, 34)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[3] = d3
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[11] = d11
			ps35.OverlayValues[12] = d12
			ps35.OverlayValues[13] = d13
			ps35.OverlayValues[14] = d14
			ps35.OverlayValues[15] = d15
			ps35.OverlayValues[16] = d16
			ps35.OverlayValues[17] = d17
			ps35.OverlayValues[18] = d18
			ps35.OverlayValues[19] = d19
			ps35.OverlayValues[21] = d21
			ps35.OverlayValues[23] = d23
			ps35.OverlayValues[24] = d24
			ps35.OverlayValues[27] = d27
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
				return bbs[6].RenderPS(ps35)
			}
			if !ps.General {
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl18 := ctx.ReserveLabel()
			lbl19 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d33.Reg, 0)
			ctx.EmitJcc(CcNE, lbl18)
			ctx.EmitJmp(lbl19)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl7)
			ps36 := PhiState{General: true}
			ps36.OverlayValues = make([]JITValueDesc, 34)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[3] = d3
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[11] = d11
			ps36.OverlayValues[12] = d12
			ps36.OverlayValues[13] = d13
			ps36.OverlayValues[14] = d14
			ps36.OverlayValues[15] = d15
			ps36.OverlayValues[16] = d16
			ps36.OverlayValues[17] = d17
			ps36.OverlayValues[18] = d18
			ps36.OverlayValues[19] = d19
			ps36.OverlayValues[21] = d21
			ps36.OverlayValues[23] = d23
			ps36.OverlayValues[24] = d24
			ps36.OverlayValues[27] = d27
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps37 := PhiState{General: true}
			ps37.OverlayValues = make([]JITValueDesc, 34)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[3] = d3
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[11] = d11
			ps37.OverlayValues[12] = d12
			ps37.OverlayValues[13] = d13
			ps37.OverlayValues[14] = d14
			ps37.OverlayValues[15] = d15
			ps37.OverlayValues[16] = d16
			ps37.OverlayValues[17] = d17
			ps37.OverlayValues[18] = d18
			ps37.OverlayValues[19] = d19
			ps37.OverlayValues[21] = d21
			ps37.OverlayValues[23] = d23
			ps37.OverlayValues[24] = d24
			ps37.OverlayValues[27] = d27
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			alloc38 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps37)
			}
			ctx.RestoreAllocState(alloc38)
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps36)
			}
			return result
			ctx.FreeDesc(&d32)
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			d39 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d39, &result)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
			}
			ctx.ReclaimUntrackedRegs()
			d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d40)
			var d41 JITValueDesc
			if d40.Loc == LocImm {
				d41 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d40.Imm.Int() > 2)}
			} else {
				r5 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d40.Reg, 2)
				ctx.EmitSetcc(r5, CcG)
				d41 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d41)
			}
			ctx.FreeDesc(&d40)
			d42 = d41
			ctx.EnsureDesc(&d42)
			if d42.Loc != LocImm && d42.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d42.Loc == LocImm {
				if d42.Imm.Bool() {
			ps43 := PhiState{General: ps.General}
			ps43.OverlayValues = make([]JITValueDesc, 43)
			ps43.OverlayValues[0] = d0
			ps43.OverlayValues[1] = d1
			ps43.OverlayValues[2] = d2
			ps43.OverlayValues[3] = d3
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[5] = d5
			ps43.OverlayValues[11] = d11
			ps43.OverlayValues[12] = d12
			ps43.OverlayValues[13] = d13
			ps43.OverlayValues[14] = d14
			ps43.OverlayValues[15] = d15
			ps43.OverlayValues[16] = d16
			ps43.OverlayValues[17] = d17
			ps43.OverlayValues[18] = d18
			ps43.OverlayValues[19] = d19
			ps43.OverlayValues[21] = d21
			ps43.OverlayValues[23] = d23
			ps43.OverlayValues[24] = d24
			ps43.OverlayValues[27] = d27
			ps43.OverlayValues[30] = d30
			ps43.OverlayValues[31] = d31
			ps43.OverlayValues[32] = d32
			ps43.OverlayValues[33] = d33
			ps43.OverlayValues[39] = d39
			ps43.OverlayValues[40] = d40
			ps43.OverlayValues[41] = d41
			ps43.OverlayValues[42] = d42
					return bbs[7].RenderPS(ps43)
				}
			ps44 := PhiState{General: ps.General}
			ps44.OverlayValues = make([]JITValueDesc, 43)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[3] = d3
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[11] = d11
			ps44.OverlayValues[12] = d12
			ps44.OverlayValues[13] = d13
			ps44.OverlayValues[14] = d14
			ps44.OverlayValues[15] = d15
			ps44.OverlayValues[16] = d16
			ps44.OverlayValues[17] = d17
			ps44.OverlayValues[18] = d18
			ps44.OverlayValues[19] = d19
			ps44.OverlayValues[21] = d21
			ps44.OverlayValues[23] = d23
			ps44.OverlayValues[24] = d24
			ps44.OverlayValues[27] = d27
			ps44.OverlayValues[30] = d30
			ps44.OverlayValues[31] = d31
			ps44.OverlayValues[32] = d32
			ps44.OverlayValues[33] = d33
			ps44.OverlayValues[39] = d39
			ps44.OverlayValues[40] = d40
			ps44.OverlayValues[41] = d41
			ps44.OverlayValues[42] = d42
				return bbs[8].RenderPS(ps44)
			}
			if !ps.General {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d42.Reg, 0)
			ctx.EmitJcc(CcNE, lbl20)
			ctx.EmitJmp(lbl21)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl9)
			ps45 := PhiState{General: true}
			ps45.OverlayValues = make([]JITValueDesc, 43)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[3] = d3
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[5] = d5
			ps45.OverlayValues[11] = d11
			ps45.OverlayValues[12] = d12
			ps45.OverlayValues[13] = d13
			ps45.OverlayValues[14] = d14
			ps45.OverlayValues[15] = d15
			ps45.OverlayValues[16] = d16
			ps45.OverlayValues[17] = d17
			ps45.OverlayValues[18] = d18
			ps45.OverlayValues[19] = d19
			ps45.OverlayValues[21] = d21
			ps45.OverlayValues[23] = d23
			ps45.OverlayValues[24] = d24
			ps45.OverlayValues[27] = d27
			ps45.OverlayValues[30] = d30
			ps45.OverlayValues[31] = d31
			ps45.OverlayValues[32] = d32
			ps45.OverlayValues[33] = d33
			ps45.OverlayValues[39] = d39
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[41] = d41
			ps45.OverlayValues[42] = d42
			ps46 := PhiState{General: true}
			ps46.OverlayValues = make([]JITValueDesc, 43)
			ps46.OverlayValues[0] = d0
			ps46.OverlayValues[1] = d1
			ps46.OverlayValues[2] = d2
			ps46.OverlayValues[3] = d3
			ps46.OverlayValues[4] = d4
			ps46.OverlayValues[5] = d5
			ps46.OverlayValues[11] = d11
			ps46.OverlayValues[12] = d12
			ps46.OverlayValues[13] = d13
			ps46.OverlayValues[14] = d14
			ps46.OverlayValues[15] = d15
			ps46.OverlayValues[16] = d16
			ps46.OverlayValues[17] = d17
			ps46.OverlayValues[18] = d18
			ps46.OverlayValues[19] = d19
			ps46.OverlayValues[21] = d21
			ps46.OverlayValues[23] = d23
			ps46.OverlayValues[24] = d24
			ps46.OverlayValues[27] = d27
			ps46.OverlayValues[30] = d30
			ps46.OverlayValues[31] = d31
			ps46.OverlayValues[32] = d32
			ps46.OverlayValues[33] = d33
			ps46.OverlayValues[39] = d39
			ps46.OverlayValues[40] = d40
			ps46.OverlayValues[41] = d41
			ps46.OverlayValues[42] = d42
			snap47 := d0
			snap48 := d14
			alloc49 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps46)
			}
			ctx.RestoreAllocState(alloc49)
			d0 = snap47
			d14 = snap48
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps45)
			}
			return result
			ctx.FreeDesc(&d41)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			ctx.ReclaimUntrackedRegs()
			d50 = args[2]
			d50.ID = 0
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d50)
			if d50.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d50.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d50.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d50)
				} else if d50.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d50)
				} else if d50.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d50)
				} else if d50.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d50.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d50 = tmpPair
			} else if d50.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d50.Type, Reg: ctx.AllocRegExcept(d50.Reg), Reg2: ctx.AllocRegExcept(d50.Reg)}
				switch d50.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d50)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d50)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d50)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d50)
				d50 = tmpPair
			}
			if d50.Loc != LocRegPair && d50.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d51 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d50}, 1)
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d51)
			var d52 JITValueDesc
			if d0.Loc == LocImm && d51.Loc == LocImm {
				d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d51.Imm.Int())}
			} else if d51.Loc == LocImm && d51.Imm.Int() == 0 {
				r6 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r6, d0.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
				ctx.BindReg(r6, &d52)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d51.Reg}
				ctx.BindReg(d51.Reg, &d52)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d51.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r7 := ctx.AllocRegExcept(d0.Reg, d51.Reg)
				ctx.EmitMovRegReg(r7, d0.Reg)
				ctx.EmitAddInt64(r7, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
				ctx.BindReg(r7, &d52)
			}
			if d52.Loc == LocReg && d0.Loc == LocReg && d52.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d14)
			var d53 JITValueDesc
			if d52.Loc == LocImm && d14.Loc == LocImm {
				d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d52.Imm.Int() > d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r8 := ctx.AllocReg()
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d52.Reg, int32(d14.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.EmitCmpInt64(d52.Reg, RegR11)
				}
				ctx.EmitSetcc(r8, CcG)
				d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d53)
			} else if d52.Loc == LocImm {
				r9 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d52.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d14.Reg)
				ctx.EmitSetcc(r9, CcG)
				d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d53)
			} else {
				r10 := ctx.AllocReg()
				ctx.EmitCmpInt64(d52.Reg, d14.Reg)
				ctx.EmitSetcc(r10, CcG)
				d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d53)
			}
			ctx.FreeDesc(&d52)
			d54 = d53
			ctx.EnsureDesc(&d54)
			if d54.Loc != LocImm && d54.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d54.Loc == LocImm {
				if d54.Imm.Bool() {
			ps55 := PhiState{General: ps.General}
			ps55.OverlayValues = make([]JITValueDesc, 55)
			ps55.OverlayValues[0] = d0
			ps55.OverlayValues[1] = d1
			ps55.OverlayValues[2] = d2
			ps55.OverlayValues[3] = d3
			ps55.OverlayValues[4] = d4
			ps55.OverlayValues[5] = d5
			ps55.OverlayValues[11] = d11
			ps55.OverlayValues[12] = d12
			ps55.OverlayValues[13] = d13
			ps55.OverlayValues[14] = d14
			ps55.OverlayValues[15] = d15
			ps55.OverlayValues[16] = d16
			ps55.OverlayValues[17] = d17
			ps55.OverlayValues[18] = d18
			ps55.OverlayValues[19] = d19
			ps55.OverlayValues[21] = d21
			ps55.OverlayValues[23] = d23
			ps55.OverlayValues[24] = d24
			ps55.OverlayValues[27] = d27
			ps55.OverlayValues[30] = d30
			ps55.OverlayValues[31] = d31
			ps55.OverlayValues[32] = d32
			ps55.OverlayValues[33] = d33
			ps55.OverlayValues[39] = d39
			ps55.OverlayValues[40] = d40
			ps55.OverlayValues[41] = d41
			ps55.OverlayValues[42] = d42
			ps55.OverlayValues[50] = d50
			ps55.OverlayValues[51] = d51
			ps55.OverlayValues[52] = d52
			ps55.OverlayValues[53] = d53
			ps55.OverlayValues[54] = d54
					return bbs[9].RenderPS(ps55)
				}
			d56 = d51
			if d56.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			ctx.EmitStoreToStack(d56, 8)
			ps57 := PhiState{General: ps.General}
			ps57.OverlayValues = make([]JITValueDesc, 57)
			ps57.OverlayValues[0] = d0
			ps57.OverlayValues[1] = d1
			ps57.OverlayValues[2] = d2
			ps57.OverlayValues[3] = d3
			ps57.OverlayValues[4] = d4
			ps57.OverlayValues[5] = d5
			ps57.OverlayValues[11] = d11
			ps57.OverlayValues[12] = d12
			ps57.OverlayValues[13] = d13
			ps57.OverlayValues[14] = d14
			ps57.OverlayValues[15] = d15
			ps57.OverlayValues[16] = d16
			ps57.OverlayValues[17] = d17
			ps57.OverlayValues[18] = d18
			ps57.OverlayValues[19] = d19
			ps57.OverlayValues[21] = d21
			ps57.OverlayValues[23] = d23
			ps57.OverlayValues[24] = d24
			ps57.OverlayValues[27] = d27
			ps57.OverlayValues[30] = d30
			ps57.OverlayValues[31] = d31
			ps57.OverlayValues[32] = d32
			ps57.OverlayValues[33] = d33
			ps57.OverlayValues[39] = d39
			ps57.OverlayValues[40] = d40
			ps57.OverlayValues[41] = d41
			ps57.OverlayValues[42] = d42
			ps57.OverlayValues[50] = d50
			ps57.OverlayValues[51] = d51
			ps57.OverlayValues[52] = d52
			ps57.OverlayValues[53] = d53
			ps57.OverlayValues[54] = d54
			ps57.OverlayValues[56] = d56
			ps57.PhiValues = make([]JITValueDesc, 1)
			d58 = d51
			ps57.PhiValues[0] = d58
				return bbs[10].RenderPS(ps57)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl22 := ctx.ReserveLabel()
			lbl23 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d54.Reg, 0)
			ctx.EmitJcc(CcNE, lbl22)
			ctx.EmitJmp(lbl23)
			ctx.MarkLabel(lbl22)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl23)
			d59 = d51
			if d59.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			ctx.EmitStoreToStack(d59, 8)
			ctx.EmitJmp(lbl11)
			ps60 := PhiState{General: true}
			ps60.OverlayValues = make([]JITValueDesc, 60)
			ps60.OverlayValues[0] = d0
			ps60.OverlayValues[1] = d1
			ps60.OverlayValues[2] = d2
			ps60.OverlayValues[3] = d3
			ps60.OverlayValues[4] = d4
			ps60.OverlayValues[5] = d5
			ps60.OverlayValues[11] = d11
			ps60.OverlayValues[12] = d12
			ps60.OverlayValues[13] = d13
			ps60.OverlayValues[14] = d14
			ps60.OverlayValues[15] = d15
			ps60.OverlayValues[16] = d16
			ps60.OverlayValues[17] = d17
			ps60.OverlayValues[18] = d18
			ps60.OverlayValues[19] = d19
			ps60.OverlayValues[21] = d21
			ps60.OverlayValues[23] = d23
			ps60.OverlayValues[24] = d24
			ps60.OverlayValues[27] = d27
			ps60.OverlayValues[30] = d30
			ps60.OverlayValues[31] = d31
			ps60.OverlayValues[32] = d32
			ps60.OverlayValues[33] = d33
			ps60.OverlayValues[39] = d39
			ps60.OverlayValues[40] = d40
			ps60.OverlayValues[41] = d41
			ps60.OverlayValues[42] = d42
			ps60.OverlayValues[50] = d50
			ps60.OverlayValues[51] = d51
			ps60.OverlayValues[52] = d52
			ps60.OverlayValues[53] = d53
			ps60.OverlayValues[54] = d54
			ps60.OverlayValues[56] = d56
			ps60.OverlayValues[58] = d58
			ps60.OverlayValues[59] = d59
			ps61 := PhiState{General: true}
			ps61.OverlayValues = make([]JITValueDesc, 60)
			ps61.OverlayValues[0] = d0
			ps61.OverlayValues[1] = d1
			ps61.OverlayValues[2] = d2
			ps61.OverlayValues[3] = d3
			ps61.OverlayValues[4] = d4
			ps61.OverlayValues[5] = d5
			ps61.OverlayValues[11] = d11
			ps61.OverlayValues[12] = d12
			ps61.OverlayValues[13] = d13
			ps61.OverlayValues[14] = d14
			ps61.OverlayValues[15] = d15
			ps61.OverlayValues[16] = d16
			ps61.OverlayValues[17] = d17
			ps61.OverlayValues[18] = d18
			ps61.OverlayValues[19] = d19
			ps61.OverlayValues[21] = d21
			ps61.OverlayValues[23] = d23
			ps61.OverlayValues[24] = d24
			ps61.OverlayValues[27] = d27
			ps61.OverlayValues[30] = d30
			ps61.OverlayValues[31] = d31
			ps61.OverlayValues[32] = d32
			ps61.OverlayValues[33] = d33
			ps61.OverlayValues[39] = d39
			ps61.OverlayValues[40] = d40
			ps61.OverlayValues[41] = d41
			ps61.OverlayValues[42] = d42
			ps61.OverlayValues[50] = d50
			ps61.OverlayValues[51] = d51
			ps61.OverlayValues[52] = d52
			ps61.OverlayValues[53] = d53
			ps61.OverlayValues[54] = d54
			ps61.OverlayValues[56] = d56
			ps61.OverlayValues[58] = d58
			ps61.OverlayValues[59] = d59
			ps61.PhiValues = make([]JITValueDesc, 1)
			d62 = d51
			ps61.PhiValues[0] = d62
			snap63 := d0
			snap64 := d14
			alloc65 := ctx.SnapshotAllocState()
			if !bbs[10].Rendered {
				bbs[10].RenderPS(ps61)
			}
			ctx.RestoreAllocState(alloc65)
			d0 = snap63
			d14 = snap64
			if !bbs[9].Rendered {
				return bbs[9].RenderPS(ps60)
			}
			return result
			ctx.FreeDesc(&d53)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
				d62 = ps.OverlayValues[62]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			var d66 JITValueDesc
			ctx.EnsureDesc(&d12)
			if d12.Loc == LocRegPair {
				d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2}
				ctx.BindReg(d12.Reg2, &d66)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d66)
			var d68 JITValueDesc
			if d66.Loc == LocImm && d0.Loc == LocImm {
				d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d66.Imm.Int() - d0.Imm.Int())}
			} else {
				r11 := ctx.AllocReg()
				if d66.Loc == LocImm {
					ctx.EmitMovRegImm64(r11, uint64(d66.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r11, d66.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitSubInt64(r11, RegR11)
				} else {
					ctx.EmitSubInt64(r11, d0.Reg)
				}
				d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d68)
			}
			var d69 JITValueDesc
			if d12.Loc == LocImm && d0.Loc == LocImm {
				d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + d0.Imm.Int())}
			} else {
				r12 := ctx.AllocReg()
				if d12.Loc == LocImm {
					ctx.EmitMovRegImm64(r12, uint64(d12.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r12, d12.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitAddInt64(r12, RegR11)
				} else {
					ctx.EmitAddInt64(r12, d0.Reg)
				}
				d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d69)
			}
			var d70 JITValueDesc
			r13 := ctx.AllocReg()
			r14 := ctx.AllocReg()
			if d69.Loc == LocImm {
				ctx.EmitMovRegImm64(r13, uint64(d69.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r13, d69.Reg)
				ctx.FreeReg(d69.Reg)
			}
			if d68.Loc == LocImm {
				ctx.EmitMovRegImm64(r14, uint64(d68.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r14, d68.Reg)
				ctx.FreeReg(d68.Reg)
			}
			d70 = JITValueDesc{Loc: LocRegPair, Reg: r13, Reg2: r14}
			ctx.BindReg(r13, &d70)
			ctx.BindReg(r14, &d70)
			d71 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d70}, 2)
			ctx.EmitMovPairToResult(&d71, &result)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
				d62 = ps.OverlayValues[62]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d0)
			var d72 JITValueDesc
			if d14.Loc == LocImm && d0.Loc == LocImm {
				d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d14.Imm.Int() - d0.Imm.Int())}
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				r15 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(r15, d14.Reg)
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
				ctx.BindReg(r15, &d72)
			} else if d14.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.EmitSubInt64(scratch, d0.Reg)
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d72)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(scratch, d14.Reg)
				if d0.Imm.Int() >= -2147483648 && d0.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d0.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitSubInt64(scratch, RegR11)
				}
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d72)
			} else {
				r16 := ctx.AllocRegExcept(d14.Reg, d0.Reg)
				ctx.EmitMovRegReg(r16, d14.Reg)
				ctx.EmitSubInt64(r16, d0.Reg)
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
				ctx.BindReg(r16, &d72)
			}
			if d72.Loc == LocReg && d14.Loc == LocReg && d72.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = LocNone
			}
			ctx.FreeDesc(&d14)
			d73 = d72
			if d73.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, 8)
			ps74 := PhiState{General: ps.General}
			ps74.OverlayValues = make([]JITValueDesc, 74)
			ps74.OverlayValues[0] = d0
			ps74.OverlayValues[1] = d1
			ps74.OverlayValues[2] = d2
			ps74.OverlayValues[3] = d3
			ps74.OverlayValues[4] = d4
			ps74.OverlayValues[5] = d5
			ps74.OverlayValues[11] = d11
			ps74.OverlayValues[12] = d12
			ps74.OverlayValues[13] = d13
			ps74.OverlayValues[14] = d14
			ps74.OverlayValues[15] = d15
			ps74.OverlayValues[16] = d16
			ps74.OverlayValues[17] = d17
			ps74.OverlayValues[18] = d18
			ps74.OverlayValues[19] = d19
			ps74.OverlayValues[21] = d21
			ps74.OverlayValues[23] = d23
			ps74.OverlayValues[24] = d24
			ps74.OverlayValues[27] = d27
			ps74.OverlayValues[30] = d30
			ps74.OverlayValues[31] = d31
			ps74.OverlayValues[32] = d32
			ps74.OverlayValues[33] = d33
			ps74.OverlayValues[39] = d39
			ps74.OverlayValues[40] = d40
			ps74.OverlayValues[41] = d41
			ps74.OverlayValues[42] = d42
			ps74.OverlayValues[50] = d50
			ps74.OverlayValues[51] = d51
			ps74.OverlayValues[52] = d52
			ps74.OverlayValues[53] = d53
			ps74.OverlayValues[54] = d54
			ps74.OverlayValues[56] = d56
			ps74.OverlayValues[58] = d58
			ps74.OverlayValues[59] = d59
			ps74.OverlayValues[62] = d62
			ps74.OverlayValues[66] = d66
			ps74.OverlayValues[67] = d67
			ps74.OverlayValues[68] = d68
			ps74.OverlayValues[69] = d69
			ps74.OverlayValues[70] = d70
			ps74.OverlayValues[71] = d71
			ps74.OverlayValues[72] = d72
			ps74.OverlayValues[73] = d73
			ps74.PhiValues = make([]JITValueDesc, 1)
			d75 = d72
			ps74.PhiValues[0] = d75
			if ps74.General && bbs[10].Rendered {
				ctx.EmitJmp(lbl11)
				return result
			}
			return bbs[10].RenderPS(ps74)
			return result
			}
			bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[10].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d76 := ps.PhiValues[0]
						ctx.EnsureDesc(&d76)
						ctx.EmitStoreToStack(d76, 8)
					}
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
				d62 = ps.OverlayValues[62]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			var d77 JITValueDesc
			if d1.Loc == LocImm {
				d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < 0)}
			} else {
				r17 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitCmpRegImm32(d1.Reg, 0)
				ctx.EmitSetcc(r17, CcL)
				d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d77)
			}
			d78 = d77
			ctx.EnsureDesc(&d78)
			if d78.Loc != LocImm && d78.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d78.Loc == LocImm {
				if d78.Imm.Bool() {
			ps79 := PhiState{General: ps.General}
			ps79.OverlayValues = make([]JITValueDesc, 79)
			ps79.OverlayValues[0] = d0
			ps79.OverlayValues[1] = d1
			ps79.OverlayValues[2] = d2
			ps79.OverlayValues[3] = d3
			ps79.OverlayValues[4] = d4
			ps79.OverlayValues[5] = d5
			ps79.OverlayValues[11] = d11
			ps79.OverlayValues[12] = d12
			ps79.OverlayValues[13] = d13
			ps79.OverlayValues[14] = d14
			ps79.OverlayValues[15] = d15
			ps79.OverlayValues[16] = d16
			ps79.OverlayValues[17] = d17
			ps79.OverlayValues[18] = d18
			ps79.OverlayValues[19] = d19
			ps79.OverlayValues[21] = d21
			ps79.OverlayValues[23] = d23
			ps79.OverlayValues[24] = d24
			ps79.OverlayValues[27] = d27
			ps79.OverlayValues[30] = d30
			ps79.OverlayValues[31] = d31
			ps79.OverlayValues[32] = d32
			ps79.OverlayValues[33] = d33
			ps79.OverlayValues[39] = d39
			ps79.OverlayValues[40] = d40
			ps79.OverlayValues[41] = d41
			ps79.OverlayValues[42] = d42
			ps79.OverlayValues[50] = d50
			ps79.OverlayValues[51] = d51
			ps79.OverlayValues[52] = d52
			ps79.OverlayValues[53] = d53
			ps79.OverlayValues[54] = d54
			ps79.OverlayValues[56] = d56
			ps79.OverlayValues[58] = d58
			ps79.OverlayValues[59] = d59
			ps79.OverlayValues[62] = d62
			ps79.OverlayValues[66] = d66
			ps79.OverlayValues[67] = d67
			ps79.OverlayValues[68] = d68
			ps79.OverlayValues[69] = d69
			ps79.OverlayValues[70] = d70
			ps79.OverlayValues[71] = d71
			ps79.OverlayValues[72] = d72
			ps79.OverlayValues[73] = d73
			ps79.OverlayValues[75] = d75
			ps79.OverlayValues[76] = d76
			ps79.OverlayValues[77] = d77
			ps79.OverlayValues[78] = d78
					return bbs[11].RenderPS(ps79)
				}
			ps80 := PhiState{General: ps.General}
			ps80.OverlayValues = make([]JITValueDesc, 79)
			ps80.OverlayValues[0] = d0
			ps80.OverlayValues[1] = d1
			ps80.OverlayValues[2] = d2
			ps80.OverlayValues[3] = d3
			ps80.OverlayValues[4] = d4
			ps80.OverlayValues[5] = d5
			ps80.OverlayValues[11] = d11
			ps80.OverlayValues[12] = d12
			ps80.OverlayValues[13] = d13
			ps80.OverlayValues[14] = d14
			ps80.OverlayValues[15] = d15
			ps80.OverlayValues[16] = d16
			ps80.OverlayValues[17] = d17
			ps80.OverlayValues[18] = d18
			ps80.OverlayValues[19] = d19
			ps80.OverlayValues[21] = d21
			ps80.OverlayValues[23] = d23
			ps80.OverlayValues[24] = d24
			ps80.OverlayValues[27] = d27
			ps80.OverlayValues[30] = d30
			ps80.OverlayValues[31] = d31
			ps80.OverlayValues[32] = d32
			ps80.OverlayValues[33] = d33
			ps80.OverlayValues[39] = d39
			ps80.OverlayValues[40] = d40
			ps80.OverlayValues[41] = d41
			ps80.OverlayValues[42] = d42
			ps80.OverlayValues[50] = d50
			ps80.OverlayValues[51] = d51
			ps80.OverlayValues[52] = d52
			ps80.OverlayValues[53] = d53
			ps80.OverlayValues[54] = d54
			ps80.OverlayValues[56] = d56
			ps80.OverlayValues[58] = d58
			ps80.OverlayValues[59] = d59
			ps80.OverlayValues[62] = d62
			ps80.OverlayValues[66] = d66
			ps80.OverlayValues[67] = d67
			ps80.OverlayValues[68] = d68
			ps80.OverlayValues[69] = d69
			ps80.OverlayValues[70] = d70
			ps80.OverlayValues[71] = d71
			ps80.OverlayValues[72] = d72
			ps80.OverlayValues[73] = d73
			ps80.OverlayValues[75] = d75
			ps80.OverlayValues[76] = d76
			ps80.OverlayValues[77] = d77
			ps80.OverlayValues[78] = d78
				return bbs[12].RenderPS(ps80)
			}
			if !ps.General {
				ps.General = true
				return bbs[10].RenderPS(ps)
			}
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d78.Reg, 0)
			ctx.EmitJcc(CcNE, lbl24)
			ctx.EmitJmp(lbl25)
			ctx.MarkLabel(lbl24)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl25)
			ctx.EmitJmp(lbl13)
			ps81 := PhiState{General: true}
			ps81.OverlayValues = make([]JITValueDesc, 79)
			ps81.OverlayValues[0] = d0
			ps81.OverlayValues[1] = d1
			ps81.OverlayValues[2] = d2
			ps81.OverlayValues[3] = d3
			ps81.OverlayValues[4] = d4
			ps81.OverlayValues[5] = d5
			ps81.OverlayValues[11] = d11
			ps81.OverlayValues[12] = d12
			ps81.OverlayValues[13] = d13
			ps81.OverlayValues[14] = d14
			ps81.OverlayValues[15] = d15
			ps81.OverlayValues[16] = d16
			ps81.OverlayValues[17] = d17
			ps81.OverlayValues[18] = d18
			ps81.OverlayValues[19] = d19
			ps81.OverlayValues[21] = d21
			ps81.OverlayValues[23] = d23
			ps81.OverlayValues[24] = d24
			ps81.OverlayValues[27] = d27
			ps81.OverlayValues[30] = d30
			ps81.OverlayValues[31] = d31
			ps81.OverlayValues[32] = d32
			ps81.OverlayValues[33] = d33
			ps81.OverlayValues[39] = d39
			ps81.OverlayValues[40] = d40
			ps81.OverlayValues[41] = d41
			ps81.OverlayValues[42] = d42
			ps81.OverlayValues[50] = d50
			ps81.OverlayValues[51] = d51
			ps81.OverlayValues[52] = d52
			ps81.OverlayValues[53] = d53
			ps81.OverlayValues[54] = d54
			ps81.OverlayValues[56] = d56
			ps81.OverlayValues[58] = d58
			ps81.OverlayValues[59] = d59
			ps81.OverlayValues[62] = d62
			ps81.OverlayValues[66] = d66
			ps81.OverlayValues[67] = d67
			ps81.OverlayValues[68] = d68
			ps81.OverlayValues[69] = d69
			ps81.OverlayValues[70] = d70
			ps81.OverlayValues[71] = d71
			ps81.OverlayValues[72] = d72
			ps81.OverlayValues[73] = d73
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps82 := PhiState{General: true}
			ps82.OverlayValues = make([]JITValueDesc, 79)
			ps82.OverlayValues[0] = d0
			ps82.OverlayValues[1] = d1
			ps82.OverlayValues[2] = d2
			ps82.OverlayValues[3] = d3
			ps82.OverlayValues[4] = d4
			ps82.OverlayValues[5] = d5
			ps82.OverlayValues[11] = d11
			ps82.OverlayValues[12] = d12
			ps82.OverlayValues[13] = d13
			ps82.OverlayValues[14] = d14
			ps82.OverlayValues[15] = d15
			ps82.OverlayValues[16] = d16
			ps82.OverlayValues[17] = d17
			ps82.OverlayValues[18] = d18
			ps82.OverlayValues[19] = d19
			ps82.OverlayValues[21] = d21
			ps82.OverlayValues[23] = d23
			ps82.OverlayValues[24] = d24
			ps82.OverlayValues[27] = d27
			ps82.OverlayValues[30] = d30
			ps82.OverlayValues[31] = d31
			ps82.OverlayValues[32] = d32
			ps82.OverlayValues[33] = d33
			ps82.OverlayValues[39] = d39
			ps82.OverlayValues[40] = d40
			ps82.OverlayValues[41] = d41
			ps82.OverlayValues[42] = d42
			ps82.OverlayValues[50] = d50
			ps82.OverlayValues[51] = d51
			ps82.OverlayValues[52] = d52
			ps82.OverlayValues[53] = d53
			ps82.OverlayValues[54] = d54
			ps82.OverlayValues[56] = d56
			ps82.OverlayValues[58] = d58
			ps82.OverlayValues[59] = d59
			ps82.OverlayValues[62] = d62
			ps82.OverlayValues[66] = d66
			ps82.OverlayValues[67] = d67
			ps82.OverlayValues[68] = d68
			ps82.OverlayValues[69] = d69
			ps82.OverlayValues[70] = d70
			ps82.OverlayValues[71] = d71
			ps82.OverlayValues[72] = d72
			ps82.OverlayValues[73] = d73
			ps82.OverlayValues[75] = d75
			ps82.OverlayValues[76] = d76
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			alloc83 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps82)
			}
			ctx.RestoreAllocState(alloc83)
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps81)
			}
			return result
			ctx.FreeDesc(&d77)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
				d62 = ps.OverlayValues[62]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
				d78 = ps.OverlayValues[78]
			}
			ctx.ReclaimUntrackedRegs()
			d84 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d84, &result)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
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
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
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
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
				d62 = ps.OverlayValues[62]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d85 JITValueDesc
			if d0.Loc == LocImm && d1.Loc == LocImm {
				d85 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d1.Imm.Int())}
			} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
				r18 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r18, d0.Reg)
				d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
				ctx.BindReg(r18, &d85)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d85)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d1.Reg)
				d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else {
				r19 := ctx.AllocRegExcept(d0.Reg, d1.Reg)
				ctx.EmitMovRegReg(r19, d0.Reg)
				ctx.EmitAddInt64(r19, d1.Reg)
				d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
				ctx.BindReg(r19, &d85)
			}
			if d85.Loc == LocReg && d0.Loc == LocReg && d85.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d85)
			var d87 JITValueDesc
			if d85.Loc == LocImm && d0.Loc == LocImm {
				d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d85.Imm.Int() - d0.Imm.Int())}
			} else {
				r20 := ctx.AllocReg()
				if d85.Loc == LocImm {
					ctx.EmitMovRegImm64(r20, uint64(d85.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r20, d85.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitSubInt64(r20, RegR11)
				} else {
					ctx.EmitSubInt64(r20, d0.Reg)
				}
				d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
				ctx.BindReg(r20, &d87)
			}
			var d88 JITValueDesc
			if d12.Loc == LocImm && d0.Loc == LocImm {
				d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + d0.Imm.Int())}
			} else {
				r21 := ctx.AllocReg()
				if d12.Loc == LocImm {
					ctx.EmitMovRegImm64(r21, uint64(d12.Imm.Int()))
				} else {
					ctx.EmitMovRegReg(r21, d12.Reg)
				}
				if d0.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.EmitAddInt64(r21, RegR11)
				} else {
					ctx.EmitAddInt64(r21, d0.Reg)
				}
				d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
				ctx.BindReg(r21, &d88)
			}
			var d89 JITValueDesc
			r22 := ctx.AllocReg()
			r23 := ctx.AllocReg()
			if d88.Loc == LocImm {
				ctx.EmitMovRegImm64(r22, uint64(d88.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r22, d88.Reg)
				ctx.FreeReg(d88.Reg)
			}
			if d87.Loc == LocImm {
				ctx.EmitMovRegImm64(r23, uint64(d87.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r23, d87.Reg)
				ctx.FreeReg(d87.Reg)
			}
			d89 = JITValueDesc{Loc: LocRegPair, Reg: r22, Reg2: r23}
			ctx.BindReg(r22, &d89)
			ctx.BindReg(r23, &d89)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d85)
			d90 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d89}, 2)
			ctx.EmitMovPairToResult(&d90, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			ps91 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps91)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(16))
			ctx.EmitAddRSP32(int32(16))
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
			ps4 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps4)
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
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
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
			var d31 JITValueDesc
			_ = d31
			var d32 JITValueDesc
			_ = d32
			var d34 JITValueDesc
			_ = d34
			var d35 JITValueDesc
			_ = d35
			var d36 JITValueDesc
			_ = d36
			var d37 JITValueDesc
			_ = d37
			var d40 JITValueDesc
			_ = d40
			var d41 JITValueDesc
			_ = d41
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
			var d52 JITValueDesc
			_ = d52
			var d53 JITValueDesc
			_ = d53
			var d54 JITValueDesc
			_ = d54
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
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
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, 0)
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
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, 0)
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
			alloc18 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc18)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
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
			d19 = args[2]
			d19.ID = 0
			d21 = d19
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d21.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d21)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d21)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d21)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d21.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d21 = tmpPair
			} else if d21.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d21.Reg), Reg2: ctx.AllocRegExcept(d21.Reg)}
				switch d21.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d21)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d21)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d21)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d21)
				d21 = tmpPair
			} else if d21.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d21.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d21.MemPtr))
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
				d21 = tmpPair
			}
			if d21.Loc != LocRegPair && d21.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d21}, 2)
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d20)
			if d20.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d20.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d20.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d20)
				} else if d20.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d20)
				} else if d20.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d20)
				} else if d20.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d20.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d20.Loc != LocRegPair && d20.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
			}
			d22 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d20}, 2)
			d23 = d22
			if d23.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocRegPair || d23.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d23, 0)
			} else {
				ctx.EmitStoreToStack(d23, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			ps24 := PhiState{General: ps.General}
			ps24.OverlayValues = make([]JITValueDesc, 24)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[3] = d3
			ps24.OverlayValues[4] = d4
			ps24.OverlayValues[5] = d5
			ps24.OverlayValues[6] = d6
			ps24.OverlayValues[7] = d7
			ps24.OverlayValues[8] = d8
			ps24.OverlayValues[9] = d9
			ps24.OverlayValues[10] = d10
			ps24.OverlayValues[11] = d11
			ps24.OverlayValues[14] = d14
			ps24.OverlayValues[17] = d17
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[22] = d22
			ps24.OverlayValues[23] = d23
			ps24.PhiValues = make([]JITValueDesc, 1)
			d25 = d22
			ps24.PhiValues[0] = d25
			if ps24.General && bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps24)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d26 := ps.PhiValues[0]
						ctx.EnsureDesc(&d26)
						ctx.EmitStoreScmerToStack(d26, 0)
					}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
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
			d27 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("_ci")}
			ctx.EnsureDesc(&d27)
			if d27.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d27.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d27.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d27)
				} else if d27.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d27)
				} else if d27.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d27)
				} else if d27.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d27.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d27.Loc != LocRegPair && d27.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.Contains arg1)")
			}
			d28 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Contains), []JITValueDesc{d0, d27}, 1)
			ctx.FreeDesc(&d0)
			d29 = d28
			ctx.EnsureDesc(&d29)
			if d29.Loc != LocImm && d29.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d29.Loc == LocImm {
				if d29.Imm.Bool() {
			ps30 := PhiState{General: ps.General}
			ps30.OverlayValues = make([]JITValueDesc, 30)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[3] = d3
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[5] = d5
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[7] = d7
			ps30.OverlayValues[8] = d8
			ps30.OverlayValues[9] = d9
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[11] = d11
			ps30.OverlayValues[14] = d14
			ps30.OverlayValues[17] = d17
			ps30.OverlayValues[19] = d19
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[21] = d21
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[23] = d23
			ps30.OverlayValues[25] = d25
			ps30.OverlayValues[26] = d26
			ps30.OverlayValues[27] = d27
			ps30.OverlayValues[28] = d28
			ps30.OverlayValues[29] = d29
					return bbs[3].RenderPS(ps30)
				}
			d31 = d4
			if d31.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d31)
			if d31.Loc == LocRegPair || d31.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d31, 16)
			} else {
				ctx.EmitStoreToStack(d31, 16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (16)+8)
			}
			d32 = d7
			if d32.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d32)
			if d32.Loc == LocRegPair || d32.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d32, 32)
			} else {
				ctx.EmitStoreToStack(d32, 32)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (32)+8)
			}
			ps33 := PhiState{General: ps.General}
			ps33.OverlayValues = make([]JITValueDesc, 33)
			ps33.OverlayValues[0] = d0
			ps33.OverlayValues[1] = d1
			ps33.OverlayValues[2] = d2
			ps33.OverlayValues[3] = d3
			ps33.OverlayValues[4] = d4
			ps33.OverlayValues[5] = d5
			ps33.OverlayValues[6] = d6
			ps33.OverlayValues[7] = d7
			ps33.OverlayValues[8] = d8
			ps33.OverlayValues[9] = d9
			ps33.OverlayValues[10] = d10
			ps33.OverlayValues[11] = d11
			ps33.OverlayValues[14] = d14
			ps33.OverlayValues[17] = d17
			ps33.OverlayValues[19] = d19
			ps33.OverlayValues[20] = d20
			ps33.OverlayValues[21] = d21
			ps33.OverlayValues[22] = d22
			ps33.OverlayValues[23] = d23
			ps33.OverlayValues[25] = d25
			ps33.OverlayValues[26] = d26
			ps33.OverlayValues[27] = d27
			ps33.OverlayValues[28] = d28
			ps33.OverlayValues[29] = d29
			ps33.OverlayValues[31] = d31
			ps33.OverlayValues[32] = d32
			ps33.PhiValues = make([]JITValueDesc, 2)
			d34 = d4
			ps33.PhiValues[0] = d34
			d35 = d7
			ps33.PhiValues[1] = d35
				return bbs[4].RenderPS(ps33)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d29.Reg, 0)
			ctx.EmitJcc(CcNE, lbl8)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl9)
			d36 = d4
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			if d36.Loc == LocRegPair || d36.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d36, 16)
			} else {
				ctx.EmitStoreToStack(d36, 16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (16)+8)
			}
			d37 = d7
			if d37.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			if d37.Loc == LocRegPair || d37.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d37, 32)
			} else {
				ctx.EmitStoreToStack(d37, 32)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (32)+8)
			}
			ctx.EmitJmp(lbl5)
			ps38 := PhiState{General: true}
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
			ps38.OverlayValues[19] = d19
			ps38.OverlayValues[20] = d20
			ps38.OverlayValues[21] = d21
			ps38.OverlayValues[22] = d22
			ps38.OverlayValues[23] = d23
			ps38.OverlayValues[25] = d25
			ps38.OverlayValues[26] = d26
			ps38.OverlayValues[27] = d27
			ps38.OverlayValues[28] = d28
			ps38.OverlayValues[29] = d29
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[34] = d34
			ps38.OverlayValues[35] = d35
			ps38.OverlayValues[36] = d36
			ps38.OverlayValues[37] = d37
			ps39 := PhiState{General: true}
			ps39.OverlayValues = make([]JITValueDesc, 38)
			ps39.OverlayValues[0] = d0
			ps39.OverlayValues[1] = d1
			ps39.OverlayValues[2] = d2
			ps39.OverlayValues[3] = d3
			ps39.OverlayValues[4] = d4
			ps39.OverlayValues[5] = d5
			ps39.OverlayValues[6] = d6
			ps39.OverlayValues[7] = d7
			ps39.OverlayValues[8] = d8
			ps39.OverlayValues[9] = d9
			ps39.OverlayValues[10] = d10
			ps39.OverlayValues[11] = d11
			ps39.OverlayValues[14] = d14
			ps39.OverlayValues[17] = d17
			ps39.OverlayValues[19] = d19
			ps39.OverlayValues[20] = d20
			ps39.OverlayValues[21] = d21
			ps39.OverlayValues[22] = d22
			ps39.OverlayValues[23] = d23
			ps39.OverlayValues[25] = d25
			ps39.OverlayValues[26] = d26
			ps39.OverlayValues[27] = d27
			ps39.OverlayValues[28] = d28
			ps39.OverlayValues[29] = d29
			ps39.OverlayValues[31] = d31
			ps39.OverlayValues[32] = d32
			ps39.OverlayValues[34] = d34
			ps39.OverlayValues[35] = d35
			ps39.OverlayValues[36] = d36
			ps39.OverlayValues[37] = d37
			ps39.PhiValues = make([]JITValueDesc, 2)
			d40 = d4
			ps39.PhiValues[0] = d40
			d41 = d7
			ps39.PhiValues[1] = d41
			snap42 := d4
			snap43 := d7
			alloc44 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps39)
			}
			ctx.RestoreAllocState(alloc44)
			d4 = snap42
			d7 = snap43
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps38)
			}
			return result
			ctx.FreeDesc(&d28)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
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
			d45 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d4}, 2)
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
			d46 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d7}, 2)
			d47 = d45
			if d47.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d47)
			if d47.Loc == LocRegPair || d47.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d47, 16)
			} else {
				ctx.EmitStoreToStack(d47, 16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (16)+8)
			}
			d48 = d46
			if d48.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d48)
			if d48.Loc == LocRegPair || d48.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d48, 32)
			} else {
				ctx.EmitStoreToStack(d48, 32)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (32)+8)
			}
			ps49 := PhiState{General: ps.General}
			ps49.OverlayValues = make([]JITValueDesc, 49)
			ps49.OverlayValues[0] = d0
			ps49.OverlayValues[1] = d1
			ps49.OverlayValues[2] = d2
			ps49.OverlayValues[3] = d3
			ps49.OverlayValues[4] = d4
			ps49.OverlayValues[5] = d5
			ps49.OverlayValues[6] = d6
			ps49.OverlayValues[7] = d7
			ps49.OverlayValues[8] = d8
			ps49.OverlayValues[9] = d9
			ps49.OverlayValues[10] = d10
			ps49.OverlayValues[11] = d11
			ps49.OverlayValues[14] = d14
			ps49.OverlayValues[17] = d17
			ps49.OverlayValues[19] = d19
			ps49.OverlayValues[20] = d20
			ps49.OverlayValues[21] = d21
			ps49.OverlayValues[22] = d22
			ps49.OverlayValues[23] = d23
			ps49.OverlayValues[25] = d25
			ps49.OverlayValues[26] = d26
			ps49.OverlayValues[27] = d27
			ps49.OverlayValues[28] = d28
			ps49.OverlayValues[29] = d29
			ps49.OverlayValues[31] = d31
			ps49.OverlayValues[32] = d32
			ps49.OverlayValues[34] = d34
			ps49.OverlayValues[35] = d35
			ps49.OverlayValues[36] = d36
			ps49.OverlayValues[37] = d37
			ps49.OverlayValues[40] = d40
			ps49.OverlayValues[41] = d41
			ps49.OverlayValues[45] = d45
			ps49.OverlayValues[46] = d46
			ps49.OverlayValues[47] = d47
			ps49.OverlayValues[48] = d48
			ps49.PhiValues = make([]JITValueDesc, 2)
			d50 = d45
			ps49.PhiValues[0] = d50
			d51 = d46
			ps49.PhiValues[1] = d51
			if ps49.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps49)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d52 := ps.PhiValues[0]
						ctx.EnsureDesc(&d52)
						ctx.EmitStoreScmerToStack(d52, 16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d53 := ps.PhiValues[1]
						ctx.EnsureDesc(&d53)
						ctx.EmitStoreScmerToStack(d53, 32)
					}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
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
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
				d53 = ps.OverlayValues[53]
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
			d54 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d1, d2}, 1)
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d54)
			ctx.EmitMakeBool(result, d54)
			if d54.Loc == LocReg { ctx.FreeReg(d54.Reg) }
			result.Type = tagBool
			ctx.EmitJmp(lbl0)
			return result
			}
			ps55 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps55)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(48))
			ctx.EmitAddRSP32(int32(48))
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
			ps7 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps7)
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
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
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
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
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
			ps11 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps11)
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
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
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
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
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
			ps6 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps6)
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
			alloc8 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc8)
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
			d9 = args[0]
			d9.ID = 0
			d11 = d9
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d11.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d11)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d11.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d11 = tmpPair
			} else if d11.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
				switch d11.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d11)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d11)
				d11 = tmpPair
			} else if d11.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d11.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.MemPtr))
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
				d11 = tmpPair
			}
			if d11.Loc != LocRegPair && d11.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d10 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d11}, 2)
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d10.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d10.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d10 = tmpPair
			} else if d10.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocRegExcept(d10.Reg), Reg2: ctx.AllocRegExcept(d10.Reg)}
				switch d10.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d10)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d10)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d10)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d10)
				d10 = tmpPair
			}
			if d10.Loc != LocRegPair && d10.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
			}
			d12 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d10}, 2)
			d13 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d12}, 2)
			ctx.EmitMovPairToResult(&d13, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			ps14 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps14)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
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
			alloc8 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc8)
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
			d9 = args[0]
			d9.ID = 0
			d11 = d9
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d11.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d11)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d11.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d11 = tmpPair
			} else if d11.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
				switch d11.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d11)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d11)
				d11 = tmpPair
			} else if d11.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d11.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.MemPtr))
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
				d11 = tmpPair
			}
			if d11.Loc != LocRegPair && d11.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d10 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d11}, 2)
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d10.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d10.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d10 = tmpPair
			} else if d10.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocRegExcept(d10.Reg), Reg2: ctx.AllocRegExcept(d10.Reg)}
				switch d10.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d10)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d10)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d10)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d10)
				d10 = tmpPair
			}
			if d10.Loc != LocRegPair && d10.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
			}
			d12 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d12)
			if d12.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d12.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d12.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d12)
				} else if d12.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d12)
				} else if d12.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d12)
				} else if d12.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d12.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d12 = tmpPair
			} else if d12.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d12.Type, Reg: ctx.AllocRegExcept(d12.Reg), Reg2: ctx.AllocRegExcept(d12.Reg)}
				switch d12.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d12)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d12)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d12)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d12)
				d12 = tmpPair
			}
			if d12.Loc != LocRegPair && d12.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
			}
			d13 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d10, d12}, 2)
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d13}, 2)
			ctx.EmitMovPairToResult(&d14, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			ps15 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps15)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
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
			alloc8 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc8)
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
			d9 = args[0]
			d9.ID = 0
			d11 = d9
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d11.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d11)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d11.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d11 = tmpPair
			} else if d11.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
				switch d11.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d11)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d11)
				d11 = tmpPair
			} else if d11.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d11.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.MemPtr))
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
				d11 = tmpPair
			}
			if d11.Loc != LocRegPair && d11.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d10 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d11}, 2)
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d10.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d10)
				} else if d10.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d10.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d10 = tmpPair
			} else if d10.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocRegExcept(d10.Reg), Reg2: ctx.AllocRegExcept(d10.Reg)}
				switch d10.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d10)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d10)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d10)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d10)
				d10 = tmpPair
			}
			if d10.Loc != LocRegPair && d10.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
			}
			d12 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d12)
			if d12.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d12.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d12.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d12)
				} else if d12.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d12)
				} else if d12.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d12)
				} else if d12.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d12.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d12 = tmpPair
			} else if d12.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d12.Type, Reg: ctx.AllocRegExcept(d12.Reg), Reg2: ctx.AllocRegExcept(d12.Reg)}
				switch d12.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d12)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d12)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d12)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d12)
				d12 = tmpPair
			}
			if d12.Loc != LocRegPair && d12.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
			}
			d13 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d10, d12}, 2)
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d13}, 2)
			ctx.EmitMovPairToResult(&d14, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			ps15 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps15)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
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
		nil /* TODO: MakeSlice: make []string t12 t12 */, /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t12 t12 */ /* TODO: MakeSlice: make []string t2 t2 */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
			var d9 JITValueDesc
			_ = d9
			var d10 JITValueDesc
			_ = d10
			var d11 JITValueDesc
			_ = d11
			var d12 JITValueDesc
			_ = d12
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
			alloc8 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc8)
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
			d9 = args[1]
			d9.ID = 0
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			if d9.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d9.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d9.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d9)
				} else if d9.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d9)
				} else if d9.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d9)
				} else if d9.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d9.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d9 = tmpPair
			} else if d9.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d9.Type, Reg: ctx.AllocRegExcept(d9.Reg), Reg2: ctx.AllocRegExcept(d9.Reg)}
				switch d9.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d9)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d9)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d9)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d9)
				d9 = tmpPair
			}
			if d9.Loc != LocRegPair && d9.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d10 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d9}, 1)
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d10)
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() <= 0)}
			} else {
				r0 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitCmpRegImm32(d10.Reg, 0)
				ctx.EmitSetcc(r0, CcLE)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d11)
			}
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocImm && d12.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
			ps13 := PhiState{General: ps.General}
			ps13.OverlayValues = make([]JITValueDesc, 13)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			ps13.OverlayValues[12] = d12
					return bbs[3].RenderPS(ps13)
				}
			ps14 := PhiState{General: ps.General}
			ps14.OverlayValues = make([]JITValueDesc, 13)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
				return bbs[4].RenderPS(ps14)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d12.Reg, 0)
			ctx.EmitJcc(CcNE, lbl8)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl5)
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 13)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[12] = d12
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 13)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[12] = d12
			alloc17 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc17)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps15)
			}
			return result
			ctx.FreeDesc(&d11)
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
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
				d12 = ps.OverlayValues[12]
			}
			ctx.ReclaimUntrackedRegs()
			d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d18, &result)
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
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			ctx.ReclaimUntrackedRegs()
			d19 = args[0]
			d19.ID = 0
			d21 = d19
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d21.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d21)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d21)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d21)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d21.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d21 = tmpPair
			} else if d21.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d21.Reg), Reg2: ctx.AllocRegExcept(d21.Reg)}
				switch d21.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d21)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d21)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d21)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d21)
				d21 = tmpPair
			} else if d21.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d21.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d21.MemPtr))
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
				d21 = tmpPair
			}
			if d21.Loc != LocRegPair && d21.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d21}, 2)
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d20)
			if d20.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d20.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d20.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d20)
				} else if d20.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d20)
				} else if d20.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d20)
				} else if d20.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d20.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
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
			if d20.Loc != LocRegPair && d20.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (strings.Repeat arg0)")
			}
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocRegPair || d10.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d22 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Repeat), []JITValueDesc{d20, d10}, 2)
			ctx.FreeDesc(&d10)
			d23 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d22}, 2)
			ctx.EmitMovPairToResult(&d23, &result)
			result.Type = tagString
			ctx.EmitJmp(lbl0)
			return result
			}
			ps24 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps24)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
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
		nil /* TODO: Slice on non-desc: slice t0[:0:int] */, /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */
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
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
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
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
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
		nil /* TODO: FieldAddr on non-receiver: &b.addr [#0] */, /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */
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
		nil /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */, /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */
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
		nil /* TODO: MakeClosure binding not an alloc-stored value */, /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */
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
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
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
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
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
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */
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
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */
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
		nil /* TODO: MakeSlice: make []byte t4 t4 */, /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */
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
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */
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
		nil /* TODO: MakeSlice: make []byte t2 t2 */, /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */
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
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
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
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
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
