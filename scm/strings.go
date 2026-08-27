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
import "bytes"
import "sync"
import "regexp"
import "unicode"
import "unsafe"
import "net/url"
import "strings"
import "unicode/utf8"
import crand "crypto/rand"
import "crypto/sha1"
import "encoding/hex"
import "crypto/sha256"
import "encoding/json"
import "encoding/base64"
import "github.com/google/uuid"
import "golang.org/x/text/collate"
import "golang.org/x/text/language"

// Collation metadata registry for stable serialization of comparator closures.
// Keyed by function pointer.
var collateRegistry sync.Map // map[uintptr]struct{Collation string; Reverse bool}

// collateLessRegistry holds the allocation-free executor belonging to a
// canonical collate callback. The callback remains the public identity and the
// single source of ordering semantics.
var collateLessRegistry sync.Map // map[uintptr]func(Scmer, Scmer) bool

// FunctionIdentity returns the runtime identity of a function value, including
// its closure context. reflect.Value.Pointer only returns the shared code entry
// and therefore aliases distinct collation closures.
func FunctionIdentity(fn func(...Scmer) Scmer) uintptr {
	return *(*uintptr)(unsafe.Pointer(&fn))
}

type collateCacheKey struct {
	Collation string
	Reverse   bool
}

// collateCache canonicalizes order relations. Auto indexes compare callback
// pointers, so equivalent plans must receive the same function instance.
var collateCache sync.Map // map[collateCacheKey]Scmer

func optimizeFNVHash(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	if len(v) == 2 {
		if producer, ok := scmerSlice(v[1]); ok && len(producer) == 2 {
			if declaration := DeclarationForValue(producer[0]); declaration != nil {
				serialize := declaration.Name == "serialize"
				if serialize || declaration.Name == "string" {
					original := NewSlice(v)
					rewritten := NewSlice([]Scmer{
						NewSymbol("stable_structural_hash"),
						producer[1],
						NewBool(serialize),
					})
					if result, td, ok := oc.OptimizeRewrite(original, rewritten, useResult, OptimizerRewriteContract{
						Name:             "fnv_hash_stream_fusion",
						PreconditionsMet: true,
						MaxGrowthNodes:   0,
					}); ok {
						return result, td
					}
				}
			}
		}
	}
	return oc.ApplyDefaultOptimization(v, useResult)
}

func formatStructuralHash(hash uint64) string {
	const digits = "0123456789abcdef"
	var result [16]byte
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = digits[hash&15]
		hash >>= 4
	}
	return string(result[:])
}

func fnvHashString(value string) string {
	hash := fnv64Offset
	for i := 0; i < len(value); i++ {
		hash = (hash ^ uint64(value[i])) * fnv64Prime
	}
	return formatStructuralHash(hash)
}

// (no additional globals needed)

// LookupCollate returns (collation, reverse, ok) for a previously built collate closure.
func LookupCollate(fn func(...Scmer) Scmer) (string, bool, bool) {
	if fn == nil {
		return "", false, false
	}
	if v, ok := collateRegistry.Load(FunctionIdentity(fn)); ok {
		m := v.(struct {
			Collation string
			Reverse   bool
		})
		return m.Collation, m.Reverse, true
	}
	return "", false, false
}

// OrderRelationLess resolves the bool executor of an order callback once. The
// fallback preserves arbitrary user callbacks; factory callbacks avoid a
// Scmer(bool) roundtrip in storage sort loops.
func OrderRelationLess(fn func(...Scmer) Scmer) func(Scmer, Scmer) bool {
	if fast, ok := collateLessRegistry.Load(FunctionIdentity(fn)); ok {
		return fast.(func(Scmer, Scmer) bool)
	}
	return func(a, b Scmer) bool { return ToBool(fn(a, b)) }
}

/* SQL LIKE operator implementation on strings */
func StrLike(str, pattern string) bool {
	if !strings.ContainsAny(pattern, "%_\\") {
		return str == pattern
	}
	if !strings.ContainsAny(pattern, "_\\") {
		wildcards := strings.Count(pattern, "%")
		if wildcards == 1 {
			if pattern[0] == '%' {
				return strings.HasSuffix(str, pattern[1:])
			}
			if pattern[len(pattern)-1] == '%' {
				return strings.HasPrefix(str, pattern[:len(pattern)-1])
			}
		}
		if wildcards == 2 && pattern[0] == '%' && pattern[len(pattern)-1] == '%' {
			return strings.Contains(str, pattern[1:len(pattern)-1])
		}
	}
	type likePosition struct {
		str     int
		pattern int
	}
	memo := make(map[likePosition]bool)
	visited := make(map[likePosition]bool)
	var match func(int, int) bool
	match = func(strPos, patternPos int) bool {
		position := likePosition{str: strPos, pattern: patternPos}
		if visited[position] {
			return memo[position]
		}
		visited[position] = true
		matched := false
		if patternPos == len(pattern) {
			matched = strPos == len(str)
		} else {
			patternRune, patternSize := utf8.DecodeRuneInString(pattern[patternPos:])
			switch patternRune {
			case '%':
				matched = match(strPos, patternPos+patternSize)
				if !matched && strPos < len(str) {
					_, strSize := utf8.DecodeRuneInString(str[strPos:])
					matched = match(strPos+strSize, patternPos)
				}
			case '_':
				if strPos < len(str) {
					_, strSize := utf8.DecodeRuneInString(str[strPos:])
					matched = match(strPos+strSize, patternPos+patternSize)
				}
			case '\\':
				literalPos := patternPos + patternSize
				literalRune := patternRune
				literalSize := patternSize
				if literalPos < len(pattern) {
					literalRune, literalSize = utf8.DecodeRuneInString(pattern[literalPos:])
				} else {
					literalPos = patternPos
				}
				if strPos < len(str) {
					strRune, strSize := utf8.DecodeRuneInString(str[strPos:])
					matched = strRune == literalRune && match(strPos+strSize, literalPos+literalSize)
				}
			default:
				if strPos < len(str) {
					strRune, strSize := utf8.DecodeRuneInString(str[strPos:])
					matched = strRune == patternRune && match(strPos+strSize, patternPos+patternSize)
				}
			}
		}
		memo[position] = matched
		return matched
	}
	return match(0, 0)
}

// StrLikeFold retains the established Unicode lower-case behavior. StrLike's
// fast paths then dispatch exact, prefix, suffix and contains patterns to Go's
// optimized string primitives.
func StrLikeFold(str, pattern string) bool {
	return StrLike(strings.ToLower(str), strings.ToLower(pattern))
}

func likePatternNeedsCaseFold(pattern string) bool {
	if strings.Contains(pattern, "_") {
		return true
	}
	for _, r := range pattern {
		if unicode.ToLower(r) != unicode.ToUpper(r) {
			return true
		}
	}
	return false
}

