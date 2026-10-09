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

import "unsafe"
import "strings"

// nibbleAt reads the 4-bit nibble at absolute nibble index absIdx from ptr.
// absIdx = byte_offset*2 + nibble_within_byte (0=low nibble, 1=high nibble).
func nibbleAt(ptr *byte, absIdx int) byte {
	b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(absIdx>>1)))
	if absIdx&1 == 0 {
		return b & 0x0F
	}
	return b >> 4
}

// ptrOff advances ptr by n bytes using unsafe arithmetic.
func ptrOff(ptr *byte, n int) *byte {
	return (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(n)))
}

// nibbleRangeEqual compares charLen nibbles starting at nibOff (0 or 1) in ptrA and ptrB.
// Both pointers must have the same nibOff. Uses memcmp for inner aligned bytes and
// nibble-mask comparisons for the leading/trailing overhangs.
//
// Byte layout for nibOff=0 (first char = low nibble of ptr[0]):
//
//	Full bytes 0..charLen/2-1; trailing low-nibble overhang if charLen is odd.
//
// Byte layout for nibOff=1 (first char = high nibble of ptr[0]):
//
//	Leading high-nibble overhang at ptr[0]; full bytes 1..(charLen-1)/2;
//	trailing low-nibble overhang at ptr[charLen/2] if charLen is even.
func nibbleRangeEqual(ptrA, ptrB *byte, nibOff, charLen int) bool {
	if charLen == 0 {
		return true
	}
	if nibOff == 0 {
		// Inner full bytes 0..charLen/2-1.
		fullBytes := charLen / 2
		if fullBytes > 0 && unsafe.String(ptrA, fullBytes) != unsafe.String(ptrB, fullBytes) {
			return false
		}
		// Trailing overhang: low nibble of ptr[fullBytes] when charLen is odd.
		if charLen%2 == 1 {
			return *ptrOff(ptrA, fullBytes)&0x0F == *ptrOff(ptrB, fullBytes)&0x0F
		}
		return true
	}
	// nibOff == 1.
	// Leading overhang: high nibble of ptr[0].
	if *ptrA>>4 != *ptrB>>4 {
		return false
	}
	if charLen == 1 {
		return true
	}
	// Inner full bytes 1..(charLen-1)/2.
	innerCount := (charLen - 1) / 2
	if innerCount > 0 {
		if unsafe.String(ptrOff(ptrA, 1), innerCount) != unsafe.String(ptrOff(ptrB, 1), innerCount) {
			return false
		}
	}
	// Trailing overhang: low nibble of ptr[charLen/2] when charLen is even.
	if charLen%2 == 0 {
		return *ptrOff(ptrA, charLen/2)&0x0F == *ptrOff(ptrB, charLen/2)&0x0F
	}
	return true
}

// cstringIsNibble identifies the legacy 4-bit formats handled by cstringEqual.
// Ordered IDs are dispatched separately before this legacy path.
// Must stay in sync with storage.StringFormat constants:
//
//	1=Phone, 2=HexLower, 3=HexUpper, 8=Decimal, 9=DateTime, 10=PhoneDTMF
func cstringIsNibble(format uint8) bool {
	return format == 1 || format == 2 || format == 3 || format == 8 || format == 9 || format == 10
}

// cstringEqual compares two tagCString Scmers without materializing strings.
// Length mismatch rejects immediately; mixed formats compare decoded prefixes;
// matching formats compare packed nibbles or raw UUID bytes.
func cstringEqual(a, b Scmer) bool {
	aVal := auxVal(a.aux)
	bVal := auxVal(b.aux)
	aCharLen := int(aVal & CStringLengthMask)
	bCharLen := int(bVal & CStringLengthMask)
	if aCharLen != bCharLen {
		return false
	}
	aFmt := uint8(aVal >> CStringFormatShift)
	bFmt := uint8(bVal >> CStringFormatShift)
	if aFmt != bFmt {
		return equalStringValues(a, b, false) // decode only until a difference
	}
	if aFmt >= 11 {
		av, aok := makeStringView(a)
		bv, bok := makeStringView(b)
		if aok && bok && av.offset == bv.offset {
			start, n := 0, aCharLen
			if av.offset == 1 && n > 0 {
				if av.data[0]&15 != bv.data[0]&15 {
					return false
				}
				start, n = 1, n-1
			}
			full := n / 2
			if av.data[start:start+full] != bv.data[start:start+full] {
				return false
			}
			return n&1 == 0 || av.data[start+full]>>4 == bv.data[start+full]>>4
		}
		return equalStringValues(a, b, false)
	}
	aNibOff := int((aVal >> CStringOffsetShift) & 1)
	bNibOff := int((bVal >> CStringOffsetShift) & 1)
	if cstringIsNibble(aFmt) {
		if aNibOff == bNibOff {
			// Same offset: memcmp inner bytes + mask overhangs, zero allocation.
			return nibbleRangeEqual(a.ptr, b.ptr, aNibOff, aCharLen)
		}
		// Different offsets (cross-column / nodict): per-nibble fallback.
		for i := 0; i < aCharLen; i++ {
			if nibbleAt(a.ptr, aNibOff+i) != nibbleAt(b.ptr, bNibOff+i) {
				return false
			}
		}
		return true
	}
	// UUID formats (6=UUIDLower, 7=UUIDUpper): stored as 16 raw bytes, nibbleOff always 0
	if aFmt == 6 || aFmt == 7 {
		return unsafe.String(a.ptr, 16) == unsafe.String(b.ptr, 16)
	}
	return a.String() == b.String() // fallback for unknown/future formats
}

func EqualScm(a, b Scmer) Scmer { return NewBool(Equal(a, b)) }

