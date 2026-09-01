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

func asciiFoldByte(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}

func asciiFoldEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := 0; index < len(left); index++ {
		if left[index] >= utf8.RuneSelf || right[index] >= utf8.RuneSelf || asciiFoldByte(left[index]) != asciiFoldByte(right[index]) {
			return false
		}
	}
	return true
}

func asciiFoldContains(value, needle string) (bool, bool) {
	if len(needle) > len(value) {
		return false, true
	}
	for offset := 0; offset <= len(value)-len(needle); offset++ {
		matched := true
		for index := 0; index < len(needle); index++ {
			left, right := value[offset+index], needle[index]
			if left >= utf8.RuneSelf || right >= utf8.RuneSelf {
				return false, false
			}
			if asciiFoldByte(left) != asciiFoldByte(right) {
				matched = false
				break
			}
		}
		if matched {
			return true, true
		}
	}
	return false, true
}

// strLikeASCIIFold handles the exact/prefix/suffix/contains forms emitted by
// ordinary SQL predicates without allocating lower-cased copies per row. The
// second return value is false whenever Unicode or general wildcard matching
// must retain the canonical StrLikeFold path.
func strLikeASCIIFold(value, pattern string) (bool, bool) {
	if strings.ContainsAny(pattern, "_\\\\") {
		return false, false
	}
	wildcards := strings.Count(pattern, "%")
	switch {
	case wildcards == 0:
		for index := 0; index < len(value); index++ {
			if value[index] >= utf8.RuneSelf {
				return false, false
			}
		}
		for index := 0; index < len(pattern); index++ {
			if pattern[index] >= utf8.RuneSelf {
				return false, false
			}
		}
		return asciiFoldEqual(value, pattern), true
	case wildcards == 1 && len(pattern) > 0 && pattern[0] == '%':
		needle := pattern[1:]
		if len(needle) > len(value) {
			return false, true
		}
		matched, ascii := asciiFoldContains(value[len(value)-len(needle):], needle)
		return matched, ascii
	case wildcards == 1 && len(pattern) > 0 && pattern[len(pattern)-1] == '%':
		needle := pattern[:len(pattern)-1]
		if len(needle) > len(value) {
			return false, true
		}
		matched, ascii := asciiFoldContains(value[:len(needle)], needle)
		return matched, ascii
	case wildcards == 2 && len(pattern) >= 2 && pattern[0] == '%' && pattern[len(pattern)-1] == '%':
		return asciiFoldContains(value, pattern[1:len(pattern)-1])
	default:
		return false, false
	}
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
		if matched, handled := strLikeASCIIFold(str, pattern); handled {
			return matched
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["string?"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
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
					panic("jit: generic call arg expects 2-word value ((Scmer).Any arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Any), []JITValueDesc{d0}, 2)
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				callResults2 := JITEmitGoCallResults(ctx, GoFuncAddr(jitAssertString), []JITValueDesc{d1}, []uint8{2, 1}, []uint8{1, 0})
				d3 := callResults2[0]
				d4 := callResults2[1]
				_ = d3
				_ = d4
				ctx.EmitAndRegImm32(d4.Reg, 1)
				d4.Type = tagBool
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d4)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d4.Loc == LocImm {
					ctx.EmitMakeBool(result, d4)
				} else {
					ctx.EmitMakeBool(result, d4)
					ctx.FreeReg(d4.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  8,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["concat"].Fn, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
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
				var d12 JITValueDesc
				_ = d12
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var dynamicArgOff27 int32
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d60 JITValueDesc
				_ = d60
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d95 JITValueDesc
				_ = d95
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [8]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(func() *strings.Builder { return new(strings.Builder) }), nil, 1)
					ctx.BindReg(d2.Reg, &d2)
					ctx.StabilizeDescForControlFlow(&d2)
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d3)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps4 := PhiState{General: ps.General}
					ps4.OverlayValues = make([]JITValueDesc, 4)
					ps4.OverlayValues[1] = d1
					ps4.OverlayValues[2] = d2
					ps4.OverlayValues[3] = d3
					ps4.PhiValues = make([]JITValueDesc, 1)
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps4.PhiValues[0] = d5
					if ps4.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps4)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d6 := ps.PhiValues[0]
							ctx.EnsureDesc(&d6)
							ctx.EmitStoreToStack(d6, int32(bbs[1].PhiBase)+int32(0))
						}
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d7 JITValueDesc
					if d1.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d7)
					}
					if d7.Loc == LocReg && d1.Loc == LocReg && d7.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d7)
					ctx.EmitStoreToStack(d7, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d3)
					var d8 JITValueDesc
					if d7.Loc == LocImm && d3.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.Int() < d3.Imm.Int())}
					} else if d3.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d7.Reg)
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d7.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
							ctx.EmitCmpInt64(d7.Reg, RegR11)
						}
						ctx.EmitSetcc(r0, CondSignedLess)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d8)
					} else if d7.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d3.Reg)
						ctx.EmitSetcc(r1, CondSignedLess)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d8)
					} else {
						r2 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpInt64(d7.Reg, d3.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.FreeDesc(&d3)
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							if ps.General {
							}
							ps10 := PhiState{General: ps.General}
							ps10.OverlayValues = make([]JITValueDesc, 10)
							ps10.OverlayValues[1] = d1
							ps10.OverlayValues[2] = d2
							ps10.OverlayValues[3] = d3
							ps10.OverlayValues[5] = d5
							ps10.OverlayValues[6] = d6
							ps10.OverlayValues[7] = d7
							ps10.OverlayValues[8] = d8
							ps10.OverlayValues[9] = d9
							return bbs[2].RenderPS(ps10)
						}
						if ps.General {
						}
						ps11 := PhiState{General: ps.General}
						ps11.OverlayValues = make([]JITValueDesc, 10)
						ps11.OverlayValues[1] = d1
						ps11.OverlayValues[2] = d2
						ps11.OverlayValues[3] = d3
						ps11.OverlayValues[5] = d5
						ps11.OverlayValues[6] = d6
						ps11.OverlayValues[7] = d7
						ps11.OverlayValues[8] = d8
						ps11.OverlayValues[9] = d9
						return bbs[3].RenderPS(ps11)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d12 := ps.PhiValues[0]
							ctx.EnsureDesc(&d12)
							ctx.EmitStoreToStack(d12, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 13)
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[5] = d5
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[9] = d9
					ps13.OverlayValues[12] = d12
					ps14 := PhiState{General: true}
					ps14.OverlayValues = make([]JITValueDesc, 13)
					ps14.OverlayValues[1] = d1
					ps14.OverlayValues[2] = d2
					ps14.OverlayValues[3] = d3
					ps14.OverlayValues[5] = d5
					ps14.OverlayValues[6] = d6
					ps14.OverlayValues[7] = d7
					ps14.OverlayValues[8] = d8
					ps14.OverlayValues[9] = d9
					ps14.OverlayValues[12] = d12
					snap15 := d1
					snap16 := d2
					snap17 := d3
					snap18 := d5
					snap19 := d6
					snap20 := d7
					snap21 := d8
					snap22 := d9
					snap23 := d12
					alloc24 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps14)
					}
					ctx.RestoreAllocState(alloc24)
					d1 = snap15
					d2 = snap16
					d3 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					d12 = snap23
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps13)
					}
					return result
					ctx.FreeDesc(&d8)
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					var d25 JITValueDesc
					if d7.Loc == LocImm {
						idx := int(d7.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d25 = args[idx]
						d25.ID = 0
					} else {
						ctx.EnsureDesc(&d7)
						protected := make([]Reg, 0, len(args)*2+1)
						seen := make(map[Reg]bool)
						if !seen[d7.Reg] {
							ctx.ProtectReg(d7.Reg)
							seen[d7.Reg] = true
							protected = append(protected, d7.Reg)
						}
						for _, ai := range args {
							if ai.Loc == LocReg {
								if !seen[ai.Reg] {
									ctx.ProtectReg(ai.Reg)
									seen[ai.Reg] = true
									protected = append(protected, ai.Reg)
								}
							} else if ai.Loc == LocRegPair {
								if !seen[ai.Reg] {
									ctx.ProtectReg(ai.Reg)
									seen[ai.Reg] = true
									protected = append(protected, ai.Reg)
								}
								if !seen[ai.Reg2] {
									ctx.ProtectReg(ai.Reg2)
									seen[ai.Reg2] = true
									protected = append(protected, ai.Reg2)
								}
							} else if ai.Loc == LocStackPair {
								// no direct registers to protect
							}
						}
						r3 := ctx.AllocReg()
						r4 := ctx.AllocRegExcept(r3)
						lbl11 := ctx.ReserveLabel()
						lbl12 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d7.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl12)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d7.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							switch ai.Loc {
							case LocRegPair:
								ctx.EmitMovRegReg(r3, ai.Reg)
								ctx.EmitMovRegReg(r4, ai.Reg2)
							case LocStackPair:
								tmp := ai
								ctx.EnsureDesc(&tmp)
								if tmp.Loc != LocRegPair {
									panic("jitgen: emitter args index expected Scmer pair")
								}
								ctx.EmitMovRegReg(r3, tmp.Reg)
								ctx.EmitMovRegReg(r4, tmp.Reg2)
								ctx.FreeDesc(&tmp)
							case LocImm:
								pair := JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
								ctx.BindReg(r3, &pair)
								ctx.BindReg(r4, &pair)
								if ai.Imm.GetTag() == tagInt {
									src := ai
									src.Type = tagInt
									src.Imm = NewInt(ai.Imm.Int())
									ctx.EmitMakeInt(pair, src)
								} else if ai.Imm.GetTag() == tagFloat {
									src := ai
									src.Type = tagFloat
									src.Imm = NewFloat(ai.Imm.Float())
									ctx.EmitMakeFloat(pair, src)
								} else if ai.Imm.GetTag() == tagBool {
									src := ai
									src.Type = tagBool
									src.Imm = NewBool(ai.Imm.Bool())
									ctx.EmitMakeBool(pair, src)
								} else if ai.Imm.GetTag() == tagNil {
									ctx.EmitMakeNil(pair)
								} else {
									ptrWord, auxWord := ai.Imm.RawWords()
									ctx.EmitMovRegImm64(r3, uint64(ptrWord))
									ctx.EmitMovRegImm64(r4, auxWord)
								}
							default:
								panic("jitgen: emitter args index expected Scmer pair")
							}
							ctx.EmitJmp(lbl11)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl12)
						d26 := JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
						ctx.BindReg(r3, &d26)
						ctx.BindReg(r4, &d26)
						ctx.BindReg(r3, &d26)
						ctx.BindReg(r4, &d26)
						ctx.EmitMakeNil(d26)
						ctx.MarkLabel(lbl11)
						for _, r := range protected {
							ctx.UnprotectReg(r)
						}
						d25 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r3, Reg2: r4}
						ctx.BindReg(r3, &d25)
						ctx.BindReg(r4, &d25)
					}
					dynamicArgOff27 = ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d25, int32(dynamicArgOff27))
					ctx.FreeDesc(&d25)
					d25 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff27), Rooted: true}
					ctx.StabilizeDescForControlFlow(&d25)
					d29 = d25
					d29.ID = 0
					d28 = ctx.EmitTagEqualsBorrowed(&d29, tagNil, JITValueDesc{Loc: LocAny})
					d30 = d28
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
							ps31.OverlayValues[5] = d5
							ps31.OverlayValues[6] = d6
							ps31.OverlayValues[7] = d7
							ps31.OverlayValues[8] = d8
							ps31.OverlayValues[9] = d9
							ps31.OverlayValues[12] = d12
							ps31.OverlayValues[25] = d25
							ps31.OverlayValues[26] = d26
							ps31.OverlayValues[28] = d28
							ps31.OverlayValues[29] = d29
							ps31.OverlayValues[30] = d30
							return bbs[4].RenderPS(ps31)
						}
						if ps.General {
						}
						ps32 := PhiState{General: ps.General}
						ps32.OverlayValues = make([]JITValueDesc, 31)
						ps32.OverlayValues[1] = d1
						ps32.OverlayValues[2] = d2
						ps32.OverlayValues[3] = d3
						ps32.OverlayValues[5] = d5
						ps32.OverlayValues[6] = d6
						ps32.OverlayValues[7] = d7
						ps32.OverlayValues[8] = d8
						ps32.OverlayValues[9] = d9
						ps32.OverlayValues[12] = d12
						ps32.OverlayValues[25] = d25
						ps32.OverlayValues[26] = d26
						ps32.OverlayValues[28] = d28
						ps32.OverlayValues[29] = d29
						ps32.OverlayValues[30] = d30
						return bbs[5].RenderPS(ps32)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl6)
					ps33 := PhiState{General: true}
					ps33.OverlayValues = make([]JITValueDesc, 31)
					ps33.OverlayValues[1] = d1
					ps33.OverlayValues[2] = d2
					ps33.OverlayValues[3] = d3
					ps33.OverlayValues[5] = d5
					ps33.OverlayValues[6] = d6
					ps33.OverlayValues[7] = d7
					ps33.OverlayValues[8] = d8
					ps33.OverlayValues[9] = d9
					ps33.OverlayValues[12] = d12
					ps33.OverlayValues[25] = d25
					ps33.OverlayValues[26] = d26
					ps33.OverlayValues[28] = d28
					ps33.OverlayValues[29] = d29
					ps33.OverlayValues[30] = d30
					ps34 := PhiState{General: true}
					ps34.OverlayValues = make([]JITValueDesc, 31)
					ps34.OverlayValues[1] = d1
					ps34.OverlayValues[2] = d2
					ps34.OverlayValues[3] = d3
					ps34.OverlayValues[5] = d5
					ps34.OverlayValues[6] = d6
					ps34.OverlayValues[7] = d7
					ps34.OverlayValues[8] = d8
					ps34.OverlayValues[9] = d9
					ps34.OverlayValues[12] = d12
					ps34.OverlayValues[25] = d25
					ps34.OverlayValues[26] = d26
					ps34.OverlayValues[28] = d28
					ps34.OverlayValues[29] = d29
					ps34.OverlayValues[30] = d30
					snap35 := d1
					snap36 := d2
					snap37 := d3
					snap38 := d5
					snap39 := d6
					snap40 := d7
					snap41 := d8
					snap42 := d9
					snap43 := d12
					snap44 := d25
					snap45 := d26
					snap46 := d28
					snap47 := d29
					snap48 := d30
					alloc49 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps34)
					}
					ctx.RestoreAllocState(alloc49)
					d1 = snap35
					d2 = snap36
					d3 = snap37
					d5 = snap38
					d6 = snap39
					d7 = snap40
					d8 = snap41
					d9 = snap42
					d12 = snap43
					d25 = snap44
					d26 = snap45
					d28 = snap46
					d29 = snap47
					d30 = snap48
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps33)
					}
					return result
					ctx.FreeDesc(&d28)
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs50 := make([]Reg, 0, 3)
					seenBlockPinnedRegs51 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs51
					for _, r := range []Reg{d2.Reg, d2.Reg2, d2.Reg3} {
						live := d2.Loc == LocRegTriple && (r == d2.Reg || r == d2.Reg2 || r == d2.Reg3)
						if live && !seenBlockPinnedRegs51[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs51[r] = true
							blockPinnedRegs50 = append(blockPinnedRegs50, r)
						}
					}
					unpinBlockRegs52 := func() {
						for _, r := range blockPinnedRegs50 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs52()
					d54 = d2
					ctx.EnsureDesc(&d54)
					if d54.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d54.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d54)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d54)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d54)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d54.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d54 = tmpPair
					} else if d54.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d54.Reg), Reg2: ctx.AllocRegExcept(d54.Reg)}
						switch d54.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d54)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d54)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d54)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d54)
						d54 = tmpPair
					} else if d54.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d54.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d54.MemPtr))
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
						d54 = tmpPair
					}
					if d54.Loc != LocRegPair && d54.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d53 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d54}, 2)
					ctx.EnsureDesc(&d53)
					d55 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d53}, 2)
					ctx.EmitMovPairToResult(&d55, &result)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
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
					ctx.ReclaimUntrackedRegs()
					d56 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d56, &result)
						result.Type = d56.Type
					} else {
						switch d56.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d56)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d56)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d56)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d56, &result)
							result.Type = d56.Type
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
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
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs57 := make([]Reg, 0, 3)
					seenBlockPinnedRegs58 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs58
					for _, r := range []Reg{d25.Reg, d25.Reg2, d25.Reg3} {
						live := d25.Loc == LocRegTriple && (r == d25.Reg || r == d25.Reg2 || r == d25.Reg3)
						if live && !seenBlockPinnedRegs58[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs58[r] = true
							blockPinnedRegs57 = append(blockPinnedRegs57, r)
						}
					}
					unpinBlockRegs59 := func() {
						for _, r := range blockPinnedRegs57 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs59()
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d25.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d25.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d25)
						} else if d25.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d25)
						} else if d25.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d25)
						} else if d25.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d25.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d25 = tmpPair
					} else if d25.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d25.Type, Reg: ctx.AllocRegExcept(d25.Reg), Reg2: ctx.AllocRegExcept(d25.Reg)}
						switch d25.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d25)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d25)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d25)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d25)
						d25 = tmpPair
					}
					if d25.Loc != LocRegPair && d25.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((Scmer).Any arg0)")
					}
					ctx.SyncDesc(&d25)
					d60 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Any), []JITValueDesc{d25}, 2)
					ctx.BindReg(d60.Reg, &d60)
					ctx.BindReg(d60.Reg2, &d60)
					ctx.EnsureDesc(&d60)
					callResults61 := JITEmitGoCallResults(ctx, GoFuncAddr(jitAssertReader), []JITValueDesc{d60}, []uint8{2, 1}, []uint8{3, 0})
					d62 = callResults61[0]
					d63 = callResults61[1]
					_ = d62
					_ = d63
					ctx.EmitAndRegImm32(d63.Reg, 1)
					d63.Type = tagBool
					ctx.FreeDesc(&d60)
					ctx.StabilizeDescForControlFlow(&d62)
					d64 = d63
					ctx.EnsureDesc(&d64)
					if d64.Loc != LocImm && d64.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d64.Loc == LocImm {
						if d64.Imm.Bool() {
							if ps.General {
							}
							ps65 := PhiState{General: ps.General}
							ps65.OverlayValues = make([]JITValueDesc, 65)
							ps65.OverlayValues[1] = d1
							ps65.OverlayValues[2] = d2
							ps65.OverlayValues[3] = d3
							ps65.OverlayValues[5] = d5
							ps65.OverlayValues[6] = d6
							ps65.OverlayValues[7] = d7
							ps65.OverlayValues[8] = d8
							ps65.OverlayValues[9] = d9
							ps65.OverlayValues[12] = d12
							ps65.OverlayValues[25] = d25
							ps65.OverlayValues[26] = d26
							ps65.OverlayValues[28] = d28
							ps65.OverlayValues[29] = d29
							ps65.OverlayValues[30] = d30
							ps65.OverlayValues[53] = d53
							ps65.OverlayValues[54] = d54
							ps65.OverlayValues[55] = d55
							ps65.OverlayValues[56] = d56
							ps65.OverlayValues[60] = d60
							ps65.OverlayValues[62] = d62
							ps65.OverlayValues[63] = d63
							ps65.OverlayValues[64] = d64
							return bbs[6].RenderPS(ps65)
						}
						if ps.General {
						}
						ps66 := PhiState{General: ps.General}
						ps66.OverlayValues = make([]JITValueDesc, 65)
						ps66.OverlayValues[1] = d1
						ps66.OverlayValues[2] = d2
						ps66.OverlayValues[3] = d3
						ps66.OverlayValues[5] = d5
						ps66.OverlayValues[6] = d6
						ps66.OverlayValues[7] = d7
						ps66.OverlayValues[8] = d8
						ps66.OverlayValues[9] = d9
						ps66.OverlayValues[12] = d12
						ps66.OverlayValues[25] = d25
						ps66.OverlayValues[26] = d26
						ps66.OverlayValues[28] = d28
						ps66.OverlayValues[29] = d29
						ps66.OverlayValues[30] = d30
						ps66.OverlayValues[53] = d53
						ps66.OverlayValues[54] = d54
						ps66.OverlayValues[55] = d55
						ps66.OverlayValues[56] = d56
						ps66.OverlayValues[60] = d60
						ps66.OverlayValues[62] = d62
						ps66.OverlayValues[63] = d63
						ps66.OverlayValues[64] = d64
						return bbs[7].RenderPS(ps66)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d64.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl8)
					ps67 := PhiState{General: true}
					ps67.OverlayValues = make([]JITValueDesc, 65)
					ps67.OverlayValues[1] = d1
					ps67.OverlayValues[2] = d2
					ps67.OverlayValues[3] = d3
					ps67.OverlayValues[5] = d5
					ps67.OverlayValues[6] = d6
					ps67.OverlayValues[7] = d7
					ps67.OverlayValues[8] = d8
					ps67.OverlayValues[9] = d9
					ps67.OverlayValues[12] = d12
					ps67.OverlayValues[25] = d25
					ps67.OverlayValues[26] = d26
					ps67.OverlayValues[28] = d28
					ps67.OverlayValues[29] = d29
					ps67.OverlayValues[30] = d30
					ps67.OverlayValues[53] = d53
					ps67.OverlayValues[54] = d54
					ps67.OverlayValues[55] = d55
					ps67.OverlayValues[56] = d56
					ps67.OverlayValues[60] = d60
					ps67.OverlayValues[62] = d62
					ps67.OverlayValues[63] = d63
					ps67.OverlayValues[64] = d64
					ps68 := PhiState{General: true}
					ps68.OverlayValues = make([]JITValueDesc, 65)
					ps68.OverlayValues[1] = d1
					ps68.OverlayValues[2] = d2
					ps68.OverlayValues[3] = d3
					ps68.OverlayValues[5] = d5
					ps68.OverlayValues[6] = d6
					ps68.OverlayValues[7] = d7
					ps68.OverlayValues[8] = d8
					ps68.OverlayValues[9] = d9
					ps68.OverlayValues[12] = d12
					ps68.OverlayValues[25] = d25
					ps68.OverlayValues[26] = d26
					ps68.OverlayValues[28] = d28
					ps68.OverlayValues[29] = d29
					ps68.OverlayValues[30] = d30
					ps68.OverlayValues[53] = d53
					ps68.OverlayValues[54] = d54
					ps68.OverlayValues[55] = d55
					ps68.OverlayValues[56] = d56
					ps68.OverlayValues[60] = d60
					ps68.OverlayValues[62] = d62
					ps68.OverlayValues[63] = d63
					ps68.OverlayValues[64] = d64
					snap69 := d1
					snap70 := d2
					snap71 := d3
					snap72 := d5
					snap73 := d6
					snap74 := d7
					snap75 := d8
					snap76 := d9
					snap77 := d12
					snap78 := d25
					snap79 := d26
					snap80 := d28
					snap81 := d29
					snap82 := d30
					snap83 := d53
					snap84 := d54
					snap85 := d55
					snap86 := d56
					snap87 := d60
					snap88 := d62
					snap89 := d63
					snap90 := d64
					alloc91 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps68)
					}
					ctx.RestoreAllocState(alloc91)
					d1 = snap69
					d2 = snap70
					d3 = snap71
					d5 = snap72
					d6 = snap73
					d7 = snap74
					d8 = snap75
					d9 = snap76
					d12 = snap77
					d25 = snap78
					d26 = snap79
					d28 = snap80
					d29 = snap81
					d30 = snap82
					d53 = snap83
					d54 = snap84
					d55 = snap85
					d56 = snap86
					d60 = snap87
					d62 = snap88
					d63 = snap89
					d64 = snap90
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps67)
					}
					return result
					ctx.FreeDesc(&d63)
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
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
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs92 := make([]Reg, 0, 3)
					seenBlockPinnedRegs93 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs93
					for _, r := range []Reg{d2.Reg, d2.Reg2, d2.Reg3} {
						live := d2.Loc == LocRegTriple && (r == d2.Reg || r == d2.Reg2 || r == d2.Reg3)
						if live && !seenBlockPinnedRegs93[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs93[r] = true
							blockPinnedRegs92 = append(blockPinnedRegs92, r)
						}
					}
					unpinBlockRegs94 := func() {
						for _, r := range blockPinnedRegs92 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs94()
					ctx.EnsureDesc(&d2)
					d95 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *strings.Builder) io.Writer { return value }), []JITValueDesc{d2}, 2)
					ctx.EnsureDesc(&d95)
					ctx.EnsureDesc(&d95)
					ctx.EnsureDesc(&d95)
					if d95.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d95.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d95.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d95)
						} else if d95.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d95)
						} else if d95.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d95)
						} else if d95.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d95.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d95 = tmpPair
					} else if d95.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d95.Type, Reg: ctx.AllocRegExcept(d95.Reg), Reg2: ctx.AllocRegExcept(d95.Reg)}
						switch d95.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d95)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d95)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d95)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d95)
						d95 = tmpPair
					}
					if d95.Loc != LocRegPair && d95.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (io.Copy arg0)")
					}
					ctx.EnsureDesc(&d62)
					ctx.EnsureDesc(&d62)
					ctx.EnsureDesc(&d62)
					if d62.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d62.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d62.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d62)
						} else if d62.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d62)
						} else if d62.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d62)
						} else if d62.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d62.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d62 = tmpPair
					} else if d62.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d62.Type, Reg: ctx.AllocRegExcept(d62.Reg), Reg2: ctx.AllocRegExcept(d62.Reg)}
						switch d62.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d62)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d62)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d62)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d62)
						d62 = tmpPair
					}
					if d62.Loc != LocRegPair && d62.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (io.Copy arg1)")
					}
					ctx.SyncDesc(&d95)
					ctx.SyncDesc(&d62)
					callResults96 := JITEmitGoCallResults(ctx, GoFuncAddr(io.Copy), []JITValueDesc{d95, d62}, []uint8{1, 2}, []uint8{0, 3})
					d97 = callResults96[0]
					_ = d97
					d98 = callResults96[1]
					_ = d98
					ctx.FreeDesc(&d62)
					if ps.General {
					}
					ps99 := PhiState{General: ps.General}
					ps99.OverlayValues = make([]JITValueDesc, 99)
					ps99.OverlayValues[1] = d1
					ps99.OverlayValues[2] = d2
					ps99.OverlayValues[3] = d3
					ps99.OverlayValues[5] = d5
					ps99.OverlayValues[6] = d6
					ps99.OverlayValues[7] = d7
					ps99.OverlayValues[8] = d8
					ps99.OverlayValues[9] = d9
					ps99.OverlayValues[12] = d12
					ps99.OverlayValues[25] = d25
					ps99.OverlayValues[26] = d26
					ps99.OverlayValues[28] = d28
					ps99.OverlayValues[29] = d29
					ps99.OverlayValues[30] = d30
					ps99.OverlayValues[53] = d53
					ps99.OverlayValues[54] = d54
					ps99.OverlayValues[55] = d55
					ps99.OverlayValues[56] = d56
					ps99.OverlayValues[60] = d60
					ps99.OverlayValues[62] = d62
					ps99.OverlayValues[63] = d63
					ps99.OverlayValues[64] = d64
					ps99.OverlayValues[95] = d95
					ps99.OverlayValues[97] = d97
					ps99.OverlayValues[98] = d98
					ps99.PhiValues = make([]JITValueDesc, 1)
					if ps99.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps99)
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
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
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
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs100 := make([]Reg, 0, 6)
					seenBlockPinnedRegs101 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs101
					for _, r := range []Reg{d2.Reg, d2.Reg2, d2.Reg3} {
						live := d2.Loc == LocRegTriple && (r == d2.Reg || r == d2.Reg2 || r == d2.Reg3)
						if live && !seenBlockPinnedRegs101[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs101[r] = true
							blockPinnedRegs100 = append(blockPinnedRegs100, r)
						}
					}
					for _, r := range []Reg{d25.Reg, d25.Reg2, d25.Reg3} {
						live := d25.Loc == LocRegTriple && (r == d25.Reg || r == d25.Reg2 || r == d25.Reg3)
						if live && !seenBlockPinnedRegs101[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs101[r] = true
							blockPinnedRegs100 = append(blockPinnedRegs100, r)
						}
					}
					unpinBlockRegs102 := func() {
						for _, r := range blockPinnedRegs100 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs102()
					d104 = d25
					ctx.EnsureDesc(&d104)
					if d104.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d104.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d104)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d104)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d104)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d104.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d104 = tmpPair
					} else if d104.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d104.Reg), Reg2: ctx.AllocRegExcept(d104.Reg)}
						switch d104.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d104)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d104)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d104)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d104)
						d104 = tmpPair
					} else if d104.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d104.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d104.MemPtr))
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
						d104 = tmpPair
					}
					if d104.Loc != LocRegPair && d104.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d103 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d104}, 2)
					ctx.FreeDesc(&d25)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d103.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d103.Imm)
						ptrWord, _ := d103.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d103.Imm.String())))
						d103 = tmpPair
					} else if d103.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d103.Type, Reg: ctx.AllocRegExcept(d103.Reg), Reg2: ctx.AllocRegExcept(d103.Reg)}
						switch d103.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d103)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d103)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d103)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d103)
						d103 = tmpPair
					}
					if d103.Loc != LocRegPair && d103.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*strings.Builder).WriteString arg1)")
					}
					ctx.SyncDesc(&d2)
					ctx.SyncDesc(&d103)
					callResults105 := JITEmitGoCallResults(ctx, GoFuncAddr((*strings.Builder).WriteString), []JITValueDesc{d2, d103}, []uint8{1, 2}, []uint8{0, 3})
					d106 = callResults105[0]
					_ = d106
					d107 = callResults105[1]
					_ = d107
					if ps.General {
					}
					ps108 := PhiState{General: ps.General}
					ps108.OverlayValues = make([]JITValueDesc, 108)
					ps108.OverlayValues[1] = d1
					ps108.OverlayValues[2] = d2
					ps108.OverlayValues[3] = d3
					ps108.OverlayValues[5] = d5
					ps108.OverlayValues[6] = d6
					ps108.OverlayValues[7] = d7
					ps108.OverlayValues[8] = d8
					ps108.OverlayValues[9] = d9
					ps108.OverlayValues[12] = d12
					ps108.OverlayValues[25] = d25
					ps108.OverlayValues[26] = d26
					ps108.OverlayValues[28] = d28
					ps108.OverlayValues[29] = d29
					ps108.OverlayValues[30] = d30
					ps108.OverlayValues[53] = d53
					ps108.OverlayValues[54] = d54
					ps108.OverlayValues[55] = d55
					ps108.OverlayValues[56] = d56
					ps108.OverlayValues[60] = d60
					ps108.OverlayValues[62] = d62
					ps108.OverlayValues[63] = d63
					ps108.OverlayValues[64] = d64
					ps108.OverlayValues[95] = d95
					ps108.OverlayValues[97] = d97
					ps108.OverlayValues[98] = d98
					ps108.OverlayValues[103] = d103
					ps108.OverlayValues[104] = d104
					ps108.OverlayValues[106] = d106
					ps108.OverlayValues[107] = d107
					ps108.PhiValues = make([]JITValueDesc, 1)
					if ps108.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps108)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps109 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps109)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  29,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_concat"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				globalLookup0 := Globalenv.Vars[Symbol("concat")]
				ctx.TrackImm(globalLookup0)
				d1 := JITValueDesc{Loc: LocImm, Type: globalLookup0.GetTag(), Imm: globalLookup0, Rooted: true}
				ctx.EnsureDesc(&d1)
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
					panic("jit: generic call arg expects 2-word value ((Scmer).Func arg0)")
				}
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Func), []JITValueDesc{d1}, 1)
				ctx.BindReg(d2.Reg, &d2)
				d3 := jitMaterializeVirtualGoSlice(ctx, args[0:])
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeGoFunctionSlice), []JITValueDesc{d2, d3}, 2)
				if d4.Loc == LocImm {
					if result.Loc == LocAny {
						return d4
					}
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
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["substr"].Fn, args, result)
				}
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
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
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
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.FreeDesc(&d0)
					d3 = args[1]
					d3.ID = 0
					ctx.EnsureDesc(&d3)
					d4 = d3
					_ = d4
					ctx.StabilizeDescForControlFlow(&d4)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d5 JITValueDesc
					if d4.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int())}
					} else if d4.Type == tagInt && d4.Loc == LocRegPair {
						ctx.FreeReg(d4.Reg)
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
						ctx.BindReg(d4.Reg2, &d5)
						ctx.BindReg(d4.Reg2, &d5)
					} else if d4.Type == tagInt && d4.Loc == LocReg {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d5)
						ctx.BindReg(d4.Reg, &d5)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d4}, 1)
						d5.Type = tagInt
						ctx.BindReg(d5.Reg, &d5)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d3)
					d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d7)
					var d8 JITValueDesc
					if d7.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d7.Reg, 2)
						ctx.EmitSetcc(r0, CondSignedGreater)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d8)
					}
					ctx.FreeDesc(&d7)
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							if ps.General {
							}
							ps10 := PhiState{General: ps.General}
							ps10.OverlayValues = make([]JITValueDesc, 10)
							ps10.OverlayValues[0] = d0
							ps10.OverlayValues[1] = d1
							ps10.OverlayValues[2] = d2
							ps10.OverlayValues[3] = d3
							ps10.OverlayValues[4] = d4
							ps10.OverlayValues[5] = d5
							ps10.OverlayValues[6] = d6
							ps10.OverlayValues[7] = d7
							ps10.OverlayValues[8] = d8
							ps10.OverlayValues[9] = d9
							return bbs[1].RenderPS(ps10)
						}
						if ps.General {
						}
						ps11 := PhiState{General: ps.General}
						ps11.OverlayValues = make([]JITValueDesc, 10)
						ps11.OverlayValues[0] = d0
						ps11.OverlayValues[1] = d1
						ps11.OverlayValues[2] = d2
						ps11.OverlayValues[3] = d3
						ps11.OverlayValues[4] = d4
						ps11.OverlayValues[5] = d5
						ps11.OverlayValues[6] = d6
						ps11.OverlayValues[7] = d7
						ps11.OverlayValues[8] = d8
						ps11.OverlayValues[9] = d9
						return bbs[2].RenderPS(ps11)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 10)
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
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 10)
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
					snap14 := d0
					snap15 := d1
					snap16 := d2
					snap17 := d3
					snap18 := d4
					snap19 := d5
					snap20 := d6
					snap21 := d7
					snap22 := d8
					snap23 := d9
					alloc24 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps13)
					}
					ctx.RestoreAllocState(alloc24)
					d0 = snap14
					d1 = snap15
					d2 = snap16
					d3 = snap17
					d4 = snap18
					d5 = snap19
					d6 = snap20
					d7 = snap21
					d8 = snap22
					d9 = snap23
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
					}
					return result
					ctx.FreeDesc(&d8)
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					ctx.ReclaimUntrackedRegs()
					d25 = args[2]
					d25.ID = 0
					ctx.EnsureDesc(&d25)
					d26 = d25
					_ = d26
					ctx.StabilizeDescForControlFlow(&d26)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.FreeDesc(&d25)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d5)
					ctx.ProtectReg(d5.Reg)
					ctx.EnsureDesc(&d27)
					ctx.UnprotectReg(d5.Reg)
					var d29 JITValueDesc
					if d5.Loc == LocImm && d27.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() + d27.Imm.Int())}
					} else if d27.Loc == LocImm && d27.Imm.Int() == 0 {
						r1 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(r1, d5.Reg)
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d29)
					} else if d5.Loc == LocImm && d5.Imm.Int() == 0 {
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg}
						ctx.BindReg(d27.Reg, &d29)
					} else if d5.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
						ctx.EmitAddInt64(scratch, d27.Reg)
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d29)
					} else if d27.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(scratch, d5.Reg)
						if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d27.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d29)
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg, d27.Reg)
						ctx.EmitMovRegReg(r2, d5.Reg)
						ctx.EmitAddInt64(r2, d27.Reg)
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d29)
					}
					if d29.Loc == LocReg && d5.Loc == LocReg && d29.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.FreeDesc(&d27)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d29)
					var d31 JITValueDesc
					if d29.Loc == LocImm && d5.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int() - d5.Imm.Int())}
					} else {
						r3 := ctx.AllocReg()
						if d29.Loc == LocImm {
							ctx.EmitMovRegImm64(r3, uint64(d29.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r3, d29.Reg)
						}
						if d5.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
							ctx.EmitSubInt64(r3, RegR11)
						} else {
							ctx.EmitSubInt64(r3, d5.Reg)
						}
						d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d31)
					}
					var d32 JITValueDesc
					if d1.Loc == LocImm && d5.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d5.Imm.Int()*1)}
					} else {
						r4 := ctx.AllocReg()
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d1.Reg)
						}
						if d5.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()*1))
							ctx.EmitAddInt64(r4, RegR11)
						} else {
							ctx.EmitAddInt64(r4, d5.Reg)
						}
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d32)
					}
					var d33 JITValueDesc
					var r5 Reg
					var r6 Reg
					ctx.SyncDesc(&d32)
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocImm {
						r5 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r5, uint64(d32.Imm.Int()))
					} else {
						r5 = d32.Reg
					}
					ctx.ProtectReg(r5)
					ctx.SyncDesc(&d31)
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d31.Imm.Int()))
					} else {
						r6 = d31.Reg
					}
					ctx.ProtectReg(r6)
					ctx.UnprotectReg(r6)
					ctx.UnprotectReg(r5)
					d33 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
					ctx.BindReg(r5, &d33)
					ctx.BindReg(r6, &d33)
					ctx.BindReg(r5, &d33)
					ctx.BindReg(r6, &d33)
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d33}, 2)
					ctx.EmitMovPairToResult(&d34, &result)
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					var d35 JITValueDesc
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d35)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d35)
					var d37 JITValueDesc
					if d35.Loc == LocImm && d5.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d35.Imm.Int() - d5.Imm.Int())}
					} else {
						r7 := ctx.AllocReg()
						if d35.Loc == LocImm {
							ctx.EmitMovRegImm64(r7, uint64(d35.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r7, d35.Reg)
						}
						if d5.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
							ctx.EmitSubInt64(r7, RegR11)
						} else {
							ctx.EmitSubInt64(r7, d5.Reg)
						}
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d37)
					}
					var d38 JITValueDesc
					if d1.Loc == LocImm && d5.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d5.Imm.Int()*1)}
					} else {
						r8 := ctx.AllocReg()
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(r8, uint64(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r8, d1.Reg)
						}
						if d5.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()*1))
							ctx.EmitAddInt64(r8, RegR11)
						} else {
							ctx.EmitAddInt64(r8, d5.Reg)
						}
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d38)
					}
					var d39 JITValueDesc
					var r9 Reg
					var r10 Reg
					ctx.SyncDesc(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						r9 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r9, uint64(d38.Imm.Int()))
					} else {
						r9 = d38.Reg
					}
					ctx.ProtectReg(r9)
					ctx.SyncDesc(&d37)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						r10 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r10, uint64(d37.Imm.Int()))
					} else {
						r10 = d37.Reg
					}
					ctx.ProtectReg(r10)
					ctx.UnprotectReg(r10)
					ctx.UnprotectReg(r9)
					d39 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d39)
					ctx.BindReg(r10, &d39)
					ctx.BindReg(r9, &d39)
					ctx.BindReg(r10, &d39)
					ctx.FreeDesc(&d5)
					ctx.EnsureDesc(&d39)
					d40 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d39}, 2)
					ctx.EmitMovPairToResult(&d40, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps41 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps41)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 25,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_substr"].Fn, args, result)
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
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d60 JITValueDesc
				_ = d60
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
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
				var d254 JITValueDesc
				_ = d254
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
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
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
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
					d18 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair {
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
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d19)
					var d22 JITValueDesc
					if d20.SliceSizeKnown {
						d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d20.KnownSliceLen))}
					} else if d20.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d20.Imm.String())))}
					} else if d20.Loc == LocStackTriple {
						d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d20.StackOff + 8, NoHeapPointer: true}
					} else if d20.Loc == LocStackPair {
						d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d20.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d20)
						if d20.Loc == LocRegPair || d20.Loc == LocRegTriple {
							d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg2, ID: 0}
						} else if d20.Loc == LocReg {
							d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d22)
					d23 = args[1]
					d23.ID = 0
					ctx.EnsureDesc(&d23)
					d24 = d23
					_ = d24
					ctx.StabilizeDescForControlFlow(&d24)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d25 JITValueDesc
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d24.Imm.Int())}
					} else if d24.Type == tagInt && d24.Loc == LocRegPair {
						ctx.FreeReg(d24.Reg)
						d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg2}
						ctx.BindReg(d24.Reg2, &d25)
						ctx.BindReg(d24.Reg2, &d25)
					} else if d24.Type == tagInt && d24.Loc == LocReg {
						d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg}
						ctx.BindReg(d24.Reg, &d25)
						ctx.BindReg(d24.Reg, &d25)
					} else {
						d25 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d24}, 1)
						d25.Type = tagInt
						ctx.BindReg(d25.Reg, &d25)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					ctx.FreeDesc(&d23)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					var d27 JITValueDesc
					if d25.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d25.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d25.Reg)
						ctx.EmitMovRegReg(scratch, d25.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d27)
					}
					if d27.Loc == LocReg && d25.Loc == LocReg && d27.Reg == d25.Reg {
						ctx.TransferReg(d25.Reg)
						d25.Loc = LocNone
					}
					ctx.EnsureDesc(&d27)
					ctx.EmitStoreToStack(d27, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.FreeDesc(&d25)
					ctx.EnsureDesc(&d27)
					var d28 JITValueDesc
					if d27.Loc == LocImm {
						d28 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitCmpRegImm32(d27.Reg, 0)
						ctx.EmitSetcc(r0, CondSignedLess)
						d28 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d28)
					}
					d29 = d28
					ctx.EnsureDesc(&d29)
					if d29.Loc != LocImm && d29.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d29.Loc == LocImm {
						if d29.Imm.Bool() {
							if ps.General {
							}
							ps30 := PhiState{General: ps.General}
							ps30.OverlayValues = make([]JITValueDesc, 30)
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
							ps30.OverlayValues[22] = d22
							ps30.OverlayValues[23] = d23
							ps30.OverlayValues[24] = d24
							ps30.OverlayValues[25] = d25
							ps30.OverlayValues[26] = d26
							ps30.OverlayValues[27] = d27
							ps30.OverlayValues[28] = d28
							ps30.OverlayValues[29] = d29
							return bbs[3].RenderPS(ps30)
						}
						if ps.General {
						}
						ps31 := PhiState{General: ps.General}
						ps31.OverlayValues = make([]JITValueDesc, 30)
						ps31.OverlayValues[1] = d1
						ps31.OverlayValues[2] = d2
						ps31.OverlayValues[3] = d3
						ps31.OverlayValues[4] = d4
						ps31.OverlayValues[5] = d5
						ps31.OverlayValues[6] = d6
						ps31.OverlayValues[18] = d18
						ps31.OverlayValues[19] = d19
						ps31.OverlayValues[20] = d20
						ps31.OverlayValues[21] = d21
						ps31.OverlayValues[22] = d22
						ps31.OverlayValues[23] = d23
						ps31.OverlayValues[24] = d24
						ps31.OverlayValues[25] = d25
						ps31.OverlayValues[26] = d26
						ps31.OverlayValues[27] = d27
						ps31.OverlayValues[28] = d28
						ps31.OverlayValues[29] = d29
						ps31.PhiValues = make([]JITValueDesc, 1)
						return bbs[4].RenderPS(ps31)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d29.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl5)
					ps32 := PhiState{General: true}
					ps32.OverlayValues = make([]JITValueDesc, 30)
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
					ps32.OverlayValues[27] = d27
					ps32.OverlayValues[28] = d28
					ps32.OverlayValues[29] = d29
					ps33 := PhiState{General: true}
					ps33.OverlayValues = make([]JITValueDesc, 30)
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
					ps33.OverlayValues[27] = d27
					ps33.OverlayValues[28] = d28
					ps33.OverlayValues[29] = d29
					ps33.PhiValues = make([]JITValueDesc, 1)
					snap34 := d1
					snap35 := d2
					snap36 := d3
					snap37 := d4
					snap38 := d5
					snap39 := d6
					snap40 := d18
					snap41 := d19
					snap42 := d20
					snap43 := d21
					snap44 := d22
					snap45 := d23
					snap46 := d24
					snap47 := d25
					snap48 := d26
					snap49 := d27
					snap50 := d28
					snap51 := d29
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps33)
					}
					ctx.RestoreAllocState(alloc52)
					d1 = snap34
					d2 = snap35
					d3 = snap36
					d4 = snap37
					d5 = snap38
					d6 = snap39
					d18 = snap40
					d19 = snap41
					d20 = snap42
					d21 = snap43
					d22 = snap44
					d23 = snap45
					d24 = snap46
					d25 = snap47
					d26 = snap48
					d27 = snap49
					d28 = snap50
					d29 = snap51
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps32)
					}
					return result
					ctx.FreeDesc(&d28)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					}
					ps53 := PhiState{General: ps.General}
					ps53.OverlayValues = make([]JITValueDesc, 30)
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
					ps53.OverlayValues[22] = d22
					ps53.OverlayValues[23] = d23
					ps53.OverlayValues[24] = d24
					ps53.OverlayValues[25] = d25
					ps53.OverlayValues[26] = d26
					ps53.OverlayValues[27] = d27
					ps53.OverlayValues[28] = d28
					ps53.OverlayValues[29] = d29
					ps53.PhiValues = make([]JITValueDesc, 1)
					d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps53.PhiValues[0] = d54
					if ps53.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps53)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d55 := ps.PhiValues[0]
							ctx.EnsureDesc(&d55)
							ctx.EmitStoreToStack(d55, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d22)
					var d56 JITValueDesc
					if d1.Loc == LocImm && d22.Loc == LocImm {
						d56 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() >= d22.Imm.Int())}
					} else if d22.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d1.Reg)
						if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d22.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedGreaterOrEqual)
						d56 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d56)
					} else if d1.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d22.Reg)
						ctx.EmitSetcc(r2, CondSignedGreaterOrEqual)
						d56 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d56)
					} else {
						r3 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d22.Reg)
						ctx.EmitSetcc(r3, CondSignedGreaterOrEqual)
						d56 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d56)
					}
					d57 = d56
					ctx.EnsureDesc(&d57)
					if d57.Loc != LocImm && d57.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d57.Loc == LocImm {
						if d57.Imm.Bool() {
							if ps.General {
							}
							ps58 := PhiState{General: ps.General}
							ps58.OverlayValues = make([]JITValueDesc, 58)
							ps58.OverlayValues[1] = d1
							ps58.OverlayValues[2] = d2
							ps58.OverlayValues[3] = d3
							ps58.OverlayValues[4] = d4
							ps58.OverlayValues[5] = d5
							ps58.OverlayValues[6] = d6
							ps58.OverlayValues[18] = d18
							ps58.OverlayValues[19] = d19
							ps58.OverlayValues[20] = d20
							ps58.OverlayValues[21] = d21
							ps58.OverlayValues[22] = d22
							ps58.OverlayValues[23] = d23
							ps58.OverlayValues[24] = d24
							ps58.OverlayValues[25] = d25
							ps58.OverlayValues[26] = d26
							ps58.OverlayValues[27] = d27
							ps58.OverlayValues[28] = d28
							ps58.OverlayValues[29] = d29
							ps58.OverlayValues[54] = d54
							ps58.OverlayValues[55] = d55
							ps58.OverlayValues[56] = d56
							ps58.OverlayValues[57] = d57
							return bbs[5].RenderPS(ps58)
						}
						if ps.General {
						}
						ps59 := PhiState{General: ps.General}
						ps59.OverlayValues = make([]JITValueDesc, 58)
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
						ps59.OverlayValues[24] = d24
						ps59.OverlayValues[25] = d25
						ps59.OverlayValues[26] = d26
						ps59.OverlayValues[27] = d27
						ps59.OverlayValues[28] = d28
						ps59.OverlayValues[29] = d29
						ps59.OverlayValues[54] = d54
						ps59.OverlayValues[55] = d55
						ps59.OverlayValues[56] = d56
						ps59.OverlayValues[57] = d57
						return bbs[6].RenderPS(ps59)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d60 := ps.PhiValues[0]
							ctx.EnsureDesc(&d60)
							ctx.EmitStoreToStack(d60, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d57.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl7)
					ps61 := PhiState{General: true}
					ps61.OverlayValues = make([]JITValueDesc, 61)
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
					ps61.OverlayValues[27] = d27
					ps61.OverlayValues[28] = d28
					ps61.OverlayValues[29] = d29
					ps61.OverlayValues[54] = d54
					ps61.OverlayValues[55] = d55
					ps61.OverlayValues[56] = d56
					ps61.OverlayValues[57] = d57
					ps61.OverlayValues[60] = d60
					ps62 := PhiState{General: true}
					ps62.OverlayValues = make([]JITValueDesc, 61)
					ps62.OverlayValues[1] = d1
					ps62.OverlayValues[2] = d2
					ps62.OverlayValues[3] = d3
					ps62.OverlayValues[4] = d4
					ps62.OverlayValues[5] = d5
					ps62.OverlayValues[6] = d6
					ps62.OverlayValues[18] = d18
					ps62.OverlayValues[19] = d19
					ps62.OverlayValues[20] = d20
					ps62.OverlayValues[21] = d21
					ps62.OverlayValues[22] = d22
					ps62.OverlayValues[23] = d23
					ps62.OverlayValues[24] = d24
					ps62.OverlayValues[25] = d25
					ps62.OverlayValues[26] = d26
					ps62.OverlayValues[27] = d27
					ps62.OverlayValues[28] = d28
					ps62.OverlayValues[29] = d29
					ps62.OverlayValues[54] = d54
					ps62.OverlayValues[55] = d55
					ps62.OverlayValues[56] = d56
					ps62.OverlayValues[57] = d57
					ps62.OverlayValues[60] = d60
					snap63 := d1
					snap64 := d2
					snap65 := d3
					snap66 := d4
					snap67 := d5
					snap68 := d6
					snap69 := d18
					snap70 := d19
					snap71 := d20
					snap72 := d21
					snap73 := d22
					snap74 := d23
					snap75 := d24
					snap76 := d25
					snap77 := d26
					snap78 := d27
					snap79 := d28
					snap80 := d29
					snap81 := d54
					snap82 := d55
					snap83 := d56
					snap84 := d57
					snap85 := d60
					alloc86 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps62)
					}
					ctx.RestoreAllocState(alloc86)
					d1 = snap63
					d2 = snap64
					d3 = snap65
					d4 = snap66
					d5 = snap67
					d6 = snap68
					d18 = snap69
					d19 = snap70
					d20 = snap71
					d21 = snap72
					d22 = snap73
					d23 = snap74
					d24 = snap75
					d25 = snap76
					d26 = snap77
					d27 = snap78
					d28 = snap79
					d29 = snap80
					d54 = snap81
					d55 = snap82
					d56 = snap83
					d57 = snap84
					d60 = snap85
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps61)
					}
					return result
					ctx.FreeDesc(&d56)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					ctx.ReclaimUntrackedRegs()
					d87 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d88 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d87}, 2)
					ctx.EmitMovPairToResult(&d88, &result)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					ctx.ReclaimUntrackedRegs()
					d89 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d89)
					var d90 JITValueDesc
					if d89.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d89.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d89.Reg, 2)
						ctx.EmitSetcc(r4, CondSignedGreater)
						d90 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d90)
					}
					ctx.FreeDesc(&d89)
					d91 = d90
					ctx.EnsureDesc(&d91)
					if d91.Loc != LocImm && d91.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d91.Loc == LocImm {
						if d91.Imm.Bool() {
							if ps.General {
							}
							ps92 := PhiState{General: ps.General}
							ps92.OverlayValues = make([]JITValueDesc, 92)
							ps92.OverlayValues[1] = d1
							ps92.OverlayValues[2] = d2
							ps92.OverlayValues[3] = d3
							ps92.OverlayValues[4] = d4
							ps92.OverlayValues[5] = d5
							ps92.OverlayValues[6] = d6
							ps92.OverlayValues[18] = d18
							ps92.OverlayValues[19] = d19
							ps92.OverlayValues[20] = d20
							ps92.OverlayValues[21] = d21
							ps92.OverlayValues[22] = d22
							ps92.OverlayValues[23] = d23
							ps92.OverlayValues[24] = d24
							ps92.OverlayValues[25] = d25
							ps92.OverlayValues[26] = d26
							ps92.OverlayValues[27] = d27
							ps92.OverlayValues[28] = d28
							ps92.OverlayValues[29] = d29
							ps92.OverlayValues[54] = d54
							ps92.OverlayValues[55] = d55
							ps92.OverlayValues[56] = d56
							ps92.OverlayValues[57] = d57
							ps92.OverlayValues[60] = d60
							ps92.OverlayValues[87] = d87
							ps92.OverlayValues[88] = d88
							ps92.OverlayValues[89] = d89
							ps92.OverlayValues[90] = d90
							ps92.OverlayValues[91] = d91
							return bbs[7].RenderPS(ps92)
						}
						if ps.General {
						}
						ps93 := PhiState{General: ps.General}
						ps93.OverlayValues = make([]JITValueDesc, 92)
						ps93.OverlayValues[1] = d1
						ps93.OverlayValues[2] = d2
						ps93.OverlayValues[3] = d3
						ps93.OverlayValues[4] = d4
						ps93.OverlayValues[5] = d5
						ps93.OverlayValues[6] = d6
						ps93.OverlayValues[18] = d18
						ps93.OverlayValues[19] = d19
						ps93.OverlayValues[20] = d20
						ps93.OverlayValues[21] = d21
						ps93.OverlayValues[22] = d22
						ps93.OverlayValues[23] = d23
						ps93.OverlayValues[24] = d24
						ps93.OverlayValues[25] = d25
						ps93.OverlayValues[26] = d26
						ps93.OverlayValues[27] = d27
						ps93.OverlayValues[28] = d28
						ps93.OverlayValues[29] = d29
						ps93.OverlayValues[54] = d54
						ps93.OverlayValues[55] = d55
						ps93.OverlayValues[56] = d56
						ps93.OverlayValues[57] = d57
						ps93.OverlayValues[60] = d60
						ps93.OverlayValues[87] = d87
						ps93.OverlayValues[88] = d88
						ps93.OverlayValues[89] = d89
						ps93.OverlayValues[90] = d90
						ps93.OverlayValues[91] = d91
						return bbs[8].RenderPS(ps93)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d91.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl9)
					ps94 := PhiState{General: true}
					ps94.OverlayValues = make([]JITValueDesc, 92)
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
					ps94.OverlayValues[27] = d27
					ps94.OverlayValues[28] = d28
					ps94.OverlayValues[29] = d29
					ps94.OverlayValues[54] = d54
					ps94.OverlayValues[55] = d55
					ps94.OverlayValues[56] = d56
					ps94.OverlayValues[57] = d57
					ps94.OverlayValues[60] = d60
					ps94.OverlayValues[87] = d87
					ps94.OverlayValues[88] = d88
					ps94.OverlayValues[89] = d89
					ps94.OverlayValues[90] = d90
					ps94.OverlayValues[91] = d91
					ps95 := PhiState{General: true}
					ps95.OverlayValues = make([]JITValueDesc, 92)
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
					ps95.OverlayValues[27] = d27
					ps95.OverlayValues[28] = d28
					ps95.OverlayValues[29] = d29
					ps95.OverlayValues[54] = d54
					ps95.OverlayValues[55] = d55
					ps95.OverlayValues[56] = d56
					ps95.OverlayValues[57] = d57
					ps95.OverlayValues[60] = d60
					ps95.OverlayValues[87] = d87
					ps95.OverlayValues[88] = d88
					ps95.OverlayValues[89] = d89
					ps95.OverlayValues[90] = d90
					ps95.OverlayValues[91] = d91
					snap96 := d1
					snap97 := d2
					snap98 := d3
					snap99 := d4
					snap100 := d5
					snap101 := d6
					snap102 := d18
					snap103 := d19
					snap104 := d20
					snap105 := d21
					snap106 := d22
					snap107 := d23
					snap108 := d24
					snap109 := d25
					snap110 := d26
					snap111 := d27
					snap112 := d28
					snap113 := d29
					snap114 := d54
					snap115 := d55
					snap116 := d56
					snap117 := d57
					snap118 := d60
					snap119 := d87
					snap120 := d88
					snap121 := d89
					snap122 := d90
					snap123 := d91
					alloc124 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps95)
					}
					ctx.RestoreAllocState(alloc124)
					d1 = snap96
					d2 = snap97
					d3 = snap98
					d4 = snap99
					d5 = snap100
					d6 = snap101
					d18 = snap102
					d19 = snap103
					d20 = snap104
					d21 = snap105
					d22 = snap106
					d23 = snap107
					d24 = snap108
					d25 = snap109
					d26 = snap110
					d27 = snap111
					d28 = snap112
					d29 = snap113
					d54 = snap114
					d55 = snap115
					d56 = snap116
					d57 = snap117
					d60 = snap118
					d87 = snap119
					d88 = snap120
					d89 = snap121
					d90 = snap122
					d91 = snap123
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps94)
					}
					return result
					ctx.FreeDesc(&d90)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
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
					ctx.ReclaimUntrackedRegs()
					d125 = args[2]
					d125.ID = 0
					ctx.EnsureDesc(&d125)
					d126 = d125
					_ = d126
					ctx.StabilizeDescForControlFlow(&d126)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d127 JITValueDesc
					if d126.Loc == LocImm {
						d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d126.Imm.Int())}
					} else if d126.Type == tagInt && d126.Loc == LocRegPair {
						ctx.FreeReg(d126.Reg)
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2}
						ctx.BindReg(d126.Reg2, &d127)
						ctx.BindReg(d126.Reg2, &d127)
					} else if d126.Type == tagInt && d126.Loc == LocReg {
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg}
						ctx.BindReg(d126.Reg, &d127)
						ctx.BindReg(d126.Reg, &d127)
					} else {
						d127 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d126}, 1)
						d127.Type = tagInt
						ctx.BindReg(d127.Reg, &d127)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d127)
					ctx.EnsureDesc(&d127)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d127)
					ctx.StabilizeDescForControlFlow(&d127)
					ctx.FreeDesc(&d125)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d127)
					ctx.EnsureDesc(&d1)
					ctx.ProtectReg(d1.Reg)
					ctx.EnsureDesc(&d127)
					ctx.UnprotectReg(d1.Reg)
					var d129 JITValueDesc
					if d1.Loc == LocImm && d127.Loc == LocImm {
						d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d127.Imm.Int())}
					} else if d127.Loc == LocImm && d127.Imm.Int() == 0 {
						r5 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r5, d1.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d129)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d127.Reg}
						ctx.BindReg(d127.Reg, &d129)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d127.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d127.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d129)
					} else if d127.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d127.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d129)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg, d127.Reg)
						ctx.EmitMovRegReg(r6, d1.Reg)
						ctx.EmitAddInt64(r6, d127.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d129)
					}
					if d129.Loc == LocReg && d1.Loc == LocReg && d129.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d22)
					var d130 JITValueDesc
					if d129.Loc == LocImm && d22.Loc == LocImm {
						d130 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d129.Imm.Int() > d22.Imm.Int())}
					} else if d22.Loc == LocImm {
						r7 := ctx.AllocReg()
						if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d129.Reg, int32(d22.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
							ctx.EmitCmpInt64(d129.Reg, RegR11)
						}
						ctx.EmitSetcc(r7, CondSignedGreater)
						d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d130)
					} else if d129.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d129.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d22.Reg)
						ctx.EmitSetcc(r8, CondSignedGreater)
						d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d130)
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpInt64(d129.Reg, d22.Reg)
						ctx.EmitSetcc(r9, CondSignedGreater)
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
							ps132.OverlayValues[22] = d22
							ps132.OverlayValues[23] = d23
							ps132.OverlayValues[24] = d24
							ps132.OverlayValues[25] = d25
							ps132.OverlayValues[26] = d26
							ps132.OverlayValues[27] = d27
							ps132.OverlayValues[28] = d28
							ps132.OverlayValues[29] = d29
							ps132.OverlayValues[54] = d54
							ps132.OverlayValues[55] = d55
							ps132.OverlayValues[56] = d56
							ps132.OverlayValues[57] = d57
							ps132.OverlayValues[60] = d60
							ps132.OverlayValues[87] = d87
							ps132.OverlayValues[88] = d88
							ps132.OverlayValues[89] = d89
							ps132.OverlayValues[90] = d90
							ps132.OverlayValues[91] = d91
							ps132.OverlayValues[125] = d125
							ps132.OverlayValues[126] = d126
							ps132.OverlayValues[127] = d127
							ps132.OverlayValues[128] = d128
							ps132.OverlayValues[129] = d129
							ps132.OverlayValues[130] = d130
							ps132.OverlayValues[131] = d131
							return bbs[9].RenderPS(ps132)
						}
						if ps.General {
							ctx.SyncDesc(&d127)
							if d127.Loc == LocReg {
								ctx.ProtectReg(d127.Reg)
							} else if d127.Loc == LocRegPair {
								ctx.ProtectReg(d127.Reg)
								ctx.ProtectReg(d127.Reg2)
							}
							d133 = d127
							if d133.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d133)
							ctx.EmitStoreToStack(d133, int32(bbs[10].PhiBase)+int32(0))
							if d127.Loc == LocReg {
								ctx.UnprotectReg(d127.Reg)
							} else if d127.Loc == LocRegPair {
								ctx.UnprotectReg(d127.Reg)
								ctx.UnprotectReg(d127.Reg2)
							}
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
						ps134.OverlayValues[27] = d27
						ps134.OverlayValues[28] = d28
						ps134.OverlayValues[29] = d29
						ps134.OverlayValues[54] = d54
						ps134.OverlayValues[55] = d55
						ps134.OverlayValues[56] = d56
						ps134.OverlayValues[57] = d57
						ps134.OverlayValues[60] = d60
						ps134.OverlayValues[87] = d87
						ps134.OverlayValues[88] = d88
						ps134.OverlayValues[89] = d89
						ps134.OverlayValues[90] = d90
						ps134.OverlayValues[91] = d91
						ps134.OverlayValues[125] = d125
						ps134.OverlayValues[126] = d126
						ps134.OverlayValues[127] = d127
						ps134.OverlayValues[128] = d128
						ps134.OverlayValues[129] = d129
						ps134.OverlayValues[130] = d130
						ps134.OverlayValues[131] = d131
						ps134.OverlayValues[133] = d133
						ps134.PhiValues = make([]JITValueDesc, 1)
						d135 = d127
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
					ctx.EmitJump(CondNotEqual, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl23)
					ctx.SyncDesc(&d127)
					if d127.Loc == LocReg {
						ctx.ProtectReg(d127.Reg)
					} else if d127.Loc == LocRegPair {
						ctx.ProtectReg(d127.Reg)
						ctx.ProtectReg(d127.Reg2)
					}
					d136 = d127
					if d136.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d136)
					ctx.EmitStoreToStack(d136, int32(bbs[10].PhiBase)+int32(0))
					if d127.Loc == LocReg {
						ctx.UnprotectReg(d127.Reg)
					} else if d127.Loc == LocRegPair {
						ctx.UnprotectReg(d127.Reg)
						ctx.UnprotectReg(d127.Reg2)
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
					ps137.OverlayValues[27] = d27
					ps137.OverlayValues[28] = d28
					ps137.OverlayValues[29] = d29
					ps137.OverlayValues[54] = d54
					ps137.OverlayValues[55] = d55
					ps137.OverlayValues[56] = d56
					ps137.OverlayValues[57] = d57
					ps137.OverlayValues[60] = d60
					ps137.OverlayValues[87] = d87
					ps137.OverlayValues[88] = d88
					ps137.OverlayValues[89] = d89
					ps137.OverlayValues[90] = d90
					ps137.OverlayValues[91] = d91
					ps137.OverlayValues[125] = d125
					ps137.OverlayValues[126] = d126
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
					ps138.OverlayValues[27] = d27
					ps138.OverlayValues[28] = d28
					ps138.OverlayValues[29] = d29
					ps138.OverlayValues[54] = d54
					ps138.OverlayValues[55] = d55
					ps138.OverlayValues[56] = d56
					ps138.OverlayValues[57] = d57
					ps138.OverlayValues[60] = d60
					ps138.OverlayValues[87] = d87
					ps138.OverlayValues[88] = d88
					ps138.OverlayValues[89] = d89
					ps138.OverlayValues[90] = d90
					ps138.OverlayValues[91] = d91
					ps138.OverlayValues[125] = d125
					ps138.OverlayValues[126] = d126
					ps138.OverlayValues[127] = d127
					ps138.OverlayValues[128] = d128
					ps138.OverlayValues[129] = d129
					ps138.OverlayValues[130] = d130
					ps138.OverlayValues[131] = d131
					ps138.OverlayValues[133] = d133
					ps138.OverlayValues[135] = d135
					ps138.OverlayValues[136] = d136
					ps138.PhiValues = make([]JITValueDesc, 1)
					d139 = d127
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
					snap155 := d27
					snap156 := d28
					snap157 := d29
					snap158 := d54
					snap159 := d55
					snap160 := d56
					snap161 := d57
					snap162 := d60
					snap163 := d87
					snap164 := d88
					snap165 := d89
					snap166 := d90
					snap167 := d91
					snap168 := d125
					snap169 := d126
					snap170 := d127
					snap171 := d128
					snap172 := d129
					snap173 := d130
					snap174 := d131
					snap175 := d133
					snap176 := d135
					snap177 := d136
					snap178 := d139
					alloc179 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps138)
					}
					ctx.RestoreAllocState(alloc179)
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
					d27 = snap155
					d28 = snap156
					d29 = snap157
					d54 = snap158
					d55 = snap159
					d56 = snap160
					d57 = snap161
					d60 = snap162
					d87 = snap163
					d88 = snap164
					d89 = snap165
					d90 = snap166
					d91 = snap167
					d125 = snap168
					d126 = snap169
					d127 = snap170
					d128 = snap171
					d129 = snap172
					d130 = snap173
					d131 = snap174
					d133 = snap175
					d135 = snap176
					d136 = snap177
					d139 = snap178
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
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
					var d180 JITValueDesc
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocRegPair || d20.Loc == LocRegTriple {
						d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg2}
						ctx.BindReg(d20.Reg2, &d180)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d180)
					var d182 JITValueDesc
					if d180.Loc == LocImm && d1.Loc == LocImm {
						d182 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d180.Imm.Int() - d1.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d180.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d180.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d180.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r10, RegR11)
						} else {
							ctx.EmitSubInt64(r10, d1.Reg)
						}
						d182 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d182)
					}
					var d183 JITValueDesc
					if d20.Loc == LocImm && d1.Loc == LocImm {
						d183 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int() + d1.Imm.Int()*1)}
					} else {
						r11 := ctx.AllocReg()
						if d20.Loc == LocImm {
							ctx.EmitMovRegImm64(r11, uint64(d20.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r11, d20.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()*1))
							ctx.EmitAddInt64(r11, RegR11)
						} else {
							ctx.EmitAddInt64(r11, d1.Reg)
						}
						d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d183)
					}
					var d184 JITValueDesc
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d183)
					ctx.EnsureDesc(&d183)
					if d183.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d183.Imm.Int()))
					} else {
						r12 = d183.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d182)
					ctx.EnsureDesc(&d182)
					if d182.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d182.Imm.Int()))
					} else {
						r13 = d182.Reg
					}
					ctx.ProtectReg(r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d184 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d184)
					ctx.BindReg(r13, &d184)
					ctx.BindReg(r12, &d184)
					ctx.BindReg(r13, &d184)
					ctx.EnsureDesc(&d184)
					d185 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d184}, 2)
					ctx.EmitMovPairToResult(&d185, &result)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d22)
					ctx.ProtectReg(d22.Reg)
					ctx.EnsureDesc(&d1)
					ctx.UnprotectReg(d22.Reg)
					var d186 JITValueDesc
					if d22.Loc == LocImm && d1.Loc == LocImm {
						d186 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d22.Imm.Int() - d1.Imm.Int())}
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						r14 := ctx.AllocRegExcept(d22.Reg)
						ctx.EmitMovRegReg(r14, d22.Reg)
						d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d186)
					} else if d22.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
						ctx.EmitSubInt64(scratch, d1.Reg)
						d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d186)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d22.Reg)
						ctx.EmitMovRegReg(scratch, d22.Reg)
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d186)
					} else {
						r15 := ctx.AllocRegExcept(d22.Reg, d1.Reg)
						ctx.EmitMovRegReg(r15, d22.Reg)
						ctx.EmitSubInt64(r15, d1.Reg)
						d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d186)
					}
					if d186.Loc == LocReg && d22.Loc == LocReg && d186.Reg == d22.Reg {
						ctx.TransferReg(d22.Reg)
						d22.Loc = LocNone
					}
					ctx.EnsureDesc(&d186)
					ctx.EmitStoreToStack(d186, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d186)
					ctx.FreeDesc(&d22)
					if ps.General {
					}
					ps187 := PhiState{General: ps.General}
					ps187.OverlayValues = make([]JITValueDesc, 187)
					ps187.OverlayValues[1] = d1
					ps187.OverlayValues[2] = d2
					ps187.OverlayValues[3] = d3
					ps187.OverlayValues[4] = d4
					ps187.OverlayValues[5] = d5
					ps187.OverlayValues[6] = d6
					ps187.OverlayValues[18] = d18
					ps187.OverlayValues[19] = d19
					ps187.OverlayValues[20] = d20
					ps187.OverlayValues[21] = d21
					ps187.OverlayValues[22] = d22
					ps187.OverlayValues[23] = d23
					ps187.OverlayValues[24] = d24
					ps187.OverlayValues[25] = d25
					ps187.OverlayValues[26] = d26
					ps187.OverlayValues[27] = d27
					ps187.OverlayValues[28] = d28
					ps187.OverlayValues[29] = d29
					ps187.OverlayValues[54] = d54
					ps187.OverlayValues[55] = d55
					ps187.OverlayValues[56] = d56
					ps187.OverlayValues[57] = d57
					ps187.OverlayValues[60] = d60
					ps187.OverlayValues[87] = d87
					ps187.OverlayValues[88] = d88
					ps187.OverlayValues[89] = d89
					ps187.OverlayValues[90] = d90
					ps187.OverlayValues[91] = d91
					ps187.OverlayValues[125] = d125
					ps187.OverlayValues[126] = d126
					ps187.OverlayValues[127] = d127
					ps187.OverlayValues[128] = d128
					ps187.OverlayValues[129] = d129
					ps187.OverlayValues[130] = d130
					ps187.OverlayValues[131] = d131
					ps187.OverlayValues[133] = d133
					ps187.OverlayValues[135] = d135
					ps187.OverlayValues[136] = d136
					ps187.OverlayValues[139] = d139
					ps187.OverlayValues[180] = d180
					ps187.OverlayValues[181] = d181
					ps187.OverlayValues[182] = d182
					ps187.OverlayValues[183] = d183
					ps187.OverlayValues[184] = d184
					ps187.OverlayValues[185] = d185
					ps187.OverlayValues[186] = d186
					ps187.PhiValues = make([]JITValueDesc, 1)
					if ps187.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps187)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d2)
					var d189 JITValueDesc
					if d2.Loc == LocImm {
						d189 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 0)}
					} else {
						r16 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r16, CondSignedLess)
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
							if ps.General {
							}
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
							ps191.OverlayValues[27] = d27
							ps191.OverlayValues[28] = d28
							ps191.OverlayValues[29] = d29
							ps191.OverlayValues[54] = d54
							ps191.OverlayValues[55] = d55
							ps191.OverlayValues[56] = d56
							ps191.OverlayValues[57] = d57
							ps191.OverlayValues[60] = d60
							ps191.OverlayValues[87] = d87
							ps191.OverlayValues[88] = d88
							ps191.OverlayValues[89] = d89
							ps191.OverlayValues[90] = d90
							ps191.OverlayValues[91] = d91
							ps191.OverlayValues[125] = d125
							ps191.OverlayValues[126] = d126
							ps191.OverlayValues[127] = d127
							ps191.OverlayValues[128] = d128
							ps191.OverlayValues[129] = d129
							ps191.OverlayValues[130] = d130
							ps191.OverlayValues[131] = d131
							ps191.OverlayValues[133] = d133
							ps191.OverlayValues[135] = d135
							ps191.OverlayValues[136] = d136
							ps191.OverlayValues[139] = d139
							ps191.OverlayValues[180] = d180
							ps191.OverlayValues[181] = d181
							ps191.OverlayValues[182] = d182
							ps191.OverlayValues[183] = d183
							ps191.OverlayValues[184] = d184
							ps191.OverlayValues[185] = d185
							ps191.OverlayValues[186] = d186
							ps191.OverlayValues[188] = d188
							ps191.OverlayValues[189] = d189
							ps191.OverlayValues[190] = d190
							return bbs[11].RenderPS(ps191)
						}
						if ps.General {
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
						ps192.OverlayValues[27] = d27
						ps192.OverlayValues[28] = d28
						ps192.OverlayValues[29] = d29
						ps192.OverlayValues[54] = d54
						ps192.OverlayValues[55] = d55
						ps192.OverlayValues[56] = d56
						ps192.OverlayValues[57] = d57
						ps192.OverlayValues[60] = d60
						ps192.OverlayValues[87] = d87
						ps192.OverlayValues[88] = d88
						ps192.OverlayValues[89] = d89
						ps192.OverlayValues[90] = d90
						ps192.OverlayValues[91] = d91
						ps192.OverlayValues[125] = d125
						ps192.OverlayValues[126] = d126
						ps192.OverlayValues[127] = d127
						ps192.OverlayValues[128] = d128
						ps192.OverlayValues[129] = d129
						ps192.OverlayValues[130] = d130
						ps192.OverlayValues[131] = d131
						ps192.OverlayValues[133] = d133
						ps192.OverlayValues[135] = d135
						ps192.OverlayValues[136] = d136
						ps192.OverlayValues[139] = d139
						ps192.OverlayValues[180] = d180
						ps192.OverlayValues[181] = d181
						ps192.OverlayValues[182] = d182
						ps192.OverlayValues[183] = d183
						ps192.OverlayValues[184] = d184
						ps192.OverlayValues[185] = d185
						ps192.OverlayValues[186] = d186
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
					ctx.EmitJump(CondNotEqual, lbl24)
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
					ps194.OverlayValues[27] = d27
					ps194.OverlayValues[28] = d28
					ps194.OverlayValues[29] = d29
					ps194.OverlayValues[54] = d54
					ps194.OverlayValues[55] = d55
					ps194.OverlayValues[56] = d56
					ps194.OverlayValues[57] = d57
					ps194.OverlayValues[60] = d60
					ps194.OverlayValues[87] = d87
					ps194.OverlayValues[88] = d88
					ps194.OverlayValues[89] = d89
					ps194.OverlayValues[90] = d90
					ps194.OverlayValues[91] = d91
					ps194.OverlayValues[125] = d125
					ps194.OverlayValues[126] = d126
					ps194.OverlayValues[127] = d127
					ps194.OverlayValues[128] = d128
					ps194.OverlayValues[129] = d129
					ps194.OverlayValues[130] = d130
					ps194.OverlayValues[131] = d131
					ps194.OverlayValues[133] = d133
					ps194.OverlayValues[135] = d135
					ps194.OverlayValues[136] = d136
					ps194.OverlayValues[139] = d139
					ps194.OverlayValues[180] = d180
					ps194.OverlayValues[181] = d181
					ps194.OverlayValues[182] = d182
					ps194.OverlayValues[183] = d183
					ps194.OverlayValues[184] = d184
					ps194.OverlayValues[185] = d185
					ps194.OverlayValues[186] = d186
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
					ps195.OverlayValues[27] = d27
					ps195.OverlayValues[28] = d28
					ps195.OverlayValues[29] = d29
					ps195.OverlayValues[54] = d54
					ps195.OverlayValues[55] = d55
					ps195.OverlayValues[56] = d56
					ps195.OverlayValues[57] = d57
					ps195.OverlayValues[60] = d60
					ps195.OverlayValues[87] = d87
					ps195.OverlayValues[88] = d88
					ps195.OverlayValues[89] = d89
					ps195.OverlayValues[90] = d90
					ps195.OverlayValues[91] = d91
					ps195.OverlayValues[125] = d125
					ps195.OverlayValues[126] = d126
					ps195.OverlayValues[127] = d127
					ps195.OverlayValues[128] = d128
					ps195.OverlayValues[129] = d129
					ps195.OverlayValues[130] = d130
					ps195.OverlayValues[131] = d131
					ps195.OverlayValues[133] = d133
					ps195.OverlayValues[135] = d135
					ps195.OverlayValues[136] = d136
					ps195.OverlayValues[139] = d139
					ps195.OverlayValues[180] = d180
					ps195.OverlayValues[181] = d181
					ps195.OverlayValues[182] = d182
					ps195.OverlayValues[183] = d183
					ps195.OverlayValues[184] = d184
					ps195.OverlayValues[185] = d185
					ps195.OverlayValues[186] = d186
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
					snap211 := d27
					snap212 := d28
					snap213 := d29
					snap214 := d54
					snap215 := d55
					snap216 := d56
					snap217 := d57
					snap218 := d60
					snap219 := d87
					snap220 := d88
					snap221 := d89
					snap222 := d90
					snap223 := d91
					snap224 := d125
					snap225 := d126
					snap226 := d127
					snap227 := d128
					snap228 := d129
					snap229 := d130
					snap230 := d131
					snap231 := d133
					snap232 := d135
					snap233 := d136
					snap234 := d139
					snap235 := d180
					snap236 := d181
					snap237 := d182
					snap238 := d183
					snap239 := d184
					snap240 := d185
					snap241 := d186
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
					d27 = snap211
					d28 = snap212
					d29 = snap213
					d54 = snap214
					d55 = snap215
					d56 = snap216
					d57 = snap217
					d60 = snap218
					d87 = snap219
					d88 = snap220
					d89 = snap221
					d90 = snap222
					d91 = snap223
					d125 = snap224
					d126 = snap225
					d127 = snap226
					d128 = snap227
					d129 = snap228
					d130 = snap229
					d131 = snap230
					d133 = snap231
					d135 = snap232
					d136 = snap233
					d139 = snap234
					d180 = snap235
					d181 = snap236
					d182 = snap237
					d183 = snap238
					d184 = snap239
					d185 = snap240
					d186 = snap241
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
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
					d247 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d248 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d247}, 2)
					ctx.EmitMovPairToResult(&d248, &result)
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
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
					if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != LocNone {
						d248 = ps.OverlayValues[248]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d1)
					ctx.ProtectReg(d1.Reg)
					ctx.EnsureDesc(&d2)
					ctx.UnprotectReg(d1.Reg)
					var d249 JITValueDesc
					if d1.Loc == LocImm && d2.Loc == LocImm {
						d249 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d2.Imm.Int())}
					} else if d2.Loc == LocImm && d2.Imm.Int() == 0 {
						r17 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r17, d1.Reg)
						d249 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d249)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d249 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
						ctx.BindReg(d2.Reg, &d249)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d2.Reg)
						d249 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d249)
					} else if d2.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d249 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d249)
					} else {
						r18 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
						ctx.EmitMovRegReg(r18, d1.Reg)
						ctx.EmitAddInt64(r18, d2.Reg)
						d249 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d249)
					}
					if d249.Loc == LocReg && d1.Loc == LocReg && d249.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d249)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d249)
					var d251 JITValueDesc
					if d249.Loc == LocImm && d1.Loc == LocImm {
						d251 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d249.Imm.Int() - d1.Imm.Int())}
					} else {
						r19 := ctx.AllocReg()
						if d249.Loc == LocImm {
							ctx.EmitMovRegImm64(r19, uint64(d249.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r19, d249.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r19, RegR11)
						} else {
							ctx.EmitSubInt64(r19, d1.Reg)
						}
						d251 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d251)
					}
					var d252 JITValueDesc
					if d20.Loc == LocImm && d1.Loc == LocImm {
						d252 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int() + d1.Imm.Int()*1)}
					} else {
						r20 := ctx.AllocReg()
						if d20.Loc == LocImm {
							ctx.EmitMovRegImm64(r20, uint64(d20.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r20, d20.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()*1))
							ctx.EmitAddInt64(r20, RegR11)
						} else {
							ctx.EmitAddInt64(r20, d1.Reg)
						}
						d252 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
						ctx.BindReg(r20, &d252)
					}
					var d253 JITValueDesc
					var r21 Reg
					var r22 Reg
					ctx.SyncDesc(&d252)
					ctx.EnsureDesc(&d252)
					if d252.Loc == LocImm {
						r21 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r21, uint64(d252.Imm.Int()))
					} else {
						r21 = d252.Reg
					}
					ctx.ProtectReg(r21)
					ctx.SyncDesc(&d251)
					ctx.EnsureDesc(&d251)
					if d251.Loc == LocImm {
						r22 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r22, uint64(d251.Imm.Int()))
					} else {
						r22 = d251.Reg
					}
					ctx.ProtectReg(r22)
					ctx.UnprotectReg(r22)
					ctx.UnprotectReg(r21)
					d253 = JITValueDesc{Loc: LocRegPair, Reg: r21, Reg2: r22}
					ctx.BindReg(r21, &d253)
					ctx.BindReg(r22, &d253)
					ctx.BindReg(r21, &d253)
					ctx.BindReg(r22, &d253)
					ctx.FreeDesc(&d1)
					ctx.FreeDesc(&d249)
					ctx.EnsureDesc(&d253)
					d254 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d253}, 2)
					ctx.EmitMovPairToResult(&d254, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps255 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps255)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
				return result
			},
			JITInlineCost: 51,
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
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["simplify"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (Simplify arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(Simplify), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
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
			},
			JITInlineCost: 5,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strlen"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
				var d3 JITValueDesc
				if d1.SliceSizeKnown {
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.KnownSliceLen))}
				} else if d1.Loc == LocImm {
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d1.Imm.String())))}
				} else if d1.Loc == LocStackTriple {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d1.StackOff + 8, NoHeapPointer: true}
				} else if d1.Loc == LocStackPair {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d1.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2, ID: 0}
					} else if d1.Loc == LocReg {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
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
			},
			JITInlineCost: 7,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strlike"].Fn, args, result)
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
				var d28 JITValueDesc
				_ = d28
				var d31 JITValueDesc
				_ = d31
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
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
				var d91 JITValueDesc
				_ = d91
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
						return bbs[3].RenderPS(ps7)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
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
					d16 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
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
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					d20 = args[1]
					d20.ID = 0
					d22 = d20
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d22.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d22)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d22)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d22)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d22.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d22 = tmpPair
					} else if d22.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d22.Reg), Reg2: ctx.AllocRegExcept(d22.Reg)}
						switch d22.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d22)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d22)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d22)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d22)
						d22 = tmpPair
					} else if d22.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d22.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d22.MemPtr))
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
						d22 = tmpPair
					}
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d21 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d22}, 2)
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d20)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d23)
					var d24 JITValueDesc
					if d23.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d23.Reg, 2)
						ctx.EmitSetcc(r0, CondSignedGreater)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d24)
					}
					ctx.FreeDesc(&d23)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							if ps.General {
							}
							ps26 := PhiState{General: ps.General}
							ps26.OverlayValues = make([]JITValueDesc, 26)
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
							ps26.OverlayValues[25] = d25
							return bbs[4].RenderPS(ps26)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
						}
						ps27 := PhiState{General: ps.General}
						ps27.OverlayValues = make([]JITValueDesc, 26)
						ps27.OverlayValues[1] = d1
						ps27.OverlayValues[2] = d2
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[16] = d16
						ps27.OverlayValues[17] = d17
						ps27.OverlayValues[18] = d18
						ps27.OverlayValues[19] = d19
						ps27.OverlayValues[20] = d20
						ps27.OverlayValues[21] = d21
						ps27.OverlayValues[22] = d22
						ps27.OverlayValues[23] = d23
						ps27.OverlayValues[24] = d24
						ps27.OverlayValues[25] = d25
						ps27.PhiValues = make([]JITValueDesc, 1)
						d28 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
						ps27.PhiValues[0] = d28
						return bbs[5].RenderPS(ps27)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
					ctx.EmitJmp(lbl6)
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 29)
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
					ps29.OverlayValues[25] = d25
					ps29.OverlayValues[28] = d28
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 29)
					ps30.OverlayValues[1] = d1
					ps30.OverlayValues[2] = d2
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[16] = d16
					ps30.OverlayValues[17] = d17
					ps30.OverlayValues[18] = d18
					ps30.OverlayValues[19] = d19
					ps30.OverlayValues[20] = d20
					ps30.OverlayValues[21] = d21
					ps30.OverlayValues[22] = d22
					ps30.OverlayValues[23] = d23
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[28] = d28
					ps30.PhiValues = make([]JITValueDesc, 1)
					d31 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
					ps30.PhiValues[0] = d31
					snap32 := d1
					snap33 := d2
					snap34 := d3
					snap35 := d4
					snap36 := d5
					snap37 := d16
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d22
					snap44 := d23
					snap45 := d24
					snap46 := d25
					snap47 := d28
					snap48 := d31
					alloc49 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc49)
					d1 = snap32
					d2 = snap33
					d3 = snap34
					d4 = snap35
					d5 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d23 = snap44
					d24 = snap45
					d25 = snap46
					d28 = snap47
					d31 = snap48
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps29)
					}
					return result
					ctx.FreeDesc(&d24)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					ctx.ReclaimUntrackedRegs()
					d50 = args[1]
					d50.ID = 0
					d52 = d50
					d52.ID = 0
					d51 = ctx.EmitTagEqualsBorrowed(&d52, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d50)
					d53 = d51
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
							ps54.OverlayValues[25] = d25
							ps54.OverlayValues[28] = d28
							ps54.OverlayValues[31] = d31
							ps54.OverlayValues[50] = d50
							ps54.OverlayValues[51] = d51
							ps54.OverlayValues[52] = d52
							ps54.OverlayValues[53] = d53
							return bbs[1].RenderPS(ps54)
						}
						if ps.General {
						}
						ps55 := PhiState{General: ps.General}
						ps55.OverlayValues = make([]JITValueDesc, 54)
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
						ps55.OverlayValues[25] = d25
						ps55.OverlayValues[28] = d28
						ps55.OverlayValues[31] = d31
						ps55.OverlayValues[50] = d50
						ps55.OverlayValues[51] = d51
						ps55.OverlayValues[52] = d52
						ps55.OverlayValues[53] = d53
						return bbs[2].RenderPS(ps55)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d53.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl3)
					ps56 := PhiState{General: true}
					ps56.OverlayValues = make([]JITValueDesc, 54)
					ps56.OverlayValues[1] = d1
					ps56.OverlayValues[2] = d2
					ps56.OverlayValues[3] = d3
					ps56.OverlayValues[4] = d4
					ps56.OverlayValues[5] = d5
					ps56.OverlayValues[16] = d16
					ps56.OverlayValues[17] = d17
					ps56.OverlayValues[18] = d18
					ps56.OverlayValues[19] = d19
					ps56.OverlayValues[20] = d20
					ps56.OverlayValues[21] = d21
					ps56.OverlayValues[22] = d22
					ps56.OverlayValues[23] = d23
					ps56.OverlayValues[24] = d24
					ps56.OverlayValues[25] = d25
					ps56.OverlayValues[28] = d28
					ps56.OverlayValues[31] = d31
					ps56.OverlayValues[50] = d50
					ps56.OverlayValues[51] = d51
					ps56.OverlayValues[52] = d52
					ps56.OverlayValues[53] = d53
					ps57 := PhiState{General: true}
					ps57.OverlayValues = make([]JITValueDesc, 54)
					ps57.OverlayValues[1] = d1
					ps57.OverlayValues[2] = d2
					ps57.OverlayValues[3] = d3
					ps57.OverlayValues[4] = d4
					ps57.OverlayValues[5] = d5
					ps57.OverlayValues[16] = d16
					ps57.OverlayValues[17] = d17
					ps57.OverlayValues[18] = d18
					ps57.OverlayValues[19] = d19
					ps57.OverlayValues[20] = d20
					ps57.OverlayValues[21] = d21
					ps57.OverlayValues[22] = d22
					ps57.OverlayValues[23] = d23
					ps57.OverlayValues[24] = d24
					ps57.OverlayValues[25] = d25
					ps57.OverlayValues[28] = d28
					ps57.OverlayValues[31] = d31
					ps57.OverlayValues[50] = d50
					ps57.OverlayValues[51] = d51
					ps57.OverlayValues[52] = d52
					ps57.OverlayValues[53] = d53
					snap58 := d1
					snap59 := d2
					snap60 := d3
					snap61 := d4
					snap62 := d5
					snap63 := d16
					snap64 := d17
					snap65 := d18
					snap66 := d19
					snap67 := d20
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d24
					snap72 := d25
					snap73 := d28
					snap74 := d31
					snap75 := d50
					snap76 := d51
					snap77 := d52
					snap78 := d53
					alloc79 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps57)
					}
					ctx.RestoreAllocState(alloc79)
					d1 = snap58
					d2 = snap59
					d3 = snap60
					d4 = snap61
					d5 = snap62
					d16 = snap63
					d17 = snap64
					d18 = snap65
					d19 = snap66
					d20 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d24 = snap71
					d25 = snap72
					d28 = snap73
					d31 = snap74
					d50 = snap75
					d51 = snap76
					d52 = snap77
					d53 = snap78
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps56)
					}
					return result
					ctx.FreeDesc(&d51)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					ctx.ReclaimUntrackedRegs()
					d80 = args[2]
					d80.ID = 0
					d82 = d80
					ctx.EnsureDesc(&d82)
					if d82.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d82.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d82)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d82)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d82)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d82.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d82 = tmpPair
					} else if d82.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d82.Reg), Reg2: ctx.AllocRegExcept(d82.Reg)}
						switch d82.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d82)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d82)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d82)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d82)
						d82 = tmpPair
					} else if d82.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d82.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d82.MemPtr))
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
						d82 = tmpPair
					}
					if d82.Loc != LocRegPair && d82.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d81 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d82}, 2)
					ctx.FreeDesc(&d80)
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d81)
					if d81.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d81.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d81.Imm)
						ptrWord, _ := d81.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d81.Imm.String())))
						d81 = tmpPair
					} else if d81.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d81.Type, Reg: ctx.AllocRegExcept(d81.Reg), Reg2: ctx.AllocRegExcept(d81.Reg)}
						switch d81.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d81)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d81)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d81)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d81)
						d81 = tmpPair
					}
					if d81.Loc != LocRegPair && d81.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d81)
					d83 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d81}, 2)
					ctx.BindReg(d83.Reg, &d83)
					ctx.BindReg(d83.Reg2, &d83)
					ctx.StabilizeDescForControlFlow(&d83)
					if ps.General {
						ctx.SyncDesc(&d83)
						if d83.Loc == LocReg {
							ctx.ProtectReg(d83.Reg)
						} else if d83.Loc == LocRegPair {
							ctx.ProtectReg(d83.Reg)
							ctx.ProtectReg(d83.Reg2)
						}
						d84 = d83
						if d84.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d84)
						if d84.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d84, int32(bbs[5].PhiBase)+int32(0), 2)
						} else if d84.Loc == LocInputPair {
							ctx.EnsureDesc(&d84)
							ctx.EmitStoreScmerToStack(d84, int32(bbs[5].PhiBase)+int32(0))
						} else if d84.Loc == LocRegPair || d84.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d84, int32(bbs[5].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d84)
							ctx.EmitStoreToStack(d84, int32(bbs[5].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[5].PhiBase)+int32(0))+8)
						}
						if d83.Loc == LocReg {
							ctx.UnprotectReg(d83.Reg)
						} else if d83.Loc == LocRegPair {
							ctx.UnprotectReg(d83.Reg)
							ctx.UnprotectReg(d83.Reg2)
						}
					}
					ps85 := PhiState{General: ps.General}
					ps85.OverlayValues = make([]JITValueDesc, 85)
					ps85.OverlayValues[1] = d1
					ps85.OverlayValues[2] = d2
					ps85.OverlayValues[3] = d3
					ps85.OverlayValues[4] = d4
					ps85.OverlayValues[5] = d5
					ps85.OverlayValues[16] = d16
					ps85.OverlayValues[17] = d17
					ps85.OverlayValues[18] = d18
					ps85.OverlayValues[19] = d19
					ps85.OverlayValues[20] = d20
					ps85.OverlayValues[21] = d21
					ps85.OverlayValues[22] = d22
					ps85.OverlayValues[23] = d23
					ps85.OverlayValues[24] = d24
					ps85.OverlayValues[25] = d25
					ps85.OverlayValues[28] = d28
					ps85.OverlayValues[31] = d31
					ps85.OverlayValues[50] = d50
					ps85.OverlayValues[51] = d51
					ps85.OverlayValues[52] = d52
					ps85.OverlayValues[53] = d53
					ps85.OverlayValues[80] = d80
					ps85.OverlayValues[81] = d81
					ps85.OverlayValues[82] = d82
					ps85.OverlayValues[83] = d83
					ps85.OverlayValues[84] = d84
					ps85.PhiValues = make([]JITValueDesc, 1)
					d86 = d83
					ps85.PhiValues[0] = d86
					if ps85.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps85)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d87 := ps.PhiValues[0]
							ctx.EnsureDesc(&d87)
							ctx.EmitStoreScmerToStack(d87, int32(bbs[5].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d1)
					d88 = d18
					_ = d88
					ctx.StabilizeDescForControlFlow(&d88)
					d89 = d21
					_ = d89
					ctx.StabilizeDescForControlFlow(&d89)
					d90 = d1
					_ = d90
					ctx.StabilizeDescForControlFlow(&d90)
					lbl13 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					bbpos_1_4 := int32(-1)
					_ = bbpos_1_4
					bbpos_1_5 := int32(-1)
					_ = bbpos_1_5
					bbpos_1_6 := int32(-1)
					_ = bbpos_1_6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d90)
					if d90.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d90.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d90.Imm)
						ptrWord, _ := d90.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d90.Imm.String())))
						d90 = tmpPair
					} else if d90.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d90.Type, Reg: ctx.AllocRegExcept(d90.Reg), Reg2: ctx.AllocRegExcept(d90.Reg)}
						switch d90.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d90)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d90)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d90)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d90)
						d90 = tmpPair
					}
					if d90.Loc != LocRegPair && d90.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d90)
					d91 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d90}, 2)
					ctx.BindReg(d91.Reg, &d91)
					ctx.BindReg(d91.Reg2, &d91)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d91)
					d92 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("_ci")}
					d93 = d91
					_ = d93
					ctx.StabilizeDescForControlFlow(&d93)
					d94 = d92
					_ = d94
					ctx.StabilizeDescForControlFlow(&d94)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d93)
					ctx.EnsureDesc(&d93)
					ctx.EnsureDesc(&d93)
					if d93.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d93.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d93.Imm)
						ptrWord, _ := d93.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d93.Imm.String())))
						d93 = tmpPair
					} else if d93.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d93.Type, Reg: ctx.AllocRegExcept(d93.Reg), Reg2: ctx.AllocRegExcept(d93.Reg)}
						switch d93.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d93)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d93)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d93)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d93)
						d93 = tmpPair
					}
					if d93.Loc != LocRegPair && d93.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.Index arg0)")
					}
					ctx.EnsureDesc(&d94)
					ctx.EnsureDesc(&d94)
					ctx.EnsureDesc(&d94)
					if d94.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d94.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d94.Imm)
						ptrWord, _ := d94.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d94.Imm.String())))
						d94 = tmpPair
					} else if d94.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d94.Type, Reg: ctx.AllocRegExcept(d94.Reg), Reg2: ctx.AllocRegExcept(d94.Reg)}
						switch d94.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d94)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d94)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d94)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d94)
						d94 = tmpPair
					}
					if d94.Loc != LocRegPair && d94.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.Index arg1)")
					}
					ctx.SyncDesc(&d93)
					ctx.SyncDesc(&d94)
					d95 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Index), []JITValueDesc{d93, d94}, 1)
					ctx.BindReg(d95.Reg, &d95)
					ctx.FreeDesc(&d94)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d95)
					var d96 JITValueDesc
					if d95.Loc == LocImm {
						d96 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d95.Imm.Int() >= 0)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d95.Reg, 0)
						ctx.EmitSetcc(r1, CondSignedGreaterOrEqual)
						d96 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d96)
					}
					ctx.FreeDesc(&d95)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d96)
					ctx.ReclaimUntrackedRegs()
					d97 = d96
					ctx.EnsureDesc(&d97)
					if d97.Loc != LocImm && d97.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d97.Loc == LocImm {
						if d97.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl15)
						}
					} else {
						ctx.EmitCmpRegImm32(d97.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
					ctx.FreeDesc(&d96)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d88.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d88.Imm)
						ptrWord, _ := d88.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d88.Imm.String())))
						d88 = tmpPair
					} else if d88.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d88.Type, Reg: ctx.AllocRegExcept(d88.Reg), Reg2: ctx.AllocRegExcept(d88.Reg)}
						switch d88.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d88)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d88)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d88)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d88)
						d88 = tmpPair
					}
					if d88.Loc != LocRegPair && d88.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					if d89.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d89.Imm)
						ptrWord, _ := d89.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d89.Imm.String())))
						d89 = tmpPair
					} else if d89.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocRegExcept(d89.Reg), Reg2: ctx.AllocRegExcept(d89.Reg)}
						switch d89.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d89)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d89)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d89)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d89)
						d89 = tmpPair
					}
					if d89.Loc != LocRegPair && d89.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d88)
					ctx.SyncDesc(&d89)
					d98 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d88, d89}, 1)
					ctx.EmitAndRegImm32(d98.Reg, 1)
					d98.Type = tagBool
					ctx.BindReg(d98.Reg, &d98)
					ctx.ReclaimUntrackedRegs()
					r2 := ctx.AllocReg()
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d98)
					if d98.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d98)
					}
					ctx.EmitJmp(lbl13)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					if d89.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d89.Imm)
						ptrWord, _ := d89.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d89.Imm.String())))
						d89 = tmpPair
					} else if d89.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocRegExcept(d89.Reg), Reg2: ctx.AllocRegExcept(d89.Reg)}
						switch d89.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d89)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d89)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d89)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d89)
						d89 = tmpPair
					}
					if d89.Loc != LocRegPair && d89.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (likePatternNeedsCaseFold arg0)")
					}
					ctx.SyncDesc(&d89)
					d99 = ctx.EmitGoCallScalar(GoFuncAddr(likePatternNeedsCaseFold), []JITValueDesc{d89}, 1)
					ctx.EmitAndRegImm32(d99.Reg, 1)
					d99.Type = tagBool
					ctx.BindReg(d99.Reg, &d99)
					ctx.ReclaimUntrackedRegs()
					d100 = d99
					ctx.EnsureDesc(&d100)
					if d100.Loc != LocImm && d100.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					if d100.Loc == LocImm {
						if d100.Imm.Bool() {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						} else {
							ctx.MarkLabel(lbl21)
							ctx.EmitJmp(lbl19)
						}
					} else {
						ctx.EmitCmpRegImm32(d100.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl20)
						ctx.EmitJmp(lbl21)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
					}
					ctx.FreeDesc(&d99)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d88.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d88.Imm)
						ptrWord, _ := d88.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d88.Imm.String())))
						d88 = tmpPair
					} else if d88.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d88.Type, Reg: ctx.AllocRegExcept(d88.Reg), Reg2: ctx.AllocRegExcept(d88.Reg)}
						switch d88.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d88)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d88)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d88)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d88)
						d88 = tmpPair
					}
					if d88.Loc != LocRegPair && d88.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					if d89.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d89.Imm)
						ptrWord, _ := d89.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d89.Imm.String())))
						d89 = tmpPair
					} else if d89.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocRegExcept(d89.Reg), Reg2: ctx.AllocRegExcept(d89.Reg)}
						switch d89.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d89)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d89)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d89)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d89)
						d89 = tmpPair
					}
					if d89.Loc != LocRegPair && d89.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d88)
					ctx.SyncDesc(&d89)
					d101 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d88, d89}, 1)
					ctx.EmitAndRegImm32(d101.Reg, 1)
					d101.Type = tagBool
					ctx.BindReg(d101.Reg, &d101)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d101)
					ctx.EnsureDesc(&d101)
					if d101.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d101)
					}
					ctx.EmitJmp(lbl13)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d88.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d88.Imm)
						ptrWord, _ := d88.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d88.Imm.String())))
						d88 = tmpPair
					} else if d88.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d88.Type, Reg: ctx.AllocRegExcept(d88.Reg), Reg2: ctx.AllocRegExcept(d88.Reg)}
						switch d88.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d88)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d88)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d88)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d88)
						d88 = tmpPair
					}
					if d88.Loc != LocRegPair && d88.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strLikeASCIIFold arg0)")
					}
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					if d89.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d89.Imm)
						ptrWord, _ := d89.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d89.Imm.String())))
						d89 = tmpPair
					} else if d89.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d89.Type, Reg: ctx.AllocRegExcept(d89.Reg), Reg2: ctx.AllocRegExcept(d89.Reg)}
						switch d89.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d89)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d89)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d89)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d89)
						d89 = tmpPair
					}
					if d89.Loc != LocRegPair && d89.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strLikeASCIIFold arg1)")
					}
					ctx.SyncDesc(&d88)
					ctx.SyncDesc(&d89)
					callResults102 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeASCIIFold), []JITValueDesc{d88, d89}, []uint8{1, 1}, []uint8{0, 0})
					d103 = callResults102[0]
					_ = d103
					d104 = callResults102[1]
					_ = d104
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d103)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d105 = d104
					ctx.EnsureDesc(&d105)
					if d105.Loc != LocImm && d105.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d105.Loc == LocImm {
						if d105.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d105.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d104)
					bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d89)
					d106 = d88
					_ = d106
					ctx.StabilizeDescForControlFlow(&d106)
					d107 = d89
					_ = d107
					ctx.StabilizeDescForControlFlow(&d107)
					r3 := d88.Loc == LocReg || d88.Loc == LocRegPair || d88.Loc == LocRegTriple
					r4 := d88.Reg
					if r3 {
						ctx.ProtectReg(r4)
					}
					r5 := d88.Loc == LocRegPair || d88.Loc == LocRegTriple
					r6 := d88.Reg2
					if r5 {
						ctx.ProtectReg(r6)
					}
					r7 := d88.Loc == LocRegTriple
					r8 := d88.Reg3
					if r7 {
						ctx.ProtectReg(r8)
					}
					r9 := d89.Loc == LocReg || d89.Loc == LocRegPair || d89.Loc == LocRegTriple
					r10 := d89.Reg
					if r9 {
						ctx.ProtectReg(r10)
					}
					r11 := d89.Loc == LocRegPair || d89.Loc == LocRegTriple
					r12 := d89.Reg2
					if r11 {
						ctx.ProtectReg(r12)
					}
					r13 := d89.Loc == LocRegTriple
					r14 := d89.Reg3
					if r13 {
						ctx.ProtectReg(r14)
					}
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d106)
					if d106.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d106.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d106.Imm)
						ptrWord, _ := d106.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d106.Imm.String())))
						d106 = tmpPair
					} else if d106.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d106.Type, Reg: ctx.AllocRegExcept(d106.Reg), Reg2: ctx.AllocRegExcept(d106.Reg)}
						switch d106.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d106)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d106)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d106)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d106)
						d106 = tmpPair
					}
					if d106.Loc != LocRegPair && d106.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d106)
					d108 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d106}, 2)
					ctx.BindReg(d108.Reg, &d108)
					ctx.BindReg(d108.Reg2, &d108)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d107)
					ctx.EnsureDesc(&d107)
					ctx.EnsureDesc(&d107)
					if d107.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d107.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d107.Imm)
						ptrWord, _ := d107.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d107.Imm.String())))
						d107 = tmpPair
					} else if d107.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d107.Type, Reg: ctx.AllocRegExcept(d107.Reg), Reg2: ctx.AllocRegExcept(d107.Reg)}
						switch d107.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d107)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d107)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d107)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d107)
						d107 = tmpPair
					}
					if d107.Loc != LocRegPair && d107.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d107)
					d109 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d107}, 2)
					ctx.BindReg(d109.Reg, &d109)
					ctx.BindReg(d109.Reg2, &d109)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d108)
					ctx.EnsureDesc(&d108)
					ctx.EnsureDesc(&d108)
					if d108.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d108.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d108.Imm)
						ptrWord, _ := d108.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d108.Imm.String())))
						d108 = tmpPair
					} else if d108.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d108.Type, Reg: ctx.AllocRegExcept(d108.Reg), Reg2: ctx.AllocRegExcept(d108.Reg)}
						switch d108.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d108)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d108)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d108)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d108)
						d108 = tmpPair
					}
					if d108.Loc != LocRegPair && d108.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
					ctx.EnsureDesc(&d109)
					ctx.EnsureDesc(&d109)
					ctx.EnsureDesc(&d109)
					if d109.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d109.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d109.Imm)
						ptrWord, _ := d109.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d109.Imm.String())))
						d109 = tmpPair
					} else if d109.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d109.Type, Reg: ctx.AllocRegExcept(d109.Reg), Reg2: ctx.AllocRegExcept(d109.Reg)}
						switch d109.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d109)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d109)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d109)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d109)
						d109 = tmpPair
					}
					if d109.Loc != LocRegPair && d109.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d108)
					ctx.SyncDesc(&d109)
					d110 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d108, d109}, 1)
					ctx.EmitAndRegImm32(d110.Reg, 1)
					d110.Type = tagBool
					ctx.BindReg(d110.Reg, &d110)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					if r3 {
						ctx.UnprotectReg(r4)
					}
					if r5 {
						ctx.UnprotectReg(r6)
					}
					if r7 {
						ctx.UnprotectReg(r8)
					}
					if r9 {
						ctx.UnprotectReg(r10)
					}
					if r11 {
						ctx.UnprotectReg(r12)
					}
					if r13 {
						ctx.UnprotectReg(r14)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EnsureDesc(&d110)
					if d110.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d110)
					}
					ctx.EmitJmp(lbl13)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d103)
					}
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl13)
					d111 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d111)
					ctx.BindReg(r2, &d111)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d111)
					if d111.Loc == LocImm {
						ctx.EmitMakeBool(result, d111)
					} else {
						ctx.EmitMovToReg(result.Reg2, d111)
						d112 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d112)
						if d111.Loc == LocReg && d111.Reg != result.Reg2 {
							ctx.FreeReg(d111.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps113 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps113)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITInlineCost: 51,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strlike_cs"].Fn, args, result)
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
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
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
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d16.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d16.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					} else if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
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
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
					d17 = args[1]
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
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
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
					if d18.Loc != LocRegPair && d18.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d18)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d15, d18}, 1)
					ctx.EmitAndRegImm32(d20.Reg, 1)
					d20.Type = tagBool
					ctx.BindReg(d20.Reg, &d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocImm {
						ctx.EmitMakeBool(result, d20)
					} else {
						ctx.EmitMovToReg(result.Reg2, d20)
						d21 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d21)
						if d20.Loc == LocReg && d20.Reg != result.Reg2 {
							ctx.FreeReg(d20.Reg)
						}
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					ctx.ReclaimUntrackedRegs()
					d22 = args[1]
					d22.ID = 0
					d24 = d22
					d24.ID = 0
					d23 = ctx.EmitTagEqualsBorrowed(&d24, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d22)
					d25 = d23
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							if ps.General {
							}
							ps26 := PhiState{General: ps.General}
							ps26.OverlayValues = make([]JITValueDesc, 26)
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
							ps26.OverlayValues[24] = d24
							ps26.OverlayValues[25] = d25
							return bbs[1].RenderPS(ps26)
						}
						if ps.General {
						}
						ps27 := PhiState{General: ps.General}
						ps27.OverlayValues = make([]JITValueDesc, 26)
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
						ps27.OverlayValues[24] = d24
						ps27.OverlayValues[25] = d25
						return bbs[2].RenderPS(ps27)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl3)
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 26)
					ps28.OverlayValues[0] = d0
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[13] = d13
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[15] = d15
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[18] = d18
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[24] = d24
					ps28.OverlayValues[25] = d25
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 26)
					ps29.OverlayValues[0] = d0
					ps29.OverlayValues[1] = d1
					ps29.OverlayValues[2] = d2
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[13] = d13
					ps29.OverlayValues[14] = d14
					ps29.OverlayValues[15] = d15
					ps29.OverlayValues[16] = d16
					ps29.OverlayValues[17] = d17
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[20] = d20
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[25] = d25
					snap30 := d0
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d13
					snap35 := d14
					snap36 := d15
					snap37 := d16
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d22
					snap44 := d23
					snap45 := d24
					snap46 := d25
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps29)
					}
					ctx.RestoreAllocState(alloc47)
					d0 = snap30
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d13 = snap34
					d14 = snap35
					d15 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d23 = snap44
					d24 = snap45
					d25 = snap46
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps28)
					}
					return result
					ctx.FreeDesc(&d23)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps48 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps48)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 19,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["toLower"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["toUpper"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["replace"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
				d3 := args[1]
				d3.ID = 0
				d5 := d3
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
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d5}, 2)
				ctx.FreeDesc(&d3)
				d6 := args[2]
				d6.ID = 0
				d8 := d6
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
				d7 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d8}, 2)
				ctx.FreeDesc(&d6)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d7)
				d9 := d1
				_ = d9
				ctx.StabilizeDescForControlFlow(&d9)
				d10 := d4
				_ = d10
				ctx.StabilizeDescForControlFlow(&d10)
				d11 := d7
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d11)
				d12 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
				d13 := d9
				_ = d13
				ctx.StabilizeDescForControlFlow(&d13)
				d14 := d10
				_ = d14
				ctx.StabilizeDescForControlFlow(&d14)
				d15 := d11
				_ = d15
				ctx.StabilizeDescForControlFlow(&d15)
				d16 := d12
				_ = d16
				ctx.StabilizeDescForControlFlow(&d16)
				r0 := d9.Loc == LocReg || d9.Loc == LocRegPair || d9.Loc == LocRegTriple
				r1 := d9.Reg
				if r0 {
					ctx.ProtectReg(r1)
				}
				r2 := d9.Loc == LocRegPair || d9.Loc == LocRegTriple
				r3 := d9.Reg2
				if r2 {
					ctx.ProtectReg(r3)
				}
				r4 := d9.Loc == LocRegTriple
				r5 := d9.Reg3
				if r4 {
					ctx.ProtectReg(r5)
				}
				r6 := d10.Loc == LocReg || d10.Loc == LocRegPair || d10.Loc == LocRegTriple
				r7 := d10.Reg
				if r6 {
					ctx.ProtectReg(r7)
				}
				r8 := d10.Loc == LocRegPair || d10.Loc == LocRegTriple
				r9 := d10.Reg2
				if r8 {
					ctx.ProtectReg(r9)
				}
				r10 := d10.Loc == LocRegTriple
				r11 := d10.Reg3
				if r10 {
					ctx.ProtectReg(r11)
				}
				r12 := d11.Loc == LocReg || d11.Loc == LocRegPair || d11.Loc == LocRegTriple
				r13 := d11.Reg
				if r12 {
					ctx.ProtectReg(r13)
				}
				r14 := d11.Loc == LocRegPair || d11.Loc == LocRegTriple
				r15 := d11.Reg2
				if r14 {
					ctx.ProtectReg(r15)
				}
				r16 := d11.Loc == LocRegTriple
				r17 := d11.Reg3
				if r16 {
					ctx.ProtectReg(r17)
				}
				phiBase17 := ctx.AllocStack(int32(64))
				d18 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				_ = d18
				d19 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				_ = d19
				d20 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				_ = d20
				d21 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				_ = d21
				lbl0 := ctx.ReserveLabel()
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_1 := int32(-1)
				_ = bbpos_2_1
				bbpos_2_2 := int32(-1)
				_ = bbpos_2_2
				bbpos_2_3 := int32(-1)
				_ = bbpos_2_3
				bbpos_2_4 := int32(-1)
				_ = bbpos_2_4
				bbpos_2_5 := int32(-1)
				_ = bbpos_2_5
				bbpos_2_6 := int32(-1)
				_ = bbpos_2_6
				bbpos_2_7 := int32(-1)
				_ = bbpos_2_7
				bbpos_2_8 := int32(-1)
				_ = bbpos_2_8
				bbpos_2_9 := int32(-1)
				_ = bbpos_2_9
				bbpos_2_10 := int32(-1)
				_ = bbpos_2_10
				bbpos_2_11 := int32(-1)
				_ = bbpos_2_11
				bbpos_2_12 := int32(-1)
				_ = bbpos_2_12
				bbpos_2_13 := int32(-1)
				_ = bbpos_2_13
				bbpos_2_14 := int32(-1)
				_ = bbpos_2_14
				bbpos_2_15 := int32(-1)
				_ = bbpos_2_15
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d15)
				d22 := ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d14, d15}, 1)
				ctx.EmitAndRegImm32(d22.Reg, 1)
				d22.Type = tagBool
				ctx.BindReg(d22.Reg, &d22)
				ctx.ReclaimUntrackedRegs()
				d23 := d22
				ctx.EnsureDesc(&d23)
				if d23.Loc != LocImm && d23.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl1 := ctx.ReserveLabel()
				lbl2 := ctx.ReserveLabel()
				lbl3 := ctx.ReserveLabel()
				lbl4 := ctx.ReserveLabel()
				if d23.Loc == LocImm {
					if d23.Imm.Bool() {
						ctx.MarkLabel(lbl3)
						ctx.EmitJmp(lbl1)
					} else {
						ctx.MarkLabel(lbl4)
						ctx.EmitJmp(lbl2)
					}
				} else {
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl3)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl3)
					ctx.EmitJmp(lbl1)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
				}
				ctx.FreeDesc(&d22)
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d16)
				var d24 JITValueDesc
				if d16.Loc == LocImm {
					d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() == 0)}
				} else {
					r18 := ctx.AllocRegExcept(d16.Reg)
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitSetcc(r18, CondEqual)
					d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
					ctx.BindReg(r18, &d24)
				}
				ctx.ReclaimUntrackedRegs()
				d25 := d24
				ctx.EnsureDesc(&d25)
				if d25.Loc != LocImm && d25.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl5 := ctx.ReserveLabel()
				lbl6 := ctx.ReserveLabel()
				lbl7 := ctx.ReserveLabel()
				if d25.Loc == LocImm {
					if d25.Imm.Bool() {
						ctx.MarkLabel(lbl6)
						ctx.EmitJmp(lbl1)
					} else {
						ctx.MarkLabel(lbl7)
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl1)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl5)
				}
				ctx.FreeDesc(&d24)
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d13)
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
				if d13.Loc != LocRegPair && d13.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.Count arg0)")
				}
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
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
				if d14.Loc != LocRegPair && d14.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.Count arg1)")
				}
				ctx.SyncDesc(&d13)
				ctx.SyncDesc(&d14)
				d26 := ctx.EmitGoCallScalar(GoFuncAddr(strings.Count), []JITValueDesc{d13, d14}, 1)
				ctx.BindReg(d26.Reg, &d26)
				ctx.StabilizeDescForControlFlow(&d26)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d26)
				var d27 JITValueDesc
				if d26.Loc == LocImm {
					d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d26.Imm.Int() == 0)}
				} else {
					r19 := ctx.AllocRegExcept(d26.Reg)
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					ctx.EmitSetcc(r19, CondEqual)
					d27 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
					ctx.BindReg(r19, &d27)
				}
				ctx.ReclaimUntrackedRegs()
				d28 := d27
				ctx.EnsureDesc(&d28)
				if d28.Loc != LocImm && d28.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl8 := ctx.ReserveLabel()
				lbl9 := ctx.ReserveLabel()
				lbl10 := ctx.ReserveLabel()
				lbl11 := ctx.ReserveLabel()
				if d28.Loc == LocImm {
					if d28.Imm.Bool() {
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					} else {
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl9)
					}
				} else {
					ctx.EmitCmpRegImm32(d28.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl9)
				}
				ctx.FreeDesc(&d27)
				bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d16)
				var d29 JITValueDesc
				if d16.Loc == LocImm {
					d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < 0)}
				} else {
					r20 := ctx.AllocRegExcept(d16.Reg)
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitSetcc(r20, CondSignedLess)
					d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
					ctx.BindReg(r20, &d29)
				}
				ctx.ReclaimUntrackedRegs()
				d30 := d29
				ctx.EnsureDesc(&d30)
				if d30.Loc != LocImm && d30.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl12 := ctx.ReserveLabel()
				lbl13 := ctx.ReserveLabel()
				lbl14 := ctx.ReserveLabel()
				lbl15 := ctx.ReserveLabel()
				if d30.Loc == LocImm {
					if d30.Imm.Bool() {
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
				} else {
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl13)
				}
				ctx.FreeDesc(&d29)
				bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d26)
				ctx.EnsureDesc(&d16)
				ctx.EnsureDesc(&d26)
				ctx.EnsureDesc(&d16)
				ctx.EnsureDesc(&d26)
				ctx.EnsureDesc(&d16)
				var d31 JITValueDesc
				if d26.Loc == LocImm && d16.Loc == LocImm {
					d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d26.Imm.Int() < d16.Imm.Int())}
				} else if d16.Loc == LocImm {
					r21 := ctx.AllocRegExcept(d26.Reg)
					if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d26.Reg, int32(d16.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
						ctx.EmitCmpInt64(d26.Reg, RegR11)
					}
					ctx.EmitSetcc(r21, CondSignedLess)
					d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
					ctx.BindReg(r21, &d31)
				} else if d26.Loc == LocImm {
					r22 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d26.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d16.Reg)
					ctx.EmitSetcc(r22, CondSignedLess)
					d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
					ctx.BindReg(r22, &d31)
				} else {
					r23 := ctx.AllocRegExcept(d26.Reg)
					ctx.EmitCmpInt64(d26.Reg, d16.Reg)
					ctx.EmitSetcc(r23, CondSignedLess)
					d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
					ctx.BindReg(r23, &d31)
				}
				ctx.ReclaimUntrackedRegs()
				d32 := d31
				ctx.EnsureDesc(&d32)
				if d32.Loc != LocImm && d32.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl16 := ctx.ReserveLabel()
				lbl17 := ctx.ReserveLabel()
				lbl18 := ctx.ReserveLabel()
				if d32.Loc == LocImm {
					if d32.Imm.Bool() {
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl18)
						ctx.SyncDesc(&d16)
						if d16.Loc == LocReg {
							ctx.ProtectReg(d16.Reg)
						} else if d16.Loc == LocRegPair {
							ctx.ProtectReg(d16.Reg)
							ctx.ProtectReg(d16.Reg2)
						}
						d33 := d16
						if d33.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d33)
						ctx.EmitStoreToStack(d33, int32(phiBase17)+int32(0))
						if d16.Loc == LocReg {
							ctx.UnprotectReg(d16.Reg)
						} else if d16.Loc == LocRegPair {
							ctx.UnprotectReg(d16.Reg)
							ctx.UnprotectReg(d16.Reg2)
						}
						ctx.EmitJmp(lbl16)
					}
				} else {
					ctx.EmitCmpRegImm32(d32.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl18)
					ctx.SyncDesc(&d16)
					if d16.Loc == LocReg {
						ctx.ProtectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.ProtectReg(d16.Reg)
						ctx.ProtectReg(d16.Reg2)
					}
					d34 := d16
					if d34.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d34)
					ctx.EmitStoreToStack(d34, int32(phiBase17)+int32(0))
					if d16.Loc == LocReg {
						ctx.UnprotectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.UnprotectReg(d16.Reg)
						ctx.UnprotectReg(d16.Reg2)
					}
					ctx.EmitJmp(lbl16)
				}
				ctx.FreeDesc(&d31)
				bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d35 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase17) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d35)
				ctx.ReclaimUntrackedRegs()
				d36 := ctx.EmitGoCallScalar(GoFuncAddr(func() *strings.Builder { return new(strings.Builder) }), nil, 1)
				ctx.BindReg(d36.Reg, &d36)
				ctx.StabilizeDescForControlFlow(&d36)
				ctx.ReclaimUntrackedRegs()
				var d37 JITValueDesc
				if d13.SliceSizeKnown {
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d13.KnownSliceLen))}
				} else if d13.Loc == LocImm {
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d13.Imm.String())))}
				} else if d13.Loc == LocStackTriple {
					d37 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d13.StackOff + 8, NoHeapPointer: true}
				} else if d13.Loc == LocStackPair {
					d37 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d13.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocRegTriple {
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg2, ID: 0}
					} else if d13.Loc == LocReg {
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				var d38 JITValueDesc
				if d15.SliceSizeKnown {
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d15.KnownSliceLen))}
				} else if d15.Loc == LocImm {
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d15.Imm.String())))}
				} else if d15.Loc == LocStackTriple {
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d15.StackOff + 8, NoHeapPointer: true}
				} else if d15.Loc == LocStackPair {
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d15.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocRegTriple {
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg2, ID: 0}
					} else if d15.Loc == LocReg {
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				var d39 JITValueDesc
				if d14.SliceSizeKnown {
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.KnownSliceLen))}
				} else if d14.Loc == LocImm {
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d14.Imm.String())))}
				} else if d14.Loc == LocStackTriple {
					d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
				} else if d14.Loc == LocStackPair {
					d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocRegTriple {
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg2, ID: 0}
					} else if d14.Loc == LocReg {
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d38)
				ctx.EnsureDesc(&d39)
				ctx.EnsureDesc(&d38)
				ctx.ProtectReg(d38.Reg)
				ctx.EnsureDesc(&d39)
				ctx.UnprotectReg(d38.Reg)
				var d40 JITValueDesc
				if d38.Loc == LocImm && d39.Loc == LocImm {
					d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() - d39.Imm.Int())}
				} else if d39.Loc == LocImm && d39.Imm.Int() == 0 {
					r24 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegReg(r24, d38.Reg)
					d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
					ctx.BindReg(r24, &d40)
				} else if d38.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d39.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
					ctx.EmitSubInt64(scratch, d39.Reg)
					d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d40)
				} else if d39.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegReg(scratch, d38.Reg)
					if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
						ctx.EmitSubRegImm32(scratch, int32(d39.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()))
						ctx.EmitSubInt64(scratch, RegR11)
					}
					d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d40)
				} else {
					r25 := ctx.AllocRegExcept(d38.Reg, d39.Reg)
					ctx.EmitMovRegReg(r25, d38.Reg)
					ctx.EmitSubInt64(r25, d39.Reg)
					d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
					ctx.BindReg(r25, &d40)
				}
				if d40.Loc == LocReg && d38.Loc == LocReg && d40.Reg == d38.Reg {
					ctx.TransferReg(d38.Reg)
					d38.Loc = LocNone
				}
				ctx.FreeDesc(&d38)
				ctx.FreeDesc(&d39)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d35)
				ctx.EnsureDesc(&d40)
				ctx.EnsureDesc(&d35)
				ctx.ProtectReg(d35.Reg)
				ctx.EnsureDesc(&d40)
				ctx.UnprotectReg(d35.Reg)
				var d41 JITValueDesc
				if d35.Loc == LocImm && d40.Loc == LocImm {
					d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d35.Imm.Int() * d40.Imm.Int())}
				} else if d35.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d40.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
					ctx.EmitImulInt64(scratch, d40.Reg)
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d41)
				} else if d40.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d35.Reg)
					ctx.EmitMovRegReg(scratch, d35.Reg)
					if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
						ctx.EmitImulRegImm32(scratch, int32(d40.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d40.Imm.Int()))
						ctx.EmitImulInt64(scratch, RegR11)
					}
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d41)
				} else {
					r26 := ctx.AllocRegExcept(d35.Reg, d40.Reg)
					ctx.EmitMovRegReg(r26, d35.Reg)
					ctx.EmitImulInt64(r26, d40.Reg)
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
					ctx.BindReg(r26, &d41)
				}
				if d41.Loc == LocReg && d35.Loc == LocReg && d41.Reg == d35.Reg {
					ctx.TransferReg(d35.Reg)
					d35.Loc = LocNone
				}
				ctx.FreeDesc(&d40)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d37)
				ctx.EnsureDesc(&d41)
				ctx.EnsureDesc(&d37)
				ctx.ProtectReg(d37.Reg)
				ctx.EnsureDesc(&d41)
				ctx.UnprotectReg(d37.Reg)
				var d42 JITValueDesc
				if d37.Loc == LocImm && d41.Loc == LocImm {
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() + d41.Imm.Int())}
				} else if d41.Loc == LocImm && d41.Imm.Int() == 0 {
					r27 := ctx.AllocRegExcept(d37.Reg)
					ctx.EmitMovRegReg(r27, d37.Reg)
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
					ctx.BindReg(r27, &d42)
				} else if d37.Loc == LocImm && d37.Imm.Int() == 0 {
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg}
					ctx.BindReg(d41.Reg, &d42)
				} else if d37.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d41.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
					ctx.EmitAddInt64(scratch, d41.Reg)
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d42)
				} else if d41.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d37.Reg)
					ctx.EmitMovRegReg(scratch, d37.Reg)
					if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(scratch, int32(d41.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d41.Imm.Int()))
						ctx.EmitAddInt64(scratch, RegR11)
					}
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d42)
				} else {
					r28 := ctx.AllocRegExcept(d37.Reg, d41.Reg)
					ctx.EmitMovRegReg(r28, d37.Reg)
					ctx.EmitAddInt64(r28, d41.Reg)
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r28}
					ctx.BindReg(r28, &d42)
				}
				if d42.Loc == LocReg && d37.Loc == LocReg && d42.Reg == d37.Reg {
					ctx.TransferReg(d37.Reg)
					d37.Loc = LocNone
				}
				ctx.FreeDesc(&d37)
				ctx.FreeDesc(&d41)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d36)
				ctx.EnsureDesc(&d36)
				if d36.Loc == LocRegPair || d36.Loc == LocStackPair || d36.Loc == LocRegTriple || d36.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d42)
				ctx.EnsureDesc(&d42)
				if d42.Loc == LocRegPair || d42.Loc == LocStackPair || d42.Loc == LocRegTriple || d42.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d42)
				ctx.EmitGoCallVoid(GoFuncAddr((*strings.Builder).Grow), []JITValueDesc{d36, d42})
				ctx.FreeDesc(&d42)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase17)+int32(16))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase17)+int32(32))
				bbpos_2_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d43 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase17) + int32(16)}
				ctx.StabilizeDescForControlFlow(&d43)
				ctx.ReclaimUntrackedRegs()
				d44 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase17) + int32(32)}
				ctx.StabilizeDescForControlFlow(&d44)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d35)
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d35)
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d35)
				var d45 JITValueDesc
				if d44.Loc == LocImm && d35.Loc == LocImm {
					d45 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d44.Imm.Int() < d35.Imm.Int())}
				} else if d35.Loc == LocImm {
					r29 := ctx.AllocRegExcept(d44.Reg)
					if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d44.Reg, int32(d35.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d35.Imm.Int()))
						ctx.EmitCmpInt64(d44.Reg, RegR11)
					}
					ctx.EmitSetcc(r29, CondSignedLess)
					d45 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r29}
					ctx.BindReg(r29, &d45)
				} else if d44.Loc == LocImm {
					r30 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d44.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d35.Reg)
					ctx.EmitSetcc(r30, CondSignedLess)
					d45 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r30}
					ctx.BindReg(r30, &d45)
				} else {
					r31 := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitCmpInt64(d44.Reg, d35.Reg)
					ctx.EmitSetcc(r31, CondSignedLess)
					d45 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r31}
					ctx.BindReg(r31, &d45)
				}
				ctx.FreeDesc(&d35)
				ctx.ReclaimUntrackedRegs()
				d46 := d45
				ctx.EnsureDesc(&d46)
				if d46.Loc != LocImm && d46.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl19 := ctx.ReserveLabel()
				lbl20 := ctx.ReserveLabel()
				lbl21 := ctx.ReserveLabel()
				lbl22 := ctx.ReserveLabel()
				if d46.Loc == LocImm {
					if d46.Imm.Bool() {
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
					} else {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl20)
					}
				} else {
					ctx.EmitCmpRegImm32(d46.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl20)
				}
				ctx.FreeDesc(&d45)
				bbpos_2_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl20)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				var d47 JITValueDesc
				ctx.EnsureDesc(&d13)
				if d13.Loc == LocRegPair || d13.Loc == LocRegTriple {
					d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg2}
					ctx.BindReg(d13.Reg2, &d47)
				} else {
					panic("Slice with omitted high requires descriptor with length in Reg2")
				}
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d47)
				var d49 JITValueDesc
				if d47.Loc == LocImm && d43.Loc == LocImm {
					d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() - d43.Imm.Int())}
				} else {
					r32 := ctx.AllocReg()
					if d47.Loc == LocImm {
						ctx.EmitMovRegImm64(r32, uint64(d47.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r32, d47.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
						ctx.EmitSubInt64(r32, RegR11)
					} else {
						ctx.EmitSubInt64(r32, d43.Reg)
					}
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r32}
					ctx.BindReg(r32, &d49)
				}
				var d50 JITValueDesc
				if d13.Loc == LocImm && d43.Loc == LocImm {
					d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d43.Imm.Int()*1)}
				} else {
					r33 := ctx.AllocReg()
					if d13.Loc == LocImm {
						ctx.EmitMovRegImm64(r33, uint64(d13.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r33, d13.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()*1))
						ctx.EmitAddInt64(r33, RegR11)
					} else {
						ctx.EmitAddInt64(r33, d43.Reg)
					}
					d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r33}
					ctx.BindReg(r33, &d50)
				}
				var d51 JITValueDesc
				var r34 Reg
				var r35 Reg
				ctx.SyncDesc(&d50)
				ctx.EnsureDesc(&d50)
				if d50.Loc == LocImm {
					r34 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r34, uint64(d50.Imm.Int()))
				} else {
					r34 = d50.Reg
				}
				ctx.ProtectReg(r34)
				ctx.SyncDesc(&d49)
				ctx.EnsureDesc(&d49)
				if d49.Loc == LocImm {
					r35 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r35, uint64(d49.Imm.Int()))
				} else {
					r35 = d49.Reg
				}
				ctx.ProtectReg(r35)
				ctx.UnprotectReg(r35)
				ctx.UnprotectReg(r34)
				d51 = JITValueDesc{Loc: LocRegPair, Reg: r34, Reg2: r35}
				ctx.BindReg(r34, &d51)
				ctx.BindReg(r35, &d51)
				ctx.BindReg(r34, &d51)
				ctx.BindReg(r35, &d51)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d36)
				ctx.EnsureDesc(&d36)
				if d36.Loc == LocRegPair || d36.Loc == LocStackPair || d36.Loc == LocRegTriple || d36.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d51)
				ctx.EnsureDesc(&d51)
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
				if d51.Loc != LocRegPair && d51.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value ((*strings.Builder).WriteString arg1)")
				}
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d51)
				callResults52 := JITEmitGoCallResults(ctx, GoFuncAddr((*strings.Builder).WriteString), []JITValueDesc{d36, d51}, []uint8{1, 2}, []uint8{0, 3})
				d53 := callResults52[0]
				_ = d53
				d54 := callResults52[1]
				_ = d54
				ctx.ReclaimUntrackedRegs()
				d56 := d36
				ctx.EnsureDesc(&d56)
				if d56.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d56.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d56)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d56)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d56)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d56.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d56 = tmpPair
				} else if d56.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d56.Reg), Reg2: ctx.AllocRegExcept(d56.Reg)}
					switch d56.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d56)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d56)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d56)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d56)
					d56 = tmpPair
				} else if d56.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d56.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d56.MemPtr))
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
					d56 = tmpPair
				}
				if d56.Loc != LocRegPair && d56.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d55 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d56}, 2)
				ctx.ReclaimUntrackedRegs()
				r36 := ctx.AllocReg()
				ctx.EnsureDesc(&d55)
				ctx.EnsureDesc(&d55)
				if d55.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r36, d55)
				}
				ctx.EmitJmp(lbl0)
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d13)
				if d13.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r36, d13)
				}
				ctx.EmitJmp(lbl0)
				bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d13)
				if d13.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r36, d13)
				}
				ctx.EmitJmp(lbl0)
				bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d26)
				if d26.Loc == LocReg {
					ctx.ProtectReg(d26.Reg)
				} else if d26.Loc == LocRegPair {
					ctx.ProtectReg(d26.Reg)
					ctx.ProtectReg(d26.Reg2)
				}
				d57 := d26
				if d57.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d57)
				ctx.EmitStoreToStack(d57, int32(phiBase17)+int32(0))
				if d26.Loc == LocReg {
					ctx.UnprotectReg(d26.Reg)
				} else if d26.Loc == LocRegPair {
					ctx.UnprotectReg(d26.Reg)
					ctx.UnprotectReg(d26.Reg2)
				}
				ctx.EmitJmp(lbl16)
				bbpos_2_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d58 JITValueDesc
				if d14.SliceSizeKnown {
					d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.KnownSliceLen))}
				} else if d14.Loc == LocImm {
					d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d14.Imm.String())))}
				} else if d14.Loc == LocStackTriple {
					d58 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
				} else if d14.Loc == LocStackPair {
					d58 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocRegTriple {
						d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg2, ID: 0}
					} else if d14.Loc == LocReg {
						d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d58)
				var d59 JITValueDesc
				if d58.Loc == LocImm {
					d59 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d58.Imm.Int() == 0)}
				} else {
					r37 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d58.Reg, 0)
					ctx.EmitSetcc(r37, CondEqual)
					d59 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r37}
					ctx.BindReg(r37, &d59)
				}
				ctx.FreeDesc(&d58)
				ctx.ReclaimUntrackedRegs()
				d60 := d59
				ctx.EnsureDesc(&d60)
				if d60.Loc != LocImm && d60.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl23 := ctx.ReserveLabel()
				lbl24 := ctx.ReserveLabel()
				lbl25 := ctx.ReserveLabel()
				lbl26 := ctx.ReserveLabel()
				if d60.Loc == LocImm {
					if d60.Imm.Bool() {
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					} else {
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl24)
					}
				} else {
					ctx.EmitCmpRegImm32(d60.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl25)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl24)
				}
				ctx.FreeDesc(&d59)
				bbpos_2_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl24)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				var d61 JITValueDesc
				ctx.EnsureDesc(&d13)
				if d13.Loc == LocRegPair || d13.Loc == LocRegTriple {
					d61 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg2}
					ctx.BindReg(d13.Reg2, &d61)
				} else {
					panic("Slice with omitted high requires descriptor with length in Reg2")
				}
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d61)
				var d63 JITValueDesc
				if d61.Loc == LocImm && d43.Loc == LocImm {
					d63 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d61.Imm.Int() - d43.Imm.Int())}
				} else {
					r38 := ctx.AllocReg()
					if d61.Loc == LocImm {
						ctx.EmitMovRegImm64(r38, uint64(d61.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r38, d61.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
						ctx.EmitSubInt64(r38, RegR11)
					} else {
						ctx.EmitSubInt64(r38, d43.Reg)
					}
					d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r38}
					ctx.BindReg(r38, &d63)
				}
				var d64 JITValueDesc
				if d13.Loc == LocImm && d43.Loc == LocImm {
					d64 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d43.Imm.Int()*1)}
				} else {
					r39 := ctx.AllocReg()
					if d13.Loc == LocImm {
						ctx.EmitMovRegImm64(r39, uint64(d13.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r39, d13.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()*1))
						ctx.EmitAddInt64(r39, RegR11)
					} else {
						ctx.EmitAddInt64(r39, d43.Reg)
					}
					d64 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r39}
					ctx.BindReg(r39, &d64)
				}
				var d65 JITValueDesc
				var r40 Reg
				var r41 Reg
				ctx.SyncDesc(&d64)
				ctx.EnsureDesc(&d64)
				if d64.Loc == LocImm {
					r40 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r40, uint64(d64.Imm.Int()))
				} else {
					r40 = d64.Reg
				}
				ctx.ProtectReg(r40)
				ctx.SyncDesc(&d63)
				ctx.EnsureDesc(&d63)
				if d63.Loc == LocImm {
					r41 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r41, uint64(d63.Imm.Int()))
				} else {
					r41 = d63.Reg
				}
				ctx.ProtectReg(r41)
				ctx.UnprotectReg(r41)
				ctx.UnprotectReg(r40)
				d65 = JITValueDesc{Loc: LocRegPair, Reg: r40, Reg2: r41}
				ctx.BindReg(r40, &d65)
				ctx.BindReg(r41, &d65)
				ctx.BindReg(r40, &d65)
				ctx.BindReg(r41, &d65)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d65)
				ctx.EnsureDesc(&d65)
				ctx.EnsureDesc(&d65)
				if d65.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d65.Imm)
					ptrWord, _ := d65.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d65.Imm.String())))
					d65 = tmpPair
				} else if d65.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocRegExcept(d65.Reg), Reg2: ctx.AllocRegExcept(d65.Reg)}
					switch d65.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d65)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d65)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d65)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d65)
					d65 = tmpPair
				}
				if d65.Loc != LocRegPair && d65.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.Index arg0)")
				}
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
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
				if d14.Loc != LocRegPair && d14.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.Index arg1)")
				}
				ctx.SyncDesc(&d65)
				ctx.SyncDesc(&d14)
				d66 := ctx.EmitGoCallScalar(GoFuncAddr(strings.Index), []JITValueDesc{d65, d14}, 1)
				ctx.BindReg(d66.Reg, &d66)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d66)
				ctx.EnsureDesc(&d43)
				ctx.ProtectReg(d43.Reg)
				ctx.EnsureDesc(&d66)
				ctx.UnprotectReg(d43.Reg)
				var d67 JITValueDesc
				if d43.Loc == LocImm && d66.Loc == LocImm {
					d67 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d43.Imm.Int() + d66.Imm.Int())}
				} else if d66.Loc == LocImm && d66.Imm.Int() == 0 {
					r42 := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegReg(r42, d43.Reg)
					d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r42}
					ctx.BindReg(r42, &d67)
				} else if d43.Loc == LocImm && d43.Imm.Int() == 0 {
					d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg}
					ctx.BindReg(d66.Reg, &d67)
				} else if d43.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d66.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
					ctx.EmitAddInt64(scratch, d66.Reg)
					d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d67)
				} else if d66.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegReg(scratch, d43.Reg)
					if d66.Imm.Int() >= -2147483648 && d66.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(scratch, int32(d66.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d66.Imm.Int()))
						ctx.EmitAddInt64(scratch, RegR11)
					}
					d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d67)
				} else {
					r43 := ctx.AllocRegExcept(d43.Reg, d66.Reg)
					ctx.EmitMovRegReg(r43, d43.Reg)
					ctx.EmitAddInt64(r43, d66.Reg)
					d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r43}
					ctx.BindReg(r43, &d67)
				}
				if d67.Loc == LocReg && d43.Loc == LocReg && d67.Reg == d43.Reg {
					ctx.TransferReg(d43.Reg)
					d43.Loc = LocNone
				}
				ctx.EnsureDesc(&d67)
				ctx.EmitStoreToStack(d67, int32(phiBase17)+int32(48))
				ctx.StabilizeDescForControlFlow(&d67)
				ctx.FreeDesc(&d66)
				ctx.ReclaimUntrackedRegs()
				bbpos_2_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d68 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d68)
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d68)
				var d70 JITValueDesc
				if d68.Loc == LocImm && d43.Loc == LocImm {
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d68.Imm.Int() - d43.Imm.Int())}
				} else {
					r44 := ctx.AllocReg()
					if d68.Loc == LocImm {
						ctx.EmitMovRegImm64(r44, uint64(d68.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r44, d68.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
						ctx.EmitSubInt64(r44, RegR11)
					} else {
						ctx.EmitSubInt64(r44, d43.Reg)
					}
					d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r44}
					ctx.BindReg(r44, &d70)
				}
				var d71 JITValueDesc
				if d13.Loc == LocImm && d43.Loc == LocImm {
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d43.Imm.Int()*1)}
				} else {
					r45 := ctx.AllocReg()
					if d13.Loc == LocImm {
						ctx.EmitMovRegImm64(r45, uint64(d13.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r45, d13.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()*1))
						ctx.EmitAddInt64(r45, RegR11)
					} else {
						ctx.EmitAddInt64(r45, d43.Reg)
					}
					d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r45}
					ctx.BindReg(r45, &d71)
				}
				var d72 JITValueDesc
				var r46 Reg
				var r47 Reg
				ctx.SyncDesc(&d71)
				ctx.EnsureDesc(&d71)
				if d71.Loc == LocImm {
					r46 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r46, uint64(d71.Imm.Int()))
				} else {
					r46 = d71.Reg
				}
				ctx.ProtectReg(r46)
				ctx.SyncDesc(&d70)
				ctx.EnsureDesc(&d70)
				if d70.Loc == LocImm {
					r47 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r47, uint64(d70.Imm.Int()))
				} else {
					r47 = d70.Reg
				}
				ctx.ProtectReg(r47)
				ctx.UnprotectReg(r47)
				ctx.UnprotectReg(r46)
				d72 = JITValueDesc{Loc: LocRegPair, Reg: r46, Reg2: r47}
				ctx.BindReg(r46, &d72)
				ctx.BindReg(r47, &d72)
				ctx.BindReg(r46, &d72)
				ctx.BindReg(r47, &d72)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d36)
				ctx.EnsureDesc(&d36)
				if d36.Loc == LocRegPair || d36.Loc == LocStackPair || d36.Loc == LocRegTriple || d36.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d72)
				ctx.EnsureDesc(&d72)
				ctx.EnsureDesc(&d72)
				if d72.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d72.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d72.Imm)
					ptrWord, _ := d72.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d72.Imm.String())))
					d72 = tmpPair
				} else if d72.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d72.Type, Reg: ctx.AllocRegExcept(d72.Reg), Reg2: ctx.AllocRegExcept(d72.Reg)}
					switch d72.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d72)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d72)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d72)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d72)
					d72 = tmpPair
				}
				if d72.Loc != LocRegPair && d72.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value ((*strings.Builder).WriteString arg1)")
				}
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d72)
				callResults73 := JITEmitGoCallResults(ctx, GoFuncAddr((*strings.Builder).WriteString), []JITValueDesc{d36, d72}, []uint8{1, 2}, []uint8{0, 3})
				d74 := callResults73[0]
				_ = d74
				d75 := callResults73[1]
				_ = d75
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d36)
				ctx.EnsureDesc(&d36)
				if d36.Loc == LocRegPair || d36.Loc == LocStackPair || d36.Loc == LocRegTriple || d36.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d15)
				ctx.EnsureDesc(&d15)
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
				if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value ((*strings.Builder).WriteString arg1)")
				}
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d15)
				callResults76 := JITEmitGoCallResults(ctx, GoFuncAddr((*strings.Builder).WriteString), []JITValueDesc{d36, d15}, []uint8{1, 2}, []uint8{0, 3})
				d77 := callResults76[0]
				_ = d77
				d78 := callResults76[1]
				_ = d78
				ctx.ReclaimUntrackedRegs()
				var d79 JITValueDesc
				if d14.SliceSizeKnown {
					d79 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.KnownSliceLen))}
				} else if d14.Loc == LocImm {
					d79 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d14.Imm.String())))}
				} else if d14.Loc == LocStackTriple {
					d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
				} else if d14.Loc == LocStackPair {
					d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocRegTriple {
						d79 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg2, ID: 0}
					} else if d14.Loc == LocReg {
						d79 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d68)
				ctx.EnsureDesc(&d79)
				ctx.EnsureDesc(&d68)
				ctx.ProtectReg(d68.Reg)
				ctx.EnsureDesc(&d79)
				ctx.UnprotectReg(d68.Reg)
				var d80 JITValueDesc
				if d68.Loc == LocImm && d79.Loc == LocImm {
					d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d68.Imm.Int() + d79.Imm.Int())}
				} else if d79.Loc == LocImm && d79.Imm.Int() == 0 {
					r48 := ctx.AllocRegExcept(d68.Reg)
					ctx.EmitMovRegReg(r48, d68.Reg)
					d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r48}
					ctx.BindReg(r48, &d80)
				} else if d68.Loc == LocImm && d68.Imm.Int() == 0 {
					d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg}
					ctx.BindReg(d79.Reg, &d80)
				} else if d68.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d79.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
					ctx.EmitAddInt64(scratch, d79.Reg)
					d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d80)
				} else if d79.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d68.Reg)
					ctx.EmitMovRegReg(scratch, d68.Reg)
					if d79.Imm.Int() >= -2147483648 && d79.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(scratch, int32(d79.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d79.Imm.Int()))
						ctx.EmitAddInt64(scratch, RegR11)
					}
					d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d80)
				} else {
					r49 := ctx.AllocRegExcept(d68.Reg, d79.Reg)
					ctx.EmitMovRegReg(r49, d68.Reg)
					ctx.EmitAddInt64(r49, d79.Reg)
					d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r49}
					ctx.BindReg(r49, &d80)
				}
				if d80.Loc == LocReg && d68.Loc == LocReg && d80.Reg == d68.Reg {
					ctx.TransferReg(d68.Reg)
					d68.Loc = LocNone
				}
				ctx.EnsureDesc(&d80)
				ctx.EmitStoreToStack(d80, int32(phiBase17)+int32(16))
				ctx.StabilizeDescForControlFlow(&d80)
				ctx.FreeDesc(&d68)
				ctx.FreeDesc(&d79)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d44)
				var d81 JITValueDesc
				if d44.Loc == LocImm {
					d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitMovRegReg(scratch, d44.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d81)
				}
				if d81.Loc == LocReg && d44.Loc == LocReg && d81.Reg == d44.Reg {
					ctx.TransferReg(d44.Reg)
					d44.Loc = LocNone
				}
				ctx.EnsureDesc(&d81)
				ctx.EmitStoreToStack(d81, int32(phiBase17)+int32(32))
				ctx.StabilizeDescForControlFlow(&d81)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmpToPos(bbpos_2_9)
				bbpos_2_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl23)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d68 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				var d82 JITValueDesc
				if d44.Loc == LocImm {
					d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d44.Imm.Int() > 0)}
				} else {
					r50 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d44.Reg, 0)
					ctx.EmitSetcc(r50, CondSignedGreater)
					d82 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r50}
					ctx.BindReg(r50, &d82)
				}
				ctx.FreeDesc(&d44)
				ctx.ReclaimUntrackedRegs()
				d83 := d82
				ctx.EnsureDesc(&d83)
				if d83.Loc != LocImm && d83.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl27 := ctx.ReserveLabel()
				lbl28 := ctx.ReserveLabel()
				lbl29 := ctx.ReserveLabel()
				lbl30 := ctx.ReserveLabel()
				if d83.Loc == LocImm {
					if d83.Imm.Bool() {
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					} else {
						ctx.MarkLabel(lbl30)
						ctx.SyncDesc(&d43)
						if d43.Loc == LocReg {
							ctx.ProtectReg(d43.Reg)
						} else if d43.Loc == LocRegPair {
							ctx.ProtectReg(d43.Reg)
							ctx.ProtectReg(d43.Reg2)
						}
						d84 := d43
						if d84.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d84)
						ctx.EmitStoreToStack(d84, int32(phiBase17)+int32(48))
						if d43.Loc == LocReg {
							ctx.UnprotectReg(d43.Reg)
						} else if d43.Loc == LocRegPair {
							ctx.UnprotectReg(d43.Reg)
							ctx.UnprotectReg(d43.Reg2)
						}
						ctx.EmitJmp(lbl28)
					}
				} else {
					ctx.EmitCmpRegImm32(d83.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl29)
					ctx.EmitJmp(lbl30)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl30)
					ctx.SyncDesc(&d43)
					if d43.Loc == LocReg {
						ctx.ProtectReg(d43.Reg)
					} else if d43.Loc == LocRegPair {
						ctx.ProtectReg(d43.Reg)
						ctx.ProtectReg(d43.Reg2)
					}
					d85 := d43
					if d85.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d85)
					ctx.EmitStoreToStack(d85, int32(phiBase17)+int32(48))
					if d43.Loc == LocReg {
						ctx.UnprotectReg(d43.Reg)
					} else if d43.Loc == LocRegPair {
						ctx.UnprotectReg(d43.Reg)
						ctx.UnprotectReg(d43.Reg2)
					}
					ctx.EmitJmp(lbl28)
				}
				ctx.FreeDesc(&d82)
				bbpos_2_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl27)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
				d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(16)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(32)}
				d68 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				var d86 JITValueDesc
				ctx.EnsureDesc(&d13)
				if d13.Loc == LocRegPair || d13.Loc == LocRegTriple {
					d86 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg2}
					ctx.BindReg(d13.Reg2, &d86)
				} else {
					panic("Slice with omitted high requires descriptor with length in Reg2")
				}
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d86)
				var d88 JITValueDesc
				if d86.Loc == LocImm && d43.Loc == LocImm {
					d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d86.Imm.Int() - d43.Imm.Int())}
				} else {
					r51 := ctx.AllocReg()
					if d86.Loc == LocImm {
						ctx.EmitMovRegImm64(r51, uint64(d86.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r51, d86.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
						ctx.EmitSubInt64(r51, RegR11)
					} else {
						ctx.EmitSubInt64(r51, d43.Reg)
					}
					d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r51}
					ctx.BindReg(r51, &d88)
				}
				var d89 JITValueDesc
				if d13.Loc == LocImm && d43.Loc == LocImm {
					d89 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d43.Imm.Int()*1)}
				} else {
					r52 := ctx.AllocReg()
					if d13.Loc == LocImm {
						ctx.EmitMovRegImm64(r52, uint64(d13.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r52, d13.Reg)
					}
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()*1))
						ctx.EmitAddInt64(r52, RegR11)
					} else {
						ctx.EmitAddInt64(r52, d43.Reg)
					}
					d89 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r52}
					ctx.BindReg(r52, &d89)
				}
				var d90 JITValueDesc
				var r53 Reg
				var r54 Reg
				ctx.SyncDesc(&d89)
				ctx.EnsureDesc(&d89)
				if d89.Loc == LocImm {
					r53 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r53, uint64(d89.Imm.Int()))
				} else {
					r53 = d89.Reg
				}
				ctx.ProtectReg(r53)
				ctx.SyncDesc(&d88)
				ctx.EnsureDesc(&d88)
				if d88.Loc == LocImm {
					r54 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r54, uint64(d88.Imm.Int()))
				} else {
					r54 = d88.Reg
				}
				ctx.ProtectReg(r54)
				ctx.UnprotectReg(r54)
				ctx.UnprotectReg(r53)
				d90 = JITValueDesc{Loc: LocRegPair, Reg: r53, Reg2: r54}
				ctx.BindReg(r53, &d90)
				ctx.BindReg(r54, &d90)
				ctx.BindReg(r53, &d90)
				ctx.BindReg(r54, &d90)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d90)
				ctx.EnsureDesc(&d90)
				ctx.EnsureDesc(&d90)
				if d90.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d90.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d90.Imm)
					ptrWord, _ := d90.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d90.Imm.String())))
					d90 = tmpPair
				} else if d90.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d90.Type, Reg: ctx.AllocRegExcept(d90.Reg), Reg2: ctx.AllocRegExcept(d90.Reg)}
					switch d90.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d90)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d90)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d90)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d90)
					d90 = tmpPair
				}
				if d90.Loc != LocRegPair && d90.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (utf8.DecodeRuneInString arg0)")
				}
				ctx.SyncDesc(&d90)
				callResults91 := JITEmitGoCallResults(ctx, GoFuncAddr(utf8.DecodeRuneInString), []JITValueDesc{d90}, []uint8{1, 1}, []uint8{0, 0})
				d92 := callResults91[0]
				_ = d92
				d93 := callResults91[1]
				_ = d93
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d93)
				ctx.EnsureDesc(&d43)
				ctx.ProtectReg(d43.Reg)
				ctx.EnsureDesc(&d93)
				ctx.UnprotectReg(d43.Reg)
				var d94 JITValueDesc
				if d43.Loc == LocImm && d93.Loc == LocImm {
					d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d43.Imm.Int() + d93.Imm.Int())}
				} else if d93.Loc == LocImm && d93.Imm.Int() == 0 {
					r55 := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegReg(r55, d43.Reg)
					d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r55}
					ctx.BindReg(r55, &d94)
				} else if d43.Loc == LocImm && d43.Imm.Int() == 0 {
					d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d93.Reg}
					ctx.BindReg(d93.Reg, &d94)
				} else if d43.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d93.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
					ctx.EmitAddInt64(scratch, d93.Reg)
					d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d94)
				} else if d93.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegReg(scratch, d43.Reg)
					if d93.Imm.Int() >= -2147483648 && d93.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(scratch, int32(d93.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d93.Imm.Int()))
						ctx.EmitAddInt64(scratch, RegR11)
					}
					d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d94)
				} else {
					r56 := ctx.AllocRegExcept(d43.Reg, d93.Reg)
					ctx.EmitMovRegReg(r56, d43.Reg)
					ctx.EmitAddInt64(r56, d93.Reg)
					d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r56}
					ctx.BindReg(r56, &d94)
				}
				if d94.Loc == LocReg && d43.Loc == LocReg && d94.Reg == d43.Reg {
					ctx.TransferReg(d43.Reg)
					d43.Loc = LocNone
				}
				ctx.EnsureDesc(&d94)
				ctx.EmitStoreToStack(d94, int32(phiBase17)+int32(48))
				ctx.StabilizeDescForControlFlow(&d94)
				ctx.FreeDesc(&d93)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl28)
				ctx.MarkLabel(lbl0)
				d95 := JITValueDesc{Loc: LocReg, Reg: r36}
				ctx.BindReg(r36, &d95)
				ctx.BindReg(r36, &d95)
				if r0 {
					ctx.UnprotectReg(r1)
				}
				if r2 {
					ctx.UnprotectReg(r3)
				}
				if r4 {
					ctx.UnprotectReg(r5)
				}
				if r6 {
					ctx.UnprotectReg(r7)
				}
				if r8 {
					ctx.UnprotectReg(r9)
				}
				if r10 {
					ctx.UnprotectReg(r11)
				}
				if r12 {
					ctx.UnprotectReg(r13)
				}
				if r14 {
					ctx.UnprotectReg(r15)
				}
				if r16 {
					ctx.UnprotectReg(r17)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d95)
				ctx.EnsureDesc(&d95)
				d96 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d95}, 2)
				if result.Loc == LocAny {
					return d96
				}
				ctx.EmitMovPairToResult(&d96, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 69,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strtrim"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strltrim"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
				}
				d3 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d3.Imm)
					ptrWord, _ := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d3.Imm.String())))
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
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d1, d3}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.FreeDesc(&d3)
				ctx.EnsureDesc(&d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strrtrim"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
				}
				d3 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d3.Imm)
					ptrWord, _ := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d3.Imm.String())))
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
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d1, d3}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.FreeDesc(&d3)
				ctx.EnsureDesc(&d4)
				d5 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
				if result.Loc == LocAny {
					return d5
				}
				ctx.EmitMovPairToResult(&d5, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_trim"].Fn, args, result)
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
						return bbs[2].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
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
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d16.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d16.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					} else if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
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
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d15)
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d15}, 2)
					ctx.BindReg(d17.Reg, &d17)
					ctx.BindReg(d17.Reg2, &d17)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d17}, 2)
					ctx.EmitMovPairToResult(&d18, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps19 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps19)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 12,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_ltrim"].Fn, args, result)
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
						return bbs[2].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
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
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d16.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d16.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					} else if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
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
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d17 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d17.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d17.Imm)
						ptrWord, _ := d17.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d17.Imm.String())))
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
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d15, d17}, 2)
					ctx.BindReg(d18.Reg, &d18)
					ctx.BindReg(d18.Reg2, &d18)
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d18)
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d18}, 2)
					ctx.EmitMovPairToResult(&d19, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps20 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps20)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 12,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_rtrim"].Fn, args, result)
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
						return bbs[2].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
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
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d16.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d16.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					} else if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
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
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d17 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d17.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d17.Imm)
						ptrWord, _ := d17.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d17.Imm.String())))
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
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d15, d17}, 2)
					ctx.BindReg(d18.Reg, &d18)
					ctx.BindReg(d18.Reg2, &d18)
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d18)
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d18}, 2)
					ctx.EmitMovPairToResult(&d19, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps20 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps20)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 12,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["split"].Fn, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d8 JITValueDesc
				_ = d8
				var d11 JITValueDesc
				_ = d11
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
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
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
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d43 JITValueDesc
				_ = d43
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d83 JITValueDesc
				_ = d83
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				var bbs [6]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[3].PhiBase = int32(phiBase0) + int32(16)
				bbs[3].PhiCount = uint16(1)
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
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d3)
					var d4 JITValueDesc
					if d3.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() > 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d3.Reg, 1)
						ctx.EmitSetcc(r0, CondSignedGreater)
						d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d4)
					}
					ctx.FreeDesc(&d3)
					d5 = d4
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
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps7 := PhiState{General: ps.General}
						ps7.OverlayValues = make([]JITValueDesc, 6)
						ps7.OverlayValues[1] = d1
						ps7.OverlayValues[2] = d2
						ps7.OverlayValues[3] = d3
						ps7.OverlayValues[4] = d4
						ps7.OverlayValues[5] = d5
						ps7.PhiValues = make([]JITValueDesc, 1)
						d8 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}
						ps7.PhiValues[0] = d8
						return bbs[2].RenderPS(ps7)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 9)
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					ps9.OverlayValues[8] = d8
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 9)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[8] = d8
					ps10.PhiValues = make([]JITValueDesc, 1)
					d11 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}
					ps10.PhiValues[0] = d11
					snap12 := d1
					snap13 := d2
					snap14 := d3
					snap15 := d4
					snap16 := d5
					snap17 := d8
					snap18 := d11
					alloc19 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc19)
					d1 = snap12
					d2 = snap13
					d3 = snap14
					d4 = snap15
					d5 = snap16
					d8 = snap17
					d11 = snap18
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					ctx.ReclaimUntrackedRegs()
					d20 = args[1]
					d20.ID = 0
					d22 = d20
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d22.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d22)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d22)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d22)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d22.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d22 = tmpPair
					} else if d22.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d22.Reg), Reg2: ctx.AllocRegExcept(d22.Reg)}
						switch d22.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d22)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d22)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d22)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d22)
						d22 = tmpPair
					} else if d22.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d22.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d22.MemPtr))
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
						d22 = tmpPair
					}
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d21 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d22}, 2)
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d20)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d23 = d21
						if d23.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d23)
						if d23.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d23, int32(bbs[2].PhiBase)+int32(0), 2)
						} else if d23.Loc == LocInputPair {
							ctx.EnsureDesc(&d23)
							ctx.EmitStoreScmerToStack(d23, int32(bbs[2].PhiBase)+int32(0))
						} else if d23.Loc == LocRegPair || d23.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d23, int32(bbs[2].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d23)
							ctx.EmitStoreToStack(d23, int32(bbs[2].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
						}
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps24 := PhiState{General: ps.General}
					ps24.OverlayValues = make([]JITValueDesc, 24)
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[8] = d8
					ps24.OverlayValues[11] = d11
					ps24.OverlayValues[20] = d20
					ps24.OverlayValues[21] = d21
					ps24.OverlayValues[22] = d22
					ps24.OverlayValues[23] = d23
					ps24.PhiValues = make([]JITValueDesc, 1)
					d25 = d21
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
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d26 := ps.PhiValues[0]
							ctx.EnsureDesc(&d26)
							ctx.EmitStoreScmerToStack(d26, int32(bbs[2].PhiBase)+int32(0))
						}
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					d27 = args[0]
					d27.ID = 0
					d29 = d27
					ctx.EnsureDesc(&d29)
					if d29.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d29.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d29)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d29)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d29)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d29.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d29 = tmpPair
					} else if d29.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d29.Reg), Reg2: ctx.AllocRegExcept(d29.Reg)}
						switch d29.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d29)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d29)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d29)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d29)
						d29 = tmpPair
					} else if d29.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d29.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d29.MemPtr))
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
						d29 = tmpPair
					}
					if d29.Loc != LocRegPair && d29.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d29}, 2)
					ctx.FreeDesc(&d27)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d28.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d28.Imm)
						ptrWord, _ := d28.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d28.Imm.String())))
						d28 = tmpPair
					} else if d28.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d28.Type, Reg: ctx.AllocRegExcept(d28.Reg), Reg2: ctx.AllocRegExcept(d28.Reg)}
						switch d28.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d28)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d28)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d28)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d28)
						d28 = tmpPair
					}
					if d28.Loc != LocRegPair && d28.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.Split arg0)")
					}
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.Split arg1)")
					}
					ctx.SyncDesc(&d28)
					ctx.SyncDesc(&d1)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Split), []JITValueDesc{d28, d1}, 3)
					ctx.BindReg(d30.Reg, &d30)
					ctx.BindReg(d30.Reg2, &d30)
					ctx.BindReg(d30.Reg3, &d30)
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.FreeDesc(&d1)
					var d31 JITValueDesc
					if d30.SliceSizeKnown {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.KnownSliceLen))}
					} else if d30.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.StackOff))}
					} else if d30.Loc == LocStackTriple {
						d31 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d30.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d30)
						if d30.Loc == LocRegPair || d30.Loc == LocRegTriple {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg2, ID: 0}
						} else if d30.Loc == LocReg {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					callResults32 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d31, d31}, []uint8{3}, []uint8{1})
					d33 = callResults32[0]
					d33.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d33)
					ctx.FreeDesc(&d31)
					var d34 JITValueDesc
					if d30.SliceSizeKnown {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.KnownSliceLen))}
					} else if d30.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.StackOff))}
					} else if d30.Loc == LocStackTriple {
						d34 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d30.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d30)
						if d30.Loc == LocRegPair || d30.Loc == LocRegTriple {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg2, ID: 0}
						} else if d30.Loc == LocReg {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d34)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					}
					ps35 := PhiState{General: ps.General}
					ps35.OverlayValues = make([]JITValueDesc, 35)
					ps35.OverlayValues[1] = d1
					ps35.OverlayValues[2] = d2
					ps35.OverlayValues[3] = d3
					ps35.OverlayValues[4] = d4
					ps35.OverlayValues[5] = d5
					ps35.OverlayValues[8] = d8
					ps35.OverlayValues[11] = d11
					ps35.OverlayValues[20] = d20
					ps35.OverlayValues[21] = d21
					ps35.OverlayValues[22] = d22
					ps35.OverlayValues[23] = d23
					ps35.OverlayValues[25] = d25
					ps35.OverlayValues[26] = d26
					ps35.OverlayValues[27] = d27
					ps35.OverlayValues[28] = d28
					ps35.OverlayValues[29] = d29
					ps35.OverlayValues[30] = d30
					ps35.OverlayValues[31] = d31
					ps35.OverlayValues[33] = d33
					ps35.OverlayValues[34] = d34
					ps35.PhiValues = make([]JITValueDesc, 1)
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps35.PhiValues[0] = d36
					if ps35.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps35)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d37 := ps.PhiValues[0]
							ctx.EnsureDesc(&d37)
							ctx.EmitStoreToStack(d37, int32(bbs[3].PhiBase)+int32(0))
						}
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d38 JITValueDesc
					if d2.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d38)
					}
					if d38.Loc == LocReg && d2.Loc == LocReg && d38.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d38)
					ctx.EmitStoreToStack(d38, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d34)
					var d39 JITValueDesc
					if d38.Loc == LocImm && d34.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d38.Imm.Int() < d34.Imm.Int())}
					} else if d34.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d38.Reg)
						if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d38.Reg, int32(d34.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d34.Imm.Int()))
							ctx.EmitCmpInt64(d38.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedLess)
						d39 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d39)
					} else if d38.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d38.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d34.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d39 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d39)
					} else {
						r3 := ctx.AllocRegExcept(d38.Reg)
						ctx.EmitCmpInt64(d38.Reg, d34.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d39 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d39)
					}
					ctx.FreeDesc(&d34)
					d40 = d39
					ctx.EnsureDesc(&d40)
					if d40.Loc != LocImm && d40.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d40.Loc == LocImm {
						if d40.Imm.Bool() {
							if ps.General {
							}
							ps41 := PhiState{General: ps.General}
							ps41.OverlayValues = make([]JITValueDesc, 41)
							ps41.OverlayValues[1] = d1
							ps41.OverlayValues[2] = d2
							ps41.OverlayValues[3] = d3
							ps41.OverlayValues[4] = d4
							ps41.OverlayValues[5] = d5
							ps41.OverlayValues[8] = d8
							ps41.OverlayValues[11] = d11
							ps41.OverlayValues[20] = d20
							ps41.OverlayValues[21] = d21
							ps41.OverlayValues[22] = d22
							ps41.OverlayValues[23] = d23
							ps41.OverlayValues[25] = d25
							ps41.OverlayValues[26] = d26
							ps41.OverlayValues[27] = d27
							ps41.OverlayValues[28] = d28
							ps41.OverlayValues[29] = d29
							ps41.OverlayValues[30] = d30
							ps41.OverlayValues[31] = d31
							ps41.OverlayValues[33] = d33
							ps41.OverlayValues[34] = d34
							ps41.OverlayValues[36] = d36
							ps41.OverlayValues[37] = d37
							ps41.OverlayValues[38] = d38
							ps41.OverlayValues[39] = d39
							ps41.OverlayValues[40] = d40
							return bbs[4].RenderPS(ps41)
						}
						if ps.General {
						}
						ps42 := PhiState{General: ps.General}
						ps42.OverlayValues = make([]JITValueDesc, 41)
						ps42.OverlayValues[1] = d1
						ps42.OverlayValues[2] = d2
						ps42.OverlayValues[3] = d3
						ps42.OverlayValues[4] = d4
						ps42.OverlayValues[5] = d5
						ps42.OverlayValues[8] = d8
						ps42.OverlayValues[11] = d11
						ps42.OverlayValues[20] = d20
						ps42.OverlayValues[21] = d21
						ps42.OverlayValues[22] = d22
						ps42.OverlayValues[23] = d23
						ps42.OverlayValues[25] = d25
						ps42.OverlayValues[26] = d26
						ps42.OverlayValues[27] = d27
						ps42.OverlayValues[28] = d28
						ps42.OverlayValues[29] = d29
						ps42.OverlayValues[30] = d30
						ps42.OverlayValues[31] = d31
						ps42.OverlayValues[33] = d33
						ps42.OverlayValues[34] = d34
						ps42.OverlayValues[36] = d36
						ps42.OverlayValues[37] = d37
						ps42.OverlayValues[38] = d38
						ps42.OverlayValues[39] = d39
						ps42.OverlayValues[40] = d40
						return bbs[5].RenderPS(ps42)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d43 := ps.PhiValues[0]
							ctx.EnsureDesc(&d43)
							ctx.EmitStoreToStack(d43, int32(bbs[3].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d40.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 44)
					ps44.OverlayValues[1] = d1
					ps44.OverlayValues[2] = d2
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[8] = d8
					ps44.OverlayValues[11] = d11
					ps44.OverlayValues[20] = d20
					ps44.OverlayValues[21] = d21
					ps44.OverlayValues[22] = d22
					ps44.OverlayValues[23] = d23
					ps44.OverlayValues[25] = d25
					ps44.OverlayValues[26] = d26
					ps44.OverlayValues[27] = d27
					ps44.OverlayValues[28] = d28
					ps44.OverlayValues[29] = d29
					ps44.OverlayValues[30] = d30
					ps44.OverlayValues[31] = d31
					ps44.OverlayValues[33] = d33
					ps44.OverlayValues[34] = d34
					ps44.OverlayValues[36] = d36
					ps44.OverlayValues[37] = d37
					ps44.OverlayValues[38] = d38
					ps44.OverlayValues[39] = d39
					ps44.OverlayValues[40] = d40
					ps44.OverlayValues[43] = d43
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 44)
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[20] = d20
					ps45.OverlayValues[21] = d21
					ps45.OverlayValues[22] = d22
					ps45.OverlayValues[23] = d23
					ps45.OverlayValues[25] = d25
					ps45.OverlayValues[26] = d26
					ps45.OverlayValues[27] = d27
					ps45.OverlayValues[28] = d28
					ps45.OverlayValues[29] = d29
					ps45.OverlayValues[30] = d30
					ps45.OverlayValues[31] = d31
					ps45.OverlayValues[33] = d33
					ps45.OverlayValues[34] = d34
					ps45.OverlayValues[36] = d36
					ps45.OverlayValues[37] = d37
					ps45.OverlayValues[38] = d38
					ps45.OverlayValues[39] = d39
					ps45.OverlayValues[40] = d40
					ps45.OverlayValues[43] = d43
					snap46 := d1
					snap47 := d2
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d8
					snap52 := d11
					snap53 := d20
					snap54 := d21
					snap55 := d22
					snap56 := d23
					snap57 := d25
					snap58 := d26
					snap59 := d27
					snap60 := d28
					snap61 := d29
					snap62 := d30
					snap63 := d31
					snap64 := d33
					snap65 := d34
					snap66 := d36
					snap67 := d37
					snap68 := d38
					snap69 := d39
					snap70 := d40
					snap71 := d43
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps45)
					}
					ctx.RestoreAllocState(alloc72)
					d1 = snap46
					d2 = snap47
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d8 = snap51
					d11 = snap52
					d20 = snap53
					d21 = snap54
					d22 = snap55
					d23 = snap56
					d25 = snap57
					d26 = snap58
					d27 = snap59
					d28 = snap60
					d29 = snap61
					d30 = snap62
					d31 = snap63
					d33 = snap64
					d34 = snap65
					d36 = snap66
					d37 = snap67
					d38 = snap68
					d39 = snap69
					d40 = snap70
					d43 = snap71
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps44)
					}
					return result
					ctx.FreeDesc(&d39)
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs73 := make([]Reg, 0, 3)
					seenBlockPinnedRegs74 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs74
					for _, r := range []Reg{d33.Reg, d33.Reg2, d33.Reg3} {
						live := d33.Loc == LocRegTriple && (r == d33.Reg || r == d33.Reg2 || r == d33.Reg3)
						if live && !seenBlockPinnedRegs74[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs74[r] = true
							blockPinnedRegs73 = append(blockPinnedRegs73, r)
						}
					}
					unpinBlockRegs75 := func() {
						for _, r := range blockPinnedRegs73 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs75()
					ctx.EnsureDesc(&d38)
					d77 = ctx.EmitSliceElementAddress(&d30, &d38, 16)
					ctx.EnsureDesc(&d77)
					r4 := ctx.AllocRegExcept(d77.Reg)
					ctx.EmitMovRegMem(r4, d77.Reg, 8)
					ctx.EmitMovRegMem(d77.Reg, d77.Reg, 0)
					d76 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d77.Reg, Reg2: r4}
					ctx.BindReg(d77.Reg, &d76)
					ctx.BindReg(r4, &d76)
					ctx.EnsureDesc(&d76)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d76)
					d78 = ctx.EmitSliceElementAddress(&d33, &d38, int32(16))
					ctx.EmitStoreScmerAt(&d78, &d76)
					ctx.FreeDesc(&d78)
					if ps.General {
					}
					ps79 := PhiState{General: ps.General}
					ps79.OverlayValues = make([]JITValueDesc, 79)
					ps79.OverlayValues[1] = d1
					ps79.OverlayValues[2] = d2
					ps79.OverlayValues[3] = d3
					ps79.OverlayValues[4] = d4
					ps79.OverlayValues[5] = d5
					ps79.OverlayValues[8] = d8
					ps79.OverlayValues[11] = d11
					ps79.OverlayValues[20] = d20
					ps79.OverlayValues[21] = d21
					ps79.OverlayValues[22] = d22
					ps79.OverlayValues[23] = d23
					ps79.OverlayValues[25] = d25
					ps79.OverlayValues[26] = d26
					ps79.OverlayValues[27] = d27
					ps79.OverlayValues[28] = d28
					ps79.OverlayValues[29] = d29
					ps79.OverlayValues[30] = d30
					ps79.OverlayValues[31] = d31
					ps79.OverlayValues[33] = d33
					ps79.OverlayValues[34] = d34
					ps79.OverlayValues[36] = d36
					ps79.OverlayValues[37] = d37
					ps79.OverlayValues[38] = d38
					ps79.OverlayValues[39] = d39
					ps79.OverlayValues[40] = d40
					ps79.OverlayValues[43] = d43
					ps79.OverlayValues[76] = d76
					ps79.OverlayValues[77] = d77
					ps79.OverlayValues[78] = d78
					ps79.PhiValues = make([]JITValueDesc, 1)
					if ps79.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps79)
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
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
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
					blockPinnedRegs80 := make([]Reg, 0, 3)
					seenBlockPinnedRegs81 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs81
					for _, r := range []Reg{d33.Reg, d33.Reg2, d33.Reg3} {
						live := d33.Loc == LocRegTriple && (r == d33.Reg || r == d33.Reg2 || r == d33.Reg3)
						if live && !seenBlockPinnedRegs81[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs81[r] = true
							blockPinnedRegs80 = append(blockPinnedRegs80, r)
						}
					}
					unpinBlockRegs82 := func() {
						for _, r := range blockPinnedRegs80 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs82()
					d83 = ctx.EmitNewSliceFromGoSlice(&d33)
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocRegPair {
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
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps84 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps84)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  28,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["string_repeat"].Fn, args, result)
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
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
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
						return bbs[2].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
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
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d15 = d14
					_ = d15
					ctx.StabilizeDescForControlFlow(&d15)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d16)
					var d18 JITValueDesc
					if d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() <= 0)}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 0)
						ctx.EmitSetcc(r0, CondSignedLessOrEqual)
						d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d18)
					}
					d19 = d18
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							if ps.General {
							}
							ps20 := PhiState{General: ps.General}
							ps20.OverlayValues = make([]JITValueDesc, 20)
							ps20.OverlayValues[0] = d0
							ps20.OverlayValues[1] = d1
							ps20.OverlayValues[2] = d2
							ps20.OverlayValues[3] = d3
							ps20.OverlayValues[13] = d13
							ps20.OverlayValues[14] = d14
							ps20.OverlayValues[15] = d15
							ps20.OverlayValues[16] = d16
							ps20.OverlayValues[17] = d17
							ps20.OverlayValues[18] = d18
							ps20.OverlayValues[19] = d19
							return bbs[3].RenderPS(ps20)
						}
						if ps.General {
						}
						ps21 := PhiState{General: ps.General}
						ps21.OverlayValues = make([]JITValueDesc, 20)
						ps21.OverlayValues[0] = d0
						ps21.OverlayValues[1] = d1
						ps21.OverlayValues[2] = d2
						ps21.OverlayValues[3] = d3
						ps21.OverlayValues[13] = d13
						ps21.OverlayValues[14] = d14
						ps21.OverlayValues[15] = d15
						ps21.OverlayValues[16] = d16
						ps21.OverlayValues[17] = d17
						ps21.OverlayValues[18] = d18
						ps21.OverlayValues[19] = d19
						return bbs[4].RenderPS(ps21)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ps22 := PhiState{General: true}
					ps22.OverlayValues = make([]JITValueDesc, 20)
					ps22.OverlayValues[0] = d0
					ps22.OverlayValues[1] = d1
					ps22.OverlayValues[2] = d2
					ps22.OverlayValues[3] = d3
					ps22.OverlayValues[13] = d13
					ps22.OverlayValues[14] = d14
					ps22.OverlayValues[15] = d15
					ps22.OverlayValues[16] = d16
					ps22.OverlayValues[17] = d17
					ps22.OverlayValues[18] = d18
					ps22.OverlayValues[19] = d19
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 20)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[13] = d13
					ps23.OverlayValues[14] = d14
					ps23.OverlayValues[15] = d15
					ps23.OverlayValues[16] = d16
					ps23.OverlayValues[17] = d17
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[19] = d19
					snap24 := d0
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d13
					snap29 := d14
					snap30 := d15
					snap31 := d16
					snap32 := d17
					snap33 := d18
					snap34 := d19
					alloc35 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps23)
					}
					ctx.RestoreAllocState(alloc35)
					d0 = snap24
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d13 = snap28
					d14 = snap29
					d15 = snap30
					d16 = snap31
					d17 = snap32
					d18 = snap33
					d19 = snap34
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps22)
					}
					return result
					ctx.FreeDesc(&d18)
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
					d36 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d36}, 2)
					ctx.EmitMovPairToResult(&d37, &result)
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
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					ctx.ReclaimUntrackedRegs()
					d38 = args[0]
					d38.ID = 0
					d40 = d38
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d40.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d40)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d40)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d40)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d40.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d40 = tmpPair
					} else if d40.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d40.Reg), Reg2: ctx.AllocRegExcept(d40.Reg)}
						switch d40.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d40)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d40)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d40)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d40)
						d40 = tmpPair
					} else if d40.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d40.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d40.MemPtr))
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
						d40 = tmpPair
					}
					if d40.Loc != LocRegPair && d40.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d39 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d40}, 2)
					ctx.FreeDesc(&d38)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d39.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d39.Imm)
						ptrWord, _ := d39.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d39.Imm.String())))
						d39 = tmpPair
					} else if d39.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d39.Type, Reg: ctx.AllocRegExcept(d39.Reg), Reg2: ctx.AllocRegExcept(d39.Reg)}
						switch d39.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d39)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d39)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d39)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d39)
						d39 = tmpPair
					}
					if d39.Loc != LocRegPair && d39.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.Repeat arg0)")
					}
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d39)
					ctx.SyncDesc(&d16)
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Repeat), []JITValueDesc{d39, d16}, 2)
					ctx.BindReg(d41.Reg, &d41)
					ctx.BindReg(d41.Reg2, &d41)
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d41)
					d42 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d41}, 2)
					ctx.EmitMovPairToResult(&d42, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps43 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps43)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 22,
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
			Return: &TypeDescriptor{Kind: "func", Label: "relation", Description: "compares two values using the selected collation and direction",
				Params: []*TypeDescriptor{
					{Kind: "any", Label: "a", Description: "left operand"},
					{Kind: "any", Label: "b", Description: "right operand"},
				},
				Return: &TypeDescriptor{Kind: "bool", Label: "ordered", Description: "whether a sorts before b"},
			},
			Const: true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["collate"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["htmlentities"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (html.EscapeString arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(html.EscapeString), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["urlencode"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (url.QueryEscape arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(url.QueryEscape), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["urldecode"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d20 JITValueDesc
				_ = d20
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (url.QueryUnescape arg0)")
					}
					ctx.SyncDesc(&d1)
					callResults3 := JITEmitGoCallResults(ctx, GoFuncAddr(url.QueryUnescape), []JITValueDesc{d1}, []uint8{2, 2}, []uint8{1, 3})
					d4 = callResults3[0]
					_ = d4
					d5 = callResults3[1]
					_ = d5
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.EnsureDesc(&d5)
					var d6 JITValueDesc
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc != LocReg && d5.Loc != LocRegPair && d5.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d6)
					}
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							if ps.General {
							}
							ps8 := PhiState{General: ps.General}
							ps8.OverlayValues = make([]JITValueDesc, 8)
							ps8.OverlayValues[0] = d0
							ps8.OverlayValues[1] = d1
							ps8.OverlayValues[2] = d2
							ps8.OverlayValues[4] = d4
							ps8.OverlayValues[5] = d5
							ps8.OverlayValues[6] = d6
							ps8.OverlayValues[7] = d7
							return bbs[1].RenderPS(ps8)
						}
						if ps.General {
						}
						ps9 := PhiState{General: ps.General}
						ps9.OverlayValues = make([]JITValueDesc, 8)
						ps9.OverlayValues[0] = d0
						ps9.OverlayValues[1] = d1
						ps9.OverlayValues[2] = d2
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
					ctx.EmitJump(CondNotEqual, lbl4)
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
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[7] = d7
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 8)
					ps11.OverlayValues[0] = d0
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[4] = d4
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[6] = d6
					ps11.OverlayValues[7] = d7
					snap12 := d0
					snap13 := d1
					snap14 := d2
					snap15 := d4
					snap16 := d5
					snap17 := d6
					snap18 := d7
					alloc19 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps11)
					}
					ctx.RestoreAllocState(alloc19)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					d7 = snap18
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["urldecode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.EnsureDesc(&d4)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
					ctx.EmitMovPairToResult(&d20, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps21 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps21)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  19,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_encode"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
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
				var d20 JITValueDesc
				_ = d20
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
					ctx.EnsureDesc(&d0)
					d1 = ctx.EmitGoCallScalar(GoFuncAddr(func(value Scmer) any { return value }), []JITValueDesc{d0}, 2)
					ctx.FreeDesc(&d0)
					ctx.EnsureDesc(&d1)
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
						panic("jit: generic call arg expects 2-word value (json.Marshal arg0)")
					}
					ctx.SyncDesc(&d1)
					callResults2 := JITEmitGoCallResults(ctx, GoFuncAddr(json.Marshal), []JITValueDesc{d1}, []uint8{3, 2}, []uint8{1, 3})
					d3 = callResults2[0]
					_ = d3
					d4 = callResults2[1]
					_ = d4
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.EnsureDesc(&d4)
					var d5 JITValueDesc
					if d4.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d4)
						if d4.Loc != LocReg && d4.Loc != LocRegPair && d4.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpRegImm32(d4.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d5)
					}
					d6 = d5
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
							ps7.OverlayValues[0] = d0
							ps7.OverlayValues[1] = d1
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
						ps8.OverlayValues[0] = d0
						ps8.OverlayValues[1] = d1
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
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 7)
					ps9.OverlayValues[0] = d0
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					ps9.OverlayValues[6] = d6
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 7)
					ps10.OverlayValues[0] = d0
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					snap11 := d0
					snap12 := d1
					snap13 := d3
					snap14 := d4
					snap15 := d5
					snap16 := d6
					alloc17 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc17)
					d0 = snap11
					d1 = snap12
					d3 = snap13
					d4 = snap14
					d5 = snap15
					d6 = snap16
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps9)
					}
					return result
					ctx.FreeDesc(&d5)
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_encode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					callResults19 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d3}, []uint8{2}, []uint8{1})
					d18 = callResults19[0]
					ctx.EnsureDesc(&d18)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d18}, 2)
					ctx.EmitMovPairToResult(&d20, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps21 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps21)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  13,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_quote"].Fn, args, result)
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
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
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
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var phiBase87 int32
				_ = phiBase87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
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
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
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
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(func() *bytes.Buffer { return new(bytes.Buffer) }), nil, 1)
					ctx.BindReg(d14.Reg, &d14)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *bytes.Buffer) io.Writer { return value }), []JITValueDesc{d14}, 2)
					ctx.EnsureDesc(&d15)
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
						panic("jit: generic call arg expects 2-word value (json.NewEncoder arg0)")
					}
					ctx.SyncDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(json.NewEncoder), []JITValueDesc{d15}, 1)
					ctx.BindReg(d16.Reg, &d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					if d17.Loc == LocRegPair || d17.Loc == LocStackPair || d17.Loc == LocRegTriple || d17.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d17)
					ctx.EmitGoCallVoid(GoFuncAddr((*json.Encoder).SetEscapeHTML), []JITValueDesc{d16, d17})
					ctx.FreeDesc(&d17)
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
					ctx.EnsureDesc(&d19)
					d21 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d19}, 2)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d21)
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
						panic("jit: generic call arg expects 2-word value ((*json.Encoder).Encode arg1)")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d21)
					d22 = ctx.EmitGoCallScalar(GoFuncAddr((*json.Encoder).Encode), []JITValueDesc{d16, d21}, 2)
					ctx.BindReg(d22.Reg, &d22)
					ctx.BindReg(d22.Reg2, &d22)
					ctx.StabilizeDescForControlFlow(&d22)
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d22)
					var d23 JITValueDesc
					if d22.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d22.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d22)
						if d22.Loc != LocReg && d22.Loc != LocRegPair && d22.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d22.Reg)
						ctx.EmitCmpRegImm32(d22.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d23)
					}
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
							ps25.OverlayValues[24] = d24
							return bbs[4].RenderPS(ps25)
						}
						if ps.General {
						}
						ps26 := PhiState{General: ps.General}
						ps26.OverlayValues = make([]JITValueDesc, 25)
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
						ps26.OverlayValues[24] = d24
						return bbs[5].RenderPS(ps26)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d24.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 25)
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
					ps27.OverlayValues[24] = d24
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 25)
					ps28.OverlayValues[0] = d0
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[13] = d13
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[15] = d15
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[18] = d18
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[24] = d24
					snap29 := d0
					snap30 := d1
					snap31 := d2
					snap32 := d3
					snap33 := d13
					snap34 := d14
					snap35 := d15
					snap36 := d16
					snap37 := d17
					snap38 := d18
					snap39 := d19
					snap40 := d20
					snap41 := d21
					snap42 := d22
					snap43 := d23
					snap44 := d24
					alloc45 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps28)
					}
					ctx.RestoreAllocState(alloc45)
					d0 = snap29
					d1 = snap30
					d2 = snap31
					d3 = snap32
					d13 = snap33
					d14 = snap34
					d15 = snap35
					d16 = snap36
					d17 = snap37
					d18 = snap38
					d19 = snap39
					d20 = snap40
					d21 = snap41
					d22 = snap42
					d23 = snap43
					d24 = snap44
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps27)
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
					ctx.ReclaimUntrackedRegs()
					d46 = args[0]
					d46.ID = 0
					d48 = d46
					d48.ID = 0
					d47 = ctx.EmitTagEqualsBorrowed(&d48, tagString, JITValueDesc{Loc: LocAny})
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
							ps50.OverlayValues[0] = d0
							ps50.OverlayValues[1] = d1
							ps50.OverlayValues[2] = d2
							ps50.OverlayValues[3] = d3
							ps50.OverlayValues[13] = d13
							ps50.OverlayValues[14] = d14
							ps50.OverlayValues[15] = d15
							ps50.OverlayValues[16] = d16
							ps50.OverlayValues[17] = d17
							ps50.OverlayValues[18] = d18
							ps50.OverlayValues[19] = d19
							ps50.OverlayValues[20] = d20
							ps50.OverlayValues[21] = d21
							ps50.OverlayValues[22] = d22
							ps50.OverlayValues[23] = d23
							ps50.OverlayValues[24] = d24
							ps50.OverlayValues[46] = d46
							ps50.OverlayValues[47] = d47
							ps50.OverlayValues[48] = d48
							ps50.OverlayValues[49] = d49
							return bbs[2].RenderPS(ps50)
						}
						if ps.General {
						}
						ps51 := PhiState{General: ps.General}
						ps51.OverlayValues = make([]JITValueDesc, 50)
						ps51.OverlayValues[0] = d0
						ps51.OverlayValues[1] = d1
						ps51.OverlayValues[2] = d2
						ps51.OverlayValues[3] = d3
						ps51.OverlayValues[13] = d13
						ps51.OverlayValues[14] = d14
						ps51.OverlayValues[15] = d15
						ps51.OverlayValues[16] = d16
						ps51.OverlayValues[17] = d17
						ps51.OverlayValues[18] = d18
						ps51.OverlayValues[19] = d19
						ps51.OverlayValues[20] = d20
						ps51.OverlayValues[21] = d21
						ps51.OverlayValues[22] = d22
						ps51.OverlayValues[23] = d23
						ps51.OverlayValues[24] = d24
						ps51.OverlayValues[46] = d46
						ps51.OverlayValues[47] = d47
						ps51.OverlayValues[48] = d48
						ps51.OverlayValues[49] = d49
						return bbs[1].RenderPS(ps51)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d49.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl2)
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 50)
					ps52.OverlayValues[0] = d0
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[13] = d13
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[16] = d16
					ps52.OverlayValues[17] = d17
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[20] = d20
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[46] = d46
					ps52.OverlayValues[47] = d47
					ps52.OverlayValues[48] = d48
					ps52.OverlayValues[49] = d49
					ps53 := PhiState{General: true}
					ps53.OverlayValues = make([]JITValueDesc, 50)
					ps53.OverlayValues[0] = d0
					ps53.OverlayValues[1] = d1
					ps53.OverlayValues[2] = d2
					ps53.OverlayValues[3] = d3
					ps53.OverlayValues[13] = d13
					ps53.OverlayValues[14] = d14
					ps53.OverlayValues[15] = d15
					ps53.OverlayValues[16] = d16
					ps53.OverlayValues[17] = d17
					ps53.OverlayValues[18] = d18
					ps53.OverlayValues[19] = d19
					ps53.OverlayValues[20] = d20
					ps53.OverlayValues[21] = d21
					ps53.OverlayValues[22] = d22
					ps53.OverlayValues[23] = d23
					ps53.OverlayValues[24] = d24
					ps53.OverlayValues[46] = d46
					ps53.OverlayValues[47] = d47
					ps53.OverlayValues[48] = d48
					ps53.OverlayValues[49] = d49
					snap54 := d0
					snap55 := d1
					snap56 := d2
					snap57 := d3
					snap58 := d13
					snap59 := d14
					snap60 := d15
					snap61 := d16
					snap62 := d17
					snap63 := d18
					snap64 := d19
					snap65 := d20
					snap66 := d21
					snap67 := d22
					snap68 := d23
					snap69 := d24
					snap70 := d46
					snap71 := d47
					snap72 := d48
					snap73 := d49
					alloc74 := ctx.SnapshotAllocState()
					if !bbs[1].Rendered {
						bbs[1].RenderPS(ps53)
					}
					ctx.RestoreAllocState(alloc74)
					d0 = snap54
					d1 = snap55
					d2 = snap56
					d3 = snap57
					d13 = snap58
					d14 = snap59
					d15 = snap60
					d16 = snap61
					d17 = snap62
					d18 = snap63
					d19 = snap64
					d20 = snap65
					d21 = snap66
					d22 = snap67
					d23 = snap68
					d24 = snap69
					d46 = snap70
					d47 = snap71
					d48 = snap72
					d49 = snap73
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps52)
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_quote"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
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
					blockPinnedRegs75 := make([]Reg, 0, 3)
					seenBlockPinnedRegs76 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs76
					for _, r := range []Reg{d14.Reg, d14.Reg2, d14.Reg3} {
						live := d14.Loc == LocRegTriple && (r == d14.Reg || r == d14.Reg2 || r == d14.Reg3)
						if live && !seenBlockPinnedRegs76[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs76[r] = true
							blockPinnedRegs75 = append(blockPinnedRegs75, r)
						}
					}
					unpinBlockRegs77 := func() {
						for _, r := range blockPinnedRegs75 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs77()
					d79 = d14
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
					ctx.EnsureDesc(&d78)
					d80 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("\n")}
					d81 = d78
					_ = d81
					ctx.StabilizeDescForControlFlow(&d81)
					d82 = d80
					_ = d82
					ctx.StabilizeDescForControlFlow(&d82)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d82)
					d83 = d81
					_ = d83
					ctx.StabilizeDescForControlFlow(&d83)
					d84 = d82
					_ = d84
					ctx.StabilizeDescForControlFlow(&d84)
					r1 := d81.Loc == LocReg || d81.Loc == LocRegPair || d81.Loc == LocRegTriple
					r2 := d81.Reg
					if r1 {
						ctx.ProtectReg(r2)
					}
					r3 := d81.Loc == LocRegPair || d81.Loc == LocRegTriple
					r4 := d81.Reg2
					if r3 {
						ctx.ProtectReg(r4)
					}
					r5 := d81.Loc == LocRegTriple
					r6 := d81.Reg3
					if r5 {
						ctx.ProtectReg(r6)
					}
					lbl13 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d84)
					d85 = d83
					_ = d85
					ctx.StabilizeDescForControlFlow(&d85)
					d86 = d84
					_ = d86
					ctx.StabilizeDescForControlFlow(&d86)
					r7 := d83.Loc == LocReg || d83.Loc == LocRegPair || d83.Loc == LocRegTriple
					r8 := d83.Reg
					if r7 {
						ctx.ProtectReg(r8)
					}
					r9 := d83.Loc == LocRegPair || d83.Loc == LocRegTriple
					r10 := d83.Reg2
					if r9 {
						ctx.ProtectReg(r10)
					}
					r11 := d83.Loc == LocRegTriple
					r12 := d83.Reg3
					if r11 {
						ctx.ProtectReg(r12)
					}
					r13 := d84.Loc == LocReg || d84.Loc == LocRegPair || d84.Loc == LocRegTriple
					r14 := d84.Reg
					if r13 {
						ctx.ProtectReg(r14)
					}
					r15 := d84.Loc == LocRegPair || d84.Loc == LocRegTriple
					r16 := d84.Reg2
					if r15 {
						ctx.ProtectReg(r16)
					}
					r17 := d84.Loc == LocRegTriple
					r18 := d84.Reg3
					if r17 {
						ctx.ProtectReg(r18)
					}
					phiBase87 = ctx.AllocStack(int32(16))
					d88 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase87) + int32(0)}
					_ = d88
					lbl14 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d88 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase87) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d89 JITValueDesc
					if d85.SliceSizeKnown {
						d89 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d85.KnownSliceLen))}
					} else if d85.Loc == LocImm {
						d89 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d85.Imm.String())))}
					} else if d85.Loc == LocStackTriple {
						d89 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d85.StackOff + 8, NoHeapPointer: true}
					} else if d85.Loc == LocStackPair {
						d89 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d85.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d85)
						if d85.Loc == LocRegPair || d85.Loc == LocRegTriple {
							d89 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d85.Reg2, ID: 0}
						} else if d85.Loc == LocReg {
							d89 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d85.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d90 JITValueDesc
					if d86.SliceSizeKnown {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d86.KnownSliceLen))}
					} else if d86.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d86.StackOff))}
					} else if d86.Loc == LocStackTriple {
						d90 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d86.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d86)
						if d86.Loc == LocRegPair || d86.Loc == LocRegTriple {
							d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg2, ID: 0}
						} else if d86.Loc == LocReg {
							d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d90)
					var d91 JITValueDesc
					if d89.Loc == LocImm && d90.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d89.Imm.Int() >= d90.Imm.Int())}
					} else if d90.Loc == LocImm {
						r19 := ctx.AllocReg()
						if d90.Imm.Int() >= -2147483648 && d90.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d89.Reg, int32(d90.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d90.Imm.Int()))
							ctx.EmitCmpInt64(d89.Reg, RegR11)
						}
						ctx.EmitSetcc(r19, CondSignedGreaterOrEqual)
						d91 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
						ctx.BindReg(r19, &d91)
					} else if d89.Loc == LocImm {
						r20 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d89.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d90.Reg)
						ctx.EmitSetcc(r20, CondSignedGreaterOrEqual)
						d91 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
						ctx.BindReg(r20, &d91)
					} else {
						r21 := ctx.AllocReg()
						ctx.EmitCmpInt64(d89.Reg, d90.Reg)
						ctx.EmitSetcc(r21, CondSignedGreaterOrEqual)
						d91 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d91)
					}
					ctx.FreeDesc(&d89)
					ctx.FreeDesc(&d90)
					ctx.ReclaimUntrackedRegs()
					d92 = d91
					ctx.EnsureDesc(&d92)
					if d92.Loc != LocImm && d92.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					if d92.Loc == LocImm {
						if d92.Imm.Bool() {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl15)
						} else {
							ctx.MarkLabel(lbl18)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase87)+int32(0))
							ctx.EmitJmp(lbl16)
						}
					} else {
						ctx.EmitCmpRegImm32(d92.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl17)
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
						ctx.MarkLabel(lbl18)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase87)+int32(0))
						ctx.EmitJmp(lbl16)
					}
					ctx.FreeDesc(&d91)
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d88 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase87) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r22 := ctx.AllocReg()
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r22, d88)
					}
					ctx.EmitJmp(lbl14)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d88 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase87) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d93 JITValueDesc
					if d85.SliceSizeKnown {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d85.KnownSliceLen))}
					} else if d85.Loc == LocImm {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d85.Imm.String())))}
					} else if d85.Loc == LocStackTriple {
						d93 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d85.StackOff + 8, NoHeapPointer: true}
					} else if d85.Loc == LocStackPair {
						d93 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d85.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d85)
						if d85.Loc == LocRegPair || d85.Loc == LocRegTriple {
							d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d85.Reg2, ID: 0}
						} else if d85.Loc == LocReg {
							d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d85.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d94 JITValueDesc
					if d86.SliceSizeKnown {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d86.KnownSliceLen))}
					} else if d86.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d86.StackOff))}
					} else if d86.Loc == LocStackTriple {
						d94 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d86.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d86)
						if d86.Loc == LocRegPair || d86.Loc == LocRegTriple {
							d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg2, ID: 0}
						} else if d86.Loc == LocReg {
							d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d93)
					ctx.EnsureDesc(&d94)
					ctx.EnsureDesc(&d93)
					ctx.ProtectReg(d93.Reg)
					ctx.EnsureDesc(&d94)
					ctx.UnprotectReg(d93.Reg)
					var d95 JITValueDesc
					if d93.Loc == LocImm && d94.Loc == LocImm {
						d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d93.Imm.Int() - d94.Imm.Int())}
					} else if d94.Loc == LocImm && d94.Imm.Int() == 0 {
						r23 := ctx.AllocRegExcept(d93.Reg)
						ctx.EmitMovRegReg(r23, d93.Reg)
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r23}
						ctx.BindReg(r23, &d95)
					} else if d93.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d94.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d93.Imm.Int()))
						ctx.EmitSubInt64(scratch, d94.Reg)
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d95)
					} else if d94.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d93.Reg)
						ctx.EmitMovRegReg(scratch, d93.Reg)
						if d94.Imm.Int() >= -2147483648 && d94.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d94.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d94.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d95)
					} else {
						r24 := ctx.AllocRegExcept(d93.Reg, d94.Reg)
						ctx.EmitMovRegReg(r24, d93.Reg)
						ctx.EmitSubInt64(r24, d94.Reg)
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
						ctx.BindReg(r24, &d95)
					}
					if d95.Loc == LocReg && d93.Loc == LocReg && d95.Reg == d93.Reg {
						ctx.TransferReg(d93.Reg)
						d93.Loc = LocNone
					}
					ctx.FreeDesc(&d93)
					ctx.FreeDesc(&d94)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d95)
					var d96 JITValueDesc
					ctx.EnsureDesc(&d85)
					if d85.Loc == LocRegPair || d85.Loc == LocRegTriple {
						d96 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d85.Reg2}
						ctx.BindReg(d85.Reg2, &d96)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d85)
					ctx.EnsureDesc(&d95)
					ctx.EnsureDesc(&d96)
					var d98 JITValueDesc
					if d96.Loc == LocImm && d95.Loc == LocImm {
						d98 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d96.Imm.Int() - d95.Imm.Int())}
					} else {
						r25 := ctx.AllocReg()
						if d96.Loc == LocImm {
							ctx.EmitMovRegImm64(r25, uint64(d96.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r25, d96.Reg)
						}
						if d95.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d95.Imm.Int()))
							ctx.EmitSubInt64(r25, RegR11)
						} else {
							ctx.EmitSubInt64(r25, d95.Reg)
						}
						d98 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d98)
					}
					var d99 JITValueDesc
					if d85.Loc == LocImm && d95.Loc == LocImm {
						d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d85.Imm.Int() + d95.Imm.Int()*1)}
					} else {
						r26 := ctx.AllocReg()
						if d85.Loc == LocImm {
							ctx.EmitMovRegImm64(r26, uint64(d85.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r26, d85.Reg)
						}
						if d95.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d95.Imm.Int()*1))
							ctx.EmitAddInt64(r26, RegR11)
						} else {
							ctx.EmitAddInt64(r26, d95.Reg)
						}
						d99 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
						ctx.BindReg(r26, &d99)
					}
					var d100 JITValueDesc
					var r27 Reg
					var r28 Reg
					ctx.SyncDesc(&d99)
					ctx.EnsureDesc(&d99)
					if d99.Loc == LocImm {
						r27 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r27, uint64(d99.Imm.Int()))
					} else {
						r27 = d99.Reg
					}
					ctx.ProtectReg(r27)
					ctx.SyncDesc(&d98)
					ctx.EnsureDesc(&d98)
					if d98.Loc == LocImm {
						r28 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r28, uint64(d98.Imm.Int()))
					} else {
						r28 = d98.Reg
					}
					ctx.ProtectReg(r28)
					ctx.UnprotectReg(r28)
					ctx.UnprotectReg(r27)
					d100 = JITValueDesc{Loc: LocRegPair, Reg: r27, Reg2: r28}
					ctx.BindReg(r27, &d100)
					ctx.BindReg(r28, &d100)
					ctx.BindReg(r27, &d100)
					ctx.BindReg(r28, &d100)
					ctx.FreeDesc(&d95)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d86)
					var d101 JITValueDesc
					if d86.Loc == LocImm {
						ctx.TrackImm(d86.Imm)
						ptrWord, _ := d86.Imm.RawWords()
						d101 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d101.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d101.Reg2, uint64(len(d86.Imm.String())))
						ctx.BindReg(d101.Reg, &d101)
						ctx.BindReg(d101.Reg2, &d101)
					} else {
						d101 = d86
					}
					d102 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d100, d101}, 1)
					ctx.EmitAndRegImm32(d102.Reg, 1)
					d102.Type = tagBool
					ctx.BindReg(d102.Reg, &d102)
					ctx.EnsureDesc(&d102)
					ctx.EmitStoreToStack(d102, int32(phiBase87)+int32(0))
					ctx.StabilizeDescForControlFlow(&d102)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl14)
					d103 = JITValueDesc{Loc: LocReg, Reg: r22}
					ctx.BindReg(r22, &d103)
					ctx.BindReg(r22, &d103)
					if r7 {
						ctx.UnprotectReg(r8)
					}
					if r9 {
						ctx.UnprotectReg(r10)
					}
					if r11 {
						ctx.UnprotectReg(r12)
					}
					if r13 {
						ctx.UnprotectReg(r14)
					}
					if r15 {
						ctx.UnprotectReg(r16)
					}
					if r17 {
						ctx.UnprotectReg(r18)
					}
					ctx.ReclaimUntrackedRegs()
					d104 = d103
					ctx.EnsureDesc(&d104)
					if d104.Loc != LocImm && d104.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					if d104.Loc == LocImm {
						if d104.Imm.Bool() {
							ctx.MarkLabel(lbl21)
							ctx.EmitJmp(lbl19)
						} else {
							ctx.MarkLabel(lbl22)
							ctx.EmitJmp(lbl20)
						}
					} else {
						ctx.EmitCmpRegImm32(d104.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl21)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl20)
					}
					ctx.FreeDesc(&d103)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r29 := ctx.AllocReg()
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r29, d83)
					}
					ctx.EmitJmp(lbl13)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d105 JITValueDesc
					if d83.SliceSizeKnown {
						d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d83.KnownSliceLen))}
					} else if d83.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d83.Imm.String())))}
					} else if d83.Loc == LocStackTriple {
						d105 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d83.StackOff + 8, NoHeapPointer: true}
					} else if d83.Loc == LocStackPair {
						d105 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d83.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d83)
						if d83.Loc == LocRegPair || d83.Loc == LocRegTriple {
							d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg2, ID: 0}
						} else if d83.Loc == LocReg {
							d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d106 JITValueDesc
					if d84.SliceSizeKnown {
						d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d84.KnownSliceLen))}
					} else if d84.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d84.StackOff))}
					} else if d84.Loc == LocStackTriple {
						d106 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d84.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d84)
						if d84.Loc == LocRegPair || d84.Loc == LocRegTriple {
							d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d84.Reg2, ID: 0}
						} else if d84.Loc == LocReg {
							d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d84.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d105)
					ctx.ProtectReg(d105.Reg)
					ctx.EnsureDesc(&d106)
					ctx.UnprotectReg(d105.Reg)
					var d107 JITValueDesc
					if d105.Loc == LocImm && d106.Loc == LocImm {
						d107 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() - d106.Imm.Int())}
					} else if d106.Loc == LocImm && d106.Imm.Int() == 0 {
						r30 := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(r30, d105.Reg)
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
						ctx.BindReg(r30, &d107)
					} else if d105.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d106.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d105.Imm.Int()))
						ctx.EmitSubInt64(scratch, d106.Reg)
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d107)
					} else if d106.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(scratch, d105.Reg)
						if d106.Imm.Int() >= -2147483648 && d106.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d106.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d106.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d107)
					} else {
						r31 := ctx.AllocRegExcept(d105.Reg, d106.Reg)
						ctx.EmitMovRegReg(r31, d105.Reg)
						ctx.EmitSubInt64(r31, d106.Reg)
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r31}
						ctx.BindReg(r31, &d107)
					}
					if d107.Loc == LocReg && d105.Loc == LocReg && d107.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d107)
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d108)
					ctx.EnsureDesc(&d107)
					var d110 JITValueDesc
					if d107.Loc == LocImm && d108.Loc == LocImm {
						d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d107.Imm.Int() - d108.Imm.Int())}
					} else {
						r32 := ctx.AllocReg()
						if d107.Loc == LocImm {
							ctx.EmitMovRegImm64(r32, uint64(d107.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r32, d107.Reg)
						}
						if d108.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d108.Imm.Int()))
							ctx.EmitSubInt64(r32, RegR11)
						} else {
							ctx.EmitSubInt64(r32, d108.Reg)
						}
						d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r32}
						ctx.BindReg(r32, &d110)
					}
					var d111 JITValueDesc
					if d83.Loc == LocImm && d108.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d83.Imm.Int() + d108.Imm.Int()*1)}
					} else {
						r33 := ctx.AllocReg()
						if d83.Loc == LocImm {
							ctx.EmitMovRegImm64(r33, uint64(d83.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r33, d83.Reg)
						}
						if d108.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d108.Imm.Int()*1))
							ctx.EmitAddInt64(r33, RegR11)
						} else {
							ctx.EmitAddInt64(r33, d108.Reg)
						}
						d111 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r33}
						ctx.BindReg(r33, &d111)
					}
					var d112 JITValueDesc
					var r34 Reg
					var r35 Reg
					ctx.SyncDesc(&d111)
					ctx.EnsureDesc(&d111)
					if d111.Loc == LocImm {
						r34 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r34, uint64(d111.Imm.Int()))
					} else {
						r34 = d111.Reg
					}
					ctx.ProtectReg(r34)
					ctx.SyncDesc(&d110)
					ctx.EnsureDesc(&d110)
					if d110.Loc == LocImm {
						r35 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r35, uint64(d110.Imm.Int()))
					} else {
						r35 = d110.Reg
					}
					ctx.ProtectReg(r35)
					ctx.UnprotectReg(r35)
					ctx.UnprotectReg(r34)
					d112 = JITValueDesc{Loc: LocRegPair, Reg: r34, Reg2: r35}
					ctx.BindReg(r34, &d112)
					ctx.BindReg(r35, &d112)
					ctx.BindReg(r34, &d112)
					ctx.BindReg(r35, &d112)
					ctx.FreeDesc(&d107)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d112)
					ctx.EnsureDesc(&d112)
					if d112.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r29, d112)
					}
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl13)
					d113 = JITValueDesc{Loc: LocReg, Reg: r29}
					ctx.BindReg(r29, &d113)
					ctx.BindReg(r29, &d113)
					if r1 {
						ctx.UnprotectReg(r2)
					}
					if r3 {
						ctx.UnprotectReg(r4)
					}
					if r5 {
						ctx.UnprotectReg(r6)
					}
					ctx.FreeDesc(&d82)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d113)
					ctx.EnsureDesc(&d113)
					d114 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d113}, 2)
					ctx.EmitMovPairToResult(&d114, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps115 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps115)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  49,
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_encode_assoc"].Fn, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode"].Fn, args, result)
				}
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
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
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
					d0 = ctx.EmitGoCallScalar(GoFuncAddr(func() *any { return new(any) }), nil, 1)
					ctx.BindReg(d0.Reg, &d0)
					ctx.StabilizeDescForControlFlow(&d0)
					d1 = args[0]
					d1.ID = 0
					d3 = d1
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
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					callResults5 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d2}, []uint8{3}, []uint8{1})
					d4 = callResults5[0]
					d4.Type = tagSlice
					ctx.EnsureDesc(&d0)
					d6 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *any) any { return value }), []JITValueDesc{d0}, 2)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (json.Unmarshal arg0)")
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d6.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d6)
						} else if d6.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d6)
						} else if d6.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d6)
						} else if d6.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d6.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d6 = tmpPair
					} else if d6.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocRegExcept(d6.Reg), Reg2: ctx.AllocRegExcept(d6.Reg)}
						switch d6.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d6)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d6)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d6)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d6)
						d6 = tmpPair
					}
					if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (json.Unmarshal arg1)")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d6)
					d7 = ctx.EmitGoCallScalar(GoFuncAddr(json.Unmarshal), []JITValueDesc{d4, d6}, 2)
					ctx.BindReg(d7.Reg, &d7)
					ctx.BindReg(d7.Reg2, &d7)
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.EnsureDesc(&d7)
					var d8 JITValueDesc
					if d7.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d7)
						if d7.Loc != LocReg && d7.Loc != LocRegPair && d7.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpRegImm32(d7.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d8)
					}
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							if ps.General {
							}
							ps10 := PhiState{General: ps.General}
							ps10.OverlayValues = make([]JITValueDesc, 10)
							ps10.OverlayValues[0] = d0
							ps10.OverlayValues[1] = d1
							ps10.OverlayValues[2] = d2
							ps10.OverlayValues[3] = d3
							ps10.OverlayValues[4] = d4
							ps10.OverlayValues[6] = d6
							ps10.OverlayValues[7] = d7
							ps10.OverlayValues[8] = d8
							ps10.OverlayValues[9] = d9
							return bbs[1].RenderPS(ps10)
						}
						if ps.General {
						}
						ps11 := PhiState{General: ps.General}
						ps11.OverlayValues = make([]JITValueDesc, 10)
						ps11.OverlayValues[0] = d0
						ps11.OverlayValues[1] = d1
						ps11.OverlayValues[2] = d2
						ps11.OverlayValues[3] = d3
						ps11.OverlayValues[4] = d4
						ps11.OverlayValues[6] = d6
						ps11.OverlayValues[7] = d7
						ps11.OverlayValues[8] = d8
						ps11.OverlayValues[9] = d9
						return bbs[2].RenderPS(ps11)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 10)
					ps12.OverlayValues[0] = d0
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					ps12.OverlayValues[9] = d9
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 10)
					ps13.OverlayValues[0] = d0
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[9] = d9
					snap14 := d0
					snap15 := d1
					snap16 := d2
					snap17 := d3
					snap18 := d4
					snap19 := d6
					snap20 := d7
					snap21 := d8
					snap22 := d9
					alloc23 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps13)
					}
					ctx.RestoreAllocState(alloc23)
					d0 = snap14
					d1 = snap15
					d2 = snap16
					d3 = snap17
					d4 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
					}
					return result
					ctx.FreeDesc(&d8)
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs24 := make([]Reg, 0, 3)
					seenBlockPinnedRegs25 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs25
					for _, r := range []Reg{d0.Reg, d0.Reg2, d0.Reg3} {
						live := d0.Loc == LocRegTriple && (r == d0.Reg || r == d0.Reg2 || r == d0.Reg3)
						if live && !seenBlockPinnedRegs25[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs25[r] = true
							blockPinnedRegs24 = append(blockPinnedRegs24, r)
						}
					}
					unpinBlockRegs26 := func() {
						for _, r := range blockPinnedRegs24 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs26()
					d27 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *any) any { return *value }), []JITValueDesc{d0}, 2)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
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
						panic("jit: generic call arg expects 2-word value (TransformFromJSON arg0)")
					}
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(TransformFromJSON), []JITValueDesc{d27}, 2)
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d28, &result)
						result.Type = d28.Type
					} else {
						switch d28.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d28)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d28)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d28)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d28, &result)
							result.Type = d28.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps29 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps29)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  14,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode_scmer"].Fn, args, result)
				}
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
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d24 JITValueDesc
				_ = d24
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
					r0 := ctx.AllocReg()
					r1 := ctx.AllocRegExcept(r0)
					ctx.EmitMovRegImm64(r0, 0)
					ctx.EmitMovRegImm64(r1, 0)
					d0 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r0, Reg2: r1}
					ctx.BindReg(r0, &d0)
					ctx.BindReg(r1, &d0)
					ctx.StabilizeDescForControlFlow(&d0)
					d1 = args[0]
					d1.ID = 0
					d3 = d1
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
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					callResults5 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d2}, []uint8{3}, []uint8{1})
					d4 = callResults5[0]
					d4.Type = tagSlice
					ctx.EnsureDesc(&d0)
					d6 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *Scmer) any { return value }), []JITValueDesc{d0}, 2)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (json.Unmarshal arg0)")
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d6.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d6)
						} else if d6.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d6)
						} else if d6.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d6)
						} else if d6.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d6.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d6 = tmpPair
					} else if d6.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocRegExcept(d6.Reg), Reg2: ctx.AllocRegExcept(d6.Reg)}
						switch d6.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d6)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d6)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d6)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d6)
						d6 = tmpPair
					}
					if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (json.Unmarshal arg1)")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d6)
					d7 = ctx.EmitGoCallScalar(GoFuncAddr(json.Unmarshal), []JITValueDesc{d4, d6}, 2)
					ctx.BindReg(d7.Reg, &d7)
					ctx.BindReg(d7.Reg2, &d7)
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.EnsureDesc(&d7)
					var d8 JITValueDesc
					if d7.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d7)
						if d7.Loc != LocReg && d7.Loc != LocRegPair && d7.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpRegImm32(d7.Reg, 0)
						ctx.EmitSetcc(r2, CondNotEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							if ps.General {
							}
							ps10 := PhiState{General: ps.General}
							ps10.OverlayValues = make([]JITValueDesc, 10)
							ps10.OverlayValues[0] = d0
							ps10.OverlayValues[1] = d1
							ps10.OverlayValues[2] = d2
							ps10.OverlayValues[3] = d3
							ps10.OverlayValues[4] = d4
							ps10.OverlayValues[6] = d6
							ps10.OverlayValues[7] = d7
							ps10.OverlayValues[8] = d8
							ps10.OverlayValues[9] = d9
							return bbs[1].RenderPS(ps10)
						}
						if ps.General {
						}
						ps11 := PhiState{General: ps.General}
						ps11.OverlayValues = make([]JITValueDesc, 10)
						ps11.OverlayValues[0] = d0
						ps11.OverlayValues[1] = d1
						ps11.OverlayValues[2] = d2
						ps11.OverlayValues[3] = d3
						ps11.OverlayValues[4] = d4
						ps11.OverlayValues[6] = d6
						ps11.OverlayValues[7] = d7
						ps11.OverlayValues[8] = d8
						ps11.OverlayValues[9] = d9
						return bbs[2].RenderPS(ps11)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 10)
					ps12.OverlayValues[0] = d0
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					ps12.OverlayValues[9] = d9
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 10)
					ps13.OverlayValues[0] = d0
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[9] = d9
					snap14 := d0
					snap15 := d1
					snap16 := d2
					snap17 := d3
					snap18 := d4
					snap19 := d6
					snap20 := d7
					snap21 := d8
					snap22 := d9
					alloc23 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps13)
					}
					ctx.RestoreAllocState(alloc23)
					d0 = snap14
					d1 = snap15
					d2 = snap16
					d3 = snap17
					d4 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
					}
					return result
					ctx.FreeDesc(&d8)
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode_scmer"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					d24 = d0
					_ = d24
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d24, &result)
						result.Type = d24.Type
					} else {
						switch d24.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d24)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d24)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d24)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d24, &result)
							result.Type = d24.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps25 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps25)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  13,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_encode"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *base64.Encoding { return base64.StdEncoding }), nil, 1)
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
				ctx.EnsureDesc(&d2)
				callResults5 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d2}, []uint8{3}, []uint8{1})
				d4 := callResults5[0]
				d4.Type = tagSlice
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d4)
				d6 := d4
				_ = d6
				ctx.StabilizeDescForControlFlow(&d6)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d7 JITValueDesc
				if d6.SliceSizeKnown {
					d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.KnownSliceLen))}
				} else if d6.Loc == LocImm {
					d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.StackOff))}
				} else if d6.Loc == LocStackTriple {
					d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocRegPair || d6.Loc == LocRegTriple {
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2, ID: 0}
					} else if d6.Loc == LocReg {
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocRegPair || d0.Loc == LocStackPair || d0.Loc == LocRegTriple || d0.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d7)
				ctx.EnsureDesc(&d7)
				if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocRegTriple || d7.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d7)
				d8 := ctx.EmitGoCallScalar(GoFuncAddr((*base64.Encoding).EncodedLen), []JITValueDesc{d0, d7}, 1)
				ctx.BindReg(d8.Reg, &d8)
				ctx.FreeDesc(&d7)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d8)
				callResults9 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d8, d8}, []uint8{3}, []uint8{1})
				d10 := callResults9[0]
				d10.Type = tagSlice
				ctx.FreeDesc(&d8)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocRegPair || d0.Loc == LocStackPair || d0.Loc == LocRegTriple || d0.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				if d10.Loc != LocRegTriple && d10.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((*base64.Encoding).Encode arg1)")
				}
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d6)
				if d6.Loc != LocRegTriple && d6.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice ((*base64.Encoding).Encode arg2)")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d10)
				ctx.SyncDesc(&d6)
				ctx.EmitGoCallVoid(GoFuncAddr((*base64.Encoding).Encode), []JITValueDesc{d0, d10, d6})
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				callResults12 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d10}, []uint8{2}, []uint8{1})
				d11 := callResults12[0]
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d11)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d11)
				d13 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d11}, 2)
				if result.Loc == LocAny {
					return d13
				}
				ctx.EmitMovPairToResult(&d13, &result)
				result.Type = tagString
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  14,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_decode"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d22 JITValueDesc
				_ = d22
				var d24 JITValueDesc
				_ = d24
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
					d0 = ctx.EmitGoCallScalar(GoFuncAddr(func() *base64.Encoding { return base64.StdEncoding }), nil, 1)
					d1 = args[0]
					d1.ID = 0
					d3 = d1
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
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d0)
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocRegPair || d0.Loc == LocStackPair || d0.Loc == LocRegTriple || d0.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d2)
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
						panic("jit: generic call arg expects 2-word value ((*base64.Encoding).DecodeString arg1)")
					}
					ctx.SyncDesc(&d0)
					ctx.SyncDesc(&d2)
					callResults4 := JITEmitGoCallResults(ctx, GoFuncAddr((*base64.Encoding).DecodeString), []JITValueDesc{d0, d2}, []uint8{3, 2}, []uint8{1, 3})
					d5 = callResults4[0]
					_ = d5
					d6 = callResults4[1]
					_ = d6
					ctx.FreeDesc(&d0)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d6)
					ctx.EnsureDesc(&d6)
					var d7 JITValueDesc
					if d6.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d6)
						if d6.Loc != LocReg && d6.Loc != LocRegPair && d6.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d7)
					}
					d8 = d7
					ctx.EnsureDesc(&d8)
					if d8.Loc != LocImm && d8.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d8.Loc == LocImm {
						if d8.Imm.Bool() {
							if ps.General {
							}
							ps9 := PhiState{General: ps.General}
							ps9.OverlayValues = make([]JITValueDesc, 9)
							ps9.OverlayValues[0] = d0
							ps9.OverlayValues[1] = d1
							ps9.OverlayValues[2] = d2
							ps9.OverlayValues[3] = d3
							ps9.OverlayValues[5] = d5
							ps9.OverlayValues[6] = d6
							ps9.OverlayValues[7] = d7
							ps9.OverlayValues[8] = d8
							return bbs[1].RenderPS(ps9)
						}
						if ps.General {
						}
						ps10 := PhiState{General: ps.General}
						ps10.OverlayValues = make([]JITValueDesc, 9)
						ps10.OverlayValues[0] = d0
						ps10.OverlayValues[1] = d1
						ps10.OverlayValues[2] = d2
						ps10.OverlayValues[3] = d3
						ps10.OverlayValues[5] = d5
						ps10.OverlayValues[6] = d6
						ps10.OverlayValues[7] = d7
						ps10.OverlayValues[8] = d8
						return bbs[2].RenderPS(ps10)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d8.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 9)
					ps11.OverlayValues[0] = d0
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[3] = d3
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[6] = d6
					ps11.OverlayValues[7] = d7
					ps11.OverlayValues[8] = d8
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 9)
					ps12.OverlayValues[0] = d0
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[5] = d5
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					snap13 := d0
					snap14 := d1
					snap15 := d2
					snap16 := d3
					snap17 := d5
					snap18 := d6
					snap19 := d7
					snap20 := d8
					alloc21 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps12)
					}
					ctx.RestoreAllocState(alloc21)
					d0 = snap13
					d1 = snap14
					d2 = snap15
					d3 = snap16
					d5 = snap17
					d6 = snap18
					d7 = snap19
					d8 = snap20
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps11)
					}
					return result
					ctx.FreeDesc(&d7)
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_decode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					callResults23 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d5}, []uint8{2}, []uint8{1})
					d22 = callResults23[0]
					ctx.EnsureDesc(&d22)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d22}, 2)
					ctx.EmitMovPairToResult(&d24, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps25 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps25)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  21,
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: static Go callback value.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_unescape"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["bin2hex"].Fn, args, result)
				}
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
				var d9 JITValueDesc
				_ = d9
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
				var d18 JITValueDesc
				_ = d18
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
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d63 JITValueDesc
				_ = d63
				var d65 JITValueDesc
				_ = d65
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d4 = d2
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d4.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d4)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d4)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d4)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d4.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d4 = tmpPair
					} else if d4.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
						switch d4.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d4)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d4)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d4)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d4)
						d4 = tmpPair
					} else if d4.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d4.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d4.MemPtr))
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
						d4 = tmpPair
					}
					if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d3 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d4}, 2)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					var d5 JITValueDesc
					if d3.SliceSizeKnown {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d3.Imm.String())))}
					} else if d3.Loc == LocStackTriple {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else if d3.Loc == LocStackPair {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d6)
					ctx.ProtectReg(d6.Reg)
					ctx.EnsureDesc(&d5)
					ctx.UnprotectReg(d6.Reg)
					var d7 JITValueDesc
					if d6.Loc == LocImm && d5.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() * d5.Imm.Int())}
					} else if d6.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d6.Imm.Int()))
						ctx.EmitImulInt64(scratch, d5.Reg)
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d7)
					} else if d5.Loc == LocImm {
						if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d6.Reg, int32(d5.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
							ctx.EmitImulInt64(d6.Reg, RegR11)
						}
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
						ctx.BindReg(d6.Reg, &d7)
					} else {
						ctx.EmitImulInt64(d6.Reg, d5.Reg)
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
						ctx.BindReg(d6.Reg, &d7)
					}
					if d7.Loc == LocReg && d6.Loc == LocReg && d7.Reg == d6.Reg {
						ctx.TransferReg(d6.Reg)
						d6.Loc = LocNone
					}
					ctx.FreeDesc(&d5)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					callResults8 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d7, d7}, []uint8{3}, []uint8{1})
					d9 = callResults8[0]
					d9.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.FreeDesc(&d7)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps10 := PhiState{General: ps.General}
					ps10.OverlayValues = make([]JITValueDesc, 10)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[7] = d7
					ps10.OverlayValues[9] = d9
					ps10.PhiValues = make([]JITValueDesc, 1)
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps10.PhiValues[0] = d11
					if ps10.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps10)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d12 := ps.PhiValues[0]
							ctx.EnsureDesc(&d12)
							ctx.EmitStoreToStack(d12, int32(bbs[1].PhiBase)+int32(0))
						}
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					var d13 JITValueDesc
					if d3.SliceSizeKnown {
						d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d3.Imm.String())))}
					} else if d3.Loc == LocStackTriple {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else if d3.Loc == LocStackPair {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d1.Loc == LocImm && d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d13.Imm.Int())}
					} else if d13.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d1.Reg)
						if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d13.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r0, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d14)
					} else if d1.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d13.Reg)
						ctx.EmitSetcc(r1, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					} else {
						r2 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d13.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d14)
					}
					ctx.FreeDesc(&d13)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							if ps.General {
							}
							ps16 := PhiState{General: ps.General}
							ps16.OverlayValues = make([]JITValueDesc, 16)
							ps16.OverlayValues[1] = d1
							ps16.OverlayValues[2] = d2
							ps16.OverlayValues[3] = d3
							ps16.OverlayValues[4] = d4
							ps16.OverlayValues[5] = d5
							ps16.OverlayValues[6] = d6
							ps16.OverlayValues[7] = d7
							ps16.OverlayValues[9] = d9
							ps16.OverlayValues[11] = d11
							ps16.OverlayValues[12] = d12
							ps16.OverlayValues[13] = d13
							ps16.OverlayValues[14] = d14
							ps16.OverlayValues[15] = d15
							return bbs[2].RenderPS(ps16)
						}
						if ps.General {
						}
						ps17 := PhiState{General: ps.General}
						ps17.OverlayValues = make([]JITValueDesc, 16)
						ps17.OverlayValues[1] = d1
						ps17.OverlayValues[2] = d2
						ps17.OverlayValues[3] = d3
						ps17.OverlayValues[4] = d4
						ps17.OverlayValues[5] = d5
						ps17.OverlayValues[6] = d6
						ps17.OverlayValues[7] = d7
						ps17.OverlayValues[9] = d9
						ps17.OverlayValues[11] = d11
						ps17.OverlayValues[12] = d12
						ps17.OverlayValues[13] = d13
						ps17.OverlayValues[14] = d14
						ps17.OverlayValues[15] = d15
						return bbs[3].RenderPS(ps17)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d18 := ps.PhiValues[0]
							ctx.EnsureDesc(&d18)
							ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 19)
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[7] = d7
					ps19.OverlayValues[9] = d9
					ps19.OverlayValues[11] = d11
					ps19.OverlayValues[12] = d12
					ps19.OverlayValues[13] = d13
					ps19.OverlayValues[14] = d14
					ps19.OverlayValues[15] = d15
					ps19.OverlayValues[18] = d18
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 19)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[18] = d18
					snap21 := d1
					snap22 := d2
					snap23 := d3
					snap24 := d4
					snap25 := d5
					snap26 := d6
					snap27 := d7
					snap28 := d9
					snap29 := d11
					snap30 := d12
					snap31 := d13
					snap32 := d14
					snap33 := d15
					snap34 := d18
					alloc35 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc35)
					d1 = snap21
					d2 = snap22
					d3 = snap23
					d4 = snap24
					d5 = snap25
					d6 = snap26
					d7 = snap27
					d9 = snap28
					d11 = snap29
					d12 = snap30
					d13 = snap31
					d14 = snap32
					d15 = snap33
					d18 = snap34
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps19)
					}
					return result
					ctx.FreeDesc(&d14)
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs36 := make([]Reg, 0, 3)
					seenBlockPinnedRegs37 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs37
					for _, r := range []Reg{d9.Reg, d9.Reg2, d9.Reg3} {
						live := d9.Loc == LocRegTriple && (r == d9.Reg || r == d9.Reg2 || r == d9.Reg3)
						if live && !seenBlockPinnedRegs37[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs37[r] = true
							blockPinnedRegs36 = append(blockPinnedRegs36, r)
						}
					}
					unpinBlockRegs38 := func() {
						for _, r := range blockPinnedRegs36 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs38()
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d39)
					ctx.ProtectReg(d39.Reg)
					ctx.EnsureDesc(&d1)
					ctx.UnprotectReg(d39.Reg)
					var d40 JITValueDesc
					if d39.Loc == LocImm && d1.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d39.Imm.Int() * d1.Imm.Int())}
					} else if d39.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d40)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d39.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d39.Reg, RegR11)
						}
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg}
						ctx.BindReg(d39.Reg, &d40)
					} else {
						ctx.EmitImulInt64(d39.Reg, d1.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg}
						ctx.BindReg(d39.Reg, &d40)
					}
					if d40.Loc == LocReg && d39.Loc == LocReg && d40.Reg == d39.Reg {
						ctx.TransferReg(d39.Reg)
						d39.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d41 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d41)
					r3 := ctx.AllocRegExcept(d41.Reg)
					ctx.EmitMovRegMemB(r3, d41.Reg, 0)
					ctx.FreeDesc(&d41)
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, NoHeapPointer: true}
					ctx.BindReg(r3, &d42)
					ctx.BindReg(r3, &d42)
					ctx.EnsureDesc(&d42)
					var d43 JITValueDesc
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d42.Reg, 4)
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
						ctx.BindReg(d42.Reg, &d43)
					}
					if d43.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: d43.Type, Imm: NewInt(int64(uint64(d43.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d43.Reg, 56)
						ctx.EmitShrRegImm8(d43.Reg, 56)
					}
					if d43.Loc == LocReg && d42.Loc == LocReg && d43.Reg == d42.Reg {
						ctx.TransferReg(d42.Reg)
						d42.Loc = LocNone
					}
					ctx.FreeDesc(&d42)
					d44 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d43)
					ctx.EnsureGoStringHeader(&d44)
					d45 = ctx.EmitSliceElementAddress(&d44, &d43, 1)
					ctx.EnsureDesc(&d45)
					r4 := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegMemB(r4, d45.Reg, 0)
					ctx.FreeDesc(&d45)
					d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4, NoHeapPointer: true}
					ctx.BindReg(r4, &d46)
					ctx.BindReg(r4, &d46)
					ctx.FreeDesc(&d43)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d46)
					d47 = ctx.EmitSliceElementAddress(&d9, &d40, int32(1))
					ctx.EmitStoreScmerAt(&d47, &d46)
					ctx.FreeDesc(&d47)
					ctx.FreeDesc(&d40)
					ctx.FreeDesc(&d46)
					d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d48)
					ctx.ProtectReg(d48.Reg)
					ctx.EnsureDesc(&d1)
					ctx.UnprotectReg(d48.Reg)
					var d49 JITValueDesc
					if d48.Loc == LocImm && d1.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d48.Imm.Int() * d1.Imm.Int())}
					} else if d48.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d48.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d48.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d48.Reg, RegR11)
						}
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
						ctx.BindReg(d48.Reg, &d49)
					} else {
						ctx.EmitImulInt64(d48.Reg, d1.Reg)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
						ctx.BindReg(d48.Reg, &d49)
					}
					if d49.Loc == LocReg && d48.Loc == LocReg && d49.Reg == d48.Reg {
						ctx.TransferReg(d48.Reg)
						d48.Loc = LocNone
					}
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d49)
					var d50 JITValueDesc
					if d49.Loc == LocImm {
						d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegReg(scratch, d49.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d50)
					}
					if d50.Loc == LocReg && d49.Loc == LocReg && d50.Reg == d49.Reg {
						ctx.TransferReg(d49.Reg)
						d49.Loc = LocNone
					}
					ctx.FreeDesc(&d49)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d51 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d51)
					r5 := ctx.AllocRegExcept(d51.Reg)
					ctx.EmitMovRegMemB(r5, d51.Reg, 0)
					ctx.FreeDesc(&d51)
					d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, NoHeapPointer: true}
					ctx.BindReg(r5, &d52)
					ctx.BindReg(r5, &d52)
					ctx.EnsureDesc(&d52)
					var d53 JITValueDesc
					if d52.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d52.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d52.Reg, 15)
						d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d52.Reg}
						ctx.BindReg(d52.Reg, &d53)
					}
					if d53.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: d53.Type, Imm: NewInt(int64(uint64(d53.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d53.Reg, 56)
						ctx.EmitShrRegImm8(d53.Reg, 56)
					}
					if d53.Loc == LocReg && d52.Loc == LocReg && d53.Reg == d52.Reg {
						ctx.TransferReg(d52.Reg)
						d52.Loc = LocNone
					}
					ctx.FreeDesc(&d52)
					d54 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d53)
					ctx.EnsureGoStringHeader(&d54)
					d55 = ctx.EmitSliceElementAddress(&d54, &d53, 1)
					ctx.EnsureDesc(&d55)
					r6 := ctx.AllocRegExcept(d55.Reg)
					ctx.EmitMovRegMemB(r6, d55.Reg, 0)
					ctx.FreeDesc(&d55)
					d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d56)
					ctx.BindReg(r6, &d56)
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d56)
					d57 = ctx.EmitSliceElementAddress(&d9, &d50, int32(1))
					ctx.EmitStoreScmerAt(&d57, &d56)
					ctx.FreeDesc(&d57)
					ctx.FreeDesc(&d50)
					ctx.FreeDesc(&d56)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d58 JITValueDesc
					if d1.Loc == LocImm {
						d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d58)
					}
					if d58.Loc == LocReg && d1.Loc == LocReg && d58.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d58)
					ctx.EmitStoreToStack(d58, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d58)
					ctx.FreeDesc(&d1)
					if ps.General {
					}
					ps59 := PhiState{General: ps.General}
					ps59.OverlayValues = make([]JITValueDesc, 59)
					ps59.OverlayValues[1] = d1
					ps59.OverlayValues[2] = d2
					ps59.OverlayValues[3] = d3
					ps59.OverlayValues[4] = d4
					ps59.OverlayValues[5] = d5
					ps59.OverlayValues[6] = d6
					ps59.OverlayValues[7] = d7
					ps59.OverlayValues[9] = d9
					ps59.OverlayValues[11] = d11
					ps59.OverlayValues[12] = d12
					ps59.OverlayValues[13] = d13
					ps59.OverlayValues[14] = d14
					ps59.OverlayValues[15] = d15
					ps59.OverlayValues[18] = d18
					ps59.OverlayValues[39] = d39
					ps59.OverlayValues[40] = d40
					ps59.OverlayValues[41] = d41
					ps59.OverlayValues[42] = d42
					ps59.OverlayValues[43] = d43
					ps59.OverlayValues[44] = d44
					ps59.OverlayValues[45] = d45
					ps59.OverlayValues[46] = d46
					ps59.OverlayValues[47] = d47
					ps59.OverlayValues[48] = d48
					ps59.OverlayValues[49] = d49
					ps59.OverlayValues[50] = d50
					ps59.OverlayValues[51] = d51
					ps59.OverlayValues[52] = d52
					ps59.OverlayValues[53] = d53
					ps59.OverlayValues[54] = d54
					ps59.OverlayValues[55] = d55
					ps59.OverlayValues[56] = d56
					ps59.OverlayValues[57] = d57
					ps59.OverlayValues[58] = d58
					ps59.PhiValues = make([]JITValueDesc, 1)
					if ps59.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps59)
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
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
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs60 := make([]Reg, 0, 3)
					seenBlockPinnedRegs61 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs61
					for _, r := range []Reg{d9.Reg, d9.Reg2, d9.Reg3} {
						live := d9.Loc == LocRegTriple && (r == d9.Reg || r == d9.Reg2 || r == d9.Reg3)
						if live && !seenBlockPinnedRegs61[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs61[r] = true
							blockPinnedRegs60 = append(blockPinnedRegs60, r)
						}
					}
					unpinBlockRegs62 := func() {
						for _, r := range blockPinnedRegs60 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs62()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					callResults64 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d9}, []uint8{2}, []uint8{1})
					d63 = callResults64[0]
					ctx.EnsureDesc(&d63)
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d63}, 2)
					ctx.EmitMovPairToResult(&d65, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps66 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps66)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  29,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["bin2hex"].Fn, args, result)
				}
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
				var d9 JITValueDesc
				_ = d9
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
				var d18 JITValueDesc
				_ = d18
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
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d63 JITValueDesc
				_ = d63
				var d65 JITValueDesc
				_ = d65
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d4 = d2
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d4.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d4)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d4)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d4)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d4.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d4 = tmpPair
					} else if d4.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
						switch d4.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d4)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d4)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d4)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d4)
						d4 = tmpPair
					} else if d4.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d4.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d4.MemPtr))
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
						d4 = tmpPair
					}
					if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d3 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d4}, 2)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					var d5 JITValueDesc
					if d3.SliceSizeKnown {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d3.Imm.String())))}
					} else if d3.Loc == LocStackTriple {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else if d3.Loc == LocStackPair {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d6)
					ctx.ProtectReg(d6.Reg)
					ctx.EnsureDesc(&d5)
					ctx.UnprotectReg(d6.Reg)
					var d7 JITValueDesc
					if d6.Loc == LocImm && d5.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() * d5.Imm.Int())}
					} else if d6.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d6.Imm.Int()))
						ctx.EmitImulInt64(scratch, d5.Reg)
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d7)
					} else if d5.Loc == LocImm {
						if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d6.Reg, int32(d5.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
							ctx.EmitImulInt64(d6.Reg, RegR11)
						}
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
						ctx.BindReg(d6.Reg, &d7)
					} else {
						ctx.EmitImulInt64(d6.Reg, d5.Reg)
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
						ctx.BindReg(d6.Reg, &d7)
					}
					if d7.Loc == LocReg && d6.Loc == LocReg && d7.Reg == d6.Reg {
						ctx.TransferReg(d6.Reg)
						d6.Loc = LocNone
					}
					ctx.FreeDesc(&d5)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					callResults8 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d7, d7}, []uint8{3}, []uint8{1})
					d9 = callResults8[0]
					d9.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.FreeDesc(&d7)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps10 := PhiState{General: ps.General}
					ps10.OverlayValues = make([]JITValueDesc, 10)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[7] = d7
					ps10.OverlayValues[9] = d9
					ps10.PhiValues = make([]JITValueDesc, 1)
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps10.PhiValues[0] = d11
					if ps10.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps10)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d12 := ps.PhiValues[0]
							ctx.EnsureDesc(&d12)
							ctx.EmitStoreToStack(d12, int32(bbs[1].PhiBase)+int32(0))
						}
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					var d13 JITValueDesc
					if d3.SliceSizeKnown {
						d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d3.Imm.String())))}
					} else if d3.Loc == LocStackTriple {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else if d3.Loc == LocStackPair {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d1.Loc == LocImm && d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d13.Imm.Int())}
					} else if d13.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d1.Reg)
						if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d13.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r0, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d14)
					} else if d1.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d13.Reg)
						ctx.EmitSetcc(r1, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					} else {
						r2 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d13.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d14)
					}
					ctx.FreeDesc(&d13)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							if ps.General {
							}
							ps16 := PhiState{General: ps.General}
							ps16.OverlayValues = make([]JITValueDesc, 16)
							ps16.OverlayValues[1] = d1
							ps16.OverlayValues[2] = d2
							ps16.OverlayValues[3] = d3
							ps16.OverlayValues[4] = d4
							ps16.OverlayValues[5] = d5
							ps16.OverlayValues[6] = d6
							ps16.OverlayValues[7] = d7
							ps16.OverlayValues[9] = d9
							ps16.OverlayValues[11] = d11
							ps16.OverlayValues[12] = d12
							ps16.OverlayValues[13] = d13
							ps16.OverlayValues[14] = d14
							ps16.OverlayValues[15] = d15
							return bbs[2].RenderPS(ps16)
						}
						if ps.General {
						}
						ps17 := PhiState{General: ps.General}
						ps17.OverlayValues = make([]JITValueDesc, 16)
						ps17.OverlayValues[1] = d1
						ps17.OverlayValues[2] = d2
						ps17.OverlayValues[3] = d3
						ps17.OverlayValues[4] = d4
						ps17.OverlayValues[5] = d5
						ps17.OverlayValues[6] = d6
						ps17.OverlayValues[7] = d7
						ps17.OverlayValues[9] = d9
						ps17.OverlayValues[11] = d11
						ps17.OverlayValues[12] = d12
						ps17.OverlayValues[13] = d13
						ps17.OverlayValues[14] = d14
						ps17.OverlayValues[15] = d15
						return bbs[3].RenderPS(ps17)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d18 := ps.PhiValues[0]
							ctx.EnsureDesc(&d18)
							ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 19)
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[7] = d7
					ps19.OverlayValues[9] = d9
					ps19.OverlayValues[11] = d11
					ps19.OverlayValues[12] = d12
					ps19.OverlayValues[13] = d13
					ps19.OverlayValues[14] = d14
					ps19.OverlayValues[15] = d15
					ps19.OverlayValues[18] = d18
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 19)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[18] = d18
					snap21 := d1
					snap22 := d2
					snap23 := d3
					snap24 := d4
					snap25 := d5
					snap26 := d6
					snap27 := d7
					snap28 := d9
					snap29 := d11
					snap30 := d12
					snap31 := d13
					snap32 := d14
					snap33 := d15
					snap34 := d18
					alloc35 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc35)
					d1 = snap21
					d2 = snap22
					d3 = snap23
					d4 = snap24
					d5 = snap25
					d6 = snap26
					d7 = snap27
					d9 = snap28
					d11 = snap29
					d12 = snap30
					d13 = snap31
					d14 = snap32
					d15 = snap33
					d18 = snap34
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps19)
					}
					return result
					ctx.FreeDesc(&d14)
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs36 := make([]Reg, 0, 3)
					seenBlockPinnedRegs37 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs37
					for _, r := range []Reg{d9.Reg, d9.Reg2, d9.Reg3} {
						live := d9.Loc == LocRegTriple && (r == d9.Reg || r == d9.Reg2 || r == d9.Reg3)
						if live && !seenBlockPinnedRegs37[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs37[r] = true
							blockPinnedRegs36 = append(blockPinnedRegs36, r)
						}
					}
					unpinBlockRegs38 := func() {
						for _, r := range blockPinnedRegs36 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs38()
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d39)
					ctx.ProtectReg(d39.Reg)
					ctx.EnsureDesc(&d1)
					ctx.UnprotectReg(d39.Reg)
					var d40 JITValueDesc
					if d39.Loc == LocImm && d1.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d39.Imm.Int() * d1.Imm.Int())}
					} else if d39.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d40)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d39.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d39.Reg, RegR11)
						}
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg}
						ctx.BindReg(d39.Reg, &d40)
					} else {
						ctx.EmitImulInt64(d39.Reg, d1.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg}
						ctx.BindReg(d39.Reg, &d40)
					}
					if d40.Loc == LocReg && d39.Loc == LocReg && d40.Reg == d39.Reg {
						ctx.TransferReg(d39.Reg)
						d39.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d41 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d41)
					r3 := ctx.AllocRegExcept(d41.Reg)
					ctx.EmitMovRegMemB(r3, d41.Reg, 0)
					ctx.FreeDesc(&d41)
					d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, NoHeapPointer: true}
					ctx.BindReg(r3, &d42)
					ctx.BindReg(r3, &d42)
					ctx.EnsureDesc(&d42)
					var d43 JITValueDesc
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d42.Reg, 4)
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
						ctx.BindReg(d42.Reg, &d43)
					}
					if d43.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: d43.Type, Imm: NewInt(int64(uint64(d43.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d43.Reg, 56)
						ctx.EmitShrRegImm8(d43.Reg, 56)
					}
					if d43.Loc == LocReg && d42.Loc == LocReg && d43.Reg == d42.Reg {
						ctx.TransferReg(d42.Reg)
						d42.Loc = LocNone
					}
					ctx.FreeDesc(&d42)
					d44 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d43)
					ctx.EnsureGoStringHeader(&d44)
					d45 = ctx.EmitSliceElementAddress(&d44, &d43, 1)
					ctx.EnsureDesc(&d45)
					r4 := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegMemB(r4, d45.Reg, 0)
					ctx.FreeDesc(&d45)
					d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4, NoHeapPointer: true}
					ctx.BindReg(r4, &d46)
					ctx.BindReg(r4, &d46)
					ctx.FreeDesc(&d43)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d46)
					d47 = ctx.EmitSliceElementAddress(&d9, &d40, int32(1))
					ctx.EmitStoreScmerAt(&d47, &d46)
					ctx.FreeDesc(&d47)
					ctx.FreeDesc(&d40)
					ctx.FreeDesc(&d46)
					d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d48)
					ctx.ProtectReg(d48.Reg)
					ctx.EnsureDesc(&d1)
					ctx.UnprotectReg(d48.Reg)
					var d49 JITValueDesc
					if d48.Loc == LocImm && d1.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d48.Imm.Int() * d1.Imm.Int())}
					} else if d48.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d48.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d48.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d48.Reg, RegR11)
						}
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
						ctx.BindReg(d48.Reg, &d49)
					} else {
						ctx.EmitImulInt64(d48.Reg, d1.Reg)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
						ctx.BindReg(d48.Reg, &d49)
					}
					if d49.Loc == LocReg && d48.Loc == LocReg && d49.Reg == d48.Reg {
						ctx.TransferReg(d48.Reg)
						d48.Loc = LocNone
					}
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d49)
					var d50 JITValueDesc
					if d49.Loc == LocImm {
						d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegReg(scratch, d49.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d50)
					}
					if d50.Loc == LocReg && d49.Loc == LocReg && d50.Reg == d49.Reg {
						ctx.TransferReg(d49.Reg)
						d49.Loc = LocNone
					}
					ctx.FreeDesc(&d49)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d51 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d51)
					r5 := ctx.AllocRegExcept(d51.Reg)
					ctx.EmitMovRegMemB(r5, d51.Reg, 0)
					ctx.FreeDesc(&d51)
					d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, NoHeapPointer: true}
					ctx.BindReg(r5, &d52)
					ctx.BindReg(r5, &d52)
					ctx.EnsureDesc(&d52)
					var d53 JITValueDesc
					if d52.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d52.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d52.Reg, 15)
						d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d52.Reg}
						ctx.BindReg(d52.Reg, &d53)
					}
					if d53.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: d53.Type, Imm: NewInt(int64(uint64(d53.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d53.Reg, 56)
						ctx.EmitShrRegImm8(d53.Reg, 56)
					}
					if d53.Loc == LocReg && d52.Loc == LocReg && d53.Reg == d52.Reg {
						ctx.TransferReg(d52.Reg)
						d52.Loc = LocNone
					}
					ctx.FreeDesc(&d52)
					d54 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d53)
					ctx.EnsureGoStringHeader(&d54)
					d55 = ctx.EmitSliceElementAddress(&d54, &d53, 1)
					ctx.EnsureDesc(&d55)
					r6 := ctx.AllocRegExcept(d55.Reg)
					ctx.EmitMovRegMemB(r6, d55.Reg, 0)
					ctx.FreeDesc(&d55)
					d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d56)
					ctx.BindReg(r6, &d56)
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d56)
					d57 = ctx.EmitSliceElementAddress(&d9, &d50, int32(1))
					ctx.EmitStoreScmerAt(&d57, &d56)
					ctx.FreeDesc(&d57)
					ctx.FreeDesc(&d50)
					ctx.FreeDesc(&d56)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d58 JITValueDesc
					if d1.Loc == LocImm {
						d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d58)
					}
					if d58.Loc == LocReg && d1.Loc == LocReg && d58.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d58)
					ctx.EmitStoreToStack(d58, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d58)
					ctx.FreeDesc(&d1)
					if ps.General {
					}
					ps59 := PhiState{General: ps.General}
					ps59.OverlayValues = make([]JITValueDesc, 59)
					ps59.OverlayValues[1] = d1
					ps59.OverlayValues[2] = d2
					ps59.OverlayValues[3] = d3
					ps59.OverlayValues[4] = d4
					ps59.OverlayValues[5] = d5
					ps59.OverlayValues[6] = d6
					ps59.OverlayValues[7] = d7
					ps59.OverlayValues[9] = d9
					ps59.OverlayValues[11] = d11
					ps59.OverlayValues[12] = d12
					ps59.OverlayValues[13] = d13
					ps59.OverlayValues[14] = d14
					ps59.OverlayValues[15] = d15
					ps59.OverlayValues[18] = d18
					ps59.OverlayValues[39] = d39
					ps59.OverlayValues[40] = d40
					ps59.OverlayValues[41] = d41
					ps59.OverlayValues[42] = d42
					ps59.OverlayValues[43] = d43
					ps59.OverlayValues[44] = d44
					ps59.OverlayValues[45] = d45
					ps59.OverlayValues[46] = d46
					ps59.OverlayValues[47] = d47
					ps59.OverlayValues[48] = d48
					ps59.OverlayValues[49] = d49
					ps59.OverlayValues[50] = d50
					ps59.OverlayValues[51] = d51
					ps59.OverlayValues[52] = d52
					ps59.OverlayValues[53] = d53
					ps59.OverlayValues[54] = d54
					ps59.OverlayValues[55] = d55
					ps59.OverlayValues[56] = d56
					ps59.OverlayValues[57] = d57
					ps59.OverlayValues[58] = d58
					ps59.PhiValues = make([]JITValueDesc, 1)
					if ps59.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps59)
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
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
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs60 := make([]Reg, 0, 3)
					seenBlockPinnedRegs61 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs61
					for _, r := range []Reg{d9.Reg, d9.Reg2, d9.Reg3} {
						live := d9.Loc == LocRegTriple && (r == d9.Reg || r == d9.Reg2 || r == d9.Reg3)
						if live && !seenBlockPinnedRegs61[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs61[r] = true
							blockPinnedRegs60 = append(blockPinnedRegs60, r)
						}
					}
					unpinBlockRegs62 := func() {
						for _, r := range blockPinnedRegs60 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs62()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					callResults64 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d9}, []uint8{2}, []uint8{1})
					d63 = callResults64[0]
					ctx.EnsureDesc(&d63)
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d63}, 2)
					ctx.EmitMovPairToResult(&d65, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps66 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps66)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  29,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["hex2bin"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d20 JITValueDesc
				_ = d20
				var d22 JITValueDesc
				_ = d22
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (hex.DecodeString arg0)")
					}
					ctx.SyncDesc(&d1)
					callResults3 := JITEmitGoCallResults(ctx, GoFuncAddr(hex.DecodeString), []JITValueDesc{d1}, []uint8{3, 2}, []uint8{1, 3})
					d4 = callResults3[0]
					_ = d4
					d5 = callResults3[1]
					_ = d5
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.EnsureDesc(&d5)
					var d6 JITValueDesc
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc != LocReg && d5.Loc != LocRegPair && d5.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d6)
					}
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							if ps.General {
							}
							ps8 := PhiState{General: ps.General}
							ps8.OverlayValues = make([]JITValueDesc, 8)
							ps8.OverlayValues[0] = d0
							ps8.OverlayValues[1] = d1
							ps8.OverlayValues[2] = d2
							ps8.OverlayValues[4] = d4
							ps8.OverlayValues[5] = d5
							ps8.OverlayValues[6] = d6
							ps8.OverlayValues[7] = d7
							return bbs[1].RenderPS(ps8)
						}
						if ps.General {
						}
						ps9 := PhiState{General: ps.General}
						ps9.OverlayValues = make([]JITValueDesc, 8)
						ps9.OverlayValues[0] = d0
						ps9.OverlayValues[1] = d1
						ps9.OverlayValues[2] = d2
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
					ctx.EmitJump(CondNotEqual, lbl4)
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
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[7] = d7
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 8)
					ps11.OverlayValues[0] = d0
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[4] = d4
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[6] = d6
					ps11.OverlayValues[7] = d7
					snap12 := d0
					snap13 := d1
					snap14 := d2
					snap15 := d4
					snap16 := d5
					snap17 := d6
					snap18 := d7
					alloc19 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps11)
					}
					ctx.RestoreAllocState(alloc19)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					d7 = snap18
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["hex2bin"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					callResults21 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d4}, []uint8{2}, []uint8{1})
					d20 = callResults21[0]
					ctx.EnsureDesc(&d20)
					d22 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d20}, 2)
					ctx.EmitMovPairToResult(&d22, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps23 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps23)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  20,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["uuid"].Fn, args, result)
				}
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
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
					callResults0 := JITEmitGoCallResults(ctx, GoFuncAddr(uuid.NewRandom), []JITValueDesc{}, []uint8{2, 2}, []uint8{0, 3})
					d1 = callResults0[0]
					_ = d1
					d2 = callResults0[1]
					_ = d2
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d2)
						if d2.Loc != LocReg && d2.Loc != LocRegPair && d2.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d3)
					}
					d4 = d3
					ctx.EnsureDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							if ps.General {
							}
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						if ps.General {
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						return bbs[2].RenderPS(ps6)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d4.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 5)
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					ps7.OverlayValues[4] = d4
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 5)
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[2] = d2
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					alloc13 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps8)
					}
					ctx.RestoreAllocState(alloc13)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps7)
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["uuid"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					d15 = d1
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
					ctx.EnsureDesc(&d14)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d14}, 2)
					ctx.EmitMovPairToResult(&d16, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps17 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps17)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  17,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
				}
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d64 JITValueDesc
				_ = d64
				var d66 JITValueDesc
				_ = d66
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					ctx.StabilizeDescForControlFlow(&d1)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d2 JITValueDesc
					if d1.Loc == LocImm {
						d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int())}
					} else if d1.Type == tagInt && d1.Loc == LocRegPair {
						ctx.FreeReg(d1.Reg)
						d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d2)
						ctx.BindReg(d1.Reg2, &d2)
					} else if d1.Type == tagInt && d1.Loc == LocReg {
						d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
						ctx.BindReg(d1.Reg, &d2)
						ctx.BindReg(d1.Reg, &d2)
					} else {
						d2 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d1}, 1)
						d2.Type = tagInt
						ctx.BindReg(d2.Reg, &d2)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.FreeDesc(&d0)
					ctx.EnsureDesc(&d2)
					var d4 JITValueDesc
					if d2.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r0, CondSignedLess)
						d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d4)
					}
					d5 = d4
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
							ps6.OverlayValues[0] = d0
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
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					callResults17 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d2, d2}, []uint8{3}, []uint8{1})
					d18 = callResults17[0]
					d18.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.EnsureDesc(&d2)
					var d19 JITValueDesc
					if d2.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() > 0)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r1, CondSignedGreater)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d19)
					}
					ctx.FreeDesc(&d2)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[0] = d0
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[2] = d2
							ps21.OverlayValues[3] = d3
							ps21.OverlayValues[4] = d4
							ps21.OverlayValues[5] = d5
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[3].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[0] = d0
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[2] = d2
						ps22.OverlayValues[3] = d3
						ps22.OverlayValues[4] = d4
						ps22.OverlayValues[5] = d5
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[4].RenderPS(ps22)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl5)
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 21)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[19] = d19
					ps23.OverlayValues[20] = d20
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 21)
					ps24.OverlayValues[0] = d0
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					snap25 := d0
					snap26 := d1
					snap27 := d2
					snap28 := d3
					snap29 := d4
					snap30 := d5
					snap31 := d18
					snap32 := d19
					snap33 := d20
					alloc34 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps24)
					}
					ctx.RestoreAllocState(alloc34)
					d0 = snap25
					d1 = snap26
					d2 = snap27
					d3 = snap28
					d4 = snap29
					d5 = snap30
					d18 = snap31
					d19 = snap32
					d20 = snap33
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps23)
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
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
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
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs35 := make([]Reg, 0, 3)
					seenBlockPinnedRegs36 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs36
					for _, r := range []Reg{d18.Reg, d18.Reg2, d18.Reg3} {
						live := d18.Loc == LocRegTriple && (r == d18.Reg || r == d18.Reg2 || r == d18.Reg3)
						if live && !seenBlockPinnedRegs36[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs36[r] = true
							blockPinnedRegs35 = append(blockPinnedRegs35, r)
						}
					}
					unpinBlockRegs37 := func() {
						for _, r := range blockPinnedRegs35 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs37()
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocRegTriple && d18.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (crand.Read arg0)")
					}
					ctx.SyncDesc(&d18)
					callResults38 := JITEmitGoCallResults(ctx, GoFuncAddr(crand.Read), []JITValueDesc{d18}, []uint8{1, 2}, []uint8{0, 3})
					d39 = callResults38[0]
					_ = d39
					d40 = callResults38[1]
					_ = d40
					ctx.StabilizeDescForControlFlow(&d40)
					ctx.EnsureDesc(&d40)
					var d41 JITValueDesc
					if d40.Loc == LocImm {
						d41 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d40.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d40)
						if d40.Loc != LocReg && d40.Loc != LocRegPair && d40.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d40.Reg)
						ctx.EmitCmpRegImm32(d40.Reg, 0)
						ctx.EmitSetcc(r2, CondNotEqual)
						d41 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d41)
					}
					d42 = d41
					ctx.EnsureDesc(&d42)
					if d42.Loc != LocImm && d42.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d42.Loc == LocImm {
						if d42.Imm.Bool() {
							if ps.General {
							}
							ps43 := PhiState{General: ps.General}
							ps43.OverlayValues = make([]JITValueDesc, 43)
							ps43.OverlayValues[0] = d0
							ps43.OverlayValues[1] = d1
							ps43.OverlayValues[2] = d2
							ps43.OverlayValues[3] = d3
							ps43.OverlayValues[4] = d4
							ps43.OverlayValues[5] = d5
							ps43.OverlayValues[18] = d18
							ps43.OverlayValues[19] = d19
							ps43.OverlayValues[20] = d20
							ps43.OverlayValues[39] = d39
							ps43.OverlayValues[40] = d40
							ps43.OverlayValues[41] = d41
							ps43.OverlayValues[42] = d42
							return bbs[5].RenderPS(ps43)
						}
						if ps.General {
						}
						ps44 := PhiState{General: ps.General}
						ps44.OverlayValues = make([]JITValueDesc, 43)
						ps44.OverlayValues[0] = d0
						ps44.OverlayValues[1] = d1
						ps44.OverlayValues[2] = d2
						ps44.OverlayValues[3] = d3
						ps44.OverlayValues[4] = d4
						ps44.OverlayValues[5] = d5
						ps44.OverlayValues[18] = d18
						ps44.OverlayValues[19] = d19
						ps44.OverlayValues[20] = d20
						ps44.OverlayValues[39] = d39
						ps44.OverlayValues[40] = d40
						ps44.OverlayValues[41] = d41
						ps44.OverlayValues[42] = d42
						return bbs[4].RenderPS(ps44)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d42.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl5)
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 43)
					ps45.OverlayValues[0] = d0
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[19] = d19
					ps45.OverlayValues[20] = d20
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
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[19] = d19
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[39] = d39
					ps46.OverlayValues[40] = d40
					ps46.OverlayValues[41] = d41
					ps46.OverlayValues[42] = d42
					snap47 := d0
					snap48 := d1
					snap49 := d2
					snap50 := d3
					snap51 := d4
					snap52 := d5
					snap53 := d18
					snap54 := d19
					snap55 := d20
					snap56 := d39
					snap57 := d40
					snap58 := d41
					snap59 := d42
					alloc60 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps46)
					}
					ctx.RestoreAllocState(alloc60)
					d0 = snap47
					d1 = snap48
					d2 = snap49
					d3 = snap50
					d4 = snap51
					d5 = snap52
					d18 = snap53
					d19 = snap54
					d20 = snap55
					d39 = snap56
					d40 = snap57
					d41 = snap58
					d42 = snap59
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps45)
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
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
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
					blockPinnedRegs61 := make([]Reg, 0, 3)
					seenBlockPinnedRegs62 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs62
					for _, r := range []Reg{d18.Reg, d18.Reg2, d18.Reg3} {
						live := d18.Loc == LocRegTriple && (r == d18.Reg || r == d18.Reg2 || r == d18.Reg3)
						if live && !seenBlockPinnedRegs62[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs62[r] = true
							blockPinnedRegs61 = append(blockPinnedRegs61, r)
						}
					}
					unpinBlockRegs63 := func() {
						for _, r := range blockPinnedRegs61 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs63()
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					callResults65 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d18}, []uint8{2}, []uint8{1})
					d64 = callResults65[0]
					ctx.EnsureDesc(&d64)
					d66 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d64}, 2)
					ctx.EmitMovPairToResult(&d66, &result)
					result.Type = tagString
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
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
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
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps67 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps67)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  30,
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
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "regex pattern"}, &TypeDescriptor{Kind: "string", Label: "replacement", Description: "replacement string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_replace"].Fn, args, result)
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
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
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
						return bbs[2].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
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
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d16 = d14
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d16.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d16.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					} else if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
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
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (regexp.Compile arg0)")
					}
					ctx.SyncDesc(&d15)
					callResults17 := JITEmitGoCallResults(ctx, GoFuncAddr(regexp.Compile), []JITValueDesc{d15}, []uint8{1, 2}, []uint8{1, 3})
					d18 = callResults17[0]
					_ = d18
					d19 = callResults17[1]
					_ = d19
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.EnsureDesc(&d19)
					var d20 JITValueDesc
					if d19.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc != LocReg && d19.Loc != LocRegPair && d19.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d20)
					}
					d21 = d20
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
							ps22.OverlayValues[3] = d3
							ps22.OverlayValues[13] = d13
							ps22.OverlayValues[14] = d14
							ps22.OverlayValues[15] = d15
							ps22.OverlayValues[16] = d16
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
						ps23.OverlayValues[3] = d3
						ps23.OverlayValues[13] = d13
						ps23.OverlayValues[14] = d14
						ps23.OverlayValues[15] = d15
						ps23.OverlayValues[16] = d16
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
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 22)
					ps24.OverlayValues[0] = d0
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[13] = d13
					ps24.OverlayValues[14] = d14
					ps24.OverlayValues[15] = d15
					ps24.OverlayValues[16] = d16
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					ps24.OverlayValues[21] = d21
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 22)
					ps25.OverlayValues[0] = d0
					ps25.OverlayValues[1] = d1
					ps25.OverlayValues[2] = d2
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[13] = d13
					ps25.OverlayValues[14] = d14
					ps25.OverlayValues[15] = d15
					ps25.OverlayValues[16] = d16
					ps25.OverlayValues[18] = d18
					ps25.OverlayValues[19] = d19
					ps25.OverlayValues[20] = d20
					ps25.OverlayValues[21] = d21
					snap26 := d0
					snap27 := d1
					snap28 := d2
					snap29 := d3
					snap30 := d13
					snap31 := d14
					snap32 := d15
					snap33 := d16
					snap34 := d18
					snap35 := d19
					snap36 := d20
					snap37 := d21
					alloc38 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps25)
					}
					ctx.RestoreAllocState(alloc38)
					d0 = snap26
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d13 = snap30
					d14 = snap31
					d15 = snap32
					d16 = snap33
					d18 = snap34
					d19 = snap35
					d20 = snap36
					d21 = snap37
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps24)
					}
					return result
					ctx.FreeDesc(&d20)
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_replace"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					d39 = args[0]
					d39.ID = 0
					d41 = d39
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d41.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d41)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d41)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d41)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d41.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d41 = tmpPair
					} else if d41.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d41.Reg), Reg2: ctx.AllocRegExcept(d41.Reg)}
						switch d41.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d41)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d41)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d41)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d41)
						d41 = tmpPair
					} else if d41.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d41.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d41.MemPtr))
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
						d41 = tmpPair
					}
					if d41.Loc != LocRegPair && d41.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d40 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d41}, 2)
					ctx.FreeDesc(&d39)
					d42 = args[2]
					d42.ID = 0
					d44 = d42
					ctx.EnsureDesc(&d44)
					if d44.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d44.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d44)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d44)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d44)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d44.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d44 = tmpPair
					} else if d44.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d44.Reg), Reg2: ctx.AllocRegExcept(d44.Reg)}
						switch d44.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d44)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d44)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d44)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d44)
						d44 = tmpPair
					} else if d44.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d44.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d44.MemPtr))
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
						d44 = tmpPair
					}
					if d44.Loc != LocRegPair && d44.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d44}, 2)
					ctx.FreeDesc(&d42)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d40.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d40.Imm)
						ptrWord, _ := d40.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d40.Imm.String())))
						d40 = tmpPair
					} else if d40.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d40.Type, Reg: ctx.AllocRegExcept(d40.Reg), Reg2: ctx.AllocRegExcept(d40.Reg)}
						switch d40.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d40)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d40)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d40)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d40)
						d40 = tmpPair
					}
					if d40.Loc != LocRegPair && d40.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).ReplaceAllString arg1)")
					}
					ctx.EnsureDesc(&d43)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d43.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d43.Imm)
						ptrWord, _ := d43.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d43.Imm.String())))
						d43 = tmpPair
					} else if d43.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d43.Type, Reg: ctx.AllocRegExcept(d43.Reg), Reg2: ctx.AllocRegExcept(d43.Reg)}
						switch d43.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d43)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d43)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d43)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d43)
						d43 = tmpPair
					}
					if d43.Loc != LocRegPair && d43.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).ReplaceAllString arg2)")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d40)
					ctx.SyncDesc(&d43)
					d45 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).ReplaceAllString), []JITValueDesc{d18, d40, d43}, 2)
					ctx.BindReg(d45.Reg, &d45)
					ctx.BindReg(d45.Reg2, &d45)
					ctx.FreeDesc(&d18)
					ctx.EnsureDesc(&d45)
					d46 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d45}, 2)
					ctx.EmitMovPairToResult(&d46, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps47 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps47)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      27,
		},
		Optimize: optimizeRegexpReplace,
	})

	Declare(&Globalenv, &Declaration{
		Name: "fnv_hash",

		Fn: func(a ...Scmer) Scmer {
			return NewString(fnvHashString(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes a fast non-cryptographic 64-bit FNV-1a hash of a string, returns a 16-character hex string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["fnv_hash"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
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
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (fnvHashString arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(fnvHashString), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d3}, 2)
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 6,
		},
		Optimize: optimizeFNVHash,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["stable_structural_hash"].Fn, args, result)
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
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d65 JITValueDesc
				_ = d65
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [8]BBDescriptor
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
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() < 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 1)
						ctx.EmitSetcc(r0, CondSignedLess)
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
						return bbs[3].RenderPS(ps4)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
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
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps6)
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["stable_structural_hash"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(func() *schemeTextWriter { return new(schemeTextWriter) }), nil, 1)
					ctx.BindReg(d11.Reg, &d11)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d12)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *schemeTextWriter, value uint64) { base.hash = value }), []JITValueDesc{d11, d12})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.StabilizeDescForControlFlow(&d11)
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() == 2)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d13.Reg, 2)
						ctx.EmitSetcc(r1, CondEqual)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					}
					ctx.FreeDesc(&d13)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							if ps.General {
							}
							ps16 := PhiState{General: ps.General}
							ps16.OverlayValues = make([]JITValueDesc, 16)
							ps16.OverlayValues[0] = d0
							ps16.OverlayValues[1] = d1
							ps16.OverlayValues[2] = d2
							ps16.OverlayValues[11] = d11
							ps16.OverlayValues[12] = d12
							ps16.OverlayValues[13] = d13
							ps16.OverlayValues[14] = d14
							ps16.OverlayValues[15] = d15
							return bbs[7].RenderPS(ps16)
						}
						if ps.General {
						}
						ps17 := PhiState{General: ps.General}
						ps17.OverlayValues = make([]JITValueDesc, 16)
						ps17.OverlayValues[0] = d0
						ps17.OverlayValues[1] = d1
						ps17.OverlayValues[2] = d2
						ps17.OverlayValues[11] = d11
						ps17.OverlayValues[12] = d12
						ps17.OverlayValues[13] = d13
						ps17.OverlayValues[14] = d14
						ps17.OverlayValues[15] = d15
						return bbs[6].RenderPS(ps17)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl7)
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 16)
					ps18.OverlayValues[0] = d0
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[11] = d11
					ps18.OverlayValues[12] = d12
					ps18.OverlayValues[13] = d13
					ps18.OverlayValues[14] = d14
					ps18.OverlayValues[15] = d15
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 16)
					ps19.OverlayValues[0] = d0
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[11] = d11
					ps19.OverlayValues[12] = d12
					ps19.OverlayValues[13] = d13
					ps19.OverlayValues[14] = d14
					ps19.OverlayValues[15] = d15
					snap20 := d0
					snap21 := d1
					snap22 := d2
					snap23 := d11
					snap24 := d12
					snap25 := d13
					snap26 := d14
					snap27 := d15
					alloc28 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps19)
					}
					ctx.RestoreAllocState(alloc28)
					d0 = snap20
					d1 = snap21
					d2 = snap22
					d11 = snap23
					d12 = snap24
					d13 = snap25
					d14 = snap26
					d15 = snap27
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps18)
					}
					return result
					ctx.FreeDesc(&d14)
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
					ctx.ReclaimUntrackedRegs()
					d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d29)
					var d30 JITValueDesc
					if d29.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d29.Imm.Int() > 2)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d29.Reg, 2)
						ctx.EmitSetcc(r2, CondSignedGreater)
						d30 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d30)
					}
					ctx.FreeDesc(&d29)
					d31 = d30
					ctx.EnsureDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							if ps.General {
							}
							ps32 := PhiState{General: ps.General}
							ps32.OverlayValues = make([]JITValueDesc, 32)
							ps32.OverlayValues[0] = d0
							ps32.OverlayValues[1] = d1
							ps32.OverlayValues[2] = d2
							ps32.OverlayValues[11] = d11
							ps32.OverlayValues[12] = d12
							ps32.OverlayValues[13] = d13
							ps32.OverlayValues[14] = d14
							ps32.OverlayValues[15] = d15
							ps32.OverlayValues[29] = d29
							ps32.OverlayValues[30] = d30
							ps32.OverlayValues[31] = d31
							return bbs[1].RenderPS(ps32)
						}
						if ps.General {
						}
						ps33 := PhiState{General: ps.General}
						ps33.OverlayValues = make([]JITValueDesc, 32)
						ps33.OverlayValues[0] = d0
						ps33.OverlayValues[1] = d1
						ps33.OverlayValues[2] = d2
						ps33.OverlayValues[11] = d11
						ps33.OverlayValues[12] = d12
						ps33.OverlayValues[13] = d13
						ps33.OverlayValues[14] = d14
						ps33.OverlayValues[15] = d15
						ps33.OverlayValues[29] = d29
						ps33.OverlayValues[30] = d30
						ps33.OverlayValues[31] = d31
						return bbs[2].RenderPS(ps33)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d31.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl3)
					ps34 := PhiState{General: true}
					ps34.OverlayValues = make([]JITValueDesc, 32)
					ps34.OverlayValues[0] = d0
					ps34.OverlayValues[1] = d1
					ps34.OverlayValues[2] = d2
					ps34.OverlayValues[11] = d11
					ps34.OverlayValues[12] = d12
					ps34.OverlayValues[13] = d13
					ps34.OverlayValues[14] = d14
					ps34.OverlayValues[15] = d15
					ps34.OverlayValues[29] = d29
					ps34.OverlayValues[30] = d30
					ps34.OverlayValues[31] = d31
					ps35 := PhiState{General: true}
					ps35.OverlayValues = make([]JITValueDesc, 32)
					ps35.OverlayValues[0] = d0
					ps35.OverlayValues[1] = d1
					ps35.OverlayValues[2] = d2
					ps35.OverlayValues[11] = d11
					ps35.OverlayValues[12] = d12
					ps35.OverlayValues[13] = d13
					ps35.OverlayValues[14] = d14
					ps35.OverlayValues[15] = d15
					ps35.OverlayValues[29] = d29
					ps35.OverlayValues[30] = d30
					ps35.OverlayValues[31] = d31
					snap36 := d0
					snap37 := d1
					snap38 := d2
					snap39 := d11
					snap40 := d12
					snap41 := d13
					snap42 := d14
					snap43 := d15
					snap44 := d29
					snap45 := d30
					snap46 := d31
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps35)
					}
					ctx.RestoreAllocState(alloc47)
					d0 = snap36
					d1 = snap37
					d2 = snap38
					d11 = snap39
					d12 = snap40
					d13 = snap41
					d14 = snap42
					d15 = snap43
					d29 = snap44
					d30 = snap45
					d31 = snap46
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps34)
					}
					return result
					ctx.FreeDesc(&d30)
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs48 := make([]Reg, 0, 3)
					seenBlockPinnedRegs49 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs49
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3)
						if live && !seenBlockPinnedRegs49[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs49[r] = true
							blockPinnedRegs48 = append(blockPinnedRegs48, r)
						}
					}
					unpinBlockRegs50 := func() {
						for _, r := range blockPinnedRegs48 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs50()
					d51 = args[0]
					d51.ID = 0
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d51.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d51.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d51)
						} else if d51.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d51)
						} else if d51.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d51)
						} else if d51.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d51.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
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
					if d51.Loc != LocRegPair && d51.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (serializeEx arg1)")
					}
					d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d52.Loc == LocRegPair || d52.Loc == LocStackPair || d52.Loc == LocRegTriple || d52.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d53.Loc == LocRegPair || d53.Loc == LocStackPair || d53.Loc == LocRegTriple || d53.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d54 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					if d54.Loc == LocRegPair || d54.Loc == LocStackPair || d54.Loc == LocRegTriple || d54.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d51)
					ctx.SyncDesc(&d52)
					ctx.SyncDesc(&d53)
					ctx.SyncDesc(&d54)
					ctx.EmitGoCallVoid(GoFuncAddr(serializeEx), []JITValueDesc{d11, d51, d52, d53, d54})
					ctx.FreeDesc(&d54)
					ctx.FreeDesc(&d51)
					if ps.General {
					}
					ps55 := PhiState{General: ps.General}
					ps55.OverlayValues = make([]JITValueDesc, 55)
					ps55.OverlayValues[0] = d0
					ps55.OverlayValues[1] = d1
					ps55.OverlayValues[2] = d2
					ps55.OverlayValues[11] = d11
					ps55.OverlayValues[12] = d12
					ps55.OverlayValues[13] = d13
					ps55.OverlayValues[14] = d14
					ps55.OverlayValues[15] = d15
					ps55.OverlayValues[29] = d29
					ps55.OverlayValues[30] = d30
					ps55.OverlayValues[31] = d31
					ps55.OverlayValues[51] = d51
					ps55.OverlayValues[52] = d52
					ps55.OverlayValues[53] = d53
					ps55.OverlayValues[54] = d54
					if ps55.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps55)
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs56 := make([]Reg, 0, 3)
					seenBlockPinnedRegs57 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs57
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3)
						if live && !seenBlockPinnedRegs57[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs57[r] = true
							blockPinnedRegs56 = append(blockPinnedRegs56, r)
						}
					}
					unpinBlockRegs58 := func() {
						for _, r := range blockPinnedRegs56 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs58()
					var d59 JITValueDesc
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						fieldAddr := uintptr(d11.Imm.Int()) + 8
						r3 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r3, fieldAddr)
						d59 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d59)
					} else {
						off := int32(8)
						baseReg := d11.Reg
						r4 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r4, baseReg, off)
						d59 = JITValueDesc{Loc: LocReg, Reg: r4}
						ctx.BindReg(r4, &d59)
					}
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocRegPair || d59.Loc == LocStackPair || d59.Loc == LocRegTriple || d59.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d59)
					d60 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d59}, 2)
					ctx.BindReg(d60.Reg, &d60)
					ctx.BindReg(d60.Reg2, &d60)
					ctx.FreeDesc(&d59)
					ctx.EnsureDesc(&d60)
					d61 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d60}, 2)
					ctx.EmitMovPairToResult(&d61, &result)
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					blockPinnedRegs62 := make([]Reg, 0, 3)
					seenBlockPinnedRegs63 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs63
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3)
						if live && !seenBlockPinnedRegs63[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs63[r] = true
							blockPinnedRegs62 = append(blockPinnedRegs62, r)
						}
					}
					unpinBlockRegs64 := func() {
						for _, r := range blockPinnedRegs62 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs64()
					d65 = args[0]
					d65.ID = 0
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d65.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d65)
						} else if d65.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d65)
						} else if d65.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d65)
						} else if d65.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d65.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d65 = tmpPair
					} else if d65.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocRegExcept(d65.Reg), Reg2: ctx.AllocRegExcept(d65.Reg)}
						switch d65.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d65)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d65)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d65)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d65)
						d65 = tmpPair
					}
					if d65.Loc != LocRegPair && d65.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (WriteStringValue arg1)")
					}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d65)
					ctx.EmitGoCallVoid(GoFuncAddr(WriteStringValue), []JITValueDesc{d11, d65})
					ctx.FreeDesc(&d65)
					if ps.General {
					}
					ps66 := PhiState{General: ps.General}
					ps66.OverlayValues = make([]JITValueDesc, 66)
					ps66.OverlayValues[0] = d0
					ps66.OverlayValues[1] = d1
					ps66.OverlayValues[2] = d2
					ps66.OverlayValues[11] = d11
					ps66.OverlayValues[12] = d12
					ps66.OverlayValues[13] = d13
					ps66.OverlayValues[14] = d14
					ps66.OverlayValues[15] = d15
					ps66.OverlayValues[29] = d29
					ps66.OverlayValues[30] = d30
					ps66.OverlayValues[31] = d31
					ps66.OverlayValues[51] = d51
					ps66.OverlayValues[52] = d52
					ps66.OverlayValues[53] = d53
					ps66.OverlayValues[54] = d54
					ps66.OverlayValues[59] = d59
					ps66.OverlayValues[60] = d60
					ps66.OverlayValues[61] = d61
					ps66.OverlayValues[65] = d65
					if ps66.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps66)
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					ctx.ReclaimUntrackedRegs()
					d67 = args[1]
					d67.ID = 0
					d69 = d67
					d69.ID = 0
					d68 = ctx.EmitBoolDesc(&d69, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d67)
					d70 = d68
					ctx.EnsureDesc(&d70)
					if d70.Loc != LocImm && d70.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d70.Loc == LocImm {
						if d70.Imm.Bool() {
							if ps.General {
							}
							ps71 := PhiState{General: ps.General}
							ps71.OverlayValues = make([]JITValueDesc, 71)
							ps71.OverlayValues[0] = d0
							ps71.OverlayValues[1] = d1
							ps71.OverlayValues[2] = d2
							ps71.OverlayValues[11] = d11
							ps71.OverlayValues[12] = d12
							ps71.OverlayValues[13] = d13
							ps71.OverlayValues[14] = d14
							ps71.OverlayValues[15] = d15
							ps71.OverlayValues[29] = d29
							ps71.OverlayValues[30] = d30
							ps71.OverlayValues[31] = d31
							ps71.OverlayValues[51] = d51
							ps71.OverlayValues[52] = d52
							ps71.OverlayValues[53] = d53
							ps71.OverlayValues[54] = d54
							ps71.OverlayValues[59] = d59
							ps71.OverlayValues[60] = d60
							ps71.OverlayValues[61] = d61
							ps71.OverlayValues[65] = d65
							ps71.OverlayValues[67] = d67
							ps71.OverlayValues[68] = d68
							ps71.OverlayValues[69] = d69
							ps71.OverlayValues[70] = d70
							return bbs[4].RenderPS(ps71)
						}
						if ps.General {
						}
						ps72 := PhiState{General: ps.General}
						ps72.OverlayValues = make([]JITValueDesc, 71)
						ps72.OverlayValues[0] = d0
						ps72.OverlayValues[1] = d1
						ps72.OverlayValues[2] = d2
						ps72.OverlayValues[11] = d11
						ps72.OverlayValues[12] = d12
						ps72.OverlayValues[13] = d13
						ps72.OverlayValues[14] = d14
						ps72.OverlayValues[15] = d15
						ps72.OverlayValues[29] = d29
						ps72.OverlayValues[30] = d30
						ps72.OverlayValues[31] = d31
						ps72.OverlayValues[51] = d51
						ps72.OverlayValues[52] = d52
						ps72.OverlayValues[53] = d53
						ps72.OverlayValues[54] = d54
						ps72.OverlayValues[59] = d59
						ps72.OverlayValues[60] = d60
						ps72.OverlayValues[61] = d61
						ps72.OverlayValues[65] = d65
						ps72.OverlayValues[67] = d67
						ps72.OverlayValues[68] = d68
						ps72.OverlayValues[69] = d69
						ps72.OverlayValues[70] = d70
						return bbs[6].RenderPS(ps72)
					}
					if !ps.General {
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d70.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl7)
					ps73 := PhiState{General: true}
					ps73.OverlayValues = make([]JITValueDesc, 71)
					ps73.OverlayValues[0] = d0
					ps73.OverlayValues[1] = d1
					ps73.OverlayValues[2] = d2
					ps73.OverlayValues[11] = d11
					ps73.OverlayValues[12] = d12
					ps73.OverlayValues[13] = d13
					ps73.OverlayValues[14] = d14
					ps73.OverlayValues[15] = d15
					ps73.OverlayValues[29] = d29
					ps73.OverlayValues[30] = d30
					ps73.OverlayValues[31] = d31
					ps73.OverlayValues[51] = d51
					ps73.OverlayValues[52] = d52
					ps73.OverlayValues[53] = d53
					ps73.OverlayValues[54] = d54
					ps73.OverlayValues[59] = d59
					ps73.OverlayValues[60] = d60
					ps73.OverlayValues[61] = d61
					ps73.OverlayValues[65] = d65
					ps73.OverlayValues[67] = d67
					ps73.OverlayValues[68] = d68
					ps73.OverlayValues[69] = d69
					ps73.OverlayValues[70] = d70
					ps74 := PhiState{General: true}
					ps74.OverlayValues = make([]JITValueDesc, 71)
					ps74.OverlayValues[0] = d0
					ps74.OverlayValues[1] = d1
					ps74.OverlayValues[2] = d2
					ps74.OverlayValues[11] = d11
					ps74.OverlayValues[12] = d12
					ps74.OverlayValues[13] = d13
					ps74.OverlayValues[14] = d14
					ps74.OverlayValues[15] = d15
					ps74.OverlayValues[29] = d29
					ps74.OverlayValues[30] = d30
					ps74.OverlayValues[31] = d31
					ps74.OverlayValues[51] = d51
					ps74.OverlayValues[52] = d52
					ps74.OverlayValues[53] = d53
					ps74.OverlayValues[54] = d54
					ps74.OverlayValues[59] = d59
					ps74.OverlayValues[60] = d60
					ps74.OverlayValues[61] = d61
					ps74.OverlayValues[65] = d65
					ps74.OverlayValues[67] = d67
					ps74.OverlayValues[68] = d68
					ps74.OverlayValues[69] = d69
					ps74.OverlayValues[70] = d70
					snap75 := d0
					snap76 := d1
					snap77 := d2
					snap78 := d11
					snap79 := d12
					snap80 := d13
					snap81 := d14
					snap82 := d15
					snap83 := d29
					snap84 := d30
					snap85 := d31
					snap86 := d51
					snap87 := d52
					snap88 := d53
					snap89 := d54
					snap90 := d59
					snap91 := d60
					snap92 := d61
					snap93 := d65
					snap94 := d67
					snap95 := d68
					snap96 := d69
					snap97 := d70
					alloc98 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps74)
					}
					ctx.RestoreAllocState(alloc98)
					d0 = snap75
					d1 = snap76
					d2 = snap77
					d11 = snap78
					d12 = snap79
					d13 = snap80
					d14 = snap81
					d15 = snap82
					d29 = snap83
					d30 = snap84
					d31 = snap85
					d51 = snap86
					d52 = snap87
					d53 = snap88
					d54 = snap89
					d59 = snap90
					d60 = snap91
					d61 = snap92
					d65 = snap93
					d67 = snap94
					d68 = snap95
					d69 = snap96
					d70 = snap97
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps73)
					}
					return result
					ctx.FreeDesc(&d68)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps99 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps99)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  33,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sha1"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *[20]byte { return new([20]byte) }), nil, 1)
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
				ctx.EnsureDesc(&d2)
				callResults5 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d2}, []uint8{3}, []uint8{1})
				d4 := callResults5[0]
				d4.Type = tagSlice
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (sha1.Sum arg0)")
				}
				ctx.SyncDesc(&d4)
				d6 := ctx.EmitGoCallScalar(GoFuncAddr(sha1.Sum), []JITValueDesc{d4}, 3)
				ctx.BindReg(d6.Reg, &d6)
				ctx.BindReg(d6.Reg2, &d6)
				ctx.BindReg(d6.Reg3, &d6)
				ctx.EnsureDesc(&d6)
				ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[20]byte, src [20]byte) { *dst = src }), []JITValueDesc{d0, d6})
				sliceResults7 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[20]byte) []byte { return value[0:20:20] }), []JITValueDesc{d0}, []uint8{3}, []uint8{1})
				d8 := sliceResults7[0]
				ctx.EnsureDesc(&d8)
				d9 := d8
				_ = d9
				ctx.StabilizeDescForControlFlow(&d9)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d10 JITValueDesc
				if d9.SliceSizeKnown {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d9.KnownSliceLen))}
				} else if d9.Loc == LocImm {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d9.StackOff))}
				} else if d9.Loc == LocStackTriple {
					d10 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d9.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocRegTriple {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg2, ID: 0}
					} else if d9.Loc == LocReg {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				d11 := d10
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d11)
				ctx.EnsureDesc(&d11)
				var d12 JITValueDesc
				if d11.Loc == LocImm {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d11.Imm.Int() * 2)}
				} else {
					scratch := ctx.AllocRegExcept(d11.Reg)
					ctx.EmitMovRegReg(scratch, d11.Reg)
					ctx.EmitAddInt64(scratch, scratch)
					d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d12)
				}
				if d12.Loc == LocReg && d11.Loc == LocReg && d12.Reg == d11.Reg {
					ctx.TransferReg(d11.Reg)
					d11.Loc = LocNone
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.FreeDesc(&d10)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d12, d12}, []uint8{3}, []uint8{1})
				d14 := callResults13[0]
				d14.Type = tagSlice
				ctx.FreeDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				if d14.Loc != LocRegTriple && d14.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (hex.Encode arg0)")
				}
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				if d9.Loc != LocRegTriple && d9.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (hex.Encode arg1)")
				}
				ctx.SyncDesc(&d14)
				ctx.SyncDesc(&d9)
				d15 := ctx.EmitGoCallScalar(GoFuncAddr(hex.Encode), []JITValueDesc{d14, d9}, 1)
				ctx.BindReg(d15.Reg, &d15)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				callResults17 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d14}, []uint8{2}, []uint8{1})
				d16 := callResults17[0]
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d16)
				ctx.EnsureDesc(&d16)
				d18 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d16}, 2)
				if result.Loc == LocAny {
					return d18
				}
				ctx.EmitMovPairToResult(&d18, &result)
				result.Type = tagString
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  19,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sha256"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *[32]byte { return new([32]byte) }), nil, 1)
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
				ctx.EnsureDesc(&d2)
				callResults5 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d2}, []uint8{3}, []uint8{1})
				d4 := callResults5[0]
				d4.Type = tagSlice
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (sha256.Sum256 arg0)")
				}
				ctx.SyncDesc(&d4)
				d6 := ctx.EmitGoCallScalar(GoFuncAddr((func(arg0 []byte) *[32]byte { value := sha256.Sum256(arg0); return &value })), []JITValueDesc{d4}, 1)
				ctx.BindReg(d6.Reg, &d6)
				ctx.EnsureDesc(&d6)
				ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *[32]byte) { *dst = *src }), []JITValueDesc{d0, d6})
				sliceResults7 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[32]byte) []byte { return value[0:32:32] }), []JITValueDesc{d0}, []uint8{3}, []uint8{1})
				d8 := sliceResults7[0]
				ctx.EnsureDesc(&d8)
				d9 := d8
				_ = d9
				ctx.StabilizeDescForControlFlow(&d9)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d10 JITValueDesc
				if d9.SliceSizeKnown {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d9.KnownSliceLen))}
				} else if d9.Loc == LocImm {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d9.StackOff))}
				} else if d9.Loc == LocStackTriple {
					d10 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d9.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocRegTriple {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg2, ID: 0}
					} else if d9.Loc == LocReg {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				d11 := d10
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d11)
				ctx.EnsureDesc(&d11)
				var d12 JITValueDesc
				if d11.Loc == LocImm {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d11.Imm.Int() * 2)}
				} else {
					scratch := ctx.AllocRegExcept(d11.Reg)
					ctx.EmitMovRegReg(scratch, d11.Reg)
					ctx.EmitAddInt64(scratch, scratch)
					d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d12)
				}
				if d12.Loc == LocReg && d11.Loc == LocReg && d12.Reg == d11.Reg {
					ctx.TransferReg(d11.Reg)
					d11.Loc = LocNone
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.FreeDesc(&d10)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d12, d12}, []uint8{3}, []uint8{1})
				d14 := callResults13[0]
				d14.Type = tagSlice
				ctx.FreeDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				if d14.Loc != LocRegTriple && d14.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (hex.Encode arg0)")
				}
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				if d9.Loc != LocRegTriple && d9.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (hex.Encode arg1)")
				}
				ctx.SyncDesc(&d14)
				ctx.SyncDesc(&d9)
				d15 := ctx.EmitGoCallScalar(GoFuncAddr(hex.Encode), []JITValueDesc{d14, d9}, 1)
				ctx.BindReg(d15.Reg, &d15)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				callResults17 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d14}, []uint8{2}, []uint8{1})
				d16 := callResults17[0]
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d16)
				ctx.EnsureDesc(&d16)
				d18 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d16}, 2)
				if result.Loc == LocAny {
					return d18
				}
				ctx.EmitMovPairToResult(&d18, &result)
				result.Type = tagString
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  19,
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
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "regex pattern"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_test"].Fn, args, result)
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
				var d42 JITValueDesc
				_ = d42
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
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
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
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
					d16 = d14
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d16.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d16.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d16 = tmpPair
					} else if d16.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d16.Reg), Reg2: ctx.AllocRegExcept(d16.Reg)}
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d16)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d16)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d16)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d16)
						d16 = tmpPair
					} else if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
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
						d16 = tmpPair
					}
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (regexp.Compile arg0)")
					}
					ctx.SyncDesc(&d15)
					callResults17 := JITEmitGoCallResults(ctx, GoFuncAddr(regexp.Compile), []JITValueDesc{d15}, []uint8{1, 2}, []uint8{1, 3})
					d18 = callResults17[0]
					_ = d18
					d19 = callResults17[1]
					_ = d19
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.EnsureDesc(&d19)
					var d20 JITValueDesc
					if d19.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc != LocReg && d19.Loc != LocRegPair && d19.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d20)
					}
					d21 = d20
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
							ps22.OverlayValues[3] = d3
							ps22.OverlayValues[13] = d13
							ps22.OverlayValues[14] = d14
							ps22.OverlayValues[15] = d15
							ps22.OverlayValues[16] = d16
							ps22.OverlayValues[18] = d18
							ps22.OverlayValues[19] = d19
							ps22.OverlayValues[20] = d20
							ps22.OverlayValues[21] = d21
							return bbs[4].RenderPS(ps22)
						}
						if ps.General {
						}
						ps23 := PhiState{General: ps.General}
						ps23.OverlayValues = make([]JITValueDesc, 22)
						ps23.OverlayValues[0] = d0
						ps23.OverlayValues[1] = d1
						ps23.OverlayValues[2] = d2
						ps23.OverlayValues[3] = d3
						ps23.OverlayValues[13] = d13
						ps23.OverlayValues[14] = d14
						ps23.OverlayValues[15] = d15
						ps23.OverlayValues[16] = d16
						ps23.OverlayValues[18] = d18
						ps23.OverlayValues[19] = d19
						ps23.OverlayValues[20] = d20
						ps23.OverlayValues[21] = d21
						return bbs[5].RenderPS(ps23)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 22)
					ps24.OverlayValues[0] = d0
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[13] = d13
					ps24.OverlayValues[14] = d14
					ps24.OverlayValues[15] = d15
					ps24.OverlayValues[16] = d16
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					ps24.OverlayValues[21] = d21
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 22)
					ps25.OverlayValues[0] = d0
					ps25.OverlayValues[1] = d1
					ps25.OverlayValues[2] = d2
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[13] = d13
					ps25.OverlayValues[14] = d14
					ps25.OverlayValues[15] = d15
					ps25.OverlayValues[16] = d16
					ps25.OverlayValues[18] = d18
					ps25.OverlayValues[19] = d19
					ps25.OverlayValues[20] = d20
					ps25.OverlayValues[21] = d21
					snap26 := d0
					snap27 := d1
					snap28 := d2
					snap29 := d3
					snap30 := d13
					snap31 := d14
					snap32 := d15
					snap33 := d16
					snap34 := d18
					snap35 := d19
					snap36 := d20
					snap37 := d21
					alloc38 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps25)
					}
					ctx.RestoreAllocState(alloc38)
					d0 = snap26
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d13 = snap30
					d14 = snap31
					d15 = snap32
					d16 = snap33
					d18 = snap34
					d19 = snap35
					d20 = snap36
					d21 = snap37
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps24)
					}
					return result
					ctx.FreeDesc(&d20)
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
					d39 = args[1]
					d39.ID = 0
					d41 = d39
					d41.ID = 0
					d40 = ctx.EmitTagEqualsBorrowed(&d41, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d39)
					d42 = d40
					ctx.EnsureDesc(&d42)
					if d42.Loc != LocImm && d42.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d42.Loc == LocImm {
						if d42.Imm.Bool() {
							if ps.General {
							}
							ps43 := PhiState{General: ps.General}
							ps43.OverlayValues = make([]JITValueDesc, 43)
							ps43.OverlayValues[0] = d0
							ps43.OverlayValues[1] = d1
							ps43.OverlayValues[2] = d2
							ps43.OverlayValues[3] = d3
							ps43.OverlayValues[13] = d13
							ps43.OverlayValues[14] = d14
							ps43.OverlayValues[15] = d15
							ps43.OverlayValues[16] = d16
							ps43.OverlayValues[18] = d18
							ps43.OverlayValues[19] = d19
							ps43.OverlayValues[20] = d20
							ps43.OverlayValues[21] = d21
							ps43.OverlayValues[39] = d39
							ps43.OverlayValues[40] = d40
							ps43.OverlayValues[41] = d41
							ps43.OverlayValues[42] = d42
							return bbs[1].RenderPS(ps43)
						}
						if ps.General {
						}
						ps44 := PhiState{General: ps.General}
						ps44.OverlayValues = make([]JITValueDesc, 43)
						ps44.OverlayValues[0] = d0
						ps44.OverlayValues[1] = d1
						ps44.OverlayValues[2] = d2
						ps44.OverlayValues[3] = d3
						ps44.OverlayValues[13] = d13
						ps44.OverlayValues[14] = d14
						ps44.OverlayValues[15] = d15
						ps44.OverlayValues[16] = d16
						ps44.OverlayValues[18] = d18
						ps44.OverlayValues[19] = d19
						ps44.OverlayValues[20] = d20
						ps44.OverlayValues[21] = d21
						ps44.OverlayValues[39] = d39
						ps44.OverlayValues[40] = d40
						ps44.OverlayValues[41] = d41
						ps44.OverlayValues[42] = d42
						return bbs[2].RenderPS(ps44)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d42.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl3)
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 43)
					ps45.OverlayValues[0] = d0
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[15] = d15
					ps45.OverlayValues[16] = d16
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[19] = d19
					ps45.OverlayValues[20] = d20
					ps45.OverlayValues[21] = d21
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
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[15] = d15
					ps46.OverlayValues[16] = d16
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[19] = d19
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[21] = d21
					ps46.OverlayValues[39] = d39
					ps46.OverlayValues[40] = d40
					ps46.OverlayValues[41] = d41
					ps46.OverlayValues[42] = d42
					snap47 := d0
					snap48 := d1
					snap49 := d2
					snap50 := d3
					snap51 := d13
					snap52 := d14
					snap53 := d15
					snap54 := d16
					snap55 := d18
					snap56 := d19
					snap57 := d20
					snap58 := d21
					snap59 := d39
					snap60 := d40
					snap61 := d41
					snap62 := d42
					alloc63 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps46)
					}
					ctx.RestoreAllocState(alloc63)
					d0 = snap47
					d1 = snap48
					d2 = snap49
					d3 = snap50
					d13 = snap51
					d14 = snap52
					d15 = snap53
					d16 = snap54
					d18 = snap55
					d19 = snap56
					d20 = snap57
					d21 = snap58
					d39 = snap59
					d40 = snap60
					d41 = snap61
					d42 = snap62
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps45)
					}
					return result
					ctx.FreeDesc(&d40)
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_test"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
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
					d64 = args[0]
					d64.ID = 0
					d66 = d64
					ctx.EnsureDesc(&d66)
					if d66.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d66.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d66)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d66)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d66)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d66.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d66 = tmpPair
					} else if d66.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d66.Reg), Reg2: ctx.AllocRegExcept(d66.Reg)}
						switch d66.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d66)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d66)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d66)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d66)
						d66 = tmpPair
					} else if d66.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d66.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d66.MemPtr))
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
						d66 = tmpPair
					}
					if d66.Loc != LocRegPair && d66.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d66}, 2)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d65.Imm)
						ptrWord, _ := d65.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d65.Imm.String())))
						d65 = tmpPair
					} else if d65.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocRegExcept(d65.Reg), Reg2: ctx.AllocRegExcept(d65.Reg)}
						switch d65.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d65)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d65)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d65)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d65)
						d65 = tmpPair
					}
					if d65.Loc != LocRegPair && d65.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).MatchString arg1)")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d65)
					d67 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).MatchString), []JITValueDesc{d18, d65}, 1)
					ctx.EmitAndRegImm32(d67.Reg, 1)
					d67.Type = tagBool
					ctx.BindReg(d67.Reg, &d67)
					ctx.FreeDesc(&d18)
					ctx.EnsureDesc(&d67)
					if d67.Loc == LocImm {
						ctx.EmitMakeBool(result, d67)
					} else {
						ctx.EmitMovToReg(result.Reg2, d67)
						d68 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d68)
						if d67.Loc == LocReg && d67.Reg != result.Reg2 {
							ctx.FreeReg(d67.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps69 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps69)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      28,
		},
		Optimize: optimizeRegexpTest,
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
