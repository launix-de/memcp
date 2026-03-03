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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagString, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d1.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d1)
			} else {
				ctx.W.EmitMakeBool(result, d1)
				ctx.FreeReg(d1.Reg)
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
		nil /* TODO: ChangeType: changetype Symbol <- string (t25) */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			d2 := args[1]
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d2}, 1)
			ctx.FreeDesc(&d2)
			d4 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d4)
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() > 2)}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d4.Reg, 2)
				ctx.W.EmitSetcc(r0, CcG)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d5)
			}
			ctx.FreeDesc(&d4)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			ctx.EnsureDesc(&d3)
			var d6 JITValueDesc
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair {
				d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
				ctx.BindReg(d1.Reg2, &d6)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			var d8 JITValueDesc
			if d6.Loc == LocImm && d3.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() - d3.Imm.Int())}
			} else {
				r1 := ctx.AllocReg()
				if d6.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r1, uint64(d6.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r1, d6.Reg)
				}
				if d3.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
					ctx.W.EmitSubInt64(r1, RegR11)
				} else {
					ctx.W.EmitSubInt64(r1, d3.Reg)
				}
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
				ctx.BindReg(r1, &d8)
			}
			var d9 JITValueDesc
			if d1.Loc == LocImm && d3.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d3.Imm.Int())}
			} else {
				r2 := ctx.AllocReg()
				if d1.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r2, uint64(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r2, d1.Reg)
				}
				if d3.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
					ctx.W.EmitAddInt64(r2, RegR11)
				} else {
					ctx.W.EmitAddInt64(r2, d3.Reg)
				}
				d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
				ctx.BindReg(r2, &d9)
			}
			var d10 JITValueDesc
			if d9.Loc == LocImm && d8.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d9.Imm.Int())}
				_ = d8
			} else {
				r3 := ctx.AllocReg()
				r4 := ctx.AllocReg()
				if d9.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r3, uint64(d9.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r3, d9.Reg)
					ctx.FreeReg(d9.Reg)
				}
				if d8.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r4, uint64(d8.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r4, d8.Reg)
					ctx.FreeReg(d8.Reg)
				}
				d10 = JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
				ctx.BindReg(r3, &d10)
				ctx.BindReg(r4, &d10)
			}
			d11 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d10}, 2)
			ctx.EmitMovPairToResult(&d11, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			d12 := args[2]
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocRegPair && d12.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d13 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d12}, 1)
			ctx.FreeDesc(&d12)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d13)
			var d14 JITValueDesc
			if d3.Loc == LocImm && d13.Loc == LocImm {
				d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d13.Imm.Int())}
			} else if d13.Loc == LocImm && d13.Imm.Int() == 0 {
				r5 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r5, d3.Reg)
				d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
				ctx.BindReg(r5, &d14)
			} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
				d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg}
				ctx.BindReg(d13.Reg, &d14)
			} else if d3.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d13.Reg)
				d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else if d13.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(scratch, d3.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d13.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else {
				r6 := ctx.AllocRegExcept(d3.Reg, d13.Reg)
				ctx.W.EmitMovRegReg(r6, d3.Reg)
				ctx.W.EmitAddInt64(r6, d13.Reg)
				d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
				ctx.BindReg(r6, &d14)
			}
			if d14.Loc == LocReg && d3.Loc == LocReg && d14.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d14)
			var d16 JITValueDesc
			if d14.Loc == LocImm && d3.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d14.Imm.Int() - d3.Imm.Int())}
			} else {
				r7 := ctx.AllocReg()
				if d14.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r7, uint64(d14.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r7, d14.Reg)
				}
				if d3.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
					ctx.W.EmitSubInt64(r7, RegR11)
				} else {
					ctx.W.EmitSubInt64(r7, d3.Reg)
				}
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
				ctx.BindReg(r7, &d16)
			}
			var d17 JITValueDesc
			if d1.Loc == LocImm && d3.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d3.Imm.Int())}
			} else {
				r8 := ctx.AllocReg()
				if d1.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r8, uint64(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r8, d1.Reg)
				}
				if d3.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
					ctx.W.EmitAddInt64(r8, RegR11)
				} else {
					ctx.W.EmitAddInt64(r8, d3.Reg)
				}
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d17)
			}
			var d18 JITValueDesc
			if d17.Loc == LocImm && d16.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d17.Imm.Int())}
				_ = d16
			} else {
				r9 := ctx.AllocReg()
				r10 := ctx.AllocReg()
				if d17.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r9, uint64(d17.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r9, d17.Reg)
					ctx.FreeReg(d17.Reg)
				}
				if d16.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r10, uint64(d16.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r10, d16.Reg)
					ctx.FreeReg(d16.Reg)
				}
				d18 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
				ctx.BindReg(r9, &d18)
				ctx.BindReg(r10, &d18)
			}
			ctx.FreeDesc(&d3)
			ctx.FreeDesc(&d14)
			d19 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d18}, 2)
			ctx.EmitMovPairToResult(&d19, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[0]
			if d3.Loc != LocImm && d3.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
			ctx.FreeDesc(&d3)
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d4.Imm.String())))}
			} else {
				ctx.EnsureDesc(&d4)
				if d4.Loc == LocRegPair {
					d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
					ctx.BindReg(d4.Reg2, &d5)
					ctx.BindReg(d4.Reg2, &d5)
				} else if d4.Loc == LocReg {
					d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
					ctx.BindReg(d4.Reg, &d5)
					ctx.BindReg(d4.Reg, &d5)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			d6 := args[1]
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d7 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d6}, 1)
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d7)
			var d8 JITValueDesc
			if d7.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d7.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(scratch, d7.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			}
			if d8.Loc == LocReg && d7.Loc == LocReg && d8.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = LocNone
			}
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d8)
			var d9 JITValueDesc
			if d8.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitCmpRegImm32(d8.Reg, 0)
				ctx.W.EmitSetcc(r1, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d9)
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d9.Loc == LocImm {
				if d9.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
			d10 := d8
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d10)
			ctx.EmitStoreToStack(d10, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl6)
			d11 := d8
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			ctx.EmitStoreToStack(d11, 0)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			d12 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d5)
			var d13 JITValueDesc
			if d12.Loc == LocImm && d5.Loc == LocImm {
				d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() >= d5.Imm.Int())}
			} else if d5.Loc == LocImm {
				r2 := ctx.AllocRegExcept(d12.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d12.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d12.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r2, CcGE)
				d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d13)
			} else if d12.Loc == LocImm {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d5.Reg)
				ctx.W.EmitSetcc(r3, CcGE)
				d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d13)
			} else {
				r4 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitCmpInt64(d12.Reg, d5.Reg)
				ctx.W.EmitSetcc(r4, CcGE)
				d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d13)
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d13.Loc == LocImm {
				if d13.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d13.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d13)
			ctx.W.MarkLabel(lbl4)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl8)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d14)
			var d15 JITValueDesc
			if d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Int() > 2)}
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d14.Reg, 2)
				ctx.W.EmitSetcc(r5, CcG)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d15)
			}
			ctx.FreeDesc(&d14)
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d15.Loc == LocImm {
				if d15.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d15)
			ctx.W.MarkLabel(lbl7)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d16 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d16, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl11)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d12)
			var d17 JITValueDesc
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocRegPair {
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
				ctx.BindReg(d4.Reg2, &d17)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			var d19 JITValueDesc
			if d17.Loc == LocImm && d12.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d17.Imm.Int() - d12.Imm.Int())}
			} else {
				r6 := ctx.AllocReg()
				if d17.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r6, uint64(d17.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r6, d17.Reg)
				}
				if d12.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitSubInt64(r6, RegR11)
				} else {
					ctx.W.EmitSubInt64(r6, d12.Reg)
				}
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
				ctx.BindReg(r6, &d19)
			}
			var d20 JITValueDesc
			if d4.Loc == LocImm && d12.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d12.Imm.Int())}
			} else {
				r7 := ctx.AllocReg()
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r7, uint64(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r7, d4.Reg)
				}
				if d12.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitAddInt64(r7, RegR11)
				} else {
					ctx.W.EmitAddInt64(r7, d12.Reg)
				}
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
				ctx.BindReg(r7, &d20)
			}
			var d21 JITValueDesc
			if d20.Loc == LocImm && d19.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d20.Imm.Int())}
				_ = d19
			} else {
				r8 := ctx.AllocReg()
				r9 := ctx.AllocReg()
				if d20.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r8, uint64(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r8, d20.Reg)
					ctx.FreeReg(d20.Reg)
				}
				if d19.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r9, uint64(d19.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r9, d19.Reg)
					ctx.FreeReg(d19.Reg)
				}
				d21 = JITValueDesc{Loc: LocRegPair, Reg: r8, Reg2: r9}
				ctx.BindReg(r8, &d21)
				ctx.BindReg(r9, &d21)
			}
			d22 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d21}, 2)
			ctx.EmitMovPairToResult(&d22, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d23 := args[2]
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			if d23.Loc != LocRegPair && d23.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d24 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d23}, 1)
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d24)
			var d25 JITValueDesc
			if d12.Loc == LocImm && d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + d24.Imm.Int())}
			} else if d24.Loc == LocImm && d24.Imm.Int() == 0 {
				r10 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r10, d12.Reg)
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
				ctx.BindReg(r10, &d25)
			} else if d12.Loc == LocImm && d12.Imm.Int() == 0 {
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg}
				ctx.BindReg(d24.Reg, &d25)
			} else if d12.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d24.Reg)
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else if d24.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d24.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d24.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else {
				r11 := ctx.AllocRegExcept(d12.Reg, d24.Reg)
				ctx.W.EmitMovRegReg(r11, d12.Reg)
				ctx.W.EmitAddInt64(r11, d24.Reg)
				d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d25)
			}
			if d25.Loc == LocReg && d12.Loc == LocReg && d25.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = LocNone
			}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d5)
			var d26 JITValueDesc
			if d25.Loc == LocImm && d5.Loc == LocImm {
				d26 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d25.Imm.Int() > d5.Imm.Int())}
			} else if d5.Loc == LocImm {
				r12 := ctx.AllocReg()
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d25.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d25.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r12, CcG)
				d26 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
				ctx.BindReg(r12, &d26)
			} else if d25.Loc == LocImm {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d25.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d5.Reg)
				ctx.W.EmitSetcc(r13, CcG)
				d26 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d26)
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d25.Reg, d5.Reg)
				ctx.W.EmitSetcc(r14, CcG)
				d26 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d26)
			}
			ctx.FreeDesc(&d25)
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d26.Loc == LocImm {
				if d26.Imm.Bool() {
					ctx.W.EmitJmp(lbl13)
				} else {
			d27 := d24
			if d27.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, 8)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d26.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl15)
			d28 := d24
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			ctx.EmitStoreToStack(d28, 8)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d26)
			ctx.W.MarkLabel(lbl14)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d29 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d29)
			var d30 JITValueDesc
			if d29.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d29.Imm.Int() < 0)}
			} else {
				r15 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitCmpRegImm32(d29.Reg, 0)
				ctx.W.EmitSetcc(r15, CcL)
				d30 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d30)
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d30.Loc == LocImm {
				if d30.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d30)
			ctx.W.MarkLabel(lbl13)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d29 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d12)
			var d31 JITValueDesc
			if d5.Loc == LocImm && d12.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() - d12.Imm.Int())}
			} else if d12.Loc == LocImm && d12.Imm.Int() == 0 {
				r16 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(r16, d5.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
				ctx.BindReg(r16, &d31)
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d12.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else if d12.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d12.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else {
				r17 := ctx.AllocRegExcept(d5.Reg, d12.Reg)
				ctx.W.EmitMovRegReg(r17, d5.Reg)
				ctx.W.EmitSubInt64(r17, d12.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
				ctx.BindReg(r17, &d31)
			}
			if d31.Loc == LocReg && d5.Loc == LocReg && d31.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d5)
			d32 := d31
			if d32.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d32)
			ctx.EmitStoreToStack(d32, 8)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl17)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d29 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d29)
			var d33 JITValueDesc
			if d12.Loc == LocImm && d29.Loc == LocImm {
				d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + d29.Imm.Int())}
			} else if d29.Loc == LocImm && d29.Imm.Int() == 0 {
				r18 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r18, d12.Reg)
				d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
				ctx.BindReg(r18, &d33)
			} else if d12.Loc == LocImm && d12.Imm.Int() == 0 {
				d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d33)
			} else if d12.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d29.Reg)
				d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d29.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d29.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d29.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r19 := ctx.AllocRegExcept(d12.Reg, d29.Reg)
				ctx.W.EmitMovRegReg(r19, d12.Reg)
				ctx.W.EmitAddInt64(r19, d29.Reg)
				d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
				ctx.BindReg(r19, &d33)
			}
			if d33.Loc == LocReg && d12.Loc == LocReg && d33.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d33)
			var d35 JITValueDesc
			if d33.Loc == LocImm && d12.Loc == LocImm {
				d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d33.Imm.Int() - d12.Imm.Int())}
			} else {
				r20 := ctx.AllocReg()
				if d33.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r20, uint64(d33.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r20, d33.Reg)
				}
				if d12.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitSubInt64(r20, RegR11)
				} else {
					ctx.W.EmitSubInt64(r20, d12.Reg)
				}
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
				ctx.BindReg(r20, &d35)
			}
			var d36 JITValueDesc
			if d4.Loc == LocImm && d12.Loc == LocImm {
				d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d12.Imm.Int())}
			} else {
				r21 := ctx.AllocReg()
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r21, uint64(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r21, d4.Reg)
				}
				if d12.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitAddInt64(r21, RegR11)
				} else {
					ctx.W.EmitAddInt64(r21, d12.Reg)
				}
				d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
				ctx.BindReg(r21, &d36)
			}
			var d37 JITValueDesc
			if d36.Loc == LocImm && d35.Loc == LocImm {
				d37 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d36.Imm.Int())}
				_ = d35
			} else {
				r22 := ctx.AllocReg()
				r23 := ctx.AllocReg()
				if d36.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r22, uint64(d36.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r22, d36.Reg)
					ctx.FreeReg(d36.Reg)
				}
				if d35.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r23, uint64(d35.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r23, d35.Reg)
					ctx.FreeReg(d35.Reg)
				}
				d37 = JITValueDesc{Loc: LocRegPair, Reg: r22, Reg2: r23}
				ctx.BindReg(r22, &d37)
				ctx.BindReg(r23, &d37)
			}
			ctx.FreeDesc(&d12)
			ctx.FreeDesc(&d33)
			d38 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d37}, 2)
			ctx.EmitMovPairToResult(&d38, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl16)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d29 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d39 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d39, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(16))
			ctx.W.EmitAddRSP32(int32(16))
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Simplify), []JITValueDesc{d1}, 2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d2, &result)
			} else {
				switch d2.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d2)
				case tagInt:
					ctx.W.EmitMakeInt(result, d2)
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d2)
				case tagNil:
					ctx.W.EmitMakeNil(result)
				default:
					panic("jit: single-block scalar return with unknown type")
				}
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d1.Imm.String())))}
			} else {
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocRegPair {
					d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
					ctx.BindReg(d1.Reg2, &d2)
					ctx.BindReg(d1.Reg2, &d2)
				} else if d1.Loc == LocReg {
					d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
					ctx.BindReg(d1.Reg, &d2)
					ctx.BindReg(d1.Reg, &d2)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeInt(result, d2)
			} else {
				ctx.W.EmitMakeInt(result, d2)
				ctx.FreeReg(d2.Reg)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			d2 := args[1]
			if d2.Loc != LocImm && d2.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d2)
			d4 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d4)
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() > 2)}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d4.Reg, 2)
				ctx.W.EmitSetcc(r1, CcG)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d5)
			}
			ctx.FreeDesc(&d4)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, 0)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, 0)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			d6 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d7 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("_ci")}
			ctx.EnsureDesc(&d7)
			if d7.Loc != LocRegPair && d7.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d8 := ctx.EmitGoCallScalar(GoFuncAddr(strings.Contains), []JITValueDesc{d6, d7}, 1)
			ctx.FreeDesc(&d6)
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d8.Loc == LocImm {
				if d8.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
			d9 := d1
			if d9.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d9)
			if d9.Loc == LocRegPair || d9.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d9, 16)
			} else {
				ctx.EmitStoreToStack(d9, 16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (16)+8)
			}
			d10 := d3
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocRegPair || d10.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d10, 32)
			} else {
				ctx.EmitStoreToStack(d10, 32)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (32)+8)
			}
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d8.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl6)
			d11 := d1
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocRegPair || d11.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d11, 16)
			} else {
				ctx.EmitStoreToStack(d11, 16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (16)+8)
			}
			d12 := d3
			if d12.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			if d12.Loc == LocRegPair || d12.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d12, 32)
			} else {
				ctx.EmitStoreToStack(d12, 32)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (32)+8)
			}
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d8)
			ctx.W.MarkLabel(lbl1)
			d6 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d13 := args[2]
			if d13.Loc != LocImm && d13.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d14 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d13}, 2)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d14)
			if d14.Loc != LocRegPair && d14.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d15 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d14}, 2)
			d16 := d15
			if d16.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			if d16.Loc == LocRegPair || d16.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d16, 0)
			} else {
				ctx.EmitStoreToStack(d16, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl5)
			d6 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d17 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d17)
			if d17.Loc != LocRegPair && d17.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			if d18.Loc != LocRegPair && d18.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d19 := ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d17, d18}, 1)
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d18)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			ctx.W.EmitMakeBool(result, d19)
			if d19.Loc == LocReg { ctx.FreeReg(d19.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			d6 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d17 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d20 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d1}, 2)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d21 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d3}, 2)
			d22 := d20
			if d22.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			if d22.Loc == LocRegPair || d22.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d22, 16)
			} else {
				ctx.EmitStoreToStack(d22, 16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (16)+8)
			}
			d23 := d21
			if d23.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocRegPair || d23.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d23, 32)
			} else {
				ctx.EmitStoreToStack(d23, 32)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (32)+8)
			}
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(48))
			ctx.W.EmitAddRSP32(int32(48))
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			d2 := args[1]
			if d2.Loc != LocImm && d2.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d1, d3}, 1)
			ctx.EnsureDesc(&d4)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d4.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d4)
			} else {
				ctx.W.EmitMakeBool(result, d4)
				ctx.FreeReg(d4.Reg)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d1}, 2)
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
			if result.Loc == LocAny { return d3 }
			ctx.EmitMovPairToResult(&d3, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d1}, 2)
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
			if result.Loc == LocAny { return d3 }
			ctx.EmitMovPairToResult(&d3, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			d2 := args[1]
			if d2.Loc != LocImm && d2.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
			ctx.FreeDesc(&d2)
			d4 := args[2]
			if d4.Loc != LocImm && d4.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d5 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d4}, 2)
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d6 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ReplaceAll), []JITValueDesc{d1, d3, d5}, 2)
			d7 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d6}, 2)
			if result.Loc == LocAny { return d7 }
			ctx.EmitMovPairToResult(&d7, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d1}, 2)
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
			if result.Loc == LocAny { return d3 }
			ctx.EmitMovPairToResult(&d3, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d2)
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d1, d2}, 2)
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d2)
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d1, d2}, 2)
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[0]
			if d3.Loc != LocImm && d3.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d5 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d4}, 2)
			d6 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d5}, 2)
			ctx.EmitMovPairToResult(&d6, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[0]
			if d3.Loc != LocImm && d3.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d5 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d6 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d4, d5}, 2)
			d7 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d6}, 2)
			ctx.EmitMovPairToResult(&d7, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[0]
			if d3.Loc != LocImm && d3.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d5 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d6 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d4, d5}, 2)
			d7 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d6}, 2)
			ctx.EmitMovPairToResult(&d7, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
		nil /* TODO: MakeSlice: make []string t2 t2 */, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[1]
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d3}, 1)
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() <= 0)}
			} else {
				r0 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitSetcc(r0, CcLE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d5)
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl6)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			d6 := args[0]
			if d6.Loc != LocImm && d6.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d7 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d6}, 2)
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d7)
			if d7.Loc != LocRegPair && d7.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocRegPair || d4.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d8 := ctx.EmitGoCallScalar(GoFuncAddr(strings.Repeat), []JITValueDesc{d7, d4}, 2)
			ctx.FreeDesc(&d4)
			d9 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d8}, 2)
			ctx.EmitMovPairToResult(&d9, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			d10 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d10, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
		nil /* TODO: Slice on non-desc: slice t0[:0:int] */, /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(html.EscapeString), []JITValueDesc{d1}, 2)
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
			if result.Loc == LocAny { return d3 }
			ctx.EmitMovPairToResult(&d3, &result)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(url.QueryEscape), []JITValueDesc{d1}, 2)
			d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
			if result.Loc == LocAny { return d3 }
			ctx.EmitMovPairToResult(&d3, &result)
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
		nil /* TODO: Index: s[t2] */, /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */
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
		nil /* TODO: Defer: defer (*sync.Pool).Put(encodeStatePool, t3) */, /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */
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
		nil /* TODO: MakeClosure binding not an alloc-stored value */, /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */
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
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
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
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
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
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */
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
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$48$1 */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */
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
		nil /* TODO: MakeSlice: make []byte t4 t4 */, /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */
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
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */
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
		nil /* TODO: MakeSlice: make []byte t2 t2 */, /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */
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
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
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
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
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
