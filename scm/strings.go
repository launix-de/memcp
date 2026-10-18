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
import "sync"
import "bytes"
import "regexp"
import "unsafe"
import "net/url"
import "strings"
import "unicode"
import "crypto/md5"
import "crypto/sha1"
import crand "crypto/rand"
import "encoding/hex"
import "unicode/utf8"
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

type CollationKeyFunc func(Scmer) (string, bool)

type collationKeyDescriptor struct {
	Key     CollationKeyFunc
	Reverse bool
}

// collateKeyRegistry exposes reusable Unicode sort keys to storage index
// builders. It uses the same stable metadata as persisted indexes, because an
// index may outlive the callback closure that originally supplied its order.
// Sorting compares each value O(log N) times; producing its key once avoids
// repeating the collator's temporary allocations in every comparison.
var collateKeyRegistry sync.Map // map[string]collationKeyDescriptor

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

// generalCIFoldCompare preserves strings.ToLower ordering without creating
// lowercase copies. The common ASCII prefix stays on the byte comparison path.
func generalCIFoldCompare(left, right string) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] >= 0x80 || right[i] >= 0x80 {
			return generalCIUnicodeCompare(left[i:], right[i:])
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

// UTF-8 byte ordering agrees with rune ordering. Decode invalid bytes as
// RuneError, exactly as strings.ToLower does, and fold one rune at a time.
// Unicode simple lowercase mappings never expand into multiple runes.
func generalCIUnicodeCompare(left, right string) int {
	li, ri := 0, 0
	for li < len(left) && ri < len(right) {
		l, r := rune(left[li]), rune(right[ri])
		li++
		ri++
		if l >= utf8.RuneSelf {
			decoded, width := utf8.DecodeRuneInString(left[li-1:])
			li += width - 1
			l = unicode.ToLower(decoded)
		} else {
			l = rune(asciiFoldByte(byte(l)))
		}
		if r >= utf8.RuneSelf {
			decoded, width := utf8.DecodeRuneInString(right[ri-1:])
			ri += width - 1
			r = unicode.ToLower(decoded)
		} else {
			r = rune(asciiFoldByte(byte(r)))
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	}
	if li < len(left) {
		return 1
	}
	if ri < len(right) {
		return -1
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

// LookupCollationKey returns a key producer for canonical Unicode collation
// metadata. Binary and custom general collations already compare without
// allocating and intentionally stay on their direct comparator paths.
func LookupCollationKey(meta string) (CollationKeyFunc, bool, bool) {
	if meta == "" {
		return nil, false, false
	}
	if value, ok := collateKeyRegistry.Load(meta); ok {
		descriptor := value.(collationKeyDescriptor)
		return descriptor.Key, descriptor.Reverse, true
	}
	return nil, false, false
}

// binaryCollationLess keeps boolean/text keys in the same textual order in
// both directions. Less is the coercing expression comparator: its bool-left
// arm converts to integers, while its string-left arm renders a bool as text.
// That asymmetry must not enter an index (true < "3" and "3" < true).
// Keep expression comparison itself unchanged, including numeric coercions.
func binaryCollationLess(a, b Scmer) bool {
	if (a.IsBool() && (b.IsString() || b.IsSymbol())) ||
		(b.IsBool() && (a.IsString() || a.IsSymbol())) {
		return String(a) < String(b)
	}
	return Less(a, b)
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

// utf16Length counts code units without constructing an encoded buffer.
func utf16Length(text string) int64 {
	var units int64
	for _, scalar := range text {
		units++
		if scalar > 0xffff {
			units++
		}
	}
	return units
}

// utf16Prefix keeps whole Unicode scalar values. A limit inside a surrogate
// pair stops before that scalar; the generic operation never emits half a pair.
func utf16Prefix(text string, limit int64) string {
	if limit < 0 {
		panic("UTF16 prefix limit must be nonnegative")
	}
	var units int64
	for offset, scalar := range text {
		width := int64(1)
		if scalar > 0xffff {
			width = 2
		}
		if width > limit-units {
			return text[:offset]
		}
		units += width
	}
	return text
}

func init_strings() {
	// Encode one scalar value independently of any frontend character type or
	// collation. Frontends select their own ranges before calling this operator.
	Declare(&Globalenv, &Declaration{
		Name: "unicode_char",
		Fn: func(a ...Scmer) Scmer {
			code := a[0].Int()
			if code < 0 || code > utf8.MaxRune || code >= 0xd800 && code <= 0xdfff {
				panic("invalid Unicode scalar value")
			}
			return NewString(string(rune(code)))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "encodes one Unicode scalar value as UTF-8",
			Params: []*TypeDescriptor{{Kind: "int", Label: "code", Description: "Unicode scalar value"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				declaration := declarations["unicode_char"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	initExpressionMetadata()
	Declare(&Globalenv, &Declaration{
		Name: "regexp_matches",
		Fn: func(a ...Scmer) Scmer {
			re, err := regexp.Compile(String(a[1]))
			if err != nil {
				panic("regexp_matches: invalid pattern: " + err.Error())
			}
			matches := re.FindAllString(String(a[0]), -1)
			result := make([]Scmer, len(matches))
			for i, match := range matches {
				result[i] = NewString(match)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns non-overlapping regular expression matches in input order",
			Params:             []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "string", Label: "pattern"}},
			Return:             &TypeDescriptor{Kind: "list"},
			Const:              true,
			JITInlineCost:      32,
			JITInlineCallbacks: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["regexp_matches"]
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
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[1]
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
					ctx.FreeDesc(&d2)
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
						panic("jit: generic call arg expects 2-word value (regexp.Compile arg0)")
					}
					ctx.SyncDesc(&d3)
					callResults5 := JITEmitGoCallResults(ctx, GoFuncAddr(regexp.Compile), []JITValueDesc{d3}, []uint8{1, 2}, []uint8{1, 3})
					d6 = callResults5[0]
					_ = d6
					d7 = callResults5[1]
					_ = d7
					ctx.StabilizeDescForControlFlow(&d6)
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.EnsureDesc(&d7)
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap10 := d1
						snap11 := d2
						snap12 := d3
						snap13 := d4
						snap14 := d6
						snap15 := d7
						snap16 := d8
						snap17 := d9
						alloc18 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc18)
						d1 = snap10
						d2 = snap11
						d3 = snap12
						d4 = snap13
						d6 = snap14
						d7 = snap15
						d8 = snap16
						d9 = snap17
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d8)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_matches"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					if d6.Loc == LocRegPair || d6.Loc == LocStackPair || d6.Loc == LocRegTriple || d6.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
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
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).FindAllString arg1)")
					}
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocRegTriple || d22.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d6)
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d22)
					d23 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).FindAllString), []JITValueDesc{d6, d20, d22}, 3)
					d23.NoHeapPointer = false
					ctx.BindReg(d23.Reg, &d23)
					ctx.BindReg(d23.Reg2, &d23)
					ctx.BindReg(d23.Reg3, &d23)
					ctx.FreeDesc(&d22)
					ctx.StabilizeDescForControlFlow(&d23)
					if d23.SliceSizeKnown {
						d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d23.KnownSliceLen))}
					} else if d23.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d23.StackOff))}
					} else if d23.Loc == LocStackTriple {
						d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d23.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d23)
						if d23.Loc == LocRegPair || d23.Loc == LocRegTriple {
							d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg2, ID: 0}
						} else if d23.Loc == LocReg {
							d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d24)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d24)
					callResults25 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d24, d24}, []uint8{3}, []uint8{1})
					d26 = callResults25[0]
					d26.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d26)
					ctx.FreeDesc(&d24)
					if d23.SliceSizeKnown {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d23.KnownSliceLen))}
					} else if d23.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d23.StackOff))}
					} else if d23.Loc == LocStackTriple {
						d27 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d23.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d23)
						if d23.Loc == LocRegPair || d23.Loc == LocRegTriple {
							d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg2, ID: 0}
						} else if d23.Loc == LocReg {
							d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					return bbs[3].Render()
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d28)
					}
					if d28.Loc == LocReg && d1.Loc == LocReg && d28.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d28)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDescsTogether(&d28, &d27)
					if d28.Loc == LocImm && d27.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d28.Imm.Int() < d27.Imm.Int())}
					} else if d27.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d28.Reg)
						if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d28.Reg, int32(d27.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d27.Imm.Int()))
							ctx.EmitCmpInt64(d28.Reg, ctx.ScratchReg)
						}
						d29 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d29)
					} else if d28.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d28.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d27.Reg)
						d29 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d29)
					} else {
						r3 := ctx.AllocRegExcept(d28.Reg)
						ctx.EmitCmpInt64(d28.Reg, d27.Reg)
						d29 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d29)
					}
					d30 = d29
					ctx.SyncDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d30.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d29)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap31 := d1
						snap32 := d2
						snap33 := d3
						snap34 := d4
						snap35 := d6
						snap36 := d7
						snap37 := d8
						snap38 := d9
						snap39 := d19
						snap40 := d20
						snap41 := d21
						snap42 := d22
						snap43 := d23
						snap44 := d24
						snap45 := d26
						snap46 := d27
						snap47 := d28
						snap48 := d29
						snap49 := d30
						alloc50 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc50)
						d1 = snap31
						d2 = snap32
						d3 = snap33
						d4 = snap34
						d6 = snap35
						d7 = snap36
						d8 = snap37
						d9 = snap38
						d19 = snap39
						d20 = snap40
						d21 = snap41
						d22 = snap42
						d23 = snap43
						d24 = snap44
						d26 = snap45
						d27 = snap46
						d28 = snap47
						d29 = snap48
						d30 = snap49
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d26)
					ctx.EnsureDesc(&d28)
					d52 = ctx.EmitSliceElementAddress(&d23, &d28, 16)
					ctx.EnsureDesc(&d52)
					r4 := ctx.AllocRegExcept(d52.Reg)
					ctx.EmitMovRegMem(r4, d52.Reg, 8)
					ctx.EmitMovRegMem(d52.Reg, d52.Reg, 0)
					d51 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d52.Reg, Reg2: r4}
					ctx.BindReg(d52.Reg, &d51)
					ctx.BindReg(r4, &d51)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d28)
					ctx.SyncDesc(&d51)
					d53 = d26
					d53.ID = 0
					d54 = d28
					d54.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d53, &d54, &d51, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d28)
						d54 = d28
						d54.ID = 0
						ctx.EmitStoreScmerSliceElement(&d53, &d54, &d51, int32(16))
					}
					ctx.FreeDesc(&d54)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocReg || d28.Loc == LocFPReg {
						ctx.ProtectReg(d28.Reg)
					} else if d28.Loc == LocRegPair {
						ctx.ProtectReg(d28.Reg)
						ctx.ProtectReg(d28.Reg2)
					}
					d55 = d28
					if d55.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d55)
					ctx.EmitStoreToStack(d55, int32(bbs[3].PhiBase)+int32(0))
					if d28.Loc == LocReg || d28.Loc == LocFPReg {
						ctx.UnprotectReg(d28.Reg)
					} else if d28.Loc == LocRegPair {
						ctx.UnprotectReg(d28.Reg)
						ctx.UnprotectReg(d28.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d26)
					d56 = ctx.EmitNewSliceFromGoSlice(&d26)
					ctx.SyncDesc(&d56)
					if d56.Loc == LocRegPair || d56.Loc == LocStackPair || d56.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
		},
		Optimize: optimizeRegexpMatches,
	})
	// string functions
	DeclareTitle("Strings")
	Declare(&Globalenv, &Declaration{
		Name: "string?",

		Fn: func(a ...Scmer) Scmer {
			if a[0].GetTag() == tagAny {
				_, ok := a[0].Any().(string)
				return NewBool(ok)
			}
			return NewBool(a[0].IsString())
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
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d1 = d0
					d1.ID = 0
					d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x11))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 17)
						d3 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap5 := d0
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						alloc10 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc10)
						d0 = snap5
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d11 = args[0]
					d11.ID = 0
					d11 = JITPrepareScmerGoArg(ctx, d11)
					ctx.SyncDesc(&d11)
					d12 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Any), []JITValueDesc{d11}, 2)
					d12.NoHeapPointer = false
					ctx.BindReg(d12.Reg, &d12)
					ctx.BindReg(d12.Reg2, &d12)
					ctx.FreeDesc(&d11)
					ctx.EnsureDesc(&d12)
					callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(jitAssertString), []JITValueDesc{d12}, []uint8{2, 1}, []uint8{1, 0})
					d14 = callResults13[0]
					d15 = callResults13[1]
					_ = d14
					_ = d15
					ctx.EmitAndRegImm32(d15.Reg, 1)
					d15.Type = tagBool
					ctx.FreeDesc(&d12)
					ctx.SyncDesc(&d15)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d15) {
						return d15
					}
					if d15.Loc == LocImm {
						ctx.EmitMakeBool(result, d15)
					} else {
						ctx.EmitMovToReg(result.Reg2, d15)
						d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d16)
						if d15.Loc == LocReg && d15.Reg != result.Reg2 {
							ctx.FreeReg(d15.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d17 = args[0]
					d17.ID = 0
					d19 = d17
					d19.ID = 0
					d18 = ctx.EmitIsStringBorrowed(&d19, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d17)
					ctx.SyncDesc(&d18)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d18) {
						return d18
					}
					if d18.Loc == LocImm {
						ctx.EmitMakeBool(result, d18)
					} else {
						ctx.EmitMovToReg(result.Reg2, d18)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d20)
						if d18.Loc == LocReg && d18.Reg != result.Reg2 {
							ctx.FreeReg(d18.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  18,
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
				if s.GetTag() == tagAny {
					if stream, ok := s.Any().(io.Reader); ok {
						_, _ = io.Copy(&sb, stream)
						continue
					}
				}
				appendOperatorText(&sb, s)
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
			i := ToInt(a[1])
			if isCompressedText(a[0]) {
				end := compressedTextLen(a[0])
				if len(a) > 2 {
					end = i + ToInt(a[2])
				}
				return cstringSubstring(a[0], i, end)
			}
			s := String(a[0])
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
				var phiBase8 int32
				_ = phiBase8
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
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
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
				var d67 JITValueDesc
				_ = d67
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [7]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[1]
					d2.ID = 0
					ctx.EnsureDesc(&d2)
					d3 = d2
					_ = d3
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.FreeDesc(&d2)
					d6 = args[0]
					d6.ID = 0
					ctx.EnsureDesc(&d6)
					d7 = d6
					_ = d7
					ctx.StabilizeDescForControlFlow(&d7)
					phiBase8 = ctx.AllocStack(int32(16))
					d9 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase8) + int32(0)}
					_ = d9
					lbl9 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d9 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase8) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d10 = d7
					d10.ID = 0
					d11 = ctx.EmitGetTagDesc(&d10, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d11.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitCmpRegImm32(d11.Reg, 19)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.SyncDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase8)+int32(0))
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase8)+int32(0))
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d9 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase8) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d9)
					}
					ctx.EmitJmp(lbl9)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d9 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase8) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d11.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitCmpRegImm32(d11.Reg, 20)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.SyncDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase8)+int32(0))
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d15.Condition, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.FreeDesc(&d14)
						ctx.MarkLabel(lbl16)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase8)+int32(0))
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl11)
					}
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d9 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase8) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d11.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d11.Reg, 26)
						r3 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d16)
					}
					ctx.EnsureDesc(&d16)
					ctx.EmitStoreToStack(d16, int32(phiBase8)+int32(0))
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl9)
					d17 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d17)
					ctx.BindReg(r1, &d17)
					ctx.FreeDesc(&d6)
					d18 = d17
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap19 := d1
						snap20 := d2
						snap21 := d3
						snap22 := d4
						snap23 := d5
						snap24 := d6
						snap25 := d7
						snap26 := d9
						snap27 := d10
						snap28 := d11
						snap29 := d12
						snap30 := d13
						snap31 := d14
						snap32 := d15
						snap33 := d16
						snap34 := d17
						snap35 := d18
						alloc36 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc36)
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d4 = snap22
						d5 = snap23
						d6 = snap24
						d7 = snap25
						d9 = snap26
						d10 = snap27
						d11 = snap28
						d12 = snap29
						d13 = snap30
						d14 = snap31
						d15 = snap32
						d16 = snap33
						d17 = snap34
						d18 = snap35
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d37 = args[0]
					d37.ID = 0
					d37 = JITPrepareScmerGoArg(ctx, d37)
					ctx.SyncDesc(&d37)
					d38 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d37}, 1)
					d38.NoHeapPointer = true
					ctx.BindReg(d38.Reg, &d38)
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.FreeDesc(&d37)
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d39.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d39.Reg, 2)
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedGreater}
						ctx.BindReg(r4, &d40)
					}
					ctx.FreeDesc(&d39)
					d41 = d40
					ctx.SyncDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.SyncDesc(&d38)
						if d38.Loc == LocReg || d38.Loc == LocFPReg {
							ctx.ProtectReg(d38.Reg)
						} else if d38.Loc == LocRegPair {
							ctx.ProtectReg(d38.Reg)
							ctx.ProtectReg(d38.Reg2)
						}
						d42 = d38
						if d42.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d42)
						ctx.EmitStoreToStack(d42, int32(bbs[4].PhiBase)+int32(0))
						if d38.Loc == LocReg || d38.Loc == LocFPReg {
							ctx.UnprotectReg(d38.Reg)
						} else if d38.Loc == LocRegPair {
							ctx.UnprotectReg(d38.Reg)
							ctx.UnprotectReg(d38.Reg2)
						}
						return bbs[4].Render()
					}
					lbl18 := ctx.ReserveLabel()
					ctx.EmitJump(d41.Condition, lbl4)
					ctx.EmitJmp(lbl18)
					ctx.FreeDesc(&d40)
					snap43 := d1
					snap44 := d2
					snap45 := d3
					snap46 := d4
					snap47 := d5
					snap48 := d6
					snap49 := d7
					snap50 := d9
					snap51 := d10
					snap52 := d11
					snap53 := d12
					snap54 := d13
					snap55 := d14
					snap56 := d15
					snap57 := d16
					snap58 := d17
					snap59 := d18
					snap60 := d37
					snap61 := d38
					snap62 := d39
					snap63 := d40
					snap64 := d41
					snap65 := d42
					alloc66 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl18)
					ctx.SyncDesc(&d38)
					if d38.Loc == LocReg || d38.Loc == LocFPReg {
						ctx.ProtectReg(d38.Reg)
					} else if d38.Loc == LocRegPair {
						ctx.ProtectReg(d38.Reg)
						ctx.ProtectReg(d38.Reg2)
					}
					d67 = d38
					if d67.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d67)
					ctx.EmitStoreToStack(d67, int32(bbs[4].PhiBase)+int32(0))
					if d38.Loc == LocReg || d38.Loc == LocFPReg {
						ctx.UnprotectReg(d38.Reg)
					} else if d38.Loc == LocRegPair {
						ctx.UnprotectReg(d38.Reg)
						ctx.UnprotectReg(d38.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc66)
					d1 = snap43
					d2 = snap44
					d3 = snap45
					d4 = snap46
					d5 = snap47
					d6 = snap48
					d7 = snap49
					d9 = snap50
					d10 = snap51
					d11 = snap52
					d12 = snap53
					d13 = snap54
					d14 = snap55
					d15 = snap56
					d16 = snap57
					d17 = snap58
					d18 = snap59
					d37 = snap60
					d38 = snap61
					d39 = snap62
					d40 = snap63
					d41 = snap64
					d42 = snap65
					if !bbs[4].Rendered {
						snap68 := d1
						snap69 := d2
						snap70 := d3
						snap71 := d4
						snap72 := d5
						snap73 := d6
						snap74 := d7
						snap75 := d9
						snap76 := d10
						snap77 := d11
						snap78 := d12
						snap79 := d13
						snap80 := d14
						snap81 := d15
						snap82 := d16
						snap83 := d17
						snap84 := d18
						snap85 := d37
						snap86 := d38
						snap87 := d39
						snap88 := d40
						snap89 := d41
						snap90 := d42
						snap91 := d67
						alloc92 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc92)
						d1 = snap68
						d2 = snap69
						d3 = snap70
						d4 = snap71
						d5 = snap72
						d6 = snap73
						d7 = snap74
						d9 = snap75
						d10 = snap76
						d11 = snap77
						d12 = snap78
						d13 = snap79
						d14 = snap80
						d15 = snap81
						d16 = snap82
						d17 = snap83
						d18 = snap84
						d37 = snap85
						d38 = snap86
						d39 = snap87
						d40 = snap88
						d41 = snap89
						d42 = snap90
						d67 = snap91
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d93 = args[0]
					d93.ID = 0
					d95 = d93
					ctx.SyncDesc(&d95)
					if d95.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d95.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d95.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d95 = tmpScalar
					}
					d95 = JITPrepareScmerGoArg(ctx, d95)
					if d95.Loc != LocRegPair && d95.Loc != LocStackPair && d95.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d94 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d95}, 2)
					ctx.StabilizeDescForControlFlow(&d94)
					ctx.FreeDesc(&d93)
					d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d96)
					if d96.Loc == LocImm {
						d97 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d96.Imm.Int() > 2)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d96.Reg, 2)
						d97 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreater}
						ctx.BindReg(r5, &d97)
					}
					ctx.FreeDesc(&d96)
					d98 = d97
					ctx.SyncDesc(&d98)
					if d98.Loc != LocImm && d98.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d98.Loc == LocImm {
						if d98.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d98.Condition, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d97)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap99 := d1
						snap100 := d2
						snap101 := d3
						snap102 := d4
						snap103 := d5
						snap104 := d6
						snap105 := d7
						snap106 := d9
						snap107 := d10
						snap108 := d11
						snap109 := d12
						snap110 := d13
						snap111 := d14
						snap112 := d15
						snap113 := d16
						snap114 := d17
						snap115 := d18
						snap116 := d37
						snap117 := d38
						snap118 := d39
						snap119 := d40
						snap120 := d41
						snap121 := d42
						snap122 := d67
						snap123 := d93
						snap124 := d94
						snap125 := d95
						snap126 := d96
						snap127 := d97
						snap128 := d98
						alloc129 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc129)
						d1 = snap99
						d2 = snap100
						d3 = snap101
						d4 = snap102
						d5 = snap103
						d6 = snap104
						d7 = snap105
						d9 = snap106
						d10 = snap107
						d11 = snap108
						d12 = snap109
						d13 = snap110
						d14 = snap111
						d15 = snap112
						d16 = snap113
						d17 = snap114
						d18 = snap115
						d37 = snap116
						d38 = snap117
						d39 = snap118
						d40 = snap119
						d41 = snap120
						d42 = snap121
						d67 = snap122
						d93 = snap123
						d94 = snap124
						d95 = snap125
						d96 = snap126
						d97 = snap127
						d98 = snap128
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d130 = args[2]
					d130.ID = 0
					ctx.EnsureDesc(&d130)
					d131 = d130
					_ = d131
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d131.Loc == LocImm {
						d132 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d131.Imm.Int())}
					} else if d131.Type == tagInt && d131.Loc == LocRegPair {
						ctx.FreeReg(d131.Reg)
						d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d131.Reg2}
						ctx.BindReg(d131.Reg2, &d132)
						ctx.BindReg(d131.Reg2, &d132)
					} else if d131.Type == tagInt && d131.Loc == LocReg {
						d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d131.Reg}
						ctx.BindReg(d131.Reg, &d132)
						ctx.BindReg(d131.Reg, &d132)
					} else {
						d132 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d131}, 1)
						d132.Type = tagInt
						ctx.BindReg(d132.Reg, &d132)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d132)
					ctx.EnsureDesc(&d132)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d132)
					ctx.FreeDesc(&d130)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d132)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d132)
					if d4.Loc == LocImm && d132.Loc == LocImm {
						d134 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d132.Imm.Int())}
					} else if d132.Loc == LocImm && d132.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						r6 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(r6, d4.Reg)
						d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d134)
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d132)
						d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d132.Reg}
						ctx.BindReg(d132.Reg, &d134)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d132)
						scratch := ctx.AllocRegExcept(d132.Reg)
						ctx.EmitMovRegReg(scratch, d132.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d134)
					} else if d132.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d132.Imm.Int())
						d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d134)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d132)
						r7 := ctx.AllocRegExceptOperand(&d132, d4.Reg)
						ctx.EmitMovRegReg(r7, d4.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r7, &d132)
						d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d134)
					}
					if d134.Loc == LocReg && d4.Loc == LocReg && d134.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d134)
					ctx.EmitStoreToStack(d134, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d134)
					ctx.FreeDesc(&d132)
					return bbs[4].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d135 = args[0]
					d135.ID = 0
					d135 = JITPrepareScmerGoArg(ctx, d135)
					if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocRegTriple || d4.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d135)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d1)
					d136 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d135, d4, d1}, 2)
					d136.NoHeapPointer = false
					ctx.BindReg(d136.Reg, &d136)
					ctx.BindReg(d136.Reg2, &d136)
					ctx.FreeDesc(&d135)
					ctx.FreeDesc(&d1)
					ctx.SyncDesc(&d136)
					if d136.Loc == LocRegPair || d136.Loc == LocStackPair || d136.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d136, &result)
						result.Type = d136.Type
					} else {
						switch d136.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d136)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d136)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d136)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d136, &result)
							result.Type = d136.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d137 = args[2]
					d137.ID = 0
					ctx.EnsureDesc(&d137)
					d138 = d137
					_ = d138
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d138.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d138.Imm.Int())}
					} else if d138.Type == tagInt && d138.Loc == LocRegPair {
						ctx.FreeReg(d138.Reg)
						d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d138.Reg2}
						ctx.BindReg(d138.Reg2, &d139)
						ctx.BindReg(d138.Reg2, &d139)
					} else if d138.Type == tagInt && d138.Loc == LocReg {
						d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d138.Reg}
						ctx.BindReg(d138.Reg, &d139)
						ctx.BindReg(d138.Reg, &d139)
					} else {
						d139 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d138}, 1)
						d139.Type = tagInt
						ctx.BindReg(d139.Reg, &d139)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d139)
					ctx.EnsureDesc(&d139)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d139)
					ctx.FreeDesc(&d137)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d139)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d139)
					if d4.Loc == LocImm && d139.Loc == LocImm {
						d141 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d139.Imm.Int())}
					} else if d139.Loc == LocImm && d139.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						r8 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(r8, d4.Reg)
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d141)
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d139)
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d139.Reg}
						ctx.BindReg(d139.Reg, &d141)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d139)
						scratch := ctx.AllocRegExcept(d139.Reg)
						ctx.EmitMovRegReg(scratch, d139.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d141)
					} else if d139.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d139.Imm.Int())
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d141)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d139)
						r9 := ctx.AllocRegExceptOperand(&d139, d4.Reg)
						ctx.EmitMovRegReg(r9, d4.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r9, &d139)
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d141)
					}
					if d141.Loc == LocReg && d4.Loc == LocReg && d141.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.FreeDesc(&d139)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d141)
					ctx.EnsureDesc(&d94)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d141)
					if d141.Loc == LocImm && d4.Loc == LocImm {
						d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d141.Imm.Int() - d4.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d141.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d141.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d141.Reg)
						}
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitSubInt64(r10, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r10, d4.Reg)
						}
						d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d143)
					}
					r11 := ctx.EmitSliceDataAfterLow(&d94, &d4, 1)
					d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d144)
					ctx.BindReg(r11, &d144)
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d144)
					ctx.EnsureDesc(&d144)
					if d144.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d144.Imm.Int()))
					} else {
						r12 = d144.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d143)
					ctx.EnsureDesc(&d143)
					if d143.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d143.Imm.Int()))
					} else {
						r13 = d143.Reg
					}
					ctx.ProtectReg(r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d145 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d145)
					ctx.BindReg(r13, &d145)
					ctx.BindReg(r12, &d145)
					ctx.BindReg(r13, &d145)
					ctx.FreeDesc(&d141)
					ctx.EnsureDesc(&d145)
					d146 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d145}, 2)
					ctx.EmitMovPairToResult(&d146, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d94)
					if d94.Loc == LocRegPair || d94.Loc == LocRegTriple {
						d147 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d94.Reg2}
						ctx.BindReg(d94.Reg2, &d147)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d94)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d147)
					if d147.Loc == LocImm && d4.Loc == LocImm {
						d149 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d147.Imm.Int() - d4.Imm.Int())}
					} else {
						r14 := ctx.AllocReg()
						if d147.Loc == LocImm {
							ctx.EmitMovRegImm64(r14, uint64(d147.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r14, d147.Reg)
						}
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitSubInt64(r14, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r14, d4.Reg)
						}
						d149 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d149)
					}
					r15 := ctx.EmitSliceDataAfterLow(&d94, &d4, 1)
					d150 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
					ctx.BindReg(r15, &d150)
					ctx.BindReg(r15, &d150)
					var r16 Reg
					var r17 Reg
					ctx.SyncDesc(&d150)
					ctx.EnsureDesc(&d150)
					if d150.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d150.Imm.Int()))
					} else {
						r16 = d150.Reg
					}
					ctx.ProtectReg(r16)
					ctx.SyncDesc(&d149)
					ctx.EnsureDesc(&d149)
					if d149.Loc == LocImm {
						r17 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r17, uint64(d149.Imm.Int()))
					} else {
						r17 = d149.Reg
					}
					ctx.ProtectReg(r17)
					ctx.UnprotectReg(r17)
					ctx.UnprotectReg(r16)
					d151 = JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
					ctx.BindReg(r16, &d151)
					ctx.BindReg(r17, &d151)
					ctx.BindReg(r16, &d151)
					ctx.BindReg(r17, &d151)
					ctx.EnsureDesc(&d151)
					d152 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d151}, 2)
					ctx.EmitMovPairToResult(&d152, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost:  57,
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_substr",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			s := ""
			slen := 0
			if isCompressedText(a[0]) {
				slen = compressedTextLen(a[0])
			} else {
				s = String(a[0])
				slen = len(s)
			}
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
				if isCompressedText(a[0]) {
					return cstringSubstring(a[0], start, start+n)
				}
				return NewString(s[start : start+n])
			}
			if isCompressedText(a[0]) {
				return cstringSubstring(a[0], start, slen)
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
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var phiBase21 int32
				_ = phiBase21
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
				var d98 JITValueDesc
				_ = d98
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
				var d298 JITValueDesc
				_ = d298
				var d355 JITValueDesc
				_ = d355
				var d356 JITValueDesc
				_ = d356
				var phiBase357 int32
				_ = phiBase357
				var d358 JITValueDesc
				_ = d358
				var d359 JITValueDesc
				_ = d359
				var d360 JITValueDesc
				_ = d360
				var d361 JITValueDesc
				_ = d361
				var d362 JITValueDesc
				_ = d362
				var d363 JITValueDesc
				_ = d363
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d436 JITValueDesc
				_ = d436
				var d437 JITValueDesc
				_ = d437
				var d438 JITValueDesc
				_ = d438
				var d510 JITValueDesc
				_ = d510
				var d511 JITValueDesc
				_ = d511
				var d512 JITValueDesc
				_ = d512
				var d513 JITValueDesc
				_ = d513
				var phiBase514 int32
				_ = phiBase514
				var d515 JITValueDesc
				_ = d515
				var d516 JITValueDesc
				_ = d516
				var d517 JITValueDesc
				_ = d517
				var d518 JITValueDesc
				_ = d518
				var d519 JITValueDesc
				_ = d519
				var d520 JITValueDesc
				_ = d520
				var d521 JITValueDesc
				_ = d521
				var d522 JITValueDesc
				_ = d522
				var d523 JITValueDesc
				_ = d523
				var d524 JITValueDesc
				_ = d524
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
				var d619 JITValueDesc
				_ = d619
				var d620 JITValueDesc
				_ = d620
				var d621 JITValueDesc
				_ = d621
				var d622 JITValueDesc
				_ = d622
				var d623 JITValueDesc
				_ = d623
				var d624 JITValueDesc
				_ = d624
				var d625 JITValueDesc
				_ = d625
				var d626 JITValueDesc
				_ = d626
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(64))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [20]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiBase = int32(phiBase0) + int32(32)
				bbs[13].PhiBase = int32(phiBase0) + int32(48)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				d4 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d5 = args[0]
					d5.ID = 0
					d7 = d5
					d7.ID = 0
					d6 = ctx.EmitTagEqualsBorrowed(&d7, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d5)
					d8 = d6
					ctx.EnsureDesc(&d8)
					if d8.Loc != LocImm && d8.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d8.Loc == LocImm {
						if d8.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d8.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap9 := d1
						snap10 := d2
						snap11 := d3
						snap12 := d4
						snap13 := d5
						snap14 := d6
						snap15 := d7
						snap16 := d8
						alloc17 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc17)
						d1 = snap9
						d2 = snap10
						d3 = snap11
						d4 = snap12
						d5 = snap13
						d6 = snap14
						d7 = snap15
						d8 = snap16
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d6)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d19 = args[0]
					d19.ID = 0
					ctx.EnsureDesc(&d19)
					d20 = d19
					_ = d20
					ctx.StabilizeDescForControlFlow(&d20)
					phiBase21 = ctx.AllocStack(int32(16))
					d22 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase21) + int32(0)}
					_ = d22
					lbl21 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl22 := ctx.ReserveLabel()
					_ = lbl22
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl24 := ctx.ReserveLabel()
					_ = lbl24
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl25 := ctx.ReserveLabel()
					_ = lbl25
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase21) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d23 = d20
					d23.ID = 0
					d24 = ctx.EmitGetTagDesc(&d23, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d24)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d24.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitCmpRegImm32(d24.Reg, 19)
						d25 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d25)
					}
					ctx.ReclaimUntrackedRegs()
					d26 = d25
					ctx.SyncDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							ctx.MarkLabel(lbl26)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase21)+int32(0))
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl27)
							ctx.EmitJmp(lbl25)
						}
					} else {
						ctx.EmitJump(d26.Condition, lbl26)
						ctx.EmitJmp(lbl27)
						ctx.FreeDesc(&d25)
						ctx.MarkLabel(lbl26)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase21)+int32(0))
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl27)
						ctx.EmitJmp(lbl25)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase21) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d22)
					}
					ctx.EmitJmp(lbl21)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl25)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase21) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d24.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitCmpRegImm32(d24.Reg, 20)
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d27)
					}
					ctx.ReclaimUntrackedRegs()
					d28 = d27
					ctx.SyncDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase21)+int32(0))
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitJump(d28.Condition, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.FreeDesc(&d27)
						ctx.MarkLabel(lbl28)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase21)+int32(0))
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl23)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase21) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d24.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d24.Reg, 26)
						r3 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d29)
					}
					ctx.EnsureDesc(&d29)
					ctx.EmitStoreToStack(d29, int32(phiBase21)+int32(0))
					ctx.StabilizeDescForControlFlow(&d29)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl24)
					ctx.MarkLabel(lbl21)
					d30 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d30)
					ctx.BindReg(r1, &d30)
					ctx.FreeDesc(&d19)
					d31 = d30
					ctx.EnsureDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d31.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap32 := d1
						snap33 := d2
						snap34 := d3
						snap35 := d4
						snap36 := d5
						snap37 := d6
						snap38 := d7
						snap39 := d8
						snap40 := d18
						snap41 := d19
						snap42 := d20
						snap43 := d22
						snap44 := d23
						snap45 := d24
						snap46 := d25
						snap47 := d26
						snap48 := d27
						snap49 := d28
						snap50 := d29
						snap51 := d30
						snap52 := d31
						alloc53 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc53)
						d1 = snap32
						d2 = snap33
						d3 = snap34
						d4 = snap35
						d5 = snap36
						d6 = snap37
						d7 = snap38
						d8 = snap39
						d18 = snap40
						d19 = snap41
						d20 = snap42
						d22 = snap43
						d23 = snap44
						d24 = snap45
						d25 = snap46
						d26 = snap47
						d27 = snap48
						d28 = snap49
						d29 = snap50
						d30 = snap51
						d31 = snap52
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d30)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d54 = args[0]
					d54.ID = 0
					d54 = JITPrepareScmerGoArg(ctx, d54)
					ctx.SyncDesc(&d54)
					d55 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d54}, 1)
					d55.NoHeapPointer = true
					ctx.BindReg(d55.Reg, &d55)
					ctx.StabilizeDescForControlFlow(&d55)
					ctx.FreeDesc(&d54)
					ctx.SyncDesc(&d55)
					if d55.Loc == LocReg || d55.Loc == LocFPReg {
						ctx.ProtectReg(d55.Reg)
					} else if d55.Loc == LocRegPair {
						ctx.ProtectReg(d55.Reg)
						ctx.ProtectReg(d55.Reg2)
					}
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}, int32(bbs[4].PhiBase)+int32(0))
					d56 = d55
					if d56.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d56)
					ctx.EmitStoreToStack(d56, int32(bbs[4].PhiBase)+int32(16))
					if d55.Loc == LocReg || d55.Loc == LocFPReg {
						ctx.UnprotectReg(d55.Reg)
					} else if d55.Loc == LocRegPair {
						ctx.UnprotectReg(d55.Reg)
						ctx.UnprotectReg(d55.Reg2)
					}
					return bbs[4].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					d57 = args[1]
					d57.ID = 0
					ctx.EnsureDesc(&d57)
					d58 = d57
					_ = d58
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl30 := ctx.ReserveLabel()
					_ = lbl30
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl30)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d58.Loc == LocImm {
						d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d58.Imm.Int())}
					} else if d58.Type == tagInt && d58.Loc == LocRegPair {
						ctx.FreeReg(d58.Reg)
						d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d58.Reg2}
						ctx.BindReg(d58.Reg2, &d59)
						ctx.BindReg(d58.Reg2, &d59)
					} else if d58.Type == tagInt && d58.Loc == LocReg {
						d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d58.Reg}
						ctx.BindReg(d58.Reg, &d59)
						ctx.BindReg(d58.Reg, &d59)
					} else {
						d59 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d58}, 1)
						d59.Type = tagInt
						ctx.BindReg(d59.Reg, &d59)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d59)
					ctx.FreeDesc(&d57)
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocImm {
						d61 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d59.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d59.Reg)
						ctx.EmitMovRegReg(scratch, d59.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d61 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d61)
					}
					if d61.Loc == LocReg && d59.Loc == LocReg && d61.Reg == d59.Reg {
						ctx.TransferReg(d59.Reg)
						d59.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d61)
					ctx.FreeDesc(&d59)
					ctx.EnsureDesc(&d61)
					if d61.Loc == LocImm {
						d62 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d61.Imm.Int() < 0)}
					} else {
						r4 := ctx.AllocRegExcept(d61.Reg)
						ctx.EmitCmpRegImm32(d61.Reg, 0)
						d62 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d62)
					}
					d63 = d62
					ctx.SyncDesc(&d63)
					if d63.Loc != LocImm && d63.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d63.Loc == LocImm {
						if d63.Imm.Bool() {
							return bbs[6].Render()
						}
						ctx.SyncDesc(&d61)
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.ProtectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.ProtectReg(d61.Reg)
							ctx.ProtectReg(d61.Reg2)
						}
						d64 = d61
						if d64.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d64)
						ctx.EmitStoreToStack(d64, int32(bbs[7].PhiBase)+int32(0))
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.UnprotectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.UnprotectReg(d61.Reg)
							ctx.UnprotectReg(d61.Reg2)
						}
						return bbs[7].Render()
					}
					lbl31 := ctx.ReserveLabel()
					ctx.EmitJump(d63.Condition, lbl7)
					ctx.EmitJmp(lbl31)
					ctx.FreeDesc(&d62)
					snap65 := d1
					snap66 := d2
					snap67 := d3
					snap68 := d4
					snap69 := d5
					snap70 := d6
					snap71 := d7
					snap72 := d8
					snap73 := d18
					snap74 := d19
					snap75 := d20
					snap76 := d22
					snap77 := d23
					snap78 := d24
					snap79 := d25
					snap80 := d26
					snap81 := d27
					snap82 := d28
					snap83 := d29
					snap84 := d30
					snap85 := d31
					snap86 := d54
					snap87 := d55
					snap88 := d56
					snap89 := d57
					snap90 := d58
					snap91 := d59
					snap92 := d60
					snap93 := d61
					snap94 := d62
					snap95 := d63
					snap96 := d64
					alloc97 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl31)
					ctx.SyncDesc(&d61)
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.ProtectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.ProtectReg(d61.Reg)
						ctx.ProtectReg(d61.Reg2)
					}
					d98 = d61
					if d98.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d98)
					ctx.EmitStoreToStack(d98, int32(bbs[7].PhiBase)+int32(0))
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.UnprotectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.UnprotectReg(d61.Reg)
						ctx.UnprotectReg(d61.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc97)
					d1 = snap65
					d2 = snap66
					d3 = snap67
					d4 = snap68
					d5 = snap69
					d6 = snap70
					d7 = snap71
					d8 = snap72
					d18 = snap73
					d19 = snap74
					d20 = snap75
					d22 = snap76
					d23 = snap77
					d24 = snap78
					d25 = snap79
					d26 = snap80
					d27 = snap81
					d28 = snap82
					d29 = snap83
					d30 = snap84
					d31 = snap85
					d54 = snap86
					d55 = snap87
					d56 = snap88
					d57 = snap89
					d58 = snap90
					d59 = snap91
					d60 = snap92
					d61 = snap93
					d62 = snap94
					d63 = snap95
					d64 = snap96
					if !bbs[7].Rendered {
						snap99 := d1
						snap100 := d2
						snap101 := d3
						snap102 := d4
						snap103 := d5
						snap104 := d6
						snap105 := d7
						snap106 := d8
						snap107 := d18
						snap108 := d19
						snap109 := d20
						snap110 := d22
						snap111 := d23
						snap112 := d24
						snap113 := d25
						snap114 := d26
						snap115 := d27
						snap116 := d28
						snap117 := d29
						snap118 := d30
						snap119 := d31
						snap120 := d54
						snap121 := d55
						snap122 := d56
						snap123 := d57
						snap124 := d58
						snap125 := d59
						snap126 := d60
						snap127 := d61
						snap128 := d62
						snap129 := d63
						snap130 := d64
						snap131 := d98
						alloc132 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc132)
						d1 = snap99
						d2 = snap100
						d3 = snap101
						d4 = snap102
						d5 = snap103
						d6 = snap104
						d7 = snap105
						d8 = snap106
						d18 = snap107
						d19 = snap108
						d20 = snap109
						d22 = snap110
						d23 = snap111
						d24 = snap112
						d25 = snap113
						d26 = snap114
						d27 = snap115
						d28 = snap116
						d29 = snap117
						d30 = snap118
						d31 = snap119
						d54 = snap120
						d55 = snap121
						d56 = snap122
						d57 = snap123
						d58 = snap124
						d59 = snap125
						d60 = snap126
						d61 = snap127
						d62 = snap128
						d63 = snap129
						d64 = snap130
						d98 = snap131
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d133 = args[0]
					d133.ID = 0
					d135 = d133
					ctx.SyncDesc(&d135)
					if d135.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d135.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d135.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d135 = tmpScalar
					}
					d135 = JITPrepareScmerGoArg(ctx, d135)
					if d135.Loc != LocRegPair && d135.Loc != LocStackPair && d135.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d134 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d135}, 2)
					ctx.StabilizeDescForControlFlow(&d134)
					ctx.FreeDesc(&d133)
					if d134.SliceSizeKnown {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d134.KnownSliceLen))}
					} else if d134.Loc == LocImm {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d134.Imm.String())))}
					} else if d134.Loc == LocStackTriple {
						d136 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d134.StackOff + 8, NoHeapPointer: true}
					} else if d134.Loc == LocStackPair {
						d136 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d134.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d134)
						if d134.Loc == LocRegPair || d134.Loc == LocRegTriple {
							d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d134.Reg2, ID: 0}
						} else if d134.Loc == LocReg {
							d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d134.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d136)
					ctx.SyncDesc(&d134)
					if d134.Loc == LocReg || d134.Loc == LocFPReg {
						ctx.ProtectReg(d134.Reg)
					} else if d134.Loc == LocRegPair {
						ctx.ProtectReg(d134.Reg)
						ctx.ProtectReg(d134.Reg2)
					}
					ctx.SyncDesc(&d136)
					if d136.Loc == LocReg || d136.Loc == LocFPReg {
						ctx.ProtectReg(d136.Reg)
					} else if d136.Loc == LocRegPair {
						ctx.ProtectReg(d136.Reg)
						ctx.ProtectReg(d136.Reg2)
					}
					d137 = d134
					if d137.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d137)
					if d137.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d137, int32(bbs[4].PhiBase)+int32(0), 2)
					} else if d137.Loc == LocInputPair {
						ctx.EnsureDesc(&d137)
						ctx.EmitStoreScmerToStack(d137, int32(bbs[4].PhiBase)+int32(0))
					} else if d137.Loc == LocRegPair || d137.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d137, int32(bbs[4].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d137)
						ctx.EmitStoreToStack(d137, int32(bbs[4].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
					}
					d138 = d136
					if d138.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d138)
					ctx.EmitStoreToStack(d138, int32(bbs[4].PhiBase)+int32(16))
					if d134.Loc == LocReg || d134.Loc == LocFPReg {
						ctx.UnprotectReg(d134.Reg)
					} else if d134.Loc == LocRegPair {
						ctx.UnprotectReg(d134.Reg)
						ctx.UnprotectReg(d134.Reg2)
					}
					if d136.Loc == LocReg || d136.Loc == LocFPReg {
						ctx.UnprotectReg(d136.Reg)
					} else if d136.Loc == LocRegPair {
						ctx.UnprotectReg(d136.Reg)
						ctx.UnprotectReg(d136.Reg2)
					}
					return bbs[4].Render()
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(0))
					return bbs[7].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDescsTogether(&d3, &d2)
					if d3.Loc == LocImm && d2.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() >= d2.Imm.Int())}
					} else if d2.Loc == LocImm {
						r5 := ctx.AllocRegExcept(d3.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d139 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r5, &d139)
					} else if d3.Loc == LocImm {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
						d139 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r6, &d139)
					} else {
						r7 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d2.Reg)
						d139 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r7, &d139)
					}
					d140 = d139
					ctx.SyncDesc(&d140)
					if d140.Loc != LocImm && d140.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d140.Loc == LocImm {
						if d140.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitJump(d140.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d139)
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap141 := d1
						snap142 := d2
						snap143 := d3
						snap144 := d4
						snap145 := d5
						snap146 := d6
						snap147 := d7
						snap148 := d8
						snap149 := d18
						snap150 := d19
						snap151 := d20
						snap152 := d22
						snap153 := d23
						snap154 := d24
						snap155 := d25
						snap156 := d26
						snap157 := d27
						snap158 := d28
						snap159 := d29
						snap160 := d30
						snap161 := d31
						snap162 := d54
						snap163 := d55
						snap164 := d56
						snap165 := d57
						snap166 := d58
						snap167 := d59
						snap168 := d60
						snap169 := d61
						snap170 := d62
						snap171 := d63
						snap172 := d64
						snap173 := d98
						snap174 := d133
						snap175 := d134
						snap176 := d135
						snap177 := d136
						snap178 := d137
						snap179 := d138
						snap180 := d139
						snap181 := d140
						alloc182 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc182)
						d1 = snap141
						d2 = snap142
						d3 = snap143
						d4 = snap144
						d5 = snap145
						d6 = snap146
						d7 = snap147
						d8 = snap148
						d18 = snap149
						d19 = snap150
						d20 = snap151
						d22 = snap152
						d23 = snap153
						d24 = snap154
						d25 = snap155
						d26 = snap156
						d27 = snap157
						d28 = snap158
						d29 = snap159
						d30 = snap160
						d31 = snap161
						d54 = snap162
						d55 = snap163
						d56 = snap164
						d57 = snap165
						d58 = snap166
						d59 = snap167
						d60 = snap168
						d61 = snap169
						d62 = snap170
						d63 = snap171
						d64 = snap172
						d98 = snap173
						d133 = snap174
						d134 = snap175
						d135 = snap176
						d136 = snap177
						d137 = snap178
						d138 = snap179
						d139 = snap180
						d140 = snap181
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d183 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d184 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d183}, 2)
					ctx.EmitMovPairToResult(&d184, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d185 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d185)
					if d185.Loc == LocImm {
						d186 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d185.Imm.Int() > 2)}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d185.Reg, 2)
						d186 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedGreater}
						ctx.BindReg(r8, &d186)
					}
					ctx.FreeDesc(&d185)
					d187 = d186
					ctx.SyncDesc(&d187)
					if d187.Loc != LocImm && d187.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d187.Loc == LocImm {
						if d187.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitJump(d187.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d186)
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap188 := d1
						snap189 := d2
						snap190 := d3
						snap191 := d4
						snap192 := d5
						snap193 := d6
						snap194 := d7
						snap195 := d8
						snap196 := d18
						snap197 := d19
						snap198 := d20
						snap199 := d22
						snap200 := d23
						snap201 := d24
						snap202 := d25
						snap203 := d26
						snap204 := d27
						snap205 := d28
						snap206 := d29
						snap207 := d30
						snap208 := d31
						snap209 := d54
						snap210 := d55
						snap211 := d56
						snap212 := d57
						snap213 := d58
						snap214 := d59
						snap215 := d60
						snap216 := d61
						snap217 := d62
						snap218 := d63
						snap219 := d64
						snap220 := d98
						snap221 := d133
						snap222 := d134
						snap223 := d135
						snap224 := d136
						snap225 := d137
						snap226 := d138
						snap227 := d139
						snap228 := d140
						snap229 := d183
						snap230 := d184
						snap231 := d185
						snap232 := d186
						snap233 := d187
						alloc234 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc234)
						d1 = snap188
						d2 = snap189
						d3 = snap190
						d4 = snap191
						d5 = snap192
						d6 = snap193
						d7 = snap194
						d8 = snap195
						d18 = snap196
						d19 = snap197
						d20 = snap198
						d22 = snap199
						d23 = snap200
						d24 = snap201
						d25 = snap202
						d26 = snap203
						d27 = snap204
						d28 = snap205
						d29 = snap206
						d30 = snap207
						d31 = snap208
						d54 = snap209
						d55 = snap210
						d56 = snap211
						d57 = snap212
						d58 = snap213
						d59 = snap214
						d60 = snap215
						d61 = snap216
						d62 = snap217
						d63 = snap218
						d64 = snap219
						d98 = snap220
						d133 = snap221
						d134 = snap222
						d135 = snap223
						d136 = snap224
						d137 = snap225
						d138 = snap226
						d139 = snap227
						d140 = snap228
						d183 = snap229
						d184 = snap230
						d185 = snap231
						d186 = snap232
						d187 = snap233
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d235 = args[2]
					d235.ID = 0
					ctx.EnsureDesc(&d235)
					d236 = d235
					_ = d236
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl32 := ctx.ReserveLabel()
					_ = lbl32
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl32)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d236.Loc == LocImm {
						d237 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d236.Imm.Int())}
					} else if d236.Type == tagInt && d236.Loc == LocRegPair {
						ctx.FreeReg(d236.Reg)
						d237 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d236.Reg2}
						ctx.BindReg(d236.Reg2, &d237)
						ctx.BindReg(d236.Reg2, &d237)
					} else if d236.Type == tagInt && d236.Loc == LocReg {
						d237 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d236.Reg}
						ctx.BindReg(d236.Reg, &d237)
						ctx.BindReg(d236.Reg, &d237)
					} else {
						d237 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d236}, 1)
						d237.Type = tagInt
						ctx.BindReg(d237.Reg, &d237)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d237)
					ctx.EnsureDesc(&d237)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d237)
					ctx.StabilizeDescForControlFlow(&d237)
					ctx.FreeDesc(&d235)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d237)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d237)
					if d3.Loc == LocImm && d237.Loc == LocImm {
						d239 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d237.Imm.Int())}
					} else if d237.Loc == LocImm && d237.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r9 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r9, d3.Reg)
						d239 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d239)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d237)
						d239 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d237.Reg}
						ctx.BindReg(d237.Reg, &d239)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d237)
						scratch := ctx.AllocRegExcept(d237.Reg)
						ctx.EmitMovRegReg(scratch, d237.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d239 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d239)
					} else if d237.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d237.Imm.Int())
						d239 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d239)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d237)
						r10 := ctx.AllocRegExceptOperand(&d237, d3.Reg)
						ctx.EmitMovRegReg(r10, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r10, &d237)
						d239 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d239)
					}
					if d239.Loc == LocReg && d3.Loc == LocReg && d239.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d239)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDescsTogether(&d239, &d2)
					if d239.Loc == LocImm && d2.Loc == LocImm {
						d240 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d239.Imm.Int() > d2.Imm.Int())}
					} else if d2.Loc == LocImm {
						r11 := ctx.AllocReg()
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d239.Reg, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
							ctx.EmitCmpInt64(d239.Reg, ctx.ScratchReg)
						}
						d240 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedGreater}
						ctx.BindReg(r11, &d240)
					} else if d239.Loc == LocImm {
						r12 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d239.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
						d240 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedGreater}
						ctx.BindReg(r12, &d240)
					} else {
						r13 := ctx.AllocReg()
						ctx.EmitCmpInt64(d239.Reg, d2.Reg)
						d240 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedGreater}
						ctx.BindReg(r13, &d240)
					}
					ctx.FreeDesc(&d239)
					d241 = d240
					ctx.SyncDesc(&d241)
					if d241.Loc != LocImm && d241.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d241.Loc == LocImm {
						if d241.Imm.Bool() {
							return bbs[12].Render()
						}
						ctx.SyncDesc(&d237)
						if d237.Loc == LocReg || d237.Loc == LocFPReg {
							ctx.ProtectReg(d237.Reg)
						} else if d237.Loc == LocRegPair {
							ctx.ProtectReg(d237.Reg)
							ctx.ProtectReg(d237.Reg2)
						}
						d242 = d237
						if d242.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d242)
						ctx.EmitStoreToStack(d242, int32(bbs[13].PhiBase)+int32(0))
						if d237.Loc == LocReg || d237.Loc == LocFPReg {
							ctx.UnprotectReg(d237.Reg)
						} else if d237.Loc == LocRegPair {
							ctx.UnprotectReg(d237.Reg)
							ctx.UnprotectReg(d237.Reg2)
						}
						return bbs[13].Render()
					}
					lbl33 := ctx.ReserveLabel()
					ctx.EmitJump(d241.Condition, lbl13)
					ctx.EmitJmp(lbl33)
					ctx.FreeDesc(&d240)
					snap243 := d1
					snap244 := d2
					snap245 := d3
					snap246 := d4
					snap247 := d5
					snap248 := d6
					snap249 := d7
					snap250 := d8
					snap251 := d18
					snap252 := d19
					snap253 := d20
					snap254 := d22
					snap255 := d23
					snap256 := d24
					snap257 := d25
					snap258 := d26
					snap259 := d27
					snap260 := d28
					snap261 := d29
					snap262 := d30
					snap263 := d31
					snap264 := d54
					snap265 := d55
					snap266 := d56
					snap267 := d57
					snap268 := d58
					snap269 := d59
					snap270 := d60
					snap271 := d61
					snap272 := d62
					snap273 := d63
					snap274 := d64
					snap275 := d98
					snap276 := d133
					snap277 := d134
					snap278 := d135
					snap279 := d136
					snap280 := d137
					snap281 := d138
					snap282 := d139
					snap283 := d140
					snap284 := d183
					snap285 := d184
					snap286 := d185
					snap287 := d186
					snap288 := d187
					snap289 := d235
					snap290 := d236
					snap291 := d237
					snap292 := d238
					snap293 := d239
					snap294 := d240
					snap295 := d241
					snap296 := d242
					alloc297 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl33)
					ctx.SyncDesc(&d237)
					if d237.Loc == LocReg || d237.Loc == LocFPReg {
						ctx.ProtectReg(d237.Reg)
					} else if d237.Loc == LocRegPair {
						ctx.ProtectReg(d237.Reg)
						ctx.ProtectReg(d237.Reg2)
					}
					d298 = d237
					if d298.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d298)
					ctx.EmitStoreToStack(d298, int32(bbs[13].PhiBase)+int32(0))
					if d237.Loc == LocReg || d237.Loc == LocFPReg {
						ctx.UnprotectReg(d237.Reg)
					} else if d237.Loc == LocRegPair {
						ctx.UnprotectReg(d237.Reg)
						ctx.UnprotectReg(d237.Reg2)
					}
					ctx.EmitJmp(lbl14)
					ctx.RestoreAllocState(alloc297)
					d1 = snap243
					d2 = snap244
					d3 = snap245
					d4 = snap246
					d5 = snap247
					d6 = snap248
					d7 = snap249
					d8 = snap250
					d18 = snap251
					d19 = snap252
					d20 = snap253
					d22 = snap254
					d23 = snap255
					d24 = snap256
					d25 = snap257
					d26 = snap258
					d27 = snap259
					d28 = snap260
					d29 = snap261
					d30 = snap262
					d31 = snap263
					d54 = snap264
					d55 = snap265
					d56 = snap266
					d57 = snap267
					d58 = snap268
					d59 = snap269
					d60 = snap270
					d61 = snap271
					d62 = snap272
					d63 = snap273
					d64 = snap274
					d98 = snap275
					d133 = snap276
					d134 = snap277
					d135 = snap278
					d136 = snap279
					d137 = snap280
					d138 = snap281
					d139 = snap282
					d140 = snap283
					d183 = snap284
					d184 = snap285
					d185 = snap286
					d186 = snap287
					d187 = snap288
					d235 = snap289
					d236 = snap290
					d237 = snap291
					d238 = snap292
					d239 = snap293
					d240 = snap294
					d241 = snap295
					d242 = snap296
					if !bbs[13].Rendered {
						snap299 := d1
						snap300 := d2
						snap301 := d3
						snap302 := d4
						snap303 := d5
						snap304 := d6
						snap305 := d7
						snap306 := d8
						snap307 := d18
						snap308 := d19
						snap309 := d20
						snap310 := d22
						snap311 := d23
						snap312 := d24
						snap313 := d25
						snap314 := d26
						snap315 := d27
						snap316 := d28
						snap317 := d29
						snap318 := d30
						snap319 := d31
						snap320 := d54
						snap321 := d55
						snap322 := d56
						snap323 := d57
						snap324 := d58
						snap325 := d59
						snap326 := d60
						snap327 := d61
						snap328 := d62
						snap329 := d63
						snap330 := d64
						snap331 := d98
						snap332 := d133
						snap333 := d134
						snap334 := d135
						snap335 := d136
						snap336 := d137
						snap337 := d138
						snap338 := d139
						snap339 := d140
						snap340 := d183
						snap341 := d184
						snap342 := d185
						snap343 := d186
						snap344 := d187
						snap345 := d235
						snap346 := d236
						snap347 := d237
						snap348 := d238
						snap349 := d239
						snap350 := d240
						snap351 := d241
						snap352 := d242
						snap353 := d298
						alloc354 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc354)
						d1 = snap299
						d2 = snap300
						d3 = snap301
						d4 = snap302
						d5 = snap303
						d6 = snap304
						d7 = snap305
						d8 = snap306
						d18 = snap307
						d19 = snap308
						d20 = snap309
						d22 = snap310
						d23 = snap311
						d24 = snap312
						d25 = snap313
						d26 = snap314
						d27 = snap315
						d28 = snap316
						d29 = snap317
						d30 = snap318
						d31 = snap319
						d54 = snap320
						d55 = snap321
						d56 = snap322
						d57 = snap323
						d58 = snap324
						d59 = snap325
						d60 = snap326
						d61 = snap327
						d62 = snap328
						d63 = snap329
						d64 = snap330
						d98 = snap331
						d133 = snap332
						d134 = snap333
						d135 = snap334
						d136 = snap335
						d137 = snap336
						d138 = snap337
						d139 = snap338
						d140 = snap339
						d183 = snap340
						d184 = snap341
						d185 = snap342
						d186 = snap343
						d187 = snap344
						d235 = snap345
						d236 = snap346
						d237 = snap347
						d238 = snap348
						d239 = snap349
						d240 = snap350
						d241 = snap351
						d242 = snap352
						d298 = snap353
					}
					if !bbs[12].Rendered {
						return bbs[12].Render()
					}
					return result
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d355 = args[0]
					d355.ID = 0
					ctx.EnsureDesc(&d355)
					d356 = d355
					_ = d356
					ctx.StabilizeDescForControlFlow(&d356)
					phiBase357 = ctx.AllocStack(int32(16))
					d358 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase357) + int32(0)}
					_ = d358
					lbl34 := ctx.ReserveLabel()
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl35 := ctx.ReserveLabel()
					_ = lbl35
					bbpos_4_1 := int32(-1)
					_ = bbpos_4_1
					lbl36 := ctx.ReserveLabel()
					_ = lbl36
					bbpos_4_2 := int32(-1)
					_ = bbpos_4_2
					lbl37 := ctx.ReserveLabel()
					_ = lbl37
					bbpos_4_3 := int32(-1)
					_ = bbpos_4_3
					lbl38 := ctx.ReserveLabel()
					_ = lbl38
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl35)
					ctx.ResolveFixups()
					d358 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase357) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d359 = d356
					d359.ID = 0
					d360 = ctx.EmitGetTagDesc(&d359, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d360)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d360)
					if d360.Loc == LocImm {
						d361 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d360.Imm.Int()) == uint64(0x13))}
					} else {
						r14 := ctx.AllocRegExcept(d360.Reg)
						ctx.EmitCmpRegImm32(d360.Reg, 19)
						d361 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondEqual}
						ctx.BindReg(r14, &d361)
					}
					ctx.ReclaimUntrackedRegs()
					d362 = d361
					ctx.SyncDesc(&d362)
					if d362.Loc != LocImm && d362.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl39 := ctx.ReserveLabel()
					lbl40 := ctx.ReserveLabel()
					if d362.Loc == LocImm {
						if d362.Imm.Bool() {
							ctx.MarkLabel(lbl39)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase357)+int32(0))
							ctx.EmitJmp(lbl37)
						} else {
							ctx.MarkLabel(lbl40)
							ctx.EmitJmp(lbl38)
						}
					} else {
						ctx.EmitJump(d362.Condition, lbl39)
						ctx.EmitJmp(lbl40)
						ctx.FreeDesc(&d361)
						ctx.MarkLabel(lbl39)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase357)+int32(0))
						ctx.EmitJmp(lbl37)
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl38)
					}
					bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl37)
					ctx.ResolveFixups()
					d358 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase357) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r15 := ctx.AllocReg()
					ctx.EnsureDesc(&d358)
					ctx.EnsureDesc(&d358)
					if d358.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r15, d358)
					}
					ctx.EmitJmp(lbl34)
					bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl38)
					ctx.ResolveFixups()
					d358 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase357) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d360)
					if d360.Loc == LocImm {
						d363 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d360.Imm.Int()) == uint64(0x14))}
					} else {
						r16 := ctx.AllocRegExcept(d360.Reg)
						ctx.EmitCmpRegImm32(d360.Reg, 20)
						d363 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondEqual}
						ctx.BindReg(r16, &d363)
					}
					ctx.ReclaimUntrackedRegs()
					d364 = d363
					ctx.SyncDesc(&d364)
					if d364.Loc != LocImm && d364.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl41 := ctx.ReserveLabel()
					lbl42 := ctx.ReserveLabel()
					if d364.Loc == LocImm {
						if d364.Imm.Bool() {
							ctx.MarkLabel(lbl41)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase357)+int32(0))
							ctx.EmitJmp(lbl37)
						} else {
							ctx.MarkLabel(lbl42)
							ctx.EmitJmp(lbl36)
						}
					} else {
						ctx.EmitJump(d364.Condition, lbl41)
						ctx.EmitJmp(lbl42)
						ctx.FreeDesc(&d363)
						ctx.MarkLabel(lbl41)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase357)+int32(0))
						ctx.EmitJmp(lbl37)
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl36)
					}
					bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl36)
					ctx.ResolveFixups()
					d358 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase357) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d360)
					if d360.Loc == LocImm {
						d365 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d360.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d360.Reg, 26)
						r17 := ctx.AllocRegExcept(d360.Reg)
						ctx.EmitSetcc(r17, CondEqual)
						d365 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
						ctx.BindReg(r17, &d365)
					}
					ctx.EnsureDesc(&d365)
					ctx.EmitStoreToStack(d365, int32(phiBase357)+int32(0))
					ctx.StabilizeDescForControlFlow(&d365)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl37)
					ctx.MarkLabel(lbl34)
					d366 = JITValueDesc{Loc: LocReg, Reg: r15}
					ctx.BindReg(r15, &d366)
					ctx.BindReg(r15, &d366)
					ctx.FreeDesc(&d355)
					d367 = d366
					ctx.EnsureDesc(&d367)
					if d367.Loc != LocImm && d367.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d367.Loc == LocImm {
						if d367.Imm.Bool() {
							return bbs[18].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d367.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap368 := d1
						snap369 := d2
						snap370 := d3
						snap371 := d4
						snap372 := d5
						snap373 := d6
						snap374 := d7
						snap375 := d8
						snap376 := d18
						snap377 := d19
						snap378 := d20
						snap379 := d22
						snap380 := d23
						snap381 := d24
						snap382 := d25
						snap383 := d26
						snap384 := d27
						snap385 := d28
						snap386 := d29
						snap387 := d30
						snap388 := d31
						snap389 := d54
						snap390 := d55
						snap391 := d56
						snap392 := d57
						snap393 := d58
						snap394 := d59
						snap395 := d60
						snap396 := d61
						snap397 := d62
						snap398 := d63
						snap399 := d64
						snap400 := d98
						snap401 := d133
						snap402 := d134
						snap403 := d135
						snap404 := d136
						snap405 := d137
						snap406 := d138
						snap407 := d139
						snap408 := d140
						snap409 := d183
						snap410 := d184
						snap411 := d185
						snap412 := d186
						snap413 := d187
						snap414 := d235
						snap415 := d236
						snap416 := d237
						snap417 := d238
						snap418 := d239
						snap419 := d240
						snap420 := d241
						snap421 := d242
						snap422 := d298
						snap423 := d355
						snap424 := d356
						snap425 := d358
						snap426 := d359
						snap427 := d360
						snap428 := d361
						snap429 := d362
						snap430 := d363
						snap431 := d364
						snap432 := d365
						snap433 := d366
						snap434 := d367
						alloc435 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc435)
						d1 = snap368
						d2 = snap369
						d3 = snap370
						d4 = snap371
						d5 = snap372
						d6 = snap373
						d7 = snap374
						d8 = snap375
						d18 = snap376
						d19 = snap377
						d20 = snap378
						d22 = snap379
						d23 = snap380
						d24 = snap381
						d25 = snap382
						d26 = snap383
						d27 = snap384
						d28 = snap385
						d29 = snap386
						d30 = snap387
						d31 = snap388
						d54 = snap389
						d55 = snap390
						d56 = snap391
						d57 = snap392
						d58 = snap393
						d59 = snap394
						d60 = snap395
						d61 = snap396
						d62 = snap397
						d63 = snap398
						d64 = snap399
						d98 = snap400
						d133 = snap401
						d134 = snap402
						d135 = snap403
						d136 = snap404
						d137 = snap405
						d138 = snap406
						d139 = snap407
						d140 = snap408
						d183 = snap409
						d184 = snap410
						d185 = snap411
						d186 = snap412
						d187 = snap413
						d235 = snap414
						d236 = snap415
						d237 = snap416
						d238 = snap417
						d239 = snap418
						d240 = snap419
						d241 = snap420
						d242 = snap421
						d298 = snap422
						d355 = snap423
						d356 = snap424
						d358 = snap425
						d359 = snap426
						d360 = snap427
						d361 = snap428
						d362 = snap429
						d363 = snap430
						d364 = snap431
						d365 = snap432
						d366 = snap433
						d367 = snap434
					}
					if !bbs[18].Rendered {
						return bbs[18].Render()
					}
					return result
					ctx.FreeDesc(&d366)
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d3)
					ctx.SyncDesc(&d2)
					ctx.SyncDesc(&d3)
					if d2.Loc == LocImm && d3.Loc == LocImm {
						d436 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() - d3.Imm.Int())}
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d2)
						r18 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(r18, d2.Reg)
						d436 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d436)
					} else if d2.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d3)
						d436 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d436)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d2)
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d3.Imm.Int())
						d436 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d436)
					} else {
						ctx.EnsureDesc(&d2)
						ctx.SyncDesc(&d3)
						r19 := ctx.AllocRegExceptOperand(&d3, d2.Reg)
						ctx.EmitMovRegReg(r19, d2.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r19, &d3)
						d436 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d436)
					}
					if d436.Loc == LocReg && d2.Loc == LocReg && d436.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d436)
					ctx.EmitStoreToStack(d436, int32(bbs[13].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d436)
					return bbs[13].Render()
					return result
				}
				bbs[13].Render = func() JITValueDesc {
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
						return result
					}
					bbs[13].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d437 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < 0)}
					} else {
						r20 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpRegImm32(d4.Reg, 0)
						d437 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d437)
					}
					d438 = d437
					ctx.SyncDesc(&d438)
					if d438.Loc != LocImm && d438.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d438.Loc == LocImm {
						if d438.Imm.Bool() {
							return bbs[14].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitJump(d438.Condition, lbl15)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FreeDesc(&d437)
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap439 := d1
						snap440 := d2
						snap441 := d3
						snap442 := d4
						snap443 := d5
						snap444 := d6
						snap445 := d7
						snap446 := d8
						snap447 := d18
						snap448 := d19
						snap449 := d20
						snap450 := d22
						snap451 := d23
						snap452 := d24
						snap453 := d25
						snap454 := d26
						snap455 := d27
						snap456 := d28
						snap457 := d29
						snap458 := d30
						snap459 := d31
						snap460 := d54
						snap461 := d55
						snap462 := d56
						snap463 := d57
						snap464 := d58
						snap465 := d59
						snap466 := d60
						snap467 := d61
						snap468 := d62
						snap469 := d63
						snap470 := d64
						snap471 := d98
						snap472 := d133
						snap473 := d134
						snap474 := d135
						snap475 := d136
						snap476 := d137
						snap477 := d138
						snap478 := d139
						snap479 := d140
						snap480 := d183
						snap481 := d184
						snap482 := d185
						snap483 := d186
						snap484 := d187
						snap485 := d235
						snap486 := d236
						snap487 := d237
						snap488 := d238
						snap489 := d239
						snap490 := d240
						snap491 := d241
						snap492 := d242
						snap493 := d298
						snap494 := d355
						snap495 := d356
						snap496 := d358
						snap497 := d359
						snap498 := d360
						snap499 := d361
						snap500 := d362
						snap501 := d363
						snap502 := d364
						snap503 := d365
						snap504 := d366
						snap505 := d367
						snap506 := d436
						snap507 := d437
						snap508 := d438
						alloc509 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc509)
						d1 = snap439
						d2 = snap440
						d3 = snap441
						d4 = snap442
						d5 = snap443
						d6 = snap444
						d7 = snap445
						d8 = snap446
						d18 = snap447
						d19 = snap448
						d20 = snap449
						d22 = snap450
						d23 = snap451
						d24 = snap452
						d25 = snap453
						d26 = snap454
						d27 = snap455
						d28 = snap456
						d29 = snap457
						d30 = snap458
						d31 = snap459
						d54 = snap460
						d55 = snap461
						d56 = snap462
						d57 = snap463
						d58 = snap464
						d59 = snap465
						d60 = snap466
						d61 = snap467
						d62 = snap468
						d63 = snap469
						d64 = snap470
						d98 = snap471
						d133 = snap472
						d134 = snap473
						d135 = snap474
						d136 = snap475
						d137 = snap476
						d138 = snap477
						d139 = snap478
						d140 = snap479
						d183 = snap480
						d184 = snap481
						d185 = snap482
						d186 = snap483
						d187 = snap484
						d235 = snap485
						d236 = snap486
						d237 = snap487
						d238 = snap488
						d239 = snap489
						d240 = snap490
						d241 = snap491
						d242 = snap492
						d298 = snap493
						d355 = snap494
						d356 = snap495
						d358 = snap496
						d359 = snap497
						d360 = snap498
						d361 = snap499
						d362 = snap500
						d363 = snap501
						d364 = snap502
						d365 = snap503
						d366 = snap504
						d367 = snap505
						d436 = snap506
						d437 = snap507
						d438 = snap508
					}
					if !bbs[14].Rendered {
						return bbs[14].Render()
					}
					return result
					return result
				}
				bbs[14].Render = func() JITValueDesc {
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
						return result
					}
					bbs[14].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d510 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d511 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d510}, 2)
					ctx.EmitMovPairToResult(&d511, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[15].Render = func() JITValueDesc {
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
						return result
					}
					bbs[15].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d512 = args[0]
					d512.ID = 0
					ctx.EnsureDesc(&d512)
					d513 = d512
					_ = d513
					ctx.StabilizeDescForControlFlow(&d513)
					phiBase514 = ctx.AllocStack(int32(16))
					d515 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase514) + int32(0)}
					_ = d515
					lbl43 := ctx.ReserveLabel()
					bbpos_5_0 := int32(-1)
					_ = bbpos_5_0
					lbl44 := ctx.ReserveLabel()
					_ = lbl44
					bbpos_5_1 := int32(-1)
					_ = bbpos_5_1
					lbl45 := ctx.ReserveLabel()
					_ = lbl45
					bbpos_5_2 := int32(-1)
					_ = bbpos_5_2
					lbl46 := ctx.ReserveLabel()
					_ = lbl46
					bbpos_5_3 := int32(-1)
					_ = bbpos_5_3
					lbl47 := ctx.ReserveLabel()
					_ = lbl47
					bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl44)
					ctx.ResolveFixups()
					d515 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase514) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d516 = d513
					d516.ID = 0
					d517 = ctx.EmitGetTagDesc(&d516, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d517)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d517)
					if d517.Loc == LocImm {
						d518 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d517.Imm.Int()) == uint64(0x13))}
					} else {
						r21 := ctx.AllocRegExcept(d517.Reg)
						ctx.EmitCmpRegImm32(d517.Reg, 19)
						d518 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondEqual}
						ctx.BindReg(r21, &d518)
					}
					ctx.ReclaimUntrackedRegs()
					d519 = d518
					ctx.SyncDesc(&d519)
					if d519.Loc != LocImm && d519.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl48 := ctx.ReserveLabel()
					lbl49 := ctx.ReserveLabel()
					if d519.Loc == LocImm {
						if d519.Imm.Bool() {
							ctx.MarkLabel(lbl48)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase514)+int32(0))
							ctx.EmitJmp(lbl46)
						} else {
							ctx.MarkLabel(lbl49)
							ctx.EmitJmp(lbl47)
						}
					} else {
						ctx.EmitJump(d519.Condition, lbl48)
						ctx.EmitJmp(lbl49)
						ctx.FreeDesc(&d518)
						ctx.MarkLabel(lbl48)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase514)+int32(0))
						ctx.EmitJmp(lbl46)
						ctx.MarkLabel(lbl49)
						ctx.EmitJmp(lbl47)
					}
					bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl46)
					ctx.ResolveFixups()
					d515 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase514) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r22 := ctx.AllocReg()
					ctx.EnsureDesc(&d515)
					ctx.EnsureDesc(&d515)
					if d515.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r22, d515)
					}
					ctx.EmitJmp(lbl43)
					bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl47)
					ctx.ResolveFixups()
					d515 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase514) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d517)
					if d517.Loc == LocImm {
						d520 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d517.Imm.Int()) == uint64(0x14))}
					} else {
						r23 := ctx.AllocRegExcept(d517.Reg)
						ctx.EmitCmpRegImm32(d517.Reg, 20)
						d520 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondEqual}
						ctx.BindReg(r23, &d520)
					}
					ctx.ReclaimUntrackedRegs()
					d521 = d520
					ctx.SyncDesc(&d521)
					if d521.Loc != LocImm && d521.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl50 := ctx.ReserveLabel()
					lbl51 := ctx.ReserveLabel()
					if d521.Loc == LocImm {
						if d521.Imm.Bool() {
							ctx.MarkLabel(lbl50)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase514)+int32(0))
							ctx.EmitJmp(lbl46)
						} else {
							ctx.MarkLabel(lbl51)
							ctx.EmitJmp(lbl45)
						}
					} else {
						ctx.EmitJump(d521.Condition, lbl50)
						ctx.EmitJmp(lbl51)
						ctx.FreeDesc(&d520)
						ctx.MarkLabel(lbl50)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase514)+int32(0))
						ctx.EmitJmp(lbl46)
						ctx.MarkLabel(lbl51)
						ctx.EmitJmp(lbl45)
					}
					bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl45)
					ctx.ResolveFixups()
					d515 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase514) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d517)
					if d517.Loc == LocImm {
						d522 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d517.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d517.Reg, 26)
						r24 := ctx.AllocRegExcept(d517.Reg)
						ctx.EmitSetcc(r24, CondEqual)
						d522 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d522)
					}
					ctx.EnsureDesc(&d522)
					ctx.EmitStoreToStack(d522, int32(phiBase514)+int32(0))
					ctx.StabilizeDescForControlFlow(&d522)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl46)
					ctx.MarkLabel(lbl43)
					d523 = JITValueDesc{Loc: LocReg, Reg: r22}
					ctx.BindReg(r22, &d523)
					ctx.BindReg(r22, &d523)
					ctx.FreeDesc(&d512)
					d524 = d523
					ctx.EnsureDesc(&d524)
					if d524.Loc != LocImm && d524.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d524.Loc == LocImm {
						if d524.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitCmpRegImm32(d524.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap525 := d1
						snap526 := d2
						snap527 := d3
						snap528 := d4
						snap529 := d5
						snap530 := d6
						snap531 := d7
						snap532 := d8
						snap533 := d18
						snap534 := d19
						snap535 := d20
						snap536 := d22
						snap537 := d23
						snap538 := d24
						snap539 := d25
						snap540 := d26
						snap541 := d27
						snap542 := d28
						snap543 := d29
						snap544 := d30
						snap545 := d31
						snap546 := d54
						snap547 := d55
						snap548 := d56
						snap549 := d57
						snap550 := d58
						snap551 := d59
						snap552 := d60
						snap553 := d61
						snap554 := d62
						snap555 := d63
						snap556 := d64
						snap557 := d98
						snap558 := d133
						snap559 := d134
						snap560 := d135
						snap561 := d136
						snap562 := d137
						snap563 := d138
						snap564 := d139
						snap565 := d140
						snap566 := d183
						snap567 := d184
						snap568 := d185
						snap569 := d186
						snap570 := d187
						snap571 := d235
						snap572 := d236
						snap573 := d237
						snap574 := d238
						snap575 := d239
						snap576 := d240
						snap577 := d241
						snap578 := d242
						snap579 := d298
						snap580 := d355
						snap581 := d356
						snap582 := d358
						snap583 := d359
						snap584 := d360
						snap585 := d361
						snap586 := d362
						snap587 := d363
						snap588 := d364
						snap589 := d365
						snap590 := d366
						snap591 := d367
						snap592 := d436
						snap593 := d437
						snap594 := d438
						snap595 := d510
						snap596 := d511
						snap597 := d512
						snap598 := d513
						snap599 := d515
						snap600 := d516
						snap601 := d517
						snap602 := d518
						snap603 := d519
						snap604 := d520
						snap605 := d521
						snap606 := d522
						snap607 := d523
						snap608 := d524
						alloc609 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc609)
						d1 = snap525
						d2 = snap526
						d3 = snap527
						d4 = snap528
						d5 = snap529
						d6 = snap530
						d7 = snap531
						d8 = snap532
						d18 = snap533
						d19 = snap534
						d20 = snap535
						d22 = snap536
						d23 = snap537
						d24 = snap538
						d25 = snap539
						d26 = snap540
						d27 = snap541
						d28 = snap542
						d29 = snap543
						d30 = snap544
						d31 = snap545
						d54 = snap546
						d55 = snap547
						d56 = snap548
						d57 = snap549
						d58 = snap550
						d59 = snap551
						d60 = snap552
						d61 = snap553
						d62 = snap554
						d63 = snap555
						d64 = snap556
						d98 = snap557
						d133 = snap558
						d134 = snap559
						d135 = snap560
						d136 = snap561
						d137 = snap562
						d138 = snap563
						d139 = snap564
						d140 = snap565
						d183 = snap566
						d184 = snap567
						d185 = snap568
						d186 = snap569
						d187 = snap570
						d235 = snap571
						d236 = snap572
						d237 = snap573
						d238 = snap574
						d239 = snap575
						d240 = snap576
						d241 = snap577
						d242 = snap578
						d298 = snap579
						d355 = snap580
						d356 = snap581
						d358 = snap582
						d359 = snap583
						d360 = snap584
						d361 = snap585
						d362 = snap586
						d363 = snap587
						d364 = snap588
						d365 = snap589
						d366 = snap590
						d367 = snap591
						d436 = snap592
						d437 = snap593
						d438 = snap594
						d510 = snap595
						d511 = snap596
						d512 = snap597
						d513 = snap598
						d515 = snap599
						d516 = snap600
						d517 = snap601
						d518 = snap602
						d519 = snap603
						d520 = snap604
						d521 = snap605
						d522 = snap606
						d523 = snap607
						d524 = snap608
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d523)
					return result
				}
				bbs[16].Render = func() JITValueDesc {
					if bbs[16].Rendered {
						ctx.EmitJmp(lbl17)
						return result
					}
					bbs[16].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d610 = args[0]
					d610.ID = 0
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d4)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d4)
					if d3.Loc == LocImm && d4.Loc == LocImm {
						d611 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d4.Imm.Int())}
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r25 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r25, d3.Reg)
						d611 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d611)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						d611 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d611)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d611 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d611)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d611 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d611)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d4)
						r26 := ctx.AllocRegExceptOperand(&d4, d3.Reg)
						ctx.EmitMovRegReg(r26, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r26, &d4)
						d611 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
						ctx.BindReg(r26, &d611)
					}
					if d611.Loc == LocReg && d3.Loc == LocReg && d611.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					d610 = JITPrepareScmerGoArg(ctx, d610)
					if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d611.Loc == LocRegPair || d611.Loc == LocStackPair || d611.Loc == LocRegTriple || d611.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d610)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d611)
					d612 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d610, d3, d611}, 2)
					d612.NoHeapPointer = false
					ctx.BindReg(d612.Reg, &d612)
					ctx.BindReg(d612.Reg2, &d612)
					ctx.FreeDesc(&d610)
					ctx.FreeDesc(&d611)
					ctx.SyncDesc(&d612)
					if d612.Loc == LocRegPair || d612.Loc == LocStackPair || d612.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d612, &result)
						result.Type = d612.Type
					} else {
						switch d612.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d612)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d612)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d612)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d612, &result)
							result.Type = d612.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[17].Render = func() JITValueDesc {
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
						return result
					}
					bbs[17].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d4)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d4)
					if d3.Loc == LocImm && d4.Loc == LocImm {
						d613 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d4.Imm.Int())}
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r27 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r27, d3.Reg)
						d613 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
						ctx.BindReg(r27, &d613)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						d613 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d613)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d613 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d613)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d613 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d613)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d4)
						r28 := ctx.AllocRegExceptOperand(&d4, d3.Reg)
						ctx.EmitMovRegReg(r28, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r28, &d4)
						d613 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r28}
						ctx.BindReg(r28, &d613)
					}
					if d613.Loc == LocReg && d3.Loc == LocReg && d613.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d613)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d613)
					if d613.Loc == LocImm && d3.Loc == LocImm {
						d615 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d613.Imm.Int() - d3.Imm.Int())}
					} else {
						r29 := ctx.AllocReg()
						if d613.Loc == LocImm {
							ctx.EmitMovRegImm64(r29, uint64(d613.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r29, d613.Reg)
						}
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitSubInt64(r29, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r29, d3.Reg)
						}
						d615 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r29}
						ctx.BindReg(r29, &d615)
					}
					r30 := ctx.EmitSliceDataAfterLow(&d1, &d3, 1)
					d616 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
					ctx.BindReg(r30, &d616)
					ctx.BindReg(r30, &d616)
					var r31 Reg
					var r32 Reg
					ctx.SyncDesc(&d616)
					ctx.EnsureDesc(&d616)
					if d616.Loc == LocImm {
						r31 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r31, uint64(d616.Imm.Int()))
					} else {
						r31 = d616.Reg
					}
					ctx.ProtectReg(r31)
					ctx.SyncDesc(&d615)
					ctx.EnsureDesc(&d615)
					if d615.Loc == LocImm {
						r32 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r32, uint64(d615.Imm.Int()))
					} else {
						r32 = d615.Reg
					}
					ctx.ProtectReg(r32)
					ctx.UnprotectReg(r32)
					ctx.UnprotectReg(r31)
					d617 = JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
					ctx.BindReg(r31, &d617)
					ctx.BindReg(r32, &d617)
					ctx.BindReg(r31, &d617)
					ctx.BindReg(r32, &d617)
					ctx.FreeDesc(&d613)
					ctx.EnsureDesc(&d617)
					d618 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d617}, 2)
					ctx.EmitMovPairToResult(&d618, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[18].Render = func() JITValueDesc {
					if bbs[18].Rendered {
						ctx.EmitJmp(lbl19)
						return result
					}
					bbs[18].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					d619 = args[0]
					d619.ID = 0
					d619 = JITPrepareScmerGoArg(ctx, d619)
					if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d619)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d2)
					d620 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d619, d3, d2}, 2)
					d620.NoHeapPointer = false
					ctx.BindReg(d620.Reg, &d620)
					ctx.BindReg(d620.Reg2, &d620)
					ctx.FreeDesc(&d619)
					ctx.SyncDesc(&d620)
					if d620.Loc == LocRegPair || d620.Loc == LocStackPair || d620.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d620, &result)
						result.Type = d620.Type
					} else {
						switch d620.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d620)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d620)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d620)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d620, &result)
							result.Type = d620.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[19].Render = func() JITValueDesc {
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
						return result
					}
					bbs[19].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(48)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
						d621 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d621)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d621)
					if d621.Loc == LocImm && d3.Loc == LocImm {
						d623 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d621.Imm.Int() - d3.Imm.Int())}
					} else {
						r33 := ctx.AllocReg()
						if d621.Loc == LocImm {
							ctx.EmitMovRegImm64(r33, uint64(d621.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r33, d621.Reg)
						}
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitSubInt64(r33, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r33, d3.Reg)
						}
						d623 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r33}
						ctx.BindReg(r33, &d623)
					}
					r34 := ctx.EmitSliceDataAfterLow(&d1, &d3, 1)
					d624 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r34}
					ctx.BindReg(r34, &d624)
					ctx.BindReg(r34, &d624)
					var r35 Reg
					var r36 Reg
					ctx.SyncDesc(&d624)
					ctx.EnsureDesc(&d624)
					if d624.Loc == LocImm {
						r35 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r35, uint64(d624.Imm.Int()))
					} else {
						r35 = d624.Reg
					}
					ctx.ProtectReg(r35)
					ctx.SyncDesc(&d623)
					ctx.EnsureDesc(&d623)
					if d623.Loc == LocImm {
						r36 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r36, uint64(d623.Imm.Int()))
					} else {
						r36 = d623.Reg
					}
					ctx.ProtectReg(r36)
					ctx.UnprotectReg(r36)
					ctx.UnprotectReg(r35)
					d625 = JITValueDesc{Loc: LocRegPair, Reg: r35, Reg2: r36}
					ctx.BindReg(r35, &d625)
					ctx.BindReg(r36, &d625)
					ctx.BindReg(r35, &d625)
					ctx.BindReg(r36, &d625)
					ctx.EnsureDesc(&d625)
					d626 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d625}, 2)
					ctx.EmitMovPairToResult(&d626, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 106,
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
			if a[0].IsBufferString() {
				return NewInt(int64(compressedTextLen(a[0])))
			}
			if a[0].IsCString() {
				return NewInt(int64(auxVal(a[0].aux) & CStringLengthMask))
			}
			if a[0].IsBString() {
				return NewInt(int64(compressedTextLen(a[0])))
			}
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d1 = d0
					d1.ID = 0
					d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x1a))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 26)
						d3 = ctx.DeferBooleanFlags(r0, CondEqual)
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.FreeDesc(&d0)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap5 := d0
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						alloc10 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc10)
						d0 = snap5
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d11 = args[0]
					d11.ID = 0
					d11 = JITPrepareScmerGoArg(ctx, d11)
					ctx.SyncDesc(&d11)
					d12 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d11}, 1)
					d12.NoHeapPointer = true
					ctx.BindReg(d12.Reg, &d12)
					ctx.FreeDesc(&d11)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						ctx.EmitMakeInt(result, d12)
					} else {
						ctx.EmitMovToReg(result.Reg2, d12)
						d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d14)
						if d12.Loc == LocReg && d12.Reg != result.Reg2 {
							ctx.FreeReg(d12.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d15 = args[0]
					d15.ID = 0
					d15 = JITPrepareScmerGoArg(ctx, d15)
					ctx.SyncDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsCString), []JITValueDesc{d15}, 1)
					d16.NoHeapPointer = true
					ctx.EmitAndRegImm32(d16.Reg, 1)
					d16.Type = tagBool
					ctx.BindReg(d16.Reg, &d16)
					ctx.FreeDesc(&d15)
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap18 := d0
						snap19 := d1
						snap20 := d2
						snap21 := d3
						snap22 := d4
						snap23 := d11
						snap24 := d12
						snap25 := d13
						snap26 := d14
						snap27 := d15
						snap28 := d16
						snap29 := d17
						alloc30 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc30)
						d0 = snap18
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d4 = snap22
						d11 = snap23
						d12 = snap24
						d13 = snap25
						d14 = snap26
						d15 = snap27
						d16 = snap28
						d17 = snap29
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d16)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d31 = args[0]
					d31.ID = 0
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocImm {
						_, auxWord := d31.Imm.RawWords()
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d31.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r1 := ctx.AllocReg()
						ctx.EmitMovRegReg(r1, d31.Reg2)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d32)
					}
					ctx.EnsureDesc(&d32)
					d33 = d32
					_ = d33
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d33.Imm.Int()) >> 8))}
					} else {
						r2 := ctx.AllocRegExcept(d33.Reg)
						ctx.EmitMovRegReg(r2, d33.Reg)
						ctx.EmitShrRegImm8(r2, 8)
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d34)
					}
					if d34.Loc == LocReg && d33.Loc == LocReg && d34.Reg == d33.Reg {
						ctx.TransferReg(d33.Reg)
						d33.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d34.Imm.Int() & 4398046511103)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0x3ffffffffff)
						ctx.EmitAndInt64(d34.Reg, ctx.ScratchReg)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg}
						ctx.BindReg(d34.Reg, &d35)
					}
					if d35.Loc == LocReg && d34.Loc == LocReg && d35.Reg == d34.Reg {
						ctx.TransferReg(d34.Reg)
						d34.Loc = LocNone
					}
					ctx.FreeDesc(&d34)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d35.Imm.Int()))))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegReg(r3, d35.Reg)
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d36)
					}
					ctx.FreeDesc(&d35)
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm {
						ctx.EmitMakeInt(result, d36)
					} else {
						ctx.EmitMovToReg(result.Reg2, d36)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d37)
						if d36.Loc == LocReg && d36.Reg != result.Reg2 {
							ctx.FreeReg(d36.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d38 = args[0]
					d38.ID = 0
					d38 = JITPrepareScmerGoArg(ctx, d38)
					ctx.SyncDesc(&d38)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsBString), []JITValueDesc{d38}, 1)
					d39.NoHeapPointer = true
					ctx.EmitAndRegImm32(d39.Reg, 1)
					d39.Type = tagBool
					ctx.BindReg(d39.Reg, &d39)
					ctx.FreeDesc(&d38)
					d40 = d39
					ctx.EnsureDesc(&d40)
					if d40.Loc != LocImm && d40.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d40.Loc == LocImm {
						if d40.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d40.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap41 := d0
						snap42 := d1
						snap43 := d2
						snap44 := d3
						snap45 := d4
						snap46 := d11
						snap47 := d12
						snap48 := d13
						snap49 := d14
						snap50 := d15
						snap51 := d16
						snap52 := d17
						snap53 := d31
						snap54 := d32
						snap55 := d33
						snap56 := d34
						snap57 := d35
						snap58 := d36
						snap59 := d37
						snap60 := d38
						snap61 := d39
						snap62 := d40
						alloc63 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc63)
						d0 = snap41
						d1 = snap42
						d2 = snap43
						d3 = snap44
						d4 = snap45
						d11 = snap46
						d12 = snap47
						d13 = snap48
						d14 = snap49
						d15 = snap50
						d16 = snap51
						d17 = snap52
						d31 = snap53
						d32 = snap54
						d33 = snap55
						d34 = snap56
						d35 = snap57
						d36 = snap58
						d37 = snap59
						d38 = snap60
						d39 = snap61
						d40 = snap62
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d39)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d64 = args[0]
					d64.ID = 0
					d64 = JITPrepareScmerGoArg(ctx, d64)
					ctx.SyncDesc(&d64)
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d64}, 1)
					d65.NoHeapPointer = true
					ctx.BindReg(d65.Reg, &d65)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						ctx.EmitMakeInt(result, d65)
					} else {
						ctx.EmitMovToReg(result.Reg2, d65)
						d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d67)
						if d65.Loc == LocReg && d65.Reg != result.Reg2 {
							ctx.FreeReg(d65.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d68 = args[0]
					d68.ID = 0
					d70 = d68
					ctx.SyncDesc(&d70)
					if d70.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d70.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d70.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d70 = tmpScalar
					}
					d70 = JITPrepareScmerGoArg(ctx, d70)
					if d70.Loc != LocRegPair && d70.Loc != LocStackPair && d70.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d70}, 2)
					ctx.FreeDesc(&d68)
					if d69.SliceSizeKnown {
						d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d69.KnownSliceLen))}
					} else if d69.Loc == LocImm {
						d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d69.Imm.String())))}
					} else if d69.Loc == LocStackTriple {
						d71 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d69.StackOff + 8, NoHeapPointer: true}
					} else if d69.Loc == LocStackPair {
						d71 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d69.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d69)
						if d69.Loc == LocRegPair || d69.Loc == LocRegTriple {
							d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg2, ID: 0}
						} else if d69.Loc == LocReg {
							d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d71)
					if d71.Loc == LocImm {
						ctx.EmitMakeInt(result, d71)
					} else {
						ctx.EmitMovToReg(result.Reg2, d71)
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d73)
						if d71.Loc == LocReg && d71.Reg != result.Reg2 {
							ctx.FreeReg(d71.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost:  44,
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strlike",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			pattern := String(a[1])
			collation := "utf8mb4_general_ci"
			if len(a) > 2 {
				collation = strings.ToLower(String(a[2]))
			}
			if isCompressedText(a[0]) {
				if matched, ok := strLikeCString(a[0], pattern, collation); ok {
					return NewBool(matched)
				}
			}
			return NewBool(StrLikeCollation(String(a[0]), pattern, collation))
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
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
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
				var phiBase73 int32
				_ = phiBase73
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
				var d118 JITValueDesc
				_ = d118
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [9]BBDescriptor
				bbs[5].PhiBase = int32(phiBase0) + int32(0)
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						snap10 := d5
						alloc11 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc11)
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
						d5 = snap10
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d3)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d12 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
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
							ctx.EmitMovPairToResult(&d12, &result)
							result.Type = d12.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d13 = args[1]
					d13.ID = 0
					d15 = d13
					ctx.SyncDesc(&d15)
					if d15.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d15.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d15.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d15 = tmpScalar
					}
					d15 = JITPrepareScmerGoArg(ctx, d15)
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair && d15.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d15}, 2)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.FreeDesc(&d13)
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d16.Reg, 2)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d17)
					}
					ctx.FreeDesc(&d16)
					d18 = d17
					ctx.SyncDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[4].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
						return bbs[5].Render()
					}
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d18.Condition, lbl5)
					ctx.EmitJmp(lbl10)
					ctx.FreeDesc(&d17)
					snap19 := d1
					snap20 := d2
					snap21 := d3
					snap22 := d4
					snap23 := d5
					snap24 := d12
					snap25 := d13
					snap26 := d14
					snap27 := d15
					snap28 := d16
					snap29 := d17
					snap30 := d18
					alloc31 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("utf8mb4_general_ci")}, int32(bbs[5].PhiBase)+int32(0))
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc31)
					d1 = snap19
					d2 = snap20
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d12 = snap24
					d13 = snap25
					d14 = snap26
					d15 = snap27
					d16 = snap28
					d17 = snap29
					d18 = snap30
					if !bbs[5].Rendered {
						snap32 := d1
						snap33 := d2
						snap34 := d3
						snap35 := d4
						snap36 := d5
						snap37 := d12
						snap38 := d13
						snap39 := d14
						snap40 := d15
						snap41 := d16
						snap42 := d17
						snap43 := d18
						alloc44 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc44)
						d1 = snap32
						d2 = snap33
						d3 = snap34
						d4 = snap35
						d5 = snap36
						d12 = snap37
						d13 = snap38
						d14 = snap39
						d15 = snap40
						d16 = snap41
						d17 = snap42
						d18 = snap43
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d45 = args[1]
					d45.ID = 0
					d47 = d45
					d47.ID = 0
					d46 = ctx.EmitTagEqualsBorrowed(&d47, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d45)
					d48 = d46
					ctx.EnsureDesc(&d48)
					if d48.Loc != LocImm && d48.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d48.Loc == LocImm {
						if d48.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d48.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap49 := d1
						snap50 := d2
						snap51 := d3
						snap52 := d4
						snap53 := d5
						snap54 := d12
						snap55 := d13
						snap56 := d14
						snap57 := d15
						snap58 := d16
						snap59 := d17
						snap60 := d18
						snap61 := d45
						snap62 := d46
						snap63 := d47
						snap64 := d48
						alloc65 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc65)
						d1 = snap49
						d2 = snap50
						d3 = snap51
						d4 = snap52
						d5 = snap53
						d12 = snap54
						d13 = snap55
						d14 = snap56
						d15 = snap57
						d16 = snap58
						d17 = snap59
						d18 = snap60
						d45 = snap61
						d46 = snap62
						d47 = snap63
						d48 = snap64
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d46)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					ctx.FreeDesc(&d66)
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
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d67)
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d67}, 2)
					d69.NoHeapPointer = false
					ctx.BindReg(d69.Reg, &d69)
					ctx.BindReg(d69.Reg2, &d69)
					ctx.StabilizeDescForControlFlow(&d69)
					ctx.SyncDesc(&d69)
					if d69.Loc == LocReg || d69.Loc == LocFPReg {
						ctx.ProtectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.ProtectReg(d69.Reg)
						ctx.ProtectReg(d69.Reg2)
					}
					d70 = d69
					if d70.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d70)
					if d70.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d70, int32(bbs[5].PhiBase)+int32(0), 2)
					} else if d70.Loc == LocInputPair {
						ctx.EnsureDesc(&d70)
						ctx.EmitStoreScmerToStack(d70, int32(bbs[5].PhiBase)+int32(0))
					} else if d70.Loc == LocRegPair || d70.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d70, int32(bbs[5].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d70)
						ctx.EmitStoreToStack(d70, int32(bbs[5].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[5].PhiBase)+int32(0))+8)
					}
					if d69.Loc == LocReg || d69.Loc == LocFPReg {
						ctx.UnprotectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.UnprotectReg(d69.Reg)
						ctx.UnprotectReg(d69.Reg2)
					}
					return bbs[5].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d71 = args[0]
					d71.ID = 0
					ctx.EnsureDesc(&d71)
					d72 = d71
					_ = d72
					ctx.StabilizeDescForControlFlow(&d72)
					phiBase73 = ctx.AllocStack(int32(16))
					d74 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase73) + int32(0)}
					_ = d74
					lbl11 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d74 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase73) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d75 = d72
					d75.ID = 0
					d76 = ctx.EmitGetTagDesc(&d75, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d76)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm {
						d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d76.Imm.Int()) == uint64(0x13))}
					} else {
						r1 := ctx.AllocRegExcept(d76.Reg)
						ctx.EmitCmpRegImm32(d76.Reg, 19)
						d77 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d77)
					}
					ctx.ReclaimUntrackedRegs()
					d78 = d77
					ctx.SyncDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase73)+int32(0))
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl15)
						}
					} else {
						ctx.EmitJump(d78.Condition, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.FreeDesc(&d77)
						ctx.MarkLabel(lbl16)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase73)+int32(0))
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d74 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase73) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r2 := ctx.AllocReg()
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d74)
					}
					ctx.EmitJmp(lbl11)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d74 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase73) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm {
						d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d76.Imm.Int()) == uint64(0x14))}
					} else {
						r3 := ctx.AllocRegExcept(d76.Reg)
						ctx.EmitCmpRegImm32(d76.Reg, 20)
						d79 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d79)
					}
					ctx.ReclaimUntrackedRegs()
					d80 = d79
					ctx.SyncDesc(&d80)
					if d80.Loc != LocImm && d80.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					if d80.Loc == LocImm {
						if d80.Imm.Bool() {
							ctx.MarkLabel(lbl18)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase73)+int32(0))
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitJump(d80.Condition, lbl18)
						ctx.EmitJmp(lbl19)
						ctx.FreeDesc(&d79)
						ctx.MarkLabel(lbl18)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase73)+int32(0))
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl13)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d74 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase73) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d76.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d76.Reg, 26)
						r4 := ctx.AllocRegExcept(d76.Reg)
						ctx.EmitSetcc(r4, CondEqual)
						d81 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d81)
					}
					ctx.EnsureDesc(&d81)
					ctx.EmitStoreToStack(d81, int32(phiBase73)+int32(0))
					ctx.StabilizeDescForControlFlow(&d81)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl11)
					d82 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d82)
					ctx.BindReg(r2, &d82)
					ctx.FreeDesc(&d71)
					d83 = d82
					ctx.EnsureDesc(&d83)
					if d83.Loc != LocImm && d83.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d83.Loc == LocImm {
						if d83.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d83.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap84 := d1
						snap85 := d2
						snap86 := d3
						snap87 := d4
						snap88 := d5
						snap89 := d12
						snap90 := d13
						snap91 := d14
						snap92 := d15
						snap93 := d16
						snap94 := d17
						snap95 := d18
						snap96 := d45
						snap97 := d46
						snap98 := d47
						snap99 := d48
						snap100 := d66
						snap101 := d67
						snap102 := d68
						snap103 := d69
						snap104 := d70
						snap105 := d71
						snap106 := d72
						snap107 := d74
						snap108 := d75
						snap109 := d76
						snap110 := d77
						snap111 := d78
						snap112 := d79
						snap113 := d80
						snap114 := d81
						snap115 := d82
						snap116 := d83
						alloc117 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc117)
						d1 = snap84
						d2 = snap85
						d3 = snap86
						d4 = snap87
						d5 = snap88
						d12 = snap89
						d13 = snap90
						d14 = snap91
						d15 = snap92
						d16 = snap93
						d17 = snap94
						d18 = snap95
						d45 = snap96
						d46 = snap97
						d47 = snap98
						d48 = snap99
						d66 = snap100
						d67 = snap101
						d68 = snap102
						d69 = snap103
						d70 = snap104
						d71 = snap105
						d72 = snap106
						d74 = snap107
						d75 = snap108
						d76 = snap109
						d77 = snap110
						d78 = snap111
						d79 = snap112
						d80 = snap113
						d81 = snap114
						d82 = snap115
						d83 = snap116
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d82)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d118 = args[0]
					d118.ID = 0
					d118 = JITPrepareScmerGoArg(ctx, d118)
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
					if d14.Loc != LocRegPair && d14.Loc != LocStackPair && d14.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strLikeCString arg1)")
					}
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
						panic("jit: generic call arg expects 2-word value (strLikeCString arg2)")
					}
					ctx.SyncDesc(&d118)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d1)
					callResults119 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeCString), []JITValueDesc{d118, d14, d1}, []uint8{1, 1}, []uint8{0, 0})
					d120 = callResults119[0]
					_ = d120
					d121 = callResults119[1]
					_ = d121
					ctx.FreeDesc(&d118)
					ctx.StabilizeDescForControlFlow(&d120)
					d122 = d121
					ctx.EnsureDesc(&d122)
					if d122.Loc != LocImm && d122.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d122.Loc == LocImm {
						if d122.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d122.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap123 := d1
						snap124 := d2
						snap125 := d3
						snap126 := d4
						snap127 := d5
						snap128 := d12
						snap129 := d13
						snap130 := d14
						snap131 := d15
						snap132 := d16
						snap133 := d17
						snap134 := d18
						snap135 := d45
						snap136 := d46
						snap137 := d47
						snap138 := d48
						snap139 := d66
						snap140 := d67
						snap141 := d68
						snap142 := d69
						snap143 := d70
						snap144 := d71
						snap145 := d72
						snap146 := d74
						snap147 := d75
						snap148 := d76
						snap149 := d77
						snap150 := d78
						snap151 := d79
						snap152 := d80
						snap153 := d81
						snap154 := d82
						snap155 := d83
						snap156 := d118
						snap157 := d120
						snap158 := d121
						snap159 := d122
						alloc160 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc160)
						d1 = snap123
						d2 = snap124
						d3 = snap125
						d4 = snap126
						d5 = snap127
						d12 = snap128
						d13 = snap129
						d14 = snap130
						d15 = snap131
						d16 = snap132
						d17 = snap133
						d18 = snap134
						d45 = snap135
						d46 = snap136
						d47 = snap137
						d48 = snap138
						d66 = snap139
						d67 = snap140
						d68 = snap141
						d69 = snap142
						d70 = snap143
						d71 = snap144
						d72 = snap145
						d74 = snap146
						d75 = snap147
						d76 = snap148
						d77 = snap149
						d78 = snap150
						d79 = snap151
						d80 = snap152
						d81 = snap153
						d82 = snap154
						d83 = snap155
						d118 = snap156
						d120 = snap157
						d121 = snap158
						d122 = snap159
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d121)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d161 = args[0]
					d161.ID = 0
					d163 = d161
					ctx.SyncDesc(&d163)
					if d163.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d163.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d163.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d163 = tmpScalar
					}
					d163 = JITPrepareScmerGoArg(ctx, d163)
					if d163.Loc != LocRegPair && d163.Loc != LocStackPair && d163.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d162 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d163}, 2)
					ctx.FreeDesc(&d161)
					ctx.EnsureDesc(&d162)
					if d162.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d162.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d162.Imm)
						ptrWord, _ := d162.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d162.Imm.String())))
						d162 = tmpPair
					} else if d162.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d162.Type, Reg: ctx.AllocRegExcept(d162.Reg), Reg2: ctx.AllocRegExcept(d162.Reg)}
						switch d162.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d162)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d162)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d162)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d162)
						d162 = tmpPair
					}
					if d162.Loc != LocRegPair && d162.Loc != LocStackPair && d162.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg0)")
					}
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
					if d14.Loc != LocRegPair && d14.Loc != LocStackPair && d14.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (StrLikeCollation arg1)")
					}
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
					ctx.SyncDesc(&d162)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d1)
					d164 = ctx.EmitGoCallScalar(GoFuncAddr(StrLikeCollation), []JITValueDesc{d162, d14, d1}, 1)
					d164.NoHeapPointer = true
					ctx.EmitAndRegImm32(d164.Reg, 1)
					d164.Type = tagBool
					ctx.BindReg(d164.Reg, &d164)
					ctx.SyncDesc(&d164)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d164) {
						return d164
					}
					if d164.Loc == LocImm {
						ctx.EmitMakeBool(result, d164)
					} else {
						ctx.EmitMovToReg(result.Reg2, d164)
						d165 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d165)
						if d164.Loc == LocReg && d164.Reg != result.Reg2 {
							ctx.FreeReg(d164.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d120)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d120) {
						return d120
					}
					if d120.Loc == LocImm {
						ctx.EmitMakeBool(result, d120)
					} else {
						ctx.EmitMovToReg(result.Reg2, d120)
						d166 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d166)
						if d120.Loc == LocReg && d120.Reg != result.Reg2 {
							ctx.FreeReg(d120.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 49,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strlike_cs",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			pattern := String(a[1])
			if isCompressedText(a[0]) {
				if matched, ok := strLikeCString(a[0], pattern, "bin"); ok {
					return NewBool(matched)
				}
			}
			return NewBool(StrLike(String(a[0]), pattern))
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
				var phiBase15 int32
				_ = phiBase15
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
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					d12 = d10
					ctx.SyncDesc(&d12)
					if d12.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d12.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d12.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d12 = tmpScalar
					}
					d12 = JITPrepareScmerGoArg(ctx, d12)
					if d12.Loc != LocRegPair && d12.Loc != LocStackPair && d12.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d12}, 2)
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.FreeDesc(&d10)
					d13 = args[0]
					d13.ID = 0
					ctx.EnsureDesc(&d13)
					d14 = d13
					_ = d14
					ctx.StabilizeDescForControlFlow(&d14)
					phiBase15 = ctx.AllocStack(int32(16))
					d16 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase15) + int32(0)}
					_ = d16
					lbl8 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d16 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase15) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d17 = d14
					d17.ID = 0
					d18 = ctx.EmitGetTagDesc(&d17, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d18.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpRegImm32(d18.Reg, 19)
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d19)
					}
					ctx.ReclaimUntrackedRegs()
					d20 = d19
					ctx.SyncDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase15)+int32(0))
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						}
					} else {
						ctx.EmitJump(d20.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d19)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase15)+int32(0))
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d16 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase15) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d16)
					}
					ctx.EmitJmp(lbl8)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d16 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase15) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d18.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpRegImm32(d18.Reg, 20)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d21)
					}
					ctx.ReclaimUntrackedRegs()
					d22 = d21
					ctx.SyncDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase15)+int32(0))
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d22.Condition, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.FreeDesc(&d21)
						ctx.MarkLabel(lbl15)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase15)+int32(0))
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d16 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase15) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d18.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d18.Reg, 26)
						r3 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d23)
					}
					ctx.EnsureDesc(&d23)
					ctx.EmitStoreToStack(d23, int32(phiBase15)+int32(0))
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl8)
					d24 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d24)
					ctx.BindReg(r1, &d24)
					ctx.FreeDesc(&d13)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap26 := d0
						snap27 := d1
						snap28 := d2
						snap29 := d3
						snap30 := d9
						snap31 := d10
						snap32 := d11
						snap33 := d12
						snap34 := d13
						snap35 := d14
						snap36 := d16
						snap37 := d17
						snap38 := d18
						snap39 := d19
						snap40 := d20
						snap41 := d21
						snap42 := d22
						snap43 := d23
						snap44 := d24
						snap45 := d25
						alloc46 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc46)
						d0 = snap26
						d1 = snap27
						d2 = snap28
						d3 = snap29
						d9 = snap30
						d10 = snap31
						d11 = snap32
						d12 = snap33
						d13 = snap34
						d14 = snap35
						d16 = snap36
						d17 = snap37
						d18 = snap38
						d19 = snap39
						d20 = snap40
						d21 = snap41
						d22 = snap42
						d23 = snap43
						d24 = snap44
						d25 = snap45
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d24)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d47 = args[1]
					d47.ID = 0
					d49 = d47
					d49.ID = 0
					d48 = ctx.EmitTagEqualsBorrowed(&d49, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d47)
					d50 = d48
					ctx.EnsureDesc(&d50)
					if d50.Loc != LocImm && d50.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d50.Loc == LocImm {
						if d50.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d50.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap51 := d0
						snap52 := d1
						snap53 := d2
						snap54 := d3
						snap55 := d9
						snap56 := d10
						snap57 := d11
						snap58 := d12
						snap59 := d13
						snap60 := d14
						snap61 := d16
						snap62 := d17
						snap63 := d18
						snap64 := d19
						snap65 := d20
						snap66 := d21
						snap67 := d22
						snap68 := d23
						snap69 := d24
						snap70 := d25
						snap71 := d47
						snap72 := d48
						snap73 := d49
						snap74 := d50
						alloc75 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc75)
						d0 = snap51
						d1 = snap52
						d2 = snap53
						d3 = snap54
						d9 = snap55
						d10 = snap56
						d11 = snap57
						d12 = snap58
						d13 = snap59
						d14 = snap60
						d16 = snap61
						d17 = snap62
						d18 = snap63
						d19 = snap64
						d20 = snap65
						d21 = snap66
						d22 = snap67
						d23 = snap68
						d24 = snap69
						d25 = snap70
						d47 = snap71
						d48 = snap72
						d49 = snap73
						d50 = snap74
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d48)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d76 = args[0]
					d76.ID = 0
					d76 = JITPrepareScmerGoArg(ctx, d76)
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
						panic("jit: generic call arg expects 2-word value (strLikeCString arg1)")
					}
					d77 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("bin")}
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d77.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d77.Imm)
						ptrWord, _ := d77.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d77.Imm.String())))
						d77 = tmpPair
					} else if d77.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d77.Type, Reg: ctx.AllocRegExcept(d77.Reg), Reg2: ctx.AllocRegExcept(d77.Reg)}
						switch d77.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d77)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d77)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d77)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d77)
						d77 = tmpPair
					}
					if d77.Loc != LocRegPair && d77.Loc != LocStackPair && d77.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strLikeCString arg2)")
					}
					ctx.SyncDesc(&d76)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d77)
					callResults78 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeCString), []JITValueDesc{d76, d11, d77}, []uint8{1, 1}, []uint8{0, 0})
					ctx.FreeDesc(&d77)
					d79 = callResults78[0]
					_ = d79
					d80 = callResults78[1]
					_ = d80
					ctx.FreeDesc(&d76)
					ctx.StabilizeDescForControlFlow(&d79)
					d81 = d80
					ctx.EnsureDesc(&d81)
					if d81.Loc != LocImm && d81.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d81.Loc == LocImm {
						if d81.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d81.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap82 := d0
						snap83 := d1
						snap84 := d2
						snap85 := d3
						snap86 := d9
						snap87 := d10
						snap88 := d11
						snap89 := d12
						snap90 := d13
						snap91 := d14
						snap92 := d16
						snap93 := d17
						snap94 := d18
						snap95 := d19
						snap96 := d20
						snap97 := d21
						snap98 := d22
						snap99 := d23
						snap100 := d24
						snap101 := d25
						snap102 := d47
						snap103 := d48
						snap104 := d49
						snap105 := d50
						snap106 := d76
						snap107 := d77
						snap108 := d79
						snap109 := d80
						snap110 := d81
						alloc111 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc111)
						d0 = snap82
						d1 = snap83
						d2 = snap84
						d3 = snap85
						d9 = snap86
						d10 = snap87
						d11 = snap88
						d12 = snap89
						d13 = snap90
						d14 = snap91
						d16 = snap92
						d17 = snap93
						d18 = snap94
						d19 = snap95
						d20 = snap96
						d21 = snap97
						d22 = snap98
						d23 = snap99
						d24 = snap100
						d25 = snap101
						d47 = snap102
						d48 = snap103
						d49 = snap104
						d50 = snap105
						d76 = snap106
						d77 = snap107
						d79 = snap108
						d80 = snap109
						d81 = snap110
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d80)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d112 = args[0]
					d112.ID = 0
					d114 = d112
					ctx.SyncDesc(&d114)
					if d114.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d114.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d114.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d114 = tmpScalar
					}
					d114 = JITPrepareScmerGoArg(ctx, d114)
					if d114.Loc != LocRegPair && d114.Loc != LocStackPair && d114.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d113 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d114}, 2)
					ctx.FreeDesc(&d112)
					ctx.EnsureDesc(&d113)
					if d113.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d113.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d113.Imm)
						ptrWord, _ := d113.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d113.Imm.String())))
						d113 = tmpPair
					} else if d113.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d113.Type, Reg: ctx.AllocRegExcept(d113.Reg), Reg2: ctx.AllocRegExcept(d113.Reg)}
						switch d113.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d113)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d113)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d113)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d113)
						d113 = tmpPair
					}
					if d113.Loc != LocRegPair && d113.Loc != LocStackPair && d113.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (StrLike arg0)")
					}
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
						panic("jit: generic call arg expects 2-word value (StrLike arg1)")
					}
					ctx.SyncDesc(&d113)
					ctx.SyncDesc(&d11)
					d115 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d113, d11}, 1)
					d115.NoHeapPointer = true
					ctx.EmitAndRegImm32(d115.Reg, 1)
					d115.Type = tagBool
					ctx.BindReg(d115.Reg, &d115)
					ctx.SyncDesc(&d115)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d115) {
						return d115
					}
					if d115.Loc == LocImm {
						ctx.EmitMakeBool(result, d115)
					} else {
						ctx.EmitMovToReg(result.Reg2, d115)
						d116 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d116)
						if d115.Loc == LocReg && d115.Reg != result.Reg2 {
							ctx.FreeReg(d115.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d79)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d79) {
						return d79
					}
					if d79.Loc == LocImm {
						ctx.EmitMakeBool(result, d79)
					} else {
						ctx.EmitMovToReg(result.Reg2, d79)
						d117 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d117)
						if d79.Loc == LocReg && d79.Reg != result.Reg2 {
							ctx.FreeReg(d79.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 40,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "toLower",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return compressedTextCase(a[0], false)
			}
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
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var phiBase2 int32
				_ = phiBase2
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
				var d12 JITValueDesc
				_ = d12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					ctx.StabilizeDescForControlFlow(&d1)
					phiBase2 = ctx.AllocStack(int32(16))
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					_ = d3
					lbl4 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d4 = d1
					d4.ID = 0
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 19)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d6)
					}
					ctx.ReclaimUntrackedRegs()
					d7 = d6
					ctx.SyncDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d3)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 26)
						r3 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d0)
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap13 := d0
						snap14 := d1
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d6
						snap19 := d7
						snap20 := d8
						snap21 := d9
						snap22 := d10
						snap23 := d11
						snap24 := d12
						alloc25 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc25)
						d0 = snap13
						d1 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d6 = snap18
						d7 = snap19
						d8 = snap20
						d9 = snap21
						d10 = snap22
						d11 = snap23
						d12 = snap24
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					d26 = JITPrepareScmerGoArg(ctx, d26)
					d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextCase), []JITValueDesc{d26, d27}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.FreeDesc(&d26)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
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
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d30.Imm)
						ptrWord, _ := d30.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d30.Imm.String())))
						d30 = tmpPair
					} else if d30.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d30)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d30)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d30)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d30)
						d30 = tmpPair
					}
					if d30.Loc != LocRegPair && d30.Loc != LocStackPair && d30.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d30)
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d30}, 2)
					d32.NoHeapPointer = false
					ctx.BindReg(d32.Reg, &d32)
					ctx.BindReg(d32.Reg2, &d32)
					ctx.EnsureDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d32}, 2)
					ctx.EmitMovPairToResult(&d33, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 23,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "toUpper",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return compressedTextCase(a[0], true)
			}
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
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var phiBase2 int32
				_ = phiBase2
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
				var d12 JITValueDesc
				_ = d12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					ctx.StabilizeDescForControlFlow(&d1)
					phiBase2 = ctx.AllocStack(int32(16))
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					_ = d3
					lbl4 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d4 = d1
					d4.ID = 0
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 19)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d6)
					}
					ctx.ReclaimUntrackedRegs()
					d7 = d6
					ctx.SyncDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d3)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 26)
						r3 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d0)
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap13 := d0
						snap14 := d1
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d6
						snap19 := d7
						snap20 := d8
						snap21 := d9
						snap22 := d10
						snap23 := d11
						snap24 := d12
						alloc25 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc25)
						d0 = snap13
						d1 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d6 = snap18
						d7 = snap19
						d8 = snap20
						d9 = snap21
						d10 = snap22
						d11 = snap23
						d12 = snap24
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					d26 = JITPrepareScmerGoArg(ctx, d26)
					d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextCase), []JITValueDesc{d26, d27}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.FreeDesc(&d26)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
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
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d30.Imm)
						ptrWord, _ := d30.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d30.Imm.String())))
						d30 = tmpPair
					} else if d30.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d30)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d30)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d30)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d30)
						d30 = tmpPair
					}
					if d30.Loc != LocRegPair && d30.Loc != LocStackPair && d30.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d30)
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d30}, 2)
					d32.NoHeapPointer = false
					ctx.BindReg(d32.Reg, &d32)
					ctx.BindReg(d32.Reg2, &d32)
					ctx.EnsureDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d32}, 2)
					ctx.EmitMovPairToResult(&d33, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 23,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "replace",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsBufferString() {
				return bufferReplace(a[0], String(a[1]), String(a[2]))
			}
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl4 := ctx.ReserveLabel()
					_ = lbl4
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d1 = d0
					d1.ID = 0
					d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x1a))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 26)
						d3 = ctx.DeferBooleanFlags(r0, CondEqual)
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.FreeDesc(&d0)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap5 := d0
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						alloc10 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc10)
						d0 = snap5
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d11 = args[0]
					d11.ID = 0
					d12 = args[1]
					d12.ID = 0
					d14 = d12
					ctx.SyncDesc(&d14)
					if d14.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d14.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d14.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d14 = tmpScalar
					}
					d14 = JITPrepareScmerGoArg(ctx, d14)
					if d14.Loc != LocRegPair && d14.Loc != LocStackPair && d14.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d14}, 2)
					ctx.FreeDesc(&d12)
					d15 = args[2]
					d15.ID = 0
					d17 = d15
					ctx.SyncDesc(&d17)
					if d17.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d17.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d17.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d17 = tmpScalar
					}
					d17 = JITPrepareScmerGoArg(ctx, d17)
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d17}, 2)
					ctx.FreeDesc(&d15)
					d11 = JITPrepareScmerGoArg(ctx, d11)
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
					if d13.Loc != LocRegPair && d13.Loc != LocStackPair && d13.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (bufferReplace arg1)")
					}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d16.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d16.Imm)
						ptrWord, _ := d16.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d16.Imm.String())))
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
					if d16.Loc != LocRegPair && d16.Loc != LocStackPair && d16.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (bufferReplace arg2)")
					}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d16)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(bufferReplace), []JITValueDesc{d11, d13, d16}, 2)
					d18.NoHeapPointer = false
					ctx.BindReg(d18.Reg, &d18)
					ctx.BindReg(d18.Reg2, &d18)
					ctx.FreeDesc(&d11)
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
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
					d25 = args[2]
					d25.ID = 0
					d27 = d25
					ctx.SyncDesc(&d27)
					if d27.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d27.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d27.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d27 = tmpScalar
					}
					d27 = JITPrepareScmerGoArg(ctx, d27)
					if d27.Loc != LocRegPair && d27.Loc != LocStackPair && d27.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d26 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d27}, 2)
					ctx.FreeDesc(&d25)
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
						panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg0)")
					}
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
						panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg1)")
					}
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d26.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d26.Imm)
						ptrWord, _ := d26.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d26.Imm.String())))
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
						panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg2)")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d26)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ReplaceAll), []JITValueDesc{d20, d23, d26}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.EnsureDesc(&d28)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d28}, 2)
					ctx.EmitMovPairToResult(&d29, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 29,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strtrim",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return compressedTextTrim(a[0], 0)
			}
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
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var phiBase2 int32
				_ = phiBase2
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
				var d12 JITValueDesc
				_ = d12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					ctx.StabilizeDescForControlFlow(&d1)
					phiBase2 = ctx.AllocStack(int32(16))
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					_ = d3
					lbl4 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d4 = d1
					d4.ID = 0
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 19)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d6)
					}
					ctx.ReclaimUntrackedRegs()
					d7 = d6
					ctx.SyncDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d3)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 26)
						r3 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d0)
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap13 := d0
						snap14 := d1
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d6
						snap19 := d7
						snap20 := d8
						snap21 := d9
						snap22 := d10
						snap23 := d11
						snap24 := d12
						alloc25 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc25)
						d0 = snap13
						d1 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d6 = snap18
						d7 = snap19
						d8 = snap20
						d9 = snap21
						d10 = snap22
						d11 = snap23
						d12 = snap24
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					d26 = JITPrepareScmerGoArg(ctx, d26)
					d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d26, d27}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.FreeDesc(&d26)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
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
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d30.Imm)
						ptrWord, _ := d30.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d30.Imm.String())))
						d30 = tmpPair
					} else if d30.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d30)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d30)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d30)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d30)
						d30 = tmpPair
					}
					if d30.Loc != LocRegPair && d30.Loc != LocStackPair && d30.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d30)
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d30}, 2)
					d32.NoHeapPointer = false
					ctx.BindReg(d32.Reg, &d32)
					ctx.BindReg(d32.Reg2, &d32)
					ctx.EnsureDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d32}, 2)
					ctx.EmitMovPairToResult(&d33, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 23,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strltrim",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return compressedTextTrim(a[0], -1)
			}
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
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var phiBase2 int32
				_ = phiBase2
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
				var d12 JITValueDesc
				_ = d12
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
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					ctx.StabilizeDescForControlFlow(&d1)
					phiBase2 = ctx.AllocStack(int32(16))
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					_ = d3
					lbl4 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d4 = d1
					d4.ID = 0
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 19)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d6)
					}
					ctx.ReclaimUntrackedRegs()
					d7 = d6
					ctx.SyncDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d3)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 26)
						r3 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d0)
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap13 := d0
						snap14 := d1
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d6
						snap19 := d7
						snap20 := d8
						snap21 := d9
						snap22 := d10
						snap23 := d11
						snap24 := d12
						alloc25 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc25)
						d0 = snap13
						d1 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d6 = snap18
						d7 = snap19
						d8 = snap20
						d9 = snap21
						d10 = snap22
						d11 = snap23
						d12 = snap24
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					d26 = JITPrepareScmerGoArg(ctx, d26)
					d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d26, d27}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.FreeDesc(&d26)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
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
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d30.Imm)
						ptrWord, _ := d30.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d30.Imm.String())))
						d30 = tmpPair
					} else if d30.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d30)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d30)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d30)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d30)
						d30 = tmpPair
					}
					if d30.Loc != LocRegPair && d30.Loc != LocStackPair && d30.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d32 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d32.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d32.Imm)
						ptrWord, _ := d32.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d32.Imm.String())))
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
					if d32.Loc != LocRegPair && d32.Loc != LocStackPair && d32.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d30)
					ctx.SyncDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d30, d32}, 2)
					d33.NoHeapPointer = false
					ctx.BindReg(d33.Reg, &d33)
					ctx.BindReg(d33.Reg2, &d33)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d33}, 2)
					ctx.EmitMovPairToResult(&d34, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 23,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "strrtrim",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return compressedTextTrim(a[0], 1)
			}
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
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var phiBase2 int32
				_ = phiBase2
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
				var d12 JITValueDesc
				_ = d12
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
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					ctx.StabilizeDescForControlFlow(&d1)
					phiBase2 = ctx.AllocStack(int32(16))
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					_ = d3
					lbl4 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d4 = d1
					d4.ID = 0
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 19)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d6)
					}
					ctx.ReclaimUntrackedRegs()
					d7 = d6
					ctx.SyncDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d3)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 26)
						r3 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d0)
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap13 := d0
						snap14 := d1
						snap15 := d3
						snap16 := d4
						snap17 := d5
						snap18 := d6
						snap19 := d7
						snap20 := d8
						snap21 := d9
						snap22 := d10
						snap23 := d11
						snap24 := d12
						alloc25 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc25)
						d0 = snap13
						d1 = snap14
						d3 = snap15
						d4 = snap16
						d5 = snap17
						d6 = snap18
						d7 = snap19
						d8 = snap20
						d9 = snap21
						d10 = snap22
						d11 = snap23
						d12 = snap24
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					d26 = JITPrepareScmerGoArg(ctx, d26)
					d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d26, d27}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.FreeDesc(&d26)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
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
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d30.Imm)
						ptrWord, _ := d30.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d30.Imm.String())))
						d30 = tmpPair
					} else if d30.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d30)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d30)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d30)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d30)
						d30 = tmpPair
					}
					if d30.Loc != LocRegPair && d30.Loc != LocStackPair && d30.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d32 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d32.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d32.Imm)
						ptrWord, _ := d32.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d32.Imm.String())))
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
					if d32.Loc != LocRegPair && d32.Loc != LocStackPair && d32.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d30)
					ctx.SyncDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d30, d32}, 2)
					d33.NoHeapPointer = false
					ctx.BindReg(d33.Reg, &d33)
					ctx.BindReg(d33.Reg2, &d33)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d33}, 2)
					ctx.EmitMovPairToResult(&d34, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 23,
		},
	})
	// SQL-level NULL-safe wrappers for TRIM/LTRIM/RTRIM
	Declare(&Globalenv, &Declaration{
		Name: "sql_trim",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			if isCompressedText(a[0]) {
				return compressedTextTrim(a[0], 0)
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
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var phiBase12 int32
				_ = phiBase12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					phiBase12 = ctx.AllocStack(int32(16))
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					_ = d13
					lbl6 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d14 = d11
					d14.ID = 0
					d15 = ctx.EmitGetTagDesc(&d14, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 19)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d16)
					}
					ctx.ReclaimUntrackedRegs()
					d17 = d16
					ctx.SyncDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 20)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d15.Reg, 26)
						r3 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d20)
					}
					ctx.EnsureDesc(&d20)
					ctx.EmitStoreToStack(d20, int32(phiBase12)+int32(0))
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d21 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d21)
					ctx.BindReg(r1, &d21)
					ctx.FreeDesc(&d10)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap23 := d0
						snap24 := d1
						snap25 := d2
						snap26 := d3
						snap27 := d9
						snap28 := d10
						snap29 := d11
						snap30 := d13
						snap31 := d14
						snap32 := d15
						snap33 := d16
						snap34 := d17
						snap35 := d18
						snap36 := d19
						snap37 := d20
						snap38 := d21
						snap39 := d22
						alloc40 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc40)
						d0 = snap23
						d1 = snap24
						d2 = snap25
						d3 = snap26
						d9 = snap27
						d10 = snap28
						d11 = snap29
						d13 = snap30
						d14 = snap31
						d15 = snap32
						d16 = snap33
						d17 = snap34
						d18 = snap35
						d19 = snap36
						d20 = snap37
						d21 = snap38
						d22 = snap39
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d21)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d41 = args[0]
					d41.ID = 0
					d41 = JITPrepareScmerGoArg(ctx, d41)
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d42.Loc == LocRegPair || d42.Loc == LocStackPair || d42.Loc == LocRegTriple || d42.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d42)
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d41, d42}, 2)
					d43.NoHeapPointer = false
					ctx.BindReg(d43.Reg, &d43)
					ctx.BindReg(d43.Reg2, &d43)
					ctx.FreeDesc(&d42)
					ctx.FreeDesc(&d41)
					ctx.SyncDesc(&d43)
					if d43.Loc == LocRegPair || d43.Loc == LocStackPair || d43.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d43, &result)
						result.Type = d43.Type
					} else {
						switch d43.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d43)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d43)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d43)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d43, &result)
							result.Type = d43.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d44 = args[0]
					d44.ID = 0
					d46 = d44
					ctx.SyncDesc(&d46)
					if d46.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d46.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d46.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d46 = tmpScalar
					}
					d46 = JITPrepareScmerGoArg(ctx, d46)
					if d46.Loc != LocRegPair && d46.Loc != LocStackPair && d46.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d46}, 2)
					ctx.FreeDesc(&d44)
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d45.Imm)
						ptrWord, _ := d45.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d45.Imm.String())))
						d45 = tmpPair
					} else if d45.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocRegExcept(d45.Reg), Reg2: ctx.AllocRegExcept(d45.Reg)}
						switch d45.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d45)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d45)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d45)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d45)
						d45 = tmpPair
					}
					if d45.Loc != LocRegPair && d45.Loc != LocStackPair && d45.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d45)
					d47 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d45}, 2)
					d47.NoHeapPointer = false
					ctx.BindReg(d47.Reg, &d47)
					ctx.BindReg(d47.Reg2, &d47)
					ctx.EnsureDesc(&d47)
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d47}, 2)
					ctx.EmitMovPairToResult(&d48, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 29,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_ltrim",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			if isCompressedText(a[0]) {
				return compressedTextTrim(a[0], -1)
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
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var phiBase12 int32
				_ = phiBase12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					phiBase12 = ctx.AllocStack(int32(16))
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					_ = d13
					lbl6 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d14 = d11
					d14.ID = 0
					d15 = ctx.EmitGetTagDesc(&d14, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 19)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d16)
					}
					ctx.ReclaimUntrackedRegs()
					d17 = d16
					ctx.SyncDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 20)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d15.Reg, 26)
						r3 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d20)
					}
					ctx.EnsureDesc(&d20)
					ctx.EmitStoreToStack(d20, int32(phiBase12)+int32(0))
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d21 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d21)
					ctx.BindReg(r1, &d21)
					ctx.FreeDesc(&d10)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap23 := d0
						snap24 := d1
						snap25 := d2
						snap26 := d3
						snap27 := d9
						snap28 := d10
						snap29 := d11
						snap30 := d13
						snap31 := d14
						snap32 := d15
						snap33 := d16
						snap34 := d17
						snap35 := d18
						snap36 := d19
						snap37 := d20
						snap38 := d21
						snap39 := d22
						alloc40 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc40)
						d0 = snap23
						d1 = snap24
						d2 = snap25
						d3 = snap26
						d9 = snap27
						d10 = snap28
						d11 = snap29
						d13 = snap30
						d14 = snap31
						d15 = snap32
						d16 = snap33
						d17 = snap34
						d18 = snap35
						d19 = snap36
						d20 = snap37
						d21 = snap38
						d22 = snap39
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d21)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d41 = args[0]
					d41.ID = 0
					d41 = JITPrepareScmerGoArg(ctx, d41)
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d42.Loc == LocRegPair || d42.Loc == LocStackPair || d42.Loc == LocRegTriple || d42.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d42)
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d41, d42}, 2)
					d43.NoHeapPointer = false
					ctx.BindReg(d43.Reg, &d43)
					ctx.BindReg(d43.Reg2, &d43)
					ctx.FreeDesc(&d42)
					ctx.FreeDesc(&d41)
					ctx.SyncDesc(&d43)
					if d43.Loc == LocRegPair || d43.Loc == LocStackPair || d43.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d43, &result)
						result.Type = d43.Type
					} else {
						switch d43.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d43)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d43)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d43)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d43, &result)
							result.Type = d43.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d44 = args[0]
					d44.ID = 0
					d46 = d44
					ctx.SyncDesc(&d46)
					if d46.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d46.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d46.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d46 = tmpScalar
					}
					d46 = JITPrepareScmerGoArg(ctx, d46)
					if d46.Loc != LocRegPair && d46.Loc != LocStackPair && d46.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d46}, 2)
					ctx.FreeDesc(&d44)
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d45.Imm)
						ptrWord, _ := d45.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d45.Imm.String())))
						d45 = tmpPair
					} else if d45.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocRegExcept(d45.Reg), Reg2: ctx.AllocRegExcept(d45.Reg)}
						switch d45.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d45)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d45)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d45)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d45)
						d45 = tmpPair
					}
					if d45.Loc != LocRegPair && d45.Loc != LocStackPair && d45.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d47 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d47)
					if d47.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d47.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d47.Imm)
						ptrWord, _ := d47.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d47.Imm.String())))
						d47 = tmpPair
					} else if d47.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d47.Type, Reg: ctx.AllocRegExcept(d47.Reg), Reg2: ctx.AllocRegExcept(d47.Reg)}
						switch d47.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d47)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d47)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d47)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d47)
						d47 = tmpPair
					}
					if d47.Loc != LocRegPair && d47.Loc != LocStackPair && d47.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d45)
					ctx.SyncDesc(&d47)
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d45, d47}, 2)
					d48.NoHeapPointer = false
					ctx.BindReg(d48.Reg, &d48)
					ctx.BindReg(d48.Reg2, &d48)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					d49 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d48}, 2)
					ctx.EmitMovPairToResult(&d49, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 29,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_rtrim",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			if isCompressedText(a[0]) {
				return compressedTextTrim(a[0], 1)
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
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var phiBase12 int32
				_ = phiBase12
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					phiBase12 = ctx.AllocStack(int32(16))
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					_ = d13
					lbl6 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d14 = d11
					d14.ID = 0
					d15 = ctx.EmitGetTagDesc(&d14, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 19)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d16)
					}
					ctx.ReclaimUntrackedRegs()
					d17 = d16
					ctx.SyncDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 20)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d15.Reg, 26)
						r3 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d20)
					}
					ctx.EnsureDesc(&d20)
					ctx.EmitStoreToStack(d20, int32(phiBase12)+int32(0))
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d21 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d21)
					ctx.BindReg(r1, &d21)
					ctx.FreeDesc(&d10)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap23 := d0
						snap24 := d1
						snap25 := d2
						snap26 := d3
						snap27 := d9
						snap28 := d10
						snap29 := d11
						snap30 := d13
						snap31 := d14
						snap32 := d15
						snap33 := d16
						snap34 := d17
						snap35 := d18
						snap36 := d19
						snap37 := d20
						snap38 := d21
						snap39 := d22
						alloc40 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc40)
						d0 = snap23
						d1 = snap24
						d2 = snap25
						d3 = snap26
						d9 = snap27
						d10 = snap28
						d11 = snap29
						d13 = snap30
						d14 = snap31
						d15 = snap32
						d16 = snap33
						d17 = snap34
						d18 = snap35
						d19 = snap36
						d20 = snap37
						d21 = snap38
						d22 = snap39
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d21)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d41 = args[0]
					d41.ID = 0
					d41 = JITPrepareScmerGoArg(ctx, d41)
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d42.Loc == LocRegPair || d42.Loc == LocStackPair || d42.Loc == LocRegTriple || d42.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d42)
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d41, d42}, 2)
					d43.NoHeapPointer = false
					ctx.BindReg(d43.Reg, &d43)
					ctx.BindReg(d43.Reg2, &d43)
					ctx.FreeDesc(&d42)
					ctx.FreeDesc(&d41)
					ctx.SyncDesc(&d43)
					if d43.Loc == LocRegPair || d43.Loc == LocStackPair || d43.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d43, &result)
						result.Type = d43.Type
					} else {
						switch d43.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d43)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d43)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d43)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d43, &result)
							result.Type = d43.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d44 = args[0]
					d44.ID = 0
					d46 = d44
					ctx.SyncDesc(&d46)
					if d46.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d46.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d46.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d46 = tmpScalar
					}
					d46 = JITPrepareScmerGoArg(ctx, d46)
					if d46.Loc != LocRegPair && d46.Loc != LocStackPair && d46.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d46}, 2)
					ctx.FreeDesc(&d44)
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d45.Imm)
						ptrWord, _ := d45.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d45.Imm.String())))
						d45 = tmpPair
					} else if d45.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocRegExcept(d45.Reg), Reg2: ctx.AllocRegExcept(d45.Reg)}
						switch d45.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d45)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d45)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d45)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d45)
						d45 = tmpPair
					}
					if d45.Loc != LocRegPair && d45.Loc != LocStackPair && d45.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d47 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d47)
					if d47.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d47.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d47.Imm)
						ptrWord, _ := d47.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d47.Imm.String())))
						d47 = tmpPair
					} else if d47.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d47.Type, Reg: ctx.AllocRegExcept(d47.Reg), Reg2: ctx.AllocRegExcept(d47.Reg)}
						switch d47.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d47)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d47)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d47)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d47)
						d47 = tmpPair
					}
					if d47.Loc != LocRegPair && d47.Loc != LocStackPair && d47.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d45)
					ctx.SyncDesc(&d47)
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d45, d47}, 2)
					d48.NoHeapPointer = false
					ctx.BindReg(d48.Reg, &d48)
					ctx.BindReg(d48.Reg2, &d48)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					d49 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d48}, 2)
					ctx.EmitMovPairToResult(&d49, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 29,
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
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[3].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() > 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d3.Reg, 1)
						d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d4)
					}
					ctx.FreeDesc(&d3)
					d5 = d4
					ctx.SyncDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							return bbs[1].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl7 := ctx.ReserveLabel()
					ctx.EmitJump(d5.Condition, lbl2)
					ctx.EmitJmp(lbl7)
					ctx.FreeDesc(&d4)
					snap6 := d1
					snap7 := d2
					snap8 := d3
					snap9 := d4
					snap10 := d5
					alloc11 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" ")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc11)
					d1 = snap6
					d2 = snap7
					d3 = snap8
					d4 = snap9
					d5 = snap10
					if !bbs[2].Rendered {
						snap12 := d1
						snap13 := d2
						snap14 := d3
						snap15 := d4
						snap16 := d5
						alloc17 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc17)
						d1 = snap12
						d2 = snap13
						d3 = snap14
						d4 = snap15
						d5 = snap16
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d18 = args[1]
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
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					ctx.SyncDesc(&d19)
					if d19.Loc == LocReg || d19.Loc == LocFPReg {
						ctx.ProtectReg(d19.Reg)
					} else if d19.Loc == LocRegPair {
						ctx.ProtectReg(d19.Reg)
						ctx.ProtectReg(d19.Reg2)
					}
					d21 = d19
					if d21.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d21)
					if d21.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d21, int32(bbs[2].PhiBase)+int32(0), 2)
					} else if d21.Loc == LocInputPair {
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
					} else if d21.Loc == LocRegPair || d21.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreToStack(d21, int32(bbs[2].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
					}
					if d19.Loc == LocReg || d19.Loc == LocFPReg {
						ctx.UnprotectReg(d19.Reg)
					} else if d19.Loc == LocRegPair {
						ctx.UnprotectReg(d19.Reg)
						ctx.UnprotectReg(d19.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d22 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.Split arg0)")
					}
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
					ctx.SyncDesc(&d23)
					ctx.SyncDesc(&d1)
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Split), []JITValueDesc{d23, d1}, 3)
					d25.NoHeapPointer = false
					ctx.BindReg(d25.Reg, &d25)
					ctx.BindReg(d25.Reg2, &d25)
					ctx.BindReg(d25.Reg3, &d25)
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.FreeDesc(&d1)
					if d25.SliceSizeKnown {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.KnownSliceLen))}
					} else if d25.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.StackOff))}
					} else if d25.Loc == LocStackTriple {
						d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d25.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d25)
						if d25.Loc == LocRegPair || d25.Loc == LocRegTriple {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2, ID: 0}
						} else if d25.Loc == LocReg {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					callResults27 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d26, d26}, []uint8{3}, []uint8{1})
					d28 = callResults27[0]
					d28.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d28)
					ctx.FreeDesc(&d26)
					if d25.SliceSizeKnown {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.KnownSliceLen))}
					} else if d25.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.StackOff))}
					} else if d25.Loc == LocStackTriple {
						d29 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d25.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d25)
						if d25.Loc == LocRegPair || d25.Loc == LocRegTriple {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2, ID: 0}
						} else if d25.Loc == LocReg {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d29)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					return bbs[3].Render()
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d30)
					}
					if d30.Loc == LocReg && d2.Loc == LocReg && d30.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDescsTogether(&d30, &d29)
					if d30.Loc == LocImm && d29.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d30.Imm.Int() < d29.Imm.Int())}
					} else if d29.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d30.Reg)
						if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d30.Reg, int32(d29.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d29.Imm.Int()))
							ctx.EmitCmpInt64(d30.Reg, ctx.ScratchReg)
						}
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d31)
					} else if d30.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d30.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d29.Reg)
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d31)
					} else {
						r3 := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitCmpInt64(d30.Reg, d29.Reg)
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d31)
					}
					d32 = d31
					ctx.SyncDesc(&d32)
					if d32.Loc != LocImm && d32.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d32.Loc == LocImm {
						if d32.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d32.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d31)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap33 := d1
						snap34 := d2
						snap35 := d3
						snap36 := d4
						snap37 := d5
						snap38 := d18
						snap39 := d19
						snap40 := d20
						snap41 := d21
						snap42 := d22
						snap43 := d23
						snap44 := d24
						snap45 := d25
						snap46 := d26
						snap47 := d28
						snap48 := d29
						snap49 := d30
						snap50 := d31
						snap51 := d32
						alloc52 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc52)
						d1 = snap33
						d2 = snap34
						d3 = snap35
						d4 = snap36
						d5 = snap37
						d18 = snap38
						d19 = snap39
						d20 = snap40
						d21 = snap41
						d22 = snap42
						d23 = snap43
						d24 = snap44
						d25 = snap45
						d26 = snap46
						d28 = snap47
						d29 = snap48
						d30 = snap49
						d31 = snap50
						d32 = snap51
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d28)
					ctx.EnsureDesc(&d30)
					d54 = ctx.EmitSliceElementAddress(&d25, &d30, 16)
					ctx.EnsureDesc(&d54)
					r4 := ctx.AllocRegExcept(d54.Reg)
					ctx.EmitMovRegMem(r4, d54.Reg, 8)
					ctx.EmitMovRegMem(d54.Reg, d54.Reg, 0)
					d53 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d54.Reg, Reg2: r4}
					ctx.BindReg(d54.Reg, &d53)
					ctx.BindReg(r4, &d53)
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d30)
					ctx.SyncDesc(&d53)
					d55 = d28
					d55.ID = 0
					d56 = d30
					d56.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d55, &d56, &d53, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d30)
						d56 = d30
						d56.ID = 0
						ctx.EmitStoreScmerSliceElement(&d55, &d56, &d53, int32(16))
					}
					ctx.FreeDesc(&d56)
					ctx.SyncDesc(&d30)
					if d30.Loc == LocReg || d30.Loc == LocFPReg {
						ctx.ProtectReg(d30.Reg)
					} else if d30.Loc == LocRegPair {
						ctx.ProtectReg(d30.Reg)
						ctx.ProtectReg(d30.Reg2)
					}
					d57 = d30
					if d57.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d57)
					ctx.EmitStoreToStack(d57, int32(bbs[3].PhiBase)+int32(0))
					if d30.Loc == LocReg || d30.Loc == LocFPReg {
						ctx.UnprotectReg(d30.Reg)
					} else if d30.Loc == LocRegPair {
						ctx.UnprotectReg(d30.Reg)
						ctx.UnprotectReg(d30.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d28)
					d58 = ctx.EmitNewSliceFromGoSlice(&d28)
					ctx.SyncDesc(&d58)
					if d58.Loc == LocRegPair || d58.Loc == LocStackPair || d58.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d58, &result)
						result.Type = d58.Type
					} else {
						switch d58.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d58)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d58)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d58)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d58, &result)
							result.Type = d58.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
				var d15 JITValueDesc
				_ = d15
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
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d11.Imm.Int())}
					} else if d11.Type == tagInt && d11.Loc == LocRegPair {
						ctx.FreeReg(d11.Reg)
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg2}
						ctx.BindReg(d11.Reg2, &d12)
						ctx.BindReg(d11.Reg2, &d12)
					} else if d11.Type == tagInt && d11.Loc == LocReg {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg}
						ctx.BindReg(d11.Reg, &d12)
						ctx.BindReg(d11.Reg, &d12)
					} else {
						d12 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d11}, 1)
						d12.Type = tagInt
						ctx.BindReg(d12.Reg, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() <= 0)}
					} else {
						r0 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpRegImm32(d12.Reg, 0)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r0, &d14)
					}
					d15 = d14
					ctx.SyncDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d15.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d14)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d2
						snap19 := d3
						snap20 := d9
						snap21 := d10
						snap22 := d11
						snap23 := d12
						snap24 := d13
						snap25 := d14
						snap26 := d15
						alloc27 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc27)
						d0 = snap16
						d1 = snap17
						d2 = snap18
						d3 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
						d12 = snap23
						d13 = snap24
						d14 = snap25
						d15 = snap26
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d28 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d28}, 2)
					ctx.EmitMovPairToResult(&d29, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d30 = args[0]
					d30.ID = 0
					d32 = d30
					ctx.SyncDesc(&d32)
					if d32.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d32.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d32.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d32 = tmpScalar
					}
					d32 = JITPrepareScmerGoArg(ctx, d32)
					if d32.Loc != LocRegPair && d32.Loc != LocStackPair && d32.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d32}, 2)
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d31.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d31.Imm)
						ptrWord, _ := d31.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d31.Imm.String())))
						d31 = tmpPair
					} else if d31.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d31.Type, Reg: ctx.AllocRegExcept(d31.Reg), Reg2: ctx.AllocRegExcept(d31.Reg)}
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d31)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d31)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d31)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d31)
						d31 = tmpPair
					}
					if d31.Loc != LocRegPair && d31.Loc != LocStackPair && d31.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.Repeat arg0)")
					}
					if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d31)
					ctx.SyncDesc(&d12)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(strings.Repeat), []JITValueDesc{d31, d12}, 2)
					d33.NoHeapPointer = false
					ctx.BindReg(d33.Reg, &d33)
					ctx.BindReg(d33.Reg2, &d33)
					ctx.EnsureDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d33}, 2)
					ctx.EmitMovPairToResult(&d34, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
			// their canonical callback to one function dispatch: binaryCollationLess owns
			// the required ASC NULL-first semantics, and swapping its operands owns
			// DESC NULL-last semantics. A closure per metadata key keeps callback
			// identity distinct even when two names have identical byte ordering.
			if collationName == "bin" || collationName == "binary" || collationName == "utf8" || collationName == "utf8mb4" {
				less := func(left, right Scmer) bool {
					if reverse {
						return binaryCollationLess(right, left)
					}
					return binaryCollationLess(left, right)
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
			var collationKey CollationKeyFunc
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
							f := func(a ...Scmer) Scmer { return NewBool(binaryCollationLess(a[1], a[0])) }
							collateRegistry.Store(FunctionIdentity(f), struct {
								Collation string
								Reverse   bool
							}{Collation: String(a[0]), Reverse: true})
							return NewFunc(f)
						}
						f := func(a ...Scmer) Scmer { return NewBool(binaryCollationLess(a[0], a[1])) }
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
								if isCompressedText(a[0]) || isCompressedText(a[1]) {
									if result, ok := generalCStringLess(a[0], a[1], reverse); ok {
										return NewBool(result)
									}
								}
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
							if isCompressedText(a[0]) || isCompressedText(a[1]) {
								if result, ok := generalCStringLess(a[0], a[1], reverse); ok {
									return NewBool(result)
								}
							}
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
					newCollator := func() *collate.Collator {
						if ci {
							return collate.New(tag, collate.Numeric, collate.IgnoreCase)
						}
						return collate.New(tag, collate.Numeric)
					}
					var c *collate.Collator
					// the following options are available:
					// IgnoreCase -> when string ends with _ci
					// IgnoreDiacritics -> o == ö
					// IgnoreWidth: half width == width
					// Numeric -> sort numbers correctly
					c = newCollator()
					type keyWorker struct {
						collator *collate.Collator
						buffer   collate.Buffer
					}
					var keyPool sync.Pool
					keyPool.New = func() any { return &keyWorker{collator: newCollator()} }
					collationKey = func(value Scmer) (string, bool) {
						if value.IsNil() {
							return "", true
						}
						// The relation stringifies every non-numeric pair before handing it
						// to x/text. Do the same here, including source-info wrapped values
						// emitted by query plans. Numeric pairs retain their native ordering
						// and therefore stay on the comparator fallback.
						if value.IsInt() || value.IsFloat() {
							return "", false
						}
						worker := keyPool.Get().(*keyWorker)
						worker.buffer.Reset()
						key := string(worker.collator.KeyFromString(&worker.buffer, String(value)))
						keyPool.Put(worker)
						return key, true
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
			/* Calling a variadic Scheme callback with two scalar arguments makes
			the argument array escape at every comparison. Index construction may
			perform millions of comparisons, so retain one frame per concurrent
			caller instead of feeding the garbage collector one tiny object each
			time. The callback itself is immutable; only the argument frame needs
			worker-local ownership. */
			var rawArgsPool sync.Pool
			rawArgsPool.New = func() any { return new([2]Scmer) }
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
				args := rawArgsPool.Get().(*[2]Scmer)
				args[0], args[1] = left, right
				result := ToBool(rawFn(args[:]...))
				args[0], args[1] = NewNil(), NewNil()
				rawArgsPool.Put(args)
				return result
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
			canonicalValue := canonical.(Scmer)
			if collationKey != nil {
				order := ":asc"
				if reverse {
					order = ":desc"
				}
				collateKeyRegistry.LoadOrStore(collationName+order,
					collationKeyDescriptor{Key: collationKey, Reverse: reverse})
			}
			return canonicalValue
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
				var d16 JITValueDesc
				_ = d16
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d7.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap8 := d0
						snap9 := d1
						snap10 := d2
						snap11 := d4
						snap12 := d5
						snap13 := d6
						snap14 := d7
						alloc15 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc15)
						d0 = snap8
						d1 = snap9
						d2 = snap10
						d4 = snap11
						d5 = snap12
						d6 = snap13
						d7 = snap14
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d6)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["urldecode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d4}, 2)
					ctx.EmitMovPairToResult(&d16, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
				var d14 JITValueDesc
				_ = d14
				var d16 JITValueDesc
				_ = d16
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = ctx.EmitGoCallScalar(GoFuncAddr(func(value Scmer) any { return value }), []JITValueDesc{d0}, 2)
					ctx.FreeDesc(&d0)
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap7 := d0
						snap8 := d1
						snap9 := d3
						snap10 := d4
						snap11 := d5
						snap12 := d6
						alloc13 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc13)
						d0 = snap7
						d1 = snap8
						d3 = snap9
						d4 = snap10
						d5 = snap11
						d6 = snap12
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d5)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_encode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					callResults15 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d3}, []uint8{2}, []uint8{1})
					d14 = callResults15[0]
					ctx.EnsureDesc(&d14)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d14}, 2)
					ctx.EmitMovPairToResult(&d16, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = ctx.EmitGoCallScalar(GoFuncAddr(func() *bytes.Buffer { return new(bytes.Buffer) }), nil, 1)
					ctx.BindReg(d10.Reg, &d10)
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.EnsureDesc(&d10)
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *bytes.Buffer) io.Writer { return value }), []JITValueDesc{d10}, 2)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d11.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d11)
						} else if d11.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d11)
						} else if d11.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d11)
						} else if d11.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d11.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
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
						panic("jit: generic call arg expects 2-word value (json.NewEncoder arg0)")
					}
					ctx.SyncDesc(&d11)
					d12 = ctx.EmitGoCallScalar(GoFuncAddr(json.NewEncoder), []JITValueDesc{d11}, 1)
					d12.NoHeapPointer = false
					ctx.BindReg(d12.Reg, &d12)
					if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d12)
					ctx.SyncDesc(&d13)
					ctx.EmitGoCallVoid(GoFuncAddr((*json.Encoder).SetEscapeHTML), []JITValueDesc{d12, d13})
					ctx.FreeDesc(&d13)
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
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d15}, 2)
					if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
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
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*json.Encoder).Encode arg1)")
					}
					ctx.SyncDesc(&d12)
					ctx.SyncDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr((*json.Encoder).Encode), []JITValueDesc{d12, d17}, 2)
					d18.NoHeapPointer = false
					ctx.BindReg(d18.Reg, &d18)
					ctx.BindReg(d18.Reg2, &d18)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d18.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d18)
						if d18.Loc != LocReg && d18.Loc != LocRegPair && d18.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpRegImm32(d18.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d19)
					}
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap21 := d0
						snap22 := d1
						snap23 := d2
						snap24 := d3
						snap25 := d9
						snap26 := d10
						snap27 := d11
						snap28 := d12
						snap29 := d13
						snap30 := d14
						snap31 := d15
						snap32 := d16
						snap33 := d17
						snap34 := d18
						snap35 := d19
						snap36 := d20
						alloc37 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc37)
						d0 = snap21
						d1 = snap22
						d2 = snap23
						d3 = snap24
						d9 = snap25
						d10 = snap26
						d11 = snap27
						d12 = snap28
						d13 = snap29
						d14 = snap30
						d15 = snap31
						d16 = snap32
						d17 = snap33
						d18 = snap34
						d19 = snap35
						d20 = snap36
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d19)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d38 = args[0]
					d38.ID = 0
					d40 = d38
					d40.ID = 0
					d39 = ctx.EmitIsStringBorrowed(&d40, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d38)
					d41 = d39
					ctx.EnsureDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[1].Render()
					}
					ctx.EmitCmpRegImm32(d41.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl3)
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
					}
					ctx.FlushRegisterMoves()
					if !bbs[1].Rendered {
						snap42 := d0
						snap43 := d1
						snap44 := d2
						snap45 := d3
						snap46 := d9
						snap47 := d10
						snap48 := d11
						snap49 := d12
						snap50 := d13
						snap51 := d14
						snap52 := d15
						snap53 := d16
						snap54 := d17
						snap55 := d18
						snap56 := d19
						snap57 := d20
						snap58 := d38
						snap59 := d39
						snap60 := d40
						snap61 := d41
						alloc62 := ctx.SnapshotAllocState()
						bbs[1].Render()
						ctx.RestoreAllocState(alloc62)
						d0 = snap42
						d1 = snap43
						d2 = snap44
						d3 = snap45
						d9 = snap46
						d10 = snap47
						d11 = snap48
						d12 = snap49
						d13 = snap50
						d14 = snap51
						d15 = snap52
						d16 = snap53
						d17 = snap54
						d18 = snap55
						d19 = snap56
						d20 = snap57
						d38 = snap58
						d39 = snap59
						d40 = snap60
						d41 = snap61
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
					ctx.FreeDesc(&d39)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_quote"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d10)
					if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d10)
					d63 = ctx.EmitGoCallScalar(GoFuncAddr((*bytes.Buffer).String), []JITValueDesc{d10}, 2)
					d63.NoHeapPointer = false
					ctx.BindReg(d63.Reg, &d63)
					ctx.BindReg(d63.Reg2, &d63)
					ctx.EnsureDesc(&d63)
					if d63.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d63.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d63.Imm)
						ptrWord, _ := d63.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d63.Imm.String())))
						d63 = tmpPair
					} else if d63.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d63.Type, Reg: ctx.AllocRegExcept(d63.Reg), Reg2: ctx.AllocRegExcept(d63.Reg)}
						switch d63.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d63)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d63)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d63)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d63)
						d63 = tmpPair
					}
					if d63.Loc != LocRegPair && d63.Loc != LocStackPair && d63.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSuffix arg0)")
					}
					d64 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("\n")}
					ctx.EnsureDesc(&d64)
					if d64.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d64.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d64.Imm)
						ptrWord, _ := d64.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d64.Imm.String())))
						d64 = tmpPair
					} else if d64.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d64.Type, Reg: ctx.AllocRegExcept(d64.Reg), Reg2: ctx.AllocRegExcept(d64.Reg)}
						switch d64.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d64)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d64)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d64)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d64)
						d64 = tmpPair
					}
					if d64.Loc != LocRegPair && d64.Loc != LocStackPair && d64.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSuffix arg1)")
					}
					ctx.SyncDesc(&d63)
					ctx.SyncDesc(&d64)
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSuffix), []JITValueDesc{d63, d64}, 2)
					d65.NoHeapPointer = false
					ctx.BindReg(d65.Reg, &d65)
					ctx.BindReg(d65.Reg2, &d65)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d65)
					d66 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d65}, 2)
					ctx.EmitMovPairToResult(&d66, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  27,
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
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
					d4 = JITPrepareGoSliceArg(ctx, d4)
					if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (json.Unmarshal arg0)")
					}
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap10 := d0
						snap11 := d1
						snap12 := d2
						snap13 := d3
						snap14 := d4
						snap15 := d6
						snap16 := d7
						snap17 := d8
						snap18 := d9
						alloc19 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc19)
						d0 = snap10
						d1 = snap11
						d2 = snap12
						d3 = snap13
						d4 = snap14
						d6 = snap15
						d7 = snap16
						d8 = snap17
						d9 = snap18
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d8)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d0)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *any) any { return *value }), []JITValueDesc{d0}, 2)
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
						panic("jit: generic call arg expects 2-word value (TransformFromJSON arg0)")
					}
					ctx.SyncDesc(&d20)
					d21 = ctx.EmitGoCallScalar(GoFuncAddr(TransformFromJSON), []JITValueDesc{d20}, 2)
					d21.NoHeapPointer = false
					ctx.BindReg(d21.Reg, &d21)
					ctx.BindReg(d21.Reg2, &d21)
					ctx.FreeDesc(&d20)
					ctx.SyncDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d21, &result)
						result.Type = d21.Type
					} else {
						switch d21.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d21)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d21)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d21)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d21, &result)
							result.Type = d21.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
				var d20 JITValueDesc
				_ = d20
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
					d4 = JITPrepareGoSliceArg(ctx, d4)
					if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (json.Unmarshal arg0)")
					}
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap10 := d0
						snap11 := d1
						snap12 := d2
						snap13 := d3
						snap14 := d4
						snap15 := d6
						snap16 := d7
						snap17 := d8
						snap18 := d9
						alloc19 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc19)
						d0 = snap10
						d1 = snap11
						d2 = snap12
						d3 = snap13
						d4 = snap14
						d6 = snap15
						d7 = snap16
						d8 = snap17
						d9 = snap18
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d8)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["json_decode_scmer"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d20 = d0
					_ = d20
					ctx.SyncDesc(&d20)
					if d20.Loc == LocRegPair || d20.Loc == LocStackPair || d20.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d20, &result)
						result.Type = d20.Type
					} else {
						switch d20.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d20)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d20)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d20)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d20, &result)
							result.Type = d20.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  13,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "base64_encode",

		Fn: func(a ...Scmer) Scmer {
			return encodeTextBase64(a[0])
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
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
				d1 := d0
				_ = d1
				ctx.StabilizeDescForControlFlow(&d1)
				lbl0 := ctx.ReserveLabel()
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_1_1 := int32(-1)
				_ = bbpos_1_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_1_2 := int32(-1)
				_ = bbpos_1_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d1)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d2 := d1
				d2.ID = 0
				d3 := ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d1)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				var d4 JITValueDesc
				if d3.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x1a))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d3.Reg, 26)
					d4 = ctx.DeferBooleanFlags(r0, CondEqual)
					ctx.BindReg(r0, &d4)
				}
				ctx.FreeDesc(&d3)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d5 := d4
				ctx.SyncDesc(&d5)
				if d5.Loc != LocImm && d5.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl5 := ctx.ReserveLabel()
				lbl6 := ctx.ReserveLabel()
				if d5.Loc == LocImm {
					if d5.Imm.Bool() {
						ctx.MarkLabel(lbl5)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl6)
						ctx.EmitJmp(lbl3)
					}
				} else {
					ctx.EmitJump(d5.Condition, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.FreeDesc(&d4)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl3)
				}
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d7 := d1
				ctx.SyncDesc(&d7)
				if d7.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d7.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d7.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d7 = tmpScalar
				}
				d7 = JITPrepareScmerGoArg(ctx, d7)
				if d7.Loc != LocRegPair && d7.Loc != LocStackPair && d7.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d6 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d7}, 2)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d6)
				var d8 JITValueDesc
				if d6.Loc == LocImm {
					ptr, _ := d6.Imm.RawWords()
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(ptr))))}
				} else if d6.Loc == LocStackPair {
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff}
				} else {
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocRegPair {
						panic("StringData requires a Go string pair")
					}
					d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg, ID: 0}
				}
				ctx.ReclaimUntrackedRegs()
				var d9 JITValueDesc
				if d6.SliceSizeKnown {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.KnownSliceLen))}
				} else if d6.Loc == LocImm {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d6.Imm.String())))}
				} else if d6.Loc == LocStackTriple {
					d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff + 8, NoHeapPointer: true}
				} else if d6.Loc == LocStackPair {
					d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocRegPair || d6.Loc == LocRegTriple {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2, ID: 0}
					} else if d6.Loc == LocReg {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocRegTriple || d9.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d10 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d11 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d8)
				ctx.SyncDesc(&d9)
				ctx.SyncDesc(&d10)
				ctx.SyncDesc(&d11)
				d12 := ctx.EmitGoCallScalar(GoFuncAddr(NewBString), []JITValueDesc{d8, d9, d10, d11}, 2)
				d12.NoHeapPointer = false
				ctx.BindReg(d12.Reg, &d12)
				ctx.BindReg(d12.Reg2, &d12)
				ctx.FreeDesc(&d10)
				ctx.FreeDesc(&d11)
				ctx.FreeDesc(&d8)
				ctx.FreeDesc(&d9)
				ctx.ReclaimUntrackedRegs()
				r1 := ctx.AllocReg()
				r2 := ctx.AllocRegExcept(r1)
				d13 := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d13)
				ctx.BindReg(r2, &d13)
				ctx.EmitMovPairToResult(&d12, &d13)
				ctx.EmitJmp(lbl0)
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d1)
				d14 := ctx.EmitGoCallScalar(GoFuncAddr(encodeBufferBase64), []JITValueDesc{d1}, 2)
				d14.NoHeapPointer = false
				ctx.BindReg(d14.Reg, &d14)
				ctx.BindReg(d14.Reg2, &d14)
				ctx.ReclaimUntrackedRegs()
				d15 := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d15)
				ctx.BindReg(r2, &d15)
				ctx.EmitMovPairToResult(&d14, &d15)
				ctx.EmitJmp(lbl0)
				ctx.MarkLabel(lbl0)
				d16 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d16)
				ctx.BindReg(r2, &d16)
				ctx.BindReg(r1, &d16)
				ctx.BindReg(r2, &d16)
				ctx.FreeDesc(&d0)
				if d16.Loc == LocImm {
					if result.Loc == LocAny {
						return d16
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
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
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  16,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "base64_decode",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsBufferString() {
				return decodeBufferBase64(a[0])
			}
			if a[0].IsBString() && auxVal(a[0].aux)>>46 == 0 {
				return bstringRawText(a[0])
			}
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
				var d4 JITValueDesc
				_ = d4
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
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
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
				var d94 JITValueDesc
				_ = d94
				var d96 JITValueDesc
				_ = d96
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d1 = d0
					d1.ID = 0
					d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x1a))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 26)
						d3 = ctx.DeferBooleanFlags(r0, CondEqual)
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.FreeDesc(&d0)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap5 := d0
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						alloc10 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc10)
						d0 = snap5
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d11 = args[0]
					d11.ID = 0
					d11 = JITPrepareScmerGoArg(ctx, d11)
					ctx.SyncDesc(&d11)
					d12 = ctx.EmitGoCallScalar(GoFuncAddr(decodeBufferBase64), []JITValueDesc{d11}, 2)
					d12.NoHeapPointer = false
					ctx.BindReg(d12.Reg, &d12)
					ctx.BindReg(d12.Reg2, &d12)
					ctx.FreeDesc(&d11)
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
							ctx.EmitMovPairToResult(&d12, &result)
							result.Type = d12.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					ctx.SyncDesc(&d13)
					d14 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsBString), []JITValueDesc{d13}, 1)
					d14.NoHeapPointer = true
					ctx.EmitAndRegImm32(d14.Reg, 1)
					d14.Type = tagBool
					ctx.BindReg(d14.Reg, &d14)
					ctx.FreeDesc(&d13)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d2
						snap19 := d3
						snap20 := d4
						snap21 := d11
						snap22 := d12
						snap23 := d13
						snap24 := d14
						snap25 := d15
						alloc26 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc26)
						d0 = snap16
						d1 = snap17
						d2 = snap18
						d3 = snap19
						d4 = snap20
						d11 = snap21
						d12 = snap22
						d13 = snap23
						d14 = snap24
						d15 = snap25
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d14)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d27 = args[0]
					d27.ID = 0
					d27 = JITPrepareScmerGoArg(ctx, d27)
					ctx.SyncDesc(&d27)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(bstringRawText), []JITValueDesc{d27}, 2)
					d28.NoHeapPointer = false
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d27)
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(func() *base64.Encoding { return base64.StdEncoding }), nil, 1)
					d30 = args[0]
					d30.ID = 0
					d32 = d30
					ctx.SyncDesc(&d32)
					if d32.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d32.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d32.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d32 = tmpScalar
					}
					d32 = JITPrepareScmerGoArg(ctx, d32)
					if d32.Loc != LocRegPair && d32.Loc != LocStackPair && d32.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d32}, 2)
					ctx.FreeDesc(&d30)
					if d29.Loc == LocRegPair || d29.Loc == LocStackPair || d29.Loc == LocRegTriple || d29.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d31.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d31.Imm)
						ptrWord, _ := d31.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d31.Imm.String())))
						d31 = tmpPair
					} else if d31.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d31.Type, Reg: ctx.AllocRegExcept(d31.Reg), Reg2: ctx.AllocRegExcept(d31.Reg)}
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d31)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d31)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d31)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d31)
						d31 = tmpPair
					}
					if d31.Loc != LocRegPair && d31.Loc != LocStackPair && d31.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*base64.Encoding).DecodeString arg1)")
					}
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d31)
					callResults33 := JITEmitGoCallResults(ctx, GoFuncAddr((*base64.Encoding).DecodeString), []JITValueDesc{d29, d31}, []uint8{3, 2}, []uint8{1, 3})
					d34 = callResults33[0]
					_ = d34
					d35 = callResults33[1]
					_ = d35
					ctx.FreeDesc(&d29)
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d35.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d35)
						if d35.Loc != LocReg && d35.Loc != LocRegPair && d35.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d35.Reg)
						ctx.EmitCmpRegImm32(d35.Reg, 0)
						ctx.EmitSetcc(r1, CondNotEqual)
						d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d36)
					}
					d37 = d36
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocImm && d37.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d37.Loc == LocImm {
						if d37.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d37.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap38 := d0
						snap39 := d1
						snap40 := d2
						snap41 := d3
						snap42 := d4
						snap43 := d11
						snap44 := d12
						snap45 := d13
						snap46 := d14
						snap47 := d15
						snap48 := d27
						snap49 := d28
						snap50 := d29
						snap51 := d30
						snap52 := d31
						snap53 := d32
						snap54 := d34
						snap55 := d35
						snap56 := d36
						snap57 := d37
						alloc58 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc58)
						d0 = snap38
						d1 = snap39
						d2 = snap40
						d3 = snap41
						d4 = snap42
						d11 = snap43
						d12 = snap44
						d13 = snap45
						d14 = snap46
						d15 = snap47
						d27 = snap48
						d28 = snap49
						d29 = snap50
						d30 = snap51
						d31 = snap52
						d32 = snap53
						d34 = snap54
						d35 = snap55
						d36 = snap56
						d37 = snap57
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d36)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d59 = args[0]
					d59.ID = 0
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocImm {
						_, auxWord := d59.Imm.RawWords()
						d60 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d59.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r2 := ctx.AllocReg()
						ctx.EmitMovRegReg(r2, d59.Reg2)
						d60 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d60)
					}
					ctx.EnsureDesc(&d60)
					d61 = d60
					_ = d61
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d61)
					if d61.Loc == LocImm {
						d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d61.Imm.Int()) >> 8))}
					} else {
						r3 := ctx.AllocRegExcept(d61.Reg)
						ctx.EmitMovRegReg(r3, d61.Reg)
						ctx.EmitShrRegImm8(r3, 8)
						d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d62)
					}
					if d62.Loc == LocReg && d61.Loc == LocReg && d62.Reg == d61.Reg {
						ctx.TransferReg(d61.Reg)
						d61.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d62)
					ctx.FreeDesc(&d60)
					ctx.EnsureDesc(&d62)
					if d62.Loc == LocImm {
						d63 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d62.Imm.Int()) >> 46))}
					} else {
						ctx.EmitShrRegImm8(d62.Reg, 46)
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d62.Reg}
						ctx.BindReg(d62.Reg, &d63)
					}
					if d63.Loc == LocReg && d62.Loc == LocReg && d63.Reg == d62.Reg {
						ctx.TransferReg(d62.Reg)
						d62.Loc = LocNone
					}
					ctx.FreeDesc(&d62)
					ctx.EnsureDesc(&d63)
					if d63.Loc == LocImm {
						d64 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d63.Imm.Int()) == uint64(0x0))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d63.Reg, 0)
						d64 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
						ctx.BindReg(r4, &d64)
					}
					ctx.FreeDesc(&d63)
					d65 = d64
					ctx.SyncDesc(&d65)
					if d65.Loc != LocImm && d65.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d65.Loc == LocImm {
						if d65.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d65.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d64)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap66 := d0
						snap67 := d1
						snap68 := d2
						snap69 := d3
						snap70 := d4
						snap71 := d11
						snap72 := d12
						snap73 := d13
						snap74 := d14
						snap75 := d15
						snap76 := d27
						snap77 := d28
						snap78 := d29
						snap79 := d30
						snap80 := d31
						snap81 := d32
						snap82 := d34
						snap83 := d35
						snap84 := d36
						snap85 := d37
						snap86 := d59
						snap87 := d60
						snap88 := d61
						snap89 := d62
						snap90 := d63
						snap91 := d64
						snap92 := d65
						alloc93 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc93)
						d0 = snap66
						d1 = snap67
						d2 = snap68
						d3 = snap69
						d4 = snap70
						d11 = snap71
						d12 = snap72
						d13 = snap73
						d14 = snap74
						d15 = snap75
						d27 = snap76
						d28 = snap77
						d29 = snap78
						d30 = snap79
						d31 = snap80
						d32 = snap81
						d34 = snap82
						d35 = snap83
						d36 = snap84
						d37 = snap85
						d59 = snap86
						d60 = snap87
						d61 = snap88
						d62 = snap89
						d63 = snap90
						d64 = snap91
						d65 = snap92
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_decode"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					callResults95 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d34}, []uint8{2}, []uint8{1})
					d94 = callResults95[0]
					ctx.EnsureDesc(&d94)
					d96 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d94}, 2)
					ctx.EmitMovPairToResult(&d96, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  49,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "bin2hex",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return encodeTextHex(a[0])
			}
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
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var phiBase4 int32
				_ = phiBase4
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
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
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
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
				var d94 JITValueDesc
				_ = d94
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					ctx.EnsureDesc(&d2)
					d3 = d2
					_ = d3
					ctx.StabilizeDescForControlFlow(&d3)
					phiBase4 = ctx.AllocStack(int32(16))
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					_ = d5
					lbl7 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d6 = d3
					d6.ID = 0
					d7 = ctx.EmitGetTagDesc(&d6, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpRegImm32(d7.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d5)
					}
					ctx.EmitJmp(lbl7)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpRegImm32(d7.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl9)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d7.Reg, 26)
						r3 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase4)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d13 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d13)
					ctx.BindReg(r1, &d13)
					ctx.FreeDesc(&d2)
					d14 = d13
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d14.Loc == LocImm {
						if d14.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d5
						snap19 := d6
						snap20 := d7
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d13
						snap27 := d14
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d5 = snap18
						d6 = snap19
						d7 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d13 = snap26
						d14 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d13)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					ctx.SyncDesc(&d29)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(encodeTextHex), []JITValueDesc{d29}, 2)
					d30.NoHeapPointer = false
					ctx.BindReg(d30.Reg, &d30)
					ctx.BindReg(d30.Reg2, &d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d30, &result)
						result.Type = d30.Type
					} else {
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d30)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d30)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d30)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d30, &result)
							result.Type = d30.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d31 = args[0]
					d31.ID = 0
					d33 = d31
					ctx.SyncDesc(&d33)
					if d33.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d33.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d33.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d33 = tmpScalar
					}
					d33 = JITPrepareScmerGoArg(ctx, d33)
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d33}, 2)
					ctx.StabilizeDescForControlFlow(&d32)
					ctx.FreeDesc(&d31)
					if d32.SliceSizeKnown {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d32.KnownSliceLen))}
					} else if d32.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d32.Imm.String())))}
					} else if d32.Loc == LocStackTriple {
						d34 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else if d32.Loc == LocStackPair {
						d34 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d32)
						if d32.Loc == LocRegPair || d32.Loc == LocRegTriple {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2, ID: 0}
						} else if d32.Loc == LocReg {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d34)
					ctx.SyncDesc(&d35)
					ctx.SyncDesc(&d34)
					if d35.Loc == LocImm && d34.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d35.Imm.Int() * d34.Imm.Int())}
					} else if d35.Loc == LocImm {
						ctx.EnsureDesc(&d34)
						scratch := ctx.AllocRegExcept(d34.Reg)
						ctx.EmitMovRegReg(scratch, d34.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d35.Imm.Int())
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d36)
					} else if d34.Loc == LocImm {
						ctx.EnsureDesc(&d35)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d35.Reg, d34.Imm.Int())
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg}
						ctx.BindReg(d35.Reg, &d36)
					} else {
						ctx.EnsureDesc(&d35)
						ctx.SyncDesc(&d34)
						ctx.EmitIntBinary(JITIntMul, 64, d35.Reg, &d34)
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg}
						ctx.BindReg(d35.Reg, &d36)
					}
					if d36.Loc == LocReg && d35.Loc == LocReg && d36.Reg == d35.Reg {
						ctx.TransferReg(d35.Reg)
						d35.Loc = LocNone
					}
					ctx.FreeDesc(&d34)
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					callResults37 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d36, d36}, []uint8{3}, []uint8{1})
					d38 = callResults37[0]
					d38.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.FreeDesc(&d36)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(0))
					return bbs[3].Render()
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					if d32.SliceSizeKnown {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d32.KnownSliceLen))}
					} else if d32.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d32.Imm.String())))}
					} else if d32.Loc == LocStackTriple {
						d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else if d32.Loc == LocStackPair {
						d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d32)
						if d32.Loc == LocRegPair || d32.Loc == LocRegTriple {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2, ID: 0}
						} else if d32.Loc == LocReg {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDescsTogether(&d1, &d39)
					if d1.Loc == LocImm && d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d39.Imm.Int())}
					} else if d39.Loc == LocImm {
						r4 := ctx.AllocRegExcept(d1.Reg)
						if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d39.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d39.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d40)
					} else if d1.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d39.Reg)
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d40)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d39.Reg)
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedLess}
						ctx.BindReg(r6, &d40)
					}
					ctx.FreeDesc(&d39)
					d41 = d40
					ctx.SyncDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d41.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d40)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap42 := d1
						snap43 := d2
						snap44 := d3
						snap45 := d5
						snap46 := d6
						snap47 := d7
						snap48 := d8
						snap49 := d9
						snap50 := d10
						snap51 := d11
						snap52 := d12
						snap53 := d13
						snap54 := d14
						snap55 := d29
						snap56 := d30
						snap57 := d31
						snap58 := d32
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d38
						snap64 := d39
						snap65 := d40
						snap66 := d41
						alloc67 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc67)
						d1 = snap42
						d2 = snap43
						d3 = snap44
						d5 = snap45
						d6 = snap46
						d7 = snap47
						d8 = snap48
						d9 = snap49
						d10 = snap50
						d11 = snap51
						d12 = snap52
						d13 = snap53
						d14 = snap54
						d29 = snap55
						d30 = snap56
						d31 = snap57
						d32 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d38 = snap63
						d39 = snap64
						d40 = snap65
						d41 = snap66
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d38)
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d68)
					ctx.SyncDesc(&d1)
					if d68.Loc == LocImm && d1.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d68.Imm.Int() * d1.Imm.Int())}
					} else if d68.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d68.Imm.Int())
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d69)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d68)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d68.Reg, d1.Imm.Int())
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d68.Reg}
						ctx.BindReg(d68.Reg, &d69)
					} else {
						ctx.EnsureDesc(&d68)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d68.Reg, &d1)
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d68.Reg}
						ctx.BindReg(d68.Reg, &d69)
					}
					if d69.Loc == LocReg && d68.Loc == LocReg && d69.Reg == d68.Reg {
						ctx.TransferReg(d68.Reg)
						d68.Loc = LocNone
					}
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d32)
					d70 = ctx.EmitSliceElementAddress(&d32, &d1, 1)
					ctx.EnsureDesc(&d70)
					r7 := ctx.AllocRegExcept(d70.Reg)
					ctx.EmitMovRegMemB(r7, d70.Reg, 0)
					ctx.FreeDesc(&d70)
					d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d71)
					ctx.BindReg(r7, &d71)
					ctx.EnsureDesc(&d71)
					if d71.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d71.Reg, 4)
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d72)
					}
					if d72.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: d72.Type, Imm: NewInt(int64(uint64(d72.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d72.Reg, 56)
						ctx.EmitShrRegImm8(d72.Reg, 56)
					}
					if d72.Loc == LocReg && d71.Loc == LocReg && d72.Reg == d71.Reg {
						ctx.TransferReg(d71.Reg)
						d71.Loc = LocNone
					}
					ctx.FreeDesc(&d71)
					d73 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d72)
					ctx.EnsureGoStringHeader(&d73)
					d74 = ctx.EmitSliceElementAddress(&d73, &d72, 1)
					ctx.EnsureDesc(&d74)
					r8 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegMemB(r8, d74.Reg, 0)
					ctx.FreeDesc(&d74)
					d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8, NoHeapPointer: true}
					ctx.BindReg(r8, &d75)
					ctx.BindReg(r8, &d75)
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d69)
					ctx.SyncDesc(&d75)
					d76 = d38
					d76.ID = 0
					d77 = d69
					d77.ID = 0
					d78 = ctx.EmitSliceElementAddress(&d76, &d77, int32(1))
					ctx.EmitStoreScalarAt(&d78, &d75, 1)
					ctx.FreeDesc(&d78)
					ctx.FreeDesc(&d77)
					ctx.FreeDesc(&d69)
					ctx.FreeDesc(&d75)
					d79 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d79)
					ctx.SyncDesc(&d1)
					if d79.Loc == LocImm && d1.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d79.Imm.Int() * d1.Imm.Int())}
					} else if d79.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d79.Imm.Int())
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d80)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d79)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d79.Reg, d1.Imm.Int())
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg}
						ctx.BindReg(d79.Reg, &d80)
					} else {
						ctx.EnsureDesc(&d79)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d79.Reg, &d1)
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg}
						ctx.BindReg(d79.Reg, &d80)
					}
					if d80.Loc == LocReg && d79.Loc == LocReg && d80.Reg == d79.Reg {
						ctx.TransferReg(d79.Reg)
						d79.Loc = LocNone
					}
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d80.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d80.Reg)
						ctx.EmitMovRegReg(scratch, d80.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d81)
					}
					if d81.Loc == LocReg && d80.Loc == LocReg && d81.Reg == d80.Reg {
						ctx.TransferReg(d80.Reg)
						d80.Loc = LocNone
					}
					ctx.FreeDesc(&d80)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d32)
					d82 = ctx.EmitSliceElementAddress(&d32, &d1, 1)
					ctx.EnsureDesc(&d82)
					r9 := ctx.AllocRegExcept(d82.Reg)
					ctx.EmitMovRegMemB(r9, d82.Reg, 0)
					ctx.FreeDesc(&d82)
					d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9, NoHeapPointer: true}
					ctx.BindReg(r9, &d83)
					ctx.BindReg(r9, &d83)
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d83.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d83.Reg, 15)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg}
						ctx.BindReg(d83.Reg, &d84)
					}
					if d84.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: d84.Type, Imm: NewInt(int64(uint64(d84.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d84.Reg, 56)
						ctx.EmitShrRegImm8(d84.Reg, 56)
					}
					if d84.Loc == LocReg && d83.Loc == LocReg && d84.Reg == d83.Reg {
						ctx.TransferReg(d83.Reg)
						d83.Loc = LocNone
					}
					ctx.FreeDesc(&d83)
					d85 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d84)
					ctx.EnsureGoStringHeader(&d85)
					d86 = ctx.EmitSliceElementAddress(&d85, &d84, 1)
					ctx.EnsureDesc(&d86)
					r10 := ctx.AllocRegExcept(d86.Reg)
					ctx.EmitMovRegMemB(r10, d86.Reg, 0)
					ctx.FreeDesc(&d86)
					d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10, NoHeapPointer: true}
					ctx.BindReg(r10, &d87)
					ctx.BindReg(r10, &d87)
					ctx.FreeDesc(&d84)
					ctx.EnsureDesc(&d81)
					ctx.SyncDesc(&d87)
					d88 = d38
					d88.ID = 0
					d89 = d81
					d89.ID = 0
					d90 = ctx.EmitSliceElementAddress(&d88, &d89, int32(1))
					ctx.EmitStoreScalarAt(&d90, &d87, 1)
					ctx.FreeDesc(&d90)
					ctx.FreeDesc(&d89)
					ctx.FreeDesc(&d81)
					ctx.FreeDesc(&d87)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d91)
					}
					if d91.Loc == LocReg && d1.Loc == LocReg && d91.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d91)
					ctx.EmitStoreToStack(d91, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d91)
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					callResults93 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d38}, []uint8{2}, []uint8{1})
					d92 = callResults93[0]
					ctx.EnsureDesc(&d92)
					d94 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d92}, 2)
					ctx.EmitMovPairToResult(&d94, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  46,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "bin2hex",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) {
				return encodeTextHex(a[0])
			}
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
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var phiBase4 int32
				_ = phiBase4
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
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
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
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
				var d94 JITValueDesc
				_ = d94
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					ctx.EnsureDesc(&d2)
					d3 = d2
					_ = d3
					ctx.StabilizeDescForControlFlow(&d3)
					phiBase4 = ctx.AllocStack(int32(16))
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					_ = d5
					lbl7 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d6 = d3
					d6.ID = 0
					d7 = ctx.EmitGetTagDesc(&d6, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpRegImm32(d7.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d5)
					}
					ctx.EmitJmp(lbl7)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitCmpRegImm32(d7.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl9)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d7.Reg, 26)
						r3 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase4)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d13 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d13)
					ctx.BindReg(r1, &d13)
					ctx.FreeDesc(&d2)
					d14 = d13
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d14.Loc == LocImm {
						if d14.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d5
						snap19 := d6
						snap20 := d7
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d13
						snap27 := d14
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d5 = snap18
						d6 = snap19
						d7 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d13 = snap26
						d14 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d13)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					ctx.SyncDesc(&d29)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(encodeTextHex), []JITValueDesc{d29}, 2)
					d30.NoHeapPointer = false
					ctx.BindReg(d30.Reg, &d30)
					ctx.BindReg(d30.Reg2, &d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d30, &result)
						result.Type = d30.Type
					} else {
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d30)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d30)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d30)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d30, &result)
							result.Type = d30.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d31 = args[0]
					d31.ID = 0
					d33 = d31
					ctx.SyncDesc(&d33)
					if d33.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d33.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d33.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d33 = tmpScalar
					}
					d33 = JITPrepareScmerGoArg(ctx, d33)
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d33}, 2)
					ctx.StabilizeDescForControlFlow(&d32)
					ctx.FreeDesc(&d31)
					if d32.SliceSizeKnown {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d32.KnownSliceLen))}
					} else if d32.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d32.Imm.String())))}
					} else if d32.Loc == LocStackTriple {
						d34 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else if d32.Loc == LocStackPair {
						d34 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d32)
						if d32.Loc == LocRegPair || d32.Loc == LocRegTriple {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2, ID: 0}
						} else if d32.Loc == LocReg {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d34)
					ctx.SyncDesc(&d35)
					ctx.SyncDesc(&d34)
					if d35.Loc == LocImm && d34.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d35.Imm.Int() * d34.Imm.Int())}
					} else if d35.Loc == LocImm {
						ctx.EnsureDesc(&d34)
						scratch := ctx.AllocRegExcept(d34.Reg)
						ctx.EmitMovRegReg(scratch, d34.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d35.Imm.Int())
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d36)
					} else if d34.Loc == LocImm {
						ctx.EnsureDesc(&d35)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d35.Reg, d34.Imm.Int())
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg}
						ctx.BindReg(d35.Reg, &d36)
					} else {
						ctx.EnsureDesc(&d35)
						ctx.SyncDesc(&d34)
						ctx.EmitIntBinary(JITIntMul, 64, d35.Reg, &d34)
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg}
						ctx.BindReg(d35.Reg, &d36)
					}
					if d36.Loc == LocReg && d35.Loc == LocReg && d36.Reg == d35.Reg {
						ctx.TransferReg(d35.Reg)
						d35.Loc = LocNone
					}
					ctx.FreeDesc(&d34)
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					callResults37 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d36, d36}, []uint8{3}, []uint8{1})
					d38 = callResults37[0]
					d38.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.FreeDesc(&d36)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(0))
					return bbs[3].Render()
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					if d32.SliceSizeKnown {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d32.KnownSliceLen))}
					} else if d32.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d32.Imm.String())))}
					} else if d32.Loc == LocStackTriple {
						d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else if d32.Loc == LocStackPair {
						d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d32)
						if d32.Loc == LocRegPair || d32.Loc == LocRegTriple {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2, ID: 0}
						} else if d32.Loc == LocReg {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDescsTogether(&d1, &d39)
					if d1.Loc == LocImm && d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d39.Imm.Int())}
					} else if d39.Loc == LocImm {
						r4 := ctx.AllocRegExcept(d1.Reg)
						if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d39.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d39.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d40)
					} else if d1.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d39.Reg)
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d40)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d39.Reg)
						d40 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedLess}
						ctx.BindReg(r6, &d40)
					}
					ctx.FreeDesc(&d39)
					d41 = d40
					ctx.SyncDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d41.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d40)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap42 := d1
						snap43 := d2
						snap44 := d3
						snap45 := d5
						snap46 := d6
						snap47 := d7
						snap48 := d8
						snap49 := d9
						snap50 := d10
						snap51 := d11
						snap52 := d12
						snap53 := d13
						snap54 := d14
						snap55 := d29
						snap56 := d30
						snap57 := d31
						snap58 := d32
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d38
						snap64 := d39
						snap65 := d40
						snap66 := d41
						alloc67 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc67)
						d1 = snap42
						d2 = snap43
						d3 = snap44
						d5 = snap45
						d6 = snap46
						d7 = snap47
						d8 = snap48
						d9 = snap49
						d10 = snap50
						d11 = snap51
						d12 = snap52
						d13 = snap53
						d14 = snap54
						d29 = snap55
						d30 = snap56
						d31 = snap57
						d32 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d38 = snap63
						d39 = snap64
						d40 = snap65
						d41 = snap66
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d38)
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d68)
					ctx.SyncDesc(&d1)
					if d68.Loc == LocImm && d1.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d68.Imm.Int() * d1.Imm.Int())}
					} else if d68.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d68.Imm.Int())
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d69)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d68)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d68.Reg, d1.Imm.Int())
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d68.Reg}
						ctx.BindReg(d68.Reg, &d69)
					} else {
						ctx.EnsureDesc(&d68)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d68.Reg, &d1)
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d68.Reg}
						ctx.BindReg(d68.Reg, &d69)
					}
					if d69.Loc == LocReg && d68.Loc == LocReg && d69.Reg == d68.Reg {
						ctx.TransferReg(d68.Reg)
						d68.Loc = LocNone
					}
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d32)
					d70 = ctx.EmitSliceElementAddress(&d32, &d1, 1)
					ctx.EnsureDesc(&d70)
					r7 := ctx.AllocRegExcept(d70.Reg)
					ctx.EmitMovRegMemB(r7, d70.Reg, 0)
					ctx.FreeDesc(&d70)
					d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d71)
					ctx.BindReg(r7, &d71)
					ctx.EnsureDesc(&d71)
					if d71.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d71.Reg, 4)
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d72)
					}
					if d72.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: d72.Type, Imm: NewInt(int64(uint64(d72.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d72.Reg, 56)
						ctx.EmitShrRegImm8(d72.Reg, 56)
					}
					if d72.Loc == LocReg && d71.Loc == LocReg && d72.Reg == d71.Reg {
						ctx.TransferReg(d71.Reg)
						d71.Loc = LocNone
					}
					ctx.FreeDesc(&d71)
					d73 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d72)
					ctx.EnsureGoStringHeader(&d73)
					d74 = ctx.EmitSliceElementAddress(&d73, &d72, 1)
					ctx.EnsureDesc(&d74)
					r8 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegMemB(r8, d74.Reg, 0)
					ctx.FreeDesc(&d74)
					d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8, NoHeapPointer: true}
					ctx.BindReg(r8, &d75)
					ctx.BindReg(r8, &d75)
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d69)
					ctx.SyncDesc(&d75)
					d76 = d38
					d76.ID = 0
					d77 = d69
					d77.ID = 0
					d78 = ctx.EmitSliceElementAddress(&d76, &d77, int32(1))
					ctx.EmitStoreScalarAt(&d78, &d75, 1)
					ctx.FreeDesc(&d78)
					ctx.FreeDesc(&d77)
					ctx.FreeDesc(&d69)
					ctx.FreeDesc(&d75)
					d79 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d79)
					ctx.SyncDesc(&d1)
					if d79.Loc == LocImm && d1.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d79.Imm.Int() * d1.Imm.Int())}
					} else if d79.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d79.Imm.Int())
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d80)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d79)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d79.Reg, d1.Imm.Int())
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg}
						ctx.BindReg(d79.Reg, &d80)
					} else {
						ctx.EnsureDesc(&d79)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d79.Reg, &d1)
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg}
						ctx.BindReg(d79.Reg, &d80)
					}
					if d80.Loc == LocReg && d79.Loc == LocReg && d80.Reg == d79.Reg {
						ctx.TransferReg(d79.Reg)
						d79.Loc = LocNone
					}
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d80.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d80.Reg)
						ctx.EmitMovRegReg(scratch, d80.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d81)
					}
					if d81.Loc == LocReg && d80.Loc == LocReg && d81.Reg == d80.Reg {
						ctx.TransferReg(d80.Reg)
						d80.Loc = LocNone
					}
					ctx.FreeDesc(&d80)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d32)
					d82 = ctx.EmitSliceElementAddress(&d32, &d1, 1)
					ctx.EnsureDesc(&d82)
					r9 := ctx.AllocRegExcept(d82.Reg)
					ctx.EmitMovRegMemB(r9, d82.Reg, 0)
					ctx.FreeDesc(&d82)
					d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9, NoHeapPointer: true}
					ctx.BindReg(r9, &d83)
					ctx.BindReg(r9, &d83)
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d83.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d83.Reg, 15)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg}
						ctx.BindReg(d83.Reg, &d84)
					}
					if d84.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: d84.Type, Imm: NewInt(int64(uint64(d84.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d84.Reg, 56)
						ctx.EmitShrRegImm8(d84.Reg, 56)
					}
					if d84.Loc == LocReg && d83.Loc == LocReg && d84.Reg == d83.Reg {
						ctx.TransferReg(d83.Reg)
						d83.Loc = LocNone
					}
					ctx.FreeDesc(&d83)
					d85 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d84)
					ctx.EnsureGoStringHeader(&d85)
					d86 = ctx.EmitSliceElementAddress(&d85, &d84, 1)
					ctx.EnsureDesc(&d86)
					r10 := ctx.AllocRegExcept(d86.Reg)
					ctx.EmitMovRegMemB(r10, d86.Reg, 0)
					ctx.FreeDesc(&d86)
					d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10, NoHeapPointer: true}
					ctx.BindReg(r10, &d87)
					ctx.BindReg(r10, &d87)
					ctx.FreeDesc(&d84)
					ctx.EnsureDesc(&d81)
					ctx.SyncDesc(&d87)
					d88 = d38
					d88.ID = 0
					d89 = d81
					d89.ID = 0
					d90 = ctx.EmitSliceElementAddress(&d88, &d89, int32(1))
					ctx.EmitStoreScalarAt(&d90, &d87, 1)
					ctx.FreeDesc(&d90)
					ctx.FreeDesc(&d89)
					ctx.FreeDesc(&d81)
					ctx.FreeDesc(&d87)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d91)
					}
					if d91.Loc == LocReg && d1.Loc == LocReg && d91.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d91)
					ctx.EmitStoreToStack(d91, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d91)
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					callResults93 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d38}, []uint8{2}, []uint8{1})
					d92 = callResults93[0]
					ctx.EnsureDesc(&d92)
					d94 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d92}, 2)
					ctx.EmitMovPairToResult(&d94, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  46,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "hex2bin",

		Fn: func(a ...Scmer) Scmer {
			decoded, err := decodeTextHex(a[0])
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
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d12 JITValueDesc
				_ = d12
				var d14 JITValueDesc
				_ = d14
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d0 = JITPrepareScmerGoArg(ctx, d0)
					ctx.SyncDesc(&d0)
					callResults1 := JITEmitGoCallResults(ctx, GoFuncAddr(decodeTextHex), []JITValueDesc{d0}, []uint8{3, 2}, []uint8{1, 3})
					d2 = callResults1[0]
					_ = d2
					d3 = callResults1[1]
					_ = d3
					ctx.FreeDesc(&d0)
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc != LocReg && d3.Loc != LocRegPair && d3.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpRegImm32(d3.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap6 := d0
						snap7 := d2
						snap8 := d3
						snap9 := d4
						snap10 := d5
						alloc11 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc11)
						d0 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
						d5 = snap10
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d4)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["hex2bin"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d2}, []uint8{2}, []uint8{1})
					d12 = callResults13[0]
					ctx.EnsureDesc(&d12)
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d12}, 2)
					ctx.EmitMovPairToResult(&d14, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  19,
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
				// JITGen native call boundary: stack-assigned array in Go call.
				ctx.Coverage.NativeCalls++
				declaration := declarations["uuid"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
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
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d46 JITValueDesc
				_ = d46
				var d48 JITValueDesc
				_ = d48
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					d1 = d0
					_ = d1
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
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
					if d2.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d4)
					}
					d5 = d4
					ctx.SyncDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d5.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d4)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap6 := d0
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						alloc12 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc12)
						d0 = snap6
						d1 = snap7
						d2 = snap8
						d3 = snap9
						d4 = snap10
						d5 = snap11
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d2, d2}, []uint8{3}, []uint8{1})
					d14 = callResults13[0]
					d14.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() > 0)}
					} else {
						r1 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						d15 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedGreater}
						ctx.BindReg(r1, &d15)
					}
					d16 = d15
					ctx.SyncDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d16.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d15)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap17 := d0
						snap18 := d1
						snap19 := d2
						snap20 := d3
						snap21 := d4
						snap22 := d5
						snap23 := d14
						snap24 := d15
						snap25 := d16
						alloc26 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc26)
						d0 = snap17
						d1 = snap18
						d2 = snap19
						d3 = snap20
						d4 = snap21
						d5 = snap22
						d14 = snap23
						d15 = snap24
						d16 = snap25
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d14)
					d14 = JITPrepareGoSliceArg(ctx, d14)
					if d14.Loc != LocRegTriple && d14.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (crand.Read arg0)")
					}
					ctx.SyncDesc(&d14)
					callResults27 := JITEmitGoCallResults(ctx, GoFuncAddr(crand.Read), []JITValueDesc{d14}, []uint8{1, 2}, []uint8{0, 3})
					d28 = callResults27[0]
					_ = d28
					d29 = callResults27[1]
					_ = d29
					ctx.StabilizeDescForControlFlow(&d29)
					ctx.EnsureDesc(&d29)
					if d29.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d29.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d29)
						if d29.Loc != LocReg && d29.Loc != LocRegPair && d29.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d29.Reg)
						ctx.EmitCmpRegImm32(d29.Reg, 0)
						ctx.EmitSetcc(r2, CondNotEqual)
						d30 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d30)
					}
					d31 = d30
					ctx.EnsureDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d31.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap32 := d0
						snap33 := d1
						snap34 := d2
						snap35 := d3
						snap36 := d4
						snap37 := d5
						snap38 := d14
						snap39 := d15
						snap40 := d16
						snap41 := d28
						snap42 := d29
						snap43 := d30
						snap44 := d31
						alloc45 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc45)
						d0 = snap32
						d1 = snap33
						d2 = snap34
						d3 = snap35
						d4 = snap36
						d5 = snap37
						d14 = snap38
						d15 = snap39
						d16 = snap40
						d28 = snap41
						d29 = snap42
						d30 = snap43
						d31 = snap44
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d30)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					callResults47 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d14}, []uint8{2}, []uint8{1})
					d46 = callResults47[0]
					ctx.EnsureDesc(&d46)
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d46}, 2)
					ctx.EmitMovPairToResult(&d48, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["randomBytes"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
			if a[0].IsBufferString() || (isCompressedText(a[0]) && compressedTextLen(a[0]) >= 128) {
				w := hashTextWriter()
				w.writeCompressedText(a[0])
				return NewString(formatStructuralHash(w.hash))
			}
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
				var d20 JITValueDesc
				_ = d20
				var phiBase21 int32
				_ = phiBase21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d25 JITValueDesc
				_ = d25
				var inlineResultOff24 int32
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
				var phiBase39 int32
				_ = phiBase39
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
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d1 = d0
					d1.ID = 0
					d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x1a))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 26)
						d3 = ctx.DeferBooleanFlags(r0, CondEqual)
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.FreeDesc(&d0)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap5 := d0
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						alloc10 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc10)
						d0 = snap5
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
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
					d13 = args[0]
					d13.ID = 0
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d13 = JITPrepareScmerGoArg(ctx, d13)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d13)
					ctx.EmitGoCallVoid(GoFuncAddr((*schemeTextWriter).writeCompressedText), []JITValueDesc{d11, d13})
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						fieldAddr := uintptr(d11.Imm.Int()) + 8
						r1 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r1, fieldAddr)
						d14 = JITValueDesc{Loc: LocReg, Reg: r1}
						ctx.BindReg(r1, &d14)
					} else {
						off := int32(8)
						baseReg := d11.Reg
						r2 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r2, baseReg, off)
						d14 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d14)
					}
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d14}, 2)
					d15.NoHeapPointer = false
					ctx.BindReg(d15.Reg, &d15)
					ctx.BindReg(d15.Reg2, &d15)
					ctx.FreeDesc(&d14)
					ctx.EnsureDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d15}, 2)
					ctx.EmitMovPairToResult(&d16, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
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
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d18)
					d20 = d18
					_ = d20
					ctx.StabilizeDescForControlFlow(&d20)
					phiBase21 = ctx.AllocStack(int32(32))
					d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(0)}
					_ = d22
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(16)}
					_ = d23
					inlineResultOff24 = ctx.AllocStack(int32(16))
					d25 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inlineResultOff24}
					lbl8 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(0)}
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}, int32(phiBase21)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase21)+int32(16))
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(0)}
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d22)
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.ReclaimUntrackedRegs()
					if d20.SliceSizeKnown {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d20.KnownSliceLen))}
					} else if d20.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d20.Imm.String())))}
					} else if d20.Loc == LocStackTriple {
						d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d20.StackOff + 8, NoHeapPointer: true}
					} else if d20.Loc == LocStackPair {
						d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d20.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d20)
						if d20.Loc == LocRegPair || d20.Loc == LocRegTriple {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg2, ID: 0}
						} else if d20.Loc == LocReg {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d26)
					ctx.EnsureDescsTogether(&d23, &d26)
					if d23.Loc == LocImm && d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d26.Imm.Int())}
					} else if d26.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d23.Reg)
						if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d26.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d26.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, ctx.ScratchReg)
						}
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d27)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d26.Reg)
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d27)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d26.Reg)
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d27)
					}
					ctx.FreeDesc(&d26)
					ctx.ReclaimUntrackedRegs()
					d28 = d27
					ctx.SyncDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						}
					} else {
						ctx.EmitJump(d28.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d27)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
					}
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(0)}
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocRegTriple || d22.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d22}, 2)
					d29.NoHeapPointer = false
					ctx.BindReg(d29.Reg, &d29)
					ctx.BindReg(d29.Reg2, &d29)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d29)
					ctx.EmitCopyDescWords(&d25, &d29, 2)
					ctx.EmitJmp(lbl8)
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(0)}
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase21) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d23)
					ctx.EnsureGoStringHeader(&d20)
					d30 = ctx.EmitSliceElementAddress(&d20, &d23, 1)
					ctx.EnsureDesc(&d30)
					r6 := ctx.AllocRegExcept(d30.Reg)
					ctx.EmitMovRegMemB(r6, d30.Reg, 0)
					ctx.FreeDesc(&d30)
					d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d31)
					ctx.BindReg(r6, &d31)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegReg(r7, d31.Reg)
						ctx.EmitShlRegImm8(r7, 56)
						ctx.EmitShrRegImm8(r7, 56)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d32)
					}
					ctx.FreeDesc(&d31)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDescsTogether(&d22, &d32)
					if d22.Loc == LocImm && d32.Loc == LocImm {
						d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d22.Imm.Int() ^ d32.Imm.Int())}
					} else {
						xorResult := ctx.AllocRegExcept(d22.Reg, d32.Reg)
						ctx.EmitMovToReg(xorResult, d22)
						if d32.Loc == LocReg {
							ctx.EmitXorInt64(xorResult, d32.Reg)
						} else {
							xorOther := ctx.AllocRegExcept(xorResult)
							ctx.EmitMovToReg(xorOther, d32)
							ctx.EmitXorInt64(xorResult, xorOther)
							ctx.FreeReg(xorOther)
						}
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: xorResult}
						ctx.BindReg(xorResult, &d33)
					}
					ctx.FreeDesc(&d32)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d33.Imm.Int() * 1099511628211)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d33.Reg, 1099511628211)
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg}
						ctx.BindReg(d33.Reg, &d34)
					}
					if d34.Loc == LocReg && d33.Loc == LocReg && d34.Reg == d33.Reg {
						ctx.TransferReg(d33.Reg)
						d33.Loc = LocNone
					}
					ctx.EnsureDesc(&d34)
					ctx.EmitStoreToStack(d34, int32(phiBase21)+int32(0))
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.FreeDesc(&d33)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitMovRegReg(scratch, d23.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d35)
					}
					if d35.Loc == LocReg && d23.Loc == LocReg && d35.Reg == d23.Reg {
						ctx.TransferReg(d23.Reg)
						d23.Loc = LocNone
					}
					ctx.EnsureDesc(&d35)
					ctx.EmitStoreToStack(d35, int32(phiBase21)+int32(16))
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl8)
					ctx.EnsureDesc(&d25)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d25}, 2)
					ctx.EmitMovPairToResult(&d36, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d37 = args[0]
					d37.ID = 0
					ctx.EnsureDesc(&d37)
					d38 = d37
					_ = d38
					ctx.StabilizeDescForControlFlow(&d38)
					phiBase39 = ctx.AllocStack(int32(16))
					d40 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase39) + int32(0)}
					_ = d40
					lbl15 := ctx.ReserveLabel()
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_4_1 := int32(-1)
					_ = bbpos_4_1
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_4_2 := int32(-1)
					_ = bbpos_4_2
					lbl18 := ctx.ReserveLabel()
					_ = lbl18
					bbpos_4_3 := int32(-1)
					_ = bbpos_4_3
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d41 = d38
					d41.ID = 0
					d42 = ctx.EmitGetTagDesc(&d41, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d42.Imm.Int()) == uint64(0x13))}
					} else {
						r8 := ctx.AllocRegExcept(d42.Reg)
						ctx.EmitCmpRegImm32(d42.Reg, 19)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondEqual}
						ctx.BindReg(r8, &d43)
					}
					ctx.ReclaimUntrackedRegs()
					d44 = d43
					ctx.SyncDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							ctx.MarkLabel(lbl20)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase39)+int32(0))
							ctx.EmitJmp(lbl18)
						} else {
							ctx.MarkLabel(lbl21)
							ctx.EmitJmp(lbl19)
						}
					} else {
						ctx.EmitJump(d44.Condition, lbl20)
						ctx.EmitJmp(lbl21)
						ctx.FreeDesc(&d43)
						ctx.MarkLabel(lbl20)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase39)+int32(0))
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
					}
					bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r9 := ctx.AllocReg()
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r9, d40)
					}
					ctx.EmitJmp(lbl15)
					bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						d45 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d42.Imm.Int()) == uint64(0x14))}
					} else {
						r10 := ctx.AllocRegExcept(d42.Reg)
						ctx.EmitCmpRegImm32(d42.Reg, 20)
						d45 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondEqual}
						ctx.BindReg(r10, &d45)
					}
					ctx.ReclaimUntrackedRegs()
					d46 = d45
					ctx.SyncDesc(&d46)
					if d46.Loc != LocImm && d46.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					if d46.Loc == LocImm {
						if d46.Imm.Bool() {
							ctx.MarkLabel(lbl22)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase39)+int32(0))
							ctx.EmitJmp(lbl18)
						} else {
							ctx.MarkLabel(lbl23)
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitJump(d46.Condition, lbl22)
						ctx.EmitJmp(lbl23)
						ctx.FreeDesc(&d45)
						ctx.MarkLabel(lbl22)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase39)+int32(0))
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl17)
					}
					bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d42.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d42.Reg, 26)
						r11 := ctx.AllocRegExcept(d42.Reg)
						ctx.EmitSetcc(r11, CondEqual)
						d47 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d47)
					}
					ctx.EnsureDesc(&d47)
					ctx.EmitStoreToStack(d47, int32(phiBase39)+int32(0))
					ctx.StabilizeDescForControlFlow(&d47)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl15)
					d48 = JITValueDesc{Loc: LocReg, Reg: r9}
					ctx.BindReg(r9, &d48)
					ctx.BindReg(r9, &d48)
					ctx.FreeDesc(&d37)
					d49 = d48
					ctx.EnsureDesc(&d49)
					if d49.Loc != LocImm && d49.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d49.Loc == LocImm {
						if d49.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d49.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap50 := d0
						snap51 := d1
						snap52 := d2
						snap53 := d3
						snap54 := d4
						snap55 := d11
						snap56 := d12
						snap57 := d13
						snap58 := d14
						snap59 := d15
						snap60 := d16
						snap61 := d17
						snap62 := d18
						snap63 := d19
						snap64 := d20
						snap65 := d22
						snap66 := d23
						snap67 := d25
						snap68 := d26
						snap69 := d27
						snap70 := d28
						snap71 := d29
						snap72 := d30
						snap73 := d31
						snap74 := d32
						snap75 := d33
						snap76 := d34
						snap77 := d35
						snap78 := d36
						snap79 := d37
						snap80 := d38
						snap81 := d40
						snap82 := d41
						snap83 := d42
						snap84 := d43
						snap85 := d44
						snap86 := d45
						snap87 := d46
						snap88 := d47
						snap89 := d48
						snap90 := d49
						alloc91 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc91)
						d0 = snap50
						d1 = snap51
						d2 = snap52
						d3 = snap53
						d4 = snap54
						d11 = snap55
						d12 = snap56
						d13 = snap57
						d14 = snap58
						d15 = snap59
						d16 = snap60
						d17 = snap61
						d18 = snap62
						d19 = snap63
						d20 = snap64
						d22 = snap65
						d23 = snap66
						d25 = snap67
						d26 = snap68
						d27 = snap69
						d28 = snap70
						d29 = snap71
						d30 = snap72
						d31 = snap73
						d32 = snap74
						d33 = snap75
						d34 = snap76
						d35 = snap77
						d36 = snap78
						d37 = snap79
						d38 = snap80
						d40 = snap81
						d41 = snap82
						d42 = snap83
						d43 = snap84
						d44 = snap85
						d45 = snap86
						d46 = snap87
						d47 = snap88
						d48 = snap89
						d49 = snap90
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d48)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d92 = args[0]
					d92.ID = 0
					d92 = JITPrepareScmerGoArg(ctx, d92)
					ctx.SyncDesc(&d92)
					d93 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d92}, 1)
					d93.NoHeapPointer = true
					ctx.BindReg(d93.Reg, &d93)
					ctx.FreeDesc(&d92)
					ctx.EnsureDesc(&d93)
					if d93.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d93.Imm.Int() >= 128)}
					} else {
						r12 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d93.Reg, 128)
						d94 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r12, &d94)
					}
					ctx.FreeDesc(&d93)
					d95 = d94
					ctx.SyncDesc(&d95)
					if d95.Loc != LocImm && d95.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d95.Loc == LocImm {
						if d95.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d95.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d94)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap96 := d0
						snap97 := d1
						snap98 := d2
						snap99 := d3
						snap100 := d4
						snap101 := d11
						snap102 := d12
						snap103 := d13
						snap104 := d14
						snap105 := d15
						snap106 := d16
						snap107 := d17
						snap108 := d18
						snap109 := d19
						snap110 := d20
						snap111 := d22
						snap112 := d23
						snap113 := d25
						snap114 := d26
						snap115 := d27
						snap116 := d28
						snap117 := d29
						snap118 := d30
						snap119 := d31
						snap120 := d32
						snap121 := d33
						snap122 := d34
						snap123 := d35
						snap124 := d36
						snap125 := d37
						snap126 := d38
						snap127 := d40
						snap128 := d41
						snap129 := d42
						snap130 := d43
						snap131 := d44
						snap132 := d45
						snap133 := d46
						snap134 := d47
						snap135 := d48
						snap136 := d49
						snap137 := d92
						snap138 := d93
						snap139 := d94
						snap140 := d95
						alloc141 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc141)
						d0 = snap96
						d1 = snap97
						d2 = snap98
						d3 = snap99
						d4 = snap100
						d11 = snap101
						d12 = snap102
						d13 = snap103
						d14 = snap104
						d15 = snap105
						d16 = snap106
						d17 = snap107
						d18 = snap108
						d19 = snap109
						d20 = snap110
						d22 = snap111
						d23 = snap112
						d25 = snap113
						d26 = snap114
						d27 = snap115
						d28 = snap116
						d29 = snap117
						d30 = snap118
						d31 = snap119
						d32 = snap120
						d33 = snap121
						d34 = snap122
						d35 = snap123
						d36 = snap124
						d37 = snap125
						d38 = snap126
						d40 = snap127
						d41 = snap128
						d42 = snap129
						d43 = snap130
						d44 = snap131
						d45 = snap132
						d46 = snap133
						d47 = snap134
						d48 = snap135
						d49 = snap136
						d92 = snap137
						d93 = snap138
						d94 = snap139
						d95 = snap140
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 58,
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
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d0)
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
					ctx.SyncDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d2.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d1)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["stable_structural_hash"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d7 = ctx.EmitGoCallScalar(GoFuncAddr(func() *schemeTextWriter { return new(schemeTextWriter) }), nil, 1)
					ctx.BindReg(d7.Reg, &d7)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *schemeTextWriter, value uint64) { base.hash = value }), []JITValueDesc{d7, d8})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					ctx.StabilizeDescForControlFlow(&d7)
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() == 2)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d9.Reg, 2)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d10)
					}
					ctx.FreeDesc(&d9)
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d11.Condition, lbl8)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d10)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap12 := d0
						snap13 := d1
						snap14 := d2
						snap15 := d7
						snap16 := d8
						snap17 := d9
						snap18 := d10
						snap19 := d11
						alloc20 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc20)
						d0 = snap12
						d1 = snap13
						d2 = snap14
						d7 = snap15
						d8 = snap16
						d9 = snap17
						d10 = snap18
						d11 = snap19
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() > 2)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d21.Reg, 2)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedGreater}
						ctx.BindReg(r2, &d22)
					}
					ctx.FreeDesc(&d21)
					d23 = d22
					ctx.SyncDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d23.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d22)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap24 := d0
						snap25 := d1
						snap26 := d2
						snap27 := d7
						snap28 := d8
						snap29 := d9
						snap30 := d10
						snap31 := d11
						snap32 := d21
						snap33 := d22
						snap34 := d23
						alloc35 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc35)
						d0 = snap24
						d1 = snap25
						d2 = snap26
						d7 = snap27
						d8 = snap28
						d9 = snap29
						d10 = snap30
						d11 = snap31
						d21 = snap32
						d22 = snap33
						d23 = snap34
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d7)
					d36 = args[0]
					d36.ID = 0
					if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocRegTriple || d7.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d36 = JITPrepareScmerGoArg(ctx, d36)
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d37.Loc == LocRegPair || d37.Loc == LocStackPair || d37.Loc == LocRegTriple || d37.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d38.Loc == LocRegPair || d38.Loc == LocStackPair || d38.Loc == LocRegTriple || d38.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d39 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					if d39.Loc == LocRegPair || d39.Loc == LocStackPair || d39.Loc == LocRegTriple || d39.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d7)
					ctx.SyncDesc(&d36)
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d38)
					ctx.SyncDesc(&d39)
					ctx.EmitGoCallVoid(GoFuncAddr(serializeEx), []JITValueDesc{d7, d36, d37, d38, d39})
					ctx.FreeDesc(&d39)
					ctx.FreeDesc(&d36)
					return bbs[5].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						fieldAddr := uintptr(d7.Imm.Int()) + 8
						r3 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r3, fieldAddr)
						d40 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d40)
					} else {
						off := int32(8)
						baseReg := d7.Reg
						r4 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r4, baseReg, off)
						d40 = JITValueDesc{Loc: LocReg, Reg: r4}
						ctx.BindReg(r4, &d40)
					}
					if d40.Loc == LocRegPair || d40.Loc == LocStackPair || d40.Loc == LocRegTriple || d40.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d40)
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d40}, 2)
					d41.NoHeapPointer = false
					ctx.BindReg(d41.Reg, &d41)
					ctx.BindReg(d41.Reg2, &d41)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d41)
					d42 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d41}, 2)
					ctx.EmitMovPairToResult(&d42, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d7)
					d43 = args[0]
					d43.ID = 0
					if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocRegTriple || d7.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d43 = JITPrepareScmerGoArg(ctx, d43)
					ctx.SyncDesc(&d7)
					ctx.SyncDesc(&d43)
					ctx.EmitGoCallVoid(GoFuncAddr(WriteStringValue), []JITValueDesc{d7, d43})
					ctx.FreeDesc(&d43)
					return bbs[5].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d44 = args[1]
					d44.ID = 0
					d46 = d44
					d46.ID = 0
					d45 = ctx.EmitBoolDesc(&d46, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d44)
					d47 = d45
					ctx.EnsureDesc(&d47)
					if d47.Loc != LocImm && d47.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d47.Loc == LocImm {
						if d47.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d47.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap48 := d0
						snap49 := d1
						snap50 := d2
						snap51 := d7
						snap52 := d8
						snap53 := d9
						snap54 := d10
						snap55 := d11
						snap56 := d21
						snap57 := d22
						snap58 := d23
						snap59 := d36
						snap60 := d37
						snap61 := d38
						snap62 := d39
						snap63 := d40
						snap64 := d41
						snap65 := d42
						snap66 := d43
						snap67 := d44
						snap68 := d45
						snap69 := d46
						snap70 := d47
						alloc71 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc71)
						d0 = snap48
						d1 = snap49
						d2 = snap50
						d7 = snap51
						d8 = snap52
						d9 = snap53
						d10 = snap54
						d11 = snap55
						d21 = snap56
						d22 = snap57
						d23 = snap58
						d36 = snap59
						d37 = snap60
						d38 = snap61
						d39 = snap62
						d40 = snap63
						d41 = snap64
						d42 = snap65
						d43 = snap66
						d44 = snap67
						d45 = snap68
						d46 = snap69
						d47 = snap70
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d45)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  33,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "md5",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsBufferString() || (isCompressedText(a[0]) && compressedTextLen(a[0]) >= 256) {
				return compressedTextDigest(a[0], 0)
			}
			sum := md5.Sum([]byte(String(a[0])))
			return NewString(hex.EncodeToString(sum[:]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes the MD5 digest of a string, returns a 32-character lowercase hex string",
			Params: []*TypeDescriptor{{Kind: "string", Label: "str", Description: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: stack-assigned array in Go call.
				ctx.Coverage.NativeCalls++
				declaration := declarations["md5"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITInlineCost:      65535,
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sha1",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsBufferString() || (isCompressedText(a[0]) && compressedTextLen(a[0]) >= 256) {
				return compressedTextDigest(a[0], 1)
			}
			sum := sha1.Sum([]byte(String(a[0])))
			return NewString(hex.EncodeToString(sum[:]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes the SHA-1 digest of a string, returns a 40-character lowercase hex string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "str", Description: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: stack-assigned array in Go call.
				ctx.Coverage.NativeCalls++
				declaration := declarations["sha1"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCost:      65535,
			JITInlineCallbacks: false,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sha256",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsBufferString() || (isCompressedText(a[0]) && compressedTextLen(a[0]) >= 256) {
				return compressedTextDigest(a[0], 2)
			}
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
				var d20 JITValueDesc
				_ = d20
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
				var phiBase27 int32
				_ = phiBase27
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
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d1 = d0
					d1.ID = 0
					d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(0x1a))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 26)
						d3 = ctx.DeferBooleanFlags(r0, CondEqual)
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					ctx.ReclaimUntrackedRegs()
					ctx.FreeDesc(&d0)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap5 := d0
						snap6 := d1
						snap7 := d2
						snap8 := d3
						snap9 := d4
						alloc10 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc10)
						d0 = snap5
						d1 = snap6
						d2 = snap7
						d3 = snap8
						d4 = snap9
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d11 = args[0]
					d11.ID = 0
					d11 = JITPrepareScmerGoArg(ctx, d11)
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d12)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextDigest), []JITValueDesc{d11, d12}, 2)
					d13.NoHeapPointer = false
					ctx.BindReg(d13.Reg, &d13)
					ctx.BindReg(d13.Reg2, &d13)
					ctx.FreeDesc(&d12)
					ctx.FreeDesc(&d11)
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[32]byte { return new([32]byte) }), nil, 1)
					d15 = args[0]
					d15.ID = 0
					d17 = d15
					ctx.SyncDesc(&d17)
					if d17.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d17.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d17.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d17 = tmpScalar
					}
					d17 = JITPrepareScmerGoArg(ctx, d17)
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d17}, 2)
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					callResults19 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d16}, []uint8{3}, []uint8{1})
					d18 = callResults19[0]
					d18.Type = tagSlice
					d18 = JITPrepareGoSliceArg(ctx, d18)
					if d18.Loc != LocRegTriple && d18.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (sha256.Sum256 arg0)")
					}
					ctx.SyncDesc(&d18)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr((func(arg0 []byte) *[32]byte { value := sha256.Sum256(arg0); return &value })), []JITValueDesc{d18}, 1)
					d20.NoHeapPointer = false
					ctx.BindReg(d20.Reg, &d20)
					ctx.EnsureDesc(&d20)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *[32]byte) { *dst = *src }), []JITValueDesc{d14, d20})
					sliceResults21 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[32]byte) []byte { return value[0:32:32] }), []JITValueDesc{d14}, []uint8{3}, []uint8{1})
					d22 = sliceResults21[0]
					d22 = JITPrepareGoSliceArg(ctx, d22)
					if d22.Loc != LocRegTriple && d22.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (hex.EncodeToString arg0)")
					}
					ctx.SyncDesc(&d22)
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(hex.EncodeToString), []JITValueDesc{d22}, 2)
					d23.NoHeapPointer = false
					ctx.BindReg(d23.Reg, &d23)
					ctx.BindReg(d23.Reg2, &d23)
					ctx.EnsureDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d23}, 2)
					ctx.EmitMovPairToResult(&d24, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d25 = args[0]
					d25.ID = 0
					ctx.EnsureDesc(&d25)
					d26 = d25
					_ = d26
					ctx.StabilizeDescForControlFlow(&d26)
					phiBase27 = ctx.AllocStack(int32(16))
					d28 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase27) + int32(0)}
					_ = d28
					lbl7 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d28 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase27) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d29 = d26
					d29.ID = 0
					d30 = ctx.EmitGetTagDesc(&d29, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d30.Imm.Int()) == uint64(0x13))}
					} else {
						r1 := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitCmpRegImm32(d30.Reg, 19)
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d31)
					}
					ctx.ReclaimUntrackedRegs()
					d32 = d31
					ctx.SyncDesc(&d32)
					if d32.Loc != LocImm && d32.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d32.Loc == LocImm {
						if d32.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase27)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d32.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d31)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase27)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d28 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase27) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r2 := ctx.AllocReg()
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d28)
					}
					ctx.EmitJmp(lbl7)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d28 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase27) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						d33 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d30.Imm.Int()) == uint64(0x14))}
					} else {
						r3 := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitCmpRegImm32(d30.Reg, 20)
						d33 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d33)
					}
					ctx.ReclaimUntrackedRegs()
					d34 = d33
					ctx.SyncDesc(&d34)
					if d34.Loc != LocImm && d34.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d34.Loc == LocImm {
						if d34.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase27)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d34.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d33)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase27)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl9)
					}
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d28 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase27) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d30.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d30.Reg, 26)
						r4 := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitSetcc(r4, CondEqual)
						d35 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d35)
					}
					ctx.EnsureDesc(&d35)
					ctx.EmitStoreToStack(d35, int32(phiBase27)+int32(0))
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d36 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d36)
					ctx.BindReg(r2, &d36)
					ctx.FreeDesc(&d25)
					d37 = d36
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocImm && d37.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d37.Loc == LocImm {
						if d37.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d37.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap38 := d0
						snap39 := d1
						snap40 := d2
						snap41 := d3
						snap42 := d4
						snap43 := d11
						snap44 := d12
						snap45 := d13
						snap46 := d14
						snap47 := d15
						snap48 := d16
						snap49 := d17
						snap50 := d18
						snap51 := d20
						snap52 := d22
						snap53 := d23
						snap54 := d24
						snap55 := d25
						snap56 := d26
						snap57 := d28
						snap58 := d29
						snap59 := d30
						snap60 := d31
						snap61 := d32
						snap62 := d33
						snap63 := d34
						snap64 := d35
						snap65 := d36
						snap66 := d37
						alloc67 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc67)
						d0 = snap38
						d1 = snap39
						d2 = snap40
						d3 = snap41
						d4 = snap42
						d11 = snap43
						d12 = snap44
						d13 = snap45
						d14 = snap46
						d15 = snap47
						d16 = snap48
						d17 = snap49
						d18 = snap50
						d20 = snap51
						d22 = snap52
						d23 = snap53
						d24 = snap54
						d25 = snap55
						d26 = snap56
						d28 = snap57
						d29 = snap58
						d30 = snap59
						d31 = snap60
						d32 = snap61
						d33 = snap62
						d34 = snap63
						d35 = snap64
						d36 = snap65
						d37 = snap66
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d36)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d68 = args[0]
					d68.ID = 0
					d68 = JITPrepareScmerGoArg(ctx, d68)
					ctx.SyncDesc(&d68)
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d68}, 1)
					d69.NoHeapPointer = true
					ctx.BindReg(d69.Reg, &d69)
					ctx.FreeDesc(&d68)
					ctx.EnsureDesc(&d69)
					if d69.Loc == LocImm {
						d70 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d69.Imm.Int() >= 256)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d69.Reg, 256)
						d70 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r5, &d70)
					}
					ctx.FreeDesc(&d69)
					d71 = d70
					ctx.SyncDesc(&d71)
					if d71.Loc != LocImm && d71.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d71.Loc == LocImm {
						if d71.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d71.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d70)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap72 := d0
						snap73 := d1
						snap74 := d2
						snap75 := d3
						snap76 := d4
						snap77 := d11
						snap78 := d12
						snap79 := d13
						snap80 := d14
						snap81 := d15
						snap82 := d16
						snap83 := d17
						snap84 := d18
						snap85 := d20
						snap86 := d22
						snap87 := d23
						snap88 := d24
						snap89 := d25
						snap90 := d26
						snap91 := d28
						snap92 := d29
						snap93 := d30
						snap94 := d31
						snap95 := d32
						snap96 := d33
						snap97 := d34
						snap98 := d35
						snap99 := d36
						snap100 := d37
						snap101 := d68
						snap102 := d69
						snap103 := d70
						snap104 := d71
						alloc105 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc105)
						d0 = snap72
						d1 = snap73
						d2 = snap74
						d3 = snap75
						d4 = snap76
						d11 = snap77
						d12 = snap78
						d13 = snap79
						d14 = snap80
						d15 = snap81
						d16 = snap82
						d17 = snap83
						d18 = snap84
						d20 = snap85
						d22 = snap86
						d23 = snap87
						d24 = snap88
						d25 = snap89
						d26 = snap90
						d28 = snap91
						d29 = snap92
						d30 = snap93
						d31 = snap94
						d32 = snap95
						d33 = snap96
						d34 = snap97
						d35 = snap98
						d36 = snap99
						d37 = snap100
						d68 = snap101
						d69 = snap102
						d70 = snap103
						d71 = snap104
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCost:      40,
			JITInlineCallbacks: false,
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
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					d12 = d10
					ctx.SyncDesc(&d12)
					if d12.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d12.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d12.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d12 = tmpScalar
					}
					d12 = JITPrepareScmerGoArg(ctx, d12)
					if d12.Loc != LocRegPair && d12.Loc != LocStackPair && d12.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d12}, 2)
					ctx.FreeDesc(&d10)
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
						panic("jit: generic call arg expects 2-word value (regexp.Compile arg0)")
					}
					ctx.SyncDesc(&d11)
					callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(regexp.Compile), []JITValueDesc{d11}, []uint8{1, 2}, []uint8{1, 3})
					d14 = callResults13[0]
					_ = d14
					d15 = callResults13[1]
					_ = d15
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d15)
						if d15.Loc != LocReg && d15.Loc != LocRegPair && d15.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d16)
					}
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap18 := d0
						snap19 := d1
						snap20 := d2
						snap21 := d3
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d14
						snap27 := d15
						snap28 := d16
						snap29 := d17
						alloc30 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc30)
						d0 = snap18
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d14 = snap26
						d15 = snap27
						d16 = snap28
						d17 = snap29
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d16)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d31 = args[1]
					d31.ID = 0
					d33 = d31
					d33.ID = 0
					d32 = ctx.EmitTagEqualsBorrowed(&d33, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d31)
					d34 = d32
					ctx.EnsureDesc(&d34)
					if d34.Loc != LocImm && d34.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d34.Loc == LocImm {
						if d34.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d34.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap35 := d0
						snap36 := d1
						snap37 := d2
						snap38 := d3
						snap39 := d9
						snap40 := d10
						snap41 := d11
						snap42 := d12
						snap43 := d14
						snap44 := d15
						snap45 := d16
						snap46 := d17
						snap47 := d31
						snap48 := d32
						snap49 := d33
						snap50 := d34
						alloc51 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc51)
						d0 = snap35
						d1 = snap36
						d2 = snap37
						d3 = snap38
						d9 = snap39
						d10 = snap40
						d11 = snap41
						d12 = snap42
						d14 = snap43
						d15 = snap44
						d16 = snap45
						d17 = snap46
						d31 = snap47
						d32 = snap48
						d33 = snap49
						d34 = snap50
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d32)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["regexp_test"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d52 = args[0]
					d52.ID = 0
					d54 = d52
					ctx.SyncDesc(&d54)
					if d54.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d54.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d54.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d54 = tmpScalar
					}
					d54 = JITPrepareScmerGoArg(ctx, d54)
					if d54.Loc != LocRegPair && d54.Loc != LocStackPair && d54.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d53 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d54}, 2)
					ctx.FreeDesc(&d52)
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d53)
					if d53.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d53.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d53.Imm)
						ptrWord, _ := d53.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d53.Imm.String())))
						d53 = tmpPair
					} else if d53.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d53.Type, Reg: ctx.AllocRegExcept(d53.Reg), Reg2: ctx.AllocRegExcept(d53.Reg)}
						switch d53.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d53)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d53)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d53)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d53)
						d53 = tmpPair
					}
					if d53.Loc != LocRegPair && d53.Loc != LocStackPair && d53.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*regexp.Regexp).MatchString arg1)")
					}
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d53)
					d55 = ctx.EmitGoCallScalar(GoFuncAddr((*regexp.Regexp).MatchString), []JITValueDesc{d14, d53}, 1)
					d55.NoHeapPointer = true
					ctx.EmitAndRegImm32(d55.Reg, 1)
					d55.Type = tagBool
					ctx.BindReg(d55.Reg, &d55)
					ctx.SyncDesc(&d55)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d55) {
						return d55
					}
					if d55.Loc == LocImm {
						ctx.EmitMakeBool(result, d55)
					} else {
						ctx.EmitMovToReg(result.Reg2, d55)
						d56 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d56)
						if d55.Loc == LocReg && d55.Reg != result.Reg2 {
							ctx.FreeReg(d55.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      28,
		},
		Optimize: optimizeRegexpTest,
	})
	registerJITRegexBuiltins()

	Declare(&Globalenv, &Declaration{
		Name: "utf16_length",
		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			return NewInt(utf16Length(String(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "counts UTF16 code units without encoding a copy",
			Params: []*TypeDescriptor{{Kind: "string", Label: "value"}},
			Return: &TypeDescriptor{Kind: "int"}, Const: true,
			JITInlineCost: 12,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["utf16_length"]
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
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[0]
					d10.ID = 0
					d12 = d10
					ctx.SyncDesc(&d12)
					if d12.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d12.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d12.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d12 = tmpScalar
					}
					d12 = JITPrepareScmerGoArg(ctx, d12)
					if d12.Loc != LocRegPair && d12.Loc != LocStackPair && d12.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d12}, 2)
					ctx.FreeDesc(&d10)
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
						panic("jit: generic call arg expects 2-word value (utf16Length arg0)")
					}
					ctx.SyncDesc(&d11)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(utf16Length), []JITValueDesc{d11}, 1)
					d13.NoHeapPointer = true
					ctx.BindReg(d13.Reg, &d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						ctx.EmitMakeInt(result, d13)
					} else {
						ctx.EmitMovToReg(result.Reg2, d13)
						d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d14)
						if d13.Loc == LocReg && d13.Reg != result.Reg2 {
							ctx.FreeReg(d13.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "utf16_prefix",
		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			if !a[1].IsInt() {
				panic("UTF16 prefix limit requires an integer")
			}
			if a[1].Int() == 0 {
				return NewString("")
			}
			return NewString(utf16Prefix(String(a[0]), a[1].Int()))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a whole-scalar prefix bounded by UTF16 code units",
			Params: []*TypeDescriptor{{Kind: "string", Label: "value"}, {Kind: "int", Label: "limit"}},
			Return: &TypeDescriptor{Kind: "string"}, Const: true,
			JITInlineCost: 32,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["utf16_prefix"]
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
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
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
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap4 := d0
						snap5 := d1
						snap6 := d2
						snap7 := d3
						alloc8 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc8)
						d0 = snap4
						d1 = snap5
						d2 = snap6
						d3 = snap7
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d9, &result)
						result.Type = d9.Type
					} else {
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d9)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d9)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d9)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d9, &result)
							result.Type = d9.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d10 = args[1]
					d10.ID = 0
					d12 = d10
					d12.ID = 0
					d11 = ctx.EmitTagEqualsBorrowed(&d12, tagInt, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d10)
					d13 = d11
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d13.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap14 := d0
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d9
						snap19 := d10
						snap20 := d11
						snap21 := d12
						snap22 := d13
						alloc23 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc23)
						d0 = snap14
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d9 = snap18
						d10 = snap19
						d11 = snap20
						d12 = snap21
						d13 = snap22
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d24 = args[1]
					d24.ID = 0
					d26 = d24
					d26.ID = 0
					d25 = ctx.EmitTagEqualsBorrowed(&d26, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d24)
					d27 = d25
					ctx.EnsureDesc(&d27)
					if d27.Loc != LocImm && d27.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d27.Loc == LocImm {
						if d27.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d27.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap28 := d0
						snap29 := d1
						snap30 := d2
						snap31 := d3
						snap32 := d9
						snap33 := d10
						snap34 := d11
						snap35 := d12
						snap36 := d13
						snap37 := d24
						snap38 := d25
						snap39 := d26
						snap40 := d27
						alloc41 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc41)
						d0 = snap28
						d1 = snap29
						d2 = snap30
						d3 = snap31
						d9 = snap32
						d10 = snap33
						d11 = snap34
						d12 = snap35
						d13 = snap36
						d24 = snap37
						d25 = snap38
						d26 = snap39
						d27 = snap40
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d25)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["utf16_prefix"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d42 = args[1]
					d42.ID = 0
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int())}
					} else if d42.Type == tagInt && d42.Loc == LocRegPair {
						ctx.FreeReg(d42.Reg)
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg2}
						ctx.BindReg(d42.Reg2, &d43)
						ctx.BindReg(d42.Reg2, &d43)
					} else if d42.Type == tagInt && d42.Loc == LocReg {
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
						ctx.BindReg(d42.Reg, &d43)
						ctx.BindReg(d42.Reg, &d43)
					} else {
						d43 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d42}, 1)
						d43.Type = tagInt
						ctx.BindReg(d43.Reg, &d43)
					}
					ctx.FreeDesc(&d42)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d43.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d43.Reg, 0)
						d44 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d44)
					}
					ctx.FreeDesc(&d43)
					d45 = d44
					ctx.SyncDesc(&d45)
					if d45.Loc != LocImm && d45.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d45.Loc == LocImm {
						if d45.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitJump(d45.Condition, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d44)
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap46 := d0
						snap47 := d1
						snap48 := d2
						snap49 := d3
						snap50 := d9
						snap51 := d10
						snap52 := d11
						snap53 := d12
						snap54 := d13
						snap55 := d24
						snap56 := d25
						snap57 := d26
						snap58 := d27
						snap59 := d42
						snap60 := d43
						snap61 := d44
						snap62 := d45
						alloc63 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc63)
						d0 = snap46
						d1 = snap47
						d2 = snap48
						d3 = snap49
						d9 = snap50
						d10 = snap51
						d11 = snap52
						d12 = snap53
						d13 = snap54
						d24 = snap55
						d25 = snap56
						d26 = snap57
						d27 = snap58
						d42 = snap59
						d43 = snap60
						d44 = snap61
						d45 = snap62
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d64 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d64}, 2)
					ctx.EmitMovPairToResult(&d65, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
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
					d69 = args[1]
					d69.ID = 0
					if d69.Loc == LocImm {
						d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d69.Imm.Int())}
					} else if d69.Type == tagInt && d69.Loc == LocRegPair {
						ctx.FreeReg(d69.Reg)
						d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg2}
						ctx.BindReg(d69.Reg2, &d70)
						ctx.BindReg(d69.Reg2, &d70)
					} else if d69.Type == tagInt && d69.Loc == LocReg {
						d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg}
						ctx.BindReg(d69.Reg, &d70)
						ctx.BindReg(d69.Reg, &d70)
					} else {
						d70 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d69}, 1)
						d70.Type = tagInt
						ctx.BindReg(d70.Reg, &d70)
					}
					ctx.FreeDesc(&d69)
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
						panic("jit: generic call arg expects 2-word value (utf16Prefix arg0)")
					}
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d67)
					ctx.SyncDesc(&d70)
					d71 = ctx.EmitGoCallScalar(GoFuncAddr(utf16Prefix), []JITValueDesc{d67, d70}, 2)
					d71.NoHeapPointer = false
					ctx.BindReg(d71.Reg, &d71)
					ctx.BindReg(d71.Reg2, &d71)
					ctx.FreeDesc(&d70)
					ctx.EnsureDesc(&d71)
					d72 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d71}, 2)
					ctx.EmitMovPairToResult(&d72, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
		},
	})

}

// optimizeRegexpMatches shares the constant-pattern lifetime used by the
// other regex operators. Only the input text varies between invocations.
func optimizeRegexpMatches(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if td != nil && td.Const {
		return result, td
	}
	rv, ok := scmerSlice(result)
	if !ok || len(rv) != 3 || !rv[2].IsString() {
		return result, td
	}
	re, err := regexp.Compile(rv[2].String())
	if err != nil {
		return result, td // Invalid patterns still fail when the call executes.
	}
	// Keep a declared callable identity so Eval and the JIT run the same
	// precompiled-regex operation; the JIT emitter drives a native byte walk and
	// returns input slice-views, the interpreter Fn does the same.
	return NewSlice([]Scmer{
		NewSymbol(jitConstantRegexpMatchesName),
		NewRegex(re),
		rv[1],
	}), td
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
