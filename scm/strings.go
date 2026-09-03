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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["string?"].Fn, args, result)
				}
				declaration := declarations["string?"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: interface type assertion.
				ctx.Coverage.NativeCalls++
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["concat"].Fn, args, result)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_concat"].Fn, args, result)
				}
				declaration := declarations["sql_concat"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["substr"].Fn, args, result)
				}
				declaration := declarations["substr"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl6)
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
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
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
					ctx.EnsureDescsTogether(&d5, &d27)
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
					r4 := ctx.EmitSliceDataAfterLow(&d1, &d5, 1)
					d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
					ctx.BindReg(r4, &d32)
					ctx.BindReg(r4, &d32)
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
					r8 := ctx.EmitSliceDataAfterLow(&d1, &d5, 1)
					d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d38)
					ctx.BindReg(r8, &d38)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_substr"].Fn, args, result)
				}
				declaration := declarations["sql_substr"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d31 JITValueDesc
				_ = d31
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d37 JITValueDesc
				_ = d37
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d68 JITValueDesc
				_ = d68
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
				var d149 JITValueDesc
				_ = d149
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d155 JITValueDesc
				_ = d155
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
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
				var d210 JITValueDesc
				_ = d210
				var d213 JITValueDesc
				_ = d213
				var d271 JITValueDesc
				_ = d271
				var d272 JITValueDesc
				_ = d272
				var d273 JITValueDesc
				_ = d273
				var d274 JITValueDesc
				_ = d274
				var d275 JITValueDesc
				_ = d275
				var d276 JITValueDesc
				_ = d276
				var d277 JITValueDesc
				_ = d277
				var d278 JITValueDesc
				_ = d278
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
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
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
							ctx.SyncDesc(&d27)
							if d27.Loc == LocReg {
								ctx.ProtectReg(d27.Reg)
							} else if d27.Loc == LocRegPair {
								ctx.ProtectReg(d27.Reg)
								ctx.ProtectReg(d27.Reg2)
							}
							d31 = d27
							if d31.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d31)
							ctx.EmitStoreToStack(d31, int32(bbs[4].PhiBase)+int32(0))
							if d27.Loc == LocReg {
								ctx.UnprotectReg(d27.Reg)
							} else if d27.Loc == LocRegPair {
								ctx.UnprotectReg(d27.Reg)
								ctx.UnprotectReg(d27.Reg2)
							}
						}
						ps32 := PhiState{General: ps.General}
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
						ps32.OverlayValues[27] = d27
						ps32.OverlayValues[28] = d28
						ps32.OverlayValues[29] = d29
						ps32.OverlayValues[31] = d31
						ps32.PhiValues = make([]JITValueDesc, 1)
						d33 = d27
						ps32.PhiValues[0] = d33
						return bbs[4].RenderPS(ps32)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d29.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl18)
					ctx.SyncDesc(&d27)
					if d27.Loc == LocReg {
						ctx.ProtectReg(d27.Reg)
					} else if d27.Loc == LocRegPair {
						ctx.ProtectReg(d27.Reg)
						ctx.ProtectReg(d27.Reg2)
					}
					d34 = d27
					if d34.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d34)
					ctx.EmitStoreToStack(d34, int32(bbs[4].PhiBase)+int32(0))
					if d27.Loc == LocReg {
						ctx.UnprotectReg(d27.Reg)
					} else if d27.Loc == LocRegPair {
						ctx.UnprotectReg(d27.Reg)
						ctx.UnprotectReg(d27.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ps35 := PhiState{General: true}
					ps35.OverlayValues = make([]JITValueDesc, 35)
					ps35.OverlayValues[1] = d1
					ps35.OverlayValues[2] = d2
					ps35.OverlayValues[3] = d3
					ps35.OverlayValues[4] = d4
					ps35.OverlayValues[5] = d5
					ps35.OverlayValues[6] = d6
					ps35.OverlayValues[18] = d18
					ps35.OverlayValues[19] = d19
					ps35.OverlayValues[20] = d20
					ps35.OverlayValues[21] = d21
					ps35.OverlayValues[22] = d22
					ps35.OverlayValues[23] = d23
					ps35.OverlayValues[24] = d24
					ps35.OverlayValues[25] = d25
					ps35.OverlayValues[26] = d26
					ps35.OverlayValues[27] = d27
					ps35.OverlayValues[28] = d28
					ps35.OverlayValues[29] = d29
					ps35.OverlayValues[31] = d31
					ps35.OverlayValues[33] = d33
					ps35.OverlayValues[34] = d34
					ps36 := PhiState{General: true}
					ps36.OverlayValues = make([]JITValueDesc, 35)
					ps36.OverlayValues[1] = d1
					ps36.OverlayValues[2] = d2
					ps36.OverlayValues[3] = d3
					ps36.OverlayValues[4] = d4
					ps36.OverlayValues[5] = d5
					ps36.OverlayValues[6] = d6
					ps36.OverlayValues[18] = d18
					ps36.OverlayValues[19] = d19
					ps36.OverlayValues[20] = d20
					ps36.OverlayValues[21] = d21
					ps36.OverlayValues[22] = d22
					ps36.OverlayValues[23] = d23
					ps36.OverlayValues[24] = d24
					ps36.OverlayValues[25] = d25
					ps36.OverlayValues[26] = d26
					ps36.OverlayValues[27] = d27
					ps36.OverlayValues[28] = d28
					ps36.OverlayValues[29] = d29
					ps36.OverlayValues[31] = d31
					ps36.OverlayValues[33] = d33
					ps36.OverlayValues[34] = d34
					ps36.PhiValues = make([]JITValueDesc, 1)
					d37 = d27
					ps36.PhiValues[0] = d37
					snap38 := d1
					snap39 := d2
					snap40 := d3
					snap41 := d4
					snap42 := d5
					snap43 := d6
					snap44 := d18
					snap45 := d19
					snap46 := d20
					snap47 := d21
					snap48 := d22
					snap49 := d23
					snap50 := d24
					snap51 := d25
					snap52 := d26
					snap53 := d27
					snap54 := d28
					snap55 := d29
					snap56 := d31
					snap57 := d33
					snap58 := d34
					snap59 := d37
					alloc60 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps36)
					}
					ctx.RestoreAllocState(alloc60)
					d1 = snap38
					d2 = snap39
					d3 = snap40
					d4 = snap41
					d5 = snap42
					d6 = snap43
					d18 = snap44
					d19 = snap45
					d20 = snap46
					d21 = snap47
					d22 = snap48
					d23 = snap49
					d24 = snap50
					d25 = snap51
					d26 = snap52
					d27 = snap53
					d28 = snap54
					d29 = snap55
					d31 = snap56
					d33 = snap57
					d34 = snap58
					d37 = snap59
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps35)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					}
					ps61 := PhiState{General: ps.General}
					ps61.OverlayValues = make([]JITValueDesc, 38)
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
					ps61.OverlayValues[31] = d31
					ps61.OverlayValues[33] = d33
					ps61.OverlayValues[34] = d34
					ps61.OverlayValues[37] = d37
					ps61.PhiValues = make([]JITValueDesc, 1)
					d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps61.PhiValues[0] = d62
					if ps61.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps61)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d63 := ps.PhiValues[0]
							ctx.EnsureDesc(&d63)
							ctx.EmitStoreToStack(d63, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDescsTogether(&d1, &d22)
					var d64 JITValueDesc
					if d1.Loc == LocImm && d22.Loc == LocImm {
						d64 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() >= d22.Imm.Int())}
					} else if d22.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d1.Reg)
						if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d22.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedGreaterOrEqual)
						d64 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d64)
					} else if d1.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d22.Reg)
						ctx.EmitSetcc(r2, CondSignedGreaterOrEqual)
						d64 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d64)
					} else {
						r3 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d22.Reg)
						ctx.EmitSetcc(r3, CondSignedGreaterOrEqual)
						d64 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d64)
					}
					d65 = d64
					ctx.EnsureDesc(&d65)
					if d65.Loc != LocImm && d65.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d65.Loc == LocImm {
						if d65.Imm.Bool() {
							if ps.General {
							}
							ps66 := PhiState{General: ps.General}
							ps66.OverlayValues = make([]JITValueDesc, 66)
							ps66.OverlayValues[1] = d1
							ps66.OverlayValues[2] = d2
							ps66.OverlayValues[3] = d3
							ps66.OverlayValues[4] = d4
							ps66.OverlayValues[5] = d5
							ps66.OverlayValues[6] = d6
							ps66.OverlayValues[18] = d18
							ps66.OverlayValues[19] = d19
							ps66.OverlayValues[20] = d20
							ps66.OverlayValues[21] = d21
							ps66.OverlayValues[22] = d22
							ps66.OverlayValues[23] = d23
							ps66.OverlayValues[24] = d24
							ps66.OverlayValues[25] = d25
							ps66.OverlayValues[26] = d26
							ps66.OverlayValues[27] = d27
							ps66.OverlayValues[28] = d28
							ps66.OverlayValues[29] = d29
							ps66.OverlayValues[31] = d31
							ps66.OverlayValues[33] = d33
							ps66.OverlayValues[34] = d34
							ps66.OverlayValues[37] = d37
							ps66.OverlayValues[62] = d62
							ps66.OverlayValues[63] = d63
							ps66.OverlayValues[64] = d64
							ps66.OverlayValues[65] = d65
							return bbs[5].RenderPS(ps66)
						}
						if ps.General {
						}
						ps67 := PhiState{General: ps.General}
						ps67.OverlayValues = make([]JITValueDesc, 66)
						ps67.OverlayValues[1] = d1
						ps67.OverlayValues[2] = d2
						ps67.OverlayValues[3] = d3
						ps67.OverlayValues[4] = d4
						ps67.OverlayValues[5] = d5
						ps67.OverlayValues[6] = d6
						ps67.OverlayValues[18] = d18
						ps67.OverlayValues[19] = d19
						ps67.OverlayValues[20] = d20
						ps67.OverlayValues[21] = d21
						ps67.OverlayValues[22] = d22
						ps67.OverlayValues[23] = d23
						ps67.OverlayValues[24] = d24
						ps67.OverlayValues[25] = d25
						ps67.OverlayValues[26] = d26
						ps67.OverlayValues[27] = d27
						ps67.OverlayValues[28] = d28
						ps67.OverlayValues[29] = d29
						ps67.OverlayValues[31] = d31
						ps67.OverlayValues[33] = d33
						ps67.OverlayValues[34] = d34
						ps67.OverlayValues[37] = d37
						ps67.OverlayValues[62] = d62
						ps67.OverlayValues[63] = d63
						ps67.OverlayValues[64] = d64
						ps67.OverlayValues[65] = d65
						return bbs[6].RenderPS(ps67)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d68 := ps.PhiValues[0]
							ctx.EnsureDesc(&d68)
							ctx.EmitStoreToStack(d68, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d65.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl7)
					ps69 := PhiState{General: true}
					ps69.OverlayValues = make([]JITValueDesc, 69)
					ps69.OverlayValues[1] = d1
					ps69.OverlayValues[2] = d2
					ps69.OverlayValues[3] = d3
					ps69.OverlayValues[4] = d4
					ps69.OverlayValues[5] = d5
					ps69.OverlayValues[6] = d6
					ps69.OverlayValues[18] = d18
					ps69.OverlayValues[19] = d19
					ps69.OverlayValues[20] = d20
					ps69.OverlayValues[21] = d21
					ps69.OverlayValues[22] = d22
					ps69.OverlayValues[23] = d23
					ps69.OverlayValues[24] = d24
					ps69.OverlayValues[25] = d25
					ps69.OverlayValues[26] = d26
					ps69.OverlayValues[27] = d27
					ps69.OverlayValues[28] = d28
					ps69.OverlayValues[29] = d29
					ps69.OverlayValues[31] = d31
					ps69.OverlayValues[33] = d33
					ps69.OverlayValues[34] = d34
					ps69.OverlayValues[37] = d37
					ps69.OverlayValues[62] = d62
					ps69.OverlayValues[63] = d63
					ps69.OverlayValues[64] = d64
					ps69.OverlayValues[65] = d65
					ps69.OverlayValues[68] = d68
					ps70 := PhiState{General: true}
					ps70.OverlayValues = make([]JITValueDesc, 69)
					ps70.OverlayValues[1] = d1
					ps70.OverlayValues[2] = d2
					ps70.OverlayValues[3] = d3
					ps70.OverlayValues[4] = d4
					ps70.OverlayValues[5] = d5
					ps70.OverlayValues[6] = d6
					ps70.OverlayValues[18] = d18
					ps70.OverlayValues[19] = d19
					ps70.OverlayValues[20] = d20
					ps70.OverlayValues[21] = d21
					ps70.OverlayValues[22] = d22
					ps70.OverlayValues[23] = d23
					ps70.OverlayValues[24] = d24
					ps70.OverlayValues[25] = d25
					ps70.OverlayValues[26] = d26
					ps70.OverlayValues[27] = d27
					ps70.OverlayValues[28] = d28
					ps70.OverlayValues[29] = d29
					ps70.OverlayValues[31] = d31
					ps70.OverlayValues[33] = d33
					ps70.OverlayValues[34] = d34
					ps70.OverlayValues[37] = d37
					ps70.OverlayValues[62] = d62
					ps70.OverlayValues[63] = d63
					ps70.OverlayValues[64] = d64
					ps70.OverlayValues[65] = d65
					ps70.OverlayValues[68] = d68
					snap71 := d1
					snap72 := d2
					snap73 := d3
					snap74 := d4
					snap75 := d5
					snap76 := d6
					snap77 := d18
					snap78 := d19
					snap79 := d20
					snap80 := d21
					snap81 := d22
					snap82 := d23
					snap83 := d24
					snap84 := d25
					snap85 := d26
					snap86 := d27
					snap87 := d28
					snap88 := d29
					snap89 := d31
					snap90 := d33
					snap91 := d34
					snap92 := d37
					snap93 := d62
					snap94 := d63
					snap95 := d64
					snap96 := d65
					snap97 := d68
					alloc98 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps70)
					}
					ctx.RestoreAllocState(alloc98)
					d1 = snap71
					d2 = snap72
					d3 = snap73
					d4 = snap74
					d5 = snap75
					d6 = snap76
					d18 = snap77
					d19 = snap78
					d20 = snap79
					d21 = snap80
					d22 = snap81
					d23 = snap82
					d24 = snap83
					d25 = snap84
					d26 = snap85
					d27 = snap86
					d28 = snap87
					d29 = snap88
					d31 = snap89
					d33 = snap90
					d34 = snap91
					d37 = snap92
					d62 = snap93
					d63 = snap94
					d64 = snap95
					d65 = snap96
					d68 = snap97
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps69)
					}
					return result
					ctx.FreeDesc(&d64)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					ctx.ReclaimUntrackedRegs()
					d99 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d100 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d99}, 2)
					ctx.EmitMovPairToResult(&d100, &result)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					ctx.ReclaimUntrackedRegs()
					d101 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d101)
					var d102 JITValueDesc
					if d101.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d101.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d101.Reg, 2)
						ctx.EmitSetcc(r4, CondSignedGreater)
						d102 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d102)
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
							ps104.OverlayValues[24] = d24
							ps104.OverlayValues[25] = d25
							ps104.OverlayValues[26] = d26
							ps104.OverlayValues[27] = d27
							ps104.OverlayValues[28] = d28
							ps104.OverlayValues[29] = d29
							ps104.OverlayValues[31] = d31
							ps104.OverlayValues[33] = d33
							ps104.OverlayValues[34] = d34
							ps104.OverlayValues[37] = d37
							ps104.OverlayValues[62] = d62
							ps104.OverlayValues[63] = d63
							ps104.OverlayValues[64] = d64
							ps104.OverlayValues[65] = d65
							ps104.OverlayValues[68] = d68
							ps104.OverlayValues[99] = d99
							ps104.OverlayValues[100] = d100
							ps104.OverlayValues[101] = d101
							ps104.OverlayValues[102] = d102
							ps104.OverlayValues[103] = d103
							return bbs[7].RenderPS(ps104)
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
						ps105.OverlayValues[24] = d24
						ps105.OverlayValues[25] = d25
						ps105.OverlayValues[26] = d26
						ps105.OverlayValues[27] = d27
						ps105.OverlayValues[28] = d28
						ps105.OverlayValues[29] = d29
						ps105.OverlayValues[31] = d31
						ps105.OverlayValues[33] = d33
						ps105.OverlayValues[34] = d34
						ps105.OverlayValues[37] = d37
						ps105.OverlayValues[62] = d62
						ps105.OverlayValues[63] = d63
						ps105.OverlayValues[64] = d64
						ps105.OverlayValues[65] = d65
						ps105.OverlayValues[68] = d68
						ps105.OverlayValues[99] = d99
						ps105.OverlayValues[100] = d100
						ps105.OverlayValues[101] = d101
						ps105.OverlayValues[102] = d102
						ps105.OverlayValues[103] = d103
						return bbs[8].RenderPS(ps105)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d103.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl9)
					ps106 := PhiState{General: true}
					ps106.OverlayValues = make([]JITValueDesc, 104)
					ps106.OverlayValues[1] = d1
					ps106.OverlayValues[2] = d2
					ps106.OverlayValues[3] = d3
					ps106.OverlayValues[4] = d4
					ps106.OverlayValues[5] = d5
					ps106.OverlayValues[6] = d6
					ps106.OverlayValues[18] = d18
					ps106.OverlayValues[19] = d19
					ps106.OverlayValues[20] = d20
					ps106.OverlayValues[21] = d21
					ps106.OverlayValues[22] = d22
					ps106.OverlayValues[23] = d23
					ps106.OverlayValues[24] = d24
					ps106.OverlayValues[25] = d25
					ps106.OverlayValues[26] = d26
					ps106.OverlayValues[27] = d27
					ps106.OverlayValues[28] = d28
					ps106.OverlayValues[29] = d29
					ps106.OverlayValues[31] = d31
					ps106.OverlayValues[33] = d33
					ps106.OverlayValues[34] = d34
					ps106.OverlayValues[37] = d37
					ps106.OverlayValues[62] = d62
					ps106.OverlayValues[63] = d63
					ps106.OverlayValues[64] = d64
					ps106.OverlayValues[65] = d65
					ps106.OverlayValues[68] = d68
					ps106.OverlayValues[99] = d99
					ps106.OverlayValues[100] = d100
					ps106.OverlayValues[101] = d101
					ps106.OverlayValues[102] = d102
					ps106.OverlayValues[103] = d103
					ps107 := PhiState{General: true}
					ps107.OverlayValues = make([]JITValueDesc, 104)
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
					ps107.OverlayValues[24] = d24
					ps107.OverlayValues[25] = d25
					ps107.OverlayValues[26] = d26
					ps107.OverlayValues[27] = d27
					ps107.OverlayValues[28] = d28
					ps107.OverlayValues[29] = d29
					ps107.OverlayValues[31] = d31
					ps107.OverlayValues[33] = d33
					ps107.OverlayValues[34] = d34
					ps107.OverlayValues[37] = d37
					ps107.OverlayValues[62] = d62
					ps107.OverlayValues[63] = d63
					ps107.OverlayValues[64] = d64
					ps107.OverlayValues[65] = d65
					ps107.OverlayValues[68] = d68
					ps107.OverlayValues[99] = d99
					ps107.OverlayValues[100] = d100
					ps107.OverlayValues[101] = d101
					ps107.OverlayValues[102] = d102
					ps107.OverlayValues[103] = d103
					snap108 := d1
					snap109 := d2
					snap110 := d3
					snap111 := d4
					snap112 := d5
					snap113 := d6
					snap114 := d18
					snap115 := d19
					snap116 := d20
					snap117 := d21
					snap118 := d22
					snap119 := d23
					snap120 := d24
					snap121 := d25
					snap122 := d26
					snap123 := d27
					snap124 := d28
					snap125 := d29
					snap126 := d31
					snap127 := d33
					snap128 := d34
					snap129 := d37
					snap130 := d62
					snap131 := d63
					snap132 := d64
					snap133 := d65
					snap134 := d68
					snap135 := d99
					snap136 := d100
					snap137 := d101
					snap138 := d102
					snap139 := d103
					alloc140 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps107)
					}
					ctx.RestoreAllocState(alloc140)
					d1 = snap108
					d2 = snap109
					d3 = snap110
					d4 = snap111
					d5 = snap112
					d6 = snap113
					d18 = snap114
					d19 = snap115
					d20 = snap116
					d21 = snap117
					d22 = snap118
					d23 = snap119
					d24 = snap120
					d25 = snap121
					d26 = snap122
					d27 = snap123
					d28 = snap124
					d29 = snap125
					d31 = snap126
					d33 = snap127
					d34 = snap128
					d37 = snap129
					d62 = snap130
					d63 = snap131
					d64 = snap132
					d65 = snap133
					d68 = snap134
					d99 = snap135
					d100 = snap136
					d101 = snap137
					d102 = snap138
					d103 = snap139
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps106)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
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
					ctx.ReclaimUntrackedRegs()
					d141 = args[2]
					d141.ID = 0
					ctx.EnsureDesc(&d141)
					d142 = d141
					_ = d142
					ctx.StabilizeDescForControlFlow(&d142)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d143 JITValueDesc
					if d142.Loc == LocImm {
						d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d142.Imm.Int())}
					} else if d142.Type == tagInt && d142.Loc == LocRegPair {
						ctx.FreeReg(d142.Reg)
						d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d142.Reg2}
						ctx.BindReg(d142.Reg2, &d143)
						ctx.BindReg(d142.Reg2, &d143)
					} else if d142.Type == tagInt && d142.Loc == LocReg {
						d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d142.Reg}
						ctx.BindReg(d142.Reg, &d143)
						ctx.BindReg(d142.Reg, &d143)
					} else {
						d143 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d142}, 1)
						d143.Type = tagInt
						ctx.BindReg(d143.Reg, &d143)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d143)
					ctx.EnsureDesc(&d143)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d143)
					ctx.StabilizeDescForControlFlow(&d143)
					ctx.FreeDesc(&d141)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDescsTogether(&d1, &d143)
					var d145 JITValueDesc
					if d1.Loc == LocImm && d143.Loc == LocImm {
						d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d143.Imm.Int())}
					} else if d143.Loc == LocImm && d143.Imm.Int() == 0 {
						r5 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r5, d1.Reg)
						d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d145)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d143.Reg}
						ctx.BindReg(d143.Reg, &d145)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d143.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d143.Reg)
						d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d145)
					} else if d143.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d143.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d143.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d145)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg, d143.Reg)
						ctx.EmitMovRegReg(r6, d1.Reg)
						ctx.EmitAddInt64(r6, d143.Reg)
						d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d145)
					}
					if d145.Loc == LocReg && d1.Loc == LocReg && d145.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d145)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDescsTogether(&d145, &d22)
					var d146 JITValueDesc
					if d145.Loc == LocImm && d22.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d145.Imm.Int() > d22.Imm.Int())}
					} else if d22.Loc == LocImm {
						r7 := ctx.AllocReg()
						if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d145.Reg, int32(d22.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
							ctx.EmitCmpInt64(d145.Reg, RegR11)
						}
						ctx.EmitSetcc(r7, CondSignedGreater)
						d146 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d146)
					} else if d145.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d145.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d22.Reg)
						ctx.EmitSetcc(r8, CondSignedGreater)
						d146 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d146)
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpInt64(d145.Reg, d22.Reg)
						ctx.EmitSetcc(r9, CondSignedGreater)
						d146 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d146)
					}
					ctx.FreeDesc(&d145)
					d147 = d146
					ctx.EnsureDesc(&d147)
					if d147.Loc != LocImm && d147.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d147.Loc == LocImm {
						if d147.Imm.Bool() {
							if ps.General {
							}
							ps148 := PhiState{General: ps.General}
							ps148.OverlayValues = make([]JITValueDesc, 148)
							ps148.OverlayValues[1] = d1
							ps148.OverlayValues[2] = d2
							ps148.OverlayValues[3] = d3
							ps148.OverlayValues[4] = d4
							ps148.OverlayValues[5] = d5
							ps148.OverlayValues[6] = d6
							ps148.OverlayValues[18] = d18
							ps148.OverlayValues[19] = d19
							ps148.OverlayValues[20] = d20
							ps148.OverlayValues[21] = d21
							ps148.OverlayValues[22] = d22
							ps148.OverlayValues[23] = d23
							ps148.OverlayValues[24] = d24
							ps148.OverlayValues[25] = d25
							ps148.OverlayValues[26] = d26
							ps148.OverlayValues[27] = d27
							ps148.OverlayValues[28] = d28
							ps148.OverlayValues[29] = d29
							ps148.OverlayValues[31] = d31
							ps148.OverlayValues[33] = d33
							ps148.OverlayValues[34] = d34
							ps148.OverlayValues[37] = d37
							ps148.OverlayValues[62] = d62
							ps148.OverlayValues[63] = d63
							ps148.OverlayValues[64] = d64
							ps148.OverlayValues[65] = d65
							ps148.OverlayValues[68] = d68
							ps148.OverlayValues[99] = d99
							ps148.OverlayValues[100] = d100
							ps148.OverlayValues[101] = d101
							ps148.OverlayValues[102] = d102
							ps148.OverlayValues[103] = d103
							ps148.OverlayValues[141] = d141
							ps148.OverlayValues[142] = d142
							ps148.OverlayValues[143] = d143
							ps148.OverlayValues[144] = d144
							ps148.OverlayValues[145] = d145
							ps148.OverlayValues[146] = d146
							ps148.OverlayValues[147] = d147
							return bbs[9].RenderPS(ps148)
						}
						if ps.General {
							ctx.SyncDesc(&d143)
							if d143.Loc == LocReg {
								ctx.ProtectReg(d143.Reg)
							} else if d143.Loc == LocRegPair {
								ctx.ProtectReg(d143.Reg)
								ctx.ProtectReg(d143.Reg2)
							}
							d149 = d143
							if d149.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d149)
							ctx.EmitStoreToStack(d149, int32(bbs[10].PhiBase)+int32(0))
							if d143.Loc == LocReg {
								ctx.UnprotectReg(d143.Reg)
							} else if d143.Loc == LocRegPair {
								ctx.UnprotectReg(d143.Reg)
								ctx.UnprotectReg(d143.Reg2)
							}
						}
						ps150 := PhiState{General: ps.General}
						ps150.OverlayValues = make([]JITValueDesc, 150)
						ps150.OverlayValues[1] = d1
						ps150.OverlayValues[2] = d2
						ps150.OverlayValues[3] = d3
						ps150.OverlayValues[4] = d4
						ps150.OverlayValues[5] = d5
						ps150.OverlayValues[6] = d6
						ps150.OverlayValues[18] = d18
						ps150.OverlayValues[19] = d19
						ps150.OverlayValues[20] = d20
						ps150.OverlayValues[21] = d21
						ps150.OverlayValues[22] = d22
						ps150.OverlayValues[23] = d23
						ps150.OverlayValues[24] = d24
						ps150.OverlayValues[25] = d25
						ps150.OverlayValues[26] = d26
						ps150.OverlayValues[27] = d27
						ps150.OverlayValues[28] = d28
						ps150.OverlayValues[29] = d29
						ps150.OverlayValues[31] = d31
						ps150.OverlayValues[33] = d33
						ps150.OverlayValues[34] = d34
						ps150.OverlayValues[37] = d37
						ps150.OverlayValues[62] = d62
						ps150.OverlayValues[63] = d63
						ps150.OverlayValues[64] = d64
						ps150.OverlayValues[65] = d65
						ps150.OverlayValues[68] = d68
						ps150.OverlayValues[99] = d99
						ps150.OverlayValues[100] = d100
						ps150.OverlayValues[101] = d101
						ps150.OverlayValues[102] = d102
						ps150.OverlayValues[103] = d103
						ps150.OverlayValues[141] = d141
						ps150.OverlayValues[142] = d142
						ps150.OverlayValues[143] = d143
						ps150.OverlayValues[144] = d144
						ps150.OverlayValues[145] = d145
						ps150.OverlayValues[146] = d146
						ps150.OverlayValues[147] = d147
						ps150.OverlayValues[149] = d149
						ps150.PhiValues = make([]JITValueDesc, 1)
						d151 = d143
						ps150.PhiValues[0] = d151
						return bbs[10].RenderPS(ps150)
					}
					if !ps.General {
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d147.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl25)
					ctx.SyncDesc(&d143)
					if d143.Loc == LocReg {
						ctx.ProtectReg(d143.Reg)
					} else if d143.Loc == LocRegPair {
						ctx.ProtectReg(d143.Reg)
						ctx.ProtectReg(d143.Reg2)
					}
					d152 = d143
					if d152.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d152)
					ctx.EmitStoreToStack(d152, int32(bbs[10].PhiBase)+int32(0))
					if d143.Loc == LocReg {
						ctx.UnprotectReg(d143.Reg)
					} else if d143.Loc == LocRegPair {
						ctx.UnprotectReg(d143.Reg)
						ctx.UnprotectReg(d143.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ps153 := PhiState{General: true}
					ps153.OverlayValues = make([]JITValueDesc, 153)
					ps153.OverlayValues[1] = d1
					ps153.OverlayValues[2] = d2
					ps153.OverlayValues[3] = d3
					ps153.OverlayValues[4] = d4
					ps153.OverlayValues[5] = d5
					ps153.OverlayValues[6] = d6
					ps153.OverlayValues[18] = d18
					ps153.OverlayValues[19] = d19
					ps153.OverlayValues[20] = d20
					ps153.OverlayValues[21] = d21
					ps153.OverlayValues[22] = d22
					ps153.OverlayValues[23] = d23
					ps153.OverlayValues[24] = d24
					ps153.OverlayValues[25] = d25
					ps153.OverlayValues[26] = d26
					ps153.OverlayValues[27] = d27
					ps153.OverlayValues[28] = d28
					ps153.OverlayValues[29] = d29
					ps153.OverlayValues[31] = d31
					ps153.OverlayValues[33] = d33
					ps153.OverlayValues[34] = d34
					ps153.OverlayValues[37] = d37
					ps153.OverlayValues[62] = d62
					ps153.OverlayValues[63] = d63
					ps153.OverlayValues[64] = d64
					ps153.OverlayValues[65] = d65
					ps153.OverlayValues[68] = d68
					ps153.OverlayValues[99] = d99
					ps153.OverlayValues[100] = d100
					ps153.OverlayValues[101] = d101
					ps153.OverlayValues[102] = d102
					ps153.OverlayValues[103] = d103
					ps153.OverlayValues[141] = d141
					ps153.OverlayValues[142] = d142
					ps153.OverlayValues[143] = d143
					ps153.OverlayValues[144] = d144
					ps153.OverlayValues[145] = d145
					ps153.OverlayValues[146] = d146
					ps153.OverlayValues[147] = d147
					ps153.OverlayValues[149] = d149
					ps153.OverlayValues[151] = d151
					ps153.OverlayValues[152] = d152
					ps154 := PhiState{General: true}
					ps154.OverlayValues = make([]JITValueDesc, 153)
					ps154.OverlayValues[1] = d1
					ps154.OverlayValues[2] = d2
					ps154.OverlayValues[3] = d3
					ps154.OverlayValues[4] = d4
					ps154.OverlayValues[5] = d5
					ps154.OverlayValues[6] = d6
					ps154.OverlayValues[18] = d18
					ps154.OverlayValues[19] = d19
					ps154.OverlayValues[20] = d20
					ps154.OverlayValues[21] = d21
					ps154.OverlayValues[22] = d22
					ps154.OverlayValues[23] = d23
					ps154.OverlayValues[24] = d24
					ps154.OverlayValues[25] = d25
					ps154.OverlayValues[26] = d26
					ps154.OverlayValues[27] = d27
					ps154.OverlayValues[28] = d28
					ps154.OverlayValues[29] = d29
					ps154.OverlayValues[31] = d31
					ps154.OverlayValues[33] = d33
					ps154.OverlayValues[34] = d34
					ps154.OverlayValues[37] = d37
					ps154.OverlayValues[62] = d62
					ps154.OverlayValues[63] = d63
					ps154.OverlayValues[64] = d64
					ps154.OverlayValues[65] = d65
					ps154.OverlayValues[68] = d68
					ps154.OverlayValues[99] = d99
					ps154.OverlayValues[100] = d100
					ps154.OverlayValues[101] = d101
					ps154.OverlayValues[102] = d102
					ps154.OverlayValues[103] = d103
					ps154.OverlayValues[141] = d141
					ps154.OverlayValues[142] = d142
					ps154.OverlayValues[143] = d143
					ps154.OverlayValues[144] = d144
					ps154.OverlayValues[145] = d145
					ps154.OverlayValues[146] = d146
					ps154.OverlayValues[147] = d147
					ps154.OverlayValues[149] = d149
					ps154.OverlayValues[151] = d151
					ps154.OverlayValues[152] = d152
					ps154.PhiValues = make([]JITValueDesc, 1)
					d155 = d143
					ps154.PhiValues[0] = d155
					snap156 := d1
					snap157 := d2
					snap158 := d3
					snap159 := d4
					snap160 := d5
					snap161 := d6
					snap162 := d18
					snap163 := d19
					snap164 := d20
					snap165 := d21
					snap166 := d22
					snap167 := d23
					snap168 := d24
					snap169 := d25
					snap170 := d26
					snap171 := d27
					snap172 := d28
					snap173 := d29
					snap174 := d31
					snap175 := d33
					snap176 := d34
					snap177 := d37
					snap178 := d62
					snap179 := d63
					snap180 := d64
					snap181 := d65
					snap182 := d68
					snap183 := d99
					snap184 := d100
					snap185 := d101
					snap186 := d102
					snap187 := d103
					snap188 := d141
					snap189 := d142
					snap190 := d143
					snap191 := d144
					snap192 := d145
					snap193 := d146
					snap194 := d147
					snap195 := d149
					snap196 := d151
					snap197 := d152
					snap198 := d155
					alloc199 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps154)
					}
					ctx.RestoreAllocState(alloc199)
					d1 = snap156
					d2 = snap157
					d3 = snap158
					d4 = snap159
					d5 = snap160
					d6 = snap161
					d18 = snap162
					d19 = snap163
					d20 = snap164
					d21 = snap165
					d22 = snap166
					d23 = snap167
					d24 = snap168
					d25 = snap169
					d26 = snap170
					d27 = snap171
					d28 = snap172
					d29 = snap173
					d31 = snap174
					d33 = snap175
					d34 = snap176
					d37 = snap177
					d62 = snap178
					d63 = snap179
					d64 = snap180
					d65 = snap181
					d68 = snap182
					d99 = snap183
					d100 = snap184
					d101 = snap185
					d102 = snap186
					d103 = snap187
					d141 = snap188
					d142 = snap189
					d143 = snap190
					d144 = snap191
					d145 = snap192
					d146 = snap193
					d147 = snap194
					d149 = snap195
					d151 = snap196
					d152 = snap197
					d155 = snap198
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps153)
					}
					return result
					ctx.FreeDesc(&d146)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
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
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
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
					ctx.EnsureDesc(&d1)
					var d200 JITValueDesc
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocRegPair || d20.Loc == LocRegTriple {
						d200 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg2}
						ctx.BindReg(d20.Reg2, &d200)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d200)
					var d202 JITValueDesc
					if d200.Loc == LocImm && d1.Loc == LocImm {
						d202 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d200.Imm.Int() - d1.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d200.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d200.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d200.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r10, RegR11)
						} else {
							ctx.EmitSubInt64(r10, d1.Reg)
						}
						d202 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d202)
					}
					var d203 JITValueDesc
					r11 := ctx.EmitSliceDataAfterLow(&d20, &d1, 1)
					d203 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d203)
					ctx.BindReg(r11, &d203)
					var d204 JITValueDesc
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d203)
					ctx.EnsureDesc(&d203)
					if d203.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d203.Imm.Int()))
					} else {
						r12 = d203.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d202)
					ctx.EnsureDesc(&d202)
					if d202.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d202.Imm.Int()))
					} else {
						r13 = d202.Reg
					}
					ctx.ProtectReg(r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d204 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d204)
					ctx.BindReg(r13, &d204)
					ctx.BindReg(r12, &d204)
					ctx.BindReg(r13, &d204)
					ctx.EnsureDesc(&d204)
					d205 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d204}, 2)
					ctx.EmitMovPairToResult(&d205, &result)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
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
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d22, &d1)
					var d206 JITValueDesc
					if d22.Loc == LocImm && d1.Loc == LocImm {
						d206 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d22.Imm.Int() - d1.Imm.Int())}
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						r14 := ctx.AllocRegExcept(d22.Reg)
						ctx.EmitMovRegReg(r14, d22.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d206)
					} else if d22.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
						ctx.EmitSubInt64(scratch, d1.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d206)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d22.Reg)
						ctx.EmitMovRegReg(scratch, d22.Reg)
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d206)
					} else {
						r15 := ctx.AllocRegExcept(d22.Reg, d1.Reg)
						ctx.EmitMovRegReg(r15, d22.Reg)
						ctx.EmitSubInt64(r15, d1.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d206)
					}
					if d206.Loc == LocReg && d22.Loc == LocReg && d206.Reg == d22.Reg {
						ctx.TransferReg(d22.Reg)
						d22.Loc = LocNone
					}
					ctx.EnsureDesc(&d206)
					ctx.EmitStoreToStack(d206, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d206)
					ctx.FreeDesc(&d22)
					if ps.General {
					}
					ps207 := PhiState{General: ps.General}
					ps207.OverlayValues = make([]JITValueDesc, 207)
					ps207.OverlayValues[1] = d1
					ps207.OverlayValues[2] = d2
					ps207.OverlayValues[3] = d3
					ps207.OverlayValues[4] = d4
					ps207.OverlayValues[5] = d5
					ps207.OverlayValues[6] = d6
					ps207.OverlayValues[18] = d18
					ps207.OverlayValues[19] = d19
					ps207.OverlayValues[20] = d20
					ps207.OverlayValues[21] = d21
					ps207.OverlayValues[22] = d22
					ps207.OverlayValues[23] = d23
					ps207.OverlayValues[24] = d24
					ps207.OverlayValues[25] = d25
					ps207.OverlayValues[26] = d26
					ps207.OverlayValues[27] = d27
					ps207.OverlayValues[28] = d28
					ps207.OverlayValues[29] = d29
					ps207.OverlayValues[31] = d31
					ps207.OverlayValues[33] = d33
					ps207.OverlayValues[34] = d34
					ps207.OverlayValues[37] = d37
					ps207.OverlayValues[62] = d62
					ps207.OverlayValues[63] = d63
					ps207.OverlayValues[64] = d64
					ps207.OverlayValues[65] = d65
					ps207.OverlayValues[68] = d68
					ps207.OverlayValues[99] = d99
					ps207.OverlayValues[100] = d100
					ps207.OverlayValues[101] = d101
					ps207.OverlayValues[102] = d102
					ps207.OverlayValues[103] = d103
					ps207.OverlayValues[141] = d141
					ps207.OverlayValues[142] = d142
					ps207.OverlayValues[143] = d143
					ps207.OverlayValues[144] = d144
					ps207.OverlayValues[145] = d145
					ps207.OverlayValues[146] = d146
					ps207.OverlayValues[147] = d147
					ps207.OverlayValues[149] = d149
					ps207.OverlayValues[151] = d151
					ps207.OverlayValues[152] = d152
					ps207.OverlayValues[155] = d155
					ps207.OverlayValues[200] = d200
					ps207.OverlayValues[201] = d201
					ps207.OverlayValues[202] = d202
					ps207.OverlayValues[203] = d203
					ps207.OverlayValues[204] = d204
					ps207.OverlayValues[205] = d205
					ps207.OverlayValues[206] = d206
					ps207.PhiValues = make([]JITValueDesc, 1)
					if ps207.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps207)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d208 := ps.PhiValues[0]
							ctx.EnsureDesc(&d208)
							ctx.EmitStoreToStack(d208, int32(bbs[10].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
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
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
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
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d2)
					var d209 JITValueDesc
					if d2.Loc == LocImm {
						d209 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 0)}
					} else {
						r16 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r16, CondSignedLess)
						d209 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
						ctx.BindReg(r16, &d209)
					}
					d210 = d209
					ctx.EnsureDesc(&d210)
					if d210.Loc != LocImm && d210.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d210.Loc == LocImm {
						if d210.Imm.Bool() {
							if ps.General {
							}
							ps211 := PhiState{General: ps.General}
							ps211.OverlayValues = make([]JITValueDesc, 211)
							ps211.OverlayValues[1] = d1
							ps211.OverlayValues[2] = d2
							ps211.OverlayValues[3] = d3
							ps211.OverlayValues[4] = d4
							ps211.OverlayValues[5] = d5
							ps211.OverlayValues[6] = d6
							ps211.OverlayValues[18] = d18
							ps211.OverlayValues[19] = d19
							ps211.OverlayValues[20] = d20
							ps211.OverlayValues[21] = d21
							ps211.OverlayValues[22] = d22
							ps211.OverlayValues[23] = d23
							ps211.OverlayValues[24] = d24
							ps211.OverlayValues[25] = d25
							ps211.OverlayValues[26] = d26
							ps211.OverlayValues[27] = d27
							ps211.OverlayValues[28] = d28
							ps211.OverlayValues[29] = d29
							ps211.OverlayValues[31] = d31
							ps211.OverlayValues[33] = d33
							ps211.OverlayValues[34] = d34
							ps211.OverlayValues[37] = d37
							ps211.OverlayValues[62] = d62
							ps211.OverlayValues[63] = d63
							ps211.OverlayValues[64] = d64
							ps211.OverlayValues[65] = d65
							ps211.OverlayValues[68] = d68
							ps211.OverlayValues[99] = d99
							ps211.OverlayValues[100] = d100
							ps211.OverlayValues[101] = d101
							ps211.OverlayValues[102] = d102
							ps211.OverlayValues[103] = d103
							ps211.OverlayValues[141] = d141
							ps211.OverlayValues[142] = d142
							ps211.OverlayValues[143] = d143
							ps211.OverlayValues[144] = d144
							ps211.OverlayValues[145] = d145
							ps211.OverlayValues[146] = d146
							ps211.OverlayValues[147] = d147
							ps211.OverlayValues[149] = d149
							ps211.OverlayValues[151] = d151
							ps211.OverlayValues[152] = d152
							ps211.OverlayValues[155] = d155
							ps211.OverlayValues[200] = d200
							ps211.OverlayValues[201] = d201
							ps211.OverlayValues[202] = d202
							ps211.OverlayValues[203] = d203
							ps211.OverlayValues[204] = d204
							ps211.OverlayValues[205] = d205
							ps211.OverlayValues[206] = d206
							ps211.OverlayValues[208] = d208
							ps211.OverlayValues[209] = d209
							ps211.OverlayValues[210] = d210
							return bbs[11].RenderPS(ps211)
						}
						if ps.General {
						}
						ps212 := PhiState{General: ps.General}
						ps212.OverlayValues = make([]JITValueDesc, 211)
						ps212.OverlayValues[1] = d1
						ps212.OverlayValues[2] = d2
						ps212.OverlayValues[3] = d3
						ps212.OverlayValues[4] = d4
						ps212.OverlayValues[5] = d5
						ps212.OverlayValues[6] = d6
						ps212.OverlayValues[18] = d18
						ps212.OverlayValues[19] = d19
						ps212.OverlayValues[20] = d20
						ps212.OverlayValues[21] = d21
						ps212.OverlayValues[22] = d22
						ps212.OverlayValues[23] = d23
						ps212.OverlayValues[24] = d24
						ps212.OverlayValues[25] = d25
						ps212.OverlayValues[26] = d26
						ps212.OverlayValues[27] = d27
						ps212.OverlayValues[28] = d28
						ps212.OverlayValues[29] = d29
						ps212.OverlayValues[31] = d31
						ps212.OverlayValues[33] = d33
						ps212.OverlayValues[34] = d34
						ps212.OverlayValues[37] = d37
						ps212.OverlayValues[62] = d62
						ps212.OverlayValues[63] = d63
						ps212.OverlayValues[64] = d64
						ps212.OverlayValues[65] = d65
						ps212.OverlayValues[68] = d68
						ps212.OverlayValues[99] = d99
						ps212.OverlayValues[100] = d100
						ps212.OverlayValues[101] = d101
						ps212.OverlayValues[102] = d102
						ps212.OverlayValues[103] = d103
						ps212.OverlayValues[141] = d141
						ps212.OverlayValues[142] = d142
						ps212.OverlayValues[143] = d143
						ps212.OverlayValues[144] = d144
						ps212.OverlayValues[145] = d145
						ps212.OverlayValues[146] = d146
						ps212.OverlayValues[147] = d147
						ps212.OverlayValues[149] = d149
						ps212.OverlayValues[151] = d151
						ps212.OverlayValues[152] = d152
						ps212.OverlayValues[155] = d155
						ps212.OverlayValues[200] = d200
						ps212.OverlayValues[201] = d201
						ps212.OverlayValues[202] = d202
						ps212.OverlayValues[203] = d203
						ps212.OverlayValues[204] = d204
						ps212.OverlayValues[205] = d205
						ps212.OverlayValues[206] = d206
						ps212.OverlayValues[208] = d208
						ps212.OverlayValues[209] = d209
						ps212.OverlayValues[210] = d210
						return bbs[12].RenderPS(ps212)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d213 := ps.PhiValues[0]
							ctx.EnsureDesc(&d213)
							ctx.EmitStoreToStack(d213, int32(bbs[10].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d210.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl13)
					ps214 := PhiState{General: true}
					ps214.OverlayValues = make([]JITValueDesc, 214)
					ps214.OverlayValues[1] = d1
					ps214.OverlayValues[2] = d2
					ps214.OverlayValues[3] = d3
					ps214.OverlayValues[4] = d4
					ps214.OverlayValues[5] = d5
					ps214.OverlayValues[6] = d6
					ps214.OverlayValues[18] = d18
					ps214.OverlayValues[19] = d19
					ps214.OverlayValues[20] = d20
					ps214.OverlayValues[21] = d21
					ps214.OverlayValues[22] = d22
					ps214.OverlayValues[23] = d23
					ps214.OverlayValues[24] = d24
					ps214.OverlayValues[25] = d25
					ps214.OverlayValues[26] = d26
					ps214.OverlayValues[27] = d27
					ps214.OverlayValues[28] = d28
					ps214.OverlayValues[29] = d29
					ps214.OverlayValues[31] = d31
					ps214.OverlayValues[33] = d33
					ps214.OverlayValues[34] = d34
					ps214.OverlayValues[37] = d37
					ps214.OverlayValues[62] = d62
					ps214.OverlayValues[63] = d63
					ps214.OverlayValues[64] = d64
					ps214.OverlayValues[65] = d65
					ps214.OverlayValues[68] = d68
					ps214.OverlayValues[99] = d99
					ps214.OverlayValues[100] = d100
					ps214.OverlayValues[101] = d101
					ps214.OverlayValues[102] = d102
					ps214.OverlayValues[103] = d103
					ps214.OverlayValues[141] = d141
					ps214.OverlayValues[142] = d142
					ps214.OverlayValues[143] = d143
					ps214.OverlayValues[144] = d144
					ps214.OverlayValues[145] = d145
					ps214.OverlayValues[146] = d146
					ps214.OverlayValues[147] = d147
					ps214.OverlayValues[149] = d149
					ps214.OverlayValues[151] = d151
					ps214.OverlayValues[152] = d152
					ps214.OverlayValues[155] = d155
					ps214.OverlayValues[200] = d200
					ps214.OverlayValues[201] = d201
					ps214.OverlayValues[202] = d202
					ps214.OverlayValues[203] = d203
					ps214.OverlayValues[204] = d204
					ps214.OverlayValues[205] = d205
					ps214.OverlayValues[206] = d206
					ps214.OverlayValues[208] = d208
					ps214.OverlayValues[209] = d209
					ps214.OverlayValues[210] = d210
					ps214.OverlayValues[213] = d213
					ps215 := PhiState{General: true}
					ps215.OverlayValues = make([]JITValueDesc, 214)
					ps215.OverlayValues[1] = d1
					ps215.OverlayValues[2] = d2
					ps215.OverlayValues[3] = d3
					ps215.OverlayValues[4] = d4
					ps215.OverlayValues[5] = d5
					ps215.OverlayValues[6] = d6
					ps215.OverlayValues[18] = d18
					ps215.OverlayValues[19] = d19
					ps215.OverlayValues[20] = d20
					ps215.OverlayValues[21] = d21
					ps215.OverlayValues[22] = d22
					ps215.OverlayValues[23] = d23
					ps215.OverlayValues[24] = d24
					ps215.OverlayValues[25] = d25
					ps215.OverlayValues[26] = d26
					ps215.OverlayValues[27] = d27
					ps215.OverlayValues[28] = d28
					ps215.OverlayValues[29] = d29
					ps215.OverlayValues[31] = d31
					ps215.OverlayValues[33] = d33
					ps215.OverlayValues[34] = d34
					ps215.OverlayValues[37] = d37
					ps215.OverlayValues[62] = d62
					ps215.OverlayValues[63] = d63
					ps215.OverlayValues[64] = d64
					ps215.OverlayValues[65] = d65
					ps215.OverlayValues[68] = d68
					ps215.OverlayValues[99] = d99
					ps215.OverlayValues[100] = d100
					ps215.OverlayValues[101] = d101
					ps215.OverlayValues[102] = d102
					ps215.OverlayValues[103] = d103
					ps215.OverlayValues[141] = d141
					ps215.OverlayValues[142] = d142
					ps215.OverlayValues[143] = d143
					ps215.OverlayValues[144] = d144
					ps215.OverlayValues[145] = d145
					ps215.OverlayValues[146] = d146
					ps215.OverlayValues[147] = d147
					ps215.OverlayValues[149] = d149
					ps215.OverlayValues[151] = d151
					ps215.OverlayValues[152] = d152
					ps215.OverlayValues[155] = d155
					ps215.OverlayValues[200] = d200
					ps215.OverlayValues[201] = d201
					ps215.OverlayValues[202] = d202
					ps215.OverlayValues[203] = d203
					ps215.OverlayValues[204] = d204
					ps215.OverlayValues[205] = d205
					ps215.OverlayValues[206] = d206
					ps215.OverlayValues[208] = d208
					ps215.OverlayValues[209] = d209
					ps215.OverlayValues[210] = d210
					ps215.OverlayValues[213] = d213
					snap216 := d1
					snap217 := d2
					snap218 := d3
					snap219 := d4
					snap220 := d5
					snap221 := d6
					snap222 := d18
					snap223 := d19
					snap224 := d20
					snap225 := d21
					snap226 := d22
					snap227 := d23
					snap228 := d24
					snap229 := d25
					snap230 := d26
					snap231 := d27
					snap232 := d28
					snap233 := d29
					snap234 := d31
					snap235 := d33
					snap236 := d34
					snap237 := d37
					snap238 := d62
					snap239 := d63
					snap240 := d64
					snap241 := d65
					snap242 := d68
					snap243 := d99
					snap244 := d100
					snap245 := d101
					snap246 := d102
					snap247 := d103
					snap248 := d141
					snap249 := d142
					snap250 := d143
					snap251 := d144
					snap252 := d145
					snap253 := d146
					snap254 := d147
					snap255 := d149
					snap256 := d151
					snap257 := d152
					snap258 := d155
					snap259 := d200
					snap260 := d201
					snap261 := d202
					snap262 := d203
					snap263 := d204
					snap264 := d205
					snap265 := d206
					snap266 := d208
					snap267 := d209
					snap268 := d210
					snap269 := d213
					alloc270 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps215)
					}
					ctx.RestoreAllocState(alloc270)
					d1 = snap216
					d2 = snap217
					d3 = snap218
					d4 = snap219
					d5 = snap220
					d6 = snap221
					d18 = snap222
					d19 = snap223
					d20 = snap224
					d21 = snap225
					d22 = snap226
					d23 = snap227
					d24 = snap228
					d25 = snap229
					d26 = snap230
					d27 = snap231
					d28 = snap232
					d29 = snap233
					d31 = snap234
					d33 = snap235
					d34 = snap236
					d37 = snap237
					d62 = snap238
					d63 = snap239
					d64 = snap240
					d65 = snap241
					d68 = snap242
					d99 = snap243
					d100 = snap244
					d101 = snap245
					d102 = snap246
					d103 = snap247
					d141 = snap248
					d142 = snap249
					d143 = snap250
					d144 = snap251
					d145 = snap252
					d146 = snap253
					d147 = snap254
					d149 = snap255
					d151 = snap256
					d152 = snap257
					d155 = snap258
					d200 = snap259
					d201 = snap260
					d202 = snap261
					d203 = snap262
					d204 = snap263
					d205 = snap264
					d206 = snap265
					d208 = snap266
					d209 = snap267
					d210 = snap268
					d213 = snap269
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps214)
					}
					return result
					ctx.FreeDesc(&d209)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
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
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
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
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					ctx.ReclaimUntrackedRegs()
					d271 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d272 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d271}, 2)
					ctx.EmitMovPairToResult(&d272, &result)
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
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
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
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
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
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDescsTogether(&d1, &d2)
					var d273 JITValueDesc
					if d1.Loc == LocImm && d2.Loc == LocImm {
						d273 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d2.Imm.Int())}
					} else if d2.Loc == LocImm && d2.Imm.Int() == 0 {
						r17 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(r17, d1.Reg)
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d273)
					} else if d1.Loc == LocImm && d1.Imm.Int() == 0 {
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
						ctx.BindReg(d2.Reg, &d273)
					} else if d1.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(scratch, d2.Reg)
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d273)
					} else if d2.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
							ctx.EmitAddInt64(scratch, RegR11)
						}
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d273)
					} else {
						r18 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
						ctx.EmitMovRegReg(r18, d1.Reg)
						ctx.EmitAddInt64(r18, d2.Reg)
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d273)
					}
					if d273.Loc == LocReg && d1.Loc == LocReg && d273.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d273)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d273)
					var d275 JITValueDesc
					if d273.Loc == LocImm && d1.Loc == LocImm {
						d275 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d273.Imm.Int() - d1.Imm.Int())}
					} else {
						r19 := ctx.AllocReg()
						if d273.Loc == LocImm {
							ctx.EmitMovRegImm64(r19, uint64(d273.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r19, d273.Reg)
						}
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitSubInt64(r19, RegR11)
						} else {
							ctx.EmitSubInt64(r19, d1.Reg)
						}
						d275 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d275)
					}
					var d276 JITValueDesc
					r20 := ctx.EmitSliceDataAfterLow(&d20, &d1, 1)
					d276 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
					ctx.BindReg(r20, &d276)
					ctx.BindReg(r20, &d276)
					var d277 JITValueDesc
					var r21 Reg
					var r22 Reg
					ctx.SyncDesc(&d276)
					ctx.EnsureDesc(&d276)
					if d276.Loc == LocImm {
						r21 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r21, uint64(d276.Imm.Int()))
					} else {
						r21 = d276.Reg
					}
					ctx.ProtectReg(r21)
					ctx.SyncDesc(&d275)
					ctx.EnsureDesc(&d275)
					if d275.Loc == LocImm {
						r22 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r22, uint64(d275.Imm.Int()))
					} else {
						r22 = d275.Reg
					}
					ctx.ProtectReg(r22)
					ctx.UnprotectReg(r22)
					ctx.UnprotectReg(r21)
					d277 = JITValueDesc{Loc: LocRegPair, Reg: r21, Reg2: r22}
					ctx.BindReg(r21, &d277)
					ctx.BindReg(r22, &d277)
					ctx.BindReg(r21, &d277)
					ctx.BindReg(r22, &d277)
					ctx.FreeDesc(&d1)
					ctx.FreeDesc(&d273)
					ctx.EnsureDesc(&d277)
					d278 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d277}, 2)
					ctx.EmitMovPairToResult(&d278, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps279 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps279)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["simplify"].Fn, args, result)
				}
				declaration := declarations["simplify"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strlen"].Fn, args, result)
				}
				declaration := declarations["strlen"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strlike"].Fn, args, result)
				}
				declaration := declarations["strlike"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				var bbs [6]BBDescriptor
				bbs[5].PhiBase = int32(phiBase0) + int32(0)
				bbs[5].PhiCount = uint16(1)
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
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					d20 = args[1]
					d20.ID = 0
					d22 = d20
					ctx.SyncDesc(&d22)
					if d22.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d22.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d22.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d22 = tmpScalar
					}
					d22 = JITPrepareScmerGoArg(ctx, d22)
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair && d22.Loc != LocInputPair {
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
					ctx.SyncDesc(&d82)
					if d82.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d82.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d82.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d82 = tmpScalar
					}
					d82 = JITPrepareScmerGoArg(ctx, d82)
					if d82.Loc != LocRegPair && d82.Loc != LocStackPair && d82.Loc != LocInputPair {
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
					if d81.Loc != LocRegPair && d81.Loc != LocStackPair && d81.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d81)
					d83 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d81}, 2)
					d83.NoHeapPointer = false
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
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg0)")
					}
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d21.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d21.Imm)
						ptrWord, _ := d21.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d21.Imm.String())))
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
					if d21.Loc != LocRegPair && d21.Loc != LocStackPair && d21.Loc != LocInputPair {
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
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d21)
					ctx.SyncDesc(&d1)
					d88 = ctx.EmitGoCallScalar(GoFuncAddr(StrLikeCollation), []JITValueDesc{d18, d21, d1}, 1)
					d88.NoHeapPointer = true
					ctx.EmitAndRegImm32(d88.Reg, 1)
					d88.Type = tagBool
					ctx.BindReg(d88.Reg, &d88)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						ctx.EmitMakeBool(result, d88)
					} else {
						ctx.EmitMovToReg(result.Reg2, d88)
						d89 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d89)
						if d88.Loc == LocReg && d88.Reg != result.Reg2 {
							ctx.FreeReg(d88.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps90 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps90)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strlike_cs"].Fn, args, result)
				}
				declaration := declarations["strlike_cs"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d16}, 2)
					ctx.FreeDesc(&d14)
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
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
					if d18.Loc != LocRegPair && d18.Loc != LocStackPair && d18.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d18)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d15, d18}, 1)
					d20.NoHeapPointer = true
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["toLower"].Fn, args, result)
				}
				declaration := declarations["toLower"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["toUpper"].Fn, args, result)
				}
				declaration := declarations["toUpper"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["replace"].Fn, args, result)
				}
				declaration := declarations["replace"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strtrim"].Fn, args, result)
				}
				declaration := declarations["strtrim"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strltrim"].Fn, args, result)
				}
				declaration := declarations["strltrim"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["strrtrim"].Fn, args, result)
				}
				declaration := declarations["strrtrim"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_trim"].Fn, args, result)
				}
				declaration := declarations["sql_trim"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d15)
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d15}, 2)
					d17.NoHeapPointer = false
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_ltrim"].Fn, args, result)
				}
				declaration := declarations["sql_ltrim"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
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
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d15, d17}, 2)
					d18.NoHeapPointer = false
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_rtrim"].Fn, args, result)
				}
				declaration := declarations["sql_rtrim"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d14 = args[0]
					d14.ID = 0
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
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
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d15, d17}, 2)
					d18.NoHeapPointer = false
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["split"].Fn, args, result)
				}
				declaration := declarations["split"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d78 JITValueDesc
				_ = d78
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
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
					ctx.SyncDesc(&d22)
					if d22.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d22.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d22.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d22 = tmpScalar
					}
					d22 = JITPrepareScmerGoArg(ctx, d22)
					if d22.Loc != LocRegPair && d22.Loc != LocStackPair && d22.Loc != LocInputPair {
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
					ctx.SyncDesc(&d29)
					if d29.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d29.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d29.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d29 = tmpScalar
					}
					d29 = JITPrepareScmerGoArg(ctx, d29)
					if d29.Loc != LocRegPair && d29.Loc != LocStackPair && d29.Loc != LocInputPair {
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
					if d28.Loc != LocRegPair && d28.Loc != LocStackPair && d28.Loc != LocInputPair {
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Split arg1)")
					}
					ctx.SyncDesc(&d28)
					ctx.SyncDesc(&d1)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Split), []JITValueDesc{d28, d1}, 3)
					d30.NoHeapPointer = false
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
					ctx.ReclaimUntrackedRegs()
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
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDescsTogether(&d38, &d34)
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
					ctx.StabilizeDescForControlFlow(&d33)
					ctx.EnsureDesc(&d38)
					d74 = ctx.EmitSliceElementAddress(&d30, &d38, 16)
					ctx.EnsureDesc(&d74)
					r4 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegMem(r4, d74.Reg, 8)
					ctx.EmitMovRegMem(d74.Reg, d74.Reg, 0)
					d73 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d74.Reg, Reg2: r4}
					ctx.BindReg(d74.Reg, &d73)
					ctx.BindReg(r4, &d73)
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d38)
					ctx.SyncDesc(&d73)
					ctx.StabilizeDescAcrossNestedCall(&d38)
					d75 = d33
					d75.ID = 0
					d76 = d38
					d76.ID = 0
					d77 = ctx.EmitSliceElementAddress(&d75, &d76, int32(16))
					ctx.FreeDesc(&d76)
					ctx.EmitStoreScmerAt(&d77, &d73)
					ctx.FreeDesc(&d77)
					if ps.General {
						ctx.SyncDesc(&d38)
						if d38.Loc == LocReg {
							ctx.ProtectReg(d38.Reg)
						} else if d38.Loc == LocRegPair {
							ctx.ProtectReg(d38.Reg)
							ctx.ProtectReg(d38.Reg2)
						}
						d78 = d38
						if d78.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d78)
						ctx.EmitStoreToStack(d78, int32(bbs[3].PhiBase)+int32(0))
						if d38.Loc == LocReg {
							ctx.UnprotectReg(d38.Reg)
						} else if d38.Loc == LocRegPair {
							ctx.UnprotectReg(d38.Reg)
							ctx.UnprotectReg(d38.Reg2)
						}
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
					ps79.OverlayValues[73] = d73
					ps79.OverlayValues[74] = d74
					ps79.OverlayValues[75] = d75
					ps79.OverlayValues[76] = d76
					ps79.OverlayValues[77] = d77
					ps79.OverlayValues[78] = d78
					ps79.PhiValues = make([]JITValueDesc, 1)
					d80 = d38
					ps79.PhiValues[0] = d80
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d33)
					d81 = ctx.EmitNewSliceFromGoSlice(&d33)
					ctx.SyncDesc(&d81)
					if d81.Loc == LocRegPair || d81.Loc == LocStackPair || d81.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d81, &result)
						result.Type = d81.Type
					} else {
						switch d81.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d81)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d81)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d81)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d81, &result)
							result.Type = d81.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps82 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps82)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["string_repeat"].Fn, args, result)
				}
				declaration := declarations["string_repeat"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d15 = d14
					_ = d15
					ctx.StabilizeDescForControlFlow(&d15)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
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
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl10)
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
					ctx.SyncDesc(&d40)
					if d40.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d40.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d40.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d40 = tmpScalar
					}
					d40 = JITPrepareScmerGoArg(ctx, d40)
					if d40.Loc != LocRegPair && d40.Loc != LocStackPair && d40.Loc != LocInputPair {
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
					if d39.Loc != LocRegPair && d39.Loc != LocStackPair && d39.Loc != LocInputPair {
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
					d41.NoHeapPointer = false
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
				ctx.Coverage.NativeCalls++
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
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["htmlentities"].Fn, args, result)
				}
				declaration := declarations["htmlentities"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["urlencode"].Fn, args, result)
				}
				declaration := declarations["urlencode"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["urldecode"].Fn, args, result)
				}
				declaration := declarations["urldecode"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_encode"].Fn, args, result)
				}
				declaration := declarations["json_encode"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_quote"].Fn, args, result)
				}
				declaration := declarations["json_quote"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				var inlineResultOff81 int32
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var phiBase85 int32
				_ = phiBase85
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (json.NewEncoder arg0)")
					}
					ctx.SyncDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(json.NewEncoder), []JITValueDesc{d15}, 1)
					d16.NoHeapPointer = false
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
					ctx.SyncDesc(&d20)
					if d20.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d20.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d20.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d20 = tmpScalar
					}
					d20 = JITPrepareScmerGoArg(ctx, d20)
					if d20.Loc != LocRegPair && d20.Loc != LocStackPair && d20.Loc != LocInputPair {
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
					if d21.Loc != LocRegPair && d21.Loc != LocStackPair && d21.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*json.Encoder).Encode arg1)")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d21)
					d22 = ctx.EmitGoCallScalar(GoFuncAddr((*json.Encoder).Encode), []JITValueDesc{d16, d21}, 2)
					d22.NoHeapPointer = false
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
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d14)
					d75 = ctx.EmitGoCallScalar(GoFuncAddr((*bytes.Buffer).String), []JITValueDesc{d14}, 2)
					d75.NoHeapPointer = false
					ctx.BindReg(d75.Reg, &d75)
					ctx.BindReg(d75.Reg2, &d75)
					ctx.EnsureDesc(&d75)
					d76 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("\n")}
					d77 = d75
					_ = d77
					ctx.StabilizeDescForControlFlow(&d77)
					d78 = d76
					_ = d78
					ctx.StabilizeDescForControlFlow(&d78)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d77)
					ctx.EnsureDesc(&d78)
					d79 = d77
					_ = d79
					ctx.StabilizeDescForControlFlow(&d79)
					d80 = d78
					_ = d80
					ctx.StabilizeDescForControlFlow(&d80)
					ctx.StabilizeDescForControlFlow(&d77)
					inlineResultOff81 = ctx.AllocStack(int32(16))
					d82 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inlineResultOff81}
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
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d80)
					d83 = d79
					_ = d83
					ctx.StabilizeDescForControlFlow(&d83)
					d84 = d80
					_ = d84
					ctx.StabilizeDescForControlFlow(&d84)
					ctx.StabilizeDescForControlFlow(&d79)
					ctx.StabilizeDescForControlFlow(&d80)
					phiBase85 = ctx.AllocStack(int32(16))
					d86 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase85) + int32(0)}
					_ = d86
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
					d86 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase85) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d87 JITValueDesc
					if d83.SliceSizeKnown {
						d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d83.KnownSliceLen))}
					} else if d83.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d83.Imm.String())))}
					} else if d83.Loc == LocStackTriple {
						d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d83.StackOff + 8, NoHeapPointer: true}
					} else if d83.Loc == LocStackPair {
						d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d83.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d83)
						if d83.Loc == LocRegPair || d83.Loc == LocRegTriple {
							d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg2, ID: 0}
						} else if d83.Loc == LocReg {
							d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d88 JITValueDesc
					if d84.SliceSizeKnown {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d84.KnownSliceLen))}
					} else if d84.Loc == LocImm {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d84.StackOff))}
					} else if d84.Loc == LocStackTriple {
						d88 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d84.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d84)
						if d84.Loc == LocRegPair || d84.Loc == LocRegTriple {
							d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d84.Reg2, ID: 0}
						} else if d84.Loc == LocReg {
							d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d84.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDescsTogether(&d87, &d88)
					var d89 JITValueDesc
					if d87.Loc == LocImm && d88.Loc == LocImm {
						d89 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d87.Imm.Int() >= d88.Imm.Int())}
					} else if d88.Loc == LocImm {
						r1 := ctx.AllocReg()
						if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d87.Reg, int32(d88.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d88.Imm.Int()))
							ctx.EmitCmpInt64(d87.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedGreaterOrEqual)
						d89 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d89)
					} else if d87.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d87.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d88.Reg)
						ctx.EmitSetcc(r2, CondSignedGreaterOrEqual)
						d89 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d89)
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpInt64(d87.Reg, d88.Reg)
						ctx.EmitSetcc(r3, CondSignedGreaterOrEqual)
						d89 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d89)
					}
					ctx.FreeDesc(&d87)
					ctx.FreeDesc(&d88)
					ctx.ReclaimUntrackedRegs()
					d90 = d89
					ctx.EnsureDesc(&d90)
					if d90.Loc != LocImm && d90.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					if d90.Loc == LocImm {
						if d90.Imm.Bool() {
							ctx.MarkLabel(lbl22)
							ctx.EmitJmp(lbl20)
						} else {
							ctx.MarkLabel(lbl23)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase85)+int32(0))
							ctx.EmitJmp(lbl21)
						}
					} else {
						ctx.EmitCmpRegImm32(d90.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl22)
						ctx.EmitJmp(lbl23)
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl23)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase85)+int32(0))
						ctx.EmitJmp(lbl21)
					}
					ctx.FreeDesc(&d89)
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl21)
					ctx.ResolveFixups()
					d86 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase85) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r4 := ctx.AllocReg()
					ctx.EnsureDesc(&d86)
					ctx.EnsureDesc(&d86)
					if d86.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r4, d86)
					}
					ctx.EmitJmp(lbl18)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					d86 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase85) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d91 JITValueDesc
					if d83.SliceSizeKnown {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d83.KnownSliceLen))}
					} else if d83.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d83.Imm.String())))}
					} else if d83.Loc == LocStackTriple {
						d91 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d83.StackOff + 8, NoHeapPointer: true}
					} else if d83.Loc == LocStackPair {
						d91 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d83.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d83)
						if d83.Loc == LocRegPair || d83.Loc == LocRegTriple {
							d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg2, ID: 0}
						} else if d83.Loc == LocReg {
							d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d92 JITValueDesc
					if d84.SliceSizeKnown {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d84.KnownSliceLen))}
					} else if d84.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d84.StackOff))}
					} else if d84.Loc == LocStackTriple {
						d92 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d84.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d84)
						if d84.Loc == LocRegPair || d84.Loc == LocRegTriple {
							d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d84.Reg2, ID: 0}
						} else if d84.Loc == LocReg {
							d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d84.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d92)
					ctx.EnsureDescsTogether(&d91, &d92)
					var d93 JITValueDesc
					if d91.Loc == LocImm && d92.Loc == LocImm {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d91.Imm.Int() - d92.Imm.Int())}
					} else if d92.Loc == LocImm && d92.Imm.Int() == 0 {
						r5 := ctx.AllocRegExcept(d91.Reg)
						ctx.EmitMovRegReg(r5, d91.Reg)
						d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d93)
					} else if d91.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d92.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d91.Imm.Int()))
						ctx.EmitSubInt64(scratch, d92.Reg)
						d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d93)
					} else if d92.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d91.Reg)
						ctx.EmitMovRegReg(scratch, d91.Reg)
						if d92.Imm.Int() >= -2147483648 && d92.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d92.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d92.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d93)
					} else {
						r6 := ctx.AllocRegExcept(d91.Reg, d92.Reg)
						ctx.EmitMovRegReg(r6, d91.Reg)
						ctx.EmitSubInt64(r6, d92.Reg)
						d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d93)
					}
					if d93.Loc == LocReg && d91.Loc == LocReg && d93.Reg == d91.Reg {
						ctx.TransferReg(d91.Reg)
						d91.Loc = LocNone
					}
					ctx.FreeDesc(&d91)
					ctx.FreeDesc(&d92)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d93)
					var d94 JITValueDesc
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocRegPair || d83.Loc == LocRegTriple {
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg2}
						ctx.BindReg(d83.Reg2, &d94)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d93)
					ctx.EnsureDesc(&d94)
					var d96 JITValueDesc
					if d94.Loc == LocImm && d93.Loc == LocImm {
						d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d94.Imm.Int() - d93.Imm.Int())}
					} else {
						r7 := ctx.AllocReg()
						if d94.Loc == LocImm {
							ctx.EmitMovRegImm64(r7, uint64(d94.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r7, d94.Reg)
						}
						if d93.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d93.Imm.Int()))
							ctx.EmitSubInt64(r7, RegR11)
						} else {
							ctx.EmitSubInt64(r7, d93.Reg)
						}
						d96 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d96)
					}
					var d97 JITValueDesc
					r8 := ctx.EmitSliceDataAfterLow(&d83, &d93, 1)
					d97 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d97)
					ctx.BindReg(r8, &d97)
					var d98 JITValueDesc
					var r9 Reg
					var r10 Reg
					ctx.SyncDesc(&d97)
					ctx.EnsureDesc(&d97)
					if d97.Loc == LocImm {
						r9 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r9, uint64(d97.Imm.Int()))
					} else {
						r9 = d97.Reg
					}
					ctx.ProtectReg(r9)
					ctx.SyncDesc(&d96)
					ctx.EnsureDesc(&d96)
					if d96.Loc == LocImm {
						r10 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r10, uint64(d96.Imm.Int()))
					} else {
						r10 = d96.Reg
					}
					ctx.ProtectReg(r10)
					ctx.UnprotectReg(r10)
					ctx.UnprotectReg(r9)
					d98 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d98)
					ctx.BindReg(r10, &d98)
					ctx.BindReg(r9, &d98)
					ctx.BindReg(r10, &d98)
					ctx.FreeDesc(&d93)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d84)
					var d99 JITValueDesc
					if d84.Loc == LocImm {
						ctx.TrackImm(d84.Imm)
						ptrWord, _ := d84.Imm.RawWords()
						d99 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d99.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d99.Reg2, uint64(len(d84.Imm.String())))
						ctx.BindReg(d99.Reg, &d99)
						ctx.BindReg(d99.Reg2, &d99)
					} else {
						d99 = d84
					}
					d100 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d98, d99}, 1)
					ctx.EmitAndRegImm32(d100.Reg, 1)
					d100.Type = tagBool
					ctx.BindReg(d100.Reg, &d100)
					ctx.EnsureDesc(&d100)
					ctx.EmitStoreToStack(d100, int32(phiBase85)+int32(0))
					ctx.StabilizeDescForControlFlow(&d100)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl18)
					d101 = JITValueDesc{Loc: LocReg, Reg: r4}
					ctx.BindReg(r4, &d101)
					ctx.BindReg(r4, &d101)
					ctx.ReclaimUntrackedRegs()
					d102 = d101
					ctx.EnsureDesc(&d102)
					if d102.Loc != LocImm && d102.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d102.Loc == LocImm {
						if d102.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl16)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitCmpRegImm32(d102.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl17)
					}
					ctx.FreeDesc(&d101)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d79)
					ctx.EmitCopyDescWords(&d82, &d79, 2)
					ctx.EmitJmp(lbl14)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d103 JITValueDesc
					if d79.SliceSizeKnown {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d79.KnownSliceLen))}
					} else if d79.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d79.Imm.String())))}
					} else if d79.Loc == LocStackTriple {
						d103 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d79.StackOff + 8, NoHeapPointer: true}
					} else if d79.Loc == LocStackPair {
						d103 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d79.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d79)
						if d79.Loc == LocRegPair || d79.Loc == LocRegTriple {
							d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg2, ID: 0}
						} else if d79.Loc == LocReg {
							d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					var d104 JITValueDesc
					if d80.SliceSizeKnown {
						d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d80.KnownSliceLen))}
					} else if d80.Loc == LocImm {
						d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d80.StackOff))}
					} else if d80.Loc == LocStackTriple {
						d104 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d80.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d80)
						if d80.Loc == LocRegPair || d80.Loc == LocRegTriple {
							d104 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d80.Reg2, ID: 0}
						} else if d80.Loc == LocReg {
							d104 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d80.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d104)
					ctx.EnsureDescsTogether(&d103, &d104)
					var d105 JITValueDesc
					if d103.Loc == LocImm && d104.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d103.Imm.Int() - d104.Imm.Int())}
					} else if d104.Loc == LocImm && d104.Imm.Int() == 0 {
						r11 := ctx.AllocRegExcept(d103.Reg)
						ctx.EmitMovRegReg(r11, d103.Reg)
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d105)
					} else if d103.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d104.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
						ctx.EmitSubInt64(scratch, d104.Reg)
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d105)
					} else if d104.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d103.Reg)
						ctx.EmitMovRegReg(scratch, d103.Reg)
						if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d104.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d104.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d105)
					} else {
						r12 := ctx.AllocRegExcept(d103.Reg, d104.Reg)
						ctx.EmitMovRegReg(r12, d103.Reg)
						ctx.EmitSubInt64(r12, d104.Reg)
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
						ctx.BindReg(r12, &d105)
					}
					if d105.Loc == LocReg && d103.Loc == LocReg && d105.Reg == d103.Reg {
						ctx.TransferReg(d103.Reg)
						d103.Loc = LocNone
					}
					ctx.FreeDesc(&d103)
					ctx.FreeDesc(&d104)
					ctx.ReclaimUntrackedRegs()
					d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d105)
					var d108 JITValueDesc
					if d105.Loc == LocImm && d106.Loc == LocImm {
						d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() - d106.Imm.Int())}
					} else {
						r13 := ctx.AllocReg()
						if d105.Loc == LocImm {
							ctx.EmitMovRegImm64(r13, uint64(d105.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r13, d105.Reg)
						}
						if d106.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d106.Imm.Int()))
							ctx.EmitSubInt64(r13, RegR11)
						} else {
							ctx.EmitSubInt64(r13, d106.Reg)
						}
						d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d108)
					}
					var d109 JITValueDesc
					r14 := ctx.EmitSliceDataAfterLow(&d79, &d106, 1)
					d109 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d109)
					ctx.BindReg(r14, &d109)
					var d110 JITValueDesc
					var r15 Reg
					var r16 Reg
					ctx.SyncDesc(&d109)
					ctx.EnsureDesc(&d109)
					if d109.Loc == LocImm {
						r15 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r15, uint64(d109.Imm.Int()))
					} else {
						r15 = d109.Reg
					}
					ctx.ProtectReg(r15)
					ctx.SyncDesc(&d108)
					ctx.EnsureDesc(&d108)
					if d108.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d108.Imm.Int()))
					} else {
						r16 = d108.Reg
					}
					ctx.ProtectReg(r16)
					ctx.UnprotectReg(r16)
					ctx.UnprotectReg(r15)
					d110 = JITValueDesc{Loc: LocRegPair, Reg: r15, Reg2: r16}
					ctx.BindReg(r15, &d110)
					ctx.BindReg(r16, &d110)
					ctx.BindReg(r15, &d110)
					ctx.BindReg(r16, &d110)
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EmitCopyDescWords(&d82, &d110, 2)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl14)
					ctx.FreeDesc(&d78)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d82)
					ctx.EnsureDesc(&d82)
					d111 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d82}, 2)
					ctx.EmitMovPairToResult(&d111, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps112 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps112)
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
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
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode"].Fn, args, result)
				}
				declaration := declarations["json_decode"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d25 JITValueDesc
				_ = d25
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					ctx.StabilizeDescForControlFlow(&d0)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *any) any { return *value }), []JITValueDesc{d0}, 2)
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d24.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d24.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d24)
						} else if d24.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d24)
						} else if d24.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d24)
						} else if d24.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d24.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
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
						panic("jit: generic call arg expects 2-word value (TransformFromJSON arg0)")
					}
					ctx.SyncDesc(&d24)
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(TransformFromJSON), []JITValueDesc{d24}, 2)
					d25.NoHeapPointer = false
					ctx.BindReg(d25.Reg, &d25)
					ctx.BindReg(d25.Reg2, &d25)
					ctx.FreeDesc(&d24)
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
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps26 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps26)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode_scmer"].Fn, args, result)
				}
				declaration := declarations["json_decode_scmer"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					ctx.SyncDesc(&d24)
					if d24.Loc == LocRegPair || d24.Loc == LocStackPair || d24.Loc == LocInputPair {
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_encode"].Fn, args, result)
				}
				declaration := declarations["base64_encode"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_decode"].Fn, args, result)
				}
				declaration := declarations["base64_decode"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				ctx.Coverage.NativeCalls++
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
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["bin2hex"].Fn, args, result)
				}
				declaration := declarations["bin2hex"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d59 JITValueDesc
				_ = d59
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d4 = d2
					ctx.SyncDesc(&d4)
					if d4.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d4.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d4.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d4 = tmpScalar
					}
					d4 = JITPrepareScmerGoArg(ctx, d4)
					if d4.Loc != LocRegPair && d4.Loc != LocStackPair && d4.Loc != LocInputPair {
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
					ctx.EnsureDescsTogether(&d6, &d5)
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
					ctx.ReclaimUntrackedRegs()
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
					ctx.EnsureDescsTogether(&d1, &d13)
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
					ctx.StabilizeDescForControlFlow(&d9)
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d36, &d1)
					var d37 JITValueDesc
					if d36.Loc == LocImm && d1.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d36.Imm.Int() * d1.Imm.Int())}
					} else if d36.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d37)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d36.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d36.Reg, RegR11)
						}
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg}
						ctx.BindReg(d36.Reg, &d37)
					} else {
						ctx.EmitImulInt64(d36.Reg, d1.Reg)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg}
						ctx.BindReg(d36.Reg, &d37)
					}
					if d37.Loc == LocReg && d36.Loc == LocReg && d37.Reg == d36.Reg {
						ctx.TransferReg(d36.Reg)
						d36.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d38 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d38)
					r3 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegMemB(r3, d38.Reg, 0)
					ctx.FreeDesc(&d38)
					d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, NoHeapPointer: true}
					ctx.BindReg(r3, &d39)
					ctx.BindReg(r3, &d39)
					ctx.EnsureDesc(&d39)
					var d40 JITValueDesc
					if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d39.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d39.Reg, 4)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg}
						ctx.BindReg(d39.Reg, &d40)
					}
					if d40.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: d40.Type, Imm: NewInt(int64(uint64(d40.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d40.Reg, 56)
						ctx.EmitShrRegImm8(d40.Reg, 56)
					}
					if d40.Loc == LocReg && d39.Loc == LocReg && d40.Reg == d39.Reg {
						ctx.TransferReg(d39.Reg)
						d39.Loc = LocNone
					}
					ctx.FreeDesc(&d39)
					d41 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d40)
					ctx.EnsureGoStringHeader(&d41)
					d42 = ctx.EmitSliceElementAddress(&d41, &d40, 1)
					ctx.EnsureDesc(&d42)
					r4 := ctx.AllocRegExcept(d42.Reg)
					ctx.EmitMovRegMemB(r4, d42.Reg, 0)
					ctx.FreeDesc(&d42)
					d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4, NoHeapPointer: true}
					ctx.BindReg(r4, &d43)
					ctx.BindReg(r4, &d43)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d37)
					ctx.SyncDesc(&d43)
					ctx.StabilizeDescAcrossNestedCall(&d37)
					d44 = d9
					d44.ID = 0
					d45 = d37
					d45.ID = 0
					d46 = ctx.EmitSliceElementAddress(&d44, &d45, int32(1))
					ctx.FreeDesc(&d45)
					ctx.EmitStoreScmerAt(&d46, &d43)
					ctx.FreeDesc(&d46)
					ctx.FreeDesc(&d37)
					ctx.FreeDesc(&d43)
					d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d47, &d1)
					var d48 JITValueDesc
					if d47.Loc == LocImm && d1.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() * d1.Imm.Int())}
					} else if d47.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d48)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d47.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d47.Reg, RegR11)
						}
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d48)
					} else {
						ctx.EmitImulInt64(d47.Reg, d1.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d48)
					}
					if d48.Loc == LocReg && d47.Loc == LocReg && d48.Reg == d47.Reg {
						ctx.TransferReg(d47.Reg)
						d47.Loc = LocNone
					}
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					var d49 JITValueDesc
					if d48.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d48.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d48.Reg)
						ctx.EmitMovRegReg(scratch, d48.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					}
					if d49.Loc == LocReg && d48.Loc == LocReg && d49.Reg == d48.Reg {
						ctx.TransferReg(d48.Reg)
						d48.Loc = LocNone
					}
					ctx.FreeDesc(&d48)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d50 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d50)
					r5 := ctx.AllocRegExcept(d50.Reg)
					ctx.EmitMovRegMemB(r5, d50.Reg, 0)
					ctx.FreeDesc(&d50)
					d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, NoHeapPointer: true}
					ctx.BindReg(r5, &d51)
					ctx.BindReg(r5, &d51)
					ctx.EnsureDesc(&d51)
					var d52 JITValueDesc
					if d51.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d51.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d51.Reg, 15)
						d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d51.Reg}
						ctx.BindReg(d51.Reg, &d52)
					}
					if d52.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: d52.Type, Imm: NewInt(int64(uint64(d52.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d52.Reg, 56)
						ctx.EmitShrRegImm8(d52.Reg, 56)
					}
					if d52.Loc == LocReg && d51.Loc == LocReg && d52.Reg == d51.Reg {
						ctx.TransferReg(d51.Reg)
						d51.Loc = LocNone
					}
					ctx.FreeDesc(&d51)
					d53 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d52)
					ctx.EnsureGoStringHeader(&d53)
					d54 = ctx.EmitSliceElementAddress(&d53, &d52, 1)
					ctx.EnsureDesc(&d54)
					r6 := ctx.AllocRegExcept(d54.Reg)
					ctx.EmitMovRegMemB(r6, d54.Reg, 0)
					ctx.FreeDesc(&d54)
					d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d55)
					ctx.BindReg(r6, &d55)
					ctx.FreeDesc(&d52)
					ctx.EnsureDesc(&d49)
					ctx.SyncDesc(&d55)
					ctx.StabilizeDescAcrossNestedCall(&d49)
					d56 = d9
					d56.ID = 0
					d57 = d49
					d57.ID = 0
					d58 = ctx.EmitSliceElementAddress(&d56, &d57, int32(1))
					ctx.FreeDesc(&d57)
					ctx.EmitStoreScmerAt(&d58, &d55)
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d49)
					ctx.FreeDesc(&d55)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d59 JITValueDesc
					if d1.Loc == LocImm {
						d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d59)
					}
					if d59.Loc == LocReg && d1.Loc == LocReg && d59.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d59)
					ctx.FreeDesc(&d1)
					if ps.General {
					}
					ps60 := PhiState{General: ps.General}
					ps60.OverlayValues = make([]JITValueDesc, 60)
					ps60.OverlayValues[1] = d1
					ps60.OverlayValues[2] = d2
					ps60.OverlayValues[3] = d3
					ps60.OverlayValues[4] = d4
					ps60.OverlayValues[5] = d5
					ps60.OverlayValues[6] = d6
					ps60.OverlayValues[7] = d7
					ps60.OverlayValues[9] = d9
					ps60.OverlayValues[11] = d11
					ps60.OverlayValues[12] = d12
					ps60.OverlayValues[13] = d13
					ps60.OverlayValues[14] = d14
					ps60.OverlayValues[15] = d15
					ps60.OverlayValues[18] = d18
					ps60.OverlayValues[36] = d36
					ps60.OverlayValues[37] = d37
					ps60.OverlayValues[38] = d38
					ps60.OverlayValues[39] = d39
					ps60.OverlayValues[40] = d40
					ps60.OverlayValues[41] = d41
					ps60.OverlayValues[42] = d42
					ps60.OverlayValues[43] = d43
					ps60.OverlayValues[44] = d44
					ps60.OverlayValues[45] = d45
					ps60.OverlayValues[46] = d46
					ps60.OverlayValues[47] = d47
					ps60.OverlayValues[48] = d48
					ps60.OverlayValues[49] = d49
					ps60.OverlayValues[50] = d50
					ps60.OverlayValues[51] = d51
					ps60.OverlayValues[52] = d52
					ps60.OverlayValues[53] = d53
					ps60.OverlayValues[54] = d54
					ps60.OverlayValues[55] = d55
					ps60.OverlayValues[56] = d56
					ps60.OverlayValues[57] = d57
					ps60.OverlayValues[58] = d58
					ps60.OverlayValues[59] = d59
					ps60.PhiValues = make([]JITValueDesc, 1)
					if ps60.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps60)
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					callResults62 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d9}, []uint8{2}, []uint8{1})
					d61 = callResults62[0]
					ctx.EnsureDesc(&d61)
					d63 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d61}, 2)
					ctx.EmitMovPairToResult(&d63, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps64 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps64)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["bin2hex"].Fn, args, result)
				}
				declaration := declarations["bin2hex"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d59 JITValueDesc
				_ = d59
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d4 = d2
					ctx.SyncDesc(&d4)
					if d4.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d4.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d4.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d4 = tmpScalar
					}
					d4 = JITPrepareScmerGoArg(ctx, d4)
					if d4.Loc != LocRegPair && d4.Loc != LocStackPair && d4.Loc != LocInputPair {
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
					ctx.EnsureDescsTogether(&d6, &d5)
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
					ctx.ReclaimUntrackedRegs()
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
					ctx.EnsureDescsTogether(&d1, &d13)
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
					ctx.StabilizeDescForControlFlow(&d9)
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d36, &d1)
					var d37 JITValueDesc
					if d36.Loc == LocImm && d1.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d36.Imm.Int() * d1.Imm.Int())}
					} else if d36.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d37)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d36.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d36.Reg, RegR11)
						}
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg}
						ctx.BindReg(d36.Reg, &d37)
					} else {
						ctx.EmitImulInt64(d36.Reg, d1.Reg)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg}
						ctx.BindReg(d36.Reg, &d37)
					}
					if d37.Loc == LocReg && d36.Loc == LocReg && d37.Reg == d36.Reg {
						ctx.TransferReg(d36.Reg)
						d36.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d38 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d38)
					r3 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegMemB(r3, d38.Reg, 0)
					ctx.FreeDesc(&d38)
					d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, NoHeapPointer: true}
					ctx.BindReg(r3, &d39)
					ctx.BindReg(r3, &d39)
					ctx.EnsureDesc(&d39)
					var d40 JITValueDesc
					if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d39.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d39.Reg, 4)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg}
						ctx.BindReg(d39.Reg, &d40)
					}
					if d40.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: d40.Type, Imm: NewInt(int64(uint64(d40.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d40.Reg, 56)
						ctx.EmitShrRegImm8(d40.Reg, 56)
					}
					if d40.Loc == LocReg && d39.Loc == LocReg && d40.Reg == d39.Reg {
						ctx.TransferReg(d39.Reg)
						d39.Loc = LocNone
					}
					ctx.FreeDesc(&d39)
					d41 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d40)
					ctx.EnsureGoStringHeader(&d41)
					d42 = ctx.EmitSliceElementAddress(&d41, &d40, 1)
					ctx.EnsureDesc(&d42)
					r4 := ctx.AllocRegExcept(d42.Reg)
					ctx.EmitMovRegMemB(r4, d42.Reg, 0)
					ctx.FreeDesc(&d42)
					d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4, NoHeapPointer: true}
					ctx.BindReg(r4, &d43)
					ctx.BindReg(r4, &d43)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d37)
					ctx.SyncDesc(&d43)
					ctx.StabilizeDescAcrossNestedCall(&d37)
					d44 = d9
					d44.ID = 0
					d45 = d37
					d45.ID = 0
					d46 = ctx.EmitSliceElementAddress(&d44, &d45, int32(1))
					ctx.FreeDesc(&d45)
					ctx.EmitStoreScmerAt(&d46, &d43)
					ctx.FreeDesc(&d46)
					ctx.FreeDesc(&d37)
					ctx.FreeDesc(&d43)
					d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDescsTogether(&d47, &d1)
					var d48 JITValueDesc
					if d47.Loc == LocImm && d1.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() * d1.Imm.Int())}
					} else if d47.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
						ctx.EmitImulInt64(scratch, d1.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d48)
					} else if d1.Loc == LocImm {
						if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
							ctx.EmitImulRegImm32(d47.Reg, int32(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
							ctx.EmitImulInt64(d47.Reg, RegR11)
						}
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d48)
					} else {
						ctx.EmitImulInt64(d47.Reg, d1.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d48)
					}
					if d48.Loc == LocReg && d47.Loc == LocReg && d48.Reg == d47.Reg {
						ctx.TransferReg(d47.Reg)
						d47.Loc = LocNone
					}
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					var d49 JITValueDesc
					if d48.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d48.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d48.Reg)
						ctx.EmitMovRegReg(scratch, d48.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					}
					if d49.Loc == LocReg && d48.Loc == LocReg && d49.Reg == d48.Reg {
						ctx.TransferReg(d48.Reg)
						d48.Loc = LocNone
					}
					ctx.FreeDesc(&d48)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d3)
					d50 = ctx.EmitSliceElementAddress(&d3, &d1, 1)
					ctx.EnsureDesc(&d50)
					r5 := ctx.AllocRegExcept(d50.Reg)
					ctx.EmitMovRegMemB(r5, d50.Reg, 0)
					ctx.FreeDesc(&d50)
					d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, NoHeapPointer: true}
					ctx.BindReg(r5, &d51)
					ctx.BindReg(r5, &d51)
					ctx.EnsureDesc(&d51)
					var d52 JITValueDesc
					if d51.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d51.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d51.Reg, 15)
						d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d51.Reg}
						ctx.BindReg(d51.Reg, &d52)
					}
					if d52.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: d52.Type, Imm: NewInt(int64(uint64(d52.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d52.Reg, 56)
						ctx.EmitShrRegImm8(d52.Reg, 56)
					}
					if d52.Loc == LocReg && d51.Loc == LocReg && d52.Reg == d51.Reg {
						ctx.TransferReg(d51.Reg)
						d51.Loc = LocNone
					}
					ctx.FreeDesc(&d51)
					d53 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d52)
					ctx.EnsureGoStringHeader(&d53)
					d54 = ctx.EmitSliceElementAddress(&d53, &d52, 1)
					ctx.EnsureDesc(&d54)
					r6 := ctx.AllocRegExcept(d54.Reg)
					ctx.EmitMovRegMemB(r6, d54.Reg, 0)
					ctx.FreeDesc(&d54)
					d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d55)
					ctx.BindReg(r6, &d55)
					ctx.FreeDesc(&d52)
					ctx.EnsureDesc(&d49)
					ctx.SyncDesc(&d55)
					ctx.StabilizeDescAcrossNestedCall(&d49)
					d56 = d9
					d56.ID = 0
					d57 = d49
					d57.ID = 0
					d58 = ctx.EmitSliceElementAddress(&d56, &d57, int32(1))
					ctx.FreeDesc(&d57)
					ctx.EmitStoreScmerAt(&d58, &d55)
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d49)
					ctx.FreeDesc(&d55)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d59 JITValueDesc
					if d1.Loc == LocImm {
						d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d59)
					}
					if d59.Loc == LocReg && d1.Loc == LocReg && d59.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d59)
					ctx.FreeDesc(&d1)
					if ps.General {
					}
					ps60 := PhiState{General: ps.General}
					ps60.OverlayValues = make([]JITValueDesc, 60)
					ps60.OverlayValues[1] = d1
					ps60.OverlayValues[2] = d2
					ps60.OverlayValues[3] = d3
					ps60.OverlayValues[4] = d4
					ps60.OverlayValues[5] = d5
					ps60.OverlayValues[6] = d6
					ps60.OverlayValues[7] = d7
					ps60.OverlayValues[9] = d9
					ps60.OverlayValues[11] = d11
					ps60.OverlayValues[12] = d12
					ps60.OverlayValues[13] = d13
					ps60.OverlayValues[14] = d14
					ps60.OverlayValues[15] = d15
					ps60.OverlayValues[18] = d18
					ps60.OverlayValues[36] = d36
					ps60.OverlayValues[37] = d37
					ps60.OverlayValues[38] = d38
					ps60.OverlayValues[39] = d39
					ps60.OverlayValues[40] = d40
					ps60.OverlayValues[41] = d41
					ps60.OverlayValues[42] = d42
					ps60.OverlayValues[43] = d43
					ps60.OverlayValues[44] = d44
					ps60.OverlayValues[45] = d45
					ps60.OverlayValues[46] = d46
					ps60.OverlayValues[47] = d47
					ps60.OverlayValues[48] = d48
					ps60.OverlayValues[49] = d49
					ps60.OverlayValues[50] = d50
					ps60.OverlayValues[51] = d51
					ps60.OverlayValues[52] = d52
					ps60.OverlayValues[53] = d53
					ps60.OverlayValues[54] = d54
					ps60.OverlayValues[55] = d55
					ps60.OverlayValues[56] = d56
					ps60.OverlayValues[57] = d57
					ps60.OverlayValues[58] = d58
					ps60.OverlayValues[59] = d59
					ps60.PhiValues = make([]JITValueDesc, 1)
					if ps60.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps60)
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
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					callResults62 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d9}, []uint8{2}, []uint8{1})
					d61 = callResults62[0]
					ctx.EnsureDesc(&d61)
					d63 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d61}, 2)
					ctx.EmitMovPairToResult(&d63, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps64 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps64)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["hex2bin"].Fn, args, result)
				}
				declaration := declarations["hex2bin"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["uuid"].Fn, args, result)
				}
				declaration := declarations["uuid"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
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
					d14 = ctx.EmitGoCallScalar(GoFuncAddr((uuid.UUID).String), []JITValueDesc{d1}, 2)
					d14.NoHeapPointer = false
					ctx.BindReg(d14.Reg, &d14)
					ctx.BindReg(d14.Reg2, &d14)
					ctx.EnsureDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d14}, 2)
					ctx.EmitMovPairToResult(&d15, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps16 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps16)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
				}
				declaration := declarations["randomBytes"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d58 JITValueDesc
				_ = d58
				var d60 JITValueDesc
				_ = d60
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [6]BBDescriptor
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
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl9)
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
					ctx.ReclaimUntrackedRegs()
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
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl11)
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
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocRegTriple && d18.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (crand.Read arg0)")
					}
					ctx.SyncDesc(&d18)
					callResults35 := JITEmitGoCallResults(ctx, GoFuncAddr(crand.Read), []JITValueDesc{d18}, []uint8{1, 2}, []uint8{0, 3})
					d36 = callResults35[0]
					_ = d36
					d37 = callResults35[1]
					_ = d37
					ctx.StabilizeDescForControlFlow(&d37)
					ctx.EnsureDesc(&d37)
					var d38 JITValueDesc
					if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d37)
						if d37.Loc != LocReg && d37.Loc != LocRegPair && d37.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitCmpRegImm32(d37.Reg, 0)
						ctx.EmitSetcc(r2, CondNotEqual)
						d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d38)
					}
					d39 = d38
					ctx.EnsureDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							if ps.General {
							}
							ps40 := PhiState{General: ps.General}
							ps40.OverlayValues = make([]JITValueDesc, 40)
							ps40.OverlayValues[0] = d0
							ps40.OverlayValues[1] = d1
							ps40.OverlayValues[2] = d2
							ps40.OverlayValues[3] = d3
							ps40.OverlayValues[4] = d4
							ps40.OverlayValues[5] = d5
							ps40.OverlayValues[18] = d18
							ps40.OverlayValues[19] = d19
							ps40.OverlayValues[20] = d20
							ps40.OverlayValues[36] = d36
							ps40.OverlayValues[37] = d37
							ps40.OverlayValues[38] = d38
							ps40.OverlayValues[39] = d39
							return bbs[5].RenderPS(ps40)
						}
						if ps.General {
						}
						ps41 := PhiState{General: ps.General}
						ps41.OverlayValues = make([]JITValueDesc, 40)
						ps41.OverlayValues[0] = d0
						ps41.OverlayValues[1] = d1
						ps41.OverlayValues[2] = d2
						ps41.OverlayValues[3] = d3
						ps41.OverlayValues[4] = d4
						ps41.OverlayValues[5] = d5
						ps41.OverlayValues[18] = d18
						ps41.OverlayValues[19] = d19
						ps41.OverlayValues[20] = d20
						ps41.OverlayValues[36] = d36
						ps41.OverlayValues[37] = d37
						ps41.OverlayValues[38] = d38
						ps41.OverlayValues[39] = d39
						return bbs[4].RenderPS(ps41)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d39.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl5)
					ps42 := PhiState{General: true}
					ps42.OverlayValues = make([]JITValueDesc, 40)
					ps42.OverlayValues[0] = d0
					ps42.OverlayValues[1] = d1
					ps42.OverlayValues[2] = d2
					ps42.OverlayValues[3] = d3
					ps42.OverlayValues[4] = d4
					ps42.OverlayValues[5] = d5
					ps42.OverlayValues[18] = d18
					ps42.OverlayValues[19] = d19
					ps42.OverlayValues[20] = d20
					ps42.OverlayValues[36] = d36
					ps42.OverlayValues[37] = d37
					ps42.OverlayValues[38] = d38
					ps42.OverlayValues[39] = d39
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 40)
					ps43.OverlayValues[0] = d0
					ps43.OverlayValues[1] = d1
					ps43.OverlayValues[2] = d2
					ps43.OverlayValues[3] = d3
					ps43.OverlayValues[4] = d4
					ps43.OverlayValues[5] = d5
					ps43.OverlayValues[18] = d18
					ps43.OverlayValues[19] = d19
					ps43.OverlayValues[20] = d20
					ps43.OverlayValues[36] = d36
					ps43.OverlayValues[37] = d37
					ps43.OverlayValues[38] = d38
					ps43.OverlayValues[39] = d39
					snap44 := d0
					snap45 := d1
					snap46 := d2
					snap47 := d3
					snap48 := d4
					snap49 := d5
					snap50 := d18
					snap51 := d19
					snap52 := d20
					snap53 := d36
					snap54 := d37
					snap55 := d38
					snap56 := d39
					alloc57 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps43)
					}
					ctx.RestoreAllocState(alloc57)
					d0 = snap44
					d1 = snap45
					d2 = snap46
					d3 = snap47
					d4 = snap48
					d5 = snap49
					d18 = snap50
					d19 = snap51
					d20 = snap52
					d36 = snap53
					d37 = snap54
					d38 = snap55
					d39 = snap56
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps42)
					}
					return result
					ctx.FreeDesc(&d38)
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
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					callResults59 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d18}, []uint8{2}, []uint8{1})
					d58 = callResults59[0]
					ctx.EnsureDesc(&d58)
					d60 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d58}, 2)
					ctx.EmitMovPairToResult(&d60, &result)
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
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps61 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps61)
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
			return NewString(re.ReplaceAllString(String(a[0]), String(a[2])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "replaces matches of a regex pattern in a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string"}, &TypeDescriptor{Kind: "string", Label: "pattern", Description: "regex pattern"}, &TypeDescriptor{Kind: "string", Label: "replacement", Description: "replacement string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_replace"].Fn, args, result)
				}
				declaration := declarations["regexp_replace"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
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
					ctx.SyncDesc(&d41)
					if d41.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d41.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d41.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d41 = tmpScalar
					}
					d41 = JITPrepareScmerGoArg(ctx, d41)
					if d41.Loc != LocRegPair && d41.Loc != LocStackPair && d41.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d40 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d41}, 2)
					ctx.FreeDesc(&d39)
					d42 = args[2]
					d42.ID = 0
					d44 = d42
					ctx.SyncDesc(&d44)
					if d44.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d44.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d44.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d44 = tmpScalar
					}
					d44 = JITPrepareScmerGoArg(ctx, d44)
					if d44.Loc != LocRegPair && d44.Loc != LocStackPair && d44.Loc != LocInputPair {
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
					if d40.Loc != LocRegPair && d40.Loc != LocStackPair && d40.Loc != LocInputPair {
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
					if d43.Loc != LocRegPair && d43.Loc != LocStackPair && d43.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).ReplaceAllString arg2)")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d40)
					ctx.SyncDesc(&d43)
					d45 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).ReplaceAllString), []JITValueDesc{d18, d40, d43}, 2)
					d45.NoHeapPointer = false
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
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["fnv_hash"].Fn, args, result)
				}
				declaration := declarations["fnv_hash"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["stable_structural_hash"].Fn, args, result)
				}
				declaration := declarations["stable_structural_hash"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
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
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [8]BBDescriptor
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
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
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
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl13)
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
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d31.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl15)
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
					ctx.StabilizeDescForControlFlow(&d11)
					d48 = args[0]
					d48.ID = 0
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					d48 = JITPrepareScmerGoArg(ctx, d48)
					d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d49.Loc == LocRegPair || d49.Loc == LocStackPair || d49.Loc == LocRegTriple || d49.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d50.Loc == LocRegPair || d50.Loc == LocStackPair || d50.Loc == LocRegTriple || d50.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d51 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocRegTriple || d51.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d48)
					ctx.SyncDesc(&d49)
					ctx.SyncDesc(&d50)
					ctx.SyncDesc(&d51)
					ctx.EmitGoCallVoid(GoFuncAddr(serializeEx), []JITValueDesc{d11, d48, d49, d50, d51})
					ctx.FreeDesc(&d51)
					ctx.FreeDesc(&d48)
					if ps.General {
					}
					ps52 := PhiState{General: ps.General}
					ps52.OverlayValues = make([]JITValueDesc, 52)
					ps52.OverlayValues[0] = d0
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[11] = d11
					ps52.OverlayValues[12] = d12
					ps52.OverlayValues[13] = d13
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[29] = d29
					ps52.OverlayValues[30] = d30
					ps52.OverlayValues[31] = d31
					ps52.OverlayValues[48] = d48
					ps52.OverlayValues[49] = d49
					ps52.OverlayValues[50] = d50
					ps52.OverlayValues[51] = d51
					if ps52.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps52)
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
					ctx.StabilizeDescForControlFlow(&d11)
					var d53 JITValueDesc
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						fieldAddr := uintptr(d11.Imm.Int()) + 8
						r3 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r3, fieldAddr)
						d53 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d53)
					} else {
						off := int32(8)
						baseReg := d11.Reg
						r4 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r4, baseReg, off)
						d53 = JITValueDesc{Loc: LocReg, Reg: r4}
						ctx.BindReg(r4, &d53)
					}
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d53)
					if d53.Loc == LocRegPair || d53.Loc == LocStackPair || d53.Loc == LocRegTriple || d53.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d53)
					d54 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d53}, 2)
					d54.NoHeapPointer = false
					ctx.BindReg(d54.Reg, &d54)
					ctx.BindReg(d54.Reg2, &d54)
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d54)
					d55 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d54}, 2)
					ctx.EmitMovPairToResult(&d55, &result)
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
					ctx.StabilizeDescForControlFlow(&d11)
					d56 = args[0]
					d56.ID = 0
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					d56 = JITPrepareScmerGoArg(ctx, d56)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d56)
					ctx.EmitGoCallVoid(GoFuncAddr(WriteStringValue), []JITValueDesc{d11, d56})
					ctx.FreeDesc(&d56)
					if ps.General {
					}
					ps57 := PhiState{General: ps.General}
					ps57.OverlayValues = make([]JITValueDesc, 57)
					ps57.OverlayValues[0] = d0
					ps57.OverlayValues[1] = d1
					ps57.OverlayValues[2] = d2
					ps57.OverlayValues[11] = d11
					ps57.OverlayValues[12] = d12
					ps57.OverlayValues[13] = d13
					ps57.OverlayValues[14] = d14
					ps57.OverlayValues[15] = d15
					ps57.OverlayValues[29] = d29
					ps57.OverlayValues[30] = d30
					ps57.OverlayValues[31] = d31
					ps57.OverlayValues[48] = d48
					ps57.OverlayValues[49] = d49
					ps57.OverlayValues[50] = d50
					ps57.OverlayValues[51] = d51
					ps57.OverlayValues[53] = d53
					ps57.OverlayValues[54] = d54
					ps57.OverlayValues[55] = d55
					ps57.OverlayValues[56] = d56
					if ps57.General && bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					return bbs[5].RenderPS(ps57)
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
					d58 = args[1]
					d58.ID = 0
					d60 = d58
					d60.ID = 0
					d59 = ctx.EmitBoolDesc(&d60, JITValueDesc{Loc: LocAny})
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
							ps62.OverlayValues[11] = d11
							ps62.OverlayValues[12] = d12
							ps62.OverlayValues[13] = d13
							ps62.OverlayValues[14] = d14
							ps62.OverlayValues[15] = d15
							ps62.OverlayValues[29] = d29
							ps62.OverlayValues[30] = d30
							ps62.OverlayValues[31] = d31
							ps62.OverlayValues[48] = d48
							ps62.OverlayValues[49] = d49
							ps62.OverlayValues[50] = d50
							ps62.OverlayValues[51] = d51
							ps62.OverlayValues[53] = d53
							ps62.OverlayValues[54] = d54
							ps62.OverlayValues[55] = d55
							ps62.OverlayValues[56] = d56
							ps62.OverlayValues[58] = d58
							ps62.OverlayValues[59] = d59
							ps62.OverlayValues[60] = d60
							ps62.OverlayValues[61] = d61
							return bbs[4].RenderPS(ps62)
						}
						if ps.General {
						}
						ps63 := PhiState{General: ps.General}
						ps63.OverlayValues = make([]JITValueDesc, 62)
						ps63.OverlayValues[0] = d0
						ps63.OverlayValues[1] = d1
						ps63.OverlayValues[2] = d2
						ps63.OverlayValues[11] = d11
						ps63.OverlayValues[12] = d12
						ps63.OverlayValues[13] = d13
						ps63.OverlayValues[14] = d14
						ps63.OverlayValues[15] = d15
						ps63.OverlayValues[29] = d29
						ps63.OverlayValues[30] = d30
						ps63.OverlayValues[31] = d31
						ps63.OverlayValues[48] = d48
						ps63.OverlayValues[49] = d49
						ps63.OverlayValues[50] = d50
						ps63.OverlayValues[51] = d51
						ps63.OverlayValues[53] = d53
						ps63.OverlayValues[54] = d54
						ps63.OverlayValues[55] = d55
						ps63.OverlayValues[56] = d56
						ps63.OverlayValues[58] = d58
						ps63.OverlayValues[59] = d59
						ps63.OverlayValues[60] = d60
						ps63.OverlayValues[61] = d61
						return bbs[6].RenderPS(ps63)
					}
					if !ps.General {
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d61.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl7)
					ps64 := PhiState{General: true}
					ps64.OverlayValues = make([]JITValueDesc, 62)
					ps64.OverlayValues[0] = d0
					ps64.OverlayValues[1] = d1
					ps64.OverlayValues[2] = d2
					ps64.OverlayValues[11] = d11
					ps64.OverlayValues[12] = d12
					ps64.OverlayValues[13] = d13
					ps64.OverlayValues[14] = d14
					ps64.OverlayValues[15] = d15
					ps64.OverlayValues[29] = d29
					ps64.OverlayValues[30] = d30
					ps64.OverlayValues[31] = d31
					ps64.OverlayValues[48] = d48
					ps64.OverlayValues[49] = d49
					ps64.OverlayValues[50] = d50
					ps64.OverlayValues[51] = d51
					ps64.OverlayValues[53] = d53
					ps64.OverlayValues[54] = d54
					ps64.OverlayValues[55] = d55
					ps64.OverlayValues[56] = d56
					ps64.OverlayValues[58] = d58
					ps64.OverlayValues[59] = d59
					ps64.OverlayValues[60] = d60
					ps64.OverlayValues[61] = d61
					ps65 := PhiState{General: true}
					ps65.OverlayValues = make([]JITValueDesc, 62)
					ps65.OverlayValues[0] = d0
					ps65.OverlayValues[1] = d1
					ps65.OverlayValues[2] = d2
					ps65.OverlayValues[11] = d11
					ps65.OverlayValues[12] = d12
					ps65.OverlayValues[13] = d13
					ps65.OverlayValues[14] = d14
					ps65.OverlayValues[15] = d15
					ps65.OverlayValues[29] = d29
					ps65.OverlayValues[30] = d30
					ps65.OverlayValues[31] = d31
					ps65.OverlayValues[48] = d48
					ps65.OverlayValues[49] = d49
					ps65.OverlayValues[50] = d50
					ps65.OverlayValues[51] = d51
					ps65.OverlayValues[53] = d53
					ps65.OverlayValues[54] = d54
					ps65.OverlayValues[55] = d55
					ps65.OverlayValues[56] = d56
					ps65.OverlayValues[58] = d58
					ps65.OverlayValues[59] = d59
					ps65.OverlayValues[60] = d60
					ps65.OverlayValues[61] = d61
					snap66 := d0
					snap67 := d1
					snap68 := d2
					snap69 := d11
					snap70 := d12
					snap71 := d13
					snap72 := d14
					snap73 := d15
					snap74 := d29
					snap75 := d30
					snap76 := d31
					snap77 := d48
					snap78 := d49
					snap79 := d50
					snap80 := d51
					snap81 := d53
					snap82 := d54
					snap83 := d55
					snap84 := d56
					snap85 := d58
					snap86 := d59
					snap87 := d60
					snap88 := d61
					alloc89 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps65)
					}
					ctx.RestoreAllocState(alloc89)
					d0 = snap66
					d1 = snap67
					d2 = snap68
					d11 = snap69
					d12 = snap70
					d13 = snap71
					d14 = snap72
					d15 = snap73
					d29 = snap74
					d30 = snap75
					d31 = snap76
					d48 = snap77
					d49 = snap78
					d50 = snap79
					d51 = snap80
					d53 = snap81
					d54 = snap82
					d55 = snap83
					d56 = snap84
					d58 = snap85
					d59 = snap86
					d60 = snap87
					d61 = snap88
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps64)
					}
					return result
					ctx.FreeDesc(&d59)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps90 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps90)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sha1"].Fn, args, result)
				}
				declaration := declarations["sha1"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sha256"].Fn, args, result)
				}
				declaration := declarations["sha256"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_test"].Fn, args, result)
				}
				declaration := declarations["regexp_test"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
					d16 = d14
					ctx.SyncDesc(&d16)
					if d16.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d16.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d16.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d16 = tmpScalar
					}
					d16 = JITPrepareScmerGoArg(ctx, d16)
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
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
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
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
					ctx.SyncDesc(&d66)
					if d66.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d66.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d66.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d66 = tmpScalar
					}
					d66 = JITPrepareScmerGoArg(ctx, d66)
					if d66.Loc != LocRegPair && d66.Loc != LocStackPair && d66.Loc != LocInputPair {
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
					if d65.Loc != LocRegPair && d65.Loc != LocStackPair && d65.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).MatchString arg1)")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d65)
					d67 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).MatchString), []JITValueDesc{d18, d65}, 1)
					d67.NoHeapPointer = true
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
	// Keep a declared callable identity in the optimized AST so both Eval and
	// the JIT can execute the same precompiled-regex operation directly.
	return NewSlice([]Scmer{
		NewSymbol(jitConstantRegexpTestName),
		NewRegex(re),
		rv[1],
	}), td
}
