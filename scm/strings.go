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

func init_strings() {
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
					ctx.EnsureDesc(&d30)
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
					ctx.EnsureDesc(&d4)
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
				var d61 JITValueDesc
				_ = d61
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
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
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
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase8)+int32(0))
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl13)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase8)+int32(0))
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl11)
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
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d9 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase8) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d11.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d11.Reg, 20)
						r2 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d14)
					}
					ctx.EnsureDesc(&d14)
					ctx.EmitStoreToStack(d14, int32(phiBase8)+int32(0))
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl9)
					d15 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d15)
					ctx.BindReg(r1, &d15)
					ctx.FreeDesc(&d6)
					d16 = d15
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap17 := d1
						snap18 := d2
						snap19 := d3
						snap20 := d4
						snap21 := d5
						snap22 := d6
						snap23 := d7
						snap24 := d9
						snap25 := d10
						snap26 := d11
						snap27 := d12
						snap28 := d13
						snap29 := d14
						snap30 := d15
						snap31 := d16
						alloc32 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc32)
						d1 = snap17
						d2 = snap18
						d3 = snap19
						d4 = snap20
						d5 = snap21
						d6 = snap22
						d7 = snap23
						d9 = snap24
						d10 = snap25
						d11 = snap26
						d12 = snap27
						d13 = snap28
						d14 = snap29
						d15 = snap30
						d16 = snap31
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d15)
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
					d33 = args[0]
					d33.ID = 0
					d33 = JITPrepareScmerGoArg(ctx, d33)
					ctx.SyncDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d33}, 1)
					d34.NoHeapPointer = true
					ctx.BindReg(d34.Reg, &d34)
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.FreeDesc(&d33)
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d35.Imm.Int() > 2)}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d35.Reg, 2)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedGreater}
						ctx.BindReg(r3, &d36)
					}
					ctx.FreeDesc(&d35)
					d37 = d36
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocImm && d37.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d37.Loc == LocImm {
						if d37.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.SyncDesc(&d34)
						if d34.Loc == LocReg || d34.Loc == LocFPReg {
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
						if d34.Loc == LocReg || d34.Loc == LocFPReg {
							ctx.UnprotectReg(d34.Reg)
						} else if d34.Loc == LocRegPair {
							ctx.UnprotectReg(d34.Reg)
							ctx.UnprotectReg(d34.Reg2)
						}
						return bbs[4].Render()
					}
					lbl15 := ctx.ReserveLabel()
					ctx.EmitJump(d37.Condition, lbl4)
					ctx.EmitJmp(lbl15)
					ctx.FreeDesc(&d36)
					snap39 := d1
					snap40 := d2
					snap41 := d3
					snap42 := d4
					snap43 := d5
					snap44 := d6
					snap45 := d7
					snap46 := d9
					snap47 := d10
					snap48 := d11
					snap49 := d12
					snap50 := d13
					snap51 := d14
					snap52 := d15
					snap53 := d16
					snap54 := d33
					snap55 := d34
					snap56 := d35
					snap57 := d36
					snap58 := d37
					snap59 := d38
					alloc60 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl15)
					ctx.SyncDesc(&d34)
					if d34.Loc == LocReg || d34.Loc == LocFPReg {
						ctx.ProtectReg(d34.Reg)
					} else if d34.Loc == LocRegPair {
						ctx.ProtectReg(d34.Reg)
						ctx.ProtectReg(d34.Reg2)
					}
					d61 = d34
					if d61.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreToStack(d61, int32(bbs[4].PhiBase)+int32(0))
					if d34.Loc == LocReg || d34.Loc == LocFPReg {
						ctx.UnprotectReg(d34.Reg)
					} else if d34.Loc == LocRegPair {
						ctx.UnprotectReg(d34.Reg)
						ctx.UnprotectReg(d34.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc60)
					d1 = snap39
					d2 = snap40
					d3 = snap41
					d4 = snap42
					d5 = snap43
					d6 = snap44
					d7 = snap45
					d9 = snap46
					d10 = snap47
					d11 = snap48
					d12 = snap49
					d13 = snap50
					d14 = snap51
					d15 = snap52
					d16 = snap53
					d33 = snap54
					d34 = snap55
					d35 = snap56
					d36 = snap57
					d37 = snap58
					d38 = snap59
					if !bbs[4].Rendered {
						snap62 := d1
						snap63 := d2
						snap64 := d3
						snap65 := d4
						snap66 := d5
						snap67 := d6
						snap68 := d7
						snap69 := d9
						snap70 := d10
						snap71 := d11
						snap72 := d12
						snap73 := d13
						snap74 := d14
						snap75 := d15
						snap76 := d16
						snap77 := d33
						snap78 := d34
						snap79 := d35
						snap80 := d36
						snap81 := d37
						snap82 := d38
						snap83 := d61
						alloc84 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc84)
						d1 = snap62
						d2 = snap63
						d3 = snap64
						d4 = snap65
						d5 = snap66
						d6 = snap67
						d7 = snap68
						d9 = snap69
						d10 = snap70
						d11 = snap71
						d12 = snap72
						d13 = snap73
						d14 = snap74
						d15 = snap75
						d16 = snap76
						d33 = snap77
						d34 = snap78
						d35 = snap79
						d36 = snap80
						d37 = snap81
						d38 = snap82
						d61 = snap83
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
					d85 = args[0]
					d85.ID = 0
					d87 = d85
					ctx.SyncDesc(&d87)
					if d87.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d87.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d87.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d87 = tmpScalar
					}
					d87 = JITPrepareScmerGoArg(ctx, d87)
					if d87.Loc != LocRegPair && d87.Loc != LocStackPair && d87.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d86 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d87}, 2)
					ctx.StabilizeDescForControlFlow(&d86)
					ctx.FreeDesc(&d85)
					d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						d89 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d88.Imm.Int() > 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d88.Reg, 2)
						d89 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedGreater}
						ctx.BindReg(r4, &d89)
					}
					ctx.FreeDesc(&d88)
					d90 = d89
					ctx.EnsureDesc(&d90)
					if d90.Loc != LocImm && d90.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d90.Loc == LocImm {
						if d90.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d90.Condition, lbl6)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d89)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap91 := d1
						snap92 := d2
						snap93 := d3
						snap94 := d4
						snap95 := d5
						snap96 := d6
						snap97 := d7
						snap98 := d9
						snap99 := d10
						snap100 := d11
						snap101 := d12
						snap102 := d13
						snap103 := d14
						snap104 := d15
						snap105 := d16
						snap106 := d33
						snap107 := d34
						snap108 := d35
						snap109 := d36
						snap110 := d37
						snap111 := d38
						snap112 := d61
						snap113 := d85
						snap114 := d86
						snap115 := d87
						snap116 := d88
						snap117 := d89
						snap118 := d90
						alloc119 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc119)
						d1 = snap91
						d2 = snap92
						d3 = snap93
						d4 = snap94
						d5 = snap95
						d6 = snap96
						d7 = snap97
						d9 = snap98
						d10 = snap99
						d11 = snap100
						d12 = snap101
						d13 = snap102
						d14 = snap103
						d15 = snap104
						d16 = snap105
						d33 = snap106
						d34 = snap107
						d35 = snap108
						d36 = snap109
						d37 = snap110
						d38 = snap111
						d61 = snap112
						d85 = snap113
						d86 = snap114
						d87 = snap115
						d88 = snap116
						d89 = snap117
						d90 = snap118
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
					d120 = args[2]
					d120.ID = 0
					ctx.EnsureDesc(&d120)
					d121 = d120
					_ = d121
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d121.Loc == LocImm {
						d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d121.Imm.Int())}
					} else if d121.Type == tagInt && d121.Loc == LocRegPair {
						ctx.FreeReg(d121.Reg)
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d121.Reg2}
						ctx.BindReg(d121.Reg2, &d122)
						ctx.BindReg(d121.Reg2, &d122)
					} else if d121.Type == tagInt && d121.Loc == LocReg {
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d121.Reg}
						ctx.BindReg(d121.Reg, &d122)
						ctx.BindReg(d121.Reg, &d122)
					} else {
						d122 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d121}, 1)
						d122.Type = tagInt
						ctx.BindReg(d122.Reg, &d122)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d122)
					ctx.EnsureDesc(&d122)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d122)
					ctx.FreeDesc(&d120)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d122)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d122)
					if d4.Loc == LocImm && d122.Loc == LocImm {
						d124 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d122.Imm.Int())}
					} else if d122.Loc == LocImm && d122.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						r5 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(r5, d4.Reg)
						d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d124)
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d122)
						d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d122.Reg}
						ctx.BindReg(d122.Reg, &d124)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d122)
						scratch := ctx.AllocRegExcept(d122.Reg)
						ctx.EmitMovRegReg(scratch, d122.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d124)
					} else if d122.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d122.Imm.Int())
						d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d124)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d122)
						r6 := ctx.AllocRegExceptOperand(&d122, d4.Reg)
						ctx.EmitMovRegReg(r6, d4.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r6, &d122)
						d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d124)
					}
					if d124.Loc == LocReg && d4.Loc == LocReg && d124.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d124)
					ctx.EmitStoreToStack(d124, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d124)
					ctx.FreeDesc(&d122)
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
					d125 = args[0]
					d125.ID = 0
					d125 = JITPrepareScmerGoArg(ctx, d125)
					if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocRegTriple || d4.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d125)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d1)
					d126 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d125, d4, d1}, 2)
					d126.NoHeapPointer = false
					ctx.BindReg(d126.Reg, &d126)
					ctx.BindReg(d126.Reg2, &d126)
					ctx.FreeDesc(&d125)
					ctx.FreeDesc(&d1)
					ctx.SyncDesc(&d126)
					if d126.Loc == LocRegPair || d126.Loc == LocStackPair || d126.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d126, &result)
						result.Type = d126.Type
					} else {
						switch d126.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d126)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d126)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d126)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d126, &result)
							result.Type = d126.Type
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
					d127 = args[2]
					d127.ID = 0
					ctx.EnsureDesc(&d127)
					d128 = d127
					_ = d128
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d128.Loc == LocImm {
						d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d128.Imm.Int())}
					} else if d128.Type == tagInt && d128.Loc == LocRegPair {
						ctx.FreeReg(d128.Reg)
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d128.Reg2}
						ctx.BindReg(d128.Reg2, &d129)
						ctx.BindReg(d128.Reg2, &d129)
					} else if d128.Type == tagInt && d128.Loc == LocReg {
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d128.Reg}
						ctx.BindReg(d128.Reg, &d129)
						ctx.BindReg(d128.Reg, &d129)
					} else {
						d129 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d128}, 1)
						d129.Type = tagInt
						ctx.BindReg(d129.Reg, &d129)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d129)
					ctx.EnsureDesc(&d129)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d129)
					ctx.FreeDesc(&d127)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d129)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d129)
					if d4.Loc == LocImm && d129.Loc == LocImm {
						d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d129.Imm.Int())}
					} else if d129.Loc == LocImm && d129.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						r7 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(r7, d4.Reg)
						d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d131)
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d129)
						d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d129.Reg}
						ctx.BindReg(d129.Reg, &d131)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d129)
						scratch := ctx.AllocRegExcept(d129.Reg)
						ctx.EmitMovRegReg(scratch, d129.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d131)
					} else if d129.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d129.Imm.Int())
						d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d131)
					} else {
						ctx.EnsureDesc(&d4)
						ctx.SyncDesc(&d129)
						r8 := ctx.AllocRegExceptOperand(&d129, d4.Reg)
						ctx.EmitMovRegReg(r8, d4.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r8, &d129)
						d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d131)
					}
					if d131.Loc == LocReg && d4.Loc == LocReg && d131.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.FreeDesc(&d129)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d131)
					ctx.EnsureDesc(&d86)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d131)
					if d131.Loc == LocImm && d4.Loc == LocImm {
						d133 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d131.Imm.Int() - d4.Imm.Int())}
					} else {
						r9 := ctx.AllocReg()
						if d131.Loc == LocImm {
							ctx.EmitMovRegImm64(r9, uint64(d131.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r9, d131.Reg)
						}
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitSubInt64(r9, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r9, d4.Reg)
						}
						d133 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d133)
					}
					r10 := ctx.EmitSliceDataAfterLow(&d86, &d4, 1)
					d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
					ctx.BindReg(r10, &d134)
					ctx.BindReg(r10, &d134)
					var r11 Reg
					var r12 Reg
					ctx.SyncDesc(&d134)
					ctx.EnsureDesc(&d134)
					if d134.Loc == LocImm {
						r11 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r11, uint64(d134.Imm.Int()))
					} else {
						r11 = d134.Reg
					}
					ctx.ProtectReg(r11)
					ctx.SyncDesc(&d133)
					ctx.EnsureDesc(&d133)
					if d133.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d133.Imm.Int()))
					} else {
						r12 = d133.Reg
					}
					ctx.ProtectReg(r12)
					ctx.UnprotectReg(r12)
					ctx.UnprotectReg(r11)
					d135 = JITValueDesc{Loc: LocRegPair, Reg: r11, Reg2: r12}
					ctx.BindReg(r11, &d135)
					ctx.BindReg(r12, &d135)
					ctx.BindReg(r11, &d135)
					ctx.BindReg(r12, &d135)
					ctx.FreeDesc(&d131)
					ctx.EnsureDesc(&d135)
					d136 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d135}, 2)
					ctx.EmitMovPairToResult(&d136, &result)
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
					ctx.EnsureDesc(&d86)
					if d86.Loc == LocRegPair || d86.Loc == LocRegTriple {
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d86.Reg2}
						ctx.BindReg(d86.Reg2, &d137)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d86)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d137)
					if d137.Loc == LocImm && d4.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d137.Imm.Int() - d4.Imm.Int())}
					} else {
						r13 := ctx.AllocReg()
						if d137.Loc == LocImm {
							ctx.EmitMovRegImm64(r13, uint64(d137.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r13, d137.Reg)
						}
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d4.Imm.Int()))
							ctx.EmitSubInt64(r13, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r13, d4.Reg)
						}
						d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d139)
					}
					r14 := ctx.EmitSliceDataAfterLow(&d86, &d4, 1)
					d140 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d140)
					ctx.BindReg(r14, &d140)
					var r15 Reg
					var r16 Reg
					ctx.SyncDesc(&d140)
					ctx.EnsureDesc(&d140)
					if d140.Loc == LocImm {
						r15 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r15, uint64(d140.Imm.Int()))
					} else {
						r15 = d140.Reg
					}
					ctx.ProtectReg(r15)
					ctx.SyncDesc(&d139)
					ctx.EnsureDesc(&d139)
					if d139.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d139.Imm.Int()))
					} else {
						r16 = d139.Reg
					}
					ctx.ProtectReg(r16)
					ctx.UnprotectReg(r16)
					ctx.UnprotectReg(r15)
					d141 = JITValueDesc{Loc: LocRegPair, Reg: r15, Reg2: r16}
					ctx.BindReg(r15, &d141)
					ctx.BindReg(r16, &d141)
					ctx.BindReg(r15, &d141)
					ctx.BindReg(r16, &d141)
					ctx.EnsureDesc(&d141)
					d142 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d141}, 2)
					ctx.EmitMovPairToResult(&d142, &result)
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
			JITInlineCost:  55,
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
				var d60 JITValueDesc
				_ = d60
				var d92 JITValueDesc
				_ = d92
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
				var d132 JITValueDesc
				_ = d132
				var d173 JITValueDesc
				_ = d173
				var d174 JITValueDesc
				_ = d174
				var d175 JITValueDesc
				_ = d175
				var d176 JITValueDesc
				_ = d176
				var d177 JITValueDesc
				_ = d177
				var d223 JITValueDesc
				_ = d223
				var d224 JITValueDesc
				_ = d224
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d227 JITValueDesc
				_ = d227
				var d228 JITValueDesc
				_ = d228
				var d229 JITValueDesc
				_ = d229
				var d230 JITValueDesc
				_ = d230
				var d284 JITValueDesc
				_ = d284
				var d339 JITValueDesc
				_ = d339
				var d340 JITValueDesc
				_ = d340
				var phiBase341 int32
				_ = phiBase341
				var d342 JITValueDesc
				_ = d342
				var d343 JITValueDesc
				_ = d343
				var d344 JITValueDesc
				_ = d344
				var d345 JITValueDesc
				_ = d345
				var d346 JITValueDesc
				_ = d346
				var d347 JITValueDesc
				_ = d347
				var d348 JITValueDesc
				_ = d348
				var d349 JITValueDesc
				_ = d349
				var d414 JITValueDesc
				_ = d414
				var d415 JITValueDesc
				_ = d415
				var d416 JITValueDesc
				_ = d416
				var d484 JITValueDesc
				_ = d484
				var d485 JITValueDesc
				_ = d485
				var d486 JITValueDesc
				_ = d486
				var d487 JITValueDesc
				_ = d487
				var phiBase488 int32
				_ = phiBase488
				var d489 JITValueDesc
				_ = d489
				var d490 JITValueDesc
				_ = d490
				var d491 JITValueDesc
				_ = d491
				var d492 JITValueDesc
				_ = d492
				var d493 JITValueDesc
				_ = d493
				var d494 JITValueDesc
				_ = d494
				var d495 JITValueDesc
				_ = d495
				var d496 JITValueDesc
				_ = d496
				var d576 JITValueDesc
				_ = d576
				var d577 JITValueDesc
				_ = d577
				var d578 JITValueDesc
				_ = d578
				var d579 JITValueDesc
				_ = d579
				var d580 JITValueDesc
				_ = d580
				var d581 JITValueDesc
				_ = d581
				var d582 JITValueDesc
				_ = d582
				var d583 JITValueDesc
				_ = d583
				var d584 JITValueDesc
				_ = d584
				var d585 JITValueDesc
				_ = d585
				var d586 JITValueDesc
				_ = d586
				var d587 JITValueDesc
				_ = d587
				var d588 JITValueDesc
				_ = d588
				var d589 JITValueDesc
				_ = d589
				var d590 JITValueDesc
				_ = d590
				var d591 JITValueDesc
				_ = d591
				var d592 JITValueDesc
				_ = d592
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
					ctx.EnsureDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							ctx.MarkLabel(lbl25)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase21)+int32(0))
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl26)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitJump(d26.Condition, lbl25)
						ctx.EmitJmp(lbl26)
						ctx.FreeDesc(&d25)
						ctx.MarkLabel(lbl25)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase21)+int32(0))
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl23)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					d22 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase21) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d24)
					if d24.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d24.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d24.Reg, 20)
						r2 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d27 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d27)
					}
					ctx.EnsureDesc(&d27)
					ctx.EmitStoreToStack(d27, int32(phiBase21)+int32(0))
					ctx.StabilizeDescForControlFlow(&d27)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl24)
					ctx.MarkLabel(lbl21)
					d28 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d28)
					ctx.BindReg(r1, &d28)
					ctx.FreeDesc(&d19)
					d29 = d28
					ctx.EnsureDesc(&d29)
					if d29.Loc != LocImm && d29.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d29.Loc == LocImm {
						if d29.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d29.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap30 := d1
						snap31 := d2
						snap32 := d3
						snap33 := d4
						snap34 := d5
						snap35 := d6
						snap36 := d7
						snap37 := d8
						snap38 := d18
						snap39 := d19
						snap40 := d20
						snap41 := d22
						snap42 := d23
						snap43 := d24
						snap44 := d25
						snap45 := d26
						snap46 := d27
						snap47 := d28
						snap48 := d29
						alloc49 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc49)
						d1 = snap30
						d2 = snap31
						d3 = snap32
						d4 = snap33
						d5 = snap34
						d6 = snap35
						d7 = snap36
						d8 = snap37
						d18 = snap38
						d19 = snap39
						d20 = snap40
						d22 = snap41
						d23 = snap42
						d24 = snap43
						d25 = snap44
						d26 = snap45
						d27 = snap46
						d28 = snap47
						d29 = snap48
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d28)
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
					d50 = args[0]
					d50.ID = 0
					d50 = JITPrepareScmerGoArg(ctx, d50)
					ctx.SyncDesc(&d50)
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d50}, 1)
					d51.NoHeapPointer = true
					ctx.BindReg(d51.Reg, &d51)
					ctx.StabilizeDescForControlFlow(&d51)
					ctx.FreeDesc(&d50)
					ctx.SyncDesc(&d51)
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.ProtectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.ProtectReg(d51.Reg)
						ctx.ProtectReg(d51.Reg2)
					}
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}, int32(bbs[4].PhiBase)+int32(0))
					d52 = d51
					if d52.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d52)
					ctx.EmitStoreToStack(d52, int32(bbs[4].PhiBase)+int32(16))
					if d51.Loc == LocReg || d51.Loc == LocFPReg {
						ctx.UnprotectReg(d51.Reg)
					} else if d51.Loc == LocRegPair {
						ctx.UnprotectReg(d51.Reg)
						ctx.UnprotectReg(d51.Reg2)
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
					d53 = args[1]
					d53.ID = 0
					ctx.EnsureDesc(&d53)
					d54 = d53
					_ = d54
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl27 := ctx.ReserveLabel()
					_ = lbl27
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d54.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d54.Imm.Int())}
					} else if d54.Type == tagInt && d54.Loc == LocRegPair {
						ctx.FreeReg(d54.Reg)
						d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg2}
						ctx.BindReg(d54.Reg2, &d55)
						ctx.BindReg(d54.Reg2, &d55)
					} else if d54.Type == tagInt && d54.Loc == LocReg {
						d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg}
						ctx.BindReg(d54.Reg, &d55)
						ctx.BindReg(d54.Reg, &d55)
					} else {
						d55 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d54}, 1)
						d55.Type = tagInt
						ctx.BindReg(d55.Reg, &d55)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d55)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d55.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d55.Reg)
						ctx.EmitMovRegReg(scratch, d55.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d57)
					}
					if d57.Loc == LocReg && d55.Loc == LocReg && d57.Reg == d55.Reg {
						ctx.TransferReg(d55.Reg)
						d55.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d57)
					ctx.FreeDesc(&d55)
					ctx.EnsureDesc(&d57)
					if d57.Loc == LocImm {
						d58 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d57.Imm.Int() < 0)}
					} else {
						r3 := ctx.AllocRegExcept(d57.Reg)
						ctx.EmitCmpRegImm32(d57.Reg, 0)
						d58 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d58)
					}
					d59 = d58
					ctx.EnsureDesc(&d59)
					if d59.Loc != LocImm && d59.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d59.Loc == LocImm {
						if d59.Imm.Bool() {
							return bbs[6].Render()
						}
						ctx.SyncDesc(&d57)
						if d57.Loc == LocReg || d57.Loc == LocFPReg {
							ctx.ProtectReg(d57.Reg)
						} else if d57.Loc == LocRegPair {
							ctx.ProtectReg(d57.Reg)
							ctx.ProtectReg(d57.Reg2)
						}
						d60 = d57
						if d60.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d60)
						ctx.EmitStoreToStack(d60, int32(bbs[7].PhiBase)+int32(0))
						if d57.Loc == LocReg || d57.Loc == LocFPReg {
							ctx.UnprotectReg(d57.Reg)
						} else if d57.Loc == LocRegPair {
							ctx.UnprotectReg(d57.Reg)
							ctx.UnprotectReg(d57.Reg2)
						}
						return bbs[7].Render()
					}
					lbl28 := ctx.ReserveLabel()
					ctx.EmitJump(d59.Condition, lbl7)
					ctx.EmitJmp(lbl28)
					ctx.FreeDesc(&d58)
					snap61 := d1
					snap62 := d2
					snap63 := d3
					snap64 := d4
					snap65 := d5
					snap66 := d6
					snap67 := d7
					snap68 := d8
					snap69 := d18
					snap70 := d19
					snap71 := d20
					snap72 := d22
					snap73 := d23
					snap74 := d24
					snap75 := d25
					snap76 := d26
					snap77 := d27
					snap78 := d28
					snap79 := d29
					snap80 := d50
					snap81 := d51
					snap82 := d52
					snap83 := d53
					snap84 := d54
					snap85 := d55
					snap86 := d56
					snap87 := d57
					snap88 := d58
					snap89 := d59
					snap90 := d60
					alloc91 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl28)
					ctx.SyncDesc(&d57)
					if d57.Loc == LocReg || d57.Loc == LocFPReg {
						ctx.ProtectReg(d57.Reg)
					} else if d57.Loc == LocRegPair {
						ctx.ProtectReg(d57.Reg)
						ctx.ProtectReg(d57.Reg2)
					}
					d92 = d57
					if d92.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d92)
					ctx.EmitStoreToStack(d92, int32(bbs[7].PhiBase)+int32(0))
					if d57.Loc == LocReg || d57.Loc == LocFPReg {
						ctx.UnprotectReg(d57.Reg)
					} else if d57.Loc == LocRegPair {
						ctx.UnprotectReg(d57.Reg)
						ctx.UnprotectReg(d57.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc91)
					d1 = snap61
					d2 = snap62
					d3 = snap63
					d4 = snap64
					d5 = snap65
					d6 = snap66
					d7 = snap67
					d8 = snap68
					d18 = snap69
					d19 = snap70
					d20 = snap71
					d22 = snap72
					d23 = snap73
					d24 = snap74
					d25 = snap75
					d26 = snap76
					d27 = snap77
					d28 = snap78
					d29 = snap79
					d50 = snap80
					d51 = snap81
					d52 = snap82
					d53 = snap83
					d54 = snap84
					d55 = snap85
					d56 = snap86
					d57 = snap87
					d58 = snap88
					d59 = snap89
					d60 = snap90
					if !bbs[7].Rendered {
						snap93 := d1
						snap94 := d2
						snap95 := d3
						snap96 := d4
						snap97 := d5
						snap98 := d6
						snap99 := d7
						snap100 := d8
						snap101 := d18
						snap102 := d19
						snap103 := d20
						snap104 := d22
						snap105 := d23
						snap106 := d24
						snap107 := d25
						snap108 := d26
						snap109 := d27
						snap110 := d28
						snap111 := d29
						snap112 := d50
						snap113 := d51
						snap114 := d52
						snap115 := d53
						snap116 := d54
						snap117 := d55
						snap118 := d56
						snap119 := d57
						snap120 := d58
						snap121 := d59
						snap122 := d60
						snap123 := d92
						alloc124 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc124)
						d1 = snap93
						d2 = snap94
						d3 = snap95
						d4 = snap96
						d5 = snap97
						d6 = snap98
						d7 = snap99
						d8 = snap100
						d18 = snap101
						d19 = snap102
						d20 = snap103
						d22 = snap104
						d23 = snap105
						d24 = snap106
						d25 = snap107
						d26 = snap108
						d27 = snap109
						d28 = snap110
						d29 = snap111
						d50 = snap112
						d51 = snap113
						d52 = snap114
						d53 = snap115
						d54 = snap116
						d55 = snap117
						d56 = snap118
						d57 = snap119
						d58 = snap120
						d59 = snap121
						d60 = snap122
						d92 = snap123
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
					d125 = args[0]
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
					ctx.StabilizeDescForControlFlow(&d126)
					ctx.FreeDesc(&d125)
					if d126.SliceSizeKnown {
						d128 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d126.KnownSliceLen))}
					} else if d126.Loc == LocImm {
						d128 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d126.Imm.String())))}
					} else if d126.Loc == LocStackTriple {
						d128 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else if d126.Loc == LocStackPair {
						d128 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d126)
						if d126.Loc == LocRegPair || d126.Loc == LocRegTriple {
							d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2, ID: 0}
						} else if d126.Loc == LocReg {
							d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d128)
					ctx.SyncDesc(&d126)
					if d126.Loc == LocReg || d126.Loc == LocFPReg {
						ctx.ProtectReg(d126.Reg)
					} else if d126.Loc == LocRegPair {
						ctx.ProtectReg(d126.Reg)
						ctx.ProtectReg(d126.Reg2)
					}
					ctx.SyncDesc(&d128)
					if d128.Loc == LocReg || d128.Loc == LocFPReg {
						ctx.ProtectReg(d128.Reg)
					} else if d128.Loc == LocRegPair {
						ctx.ProtectReg(d128.Reg)
						ctx.ProtectReg(d128.Reg2)
					}
					d129 = d126
					if d129.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d129)
					if d129.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d129, int32(bbs[4].PhiBase)+int32(0), 2)
					} else if d129.Loc == LocInputPair {
						ctx.EnsureDesc(&d129)
						ctx.EmitStoreScmerToStack(d129, int32(bbs[4].PhiBase)+int32(0))
					} else if d129.Loc == LocRegPair || d129.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d129, int32(bbs[4].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d129)
						ctx.EmitStoreToStack(d129, int32(bbs[4].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[4].PhiBase)+int32(0))+8)
					}
					d130 = d128
					if d130.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d130)
					ctx.EmitStoreToStack(d130, int32(bbs[4].PhiBase)+int32(16))
					if d126.Loc == LocReg || d126.Loc == LocFPReg {
						ctx.UnprotectReg(d126.Reg)
					} else if d126.Loc == LocRegPair {
						ctx.UnprotectReg(d126.Reg)
						ctx.UnprotectReg(d126.Reg2)
					}
					if d128.Loc == LocReg || d128.Loc == LocFPReg {
						ctx.UnprotectReg(d128.Reg)
					} else if d128.Loc == LocRegPair {
						ctx.UnprotectReg(d128.Reg)
						ctx.UnprotectReg(d128.Reg2)
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
						d131 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() >= d2.Imm.Int())}
					} else if d2.Loc == LocImm {
						r4 := ctx.AllocRegExcept(d3.Reg)
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d131 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r4, &d131)
					} else if d3.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
						d131 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r5, &d131)
					} else {
						r6 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d2.Reg)
						d131 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r6, &d131)
					}
					d132 = d131
					ctx.EnsureDesc(&d132)
					if d132.Loc != LocImm && d132.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d132.Loc == LocImm {
						if d132.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitJump(d132.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d131)
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap133 := d1
						snap134 := d2
						snap135 := d3
						snap136 := d4
						snap137 := d5
						snap138 := d6
						snap139 := d7
						snap140 := d8
						snap141 := d18
						snap142 := d19
						snap143 := d20
						snap144 := d22
						snap145 := d23
						snap146 := d24
						snap147 := d25
						snap148 := d26
						snap149 := d27
						snap150 := d28
						snap151 := d29
						snap152 := d50
						snap153 := d51
						snap154 := d52
						snap155 := d53
						snap156 := d54
						snap157 := d55
						snap158 := d56
						snap159 := d57
						snap160 := d58
						snap161 := d59
						snap162 := d60
						snap163 := d92
						snap164 := d125
						snap165 := d126
						snap166 := d127
						snap167 := d128
						snap168 := d129
						snap169 := d130
						snap170 := d131
						snap171 := d132
						alloc172 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc172)
						d1 = snap133
						d2 = snap134
						d3 = snap135
						d4 = snap136
						d5 = snap137
						d6 = snap138
						d7 = snap139
						d8 = snap140
						d18 = snap141
						d19 = snap142
						d20 = snap143
						d22 = snap144
						d23 = snap145
						d24 = snap146
						d25 = snap147
						d26 = snap148
						d27 = snap149
						d28 = snap150
						d29 = snap151
						d50 = snap152
						d51 = snap153
						d52 = snap154
						d53 = snap155
						d54 = snap156
						d55 = snap157
						d56 = snap158
						d57 = snap159
						d58 = snap160
						d59 = snap161
						d60 = snap162
						d92 = snap163
						d125 = snap164
						d126 = snap165
						d127 = snap166
						d128 = snap167
						d129 = snap168
						d130 = snap169
						d131 = snap170
						d132 = snap171
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
					d173 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d174 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d173}, 2)
					ctx.EmitMovPairToResult(&d174, &result)
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
					d175 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d175)
					if d175.Loc == LocImm {
						d176 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d175.Imm.Int() > 2)}
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d175.Reg, 2)
						d176 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedGreater}
						ctx.BindReg(r7, &d176)
					}
					ctx.FreeDesc(&d175)
					d177 = d176
					ctx.EnsureDesc(&d177)
					if d177.Loc != LocImm && d177.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d177.Loc == LocImm {
						if d177.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitJump(d177.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d176)
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap178 := d1
						snap179 := d2
						snap180 := d3
						snap181 := d4
						snap182 := d5
						snap183 := d6
						snap184 := d7
						snap185 := d8
						snap186 := d18
						snap187 := d19
						snap188 := d20
						snap189 := d22
						snap190 := d23
						snap191 := d24
						snap192 := d25
						snap193 := d26
						snap194 := d27
						snap195 := d28
						snap196 := d29
						snap197 := d50
						snap198 := d51
						snap199 := d52
						snap200 := d53
						snap201 := d54
						snap202 := d55
						snap203 := d56
						snap204 := d57
						snap205 := d58
						snap206 := d59
						snap207 := d60
						snap208 := d92
						snap209 := d125
						snap210 := d126
						snap211 := d127
						snap212 := d128
						snap213 := d129
						snap214 := d130
						snap215 := d131
						snap216 := d132
						snap217 := d173
						snap218 := d174
						snap219 := d175
						snap220 := d176
						snap221 := d177
						alloc222 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc222)
						d1 = snap178
						d2 = snap179
						d3 = snap180
						d4 = snap181
						d5 = snap182
						d6 = snap183
						d7 = snap184
						d8 = snap185
						d18 = snap186
						d19 = snap187
						d20 = snap188
						d22 = snap189
						d23 = snap190
						d24 = snap191
						d25 = snap192
						d26 = snap193
						d27 = snap194
						d28 = snap195
						d29 = snap196
						d50 = snap197
						d51 = snap198
						d52 = snap199
						d53 = snap200
						d54 = snap201
						d55 = snap202
						d56 = snap203
						d57 = snap204
						d58 = snap205
						d59 = snap206
						d60 = snap207
						d92 = snap208
						d125 = snap209
						d126 = snap210
						d127 = snap211
						d128 = snap212
						d129 = snap213
						d130 = snap214
						d131 = snap215
						d132 = snap216
						d173 = snap217
						d174 = snap218
						d175 = snap219
						d176 = snap220
						d177 = snap221
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
					d223 = args[2]
					d223.ID = 0
					ctx.EnsureDesc(&d223)
					d224 = d223
					_ = d224
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl29 := ctx.ReserveLabel()
					_ = lbl29
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl29)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d224.Loc == LocImm {
						d225 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d224.Imm.Int())}
					} else if d224.Type == tagInt && d224.Loc == LocRegPair {
						ctx.FreeReg(d224.Reg)
						d225 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d224.Reg2}
						ctx.BindReg(d224.Reg2, &d225)
						ctx.BindReg(d224.Reg2, &d225)
					} else if d224.Type == tagInt && d224.Loc == LocReg {
						d225 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d224.Reg}
						ctx.BindReg(d224.Reg, &d225)
						ctx.BindReg(d224.Reg, &d225)
					} else {
						d225 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d224}, 1)
						d225.Type = tagInt
						ctx.BindReg(d225.Reg, &d225)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d225)
					ctx.EnsureDesc(&d225)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d225)
					ctx.StabilizeDescForControlFlow(&d225)
					ctx.FreeDesc(&d223)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d225)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d225)
					if d3.Loc == LocImm && d225.Loc == LocImm {
						d227 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d225.Imm.Int())}
					} else if d225.Loc == LocImm && d225.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r8 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r8, d3.Reg)
						d227 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d227)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d225)
						d227 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d225.Reg}
						ctx.BindReg(d225.Reg, &d227)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d225)
						scratch := ctx.AllocRegExcept(d225.Reg)
						ctx.EmitMovRegReg(scratch, d225.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d227 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d227)
					} else if d225.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d225.Imm.Int())
						d227 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d227)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d225)
						r9 := ctx.AllocRegExceptOperand(&d225, d3.Reg)
						ctx.EmitMovRegReg(r9, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r9, &d225)
						d227 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
						ctx.BindReg(r9, &d227)
					}
					if d227.Loc == LocReg && d3.Loc == LocReg && d227.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d227)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDescsTogether(&d227, &d2)
					if d227.Loc == LocImm && d2.Loc == LocImm {
						d228 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d227.Imm.Int() > d2.Imm.Int())}
					} else if d2.Loc == LocImm {
						r10 := ctx.AllocReg()
						if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d227.Reg, int32(d2.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
							ctx.EmitCmpInt64(d227.Reg, ctx.ScratchReg)
						}
						d228 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedGreater}
						ctx.BindReg(r10, &d228)
					} else if d227.Loc == LocImm {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d227.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
						d228 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedGreater}
						ctx.BindReg(r11, &d228)
					} else {
						r12 := ctx.AllocReg()
						ctx.EmitCmpInt64(d227.Reg, d2.Reg)
						d228 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedGreater}
						ctx.BindReg(r12, &d228)
					}
					ctx.FreeDesc(&d227)
					d229 = d228
					ctx.EnsureDesc(&d229)
					if d229.Loc != LocImm && d229.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d229.Loc == LocImm {
						if d229.Imm.Bool() {
							return bbs[12].Render()
						}
						ctx.SyncDesc(&d225)
						if d225.Loc == LocReg || d225.Loc == LocFPReg {
							ctx.ProtectReg(d225.Reg)
						} else if d225.Loc == LocRegPair {
							ctx.ProtectReg(d225.Reg)
							ctx.ProtectReg(d225.Reg2)
						}
						d230 = d225
						if d230.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d230)
						ctx.EmitStoreToStack(d230, int32(bbs[13].PhiBase)+int32(0))
						if d225.Loc == LocReg || d225.Loc == LocFPReg {
							ctx.UnprotectReg(d225.Reg)
						} else if d225.Loc == LocRegPair {
							ctx.UnprotectReg(d225.Reg)
							ctx.UnprotectReg(d225.Reg2)
						}
						return bbs[13].Render()
					}
					lbl30 := ctx.ReserveLabel()
					ctx.EmitJump(d229.Condition, lbl13)
					ctx.EmitJmp(lbl30)
					ctx.FreeDesc(&d228)
					snap231 := d1
					snap232 := d2
					snap233 := d3
					snap234 := d4
					snap235 := d5
					snap236 := d6
					snap237 := d7
					snap238 := d8
					snap239 := d18
					snap240 := d19
					snap241 := d20
					snap242 := d22
					snap243 := d23
					snap244 := d24
					snap245 := d25
					snap246 := d26
					snap247 := d27
					snap248 := d28
					snap249 := d29
					snap250 := d50
					snap251 := d51
					snap252 := d52
					snap253 := d53
					snap254 := d54
					snap255 := d55
					snap256 := d56
					snap257 := d57
					snap258 := d58
					snap259 := d59
					snap260 := d60
					snap261 := d92
					snap262 := d125
					snap263 := d126
					snap264 := d127
					snap265 := d128
					snap266 := d129
					snap267 := d130
					snap268 := d131
					snap269 := d132
					snap270 := d173
					snap271 := d174
					snap272 := d175
					snap273 := d176
					snap274 := d177
					snap275 := d223
					snap276 := d224
					snap277 := d225
					snap278 := d226
					snap279 := d227
					snap280 := d228
					snap281 := d229
					snap282 := d230
					alloc283 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl30)
					ctx.SyncDesc(&d225)
					if d225.Loc == LocReg || d225.Loc == LocFPReg {
						ctx.ProtectReg(d225.Reg)
					} else if d225.Loc == LocRegPair {
						ctx.ProtectReg(d225.Reg)
						ctx.ProtectReg(d225.Reg2)
					}
					d284 = d225
					if d284.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d284)
					ctx.EmitStoreToStack(d284, int32(bbs[13].PhiBase)+int32(0))
					if d225.Loc == LocReg || d225.Loc == LocFPReg {
						ctx.UnprotectReg(d225.Reg)
					} else if d225.Loc == LocRegPair {
						ctx.UnprotectReg(d225.Reg)
						ctx.UnprotectReg(d225.Reg2)
					}
					ctx.EmitJmp(lbl14)
					ctx.RestoreAllocState(alloc283)
					d1 = snap231
					d2 = snap232
					d3 = snap233
					d4 = snap234
					d5 = snap235
					d6 = snap236
					d7 = snap237
					d8 = snap238
					d18 = snap239
					d19 = snap240
					d20 = snap241
					d22 = snap242
					d23 = snap243
					d24 = snap244
					d25 = snap245
					d26 = snap246
					d27 = snap247
					d28 = snap248
					d29 = snap249
					d50 = snap250
					d51 = snap251
					d52 = snap252
					d53 = snap253
					d54 = snap254
					d55 = snap255
					d56 = snap256
					d57 = snap257
					d58 = snap258
					d59 = snap259
					d60 = snap260
					d92 = snap261
					d125 = snap262
					d126 = snap263
					d127 = snap264
					d128 = snap265
					d129 = snap266
					d130 = snap267
					d131 = snap268
					d132 = snap269
					d173 = snap270
					d174 = snap271
					d175 = snap272
					d176 = snap273
					d177 = snap274
					d223 = snap275
					d224 = snap276
					d225 = snap277
					d226 = snap278
					d227 = snap279
					d228 = snap280
					d229 = snap281
					d230 = snap282
					if !bbs[13].Rendered {
						snap285 := d1
						snap286 := d2
						snap287 := d3
						snap288 := d4
						snap289 := d5
						snap290 := d6
						snap291 := d7
						snap292 := d8
						snap293 := d18
						snap294 := d19
						snap295 := d20
						snap296 := d22
						snap297 := d23
						snap298 := d24
						snap299 := d25
						snap300 := d26
						snap301 := d27
						snap302 := d28
						snap303 := d29
						snap304 := d50
						snap305 := d51
						snap306 := d52
						snap307 := d53
						snap308 := d54
						snap309 := d55
						snap310 := d56
						snap311 := d57
						snap312 := d58
						snap313 := d59
						snap314 := d60
						snap315 := d92
						snap316 := d125
						snap317 := d126
						snap318 := d127
						snap319 := d128
						snap320 := d129
						snap321 := d130
						snap322 := d131
						snap323 := d132
						snap324 := d173
						snap325 := d174
						snap326 := d175
						snap327 := d176
						snap328 := d177
						snap329 := d223
						snap330 := d224
						snap331 := d225
						snap332 := d226
						snap333 := d227
						snap334 := d228
						snap335 := d229
						snap336 := d230
						snap337 := d284
						alloc338 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc338)
						d1 = snap285
						d2 = snap286
						d3 = snap287
						d4 = snap288
						d5 = snap289
						d6 = snap290
						d7 = snap291
						d8 = snap292
						d18 = snap293
						d19 = snap294
						d20 = snap295
						d22 = snap296
						d23 = snap297
						d24 = snap298
						d25 = snap299
						d26 = snap300
						d27 = snap301
						d28 = snap302
						d29 = snap303
						d50 = snap304
						d51 = snap305
						d52 = snap306
						d53 = snap307
						d54 = snap308
						d55 = snap309
						d56 = snap310
						d57 = snap311
						d58 = snap312
						d59 = snap313
						d60 = snap314
						d92 = snap315
						d125 = snap316
						d126 = snap317
						d127 = snap318
						d128 = snap319
						d129 = snap320
						d130 = snap321
						d131 = snap322
						d132 = snap323
						d173 = snap324
						d174 = snap325
						d175 = snap326
						d176 = snap327
						d177 = snap328
						d223 = snap329
						d224 = snap330
						d225 = snap331
						d226 = snap332
						d227 = snap333
						d228 = snap334
						d229 = snap335
						d230 = snap336
						d284 = snap337
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
					d339 = args[0]
					d339.ID = 0
					ctx.EnsureDesc(&d339)
					d340 = d339
					_ = d340
					ctx.StabilizeDescForControlFlow(&d340)
					phiBase341 = ctx.AllocStack(int32(16))
					d342 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase341) + int32(0)}
					_ = d342
					lbl31 := ctx.ReserveLabel()
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl32 := ctx.ReserveLabel()
					_ = lbl32
					bbpos_4_1 := int32(-1)
					_ = bbpos_4_1
					lbl33 := ctx.ReserveLabel()
					_ = lbl33
					bbpos_4_2 := int32(-1)
					_ = bbpos_4_2
					lbl34 := ctx.ReserveLabel()
					_ = lbl34
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl32)
					ctx.ResolveFixups()
					d342 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase341) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d343 = d340
					d343.ID = 0
					d344 = ctx.EmitGetTagDesc(&d343, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d344)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d344)
					if d344.Loc == LocImm {
						d345 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d344.Imm.Int()) == uint64(0x13))}
					} else {
						r13 := ctx.AllocRegExcept(d344.Reg)
						ctx.EmitCmpRegImm32(d344.Reg, 19)
						d345 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondEqual}
						ctx.BindReg(r13, &d345)
					}
					ctx.ReclaimUntrackedRegs()
					d346 = d345
					ctx.EnsureDesc(&d346)
					if d346.Loc != LocImm && d346.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl35 := ctx.ReserveLabel()
					lbl36 := ctx.ReserveLabel()
					if d346.Loc == LocImm {
						if d346.Imm.Bool() {
							ctx.MarkLabel(lbl35)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase341)+int32(0))
							ctx.EmitJmp(lbl34)
						} else {
							ctx.MarkLabel(lbl36)
							ctx.EmitJmp(lbl33)
						}
					} else {
						ctx.EmitJump(d346.Condition, lbl35)
						ctx.EmitJmp(lbl36)
						ctx.FreeDesc(&d345)
						ctx.MarkLabel(lbl35)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase341)+int32(0))
						ctx.EmitJmp(lbl34)
						ctx.MarkLabel(lbl36)
						ctx.EmitJmp(lbl33)
					}
					bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl34)
					ctx.ResolveFixups()
					d342 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase341) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r14 := ctx.AllocReg()
					ctx.EnsureDesc(&d342)
					ctx.EnsureDesc(&d342)
					if d342.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r14, d342)
					}
					ctx.EmitJmp(lbl31)
					bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl33)
					ctx.ResolveFixups()
					d342 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase341) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d344)
					if d344.Loc == LocImm {
						d347 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d344.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d344.Reg, 20)
						r15 := ctx.AllocRegExcept(d344.Reg)
						ctx.EmitSetcc(r15, CondEqual)
						d347 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
						ctx.BindReg(r15, &d347)
					}
					ctx.EnsureDesc(&d347)
					ctx.EmitStoreToStack(d347, int32(phiBase341)+int32(0))
					ctx.StabilizeDescForControlFlow(&d347)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl34)
					ctx.MarkLabel(lbl31)
					d348 = JITValueDesc{Loc: LocReg, Reg: r14}
					ctx.BindReg(r14, &d348)
					ctx.BindReg(r14, &d348)
					ctx.FreeDesc(&d339)
					d349 = d348
					ctx.EnsureDesc(&d349)
					if d349.Loc != LocImm && d349.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d349.Loc == LocImm {
						if d349.Imm.Bool() {
							return bbs[18].Render()
						}
						return bbs[19].Render()
					}
					ctx.EmitCmpRegImm32(d349.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					if bbs[19].Rendered {
						ctx.EmitJmp(lbl20)
					}
					ctx.FlushRegisterMoves()
					if !bbs[19].Rendered {
						snap350 := d1
						snap351 := d2
						snap352 := d3
						snap353 := d4
						snap354 := d5
						snap355 := d6
						snap356 := d7
						snap357 := d8
						snap358 := d18
						snap359 := d19
						snap360 := d20
						snap361 := d22
						snap362 := d23
						snap363 := d24
						snap364 := d25
						snap365 := d26
						snap366 := d27
						snap367 := d28
						snap368 := d29
						snap369 := d50
						snap370 := d51
						snap371 := d52
						snap372 := d53
						snap373 := d54
						snap374 := d55
						snap375 := d56
						snap376 := d57
						snap377 := d58
						snap378 := d59
						snap379 := d60
						snap380 := d92
						snap381 := d125
						snap382 := d126
						snap383 := d127
						snap384 := d128
						snap385 := d129
						snap386 := d130
						snap387 := d131
						snap388 := d132
						snap389 := d173
						snap390 := d174
						snap391 := d175
						snap392 := d176
						snap393 := d177
						snap394 := d223
						snap395 := d224
						snap396 := d225
						snap397 := d226
						snap398 := d227
						snap399 := d228
						snap400 := d229
						snap401 := d230
						snap402 := d284
						snap403 := d339
						snap404 := d340
						snap405 := d342
						snap406 := d343
						snap407 := d344
						snap408 := d345
						snap409 := d346
						snap410 := d347
						snap411 := d348
						snap412 := d349
						alloc413 := ctx.SnapshotAllocState()
						bbs[19].Render()
						ctx.RestoreAllocState(alloc413)
						d1 = snap350
						d2 = snap351
						d3 = snap352
						d4 = snap353
						d5 = snap354
						d6 = snap355
						d7 = snap356
						d8 = snap357
						d18 = snap358
						d19 = snap359
						d20 = snap360
						d22 = snap361
						d23 = snap362
						d24 = snap363
						d25 = snap364
						d26 = snap365
						d27 = snap366
						d28 = snap367
						d29 = snap368
						d50 = snap369
						d51 = snap370
						d52 = snap371
						d53 = snap372
						d54 = snap373
						d55 = snap374
						d56 = snap375
						d57 = snap376
						d58 = snap377
						d59 = snap378
						d60 = snap379
						d92 = snap380
						d125 = snap381
						d126 = snap382
						d127 = snap383
						d128 = snap384
						d129 = snap385
						d130 = snap386
						d131 = snap387
						d132 = snap388
						d173 = snap389
						d174 = snap390
						d175 = snap391
						d176 = snap392
						d177 = snap393
						d223 = snap394
						d224 = snap395
						d225 = snap396
						d226 = snap397
						d227 = snap398
						d228 = snap399
						d229 = snap400
						d230 = snap401
						d284 = snap402
						d339 = snap403
						d340 = snap404
						d342 = snap405
						d343 = snap406
						d344 = snap407
						d345 = snap408
						d346 = snap409
						d347 = snap410
						d348 = snap411
						d349 = snap412
					}
					if !bbs[18].Rendered {
						return bbs[18].Render()
					}
					return result
					ctx.FreeDesc(&d348)
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
						d414 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() - d3.Imm.Int())}
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d2)
						r16 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(r16, d2.Reg)
						d414 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d414)
					} else if d2.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d3)
						d414 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d414)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d2)
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d3.Imm.Int())
						d414 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d414)
					} else {
						ctx.EnsureDesc(&d2)
						ctx.SyncDesc(&d3)
						r17 := ctx.AllocRegExceptOperand(&d3, d2.Reg)
						ctx.EmitMovRegReg(r17, d2.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r17, &d3)
						d414 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d414)
					}
					if d414.Loc == LocReg && d2.Loc == LocReg && d414.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d414)
					ctx.EmitStoreToStack(d414, int32(bbs[13].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d414)
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
						d415 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < 0)}
					} else {
						r18 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpRegImm32(d4.Reg, 0)
						d415 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
						ctx.BindReg(r18, &d415)
					}
					d416 = d415
					ctx.EnsureDesc(&d416)
					if d416.Loc != LocImm && d416.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d416.Loc == LocImm {
						if d416.Imm.Bool() {
							return bbs[14].Render()
						}
						return bbs[15].Render()
					}
					ctx.EmitJump(d416.Condition, lbl15)
					if bbs[15].Rendered {
						ctx.EmitJmp(lbl16)
					}
					ctx.FreeDesc(&d415)
					ctx.FlushRegisterMoves()
					if !bbs[15].Rendered {
						snap417 := d1
						snap418 := d2
						snap419 := d3
						snap420 := d4
						snap421 := d5
						snap422 := d6
						snap423 := d7
						snap424 := d8
						snap425 := d18
						snap426 := d19
						snap427 := d20
						snap428 := d22
						snap429 := d23
						snap430 := d24
						snap431 := d25
						snap432 := d26
						snap433 := d27
						snap434 := d28
						snap435 := d29
						snap436 := d50
						snap437 := d51
						snap438 := d52
						snap439 := d53
						snap440 := d54
						snap441 := d55
						snap442 := d56
						snap443 := d57
						snap444 := d58
						snap445 := d59
						snap446 := d60
						snap447 := d92
						snap448 := d125
						snap449 := d126
						snap450 := d127
						snap451 := d128
						snap452 := d129
						snap453 := d130
						snap454 := d131
						snap455 := d132
						snap456 := d173
						snap457 := d174
						snap458 := d175
						snap459 := d176
						snap460 := d177
						snap461 := d223
						snap462 := d224
						snap463 := d225
						snap464 := d226
						snap465 := d227
						snap466 := d228
						snap467 := d229
						snap468 := d230
						snap469 := d284
						snap470 := d339
						snap471 := d340
						snap472 := d342
						snap473 := d343
						snap474 := d344
						snap475 := d345
						snap476 := d346
						snap477 := d347
						snap478 := d348
						snap479 := d349
						snap480 := d414
						snap481 := d415
						snap482 := d416
						alloc483 := ctx.SnapshotAllocState()
						bbs[15].Render()
						ctx.RestoreAllocState(alloc483)
						d1 = snap417
						d2 = snap418
						d3 = snap419
						d4 = snap420
						d5 = snap421
						d6 = snap422
						d7 = snap423
						d8 = snap424
						d18 = snap425
						d19 = snap426
						d20 = snap427
						d22 = snap428
						d23 = snap429
						d24 = snap430
						d25 = snap431
						d26 = snap432
						d27 = snap433
						d28 = snap434
						d29 = snap435
						d50 = snap436
						d51 = snap437
						d52 = snap438
						d53 = snap439
						d54 = snap440
						d55 = snap441
						d56 = snap442
						d57 = snap443
						d58 = snap444
						d59 = snap445
						d60 = snap446
						d92 = snap447
						d125 = snap448
						d126 = snap449
						d127 = snap450
						d128 = snap451
						d129 = snap452
						d130 = snap453
						d131 = snap454
						d132 = snap455
						d173 = snap456
						d174 = snap457
						d175 = snap458
						d176 = snap459
						d177 = snap460
						d223 = snap461
						d224 = snap462
						d225 = snap463
						d226 = snap464
						d227 = snap465
						d228 = snap466
						d229 = snap467
						d230 = snap468
						d284 = snap469
						d339 = snap470
						d340 = snap471
						d342 = snap472
						d343 = snap473
						d344 = snap474
						d345 = snap475
						d346 = snap476
						d347 = snap477
						d348 = snap478
						d349 = snap479
						d414 = snap480
						d415 = snap481
						d416 = snap482
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
					d484 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					d485 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d484}, 2)
					ctx.EmitMovPairToResult(&d485, &result)
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
					d486 = args[0]
					d486.ID = 0
					ctx.EnsureDesc(&d486)
					d487 = d486
					_ = d487
					ctx.StabilizeDescForControlFlow(&d487)
					phiBase488 = ctx.AllocStack(int32(16))
					d489 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase488) + int32(0)}
					_ = d489
					lbl37 := ctx.ReserveLabel()
					bbpos_5_0 := int32(-1)
					_ = bbpos_5_0
					lbl38 := ctx.ReserveLabel()
					_ = lbl38
					bbpos_5_1 := int32(-1)
					_ = bbpos_5_1
					lbl39 := ctx.ReserveLabel()
					_ = lbl39
					bbpos_5_2 := int32(-1)
					_ = bbpos_5_2
					lbl40 := ctx.ReserveLabel()
					_ = lbl40
					bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl38)
					ctx.ResolveFixups()
					d489 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase488) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d490 = d487
					d490.ID = 0
					d491 = ctx.EmitGetTagDesc(&d490, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d491)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d491)
					if d491.Loc == LocImm {
						d492 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d491.Imm.Int()) == uint64(0x13))}
					} else {
						r19 := ctx.AllocRegExcept(d491.Reg)
						ctx.EmitCmpRegImm32(d491.Reg, 19)
						d492 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondEqual}
						ctx.BindReg(r19, &d492)
					}
					ctx.ReclaimUntrackedRegs()
					d493 = d492
					ctx.EnsureDesc(&d493)
					if d493.Loc != LocImm && d493.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl41 := ctx.ReserveLabel()
					lbl42 := ctx.ReserveLabel()
					if d493.Loc == LocImm {
						if d493.Imm.Bool() {
							ctx.MarkLabel(lbl41)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase488)+int32(0))
							ctx.EmitJmp(lbl40)
						} else {
							ctx.MarkLabel(lbl42)
							ctx.EmitJmp(lbl39)
						}
					} else {
						ctx.EmitJump(d493.Condition, lbl41)
						ctx.EmitJmp(lbl42)
						ctx.FreeDesc(&d492)
						ctx.MarkLabel(lbl41)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase488)+int32(0))
						ctx.EmitJmp(lbl40)
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl39)
					}
					bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl40)
					ctx.ResolveFixups()
					d489 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase488) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r20 := ctx.AllocReg()
					ctx.EnsureDesc(&d489)
					ctx.EnsureDesc(&d489)
					if d489.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r20, d489)
					}
					ctx.EmitJmp(lbl37)
					bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl39)
					ctx.ResolveFixups()
					d489 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase488) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d491)
					if d491.Loc == LocImm {
						d494 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d491.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d491.Reg, 20)
						r21 := ctx.AllocRegExcept(d491.Reg)
						ctx.EmitSetcc(r21, CondEqual)
						d494 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d494)
					}
					ctx.EnsureDesc(&d494)
					ctx.EmitStoreToStack(d494, int32(phiBase488)+int32(0))
					ctx.StabilizeDescForControlFlow(&d494)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl40)
					ctx.MarkLabel(lbl37)
					d495 = JITValueDesc{Loc: LocReg, Reg: r20}
					ctx.BindReg(r20, &d495)
					ctx.BindReg(r20, &d495)
					ctx.FreeDesc(&d486)
					d496 = d495
					ctx.EnsureDesc(&d496)
					if d496.Loc != LocImm && d496.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d496.Loc == LocImm {
						if d496.Imm.Bool() {
							return bbs[16].Render()
						}
						return bbs[17].Render()
					}
					ctx.EmitCmpRegImm32(d496.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					if bbs[17].Rendered {
						ctx.EmitJmp(lbl18)
					}
					ctx.FlushRegisterMoves()
					if !bbs[17].Rendered {
						snap497 := d1
						snap498 := d2
						snap499 := d3
						snap500 := d4
						snap501 := d5
						snap502 := d6
						snap503 := d7
						snap504 := d8
						snap505 := d18
						snap506 := d19
						snap507 := d20
						snap508 := d22
						snap509 := d23
						snap510 := d24
						snap511 := d25
						snap512 := d26
						snap513 := d27
						snap514 := d28
						snap515 := d29
						snap516 := d50
						snap517 := d51
						snap518 := d52
						snap519 := d53
						snap520 := d54
						snap521 := d55
						snap522 := d56
						snap523 := d57
						snap524 := d58
						snap525 := d59
						snap526 := d60
						snap527 := d92
						snap528 := d125
						snap529 := d126
						snap530 := d127
						snap531 := d128
						snap532 := d129
						snap533 := d130
						snap534 := d131
						snap535 := d132
						snap536 := d173
						snap537 := d174
						snap538 := d175
						snap539 := d176
						snap540 := d177
						snap541 := d223
						snap542 := d224
						snap543 := d225
						snap544 := d226
						snap545 := d227
						snap546 := d228
						snap547 := d229
						snap548 := d230
						snap549 := d284
						snap550 := d339
						snap551 := d340
						snap552 := d342
						snap553 := d343
						snap554 := d344
						snap555 := d345
						snap556 := d346
						snap557 := d347
						snap558 := d348
						snap559 := d349
						snap560 := d414
						snap561 := d415
						snap562 := d416
						snap563 := d484
						snap564 := d485
						snap565 := d486
						snap566 := d487
						snap567 := d489
						snap568 := d490
						snap569 := d491
						snap570 := d492
						snap571 := d493
						snap572 := d494
						snap573 := d495
						snap574 := d496
						alloc575 := ctx.SnapshotAllocState()
						bbs[17].Render()
						ctx.RestoreAllocState(alloc575)
						d1 = snap497
						d2 = snap498
						d3 = snap499
						d4 = snap500
						d5 = snap501
						d6 = snap502
						d7 = snap503
						d8 = snap504
						d18 = snap505
						d19 = snap506
						d20 = snap507
						d22 = snap508
						d23 = snap509
						d24 = snap510
						d25 = snap511
						d26 = snap512
						d27 = snap513
						d28 = snap514
						d29 = snap515
						d50 = snap516
						d51 = snap517
						d52 = snap518
						d53 = snap519
						d54 = snap520
						d55 = snap521
						d56 = snap522
						d57 = snap523
						d58 = snap524
						d59 = snap525
						d60 = snap526
						d92 = snap527
						d125 = snap528
						d126 = snap529
						d127 = snap530
						d128 = snap531
						d129 = snap532
						d130 = snap533
						d131 = snap534
						d132 = snap535
						d173 = snap536
						d174 = snap537
						d175 = snap538
						d176 = snap539
						d177 = snap540
						d223 = snap541
						d224 = snap542
						d225 = snap543
						d226 = snap544
						d227 = snap545
						d228 = snap546
						d229 = snap547
						d230 = snap548
						d284 = snap549
						d339 = snap550
						d340 = snap551
						d342 = snap552
						d343 = snap553
						d344 = snap554
						d345 = snap555
						d346 = snap556
						d347 = snap557
						d348 = snap558
						d349 = snap559
						d414 = snap560
						d415 = snap561
						d416 = snap562
						d484 = snap563
						d485 = snap564
						d486 = snap565
						d487 = snap566
						d489 = snap567
						d490 = snap568
						d491 = snap569
						d492 = snap570
						d493 = snap571
						d494 = snap572
						d495 = snap573
						d496 = snap574
					}
					if !bbs[16].Rendered {
						return bbs[16].Render()
					}
					return result
					ctx.FreeDesc(&d495)
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
					d576 = args[0]
					d576.ID = 0
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d4)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d4)
					if d3.Loc == LocImm && d4.Loc == LocImm {
						d577 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d4.Imm.Int())}
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r22 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r22, d3.Reg)
						d577 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r22}
						ctx.BindReg(r22, &d577)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						d577 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d577)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d577 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d577)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d577 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d577)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d4)
						r23 := ctx.AllocRegExceptOperand(&d4, d3.Reg)
						ctx.EmitMovRegReg(r23, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r23, &d4)
						d577 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r23}
						ctx.BindReg(r23, &d577)
					}
					if d577.Loc == LocReg && d3.Loc == LocReg && d577.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					d576 = JITPrepareScmerGoArg(ctx, d576)
					if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d577.Loc == LocRegPair || d577.Loc == LocStackPair || d577.Loc == LocRegTriple || d577.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d576)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d577)
					d578 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d576, d3, d577}, 2)
					d578.NoHeapPointer = false
					ctx.BindReg(d578.Reg, &d578)
					ctx.BindReg(d578.Reg2, &d578)
					ctx.FreeDesc(&d576)
					ctx.FreeDesc(&d577)
					ctx.SyncDesc(&d578)
					if d578.Loc == LocRegPair || d578.Loc == LocStackPair || d578.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d578, &result)
						result.Type = d578.Type
					} else {
						switch d578.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d578)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d578)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d578)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d578, &result)
							result.Type = d578.Type
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
						d579 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d4.Imm.Int())}
					} else if d4.Loc == LocImm && d4.Imm.Int() == 0 {
						ctx.EnsureDesc(&d3)
						r24 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(r24, d3.Reg)
						d579 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
						ctx.BindReg(r24, &d579)
					} else if d3.Loc == LocImm && d3.Imm.Int() == 0 {
						ctx.EnsureDesc(&d4)
						d579 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d579)
					} else if d3.Loc == LocImm {
						ctx.EnsureDesc(&d4)
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d3.Imm.Int())
						d579 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d579)
					} else if d4.Loc == LocImm {
						ctx.EnsureDesc(&d3)
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, d4.Imm.Int())
						d579 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d579)
					} else {
						ctx.EnsureDesc(&d3)
						ctx.SyncDesc(&d4)
						r25 := ctx.AllocRegExceptOperand(&d4, d3.Reg)
						ctx.EmitMovRegReg(r25, d3.Reg)
						ctx.EmitIntBinary(JITIntAdd, 64, r25, &d4)
						d579 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d579)
					}
					if d579.Loc == LocReg && d3.Loc == LocReg && d579.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d579)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d579)
					if d579.Loc == LocImm && d3.Loc == LocImm {
						d581 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d579.Imm.Int() - d3.Imm.Int())}
					} else {
						r26 := ctx.AllocReg()
						if d579.Loc == LocImm {
							ctx.EmitMovRegImm64(r26, uint64(d579.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r26, d579.Reg)
						}
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitSubInt64(r26, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r26, d3.Reg)
						}
						d581 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
						ctx.BindReg(r26, &d581)
					}
					r27 := ctx.EmitSliceDataAfterLow(&d1, &d3, 1)
					d582 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
					ctx.BindReg(r27, &d582)
					ctx.BindReg(r27, &d582)
					var r28 Reg
					var r29 Reg
					ctx.SyncDesc(&d582)
					ctx.EnsureDesc(&d582)
					if d582.Loc == LocImm {
						r28 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r28, uint64(d582.Imm.Int()))
					} else {
						r28 = d582.Reg
					}
					ctx.ProtectReg(r28)
					ctx.SyncDesc(&d581)
					ctx.EnsureDesc(&d581)
					if d581.Loc == LocImm {
						r29 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r29, uint64(d581.Imm.Int()))
					} else {
						r29 = d581.Reg
					}
					ctx.ProtectReg(r29)
					ctx.UnprotectReg(r29)
					ctx.UnprotectReg(r28)
					d583 = JITValueDesc{Loc: LocRegPair, Reg: r28, Reg2: r29}
					ctx.BindReg(r28, &d583)
					ctx.BindReg(r29, &d583)
					ctx.BindReg(r28, &d583)
					ctx.BindReg(r29, &d583)
					ctx.FreeDesc(&d579)
					ctx.EnsureDesc(&d583)
					d584 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d583}, 2)
					ctx.EmitMovPairToResult(&d584, &result)
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
					d585 = args[0]
					d585.ID = 0
					d585 = JITPrepareScmerGoArg(ctx, d585)
					if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d585)
					ctx.SyncDesc(&d3)
					ctx.SyncDesc(&d2)
					d586 = ctx.EmitGoCallScalar(GoFuncAddr(cstringSubstring), []JITValueDesc{d585, d3, d2}, 2)
					d586.NoHeapPointer = false
					ctx.BindReg(d586.Reg, &d586)
					ctx.BindReg(d586.Reg2, &d586)
					ctx.FreeDesc(&d585)
					ctx.SyncDesc(&d586)
					if d586.Loc == LocRegPair || d586.Loc == LocStackPair || d586.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d586, &result)
						result.Type = d586.Type
					} else {
						switch d586.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d586)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d586)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d586)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d586, &result)
							result.Type = d586.Type
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
						d587 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d587)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d587)
					if d587.Loc == LocImm && d3.Loc == LocImm {
						d589 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d587.Imm.Int() - d3.Imm.Int())}
					} else {
						r30 := ctx.AllocReg()
						if d587.Loc == LocImm {
							ctx.EmitMovRegImm64(r30, uint64(d587.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r30, d587.Reg)
						}
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
							ctx.EmitSubInt64(r30, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r30, d3.Reg)
						}
						d589 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
						ctx.BindReg(r30, &d589)
					}
					r31 := ctx.EmitSliceDataAfterLow(&d1, &d3, 1)
					d590 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r31}
					ctx.BindReg(r31, &d590)
					ctx.BindReg(r31, &d590)
					var r32 Reg
					var r33 Reg
					ctx.SyncDesc(&d590)
					ctx.EnsureDesc(&d590)
					if d590.Loc == LocImm {
						r32 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r32, uint64(d590.Imm.Int()))
					} else {
						r32 = d590.Reg
					}
					ctx.ProtectReg(r32)
					ctx.SyncDesc(&d589)
					ctx.EnsureDesc(&d589)
					if d589.Loc == LocImm {
						r33 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r33, uint64(d589.Imm.Int()))
					} else {
						r33 = d589.Reg
					}
					ctx.ProtectReg(r33)
					ctx.UnprotectReg(r33)
					ctx.UnprotectReg(r32)
					d591 = JITValueDesc{Loc: LocRegPair, Reg: r32, Reg2: r33}
					ctx.BindReg(r32, &d591)
					ctx.BindReg(r33, &d591)
					ctx.BindReg(r32, &d591)
					ctx.BindReg(r33, &d591)
					ctx.EnsureDesc(&d591)
					d592 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d591}, 2)
					ctx.EmitMovPairToResult(&d592, &result)
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
			JITInlineCost: 100,
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
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
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
					d0 = JITPrepareScmerGoArg(ctx, d0)
					ctx.SyncDesc(&d0)
					d1 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsCString), []JITValueDesc{d0}, 1)
					d1.NoHeapPointer = true
					ctx.EmitAndRegImm32(d1.Reg, 1)
					d1.Type = tagBool
					ctx.BindReg(d1.Reg, &d1)
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
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
					d7 = args[0]
					d7.ID = 0
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						_, auxWord := d7.Imm.RawWords()
						d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d7.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r0 := ctx.AllocReg()
						ctx.EmitMovRegReg(r0, d7.Reg2)
						d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d8)
					}
					ctx.EnsureDesc(&d8)
					d9 = d8
					_ = d9
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d9.Imm.Int()) >> 8))}
					} else {
						r1 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitMovRegReg(r1, d9.Reg)
						ctx.EmitShrRegImm8(r1, 8)
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d10)
					}
					if d10.Loc == LocReg && d9.Loc == LocReg && d10.Reg == d9.Reg {
						ctx.TransferReg(d9.Reg)
						d9.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d10)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d10.Imm.Int() & 4398046511103)}
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0x3ffffffffff)
						ctx.EmitAndInt64(d10.Reg, ctx.ScratchReg)
						d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg}
						ctx.BindReg(d10.Reg, &d11)
					}
					if d11.Loc == LocReg && d10.Loc == LocReg && d11.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d11.Imm.Int()))))}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegReg(r2, d11.Reg)
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d12)
					}
					ctx.FreeDesc(&d11)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						ctx.EmitMakeInt(result, d12)
					} else {
						ctx.EmitMovToReg(result.Reg2, d12)
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d13)
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
					d14 = args[0]
					d14.ID = 0
					d14 = JITPrepareScmerGoArg(ctx, d14)
					ctx.SyncDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsBString), []JITValueDesc{d14}, 1)
					d15.NoHeapPointer = true
					ctx.EmitAndRegImm32(d15.Reg, 1)
					d15.Type = tagBool
					ctx.BindReg(d15.Reg, &d15)
					ctx.FreeDesc(&d14)
					d16 = d15
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap17 := d0
						snap18 := d1
						snap19 := d2
						snap20 := d7
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d13
						snap27 := d14
						snap28 := d15
						snap29 := d16
						alloc30 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc30)
						d0 = snap17
						d1 = snap18
						d2 = snap19
						d7 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d13 = snap26
						d14 = snap27
						d15 = snap28
						d16 = snap29
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d15)
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
					d31 = JITPrepareScmerGoArg(ctx, d31)
					ctx.SyncDesc(&d31)
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d31}, 1)
					d32.NoHeapPointer = true
					ctx.BindReg(d32.Reg, &d32)
					ctx.FreeDesc(&d31)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocImm {
						ctx.EmitMakeInt(result, d32)
					} else {
						ctx.EmitMovToReg(result.Reg2, d32)
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d34)
						if d32.Loc == LocReg && d32.Reg != result.Reg2 {
							ctx.FreeReg(d32.Reg)
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
					d35 = args[0]
					d35.ID = 0
					d37 = d35
					ctx.SyncDesc(&d37)
					if d37.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d37.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d37.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d37 = tmpScalar
					}
					d37 = JITPrepareScmerGoArg(ctx, d37)
					if d37.Loc != LocRegPair && d37.Loc != LocStackPair && d37.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d36 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d37}, 2)
					ctx.FreeDesc(&d35)
					if d36.SliceSizeKnown {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d36.KnownSliceLen))}
					} else if d36.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d36.Imm.String())))}
					} else if d36.Loc == LocStackTriple {
						d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d36.StackOff + 8, NoHeapPointer: true}
					} else if d36.Loc == LocStackPair {
						d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d36.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d36)
						if d36.Loc == LocRegPair || d36.Loc == LocRegTriple {
							d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg2, ID: 0}
						} else if d36.Loc == LocReg {
							d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						ctx.EmitMakeInt(result, d38)
					} else {
						ctx.EmitMovToReg(result.Reg2, d38)
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d40)
						if d38.Loc == LocReg && d38.Reg != result.Reg2 {
							ctx.FreeReg(d38.Reg)
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
			JITInlineCost:  31,
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
				var d114 JITValueDesc
				_ = d114
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
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
				var d160 JITValueDesc
				_ = d160
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
					ctx.EnsureDesc(&d18)
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
					ctx.EnsureDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase73)+int32(0))
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitJump(d78.Condition, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.FreeDesc(&d77)
						ctx.MarkLabel(lbl15)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase73)+int32(0))
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl13)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d74 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase73) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm {
						d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d76.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d76.Reg, 20)
						r3 := ctx.AllocRegExcept(d76.Reg)
						ctx.EmitSetcc(r3, CondEqual)
						d79 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d79)
					}
					ctx.EnsureDesc(&d79)
					ctx.EmitStoreToStack(d79, int32(phiBase73)+int32(0))
					ctx.StabilizeDescForControlFlow(&d79)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl11)
					d80 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d80)
					ctx.BindReg(r2, &d80)
					ctx.FreeDesc(&d71)
					d81 = d80
					ctx.EnsureDesc(&d81)
					if d81.Loc != LocImm && d81.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d81.Loc == LocImm {
						if d81.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d81.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap82 := d1
						snap83 := d2
						snap84 := d3
						snap85 := d4
						snap86 := d5
						snap87 := d12
						snap88 := d13
						snap89 := d14
						snap90 := d15
						snap91 := d16
						snap92 := d17
						snap93 := d18
						snap94 := d45
						snap95 := d46
						snap96 := d47
						snap97 := d48
						snap98 := d66
						snap99 := d67
						snap100 := d68
						snap101 := d69
						snap102 := d70
						snap103 := d71
						snap104 := d72
						snap105 := d74
						snap106 := d75
						snap107 := d76
						snap108 := d77
						snap109 := d78
						snap110 := d79
						snap111 := d80
						snap112 := d81
						alloc113 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc113)
						d1 = snap82
						d2 = snap83
						d3 = snap84
						d4 = snap85
						d5 = snap86
						d12 = snap87
						d13 = snap88
						d14 = snap89
						d15 = snap90
						d16 = snap91
						d17 = snap92
						d18 = snap93
						d45 = snap94
						d46 = snap95
						d47 = snap96
						d48 = snap97
						d66 = snap98
						d67 = snap99
						d68 = snap100
						d69 = snap101
						d70 = snap102
						d71 = snap103
						d72 = snap104
						d74 = snap105
						d75 = snap106
						d76 = snap107
						d77 = snap108
						d78 = snap109
						d79 = snap110
						d80 = snap111
						d81 = snap112
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d80)
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
					d114 = args[0]
					d114.ID = 0
					d114 = JITPrepareScmerGoArg(ctx, d114)
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
					ctx.SyncDesc(&d114)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d1)
					callResults115 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeCString), []JITValueDesc{d114, d14, d1}, []uint8{1, 1}, []uint8{0, 0})
					d116 = callResults115[0]
					_ = d116
					d117 = callResults115[1]
					_ = d117
					ctx.FreeDesc(&d114)
					ctx.StabilizeDescForControlFlow(&d116)
					d118 = d117
					ctx.EnsureDesc(&d118)
					if d118.Loc != LocImm && d118.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d118.Loc == LocImm {
						if d118.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d118.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap119 := d1
						snap120 := d2
						snap121 := d3
						snap122 := d4
						snap123 := d5
						snap124 := d12
						snap125 := d13
						snap126 := d14
						snap127 := d15
						snap128 := d16
						snap129 := d17
						snap130 := d18
						snap131 := d45
						snap132 := d46
						snap133 := d47
						snap134 := d48
						snap135 := d66
						snap136 := d67
						snap137 := d68
						snap138 := d69
						snap139 := d70
						snap140 := d71
						snap141 := d72
						snap142 := d74
						snap143 := d75
						snap144 := d76
						snap145 := d77
						snap146 := d78
						snap147 := d79
						snap148 := d80
						snap149 := d81
						snap150 := d114
						snap151 := d116
						snap152 := d117
						snap153 := d118
						alloc154 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc154)
						d1 = snap119
						d2 = snap120
						d3 = snap121
						d4 = snap122
						d5 = snap123
						d12 = snap124
						d13 = snap125
						d14 = snap126
						d15 = snap127
						d16 = snap128
						d17 = snap129
						d18 = snap130
						d45 = snap131
						d46 = snap132
						d47 = snap133
						d48 = snap134
						d66 = snap135
						d67 = snap136
						d68 = snap137
						d69 = snap138
						d70 = snap139
						d71 = snap140
						d72 = snap141
						d74 = snap142
						d75 = snap143
						d76 = snap144
						d77 = snap145
						d78 = snap146
						d79 = snap147
						d80 = snap148
						d81 = snap149
						d114 = snap150
						d116 = snap151
						d117 = snap152
						d118 = snap153
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d117)
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
					d155 = args[0]
					d155.ID = 0
					d157 = d155
					ctx.SyncDesc(&d157)
					if d157.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d157.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d157.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d157 = tmpScalar
					}
					d157 = JITPrepareScmerGoArg(ctx, d157)
					if d157.Loc != LocRegPair && d157.Loc != LocStackPair && d157.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d156 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d157}, 2)
					ctx.FreeDesc(&d155)
					ctx.EnsureDesc(&d156)
					if d156.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d156.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d156.Imm)
						ptrWord, _ := d156.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d156.Imm.String())))
						d156 = tmpPair
					} else if d156.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d156.Type, Reg: ctx.AllocRegExcept(d156.Reg), Reg2: ctx.AllocRegExcept(d156.Reg)}
						switch d156.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d156)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d156)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d156)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d156)
						d156 = tmpPair
					}
					if d156.Loc != LocRegPair && d156.Loc != LocStackPair && d156.Loc != LocInputPair {
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
					ctx.SyncDesc(&d156)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d1)
					d158 = ctx.EmitGoCallScalar(GoFuncAddr(StrLikeCollation), []JITValueDesc{d156, d14, d1}, 1)
					d158.NoHeapPointer = true
					ctx.EmitAndRegImm32(d158.Reg, 1)
					d158.Type = tagBool
					ctx.BindReg(d158.Reg, &d158)
					ctx.SyncDesc(&d158)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d158) {
						return d158
					}
					if d158.Loc == LocImm {
						ctx.EmitMakeBool(result, d158)
					} else {
						ctx.EmitMovToReg(result.Reg2, d158)
						d159 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d159)
						if d158.Loc == LocReg && d158.Reg != result.Reg2 {
							ctx.FreeReg(d158.Reg)
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
					ctx.SyncDesc(&d116)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d116) {
						return d116
					}
					if d116.Loc == LocImm {
						ctx.EmitMakeBool(result, d116)
					} else {
						ctx.EmitMovToReg(result.Reg2, d116)
						d160 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d160)
						if d116.Loc == LocReg && d116.Reg != result.Reg2 {
							ctx.FreeReg(d116.Reg)
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
			JITInlineCost: 47,
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
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
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
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase15)+int32(0))
							ctx.EmitJmp(lbl11)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d20.Condition, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.FreeDesc(&d19)
						ctx.MarkLabel(lbl12)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase15)+int32(0))
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl10)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d16 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase15) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d18.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d18.Reg, 20)
						r2 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d21)
					}
					ctx.EnsureDesc(&d21)
					ctx.EmitStoreToStack(d21, int32(phiBase15)+int32(0))
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl8)
					d22 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d22)
					ctx.BindReg(r1, &d22)
					ctx.FreeDesc(&d13)
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap24 := d0
						snap25 := d1
						snap26 := d2
						snap27 := d3
						snap28 := d9
						snap29 := d10
						snap30 := d11
						snap31 := d12
						snap32 := d13
						snap33 := d14
						snap34 := d16
						snap35 := d17
						snap36 := d18
						snap37 := d19
						snap38 := d20
						snap39 := d21
						snap40 := d22
						snap41 := d23
						alloc42 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc42)
						d0 = snap24
						d1 = snap25
						d2 = snap26
						d3 = snap27
						d9 = snap28
						d10 = snap29
						d11 = snap30
						d12 = snap31
						d13 = snap32
						d14 = snap33
						d16 = snap34
						d17 = snap35
						d18 = snap36
						d19 = snap37
						d20 = snap38
						d21 = snap39
						d22 = snap40
						d23 = snap41
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d22)
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
					d43 = args[1]
					d43.ID = 0
					d45 = d43
					d45.ID = 0
					d44 = ctx.EmitTagEqualsBorrowed(&d45, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d43)
					d46 = d44
					ctx.EnsureDesc(&d46)
					if d46.Loc != LocImm && d46.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d46.Loc == LocImm {
						if d46.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d46.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap47 := d0
						snap48 := d1
						snap49 := d2
						snap50 := d3
						snap51 := d9
						snap52 := d10
						snap53 := d11
						snap54 := d12
						snap55 := d13
						snap56 := d14
						snap57 := d16
						snap58 := d17
						snap59 := d18
						snap60 := d19
						snap61 := d20
						snap62 := d21
						snap63 := d22
						snap64 := d23
						snap65 := d43
						snap66 := d44
						snap67 := d45
						snap68 := d46
						alloc69 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc69)
						d0 = snap47
						d1 = snap48
						d2 = snap49
						d3 = snap50
						d9 = snap51
						d10 = snap52
						d11 = snap53
						d12 = snap54
						d13 = snap55
						d14 = snap56
						d16 = snap57
						d17 = snap58
						d18 = snap59
						d19 = snap60
						d20 = snap61
						d21 = snap62
						d22 = snap63
						d23 = snap64
						d43 = snap65
						d44 = snap66
						d45 = snap67
						d46 = snap68
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d44)
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
					d70 = args[0]
					d70.ID = 0
					d70 = JITPrepareScmerGoArg(ctx, d70)
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
					d71 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("bin")}
					ctx.EnsureDesc(&d71)
					if d71.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d71.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d71.Imm)
						ptrWord, _ := d71.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d71.Imm.String())))
						d71 = tmpPair
					} else if d71.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d71.Type, Reg: ctx.AllocRegExcept(d71.Reg), Reg2: ctx.AllocRegExcept(d71.Reg)}
						switch d71.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d71)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d71)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d71)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d71)
						d71 = tmpPair
					}
					if d71.Loc != LocRegPair && d71.Loc != LocStackPair && d71.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strLikeCString arg2)")
					}
					ctx.SyncDesc(&d70)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d71)
					callResults72 := JITEmitGoCallResults(ctx, GoFuncAddr(strLikeCString), []JITValueDesc{d70, d11, d71}, []uint8{1, 1}, []uint8{0, 0})
					ctx.FreeDesc(&d71)
					d73 = callResults72[0]
					_ = d73
					d74 = callResults72[1]
					_ = d74
					ctx.FreeDesc(&d70)
					ctx.StabilizeDescForControlFlow(&d73)
					d75 = d74
					ctx.EnsureDesc(&d75)
					if d75.Loc != LocImm && d75.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d75.Loc == LocImm {
						if d75.Imm.Bool() {
							return bbs[6].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d75.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap76 := d0
						snap77 := d1
						snap78 := d2
						snap79 := d3
						snap80 := d9
						snap81 := d10
						snap82 := d11
						snap83 := d12
						snap84 := d13
						snap85 := d14
						snap86 := d16
						snap87 := d17
						snap88 := d18
						snap89 := d19
						snap90 := d20
						snap91 := d21
						snap92 := d22
						snap93 := d23
						snap94 := d43
						snap95 := d44
						snap96 := d45
						snap97 := d46
						snap98 := d70
						snap99 := d71
						snap100 := d73
						snap101 := d74
						snap102 := d75
						alloc103 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc103)
						d0 = snap76
						d1 = snap77
						d2 = snap78
						d3 = snap79
						d9 = snap80
						d10 = snap81
						d11 = snap82
						d12 = snap83
						d13 = snap84
						d14 = snap85
						d16 = snap86
						d17 = snap87
						d18 = snap88
						d19 = snap89
						d20 = snap90
						d21 = snap91
						d22 = snap92
						d23 = snap93
						d43 = snap94
						d44 = snap95
						d45 = snap96
						d46 = snap97
						d70 = snap98
						d71 = snap99
						d73 = snap100
						d74 = snap101
						d75 = snap102
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
					ctx.FreeDesc(&d74)
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
					d104 = args[0]
					d104.ID = 0
					d106 = d104
					ctx.SyncDesc(&d106)
					if d106.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d106.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d106.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d106 = tmpScalar
					}
					d106 = JITPrepareScmerGoArg(ctx, d106)
					if d106.Loc != LocRegPair && d106.Loc != LocStackPair && d106.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d105 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d106}, 2)
					ctx.FreeDesc(&d104)
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d105.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d105.Imm)
						ptrWord, _ := d105.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d105.Imm.String())))
						d105 = tmpPair
					} else if d105.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d105.Type, Reg: ctx.AllocRegExcept(d105.Reg), Reg2: ctx.AllocRegExcept(d105.Reg)}
						switch d105.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d105)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d105)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d105)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d105)
						d105 = tmpPair
					}
					if d105.Loc != LocRegPair && d105.Loc != LocStackPair && d105.Loc != LocInputPair {
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
					ctx.SyncDesc(&d105)
					ctx.SyncDesc(&d11)
					d107 = ctx.EmitGoCallScalar(GoFuncAddr(StrLike), []JITValueDesc{d105, d11}, 1)
					d107.NoHeapPointer = true
					ctx.EmitAndRegImm32(d107.Reg, 1)
					d107.Type = tagBool
					ctx.BindReg(d107.Reg, &d107)
					ctx.SyncDesc(&d107)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d107) {
						return d107
					}
					if d107.Loc == LocImm {
						ctx.EmitMakeBool(result, d107)
					} else {
						ctx.EmitMovToReg(result.Reg2, d107)
						d108 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d108)
						if d107.Loc == LocReg && d107.Reg != result.Reg2 {
							ctx.FreeReg(d107.Reg)
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
					ctx.SyncDesc(&d73)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d73) {
						return d73
					}
					if d73.Loc == LocImm {
						ctx.EmitMakeBool(result, d73)
					} else {
						ctx.EmitMovToReg(result.Reg2, d73)
						d109 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d109)
						if d73.Loc == LocReg && d73.Reg != result.Reg2 {
							ctx.FreeReg(d73.Reg)
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
			JITInlineCost: 38,
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl8)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl6)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextCase), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.ToLower arg0)")
					}
					ctx.SyncDesc(&d26)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToLower), []JITValueDesc{d26}, 2)
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
			JITInlineCost: 21,
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl8)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl6)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextCase), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d26)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d26}, 2)
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
			JITInlineCost: 21,
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
					panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg0)")
				}
				ctx.EnsureDesc(&d4)
				if d4.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d4.Imm)
					ptrWord, _ := d4.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d4.Imm.String())))
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
				if d4.Loc != LocRegPair && d4.Loc != LocStackPair && d4.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg1)")
				}
				ctx.EnsureDesc(&d7)
				if d7.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d7.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d7.Imm)
					ptrWord, _ := d7.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d7.Imm.String())))
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
				if d7.Loc != LocRegPair && d7.Loc != LocStackPair && d7.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (strings.ReplaceAll arg2)")
				}
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d4)
				ctx.SyncDesc(&d7)
				d9 := ctx.EmitGoCallScalar(GoFuncAddr(strings.ReplaceAll), []JITValueDesc{d1, d4, d7}, 2)
				d9.NoHeapPointer = false
				ctx.BindReg(d9.Reg, &d9)
				ctx.BindReg(d9.Reg2, &d9)
				ctx.EnsureDesc(&d9)
				d10 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d9}, 2)
				if result.Loc == LocAny {
					return d10
				}
				ctx.EmitMovPairToResult(&d10, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 12,
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl8)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl6)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d26)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d26}, 2)
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
			JITInlineCost: 21,
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl8)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl6)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d28 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
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
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d28)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d26, d28}, 2)
					d29.NoHeapPointer = false
					ctx.BindReg(d29.Reg, &d29)
					ctx.BindReg(d29.Reg2, &d29)
					ctx.FreeDesc(&d28)
					ctx.EnsureDesc(&d29)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d29}, 2)
					ctx.EmitMovPairToResult(&d30, &result)
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
			JITInlineCost: 21,
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl6)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl8)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl6)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl4)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = args[0]
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
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d28 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
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
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d28)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d26, d28}, 2)
					d29.NoHeapPointer = false
					ctx.BindReg(d29.Reg, &d29)
					ctx.BindReg(d29.Reg2, &d29)
					ctx.FreeDesc(&d28)
					ctx.EnsureDesc(&d29)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d29}, 2)
					ctx.EmitMovPairToResult(&d30, &result)
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
			JITInlineCost: 21,
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
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl10)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl10)
						ctx.EmitJmp(lbl11)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl10)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl8)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d15.Reg, 20)
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d18)
					}
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(phiBase12)+int32(0))
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d19 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d19)
					ctx.BindReg(r1, &d19)
					ctx.FreeDesc(&d10)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap21 := d0
						snap22 := d1
						snap23 := d2
						snap24 := d3
						snap25 := d9
						snap26 := d10
						snap27 := d11
						snap28 := d13
						snap29 := d14
						snap30 := d15
						snap31 := d16
						snap32 := d17
						snap33 := d18
						snap34 := d19
						snap35 := d20
						alloc36 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc36)
						d0 = snap21
						d1 = snap22
						d2 = snap23
						d3 = snap24
						d9 = snap25
						d10 = snap26
						d11 = snap27
						d13 = snap28
						d14 = snap29
						d15 = snap30
						d16 = snap31
						d17 = snap32
						d18 = snap33
						d19 = snap34
						d20 = snap35
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d37 = args[0]
					d37.ID = 0
					d37 = JITPrepareScmerGoArg(ctx, d37)
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d38.Loc == LocRegPair || d38.Loc == LocStackPair || d38.Loc == LocRegTriple || d38.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d38)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d37, d38}, 2)
					d39.NoHeapPointer = false
					ctx.BindReg(d39.Reg, &d39)
					ctx.BindReg(d39.Reg2, &d39)
					ctx.FreeDesc(&d38)
					ctx.FreeDesc(&d37)
					ctx.SyncDesc(&d39)
					if d39.Loc == LocRegPair || d39.Loc == LocStackPair || d39.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d39, &result)
						result.Type = d39.Type
					} else {
						switch d39.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d39)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d39)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d39)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d39, &result)
							result.Type = d39.Type
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
					d40 = args[0]
					d40.ID = 0
					d42 = d40
					ctx.SyncDesc(&d42)
					if d42.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d42.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d42.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d42 = tmpScalar
					}
					d42 = JITPrepareScmerGoArg(ctx, d42)
					if d42.Loc != LocRegPair && d42.Loc != LocStackPair && d42.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d42}, 2)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d41.Imm)
						ptrWord, _ := d41.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d41.Imm.String())))
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
					if d41.Loc != LocRegPair && d41.Loc != LocStackPair && d41.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimSpace arg0)")
					}
					ctx.SyncDesc(&d41)
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimSpace), []JITValueDesc{d41}, 2)
					d43.NoHeapPointer = false
					ctx.BindReg(d43.Reg, &d43)
					ctx.BindReg(d43.Reg2, &d43)
					ctx.EnsureDesc(&d43)
					d44 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d43}, 2)
					ctx.EmitMovPairToResult(&d44, &result)
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
			JITInlineCost: 27,
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
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl10)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl10)
						ctx.EmitJmp(lbl11)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl10)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl8)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d15.Reg, 20)
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d18)
					}
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(phiBase12)+int32(0))
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d19 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d19)
					ctx.BindReg(r1, &d19)
					ctx.FreeDesc(&d10)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap21 := d0
						snap22 := d1
						snap23 := d2
						snap24 := d3
						snap25 := d9
						snap26 := d10
						snap27 := d11
						snap28 := d13
						snap29 := d14
						snap30 := d15
						snap31 := d16
						snap32 := d17
						snap33 := d18
						snap34 := d19
						snap35 := d20
						alloc36 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc36)
						d0 = snap21
						d1 = snap22
						d2 = snap23
						d3 = snap24
						d9 = snap25
						d10 = snap26
						d11 = snap27
						d13 = snap28
						d14 = snap29
						d15 = snap30
						d16 = snap31
						d17 = snap32
						d18 = snap33
						d19 = snap34
						d20 = snap35
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d37 = args[0]
					d37.ID = 0
					d37 = JITPrepareScmerGoArg(ctx, d37)
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					if d38.Loc == LocRegPair || d38.Loc == LocStackPair || d38.Loc == LocRegTriple || d38.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d38)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d37, d38}, 2)
					d39.NoHeapPointer = false
					ctx.BindReg(d39.Reg, &d39)
					ctx.BindReg(d39.Reg2, &d39)
					ctx.FreeDesc(&d38)
					ctx.FreeDesc(&d37)
					ctx.SyncDesc(&d39)
					if d39.Loc == LocRegPair || d39.Loc == LocStackPair || d39.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d39, &result)
						result.Type = d39.Type
					} else {
						switch d39.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d39)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d39)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d39)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d39, &result)
							result.Type = d39.Type
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
					d40 = args[0]
					d40.ID = 0
					d42 = d40
					ctx.SyncDesc(&d42)
					if d42.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d42.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d42.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d42 = tmpScalar
					}
					d42 = JITPrepareScmerGoArg(ctx, d42)
					if d42.Loc != LocRegPair && d42.Loc != LocStackPair && d42.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d42}, 2)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d41.Imm)
						ptrWord, _ := d41.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d41.Imm.String())))
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
					if d41.Loc != LocRegPair && d41.Loc != LocStackPair && d41.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg0)")
					}
					d43 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
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
						panic("jit: generic call arg expects 2-word value (strings.TrimLeft arg1)")
					}
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d43)
					d44 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimLeft), []JITValueDesc{d41, d43}, 2)
					d44.NoHeapPointer = false
					ctx.BindReg(d44.Reg, &d44)
					ctx.BindReg(d44.Reg2, &d44)
					ctx.FreeDesc(&d43)
					ctx.EnsureDesc(&d44)
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d44}, 2)
					ctx.EmitMovPairToResult(&d45, &result)
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
			JITInlineCost: 27,
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
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							ctx.MarkLabel(lbl10)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d17.Condition, lbl10)
						ctx.EmitJmp(lbl11)
						ctx.FreeDesc(&d16)
						ctx.MarkLabel(lbl10)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase12)+int32(0))
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl8)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase12) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d15.Reg, 20)
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d18)
					}
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(phiBase12)+int32(0))
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl6)
					d19 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d19)
					ctx.BindReg(r1, &d19)
					ctx.FreeDesc(&d10)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap21 := d0
						snap22 := d1
						snap23 := d2
						snap24 := d3
						snap25 := d9
						snap26 := d10
						snap27 := d11
						snap28 := d13
						snap29 := d14
						snap30 := d15
						snap31 := d16
						snap32 := d17
						snap33 := d18
						snap34 := d19
						snap35 := d20
						alloc36 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc36)
						d0 = snap21
						d1 = snap22
						d2 = snap23
						d3 = snap24
						d9 = snap25
						d10 = snap26
						d11 = snap27
						d13 = snap28
						d14 = snap29
						d15 = snap30
						d16 = snap31
						d17 = snap32
						d18 = snap33
						d19 = snap34
						d20 = snap35
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d37 = args[0]
					d37.ID = 0
					d37 = JITPrepareScmerGoArg(ctx, d37)
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d38.Loc == LocRegPair || d38.Loc == LocStackPair || d38.Loc == LocRegTriple || d38.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d38)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextTrim), []JITValueDesc{d37, d38}, 2)
					d39.NoHeapPointer = false
					ctx.BindReg(d39.Reg, &d39)
					ctx.BindReg(d39.Reg2, &d39)
					ctx.FreeDesc(&d38)
					ctx.FreeDesc(&d37)
					ctx.SyncDesc(&d39)
					if d39.Loc == LocRegPair || d39.Loc == LocStackPair || d39.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d39, &result)
						result.Type = d39.Type
					} else {
						switch d39.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d39)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d39)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d39)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d39, &result)
							result.Type = d39.Type
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
					d40 = args[0]
					d40.ID = 0
					d42 = d40
					ctx.SyncDesc(&d42)
					if d42.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d42.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d42.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d42 = tmpScalar
					}
					d42 = JITPrepareScmerGoArg(ctx, d42)
					if d42.Loc != LocRegPair && d42.Loc != LocStackPair && d42.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d42}, 2)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d41.Imm)
						ptrWord, _ := d41.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d41.Imm.String())))
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
					if d41.Loc != LocRegPair && d41.Loc != LocStackPair && d41.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg0)")
					}
					d43 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(" \t\n\r")}
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
						panic("jit: generic call arg expects 2-word value (strings.TrimRight arg1)")
					}
					ctx.SyncDesc(&d41)
					ctx.SyncDesc(&d43)
					d44 = ctx.EmitGoCallScalar(GoFuncAddr(strings.TrimRight), []JITValueDesc{d41, d43}, 2)
					d44.NoHeapPointer = false
					ctx.BindReg(d44.Reg, &d44)
					ctx.BindReg(d44.Reg2, &d44)
					ctx.FreeDesc(&d43)
					ctx.EnsureDesc(&d44)
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d44}, 2)
					ctx.EmitMovPairToResult(&d45, &result)
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
			JITInlineCost: 27,
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
					ctx.EnsureDesc(&d5)
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
					ctx.EnsureDesc(&d32)
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
					ctx.EnsureDesc(&d15)
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
								if isCompressedText(a[0]) || a[1].GetTag() == tagCString {
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
							if isCompressedText(a[0]) || a[1].GetTag() == tagCString {
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
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
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
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d2)
				var d4 JITValueDesc
				if d2.Loc == LocImm {
					ptr, _ := d2.Imm.RawWords()
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(ptr))))}
				} else if d2.Loc == LocStackPair {
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d2.StackOff}
				} else {
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocRegPair {
						panic("StringData requires a Go string pair")
					}
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg, ID: 0}
				}
				ctx.ReclaimUntrackedRegs()
				var d5 JITValueDesc
				if d2.SliceSizeKnown {
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d2.KnownSliceLen))}
				} else if d2.Loc == LocImm {
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d2.Imm.String())))}
				} else if d2.Loc == LocStackTriple {
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d2.StackOff + 8, NoHeapPointer: true}
				} else if d2.Loc == LocStackPair {
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d2.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocRegPair || d2.Loc == LocRegTriple {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg2, ID: 0}
					} else if d2.Loc == LocReg {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocRegTriple || d4.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				if d5.Loc == LocRegPair || d5.Loc == LocStackPair || d5.Loc == LocRegTriple || d5.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d6 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if d6.Loc == LocRegPair || d6.Loc == LocStackPair || d6.Loc == LocRegTriple || d6.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d7 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocRegTriple || d7.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d4)
				ctx.SyncDesc(&d5)
				ctx.SyncDesc(&d6)
				ctx.SyncDesc(&d7)
				d8 := ctx.EmitGoCallScalar(GoFuncAddr(NewBString), []JITValueDesc{d4, d5, d6, d7}, 2)
				d8.NoHeapPointer = false
				ctx.BindReg(d8.Reg, &d8)
				ctx.BindReg(d8.Reg2, &d8)
				ctx.FreeDesc(&d6)
				ctx.FreeDesc(&d7)
				ctx.FreeDesc(&d4)
				ctx.FreeDesc(&d5)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d8)
				ctx.FreeDesc(&d0)
				if d8.Loc == LocImm {
					if result.Loc == LocAny {
						return d8
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d8)
				if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d8, &result)
					result.Type = d8.Type
				} else {
					switch d8.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d8)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d8)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d8)
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
			JITInlineCost:  9,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "base64_decode",

		Fn: func(a ...Scmer) Scmer {
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
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
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
				var d60 JITValueDesc
				_ = d60
				var d62 JITValueDesc
				_ = d62
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
					d0 = JITPrepareScmerGoArg(ctx, d0)
					ctx.SyncDesc(&d0)
					d1 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsBString), []JITValueDesc{d0}, 1)
					d1.NoHeapPointer = true
					ctx.EmitAndRegImm32(d1.Reg, 1)
					d1.Type = tagBool
					ctx.BindReg(d1.Reg, &d1)
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
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
					d7 = args[0]
					d7.ID = 0
					d7 = JITPrepareScmerGoArg(ctx, d7)
					ctx.SyncDesc(&d7)
					d8 = ctx.EmitGoCallScalar(GoFuncAddr(bstringRawText), []JITValueDesc{d7}, 2)
					d8.NoHeapPointer = false
					ctx.BindReg(d8.Reg, &d8)
					ctx.BindReg(d8.Reg2, &d8)
					ctx.FreeDesc(&d7)
					ctx.SyncDesc(&d8)
					if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d8, &result)
						result.Type = d8.Type
					} else {
						switch d8.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d8)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d8)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d8)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d8, &result)
							result.Type = d8.Type
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
					d9 = ctx.EmitGoCallScalar(GoFuncAddr(func() *base64.Encoding { return base64.StdEncoding }), nil, 1)
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
					if d9.Loc == LocRegPair || d9.Loc == LocStackPair || d9.Loc == LocRegTriple || d9.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
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
						panic("jit: generic call arg expects 2-word value ((*base64.Encoding).DecodeString arg1)")
					}
					ctx.SyncDesc(&d9)
					ctx.SyncDesc(&d11)
					callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr((*base64.Encoding).DecodeString), []JITValueDesc{d9, d11}, []uint8{3, 2}, []uint8{1, 3})
					d14 = callResults13[0]
					_ = d14
					d15 = callResults13[1]
					_ = d15
					ctx.FreeDesc(&d9)
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
						snap21 := d7
						snap22 := d8
						snap23 := d9
						snap24 := d10
						snap25 := d11
						snap26 := d12
						snap27 := d14
						snap28 := d15
						snap29 := d16
						snap30 := d17
						alloc31 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc31)
						d0 = snap18
						d1 = snap19
						d2 = snap20
						d7 = snap21
						d8 = snap22
						d9 = snap23
						d10 = snap24
						d11 = snap25
						d12 = snap26
						d14 = snap27
						d15 = snap28
						d16 = snap29
						d17 = snap30
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
					d32 = args[0]
					d32.ID = 0
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocImm {
						_, auxWord := d32.Imm.RawWords()
						d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d32.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r1 := ctx.AllocReg()
						ctx.EmitMovRegReg(r1, d32.Reg2)
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d33)
					}
					ctx.EnsureDesc(&d33)
					d34 = d33
					_ = d34
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d34.Imm.Int()) >> 8))}
					} else {
						r2 := ctx.AllocRegExcept(d34.Reg)
						ctx.EmitMovRegReg(r2, d34.Reg)
						ctx.EmitShrRegImm8(r2, 8)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d35)
					}
					if d35.Loc == LocReg && d34.Loc == LocReg && d35.Reg == d34.Reg {
						ctx.TransferReg(d34.Reg)
						d34.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d35)
					ctx.FreeDesc(&d33)
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d35.Imm.Int()) >> 46))}
					} else {
						ctx.EmitShrRegImm8(d35.Reg, 46)
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg}
						ctx.BindReg(d35.Reg, &d36)
					}
					if d36.Loc == LocReg && d35.Loc == LocReg && d36.Reg == d35.Reg {
						ctx.TransferReg(d35.Reg)
						d35.Loc = LocNone
					}
					ctx.FreeDesc(&d35)
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d36.Imm.Int()) == uint64(0x0))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d36.Reg, 0)
						d37 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
						ctx.BindReg(r3, &d37)
					}
					ctx.FreeDesc(&d36)
					d38 = d37
					ctx.EnsureDesc(&d38)
					if d38.Loc != LocImm && d38.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d38.Loc == LocImm {
						if d38.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d38.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d37)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap39 := d0
						snap40 := d1
						snap41 := d2
						snap42 := d7
						snap43 := d8
						snap44 := d9
						snap45 := d10
						snap46 := d11
						snap47 := d12
						snap48 := d14
						snap49 := d15
						snap50 := d16
						snap51 := d17
						snap52 := d32
						snap53 := d33
						snap54 := d34
						snap55 := d35
						snap56 := d36
						snap57 := d37
						snap58 := d38
						alloc59 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc59)
						d0 = snap39
						d1 = snap40
						d2 = snap41
						d7 = snap42
						d8 = snap43
						d9 = snap44
						d10 = snap45
						d11 = snap46
						d12 = snap47
						d14 = snap48
						d15 = snap49
						d16 = snap50
						d17 = snap51
						d32 = snap52
						d33 = snap53
						d34 = snap54
						d35 = snap55
						d36 = snap56
						d37 = snap57
						d38 = snap58
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
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["base64_decode"].Fn, args, result)
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
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					callResults61 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d14}, []uint8{2}, []uint8{1})
					d60 = callResults61[0]
					ctx.EnsureDesc(&d60)
					d62 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d60}, 2)
					ctx.EmitMovPairToResult(&d62, &result)
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
			JITInlineCost:  38,
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
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
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
				var d88 JITValueDesc
				_ = d88
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
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl9)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d7.Reg, 20)
						r2 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase4)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d2)
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
						snap13 := d1
						snap14 := d2
						snap15 := d3
						snap16 := d5
						snap17 := d6
						snap18 := d7
						snap19 := d8
						snap20 := d9
						snap21 := d10
						snap22 := d11
						snap23 := d12
						alloc24 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc24)
						d1 = snap13
						d2 = snap14
						d3 = snap15
						d5 = snap16
						d6 = snap17
						d7 = snap18
						d8 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
						d12 = snap23
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d25 = args[0]
					d25.ID = 0
					d25 = JITPrepareScmerGoArg(ctx, d25)
					ctx.SyncDesc(&d25)
					d26 = ctx.EmitGoCallScalar(GoFuncAddr(encodeTextHex), []JITValueDesc{d25}, 2)
					d26.NoHeapPointer = false
					ctx.BindReg(d26.Reg, &d26)
					ctx.BindReg(d26.Reg2, &d26)
					ctx.FreeDesc(&d25)
					ctx.SyncDesc(&d26)
					if d26.Loc == LocRegPair || d26.Loc == LocStackPair || d26.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d26, &result)
						result.Type = d26.Type
					} else {
						switch d26.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d26)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d26)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d26)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d26, &result)
							result.Type = d26.Type
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
					ctx.StabilizeDescForControlFlow(&d28)
					ctx.FreeDesc(&d27)
					if d28.SliceSizeKnown {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.KnownSliceLen))}
					} else if d28.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d28.Imm.String())))}
					} else if d28.Loc == LocStackTriple {
						d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else if d28.Loc == LocStackPair {
						d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d28)
						if d28.Loc == LocRegPair || d28.Loc == LocRegTriple {
							d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg2, ID: 0}
						} else if d28.Loc == LocReg {
							d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d30)
					ctx.SyncDesc(&d31)
					ctx.SyncDesc(&d30)
					if d31.Loc == LocImm && d30.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int() * d30.Imm.Int())}
					} else if d31.Loc == LocImm {
						ctx.EnsureDesc(&d30)
						scratch := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitMovRegReg(scratch, d30.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d31.Imm.Int())
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d32)
					} else if d30.Loc == LocImm {
						ctx.EnsureDesc(&d31)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d31.Reg, d30.Imm.Int())
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
						ctx.BindReg(d31.Reg, &d32)
					} else {
						ctx.EnsureDesc(&d31)
						ctx.SyncDesc(&d30)
						ctx.EmitIntBinary(JITIntMul, 64, d31.Reg, &d30)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
						ctx.BindReg(d31.Reg, &d32)
					}
					if d32.Loc == LocReg && d31.Loc == LocReg && d32.Reg == d31.Reg {
						ctx.TransferReg(d31.Reg)
						d31.Loc = LocNone
					}
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					callResults33 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d32, d32}, []uint8{3}, []uint8{1})
					d34 = callResults33[0]
					d34.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.FreeDesc(&d32)
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
					if d28.SliceSizeKnown {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.KnownSliceLen))}
					} else if d28.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d28.Imm.String())))}
					} else if d28.Loc == LocStackTriple {
						d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else if d28.Loc == LocStackPair {
						d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d28)
						if d28.Loc == LocRegPair || d28.Loc == LocRegTriple {
							d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg2, ID: 0}
						} else if d28.Loc == LocReg {
							d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDescsTogether(&d1, &d35)
					if d1.Loc == LocImm && d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d35.Imm.Int())}
					} else if d35.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d1.Reg)
						if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d35.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d35.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d36)
					} else if d1.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d35.Reg)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d36)
					} else {
						r5 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d35.Reg)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d36)
					}
					ctx.FreeDesc(&d35)
					d37 = d36
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocImm && d37.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d37.Loc == LocImm {
						if d37.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d37.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d36)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap38 := d1
						snap39 := d2
						snap40 := d3
						snap41 := d5
						snap42 := d6
						snap43 := d7
						snap44 := d8
						snap45 := d9
						snap46 := d10
						snap47 := d11
						snap48 := d12
						snap49 := d25
						snap50 := d26
						snap51 := d27
						snap52 := d28
						snap53 := d29
						snap54 := d30
						snap55 := d31
						snap56 := d32
						snap57 := d34
						snap58 := d35
						snap59 := d36
						snap60 := d37
						alloc61 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc61)
						d1 = snap38
						d2 = snap39
						d3 = snap40
						d5 = snap41
						d6 = snap42
						d7 = snap43
						d8 = snap44
						d9 = snap45
						d10 = snap46
						d11 = snap47
						d12 = snap48
						d25 = snap49
						d26 = snap50
						d27 = snap51
						d28 = snap52
						d29 = snap53
						d30 = snap54
						d31 = snap55
						d32 = snap56
						d34 = snap57
						d35 = snap58
						d36 = snap59
						d37 = snap60
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
					ctx.StabilizeDescForControlFlow(&d34)
					d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d62)
					ctx.SyncDesc(&d1)
					if d62.Loc == LocImm && d1.Loc == LocImm {
						d63 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d62.Imm.Int() * d1.Imm.Int())}
					} else if d62.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d62.Imm.Int())
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d63)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d62)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d62.Reg, d1.Imm.Int())
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d62.Reg}
						ctx.BindReg(d62.Reg, &d63)
					} else {
						ctx.EnsureDesc(&d62)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d62.Reg, &d1)
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d62.Reg}
						ctx.BindReg(d62.Reg, &d63)
					}
					if d63.Loc == LocReg && d62.Loc == LocReg && d63.Reg == d62.Reg {
						ctx.TransferReg(d62.Reg)
						d62.Loc = LocNone
					}
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d28)
					d64 = ctx.EmitSliceElementAddress(&d28, &d1, 1)
					ctx.EnsureDesc(&d64)
					r6 := ctx.AllocRegExcept(d64.Reg)
					ctx.EmitMovRegMemB(r6, d64.Reg, 0)
					ctx.FreeDesc(&d64)
					d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d65)
					ctx.BindReg(r6, &d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d65.Reg, 4)
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg}
						ctx.BindReg(d65.Reg, &d66)
					}
					if d66.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: d66.Type, Imm: NewInt(int64(uint64(d66.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d66.Reg, 56)
						ctx.EmitShrRegImm8(d66.Reg, 56)
					}
					if d66.Loc == LocReg && d65.Loc == LocReg && d66.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.FreeDesc(&d65)
					d67 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d66)
					ctx.EnsureGoStringHeader(&d67)
					d68 = ctx.EmitSliceElementAddress(&d67, &d66, 1)
					ctx.EnsureDesc(&d68)
					r7 := ctx.AllocRegExcept(d68.Reg)
					ctx.EmitMovRegMemB(r7, d68.Reg, 0)
					ctx.FreeDesc(&d68)
					d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d69)
					ctx.BindReg(r7, &d69)
					ctx.FreeDesc(&d66)
					ctx.EnsureDesc(&d63)
					ctx.SyncDesc(&d69)
					d70 = d34
					d70.ID = 0
					d71 = d63
					d71.ID = 0
					d72 = ctx.EmitSliceElementAddress(&d70, &d71, int32(1))
					ctx.EmitStoreScalarAt(&d72, &d69, 1)
					ctx.FreeDesc(&d72)
					ctx.FreeDesc(&d71)
					ctx.FreeDesc(&d63)
					ctx.FreeDesc(&d69)
					d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d73)
					ctx.SyncDesc(&d1)
					if d73.Loc == LocImm && d1.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() * d1.Imm.Int())}
					} else if d73.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d73.Imm.Int())
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d74)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d73)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d73.Reg, d1.Imm.Int())
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d74)
					} else {
						ctx.EnsureDesc(&d73)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d73.Reg, &d1)
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d74)
					}
					if d74.Loc == LocReg && d73.Loc == LocReg && d74.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d74.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d74.Reg)
						ctx.EmitMovRegReg(scratch, d74.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d75)
					}
					if d75.Loc == LocReg && d74.Loc == LocReg && d75.Reg == d74.Reg {
						ctx.TransferReg(d74.Reg)
						d74.Loc = LocNone
					}
					ctx.FreeDesc(&d74)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d28)
					d76 = ctx.EmitSliceElementAddress(&d28, &d1, 1)
					ctx.EnsureDesc(&d76)
					r8 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMemB(r8, d76.Reg, 0)
					ctx.FreeDesc(&d76)
					d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8, NoHeapPointer: true}
					ctx.BindReg(r8, &d77)
					ctx.BindReg(r8, &d77)
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d77.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d77.Reg, 15)
						d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d77.Reg}
						ctx.BindReg(d77.Reg, &d78)
					}
					if d78.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: d78.Type, Imm: NewInt(int64(uint64(d78.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d78.Reg, 56)
						ctx.EmitShrRegImm8(d78.Reg, 56)
					}
					if d78.Loc == LocReg && d77.Loc == LocReg && d78.Reg == d77.Reg {
						ctx.TransferReg(d77.Reg)
						d77.Loc = LocNone
					}
					ctx.FreeDesc(&d77)
					d79 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d78)
					ctx.EnsureGoStringHeader(&d79)
					d80 = ctx.EmitSliceElementAddress(&d79, &d78, 1)
					ctx.EnsureDesc(&d80)
					r9 := ctx.AllocRegExcept(d80.Reg)
					ctx.EmitMovRegMemB(r9, d80.Reg, 0)
					ctx.FreeDesc(&d80)
					d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9, NoHeapPointer: true}
					ctx.BindReg(r9, &d81)
					ctx.BindReg(r9, &d81)
					ctx.FreeDesc(&d78)
					ctx.EnsureDesc(&d75)
					ctx.SyncDesc(&d81)
					d82 = d34
					d82.ID = 0
					d83 = d75
					d83.ID = 0
					d84 = ctx.EmitSliceElementAddress(&d82, &d83, int32(1))
					ctx.EmitStoreScalarAt(&d84, &d81, 1)
					ctx.FreeDesc(&d84)
					ctx.FreeDesc(&d83)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d81)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d85 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d85)
					}
					if d85.Loc == LocReg && d1.Loc == LocReg && d85.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d85)
					ctx.EmitStoreToStack(d85, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d85)
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
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					callResults87 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d34}, []uint8{2}, []uint8{1})
					d86 = callResults87[0]
					ctx.EnsureDesc(&d86)
					d88 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d86}, 2)
					ctx.EmitMovPairToResult(&d88, &result)
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
			JITInlineCost:  44,
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
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
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
				var d88 JITValueDesc
				_ = d88
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
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitJump(d9.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d8)
						ctx.MarkLabel(lbl11)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase4)+int32(0))
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl9)
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
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d5 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase4) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d7.Reg, 20)
						r2 := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d10)
					}
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(phiBase4)+int32(0))
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl7)
					d11 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d11)
					ctx.BindReg(r1, &d11)
					ctx.FreeDesc(&d2)
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
						snap13 := d1
						snap14 := d2
						snap15 := d3
						snap16 := d5
						snap17 := d6
						snap18 := d7
						snap19 := d8
						snap20 := d9
						snap21 := d10
						snap22 := d11
						snap23 := d12
						alloc24 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc24)
						d1 = snap13
						d2 = snap14
						d3 = snap15
						d5 = snap16
						d6 = snap17
						d7 = snap18
						d8 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
						d12 = snap23
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d25 = args[0]
					d25.ID = 0
					d25 = JITPrepareScmerGoArg(ctx, d25)
					ctx.SyncDesc(&d25)
					d26 = ctx.EmitGoCallScalar(GoFuncAddr(encodeTextHex), []JITValueDesc{d25}, 2)
					d26.NoHeapPointer = false
					ctx.BindReg(d26.Reg, &d26)
					ctx.BindReg(d26.Reg2, &d26)
					ctx.FreeDesc(&d25)
					ctx.SyncDesc(&d26)
					if d26.Loc == LocRegPair || d26.Loc == LocStackPair || d26.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d26, &result)
						result.Type = d26.Type
					} else {
						switch d26.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d26)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d26)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d26)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d26, &result)
							result.Type = d26.Type
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
					ctx.StabilizeDescForControlFlow(&d28)
					ctx.FreeDesc(&d27)
					if d28.SliceSizeKnown {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.KnownSliceLen))}
					} else if d28.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d28.Imm.String())))}
					} else if d28.Loc == LocStackTriple {
						d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else if d28.Loc == LocStackPair {
						d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d28)
						if d28.Loc == LocRegPair || d28.Loc == LocRegTriple {
							d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg2, ID: 0}
						} else if d28.Loc == LocReg {
							d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d30)
					ctx.SyncDesc(&d31)
					ctx.SyncDesc(&d30)
					if d31.Loc == LocImm && d30.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int() * d30.Imm.Int())}
					} else if d31.Loc == LocImm {
						ctx.EnsureDesc(&d30)
						scratch := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitMovRegReg(scratch, d30.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d31.Imm.Int())
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d32)
					} else if d30.Loc == LocImm {
						ctx.EnsureDesc(&d31)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d31.Reg, d30.Imm.Int())
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
						ctx.BindReg(d31.Reg, &d32)
					} else {
						ctx.EnsureDesc(&d31)
						ctx.SyncDesc(&d30)
						ctx.EmitIntBinary(JITIntMul, 64, d31.Reg, &d30)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
						ctx.BindReg(d31.Reg, &d32)
					}
					if d32.Loc == LocReg && d31.Loc == LocReg && d32.Reg == d31.Reg {
						ctx.TransferReg(d31.Reg)
						d31.Loc = LocNone
					}
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					callResults33 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeByteSlice), []JITValueDesc{d32, d32}, []uint8{3}, []uint8{1})
					d34 = callResults33[0]
					d34.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.FreeDesc(&d32)
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
					if d28.SliceSizeKnown {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.KnownSliceLen))}
					} else if d28.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d28.Imm.String())))}
					} else if d28.Loc == LocStackTriple {
						d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else if d28.Loc == LocStackPair {
						d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d28)
						if d28.Loc == LocRegPair || d28.Loc == LocRegTriple {
							d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg2, ID: 0}
						} else if d28.Loc == LocReg {
							d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDescsTogether(&d1, &d35)
					if d1.Loc == LocImm && d35.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d35.Imm.Int())}
					} else if d35.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d1.Reg)
						if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d35.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d35.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d36)
					} else if d1.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d35.Reg)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d36)
					} else {
						r5 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d35.Reg)
						d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d36)
					}
					ctx.FreeDesc(&d35)
					d37 = d36
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocImm && d37.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d37.Loc == LocImm {
						if d37.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d37.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d36)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap38 := d1
						snap39 := d2
						snap40 := d3
						snap41 := d5
						snap42 := d6
						snap43 := d7
						snap44 := d8
						snap45 := d9
						snap46 := d10
						snap47 := d11
						snap48 := d12
						snap49 := d25
						snap50 := d26
						snap51 := d27
						snap52 := d28
						snap53 := d29
						snap54 := d30
						snap55 := d31
						snap56 := d32
						snap57 := d34
						snap58 := d35
						snap59 := d36
						snap60 := d37
						alloc61 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc61)
						d1 = snap38
						d2 = snap39
						d3 = snap40
						d5 = snap41
						d6 = snap42
						d7 = snap43
						d8 = snap44
						d9 = snap45
						d10 = snap46
						d11 = snap47
						d12 = snap48
						d25 = snap49
						d26 = snap50
						d27 = snap51
						d28 = snap52
						d29 = snap53
						d30 = snap54
						d31 = snap55
						d32 = snap56
						d34 = snap57
						d35 = snap58
						d36 = snap59
						d37 = snap60
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
					ctx.StabilizeDescForControlFlow(&d34)
					d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d62)
					ctx.SyncDesc(&d1)
					if d62.Loc == LocImm && d1.Loc == LocImm {
						d63 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d62.Imm.Int() * d1.Imm.Int())}
					} else if d62.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d62.Imm.Int())
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d63)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d62)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d62.Reg, d1.Imm.Int())
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d62.Reg}
						ctx.BindReg(d62.Reg, &d63)
					} else {
						ctx.EnsureDesc(&d62)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d62.Reg, &d1)
						d63 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d62.Reg}
						ctx.BindReg(d62.Reg, &d63)
					}
					if d63.Loc == LocReg && d62.Loc == LocReg && d63.Reg == d62.Reg {
						ctx.TransferReg(d62.Reg)
						d62.Loc = LocNone
					}
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d28)
					d64 = ctx.EmitSliceElementAddress(&d28, &d1, 1)
					ctx.EnsureDesc(&d64)
					r6 := ctx.AllocRegExcept(d64.Reg)
					ctx.EmitMovRegMemB(r6, d64.Reg, 0)
					ctx.FreeDesc(&d64)
					d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6, NoHeapPointer: true}
					ctx.BindReg(r6, &d65)
					ctx.BindReg(r6, &d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() / 16)}
					} else {
						ctx.EmitShrRegImm8(d65.Reg, 4)
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg}
						ctx.BindReg(d65.Reg, &d66)
					}
					if d66.Loc == LocImm {
						d66 = JITValueDesc{Loc: LocImm, Type: d66.Type, Imm: NewInt(int64(uint64(d66.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d66.Reg, 56)
						ctx.EmitShrRegImm8(d66.Reg, 56)
					}
					if d66.Loc == LocReg && d65.Loc == LocReg && d66.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.FreeDesc(&d65)
					d67 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d66)
					ctx.EnsureGoStringHeader(&d67)
					d68 = ctx.EmitSliceElementAddress(&d67, &d66, 1)
					ctx.EnsureDesc(&d68)
					r7 := ctx.AllocRegExcept(d68.Reg)
					ctx.EmitMovRegMemB(r7, d68.Reg, 0)
					ctx.FreeDesc(&d68)
					d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7, NoHeapPointer: true}
					ctx.BindReg(r7, &d69)
					ctx.BindReg(r7, &d69)
					ctx.FreeDesc(&d66)
					ctx.EnsureDesc(&d63)
					ctx.SyncDesc(&d69)
					d70 = d34
					d70.ID = 0
					d71 = d63
					d71.ID = 0
					d72 = ctx.EmitSliceElementAddress(&d70, &d71, int32(1))
					ctx.EmitStoreScalarAt(&d72, &d69, 1)
					ctx.FreeDesc(&d72)
					ctx.FreeDesc(&d71)
					ctx.FreeDesc(&d63)
					ctx.FreeDesc(&d69)
					d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					ctx.EnsureDesc(&d1)
					ctx.SyncDesc(&d73)
					ctx.SyncDesc(&d1)
					if d73.Loc == LocImm && d1.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() * d1.Imm.Int())}
					} else if d73.Loc == LocImm {
						ctx.EnsureDesc(&d1)
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntMul, 64, scratch, d73.Imm.Int())
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d74)
					} else if d1.Loc == LocImm {
						ctx.EnsureDesc(&d73)
						ctx.EmitIntBinaryImm(JITIntMul, 64, d73.Reg, d1.Imm.Int())
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d74)
					} else {
						ctx.EnsureDesc(&d73)
						ctx.SyncDesc(&d1)
						ctx.EmitIntBinary(JITIntMul, 64, d73.Reg, &d1)
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d74)
					}
					if d74.Loc == LocReg && d73.Loc == LocReg && d74.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d74.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d74.Reg)
						ctx.EmitMovRegReg(scratch, d74.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d75)
					}
					if d75.Loc == LocReg && d74.Loc == LocReg && d75.Reg == d74.Reg {
						ctx.TransferReg(d74.Reg)
						d74.Loc = LocNone
					}
					ctx.FreeDesc(&d74)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d1)
					ctx.EnsureGoStringHeader(&d28)
					d76 = ctx.EmitSliceElementAddress(&d28, &d1, 1)
					ctx.EnsureDesc(&d76)
					r8 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMemB(r8, d76.Reg, 0)
					ctx.FreeDesc(&d76)
					d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8, NoHeapPointer: true}
					ctx.BindReg(r8, &d77)
					ctx.BindReg(r8, &d77)
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d77.Imm.Int() % 16)}
					} else {
						ctx.EmitAndRegImm32(d77.Reg, 15)
						d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d77.Reg}
						ctx.BindReg(d77.Reg, &d78)
					}
					if d78.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: d78.Type, Imm: NewInt(int64(uint64(d78.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d78.Reg, 56)
						ctx.EmitShrRegImm8(d78.Reg, 56)
					}
					if d78.Loc == LocReg && d77.Loc == LocReg && d78.Reg == d77.Reg {
						ctx.TransferReg(d77.Reg)
						d77.Loc = LocNone
					}
					ctx.FreeDesc(&d77)
					d79 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("0123456789abcdef")}
					ctx.EnsureDesc(&d78)
					ctx.EnsureGoStringHeader(&d79)
					d80 = ctx.EmitSliceElementAddress(&d79, &d78, 1)
					ctx.EnsureDesc(&d80)
					r9 := ctx.AllocRegExcept(d80.Reg)
					ctx.EmitMovRegMemB(r9, d80.Reg, 0)
					ctx.FreeDesc(&d80)
					d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9, NoHeapPointer: true}
					ctx.BindReg(r9, &d81)
					ctx.BindReg(r9, &d81)
					ctx.FreeDesc(&d78)
					ctx.EnsureDesc(&d75)
					ctx.SyncDesc(&d81)
					d82 = d34
					d82.ID = 0
					d83 = d75
					d83.ID = 0
					d84 = ctx.EmitSliceElementAddress(&d82, &d83, int32(1))
					ctx.EmitStoreScalarAt(&d84, &d81, 1)
					ctx.FreeDesc(&d84)
					ctx.FreeDesc(&d83)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d81)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d85 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d85)
					}
					if d85.Loc == LocReg && d1.Loc == LocReg && d85.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d85)
					ctx.EmitStoreToStack(d85, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d85)
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
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					callResults87 := JITEmitGoCallResults(ctx, GoFuncAddr(jitBytesToString), []JITValueDesc{d34}, []uint8{2}, []uint8{1})
					d86 = callResults87[0]
					ctx.EnsureDesc(&d86)
					d88 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d86}, 2)
					ctx.EmitMovPairToResult(&d88, &result)
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
			JITInlineCost:  44,
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
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
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
					callResults0 := JITEmitGoCallResults(ctx, GoFuncAddr(uuid.NewRandom), []JITValueDesc{}, []uint8{2, 2}, []uint8{0, 3})
					d1 = callResults0[0]
					_ = d1
					d2 = callResults0[1]
					_ = d2
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.EnsureDesc(&d2)
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d4.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap5 := d1
						snap6 := d2
						snap7 := d3
						snap8 := d4
						alloc9 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc9)
						d1 = snap5
						d2 = snap6
						d3 = snap7
						d4 = snap8
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["uuid"].Fn, args, result)
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
					d10 = ctx.EmitGoCallScalar(GoFuncAddr((uuid.UUID).String), []JITValueDesc{d1}, 2)
					d10.NoHeapPointer = false
					ctx.BindReg(d10.Reg, &d10)
					ctx.BindReg(d10.Reg2, &d10)
					ctx.EnsureDesc(&d10)
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d10}, 2)
					ctx.EmitMovPairToResult(&d11, &result)
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
					ctx.EnsureDesc(&d5)
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
					ctx.EnsureDesc(&d16)
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
			if isCompressedText(a[0]) && compressedTextLen(a[0]) >= 128 {
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
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl8)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl7)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
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
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl5)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d22 = ctx.EmitGoCallScalar(GoFuncAddr(func() *schemeTextWriter { return new(schemeTextWriter) }), nil, 1)
					ctx.BindReg(d22.Reg, &d22)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-3750763034362895579)}
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d23)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *schemeTextWriter, value uint64) { base.hash = value }), []JITValueDesc{d22, d23})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d22)
					d24 = args[0]
					d24.ID = 0
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocRegTriple || d22.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d24 = JITPrepareScmerGoArg(ctx, d24)
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d24)
					ctx.EmitGoCallVoid(GoFuncAddr((*schemeTextWriter).writeCompressedText), []JITValueDesc{d22, d24})
					ctx.FreeDesc(&d24)
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						fieldAddr := uintptr(d22.Imm.Int()) + 8
						r3 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r3, fieldAddr)
						d25 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d25)
					} else {
						off := int32(8)
						baseReg := d22.Reg
						r4 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r4, baseReg, off)
						d25 = JITValueDesc{Loc: LocReg, Reg: r4}
						ctx.BindReg(r4, &d25)
					}
					if d25.Loc == LocRegPair || d25.Loc == LocStackPair || d25.Loc == LocRegTriple || d25.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d25)
					d26 = ctx.EmitGoCallScalar(GoFuncAddr(formatStructuralHash), []JITValueDesc{d25}, 2)
					d26.NoHeapPointer = false
					ctx.BindReg(d26.Reg, &d26)
					ctx.BindReg(d26.Reg2, &d26)
					ctx.FreeDesc(&d25)
					ctx.EnsureDesc(&d26)
					d27 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d26}, 2)
					ctx.EmitMovPairToResult(&d27, &result)
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
					d28 = args[0]
					d28.ID = 0
					d30 = d28
					ctx.SyncDesc(&d30)
					if d30.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d30.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d30.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d30 = tmpScalar
					}
					d30 = JITPrepareScmerGoArg(ctx, d30)
					if d30.Loc != LocRegPair && d30.Loc != LocStackPair && d30.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d30}, 2)
					ctx.FreeDesc(&d28)
					ctx.EnsureDesc(&d29)
					if d29.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d29.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d29.Imm)
						ptrWord, _ := d29.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d29.Imm.String())))
						d29 = tmpPair
					} else if d29.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d29.Type, Reg: ctx.AllocRegExcept(d29.Reg), Reg2: ctx.AllocRegExcept(d29.Reg)}
						switch d29.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d29)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d29)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d29)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d29)
						d29 = tmpPair
					}
					if d29.Loc != LocRegPair && d29.Loc != LocStackPair && d29.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (fnvHashString arg0)")
					}
					ctx.SyncDesc(&d29)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(fnvHashString), []JITValueDesc{d29}, 2)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.EnsureDesc(&d31)
					d32 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d31}, 2)
					ctx.EmitMovPairToResult(&d32, &result)
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
					d33 = args[0]
					d33.ID = 0
					d33 = JITPrepareScmerGoArg(ctx, d33)
					ctx.SyncDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d33}, 1)
					d34.NoHeapPointer = true
					ctx.BindReg(d34.Reg, &d34)
					ctx.FreeDesc(&d33)
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d34.Imm.Int() >= 128)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d34.Reg, 128)
						d35 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r5, &d35)
					}
					ctx.FreeDesc(&d34)
					d36 = d35
					ctx.EnsureDesc(&d36)
					if d36.Loc != LocImm && d36.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d36.Loc == LocImm {
						if d36.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d36.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d35)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap37 := d0
						snap38 := d1
						snap39 := d3
						snap40 := d4
						snap41 := d5
						snap42 := d6
						snap43 := d7
						snap44 := d8
						snap45 := d9
						snap46 := d10
						snap47 := d22
						snap48 := d23
						snap49 := d24
						snap50 := d25
						snap51 := d26
						snap52 := d27
						snap53 := d28
						snap54 := d29
						snap55 := d30
						snap56 := d31
						snap57 := d32
						snap58 := d33
						snap59 := d34
						snap60 := d35
						snap61 := d36
						alloc62 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc62)
						d0 = snap37
						d1 = snap38
						d3 = snap39
						d4 = snap40
						d5 = snap41
						d6 = snap42
						d7 = snap43
						d8 = snap44
						d9 = snap45
						d10 = snap46
						d22 = snap47
						d23 = snap48
						d24 = snap49
						d25 = snap50
						d26 = snap51
						d27 = snap52
						d28 = snap53
						d29 = snap54
						d30 = snap55
						d31 = snap56
						d32 = snap57
						d33 = snap58
						d34 = snap59
						d35 = snap60
						d36 = snap61
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
			JITInlineCost: 35,
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
					ctx.EnsureDesc(&d2)
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
					ctx.EnsureDesc(&d11)
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
					ctx.EnsureDesc(&d23)
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
			if isCompressedText(a[0]) && compressedTextLen(a[0]) >= 256 {
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
				declaration := declarations["md5"]
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
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl8)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl7)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
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
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl5)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextDigest), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[16]byte { return new([16]byte) }), nil, 1)
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
					ctx.FreeDesc(&d26)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					callResults30 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d27}, []uint8{3}, []uint8{1})
					d29 = callResults30[0]
					d29.Type = tagSlice
					d29 = JITPrepareGoSliceArg(ctx, d29)
					if d29.Loc != LocRegTriple && d29.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (md5.Sum arg0)")
					}
					ctx.SyncDesc(&d29)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(md5.Sum), []JITValueDesc{d29}, 2)
					d31.NoHeapPointer = true
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.EnsureDesc(&d31)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[16]byte, src [16]byte) { *dst = src }), []JITValueDesc{d25, d31})
					sliceResults32 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[16]byte) []byte { return value[0:16:16] }), []JITValueDesc{d25}, []uint8{3}, []uint8{1})
					d33 = sliceResults32[0]
					d33 = JITPrepareGoSliceArg(ctx, d33)
					if d33.Loc != LocRegTriple && d33.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (hex.EncodeToString arg0)")
					}
					ctx.SyncDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(hex.EncodeToString), []JITValueDesc{d33}, 2)
					d34.NoHeapPointer = false
					ctx.BindReg(d34.Reg, &d34)
					ctx.BindReg(d34.Reg2, &d34)
					ctx.EnsureDesc(&d34)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d34}, 2)
					ctx.EmitMovPairToResult(&d35, &result)
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
					d36 = args[0]
					d36.ID = 0
					d36 = JITPrepareScmerGoArg(ctx, d36)
					ctx.SyncDesc(&d36)
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d36}, 1)
					d37.NoHeapPointer = true
					ctx.BindReg(d37.Reg, &d37)
					ctx.FreeDesc(&d36)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() >= 256)}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d37.Reg, 256)
						d38 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r3, &d38)
					}
					ctx.FreeDesc(&d37)
					d39 = d38
					ctx.EnsureDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d39.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d38)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap40 := d0
						snap41 := d1
						snap42 := d3
						snap43 := d4
						snap44 := d5
						snap45 := d6
						snap46 := d7
						snap47 := d8
						snap48 := d9
						snap49 := d10
						snap50 := d22
						snap51 := d23
						snap52 := d24
						snap53 := d25
						snap54 := d26
						snap55 := d27
						snap56 := d28
						snap57 := d29
						snap58 := d31
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d37
						snap64 := d38
						snap65 := d39
						alloc66 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc66)
						d0 = snap40
						d1 = snap41
						d3 = snap42
						d4 = snap43
						d5 = snap44
						d6 = snap45
						d7 = snap46
						d8 = snap47
						d9 = snap48
						d10 = snap49
						d22 = snap50
						d23 = snap51
						d24 = snap52
						d25 = snap53
						d26 = snap54
						d27 = snap55
						d28 = snap56
						d29 = snap57
						d31 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d37 = snap63
						d38 = snap64
						d39 = snap65
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
			JITInlineCost:      31,
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sha1",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) && compressedTextLen(a[0]) >= 256 {
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
				declaration := declarations["sha1"]
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
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl8)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl7)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
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
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl5)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextDigest), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[20]byte { return new([20]byte) }), nil, 1)
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
					ctx.FreeDesc(&d26)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					callResults30 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d27}, []uint8{3}, []uint8{1})
					d29 = callResults30[0]
					d29.Type = tagSlice
					d29 = JITPrepareGoSliceArg(ctx, d29)
					if d29.Loc != LocRegTriple && d29.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (sha1.Sum arg0)")
					}
					ctx.SyncDesc(&d29)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(sha1.Sum), []JITValueDesc{d29}, 3)
					d31.NoHeapPointer = true
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.BindReg(d31.Reg3, &d31)
					ctx.EnsureDesc(&d31)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[20]byte, src [20]byte) { *dst = src }), []JITValueDesc{d25, d31})
					sliceResults32 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[20]byte) []byte { return value[0:20:20] }), []JITValueDesc{d25}, []uint8{3}, []uint8{1})
					d33 = sliceResults32[0]
					d33 = JITPrepareGoSliceArg(ctx, d33)
					if d33.Loc != LocRegTriple && d33.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (hex.EncodeToString arg0)")
					}
					ctx.SyncDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(hex.EncodeToString), []JITValueDesc{d33}, 2)
					d34.NoHeapPointer = false
					ctx.BindReg(d34.Reg, &d34)
					ctx.BindReg(d34.Reg2, &d34)
					ctx.EnsureDesc(&d34)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d34}, 2)
					ctx.EmitMovPairToResult(&d35, &result)
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
					d36 = args[0]
					d36.ID = 0
					d36 = JITPrepareScmerGoArg(ctx, d36)
					ctx.SyncDesc(&d36)
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d36}, 1)
					d37.NoHeapPointer = true
					ctx.BindReg(d37.Reg, &d37)
					ctx.FreeDesc(&d36)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() >= 256)}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d37.Reg, 256)
						d38 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r3, &d38)
					}
					ctx.FreeDesc(&d37)
					d39 = d38
					ctx.EnsureDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d39.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d38)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap40 := d0
						snap41 := d1
						snap42 := d3
						snap43 := d4
						snap44 := d5
						snap45 := d6
						snap46 := d7
						snap47 := d8
						snap48 := d9
						snap49 := d10
						snap50 := d22
						snap51 := d23
						snap52 := d24
						snap53 := d25
						snap54 := d26
						snap55 := d27
						snap56 := d28
						snap57 := d29
						snap58 := d31
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d37
						snap64 := d38
						snap65 := d39
						alloc66 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc66)
						d0 = snap40
						d1 = snap41
						d3 = snap42
						d4 = snap43
						d5 = snap44
						d6 = snap45
						d7 = snap46
						d8 = snap47
						d9 = snap48
						d10 = snap49
						d22 = snap50
						d23 = snap51
						d24 = snap52
						d25 = snap53
						d26 = snap54
						d27 = snap55
						d28 = snap56
						d29 = snap57
						d31 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d37 = snap63
						d38 = snap64
						d39 = snap65
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
			JITInlineCost:      31,
			JITInlineCallbacks: false,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sha256",

		Fn: func(a ...Scmer) Scmer {
			if isCompressedText(a[0]) && compressedTextLen(a[0]) >= 256 {
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
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
							ctx.EmitJmp(lbl8)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitJump(d7.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d6)
						ctx.MarkLabel(lbl9)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase2)+int32(0))
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl7)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
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
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase2) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0x14))}
					} else {
						ctx.EmitCmpRegImm32(d5.Reg, 20)
						r2 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitSetcc(r2, CondEqual)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d8)
					}
					ctx.EnsureDesc(&d8)
					ctx.EmitStoreToStack(d8, int32(phiBase2)+int32(0))
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl5)
					d9 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d9)
					ctx.BindReg(r1, &d9)
					ctx.FreeDesc(&d0)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap11 := d0
						snap12 := d1
						snap13 := d3
						snap14 := d4
						snap15 := d5
						snap16 := d6
						snap17 := d7
						snap18 := d8
						snap19 := d9
						snap20 := d10
						alloc21 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap11
						d1 = snap12
						d3 = snap13
						d4 = snap14
						d5 = snap15
						d6 = snap16
						d7 = snap17
						d8 = snap18
						d9 = snap19
						d10 = snap20
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d9)
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
					d22 = args[0]
					d22.ID = 0
					d22 = JITPrepareScmerGoArg(ctx, d22)
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocRegTriple || d23.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d23)
					d24 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextDigest), []JITValueDesc{d22, d23}, 2)
					d24.NoHeapPointer = false
					ctx.BindReg(d24.Reg, &d24)
					ctx.BindReg(d24.Reg2, &d24)
					ctx.FreeDesc(&d23)
					ctx.FreeDesc(&d22)
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
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[32]byte { return new([32]byte) }), nil, 1)
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
					ctx.FreeDesc(&d26)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					callResults30 := JITEmitGoCallResults(ctx, GoFuncAddr(jitStringToBytes), []JITValueDesc{d27}, []uint8{3}, []uint8{1})
					d29 = callResults30[0]
					d29.Type = tagSlice
					d29 = JITPrepareGoSliceArg(ctx, d29)
					if d29.Loc != LocRegTriple && d29.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (sha256.Sum256 arg0)")
					}
					ctx.SyncDesc(&d29)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr((func(arg0 []byte) *[32]byte { value := sha256.Sum256(arg0); return &value })), []JITValueDesc{d29}, 1)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.EnsureDesc(&d31)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *[32]byte) { *dst = *src }), []JITValueDesc{d25, d31})
					sliceResults32 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[32]byte) []byte { return value[0:32:32] }), []JITValueDesc{d25}, []uint8{3}, []uint8{1})
					d33 = sliceResults32[0]
					d33 = JITPrepareGoSliceArg(ctx, d33)
					if d33.Loc != LocRegTriple && d33.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (hex.EncodeToString arg0)")
					}
					ctx.SyncDesc(&d33)
					d34 = ctx.EmitGoCallScalar(GoFuncAddr(hex.EncodeToString), []JITValueDesc{d33}, 2)
					d34.NoHeapPointer = false
					ctx.BindReg(d34.Reg, &d34)
					ctx.BindReg(d34.Reg2, &d34)
					ctx.EnsureDesc(&d34)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d34}, 2)
					ctx.EmitMovPairToResult(&d35, &result)
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
					d36 = args[0]
					d36.ID = 0
					d36 = JITPrepareScmerGoArg(ctx, d36)
					ctx.SyncDesc(&d36)
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(compressedTextLen), []JITValueDesc{d36}, 1)
					d37.NoHeapPointer = true
					ctx.BindReg(d37.Reg, &d37)
					ctx.FreeDesc(&d36)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() >= 256)}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d37.Reg, 256)
						d38 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r3, &d38)
					}
					ctx.FreeDesc(&d37)
					d39 = d38
					ctx.EnsureDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d39.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d38)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap40 := d0
						snap41 := d1
						snap42 := d3
						snap43 := d4
						snap44 := d5
						snap45 := d6
						snap46 := d7
						snap47 := d8
						snap48 := d9
						snap49 := d10
						snap50 := d22
						snap51 := d23
						snap52 := d24
						snap53 := d25
						snap54 := d26
						snap55 := d27
						snap56 := d28
						snap57 := d29
						snap58 := d31
						snap59 := d33
						snap60 := d34
						snap61 := d35
						snap62 := d36
						snap63 := d37
						snap64 := d38
						snap65 := d39
						alloc66 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc66)
						d0 = snap40
						d1 = snap41
						d3 = snap42
						d4 = snap43
						d5 = snap44
						d6 = snap45
						d7 = snap46
						d8 = snap47
						d9 = snap48
						d10 = snap49
						d22 = snap50
						d23 = snap51
						d24 = snap52
						d25 = snap53
						d26 = snap54
						d27 = snap55
						d28 = snap56
						d29 = snap57
						d31 = snap58
						d33 = snap59
						d34 = snap60
						d35 = snap61
						d36 = snap62
						d37 = snap63
						d38 = snap64
						d39 = snap65
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
			JITInlineCost:      31,
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