// StrLikeCollation is the canonical LIKE implementation shared by the Scheme
// builtin and storage match indexes. Keeping both paths here guarantees that an
// exact cached match set has the same case semantics as residual evaluation.
func StrLikeCollation(str, pattern, collation string) bool {
	if strings.Contains(strings.ToLower(collation), "_ci") {
		// Numeric and punctuation-only patterns cannot be affected by case
		// folding. Avoid allocating and walking a potentially large text value.
		// Keep '_' on the folded path because folding may change its byte width.
		if !likePatternNeedsCaseFold(pattern) {
			return StrLike(str, pattern)
		}
		return StrLikeFold(str, pattern)
	}
	return StrLike(str, pattern)
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
	declareSQLLiteralParameterizer()
	// string functions
	DeclareTitle("Strings")

	Declare(&Globalenv, &Declaration{
		Name: "string?",

		Fn: func(a ...Scmer) Scmer {
			_, ok := a[0].Any().(string)
			return NewBool(ok)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells if the value is a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "concat",

		Fn: func(a ...Scmer) Scmer {
			var sb strings.Builder
			for _, s := range a {
				if s.IsNil() {
					return NewNil()
				}
				if stream, ok := s.Any().(io.Reader); ok {
					_, _ = io.Copy(&sb, stream)
				} else {
					sb.WriteString(String(s))
				}
			}
			return NewString(sb.String())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "concatenates stringable values and returns a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "first value to concat"}, &TypeDescriptor{Kind: "any", Label: "more...", Description: "additional values to concat", Variadic: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_concat",

		Fn: func(a ...Scmer) Scmer {
			return Globalenv.Vars["concat"].Func()(a...)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "SQL CONCAT semantics: returns NULL if any argument is NULL",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "first value to concat"}, &TypeDescriptor{Kind: "any", Label: "more...", Description: "additional values to concat", Variadic: true}},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "substr",

		Fn: func(a ...Scmer) Scmer {
			s := String(a[0])
			i := ToInt(a[1])
			if len(a) > 2 {
				return NewString(s[i : i+ToInt(a[2])])
			}
			return NewString(s[i:])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a substring (0-based index)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to cut"}, &TypeDescriptor{Kind: "number", Label: "start", Description: "first character index (0-based)"}, &TypeDescriptor{Kind: "number", Label: "len", Description: "optional length", Optional: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
					ctx.BindReg(d4.Reg, &d4)
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
					ctx.BindReg(d22.Reg, &d22)
					ctx.FreeDesc(&d21)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d4)
					ctx.ProtectReg(d4.Reg)
					ctx.EnsureDesc(&d22)
					ctx.UnprotectReg(d4.Reg)
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
				ps36 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps36)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				for _, r := range argPinned35 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_substr",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "SQL SUBSTR/SUBSTRING with 1-based index and bounds checking",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to cut"}, &TypeDescriptor{Kind: "number", Label: "start", Description: "first character position (1-based)"}, &TypeDescriptor{Kind: "number", Label: "len", Description: "optional length", Optional: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d28 JITValueDesc
				_ = d28
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d34 JITValueDesc
				_ = d34
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d62 JITValueDesc
				_ = d62
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
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
				var d133 JITValueDesc
				_ = d133
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d139 JITValueDesc
				_ = d139
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
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
				var d193 JITValueDesc
				_ = d193
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
				var d253 JITValueDesc
				_ = d253
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				var bbs [13]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiCount = uint16(1)
				bbs[10].PhiBase = int32(phiBase0) + int32(16)
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
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJcc(CcNE, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl15)
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
					ctx.EmitMakeNil(result)
					result.Type = tagNil
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
					ctx.ReclaimUntrackedRegs()
					d18 = args[0]
					d18.ID = 0
					d20 = d18
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d20.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d20)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d20)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d20)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d20.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d20 = tmpPair
					} else if d20.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d20.Reg), Reg2: ctx.AllocRegExcept(d20.Reg)}
						switch d20.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d20)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d20)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d20)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d20)
						d20 = tmpPair
					} else if d20.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d20.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d20.MemPtr))
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
						d20 = tmpPair
					}
					if d20.Loc != LocRegPair && d20.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d20}, 2)
					ctx.FreeDesc(&d18)
					var d21 JITValueDesc
					if d19.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d19.Imm.String())))}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair {
							d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2}
							ctx.BindReg(d19.Reg2, &d21)
							ctx.BindReg(d19.Reg2, &d21)
						} else if d19.Loc == LocReg {
							d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg}
							ctx.BindReg(d19.Reg, &d21)
							ctx.BindReg(d19.Reg, &d21)
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d22 = args[1]
					d22.ID = 0
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d22.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d22.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d22)
						} else if d22.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d22)
						} else if d22.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d22)
						} else if d22.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d22.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d22 = tmpPair
					} else if d22.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d22.Type, Reg: ctx.AllocRegExcept(d22.Reg), Reg2: ctx.AllocRegExcept(d22.Reg)}
						switch d22.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d22)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d22)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d22)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d22)
						d22 = tmpPair
					}
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (ToInt arg0)")
					}
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d22}, 1)
					ctx.BindReg(d23.Reg, &d23)
					ctx.FreeDesc(&d22)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					var d24 JITValueDesc
					if d23.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitMovRegReg(scratch, d23.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d24)
					}
					if d24.Loc == LocReg && d23.Loc == LocReg && d24.Reg == d23.Reg {
						ctx.TransferReg(d23.Reg)
						d23.Loc = LocNone
					}
					ctx.FreeDesc(&d23)
					ctx.EnsureDesc(&d24)
					var d25 JITValueDesc
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d24.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitCmpRegImm32(d24.Reg, 0)
						ctx.EmitSetcc(r0, CcL)
						d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d25)
					}
					d26 = d25
					ctx.EnsureDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
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
							ps27.OverlayValues[22] = d22
							ps27.OverlayValues[23] = d23
							ps27.OverlayValues[24] = d24
							ps27.OverlayValues[25] = d25
							ps27.OverlayValues[26] = d26
							return bbs[3].RenderPS(ps27)
						}
						ctx.EnsureDesc(&d24)
						if d24.Loc == LocReg {
							ctx.ProtectReg(d24.Reg)
						} else if d24.Loc == LocRegPair {
							ctx.ProtectReg(d24.Reg)
							ctx.ProtectReg(d24.Reg2)
						}
						d28 = d24
						if d28.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d28)
						ctx.EmitStoreToStack(d28, int32(bbs[4].PhiBase)+int32(0))
						if d24.Loc == LocReg {
							ctx.UnprotectReg(d24.Reg)
						} else if d24.Loc == LocRegPair {
							ctx.UnprotectReg(d24.Reg)
							ctx.UnprotectReg(d24.Reg2)
						}
						ps29 := PhiState{General: ps.General}
						ps29.OverlayValues = make([]JITValueDesc, 29)
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
						ps29.OverlayValues[22] = d22
						ps29.OverlayValues[23] = d23
						ps29.OverlayValues[24] = d24
						ps29.OverlayValues[25] = d25
						ps29.OverlayValues[26] = d26
						ps29.OverlayValues[28] = d28
						ps29.PhiValues = make([]JITValueDesc, 1)
						d30 = d24
						ps29.PhiValues[0] = d30
						return bbs[4].RenderPS(ps29)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					ctx.EmitJcc(CcNE, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl17)
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocReg {
						ctx.ProtectReg(d24.Reg)
					} else if d24.Loc == LocRegPair {
						ctx.ProtectReg(d24.Reg)
						ctx.ProtectReg(d24.Reg2)
					}
					d31 = d24
					if d31.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d31)
					ctx.EmitStoreToStack(d31, int32(bbs[4].PhiBase)+int32(0))
					if d24.Loc == LocReg {
						ctx.UnprotectReg(d24.Reg)
					} else if d24.Loc == LocRegPair {
						ctx.UnprotectReg(d24.Reg)
						ctx.UnprotectReg(d24.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ps32 := PhiState{General: true}
					ps32.OverlayValues = make([]JITValueDesc, 32)
					ps32.OverlayValues[1] = d1
					ps32.OverlayValues[2] = d2
					ps32.OverlayValues[3] = d3
					ps32.OverlayValues[4] = d4
					ps32.OverlayValues[5] = d5
					ps32.OverlayValues[6] = d6
					ps32.OverlayValues[18] = d18
					ps32.OverlayValues[19] = d19
					ps32.OverlayValues[20] = d20
					ps32.OverlayValues[21] = d21
					ps32.OverlayValues[22] = d22
					ps32.OverlayValues[23] = d23
					ps32.OverlayValues[24] = d24
					ps32.OverlayValues[25] = d25
					ps32.OverlayValues[26] = d26
					ps32.OverlayValues[28] = d28
					ps32.OverlayValues[30] = d30
					ps32.OverlayValues[31] = d31
					ps33 := PhiState{General: true}
					ps33.OverlayValues = make([]JITValueDesc, 32)
					ps33.OverlayValues[1] = d1
					ps33.OverlayValues[2] = d2
					ps33.OverlayValues[3] = d3
					ps33.OverlayValues[4] = d4
					ps33.OverlayValues[5] = d5
					ps33.OverlayValues[6] = d6
					ps33.OverlayValues[18] = d18
					ps33.OverlayValues[19] = d19
					ps33.OverlayValues[20] = d20
					ps33.OverlayValues[21] = d21
					ps33.OverlayValues[22] = d22
					ps33.OverlayValues[23] = d23
					ps33.OverlayValues[24] = d24
					ps33.OverlayValues[25] = d25
					ps33.OverlayValues[26] = d26
					ps33.OverlayValues[28] = d28
					ps33.OverlayValues[30] = d30
					ps33.OverlayValues[31] = d31
					ps33.PhiValues = make([]JITValueDesc, 1)
					d34 = d24
					ps33.PhiValues[0] = d34
					snap35 := d1
					snap36 := d2
					snap37 := d3
					snap38 := d4
					snap39 := d5
					snap40 := d6
					snap41 := d18
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d24
					snap48 := d25
					snap49 := d26
					snap50 := d28
					snap51 := d30
					snap52 := d31
					snap53 := d34
					alloc54 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps33)
					}
					ctx.RestoreAllocState(alloc54)
					d1 = snap35
					d2 = snap36
					d3 = snap37
					d4 = snap38
					d5 = snap39
					d6 = snap40
					d18 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d26 = snap49
					d28 = snap50
					d30 = snap51
					d31 = snap52
					d34 = snap53
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps32)
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ps55 := PhiState{General: ps.General}
					ps55.OverlayValues = make([]JITValueDesc, 35)
					ps55.OverlayValues[1] = d1
					ps55.OverlayValues[2] = d2
					ps55.OverlayValues[3] = d3
					ps55.OverlayValues[4] = d4
					ps55.OverlayValues[5] = d5
					ps55.OverlayValues[6] = d6
					ps55.OverlayValues[18] = d18
					ps55.OverlayValues[19] = d19
					ps55.OverlayValues[20] = d20
					ps55.OverlayValues[21] = d21
					ps55.OverlayValues[22] = d22
					ps55.OverlayValues[23] = d23
					ps55.OverlayValues[24] = d24
					ps55.OverlayValues[25] = d25
					ps55.OverlayValues[26] = d26
					ps55.OverlayValues[28] = d28
					ps55.OverlayValues[30] = d30
					ps55.OverlayValues[31] = d31
					ps55.OverlayValues[34] = d34
					ps55.PhiValues = make([]JITValueDesc, 1)
					d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps55.PhiValues[0] = d56
					if ps55.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps55)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d57 := ps.PhiValues[0]
							ctx.EnsureDesc(&d57)
							ctx.EmitStoreToStack(d57, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d21)
					var d58 JITValueDesc
					if d1.Loc == LocImm && d21.Loc == LocImm {
						d58 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() >= d21.Imm.Int())}
					} else if d21.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d1.Reg)
						if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d21.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CcGE)
						d58 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d58)
					} else if d1.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d21.Reg)
						ctx.EmitSetcc(r2, CcGE)
						d58 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d58)
					} else {
						r3 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d21.Reg)
						ctx.EmitSetcc(r3, CcGE)
						d58 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d58)
					}
					d59 = d58
					ctx.EnsureDesc(&d59)
					if d59.Loc != LocImm && d59.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d59.Loc == LocImm {
						if d59.Imm.Bool() {
							ps60 := PhiState{General: ps.General}
							ps60.OverlayValues = make([]JITValueDesc, 60)
							ps60.OverlayValues[1] = d1
							ps60.OverlayValues[2] = d2
							ps60.OverlayValues[3] = d3
							ps60.OverlayValues[4] = d4
							ps60.OverlayValues[5] = d5
							ps60.OverlayValues[6] = d6
							ps60.OverlayValues[18] = d18
							ps60.OverlayValues[19] = d19
							ps60.OverlayValues[20] = d20
							ps60.OverlayValues[21] = d21
							ps60.OverlayValues[22] = d22
							ps60.OverlayValues[23] = d23
							ps60.OverlayValues[24] = d24
							ps60.OverlayValues[25] = d25
							ps60.OverlayValues[26] = d26
							ps60.OverlayValues[28] = d28
							ps60.OverlayValues[30] = d30
							ps60.OverlayValues[31] = d31
							ps60.OverlayValues[34] = d34
							ps60.OverlayValues[56] = d56
							ps60.OverlayValues[57] = d57
							ps60.OverlayValues[58] = d58
							ps60.OverlayValues[59] = d59
							return bbs[5].RenderPS(ps60)
						}
						ps61 := PhiState{General: ps.General}
						ps61.OverlayValues = make([]JITValueDesc, 60)
						ps61.OverlayValues[1] = d1
						ps61.OverlayValues[2] = d2
						ps61.OverlayValues[3] = d3
						ps61.OverlayValues[4] = d4
						ps61.OverlayValues[5] = d5
						ps61.OverlayValues[6] = d6
						ps61.OverlayValues[18] = d18
						ps61.OverlayValues[19] = d19
						ps61.OverlayValues[20] = d20
						ps61.OverlayValues[21] = d21
						ps61.OverlayValues[22] = d22
						ps61.OverlayValues[23] = d23
						ps61.OverlayValues[24] = d24
						ps61.OverlayValues[25] = d25
						ps61.OverlayValues[26] = d26
						ps61.OverlayValues[28] = d28
						ps61.OverlayValues[30] = d30
						ps61.OverlayValues[31] = d31
						ps61.OverlayValues[34] = d34
						ps61.OverlayValues[56] = d56
						ps61.OverlayValues[57] = d57
						ps61.OverlayValues[58] = d58
						ps61.OverlayValues[59] = d59
						return bbs[6].RenderPS(ps61)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d62 := ps.PhiValues[0]
							ctx.EnsureDesc(&d62)
							ctx.EmitStoreToStack(d62, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d59.Reg, 0)
					ctx.EmitJcc(CcNE, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl19)
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
					ps63.OverlayValues[24] = d24
					ps63.OverlayValues[25] = d25
					ps63.OverlayValues[26] = d26
					ps63.OverlayValues[28] = d28
					ps63.OverlayValues[30] = d30
					ps63.OverlayValues[31] = d31
					ps63.OverlayValues[34] = d34
					ps63.OverlayValues[56] = d56
					ps63.OverlayValues[57] = d57
					ps63.OverlayValues[58] = d58
					ps63.OverlayValues[59] = d59
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
					ps64.OverlayValues[24] = d24
					ps64.OverlayValues[25] = d25
					ps64.OverlayValues[26] = d26
					ps64.OverlayValues[28] = d28
					ps64.OverlayValues[30] = d30
					ps64.OverlayValues[31] = d31
					ps64.OverlayValues[34] = d34
					ps64.OverlayValues[56] = d56
					ps64.OverlayValues[57] = d57
					ps64.OverlayValues[58] = d58
					ps64.OverlayValues[59] = d59
					ps64.OverlayValues[62] = d62
					snap65 := d1
					snap66 := d2
					snap67 := d3
					snap68 := d4
					snap69 := d5
					snap70 := d6
					snap71 := d18
					snap72 := d19
					snap73 := d20
					snap74 := d21
					snap75 := d22
					snap76 := d23
					snap77 := d24
					snap78 := d25
					snap79 := d26
					snap80 := d28
					snap81 := d30
					snap82 := d31
					snap83 := d34
					snap84 := d56
					snap85 := d57
					snap86 := d58
					snap87 := d59
					snap88 := d62
					alloc89 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps64)
					}
					ctx.RestoreAllocState(alloc89)
					d1 = snap65
					d2 = snap66
					d3 = snap67
					d4 = snap68
					d5 = snap69
					d6 = snap70
					d18 = snap71
					d19 = snap72
					d20 = snap73
					d21 = snap74
					d22 = snap75
					d23 = snap76
					d24 = snap77
					d25 = snap78
					d26 = snap79
					d28 = snap80
					d30 = snap81
					d31 = snap82
					d34 = snap83
					d56 = snap84
					d57 = snap85
					d58 = snap86
					d59 = snap87
					d62 = snap88
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps63)
					}
					return result
					ctx.FreeDesc(&d58)
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					ctx.ReclaimUntrackedRegs()
					d90 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
					ctx.EmitMovPairToResult(&d90, &result)
					result.Type = tagString
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					ctx.ReclaimUntrackedRegs()
					d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d91)
					var d92 JITValueDesc
					if d91.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d91.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d91.Reg, 2)
						ctx.EmitSetcc(r4, CcG)
						d92 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d92)
					}
					ctx.FreeDesc(&d91)
					d93 = d92
					ctx.EnsureDesc(&d93)
					if d93.Loc != LocImm && d93.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d93.Loc == LocImm {
						if d93.Imm.Bool() {
							ps94 := PhiState{General: ps.General}
							ps94.OverlayValues = make([]JITValueDesc, 94)
							ps94.OverlayValues[1] = d1
							ps94.OverlayValues[2] = d2
							ps94.OverlayValues[3] = d3
							ps94.OverlayValues[4] = d4
							ps94.OverlayValues[5] = d5
							ps94.OverlayValues[6] = d6
							ps94.OverlayValues[18] = d18
							ps94.OverlayValues[19] = d19
							ps94.OverlayValues[20] = d20
							ps94.OverlayValues[21] = d21
							ps94.OverlayValues[22] = d22
							ps94.OverlayValues[23] = d23
							ps94.OverlayValues[24] = d24
							ps94.OverlayValues[25] = d25
							ps94.OverlayValues[26] = d26
							ps94.OverlayValues[28] = d28
							ps94.OverlayValues[30] = d30
							ps94.OverlayValues[31] = d31
							ps94.OverlayValues[34] = d34
							ps94.OverlayValues[56] = d56
							ps94.OverlayValues[57] = d57
							ps94.OverlayValues[58] = d58
							ps94.OverlayValues[59] = d59
							ps94.OverlayValues[62] = d62
							ps94.OverlayValues[90] = d90
							ps94.OverlayValues[91] = d91
							ps94.OverlayValues[92] = d92
							ps94.OverlayValues[93] = d93
							return bbs[7].RenderPS(ps94)
						}
						ps95 := PhiState{General: ps.General}
						ps95.OverlayValues = make([]JITValueDesc, 94)
						ps95.OverlayValues[1] = d1
						ps95.OverlayValues[2] = d2
						ps95.OverlayValues[3] = d3
						ps95.OverlayValues[4] = d4
						ps95.OverlayValues[5] = d5
						ps95.OverlayValues[6] = d6
						ps95.OverlayValues[18] = d18
						ps95.OverlayValues[19] = d19
						ps95.OverlayValues[20] = d20
						ps95.OverlayValues[21] = d21
						ps95.OverlayValues[22] = d22
						ps95.OverlayValues[23] = d23
						ps95.OverlayValues[24] = d24
						ps95.OverlayValues[25] = d25
						ps95.OverlayValues[26] = d26
						ps95.OverlayValues[28] = d28
						ps95.OverlayValues[30] = d30
						ps95.OverlayValues[31] = d31
						ps95.OverlayValues[34] = d34
						ps95.OverlayValues[56] = d56
						ps95.OverlayValues[57] = d57
						ps95.OverlayValues[58] = d58
						ps95.OverlayValues[59] = d59
						ps95.OverlayValues[62] = d62
						ps95.OverlayValues[90] = d90
						ps95.OverlayValues[91] = d91
						ps95.OverlayValues[92] = d92
						ps95.OverlayValues[93] = d93
						return bbs[8].RenderPS(ps95)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d93.Reg, 0)
					ctx.EmitJcc(CcNE, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl9)
					ps96 := PhiState{General: true}
					ps96.OverlayValues = make([]JITValueDesc, 94)
					ps96.OverlayValues[1] = d1
					ps96.OverlayValues[2] = d2
					ps96.OverlayValues[3] = d3
					ps96.OverlayValues[4] = d4
					ps96.OverlayValues[5] = d5
					ps96.OverlayValues[6] = d6
					ps96.OverlayValues[18] = d18
					ps96.OverlayValues[19] = d19
					ps96.OverlayValues[20] = d20
					ps96.OverlayValues[21] = d21
					ps96.OverlayValues[22] = d22
					ps96.OverlayValues[23] = d23
					ps96.OverlayValues[24] = d24
					ps96.OverlayValues[25] = d25
					ps96.OverlayValues[26] = d26
					ps96.OverlayValues[28] = d28
					ps96.OverlayValues[30] = d30
					ps96.OverlayValues[31] = d31
					ps96.OverlayValues[34] = d34
					ps96.OverlayValues[56] = d56
					ps96.OverlayValues[57] = d57
					ps96.OverlayValues[58] = d58
					ps96.OverlayValues[59] = d59
					ps96.OverlayValues[62] = d62
					ps96.OverlayValues[90] = d90
					ps96.OverlayValues[91] = d91
					ps96.OverlayValues[92] = d92
					ps96.OverlayValues[93] = d93
					ps97 := PhiState{General: true}
					ps97.OverlayValues = make([]JITValueDesc, 94)
					ps97.OverlayValues[1] = d1
					ps97.OverlayValues[2] = d2
					ps97.OverlayValues[3] = d3
					ps97.OverlayValues[4] = d4
					ps97.OverlayValues[5] = d5
					ps97.OverlayValues[6] = d6
					ps97.OverlayValues[18] = d18
					ps97.OverlayValues[19] = d19
					ps97.OverlayValues[20] = d20
					ps97.OverlayValues[21] = d21
					ps97.OverlayValues[22] = d22
					ps97.OverlayValues[23] = d23
					ps97.OverlayValues[24] = d24
					ps97.OverlayValues[25] = d25
					ps97.OverlayValues[26] = d26
					ps97.OverlayValues[28] = d28
					ps97.OverlayValues[30] = d30
					ps97.OverlayValues[31] = d31
					ps97.OverlayValues[34] = d34
					ps97.OverlayValues[56] = d56
					ps97.OverlayValues[57] = d57
					ps97.OverlayValues[58] = d58
					ps97.OverlayValues[59] = d59
					ps97.OverlayValues[62] = d62
					ps97.OverlayValues[90] = d90
					ps97.OverlayValues[91] = d91
					ps97.OverlayValues[92] = d92
					ps97.OverlayValues[93] = d93
					snap98 := d1
					snap99 := d2
					snap100 := d3
					snap101 := d4
					snap102 := d5
					snap103 := d6
					snap104 := d18
					snap105 := d19
					snap106 := d20
					snap107 := d21
					snap108 := d22
					snap109 := d23
					snap110 := d24
					snap111 := d25
					snap112 := d26
					snap113 := d28
					snap114 := d30
					snap115 := d31
					snap116 := d34
					snap117 := d56
					snap118 := d57
					snap119 := d58
					snap120 := d59
					snap121 := d62
					snap122 := d90
					snap123 := d91
					snap124 := d92
					snap125 := d93
					alloc126 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps97)
					}
					ctx.RestoreAllocState(alloc126)
					d1 = snap98
					d2 = snap99
					d3 = snap100
					d4 = snap101
					d5 = snap102
					d6 = snap103
					d18 = snap104
					d19 = snap105
					d20 = snap106
					d21 = snap107
					d22 = snap108
					d23 = snap109
					d24 = snap110
					d25 = snap111
					d26 = snap112
					d28 = snap113
					d30 = snap114
					d31 = snap115
					d34 = snap116
					d56 = snap117
					d57 = snap118
					d58 = snap119
					d59 = snap120
					d62 = snap121
					d90 = snap122
					d91 = snap123
					d92 = snap124
					d93 = snap125
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps96)
					}
					return result
					ctx.FreeDesc(&d92)
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					ctx.ReclaimUntrackedRegs()
					d127 = args[2]
					d127.ID = 0
					ctx.EnsureDesc(&d127)
					ctx.EnsureDesc(&d127)
					if d127.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d127.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d127.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d127)
						} else if d127.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d127)
						} else if d127.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d127)
						} else if d127.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d127.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d127 = tmpPair
					} else if d127.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d127.Type, Reg: ctx.AllocRegExcept(d127.Reg), Reg2: ctx.AllocRegExcept(d127.Reg)}
						switch d127.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d127)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d127)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d127)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d127)
						d127 = tmpPair
					}
					if d127.Loc != LocRegPair && d127.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (ToInt arg0)")
					}
					d128 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d127}, 1)
					ctx.BindReg(d128.Reg, &d128)
					ctx.FreeDesc(&d127)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d128)
					ctx.EnsureDesc(&d1)
					ctx.ProtectReg(d1.Reg)
					ctx.EnsureDesc(&d128)
					ctx.UnprotectReg(d1.Reg)
					var d129 JITValueDesc
					if d1.Loc == LocImm && d128.Loc == LocImm {
						d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d128.Imm.Int())}
					} else if d128.Loc == LocImm && d128.Imm.Int() == 0 {
						r5 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r5, d1.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d129)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d128.Reg}
						ctx.BindReg(d128.Reg, &d129)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d128.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d128.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d129)
					} else if d128.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d128.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d128.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d129)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg, d128.Reg)
						ctx.EmitMovRegReg(r6, d1.Reg)
						ctx.EmitAddInt64(r6, d128.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d129)
					}
					if d129.Loc == LocReg && d1.Loc == LocReg && d129.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d21)
					var d130 JITValueDesc
					if d129.Loc == LocImm && d21.Loc == LocImm {
						d130 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d129.Imm.Int() > d21.Imm.Int())}
					} else if d21.Loc == LocImm {
						r7 := ctx.AllocReg()
						if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d129.Reg, int32(d21.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
							ctx.EmitCmpInt64(d129.Reg, RegR11)
						}
						ctx.EmitSetcc(r7, CcG)
						d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d130)
					} else if d129.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d129.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d21.Reg)
						ctx.EmitSetcc(r8, CcG)
						d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d130)
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpInt64(d129.Reg, d21.Reg)
						ctx.EmitSetcc(r9, CcG)
						d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d130)
					}
					ctx.FreeDesc(&d129)
					d131 = d130
					ctx.EnsureDesc(&d131)
					if d131.Loc != LocImm && d131.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d131.Loc == LocImm {
						if d131.Imm.Bool() {
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
							ps132.OverlayValues[22] = d22
							ps132.OverlayValues[23] = d23
							ps132.OverlayValues[24] = d24
							ps132.OverlayValues[25] = d25
							ps132.OverlayValues[26] = d26
							ps132.OverlayValues[28] = d28
							ps132.OverlayValues[30] = d30
							ps132.OverlayValues[31] = d31
							ps132.OverlayValues[34] = d34
							ps132.OverlayValues[56] = d56
							ps132.OverlayValues[57] = d57
							ps132.OverlayValues[58] = d58
							ps132.OverlayValues[59] = d59
							ps132.OverlayValues[62] = d62
							ps132.OverlayValues[90] = d90
							ps132.OverlayValues[91] = d91
							ps132.OverlayValues[92] = d92
							ps132.OverlayValues[93] = d93
							ps132.OverlayValues[127] = d127
							ps132.OverlayValues[128] = d128
							ps132.OverlayValues[129] = d129
							ps132.OverlayValues[130] = d130
							ps132.OverlayValues[131] = d131
							return bbs[9].RenderPS(ps132)
						}
						ctx.EnsureDesc(&d128)
						if d128.Loc == LocReg {
							ctx.ProtectReg(d128.Reg)
						} else if d128.Loc == LocRegPair {
							ctx.ProtectReg(d128.Reg)
							ctx.ProtectReg(d128.Reg2)
						}
						d133 = d128
						if d133.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d133)
						ctx.EmitStoreToStack(d133, int32(bbs[10].PhiBase)+int32(0))
						if d128.Loc == LocReg {
							ctx.UnprotectReg(d128.Reg)
						} else if d128.Loc == LocRegPair {
							ctx.UnprotectReg(d128.Reg)
							ctx.UnprotectReg(d128.Reg2)
						}
						ps134 := PhiState{General: ps.General}
						ps134.OverlayValues = make([]JITValueDesc, 134)
						ps134.OverlayValues[1] = d1
						ps134.OverlayValues[2] = d2
						ps134.OverlayValues[3] = d3
						ps134.OverlayValues[4] = d4
						ps134.OverlayValues[5] = d5
						ps134.OverlayValues[6] = d6
						ps134.OverlayValues[18] = d18
						ps134.OverlayValues[19] = d19
						ps134.OverlayValues[20] = d20
						ps134.OverlayValues[21] = d21
						ps134.OverlayValues[22] = d22
						ps134.OverlayValues[23] = d23
						ps134.OverlayValues[24] = d24
						ps134.OverlayValues[25] = d25
						ps134.OverlayValues[26] = d26
						ps134.OverlayValues[28] = d28
						ps134.OverlayValues[30] = d30
						ps134.OverlayValues[31] = d31
						ps134.OverlayValues[34] = d34
						ps134.OverlayValues[56] = d56
						ps134.OverlayValues[57] = d57
						ps134.OverlayValues[58] = d58
						ps134.OverlayValues[59] = d59
						ps134.OverlayValues[62] = d62
						ps134.OverlayValues[90] = d90
						ps134.OverlayValues[91] = d91
						ps134.OverlayValues[92] = d92
						ps134.OverlayValues[93] = d93
						ps134.OverlayValues[127] = d127
						ps134.OverlayValues[128] = d128
						ps134.OverlayValues[129] = d129
						ps134.OverlayValues[130] = d130
						ps134.OverlayValues[131] = d131
						ps134.OverlayValues[133] = d133
						ps134.PhiValues = make([]JITValueDesc, 1)
						d135 = d128
						ps134.PhiValues[0] = d135
						return bbs[10].RenderPS(ps134)
					}
					if !ps.General {
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d131.Reg, 0)
					ctx.EmitJcc(CcNE, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl23)
					ctx.EnsureDesc(&d128)
					if d128.Loc == LocReg {
						ctx.ProtectReg(d128.Reg)
					} else if d128.Loc == LocRegPair {
						ctx.ProtectReg(d128.Reg)
						ctx.ProtectReg(d128.Reg2)
					}
					d136 = d128
					if d136.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d136)
					ctx.EmitStoreToStack(d136, int32(bbs[10].PhiBase)+int32(0))
					if d128.Loc == LocReg {
						ctx.UnprotectReg(d128.Reg)
					} else if d128.Loc == LocRegPair {
						ctx.UnprotectReg(d128.Reg)
						ctx.UnprotectReg(d128.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ps137 := PhiState{General: true}
					ps137.OverlayValues = make([]JITValueDesc, 137)
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
					ps137.OverlayValues[22] = d22
					ps137.OverlayValues[23] = d23
					ps137.OverlayValues[24] = d24
					ps137.OverlayValues[25] = d25
					ps137.OverlayValues[26] = d26
					ps137.OverlayValues[28] = d28
					ps137.OverlayValues[30] = d30
					ps137.OverlayValues[31] = d31
					ps137.OverlayValues[34] = d34
					ps137.OverlayValues[56] = d56
					ps137.OverlayValues[57] = d57
					ps137.OverlayValues[58] = d58
					ps137.OverlayValues[59] = d59
					ps137.OverlayValues[62] = d62
					ps137.OverlayValues[90] = d90
					ps137.OverlayValues[91] = d91
					ps137.OverlayValues[92] = d92
					ps137.OverlayValues[93] = d93
					ps137.OverlayValues[127] = d127
					ps137.OverlayValues[128] = d128
					ps137.OverlayValues[129] = d129
					ps137.OverlayValues[130] = d130
					ps137.OverlayValues[131] = d131
					ps137.OverlayValues[133] = d133
					ps137.OverlayValues[135] = d135
					ps137.OverlayValues[136] = d136
					ps138 := PhiState{General: true}
					ps138.OverlayValues = make([]JITValueDesc, 137)
					ps138.OverlayValues[1] = d1
					ps138.OverlayValues[2] = d2
					ps138.OverlayValues[3] = d3
					ps138.OverlayValues[4] = d4
					ps138.OverlayValues[5] = d5
					ps138.OverlayValues[6] = d6
					ps138.OverlayValues[18] = d18
					ps138.OverlayValues[19] = d19
					ps138.OverlayValues[20] = d20
					ps138.OverlayValues[21] = d21
					ps138.OverlayValues[22] = d22
					ps138.OverlayValues[23] = d23
					ps138.OverlayValues[24] = d24
					ps138.OverlayValues[25] = d25
					ps138.OverlayValues[26] = d26
					ps138.OverlayValues[28] = d28
					ps138.OverlayValues[30] = d30
					ps138.OverlayValues[31] = d31
					ps138.OverlayValues[34] = d34
					ps138.OverlayValues[56] = d56
					ps138.OverlayValues[57] = d57
					ps138.OverlayValues[58] = d58
					ps138.OverlayValues[59] = d59
					ps138.OverlayValues[62] = d62
					ps138.OverlayValues[90] = d90
					ps138.OverlayValues[91] = d91
					ps138.OverlayValues[92] = d92
					ps138.OverlayValues[93] = d93
					ps138.OverlayValues[127] = d127
					ps138.OverlayValues[128] = d128
					ps138.OverlayValues[129] = d129
					ps138.OverlayValues[130] = d130
					ps138.OverlayValues[131] = d131
					ps138.OverlayValues[133] = d133
					ps138.OverlayValues[135] = d135
					ps138.OverlayValues[136] = d136
					ps138.PhiValues = make([]JITValueDesc, 1)
					d139 = d128
					ps138.PhiValues[0] = d139
					snap140 := d1
					snap141 := d2
					snap142 := d3
					snap143 := d4
					snap144 := d5
					snap145 := d6
					snap146 := d18
					snap147 := d19
					snap148 := d20
					snap149 := d21
					snap150 := d22
					snap151 := d23
					snap152 := d24
					snap153 := d25
					snap154 := d26
					snap155 := d28
					snap156 := d30
					snap157 := d31
					snap158 := d34
					snap159 := d56
					snap160 := d57
					snap161 := d58
					snap162 := d59
					snap163 := d62
					snap164 := d90
					snap165 := d91
					snap166 := d92
					snap167 := d93
					snap168 := d127
					snap169 := d128
					snap170 := d129
					snap171 := d130
					snap172 := d131
					snap173 := d133
					snap174 := d135
					snap175 := d136
					snap176 := d139
					alloc177 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps138)
					}
					ctx.RestoreAllocState(alloc177)
					d1 = snap140
					d2 = snap141
					d3 = snap142
					d4 = snap143
					d5 = snap144
					d6 = snap145
					d18 = snap146
					d19 = snap147
					d20 = snap148
					d21 = snap149
					d22 = snap150
					d23 = snap151
					d24 = snap152
					d25 = snap153
					d26 = snap154
					d28 = snap155
					d30 = snap156
					d31 = snap157
					d34 = snap158
					d56 = snap159
					d57 = snap160
					d58 = snap161
					d59 = snap162
					d62 = snap163
					d90 = snap164
					d91 = snap165
					d92 = snap166
					d93 = snap167
					d127 = snap168
					d128 = snap169
					d129 = snap170
					d130 = snap171
					d131 = snap172
					d133 = snap173
					d135 = snap174
					d136 = snap175
					d139 = snap176
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps137)
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
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
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					var d178 JITValueDesc
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocRegPair {
						d178 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2}
						ctx.BindReg(d19.Reg2, &d178)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d178)
					var d180 JITValueDesc
					if d178.Loc == LocImm && d1.Loc == LocImm {
						d180 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d178.Imm.Int() - d1.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d178.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d178.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d178.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r10, RegR11)
						} else {
							ctx.EmitSubInt64(r10, d1.Reg)
						}
						d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d180)
					}
					var d181 JITValueDesc
					if d19.Loc == LocImm && d1.Loc == LocImm {
						d181 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int() + d1.Imm.Int())}
					} else {
						r11 := ctx.AllocReg()
						if d19.Loc == LocImm {
							ctx.EmitMovRegImm64(r11, uint64(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r11, d19.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitAddInt64(r11, RegR11)
						} else {
							ctx.EmitAddInt64(r11, d1.Reg)
						}
						d181 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d181)
					}
					var d182 JITValueDesc
					r12 := ctx.AllocReg()
					r13 := ctx.AllocReg()
					if d181.Loc == LocImm {
						ctx.EmitMovRegImm64(r12, uint64(d181.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r12, d181.Reg)
						ctx.FreeReg(d181.Reg)
					}
					if d180.Loc == LocImm {
						ctx.EmitMovRegImm64(r13, uint64(d180.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r13, d180.Reg)
						ctx.FreeReg(d180.Reg)
					}
					d182 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d182)
					ctx.BindReg(r13, &d182)
					d183 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d182}, 2)
					ctx.EmitMovPairToResult(&d183, &result)
					result.Type = tagString
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
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
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.ProtectReg(d21.Reg)
					ctx.EnsureDesc(&d1)
					ctx.UnprotectReg(d21.Reg)
					var d184 JITValueDesc
					if d21.Loc == LocImm && d1.Loc == LocImm {
						d184 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d21.Imm.Int() - d1.Imm.Int())}
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						r14 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitMovRegReg(r14, d21.Reg)
						d184 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d184)
					} else if d21.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
						ctx.EmitSubInt64(scratch, d1.Reg)
						d184 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d184)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitMovRegReg(scratch, d21.Reg)
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d184 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d184)
					} else {
						r15 := ctx.AllocRegExcept(d21.Reg, d1.Reg)
						ctx.EmitMovRegReg(r15, d21.Reg)
						ctx.EmitSubInt64(r15, d1.Reg)
						d184 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d184)
					}
					if d184.Loc == LocReg && d21.Loc == LocReg && d184.Reg == d21.Reg {
						ctx.TransferReg(d21.Reg)
						d21.Loc = LocNone
					}
					ctx.FreeDesc(&d21)
					ctx.EnsureDesc(&d184)
					if d184.Loc == LocReg {
						ctx.ProtectReg(d184.Reg)
					} else if d184.Loc == LocRegPair {
						ctx.ProtectReg(d184.Reg)
						ctx.ProtectReg(d184.Reg2)
					}
					d185 = d184
					if d185.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d185)
					ctx.EmitStoreToStack(d185, int32(bbs[10].PhiBase)+int32(0))
					if d184.Loc == LocReg {
						ctx.UnprotectReg(d184.Reg)
					} else if d184.Loc == LocRegPair {
						ctx.UnprotectReg(d184.Reg)
						ctx.UnprotectReg(d184.Reg2)
					}
					ps186 := PhiState{General: ps.General}
					ps186.OverlayValues = make([]JITValueDesc, 186)
					ps186.OverlayValues[1] = d1
					ps186.OverlayValues[2] = d2
					ps186.OverlayValues[3] = d3
					ps186.OverlayValues[4] = d4
					ps186.OverlayValues[5] = d5
					ps186.OverlayValues[6] = d6
					ps186.OverlayValues[18] = d18
					ps186.OverlayValues[19] = d19
					ps186.OverlayValues[20] = d20
					ps186.OverlayValues[21] = d21
					ps186.OverlayValues[22] = d22
					ps186.OverlayValues[23] = d23
					ps186.OverlayValues[24] = d24
					ps186.OverlayValues[25] = d25
					ps186.OverlayValues[26] = d26
					ps186.OverlayValues[28] = d28
					ps186.OverlayValues[30] = d30
					ps186.OverlayValues[31] = d31
					ps186.OverlayValues[34] = d34
					ps186.OverlayValues[56] = d56
					ps186.OverlayValues[57] = d57
					ps186.OverlayValues[58] = d58
					ps186.OverlayValues[59] = d59
					ps186.OverlayValues[62] = d62
					ps186.OverlayValues[90] = d90
					ps186.OverlayValues[91] = d91
					ps186.OverlayValues[92] = d92
					ps186.OverlayValues[93] = d93
					ps186.OverlayValues[127] = d127
					ps186.OverlayValues[128] = d128
					ps186.OverlayValues[129] = d129
					ps186.OverlayValues[130] = d130
					ps186.OverlayValues[131] = d131
					ps186.OverlayValues[133] = d133
					ps186.OverlayValues[135] = d135
					ps186.OverlayValues[136] = d136
					ps186.OverlayValues[139] = d139
					ps186.OverlayValues[178] = d178
					ps186.OverlayValues[179] = d179
					ps186.OverlayValues[180] = d180
					ps186.OverlayValues[181] = d181
					ps186.OverlayValues[182] = d182
					ps186.OverlayValues[183] = d183
					ps186.OverlayValues[184] = d184
					ps186.OverlayValues[185] = d185
					ps186.PhiValues = make([]JITValueDesc, 1)
					d187 = d184
					ps186.PhiValues[0] = d187
					if ps186.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps186)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d188 := ps.PhiValues[0]
							ctx.EnsureDesc(&d188)
							ctx.EmitStoreToStack(d188, int32(bbs[10].PhiBase)+int32(0))
						}
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
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
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
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
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					var d189 JITValueDesc
					if d2.Loc == LocImm {
						d189 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 0)}
					} else {
						r16 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r16, CcL)
						d189 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
						ctx.BindReg(r16, &d189)
					}
					d190 = d189
					ctx.EnsureDesc(&d190)
					if d190.Loc != LocImm && d190.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d190.Loc == LocImm {
						if d190.Imm.Bool() {
							ps191 := PhiState{General: ps.General}
							ps191.OverlayValues = make([]JITValueDesc, 191)
							ps191.OverlayValues[1] = d1
							ps191.OverlayValues[2] = d2
							ps191.OverlayValues[3] = d3
							ps191.OverlayValues[4] = d4
							ps191.OverlayValues[5] = d5
							ps191.OverlayValues[6] = d6
							ps191.OverlayValues[18] = d18
							ps191.OverlayValues[19] = d19
							ps191.OverlayValues[20] = d20
							ps191.OverlayValues[21] = d21
							ps191.OverlayValues[22] = d22
							ps191.OverlayValues[23] = d23
							ps191.OverlayValues[24] = d24
							ps191.OverlayValues[25] = d25
							ps191.OverlayValues[26] = d26
							ps191.OverlayValues[28] = d28
							ps191.OverlayValues[30] = d30
							ps191.OverlayValues[31] = d31
							ps191.OverlayValues[34] = d34
							ps191.OverlayValues[56] = d56
							ps191.OverlayValues[57] = d57
							ps191.OverlayValues[58] = d58
							ps191.OverlayValues[59] = d59
							ps191.OverlayValues[62] = d62
							ps191.OverlayValues[90] = d90
							ps191.OverlayValues[91] = d91
							ps191.OverlayValues[92] = d92
							ps191.OverlayValues[93] = d93
							ps191.OverlayValues[127] = d127
							ps191.OverlayValues[128] = d128
							ps191.OverlayValues[129] = d129
							ps191.OverlayValues[130] = d130
							ps191.OverlayValues[131] = d131
							ps191.OverlayValues[133] = d133
							ps191.OverlayValues[135] = d135
							ps191.OverlayValues[136] = d136
							ps191.OverlayValues[139] = d139
							ps191.OverlayValues[178] = d178
							ps191.OverlayValues[179] = d179
							ps191.OverlayValues[180] = d180
							ps191.OverlayValues[181] = d181
							ps191.OverlayValues[182] = d182
							ps191.OverlayValues[183] = d183
							ps191.OverlayValues[184] = d184
							ps191.OverlayValues[185] = d185
							ps191.OverlayValues[187] = d187
							ps191.OverlayValues[188] = d188
							ps191.OverlayValues[189] = d189
							ps191.OverlayValues[190] = d190
							return bbs[11].RenderPS(ps191)
						}
						ps192 := PhiState{General: ps.General}
						ps192.OverlayValues = make([]JITValueDesc, 191)
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
						ps192.OverlayValues[22] = d22
						ps192.OverlayValues[23] = d23
						ps192.OverlayValues[24] = d24
						ps192.OverlayValues[25] = d25
						ps192.OverlayValues[26] = d26
						ps192.OverlayValues[28] = d28
						ps192.OverlayValues[30] = d30
						ps192.OverlayValues[31] = d31
						ps192.OverlayValues[34] = d34
						ps192.OverlayValues[56] = d56
						ps192.OverlayValues[57] = d57
						ps192.OverlayValues[58] = d58
						ps192.OverlayValues[59] = d59
						ps192.OverlayValues[62] = d62
						ps192.OverlayValues[90] = d90
						ps192.OverlayValues[91] = d91
						ps192.OverlayValues[92] = d92
						ps192.OverlayValues[93] = d93
						ps192.OverlayValues[127] = d127
						ps192.OverlayValues[128] = d128
						ps192.OverlayValues[129] = d129
						ps192.OverlayValues[130] = d130
						ps192.OverlayValues[131] = d131
						ps192.OverlayValues[133] = d133
						ps192.OverlayValues[135] = d135
						ps192.OverlayValues[136] = d136
						ps192.OverlayValues[139] = d139
						ps192.OverlayValues[178] = d178
						ps192.OverlayValues[179] = d179
						ps192.OverlayValues[180] = d180
						ps192.OverlayValues[181] = d181
						ps192.OverlayValues[182] = d182
						ps192.OverlayValues[183] = d183
						ps192.OverlayValues[184] = d184
						ps192.OverlayValues[185] = d185
						ps192.OverlayValues[187] = d187
						ps192.OverlayValues[188] = d188
						ps192.OverlayValues[189] = d189
						ps192.OverlayValues[190] = d190
						return bbs[12].RenderPS(ps192)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d193 := ps.PhiValues[0]
							ctx.EnsureDesc(&d193)
							ctx.EmitStoreToStack(d193, int32(bbs[10].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d190.Reg, 0)
					ctx.EmitJcc(CcNE, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl13)
					ps194 := PhiState{General: true}
					ps194.OverlayValues = make([]JITValueDesc, 194)
					ps194.OverlayValues[1] = d1
					ps194.OverlayValues[2] = d2
					ps194.OverlayValues[3] = d3
					ps194.OverlayValues[4] = d4
					ps194.OverlayValues[5] = d5
					ps194.OverlayValues[6] = d6
					ps194.OverlayValues[18] = d18
					ps194.OverlayValues[19] = d19
					ps194.OverlayValues[20] = d20
					ps194.OverlayValues[21] = d21
					ps194.OverlayValues[22] = d22
					ps194.OverlayValues[23] = d23
					ps194.OverlayValues[24] = d24
					ps194.OverlayValues[25] = d25
					ps194.OverlayValues[26] = d26
					ps194.OverlayValues[28] = d28
					ps194.OverlayValues[30] = d30
					ps194.OverlayValues[31] = d31
					ps194.OverlayValues[34] = d34
					ps194.OverlayValues[56] = d56
					ps194.OverlayValues[57] = d57
					ps194.OverlayValues[58] = d58
					ps194.OverlayValues[59] = d59
					ps194.OverlayValues[62] = d62
					ps194.OverlayValues[90] = d90
					ps194.OverlayValues[91] = d91
					ps194.OverlayValues[92] = d92
					ps194.OverlayValues[93] = d93
					ps194.OverlayValues[127] = d127
					ps194.OverlayValues[128] = d128
					ps194.OverlayValues[129] = d129
					ps194.OverlayValues[130] = d130
					ps194.OverlayValues[131] = d131
					ps194.OverlayValues[133] = d133
					ps194.OverlayValues[135] = d135
					ps194.OverlayValues[136] = d136
					ps194.OverlayValues[139] = d139
					ps194.OverlayValues[178] = d178
					ps194.OverlayValues[179] = d179
					ps194.OverlayValues[180] = d180
					ps194.OverlayValues[181] = d181
					ps194.OverlayValues[182] = d182
					ps194.OverlayValues[183] = d183
					ps194.OverlayValues[184] = d184
					ps194.OverlayValues[185] = d185
					ps194.OverlayValues[187] = d187
					ps194.OverlayValues[188] = d188
					ps194.OverlayValues[189] = d189
					ps194.OverlayValues[190] = d190
					ps194.OverlayValues[193] = d193
					ps195 := PhiState{General: true}
					ps195.OverlayValues = make([]JITValueDesc, 194)
					ps195.OverlayValues[1] = d1
					ps195.OverlayValues[2] = d2
					ps195.OverlayValues[3] = d3
					ps195.OverlayValues[4] = d4
					ps195.OverlayValues[5] = d5
					ps195.OverlayValues[6] = d6
					ps195.OverlayValues[18] = d18
					ps195.OverlayValues[19] = d19
					ps195.OverlayValues[20] = d20
					ps195.OverlayValues[21] = d21
					ps195.OverlayValues[22] = d22
					ps195.OverlayValues[23] = d23
					ps195.OverlayValues[24] = d24
					ps195.OverlayValues[25] = d25
					ps195.OverlayValues[26] = d26
					ps195.OverlayValues[28] = d28
					ps195.OverlayValues[30] = d30
					ps195.OverlayValues[31] = d31
					ps195.OverlayValues[34] = d34
					ps195.OverlayValues[56] = d56
					ps195.OverlayValues[57] = d57
					ps195.OverlayValues[58] = d58
					ps195.OverlayValues[59] = d59
					ps195.OverlayValues[62] = d62
					ps195.OverlayValues[90] = d90
					ps195.OverlayValues[91] = d91
					ps195.OverlayValues[92] = d92
					ps195.OverlayValues[93] = d93
					ps195.OverlayValues[127] = d127
					ps195.OverlayValues[128] = d128
					ps195.OverlayValues[129] = d129
					ps195.OverlayValues[130] = d130
					ps195.OverlayValues[131] = d131
					ps195.OverlayValues[133] = d133
					ps195.OverlayValues[135] = d135
					ps195.OverlayValues[136] = d136
					ps195.OverlayValues[139] = d139
					ps195.OverlayValues[178] = d178
					ps195.OverlayValues[179] = d179
					ps195.OverlayValues[180] = d180
					ps195.OverlayValues[181] = d181
					ps195.OverlayValues[182] = d182
					ps195.OverlayValues[183] = d183
					ps195.OverlayValues[184] = d184
					ps195.OverlayValues[185] = d185
					ps195.OverlayValues[187] = d187
					ps195.OverlayValues[188] = d188
					ps195.OverlayValues[189] = d189
					ps195.OverlayValues[190] = d190
					ps195.OverlayValues[193] = d193
					snap196 := d1
					snap197 := d2
					snap198 := d3
					snap199 := d4
					snap200 := d5
					snap201 := d6
					snap202 := d18
					snap203 := d19
					snap204 := d20
					snap205 := d21
					snap206 := d22
					snap207 := d23
					snap208 := d24
					snap209 := d25
					snap210 := d26
					snap211 := d28
					snap212 := d30
					snap213 := d31
					snap214 := d34
					snap215 := d56
					snap216 := d57
					snap217 := d58
					snap218 := d59
					snap219 := d62
					snap220 := d90
					snap221 := d91
					snap222 := d92
					snap223 := d93
					snap224 := d127
					snap225 := d128
					snap226 := d129
					snap227 := d130
					snap228 := d131
					snap229 := d133
					snap230 := d135
					snap231 := d136
					snap232 := d139
					snap233 := d178
					snap234 := d179
					snap235 := d180
					snap236 := d181
					snap237 := d182
					snap238 := d183
					snap239 := d184
					snap240 := d185
					snap241 := d187
					snap242 := d188
					snap243 := d189
					snap244 := d190
					snap245 := d193
					alloc246 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps195)
					}
					ctx.RestoreAllocState(alloc246)
					d1 = snap196
					d2 = snap197
					d3 = snap198
					d4 = snap199
					d5 = snap200
					d6 = snap201
					d18 = snap202
					d19 = snap203
					d20 = snap204
					d21 = snap205
					d22 = snap206
					d23 = snap207
					d24 = snap208
					d25 = snap209
					d26 = snap210
					d28 = snap211
					d30 = snap212
					d31 = snap213
					d34 = snap214
					d56 = snap215
					d57 = snap216
					d58 = snap217
					d59 = snap218
					d62 = snap219
					d90 = snap220
					d91 = snap221
					d92 = snap222
					d93 = snap223
					d127 = snap224
					d128 = snap225
					d129 = snap226
					d130 = snap227
					d131 = snap228
					d133 = snap229
					d135 = snap230
					d136 = snap231
					d139 = snap232
					d178 = snap233
					d179 = snap234
					d180 = snap235
					d181 = snap236
					d182 = snap237
					d183 = snap238
					d184 = snap239
					d185 = snap240
					d187 = snap241
					d188 = snap242
					d189 = snap243
					d190 = snap244
					d193 = snap245
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps194)
					}
					return result
					ctx.FreeDesc(&d189)
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
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
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
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
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					ctx.ReclaimUntrackedRegs()
					d247 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{}, 2)
					ctx.EmitMovPairToResult(&d247, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
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
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
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
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
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
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != LocNone {
						d247 = ps.OverlayValues[247]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d1)
					ctx.ProtectReg(d1.Reg)
					ctx.EnsureDesc(&d2)
					ctx.UnprotectReg(d1.Reg)
					var d248 JITValueDesc
					if d1.Loc == LocImm && d2.Loc == LocImm {
						d248 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d2.Imm.Int())}
					} else if d2.Loc == LocImm && d2.Imm.Int() == 0 {
						r17 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r17, d1.Reg)
						d248 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d248)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d248 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
						ctx.BindReg(d2.Reg, &d248)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d2.Reg)
						d248 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d248)
					} else if d2.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d248 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d248)
					} else {
						r18 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
						ctx.EmitMovRegReg(r18, d1.Reg)
						ctx.EmitAddInt64(r18, d2.Reg)
						d248 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d248)
					}
					if d248.Loc == LocReg && d1.Loc == LocReg && d248.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d248)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d248)
					var d250 JITValueDesc
					if d248.Loc == LocImm && d1.Loc == LocImm {
						d250 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d248.Imm.Int() - d1.Imm.Int())}
					} else {
						r19 := ctx.AllocReg()
						if d248.Loc == LocImm {
							ctx.EmitMovRegImm64(r19, uint64(d248.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r19, d248.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r19, RegR11)
						} else {
							ctx.EmitSubInt64(r19, d1.Reg)
						}
						d250 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d250)
					}
					var d251 JITValueDesc
					if d19.Loc == LocImm && d1.Loc == LocImm {
						d251 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int() + d1.Imm.Int())}
					} else {
						r20 := ctx.AllocReg()
						if d19.Loc == LocImm {
							ctx.EmitMovRegImm64(r20, uint64(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r20, d19.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitAddInt64(r20, RegR11)
						} else {
							ctx.EmitAddInt64(r20, d1.Reg)
						}
						d251 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
						ctx.BindReg(r20, &d251)
					}
					var d252 JITValueDesc
					r21 := ctx.AllocReg()
					r22 := ctx.AllocReg()
					if d251.Loc == LocImm {
						ctx.EmitMovRegImm64(r21, uint64(d251.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r21, d251.Reg)
						ctx.FreeReg(d251.Reg)
					}
					if d250.Loc == LocImm {
						ctx.EmitMovRegImm64(r22, uint64(d250.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r22, d250.Reg)
						ctx.FreeReg(d250.Reg)
					}
					d252 = JITValueDesc{Loc: LocRegPair, Reg: r21, Reg2: r22}
					ctx.BindReg(r21, &d252)
					ctx.BindReg(r22, &d252)
					ctx.FreeDesc(&d1)
					ctx.FreeDesc(&d248)
					d253 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d252}, 2)
					ctx.EmitMovPairToResult(&d253, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned254 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned254 = append(argPinned254, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned254 = append(argPinned254, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned254 = append(argPinned254, ai.Reg2)
						}
					}
				}
				ps255 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps255)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
				for _, r := range argPinned254 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "simplify",

		Fn: func(a ...Scmer) Scmer {
			// turn string to number or so
			return Simplify(String(a[0]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Converts numeric text to a number. Text beginning with { or [ becomes a native JSON value when it is valid JSON; other input remains a string.",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "value to interpret as a number or JSON value"}},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned0 = append(argPinned0, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned0 {
						ctx.UnprotectReg(r)
					}
				}()
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
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
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (Simplify arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(Simplify), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				if d4.Loc == LocImm {
					if result.Loc == LocAny { return d4 }
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d4)
				if d4.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d4, &result)
					result.Type = d4.Type
				} else {
					switch d4.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d4)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d4)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d4)
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
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strlen",

		Fn: func(a ...Scmer) Scmer {
			return NewInt(int64(len(String(a[0]))))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the length of a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
				var d4 JITValueDesc
				if d2.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d2.Imm.String())))}
				} else {
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocRegPair {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg2}
						ctx.BindReg(d2.Reg2, &d4)
						ctx.BindReg(d2.Reg2, &d4)
					} else if d2.Loc == LocReg {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
						ctx.BindReg(d2.Reg, &d4)
						ctx.BindReg(d2.Reg, &d4)
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d4.Loc == LocImm {
					ctx.EmitMakeInt(result, d4)
				} else {
					ctx.EmitMakeInt(result, d4)
					ctx.FreeReg(d4.Reg)
				}
				result.Type = tagInt
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strlike",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			value := String(a[0])
			pattern := String(a[1])
			collation := "utf8mb4_general_ci"
			if len(a) > 2 {
				collation = strings.ToLower(String(a[2]))
			}
			return NewBool(StrLikeCollation(value, pattern, collation))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "matches the string against a wildcard pattern using SQL NULL semantics",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "pattern with % and _ in them"}, &TypeDescriptor{Kind: "string", Label: "collation", Description: "collation in which to compare them", Optional: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
				var d27 JITValueDesc
				_ = d27
				var d30 JITValueDesc
				_ = d30
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
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
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				var bbs [6]BBDescriptor
				bbs[5].PhiBase = int32(phiBase0) + int32(0)
				bbs[5].PhiCount = uint16(1)
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
							ps6.OverlayValues[1] = d1
							ps6.OverlayValues[2] = d2
							ps6.OverlayValues[3] = d3
							ps6.OverlayValues[4] = d4
							ps6.OverlayValues[5] = d5
							return bbs[1].RenderPS(ps6)
						}
						ps7 := PhiState{General: ps.General}
						ps7.OverlayValues = make([]JITValueDesc, 6)
						ps7.OverlayValues[1] = d1
						ps7.OverlayValues[2] = d2
						ps7.OverlayValues[3] = d3
						ps7.OverlayValues[4] = d4
						ps7.OverlayValues[5] = d5
						return bbs[3].RenderPS(ps7)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJcc(CcNE, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
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
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps9)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					d16 = args[0]
					d16.ID = 0
					d18 = d16
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d18.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d18)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d18)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d18)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d18.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d18 = tmpPair
					} else if d18.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d18.Reg), Reg2: ctx.AllocRegExcept(d18.Reg)}
						switch d18.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d18)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d18)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d18)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d18)
						d18 = tmpPair
					} else if d18.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d18.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d18.MemPtr))
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
						d18 = tmpPair
					}
					if d18.Loc != LocRegPair && d18.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d18}, 2)
					ctx.FreeDesc(&d16)
					d19 = args[1]
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
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d22)
					var d23 JITValueDesc
					if d22.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d22.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d22.Reg, 2)
						ctx.EmitSetcc(r0, CcG)
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
							ps25.OverlayValues[20] = d20
							ps25.OverlayValues[21] = d21
							ps25.OverlayValues[22] = d22
							ps25.OverlayValues[23] = d23
							ps25.OverlayValues[24] = d24
							return bbs[4].RenderPS(ps25)
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
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
						ps26.OverlayValues[20] = d20
						ps26.OverlayValues[21] = d21
						ps26.OverlayValues[22] = d22
						ps26.OverlayValues[23] = d23
						ps26.OverlayValues[24] = d24
						ps26.PhiValues = make([]JITValueDesc, 1)
						d27 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
						ps26.PhiValues[0] = d27
						return bbs[5].RenderPS(ps26)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d24.Reg, 0)
					ctx.EmitJcc(CcNE, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
					ctx.EmitJmp(lbl6)
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 28)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[18] = d18
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[24] = d24
					ps28.OverlayValues[27] = d27
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 28)
					ps29.OverlayValues[1] = d1
					ps29.OverlayValues[2] = d2
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[16] = d16
					ps29.OverlayValues[17] = d17
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[20] = d20
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[27] = d27
					ps29.PhiValues = make([]JITValueDesc, 1)
					d30 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
					ps29.PhiValues[0] = d30
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d4
					snap35 := d5
					snap36 := d16
					snap37 := d17
					snap38 := d18
					snap39 := d19
					snap40 := d20
					snap41 := d21
					snap42 := d22
					snap43 := d23
					snap44 := d24
					snap45 := d27
					snap46 := d30
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps29)
					}
					ctx.RestoreAllocState(alloc47)
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d4 = snap34
					d5 = snap35
					d16 = snap36
					d17 = snap37
					d18 = snap38
					d19 = snap39
					d20 = snap40
					d21 = snap41
					d22 = snap42
					d23 = snap43
					d24 = snap44
					d27 = snap45
					d30 = snap46
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps28)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
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
							ps52 := PhiState{General: ps.General}
							ps52.OverlayValues = make([]JITValueDesc, 52)
							ps52.OverlayValues[1] = d1
							ps52.OverlayValues[2] = d2
							ps52.OverlayValues[3] = d3
							ps52.OverlayValues[4] = d4
							ps52.OverlayValues[5] = d5
							ps52.OverlayValues[16] = d16
							ps52.OverlayValues[17] = d17
							ps52.OverlayValues[18] = d18
							ps52.OverlayValues[19] = d19
							ps52.OverlayValues[20] = d20
							ps52.OverlayValues[21] = d21
							ps52.OverlayValues[22] = d22
							ps52.OverlayValues[23] = d23
							ps52.OverlayValues[24] = d24
							ps52.OverlayValues[27] = d27
							ps52.OverlayValues[30] = d30
							ps52.OverlayValues[48] = d48
							ps52.OverlayValues[49] = d49
							ps52.OverlayValues[50] = d50
							ps52.OverlayValues[51] = d51
							return bbs[1].RenderPS(ps52)
						}
						ps53 := PhiState{General: ps.General}
						ps53.OverlayValues = make([]JITValueDesc, 52)
						ps53.OverlayValues[1] = d1
						ps53.OverlayValues[2] = d2
						ps53.OverlayValues[3] = d3
						ps53.OverlayValues[4] = d4
						ps53.OverlayValues[5] = d5
						ps53.OverlayValues[16] = d16
						ps53.OverlayValues[17] = d17
						ps53.OverlayValues[18] = d18
						ps53.OverlayValues[19] = d19
						ps53.OverlayValues[20] = d20
						ps53.OverlayValues[21] = d21
						ps53.OverlayValues[22] = d22
						ps53.OverlayValues[23] = d23
						ps53.OverlayValues[24] = d24
						ps53.OverlayValues[27] = d27
						ps53.OverlayValues[30] = d30
						ps53.OverlayValues[48] = d48
						ps53.OverlayValues[49] = d49
						ps53.OverlayValues[50] = d50
						ps53.OverlayValues[51] = d51
						return bbs[2].RenderPS(ps53)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d51.Reg, 0)
					ctx.EmitJcc(CcNE, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl3)
					ps54 := PhiState{General: true}
					ps54.OverlayValues = make([]JITValueDesc, 52)
					ps54.OverlayValues[1] = d1
					ps54.OverlayValues[2] = d2
					ps54.OverlayValues[3] = d3
					ps54.OverlayValues[4] = d4
					ps54.OverlayValues[5] = d5
					ps54.OverlayValues[16] = d16
					ps54.OverlayValues[17] = d17
					ps54.OverlayValues[18] = d18
					ps54.OverlayValues[19] = d19
					ps54.OverlayValues[20] = d20
					ps54.OverlayValues[21] = d21
					ps54.OverlayValues[22] = d22
					ps54.OverlayValues[23] = d23
					ps54.OverlayValues[24] = d24
					ps54.OverlayValues[27] = d27
					ps54.OverlayValues[30] = d30
					ps54.OverlayValues[48] = d48
					ps54.OverlayValues[49] = d49
					ps54.OverlayValues[50] = d50
					ps54.OverlayValues[51] = d51
					ps55 := PhiState{General: true}
					ps55.OverlayValues = make([]JITValueDesc, 52)
					ps55.OverlayValues[1] = d1
					ps55.OverlayValues[2] = d2
					ps55.OverlayValues[3] = d3
					ps55.OverlayValues[4] = d4
					ps55.OverlayValues[5] = d5
					ps55.OverlayValues[16] = d16
					ps55.OverlayValues[17] = d17
					ps55.OverlayValues[18] = d18
					ps55.OverlayValues[19] = d19
					ps55.OverlayValues[20] = d20
					ps55.OverlayValues[21] = d21
					ps55.OverlayValues[22] = d22
					ps55.OverlayValues[23] = d23
					ps55.OverlayValues[24] = d24
					ps55.OverlayValues[27] = d27
					ps55.OverlayValues[30] = d30
					ps55.OverlayValues[48] = d48
					ps55.OverlayValues[49] = d49
					ps55.OverlayValues[50] = d50
					ps55.OverlayValues[51] = d51
					snap56 := d1
					snap57 := d2
					snap58 := d3
					snap59 := d4
					snap60 := d5
					snap61 := d16
					snap62 := d17
					snap63 := d18
					snap64 := d19
					snap65 := d20
					snap66 := d21
					snap67 := d22
					snap68 := d23
					snap69 := d24
					snap70 := d27
					snap71 := d30
					snap72 := d48
					snap73 := d49
					snap74 := d50
					snap75 := d51
					alloc76 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps55)
					}
					ctx.RestoreAllocState(alloc76)
					d1 = snap56
					d2 = snap57
					d3 = snap58
					d4 = snap59
					d5 = snap60
					d16 = snap61
					d17 = snap62
					d18 = snap63
					d19 = snap64
					d20 = snap65
					d21 = snap66
					d22 = snap67
					d23 = snap68
					d24 = snap69
					d27 = snap70
					d30 = snap71
					d48 = snap72
					d49 = snap73
					d50 = snap74
					d51 = snap75
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps54)
					}
					return result
					ctx.FreeDesc(&d49)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
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
					ctx.ReclaimUntrackedRegs()
					d77 = args[2]
					d77.ID = 0
					d79 = d77
					ctx.EnsureDesc(&d79)
					if d79.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d79.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d79)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d79)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d79)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d79.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d79 = tmpPair
					} else if d79.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d79.Reg), Reg2: ctx.AllocRegExcept(d79.Reg)}
						switch d79.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d79)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d79)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d79)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d79)
						d79 = tmpPair
					} else if d79.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d79.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d79.MemPtr))
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
						d79 = tmpPair
					}
					if d79.Loc != LocRegPair && d79.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d78 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d79}, 2)
					ctx.FreeDesc(&d77)
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d78)
					if d78.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d78.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d78.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d78)
						} else if d78.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d78)
						} else if d78.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d78)
						} else if d78.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d78.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d78 = tmpPair
					} else if d78.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d78.Type, Reg: ctx.AllocRegExcept(d78.Reg), Reg2: ctx.AllocRegExcept(d78.Reg)}
						switch d78.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d78)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d78)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d78)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d78)
						d78 = tmpPair
					}
					if d78.Loc != LocRegPair && d78.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					d80 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d78}, 2)
					ctx.BindReg(d80.Reg, &d80)
					ctx.BindReg(d80.Reg2, &d80)
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocReg {
						ctx.ProtectReg(d80.Reg)
					} else if d80.Loc == LocRegPair {
						ctx.ProtectReg(d80.Reg)
						ctx.ProtectReg(d80.Reg2)
					}
					d81 = d80
					if d81.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d81)
					if d81.Loc == LocRegPair || d81.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d81, int32(bbs[5].PhiBase)+int32(0))
					} else {
						ctx.EmitStoreToStack(d81, int32(bbs[5].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[5].PhiBase)+int32(0))+8)
					}
					if d80.Loc == LocReg {
						ctx.UnprotectReg(d80.Reg)
					} else if d80.Loc == LocRegPair {
						ctx.UnprotectReg(d80.Reg)
						ctx.UnprotectReg(d80.Reg2)
					}
					ps82 := PhiState{General: ps.General}
					ps82.OverlayValues = make([]JITValueDesc, 82)
					ps82.OverlayValues[1] = d1
					ps82.OverlayValues[2] = d2
					ps82.OverlayValues[3] = d3
					ps82.OverlayValues[4] = d4
					ps82.OverlayValues[5] = d5
					ps82.OverlayValues[16] = d16
					ps82.OverlayValues[17] = d17
					ps82.OverlayValues[18] = d18
					ps82.OverlayValues[19] = d19
					ps82.OverlayValues[20] = d20
					ps82.OverlayValues[21] = d21
					ps82.OverlayValues[22] = d22
					ps82.OverlayValues[23] = d23
					ps82.OverlayValues[24] = d24
					ps82.OverlayValues[27] = d27
					ps82.OverlayValues[30] = d30
					ps82.OverlayValues[48] = d48
					ps82.OverlayValues[49] = d49
					ps82.OverlayValues[50] = d50
					ps82.OverlayValues[51] = d51
					ps82.OverlayValues[77] = d77
					ps82.OverlayValues[78] = d78
					ps82.OverlayValues[79] = d79
					ps82.OverlayValues[80] = d80
					ps82.OverlayValues[81] = d81
					ps82.PhiValues = make([]JITValueDesc, 1)
					d83 = d80
					ps82.PhiValues[0] = d83
					if ps82.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps82)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d84 := ps.PhiValues[0]
							ctx.EnsureDesc(&d84)
							ctx.EmitStoreScmerToStack(d84, int32(bbs[5].PhiBase)+int32(0))
						}
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d17.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d17.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d17)
						} else if d17.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d17)
						} else if d17.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d17)
						} else if d17.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d17.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d17 = tmpPair
					} else if d17.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d17.Type, Reg: ctx.AllocRegExcept(d17.Reg), Reg2: ctx.AllocRegExcept(d17.Reg)}
						switch d17.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d17)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d17)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d17)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d17)
						d17 = tmpPair
					}
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg0)")
					}
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
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg1)")
					}
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
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg2)")
					}
					d85 = ctx.EmitGoCallScalar(GoFuncAddr(StrLikeCollation), []JITValueDesc{d17, d20, d1}, 1)
					ctx.BindReg(d85.Reg, &d85)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d85)
					ctx.EnsureDesc(&d85)
					ctx.EmitMakeBool(result, d85)
					if d85.Loc == LocReg {
						ctx.FreeReg(d85.Reg)
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned86 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned86 = append(argPinned86, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned86 = append(argPinned86, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned86 = append(argPinned86, ai.Reg2)
						}
					}
				}
				ps87 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps87)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				for _, r := range argPinned86 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strlike_cs",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			return NewBool(StrLike(String(a[0]), String(a[1])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "matches the string against a wildcard pattern case-sensitively using SQL NULL semantics",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "pattern with % and _ in them"}, &TypeDescriptor{Kind: "string", Label: "collation", Description: "ignored (present for parser compatibility)", Optional: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
				var bbs [4]BBDescriptor
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
						return bbs[3].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJcc(CcNE, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl6)
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
					ctx.EmitMakeNil(result)
					result.Type = tagNil
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
					d16 = args[1]
					d16.ID = 0
					d18 = d16
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d18.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d18)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d18)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d18)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d18.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d18 = tmpPair
					} else if d18.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d18.Reg), Reg2: ctx.AllocRegExcept(d18.Reg)}
						switch d18.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d18)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d18)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d18)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d18)
						d18 = tmpPair
					} else if d18.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d18.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d18.MemPtr))
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
						d18 = tmpPair
					}
					if d18.Loc != LocRegPair && d18.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d18}, 2)
					ctx.FreeDesc(&d16)
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
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d17.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d17.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d17)
						} else if d17.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d17)
						} else if d17.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d17)
						} else if d17.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d17.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d17 = tmpPair
					} else if d17.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d17.Type, Reg: ctx.AllocRegExcept(d17.Reg), Reg2: ctx.AllocRegExcept(d17.Reg)}
						switch d17.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d17)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d17)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d17)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d17)
						d17 = tmpPair
					}
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d14, d17}, 1)
					ctx.BindReg(d19.Reg, &d19)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d19)
					ctx.EmitMakeBool(result, d19)
					if d19.Loc == LocReg {
						ctx.FreeReg(d19.Reg)
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
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
					ctx.ReclaimUntrackedRegs()
					d20 = args[1]
					d20.ID = 0
					d22 = d20
					d22.ID = 0
					d21 = ctx.EmitTagEqualsBorrowed(&d22, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d20)
					d23 = d21
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[0] = d0
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[2] = d2
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[13] = d13
							ps24.OverlayValues[14] = d14
							ps24.OverlayValues[15] = d15
							ps24.OverlayValues[16] = d16
							ps24.OverlayValues[17] = d17
							ps24.OverlayValues[18] = d18
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[1].RenderPS(ps24)
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[0] = d0
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[2] = d2
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[13] = d13
						ps25.OverlayValues[14] = d14
						ps25.OverlayValues[15] = d15
						ps25.OverlayValues[16] = d16
						ps25.OverlayValues[17] = d17
						ps25.OverlayValues[18] = d18
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						return bbs[2].RenderPS(ps25)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJcc(CcNE, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl3)
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 24)
					ps26.OverlayValues[0] = d0
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[2] = d2
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[13] = d13
					ps26.OverlayValues[14] = d14
					ps26.OverlayValues[15] = d15
					ps26.OverlayValues[16] = d16
					ps26.OverlayValues[17] = d17
					ps26.OverlayValues[18] = d18
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[23] = d23
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 24)
					ps27.OverlayValues[0] = d0
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[17] = d17
					ps27.OverlayValues[18] = d18
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					snap28 := d0
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d13
					snap33 := d14
					snap34 := d15
					snap35 := d16
					snap36 := d17
					snap37 := d18
					snap38 := d19
					snap39 := d20
					snap40 := d21
					snap41 := d22
					snap42 := d23
					alloc43 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc43)
					d0 = snap28
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d13 = snap32
					d14 = snap33
					d15 = snap34
					d16 = snap35
					d17 = snap36
					d18 = snap37
					d19 = snap38
					d20 = snap39
					d21 = snap40
					d22 = snap41
					d23 = snap42
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps26)
					}
					return result
					ctx.FreeDesc(&d21)
					return result
				}
				argPinned44 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned44 = append(argPinned44, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned44 = append(argPinned44, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned44 = append(argPinned44, ai.Reg2)
						}
					}
				}
				ps45 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps45)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				for _, r := range argPinned44 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "toLower",

		Fn: func(a ...Scmer) Scmer {
			return NewString(strings.ToLower(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns a string into lower case",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "toUpper",

		Fn: func(a ...Scmer) Scmer {
			return NewString(strings.ToUpper(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns a string into upper case",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "replace",

		Fn: func(a ...Scmer) Scmer {
			return NewString(strings.ReplaceAll(String(a[0]), String(a[1]), String(a[2])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "replaces all occurances in a string with another string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "s", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "find", Description: "search string"}, &TypeDescriptor{Kind: "string", Label: "replace", Description: "replace string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
				d4 := args[1]
				d4.ID = 0
				d6 := d4
				ctx.EnsureDesc(&d6)
				if d6.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d6.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d6)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d6)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d6)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d6.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d6 = tmpPair
				} else if d6.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d6.Reg), Reg2: ctx.AllocRegExcept(d6.Reg)}
					switch d6.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d6)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d6)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d6)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d6)
					d6 = tmpPair
				} else if d6.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d6.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d6.MemPtr))
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
					d6 = tmpPair
				}
				if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d6}, 2)
				ctx.FreeDesc(&d4)
				d7 := args[2]
				d7.ID = 0
				d9 := d7
				ctx.EnsureDesc(&d9)
				if d9.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d9.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d9)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d9)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d9)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d9.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d9 = tmpPair
				} else if d9.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d9.Reg), Reg2: ctx.AllocRegExcept(d9.Reg)}
					switch d9.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d9)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d9)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d9)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d9)
					d9 = tmpPair
				} else if d9.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d9.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d9.MemPtr))
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
					d9 = tmpPair
				}
				if d9.Loc != LocRegPair && d9.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d8 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d9}, 2)
				ctx.FreeDesc(&d7)
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
					panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg0)")
				}
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d5)
				if d5.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d5.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d5)
					} else if d5.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d5)
					} else if d5.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d5)
					} else if d5.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d5 = tmpPair
				} else if d5.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
					switch d5.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d5)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d5)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d5)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d5)
					d5 = tmpPair
				}
				if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg1)")
				}
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d8)
				if d8.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d8.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d8.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d8)
					} else if d8.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d8)
					} else if d8.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d8)
					} else if d8.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d8.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d8 = tmpPair
				} else if d8.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d8.Type, Reg: ctx.AllocRegExcept(d8.Reg), Reg2: ctx.AllocRegExcept(d8.Reg)}
					switch d8.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d8)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d8)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d8)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d8)
					d8 = tmpPair
				}
				if d8.Loc != LocRegPair && d8.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg2)")
				}
				d10 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ReplaceAll), []JITValueDesc{d2, d5, d8}, 2)
				ctx.BindReg(d10.Reg, &d10)
				ctx.BindReg(d10.Reg2, &d10)
				d11 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d10}, 2)
				if result.Loc == LocAny {
					return d11
				}
				ctx.EmitMovPairToResult(&d11, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strtrim",

		Fn: func(a ...Scmer) Scmer {
			return NewString(strings.TrimSpace(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "trims whitespace from both ends of a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strltrim",

		Fn: func(a ...Scmer) Scmer {
			return NewString(strings.TrimLeft(String(a[0]), " \t\n\r"))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "trims whitespace from the left of a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
				}
				d4 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
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
					panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
				}
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d2, d4}, 2)
				ctx.BindReg(d5.Reg, &d5)
				ctx.BindReg(d5.Reg2, &d5)
				d6 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d5}, 2)
				if result.Loc == LocAny {
					return d6
				}
				ctx.EmitMovPairToResult(&d6, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strrtrim",

		Fn: func(a ...Scmer) Scmer {
			return NewString(strings.TrimRight(String(a[0]), " \t\n\r"))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "trims whitespace from the right of a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
				}
				d4 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
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
					panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
				}
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d2, d4}, 2)
				ctx.BindReg(d5.Reg, &d5)
				ctx.BindReg(d5.Reg2, &d5)
				d6 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d5}, 2)
				if result.Loc == LocAny {
					return d6
				}
				ctx.EmitMovPairToResult(&d6, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	// SQL-level NULL-safe wrappers for TRIM/LTRIM/RTRIM
	Declare(&Globalenv, &Declaration{
		Name: "sql_trim",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewString(strings.TrimSpace(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "SQL TRIM(): NULL-safe trim of whitespace from both ends",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
					ctx.EmitMakeNil(result)
					result.Type = tagNil
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
					ctx.BindReg(d16.Reg, &d16)
					ctx.BindReg(d16.Reg2, &d16)
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
				ps19 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps19)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				for _, r := range argPinned18 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_ltrim",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewString(strings.TrimLeft(String(a[0]), " \t\n\r"))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "SQL LTRIM(): NULL-safe trim of whitespace from left",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
					ctx.EmitMakeNil(result)
					result.Type = tagNil
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
					ctx.BindReg(d17.Reg, &d17)
					ctx.BindReg(d17.Reg2, &d17)
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
				ps20 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps20)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				for _, r := range argPinned19 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_rtrim",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewString(strings.TrimRight(String(a[0]), " \t\n\r"))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "SQL RTRIM(): NULL-safe trim of whitespace from right",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
					ctx.EmitMakeNil(result)
					result.Type = tagNil
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
					ctx.BindReg(d17.Reg, &d17)
					ctx.BindReg(d17.Reg2, &d17)
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
				ps20 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps20)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				for _, r := range argPinned19 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "split",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "splits a string using a separator or space",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "separator", Description: "(optional) parameter, defaults to \" \"", Optional: true}},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "string_repeat",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			n := ToInt(a[1])
			if n <= 0 {
				return NewString("")
			}
			return NewString(strings.Repeat(String(a[0]), int(n)))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "repeats a string n times",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to repeat"}, &TypeDescriptor{Kind: "number", Label: "count", Description: "number of repetitions"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
					ctx.EmitMakeNil(result)
					result.Type = tagNil
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
					ctx.BindReg(d14.Reg, &d14)
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
					ctx.BindReg(d34.Reg, &d34)
					ctx.BindReg(d34.Reg2, &d34)
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
				ps37 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps37)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				for _, r := range argPinned36 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})

	/* comparison */
	collation_re := regexp.MustCompile("^([^_]+_)?(.+?)$") // caracterset_language_case
	Declare(&Globalenv, &Declaration{
		Name: "collate",

		Fn: func(a ...Scmer) Scmer {
			collationName := String(a[0])
			reverse := len(a) > 1 && ToBool(a[1])
			key := collateCacheKey{Collation: collationName, Reverse: reverse}
			if cached, ok := collateCache.Load(key); ok {
				return cached.(Scmer)
			}
			// Binary and bare-charset relations are the dominant index case. Keep
			// their canonical callback to one function dispatch: Less already owns
			// the required ASC NULL-first semantics, and swapping its operands owns
			// DESC NULL-last semantics. A closure per metadata key keeps callback
			// identity distinct even when two names have identical byte ordering.
			if collationName == "bin" || collationName == "binary" || collationName == "utf8" || collationName == "utf8mb4" {
				less := func(left, right Scmer) bool {
					if reverse {
						return Less(right, left)
					}
					return Less(left, right)
				}
				fn := func(args ...Scmer) Scmer {
					return NewBool(less(args[0], args[1]))
				}
				result := NewFunc(fn)
				collateRegistry.Store(FunctionIdentity(fn), struct {
					Collation string
					Reverse   bool
				}{Collation: collationName, Reverse: reverse})
				collateLessRegistry.Store(FunctionIdentity(fn), less)
				canonical, _ := collateCache.LoadOrStore(key, result)
				return canonical.(Scmer)
			}
			raw := func() Scmer {
				collation := String(a[0])
				// Bare charset names carry no language/case ordering. SQL columns use
				// them as their default metadata and historically sort bytewise.
				if collation == "utf8" || collation == "utf8mb4" || collation == "binary" {
					collation = "bin"
				}
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
							collateRegistry.Store(FunctionIdentity(f), struct {
								Collation string
								Reverse   bool
							}{Collation: String(a[0]), Reverse: true})
							return NewFunc(f)
						}
						f := func(a ...Scmer) Scmer { return LessScm(a...) }
						collateRegistry.Store(FunctionIdentity(f), struct {
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
							collateRegistry.Store(FunctionIdentity(f), struct {
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
						collateRegistry.Store(FunctionIdentity(f), struct {
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
						collateRegistry.Store(FunctionIdentity(f), struct {
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
					collateRegistry.Store(FunctionIdentity(f), struct {
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
			}()
			rawFn := raw.Func()
			less := func(left, right Scmer) bool {
				leftNil := left.IsNil()
				rightNil := right.IsNil()
				if leftNil || rightNil {
					if leftNil && rightNil {
						return false
					}
					if reverse {
						return !leftNil && rightNil
					}
					return leftNil && !rightNil
				}
				return ToBool(rawFn(left, right))
			}
			fn := func(args ...Scmer) Scmer {
				return NewBool(less(args[0], args[1]))
			}
			result := NewFunc(fn)
			collateRegistry.Store(FunctionIdentity(fn), struct {
				Collation string
				Reverse   bool
			}{Collation: collationName, Reverse: reverse})
			collateLessRegistry.Store(FunctionIdentity(fn), less)
			canonical, _ := collateCache.LoadOrStore(key, result)
			return canonical.(Scmer)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a canonical order relation for a collation and direction. MemCP allows natural sorting of numeric literals.",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "collation", Description: "collation string of the form LANG or LANG_cs or LANG_ci where LANG is a BCP 47 code, for compatibility to MySQL, a CHARSET_ prefix is allowed and ignored as well as the aliases bin, danish, general, german1, german2, spanish and swedish are allowed for language codes"}, &TypeDescriptor{Kind: "bool", Label: "reverse", Description: "whether to reverse the order like in ORDER BY DESC", Optional: true}},
			Return: &TypeDescriptor{Kind: "func",
				Params: []*TypeDescriptor{
					{Kind: "any", Label: "a", Description: "left operand"},
					{Kind: "any", Label: "b", Description: "right operand"},
				},
				Return: &TypeDescriptor{Kind: "bool"},
			},
			Const: true,

			JITEmit: nil,
		},
	})

	/* escaping functions similar to PHP */
	Declare(&Globalenv, &Declaration{
		Name: "htmlentities",

		Fn: func(a ...Scmer) Scmer {
			return NewString(html.EscapeString(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "escapes the string for use in HTML",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (html.EscapeString arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(html.EscapeString), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "urlencode",

		Fn: func(a ...Scmer) Scmer {
			return NewString(url.QueryEscape(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "encodes a string according to URI coding schema",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (url.QueryEscape arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(url.QueryEscape), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "urldecode",

		Fn: func(a ...Scmer) Scmer {
			result, err := url.QueryUnescape(String(a[0]))
			if err != nil {
				panic("error while decoding URL: " + fmt.Sprint(err))
			}
			return NewString(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "decodes a string according to URI coding schema",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to decode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "json_encode",

		Fn: func(a ...Scmer) Scmer {
			b, err := json.Marshal(a[0])
			if err != nil {
				panic(err)
			}
			return NewString(string(b))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "encodes a value in JSON, treats lists as lists",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "value to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "json_quote",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || !a[0].IsString() {
				return NewNil()
			}
			var encoded bytes.Buffer
			encoder := json.NewEncoder(&encoded)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(a[0].String()); err != nil {
				panic(err)
			}
			return NewString(strings.TrimSuffix(encoded.String(), "\n"))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "quotes a string as a JSON string literal without HTML escaping",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "string to quote"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "json_encode_assoc",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "encodes a value in JSON, treats lists as associative arrays",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "value to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "json_decode",

		Fn: func(a ...Scmer) Scmer {
			var result any
			err := json.Unmarshal([]byte(String(a[0])), &result)
			if err != nil {
				panic(err)
			}
			return TransformFromJSON(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses JSON into a map",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to decode"}},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "json_decode_scmer",

		Fn: func(a ...Scmer) Scmer {
			var result Scmer
			err := json.Unmarshal([]byte(String(a[0])), &result)
			if err != nil {
				panic(err)
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses JSON produced by json_encode and preserves Scheme symbols and lists",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "Scmer JSON to decode"}},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "base64_encode",

		Fn: func(a ...Scmer) Scmer {
			return NewString(base64.StdEncoding.EncodeToString([]byte(String(a[0]))))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "encodes a string as Base64 (standard encoding)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "binary string to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "base64_decode",

		Fn: func(a ...Scmer) Scmer {
			decoded, err := base64.StdEncoding.DecodeString(String(a[0]))
			if err != nil {
				panic("error while decoding base64: " + fmt.Sprint(err))
			}
			return NewString(string(decoded))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "decodes a Base64 string (standard encoding)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "base64-encoded string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	sql_escapings := regexp.MustCompile("\\\\[\\\\'\"nr0]")
	Declare(&Globalenv, &Declaration{
		Name: "sql_unescape",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "unescapes the inner part of a sql string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to decode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "bin2hex",

		Fn: func(a ...Scmer) Scmer {
			input := String(a[0])
			result := make([]byte, 2*len(input))
			hexmap := "0123456789abcdef"
			for i := 0; i < len(input); i++ {
				result[2*i] = hexmap[input[i]/16]
				result[2*i+1] = hexmap[input[i]%16]
			}
			return NewString(string(result))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns binary data into hex with lowercase letters",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to decode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "bin2hex",

		Fn: func(a ...Scmer) Scmer {
			input := String(a[0])
			result := make([]byte, 2*len(input))
			hexmap := "0123456789abcdef"
			for i := 0; i < len(input); i++ {
				result[2*i] = hexmap[input[i]/16]
				result[2*i+1] = hexmap[input[i]%16]
			}
			return NewString(string(result))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns binary data into hex with lowercase letters",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "string to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "hex2bin",

		Fn: func(a ...Scmer) Scmer {
			decoded, err := hex.DecodeString(String(a[0]))
			if err != nil {
				panic("error while decoding hex: " + fmt.Sprint(err))
			}
			return NewString(string(decoded))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "decodes a hex string into binary data",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "hex string (even length)"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "uuid",

		Fn: func(a ...Scmer) Scmer {
			id, err := uuid.NewRandom()
			if err != nil {
				panic("error generating UUID: " + fmt.Sprint(err))
			}
			return NewString(id.String())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "generates a new random UUID v4 string",
			Return: &TypeDescriptor{Kind: "string"},
			Const:  false, /* NOT const — each call must return a unique value */

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "randomBytes",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a string with numBytes cryptographically secure random bytes",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", Label: "numBytes", Description: "number of random bytes"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "regexp_replace",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			re, err := regexp.Compile(String(a[1]))
			if err != nil {
				panic("regexp_replace: invalid pattern: " + err.Error())
			}
			return NewString(re.ReplaceAllString(String(a[0]), String(a[2])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "replaces matches of a regex pattern in a string",
			Params:   []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "regex pattern"}, &TypeDescriptor{Kind: "string", Label: "replacement", Description: "replacement string"}},
			Return:   &TypeDescriptor{Kind: "string"},
			Const:    true,
			Optimize: optimizeRegexpReplace,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "fnv_hash",

		Fn: func(a ...Scmer) Scmer {
			return NewString(fnvHashString(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes a fast non-cryptographic 64-bit FNV-1a hash of a string, returns a 16-character hex string",
			Params:   []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string to hash"}},
			Return:   &TypeDescriptor{Kind: "string"},
			Const:    true,
			Optimize: optimizeFNVHash,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d3.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				} else if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
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
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value (fnvHashString arg0)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(fnvHashString), []JITValueDesc{d2}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "stable_structural_hash",

		Fn: func(a ...Scmer) Scmer {
			if len(a) < 1 || len(a) > 2 {
				panic("stable_structural_hash expects a value and optional serialize flag")
			}
			writer := hashTextWriter()
			if len(a) == 2 && a[1].Bool() {
				serializeEx(writer, a[0], &Globalenv, &Globalenv, nil)
			} else {
				WriteStringValue(writer, a[0])
			}
			return NewString(formatStructuralHash(writer.hash))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "streams the string or serialized representation of a Scheme value into stable FNV-1a without constructing the complete representation",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to hash", NoEscape: true},
				{Kind: "bool", Label: "serialize", Description: "use the Scheme serializer instead of string rendering", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sha1",

		Fn: func(a ...Scmer) Scmer {
			sum := sha1.Sum([]byte(String(a[0])))
			return NewString(hex.EncodeToString(sum[:]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes the SHA-1 digest of a string, returns a 40-character lowercase hex string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sha256",

		Fn: func(a ...Scmer) Scmer {
			sum := sha256.Sum256([]byte(String(a[0])))
			return NewString(hex.EncodeToString(sum[:]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes the SHA-256 digest of a string, returns a 64-character lowercase hex string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "regexp_test",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			re, err := regexp.Compile(String(a[1]))
			if err != nil {
				panic("regexp_test: invalid pattern: " + err.Error())
			}
			return NewBool(re.MatchString(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tests if a string matches a regex pattern, returns true/false",
			Params:   []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "regex pattern"}},
			Return:   &TypeDescriptor{Kind: "bool"},
			Const:    true,
			Optimize: optimizeRegexpTest,

			JITEmit: nil,
		},
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
