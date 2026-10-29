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
	init_uuid()
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
			JITInlineCallbacks: false,
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
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d8)
					}
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d9.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d8)
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
					_ = d24
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
					_ = d27
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
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d28)
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
				d3 := jitMaterializeVirtualGoSlice(ctx, args)
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
			JITInlineCallbacks: false,
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
				var phiBase9 int32
				_ = phiBase9
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
				var d18 JITValueDesc
				_ = d18
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
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
				var d70 JITValueDesc
				_ = d70
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
				var d155 JITValueDesc
				_ = d155
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
					ctx.EnsureDesc(&d3)
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
						boxed := JITPrepareScmerGoArg(ctx, d3)
						d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
					inlineHome8 := d7
					inlineHome8.ID = 0
					phiBase9 = ctx.AllocStack(int32(16))
					d10 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase9) + int32(0)}
					_ = d10
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
					d7 = inlineHome8
					d10 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase9) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d11 = d7
					d11.ID = 0
					d12 = ctx.EmitGetTagDesc(&d11, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d12)
					inlineHome13 := d12
					inlineHome13.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpRegImm32(d12.Reg, 19)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.SyncDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase9)+int32(0))
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitJump(d15.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d14)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase9)+int32(0))
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d7 = inlineHome8
					d12 = inlineHome13
					d10 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase9) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d10)
					}
					ctx.EmitJmp(lbl9)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d7 = inlineHome8
					d12 = inlineHome13
					d10 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase9) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpRegImm32(d12.Reg, 20)
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d16)
					}
					ctx.ReclaimUntrackedRegs()
					d17 = d16
					ctx.SyncDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase9)+int32(0))
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl16)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase9)+int32(0))
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl11)
					}
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d7 = inlineHome8
					d12 = inlineHome13
					d10 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase9) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d12.Reg, 26)
						r3 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d18)
					}
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(phiBase9)+int32(0))
					ctx.StabilizeDescForControlFlow(&d18)
					inlineHome19 := d18
					inlineHome19.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl9)
					d20 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d20)
					ctx.BindReg(r1, &d20)
					ctx.FreeDesc(&d6)
					d21 = d20
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap22 := d1
						snap23 := d2
						snap24 := d3
						snap25 := d4
						snap26 := d5
						snap27 := d6
						snap28 := d7
						snap29 := d10
						snap30 := d11
						snap31 := d12
						snap32 := d14
						snap33 := d15
						snap34 := d16
						snap35 := d17
						snap36 := d18
						snap37 := d20
						snap38 := d21
						alloc39 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc39)
						d1 = snap22
						d2 = snap23
						d3 = snap24
						d4 = snap25
						d5 = snap26
						d6 = snap27
						d7 = snap28
						d10 = snap29
						d11 = snap30
						d12 = snap31
						d14 = snap32
						d15 = snap33
						d16 = snap34
						d17 = snap35
						d18 = snap36
						d20 = snap37
						d21 = snap38
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d20)
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
					d40 = args[0]
					d40.ID = 0
					d40 = JITPrepareScmerGoArg(ctx, d40)
					ctx.SyncDesc(&d40)
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d40}, 1)
					d41.NoHeapPointer = true
					ctx.BindReg(d41.Reg, &d41)
					ctx.StabilizeDescForControlFlow(&d41)
					ctx.FreeDesc(&d40)
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d42
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d42.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d42.Reg, 2)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedGreater}
						ctx.BindReg(r4, &d43)
					}
					ctx.FreeDesc(&d42)
					d44 = d43
					ctx.SyncDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.SyncDesc(&d41)
						if d41.Loc == LocReg || d41.Loc == LocFPReg {
							ctx.ProtectReg(d41.Reg)
						} else if d41.Loc == LocRegPair {
							ctx.ProtectReg(d41.Reg)
							ctx.ProtectReg(d41.Reg2)
						}
						d45 = d41
						if d45.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d45)
						ctx.EmitStoreToStack(d45, int32(bbs[4].PhiBase)+int32(0))
						if d41.Loc == LocReg || d41.Loc == LocFPReg {
							ctx.UnprotectReg(d41.Reg)
						} else if d41.Loc == LocRegPair {
							ctx.UnprotectReg(d41.Reg)
							ctx.UnprotectReg(d41.Reg2)
						}
						return bbs[4].Render()
					}
					lbl18 := ctx.ReserveLabel()
					ctx.EmitJump(d44.Condition, lbl4)
					ctx.EmitJmp(lbl18)
					ctx.FreeDesc(&d43)
					snap46 := d1
					snap47 := d2
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d6
					snap52 := d7
					snap53 := d10
					snap54 := d11
					snap55 := d12
					snap56 := d14
					snap57 := d15
					snap58 := d16
					snap59 := d17
					snap60 := d18
					snap61 := d20
					snap62 := d21
					snap63 := d40
					snap64 := d41
					snap65 := d42
					snap66 := d43
					snap67 := d44
					snap68 := d45
					alloc69 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl18)
					ctx.SyncDesc(&d41)
					if d41.Loc == LocReg || d41.Loc == LocFPReg {
						ctx.ProtectReg(d41.Reg)
					} else if d41.Loc == LocRegPair {
						ctx.ProtectReg(d41.Reg)
						ctx.ProtectReg(d41.Reg2)
					}
					d70 = d41
					if d70.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d70)
					ctx.EmitStoreToStack(d70, int32(bbs[4].PhiBase)+int32(0))
					if d41.Loc == LocReg || d41.Loc == LocFPReg {
						ctx.UnprotectReg(d41.Reg)
					} else if d41.Loc == LocRegPair {
						ctx.UnprotectReg(d41.Reg)
						ctx.UnprotectReg(d41.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc69)
					d1 = snap46
					d2 = snap47
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d6 = snap51
					d7 = snap52
					d10 = snap53
					d11 = snap54
					d12 = snap55
					d14 = snap56
					d15 = snap57
					d16 = snap58
					d17 = snap59
					d18 = snap60
					d20 = snap61
					d21 = snap62
					d40 = snap63
					d41 = snap64
					d42 = snap65
					d43 = snap66
					d44 = snap67
					d45 = snap68
					if !bbs[4].Rendered {
						snap71 := d1
						snap72 := d2
						snap73 := d3
						snap74 := d4
						snap75 := d5
						snap76 := d6
						snap77 := d7
						snap78 := d10
						snap79 := d11
						snap80 := d12
						snap81 := d14
						snap82 := d15
						snap83 := d16
						snap84 := d17
						snap85 := d18
						snap86 := d20
						snap87 := d21
						snap88 := d40
						snap89 := d41
						snap90 := d42
						snap91 := d43
						snap92 := d44
						snap93 := d45
						snap94 := d70
						alloc95 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc95)
						d1 = snap71
						d2 = snap72
						d3 = snap73
						d4 = snap74
						d5 = snap75
						d6 = snap76
						d7 = snap77
						d10 = snap78
						d11 = snap79
						d12 = snap80
						d14 = snap81
						d15 = snap82
						d16 = snap83
						d17 = snap84
						d18 = snap85
						d20 = snap86
						d21 = snap87
						d40 = snap88
						d41 = snap89
						d42 = snap90
						d43 = snap91
						d44 = snap92
						d45 = snap93
						d70 = snap94
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
					d96 = args[0]
					d96.ID = 0
					d98 = d96
					ctx.SyncDesc(&d98)
					if d98.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d98.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d98.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d98 = tmpScalar
					}
					d98 = JITPrepareScmerGoArg(ctx, d98)
					if d98.Loc != LocRegPair && d98.Loc != LocStackPair && d98.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d97 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d98}, 2)
					ctx.StabilizeDescForControlFlow(&d97)
					ctx.FreeDesc(&d96)
					d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d99
					ctx.EnsureDesc(&d99)
					if d99.Loc == LocImm {
						d100 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d99.Imm.Int() > 2)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d99.Reg, 2)
						d100 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreater}
						ctx.BindReg(r5, &d100)
					}
					ctx.FreeDesc(&d99)
					d101 = d100
					ctx.SyncDesc(&d101)
					if d101.Loc != LocImm && d101.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d101.Loc == LocImm {
						if d101.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d101.Condition, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d100)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap102 := d1
						snap103 := d2
						snap104 := d3
						snap105 := d4
						snap106 := d5
						snap107 := d6
						snap108 := d7
						snap109 := d10
						snap110 := d11
						snap111 := d12
						snap112 := d14
						snap113 := d15
						snap114 := d16
						snap115 := d17
						snap116 := d18
						snap117 := d20
						snap118 := d21
						snap119 := d40
						snap120 := d41
						snap121 := d42
						snap122 := d43
						snap123 := d44
						snap124 := d45
						snap125 := d70
						snap126 := d96
						snap127 := d97
						snap128 := d98
						snap129 := d99
						snap130 := d100
						snap131 := d101
						alloc132 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc132)
						d1 = snap102
						d2 = snap103
						d3 = snap104
						d4 = snap105
						d5 = snap106
						d6 = snap107
						d7 = snap108
						d10 = snap109
						d11 = snap110
						d12 = snap111
						d14 = snap112
						d15 = snap113
						d16 = snap114
						d17 = snap115
						d18 = snap116
						d20 = snap117
						d21 = snap118
						d40 = snap119
						d41 = snap120
						d42 = snap121
						d43 = snap122
						d44 = snap123
						d45 = snap124
						d70 = snap125
						d96 = snap126
						d97 = snap127
						d98 = snap128
						d99 = snap129
						d100 = snap130
						d101 = snap131
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
					d133 = args[2]
					d133.ID = 0
					ctx.EnsureDesc(&d133)
					d134 = d133
					_ = d134
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d134)
					if d134.Loc == LocImm {
						d135 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d134.Imm.Int())}
					} else if d134.Type == tagInt && d134.Loc == LocRegPair {
						ctx.FreeReg(d134.Reg)
						d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d134.Reg2}
						ctx.BindReg(d134.Reg2, &d135)
						ctx.BindReg(d134.Reg2, &d135)
					} else if d134.Type == tagInt && d134.Loc == LocReg {
						d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d134.Reg}
						ctx.BindReg(d134.Reg, &d135)
						ctx.BindReg(d134.Reg, &d135)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d134)
						d135 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
						d135.Type = tagInt
						ctx.BindReg(d135.Reg, &d135)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d135)
					ctx.EnsureDesc(&d135)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d135)
					ctx.FreeDesc(&d133)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d135)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d135)
					if d4.Loc == LocImm && d135.Loc == LocImm {
						d137 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d135.Imm.Int())}
					} else if d135.Loc == LocImm && d135.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						r6 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(r6, d4.Reg)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d137)
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d135)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d135.Reg}
						ctx.BindReg(d135.Reg, &d137)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d135)
						scratch := ctx.AllocRegExcept(d135.Reg)
						ctx.EmitMovRegReg(scratch, d135.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d137)
					} else if d135.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d135.Imm.Int())
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d137)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d135)
						r7 := ctx.AllocRegExceptOperand(&d135, d4.Reg)
						ctx.EmitMovRegReg(r7, d4.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r7, &d135)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d137)
					}
					if d137.Loc == LocReg && d4.Loc == LocReg && d137.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d137)
					ctx.EmitStoreToStack(d137, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d137)
					ctx.FreeDesc(&d135)
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
					d138 = args[0]
					d138.ID = 0
					d138 = JITPrepareScmerGoArg(ctx, d138)
					if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocRegTriple || d4.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d138)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d1)
					d139 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d138, d4, d1}, 2)
					d139.NoHeapPointer = false
					ctx.BindReg(d139.Reg, &d139)
					ctx.BindReg(d139.Reg2, &d139)
					ctx.FreeDesc(&d138)
					ctx.FreeDesc(&d1)
					ctx.SyncDesc(&d139)
					if d139.Loc == LocRegPair || d139.Loc == LocStackPair || d139.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d139, &result)
						result.Type = d139.Type
					} else {
						switch d139.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d139)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d139)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d139)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d139, &result)
							result.Type = d139.Type
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
					d140 = args[2]
					d140.ID = 0
					ctx.EnsureDesc(&d140)
					d141 = d140
					_ = d141
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d141)
					if d141.Loc == LocImm {
						d142 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d141.Imm.Int())}
					} else if d141.Type == tagInt && d141.Loc == LocRegPair {
						ctx.FreeReg(d141.Reg)
						d142 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d141.Reg2}
						ctx.BindReg(d141.Reg2, &d142)
						ctx.BindReg(d141.Reg2, &d142)
					} else if d141.Type == tagInt && d141.Loc == LocReg {
						d142 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d141.Reg}
						ctx.BindReg(d141.Reg, &d142)
						ctx.BindReg(d141.Reg, &d142)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d141)
						d142 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
						d142.Type = tagInt
						ctx.BindReg(d142.Reg, &d142)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d142)
					ctx.EnsureDesc(&d142)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d142)
					ctx.FreeDesc(&d140)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d142)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d142)
					if d4.Loc == LocImm && d142.Loc == LocImm {
						d144 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d142.Imm.Int())}
					} else if d142.Loc == LocImm && d142.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						r8 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(r8, d4.Reg)
						d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d144)
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d142)
						d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d142.Reg}
						ctx.BindReg(d142.Reg, &d144)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d142)
						scratch := ctx.AllocRegExcept(d142.Reg)
						ctx.EmitMovRegReg(scratch, d142.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d144)
					} else if d142.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d142.Imm.Int())
						d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d144)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d142)
						r9 := ctx.AllocRegExceptOperand(&d142, d4.Reg)
						ctx.EmitMovRegReg(r9, d4.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r9, &d142)
						d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d144)
					}
					if d144.Loc == LocReg && d4.Loc == LocReg && d144.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.FreeDesc(&d142)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d144)
					if d144.Loc == LocImm && d4.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d144.Imm.Int() - d4.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d144.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d144.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d144.Reg)
						}
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitSubInt64(r10, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r10, d4.Reg)
						}
						d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d146)
					}
					r11 := ctx.EmitSliceDataAfterLow(&d97, &d4, 1)
					d147 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d147)
					ctx.BindReg(r11, &d147)
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d147)
					ctx.EnsureDesc(&d147)
					if d147.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d147.Imm.Int()))
					} else {
						r12 = d147.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d146)
					ctx.EnsureDesc(&d146)
					if d146.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d146.Imm.Int()))
					} else {
						r13 = d146.Reg
					}
					ctx.ProtectReg(r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d148 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d148)
					ctx.BindReg(r13, &d148)
					ctx.BindReg(r12, &d148)
					ctx.BindReg(r13, &d148)
					ctx.FreeDesc(&d144)
					ctx.EnsureDesc(&d148)
					d149 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d148}, 2)
					ctx.EmitMovPairToResult(&d149, &result)
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
					ctx.EnsureDesc(&d97)
					if d97.Loc == LocRegPair || d97.Loc == LocRegTriple {
						d150 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d97.Reg2}
						ctx.BindReg(d97.Reg2, &d150)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d150)
					if d150.Loc == LocImm && d4.Loc == LocImm {
						d152 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d150.Imm.Int() - d4.Imm.Int())}
					} else {
						r14 := ctx.AllocReg()
						if d150.Loc == LocImm {
							ctx.EmitMovRegImm64(r14, uint64(d150.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r14, d150.Reg)
						}
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitSubInt64(r14, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r14, d4.Reg)
						}
						d152 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d152)
					}
					r15 := ctx.EmitSliceDataAfterLow(&d97, &d4, 1)
					d153 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
					ctx.BindReg(r15, &d153)
					ctx.BindReg(r15, &d153)
					var r16 Reg
					var r17 Reg
					ctx.SyncDesc(&d153)
					ctx.EnsureDesc(&d153)
					if d153.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d153.Imm.Int()))
					} else {
						r16 = d153.Reg
					}
					ctx.ProtectReg(r16)
					ctx.SyncDesc(&d152)
					ctx.EnsureDesc(&d152)
					if d152.Loc == LocImm {
						r17 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r17, uint64(d152.Imm.Int()))
					} else {
						r17 = d152.Reg
					}
					ctx.ProtectReg(r17)
					ctx.UnprotectReg(r17)
					ctx.UnprotectReg(r16)
					d154 = JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
					ctx.BindReg(r16, &d154)
					ctx.BindReg(r17, &d154)
					ctx.BindReg(r16, &d154)
					ctx.BindReg(r17, &d154)
					ctx.EnsureDesc(&d154)
					d155 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d154}, 2)
					ctx.EmitMovPairToResult(&d155, &result)
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
				var phiBase22 int32
				_ = phiBase22
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
				var d31 JITValueDesc
				_ = d31
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
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
				var d101 JITValueDesc
				_ = d101
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
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
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
				var d301 JITValueDesc
				_ = d301
				var d358 JITValueDesc
				_ = d358
				var d359 JITValueDesc
				_ = d359
				var phiBase361 int32
				_ = phiBase361
				var d362 JITValueDesc
				_ = d362
				var d363 JITValueDesc
				_ = d363
				var d364 JITValueDesc
				_ = d364
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d368 JITValueDesc
				_ = d368
				var d369 JITValueDesc
				_ = d369
				var d370 JITValueDesc
				_ = d370
				var d372 JITValueDesc
				_ = d372
				var d373 JITValueDesc
				_ = d373
				var d442 JITValueDesc
				_ = d442
				var d443 JITValueDesc
				_ = d443
				var d444 JITValueDesc
				_ = d444
				var d516 JITValueDesc
				_ = d516
				var d517 JITValueDesc
				_ = d517
				var d518 JITValueDesc
				_ = d518
				var d519 JITValueDesc
				_ = d519
				var phiBase521 int32
				_ = phiBase521
				var d522 JITValueDesc
				_ = d522
				var d523 JITValueDesc
				_ = d523
				var d524 JITValueDesc
				_ = d524
				var d526 JITValueDesc
				_ = d526
				var d527 JITValueDesc
				_ = d527
				var d528 JITValueDesc
				_ = d528
				var d529 JITValueDesc
				_ = d529
				var d530 JITValueDesc
				_ = d530
				var d532 JITValueDesc
				_ = d532
				var d533 JITValueDesc
				_ = d533
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
				var d627 JITValueDesc
				_ = d627
				var d628 JITValueDesc
				_ = d628
				var d629 JITValueDesc
				_ = d629
				var d630 JITValueDesc
				_ = d630
				var d631 JITValueDesc
				_ = d631
				var d632 JITValueDesc
				_ = d632
				var d633 JITValueDesc
				_ = d633
				var d634 JITValueDesc
				_ = d634
				var d635 JITValueDesc
				_ = d635
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
					inlineHome21 := d20
					inlineHome21.ID = 0
					phiBase22 = ctx.AllocStack(int32(16))
					d23 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase22) + int32(0)}
					_ = d23
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
					d20 = inlineHome21
					d23 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d24 = d20
					d24.ID = 0
					d25 = ctx.EmitGetTagDesc(&d24, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d25)
					inlineHome26 := d25
					inlineHome26.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d25.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d25.Reg)
						ctx.EmitCmpRegImm32(d25.Reg, 19)
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d27)
					}
					ctx.ReclaimUntrackedRegs()
					d28 = d27
					ctx.SyncDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							ctx.MarkLabel(lbl26)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase22)+int32(0))
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl27)
							ctx.EmitJmp(lbl25)
						}
					} else {
						ctx.EmitJump(d28.Condition, lbl26)
						ctx.EmitJmp(lbl27)
						ctx.FreeDesc(&d27)
						ctx.MarkLabel(lbl26)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase22)+int32(0))
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl27)
						ctx.EmitJmp(lbl25)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					d20 = inlineHome21
					d25 = inlineHome26
					d23 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d23)
					}
					ctx.EmitJmp(lbl21)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl25)
					ctx.ResolveFixups()
					d20 = inlineHome21
					d25 = inlineHome26
					d23 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d25.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d25.Reg)
						ctx.EmitCmpRegImm32(d25.Reg, 20)
						d29 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d29)
					}
					ctx.ReclaimUntrackedRegs()
					d30 = d29
					ctx.SyncDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase22)+int32(0))
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitJump(d30.Condition, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.FreeDesc(&d29)
						ctx.MarkLabel(lbl28)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase22)+int32(0))
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl23)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					d20 = inlineHome21
					d25 = inlineHome26
					d23 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d25.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d25.Reg, 26)
						r3 := ctx.AllocRegExcept(d25.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d31)
					}
					ctx.EnsureDesc(&d31)
					ctx.EmitStoreToStack(d31, int32(phiBase22)+int32(0))
					ctx.StabilizeDescForControlFlow(&d31)
					inlineHome32 := d31
					inlineHome32.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl24)
					ctx.MarkLabel(lbl21)
					d33 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d33)
					ctx.BindReg(r1, &d33)
					ctx.FreeDesc(&d19)
					d34 = d33
					ctx.EnsureDesc(&d34)
					if d34.Loc != LocImm && d34.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d34.Loc == LocImm {
						if d34.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d34.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap35 := d1
						snap36 := d2
						snap37 := d3
						snap38 := d4
						snap39 := d5
						snap40 := d6
						snap41 := d7
						snap42 := d8
						snap43 := d18
						snap44 := d19
						snap45 := d20
						snap46 := d23
						snap47 := d24
						snap48 := d25
						snap49 := d27
						snap50 := d28
						snap51 := d29
						snap52 := d30
						snap53 := d31
						snap54 := d33
						snap55 := d34
						alloc56 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc56)
						d1 = snap35
						d2 = snap36
						d3 = snap37
						d4 = snap38
						d5 = snap39
						d6 = snap40
						d7 = snap41
						d8 = snap42
						d18 = snap43
						d19 = snap44
						d20 = snap45
						d23 = snap46
						d24 = snap47
						d25 = snap48
						d27 = snap49
						d28 = snap50
						d29 = snap51
						d30 = snap52
						d31 = snap53
						d33 = snap54
						d34 = snap55
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d33)
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
					d57 = args[0]
					d57.ID = 0
					d57 = JITPrepareScmerGoArg(ctx, d57)
					ctx.SyncDesc(&d57)
					d58 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d57}, 1)
					d58.NoHeapPointer = true
					ctx.BindReg(d58.Reg, &d58)
					ctx.StabilizeDescForControlFlow(&d58)
					ctx.FreeDesc(&d57)
					ctx.SyncDesc(&d58)
					if d58.Loc == LocReg || d58.Loc == LocFPReg {
						ctx.ProtectReg(d58.Reg)
					} else if d58.Loc == LocRegPair {
						ctx.ProtectReg(d58.Reg)
						ctx.ProtectReg(d58.Reg2)
					}
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}, int32(bbs[4].PhiBase)+int32(0))
					d59 = d58
					if d59.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(bbs[4].PhiBase)+int32(16))
					if d58.Loc == LocReg || d58.Loc == LocFPReg {
						ctx.UnprotectReg(d58.Reg)
					} else if d58.Loc == LocRegPair {
						ctx.UnprotectReg(d58.Reg)
						ctx.UnprotectReg(d58.Reg2)
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
					d60 = args[1]
					d60.ID = 0
					ctx.EnsureDesc(&d60)
					d61 = d60
					_ = d61
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl30 := ctx.ReserveLabel()
					_ = lbl30
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl30)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d61)
					if d61.Loc == LocImm {
						d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d61.Imm.Int())}
					} else if d61.Type == tagInt && d61.Loc == LocRegPair {
						ctx.FreeReg(d61.Reg)
						d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d61.Reg2}
						ctx.BindReg(d61.Reg2, &d62)
						ctx.BindReg(d61.Reg2, &d62)
					} else if d61.Type == tagInt && d61.Loc == LocReg {
						d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d61.Reg}
						ctx.BindReg(d61.Reg, &d62)
						ctx.BindReg(d61.Reg, &d62)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d61)
						d62 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
						d62.Type = tagInt
						ctx.BindReg(d62.Reg, &d62)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d62)
					ctx.EnsureDesc(&d62)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d62)
					ctx.FreeDesc(&d60)
					ctx.EnsureDesc(&d62)
					ctx.EnsureDesc(&d62)
					if d62.Loc == LocImm {
						d64 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d62.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d62.Reg)
						ctx.EmitMovRegReg(scratch, d62.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d64 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d64)
					}
					if d64.Loc == LocReg && d62.Loc == LocReg && d64.Reg == d62.Reg {
						ctx.TransferReg(d62.Reg)
						d62.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d64)
					ctx.FreeDesc(&d62)
					ctx.EnsureDesc(&d64)
					if d64.Loc == LocImm {
						d65 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d64.Imm.Int() < 0)}
					} else {
						r4 := ctx.AllocRegExcept(d64.Reg)
						ctx.EmitCmpRegImm32(d64.Reg, 0)
						d65 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d65)
					}
					d66 = d65
					ctx.SyncDesc(&d66)
					if d66.Loc != LocImm && d66.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d66.Loc == LocImm {
						if d66.Imm.Bool() {
							return bbs[6].Render()
						}
						ctx.SyncDesc(&d64)
						if d64.Loc == LocReg || d64.Loc == LocFPReg {
							ctx.ProtectReg(d64.Reg)
						} else if d64.Loc == LocRegPair {
							ctx.ProtectReg(d64.Reg)
							ctx.ProtectReg(d64.Reg2)
						}
						d67 = d64
						if d67.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d67)
						ctx.EmitStoreToStack(d67, int32(bbs[7].PhiBase)+int32(0))
						if d64.Loc == LocReg || d64.Loc == LocFPReg {
							ctx.UnprotectReg(d64.Reg)
						} else if d64.Loc == LocRegPair {
							ctx.UnprotectReg(d64.Reg)
							ctx.UnprotectReg(d64.Reg2)
						}
						return bbs[7].Render()
					}
					lbl31 := ctx.ReserveLabel()
					ctx.EmitJump(d66.Condition, lbl7)
					ctx.EmitJmp(lbl31)
					ctx.FreeDesc(&d65)
					snap68 := d1
					snap69 := d2
					snap70 := d3
					snap71 := d4
					snap72 := d5
					snap73 := d6
					snap74 := d7
					snap75 := d8
					snap76 := d18
					snap77 := d19
					snap78 := d20
					snap79 := d23
					snap80 := d24
					snap81 := d25
					snap82 := d27
					snap83 := d28
					snap84 := d29
					snap85 := d30
					snap86 := d31
					snap87 := d33
					snap88 := d34
					snap89 := d57
					snap90 := d58
					snap91 := d59
					snap92 := d60
					snap93 := d61
					snap94 := d62
					snap95 := d63
					snap96 := d64
					snap97 := d65
					snap98 := d66
					snap99 := d67
					alloc100 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl31)
					ctx.SyncDesc(&d64)
					if d64.Loc == LocReg || d64.Loc == LocFPReg {
						ctx.ProtectReg(d64.Reg)
					} else if d64.Loc == LocRegPair {
						ctx.ProtectReg(d64.Reg)
						ctx.ProtectReg(d64.Reg2)
					}
					d101 = d64
					if d101.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d101)
					ctx.EmitStoreToStack(d101, int32(bbs[7].PhiBase)+int32(0))
					if d64.Loc == LocReg || d64.Loc == LocFPReg {
						ctx.UnprotectReg(d64.Reg)
					} else if d64.Loc == LocRegPair {
						ctx.UnprotectReg(d64.Reg)
						ctx.UnprotectReg(d64.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc100)
					d1 = snap68
					d2 = snap69
					d3 = snap70
					d4 = snap71
					d5 = snap72
					d6 = snap73
					d7 = snap74
					d8 = snap75
					d18 = snap76
					d19 = snap77
					d20 = snap78
					d23 = snap79
					d24 = snap80
					d25 = snap81
					d27 = snap82
					d28 = snap83
					d29 = snap84
					d30 = snap85
					d31 = snap86
					d33 = snap87
					d34 = snap88
					d57 = snap89
					d58 = snap90
					d59 = snap91
					d60 = snap92
					d61 = snap93
					d62 = snap94
					d63 = snap95
					d64 = snap96
					d65 = snap97
					d66 = snap98
					d67 = snap99
					if !bbs[7].Rendered {
						snap102 := d1
						snap103 := d2
						snap104 := d3
						snap105 := d4
						snap106 := d5
						snap107 := d6
						snap108 := d7
						snap109 := d8
						snap110 := d18
						snap111 := d19
						snap112 := d20
						snap113 := d23
						snap114 := d24
						snap115 := d25
						snap116 := d27
						snap117 := d28
						snap118 := d29
						snap119 := d30
						snap120 := d31
						snap121 := d33
						snap122 := d34
						snap123 := d57
						snap124 := d58
						snap125 := d59
						snap126 := d60
						snap127 := d61
						snap128 := d62
						snap129 := d63
						snap130 := d64
						snap131 := d65
						snap132 := d66
						snap133 := d67
						snap134 := d101
						alloc135 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc135)
						d1 = snap102
						d2 = snap103
						d3 = snap104
						d4 = snap105
						d5 = snap106
						d6 = snap107
						d7 = snap108
						d8 = snap109
						d18 = snap110
						d19 = snap111
						d20 = snap112
						d23 = snap113
						d24 = snap114
						d25 = snap115
						d27 = snap116
						d28 = snap117
						d29 = snap118
						d30 = snap119
						d31 = snap120
						d33 = snap121
						d34 = snap122
						d57 = snap123
						d58 = snap124
						d59 = snap125
						d60 = snap126
						d61 = snap127
						d62 = snap128
						d63 = snap129
						d64 = snap130
						d65 = snap131
						d66 = snap132
						d67 = snap133
						d101 = snap134
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
					d136 = args[0]
					d136.ID = 0
					d138 = d136
					ctx.SyncDesc(&d138)
					if d138.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d138.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d138.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d138 = tmpScalar
					}
					d138 = JITPrepareScmerGoArg(ctx, d138)
					if d138.Loc != LocRegPair && d138.Loc != LocStackPair && d138.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d137 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d138}, 2)
					ctx.StabilizeDescForControlFlow(&d137)
					ctx.FreeDesc(&d136)
					if d137.SliceSizeKnown {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d137.KnownSliceLen))}
					} else if d137.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d137.Imm.String())))}
					} else if d137.Loc == LocStackTriple {
						d139 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d137.StackOff + 8, NoHeapPointer: true}
					} else if d137.Loc == LocStackPair {
						d139 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d137.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d137)
						if d137.Loc == LocRegPair || d137.Loc == LocRegTriple {
							d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d137.Reg2, ID: 0}
						} else if d137.Loc == LocReg {
							d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d137.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d139
					ctx.StabilizeDescForControlFlow(&d139)
					ctx.SyncDesc(&d137)
					if d137.Loc == LocReg || d137.Loc == LocFPReg {
						ctx.ProtectReg(d137.Reg)
					} else if d137.Loc == LocRegPair {
						ctx.ProtectReg(d137.Reg)
						ctx.ProtectReg(d137.Reg2)
					}
					ctx.SyncDesc(&d139)
					if d139.Loc == LocReg || d139.Loc == LocFPReg {
						ctx.ProtectReg(d139.Reg)
					} else if d139.Loc == LocRegPair {
						ctx.ProtectReg(d139.Reg)
						ctx.ProtectReg(d139.Reg2)
					}
					d140 = d137
					if d140.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d140)
					if d140.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d140, int32(bbs[4].PhiBase)+int32(0), 2)
					} else if d140.Loc == LocInputPair {
						ctx.EnsureDesc(&d140)
						ctx.EmitStoreScmerToStack(d140, int32(bbs[4].PhiBase)+int32(0))
					} else if d140.Loc == LocRegPair || d140.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d140, int32(bbs[4].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d140)
						ctx.EmitStoreToStack(d140, int32(bbs[4].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
					}
					d141 = d139
					if d141.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d141)
					ctx.EmitStoreToStack(d141, int32(bbs[4].PhiBase)+int32(16))
					if d137.Loc == LocReg || d137.Loc == LocFPReg {
						ctx.UnprotectReg(d137.Reg)
					} else if d137.Loc == LocRegPair {
						ctx.UnprotectReg(d137.Reg)
						ctx.UnprotectReg(d137.Reg2)
					}
					if d139.Loc == LocReg || d139.Loc == LocFPReg {
						ctx.UnprotectReg(d139.Reg)
					} else if d139.Loc == LocRegPair {
						ctx.UnprotectReg(d139.Reg)
						ctx.UnprotectReg(d139.Reg2)
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
						d142 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() >= d2.Imm.Int())}
					} else if d2.Loc == LocImm {
						r5 := ctx.AllocRegExcept(d3.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d142 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r5, &d142)
					} else if d3.Loc == LocImm {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
						d142 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r6, &d142)
					} else {
						r7 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d2.Reg)
						d142 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r7, &d142)
					}
					d143 = d142
					ctx.SyncDesc(&d143)
					if d143.Loc != LocImm && d143.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d143.Loc == LocImm {
						if d143.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitJump(d143.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d142)
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap144 := d1
						snap145 := d2
						snap146 := d3
						snap147 := d4
						snap148 := d5
						snap149 := d6
						snap150 := d7
						snap151 := d8
						snap152 := d18
						snap153 := d19
						snap154 := d20
						snap155 := d23
						snap156 := d24
						snap157 := d25
						snap158 := d27
						snap159 := d28
						snap160 := d29
						snap161 := d30
						snap162 := d31
						snap163 := d33
						snap164 := d34
						snap165 := d57
						snap166 := d58
						snap167 := d59
						snap168 := d60
						snap169 := d61
						snap170 := d62
						snap171 := d63
						snap172 := d64
						snap173 := d65
						snap174 := d66
						snap175 := d67
						snap176 := d101
						snap177 := d136
						snap178 := d137
						snap179 := d138
						snap180 := d139
						snap181 := d140
						snap182 := d141
						snap183 := d142
						snap184 := d143
						alloc185 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc185)
						d1 = snap144
						d2 = snap145
						d3 = snap146
						d4 = snap147
						d5 = snap148
						d6 = snap149
						d7 = snap150
						d8 = snap151
						d18 = snap152
						d19 = snap153
						d20 = snap154
						d23 = snap155
						d24 = snap156
						d25 = snap157
						d27 = snap158
						d28 = snap159
						d29 = snap160
						d30 = snap161
						d31 = snap162
						d33 = snap163
						d34 = snap164
						d57 = snap165
						d58 = snap166
						d59 = snap167
						d60 = snap168
						d61 = snap169
						d62 = snap170
						d63 = snap171
						d64 = snap172
						d65 = snap173
						d66 = snap174
						d67 = snap175
						d101 = snap176
						d136 = snap177
						d137 = snap178
						d138 = snap179
						d139 = snap180
						d140 = snap181
						d141 = snap182
						d142 = snap183
						d143 = snap184
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
					d186 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d187 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d186}, 2)
					ctx.EmitMovPairToResult(&d187, &result)
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
					d188 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d188
					ctx.EnsureDesc(&d188)
					if d188.Loc == LocImm {
						d189 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d188.Imm.Int() > 2)}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d188.Reg, 2)
						d189 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedGreater}
						ctx.BindReg(r8, &d189)
					}
					ctx.FreeDesc(&d188)
					d190 = d189
					ctx.SyncDesc(&d190)
					if d190.Loc != LocImm && d190.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d190.Loc == LocImm {
						if d190.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitJump(d190.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d189)
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap191 := d1
						snap192 := d2
						snap193 := d3
						snap194 := d4
						snap195 := d5
						snap196 := d6
						snap197 := d7
						snap198 := d8
						snap199 := d18
						snap200 := d19
						snap201 := d20
						snap202 := d23
						snap203 := d24
						snap204 := d25
						snap205 := d27
						snap206 := d28
						snap207 := d29
						snap208 := d30
						snap209 := d31
						snap210 := d33
						snap211 := d34
						snap212 := d57
						snap213 := d58
						snap214 := d59
						snap215 := d60
						snap216 := d61
						snap217 := d62
						snap218 := d63
						snap219 := d64
						snap220 := d65
						snap221 := d66
						snap222 := d67
						snap223 := d101
						snap224 := d136
						snap225 := d137
						snap226 := d138
						snap227 := d139
						snap228 := d140
						snap229 := d141
						snap230 := d142
						snap231 := d143
						snap232 := d186
						snap233 := d187
						snap234 := d188
						snap235 := d189
						snap236 := d190
						alloc237 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc237)
						d1 = snap191
						d2 = snap192
						d3 = snap193
						d4 = snap194
						d5 = snap195
						d6 = snap196
						d7 = snap197
						d8 = snap198
						d18 = snap199
						d19 = snap200
						d20 = snap201
						d23 = snap202
						d24 = snap203
						d25 = snap204
						d27 = snap205
						d28 = snap206
						d29 = snap207
						d30 = snap208
						d31 = snap209
						d33 = snap210
						d34 = snap211
						d57 = snap212
						d58 = snap213
						d59 = snap214
						d60 = snap215
						d61 = snap216
						d62 = snap217
						d63 = snap218
						d64 = snap219
						d65 = snap220
						d66 = snap221
						d67 = snap222
						d101 = snap223
						d136 = snap224
						d137 = snap225
						d138 = snap226
						d139 = snap227
						d140 = snap228
						d141 = snap229
						d142 = snap230
						d143 = snap231
						d186 = snap232
						d187 = snap233
						d188 = snap234
						d189 = snap235
						d190 = snap236
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
					d238 = args[2]
					d238.ID = 0
					ctx.EnsureDesc(&d238)
					d239 = d238
					_ = d239
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl32 := ctx.ReserveLabel()
					_ = lbl32
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl32)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d239)
					if d239.Loc == LocImm {
						d240 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d239.Imm.Int())}
					} else if d239.Type == tagInt && d239.Loc == LocRegPair {
						ctx.FreeReg(d239.Reg)
						d240 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d239.Reg2}
						ctx.BindReg(d239.Reg2, &d240)
						ctx.BindReg(d239.Reg2, &d240)
					} else if d239.Type == tagInt && d239.Loc == LocReg {
						d240 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d239.Reg}
						ctx.BindReg(d239.Reg, &d240)
						ctx.BindReg(d239.Reg, &d240)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d239)
						d240 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
						d240.Type = tagInt
						ctx.BindReg(d240.Reg, &d240)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d240)
					ctx.EnsureDesc(&d240)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d240)
					ctx.StabilizeDescForControlFlow(&d240)
					ctx.FreeDesc(&d238)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d240)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d240)
					if d3.Loc == LocImm && d240.Loc == LocImm {
						d242 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d240.Imm.Int())}
					} else if d240.Loc == LocImm && d240.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r9 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r9, d3.Reg)
						d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d242)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d240)
						d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d240.Reg}
						ctx.BindReg(d240.Reg, &d242)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d240)
						scratch := ctx.AllocRegExcept(d240.Reg)
						ctx.EmitMovRegReg(scratch, d240.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d242)
					} else if d240.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d240.Imm.Int())
						d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d242)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d240)
						r10 := ctx.AllocRegExceptOperand(&d240, d3.Reg)
						ctx.EmitMovRegReg(r10, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r10, &d240)
						d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d242)
					}
					if d242.Loc == LocReg && d3.Loc == LocReg && d242.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d242)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDescsTogether(&d242, &d2)
					if d242.Loc == LocImm && d2.Loc == LocImm {
						d243 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d242.Imm.Int() > d2.Imm.Int())}
					} else if d2.Loc == LocImm {
						r11 := ctx.AllocReg()
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d242.Reg, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
							ctx.EmitCmpInt64(d242.Reg, ctx.ScratchReg)
						}
						d243 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedGreater}
						ctx.BindReg(r11, &d243)
					} else if d242.Loc == LocImm {
						r12 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d242.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
						d243 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedGreater}
						ctx.BindReg(r12, &d243)
					} else {
						r13 := ctx.AllocReg()
						ctx.EmitCmpInt64(d242.Reg, d2.Reg)
						d243 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedGreater}
						ctx.BindReg(r13, &d243)
					}
					ctx.FreeDesc(&d242)
					d244 = d243
					ctx.SyncDesc(&d244)
					if d244.Loc != LocImm && d244.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d244.Loc == LocImm {
						if d244.Imm.Bool() {
							return bbs[12].Render()
						}
						ctx.SyncDesc(&d240)
						if d240.Loc == LocReg || d240.Loc == LocFPReg {
							ctx.ProtectReg(d240.Reg)
						} else if d240.Loc == LocRegPair {
							ctx.ProtectReg(d240.Reg)
							ctx.ProtectReg(d240.Reg2)
						}
						d245 = d240
						if d245.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d245)
						ctx.EmitStoreToStack(d245, int32(bbs[13].PhiBase)+int32(0))
						if d240.Loc == LocReg || d240.Loc == LocFPReg {
							ctx.UnprotectReg(d240.Reg)
						} else if d240.Loc == LocRegPair {
							ctx.UnprotectReg(d240.Reg)
							ctx.UnprotectReg(d240.Reg2)
						}
						return bbs[13].Render()
					}
					lbl33 := ctx.ReserveLabel()
					ctx.EmitJump(d244.Condition, lbl13)
					ctx.EmitJmp(lbl33)
					ctx.FreeDesc(&d243)
					snap246 := d1
					snap247 := d2
					snap248 := d3
					snap249 := d4
					snap250 := d5
					snap251 := d6
					snap252 := d7
					snap253 := d8
					snap254 := d18
					snap255 := d19
					snap256 := d20
					snap257 := d23
					snap258 := d24
					snap259 := d25
					snap260 := d27
					snap261 := d28
					snap262 := d29
					snap263 := d30
					snap264 := d31
					snap265 := d33
					snap266 := d34
					snap267 := d57
					snap268 := d58
					snap269 := d59
					snap270 := d60
					snap271 := d61
					snap272 := d62
					snap273 := d63
					snap274 := d64
					snap275 := d65
					snap276 := d66
					snap277 := d67
					snap278 := d101
					snap279 := d136
					snap280 := d137
					snap281 := d138
					snap282 := d139
					snap283 := d140
					snap284 := d141
					snap285 := d142
					snap286 := d143
					snap287 := d186
					snap288 := d187
					snap289 := d188
					snap290 := d189
					snap291 := d190
					snap292 := d238
					snap293 := d239
					snap294 := d240
					snap295 := d241
					snap296 := d242
					snap297 := d243
					snap298 := d244
					snap299 := d245
					alloc300 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl33)
					ctx.SyncDesc(&d240)
					if d240.Loc == LocReg || d240.Loc == LocFPReg {
						ctx.ProtectReg(d240.Reg)
					} else if d240.Loc == LocRegPair {
						ctx.ProtectReg(d240.Reg)
						ctx.ProtectReg(d240.Reg2)
					}
					d301 = d240
					if d301.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d301)
					ctx.EmitStoreToStack(d301, int32(bbs[13].PhiBase)+int32(0))
					if d240.Loc == LocReg || d240.Loc == LocFPReg {
						ctx.UnprotectReg(d240.Reg)
					} else if d240.Loc == LocRegPair {
						ctx.UnprotectReg(d240.Reg)
						ctx.UnprotectReg(d240.Reg2)
					}
					ctx.EmitJmp(lbl14)
					ctx.RestoreAllocState(alloc300)
					d1 = snap246
					d2 = snap247
					d3 = snap248
					d4 = snap249
					d5 = snap250
					d6 = snap251
					d7 = snap252
					d8 = snap253
					d18 = snap254
					d19 = snap255
					d20 = snap256
					d23 = snap257
					d24 = snap258
					d25 = snap259
					d27 = snap260
					d28 = snap261
					d29 = snap262
					d30 = snap263
					d31 = snap264
					d33 = snap265
					d34 = snap266
					d57 = snap267
					d58 = snap268
					d59 = snap269
					d60 = snap270
					d61 = snap271
					d62 = snap272
					d63 = snap273
					d64 = snap274
					d65 = snap275
					d66 = snap276
					d67 = snap277
					d101 = snap278
					d136 = snap279
					d137 = snap280
					d138 = snap281
					d139 = snap282
					d140 = snap283
					d141 = snap284
					d142 = snap285
					d143 = snap286
					d186 = snap287
					d187 = snap288
					d188 = snap289
					d189 = snap290
					d190 = snap291
					d238 = snap292
					d239 = snap293
					d240 = snap294
					d241 = snap295
					d242 = snap296
					d243 = snap297
					d244 = snap298
					d245 = snap299
					if !bbs[13].Rendered {
						snap302 := d1
						snap303 := d2
						snap304 := d3
						snap305 := d4
						snap306 := d5
						snap307 := d6
						snap308 := d7
						snap309 := d8
						snap310 := d18
						snap311 := d19
						snap312 := d20
						snap313 := d23
						snap314 := d24
						snap315 := d25
						snap316 := d27
						snap317 := d28
						snap318 := d29
						snap319 := d30
						snap320 := d31
						snap321 := d33
						snap322 := d34
						snap323 := d57
						snap324 := d58
						snap325 := d59
						snap326 := d60
						snap327 := d61
						snap328 := d62
						snap329 := d63
						snap330 := d64
						snap331 := d65
						snap332 := d66
						snap333 := d67
						snap334 := d101
						snap335 := d136
						snap336 := d137
						snap337 := d138
						snap338 := d139
						snap339 := d140
						snap340 := d141
						snap341 := d142
						snap342 := d143
						snap343 := d186
						snap344 := d187
						snap345 := d188
						snap346 := d189
						snap347 := d190
						snap348 := d238
						snap349 := d239
						snap350 := d240
						snap351 := d241
						snap352 := d242
						snap353 := d243
						snap354 := d244
						snap355 := d245
						snap356 := d301
						alloc357 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc357)
						d1 = snap302
						d2 = snap303
						d3 = snap304
						d4 = snap305
						d5 = snap306
						d6 = snap307
						d7 = snap308
						d8 = snap309
						d18 = snap310
						d19 = snap311
						d20 = snap312
						d23 = snap313
						d24 = snap314
						d25 = snap315
						d27 = snap316
						d28 = snap317
						d29 = snap318
						d30 = snap319
						d31 = snap320
						d33 = snap321
						d34 = snap322
						d57 = snap323
						d58 = snap324
						d59 = snap325
						d60 = snap326
						d61 = snap327
						d62 = snap328
						d63 = snap329
						d64 = snap330
						d65 = snap331
						d66 = snap332
						d67 = snap333
						d101 = snap334
						d136 = snap335
						d137 = snap336
						d138 = snap337
						d139 = snap338
						d140 = snap339
						d141 = snap340
						d142 = snap341
						d143 = snap342
						d186 = snap343
						d187 = snap344
						d188 = snap345
						d189 = snap346
						d190 = snap347
						d238 = snap348
						d239 = snap349
						d240 = snap350
						d241 = snap351
						d242 = snap352
						d243 = snap353
						d244 = snap354
						d245 = snap355
						d301 = snap356
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
					d358 = args[0]
					d358.ID = 0
					ctx.EnsureDesc(&d358)
					d359 = d358
					_ = d359
					ctx.StabilizeDescForControlFlow(&d359)
					inlineHome360 := d359
					inlineHome360.ID = 0
					phiBase361 = ctx.AllocStack(int32(16))
					d362 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase361) + int32(0)}
					_ = d362
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
					d359 = inlineHome360
					d362 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase361) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d363 = d359
					d363.ID = 0
					d364 = ctx.EmitGetTagDesc(&d363, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d364)
					inlineHome365 := d364
					inlineHome365.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d364)
					if d364.Loc == LocImm {
						d366 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d364.Imm.Int()) == uint64(0x13))}
					} else {
						r14 := ctx.AllocRegExcept(d364.Reg)
						ctx.EmitCmpRegImm32(d364.Reg, 19)
						d366 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondEqual}
						ctx.BindReg(r14, &d366)
					}
					ctx.ReclaimUntrackedRegs()
					d367 = d366
					ctx.SyncDesc(&d367)
					if d367.Loc != LocImm && d367.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl39 := ctx.ReserveLabel()
					lbl40 := ctx.ReserveLabel()
					if d367.Loc == LocImm {
						if d367.Imm.Bool() {
							ctx.MarkLabel(lbl39)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase361)+int32(0))
							ctx.EmitJmp(lbl37)
						} else {
							ctx.MarkLabel(lbl40)
							ctx.EmitJmp(lbl38)
						}
					} else {
						ctx.EmitJump(d367.Condition, lbl39)
						ctx.EmitJmp(lbl40)
						ctx.FreeDesc(&d366)
						ctx.MarkLabel(lbl39)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase361)+int32(0))
						ctx.EmitJmp(lbl37)
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl38)
					}
					bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl37)
					ctx.ResolveFixups()
					d359 = inlineHome360
					d364 = inlineHome365
					d362 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase361) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r15 := ctx.AllocReg()
					ctx.EnsureDesc(&d362)
					ctx.EnsureDesc(&d362)
					if d362.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r15, d362)
					}
					ctx.EmitJmp(lbl34)
					bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl38)
					ctx.ResolveFixups()
					d359 = inlineHome360
					d364 = inlineHome365
					d362 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase361) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d364)
					if d364.Loc == LocImm {
						d368 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d364.Imm.Int()) == uint64(0x14))}
					} else {
						r16 := ctx.AllocRegExcept(d364.Reg)
						ctx.EmitCmpRegImm32(d364.Reg, 20)
						d368 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondEqual}
						ctx.BindReg(r16, &d368)
					}
					ctx.ReclaimUntrackedRegs()
					d369 = d368
					ctx.SyncDesc(&d369)
					if d369.Loc != LocImm && d369.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl41 := ctx.ReserveLabel()
					lbl42 := ctx.ReserveLabel()
					if d369.Loc == LocImm {
						if d369.Imm.Bool() {
							ctx.MarkLabel(lbl41)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase361)+int32(0))
							ctx.EmitJmp(lbl37)
						} else {
							ctx.MarkLabel(lbl42)
							ctx.EmitJmp(lbl36)
						}
					} else {
						ctx.EmitJump(d369.Condition, lbl41)
						ctx.EmitJmp(lbl42)
						ctx.FreeDesc(&d368)
						ctx.MarkLabel(lbl41)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase361)+int32(0))
						ctx.EmitJmp(lbl37)
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl36)
					}
					bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl36)
					ctx.ResolveFixups()
					d359 = inlineHome360
					d364 = inlineHome365
					d362 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase361) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d364)
					if d364.Loc == LocImm {
						d370 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d364.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d364.Reg, 26)
						r17 := ctx.AllocRegExcept(d364.Reg)
						ctx.EmitSetcc(r17, CondEqual)
						d370 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
						ctx.BindReg(r17, &d370)
					}
					ctx.EnsureDesc(&d370)
					ctx.EmitStoreToStack(d370, int32(phiBase361)+int32(0))
					ctx.StabilizeDescForControlFlow(&d370)
					inlineHome371 := d370
					inlineHome371.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl37)
					ctx.MarkLabel(lbl34)
					d372 = JITValueDesc{Loc: LocReg, Reg: r15}
					ctx.BindReg(r15, &d372)
					ctx.BindReg(r15, &d372)
					ctx.FreeDesc(&d358)
					d373 = d372
					ctx.EnsureDesc(&d373)
					if d373.Loc != LocImm && d373.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d373.Loc == LocImm {
						if d373.Imm.Bool() {
							return bbs[18].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d373.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap374 := d1
						snap375 := d2
						snap376 := d3
						snap377 := d4
						snap378 := d5
						snap379 := d6
						snap380 := d7
						snap381 := d8
						snap382 := d18
						snap383 := d19
						snap384 := d20
						snap385 := d23
						snap386 := d24
						snap387 := d25
						snap388 := d27
						snap389 := d28
						snap390 := d29
						snap391 := d30
						snap392 := d31
						snap393 := d33
						snap394 := d34
						snap395 := d57
						snap396 := d58
						snap397 := d59
						snap398 := d60
						snap399 := d61
						snap400 := d62
						snap401 := d63
						snap402 := d64
						snap403 := d65
						snap404 := d66
						snap405 := d67
						snap406 := d101
						snap407 := d136
						snap408 := d137
						snap409 := d138
						snap410 := d139
						snap411 := d140
						snap412 := d141
						snap413 := d142
						snap414 := d143
						snap415 := d186
						snap416 := d187
						snap417 := d188
						snap418 := d189
						snap419 := d190
						snap420 := d238
						snap421 := d239
						snap422 := d240
						snap423 := d241
						snap424 := d242
						snap425 := d243
						snap426 := d244
						snap427 := d245
						snap428 := d301
						snap429 := d358
						snap430 := d359
						snap431 := d362
						snap432 := d363
						snap433 := d364
						snap434 := d366
						snap435 := d367
						snap436 := d368
						snap437 := d369
						snap438 := d370
						snap439 := d372
						snap440 := d373
						alloc441 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc441)
						d1 = snap374
						d2 = snap375
						d3 = snap376
						d4 = snap377
						d5 = snap378
						d6 = snap379
						d7 = snap380
						d8 = snap381
						d18 = snap382
						d19 = snap383
						d20 = snap384
						d23 = snap385
						d24 = snap386
						d25 = snap387
						d27 = snap388
						d28 = snap389
						d29 = snap390
						d30 = snap391
						d31 = snap392
						d33 = snap393
						d34 = snap394
						d57 = snap395
						d58 = snap396
						d59 = snap397
						d60 = snap398
						d61 = snap399
						d62 = snap400
						d63 = snap401
						d64 = snap402
						d65 = snap403
						d66 = snap404
						d67 = snap405
						d101 = snap406
						d136 = snap407
						d137 = snap408
						d138 = snap409
						d139 = snap410
						d140 = snap411
						d141 = snap412
						d142 = snap413
						d143 = snap414
						d186 = snap415
						d187 = snap416
						d188 = snap417
						d189 = snap418
						d190 = snap419
						d238 = snap420
						d239 = snap421
						d240 = snap422
						d241 = snap423
						d242 = snap424
						d243 = snap425
						d244 = snap426
						d245 = snap427
						d301 = snap428
						d358 = snap429
						d359 = snap430
						d362 = snap431
						d363 = snap432
						d364 = snap433
						d366 = snap434
						d367 = snap435
						d368 = snap436
						d369 = snap437
						d370 = snap438
						d372 = snap439
						d373 = snap440
					}
					if !bbs[18].Rendered {
						return bbs[18].Render()
					}
					return result
					ctx.FreeDesc(&d372)
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
						d442 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() - d3.Imm.Int())}
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d2)
						r18 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(r18, d2.Reg)
						d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d442)
					} else if d2.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d3)
						d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d442)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d2)
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d3.Imm.Int())
						d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d442)
					} else {
						ctx.EnsureDesc(&d2)
						ctx.SyncDesc(&d3)
						r19 := ctx.AllocRegExceptOperand(&d3, d2.Reg)
						ctx.EmitMovRegReg(r19, d2.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r19, &d3)
						d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d442)
					}
					if d442.Loc == LocReg && d2.Loc == LocReg && d442.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d442)
					ctx.EmitStoreToStack(d442, int32(bbs[13].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d442)
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
						d443 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < 0)}
					} else {
						r20 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpRegImm32(d4.Reg, 0)
						d443 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d443)
					}
					d444 = d443
					ctx.SyncDesc(&d444)
					if d444.Loc != LocImm && d444.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d444.Loc == LocImm {
						if d444.Imm.Bool() {
							return bbs[14].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitJump(d444.Condition, lbl15)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FreeDesc(&d443)
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap445 := d1
						snap446 := d2
						snap447 := d3
						snap448 := d4
						snap449 := d5
						snap450 := d6
						snap451 := d7
						snap452 := d8
						snap453 := d18
						snap454 := d19
						snap455 := d20
						snap456 := d23
						snap457 := d24
						snap458 := d25
						snap459 := d27
						snap460 := d28
						snap461 := d29
						snap462 := d30
						snap463 := d31
						snap464 := d33
						snap465 := d34
						snap466 := d57
						snap467 := d58
						snap468 := d59
						snap469 := d60
						snap470 := d61
						snap471 := d62
						snap472 := d63
						snap473 := d64
						snap474 := d65
						snap475 := d66
						snap476 := d67
						snap477 := d101
						snap478 := d136
						snap479 := d137
						snap480 := d138
						snap481 := d139
						snap482 := d140
						snap483 := d141
						snap484 := d142
						snap485 := d143
						snap486 := d186
						snap487 := d187
						snap488 := d188
						snap489 := d189
						snap490 := d190
						snap491 := d238
						snap492 := d239
						snap493 := d240
						snap494 := d241
						snap495 := d242
						snap496 := d243
						snap497 := d244
						snap498 := d245
						snap499 := d301
						snap500 := d358
						snap501 := d359
						snap502 := d362
						snap503 := d363
						snap504 := d364
						snap505 := d366
						snap506 := d367
						snap507 := d368
						snap508 := d369
						snap509 := d370
						snap510 := d372
						snap511 := d373
						snap512 := d442
						snap513 := d443
						snap514 := d444
						alloc515 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc515)
						d1 = snap445
						d2 = snap446
						d3 = snap447
						d4 = snap448
						d5 = snap449
						d6 = snap450
						d7 = snap451
						d8 = snap452
						d18 = snap453
						d19 = snap454
						d20 = snap455
						d23 = snap456
						d24 = snap457
						d25 = snap458
						d27 = snap459
						d28 = snap460
						d29 = snap461
						d30 = snap462
						d31 = snap463
						d33 = snap464
						d34 = snap465
						d57 = snap466
						d58 = snap467
						d59 = snap468
						d60 = snap469
						d61 = snap470
						d62 = snap471
						d63 = snap472
						d64 = snap473
						d65 = snap474
						d66 = snap475
						d67 = snap476
						d101 = snap477
						d136 = snap478
						d137 = snap479
						d138 = snap480
						d139 = snap481
						d140 = snap482
						d141 = snap483
						d142 = snap484
						d143 = snap485
						d186 = snap486
						d187 = snap487
						d188 = snap488
						d189 = snap489
						d190 = snap490
						d238 = snap491
						d239 = snap492
						d240 = snap493
						d241 = snap494
						d242 = snap495
						d243 = snap496
						d244 = snap497
						d245 = snap498
						d301 = snap499
						d358 = snap500
						d359 = snap501
						d362 = snap502
						d363 = snap503
						d364 = snap504
						d366 = snap505
						d367 = snap506
						d368 = snap507
						d369 = snap508
						d370 = snap509
						d372 = snap510
						d373 = snap511
						d442 = snap512
						d443 = snap513
						d444 = snap514
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
					d516 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d517 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d516}, 2)
					ctx.EmitMovPairToResult(&d517, &result)
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
					d518 = args[0]
					d518.ID = 0
					ctx.EnsureDesc(&d518)
					d519 = d518
					_ = d519
					ctx.StabilizeDescForControlFlow(&d519)
					inlineHome520 := d519
					inlineHome520.ID = 0
					phiBase521 = ctx.AllocStack(int32(16))
					d522 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase521) + int32(0)}
					_ = d522
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
					d519 = inlineHome520
					d522 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase521) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d523 = d519
					d523.ID = 0
					d524 = ctx.EmitGetTagDesc(&d523, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d524)
					inlineHome525 := d524
					inlineHome525.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d524)
					if d524.Loc == LocImm {
						d526 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d524.Imm.Int()) == uint64(0x13))}
					} else {
						r21 := ctx.AllocRegExcept(d524.Reg)
						ctx.EmitCmpRegImm32(d524.Reg, 19)
						d526 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondEqual}
						ctx.BindReg(r21, &d526)
					}
					ctx.ReclaimUntrackedRegs()
					d527 = d526
					ctx.SyncDesc(&d527)
					if d527.Loc != LocImm && d527.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl48 := ctx.ReserveLabel()
					lbl49 := ctx.ReserveLabel()
					if d527.Loc == LocImm {
						if d527.Imm.Bool() {
							ctx.MarkLabel(lbl48)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase521)+int32(0))
							ctx.EmitJmp(lbl46)
						} else {
							ctx.MarkLabel(lbl49)
							ctx.EmitJmp(lbl47)
						}
					} else {
						ctx.EmitJump(d527.Condition, lbl48)
						ctx.EmitJmp(lbl49)
						ctx.FreeDesc(&d526)
						ctx.MarkLabel(lbl48)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase521)+int32(0))
						ctx.EmitJmp(lbl46)
						ctx.MarkLabel(lbl49)
						ctx.EmitJmp(lbl47)
					}
					bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl46)
					ctx.ResolveFixups()
					d519 = inlineHome520
					d524 = inlineHome525
					d522 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase521) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r22 := ctx.AllocReg()
					ctx.EnsureDesc(&d522)
					ctx.EnsureDesc(&d522)
					if d522.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r22, d522)
					}
					ctx.EmitJmp(lbl43)
					bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl47)
					ctx.ResolveFixups()
					d519 = inlineHome520
					d524 = inlineHome525
					d522 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase521) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d524)
					if d524.Loc == LocImm {
						d528 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d524.Imm.Int()) == uint64(0x14))}
					} else {
						r23 := ctx.AllocRegExcept(d524.Reg)
						ctx.EmitCmpRegImm32(d524.Reg, 20)
						d528 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondEqual}
						ctx.BindReg(r23, &d528)
					}
					ctx.ReclaimUntrackedRegs()
					d529 = d528
					ctx.SyncDesc(&d529)
					if d529.Loc != LocImm && d529.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl50 := ctx.ReserveLabel()
					lbl51 := ctx.ReserveLabel()
					if d529.Loc == LocImm {
						if d529.Imm.Bool() {
							ctx.MarkLabel(lbl50)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase521)+int32(0))
							ctx.EmitJmp(lbl46)
						} else {
							ctx.MarkLabel(lbl51)
							ctx.EmitJmp(lbl45)
						}
					} else {
						ctx.EmitJump(d529.Condition, lbl50)
						ctx.EmitJmp(lbl51)
						ctx.FreeDesc(&d528)
						ctx.MarkLabel(lbl50)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase521)+int32(0))
						ctx.EmitJmp(lbl46)
						ctx.MarkLabel(lbl51)
						ctx.EmitJmp(lbl45)
					}
					bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl45)
					ctx.ResolveFixups()
					d519 = inlineHome520
					d524 = inlineHome525
					d522 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase521) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d524)
					if d524.Loc == LocImm {
						d530 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d524.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d524.Reg, 26)
						r24 := ctx.AllocRegExcept(d524.Reg)
						ctx.EmitSetcc(r24, CondEqual)
						d530 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d530)
					}
					ctx.EnsureDesc(&d530)
					ctx.EmitStoreToStack(d530, int32(phiBase521)+int32(0))
					ctx.StabilizeDescForControlFlow(&d530)
					inlineHome531 := d530
					inlineHome531.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl46)
					ctx.MarkLabel(lbl43)
					d532 = JITValueDesc{Loc: LocReg, Reg: r22}
					ctx.BindReg(r22, &d532)
					ctx.BindReg(r22, &d532)
					ctx.FreeDesc(&d518)
					d533 = d532
					ctx.EnsureDesc(&d533)
					if d533.Loc != LocImm && d533.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d533.Loc == LocImm {
						if d533.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitCmpRegImm32(d533.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap534 := d1
						snap535 := d2
						snap536 := d3
						snap537 := d4
						snap538 := d5
						snap539 := d6
						snap540 := d7
						snap541 := d8
						snap542 := d18
						snap543 := d19
						snap544 := d20
						snap545 := d23
						snap546 := d24
						snap547 := d25
						snap548 := d27
						snap549 := d28
						snap550 := d29
						snap551 := d30
						snap552 := d31
						snap553 := d33
						snap554 := d34
						snap555 := d57
						snap556 := d58
						snap557 := d59
						snap558 := d60
						snap559 := d61
						snap560 := d62
						snap561 := d63
						snap562 := d64
						snap563 := d65
						snap564 := d66
						snap565 := d67
						snap566 := d101
						snap567 := d136
						snap568 := d137
						snap569 := d138
						snap570 := d139
						snap571 := d140
						snap572 := d141
						snap573 := d142
						snap574 := d143
						snap575 := d186
						snap576 := d187
						snap577 := d188
						snap578 := d189
						snap579 := d190
						snap580 := d238
						snap581 := d239
						snap582 := d240
						snap583 := d241
						snap584 := d242
						snap585 := d243
						snap586 := d244
						snap587 := d245
						snap588 := d301
						snap589 := d358
						snap590 := d359
						snap591 := d362
						snap592 := d363
						snap593 := d364
						snap594 := d366
						snap595 := d367
						snap596 := d368
						snap597 := d369
						snap598 := d370
						snap599 := d372
						snap600 := d373
						snap601 := d442
						snap602 := d443
						snap603 := d444
						snap604 := d516
						snap605 := d517
						snap606 := d518
						snap607 := d519
						snap608 := d522
						snap609 := d523
						snap610 := d524
						snap611 := d526
						snap612 := d527
						snap613 := d528
						snap614 := d529
						snap615 := d530
						snap616 := d532
						snap617 := d533
						alloc618 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc618)
						d1 = snap534
						d2 = snap535
						d3 = snap536
						d4 = snap537
						d5 = snap538
						d6 = snap539
						d7 = snap540
						d8 = snap541
						d18 = snap542
						d19 = snap543
						d20 = snap544
						d23 = snap545
						d24 = snap546
						d25 = snap547
						d27 = snap548
						d28 = snap549
						d29 = snap550
						d30 = snap551
						d31 = snap552
						d33 = snap553
						d34 = snap554
						d57 = snap555
						d58 = snap556
						d59 = snap557
						d60 = snap558
						d61 = snap559
						d62 = snap560
						d63 = snap561
						d64 = snap562
						d65 = snap563
						d66 = snap564
						d67 = snap565
						d101 = snap566
						d136 = snap567
						d137 = snap568
						d138 = snap569
						d139 = snap570
						d140 = snap571
						d141 = snap572
						d142 = snap573
						d143 = snap574
						d186 = snap575
						d187 = snap576
						d188 = snap577
						d189 = snap578
						d190 = snap579
						d238 = snap580
						d239 = snap581
						d240 = snap582
						d241 = snap583
						d242 = snap584
						d243 = snap585
						d244 = snap586
						d245 = snap587
						d301 = snap588
						d358 = snap589
						d359 = snap590
						d362 = snap591
						d363 = snap592
						d364 = snap593
						d366 = snap594
						d367 = snap595
						d368 = snap596
						d369 = snap597
						d370 = snap598
						d372 = snap599
						d373 = snap600
						d442 = snap601
						d443 = snap602
						d444 = snap603
						d516 = snap604
						d517 = snap605
						d518 = snap606
						d519 = snap607
						d522 = snap608
						d523 = snap609
						d524 = snap610
						d526 = snap611
						d527 = snap612
						d528 = snap613
						d529 = snap614
						d530 = snap615
						d532 = snap616
						d533 = snap617
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d532)
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
					d619 = args[0]
					d619.ID = 0
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d4)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d4)
					if d3.Loc == LocImm && d4.Loc == LocImm {
						d620 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d4.Imm.Int())}
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r25 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r25, d3.Reg)
						d620 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d620)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						d620 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d620)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d620 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d620)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d620 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d620)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d4)
						r26 := ctx.AllocRegExceptOperand(&d4, d3.Reg)
						ctx.EmitMovRegReg(r26, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r26, &d4)
						d620 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
						ctx.BindReg(r26, &d620)
					}
					if d620.Loc == LocReg && d3.Loc == LocReg && d620.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					d619 = JITPrepareScmerGoArg(ctx, d619)
					if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d620.Loc == LocRegPair || d620.Loc == LocStackPair || d620.Loc == LocRegTriple || d620.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d619)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d620)
					d621 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d619, d3, d620}, 2)
					d621.NoHeapPointer = false
					ctx.BindReg(d621.Reg, &d621)
					ctx.BindReg(d621.Reg2, &d621)
					ctx.FreeDesc(&d619)
					ctx.FreeDesc(&d620)
					ctx.SyncDesc(&d621)
					if d621.Loc == LocRegPair || d621.Loc == LocStackPair || d621.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d621, &result)
						result.Type = d621.Type
					} else {
						switch d621.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d621)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d621)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d621)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d621, &result)
							result.Type = d621.Type
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
						d622 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d4.Imm.Int())}
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r27 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r27, d3.Reg)
						d622 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
						ctx.BindReg(r27, &d622)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						d622 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d622)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d622 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d622)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d622 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d622)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d4)
						r28 := ctx.AllocRegExceptOperand(&d4, d3.Reg)
						ctx.EmitMovRegReg(r28, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r28, &d4)
						d622 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r28}
						ctx.BindReg(r28, &d622)
					}
					if d622.Loc == LocReg && d3.Loc == LocReg && d622.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d622)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d622)
					if d622.Loc == LocImm && d3.Loc == LocImm {
						d624 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d622.Imm.Int() - d3.Imm.Int())}
					} else {
						r29 := ctx.AllocReg()
						if d622.Loc == LocImm {
							ctx.EmitMovRegImm64(r29, uint64(d622.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r29, d622.Reg)
						}
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitSubInt64(r29, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r29, d3.Reg)
						}
						d624 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r29}
						ctx.BindReg(r29, &d624)
					}
					r30 := ctx.EmitSliceDataAfterLow(&d1, &d3, 1)
					d625 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
					ctx.BindReg(r30, &d625)
					ctx.BindReg(r30, &d625)
					var r31 Reg
					var r32 Reg
					ctx.SyncDesc(&d625)
					ctx.EnsureDesc(&d625)
					if d625.Loc == LocImm {
						r31 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r31, uint64(d625.Imm.Int()))
					} else {
						r31 = d625.Reg
					}
					ctx.ProtectReg(r31)
					ctx.SyncDesc(&d624)
					ctx.EnsureDesc(&d624)
					if d624.Loc == LocImm {
						r32 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r32, uint64(d624.Imm.Int()))
					} else {
						r32 = d624.Reg
					}
					ctx.ProtectReg(r32)
					ctx.UnprotectReg(r32)
					ctx.UnprotectReg(r31)
					d626 = JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
					ctx.BindReg(r31, &d626)
					ctx.BindReg(r32, &d626)
					ctx.BindReg(r31, &d626)
					ctx.BindReg(r32, &d626)
					ctx.FreeDesc(&d622)
					ctx.EnsureDesc(&d626)
					d627 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d626}, 2)
					ctx.EmitMovPairToResult(&d627, &result)
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
					d628 = args[0]
					d628.ID = 0
					d628 = JITPrepareScmerGoArg(ctx, d628)
					if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d628)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d2)
					d629 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d628, d3, d2}, 2)
					d629.NoHeapPointer = false
					ctx.BindReg(d629.Reg, &d629)
					ctx.BindReg(d629.Reg2, &d629)
					ctx.FreeDesc(&d628)
					ctx.SyncDesc(&d629)
					if d629.Loc == LocRegPair || d629.Loc == LocStackPair || d629.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d629, &result)
						result.Type = d629.Type
					} else {
						switch d629.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d629)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d629)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d629)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d629, &result)
							result.Type = d629.Type
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
						d630 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d630)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d630)
					if d630.Loc == LocImm && d3.Loc == LocImm {
						d632 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d630.Imm.Int() - d3.Imm.Int())}
					} else {
						r33 := ctx.AllocReg()
						if d630.Loc == LocImm {
							ctx.EmitMovRegImm64(r33, uint64(d630.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r33, d630.Reg)
						}
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitSubInt64(r33, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r33, d3.Reg)
						}
						d632 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r33}
						ctx.BindReg(r33, &d632)
					}
					r34 := ctx.EmitSliceDataAfterLow(&d1, &d3, 1)
					d633 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r34}
					ctx.BindReg(r34, &d633)
					ctx.BindReg(r34, &d633)
					var r35 Reg
					var r36 Reg
					ctx.SyncDesc(&d633)
					ctx.EnsureDesc(&d633)
					if d633.Loc == LocImm {
						r35 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r35, uint64(d633.Imm.Int()))
					} else {
						r35 = d633.Reg
					}
					ctx.ProtectReg(r35)
					ctx.SyncDesc(&d632)
					ctx.EnsureDesc(&d632)
					if d632.Loc == LocImm {
						r36 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r36, uint64(d632.Imm.Int()))
					} else {
						r36 = d632.Reg
					}
					ctx.ProtectReg(r36)
					ctx.UnprotectReg(r36)
					ctx.UnprotectReg(r35)
					d634 = JITValueDesc{Loc: LocRegPair, Reg: r35, Reg2: r36}
					ctx.BindReg(r35, &d634)
					ctx.BindReg(r36, &d634)
					ctx.BindReg(r35, &d634)
					ctx.BindReg(r36, &d634)
					ctx.EnsureDesc(&d634)
					d635 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d634}, 2)
					ctx.EmitMovPairToResult(&d635, &result)
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
				var d5 JITValueDesc
				_ = d5
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
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
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
					d1 = d0
					_ = d1
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d2 = d1
					d2.ID = 0
					d3 = ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
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
					ctx.FreeDesc(&d0)
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
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					ctx.SyncDesc(&d13)
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d13}, 1)
					d14.NoHeapPointer = true
					ctx.BindReg(d14.Reg, &d14)
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						ctx.EmitMakeInt(result, d14)
					} else {
						ctx.EmitMovToReg(result.Reg2, d14)
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d16)
						if d14.Loc == LocReg && d14.Reg != result.Reg2 {
							ctx.FreeReg(d14.Reg)
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
					d17 = args[0]
					d17.ID = 0
					d17 = JITPrepareScmerGoArg(ctx, d17)
					ctx.SyncDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsCString), []JITValueDesc{d17}, 1)
					d18.NoHeapPointer = true
					ctx.EmitAndRegImm32(d18.Reg, 1)
					d18.Type = tagBool
					ctx.BindReg(d18.Reg, &d18)
					ctx.FreeDesc(&d17)
					d19 = d18
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap20 := d0
						snap21 := d1
						snap22 := d2
						snap23 := d3
						snap24 := d4
						snap25 := d5
						snap26 := d13
						snap27 := d14
						snap28 := d15
						snap29 := d16
						snap30 := d17
						snap31 := d18
						snap32 := d19
						alloc33 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc33)
						d0 = snap20
						d1 = snap21
						d2 = snap22
						d3 = snap23
						d4 = snap24
						d5 = snap25
						d13 = snap26
						d14 = snap27
						d15 = snap28
						d16 = snap29
						d17 = snap30
						d18 = snap31
						d19 = snap32
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d18)
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
					d34 = args[0]
					d34.ID = 0
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocImm {
						_, auxWord := d34.Imm.RawWords()
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d34.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r1 := ctx.AllocReg()
						ctx.EmitMovRegReg(r1, d34.Reg2)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d35)
					}
					ctx.EnsureDesc(&d35)
					d36 = d35
					_ = d36
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d36.Imm.Int()) >> 8))}
					} else {
						r2 := ctx.AllocRegExcept(d36.Reg)
						ctx.EmitMovRegReg(r2, d36.Reg)
						ctx.EmitShrRegImm8(r2, 8)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d37)
					}
					if d37.Loc == LocReg && d36.Loc == LocReg && d37.Reg == d36.Reg {
						ctx.TransferReg(d36.Reg)
						d36.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.FreeDesc(&d35)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() & 4398046511103)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0x3ffffffffff)
						ctx.EmitAndInt64(d37.Reg, ctx.ScratchReg)
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg}
						ctx.BindReg(d37.Reg, &d38)
					}
					if d38.Loc == LocReg && d37.Loc == LocReg && d38.Reg == d37.Reg {
						ctx.TransferReg(d37.Reg)
						d37.Loc = LocNone
					}
					ctx.FreeDesc(&d37)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d38.Imm.Int()))))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegReg(r3, d38.Reg)
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d39)
					}
					ctx.FreeDesc(&d38)
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocImm {
						ctx.EmitMakeInt(result, d39)
					} else {
						ctx.EmitMovToReg(result.Reg2, d39)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d40)
						if d39.Loc == LocReg && d39.Reg != result.Reg2 {
							ctx.FreeReg(d39.Reg)
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
					d41 = args[0]
					d41.ID = 0
					d41 = JITPrepareScmerGoArg(ctx, d41)
					ctx.SyncDesc(&d41)
					d42 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsBString), []JITValueDesc{d41}, 1)
					d42.NoHeapPointer = true
					ctx.EmitAndRegImm32(d42.Reg, 1)
					d42.Type = tagBool
					ctx.BindReg(d42.Reg, &d42)
					ctx.FreeDesc(&d41)
					d43 = d42
					ctx.EnsureDesc(&d43)
					if d43.Loc != LocImm && d43.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d43.Loc == LocImm {
						if d43.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d43.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap44 := d0
						snap45 := d1
						snap46 := d2
						snap47 := d3
						snap48 := d4
						snap49 := d5
						snap50 := d13
						snap51 := d14
						snap52 := d15
						snap53 := d16
						snap54 := d17
						snap55 := d18
						snap56 := d19
						snap57 := d34
						snap58 := d35
						snap59 := d36
						snap60 := d37
						snap61 := d38
						snap62 := d39
						snap63 := d40
						snap64 := d41
						snap65 := d42
						snap66 := d43
						alloc67 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc67)
						d0 = snap44
						d1 = snap45
						d2 = snap46
						d3 = snap47
						d4 = snap48
						d5 = snap49
						d13 = snap50
						d14 = snap51
						d15 = snap52
						d16 = snap53
						d17 = snap54
						d18 = snap55
						d19 = snap56
						d34 = snap57
						d35 = snap58
						d36 = snap59
						d37 = snap60
						d38 = snap61
						d39 = snap62
						d40 = snap63
						d41 = snap64
						d42 = snap65
						d43 = snap66
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					ctx.FreeDesc(&d42)
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
					d68 = args[0]
					d68.ID = 0
					d68 = JITPrepareScmerGoArg(ctx, d68)
					ctx.SyncDesc(&d68)
					d69 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d68}, 1)
					d69.NoHeapPointer = true
					ctx.BindReg(d69.Reg, &d69)
					ctx.FreeDesc(&d68)
					ctx.EnsureDesc(&d69)
					ctx.EnsureDesc(&d69)
					ctx.EnsureDesc(&d69)
					if d69.Loc == LocImm {
						ctx.EmitMakeInt(result, d69)
					} else {
						ctx.EmitMovToReg(result.Reg2, d69)
						d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d71)
						if d69.Loc == LocReg && d69.Reg != result.Reg2 {
							ctx.FreeReg(d69.Reg)
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
					d72 = args[0]
					d72.ID = 0
					d74 = d72
					ctx.SyncDesc(&d74)
					if d74.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d74.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d74.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d74 = tmpScalar
					}
					d74 = JITPrepareScmerGoArg(ctx, d74)
					if d74.Loc != LocRegPair && d74.Loc != LocStackPair && d74.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d73 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d74}, 2)
					ctx.FreeDesc(&d72)
					if d73.SliceSizeKnown {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d73.KnownSliceLen))}
					} else if d73.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d73.Imm.String())))}
					} else if d73.Loc == LocStackTriple {
						d75 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d73.StackOff + 8, NoHeapPointer: true}
					} else if d73.Loc == LocStackPair {
						d75 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d73.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d73)
						if d73.Loc == LocRegPair || d73.Loc == LocRegTriple {
							d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg2, ID: 0}
						} else if d73.Loc == LocReg {
							d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d75
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						ctx.EmitMakeInt(result, d75)
					} else {
						ctx.EmitMovToReg(result.Reg2, d75)
						d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d77)
						if d75.Loc == LocReg && d75.Reg != result.Reg2 {
							ctx.FreeReg(d75.Reg)
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
				var phiBase74 int32
				_ = phiBase74
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
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d121 JITValueDesc
				_ = d121
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
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
					_ = d16
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
					inlineHome73 := d72
					inlineHome73.ID = 0
					phiBase74 = ctx.AllocStack(int32(16))
					d75 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase74) + int32(0)}
					_ = d75
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
					d72 = inlineHome73
					d75 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase74) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d76 = d72
					d76.ID = 0
					d77 = ctx.EmitGetTagDesc(&d76, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d77)
					inlineHome78 := d77
					inlineHome78.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d77.Imm.Int()) == uint64(0x13))}
					} else {
						r1 := ctx.AllocRegExcept(d77.Reg)
						ctx.EmitCmpRegImm32(d77.Reg, 19)
						d79 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d79)
					}
					ctx.ReclaimUntrackedRegs()
					d80 = d79
					ctx.SyncDesc(&d80)
					if d80.Loc != LocImm && d80.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d80.Loc == LocImm {
						if d80.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase74)+int32(0))
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl15)
						}
					} else {
						ctx.EmitJump(d80.Condition, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.FreeDesc(&d79)
						ctx.MarkLabel(lbl16)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase74)+int32(0))
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d72 = inlineHome73
					d77 = inlineHome78
					d75 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase74) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r2 := ctx.AllocReg()
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d75)
					}
					ctx.EmitJmp(lbl11)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d72 = inlineHome73
					d77 = inlineHome78
					d75 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase74) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d77.Imm.Int()) == uint64(0x14))}
					} else {
						r3 := ctx.AllocRegExcept(d77.Reg)
						ctx.EmitCmpRegImm32(d77.Reg, 20)
						d81 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d81)
					}
					ctx.ReclaimUntrackedRegs()
					d82 = d81
					ctx.SyncDesc(&d82)
					if d82.Loc != LocImm && d82.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					if d82.Loc == LocImm {
						if d82.Imm.Bool() {
							ctx.MarkLabel(lbl18)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase74)+int32(0))
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitJump(d82.Condition, lbl18)
						ctx.EmitJmp(lbl19)
						ctx.FreeDesc(&d81)
						ctx.MarkLabel(lbl18)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase74)+int32(0))
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl13)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d72 = inlineHome73
					d77 = inlineHome78
					d75 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase74) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d77.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d77.Reg, 26)
						r4 := ctx.AllocRegExcept(d77.Reg)
						ctx.EmitSetcc(r4, CondEqual)
						d83 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d83)
					}
					ctx.EnsureDesc(&d83)
					ctx.EmitStoreToStack(d83, int32(phiBase74)+int32(0))
					ctx.StabilizeDescForControlFlow(&d83)
					inlineHome84 := d83
					inlineHome84.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl11)
					d85 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d85)
					ctx.BindReg(r2, &d85)
					ctx.FreeDesc(&d71)
					d86 = d85
					ctx.EnsureDesc(&d86)
					if d86.Loc != LocImm && d86.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d86.Loc == LocImm {
						if d86.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d86.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap87 := d1
						snap88 := d2
						snap89 := d3
						snap90 := d4
						snap91 := d5
						snap92 := d12
						snap93 := d13
						snap94 := d14
						snap95 := d15
						snap96 := d16
						snap97 := d17
						snap98 := d18
						snap99 := d45
						snap100 := d46
						snap101 := d47
						snap102 := d48
						snap103 := d66
						snap104 := d67
						snap105 := d68
						snap106 := d69
						snap107 := d70
						snap108 := d71
						snap109 := d72
						snap110 := d75
						snap111 := d76
						snap112 := d77
						snap113 := d79
						snap114 := d80
						snap115 := d81
						snap116 := d82
						snap117 := d83
						snap118 := d85
						snap119 := d86
						alloc120 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc120)
						d1 = snap87
						d2 = snap88
						d3 = snap89
						d4 = snap90
						d5 = snap91
						d12 = snap92
						d13 = snap93
						d14 = snap94
						d15 = snap95
						d16 = snap96
						d17 = snap97
						d18 = snap98
						d45 = snap99
						d46 = snap100
						d47 = snap101
						d48 = snap102
						d66 = snap103
						d67 = snap104
						d68 = snap105
						d69 = snap106
						d70 = snap107
						d71 = snap108
						d72 = snap109
						d75 = snap110
						d76 = snap111
						d77 = snap112
						d79 = snap113
						d80 = snap114
						d81 = snap115
						d82 = snap116
						d83 = snap117
						d85 = snap118
						d86 = snap119
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d85)
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
					d121 = args[0]
					d121.ID = 0
					d121 = JITPrepareScmerGoArg(ctx, d121)
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
					ctx.SyncDesc(&d121)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d1)
					callResults122 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeCString), []JITValueDesc{d121, d14, d1}, []uint8{1, 1}, []uint8{0, 0})
					d123 = callResults122[0]
					_ = d123
					d124 = callResults122[1]
					_ = d124
					ctx.FreeDesc(&d121)
					ctx.StabilizeDescForControlFlow(&d123)
					d125 = d124
					ctx.EnsureDesc(&d125)
					if d125.Loc != LocImm && d125.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d125.Loc == LocImm {
						if d125.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d125.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap126 := d1
						snap127 := d2
						snap128 := d3
						snap129 := d4
						snap130 := d5
						snap131 := d12
						snap132 := d13
						snap133 := d14
						snap134 := d15
						snap135 := d16
						snap136 := d17
						snap137 := d18
						snap138 := d45
						snap139 := d46
						snap140 := d47
						snap141 := d48
						snap142 := d66
						snap143 := d67
						snap144 := d68
						snap145 := d69
						snap146 := d70
						snap147 := d71
						snap148 := d72
						snap149 := d75
						snap150 := d76
						snap151 := d77
						snap152 := d79
						snap153 := d80
						snap154 := d81
						snap155 := d82
						snap156 := d83
						snap157 := d85
						snap158 := d86
						snap159 := d121
						snap160 := d123
						snap161 := d124
						snap162 := d125
						alloc163 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc163)
						d1 = snap126
						d2 = snap127
						d3 = snap128
						d4 = snap129
						d5 = snap130
						d12 = snap131
						d13 = snap132
						d14 = snap133
						d15 = snap134
						d16 = snap135
						d17 = snap136
						d18 = snap137
						d45 = snap138
						d46 = snap139
						d47 = snap140
						d48 = snap141
						d66 = snap142
						d67 = snap143
						d68 = snap144
						d69 = snap145
						d70 = snap146
						d71 = snap147
						d72 = snap148
						d75 = snap149
						d76 = snap150
						d77 = snap151
						d79 = snap152
						d80 = snap153
						d81 = snap154
						d82 = snap155
						d83 = snap156
						d85 = snap157
						d86 = snap158
						d121 = snap159
						d123 = snap160
						d124 = snap161
						d125 = snap162
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d124)
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
					d164 = args[0]
					d164.ID = 0
					d166 = d164
					ctx.SyncDesc(&d166)
					if d166.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d166.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d166.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d166 = tmpScalar
					}
					d166 = JITPrepareScmerGoArg(ctx, d166)
					if d166.Loc != LocRegPair && d166.Loc != LocStackPair && d166.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d165 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d166}, 2)
					ctx.FreeDesc(&d164)
					ctx.EnsureDesc(&d165)
					if d165.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d165.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d165.Imm)
						ptrWord, _ := d165.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d165.Imm.String())))
						d165 = tmpPair
					} else if d165.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d165.Type, Reg: ctx.AllocRegExcept(d165.Reg), Reg2: ctx.AllocRegExcept(d165.Reg)}
						switch d165.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d165)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d165)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d165)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d165)
						d165 = tmpPair
					}
					if d165.Loc != LocRegPair && d165.Loc != LocStackPair && d165.Loc != LocInputPair {
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
					ctx.SyncDesc(&d165)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d1)
					d167 = ctx.EmitGoCallScalar(GoFuncAddr(StrLikeCollation), []JITValueDesc{d165, d14, d1}, 1)
					d167.NoHeapPointer = true
					ctx.EmitAndRegImm32(d167.Reg, 1)
					d167.Type = tagBool
					ctx.BindReg(d167.Reg, &d167)
					ctx.SyncDesc(&d167)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d167) {
						return d167
					}
					if d167.Loc == LocImm {
						ctx.EmitMakeBool(result, d167)
					} else {
						ctx.EmitMovToReg(result.Reg2, d167)
						d168 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d168)
						if d167.Loc == LocReg && d167.Reg != result.Reg2 {
							ctx.FreeReg(d167.Reg)
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
					ctx.SyncDesc(&d123)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d123) {
						return d123
					}
					if d123.Loc == LocImm {
						ctx.EmitMakeBool(result, d123)
					} else {
						ctx.EmitMovToReg(result.Reg2, d123)
						d169 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d169)
						if d123.Loc == LocReg && d123.Reg != result.Reg2 {
							ctx.FreeReg(d123.Reg)
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
				var phiBase16 int32
				_ = phiBase16
				var d17 JITValueDesc
				_ = d17
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
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
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
					inlineHome15 := d14
					inlineHome15.ID = 0
					phiBase16 = ctx.AllocStack(int32(16))
					d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(0)}
					_ = d17
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
					d14 = inlineHome15
					d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d18 = d14
					d18.ID = 0
					d19 = ctx.EmitGetTagDesc(&d18, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d19)
					inlineHome20 := d19
					inlineHome20.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d19.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 19)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d21)
					}
					ctx.ReclaimUntrackedRegs()
					d22 = d21
					ctx.SyncDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase16)+int32(0))
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						}
					} else {
						ctx.EmitJump(d22.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d21)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase16)+int32(0))
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d19 = inlineHome20
					d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d17)
					}
					ctx.EmitJmp(lbl8)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d19 = inlineHome20
					d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d19.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpRegImm32(d19.Reg, 20)
						d23 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d23)
					}
					ctx.ReclaimUntrackedRegs()
					d24 = d23
					ctx.SyncDesc(&d24)
					if d24.Loc != LocImm && d24.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d24.Loc == LocImm {
						if d24.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase16)+int32(0))
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d24.Condition, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.FreeDesc(&d23)
						ctx.MarkLabel(lbl15)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase16)+int32(0))
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d19 = inlineHome20
					d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d19.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d19.Reg, 26)
						r3 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d25)
					}
					ctx.EnsureDesc(&d25)
					ctx.EmitStoreToStack(d25, int32(phiBase16)+int32(0))
					ctx.StabilizeDescForControlFlow(&d25)
					inlineHome26 := d25
					inlineHome26.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl8)
					d27 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d27)
					ctx.BindReg(r1, &d27)
					ctx.FreeDesc(&d13)
					d28 = d27
					ctx.EnsureDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d28.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap29 := d0
						snap30 := d1
						snap31 := d2
						snap32 := d3
						snap33 := d9
						snap34 := d10
						snap35 := d11
						snap36 := d12
						snap37 := d13
						snap38 := d14
						snap39 := d17
						snap40 := d18
						snap41 := d19
						snap42 := d21
						snap43 := d22
						snap44 := d23
						snap45 := d24
						snap46 := d25
						snap47 := d27
						snap48 := d28
						alloc49 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc49)
						d0 = snap29
						d1 = snap30
						d2 = snap31
						d3 = snap32
						d9 = snap33
						d10 = snap34
						d11 = snap35
						d12 = snap36
						d13 = snap37
						d14 = snap38
						d17 = snap39
						d18 = snap40
						d19 = snap41
						d21 = snap42
						d22 = snap43
						d23 = snap44
						d24 = snap45
						d25 = snap46
						d27 = snap47
						d28 = snap48
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d27)
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d53.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap54 := d0
						snap55 := d1
						snap56 := d2
						snap57 := d3
						snap58 := d9
						snap59 := d10
						snap60 := d11
						snap61 := d12
						snap62 := d13
						snap63 := d14
						snap64 := d17
						snap65 := d18
						snap66 := d19
						snap67 := d21
						snap68 := d22
						snap69 := d23
						snap70 := d24
						snap71 := d25
						snap72 := d27
						snap73 := d28
						snap74 := d50
						snap75 := d51
						snap76 := d52
						snap77 := d53
						alloc78 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc78)
						d0 = snap54
						d1 = snap55
						d2 = snap56
						d3 = snap57
						d9 = snap58
						d10 = snap59
						d11 = snap60
						d12 = snap61
						d13 = snap62
						d14 = snap63
						d17 = snap64
						d18 = snap65
						d19 = snap66
						d21 = snap67
						d22 = snap68
						d23 = snap69
						d24 = snap70
						d25 = snap71
						d27 = snap72
						d28 = snap73
						d50 = snap74
						d51 = snap75
						d52 = snap76
						d53 = snap77
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d51)
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
					d79 = args[0]
					d79.ID = 0
					d79 = JITPrepareScmerGoArg(ctx, d79)
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
					d80 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("bin")}
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d80.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d80.Imm)
						ptrWord, _ := d80.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d80.Imm.String())))
						d80 = tmpPair
					} else if d80.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d80.Type, Reg: ctx.AllocRegExcept(d80.Reg), Reg2: ctx.AllocRegExcept(d80.Reg)}
						switch d80.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d80)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d80)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d80)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d80)
						d80 = tmpPair
					}
					if d80.Loc != LocRegPair && d80.Loc != LocStackPair && d80.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strLikeCString arg2)")
					}
					ctx.SyncDesc(&d79)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d80)
					callResults81 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeCString), []JITValueDesc{d79, d11, d80}, []uint8{1, 1}, []uint8{0, 0})
					ctx.FreeDesc(&d80)
					d82 = callResults81[0]
					_ = d82
					d83 = callResults81[1]
					_ = d83
					ctx.FreeDesc(&d79)
					ctx.StabilizeDescForControlFlow(&d82)
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d84.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap85 := d0
						snap86 := d1
						snap87 := d2
						snap88 := d3
						snap89 := d9
						snap90 := d10
						snap91 := d11
						snap92 := d12
						snap93 := d13
						snap94 := d14
						snap95 := d17
						snap96 := d18
						snap97 := d19
						snap98 := d21
						snap99 := d22
						snap100 := d23
						snap101 := d24
						snap102 := d25
						snap103 := d27
						snap104 := d28
						snap105 := d50
						snap106 := d51
						snap107 := d52
						snap108 := d53
						snap109 := d79
						snap110 := d80
						snap111 := d82
						snap112 := d83
						snap113 := d84
						alloc114 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc114)
						d0 = snap85
						d1 = snap86
						d2 = snap87
						d3 = snap88
						d9 = snap89
						d10 = snap90
						d11 = snap91
						d12 = snap92
						d13 = snap93
						d14 = snap94
						d17 = snap95
						d18 = snap96
						d19 = snap97
						d21 = snap98
						d22 = snap99
						d23 = snap100
						d24 = snap101
						d25 = snap102
						d27 = snap103
						d28 = snap104
						d50 = snap105
						d51 = snap106
						d52 = snap107
						d53 = snap108
						d79 = snap109
						d80 = snap110
						d82 = snap111
						d83 = snap112
						d84 = snap113
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d83)
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
					d115 = args[0]
					d115.ID = 0
					d117 = d115
					ctx.SyncDesc(&d117)
					if d117.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d117.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d117.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d117 = tmpScalar
					}
					d117 = JITPrepareScmerGoArg(ctx, d117)
					if d117.Loc != LocRegPair && d117.Loc != LocStackPair && d117.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d116 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d117}, 2)
					ctx.FreeDesc(&d115)
					ctx.EnsureDesc(&d116)
					if d116.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d116.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d116.Imm)
						ptrWord, _ := d116.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d116.Imm.String())))
						d116 = tmpPair
					} else if d116.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d116.Type, Reg: ctx.AllocRegExcept(d116.Reg), Reg2: ctx.AllocRegExcept(d116.Reg)}
						switch d116.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d116)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d116)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d116)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d116)
						d116 = tmpPair
					}
					if d116.Loc != LocRegPair && d116.Loc != LocStackPair && d116.Loc != LocInputPair {
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
					ctx.SyncDesc(&d116)
					ctx.SyncDesc(&d11)
					d118 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d116, d11}, 1)
					d118.NoHeapPointer = true
					ctx.EmitAndRegImm32(d118.Reg, 1)
					d118.Type = tagBool
					ctx.BindReg(d118.Reg, &d118)
					ctx.SyncDesc(&d118)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d118) {
						return d118
					}
					if d118.Loc == LocImm {
						ctx.EmitMakeBool(result, d118)
					} else {
						ctx.EmitMovToReg(result.Reg2, d118)
						d119 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d119)
						if d118.Loc == LocReg && d118.Reg != result.Reg2 {
							ctx.FreeReg(d118.Reg)
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
					ctx.SyncDesc(&d82)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d82) {
						return d82
					}
					if d82.Loc == LocImm {
						ctx.EmitMakeBool(result, d82)
					} else {
						ctx.EmitMovToReg(result.Reg2, d82)
						d120 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d120)
						if d82.Loc == LocReg && d82.Reg != result.Reg2 {
							ctx.FreeReg(d82.Reg)
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
				var phiBase3 int32
				_ = phiBase3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
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
					inlineHome2 := d1
					inlineHome2.ID = 0
					phiBase3 = ctx.AllocStack(int32(16))
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					_ = d4
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
					d1 = inlineHome2
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d5 = d1
					d5.ID = 0
					d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d6)
					inlineHome7 := d6
					inlineHome7.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d4)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d6.Reg, 26)
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase3)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					inlineHome13 := d12
					inlineHome13.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d0)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d4
						snap19 := d5
						snap20 := d6
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d14
						snap27 := d15
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d0 = snap16
						d1 = snap17
						d4 = snap18
						d5 = snap19
						d6 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d14 = snap26
						d15 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d14)
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
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextCase), []JITValueDesc{d29, d30}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.FreeDesc(&d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d31, &result)
						result.Type = d31.Type
					} else {
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d31)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d31)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d31)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d31, &result)
							result.Type = d31.Type
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
					d32 = args[0]
					d32.ID = 0
					d34 = d32
					ctx.SyncDesc(&d34)
					if d34.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d34.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d34.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d34 = tmpScalar
					}
					d34 = JITPrepareScmerGoArg(ctx, d34)
					if d34.Loc != LocRegPair && d34.Loc != LocStackPair && d34.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d34}, 2)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d33.Imm)
						ptrWord, _ := d33.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d33.Imm.String())))
						d33 = tmpPair
					} else if d33.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocRegExcept(d33.Reg), Reg2: ctx.AllocRegExcept(d33.Reg)}
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d33)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d33)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d33)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d33)
						d33 = tmpPair
					}
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d33)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d33}, 2)
					d35.NoHeapPointer = false
					ctx.BindReg(d35.Reg, &d35)
					ctx.BindReg(d35.Reg2, &d35)
					ctx.EnsureDesc(&d35)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d35}, 2)
					ctx.EmitMovPairToResult(&d36, &result)
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
				var phiBase3 int32
				_ = phiBase3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
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
					inlineHome2 := d1
					inlineHome2.ID = 0
					phiBase3 = ctx.AllocStack(int32(16))
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					_ = d4
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
					d1 = inlineHome2
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d5 = d1
					d5.ID = 0
					d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d6)
					inlineHome7 := d6
					inlineHome7.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d4)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d6.Reg, 26)
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase3)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					inlineHome13 := d12
					inlineHome13.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d0)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d4
						snap19 := d5
						snap20 := d6
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d14
						snap27 := d15
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d0 = snap16
						d1 = snap17
						d4 = snap18
						d5 = snap19
						d6 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d14 = snap26
						d15 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d14)
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
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextCase), []JITValueDesc{d29, d30}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.FreeDesc(&d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d31, &result)
						result.Type = d31.Type
					} else {
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d31)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d31)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d31)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d31, &result)
							result.Type = d31.Type
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
					d32 = args[0]
					d32.ID = 0
					d34 = d32
					ctx.SyncDesc(&d34)
					if d34.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d34.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d34.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d34 = tmpScalar
					}
					d34 = JITPrepareScmerGoArg(ctx, d34)
					if d34.Loc != LocRegPair && d34.Loc != LocStackPair && d34.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d34}, 2)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d33.Imm)
						ptrWord, _ := d33.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d33.Imm.String())))
						d33 = tmpPair
					} else if d33.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocRegExcept(d33.Reg), Reg2: ctx.AllocRegExcept(d33.Reg)}
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d33)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d33)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d33)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d33)
						d33 = tmpPair
					}
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d33)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d33}, 2)
					d35.NoHeapPointer = false
					ctx.BindReg(d35.Reg, &d35)
					ctx.BindReg(d35.Reg2, &d35)
					ctx.EnsureDesc(&d35)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d35}, 2)
					ctx.EmitMovPairToResult(&d36, &result)
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
				var d5 JITValueDesc
				_ = d5
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
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
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
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl4 := ctx.ReserveLabel()
					_ = lbl4
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d2 = d1
					d2.ID = 0
					d3 = ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
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
					ctx.FreeDesc(&d0)
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
					d13 = args[0]
					d13.ID = 0
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
					d17 = args[2]
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
					d13 = JITPrepareScmerGoArg(ctx, d13)
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
						panic("jit: generic call arg expects 2-word value (bufferReplace arg1)")
					}
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
						panic("jit: generic call arg expects 2-word value (bufferReplace arg2)")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d18)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(bufferReplace), []JITValueDesc{d13, d15, d18}, 2)
					d20.NoHeapPointer = false
					ctx.BindReg(d20.Reg, &d20)
					ctx.BindReg(d20.Reg2, &d20)
					ctx.FreeDesc(&d13)
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
					d21 = args[0]
					d21.ID = 0
					d23 = d21
					ctx.SyncDesc(&d23)
					if d23.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d23.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d23.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d23 = tmpScalar
					}
					d23 = JITPrepareScmerGoArg(ctx, d23)
					if d23.Loc != LocRegPair && d23.Loc != LocStackPair && d23.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d22 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d23}, 2)
					ctx.FreeDesc(&d21)
					d24 = args[1]
					d24.ID = 0
					d26 = d24
					ctx.SyncDesc(&d26)
					if d26.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d26.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d26.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d26 = tmpScalar
					}
					d26 = JITPrepareScmerGoArg(ctx, d26)
					if d26.Loc != LocRegPair && d26.Loc != LocStackPair && d26.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d26}, 2)
					ctx.FreeDesc(&d24)
					d27 = args[2]
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
						panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg0)")
					}
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d25.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d25.Imm)
						ptrWord, _ := d25.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d25.Imm.String())))
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
					if d25.Loc != LocRegPair && d25.Loc != LocStackPair && d25.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg1)")
					}
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
						panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg2)")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d25)
					ctx.SyncDesc(&d28)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ReplaceAll), []JITValueDesc{d22, d25, d28}, 2)
					d30.NoHeapPointer = false
					ctx.BindReg(d30.Reg, &d30)
					ctx.BindReg(d30.Reg2, &d30)
					ctx.EnsureDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d30}, 2)
					ctx.EmitMovPairToResult(&d31, &result)
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
				var phiBase3 int32
				_ = phiBase3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
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
					inlineHome2 := d1
					inlineHome2.ID = 0
					phiBase3 = ctx.AllocStack(int32(16))
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					_ = d4
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
					d1 = inlineHome2
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d5 = d1
					d5.ID = 0
					d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d6)
					inlineHome7 := d6
					inlineHome7.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d4)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d6.Reg, 26)
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase3)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					inlineHome13 := d12
					inlineHome13.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d0)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d4
						snap19 := d5
						snap20 := d6
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d14
						snap27 := d15
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d0 = snap16
						d1 = snap17
						d4 = snap18
						d5 = snap19
						d6 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d14 = snap26
						d15 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d14)
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
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d29, d30}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.FreeDesc(&d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d31, &result)
						result.Type = d31.Type
					} else {
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d31)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d31)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d31)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d31, &result)
							result.Type = d31.Type
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
					d32 = args[0]
					d32.ID = 0
					d34 = d32
					ctx.SyncDesc(&d34)
					if d34.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d34.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d34.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d34 = tmpScalar
					}
					d34 = JITPrepareScmerGoArg(ctx, d34)
					if d34.Loc != LocRegPair && d34.Loc != LocStackPair && d34.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d34}, 2)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d33.Imm)
						ptrWord, _ := d33.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d33.Imm.String())))
						d33 = tmpPair
					} else if d33.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocRegExcept(d33.Reg), Reg2: ctx.AllocRegExcept(d33.Reg)}
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d33)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d33)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d33)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d33)
						d33 = tmpPair
					}
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d33)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d33}, 2)
					d35.NoHeapPointer = false
					ctx.BindReg(d35.Reg, &d35)
					ctx.BindReg(d35.Reg2, &d35)
					ctx.EnsureDesc(&d35)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d35}, 2)
					ctx.EmitMovPairToResult(&d36, &result)
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
				var phiBase3 int32
				_ = phiBase3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
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
					inlineHome2 := d1
					inlineHome2.ID = 0
					phiBase3 = ctx.AllocStack(int32(16))
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					_ = d4
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
					d1 = inlineHome2
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d5 = d1
					d5.ID = 0
					d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d6)
					inlineHome7 := d6
					inlineHome7.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d4)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d6.Reg, 26)
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase3)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					inlineHome13 := d12
					inlineHome13.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d0)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d4
						snap19 := d5
						snap20 := d6
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d14
						snap27 := d15
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d0 = snap16
						d1 = snap17
						d4 = snap18
						d5 = snap19
						d6 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d14 = snap26
						d15 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d14)
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
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d29, d30}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.FreeDesc(&d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d31, &result)
						result.Type = d31.Type
					} else {
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d31)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d31)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d31)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d31, &result)
							result.Type = d31.Type
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
					d32 = args[0]
					d32.ID = 0
					d34 = d32
					ctx.SyncDesc(&d34)
					if d34.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d34.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d34.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d34 = tmpScalar
					}
					d34 = JITPrepareScmerGoArg(ctx, d34)
					if d34.Loc != LocRegPair && d34.Loc != LocStackPair && d34.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d34}, 2)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d33.Imm)
						ptrWord, _ := d33.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d33.Imm.String())))
						d33 = tmpPair
					} else if d33.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocRegExcept(d33.Reg), Reg2: ctx.AllocRegExcept(d33.Reg)}
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d33)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d33)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d33)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d33)
						d33 = tmpPair
					}
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d35 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d35.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d35.Imm)
						ptrWord, _ := d35.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d35.Imm.String())))
						d35 = tmpPair
					} else if d35.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d35.Type, Reg: ctx.AllocRegExcept(d35.Reg), Reg2: ctx.AllocRegExcept(d35.Reg)}
						switch d35.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d35)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d35)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d35)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d35)
						d35 = tmpPair
					}
					if d35.Loc != LocRegPair && d35.Loc != LocStackPair && d35.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d33)
					ctx.SyncDesc(&d35)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d33, d35}, 2)
					d36.NoHeapPointer = false
					ctx.BindReg(d36.Reg, &d36)
					ctx.BindReg(d36.Reg2, &d36)
					ctx.FreeDesc(&d35)
					ctx.EnsureDesc(&d36)
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d36}, 2)
					ctx.EmitMovPairToResult(&d37, &result)
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
				var phiBase3 int32
				_ = phiBase3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
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
					inlineHome2 := d1
					inlineHome2.ID = 0
					phiBase3 = ctx.AllocStack(int32(16))
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					_ = d4
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
					d1 = inlineHome2
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d5 = d1
					d5.ID = 0
					d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d6)
					inlineHome7 := d6
					inlineHome7.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 19)
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d8)
					}
					ctx.ReclaimUntrackedRegs()
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d4)
					}
					ctx.EmitJmp(lbl4)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpRegImm32(d6.Reg, 20)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl6)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = inlineHome2
					d6 = inlineHome7
					d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d6.Reg, 26)
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.EnsureDesc(&d12)
					ctx.EmitStoreToStack(d12, int32(phiBase3)+int32(0))
					ctx.StabilizeDescForControlFlow(&d12)
					inlineHome13 := d12
					inlineHome13.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d0)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap16 := d0
						snap17 := d1
						snap18 := d4
						snap19 := d5
						snap20 := d6
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d14
						snap27 := d15
						alloc28 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc28)
						d0 = snap16
						d1 = snap17
						d4 = snap18
						d5 = snap19
						d6 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d14 = snap26
						d15 = snap27
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d14)
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
					d29 = args[0]
					d29.ID = 0
					d29 = JITPrepareScmerGoArg(ctx, d29)
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d29, d30}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.FreeDesc(&d30)
					ctx.FreeDesc(&d29)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d31, &result)
						result.Type = d31.Type
					} else {
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d31)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d31)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d31)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d31, &result)
							result.Type = d31.Type
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
					d32 = args[0]
					d32.ID = 0
					d34 = d32
					ctx.SyncDesc(&d34)
					if d34.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d34.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d34.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d34 = tmpScalar
					}
					d34 = JITPrepareScmerGoArg(ctx, d34)
					if d34.Loc != LocRegPair && d34.Loc != LocStackPair && d34.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d34}, 2)
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d33.Imm)
						ptrWord, _ := d33.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d33.Imm.String())))
						d33 = tmpPair
					} else if d33.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d33.Type, Reg: ctx.AllocRegExcept(d33.Reg), Reg2: ctx.AllocRegExcept(d33.Reg)}
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d33)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d33)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d33)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d33)
						d33 = tmpPair
					}
					if d33.Loc != LocRegPair && d33.Loc != LocStackPair && d33.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d35 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d35.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d35.Imm)
						ptrWord, _ := d35.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d35.Imm.String())))
						d35 = tmpPair
					} else if d35.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d35.Type, Reg: ctx.AllocRegExcept(d35.Reg), Reg2: ctx.AllocRegExcept(d35.Reg)}
						switch d35.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d35)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d35)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d35)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d35)
						d35 = tmpPair
					}
					if d35.Loc != LocRegPair && d35.Loc != LocStackPair && d35.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d33)
					ctx.SyncDesc(&d35)
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d33, d35}, 2)
					d36.NoHeapPointer = false
					ctx.BindReg(d36.Reg, &d36)
					ctx.BindReg(d36.Reg2, &d36)
					ctx.FreeDesc(&d35)
					ctx.EnsureDesc(&d36)
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d36}, 2)
					ctx.EmitMovPairToResult(&d37, &result)
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
				var phiBase13 int32
				_ = phiBase13
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
				var d22 JITValueDesc
				_ = d22
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
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
					inlineHome12 := d11
					inlineHome12.ID = 0
					phiBase13 = ctx.AllocStack(int32(16))
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					_ = d14
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
					d11 = inlineHome12
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d15 = d11
					d15.ID = 0
					d16 = ctx.EmitGetTagDesc(&d15, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d16)
					inlineHome17 := d16
					inlineHome17.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 19)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d14)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 20)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d20)
					}
					ctx.ReclaimUntrackedRegs()
					d21 = d20
					ctx.SyncDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d21.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d20)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d16.Reg, 26)
						r3 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d22)
					}
					ctx.EnsureDesc(&d22)
					ctx.EmitStoreToStack(d22, int32(phiBase13)+int32(0))
					ctx.StabilizeDescForControlFlow(&d22)
					inlineHome23 := d22
					inlineHome23.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d24 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d24)
					ctx.BindReg(r1, &d24)
					ctx.FreeDesc(&d10)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap26 := d0
						snap27 := d1
						snap28 := d2
						snap29 := d3
						snap30 := d9
						snap31 := d10
						snap32 := d11
						snap33 := d14
						snap34 := d15
						snap35 := d16
						snap36 := d18
						snap37 := d19
						snap38 := d20
						snap39 := d21
						snap40 := d22
						snap41 := d24
						snap42 := d25
						alloc43 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc43)
						d0 = snap26
						d1 = snap27
						d2 = snap28
						d3 = snap29
						d9 = snap30
						d10 = snap31
						d11 = snap32
						d14 = snap33
						d15 = snap34
						d16 = snap35
						d18 = snap36
						d19 = snap37
						d20 = snap38
						d21 = snap39
						d22 = snap40
						d24 = snap41
						d25 = snap42
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d44 = args[0]
					d44.ID = 0
					d44 = JITPrepareScmerGoArg(ctx, d44)
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d45.Loc == LocRegPair || d45.Loc == LocStackPair || d45.Loc == LocRegTriple || d45.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d44)
					ctx.SyncDesc(&d45)
					d46 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d44, d45}, 2)
					d46.NoHeapPointer = false
					ctx.BindReg(d46.Reg, &d46)
					ctx.BindReg(d46.Reg2, &d46)
					ctx.FreeDesc(&d45)
					ctx.FreeDesc(&d44)
					ctx.SyncDesc(&d46)
					if d46.Loc == LocRegPair || d46.Loc == LocStackPair || d46.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d46, &result)
						result.Type = d46.Type
					} else {
						switch d46.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d46)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d46)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d46)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d46, &result)
							result.Type = d46.Type
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
					d47 = args[0]
					d47.ID = 0
					d49 = d47
					ctx.SyncDesc(&d49)
					if d49.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d49.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d49.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d49 = tmpScalar
					}
					d49 = JITPrepareScmerGoArg(ctx, d49)
					if d49.Loc != LocRegPair && d49.Loc != LocStackPair && d49.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d49}, 2)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d48.Imm)
						ptrWord, _ := d48.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d48.Imm.String())))
						d48 = tmpPair
					} else if d48.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocRegExcept(d48.Reg), Reg2: ctx.AllocRegExcept(d48.Reg)}
						switch d48.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d48)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d48)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d48)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d48)
						d48 = tmpPair
					}
					if d48.Loc != LocRegPair && d48.Loc != LocStackPair && d48.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d48)
					d50 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d48}, 2)
					d50.NoHeapPointer = false
					ctx.BindReg(d50.Reg, &d50)
					ctx.BindReg(d50.Reg2, &d50)
					ctx.EnsureDesc(&d50)
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d50}, 2)
					ctx.EmitMovPairToResult(&d51, &result)
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
				var phiBase13 int32
				_ = phiBase13
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
				var d22 JITValueDesc
				_ = d22
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
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
					inlineHome12 := d11
					inlineHome12.ID = 0
					phiBase13 = ctx.AllocStack(int32(16))
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					_ = d14
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
					d11 = inlineHome12
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d15 = d11
					d15.ID = 0
					d16 = ctx.EmitGetTagDesc(&d15, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d16)
					inlineHome17 := d16
					inlineHome17.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 19)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d14)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 20)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d20)
					}
					ctx.ReclaimUntrackedRegs()
					d21 = d20
					ctx.SyncDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d21.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d20)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d16.Reg, 26)
						r3 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d22)
					}
					ctx.EnsureDesc(&d22)
					ctx.EmitStoreToStack(d22, int32(phiBase13)+int32(0))
					ctx.StabilizeDescForControlFlow(&d22)
					inlineHome23 := d22
					inlineHome23.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d24 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d24)
					ctx.BindReg(r1, &d24)
					ctx.FreeDesc(&d10)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap26 := d0
						snap27 := d1
						snap28 := d2
						snap29 := d3
						snap30 := d9
						snap31 := d10
						snap32 := d11
						snap33 := d14
						snap34 := d15
						snap35 := d16
						snap36 := d18
						snap37 := d19
						snap38 := d20
						snap39 := d21
						snap40 := d22
						snap41 := d24
						snap42 := d25
						alloc43 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc43)
						d0 = snap26
						d1 = snap27
						d2 = snap28
						d3 = snap29
						d9 = snap30
						d10 = snap31
						d11 = snap32
						d14 = snap33
						d15 = snap34
						d16 = snap35
						d18 = snap36
						d19 = snap37
						d20 = snap38
						d21 = snap39
						d22 = snap40
						d24 = snap41
						d25 = snap42
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d44 = args[0]
					d44.ID = 0
					d44 = JITPrepareScmerGoArg(ctx, d44)
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d45.Loc == LocRegPair || d45.Loc == LocStackPair || d45.Loc == LocRegTriple || d45.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d44)
					ctx.SyncDesc(&d45)
					d46 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d44, d45}, 2)
					d46.NoHeapPointer = false
					ctx.BindReg(d46.Reg, &d46)
					ctx.BindReg(d46.Reg2, &d46)
					ctx.FreeDesc(&d45)
					ctx.FreeDesc(&d44)
					ctx.SyncDesc(&d46)
					if d46.Loc == LocRegPair || d46.Loc == LocStackPair || d46.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d46, &result)
						result.Type = d46.Type
					} else {
						switch d46.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d46)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d46)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d46)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d46, &result)
							result.Type = d46.Type
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
					d47 = args[0]
					d47.ID = 0
					d49 = d47
					ctx.SyncDesc(&d49)
					if d49.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d49.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d49.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d49 = tmpScalar
					}
					d49 = JITPrepareScmerGoArg(ctx, d49)
					if d49.Loc != LocRegPair && d49.Loc != LocStackPair && d49.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d49}, 2)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d48.Imm)
						ptrWord, _ := d48.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d48.Imm.String())))
						d48 = tmpPair
					} else if d48.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocRegExcept(d48.Reg), Reg2: ctx.AllocRegExcept(d48.Reg)}
						switch d48.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d48)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d48)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d48)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d48)
						d48 = tmpPair
					}
					if d48.Loc != LocRegPair && d48.Loc != LocStackPair && d48.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d50 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d50)
					if d50.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d50.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d50.Imm)
						ptrWord, _ := d50.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d50.Imm.String())))
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
					if d50.Loc != LocRegPair && d50.Loc != LocStackPair && d50.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d48)
					ctx.SyncDesc(&d50)
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d48, d50}, 2)
					d51.NoHeapPointer = false
					ctx.BindReg(d51.Reg, &d51)
					ctx.BindReg(d51.Reg2, &d51)
					ctx.FreeDesc(&d50)
					ctx.EnsureDesc(&d51)
					d52 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d51}, 2)
					ctx.EmitMovPairToResult(&d52, &result)
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
				var phiBase13 int32
				_ = phiBase13
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
				var d22 JITValueDesc
				_ = d22
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
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
					inlineHome12 := d11
					inlineHome12.ID = 0
					phiBase13 = ctx.AllocStack(int32(16))
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					_ = d14
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
					d11 = inlineHome12
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d15 = d11
					d15.ID = 0
					d16 = ctx.EmitGetTagDesc(&d15, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d16)
					inlineHome17 := d16
					inlineHome17.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 19)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d14)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 20)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d20)
					}
					ctx.ReclaimUntrackedRegs()
					d21 = d20
					ctx.SyncDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d21.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d20)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d11 = inlineHome12
					d16 = inlineHome17
					d14 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d16.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d16.Reg, 26)
						r3 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d22)
					}
					ctx.EnsureDesc(&d22)
					ctx.EmitStoreToStack(d22, int32(phiBase13)+int32(0))
					ctx.StabilizeDescForControlFlow(&d22)
					inlineHome23 := d22
					inlineHome23.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d24 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d24)
					ctx.BindReg(r1, &d24)
					ctx.FreeDesc(&d10)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap26 := d0
						snap27 := d1
						snap28 := d2
						snap29 := d3
						snap30 := d9
						snap31 := d10
						snap32 := d11
						snap33 := d14
						snap34 := d15
						snap35 := d16
						snap36 := d18
						snap37 := d19
						snap38 := d20
						snap39 := d21
						snap40 := d22
						snap41 := d24
						snap42 := d25
						alloc43 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc43)
						d0 = snap26
						d1 = snap27
						d2 = snap28
						d3 = snap29
						d9 = snap30
						d10 = snap31
						d11 = snap32
						d14 = snap33
						d15 = snap34
						d16 = snap35
						d18 = snap36
						d19 = snap37
						d20 = snap38
						d21 = snap39
						d22 = snap40
						d24 = snap41
						d25 = snap42
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d44 = args[0]
					d44.ID = 0
					d44 = JITPrepareScmerGoArg(ctx, d44)
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d45.Loc == LocRegPair || d45.Loc == LocStackPair || d45.Loc == LocRegTriple || d45.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d44)
					ctx.SyncDesc(&d45)
					d46 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d44, d45}, 2)
					d46.NoHeapPointer = false
					ctx.BindReg(d46.Reg, &d46)
					ctx.BindReg(d46.Reg2, &d46)
					ctx.FreeDesc(&d45)
					ctx.FreeDesc(&d44)
					ctx.SyncDesc(&d46)
					if d46.Loc == LocRegPair || d46.Loc == LocStackPair || d46.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d46, &result)
						result.Type = d46.Type
					} else {
						switch d46.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d46)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d46)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d46)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d46, &result)
							result.Type = d46.Type
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
					d47 = args[0]
					d47.ID = 0
					d49 = d47
					ctx.SyncDesc(&d49)
					if d49.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d49.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d49.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d49 = tmpScalar
					}
					d49 = JITPrepareScmerGoArg(ctx, d49)
					if d49.Loc != LocRegPair && d49.Loc != LocStackPair && d49.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d49}, 2)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d48.Imm)
						ptrWord, _ := d48.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d48.Imm.String())))
						d48 = tmpPair
					} else if d48.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocRegExcept(d48.Reg), Reg2: ctx.AllocRegExcept(d48.Reg)}
						switch d48.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d48)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d48)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d48)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d48)
						d48 = tmpPair
					}
					if d48.Loc != LocRegPair && d48.Loc != LocStackPair && d48.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d50 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
					ctx.EnsureDesc(&d50)
					if d50.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d50.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d50.Imm)
						ptrWord, _ := d50.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d50.Imm.String())))
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
					if d50.Loc != LocRegPair && d50.Loc != LocStackPair && d50.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d48)
					ctx.SyncDesc(&d50)
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d48, d50}, 2)
					d51.NoHeapPointer = false
					ctx.BindReg(d51.Reg, &d51)
					ctx.BindReg(d51.Reg2, &d51)
					ctx.FreeDesc(&d50)
					ctx.EnsureDesc(&d51)
					d52 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d51}, 2)
					ctx.EmitMovPairToResult(&d52, &result)
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
					_ = d3
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
					_ = d26
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
					_ = d29
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
					ctx.SyncDesc(&d28)
					ctx.SyncDesc(&d30)
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
					ctx.EnsureDesc(&d11)
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
						boxed := JITPrepareScmerGoArg(ctx, d11)
						d12 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d6)
					}
					d7 = d6
					ctx.SyncDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d7.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d6)
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
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d5)
					}
					d6 = d5
					ctx.SyncDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d6.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d5)
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
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d19)
					}
					d20 = d19
					ctx.SyncDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d20.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d19)
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
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d8)
					}
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d9.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d8)
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
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondNotEqual}
						ctx.BindReg(r2, &d8)
					}
					d9 = d8
					ctx.SyncDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d9.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d8)
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
				inlineHome2 := d1
				inlineHome2.ID = 0
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
				d1 = inlineHome2
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d1)
				d3 := d1
				_ = d3
				ctx.StabilizeDescForControlFlow(&d1)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d4 := d3
				d4.ID = 0
				d5 := ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d5)
				var d6 JITValueDesc
				if d5.Loc == LocImm {
					d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x1a))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d5.Reg, 26)
					d6 = ctx.DeferBooleanFlags(r0, CondEqual)
					ctx.BindReg(r0, &d6)
				}
				ctx.FreeDesc(&d5)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d7 := d6
				ctx.SyncDesc(&d7)
				if d7.Loc != LocImm && d7.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl5 := ctx.ReserveLabel()
				lbl6 := ctx.ReserveLabel()
				if d7.Loc == LocImm {
					if d7.Imm.Bool() {
						ctx.MarkLabel(lbl5)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl6)
						ctx.EmitJmp(lbl3)
					}
				} else {
					ctx.EmitJump(d7.Condition, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.FreeDesc(&d6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl3)
				}
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d1 = inlineHome2
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d9 := d1
				ctx.SyncDesc(&d9)
				if d9.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d9.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d9.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d9 = tmpScalar
				}
				d9 = JITPrepareScmerGoArg(ctx, d9)
				if d9.Loc != LocRegPair && d9.Loc != LocStackPair && d9.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d8 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d9}, 2)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d8)
				var d10 JITValueDesc
				if d8.Loc == LocImm {
					ptr, _ := d8.Imm.RawWords()
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(ptr))))}
				} else if d8.Loc == LocStackPair {
					d10 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d8.StackOff}
				} else {
					ctx.EnsureDesc(&d8)
					if d8.Loc != LocRegPair {
						panic("StringData requires a Go string pair")
					}
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg, ID: 0}
				}
				ctx.ReclaimUntrackedRegs()
				var d11 JITValueDesc
				if d8.SliceSizeKnown {
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.KnownSliceLen))}
				} else if d8.Loc == LocImm {
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d8.Imm.String())))}
				} else if d8.Loc == LocStackTriple {
					d11 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d8.StackOff + 8, NoHeapPointer: true}
				} else if d8.Loc == LocStackPair {
					d11 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d8.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocRegPair || d8.Loc == LocRegTriple {
						d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg2, ID: 0}
					} else if d8.Loc == LocReg {
						d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d11
				ctx.ReclaimUntrackedRegs()
				if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d12 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d12
				if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d13 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d13
				if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d10)
				ctx.SyncDesc(&d11)
				ctx.SyncDesc(&d12)
				ctx.SyncDesc(&d13)
				d14 := ctx.EmitGoCallScalar(GoFuncAddr(NewBString), []JITValueDesc{d10, d11, d12, d13}, 2)
				d14.NoHeapPointer = false
				ctx.BindReg(d14.Reg, &d14)
				ctx.BindReg(d14.Reg2, &d14)
				ctx.FreeDesc(&d12)
				ctx.FreeDesc(&d13)
				ctx.FreeDesc(&d10)
				ctx.FreeDesc(&d11)
				ctx.ReclaimUntrackedRegs()
				r1 := ctx.AllocReg()
				r2 := ctx.AllocRegExcept(r1)
				d15 := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d15)
				ctx.BindReg(r2, &d15)
				ctx.EmitMovPairToResult(&d14, &d15)
				ctx.EmitJmp(lbl0)
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d1 = inlineHome2
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d1)
				d16 := ctx.EmitGoCallScalar(GoFuncAddr(encodeBufferBase64), []JITValueDesc{d1}, 2)
				d16.NoHeapPointer = false
				ctx.BindReg(d16.Reg, &d16)
				ctx.BindReg(d16.Reg2, &d16)
				ctx.ReclaimUntrackedRegs()
				d17 := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d17)
				ctx.BindReg(r2, &d17)
				ctx.EmitMovPairToResult(&d16, &d17)
				ctx.EmitJmp(lbl0)
				ctx.MarkLabel(lbl0)
				d18 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d18)
				ctx.BindReg(r2, &d18)
				ctx.BindReg(r1, &d18)
				ctx.BindReg(r2, &d18)
				ctx.FreeDesc(&d0)
				if d18.Loc == LocImm {
					if result.Loc == LocAny {
						return d18
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
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
				var d5 JITValueDesc
				_ = d5
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
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
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
				var d99 JITValueDesc
				_ = d99
				var d101 JITValueDesc
				_ = d101
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
					d1 = d0
					_ = d1
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d2 = d1
					d2.ID = 0
					d3 = ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
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
					ctx.FreeDesc(&d0)
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
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					ctx.SyncDesc(&d13)
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(decodeBufferBase64), []JITValueDesc{d13}, 2)
					d14.NoHeapPointer = false
					ctx.BindReg(d14.Reg, &d14)
					ctx.BindReg(d14.Reg2, &d14)
					ctx.FreeDesc(&d13)
					ctx.SyncDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d14, &result)
						result.Type = d14.Type
					} else {
						switch d14.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d14)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d14)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d14)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d14, &result)
							result.Type = d14.Type
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
					d15 = args[0]
					d15.ID = 0
					d15 = JITPrepareScmerGoArg(ctx, d15)
					ctx.SyncDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsBString), []JITValueDesc{d15}, 1)
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
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
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
						snap23 := d5
						snap24 := d13
						snap25 := d14
						snap26 := d15
						snap27 := d16
						snap28 := d17
						alloc29 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc29)
						d0 = snap18
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d4 = snap22
						d5 = snap23
						d13 = snap24
						d14 = snap25
						d15 = snap26
						d16 = snap27
						d17 = snap28
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
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
					d30 = args[0]
					d30.ID = 0
					d30 = JITPrepareScmerGoArg(ctx, d30)
					ctx.SyncDesc(&d30)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(bstringRawText), []JITValueDesc{d30}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.FreeDesc(&d30)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocStackPair || d31.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d31, &result)
						result.Type = d31.Type
					} else {
						switch d31.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d31)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d31)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d31)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d31, &result)
							result.Type = d31.Type
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
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(func() *base64.Encoding { return base64.StdEncoding }), nil, 1)
					d33 = args[0]
					d33.ID = 0
					d35 = d33
					ctx.SyncDesc(&d35)
					if d35.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d35.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d35.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d35 = tmpScalar
					}
					d35 = JITPrepareScmerGoArg(ctx, d35)
					if d35.Loc != LocRegPair && d35.Loc != LocStackPair && d35.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d35}, 2)
					ctx.FreeDesc(&d33)
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocRegTriple || d32.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d34.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d34.Imm)
						ptrWord, _ := d34.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d34.Imm.String())))
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
						panic("jit: generic call arg expects 2-word value ((*base64.Encoding).DecodeString arg1)")
					}
					ctx.SyncDesc(&d32)
					ctx.SyncDesc(&d34)
					callResults36 := JITEmitGoCallResults(ctx, GoFuncAddr((*base64.Encoding).DecodeString), []JITValueDesc{d32, d34}, []uint8{3, 2}, []uint8{1, 3})
					d37 = callResults36[0]
					_ = d37
					d38 = callResults36[1]
					_ = d38
					ctx.FreeDesc(&d32)
					ctx.StabilizeDescForControlFlow(&d37)
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d38.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d38)
						if d38.Loc != LocReg && d38.Loc != LocRegPair && d38.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d38.Reg)
						ctx.EmitCmpRegImm32(d38.Reg, 0)
						d39 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondNotEqual}
						ctx.BindReg(r1, &d39)
					}
					d40 = d39
					ctx.SyncDesc(&d40)
					if d40.Loc != LocImm && d40.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d40.Loc == LocImm {
						if d40.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitJump(d40.Condition, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d39)
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap41 := d0
						snap42 := d1
						snap43 := d2
						snap44 := d3
						snap45 := d4
						snap46 := d5
						snap47 := d13
						snap48 := d14
						snap49 := d15
						snap50 := d16
						snap51 := d17
						snap52 := d30
						snap53 := d31
						snap54 := d32
						snap55 := d33
						snap56 := d34
						snap57 := d35
						snap58 := d37
						snap59 := d38
						snap60 := d39
						snap61 := d40
						alloc62 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc62)
						d0 = snap41
						d1 = snap42
						d2 = snap43
						d3 = snap44
						d4 = snap45
						d5 = snap46
						d13 = snap47
						d14 = snap48
						d15 = snap49
						d16 = snap50
						d17 = snap51
						d30 = snap52
						d31 = snap53
						d32 = snap54
						d33 = snap55
						d34 = snap56
						d35 = snap57
						d37 = snap58
						d38 = snap59
						d39 = snap60
						d40 = snap61
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
					ctx.ReclaimUntrackedRegs()
					d63 = args[0]
					d63.ID = 0
					ctx.EnsureDesc(&d63)
					if d63.Loc == LocImm {
						_, auxWord := d63.Imm.RawWords()
						d64 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d63.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r2 := ctx.AllocReg()
						ctx.EmitMovRegReg(r2, d63.Reg2)
						d64 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d64)
					}
					ctx.EnsureDesc(&d64)
					d65 = d64
					_ = d65
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d65.Imm.Int()) >> 8))}
					} else {
						r3 := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegReg(r3, d65.Reg)
						ctx.EmitShrRegImm8(r3, 8)
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d66)
					}
					if d66.Loc == LocReg && d65.Loc == LocReg && d66.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d66)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d66)
					if d66.Loc == LocImm {
						d67 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d66.Imm.Int()) >> 46))}
					} else {
						ctx.EmitShrRegImm8(d66.Reg, 46)
						d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg}
						ctx.BindReg(d66.Reg, &d67)
					}
					if d67.Loc == LocReg && d66.Loc == LocReg && d67.Reg == d66.Reg {
						ctx.TransferReg(d66.Reg)
						d66.Loc = LocNone
					}
					ctx.FreeDesc(&d66)
					ctx.EnsureDesc(&d67)
					if d67.Loc == LocImm {
						d68 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d67.Imm.Int()) == uint64(0x0))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d67.Reg, 0)
						d68 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
						ctx.BindReg(r4, &d68)
					}
					ctx.FreeDesc(&d67)
					d69 = d68
					ctx.SyncDesc(&d69)
					if d69.Loc != LocImm && d69.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d69.Loc == LocImm {
						if d69.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d69.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d68)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap70 := d0
						snap71 := d1
						snap72 := d2
						snap73 := d3
						snap74 := d4
						snap75 := d5
						snap76 := d13
						snap77 := d14
						snap78 := d15
						snap79 := d16
						snap80 := d17
						snap81 := d30
						snap82 := d31
						snap83 := d32
						snap84 := d33
						snap85 := d34
						snap86 := d35
						snap87 := d37
						snap88 := d38
						snap89 := d39
						snap90 := d40
						snap91 := d63
						snap92 := d64
						snap93 := d65
						snap94 := d66
						snap95 := d67
						snap96 := d68
						snap97 := d69
						alloc98 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc98)
						d0 = snap70
						d1 = snap71
						d2 = snap72
						d3 = snap73
						d4 = snap74
						d5 = snap75
						d13 = snap76
						d14 = snap77
						d15 = snap78
						d16 = snap79
						d17 = snap80
						d30 = snap81
						d31 = snap82
						d32 = snap83
						d33 = snap84
						d34 = snap85
						d35 = snap86
						d37 = snap87
						d38 = snap88
						d39 = snap89
						d40 = snap90
						d63 = snap91
						d64 = snap92
						d65 = snap93
						d66 = snap94
						d67 = snap95
						d68 = snap96
						d69 = snap97
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
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					callResults100 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d37}, []uint8{2}, []uint8{1})
					d99 = callResults100[0]
					ctx.EnsureDesc(&d99)
					d101 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d99}, 2)
					ctx.EmitMovPairToResult(&d101, &result)
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
				var phiBase5 int32
				_ = phiBase5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
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
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
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
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d97 JITValueDesc
				_ = d97
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
					inlineHome4 := d3
					inlineHome4.ID = 0
					phiBase5 = ctx.AllocStack(int32(16))
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					_ = d6
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
					d3 = inlineHome4
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d7 = d3
					d7.ID = 0
					d8 = ctx.EmitGetTagDesc(&d7, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d8)
					inlineHome9 := d8
					inlineHome9.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d8.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpRegImm32(d8.Reg, 19)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d3 = inlineHome4
					d8 = inlineHome9
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d6)
					}
					ctx.EmitJmp(lbl7)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d3 = inlineHome4
					d8 = inlineHome9
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d8.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpRegImm32(d8.Reg, 20)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d12)
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
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl9)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d3 = inlineHome4
					d8 = inlineHome9
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d8.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d8.Reg, 26)
						r3 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d14)
					}
					ctx.EnsureDesc(&d14)
					ctx.EmitStoreToStack(d14, int32(phiBase5)+int32(0))
					ctx.StabilizeDescForControlFlow(&d14)
					inlineHome15 := d14
					inlineHome15.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d16 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d16)
					ctx.BindReg(r1, &d16)
					ctx.FreeDesc(&d2)
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap18 := d1
						snap19 := d2
						snap20 := d3
						snap21 := d6
						snap22 := d7
						snap23 := d8
						snap24 := d10
						snap25 := d11
						snap26 := d12
						snap27 := d13
						snap28 := d14
						snap29 := d16
						snap30 := d17
						alloc31 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc31)
						d1 = snap18
						d2 = snap19
						d3 = snap20
						d6 = snap21
						d7 = snap22
						d8 = snap23
						d10 = snap24
						d11 = snap25
						d12 = snap26
						d13 = snap27
						d14 = snap28
						d16 = snap29
						d17 = snap30
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d16)
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
					d32 = args[0]
					d32.ID = 0
					d32 = JITPrepareScmerGoArg(ctx, d32)
					ctx.SyncDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(encodeTextHex), []JITValueDesc{d32}, 2)
					d33.NoHeapPointer = false
					ctx.BindReg(d33.Reg, &d33)
					ctx.BindReg(d33.Reg2, &d33)
					ctx.FreeDesc(&d32)
					ctx.SyncDesc(&d33)
					if d33.Loc == LocRegPair || d33.Loc == LocStackPair || d33.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d33, &result)
						result.Type = d33.Type
					} else {
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d33)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d33)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d33)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d33, &result)
							result.Type = d33.Type
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
					d34 = args[0]
					d34.ID = 0
					d36 = d34
					ctx.SyncDesc(&d36)
					if d36.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d36.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d36.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d36 = tmpScalar
					}
					d36 = JITPrepareScmerGoArg(ctx, d36)
					if d36.Loc != LocRegPair && d36.Loc != LocStackPair && d36.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d36}, 2)
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.FreeDesc(&d34)
					if d35.SliceSizeKnown {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d35.KnownSliceLen))}
					} else if d35.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d35.Imm.String())))}
					} else if d35.Loc == LocStackTriple {
						d37 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else if d35.Loc == LocStackPair {
						d37 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d35)
						if d35.Loc == LocRegPair || d35.Loc == LocRegTriple {
							d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg2, ID: 0}
						} else if d35.Loc == LocReg {
							d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d37
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d37)
					ctx.SyncDesc(&d38)
					ctx.SyncDesc(&d37)
					if d38.Loc == LocImm && d37.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() * d37.Imm.Int())}
					} else if d38.Loc == LocImm {
						ctx.EnsureDesc(&d37)
						scratch := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitMovRegReg(scratch, d37.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d38.Imm.Int())
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d39)
					} else if d37.Loc == LocImm {
						ctx.EnsureDesc(&d38)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d38.Reg, d37.Imm.Int())
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg}
						ctx.BindReg(d38.Reg, &d39)
					} else {
						ctx.EnsureDesc(&d38)
						ctx.SyncDesc(&d37)
						ctx.EmitIntBinary(JITIntMul, 64, d38.Reg, &d37)
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg}
						ctx.BindReg(d38.Reg, &d39)
					}
					if d39.Loc == LocReg && d38.Loc == LocReg && d39.Reg == d38.Reg {
						ctx.TransferReg(d38.Reg)
						d38.Loc = LocNone
					}
					ctx.FreeDesc(&d37)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					callResults40 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d39, d39}, []uint8{3}, []uint8{1})
					d41 = callResults40[0]
					d41.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d41)
					ctx.FreeDesc(&d39)
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
					if d35.SliceSizeKnown {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d35.KnownSliceLen))}
					} else if d35.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d35.Imm.String())))}
					} else if d35.Loc == LocStackTriple {
						d42 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else if d35.Loc == LocStackPair {
						d42 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d35)
						if d35.Loc == LocRegPair || d35.Loc == LocRegTriple {
							d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg2, ID: 0}
						} else if d35.Loc == LocReg {
							d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d42
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDescsTogether(&d1, &d42)
					if d1.Loc == LocImm && d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d42.Imm.Int())}
					} else if d42.Loc == LocImm {
						r4 := ctx.AllocRegExcept(d1.Reg)
						if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d42.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d42.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d43)
					} else if d1.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d42.Reg)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d43)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d42.Reg)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedLess}
						ctx.BindReg(r6, &d43)
					}
					ctx.FreeDesc(&d42)
					d44 = d43
					ctx.SyncDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d44.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d43)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap45 := d1
						snap46 := d2
						snap47 := d3
						snap48 := d6
						snap49 := d7
						snap50 := d8
						snap51 := d10
						snap52 := d11
						snap53 := d12
						snap54 := d13
						snap55 := d14
						snap56 := d16
						snap57 := d17
						snap58 := d32
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d37
						snap64 := d38
						snap65 := d39
						snap66 := d41
						snap67 := d42
						snap68 := d43
						snap69 := d44
						alloc70 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc70)
						d1 = snap45
						d2 = snap46
						d3 = snap47
						d6 = snap48
						d7 = snap49
						d8 = snap50
						d10 = snap51
						d11 = snap52
						d12 = snap53
						d13 = snap54
						d14 = snap55
						d16 = snap56
						d17 = snap57
						d32 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d37 = snap63
						d38 = snap64
						d39 = snap65
						d41 = snap66
						d42 = snap67
						d43 = snap68
						d44 = snap69
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
					ctx.StabilizeDescForControlFlow(&d41)
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d71)
					ctx.SyncDesc(&d1)
					if d71.Loc == LocImm && d1.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int() * d1.Imm.Int())}
					} else if d71.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d71.Imm.Int())
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d72)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d71)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d71.Reg, d1.Imm.Int())
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d72)
					} else {
						ctx.EnsureDesc(&d71)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d71.Reg, &d1)
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d72)
					}
					if d72.Loc == LocReg && d71.Loc == LocReg && d72.Reg == d71.Reg {
						ctx.TransferReg(d71.Reg)
						d71.Loc = LocNone
					}
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d35)
					d73 = ctx.EmitSliceElementAddress(&d35, &d1, 1)
					ctx.EnsureDesc(&d73)
					r7 := ctx.AllocRegExcept(d73.Reg)
					ctx.EmitMovRegMemB(r7, d73.Reg, 0)
					ctx.FreeDesc(&d73)
					d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d74)
					ctx.BindReg(r7, &d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d74.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d74.Reg, 4)
						d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d74.Reg}
						ctx.BindReg(d74.Reg, &d75)
					}
					if d75.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: d75.Type, Imm: NewInt(int64(uint64(d75.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d75.Reg, 56)
						ctx.EmitShrRegImm8(d75.Reg, 56)
					}
					if d75.Loc == LocReg && d74.Loc == LocReg && d75.Reg == d74.Reg {
						ctx.TransferReg(d74.Reg)
						d74.Loc = LocNone
					}
					ctx.FreeDesc(&d74)
					d76 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d75)
					ctx.EnsureGoStringHeader(&d76)
					d77 = ctx.EmitSliceElementAddress(&d76, &d75, 1)
					ctx.EnsureDesc(&d77)
					r8 := ctx.AllocRegExcept(d77.Reg)
					ctx.EmitMovRegMemB(r8, d77.Reg, 0)
					ctx.FreeDesc(&d77)
					d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8, NoHeapPointer: true}
					ctx.BindReg(r8, &d78)
					ctx.BindReg(r8, &d78)
					ctx.FreeDesc(&d75)
					ctx.EnsureDesc(&d72)
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d72)
					d79 = d41
					d79.ID = 0
					d80 = d72
					d80.ID = 0
					d81 = ctx.EmitSliceElementAddress(&d79, &d80, int32(1))
					ctx.EmitStoreScalarAt(&d81, &d78, 1)
					ctx.FreeDesc(&d81)
					ctx.FreeDesc(&d80)
					ctx.FreeDesc(&d72)
					ctx.FreeDesc(&d78)
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d82)
					ctx.SyncDesc(&d1)
					if d82.Loc == LocImm && d1.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d82.Imm.Int() * d1.Imm.Int())}
					} else if d82.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d82.Imm.Int())
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d83)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d82)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d82.Reg, d1.Imm.Int())
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d82.Reg}
						ctx.BindReg(d82.Reg, &d83)
					} else {
						ctx.EnsureDesc(&d82)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d82.Reg, &d1)
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d82.Reg}
						ctx.BindReg(d82.Reg, &d83)
					}
					if d83.Loc == LocReg && d82.Loc == LocReg && d83.Reg == d82.Reg {
						ctx.TransferReg(d82.Reg)
						d82.Loc = LocNone
					}
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d83.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d83.Reg)
						ctx.EmitMovRegReg(scratch, d83.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d84)
					}
					if d84.Loc == LocReg && d83.Loc == LocReg && d84.Reg == d83.Reg {
						ctx.TransferReg(d83.Reg)
						d83.Loc = LocNone
					}
					ctx.FreeDesc(&d83)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d35)
					d85 = ctx.EmitSliceElementAddress(&d35, &d1, 1)
					ctx.EnsureDesc(&d85)
					r9 := ctx.AllocRegExcept(d85.Reg)
					ctx.EmitMovRegMemB(r9, d85.Reg, 0)
					ctx.FreeDesc(&d85)
					d86 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9, NoHeapPointer: true}
					ctx.BindReg(r9, &d86)
					ctx.BindReg(r9, &d86)
					ctx.EnsureDesc(&d86)
					if d86.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d86.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d86.Reg, 15)
						d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg}
						ctx.BindReg(d86.Reg, &d87)
					}
					if d87.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: d87.Type, Imm: NewInt(int64(uint64(d87.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d87.Reg, 56)
						ctx.EmitShrRegImm8(d87.Reg, 56)
					}
					if d87.Loc == LocReg && d86.Loc == LocReg && d87.Reg == d86.Reg {
						ctx.TransferReg(d86.Reg)
						d86.Loc = LocNone
					}
					ctx.FreeDesc(&d86)
					d88 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d87)
					ctx.EnsureGoStringHeader(&d88)
					d89 = ctx.EmitSliceElementAddress(&d88, &d87, 1)
					ctx.EnsureDesc(&d89)
					r10 := ctx.AllocRegExcept(d89.Reg)
					ctx.EmitMovRegMemB(r10, d89.Reg, 0)
					ctx.FreeDesc(&d89)
					d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10, NoHeapPointer: true}
					ctx.BindReg(r10, &d90)
					ctx.BindReg(r10, &d90)
					ctx.FreeDesc(&d87)
					ctx.EnsureDesc(&d84)
					ctx.SyncDesc(&d90)
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d84)
					d91 = d41
					d91.ID = 0
					d92 = d84
					d92.ID = 0
					d93 = ctx.EmitSliceElementAddress(&d91, &d92, int32(1))
					ctx.EmitStoreScalarAt(&d93, &d90, 1)
					ctx.FreeDesc(&d93)
					ctx.FreeDesc(&d92)
					ctx.FreeDesc(&d84)
					ctx.FreeDesc(&d90)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d94)
					}
					if d94.Loc == LocReg && d1.Loc == LocReg && d94.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d94)
					ctx.EmitStoreToStack(d94, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d94)
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
					ctx.StabilizeDescForControlFlow(&d41)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d41)
					callResults96 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d41}, []uint8{2}, []uint8{1})
					d95 = callResults96[0]
					ctx.EnsureDesc(&d95)
					d97 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d95}, 2)
					ctx.EmitMovPairToResult(&d97, &result)
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
				var phiBase5 int32
				_ = phiBase5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
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
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
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
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d97 JITValueDesc
				_ = d97
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
					inlineHome4 := d3
					inlineHome4.ID = 0
					phiBase5 = ctx.AllocStack(int32(16))
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					_ = d6
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
					d3 = inlineHome4
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d7 = d3
					d7.ID = 0
					d8 = ctx.EmitGetTagDesc(&d7, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d8)
					inlineHome9 := d8
					inlineHome9.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d8.Imm.Int()) == uint64(0x13))}
					} else {
						r0 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpRegImm32(d8.Reg, 19)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.SyncDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d10)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d3 = inlineHome4
					d8 = inlineHome9
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d6)
					}
					ctx.EmitJmp(lbl7)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d3 = inlineHome4
					d8 = inlineHome9
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d8.Imm.Int()) == uint64(0x14))}
					} else {
						r2 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpRegImm32(d8.Reg, 20)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d12)
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
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase5)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl9)
					}
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d3 = inlineHome4
					d8 = inlineHome9
					d6 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d8.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d8.Reg, 26)
						r3 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d14)
					}
					ctx.EnsureDesc(&d14)
					ctx.EmitStoreToStack(d14, int32(phiBase5)+int32(0))
					ctx.StabilizeDescForControlFlow(&d14)
					inlineHome15 := d14
					inlineHome15.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d16 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d16)
					ctx.BindReg(r1, &d16)
					ctx.FreeDesc(&d2)
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap18 := d1
						snap19 := d2
						snap20 := d3
						snap21 := d6
						snap22 := d7
						snap23 := d8
						snap24 := d10
						snap25 := d11
						snap26 := d12
						snap27 := d13
						snap28 := d14
						snap29 := d16
						snap30 := d17
						alloc31 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc31)
						d1 = snap18
						d2 = snap19
						d3 = snap20
						d6 = snap21
						d7 = snap22
						d8 = snap23
						d10 = snap24
						d11 = snap25
						d12 = snap26
						d13 = snap27
						d14 = snap28
						d16 = snap29
						d17 = snap30
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d16)
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
					d32 = args[0]
					d32.ID = 0
					d32 = JITPrepareScmerGoArg(ctx, d32)
					ctx.SyncDesc(&d32)
					d33 = ctx.EmitGoCallScalar(GoFuncAddr(encodeTextHex), []JITValueDesc{d32}, 2)
					d33.NoHeapPointer = false
					ctx.BindReg(d33.Reg, &d33)
					ctx.BindReg(d33.Reg2, &d33)
					ctx.FreeDesc(&d32)
					ctx.SyncDesc(&d33)
					if d33.Loc == LocRegPair || d33.Loc == LocStackPair || d33.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d33, &result)
						result.Type = d33.Type
					} else {
						switch d33.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d33)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d33)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d33)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d33, &result)
							result.Type = d33.Type
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
					d34 = args[0]
					d34.ID = 0
					d36 = d34
					ctx.SyncDesc(&d36)
					if d36.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d36.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d36.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d36 = tmpScalar
					}
					d36 = JITPrepareScmerGoArg(ctx, d36)
					if d36.Loc != LocRegPair && d36.Loc != LocStackPair && d36.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d36}, 2)
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.FreeDesc(&d34)
					if d35.SliceSizeKnown {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d35.KnownSliceLen))}
					} else if d35.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d35.Imm.String())))}
					} else if d35.Loc == LocStackTriple {
						d37 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else if d35.Loc == LocStackPair {
						d37 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d35)
						if d35.Loc == LocRegPair || d35.Loc == LocRegTriple {
							d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg2, ID: 0}
						} else if d35.Loc == LocReg {
							d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d37
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d37)
					ctx.SyncDesc(&d38)
					ctx.SyncDesc(&d37)
					if d38.Loc == LocImm && d37.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() * d37.Imm.Int())}
					} else if d38.Loc == LocImm {
						ctx.EnsureDesc(&d37)
						scratch := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitMovRegReg(scratch, d37.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d38.Imm.Int())
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d39)
					} else if d37.Loc == LocImm {
						ctx.EnsureDesc(&d38)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d38.Reg, d37.Imm.Int())
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg}
						ctx.BindReg(d38.Reg, &d39)
					} else {
						ctx.EnsureDesc(&d38)
						ctx.SyncDesc(&d37)
						ctx.EmitIntBinary(JITIntMul, 64, d38.Reg, &d37)
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg}
						ctx.BindReg(d38.Reg, &d39)
					}
					if d39.Loc == LocReg && d38.Loc == LocReg && d39.Reg == d38.Reg {
						ctx.TransferReg(d38.Reg)
						d38.Loc = LocNone
					}
					ctx.FreeDesc(&d37)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					callResults40 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d39, d39}, []uint8{3}, []uint8{1})
					d41 = callResults40[0]
					d41.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d41)
					ctx.FreeDesc(&d39)
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
					if d35.SliceSizeKnown {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d35.KnownSliceLen))}
					} else if d35.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d35.Imm.String())))}
					} else if d35.Loc == LocStackTriple {
						d42 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else if d35.Loc == LocStackPair {
						d42 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d35)
						if d35.Loc == LocRegPair || d35.Loc == LocRegTriple {
							d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg2, ID: 0}
						} else if d35.Loc == LocReg {
							d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d42
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDescsTogether(&d1, &d42)
					if d1.Loc == LocImm && d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d42.Imm.Int())}
					} else if d42.Loc == LocImm {
						r4 := ctx.AllocRegExcept(d1.Reg)
						if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d42.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d42.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d43)
					} else if d1.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d42.Reg)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d43)
					} else {
						r6 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d42.Reg)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedLess}
						ctx.BindReg(r6, &d43)
					}
					ctx.FreeDesc(&d42)
					d44 = d43
					ctx.SyncDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d44.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d43)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap45 := d1
						snap46 := d2
						snap47 := d3
						snap48 := d6
						snap49 := d7
						snap50 := d8
						snap51 := d10
						snap52 := d11
						snap53 := d12
						snap54 := d13
						snap55 := d14
						snap56 := d16
						snap57 := d17
						snap58 := d32
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d37
						snap64 := d38
						snap65 := d39
						snap66 := d41
						snap67 := d42
						snap68 := d43
						snap69 := d44
						alloc70 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc70)
						d1 = snap45
						d2 = snap46
						d3 = snap47
						d6 = snap48
						d7 = snap49
						d8 = snap50
						d10 = snap51
						d11 = snap52
						d12 = snap53
						d13 = snap54
						d14 = snap55
						d16 = snap56
						d17 = snap57
						d32 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d37 = snap63
						d38 = snap64
						d39 = snap65
						d41 = snap66
						d42 = snap67
						d43 = snap68
						d44 = snap69
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
					ctx.StabilizeDescForControlFlow(&d41)
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d71)
					ctx.SyncDesc(&d1)
					if d71.Loc == LocImm && d1.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int() * d1.Imm.Int())}
					} else if d71.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d71.Imm.Int())
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d72)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d71)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d71.Reg, d1.Imm.Int())
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d72)
					} else {
						ctx.EnsureDesc(&d71)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d71.Reg, &d1)
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d72)
					}
					if d72.Loc == LocReg && d71.Loc == LocReg && d72.Reg == d71.Reg {
						ctx.TransferReg(d71.Reg)
						d71.Loc = LocNone
					}
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d35)
					d73 = ctx.EmitSliceElementAddress(&d35, &d1, 1)
					ctx.EnsureDesc(&d73)
					r7 := ctx.AllocRegExcept(d73.Reg)
					ctx.EmitMovRegMemB(r7, d73.Reg, 0)
					ctx.FreeDesc(&d73)
					d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d74)
					ctx.BindReg(r7, &d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d74.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d74.Reg, 4)
						d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d74.Reg}
						ctx.BindReg(d74.Reg, &d75)
					}
					if d75.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: d75.Type, Imm: NewInt(int64(uint64(d75.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d75.Reg, 56)
						ctx.EmitShrRegImm8(d75.Reg, 56)
					}
					if d75.Loc == LocReg && d74.Loc == LocReg && d75.Reg == d74.Reg {
						ctx.TransferReg(d74.Reg)
						d74.Loc = LocNone
					}
					ctx.FreeDesc(&d74)
					d76 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d75)
					ctx.EnsureGoStringHeader(&d76)
					d77 = ctx.EmitSliceElementAddress(&d76, &d75, 1)
					ctx.EnsureDesc(&d77)
					r8 := ctx.AllocRegExcept(d77.Reg)
					ctx.EmitMovRegMemB(r8, d77.Reg, 0)
					ctx.FreeDesc(&d77)
					d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8, NoHeapPointer: true}
					ctx.BindReg(r8, &d78)
					ctx.BindReg(r8, &d78)
					ctx.FreeDesc(&d75)
					ctx.EnsureDesc(&d72)
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d72)
					d79 = d41
					d79.ID = 0
					d80 = d72
					d80.ID = 0
					d81 = ctx.EmitSliceElementAddress(&d79, &d80, int32(1))
					ctx.EmitStoreScalarAt(&d81, &d78, 1)
					ctx.FreeDesc(&d81)
					ctx.FreeDesc(&d80)
					ctx.FreeDesc(&d72)
					ctx.FreeDesc(&d78)
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d82)
					ctx.SyncDesc(&d1)
					if d82.Loc == LocImm && d1.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d82.Imm.Int() * d1.Imm.Int())}
					} else if d82.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d82.Imm.Int())
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d83)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d82)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d82.Reg, d1.Imm.Int())
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d82.Reg}
						ctx.BindReg(d82.Reg, &d83)
					} else {
						ctx.EnsureDesc(&d82)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d82.Reg, &d1)
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d82.Reg}
						ctx.BindReg(d82.Reg, &d83)
					}
					if d83.Loc == LocReg && d82.Loc == LocReg && d83.Reg == d82.Reg {
						ctx.TransferReg(d82.Reg)
						d82.Loc = LocNone
					}
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d83)
					if d83.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d83.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d83.Reg)
						ctx.EmitMovRegReg(scratch, d83.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d84)
					}
					if d84.Loc == LocReg && d83.Loc == LocReg && d84.Reg == d83.Reg {
						ctx.TransferReg(d83.Reg)
						d83.Loc = LocNone
					}
					ctx.FreeDesc(&d83)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d35)
					d85 = ctx.EmitSliceElementAddress(&d35, &d1, 1)
					ctx.EnsureDesc(&d85)
					r9 := ctx.AllocRegExcept(d85.Reg)
					ctx.EmitMovRegMemB(r9, d85.Reg, 0)
					ctx.FreeDesc(&d85)
					d86 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9, NoHeapPointer: true}
					ctx.BindReg(r9, &d86)
					ctx.BindReg(r9, &d86)
					ctx.EnsureDesc(&d86)
					if d86.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d86.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d86.Reg, 15)
						d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg}
						ctx.BindReg(d86.Reg, &d87)
					}
					if d87.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: d87.Type, Imm: NewInt(int64(uint64(d87.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d87.Reg, 56)
						ctx.EmitShrRegImm8(d87.Reg, 56)
					}
					if d87.Loc == LocReg && d86.Loc == LocReg && d87.Reg == d86.Reg {
						ctx.TransferReg(d86.Reg)
						d86.Loc = LocNone
					}
					ctx.FreeDesc(&d86)
					d88 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d87)
					ctx.EnsureGoStringHeader(&d88)
					d89 = ctx.EmitSliceElementAddress(&d88, &d87, 1)
					ctx.EnsureDesc(&d89)
					r10 := ctx.AllocRegExcept(d89.Reg)
					ctx.EmitMovRegMemB(r10, d89.Reg, 0)
					ctx.FreeDesc(&d89)
					d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10, NoHeapPointer: true}
					ctx.BindReg(r10, &d90)
					ctx.BindReg(r10, &d90)
					ctx.FreeDesc(&d87)
					ctx.EnsureDesc(&d84)
					ctx.SyncDesc(&d90)
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d84)
					d91 = d41
					d91.ID = 0
					d92 = d84
					d92.ID = 0
					d93 = ctx.EmitSliceElementAddress(&d91, &d92, int32(1))
					ctx.EmitStoreScalarAt(&d93, &d90, 1)
					ctx.FreeDesc(&d93)
					ctx.FreeDesc(&d92)
					ctx.FreeDesc(&d84)
					ctx.FreeDesc(&d90)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d94)
					}
					if d94.Loc == LocReg && d1.Loc == LocReg && d94.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d94)
					ctx.EmitStoreToStack(d94, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d94)
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
					ctx.StabilizeDescForControlFlow(&d41)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d41)
					callResults96 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d41}, []uint8{2}, []uint8{1})
					d95 = callResults96[0]
					ctx.EnsureDesc(&d95)
					d97 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d95}, 2)
					ctx.EmitMovPairToResult(&d97, &result)
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
						d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
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
					ctx.EnsureDesc(&d1)
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
						boxed := JITPrepareScmerGoArg(ctx, d1)
						d2 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
						d30 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondNotEqual}
						ctx.BindReg(r2, &d30)
					}
					d31 = d30
					ctx.SyncDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d31.Condition, lbl6)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d30)
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
				var d5 JITValueDesc
				_ = d5
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
				var phiBase24 int32
				_ = phiBase24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d28 JITValueDesc
				_ = d28
				var inlineResultOff27 int32
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
				var d39 JITValueDesc
				_ = d39
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var phiBase45 int32
				_ = phiBase45
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
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
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
					d1 = d0
					_ = d1
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d2 = d1
					d2.ID = 0
					d3 = ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
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
					ctx.FreeDesc(&d0)
					d5 = d4
					ctx.SyncDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d5.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d4)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap6 := d0
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						alloc12 := ctx.SnapshotAllocState()
						bbs[3].Render()
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
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(func() *schemeTextWriter { return new(schemeTextWriter) }), nil, 1)
					ctx.BindReg(d13.Reg, &d13)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d14)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *schemeTextWriter, value uint64) { base.hash = value }), []JITValueDesc{d13, d14})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					d15 = args[0]
					d15.ID = 0
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d15 = JITPrepareScmerGoArg(ctx, d15)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d15)
					ctx.EmitGoCallVoid(GoFuncAddr((*schemeTextWriter).writeCompressedText), []JITValueDesc{d13, d15})
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r1 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r1, fieldAddr)
						d16 = JITValueDesc{Loc: LocReg, Reg: r1}
						ctx.BindReg(r1, &d16)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r2 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r2, baseReg, off)
						d16 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d16)
					}
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d16}, 2)
					d17.NoHeapPointer = false
					ctx.BindReg(d17.Reg, &d17)
					ctx.BindReg(d17.Reg2, &d17)
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d17}, 2)
					ctx.EmitMovPairToResult(&d18, &result)
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
					d22 = d20
					_ = d22
					ctx.StabilizeDescForControlFlow(&d22)
					inlineHome23 := d22
					inlineHome23.ID = 0
					phiBase24 = ctx.AllocStack(int32(32))
					d25 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(0)}
					_ = d25
					d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(16)}
					_ = d26
					inlineResultOff27 = ctx.AllocStack(int32(16))
					d28 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inlineResultOff27}
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
					d22 = inlineHome23
					d25 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(0)}
					d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}, int32(phiBase24)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase24)+int32(16))
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d22 = inlineHome23
					d25 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(0)}
					d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d26)
					ctx.ReclaimUntrackedRegs()
					if d22.SliceSizeKnown {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d22.KnownSliceLen))}
					} else if d22.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d22.Imm.String())))}
					} else if d22.Loc == LocStackTriple {
						d29 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d22.StackOff + 8, NoHeapPointer: true}
					} else if d22.Loc == LocStackPair {
						d29 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d22.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d22)
						if d22.Loc == LocRegPair || d22.Loc == LocRegTriple {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg2, ID: 0}
						} else if d22.Loc == LocReg {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d29
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDescsTogether(&d26, &d29)
					if d26.Loc == LocImm && d29.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d26.Imm.Int() < d29.Imm.Int())}
					} else if d29.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d26.Reg)
						if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d26.Reg, int32(d29.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d29.Imm.Int()))
							ctx.EmitCmpInt64(d26.Reg, ctx.ScratchReg)
						}
						d30 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d30)
					} else if d26.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d26.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d29.Reg)
						d30 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d30)
					} else {
						r5 := ctx.AllocRegExcept(d26.Reg)
						ctx.EmitCmpInt64(d26.Reg, d29.Reg)
						d30 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d30)
					}
					ctx.FreeDesc(&d29)
					ctx.ReclaimUntrackedRegs()
					d31 = d30
					ctx.SyncDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						}
					} else {
						ctx.EmitJump(d31.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d30)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
					}
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d22 = inlineHome23
					d25 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(0)}
					d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d25.Loc == LocRegPair || d25.Loc == LocStackPair || d25.Loc == LocRegTriple || d25.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d25)
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d25}, 2)
					d32.NoHeapPointer = false
					ctx.BindReg(d32.Reg, &d32)
					ctx.BindReg(d32.Reg2, &d32)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EmitCopyDescWords(&d28, &d32, 2)
					ctx.EmitJmp(lbl8)
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d22 = inlineHome23
					d25 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(0)}
					d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase24) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d26)
					ctx.EnsureGoStringHeader(&d22)
					d33 = ctx.EmitSliceElementAddress(&d22, &d26, 1)
					ctx.EnsureDesc(&d33)
					r6 := ctx.AllocRegExcept(d33.Reg)
					ctx.EmitMovRegMemB(r6, d33.Reg, 0)
					ctx.FreeDesc(&d33)
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d34)
					ctx.BindReg(r6, &d34)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d34.Imm.Int()))))}
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegReg(r7, d34.Reg)
						ctx.EmitShlRegImm8(r7, 56)
						ctx.EmitShrRegImm8(r7, 56)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d35)
					}
					ctx.FreeDesc(&d34)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDescsTogether(&d25, &d35)
					if d25.Loc == LocImm && d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d25.Imm.Int() ^ d35.Imm.Int())}
					} else {
						xorResult := ctx.AllocRegExcept(d25.Reg, d35.Reg)
						ctx.EmitMovToReg(xorResult, d25)
						if d35.Loc == LocReg {
							ctx.EmitXorInt64(xorResult, d35.Reg)
						} else {
							xorOther := ctx.AllocRegExcept(xorResult)
							ctx.EmitMovToReg(xorOther, d35)
							ctx.EmitXorInt64(xorResult, xorOther)
							ctx.FreeReg(xorOther)
						}
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: xorResult}
						ctx.BindReg(xorResult, &d36)
					}
					ctx.FreeDesc(&d35)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d36.Imm.Int() * 1099511628211)}
					} else {
						ctx.EmitIntBinaryImm(JITIntMul, 64, d36.Reg, 1099511628211)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg}
						ctx.BindReg(d36.Reg, &d37)
					}
					if d37.Loc == LocReg && d36.Loc == LocReg && d37.Reg == d36.Reg {
						ctx.TransferReg(d36.Reg)
						d36.Loc = LocNone
					}
					ctx.EnsureDesc(&d37)
					ctx.EmitStoreToStack(d37, int32(phiBase24)+int32(0))
					ctx.StabilizeDescForControlFlow(&d37)
					inlineHome38 := d37
					inlineHome38.ID = 0
					ctx.FreeDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d26.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d26.Reg)
						ctx.EmitMovRegReg(scratch, d26.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d39)
					}
					if d39.Loc == LocReg && d26.Loc == LocReg && d39.Reg == d26.Reg {
						ctx.TransferReg(d26.Reg)
						d26.Loc = LocNone
					}
					ctx.EnsureDesc(&d39)
					ctx.EmitStoreToStack(d39, int32(phiBase24)+int32(16))
					ctx.StabilizeDescForControlFlow(&d39)
					inlineHome40 := d39
					inlineHome40.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl8)
					ctx.EnsureDesc(&d28)
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d28}, 2)
					ctx.EmitMovPairToResult(&d41, &result)
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
					d42 = args[0]
					d42.ID = 0
					ctx.EnsureDesc(&d42)
					d43 = d42
					_ = d43
					ctx.StabilizeDescForControlFlow(&d43)
					inlineHome44 := d43
					inlineHome44.ID = 0
					phiBase45 = ctx.AllocStack(int32(16))
					d46 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase45) + int32(0)}
					_ = d46
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
					d43 = inlineHome44
					d46 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase45) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d47 = d43
					d47.ID = 0
					d48 = ctx.EmitGetTagDesc(&d47, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d48)
					inlineHome49 := d48
					inlineHome49.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						d50 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d48.Imm.Int()) == uint64(0x13))}
					} else {
						r8 := ctx.AllocRegExcept(d48.Reg)
						ctx.EmitCmpRegImm32(d48.Reg, 19)
						d50 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondEqual}
						ctx.BindReg(r8, &d50)
					}
					ctx.ReclaimUntrackedRegs()
					d51 = d50
					ctx.SyncDesc(&d51)
					if d51.Loc != LocImm && d51.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					if d51.Loc == LocImm {
						if d51.Imm.Bool() {
							ctx.MarkLabel(lbl20)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase45)+int32(0))
							ctx.EmitJmp(lbl18)
						} else {
							ctx.MarkLabel(lbl21)
							ctx.EmitJmp(lbl19)
						}
					} else {
						ctx.EmitJump(d51.Condition, lbl20)
						ctx.EmitJmp(lbl21)
						ctx.FreeDesc(&d50)
						ctx.MarkLabel(lbl20)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase45)+int32(0))
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
					}
					bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					d43 = inlineHome44
					d48 = inlineHome49
					d46 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase45) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r9 := ctx.AllocReg()
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r9, d46)
					}
					ctx.EmitJmp(lbl15)
					bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d43 = inlineHome44
					d48 = inlineHome49
					d46 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase45) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d48.Imm.Int()) == uint64(0x14))}
					} else {
						r10 := ctx.AllocRegExcept(d48.Reg)
						ctx.EmitCmpRegImm32(d48.Reg, 20)
						d52 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondEqual}
						ctx.BindReg(r10, &d52)
					}
					ctx.ReclaimUntrackedRegs()
					d53 = d52
					ctx.SyncDesc(&d53)
					if d53.Loc != LocImm && d53.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					if d53.Loc == LocImm {
						if d53.Imm.Bool() {
							ctx.MarkLabel(lbl22)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase45)+int32(0))
							ctx.EmitJmp(lbl18)
						} else {
							ctx.MarkLabel(lbl23)
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitJump(d53.Condition, lbl22)
						ctx.EmitJmp(lbl23)
						ctx.FreeDesc(&d52)
						ctx.MarkLabel(lbl22)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase45)+int32(0))
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl17)
					}
					bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d43 = inlineHome44
					d48 = inlineHome49
					d46 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase45) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d48.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d48.Reg, 26)
						r11 := ctx.AllocRegExcept(d48.Reg)
						ctx.EmitSetcc(r11, CondEqual)
						d54 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d54)
					}
					ctx.EnsureDesc(&d54)
					ctx.EmitStoreToStack(d54, int32(phiBase45)+int32(0))
					ctx.StabilizeDescForControlFlow(&d54)
					inlineHome55 := d54
					inlineHome55.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl15)
					d56 = JITValueDesc{Loc: LocReg, Reg: r9}
					ctx.BindReg(r9, &d56)
					ctx.BindReg(r9, &d56)
					ctx.FreeDesc(&d42)
					d57 = d56
					ctx.EnsureDesc(&d57)
					if d57.Loc != LocImm && d57.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d57.Loc == LocImm {
						if d57.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d57.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap58 := d0
						snap59 := d1
						snap60 := d2
						snap61 := d3
						snap62 := d4
						snap63 := d5
						snap64 := d13
						snap65 := d14
						snap66 := d15
						snap67 := d16
						snap68 := d17
						snap69 := d18
						snap70 := d19
						snap71 := d20
						snap72 := d21
						snap73 := d22
						snap74 := d25
						snap75 := d26
						snap76 := d28
						snap77 := d29
						snap78 := d30
						snap79 := d31
						snap80 := d32
						snap81 := d33
						snap82 := d34
						snap83 := d35
						snap84 := d36
						snap85 := d37
						snap86 := d39
						snap87 := d41
						snap88 := d42
						snap89 := d43
						snap90 := d46
						snap91 := d47
						snap92 := d48
						snap93 := d50
						snap94 := d51
						snap95 := d52
						snap96 := d53
						snap97 := d54
						snap98 := d56
						snap99 := d57
						alloc100 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc100)
						d0 = snap58
						d1 = snap59
						d2 = snap60
						d3 = snap61
						d4 = snap62
						d5 = snap63
						d13 = snap64
						d14 = snap65
						d15 = snap66
						d16 = snap67
						d17 = snap68
						d18 = snap69
						d19 = snap70
						d20 = snap71
						d21 = snap72
						d22 = snap73
						d25 = snap74
						d26 = snap75
						d28 = snap76
						d29 = snap77
						d30 = snap78
						d31 = snap79
						d32 = snap80
						d33 = snap81
						d34 = snap82
						d35 = snap83
						d36 = snap84
						d37 = snap85
						d39 = snap86
						d41 = snap87
						d42 = snap88
						d43 = snap89
						d46 = snap90
						d47 = snap91
						d48 = snap92
						d50 = snap93
						d51 = snap94
						d52 = snap95
						d53 = snap96
						d54 = snap97
						d56 = snap98
						d57 = snap99
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d56)
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
					d101 = args[0]
					d101.ID = 0
					d101 = JITPrepareScmerGoArg(ctx, d101)
					ctx.SyncDesc(&d101)
					d102 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d101}, 1)
					d102.NoHeapPointer = true
					ctx.BindReg(d102.Reg, &d102)
					ctx.FreeDesc(&d101)
					ctx.EnsureDesc(&d102)
					if d102.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d102.Imm.Int() >= 128)}
					} else {
						r12 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d102.Reg, 128)
						d103 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r12, &d103)
					}
					ctx.FreeDesc(&d102)
					d104 = d103
					ctx.SyncDesc(&d104)
					if d104.Loc != LocImm && d104.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d104.Loc == LocImm {
						if d104.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d104.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d103)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap105 := d0
						snap106 := d1
						snap107 := d2
						snap108 := d3
						snap109 := d4
						snap110 := d5
						snap111 := d13
						snap112 := d14
						snap113 := d15
						snap114 := d16
						snap115 := d17
						snap116 := d18
						snap117 := d19
						snap118 := d20
						snap119 := d21
						snap120 := d22
						snap121 := d25
						snap122 := d26
						snap123 := d28
						snap124 := d29
						snap125 := d30
						snap126 := d31
						snap127 := d32
						snap128 := d33
						snap129 := d34
						snap130 := d35
						snap131 := d36
						snap132 := d37
						snap133 := d39
						snap134 := d41
						snap135 := d42
						snap136 := d43
						snap137 := d46
						snap138 := d47
						snap139 := d48
						snap140 := d50
						snap141 := d51
						snap142 := d52
						snap143 := d53
						snap144 := d54
						snap145 := d56
						snap146 := d57
						snap147 := d101
						snap148 := d102
						snap149 := d103
						snap150 := d104
						alloc151 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc151)
						d0 = snap105
						d1 = snap106
						d2 = snap107
						d3 = snap108
						d4 = snap109
						d5 = snap110
						d13 = snap111
						d14 = snap112
						d15 = snap113
						d16 = snap114
						d17 = snap115
						d18 = snap116
						d19 = snap117
						d20 = snap118
						d21 = snap119
						d22 = snap120
						d25 = snap121
						d26 = snap122
						d28 = snap123
						d29 = snap124
						d30 = snap125
						d31 = snap126
						d32 = snap127
						d33 = snap128
						d34 = snap129
						d35 = snap130
						d36 = snap131
						d37 = snap132
						d39 = snap133
						d41 = snap134
						d42 = snap135
						d43 = snap136
						d46 = snap137
						d47 = snap138
						d48 = snap139
						d50 = snap140
						d51 = snap141
						d52 = snap142
						d53 = snap143
						d54 = snap144
						d56 = snap145
						d57 = snap146
						d101 = snap147
						d102 = snap148
						d103 = snap149
						d104 = snap150
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
					_ = d0
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
					_ = d9
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
					_ = d21
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
						fieldAddr := uintptr(d7.Imm.Int()) + 32
						r3 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r3, fieldAddr)
						d40 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d40)
					} else {
						off := int32(32)
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
				var d5 JITValueDesc
				_ = d5
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
				var d22 JITValueDesc
				_ = d22
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
				var phiBase30 int32
				_ = phiBase30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
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
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
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
					d1 = d0
					_ = d1
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d2 = d1
					d2.ID = 0
					d3 = ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
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
					ctx.FreeDesc(&d0)
					d5 = d4
					ctx.SyncDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d5.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d4)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap6 := d0
						snap7 := d1
						snap8 := d2
						snap9 := d3
						snap10 := d4
						snap11 := d5
						alloc12 := ctx.SnapshotAllocState()
						bbs[3].Render()
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
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextDigest), []JITValueDesc{d13, d14}, 2)
					d15.NoHeapPointer = false
					ctx.BindReg(d15.Reg, &d15)
					ctx.BindReg(d15.Reg2, &d15)
					ctx.FreeDesc(&d14)
					ctx.FreeDesc(&d13)
					ctx.SyncDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d15, &result)
						result.Type = d15.Type
					} else {
						switch d15.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d15)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d15)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d15)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d15, &result)
							result.Type = d15.Type
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
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[32]byte { return new([32]byte) }), nil, 1)
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
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					callResults21 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d18}, []uint8{3}, []uint8{1})
					d20 = callResults21[0]
					d20.Type = tagSlice
					d20 = JITPrepareGoSliceArg(ctx, d20)
					if d20.Loc != LocRegTriple && d20.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (sha256.Sum256 arg0)")
					}
					ctx.SyncDesc(&d20)
					d22 = ctx.EmitGoCallScalar(GoFuncAddr((func(arg0 []byte) *[32]byte { value := sha256.Sum256(arg0); return &value })), []JITValueDesc{d20}, 1)
					d22.NoHeapPointer = false
					ctx.BindReg(d22.Reg, &d22)
					ctx.EnsureDesc(&d22)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *[32]byte) { *dst = *src }), []JITValueDesc{d16, d22})
					sliceResults23 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[32]byte) []byte { return value[0:32:32] }), []JITValueDesc{d16}, []uint8{3}, []uint8{1})
					d24 = sliceResults23[0]
					_ = d24
					d24 = JITPrepareGoSliceArg(ctx, d24)
					if d24.Loc != LocRegTriple && d24.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (hex.EncodeToString arg0)")
					}
					ctx.SyncDesc(&d24)
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(hex.EncodeToString), []JITValueDesc{d24}, 2)
					d25.NoHeapPointer = false
					ctx.BindReg(d25.Reg, &d25)
					ctx.BindReg(d25.Reg2, &d25)
					ctx.EnsureDesc(&d25)
					d26 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d25}, 2)
					ctx.EmitMovPairToResult(&d26, &result)
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
					d27 = args[0]
					d27.ID = 0
					ctx.EnsureDesc(&d27)
					d28 = d27
					_ = d28
					ctx.StabilizeDescForControlFlow(&d28)
					inlineHome29 := d28
					inlineHome29.ID = 0
					phiBase30 = ctx.AllocStack(int32(16))
					d31 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase30) + int32(0)}
					_ = d31
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
					d28 = inlineHome29
					d31 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase30) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d32 = d28
					d32.ID = 0
					d33 = ctx.EmitGetTagDesc(&d32, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d33)
					inlineHome34 := d33
					inlineHome34.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d33.Imm.Int()) == uint64(0x13))}
					} else {
						r1 := ctx.AllocRegExcept(d33.Reg)
						ctx.EmitCmpRegImm32(d33.Reg, 19)
						d35 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d35)
					}
					ctx.ReclaimUntrackedRegs()
					d36 = d35
					ctx.SyncDesc(&d36)
					if d36.Loc != LocImm && d36.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d36.Loc == LocImm {
						if d36.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase30)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d36.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d35)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase30)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d28 = inlineHome29
					d33 = inlineHome34
					d31 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase30) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r2 := ctx.AllocReg()
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d31)
					}
					ctx.EmitJmp(lbl7)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d28 = inlineHome29
					d33 = inlineHome34
					d31 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase30) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d33.Imm.Int()) == uint64(0x14))}
					} else {
						r3 := ctx.AllocRegExcept(d33.Reg)
						ctx.EmitCmpRegImm32(d33.Reg, 20)
						d37 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d37)
					}
					ctx.ReclaimUntrackedRegs()
					d38 = d37
					ctx.SyncDesc(&d38)
					if d38.Loc != LocImm && d38.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d38.Loc == LocImm {
						if d38.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase30)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d38.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d37)
						ctx.MarkLabel(lbl14)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase30)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl9)
					}
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d28 = inlineHome29
					d33 = inlineHome34
					d31 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase30) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d33.Imm.Int()) == uint64(0x1a))}
					} else {
						ctx.EmitCmpRegImm32(d33.Reg, 26)
						r4 := ctx.AllocRegExcept(d33.Reg)
						ctx.EmitSetcc(r4, CondEqual)
						d39 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d39)
					}
					ctx.EnsureDesc(&d39)
					ctx.EmitStoreToStack(d39, int32(phiBase30)+int32(0))
					ctx.StabilizeDescForControlFlow(&d39)
					inlineHome40 := d39
					inlineHome40.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d41 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d41)
					ctx.BindReg(r2, &d41)
					ctx.FreeDesc(&d27)
					d42 = d41
					ctx.EnsureDesc(&d42)
					if d42.Loc != LocImm && d42.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d42.Loc == LocImm {
						if d42.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d42.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap43 := d0
						snap44 := d1
						snap45 := d2
						snap46 := d3
						snap47 := d4
						snap48 := d5
						snap49 := d13
						snap50 := d14
						snap51 := d15
						snap52 := d16
						snap53 := d17
						snap54 := d18
						snap55 := d19
						snap56 := d20
						snap57 := d22
						snap58 := d24
						snap59 := d25
						snap60 := d26
						snap61 := d27
						snap62 := d28
						snap63 := d31
						snap64 := d32
						snap65 := d33
						snap66 := d35
						snap67 := d36
						snap68 := d37
						snap69 := d38
						snap70 := d39
						snap71 := d41
						snap72 := d42
						alloc73 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc73)
						d0 = snap43
						d1 = snap44
						d2 = snap45
						d3 = snap46
						d4 = snap47
						d5 = snap48
						d13 = snap49
						d14 = snap50
						d15 = snap51
						d16 = snap52
						d17 = snap53
						d18 = snap54
						d19 = snap55
						d20 = snap56
						d22 = snap57
						d24 = snap58
						d25 = snap59
						d26 = snap60
						d27 = snap61
						d28 = snap62
						d31 = snap63
						d32 = snap64
						d33 = snap65
						d35 = snap66
						d36 = snap67
						d37 = snap68
						d38 = snap69
						d39 = snap70
						d41 = snap71
						d42 = snap72
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d41)
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
					d74 = args[0]
					d74.ID = 0
					d74 = JITPrepareScmerGoArg(ctx, d74)
					ctx.SyncDesc(&d74)
					d75 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d74}, 1)
					d75.NoHeapPointer = true
					ctx.BindReg(d75.Reg, &d75)
					ctx.FreeDesc(&d74)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d75.Imm.Int() >= 256)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d75.Reg, 256)
						d76 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r5, &d76)
					}
					ctx.FreeDesc(&d75)
					d77 = d76
					ctx.SyncDesc(&d77)
					if d77.Loc != LocImm && d77.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d77.Loc == LocImm {
						if d77.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d77.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d76)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap78 := d0
						snap79 := d1
						snap80 := d2
						snap81 := d3
						snap82 := d4
						snap83 := d5
						snap84 := d13
						snap85 := d14
						snap86 := d15
						snap87 := d16
						snap88 := d17
						snap89 := d18
						snap90 := d19
						snap91 := d20
						snap92 := d22
						snap93 := d24
						snap94 := d25
						snap95 := d26
						snap96 := d27
						snap97 := d28
						snap98 := d31
						snap99 := d32
						snap100 := d33
						snap101 := d35
						snap102 := d36
						snap103 := d37
						snap104 := d38
						snap105 := d39
						snap106 := d41
						snap107 := d42
						snap108 := d74
						snap109 := d75
						snap110 := d76
						snap111 := d77
						alloc112 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc112)
						d0 = snap78
						d1 = snap79
						d2 = snap80
						d3 = snap81
						d4 = snap82
						d5 = snap83
						d13 = snap84
						d14 = snap85
						d15 = snap86
						d16 = snap87
						d17 = snap88
						d18 = snap89
						d19 = snap90
						d20 = snap91
						d22 = snap92
						d24 = snap93
						d25 = snap94
						d26 = snap95
						d27 = snap96
						d28 = snap97
						d31 = snap98
						d32 = snap99
						d33 = snap100
						d35 = snap101
						d36 = snap102
						d37 = snap103
						d38 = snap104
						d39 = snap105
						d41 = snap106
						d42 = snap107
						d74 = snap108
						d75 = snap109
						d76 = snap110
						d77 = snap111
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
						d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d16)
					}
					d17 = d16
					ctx.SyncDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d17.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d16)
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
			JITInlineCallbacks: false,
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
					ctx.EnsureDesc(&d42)
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
						boxed := JITPrepareScmerGoArg(ctx, d42)
						d43 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
					ctx.EnsureDesc(&d69)
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
						boxed := JITPrepareScmerGoArg(ctx, d69)
						d70 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