func Equal(a, b Scmer) bool {
	ta := a.GetTag()
	tb := b.GetTag()
	if a.IsSourceInfo() {
		return Equal(a.SourceInfo().value, b)
	}
	if b.IsSourceInfo() {
		return Equal(a, b.SourceInfo().value)
	}
	if ta == tagAny {
		if si, ok := a.Any().(SourceInfo); ok {
			return Equal(si.value, b)
		}
	}
	if tb == tagAny {
		if si, ok := b.Any().(SourceInfo); ok {
			return Equal(a, si.value)
		}
	}

	if ta == tagNil && tb == tagNil {
		return true
	}
	if ta == tagNil {
		return !b.Bool()
	}
	if tb == tagNil {
		return !a.Bool()
	}

	if ta == tb {
		switch ta {
		case tagBool:
			return a.Bool() == b.Bool()
		case tagInt:
			return a.Int() == b.Int()
		case tagDate:
			return a.Int() == b.Int()
		case tagFloat:
			return a.Float() == b.Float()
		case tagString, tagSymbol:
			// Both tags already prove that ptr/aux encode a plain Go string.
			// Keep this dominant comparison path out of AppendString: that
			// general converter carries a large type switch and stack frame for
			// compressed strings, BSON, lists, and numeric formatting.
			return unsafe.String(a.ptr, int(auxVal(a.aux))) == unsafe.String(b.ptr, int(auxVal(b.aux)))
		case tagCString:
			return cstringEqual(a, b)
		case tagBString:
			if auxVal(a.aux)>>46 == auxVal(b.aux)>>46 {
				an, bn := int(auxVal(a.aux)&bstringLengthMask), int(auxVal(b.aux)&bstringLengthMask)
				return an == bn && unsafe.String(a.ptr, an) == unsafe.String(b.ptr, bn)
			}
			return equalStringValues(a, b, false)
		case tagBSON:
			return bsonRawEqual(bsonRawValue(a), bsonRawValue(b))
		case tagSlice:
			as := a.Slice()
			bs := b.Slice()
			if len(as) != len(bs) {
				return false
			}
			for i := range as {
				if !Equal(as[i], bs[i]) {
					return false
				}
			}
			return true
		case tagVector:
			av := a.Vector()
			bv := b.Vector()
			if len(av) != len(bv) {
				return false
			}
			for i := range av {
				if av[i] != bv[i] {
					return false
				}
			}
			return true
		case tagFastDict:
			af := a.FastDict()
			bf := b.FastDict()
			if af == nil || bf == nil {
				return af == nil && bf == nil
			}
			return equalAssocPairs(af.Pairs, bf.Pairs)
		case tagProc:
			return a.ptr == b.ptr
		case tagAny:
			return a.Any() == b.Any()
		}
	}

	switch ta {
	case tagBool:
		return a.Bool() == b.Bool()
	case tagDate:
		if tb == tagString || tb == tagSymbol {
			if ts, ok := ParseDateString(b.String()); ok {
				return a.Int() == ts
			}
		}
		return a.Int() == b.Int()
	case tagInt:
		if tb == tagFloat {
			return float64(a.Int()) == b.Float()
		}
		if tb == tagString || tb == tagSymbol {
			return a.Int() == b.Int()
		}
		if tb == tagBool || tb == tagDate {
			return a.Int() == b.Int()
		}
		return false
	case tagFloat:
		if tb == tagInt {
			return a.Float() == float64(b.Int())
		}
		if tb == tagString || tb == tagSymbol {
			return a.Float() == b.Float()
		}
		if tb == tagBool || tb == tagDate {
			return a.Float() == b.Float()
		}
		return false
	case tagString, tagSymbol:
		if tb == tagDate {
			if ts, ok := ParseDateString(a.String()); ok {
				return ts == b.Int()
			}
		}
		if tb == tagInt {
			return a.Int() == b.Int()
		}
		if tb == tagFloat {
			return a.Float() == b.Float()
		}
		if tb == tagBool {
			return a.Bool() == b.Bool()
		}
		return equalStringValues(a, b, false)
	case tagCString:
		return equalStringValues(a, b, false)
	case tagBString:
		return equalStringValues(a, b, false)
	case tagBSON:
		if tb == tagBSON {
			return bsonRawEqual(bsonRawValue(a), bsonRawValue(b))
		}
		return equalStringValues(a, b, false)
	case tagSlice:
		if len(a.Slice()) == 0 {
			return !b.Bool()
		}
	case tagVector:
		if len(a.Vector()) == 0 {
			return !b.Bool()
		}
	case tagFunc:
		if tb == tagFunc {
			return a.ptr == b.ptr
		}
		return false
	case tagPromise:
		if tb == tagPromise {
			return a.ptr == b.ptr
		}
		return false
	case tagAny:
		return a.Any() == b.Any()
	}

	if pairsA, ok := assocPairs(a); ok {
		if pairsB, ok := assocPairs(b); ok {
			return equalAssocPairs(pairsA, pairsB)
		}
	}

	return equalStringValues(a, b, false)
}

func assocPairs(v Scmer) ([]Scmer, bool) {
	v = unwrapAssoc(v)
	switch v.GetTag() {
	case tagSlice:
		s := v.Slice()
		if len(s)%2 == 0 {
			return s, true
		}
	case tagFastDict:
		fd := v.FastDict()
		if fd == nil {
			return []Scmer{}, true
		}
		return fd.Pairs, true
	}
	return nil, false
}

func unwrapAssoc(v Scmer) Scmer {
	if v.IsSourceInfo() {
		return v.SourceInfo().value
	}
	return v
}

