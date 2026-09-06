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

// generalCIFoldCompare preserves the existing strings.ToLower ordering while
// keeping the overwhelmingly common ASCII index path allocation-free.
func generalCIFoldCompare(left, right string) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] >= 0x80 || right[i] >= 0x80 {
			return strings.Compare(strings.ToLower(left), strings.ToLower(right))
		}
		l, r := asciiFoldByte(left[i]), asciiFoldByte(right[i])
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

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
	case json.Number:
		if value, err := a.Int64(); err == nil {
			return NewInt(value)
		}
		if value, err := a.Float64(); err == nil {
			return NewFloat(value)
		}
		return NewString(string(a))
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
				declaration := declarations["string?"]
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Any), []JITValueDesc{d0}, 2)
				d1.NoHeapPointer = false
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
				// JITGen native call boundary: interface type assertion.
				ctx.Coverage.NativeCalls++
				declaration := declarations["concat"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
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
				declaration := declarations["sql_concat"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Func), []JITValueDesc{d1}, 1)
				d2.NoHeapPointer = false
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
				ctx.SyncDesc(&d4)
				if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocInputPair {
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
				declaration := declarations["substr"]
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					ctx.SyncDesc(&d2)
					if d2.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d2 = tmpScalar
					}
					d2 = JITPrepareScmerGoArg(ctx, d2)
					if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
					lbl4 := ctx.ReserveLabel()
					_ = lbl4
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
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
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d8)
					}
					ctx.FreeDesc(&d7)
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitJump(d9.Condition, lbl5)
					ctx.EmitJmp(lbl6)
					snap12 := d0
					snap13 := d1
					snap14 := d2
					snap15 := d3
					snap16 := d4
					snap17 := d5
					snap18 := d6
					snap19 := d7
					snap20 := d8
					snap21 := d9
					alloc22 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc22)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d5 = snap17
					d6 = snap18
					d7 = snap19
					d8 = snap20
					d9 = snap21
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc22)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d5 = snap17
					d6 = snap18
					d7 = snap19
					d8 = snap20
					d9 = snap21
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 10)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[6] = d6
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[8] = d8
					ps23.OverlayValues[9] = d9
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 10)
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
					snap25 := d0
					snap26 := d1
					snap27 := d2
					snap28 := d3
					snap29 := d4
					snap30 := d5
					snap31 := d6
					snap32 := d7
					snap33 := d8
					snap34 := d9
					alloc35 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps24)
					}
					ctx.RestoreAllocState(alloc35)
					d0 = snap25
					d1 = snap26
					d2 = snap27
					d3 = snap28
					d4 = snap29
					d5 = snap30
					d6 = snap31
					d7 = snap32
					d8 = snap33
					d9 = snap34
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps23)
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
					d36 = args[2]
					d36.ID = 0
					ctx.EnsureDesc(&d36)
					d37 = d36
					_ = d37
					ctx.StabilizeDescForControlFlow(&d37)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d38 JITValueDesc
					if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int())}
					} else if d37.Type == tagInt && d37.Loc == LocRegPair {
						ctx.FreeReg(d37.Reg)
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg2}
						ctx.BindReg(d37.Reg2, &d38)
						ctx.BindReg(d37.Reg2, &d38)
					} else if d37.Type == tagInt && d37.Loc == LocReg {
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg}
						ctx.BindReg(d37.Reg, &d38)
						ctx.BindReg(d37.Reg, &d38)
					} else {
						d38 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d37}, 1)
						d38.Type = tagInt
						ctx.BindReg(d38.Reg, &d38)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.FreeDesc(&d36)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDescsTogether(&d5, &d38)
					var d40 JITValueDesc
					if d5.Loc == LocImm && d38.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() + d38.Imm.Int())}
					} else if d38.Loc == LocImm && d38.Imm.Int() == 0 {
						r1 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(r1, d5.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d40)
					} else if d5.Loc == LocImm && d5.Imm.Int() == 0 {
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg}
						ctx.BindReg(d38.Reg, &d40)
					} else if d5.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d38.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
						ctx.EmitAddInt64(scratch, d38.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d40)
					} else if d38.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(scratch, d5.Reg)
						if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d38.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d38.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d40)
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg, d38.Reg)
						ctx.EmitMovRegReg(r2, d5.Reg)
						ctx.EmitAddInt64(r2, d38.Reg)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d40)
					}
					if d40.Loc == LocReg && d5.Loc == LocReg && d40.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.FreeDesc(&d38)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d40)
					var d42 JITValueDesc
					if d40.Loc == LocImm && d5.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int() - d5.Imm.Int())}
					} else {
						r3 := ctx.AllocReg()
						if d40.Loc == LocImm {
							ctx.EmitMovRegImm64(r3, uint64(d40.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r3, d40.Reg)
						}
						if d5.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
							ctx.EmitSubInt64(r3, RegR11)
						} else {
							ctx.EmitSubInt64(r3, d5.Reg)
						}
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d42)
					}
					var d43 JITValueDesc
					r4 := ctx.EmitSliceDataAfterLow(&d1, &d5, 1)
					d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
					ctx.BindReg(r4, &d43)
					ctx.BindReg(r4, &d43)
					var d44 JITValueDesc
					var r5 Reg
					var r6 Reg
					ctx.SyncDesc(&d43)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						r5 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r5, uint64(d43.Imm.Int()))
					} else {
						r5 = d43.Reg
					}
					ctx.ProtectReg(r5)
					ctx.SyncDesc(&d42)
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d42.Imm.Int()))
					} else {
						r6 = d42.Reg
					}
					ctx.ProtectReg(r6)
					ctx.UnprotectReg(r6)
					ctx.UnprotectReg(r5)
					d44 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
					ctx.BindReg(r5, &d44)
					ctx.BindReg(r6, &d44)
					ctx.BindReg(r5, &d44)
					ctx.BindReg(r6, &d44)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d44)
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d44}, 2)
					ctx.EmitMovPairToResult(&d45, &result)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					var d46 JITValueDesc
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
						d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d46)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d46)
					var d48 JITValueDesc
					if d46.Loc == LocImm && d5.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d46.Imm.Int() - d5.Imm.Int())}
					} else {
						r7 := ctx.AllocReg()
						if d46.Loc == LocImm {
							ctx.EmitMovRegImm64(r7, uint64(d46.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r7, d46.Reg)
						}
						if d5.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
							ctx.EmitSubInt64(r7, RegR11)
						} else {
							ctx.EmitSubInt64(r7, d5.Reg)
						}
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d48)
					}
					var d49 JITValueDesc
					r8 := ctx.EmitSliceDataAfterLow(&d1, &d5, 1)
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d49)
					ctx.BindReg(r8, &d49)
					var d50 JITValueDesc
					var r9 Reg
					var r10 Reg
					ctx.SyncDesc(&d49)
					ctx.EnsureDesc(&d49)
					if d49.Loc == LocImm {
						r9 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r9, uint64(d49.Imm.Int()))
					} else {
						r9 = d49.Reg
					}
					ctx.ProtectReg(r9)
					ctx.SyncDesc(&d48)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						r10 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r10, uint64(d48.Imm.Int()))
					} else {
						r10 = d48.Reg
					}
					ctx.ProtectReg(r10)
					ctx.UnprotectReg(r10)
					ctx.UnprotectReg(r9)
					d50 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d50)
					ctx.BindReg(r10, &d50)
					ctx.BindReg(r9, &d50)
					ctx.BindReg(r10, &d50)
					ctx.EnsureDesc(&d50)
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d50}, 2)
					ctx.EmitMovPairToResult(&d51, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps52 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps52)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["sql_substr"]
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
				var d38 JITValueDesc
				_ = d38
				var d40 JITValueDesc
				_ = d40
				var d62 JITValueDesc
				_ = d62
				var d65 JITValueDesc
				_ = d65
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d96 JITValueDesc
				_ = d96
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
				var d238 JITValueDesc
				_ = d238
				var d240 JITValueDesc
				_ = d240
				var d283 JITValueDesc
				_ = d283
				var d286 JITValueDesc
				_ = d286
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
				var d344 JITValueDesc
				_ = d344
				var d457 JITValueDesc
				_ = d457
				var d458 JITValueDesc
				_ = d458
				var d459 JITValueDesc
				_ = d459
				var d460 JITValueDesc
				_ = d460
				var d461 JITValueDesc
				_ = d461
				var d462 JITValueDesc
				_ = d462
				var d463 JITValueDesc
				_ = d463
				var d464 JITValueDesc
				_ = d464
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				var bbs [13]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiCount = uint16(1)
				bbs[10].PhiBase = int32(phiBase0) + int32(16)
				bbs[10].PhiCount = uint16(1)
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
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
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
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					snap13 := d5
					snap14 := d6
					alloc15 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc15)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl3)
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
					d26 = args[0]
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
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.FreeDesc(&d26)
					var d29 JITValueDesc
					if d27.SliceSizeKnown {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d27.KnownSliceLen))}
					} else if d27.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d27.Imm.String())))}
					} else if d27.Loc == LocStackTriple {
						d29 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d27.StackOff + 8, NoHeapPointer: true}
					} else if d27.Loc == LocStackPair {
						d29 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d27.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d27)
						if d27.Loc == LocRegPair || d27.Loc == LocRegTriple {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2, ID: 0}
						} else if d27.Loc == LocReg {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d29)
					d30 = args[1]
					d30.ID = 0
					ctx.EnsureDesc(&d30)
					d31 = d30
					_ = d31
					ctx.StabilizeDescForControlFlow(&d31)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d32 JITValueDesc
					if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int())}
					} else if d31.Type == tagInt && d31.Loc == LocRegPair {
						ctx.FreeReg(d31.Reg)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2}
						ctx.BindReg(d31.Reg2, &d32)
						ctx.BindReg(d31.Reg2, &d32)
					} else if d31.Type == tagInt && d31.Loc == LocReg {
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
						ctx.BindReg(d31.Reg, &d32)
						ctx.BindReg(d31.Reg, &d32)
					} else {
						d32 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d31}, 1)
						d32.Type = tagInt
						ctx.BindReg(d32.Reg, &d32)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					var d34 JITValueDesc
					if d32.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d32.Reg)
						ctx.EmitMovRegReg(scratch, d32.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d34)
					}
					if d34.Loc == LocReg && d32.Loc == LocReg && d34.Reg == d32.Reg {
						ctx.TransferReg(d32.Reg)
						d32.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d34)
					var d35 JITValueDesc
					if d34.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d34.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d34.Reg)
						ctx.EmitCmpRegImm32(d34.Reg, 0)
						d35 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d35)
					}
					d36 = d35
					ctx.EnsureDesc(&d36)
					if d36.Loc != LocImm && d36.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d36.Loc == LocImm {
						if d36.Imm.Bool() {
							if ps.General {
							}
							ps37 := PhiState{General: ps.General}
							ps37.OverlayValues = make([]JITValueDesc, 37)
							ps37.OverlayValues[1] = d1
							ps37.OverlayValues[2] = d2
							ps37.OverlayValues[3] = d3
							ps37.OverlayValues[4] = d4
							ps37.OverlayValues[5] = d5
							ps37.OverlayValues[6] = d6
							ps37.OverlayValues[25] = d25
							ps37.OverlayValues[26] = d26
							ps37.OverlayValues[27] = d27
							ps37.OverlayValues[28] = d28
							ps37.OverlayValues[29] = d29
							ps37.OverlayValues[30] = d30
							ps37.OverlayValues[31] = d31
							ps37.OverlayValues[32] = d32
							ps37.OverlayValues[33] = d33
							ps37.OverlayValues[34] = d34
							ps37.OverlayValues[35] = d35
							ps37.OverlayValues[36] = d36
							return bbs[3].RenderPS(ps37)
						}
						if ps.General {
							ctx.SyncDesc(&d34)
							if d34.Loc == LocReg {
								ctx.ProtectReg(d34.Reg)
							} else if d34.Loc == LocRegPair {
								ctx.ProtectReg(d34.Reg)
								ctx.ProtectReg(d34.Reg2)
							}
							d38 = d34
							if d38.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d38)
							ctx.EmitStoreToStack(d38, int32(bbs[4].PhiBase)+int32(0))
							if d34.Loc == LocReg {
								ctx.UnprotectReg(d34.Reg)
							} else if d34.Loc == LocRegPair {
								ctx.UnprotectReg(d34.Reg)
								ctx.UnprotectReg(d34.Reg2)
							}
						}
						ps39 := PhiState{General: ps.General}
						ps39.OverlayValues = make([]JITValueDesc, 39)
						ps39.OverlayValues[1] = d1
						ps39.OverlayValues[2] = d2
						ps39.OverlayValues[3] = d3
						ps39.OverlayValues[4] = d4
						ps39.OverlayValues[5] = d5
						ps39.OverlayValues[6] = d6
						ps39.OverlayValues[25] = d25
						ps39.OverlayValues[26] = d26
						ps39.OverlayValues[27] = d27
						ps39.OverlayValues[28] = d28
						ps39.OverlayValues[29] = d29
						ps39.OverlayValues[30] = d30
						ps39.OverlayValues[31] = d31
						ps39.OverlayValues[32] = d32
						ps39.OverlayValues[33] = d33
						ps39.OverlayValues[34] = d34
						ps39.OverlayValues[35] = d35
						ps39.OverlayValues[36] = d36
						ps39.OverlayValues[38] = d38
						ps39.PhiValues = make([]JITValueDesc, 1)
						d40 = d34
						ps39.PhiValues[0] = d40
						return bbs[4].RenderPS(ps39)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					ctx.EmitJump(d36.Condition, lbl17)
					ctx.EmitJmp(lbl18)
					snap41 := d1
					snap42 := d2
					snap43 := d3
					snap44 := d4
					snap45 := d5
					snap46 := d6
					snap47 := d25
					snap48 := d26
					snap49 := d27
					snap50 := d28
					snap51 := d29
					snap52 := d30
					snap53 := d31
					snap54 := d32
					snap55 := d33
					snap56 := d34
					snap57 := d35
					snap58 := d36
					snap59 := d38
					snap60 := d40
					alloc61 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc61)
					d1 = snap41
					d2 = snap42
					d3 = snap43
					d4 = snap44
					d5 = snap45
					d6 = snap46
					d25 = snap47
					d26 = snap48
					d27 = snap49
					d28 = snap50
					d29 = snap51
					d30 = snap52
					d31 = snap53
					d32 = snap54
					d33 = snap55
					d34 = snap56
					d35 = snap57
					d36 = snap58
					d38 = snap59
					d40 = snap60
					ctx.MarkLabel(lbl18)
					ctx.SyncDesc(&d34)
					if d34.Loc == LocReg {
						ctx.ProtectReg(d34.Reg)
					} else if d34.Loc == LocRegPair {
						ctx.ProtectReg(d34.Reg)
						ctx.ProtectReg(d34.Reg2)
					}
					d62 = d34
					if d62.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d62)
					ctx.EmitStoreToStack(d62, int32(bbs[4].PhiBase)+int32(0))
					if d34.Loc == LocReg {
						ctx.UnprotectReg(d34.Reg)
					} else if d34.Loc == LocRegPair {
						ctx.UnprotectReg(d34.Reg)
						ctx.UnprotectReg(d34.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc61)
					d1 = snap41
					d2 = snap42
					d3 = snap43
					d4 = snap44
					d5 = snap45
					d6 = snap46
					d25 = snap47
					d26 = snap48
					d27 = snap49
					d28 = snap50
					d29 = snap51
					d30 = snap52
					d31 = snap53
					d32 = snap54
					d33 = snap55
					d34 = snap56
					d35 = snap57
					d36 = snap58
					d38 = snap59
					d40 = snap60
					ps63 := PhiState{General: true}
					ps63.OverlayValues = make([]JITValueDesc, 63)
					ps63.OverlayValues[1] = d1
					ps63.OverlayValues[2] = d2
					ps63.OverlayValues[3] = d3
					ps63.OverlayValues[4] = d4
					ps63.OverlayValues[5] = d5
					ps63.OverlayValues[6] = d6
					ps63.OverlayValues[25] = d25
					ps63.OverlayValues[26] = d26
					ps63.OverlayValues[27] = d27
					ps63.OverlayValues[28] = d28
					ps63.OverlayValues[29] = d29
					ps63.OverlayValues[30] = d30
					ps63.OverlayValues[31] = d31
					ps63.OverlayValues[32] = d32
					ps63.OverlayValues[33] = d33
					ps63.OverlayValues[34] = d34
					ps63.OverlayValues[35] = d35
					ps63.OverlayValues[36] = d36
					ps63.OverlayValues[38] = d38
					ps63.OverlayValues[40] = d40
					ps63.OverlayValues[62] = d62
					ps64 := PhiState{General: true}
					ps64.OverlayValues = make([]JITValueDesc, 63)
					ps64.OverlayValues[1] = d1
					ps64.OverlayValues[2] = d2
					ps64.OverlayValues[3] = d3
					ps64.OverlayValues[4] = d4
					ps64.OverlayValues[5] = d5
					ps64.OverlayValues[6] = d6
					ps64.OverlayValues[25] = d25
					ps64.OverlayValues[26] = d26
					ps64.OverlayValues[27] = d27
					ps64.OverlayValues[28] = d28
					ps64.OverlayValues[29] = d29
					ps64.OverlayValues[30] = d30
					ps64.OverlayValues[31] = d31
					ps64.OverlayValues[32] = d32
					ps64.OverlayValues[33] = d33
					ps64.OverlayValues[34] = d34
					ps64.OverlayValues[35] = d35
					ps64.OverlayValues[36] = d36
					ps64.OverlayValues[38] = d38
					ps64.OverlayValues[40] = d40
					ps64.OverlayValues[62] = d62
					ps64.PhiValues = make([]JITValueDesc, 1)
					d65 = d34
					ps64.PhiValues[0] = d65
					snap66 := d1
					snap67 := d2
					snap68 := d3
					snap69 := d4
					snap70 := d5
					snap71 := d6
					snap72 := d25
					snap73 := d26
					snap74 := d27
					snap75 := d28
					snap76 := d29
					snap77 := d30
					snap78 := d31
					snap79 := d32
					snap80 := d33
					snap81 := d34
					snap82 := d35
					snap83 := d36
					snap84 := d38
					snap85 := d40
					snap86 := d62
					snap87 := d65
					alloc88 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps64)
					}
					ctx.RestoreAllocState(alloc88)
					d1 = snap66
					d2 = snap67
					d3 = snap68
					d4 = snap69
					d5 = snap70
					d6 = snap71
					d25 = snap72
					d26 = snap73
					d27 = snap74
					d28 = snap75
					d29 = snap76
					d30 = snap77
					d31 = snap78
					d32 = snap79
					d33 = snap80
					d34 = snap81
					d35 = snap82
					d36 = snap83
					d38 = snap84
					d40 = snap85
					d62 = snap86
					d65 = snap87
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps63)
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					}
					ps89 := PhiState{General: ps.General}
					ps89.OverlayValues = make([]JITValueDesc, 66)
					ps89.OverlayValues[1] = d1
					ps89.OverlayValues[2] = d2
					ps89.OverlayValues[3] = d3
					ps89.OverlayValues[4] = d4
					ps89.OverlayValues[5] = d5
					ps89.OverlayValues[6] = d6
					ps89.OverlayValues[25] = d25
					ps89.OverlayValues[26] = d26
					ps89.OverlayValues[27] = d27
					ps89.OverlayValues[28] = d28
					ps89.OverlayValues[29] = d29
					ps89.OverlayValues[30] = d30
					ps89.OverlayValues[31] = d31
					ps89.OverlayValues[32] = d32
					ps89.OverlayValues[33] = d33
					ps89.OverlayValues[34] = d34
					ps89.OverlayValues[35] = d35
					ps89.OverlayValues[36] = d36
					ps89.OverlayValues[38] = d38
					ps89.OverlayValues[40] = d40
					ps89.OverlayValues[62] = d62
					ps89.OverlayValues[65] = d65
					ps89.PhiValues = make([]JITValueDesc, 1)
					d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps89.PhiValues[0] = d90
					if ps89.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps89)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d91 := ps.PhiValues[0]
							ctx.EnsureDesc(&d91)
							ctx.EmitStoreToStack(d91, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDescsTogether(&d1, &d29)
					var d92 JITValueDesc
					if d1.Loc == LocImm && d29.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() >= d29.Imm.Int())}
					} else if d29.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d1.Reg)
						if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d29.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d29.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						d92 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r1, &d92)
					} else if d1.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d29.Reg)
						d92 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r2, &d92)
					} else {
						r3 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d29.Reg)
						d92 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r3, &d92)
					}
					d93 = d92
					ctx.EnsureDesc(&d93)
					if d93.Loc != LocImm && d93.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d93.Loc == LocImm {
						if d93.Imm.Bool() {
							if ps.General {
							}
							ps94 := PhiState{General: ps.General}
							ps94.OverlayValues = make([]JITValueDesc, 94)
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
							ps94.OverlayValues[29] = d29
							ps94.OverlayValues[30] = d30
							ps94.OverlayValues[31] = d31
							ps94.OverlayValues[32] = d32
							ps94.OverlayValues[33] = d33
							ps94.OverlayValues[34] = d34
							ps94.OverlayValues[35] = d35
							ps94.OverlayValues[36] = d36
							ps94.OverlayValues[38] = d38
							ps94.OverlayValues[40] = d40
							ps94.OverlayValues[62] = d62
							ps94.OverlayValues[65] = d65
							ps94.OverlayValues[90] = d90
							ps94.OverlayValues[91] = d91
							ps94.OverlayValues[92] = d92
							ps94.OverlayValues[93] = d93
							return bbs[5].RenderPS(ps94)
						}
						if ps.General {
						}
						ps95 := PhiState{General: ps.General}
						ps95.OverlayValues = make([]JITValueDesc, 94)
						ps95.OverlayValues[1] = d1
						ps95.OverlayValues[2] = d2
						ps95.OverlayValues[3] = d3
						ps95.OverlayValues[4] = d4
						ps95.OverlayValues[5] = d5
						ps95.OverlayValues[6] = d6
						ps95.OverlayValues[25] = d25
						ps95.OverlayValues[26] = d26
						ps95.OverlayValues[27] = d27
						ps95.OverlayValues[28] = d28
						ps95.OverlayValues[29] = d29
						ps95.OverlayValues[30] = d30
						ps95.OverlayValues[31] = d31
						ps95.OverlayValues[32] = d32
						ps95.OverlayValues[33] = d33
						ps95.OverlayValues[34] = d34
						ps95.OverlayValues[35] = d35
						ps95.OverlayValues[36] = d36
						ps95.OverlayValues[38] = d38
						ps95.OverlayValues[40] = d40
						ps95.OverlayValues[62] = d62
						ps95.OverlayValues[65] = d65
						ps95.OverlayValues[90] = d90
						ps95.OverlayValues[91] = d91
						ps95.OverlayValues[92] = d92
						ps95.OverlayValues[93] = d93
						return bbs[6].RenderPS(ps95)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d96 := ps.PhiValues[0]
							ctx.EnsureDesc(&d96)
							ctx.EmitStoreToStack(d96, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					ctx.EmitJump(d93.Condition, lbl19)
					ctx.EmitJmp(lbl20)
					snap97 := d1
					snap98 := d2
					snap99 := d3
					snap100 := d4
					snap101 := d5
					snap102 := d6
					snap103 := d25
					snap104 := d26
					snap105 := d27
					snap106 := d28
					snap107 := d29
					snap108 := d30
					snap109 := d31
					snap110 := d32
					snap111 := d33
					snap112 := d34
					snap113 := d35
					snap114 := d36
					snap115 := d38
					snap116 := d40
					snap117 := d62
					snap118 := d65
					snap119 := d90
					snap120 := d91
					snap121 := d92
					snap122 := d93
					snap123 := d96
					alloc124 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc124)
					d1 = snap97
					d2 = snap98
					d3 = snap99
					d4 = snap100
					d5 = snap101
					d6 = snap102
					d25 = snap103
					d26 = snap104
					d27 = snap105
					d28 = snap106
					d29 = snap107
					d30 = snap108
					d31 = snap109
					d32 = snap110
					d33 = snap111
					d34 = snap112
					d35 = snap113
					d36 = snap114
					d38 = snap115
					d40 = snap116
					d62 = snap117
					d65 = snap118
					d90 = snap119
					d91 = snap120
					d92 = snap121
					d93 = snap122
					d96 = snap123
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc124)
					d1 = snap97
					d2 = snap98
					d3 = snap99
					d4 = snap100
					d5 = snap101
					d6 = snap102
					d25 = snap103
					d26 = snap104
					d27 = snap105
					d28 = snap106
					d29 = snap107
					d30 = snap108
					d31 = snap109
					d32 = snap110
					d33 = snap111
					d34 = snap112
					d35 = snap113
					d36 = snap114
					d38 = snap115
					d40 = snap116
					d62 = snap117
					d65 = snap118
					d90 = snap119
					d91 = snap120
					d92 = snap121
					d93 = snap122
					d96 = snap123
					ps125 := PhiState{General: true}
					ps125.OverlayValues = make([]JITValueDesc, 97)
					ps125.OverlayValues[1] = d1
					ps125.OverlayValues[2] = d2
					ps125.OverlayValues[3] = d3
					ps125.OverlayValues[4] = d4
					ps125.OverlayValues[5] = d5
					ps125.OverlayValues[6] = d6
					ps125.OverlayValues[25] = d25
					ps125.OverlayValues[26] = d26
					ps125.OverlayValues[27] = d27
					ps125.OverlayValues[28] = d28
					ps125.OverlayValues[29] = d29
					ps125.OverlayValues[30] = d30
					ps125.OverlayValues[31] = d31
					ps125.OverlayValues[32] = d32
					ps125.OverlayValues[33] = d33
					ps125.OverlayValues[34] = d34
					ps125.OverlayValues[35] = d35
					ps125.OverlayValues[36] = d36
					ps125.OverlayValues[38] = d38
					ps125.OverlayValues[40] = d40
					ps125.OverlayValues[62] = d62
					ps125.OverlayValues[65] = d65
					ps125.OverlayValues[90] = d90
					ps125.OverlayValues[91] = d91
					ps125.OverlayValues[92] = d92
					ps125.OverlayValues[93] = d93
					ps125.OverlayValues[96] = d96
					ps126 := PhiState{General: true}
					ps126.OverlayValues = make([]JITValueDesc, 97)
					ps126.OverlayValues[1] = d1
					ps126.OverlayValues[2] = d2
					ps126.OverlayValues[3] = d3
					ps126.OverlayValues[4] = d4
					ps126.OverlayValues[5] = d5
					ps126.OverlayValues[6] = d6
					ps126.OverlayValues[25] = d25
					ps126.OverlayValues[26] = d26
					ps126.OverlayValues[27] = d27
					ps126.OverlayValues[28] = d28
					ps126.OverlayValues[29] = d29
					ps126.OverlayValues[30] = d30
					ps126.OverlayValues[31] = d31
					ps126.OverlayValues[32] = d32
					ps126.OverlayValues[33] = d33
					ps126.OverlayValues[34] = d34
					ps126.OverlayValues[35] = d35
					ps126.OverlayValues[36] = d36
					ps126.OverlayValues[38] = d38
					ps126.OverlayValues[40] = d40
					ps126.OverlayValues[62] = d62
					ps126.OverlayValues[65] = d65
					ps126.OverlayValues[90] = d90
					ps126.OverlayValues[91] = d91
					ps126.OverlayValues[92] = d92
					ps126.OverlayValues[93] = d93
					ps126.OverlayValues[96] = d96
					snap127 := d1
					snap128 := d2
					snap129 := d3
					snap130 := d4
					snap131 := d5
					snap132 := d6
					snap133 := d25
					snap134 := d26
					snap135 := d27
					snap136 := d28
					snap137 := d29
					snap138 := d30
					snap139 := d31
					snap140 := d32
					snap141 := d33
					snap142 := d34
					snap143 := d35
					snap144 := d36
					snap145 := d38
					snap146 := d40
					snap147 := d62
					snap148 := d65
					snap149 := d90
					snap150 := d91
					snap151 := d92
					snap152 := d93
					snap153 := d96
					alloc154 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps126)
					}
					ctx.RestoreAllocState(alloc154)
					d1 = snap127
					d2 = snap128
					d3 = snap129
					d4 = snap130
					d5 = snap131
					d6 = snap132
					d25 = snap133
					d26 = snap134
					d27 = snap135
					d28 = snap136
					d29 = snap137
					d30 = snap138
					d31 = snap139
					d32 = snap140
					d33 = snap141
					d34 = snap142
					d35 = snap143
					d36 = snap144
					d38 = snap145
					d40 = snap146
					d62 = snap147
					d65 = snap148
					d90 = snap149
					d91 = snap150
					d92 = snap151
					d93 = snap152
					d96 = snap153
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps125)
					}
					return result
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					ctx.ReclaimUntrackedRegs()
					d155 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d156 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d155}, 2)
					ctx.EmitMovPairToResult(&d156, &result)
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					ctx.ReclaimUntrackedRegs()
					d157 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d157)
					var d158 JITValueDesc
					if d157.Loc == LocImm {
						d158 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d157.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d157.Reg, 2)
						d158 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedGreater}
						ctx.BindReg(r4, &d158)
					}
					ctx.FreeDesc(&d157)
					d159 = d158
					ctx.EnsureDesc(&d159)
					if d159.Loc != LocImm && d159.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
							ps160.OverlayValues[25] = d25
							ps160.OverlayValues[26] = d26
							ps160.OverlayValues[27] = d27
							ps160.OverlayValues[28] = d28
							ps160.OverlayValues[29] = d29
							ps160.OverlayValues[30] = d30
							ps160.OverlayValues[31] = d31
							ps160.OverlayValues[32] = d32
							ps160.OverlayValues[33] = d33
							ps160.OverlayValues[34] = d34
							ps160.OverlayValues[35] = d35
							ps160.OverlayValues[36] = d36
							ps160.OverlayValues[38] = d38
							ps160.OverlayValues[40] = d40
							ps160.OverlayValues[62] = d62
							ps160.OverlayValues[65] = d65
							ps160.OverlayValues[90] = d90
							ps160.OverlayValues[91] = d91
							ps160.OverlayValues[92] = d92
							ps160.OverlayValues[93] = d93
							ps160.OverlayValues[96] = d96
							ps160.OverlayValues[155] = d155
							ps160.OverlayValues[156] = d156
							ps160.OverlayValues[157] = d157
							ps160.OverlayValues[158] = d158
							ps160.OverlayValues[159] = d159
							return bbs[7].RenderPS(ps160)
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
						ps161.OverlayValues[25] = d25
						ps161.OverlayValues[26] = d26
						ps161.OverlayValues[27] = d27
						ps161.OverlayValues[28] = d28
						ps161.OverlayValues[29] = d29
						ps161.OverlayValues[30] = d30
						ps161.OverlayValues[31] = d31
						ps161.OverlayValues[32] = d32
						ps161.OverlayValues[33] = d33
						ps161.OverlayValues[34] = d34
						ps161.OverlayValues[35] = d35
						ps161.OverlayValues[36] = d36
						ps161.OverlayValues[38] = d38
						ps161.OverlayValues[40] = d40
						ps161.OverlayValues[62] = d62
						ps161.OverlayValues[65] = d65
						ps161.OverlayValues[90] = d90
						ps161.OverlayValues[91] = d91
						ps161.OverlayValues[92] = d92
						ps161.OverlayValues[93] = d93
						ps161.OverlayValues[96] = d96
						ps161.OverlayValues[155] = d155
						ps161.OverlayValues[156] = d156
						ps161.OverlayValues[157] = d157
						ps161.OverlayValues[158] = d158
						ps161.OverlayValues[159] = d159
						return bbs[8].RenderPS(ps161)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					ctx.EmitJump(d159.Condition, lbl21)
					ctx.EmitJmp(lbl22)
					snap162 := d1
					snap163 := d2
					snap164 := d3
					snap165 := d4
					snap166 := d5
					snap167 := d6
					snap168 := d25
					snap169 := d26
					snap170 := d27
					snap171 := d28
					snap172 := d29
					snap173 := d30
					snap174 := d31
					snap175 := d32
					snap176 := d33
					snap177 := d34
					snap178 := d35
					snap179 := d36
					snap180 := d38
					snap181 := d40
					snap182 := d62
					snap183 := d65
					snap184 := d90
					snap185 := d91
					snap186 := d92
					snap187 := d93
					snap188 := d96
					snap189 := d155
					snap190 := d156
					snap191 := d157
					snap192 := d158
					snap193 := d159
					alloc194 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc194)
					d1 = snap162
					d2 = snap163
					d3 = snap164
					d4 = snap165
					d5 = snap166
					d6 = snap167
					d25 = snap168
					d26 = snap169
					d27 = snap170
					d28 = snap171
					d29 = snap172
					d30 = snap173
					d31 = snap174
					d32 = snap175
					d33 = snap176
					d34 = snap177
					d35 = snap178
					d36 = snap179
					d38 = snap180
					d40 = snap181
					d62 = snap182
					d65 = snap183
					d90 = snap184
					d91 = snap185
					d92 = snap186
					d93 = snap187
					d96 = snap188
					d155 = snap189
					d156 = snap190
					d157 = snap191
					d158 = snap192
					d159 = snap193
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl9)
					ctx.RestoreAllocState(alloc194)
					d1 = snap162
					d2 = snap163
					d3 = snap164
					d4 = snap165
					d5 = snap166
					d6 = snap167
					d25 = snap168
					d26 = snap169
					d27 = snap170
					d28 = snap171
					d29 = snap172
					d30 = snap173
					d31 = snap174
					d32 = snap175
					d33 = snap176
					d34 = snap177
					d35 = snap178
					d36 = snap179
					d38 = snap180
					d40 = snap181
					d62 = snap182
					d65 = snap183
					d90 = snap184
					d91 = snap185
					d92 = snap186
					d93 = snap187
					d96 = snap188
					d155 = snap189
					d156 = snap190
					d157 = snap191
					d158 = snap192
					d159 = snap193
					ps195 := PhiState{General: true}
					ps195.OverlayValues = make([]JITValueDesc, 160)
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
					ps195.OverlayValues[31] = d31
					ps195.OverlayValues[32] = d32
					ps195.OverlayValues[33] = d33
					ps195.OverlayValues[34] = d34
					ps195.OverlayValues[35] = d35
					ps195.OverlayValues[36] = d36
					ps195.OverlayValues[38] = d38
					ps195.OverlayValues[40] = d40
					ps195.OverlayValues[62] = d62
					ps195.OverlayValues[65] = d65
					ps195.OverlayValues[90] = d90
					ps195.OverlayValues[91] = d91
					ps195.OverlayValues[92] = d92
					ps195.OverlayValues[93] = d93
					ps195.OverlayValues[96] = d96
					ps195.OverlayValues[155] = d155
					ps195.OverlayValues[156] = d156
					ps195.OverlayValues[157] = d157
					ps195.OverlayValues[158] = d158
					ps195.OverlayValues[159] = d159
					ps196 := PhiState{General: true}
					ps196.OverlayValues = make([]JITValueDesc, 160)
					ps196.OverlayValues[1] = d1
					ps196.OverlayValues[2] = d2
					ps196.OverlayValues[3] = d3
					ps196.OverlayValues[4] = d4
					ps196.OverlayValues[5] = d5
					ps196.OverlayValues[6] = d6
					ps196.OverlayValues[25] = d25
					ps196.OverlayValues[26] = d26
					ps196.OverlayValues[27] = d27
					ps196.OverlayValues[28] = d28
					ps196.OverlayValues[29] = d29
					ps196.OverlayValues[30] = d30
					ps196.OverlayValues[31] = d31
					ps196.OverlayValues[32] = d32
					ps196.OverlayValues[33] = d33
					ps196.OverlayValues[34] = d34
					ps196.OverlayValues[35] = d35
					ps196.OverlayValues[36] = d36
					ps196.OverlayValues[38] = d38
					ps196.OverlayValues[40] = d40
					ps196.OverlayValues[62] = d62
					ps196.OverlayValues[65] = d65
					ps196.OverlayValues[90] = d90
					ps196.OverlayValues[91] = d91
					ps196.OverlayValues[92] = d92
					ps196.OverlayValues[93] = d93
					ps196.OverlayValues[96] = d96
					ps196.OverlayValues[155] = d155
					ps196.OverlayValues[156] = d156
					ps196.OverlayValues[157] = d157
					ps196.OverlayValues[158] = d158
					ps196.OverlayValues[159] = d159
					snap197 := d1
					snap198 := d2
					snap199 := d3
					snap200 := d4
					snap201 := d5
					snap202 := d6
					snap203 := d25
					snap204 := d26
					snap205 := d27
					snap206 := d28
					snap207 := d29
					snap208 := d30
					snap209 := d31
					snap210 := d32
					snap211 := d33
					snap212 := d34
					snap213 := d35
					snap214 := d36
					snap215 := d38
					snap216 := d40
					snap217 := d62
					snap218 := d65
					snap219 := d90
					snap220 := d91
					snap221 := d92
					snap222 := d93
					snap223 := d96
					snap224 := d155
					snap225 := d156
					snap226 := d157
					snap227 := d158
					snap228 := d159
					alloc229 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps196)
					}
					ctx.RestoreAllocState(alloc229)
					d1 = snap197
					d2 = snap198
					d3 = snap199
					d4 = snap200
					d5 = snap201
					d6 = snap202
					d25 = snap203
					d26 = snap204
					d27 = snap205
					d28 = snap206
					d29 = snap207
					d30 = snap208
					d31 = snap209
					d32 = snap210
					d33 = snap211
					d34 = snap212
					d35 = snap213
					d36 = snap214
					d38 = snap215
					d40 = snap216
					d62 = snap217
					d65 = snap218
					d90 = snap219
					d91 = snap220
					d92 = snap221
					d93 = snap222
					d96 = snap223
					d155 = snap224
					d156 = snap225
					d157 = snap226
					d158 = snap227
					d159 = snap228
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps195)
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					ctx.ReclaimUntrackedRegs()
					d230 = args[2]
					d230.ID = 0
					ctx.EnsureDesc(&d230)
					d231 = d230
					_ = d231
					ctx.StabilizeDescForControlFlow(&d231)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d232 JITValueDesc
					if d231.Loc == LocImm {
						d232 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d231.Imm.Int())}
					} else if d231.Type == tagInt && d231.Loc == LocRegPair {
						ctx.FreeReg(d231.Reg)
						d232 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d231.Reg2}
						ctx.BindReg(d231.Reg2, &d232)
						ctx.BindReg(d231.Reg2, &d232)
					} else if d231.Type == tagInt && d231.Loc == LocReg {
						d232 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d231.Reg}
						ctx.BindReg(d231.Reg, &d232)
						ctx.BindReg(d231.Reg, &d232)
					} else {
						d232 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d231}, 1)
						d232.Type = tagInt
						ctx.BindReg(d232.Reg, &d232)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d232)
					ctx.EnsureDesc(&d232)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d232)
					ctx.StabilizeDescForControlFlow(&d232)
					ctx.FreeDesc(&d230)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d232)
					ctx.EnsureDescsTogether(&d1, &d232)
					var d234 JITValueDesc
					if d1.Loc == LocImm && d232.Loc == LocImm {
						d234 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d232.Imm.Int())}
					} else if d232.Loc == LocImm && d232.Imm.Int() == 0 {
						r5 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r5, d1.Reg)
						d234 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d234)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d234 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d232.Reg}
						ctx.BindReg(d232.Reg, &d234)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d232.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d232.Reg)
						d234 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d234)
					} else if d232.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d232.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d232.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d234 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d234)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg, d232.Reg)
						ctx.EmitMovRegReg(r6, d1.Reg)
						ctx.EmitAddInt64(r6, d232.Reg)
						d234 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d234)
					}
					if d234.Loc == LocReg && d1.Loc == LocReg && d234.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d234)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDescsTogether(&d234, &d29)
					var d235 JITValueDesc
					if d234.Loc == LocImm && d29.Loc == LocImm {
						d235 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d234.Imm.Int() > d29.Imm.Int())}
					} else if d29.Loc == LocImm {
						r7 := ctx.AllocReg()
						if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d234.Reg, int32(d29.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d29.Imm.Int()))
							ctx.EmitCmpInt64(d234.Reg, RegR11)
						}
						d235 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedGreater}
						ctx.BindReg(r7, &d235)
					} else if d234.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d234.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d29.Reg)
						d235 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedGreater}
						ctx.BindReg(r8, &d235)
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpInt64(d234.Reg, d29.Reg)
						d235 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedGreater}
						ctx.BindReg(r9, &d235)
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
							ps237.OverlayValues[6] = d6
							ps237.OverlayValues[25] = d25
							ps237.OverlayValues[26] = d26
							ps237.OverlayValues[27] = d27
							ps237.OverlayValues[28] = d28
							ps237.OverlayValues[29] = d29
							ps237.OverlayValues[30] = d30
							ps237.OverlayValues[31] = d31
							ps237.OverlayValues[32] = d32
							ps237.OverlayValues[33] = d33
							ps237.OverlayValues[34] = d34
							ps237.OverlayValues[35] = d35
							ps237.OverlayValues[36] = d36
							ps237.OverlayValues[38] = d38
							ps237.OverlayValues[40] = d40
							ps237.OverlayValues[62] = d62
							ps237.OverlayValues[65] = d65
							ps237.OverlayValues[90] = d90
							ps237.OverlayValues[91] = d91
							ps237.OverlayValues[92] = d92
							ps237.OverlayValues[93] = d93
							ps237.OverlayValues[96] = d96
							ps237.OverlayValues[155] = d155
							ps237.OverlayValues[156] = d156
							ps237.OverlayValues[157] = d157
							ps237.OverlayValues[158] = d158
							ps237.OverlayValues[159] = d159
							ps237.OverlayValues[230] = d230
							ps237.OverlayValues[231] = d231
							ps237.OverlayValues[232] = d232
							ps237.OverlayValues[233] = d233
							ps237.OverlayValues[234] = d234
							ps237.OverlayValues[235] = d235
							ps237.OverlayValues[236] = d236
							return bbs[9].RenderPS(ps237)
						}
						if ps.General {
							ctx.SyncDesc(&d232)
							if d232.Loc == LocReg {
								ctx.ProtectReg(d232.Reg)
							} else if d232.Loc == LocRegPair {
								ctx.ProtectReg(d232.Reg)
								ctx.ProtectReg(d232.Reg2)
							}
							d238 = d232
							if d238.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d238)
							ctx.EmitStoreToStack(d238, int32(bbs[10].PhiBase)+int32(0))
							if d232.Loc == LocReg {
								ctx.UnprotectReg(d232.Reg)
							} else if d232.Loc == LocRegPair {
								ctx.UnprotectReg(d232.Reg)
								ctx.UnprotectReg(d232.Reg2)
							}
						}
						ps239 := PhiState{General: ps.General}
						ps239.OverlayValues = make([]JITValueDesc, 239)
						ps239.OverlayValues[1] = d1
						ps239.OverlayValues[2] = d2
						ps239.OverlayValues[3] = d3
						ps239.OverlayValues[4] = d4
						ps239.OverlayValues[5] = d5
						ps239.OverlayValues[6] = d6
						ps239.OverlayValues[25] = d25
						ps239.OverlayValues[26] = d26
						ps239.OverlayValues[27] = d27
						ps239.OverlayValues[28] = d28
						ps239.OverlayValues[29] = d29
						ps239.OverlayValues[30] = d30
						ps239.OverlayValues[31] = d31
						ps239.OverlayValues[32] = d32
						ps239.OverlayValues[33] = d33
						ps239.OverlayValues[34] = d34
						ps239.OverlayValues[35] = d35
						ps239.OverlayValues[36] = d36
						ps239.OverlayValues[38] = d38
						ps239.OverlayValues[40] = d40
						ps239.OverlayValues[62] = d62
						ps239.OverlayValues[65] = d65
						ps239.OverlayValues[90] = d90
						ps239.OverlayValues[91] = d91
						ps239.OverlayValues[92] = d92
						ps239.OverlayValues[93] = d93
						ps239.OverlayValues[96] = d96
						ps239.OverlayValues[155] = d155
						ps239.OverlayValues[156] = d156
						ps239.OverlayValues[157] = d157
						ps239.OverlayValues[158] = d158
						ps239.OverlayValues[159] = d159
						ps239.OverlayValues[230] = d230
						ps239.OverlayValues[231] = d231
						ps239.OverlayValues[232] = d232
						ps239.OverlayValues[233] = d233
						ps239.OverlayValues[234] = d234
						ps239.OverlayValues[235] = d235
						ps239.OverlayValues[236] = d236
						ps239.OverlayValues[238] = d238
						ps239.PhiValues = make([]JITValueDesc, 1)
						d240 = d232
						ps239.PhiValues[0] = d240
						return bbs[10].RenderPS(ps239)
					}
					if !ps.General {
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitJump(d236.Condition, lbl24)
					ctx.EmitJmp(lbl25)
					snap241 := d1
					snap242 := d2
					snap243 := d3
					snap244 := d4
					snap245 := d5
					snap246 := d6
					snap247 := d25
					snap248 := d26
					snap249 := d27
					snap250 := d28
					snap251 := d29
					snap252 := d30
					snap253 := d31
					snap254 := d32
					snap255 := d33
					snap256 := d34
					snap257 := d35
					snap258 := d36
					snap259 := d38
					snap260 := d40
					snap261 := d62
					snap262 := d65
					snap263 := d90
					snap264 := d91
					snap265 := d92
					snap266 := d93
					snap267 := d96
					snap268 := d155
					snap269 := d156
					snap270 := d157
					snap271 := d158
					snap272 := d159
					snap273 := d230
					snap274 := d231
					snap275 := d232
					snap276 := d233
					snap277 := d234
					snap278 := d235
					snap279 := d236
					snap280 := d238
					snap281 := d240
					alloc282 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl10)
					ctx.RestoreAllocState(alloc282)
					d1 = snap241
					d2 = snap242
					d3 = snap243
					d4 = snap244
					d5 = snap245
					d6 = snap246
					d25 = snap247
					d26 = snap248
					d27 = snap249
					d28 = snap250
					d29 = snap251
					d30 = snap252
					d31 = snap253
					d32 = snap254
					d33 = snap255
					d34 = snap256
					d35 = snap257
					d36 = snap258
					d38 = snap259
					d40 = snap260
					d62 = snap261
					d65 = snap262
					d90 = snap263
					d91 = snap264
					d92 = snap265
					d93 = snap266
					d96 = snap267
					d155 = snap268
					d156 = snap269
					d157 = snap270
					d158 = snap271
					d159 = snap272
					d230 = snap273
					d231 = snap274
					d232 = snap275
					d233 = snap276
					d234 = snap277
					d235 = snap278
					d236 = snap279
					d238 = snap280
					d240 = snap281
					ctx.MarkLabel(lbl25)
					ctx.SyncDesc(&d232)
					if d232.Loc == LocReg {
						ctx.ProtectReg(d232.Reg)
					} else if d232.Loc == LocRegPair {
						ctx.ProtectReg(d232.Reg)
						ctx.ProtectReg(d232.Reg2)
					}
					d283 = d232
					if d283.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d283)
					ctx.EmitStoreToStack(d283, int32(bbs[10].PhiBase)+int32(0))
					if d232.Loc == LocReg {
						ctx.UnprotectReg(d232.Reg)
					} else if d232.Loc == LocRegPair {
						ctx.UnprotectReg(d232.Reg)
						ctx.UnprotectReg(d232.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ctx.RestoreAllocState(alloc282)
					d1 = snap241
					d2 = snap242
					d3 = snap243
					d4 = snap244
					d5 = snap245
					d6 = snap246
					d25 = snap247
					d26 = snap248
					d27 = snap249
					d28 = snap250
					d29 = snap251
					d30 = snap252
					d31 = snap253
					d32 = snap254
					d33 = snap255
					d34 = snap256
					d35 = snap257
					d36 = snap258
					d38 = snap259
					d40 = snap260
					d62 = snap261
					d65 = snap262
					d90 = snap263
					d91 = snap264
					d92 = snap265
					d93 = snap266
					d96 = snap267
					d155 = snap268
					d156 = snap269
					d157 = snap270
					d158 = snap271
					d159 = snap272
					d230 = snap273
					d231 = snap274
					d232 = snap275
					d233 = snap276
					d234 = snap277
					d235 = snap278
					d236 = snap279
					d238 = snap280
					d240 = snap281
					ps284 := PhiState{General: true}
					ps284.OverlayValues = make([]JITValueDesc, 284)
					ps284.OverlayValues[1] = d1
					ps284.OverlayValues[2] = d2
					ps284.OverlayValues[3] = d3
					ps284.OverlayValues[4] = d4
					ps284.OverlayValues[5] = d5
					ps284.OverlayValues[6] = d6
					ps284.OverlayValues[25] = d25
					ps284.OverlayValues[26] = d26
					ps284.OverlayValues[27] = d27
					ps284.OverlayValues[28] = d28
					ps284.OverlayValues[29] = d29
					ps284.OverlayValues[30] = d30
					ps284.OverlayValues[31] = d31
					ps284.OverlayValues[32] = d32
					ps284.OverlayValues[33] = d33
					ps284.OverlayValues[34] = d34
					ps284.OverlayValues[35] = d35
					ps284.OverlayValues[36] = d36
					ps284.OverlayValues[38] = d38
					ps284.OverlayValues[40] = d40
					ps284.OverlayValues[62] = d62
					ps284.OverlayValues[65] = d65
					ps284.OverlayValues[90] = d90
					ps284.OverlayValues[91] = d91
					ps284.OverlayValues[92] = d92
					ps284.OverlayValues[93] = d93
					ps284.OverlayValues[96] = d96
					ps284.OverlayValues[155] = d155
					ps284.OverlayValues[156] = d156
					ps284.OverlayValues[157] = d157
					ps284.OverlayValues[158] = d158
					ps284.OverlayValues[159] = d159
					ps284.OverlayValues[230] = d230
					ps284.OverlayValues[231] = d231
					ps284.OverlayValues[232] = d232
					ps284.OverlayValues[233] = d233
					ps284.OverlayValues[234] = d234
					ps284.OverlayValues[235] = d235
					ps284.OverlayValues[236] = d236
					ps284.OverlayValues[238] = d238
					ps284.OverlayValues[240] = d240
					ps284.OverlayValues[283] = d283
					ps285 := PhiState{General: true}
					ps285.OverlayValues = make([]JITValueDesc, 284)
					ps285.OverlayValues[1] = d1
					ps285.OverlayValues[2] = d2
					ps285.OverlayValues[3] = d3
					ps285.OverlayValues[4] = d4
					ps285.OverlayValues[5] = d5
					ps285.OverlayValues[6] = d6
					ps285.OverlayValues[25] = d25
					ps285.OverlayValues[26] = d26
					ps285.OverlayValues[27] = d27
					ps285.OverlayValues[28] = d28
					ps285.OverlayValues[29] = d29
					ps285.OverlayValues[30] = d30
					ps285.OverlayValues[31] = d31
					ps285.OverlayValues[32] = d32
					ps285.OverlayValues[33] = d33
					ps285.OverlayValues[34] = d34
					ps285.OverlayValues[35] = d35
					ps285.OverlayValues[36] = d36
					ps285.OverlayValues[38] = d38
					ps285.OverlayValues[40] = d40
					ps285.OverlayValues[62] = d62
					ps285.OverlayValues[65] = d65
					ps285.OverlayValues[90] = d90
					ps285.OverlayValues[91] = d91
					ps285.OverlayValues[92] = d92
					ps285.OverlayValues[93] = d93
					ps285.OverlayValues[96] = d96
					ps285.OverlayValues[155] = d155
					ps285.OverlayValues[156] = d156
					ps285.OverlayValues[157] = d157
					ps285.OverlayValues[158] = d158
					ps285.OverlayValues[159] = d159
					ps285.OverlayValues[230] = d230
					ps285.OverlayValues[231] = d231
					ps285.OverlayValues[232] = d232
					ps285.OverlayValues[233] = d233
					ps285.OverlayValues[234] = d234
					ps285.OverlayValues[235] = d235
					ps285.OverlayValues[236] = d236
					ps285.OverlayValues[238] = d238
					ps285.OverlayValues[240] = d240
					ps285.OverlayValues[283] = d283
					ps285.PhiValues = make([]JITValueDesc, 1)
					d286 = d232
					ps285.PhiValues[0] = d286
					snap287 := d1
					snap288 := d2
					snap289 := d3
					snap290 := d4
					snap291 := d5
					snap292 := d6
					snap293 := d25
					snap294 := d26
					snap295 := d27
					snap296 := d28
					snap297 := d29
					snap298 := d30
					snap299 := d31
					snap300 := d32
					snap301 := d33
					snap302 := d34
					snap303 := d35
					snap304 := d36
					snap305 := d38
					snap306 := d40
					snap307 := d62
					snap308 := d65
					snap309 := d90
					snap310 := d91
					snap311 := d92
					snap312 := d93
					snap313 := d96
					snap314 := d155
					snap315 := d156
					snap316 := d157
					snap317 := d158
					snap318 := d159
					snap319 := d230
					snap320 := d231
					snap321 := d232
					snap322 := d233
					snap323 := d234
					snap324 := d235
					snap325 := d236
					snap326 := d238
					snap327 := d240
					snap328 := d283
					snap329 := d286
					alloc330 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps285)
					}
					ctx.RestoreAllocState(alloc330)
					d1 = snap287
					d2 = snap288
					d3 = snap289
					d4 = snap290
					d5 = snap291
					d6 = snap292
					d25 = snap293
					d26 = snap294
					d27 = snap295
					d28 = snap296
					d29 = snap297
					d30 = snap298
					d31 = snap299
					d32 = snap300
					d33 = snap301
					d34 = snap302
					d35 = snap303
					d36 = snap304
					d38 = snap305
					d40 = snap306
					d62 = snap307
					d65 = snap308
					d90 = snap309
					d91 = snap310
					d92 = snap311
					d93 = snap312
					d96 = snap313
					d155 = snap314
					d156 = snap315
					d157 = snap316
					d158 = snap317
					d159 = snap318
					d230 = snap319
					d231 = snap320
					d232 = snap321
					d233 = snap322
					d234 = snap323
					d235 = snap324
					d236 = snap325
					d238 = snap326
					d240 = snap327
					d283 = snap328
					d286 = snap329
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps284)
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
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
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					var d331 JITValueDesc
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocRegTriple {
						d331 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2}
						ctx.BindReg(d27.Reg2, &d331)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d331)
					var d333 JITValueDesc
					if d331.Loc == LocImm && d1.Loc == LocImm {
						d333 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d331.Imm.Int() - d1.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d331.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d331.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d331.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r10, RegR11)
						} else {
							ctx.EmitSubInt64(r10, d1.Reg)
						}
						d333 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d333)
					}
					var d334 JITValueDesc
					r11 := ctx.EmitSliceDataAfterLow(&d27, &d1, 1)
					d334 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d334)
					ctx.BindReg(r11, &d334)
					var d335 JITValueDesc
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d334)
					ctx.EnsureDesc(&d334)
					if d334.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d334.Imm.Int()))
					} else {
						r12 = d334.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d333)
					ctx.EnsureDesc(&d333)
					if d333.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d333.Imm.Int()))
					} else {
						r13 = d333.Reg
					}
					ctx.ProtectReg(r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d335 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d335)
					ctx.BindReg(r13, &d335)
					ctx.BindReg(r12, &d335)
					ctx.BindReg(r13, &d335)
					ctx.EnsureDesc(&d335)
					d336 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d335}, 2)
					ctx.EmitMovPairToResult(&d336, &result)
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
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
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d29, &d1)
					var d337 JITValueDesc
					if d29.Loc == LocImm && d1.Loc == LocImm {
						d337 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int() - d1.Imm.Int())}
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						r14 := ctx.AllocRegExcept(d29.Reg)
						ctx.EmitMovRegReg(r14, d29.Reg)
						d337 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d337)
					} else if d29.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
						ctx.EmitSubInt64(scratch, d1.Reg)
						d337 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d337)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d29.Reg)
						ctx.EmitMovRegReg(scratch, d29.Reg)
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d337 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d337)
					} else {
						r15 := ctx.AllocRegExcept(d29.Reg, d1.Reg)
						ctx.EmitMovRegReg(r15, d29.Reg)
						ctx.EmitSubInt64(r15, d1.Reg)
						d337 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d337)
					}
					if d337.Loc == LocReg && d29.Loc == LocReg && d337.Reg == d29.Reg {
						ctx.TransferReg(d29.Reg)
						d29.Loc = LocNone
					}
					ctx.EnsureDesc(&d337)
					ctx.EmitStoreToStack(d337, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d337)
					if ps.General {
					}
					ps338 := PhiState{General: ps.General}
					ps338.OverlayValues = make([]JITValueDesc, 338)
					ps338.OverlayValues[1] = d1
					ps338.OverlayValues[2] = d2
					ps338.OverlayValues[3] = d3
					ps338.OverlayValues[4] = d4
					ps338.OverlayValues[5] = d5
					ps338.OverlayValues[6] = d6
					ps338.OverlayValues[25] = d25
					ps338.OverlayValues[26] = d26
					ps338.OverlayValues[27] = d27
					ps338.OverlayValues[28] = d28
					ps338.OverlayValues[29] = d29
					ps338.OverlayValues[30] = d30
					ps338.OverlayValues[31] = d31
					ps338.OverlayValues[32] = d32
					ps338.OverlayValues[33] = d33
					ps338.OverlayValues[34] = d34
					ps338.OverlayValues[35] = d35
					ps338.OverlayValues[36] = d36
					ps338.OverlayValues[38] = d38
					ps338.OverlayValues[40] = d40
					ps338.OverlayValues[62] = d62
					ps338.OverlayValues[65] = d65
					ps338.OverlayValues[90] = d90
					ps338.OverlayValues[91] = d91
					ps338.OverlayValues[92] = d92
					ps338.OverlayValues[93] = d93
					ps338.OverlayValues[96] = d96
					ps338.OverlayValues[155] = d155
					ps338.OverlayValues[156] = d156
					ps338.OverlayValues[157] = d157
					ps338.OverlayValues[158] = d158
					ps338.OverlayValues[159] = d159
					ps338.OverlayValues[230] = d230
					ps338.OverlayValues[231] = d231
					ps338.OverlayValues[232] = d232
					ps338.OverlayValues[233] = d233
					ps338.OverlayValues[234] = d234
					ps338.OverlayValues[235] = d235
					ps338.OverlayValues[236] = d236
					ps338.OverlayValues[238] = d238
					ps338.OverlayValues[240] = d240
					ps338.OverlayValues[283] = d283
					ps338.OverlayValues[286] = d286
					ps338.OverlayValues[331] = d331
					ps338.OverlayValues[332] = d332
					ps338.OverlayValues[333] = d333
					ps338.OverlayValues[334] = d334
					ps338.OverlayValues[335] = d335
					ps338.OverlayValues[336] = d336
					ps338.OverlayValues[337] = d337
					ps338.PhiValues = make([]JITValueDesc, 1)
					if ps338.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps338)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d339 := ps.PhiValues[0]
							ctx.EnsureDesc(&d339)
							ctx.EmitStoreToStack(d339, int32(bbs[10].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
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
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d2)
					var d340 JITValueDesc
					if d2.Loc == LocImm {
						d340 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 0)}
					} else {
						r16 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						d340 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
						ctx.BindReg(r16, &d340)
					}
					d341 = d340
					ctx.EnsureDesc(&d341)
					if d341.Loc != LocImm && d341.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d341.Loc == LocImm {
						if d341.Imm.Bool() {
							if ps.General {
							}
							ps342 := PhiState{General: ps.General}
							ps342.OverlayValues = make([]JITValueDesc, 342)
							ps342.OverlayValues[1] = d1
							ps342.OverlayValues[2] = d2
							ps342.OverlayValues[3] = d3
							ps342.OverlayValues[4] = d4
							ps342.OverlayValues[5] = d5
							ps342.OverlayValues[6] = d6
							ps342.OverlayValues[25] = d25
							ps342.OverlayValues[26] = d26
							ps342.OverlayValues[27] = d27
							ps342.OverlayValues[28] = d28
							ps342.OverlayValues[29] = d29
							ps342.OverlayValues[30] = d30
							ps342.OverlayValues[31] = d31
							ps342.OverlayValues[32] = d32
							ps342.OverlayValues[33] = d33
							ps342.OverlayValues[34] = d34
							ps342.OverlayValues[35] = d35
							ps342.OverlayValues[36] = d36
							ps342.OverlayValues[38] = d38
							ps342.OverlayValues[40] = d40
							ps342.OverlayValues[62] = d62
							ps342.OverlayValues[65] = d65
							ps342.OverlayValues[90] = d90
							ps342.OverlayValues[91] = d91
							ps342.OverlayValues[92] = d92
							ps342.OverlayValues[93] = d93
							ps342.OverlayValues[96] = d96
							ps342.OverlayValues[155] = d155
							ps342.OverlayValues[156] = d156
							ps342.OverlayValues[157] = d157
							ps342.OverlayValues[158] = d158
							ps342.OverlayValues[159] = d159
							ps342.OverlayValues[230] = d230
							ps342.OverlayValues[231] = d231
							ps342.OverlayValues[232] = d232
							ps342.OverlayValues[233] = d233
							ps342.OverlayValues[234] = d234
							ps342.OverlayValues[235] = d235
							ps342.OverlayValues[236] = d236
							ps342.OverlayValues[238] = d238
							ps342.OverlayValues[240] = d240
							ps342.OverlayValues[283] = d283
							ps342.OverlayValues[286] = d286
							ps342.OverlayValues[331] = d331
							ps342.OverlayValues[332] = d332
							ps342.OverlayValues[333] = d333
							ps342.OverlayValues[334] = d334
							ps342.OverlayValues[335] = d335
							ps342.OverlayValues[336] = d336
							ps342.OverlayValues[337] = d337
							ps342.OverlayValues[339] = d339
							ps342.OverlayValues[340] = d340
							ps342.OverlayValues[341] = d341
							return bbs[11].RenderPS(ps342)
						}
						if ps.General {
						}
						ps343 := PhiState{General: ps.General}
						ps343.OverlayValues = make([]JITValueDesc, 342)
						ps343.OverlayValues[1] = d1
						ps343.OverlayValues[2] = d2
						ps343.OverlayValues[3] = d3
						ps343.OverlayValues[4] = d4
						ps343.OverlayValues[5] = d5
						ps343.OverlayValues[6] = d6
						ps343.OverlayValues[25] = d25
						ps343.OverlayValues[26] = d26
						ps343.OverlayValues[27] = d27
						ps343.OverlayValues[28] = d28
						ps343.OverlayValues[29] = d29
						ps343.OverlayValues[30] = d30
						ps343.OverlayValues[31] = d31
						ps343.OverlayValues[32] = d32
						ps343.OverlayValues[33] = d33
						ps343.OverlayValues[34] = d34
						ps343.OverlayValues[35] = d35
						ps343.OverlayValues[36] = d36
						ps343.OverlayValues[38] = d38
						ps343.OverlayValues[40] = d40
						ps343.OverlayValues[62] = d62
						ps343.OverlayValues[65] = d65
						ps343.OverlayValues[90] = d90
						ps343.OverlayValues[91] = d91
						ps343.OverlayValues[92] = d92
						ps343.OverlayValues[93] = d93
						ps343.OverlayValues[96] = d96
						ps343.OverlayValues[155] = d155
						ps343.OverlayValues[156] = d156
						ps343.OverlayValues[157] = d157
						ps343.OverlayValues[158] = d158
						ps343.OverlayValues[159] = d159
						ps343.OverlayValues[230] = d230
						ps343.OverlayValues[231] = d231
						ps343.OverlayValues[232] = d232
						ps343.OverlayValues[233] = d233
						ps343.OverlayValues[234] = d234
						ps343.OverlayValues[235] = d235
						ps343.OverlayValues[236] = d236
						ps343.OverlayValues[238] = d238
						ps343.OverlayValues[240] = d240
						ps343.OverlayValues[283] = d283
						ps343.OverlayValues[286] = d286
						ps343.OverlayValues[331] = d331
						ps343.OverlayValues[332] = d332
						ps343.OverlayValues[333] = d333
						ps343.OverlayValues[334] = d334
						ps343.OverlayValues[335] = d335
						ps343.OverlayValues[336] = d336
						ps343.OverlayValues[337] = d337
						ps343.OverlayValues[339] = d339
						ps343.OverlayValues[340] = d340
						ps343.OverlayValues[341] = d341
						return bbs[12].RenderPS(ps343)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d344 := ps.PhiValues[0]
							ctx.EnsureDesc(&d344)
							ctx.EmitStoreToStack(d344, int32(bbs[10].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitJump(d341.Condition, lbl26)
					ctx.EmitJmp(lbl27)
					snap345 := d1
					snap346 := d2
					snap347 := d3
					snap348 := d4
					snap349 := d5
					snap350 := d6
					snap351 := d25
					snap352 := d26
					snap353 := d27
					snap354 := d28
					snap355 := d29
					snap356 := d30
					snap357 := d31
					snap358 := d32
					snap359 := d33
					snap360 := d34
					snap361 := d35
					snap362 := d36
					snap363 := d38
					snap364 := d40
					snap365 := d62
					snap366 := d65
					snap367 := d90
					snap368 := d91
					snap369 := d92
					snap370 := d93
					snap371 := d96
					snap372 := d155
					snap373 := d156
					snap374 := d157
					snap375 := d158
					snap376 := d159
					snap377 := d230
					snap378 := d231
					snap379 := d232
					snap380 := d233
					snap381 := d234
					snap382 := d235
					snap383 := d236
					snap384 := d238
					snap385 := d240
					snap386 := d283
					snap387 := d286
					snap388 := d331
					snap389 := d332
					snap390 := d333
					snap391 := d334
					snap392 := d335
					snap393 := d336
					snap394 := d337
					snap395 := d339
					snap396 := d340
					snap397 := d341
					snap398 := d344
					alloc399 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl12)
					ctx.RestoreAllocState(alloc399)
					d1 = snap345
					d2 = snap346
					d3 = snap347
					d4 = snap348
					d5 = snap349
					d6 = snap350
					d25 = snap351
					d26 = snap352
					d27 = snap353
					d28 = snap354
					d29 = snap355
					d30 = snap356
					d31 = snap357
					d32 = snap358
					d33 = snap359
					d34 = snap360
					d35 = snap361
					d36 = snap362
					d38 = snap363
					d40 = snap364
					d62 = snap365
					d65 = snap366
					d90 = snap367
					d91 = snap368
					d92 = snap369
					d93 = snap370
					d96 = snap371
					d155 = snap372
					d156 = snap373
					d157 = snap374
					d158 = snap375
					d159 = snap376
					d230 = snap377
					d231 = snap378
					d232 = snap379
					d233 = snap380
					d234 = snap381
					d235 = snap382
					d236 = snap383
					d238 = snap384
					d240 = snap385
					d283 = snap386
					d286 = snap387
					d331 = snap388
					d332 = snap389
					d333 = snap390
					d334 = snap391
					d335 = snap392
					d336 = snap393
					d337 = snap394
					d339 = snap395
					d340 = snap396
					d341 = snap397
					d344 = snap398
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl13)
					ctx.RestoreAllocState(alloc399)
					d1 = snap345
					d2 = snap346
					d3 = snap347
					d4 = snap348
					d5 = snap349
					d6 = snap350
					d25 = snap351
					d26 = snap352
					d27 = snap353
					d28 = snap354
					d29 = snap355
					d30 = snap356
					d31 = snap357
					d32 = snap358
					d33 = snap359
					d34 = snap360
					d35 = snap361
					d36 = snap362
					d38 = snap363
					d40 = snap364
					d62 = snap365
					d65 = snap366
					d90 = snap367
					d91 = snap368
					d92 = snap369
					d93 = snap370
					d96 = snap371
					d155 = snap372
					d156 = snap373
					d157 = snap374
					d158 = snap375
					d159 = snap376
					d230 = snap377
					d231 = snap378
					d232 = snap379
					d233 = snap380
					d234 = snap381
					d235 = snap382
					d236 = snap383
					d238 = snap384
					d240 = snap385
					d283 = snap386
					d286 = snap387
					d331 = snap388
					d332 = snap389
					d333 = snap390
					d334 = snap391
					d335 = snap392
					d336 = snap393
					d337 = snap394
					d339 = snap395
					d340 = snap396
					d341 = snap397
					d344 = snap398
					ps400 := PhiState{General: true}
					ps400.OverlayValues = make([]JITValueDesc, 345)
					ps400.OverlayValues[1] = d1
					ps400.OverlayValues[2] = d2
					ps400.OverlayValues[3] = d3
					ps400.OverlayValues[4] = d4
					ps400.OverlayValues[5] = d5
					ps400.OverlayValues[6] = d6
					ps400.OverlayValues[25] = d25
					ps400.OverlayValues[26] = d26
					ps400.OverlayValues[27] = d27
					ps400.OverlayValues[28] = d28
					ps400.OverlayValues[29] = d29
					ps400.OverlayValues[30] = d30
					ps400.OverlayValues[31] = d31
					ps400.OverlayValues[32] = d32
					ps400.OverlayValues[33] = d33
					ps400.OverlayValues[34] = d34
					ps400.OverlayValues[35] = d35
					ps400.OverlayValues[36] = d36
					ps400.OverlayValues[38] = d38
					ps400.OverlayValues[40] = d40
					ps400.OverlayValues[62] = d62
					ps400.OverlayValues[65] = d65
					ps400.OverlayValues[90] = d90
					ps400.OverlayValues[91] = d91
					ps400.OverlayValues[92] = d92
					ps400.OverlayValues[93] = d93
					ps400.OverlayValues[96] = d96
					ps400.OverlayValues[155] = d155
					ps400.OverlayValues[156] = d156
					ps400.OverlayValues[157] = d157
					ps400.OverlayValues[158] = d158
					ps400.OverlayValues[159] = d159
					ps400.OverlayValues[230] = d230
					ps400.OverlayValues[231] = d231
					ps400.OverlayValues[232] = d232
					ps400.OverlayValues[233] = d233
					ps400.OverlayValues[234] = d234
					ps400.OverlayValues[235] = d235
					ps400.OverlayValues[236] = d236
					ps400.OverlayValues[238] = d238
					ps400.OverlayValues[240] = d240
					ps400.OverlayValues[283] = d283
					ps400.OverlayValues[286] = d286
					ps400.OverlayValues[331] = d331
					ps400.OverlayValues[332] = d332
					ps400.OverlayValues[333] = d333
					ps400.OverlayValues[334] = d334
					ps400.OverlayValues[335] = d335
					ps400.OverlayValues[336] = d336
					ps400.OverlayValues[337] = d337
					ps400.OverlayValues[339] = d339
					ps400.OverlayValues[340] = d340
					ps400.OverlayValues[341] = d341
					ps400.OverlayValues[344] = d344
					ps401 := PhiState{General: true}
					ps401.OverlayValues = make([]JITValueDesc, 345)
					ps401.OverlayValues[1] = d1
					ps401.OverlayValues[2] = d2
					ps401.OverlayValues[3] = d3
					ps401.OverlayValues[4] = d4
					ps401.OverlayValues[5] = d5
					ps401.OverlayValues[6] = d6
					ps401.OverlayValues[25] = d25
					ps401.OverlayValues[26] = d26
					ps401.OverlayValues[27] = d27
					ps401.OverlayValues[28] = d28
					ps401.OverlayValues[29] = d29
					ps401.OverlayValues[30] = d30
					ps401.OverlayValues[31] = d31
					ps401.OverlayValues[32] = d32
					ps401.OverlayValues[33] = d33
					ps401.OverlayValues[34] = d34
					ps401.OverlayValues[35] = d35
					ps401.OverlayValues[36] = d36
					ps401.OverlayValues[38] = d38
					ps401.OverlayValues[40] = d40
					ps401.OverlayValues[62] = d62
					ps401.OverlayValues[65] = d65
					ps401.OverlayValues[90] = d90
					ps401.OverlayValues[91] = d91
					ps401.OverlayValues[92] = d92
					ps401.OverlayValues[93] = d93
					ps401.OverlayValues[96] = d96
					ps401.OverlayValues[155] = d155
					ps401.OverlayValues[156] = d156
					ps401.OverlayValues[157] = d157
					ps401.OverlayValues[158] = d158
					ps401.OverlayValues[159] = d159
					ps401.OverlayValues[230] = d230
					ps401.OverlayValues[231] = d231
					ps401.OverlayValues[232] = d232
					ps401.OverlayValues[233] = d233
					ps401.OverlayValues[234] = d234
					ps401.OverlayValues[235] = d235
					ps401.OverlayValues[236] = d236
					ps401.OverlayValues[238] = d238
					ps401.OverlayValues[240] = d240
					ps401.OverlayValues[283] = d283
					ps401.OverlayValues[286] = d286
					ps401.OverlayValues[331] = d331
					ps401.OverlayValues[332] = d332
					ps401.OverlayValues[333] = d333
					ps401.OverlayValues[334] = d334
					ps401.OverlayValues[335] = d335
					ps401.OverlayValues[336] = d336
					ps401.OverlayValues[337] = d337
					ps401.OverlayValues[339] = d339
					ps401.OverlayValues[340] = d340
					ps401.OverlayValues[341] = d341
					ps401.OverlayValues[344] = d344
					snap402 := d1
					snap403 := d2
					snap404 := d3
					snap405 := d4
					snap406 := d5
					snap407 := d6
					snap408 := d25
					snap409 := d26
					snap410 := d27
					snap411 := d28
					snap412 := d29
					snap413 := d30
					snap414 := d31
					snap415 := d32
					snap416 := d33
					snap417 := d34
					snap418 := d35
					snap419 := d36
					snap420 := d38
					snap421 := d40
					snap422 := d62
					snap423 := d65
					snap424 := d90
					snap425 := d91
					snap426 := d92
					snap427 := d93
					snap428 := d96
					snap429 := d155
					snap430 := d156
					snap431 := d157
					snap432 := d158
					snap433 := d159
					snap434 := d230
					snap435 := d231
					snap436 := d232
					snap437 := d233
					snap438 := d234
					snap439 := d235
					snap440 := d236
					snap441 := d238
					snap442 := d240
					snap443 := d283
					snap444 := d286
					snap445 := d331
					snap446 := d332
					snap447 := d333
					snap448 := d334
					snap449 := d335
					snap450 := d336
					snap451 := d337
					snap452 := d339
					snap453 := d340
					snap454 := d341
					snap455 := d344
					alloc456 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps401)
					}
					ctx.RestoreAllocState(alloc456)
					d1 = snap402
					d2 = snap403
					d3 = snap404
					d4 = snap405
					d5 = snap406
					d6 = snap407
					d25 = snap408
					d26 = snap409
					d27 = snap410
					d28 = snap411
					d29 = snap412
					d30 = snap413
					d31 = snap414
					d32 = snap415
					d33 = snap416
					d34 = snap417
					d35 = snap418
					d36 = snap419
					d38 = snap420
					d40 = snap421
					d62 = snap422
					d65 = snap423
					d90 = snap424
					d91 = snap425
					d92 = snap426
					d93 = snap427
					d96 = snap428
					d155 = snap429
					d156 = snap430
					d157 = snap431
					d158 = snap432
					d159 = snap433
					d230 = snap434
					d231 = snap435
					d232 = snap436
					d233 = snap437
					d234 = snap438
					d235 = snap439
					d236 = snap440
					d238 = snap441
					d240 = snap442
					d283 = snap443
					d286 = snap444
					d331 = snap445
					d332 = snap446
					d333 = snap447
					d334 = snap448
					d335 = snap449
					d336 = snap450
					d337 = snap451
					d339 = snap452
					d340 = snap453
					d341 = snap454
					d344 = snap455
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps400)
					}
					return result
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
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
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
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
					if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
						d344 = ps.OverlayValues[344]
					}
					ctx.ReclaimUntrackedRegs()
					d457 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d458 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d457}, 2)
					ctx.EmitMovPairToResult(&d458, &result)
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
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
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
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
					if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
						d344 = ps.OverlayValues[344]
					}
					if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != LocNone {
						d457 = ps.OverlayValues[457]
					}
					if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != LocNone {
						d458 = ps.OverlayValues[458]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDescsTogether(&d1, &d2)
					var d459 JITValueDesc
					if d1.Loc == LocImm && d2.Loc == LocImm {
						d459 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d2.Imm.Int())}
					} else if d2.Loc == LocImm && d2.Imm.Int() == 0 {
						r17 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r17, d1.Reg)
						d459 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d459)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d459 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
						ctx.BindReg(d2.Reg, &d459)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d2.Reg)
						d459 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d459)
					} else if d2.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d459 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d459)
					} else {
						r18 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
						ctx.EmitMovRegReg(r18, d1.Reg)
						ctx.EmitAddInt64(r18, d2.Reg)
						d459 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d459)
					}
					if d459.Loc == LocReg && d1.Loc == LocReg && d459.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d459)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d459)
					var d461 JITValueDesc
					if d459.Loc == LocImm && d1.Loc == LocImm {
						d461 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d459.Imm.Int() - d1.Imm.Int())}
					} else {
						r19 := ctx.AllocReg()
						if d459.Loc == LocImm {
							ctx.EmitMovRegImm64(r19, uint64(d459.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r19, d459.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r19, RegR11)
						} else {
							ctx.EmitSubInt64(r19, d1.Reg)
						}
						d461 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d461)
					}
					var d462 JITValueDesc
					r20 := ctx.EmitSliceDataAfterLow(&d27, &d1, 1)
					d462 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
					ctx.BindReg(r20, &d462)
					ctx.BindReg(r20, &d462)
					var d463 JITValueDesc
					var r21 Reg
					var r22 Reg
					ctx.SyncDesc(&d462)
					ctx.EnsureDesc(&d462)
					if d462.Loc == LocImm {
						r21 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r21, uint64(d462.Imm.Int()))
					} else {
						r21 = d462.Reg
					}
					ctx.ProtectReg(r21)
					ctx.SyncDesc(&d461)
					ctx.EnsureDesc(&d461)
					if d461.Loc == LocImm {
						r22 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r22, uint64(d461.Imm.Int()))
					} else {
						r22 = d461.Reg
					}
					ctx.ProtectReg(r22)
					ctx.UnprotectReg(r22)
					ctx.UnprotectReg(r21)
					d463 = JITValueDesc{Loc: LocRegPair, Reg: r21, Reg2: r22}
					ctx.BindReg(r21, &d463)
					ctx.BindReg(r22, &d463)
					ctx.BindReg(r21, &d463)
					ctx.BindReg(r22, &d463)
					ctx.FreeDesc(&d459)
					ctx.EnsureDesc(&d463)
					d464 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d463}, 2)
					ctx.EmitMovPairToResult(&d464, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps465 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps465)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["simplify"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (Simplify arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(Simplify), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["strlen"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				declaration := declarations["strlike"]
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
				var d34 JITValueDesc
				_ = d34
				var d54 JITValueDesc
				_ = d54
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
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
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [6]BBDescriptor
				bbs[5].PhiBase = int32(phiBase0) + int32(0)
				bbs[5].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					snap8 := d1
					snap9 := d2
					snap10 := d3
					snap11 := d4
					snap12 := d5
					alloc13 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc13)
					d1 = snap8
					d2 = snap9
					d3 = snap10
					d4 = snap11
					d5 = snap12
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
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
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps15)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					ctx.ReclaimUntrackedRegs()
					d23 = args[0]
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
					ctx.StabilizeDescForControlFlow(&d24)
					ctx.FreeDesc(&d23)
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
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.FreeDesc(&d26)
					d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d29)
					var d30 JITValueDesc
					if d29.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d29.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d29.Reg, 2)
						d30 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d30)
					}
					ctx.FreeDesc(&d29)
					d31 = d30
					ctx.EnsureDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							if ps.General {
							}
							ps32 := PhiState{General: ps.General}
							ps32.OverlayValues = make([]JITValueDesc, 32)
							ps32.OverlayValues[1] = d1
							ps32.OverlayValues[2] = d2
							ps32.OverlayValues[3] = d3
							ps32.OverlayValues[4] = d4
							ps32.OverlayValues[5] = d5
							ps32.OverlayValues[22] = d22
							ps32.OverlayValues[23] = d23
							ps32.OverlayValues[24] = d24
							ps32.OverlayValues[25] = d25
							ps32.OverlayValues[26] = d26
							ps32.OverlayValues[27] = d27
							ps32.OverlayValues[28] = d28
							ps32.OverlayValues[29] = d29
							ps32.OverlayValues[30] = d30
							ps32.OverlayValues[31] = d31
							return bbs[4].RenderPS(ps32)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
						}
						ps33 := PhiState{General: ps.General}
						ps33.OverlayValues = make([]JITValueDesc, 32)
						ps33.OverlayValues[1] = d1
						ps33.OverlayValues[2] = d2
						ps33.OverlayValues[3] = d3
						ps33.OverlayValues[4] = d4
						ps33.OverlayValues[5] = d5
						ps33.OverlayValues[22] = d22
						ps33.OverlayValues[23] = d23
						ps33.OverlayValues[24] = d24
						ps33.OverlayValues[25] = d25
						ps33.OverlayValues[26] = d26
						ps33.OverlayValues[27] = d27
						ps33.OverlayValues[28] = d28
						ps33.OverlayValues[29] = d29
						ps33.OverlayValues[30] = d30
						ps33.OverlayValues[31] = d31
						ps33.PhiValues = make([]JITValueDesc, 1)
						d34 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
						ps33.PhiValues[0] = d34
						return bbs[5].RenderPS(ps33)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d31.Condition, lbl9)
					ctx.EmitJmp(lbl10)
					snap35 := d1
					snap36 := d2
					snap37 := d3
					snap38 := d4
					snap39 := d5
					snap40 := d22
					snap41 := d23
					snap42 := d24
					snap43 := d25
					snap44 := d26
					snap45 := d27
					snap46 := d28
					snap47 := d29
					snap48 := d30
					snap49 := d31
					snap50 := d34
					alloc51 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc51)
					d1 = snap35
					d2 = snap36
					d3 = snap37
					d4 = snap38
					d5 = snap39
					d22 = snap40
					d23 = snap41
					d24 = snap42
					d25 = snap43
					d26 = snap44
					d27 = snap45
					d28 = snap46
					d29 = snap47
					d30 = snap48
					d31 = snap49
					d34 = snap50
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc51)
					d1 = snap35
					d2 = snap36
					d3 = snap37
					d4 = snap38
					d5 = snap39
					d22 = snap40
					d23 = snap41
					d24 = snap42
					d25 = snap43
					d26 = snap44
					d27 = snap45
					d28 = snap46
					d29 = snap47
					d30 = snap48
					d31 = snap49
					d34 = snap50
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 35)
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[26] = d26
					ps52.OverlayValues[27] = d27
					ps52.OverlayValues[28] = d28
					ps52.OverlayValues[29] = d29
					ps52.OverlayValues[30] = d30
					ps52.OverlayValues[31] = d31
					ps52.OverlayValues[34] = d34
					ps53 := PhiState{General: true}
					ps53.OverlayValues = make([]JITValueDesc, 35)
					ps53.OverlayValues[1] = d1
					ps53.OverlayValues[2] = d2
					ps53.OverlayValues[3] = d3
					ps53.OverlayValues[4] = d4
					ps53.OverlayValues[5] = d5
					ps53.OverlayValues[22] = d22
					ps53.OverlayValues[23] = d23
					ps53.OverlayValues[24] = d24
					ps53.OverlayValues[25] = d25
					ps53.OverlayValues[26] = d26
					ps53.OverlayValues[27] = d27
					ps53.OverlayValues[28] = d28
					ps53.OverlayValues[29] = d29
					ps53.OverlayValues[30] = d30
					ps53.OverlayValues[31] = d31
					ps53.OverlayValues[34] = d34
					ps53.PhiValues = make([]JITValueDesc, 1)
					d54 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}
					ps53.PhiValues[0] = d54
					snap55 := d1
					snap56 := d2
					snap57 := d3
					snap58 := d4
					snap59 := d5
					snap60 := d22
					snap61 := d23
					snap62 := d24
					snap63 := d25
					snap64 := d26
					snap65 := d27
					snap66 := d28
					snap67 := d29
					snap68 := d30
					snap69 := d31
					snap70 := d34
					snap71 := d54
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps53)
					}
					ctx.RestoreAllocState(alloc72)
					d1 = snap55
					d2 = snap56
					d3 = snap57
					d4 = snap58
					d5 = snap59
					d22 = snap60
					d23 = snap61
					d24 = snap62
					d25 = snap63
					d26 = snap64
					d27 = snap65
					d28 = snap66
					d29 = snap67
					d30 = snap68
					d31 = snap69
					d34 = snap70
					d54 = snap71
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps52)
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					ctx.ReclaimUntrackedRegs()
					d73 = args[1]
					d73.ID = 0
					d75 = d73
					d75.ID = 0
					d74 = ctx.EmitTagEqualsBorrowed(&d75, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d73)
					d76 = d74
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							if ps.General {
							}
							ps77 := PhiState{General: ps.General}
							ps77.OverlayValues = make([]JITValueDesc, 77)
							ps77.OverlayValues[1] = d1
							ps77.OverlayValues[2] = d2
							ps77.OverlayValues[3] = d3
							ps77.OverlayValues[4] = d4
							ps77.OverlayValues[5] = d5
							ps77.OverlayValues[22] = d22
							ps77.OverlayValues[23] = d23
							ps77.OverlayValues[24] = d24
							ps77.OverlayValues[25] = d25
							ps77.OverlayValues[26] = d26
							ps77.OverlayValues[27] = d27
							ps77.OverlayValues[28] = d28
							ps77.OverlayValues[29] = d29
							ps77.OverlayValues[30] = d30
							ps77.OverlayValues[31] = d31
							ps77.OverlayValues[34] = d34
							ps77.OverlayValues[54] = d54
							ps77.OverlayValues[73] = d73
							ps77.OverlayValues[74] = d74
							ps77.OverlayValues[75] = d75
							ps77.OverlayValues[76] = d76
							return bbs[1].RenderPS(ps77)
						}
						if ps.General {
						}
						ps78 := PhiState{General: ps.General}
						ps78.OverlayValues = make([]JITValueDesc, 77)
						ps78.OverlayValues[1] = d1
						ps78.OverlayValues[2] = d2
						ps78.OverlayValues[3] = d3
						ps78.OverlayValues[4] = d4
						ps78.OverlayValues[5] = d5
						ps78.OverlayValues[22] = d22
						ps78.OverlayValues[23] = d23
						ps78.OverlayValues[24] = d24
						ps78.OverlayValues[25] = d25
						ps78.OverlayValues[26] = d26
						ps78.OverlayValues[27] = d27
						ps78.OverlayValues[28] = d28
						ps78.OverlayValues[29] = d29
						ps78.OverlayValues[30] = d30
						ps78.OverlayValues[31] = d31
						ps78.OverlayValues[34] = d34
						ps78.OverlayValues[54] = d54
						ps78.OverlayValues[73] = d73
						ps78.OverlayValues[74] = d74
						ps78.OverlayValues[75] = d75
						ps78.OverlayValues[76] = d76
						return bbs[2].RenderPS(ps78)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					snap79 := d1
					snap80 := d2
					snap81 := d3
					snap82 := d4
					snap83 := d5
					snap84 := d22
					snap85 := d23
					snap86 := d24
					snap87 := d25
					snap88 := d26
					snap89 := d27
					snap90 := d28
					snap91 := d29
					snap92 := d30
					snap93 := d31
					snap94 := d34
					snap95 := d54
					snap96 := d73
					snap97 := d74
					snap98 := d75
					snap99 := d76
					alloc100 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc100)
					d1 = snap79
					d2 = snap80
					d3 = snap81
					d4 = snap82
					d5 = snap83
					d22 = snap84
					d23 = snap85
					d24 = snap86
					d25 = snap87
					d26 = snap88
					d27 = snap89
					d28 = snap90
					d29 = snap91
					d30 = snap92
					d31 = snap93
					d34 = snap94
					d54 = snap95
					d73 = snap96
					d74 = snap97
					d75 = snap98
					d76 = snap99
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc100)
					d1 = snap79
					d2 = snap80
					d3 = snap81
					d4 = snap82
					d5 = snap83
					d22 = snap84
					d23 = snap85
					d24 = snap86
					d25 = snap87
					d26 = snap88
					d27 = snap89
					d28 = snap90
					d29 = snap91
					d30 = snap92
					d31 = snap93
					d34 = snap94
					d54 = snap95
					d73 = snap96
					d74 = snap97
					d75 = snap98
					d76 = snap99
					ps101 := PhiState{General: true}
					ps101.OverlayValues = make([]JITValueDesc, 77)
					ps101.OverlayValues[1] = d1
					ps101.OverlayValues[2] = d2
					ps101.OverlayValues[3] = d3
					ps101.OverlayValues[4] = d4
					ps101.OverlayValues[5] = d5
					ps101.OverlayValues[22] = d22
					ps101.OverlayValues[23] = d23
					ps101.OverlayValues[24] = d24
					ps101.OverlayValues[25] = d25
					ps101.OverlayValues[26] = d26
					ps101.OverlayValues[27] = d27
					ps101.OverlayValues[28] = d28
					ps101.OverlayValues[29] = d29
					ps101.OverlayValues[30] = d30
					ps101.OverlayValues[31] = d31
					ps101.OverlayValues[34] = d34
					ps101.OverlayValues[54] = d54
					ps101.OverlayValues[73] = d73
					ps101.OverlayValues[74] = d74
					ps101.OverlayValues[75] = d75
					ps101.OverlayValues[76] = d76
					ps102 := PhiState{General: true}
					ps102.OverlayValues = make([]JITValueDesc, 77)
					ps102.OverlayValues[1] = d1
					ps102.OverlayValues[2] = d2
					ps102.OverlayValues[3] = d3
					ps102.OverlayValues[4] = d4
					ps102.OverlayValues[5] = d5
					ps102.OverlayValues[22] = d22
					ps102.OverlayValues[23] = d23
					ps102.OverlayValues[24] = d24
					ps102.OverlayValues[25] = d25
					ps102.OverlayValues[26] = d26
					ps102.OverlayValues[27] = d27
					ps102.OverlayValues[28] = d28
					ps102.OverlayValues[29] = d29
					ps102.OverlayValues[30] = d30
					ps102.OverlayValues[31] = d31
					ps102.OverlayValues[34] = d34
					ps102.OverlayValues[54] = d54
					ps102.OverlayValues[73] = d73
					ps102.OverlayValues[74] = d74
					ps102.OverlayValues[75] = d75
					ps102.OverlayValues[76] = d76
					snap103 := d1
					snap104 := d2
					snap105 := d3
					snap106 := d4
					snap107 := d5
					snap108 := d22
					snap109 := d23
					snap110 := d24
					snap111 := d25
					snap112 := d26
					snap113 := d27
					snap114 := d28
					snap115 := d29
					snap116 := d30
					snap117 := d31
					snap118 := d34
					snap119 := d54
					snap120 := d73
					snap121 := d74
					snap122 := d75
					snap123 := d76
					alloc124 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps102)
					}
					ctx.RestoreAllocState(alloc124)
					d1 = snap103
					d2 = snap104
					d3 = snap105
					d4 = snap106
					d5 = snap107
					d22 = snap108
					d23 = snap109
					d24 = snap110
					d25 = snap111
					d26 = snap112
					d27 = snap113
					d28 = snap114
					d29 = snap115
					d30 = snap116
					d31 = snap117
					d34 = snap118
					d54 = snap119
					d73 = snap120
					d74 = snap121
					d75 = snap122
					d76 = snap123
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps101)
					}
					return result
					ctx.FreeDesc(&d74)
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
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
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
					ctx.ReclaimUntrackedRegs()
					d125 = args[2]
					d125.ID = 0
					d127 = d125
					ctx.SyncDesc(&d127)
					if d127.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d127.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d127.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d127 = tmpScalar
					}
					d127 = JITPrepareScmerGoArg(ctx, d127)
					if d127.Loc != LocRegPair && d127.Loc != LocStackPair && d127.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d126 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d127}, 2)
					ctx.FreeDesc(&d125)
					ctx.EnsureDesc(&d126)
					ctx.EnsureDesc(&d126)
					ctx.EnsureDesc(&d126)
					if d126.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d126.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d126.Imm)
						ptrWord, _ := d126.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d126.Imm.String())))
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
					if d126.Loc != LocRegPair && d126.Loc != LocStackPair && d126.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d126)
					d128 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d126}, 2)
					d128.NoHeapPointer = false
					ctx.BindReg(d128.Reg, &d128)
					ctx.BindReg(d128.Reg2, &d128)
					ctx.StabilizeDescForControlFlow(&d128)
					if ps.General {
						ctx.SyncDesc(&d128)
						if d128.Loc == LocReg {
							ctx.ProtectReg(d128.Reg)
						} else if d128.Loc == LocRegPair {
							ctx.ProtectReg(d128.Reg)
							ctx.ProtectReg(d128.Reg2)
						}
						d129 = d128
						if d129.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d129)
						if d129.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d129, int32(bbs[5].PhiBase)+int32(0), 2)
						} else if d129.Loc == LocInputPair {
							ctx.EnsureDesc(&d129)
							ctx.EmitStoreScmerToStack(d129, int32(bbs[5].PhiBase)+int32(0))
						} else if d129.Loc == LocRegPair || d129.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d129, int32(bbs[5].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d129)
							ctx.EmitStoreToStack(d129, int32(bbs[5].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[5].PhiBase)+int32(0))+8)
						}
						if d128.Loc == LocReg {
							ctx.UnprotectReg(d128.Reg)
						} else if d128.Loc == LocRegPair {
							ctx.UnprotectReg(d128.Reg)
							ctx.UnprotectReg(d128.Reg2)
						}
					}
					ps130 := PhiState{General: ps.General}
					ps130.OverlayValues = make([]JITValueDesc, 130)
					ps130.OverlayValues[1] = d1
					ps130.OverlayValues[2] = d2
					ps130.OverlayValues[3] = d3
					ps130.OverlayValues[4] = d4
					ps130.OverlayValues[5] = d5
					ps130.OverlayValues[22] = d22
					ps130.OverlayValues[23] = d23
					ps130.OverlayValues[24] = d24
					ps130.OverlayValues[25] = d25
					ps130.OverlayValues[26] = d26
					ps130.OverlayValues[27] = d27
					ps130.OverlayValues[28] = d28
					ps130.OverlayValues[29] = d29
					ps130.OverlayValues[30] = d30
					ps130.OverlayValues[31] = d31
					ps130.OverlayValues[34] = d34
					ps130.OverlayValues[54] = d54
					ps130.OverlayValues[73] = d73
					ps130.OverlayValues[74] = d74
					ps130.OverlayValues[75] = d75
					ps130.OverlayValues[76] = d76
					ps130.OverlayValues[125] = d125
					ps130.OverlayValues[126] = d126
					ps130.OverlayValues[127] = d127
					ps130.OverlayValues[128] = d128
					ps130.OverlayValues[129] = d129
					ps130.PhiValues = make([]JITValueDesc, 1)
					d131 = d128
					ps130.PhiValues[0] = d131
					if ps130.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps130)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d132 := ps.PhiValues[0]
							ctx.EnsureDesc(&d132)
							ctx.EmitStoreScmerToStack(d132, int32(bbs[5].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
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
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
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
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg0)")
					}
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
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg1)")
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg2)")
					}
					ctx.SyncDesc(&d24)
					ctx.SyncDesc(&d27)
					ctx.SyncDesc(&d1)
					d133 = ctx.EmitGoCallScalar(GoFuncAddr(StrLikeCollation), []JITValueDesc{d24, d27, d1}, 1)
					d133.NoHeapPointer = true
					ctx.EmitAndRegImm32(d133.Reg, 1)
					d133.Type = tagBool
					ctx.BindReg(d133.Reg, &d133)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d133)
					if d133.Loc == LocImm {
						ctx.EmitMakeBool(result, d133)
					} else {
						ctx.EmitMovToReg(result.Reg2, d133)
						d134 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d134)
						if d133.Loc == LocReg && d133.Reg != result.Reg2 {
							ctx.FreeReg(d133.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				ps135 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps135)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCost: 28,
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
				declaration := declarations["strlike_cs"]
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
				var d30 JITValueDesc
				_ = d30
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [4]BBDescriptor
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
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
					d19 = args[0]
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
					d22 = args[1]
					d22.ID = 0
					d24 = d22
					ctx.SyncDesc(&d24)
					if d24.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d24.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d24.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d24 = tmpScalar
					}
					d24 = JITPrepareScmerGoArg(ctx, d24)
					if d24.Loc != LocRegPair && d24.Loc != LocStackPair && d24.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d24}, 2)
					ctx.FreeDesc(&d22)
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
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d23.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d23.Imm)
						ptrWord, _ := d23.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d23.Imm.String())))
						d23 = tmpPair
					} else if d23.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d23.Type, Reg: ctx.AllocRegExcept(d23.Reg), Reg2: ctx.AllocRegExcept(d23.Reg)}
						switch d23.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d23)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d23)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d23)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d23)
						d23 = tmpPair
					}
					if d23.Loc != LocRegPair && d23.Loc != LocStackPair && d23.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d23)
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d20, d23}, 1)
					d25.NoHeapPointer = true
					ctx.EmitAndRegImm32(d25.Reg, 1)
					d25.Type = tagBool
					ctx.BindReg(d25.Reg, &d25)
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						ctx.EmitMakeBool(result, d25)
					} else {
						ctx.EmitMovToReg(result.Reg2, d25)
						d26 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d26)
						if d25.Loc == LocReg && d25.Reg != result.Reg2 {
							ctx.FreeReg(d25.Reg)
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
					ctx.ReclaimUntrackedRegs()
					d27 = args[1]
					d27.ID = 0
					d29 = d27
					d29.ID = 0
					d28 = ctx.EmitTagEqualsBorrowed(&d29, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d27)
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
							ps31.OverlayValues[0] = d0
							ps31.OverlayValues[1] = d1
							ps31.OverlayValues[2] = d2
							ps31.OverlayValues[3] = d3
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
							ps31.OverlayValues[30] = d30
							return bbs[1].RenderPS(ps31)
						}
						if ps.General {
						}
						ps32 := PhiState{General: ps.General}
						ps32.OverlayValues = make([]JITValueDesc, 31)
						ps32.OverlayValues[0] = d0
						ps32.OverlayValues[1] = d1
						ps32.OverlayValues[2] = d2
						ps32.OverlayValues[3] = d3
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
						ps32.OverlayValues[30] = d30
						return bbs[2].RenderPS(ps32)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					snap33 := d0
					snap34 := d1
					snap35 := d2
					snap36 := d3
					snap37 := d18
					snap38 := d19
					snap39 := d20
					snap40 := d21
					snap41 := d22
					snap42 := d23
					snap43 := d24
					snap44 := d25
					snap45 := d26
					snap46 := d27
					snap47 := d28
					snap48 := d29
					snap49 := d30
					alloc50 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc50)
					d0 = snap33
					d1 = snap34
					d2 = snap35
					d3 = snap36
					d18 = snap37
					d19 = snap38
					d20 = snap39
					d21 = snap40
					d22 = snap41
					d23 = snap42
					d24 = snap43
					d25 = snap44
					d26 = snap45
					d27 = snap46
					d28 = snap47
					d29 = snap48
					d30 = snap49
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc50)
					d0 = snap33
					d1 = snap34
					d2 = snap35
					d3 = snap36
					d18 = snap37
					d19 = snap38
					d20 = snap39
					d21 = snap40
					d22 = snap41
					d23 = snap42
					d24 = snap43
					d25 = snap44
					d26 = snap45
					d27 = snap46
					d28 = snap47
					d29 = snap48
					d30 = snap49
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 31)
					ps51.OverlayValues[0] = d0
					ps51.OverlayValues[1] = d1
					ps51.OverlayValues[2] = d2
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[18] = d18
					ps51.OverlayValues[19] = d19
					ps51.OverlayValues[20] = d20
					ps51.OverlayValues[21] = d21
					ps51.OverlayValues[22] = d22
					ps51.OverlayValues[23] = d23
					ps51.OverlayValues[24] = d24
					ps51.OverlayValues[25] = d25
					ps51.OverlayValues[26] = d26
					ps51.OverlayValues[27] = d27
					ps51.OverlayValues[28] = d28
					ps51.OverlayValues[29] = d29
					ps51.OverlayValues[30] = d30
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 31)
					ps52.OverlayValues[0] = d0
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[20] = d20
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[26] = d26
					ps52.OverlayValues[27] = d27
					ps52.OverlayValues[28] = d28
					ps52.OverlayValues[29] = d29
					ps52.OverlayValues[30] = d30
					snap53 := d0
					snap54 := d1
					snap55 := d2
					snap56 := d3
					snap57 := d18
					snap58 := d19
					snap59 := d20
					snap60 := d21
					snap61 := d22
					snap62 := d23
					snap63 := d24
					snap64 := d25
					snap65 := d26
					snap66 := d27
					snap67 := d28
					snap68 := d29
					snap69 := d30
					alloc70 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc70)
					d0 = snap53
					d1 = snap54
					d2 = snap55
					d3 = snap56
					d18 = snap57
					d19 = snap58
					d20 = snap59
					d21 = snap60
					d22 = snap61
					d23 = snap62
					d24 = snap63
					d25 = snap64
					d26 = snap65
					d27 = snap66
					d28 = snap67
					d29 = snap68
					d30 = snap69
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps51)
					}
					return result
					ctx.FreeDesc(&d28)
					return result
				}
				ps71 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps71)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["toLower"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["toUpper"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["replace"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
				d3 := args[1]
				d3.ID = 0
				d5 := d3
				ctx.SyncDesc(&d5)
				if d5.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d5.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d5.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d5 = tmpScalar
				}
				d5 = JITPrepareScmerGoArg(ctx, d5)
				if d5.Loc != LocRegPair && d5.Loc != LocStackPair && d5.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d5}, 2)
				ctx.FreeDesc(&d3)
				d6 := args[2]
				d6.ID = 0
				d8 := d6
				ctx.SyncDesc(&d8)
				if d8.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d8.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d8.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d8 = tmpScalar
				}
				d8 = JITPrepareScmerGoArg(ctx, d8)
				if d8.Loc != LocRegPair && d8.Loc != LocStackPair && d8.Loc != LocInputPair {
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
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				if d9.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d9.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d9.Imm)
					ptrWord, _ := d9.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d9.Imm.String())))
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
				if d9.Loc != LocRegPair && d9.Loc != LocStackPair && d9.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.Replace arg0)")
				}
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d10.Imm)
					ptrWord, _ := d10.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d10.Imm.String())))
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
				if d10.Loc != LocRegPair && d10.Loc != LocStackPair && d10.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.Replace arg1)")
				}
				ctx.EnsureDesc(&d11)
				ctx.EnsureDesc(&d11)
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
					panic("jit: generic call arg expects 2-word value (strings.Replace arg2)")
				}
				d12 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
				if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d9)
				ctx.SyncDesc(&d10)
				ctx.SyncDesc(&d11)
				ctx.SyncDesc(&d12)
				d13 := ctx.EmitGoCallScalar(GoFuncAddr(strings.Replace), []JITValueDesc{d9, d10, d11, d12}, 2)
				d13.NoHeapPointer = false
				ctx.BindReg(d13.Reg, &d13)
				ctx.BindReg(d13.Reg2, &d13)
				ctx.FreeDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d13)
				d14 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d13}, 2)
				if result.Loc == LocAny {
					return d14
				}
				ctx.EmitMovPairToResult(&d14, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 14,
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
				declaration := declarations["strtrim"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["strltrim"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
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
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair && d3.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
				}
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d1, d3}, 2)
				d4.NoHeapPointer = false
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
				declaration := declarations["strrtrim"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
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
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair && d3.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
				}
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d1, d3}, 2)
				d4.NoHeapPointer = false
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
				declaration := declarations["sql_trim"]
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
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
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
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps12)
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
					d19 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d20)
					d22 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d20}, 2)
					d22.NoHeapPointer = false
					ctx.BindReg(d22.Reg, &d22)
					ctx.BindReg(d22.Reg2, &d22)
					ctx.EnsureDesc(&d22)
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d22}, 2)
					ctx.EmitMovPairToResult(&d23, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps24 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps24)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["sql_ltrim"]
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
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
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps12)
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
					d19 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d22 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d22.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d22.Imm)
						ptrWord, _ := d22.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d22.Imm.String())))
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
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair && d22.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d22)
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d20, d22}, 2)
					d23.NoHeapPointer = false
					ctx.BindReg(d23.Reg, &d23)
					ctx.BindReg(d23.Reg2, &d23)
					ctx.FreeDesc(&d22)
					ctx.EnsureDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d23}, 2)
					ctx.EmitMovPairToResult(&d24, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps25 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps25)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["sql_rtrim"]
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
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
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps12)
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
					d19 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d22 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d22.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d22.Imm)
						ptrWord, _ := d22.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d22.Imm.String())))
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
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair && d22.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d22)
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d20, d22}, 2)
					d23.NoHeapPointer = false
					ctx.BindReg(d23.Reg, &d23)
					ctx.BindReg(d23.Reg2, &d23)
					ctx.FreeDesc(&d22)
					ctx.EnsureDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d23}, 2)
					ctx.EmitMovPairToResult(&d24, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps25 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps25)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["split"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d10 JITValueDesc
				_ = d10
				var d20 JITValueDesc
				_ = d20
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
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
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
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
				var d52 JITValueDesc
				_ = d52
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
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				var bbs [6]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[3].PhiBase = int32(phiBase0) + int32(16)
				bbs[3].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				d3 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d3
				var d4 JITValueDesc
				if phiHomeOK2 {
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				}
				_ = d4
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
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					ctx.ReclaimUntrackedRegs()
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d5)
					var d6 JITValueDesc
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() > 1)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d5.Reg, 1)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedGreater}
						ctx.BindReg(r1, &d6)
					}
					ctx.FreeDesc(&d5)
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							if ps.General {
							}
							ps8 := PhiState{General: ps.General}
							ps8.OverlayValues = make([]JITValueDesc, 8)
							ps8.OverlayValues[3] = d3
							ps8.OverlayValues[4] = d4
							ps8.OverlayValues[5] = d5
							ps8.OverlayValues[6] = d6
							ps8.OverlayValues[7] = d7
							return bbs[1].RenderPS(ps8)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps9 := PhiState{General: ps.General}
						ps9.OverlayValues = make([]JITValueDesc, 8)
						ps9.OverlayValues[3] = d3
						ps9.OverlayValues[4] = d4
						ps9.OverlayValues[5] = d5
						ps9.OverlayValues[6] = d6
						ps9.OverlayValues[7] = d7
						ps9.PhiValues = make([]JITValueDesc, 1)
						d10 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}
						ps9.PhiValues[0] = d10
						return bbs[2].RenderPS(ps9)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitJump(d7.Condition, lbl7)
					ctx.EmitJmp(lbl8)
					snap11 := d3
					snap12 := d4
					snap13 := d5
					snap14 := d6
					snap15 := d7
					snap16 := d10
					alloc17 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc17)
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					d7 = snap15
					d10 = snap16
					ctx.MarkLabel(lbl8)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc17)
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					d7 = snap15
					d10 = snap16
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 11)
					ps18.OverlayValues[3] = d3
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[6] = d6
					ps18.OverlayValues[7] = d7
					ps18.OverlayValues[10] = d10
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 11)
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[7] = d7
					ps19.OverlayValues[10] = d10
					ps19.PhiValues = make([]JITValueDesc, 1)
					d20 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}
					ps19.PhiValues[0] = d20
					snap21 := d3
					snap22 := d4
					snap23 := d5
					snap24 := d6
					snap25 := d7
					snap26 := d10
					snap27 := d20
					alloc28 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps19)
					}
					ctx.RestoreAllocState(alloc28)
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d10 = snap26
					d20 = snap27
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps18)
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
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
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
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					d29 = args[1]
					d29.ID = 0
					d31 = d29
					ctx.SyncDesc(&d31)
					if d31.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d31.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d31.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d31 = tmpScalar
					}
					d31 = JITPrepareScmerGoArg(ctx, d31)
					if d31.Loc != LocRegPair && d31.Loc != LocStackPair && d31.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d31}, 2)
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.FreeDesc(&d29)
					if ps.General {
						ctx.SyncDesc(&d30)
						if d30.Loc == LocReg {
							ctx.ProtectReg(d30.Reg)
						} else if d30.Loc == LocRegPair {
							ctx.ProtectReg(d30.Reg)
							ctx.ProtectReg(d30.Reg2)
						}
						d32 = d30
						if d32.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d32)
						if d32.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d32, int32(bbs[2].PhiBase)+int32(0), 2)
						} else if d32.Loc == LocInputPair {
							ctx.EnsureDesc(&d32)
							ctx.EmitStoreScmerToStack(d32, int32(bbs[2].PhiBase)+int32(0))
						} else if d32.Loc == LocRegPair || d32.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d32, int32(bbs[2].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d32)
							ctx.EmitStoreToStack(d32, int32(bbs[2].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
						}
						if d30.Loc == LocReg {
							ctx.UnprotectReg(d30.Reg)
						} else if d30.Loc == LocRegPair {
							ctx.UnprotectReg(d30.Reg)
							ctx.UnprotectReg(d30.Reg2)
						}
					}
					ps33 := PhiState{General: ps.General}
					ps33.OverlayValues = make([]JITValueDesc, 33)
					ps33.OverlayValues[3] = d3
					ps33.OverlayValues[4] = d4
					ps33.OverlayValues[5] = d5
					ps33.OverlayValues[6] = d6
					ps33.OverlayValues[7] = d7
					ps33.OverlayValues[10] = d10
					ps33.OverlayValues[20] = d20
					ps33.OverlayValues[29] = d29
					ps33.OverlayValues[30] = d30
					ps33.OverlayValues[31] = d31
					ps33.OverlayValues[32] = d32
					ps33.PhiValues = make([]JITValueDesc, 1)
					d34 = d30
					ps33.PhiValues[0] = d34
					if ps33.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps33)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d35 := ps.PhiValues[0]
							ctx.EnsureDesc(&d35)
							ctx.EmitStoreScmerToStack(d35, int32(bbs[2].PhiBase)+int32(0))
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
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
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
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					d36 = args[0]
					d36.ID = 0
					d38 = d36
					ctx.SyncDesc(&d38)
					if d38.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d38.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d38.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d38 = tmpScalar
					}
					d38 = JITPrepareScmerGoArg(ctx, d38)
					if d38.Loc != LocRegPair && d38.Loc != LocStackPair && d38.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d38}, 2)
					ctx.FreeDesc(&d36)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d37.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d37.Imm)
						ptrWord, _ := d37.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d37.Imm.String())))
						d37 = tmpPair
					} else if d37.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d37.Type, Reg: ctx.AllocRegExcept(d37.Reg), Reg2: ctx.AllocRegExcept(d37.Reg)}
						switch d37.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d37)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d37)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d37)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d37)
						d37 = tmpPair
					}
					if d37.Loc != LocRegPair && d37.Loc != LocStackPair && d37.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Split arg0)")
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
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
					if d3.Loc != LocRegPair && d3.Loc != LocStackPair && d3.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Split arg1)")
					}
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d3)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Split), []JITValueDesc{d37, d3}, 3)
					d39.NoHeapPointer = false
					ctx.BindReg(d39.Reg, &d39)
					ctx.BindReg(d39.Reg2, &d39)
					ctx.BindReg(d39.Reg3, &d39)
					ctx.StabilizeDescForControlFlow(&d39)
					ctx.FreeDesc(&d3)
					var d40 JITValueDesc
					if d39.SliceSizeKnown {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.KnownSliceLen))}
					} else if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.StackOff))}
					} else if d39.Loc == LocStackTriple {
						d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d39.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d39)
						if d39.Loc == LocRegPair || d39.Loc == LocRegTriple {
							d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg2, ID: 0}
						} else if d39.Loc == LocReg {
							d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					callResults41 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d40, d40}, []uint8{3}, []uint8{1})
					d42 = callResults41[0]
					d42.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.FreeDesc(&d40)
					var d43 JITValueDesc
					if d39.SliceSizeKnown {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.KnownSliceLen))}
					} else if d39.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.StackOff))}
					} else if d39.Loc == LocStackTriple {
						d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d39.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d39)
						if d39.Loc == LocRegPair || d39.Loc == LocRegTriple {
							d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg2, ID: 0}
						} else if d39.Loc == LocReg {
							d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d43)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
						}
					}
					ps44 := PhiState{General: ps.General}
					ps44.OverlayValues = make([]JITValueDesc, 44)
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[6] = d6
					ps44.OverlayValues[7] = d7
					ps44.OverlayValues[10] = d10
					ps44.OverlayValues[20] = d20
					ps44.OverlayValues[29] = d29
					ps44.OverlayValues[30] = d30
					ps44.OverlayValues[31] = d31
					ps44.OverlayValues[32] = d32
					ps44.OverlayValues[34] = d34
					ps44.OverlayValues[35] = d35
					ps44.OverlayValues[36] = d36
					ps44.OverlayValues[37] = d37
					ps44.OverlayValues[38] = d38
					ps44.OverlayValues[39] = d39
					ps44.OverlayValues[40] = d40
					ps44.OverlayValues[42] = d42
					ps44.OverlayValues[43] = d43
					ps44.PhiValues = make([]JITValueDesc, 1)
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps44.PhiValues[0] = d45
					if ps44.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps44)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d46 := ps.PhiValues[0]
							ctx.EnsureDesc(&d46)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d46)
							} else {
								ctx.EmitStoreToStack(d46, int32(bbs[3].PhiBase)+int32(0))
							}
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
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
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
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d4 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d4.Loc == LocReg {
						ctx.BindReg(r0, &d4)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					var d47 JITValueDesc
					if d4.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d47)
					}
					if d47.Loc == LocReg && d4.Loc == LocReg && d47.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d47)
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDescsTogether(&d47, &d43)
					var d48 JITValueDesc
					if d47.Loc == LocImm && d43.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d47.Imm.Int() < d43.Imm.Int())}
					} else if d43.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d47.Reg)
						if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d47.Reg, int32(d43.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
							ctx.EmitCmpInt64(d47.Reg, RegR11)
						}
						d48 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d48)
					} else if d47.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d47.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d43.Reg)
						d48 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d48)
					} else {
						r4 := ctx.AllocRegExcept(d47.Reg)
						ctx.EmitCmpInt64(d47.Reg, d43.Reg)
						d48 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d48)
					}
					d49 = d48
					ctx.EnsureDesc(&d49)
					if d49.Loc != LocImm && d49.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d49.Loc == LocImm {
						if d49.Imm.Bool() {
							if ps.General {
							}
							ps50 := PhiState{General: ps.General}
							ps50.OverlayValues = make([]JITValueDesc, 50)
							ps50.OverlayValues[3] = d3
							ps50.OverlayValues[4] = d4
							ps50.OverlayValues[5] = d5
							ps50.OverlayValues[6] = d6
							ps50.OverlayValues[7] = d7
							ps50.OverlayValues[10] = d10
							ps50.OverlayValues[20] = d20
							ps50.OverlayValues[29] = d29
							ps50.OverlayValues[30] = d30
							ps50.OverlayValues[31] = d31
							ps50.OverlayValues[32] = d32
							ps50.OverlayValues[34] = d34
							ps50.OverlayValues[35] = d35
							ps50.OverlayValues[36] = d36
							ps50.OverlayValues[37] = d37
							ps50.OverlayValues[38] = d38
							ps50.OverlayValues[39] = d39
							ps50.OverlayValues[40] = d40
							ps50.OverlayValues[42] = d42
							ps50.OverlayValues[43] = d43
							ps50.OverlayValues[45] = d45
							ps50.OverlayValues[46] = d46
							ps50.OverlayValues[47] = d47
							ps50.OverlayValues[48] = d48
							ps50.OverlayValues[49] = d49
							return bbs[4].RenderPS(ps50)
						}
						if ps.General {
						}
						ps51 := PhiState{General: ps.General}
						ps51.OverlayValues = make([]JITValueDesc, 50)
						ps51.OverlayValues[3] = d3
						ps51.OverlayValues[4] = d4
						ps51.OverlayValues[5] = d5
						ps51.OverlayValues[6] = d6
						ps51.OverlayValues[7] = d7
						ps51.OverlayValues[10] = d10
						ps51.OverlayValues[20] = d20
						ps51.OverlayValues[29] = d29
						ps51.OverlayValues[30] = d30
						ps51.OverlayValues[31] = d31
						ps51.OverlayValues[32] = d32
						ps51.OverlayValues[34] = d34
						ps51.OverlayValues[35] = d35
						ps51.OverlayValues[36] = d36
						ps51.OverlayValues[37] = d37
						ps51.OverlayValues[38] = d38
						ps51.OverlayValues[39] = d39
						ps51.OverlayValues[40] = d40
						ps51.OverlayValues[42] = d42
						ps51.OverlayValues[43] = d43
						ps51.OverlayValues[45] = d45
						ps51.OverlayValues[46] = d46
						ps51.OverlayValues[47] = d47
						ps51.OverlayValues[48] = d48
						ps51.OverlayValues[49] = d49
						return bbs[5].RenderPS(ps51)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d52 := ps.PhiValues[0]
							ctx.EnsureDesc(&d52)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d52)
							} else {
								ctx.EmitStoreToStack(d52, int32(bbs[3].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d49.Condition, lbl9)
					ctx.EmitJmp(lbl10)
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d10
					snap59 := d20
					snap60 := d29
					snap61 := d30
					snap62 := d31
					snap63 := d32
					snap64 := d34
					snap65 := d35
					snap66 := d36
					snap67 := d37
					snap68 := d38
					snap69 := d39
					snap70 := d40
					snap71 := d42
					snap72 := d43
					snap73 := d45
					snap74 := d46
					snap75 := d47
					snap76 := d48
					snap77 := d49
					snap78 := d52
					alloc79 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc79)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d10 = snap58
					d20 = snap59
					d29 = snap60
					d30 = snap61
					d31 = snap62
					d32 = snap63
					d34 = snap64
					d35 = snap65
					d36 = snap66
					d37 = snap67
					d38 = snap68
					d39 = snap69
					d40 = snap70
					d42 = snap71
					d43 = snap72
					d45 = snap73
					d46 = snap74
					d47 = snap75
					d48 = snap76
					d49 = snap77
					d52 = snap78
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc79)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d10 = snap58
					d20 = snap59
					d29 = snap60
					d30 = snap61
					d31 = snap62
					d32 = snap63
					d34 = snap64
					d35 = snap65
					d36 = snap66
					d37 = snap67
					d38 = snap68
					d39 = snap69
					d40 = snap70
					d42 = snap71
					d43 = snap72
					d45 = snap73
					d46 = snap74
					d47 = snap75
					d48 = snap76
					d49 = snap77
					d52 = snap78
					ps80 := PhiState{General: true}
					ps80.OverlayValues = make([]JITValueDesc, 53)
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[4] = d4
					ps80.OverlayValues[5] = d5
					ps80.OverlayValues[6] = d6
					ps80.OverlayValues[7] = d7
					ps80.OverlayValues[10] = d10
					ps80.OverlayValues[20] = d20
					ps80.OverlayValues[29] = d29
					ps80.OverlayValues[30] = d30
					ps80.OverlayValues[31] = d31
					ps80.OverlayValues[32] = d32
					ps80.OverlayValues[34] = d34
					ps80.OverlayValues[35] = d35
					ps80.OverlayValues[36] = d36
					ps80.OverlayValues[37] = d37
					ps80.OverlayValues[38] = d38
					ps80.OverlayValues[39] = d39
					ps80.OverlayValues[40] = d40
					ps80.OverlayValues[42] = d42
					ps80.OverlayValues[43] = d43
					ps80.OverlayValues[45] = d45
					ps80.OverlayValues[46] = d46
					ps80.OverlayValues[47] = d47
					ps80.OverlayValues[48] = d48
					ps80.OverlayValues[49] = d49
					ps80.OverlayValues[52] = d52
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 53)
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[4] = d4
					ps81.OverlayValues[5] = d5
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[7] = d7
					ps81.OverlayValues[10] = d10
					ps81.OverlayValues[20] = d20
					ps81.OverlayValues[29] = d29
					ps81.OverlayValues[30] = d30
					ps81.OverlayValues[31] = d31
					ps81.OverlayValues[32] = d32
					ps81.OverlayValues[34] = d34
					ps81.OverlayValues[35] = d35
					ps81.OverlayValues[36] = d36
					ps81.OverlayValues[37] = d37
					ps81.OverlayValues[38] = d38
					ps81.OverlayValues[39] = d39
					ps81.OverlayValues[40] = d40
					ps81.OverlayValues[42] = d42
					ps81.OverlayValues[43] = d43
					ps81.OverlayValues[45] = d45
					ps81.OverlayValues[46] = d46
					ps81.OverlayValues[47] = d47
					ps81.OverlayValues[48] = d48
					ps81.OverlayValues[49] = d49
					ps81.OverlayValues[52] = d52
					snap82 := d3
					snap83 := d4
					snap84 := d5
					snap85 := d6
					snap86 := d7
					snap87 := d10
					snap88 := d20
					snap89 := d29
					snap90 := d30
					snap91 := d31
					snap92 := d32
					snap93 := d34
					snap94 := d35
					snap95 := d36
					snap96 := d37
					snap97 := d38
					snap98 := d39
					snap99 := d40
					snap100 := d42
					snap101 := d43
					snap102 := d45
					snap103 := d46
					snap104 := d47
					snap105 := d48
					snap106 := d49
					snap107 := d52
					alloc108 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps81)
					}
					ctx.RestoreAllocState(alloc108)
					d3 = snap82
					d4 = snap83
					d5 = snap84
					d6 = snap85
					d7 = snap86
					d10 = snap87
					d20 = snap88
					d29 = snap89
					d30 = snap90
					d31 = snap91
					d32 = snap92
					d34 = snap93
					d35 = snap94
					d36 = snap95
					d37 = snap96
					d38 = snap97
					d39 = snap98
					d40 = snap99
					d42 = snap100
					d43 = snap101
					d45 = snap102
					d46 = snap103
					d47 = snap104
					d48 = snap105
					d49 = snap106
					d52 = snap107
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps80)
					}
					return result
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
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
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
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.EnsureDesc(&d47)
					d110 = ctx.EmitSliceElementAddress(&d39, &d47, 16)
					ctx.EnsureDesc(&d110)
					r5 := ctx.AllocRegExcept(d110.Reg)
					ctx.EmitMovRegMem(r5, d110.Reg, 8)
					ctx.EmitMovRegMem(d110.Reg, d110.Reg, 0)
					d109 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d110.Reg, Reg2: r5}
					ctx.BindReg(d110.Reg, &d109)
					ctx.BindReg(r5, &d109)
					ctx.EnsureDesc(&d109)
					ctx.EnsureDesc(&d47)
					ctx.SyncDesc(&d109)
					ctx.StabilizeDescAcrossNestedCall(&d47)
					d111 = d42
					d111.ID = 0
					d112 = d47
					d112.ID = 0
					d113 = ctx.EmitSliceElementAddress(&d111, &d112, int32(16))
					ctx.FreeDesc(&d112)
					ctx.EmitStoreScmerAt(&d113, &d109)
					ctx.FreeDesc(&d113)
					if ps.General {
						ctx.SyncDesc(&d47)
						if d47.Loc == LocReg {
							ctx.ProtectReg(d47.Reg)
						} else if d47.Loc == LocRegPair {
							ctx.ProtectReg(d47.Reg)
							ctx.ProtectReg(d47.Reg2)
						}
						d114 = d47
						if d114.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d114)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d114)
						} else {
							ctx.EmitStoreToStack(d114, int32(bbs[3].PhiBase)+int32(0))
						}
						if d47.Loc == LocReg {
							ctx.UnprotectReg(d47.Reg)
						} else if d47.Loc == LocRegPair {
							ctx.UnprotectReg(d47.Reg)
							ctx.UnprotectReg(d47.Reg2)
						}
					}
					ps115 := PhiState{General: ps.General}
					ps115.OverlayValues = make([]JITValueDesc, 115)
					ps115.OverlayValues[3] = d3
					ps115.OverlayValues[4] = d4
					ps115.OverlayValues[5] = d5
					ps115.OverlayValues[6] = d6
					ps115.OverlayValues[7] = d7
					ps115.OverlayValues[10] = d10
					ps115.OverlayValues[20] = d20
					ps115.OverlayValues[29] = d29
					ps115.OverlayValues[30] = d30
					ps115.OverlayValues[31] = d31
					ps115.OverlayValues[32] = d32
					ps115.OverlayValues[34] = d34
					ps115.OverlayValues[35] = d35
					ps115.OverlayValues[36] = d36
					ps115.OverlayValues[37] = d37
					ps115.OverlayValues[38] = d38
					ps115.OverlayValues[39] = d39
					ps115.OverlayValues[40] = d40
					ps115.OverlayValues[42] = d42
					ps115.OverlayValues[43] = d43
					ps115.OverlayValues[45] = d45
					ps115.OverlayValues[46] = d46
					ps115.OverlayValues[47] = d47
					ps115.OverlayValues[48] = d48
					ps115.OverlayValues[49] = d49
					ps115.OverlayValues[52] = d52
					ps115.OverlayValues[109] = d109
					ps115.OverlayValues[110] = d110
					ps115.OverlayValues[111] = d111
					ps115.OverlayValues[112] = d112
					ps115.OverlayValues[113] = d113
					ps115.OverlayValues[114] = d114
					ps115.PhiValues = make([]JITValueDesc, 1)
					d116 = d47
					ps115.PhiValues[0] = d116
					if ps115.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps115)
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
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
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
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
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
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d42)
					d117 = ctx.EmitNewSliceFromGoSlice(&d42)
					ctx.SyncDesc(&d117)
					if d117.Loc == LocRegPair || d117.Loc == LocStackPair || d117.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d117, &result)
						result.Type = d117.Type
					} else {
						switch d117.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d117)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d117)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d117)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d117, &result)
							result.Type = d117.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps118 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps118)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["string_repeat"]
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl3)
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
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps12)
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
					d20 = d19
					_ = d20
					ctx.StabilizeDescForControlFlow(&d20)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d21 JITValueDesc
					if d20.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int())}
					} else if d20.Type == tagInt && d20.Loc == LocRegPair {
						ctx.FreeReg(d20.Reg)
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg2}
						ctx.BindReg(d20.Reg2, &d21)
						ctx.BindReg(d20.Reg2, &d21)
					} else if d20.Type == tagInt && d20.Loc == LocReg {
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg}
						ctx.BindReg(d20.Reg, &d21)
						ctx.BindReg(d20.Reg, &d21)
					} else {
						d21 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d20}, 1)
						d21.Type = tagInt
						ctx.BindReg(d21.Reg, &d21)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d19)
					ctx.EnsureDesc(&d21)
					var d23 JITValueDesc
					if d21.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() <= 0)}
					} else {
						r0 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpRegImm32(d21.Reg, 0)
						d23 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r0, &d23)
					}
					d24 = d23
					ctx.EnsureDesc(&d24)
					if d24.Loc != LocImm && d24.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
							ps25.OverlayValues[18] = d18
							ps25.OverlayValues[19] = d19
							ps25.OverlayValues[20] = d20
							ps25.OverlayValues[21] = d21
							ps25.OverlayValues[22] = d22
							ps25.OverlayValues[23] = d23
							ps25.OverlayValues[24] = d24
							return bbs[3].RenderPS(ps25)
						}
						if ps.General {
						}
						ps26 := PhiState{General: ps.General}
						ps26.OverlayValues = make([]JITValueDesc, 25)
						ps26.OverlayValues[0] = d0
						ps26.OverlayValues[1] = d1
						ps26.OverlayValues[2] = d2
						ps26.OverlayValues[3] = d3
						ps26.OverlayValues[18] = d18
						ps26.OverlayValues[19] = d19
						ps26.OverlayValues[20] = d20
						ps26.OverlayValues[21] = d21
						ps26.OverlayValues[22] = d22
						ps26.OverlayValues[23] = d23
						ps26.OverlayValues[24] = d24
						return bbs[4].RenderPS(ps26)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d24.Condition, lbl9)
					ctx.EmitJmp(lbl10)
					snap27 := d0
					snap28 := d1
					snap29 := d2
					snap30 := d3
					snap31 := d18
					snap32 := d19
					snap33 := d20
					snap34 := d21
					snap35 := d22
					snap36 := d23
					snap37 := d24
					alloc38 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc38)
					d0 = snap27
					d1 = snap28
					d2 = snap29
					d3 = snap30
					d18 = snap31
					d19 = snap32
					d20 = snap33
					d21 = snap34
					d22 = snap35
					d23 = snap36
					d24 = snap37
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc38)
					d0 = snap27
					d1 = snap28
					d2 = snap29
					d3 = snap30
					d18 = snap31
					d19 = snap32
					d20 = snap33
					d21 = snap34
					d22 = snap35
					d23 = snap36
					d24 = snap37
					ps39 := PhiState{General: true}
					ps39.OverlayValues = make([]JITValueDesc, 25)
					ps39.OverlayValues[0] = d0
					ps39.OverlayValues[1] = d1
					ps39.OverlayValues[2] = d2
					ps39.OverlayValues[3] = d3
					ps39.OverlayValues[18] = d18
					ps39.OverlayValues[19] = d19
					ps39.OverlayValues[20] = d20
					ps39.OverlayValues[21] = d21
					ps39.OverlayValues[22] = d22
					ps39.OverlayValues[23] = d23
					ps39.OverlayValues[24] = d24
					ps40 := PhiState{General: true}
					ps40.OverlayValues = make([]JITValueDesc, 25)
					ps40.OverlayValues[0] = d0
					ps40.OverlayValues[1] = d1
					ps40.OverlayValues[2] = d2
					ps40.OverlayValues[3] = d3
					ps40.OverlayValues[18] = d18
					ps40.OverlayValues[19] = d19
					ps40.OverlayValues[20] = d20
					ps40.OverlayValues[21] = d21
					ps40.OverlayValues[22] = d22
					ps40.OverlayValues[23] = d23
					ps40.OverlayValues[24] = d24
					snap41 := d0
					snap42 := d1
					snap43 := d2
					snap44 := d3
					snap45 := d18
					snap46 := d19
					snap47 := d20
					snap48 := d21
					snap49 := d22
					snap50 := d23
					snap51 := d24
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps40)
					}
					ctx.RestoreAllocState(alloc52)
					d0 = snap41
					d1 = snap42
					d2 = snap43
					d3 = snap44
					d18 = snap45
					d19 = snap46
					d20 = snap47
					d21 = snap48
					d22 = snap49
					d23 = snap50
					d24 = snap51
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps39)
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
					d53 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d54 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d53}, 2)
					ctx.EmitMovPairToResult(&d54, &result)
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					ctx.ReclaimUntrackedRegs()
					d55 = args[0]
					d55.ID = 0
					d57 = d55
					ctx.SyncDesc(&d57)
					if d57.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d57.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d57.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d57 = tmpScalar
					}
					d57 = JITPrepareScmerGoArg(ctx, d57)
					if d57.Loc != LocRegPair && d57.Loc != LocStackPair && d57.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d56 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d57}, 2)
					ctx.FreeDesc(&d55)
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d56.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d56.Imm)
						ptrWord, _ := d56.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d56.Imm.String())))
						d56 = tmpPair
					} else if d56.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d56.Type, Reg: ctx.AllocRegExcept(d56.Reg), Reg2: ctx.AllocRegExcept(d56.Reg)}
						switch d56.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d56)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d56)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d56)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d56)
						d56 = tmpPair
					}
					if d56.Loc != LocRegPair && d56.Loc != LocStackPair && d56.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Repeat arg0)")
					}
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d56)
					ctx.SyncDesc(&d21)
					d58 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Repeat), []JITValueDesc{d56, d21}, 2)
					d58.NoHeapPointer = false
					ctx.BindReg(d58.Reg, &d58)
					ctx.BindReg(d58.Reg2, &d58)
					ctx.EnsureDesc(&d58)
					d59 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d58}, 2)
					ctx.EmitMovPairToResult(&d59, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps60 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps60)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
						classify := func(s string) (isASCII bool, key byte) {
							if s == "" {
								return true, 0
							}
							// map leading "aa" to non-ASCII class
							if len(s) >= 2 && asciiFoldByte(s[0]) == 'a' && asciiFoldByte(s[1]) == 'a' {
								return false, 0
							}
							b := asciiFoldByte(s[0])
							// check ASCII letter
							if b >= 'a' && b <= 'z' && (s[0] < 128) {
								return true, b
							}
							return false, 0
						}
						if reverse {
							f := func(a ...Scmer) Scmer {
								as := String(a[0])
								bs := String(a[1])
								aAsc, ak := classify(as)
								bAsc, bk := classify(bs)
								var res bool
								if aAsc != bAsc {
									// ASCII ranks above non-ASCII for DESC too
									res = aAsc && !bAsc
								} else if aAsc { // both ASCII letters: reverse letter order
									if ak != bk {
										res = ak > bk
									} else {
										res = generalCIFoldCompare(as, bs) > 0
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
							aAsc, ak := classify(as)
							bAsc, bk := classify(bs)
							var res bool
							if aAsc != bAsc {
								// ASCII first for ASC
								res = aAsc && !bAsc
							} else if aAsc { // both ASCII letters
								if ak != bk {
									res = ak < bk
								} else {
									res = generalCIFoldCompare(as, bs) < 0
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["collate"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				declaration := declarations["htmlentities"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (html.EscapeString arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(html.EscapeString), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["urlencode"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (url.QueryEscape arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(url.QueryEscape), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["urldecode"]
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d28 JITValueDesc
				_ = d28
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					ctx.SyncDesc(&d2)
					if d2.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d2 = tmpScalar
					}
					d2 = JITPrepareScmerGoArg(ctx, d2)
					if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
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
					snap10 := d0
					snap11 := d1
					snap12 := d2
					snap13 := d4
					snap14 := d5
					snap15 := d6
					snap16 := d7
					alloc17 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc17)
					d0 = snap10
					d1 = snap11
					d2 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc17)
					d0 = snap10
					d1 = snap11
					d2 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 8)
					ps18.OverlayValues[0] = d0
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[6] = d6
					ps18.OverlayValues[7] = d7
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 8)
					ps19.OverlayValues[0] = d0
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[7] = d7
					snap20 := d0
					snap21 := d1
					snap22 := d2
					snap23 := d4
					snap24 := d5
					snap25 := d6
					snap26 := d7
					alloc27 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps19)
					}
					ctx.RestoreAllocState(alloc27)
					d0 = snap20
					d1 = snap21
					d2 = snap22
					d4 = snap23
					d5 = snap24
					d6 = snap25
					d7 = snap26
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps18)
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
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
					ctx.EmitMovPairToResult(&d28, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps29 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps29)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["json_encode"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				var d25 JITValueDesc
				_ = d25
				var d27 JITValueDesc
				_ = d27
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
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
					snap9 := d0
					snap10 := d1
					snap11 := d3
					snap12 := d4
					snap13 := d5
					snap14 := d6
					alloc15 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc15)
					d0 = snap9
					d1 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc15)
					d0 = snap9
					d1 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					ps16 := PhiState{General: true}
					ps16.OverlayValues = make([]JITValueDesc, 7)
					ps16.OverlayValues[0] = d0
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					ps16.OverlayValues[6] = d6
					ps17 := PhiState{General: true}
					ps17.OverlayValues = make([]JITValueDesc, 7)
					ps17.OverlayValues[0] = d0
					ps17.OverlayValues[1] = d1
					ps17.OverlayValues[3] = d3
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					snap18 := d0
					snap19 := d1
					snap20 := d3
					snap21 := d4
					snap22 := d5
					snap23 := d6
					alloc24 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps17)
					}
					ctx.RestoreAllocState(alloc24)
					d0 = snap18
					d1 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					d6 = snap23
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps16)
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
					callResults26 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d3}, []uint8{2}, []uint8{1})
					d25 = callResults26[0]
					ctx.EnsureDesc(&d25)
					d27 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d25}, 2)
					ctx.EmitMovPairToResult(&d27, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps28 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps28)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["json_quote"]
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
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
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
				var d125 JITValueDesc
				_ = d125
				var inlineResultOff124 int32
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var phiBase128 int32
				_ = phiBase128
				var d129 JITValueDesc
				_ = d129
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
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
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(func() *bytes.Buffer { return new(bytes.Buffer) }), nil, 1)
					ctx.BindReg(d19.Reg, &d19)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.EnsureDesc(&d19)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *bytes.Buffer) io.Writer { return value }), []JITValueDesc{d19}, 2)
					ctx.EnsureDesc(&d20)
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
					if d20.Loc != LocRegPair && d20.Loc != LocStackPair && d20.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (json.NewEncoder arg0)")
					}
					ctx.SyncDesc(&d20)
					d21 = ctx.EmitGoCallScalar(GoFuncAddr(json.NewEncoder), []JITValueDesc{d20}, 1)
					d21.NoHeapPointer = false
					ctx.BindReg(d21.Reg, &d21)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocRegTriple || d22.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d21)
					ctx.SyncDesc(&d22)
					ctx.EmitGoCallVoid(GoFuncAddr((*json.Encoder).SetEscapeHTML), []JITValueDesc{d21, d22})
					ctx.FreeDesc(&d22)
					d23 = args[0]
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
					d26 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d24}, 2)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d26.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d26.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d26)
						} else if d26.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d26)
						} else if d26.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d26)
						} else if d26.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d26.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d26 = tmpPair
					} else if d26.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d26.Type, Reg: ctx.AllocRegExcept(d26.Reg), Reg2: ctx.AllocRegExcept(d26.Reg)}
						switch d26.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d26)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d26)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d26)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d26)
						d26 = tmpPair
					}
					if d26.Loc != LocRegPair && d26.Loc != LocStackPair && d26.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*json.Encoder).Encode arg1)")
					}
					ctx.SyncDesc(&d21)
					ctx.SyncDesc(&d26)
					d27 = ctx.EmitGoCallScalar(GoFuncAddr((*json.Encoder).Encode), []JITValueDesc{d21, d26}, 2)
					d27.NoHeapPointer = false
					ctx.BindReg(d27.Reg, &d27)
					ctx.BindReg(d27.Reg2, &d27)
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.FreeDesc(&d21)
					ctx.EnsureDesc(&d27)
					var d28 JITValueDesc
					if d27.Loc == LocImm {
						d28 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d27)
						if d27.Loc != LocReg && d27.Loc != LocRegPair && d27.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitCmpRegImm32(d27.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
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
							ps30.OverlayValues[0] = d0
							ps30.OverlayValues[1] = d1
							ps30.OverlayValues[2] = d2
							ps30.OverlayValues[3] = d3
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
							return bbs[4].RenderPS(ps30)
						}
						if ps.General {
						}
						ps31 := PhiState{General: ps.General}
						ps31.OverlayValues = make([]JITValueDesc, 30)
						ps31.OverlayValues[0] = d0
						ps31.OverlayValues[1] = d1
						ps31.OverlayValues[2] = d2
						ps31.OverlayValues[3] = d3
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
						return bbs[5].RenderPS(ps31)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d29.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					snap32 := d0
					snap33 := d1
					snap34 := d2
					snap35 := d3
					snap36 := d18
					snap37 := d19
					snap38 := d20
					snap39 := d21
					snap40 := d22
					snap41 := d23
					snap42 := d24
					snap43 := d25
					snap44 := d26
					snap45 := d27
					snap46 := d28
					snap47 := d29
					alloc48 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc48)
					d0 = snap32
					d1 = snap33
					d2 = snap34
					d3 = snap35
					d18 = snap36
					d19 = snap37
					d20 = snap38
					d21 = snap39
					d22 = snap40
					d23 = snap41
					d24 = snap42
					d25 = snap43
					d26 = snap44
					d27 = snap45
					d28 = snap46
					d29 = snap47
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc48)
					d0 = snap32
					d1 = snap33
					d2 = snap34
					d3 = snap35
					d18 = snap36
					d19 = snap37
					d20 = snap38
					d21 = snap39
					d22 = snap40
					d23 = snap41
					d24 = snap42
					d25 = snap43
					d26 = snap44
					d27 = snap45
					d28 = snap46
					d29 = snap47
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 30)
					ps49.OverlayValues[0] = d0
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[18] = d18
					ps49.OverlayValues[19] = d19
					ps49.OverlayValues[20] = d20
					ps49.OverlayValues[21] = d21
					ps49.OverlayValues[22] = d22
					ps49.OverlayValues[23] = d23
					ps49.OverlayValues[24] = d24
					ps49.OverlayValues[25] = d25
					ps49.OverlayValues[26] = d26
					ps49.OverlayValues[27] = d27
					ps49.OverlayValues[28] = d28
					ps49.OverlayValues[29] = d29
					ps50 := PhiState{General: true}
					ps50.OverlayValues = make([]JITValueDesc, 30)
					ps50.OverlayValues[0] = d0
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[18] = d18
					ps50.OverlayValues[19] = d19
					ps50.OverlayValues[20] = d20
					ps50.OverlayValues[21] = d21
					ps50.OverlayValues[22] = d22
					ps50.OverlayValues[23] = d23
					ps50.OverlayValues[24] = d24
					ps50.OverlayValues[25] = d25
					ps50.OverlayValues[26] = d26
					ps50.OverlayValues[27] = d27
					ps50.OverlayValues[28] = d28
					ps50.OverlayValues[29] = d29
					snap51 := d0
					snap52 := d1
					snap53 := d2
					snap54 := d3
					snap55 := d18
					snap56 := d19
					snap57 := d20
					snap58 := d21
					snap59 := d22
					snap60 := d23
					snap61 := d24
					snap62 := d25
					snap63 := d26
					snap64 := d27
					snap65 := d28
					snap66 := d29
					alloc67 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps50)
					}
					ctx.RestoreAllocState(alloc67)
					d0 = snap51
					d1 = snap52
					d2 = snap53
					d3 = snap54
					d18 = snap55
					d19 = snap56
					d20 = snap57
					d21 = snap58
					d22 = snap59
					d23 = snap60
					d24 = snap61
					d25 = snap62
					d26 = snap63
					d27 = snap64
					d28 = snap65
					d29 = snap66
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps49)
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
					d68 = args[0]
					d68.ID = 0
					d70 = d68
					d70.ID = 0
					d69 = ctx.EmitIsStringBorrowed(&d70, JITValueDesc{Loc: LocAny})
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
							ps72.OverlayValues[0] = d0
							ps72.OverlayValues[1] = d1
							ps72.OverlayValues[2] = d2
							ps72.OverlayValues[3] = d3
							ps72.OverlayValues[18] = d18
							ps72.OverlayValues[19] = d19
							ps72.OverlayValues[20] = d20
							ps72.OverlayValues[21] = d21
							ps72.OverlayValues[22] = d22
							ps72.OverlayValues[23] = d23
							ps72.OverlayValues[24] = d24
							ps72.OverlayValues[25] = d25
							ps72.OverlayValues[26] = d26
							ps72.OverlayValues[27] = d27
							ps72.OverlayValues[28] = d28
							ps72.OverlayValues[29] = d29
							ps72.OverlayValues[68] = d68
							ps72.OverlayValues[69] = d69
							ps72.OverlayValues[70] = d70
							ps72.OverlayValues[71] = d71
							return bbs[2].RenderPS(ps72)
						}
						if ps.General {
						}
						ps73 := PhiState{General: ps.General}
						ps73.OverlayValues = make([]JITValueDesc, 72)
						ps73.OverlayValues[0] = d0
						ps73.OverlayValues[1] = d1
						ps73.OverlayValues[2] = d2
						ps73.OverlayValues[3] = d3
						ps73.OverlayValues[18] = d18
						ps73.OverlayValues[19] = d19
						ps73.OverlayValues[20] = d20
						ps73.OverlayValues[21] = d21
						ps73.OverlayValues[22] = d22
						ps73.OverlayValues[23] = d23
						ps73.OverlayValues[24] = d24
						ps73.OverlayValues[25] = d25
						ps73.OverlayValues[26] = d26
						ps73.OverlayValues[27] = d27
						ps73.OverlayValues[28] = d28
						ps73.OverlayValues[29] = d29
						ps73.OverlayValues[68] = d68
						ps73.OverlayValues[69] = d69
						ps73.OverlayValues[70] = d70
						ps73.OverlayValues[71] = d71
						return bbs[1].RenderPS(ps73)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d71.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					snap74 := d0
					snap75 := d1
					snap76 := d2
					snap77 := d3
					snap78 := d18
					snap79 := d19
					snap80 := d20
					snap81 := d21
					snap82 := d22
					snap83 := d23
					snap84 := d24
					snap85 := d25
					snap86 := d26
					snap87 := d27
					snap88 := d28
					snap89 := d29
					snap90 := d68
					snap91 := d69
					snap92 := d70
					snap93 := d71
					alloc94 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc94)
					d0 = snap74
					d1 = snap75
					d2 = snap76
					d3 = snap77
					d18 = snap78
					d19 = snap79
					d20 = snap80
					d21 = snap81
					d22 = snap82
					d23 = snap83
					d24 = snap84
					d25 = snap85
					d26 = snap86
					d27 = snap87
					d28 = snap88
					d29 = snap89
					d68 = snap90
					d69 = snap91
					d70 = snap92
					d71 = snap93
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc94)
					d0 = snap74
					d1 = snap75
					d2 = snap76
					d3 = snap77
					d18 = snap78
					d19 = snap79
					d20 = snap80
					d21 = snap81
					d22 = snap82
					d23 = snap83
					d24 = snap84
					d25 = snap85
					d26 = snap86
					d27 = snap87
					d28 = snap88
					d29 = snap89
					d68 = snap90
					d69 = snap91
					d70 = snap92
					d71 = snap93
					ps95 := PhiState{General: true}
					ps95.OverlayValues = make([]JITValueDesc, 72)
					ps95.OverlayValues[0] = d0
					ps95.OverlayValues[1] = d1
					ps95.OverlayValues[2] = d2
					ps95.OverlayValues[3] = d3
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
					ps95.OverlayValues[68] = d68
					ps95.OverlayValues[69] = d69
					ps95.OverlayValues[70] = d70
					ps95.OverlayValues[71] = d71
					ps96 := PhiState{General: true}
					ps96.OverlayValues = make([]JITValueDesc, 72)
					ps96.OverlayValues[0] = d0
					ps96.OverlayValues[1] = d1
					ps96.OverlayValues[2] = d2
					ps96.OverlayValues[3] = d3
					ps96.OverlayValues[18] = d18
					ps96.OverlayValues[19] = d19
					ps96.OverlayValues[20] = d20
					ps96.OverlayValues[21] = d21
					ps96.OverlayValues[22] = d22
					ps96.OverlayValues[23] = d23
					ps96.OverlayValues[24] = d24
					ps96.OverlayValues[25] = d25
					ps96.OverlayValues[26] = d26
					ps96.OverlayValues[27] = d27
					ps96.OverlayValues[28] = d28
					ps96.OverlayValues[29] = d29
					ps96.OverlayValues[68] = d68
					ps96.OverlayValues[69] = d69
					ps96.OverlayValues[70] = d70
					ps96.OverlayValues[71] = d71
					snap97 := d0
					snap98 := d1
					snap99 := d2
					snap100 := d3
					snap101 := d18
					snap102 := d19
					snap103 := d20
					snap104 := d21
					snap105 := d22
					snap106 := d23
					snap107 := d24
					snap108 := d25
					snap109 := d26
					snap110 := d27
					snap111 := d28
					snap112 := d29
					snap113 := d68
					snap114 := d69
					snap115 := d70
					snap116 := d71
					alloc117 := ctx.SnapshotAllocState()
					if !bbs[1].Rendered {
						bbs[1].RenderPS(ps96)
					}
					ctx.RestoreAllocState(alloc117)
					d0 = snap97
					d1 = snap98
					d2 = snap99
					d3 = snap100
					d18 = snap101
					d19 = snap102
					d20 = snap103
					d21 = snap104
					d22 = snap105
					d23 = snap106
					d24 = snap107
					d25 = snap108
					d26 = snap109
					d27 = snap110
					d28 = snap111
					d29 = snap112
					d68 = snap113
					d69 = snap114
					d70 = snap115
					d71 = snap116
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps95)
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
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocRegPair || d19.Loc == LocStackPair || d19.Loc == LocRegTriple || d19.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d19)
					d118 = ctx.EmitGoCallScalar(GoFuncAddr((*bytes.Buffer).String), []JITValueDesc{d19}, 2)
					d118.NoHeapPointer = false
					ctx.BindReg(d118.Reg, &d118)
					ctx.BindReg(d118.Reg2, &d118)
					ctx.EnsureDesc(&d118)
					d119 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("\n")}
					d120 = d118
					_ = d120
					ctx.StabilizeDescForControlFlow(&d120)
					d121 = d119
					_ = d121
					ctx.StabilizeDescForControlFlow(&d121)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d120)
					ctx.EnsureDesc(&d121)
					d122 = d120
					_ = d122
					ctx.StabilizeDescForControlFlow(&d122)
					d123 = d121
					_ = d123
					ctx.StabilizeDescForControlFlow(&d123)
					ctx.StabilizeDescForControlFlow(&d120)
					inlineResultOff124 = ctx.AllocStack(int32(16))
					d125 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inlineResultOff124}
					lbl14 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d122)
					ctx.EnsureDesc(&d123)
					d126 = d122
					_ = d126
					ctx.StabilizeDescForControlFlow(&d126)
					d127 = d123
					_ = d127
					ctx.StabilizeDescForControlFlow(&d127)
					ctx.StabilizeDescForControlFlow(&d122)
					ctx.StabilizeDescForControlFlow(&d123)
					phiBase128 = ctx.AllocStack(int32(16))
					d129 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase128) + int32(0)}
					_ = d129
					lbl18 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					lbl21 := ctx.ReserveLabel()
					_ = lbl21
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d129 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase128) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d130 JITValueDesc
					if d126.SliceSizeKnown {
						d130 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d126.KnownSliceLen))}
					} else if d126.Loc == LocImm {
						d130 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d126.Imm.String())))}
					} else if d126.Loc == LocStackTriple {
						d130 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else if d126.Loc == LocStackPair {
						d130 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d126)
						if d126.Loc == LocRegPair || d126.Loc == LocRegTriple {
							d130 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2, ID: 0}
						} else if d126.Loc == LocReg {
							d130 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d131 JITValueDesc
					if d127.SliceSizeKnown {
						d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d127.KnownSliceLen))}
					} else if d127.Loc == LocImm {
						d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d127.StackOff))}
					} else if d127.Loc == LocStackTriple {
						d131 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d127.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d127)
						if d127.Loc == LocRegPair || d127.Loc == LocRegTriple {
							d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d127.Reg2, ID: 0}
						} else if d127.Loc == LocReg {
							d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d127.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d130)
					ctx.EnsureDesc(&d131)
					ctx.EnsureDescsTogether(&d130, &d131)
					var d132 JITValueDesc
					if d130.Loc == LocImm && d131.Loc == LocImm {
						d132 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d130.Imm.Int() >= d131.Imm.Int())}
					} else if d131.Loc == LocImm {
						r1 := ctx.AllocReg()
						if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d130.Reg, int32(d131.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d131.Imm.Int()))
							ctx.EmitCmpInt64(d130.Reg, RegR11)
						}
						d132 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r1, &d132)
					} else if d130.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d130.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d131.Reg)
						d132 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r2, &d132)
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpInt64(d130.Reg, d131.Reg)
						d132 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r3, &d132)
					}
					ctx.FreeDesc(&d130)
					ctx.FreeDesc(&d131)
					ctx.ReclaimUntrackedRegs()
					d133 = d132
					ctx.EnsureDesc(&d133)
					if d133.Loc != LocImm && d133.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					if d133.Loc == LocImm {
						if d133.Imm.Bool() {
							ctx.MarkLabel(lbl22)
							ctx.EmitJmp(lbl20)
						} else {
							ctx.MarkLabel(lbl23)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase128)+int32(0))
							ctx.EmitJmp(lbl21)
						}
					} else {
						ctx.EmitJump(d133.Condition, lbl22)
						ctx.EmitJmp(lbl23)
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl23)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase128)+int32(0))
						ctx.EmitJmp(lbl21)
					}
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl21)
					ctx.ResolveFixups()
					d129 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase128) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r4 := ctx.AllocReg()
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d129)
					if d129.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r4, d129)
					}
					ctx.EmitJmp(lbl18)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					d129 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase128) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d134 JITValueDesc
					if d126.SliceSizeKnown {
						d134 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d126.KnownSliceLen))}
					} else if d126.Loc == LocImm {
						d134 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d126.Imm.String())))}
					} else if d126.Loc == LocStackTriple {
						d134 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else if d126.Loc == LocStackPair {
						d134 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d126)
						if d126.Loc == LocRegPair || d126.Loc == LocRegTriple {
							d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2, ID: 0}
						} else if d126.Loc == LocReg {
							d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d135 JITValueDesc
					if d127.SliceSizeKnown {
						d135 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d127.KnownSliceLen))}
					} else if d127.Loc == LocImm {
						d135 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d127.StackOff))}
					} else if d127.Loc == LocStackTriple {
						d135 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d127.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d127)
						if d127.Loc == LocRegPair || d127.Loc == LocRegTriple {
							d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d127.Reg2, ID: 0}
						} else if d127.Loc == LocReg {
							d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d127.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d134)
					ctx.EnsureDesc(&d135)
					ctx.EnsureDescsTogether(&d134, &d135)
					var d136 JITValueDesc
					if d134.Loc == LocImm && d135.Loc == LocImm {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d134.Imm.Int() - d135.Imm.Int())}
					} else if d135.Loc == LocImm && d135.Imm.Int() == 0 {
						r5 := ctx.AllocRegExcept(d134.Reg)
						ctx.EmitMovRegReg(r5, d134.Reg)
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d136)
					} else if d134.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d135.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
						ctx.EmitSubInt64(scratch, d135.Reg)
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d136)
					} else if d135.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d134.Reg)
						ctx.EmitMovRegReg(scratch, d134.Reg)
						if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d135.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d135.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d136)
					} else {
						r6 := ctx.AllocRegExcept(d134.Reg, d135.Reg)
						ctx.EmitMovRegReg(r6, d134.Reg)
						ctx.EmitSubInt64(r6, d135.Reg)
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d136)
					}
					if d136.Loc == LocReg && d134.Loc == LocReg && d136.Reg == d134.Reg {
						ctx.TransferReg(d134.Reg)
						d134.Loc = LocNone
					}
					ctx.FreeDesc(&d134)
					ctx.FreeDesc(&d135)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					var d137 JITValueDesc
					ctx.EnsureDesc(&d126)
					if d126.Loc == LocRegPair || d126.Loc == LocRegTriple {
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2}
						ctx.BindReg(d126.Reg2, &d137)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d126)
					ctx.EnsureDesc(&d136)
					ctx.EnsureDesc(&d137)
					var d139 JITValueDesc
					if d137.Loc == LocImm && d136.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d137.Imm.Int() - d136.Imm.Int())}
					} else {
						r7 := ctx.AllocReg()
						if d137.Loc == LocImm {
							ctx.EmitMovRegImm64(r7, uint64(d137.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r7, d137.Reg)
						}
						if d136.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d136.Imm.Int()))
							ctx.EmitSubInt64(r7, RegR11)
						} else {
							ctx.EmitSubInt64(r7, d136.Reg)
						}
						d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d139)
					}
					var d140 JITValueDesc
					r8 := ctx.EmitSliceDataAfterLow(&d126, &d136, 1)
					d140 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d140)
					ctx.BindReg(r8, &d140)
					var d141 JITValueDesc
					var r9 Reg
					var r10 Reg
					ctx.SyncDesc(&d140)
					ctx.EnsureDesc(&d140)
					if d140.Loc == LocImm {
						r9 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r9, uint64(d140.Imm.Int()))
					} else {
						r9 = d140.Reg
					}
					ctx.ProtectReg(r9)
					ctx.SyncDesc(&d139)
					ctx.EnsureDesc(&d139)
					if d139.Loc == LocImm {
						r10 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r10, uint64(d139.Imm.Int()))
					} else {
						r10 = d139.Reg
					}
					ctx.ProtectReg(r10)
					ctx.UnprotectReg(r10)
					ctx.UnprotectReg(r9)
					d141 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d141)
					ctx.BindReg(r10, &d141)
					ctx.BindReg(r9, &d141)
					ctx.BindReg(r10, &d141)
					ctx.FreeDesc(&d136)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d141)
					ctx.EnsureDesc(&d127)
					var d142 JITValueDesc
					if d127.Loc == LocImm {
						ctx.TrackImm(d127.Imm)
						ptrWord, _ := d127.Imm.RawWords()
						d142 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d142.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d142.Reg2, uint64(len(d127.Imm.String())))
						ctx.BindReg(d142.Reg, &d142)
						ctx.BindReg(d142.Reg2, &d142)
					} else {
						d142 = d127
					}
					d143 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d141, d142}, 1)
					ctx.EmitAndRegImm32(d143.Reg, 1)
					d143.Type = tagBool
					ctx.BindReg(d143.Reg, &d143)
					ctx.EnsureDesc(&d143)
					ctx.EmitStoreToStack(d143, int32(phiBase128)+int32(0))
					ctx.StabilizeDescForControlFlow(&d143)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl18)
					d144 = JITValueDesc{Loc: LocReg, Reg: r4}
					ctx.BindReg(r4, &d144)
					ctx.BindReg(r4, &d144)
					ctx.ReclaimUntrackedRegs()
					d145 = d144
					ctx.EnsureDesc(&d145)
					if d145.Loc != LocImm && d145.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d145.Loc == LocImm {
						if d145.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl16)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitCmpRegImm32(d145.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl17)
					}
					ctx.FreeDesc(&d144)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d122)
					ctx.EmitCopyDescWords(&d125, &d122, 2)
					ctx.EmitJmp(lbl14)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d146 JITValueDesc
					if d122.SliceSizeKnown {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d122.KnownSliceLen))}
					} else if d122.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d122.Imm.String())))}
					} else if d122.Loc == LocStackTriple {
						d146 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d122.StackOff + 8, NoHeapPointer: true}
					} else if d122.Loc == LocStackPair {
						d146 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d122.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d122)
						if d122.Loc == LocRegPair || d122.Loc == LocRegTriple {
							d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d122.Reg2, ID: 0}
						} else if d122.Loc == LocReg {
							d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d122.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d147 JITValueDesc
					if d123.SliceSizeKnown {
						d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d123.KnownSliceLen))}
					} else if d123.Loc == LocImm {
						d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d123.StackOff))}
					} else if d123.Loc == LocStackTriple {
						d147 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d123.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d123)
						if d123.Loc == LocRegPair || d123.Loc == LocRegTriple {
							d147 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d123.Reg2, ID: 0}
						} else if d123.Loc == LocReg {
							d147 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d123.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d146)
					ctx.EnsureDesc(&d147)
					ctx.EnsureDescsTogether(&d146, &d147)
					var d148 JITValueDesc
					if d146.Loc == LocImm && d147.Loc == LocImm {
						d148 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d146.Imm.Int() - d147.Imm.Int())}
					} else if d147.Loc == LocImm && d147.Imm.Int() == 0 {
						r11 := ctx.AllocRegExcept(d146.Reg)
						ctx.EmitMovRegReg(r11, d146.Reg)
						d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d148)
					} else if d146.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d147.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
						ctx.EmitSubInt64(scratch, d147.Reg)
						d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d148)
					} else if d147.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d146.Reg)
						ctx.EmitMovRegReg(scratch, d146.Reg)
						if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d147.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d147.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d148)
					} else {
						r12 := ctx.AllocRegExcept(d146.Reg, d147.Reg)
						ctx.EmitMovRegReg(r12, d146.Reg)
						ctx.EmitSubInt64(r12, d147.Reg)
						d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
						ctx.BindReg(r12, &d148)
					}
					if d148.Loc == LocReg && d146.Loc == LocReg && d148.Reg == d146.Reg {
						ctx.TransferReg(d146.Reg)
						d146.Loc = LocNone
					}
					ctx.FreeDesc(&d146)
					ctx.FreeDesc(&d147)
					ctx.ReclaimUntrackedRegs()
					d149 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d148)
					ctx.EnsureDesc(&d122)
					ctx.EnsureDesc(&d149)
					ctx.EnsureDesc(&d148)
					var d151 JITValueDesc
					if d148.Loc == LocImm && d149.Loc == LocImm {
						d151 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d148.Imm.Int() - d149.Imm.Int())}
					} else {
						r13 := ctx.AllocReg()
						if d148.Loc == LocImm {
							ctx.EmitMovRegImm64(r13, uint64(d148.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r13, d148.Reg)
						}
						if d149.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d149.Imm.Int()))
							ctx.EmitSubInt64(r13, RegR11)
						} else {
							ctx.EmitSubInt64(r13, d149.Reg)
						}
						d151 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d151)
					}
					var d152 JITValueDesc
					r14 := ctx.EmitSliceDataAfterLow(&d122, &d149, 1)
					d152 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d152)
					ctx.BindReg(r14, &d152)
					var d153 JITValueDesc
					var r15 Reg
					var r16 Reg
					ctx.SyncDesc(&d152)
					ctx.EnsureDesc(&d152)
					if d152.Loc == LocImm {
						r15 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r15, uint64(d152.Imm.Int()))
					} else {
						r15 = d152.Reg
					}
					ctx.ProtectReg(r15)
					ctx.SyncDesc(&d151)
					ctx.EnsureDesc(&d151)
					if d151.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d151.Imm.Int()))
					} else {
						r16 = d151.Reg
					}
					ctx.ProtectReg(r16)
					ctx.UnprotectReg(r16)
					ctx.UnprotectReg(r15)
					d153 = JITValueDesc{Loc: LocRegPair, Reg: r15, Reg2: r16}
					ctx.BindReg(r15, &d153)
					ctx.BindReg(r16, &d153)
					ctx.BindReg(r15, &d153)
					ctx.BindReg(r16, &d153)
					ctx.FreeDesc(&d148)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d153)
					ctx.EmitCopyDescWords(&d125, &d153, 2)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl14)
					ctx.FreeDesc(&d121)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d125)
					ctx.EnsureDesc(&d125)
					d154 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d125}, 2)
					ctx.EmitMovPairToResult(&d154, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps155 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps155)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["json_encode_assoc"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				declaration := declarations["json_decode"]
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
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					if d6.Loc != LocRegPair && d6.Loc != LocStackPair && d6.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (json.Unmarshal arg1)")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d6)
					d7 = ctx.EmitGoCallScalar(GoFuncAddr(json.Unmarshal), []JITValueDesc{d4, d6}, 2)
					d7.NoHeapPointer = false
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
					snap12 := d0
					snap13 := d1
					snap14 := d2
					snap15 := d3
					snap16 := d4
					snap17 := d6
					snap18 := d7
					snap19 := d8
					snap20 := d9
					alloc21 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc21)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc21)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					ps22 := PhiState{General: true}
					ps22.OverlayValues = make([]JITValueDesc, 10)
					ps22.OverlayValues[0] = d0
					ps22.OverlayValues[1] = d1
					ps22.OverlayValues[2] = d2
					ps22.OverlayValues[3] = d3
					ps22.OverlayValues[4] = d4
					ps22.OverlayValues[6] = d6
					ps22.OverlayValues[7] = d7
					ps22.OverlayValues[8] = d8
					ps22.OverlayValues[9] = d9
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 10)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[6] = d6
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[8] = d8
					ps23.OverlayValues[9] = d9
					snap24 := d0
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d4
					snap29 := d6
					snap30 := d7
					snap31 := d8
					snap32 := d9
					alloc33 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps23)
					}
					ctx.RestoreAllocState(alloc33)
					d0 = snap24
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d6 = snap29
					d7 = snap30
					d8 = snap31
					d9 = snap32
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps22)
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
					ctx.StabilizeDescForControlFlow(&d0)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *any) any { return *value }), []JITValueDesc{d0}, 2)
					ctx.EnsureDesc(&d34)
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
					if d34.Loc != LocRegPair && d34.Loc != LocStackPair && d34.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (TransformFromJSON arg0)")
					}
					ctx.SyncDesc(&d34)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(TransformFromJSON), []JITValueDesc{d34}, 2)
					d35.NoHeapPointer = false
					ctx.BindReg(d35.Reg, &d35)
					ctx.BindReg(d35.Reg2, &d35)
					ctx.FreeDesc(&d34)
					ctx.SyncDesc(&d35)
					if d35.Loc == LocRegPair || d35.Loc == LocStackPair || d35.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d35, &result)
						result.Type = d35.Type
					} else {
						switch d35.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d35)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d35)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d35)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d35, &result)
							result.Type = d35.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps36 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps36)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["json_decode_scmer"]
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
				var d34 JITValueDesc
				_ = d34
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					if d6.Loc != LocRegPair && d6.Loc != LocStackPair && d6.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (json.Unmarshal arg1)")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d6)
					d7 = ctx.EmitGoCallScalar(GoFuncAddr(json.Unmarshal), []JITValueDesc{d4, d6}, 2)
					d7.NoHeapPointer = false
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
					snap12 := d0
					snap13 := d1
					snap14 := d2
					snap15 := d3
					snap16 := d4
					snap17 := d6
					snap18 := d7
					snap19 := d8
					snap20 := d9
					alloc21 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc21)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc21)
					d0 = snap12
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					ps22 := PhiState{General: true}
					ps22.OverlayValues = make([]JITValueDesc, 10)
					ps22.OverlayValues[0] = d0
					ps22.OverlayValues[1] = d1
					ps22.OverlayValues[2] = d2
					ps22.OverlayValues[3] = d3
					ps22.OverlayValues[4] = d4
					ps22.OverlayValues[6] = d6
					ps22.OverlayValues[7] = d7
					ps22.OverlayValues[8] = d8
					ps22.OverlayValues[9] = d9
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 10)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[6] = d6
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[8] = d8
					ps23.OverlayValues[9] = d9
					snap24 := d0
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d4
					snap29 := d6
					snap30 := d7
					snap31 := d8
					snap32 := d9
					alloc33 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps23)
					}
					ctx.RestoreAllocState(alloc33)
					d0 = snap24
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d6 = snap29
					d7 = snap30
					d8 = snap31
					d9 = snap32
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps22)
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
					d34 = d0
					_ = d34
					ctx.SyncDesc(&d34)
					if d34.Loc == LocRegPair || d34.Loc == LocStackPair || d34.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d34, &result)
						result.Type = d34.Type
					} else {
						switch d34.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d34)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d34)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d34)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d34, &result)
							result.Type = d34.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps35 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps35)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["base64_encode"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *base64.Encoding { return base64.StdEncoding }), nil, 1)
				d1 := args[0]
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
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
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
				d8.NoHeapPointer = true
				ctx.BindReg(d8.Reg, &d8)
				ctx.FreeDesc(&d7)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d8)
				ctx.ReclaimUntrackedRegs()
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
				ctx.FreeDesc(&d0)
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
				declaration := declarations["base64_decode"]
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
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d31 JITValueDesc
				_ = d31
				var d33 JITValueDesc
				_ = d33
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
					snap11 := d0
					snap12 := d1
					snap13 := d2
					snap14 := d3
					snap15 := d5
					snap16 := d6
					snap17 := d7
					snap18 := d8
					alloc19 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc19)
					d0 = snap11
					d1 = snap12
					d2 = snap13
					d3 = snap14
					d5 = snap15
					d6 = snap16
					d7 = snap17
					d8 = snap18
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc19)
					d0 = snap11
					d1 = snap12
					d2 = snap13
					d3 = snap14
					d5 = snap15
					d6 = snap16
					d7 = snap17
					d8 = snap18
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 9)
					ps20.OverlayValues[0] = d0
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
					ps21 := PhiState{General: true}
					ps21.OverlayValues = make([]JITValueDesc, 9)
					ps21.OverlayValues[0] = d0
					ps21.OverlayValues[1] = d1
					ps21.OverlayValues[2] = d2
					ps21.OverlayValues[3] = d3
					ps21.OverlayValues[5] = d5
					ps21.OverlayValues[6] = d6
					ps21.OverlayValues[7] = d7
					ps21.OverlayValues[8] = d8
					snap22 := d0
					snap23 := d1
					snap24 := d2
					snap25 := d3
					snap26 := d5
					snap27 := d6
					snap28 := d7
					snap29 := d8
					alloc30 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps21)
					}
					ctx.RestoreAllocState(alloc30)
					d0 = snap22
					d1 = snap23
					d2 = snap24
					d3 = snap25
					d5 = snap26
					d6 = snap27
					d7 = snap28
					d8 = snap29
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps20)
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
					callResults32 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d5}, []uint8{2}, []uint8{1})
					d31 = callResults32[0]
					ctx.EnsureDesc(&d31)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d31}, 2)
					ctx.EmitMovPairToResult(&d33, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps34 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps34)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  21,
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
				declaration := declarations["bin2hex"]
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
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d11 JITValueDesc
				_ = d11
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
				var d20 JITValueDesc
				_ = d20
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
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 37}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					d6 = d4
					ctx.SyncDesc(&d6)
					if d6.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d6.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d6.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d6 = tmpScalar
					}
					d6 = JITPrepareScmerGoArg(ctx, d6)
					if d6.Loc != LocRegPair && d6.Loc != LocStackPair && d6.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d5 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d6}, 2)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					var d7 JITValueDesc
					if d5.SliceSizeKnown {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d5.Imm.String())))}
					} else if d5.Loc == LocStackTriple {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else if d5.Loc == LocStackPair {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d7)
					ctx.EnsureDescsTogether(&d8, &d7)
					var d9 JITValueDesc
					if d8.Loc == LocImm && d7.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() * d7.Imm.Int())}
					} else if d8.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
						ctx.EmitImulInt64(scratch, d7.Reg)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d9)
					} else if d7.Loc == LocImm {
						if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d8.Reg, int32(d7.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
							ctx.EmitImulInt64(d8.Reg, RegR11)
						}
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
						ctx.BindReg(d8.Reg, &d9)
					} else {
						ctx.EmitImulInt64(d8.Reg, d7.Reg)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
						ctx.BindReg(d8.Reg, &d9)
					}
					if d9.Loc == LocReg && d8.Loc == LocReg && d9.Reg == d8.Reg {
						ctx.TransferReg(d8.Reg)
						d8.Loc = LocNone
					}
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					callResults10 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d9, d9}, []uint8{3}, []uint8{1})
					d11 = callResults10[0]
					d11.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.FreeDesc(&d9)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps12 := PhiState{General: ps.General}
					ps12.OverlayValues = make([]JITValueDesc, 12)
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[5] = d5
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					ps12.OverlayValues[9] = d9
					ps12.OverlayValues[11] = d11
					ps12.PhiValues = make([]JITValueDesc, 1)
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps12.PhiValues[0] = d13
					if ps12.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps12)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d14 := ps.PhiValues[0]
							ctx.EnsureDesc(&d14)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d14)
							} else {
								ctx.EmitStoreToStack(d14, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					var d15 JITValueDesc
					if d5.SliceSizeKnown {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d5.Imm.String())))}
					} else if d5.Loc == LocStackTriple {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else if d5.Loc == LocStackPair {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDescsTogether(&d3, &d15)
					var d16 JITValueDesc
					if d3.Loc == LocImm && d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d3.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, RegR11)
						}
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d16)
					} else if d3.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d15.Reg)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d16)
					} else {
						r3 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d15.Reg)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d16)
					}
					ctx.FreeDesc(&d15)
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							if ps.General {
							}
							ps18 := PhiState{General: ps.General}
							ps18.OverlayValues = make([]JITValueDesc, 18)
							ps18.OverlayValues[3] = d3
							ps18.OverlayValues[4] = d4
							ps18.OverlayValues[5] = d5
							ps18.OverlayValues[6] = d6
							ps18.OverlayValues[7] = d7
							ps18.OverlayValues[8] = d8
							ps18.OverlayValues[9] = d9
							ps18.OverlayValues[11] = d11
							ps18.OverlayValues[13] = d13
							ps18.OverlayValues[14] = d14
							ps18.OverlayValues[15] = d15
							ps18.OverlayValues[16] = d16
							ps18.OverlayValues[17] = d17
							return bbs[2].RenderPS(ps18)
						}
						if ps.General {
						}
						ps19 := PhiState{General: ps.General}
						ps19.OverlayValues = make([]JITValueDesc, 18)
						ps19.OverlayValues[3] = d3
						ps19.OverlayValues[4] = d4
						ps19.OverlayValues[5] = d5
						ps19.OverlayValues[6] = d6
						ps19.OverlayValues[7] = d7
						ps19.OverlayValues[8] = d8
						ps19.OverlayValues[9] = d9
						ps19.OverlayValues[11] = d11
						ps19.OverlayValues[13] = d13
						ps19.OverlayValues[14] = d14
						ps19.OverlayValues[15] = d15
						ps19.OverlayValues[16] = d16
						ps19.OverlayValues[17] = d17
						return bbs[3].RenderPS(ps19)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d20 := ps.PhiValues[0]
							ctx.EnsureDesc(&d20)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d20)
							} else {
								ctx.EmitStoreToStack(d20, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitJump(d17.Condition, lbl5)
					ctx.EmitJmp(lbl6)
					snap21 := d3
					snap22 := d4
					snap23 := d5
					snap24 := d6
					snap25 := d7
					snap26 := d8
					snap27 := d9
					snap28 := d11
					snap29 := d13
					snap30 := d14
					snap31 := d15
					snap32 := d16
					snap33 := d17
					snap34 := d20
					alloc35 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc35)
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d8 = snap26
					d9 = snap27
					d11 = snap28
					d13 = snap29
					d14 = snap30
					d15 = snap31
					d16 = snap32
					d17 = snap33
					d20 = snap34
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc35)
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d8 = snap26
					d9 = snap27
					d11 = snap28
					d13 = snap29
					d14 = snap30
					d15 = snap31
					d16 = snap32
					d17 = snap33
					d20 = snap34
					ps36 := PhiState{General: true}
					ps36.OverlayValues = make([]JITValueDesc, 21)
					ps36.OverlayValues[3] = d3
					ps36.OverlayValues[4] = d4
					ps36.OverlayValues[5] = d5
					ps36.OverlayValues[6] = d6
					ps36.OverlayValues[7] = d7
					ps36.OverlayValues[8] = d8
					ps36.OverlayValues[9] = d9
					ps36.OverlayValues[11] = d11
					ps36.OverlayValues[13] = d13
					ps36.OverlayValues[14] = d14
					ps36.OverlayValues[15] = d15
					ps36.OverlayValues[16] = d16
					ps36.OverlayValues[17] = d17
					ps36.OverlayValues[20] = d20
					ps37 := PhiState{General: true}
					ps37.OverlayValues = make([]JITValueDesc, 21)
					ps37.OverlayValues[3] = d3
					ps37.OverlayValues[4] = d4
					ps37.OverlayValues[5] = d5
					ps37.OverlayValues[6] = d6
					ps37.OverlayValues[7] = d7
					ps37.OverlayValues[8] = d8
					ps37.OverlayValues[9] = d9
					ps37.OverlayValues[11] = d11
					ps37.OverlayValues[13] = d13
					ps37.OverlayValues[14] = d14
					ps37.OverlayValues[15] = d15
					ps37.OverlayValues[16] = d16
					ps37.OverlayValues[17] = d17
					ps37.OverlayValues[20] = d20
					snap38 := d3
					snap39 := d4
					snap40 := d5
					snap41 := d6
					snap42 := d7
					snap43 := d8
					snap44 := d9
					snap45 := d11
					snap46 := d13
					snap47 := d14
					snap48 := d15
					snap49 := d16
					snap50 := d17
					snap51 := d20
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps37)
					}
					ctx.RestoreAllocState(alloc52)
					d3 = snap38
					d4 = snap39
					d5 = snap40
					d6 = snap41
					d7 = snap42
					d8 = snap43
					d9 = snap44
					d11 = snap45
					d13 = snap46
					d14 = snap47
					d15 = snap48
					d16 = snap49
					d17 = snap50
					d20 = snap51
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps36)
					}
					return result
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d11)
					d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDescsTogether(&d53, &d3)
					var d54 JITValueDesc
					if d53.Loc == LocImm && d3.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d53.Imm.Int() * d3.Imm.Int())}
					} else if d53.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
						ctx.EmitImulInt64(scratch, d3.Reg)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d54)
					} else if d3.Loc == LocImm {
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d53.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
							ctx.EmitImulInt64(d53.Reg, RegR11)
						}
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg}
						ctx.BindReg(d53.Reg, &d54)
					} else {
						ctx.EmitImulInt64(d53.Reg, d3.Reg)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg}
						ctx.BindReg(d53.Reg, &d54)
					}
					if d54.Loc == LocReg && d53.Loc == LocReg && d54.Reg == d53.Reg {
						ctx.TransferReg(d53.Reg)
						d53.Loc = LocNone
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d3)
					ctx.EnsureGoStringHeader(&d5)
					d55 = ctx.EmitSliceElementAddress(&d5, &d3, 1)
					ctx.EnsureDesc(&d55)
					r4 := ctx.AllocRegExcept(d55.Reg)
					ctx.EmitMovRegMemB(r4, d55.Reg, 0)
					ctx.FreeDesc(&d55)
					d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4, NoHeapPointer: true}
					ctx.BindReg(r4, &d56)
					ctx.BindReg(r4, &d56)
					ctx.EnsureDesc(&d56)
					var d57 JITValueDesc
					if d56.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d56.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d56.Reg, 4)
						d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg}
						ctx.BindReg(d56.Reg, &d57)
					}
					if d57.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: d57.Type, Imm: NewInt(int64(uint64(d57.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d57.Reg, 56)
						ctx.EmitShrRegImm8(d57.Reg, 56)
					}
					if d57.Loc == LocReg && d56.Loc == LocReg && d57.Reg == d56.Reg {
						ctx.TransferReg(d56.Reg)
						d56.Loc = LocNone
					}
					ctx.FreeDesc(&d56)
					d58 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d57)
					ctx.EnsureGoStringHeader(&d58)
					d59 = ctx.EmitSliceElementAddress(&d58, &d57, 1)
					ctx.EnsureDesc(&d59)
					r5 := ctx.AllocRegExcept(d59.Reg)
					ctx.EmitMovRegMemB(r5, d59.Reg, 0)
					ctx.FreeDesc(&d59)
					d60 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, NoHeapPointer: true}
					ctx.BindReg(r5, &d60)
					ctx.BindReg(r5, &d60)
					ctx.FreeDesc(&d57)
					ctx.EnsureDesc(&d54)
					ctx.SyncDesc(&d60)
					ctx.StabilizeDescAcrossNestedCall(&d54)
					d61 = d11
					d61.ID = 0
					d62 = d54
					d62.ID = 0
					d63 = ctx.EmitSliceElementAddress(&d61, &d62, int32(1))
					ctx.FreeDesc(&d62)
					ctx.EmitStoreScalarAt(&d63, &d60, 1)
					ctx.FreeDesc(&d63)
					ctx.FreeDesc(&d54)
					ctx.FreeDesc(&d60)
					d64 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDescsTogether(&d64, &d3)
					var d65 JITValueDesc
					if d64.Loc == LocImm && d3.Loc == LocImm {
						d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d64.Imm.Int() * d3.Imm.Int())}
					} else if d64.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
						ctx.EmitImulInt64(scratch, d3.Reg)
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d65)
					} else if d3.Loc == LocImm {
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d64.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
							ctx.EmitImulInt64(d64.Reg, RegR11)
						}
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d64.Reg}
						ctx.BindReg(d64.Reg, &d65)
					} else {
						ctx.EmitImulInt64(d64.Reg, d3.Reg)
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d64.Reg}
						ctx.BindReg(d64.Reg, &d65)
					}
					if d65.Loc == LocReg && d64.Loc == LocReg && d65.Reg == d64.Reg {
						ctx.TransferReg(d64.Reg)
						d64.Loc = LocNone
					}
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					var d66 JITValueDesc
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegReg(scratch, d65.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d66)
					}
					if d66.Loc == LocReg && d65.Loc == LocReg && d66.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.FreeDesc(&d65)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d3)
					ctx.EnsureGoStringHeader(&d5)
					d67 = ctx.EmitSliceElementAddress(&d5, &d3, 1)
					ctx.EnsureDesc(&d67)
					r6 := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegMemB(r6, d67.Reg, 0)
					ctx.FreeDesc(&d67)
					d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d68)
					ctx.BindReg(r6, &d68)
					ctx.EnsureDesc(&d68)
					var d69 JITValueDesc
					if d68.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d68.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d68.Reg, 15)
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d68.Reg}
						ctx.BindReg(d68.Reg, &d69)
					}
					if d69.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: d69.Type, Imm: NewInt(int64(uint64(d69.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d69.Reg, 56)
						ctx.EmitShrRegImm8(d69.Reg, 56)
					}
					if d69.Loc == LocReg && d68.Loc == LocReg && d69.Reg == d68.Reg {
						ctx.TransferReg(d68.Reg)
						d68.Loc = LocNone
					}
					ctx.FreeDesc(&d68)
					d70 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d69)
					ctx.EnsureGoStringHeader(&d70)
					d71 = ctx.EmitSliceElementAddress(&d70, &d69, 1)
					ctx.EnsureDesc(&d71)
					r7 := ctx.AllocRegExcept(d71.Reg)
					ctx.EmitMovRegMemB(r7, d71.Reg, 0)
					ctx.FreeDesc(&d71)
					d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d72)
					ctx.BindReg(r7, &d72)
					ctx.FreeDesc(&d69)
					ctx.EnsureDesc(&d66)
					ctx.SyncDesc(&d72)
					ctx.StabilizeDescAcrossNestedCall(&d66)
					d73 = d11
					d73.ID = 0
					d74 = d66
					d74.ID = 0
					d75 = ctx.EmitSliceElementAddress(&d73, &d74, int32(1))
					ctx.FreeDesc(&d74)
					ctx.EmitStoreScalarAt(&d75, &d72, 1)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d66)
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d76 JITValueDesc
					if d3.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d3.Reg)
						}
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d76)
					}
					if d76.Loc == LocReg && d3.Loc == LocReg && d76.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d76)
						if d76.Loc == LocReg {
							ctx.ProtectReg(d76.Reg)
						} else if d76.Loc == LocRegPair {
							ctx.ProtectReg(d76.Reg)
							ctx.ProtectReg(d76.Reg2)
						}
						d77 = d76
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d77)
						} else {
							ctx.EmitStoreToStack(d77, int32(bbs[1].PhiBase)+int32(0))
						}
						if d76.Loc == LocReg {
							ctx.UnprotectReg(d76.Reg)
						} else if d76.Loc == LocRegPair {
							ctx.UnprotectReg(d76.Reg)
							ctx.UnprotectReg(d76.Reg2)
						}
					}
					ps78 := PhiState{General: ps.General}
					ps78.OverlayValues = make([]JITValueDesc, 78)
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[4] = d4
					ps78.OverlayValues[5] = d5
					ps78.OverlayValues[6] = d6
					ps78.OverlayValues[7] = d7
					ps78.OverlayValues[8] = d8
					ps78.OverlayValues[9] = d9
					ps78.OverlayValues[11] = d11
					ps78.OverlayValues[13] = d13
					ps78.OverlayValues[14] = d14
					ps78.OverlayValues[15] = d15
					ps78.OverlayValues[16] = d16
					ps78.OverlayValues[17] = d17
					ps78.OverlayValues[20] = d20
					ps78.OverlayValues[53] = d53
					ps78.OverlayValues[54] = d54
					ps78.OverlayValues[55] = d55
					ps78.OverlayValues[56] = d56
					ps78.OverlayValues[57] = d57
					ps78.OverlayValues[58] = d58
					ps78.OverlayValues[59] = d59
					ps78.OverlayValues[60] = d60
					ps78.OverlayValues[61] = d61
					ps78.OverlayValues[62] = d62
					ps78.OverlayValues[63] = d63
					ps78.OverlayValues[64] = d64
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
					ps78.OverlayValues[75] = d75
					ps78.OverlayValues[76] = d76
					ps78.OverlayValues[77] = d77
					ps78.PhiValues = make([]JITValueDesc, 1)
					d79 = d76
					ps78.PhiValues[0] = d79
					if ps78.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps78)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
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
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					callResults81 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d11}, []uint8{2}, []uint8{1})
					d80 = callResults81[0]
					ctx.EnsureDesc(&d80)
					d82 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d80}, 2)
					ctx.EmitMovPairToResult(&d82, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps83 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps83)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["bin2hex"]
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
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d11 JITValueDesc
				_ = d11
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
				var d20 JITValueDesc
				_ = d20
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
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 37}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					d6 = d4
					ctx.SyncDesc(&d6)
					if d6.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d6.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d6.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d6 = tmpScalar
					}
					d6 = JITPrepareScmerGoArg(ctx, d6)
					if d6.Loc != LocRegPair && d6.Loc != LocStackPair && d6.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d5 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d6}, 2)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					var d7 JITValueDesc
					if d5.SliceSizeKnown {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d5.Imm.String())))}
					} else if d5.Loc == LocStackTriple {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else if d5.Loc == LocStackPair {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d7)
					ctx.EnsureDescsTogether(&d8, &d7)
					var d9 JITValueDesc
					if d8.Loc == LocImm && d7.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() * d7.Imm.Int())}
					} else if d8.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
						ctx.EmitImulInt64(scratch, d7.Reg)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d9)
					} else if d7.Loc == LocImm {
						if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d8.Reg, int32(d7.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
							ctx.EmitImulInt64(d8.Reg, RegR11)
						}
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
						ctx.BindReg(d8.Reg, &d9)
					} else {
						ctx.EmitImulInt64(d8.Reg, d7.Reg)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
						ctx.BindReg(d8.Reg, &d9)
					}
					if d9.Loc == LocReg && d8.Loc == LocReg && d9.Reg == d8.Reg {
						ctx.TransferReg(d8.Reg)
						d8.Loc = LocNone
					}
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					callResults10 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d9, d9}, []uint8{3}, []uint8{1})
					d11 = callResults10[0]
					d11.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.FreeDesc(&d9)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps12 := PhiState{General: ps.General}
					ps12.OverlayValues = make([]JITValueDesc, 12)
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[5] = d5
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					ps12.OverlayValues[9] = d9
					ps12.OverlayValues[11] = d11
					ps12.PhiValues = make([]JITValueDesc, 1)
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps12.PhiValues[0] = d13
					if ps12.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps12)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d14 := ps.PhiValues[0]
							ctx.EnsureDesc(&d14)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d14)
							} else {
								ctx.EmitStoreToStack(d14, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					var d15 JITValueDesc
					if d5.SliceSizeKnown {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d5.Imm.String())))}
					} else if d5.Loc == LocStackTriple {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else if d5.Loc == LocStackPair {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDescsTogether(&d3, &d15)
					var d16 JITValueDesc
					if d3.Loc == LocImm && d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d3.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, RegR11)
						}
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d16)
					} else if d3.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d15.Reg)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d16)
					} else {
						r3 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d15.Reg)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d16)
					}
					ctx.FreeDesc(&d15)
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							if ps.General {
							}
							ps18 := PhiState{General: ps.General}
							ps18.OverlayValues = make([]JITValueDesc, 18)
							ps18.OverlayValues[3] = d3
							ps18.OverlayValues[4] = d4
							ps18.OverlayValues[5] = d5
							ps18.OverlayValues[6] = d6
							ps18.OverlayValues[7] = d7
							ps18.OverlayValues[8] = d8
							ps18.OverlayValues[9] = d9
							ps18.OverlayValues[11] = d11
							ps18.OverlayValues[13] = d13
							ps18.OverlayValues[14] = d14
							ps18.OverlayValues[15] = d15
							ps18.OverlayValues[16] = d16
							ps18.OverlayValues[17] = d17
							return bbs[2].RenderPS(ps18)
						}
						if ps.General {
						}
						ps19 := PhiState{General: ps.General}
						ps19.OverlayValues = make([]JITValueDesc, 18)
						ps19.OverlayValues[3] = d3
						ps19.OverlayValues[4] = d4
						ps19.OverlayValues[5] = d5
						ps19.OverlayValues[6] = d6
						ps19.OverlayValues[7] = d7
						ps19.OverlayValues[8] = d8
						ps19.OverlayValues[9] = d9
						ps19.OverlayValues[11] = d11
						ps19.OverlayValues[13] = d13
						ps19.OverlayValues[14] = d14
						ps19.OverlayValues[15] = d15
						ps19.OverlayValues[16] = d16
						ps19.OverlayValues[17] = d17
						return bbs[3].RenderPS(ps19)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d20 := ps.PhiValues[0]
							ctx.EnsureDesc(&d20)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d20)
							} else {
								ctx.EmitStoreToStack(d20, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitJump(d17.Condition, lbl5)
					ctx.EmitJmp(lbl6)
					snap21 := d3
					snap22 := d4
					snap23 := d5
					snap24 := d6
					snap25 := d7
					snap26 := d8
					snap27 := d9
					snap28 := d11
					snap29 := d13
					snap30 := d14
					snap31 := d15
					snap32 := d16
					snap33 := d17
					snap34 := d20
					alloc35 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc35)
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d8 = snap26
					d9 = snap27
					d11 = snap28
					d13 = snap29
					d14 = snap30
					d15 = snap31
					d16 = snap32
					d17 = snap33
					d20 = snap34
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc35)
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d8 = snap26
					d9 = snap27
					d11 = snap28
					d13 = snap29
					d14 = snap30
					d15 = snap31
					d16 = snap32
					d17 = snap33
					d20 = snap34
					ps36 := PhiState{General: true}
					ps36.OverlayValues = make([]JITValueDesc, 21)
					ps36.OverlayValues[3] = d3
					ps36.OverlayValues[4] = d4
					ps36.OverlayValues[5] = d5
					ps36.OverlayValues[6] = d6
					ps36.OverlayValues[7] = d7
					ps36.OverlayValues[8] = d8
					ps36.OverlayValues[9] = d9
					ps36.OverlayValues[11] = d11
					ps36.OverlayValues[13] = d13
					ps36.OverlayValues[14] = d14
					ps36.OverlayValues[15] = d15
					ps36.OverlayValues[16] = d16
					ps36.OverlayValues[17] = d17
					ps36.OverlayValues[20] = d20
					ps37 := PhiState{General: true}
					ps37.OverlayValues = make([]JITValueDesc, 21)
					ps37.OverlayValues[3] = d3
					ps37.OverlayValues[4] = d4
					ps37.OverlayValues[5] = d5
					ps37.OverlayValues[6] = d6
					ps37.OverlayValues[7] = d7
					ps37.OverlayValues[8] = d8
					ps37.OverlayValues[9] = d9
					ps37.OverlayValues[11] = d11
					ps37.OverlayValues[13] = d13
					ps37.OverlayValues[14] = d14
					ps37.OverlayValues[15] = d15
					ps37.OverlayValues[16] = d16
					ps37.OverlayValues[17] = d17
					ps37.OverlayValues[20] = d20
					snap38 := d3
					snap39 := d4
					snap40 := d5
					snap41 := d6
					snap42 := d7
					snap43 := d8
					snap44 := d9
					snap45 := d11
					snap46 := d13
					snap47 := d14
					snap48 := d15
					snap49 := d16
					snap50 := d17
					snap51 := d20
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps37)
					}
					ctx.RestoreAllocState(alloc52)
					d3 = snap38
					d4 = snap39
					d5 = snap40
					d6 = snap41
					d7 = snap42
					d8 = snap43
					d9 = snap44
					d11 = snap45
					d13 = snap46
					d14 = snap47
					d15 = snap48
					d16 = snap49
					d17 = snap50
					d20 = snap51
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps36)
					}
					return result
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d11)
					d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDescsTogether(&d53, &d3)
					var d54 JITValueDesc
					if d53.Loc == LocImm && d3.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d53.Imm.Int() * d3.Imm.Int())}
					} else if d53.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
						ctx.EmitImulInt64(scratch, d3.Reg)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d54)
					} else if d3.Loc == LocImm {
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d53.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
							ctx.EmitImulInt64(d53.Reg, RegR11)
						}
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg}
						ctx.BindReg(d53.Reg, &d54)
					} else {
						ctx.EmitImulInt64(d53.Reg, d3.Reg)
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg}
						ctx.BindReg(d53.Reg, &d54)
					}
					if d54.Loc == LocReg && d53.Loc == LocReg && d54.Reg == d53.Reg {
						ctx.TransferReg(d53.Reg)
						d53.Loc = LocNone
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d3)
					ctx.EnsureGoStringHeader(&d5)
					d55 = ctx.EmitSliceElementAddress(&d5, &d3, 1)
					ctx.EnsureDesc(&d55)
					r4 := ctx.AllocRegExcept(d55.Reg)
					ctx.EmitMovRegMemB(r4, d55.Reg, 0)
					ctx.FreeDesc(&d55)
					d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4, NoHeapPointer: true}
					ctx.BindReg(r4, &d56)
					ctx.BindReg(r4, &d56)
					ctx.EnsureDesc(&d56)
					var d57 JITValueDesc
					if d56.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d56.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d56.Reg, 4)
						d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg}
						ctx.BindReg(d56.Reg, &d57)
					}
					if d57.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: d57.Type, Imm: NewInt(int64(uint64(d57.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d57.Reg, 56)
						ctx.EmitShrRegImm8(d57.Reg, 56)
					}
					if d57.Loc == LocReg && d56.Loc == LocReg && d57.Reg == d56.Reg {
						ctx.TransferReg(d56.Reg)
						d56.Loc = LocNone
					}
					ctx.FreeDesc(&d56)
					d58 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d57)
					ctx.EnsureGoStringHeader(&d58)
					d59 = ctx.EmitSliceElementAddress(&d58, &d57, 1)
					ctx.EnsureDesc(&d59)
					r5 := ctx.AllocRegExcept(d59.Reg)
					ctx.EmitMovRegMemB(r5, d59.Reg, 0)
					ctx.FreeDesc(&d59)
					d60 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, NoHeapPointer: true}
					ctx.BindReg(r5, &d60)
					ctx.BindReg(r5, &d60)
					ctx.FreeDesc(&d57)
					ctx.EnsureDesc(&d54)
					ctx.SyncDesc(&d60)
					ctx.StabilizeDescAcrossNestedCall(&d54)
					d61 = d11
					d61.ID = 0
					d62 = d54
					d62.ID = 0
					d63 = ctx.EmitSliceElementAddress(&d61, &d62, int32(1))
					ctx.FreeDesc(&d62)
					ctx.EmitStoreScalarAt(&d63, &d60, 1)
					ctx.FreeDesc(&d63)
					ctx.FreeDesc(&d54)
					ctx.FreeDesc(&d60)
					d64 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDescsTogether(&d64, &d3)
					var d65 JITValueDesc
					if d64.Loc == LocImm && d3.Loc == LocImm {
						d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d64.Imm.Int() * d3.Imm.Int())}
					} else if d64.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
						ctx.EmitImulInt64(scratch, d3.Reg)
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d65)
					} else if d3.Loc == LocImm {
						if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d64.Reg, int32(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
							ctx.EmitImulInt64(d64.Reg, RegR11)
						}
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d64.Reg}
						ctx.BindReg(d64.Reg, &d65)
					} else {
						ctx.EmitImulInt64(d64.Reg, d3.Reg)
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d64.Reg}
						ctx.BindReg(d64.Reg, &d65)
					}
					if d65.Loc == LocReg && d64.Loc == LocReg && d65.Reg == d64.Reg {
						ctx.TransferReg(d64.Reg)
						d64.Loc = LocNone
					}
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					var d66 JITValueDesc
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegReg(scratch, d65.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d66)
					}
					if d66.Loc == LocReg && d65.Loc == LocReg && d66.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.FreeDesc(&d65)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d3)
					ctx.EnsureGoStringHeader(&d5)
					d67 = ctx.EmitSliceElementAddress(&d5, &d3, 1)
					ctx.EnsureDesc(&d67)
					r6 := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegMemB(r6, d67.Reg, 0)
					ctx.FreeDesc(&d67)
					d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d68)
					ctx.BindReg(r6, &d68)
					ctx.EnsureDesc(&d68)
					var d69 JITValueDesc
					if d68.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d68.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d68.Reg, 15)
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d68.Reg}
						ctx.BindReg(d68.Reg, &d69)
					}
					if d69.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: d69.Type, Imm: NewInt(int64(uint64(d69.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d69.Reg, 56)
						ctx.EmitShrRegImm8(d69.Reg, 56)
					}
					if d69.Loc == LocReg && d68.Loc == LocReg && d69.Reg == d68.Reg {
						ctx.TransferReg(d68.Reg)
						d68.Loc = LocNone
					}
					ctx.FreeDesc(&d68)
					d70 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d69)
					ctx.EnsureGoStringHeader(&d70)
					d71 = ctx.EmitSliceElementAddress(&d70, &d69, 1)
					ctx.EnsureDesc(&d71)
					r7 := ctx.AllocRegExcept(d71.Reg)
					ctx.EmitMovRegMemB(r7, d71.Reg, 0)
					ctx.FreeDesc(&d71)
					d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d72)
					ctx.BindReg(r7, &d72)
					ctx.FreeDesc(&d69)
					ctx.EnsureDesc(&d66)
					ctx.SyncDesc(&d72)
					ctx.StabilizeDescAcrossNestedCall(&d66)
					d73 = d11
					d73.ID = 0
					d74 = d66
					d74.ID = 0
					d75 = ctx.EmitSliceElementAddress(&d73, &d74, int32(1))
					ctx.FreeDesc(&d74)
					ctx.EmitStoreScalarAt(&d75, &d72, 1)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d66)
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d76 JITValueDesc
					if d3.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d3.Reg)
						}
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d76)
					}
					if d76.Loc == LocReg && d3.Loc == LocReg && d76.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d76)
						if d76.Loc == LocReg {
							ctx.ProtectReg(d76.Reg)
						} else if d76.Loc == LocRegPair {
							ctx.ProtectReg(d76.Reg)
							ctx.ProtectReg(d76.Reg2)
						}
						d77 = d76
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d77)
						} else {
							ctx.EmitStoreToStack(d77, int32(bbs[1].PhiBase)+int32(0))
						}
						if d76.Loc == LocReg {
							ctx.UnprotectReg(d76.Reg)
						} else if d76.Loc == LocRegPair {
							ctx.UnprotectReg(d76.Reg)
							ctx.UnprotectReg(d76.Reg2)
						}
					}
					ps78 := PhiState{General: ps.General}
					ps78.OverlayValues = make([]JITValueDesc, 78)
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[4] = d4
					ps78.OverlayValues[5] = d5
					ps78.OverlayValues[6] = d6
					ps78.OverlayValues[7] = d7
					ps78.OverlayValues[8] = d8
					ps78.OverlayValues[9] = d9
					ps78.OverlayValues[11] = d11
					ps78.OverlayValues[13] = d13
					ps78.OverlayValues[14] = d14
					ps78.OverlayValues[15] = d15
					ps78.OverlayValues[16] = d16
					ps78.OverlayValues[17] = d17
					ps78.OverlayValues[20] = d20
					ps78.OverlayValues[53] = d53
					ps78.OverlayValues[54] = d54
					ps78.OverlayValues[55] = d55
					ps78.OverlayValues[56] = d56
					ps78.OverlayValues[57] = d57
					ps78.OverlayValues[58] = d58
					ps78.OverlayValues[59] = d59
					ps78.OverlayValues[60] = d60
					ps78.OverlayValues[61] = d61
					ps78.OverlayValues[62] = d62
					ps78.OverlayValues[63] = d63
					ps78.OverlayValues[64] = d64
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
					ps78.OverlayValues[75] = d75
					ps78.OverlayValues[76] = d76
					ps78.OverlayValues[77] = d77
					ps78.PhiValues = make([]JITValueDesc, 1)
					d79 = d76
					ps78.PhiValues[0] = d79
					if ps78.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps78)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
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
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					callResults81 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d11}, []uint8{2}, []uint8{1})
					d80 = callResults81[0]
					ctx.EnsureDesc(&d80)
					d82 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d80}, 2)
					ctx.EmitMovPairToResult(&d82, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps83 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps83)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["hex2bin"]
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d28 JITValueDesc
				_ = d28
				var d30 JITValueDesc
				_ = d30
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					ctx.SyncDesc(&d2)
					if d2.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d2 = tmpScalar
					}
					d2 = JITPrepareScmerGoArg(ctx, d2)
					if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
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
					snap10 := d0
					snap11 := d1
					snap12 := d2
					snap13 := d4
					snap14 := d5
					snap15 := d6
					snap16 := d7
					alloc17 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc17)
					d0 = snap10
					d1 = snap11
					d2 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc17)
					d0 = snap10
					d1 = snap11
					d2 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 8)
					ps18.OverlayValues[0] = d0
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[6] = d6
					ps18.OverlayValues[7] = d7
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 8)
					ps19.OverlayValues[0] = d0
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[7] = d7
					snap20 := d0
					snap21 := d1
					snap22 := d2
					snap23 := d4
					snap24 := d5
					snap25 := d6
					snap26 := d7
					alloc27 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps19)
					}
					ctx.RestoreAllocState(alloc27)
					d0 = snap20
					d1 = snap21
					d2 = snap22
					d4 = snap23
					d5 = snap24
					d6 = snap25
					d7 = snap26
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps18)
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
					callResults29 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d4}, []uint8{2}, []uint8{1})
					d28 = callResults29[0]
					ctx.EnsureDesc(&d28)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d28}, 2)
					ctx.EmitMovPairToResult(&d30, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps31 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps31)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["uuid"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					snap7 := d1
					snap8 := d2
					snap9 := d3
					snap10 := d4
					alloc11 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc11)
					d1 = snap7
					d2 = snap8
					d3 = snap9
					d4 = snap10
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc11)
					d1 = snap7
					d2 = snap8
					d3 = snap9
					d4 = snap10
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 5)
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 5)
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					snap14 := d1
					snap15 := d2
					snap16 := d3
					snap17 := d4
					alloc18 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps13)
					}
					ctx.RestoreAllocState(alloc18)
					d1 = snap14
					d2 = snap15
					d3 = snap16
					d4 = snap17
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((uuid.UUID).String arg0)")
					}
					ctx.SyncDesc(&d1)
					d19 = ctx.EmitGoCallScalar(GoFuncAddr((uuid.UUID).String), []JITValueDesc{d1}, 2)
					d19.NoHeapPointer = false
					ctx.BindReg(d19.Reg, &d19)
					ctx.BindReg(d19.Reg2, &d19)
					ctx.EnsureDesc(&d19)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d19}, 2)
					ctx.EmitMovPairToResult(&d20, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				ps21 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps21)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["randomBytes"]
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d89 JITValueDesc
				_ = d89
				var d91 JITValueDesc
				_ = d91
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
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
						d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d4)
					}
					d5 = d4
					ctx.EnsureDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitJump(d5.Condition, lbl8)
					ctx.EmitJmp(lbl9)
					snap8 := d0
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					snap13 := d5
					alloc14 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc14)
					d0 = snap8
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc14)
					d0 = snap8
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					ps15 := PhiState{General: true}
					ps15.OverlayValues = make([]JITValueDesc, 6)
					ps15.OverlayValues[0] = d0
					ps15.OverlayValues[1] = d1
					ps15.OverlayValues[2] = d2
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					ps16 := PhiState{General: true}
					ps16.OverlayValues = make([]JITValueDesc, 6)
					ps16.OverlayValues[0] = d0
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					snap17 := d0
					snap18 := d1
					snap19 := d2
					snap20 := d3
					snap21 := d4
					snap22 := d5
					alloc23 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps16)
					}
					ctx.RestoreAllocState(alloc23)
					d0 = snap17
					d1 = snap18
					d2 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps15)
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					callResults24 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d2, d2}, []uint8{3}, []uint8{1})
					d25 = callResults24[0]
					d25.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.EnsureDesc(&d2)
					var d26 JITValueDesc
					if d2.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() > 0)}
					} else {
						r1 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						d26 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedGreater}
						ctx.BindReg(r1, &d26)
					}
					d27 = d26
					ctx.EnsureDesc(&d27)
					if d27.Loc != LocImm && d27.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
							ps28.OverlayValues[4] = d4
							ps28.OverlayValues[5] = d5
							ps28.OverlayValues[25] = d25
							ps28.OverlayValues[26] = d26
							ps28.OverlayValues[27] = d27
							return bbs[3].RenderPS(ps28)
						}
						if ps.General {
						}
						ps29 := PhiState{General: ps.General}
						ps29.OverlayValues = make([]JITValueDesc, 28)
						ps29.OverlayValues[0] = d0
						ps29.OverlayValues[1] = d1
						ps29.OverlayValues[2] = d2
						ps29.OverlayValues[3] = d3
						ps29.OverlayValues[4] = d4
						ps29.OverlayValues[5] = d5
						ps29.OverlayValues[25] = d25
						ps29.OverlayValues[26] = d26
						ps29.OverlayValues[27] = d27
						return bbs[4].RenderPS(ps29)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitJump(d27.Condition, lbl10)
					ctx.EmitJmp(lbl11)
					snap30 := d0
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d4
					snap35 := d5
					snap36 := d25
					snap37 := d26
					snap38 := d27
					alloc39 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc39)
					d0 = snap30
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d4 = snap34
					d5 = snap35
					d25 = snap36
					d26 = snap37
					d27 = snap38
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc39)
					d0 = snap30
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d4 = snap34
					d5 = snap35
					d25 = snap36
					d26 = snap37
					d27 = snap38
					ps40 := PhiState{General: true}
					ps40.OverlayValues = make([]JITValueDesc, 28)
					ps40.OverlayValues[0] = d0
					ps40.OverlayValues[1] = d1
					ps40.OverlayValues[2] = d2
					ps40.OverlayValues[3] = d3
					ps40.OverlayValues[4] = d4
					ps40.OverlayValues[5] = d5
					ps40.OverlayValues[25] = d25
					ps40.OverlayValues[26] = d26
					ps40.OverlayValues[27] = d27
					ps41 := PhiState{General: true}
					ps41.OverlayValues = make([]JITValueDesc, 28)
					ps41.OverlayValues[0] = d0
					ps41.OverlayValues[1] = d1
					ps41.OverlayValues[2] = d2
					ps41.OverlayValues[3] = d3
					ps41.OverlayValues[4] = d4
					ps41.OverlayValues[5] = d5
					ps41.OverlayValues[25] = d25
					ps41.OverlayValues[26] = d26
					ps41.OverlayValues[27] = d27
					snap42 := d0
					snap43 := d1
					snap44 := d2
					snap45 := d3
					snap46 := d4
					snap47 := d5
					snap48 := d25
					snap49 := d26
					snap50 := d27
					alloc51 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps41)
					}
					ctx.RestoreAllocState(alloc51)
					d0 = snap42
					d1 = snap43
					d2 = snap44
					d3 = snap45
					d4 = snap46
					d5 = snap47
					d25 = snap48
					d26 = snap49
					d27 = snap50
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps40)
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
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocRegTriple && d25.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (crand.Read arg0)")
					}
					ctx.SyncDesc(&d25)
					callResults52 := JITEmitGoCallResults(ctx, GoFuncAddr(crand.Read), []JITValueDesc{d25}, []uint8{1, 2}, []uint8{0, 3})
					d53 = callResults52[0]
					_ = d53
					d54 = callResults52[1]
					_ = d54
					ctx.StabilizeDescForControlFlow(&d54)
					ctx.EnsureDesc(&d54)
					var d55 JITValueDesc
					if d54.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d54.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d54)
						if d54.Loc != LocReg && d54.Loc != LocRegPair && d54.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d54.Reg)
						ctx.EmitCmpRegImm32(d54.Reg, 0)
						ctx.EmitSetcc(r2, CondNotEqual)
						d55 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d55)
					}
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
							ps57.OverlayValues[0] = d0
							ps57.OverlayValues[1] = d1
							ps57.OverlayValues[2] = d2
							ps57.OverlayValues[3] = d3
							ps57.OverlayValues[4] = d4
							ps57.OverlayValues[5] = d5
							ps57.OverlayValues[25] = d25
							ps57.OverlayValues[26] = d26
							ps57.OverlayValues[27] = d27
							ps57.OverlayValues[53] = d53
							ps57.OverlayValues[54] = d54
							ps57.OverlayValues[55] = d55
							ps57.OverlayValues[56] = d56
							return bbs[5].RenderPS(ps57)
						}
						if ps.General {
						}
						ps58 := PhiState{General: ps.General}
						ps58.OverlayValues = make([]JITValueDesc, 57)
						ps58.OverlayValues[0] = d0
						ps58.OverlayValues[1] = d1
						ps58.OverlayValues[2] = d2
						ps58.OverlayValues[3] = d3
						ps58.OverlayValues[4] = d4
						ps58.OverlayValues[5] = d5
						ps58.OverlayValues[25] = d25
						ps58.OverlayValues[26] = d26
						ps58.OverlayValues[27] = d27
						ps58.OverlayValues[53] = d53
						ps58.OverlayValues[54] = d54
						ps58.OverlayValues[55] = d55
						ps58.OverlayValues[56] = d56
						return bbs[4].RenderPS(ps58)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d56.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					snap59 := d0
					snap60 := d1
					snap61 := d2
					snap62 := d3
					snap63 := d4
					snap64 := d5
					snap65 := d25
					snap66 := d26
					snap67 := d27
					snap68 := d53
					snap69 := d54
					snap70 := d55
					snap71 := d56
					alloc72 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc72)
					d0 = snap59
					d1 = snap60
					d2 = snap61
					d3 = snap62
					d4 = snap63
					d5 = snap64
					d25 = snap65
					d26 = snap66
					d27 = snap67
					d53 = snap68
					d54 = snap69
					d55 = snap70
					d56 = snap71
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc72)
					d0 = snap59
					d1 = snap60
					d2 = snap61
					d3 = snap62
					d4 = snap63
					d5 = snap64
					d25 = snap65
					d26 = snap66
					d27 = snap67
					d53 = snap68
					d54 = snap69
					d55 = snap70
					d56 = snap71
					ps73 := PhiState{General: true}
					ps73.OverlayValues = make([]JITValueDesc, 57)
					ps73.OverlayValues[0] = d0
					ps73.OverlayValues[1] = d1
					ps73.OverlayValues[2] = d2
					ps73.OverlayValues[3] = d3
					ps73.OverlayValues[4] = d4
					ps73.OverlayValues[5] = d5
					ps73.OverlayValues[25] = d25
					ps73.OverlayValues[26] = d26
					ps73.OverlayValues[27] = d27
					ps73.OverlayValues[53] = d53
					ps73.OverlayValues[54] = d54
					ps73.OverlayValues[55] = d55
					ps73.OverlayValues[56] = d56
					ps74 := PhiState{General: true}
					ps74.OverlayValues = make([]JITValueDesc, 57)
					ps74.OverlayValues[0] = d0
					ps74.OverlayValues[1] = d1
					ps74.OverlayValues[2] = d2
					ps74.OverlayValues[3] = d3
					ps74.OverlayValues[4] = d4
					ps74.OverlayValues[5] = d5
					ps74.OverlayValues[25] = d25
					ps74.OverlayValues[26] = d26
					ps74.OverlayValues[27] = d27
					ps74.OverlayValues[53] = d53
					ps74.OverlayValues[54] = d54
					ps74.OverlayValues[55] = d55
					ps74.OverlayValues[56] = d56
					snap75 := d0
					snap76 := d1
					snap77 := d2
					snap78 := d3
					snap79 := d4
					snap80 := d5
					snap81 := d25
					snap82 := d26
					snap83 := d27
					snap84 := d53
					snap85 := d54
					snap86 := d55
					snap87 := d56
					alloc88 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps74)
					}
					ctx.RestoreAllocState(alloc88)
					d0 = snap75
					d1 = snap76
					d2 = snap77
					d3 = snap78
					d4 = snap79
					d5 = snap80
					d25 = snap81
					d26 = snap82
					d27 = snap83
					d53 = snap84
					d54 = snap85
					d55 = snap86
					d56 = snap87
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps73)
					}
					return result
					ctx.FreeDesc(&d55)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
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
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					callResults90 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d25}, []uint8{2}, []uint8{1})
					d89 = callResults90[0]
					ctx.EnsureDesc(&d89)
					d91 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d89}, 2)
					ctx.EmitMovPairToResult(&d91, &result)
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				ps92 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps92)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
			if scmerCallable(a[2]) {
				replacer := a[2]
				return NewString(re.ReplaceAllStringFunc(String(a[0]), func(match string) string {
					return String(Apply(replacer, NewString(match)))
				}))
			}
			return NewString(re.ReplaceAllString(String(a[0]), String(a[2])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "replaces matches of a regex pattern in a string; the replacement may be a string ($1 expansion) or a function called with each match",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "regex pattern"}, &TypeDescriptor{Kind: "any", Label: "replacement", Description: "replacement string ($1 expansion) or function (match) -> string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["regexp_replace"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
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
				declaration := declarations["fnv_hash"]
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
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (fnvHashString arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(fnvHashString), []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				declaration := declarations["stable_structural_hash"]
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
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
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
				var d86 JITValueDesc
				_ = d86
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [8]BBDescriptor
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
						d1 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
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
						return bbs[3].RenderPS(ps4)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d2.Condition, lbl9)
					ctx.EmitJmp(lbl10)
					snap5 := d0
					snap6 := d1
					snap7 := d2
					alloc8 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc8)
					d0 = snap5
					d1 = snap6
					d2 = snap7
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
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
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps10)
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
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(func() *schemeTextWriter { return new(schemeTextWriter) }), nil, 1)
					ctx.BindReg(d15.Reg, &d15)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d16)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *schemeTextWriter, value uint64) { base.hash = value }), []JITValueDesc{d15, d16})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					ctx.StabilizeDescForControlFlow(&d15)
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d17)
					var d18 JITValueDesc
					if d17.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() == 2)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d17.Reg, 2)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d18)
					}
					ctx.FreeDesc(&d17)
					d19 = d18
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
							ps20.OverlayValues[15] = d15
							ps20.OverlayValues[16] = d16
							ps20.OverlayValues[17] = d17
							ps20.OverlayValues[18] = d18
							ps20.OverlayValues[19] = d19
							return bbs[7].RenderPS(ps20)
						}
						if ps.General {
						}
						ps21 := PhiState{General: ps.General}
						ps21.OverlayValues = make([]JITValueDesc, 20)
						ps21.OverlayValues[0] = d0
						ps21.OverlayValues[1] = d1
						ps21.OverlayValues[2] = d2
						ps21.OverlayValues[15] = d15
						ps21.OverlayValues[16] = d16
						ps21.OverlayValues[17] = d17
						ps21.OverlayValues[18] = d18
						ps21.OverlayValues[19] = d19
						return bbs[6].RenderPS(ps21)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitJump(d19.Condition, lbl12)
					ctx.EmitJmp(lbl13)
					snap22 := d0
					snap23 := d1
					snap24 := d2
					snap25 := d15
					snap26 := d16
					snap27 := d17
					snap28 := d18
					snap29 := d19
					alloc30 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc30)
					d0 = snap22
					d1 = snap23
					d2 = snap24
					d15 = snap25
					d16 = snap26
					d17 = snap27
					d18 = snap28
					d19 = snap29
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc30)
					d0 = snap22
					d1 = snap23
					d2 = snap24
					d15 = snap25
					d16 = snap26
					d17 = snap27
					d18 = snap28
					d19 = snap29
					ps31 := PhiState{General: true}
					ps31.OverlayValues = make([]JITValueDesc, 20)
					ps31.OverlayValues[0] = d0
					ps31.OverlayValues[1] = d1
					ps31.OverlayValues[2] = d2
					ps31.OverlayValues[15] = d15
					ps31.OverlayValues[16] = d16
					ps31.OverlayValues[17] = d17
					ps31.OverlayValues[18] = d18
					ps31.OverlayValues[19] = d19
					ps32 := PhiState{General: true}
					ps32.OverlayValues = make([]JITValueDesc, 20)
					ps32.OverlayValues[0] = d0
					ps32.OverlayValues[1] = d1
					ps32.OverlayValues[2] = d2
					ps32.OverlayValues[15] = d15
					ps32.OverlayValues[16] = d16
					ps32.OverlayValues[17] = d17
					ps32.OverlayValues[18] = d18
					ps32.OverlayValues[19] = d19
					snap33 := d0
					snap34 := d1
					snap35 := d2
					snap36 := d15
					snap37 := d16
					snap38 := d17
					snap39 := d18
					snap40 := d19
					alloc41 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps32)
					}
					ctx.RestoreAllocState(alloc41)
					d0 = snap33
					d1 = snap34
					d2 = snap35
					d15 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps31)
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
					ctx.ReclaimUntrackedRegs()
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d42)
					var d43 JITValueDesc
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d42.Imm.Int() > 2)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d42.Reg, 2)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreater}
						ctx.BindReg(r2, &d43)
					}
					ctx.FreeDesc(&d42)
					d44 = d43
					ctx.EnsureDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							if ps.General {
							}
							ps45 := PhiState{General: ps.General}
							ps45.OverlayValues = make([]JITValueDesc, 45)
							ps45.OverlayValues[0] = d0
							ps45.OverlayValues[1] = d1
							ps45.OverlayValues[2] = d2
							ps45.OverlayValues[15] = d15
							ps45.OverlayValues[16] = d16
							ps45.OverlayValues[17] = d17
							ps45.OverlayValues[18] = d18
							ps45.OverlayValues[19] = d19
							ps45.OverlayValues[42] = d42
							ps45.OverlayValues[43] = d43
							ps45.OverlayValues[44] = d44
							return bbs[1].RenderPS(ps45)
						}
						if ps.General {
						}
						ps46 := PhiState{General: ps.General}
						ps46.OverlayValues = make([]JITValueDesc, 45)
						ps46.OverlayValues[0] = d0
						ps46.OverlayValues[1] = d1
						ps46.OverlayValues[2] = d2
						ps46.OverlayValues[15] = d15
						ps46.OverlayValues[16] = d16
						ps46.OverlayValues[17] = d17
						ps46.OverlayValues[18] = d18
						ps46.OverlayValues[19] = d19
						ps46.OverlayValues[42] = d42
						ps46.OverlayValues[43] = d43
						ps46.OverlayValues[44] = d44
						return bbs[2].RenderPS(ps46)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitJump(d44.Condition, lbl14)
					ctx.EmitJmp(lbl15)
					snap47 := d0
					snap48 := d1
					snap49 := d2
					snap50 := d15
					snap51 := d16
					snap52 := d17
					snap53 := d18
					snap54 := d19
					snap55 := d42
					snap56 := d43
					snap57 := d44
					alloc58 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc58)
					d0 = snap47
					d1 = snap48
					d2 = snap49
					d15 = snap50
					d16 = snap51
					d17 = snap52
					d18 = snap53
					d19 = snap54
					d42 = snap55
					d43 = snap56
					d44 = snap57
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc58)
					d0 = snap47
					d1 = snap48
					d2 = snap49
					d15 = snap50
					d16 = snap51
					d17 = snap52
					d18 = snap53
					d19 = snap54
					d42 = snap55
					d43 = snap56
					d44 = snap57
					ps59 := PhiState{General: true}
					ps59.OverlayValues = make([]JITValueDesc, 45)
					ps59.OverlayValues[0] = d0
					ps59.OverlayValues[1] = d1
					ps59.OverlayValues[2] = d2
					ps59.OverlayValues[15] = d15
					ps59.OverlayValues[16] = d16
					ps59.OverlayValues[17] = d17
					ps59.OverlayValues[18] = d18
					ps59.OverlayValues[19] = d19
					ps59.OverlayValues[42] = d42
					ps59.OverlayValues[43] = d43
					ps59.OverlayValues[44] = d44
					ps60 := PhiState{General: true}
					ps60.OverlayValues = make([]JITValueDesc, 45)
					ps60.OverlayValues[0] = d0
					ps60.OverlayValues[1] = d1
					ps60.OverlayValues[2] = d2
					ps60.OverlayValues[15] = d15
					ps60.OverlayValues[16] = d16
					ps60.OverlayValues[17] = d17
					ps60.OverlayValues[18] = d18
					ps60.OverlayValues[19] = d19
					ps60.OverlayValues[42] = d42
					ps60.OverlayValues[43] = d43
					ps60.OverlayValues[44] = d44
					snap61 := d0
					snap62 := d1
					snap63 := d2
					snap64 := d15
					snap65 := d16
					snap66 := d17
					snap67 := d18
					snap68 := d19
					snap69 := d42
					snap70 := d43
					snap71 := d44
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps60)
					}
					ctx.RestoreAllocState(alloc72)
					d0 = snap61
					d1 = snap62
					d2 = snap63
					d15 = snap64
					d16 = snap65
					d17 = snap66
					d18 = snap67
					d19 = snap68
					d42 = snap69
					d43 = snap70
					d44 = snap71
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps59)
					}
					return result
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d15)
					d73 = args[0]
					d73.ID = 0
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					d73 = JITPrepareScmerGoArg(ctx, d73)
					d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d74.Loc == LocRegPair || d74.Loc == LocStackPair || d74.Loc == LocRegTriple || d74.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d75.Loc == LocRegPair || d75.Loc == LocStackPair || d75.Loc == LocRegTriple || d75.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d76 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					if d76.Loc == LocRegPair || d76.Loc == LocStackPair || d76.Loc == LocRegTriple || d76.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d73)
					ctx.SyncDesc(&d74)
					ctx.SyncDesc(&d75)
					ctx.SyncDesc(&d76)
					ctx.EmitGoCallVoid(GoFuncAddr(serializeEx), []JITValueDesc{d15, d73, d74, d75, d76})
					ctx.FreeDesc(&d76)
					ctx.FreeDesc(&d73)
					if ps.General {
					}
					ps77 := PhiState{General: ps.General}
					ps77.OverlayValues = make([]JITValueDesc, 77)
					ps77.OverlayValues[0] = d0
					ps77.OverlayValues[1] = d1
					ps77.OverlayValues[2] = d2
					ps77.OverlayValues[15] = d15
					ps77.OverlayValues[16] = d16
					ps77.OverlayValues[17] = d17
					ps77.OverlayValues[18] = d18
					ps77.OverlayValues[19] = d19
					ps77.OverlayValues[42] = d42
					ps77.OverlayValues[43] = d43
					ps77.OverlayValues[44] = d44
					ps77.OverlayValues[73] = d73
					ps77.OverlayValues[74] = d74
					ps77.OverlayValues[75] = d75
					ps77.OverlayValues[76] = d76
					if ps77.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps77)
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d15)
					var d78 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						fieldAddr := uintptr(d15.Imm.Int()) + 8
						r3 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r3, fieldAddr)
						d78 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d78)
					} else {
						off := int32(8)
						baseReg := d15.Reg
						r4 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r4, baseReg, off)
						d78 = JITValueDesc{Loc: LocReg, Reg: r4}
						ctx.BindReg(r4, &d78)
					}
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d78)
					if d78.Loc == LocRegPair || d78.Loc == LocStackPair || d78.Loc == LocRegTriple || d78.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d78)
					d79 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d78}, 2)
					d79.NoHeapPointer = false
					ctx.BindReg(d79.Reg, &d79)
					ctx.BindReg(d79.Reg2, &d79)
					ctx.FreeDesc(&d78)
					ctx.EnsureDesc(&d79)
					d80 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d79}, 2)
					ctx.EmitMovPairToResult(&d80, &result)
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d15)
					d81 = args[0]
					d81.ID = 0
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d81)
					d81 = JITPrepareScmerGoArg(ctx, d81)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d81)
					ctx.EmitGoCallVoid(GoFuncAddr(WriteStringValue), []JITValueDesc{d15, d81})
					ctx.FreeDesc(&d81)
					if ps.General {
					}
					ps82 := PhiState{General: ps.General}
					ps82.OverlayValues = make([]JITValueDesc, 82)
					ps82.OverlayValues[0] = d0
					ps82.OverlayValues[1] = d1
					ps82.OverlayValues[2] = d2
					ps82.OverlayValues[15] = d15
					ps82.OverlayValues[16] = d16
					ps82.OverlayValues[17] = d17
					ps82.OverlayValues[18] = d18
					ps82.OverlayValues[19] = d19
					ps82.OverlayValues[42] = d42
					ps82.OverlayValues[43] = d43
					ps82.OverlayValues[44] = d44
					ps82.OverlayValues[73] = d73
					ps82.OverlayValues[74] = d74
					ps82.OverlayValues[75] = d75
					ps82.OverlayValues[76] = d76
					ps82.OverlayValues[78] = d78
					ps82.OverlayValues[79] = d79
					ps82.OverlayValues[80] = d80
					ps82.OverlayValues[81] = d81
					if ps82.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps82)
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					ctx.ReclaimUntrackedRegs()
					d83 = args[1]
					d83.ID = 0
					d85 = d83
					d85.ID = 0
					d84 = ctx.EmitBoolDesc(&d85, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d83)
					d86 = d84
					ctx.EnsureDesc(&d86)
					if d86.Loc != LocImm && d86.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d86.Loc == LocImm {
						if d86.Imm.Bool() {
							if ps.General {
							}
							ps87 := PhiState{General: ps.General}
							ps87.OverlayValues = make([]JITValueDesc, 87)
							ps87.OverlayValues[0] = d0
							ps87.OverlayValues[1] = d1
							ps87.OverlayValues[2] = d2
							ps87.OverlayValues[15] = d15
							ps87.OverlayValues[16] = d16
							ps87.OverlayValues[17] = d17
							ps87.OverlayValues[18] = d18
							ps87.OverlayValues[19] = d19
							ps87.OverlayValues[42] = d42
							ps87.OverlayValues[43] = d43
							ps87.OverlayValues[44] = d44
							ps87.OverlayValues[73] = d73
							ps87.OverlayValues[74] = d74
							ps87.OverlayValues[75] = d75
							ps87.OverlayValues[76] = d76
							ps87.OverlayValues[78] = d78
							ps87.OverlayValues[79] = d79
							ps87.OverlayValues[80] = d80
							ps87.OverlayValues[81] = d81
							ps87.OverlayValues[83] = d83
							ps87.OverlayValues[84] = d84
							ps87.OverlayValues[85] = d85
							ps87.OverlayValues[86] = d86
							return bbs[4].RenderPS(ps87)
						}
						if ps.General {
						}
						ps88 := PhiState{General: ps.General}
						ps88.OverlayValues = make([]JITValueDesc, 87)
						ps88.OverlayValues[0] = d0
						ps88.OverlayValues[1] = d1
						ps88.OverlayValues[2] = d2
						ps88.OverlayValues[15] = d15
						ps88.OverlayValues[16] = d16
						ps88.OverlayValues[17] = d17
						ps88.OverlayValues[18] = d18
						ps88.OverlayValues[19] = d19
						ps88.OverlayValues[42] = d42
						ps88.OverlayValues[43] = d43
						ps88.OverlayValues[44] = d44
						ps88.OverlayValues[73] = d73
						ps88.OverlayValues[74] = d74
						ps88.OverlayValues[75] = d75
						ps88.OverlayValues[76] = d76
						ps88.OverlayValues[78] = d78
						ps88.OverlayValues[79] = d79
						ps88.OverlayValues[80] = d80
						ps88.OverlayValues[81] = d81
						ps88.OverlayValues[83] = d83
						ps88.OverlayValues[84] = d84
						ps88.OverlayValues[85] = d85
						ps88.OverlayValues[86] = d86
						return bbs[6].RenderPS(ps88)
					}
					if !ps.General {
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d86.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					snap89 := d0
					snap90 := d1
					snap91 := d2
					snap92 := d15
					snap93 := d16
					snap94 := d17
					snap95 := d18
					snap96 := d19
					snap97 := d42
					snap98 := d43
					snap99 := d44
					snap100 := d73
					snap101 := d74
					snap102 := d75
					snap103 := d76
					snap104 := d78
					snap105 := d79
					snap106 := d80
					snap107 := d81
					snap108 := d83
					snap109 := d84
					snap110 := d85
					snap111 := d86
					alloc112 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc112)
					d0 = snap89
					d1 = snap90
					d2 = snap91
					d15 = snap92
					d16 = snap93
					d17 = snap94
					d18 = snap95
					d19 = snap96
					d42 = snap97
					d43 = snap98
					d44 = snap99
					d73 = snap100
					d74 = snap101
					d75 = snap102
					d76 = snap103
					d78 = snap104
					d79 = snap105
					d80 = snap106
					d81 = snap107
					d83 = snap108
					d84 = snap109
					d85 = snap110
					d86 = snap111
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc112)
					d0 = snap89
					d1 = snap90
					d2 = snap91
					d15 = snap92
					d16 = snap93
					d17 = snap94
					d18 = snap95
					d19 = snap96
					d42 = snap97
					d43 = snap98
					d44 = snap99
					d73 = snap100
					d74 = snap101
					d75 = snap102
					d76 = snap103
					d78 = snap104
					d79 = snap105
					d80 = snap106
					d81 = snap107
					d83 = snap108
					d84 = snap109
					d85 = snap110
					d86 = snap111
					ps113 := PhiState{General: true}
					ps113.OverlayValues = make([]JITValueDesc, 87)
					ps113.OverlayValues[0] = d0
					ps113.OverlayValues[1] = d1
					ps113.OverlayValues[2] = d2
					ps113.OverlayValues[15] = d15
					ps113.OverlayValues[16] = d16
					ps113.OverlayValues[17] = d17
					ps113.OverlayValues[18] = d18
					ps113.OverlayValues[19] = d19
					ps113.OverlayValues[42] = d42
					ps113.OverlayValues[43] = d43
					ps113.OverlayValues[44] = d44
					ps113.OverlayValues[73] = d73
					ps113.OverlayValues[74] = d74
					ps113.OverlayValues[75] = d75
					ps113.OverlayValues[76] = d76
					ps113.OverlayValues[78] = d78
					ps113.OverlayValues[79] = d79
					ps113.OverlayValues[80] = d80
					ps113.OverlayValues[81] = d81
					ps113.OverlayValues[83] = d83
					ps113.OverlayValues[84] = d84
					ps113.OverlayValues[85] = d85
					ps113.OverlayValues[86] = d86
					ps114 := PhiState{General: true}
					ps114.OverlayValues = make([]JITValueDesc, 87)
					ps114.OverlayValues[0] = d0
					ps114.OverlayValues[1] = d1
					ps114.OverlayValues[2] = d2
					ps114.OverlayValues[15] = d15
					ps114.OverlayValues[16] = d16
					ps114.OverlayValues[17] = d17
					ps114.OverlayValues[18] = d18
					ps114.OverlayValues[19] = d19
					ps114.OverlayValues[42] = d42
					ps114.OverlayValues[43] = d43
					ps114.OverlayValues[44] = d44
					ps114.OverlayValues[73] = d73
					ps114.OverlayValues[74] = d74
					ps114.OverlayValues[75] = d75
					ps114.OverlayValues[76] = d76
					ps114.OverlayValues[78] = d78
					ps114.OverlayValues[79] = d79
					ps114.OverlayValues[80] = d80
					ps114.OverlayValues[81] = d81
					ps114.OverlayValues[83] = d83
					ps114.OverlayValues[84] = d84
					ps114.OverlayValues[85] = d85
					ps114.OverlayValues[86] = d86
					snap115 := d0
					snap116 := d1
					snap117 := d2
					snap118 := d15
					snap119 := d16
					snap120 := d17
					snap121 := d18
					snap122 := d19
					snap123 := d42
					snap124 := d43
					snap125 := d44
					snap126 := d73
					snap127 := d74
					snap128 := d75
					snap129 := d76
					snap130 := d78
					snap131 := d79
					snap132 := d80
					snap133 := d81
					snap134 := d83
					snap135 := d84
					snap136 := d85
					snap137 := d86
					alloc138 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps114)
					}
					ctx.RestoreAllocState(alloc138)
					d0 = snap115
					d1 = snap116
					d2 = snap117
					d15 = snap118
					d16 = snap119
					d17 = snap120
					d18 = snap121
					d19 = snap122
					d42 = snap123
					d43 = snap124
					d44 = snap125
					d73 = snap126
					d74 = snap127
					d75 = snap128
					d76 = snap129
					d78 = snap130
					d79 = snap131
					d80 = snap132
					d81 = snap133
					d83 = snap134
					d84 = snap135
					d85 = snap136
					d86 = snap137
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps113)
					}
					return result
					ctx.FreeDesc(&d84)
					return result
				}
				ps139 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps139)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["sha1"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *[20]byte { return new([20]byte) }), nil, 1)
				d1 := args[0]
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
				d6.NoHeapPointer = true
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
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
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
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
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
				ctx.ReclaimUntrackedRegs()
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
				d15.NoHeapPointer = true
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
				declaration := declarations["sha256"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *[32]byte { return new([32]byte) }), nil, 1)
				d1 := args[0]
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
				d6.NoHeapPointer = false
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
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
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
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
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
				ctx.ReclaimUntrackedRegs()
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
				d15.NoHeapPointer = true
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
				declaration := declarations["regexp_test"]
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
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					snap6 := d0
					snap7 := d1
					snap8 := d2
					snap9 := d3
					alloc10 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc10)
					d0 = snap6
					d1 = snap7
					d2 = snap8
					d3 = snap9
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
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
						panic("jit: generic call arg expects 2-word value (regexp.Compile arg0)")
					}
					ctx.SyncDesc(&d20)
					callResults22 := JITEmitGoCallResults(ctx, GoFuncAddr(regexp.Compile), []JITValueDesc{d20}, []uint8{1, 2}, []uint8{1, 3})
					d23 = callResults22[0]
					_ = d23
					d24 = callResults22[1]
					_ = d24
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.StabilizeDescForControlFlow(&d24)
					ctx.EnsureDesc(&d24)
					var d25 JITValueDesc
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d24.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d24)
						if d24.Loc != LocReg && d24.Loc != LocRegPair && d24.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitCmpRegImm32(d24.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
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
							if ps.General {
							}
							ps27 := PhiState{General: ps.General}
							ps27.OverlayValues = make([]JITValueDesc, 27)
							ps27.OverlayValues[0] = d0
							ps27.OverlayValues[1] = d1
							ps27.OverlayValues[2] = d2
							ps27.OverlayValues[3] = d3
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
						ps28.OverlayValues[0] = d0
						ps28.OverlayValues[1] = d1
						ps28.OverlayValues[2] = d2
						ps28.OverlayValues[3] = d3
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
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					snap29 := d0
					snap30 := d1
					snap31 := d2
					snap32 := d3
					snap33 := d18
					snap34 := d19
					snap35 := d20
					snap36 := d21
					snap37 := d23
					snap38 := d24
					snap39 := d25
					snap40 := d26
					alloc41 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc41)
					d0 = snap29
					d1 = snap30
					d2 = snap31
					d3 = snap32
					d18 = snap33
					d19 = snap34
					d20 = snap35
					d21 = snap36
					d23 = snap37
					d24 = snap38
					d25 = snap39
					d26 = snap40
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc41)
					d0 = snap29
					d1 = snap30
					d2 = snap31
					d3 = snap32
					d18 = snap33
					d19 = snap34
					d20 = snap35
					d21 = snap36
					d23 = snap37
					d24 = snap38
					d25 = snap39
					d26 = snap40
					ps42 := PhiState{General: true}
					ps42.OverlayValues = make([]JITValueDesc, 27)
					ps42.OverlayValues[0] = d0
					ps42.OverlayValues[1] = d1
					ps42.OverlayValues[2] = d2
					ps42.OverlayValues[3] = d3
					ps42.OverlayValues[18] = d18
					ps42.OverlayValues[19] = d19
					ps42.OverlayValues[20] = d20
					ps42.OverlayValues[21] = d21
					ps42.OverlayValues[23] = d23
					ps42.OverlayValues[24] = d24
					ps42.OverlayValues[25] = d25
					ps42.OverlayValues[26] = d26
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 27)
					ps43.OverlayValues[0] = d0
					ps43.OverlayValues[1] = d1
					ps43.OverlayValues[2] = d2
					ps43.OverlayValues[3] = d3
					ps43.OverlayValues[18] = d18
					ps43.OverlayValues[19] = d19
					ps43.OverlayValues[20] = d20
					ps43.OverlayValues[21] = d21
					ps43.OverlayValues[23] = d23
					ps43.OverlayValues[24] = d24
					ps43.OverlayValues[25] = d25
					ps43.OverlayValues[26] = d26
					snap44 := d0
					snap45 := d1
					snap46 := d2
					snap47 := d3
					snap48 := d18
					snap49 := d19
					snap50 := d20
					snap51 := d21
					snap52 := d23
					snap53 := d24
					snap54 := d25
					snap55 := d26
					alloc56 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps43)
					}
					ctx.RestoreAllocState(alloc56)
					d0 = snap44
					d1 = snap45
					d2 = snap46
					d3 = snap47
					d18 = snap48
					d19 = snap49
					d20 = snap50
					d21 = snap51
					d23 = snap52
					d24 = snap53
					d25 = snap54
					d26 = snap55
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps42)
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
							if ps.General {
							}
							ps61 := PhiState{General: ps.General}
							ps61.OverlayValues = make([]JITValueDesc, 61)
							ps61.OverlayValues[0] = d0
							ps61.OverlayValues[1] = d1
							ps61.OverlayValues[2] = d2
							ps61.OverlayValues[3] = d3
							ps61.OverlayValues[18] = d18
							ps61.OverlayValues[19] = d19
							ps61.OverlayValues[20] = d20
							ps61.OverlayValues[21] = d21
							ps61.OverlayValues[23] = d23
							ps61.OverlayValues[24] = d24
							ps61.OverlayValues[25] = d25
							ps61.OverlayValues[26] = d26
							ps61.OverlayValues[57] = d57
							ps61.OverlayValues[58] = d58
							ps61.OverlayValues[59] = d59
							ps61.OverlayValues[60] = d60
							return bbs[1].RenderPS(ps61)
						}
						if ps.General {
						}
						ps62 := PhiState{General: ps.General}
						ps62.OverlayValues = make([]JITValueDesc, 61)
						ps62.OverlayValues[0] = d0
						ps62.OverlayValues[1] = d1
						ps62.OverlayValues[2] = d2
						ps62.OverlayValues[3] = d3
						ps62.OverlayValues[18] = d18
						ps62.OverlayValues[19] = d19
						ps62.OverlayValues[20] = d20
						ps62.OverlayValues[21] = d21
						ps62.OverlayValues[23] = d23
						ps62.OverlayValues[24] = d24
						ps62.OverlayValues[25] = d25
						ps62.OverlayValues[26] = d26
						ps62.OverlayValues[57] = d57
						ps62.OverlayValues[58] = d58
						ps62.OverlayValues[59] = d59
						ps62.OverlayValues[60] = d60
						return bbs[2].RenderPS(ps62)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d60.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					snap63 := d0
					snap64 := d1
					snap65 := d2
					snap66 := d3
					snap67 := d18
					snap68 := d19
					snap69 := d20
					snap70 := d21
					snap71 := d23
					snap72 := d24
					snap73 := d25
					snap74 := d26
					snap75 := d57
					snap76 := d58
					snap77 := d59
					snap78 := d60
					alloc79 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc79)
					d0 = snap63
					d1 = snap64
					d2 = snap65
					d3 = snap66
					d18 = snap67
					d19 = snap68
					d20 = snap69
					d21 = snap70
					d23 = snap71
					d24 = snap72
					d25 = snap73
					d26 = snap74
					d57 = snap75
					d58 = snap76
					d59 = snap77
					d60 = snap78
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc79)
					d0 = snap63
					d1 = snap64
					d2 = snap65
					d3 = snap66
					d18 = snap67
					d19 = snap68
					d20 = snap69
					d21 = snap70
					d23 = snap71
					d24 = snap72
					d25 = snap73
					d26 = snap74
					d57 = snap75
					d58 = snap76
					d59 = snap77
					d60 = snap78
					ps80 := PhiState{General: true}
					ps80.OverlayValues = make([]JITValueDesc, 61)
					ps80.OverlayValues[0] = d0
					ps80.OverlayValues[1] = d1
					ps80.OverlayValues[2] = d2
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[18] = d18
					ps80.OverlayValues[19] = d19
					ps80.OverlayValues[20] = d20
					ps80.OverlayValues[21] = d21
					ps80.OverlayValues[23] = d23
					ps80.OverlayValues[24] = d24
					ps80.OverlayValues[25] = d25
					ps80.OverlayValues[26] = d26
					ps80.OverlayValues[57] = d57
					ps80.OverlayValues[58] = d58
					ps80.OverlayValues[59] = d59
					ps80.OverlayValues[60] = d60
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 61)
					ps81.OverlayValues[0] = d0
					ps81.OverlayValues[1] = d1
					ps81.OverlayValues[2] = d2
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[18] = d18
					ps81.OverlayValues[19] = d19
					ps81.OverlayValues[20] = d20
					ps81.OverlayValues[21] = d21
					ps81.OverlayValues[23] = d23
					ps81.OverlayValues[24] = d24
					ps81.OverlayValues[25] = d25
					ps81.OverlayValues[26] = d26
					ps81.OverlayValues[57] = d57
					ps81.OverlayValues[58] = d58
					ps81.OverlayValues[59] = d59
					ps81.OverlayValues[60] = d60
					snap82 := d0
					snap83 := d1
					snap84 := d2
					snap85 := d3
					snap86 := d18
					snap87 := d19
					snap88 := d20
					snap89 := d21
					snap90 := d23
					snap91 := d24
					snap92 := d25
					snap93 := d26
					snap94 := d57
					snap95 := d58
					snap96 := d59
					snap97 := d60
					alloc98 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps81)
					}
					ctx.RestoreAllocState(alloc98)
					d0 = snap82
					d1 = snap83
					d2 = snap84
					d3 = snap85
					d18 = snap86
					d19 = snap87
					d20 = snap88
					d21 = snap89
					d23 = snap90
					d24 = snap91
					d25 = snap92
					d26 = snap93
					d57 = snap94
					d58 = snap95
					d59 = snap96
					d60 = snap97
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps80)
					}
					return result
					ctx.FreeDesc(&d58)
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
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
					ctx.ReclaimUntrackedRegs()
					d99 = args[0]
					d99.ID = 0
					d101 = d99
					ctx.SyncDesc(&d101)
					if d101.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d101.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d101.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d101 = tmpScalar
					}
					d101 = JITPrepareScmerGoArg(ctx, d101)
					if d101.Loc != LocRegPair && d101.Loc != LocStackPair && d101.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d100 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d101}, 2)
					ctx.FreeDesc(&d99)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d100)
					if d100.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d100.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d100.Imm)
						ptrWord, _ := d100.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d100.Imm.String())))
						d100 = tmpPair
					} else if d100.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d100.Type, Reg: ctx.AllocRegExcept(d100.Reg), Reg2: ctx.AllocRegExcept(d100.Reg)}
						switch d100.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d100)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d100)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d100)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d100)
						d100 = tmpPair
					}
					if d100.Loc != LocRegPair && d100.Loc != LocStackPair && d100.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).MatchString arg1)")
					}
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d100)
					d102 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).MatchString), []JITValueDesc{d23, d100}, 1)
					d102.NoHeapPointer = true
					ctx.EmitAndRegImm32(d102.Reg, 1)
					d102.Type = tagBool
					ctx.BindReg(d102.Reg, &d102)
					ctx.EnsureDesc(&d102)
					if d102.Loc == LocImm {
						ctx.EmitMakeBool(result, d102)
					} else {
						ctx.EmitMovToReg(result.Reg2, d102)
						d103 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d103)
						if d102.Loc == LocReg && d102.Reg != result.Reg2 {
							ctx.FreeReg(d102.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				ps104 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps104)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      28,
		},
		Optimize: optimizeRegexpTest,
	})
	registerJITRegexBuiltins()

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
	// A function replacement over a constant pattern is lowered to a declared
	// identity the JIT emits as an inline byte walk (jit-constant-regexp-replace-func),
	// mirroring optimizeRegexpTest -> jit-constant-regexp-test. Resolve a bare
	// symbol to the callable it names so the emitter sees the lambda body.
	replacement := rv[3]
	if sym, ok := scmerSymbol(replacement.WithoutSourceInfo()); ok && oc != nil && oc.Env != nil {
		if binding := oc.Env.FindRead(sym); binding != nil {
			if bound, exists := binding.Vars[sym]; exists {
				replacement = bound
			}
		}
	}
	if scmerCallable(replacement.WithoutSourceInfo()) {
		return NewSlice([]Scmer{
			NewSymbol(jitConstantRegexpReplaceFuncName),
			NewRegex(re),
			replacement,
			rv[1],
		}), td
	}
	// Replace call with a precompiled closure. The replacement stays a runtime
	// argument (arg 1 after the rewrite) so a string ($1 expansion) and a
	// function (match) -> string are both still accepted.
	compiled := NewFunc(func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return NewNil()
		}
		if scmerCallable(a[1]) {
			replacer := a[1]
			return NewString(re.ReplaceAllStringFunc(String(a[0]), func(match string) string {
				return String(Apply(replacer, NewString(match)))
			}))
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
	// Keep a declared callable identity in the optimized AST so both Eval and
	// the JIT can execute the same precompiled-regex operation directly.
	return NewSlice([]Scmer{
		NewSymbol(jitConstantRegexpTestName),
		NewRegex(re),
		rv[1],
	}), td
}
