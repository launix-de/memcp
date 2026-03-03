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
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d1.Loc == LocImm {
				if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: d1.Imm} }
				ctx.W.EmitMakeBool(result, d1)
			} else {
				if result.Loc == LocAny { return d1 }
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
		nil /* TODO: unsupported compare const kind: 0:float64 */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			d3 := d2
			_ = d3
			r0 := d2.Loc == LocReg
			r1 := d2.Reg
			if r0 { ctx.ProtectReg(r1) }
			var d4 JITValueDesc
			if d3.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int())}
			} else if d3.Type == tagInt && d3.Loc == LocRegPair {
				ctx.FreeReg(d3.Reg)
				d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2}
				ctx.BindReg(d3.Reg2, &d4)
				ctx.BindReg(d3.Reg2, &d4)
			} else if d3.Type == tagInt && d3.Loc == LocReg {
				d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg}
				ctx.BindReg(d3.Reg, &d4)
				ctx.BindReg(d3.Reg, &d4)
			} else {
				d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d3}, 1)
				d4.Type = tagInt
				ctx.BindReg(d4.Reg, &d4)
			}
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if r0 { ctx.UnprotectReg(r1) }
			ctx.FreeDesc(&d2)
			d6 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d6.Loc == LocStack || d6.Loc == LocStackPair { ctx.EnsureDesc(&d6) }
			var d7 JITValueDesc
			if d6.Loc == LocImm {
				d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() > 2)}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d6.Reg, 2)
				ctx.W.EmitSetcc(r2, CcG)
				d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d7)
			}
			ctx.FreeDesc(&d6)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl2)
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			var d8 JITValueDesc
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d1.Loc == LocRegPair {
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
				ctx.BindReg(d1.Reg2, &d8)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			var d10 JITValueDesc
			if d8.Loc == LocImm && d4.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() - d4.Imm.Int())}
			} else {
				r3 := ctx.AllocReg()
				if d8.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r3, uint64(d8.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r3, d8.Reg)
				}
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitSubInt64(r3, RegR11)
				} else {
					ctx.W.EmitSubInt64(r3, d4.Reg)
				}
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
				ctx.BindReg(r3, &d10)
			}
			var d11 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d4.Imm.Int())}
			} else {
				r4 := ctx.AllocReg()
				if d1.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r4, uint64(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r4, d1.Reg)
				}
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitAddInt64(r4, RegR11)
				} else {
					ctx.W.EmitAddInt64(r4, d4.Reg)
				}
				d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
				ctx.BindReg(r4, &d11)
			}
			var d12 JITValueDesc
			if d11.Loc == LocImm && d10.Loc == LocImm {
				d12 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d11.Imm.Int())}
				_ = d10
			} else {
				r5 := ctx.AllocReg()
				r6 := ctx.AllocReg()
				if d11.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r5, uint64(d11.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r5, d11.Reg)
					ctx.FreeReg(d11.Reg)
				}
				if d10.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r6, uint64(d10.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r6, d10.Reg)
					ctx.FreeReg(d10.Reg)
				}
				d12 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d12)
				ctx.BindReg(r6, &d12)
			}
			d13 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d12}, 2)
			ctx.EmitMovPairToResult(&d13, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			d14 := args[2]
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			d15 := d14
			_ = d15
			r7 := d14.Loc == LocReg
			r8 := d14.Reg
			if r7 { ctx.ProtectReg(r8) }
			var d16 JITValueDesc
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
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if r7 { ctx.UnprotectReg(r8) }
			ctx.FreeDesc(&d14)
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			var d18 JITValueDesc
			if d4.Loc == LocImm && d16.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d16.Imm.Int())}
			} else if d16.Loc == LocImm && d16.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r9, d4.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
				ctx.BindReg(r9, &d18)
			} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
				ctx.BindReg(d16.Reg, &d18)
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d16.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else if d16.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else {
				r10 := ctx.AllocRegExcept(d4.Reg, d16.Reg)
				ctx.W.EmitMovRegReg(r10, d4.Reg)
				ctx.W.EmitAddInt64(r10, d16.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
				ctx.BindReg(r10, &d18)
			}
			if d18.Loc == LocReg && d4.Loc == LocReg && d18.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d16)
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d18.Loc == LocStack || d18.Loc == LocStackPair { ctx.EnsureDesc(&d18) }
			var d20 JITValueDesc
			if d18.Loc == LocImm && d4.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d18.Imm.Int() - d4.Imm.Int())}
			} else {
				r11 := ctx.AllocReg()
				if d18.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r11, uint64(d18.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r11, d18.Reg)
				}
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitSubInt64(r11, RegR11)
				} else {
					ctx.W.EmitSubInt64(r11, d4.Reg)
				}
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d20)
			}
			var d21 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d4.Imm.Int())}
			} else {
				r12 := ctx.AllocReg()
				if d1.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r12, uint64(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r12, d1.Reg)
				}
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitAddInt64(r12, RegR11)
				} else {
					ctx.W.EmitAddInt64(r12, d4.Reg)
				}
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d21)
			}
			var d22 JITValueDesc
			if d21.Loc == LocImm && d20.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d21.Imm.Int())}
				_ = d20
			} else {
				r13 := ctx.AllocReg()
				r14 := ctx.AllocReg()
				if d21.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r13, uint64(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r13, d21.Reg)
					ctx.FreeReg(d21.Reg)
				}
				if d20.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r14, uint64(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r14, d20.Reg)
					ctx.FreeReg(d20.Reg)
				}
				d22 = JITValueDesc{Loc: LocRegPair, Reg: r13, Reg2: r14}
				ctx.BindReg(r13, &d22)
				ctx.BindReg(r14, &d22)
			}
			ctx.FreeDesc(&d4)
			ctx.FreeDesc(&d18)
			d23 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d22}, 2)
			ctx.EmitMovPairToResult(&d23, &result)
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
				if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
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
			if d6.Loc == LocStack || d6.Loc == LocStackPair { ctx.EnsureDesc(&d6) }
			d7 := d6
			_ = d7
			r1 := d6.Loc == LocReg
			r2 := d6.Reg
			if r1 { ctx.ProtectReg(r2) }
			var d8 JITValueDesc
			if d7.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d7.Imm.Int())}
			} else if d7.Type == tagInt && d7.Loc == LocRegPair {
				ctx.FreeReg(d7.Reg)
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg2}
				ctx.BindReg(d7.Reg2, &d8)
				ctx.BindReg(d7.Reg2, &d8)
			} else if d7.Type == tagInt && d7.Loc == LocReg {
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg}
				ctx.BindReg(d7.Reg, &d8)
				ctx.BindReg(d7.Reg, &d8)
			} else {
				d8 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d7}, 1)
				d8.Type = tagInt
				ctx.BindReg(d8.Reg, &d8)
			}
			if d8.Loc == LocStack || d8.Loc == LocStackPair { ctx.EnsureDesc(&d8) }
			if d8.Loc == LocStack || d8.Loc == LocStackPair { ctx.EnsureDesc(&d8) }
			if d8.Loc == LocStack || d8.Loc == LocStackPair { ctx.EnsureDesc(&d8) }
			if r1 { ctx.UnprotectReg(r2) }
			ctx.FreeDesc(&d6)
			if d8.Loc == LocStack || d8.Loc == LocStackPair { ctx.EnsureDesc(&d8) }
			var d10 JITValueDesc
			if d8.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() - 1)}
			} else {
				ctx.W.EmitSubRegImm32(d8.Reg, int32(1))
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
				ctx.BindReg(d8.Reg, &d10)
			}
			if d10.Loc == LocReg && d8.Loc == LocReg && d10.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = LocNone
			}
			ctx.FreeDesc(&d8)
			if d10.Loc == LocStack || d10.Loc == LocStackPair { ctx.EnsureDesc(&d10) }
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() < 0)}
			} else {
				r3 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitCmpRegImm32(d10.Reg, 0)
				ctx.W.EmitSetcc(r3, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d11)
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d11.Loc == LocImm {
				if d11.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
			d12 := d10
			if d12.Loc == LocNone { panic("jit: phi source has no location") }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			ctx.EmitStoreToStack(d12, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d11.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl6)
			d13 := d10
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			ctx.EmitStoreToStack(d13, 0)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d11)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			d14 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			var d15 JITValueDesc
			if d14.Loc == LocImm && d5.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Int() >= d5.Imm.Int())}
			} else if d5.Loc == LocImm {
				r4 := ctx.AllocRegExcept(d14.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d14.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d14.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r4, CcGE)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d15)
			} else if d14.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d5.Reg)
				ctx.W.EmitSetcc(r5, CcGE)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d15)
			} else {
				r6 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitCmpInt64(d14.Reg, d5.Reg)
				ctx.W.EmitSetcc(r6, CcGE)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d15)
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d15.Loc == LocImm {
				if d15.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d15)
			ctx.W.MarkLabel(lbl4)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl8)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d16 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			var d17 JITValueDesc
			if d16.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() > 2)}
			} else {
				r7 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d16.Reg, 2)
				ctx.W.EmitSetcc(r7, CcG)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d17)
			}
			ctx.FreeDesc(&d16)
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d17.Loc == LocImm {
				if d17.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d17.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d17)
			ctx.W.MarkLabel(lbl7)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d18 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d18, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl11)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			var d19 JITValueDesc
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d4.Loc == LocRegPair {
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
				ctx.BindReg(d4.Reg2, &d19)
			} else {
				panic("Slice with omitted high requires descriptor with length in Reg2")
			}
			var d21 JITValueDesc
			if d19.Loc == LocImm && d14.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int() - d14.Imm.Int())}
			} else {
				r8 := ctx.AllocReg()
				if d19.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r8, uint64(d19.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r8, d19.Reg)
				}
				if d14.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.W.EmitSubInt64(r8, RegR11)
				} else {
					ctx.W.EmitSubInt64(r8, d14.Reg)
				}
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d21)
			}
			var d22 JITValueDesc
			if d4.Loc == LocImm && d14.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d14.Imm.Int())}
			} else {
				r9 := ctx.AllocReg()
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r9, uint64(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r9, d4.Reg)
				}
				if d14.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.W.EmitAddInt64(r9, RegR11)
				} else {
					ctx.W.EmitAddInt64(r9, d14.Reg)
				}
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
				ctx.BindReg(r9, &d22)
			}
			var d23 JITValueDesc
			if d22.Loc == LocImm && d21.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d22.Imm.Int())}
				_ = d21
			} else {
				r10 := ctx.AllocReg()
				r11 := ctx.AllocReg()
				if d22.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r10, uint64(d22.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r10, d22.Reg)
					ctx.FreeReg(d22.Reg)
				}
				if d21.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r11, uint64(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r11, d21.Reg)
					ctx.FreeReg(d21.Reg)
				}
				d23 = JITValueDesc{Loc: LocRegPair, Reg: r10, Reg2: r11}
				ctx.BindReg(r10, &d23)
				ctx.BindReg(r11, &d23)
			}
			d24 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d23}, 2)
			ctx.EmitMovPairToResult(&d24, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d25 := args[2]
			if d25.Loc == LocStack || d25.Loc == LocStackPair { ctx.EnsureDesc(&d25) }
			d26 := d25
			_ = d26
			r12 := d25.Loc == LocReg
			r13 := d25.Reg
			if r12 { ctx.ProtectReg(r13) }
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
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			if r12 { ctx.UnprotectReg(r13) }
			ctx.FreeDesc(&d25)
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			var d29 JITValueDesc
			if d14.Loc == LocImm && d27.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d14.Imm.Int() + d27.Imm.Int())}
			} else if d27.Loc == LocImm && d27.Imm.Int() == 0 {
				r14 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r14, d14.Reg)
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
				ctx.BindReg(r14, &d29)
			} else if d14.Loc == LocImm && d14.Imm.Int() == 0 {
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg}
				ctx.BindReg(d27.Reg, &d29)
			} else if d14.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d27.Reg)
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else if d27.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(scratch, d14.Reg)
				if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d27.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else {
				r15 := ctx.AllocRegExcept(d14.Reg, d27.Reg)
				ctx.W.EmitMovRegReg(r15, d14.Reg)
				ctx.W.EmitAddInt64(r15, d27.Reg)
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
				ctx.BindReg(r15, &d29)
			}
			if d29.Loc == LocReg && d14.Loc == LocReg && d29.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = LocNone
			}
			if d29.Loc == LocStack || d29.Loc == LocStackPair { ctx.EnsureDesc(&d29) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d29.Loc == LocStack || d29.Loc == LocStackPair { ctx.EnsureDesc(&d29) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d29.Loc == LocStack || d29.Loc == LocStackPair { ctx.EnsureDesc(&d29) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			var d30 JITValueDesc
			if d29.Loc == LocImm && d5.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d29.Imm.Int() > d5.Imm.Int())}
			} else if d5.Loc == LocImm {
				r16 := ctx.AllocReg()
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d29.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d29.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r16, CcG)
				d30 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d30)
			} else if d29.Loc == LocImm {
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d29.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d5.Reg)
				ctx.W.EmitSetcc(r17, CcG)
				d30 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d30)
			} else {
				r18 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d29.Reg, d5.Reg)
				ctx.W.EmitSetcc(r18, CcG)
				d30 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
				ctx.BindReg(r18, &d30)
			}
			ctx.FreeDesc(&d29)
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d30.Loc == LocImm {
				if d30.Imm.Bool() {
					ctx.W.EmitJmp(lbl13)
				} else {
			d31 := d27
			if d31.Loc == LocNone { panic("jit: phi source has no location") }
			if d31.Loc == LocStack || d31.Loc == LocStackPair { ctx.EnsureDesc(&d31) }
			ctx.EmitStoreToStack(d31, 8)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl15)
			d32 := d27
			if d32.Loc == LocNone { panic("jit: phi source has no location") }
			if d32.Loc == LocStack || d32.Loc == LocStackPair { ctx.EnsureDesc(&d32) }
			ctx.EmitStoreToStack(d32, 8)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d30)
			ctx.W.MarkLabel(lbl14)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d33 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if d33.Loc == LocStack || d33.Loc == LocStackPair { ctx.EnsureDesc(&d33) }
			var d34 JITValueDesc
			if d33.Loc == LocImm {
				d34 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() < 0)}
			} else {
				r19 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitCmpRegImm32(d33.Reg, 0)
				ctx.W.EmitSetcc(r19, CcL)
				d34 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
				ctx.BindReg(r19, &d34)
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d34.Loc == LocImm {
				if d34.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d34.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d34)
			ctx.W.MarkLabel(lbl13)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d33 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d5.Loc == LocStack || d5.Loc == LocStackPair { ctx.EnsureDesc(&d5) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			var d35 JITValueDesc
			if d5.Loc == LocImm && d14.Loc == LocImm {
				d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() - d14.Imm.Int())}
			} else if d14.Loc == LocImm && d14.Imm.Int() == 0 {
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
				ctx.BindReg(d5.Reg, &d35)
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d14.Reg)
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d14.Loc == LocImm {
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d5.Reg, int32(d14.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
				ctx.W.EmitSubInt64(d5.Reg, RegR11)
				}
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
				ctx.BindReg(d5.Reg, &d35)
			} else {
				ctx.W.EmitSubInt64(d5.Reg, d14.Reg)
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
				ctx.BindReg(d5.Reg, &d35)
			}
			if d35.Loc == LocReg && d5.Loc == LocReg && d35.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d5)
			d36 := d35
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			if d36.Loc == LocStack || d36.Loc == LocStackPair { ctx.EnsureDesc(&d36) }
			ctx.EmitStoreToStack(d36, 8)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl17)
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d33 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d33.Loc == LocStack || d33.Loc == LocStackPair { ctx.EnsureDesc(&d33) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d33.Loc == LocStack || d33.Loc == LocStackPair { ctx.EnsureDesc(&d33) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d33.Loc == LocStack || d33.Loc == LocStackPair { ctx.EnsureDesc(&d33) }
			var d37 JITValueDesc
			if d14.Loc == LocImm && d33.Loc == LocImm {
				d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d14.Imm.Int() + d33.Imm.Int())}
			} else if d33.Loc == LocImm && d33.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r20, d14.Reg)
				d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
				ctx.BindReg(r20, &d37)
			} else if d14.Loc == LocImm && d14.Imm.Int() == 0 {
				d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg}
				ctx.BindReg(d33.Reg, &d37)
			} else if d14.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d33.Reg)
				d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d37)
			} else if d33.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(scratch, d14.Reg)
				if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d33.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d33.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d37)
			} else {
				r21 := ctx.AllocRegExcept(d14.Reg, d33.Reg)
				ctx.W.EmitMovRegReg(r21, d14.Reg)
				ctx.W.EmitAddInt64(r21, d33.Reg)
				d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
				ctx.BindReg(r21, &d37)
			}
			if d37.Loc == LocReg && d14.Loc == LocReg && d37.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = LocNone
			}
			ctx.FreeDesc(&d33)
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d37.Loc == LocStack || d37.Loc == LocStackPair { ctx.EnsureDesc(&d37) }
			var d39 JITValueDesc
			if d37.Loc == LocImm && d14.Loc == LocImm {
				d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() - d14.Imm.Int())}
			} else {
				r22 := ctx.AllocReg()
				if d37.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r22, uint64(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r22, d37.Reg)
				}
				if d14.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.W.EmitSubInt64(r22, RegR11)
				} else {
					ctx.W.EmitSubInt64(r22, d14.Reg)
				}
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r22}
				ctx.BindReg(r22, &d39)
			}
			var d40 JITValueDesc
			if d4.Loc == LocImm && d14.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d14.Imm.Int())}
			} else {
				r23 := ctx.AllocReg()
				if d4.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r23, uint64(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r23, d4.Reg)
				}
				if d14.Loc == LocImm {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.W.EmitAddInt64(r23, RegR11)
				} else {
					ctx.W.EmitAddInt64(r23, d14.Reg)
				}
				d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r23}
				ctx.BindReg(r23, &d40)
			}
			var d41 JITValueDesc
			if d40.Loc == LocImm && d39.Loc == LocImm {
				d41 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewInt(d40.Imm.Int())}
				_ = d39
			} else {
				r24 := ctx.AllocReg()
				r25 := ctx.AllocReg()
				if d40.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r24, uint64(d40.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r24, d40.Reg)
					ctx.FreeReg(d40.Reg)
				}
				if d39.Loc == LocImm {
					ctx.W.EmitMovRegImm64(r25, uint64(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r25, d39.Reg)
					ctx.FreeReg(d39.Reg)
				}
				d41 = JITValueDesc{Loc: LocRegPair, Reg: r24, Reg2: r25}
				ctx.BindReg(r24, &d41)
				ctx.BindReg(r25, &d41)
			}
			ctx.FreeDesc(&d14)
			ctx.FreeDesc(&d37)
			d42 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d41}, 2)
			ctx.EmitMovPairToResult(&d42, &result)
			result.Type = tagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl16)
			d33 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d14 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d43 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
			ctx.EmitMovPairToResult(&d43, &result)
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
		nil /* TODO: Index: s[0:int] */, /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */ /* TODO: Index: s[0:int] */
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
				if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
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
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d2.Loc == LocImm {
				if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: d2.Imm} }
				ctx.W.EmitMakeInt(result, d2)
			} else {
				if result.Loc == LocAny { return d2 }
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
		nil /* TODO: Index: substr[0:int] */, /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */ /* TODO: Index: substr[0:int] */
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
		nil /* TODO: Index: t1[0:int] */, /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */ /* TODO: Index: t1[0:int] */
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
		nil /* TODO: Index: s[t1] */, /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
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
		nil /* TODO: Index: s[t1] */, /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
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
		nil /* TODO: Range: range s */, /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */ /* TODO: Range: range s */
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
		nil /* TODO: Index: s[t3] */, /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */
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
		nil /* TODO: unsupported compare const kind: "":string */, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
		nil /* TODO: unsupported compare const kind: "":string */, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
		nil /* TODO: Index: s[t3] */, /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */ /* TODO: Index: s[t3] */
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
		nil /* TODO: unsupported compare const kind: "":string */, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
		nil /* TODO: unsupported compare const kind: "":string */, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
		nil /* TODO: unsupported compare const kind: "":string */, /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */ /* TODO: unsupported compare const kind: "":string */
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
		nil /* TODO: Extract: extract t9 #0 */, /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */ /* TODO: Extract: extract t9 #0 */
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
		nil /* TODO: FieldAddr on non-receiver: &re.prog [#1] */, /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */ /* TODO: FieldAddr on non-receiver: &re.prog [#1] */
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
		nil /* TODO: FieldAddr on non-receiver: &r.once [#0] */, /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */ /* TODO: FieldAddr on non-receiver: &r.once [#0] */
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
		nil /* TODO: Index: s[t2] */, /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */
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
		nil /* TODO: Index: s[t2] */, /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */ /* TODO: Index: s[t2] */
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
		nil /* TODO: unresolved SSA value: encoding/json.encodeStatePool */, /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */ /* TODO: unresolved SSA value: encoding/json.encodeStatePool */
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
		nil /* TODO: MakeClosure binding not an alloc-stored value */, /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */ /* TODO: MakeClosure binding not an alloc-stored value */
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
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
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
		nil /* TODO: unsupported Convert string → []byte */, /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */ /* TODO: unsupported Convert string → []byte */
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
		nil /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */, /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */ /* TODO: FieldAddr on non-receiver: &enc.padChar [#2] */
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
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.init_strings$33$1 */
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
		nil /* TODO: MakeSlice: make []byte t4 t4 */, /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */ /* TODO: MakeSlice: make []byte t4 t4 */
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
		nil /* TODO: MakeSlice: make []byte t1 t1 */, /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */ /* TODO: MakeSlice: make []byte t1 t1 */
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
		nil /* TODO: MakeSlice: make []byte t2 t2 */, /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */ /* TODO: MakeSlice: make []byte t2 t2 */
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
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
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
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
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