func equalAssocPairs(aPairs, bPairs []Scmer) bool {
	if len(aPairs)%2 != 0 || len(bPairs)%2 != 0 {
		return false
	}
	if len(aPairs) != len(bPairs) {
		return false
	}
	type entry struct {
		key Scmer
		val Scmer
	}
	buckets := make(map[uint64][]entry)
	for i := 0; i < len(bPairs); i += 2 {
		h := HashKey(bPairs[i])
		buckets[h] = append(buckets[h], entry{bPairs[i], bPairs[i+1]})
	}
	for i := 0; i < len(aPairs); i += 2 {
		h := HashKey(aPairs[i])
		entries := buckets[h]
		found := false
		for idx, e := range entries {
			if Equal(aPairs[i], e.key) && Equal(aPairs[i+1], e.val) {
				buckets[h] = append(entries[:idx], entries[idx+1:]...)
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, entries := range buckets {
		if len(entries) > 0 {
			return false
		}
	}
	return true
}

func EqualSQL(a, b Scmer) Scmer {
	ta := a.GetTag()
	tb := b.GetTag()

	if ta == tagNil || tb == tagNil {
		return NewNil()
	}

	if ta == tb {
		switch ta {
		case tagBool:
			return NewBool(a.Bool() == b.Bool())
		case tagInt:
			return NewBool(a.Int() == b.Int())
		case tagDate:
			return NewBool(a.Int() == b.Int())
		case tagFloat:
			return NewBool(a.Float() == b.Float())
		case tagString, tagSymbol:
			return NewBool(equalStringValues(a, b, true))
		case tagCString:
			return NewBool(equalStringValues(a, b, true))
		case tagBString:
			if a.aux == b.aux && unsafe.String(a.ptr, int(auxVal(a.aux)&bstringLengthMask)) == unsafe.String(b.ptr, int(auxVal(b.aux)&bstringLengthMask)) {
				return NewBool(true)
			}
			return NewBool(equalStringValues(a, b, true))
		case tagBSON:
			return NewBool(bsonRawEqual(bsonRawValue(a), bsonRawValue(b)))
		case tagSlice:
			as := a.Slice()
			bs := b.Slice()
			if len(as) != len(bs) {
				return NewBool(false)
			}
			for i := range as {
				if !EqualSQL(as[i], bs[i]).Bool() {
					return NewBool(false)
				}
			}
			return NewBool(true)
		case tagVector:
			av := a.Vector()
			bv := b.Vector()
			if len(av) != len(bv) {
				return NewBool(false)
			}
			for i := range av {
				if av[i] != bv[i] {
					return NewBool(false)
				}
			}
			return NewBool(true)
		case tagAny:
			return NewBool(a.Any() == b.Any())
		}
	}

	switch ta {
	case tagDate:
		if tb == tagString || tb == tagSymbol {
			if ts, ok := ParseDateString(b.String()); ok {
				return NewBool(a.Int() == ts)
			}
		}
		return NewBool(a.Int() == b.Int())
	case tagInt:
		if tb == tagFloat {
			return NewBool(float64(a.Int()) == b.Float())
		}
		if tb == tagString || tb == tagSymbol || tb == tagCString || tb == tagBString {
			return NewBool(numericStringEqualsInt(b.String(), a.Int()))
		}
		return NewBool(a.Int() == b.Int())
	case tagFloat:
		if tb == tagInt {
			return NewBool(a.Float() == float64(b.Int()))
		}
		if tb == tagString || tb == tagSymbol || tb == tagCString || tb == tagBString {
			return NewBool(a.Float() == b.Float())
		}
		return NewBool(a.Float() == b.Float())
	case tagString, tagSymbol, tagCString:
		if tb == tagDate {
			if ts, ok := ParseDateString(a.String()); ok {
				return NewBool(ts == b.Int())
			}
		}
		if tb == tagInt {
			return NewBool(numericStringEqualsInt(a.String(), b.Int()))
		}
		if tb == tagFloat {
			return NewBool(a.Float() == b.Float())
		}
		if tb == tagBool {
			return NewBool(a.Bool() == b.Bool())
		}
		return NewBool(equalStringValues(a, b, true))
	case tagBString:
		if tb == tagInt {
			return NewBool(numericStringEqualsInt(a.String(), b.Int()))
		}
		if tb == tagFloat {
			return NewBool(a.Float() == b.Float())
		}
		if tb == tagBool {
			return NewBool(a.Bool() == b.Bool())
		}
		return NewBool(equalStringValues(a, b, true))
	case tagBSON:
		if tb == tagBSON {
			return NewBool(bsonRawEqual(bsonRawValue(a), bsonRawValue(b)))
		}
		return NewBool(a.String() == b.String())
	case tagBool:
		return NewBool(a.Bool() == b.Bool())
	case tagSlice:
		if len(a.Slice()) == 0 {
			return NewBool(!b.Bool())
		}
	case tagVector:
		if len(a.Vector()) == 0 {
			return NewBool(!b.Bool())
		}
	case tagFunc:
		if tb == tagFunc {
			return NewBool(a.ptr == b.ptr)
		}
		return NewBool(false)
	case tagPromise:
		if tb == tagPromise {
			return NewBool(a.ptr == b.ptr)
		}
		return NewBool(false)
	case tagAny:
		return NewBool(a.Any() == b.Any())
	}

	return NewBool(equalStringValues(a, b, true))
}

// Keep representation dispatch out of generated collation emitters; their
// existing plain-string arm stays small and eligible for JIT inlining.
//
//jitgen:noinline
func equalCollatedValues(a, b Scmer, collation string) Scmer {
	if (a.IsString() || a.IsSymbol()) && (b.IsString() || b.IsSymbol()) {
		return NewBool(equalStringValues(a, b, strings.Contains(collation, "_ci")))
	}
	return EqualSQL(a, b)
}

// Normalize integral bounds once, outside evaluation loops. Mixed comparisons
// retain their original float rounding at ±2^53 and their string fallback.
func optimizeOrderedComparison(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	code, typeInfo := oc.ApplyDefaultOptimization(v, useResult)
	items, ok := scmerSlice(code)
	if !ok || len(items) != 3 {
		return code, typeInfo
	}
	var rewritten []Scmer
	for i := 1; i < 3; i++ {
		if !items[i].IsFloat() {
			continue
		}
		f := items[i].Float()
		if !(f > -(1<<53) && f < 1<<53) {
			continue
		}
		integer := int64(f)
		if float64(integer) != f {
			continue
		}
		bound := NewInt(integer)
		// Non-numeric values may compare through their printed representation.
		if items[i].String() != bound.String() {
			continue
		}
		if rewritten == nil {
			rewritten = append([]Scmer(nil), items...)
		}
		rewritten[i] = bound
	}
	if rewritten != nil {
		return NewSlice(rewritten), typeInfo
	}
	return code, typeInfo
}

func LessScm(a ...Scmer) Scmer    { return NewBool(Less(a[0], a[1])) }
func GreaterScm(a ...Scmer) Scmer { return NewBool(Less(a[1], a[0])) }

//jitgen:emitter jitEmitLess
func Less(a, b Scmer) bool {
	ta := a.GetTag()
	tb := b.GetTag()

	if ta == tagNil && tb == tagNil {
		return false
	}
	if ta == tagNil {
		return true
	}
	if tb == tagNil {
		return false
	}

	switch ta {
	case tagDate:
		if tb == tagString || tb == tagSymbol {
			if ts, ok := ParseDateString(b.String()); ok {
				return a.Int() < ts
			}
		}
		return float64(a.Int()) < b.Float()
	case tagInt:
		if tb == tagInt {
			return a.Int() < b.Int()
		}
		return float64(a.Int()) < b.Float()
	case tagFloat:
		return a.Float() < b.Float()
	case tagBool:
		return a.Int() < b.Int()
	default:
		return lessNonNumeric(a, b, ta, tb)
	}
}

//jitgen:noinline
func lessNonNumeric(a, b Scmer, ta, tb uint8) bool {
	if ta == tagCString || tb == tagCString {
		if c, ok := compareCString(a, b, false); ok {
			return c < 0
		}
	}
	if ta == tagBString || tb == tagBString {
		if c, ok := compareBase64Text(a, b, false); ok {
			return c < 0
		}
	}
	switch ta {
	case tagBSON:
		switch tb {
		case tagBSON:
			return bsonRawLess(bsonRawValue(a), bsonRawValue(b))
		case tagDate, tagInt, tagFloat:
			return a.Float() < b.Float()
		case tagBool:
			return a.Int() < b.Int()
		default:
			return a.String() < b.String()
		}
	case tagString, tagSymbol, tagCString, tagBString:
		switch tb {
		case tagDate:
			if ts, ok := ParseDateString(a.String()); ok {
				return ts < b.Int()
			}
			return a.Float() < b.Float()
		case tagInt:
			return a.Float() < b.Float()
		case tagFloat:
			return a.Float() < b.Float()
		case tagString, tagSymbol:
			if ta == tagString || ta == tagSymbol {
				// As in Equal, known plain-string tags can be compared directly.
				// Compressed representations retain the materializing fallback.
				return unsafe.String(a.ptr, int(auxVal(a.aux))) < unsafe.String(b.ptr, int(auxVal(b.aux)))
			}
			return a.String() < b.String()
		case tagCString, tagBString, tagBSON:
			return a.String() < b.String()
		default:
			// Fallback: compare by string representation to avoid panics on mixed types
			return strings.Compare(a.String(), b.String()) < 0
		}
	case tagFunc:
		return uintptr(unsafe.Pointer(a.ptr)) < uintptr(unsafe.Pointer(b.ptr))
	case tagAny:
		return strings.Compare(a.String(), b.String()) < 0
	default:
		return strings.Compare(a.String(), b.String()) < 0
	}
}

// jitEmitLess is generated from Less by jitgen. It is called while a parent
// emitter is producing machine code, so known argument tags prune Less' cold
// comparison arms without duplicating its SSA in every generated builtin.
func jitEmitLess(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
	var d26 JITValueDesc
	_ = d26
	var d27 JITValueDesc
	_ = d27
	var d40 JITValueDesc
	_ = d40
	var d41 JITValueDesc
	_ = d41
	var d42 JITValueDesc
	_ = d42
	var d58 JITValueDesc
	_ = d58
	var d59 JITValueDesc
	_ = d59
	var d60 JITValueDesc
	_ = d60
	var d79 JITValueDesc
	_ = d79
	var d80 JITValueDesc
	_ = d80
	var d101 JITValueDesc
	_ = d101
	var d102 JITValueDesc
	_ = d102
	var d125 JITValueDesc
	_ = d125
	var d126 JITValueDesc
	_ = d126
	var d151 JITValueDesc
	_ = d151
	var d152 JITValueDesc
	_ = d152
	var d154 JITValueDesc
	_ = d154
	var d155 JITValueDesc
	_ = d155
	var d156 JITValueDesc
	_ = d156
	var d186 JITValueDesc
	_ = d186
	var d187 JITValueDesc
	_ = d187
	var d188 JITValueDesc
	_ = d188
	var d190 JITValueDesc
	_ = d190
	var d191 JITValueDesc
	_ = d191
	var d192 JITValueDesc
	_ = d192
	var d228 JITValueDesc
	_ = d228
	var d230 JITValueDesc
	_ = d230
	var d231 JITValueDesc
	_ = d231
	var d232 JITValueDesc
	_ = d232
	var d234 JITValueDesc
	_ = d234
	var d235 JITValueDesc
	_ = d235
	var d236 JITValueDesc
	_ = d236
	var d279 JITValueDesc
	_ = d279
	var d280 JITValueDesc
	_ = d280
	var d282 JITValueDesc
	_ = d282
	var d283 JITValueDesc
	_ = d283
	var d284 JITValueDesc
	_ = d284
	var d285 JITValueDesc
	_ = d285
	var d287 JITValueDesc
	_ = d287
	var d288 JITValueDesc
	_ = d288
	var d289 JITValueDesc
	_ = d289
	var d291 JITValueDesc
	_ = d291
	var d292 JITValueDesc
	_ = d292
	var d293 JITValueDesc
	_ = d293
	var d348 JITValueDesc
	_ = d348
	for i := range args {
		if args[i].Type != JITTypeUnknown {
			continue
		}
		nativeArgs := [...]JITValueDesc{args[0], args[1]}
		for j := range nativeArgs {
			nativeArgs[j] = JITPrepareScmerGoArg(ctx, nativeArgs[j])
		}
		native := ctx.EmitGoCallScalar(GoFuncAddr(Less), nativeArgs[:], 1)
		ctx.EmitAndRegImm32(native.Reg, 1)
		native.Type = tagBool
		if result.Loc == LocAny {
			return native
		}
		ctx.EmitMovToReg(result.Reg, native)
		result.Type = tagBool
		return result
	}
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	branchSerial := ctx.branchSerial
	_ = branchSerial
	var bbs [22]BBDescriptor
	if result.Loc == LocAny {
		result = JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
	}
	resultRegsProtected := result.Loc == LocReg
	if resultRegsProtected {
		ctx.ProtectReg(result.Reg)
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
	bbpos_0_20 := int32(-1)
	_ = bbpos_0_20
	lbl21 := ctx.ReserveLabel()
	_ = lbl21
	bbpos_0_21 := int32(-1)
	_ = bbpos_0_21
	lbl22 := ctx.ReserveLabel()
	_ = lbl22
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
		d1 = ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
		ctx.StabilizeDescForControlFlow(&d1)
		d2 = args[1]
		d2.ID = 0
		d3 = ctx.EmitGetTagDesc(&d2, JITValueDesc{Loc: LocAny})
		ctx.StabilizeDescForControlFlow(&d3)
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x0))}
		} else {
			r0 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 0)
			d4 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
			ctx.BindReg(r0, &d4)
		}
		d5 = d4
		ctx.EnsureDesc(&d5)
		if d5.Loc != LocImm && d5.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d5.Loc == LocImm {
			if d5.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d5.Condition, lbl4)
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
		if !bbs[3].Rendered {
			return bbs[3].Render()
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
		d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d13) {
			return d13
		}
		ctx.EnsureDesc(&d13)
		ctx.EmitMovToReg(result.Reg, d13)
		result.Type = d13.Type
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
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x0))}
		} else {
			r1 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 0)
			d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
			ctx.BindReg(r1, &d14)
		}
		d15 = d14
		ctx.EnsureDesc(&d15)
		if d15.Loc != LocImm && d15.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d15.Loc == LocImm {
			if d15.Imm.Bool() {
				return bbs[4].Render()
			}
			return bbs[5].Render()
		}
		ctx.EmitJump(d15.Condition, lbl5)
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
		ctx.FreeDesc(&d14)
		ctx.FlushRegisterMoves()
		if !bbs[5].Rendered {
			snap16 := d0
			snap17 := d1
			snap18 := d2
			snap19 := d3
			snap20 := d4
			snap21 := d5
			snap22 := d13
			snap23 := d14
			snap24 := d15
			alloc25 := ctx.SnapshotAllocState()
			bbs[5].Render()
			ctx.RestoreAllocState(alloc25)
			d0 = snap16
			d1 = snap17
			d2 = snap18
			d3 = snap19
			d4 = snap20
			d5 = snap21
			d13 = snap22
			d14 = snap23
			d15 = snap24
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
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d26 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x0))}
		} else {
			r2 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			d26 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
			ctx.BindReg(r2, &d26)
		}
		d27 = d26
		ctx.EnsureDesc(&d27)
		if d27.Loc != LocImm && d27.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d27.Loc == LocImm {
			if d27.Imm.Bool() {
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d27.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d26)
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap28 := d0
			snap29 := d1
			snap30 := d2
			snap31 := d3
			snap32 := d4
			snap33 := d5
			snap34 := d13
			snap35 := d14
			snap36 := d15
			snap37 := d26
			snap38 := d27
			alloc39 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc39)
			d0 = snap28
			d1 = snap29
			d2 = snap30
			d3 = snap31
			d4 = snap32
			d5 = snap33
			d13 = snap34
			d14 = snap35
			d15 = snap36
			d26 = snap37
			d27 = snap38
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
		d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d40) {
			return d40
		}
		ctx.EnsureDesc(&d40)
		ctx.EmitMovToReg(result.Reg, d40)
		result.Type = d40.Type
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
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d41 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x0))}
		} else {
			r3 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			d41 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
			ctx.BindReg(r3, &d41)
		}
		d42 = d41
		ctx.EnsureDesc(&d42)
		if d42.Loc != LocImm && d42.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d42.Loc == LocImm {
			if d42.Imm.Bool() {
				return bbs[6].Render()
			}
			return bbs[7].Render()
		}
		ctx.EmitJump(d42.Condition, lbl7)
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d41)
		ctx.FlushRegisterMoves()
		if !bbs[7].Rendered {
			snap43 := d0
			snap44 := d1
			snap45 := d2
			snap46 := d3
			snap47 := d4
			snap48 := d5
			snap49 := d13
			snap50 := d14
			snap51 := d15
			snap52 := d26
			snap53 := d27
			snap54 := d40
			snap55 := d41
			snap56 := d42
			alloc57 := ctx.SnapshotAllocState()
			bbs[7].Render()
			ctx.RestoreAllocState(alloc57)
			d0 = snap43
			d1 = snap44
			d2 = snap45
			d3 = snap46
			d4 = snap47
			d5 = snap48
			d13 = snap49
			d14 = snap50
			d15 = snap51
			d26 = snap52
			d27 = snap53
			d40 = snap54
			d41 = snap55
			d42 = snap56
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
		d58 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d58) {
			return d58
		}
		ctx.EnsureDesc(&d58)
		ctx.EmitMovToReg(result.Reg, d58)
		result.Type = d58.Type
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
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d59 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x10))}
		} else {
			r4 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 16)
			d59 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
			ctx.BindReg(r4, &d59)
		}
		d60 = d59
		ctx.EnsureDesc(&d60)
		if d60.Loc != LocImm && d60.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d60.Loc == LocImm {
			if d60.Imm.Bool() {
				return bbs[8].Render()
			}
			return bbs[10].Render()
		}
		ctx.EmitJump(d60.Condition, lbl9)
		if bbs[10].Rendered {
			ctx.EmitJmp(lbl11)
		}
		ctx.FreeDesc(&d59)
		ctx.FlushRegisterMoves()
		if !bbs[10].Rendered {
			snap61 := d0
			snap62 := d1
			snap63 := d2
			snap64 := d3
			snap65 := d4
			snap66 := d5
			snap67 := d13
			snap68 := d14
			snap69 := d15
			snap70 := d26
			snap71 := d27
			snap72 := d40
			snap73 := d41
			snap74 := d42
			snap75 := d58
			snap76 := d59
			snap77 := d60
			alloc78 := ctx.SnapshotAllocState()
			bbs[10].Render()
			ctx.RestoreAllocState(alloc78)
			d0 = snap61
			d1 = snap62
			d2 = snap63
			d3 = snap64
			d4 = snap65
			d5 = snap66
			d13 = snap67
			d14 = snap68
			d15 = snap69
			d26 = snap70
			d27 = snap71
			d40 = snap72
			d41 = snap73
			d42 = snap74
			d58 = snap75
			d59 = snap76
			d60 = snap77
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x1))}
		} else {
			r5 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 1)
			d79 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
			ctx.BindReg(r5, &d79)
		}
		d80 = d79
		ctx.EnsureDesc(&d80)
		if d80.Loc != LocImm && d80.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d80.Loc == LocImm {
			if d80.Imm.Bool() {
				return bbs[11].Render()
			}
			return bbs[13].Render()
		}
		ctx.EmitJump(d80.Condition, lbl12)
		if bbs[13].Rendered {
			ctx.EmitJmp(lbl14)
		}
		ctx.FreeDesc(&d79)
		ctx.FlushRegisterMoves()
		if !bbs[13].Rendered {
			snap81 := d0
			snap82 := d1
			snap83 := d2
			snap84 := d3
			snap85 := d4
			snap86 := d5
			snap87 := d13
			snap88 := d14
			snap89 := d15
			snap90 := d26
			snap91 := d27
			snap92 := d40
			snap93 := d41
			snap94 := d42
			snap95 := d58
			snap96 := d59
			snap97 := d60
			snap98 := d79
			snap99 := d80
			alloc100 := ctx.SnapshotAllocState()
			bbs[13].Render()
			ctx.RestoreAllocState(alloc100)
			d0 = snap81
			d1 = snap82
			d2 = snap83
			d3 = snap84
			d4 = snap85
			d5 = snap86
			d13 = snap87
			d14 = snap88
			d15 = snap89
			d26 = snap90
			d27 = snap91
			d40 = snap92
			d41 = snap93
			d42 = snap94
			d58 = snap95
			d59 = snap96
			d60 = snap97
			d79 = snap98
			d80 = snap99
		}
		if !bbs[11].Rendered {
			return bbs[11].Render()
		}
		return result
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d101 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x4))}
		} else {
			r6 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 4)
			d101 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
			ctx.BindReg(r6, &d101)
		}
		d102 = d101
		ctx.EnsureDesc(&d102)
		if d102.Loc != LocImm && d102.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d102.Loc == LocImm {
			if d102.Imm.Bool() {
				return bbs[17].Render()
			}
			return bbs[18].Render()
		}
		ctx.EmitJump(d102.Condition, lbl18)
		if bbs[18].Rendered {
			ctx.EmitJmp(lbl19)
		}
		ctx.FreeDesc(&d101)
		ctx.FlushRegisterMoves()
		if !bbs[18].Rendered {
			snap103 := d0
			snap104 := d1
			snap105 := d2
			snap106 := d3
			snap107 := d4
			snap108 := d5
			snap109 := d13
			snap110 := d14
			snap111 := d15
			snap112 := d26
			snap113 := d27
			snap114 := d40
			snap115 := d41
			snap116 := d42
			snap117 := d58
			snap118 := d59
			snap119 := d60
			snap120 := d79
			snap121 := d80
			snap122 := d101
			snap123 := d102
			alloc124 := ctx.SnapshotAllocState()
			bbs[18].Render()
			ctx.RestoreAllocState(alloc124)
			d0 = snap103
			d1 = snap104
			d2 = snap105
			d3 = snap106
			d4 = snap107
			d5 = snap108
			d13 = snap109
			d14 = snap110
			d15 = snap111
			d26 = snap112
			d27 = snap113
			d40 = snap114
			d41 = snap115
			d42 = snap116
			d58 = snap117
			d59 = snap118
			d60 = snap119
			d79 = snap120
			d80 = snap121
			d101 = snap122
			d102 = snap123
		}
		if !bbs[17].Rendered {
			return bbs[17].Render()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d125 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x4))}
		} else {
			r7 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 4)
			d125 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
			ctx.BindReg(r7, &d125)
		}
		d126 = d125
		ctx.EnsureDesc(&d126)
		if d126.Loc != LocImm && d126.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d126.Loc == LocImm {
			if d126.Imm.Bool() {
				return bbs[9].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitJump(d126.Condition, lbl10)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d125)
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
			snap127 := d0
			snap128 := d1
			snap129 := d2
			snap130 := d3
			snap131 := d4
			snap132 := d5
			snap133 := d13
			snap134 := d14
			snap135 := d15
			snap136 := d26
			snap137 := d27
			snap138 := d40
			snap139 := d41
			snap140 := d42
			snap141 := d58
			snap142 := d59
			snap143 := d60
			snap144 := d79
			snap145 := d80
			snap146 := d101
			snap147 := d102
			snap148 := d125
			snap149 := d126
			alloc150 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc150)
			d0 = snap127
			d1 = snap128
			d2 = snap129
			d3 = snap130
			d4 = snap131
			d5 = snap132
			d13 = snap133
			d14 = snap134
			d15 = snap135
			d26 = snap136
			d27 = snap137
			d40 = snap138
			d41 = snap139
			d42 = snap140
			d58 = snap141
			d59 = snap142
			d60 = snap143
			d79 = snap144
			d80 = snap145
			d101 = snap146
			d102 = snap147
			d125 = snap148
			d126 = snap149
		}
		if !bbs[9].Rendered {
			return bbs[9].Render()
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
		ctx.ReclaimUntrackedRegs()
		d152 = args[1]
		ctx.SyncDesc(&d152)
		if d152.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d152.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d152.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d152 = tmpScalar
		}
		d152 = JITPrepareScmerGoArg(ctx, d152)
		if d152.Loc != LocRegPair && d152.Loc != LocStackPair && d152.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d151 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d152}, 2)
		ctx.EnsureDesc(&d151)
		if d151.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d151.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d151.Imm)
			ptrWord, _ := d151.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d151.Imm.String())))
			d151 = tmpPair
		} else if d151.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d151.Type, Reg: ctx.AllocRegExcept(d151.Reg), Reg2: ctx.AllocRegExcept(d151.Reg)}
			switch d151.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d151)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d151)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d151)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d151)
			d151 = tmpPair
		}
		if d151.Loc != LocRegPair && d151.Loc != LocStackPair && d151.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateString arg0)")
		}
		ctx.SyncDesc(&d151)
		callResults153 := JITEmitGoCallResults(ctx, GoFuncAddr(ParseDateString), []JITValueDesc{d151}, []uint8{1, 1}, []uint8{0, 0})
		d154 = callResults153[0]
		_ = d154
		d155 = callResults153[1]
		_ = d155
		ctx.StabilizeDescForControlFlow(&d154)
		d156 = d155
		ctx.EnsureDesc(&d156)
		if d156.Loc != LocImm && d156.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d156.Loc == LocImm {
			if d156.Imm.Bool() {
				return bbs[14].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitCmpRegImm32(d156.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl15)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
			snap157 := d0
			snap158 := d1
			snap159 := d2
			snap160 := d3
			snap161 := d4
			snap162 := d5
			snap163 := d13
			snap164 := d14
			snap165 := d15
			snap166 := d26
			snap167 := d27
			snap168 := d40
			snap169 := d41
			snap170 := d42
			snap171 := d58
			snap172 := d59
			snap173 := d60
			snap174 := d79
			snap175 := d80
			snap176 := d101
			snap177 := d102
			snap178 := d125
			snap179 := d126
			snap180 := d151
			snap181 := d152
			snap182 := d154
			snap183 := d155
			snap184 := d156
			alloc185 := ctx.SnapshotAllocState()
			bbs[12].Render()
			ctx.RestoreAllocState(alloc185)
			d0 = snap157
			d1 = snap158
			d2 = snap159
			d3 = snap160
			d4 = snap161
			d5 = snap162
			d13 = snap163
			d14 = snap164
			d15 = snap165
			d26 = snap166
			d27 = snap167
			d40 = snap168
			d41 = snap169
			d42 = snap170
			d58 = snap171
			d59 = snap172
			d60 = snap173
			d79 = snap174
			d80 = snap175
			d101 = snap176
			d102 = snap177
			d125 = snap178
			d126 = snap179
			d151 = snap180
			d152 = snap181
			d154 = snap182
			d155 = snap183
			d156 = snap184
		}
		if !bbs[14].Rendered {
			return bbs[14].Render()
		}
		return result
		ctx.FreeDesc(&d155)
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
		ctx.ReclaimUntrackedRegs()
		if args[0].Loc == LocImm {
			d186 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d186)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d186)
		} else {
			d186 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d186.Type = tagInt
			ctx.BindReg(d186.Reg, &d186)
		}
		ctx.EnsureDesc(&d186)
		ctx.EnsureDesc(&d186)
		if d186.Loc == LocImm {
			d187 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d186.Imm.Int()))}
		} else {
			var r8 Reg
			r8 = d186.Reg
			d186.Loc = LocNone
			ctx.EmitInt64ToFloatBits(r8)
			d187 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
			ctx.BindReg(r8, &d187)
		}
		ctx.FreeDesc(&d186)
		d188 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d187)
		resultTarget189 := false
		_ = resultTarget189
		ctx.EnsureDesc(&d188)
		ctx.EnsureDescsTogether(&d187, &d188)
		if d187.Loc == LocImm && d188.Loc == LocImm {
			d190 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d187.Imm.Float() < d188.Imm.Float())}
		} else if d188.Loc == LocImm {
			var r9 Reg
			if result.Loc == LocReg && result.Reg != d187.Reg {
				r9 = result.Reg
				resultTarget189 = true
			} else {
				r9 = ctx.AllocRegExcept(d187.Reg)
			}
			_, yBits := d188.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d187.Reg)
			d190 = ctx.DeferBooleanFlags(r9, CondUnsignedAbove)
			ctx.BindReg(r9, &d190)
		} else if d187.Loc == LocImm {
			var r10 Reg
			if result.Loc == LocReg && result.Reg != d188.Reg {
				r10 = result.Reg
				resultTarget189 = true
			} else {
				r10 = ctx.AllocRegExcept(d188.Reg)
			}
			_, xBits := d187.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d188.Reg, ctx.ScratchReg)
			d190 = ctx.DeferBooleanFlags(r10, CondUnsignedAbove)
			ctx.BindReg(r10, &d190)
		} else {
			var r11 Reg
			if result.Loc == LocReg && result.Reg != d187.Reg && result.Reg != d188.Reg {
				r11 = result.Reg
				resultTarget189 = true
			} else {
				r11 = ctx.AllocRegExcept(d187.Reg, d188.Reg)
			}
			ctx.EmitCmpFloat64(d188.Reg, d187.Reg)
			d190 = ctx.DeferBooleanFlags(r11, CondUnsignedAbove)
			ctx.BindReg(r11, &d190)
		}
		ctx.FreeDesc(&d187)
		ctx.FreeDesc(&d188)
		ctx.SyncDesc(&d190)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d190) {
			return d190
		}
		ctx.EnsureDesc(&d190)
		ctx.EmitMovToReg(result.Reg, d190)
		result.Type = d190.Type
		ctx.EmitJmp(lbl0)
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d191 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x2))}
		} else {
			r12 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 2)
			d191 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondEqual}
			ctx.BindReg(r12, &d191)
		}
		d192 = d191
		ctx.EnsureDesc(&d192)
		if d192.Loc != LocImm && d192.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d192.Loc == LocImm {
			if d192.Imm.Bool() {
				return bbs[11].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitJump(d192.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d191)
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
			snap193 := d0
			snap194 := d1
			snap195 := d2
			snap196 := d3
			snap197 := d4
			snap198 := d5
			snap199 := d13
			snap200 := d14
			snap201 := d15
			snap202 := d26
			snap203 := d27
			snap204 := d40
			snap205 := d41
			snap206 := d42
			snap207 := d58
			snap208 := d59
			snap209 := d60
			snap210 := d79
			snap211 := d80
			snap212 := d101
			snap213 := d102
			snap214 := d125
			snap215 := d126
			snap216 := d151
			snap217 := d152
			snap218 := d154
			snap219 := d155
			snap220 := d156
			snap221 := d186
			snap222 := d187
			snap223 := d188
			snap224 := d190
			snap225 := d191
			snap226 := d192
			alloc227 := ctx.SnapshotAllocState()
			bbs[12].Render()
			ctx.RestoreAllocState(alloc227)
			d0 = snap193
			d1 = snap194
			d2 = snap195
			d3 = snap196
			d4 = snap197
			d5 = snap198
			d13 = snap199
			d14 = snap200
			d15 = snap201
			d26 = snap202
			d27 = snap203
			d40 = snap204
			d41 = snap205
			d42 = snap206
			d58 = snap207
			d59 = snap208
			d60 = snap209
			d79 = snap210
			d80 = snap211
			d101 = snap212
			d102 = snap213
			d125 = snap214
			d126 = snap215
			d151 = snap216
			d152 = snap217
			d154 = snap218
			d155 = snap219
			d156 = snap220
			d186 = snap221
			d187 = snap222
			d188 = snap223
			d190 = snap224
			d191 = snap225
			d192 = snap226
		}
		if !bbs[11].Rendered {
			return bbs[11].Render()
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
		ctx.ReclaimUntrackedRegs()
		if args[0].Loc == LocImm {
			d228 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d228 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d228)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d228 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d228)
		} else {
			d228 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d228.Type = tagInt
			ctx.BindReg(d228.Reg, &d228)
		}
		ctx.EnsureDesc(&d228)
		resultTarget229 := false
		_ = resultTarget229
		ctx.EnsureDesc(&d154)
		ctx.EnsureDescsTogether(&d228, &d154)
		if d228.Loc == LocImm && d154.Loc == LocImm {
			d230 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d228.Imm.Int() < d154.Imm.Int())}
		} else if d154.Loc == LocImm {
			r13 := ctx.AllocReg()
			if d154.Imm.Int() >= -2147483648 && d154.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d228.Reg, int32(d154.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d154.Imm.Int()))
				ctx.EmitCmpInt64(d228.Reg, ctx.ScratchReg)
			}
			d230 = ctx.DeferBooleanFlags(r13, CondSignedLess)
			ctx.BindReg(r13, &d230)
		} else if d228.Loc == LocImm {
			r14 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d228.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d154.Reg)
			d230 = ctx.DeferBooleanFlags(r14, CondSignedLess)
			ctx.BindReg(r14, &d230)
		} else {
			r15 := ctx.AllocReg()
			ctx.EmitCmpInt64(d228.Reg, d154.Reg)
			d230 = ctx.DeferBooleanFlags(r15, CondSignedLess)
			ctx.BindReg(r15, &d230)
		}
		ctx.FreeDesc(&d228)
		ctx.SyncDesc(&d230)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d230) {
			return d230
		}
		ctx.EnsureDesc(&d230)
		ctx.EmitMovToReg(result.Reg, d230)
		result.Type = d230.Type
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
		ctx.ReclaimUntrackedRegs()
		d231 = ctx.EmitFloatDesc(args[0])
		d232 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d231)
		resultTarget233 := false
		_ = resultTarget233
		ctx.EnsureDesc(&d232)
		ctx.EnsureDescsTogether(&d231, &d232)
		if d231.Loc == LocImm && d232.Loc == LocImm {
			d234 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d231.Imm.Float() < d232.Imm.Float())}
		} else if d232.Loc == LocImm {
			var r16 Reg
			if result.Loc == LocReg && result.Reg != d231.Reg {
				r16 = result.Reg
				resultTarget233 = true
			} else {
				r16 = ctx.AllocRegExcept(d231.Reg)
			}
			_, yBits := d232.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d231.Reg)
			d234 = ctx.DeferBooleanFlags(r16, CondUnsignedAbove)
			ctx.BindReg(r16, &d234)
		} else if d231.Loc == LocImm {
			var r17 Reg
			if result.Loc == LocReg && result.Reg != d232.Reg {
				r17 = result.Reg
				resultTarget233 = true
			} else {
				r17 = ctx.AllocRegExcept(d232.Reg)
			}
			_, xBits := d231.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d232.Reg, ctx.ScratchReg)
			d234 = ctx.DeferBooleanFlags(r17, CondUnsignedAbove)
			ctx.BindReg(r17, &d234)
		} else {
			var r18 Reg
			if result.Loc == LocReg && result.Reg != d231.Reg && result.Reg != d232.Reg {
				r18 = result.Reg
				resultTarget233 = true
			} else {
				r18 = ctx.AllocRegExcept(d231.Reg, d232.Reg)
			}
			ctx.EmitCmpFloat64(d232.Reg, d231.Reg)
			d234 = ctx.DeferBooleanFlags(r18, CondUnsignedAbove)
			ctx.BindReg(r18, &d234)
		}
		ctx.FreeDesc(&d231)
		ctx.FreeDesc(&d232)
		ctx.SyncDesc(&d234)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d234) {
			return d234
		}
		ctx.EnsureDesc(&d234)
		ctx.EmitMovToReg(result.Reg, d234)
		result.Type = d234.Type
		ctx.EmitJmp(lbl0)
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d235 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x3))}
		} else {
			r19 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 3)
			d235 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondEqual}
			ctx.BindReg(r19, &d235)
		}
		d236 = d235
		ctx.EnsureDesc(&d236)
		if d236.Loc != LocImm && d236.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d236.Loc == LocImm {
			if d236.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[20].Render()
		}
		ctx.EmitJump(d236.Condition, lbl16)
		if bbs[20].Rendered {
			ctx.EmitJmp(lbl21)
		}
		ctx.FreeDesc(&d235)
		ctx.FlushRegisterMoves()
		if !bbs[20].Rendered {
			snap237 := d0
			snap238 := d1
			snap239 := d2
			snap240 := d3
			snap241 := d4
			snap242 := d5
			snap243 := d13
			snap244 := d14
			snap245 := d15
			snap246 := d26
			snap247 := d27
			snap248 := d40
			snap249 := d41
			snap250 := d42
			snap251 := d58
			snap252 := d59
			snap253 := d60
			snap254 := d79
			snap255 := d80
			snap256 := d101
			snap257 := d102
			snap258 := d125
			snap259 := d126
			snap260 := d151
			snap261 := d152
			snap262 := d154
			snap263 := d155
			snap264 := d156
			snap265 := d186
			snap266 := d187
			snap267 := d188
			snap268 := d190
			snap269 := d191
			snap270 := d192
			snap271 := d228
			snap272 := d230
			snap273 := d231
			snap274 := d232
			snap275 := d234
			snap276 := d235
			snap277 := d236
			alloc278 := ctx.SnapshotAllocState()
			bbs[20].Render()
			ctx.RestoreAllocState(alloc278)
			d0 = snap237
			d1 = snap238
			d2 = snap239
			d3 = snap240
			d4 = snap241
			d5 = snap242
			d13 = snap243
			d14 = snap244
			d15 = snap245
			d26 = snap246
			d27 = snap247
			d40 = snap248
			d41 = snap249
			d42 = snap250
			d58 = snap251
			d59 = snap252
			d60 = snap253
			d79 = snap254
			d80 = snap255
			d101 = snap256
			d102 = snap257
			d125 = snap258
			d126 = snap259
			d151 = snap260
			d152 = snap261
			d154 = snap262
			d155 = snap263
			d156 = snap264
			d186 = snap265
			d187 = snap266
			d188 = snap267
			d190 = snap268
			d191 = snap269
			d192 = snap270
			d228 = snap271
			d230 = snap272
			d231 = snap273
			d232 = snap274
			d234 = snap275
			d235 = snap276
			d236 = snap277
		}
		if !bbs[15].Rendered {
			return bbs[15].Render()
		}
		return result
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
		ctx.ReclaimUntrackedRegs()
		if args[0].Loc == LocImm {
			d279 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d279 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d279)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d279 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d279)
		} else {
			d279 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d279.Type = tagInt
			ctx.BindReg(d279.Reg, &d279)
		}
		if args[1].Loc == LocImm {
			d280 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d280 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d280)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d280 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d280)
		} else {
			d280 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[1]}, 1)
			d280.Type = tagInt
			ctx.BindReg(d280.Reg, &d280)
		}
		ctx.EnsureDesc(&d279)
		resultTarget281 := false
		_ = resultTarget281
		ctx.EnsureDesc(&d280)
		ctx.EnsureDescsTogether(&d279, &d280)
		if d279.Loc == LocImm && d280.Loc == LocImm {
			d282 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d279.Imm.Int() < d280.Imm.Int())}
		} else if d280.Loc == LocImm {
			r20 := ctx.AllocReg()
			if d280.Imm.Int() >= -2147483648 && d280.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d279.Reg, int32(d280.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d280.Imm.Int()))
				ctx.EmitCmpInt64(d279.Reg, ctx.ScratchReg)
			}
			d282 = ctx.DeferBooleanFlags(r20, CondSignedLess)
			ctx.BindReg(r20, &d282)
		} else if d279.Loc == LocImm {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d279.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d280.Reg)
			d282 = ctx.DeferBooleanFlags(r21, CondSignedLess)
			ctx.BindReg(r21, &d282)
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitCmpInt64(d279.Reg, d280.Reg)
			d282 = ctx.DeferBooleanFlags(r22, CondSignedLess)
			ctx.BindReg(r22, &d282)
		}
		ctx.FreeDesc(&d279)
		ctx.FreeDesc(&d280)
		ctx.SyncDesc(&d282)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d282) {
			return d282
		}
		ctx.EnsureDesc(&d282)
		ctx.EmitMovToReg(result.Reg, d282)
		result.Type = d282.Type
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
		ctx.ReclaimUntrackedRegs()
		if args[0].Loc == LocImm {
			d283 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d283 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d283)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d283 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d283)
		} else {
			d283 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d283.Type = tagInt
			ctx.BindReg(d283.Reg, &d283)
		}
		ctx.EnsureDesc(&d283)
		ctx.EnsureDesc(&d283)
		if d283.Loc == LocImm {
			d284 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d283.Imm.Int()))}
		} else {
			var r23 Reg
			r23 = d283.Reg
			d283.Loc = LocNone
			ctx.EmitInt64ToFloatBits(r23)
			d284 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r23}
			ctx.BindReg(r23, &d284)
		}
		ctx.FreeDesc(&d283)
		d285 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d284)
		resultTarget286 := false
		_ = resultTarget286
		ctx.EnsureDesc(&d285)
		ctx.EnsureDescsTogether(&d284, &d285)
		if d284.Loc == LocImm && d285.Loc == LocImm {
			d287 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d284.Imm.Float() < d285.Imm.Float())}
		} else if d285.Loc == LocImm {
			var r24 Reg
			if result.Loc == LocReg && result.Reg != d284.Reg {
				r24 = result.Reg
				resultTarget286 = true
			} else {
				r24 = ctx.AllocRegExcept(d284.Reg)
			}
			_, yBits := d285.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d284.Reg)
			d287 = ctx.DeferBooleanFlags(r24, CondUnsignedAbove)
			ctx.BindReg(r24, &d287)
		} else if d284.Loc == LocImm {
			var r25 Reg
			if result.Loc == LocReg && result.Reg != d285.Reg {
				r25 = result.Reg
				resultTarget286 = true
			} else {
				r25 = ctx.AllocRegExcept(d285.Reg)
			}
			_, xBits := d284.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d285.Reg, ctx.ScratchReg)
			d287 = ctx.DeferBooleanFlags(r25, CondUnsignedAbove)
			ctx.BindReg(r25, &d287)
		} else {
			var r26 Reg
			if result.Loc == LocReg && result.Reg != d284.Reg && result.Reg != d285.Reg {
				r26 = result.Reg
				resultTarget286 = true
			} else {
				r26 = ctx.AllocRegExcept(d284.Reg, d285.Reg)
			}
			ctx.EmitCmpFloat64(d285.Reg, d284.Reg)
			d287 = ctx.DeferBooleanFlags(r26, CondUnsignedAbove)
			ctx.BindReg(r26, &d287)
		}
		ctx.FreeDesc(&d284)
		ctx.FreeDesc(&d285)
		ctx.SyncDesc(&d287)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d287) {
			return d287
		}
		ctx.EnsureDesc(&d287)
		ctx.EmitMovToReg(result.Reg, d287)
		result.Type = d287.Type
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
		ctx.ReclaimUntrackedRegs()
		if args[0].Loc == LocImm {
			d288 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d288 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d288)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d288 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d288)
		} else {
			d288 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d288.Type = tagInt
			ctx.BindReg(d288.Reg, &d288)
		}
		if args[1].Loc == LocImm {
			d289 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d289 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d289)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d289 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d289)
		} else {
			d289 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[1]}, 1)
			d289.Type = tagInt
			ctx.BindReg(d289.Reg, &d289)
		}
		ctx.EnsureDesc(&d288)
		resultTarget290 := false
		_ = resultTarget290
		ctx.EnsureDesc(&d289)
		ctx.EnsureDescsTogether(&d288, &d289)
		if d288.Loc == LocImm && d289.Loc == LocImm {
			d291 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d288.Imm.Int() < d289.Imm.Int())}
		} else if d289.Loc == LocImm {
			r27 := ctx.AllocReg()
			if d289.Imm.Int() >= -2147483648 && d289.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d288.Reg, int32(d289.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d289.Imm.Int()))
				ctx.EmitCmpInt64(d288.Reg, ctx.ScratchReg)
			}
			d291 = ctx.DeferBooleanFlags(r27, CondSignedLess)
			ctx.BindReg(r27, &d291)
		} else if d288.Loc == LocImm {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d288.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d289.Reg)
			d291 = ctx.DeferBooleanFlags(r28, CondSignedLess)
			ctx.BindReg(r28, &d291)
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitCmpInt64(d288.Reg, d289.Reg)
			d291 = ctx.DeferBooleanFlags(r29, CondSignedLess)
			ctx.BindReg(r29, &d291)
		}
		ctx.FreeDesc(&d288)
		ctx.FreeDesc(&d289)
		ctx.SyncDesc(&d291)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d291) {
			return d291
		}
		ctx.EnsureDesc(&d291)
		ctx.EmitMovToReg(result.Reg, d291)
		result.Type = d291.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[20].Render = func() JITValueDesc {
		if bbs[20].Rendered {
			ctx.EmitJmp(lbl21)
			return result
		}
		bbs[20].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_20 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl21)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d292 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x5))}
		} else {
			r30 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 5)
			d292 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondEqual}
			ctx.BindReg(r30, &d292)
		}
		d293 = d292
		ctx.EnsureDesc(&d293)
		if d293.Loc != LocImm && d293.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d293.Loc == LocImm {
			if d293.Imm.Bool() {
				return bbs[19].Render()
			}
			return bbs[21].Render()
		}
		ctx.EmitJump(d293.Condition, lbl20)
		if bbs[21].Rendered {
			ctx.EmitJmp(lbl22)
		}
		ctx.FreeDesc(&d292)
		ctx.FlushRegisterMoves()
		if !bbs[21].Rendered {
			snap294 := d0
			snap295 := d1
			snap296 := d2
			snap297 := d3
			snap298 := d4
			snap299 := d5
			snap300 := d13
			snap301 := d14
			snap302 := d15
			snap303 := d26
			snap304 := d27
			snap305 := d40
			snap306 := d41
			snap307 := d42
			snap308 := d58
			snap309 := d59
			snap310 := d60
			snap311 := d79
			snap312 := d80
			snap313 := d101
			snap314 := d102
			snap315 := d125
			snap316 := d126
			snap317 := d151
			snap318 := d152
			snap319 := d154
			snap320 := d155
			snap321 := d156
			snap322 := d186
			snap323 := d187
			snap324 := d188
			snap325 := d190
			snap326 := d191
			snap327 := d192
			snap328 := d228
			snap329 := d230
			snap330 := d231
			snap331 := d232
			snap332 := d234
			snap333 := d235
			snap334 := d236
			snap335 := d279
			snap336 := d280
			snap337 := d282
			snap338 := d283
			snap339 := d284
			snap340 := d285
			snap341 := d287
			snap342 := d288
			snap343 := d289
			snap344 := d291
			snap345 := d292
			snap346 := d293
			alloc347 := ctx.SnapshotAllocState()
			bbs[21].Render()
			ctx.RestoreAllocState(alloc347)
			d0 = snap294
			d1 = snap295
			d2 = snap296
			d3 = snap297
			d4 = snap298
			d5 = snap299
			d13 = snap300
			d14 = snap301
			d15 = snap302
			d26 = snap303
			d27 = snap304
			d40 = snap305
			d41 = snap306
			d42 = snap307
			d58 = snap308
			d59 = snap309
			d60 = snap310
			d79 = snap311
			d80 = snap312
			d101 = snap313
			d102 = snap314
			d125 = snap315
			d126 = snap316
			d151 = snap317
			d152 = snap318
			d154 = snap319
			d155 = snap320
			d156 = snap321
			d186 = snap322
			d187 = snap323
			d188 = snap324
			d190 = snap325
			d191 = snap326
			d192 = snap327
			d228 = snap328
			d230 = snap329
			d231 = snap330
			d232 = snap331
			d234 = snap332
			d235 = snap333
			d236 = snap334
			d279 = snap335
			d280 = snap336
			d282 = snap337
			d283 = snap338
			d284 = snap339
			d285 = snap340
			d287 = snap341
			d288 = snap342
			d289 = snap343
			d291 = snap344
			d292 = snap345
			d293 = snap346
		}
		if !bbs[19].Rendered {
			return bbs[19].Render()
		}
		return result
		return result
	}
	bbs[21].Render = func() JITValueDesc {
		if bbs[21].Rendered {
			ctx.EmitJmp(lbl22)
			return result
		}
		bbs[21].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_21 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl22)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		args[0] = JITPrepareScmerGoArg(ctx, args[0])
		args[1] = JITPrepareScmerGoArg(ctx, args[1])
		if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocRegTriple || d3.Loc == LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&args[0])
		ctx.SyncDesc(&args[1])
		ctx.SyncDesc(&d1)
		ctx.SyncDesc(&d3)
		d348 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{args[0], args[1], d1, d3}, 1)
		d348.NoHeapPointer = true
		ctx.EmitAndRegImm32(d348.Reg, 1)
		d348.Type = tagBool
		ctx.BindReg(d348.Reg, &d348)
		ctx.FreeDesc(&args[0])
		ctx.FreeDesc(&args[1])
		ctx.SyncDesc(&d348)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d348) {
			return d348
		}
		ctx.EnsureDesc(&d348)
		ctx.EmitMovToReg(result.Reg, d348)
		result.Type = d348.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	returned := bbs[0].Render()
	if ctx.hasBooleanFlags(returned) {
		if resultRegsProtected {
			ctx.UnprotectReg(result.Reg)
		}
		return returned
	}
	ctx.MarkLabel(lbl0)
	ctx.ResolveFixups()
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg)
	}
	return result

}
